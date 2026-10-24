package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api/apiconnect"
)

// Proposals are the operator's decision surface: an agent workflow proposes a
// list of mutations, and the operator accepts all of it, a subset of it, or
// none. Until this group existed the whole surface was UI-only, so an agent or
// a headless operator could start a goal workflow but never decide its result.

type cliProposalDecision struct {
	Kind                string   `json:"kind"`
	AcceptedMutationIDs []string `json:"accepted_mutation_ids,omitempty"`
	RejectedMutationIDs []string `json:"rejected_mutation_ids,omitempty"`
	Note                string   `json:"note,omitempty"`
	DecidedAt           string   `json:"decided_at,omitempty"`
}

type cliProposalTarget struct {
	Type string `json:"type"`
	Ref  string `json:"ref"`
	Name string `json:"name"`
}

type cliProposal struct {
	ID               string                `json:"id"`
	Kind             string                `json:"kind"`
	Status           string                `json:"status"`
	Summary          string                `json:"summary"`
	PayloadJSON      string                `json:"payload_json"`
	CreatedAt        string                `json:"created_at"`
	UpdatedAt        string                `json:"updated_at"`
	Target           *cliProposalTarget    `json:"target,omitempty"`
	NeedsRevision    bool                  `json:"needs_revision,omitempty"`
	ParseWarnings    []string              `json:"parse_warnings,omitempty"`
	ValidationErrors []string              `json:"validation_errors,omitempty"`
	Decisions        []cliProposalDecision `json:"decisions,omitempty"`
}

type cliProposalSession struct {
	ID             string             `json:"id"`
	Title          string             `json:"title"`
	Kind           string             `json:"kind"`
	Status         string             `json:"status"`
	UpdatedAt      string             `json:"updated_at"`
	Proposals      []cliProposal      `json:"proposals,omitempty"`
	ProposalTarget *cliProposalTarget `json:"proposal_target,omitempty"`
}

type cliProposalSessionsResponse struct {
	Sessions []cliProposalSession `json:"sessions"`
}

// mutationSummary decodes the payload envelope for human rendering.
//
// The CLI is a separate module from the API, so it cannot import
// proposals.Proposal; this mirrors the payload fields the renderer needs.
// It previously carried only id and op, which made `proposals get` print
// "m1  update_item" and nothing else — thinner than the UI it backstops.
type mutationSummary struct {
	BaseVersion string        `json:"base_version"`
	Rationale   string        `json:"rationale"`
	Mutations   []cliMutation `json:"mutations"`
}

type cliMutation struct {
	ID         string   `json:"id"`
	Op         string   `json:"op"`
	Rationale  string   `json:"rationale"`
	Target     string   `json:"target"`
	Status     string   `json:"status"`
	Priority   *int     `json:"priority"`
	From       string   `json:"from"`
	To         string   `json:"to"`
	Milestone  string   `json:"milestone"`
	Sources    []string `json:"sources"`
	Items      []string `json:"items"`
	Targets    []string `json:"targets"`
	ResetScope []string `json:"reset_scope"`

	Item  *cliItemSpec   `json:"item"`
	Into  []cliItemSpec  `json:"into"`
	Patch map[string]any `json:"patch"`

	Goal          *cliNamed `json:"goal"`
	GoalMilestone *cliNamed `json:"goal_milestone"`
	MilestoneName string    `json:"milestone_name"`
}

type cliItemSpec struct {
	Kind            string   `json:"kind"`
	Name            string   `json:"name"`
	Title           string   `json:"title"`
	Description     string   `json:"description"`
	Priority        int      `json:"priority"`
	Effort          string   `json:"effort"`
	Tags            []string `json:"tags"`
	Milestone       string   `json:"milestone"`
	AcceptanceAllow []string `json:"acceptance_allow"`
}

type cliNamed struct {
	Name  string `json:"name"`
	Title string `json:"title"`
}

func (s cliItemSpec) ref() string {
	if s.Kind == "" || s.Name == "" {
		return ""
	}
	return s.Kind + "/" + s.Name
}

// subject resolves the reference a mutation acts on. Creation ops carry no
// target by design, so they resolve to the ref they will bring into existence
// rather than printing an empty column.
func (m cliMutation) subject() string {
	switch m.Op {
	case "add_item", "merge_items":
		if m.Item != nil {
			if ref := m.Item.ref(); ref != "" {
				return ref
			}
		}
	case "create_goal":
		if m.Goal != nil {
			return m.Goal.Name
		}
	case "create_milestone", "update_milestone", "archive_milestone":
		if m.GoalMilestone != nil && m.GoalMilestone.Name != "" {
			return m.GoalMilestone.Name
		}
		return m.MilestoneName
	case "assign_milestone_items", "unassign_milestone_items":
		if m.MilestoneName != "" {
			return m.MilestoneName
		}
	case "add_edge", "remove_edge":
		if m.From != "" {
			return m.From
		}
	}
	return m.Target
}

func (a *App) listProposalSessions(targetType, targetRef string) ([]cliProposalSession, error) {
	query := url.Values{}
	if trimmed := strings.TrimSpace(targetType); trimmed != "" {
		query.Set("target_type", trimmed)
	}
	if trimmed := strings.TrimSpace(targetRef); trimmed != "" {
		query.Set("target_ref", trimmed)
	}
	path := "/proposal-sessions"
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	body, err := a.requestMultipart("GET", path, nil, "")
	if err != nil {
		return nil, err
	}
	response, err := decodeResponse[cliProposalSessionsResponse](body)
	if err != nil {
		return nil, err
	}
	return response.Sessions, nil
}

func (a *App) cmdProposalsList(args []string) error {
	fs := flag.NewFlagSet("proposals list", flag.ContinueOnError)
	targetType := fs.String("target-type", "", "Filter by target type (goal, backlog)")
	targetRef := fs.String("target-ref", "", "Filter by target ref")
	status := fs.String("status", "", "Filter by proposal status (ready, applied, needs_revision, ...)")
	pendingOnly := fs.Bool("pending", false, "Only proposals still awaiting a decision")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	sessions, err := a.listProposalSessions(*targetType, *targetRef)
	if err != nil {
		return err
	}

	type row struct {
		SessionID string      `json:"session_id"`
		Target    string      `json:"target"`
		Proposal  cliProposal `json:"proposal"`
	}
	rows := make([]row, 0)
	for _, session := range sessions {
		target := ""
		if session.ProposalTarget != nil {
			target = session.ProposalTarget.Type + "/" + session.ProposalTarget.Ref
		}
		for _, proposal := range session.Proposals {
			if trimmed := strings.TrimSpace(*status); trimmed != "" && proposal.Status != trimmed {
				continue
			}
			// "Pending" is anything an operator still has to act on: awaiting a
			// decision, or bounced back for revision.
			if *pendingOnly && proposal.Status != "ready" && proposal.Status != "needs_revision" {
				continue
			}
			rows = append(rows, row{SessionID: session.ID, Target: target, Proposal: proposal})
		}
	}

	if *jsonOut {
		encoded, err := json.MarshalIndent(map[string]any{"proposals": rows}, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(encoded))
		return nil
	}
	if len(rows) == 0 {
		fmt.Println("No proposals match.")
		return nil
	}
	fmt.Printf("Proposals: %d\n", len(rows))
	for _, entry := range rows {
		fmt.Printf("  %s  [%s]  %s\n", entry.Proposal.ID, entry.Proposal.Status, entry.Target)
		fmt.Printf("      session %s — %s\n", entry.SessionID, entry.Proposal.Summary)
		for _, problem := range entry.Proposal.ValidationErrors {
			fmt.Printf("      validation: %s\n", problem)
		}
	}
	return nil
}

// findProposal locates one proposal without requiring the caller to know which
// session holds it. Session is optional precisely because the proposal id is
// what an operator reads off a listing.
func (a *App) findProposal(sessionID, proposalID string) (cliProposalSession, cliProposal, error) {
	sessions, err := a.listProposalSessions("", "")
	if err != nil {
		return cliProposalSession{}, cliProposal{}, err
	}
	for _, session := range sessions {
		if sessionID != "" && session.ID != sessionID {
			continue
		}
		for _, proposal := range session.Proposals {
			if proposal.ID == proposalID {
				return session, proposal, nil
			}
		}
	}
	return cliProposalSession{}, cliProposal{}, fmt.Errorf("proposal %q not found", proposalID)
}

func (a *App) cmdProposalsGet(args []string) error {
	fs := flag.NewFlagSet("proposals get", flag.ContinueOnError)
	sessionID := fs.String("session", "", "Session id (optional; narrows the search)")
	proposalID := fs.String("id", "", "Proposal id")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("id", *proposalID); err != nil {
		return err
	}
	session, proposal, err := a.findProposal(strings.TrimSpace(*sessionID), strings.TrimSpace(*proposalID))
	if err != nil {
		return err
	}
	if *jsonOut {
		encoded, err := json.MarshalIndent(map[string]any{"session_id": session.ID, "proposal": proposal}, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(encoded))
		return nil
	}
	fmt.Printf("Proposal %s [%s] in session %s\n", proposal.ID, proposal.Status, session.ID)
	if session.ProposalTarget != nil {
		fmt.Printf("  Target: %s/%s\n", session.ProposalTarget.Type, session.ProposalTarget.Ref)
	}
	fmt.Printf("  Summary: %s\n", proposal.Summary)
	for _, problem := range proposal.ValidationErrors {
		fmt.Printf("  Validation error: %s\n", problem)
	}
	for _, warning := range proposal.ParseWarnings {
		fmt.Printf("  Parse warning: %s\n", warning)
	}
	var payload mutationSummary
	if err := json.Unmarshal([]byte(proposal.PayloadJSON), &payload); err == nil {
		if payload.Rationale != "" {
			fmt.Printf("  Rationale: %s\n", payload.Rationale)
		}
		if len(payload.Mutations) > 0 {
			fmt.Printf("  Mutations (base version %s):\n", payload.BaseVersion)
			for _, mutation := range payload.Mutations {
				renderMutation(mutation)
			}
			fmt.Println("  Accept a subset with: proposals decide --id " + proposal.ID + " --accept <id>,<id>")
		}
	}
	for _, decision := range proposal.Decisions {
		fmt.Printf("  Decision %s at %s — accepted %d, rejected %d\n", decision.Kind, decision.DecidedAt, len(decision.AcceptedMutationIDs), len(decision.RejectedMutationIDs))
	}
	return nil
}

// patchFieldOrder keeps `proposals get` output stable across runs; ranging a
// map directly would reorder the fields on every invocation and make the
// command useless for diffing two proposals.
var patchFieldOrder = []string{
	"title", "description", "note", "priority", "effort",
	"tags", "depends_on", "acceptance_allow", "acceptance_deny",
}

// renderMutation prints one mutation with the payload that decides it. The op
// name alone is not a reviewable change.
func renderMutation(mutation cliMutation) {
	subject := mutation.subject()
	if subject == "" {
		subject = "(no target)"
	}
	fmt.Printf("    %s  %s  %s\n", mutation.ID, mutation.Op, subject)
	if mutation.Rationale != "" {
		fmt.Printf("        why: %s\n", mutation.Rationale)
	}

	if mutation.Item != nil {
		renderItemSpec("        ", *mutation.Item)
	}
	for _, child := range mutation.Into {
		renderItemSpec("        + ", child)
	}
	if len(mutation.Patch) > 0 {
		fmt.Printf("        patch: %s\n", strings.Join(patchedFieldNames(mutation.Patch), ", "))
		for _, field := range patchFieldOrder {
			value, ok := mutation.Patch[field]
			if !ok {
				continue
			}
			fmt.Printf("          %s = %s\n", field, truncate(formatPatchValue(value), 400))
		}
	}
	if mutation.Status != "" {
		fmt.Printf("        status -> %s\n", mutation.Status)
	}
	if mutation.Priority != nil {
		fmt.Printf("        priority -> %d\n", *mutation.Priority)
	}
	if mutation.Op == "move_milestone" {
		destination := mutation.Milestone
		if destination == "" {
			destination = "(detach from milestone)"
		}
		fmt.Printf("        milestone -> %s\n", destination)
	}
	if mutation.From != "" || mutation.To != "" {
		fmt.Printf("        edge: %s depends on %s\n", orPlaceholder(mutation.From), orPlaceholder(mutation.To))
	}
	for _, source := range mutation.Sources {
		fmt.Printf("        source (archived): %s\n", source)
	}
	if len(mutation.Items) > 0 {
		fmt.Printf("        items: %s\n", strings.Join(mutation.Items, ", "))
	}
	if len(mutation.Targets) > 0 {
		fmt.Printf("        targets: %s\n", strings.Join(mutation.Targets, ", "))
	}
	if len(mutation.ResetScope) > 0 {
		fmt.Printf("        removes: %s\n", strings.Join(mutation.ResetScope, ", "))
	}
	if mutation.Goal != nil {
		fmt.Printf("        goal: %s (%s)\n", mutation.Goal.Title, mutation.Goal.Name)
	}
	if mutation.GoalMilestone != nil {
		fmt.Printf("        milestone: %s (%s)\n", mutation.GoalMilestone.Title, mutation.GoalMilestone.Name)
	}
}

func renderItemSpec(indent string, spec cliItemSpec) {
	fmt.Printf("%stitle: %s\n", indent, spec.Title)
	if ref := spec.ref(); ref != "" {
		fmt.Printf("%sref: %s\n", indent, ref)
	}
	meta := make([]string, 0, 4)
	if spec.Priority != 0 {
		meta = append(meta, fmt.Sprintf("priority %d", spec.Priority))
	}
	if spec.Effort != "" {
		meta = append(meta, "effort "+spec.Effort)
	}
	if spec.Milestone != "" {
		meta = append(meta, "milestone "+spec.Milestone)
	}
	if len(spec.Tags) > 0 {
		meta = append(meta, "tags "+strings.Join(spec.Tags, "/"))
	}
	if len(meta) > 0 {
		fmt.Printf("%s%s\n", indent, strings.Join(meta, " · "))
	}
	if spec.Description != "" {
		fmt.Printf("%sdescription: %s\n", indent, truncate(spec.Description, 400))
	}
	for index, criterion := range spec.AcceptanceAllow {
		fmt.Printf("%sacceptance %d: %s\n", indent, index+1, truncate(criterion, 240))
	}
}

func patchedFieldNames(patch map[string]any) []string {
	names := make([]string, 0, len(patch))
	for _, field := range patchFieldOrder {
		if _, ok := patch[field]; ok {
			names = append(names, field)
		}
	}
	// Any field the server gained that this list predates still gets named.
	for field := range patch {
		if !slices.Contains(patchFieldOrder, field) {
			names = append(names, field)
		}
	}
	return names
}

func formatPatchValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return "(cleared)"
	case string:
		if typed == "" {
			return "(cleared)"
		}
		return typed
	case []any:
		if len(typed) == 0 {
			return "(cleared)"
		}
		parts := make([]string, 0, len(typed))
		for _, entry := range typed {
			parts = append(parts, fmt.Sprint(entry))
		}
		return strings.Join(parts, ", ")
	case float64:
		// encoding/json decodes every number into float64; integers are the
		// only numeric values the patch schema carries.
		return strconv.FormatFloat(typed, 'f', -1, 64)
	default:
		return fmt.Sprint(typed)
	}
}

func orPlaceholder(value string) string {
	if value == "" {
		return "(unset)"
	}
	return value
}

// cmdProposalsDecide applies a mutation-list proposal. Omitting --accept
// accepts every mutation; naming a subset re-validates that subset on its own,
// so a partial acceptance can never leave the target in an incoherent state.
func (a *App) cmdProposalsDecide(args []string) error {
	fs := flag.NewFlagSet("proposals decide", flag.ContinueOnError)
	sessionID := fs.String("session", "", "Session id (optional; narrows the search)")
	proposalID := fs.String("id", "", "Proposal id")
	accept := fs.String("accept", "", "Comma-separated mutation ids to accept (default: all)")
	note := fs.String("note", "", "Operator note recorded with the decision")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("id", *proposalID); err != nil {
		return err
	}
	session, proposal, err := a.findProposal(strings.TrimSpace(*sessionID), strings.TrimSpace(*proposalID))
	if err != nil {
		return err
	}
	accepted := splitCSV(strings.TrimSpace(*accept))
	h, base := cliapp.NewConnectHTTPClient(a.core)
	response, err := apiconnect.NewBacklogServiceClient(h, base).DecideAttempt(context.Background(), connect.NewRequest(&apipb.DecideAttemptRequest{
		SubjectKind:         "agent-session-proposal",
		SubjectRef:          session.ID + "/" + proposal.ID,
		RoundNum:            1,
		Decision:            "accept",
		Actor:               "operator-cli",
		Rationale:           strings.TrimSpace(*note),
		AcceptedProposalIds: accepted,
	}))
	if err != nil {
		return err
	}
	if *jsonOut {
		body, err := json.Marshal(response.Msg)
		if err != nil {
			return fmt.Errorf("encode decision response: %w", err)
		}
		fmt.Println(string(body))
		return nil
	}
	fmt.Printf("Proposal %s decided: %s (%s)\n", proposal.ID, response.Msg.Decision, response.Msg.Status)
	return nil
}

func (a *App) cmdProposalsAcceptKeep(args []string) error {
	return a.proposalNoteAction(args, "proposals accept-keep", "accept-keep", "accepted as no-change")
}

func (a *App) cmdProposalsRevise(args []string) error {
	return a.proposalNoteAction(args, "proposals revise", "revise", "sent back for revision")
}

// proposalNoteAction runs the two proposal decisions whose only input is a
// note: accepting a keep-as-is recommendation, and bouncing one for revision.
func (a *App) proposalNoteAction(args []string, command, action, verb string) error {
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	sessionID := fs.String("session", "", "Session id (optional; narrows the search)")
	proposalID := fs.String("id", "", "Proposal id")
	note := fs.String("note", "", "Operator note recorded with the decision")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("id", *proposalID); err != nil {
		return err
	}
	session, proposal, err := a.findProposal(strings.TrimSpace(*sessionID), strings.TrimSpace(*proposalID))
	if err != nil {
		return err
	}
	payload, err := json.Marshal(map[string]any{"note": strings.TrimSpace(*note)})
	if err != nil {
		return err
	}
	body, err := a.requestMultipart("POST", "/agent-sessions/"+session.ID+"/proposals/"+proposal.ID+"/"+action, payload, "application/json")
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}
	return printProposalOutcome(body, proposal.ID, verb)
}

// printProposalOutcome reports the proposal's post-decision state. The decide
// endpoint returns the whole session, and a rejected apply comes back as
// needs_revision with the reason rather than as an HTTP error — so reading the
// status back is the only way to tell success from a validation bounce.
func printProposalOutcome(body []byte, proposalID, verb string) error {
	session, err := decodeResponse[cliProposalSession](body)
	if err != nil {
		return err
	}
	for _, proposal := range session.Proposals {
		if proposal.ID != proposalID {
			continue
		}
		fmt.Printf("Proposal %s %s — status %s\n", proposal.ID, verb, proposal.Status)
		for _, problem := range proposal.ValidationErrors {
			fmt.Printf("  validation: %s\n", problem)
		}
		return nil
	}
	fmt.Printf("Proposal %s %s\n", proposalID, verb)
	return nil
}
