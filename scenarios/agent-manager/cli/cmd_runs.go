package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
	"google.golang.org/protobuf/proto"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
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
	case "get-by-tag":
		return a.runGetByTag(args[1:])
	case "create":
		return a.runCreate(args[1:])
	case "delete":
		return a.runDelete(args[1:])
	case "stop":
		return a.runStop(args[1:])
	case "stop-by-tag":
		return a.runStopByTag(args[1:])
	case "stop-all":
		return a.runStopAll(args[1:])
	case "continue":
		return a.runContinue(args[1:])
	case "investigate":
		return a.runInvestigate(args[1:])
	case "apply-investigation":
		return a.runApplyInvestigation(args[1:])
	case "sandbox-sync":
		return a.runSandboxSync(args[1:])
	case "extract-recommendations":
		return a.runExtractRecommendations(args[1:])
	case "regenerate-recommendations":
		return a.runRegenerateRecommendations(args[1:])
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
  list                        List runs (with optional filters)
  get <id>                    Get run details by UUID
  get-by-tag <tag>            Get run details by custom tag
  create                      Create and start a new run
  delete <id>                 Delete a run
  stop <id>                   Stop a run by UUID
  stop-by-tag <tag>           Stop a run by custom tag
  stop-all                    Stop all running runs
  continue <id>               Continue a run with a follow-up message
  investigate                 Create an investigation run from existing runs
  apply-investigation <id>    Apply investigation recommendations
  sandbox-sync <id>           Sync run state from sandbox
  extract-recommendations <id>     Extract recommendations from investigation run
  regenerate-recommendations <id>  Regenerate recommendations for investigation run
  approve <id>                Approve run changes
  reject <id>                 Reject run changes
  diff <id>                   Show sandbox diff
  events <id>                 Get run events (--follow for streaming)

Filters (for 'list'):
  --task-id         Filter by task ID
  --profile-id      Filter by profile ID
  --status          Filter by status (running, pending, complete, etc.)
  --tag-prefix      Filter by tag prefix (e.g., "ecosystem-")

Options:
  --json            Output raw JSON
  --quiet           Output only IDs (for piping)

Examples:
  agent-manager run list
  agent-manager run list --status running
  agent-manager run create --task-id abc123 --profile-id def456
  agent-manager run delete abc123 --force
  agent-manager run continue abc123 --message "Also update tests"
  agent-manager run investigate --run-ids id1,id2 --depth standard
  agent-manager run apply-investigation abc123
  agent-manager run extract-recommendations abc123
  agent-manager run events xyz789 --follow`)
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
	tagPrefix := fs.String("tag-prefix", "", "Filter by tag prefix")

	if err := fs.Parse(args); err != nil {
		return err
	}

	body, runs, err := a.services.Runs.List(*limit, *offset, *taskID, *profileID, *status, *tagPrefix)
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
			fmt.Println(r.Id)
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
		phase := formatEnumValue(r.Phase, "RUN_PHASE_", "_")
		if len(phase) > 18 {
			phase = phase[:15] + "..."
		}
		updated := formatTimestamp(r.UpdatedAt)
		if len(updated) > 20 {
			updated = updated[:19]
		}
		progress := fmt.Sprintf("%d%%", r.ProgressPercent)
		status := formatEnumValue(r.Status, "RUN_STATUS_", "_")
		fmt.Printf("%-36s  %-12s  %-18s  %-4s  %-20s\n", r.Id, status, phase, progress, updated)
	}

	return nil
}

// =============================================================================
// Run Get
// =============================================================================

func (a *App) runGet(args []string) error {
	fs := flag.NewFlagSet("run get", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)

	var id string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		id = args[0]
		args = args[1:]
	}

	if err := fs.Parse(args); err != nil {
		return err
	}

	if id == "" {
		remaining := fs.Args()
		if len(remaining) == 0 {
			return fmt.Errorf("usage: agent-manager run get <id>")
		}
		id = remaining[0]
	}

	body, run, err := a.services.Runs.Get(id)
	if err != nil {
		return err
	}

	if *jsonOutput || run == nil {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("ID:              %s\n", run.Id)
	fmt.Printf("Task ID:         %s\n", run.TaskId)
	if run.AgentProfileId != nil {
		fmt.Printf("Profile ID:      %s\n", run.GetAgentProfileId())
	}
	fmt.Printf("Status:          %s\n", formatEnumValue(run.Status, "RUN_STATUS_", "_"))
	fmt.Printf("Phase:           %s\n", formatEnumValue(run.Phase, "RUN_PHASE_", "_"))
	fmt.Printf("Progress:        %d%%\n", run.ProgressPercent)
	fmt.Printf("Run Mode:        %s\n", formatEnumValue(run.RunMode, "RUN_MODE_", "_"))
	if run.SandboxId != nil && run.GetSandboxId() != "" {
		fmt.Printf("Sandbox ID:      %s\n", run.GetSandboxId())
	}
	if started := formatTimestamp(run.StartedAt); started != "" {
		fmt.Printf("Started:         %s\n", started)
	}
	if ended := formatTimestamp(run.EndedAt); ended != "" {
		fmt.Printf("Ended:           %s\n", ended)
	}
	if approval := formatEnumValue(run.ApprovalState, "APPROVAL_STATE_", "_"); approval != "" && approval != "none" {
		fmt.Printf("Approval State:  %s\n", approval)
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
		fmt.Printf("Exit Code:       %d\n", run.GetExitCode())
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
	existingSandboxID := fs.String("existing-sandbox-id", "", "Reuse an existing sandbox ID (sandboxed runs only)")
	sandboxConfig := fs.String("sandbox-config", "", "Sandbox config JSON (proto JSON)")
	sandboxConfigFile := fs.String("sandbox-config-file", "", "Path to sandbox config JSON")
	sandboxRetentionMode := fs.String("sandbox-retention-mode", "", "Sandbox retention mode (keep_active, stop_on_terminal, delete_on_terminal)")
	sandboxRetentionTTL := fs.String("sandbox-retention-ttl", "", "Sandbox retention TTL (e.g., 2h, 30m)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *taskID == "" {
		return fmt.Errorf("--task-id is required")
	}
	if *profileID == "" {
		return fmt.Errorf("--profile-id is required")
	}

	req := &apipb.CreateRunRequest{
		TaskId: *taskID,
	}
	if *profileID != "" {
		req.AgentProfileId = protoString(*profileID)
	}
	if *prompt != "" {
		req.Prompt = protoString(*prompt)
	}
	if *idempotencyKey != "" {
		req.IdempotencyKey = protoString(*idempotencyKey)
	}
	if *existingSandboxID != "" {
		req.ExistingSandboxId = protoString(*existingSandboxID)
	}
	if *runMode != "" {
		mode := parseRunMode(*runMode)
		if mode == domainpb.RunMode_RUN_MODE_UNSPECIFIED {
			return fmt.Errorf("invalid run mode: %s", *runMode)
		}
		req.RunMode = &mode
	} else if *forceInPlace {
		mode := domainpb.RunMode_RUN_MODE_IN_PLACE
		req.RunMode = &mode
	}
	if cfg, err := parseSandboxConfig(*sandboxConfig, *sandboxConfigFile); err != nil {
		return err
	} else {
		cfg, err = applySandboxRetention(cfg, *sandboxRetentionMode, *sandboxRetentionTTL)
		if err != nil {
			return err
		}
		if cfg != nil {
			req.InlineConfig = &domainpb.RunConfigOverrides{
				SandboxConfig: cfg,
			}
		}
	}

	body, run, err := a.services.Runs.Create(req)
	if err != nil {
		return err
	}

	if *jsonOutput || run == nil {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("Created run: %s\n", run.Id)
	fmt.Printf("Status: %s\n", formatEnumValue(run.Status, "RUN_STATUS_", "_"))
	fmt.Printf("Phase: %s\n", formatEnumValue(run.Phase, "RUN_PHASE_", "_"))
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
// Run Get By Tag
// =============================================================================

func (a *App) runGetByTag(args []string) error {
	fs := flag.NewFlagSet("run get-by-tag", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)

	// Parse with positional tag first
	var tag string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		tag = args[0]
		args = args[1:]
	}

	if err := fs.Parse(args); err != nil {
		return err
	}

	if tag == "" {
		return fmt.Errorf("usage: agent-manager run get-by-tag <tag>")
	}

	body, run, err := a.services.Runs.GetByTag(tag)
	if err != nil {
		return err
	}

	if *jsonOutput || run == nil {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("ID:              %s\n", run.Id)
	fmt.Printf("Tag:             %s\n", run.Tag)
	fmt.Printf("Task ID:         %s\n", run.TaskId)
	fmt.Printf("Status:          %s\n", formatEnumValue(run.Status, "RUN_STATUS_", "_"))
	fmt.Printf("Phase:           %s\n", formatEnumValue(run.Phase, "RUN_PHASE_", "_"))
	fmt.Printf("Progress:        %d%%\n", run.ProgressPercent)
	if started := formatTimestamp(run.StartedAt); started != "" {
		fmt.Printf("Started:         %s\n", started)
	}
	if run.ErrorMsg != "" {
		fmt.Printf("Error:           %s\n", run.ErrorMsg)
	}

	return nil
}

// =============================================================================
// Run Stop By Tag
// =============================================================================

func (a *App) runStopByTag(args []string) error {
	fs := flag.NewFlagSet("run stop-by-tag", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)

	// Parse with positional tag first
	var tag string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		tag = args[0]
		args = args[1:]
	}

	if err := fs.Parse(args); err != nil {
		return err
	}

	if tag == "" {
		return fmt.Errorf("usage: agent-manager run stop-by-tag <tag>")
	}

	body, err := a.services.Runs.StopByTag(tag)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("Stopped run with tag: %s\n", tag)
	return nil
}

// =============================================================================
// Run Stop All
// =============================================================================

func (a *App) runStopAll(args []string) error {
	fs := flag.NewFlagSet("run stop-all", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	tagPrefix := fs.String("tag-prefix", "", "Only stop runs with this tag prefix")
	force := fs.Bool("force", false, "Force termination even if graceful stop fails")

	if err := fs.Parse(args); err != nil {
		return err
	}

	req := &apipb.StopAllRunsRequest{Force: *force}
	if *tagPrefix != "" {
		req.TagPrefix = protoString(*tagPrefix)
	}
	body, result, err := a.services.Runs.StopAll(req)
	if err != nil {
		return err
	}

	if *jsonOutput || result == nil {
		cliutil.PrintJSON(body)
		return nil
	}

	if result != nil {
		fmt.Printf("Stopped:  %d\n", result.StoppedCount)
		if len(result.Failures) > 0 {
			fmt.Printf("Failed:   %d\n", len(result.Failures))
			failedIDs := make([]string, 0, len(result.Failures))
			for _, failure := range result.Failures {
				if failure == nil {
					continue
				}
				failedIDs = append(failedIDs, failure.RunId)
			}
			if len(failedIDs) > 0 {
				fmt.Printf("Failed IDs: %v\n", failedIDs)
			}
		}
	}
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

	req := &apipb.ApproveRunRequest{
		RunId: id,
		Force: *force,
	}
	if trimmed := strings.TrimSpace(*actor); trimmed != "" {
		req.Actor = protoString(trimmed)
	}
	if *commitMsg != "" {
		req.CommitMsg = protoString(*commitMsg)
	}

	body, result, err := a.services.Runs.Approve(id, req)
	if err != nil {
		return err
	}

	if *jsonOutput || result == nil {
		cliutil.PrintJSON(body)
		return nil
	}

	if result != nil && result.Success {
		fmt.Printf("Approved run: %s\n", id)
		fmt.Printf("Applied: %d files\n", result.FilesApplied)
		if result.CommitHash != "" {
			fmt.Printf("Commit: %s\n", result.CommitHash)
		}
	} else {
		message := ""
		if result != nil {
			message = result.Message
		}
		fmt.Printf("Approval failed: %s\n", message)
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

	req := &apipb.RejectRunRequest{
		RunId:  id,
		Reason: *reason,
	}
	if trimmed := strings.TrimSpace(*actor); trimmed != "" {
		req.Actor = protoString(trimmed)
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

	body, diff, err := a.services.Runs.GetDiff(id)
	if err != nil {
		return err
	}

	// Just print the diff output directly
	if diff != nil && diff.Content != "" {
		fmt.Println(diff.Content)
	} else if diff != nil && len(diff.Files) > 0 {
		fmt.Println("No unified diff content available. Changed files:")
		for _, file := range diff.Files {
			fmt.Printf("- %s (%s, +%d -%d)\n", file.Path, file.ChangeType, file.Additions, file.Deletions)
		}
	} else {
		fmt.Println(string(body))
	}
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
		dataStr := runEventDataString(e)
		if len(dataStr) > 60 {
			dataStr = dataStr[:57] + "..."
		}
		timestamp := formatTimestamp(e.Timestamp)
		if timestamp != "" {
			timestamp = trimTimestamp(timestamp)
		}
		eventType := formatEnumValue(e.EventType, "RUN_EVENT_TYPE_", "_")
		fmt.Printf("%-6d  %-12s  %-24s  %s\n", e.Sequence, eventType, timestamp, dataStr)
	}

	return nil
}

// streamEvents is implemented in cmd_events.go

func runEventDataString(event *domainpb.RunEvent) string {
	if event == nil {
		return ""
	}

	var payload proto.Message
	switch data := event.Data.(type) {
	case *domainpb.RunEvent_Log:
		payload = data.Log
	case *domainpb.RunEvent_Message:
		payload = data.Message
	case *domainpb.RunEvent_ToolCall:
		payload = data.ToolCall
	case *domainpb.RunEvent_ToolResult:
		payload = data.ToolResult
	case *domainpb.RunEvent_Status:
		payload = data.Status
	case *domainpb.RunEvent_Metric:
		payload = data.Metric
	case *domainpb.RunEvent_Artifact:
		payload = data.Artifact
	case *domainpb.RunEvent_Error:
		payload = data.Error
	case *domainpb.RunEvent_Progress:
		payload = data.Progress
	case *domainpb.RunEvent_Cost:
		payload = data.Cost
	case *domainpb.RunEvent_RateLimit:
		payload = data.RateLimit
	default:
		return ""
	}

	return marshalProtoJSON(payload)
}

// =============================================================================
// Run Delete
// =============================================================================

func (a *App) runDelete(args []string) error {
	fs := flag.NewFlagSet("run delete", flag.ContinueOnError)
	force := fs.Bool("force", false, "Skip confirmation")

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
		return fmt.Errorf("usage: agent-manager run delete <id>")
	}

	if !*force {
		fmt.Printf("Delete run %s? [y/N]: ", id)
		var confirm string
		fmt.Scanln(&confirm)
		if strings.ToLower(confirm) != "y" && strings.ToLower(confirm) != "yes" {
			fmt.Println("Cancelled")
			return nil
		}
	}

	if err := a.services.Runs.Delete(id); err != nil {
		return err
	}

	fmt.Printf("Deleted run: %s\n", id)
	return nil
}

// =============================================================================
// Run Continue
// =============================================================================

func (a *App) runContinue(args []string) error {
	fs := flag.NewFlagSet("run continue", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	message := fs.String("message", "", "Follow-up message (required)")

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
		return fmt.Errorf("usage: agent-manager run continue <id> --message <message>")
	}

	if *message == "" {
		return fmt.Errorf("--message is required")
	}

	req := &domainpb.ContinueRunRequest{
		RunId:   id,
		Message: *message,
	}

	body, run, err := a.services.Runs.Continue(id, req)
	if err != nil {
		return err
	}

	if *jsonOutput || run == nil {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("Continued run: %s (status: %s)\n", run.Id, formatEnumValue(run.Status, "RUN_STATUS_", "_"))
	return nil
}

// =============================================================================
// Run Investigate
// =============================================================================

func (a *App) runInvestigate(args []string) error {
	fs := flag.NewFlagSet("run investigate", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	runIDs := fs.String("run-ids", "", "Comma-separated run IDs to investigate (required)")
	customContext := fs.String("context", "", "Custom context for investigation")
	depth := fs.String("depth", "standard", "Investigation depth: quick, standard, deep")
	projectRoot := fs.String("project-root", "", "Project root directory")
	scopePaths := fs.String("scope-paths", "", "Comma-separated scope paths")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *runIDs == "" {
		return fmt.Errorf("--run-ids is required")
	}

	ids := strings.Split(*runIDs, ",")
	for i, id := range ids {
		ids[i] = strings.TrimSpace(id)
	}

	req := map[string]interface{}{
		"runIds": ids,
	}
	if *customContext != "" {
		req["customContext"] = *customContext
	}
	if *depth != "" {
		req["depth"] = *depth
	}
	if *projectRoot != "" {
		req["projectRoot"] = *projectRoot
	}
	if *scopePaths != "" {
		paths := strings.Split(*scopePaths, ",")
		for i, p := range paths {
			paths[i] = strings.TrimSpace(p)
		}
		req["scopePaths"] = paths
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return err
	}

	body, run, err := a.services.Runs.Investigate(payload)
	if err != nil {
		return err
	}

	if *jsonOutput || run == nil {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("Created investigation run: %s\n", run.Id)
	return nil
}

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

	if err := fs.Parse(args); err != nil {
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

// =============================================================================
// Run Sandbox Sync
// =============================================================================

func (a *App) runSandboxSync(args []string) error {
	fs := flag.NewFlagSet("run sandbox-sync", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	status := fs.String("status", "", "Status to sync (required)")
	sandboxID := fs.String("sandbox-id", "", "Sandbox ID")
	actor := fs.String("actor", "", "Actor identifier")
	reason := fs.String("reason", "", "Reason for sync")

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
		return fmt.Errorf("usage: agent-manager run sandbox-sync <id> --status <status>")
	}

	if *status == "" {
		return fmt.Errorf("--status is required")
	}

	req := map[string]interface{}{
		"runId":  id,
		"status": *status,
	}
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

// =============================================================================
// Run Extract Recommendations
// =============================================================================

func (a *App) runExtractRecommendations(args []string) error {
	fs := flag.NewFlagSet("run extract-recommendations", flag.ContinueOnError)
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
		return fmt.Errorf("usage: agent-manager run extract-recommendations <id>")
	}

	body, err := a.services.Runs.ExtractRecommendations(id)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	// Pretty print the JSON result
	var prettyJSON interface{}
	if err := json.Unmarshal(body, &prettyJSON); err == nil {
		formatted, _ := json.MarshalIndent(prettyJSON, "", "  ")
		fmt.Println(string(formatted))
	} else {
		cliutil.PrintJSON(body)
	}

	return nil
}

// =============================================================================
// Run Regenerate Recommendations
// =============================================================================

func (a *App) runRegenerateRecommendations(args []string) error {
	fs := flag.NewFlagSet("run regenerate-recommendations", flag.ContinueOnError)
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
		return fmt.Errorf("usage: agent-manager run regenerate-recommendations <id>")
	}

	body, err := a.services.Runs.RegenerateRecommendations(id)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("Regenerating recommendations for run: %s\n", id)
	return nil
}
