package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) cmdFindings(args []string) error {
	if len(args) == 0 {
		return a.findingsList(nil)
	}
	if args[0] == "list" {
		return a.findingsList(args[1:])
	}
	return fmt.Errorf("usage: agent-manager findings list [--since RFC3339] [--fingerprint value] [--severity value] [--json]")
}

func (a *App) findingsList(args []string) error {
	fs := flag.NewFlagSet("findings list", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	since, fingerprint, severity := fs.String("since", "", "Only findings created since RFC3339 time"), fs.String("fingerprint", "", "Exact recurrence fingerprint"), fs.String("severity", "", "Severity filter")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	query := url.Values{}
	for key, value := range map[string]string{"since": *since, "fingerprint": *fingerprint, "severity": *severity} {
		if strings.TrimSpace(value) != "" {
			query.Set(key, strings.TrimSpace(value))
		}
	}
	body, err := a.services.Findings.List(query)
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}
	var result struct {
		Findings []struct {
			Fingerprint    string `json:"fingerprint"`
			Occurrences    int    `json:"occurrences"`
			Severity       string `json:"severity"`
			Recommendation string `json:"recommendation"`
		} `json:"findings"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("decode findings: %w", err)
	}
	for _, finding := range result.Findings {
		fmt.Printf("%s occurrences=%d severity=%s\n  %s\n", finding.Fingerprint, finding.Occurrences, finding.Severity, finding.Recommendation)
	}
	fmt.Println("Next: agent-manager run report <run-id>; agent-manager findings list --fingerprint <fingerprint>")
	return nil
}
