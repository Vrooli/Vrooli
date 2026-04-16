package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

func (a *App) cmdReviewList(args []string) error {
	fs := flag.NewFlagSet("review-list", flag.ContinueOnError)
	kind := fs.String("kind", "", "Backlog kind (required)")
	name := fs.String("name", "", "Backlog name (required)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *kind == "" || *name == "" {
		return fmt.Errorf("usage: review-list --kind KIND --name NAME [--json]")
	}

	body, err := a.core.Get(fmt.Sprintf("/backlog/%s/%s/review", *kind, *name), nil)
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var resp struct {
		Rounds []struct {
			Round           int    `json:"round"`
			Status          string `json:"status"`
			Classification  string `json:"classification"`
			AgentAssessment string `json:"agent_assessment"`
			Evidence        []struct {
				ID       string `json:"id"`
				Type     string `json:"type"`
				Title    string `json:"title"`
				Verified bool   `json:"verified"`
			} `json:"evidence"`
		} `json:"rounds"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	if len(resp.Rounds) == 0 {
		fmt.Printf("No review evidence rounds for %s/%s\n", *kind, *name)
		return nil
	}

	fmt.Printf("Evidence Review: %s/%s\n", *kind, *name)
	for _, round := range resp.Rounds {
		verifiedCount := 0
		for _, e := range round.Evidence {
			if e.Verified {
				verifiedCount++
			}
		}
		fmt.Printf("\nRound %d (%s, %d evidence items, %d verified)\n",
			round.Round, round.Status, len(round.Evidence), verifiedCount)

		for _, e := range round.Evidence {
			check := "[ ]"
			if e.Verified {
				check = "[~]"
			}
			typeLabel := strings.ToUpper(e.Type[:1]) + e.Type[1:]
			fmt.Printf("  %s %s: %s\n", check, typeLabel, e.Title)
		}

		if round.AgentAssessment != "" {
			fmt.Printf("\nAgent assessment: %s\n", round.AgentAssessment)
		}
		if round.Classification != "" {
			fmt.Printf("Classification: %s\n", round.Classification)
		}
	}
	return nil
}

func (a *App) cmdReviewVerify(args []string) error {
	fs := flag.NewFlagSet("review-verify", flag.ContinueOnError)
	kind := fs.String("kind", "", "Backlog kind (required)")
	name := fs.String("name", "", "Backlog name (required)")
	round := fs.Int("round", 0, "Round number (required)")
	evidenceID := fs.String("evidence-id", "", "Evidence item ID (required)")
	unverify := fs.Bool("unverify", false, "Remove verification (default: verify)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *kind == "" || *name == "" || *round == 0 || *evidenceID == "" {
		return fmt.Errorf("usage: review-verify --kind KIND --name NAME --round N --evidence-id ID [--unverify]")
	}

	verified := !*unverify
	payload := map[string]any{"verified": verified}
	path := fmt.Sprintf("/backlog/%s/%s/review/%d/verify/%s", *kind, *name, *round, *evidenceID)
	body, err := a.core.Request("POST", path, nil, payload)
	if err != nil {
		return err
	}
	_ = body
	if verified {
		fmt.Printf("Verified evidence %s in round %d\n", *evidenceID, *round)
	} else {
		fmt.Printf("Unverified evidence %s in round %d\n", *evidenceID, *round)
	}
	return nil
}

func (a *App) cmdReviewRequest(args []string) error {
	fs := flag.NewFlagSet("review-request", flag.ContinueOnError)
	kind := fs.String("kind", "", "Backlog kind (required)")
	name := fs.String("name", "", "Backlog name (required)")
	round := fs.Int("round", 0, "Round number (required)")
	message := fs.String("message", "", "Evidence request message (required)")
	evidenceID := fs.String("evidence-id", "", "Specific evidence item ID (optional)")
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *kind == "" || *name == "" || *round == 0 || *message == "" {
		return fmt.Errorf("usage: review-request --kind KIND --name NAME --round N --message MSG [--evidence-id ID] [--json]")
	}

	payload := map[string]any{
		"message": *message,
	}
	if *evidenceID != "" {
		payload["evidence_id"] = *evidenceID
	}

	path := fmt.Sprintf("/backlog/%s/%s/review/%d/request", *kind, *name, *round)
	body, err := a.core.Request("POST", path, nil, payload)
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var resp struct {
		ThreadID string `json:"thread_id"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}
	fmt.Printf("Evidence request created: thread %s\n", resp.ThreadID)
	return nil
}

func (a *App) cmdReviewTrigger(args []string) error {
	fs := flag.NewFlagSet("review-trigger", flag.ContinueOnError)
	execID := fs.String("id", "", "Execution ID (required)")
	kind := fs.String("kind", "", "Backlog kind (required)")
	name := fs.String("name", "", "Backlog name (required)")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *execID == "" || *kind == "" || *name == "" {
		return fmt.Errorf("usage: review-trigger --id EXECUTION_ID --kind KIND --name NAME")
	}

	payload := map[string]any{
		"backlog_kind": *kind,
		"backlog_name": *name,
	}
	path := fmt.Sprintf("/execution/%s/trigger-review-agent", *execID)
	body, err := a.core.Request("POST", path, nil, payload)
	if err != nil {
		return err
	}
	_ = body
	fmt.Printf("Review agent triggered for execution %s\n", *execID)
	return nil
}
