package orchestration

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

func TestInvestigationEventEvidenceFormattingAndStatistics(t *testing.T) {
	events := []*domain.RunEvent{
		{EventType: domain.EventTypeToolCall, Data: &domain.ToolCallEventData{ToolName: "bash", Input: map[string]interface{}{"command": "go test ./..."}}},
		{EventType: domain.EventTypeToolResult, Data: &domain.ToolResultEventData{ToolName: "bash", Success: true, Output: "ok"}},
		{EventType: domain.EventTypeToolResult, Data: &domain.ToolResultEventData{ToolName: "bash", Success: false, Error: "failed"}},
		{EventType: domain.EventTypeStatus, Data: &domain.StatusEventData{OldStatus: "running", NewStatus: "failed", Reason: "runner exit"}},
		{EventType: domain.EventTypeError, Data: &domain.ErrorEventData{Code: "RUNNER", Message: "boom"}},
		{EventType: domain.EventTypeMetric, Data: &domain.CostEventData{TotalCostUSD: 1.5, InputTokens: 10, OutputTokens: 20}},
	}
	stats := computeEventStats(events)
	if stats.toolCalls != 1 || stats.toolSuccesses != 1 || stats.toolFailures != 1 || stats.statusChanges != 1 || stats.errors != 1 || stats.totalCostUSD != 1.5 || stats.totalTokens != 30 {
		t.Fatalf("stats=%+v", stats)
	}
	for _, event := range events {
		if summary := formatEventSummary(event); summary == "" || summary == "(no data)" {
			t.Fatalf("event=%T summary=%q", event.Data, summary)
		}
	}
	if got := formatEventSummary(&domain.RunEvent{}); got != "(no data)" {
		t.Fatalf("empty=%q", got)
	}
	if got := summarizeToolInput("tool", map[string]interface{}{"path": strings.Repeat("a", 101)}); !strings.HasSuffix(got, "...") {
		t.Fatalf("input=%q", got)
	}
	if summarizeToolInput("tool", nil) != "(no input)" || summarizeToolInput("tool", map[string]interface{}{}) != "(empty)" {
		t.Fatal("tool input fallbacks incorrect")
	}
	if formatDuration(-time.Second) != "00:00" || formatDuration(65*time.Second) != "01:05" || formatDuration(3661*time.Second) != "1:01:01" {
		t.Fatal("duration format mismatch")
	}
}

func TestFormatEventSummaryCoversReadableOperatorPayloads(t *testing.T) {
	long := strings.Repeat("x", 121)
	cases := []struct {
		event    *domain.RunEvent
		expected string
	}{
		{&domain.RunEvent{Data: &domain.LogEventData{Level: "warn", Message: long}}, "[warn] " + strings.Repeat("x", 120) + "..."},
		{&domain.RunEvent{Data: &domain.MessageEventData{Role: "assistant", Content: long}}, "assistant: " + strings.Repeat("x", 120) + "..."},
		{&domain.RunEvent{Data: &domain.ToolResultEventData{ToolName: "bash", Success: false, Error: strings.Repeat("e", 101)}}, "FAILED — " + strings.Repeat("e", 100) + "..."},
		{&domain.RunEvent{Data: &domain.ToolResultEventData{ToolName: "bash", Success: true, Output: strings.Repeat("o", 81)}}, "OK (81 chars)"},
		{&domain.RunEvent{Data: &domain.ToolResultEventData{ToolName: "bash", Success: true, Output: "ok"}}, "Result `bash`: ok"},
		{&domain.RunEvent{Data: &domain.ToolResultEventData{ToolName: "bash", Success: true}}, "Result `bash`: OK"},
		{&domain.RunEvent{Data: &domain.StatusEventData{OldStatus: "queued", NewStatus: "running"}}, "queued → running"},
		{&domain.RunEvent{Data: &domain.ErrorEventData{Message: "uncoded"}}, "uncoded"},
		{&domain.RunEvent{Data: &domain.RateLimitEventData{LimitType: "requests", Message: "slow down"}}, "Rate limit (requests): slow down"},
		{&domain.RunEvent{Data: &domain.ArtifactEventData{Type: "diff", Path: "patch.diff"}}, "Artifact [diff]: patch.diff"},
		{&domain.RunEvent{Data: &domain.ProgressEventData{PercentComplete: 50, CurrentAction: "testing"}}, "Progress: 50% — testing"},
		{&domain.RunEvent{Data: &domain.ProgressEventData{PercentComplete: 50, Phase: "verify"}}, "Progress: 50% (phase=verify)"},
		{&domain.RunEvent{Data: &domain.MessageDeletedEventData{TargetEventID: "evt-1"}}, "Message deleted: evt-1"},
		{&domain.RunEvent{Data: &domain.TypedEventData{Type: "custom", Body: []byte(`{"kind":"unknown"}`)}}, `{"kind":"unknown"}`},
	}
	for _, tc := range cases {
		if got := formatEventSummary(tc.event); !strings.Contains(got, tc.expected) {
			t.Fatalf("event=%T got=%q expected to contain %q", tc.event.Data, got, tc.expected)
		}
	}
	if got := summarizeToolInput("tool", map[string]interface{}{"other": "value"}); got != "other=value" {
		t.Fatalf("fallback input=%q", got)
	}
}

func TestInvestigationAttachmentHelpersExposeStableEvidence(t *testing.T) {
	profile := &domain.AgentProfile{ID: uuid.New(), Name: "reviewer", ProfileKey: "reviewer", RoleRef: "code.default", Description: "reviews", AllowedTools: []string{"bash"}}
	attachment, ok := buildAgentSetupAttachment(profile, t.TempDir(), shortID(profile.ID))
	if !ok || attachment.Key == "" || !strings.Contains(attachment.Content, "reviewer") || !strings.Contains(attachment.Content, "No agent directory") {
		t.Fatalf("attachment=%+v", attachment)
	}
	if raw, err := marshalJSON(map[string]string{"status": "ok"}); err != nil || !strings.Contains(raw, "status") {
		t.Fatalf("json=%q err=%v", raw, err)
	}
	if got := shortID(profile.ID); len(got) != 8 {
		t.Fatalf("short=%q", got)
	}
}

func TestBuildAgentSetupAttachmentDiscoversPromptAndTeamFiles(t *testing.T) {
	root := t.TempDir()
	profile := &domain.AgentProfile{
		Name:         "reviewer",
		ProfileKey:   "reviewer",
		RoleRef:      "code.default",
		Description:  "reviews changes",
		AllowedTools: []string{"bash", "git"},
		DeniedTools:  []string{"network"},
	}
	store := filepath.Join(root, "scenarios", "prompt-manager", "store")
	paths := []string{
		filepath.Join(store, "agents", "reviewer", "agent.json"),
		filepath.Join(store, "agents", "reviewer", "SOUL.md"),
		filepath.Join(store, "relations", "team-member", "platform__reviewer.json"),
		filepath.Join(store, "teams", "platform", "team.json"),
		filepath.Join(store, "teams", "platform", "shared", "TEAM.md"),
		filepath.Join(store, "teams", "platform", "members", "reviewer", "heartbeat.json"),
		filepath.Join(store, "teams", "platform", "members", "reviewer", "logs", "first.log"),
	}
	for _, path := range paths {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("fixture"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	attachment, ok := buildAgentSetupAttachment(profile, root, "reviewer")
	if !ok || attachment.Key != "agent-setup-reviewer" {
		t.Fatalf("attachment=%+v ok=%v", attachment, ok)
	}
	for _, expected := range []string{
		"**Allowed Tools**: bash, git", "**Denied Tools**: network", "`agent.json`", "`SOUL.md`",
		"### Team: `platform`", "`team.json`", "`shared/TEAM.md`", "`heartbeat.json`", "1 execution log(s)",
		"Read these files to understand", "platform__reviewer.json",
	} {
		if !strings.Contains(attachment.Content, expected) {
			t.Fatalf("attachment missing %q: %s", expected, attachment.Content)
		}
	}
}

func TestBuildHistoricalContextSeparatesComparableRunsFromMetaRuns(t *testing.T) {
	ctx := context.Background()
	launcher := newFakeRunLauncher()
	o, repos := newRelayOrchestrator(t, launcher)
	now := time.Date(2026, time.March, 4, 5, 6, 7, 0, time.UTC)
	profileID := uuid.New()
	taskID := uuid.New()
	if err := repos.Profiles.Create(ctx, &domain.AgentProfile{ID: profileID, Name: "reviewer", ProfileKey: "reviewer", RoleRef: "code.default", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("profile: %v", err)
	}
	if err := repos.Tasks.Create(ctx, &domain.Task{ID: taskID, Title: "Review", Description: "Review", ScopePath: ".", ProjectRoot: "/project", Status: domain.TaskStatusQueued, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("task: %v", err)
	}
	create := func(id uuid.UUID, tag string, status domain.RunStatus, errorMsg string, offset time.Duration) *domain.Run {
		started := now.Add(offset)
		ended := started.Add(time.Minute)
		run := &domain.Run{ID: id, TaskID: taskID, AgentProfileID: &profileID, Tag: tag, Status: status, Phase: domain.RunPhaseCompleted, StartedAt: &started, EndedAt: &ended, ErrorMsg: errorMsg, CreatedAt: started, UpdatedAt: ended}
		if err := repos.Runs.Create(ctx, run); err != nil {
			t.Fatalf("create %s: %v", tag, err)
		}
		return run
	}
	current := create(uuid.New(), "current", domain.RunStatusFailed, "current failure", 0)
	create(uuid.New(), "successful", domain.RunStatusComplete, "", time.Minute)
	create(uuid.New(), "failed", domain.RunStatusFailed, strings.Repeat("failure ", 12), 2*time.Minute)
	create(uuid.New(), "agent-manager-investigation", domain.RunStatusFailed, "meta run", 3*time.Minute)

	attachment, ok := o.buildHistoricalContext(ctx, current, "current")
	if !ok {
		t.Fatal("expected historical attachment")
	}
	for _, expected := range []string{"### Recent Runs", "successful", "failed", "1 of last 2 runs succeeded, 1 failed", "Compare successful vs failed"} {
		if !strings.Contains(attachment.Content, expected) {
			t.Fatalf("history missing %q: %s", expected, attachment.Content)
		}
	}
	if strings.Contains(attachment.Content, "agent-manager-investigation") || strings.Contains(attachment.Content, "current failure") {
		t.Fatalf("history included excluded run: %s", attachment.Content)
	}
	if _, ok := o.buildHistoricalContext(ctx, &domain.Run{}, "none"); ok {
		t.Fatal("run without a profile should not have historical context")
	}
}

func TestRunTimelineBuildsCuratedFailureAndCostEvidence(t *testing.T) {
	run := &domain.Run{ID: uuid.New(), Status: domain.RunStatusFailed}
	empty := buildRunTimeline(nil, run, "run")
	if empty.Summary != "No events recorded" || !strings.Contains(empty.Content, "No events") {
		t.Fatalf("empty=%+v", empty)
	}
	now := time.Now().UTC()
	events := []*domain.RunEvent{
		{EventType: domain.EventTypeToolCall, Timestamp: now, Data: &domain.ToolCallEventData{ToolName: "bash", Input: map[string]interface{}{"command": "test"}}},
		{EventType: domain.EventTypeError, Timestamp: now.Add(2 * time.Second), Data: &domain.ErrorEventData{Code: "FAIL", Message: "failure"}},
		{EventType: domain.EventTypeMetric, Timestamp: now.Add(3 * time.Second), Data: &domain.CostEventData{TotalCostUSD: 2, InputTokens: 4, OutputTokens: 5}},
	}
	timeline := buildRunTimeline(events, run, "run")
	for _, expected := range []string{"Failure Points", "Last error occurred at event #2", "Tool Calls", "Total Cost", "Total Tokens", "run events " + run.ID.String()} {
		if !strings.Contains(timeline.Content, expected) {
			t.Fatalf("timeline missing %q: %s", expected, timeline.Content)
		}
	}
	if timeline.Summary != "3 events, 1 errors, 1 tool calls" {
		t.Fatalf("summary=%q", timeline.Summary)
	}
}

func TestRunOverviewBuildsTriageReadyExecutionRecord(t *testing.T) {
	started := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	ended := started.Add(2*time.Minute + 3*time.Second)
	heartbeat := ended.Add(-10 * time.Second)
	exitCode := 17
	taskID := uuid.New()
	run := &domain.Run{
		ID:              uuid.New(),
		TaskID:          taskID,
		Tag:             "repair-production",
		Status:          domain.RunStatusFailed,
		StartedAt:       &started,
		EndedAt:         &ended,
		ExitCode:        &exitCode,
		ErrorMsg:        "runner exited unexpectedly",
		Phase:           domain.RunPhaseExecuting,
		LastHeartbeat:   &heartbeat,
		ProgressPercent: 75,
		Summary: &domain.RunSummary{
			TurnsUsed:    4,
			TokensUsed:   321,
			CostEstimate: 0.1234,
			Description:  "The repair stopped before validation.",
		},
		ChangedFiles: 2,
		ResolvedConfig: &domain.RunConfig{
			RunnerType: domain.RunnerTypeCodex,
			Model:      "gpt-5",
			RoleRef:    "code.default",
			MaxTurns:   8,
			Timeout:    5 * time.Minute,
		},
	}
	task := &domain.Task{
		ID:          taskID,
		Title:       "Repair production",
		Description: strings.Repeat("x", 501),
		ProjectRoot: "/workspace/project",
	}
	profile := &domain.AgentProfile{Name: "production repair", ProfileKey: "prod-repair", Description: "Careful repair agent"}

	overview := buildRunOverview(run, task, profile, "run123")
	for _, expected := range []string{
		"**Run ID**", "repair-production", "**Duration**: 2m3s", "**Exit Code**: 17",
		"runner exited unexpectedly", "### Task", "Repair production", "... (truncated",
		"### Execution Details", "**Heartbeat-to-End Gap**: 10s", "**Cost Estimate**: $0.1234",
		"### Runner Configuration", "**Runner**: codex", "**Model**: gpt-5", "### Agent Profile", "prod-repair",
	} {
		if !strings.Contains(overview.Content, expected) {
			t.Fatalf("overview missing %q: %s", expected, overview.Content)
		}
	}
	if overview.Key != "run-overview-run123" || overview.Summary != "Run run123: repair-production (status=failed)" {
		t.Fatalf("attachment metadata=%+v", overview)
	}

	withoutDetails := buildRunOverview(run, nil, nil, "fallback")
	if !strings.Contains(withoutDetails.Content, "**Task ID**: `"+taskID.String()+"` (details unavailable)") {
		t.Fatalf("fallback overview=%s", withoutDetails.Content)
	}
}

func TestInvestigationContextMetadataAndRenderingPreserveOperatorEvidence(t *testing.T) {
	runIDs := []uuid.UUID{uuid.New(), uuid.New()}
	context := buildInvestigationContextAttachment("/workspace/project", []string{"api", "ui"}, runIDs)
	for _, expected := range []string{"/workspace/project", "`api`", "`ui`", runIDs[0].String(), "agent-manager run events <run-id>"} {
		if !strings.Contains(context.Content, expected) {
			t.Fatalf("context missing %q: %s", expected, context.Content)
		}
	}
	if context.Summary != "Scope and CLI commands for 2 run(s) under investigation" {
		t.Fatalf("summary=%q", context.Summary)
	}

	now := time.Date(2026, time.February, 3, 4, 5, 6, 0, time.FixedZone("local", -5*60*60))
	for _, tc := range []struct {
		depth    domain.InvestigationDepth
		guidance string
	}{
		{domain.InvestigationDepthQuick, "QUICK mode"},
		{domain.InvestigationDepthStandard, "STANDARD mode"},
		{domain.InvestigationDepthDeep, "DEEP mode"},
	} {
		metadata := buildInvestigationMetadataAttachment(runIDs, tc.depth, now)
		if !strings.Contains(metadata.Content, tc.guidance) || !strings.Contains(metadata.Content, "2026-02-03T09:05:06Z") {
			t.Fatalf("depth=%s metadata=%s", tc.depth, metadata.Content)
		}
	}

	rendered := renderInvestigationContext([]domain.ContextAttachment{
		{Key: "fallback", Content: "first"},
		{Label: "Named", Content: "second"},
		{Label: "ignored", Content: " \n "},
	})
	if rendered != "\n## fallback\n\nfirst\n\n## Named\n\nsecond\n" {
		t.Fatalf("rendered=%q", rendered)
	}
	truncated := renderInvestigationContext([]domain.ContextAttachment{{Label: "Large", Content: strings.Repeat("x", maxInvestigationContextBytes+1)}})
	if !strings.HasSuffix(truncated, "... (context truncated to fit the run prompt budget)\n") || len(truncated) <= maxInvestigationContextBytes {
		t.Fatalf("truncation missing or too short: len=%d", len(truncated))
	}
}
