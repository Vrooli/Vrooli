package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) cmdBacklogProcessPreflight(args []string) error {
	fs := flag.NewFlagSet("backlog process-preflight", flag.ContinueOnError)
	kindFlag := fs.String("kind", "", "Backlog item kind")
	nameFlag := fs.String("name", "", "Backlog item name")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlags("kind", *kindFlag, "name", *nameFlag); err != nil {
		return fmt.Errorf("usage: backlog process-preflight --kind KIND --name NAME [--json]\n\n%s", err)
	}

	kind := strings.TrimSpace(*kindFlag)
	name := strings.TrimSpace(*nameFlag)
	body, err := a.core.Get("/backlog/"+kind+"/"+name+"/process-preflight", nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[ProcessPreflightResponse](body)
	if err != nil {
		return err
	}

	printSection("Summary")
	fmt.Printf("  Readiness: %t\n", response.Preflight.Ready)
	fmt.Printf("  Item: %s/%s\n", response.Item.Kind, response.Item.Name)
	if response.Preflight.ResolvedTargetScenarioID != "" {
		fmt.Printf("  Target Scenario: %s\n", response.Preflight.ResolvedTargetScenarioID)
	}
	fmt.Printf("  Target Exists: %t\n", response.Preflight.TargetScenarioExists)
	if response.Preflight.SuggestedOperation != "" {
		fmt.Printf("  Suggested Operation: %s\n", response.Preflight.SuggestedOperation)
	}
	if response.Preflight.SuggestedSteerProfileID != "" {
		fmt.Printf("  Suggested Steer Profile: %s\n", response.Preflight.SuggestedSteerProfileID)
	}

	if len(response.Preflight.BlockingReasons) > 0 {
		printSection("Blocking Reasons")
		for _, reason := range response.Preflight.BlockingReasons {
			fmt.Printf("  - %s\n", reason)
		}
	}
	if len(response.Preflight.BlockingQuestions) > 0 {
		printSection("Blocking Questions")
		for _, q := range response.Preflight.BlockingQuestions {
			fmt.Printf("  - %s: %s\n", q.ID, q.Question)
		}
	}

	printCommandListSection("Next Steps", []string{
		cliCommand("backlog", "queue", "--kind", kind, "--name", name, "--execute", "--operation", response.Preflight.SuggestedOperation),
		cliCommand("backlog", "get", "--kind", kind, "--name", name),
	})
	return nil
}
