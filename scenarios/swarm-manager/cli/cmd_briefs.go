package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

type cliBriefResponse struct {
	Brief cliAgentSessionContextItem `json:"brief"`
}

func (a *App) cmdPortfolioBrief(args []string) error {
	fs := flag.NewFlagSet("portfolio brief", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	body, err := a.core.Get("/portfolio/brief", nil)
	if err != nil {
		return err
	}
	return printBriefResponse(body, *jsonOut)
}

func (a *App) cmdInitiativesCandidates(args []string) error {
	fs := flag.NewFlagSet("initiatives candidates", flag.ContinueOnError)
	purposeFlag := fs.String("purpose", "next-action", "Candidate purpose")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	query := url.Values{}
	if purpose := strings.TrimSpace(*purposeFlag); purpose != "" {
		query.Set("purpose", purpose)
	}
	body, err := a.core.Get("/initiative-candidates", query)
	if err != nil {
		return err
	}
	return printBriefResponse(body, *jsonOut)
}

func (a *App) cmdOperatingModeBrief(args []string) error {
	fs := flag.NewFlagSet("operating-mode brief", flag.ContinueOnError)
	modeFlag := fs.String("mode", "", "Mode to use as comparison target")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	query := url.Values{}
	if mode := strings.TrimSpace(*modeFlag); mode != "" {
		query.Set("mode", mode)
	}
	body, err := a.core.Get("/operating-mode/brief", query)
	if err != nil {
		return err
	}
	return printBriefResponse(body, *jsonOut)
}

func printBriefResponse(body []byte, jsonOut bool) error {
	if printJSONIfRequested(jsonOut, body) {
		return nil
	}
	var response cliBriefResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse brief response: %w", err)
	}
	brief := response.Brief
	printSection("Brief")
	fmt.Printf("  %s\n", brief.Title)
	if brief.Ref != "" {
		fmt.Printf("  Ref: %s\n", brief.Ref)
	}
	if brief.SelectedAt != "" {
		fmt.Printf("  Generated: %s\n", brief.SelectedAt)
	}
	printSection("Summary")
	for _, line := range strings.Split(strings.TrimSpace(brief.Summary), "\n") {
		if strings.TrimSpace(line) != "" {
			fmt.Printf("  %s\n", line)
		}
	}
	return nil
}
