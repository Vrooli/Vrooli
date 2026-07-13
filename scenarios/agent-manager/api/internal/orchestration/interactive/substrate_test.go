package interactive

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/webconsole"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// fakeSessions is an in-memory SessionController for unit tests. It records the
// order of lifecycle calls and lets each be configured to fail.
type fakeSessions struct {
	createID     string
	createErr    error
	interruptErr error
	deleteErr    error
	deleted      map[string]bool

	// sendPromptErr fails every SendPrompt call. onSendPrompt (if set) fires with
	// the 1-based send count before the error check, letting a test create the
	// transcript on a specific attempt (e.g. only on the resend).
	sendPromptErr error
	onSendPrompt  func(callN int)
	sendPrompts   int

	calls []string
}

func newFakeSessions(id string) *fakeSessions {
	return &fakeSessions{createID: id, deleted: map[string]bool{}}
}

func (f *fakeSessions) CreateSession(_ context.Context, _ webconsole.CreateSessionParams) (string, error) {
	f.calls = append(f.calls, "create")
	if f.createErr != nil {
		return "", f.createErr
	}
	return f.createID, nil
}

func (f *fakeSessions) GetSession(_ context.Context, id string) (webconsole.SessionInfo, error) {
	if f.deleted[id] {
		return webconsole.SessionInfo{}, webconsole.ErrSessionNotFound
	}
	return webconsole.SessionInfo{ID: id, Owner: webconsole.OwnerAgentManager}, nil
}

func (f *fakeSessions) DeleteSession(_ context.Context, id string) error {
	f.calls = append(f.calls, "delete")
	if f.deleteErr != nil {
		return f.deleteErr
	}
	f.deleted[id] = true
	return nil
}

func (f *fakeSessions) SendText(_ context.Context, _, _, _ string) error {
	f.calls = append(f.calls, "sendtext")
	return nil
}

func (f *fakeSessions) SendPrompt(_ context.Context, _, _, _ string) error {
	f.calls = append(f.calls, "sendprompt")
	f.sendPrompts++
	if f.onSendPrompt != nil {
		f.onSendPrompt(f.sendPrompts)
	}
	return f.sendPromptErr
}

func (f *fakeSessions) Interrupt(_ context.Context, _, _ string) error {
	f.calls = append(f.calls, "interrupt")
	return f.interruptErr
}

func (f *fakeSessions) Screen(_ context.Context, _ string, _ bool) (string, error) {
	return "", nil
}

// fakeLaunchInfo satisfies runner.AgentLaunchInfo without a full Runner.
type fakeLaunchInfo struct {
	rt     domain.RunnerType
	tagKey string
	binary string
}

func (f fakeLaunchInfo) Type() domain.RunnerType { return f.rt }
func (f fakeLaunchInfo) TagEnvKey() string       { return f.tagKey }
func (f fakeLaunchInfo) BinaryPath() string      { return f.binary }

func fakeResolver(info runner.AgentLaunchInfo) LaunchInfoResolver {
	return func(domain.RunnerType) (runner.AgentLaunchInfo, error) { return info, nil }
}

func TestSubstrateLaunch_Codex_HappyPath(t *testing.T) {
	runDir := t.TempDir()
	// Pre-create the codex rollout the "CLI" would write, so discovery finds it.
	rollout := filepath.Join(runDir, "codex", "sessions", "2026", "07", "13", "rollout-x.jsonl")
	writeFile(t, rollout)

	fs := newFakeSessions("sess-1")
	sub := NewSubstrate(fs, fakeResolver(fakeLaunchInfo{
		rt: domain.RunnerTypeCodex, tagKey: "CODEX_AGENT_TAG", binary: "/usr/bin/codex",
	}), WithDiscoveryTimeout(2*time.Second), WithPollInterval(10*time.Millisecond), WithPromptBootDelay(0))

	res, err := sub.Launch(context.Background(), LaunchParams{
		RunID:      uuid.New(),
		RunnerType: domain.RunnerTypeCodex,
		Tag:        "run-77",
		WorkingDir: "/work/dir",
		RunDir:     runDir,
		Prompt:     "Reply with exactly: pong",
	})
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}
	// The initial prompt is typed into the session before discovery, and the
	// session is created first.
	if fs.sendPrompts != 1 {
		t.Errorf("expected exactly one prompt delivery, got %d", fs.sendPrompts)
	}
	if len(fs.calls) < 2 || fs.calls[0] != "create" || fs.calls[1] != "sendprompt" {
		t.Errorf("expected create then sendprompt, got %v", fs.calls)
	}
	if res.SessionID != "sess-1" {
		t.Errorf("session id: got %q", res.SessionID)
	}
	if res.TranscriptPath != rollout {
		t.Errorf("transcript path: got %q, want %q", res.TranscriptPath, rollout)
	}
	if res.ExecutionMode != domain.ExecutionModeInteractive {
		t.Errorf("execution mode: got %q", res.ExecutionMode)
	}
	if res.LaunchCommand == "" {
		t.Error("launch command should be populated")
	}
	// The relocated CODEX_HOME must have been created.
	if _, err := os.Stat(filepath.Join(runDir, "codex")); err != nil {
		t.Errorf("CODEX_HOME not created: %v", err)
	}

	// ApplyToRun writes durable facts.
	run := &domain.Run{}
	ApplyToRun(run, res)
	if run.ExecutionMode != domain.ExecutionModeInteractive || run.WebConsoleSessionID != "sess-1" || run.TranscriptPath != rollout {
		t.Errorf("ApplyToRun did not set fields: %+v", run)
	}
}

func TestSubstrateLaunch_DiscoveryTimeout_ReturnsSessionID(t *testing.T) {
	fs := newFakeSessions("sess-live")
	sub := NewSubstrate(fs, fakeResolver(fakeLaunchInfo{
		rt: domain.RunnerTypeCodex, tagKey: "CODEX_AGENT_TAG", binary: "/usr/bin/codex",
	}), WithDiscoveryTimeout(30*time.Millisecond), WithPollInterval(5*time.Millisecond))

	res, err := sub.Launch(context.Background(), LaunchParams{
		RunnerType: domain.RunnerTypeCodex,
		Tag:        "run-x",
		WorkingDir: "/work/dir",
		RunDir:     t.TempDir(), // no rollout ever appears
	})
	if err == nil {
		t.Fatal("expected discovery-timeout error")
	}
	// Session id still returned so the caller can persist + tear down.
	if res.SessionID != "sess-live" {
		t.Errorf("expected session id on discovery failure, got %q", res.SessionID)
	}
}

// TestSubstrateLaunch_ResendRecoversDroppedPrompt proves the readiness safety
// net: the first paste lands in a not-yet-rendered TUI and is dropped (no
// transcript), so discovery re-delivers the prompt, and the (simulated) turn the
// second delivery starts writes the rollout, which discovery then finds.
func TestSubstrateLaunch_ResendRecoversDroppedPrompt(t *testing.T) {
	runDir := t.TempDir()
	rollout := filepath.Join(runDir, "codex", "sessions", "2026", "07", "13", "rollout-resend.jsonl")

	fs := newFakeSessions("sess-resend")
	// Only the SECOND send (the resend) "starts a turn" that writes the rollout.
	fs.onSendPrompt = func(n int) {
		if n == 2 {
			writeFile(t, rollout)
		}
	}
	sub := NewSubstrate(fs, fakeResolver(fakeLaunchInfo{
		rt: domain.RunnerTypeCodex, tagKey: "CODEX_AGENT_TAG", binary: "/usr/bin/codex",
	}),
		WithDiscoveryTimeout(2*time.Second),
		WithPollInterval(5*time.Millisecond),
		WithPromptBootDelay(0),
		WithPromptResendAfter(20*time.Millisecond),
	)

	res, err := sub.Launch(context.Background(), LaunchParams{
		RunID:      uuid.New(),
		RunnerType: domain.RunnerTypeCodex,
		Tag:        "run-resend",
		WorkingDir: "/work/dir",
		RunDir:     runDir,
		Prompt:     "do the task",
	})
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}
	if res.TranscriptPath != rollout {
		t.Errorf("transcript path: got %q, want %q", res.TranscriptPath, rollout)
	}
	if fs.sendPrompts != 2 {
		t.Errorf("expected prompt to be sent twice (initial + one resend), got %d", fs.sendPrompts)
	}
}

// TestSubstrateLaunch_NoPromptSkipsDelivery keeps the fake-CLI integration-harness
// contract: an empty prompt creates the session and discovers the transcript
// without any prompt delivery or boot delay.
func TestSubstrateLaunch_NoPromptSkipsDelivery(t *testing.T) {
	runDir := t.TempDir()
	rollout := filepath.Join(runDir, "codex", "sessions", "2026", "07", "13", "rollout-noprompt.jsonl")
	writeFile(t, rollout)

	fs := newFakeSessions("sess-noprompt")
	sub := NewSubstrate(fs, fakeResolver(fakeLaunchInfo{
		rt: domain.RunnerTypeCodex, tagKey: "CODEX_AGENT_TAG", binary: "/usr/bin/codex",
	}), WithDiscoveryTimeout(2*time.Second), WithPollInterval(5*time.Millisecond))

	res, err := sub.Launch(context.Background(), LaunchParams{
		RunID:      uuid.New(),
		RunnerType: domain.RunnerTypeCodex,
		Tag:        "run-noprompt",
		WorkingDir: "/work/dir",
		RunDir:     runDir,
	})
	if err != nil {
		t.Fatalf("Launch: %v", err)
	}
	if res.TranscriptPath != rollout {
		t.Errorf("transcript path: got %q", res.TranscriptPath)
	}
	if fs.sendPrompts != 0 {
		t.Errorf("no prompt should be delivered when Prompt is empty, got %d sends", fs.sendPrompts)
	}
}

// TestSubstrateLaunch_PromptDeliveryErrorReturnsSessionID verifies a hard
// SendPrompt failure is surfaced immediately (not swallowed by the resend path)
// while still returning the session id so the caller can tear the session down.
func TestSubstrateLaunch_PromptDeliveryErrorReturnsSessionID(t *testing.T) {
	fs := newFakeSessions("sess-senderr")
	fs.sendPromptErr = errors.New("paste boom")
	sub := NewSubstrate(fs, fakeResolver(fakeLaunchInfo{
		rt: domain.RunnerTypeCodex, tagKey: "CODEX_AGENT_TAG", binary: "/usr/bin/codex",
	}), WithDiscoveryTimeout(2*time.Second), WithPollInterval(5*time.Millisecond), WithPromptBootDelay(0))

	res, err := sub.Launch(context.Background(), LaunchParams{
		RunnerType: domain.RunnerTypeCodex,
		Tag:        "run-senderr",
		WorkingDir: "/work/dir",
		RunDir:     t.TempDir(),
		Prompt:     "do the task",
	})
	if err == nil {
		t.Fatal("expected prompt-delivery error")
	}
	if !strings.Contains(err.Error(), "initial prompt") {
		t.Errorf("error should identify prompt delivery, got: %v", err)
	}
	if res.SessionID != "sess-senderr" {
		t.Errorf("expected session id on delivery failure, got %q", res.SessionID)
	}
}

func TestSubstrateLaunch_OpenCodeDescoped(t *testing.T) {
	fs := newFakeSessions("x")
	sub := NewSubstrate(fs, fakeResolver(fakeLaunchInfo{rt: domain.RunnerTypeOpenCode}))
	_, err := sub.Launch(context.Background(), LaunchParams{
		RunnerType: domain.RunnerTypeOpenCode, Tag: "t", WorkingDir: "/w", RunDir: t.TempDir(),
	})
	if err == nil {
		t.Fatal("expected opencode Launch to be rejected (descoped)")
	}
	// Must reject before creating any session.
	for _, c := range fs.calls {
		if c == "create" {
			t.Fatal("opencode must not create a session")
		}
	}
}

func TestSubstrateStop_InterruptThenDelete(t *testing.T) {
	fs := newFakeSessions("s")
	sub := NewSubstrate(fs, fakeResolver(fakeLaunchInfo{rt: domain.RunnerTypeCodex}), WithStopGrace(0))
	if err := sub.Stop(context.Background(), "s", "agent-manager:run-1"); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if len(fs.calls) != 2 || fs.calls[0] != "interrupt" || fs.calls[1] != "delete" {
		t.Fatalf("expected interrupt then delete, got %v", fs.calls)
	}
}

func TestSubstrateStop_InterruptFailsButDeleteSucceeds(t *testing.T) {
	fs := newFakeSessions("s")
	fs.interruptErr = errors.New("pane busy")
	sub := NewSubstrate(fs, fakeResolver(fakeLaunchInfo{rt: domain.RunnerTypeCodex}), WithStopGrace(0))
	if err := sub.Stop(context.Background(), "s", "src"); err != nil {
		t.Fatalf("Stop should succeed when delete succeeds even if interrupt failed: %v", err)
	}
	if !fs.deleted["s"] {
		t.Error("session should be deleted as hard-kill fallback")
	}
}

func TestSubstrateStop_BothFailReturnsJoinedError(t *testing.T) {
	fs := newFakeSessions("s")
	fs.interruptErr = errors.New("interrupt boom")
	fs.deleteErr = errors.New("delete boom")
	sub := NewSubstrate(fs, fakeResolver(fakeLaunchInfo{rt: domain.RunnerTypeCodex}), WithStopGrace(0))
	err := sub.Stop(context.Background(), "s", "src")
	if err == nil {
		t.Fatal("expected error when both interrupt and delete fail")
	}
	if !strings.Contains(err.Error(), "interrupt boom") || !strings.Contains(err.Error(), "delete boom") {
		t.Errorf("expected joined error carrying both causes, got: %v", err)
	}
}

func TestSubstrateStop_EmptySessionID(t *testing.T) {
	fs := newFakeSessions("s")
	sub := NewSubstrate(fs, fakeResolver(fakeLaunchInfo{rt: domain.RunnerTypeCodex}))
	if err := sub.Stop(context.Background(), "", "src"); err == nil {
		t.Fatal("expected error for empty session id")
	}
}
