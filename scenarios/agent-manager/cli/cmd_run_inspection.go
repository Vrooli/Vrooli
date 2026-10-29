package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"agent-manager/cli/internal/support"
	"github.com/vrooli/cli-core/cliutil"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	measurepb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures"
	sharedmeasurepb "github.com/vrooli/vrooli/packages/proto/gen/go/measures/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (a *App) runReport(args []string) error {
	fs := flag.NewFlagSet("run report", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return fmt.Errorf("usage: agent-manager run report <id> [--json]")
	}
	id := args[0]
	if err := cliutil.ParseInterspersed(fs, args[1:]); err != nil {
		return err
	}
	body, err := a.services.Runs.GetReport(id)
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}
	var r struct {
		RunID     string      `json:"run_id"`
		Status    string      `json:"status"`
		Error     string      `json:"error"`
		Turns     int         `json:"turns"`
		Tokens    json.Number `json:"tokens"`
		Cost      float64     `json:"cost_usd"`
		Project   int         `json:"project_owned_tool_calls"`
		External  int         `json:"external_tool_calls"`
		Fallbacks int         `json:"fallback_count"`
		Repeated  int         `json:"repeated_tool_calls"`
		Rereads   int         `json:"files_read_more_than_once"`
		Longest   int64       `json:"longest_event_gap_ms,string"`
		Requested string      `json:"requested_model"`
		Actual    string      `json:"actual_model"`
		Result    struct {
			Selection   string   `json:"selection_status"`
			Rule        string   `json:"selection_rule"`
			Candidates  int      `json:"candidate_count"`
			Structured  string   `json:"structured_status"`
			Method      string   `json:"structured_method"`
			Diagnostics []string `json:"diagnostic_codes"`
		} `json:"result"`
		Tools []struct {
			Name           string `json:"name"`
			Calls          int    `json:"calls"`
			Successes      int    `json:"successes"`
			Failures       int    `json:"failures"`
			Unresolved     int    `json:"unresolved"`
			Returned       int    `json:"returned"`
			UnknownOutcome int    `json:"unknown_outcome"`
		} `json:"tools"`
		Events       map[string]int `json:"event_counts"`
		ReceiptCount int            `json:"receipt_count"`
		Receipts     struct {
			State string `json:"state"`
		} `json:"receipts_availability"`
		Diff struct {
			Files int `json:"files"`
			// Proto JSON encodes int64 fields as decimal strings. Keep this
			// decoder aligned with the report endpoint rather than relying on
			// the historical numeric-only test fixture.
			Bytes     int64 `json:"bytes,string"`
			Available struct {
				State string `json:"state"`
			} `json:"available"`
		} `json:"diff"`
	}
	if err := json.Unmarshal(body, &r); err != nil {
		return fmt.Errorf("decode run report: %w", err)
	}
	tokens := int64(0)
	if r.Tokens != "" {
		tokens, err = r.Tokens.Int64()
		if err != nil {
			return fmt.Errorf("decode run report tokens: %w", err)
		}
	}
	fmt.Printf("Run %s\nStatus: %s\n", r.RunID, r.Status)
	if r.Error != "" {
		fmt.Printf("Error: %s\n", r.Error)
	}
	fmt.Printf("Turns: %d | tokens: %d | cost: $%.4f\nFinal output: %s (%s), candidates=%d\n", r.Turns, tokens, r.Cost, r.Result.Selection, r.Result.Rule, r.Result.Candidates)
	if r.Result.Structured != "" {
		fmt.Printf("Structured result: %s (%s) diagnostics=%s\n", r.Result.Structured, r.Result.Method, strings.Join(r.Result.Diagnostics, ","))
	}
	fmt.Printf("Tools: project-owned=%d external=%d | model: requested=%s actual=%s fallbacks=%d\n", r.Project, r.External, r.Requested, r.Actual, r.Fallbacks)
	fmt.Printf("Efficiency: recognized command retries=%d files reread=%d longest event gap=%s\n", r.Repeated, r.Rereads, (time.Duration(r.Longest) * time.Millisecond).Round(time.Millisecond))
	for _, tool := range r.Tools {
		fmt.Printf("  %s: calls=%d returned=%d success=%d failed=%d unknown=%d unresolved=%d\n", tool.Name, tool.Calls, tool.Returned, tool.Successes, tool.Failures, tool.UnknownOutcome, tool.Unresolved)
	}
	keys := make([]string, 0, len(r.Events))
	for key := range r.Events {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	fmt.Print("Events:")
	for _, key := range keys {
		fmt.Printf(" %s=%d", key, r.Events[key])
	}
	fmt.Println()
	fmt.Printf("Diff: files=%d bytes=%d (%s)\n", r.Diff.Files, r.Diff.Bytes, r.Diff.Available.State)
	fmt.Printf("Receipts: %s (%d)\n", r.Receipts.State, r.ReceiptCount)
	support.NextSteps(fmt.Sprintf("agent-manager run result %s", id), fmt.Sprintf("agent-manager run events %s", id), fmt.Sprintf("agent-manager run tools %s --failed", id), fmt.Sprintf("agent-manager run diff %s", id), fmt.Sprintf("agent-manager run receipts %s", id))
	return nil
}

func (a *App) runRecent(args []string) error {
	fs := flag.NewFlagSet("run recent", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	limit := fs.Int("limit", 10, "Maximum recent runs")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	if *limit < 1 || *limit > 100 {
		return fmt.Errorf("limit must be between 1 and 100")
	}
	_, runs, err := a.services.Runs.List(*limit, 0, "", "", "", "")
	if err != nil {
		return err
	}
	type item struct {
		ID         string         `json:"id"`
		Label      string         `json:"label,omitempty"`
		Subject    []string       `json:"subject,omitempty"`
		Durability map[string]any `json:"durability"`
		Lane       string         `json:"lane"`
	}
	items := make([]item, 0, len(runs))
	for _, run := range runs {
		if run == nil || run.GetId() == "" {
			continue
		}
		body, err := a.services.Runs.Durability(run.GetId())
		if err != nil {
			return err
		}
		var projection map[string]any
		if err := json.Unmarshal(body, &projection); err != nil {
			return fmt.Errorf("decode durability for %s: %w", run.GetId(), err)
		}
		// The work's own attribution lane comes from the projection. It is not
		// the same question as coverage, which counts the lanes of individual
		// evidence items: work with no evidence at all still has a lane.
		lane, _ := projection["lane"].(string)
		if lane == "" {
			lane = "unlinked"
		}
		items = append(items, item{ID: run.GetId(), Label: run.GetLabel(), Subject: append([]string(nil), run.GetSubject()...), Durability: projection, Lane: lane})
	}
	out := map[string]any{"sampleSize": len(items), "items": items}
	if *jsonOutput {
		encoded, _ := json.Marshal(out)
		cliutil.PrintJSON(encoded)
		return nil
	}
	fmt.Printf("Recent work (sample size=%d)\n", len(items))
	for _, current := range items {
		verdict, _ := current.Durability["verdict"].(string)
		if reason, _ := current.Durability["unknownReason"].(string); reason != "" {
			verdict += "(" + reason + ")"
		}
		if state, _ := current.Durability["boundaryState"].(string); state != "" {
			verdict = state
		}
		fmt.Printf("- %s %s lane=%s verdict=%s subject=%s\n", current.ID, current.Label, current.Lane, verdict, strings.Join(current.Subject, ","))
	}
	return nil
}

func (a *App) runStats(args []string) error {
	fs := flag.NewFlagSet("run stats", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	profile := fs.String("profile", "", "Profile UUID")
	since := fs.String("since", "", "RFC3339 lower time bound (default: 24h before --until)")
	until := fs.String("until", "", "RFC3339 exclusive upper time bound (default: now)")
	tagPrefix := fs.String("tag-prefix", "", "Run tag prefix")
	runner := fs.String("runner", "", "Runner type")
	model := fs.String("model", "", "Model")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	window, err := runStatsWindow(*since, *until, time.Now().UTC())
	if err != nil {
		return err
	}
	filter := &measurepb.InvocationFilter{ProfileId: *profile, TagPrefix: *tagPrefix, RunnerType: *runner, Model: *model}
	report, err := a.services.Runs.Stats(filter, window)
	if err != nil {
		return err
	}
	if *jsonOutput {
		body, err := json.Marshal(report.Raw)
		if err != nil {
			return err
		}
		cliutil.PrintJSON(body)
		return nil
	}
	printRunStats(report, window.GetCustom().GetFrom().AsTime(), window.GetCustom().GetTo().AsTime())
	support.NextSteps("agent-manager run stats --json", "agent-manager measures cohort-report --cohort <name>", "agent-manager run report <run-id>")
	return nil
}

// runStatsWindow resolves the CLI bounds to one explicit [from, to) range.
func runStatsWindow(since, until string, now time.Time) (*sharedmeasurepb.TimeWindow, error) {
	to := now
	if until != "" {
		parsed, err := time.Parse(time.RFC3339, until)
		if err != nil {
			return nil, fmt.Errorf("--until must be RFC3339: %w", err)
		}
		to = parsed
	}
	from := to.Add(-24 * time.Hour)
	if since != "" {
		parsed, err := time.Parse(time.RFC3339, since)
		if err != nil {
			return nil, fmt.Errorf("--since must be RFC3339: %w", err)
		}
		from = parsed
	}
	if !from.Before(to) {
		return nil, fmt.Errorf("--since must be before --until")
	}
	return &sharedmeasurepb.TimeWindow{Window: &sharedmeasurepb.TimeWindow_Custom{Custom: &sharedmeasurepb.CustomRange{From: timestamppb.New(from), To: timestamppb.New(to)}}}, nil
}

func printRunStats(report *RunStatsReport, from, to time.Time) {
	fmt.Printf("Run stats %s .. %s\n", from.Format(time.RFC3339), to.Format(time.RFC3339))
	var total int64
	statuses := make([]string, 0, len(report.Status.GetRows()))
	for _, row := range report.Status.GetRows() {
		total += row.GetCount()
		statuses = append(statuses, fmt.Sprintf("%s=%d", row.GetStatus(), row.GetCount()))
	}
	fmt.Printf("Runs: total=%d %s\n", total, strings.Join(statuses, " "))
	fmt.Printf("Success: rate=%.3f (%d/%d terminal)\n", report.Success.GetRate(), report.Success.GetSuccessfulRuns(), report.Success.GetTerminalRuns())
	d := report.Duration
	fmt.Printf("Duration: n=%d avg=%s p50=%s p95=%s p99=%s min=%s max=%s\n", d.GetCount(), formatMS(d.GetAverageDurationMs()), formatMS(d.GetP50DurationMs()), formatMS(d.GetP95DurationMs()), formatMS(d.GetP99DurationMs()), formatMS(float64(d.GetMinDurationMs())), formatMS(float64(d.GetMaxDurationMs())))
	c := report.Cost
	fmt.Printf("Cost: total=$%.4f avg=$%.4f runs=%d\n", c.GetTotalCostUsd(), c.GetAverageCostUsd(), c.GetTotalRuns())
	fmt.Printf("Tokens: total=%d input=%d output=%d cache_read=%d cache_creation=%d unpriced=%d\n", c.GetTotalTokens(), c.GetInputTokens(), c.GetOutputTokens(), c.GetCacheReadTokens(), c.GetCacheCreationTokens(), c.GetUnpricedTokenCount())
	printBreakdown("Runners", report.Runners.GetRows())
	printBreakdown("Profiles", report.Profiles.GetRows())
	printBreakdown("Models", report.Models.GetRows())
	if rows := report.Tools.GetRows(); len(rows) > 0 {
		fmt.Println("Tools:")
		for _, row := range rows {
			fmt.Printf("  %-28s calls=%d ok=%d failed=%d tokens=%d\n", row.GetToolName(), row.GetCallCount(), row.GetSuccessCount(), row.GetFailedCount(), row.GetTotalTokens())
		}
	}
	for _, validity := range []*measurepb.MeasureValidity{report.Status.GetValidity(), report.Success.GetValidity(), report.Duration.GetValidity(), report.Cost.GetValidity()} {
		if validity != nil && validity.GetState() != "" && validity.GetState() != "meaningful" {
			fmt.Printf("Validity: %s (%s)\n", validity.GetState(), validity.GetReason())
			break
		}
	}
}

func printBreakdown(title string, rows []*measurepb.RunBreakdownRow) {
	if len(rows) == 0 {
		return
	}
	fmt.Printf("%s:\n", title)
	for _, row := range rows {
		fmt.Printf("  %-28s runs=%d ok=%d failed=%d cost=$%.4f tokens=%d avg=%s\n", row.GetValue(), row.GetRunCount(), row.GetSuccessCount(), row.GetFailedCount(), row.GetTotalCostUsd(), row.GetTotalTokens(), formatMS(row.GetAverageDurationMs()))
	}
}

func formatMS(ms float64) string {
	return (time.Duration(ms) * time.Millisecond).Round(time.Millisecond).String()
}

func (a *App) runResult(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: agent-manager run result <id>")
	}
	_, run, err := a.services.Runs.Get(args[0])
	if err != nil {
		return err
	}
	if run == nil || run.Result == nil {
		fmt.Println("No final result is available")
		support.NextSteps(fmt.Sprintf("agent-manager run messages %s", args[0]), fmt.Sprintf("agent-manager run report %s", args[0]))
		return nil
	}
	if run.EndedAt == nil && !isTerminalRunStatus(run.Status) {
		fmt.Printf("Current run status: %s; current execution has no terminal result yet.\n", formatEnumValue(run.Status, "RUN_STATUS_", " "))
		fmt.Println("The stored result below is retained from an earlier invocation and does not report the current execution.")
	}
	selection := run.Result.Selection
	nextMessageCommand := fmt.Sprintf("agent-manager run messages %s", args[0])
	fmt.Printf("Selection: %s (%s) candidates=%d\n", selection.GetStatus(), selection.GetRule(), len(run.Result.Candidates))
	if selection.GetStatus() == domainpb.FinalOutputSelectionStatus_FINAL_OUTPUT_SELECTION_STATUS_SELECTED && selection.GetSelectedCandidateId() != "" {
		fmt.Printf("Selected candidate: %s\n", selection.GetSelectedCandidateId())
		for _, candidate := range run.Result.Candidates {
			if candidate.GetId() == selection.GetSelectedCandidateId() && candidate.GetSequence() >= 0 {
				nextMessageCommand = fmt.Sprintf("agent-manager run messages %s --range %d:%d", args[0], candidate.GetSequence(), candidate.GetSequence())
				break
			}
		}
	}
	if run.Result.GetTerminalReason() != "" {
		fmt.Printf("Terminal reason: %s\n", run.Result.GetTerminalReason())
	}
	for _, candidate := range run.Result.Candidates {
		selected := ""
		if candidate.GetId() == selection.GetSelectedCandidateId() {
			selected = " selected"
		}
		fmt.Printf("Candidate %s%s: sequence=%d terminal=%t completion=%s evidence-tier=%d\n", candidate.GetId(), selected, candidate.GetSequence(), candidate.GetTerminal(), candidate.GetCompletionReason(), candidate.GetEvidenceTier())
	}
	if run.Result.Structured != nil {
		structured := run.Result.Structured
		fmt.Printf("Structured result: %s (%s)\n", structured.GetStatus(), structured.GetMethod())
		for _, diagnostic := range structured.GetDiagnostics() {
			fmt.Printf("Diagnostic %s", diagnostic.GetCode())
			if diagnostic.GetPath() != "" {
				fmt.Printf(" path=%s", diagnostic.GetPath())
			}
			fmt.Printf(": %s\n", diagnostic.GetMessage())
		}
	}
	support.NextSteps(nextMessageCommand, fmt.Sprintf("agent-manager run report %s", args[0]))
	return nil
}

func (a *App) runTools(args []string) error {
	fs := flag.NewFlagSet("run tools", flag.ContinueOnError)
	failed := fs.Bool("failed", false, "Show failed tool events only")
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return fmt.Errorf("usage: agent-manager run tools <id> [--failed]")
	}
	id := args[0]
	if err := cliutil.ParseInterspersed(fs, args[1:]); err != nil {
		return err
	}
	reportBody, err := a.services.Runs.GetReport(id)
	if err != nil {
		return err
	}
	var report struct {
		Project  int `json:"project_owned_tool_calls"`
		External int `json:"external_tool_calls"`
		Tools    []struct {
			Name           string `json:"name"`
			Calls          int    `json:"calls"`
			Successes      int    `json:"successes"`
			Failures       int    `json:"failures"`
			Unresolved     int    `json:"unresolved"`
			Returned       int    `json:"returned"`
			UnknownOutcome int    `json:"unknown_outcome"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(reportBody, &report); err != nil {
		return fmt.Errorf("decode run tool summary: %w", err)
	}
	fmt.Printf("Tool calls: project-owned=%d external=%d\n", report.Project, report.External)
	for _, tool := range report.Tools {
		fmt.Printf("%s: calls=%d returned=%d success=%d failed=%d unknown=%d unresolved=%d\n", tool.Name, tool.Calls, tool.Returned, tool.Successes, tool.Failures, tool.UnknownOutcome, tool.Unresolved)
	}
	body, events, err := a.services.Runs.GetEvents(id, 0, nil)
	if err != nil {
		return err
	}
	count := 0
	for _, event := range events {
		if event.GetEventType() != domainpb.RunEventType_RUN_EVENT_TYPE_TOOL_CALL && event.GetEventType() != domainpb.RunEventType_RUN_EVENT_TYPE_TOOL_RESULT {
			continue
		}
		result := event.GetToolResult()
		if *failed && toolResultOutcome(result) != "failure" {
			continue
		}
		if result != nil {
			fmt.Printf("%d result %s outcome=%s error=%s\n", event.GetSequence(), result.GetToolName(), toolResultOutcome(result), result.GetError())
		} else if call := event.GetToolCall(); call != nil {
			fmt.Printf("%d call %s\n", event.GetSequence(), call.GetToolName())
		}
		count++
	}
	if count == 0 && len(events) == 0 {
		fmt.Print(string(body))
	}
	support.NextSteps(fmt.Sprintf("agent-manager run events %s", id), fmt.Sprintf("agent-manager run report %s", id))
	return nil
}

func (a *App) runMessages(args []string) error {
	fs := flag.NewFlagSet("run messages", flag.ContinueOnError)
	all := fs.Bool("all", false, "Include evidence-only messages")
	rangeValue := fs.String("range", "", "Inclusive event sequence range, for example 10:25")
	grep := fs.String("grep", "", "Only messages containing text")
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return fmt.Errorf("usage: agent-manager run messages <id> [--all] [--grep text]")
	}
	id := args[0]
	if err := cliutil.ParseInterspersed(fs, args[1:]); err != nil {
		return err
	}
	start, end, err := parseSequenceRange(*rangeValue)
	if err != nil {
		return err
	}
	var events []*domainpb.RunEvent
	if strings.TrimSpace(*rangeValue) == "" {
		_, events, err = a.services.Runs.GetEvents(id, 0, nil)
	} else {
		events, err = a.messageEventsInRange(id, start, end)
	}
	if err != nil {
		return err
	}
	_, run, err := a.services.Runs.Get(id)
	if err != nil {
		return err
	}
	selectedSequence := int64(-1)
	if run != nil && run.Result != nil {
		selectedID := run.Result.Selection.GetSelectedCandidateId()
		for _, candidate := range run.Result.Candidates {
			if candidate.GetId() == selectedID {
				selectedSequence = candidate.GetSequence()
				break
			}
		}
	}
	total, shown := 0, 0
	for _, event := range events {
		if event.GetSequence() < start || (end > 0 && event.GetSequence() > end) {
			continue
		}
		if msg := event.GetMessage(); msg != nil {
			total++
			if !(*all || !msg.GetEvidenceOnly()) || (*grep != "" && !strings.Contains(strings.ToLower(msg.GetContent()), strings.ToLower(*grep))) {
				continue
			}
			label := ""
			if event.GetSequence() == selectedSequence {
				label = " [selected final output]"
			}
			fmt.Printf("%d %s%s: %s\n", event.GetSequence(), msg.GetRole(), label, msg.GetContent())
			shown++
		}
	}
	fmt.Printf("Messages: shown=%d total=%d\n", shown, total)
	support.NextSteps(fmt.Sprintf("agent-manager run result %s", id), fmt.Sprintf("agent-manager run report %s", id))
	return nil
}

func (a *App) messageEventsInRange(id string, start, end int64) ([]*domainpb.RunEvent, error) {
	const maxPageSize = 1000
	var after *int64
	if start > 0 {
		firstSequence := start - 1
		after = &firstSequence
	}
	cursor := int64(-1)
	if after != nil {
		cursor = *after
	}
	var events []*domainpb.RunEvent
	for {
		// Bound transport by the remaining sequence interval as well as the
		// API page ceiling. Compare before adding one so [0, MaxInt64] cannot
		// overflow while calculating its inclusive width.
		nextSequence := cursor + 1
		if cursor < start-1 {
			nextSequence = start
		}
		remainingWidth := end - nextSequence
		pageSize := maxPageSize
		if remainingWidth < maxPageSize-1 {
			pageSize = int(remainingWidth + 1)
		}
		page, err := a.services.Runs.GetEventPage(id, pageSize, after)
		if err != nil {
			return nil, err
		}
		if len(page.GetEvents()) == 0 {
			if page.GetHasMore() {
				return nil, fmt.Errorf("run event page is empty while more events are available")
			}
			return events, nil
		}
		lastSequence := cursor
		for _, event := range page.GetEvents() {
			if event.GetSequence() <= lastSequence {
				return nil, fmt.Errorf("run event page is not strictly ordered after sequence %d", lastSequence)
			}
			lastSequence = event.GetSequence()
			if event.GetSequence() >= start && event.GetSequence() <= end {
				events = append(events, event)
			}
		}
		if lastSequence >= end || !page.GetHasMore() {
			return events, nil
		}
		nextAfter := lastSequence
		after = &nextAfter
		cursor = lastSequence
	}
}

func parseSequenceRange(value string) (int64, int64, error) {
	if strings.TrimSpace(value) == "" {
		return 0, 0, nil
	}
	parts := strings.Split(value, ":")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("--range must be start:end")
	}
	start, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || start < 0 {
		return 0, 0, fmt.Errorf("--range start must be a non-negative sequence")
	}
	end, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || end < start {
		return 0, 0, fmt.Errorf("--range end must be at least start")
	}
	return start, end, nil
}

func (a *App) runReceipts(args []string) error {
	fs := flag.NewFlagSet("run receipts", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return fmt.Errorf("usage: agent-manager run receipts <id> [--json]")
	}
	id := args[0]
	if err := cliutil.ParseInterspersed(fs, args[1:]); err != nil {
		return err
	}
	body, err := a.services.Runs.GetReceipts(id)
	if err != nil {
		return err
	}
	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}
	var receipt struct {
		Status       string            `json:"status"`
		Message      string            `json:"message"`
		Observations []json.RawMessage `json:"observations"`
	}
	if err := json.Unmarshal(body, &receipt); err != nil {
		return fmt.Errorf("decode receipt observations: %w", err)
	}
	fmt.Printf("Receipt observations: %s (%d)\n", receipt.Status, len(receipt.Observations))
	if receipt.Message != "" {
		fmt.Printf("Detail: %s\n", receipt.Message)
	}
	support.NextSteps(fmt.Sprintf("agent-manager run report %s", id), fmt.Sprintf("agent-manager run events %s", id))
	return nil
}

// toolResultOutcome projects the public wire facts without treating an observed
// return with explicitly unknown outcome as a failed invocation.
func toolResultOutcome(result *domainpb.ToolResultEventData) string {
	if result == nil {
		return "unknown"
	}
	if result.ExitCode != nil {
		if *result.ExitCode == 0 {
			return "success"
		}
		return "failure"
	}
	if result.OutcomeKnown != nil && !*result.OutcomeKnown {
		return "unknown"
	}
	if result.Success && strings.TrimSpace(result.Error) == "" {
		return "success"
	}
	return "failure"
}
