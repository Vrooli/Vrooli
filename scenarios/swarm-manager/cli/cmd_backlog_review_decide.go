package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api/apiconnect"
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
	roundFlag := fs.Uint("round", 0, "Review round number")
	acceptFlag := fs.Bool("accept", false, "Accept review → status = completed")
	failFlag := fs.Bool("fail", false, "Reject review → status = failed")
	followupFlag := fs.Bool("followup", false, "Needs follow-up → status = needs_followup")
	dropFlag := fs.Bool("drop", false, "Decided not to pursue → status = dropped (no verdict on the work)")
	rationaleFlag := fs.String("rationale", "", "Short explanation of the decision (logged alongside the review rounds)")
	decidedByFlag := fs.String("decided-by", "", "Identifier for who made the decision")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlags("kind", *kindFlag, "name", *nameFlag, "decided-by", *decidedByFlag); err != nil {
		return fmt.Errorf("usage: backlog review-decide --kind KIND --name NAME --round N --decided-by ACTOR (--accept|--fail|--followup|--drop) [--rationale MSG] [--json]\n\n%s", err)
	}

	// Counted rather than compared pairwise: the exclusivity check stays O(n)
	// as verdicts are added.
	verdicts := []struct {
		set  bool
		name string
	}{
		{*acceptFlag, "accept"},
		{*failFlag, "fail"},
		{*followupFlag, "followup"},
		{*dropFlag, "drop"},
	}
	var decision string
	for _, v := range verdicts {
		if !v.set {
			continue
		}
		if decision != "" {
			return fmt.Errorf("exactly one of --accept, --fail, --followup, --drop must be provided")
		}
		decision = v.name
	}
	if decision == "" {
		return fmt.Errorf("exactly one of --accept, --fail, --followup, --drop must be provided")
	}

	kind := strings.TrimSpace(*kindFlag)
	name := strings.TrimSpace(*nameFlag)
	if *roundFlag == 0 {
		return fmt.Errorf("--round must be a positive integer")
	}

	h, base := cliapp.NewConnectHTTPClient(a.core)
	result, err := apiconnect.NewBacklogServiceClient(h, base).DecideAttempt(context.Background(), connect.NewRequest(&apipb.DecideAttemptRequest{
		SubjectKind: "backlog-item", SubjectRef: kind + "/" + name, RoundNum: uint32(*roundFlag), Decision: decision, Actor: strings.TrimSpace(*decidedByFlag), Rationale: strings.TrimSpace(*rationaleFlag),
	}))
	if err != nil {
		return err
	}
	if *jsonOut {
		body, err := json.Marshal(result.Msg)
		if err != nil {
			return fmt.Errorf("encode response: %w", err)
		}
		fmt.Println(string(body))
		return nil
	}
	response := result.Msg

	printSection("Review Decision")
	fmt.Printf("  Item:     %s/%s\n", kind, name)
	fmt.Printf("  Decision: %s\n", response.Decision)
	fmt.Printf("  Status:   %s\n", response.Status)
	if response.Rationale != "" {
		fmt.Printf("  Reason:   %s\n", response.Rationale)
	}
	fmt.Printf("  At:       %s\n", response.DecidedAt)
	// A terminal accept/fail auto-captures a filled, searchable record of the
	// work (the recursive-learning write-side). Surface its id with the enrich
	// path so the agent can improve the narrative — records are born immutable,
	// so enrichment is via `records supersede`, not `records edit`.
	if response.GetRecordId() != "" {
		fmt.Printf("  Record:   %s (auto-captured; enrich via `swarm-manager records supersede %s`)\n",
			response.GetRecordId(), response.GetRecordId())
	}
	return nil
}
