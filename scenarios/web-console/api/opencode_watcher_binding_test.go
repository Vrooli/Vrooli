package main

import (
	"context"
	"testing"
	"time"

	"web-console/backends/opencode"
	"web-console/internal/sessionstore"

	opencodeversion "github.com/vrooli/vrooli/packages/opencode-version"
)

// These tests pin the failure observed on 2026-11-24..26: every OpenCode pane
// ran in the repository root, so attribution by directory + time saw them all
// as candidates for every session. Pane d6d6d301 (created 14:42:46Z) matched its
// own root session plus two subagent children of another pane's session that
// kept updating at 14:48 and 14:53, was skipped as ambiguous, stayed unbound —
// and from then on every later root session matched it as well as the new
// pane, so no OpenCode pane was ever attributed again and Messages stayed empty.

const sharedRepoDir = "/home/operator/Vrooli"

type openCodeBindingFixture struct {
	srv  *Server
	w    *OpenCodeWatcher
	root string
}

func newOpenCodeBindingFixture(t *testing.T) *openCodeBindingFixture {
	t.Helper()
	srv, w, firstPane := newOpenCodeWatcherTest(t)
	// The default fixture pane lives in /work; retire it so each test declares
	// exactly the panes of the scenario it reproduces.
	if err := srv.sessionStore.MarkArchived(context.Background(), firstPane, time.Now(), ""); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	w.bindingRoot = func() string { return root }
	return &openCodeBindingFixture{srv: srv, w: w, root: root}
}

// pane creates a live opencode pane in the shared repository directory.
func (f *openCodeBindingFixture) pane(t *testing.T, created time.Time, agentSessionID string) string {
	t.Helper()
	sess, err := f.srv.sessions.Create(context.Background(), "", 80, 24, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.srv.sessions.Delete(context.Background(), sess.ID) })
	if err := f.srv.sessionStore.Save(context.Background(), sessionstore.Metadata{
		ID: sess.ID, AgentType: sessionstore.AgentOpenCode, CWD: sharedRepoDir, Created: created, AgentSessionID: agentSessionID,
	}); err != nil {
		t.Fatal(err)
	}
	return sess.ID
}

func (f *openCodeBindingFixture) bind(t *testing.T, paneID, nativeID string, major int) {
	t.Helper()
	if err := opencodeversion.WritePaneBinding(f.root, paneID, opencodeversion.PaneBinding{
		NativeSessionID: nativeID, ProviderMajor: major, BoundAt: time.Now(), Source: opencodeversion.PaneBindingSourceLauncher,
	}); err != nil {
		t.Fatal(err)
	}
}

func (f *openCodeBindingFixture) agentSession(t *testing.T, paneID string) string {
	t.Helper()
	meta, err := f.srv.sessionStore.Get(context.Background(), paneID)
	if err != nil {
		t.Fatal(err)
	}
	return meta.AgentSessionID
}

func (f *openCodeBindingFixture) texts(paneID string) []string {
	var out []string
	for _, event := range f.srv.conversations.ListSession(context.Background(), paneID).Events {
		out = append(out, event.Text)
	}
	return out
}

func repoSession(id, parent string, created, updated time.Time) opencode.Session {
	s := opencode.Session{ID: id, ParentID: parent, Directory: sharedRepoDir}
	s.Time.Created = created.UnixMilli()
	s.Time.Updated = updated.UnixMilli()
	return s
}

func TestOpenCodeWatcherBindsSharedDirectoryPanesByLaunchIdentity(t *testing.T) {
	f := newOpenCodeBindingFixture(t)
	base := time.Now().Add(-2 * time.Hour)
	stale := f.pane(t, base, "") // the never-attributed pane that poisoned the fallback
	first := f.pane(t, base.Add(10*time.Minute), "")
	second := f.pane(t, base.Add(20*time.Minute), "")
	waiting := f.pane(t, base.Add(30*time.Minute), "")
	f.bind(t, first, "ses_first00000000000000001", 2)
	f.bind(t, second, "ses_second0000000000000001", 2)
	f.bind(t, waiting, "ses_waiting000000000000001", 2) // no prompt yet: not created
	f.w.major = 2

	client := &fakeOpenCodeClient{
		sessions: []opencode.Session{
			repoSession("ses_first00000000000000001", "", base.Add(11*time.Minute), base.Add(40*time.Minute)),
			repoSession("ses_second0000000000000001", "", base.Add(21*time.Minute), base.Add(40*time.Minute)),
			repoSession("ses_child00000000000000001", "ses_first00000000000000001", base.Add(25*time.Minute), base.Add(45*time.Minute)),
		},
		messages: map[string][]opencode.MessageWithParts{
			"ses_first00000000000000001": {ocUser(10, "first pane prompt")},
			"ses_second0000000000000001": {ocUser(20, "second pane prompt")},
		},
	}
	f.w.reconcileAll(context.Background(), client)

	if got := f.agentSession(t, first); got != "ses_first00000000000000001" {
		t.Fatalf("first pane bound to %q", got)
	}
	if got := f.agentSession(t, second); got != "ses_second0000000000000001" {
		t.Fatalf("second pane bound to %q", got)
	}
	if got := f.texts(first); len(got) != 1 || got[0] != "first pane prompt" {
		t.Fatalf("first pane messages = %q", got)
	}
	if got := f.texts(second); len(got) != 1 || got[0] != "second pane prompt" {
		t.Fatalf("second pane messages = %q", got)
	}
	// A binding whose session does not exist yet waits; it is never handed a
	// neighbour's session. The stale unbound pane is left to the fallback, which
	// only ever saw the subagent child (excluded) and claimed roots.
	if got := f.agentSession(t, waiting); got != "" {
		t.Fatalf("waiting pane bound early to %q", got)
	}
	if got := f.agentSession(t, stale); got != "" || len(f.texts(stale)) != 0 {
		t.Fatalf("stale pane bound to %q with %q", got, f.texts(stale))
	}

	// The first prompt creates the preselected session; the next pass binds it.
	client.mu.Lock()
	client.sessions = append(client.sessions, repoSession("ses_waiting000000000000001", "", time.Now(), time.Now()))
	client.messages["ses_waiting000000000000001"] = []opencode.MessageWithParts{ocUser(30, "late first prompt")}
	client.mu.Unlock()
	f.w.reconcileAll(context.Background(), client)
	if got := f.texts(waiting); len(got) != 1 || got[0] != "late first prompt" {
		t.Fatalf("waiting pane messages = %q", got)
	}
}

// The exact first failure: a fresh pane with no binding (OpenCode 1 or an
// older launcher) must not be made ambiguous by another pane's subagents.
func TestOpenCodeWatcherFallbackIgnoresAnotherPanesSubagents(t *testing.T) {
	f := newOpenCodeBindingFixture(t)
	parentCreated := time.Now().Add(-10 * time.Hour)
	busy := f.pane(t, parentCreated.Add(-time.Minute), "ses_df42236f2ffeGFzYqy9wl5dhMU")
	f.w.claimed["ses_df42236f2ffeGFzYqy9wl5dhMU"] = busy
	paneCreated := time.Now().Add(-time.Hour)
	fresh := f.pane(t, paneCreated, "")

	client := &fakeOpenCodeClient{
		sessions: []opencode.Session{
			repoSession("ses_df42236f2ffeGFzYqy9wl5dhMU", "", parentCreated, paneCreated.Add(20*time.Minute)),
			repoSession("ses_df212805cffez3BgOThuejprIF", "ses_df42236f2ffeGFzYqy9wl5dhMU", paneCreated.Add(-23*time.Minute), paneCreated.Add(6*time.Minute)),
			repoSession("ses_df212805cffd1elH61TopLMYjm", "ses_df42236f2ffeGFzYqy9wl5dhMU", paneCreated.Add(-23*time.Minute), paneCreated.Add(11*time.Minute)),
			repoSession("ses_df1fd7927ffe2ea48qbAV0OJgL", "", paneCreated.Add(10*time.Second), paneCreated.Add(30*time.Minute)),
		},
		messages: map[string][]opencode.MessageWithParts{
			"ses_df1fd7927ffe2ea48qbAV0OJgL": {ocUser(10, "fresh pane prompt")},
		},
	}
	f.w.reconcileAll(context.Background(), client)

	if got := f.agentSession(t, fresh); got != "ses_df1fd7927ffe2ea48qbAV0OJgL" {
		t.Fatalf("fresh pane bound to %q; subagent children must not make it ambiguous", got)
	}
	if got := f.texts(fresh); len(got) != 1 || got[0] != "fresh pane prompt" {
		t.Fatalf("fresh pane messages = %q", got)
	}
}

func TestOpenCodeWatcherRebindsPaneWhenOpenCodeStartsAgain(t *testing.T) {
	f := newOpenCodeBindingFixture(t)
	created := time.Now().Add(-time.Hour)
	pane := f.pane(t, created, "ses_earlier000000000000001")
	f.w.claimed["ses_earlier000000000000001"] = pane
	f.bind(t, pane, "ses_later00000000000000001", 2)

	client := &fakeOpenCodeClient{
		sessions: []opencode.Session{
			repoSession("ses_earlier000000000000001", "", created, created),
			repoSession("ses_later00000000000000001", "", time.Now(), time.Now()),
		},
		messages: map[string][]opencode.MessageWithParts{
			"ses_earlier000000000000001": {ocUser(10, "earlier run")},
			"ses_later00000000000000001": {ocUser(20, "later run")},
		},
	}
	f.w.reconcileAll(context.Background(), client)

	if got := f.agentSession(t, pane); got != "ses_later00000000000000001" {
		t.Fatalf("pane bound to %q, want the relaunched session", got)
	}
	got := f.texts(pane)
	if len(got) != 2 {
		t.Fatalf("pane messages = %q, want both runs", got)
	}
}

func TestOpenCodeWatcherLeavesOtherMajorsBindingToItsCapture(t *testing.T) {
	f := newOpenCodeBindingFixture(t)
	created := time.Now().Add(-time.Hour)
	pane := f.pane(t, created, "")
	f.bind(t, pane, "ses_v2pane0000000000000001", 2)
	f.w.major = 1

	client := &fakeOpenCodeClient{
		sessions: []opencode.Session{repoSession("ses_v1stranger00000000001", "", time.Now(), time.Now())},
		messages: map[string][]opencode.MessageWithParts{"ses_v1stranger00000000001": {ocUser(10, "not this pane")}},
	}
	f.w.reconcileAll(context.Background(), client)

	if got := f.agentSession(t, pane); got != "" {
		t.Fatalf("V1 capture bound a V2-launched pane to %q", got)
	}
}
