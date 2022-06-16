package main

import (
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

type EventType int64

const (
	Approved EventType = iota
	Rejected
)

type ComplexField struct {
	Value string `json:"value"`
}

type Field struct {
	Summary     string       `json:"summary"`
	Description string       `json:"description"`
	DocLink     string       `json:"customfield_11264"`
	Product     string       `json:"customfield_10576"`
	Title       string       `json:"customfield_10011"`
	BU          ComplexField `json:"customfield_10952"`
	Company     ComplexField `json:"customfield_10412"`
	SpecType    ComplexField `json:"customfield_11368"`
	Created     string       `json:"created"`
}

type Issue struct {
	Id     string `json:"id"`
	Key    string `json:"key"`
	Self   string `json:"self"`
	Fields Field  `json:"fields"`
}

type IssuesResponse struct {
	Total    int     `json:"total"`
	StartAt  int     `json:"startAt"`
	PageSize int     `json:"maxResults"`
	Issues   []Issue `json:"issues"`
}

type Paragraph struct {
	Type string `json:"type"`
	Text string `json:"text`
}

type Content struct {
	Paragaph []Paragraph `json:"content"`
}

type ContentBody struct {
	Content []Content `json:"content"`
}

type Comment struct {
	CommentBody ContentBody `json:"body"`
	Created     string      `json:"created"`
}

type CommentsResponse struct {
	Total    int       `json:"total"`
	StartAt  int       `json:"startAt"`
	Comments []Comment `json:"comments"`
}

// queryIssues - get a page of JIRA data
func queryIssues(startAt int) *IssuesResponse {
	user := os.Getenv("JIRA_user")

	apiToken := os.Getenv("JIRA_token")

	accessString := user + ":" + apiToken

	accessString = b64.StdEncoding.EncodeToString([]byte(accessString))

	jql := url.QueryEscape("project=CENPRO and type=EPIC and labels not in (TestData) and updated>=startOfYear()")

	client := &http.Client{}
	query := fmt.Sprintf("https://workstation-df.atlassian.net/rest/api/2/search?jql=%s&startAt=%d&maxResults=1000", jql, startAt)
	request, err := http.NewRequest("GET", query, nil)

	request.Header.Add("Authorization", "Basic "+accessString)
	resp, err := client.Do(request)
	if err != nil {
		log.Fatal(err)
	}
	responseData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var jiraResponse IssuesResponse

	json.Unmarshal(responseData, &jiraResponse)

	return &jiraResponse
}

// queryComments - get a page of JIRA data
func queryComments(issueId string) *CommentsResponse {
	user := os.Getenv("JIRA_user")

	apiToken := os.Getenv("JIRA_token")

	accessString := user + ":" + apiToken

	accessString = b64.StdEncoding.EncodeToString([]byte(accessString))

	client := &http.Client{}
	query := fmt.Sprintf("https://workstation-df.atlassian.net/rest/api/3/issue/%s/comment?orderBy=+created&maxResults=100", issueId)
	request, err := http.NewRequest("GET", query, nil)

	request.Header.Add("Authorization", "Basic "+accessString)
	resp, err := client.Do(request)
	if err != nil {
		log.Fatal(err)
	}
	responseData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var jiraResponse CommentsResponse

	json.Unmarshal(responseData, &jiraResponse)

	return &jiraResponse
}

func main() {
	//	weekRegex, _ := regexp.Compile("W(\\d)+")

	// Query JIRA

	cursor := 0

	fmt.Println("Type,Id,Summary,Company,Product,First Approved (year),Week,Subsequent Approvals,Latency,Rejections (before first approval),Approved 1st time,Quarter")

	for atEnd := false; !atEnd; {
		response := queryIssues(cursor)

		// Go look for approved rejected status

		for _, issue := range response.Issues {
			comments := queryComments(issue.Id)

			approved := false

			var firstApproved time.Time

			approvalWeek := ""

			rejectionsPriorToFirstApproval := 0

			subsequentApprovals := 0

			for _, comment := range comments.Comments {
				for _, paragraph := range comment.CommentBody.Content {

					if len(paragraph.Paragaph) >= 2 {
						if paragraph.Paragaph[0].Text == "Approved in " {
							createDate, _ := time.Parse("2006-01-02T15:04:05+0000", comment.Created)

							if !approved {
								approved = true
								firstApproved = createDate
								approvalWeek = paragraph.Paragaph[1].Text[1:]
							} else {
								subsequentApprovals += 1
							}
						}
						if paragraph.Paragaph[0].Text == "Rejected in " {
							if !approved {
								rejectionsPriorToFirstApproval += 1
							}
						}
					}
				}
			}

			if approved {
				created, _ := time.Parse("2006-01-02T15:04:05+0000", issue.Fields.Created)
				latency := firstApproved.Sub(created)
				approvedFirstTime := "N"
				if rejectionsPriorToFirstApproval == 0 {
					approvedFirstTime = "Y"
				}
				weekNum, _ := strconv.ParseFloat(approvalWeek, 64)
				quarter := int(weekNum/13.04) + 1
				fmt.Printf("%s,%s,\"%s\",%s,%s,%d,%s,%d,%d,%d,%s,%d\n", issue.Fields.SpecType.Value, issue.Key, issue.Fields.Summary, issue.Fields.Company.Value, issue.Fields.BU.Value, firstApproved.Year(), approvalWeek, subsequentApprovals, int64(latency.Hours()/24), rejectionsPriorToFirstApproval, approvedFirstTime, quarter)

			}
		}

		// House-keeping

		cursor = response.StartAt + response.PageSize

		if response.StartAt+len(response.Issues) >= response.Total {
			atEnd = true
		}
	}
}
