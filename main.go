package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type EventType int64

const (
	Approved EventType = iota
	Rejected
)

type specStruct struct {
	eventType      EventType
	specType       string
	issue          string
	status         string
	BU             string
	product        string
	eventDate      time.Time
	created        time.Time
	week           int
	laterApprovals []string
	rejections     int
}

func main() {
	f, err := os.Open("cenpro.csv")
	if err != nil {
		log.Fatal(err)
	}

	// remember to close the file at the end of the program
	defer f.Close()

	// read csv values using csv.Reader
	csvReader := csv.NewReader(f)
	data, err := csvReader.ReadAll()
	if err != nil {
		log.Fatal(err)
	}

	weekRegex, _ := regexp.Compile("W(\\d)+")

	specs := make(map[string]([]specStruct))

	for _, row := range data {
		for _, cell := range row {
			if strings.Contains(cell, "Approved in") || strings.Contains(cell, "Rejected in") {
				var eventType EventType
				if strings.Contains(cell, "Approved in") {
					eventType = Approved
				} else {
					eventType = Rejected
				}
				lines := strings.Split(cell, "\n")
				parts1 := strings.Split(lines[0], " ")
				eventDate, err := time.Parse("02/Jan/06", parts1[0])
				if err != nil {
					fmt.Printf("Date %s not parsed\n", parts1[0])
				}

				if eventDate.Year() == 2022 {
					issue := row[1]
					status := row[4]
					BU := row[604]
					product := row[629]
					specType := row[1096]
					created := strings.Split(row[20], " ")[0]
					createDate, err := time.Parse("02/Jan/06", created)
					if err != nil {
						fmt.Printf("Date %s not parsed\n", created)
					}

					week := weekRegex.FindString(lines[0])

					weekNum, _ := strconv.Atoi(week[1:len(week)])

					spec := specStruct{eventType: eventType, specType: specType, issue: issue, status: status, BU: BU, product: product, eventDate: eventDate, created: createDate, week: weekNum}

					priorFoundSpecs, found := specs[issue]

					if found {
						priorFoundSpecs = append(priorFoundSpecs, spec)
						specs[issue] = priorFoundSpecs
					} else {
						specArray := (make([]specStruct, 0))
						specArray = append(specArray, spec)
						specs[issue] = specArray
					}
				}
			}
		}
	}

	// Now compile the specs together to find the first approval time, rejected rates, etc
	fmt.Println("Type,Issue,Status,BU,Product,Year,Week,Later Approved Count,Later Approved Weeks, Latency(Days),Rejected prior to 1st approval")

	for _, specArray := range specs {

		// Sort the array by time

		sort.Slice(specArray, func(i, j int) bool {
			return specArray[i].eventDate.Before(specArray[j].eventDate)
		})

		// Now go calculate the important stuff

		foundFirstPass := false

		rejectedCount := 0

		var firstSpec specStruct

		for _, value := range specArray {

			switch value.eventType {
			case Approved:
				if foundFirstPass == false {
					foundFirstPass = true
					value.rejections = rejectedCount
					firstSpec = value
				} else {
					firstSpec.laterApprovals = append(firstSpec.laterApprovals, "W"+strconv.Itoa(value.week))
				}
				rejectedCount = 0
				break
			case Rejected:
				rejectedCount += 1
			}

		}

		latency := firstSpec.eventDate.Sub(firstSpec.created)

		fmt.Printf("%s,%s,%s,%s,%s,%d,%d,%d,%s,%d,%d\n", firstSpec.specType, firstSpec.issue, firstSpec.status, firstSpec.BU, firstSpec.product, firstSpec.eventDate.Year(), firstSpec.week, len(firstSpec.laterApprovals), strings.Join(firstSpec.laterApprovals, ":"), int64(latency.Hours()/24), firstSpec.rejections)

	}

}
