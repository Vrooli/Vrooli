package codecs

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func TestOpenCode_DecodeStreamLineAcceptsProviderEventAliasesAndFallbackPartTypes(t *testing.T) {
	cases := []struct {
		name      string
		line      string
		wantTypes []domain.RunEventType
	}{
		{"tool-call alias", `{"type":"tool-call","part":{"tool":"read","callID":"c"}}`, []domain.RunEventType{domain.EventTypeToolCall}},
		{"tool-result alias", `{"type":"tool-result","part":{"tool":"read","callID":"c","output":"ok"}}`, []domain.RunEventType{domain.EventTypeToolResult}},
		{"assistant output", `{"type":"assistant","sessionID":"s","part":{"output":"answer"}}`, []domain.RunEventType{domain.EventTypeMessage}},
		{"content user", `{"type":"content","part":{"type":"user","text":"question"}}`, []domain.RunEventType{domain.EventTypeMessage}},
		{"fallback text part", `{"type":"notice","part":{"type":"text","text":"answer"}}`, []domain.RunEventType{domain.EventTypeMessage}},
		{"fallback tool part", `{"type":"notice","part":{"type":"tool","tool":"edit","callID":"c"}}`, []domain.RunEventType{domain.EventTypeToolCall}},
		{"unknown event", `{"type":"notice"}`, []domain.RunEventType{domain.EventTypeLog}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			events, _ := opencodeDecode(t, NewOpenCodeForTest(), tc.line)
			if len(events) != len(tc.wantTypes) {
				t.Fatalf("events=%#v, want %d", events, len(tc.wantTypes))
			}
			for i, want := range tc.wantTypes {
				if events[i].EventType != want {
					t.Fatalf("event[%d] = %s, want %s", i, events[i].EventType, want)
				}
			}
		})
	}
}

func TestOpenCodeToolResultPreservesProviderFailureAndScopeProtection(t *testing.T) {
	runID := uuid.New()
	cases := []struct {
		name       string
		part       *OpenCodePart
		workingDir string
		wantEvents int
		wantOK     bool
	}{
		{"provider error", &OpenCodePart{Tool: "read", CallID: "call", IsError: true, Output: "permission denied"}, "/work", 1, false},
		{"successful state result", &OpenCodePart{Tool: "read", CallID: "call", State: &OpenCodeState{Input: map[string]interface{}{"path": "file"}, Output: "contents"}}, "/work", 2, true},
		{"out of scope write", &OpenCodePart{Tool: "write", CallID: "call", State: &OpenCodeState{Input: map[string]interface{}{"path": "/outside/file"}, Output: "written"}}, "/work", 2, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			events := parseOpenCodeToolResult(runID, tc.part, tc.workingDir)
			if len(events) != tc.wantEvents {
				t.Fatalf("events=%#v", events)
			}
			result := events[len(events)-1].Data.(*domain.ToolResultEventData)
			if result.Success != tc.wantOK {
				t.Fatalf("result=%#v", result)
			}
		})
	}
}

func TestOpenCodeLogErrorResolutionUsesNewestCompleteJSONMessage(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)
	logDir := filepath.Join(dir, "opencode", "log")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		t.Fatal(err)
	}
	oldLog := filepath.Join(logDir, "old.log")
	newLog := filepath.Join(logDir, "new.log")
	if err := os.WriteFile(oldLog, []byte(`{"message":"older failure"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newLog, []byte("noise\n{\"message\":\"first failure\"}\n{\"message\":\"latest \\\"failure\\\"\"}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(newLog, time.Now(), time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if got := resolveOpenCodeLogError(); got != `latest "failure"` {
		t.Fatalf("resolveOpenCodeLogError() = %q", got)
	}
	if got, err := tailFile(newLog, 8); err != nil || got == "" {
		t.Fatalf("tailFile = %q, %v", got, err)
	}
	if got := extractErrorMessage("not json"); got != "" {
		t.Fatalf("extractErrorMessage = %q", got)
	}
}

func TestOpenCodeStateAndSnapshotHelpersPreserveProviderSemantics(t *testing.T) {
	state := NewOpenCodeForTest().NewState()
	if state.SessionID() != "" {
		t.Fatalf("empty state session = %q", state.SessionID())
	}
	parsed := NewOpenCodeForTest().NewTranscriptParser().ParseTranscriptLine(uuid.New(), `{"type":"text","sessionID":"session-1","part":{"text":"hello"}}`)
	if parsed.Err != nil || parsed.SessionID != "session-1" {
		t.Fatalf("parsed = %#v", parsed)
	}
	for value, want := range map[string]bool{
		"": false, strings.Repeat("a", 40): true, strings.Repeat("b", 64): true,
		strings.Repeat("g", 40): false, "useful completion text": false,
	} {
		if got := isLikelyHash(value); got != want {
			t.Errorf("isLikelyHash(%q) = %t, want %t", value, got, want)
		}
	}
}

func TestNewOpenCodeUnavailableBinaryFailsClosedWithoutProbeSideEffects(t *testing.T) {
	t.Setenv("PATH", "")
	codec, err := NewOpenCode()
	if err != nil {
		t.Fatal(err)
	}
	if available, _ := codec.Available(context.Background()); available {
		t.Fatal("OpenCode reported available with an empty PATH")
	}
	if err := codec.ProbeModel(context.Background(), "provider/model"); err == nil {
		t.Fatal("ProbeModel accepted unavailable OpenCode")
	}
	if path := opencodeCatalogPath(); path == "" || !strings.HasSuffix(path, filepath.Join("opencode", "models.json")) {
		t.Fatalf("catalog path = %q", path)
	}
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
	// First two args are: "run", <prompt> (raw binary, no wrapper prefix).
	if args[0] != "run" {
		t.Errorf("expected `run` subcommand, got %q", args[0])
	}
	if args[1] != "do the thing" {
		t.Errorf("expected prompt at index 1, got %q", args[1])
	}
	hasFormat, hasModel, hasPrintLogs := false, false, false
	for i, a := range args {
		if a == "--format" && i+1 < len(args) && args[i+1] == "json" {
			hasFormat = true
		}
		if a == "-m" && i+1 < len(args) && args[i+1] == "anthropic/claude-sonnet-4-5" {
			hasModel = true
		}
		if a == "--print-logs" {
			hasPrintLogs = true
		}
	}
	if !hasFormat {
		t.Error("missing --format json")
	}
	if !hasModel {
		t.Error("missing -m <model>")
	}
	if !hasPrintLogs {
		t.Error("missing --print-logs")
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
	prompt := args[1]
	if !strings.Contains(prompt, "<system-instructions>") || !strings.Contains(prompt, "user data") {
		t.Errorf("expected wrapped prompt, got %q", prompt)
	}
}

func TestOpenCode_BuildContinueArgs(t *testing.T) {
	c := NewOpenCodeForTest()
	args := c.BuildContinueArgs(c.NewState(), runner.ContinueRequest{
		RunID: uuid.New(), SessionID: "sess-abc", Prompt: "follow up",
	})
	want := []string{"run", "follow up", "--session", "sess-abc", "--format", "json", "--print-logs"}
	if len(args) != len(want) {
		t.Fatalf("len=%d want=%d", len(args), len(want))
	}
	for i := range want {
		if args[i] != want[i] {
			t.Errorf("[%d]=%q want %q", i, args[i], want[i])
		}
	}
}

// findDirArg returns the value following "--dir", or "" if the flag is absent.
func findDirArg(args []string) (string, bool) {
	for i, a := range args {
		if a == "--dir" && i+1 < len(args) {
			return args[i+1], true
		}
	}
	return "", false
}

func TestOpenCode_BuildArgs_PinsWorkingDir(t *testing.T) {
	c := NewOpenCodeForTest()
	args := c.BuildArgs(c.NewState(), runner.ExecuteRequest{
		RunID:      uuid.New(),
		Prompt:     "do the thing",
		WorkingDir: "/work/project",
	})
	got, ok := findDirArg(args)
	if !ok {
		t.Fatalf("expected --dir to pin the working directory, got %v", args)
	}
	if got != "/work/project" {
		t.Errorf("--dir = %q, want /work/project", got)
	}
}

func TestOpenCode_BuildArgs_OmitsDirWhenWorkingDirEmpty(t *testing.T) {
	c := NewOpenCodeForTest()
	args := c.BuildArgs(c.NewState(), runner.ExecuteRequest{
		RunID:  uuid.New(),
		Prompt: "do the thing",
	})
	if v, ok := findDirArg(args); ok {
		t.Errorf("expected no --dir when WorkingDir is empty, got %q", v)
	}
}

func TestOpenCode_BuildContinueArgs_PinsWorkingDir(t *testing.T) {
	c := NewOpenCodeForTest()
	args := c.BuildContinueArgs(c.NewState(), runner.ContinueRequest{
		RunID: uuid.New(), SessionID: "sess-abc", Prompt: "follow up",
		WorkingDir: "/work/project",
	})
	got, ok := findDirArg(args)
	if !ok || got != "/work/project" {
		t.Errorf("expected --dir /work/project on resume, got %v", args)
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

func TestOpenCode_PostClassify_NoOpForSuccessWithToolCalls(t *testing.T) {
	c := NewOpenCodeForTest()
	// A genuine success: it executed a tool call that produced a successful
	// result, so its actions actually landed. Left untouched.
	result := &runner.ExecuteResult{
		Success: true,
		Metrics: runner.ExecutionMetrics{ToolCallCount: 1, SuccessfulToolResults: 1},
	}
	c.PostClassify(c.NewState(), result)
	if !result.Success {
		t.Errorf("expected Success to remain true for a run with a successful tool result")
	}
	if result.ErrorMessage != "" {
		t.Errorf("expected unchanged ErrorMessage, got %q", result.ErrorMessage)
	}
}

func TestOpenCode_PostClassify_FlipsToolCallsWithNoSuccessfulResult(t *testing.T) {
	c := NewOpenCodeForTest()
	// Clean exit, the agent emitted a tool call, but nothing landed (no
	// successful tool result) — e.g. a write to a hallucinated path. This is
	// the silent-success hole: exit 0 with no effective work. Reclassify.
	result := &runner.ExecuteResult{
		Success:  true,
		ExitCode: 0,
		Metrics:  runner.ExecutionMetrics{ToolCallCount: 1, SuccessfulToolResults: 0},
	}
	c.PostClassify(c.NewState(), result)
	if result.Success {
		t.Fatal("expected a run with tool calls but no successful result to be reclassified as failure")
	}
	if result.ExitCode != openCodeNoOpExitCode {
		t.Errorf("ExitCode = %d, want %d", result.ExitCode, openCodeNoOpExitCode)
	}
	if !strings.Contains(result.ErrorMessage, "none completed successfully") {
		t.Errorf("expected no-effective-work message, got %q", result.ErrorMessage)
	}
}

func TestOpenCode_PostClassify_FlipsZeroToolCallNoOp(t *testing.T) {
	c := NewOpenCodeForTest()
	// Clean exit but zero tool calls — a no-op, not a success.
	result := &runner.ExecuteResult{
		Success:  true,
		ExitCode: 0,
		Metrics:  runner.ExecutionMetrics{ToolCallCount: 0},
	}
	c.PostClassify(c.NewState(), result)
	if result.Success {
		t.Fatal("expected zero-tool-call run to be reclassified as failure")
	}
	if result.ErrorMessage == "" {
		t.Error("expected a descriptive no-op error message")
	}
}

func TestOpenCode_PostClassify_NoOpMessageHintsModelOnTextToolCall(t *testing.T) {
	c := NewOpenCodeForTest()
	result := &runner.ExecuteResult{
		Success: true,
		Metrics: runner.ExecutionMetrics{ToolCallCount: 0},
		Summary: &domain.RunSummary{
			Description: `{"name":"write","arguments":{"filePath":"hello.txt","content":"hi"}}`,
		},
	}
	c.PostClassify(c.NewState(), result)
	if result.Success {
		t.Fatal("expected no-op reclassification")
	}
	if !strings.Contains(result.ErrorMessage, "tool call as text") {
		t.Errorf("expected model/template hint, got %q", result.ErrorMessage)
	}
}

func TestLooksLikeUnexecutedToolCall(t *testing.T) {
	cases := []struct {
		name string
		text string
		want bool
	}{
		{"bare tool call", `{"name":"write","arguments":{"filePath":"a"}}`, true},
		{"parameters variant", `{"name":"write","parameters":{"filePath":"a"}}`, true},
		{"fenced tool call", "```json\n{\"name\":\"write\",\"arguments\":{}}\n```", true},
		{"plain prose", "Done! I created the file.", false},
		{"json without name", `{"arguments":{"filePath":"a"}}`, false},
		{"not json", "name: write", false},
	}
	for _, tc := range cases {
		if got := looksLikeUnexecutedToolCall(tc.text); got != tc.want {
			t.Errorf("%s: looksLikeUnexecutedToolCall(%q) = %v, want %v", tc.name, tc.text, got, tc.want)
		}
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
// Out-of-dir write guard (defense-in-depth against fabricated absolute paths)
// =============================================================================

func TestToolTargetOutsideDir(t *testing.T) {
	dir := "/work/project"
	cases := []struct {
		name     string
		tool     string
		input    map[string]interface{}
		wantBlnk bool // true => guard returns nil (allowed)
	}{
		{"write inside dir", "write", map[string]interface{}{"filePath": "/work/project/sub/a.txt"}, true},
		{"write the dir itself", "write", map[string]interface{}{"filePath": "/work/project"}, true},
		{"write outside dir", "write", map[string]interface{}{"filePath": "/tmp/elsewhere/a.txt"}, false},
		{"edit outside dir", "edit", map[string]interface{}{"filePath": "/etc/passwd"}, false},
		{"relative path allowed", "write", map[string]interface{}{"filePath": "hello.txt"}, true},
		{"non-mutating read outside dir", "read", map[string]interface{}{"filePath": "/tmp/x"}, true},
		{"bash never guarded", "bash", map[string]interface{}{"command": "ls /tmp"}, true},
		{"no target path", "write", map[string]interface{}{"content": "hi"}, true},
		{"path key variant outside", "patch", map[string]interface{}{"path": "/var/data/x"}, false},
	}
	for _, tc := range cases {
		err := toolTargetOutsideDir(tc.tool, tc.input, dir)
		if tc.wantBlnk && err != nil {
			t.Errorf("%s: expected allowed, got error %v", tc.name, err)
		}
		if !tc.wantBlnk && err == nil {
			t.Errorf("%s: expected guard to reject, got nil", tc.name)
		}
	}
}

func TestToolTargetOutsideDir_EmptyDirDisablesGuard(t *testing.T) {
	if err := toolTargetOutsideDir("write", map[string]interface{}{"filePath": "/tmp/anywhere"}, ""); err != nil {
		t.Errorf("empty workingDir must disable the guard, got %v", err)
	}
}

// TestOpenCode_Decode_OutOfDirWrite_NotSuccessful drives the full decode path:
// BuildArgs pins --dir onto the shared state, then a completed write to a
// fabricated absolute path outside that dir must decode to a FAILED tool
// result (so SuccessfulToolResults stays 0 and PostClassify flips the run).
func TestOpenCode_Decode_OutOfDirWrite_NotSuccessful(t *testing.T) {
	c := NewOpenCodeForTest()
	state := c.NewState().(*opencodeState)
	c.BuildArgs(state, runner.ExecuteRequest{
		RunID:      uuid.New(),
		Prompt:     "write hello",
		WorkingDir: "/work/project",
	})
	line := `{"type":"tool_use","sessionID":"s","part":{"type":"tool","tool":"write","callID":"c1","state":{"status":"completed","input":{"filePath":"/tmp/opencode-smoketown/hello.txt"},"output":"file written"}}}`
	events, err := c.DecodeStreamLine(state, uuid.New(), line)
	if err != nil {
		t.Fatalf("decode err: %v", err)
	}
	var sawResult bool
	for _, ev := range events {
		if data, ok := ev.Data.(*domain.ToolResultEventData); ok {
			sawResult = true
			if data.Success {
				t.Error("expected out-of-dir write to decode as a NON-successful tool result")
			}
			if !strings.Contains(data.Error, "outside the run's working directory") {
				t.Errorf("expected out-of-dir error message, got %q", data.Error)
			}
		}
	}
	if !sawResult {
		t.Fatal("expected a tool result event")
	}
}

func TestOpenCode_Decode_InDirWrite_Successful(t *testing.T) {
	c := NewOpenCodeForTest()
	state := c.NewState().(*opencodeState)
	c.BuildArgs(state, runner.ExecuteRequest{
		RunID:      uuid.New(),
		Prompt:     "write hello",
		WorkingDir: "/work/project",
	})
	line := `{"type":"tool_use","sessionID":"s","part":{"type":"tool","tool":"write","callID":"c1","state":{"status":"completed","input":{"filePath":"/work/project/hello.txt"},"output":"file written"}}}`
	events, err := c.DecodeStreamLine(state, uuid.New(), line)
	if err != nil {
		t.Fatalf("decode err: %v", err)
	}
	for _, ev := range events {
		if data, ok := ev.Data.(*domain.ToolResultEventData); ok && !data.Success {
			t.Errorf("expected in-dir write to remain successful, got error %q", data.Error)
		}
	}
}

// =============================================================================
// ClassifyTerminalError
// =============================================================================

func TestOpenCode_ClassifyTerminalError(t *testing.T) {
	c := NewOpenCodeForTest()
	cases := []struct {
		name     string
		stderr   string
		wantCode domain.ErrorCode // empty = expect nil
	}{
		{"session not found", "session not found", domain.ErrCodeRunnerSessionExpired},
		{"session expired", "session expired", domain.ErrCodeRunnerSessionExpired},
		{"session invalid", "session is invalid", domain.ErrCodeRunnerSessionExpired},
		{"thread missing without session", "thread missing", ""},
		{"random error", "random error", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := c.ClassifyTerminalError(tc.stderr, 1)
			if tc.wantCode == "" {
				if got != nil {
					t.Fatalf("expected nil, got %v", got)
				}
				return
			}
			if got == nil {
				t.Fatalf("expected %s, got nil", tc.wantCode)
			}
			if got.Code() != tc.wantCode {
				t.Errorf("Code() = %s, want %s", got.Code(), tc.wantCode)
			}
		})
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

// TestOpenCode_UpdateMetrics_SuccessfulToolResults asserts SuccessfulToolResults
// increments ONLY on a tool result that reported success — a failed result (the
// shape an out-of-dir/hallucinated write produces) bumps ToolCallCount but not
// SuccessfulToolResults, which is what PostClassify keys the silent-success
// reclassification on.
func TestOpenCode_UpdateMetrics_SuccessfulToolResults(t *testing.T) {
	c := NewOpenCodeForTest()
	metrics := runner.ExecutionMetrics{}
	last := ""

	ok := domain.NewToolResultEvent(uuid.New(), "write", "c1", "file written", nil)
	c.UpdateMetrics(ok, &metrics, &last)
	if metrics.SuccessfulToolResults != 1 {
		t.Errorf("after a successful result, SuccessfulToolResults=%d want 1", metrics.SuccessfulToolResults)
	}

	failed := domain.NewToolResultEvent(uuid.New(), "write", "c2", "", errTestToolFailure)
	c.UpdateMetrics(failed, &metrics, &last)
	if metrics.SuccessfulToolResults != 1 {
		t.Errorf("a failed result must not increment SuccessfulToolResults, got %d", metrics.SuccessfulToolResults)
	}
	if metrics.ToolCallCount != 2 {
		t.Errorf("both results count as tool calls, ToolCallCount=%d want 2", metrics.ToolCallCount)
	}
}

var errTestToolFailure = errTest("tool failed")

type errTest string

func (e errTest) Error() string { return string(e) }

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

func TestOpenCodeLogDirUsesXDGDataHome(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", filepath.Join("custom", "data"))
	got := openCodeLogDir()
	want := filepath.Join("custom", "data", "opencode", "log")
	if got != want {
		t.Fatalf("openCodeLogDir() = %q, want %q", got, want)
	}
}

func TestOpenCodeLogDirDefaultsToHomeLocalShare(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("HOME", home)
	got := openCodeLogDir()
	want := filepath.Join(home, ".local", "share", "opencode", "log")
	if got != want {
		t.Fatalf("openCodeLogDir() = %q, want %q", got, want)
	}
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
	if !caps.SupportsImageAttachments {
		t.Error("expected SupportsImageAttachments (opencode run -f/--file)")
	}
}

func TestOpenCode_Type_Constant(t *testing.T) {
	c := NewOpenCodeForTest()
	if c.Type() != domain.RunnerTypeOpenCode {
		t.Errorf("type=%s", c.Type())
	}
}
