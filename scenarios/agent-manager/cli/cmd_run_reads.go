// Responsibility: retain cmd runs declarations within their original package.
package main

import (
	"agent-manager/cli/internal/support"
	"flag"
	"fmt"
	"github.com/vrooli/cli-core/cliutil"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/proto"
	"sort"
	"strings"
	"time"
)

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
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
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

	fmt.Printf("%-36s  %-28s  %-12s  %-18s  %-11s  %-4s  %-10s  %-18s  %-20s\n", "ID", "LABEL", "STATUS", "PHASE", "EXEC", "PROG", "SOURCE", "SESSION", "UPDATED")
	fmt.Printf("%-36s  %-28s  %-12s  %-18s  %-11s  %-4s  %-10s  %-18s  %-20s\n", strings.Repeat("-", 36), strings.Repeat("-", 28), strings.Repeat("-", 12), strings.Repeat("-", 18), strings.Repeat("-", 11), strings.Repeat("-", 4), strings.Repeat("-", 10), strings.Repeat("-", 18), strings.Repeat("-", 20))
	for _, r := range runs {
		label := r.Label
		if label == "" {
			label = "-"
		}
		if len(label) > 28 {
			label = label[:25] + "..."
		}
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
		exec := formatEnumValue(r.ExecutionMode, "EXECUTION_MODE_", "_")
		if exec == "" || exec == "unspecified" {
			exec = "codec_pipe"
		}
		source := "native"
		session := "-"
		if exec == "imported" {
			source = r.ImportSourceHarness
			if source == "" {
				source = "imported"
			}
			session = r.ImportSourceSessionId
			if session == "" {
				session = "-"
			}
		}
		if len(source) > 10 {
			source = source[:10]
		}
		if len(session) > 18 {
			session = session[:18]
		}
		fmt.Printf("%-36s  %-28s  %-12s  %-18s  %-11s  %-4s  %-10s  %-18s  %-20s\n", r.Id, label, status, phase, exec, progress, source, session, updated)
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

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
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
	if run.Label != "" {
		fmt.Printf("Label:           %s\n", run.Label)
		if run.LabelSource != "" {
			fmt.Printf("Label Source:    %s\n", run.LabelSource)
		}
	}
	if run.AgentProfileId != nil {
		fmt.Printf("Profile ID:      %s\n", run.GetAgentProfileId())
	}
	fmt.Printf("Status:          %s\n", formatEnumValue(run.Status, "RUN_STATUS_", "_"))
	fmt.Printf("Phase:           %s\n", formatEnumValue(run.Phase, "RUN_PHASE_", "_"))
	fmt.Printf("Progress:        %d%%\n", run.ProgressPercent)
	fmt.Printf("Run Mode:        %s\n", formatEnumValue(run.RunMode, "RUN_MODE_", "_"))
	fmt.Printf("Execution Mode:  %s\n", formatEnumValue(run.ExecutionMode, "EXECUTION_MODE_", "_"))
	if run.ObservedGoalStatus != "" {
		goal := run.ObservedGoalStatus
		if goal == "paused" {
			goal += " (provider nonproductive; session may remain live)"
		}
		fmt.Printf("Provider Goal:   %s\n", goal)
	} else if run.ExecutionMode == domainpb.ExecutionMode_EXECUTION_MODE_INTERACTIVE && run.ResolvedConfig != nil && strings.TrimSpace(run.ResolvedConfig.Until) != "" {
		fmt.Println("Provider Goal:   unknown (no qualified marker)")
	}
	if activity := formatTimestamp(run.ProviderActivityAt); activity != "" {
		fmt.Printf("Provider Activity: %s (%s ago)\n", activity, time.Since(run.ProviderActivityAt.AsTime()).Round(time.Second))
	} else if run.ExecutionMode == domainpb.ExecutionMode_EXECUTION_MODE_INTERACTIVE {
		fmt.Println("Provider Activity: unknown")
	}
	if heartbeat := formatTimestamp(run.LastHeartbeat); heartbeat != "" {
		fmt.Printf("Coordinator Heartbeat: %s (liveness only)\n", heartbeat)
	}
	if run.WebConsoleSessionId != "" {
		fmt.Printf("Live Session:    %s\n", run.WebConsoleSessionId)
		if run.WebConsoleSessionUrl != "" {
			fmt.Printf("Live Session URL: %s\n", run.WebConsoleSessionUrl)
		}
	}
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
		if run.Summary.ContextTokens > 0 {
			fmt.Printf("  Context:       %d tokens\n", run.Summary.ContextTokens)
		}
		if run.Summary.CostEstimate > 0 {
			fmt.Printf("  Cost Estimate: $%.4f\n", run.Summary.CostEstimate)
		}
	}
	if run.Result != nil {
		if run.Result.Selection != nil {
			fmt.Printf("Result Selection: %s\n", formatEnumValue(run.Result.Selection.Status, "FINAL_OUTPUT_SELECTION_STATUS_", "_"))
		} else {
			fmt.Println("Result Selection: unavailable")
		}
		if run.Result.Structured != nil {
			fmt.Printf("Structured:      %s", formatEnumValue(run.Result.Structured.Status, "STRUCTURED_RESULT_STATUS_", "_"))
			if run.Result.Structured.Method != "" {
				fmt.Printf(" (%s)", run.Result.Structured.Method)
			}
			fmt.Println()
			if len(run.Result.Structured.Value) > 0 {
				fmt.Printf("Structured Value: %s\n", string(run.Result.Structured.Value))
			}
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
// Run Attach / Detach
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

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
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

func (a *App) runDiff(args []string) error {
	fs := flag.NewFlagSet("run diff", flag.ContinueOnError)
	stat := fs.Bool("stat", false, "Show changed-file statistics without unified diff content")

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
		return fmt.Errorf("usage: agent-manager run diff <id> [--stat]")
	}

	body, diff, err := a.services.Runs.GetDiff(id)
	if err != nil {
		return err
	}
	if *stat && diff != nil {
		additions, deletions := int64(0), int64(0)
		for _, file := range diff.Files {
			additions += int64(file.Additions)
			deletions += int64(file.Deletions)
		}
		fmt.Printf("Files: %d | additions: %d | deletions: %d\n", len(diff.Files), additions, deletions)
		support.NextSteps(fmt.Sprintf("agent-manager run report %s", id), fmt.Sprintf("agent-manager run diff %s", id))
		return nil
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
	support.NextSteps(fmt.Sprintf("agent-manager run diff %s --stat", id), fmt.Sprintf("agent-manager run report %s", id))
	return nil
}

func (a *App) runEvents(args []string) error {
	fs := flag.NewFlagSet("run events", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	follow := fs.Bool("follow", false, "Stream events in real-time (WebSocket)")
	limit := fs.Int("limit", 0, "Maximum number of events to return")
	afterSequence := fs.Int64("after-sequence", -1, "Only return events with sequence greater than this value")
	stats := fs.Bool("stats", false, "Show an event-type summary instead of event payloads")
	failed := fs.Bool("failed", false, "Only show failed tool-result events")

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
		return fmt.Errorf("usage: agent-manager run events <id> [--follow] [--stats] [--failed] [--after-sequence N] [--limit N]")
	}

	if *follow {
		return a.streamEvents(id)
	}

	var after *int64
	if *afterSequence >= 0 {
		after = afterSequence
	}
	body, events, err := a.services.Runs.GetEvents(id, *limit, after)
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
	if *stats {
		counts := map[string]int{}
		for _, event := range events {
			counts[formatEnumValue(event.EventType, "RUN_EVENT_TYPE_", "_")]++
		}
		keys := make([]string, 0, len(counts))
		for key := range counts {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			fmt.Printf("%s=%d ", key, counts[key])
		}
		fmt.Println()
		support.NextSteps(fmt.Sprintf("agent-manager run report %s", id), fmt.Sprintf("agent-manager run tools %s --failed", id))
		return nil
	}

	fmt.Printf("%-6s  %-12s  %-24s  %s\n", "SEQ", "TYPE", "TIMESTAMP", "DATA")
	fmt.Printf("%-6s  %-12s  %-24s  %s\n", strings.Repeat("-", 6), strings.Repeat("-", 12), strings.Repeat("-", 24), strings.Repeat("-", 40))
	for _, e := range events {
		if *failed && (e.GetToolResult() == nil || e.GetToolResult().GetSuccess()) {
			continue
		}
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
	support.NextSteps(fmt.Sprintf("agent-manager run report %s", id), fmt.Sprintf("agent-manager run tools %s --failed", id))

	return nil
}

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
	case *domainpb.RunEvent_MessageDeleted:
		payload = data.MessageDeleted
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
	case *domainpb.RunEvent_Compaction:
		payload = data.Compaction
	default:
		return ""
	}

	return marshalProtoJSON(payload)
}
