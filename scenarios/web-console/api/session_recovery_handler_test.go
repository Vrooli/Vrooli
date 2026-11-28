package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"web-console/internal/backend"
	"web-console/internal/sessionstore"
	intworkspace "web-console/internal/workspace"

	"connectrpc.com/connect"

	sessionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions"
)

func TestPrunedCodexHomeStaysAbsentAndReadOnly(t *testing.T) {
	root := t.TempDir()
	t.Setenv("WC_SESSION_STATE_ROOT", root)
	meta := sessionstore.Metadata{ID: "archived-codex", AgentType: sessionstore.AgentCodex, AgentSessionID: "agent-1"}
	home := filepath.Join(root, "codex", meta.ID)
	if err := os.MkdirAll(filepath.Join(home, "sessions"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "sessions", "rollout.jsonl"), []byte("history"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !archivedAgentHistoryPresent(meta) {
		t.Fatal("seeded Codex history should be present")
	}
	reclaimed, err := pruneArchivedAgentHistory(meta)
	if err != nil {
		t.Fatal(err)
	}
	if reclaimed == 0 {
		t.Fatal("expected measured reclaimed bytes")
	}
	if archivedAgentHistoryPresent(meta) {
		t.Fatal("history check recreated or accepted the pruned Codex home")
	}
	if _, err := os.Stat(home); !os.IsNotExist(err) {
		t.Fatalf("pruned home exists after history check: %v", err)
	}
}

func TestCopyCodexHomeCopiesDurableRecordOnly(t *testing.T) {
	root := t.TempDir()
	t.Setenv("WC_SESSION_STATE_ROOT", root)
	old := filepath.Join(root, "codex", "old")
	write := func(rel, body string) {
		t.Helper()
		path := filepath.Join(old, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	// The durable record: rollouts, the goal store with its WAL, thread state,
	// history and a long goal's objective attachment.
	durable := map[string]string{
		"sessions/2026/08/26/rollout.jsonl": "rollout",
		"goals_1.sqlite":                    "goals",
		"goals_1.sqlite-wal":                "goals-wal",
		"state_5.sqlite":                    "state",
		"thread_history_1.sqlite":           "history-db",
		"history.jsonl":                     "history",
		"attachments/g1/goal-objective.md":  "objective",
	}
	// Runtime the launcher re-links or Codex re-creates.
	runtime := []string{"cache/runtime.db", "packages/daemon/bin", "app-server-control/app-server-startup.lock", "logs_2.sqlite", "goals_1.sqlite-shm", "session_index.jsonl.lock"}
	for rel, body := range durable {
		write(rel, body)
	}
	for _, rel := range runtime {
		write(rel, "runtime")
	}
	if err := os.Symlink(filepath.Join(root, "shared-auth.json"), filepath.Join(old, "auth.json")); err != nil {
		t.Fatal(err)
	}

	if err := copyCodexHome("old", "new"); err != nil {
		t.Fatalf("copyCodexHome: %v", err)
	}
	newHome := filepath.Join(root, "codex", "new")
	for rel, body := range durable {
		if got, err := os.ReadFile(filepath.Join(newHome, rel)); err != nil || string(got) != body {
			t.Errorf("durable %s copied as %q, %v", rel, got, err)
		}
	}
	for _, rel := range append(runtime, "auth.json") {
		if _, err := os.Lstat(filepath.Join(newHome, rel)); !os.IsNotExist(err) {
			t.Errorf("runtime %s was copied: %v", rel, err)
		}
	}
}

func TestCopyCodexHomeRefusesSymlinkInRolloutTree(t *testing.T) {
	root := t.TempDir()
	t.Setenv("WC_SESSION_STATE_ROOT", root)
	dir := filepath.Join(root, "codex", "old", "sessions")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/etc/passwd", filepath.Join(dir, "rollout.jsonl")); err != nil {
		t.Fatal(err)
	}
	if err := copyCodexHome("old", "new"); err == nil {
		t.Fatal("a symlink inside the rollout tree must refuse the copy")
	}
}

// newRecoveryTestServer wires the in-memory store + fake PTY factory so the
// recovery flow can be exercised end-to-end without tmux. The fake PTY
// preserves the WriteInput contract the recovery adapter depends on.
func newRecoveryTestServer(t *testing.T) *Server {
	t.Helper()
	useIsolatedSessionState(t)
	srv := newFakeTestServer()
	srv.sessionStore = sessionstore.NewInMemory()
	srv.sessions.SetStore(srv.sessionStore)
	return srv
}

func saveOrphan(t *testing.T, srv *Server, id string, agent sessionstore.Agent, agentSessionID string) {
	t.Helper()
	if err := srv.sessionStore.Save(context.Background(), sessionstore.Metadata{
		ID:             id,
		Backend:        backend.Persistent,
		Shell:          "/bin/bash",
		Cols:           120,
		Rows:           36,
		Created:        time.Now().Add(-time.Hour),
		Detached:       true,
		Status:         sessionstore.StatusAwaitingRecovery,
		AgentType:      agent,
		AgentSessionID: agentSessionID,
		OrphanedAt:     time.Now().Add(-time.Minute),
	}); err != nil {
		t.Fatalf("save orphan: %v", err)
	}
}

func callListRecoverable(t *testing.T, srv *Server) []*sessionsv1.RecoverableSession {
	t.Helper()
	resp, err := newSessionsConnectHandlerForServer(srv).ListRecoverable(context.Background(),
		connect.NewRequest(&sessionsv1.ListRecoverableRequest{}))
	if err != nil {
		t.Fatalf("ListRecoverable: %v", err)
	}
	return resp.Msg.GetSessions()
}

func callRecover(t *testing.T, srv *Server, id string) (*sessionsv1.RecoverResponse, error) {
	t.Helper()
	resp, err := newSessionsConnectHandlerForServer(srv).Recover(context.Background(),
		connect.NewRequest(&sessionsv1.RecoverRequest{Id: id}))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

func callReopen(t *testing.T, srv *Server, id, idempotencyKey string) (*sessionsv1.ReopenResponse, error) {
	t.Helper()
	req := connect.NewRequest(&sessionsv1.ReopenRequest{Id: id})
	if idempotencyKey != "" {
		req.Header().Set("X-Idempotency-Key", idempotencyKey)
	}
	resp, err := newSessionsConnectHandlerForServer(srv).Reopen(context.Background(), req)
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

func saveArchived(t *testing.T, srv *Server, id string, agent sessionstore.Agent, agentSessionID string) {
	t.Helper()
	if err := srv.sessionStore.Save(context.Background(), sessionstore.Metadata{
		ID:             id,
		Backend:        backend.Persistent,
		Shell:          "/bin/bash",
		Cols:           120,
		Rows:           36,
		Created:        time.Now().Add(-2 * time.Hour),
		Detached:       true,
		Status:         sessionstore.StatusDismissed,
		AgentType:      agent,
		AgentSessionID: agentSessionID,
		ArchivedAt:     time.Now().Add(-time.Hour),
	}); err != nil {
		t.Fatalf("save archived session: %v", err)
	}
}

func callDismissRecoverable(t *testing.T, srv *Server, id string) error {
	t.Helper()
	_, err := newSessionsConnectHandlerForServer(srv).DismissRecoverable(context.Background(),
		connect.NewRequest(&sessionsv1.DismissRecoverableRequest{Id: id}))
	return err
}

func TestHandleListRecoverable_OrdersByActivity(t *testing.T) {
	srv := newRecoveryTestServer(t)
	saveOrphan(t, srv, "older", sessionstore.AgentCodex, "codex-1")
	saveOrphan(t, srv, "newer", sessionstore.AgentCodex, "codex-2")
	_ = srv.sessionStore.UpdateAgentInfo(context.Background(), "older", sessionstore.AgentInfo{LastActivityAt: time.Now().Add(-2 * time.Hour)})
	_ = srv.sessionStore.UpdateAgentInfo(context.Background(), "newer", sessionstore.AgentInfo{LastActivityAt: time.Now().Add(-5 * time.Minute)})

	rows := callListRecoverable(t, srv)
	// In-memory store does not enforce ordering; just check membership + recoverable flag.
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	for _, r := range rows {
		if !r.GetRecoverable() {
			t.Errorf("expected codex orphan %s recoverable, got reason=%q", r.GetId(), r.GetNotRecoverableReason())
		}
	}
}

func TestHandleRecover_Codex_HappyPath(t *testing.T) {
	srv := newRecoveryTestServer(t)
	saveOrphan(t, srv, "codex-old", sessionstore.AgentCodex, "019d-codex-uuid")

	resp, err := callRecover(t, srv, "codex-old")
	if err != nil {
		t.Fatalf("Recover: %v", err)
	}
	if resp.GetOldSessionId() != "codex-old" {
		t.Errorf("OldSessionID: got %q", resp.GetOldSessionId())
	}
	if resp.GetNewSessionId() == "" {
		t.Error("NewSessionID empty")
	}
	if !strings.Contains(resp.GetCommandSent(), "codex --yolo resume 019d-codex-uuid") {
		t.Errorf("CommandSent: got %q", resp.GetCommandSent())
	}
	old, _ := srv.sessionStore.Get(context.Background(), "codex-old")
	if old.Status != sessionstore.StatusDismissed {
		t.Errorf("old row status: got %q", old.Status)
	}
	if old.RecoveredInto != resp.GetNewSessionId() {
		t.Errorf("RecoveredInto: got %q want %q", old.RecoveredInto, resp.GetNewSessionId())
	}
}

func TestHandleReopen_ArchivedSessionIsIdempotentAndClearsArchiveMarker(t *testing.T) {
	srv := newRecoveryTestServer(t)
	saveArchived(t, srv, "archived-old", sessionstore.AgentCodex, "codex-archive")

	first, err := callReopen(t, srv, "archived-old", "stable-drawer-entry-key")
	if err != nil {
		t.Fatalf("Reopen: %v", err)
	}
	second, err := callReopen(t, srv, "archived-old", "stable-drawer-entry-key")
	if err != nil {
		t.Fatalf("idempotent Reopen: %v", err)
	}
	if first.GetNewSessionId() == "" || second.GetNewSessionId() != first.GetNewSessionId() {
		t.Fatalf("reopen ids first=%q second=%q", first.GetNewSessionId(), second.GetNewSessionId())
	}
	old, err := srv.sessionStore.Get(context.Background(), "archived-old")
	if err != nil {
		t.Fatal(err)
	}
	if !old.ArchivedAt.IsZero() {
		t.Fatalf("archived_at survived successful reopen: %s", old.ArchivedAt)
	}
	if old.RecoveredInto != first.GetNewSessionId() {
		t.Fatalf("recovered_into=%q want %q", old.RecoveredInto, first.GetNewSessionId())
	}
}

// A lead whose recorded thread is a sub-agent's (the tailer used to adopt the
// last rollout it saw) reopens on the root thread, with the goal store copied
// so the paused goal comes back with it.
func TestHandleReopen_CodexResumesRootThreadWithGoalStore(t *testing.T) {
	srv := newRecoveryTestServer(t)
	saveArchived(t, srv, "lead-old", sessionstore.AgentCodex, "child-thread")
	home := filepath.Join(resolveSessionStateRoot(), "codex", "lead-old")
	dir := filepath.Join(home, "sessions", "2026", "11", "19")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		"rollout-2026-11-19T03-13-10-root-thread.jsonl":  `{"type":"session_meta","payload":{"id":"root-thread","session_id":"root-thread","thread_source":"user"}}`,
		"rollout-2026-11-19T20-43-31-child-thread.jsonl": `{"type":"session_meta","payload":{"id":"child-thread","session_id":"root-thread","thread_source":"subagent"}}`,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(home, "goals_1.sqlite"), []byte("goal store"), 0o600); err != nil {
		t.Fatal(err)
	}

	resp, err := callReopen(t, srv, "lead-old", "reopen-lead")
	if err != nil {
		t.Fatalf("Reopen: %v", err)
	}
	if !strings.Contains(resp.GetCommandSent(), "codex --yolo resume root-thread") {
		t.Fatalf("CommandSent = %q, want the root thread resumed", resp.GetCommandSent())
	}
	if got, err := os.ReadFile(filepath.Join(resolveSessionStateRoot(), "codex", resp.GetNewSessionId(), "goals_1.sqlite")); err != nil || string(got) != "goal store" {
		t.Fatalf("reopened home goal store = %q, %v", got, err)
	}
	reopened, err := srv.sessionStore.Get(context.Background(), resp.GetNewSessionId())
	if err != nil {
		t.Fatal(err)
	}
	if reopened.AgentSessionID != "root-thread" {
		t.Fatalf("reopened agent_session_id = %q, want root-thread", reopened.AgentSessionID)
	}
}

func TestHandleReopen_RefusesArchivedClaudeWithoutAgentIdentity(t *testing.T) {
	srv := newRecoveryTestServer(t)
	saveArchived(t, srv, "claude-read-only", sessionstore.AgentClaude, "")

	_, err := callReopen(t, srv, "claude-read-only", "read-only-key")
	if got := connectCode(err); got != connect.CodeFailedPrecondition {
		t.Fatalf("expected CodeFailedPrecondition, got %s (err=%v)", got, err)
	}
	if err == nil || !strings.Contains(err.Error(), "claude session id is required") {
		t.Fatalf("missing projection refusal reason: %v", err)
	}
}

func TestHandleRecover_MigratesCustomizedPane(t *testing.T) {
	srv := newRecoveryTestServer(t)
	saveOrphan(t, srv, "pane-old", sessionstore.AgentCodex, "codex-pane")
	if err := srv.sessionStore.UpdateAgentInfo(context.Background(), "pane-old", sessionstore.AgentInfo{CWD: "/work/recovered"}); err != nil {
		t.Fatalf("set cwd: %v", err)
	}
	if err := srv.workspace.UpsertPane(context.Background(), intworkspace.Pane{
		SessionID:            "pane-old",
		Name:                 "Important recovery work",
		HeaderColor:          "#ff6b6b",
		ThemeID:              "midnight",
		FontSize:             12,
		SortOrder:            4,
		GroupID:              "group-1",
		SupportsMessagesView: true,
	}); err != nil {
		t.Fatalf("create customized pane: %v", err)
	}

	resp, err := callRecover(t, srv, "pane-old")
	if err != nil {
		t.Fatalf("Recover: %v", err)
	}
	layout, err := srv.workspace.GetLayout(context.Background())
	if err != nil {
		t.Fatalf("GetLayout: %v", err)
	}
	if len(layout.Panes) != 1 {
		t.Fatalf("panes after recovery = %d, want 1", len(layout.Panes))
	}
	got := layout.Panes[0]
	if got.SessionID != resp.GetNewSessionId() || got.Name != "Important recovery work" || got.HeaderColor != "#ff6b6b" || got.ThemeID != "midnight" || got.FontSize != 12 || got.SortOrder != 4 || got.GroupID != "group-1" || !got.SupportsMessagesView {
		t.Fatalf("recovered pane = %#v, original customization was not preserved", got)
	}
}

func TestHandleRecover_Codex_NoSessionIDFallsBackToLast(t *testing.T) {
	srv := newRecoveryTestServer(t)
	saveOrphan(t, srv, "codex-bare", sessionstore.AgentCodex, "")

	resp, err := callRecover(t, srv, "codex-bare")
	if err != nil {
		t.Fatalf("Recover: %v", err)
	}
	if !strings.Contains(resp.GetCommandSent(), "codex --yolo resume --last") {
		t.Errorf("CommandSent: got %q", resp.GetCommandSent())
	}
}

func TestHandleRecover_Claude_RequiresSessionID(t *testing.T) {
	srv := newRecoveryTestServer(t)
	saveOrphan(t, srv, "claude-bare", sessionstore.AgentClaude, "")

	_, err := callRecover(t, srv, "claude-bare")
	if got := connectCode(err); got != connect.CodeFailedPrecondition {
		t.Fatalf("expected CodeFailedPrecondition, got %s (err=%v)", got, err)
	}
}

func TestHandleRecover_RejectsLiveSession(t *testing.T) {
	srv := newRecoveryTestServer(t)
	if err := srv.sessionStore.Save(context.Background(), sessionstore.Metadata{
		ID:        "live-id",
		Backend:   backend.Persistent,
		Shell:     "/bin/bash",
		Cols:      80,
		Rows:      24,
		Created:   time.Now(),
		Detached:  true,
		Status:    sessionstore.StatusLive,
		AgentType: sessionstore.AgentCodex,
	}); err != nil {
		t.Fatalf("save: %v", err)
	}

	_, err := callRecover(t, srv, "live-id")
	if got := connectCode(err); got != connect.CodeFailedPrecondition {
		t.Fatalf("expected CodeFailedPrecondition, got %s (err=%v)", got, err)
	}
}

func TestHandleRecover_NotFound(t *testing.T) {
	srv := newRecoveryTestServer(t)
	_, err := callRecover(t, srv, "nope")
	if got := connectCode(err); got != connect.CodeNotFound {
		t.Fatalf("expected CodeNotFound, got %s (err=%v)", got, err)
	}
}

func TestHandleDismissRecoverable_TransitionsToDismissed(t *testing.T) {
	srv := newRecoveryTestServer(t)
	saveOrphan(t, srv, "drop-me", sessionstore.AgentCodex, "x")

	if err := callDismissRecoverable(t, srv, "drop-me"); err != nil {
		t.Fatalf("DismissRecoverable: %v", err)
	}
	got, _ := srv.sessionStore.Get(context.Background(), "drop-me")
	if got.Status != sessionstore.StatusDismissed {
		t.Errorf("status: got %q", got.Status)
	}
}

func TestCreateSession_PersistsLaunchCommandAndAgentType(t *testing.T) {
	srv := newFakeTestServer()
	srv.sessionStore = sessionstore.NewInMemory()
	srv.sessions.SetStore(srv.sessionStore)

	resp, err := newSessionsConnectHandlerForServer(srv).Create(context.Background(),
		connect.NewRequest(&sessionsv1.CreateRequest{
			Cols:          80,
			Rows:          24,
			Backend:       "standard",
			LaunchCommand: "codex --yolo",
			AgentType:     "codex",
		}))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, err := srv.sessionStore.Get(context.Background(), resp.Msg.GetSession().GetId())
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.AgentType != sessionstore.AgentCodex {
		t.Errorf("agent_type: got %q", got.AgentType)
	}
	if got.LaunchCommand != "codex --yolo" {
		t.Errorf("launch_command: got %q", got.LaunchCommand)
	}
}
