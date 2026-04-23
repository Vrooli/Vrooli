package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

// cmdBacklogReviewDecide handles `swarm-manager backlog review-decide`.
//
// The review-decide command is the ONLY path that flips a backlog item from
// `review_pending` to a terminal status. Regular PATCH requests cannot set
// terminal statuses — they require an explicit decision here so the rationale
// and decider are persisted alongside the review rounds for audit.
func (a *App) cmdBacklogReviewDecide(args []string) error {
	fs := flag.NewFlagSet("backlog review-decide", flag.ContinueOnError)
	kindFlag := fs.String("kind", "", "Backlog item kind")
	nameFlag := fs.String("name", "", "Backlog item name")
	acceptFlag := fs.Bool("accept", false, "Accept review → status = completed")
	failFlag := fs.Bool("fail", false, "Reject review → status = failed")
	followupFlag := fs.Bool("followup", false, "Needs follow-up → status = needs_followup")
	rationaleFlag := fs.String("rationale", "", "Short explanation of the decision (logged alongside the review rounds)")
	decidedByFlag := fs.String("decided-by", "", "Identifier for who made the decision (defaults to 'user')")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlags("kind", *kindFlag, "name", *nameFlag); err != nil {
		return fmt.Errorf("usage: backlog review-decide --kind KIND --name NAME (--accept|--fail|--followup) [--rationale MSG] [--decided-by NAME] [--json]\n\n%s", err)
	}

	var decision string
	switch {
	case *acceptFlag && !*failFlag && !*followupFlag:
		decision = "accept"
	case *failFlag && !*acceptFlag && !*followupFlag:
		decision = "fail"
	case *followupFlag && !*acceptFlag && !*failFlag:
		decision = "followup"
	default:
		return fmt.Errorf("exactly one of --accept, --fail, --followup must be provided")
	}

	kind := strings.TrimSpace(*kindFlag)
	name := strings.TrimSpace(*nameFlag)

	payload, err := json.Marshal(map[string]any{
		"decision":   decision,
		"rationale":  strings.TrimSpace(*rationaleFlag),
		"decided_by": strings.TrimSpace(*decidedByFlag),
	})
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}

	body, err := a.core.Request("POST", "/backlog/"+kind+"/"+name+"/review-decide", nil, json.RawMessage(payload))
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	var response struct {
		Decision  string `json:"decision"`
		Status    string `json:"status"`
		Rationale string `json:"rationale,omitempty"`
		DecidedAt string `json:"decided_at"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	printSection("Review Decision")
	fmt.Printf("  Item:     %s/%s\n", kind, name)
	fmt.Printf("  Decision: %s\n", response.Decision)
	fmt.Printf("  Status:   %s\n", response.Status)
	if response.Rationale != "" {
		fmt.Printf("  Reason:   %s\n", response.Rationale)
	}
	fmt.Printf("  At:       %s\n", response.DecidedAt)
	return nil
}
