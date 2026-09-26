package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

// =============================================================================
// Run Apply Investigation
// =============================================================================

func (a *App) runApplyInvestigation(args []string) error {
	fs := flag.NewFlagSet("run apply-investigation", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	customContext := fs.String("context", "", "Custom context for apply run")
	// Parse with positional ID first
	var id string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		id = args[0]
		args = args[1:]
	}
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if id == "" {
		return fmt.Errorf("usage: agent-manager run apply-investigation <investigation-run-id>")
	}
	req := map[string]interface{}{
		"investigationRunId": id,
	}
	if *customContext != "" {
		req["customContext"] = *customContext
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return err
	}
	body, run, err := a.services.Runs.InvestigationApply(payload)
	if err != nil {
		return err
	}
	if *jsonOutput || run == nil {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("Created apply run: %s\n", run.Id)
	return nil
}
