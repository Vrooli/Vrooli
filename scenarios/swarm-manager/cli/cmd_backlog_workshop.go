package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) cmdBacklogWorkshopReset(args []string) error {
	fs := flag.NewFlagSet("backlog workshop-reset", flag.ContinueOnError)
	kindFlag := fs.String("kind", "", "Backlog item kind")
	nameFlag := fs.String("name", "", "Backlog item name")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlags("kind", *kindFlag, "name", *nameFlag); err != nil {
		return fmt.Errorf("usage: backlog workshop-reset --kind KIND --name NAME [--json]\n\n%s", err)
	}
	kind := strings.TrimSpace(*kindFlag)
	name := strings.TrimSpace(*nameFlag)

	body, err := a.core.Request("POST", "/backlog/"+kind+"/"+name+"/workshop/reset", nil, nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	type workshopResetResponse struct {
		DeletedRounds  int  `json:"deleted_rounds"`
		StatusReverted bool `json:"status_reverted"`
	}
	resp, err := decodeResponse[workshopResetResponse](body)
	if err != nil {
		return err
	}

	printSection("Result")
	fmt.Printf("  Reset workshop for %s/%s\n", kind, name)
	fmt.Printf("  Deleted rounds: %d\n", resp.DeletedRounds)
	if resp.StatusReverted {
		fmt.Println("  Status reverted from \"ready\" to \"backlog\"")
	}
	printCommandListSection("Next Steps", []string{
		cliCommand("backlog", "get", "--kind", kind, "--name", name),
	})
	return nil
}

// cmdBacklogReWorkshop drives the "plan is stale, redo the workshop" flow:
// clears prior workshop rounds and the deliverable, reverts status to
// backlog, and queues a fresh workshop round.
func (a *App) cmdBacklogReWorkshop(args []string) error {
	fs := flag.NewFlagSet("backlog re-workshop", flag.ContinueOnError)
	kindFlag := fs.String("kind", "", "Backlog item kind")
	nameFlag := fs.String("name", "", "Backlog item name")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlags("kind", *kindFlag, "name", *nameFlag); err != nil {
		return fmt.Errorf("usage: backlog re-workshop --kind KIND --name NAME [--json]\n\n%s", err)
	}
	kind := strings.TrimSpace(*kindFlag)
	name := strings.TrimSpace(*nameFlag)

	body, err := a.core.Request("POST", "/backlog/"+kind+"/"+name+"/re-workshop", nil, nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	type reWorkshopResponse struct {
		DeletedRounds  int  `json:"deleted_rounds"`
		StatusReverted bool `json:"status_reverted"`
	}
	resp, err := decodeResponse[reWorkshopResponse](body)
	if err != nil {
		return err
	}

	printSection("Result")
	fmt.Printf("  Re-workshop triggered for %s/%s\n", kind, name)
	fmt.Printf("  Deleted rounds: %d\n", resp.DeletedRounds)
	if resp.StatusReverted {
		fmt.Println("  Status reverted to \"backlog\"")
	}
	fmt.Println("  A fresh workshop round has been queued.")
	printCommandListSection("Next Steps", []string{
		cliCommand("backlog", "get", "--kind", kind, "--name", name),
	})
	return nil
}

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
