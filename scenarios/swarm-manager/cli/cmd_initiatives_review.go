package main

// Initiative-review CLI commands.
//
// These wrap the `/api/v1/initiatives/{name}/review/*` HTTP surface from
// api/internal/initiativereview. Unlike backlog review-decide (which flips a
// single item), these commands operate at the initiative scope — trigger a
// review round, list rounds and decisions, and issue the user's final verdict.

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

type reviewRoundSummary struct {
	Number    int    `json:"number"`
	Slug      string `json:"slug,omitempty"`
	Kind      string `json:"kind,omitempty"`
	Name      string `json:"name,omitempty"`
	Status    string `json:"status,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
	// Verdict captured at round level when the review agent produces one.
	// The user-facing terminal verdict lives in DecisionRecord, not here.
	AgentVerdict string `json:"agent_verdict,omitempty"`
	Summary      string `json:"summary,omitempty"`
}

type reviewRoundsResponse struct {
	Rounds []reviewRoundSummary `json:"rounds"`
}

type reviewDecisionsResponse struct {
	Decisions []reviewDecisionRecord `json:"decisions"`
}

type reviewDecisionRecord struct {
	Verdict     string `json:"verdict"`
	Status      string `json:"status"`
	Rationale   string `json:"rationale,omitempty"`
	DecidedBy   string `json:"decided_by,omitempty"`
	DecidedAt   string `json:"decided_at,omitempty"`
	PriorStatus string `json:"prior_status,omitempty"`
	Round       int    `json:"round,omitempty"`
}

type reviewTriggerResponse struct {
	Started bool   `json:"started"`
	Reason  string `json:"reason,omitempty"`
	Round   int    `json:"round,omitempty"`
	RunID   string `json:"run_id,omitempty"`
}

type reviewDecideResponse struct {
	Initiative string `json:"initiative"`
	Verdict    string `json:"verdict"`
	Status     string `json:"status"`
	Rationale  string `json:"rationale,omitempty"`
	DecidedAt  string `json:"decided_at"`
}

// --- review-list -------------------------------------------------------

func (a *App) cmdInitiativesReviewList(args []string) error {
	fs := flag.NewFlagSet("initiatives review-list", flag.ContinueOnError)
	nameFlag := fs.String("name", "", "Initiative name")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("name", *nameFlag); err != nil {
		return fmt.Errorf("usage: initiatives review-list --name NAME [--json]\n\n%s", err)
	}
	name := strings.TrimSpace(*nameFlag)

	body, err := a.core.Get("/initiatives/"+name+"/review", nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	resp, err := decodeResponse[reviewRoundsResponse](body)
	if err != nil {
		return err
	}

	printSection("Summary")
	if len(resp.Rounds) == 0 {
		fmt.Printf("  No review rounds for initiative %s.\n", name)
		printCommandListSection("Next Steps", []string{
			cliCommand("initiatives", "review-trigger", "--name", name),
		})
		return nil
	}
	fmt.Printf("  Found %d review round(s) for initiative %s\n", len(resp.Rounds), name)

	printSection("Rounds")
	for _, r := range resp.Rounds {
		fmt.Printf("  - round %d", r.Number)
		if r.Kind != "" && r.Name != "" {
			fmt.Printf(" (%s/%s)", r.Kind, r.Name)
		}
		if r.Status != "" {
			fmt.Printf(" — %s", r.Status)
		}
		if r.AgentVerdict != "" {
			fmt.Printf(" verdict=%s", r.AgentVerdict)
		}
		fmt.Println()
		if preview := submissionPreview(r.Summary); preview != "" {
			fmt.Printf("      summary: %s\n", preview)
		}
	}
	return nil
}

// --- review-get --------------------------------------------------------

func (a *App) cmdInitiativesReviewGet(args []string) error {
	fs := flag.NewFlagSet("initiatives review-get", flag.ContinueOnError)
	nameFlag := fs.String("name", "", "Initiative name")
	roundFlag := parseRoundFlag(fs, "round", "Round number")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("name", *nameFlag); err != nil {
		return fmt.Errorf("usage: initiatives review-get --name NAME --round N [--json]\n\n%s", err)
	}
	if *roundFlag <= 0 {
		return fmt.Errorf("--round must be a positive integer")
	}
	name := strings.TrimSpace(*nameFlag)

	body, err := a.core.Get(fmt.Sprintf("/initiatives/%s/review/%d", name, *roundFlag), nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	// Round shape comes from internal/review and varies by owner type. Pretty
	// print the top-level fields we know about and dump the rest as raw JSON
	// so the CLI doesn't rot every time a field is added server-side.
	var round map[string]any
	if err := json.Unmarshal(body, &round); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	printSection("Round")
	for _, key := range []string{"number", "slug", "kind", "name", "status", "created_at", "updated_at"} {
		if v, ok := round[key]; ok && v != nil {
			fmt.Printf("  %s: %v\n", key, v)
		}
	}
	// Any remaining fields — render as pretty JSON for inspection.
	extra := make(map[string]any, len(round))
	for k, v := range round {
		switch k {
		case "number", "slug", "kind", "name", "status", "created_at", "updated_at":
			continue
		}
		extra[k] = v
	}
	if len(extra) > 0 {
		rest, _ := json.MarshalIndent(extra, "  ", "  ")
		printSection("Details")
		fmt.Printf("  %s\n", string(rest))
	}
	return nil
}

// --- review-trigger ----------------------------------------------------

func (a *App) cmdInitiativesReviewTrigger(args []string) error {
	fs := flag.NewFlagSet("initiatives review-trigger", flag.ContinueOnError)
	nameFlag := fs.String("name", "", "Initiative name")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("name", *nameFlag); err != nil {
		return fmt.Errorf("usage: initiatives review-trigger --name NAME [--json]\n\n%s", err)
	}
	name := strings.TrimSpace(*nameFlag)

	body, err := a.core.Request("POST", "/initiatives/"+name+"/review/trigger", nil, nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	resp, err := decodeResponse[reviewTriggerResponse](body)
	if err != nil {
		return err
	}

	printSection("Trigger")
	fmt.Printf("  Started: %t\n", resp.Started)
	if resp.Reason != "" {
		fmt.Printf("  Reason:  %s\n", resp.Reason)
	}
	if resp.Round > 0 {
		fmt.Printf("  Round:   %d\n", resp.Round)
	}
	if resp.RunID != "" {
		fmt.Printf("  Run ID:  %s\n", resp.RunID)
	}
	printCommandListSection("Next Steps", []string{
		cliCommand("initiatives", "review-list", "--name", name),
	})
	return nil
}

// --- review-decide -----------------------------------------------------

// cmdInitiativesReviewDecide is the initiative-level counterpart to
// `backlog review-decide`. It's the only path that flips an initiative from
// review_pending to a terminal status, carrying rationale + decider for
// audit.
func (a *App) cmdInitiativesReviewDecide(args []string) error {
	fs := flag.NewFlagSet("initiatives review-decide", flag.ContinueOnError)
	nameFlag := fs.String("name", "", "Initiative name")
	acceptFlag := fs.Bool("accept", false, "Accept the review → status = completed")
	failFlag := fs.Bool("fail", false, "Reject the review → status = failed")
	followupFlag := fs.Bool("followup", false, "Needs follow-up → status = needs_followup")
	rationaleFlag := fs.String("rationale", "", "Short explanation of the decision")
	decidedByFlag := fs.String("decided-by", "", "Identifier for the deciding user (defaults to 'user')")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("name", *nameFlag); err != nil {
		return fmt.Errorf("usage: initiatives review-decide --name NAME (--accept|--fail|--followup) [--rationale MSG] [--decided-by WHO] [--json]\n\n%s", err)
	}
	var verdict string
	switch {
	case *acceptFlag && !*failFlag && !*followupFlag:
		verdict = "accept"
	case *failFlag && !*acceptFlag && !*followupFlag:
		verdict = "fail"
	case *followupFlag && !*acceptFlag && !*failFlag:
		verdict = "followup"
	default:
		return fmt.Errorf("exactly one of --accept, --fail, --followup must be provided")
	}

	name := strings.TrimSpace(*nameFlag)
	payload, err := json.Marshal(map[string]any{
		"verdict":    verdict,
		"rationale":  strings.TrimSpace(*rationaleFlag),
		"decided_by": strings.TrimSpace(*decidedByFlag),
	})
	if err != nil {
		return fmt.Errorf("encode request: %w", err)
	}

	body, err := a.core.Request("POST", "/initiatives/"+name+"/review/decide", nil, json.RawMessage(payload))
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	resp, err := decodeResponse[reviewDecideResponse](body)
	if err != nil {
		return err
	}

	printSection("Review Decision")
	fmt.Printf("  Initiative: %s\n", resp.Initiative)
	fmt.Printf("  Verdict:    %s\n", resp.Verdict)
	fmt.Printf("  Status:     %s\n", resp.Status)
	if resp.Rationale != "" {
		fmt.Printf("  Reason:     %s\n", resp.Rationale)
	}
	if resp.DecidedAt != "" {
		fmt.Printf("  At:         %s\n", resp.DecidedAt)
	}
	return nil
}

// --- review-decisions --------------------------------------------------

func (a *App) cmdInitiativesReviewDecisions(args []string) error {
	fs := flag.NewFlagSet("initiatives review-decisions", flag.ContinueOnError)
	nameFlag := fs.String("name", "", "Initiative name")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("name", *nameFlag); err != nil {
		return fmt.Errorf("usage: initiatives review-decisions --name NAME [--json]\n\n%s", err)
	}
	name := strings.TrimSpace(*nameFlag)

	body, err := a.core.Get("/initiatives/"+name+"/review/decisions", nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	resp, err := decodeResponse[reviewDecisionsResponse](body)
	if err != nil {
		return err
	}
	printSection("Decisions")
	if len(resp.Decisions) == 0 {
		fmt.Printf("  No review decisions for initiative %s.\n", name)
		return nil
	}
	for _, d := range resp.Decisions {
		fmt.Printf("  - verdict=%s status=%s", d.Verdict, d.Status)
		if d.Round > 0 {
			fmt.Printf(" round=%d", d.Round)
		}
		if d.PriorStatus != "" {
			fmt.Printf(" (was %s)", d.PriorStatus)
		}
		fmt.Println()
		if d.Rationale != "" {
			fmt.Printf("      rationale: %s\n", d.Rationale)
		}
		if d.DecidedBy != "" {
			fmt.Printf("      by: %s at %s\n", d.DecidedBy, d.DecidedAt)
		} else if d.DecidedAt != "" {
			fmt.Printf("      at: %s\n", d.DecidedAt)
		}
	}
	return nil
}
