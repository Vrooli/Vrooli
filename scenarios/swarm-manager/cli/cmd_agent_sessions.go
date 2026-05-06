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
	jsonOut := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if err := requireFlag("id", *idFlag); err != nil {
		return fmt.Errorf("usage: sessions get --id ID [--json]\n\n%s", err)
	}
	id := strings.TrimSpace(*idFlag)

	body, err := a.core.Get("/agent-sessions/"+id, nil)
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
	fmt.Println("  Created backlog items, initiatives, captures, files, and agent activity records were preserved.")
	return nil
}
