package codecs

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// =============================================================================
// Test helpers
// =============================================================================

// mockPricingService is a configurable PricingService for codec tests.
type mockPricingService struct {
	pricing map[string]*PricingCostCalculation
}

func (m *mockPricingService) CalculateCost(ctx context.Context, req PricingCostRequest) (*PricingCostCalculation, error) {
	if calc, ok := m.pricing[req.Model]; ok {
		inputPrice := 0.00000025
		outputPrice := 0.000002
		cacheReadPrice := 0.000000025
		return &PricingCostCalculation{
			InputCostUSD:     float64(req.InputTokens) * inputPrice,
			OutputCostUSD:    float64(req.OutputTokens) * outputPrice,
			CacheReadCostUSD: float64(req.CacheReadTokens) * cacheReadPrice,
			TotalCostUSD:     float64(req.InputTokens)*inputPrice + float64(req.OutputTokens)*outputPrice + float64(req.CacheReadTokens)*cacheReadPrice,
			CostSource:       calc.CostSource,
			Provider:         calc.Provider,
			CanonicalModel:   calc.CanonicalModel,
			PricingFetchedAt: calc.PricingFetchedAt,
		}, nil
	}
	return nil, nil
}

// codexSamples holds real codex CLI --json output captured 2025-12-19.
var codexSamples = map[string]string{
	"thread.started":              `{"type":"thread.started","thread_id":"019b3906-b365-7403-b3d1-70d60f6f06c4"}`,
	"turn.started":                `{"type":"turn.started"}`,
	"reasoning":                   `{"type":"item.completed","item":{"id":"item_0","type":"reasoning","text":"thinking about the file…"}}`,
	"file_change":                 `{"type":"item.completed","item":{"id":"item_1","type":"file_change","changes":[{"path":"/tmp/test123.txt","kind":"add"}],"status":"completed"}}`,
	"file_change_multiple":        `{"type":"item.completed","item":{"id":"item_2","type":"file_change","changes":[{"path":"/tmp/file1.txt","kind":"add"},{"path":"/tmp/file2.txt","kind":"modify"},{"path":"/tmp/file3.txt","kind":"delete"}],"status":"completed"}}`,
	"agent_message":               `{"type":"item.completed","item":{"id":"item_2","type":"agent_message","text":"Created ` + "`test123.txt`" + ` containing ` + "`hello`" + `."}}`,
	"turn.completed":              `{"type":"turn.completed","usage":{"input_tokens":12810,"cached_input_tokens":12416,"output_tokens":83}}`,
	"error":                       `{"type":"error","error":{"code":"RATE_LIMIT","message":"Rate limit exceeded, please try again later"}}`,
	"data_prefix_tool_call":       `data: {"type":"item.started","item":{"id":"item_3","type":"tool_call","name":"bash","input":{"command":"ls -la"}}}`,
	"command_execution_started":   `{"type":"item.started","item":{"id":"item_4","type":"command_execution","command":"/bin/bash -lc \"echo test\"","aggregated_output":"","exit_code":null,"status":"in_progress"}}`,
	"command_execution_completed": `{"type":"item.completed","item":{"id":"item_4","type":"command_execution","command":"/bin/bash -lc \"echo test\"","aggregated_output":"test\n","exit_code":0,"status":"completed"}}`,
	"command_execution_failed":    `{"type":"item.completed","item":{"id":"item_5","type":"command_execution","command":"/bin/bash -lc \"badcmd\"","aggregated_output":"bash: badcmd: command not found\n","exit_code":127,"status":"failed"}}`,
}

// codexDecodeOne returns the events parsed from a single line, allocating
// a fresh state. The state's runModel is preset so cost-event tests carry
// the expected label.
func codexDecodeOne(t *testing.T, c *Codex, line string, model string) []*domain.RunEvent {
	t.Helper()
	state := c.NewState().(*codexState)
	state.runModel = model
	events, err := c.DecodeStreamLine(state, uuid.New(), line)
	if err != nil {
		t.Fatalf("DecodeStreamLine err: %v", err)
	}
	return events
}

// =============================================================================
// BuildArgs / BuildEnv / Continue
// =============================================================================

func TestCodex_BuildArgs_DefaultsAndStdinSentinel(t *testing.T) {
	c := NewCodexForTest()
	state := c.NewState()
	args := c.BuildArgs(state, runner.ExecuteRequest{
		RunID: uuid.New(),
		ResolvedConfig: &domain.RunConfig{
			RunnerType:    domain.RunnerTypeCodex,
			Model:         "gpt-5.1-codex-mini",
			NetworkAccess: domain.NetworkAccessNone,
		},
		WorkingDir: "/tmp",
	})
	if args[0] != "exec" || args[1] != "--json" || args[2] != "--skip-git-repo-check" {
		t.Errorf("expected exec --json --skip-git-repo-check first, got %v", args[:3])
	}
	if args[len(args)-1] != "-" {
		t.Errorf("expected stdin sentinel '-' as last arg, got %q", args[len(args)-1])
	}
	hasSandbox, hasModel, hasC := false, false, false
	for i, a := range args {
		switch a {
		case "--sandbox":
			if i+1 < len(args) && args[i+1] == "workspace-write" {
				hasSandbox = true
			}
		case "--full-auto":
			t.Errorf("did not expect deprecated --full-auto, got %v", args)
		case "-m":
			if i+1 < len(args) && args[i+1] == "gpt-5.1-codex-mini" {
				hasModel = true
			}
		case "-C":
			if i+1 < len(args) && args[i+1] == "/tmp" {
				hasC = true
			}
		}
	}
	if !hasSandbox {
		t.Errorf("expected --sandbox workspace-write for NetworkAccessNone, got %v", args)
	}
	if !hasModel {
		t.Error("expected -m gpt-5.1-codex-mini")
	}
	if !hasC {
		t.Error("expected -C /tmp")
	}
	// State should now carry the runModel.
	if cs, ok := state.(*codexState); !ok || cs.runModel != "gpt-5.1-codex-mini" {
		t.Errorf("state.runModel=%v want gpt-5.1-codex-mini", state)
	}
}

func TestCodex_BuildArgs_NetworkAccessLocalhost(t *testing.T) {
	c := NewCodexForTest()
	args := c.BuildArgs(c.NewState(), runner.ExecuteRequest{
		RunID: uuid.New(),
		ResolvedConfig: &domain.RunConfig{
			RunnerType:    domain.RunnerTypeCodex,
			NetworkAccess: domain.NetworkAccessLocalhost,
		},
	})
	hasBypass, hasSandbox := false, false
	for _, a := range args {
		switch a {
		case "--dangerously-bypass-approvals-and-sandbox":
			hasBypass = true
		case "--sandbox":
			hasSandbox = true
		}
	}
	if !hasBypass || hasSandbox {
		t.Errorf("expected --dangerously-bypass-approvals-and-sandbox without --sandbox for localhost, got %v", args)
	}
}

func TestCodex_BuildContinueArgs_IncludesSessionAndPrompt(t *testing.T) {
	c := NewCodexForTest()
	args := c.BuildContinueArgs(c.NewState(), runner.ContinueRequest{
		RunID:     uuid.New(),
		SessionID: "thread-abc",
		Prompt:    "follow up message",
	})
	want := []string{"exec", "resume", "--json", "--skip-git-repo-check", "--dangerously-bypass-approvals-and-sandbox", "thread-abc", "follow up message"}
	if len(args) != len(want) {
		t.Fatalf("len=%d want=%d args=%v", len(args), len(want), args)
	}
	for i := range want {
		if args[i] != want[i] {
			t.Errorf("args[%d]=%q want %q", i, args[i], want[i])
		}
	}
}

func TestCodex_BuildContinueArgs_NoPromptOmitsArg(t *testing.T) {
	c := NewCodexForTest()
	args := c.BuildContinueArgs(c.NewState(), runner.ContinueRequest{
		RunID: uuid.New(), SessionID: "thread-xyz", Prompt: "",
	})
	for _, a := range args {
		if a == "" {
			t.Errorf("empty prompt should not appear as arg, got %v", args)
		}
	}
	if args[len(args)-1] != "thread-xyz" {
		t.Errorf("last arg=%q want thread-xyz", args[len(args)-1])
	}
}

func TestCodex_BuildContinueArgs_DoesNotEmitUnsupportedWorkingDirFlag(t *testing.T) {
	args := NewCodexForTest().BuildContinueArgs(NewCodexForTest().NewState(), runner.ContinueRequest{
		RunID: uuid.New(), SessionID: "thread-xyz", WorkingDir: "/workspace", Prompt: "continue",
	})
	for _, arg := range args {
		if arg == "-C" || arg == "/workspace" {
			t.Fatalf("codex exec resume does not accept a working-directory argument: %v", args)
		}
	}
}

func TestCodex_BuildEnv_NonInteractive(t *testing.T) {
	c := NewCodexForTest()
	env := c.BuildEnv("codex-tag-1", nil)
	hasTag, hasNonInt := false, false
	for _, e := range env {
		if e == "CODEX_AGENT_TAG=codex-tag-1" {
			hasTag = true
		}
		if e == "CODEX_NON_INTERACTIVE=true" {
			hasNonInt = true
		}
	}
	if !hasTag {
		t.Error("missing CODEX_AGENT_TAG")
	}
	if !hasNonInt {
		t.Error("missing CODEX_NON_INTERACTIVE=true")
	}
}

func TestCodex_ContinueTag(t *testing.T) {
	c := NewCodexForTest()
	id := uuid.New()
	tag := c.ContinueTag(runner.ContinueRequest{RunID: id})
	want := "codex-continue-" + id.String()[:8]
	if tag != want {
		t.Errorf("tag=%q want %q", tag, want)
	}
}

// =============================================================================
// Stream parsing
// =============================================================================

func TestCodex_DecodeStreamLine_ThreadStarted(t *testing.T) {
	c := NewCodexForTest()
	state := c.NewState()
	events, _ := c.DecodeStreamLine(state, uuid.New(), codexSamples["thread.started"])
	if len(events) != 1 {
		t.Fatalf("got %d events", len(events))
	}
	logData := events[0].Data.(*domain.LogEventData)
	if logData.Level != "debug" {
		t.Errorf("level=%s", logData.Level)
	}
	cs := state.(*codexState)
	if cs.threadID != "019b3906-b365-7403-b3d1-70d60f6f06c4" {
		t.Errorf("threadID=%s", cs.threadID)
	}
}

func TestCodex_DecodeStreamLine_TurnStarted(t *testing.T) {
	events := codexDecodeOne(t, NewCodexForTest(), codexSamples["turn.started"], "")
	if len(events) != 1 {
		t.Fatalf("got %d events", len(events))
	}
	logData := events[0].Data.(*domain.LogEventData)
	if logData.Message != "Turn started" {
		t.Errorf("message=%q", logData.Message)
	}
}

func TestCodex_DecodeStreamLine_Reasoning(t *testing.T) {
	events := codexDecodeOne(t, NewCodexForTest(), codexSamples["reasoning"], "")
	if len(events) != 1 {
		t.Fatalf("got %d events", len(events))
	}
	logData := events[0].Data.(*domain.LogEventData)
	if logData.Level != "debug" {
		t.Errorf("level=%s", logData.Level)
	}
	if len(logData.Message) < 11 || logData.Message[:11] != "Reasoning: " {
		t.Errorf("expected 'Reasoning: ' prefix, got %s", logData.Message)
	}
}

func TestCodex_DecodeStreamLine_FileChange(t *testing.T) {
	events := codexDecodeOne(t, NewCodexForTest(), codexSamples["file_change"], "")
	if len(events) != 1 {
		t.Fatalf("got %d events", len(events))
	}
	tool := requireToolCall(t, events, 0, "file_change")
	files, ok := tool.Input["files"].([]map[string]string)
	if !ok || len(files) != 1 {
		t.Fatalf("files=%v", tool.Input["files"])
	}
	if files[0]["path"] != "/tmp/test123.txt" || files[0]["kind"] != "add" {
		t.Errorf("file=%v", files[0])
	}
}

func TestCodex_DecodeStreamLine_FileChangeMultiple(t *testing.T) {
	events := codexDecodeOne(t, NewCodexForTest(), codexSamples["file_change_multiple"], "")
	tool := events[0].Data.(*domain.ToolCallEventData)
	files := tool.Input["files"].([]map[string]string)
	if len(files) != 3 {
		t.Fatalf("got %d files", len(files))
	}
	expected := []struct{ path, kind string }{
		{"/tmp/file1.txt", "add"},
		{"/tmp/file2.txt", "modify"},
		{"/tmp/file3.txt", "delete"},
	}
	for i, want := range expected {
		if files[i]["path"] != want.path || files[i]["kind"] != want.kind {
			t.Errorf("[%d] got %v want %v", i, files[i], want)
		}
	}
}

func TestCodex_DecodeStreamLine_AgentMessage(t *testing.T) {
	events := codexDecodeOne(t, NewCodexForTest(), codexSamples["agent_message"], "")
	requireMessage(t, events, 0, "assistant", "Created `test123.txt` containing `hello`.")
}

func TestCodex_DecodeStreamLine_TurnCompleted_BuildsCostEvent(t *testing.T) {
	c, _ := NewCodex(WithPricingService(&mockPricingService{
		pricing: map[string]*PricingCostCalculation{
			"gpt-5.1-codex-mini": {
				CostSource:       "pricing_table_estimate",
				Provider:         "openrouter",
				CanonicalModel:   "openai/gpt-5.1-codex-mini",
				PricingFetchedAt: time.Now().UTC(),
			},
		},
	}))
	events := codexDecodeOne(t, c, codexSamples["turn.completed"], "gpt-5.1-codex-mini")
	if len(events) != 2 {
		t.Fatalf("got %d events", len(events))
	}
	usage := events[0].Data.(*domain.UsageEventData)
	if usage.InputTokens != 394 || usage.OutputTokens != 83 {
		t.Errorf("tokens in/out=%d/%d", usage.InputTokens, usage.OutputTokens)
	}
	charge := events[1].Data.(*domain.ChargeEventData)
	if charge.AmountMicroUSD == nil || *charge.AmountMicroUSD <= 0 {
		t.Errorf("expected non-zero charge, got %#v", charge.AmountMicroUSD)
	}
	if usage.Model != "gpt-5.1-codex-mini" {
		t.Errorf("model=%s", usage.Model)
	}
	if usage.TurnIndex != 1 {
		t.Errorf("turn index=%d, want 1", usage.TurnIndex)
	}
	if charge.Basis != domain.ChargeBasisMetered {
		t.Errorf("basis=%s", charge.Basis)
	}
}

func TestCodex_DecodeStreamLine_TurnCompleted_NoPricingSvcGivesZeroCost(t *testing.T) {
	c := NewCodexForTest()
	events := codexDecodeOne(t, c, codexSamples["turn.completed"], "gpt-5.1-codex-mini")
	usage := events[0].Data.(*domain.UsageEventData)
	if usage.InputTokens != 394 {
		t.Errorf("tokens=%d", usage.InputTokens)
	}
	if len(events) != 1 {
		t.Fatalf("expected usage-only event without pricing, got %d events", len(events))
	}
}

func TestCodexCompletedUsageCountsCachedInputOnceAndUsesProviderTurns(t *testing.T) {
	c := NewCodexForTest()
	state := c.NewState()
	runID := uuid.New()
	metrics, last := runner.ExecutionMetrics{}, ""
	lines := []string{`{"type":"turn.started"}`}
	for i := 0; i < 6; i++ {
		lines = append(lines, codexSamples["agent_message"])
	}
	// Exact usage shape from the first corrected ordinary qualification run.
	lines = append(lines, `{"type":"turn.completed","usage":{"input_tokens":1197989,"cached_input_tokens":1108224,"cache_write_input_tokens":0,"output_tokens":7080,"reasoning_output_tokens":2467}}`)
	var finalUsage *domain.UsageEventData
	for _, line := range lines {
		events, err := c.DecodeStreamLine(state, runID, line)
		if err != nil {
			t.Fatal(err)
		}
		for _, event := range events {
			c.UpdateMetrics(event, &metrics, &last)
			if usage, ok := event.Data.(*domain.UsageEventData); ok {
				finalUsage = usage
			}
		}
	}
	if finalUsage == nil || finalUsage.InputTokens != 89765 || finalUsage.CacheReadTokens != 1108224 || finalUsage.OutputTokens != 7080 || !finalUsage.ReconciliationAuthority || finalUsage.Turns != 1 {
		t.Fatalf("provider receipt not normalized into disjoint usage: %+v", finalUsage)
	}
	if runner.TotalTokens(metrics) != 1205069 || metrics.TurnsUsed != 1 {
		t.Fatalf("cached input or assistant messages inflated invocation usage: %+v", metrics)
	}
}

func TestCodex_DecodeStreamLine_DataPrefix(t *testing.T) {
	events := codexDecodeOne(t, NewCodexForTest(), codexSamples["data_prefix_tool_call"], "")
	if len(events) == 0 {
		t.Fatal("expected events")
	}
	if events[0].EventType != domain.EventTypeToolCall {
		t.Errorf("type=%s", events[0].EventType)
	}
}

func TestCodex_DecodeStreamLine_CommandExecution(t *testing.T) {
	t.Run("started emits tool_call", func(t *testing.T) {
		events := codexDecodeOne(t, NewCodexForTest(), codexSamples["command_execution_started"], "")
		if len(events) != 1 || events[0].EventType != domain.EventTypeToolCall {
			t.Fatalf("got %v", events)
		}
		tc := events[0].Data.(*domain.ToolCallEventData)
		if tc.ToolName != "bash" {
			t.Errorf("name=%s", tc.ToolName)
		}
		if tc.ToolCallID != "item_4" {
			t.Errorf("toolCallID=%q", tc.ToolCallID)
		}
	})
	t.Run("completed emits tool_result only", func(t *testing.T) {
		events := codexDecodeOne(t, NewCodexForTest(), codexSamples["command_execution_completed"], "")
		if len(events) != 1 || events[0].EventType != domain.EventTypeToolResult {
			t.Fatalf("got %v", events)
		}
		if result := events[0].Data.(*domain.ToolResultEventData); result.ToolCallID != "item_4" {
			t.Errorf("toolCallID=%q", result.ToolCallID)
		}
	})
	t.Run("failed emits tool_call + tool_result with error", func(t *testing.T) {
		events := codexDecodeOne(t, NewCodexForTest(), codexSamples["command_execution_failed"], "")
		if len(events) != 2 {
			t.Fatalf("got %d events", len(events))
		}
		callData := events[0].Data.(*domain.ToolCallEventData)
		if status, _ := callData.Input["status"].(string); status != "failed" {
			t.Errorf("status=%v", callData.Input["status"])
		}
		resultData := events[1].Data.(*domain.ToolResultEventData)
		if callData.ToolCallID != "item_5" || resultData.ToolCallID != "item_5" {
			t.Fatalf("callID=%q resultID=%q", callData.ToolCallID, resultData.ToolCallID)
		}
		if resultData.Error == "" {
			t.Error("expected error on failed command result")
		}
		if resultData.Output == "" {
			t.Error("expected output on failed command result")
		}
	})
}

func TestCodex_DecodeStreamLine_Error(t *testing.T) {
	events := codexDecodeOne(t, NewCodexForTest(), codexSamples["error"], "")
	if len(events) != 1 || events[0].EventType != domain.EventTypeError {
		t.Fatalf("got %v", events)
	}
	errData := events[0].Data.(*domain.ErrorEventData)
	if errData.Code != "RATE_LIMIT" {
		t.Errorf("code=%s", errData.Code)
	}
	if errData.Message != "Rate limit exceeded, please try again later" {
		t.Errorf("msg=%s", errData.Message)
	}
}

func TestCodexTokenCountPreservesNativeRateLimitMetadata(t *testing.T) {
	parser := NewCodexForTest().NewTranscriptParser().(*codexTranscriptParser)
	runID := uuid.New()
	result := parser.ParseTranscriptLine(runID, `{"type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"input_tokens":10,"output_tokens":2}},"rate_limits":{"primary":{"used_percent":42.5,"window_minutes":300,"resets_at":1789300000},"secondary":{"used_percent":100,"window_minutes":10080,"resets_at":1789400000}}}}`)
	if len(result.Events) != 3 {
		t.Fatalf("events=%d, want usage plus two quota observations", len(result.Events))
	}
	var observations []*domain.RateLimitEventData
	for _, event := range result.Events {
		if data, ok := event.Data.(*domain.RateLimitEventData); ok {
			observations = append(observations, data)
		}
	}
	if len(observations) != 2 {
		t.Fatalf("rate limit observations=%d", len(observations))
	}
	byPool := make(map[string]*domain.RateLimitEventData, len(observations))
	for _, observation := range observations {
		byPool[observation.Pool] = observation
	}
	primary := byPool["primary"]
	if primary == nil || primary.Provider != "openai" || primary.WindowMinutes != 300 || primary.UsedPercent == nil || *primary.UsedPercent != 42.5 || primary.ResetTime == nil {
		t.Fatalf("primary observation=%+v", primary)
	}
	secondary := byPool["secondary"]
	if secondary == nil || secondary.UsedPercent == nil || *secondary.UsedPercent != 100 || secondary.ResetTime == nil {
		t.Fatalf("secondary observation=%+v", secondary)
	}
}

// Codex 0.15x rollouts add scalar and non-window siblings to rate_limits
// (limit_id, plan_type, credits, null windows). The frame's usage must still be
// recorded and only real windows observed.
func TestCodexTokenCountToleratesCurrentRateLimitShape(t *testing.T) {
	parser := NewCodexForTest().NewTranscriptParser()
	result := parser.ParseTranscriptLine(uuid.New(), `{"timestamp":"2026-09-30T15:52:35.016Z","ordinal":2212,"type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":213127,"cached_input_tokens":211712,"output_tokens":2186,"reasoning_output_tokens":1364,"total_tokens":215313},"model_context_window":258400},"rate_limits":{"limit_id":"codex","limit_name":null,"primary":{"used_percent":48.0,"window_minutes":10080,"resets_at":1791046702},"secondary":null,"credits":{"has_credits":true,"unlimited":false,"balance":"62500"},"individual_limit":null,"spend_control_reached":null,"plan_type":"pro","rate_limit_reached_type":null}}}`)
	var usage *domain.UsageEventData
	var windows []*domain.RateLimitEventData
	for _, event := range result.Events {
		switch data := event.Data.(type) {
		case *domain.UsageEventData:
			usage = data
		case *domain.RateLimitEventData:
			windows = append(windows, data)
		}
	}
	if usage == nil || usage.InputTokens != 213127-211712 || usage.CacheReadTokens != 211712 || usage.OutputTokens != 2186 {
		t.Fatalf("usage=%+v, want the frame's cumulative usage", usage)
	}
	if len(windows) != 1 || windows[0].Pool != "primary" || windows[0].UsedPercent == nil || *windows[0].UsedPercent != 48 {
		t.Fatalf("rate limit observations=%+v, want only the primary window", windows)
	}
}

func TestCodexLiveCodecPipeObservesNativeRateLimitMetadata(t *testing.T) {
	c := NewCodexForTest()
	state := c.NewState()
	percent := 42.5
	events, err := c.DecodeStreamLine(state, uuid.New(), `{"timestamp":"2026-09-13T12:00:00Z","type":"event_msg","payload":{"type":"token_count","info":null,"rate_limits":{"primary":{"used_percent":42.5,"window_minutes":300,"resets_at":1789300000}}}}`)
	if err != nil || len(events) != 1 {
		t.Fatalf("live decode events=%d err=%v", len(events), err)
	}
	data := requireType[*domain.RateLimitEventData](t, events, 0)
	if data.UsedPercent == nil || *data.UsedPercent != percent || data.Provider != "openai" {
		t.Fatalf("live quota event=%+v", events[0].Data)
	}
}

func TestCodexToolResultRetainsNestedProcessMetadata(t *testing.T) {
	line := `{"type":"item.completed","item":{"id":"item-meta","type":"tool_result","name":"bash","output":"{\"metadata\":{\"duration_seconds\":1.25}}"}}`
	events := codexDecodeOne(t, NewCodexForTest(), line, "")
	if len(events) != 1 {
		t.Fatalf("got %d events", len(events))
	}
	data := requireType[*domain.ToolResultEventData](t, events, 0)
	if data.DurationMS == nil || *data.DurationMS != 1250 || data.ExitCode != nil {
		t.Fatalf("nested metadata=%+v", data)
	}
}

func TestCodex_DecodeStreamLine_NonJsonAndUnknown(t *testing.T) {
	c := NewCodexForTest()
	cases := []string{"", "Shell cwd was reset to /tmp", `{"type":"invalid json`, `{"type":"unknown.event.type"}`}
	for _, line := range cases {
		events, _ := c.DecodeStreamLine(c.NewState(), uuid.New(), line)
		if len(events) != 0 {
			t.Errorf("line %q produced %d events", line, len(events))
		}
	}
}

// =============================================================================
// ANSI handling
// =============================================================================

func TestCodex_ANSIStrippedFromMessage(t *testing.T) {
	line := "{\"type\":\"item.completed\",\"item\":{\"id\":\"item_1\",\"type\":\"agent_message\",\"text\":\"\\u001b[1mBold\\u001b[0m and \\u001b[32mgreen\\u001b[0m text\"}}"
	events := codexDecodeOne(t, NewCodexForTest(), line, "")
	if len(events) != 1 {
		t.Fatalf("got %d events", len(events))
	}
	requireMessage(t, events, 0, "", "Bold and green text")
}

func TestCodex_ANSIStrippedFromToolOutput(t *testing.T) {
	line := "{\"type\":\"item.completed\",\"item\":{\"id\":\"item_2\",\"type\":\"tool_result\",\"name\":\"bash\",\"output\":\"\\u001b[32mSuccess\\u001b[0m: file created\"}}"
	events := codexDecodeOne(t, NewCodexForTest(), line, "")
	results := eventsOfType[*domain.ToolResultEventData](events)
	if len(results) == 0 {
		t.Error("no ToolResultEventData found")
	}
	for _, r := range results {
		if r.Output != "Success: file created" {
			t.Errorf("got %q", r.Output)
		}
	}
}

func TestCodex_HighVolumeANSINoSpam(t *testing.T) {
	c := NewCodexForTest()
	state := c.NewState()
	totalEvents := 0
	for i := 0; i < 1000; i++ {
		events, _ := c.DecodeStreamLine(state, uuid.New(), "\x1b[39;49m\x1b[K\x1b[2m└\x1b[39m\x1b[49m\x1b[0m")
		totalEvents += len(events)
	}
	if totalEvents != 0 {
		t.Errorf("expected 0 events from 1000 ANSI lines, got %d", totalEvents)
	}
}

// =============================================================================
// UpdateMetrics
// =============================================================================

func TestCodex_UpdateMetrics(t *testing.T) {
	c := NewCodexForTest()
	metrics := runner.ExecutionMetrics{}
	last := ""

	t.Run("ToolCallEvent", func(t *testing.T) {
		events := codexDecodeOne(t, c, codexSamples["file_change"], "")
		c.UpdateMetrics(events[0], &metrics, &last)
		if metrics.ToolCallCount != 1 {
			t.Errorf("ToolCallCount=%d", metrics.ToolCallCount)
		}
	})

	t.Run("MessageEvent", func(t *testing.T) {
		events := codexDecodeOne(t, c, codexSamples["agent_message"], "")
		c.UpdateMetrics(events[0], &metrics, &last)
		if metrics.TurnsUsed != 1 {
			t.Errorf("TurnsUsed=%d", metrics.TurnsUsed)
		}
		if last != "Created `test123.txt` containing `hello`." {
			t.Errorf("last=%q", last)
		}
	})

	t.Run("CostEvent", func(t *testing.T) {
		c2, _ := NewCodex(WithPricingService(&mockPricingService{
			pricing: map[string]*PricingCostCalculation{
				"gpt-5.1-codex-mini": {CostSource: "metered", Provider: "test"},
			},
		}))
		events := codexDecodeOne(t, c2, codexSamples["turn.completed"], "gpt-5.1-codex-mini")
		c2.UpdateMetrics(events[0], &metrics, &last)
		c2.UpdateMetrics(events[1], &metrics, &last)
		if metrics.TokensInput != 394 {
			t.Errorf("TokensInput=%d", metrics.TokensInput)
		}
		if metrics.TokensOutput != 83 {
			t.Errorf("TokensOutput=%d", metrics.TokensOutput)
		}
		if metrics.CostEstimateUSD <= 0 {
			t.Errorf("expected non-zero cost, got %v", metrics.CostEstimateUSD)
		}
	})
}

// =============================================================================
// ParseTranscriptLine
// =============================================================================

func TestCodex_ParseTranscriptLine_TerminalSuccess(t *testing.T) {
	c := NewCodexForTest()
	r := c.ParseTranscriptLine(uuid.New(), codexSamples["turn.completed"])
	if r.Terminal == nil {
		t.Fatal("expected Terminal")
	}
	if !r.Terminal.Success {
		t.Errorf("success=false")
	}
}

func TestCodex_ParseTranscriptLine_TerminalError(t *testing.T) {
	c := NewCodexForTest()
	r := c.ParseTranscriptLine(uuid.New(), codexSamples["error"])
	if r.Terminal == nil {
		t.Fatal("expected Terminal")
	}
	if r.Terminal.Success {
		t.Errorf("expected failure")
	}
	if r.Terminal.ErrorMessage == "" {
		t.Errorf("expected error message")
	}
}

func TestCodex_ParseTranscriptLine_SessionID(t *testing.T) {
	c := NewCodexForTest()
	r := c.ParseTranscriptLine(uuid.New(), codexSamples["thread.started"])
	if r.SessionID != "019b3906-b365-7403-b3d1-70d60f6f06c4" {
		t.Errorf("sessionID=%q", r.SessionID)
	}
}

func TestCodex_ParseTranscriptLine_RolloutTokenCountProjectsUsage(t *testing.T) {
	parser := NewCodexForTest().NewTranscriptParser()
	result := parser.ParseTranscriptLine(uuid.New(), `{"timestamp":"2026-08-02T12:00:00Z","type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"input_tokens":120,"cached_input_tokens":80,"output_tokens":30}}}}`)
	if len(result.Events) != 1 {
		t.Fatalf("events=%d, want one usage event", len(result.Events))
	}
	usage, ok := result.Events[0].Data.(*domain.UsageEventData)
	if !ok || usage.InputTokens != 120 || usage.OutputTokens != 30 || usage.CacheReadTokens != 80 {
		t.Fatalf("usage=%+v, want transcript token counts", result.Events[0].Data)
	}
}

func TestCodex_ParseTranscriptLine_RolloutTokenCountUsesCumulativeTotal(t *testing.T) {
	parser := NewCodexForTest().NewTranscriptParser()
	runID := uuid.New()
	first := parser.ParseTranscriptLine(runID, `{"type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"input_tokens":120,"cached_input_tokens":80,"output_tokens":30},"total_token_usage":{"input_tokens":120,"cached_input_tokens":80,"output_tokens":30}}}}`)
	second := parser.ParseTranscriptLine(runID, `{"type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"input_tokens":150,"cached_input_tokens":100,"output_tokens":40},"total_token_usage":{"input_tokens":270,"cached_input_tokens":180,"output_tokens":70}}}}`)
	if len(first.Events) != 1 || len(second.Events) != 1 {
		t.Fatalf("events=%d/%d, want one usage event each", len(first.Events), len(second.Events))
	}
	usage, ok := second.Events[0].Data.(*domain.UsageEventData)
	if !ok || usage.InputTokens != 50 || usage.OutputTokens != 40 || usage.CacheReadTokens != 100 {
		t.Fatalf("usage=%+v, want cumulative delta split from cache", second.Events[0].Data)
	}
}

func TestCodex_ParseTranscriptLine_RolloutTaskCompleteAuthorizesLatestUsage(t *testing.T) {
	parser := NewCodexForTest().NewTranscriptParser()
	runID := uuid.New()
	usage := parser.ParseTranscriptLine(runID, `{"type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"input_tokens":120,"cached_input_tokens":80,"output_tokens":30}}}}`)
	if len(usage.Events) != 1 {
		t.Fatalf("usage events=%d, want one", len(usage.Events))
	}
	if got := usage.Events[0].Data.(*domain.UsageEventData).ReconciliationAuthority; got {
		t.Fatal("interim rollout usage was authoritative")
	}
	terminal := parser.ParseTranscriptLine(runID, `{"type":"event_msg","payload":{"type":"task_complete"}}`)
	if terminal.Terminal == nil || !terminal.Terminal.Success {
		t.Fatalf("terminal=%+v, want successful task_complete", terminal.Terminal)
	}
	if len(terminal.Events) != 1 {
		t.Fatalf("terminal events=%d, want one authoritative usage event", len(terminal.Events))
	}
	if got := terminal.Events[0].Data.(*domain.UsageEventData).ReconciliationAuthority; !got {
		t.Fatal("latest rollout usage was not emitted with terminal authority")
	}
}

func TestCodexRetainedTerminalReceiptConservesCumulativeUsage(t *testing.T) {
	for _, ending := range []string{"turn_aborted", "task_complete"} {
		t.Run(ending, func(t *testing.T) {
			parser := NewCodexForTest().NewTranscriptParser()
			parser.(runner.TranscriptBillingSetter).SetTranscriptBilling(domain.BillingSnapshot{Basis: domain.ChargeBasisSubscription})
			id := uuid.New()
			parser.ParseTranscriptLine(id, `{"type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":15211,"cached_input_tokens":10880,"output_tokens":205}}}}`)
			parser.ParseTranscriptLine(id, `{"type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":30666,"cached_input_tokens":21760,"output_tokens":415}}}}`)
			result := parser.ParseTranscriptLine(id, fmt.Sprintf(`{"type":"event_msg","payload":{"type":%q}}`, ending))
			var receipt *domain.UsageEventData
			for _, event := range result.Events {
				if usage, ok := event.Data.(*domain.UsageEventData); ok && usage.ReconciliationAuthority {
					receipt = usage
				}
			}
			if receipt == nil || receipt.InputTokens != 8906 || receipt.CacheReadTokens != 21760 || receipt.OutputTokens != 415 || receipt.Turns != 1 {
				t.Fatalf("terminal lost cumulative receipt: %+v", receipt)
			}
			if result.Terminal == nil || result.Terminal.Success != (ending == "task_complete") {
				t.Fatal("accounting changed terminal outcome")
			}
			if receipt.Charge == nil || receipt.Charge.AmountMicroUSD == nil || *receipt.Charge.AmountMicroUSD != 0 || receipt.Charge.Basis != domain.ChargeBasisSubscription {
				t.Fatalf("immutable billing lost: %+v", receipt.Charge)
			}
		})
	}
}

func TestCodexContinuationUsesSameSessionCumulativeBaseline(t *testing.T) {
	const session = "codex-session-a"
	parse := func(parser runner.TranscriptParser, runID uuid.UUID, typ, payload string) runner.TranscriptParseResult {
		return parser.ParseTranscriptLine(runID, fmt.Sprintf(`{"type":%q,"payload":%s}`, typ, payload))
	}

	runID := uuid.New()
	first := NewCodexForTest().NewTranscriptParser()
	parse(first, runID, "session_meta", `{"id":"`+session+`"}`)
	parse(first, runID, "event_msg", `{"type":"token_count","info":{"total_token_usage":{"input_tokens":100,"cached_input_tokens":20,"output_tokens":30}}}`)
	firstReceipt := requireTerminalUsage(t, parse(first, runID, "event_msg", `{"type":"task_complete"}`))
	if firstReceipt.InputTokens != 80 || firstReceipt.CacheReadTokens != 20 || firstReceipt.OutputTokens != 30 || firstReceipt.ReconciliationComplete == nil || !*firstReceipt.ReconciliationComplete {
		t.Fatalf("first session receipt=%+v", firstReceipt)
	}
	if firstReceipt.CodexUsageSnapshot == nil || firstReceipt.CodexUsageSnapshot.SessionID != session {
		t.Fatalf("raw cumulative snapshot missing: %+v", firstReceipt.CodexUsageSnapshot)
	}

	continued := NewCodexForTest().NewTranscriptParser()
	continued.(runner.ContinuationUsageBaselineSetter).SetContinuationUsageBaseline(runner.CodexUsageBaseline{SessionID: session, Snapshot: firstReceipt.CodexUsageSnapshot})
	parse(continued, runID, "session_meta", `{"id":"`+session+`"}`)
	parse(continued, runID, "event_msg", `{"type":"token_count","info":{"total_token_usage":{"input_tokens":155,"cached_input_tokens":50,"output_tokens":35}}}`)
	secondReceipt := requireTerminalUsage(t, parse(continued, runID, "event_msg", `{"type":"task_complete"}`))
	if secondReceipt.InputTokens != 25 || secondReceipt.CacheReadTokens != 30 || secondReceipt.OutputTokens != 5 || secondReceipt.ReconciliationComplete == nil || !*secondReceipt.ReconciliationComplete {
		t.Fatalf("continuation receipt did not use raw same-session baseline: %+v", secondReceipt)
	}

	// A replayed cumulative snapshot contributes no second increment.
	replay := NewCodexForTest().NewTranscriptParser()
	replay.(runner.ContinuationUsageBaselineSetter).SetContinuationUsageBaseline(runner.CodexUsageBaseline{SessionID: session, Snapshot: firstReceipt.CodexUsageSnapshot})
	parse(replay, runID, "session_meta", `{"id":"`+session+`"}`)
	line := `{"type":"token_count","info":{"total_token_usage":{"input_tokens":155,"cached_input_tokens":50,"output_tokens":35}}}`
	parse(replay, runID, "event_msg", line)
	duplicate := parse(replay, runID, "event_msg", line)
	if len(duplicate.Events) != 0 {
		t.Fatalf("identical provider snapshot replayed as usage: %+v", duplicate.Events)
	}
	replayReceipt := requireTerminalUsage(t, parse(replay, runID, "event_msg", `{"type":"task_complete"}`))
	if replayReceipt.InputTokens != secondReceipt.InputTokens || replayReceipt.CacheReadTokens != secondReceipt.CacheReadTokens || replayReceipt.OutputTokens != secondReceipt.OutputTokens {
		t.Fatalf("replay changed terminal delta: got %+v want %+v", replayReceipt, secondReceipt)
	}
}

func TestCodexContinuationBaselineUnknownResetAndSessionSwitch(t *testing.T) {
	const session = "codex-session-a"
	parse := func(parser runner.TranscriptParser, runID uuid.UUID, typ, payload string) runner.TranscriptParseResult {
		return parser.ParseTranscriptLine(runID, fmt.Sprintf(`{"type":%q,"payload":%s}`, typ, payload))
	}
	runID := uuid.New()

	unknown := NewCodexForTest().NewTranscriptParser()
	unknown.(runner.ContinuationUsageBaselineSetter).SetContinuationUsageBaseline(runner.CodexUsageBaseline{SessionID: session})
	parse(unknown, runID, "session_meta", `{"id":"`+session+`"}`)
	parse(unknown, runID, "event_msg", `{"type":"token_count","info":{"total_token_usage":{"input_tokens":155,"cached_input_tokens":50,"output_tokens":35}}}`)
	unknownReceipt := requireTerminalUsage(t, parse(unknown, runID, "event_msg", `{"type":"task_complete"}`))
	if unknownReceipt.ReconciliationComplete == nil || *unknownReceipt.ReconciliationComplete {
		t.Fatalf("missing baseline was qualified: %+v", unknownReceipt)
	}
	if unknownReceipt.CodexUsageSnapshot == nil || unknownReceipt.CodexUsageSnapshot.SessionID != session {
		t.Fatalf("unseeded receipt did not preserve its current raw session snapshot: %+v", unknownReceipt.CodexUsageSnapshot)
	}

	unknownThenSwitched := NewCodexForTest().NewTranscriptParser()
	unknownThenSwitched.(runner.ContinuationUsageBaselineSetter).SetContinuationUsageBaseline(runner.CodexUsageBaseline{SessionID: session})
	parse(unknownThenSwitched, runID, "session_meta", `{"id":"`+session+`"}`)
	parse(unknownThenSwitched, runID, "event_msg", `{"type":"token_count","info":{"total_token_usage":{"input_tokens":155,"cached_input_tokens":50,"output_tokens":35}}}`)
	parse(unknownThenSwitched, runID, "session_meta", `{"id":"codex-session-b"}`)
	parse(unknownThenSwitched, runID, "event_msg", `{"type":"token_count","info":{"total_token_usage":{"input_tokens":12,"cached_input_tokens":4,"output_tokens":3}}}`)
	unknownSwitchReceipt := requireTerminalUsage(t, parse(unknownThenSwitched, runID, "event_msg", `{"type":"task_complete"}`))
	if unknownSwitchReceipt.ReconciliationComplete == nil || *unknownSwitchReceipt.ReconciliationComplete {
		t.Fatalf("session switch incorrectly qualified an earlier unknown counter interval: %+v", unknownSwitchReceipt)
	}

	reset := NewCodexForTest().NewTranscriptParser()
	reset.(runner.ContinuationUsageBaselineSetter).SetContinuationUsageBaseline(runner.CodexUsageBaseline{SessionID: session, Snapshot: &domain.CodexUsageSnapshot{SessionID: session, InputTokens: 1000, CachedInputTokens: 300, OutputTokens: 500}})
	parse(reset, runID, "session_meta", `{"id":"`+session+`"}`)
	parse(reset, runID, "event_msg", `{"type":"token_count","info":{"total_token_usage":{"input_tokens":12,"cached_input_tokens":4,"output_tokens":3}}}`)
	resetReceipt := requireTerminalUsage(t, parse(reset, runID, "event_msg", `{"type":"task_complete"}`))
	if resetReceipt.InputTokens != 8 || resetReceipt.CacheReadTokens != 4 || resetReceipt.OutputTokens != 3 || resetReceipt.ReconciliationComplete == nil || !*resetReceipt.ReconciliationComplete {
		t.Fatalf("counter reset did not start a trusted new series: %+v", resetReceipt)
	}

	switched := NewCodexForTest().NewTranscriptParser()
	switched.(runner.ContinuationUsageBaselineSetter).SetContinuationUsageBaseline(runner.CodexUsageBaseline{SessionID: session, Snapshot: &domain.CodexUsageSnapshot{SessionID: session, InputTokens: 1000, CachedInputTokens: 300, OutputTokens: 500}})
	parse(switched, runID, "session_meta", `{"id":"codex-session-b"}`)
	parse(switched, runID, "event_msg", `{"type":"token_count","info":{"total_token_usage":{"input_tokens":12,"cached_input_tokens":4,"output_tokens":3}}}`)
	switchReceipt := requireTerminalUsage(t, parse(switched, runID, "event_msg", `{"type":"task_complete"}`))
	if switchReceipt.InputTokens != 8 || switchReceipt.CacheReadTokens != 4 || switchReceipt.OutputTokens != 3 || switchReceipt.ReconciliationComplete == nil || !*switchReceipt.ReconciliationComplete || switchReceipt.CodexUsageSnapshot.SessionID != "codex-session-b" {
		t.Fatalf("new session reused prior session counters: %+v", switchReceipt)
	}
}

func TestCodexStreamContinuationDifferencesTerminalSnapshotFromBaseline(t *testing.T) {
	c := NewCodexForTest()
	state := c.NewState()
	state.(runner.ContinuationUsageBaselineSetter).SetContinuationUsageBaseline(runner.CodexUsageBaseline{
		SessionID: "stream-session",
		Snapshot:  &domain.CodexUsageSnapshot{SessionID: "stream-session", InputTokens: 100, CachedInputTokens: 20, OutputTokens: 30},
	})
	if _, err := c.DecodeStreamLine(state, uuid.New(), `{"type":"thread.started","thread_id":"stream-session"}`); err != nil {
		t.Fatal(err)
	}
	events, err := c.DecodeStreamLine(state, uuid.New(), `{"type":"turn.completed","usage":{"input_tokens":155,"cached_input_tokens":50,"output_tokens":35}}`)
	if err != nil {
		t.Fatal(err)
	}
	for _, evt := range events {
		usage, ok := evt.Data.(*domain.UsageEventData)
		if !ok {
			continue
		}
		if usage.InputTokens != 25 || usage.CacheReadTokens != 30 || usage.OutputTokens != 5 || !usage.ReconciliationAuthority || usage.ReconciliationComplete == nil || !*usage.ReconciliationComplete || usage.CodexUsageSnapshot == nil || usage.CodexUsageSnapshot.InputTokens != 155 {
			t.Fatalf("stream continuation receipt=%+v", usage)
		}
		return
	}
	t.Fatalf("stream continuation usage receipt missing: %+v", events)
}

func TestCodexStreamSessionSwitchStartsIndependentCounterSeries(t *testing.T) {
	c := NewCodexForTest()
	state := c.NewState()
	if _, err := c.DecodeStreamLine(state, uuid.New(), `{"type":"thread.started","thread_id":"old-session"}`); err != nil {
		t.Fatal(err)
	}
	if _, err := c.DecodeStreamLine(state, uuid.New(), `{"type":"turn.completed","usage":{"input_tokens":1000,"cached_input_tokens":300,"output_tokens":500}}`); err != nil {
		t.Fatal(err)
	}
	if _, err := c.DecodeStreamLine(state, uuid.New(), `{"type":"thread.started","thread_id":"new-session"}`); err != nil {
		t.Fatal(err)
	}
	events, err := c.DecodeStreamLine(state, uuid.New(), `{"type":"turn.completed","usage":{"input_tokens":12,"cached_input_tokens":4,"output_tokens":3}}`)
	if err != nil {
		t.Fatal(err)
	}
	for _, evt := range events {
		usage, ok := evt.Data.(*domain.UsageEventData)
		if !ok {
			continue
		}
		if usage.InputTokens != 8 || usage.CacheReadTokens != 4 || usage.OutputTokens != 3 || usage.ReconciliationComplete == nil || !*usage.ReconciliationComplete || usage.CodexUsageSnapshot == nil || usage.CodexUsageSnapshot.SessionID != "new-session" {
			t.Fatalf("new provider session reused old counters: %+v", usage)
		}
		return
	}
	t.Fatalf("session-switch usage receipt missing: %+v", events)
}

func TestCodex_ParseTranscriptLine_RolloutTurnsPreserveProviderTurnIDs(t *testing.T) {
	parser := NewCodexForTest().NewTranscriptParser()
	runID := uuid.New()
	for _, turn := range []string{"turn-a", "turn-b"} {
		started := parser.ParseTranscriptLine(runID, fmt.Sprintf(`{"type":"event_msg","payload":{"type":"task_started","turn_id":%q}}`, turn))
		if started.Terminal != nil {
			t.Fatalf("task_started became terminal: %+v", started.Terminal)
		}
		message := parser.ParseTranscriptLine(runID, fmt.Sprintf(`{"type":"event_msg","payload":{"type":"agent_message","message":%q}}`, turn))
		if len(message.Events) != 1 {
			t.Fatalf("message events=%d, want one", len(message.Events))
		}
		got := message.Events[0].Data.(*domain.MessageEventData).TurnID
		if got != turn {
			t.Fatalf("turn id=%q, want %q", got, turn)
		}
		parser.ParseTranscriptLine(runID, `{"type":"event_msg","payload":{"type":"task_complete"}}`)
	}
}

func TestCodex_ParseTranscriptLine_RolloutResponseItemMessage(t *testing.T) {
	parser := NewCodexForTest().NewTranscriptParser()
	runID := uuid.New()
	result := parser.ParseTranscriptLine(runID, `{"type":"response_item","payload":{"type":"message","id":"msg-1","role":"assistant","content":[{"type":"output_text","text":"The validation-intent receipt contract is preserved."}]}}`)
	if len(result.Events) != 1 {
		t.Fatalf("events=%v, want one assistant message", result.Events)
	}
	requireMessage(t, result.Events, 0, "assistant", "The validation-intent receipt contract is preserved.")
}

func TestCodex_ParseTranscriptLine_RolloutResponseItemUserMessageHonorsRetention(t *testing.T) {
	parser := NewCodexForTest().NewTranscriptParser()
	parser.(*codexTranscriptParser).SetTranscriptRetention(true)
	result := parser.ParseTranscriptLine(uuid.New(), `{"type":"response_item","payload":{"type":"message","id":"msg-2","role":"user","content":[{"type":"input_text","text":"remember this request"}]}}`)
	if len(result.Events) != 1 {
		t.Fatalf("events=%d, want retained user message", len(result.Events))
	}
	requireMessage(t, result.Events, 0, "user", "remember this request")
}

// =============================================================================
// Continue rejects empty session
// =============================================================================

func TestCodex_ClassifyTerminalError(t *testing.T) {
	c := NewCodexForTest()
	cases := []struct {
		name     string
		stderr   string
		exitCode int
		wantCode domain.ErrorCode // empty = expect nil result
	}{
		{
			name:     "session-expired bare thread-not-found",
			stderr:   "thread abc was not found",
			exitCode: 1,
			wantCode: domain.ErrCodeRunnerSessionExpired,
		},
		{
			name:     "state-lost rollout-writer race (function-name form)",
			stderr:   "ERROR codex_rollout::recorder: record_rollout_items: thread 019dda9c was not found",
			exitCode: 1,
			wantCode: domain.ErrCodeRunnerSessionStateLost,
		},
		{
			// Real production stderr from a heartbeat-driven failure
			// (run 08f9fb93, 2026-04-29 23:15 UTC). The codex binary
			// emits the human-readable form, not the function-name
			// form, so the classifier MUST match both shapes or the
			// state-lost burst stays misclassified as session-expired.
			name:     "state-lost rollout-writer race (human-readable form)",
			stderr:   "2026-04-29T23:15:03.309169Z ERROR codex_core::session: failed to record rollout items: thread 019ddb86-3b2e-72a0-9a6b-cd6bc787b155 not found",
			exitCode: 1,
			wantCode: domain.ErrCodeRunnerSessionStateLost,
		},
		{
			name:     "missing thread without not-found is not classified",
			stderr:   "the thread is missing",
			exitCode: 1,
			wantCode: "",
		},
		{
			name:     "unrelated stderr is not classified",
			stderr:   "some other error",
			exitCode: 1,
			wantCode: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := c.ClassifyTerminalError(tc.stderr, tc.exitCode)
			if tc.wantCode == "" {
				if got != nil {
					t.Fatalf("expected nil result, got %v", got)
				}
				return
			}
			if got == nil {
				t.Fatalf("expected %s, got nil", tc.wantCode)
			}
			if got.Code() != tc.wantCode {
				t.Errorf("Code() = %s, want %s", got.Code(), tc.wantCode)
			}
			if got.RunnerType != domain.RunnerTypeCodex {
				t.Errorf("RunnerType = %s, want codex", got.RunnerType)
			}
		})
	}
}

// =============================================================================
// JSON unmarshal sanity for stream-event types
// =============================================================================

func TestCodexStreamEvent_Unmarshal(t *testing.T) {
	var ev CodexStreamEvent
	if err := json.Unmarshal([]byte(codexSamples["file_change"]), &ev); err != nil {
		t.Fatalf("unmarshal err: %v", err)
	}
	if ev.Type != "item.completed" {
		t.Errorf("type=%s", ev.Type)
	}
	if ev.Item == nil || ev.Item.Type != "file_change" {
		t.Fatalf("item=%v", ev.Item)
	}
	if len(ev.Item.Changes) != 1 {
		t.Fatalf("changes=%d", len(ev.Item.Changes))
	}
	if ev.Item.Changes[0].Path != "/tmp/test123.txt" {
		t.Errorf("path=%s", ev.Item.Changes[0].Path)
	}
}

func TestCodexStreamEvent_UnmarshalUsage(t *testing.T) {
	var ev CodexStreamEvent
	if err := json.Unmarshal([]byte(codexSamples["turn.completed"]), &ev); err != nil {
		t.Fatalf("unmarshal err: %v", err)
	}
	if ev.Usage == nil {
		t.Fatal("usage nil")
	}
	if ev.Usage.InputTokens != 12810 || ev.Usage.OutputTokens != 83 || ev.Usage.CachedInputTokens != 12416 {
		t.Errorf("usage=%+v", ev.Usage)
	}
}

// =============================================================================
// Full stream simulation
// =============================================================================

func TestCodex_FullStream(t *testing.T) {
	c, _ := NewCodex(WithPricingService(&mockPricingService{
		pricing: map[string]*PricingCostCalculation{
			"gpt-5.1-codex-mini": {CostSource: "pricing_table_estimate"},
		},
	}))
	state := c.NewState().(*codexState)
	state.runModel = "gpt-5.1-codex-mini"
	runID := uuid.New()

	streamLines := []string{
		codexSamples["thread.started"],
		codexSamples["turn.started"],
		codexSamples["reasoning"],
		codexSamples["file_change"],
		codexSamples["agent_message"],
		codexSamples["turn.completed"],
	}

	metrics := runner.ExecutionMetrics{}
	last := ""
	allEvents := []*domain.RunEvent{}
	for _, line := range streamLines {
		events, _ := c.DecodeStreamLine(state, runID, line)
		for _, evt := range events {
			c.UpdateMetrics(evt, &metrics, &last)
			allEvents = append(allEvents, evt)
		}
	}

	if len(allEvents) != 8 {
		t.Errorf("expected 8 events, got %d", len(allEvents))
	}
	expected := []domain.RunEventType{
		domain.EventTypeLog,
		domain.EventTypeLog,
		domain.EventTypeLog,
		domain.EventTypeToolCall,
		domain.EventTypeMessage,
		domain.EventTypeMessage,
		domain.EventTypeMetric,
		domain.EventTypeMetric,
	}
	for i, et := range expected {
		if i >= len(allEvents) {
			break
		}
		if allEvents[i].EventType != et {
			t.Errorf("[%d] type=%s want %s", i, allEvents[i].EventType, et)
		}
	}
	if metrics.TurnsUsed != 1 {
		t.Errorf("turns=%d", metrics.TurnsUsed)
	}
	if metrics.ToolCallCount != 1 {
		t.Errorf("toolcalls=%d", metrics.ToolCallCount)
	}
	if metrics.TokensInput != 394 || metrics.TokensOutput != 83 {
		t.Errorf("tokens in/out=%d/%d", metrics.TokensInput, metrics.TokensOutput)
	}
}

func TestCodexExtractCommandFromExecCommandWrapper(t *testing.T) {
	codec := NewCodexForTest()
	got := codec.ExtractCommand(map[string]any{
		"input": `const r = await tools.exec_command({"cmd":"rg -n \"InvocationFactVersion\" scenarios/agent-manager/api"}); text(r.output);`,
	})
	if got.Command != `rg -n "InvocationFactVersion" scenarios/agent-manager/api` {
		t.Fatalf("command=%q reason=%q", got.Command, got.Reason)
	}
	if got.LiteralArgs {
		t.Fatal("wrapped shell command must remain shell text")
	}
}

func TestCodexExtractCommandFromJavaScriptExecCommandWrapper(t *testing.T) {
	codec := NewCodexForTest()
	got := codec.ExtractCommand(map[string]any{
		"input": `const r = await tools.exec_command({cmd:"sed -n '1,20p' README.md",workdir:"/tmp"}); text(r.output);`,
	})
	if got.Command != `sed -n '1,20p' README.md` {
		t.Fatalf("command=%q reason=%q", got.Command, got.Reason)
	}
}

func TestCodexRolloutToolOutputContract(t *testing.T) {
	for _, tc := range []struct {
		name, output string
		blocks       int
		known        bool
		success      bool
		exit         *int
		warning      bool
	}{
		{"scalar unknown", `"returned"`, 0, false, false, nil, false},
		{"ordered text", `[{"type":"input_text","text":"one"},{"type":"input_text","text":"two"}]`, 2, false, false, nil, false},
		{"mixed content", `[{"type":"input_text","text":"one"},{"type":"image","image_url":"retained"}]`, 2, false, false, nil, true},
		{"unknown scalar", `7`, 0, false, false, nil, true},
		{"wrong exit type", `"{\"metadata\":{\"exit_code\":\"bogus\"}}"`, 0, false, false, nil, true},
		{"oversized exit", `"{\"metadata\":{\"exit_code\":4294967296}}"`, 0, false, false, nil, true},
		{"null content", `null`, 0, false, false, nil, true},
		{"conflicting metadata", `[{"type":"input_text","text":"{\"metadata\":{\"exit_code\":0}}"},{"type":"input_text","text":"{\"metadata\":{\"exit_code\":9}}"}]`, 2, false, false, nil, true},
		{"explicit zero", `"{\"metadata\":{\"exit_code\":0,\"duration_seconds\":0.125}}"`, 0, true, true, codexExit(0), false},
		{"explicit failure", `[{"type":"input_text","text":"{\"metadata\":{\"exit_code\":9}}"}]`, 1, true, false, codexExit(9), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := NewCodexForTest().NewTranscriptParser()
			line := `{"type":"response_item","payload":{"type":"custom_tool_call_output","call_id":"original","output":` + tc.output + `}}`
			got := p.ParseTranscriptLine(uuid.New(), line)
			if got.Err != nil || got.Terminal != nil || len(got.Events) == 0 {
				t.Fatalf("parse=%+v", got)
			}
			data, ok := got.Events[0].Data.(*domain.ToolResultEventData)
			if !ok || data.ToolCallID != "original" || len(data.OutputBlocks) != tc.blocks || data.OutcomeKnown == nil || *data.OutcomeKnown != tc.known || data.Success != tc.success || string(data.RawOutputJSON) != tc.output {
				t.Fatalf("tool=%+v", data)
			}
			if (data.ExitCode == nil) != (tc.exit == nil) || tc.exit != nil && *data.ExitCode != *tc.exit {
				t.Fatalf("exit=%v", data.ExitCode)
			}
			if tc.name == "explicit zero" && (data.DurationMS == nil || *data.DurationMS != 125) {
				t.Fatalf("duration=%v", data.DurationMS)
			}
			if tc.blocks > 0 && string(data.OutputBlocks[0].RawJSON) == "" {
				t.Fatal("raw block lost")
			}
			if (len(got.Events) > 1) != tc.warning {
				t.Fatalf("diagnostics=%+v", got.Events)
			}
			// A warning must not stop a later user boundary or valid terminal.
			user := p.ParseTranscriptLine(uuid.New(), `{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"next"}]}}`)
			if !user.UserTurnStarted || user.Err != nil {
				t.Fatalf("user=%+v", user)
			}
			terminal := p.ParseTranscriptLine(uuid.New(), `{"type":"event_msg","payload":{"type":"task_complete","turn_id":"later"}}`)
			if terminal.Terminal == nil || !terminal.Terminal.Success {
				t.Fatalf("terminal=%+v", terminal)
			}
		})
	}
}

func codexExit(n int) *int { return &n }

func TestCodexE30FourteenOrderedReturnsReplay(t *testing.T) {
	content, err := os.ReadFile("testdata/codex_tool_returns_e30.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	p := NewCodexForTest().NewTranscriptParser()
	runID := uuid.New()
	calls := map[string]bool{}
	results := 0
	for _, line := range strings.Split(strings.TrimSpace(string(content)), "\n") {
		got := p.ParseTranscriptLine(runID, line)
		if got.Err != nil {
			t.Fatal(got.Err)
		}
		for _, event := range got.Events {
			switch data := event.Data.(type) {
			case *domain.ToolCallEventData:
				calls[data.ToolCallID] = true
			case *domain.ToolResultEventData:
				results++
				if !calls[data.ToolCallID] || len(data.OutputBlocks) != 2 || data.OutputBlocks[0].Type != "input_text" || data.OutputBlocks[1].Type != "input_text" || data.ExitCode != nil || data.OutcomeKnown == nil || *data.OutcomeKnown || data.Success {
					t.Fatalf("return %d=%+v", results, data)
				}
			}
		}
	}
	if len(calls) != 14 || results != 14 {
		t.Fatalf("calls=%d returns=%d", len(calls), results)
	}
}

func TestCodexToolDurationRefusesInvalidMetadataIndependently(t *testing.T) {
	for _, tc := range []struct {
		name, seconds string
		want          *int64
		warning       bool
	}{
		{"zero", "0", codexMS(0), false},
		{"fractional", "0.125", codexMS(125), false},
		{"negative", "-0.001", nil, true},
		{"overflow", "1e308", nil, true},
		{"wrong type", `"bogus"`, nil, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			output := `{"metadata":{"exit_code":0,"duration_seconds":` + tc.seconds + `}}`
			raw, _ := json.Marshal(output)
			got := codexRolloutToolResult(uuid.New(), codexRolloutPayload{CallID: "duration", Output: raw})
			d := got[0].Data.(*domain.ToolResultEventData)
			if d.ExitCode == nil || *d.ExitCode != 0 || !d.Success || d.OutcomeKnown == nil || !*d.OutcomeKnown {
				t.Fatalf("independent exit lost: %+v", d)
			}
			if (d.DurationMS == nil) != (tc.want == nil) || tc.want != nil && *d.DurationMS != *tc.want || (len(got) > 1) != tc.warning {
				t.Fatalf("duration/warning=%+v events=%d", d, len(got))
			}
			if d.RawOutputJSON != string(raw) {
				t.Fatal("raw duration evidence lost")
			}
		})
	}
}
func codexMS(n int64) *int64 { return &n }

func TestCodexFlatTypedMetadataRetainsIndependentFacts(t *testing.T) {
	for _, tc := range []struct {
		name, output   string
		known, success bool
		warning        bool
	}{
		{"invalid exit", `{"metadata":{"exit_code":"bad"}}`, false, false, true},
		{"oversized exit", `{"metadata":{"exit_code":4294967296}}`, false, false, true},
		{"valid exit invalid duration", `{"metadata":{"exit_code":0,"duration_seconds":"bad"}}`, true, true, true},
		{"nonzero", `{"metadata":{"exit_code":9}}`, true, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			events := parseCodexItemEvents(nil, uuid.New(), &CodexItem{Type: "tool_result", Name: "exec", ID: "native", Output: tc.output})
			var d *domain.ToolResultEventData
			warnings := 0
			for _, event := range events {
				if got, ok := event.Data.(*domain.ToolResultEventData); ok {
					d = got
				}
				if _, ok := event.Data.(*domain.LogEventData); ok {
					warnings++
				}
			}
			if d == nil || d.OutcomeKnown == nil || *d.OutcomeKnown != tc.known || d.Success != tc.success || (warnings > 0) != tc.warning || d.Output != tc.output {
				t.Fatalf("data=%+v warnings=%d", d, warnings)
			}
		})
	}
}
