package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"web-console/internal/sessionstore"
)

// identityStoreStub is a minimal sessionstore.Store recording only what the
// resolver uses. Embedding the interface keeps the stub honest: if the resolver
// starts calling something else, the nil embed panics rather than silently
// passing.
type identityStoreStub struct {
	sessionstore.Store
	rows    []sessionstore.Metadata
	updates map[string]sessionstore.AgentInfo
}

func (s *identityStoreStub) List(context.Context) ([]sessionstore.Metadata, error) {
	return s.rows, nil
}

func (s *identityStoreStub) UpdateAgentInfo(_ context.Context, id string, info sessionstore.AgentInfo) error {
	if s.updates == nil {
		s.updates = map[string]sessionstore.AgentInfo{}
	}
	s.updates[id] = info
	return nil
}

// writeTranscript creates a Claude transcript whose first timestamped entry is
// firstEntry, then stamps its modification time.
func writeTranscript(t *testing.T, projectDir, agentSessionID string, firstEntry, modTime time.Time) string {
	t.Helper()
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir project dir: %v", err)
	}
	path := filepath.Join(projectDir, agentSessionID+".jsonl")
	// The untimestamped preamble is what Claude actually writes first; including
	// it keeps the fixture honest about the scan the resolver has to perform.
	lines := []any{
		map[string]any{"type": "mode", "mode": "normal", "sessionId": agentSessionID},
		map[string]any{"type": "user", "sessionId": agentSessionID, "timestamp": firstEntry.UTC().Format(time.RFC3339)},
	}
	var payload []byte
	for _, line := range lines {
		encoded, err := json.Marshal(line)
		if err != nil {
			t.Fatalf("marshal fixture: %v", err)
		}
		payload = append(payload, encoded...)
		payload = append(payload, '\n')
	}
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatalf("write transcript: %v", err)
	}
	if err := os.Chtimes(path, modTime, modTime); err != nil {
		t.Fatalf("set mtime: %v", err)
	}
	return path
}

func newResolverForTest(t *testing.T, home string, store sessionstore.Store, procs []agentProcess) *claudeIdentityResolver {
	t.Helper()
	return &claudeIdentityResolver{
		store:    store,
		home:     home,
		discover: func() ([]agentProcess, error) { return procs, nil },
		logger:   nil,
	}
}

func TestResolverAdoptsTheOnlyPlausibleTranscript(t *testing.T) {
	home := t.TempDir()
	cwd := "/work/project"
	started := time.Now().Add(-10 * time.Minute)
	writeTranscript(t, claudeProjectDir(home, cwd), "agent-1", started.Add(30*time.Second), time.Now())

	store := &identityStoreStub{rows: []sessionstore.Metadata{
		{ID: "pane-1", Status: sessionstore.StatusLive, AgentType: sessionstore.AgentClaude, CWD: cwd},
	}}
	resolver := newResolverForTest(t, home, store, []agentProcess{
		{PID: 42, SessionID: "pane-1", WorkingDir: cwd, StartedAt: started},
	})

	if adopted := resolver.Resolve(context.Background()); adopted != 1 {
		t.Fatalf("adopted = %d, want 1", adopted)
	}
	got, ok := store.updates["pane-1"]
	if !ok {
		t.Fatal("expected the pane's identity to be persisted")
	}
	if got.AgentSessionID != "agent-1" {
		t.Errorf("AgentSessionID = %q, want %q", got.AgentSessionID, "agent-1")
	}
	// The working directory is persisted alongside the id because both are
	// required to locate the transcript again on the next discovery pass.
	if got.CWD != cwd {
		t.Errorf("CWD = %q, want %q", got.CWD, cwd)
	}
}

func TestResolverRecoversWorkingDirectoryFromTheProcess(t *testing.T) {
	home := t.TempDir()
	cwd := "/work/project"
	started := time.Now().Add(-5 * time.Minute)
	writeTranscript(t, claudeProjectDir(home, cwd), "agent-1", started.Add(time.Second), time.Now())

	// A pane created before sessions recorded their working directory. The
	// running process still knows where it is, which is what makes these
	// sessions recoverable at all.
	store := &identityStoreStub{rows: []sessionstore.Metadata{
		{ID: "pane-1", Status: sessionstore.StatusLive, AgentType: sessionstore.AgentClaude, CWD: ""},
	}}
	resolver := newResolverForTest(t, home, store, []agentProcess{
		{PID: 42, SessionID: "pane-1", WorkingDir: cwd, StartedAt: started},
	})

	if adopted := resolver.Resolve(context.Background()); adopted != 1 {
		t.Fatalf("adopted = %d, want 1", adopted)
	}
	if store.updates["pane-1"].CWD != cwd {
		t.Errorf("CWD = %q, want %q", store.updates["pane-1"].CWD, cwd)
	}
}

func TestResolverRefusesToGuessBetweenConcurrentSessions(t *testing.T) {
	home := t.TempDir()
	cwd := "/work/project"
	dir := claudeProjectDir(home, cwd)
	base := time.Now().Add(-10 * time.Minute)

	// Two panes started within the slack window of each other, each with a
	// transcript. Nothing distinguishes them, so adopting either would be a
	// coin flip that silently shows one session another's conversation.
	writeTranscript(t, dir, "agent-a", base.Add(time.Second), time.Now())
	writeTranscript(t, dir, "agent-b", base.Add(2*time.Second), time.Now())

	store := &identityStoreStub{rows: []sessionstore.Metadata{
		{ID: "pane-1", Status: sessionstore.StatusLive, AgentType: sessionstore.AgentClaude, CWD: cwd},
	}}
	resolver := newResolverForTest(t, home, store, []agentProcess{
		{PID: 1, SessionID: "pane-1", WorkingDir: cwd, StartedAt: base},
	})

	if adopted := resolver.Resolve(context.Background()); adopted != 0 {
		t.Fatalf("adopted = %d, want 0 — ambiguity must not be resolved by guessing", adopted)
	}
	if len(store.updates) != 0 {
		t.Errorf("nothing should have been persisted, got %v", store.updates)
	}
}

func TestResolverNeverStealsAnAlreadyBoundTranscript(t *testing.T) {
	home := t.TempDir()
	cwd := "/work/project"
	started := time.Now().Add(-5 * time.Minute)
	writeTranscript(t, claudeProjectDir(home, cwd), "agent-1", started.Add(time.Second), time.Now())

	// agent-1 belongs to a session the operator has since closed. Dismissed
	// rows still hold their claim; ignoring them would let a new pane adopt a
	// finished session's history.
	store := &identityStoreStub{rows: []sessionstore.Metadata{
		{ID: "pane-old", Status: sessionstore.StatusDismissed, AgentType: sessionstore.AgentClaude, AgentSessionID: "agent-1", CWD: cwd},
		{ID: "pane-new", Status: sessionstore.StatusLive, AgentType: sessionstore.AgentClaude, CWD: cwd},
	}}
	resolver := newResolverForTest(t, home, store, []agentProcess{
		{PID: 1, SessionID: "pane-new", WorkingDir: cwd, StartedAt: started},
	})

	if adopted := resolver.Resolve(context.Background()); adopted != 0 {
		t.Fatalf("adopted = %d, want 0", adopted)
	}
}

func TestResolverIgnoresTranscriptsThatPredateTheProcess(t *testing.T) {
	home := t.TempDir()
	cwd := "/work/project"
	started := time.Now()
	// Written and last touched well before this process existed, so it cannot
	// be the transcript the process is currently writing.
	writeTranscript(t, claudeProjectDir(home, cwd), "agent-old", started.Add(-2*time.Hour), started.Add(-time.Hour))

	store := &identityStoreStub{rows: []sessionstore.Metadata{
		{ID: "pane-1", Status: sessionstore.StatusLive, AgentType: sessionstore.AgentClaude, CWD: cwd},
	}}
	resolver := newResolverForTest(t, home, store, []agentProcess{
		{PID: 1, SessionID: "pane-1", WorkingDir: cwd, StartedAt: started},
	})

	if adopted := resolver.Resolve(context.Background()); adopted != 0 {
		t.Fatalf("adopted = %d, want 0", adopted)
	}
}

func TestResolverSkipsPanesWithNoRunningAgent(t *testing.T) {
	home := t.TempDir()
	cwd := "/work/project"
	writeTranscript(t, claudeProjectDir(home, cwd), "agent-1", time.Now(), time.Now())

	store := &identityStoreStub{rows: []sessionstore.Metadata{
		{ID: "pane-1", Status: sessionstore.StatusLive, AgentType: sessionstore.AgentClaude, CWD: cwd},
	}}
	// No process reports this pane — there is nothing to attribute.
	resolver := newResolverForTest(t, home, store, nil)

	if adopted := resolver.Resolve(context.Background()); adopted != 0 {
		t.Fatalf("adopted = %d, want 0", adopted)
	}
}

func TestResolverAssignsEachPaneItsOwnTranscript(t *testing.T) {
	home := t.TempDir()
	cwd := "/work/project"
	dir := claudeProjectDir(home, cwd)
	now := time.Now()

	// Two panes with clearly separated start times, each followed by its own
	// transcript. Every transcript nominates the newest process that had
	// already started, so both pairings are unambiguous.
	firstStart := now.Add(-60 * time.Minute)
	secondStart := now.Add(-30 * time.Minute)
	writeTranscript(t, dir, "agent-first", firstStart.Add(time.Minute), now)
	writeTranscript(t, dir, "agent-second", secondStart.Add(time.Minute), now)

	store := &identityStoreStub{rows: []sessionstore.Metadata{
		{ID: "pane-first", Status: sessionstore.StatusLive, AgentType: sessionstore.AgentClaude, CWD: cwd},
		{ID: "pane-second", Status: sessionstore.StatusLive, AgentType: sessionstore.AgentClaude, CWD: cwd},
	}}
	resolver := newResolverForTest(t, home, store, []agentProcess{
		{PID: 1, SessionID: "pane-first", WorkingDir: cwd, StartedAt: firstStart},
		{PID: 2, SessionID: "pane-second", WorkingDir: cwd, StartedAt: secondStart},
	})

	if adopted := resolver.Resolve(context.Background()); adopted != 2 {
		t.Fatalf("adopted = %d, want 2", adopted)
	}
	if got := store.updates["pane-first"].AgentSessionID; got != "agent-first" {
		t.Errorf("pane-first bound to %q, want %q", got, "agent-first")
	}
	if got := store.updates["pane-second"].AgentSessionID; got != "agent-second" {
		t.Errorf("pane-second bound to %q, want %q", got, "agent-second")
	}
}

func TestClaudeTranscriptFirstEntryTimeSkipsUntimestampedPreamble(t *testing.T) {
	dir := t.TempDir()
	want := time.Now().UTC().Truncate(time.Second)
	path := writeTranscript(t, dir, "agent-1", want, time.Now())

	got, ok := claudeTranscriptFirstEntryTime(path)
	if !ok {
		t.Fatal("expected a first entry time")
	}
	if !got.Equal(want) {
		t.Errorf("first entry = %s, want %s", got, want)
	}
}

func TestClaudeTranscriptFirstEntryTimeReportsNoMessagesYet(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent-1.jsonl")
	// Preamble only: a session that exists but has said nothing. There is
	// nothing to attribute and nothing to show.
	if err := os.WriteFile(path, []byte(`{"type":"mode","mode":"normal"}`+"\n"), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	if _, ok := claudeTranscriptFirstEntryTime(path); ok {
		t.Error("a transcript with no timestamped entry must not report a time")
	}
}

func TestClaudeProjectDirMatchesTranscriptPathEncoding(t *testing.T) {
	home := "/home/user"
	cwd := "/home/user/Vrooli"
	// The two encoders must not drift: discovery lists a directory it cannot
	// yet name a file inside, and the tailer then builds the file path.
	wantDir := claudeProjectDir(home, cwd)
	gotPath := claudeTranscriptPath(home, cwd, "agent-1")
	if filepath.Dir(gotPath) != wantDir {
		t.Errorf("claudeTranscriptPath dir = %q, want %q", filepath.Dir(gotPath), wantDir)
	}
}

func TestResolverIgnoresNestedAgentInvocations(t *testing.T) {
	// Found in live validation, not in review. Every process started inside a
	// pane inherits WC_WEB_CONSOLE_SESSION_ID, so a nested `claude` call — a
	// subagent, a scripted one-shot, an agent the agent itself launched —
	// reports the same session as the pane's interactive agent. It starts
	// later and runs in whatever directory that command happened to use, so
	// preferring the newest process sent the transcript search into an
	// unrelated project folder and the pane silently never resolved.
	home := t.TempDir()
	paneCWD := "/work/project"
	nestedCWD := "/work/project/subdir"
	interactiveStart := time.Now().Add(-40 * time.Minute)
	writeTranscript(t, claudeProjectDir(home, paneCWD), "agent-1", interactiveStart.Add(time.Minute), time.Now())

	store := &identityStoreStub{rows: []sessionstore.Metadata{
		{ID: "pane-1", Status: sessionstore.StatusLive, AgentType: sessionstore.AgentClaude, CWD: ""},
	}}
	resolver := newResolverForTest(t, home, store, []agentProcess{
		{PID: 100, SessionID: "pane-1", WorkingDir: paneCWD, StartedAt: interactiveStart},
		// Started later, from inside the pane's own agent, elsewhere on disk.
		{PID: 200, SessionID: "pane-1", WorkingDir: nestedCWD, StartedAt: time.Now().Add(-time.Minute)},
	})

	if adopted := resolver.Resolve(context.Background()); adopted != 1 {
		t.Fatalf("adopted = %d, want 1 — the pane's own agent must win over a nested invocation", adopted)
	}
	if got := store.updates["pane-1"].CWD; got != paneCWD {
		t.Errorf("CWD = %q, want the interactive agent's directory %q", got, paneCWD)
	}
	if got := store.updates["pane-1"].AgentSessionID; got != "agent-1" {
		t.Errorf("AgentSessionID = %q, want %q", got, "agent-1")
	}
}

func TestResolverSkipsProcessesWithNoStartTime(t *testing.T) {
	home := t.TempDir()
	cwd := "/work/project"
	writeTranscript(t, claudeProjectDir(home, cwd), "agent-1", time.Now().Add(-time.Minute), time.Now())

	store := &identityStoreStub{rows: []sessionstore.Metadata{
		{ID: "pane-1", Status: sessionstore.StatusLive, AgentType: sessionstore.AgentClaude, CWD: cwd},
	}}
	// A host where the start time could not be read gives no ordering signal,
	// so the pane must be left alone rather than matched on directory alone.
	resolver := newResolverForTest(t, home, store, []agentProcess{
		{PID: 1, SessionID: "pane-1", WorkingDir: cwd},
	})

	if adopted := resolver.Resolve(context.Background()); adopted != 0 {
		t.Fatalf("adopted = %d, want 0", adopted)
	}
}

func TestResolverLogsAmbiguityOncePerCandidateSet(t *testing.T) {
	// Resolve runs on the transcript poller's cadence. An unresolvable session
	// must not reprint the same standoff every couple of seconds forever.
	home := t.TempDir()
	cwd := "/work/project"
	dir := claudeProjectDir(home, cwd)
	base := time.Now().Add(-10 * time.Minute)
	writeTranscript(t, dir, "agent-a", base.Add(time.Second), time.Now())
	writeTranscript(t, dir, "agent-b", base.Add(2*time.Second), time.Now())

	store := &identityStoreStub{rows: []sessionstore.Metadata{
		{ID: "pane-1", Status: sessionstore.StatusLive, AgentType: sessionstore.AgentClaude, CWD: cwd},
	}}
	resolver := newResolverForTest(t, home, store, []agentProcess{
		{PID: 1, SessionID: "pane-1", WorkingDir: cwd, StartedAt: base},
	})
	var sink strings.Builder
	resolver.logger = log.New(&sink, "", 0)
	resolver.reported = map[string]string{}

	for range 5 {
		resolver.Resolve(context.Background())
	}
	if got := strings.Count(sink.String(), "leaving unidentified"); got != 1 {
		t.Fatalf("logged %d times across 5 passes, want 1", got)
	}

	// A new candidate changes the situation, which is worth saying again.
	writeTranscript(t, dir, "agent-c", base.Add(3*time.Second), time.Now())
	resolver.Resolve(context.Background())
	if got := strings.Count(sink.String(), "leaving unidentified"); got != 2 {
		t.Fatalf("logged %d times after the candidate set changed, want 2", got)
	}
}
