package main

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"web-console/backends/codex"
	"web-console/internal/backend"
	"web-console/internal/policy"
	"web-console/internal/ptyfake"
	"web-console/internal/sessionstore"
	"web-console/session"
)

// idleFixture is one live Codex session under the idle policy, with a real
// per-session Codex home (rollout tree and goal store) on disk.
type idleFixture struct {
	t     *testing.T
	root  string
	sess  *session.Session
	meta  sessionstore.Metadata
	now   time.Time
	judge *sessionIdleJudge
	// native is what a managed owner reports; nativeErr makes it unreadable.
	native    []codex.NativeThreadState
	nativeErr error
}

const idleLeadThread = "01a1f2dd-e24e-7ec2-8d29-13d774b19b1e"

func newIdleFixture(t *testing.T) *idleFixture {
	t.Helper()
	sm := session.NewManagerWithFactory(ptyfake.NewFactory())
	sm.SetActivityFactory(func(s *session.Session) *session.ActivityDetector {
		return session.NewActivityDetector(session.ActivityConfig{SessionID: s.ID})
	})
	idle := policy.Policy{Mode: policy.Idle, Duration: "6h"}
	sess, err := sm.Create(context.Background(), "", 80, 24, "", &idle)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	t.Cleanup(func() { _ = sm.Delete(context.Background(), sess.ID) })
	f := &idleFixture{t: t, root: t.TempDir(), sess: sess, now: time.Now()}
	sess.CreatedAt = f.now.Add(-48 * time.Hour)
	f.meta = sessionstore.Metadata{ID: sess.ID, AgentType: sessionstore.AgentCodex, LaunchMode: sessionstore.LaunchModeTerminalPTY, Policy: idle}
	f.judge = &sessionIdleJudge{
		now:       func() time.Time { return f.now },
		metadata:  func(context.Context, string) (sessionstore.Metadata, error) { return f.meta, nil },
		stateRoot: func() string { return f.root },
		nativeState: func(context.Context, sessionstore.Metadata) ([]codex.NativeThreadState, error) {
			return f.native, f.nativeErr
		},
		memory: func(string) (int64, bool) { return 5 << 29, true }, // 2.5 GB
	}
	// The lead's rollout and one sub-agent's, both last written a day ago.
	f.rollout(idleLeadThread, f.now.Add(-21*time.Hour))
	f.rollout("01a1f69f-825d-7c13-b973-058e7933ad0d", f.now.Add(-20*time.Hour))
	f.goal(idleLeadThread, "paused")
	sess.ActivityDetector().OnHook(session.HookIdle)
	return f
}

func (f *idleFixture) home() string { return codex.SessionHomePath(f.root, f.sess.ID) }

func (f *idleFixture) rollout(thread string, written time.Time) {
	f.t.Helper()
	dir := filepath.Join(f.home(), "sessions", "2026", "11", "19")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		f.t.Fatal(err)
	}
	path := filepath.Join(dir, "rollout-2026-11-19T03-13-10-"+thread+".jsonl")
	if err := os.WriteFile(path, []byte(`{"type":"session_meta","payload":{"id":"`+thread+`","session_id":"`+idleLeadThread+`"}}`+"\n"), 0o600); err != nil {
		f.t.Fatal(err)
	}
	if err := os.Chtimes(path, written, written); err != nil {
		f.t.Fatal(err)
	}
}

func (f *idleFixture) goal(thread, status string) {
	f.t.Helper()
	db, err := sql.Open("sqlite", "file:"+filepath.Join(f.home(), "goals_1.sqlite"))
	if err != nil {
		f.t.Fatal(err)
	}
	defer db.Close()
	for _, stmt := range []string{
		`CREATE TABLE IF NOT EXISTS thread_goals (thread_id TEXT PRIMARY KEY NOT NULL, goal_id TEXT NOT NULL, objective TEXT NOT NULL, status TEXT NOT NULL, token_budget INTEGER, tokens_used INTEGER NOT NULL DEFAULT 0, time_used_seconds INTEGER NOT NULL DEFAULT 0, created_at_ms INTEGER NOT NULL, updated_at_ms INTEGER NOT NULL)`,
		`INSERT OR REPLACE INTO thread_goals (thread_id, goal_id, objective, status, created_at_ms, updated_at_ms) VALUES ('` + thread + `', 'g', 'objective', '` + status + `', 0, 0)`,
	} {
		if _, err := db.Exec(stmt); err != nil {
			f.t.Fatal(err)
		}
	}
}

func (f *idleFixture) verdict() session.IdleVerdict {
	return f.judge.judge(context.Background(), f.sess, 6*time.Hour)
}

func (f *idleFixture) wantKept(contains string) {
	f.t.Helper()
	v := f.verdict()
	if v.Archive {
		f.t.Fatalf("session archived (%q); want kept because %s", v.Note, contains)
	}
	if !strings.Contains(v.Note, contains) {
		f.t.Fatalf("keep reason %q does not name %q", v.Note, contains)
	}
}

func TestIdleJudgeArchivesIdleSessionWithInactiveGoal(t *testing.T) {
	f := newIdleFixture(t)
	v := f.verdict()
	if !v.Archive {
		t.Fatalf("idle session with a paused goal was kept: %s", v.Note)
	}
	for _, want := range []string{
		"Archived by the idle policy (6h0m0s)",
		"(20h0m0s idle)",
		"goal paused on thread " + idleLeadThread,
		"no turn or sub-agent running",
		"Memory freed: 2.5 GB",
		"web-console session reopen " + f.sess.ID,
	} {
		if !strings.Contains(v.Note, want) {
			t.Errorf("archive note %q lacks %q", v.Note, want)
		}
	}
}

func TestIdleJudgeKeepsActiveGoal(t *testing.T) {
	f := newIdleFixture(t)
	f.goal(idleLeadThread, "active")
	f.wantKept("goal active")
}

func TestIdleJudgeKeepsRunningTurn(t *testing.T) {
	f := newIdleFixture(t)
	// A turn shows Codex's working marker; any output makes the agent working.
	f.sess.ActivityDetector().OnFrame()
	f.wantKept("agent is working")
}

func TestIdleJudgeKeepsRunningSubAgent(t *testing.T) {
	f := newIdleFixture(t)
	// The lead sits at its prompt while a sub-agent thread keeps writing.
	f.rollout("01a1f6a0-0000-7000-8000-000000000001", f.now.Add(-10*time.Minute))
	f.wantKept("agent worked 10m0s ago")
}

func TestIdleJudgeKeepsManagedOwnerTurnAndUnreadableOwner(t *testing.T) {
	f := newIdleFixture(t)
	f.meta.LaunchMode = sessionstore.LaunchModeCodexAppServer
	f.meta.NativeThreadID = idleLeadThread
	f.native = []codex.NativeThreadState{{ThreadID: idleLeadThread, Status: "active", ActiveTurnID: "turn-1"}}
	f.wantKept("running a turn")
	f.native, f.nativeErr = nil, errors.New("owner gone")
	f.wantKept("native owner unreadable")
	f.nativeErr = nil
	f.native = []codex.NativeThreadState{{ThreadID: idleLeadThread, Status: "idle", GoalAvailable: true, Goal: &codex.NativeGoalState{ThreadID: idleLeadThread, Status: "paused"}}}
	if v := f.verdict(); !v.Archive {
		t.Fatalf("idle managed session with a paused goal was kept: %s", v.Note)
	}
}

func TestIdleJudgeRecordsWaitingPrompt(t *testing.T) {
	f := newIdleFixture(t)
	f.sess.ActivityDetector().OnHarnessPrompt(backend.PendingPrompt{
		Kind: "question", Text: "Merge the MB7 branch now?",
		Options: []backend.PromptOption{{Key: "1", Label: "Yes"}, {Key: "2", Label: "No"}},
	})
	v := f.verdict()
	if !v.Archive {
		t.Fatalf("a session waiting on an in-chat prompt counts as idle; kept: %s", v.Note)
	}
	if !strings.Contains(v.Note, `It was waiting on a question: "Merge the MB7 branch now?" (options: 1 Yes; 2 No).`) {
		t.Fatalf("archive note %q does not record the prompt", v.Note)
	}
}

func TestIdleJudgeKeepsWhenEvidenceIsMissing(t *testing.T) {
	t.Run("recently reopened", func(t *testing.T) {
		f := newIdleFixture(t)
		f.sess.CreatedAt = f.now.Add(-time.Hour)
		f.wantKept("agent worked 1h0m0s ago")
	})
	t.Run("goal store unreadable", func(t *testing.T) {
		f := newIdleFixture(t)
		if err := os.WriteFile(filepath.Join(f.home(), "goals_1.sqlite"), []byte("not a database"), 0o600); err != nil {
			t.Fatal(err)
		}
		f.wantKept("goal store unreadable")
	})
	t.Run("no rollout tree", func(t *testing.T) {
		f := newIdleFixture(t)
		if err := os.RemoveAll(filepath.Join(f.home(), "sessions")); err != nil {
			t.Fatal(err)
		}
		f.wantKept("no codex rollout tree")
	})
	t.Run("unknown harness", func(t *testing.T) {
		f := newIdleFixture(t)
		f.meta.AgentType = sessionstore.AgentNone
		f.wantKept("no work record")
	})
	t.Run("harness reports working", func(t *testing.T) {
		f := newIdleFixture(t)
		f.sess.ActivityDetector().OnHarnessState(session.ActivityWorking)
		f.wantKept("agent is working")
	})
}

// Codex's own selection prompts (such as "Resume paused goal?") reach the
// detector as an unknown prompt without text; the note keeps the screen.
func TestDescribeUnparsedPromptKeepsTheScreen(t *testing.T) {
	waiting := session.Activity{State: session.ActivityWaiting, Prompt: &backend.PendingPrompt{Kind: "unknown"}}
	screen := "worked for 9m\n\n    Resume paused goal?\n  › 1. Resume goal   Mark it active\n    2. Leave paused  Keep it paused\n"
	got := describePrompt(waiting, screen)
	if !strings.Contains(got, "waiting on a prompt; the screen read:") || !strings.Contains(got, "Resume paused goal? / › 1. Resume goal Mark it active / 2. Leave paused Keep it paused") {
		t.Fatalf("note = %q", got)
	}
	if got := describePrompt(waiting, ""); got != "It was waiting on a prompt whose text could not be read." {
		t.Fatalf("blank screen note = %q", got)
	}
}
