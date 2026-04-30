package codecs

import (
	"context"
	"encoding/json"
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
	hasFullAuto, hasModel, hasC := false, false, false
	for i, a := range args {
		switch a {
		case "--full-auto":
			hasFullAuto = true
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
	if !hasFullAuto {
		t.Errorf("expected --full-auto for NetworkAccessNone, got %v", args)
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
	hasBypass, hasFullAuto := false, false
	for _, a := range args {
		switch a {
		case "--dangerously-bypass-approvals-and-sandbox":
			hasBypass = true
		case "--full-auto":
			hasFullAuto = true
		}
	}
	if !hasBypass || hasFullAuto {
		t.Errorf("expected --dangerously-bypass-approvals-and-sandbox without --full-auto for localhost, got %v", args)
	}
}

func TestCodex_BuildContinueArgs_IncludesSessionAndPrompt(t *testing.T) {
	c := NewCodexForTest()
	args := c.BuildContinueArgs(c.NewState(), runner.ContinueRequest{
		RunID:     uuid.New(),
		SessionID: "thread-abc",
		Prompt:    "follow up message",
	})
	want := []string{"exec", "resume", "--json", "--skip-git-repo-check", "--full-auto", "thread-abc", "follow up message"}
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
	if events[0].EventType != domain.EventTypeToolCall {
		t.Errorf("type=%s want tool_call", events[0].EventType)
	}
	tool := events[0].Data.(*domain.ToolCallEventData)
	if tool.ToolName != "file_change" {
		t.Errorf("name=%s", tool.ToolName)
	}
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
	if events[0].EventType != domain.EventTypeMessage {
		t.Errorf("type=%s", events[0].EventType)
	}
	msg := events[0].Data.(*domain.MessageEventData)
	if msg.Role != "assistant" {
		t.Errorf("role=%s", msg.Role)
	}
	if msg.Content != "Created `test123.txt` containing `hello`." {
		t.Errorf("content=%q", msg.Content)
	}
}

func TestCodex_DecodeStreamLine_TurnCompleted_BuildsCostEvent(t *testing.T) {
	c, _ := NewCodex(WithCodexPricingService(&mockPricingService{
		pricing: map[string]*PricingCostCalculation{
			"gpt-5.1-codex-mini": {
				CostSource:       domain.CostSourcePricingTableEstimate,
				Provider:         "openrouter",
				CanonicalModel:   "openai/gpt-5.1-codex-mini",
				PricingFetchedAt: time.Now().UTC(),
			},
		},
	}))
	events := codexDecodeOne(t, c, codexSamples["turn.completed"], "gpt-5.1-codex-mini")
	if len(events) != 1 {
		t.Fatalf("got %d events", len(events))
	}
	cost := events[0].Data.(*domain.CostEventData)
	if cost.InputTokens != 12810 || cost.OutputTokens != 83 {
		t.Errorf("tokens in/out=%d/%d", cost.InputTokens, cost.OutputTokens)
	}
	if cost.TotalCostUSD <= 0 {
		t.Errorf("expected non-zero cost, got %v", cost.TotalCostUSD)
	}
	if cost.Model != "gpt-5.1-codex-mini" {
		t.Errorf("model=%s", cost.Model)
	}
	if cost.CostSource != domain.CostSourcePricingTableEstimate {
		t.Errorf("costSource=%s", cost.CostSource)
	}
	if cost.PricingProvider != "openrouter" {
		t.Errorf("provider=%s", cost.PricingProvider)
	}
}

func TestCodex_DecodeStreamLine_TurnCompleted_NoPricingSvcGivesZeroCost(t *testing.T) {
	c := NewCodexForTest()
	events := codexDecodeOne(t, c, codexSamples["turn.completed"], "gpt-5.1-codex-mini")
	cost := events[0].Data.(*domain.CostEventData)
	if cost.InputTokens != 12810 {
		t.Errorf("tokens=%d", cost.InputTokens)
	}
	if cost.TotalCostUSD != 0 {
		t.Errorf("expected zero cost without pricing service, got %v", cost.TotalCostUSD)
	}
	if cost.CostSource != domain.CostSourceUnknown {
		t.Errorf("costSource=%s want unknown", cost.CostSource)
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
	})
	t.Run("completed emits tool_result only", func(t *testing.T) {
		events := codexDecodeOne(t, NewCodexForTest(), codexSamples["command_execution_completed"], "")
		if len(events) != 1 || events[0].EventType != domain.EventTypeToolResult {
			t.Fatalf("got %v", events)
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
	msg := events[0].Data.(*domain.MessageEventData)
	if msg.Content != "Bold and green text" {
		t.Errorf("got %q", msg.Content)
	}
}

func TestCodex_ANSIStrippedFromToolOutput(t *testing.T) {
	line := "{\"type\":\"item.completed\",\"item\":{\"id\":\"item_2\",\"type\":\"tool_result\",\"name\":\"bash\",\"output\":\"\\u001b[32mSuccess\\u001b[0m: file created\"}}"
	events := codexDecodeOne(t, NewCodexForTest(), line, "")
	found := false
	for _, e := range events {
		if r, ok := e.Data.(*domain.ToolResultEventData); ok {
			if r.Output != "Success: file created" {
				t.Errorf("got %q", r.Output)
			}
			found = true
		}
	}
	if !found {
		t.Error("no ToolResultEventData found")
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
		c2, _ := NewCodex(WithCodexPricingService(&mockPricingService{
			pricing: map[string]*PricingCostCalculation{
				"gpt-5.1-codex-mini": {CostSource: "test", Provider: "test"},
			},
		}))
		events := codexDecodeOne(t, c2, codexSamples["turn.completed"], "gpt-5.1-codex-mini")
		c2.UpdateMetrics(events[0], &metrics, &last)
		if metrics.TokensInput != 12810 {
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
	c, _ := NewCodex(WithCodexPricingService(&mockPricingService{
		pricing: map[string]*PricingCostCalculation{
			"gpt-5.1-codex-mini": {CostSource: domain.CostSourcePricingTableEstimate},
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

	if len(allEvents) != 6 {
		t.Errorf("expected 6 events, got %d", len(allEvents))
	}
	expected := []domain.RunEventType{
		domain.EventTypeLog,
		domain.EventTypeLog,
		domain.EventTypeLog,
		domain.EventTypeToolCall,
		domain.EventTypeMessage,
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
	if metrics.TokensInput != 12810 || metrics.TokensOutput != 83 {
		t.Errorf("tokens in/out=%d/%d", metrics.TokensInput, metrics.TokensOutput)
	}
}
