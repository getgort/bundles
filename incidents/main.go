package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type Input struct {
	Channel struct {
		ID   string
		Name string
	}
}

type Output struct {
	Action    string
	ChannelID string
	Title     string
	Content   string
}

type PagerDutyIncident struct {
	Id                 string
	IncidentNumber     int64 `json:"incident_number"`
	ExternalReferences []struct {
		Summary     string
		ExternalID  string `json:"external_id"`
		ExternalURL string `json:"external_url"`
	} `json:"external_references"`
	HtmlUrl string `json:"html_url"`
}

type PagerdutyIncidentsList struct {
	Incidents []PagerDutyIncident
	More      bool
	Limit     int
}

const pagerdutyUrl = "https://api.pagerduty.com/incidents?include[]=external_references&statuses[]=triggered&statuses[]=acknowledged&limit=25"

func do() error {
	token, present := os.LookupEnv("PAGERDUTY_API_TOKEN")
	if !present {
		return fmt.Errorf("PagerDuty API token not found. Please specify it as PAGERDUTY_API_TOKEN with dynamic config")
	}

	var input Input
	err := json.NewDecoder(strings.NewReader(os.Args[1])).Decode(&input)
	if err != nil {
		return err
	}

	name := strings.Split(input.Channel.Name, "_")
	num := name[len(name)-1]

	incidentNumber, err := strconv.ParseInt(num, 10, 0)

	var client http.Client

	more := true
	offset := 0
	var incidentUrl string
	var issueId string
	var issueUrl string

pagination:
	for more {
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s&offset=%d", pagerdutyUrl, offset), nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", fmt.Sprintf("Token token=%s", token))
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		var incidents PagerdutyIncidentsList
		err = json.NewDecoder(resp.Body).Decode(&incidents)
		if err != nil {
			return err
		}

		more = incidents.More
		offset += incidents.Limit

		for _, i := range incidents.Incidents {
			if i.IncidentNumber != incidentNumber {
				continue
			}

			incidentUrl = i.HtmlUrl

			for _, r := range i.ExternalReferences {
				if r.Summary == "JIRA" {
					issueId = r.ExternalID
					issueUrl = r.ExternalURL
				}
			}

			break pagination
		}
	}

	if incidentUrl == "" {
		return fmt.Errorf("Failed to find incident.")
	}

	bookmarkIncident := Output{
		Action:    "bookmark",
		ChannelID: input.Channel.ID,
		Title:     "PagerDuty",
		Content:   incidentUrl,
	}
	bmi, err := json.Marshal(&bookmarkIncident)
	fmt.Printf("#!#%s\n", string(bmi))

	if issueUrl == "" {
		fmt.Println("Failed to find JIRA issue for incident")
	} else {
		bookmarkJira := Output{
			Action:    "bookmark",
			ChannelID: input.Channel.ID,
			Title:     fmt.Sprintf("JIRA: %s", issueId),
			Content:   issueUrl,
		}
		bj, err := json.Marshal(bookmarkJira)
		if err != nil {
			return err
		}
		fmt.Printf("#!#%s\n", string(bj))
	}

	return nil
}

func main() {
	if err := do(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
