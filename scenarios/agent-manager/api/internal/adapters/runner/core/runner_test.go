package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/runner/codecs"
	"agent-manager/internal/domain"
	"agent-manager/internal/fallback"

	"github.com/google/uuid"
)

// =============================================================================
// fakeLauncher / fakeProcess — controllable stand-in for a real Launcher
// =============================================================================

// fakeLauncher returns a fakeProcess seeded with controllable stdout,
// stderr, exit error, and an optional delay before stdout closes (so
// tests can race the live transcript tail against the wait error).
type fakeLauncher struct {
	mu          sync.Mutex
	lastRequest runner.LaunchRequest
	stdout      string
	stderr      string
	exitErr     error
	pid         int
	earlyExit   chan struct{}
}

func (f *fakeLauncher) Launch(ctx context.Context, req runner.LaunchRequest) (runner.LaunchedProcess, error) {
	f.mu.Lock()
	f.lastRequest = req
	f.mu.Unlock()

	stdoutR, stdoutW := io.Pipe()
	stderrR, stderrW := io.Pipe()
	stdoutClosed := make(chan struct{})

	go func() {
		defer close(stdoutClosed)
		_, _ = io.WriteString(stdoutW, f.stdout)
		if f.earlyExit != nil {
			<-f.earlyExit
		}
		_ = stdoutW.Close()
	}()
	go func() {
		if f.stderr != "" {
			_, _ = io.WriteString(stderrW, f.stderr)
		}
		_ = stderrW.Close()
	}()

	return &fakeProcess{
		stdout:       stdoutR,
		stderr:       stderrR,
		exitErr:      f.exitErr,
		pid:          f.pid,
		done:         make(chan struct{}),
		stdoutClosed: stdoutClosed,
	}, nil
}

func (f *fakeLauncher) lastRequestSnapshot() runner.LaunchRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.lastRequest
}

type fakeProcess struct {
	stdout  io.ReadCloser
	stderr  io.ReadCloser
	exitErr error
	pid     int

	mu           sync.Mutex
	signaled     bool
	killed       bool
	timedOut     bool
	done         chan struct{}
	stdoutClosed <-chan struct{}
	once         sync.Once
}

func (p *fakeProcess) Stdout() io.Reader { return p.stdout }
func (p *fakeProcess) Stderr() io.Reader { return p.stderr }
func (p *fakeProcess) ResetIdleTimer()   {}
func (p *fakeProcess) TimedOut() bool    { return p.timedOut }
func (p *fakeProcess) Kill() {
	p.mu.Lock()
	p.killed = true
	p.mu.Unlock()
	p.once.Do(func() { close(p.done) })
}

func (p *fakeProcess) Signal(grace time.Duration) {
	p.mu.Lock()
	p.signaled = true
	p.mu.Unlock()
	go func() {
		time.Sleep(grace)
		p.Kill()
	}()
}
func (p *fakeProcess) PID() int { return p.pid }
func (p *fakeProcess) Wait() error {
	// Wait for the fake child to close its producer side, but never consume or
	// close the runner-owned stream. Real launchers leave stream draining to
	// the runner; consuming it here races durable transcript capture.
	if p.stdoutClosed != nil {
		<-p.stdoutClosed
	}
	p.once.Do(func() { close(p.done) })
	return p.exitErr
}

// =============================================================================
// fakeCodec — minimal Codec for exercising the core machinery
// =============================================================================

type fakeState struct {
	sessionID     string
	rateLimit     string
	earlyTermSeen bool
}

func (s *fakeState) SessionID() string { return s.sessionID }

type fakeCodec struct {
	rt              domain.RunnerType
	binaryPath      string
	available       bool
	availableMsg    string
	probeErr        error
	controlErr      error
	expirePhrase    string
	earlyTermPrefix string

	mu           sync.Mutex
	args         []string
	continueArgs []string
}

func newFakeCodec() *fakeCodec {
	return &fakeCodec{
		rt:           domain.RunnerTypeClaudeCode,
		binaryPath:   "/usr/bin/fake-agent",
		available:    true,
		availableMsg: "fake codec available",
	}
}

func (c *fakeCodec) Type() domain.RunnerType { return c.rt }

func (c *fakeCodec) Capabilities() runner.Capabilities {
	return runner.Capabilities{
		SupportsMessages:     true,
		SupportsStreaming:    true,
		SupportsCancellation: true,
		SupportsContinuation: true,
		MaxTurns:             10,
		SupportedModels:      []string{"fake-model"},
	}
}

func (c *fakeCodec) BinaryPath() string                                   { return c.binaryPath }
func (c *fakeCodec) BinaryDescription() string                            { return "fake CLI" }
func (c *fakeCodec) TagEnvKey() string                                    { return "FAKE_AGENT_TAG" }
func (c *fakeCodec) Available(ctx context.Context) (bool, string)         { return c.available, c.availableMsg }
func (c *fakeCodec) ProbeModel(ctx context.Context, modelID string) error { return c.probeErr }
func (c *fakeCodec) ControlArgs(*domain.RunConfig) ([]string, error)      { return nil, c.controlErr }

func (c *fakeCodec) BuildArgs(_ codecs.State, req runner.ExecuteRequest) []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	args := []string{"--print", "--model", req.GetConfig().Model}
	c.args = args
	return args
}

func (c *fakeCodec) BuildContinueArgs(_ codecs.State, req runner.ContinueRequest) []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	args := []string{"--resume", req.SessionID}
	c.continueArgs = args
	return args
}

func (c *fakeCodec) BuildPrompt(prompt string, _ []runner.Attachment) string { return prompt }

func (c *fakeCodec) BuildEnv(tag string, extras map[string]string) []string {
	env := []string{c.TagEnvKey() + "=" + tag}
	for k, v := range extras {
		env = append(env, k+"="+v)
	}
	return env
}

func (c *fakeCodec) ContinueTag(req runner.ContinueRequest) string {
	return "fake-continue-" + req.RunID.String()[:8]
}

func (c *fakeCodec) NewState() codecs.State { return &fakeState{} }

// fakeStreamEvent is the on-the-wire shape DecodeStreamLine consumes.
type fakeStreamEvent struct {
	Type      string `json:"type"`
	Content   string `json:"content,omitempty"`
	SessionID string `json:"session_id,omitempty"`
	RateLimit string `json:"rate_limit,omitempty"`
	Bad       bool   `json:"bad,omitempty"`
}

func (c *fakeCodec) DecodeStreamLine(state codecs.State, runID uuid.UUID, line string) ([]*domain.RunEvent, error) {
	s := state.(*fakeState)
	var ev fakeStreamEvent
	if err := json.Unmarshal([]byte(line), &ev); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	if ev.Bad {
		return nil, errors.New("synthetic decode error")
	}
	if ev.SessionID != "" && s.sessionID == "" {
		s.sessionID = ev.SessionID
	}
	if ev.RateLimit != "" {
		s.rateLimit = ev.RateLimit
		return []*domain.RunEvent{domain.NewLogEvent(runID, "warn", "rate limit detected")}, nil
	}
	switch ev.Type {
	case "message":
		return []*domain.RunEvent{domain.NewMessageEvent(runID, "assistant", ev.Content)}, nil
	case "tool_call":
		return []*domain.RunEvent{domain.NewToolCallEvent(runID, "fake-tool", "tool-1", nil)}, nil
	case "log":
		return []*domain.RunEvent{domain.NewLogEvent(runID, "info", ev.Content)}, nil
	}
	return nil, nil
}

func (c *fakeCodec) ParseTranscriptLine(runID uuid.UUID, line string) runner.TranscriptParseResult {
	events, err := c.DecodeStreamLine(&fakeState{}, runID, strings.TrimSpace(line))
	return runner.TranscriptParseResult{Events: events, Err: err}
}

func (c *fakeCodec) NewTranscriptParser() runner.TranscriptParser {
	return &fakeTranscriptParser{codec: c, state: &fakeState{}}
}

type fakeTranscriptParser struct {
	codec *fakeCodec
	state *fakeState
}

func (p *fakeTranscriptParser) ParseTranscriptLine(runID uuid.UUID, line string) runner.TranscriptParseResult {
	events, err := p.codec.DecodeStreamLine(p.state, runID, strings.TrimSpace(line))
	return runner.TranscriptParseResult{Events: events, Err: err}
}

func (c *fakeCodec) UpdateMetrics(event *domain.RunEvent, metrics *runner.ExecutionMetrics, lastAssistant *string) {
	switch data := event.Data.(type) {
	case *domain.MessageEventData:
		if data.Role == "assistant" {
			*lastAssistant = data.Content
			metrics.TurnsUsed++
		}
	case *domain.ToolCallEventData:
		metrics.ToolCallCount++
	}
}

func (c *fakeCodec) OnEarlyTerminate(state codecs.State, line string) bool {
	if c.earlyTermPrefix == "" {
		return false
	}
	if strings.HasPrefix(line, c.earlyTermPrefix) {
		state.(*fakeState).earlyTermSeen = true
		return true
	}
	return false
}

func (c *fakeCodec) PostClassify(state codecs.State, result *runner.ExecuteResult) {
	if s, ok := state.(*fakeState); ok && s.rateLimit != "" {
		result.Success = false
		result.ExitCode = 429
		result.ErrorMessage = s.rateLimit
		result.Summary = nil
	}
}

func (c *fakeCodec) ClassifyTerminalError(stderr string, exitCode int) *domain.RunnerError {
	if c.expirePhrase == "" || !strings.Contains(stderr, c.expirePhrase) {
		return nil
	}
	return domain.NewRunnerSessionExpiredError(c.Type(), errors.New(stderr))
}

func (c *fakeCodec) Classify(stderr string, exitCode int) *fallback.ClassifiedError {
	if stderr == "" && exitCode == 0 {
		return nil
	}
	return fallback.NewTextClassifier().Classify(fallback.ClassifyInput{
		RunnerType: string(c.Type()),
		Stderr:     stderr,
		ExitCode:   exitCode,
	})
}

func (c *fakeCodec) Labels() codecs.Labels {
	return codecs.Labels{
		StartMessage:         "fake start",
		EndMessage:           "fake end",
		ContinueStartMessage: "fake continue start",
		ContinueEndMessage:   "fake continue end",
	}
}

// =============================================================================
// recordingSink — captures emitted events for assertions
// =============================================================================

type recordingSink struct {
	mu     sync.Mutex
	events []*domain.RunEvent
	closed bool
}

func (s *recordingSink) Emit(event *domain.RunEvent) error {
	s.mu.Lock()
	s.events = append(s.events, event)
	s.mu.Unlock()
	return nil
}

func (s *recordingSink) Close() error {
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()
	return nil
}

func (s *recordingSink) snapshot() []*domain.RunEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*domain.RunEvent, len(s.events))
	copy(out, s.events)
	return out
}

// =============================================================================
// Tests
// =============================================================================

func newRunnerForTest(t *testing.T, codec codecs.Codec, launcher runner.Launcher) *Runner {
	t.Helper()
	r := NewRunner(codec, launcher, nil)
	// Replace selector with one that always returns the test launcher so the
	// per-codec wiring path runs end-to-end.
	r.selector = &fixedSelector{launcher: launcher}
	return r
}

func TestExecuteRejectsControlTranslationFailureBeforeLaunch(t *testing.T) {
	codec := newFakeCodec()
	codec.controlErr = errors.New("unknown canonical tool")
	launcher := &fakeLauncher{}
	r := newRunnerForTest(t, codec, launcher)

	_, err := r.Execute(context.Background(), newExecuteRequest(uuid.New(), "test", &recordingSink{}))
	if err == nil || !strings.Contains(err.Error(), "translate runner controls") {
		t.Fatalf("Execute() error = %v, want control translation failure", err)
	}
	if launcher.lastRequest.Command != "" {
		t.Fatalf("launcher invoked with command %q after control translation failure", launcher.lastRequest.Command)
	}
}

// fixedSelector always returns the same launcher; isolates tests from the
// production host-vs-sandbox routing.
type fixedSelector struct{ launcher runner.Launcher }

func (s *fixedSelector) Pick(ctx context.Context, req runner.ExecuteRequest) runner.Launcher {
	return s.launcher
}

func (s *fixedSelector) PickFor(ctx context.Context, runID uuid.UUID, cfg *domain.RunConfig, sandboxID *uuid.UUID, sink runner.EventSink) runner.Launcher {
	return s.launcher
}
func (s *fixedSelector) SetSandboxLauncherFactory(factory runner.SandboxLauncherFactory) {}

func newExecuteRequest(runID uuid.UUID, prompt string, sink runner.EventSink) runner.ExecuteRequest {
	cfg := domain.DefaultRunConfig()
	cfg.Model = "fake-model"
	return runner.ExecuteRequest{
		RunID:          runID,
		ResolvedConfig: cfg,
		Prompt:         prompt,
		WorkingDir:     "/tmp",
		EventSink:      sink,
	}
}

func TestExecute_StreamingSuccess(t *testing.T) {
	codec := newFakeCodec()
	launcher := &fakeLauncher{
		stdout: `{"type":"message","content":"hello","session_id":"sess-1"}` + "\n" +
			`{"type":"tool_call"}` + "\n",
		exitErr: nil,
	}
	r := newRunnerForTest(t, codec, launcher)
	sink := &recordingSink{}
	runID := uuid.New()

	res, err := r.Execute(context.Background(), newExecuteRequest(runID, "hi", sink))
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !res.Success {
		t.Fatalf("Success=false, got %+v", res)
	}
	if res.SessionID != "sess-1" {
		t.Fatalf("SessionID = %q, want sess-1", res.SessionID)
	}
	if res.Summary == nil || res.Summary.Description != "hello" {
		t.Fatalf("Summary = %+v, want Description=hello", res.Summary)
	}
	if res.Metrics.TurnsUsed != 1 {
		t.Fatalf("TurnsUsed = %d, want 1", res.Metrics.TurnsUsed)
	}
	if res.Metrics.ToolCallCount != 1 {
		t.Fatalf("ToolCallCount = %d, want 1", res.Metrics.ToolCallCount)
	}

	events := sink.snapshot()
	// Expect: status starting, message, tool_call, status complete
	if len(events) < 4 {
		t.Fatalf("expected ≥4 events, got %d: %+v", len(events), events)
	}
	if !sink.closed {
		t.Fatalf("expected sink to be closed on completion")
	}

	if got := launcher.lastRequestSnapshot(); got.Command != "env" {
		t.Fatalf("expected env-wrapped launch, got command=%q", got.Command)
	}
}

func TestExecute_StreamingFailure(t *testing.T) {
	codec := newFakeCodec()
	launcher := &fakeLauncher{
		stdout:  `{"type":"log","content":"oops"}` + "\n",
		stderr:  "fatal: no auth\n",
		exitErr: &fakeExitError{code: 2},
	}
	r := newRunnerForTest(t, codec, launcher)
	sink := &recordingSink{}
	runID := uuid.New()

	res, err := r.Execute(context.Background(), newExecuteRequest(runID, "hi", sink))
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if res.Success {
		t.Fatalf("expected Success=false")
	}
	if res.ExitCode != 2 {
		t.Fatalf("ExitCode = %d, want 2", res.ExitCode)
	}
	if !strings.Contains(res.ErrorMessage, "no auth") {
		t.Fatalf("ErrorMessage = %q, expected to contain stderr", res.ErrorMessage)
	}
}

func TestExecute_ContextCanceled(t *testing.T) {
	codec := newFakeCodec()
	earlyExit := make(chan struct{})
	launcher := &fakeLauncher{
		stdout:    "",
		exitErr:   context.Canceled,
		earlyExit: earlyExit,
	}
	r := newRunnerForTest(t, codec, launcher)
	sink := &recordingSink{}
	runID := uuid.New()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
		close(earlyExit)
	}()

	res, err := r.Execute(ctx, newExecuteRequest(runID, "hi", sink))
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if res.Success {
		t.Fatalf("expected Success=false")
	}
	if !strings.Contains(res.ErrorMessage, "cancel") {
		t.Fatalf("ErrorMessage = %q, expected cancel sentinel", res.ErrorMessage)
	}
}

func TestExecute_DecodeErrorWarnsAndContinues(t *testing.T) {
	codec := newFakeCodec()
	// First line errors, second succeeds — the run should still complete.
	launcher := &fakeLauncher{
		stdout: `{"bad":true}` + "\n" +
			`{"type":"message","content":"recovered"}` + "\n",
	}
	r := newRunnerForTest(t, codec, launcher)
	sink := &recordingSink{}
	runID := uuid.New()

	res, err := r.Execute(context.Background(), newExecuteRequest(runID, "hi", sink))
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected Success=true, got %+v", res)
	}
	if res.Summary.Description != "recovered" {
		t.Fatalf("expected recovered, got %q", res.Summary.Description)
	}

	// Verify a warn log was emitted.
	gotWarn := false
	for _, e := range sink.snapshot() {
		if logData, ok := e.Data.(*domain.LogEventData); ok && logData.Level == "warn" && strings.Contains(logData.Message, "Failed to parse event") {
			gotWarn = true
			break
		}
	}
	if !gotWarn {
		t.Fatalf("expected warn log for parse error")
	}
}

func TestExecute_PostClassifyFlipsResult(t *testing.T) {
	codec := newFakeCodec()
	launcher := &fakeLauncher{
		stdout: `{"type":"message","content":"hi","rate_limit":"too many requests"}` + "\n",
	}
	r := newRunnerForTest(t, codec, launcher)
	sink := &recordingSink{}
	runID := uuid.New()

	res, err := r.Execute(context.Background(), newExecuteRequest(runID, "hi", sink))
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if res.Success {
		t.Fatalf("expected PostClassify to flip Success to false")
	}
	if res.ExitCode != 429 {
		t.Fatalf("expected exit 429, got %d", res.ExitCode)
	}
	if res.ErrorMessage != "too many requests" {
		t.Fatalf("ErrorMessage = %q, want 'too many requests'", res.ErrorMessage)
	}
}

func TestExecute_OnEarlyTerminate(t *testing.T) {
	codec := newFakeCodec()
	codec.earlyTermPrefix = `{"type":"step_finish"`
	launcher := &fakeLauncher{
		stdout: `{"type":"message","content":"first"}` + "\n" +
			`{"type":"step_finish"}` + "\n" +
			`{"type":"message","content":"unreachable"}` + "\n",
	}
	r := newRunnerForTest(t, codec, launcher)
	sink := &recordingSink{}
	runID := uuid.New()

	res, err := r.Execute(context.Background(), newExecuteRequest(runID, "hi", sink))
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if res.Summary.Description != "first" {
		t.Fatalf("expected first, got %q (early-term should block 'unreachable')", res.Summary.Description)
	}
}

func TestContinue_RoutesContinueArgs(t *testing.T) {
	codec := newFakeCodec()
	launcher := &fakeLauncher{
		stdout: `{"type":"message","content":"continued","session_id":"sess-2"}` + "\n",
	}
	r := newRunnerForTest(t, codec, launcher)
	sink := &recordingSink{}
	runID := uuid.New()

	cfg := domain.DefaultRunConfig()
	cfg.Model = "fake-model"
	res, err := r.Continue(context.Background(), runner.ContinueRequest{
		RunID:          runID,
		SessionID:      "sess-1",
		Prompt:         "more",
		WorkingDir:     "/tmp",
		EventSink:      sink,
		ResolvedConfig: cfg,
	})
	if err != nil {
		t.Fatalf("Continue returned error: %v", err)
	}
	if !res.Success {
		t.Fatalf("expected Success=true, got %+v", res)
	}
	if res.SessionID != "sess-2" {
		t.Fatalf("SessionID = %q, want sess-2 (codec captured fresh session)", res.SessionID)
	}

	// Continue must call BuildContinueArgs, not BuildArgs.
	codec.mu.Lock()
	defer codec.mu.Unlock()
	if codec.continueArgs == nil {
		t.Fatalf("expected BuildContinueArgs to be called")
	}
	if codec.continueArgs[0] != "--resume" {
		t.Fatalf("continueArgs[0] = %q, want --resume", codec.continueArgs[0])
	}
}

func TestContinue_EmptySessionReturnsTypedError(t *testing.T) {
	codec := newFakeCodec()
	launcher := &fakeLauncher{}
	r := newRunnerForTest(t, codec, launcher)

	_, err := r.Continue(context.Background(), runner.ContinueRequest{
		RunID:          uuid.New(),
		SessionID:      "",
		ResolvedConfig: domain.DefaultRunConfig(),
	})
	var rerr *domain.RunnerError
	if !errors.As(err, &rerr) {
		t.Fatalf("expected *domain.RunnerError, got %T (%v)", err, err)
	}
	if rerr.Code() != domain.ErrCodeRunnerSessionExpired {
		t.Errorf("Code = %s, want RUNNER_SESSION_EXPIRED", rerr.Code())
	}
}

func TestContinue_ClassifiesSessionExpired(t *testing.T) {
	codec := newFakeCodec()
	codec.expirePhrase = "session expired"
	launcher := &fakeLauncher{
		stdout:  "",
		stderr:  "session expired and gone\n",
		exitErr: &fakeExitError{code: 1},
	}
	r := newRunnerForTest(t, codec, launcher)
	sink := &recordingSink{}

	cfg := domain.DefaultRunConfig()
	cfg.Model = "fake-model"
	_, err := r.Continue(context.Background(), runner.ContinueRequest{
		RunID:          uuid.New(),
		SessionID:      "sess-1",
		WorkingDir:     "/tmp",
		EventSink:      sink,
		ResolvedConfig: cfg,
	})
	var rerr *domain.RunnerError
	if !errors.As(err, &rerr) {
		t.Fatalf("expected *domain.RunnerError, got %T (%v)", err, err)
	}
	if rerr.Code() != domain.ErrCodeRunnerSessionExpired {
		t.Errorf("Code = %s, want RUNNER_SESSION_EXPIRED", rerr.Code())
	}
}

func TestStop_NotFoundForUnknownRun(t *testing.T) {
	r := newRunnerForTest(t, newFakeCodec(), &fakeLauncher{})
	err := r.Stop(context.Background(), uuid.New())
	if err == nil {
		t.Fatalf("expected error for unknown runID, got nil")
	}
}

func TestStop_SignalsRegisteredProcess(t *testing.T) {
	codec := newFakeCodec()
	earlyExit := make(chan struct{})
	launcher := &fakeLauncher{
		stdout:    `{"type":"message","content":"hi"}` + "\n",
		earlyExit: earlyExit,
	}
	r := newRunnerForTest(t, codec, launcher)
	sink := &recordingSink{}
	runID := uuid.New()

	done := make(chan *runner.ExecuteResult, 1)
	go func() {
		res, _ := r.Execute(context.Background(), newExecuteRequest(runID, "hi", sink))
		done <- res
	}()

	// Wait until the runner has registered the process.
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if r.PID(runID) != 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if r.PID(runID) == 0 && len(launcher.stdout) > 0 {
		// PID is zero in fakeProcess; verify the registered process is non-nil instead.
		if r.LaunchedProcess(runID) == nil {
			t.Fatalf("process not registered")
		}
	}

	if err := r.Stop(context.Background(), runID); err != nil {
		t.Fatalf("Stop error: %v", err)
	}
	close(earlyExit)
	<-done
}

func TestRunner_TypeAndCapabilities(t *testing.T) {
	codec := newFakeCodec()
	r := newRunnerForTest(t, codec, &fakeLauncher{})
	if r.Type() != codec.Type() {
		t.Fatalf("Type mismatch")
	}
	if r.Capabilities().MaxTurns != codec.Capabilities().MaxTurns {
		t.Fatalf("Capabilities not threaded through")
	}
}

func TestRunner_IsAvailableUnavailable(t *testing.T) {
	codec := newFakeCodec()
	codec.available = false
	codec.availableMsg = "not installed"
	r := newRunnerForTest(t, codec, &fakeLauncher{})

	available, msg := r.IsAvailable(context.Background())
	if available {
		t.Fatalf("expected not available")
	}
	if msg != "not installed" {
		t.Fatalf("msg = %q, want 'not installed'", msg)
	}

	_, err := r.Execute(context.Background(), newExecuteRequest(uuid.New(), "hi", nil))
	if err == nil {
		t.Fatalf("expected Execute to fail when codec unavailable")
	}
	var rerr *domain.RunnerError
	if !errors.As(err, &rerr) {
		t.Fatalf("expected *domain.RunnerError, got %T", err)
	}
}

func TestRunner_ParseTranscriptLineDelegates(t *testing.T) {
	codec := newFakeCodec()
	r := newRunnerForTest(t, codec, &fakeLauncher{})
	res := r.ParseTranscriptLine(uuid.New(), `{"type":"message","content":"x"}`)
	if res.Err != nil {
		t.Fatalf("Err = %v", res.Err)
	}
	if len(res.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(res.Events))
	}
}

func TestExecute_DurableTranscriptReportsCursorPersistenceFailure(t *testing.T) {
	codec := newFakeCodec()
	launcher := &fakeLauncher{stdout: `{"type":"message","content":"durable evidence","session_id":"session-1"}` + "\n"}
	r := newRunnerForTest(t, codec, launcher)
	sink := &recordingSink{}
	stdout, err := os.CreateTemp(t.TempDir(), "transcript-*.ndjson")
	if err != nil {
		t.Fatal(err)
	}
	defer stdout.Close()
	req := newExecuteRequest(uuid.New(), "triage the failure", sink)
	req.Transcript = &runner.TranscriptConfig{
		TranscriptPath: stdout.Name(),
		StdoutFile:     stdout,
		OnAdvance: func(int64, int64) error {
			return errors.New("cursor store unavailable")
		},
	}
	result, err := r.Execute(context.Background(), req)
	if err != nil || !result.Success {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	var cursorFailure bool
	for _, event := range sink.snapshot() {
		if log, ok := event.Data.(*domain.LogEventData); ok && strings.Contains(log.Message, "transcript cursor persistence failed: cursor store unavailable") {
			cursorFailure = true
		}
	}
	if !cursorFailure {
		t.Fatalf("cursor persistence failure was not emitted: %+v", sink.snapshot())
	}
}

// =============================================================================
// fakeExitError satisfies runner.ExtractExitCode (the exitCoder interface).
// =============================================================================

type fakeExitError struct{ code int }

func (e *fakeExitError) Error() string { return fmt.Sprintf("exit %d", e.code) }
func (e *fakeExitError) ExitCode() int { return e.code }
