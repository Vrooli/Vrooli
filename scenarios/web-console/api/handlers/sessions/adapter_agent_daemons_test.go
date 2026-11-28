package sessions

import (
	"context"
	"errors"
	"sort"
	"testing"
	"time"

	"web-console/internal/events"
	"web-console/internal/metrics"
	"web-console/internal/sessionstore"
	"web-console/internal/workspace"
)

// fakeAgentDaemons models the per-home daemons: running holds the sessions
// whose daemon is up, Stop takes one down and records the order of events.
type fakeAgentDaemons struct {
	running map[string]AgentDaemonUsage
	stopped []string
	log     *[]string
	stopErr error
	// afterRunning runs after Running reports, to model a session changing
	// between the scan and the stop.
	afterRunning func()
}

func newFakeAgentDaemons(ids ...string) *fakeAgentDaemons {
	f := &fakeAgentDaemons{running: map[string]AgentDaemonUsage{}, log: &[]string{}}
	for _, id := range ids {
		f.running[id] = AgentDaemonUsage{Processes: 3, MemoryBytes: 100}
	}
	return f
}

func (f *fakeAgentDaemons) Running(context.Context) (map[string]AgentDaemonUsage, error) {
	out := make(map[string]AgentDaemonUsage, len(f.running))
	for id, usage := range f.running {
		out[id] = usage
	}
	if f.afterRunning != nil {
		f.afterRunning()
	}
	return out, nil
}

func (f *fakeAgentDaemons) Stop(_ context.Context, id string) (AgentDaemonUsage, error) {
	if f.stopErr != nil {
		return AgentDaemonUsage{}, f.stopErr
	}
	usage := f.running[id]
	delete(f.running, id)
	f.stopped = append(f.stopped, id)
	*f.log = append(*f.log, "stop:"+id)
	return usage, nil
}

func (f *fakeAgentDaemons) isRunning(id string) bool { _, ok := f.running[id]; return ok }

func daemonTestAdapter(store sessionstore.Store, daemons *fakeAgentDaemons) *Adapter {
	return &Adapter{
		Store: store, Manager: emptySessionManager{}, Workspace: workspace.NewMemStore(),
		Events: events.NewLogger(20), Metrics: metrics.New(), Conversations: adapterConversationStub{},
		AgentDaemons: daemons,
	}
}

func saveRow(t *testing.T, store sessionstore.Store, meta sessionstore.Metadata) {
	t.Helper()
	meta.AgentType, meta.Created = sessionstore.AgentCodex, time.Now().UTC()
	if err := store.Save(context.Background(), meta); err != nil {
		t.Fatal(err)
	}
}

func TestAdapterArchiveStopsSessionAgentDaemon(t *testing.T) {
	ctx := context.Background()
	store := sessionstore.NewInMemory()
	saveRow(t, store, sessionstore.Metadata{ID: "archiving"})
	saveRow(t, store, sessionstore.Metadata{ID: "active"})
	daemons := newFakeAgentDaemons("archiving", "active")
	a := daemonTestAdapter(store, daemons)

	if err := a.ArchiveWithNote(ctx, "archiving", "idle policy"); err != nil {
		t.Fatal(err)
	}
	if daemons.isRunning("archiving") {
		t.Fatal("archived session's daemon is still running")
	}
	if !daemons.isRunning("active") {
		t.Fatal("archiving one session stopped another session's daemon")
	}
}

func TestAdapterArchiveSurvivesDaemonStopFailure(t *testing.T) {
	ctx := context.Background()
	store := sessionstore.NewInMemory()
	saveRow(t, store, sessionstore.Metadata{ID: "stuck"})
	daemons := newFakeAgentDaemons("stuck")
	daemons.stopErr = errors.New("still running after kill")
	a := daemonTestAdapter(store, daemons)

	if err := a.Archive(ctx, "stuck"); err != nil {
		t.Fatalf("a daemon that will not stop must not undo the archive: %v", err)
	}
	if meta, err := store.Get(ctx, "stuck"); err != nil || meta.ArchivedAt.IsZero() {
		t.Fatalf("session not archived: %+v, %v", meta, err)
	}
}

func TestAdapterDismissStopsSessionAgentDaemon(t *testing.T) {
	ctx := context.Background()
	store := sessionstore.NewInMemory()
	saveRow(t, store, sessionstore.Metadata{ID: "orphan", Status: sessionstore.StatusAwaitingRecovery, Detached: true})
	daemons := newFakeAgentDaemons("orphan")
	a := daemonTestAdapter(store, daemons)

	if err := a.DismissRecoverable(ctx, "orphan"); err != nil {
		t.Fatal(err)
	}
	if daemons.isRunning("orphan") {
		t.Fatal("dismissed session's daemon is still running")
	}
}

func TestAdapterDeleteStopsDaemonBeforeRemovingHome(t *testing.T) {
	ctx := context.Background()
	store := sessionstore.NewInMemory()
	saveRow(t, store, sessionstore.Metadata{ID: "doomed", ArchivedAt: time.Now().UTC()})
	daemons := newFakeAgentDaemons("doomed")
	a := daemonTestAdapter(store, daemons)
	a.RemoveAgentHomes = func(id string) error {
		*daemons.log = append(*daemons.log, "remove:"+id)
		return nil
	}
	if err := a.Delete(ctx, "doomed"); err != nil {
		t.Fatal(err)
	}
	if got := *daemons.log; len(got) != 2 || got[0] != "stop:doomed" || got[1] != "remove:doomed" {
		t.Fatalf("order = %v, want daemon stopped before its home is removed", got)
	}
}

func TestAdapterPruneReconcilesLeakedDaemonsOfRetiredSessionsOnly(t *testing.T) {
	ctx := context.Background()
	store := sessionstore.NewInMemory()
	saveRow(t, store, sessionstore.Metadata{ID: "archived", ArchivedAt: time.Now().UTC()})
	saveRow(t, store, sessionstore.Metadata{ID: "dismissed", Status: sessionstore.StatusDismissed})
	saveRow(t, store, sessionstore.Metadata{ID: "active"})
	saveRow(t, store, sessionstore.Metadata{ID: "orphan-awaiting", Status: sessionstore.StatusAwaitingRecovery, Detached: true})
	daemons := newFakeAgentDaemons("archived", "dismissed", "active", "orphan-awaiting", "no-row")
	a := daemonTestAdapter(store, daemons)

	dry, err := a.PruneArchive(ctx, false)
	if err != nil {
		t.Fatal(err)
	}
	if got := daemonActionIDs(dry); len(got) != 2 || got[0] != "archived" || got[1] != "dismissed" {
		t.Fatalf("dry run daemon actions = %v", got)
	}
	for _, action := range dry.Actions {
		if action.Kind != PruneAgentDaemon || action.Applied || action.Bytes != 100 {
			t.Fatalf("dry-run action = %+v", action)
		}
	}
	if len(daemons.stopped) != 0 || dry.ReclaimedBytes != 0 {
		t.Fatalf("dry run stopped %v and reclaimed %d disk bytes", daemons.stopped, dry.ReclaimedBytes)
	}

	applied, err := a.PruneArchive(ctx, true)
	if err != nil {
		t.Fatal(err)
	}
	if got := daemonActionIDs(applied); len(got) != 2 {
		t.Fatalf("applied daemon actions = %v", got)
	}
	for _, id := range []string{"archived", "dismissed"} {
		if daemons.isRunning(id) {
			t.Errorf("retired session %s still has a daemon", id)
		}
	}
	for _, id := range []string{"active", "orphan-awaiting", "no-row"} {
		if !daemons.isRunning(id) {
			t.Errorf("daemon of %s, which is not archived or dismissed, was stopped", id)
		}
	}
	for _, action := range applied.Actions {
		if !action.Applied || action.Bytes != 100 {
			t.Errorf("applied action = %+v", action)
		}
	}
	again, err := a.PruneArchive(ctx, true)
	if err != nil || len(daemonActionIDs(again)) != 0 {
		t.Fatalf("second run = %+v, %v; reconcile must be idempotent", again, err)
	}
}

func TestAdapterPruneKeepsDaemonOfSessionUnarchivedSinceScan(t *testing.T) {
	ctx := context.Background()
	store := sessionstore.NewInMemory()
	saveRow(t, store, sessionstore.Metadata{ID: "back", ArchivedAt: time.Now().UTC()})
	daemons := newFakeAgentDaemons("back")
	daemons.afterRunning = func() { _ = store.MarkUnarchived(ctx, "back") }
	a := daemonTestAdapter(store, daemons)

	if _, err := a.PruneArchive(ctx, true); err != nil {
		t.Fatal(err)
	}
	if !daemons.isRunning("back") {
		t.Fatal("a session unarchived between scan and stop lost its daemon")
	}
}

func TestAdapterPruneReportsDaemonStopFailure(t *testing.T) {
	store := sessionstore.NewInMemory()
	saveRow(t, store, sessionstore.Metadata{ID: "archived", ArchivedAt: time.Now().UTC()})
	daemons := newFakeAgentDaemons("archived")
	daemons.stopErr = errors.New("still running after kill")
	a := daemonTestAdapter(store, daemons)

	if _, err := a.PruneArchive(context.Background(), true); !errors.Is(err, ErrInternal) {
		t.Fatalf("err = %v, want a surfaced stop failure", err)
	}
}

func daemonActionIDs(result ArchivePruneResult) []string {
	var ids []string
	for _, action := range result.Actions {
		if action.Kind == PruneAgentDaemon {
			ids = append(ids, action.SessionID)
		}
	}
	sort.Strings(ids)
	return ids
}
