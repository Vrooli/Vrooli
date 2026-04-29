package codecs

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// =============================================================================
// Test fixtures
// =============================================================================

// opencodeSamples carries representative OpenCode JSON output shapes.
var opencodeSamples = map[string]string{
	"step_start":            `{"type":"step_start","sessionID":"sess-1","part":{"type":"step-start","sessionID":"sess-1"}}`,
	"text":                  `{"type":"text","sessionID":"sess-1","part":{"type":"text","text":"Hello there"}}`,
	"tool_use_in_progress":  `{"type":"tool_use","sessionID":"sess-1","part":{"type":"tool","tool":"write","callID":"call-1","state":{"status":"pending","input":{"path":"/tmp/x.txt","content":"hi"}}}}`,
	"tool_use_completed":    `{"type":"tool_use","sessionID":"sess-1","part":{"type":"tool","tool":"write","callID":"call-1","state":{"status":"completed","input":{"path":"/tmp/x.txt"},"output":"file written"}}}`,
	"step_finish_running":   `{"type":"step_finish","sessionID":"sess-1","part":{"type":"step-finish","reason":"continue","tokens":{"input":50,"output":10},"cost":0.001}}`,
	"step_finish_terminal":  `{"type":"step_finish","sessionID":"sess-1","part":{"type":"step-finish","reason":"stop","text":"All done.","tokens":{"input":120,"output":40,"cache":{"read":5,"write":3}},"cost":0.0042}}`,
	"step_finish_error":     `{"type":"step_finish","sessionID":"sess-1","part":{"type":"step-finish","reason":"error","output":"opencode crashed","tokens":{"input":1,"output":0}}}`,
	"step_finish_cancelled": `{"type":"step_finish","sessionID":"sess-1","part":{"type":"step-finish","reason":"cancelled","tokens":{"input":1,"output":0}}}`,
	"error_top":             `{"type":"error","error":{"code":"AUTH","message":"Invalid API key"}}`,
	"error_in_part":         `{"type":"error","sessionID":"sess-1","part":{"type":"error","isError":true,"output":"runtime failure"}}`,
	"thinking":              `{"type":"thinking","sessionID":"sess-1","part":{"type":"thinking","text":"working through it…"}}`,
	"user_message":          `{"type":"user_message","sessionID":"sess-1","part":{"type":"user","text":"please retry"}}`,
}

func opencodeDecode(t *testing.T, c *OpenCode, line string) ([]*domain.RunEvent, *opencodeState) {
	t.Helper()
	state := c.NewState().(*opencodeState)
	events, err := c.DecodeStreamLine(state, uuid.New(), line)
	if err != nil {
		t.Fatalf("DecodeStreamLine err: %v", err)
	}
	return events, state
}

// =============================================================================
// BuildArgs / BuildEnv / BuildContinueArgs
// =============================================================================

func TestOpenCode_BuildArgs_BasicShape(t *testing.T) {
	c := NewOpenCodeForTest()
	args := c.BuildArgs(c.NewState(), runner.ExecuteRequest{
		RunID:  uuid.New(),
		Prompt: "do the thing",
		ResolvedConfig: &domain.RunConfig{
			RunnerType: domain.RunnerTypeOpenCode,
			Model:      "anthropic/claude-sonnet-4-5",
		},
	})
	// First three args are: "run", "run", <prompt>
	if args[0] != "run" || args[1] != "run" {
		t.Errorf("expected `run run` prefix, got %v", args[:2])
	}
	if args[2] != "do the thing" {
		t.Errorf("expected prompt at index 2, got %q", args[2])
	}
	hasFormat, hasModel := false, false
	for i, a := range args {
		if a == "--format" && i+1 < len(args) && args[i+1] == "json" {
			hasFormat = true
		}
		if a == "--model" && i+1 < len(args) && args[i+1] == "anthropic/claude-sonnet-4-5" {
			hasModel = true
		}
	}
	if !hasFormat {
		t.Error("missing --format json")
	}
	if !hasModel {
		t.Error("missing --model")
	}
}

func TestOpenCode_BuildArgs_WrapsSystemPrompt(t *testing.T) {
	c := NewOpenCodeForTest()
	args := c.BuildArgs(c.NewState(), runner.ExecuteRequest{
		RunID:        uuid.New(),
		Prompt:       "user data",
		SystemPrompt: "you are a code reviewer",
	})
	// Effective prompt should include both pieces wrapped in tags.
	prompt := args[2]
	if !strings.Contains(prompt, "<system-instructions>") || !strings.Contains(prompt, "user data") {
		t.Errorf("expected wrapped prompt, got %q", prompt)
	}
}

func TestOpenCode_BuildContinueArgs(t *testing.T) {
	c := NewOpenCodeForTest()
	args := c.BuildContinueArgs(c.NewState(), runner.ContinueRequest{
		RunID: uuid.New(), SessionID: "sess-abc", Prompt: "follow up",
	})
	want := []string{"run", "run", "follow up", "--session", "sess-abc", "--format", "json"}
	if len(args) != len(want) {
		t.Fatalf("len=%d want=%d", len(args), len(want))
	}
	for i := range want {
		if args[i] != want[i] {
			t.Errorf("[%d]=%q want %q", i, args[i], want[i])
		}
	}
}

func TestOpenCode_BuildEnv_NonInteractive(t *testing.T) {
	c := NewOpenCodeForTest()
	env := c.BuildEnv("opencode-tag-1", nil)
	hasTag, hasNonInt := false, false
	for _, e := range env {
		if e == "OPENCODE_AGENT_TAG=opencode-tag-1" {
			hasTag = true
		}
		if e == "OPENCODE_NON_INTERACTIVE=true" {
			hasNonInt = true
		}
	}
	if !hasTag {
		t.Error("missing OPENCODE_AGENT_TAG")
	}
	if !hasNonInt {
		t.Error("missing OPENCODE_NON_INTERACTIVE")
	}
}

func TestOpenCode_BuildPrompt_AlwaysEmpty(t *testing.T) {
	c := NewOpenCodeForTest()
	if got := c.BuildPrompt("user prompt", nil); got != "" {
		t.Errorf("expected empty (prompt is on CLI), got %q", got)
	}
}

func TestOpenCode_ContinueTag(t *testing.T) {
	c := NewOpenCodeForTest()
	id := uuid.New()
	tag := c.ContinueTag(runner.ContinueRequest{RunID: id})
	if tag != "opencode-continue-"+id.String()[:8] {
		t.Errorf("tag=%q", tag)
	}
}

// =============================================================================
// Stream parsing
// =============================================================================

func TestOpenCode_DecodeStreamLine_StepStart(t *testing.T) {
	events, _ := opencodeDecode(t, NewOpenCodeForTest(), opencodeSamples["step_start"])
	if len(events) != 1 || events[0].EventType != domain.EventTypeLog {
		t.Fatalf("got %v", events)
	}
}

func TestOpenCode_DecodeStreamLine_TextEmitsAssistantMessage(t *testing.T) {
	events, _ := opencodeDecode(t, NewOpenCodeForTest(), opencodeSamples["text"])
	if len(events) != 1 {
		t.Fatalf("got %d events", len(events))
	}
	msg := events[0].Data.(*domain.MessageEventData)
	if msg.Role != "assistant" || msg.Content != "Hello there" {
		t.Errorf("got role=%s content=%q", msg.Role, msg.Content)
	}
}

func TestOpenCode_DecodeStreamLine_ToolUseInProgress(t *testing.T) {
	events, _ := opencodeDecode(t, NewOpenCodeForTest(), opencodeSamples["tool_use_in_progress"])
	if len(events) != 1 {
		t.Fatalf("got %d events", len(events))
	}
	tool := events[0].Data.(*domain.ToolCallEventData)
	if tool.ToolName != "write" {
		t.Errorf("tool=%s", tool.ToolName)
	}
}

func TestOpenCode_DecodeStreamLine_ToolUseCompleted_EmitsCallAndResult(t *testing.T) {
	events, _ := opencodeDecode(t, NewOpenCodeForTest(), opencodeSamples["tool_use_completed"])
	if len(events) != 2 {
		t.Fatalf("got %d events", len(events))
	}
	if events[0].EventType != domain.EventTypeToolCall || events[1].EventType != domain.EventTypeToolResult {
		t.Errorf("event types: %s, %s", events[0].EventType, events[1].EventType)
	}
	r := events[1].Data.(*domain.ToolResultEventData)
	if r.Output != "file written" {
		t.Errorf("output=%q", r.Output)
	}
}

func TestOpenCode_DecodeStreamLine_StepFinish_NonTerminal(t *testing.T) {
	events, state := opencodeDecode(t, NewOpenCodeForTest(), opencodeSamples["step_finish_running"])
	if state.stepTermina {
		t.Error("non-terminal reason should not flag stepTermina")
	}
	hasCost := false
	for _, e := range events {
		if _, ok := e.Data.(*domain.CostEventData); ok {
			hasCost = true
		}
	}
	if !hasCost {
		t.Error("expected cost event")
	}
}

func TestOpenCode_DecodeStreamLine_StepFinish_Terminal_FlagsState(t *testing.T) {
	c := NewOpenCodeForTest()
	events, state := opencodeDecode(t, c, opencodeSamples["step_finish_terminal"])
	if !state.stepTermina {
		t.Error("expected stepTermina=true for reason=stop")
	}
	// Should produce both cost and message events.
	hasCost, hasMsg := false, false
	for _, e := range events {
		if _, ok := e.Data.(*domain.CostEventData); ok {
			hasCost = true
		}
		if md, ok := e.Data.(*domain.MessageEventData); ok && md.Content == "All done." {
			hasMsg = true
		}
	}
	if !hasCost {
		t.Error("expected cost event")
	}
	if !hasMsg {
		t.Error("expected message event from step_finish.text")
	}
	// OnEarlyTerminate must reflect the flag.
	if !c.OnEarlyTerminate(state, "") {
		t.Error("OnEarlyTerminate should return true after terminal step_finish")
	}
}

func TestOpenCode_DecodeStreamLine_StepFinish_Error(t *testing.T) {
	_, state := opencodeDecode(t, NewOpenCodeForTest(), opencodeSamples["step_finish_error"])
	if !state.stepTermina {
		t.Error("expected stepTermina=true for reason=error")
	}
}

func TestOpenCode_DecodeStreamLine_StepFinish_Cancelled(t *testing.T) {
	_, state := opencodeDecode(t, NewOpenCodeForTest(), opencodeSamples["step_finish_cancelled"])
	if !state.stepTermina {
		t.Error("expected stepTermina=true for reason=cancelled")
	}
}

func TestOpenCode_DecodeStreamLine_TopLevelError(t *testing.T) {
	events, _ := opencodeDecode(t, NewOpenCodeForTest(), opencodeSamples["error_top"])
	if len(events) != 1 || events[0].EventType != domain.EventTypeError {
		t.Fatalf("got %v", events)
	}
	errData := events[0].Data.(*domain.ErrorEventData)
	if errData.Code != "AUTH" {
		t.Errorf("code=%s", errData.Code)
	}
	if errData.Message != "Invalid API key" {
		t.Errorf("msg=%s", errData.Message)
	}
}

func TestOpenCode_DecodeStreamLine_ErrorInPart(t *testing.T) {
	events, _ := opencodeDecode(t, NewOpenCodeForTest(), opencodeSamples["error_in_part"])
	if len(events) != 1 || events[0].EventType != domain.EventTypeError {
		t.Fatalf("got %v", events)
	}
	errData := events[0].Data.(*domain.ErrorEventData)
	if errData.Message != "runtime failure" {
		t.Errorf("msg=%s", errData.Message)
	}
}

func TestOpenCode_DecodeStreamLine_Thinking(t *testing.T) {
	events, _ := opencodeDecode(t, NewOpenCodeForTest(), opencodeSamples["thinking"])
	if len(events) != 1 || events[0].EventType != domain.EventTypeLog {
		t.Fatalf("got %v", events)
	}
	logData := events[0].Data.(*domain.LogEventData)
	if logData.Level != "debug" {
		t.Errorf("level=%s", logData.Level)
	}
	if !strings.HasPrefix(logData.Message, "Thinking:") {
		t.Errorf("message=%q", logData.Message)
	}
}

func TestOpenCode_DecodeStreamLine_UserMessage(t *testing.T) {
	events, _ := opencodeDecode(t, NewOpenCodeForTest(), opencodeSamples["user_message"])
	if len(events) != 1 {
		t.Fatalf("got %d events", len(events))
	}
	msg := events[0].Data.(*domain.MessageEventData)
	if msg.Role != "user" {
		t.Errorf("role=%s", msg.Role)
	}
}

func TestOpenCode_DecodeStreamLine_NonJsonSilentlySkipped(t *testing.T) {
	c := NewOpenCodeForTest()
	cases := []string{"", "  \t  ", "Loading config..."}
	for _, line := range cases {
		events, err := c.DecodeStreamLine(c.NewState(), uuid.New(), line)
		if err != nil {
			t.Errorf("err for %q: %v", line, err)
		}
		if len(events) != 0 {
			t.Errorf("line %q produced %d events", line, len(events))
		}
	}
}

func TestOpenCode_DecodeStreamLine_ArrayShapes(t *testing.T) {
	c := NewOpenCodeForTest()
	state := c.NewState()
	line := "[" + opencodeSamples["text"] + "," + opencodeSamples["thinking"] + "]"
	events, err := c.DecodeStreamLine(state, uuid.New(), line)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(events) != 2 {
		t.Errorf("expected 2 events, got %d", len(events))
	}
}

func TestOpenCode_DecodeStreamLine_CapturesSessionID(t *testing.T) {
	_, state := opencodeDecode(t, NewOpenCodeForTest(), opencodeSamples["text"])
	if state.sessionID != "sess-1" {
		t.Errorf("sessionID=%q", state.sessionID)
	}
}

// =============================================================================
// PostClassify (log-file fallback)
// =============================================================================

func TestOpenCode_PostClassify_NoOpForSuccess(t *testing.T) {
	c := NewOpenCodeForTest()
	result := &runner.ExecuteResult{Success: true, ErrorMessage: ""}
	c.PostClassify(c.NewState(), result)
	if result.ErrorMessage != "" {
		t.Errorf("expected unchanged ErrorMessage, got %q", result.ErrorMessage)
	}
}

func TestOpenCode_PostClassify_NoOpWhenMessageMeaningful(t *testing.T) {
	c := NewOpenCodeForTest()
	result := &runner.ExecuteResult{Success: false, ErrorMessage: "Specific failure description"}
	c.PostClassify(c.NewState(), result)
	if result.ErrorMessage != "Specific failure description" {
		t.Errorf("PostClassify clobbered meaningful message: %q", result.ErrorMessage)
	}
}

// =============================================================================
// DetectSessionExpiry
// =============================================================================

func TestOpenCode_DetectSessionExpiry(t *testing.T) {
	c := NewOpenCodeForTest()
	cases := []struct {
		msg    string
		expect bool
	}{
		{"session not found", true},
		{"session expired", true},
		{"session is invalid", true},
		{"thread missing", false},
		{"random error", false},
	}
	for _, tc := range cases {
		if got := c.DetectSessionExpiry(tc.msg); got != tc.expect {
			t.Errorf("DetectSessionExpiry(%q)=%v want %v", tc.msg, got, tc.expect)
		}
	}
}

// =============================================================================
// UpdateMetrics
// =============================================================================

func TestOpenCode_UpdateMetrics(t *testing.T) {
	c := NewOpenCodeForTest()
	metrics := runner.ExecutionMetrics{}
	last := ""

	t.Run("MessageEvent", func(t *testing.T) {
		c.UpdateMetrics(domain.NewMessageEvent(uuid.New(), "assistant", "ok"), &metrics, &last)
		if metrics.TurnsUsed != 1 {
			t.Errorf("TurnsUsed=%d", metrics.TurnsUsed)
		}
	})

	t.Run("CostEvent", func(t *testing.T) {
		ev := &domain.RunEvent{Data: &domain.CostEventData{InputTokens: 100, OutputTokens: 50, TotalCostUSD: 0.01}}
		c.UpdateMetrics(ev, &metrics, &last)
		if metrics.TokensInput != 100 || metrics.TokensOutput != 50 {
			t.Errorf("tokens=%d/%d", metrics.TokensInput, metrics.TokensOutput)
		}
	})
}

// =============================================================================
// ParseTranscriptLine — terminal extraction
// =============================================================================

func TestOpenCode_ParseTranscriptLine_TerminalSuccess(t *testing.T) {
	c := NewOpenCodeForTest()
	r := c.ParseTranscriptLine(uuid.New(), opencodeSamples["step_finish_terminal"])
	if r.Terminal == nil {
		t.Fatal("expected Terminal")
	}
	if !r.Terminal.Success {
		t.Errorf("success=false")
	}
}

func TestOpenCode_ParseTranscriptLine_TerminalError(t *testing.T) {
	c := NewOpenCodeForTest()
	r := c.ParseTranscriptLine(uuid.New(), opencodeSamples["step_finish_error"])
	if r.Terminal == nil {
		t.Fatal("expected Terminal")
	}
	if r.Terminal.Success {
		t.Errorf("expected failure")
	}
	if r.Terminal.ExitCode != 1 {
		t.Errorf("exitCode=%d want 1", r.Terminal.ExitCode)
	}
}

func TestOpenCode_ParseTranscriptLine_TerminalCancelled(t *testing.T) {
	c := NewOpenCodeForTest()
	r := c.ParseTranscriptLine(uuid.New(), opencodeSamples["step_finish_cancelled"])
	if r.Terminal == nil {
		t.Fatal("expected Terminal")
	}
	if r.Terminal.ExitCode != 130 {
		t.Errorf("exitCode=%d want 130", r.Terminal.ExitCode)
	}
}

// =============================================================================
// JSON unmarshal sanity
// =============================================================================

func TestOpenCodeStreamEvent_Unmarshal(t *testing.T) {
	var ev OpenCodeStreamEvent
	if err := json.Unmarshal([]byte(opencodeSamples["step_finish_terminal"]), &ev); err != nil {
		t.Fatalf("unmarshal err: %v", err)
	}
	if ev.Type != "step_finish" {
		t.Errorf("type=%s", ev.Type)
	}
	if ev.Part == nil || ev.Part.Reason != "stop" || ev.Part.Tokens == nil {
		t.Fatalf("part=%v", ev.Part)
	}
	if ev.Part.Tokens.Input != 120 {
		t.Errorf("tokens.input=%d", ev.Part.Tokens.Input)
	}
}

// =============================================================================
// openCodeLogDir resolution
// =============================================================================

func TestOpenCodeLogDirCanonicalizesContractDescendant(t *testing.T) {
	root := newOpenCodeContractFixtureRepo(t)
	nested := filepath.Join(root, "scenarios", "agent-manager", "api")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	t.Setenv("OPENCODE_XDG_DATA_HOME", "")
	t.Setenv("OPENCODE_DATA_DIR", "")
	t.Setenv("VROOLI_SOURCE_ROOT", nested)
	t.Setenv("VROOLI_ROOT", "")

	got := openCodeLogDir()
	want := filepath.Join(root, "data", "opencode", "xdg-data", "opencode", "log")
	if got != want {
		t.Fatalf("openCodeLogDir() = %q, want %q", got, want)
	}
}

func newOpenCodeContractFixtureRepo(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	repoRoot := opencodeTestRepoRoot(t)
	contractData, err := os.ReadFile(filepath.Join(repoRoot, ".vrooli", "repo-contract.json"))
	if err != nil {
		t.Fatalf("read repo contract: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir .vrooli: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "repo-contract.json"), contractData, 0o644); err != nil {
		t.Fatalf("write repo contract: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/agent-manager-codec-test\n\ngo 1.24.0\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	for _, dir := range []string{"templates", "scenarios", "resources", "packages", "cmd", "internal"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	return root
}

func opencodeTestRepoRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// codecs_test.go lives at scenarios/agent-manager/api/internal/adapters/runner/codecs/
	// repo root is six parents up.
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", "..", "..", "..", "..", ".."))
}

// =============================================================================
// Helpers (sanity checks for static fields)
// =============================================================================

func TestOpenCode_Capabilities_HasContinuation(t *testing.T) {
	c := NewOpenCodeForTest()
	caps := c.Capabilities()
	if !caps.SupportsContinuation {
		t.Error("expected SupportsContinuation")
	}
	if caps.SupportsImageAttachments {
		t.Error("did not expect SupportsImageAttachments")
	}
}

func TestOpenCode_Type_Constant(t *testing.T) {
	c := NewOpenCodeForTest()
	if c.Type() != domain.RunnerTypeOpenCode {
		t.Errorf("type=%s", c.Type())
	}
}
