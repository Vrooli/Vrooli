package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

type planWorkshopCLIResponse struct {
	ID      string `json:"id"`
	Subject struct {
		Kind string `json:"kind"`
		Ref  string `json:"ref"`
	} `json:"subject"`
	Review *struct {
		State string `json:"state"`
	} `json:"review,omitempty"`
}

type planWorkshopReviewCLIResponse struct {
	Session planWorkshopCLIResponse `json:"session"`
	Review  *struct {
		State string `json:"state"`
	} `json:"review,omitempty"`
}

func (a *App) cmdBacklogPlanWorkshop(args []string) error {
	fs := flag.NewFlagSet("backlog plan-workshop", flag.ContinueOnError)
	kindFlag := fs.String("kind", "", "Backlog item kind")
	nameFlag := fs.String("name", "", "Backlog item name")
	startReview := fs.Bool("start-review", false, "Start the bounded Plan Workshop review after opening the session")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlags("kind", *kindFlag, "name", *nameFlag); err != nil {
		return fmt.Errorf("usage: backlog plan-workshop --kind KIND --name NAME [--start-review] [--json]\n\n%s", err)
	}
	payload, err := json.Marshal(map[string]any{"subject": map[string]string{"kind": "backlog_item", "ref": strings.TrimSpace(*kindFlag) + "/" + strings.TrimSpace(*nameFlag)}})
	if err != nil {
		return err
	}
	body, err := a.core.Request("POST", "/plan-workshops", nil, json.RawMessage(payload))
	if err != nil {
		return err
	}
	response, err := decodeResponse[planWorkshopCLIResponse](body)
	if err != nil {
		return err
	}
	if *startReview {
		body, err = a.core.Request("POST", "/plan-workshops/"+response.ID+"/review", nil, nil)
		if err != nil {
			return err
		}
		if !*jsonOut {
			reviewResult, err := decodeResponse[planWorkshopReviewCLIResponse](body)
			if err != nil {
				return err
			}
			response = reviewResult.Session
			response.Review = reviewResult.Review
		}
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}
	printSection("Plan Workshop")
	fmt.Printf("  Session: %s\n", response.ID)
	fmt.Printf("  Subject: %s\n", response.Subject.Ref)
	if response.Review != nil {
		fmt.Printf("  Review: %s\n", response.Review.State)
	}
	printCommandListSection("Next Steps", []string{cliCommand("backlog", "get", "--kind", *kindFlag, "--name", *nameFlag)})
	return nil
}
