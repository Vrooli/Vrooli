package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

// runSandboxSync forwards an explicit sandbox state reconciliation request.
func (a *App) runSandboxSync(args []string) error {
	fs := flag.NewFlagSet("run sandbox-sync", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	status := fs.String("status", "", "Status to sync (required)")
	sandboxID := fs.String("sandbox-id", "", "Sandbox ID")
	actor := fs.String("actor", "", "Actor identifier")
	reason := fs.String("reason", "", "Reason for sync")
	var id string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		id, args = args[0], args[1:]
	}
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if id == "" {
		return fmt.Errorf("usage: agent-manager run sandbox-sync <id> --status <status>")
	}
	if *status == "" {
		return fmt.Errorf("--status is required")
	}
	req := map[string]interface{}{"runId": id, "status": *status}
	if *sandboxID != "" {
		req["sandboxId"] = *sandboxID
	}
	if *actor != "" {
		req["actor"] = *actor
	}
	if *reason != "" {
		req["reason"] = *reason
	}
	payload, err := json.Marshal(req)
	if err != nil {
		return err
	}
	body, err := a.services.Runs.SandboxSync(id, payload)
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}
	fmt.Printf("Synced run: %s\n", id)
	return nil
}
