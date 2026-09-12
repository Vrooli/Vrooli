package codecs

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/fallback"

	"github.com/google/uuid"
)

// grokDecodeTrace replays every line of a testdata trace through a single
// codec/state and returns the emitted events. It mirrors the live scan loop.
func grokDecodeTrace(t *testing.T, name string) (*Grok, []*domain.RunEvent, State) {
	t.Helper()
	c := NewGrokForTest()
	state := c.NewState()
	raw, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	var events []*domain.RunEvent
	runID := uuid.New()
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		evs, err := c.DecodeStreamLine(state, runID, line)
		if err != nil {
			t.Fatalf("decode line %q: %v", line, err)
		}
		events = append(events, evs...)
	}
	return c, events, state
}

func TestGrok_DecodeStreamLine_FromTrace(t *testing.T) {
	_, events, state := grokDecodeTrace(t, "grok_trace.jsonl")

	// The trace is thought* + text* + end. The decoder accumulates text and
	// flushes exactly one assistant message on the terminal "end" event.
	var messages []*domain.MessageEventData
	for _, e := range events {
		if m, ok := e.Data.(*domain.MessageEventData); ok {
			messages = append(messages, m)
		}
	}
	if len(messages) != 1 {
		t.Fatalf("expected exactly 1 assistant message, got %d (events=%d)", len(messages), len(events))
	}
	if messages[0].Role != "assistant" {
		t.Errorf("role=%q want assistant", messages[0].Role)
	}
	if messages[0].Content != "pong" {
		t.Errorf("content=%q want %q", messages[0].Content, "pong")
	}
	// Session id is carried on the terminal "end" event and captured on state.
	if state.SessionID() != "019f10db-c647-7d93-9278-8bd0ab1e7528" {
		t.Errorf("sessionID=%q", state.SessionID())
	}
}

func TestGrok_DecodeStreamLine_Resume_CapturesSessionAndMessage(t *testing.T) {
	_, events, state := grokDecodeTrace(t, "grok_resume_trace.jsonl")
	if state.SessionID() != "019f10dc-9f9c-7982-a3a8-2b142cbe23ba" {
		t.Errorf("resume sessionID=%q", state.SessionID())
	}
	var got string
	for _, e := range events {
		if m, ok := e.Data.(*domain.MessageEventData); ok && m.Role == "assistant" {
			got = m.Content
		}
	}
	if !strings.Contains(got, "42") {
		t.Errorf("resume assistant message %q should mention the remembered number", got)
	}
}

func TestGrok_DecodeStreamLine_ErrorEvent(t *testing.T) {
	_, events, _ := grokDecodeTrace(t, "grok_error_trace.jsonl")
	var errEvt *domain.ErrorEventData
	for _, e := range events {
		if d, ok := e.Data.(*domain.ErrorEventData); ok {
			errEvt = d
		}
	}
	if errEvt == nil {
		t.Fatalf("expected an error event from the error trace, got %d events", len(events))
	}
	if !strings.Contains(errEvt.Message, "unknown model id") {
		t.Errorf("error message=%q", errEvt.Message)
	}
}

func TestGrok_DecodeStreamLine_ThoughtsNotEmitted(t *testing.T) {
	// The trivial trace has many "thought" lines; none should surface as
	// events (only the single assistant message + nothing else).
	_, events, _ := grokDecodeTrace(t, "grok_trace.jsonl")
	for _, e := range events {
		if _, ok := e.Data.(*domain.MessageEventData); ok {
			continue
		}
		t.Errorf("unexpected non-message event %T from a thought/text/end stream", e.Data)
	}
}

func TestGrok_BuildArgs(t *testing.T) {
	c := NewGrokForTest()
	cfg := &domain.RunConfig{
		RunnerType:           domain.RunnerTypeGrok,
		Model:                "grok-build",
		MaxTurns:             7,
		SkipPermissionPrompt: true,
	}
	req := runner.ExecuteRequest{
		RunID:          uuid.New(),
		ResolvedConfig: cfg,
		Prompt:         "do the thing",
		WorkingDir:     "/work/dir",
	}
	args := c.BuildArgs(c.NewState(), req)
	joined := strings.Join(args, " ")

	wantPairs := [][2]string{
		{"-p", "do the thing"},
		{"--output-format", "streaming-json"},
		{"-m", "grok-build"},
		{"--max-turns", "7"},
		{"--cwd", "/work/dir"},
	}
	for _, p := range wantPairs {
		if !argPairPresent(args, p[0], p[1]) {
			t.Errorf("expected %s %q in args: %v", p[0], p[1], args)
		}
	}
	if !strings.Contains(joined, "--always-approve") {
		t.Errorf("SkipPermissionPrompt should map to --always-approve: %v", args)
	}
}

func TestGrokControlArgsRejectsUnmappedCanonicalTool(t *testing.T) {
	_, err := NewGrokForTest().ControlArgs(&domain.RunConfig{AllowedTools: []string{"not-a-canonical-tool"}})
	if err == nil || !strings.Contains(err.Error(), "no native mapping") {
		t.Fatalf("expected unmapped tool error, got %v", err)
	}
}

func TestGrok_BuildArgs_DefaultMaxTurns_NoApproveWhenNotSkipped(t *testing.T) {
	c := NewGrokForTest()
	req := runner.ExecuteRequest{
		RunID:          uuid.New(),
		ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeGrok},
		Prompt:         "hi",
	}
	args := c.BuildArgs(c.NewState(), req)
	if !argPairPresent(args, "--max-turns", "30") {
		t.Errorf("expected default --max-turns 30: %v", args)
	}
	for _, a := range args {
		if a == "--always-approve" {
			t.Errorf("must not auto-approve when SkipPermissionPrompt is false: %v", args)
		}
		if a == "-m" {
			t.Errorf("must not pass -m when model is empty: %v", args)
		}
	}
}

func TestGrok_BuildArgs_ExtraFlags(t *testing.T) {
	c := NewGrokForTest()
	req := runner.ExecuteRequest{
		RunID: uuid.New(),
		ResolvedConfig: &domain.RunConfig{
			RunnerType: domain.RunnerTypeGrok,
			ExtraFlags: domain.RunnerExtraFlags{domain.RunnerTypeGrok: {"--effort", "high"}},
		},
		Prompt: "hi",
	}
	args := c.BuildArgs(c.NewState(), req)
	if !argPairPresent(args, "--effort", "high") {
		t.Errorf("expected grok extra flags appended: %v", args)
	}
}

func TestGrok_BuildContinueArgs(t *testing.T) {
	c := NewGrokForTest()
	req := runner.ContinueRequest{
		RunID:          uuid.New(),
		SessionID:      "sess-123",
		Prompt:         "and then?",
		WorkingDir:     "/w",
		ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeGrok, Model: "grok-build", SkipPermissionPrompt: true},
	}
	args := c.BuildContinueArgs(c.NewState(), req)
	if !argPairPresent(args, "--resume", "sess-123") {
		t.Errorf("expected --resume sess-123: %v", args)
	}
	if !argPairPresent(args, "-p", "and then?") {
		t.Errorf("expected -p prompt: %v", args)
	}
	if !argPairPresent(args, "--output-format", "streaming-json") {
		t.Errorf("expected streaming-json: %v", args)
	}
}

func TestGrok_BuildPrompt_AlwaysEmpty(t *testing.T) {
	c := NewGrokForTest()
	if got := c.BuildPrompt("user prompt", nil); got != "" {
		t.Errorf("expected empty (prompt is on the -p CLI flag), got %q", got)
	}
}

func TestGrok_BuildEnv_Tag(t *testing.T) {
	c := NewGrokForTest()
	env := c.BuildEnv("grok-tag-1", nil)
	found := false
	for _, e := range env {
		if e == "GROK_AGENT_TAG=grok-tag-1" {
			found = true
		}
	}
	if !found {
		t.Errorf("missing GROK_AGENT_TAG in env: %v", env)
	}
}

func TestGrok_ContinueTag(t *testing.T) {
	c := NewGrokForTest()
	id := uuid.New()
	tag := c.ContinueTag(runner.ContinueRequest{RunID: id})
	if want := "grok-continue-" + id.String()[:8]; tag != want {
		t.Errorf("tag=%q want %q", tag, want)
	}
}

func TestGrok_Classify_SessionExpired(t *testing.T) {
	c := NewGrokForTest()
	stderr := "Error: Failed to restore session from remote: fetching session record: session get failed: 404 Not Found"

	if got := c.Classify(stderr, 1); got == nil || got.Reason != fallback.ReasonSessionExpired {
		t.Errorf("Classify reason=%v want session_expired", got)
	}
	if got := c.ClassifyTerminalError(stderr, 1); got == nil {
		t.Errorf("ClassifyTerminalError should recognise session-expired stderr")
	}
}

func TestGrok_Classify_BadModelDelegatesToTextClassifier(t *testing.T) {
	c := NewGrokForTest()
	stderr := "Error: Couldn't set model 'no-such-model-xyz': Invalid params: \"unknown model id\". Run 'grok models' to see available models."
	got := c.Classify(stderr, 1)
	if got == nil {
		t.Fatalf("expected a classified error for a bad-model failure")
	}
	// Not session-expired; delegated to the residual classifier.
	if got.Reason == fallback.ReasonSessionExpired {
		t.Errorf("bad-model must not classify as session_expired")
	}
	// A clean run is never classified.
	if c.Classify("", 0) != nil {
		t.Errorf("Classify must return nil for stderr=\"\" exitCode=0")
	}
	// ClassifyTerminalError returns nil for a non-session failure.
	if c.ClassifyTerminalError(stderr, 1) != nil {
		t.Errorf("ClassifyTerminalError should not type a bad-model failure")
	}
}

func TestGrok_Available_ForTest(t *testing.T) {
	c := NewGrokForTest()
	ok, msg := c.Available(context.Background())
	if ok {
		t.Errorf("ForTest codec must report unavailable")
	}
	if msg != "test grok codec" {
		t.Errorf("msg=%q", msg)
	}
}

func TestGrok_Type(t *testing.T) {
	if got := NewGrokForTest().Type(); got != domain.RunnerTypeGrok {
		t.Errorf("Type=%q want grok", got)
	}
}

// argPairPresent reports whether flag immediately precedes value in args.
func argPairPresent(args []string, flag, value string) bool {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == flag && args[i+1] == value {
			return true
		}
	}
	return false
}
