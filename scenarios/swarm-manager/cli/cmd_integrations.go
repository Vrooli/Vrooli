package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

type integrationStatusResponse struct {
	Integrations []integrationStatus `json:"integrations"`
}

type integrationStatus struct {
	ID                  string   `json:"id"`
	Required            bool     `json:"required"`
	Configured          bool     `json:"configured"`
	Availability        string   `json:"availability"`
	CheckedAt           string   `json:"checkedAt"`
	FreshUntil          string   `json:"freshUntil"`
	DegradedBehavior    string   `json:"degradedBehavior"`
	Diagnostic          string   `json:"diagnostic"`
	AffectedTransitions []string `json:"affectedTransitions"`
}

// cmdIntegrationStatus intentionally reads only Swarm's projection. It must
// never add CLI-local health probes that could disagree with workflow preflight.
func (a *App) cmdIntegrationStatus(args []string) error {
	fs := flag.NewFlagSet("integrations", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	body, err := a.core.Get("/integrations", nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}
	response, err := decodeResponse[integrationStatusResponse](body)
	if err != nil {
		return err
	}
	printSection("Integrations")
	for _, integration := range response.Integrations {
		state := integration.Availability
		if !integration.Configured {
			state = "unconfigured"
		}
		fmt.Printf("  %-20s %-14s required=%t\n", integration.ID, state, integration.Required)
		if diagnostic := strings.TrimSpace(integration.Diagnostic); diagnostic != "" {
			fmt.Printf("    Diagnostic: %s\n", diagnostic)
		}
		if behavior := strings.TrimSpace(integration.DegradedBehavior); behavior != "" {
			fmt.Printf("    Degraded behavior: %s\n", behavior)
		}
		if len(integration.AffectedTransitions) > 0 {
			fmt.Printf("    Affects: %s\n", strings.Join(integration.AffectedTransitions, ", "))
		}
	}
	printCommandListSection("Next Steps", []string{cliCommand("integrations", "--json")})
	return nil
}
