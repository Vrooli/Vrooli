package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

// runIdentity exposes the verified identity of the current agent-manager run.
// It is intentionally read-only: coordinators need the server-verified parent
// run ID to create linked children and supervision watches, but they must not
// be given a way to mint or alter authority.
func (a *App) runIdentity(args []string) error {
	jsonOutput := false
	for _, arg := range args {
		switch arg {
		case "--json":
			jsonOutput = true
		case "", "--":
		default:
			return fmt.Errorf("usage: agent-manager run identity [--json]")
		}
	}

	identity := cliutil.DetectIdentity()
	if !identity.IsIdentityPresent() {
		return fmt.Errorf("no agent-manager run identity: set %s or run inside an agent-manager run", cliutil.EnvIdentityToken)
	}
	verified, err := identity.VerifyIdentity()
	if err != nil {
		return err
	}
	if verified == nil || !verified.Valid || verified.Claims == nil || strings.TrimSpace(verified.Claims.RunID) == "" {
		if verified != nil && verified.Error != "" {
			return fmt.Errorf("agent-manager run identity is invalid: %s", verified.Error)
		}
		return fmt.Errorf("agent-manager run identity is invalid")
	}

	if jsonOutput {
		payload, err := json.Marshal(verified)
		if err != nil {
			return err
		}
		cliutil.PrintJSON(payload)
		return nil
	}
	fmt.Printf("Run ID: %s\n", verified.Claims.RunID)
	fmt.Printf("Task ID: %s\n", verified.Claims.TaskID)
	fmt.Printf("Profile: %s\n", verified.Claims.ProfileKey)
	if verified.RunStatus != "" {
		fmt.Printf("Status: %s\n", verified.RunStatus)
	}
	return nil
}
