package interactive

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/runner/codecs"
	"agent-manager/internal/adapters/webconsole"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

type codexGoalLaunchInfo struct{ fakeLaunchInfo }

func (codexGoalLaunchInfo) NewTranscriptParser() runner.TranscriptParser {
	return codecs.NewCodexForTest().NewTranscriptParser()
}

func (codexGoalLaunchInfo) NativeCodexSettings(cfg *domain.RunConfig) (runner.CodexLaunchSettings, error) {
	return codecs.NewCodexForTest().NativeCodexSettings(cfg)
}

type claudeGoalLaunchInfo struct{ fakeLaunchInfo }

func (claudeGoalLaunchInfo) NewTranscriptParser() runner.TranscriptParser {
	return codecs.NewClaudeForTest().NewTranscriptParser()
}

type goalInstallSessions struct {
	*fakeSessions
	onLaunch            func()
	onGoal              func()
	goalCommand         string
	nativeGoalInstalled bool
}

func (s *goalInstallSessions) GetSession(ctx context.Context, id string) (webconsole.SessionInfo, error) {
	info, err := s.fakeSessions.GetSession(ctx, id)
	info.LaunchMode = "codex_app_server"
	info.ControlMode = "native_capable"
	info.NativeOwner = "fixture-generation"
	info.NativeThreadID = "native-thread-goal"
	return info, err
}
func (s *goalInstallSessions) ObserveNativeAgent(context.Context, string, string) (webconsole.NativeAgentObservation, error) {
	return webconsole.NativeAgentObservation{State: webconsole.NativeAgentIdle, Generation: "fixture-generation", GoalAvailable: s.nativeGoalInstalled, HasGoal: s.nativeGoalInstalled, GoalStatus: "active"}, nil
}
func (s *goalInstallSessions) SetNativeGoalForSession(_ context.Context, _, _, generation, objective, status string, origin webconsole.NativeGoalOrigin, _ *int64) (string, error) {
	s.goalSendCommands++
	if origin != webconsole.NativeGoalAutomatic && origin != webconsole.NativeGoalUserDirected {
		return "", errors.New("fixture requires explicit goal origin")
	}
	if generation != "fixture-generation" {
		return "", errors.New("fixture rejects changed generation")
	}
	if s.sendCommandErr != nil {
		return "", s.sendCommandErr
	}
	s.nativeGoalInstalled = true
	if s.onGoal != nil {
		s.onGoal()
	}
	return status, nil
}

func (s *goalInstallSessions) SendText(ctx context.Context, sessionID, text, source string) error {
	if err := s.fakeSessions.SendText(ctx, sessionID, text, source); err != nil {
		return err
	}
	if strings.HasPrefix(text, "/goal ") {
		if s.onGoal != nil {
			s.onGoal()
		}
	} else if s.onLaunch != nil {
		s.onLaunch()
	}
	return nil
}

func (s *goalInstallSessions) SendCommand(ctx context.Context, sessionID, command, source string) (webconsole.PromptSubmission, error) {
	if strings.HasPrefix(command, "/goal ") {
		s.goalCommand = command
		s.fakeSessions.goalSendCommands++
		if s.fakeSessions.sendCommandErr != nil {
			return webconsole.PromptSubmission{}, s.fakeSessions.sendCommandErr
		}
		if s.onGoal != nil {
			s.onGoal()
		}
		return webconsole.PromptSubmission{Verified: true, EnterPresses: 1}, nil
	}
	return s.fakeSessions.SendCommand(ctx, sessionID, command, source)
}

func (s *goalInstallSessions) SendPrompt(ctx context.Context, sessionID, prompt, source string) (webconsole.PromptSubmission, error) {
	return s.fakeSessions.SendPrompt(ctx, sessionID, prompt, source)
}

func goalInstallTranscript(t *testing.T, path, threadID, objective string, status runner.GoalStatus, markerThreadID string) {
	t.Helper()
	lines := []any{
		map[string]any{"type": "session_meta", "payload": map[string]any{"id": threadID}},
	}
	if status != "" {
		goal := map[string]any{"objective": objective, "status": status}
		if markerThreadID != "" {
			goal["threadId"] = markerThreadID
		}
		lines = append(lines, map[string]any{"type": "event_msg", "payload": map[string]any{"type": "thread_goal_updated", "goal": goal}})
	}
	var body strings.Builder
	for _, line := range lines {
		encoded, err := json.Marshal(line)
		if err != nil {
			t.Fatal(err)
		}
		body.Write(encoded)
		body.WriteByte('\n')
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body.String()), 0o600); err != nil {
		t.Fatal(err)
	}
}

func goalInstallMarkerLine(t *testing.T, markerThreadID, objective string) []byte {
	t.Helper()
	goal := map[string]any{"threadId": markerThreadID, "objective": objective, "status": runner.GoalStatusActive}
	line, err := json.Marshal(map[string]any{"type": "event_msg", "payload": map[string]any{"type": "thread_goal_updated", "goal": goal}})
	if err != nil {
		t.Fatal(err)
	}
	return append(line, '\n')
}

func appendGoalInstallBytes(t *testing.T, path string, body []byte) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := f.Write(body); err != nil {
		t.Fatal(err)
	}
}

func appendGoalInstallMarker(t *testing.T, path, threadID, objective string) {
	t.Helper()
	appendGoalInstallBytes(t, path, goalInstallMarkerLine(t, threadID, objective))
}

// Codex app-server goals use typed RPC. These parser tests retain the legacy
// transcript watcher's independent freshness contract without pretending it is
// the native app-server launch path.
func TestGoalInstallWatcherRequiresFreshCurrentActiveMarker(t *testing.T) {
	const objective = "finish the verified task"
	runDir := t.TempDir()
	transcript := filepath.Join(runDir, "codex", "sessions", "2026", "10", "17", "rollout-current.jsonl")
	goalInstallTranscript(t, transcript, "native-thread-current", objective, runner.GoalStatusActive, "native-thread-current")
	sub := NewSubstrate(newFakeSessions("unused"), nil, WithGoalInstallTimeout(time.Second), WithPollInterval(time.Millisecond))
	params := DiscoverParams{RunnerType: domain.RunnerTypeCodex, RunDir: runDir}
	watcher, err := sub.newGoalInstallWatcher(context.Background(), params, codexGoalLaunchInfo{}, uuid.New(), objective)
	if err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 1)
	go func() { result <- sub.awaitGoalInstalled(context.Background(), params, watcher) }()
	select {
	case err := <-result:
		t.Fatalf("stale prefix acknowledged: %v", err)
	case <-time.After(20 * time.Millisecond):
	}
	appendGoalInstallMarker(t, transcript, "native-thread-current", objective)
	if err := <-result; err != nil {
		t.Fatal(err)
	}
}

func TestGoalInstallWatcherRejectsWrongObjectiveOrThread(t *testing.T) {
	const objective = "finish the verified task"
	for _, tc := range []struct{ name, objective, thread string }{{"wrong objective", "another task", "native-thread-current"}, {"wrong current thread", objective, "native-thread-other"}, {"no active marker", "", ""}} {
		t.Run(tc.name, func(t *testing.T) {
			runDir := t.TempDir()
			transcript := filepath.Join(runDir, "codex", "sessions", "2026", "10", "17", "rollout-current.jsonl")
			goalInstallTranscript(t, transcript, "native-thread-current", "", runner.GoalStatus(""), "")
			sub := NewSubstrate(newFakeSessions("unused"), nil, WithGoalInstallTimeout(20*time.Millisecond), WithPollInterval(time.Millisecond))
			params := DiscoverParams{RunnerType: domain.RunnerTypeCodex, RunDir: runDir}
			watcher, err := sub.newGoalInstallWatcher(context.Background(), params, codexGoalLaunchInfo{}, uuid.New(), objective)
			if err != nil {
				t.Fatal(err)
			}
			if tc.objective != "" {
				appendGoalInstallMarker(t, transcript, tc.thread, tc.objective)
			}
			if err := sub.awaitGoalInstalled(context.Background(), params, watcher); err == nil {
				t.Fatal("accepted absent or mismatched fresh goal")
			}
		})
	}
}

func TestSubstrateLaunch_DoesNotSendTaskWhenNativeGoalRefused(t *testing.T) {
	base := newFakeSessions("web-session")
	base.sendCommandErr = webconsole.ErrCommandNotSubmitted
	fs := &goalInstallSessions{fakeSessions: base}
	sub := NewSubstrate(fs, fakeResolver(codexGoalLaunchInfo{fakeLaunchInfo{rt: domain.RunnerTypeCodex, tagKey: "CODEX_AGENT_TAG", binary: "/fixture/codex"}}), WithHomeDir(t.TempDir()))
	_, err := sub.Launch(context.Background(), LaunchParams{RunID: uuid.New(), RunnerType: domain.RunnerTypeCodex, WorkingDir: "/fixture/work", RunDir: t.TempDir(), Prompt: "perform the task", NativeObjective: "finish the verified task", NativeGoalOrigin: webconsole.NativeGoalAutomatic, Config: &domain.RunConfig{NetworkAccess: domain.NetworkAccessFull}})
	if !errors.Is(err, webconsole.ErrCommandNotSubmitted) {
		t.Fatalf("Launch error = %v", err)
	}
	if fs.goalSendCommands != 1 || fs.sendPrompts != 0 {
		t.Fatalf("goal submissions=%d task submissions=%d", fs.goalSendCommands, fs.sendPrompts)
	}
}

func TestGoalInstallWatcherCompletesPartialSessionPrefix(t *testing.T) {
	const objective = "finish the verified task"
	runDir := t.TempDir()
	transcript := filepath.Join(runDir, "codex", "sessions", "2026", "10", "17", "rollout-current.jsonl")
	if err := os.MkdirAll(filepath.Dir(transcript), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(transcript, []byte(`{"type":"session_meta","payload":{"id":"native-thread-current"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	sub := NewSubstrate(newFakeSessions("unused"), nil, WithGoalInstallTimeout(time.Second), WithPollInterval(time.Millisecond))
	params := DiscoverParams{RunnerType: domain.RunnerTypeCodex, RunDir: runDir}
	watcher, err := sub.newGoalInstallWatcher(context.Background(), params, codexGoalLaunchInfo{}, uuid.New(), objective)
	if err != nil {
		t.Fatal(err)
	}
	appendGoalInstallBytes(t, transcript, []byte("}\n"))
	appendGoalInstallMarker(t, transcript, "native-thread-current", objective)
	if err := sub.awaitGoalInstalled(context.Background(), params, watcher); err != nil {
		t.Fatal(err)
	}
}

func TestGoalInstallWatcherDoesNotAckCompletedOldPartialMarker(t *testing.T) {
	const objective = "finish the verified task"
	runDir := t.TempDir()
	transcript := filepath.Join(runDir, "codex", "sessions", "2026", "10", "17", "rollout-current.jsonl")
	goalInstallTranscript(t, transcript, "native-thread-current", "", runner.GoalStatus(""), "")
	oldMarker := goalInstallMarkerLine(t, "native-thread-current", objective)
	appendGoalInstallBytes(t, transcript, oldMarker[:len(oldMarker)-3])
	sub := NewSubstrate(newFakeSessions("unused"), nil, WithGoalInstallTimeout(time.Second), WithPollInterval(time.Millisecond))
	params := DiscoverParams{RunnerType: domain.RunnerTypeCodex, RunDir: runDir}
	watcher, err := sub.newGoalInstallWatcher(context.Background(), params, codexGoalLaunchInfo{}, uuid.New(), objective)
	if err != nil {
		t.Fatal(err)
	}
	appendGoalInstallBytes(t, transcript, oldMarker[len(oldMarker)-3:])
	result := make(chan error, 1)
	go func() { result <- sub.awaitGoalInstalled(context.Background(), params, watcher) }()
	select {
	case err := <-result:
		t.Fatalf("completed old partial record acknowledged: %v", err)
	case <-time.After(20 * time.Millisecond):
	}
	appendGoalInstallMarker(t, transcript, "native-thread-current", objective)
	if err := <-result; err != nil {
		t.Fatal(err)
	}
}

func TestSubstrateLaunch_ClaudeWaitsForCurrentActiveGoalBeforePrompt(t *testing.T) {
	const objective = "finish the verified Claude task"
	runDir := t.TempDir()
	home := t.TempDir()
	workingDir := "/work/claude-project"
	transcript := filepath.Join(home, ".claude", "projects", SlugifyCwd(workingDir), "claude-session.jsonl")
	goalSent := make(chan struct{}, 1)
	fs := &goalInstallSessions{fakeSessions: newFakeSessions("web-session")}
	fs.onLaunch = func() {
		if err := os.MkdirAll(filepath.Dir(transcript), 0o755); err != nil {
			t.Error(err)
			return
		}
		if err := os.WriteFile(transcript, []byte(`{"type":"mode","sessionId":"claude-session"}`+"\n"), 0o600); err != nil {
			t.Error(err)
		}
	}
	fs.onGoal = func() { goalSent <- struct{}{} }
	sub := NewSubstrate(fs, fakeResolver(claudeGoalLaunchInfo{fakeLaunchInfo{
		rt: domain.RunnerTypeClaudeCode, tagKey: "CLAUDE_CODE_AGENT_TAG", binary: "/usr/bin/claude",
	}}), WithHomeDir(home), WithDiscoveryTimeout(time.Second), WithGoalInstallTimeout(time.Second), WithPollInterval(time.Millisecond), WithPromptBootDelay(0))
	result := make(chan error, 1)
	go func() {
		_, err := sub.Launch(context.Background(), LaunchParams{
			RunID: uuid.New(), RunnerType: domain.RunnerTypeClaudeCode, Tag: "claude-install-order",
			WorkingDir: workingDir, RunDir: runDir, Prompt: "perform the task", NativeObjective: objective,
		})
		result <- err
	}()
	select {
	case <-goalSent:
	case err := <-result:
		t.Fatalf("Claude Launch exited before sending the goal: %v", err)
	case <-time.After(time.Second):
		t.Fatal("Claude Launch did not submit the native goal")
	}
	if fs.sendPrompts != 0 {
		t.Fatalf("Claude task prompt was submitted before its active goal marker: %d submissions", fs.sendPrompts)
	}
	marker, err := json.Marshal(map[string]any{"type": "attachment", "attachment": map[string]any{
		"type": "goal_status", "condition": objective, "met": false,
	}})
	if err != nil {
		t.Fatal(err)
	}
	appendGoalInstallBytes(t, transcript, append(marker, '\n'))
	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("Claude Launch after matching active marker: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Claude Launch did not continue after matching active goal marker")
	}
	if fs.sendPrompts != 1 {
		t.Fatalf("Claude prompt submissions = %d, want 1 after marker", fs.sendPrompts)
	}
}
