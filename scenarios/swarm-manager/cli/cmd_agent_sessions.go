package main

import (
	"flag"
	"fmt"
	"net/url"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

type cliAgentSession struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Kind      string `json:"kind"`
	Status    string `json:"status"`
	RunID     string `json:"run_id,omitempty"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
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
