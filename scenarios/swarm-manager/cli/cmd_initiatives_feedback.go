package main

// Feedback-round CLI commands for swarm-manager initiatives.
//
// These commands are the CLI counterpart to the feedback HTTP surface in
// api/internal/feedback. Every command is a thin transport wrapper: it
// collects flags, builds the request, and prints a human summary plus the
// same JSON that the HTTP layer returns (via --json).

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

// feedbackRoundSummary is the minimal view used by list / get output. Keeps
// the CLI independent from the server's internal Round shape while still
// rendering the fields a human cares about.
type feedbackRoundSummary struct {
	Number            int    `json:"number"`
	Slug              string `json:"slug"`
	Type              string `json:"type"`
	Status            string `json:"status"`
	CurrentProposalID string `json:"current_proposal_id,omitempty"`
	RunID             string `json:"run_id,omitempty"`
	NeedsRevision     bool   `json:"needs_revision,omitempty"`
	CreatedAt         string `json:"created_at,omitempty"`
	UpdatedAt         string `json:"updated_at,omitempty"`
	Submission        struct {
		Text string `json:"text,omitempty"`
	} `json:"submission"`
	Thread []struct {
		Role    string `json:"role"`
		Content string `json:"content,omitempty"`
	} `json:"thread,omitempty"`
	Proposals []struct {
		ID           string `json:"id"`
		MessageIndex int    `json:"message_index"`
		CreatedAt    string `json:"created_at,omitempty"`
		Proposal     struct {
			Form      string           `json:"form"`
			Mutations []proposalMutDTO `json:"mutations,omitempty"`
		} `json:"proposal"`
	} `json:"proposals,omitempty"`
	Decision *struct {
		Kind                string   `json:"kind"`
		AcceptedMutationIDs []string `json:"accepted_mutation_ids,omitempty"`
		RejectedMutationIDs []string `json:"rejected_mutation_ids,omitempty"`
		Rationale           string   `json:"rationale,omitempty"`
		DecidedAt           string   `json:"decided_at,omitempty"`
		DecidedBy           string   `json:"decided_by,omitempty"`
	} `json:"decision,omitempty"`
}

type proposalMutDTO struct {
	ID     string `json:"id"`
	Op     string `json:"op"`
	Target string `json:"target,omitempty"`
}

type feedbackListResponse struct {
	Rounds []feedbackRoundSummary `json:"rounds"`
	Count  int                    `json:"count"`
}

func parseRoundFlag(fs *flag.FlagSet, name, help string) *int {
	return fs.Int(name, 0, help)
}

// --- feedback-list -----------------------------------------------------

func (a *App) cmdInitiativesFeedbackList(args []string) error {
	fs := flag.NewFlagSet("initiatives feedback-list", flag.ContinueOnError)
	nameFlag := fs.String("name", "", "Initiative name")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("name", *nameFlag); err != nil {
		return fmt.Errorf("usage: initiatives feedback-list --name NAME [--json]\n\n%s", err)
	}
	name := strings.TrimSpace(*nameFlag)

	body, err := a.core.Get("/initiatives/"+name+"/feedback", nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	resp, err := decodeResponse[feedbackListResponse](body)
	if err != nil {
		return err
	}

	printSection("Summary")
	if len(resp.Rounds) == 0 {
		fmt.Printf("  No feedback rounds for initiative %s.\n", name)
		printCommandListSection("Next Steps", []string{
			cliCommand("initiatives", "feedback-submit", "--name", name, "--type", "feedback", "--text", "\"...\""),
		})
		return nil
	}
	fmt.Printf("  Found %d round(s) for initiative %s\n", resp.Count, name)

	printSection("Rounds")
	for _, r := range resp.Rounds {
		fmt.Printf("  - round %d %q (%s, %s)\n", r.Number, r.Slug, r.Type, r.Status)
		if preview := submissionPreview(r.Submission.Text); preview != "" {
			fmt.Printf("      submission: %s\n", preview)
		}
		if len(r.Proposals) > 0 {
			fmt.Printf("      proposals: %d (current=%s)\n", len(r.Proposals), r.CurrentProposalID)
		}
		if r.Decision != nil {
			fmt.Printf("      decision: %s", r.Decision.Kind)
			if len(r.Decision.AcceptedMutationIDs) > 0 {
				fmt.Printf(" accepted=%s", strings.Join(r.Decision.AcceptedMutationIDs, ","))
			}
			if r.Decision.Rationale != "" {
				fmt.Printf(" — %s", r.Decision.Rationale)
			}
			fmt.Println()
		}
	}

	first := resp.Rounds[0]
	printCommandListSection("Next Steps", []string{
		cliCommand("initiatives", "feedback-get", "--name", name, "--round", fmt.Sprint(first.Number)),
	})
	return nil
}

// --- feedback-get ------------------------------------------------------

func (a *App) cmdInitiativesFeedbackGet(args []string) error {
	fs := flag.NewFlagSet("initiatives feedback-get", flag.ContinueOnError)
	nameFlag := fs.String("name", "", "Initiative name")
	roundFlag := parseRoundFlag(fs, "round", "Round number")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("name", *nameFlag); err != nil {
		return fmt.Errorf("usage: initiatives feedback-get --name NAME --round N [--json]\n\n%s", err)
	}
	if *roundFlag <= 0 {
		return fmt.Errorf("--round must be a positive integer")
	}
	name := strings.TrimSpace(*nameFlag)

	body, err := a.core.Get(fmt.Sprintf("/initiatives/%s/feedback/%d", name, *roundFlag), nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	round, err := decodeResponse[feedbackRoundSummary](body)
	if err != nil {
		return err
	}
	printFeedbackRound(round)
	printCommandListSection("Next Steps", []string{
		cliCommand("initiatives", "feedback-continue", "--name", name, "--round", fmt.Sprint(round.Number), "--text", "\"revise because...\""),
		cliCommand("initiatives", "feedback-decide", "--name", name, "--round", fmt.Sprint(round.Number), "--accept"),
	})
	return nil
}

// --- feedback-submit ---------------------------------------------------

func (a *App) cmdInitiativesFeedbackSubmit(args []string) error {
	fs := flag.NewFlagSet("initiatives feedback-submit", flag.ContinueOnError)
	nameFlag := fs.String("name", "", "Initiative name")
	typeFlag := fs.String("type", "feedback", "Round type (feedback|note)")
	textFlag := fs.String("text", "", "Submission text")
	slugFlag := fs.String("slug", "", "Round slug hint (optional)")
	overrideFlag := fs.Bool("override", false, "Acquire the initiative lock even if it is held")
	decidedByFlag := fs.String("decided-by", "", "Identifier for the submitting user (defaults to server's current user)")
	var files stringSlice
	fs.Var(&files, "file", "Local file to attach (repeatable)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("name", *nameFlag); err != nil {
		return fmt.Errorf("usage: initiatives feedback-submit --name NAME --type feedback|note --text MSG [--file PATH ...] [--slug SLUG] [--override] [--decided-by WHO] [--json]\n\n%s", err)
	}
	roundType := strings.ToLower(strings.TrimSpace(*typeFlag))
	switch roundType {
	case "feedback", "note":
		// valid
	default:
		return fmt.Errorf("--type must be 'feedback' or 'note' (got %q)", roundType)
	}
	text := strings.TrimSpace(*textFlag)
	if text == "" && len(files) == 0 {
		return fmt.Errorf("--text or --file is required")
	}
	name := strings.TrimSpace(*nameFlag)

	var (
		body []byte
		err  error
	)
	if len(files) > 0 {
		body, err = a.postFeedbackMultipart("/initiatives/"+name+"/feedback", feedbackFormFields{
			Type:      roundType,
			Text:      text,
			Slug:      strings.TrimSpace(*slugFlag),
			Override:  *overrideFlag,
			DecidedBy: strings.TrimSpace(*decidedByFlag),
			Files:     files,
		})
	} else {
		payload, pErr := json.Marshal(map[string]any{
			"type":       roundType,
			"text":       text,
			"slug":       strings.TrimSpace(*slugFlag),
			"override":   *overrideFlag,
			"decided_by": strings.TrimSpace(*decidedByFlag),
		})
		if pErr != nil {
			return fmt.Errorf("encode request: %w", pErr)
		}
		body, err = a.core.Request("POST", "/initiatives/"+name+"/feedback", nil, payload)
	}
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	round, err := decodeResponse[feedbackRoundSummary](body)
	if err != nil {
		return err
	}
	printSection("Submitted")
	fmt.Printf("  Initiative: %s\n", name)
	fmt.Printf("  Round:      %d (%s)\n", round.Number, round.Slug)
	fmt.Printf("  Type:       %s\n", round.Type)
	fmt.Printf("  Status:     %s\n", round.Status)
	if len(files) > 0 {
		fmt.Printf("  Attached:   %d file(s)\n", len(files))
	}
	printCommandListSection("Next Steps", []string{
		cliCommand("initiatives", "feedback-get", "--name", name, "--round", fmt.Sprint(round.Number)),
	})
	return nil
}

// --- feedback-continue -------------------------------------------------

func (a *App) cmdInitiativesFeedbackContinue(args []string) error {
	fs := flag.NewFlagSet("initiatives feedback-continue", flag.ContinueOnError)
	nameFlag := fs.String("name", "", "Initiative name")
	roundFlag := parseRoundFlag(fs, "round", "Round number")
	textFlag := fs.String("text", "", "Continuation text")
	decidedByFlag := fs.String("decided-by", "", "Identifier for the submitting user")
	var files stringSlice
	fs.Var(&files, "file", "Local file to attach (repeatable)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("name", *nameFlag); err != nil {
		return fmt.Errorf("usage: initiatives feedback-continue --name NAME --round N --text MSG [--file PATH ...] [--decided-by WHO] [--json]\n\n%s", err)
	}
	if *roundFlag <= 0 {
		return fmt.Errorf("--round must be a positive integer")
	}
	text := strings.TrimSpace(*textFlag)
	if text == "" && len(files) == 0 {
		return fmt.Errorf("--text or --file is required")
	}
	name := strings.TrimSpace(*nameFlag)
	endpoint := fmt.Sprintf("/initiatives/%s/feedback/%d/continue", name, *roundFlag)

	var (
		body []byte
		err  error
	)
	if len(files) > 0 {
		body, err = a.postFeedbackMultipart(endpoint, feedbackFormFields{
			Text:      text,
			DecidedBy: strings.TrimSpace(*decidedByFlag),
			Files:     files,
		})
	} else {
		payload, pErr := json.Marshal(map[string]any{
			"text":       text,
			"decided_by": strings.TrimSpace(*decidedByFlag),
		})
		if pErr != nil {
			return fmt.Errorf("encode request: %w", pErr)
		}
		body, err = a.core.Request("POST", endpoint, nil, payload)
	}
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	round, err := decodeResponse[feedbackRoundSummary](body)
	if err != nil {
		return err
	}
	printSection("Continued")
	fmt.Printf("  Round:  %d\n", round.Number)
	fmt.Printf("  Status: %s\n", round.Status)
	return nil
}

// --- feedback-decide ---------------------------------------------------

func (a *App) cmdInitiativesFeedbackDecide(args []string) error {
	fs := flag.NewFlagSet("initiatives feedback-decide", flag.ContinueOnError)
	nameFlag := fs.String("name", "", "Initiative name")
	roundFlag := parseRoundFlag(fs, "round", "Round number")
	acceptFlag := fs.Bool("accept", false, "Accept the proposal (partial_accept if --mutations is set)")
	rejectFlag := fs.Bool("reject", false, "Reject the proposal")
	dismissFlag := fs.Bool("dismiss", false, "Dismiss the round without applying")
	mutationsFlag := fs.String("mutations", "", "Comma-separated accepted mutation IDs (e.g. m1,m3)")
	rationaleFlag := fs.String("rationale", "", "Short explanation recorded with the decision")
	decidedByFlag := fs.String("decided-by", "", "Identifier for the deciding user")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("name", *nameFlag); err != nil {
		return fmt.Errorf("usage: initiatives feedback-decide --name NAME --round N (--accept|--reject|--dismiss) [--mutations m1,m3] [--rationale MSG] [--decided-by WHO] [--json]\n\n%s", err)
	}
	if *roundFlag <= 0 {
		return fmt.Errorf("--round must be a positive integer")
	}

	if countTrue(*acceptFlag, *rejectFlag, *dismissFlag) != 1 {
		return fmt.Errorf("exactly one of --accept, --reject, --dismiss must be provided")
	}

	mutations := parseCommaSeparated(*mutationsFlag)
	kind, err := feedbackDecisionKind(*acceptFlag, *rejectFlag, *dismissFlag, mutations)
	if err != nil {
		return err
	}

	name := strings.TrimSpace(*nameFlag)
	rationale := strings.TrimSpace(*rationaleFlag)
	decidedBy := strings.TrimSpace(*decidedByFlag)

	endpoint, payload, err := buildFeedbackDecideRequest(
		name, *roundFlag, kind, rationale, decidedBy, mutations,
		*dismissFlag && strings.TrimSpace(*mutationsFlag) == "",
	)
	if err != nil {
		return err
	}

	body, err := a.core.Request("POST", endpoint, nil, json.RawMessage(payload))
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	// Accept/reject/partial_accept return {round, apply_result}. Dismiss
	// returns the round directly. Decode permissively.
	var wrap struct {
		Round       *feedbackRoundSummary `json:"round,omitempty"`
		ApplyResult *struct {
			Outcomes []struct {
				MutationID string `json:"mutation_id"`
				Op         string `json:"op"`
				Target     string `json:"target,omitempty"`
				Applied    bool   `json:"applied"`
				Skipped    bool   `json:"skipped,omitempty"`
				Error      string `json:"error,omitempty"`
			} `json:"outcomes,omitempty"`
			Applied int `json:"applied"`
			Failed  int `json:"failed"`
			Skipped int `json:"skipped"`
		} `json:"apply_result,omitempty"`
	}
	if err := json.Unmarshal(body, &wrap); err != nil || wrap.Round == nil {
		// Might be a bare round (dismiss).
		var r feedbackRoundSummary
		if err := json.Unmarshal(body, &r); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
		wrap.Round = &r
	}

	printSection("Decision")
	fmt.Printf("  Round:      %d\n", wrap.Round.Number)
	fmt.Printf("  Kind:       %s\n", kind)
	fmt.Printf("  Status:     %s\n", wrap.Round.Status)
	if wrap.Round.Decision != nil {
		if wrap.Round.Decision.Rationale != "" {
			fmt.Printf("  Rationale:  %s\n", wrap.Round.Decision.Rationale)
		}
		if wrap.Round.Decision.DecidedAt != "" {
			fmt.Printf("  At:         %s\n", wrap.Round.Decision.DecidedAt)
		}
	}
	if wrap.ApplyResult != nil {
		printSection("Apply Result")
		fmt.Printf("  Applied:    %d\n", wrap.ApplyResult.Applied)
		fmt.Printf("  Failed:     %d\n", wrap.ApplyResult.Failed)
		fmt.Printf("  Skipped:    %d\n", wrap.ApplyResult.Skipped)
		for _, o := range wrap.ApplyResult.Outcomes {
			fmt.Println(formatApplyOutcome(o.MutationID, o.Op, o.Target, o.Error, o.Applied, o.Skipped))
		}
	}
	return nil
}

// countTrue returns how many of the given booleans are true.
func countTrue(flags ...bool) int {
	n := 0
	for _, f := range flags {
		if f {
			n++
		}
	}
	return n
}

// buildFeedbackDecideRequest builds the endpoint path and JSON request body for
// a feedback decision. When useDismissEndpoint is set, it targets the ergonomic
// /dismiss sibling endpoint (which accepts a smaller body without a kind);
// otherwise it targets /decide with the full decision payload.
func buildFeedbackDecideRequest(name string, round int, kind, rationale, decidedBy string, mutations []string, useDismissEndpoint bool) (string, []byte, error) {
	if useDismissEndpoint {
		// /dismiss is the ergonomic sibling endpoint; both paths reach the
		// same Decide code with kind=dismiss, but /dismiss accepts a
		// smaller body (no kind required) so we use it for the dismiss
		// flow here for parity with the UI and to keep parity with the
		// plan's wire shape.
		endpoint := fmt.Sprintf("/initiatives/%s/feedback/%d/dismiss", name, round)
		payload, err := json.Marshal(map[string]any{
			"rationale":  rationale,
			"decided_by": decidedBy,
		})
		if err != nil {
			return "", nil, fmt.Errorf("encode request: %w", err)
		}
		return endpoint, payload, nil
	}

	endpoint := fmt.Sprintf("/initiatives/%s/feedback/%d/decide", name, round)
	payloadMap := map[string]any{
		"kind":       kind,
		"rationale":  rationale,
		"decided_by": decidedBy,
	}
	if len(mutations) > 0 {
		payloadMap["accepted_mutation_ids"] = mutations
	}
	payload, err := json.Marshal(payloadMap)
	if err != nil {
		return "", nil, fmt.Errorf("encode request: %w", err)
	}
	return endpoint, payload, nil
}

// feedbackDecisionKind validates the mutually-exclusive accept/reject/dismiss
// flags against any provided mutation IDs and returns the wire "kind" value.
func feedbackDecisionKind(accept, reject, dismiss bool, mutations []string) (string, error) {
	switch {
	case accept:
		if len(mutations) > 0 {
			return "partial_accept", nil
		}
		return "accept", nil
	case reject:
		if len(mutations) > 0 {
			return "", fmt.Errorf("--mutations is only valid with --accept")
		}
		return "reject", nil
	case dismiss:
		if len(mutations) > 0 {
			return "", fmt.Errorf("--mutations is only valid with --accept")
		}
		return "dismiss", nil
	}
	return "", nil
}

// formatApplyOutcome renders a single mutation apply outcome as a display line.
func formatApplyOutcome(mutationID, op, target, errMsg string, applied, skipped bool) string {
	badge := "applied"
	if skipped {
		badge = "skipped"
	} else if !applied {
		badge = "failed"
	}
	line := fmt.Sprintf("  - %s [%s] %s", mutationID, badge, op)
	if target != "" {
		line += " " + target
	}
	if errMsg != "" {
		line += " — " + errMsg
	}
	return line
}

// --- feedback-cancel ---------------------------------------------------

// cmdInitiativesFeedbackCancel forces a stuck agent_thinking round into
// dismissed by hitting POST /feedback/{round}/cancel. It is the user-
// facing escape hatch when the agent run has crashed or the user no
// longer wants to wait. Mirrors the wire shape of /dismiss but takes a
// different route on the API side so it can be the only path that calls
// agent-manager StopRun + releases the lock.
func (a *App) cmdInitiativesFeedbackCancel(args []string) error {
	fs := flag.NewFlagSet("initiatives feedback-cancel", flag.ContinueOnError)
	nameFlag := fs.String("name", "", "Initiative name")
	roundFlag := parseRoundFlag(fs, "round", "Round number")
	rationaleFlag := fs.String("rationale", "", "Short explanation recorded with the cancel decision")
	decidedByFlag := fs.String("decided-by", "", "Identifier for the user issuing the cancel")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("name", *nameFlag); err != nil {
		return fmt.Errorf("usage: initiatives feedback-cancel --name NAME --round N [--rationale MSG] [--decided-by WHO] [--json]\n\n%s", err)
	}
	if *roundFlag <= 0 {
		return fmt.Errorf("--round must be a positive integer")
	}

	name := strings.TrimSpace(*nameFlag)
	payload, err := json.Marshal(map[string]any{
		"rationale":  strings.TrimSpace(*rationaleFlag),
		"decided_by": strings.TrimSpace(*decidedByFlag),
	})
	if err != nil {
		return fmt.Errorf("encode request: %w", err)
	}

	endpoint := fmt.Sprintf("/initiatives/%s/feedback/%d/cancel", name, *roundFlag)
	body, err := a.core.Request("POST", endpoint, nil, json.RawMessage(payload))
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	var r feedbackRoundSummary
	if err := json.Unmarshal(body, &r); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	printSection("Cancelled")
	fmt.Printf("  Round:      %d\n", r.Number)
	fmt.Printf("  Status:     %s\n", r.Status)
	if r.Decision != nil {
		if r.Decision.Rationale != "" {
			fmt.Printf("  Rationale:  %s\n", r.Decision.Rationale)
		}
		if r.Decision.DecidedAt != "" {
			fmt.Printf("  At:         %s\n", r.Decision.DecidedAt)
		}
	}
	return nil
}

// cmdInitiativesFeedbackDelete permanently removes a terminal feedback round
// from disk. Only allowed on rounds that have already reached
// applied/rejected/dismissed — for in-flight rounds the API returns 409 and
// the user is expected to call feedback-cancel first.
func (a *App) cmdInitiativesFeedbackDelete(args []string) error {
	fs := flag.NewFlagSet("initiatives feedback-delete", flag.ContinueOnError)
	nameFlag := fs.String("name", "", "Initiative name")
	roundFlag := parseRoundFlag(fs, "round", "Round number")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("name", *nameFlag); err != nil {
		return fmt.Errorf("usage: initiatives feedback-delete --name NAME --round N\n\n%s", err)
	}
	if *roundFlag <= 0 {
		return fmt.Errorf("--round must be a positive integer")
	}
	name := strings.TrimSpace(*nameFlag)
	endpoint := fmt.Sprintf("/initiatives/%s/feedback/%d", name, *roundFlag)
	if _, err := a.core.Request("DELETE", endpoint, nil, nil); err != nil {
		return err
	}
	printSection("Deleted")
	fmt.Printf("  Initiative: %s\n", name)
	fmt.Printf("  Round:      %d\n", *roundFlag)
	return nil
}

// --- feedback-lock -----------------------------------------------------

func (a *App) cmdInitiativesFeedbackLock(args []string) error {
	fs := flag.NewFlagSet("initiatives feedback-lock", flag.ContinueOnError)
	nameFlag := fs.String("name", "", "Initiative name")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("name", *nameFlag); err != nil {
		return fmt.Errorf("usage: initiatives feedback-lock --name NAME [--json]\n\n%s", err)
	}
	name := strings.TrimSpace(*nameFlag)

	body, err := a.core.Get("/initiatives/"+name+"/feedback/lock", nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	var payload struct {
		Locked bool `json:"locked"`
		Holder *struct {
			RunID      string `json:"run_id,omitempty"`
			Purpose    string `json:"purpose,omitempty"`
			AcquiredAt string `json:"acquired_at,omitempty"`
		} `json:"holder,omitempty"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	printSection("Lock")
	if !payload.Locked {
		fmt.Printf("  Initiative %s: unlocked\n", name)
		return nil
	}
	fmt.Printf("  Initiative %s: LOCKED\n", name)
	if payload.Holder != nil {
		if payload.Holder.RunID != "" {
			fmt.Printf("  Run ID:    %s\n", payload.Holder.RunID)
		}
		if payload.Holder.Purpose != "" {
			fmt.Printf("  Purpose:   %s\n", payload.Holder.Purpose)
		}
		if payload.Holder.AcquiredAt != "" {
			fmt.Printf("  Acquired:  %s\n", payload.Holder.AcquiredAt)
		}
	}
	printCommandListSection("Next Steps", []string{
		cliCommand("initiatives", "feedback-submit", "--name", name, "--type", "feedback", "--text", "\"...\"", "--override"),
	})
	return nil
}

// --- helpers -----------------------------------------------------------

type feedbackFormFields struct {
	Type      string // empty on continue
	Text      string
	Slug      string
	Override  bool
	DecidedBy string
	Files     []string
}

// postFeedbackMultipart builds a multipart/form-data body matching the
// server's parseStartRequest / Continue multipart expectations and posts it.
// Consolidated here so start/continue share the same wire format.
func (a *App) postFeedbackMultipart(endpoint string, fields feedbackFormFields) ([]byte, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	writeField := func(k, v string) error {
		if v == "" {
			return nil
		}
		return writer.WriteField(k, v)
	}
	if err := writeField("type", fields.Type); err != nil {
		return nil, err
	}
	if err := writeField("text", fields.Text); err != nil {
		return nil, err
	}
	if err := writeField("slug", fields.Slug); err != nil {
		return nil, err
	}
	if err := writeField("decided_by", fields.DecidedBy); err != nil {
		return nil, err
	}
	if fields.Override {
		if err := writer.WriteField("override", "true"); err != nil {
			return nil, err
		}
	}

	for _, path := range fields.Files {
		f, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("open %s: %w", path, err)
		}
		part, err := writer.CreateFormFile("files", filepath.Base(path))
		if err != nil {
			f.Close()
			return nil, fmt.Errorf("create form file %s: %w", path, err)
		}
		if _, err := io.Copy(part, f); err != nil {
			f.Close()
			return nil, fmt.Errorf("copy %s: %w", path, err)
		}
		f.Close()
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("finalize multipart: %w", err)
	}
	body, err := a.requestMultipart(http.MethodPost, endpoint, buf.Bytes(), writer.FormDataContentType())
	if err != nil {
		return nil, err
	}
	return body, nil
}

func submissionPreview(text string) string {
	t := strings.TrimSpace(text)
	if t == "" {
		return ""
	}
	const maxLen = 100
	if len(t) <= maxLen {
		return t
	}
	return t[:maxLen] + "..."
}

func printFeedbackRound(r feedbackRoundSummary) {
	printSection("Round")
	fmt.Printf("  Number:   %d\n", r.Number)
	fmt.Printf("  Slug:     %s\n", r.Slug)
	fmt.Printf("  Type:     %s\n", r.Type)
	fmt.Printf("  Status:   %s\n", r.Status)
	if r.RunID != "" {
		fmt.Printf("  Run ID:   %s\n", r.RunID)
	}
	if r.NeedsRevision {
		fmt.Printf("  Needs revision: yes\n")
	}
	if r.CreatedAt != "" {
		fmt.Printf("  Created:  %s\n", r.CreatedAt)
	}
	if r.UpdatedAt != "" {
		fmt.Printf("  Updated:  %s\n", r.UpdatedAt)
	}

	if r.Submission.Text != "" {
		printSection("Submission")
		fmt.Printf("  %s\n", r.Submission.Text)
	}

	if len(r.Thread) > 0 {
		printSection("Thread")
		for i, m := range r.Thread {
			preview := submissionPreview(m.Content)
			fmt.Printf("  [%d] %s: %s\n", i, m.Role, preview)
		}
	}

	if len(r.Proposals) > 0 {
		printSection("Proposals")
		for _, p := range r.Proposals {
			marker := " "
			if p.ID == r.CurrentProposalID {
				marker = "*"
			}
			fmt.Printf("  %s %s (form=%s, from msg #%d, %d mutation(s))\n",
				marker, p.ID, p.Proposal.Form, p.MessageIndex, len(p.Proposal.Mutations))
			for _, m := range p.Proposal.Mutations {
				target := m.Target
				if target == "" {
					target = "-"
				}
				fmt.Printf("       - %s %s %s\n", m.ID, m.Op, target)
			}
		}
	}

	if r.Decision != nil {
		printSection("Decision")
		fmt.Printf("  Kind:       %s\n", r.Decision.Kind)
		if len(r.Decision.AcceptedMutationIDs) > 0 {
			fmt.Printf("  Accepted:   %s\n", strings.Join(r.Decision.AcceptedMutationIDs, ", "))
		}
		if len(r.Decision.RejectedMutationIDs) > 0 {
			fmt.Printf("  Rejected:   %s\n", strings.Join(r.Decision.RejectedMutationIDs, ", "))
		}
		if r.Decision.Rationale != "" {
			fmt.Printf("  Rationale:  %s\n", r.Decision.Rationale)
		}
		if r.Decision.DecidedAt != "" {
			fmt.Printf("  At:         %s\n", r.Decision.DecidedAt)
		}
		if r.Decision.DecidedBy != "" {
			fmt.Printf("  By:         %s\n", r.Decision.DecidedBy)
		}
	}
}
