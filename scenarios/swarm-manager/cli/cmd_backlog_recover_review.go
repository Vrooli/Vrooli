package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

// cmdBacklogRecoverReview handles `swarm-manager backlog recover-review`.
//
// It is the in-band exit for an item stranded in a review-gated status with no
// live review round behind it (work done out-of-band, a review run that died,
// or a premature in_review). It routes the item to review_pending (default —
// decide it via review-decide) or back to backlog (re-do never-started work),
// and refuses (409) while a real review round is still gathering.
func (a *App) cmdBacklogRecoverReview(args []string) error {
	fs := flag.NewFlagSet("backlog recover-review", flag.ContinueOnError)
	kindFlag := fs.String("kind", "", "Backlog item kind")
	nameFlag := fs.String("name", "", "Backlog item name")
	toFlag := fs.String("to", "review_pending", "Recovery target: review_pending (default) or backlog")
	rationaleFlag := fs.String("rationale", "", "Short explanation of why the item is being recovered")
	decidedByFlag := fs.String("decided-by", "", "Identifier for who recovered the item (defaults to 'user:recover-review')")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlags("kind", *kindFlag, "name", *nameFlag); err != nil {
		return fmt.Errorf("usage: backlog recover-review --kind KIND --name NAME [--to review_pending|backlog] [--rationale MSG] [--decided-by NAME] [--json]\n\n%s", err)
	}

	kind := strings.TrimSpace(*kindFlag)
	name := strings.TrimSpace(*nameFlag)

	payload, err := json.Marshal(map[string]any{
		"to":         strings.TrimSpace(*toFlag),
		"rationale":  strings.TrimSpace(*rationaleFlag),
		"decided_by": strings.TrimSpace(*decidedByFlag),
	})
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}

	body, err := a.core.Request("POST", "/backlog/"+kind+"/"+name+"/recover-review", nil, json.RawMessage(payload))
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	var response struct {
		PriorStatus string `json:"prior_status"`
		Status      string `json:"status"`
		Reason      string `json:"reason,omitempty"`
		RecoveredAt string `json:"recovered_at"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	printSection("Review Recovery")
	fmt.Printf("  Item:   %s/%s\n", kind, name)
	fmt.Printf("  From:   %s\n", response.PriorStatus)
	fmt.Printf("  To:     %s\n", response.Status)
	if response.Reason != "" {
		fmt.Printf("  Reason: %s\n", response.Reason)
	}
	fmt.Printf("  At:     %s\n", response.RecoveredAt)
	return nil
}
