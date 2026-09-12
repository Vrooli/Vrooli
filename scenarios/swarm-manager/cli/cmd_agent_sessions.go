package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

type cliAgentSession struct {
	ID                string                      `json:"id"`
	Title             string                      `json:"title"`
	Kind              string                      `json:"kind"`
	Status            string                      `json:"status"`
	RunID             string                      `json:"run_id,omitempty"`
	CreatedAt         string                      `json:"created_at"`
	UpdatedAt         string                      `json:"updated_at"`
	StagedContextRefs []cliAgentSessionContextRef `json:"staged_context_refs,omitempty"`
}

type cliAgentSessionContextRef struct {
	Type string `json:"type"`
	Ref  string `json:"ref"`
}

type cliAgentSessionResponse struct {
	Session cliAgentSession `json:"session"`
}

type cliAgentSessionContextItem struct {
	Type         string `json:"type"`
	Ref          string `json:"ref"`
	Title        string `json:"title"`
	Summary      string `json:"summary"`
	NodeID       string `json:"node_id,omitempty"`
	MetadataJSON string `json:"metadata_json,omitempty"`
	SelectedAt   string `json:"selected_at"`
}

type cliAgentSessionStartupBriefResponse struct {
	Brief cliAgentSessionContextItem `json:"brief"`
}

type cliListAgentSessionsResponse struct {
	Sessions []cliAgentSession `json:"sessions"`
}

type cliDeleteAgentSessionResponse struct {
	SessionID string `json:"session_id"`
}

type cliReapAgentSessionsResponse struct {
	Sessions []cliAgentSession `json:"sessions"`
	Reaped   int               `json:"reaped"`
}

type cliAgentSessionRunEvent struct {
	Sequence  int64  `json:"sequence"`
	CreatedAt string `json:"created_at"`
	EventType string `json:"event_type"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	Status    string `json:"status"`
	Summary   string `json:"summary"`
}

type cliListAgentSessionEventsResponse struct {
	Events            []cliAgentSessionRunEvent `json:"events"`
	HasMore           bool                      `json:"has_more"`
	NextAfterSequence int64                     `json:"next_after_sequence"`
}

func (a *App) cmdSessionsList(args []string) error {
	fs := flag.NewFlagSet("sessions list", flag.ContinueOnError)
	kindFlag := fs.String("kind", "", "Session kind filter")
	statusFlag := fs.String("status", "", "Session status filter")
	activeOnly := fs.Bool("active-only", false, "Only list active sessions")
	limitFlag := fs.Int("limit", 0, "Maximum sessions to return")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	query := url.Values{}
	if kind := strings.TrimSpace(*kindFlag); kind != "" {
		query.Set("kind", kind)
	}
	if status := strings.TrimSpace(*statusFlag); status != "" {
		query.Set("status", status)
	}
	if *activeOnly {
		query.Set("active_only", "true")
	}
	if *limitFlag > 0 {
		query.Set("limit", fmt.Sprintf("%d", *limitFlag))
	}

	body, err := a.core.Get("/agent-sessions", query)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[cliListAgentSessionsResponse](body)
	if err != nil {
		return err
	}
	if len(response.Sessions) == 0 {
		printSection("Summary")
		fmt.Println("  No agent sessions found.")
		return nil
	}

	printSection("Summary")
	fmt.Printf("  Found %d agent session(s)\n", len(response.Sessions))
	printSection("Results")
	for _, session := range response.Sessions {
		fmt.Printf("  [%s] %s  %s  %s\n", session.Status, session.ID, session.Kind, session.Title)
	}
	first := response.Sessions[0]
	printCommandListSection("Retrieval Hints", []string{
		cliCommand("sessions", "get", "--id", first.ID),
		cliCommand("sessions", "delete", "--id", first.ID, "--yes"),
	})
	return nil
}

func (a *App) cmdSessionsGet(args []string) error {
	fs := flag.NewFlagSet("sessions get", flag.ContinueOnError)
	idFlag := fs.String("id", "", "Session ID")
	refreshFlag := fs.Bool("refresh", false, "Refresh backing agent run state before printing")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("id", *idFlag); err != nil {
		return fmt.Errorf("usage: sessions get --id ID [--json]\n\n%s", err)
	}
	id := strings.TrimSpace(*idFlag)

	var body []byte
	var err error
	if *refreshFlag {
		body, err = a.core.Request("POST", "/agent-sessions/"+id+"/refresh", nil, nil)
	} else {
		body, err = a.core.Get("/agent-sessions/"+id, nil)
	}
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[cliAgentSessionResponse](body)
	if err != nil {
		return err
	}
	session := response.Session
	printSection("Summary")
	fmt.Printf("  %s (%s)\n", session.Title, session.Status)
	printSection("Details")
	fmt.Printf("  ID: %s\n", session.ID)
	fmt.Printf("  Kind: %s\n", session.Kind)
	if session.RunID != "" {
		fmt.Printf("  Run ID: %s\n", session.RunID)
	}
	fmt.Printf("  Created: %s\n", session.CreatedAt)
	fmt.Printf("  Updated: %s\n", session.UpdatedAt)
	printCommandListSection("Next Steps", []string{
		cliCommand("sessions", "delete", "--id", session.ID, "--yes"),
	})
	return nil
}

func (a *App) cmdSessionsCreate(args []string) error {
	fs := flag.NewFlagSet("sessions create", flag.ContinueOnError)
	kind := fs.String("kind", "", "Session kind")
	title := fs.String("title", "", "Human-readable title")
	starterJob := fs.String("starter-job", "", "Server-declared starter job")
	target := fs.String("target", "", "Optional proposal target as TYPE/REF")
	targetName := fs.String("target-name", "", "Human-readable proposal target name")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("kind", *kind); err != nil {
		return fmt.Errorf("usage: sessions create --kind KIND [--title TEXT] [--starter-job ID] [--target TYPE/REF] [--target-name NAME] [--json]\n\n%s", err)
	}
	trimmedKind := strings.TrimSpace(*kind)
	trimmedTitle := strings.TrimSpace(*title)
	if trimmedTitle == "" {
		trimmedTitle = strings.ReplaceAll(trimmedKind, "_", " ") + " session"
	}
	payload := map[string]any{"kind": trimmedKind, "title": trimmedTitle}
	if value := strings.TrimSpace(*starterJob); value != "" {
		payload["starter_job_id"] = value
	}
	path := "/agent-sessions"
	if value := strings.TrimSpace(*target); value != "" {
		proposalTarget, targetErr := proposalTargetPayload(value, *targetName)
		if targetErr != nil {
			return targetErr
		}
		payload["target"] = proposalTarget
		path = "/proposal-sessions"
	}
	body, err := a.core.Request("POST", path, nil, payload)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}
	if path == "/proposal-sessions" {
		response, decodeErr := decodeResponse[cliProposalSession](body)
		if decodeErr != nil {
			return decodeErr
		}
		fmt.Printf("Created targeted draft session %s (%s)\n", response.ID, response.Kind)
		return nil
	}
	response, err := decodeResponse[cliAgentSessionResponse](body)
	if err != nil {
		return err
	}
	fmt.Printf("Created draft session %s (%s)\n", response.Session.ID, response.Session.Kind)
	return nil
}

func (a *App) cmdSessionsCreateBatch(args []string) error {
	fs := flag.NewFlagSet("sessions create-batch", flag.ContinueOnError)
	file := fs.String("file", "", "JSON file containing {\"sessions\":[...]}")
	actor := fs.String("actor", "", "Named actor responsible for this batch")
	overrideReason := fs.String("override-reason", "", "Required reason when the batch exceeds the server cap")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("file", *file); err != nil {
		return err
	}
	if err := requireFlag("actor", *actor); err != nil {
		return err
	}
	data, err := os.ReadFile(*file)
	if err != nil {
		return fmt.Errorf("read batch file: %w", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("parse batch file: %w", err)
	}
	payload["actor"] = strings.TrimSpace(*actor)
	if value := strings.TrimSpace(*overrideReason); value != "" {
		payload["override_reason"] = value
	}
	body, err := a.core.Request("POST", "/agent-sessions/batch", nil, payload)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}
	var response struct {
		Created int `json:"created"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return err
	}
	fmt.Printf("Created %d agent session(s)\n", response.Created)
	return nil
}

func (a *App) cmdSessionsAttach(args []string) error {
	fs := flag.NewFlagSet("sessions attach", flag.ContinueOnError)
	id := fs.String("id", "", "Session ID")
	var entities stringSlice
	fs.Var(&entities, "entity", "Typed entity TYPE/REF (repeatable)")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("id", *id); err != nil {
		return err
	}
	if len(entities) == 0 {
		return fmt.Errorf("at least one --entity TYPE/REF is required")
	}
	refs, err := parseSessionEntities(entities)
	if err != nil {
		return err
	}
	body, err := a.core.Request("POST", "/agent-sessions/"+strings.TrimSpace(*id)+"/context", nil, map[string]any{"context_refs": refs})
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}
	response, err := decodeResponse[cliAgentSessionResponse](body)
	if err != nil {
		return err
	}
	fmt.Printf("Session %s now holds %d staged context item(s)\n", response.Session.ID, len(response.Session.StagedContextRefs))
	return nil
}

func (a *App) cmdSessionsStart(args []string) error {
	return a.sessionMessageAction(args, "sessions start", "start", false)
}

func (a *App) cmdSessionsContinue(args []string) error {
	return a.sessionMessageAction(args, "sessions continue", "continue", true)
}

func (a *App) cmdSessionsComplete(args []string) error {
	fs := flag.NewFlagSet("sessions complete", flag.ContinueOnError)
	id := fs.String("id", "", "Session ID")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("id", *id); err != nil {
		return err
	}
	body, err := a.core.Request("POST", "/agent-sessions/"+strings.TrimSpace(*id)+"/complete", nil, nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}
	response, err := decodeResponse[cliAgentSessionResponse](body)
	if err != nil {
		return err
	}
	fmt.Printf("Session %s status: %s\n", response.Session.ID, response.Session.Status)
	return nil
}

func (a *App) cmdSessionsReap(args []string) error {
	fs := flag.NewFlagSet("sessions reap", flag.ContinueOnError)
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	body, err := a.core.Request("POST", "/agent-sessions/reap", nil, nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}
	response, err := decodeResponse[cliReapAgentSessionsResponse](body)
	if err != nil {
		return err
	}
	fmt.Printf("Reaped %d stale running session(s)\n", response.Reaped)
	return nil
}

func (a *App) cmdSessionsDisposition(args []string) error {
	fs := flag.NewFlagSet("sessions disposition", flag.ContinueOnError)
	id := fs.String("id", "", "Session ID")
	disposition := fs.String("disposition", "", "dropped or retained")
	reason := fs.String("reason", "", "Why this disposition was chosen")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	for name, value := range map[string]string{"id": *id, "disposition": *disposition, "reason": *reason} {
		if err := requireFlag(name, value); err != nil {
			return err
		}
	}
	body, err := a.core.Request("POST", "/agent-sessions/"+strings.TrimSpace(*id)+"/disposition", nil, map[string]string{"disposition": strings.TrimSpace(*disposition), "reason": strings.TrimSpace(*reason)})
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}
	var response struct {
		Session cliAgentSession `json:"session"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return err
	}
	fmt.Printf("Session %s disposition: %s (%s)\n", response.Session.ID, response.Session.Status, strings.TrimSpace(*disposition))
	return nil
}

func (a *App) sessionMessageAction(args []string, command, action string, messageRequired bool) error {
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	id := fs.String("id", "", "Session ID")
	message := fs.String("message", "", "Operator message")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("id", *id); err != nil {
		return err
	}
	if messageRequired {
		if err := requireFlag("message", *message); err != nil {
			return err
		}
	}
	body, err := a.core.Request("POST", "/agent-sessions/"+strings.TrimSpace(*id)+"/"+action, nil, map[string]any{"message": *message})
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}
	response, err := decodeResponse[cliAgentSessionResponse](body)
	if err != nil {
		return err
	}
	fmt.Printf("Session %s status: %s\n", response.Session.ID, response.Session.Status)
	return nil
}

func (a *App) cmdSessionsEvents(args []string) error {
	fs := flag.NewFlagSet("sessions events", flag.ContinueOnError)
	id := fs.String("id", "", "Session ID")
	after := fs.Int64("after-sequence", -1, "Return events after this sequence")
	limit := fs.Int("limit", 0, "Maximum events")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("id", *id); err != nil {
		return err
	}
	query := url.Values{}
	if *after >= 0 {
		query.Set("after_sequence", fmt.Sprintf("%d", *after))
	}
	if *limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", *limit))
	}
	body, err := a.core.Get("/agent-sessions/"+strings.TrimSpace(*id)+"/events", query)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}
	response, err := decodeResponse[cliListAgentSessionEventsResponse](body)
	if err != nil {
		return err
	}
	for _, event := range response.Events {
		text := strings.TrimSpace(event.Summary)
		if text == "" {
			text = strings.TrimSpace(event.Content)
		}
		fmt.Printf("  %d  %s  %s\n", event.Sequence, event.EventType, text)
	}
	if len(response.Events) == 0 {
		fmt.Println("No session events found.")
	}
	return nil
}

func (a *App) cmdSessionsProposalApply(args []string) error {
	fs := flag.NewFlagSet("sessions proposal-apply", flag.ContinueOnError)
	id := fs.String("id", "", "Session ID")
	proposal := fs.String("proposal", "", "Proposal ID")
	var accepted stringSlice
	fs.Var(&accepted, "accept", "Accepted mutation ID (repeatable)")
	note := fs.String("note", "", "Decision note")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("id", *id); err != nil {
		return err
	}
	if err := requireFlag("proposal", *proposal); err != nil {
		return err
	}
	if len(accepted) == 0 {
		return fmt.Errorf("at least one --accept MUTATION_ID is required")
	}
	return a.sessionProposalDecision(strings.TrimSpace(*id), strings.TrimSpace(*proposal), "apply", map[string]any{"accepted_mutation_ids": []string(accepted), "note": strings.TrimSpace(*note)}, *jsonOut)
}

func (a *App) cmdSessionsProposalRevise(args []string) error {
	return a.sessionProposalNoteAction(args, "sessions proposal-revise", "revise")
}

func (a *App) cmdSessionsProposalWait(args []string) error {
	return a.sessionProposalNoteAction(args, "sessions proposal-wait", "wait")
}

func (a *App) cmdSessionsProposalAcceptKeep(args []string) error {
	return a.sessionProposalNoteAction(args, "sessions proposal-accept-keep", "accept-keep")
}

func (a *App) sessionProposalNoteAction(args []string, command, action string) error {
	fs := flag.NewFlagSet(command, flag.ContinueOnError)
	id := fs.String("id", "", "Session ID")
	proposal := fs.String("proposal", "", "Proposal ID")
	note := fs.String("note", "", "Decision note")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("id", *id); err != nil {
		return err
	}
	if err := requireFlag("proposal", *proposal); err != nil {
		return err
	}
	return a.sessionProposalDecision(strings.TrimSpace(*id), strings.TrimSpace(*proposal), action, map[string]any{"note": strings.TrimSpace(*note)}, *jsonOut)
}

func (a *App) sessionProposalDecision(sessionID, proposalID, action string, payload map[string]any, jsonOut bool) error {
	body, err := a.core.Request("POST", "/agent-sessions/"+sessionID+"/proposals/"+proposalID+"/"+action, nil, payload)
	if err != nil {
		return err
	}
	if printJSONIfRequested(jsonOut, body) {
		return nil
	}
	fmt.Printf("Proposal %s decision submitted (%s)\n", proposalID, action)
	return nil
}

func (a *App) cmdSessionsStartupBrief(args []string) error {
	fs := flag.NewFlagSet("sessions startup-brief", flag.ContinueOnError)
	idFlag := fs.String("id", "", "Session ID")
	refreshFlag := fs.Bool("refresh", false, "Regenerate the brief")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("id", *idFlag); err != nil {
		return fmt.Errorf("usage: sessions startup-brief --id ID [--refresh] [--json]\n\n%s", err)
	}
	id := strings.TrimSpace(*idFlag)
	method := "GET"
	if *refreshFlag {
		method = "POST"
	}
	body, err := a.core.Request(method, "/agent-sessions/"+id+"/startup-brief", nil, nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}
	response, err := decodeResponse[cliAgentSessionStartupBriefResponse](body)
	if err != nil {
		return err
	}
	brief := response.Brief
	printSection("Startup Brief")
	fmt.Printf("  %s\n", brief.Title)
	fmt.Printf("  Ref: %s\n", brief.Ref)
	if brief.SelectedAt != "" {
		fmt.Printf("  Generated: %s\n", brief.SelectedAt)
	}
	printSection("Summary")
	for _, line := range strings.Split(strings.TrimSpace(brief.Summary), "\n") {
		if strings.TrimSpace(line) != "" {
			fmt.Printf("  %s\n", line)
		}
	}
	printCommandListSection("Drill Down", []string{
		cliCommand("sessions", "startup-brief", "--id", id, "--refresh", "--json"),
	})
	return nil
}

func (a *App) cmdSessionsDelete(args []string) error {
	fs := flag.NewFlagSet("sessions delete", flag.ContinueOnError)
	idFlag := fs.String("id", "", "Session ID")
	yesFlag := fs.Bool("yes", false, "Confirm deletion")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("id", *idFlag); err != nil {
		return fmt.Errorf("usage: sessions delete --id ID --yes [--json]\n\n%s", err)
	}
	if !*yesFlag {
		return fmt.Errorf("refusing to delete session without --yes; created outputs remain, but the session conversation, details, proposal drafts, and artifact links will be removed")
	}
	id := strings.TrimSpace(*idFlag)

	body, err := a.core.Request("DELETE", "/agent-sessions/"+id, nil, nil)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[cliDeleteAgentSessionResponse](body)
	if err != nil {
		return err
	}
	printSection("Deleted")
	fmt.Printf("  Session %s deleted.\n", response.SessionID)
	fmt.Println("  Created backlog items, milestones, captures, files, and agent activity records were preserved.")
	return nil
}

type cliPromptPreviewResponse struct {
	Prompt  string `json:"prompt"`
	Initial bool   `json:"initial"`
}

// cmdSessionsPromptPreview prints the prompt a message would produce without
// sending it. Assembly is server-owned, so this is the authoritative view of
// what an agent actually receives.
func (a *App) cmdSessionsPromptPreview(args []string) error {
	fs := flag.NewFlagSet("sessions prompt-preview", flag.ContinueOnError)
	idFlag := fs.String("id", "", "Session ID")
	messageFlag := fs.String("message", "", "Draft message to preview")
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("id", *idFlag); err != nil {
		return fmt.Errorf("usage: sessions prompt-preview --id ID [--message TEXT] [--json]\n\n%s", err)
	}
	id := strings.TrimSpace(*idFlag)

	payload := map[string]any{"message": *messageFlag}
	body, err := a.core.Request("POST", "/agent-sessions/"+id+"/prompt-preview", nil, payload)
	if err != nil {
		return err
	}
	if printJSONIfRequested(*jsonOut, body) {
		return nil
	}

	response, err := decodeResponse[cliPromptPreviewResponse](body)
	if err != nil {
		return err
	}
	printSection("Summary")
	builder := "continuation"
	if response.Initial {
		builder = "initial"
	}
	fmt.Printf("  %s prompt, %d characters\n", builder, len(response.Prompt))
	printSection("Prompt")
	fmt.Println(response.Prompt)
	return nil
}
