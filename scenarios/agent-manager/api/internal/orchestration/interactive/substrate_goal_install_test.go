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

type claudeGoalLaunchInfo struct{ fakeLaunchInfo }

func (claudeGoalLaunchInfo) NewTranscriptParser() runner.TranscriptParser {
	return codecs.NewClaudeForTest().NewTranscriptParser()
}

type goalInstallSessions struct {
	*fakeSessions
	onLaunch    func()
	onGoal      func()
	goalCommand string
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

func TestSubstrateLaunch_WaitsForCurrentActiveGoalMarkerBeforePrompt(t *testing.T) {
	const objective = "finish the verified task"
	runDir := t.TempDir()
	transcript := filepath.Join(runDir, "codex", "sessions", "2026", "10", "17", "rollout-current.jsonl")
	goalInstallTranscript(t, transcript, "native-thread-current", objective, runner.GoalStatusActive, "native-thread-current")
	goalSent := make(chan struct{}, 1)
	fs := &goalInstallSessions{fakeSessions: newFakeSessions("web-session"), onGoal: func() { goalSent <- struct{}{} }}
	sub := NewSubstrate(fs, fakeResolver(codexGoalLaunchInfo{fakeLaunchInfo{
		rt: domain.RunnerTypeCodex, tagKey: "CODEX_AGENT_TAG", binary: "/usr/bin/codex",
	}}), WithDiscoveryTimeout(time.Second), WithGoalInstallTimeout(time.Second), WithPollInterval(time.Millisecond), WithPromptBootDelay(0))
	result := make(chan error, 1)
	go func() {
		_, err := sub.Launch(context.Background(), LaunchParams{
			RunID: uuid.New(), RunnerType: domain.RunnerTypeCodex, Tag: "run-install-order",
			WorkingDir: "/work", RunDir: runDir, Prompt: "perform the task", NativeObjective: objective,
		})
		result <- err
	}()

	select {
	case <-goalSent:
	case err := <-result:
		t.Fatalf("Launch exited before sending the goal: %v", err)
	case <-time.After(time.Second):
		t.Fatal("Launch did not submit the native goal")
	}
	if fs.sendPrompts != 0 {
		t.Fatalf("task prompt was submitted before the active goal marker: %d submissions", fs.sendPrompts)
	}
	if want := "/goal " + objective; fs.goalCommand != want || strings.Contains(fs.goalCommand, "\n") {
		t.Fatalf("goal command = %q, want one literal line %q", fs.goalCommand, want)
	}
	appendGoalInstallMarker(t, transcript, "native-thread-current", objective)
	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("Launch after matching active marker: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Launch did not continue after the matching active goal marker")
	}
	if fs.sendPrompts != 1 {
		t.Fatalf("prompt submissions = %d, want 1 after marker", fs.sendPrompts)
	}
}

func TestSubstrateLaunch_DoesNotSubmitPromptWithoutMatchingCurrentGoal(t *testing.T) {
	const objective = "finish the verified task"
	for _, tc := range []struct {
		name            string
		markerObjective string
		markerThreadID  string
	}{
		{name: "wrong objective", markerObjective: "another task", markerThreadID: "native-thread-current"},
		{name: "wrong current thread", markerObjective: objective, markerThreadID: "native-thread-other"},
		{name: "no active marker", markerObjective: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runDir := t.TempDir()
			transcript := filepath.Join(runDir, "codex", "sessions", "2026", "10", "17", "rollout-current.jsonl")
			goalInstallTranscript(t, transcript, "native-thread-current", "", runner.GoalStatus(""), "")
			fs := &goalInstallSessions{fakeSessions: newFakeSessions("web-session")}
			if tc.markerObjective != "" {
				fs.onGoal = func() { appendGoalInstallMarker(t, transcript, tc.markerThreadID, tc.markerObjective) }
			}
			sub := NewSubstrate(fs, fakeResolver(codexGoalLaunchInfo{fakeLaunchInfo{
				rt: domain.RunnerTypeCodex, tagKey: "CODEX_AGENT_TAG", binary: "/usr/bin/codex",
			}}), WithDiscoveryTimeout(time.Second), WithGoalInstallTimeout(20*time.Millisecond), WithPollInterval(time.Millisecond), WithPromptBootDelay(0))
			_, err := sub.Launch(context.Background(), LaunchParams{
				RunID: uuid.New(), RunnerType: domain.RunnerTypeCodex, Tag: "run-install-order",
				WorkingDir: "/work", RunDir: runDir, Prompt: "perform the task", NativeObjective: objective,
			})
			if err == nil || !strings.Contains(err.Error(), "native objective was not acknowledged") {
				t.Fatalf("Launch error = %v, want bounded objective-acknowledgement failure", err)
			}
			if fs.sendPrompts != 0 {
				t.Fatalf("prompt submissions = %d, want none without an exact active marker", fs.sendPrompts)
			}
		})
	}
}

func TestSubstrateLaunch_DoesNotSendTaskWhenGoalCommandRemainsComposed(t *testing.T) {
	base := newFakeSessions("web-session")
	base.sendCommandErr = webconsole.ErrCommandNotSubmitted
	fs := &goalInstallSessions{fakeSessions: base}
	sub := NewSubstrate(fs, fakeResolver(codexGoalLaunchInfo{fakeLaunchInfo{
		rt: domain.RunnerTypeCodex, tagKey: "CODEX_AGENT_TAG", binary: "/usr/bin/codex",
	}}), WithDiscoveryTimeout(time.Second), WithGoalInstallTimeout(20*time.Millisecond), WithPollInterval(time.Millisecond), WithPromptBootDelay(0))

	_, err := sub.Launch(context.Background(), LaunchParams{
		RunID: uuid.New(), RunnerType: domain.RunnerTypeCodex, Tag: "goal-submit-failed",
		WorkingDir: "/work", RunDir: t.TempDir(), Prompt: "perform the task", NativeObjective: "finish the verified task",
	})
	if !errors.Is(err, webconsole.ErrCommandNotSubmitted) {
		t.Fatalf("Launch error = %v, want ErrCommandNotSubmitted", err)
	}
	if fs.goalSendCommands != 1 || fs.sendPrompts != 0 {
		t.Fatalf("goal submissions=%d task submissions=%d; want goal attempt only", fs.goalSendCommands, fs.sendPrompts)
	}
}

func TestSubstrateLaunch_CompletesPartialSessionPrefixBeforeFreshGoal(t *testing.T) {
	const objective = "finish the verified task"
	runDir := t.TempDir()
	transcript := filepath.Join(runDir, "codex", "sessions", "2026", "10", "17", "rollout-current.jsonl")
	if err := os.MkdirAll(filepath.Dir(transcript), 0o755); err != nil {
		t.Fatal(err)
	}
	partialMeta := `{"type":"session_meta","payload":{"id":"native-thread-current"}`
	if err := os.WriteFile(transcript, []byte(partialMeta), 0o600); err != nil {
		t.Fatal(err)
	}
	fs := &goalInstallSessions{fakeSessions: newFakeSessions("web-session"), onGoal: func() {
		appendGoalInstallBytes(t, transcript, []byte("}\n"))
		appendGoalInstallMarker(t, transcript, "native-thread-current", objective)
	}}
	sub := NewSubstrate(fs, fakeResolver(codexGoalLaunchInfo{fakeLaunchInfo{
		rt: domain.RunnerTypeCodex, tagKey: "CODEX_AGENT_TAG", binary: "/usr/bin/codex",
	}}), WithDiscoveryTimeout(time.Second), WithGoalInstallTimeout(time.Second), WithPollInterval(time.Millisecond), WithPromptBootDelay(0))
	_, err := sub.Launch(context.Background(), LaunchParams{
		RunID: uuid.New(), RunnerType: domain.RunnerTypeCodex, Tag: "partial-session-prefix",
		WorkingDir: "/work", RunDir: runDir, Prompt: "perform the task", NativeObjective: objective,
	})
	if err != nil {
		t.Fatalf("Launch after partial session prefix and fresh marker: %v", err)
	}
	if fs.sendPrompts != 1 {
		t.Fatalf("prompt submissions = %d, want 1 after fresh marker", fs.sendPrompts)
	}
}

func TestSubstrateLaunch_DoesNotAckPartialOldActiveMarker(t *testing.T) {
	const objective = "finish the verified task"
	runDir := t.TempDir()
	transcript := filepath.Join(runDir, "codex", "sessions", "2026", "10", "17", "rollout-current.jsonl")
	goalInstallTranscript(t, transcript, "native-thread-current", "", runner.GoalStatus(""), "")
	oldMarker := goalInstallMarkerLine(t, "native-thread-current", objective)
	if err := os.WriteFile(transcript, append(mustReadFile(t, transcript), oldMarker[:len(oldMarker)-3]...), 0o600); err != nil {
		t.Fatal(err)
	}
	partialFinished := make(chan struct{})
	fs := &goalInstallSessions{fakeSessions: newFakeSessions("web-session"), onGoal: func() {
		appendGoalInstallBytes(t, transcript, oldMarker[len(oldMarker)-3:])
		close(partialFinished)
	}}
	sub := NewSubstrate(fs, fakeResolver(codexGoalLaunchInfo{fakeLaunchInfo{
		rt: domain.RunnerTypeCodex, tagKey: "CODEX_AGENT_TAG", binary: "/usr/bin/codex",
	}}), WithDiscoveryTimeout(time.Second), WithGoalInstallTimeout(500*time.Millisecond), WithPollInterval(time.Millisecond), WithPromptBootDelay(0))
	result := make(chan error, 1)
	go func() {
		_, err := sub.Launch(context.Background(), LaunchParams{
			RunID: uuid.New(), RunnerType: domain.RunnerTypeCodex, Tag: "partial-old-active",
			WorkingDir: "/work", RunDir: runDir, Prompt: "perform the task", NativeObjective: objective,
		})
		result <- err
	}()
	select {
	case <-partialFinished:
	case err := <-result:
		t.Fatalf("Launch exited before completing the old partial marker: %v", err)
	case <-time.After(time.Second):
		t.Fatal("Launch did not submit /goal")
	}
	select {
	case err := <-result:
		t.Fatalf("old partial active marker was treated as fresh: Launch returned %v", err)
	case <-time.After(25 * time.Millisecond):
	}
	if fs.sendPrompts != 0 {
		t.Fatalf("prompt submissions = %d, want none for old partial marker", fs.sendPrompts)
	}
	appendGoalInstallMarker(t, transcript, "native-thread-current", objective)
	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("Launch after fresh marker: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Launch did not continue after fresh active marker")
	}
	if fs.sendPrompts != 1 {
		t.Fatalf("prompt submissions = %d, want 1 after fresh marker", fs.sendPrompts)
	}
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return body
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
