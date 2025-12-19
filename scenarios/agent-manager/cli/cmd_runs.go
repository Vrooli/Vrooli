package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
)

// =============================================================================
// Run Command Dispatcher
// =============================================================================

func (a *App) cmdRun(args []string) error {
	if len(args) == 0 {
		return a.runHelp()
	}

	switch args[0] {
	case "list":
		return a.runList(args[1:])
	case "get":
		return a.runGet(args[1:])
	case "create":
		return a.runCreate(args[1:])
	case "stop":
		return a.runStop(args[1:])
	case "approve":
		return a.runApprove(args[1:])
	case "reject":
		return a.runReject(args[1:])
	case "diff":
		return a.runDiff(args[1:])
	case "events":
		return a.runEvents(args[1:])
	case "help", "-h", "--help":
		return a.runHelp()
	default:
		return fmt.Errorf("unknown run subcommand: %s\n\nRun 'agent-manager run help' for usage", args[0])
	}
}

func (a *App) runHelp() error {
	fmt.Println(`Usage: agent-manager run <subcommand> [options]

Subcommands:
  list              List runs
  get <id>          Get run details and progress
  create            Create and start a new run
  stop <id>         Stop a running execution
  approve <id>      Approve run changes
  reject <id>       Reject run changes
  diff <id>         Show sandbox diff
  events <id>       Get run events (--follow for streaming)

Options:
  --json            Output raw JSON
  --quiet           Output only IDs (for piping)
  --task-id         Filter by task ID
  --profile-id      Filter by profile ID
  --status          Filter by status

Examples:
  agent-manager run list
  agent-manager run list --status running
  agent-manager run create --task-id abc123 --profile-id def456
  agent-manager run get xyz789
  agent-manager run events xyz789 --follow
  agent-manager run approve xyz789 --actor "user@example.com"`)
	return nil
}

// =============================================================================
// Run List
// =============================================================================

func (a *App) runList(args []string) error {
	fs := flag.NewFlagSet("run list", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	quiet := fs.Bool("quiet", false, "Output only IDs")
	limit := fs.Int("limit", 0, "Maximum number of runs to return")
	offset := fs.Int("offset", 0, "Number of runs to skip")
	taskID := fs.String("task-id", "", "Filter by task ID")
	profileID := fs.String("profile-id", "", "Filter by profile ID")
	status := fs.String("status", "", "Filter by status")

	if err := fs.Parse(args); err != nil {
		return err
	}

	body, runs, err := a.services.Runs.List(*limit, *offset, *taskID, *profileID, *status)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	if runs == nil {
		cliutil.PrintJSON(body)
		return nil
	}

	if *quiet {
		for _, r := range runs {
			fmt.Println(r.ID)
		}
		return nil
	}

	if len(runs) == 0 {
		fmt.Println("No runs found")
		return nil
	}

	fmt.Printf("%-36s  %-12s  %-18s  %-4s  %-20s\n", "ID", "STATUS", "PHASE", "PROG", "UPDATED")
	fmt.Printf("%-36s  %-12s  %-18s  %-4s  %-20s\n", strings.Repeat("-", 36), strings.Repeat("-", 12), strings.Repeat("-", 18), strings.Repeat("-", 4), strings.Repeat("-", 20))
	for _, r := range runs {
		phase := r.Phase
		if len(phase) > 18 {
			phase = phase[:15] + "..."
		}
		updated := r.UpdatedAt
		if len(updated) > 20 {
			updated = updated[:19]
		}
		progress := fmt.Sprintf("%d%%", r.ProgressPercent)
		fmt.Printf("%-36s  %-12s  %-18s  %-4s  %-20s\n", r.ID, r.Status, phase, progress, updated)
	}

	return nil
}

// =============================================================================
// Run Get
// =============================================================================

func (a *App) runGet(args []string) error {
	fs := flag.NewFlagSet("run get", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)

	if err := fs.Parse(args); err != nil {
		return err
	}

	remaining := fs.Args()
	if len(remaining) == 0 {
		return fmt.Errorf("usage: agent-manager run get <id>")
	}

	id := remaining[0]
	body, run, err := a.services.Runs.Get(id)
	if err != nil {
		return err
	}

	if *jsonOutput || run == nil {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("ID:              %s\n", run.ID)
	fmt.Printf("Task ID:         %s\n", run.TaskID)
	fmt.Printf("Profile ID:      %s\n", run.AgentProfileID)
	fmt.Printf("Status:          %s\n", run.Status)
	fmt.Printf("Phase:           %s\n", run.Phase)
	fmt.Printf("Progress:        %d%%\n", run.ProgressPercent)
	fmt.Printf("Run Mode:        %s\n", run.RunMode)
	if run.SandboxID != "" {
		fmt.Printf("Sandbox ID:      %s\n", run.SandboxID)
	}
	if run.StartedAt != "" {
		fmt.Printf("Started:         %s\n", run.StartedAt)
	}
	if run.EndedAt != "" {
		fmt.Printf("Ended:           %s\n", run.EndedAt)
	}
	if run.ApprovalState != "" && run.ApprovalState != "none" {
		fmt.Printf("Approval State:  %s\n", run.ApprovalState)
		if run.ApprovedBy != "" {
			fmt.Printf("Approved By:     %s\n", run.ApprovedBy)
		}
	}
	if run.Summary != nil {
		fmt.Println("Summary:")
		if run.Summary.Description != "" {
			fmt.Printf("  Description:   %s\n", run.Summary.Description)
		}
		if run.Summary.TurnsUsed > 0 {
			fmt.Printf("  Turns Used:    %d\n", run.Summary.TurnsUsed)
		}
		if run.Summary.TokensUsed > 0 {
			fmt.Printf("  Tokens Used:   %d\n", run.Summary.TokensUsed)
		}
		if run.Summary.CostEstimate > 0 {
			fmt.Printf("  Cost Estimate: $%.4f\n", run.Summary.CostEstimate)
		}
	}
	if run.ErrorMsg != "" {
		fmt.Printf("Error:           %s\n", run.ErrorMsg)
	}
	if run.ExitCode != nil {
		fmt.Printf("Exit Code:       %d\n", *run.ExitCode)
	}
	if run.ChangedFiles > 0 {
		fmt.Printf("Changed Files:   %d\n", run.ChangedFiles)
	}

	return nil
}

// =============================================================================
// Run Create
// =============================================================================

func (a *App) runCreate(args []string) error {
	fs := flag.NewFlagSet("run create", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	taskID := fs.String("task-id", "", "Task ID (required)")
	profileID := fs.String("profile-id", "", "Agent profile ID (required)")
	prompt := fs.String("prompt", "", "Optional override prompt")
	runMode := fs.String("run-mode", "", "Run mode (sandboxed or in_place)")
	forceInPlace := fs.Bool("force-in-place", false, "Force in-place execution")
	idempotencyKey := fs.String("idempotency-key", "", "Idempotency key for safe retries")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *taskID == "" {
		return fmt.Errorf("--task-id is required")
	}
	if *profileID == "" {
		return fmt.Errorf("--profile-id is required")
	}

	req := CreateRunRequest{
		TaskID:         *taskID,
		AgentProfileID: *profileID,
		Prompt:         *prompt,
		RunMode:        *runMode,
		ForceInPlace:   *forceInPlace,
		IdempotencyKey: *idempotencyKey,
	}

	body, run, err := a.services.Runs.Create(req)
	if err != nil {
		return err
	}

	if *jsonOutput || run == nil {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("Created run: %s\n", run.ID)
	fmt.Printf("Status: %s\n", run.Status)
	fmt.Printf("Phase: %s\n", run.Phase)
	return nil
}

// =============================================================================
// Run Stop
// =============================================================================

func (a *App) runStop(args []string) error {
	fs := flag.NewFlagSet("run stop", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)

	// Parse with positional ID first
	var id string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		id = args[0]
		args = args[1:]
	}

	if err := fs.Parse(args); err != nil {
		return err
	}

	if id == "" {
		return fmt.Errorf("usage: agent-manager run stop <id>")
	}

	body, err := a.services.Runs.Stop(id)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("Stopped run: %s\n", id)
	return nil
}

// =============================================================================
// Run Approve
// =============================================================================

func (a *App) runApprove(args []string) error {
	fs := flag.NewFlagSet("run approve", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	actor := fs.String("actor", "", "Who is approving")
	commitMsg := fs.String("commit-msg", "", "Commit message for changes")
	force := fs.Bool("force", false, "Force approval despite conflicts")

	// Parse with positional ID first
	var id string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		id = args[0]
		args = args[1:]
	}

	if err := fs.Parse(args); err != nil {
		return err
	}

	if id == "" {
		return fmt.Errorf("usage: agent-manager run approve <id> [options]")
	}

	req := ApproveRequest{
		Actor:     *actor,
		CommitMsg: *commitMsg,
		Force:     *force,
	}

	body, result, err := a.services.Runs.Approve(id, req)
	if err != nil {
		return err
	}

	if *jsonOutput || result == nil {
		cliutil.PrintJSON(body)
		return nil
	}

	if result.Success {
		fmt.Printf("Approved run: %s\n", id)
		fmt.Printf("Applied: %d files\n", result.Applied)
		if result.CommitHash != "" {
			fmt.Printf("Commit: %s\n", result.CommitHash)
		}
	} else {
		fmt.Printf("Approval failed: %s\n", result.ErrorMsg)
	}
	return nil
}

// =============================================================================
// Run Reject
// =============================================================================

func (a *App) runReject(args []string) error {
	fs := flag.NewFlagSet("run reject", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	actor := fs.String("actor", "", "Who is rejecting")
	reason := fs.String("reason", "", "Reason for rejection")

	// Parse with positional ID first
	var id string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		id = args[0]
		args = args[1:]
	}

	if err := fs.Parse(args); err != nil {
		return err
	}

	if id == "" {
		return fmt.Errorf("usage: agent-manager run reject <id> [options]")
	}

	req := RejectRequest{
		Actor:  *actor,
		Reason: *reason,
	}

	body, err := a.services.Runs.Reject(id, req)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("Rejected run: %s\n", id)
	return nil
}

// =============================================================================
// Run Diff
// =============================================================================

func (a *App) runDiff(args []string) error {
	fs := flag.NewFlagSet("run diff", flag.ContinueOnError)

	// Parse with positional ID first
	var id string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		id = args[0]
		args = args[1:]
	}

	if err := fs.Parse(args); err != nil {
		return err
	}

	if id == "" {
		return fmt.Errorf("usage: agent-manager run diff <id>")
	}

	body, err := a.services.Runs.GetDiff(id)
	if err != nil {
		return err
	}

	// Just print the diff output directly
	fmt.Println(string(body))
	return nil
}

// =============================================================================
// Run Events
// =============================================================================

func (a *App) runEvents(args []string) error {
	fs := flag.NewFlagSet("run events", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	follow := fs.Bool("follow", false, "Stream events in real-time (WebSocket)")
	limit := fs.Int("limit", 0, "Maximum number of events to return")

	// Parse with positional ID first
	var id string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		id = args[0]
		args = args[1:]
	}

	if err := fs.Parse(args); err != nil {
		return err
	}

	if id == "" {
		return fmt.Errorf("usage: agent-manager run events <id> [--follow]")
	}

	if *follow {
		return a.streamEvents(id)
	}

	body, events, err := a.services.Runs.GetEvents(id, *limit)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	if events == nil {
		cliutil.PrintJSON(body)
		return nil
	}

	if len(events) == 0 {
		fmt.Println("No events found")
		return nil
	}

	fmt.Printf("%-6s  %-12s  %-24s  %s\n", "SEQ", "TYPE", "TIMESTAMP", "DATA")
	fmt.Printf("%-6s  %-12s  %-24s  %s\n", strings.Repeat("-", 6), strings.Repeat("-", 12), strings.Repeat("-", 24), strings.Repeat("-", 40))
	for _, e := range events {
		dataStr := ""
		if e.Data != nil {
			dataStr = fmt.Sprintf("%v", e.Data)
			if len(dataStr) > 60 {
				dataStr = dataStr[:57] + "..."
			}
		}
		timestamp := e.Timestamp.Format("2006-01-02 15:04:05")
		fmt.Printf("%-6d  %-12s  %-24s  %s\n", e.Sequence, e.EventType, timestamp, dataStr)
	}

	return nil
}

// streamEvents is implemented in cmd_events.go
