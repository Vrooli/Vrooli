package orchestration

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"agent-manager/internal/adapters/database"
	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/webconsole"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/testutil"

	"github.com/google/uuid"
)

// inventorySessions is a web-console inventory fake for the retention sweep:
// it serves a fixed session list and records every call that would touch a
// session.
type inventorySessions struct {
	mu        sync.Mutex
	listed    []webconsole.SessionInfo
	listErr   error
	listCalls int
	archived  []string
}

func (f *inventorySessions) ListSessions(context.Context) ([]webconsole.SessionInfo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.listCalls++
	if f.listErr != nil {
		return nil, f.listErr
	}
	return slices.Clone(f.listed), nil
}

func (f *inventorySessions) ArchiveSession(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.archived = append(f.archived, id)
	return nil
}

func (f *inventorySessions) archivedIDs() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.archived)
}

func (f *inventorySessions) CreateSession(context.Context, webconsole.CreateSessionParams) (string, error) {
	return "", errors.New("retention sweep must not create sessions")
}

func (f *inventorySessions) GetSession(_ context.Context, id string) (webconsole.SessionInfo, error) {
	return webconsole.SessionInfo{ID: id, Owner: webconsole.OwnerAgentManager}, nil
}

func (f *inventorySessions) SendText(context.Context, string, string, string) error {
	return errors.New("retention sweep must not type into sessions")
}

func (f *inventorySessions) SendPrompt(context.Context, string, string, string) (webconsole.PromptSubmission, error) {
	return webconsole.PromptSubmission{}, errors.New("retention sweep must not type into sessions")
}

func (f *inventorySessions) Interrupt(context.Context, string, string) error {
	return errors.New("retention sweep must not interrupt sessions")
}

func (f *inventorySessions) Screen(context.Context, string, bool) (string, error) { return "", nil }

var retentionNow = time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)

func agentManagerSession(id string, created time.Time) webconsole.SessionInfo {
	return webconsole.SessionInfo{ID: id, Owner: webconsole.OwnerAgentManager, Origin: webconsole.OriginProgrammatic, DisplayLabel: "label-" + id, CreatedAt: created}
}

type retentionFixture struct {
	repos      *database.Repositories
	events     event.Store
	sessions   *inventorySessions
	reconciler *Reconciler
	task       *domain.Task
}

func newRetentionFixture(t *testing.T, retention time.Duration) *retentionFixture {
	t.Helper()
	repos, eventStore, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	task := &domain.Task{ID: uuid.New(), Title: "interactive retention", ScopePath: ".", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(context.Background(), task); err != nil {
		t.Fatal(err)
	}
	sessions := &inventorySessions{}
	reconciler := NewReconciler(repos.Runs, nil,
		WithReconcilerEvents(eventStore),
		WithReconcilerInteractive(sessions),
		WithReconcilerClock(func() time.Time { return retentionNow }),
		WithReconcilerConfig(ReconcilerConfig{StaleThreshold: 24 * time.Hour, InteractiveSessionRetention: retention}),
	)
	reconciler.interactiveLiveDrivers = newInteractiveDriverRegistry()
	return &retentionFixture{repos: repos, events: eventStore, sessions: sessions, reconciler: reconciler, task: task}
}

// interactiveRun persists an interactive run bound to sessionID. A nil endedAt
// leaves the run without an end time (as every non-terminal run has).
func (f *retentionFixture) interactiveRun(t *testing.T, sessionID string, status domain.RunStatus, endedAt *time.Time) *domain.Run {
	t.Helper()
	heartbeat := time.Now()
	run := &domain.Run{
		ID:                  uuid.New(),
		TaskID:              f.task.ID,
		Tag:                 uuid.NewString(),
		RunMode:             domain.RunModeInPlace,
		Status:              status,
		Phase:               domain.RunPhaseExecuting,
		ExecutionMode:       domain.ExecutionModeInteractive,
		WebConsoleSessionID: sessionID,
		LastHeartbeat:       &heartbeat,
		EndedAt:             endedAt,
	}
	if status.IsTerminal() {
		run.Phase = domain.RunPhaseCompleted
	}
	if err := f.repos.Runs.Create(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	return run
}

func ago(d time.Duration) *time.Time {
	at := retentionNow.Add(-d)
	return &at
}

func TestReconcilerReleasesInteractiveSessionsOnlyAfterRetention(t *testing.T) {
	f := newRetentionFixture(t, 2*time.Hour)
	old := retentionNow.Add(-72 * time.Hour)
	young := retentionNow.Add(-10 * time.Minute)

	expiredComplete := f.interactiveRun(t, "ended-long-ago", domain.RunStatusComplete, ago(3*time.Hour))
	f.interactiveRun(t, "ended-recently", domain.RunStatusFailed, ago(30*time.Minute))
	f.interactiveRun(t, "still-running", domain.RunStatusRunning, nil)
	f.interactiveRun(t, "parked", domain.RunStatusParked, nil)
	// A session shared by an expired run and a later, still-running run is in use.
	f.interactiveRun(t, "shared", domain.RunStatusCancelled, ago(5*time.Hour))
	f.interactiveRun(t, "shared", domain.RunStatusRunning, nil)
	f.sessions.listed = []webconsole.SessionInfo{
		agentManagerSession("ended-long-ago", old),
		agentManagerSession("ended-recently", old),
		agentManagerSession("still-running", old),
		agentManagerSession("parked", old),
		agentManagerSession("shared", old),
		agentManagerSession("unreferenced-old", old),
		agentManagerSession("unreferenced-young", young),
		agentManagerSession("unreferenced-unknown-age", time.Time{}),
		{ID: "operator-shell", Owner: "", Origin: "SESSION_ORIGIN_UI", CreatedAt: old},
		{ID: "other-owner", Owner: "swarm-manager", Origin: webconsole.OriginProgrammatic, CreatedAt: old},
		{ID: "am-from-ui", Owner: webconsole.OwnerAgentManager, Origin: "SESSION_ORIGIN_UI", CreatedAt: old},
	}

	stats := f.reconciler.RunOnce(context.Background())

	deleted := f.sessions.archivedIDs()
	slices.Sort(deleted)
	if want := []string{"ended-long-ago", "unreferenced-old"}; !slices.Equal(deleted, want) {
		t.Fatalf("released sessions = %v, want %v", deleted, want)
	}
	if stats.InteractiveSessionsReleased != 2 || len(stats.Errors) != 0 {
		t.Fatalf("stats = %+v, want 2 releases and no errors", stats)
	}
	events, err := f.events.Get(context.Background(), expiredComplete.ID, event.GetOptions{AfterSequence: -1})
	if err != nil || len(events) == 0 {
		t.Fatalf("release evidence events=%v err=%v", events, err)
	}
	logData, ok := events[len(events)-1].Data.(*domain.LogEventData)
	if !ok || !strings.Contains(logData.Message, "ended-long-ago") || !strings.Contains(logData.Message, "archived after retention") {
		t.Fatalf("release event = %+v, want a log naming the released session", events[len(events)-1])
	}
	stored, err := f.repos.Runs.Get(context.Background(), expiredComplete.ID)
	if err != nil || stored.Status != domain.RunStatusComplete {
		t.Fatalf("released run = %+v err=%v, want its terminal status untouched", stored, err)
	}
}

func TestReconcilerKeepsExpiredSessionWhileContinuationOwnsRun(t *testing.T) {
	t.Run("live driver", func(t *testing.T) {
		f := newRetentionFixture(t, time.Hour)
		run := f.interactiveRun(t, "continuing", domain.RunStatusComplete, ago(4*time.Hour))
		_, driver := f.reconciler.interactiveLiveDrivers.register(context.Background(), run.ID)
		t.Cleanup(func() { f.reconciler.interactiveLiveDrivers.finish(run.ID, driver) })
		f.sessions.listed = []webconsole.SessionInfo{agentManagerSession("continuing", retentionNow.Add(-5*time.Hour))}

		stats := f.reconciler.RunOnce(context.Background())

		if got := f.sessions.archivedIDs(); len(got) != 0 || stats.InteractiveSessionsReleased != 0 {
			t.Fatalf("released %v with a live continuation driver", got)
		}
	})
	t.Run("continuation admission holds the owner lock", func(t *testing.T) {
		f := newRetentionFixture(t, time.Hour)
		f.interactiveRun(t, "admitting", domain.RunStatusComplete, ago(4*time.Hour))
		f.sessions.listed = []webconsole.SessionInfo{agentManagerSession("admitting", retentionNow.Add(-5*time.Hour))}
		f.reconciler.interactiveRecoveryMu.Lock()
		defer f.reconciler.interactiveRecoveryMu.Unlock()

		f.reconciler.RunOnce(context.Background())

		if got := f.sessions.archivedIDs(); len(got) != 0 {
			t.Fatalf("released %v while continuation admission held the lock", got)
		}
	})
}

func TestReconcilerInteractiveSessionReleaseIsBoundedPerCycle(t *testing.T) {
	f := newRetentionFixture(t, time.Hour)
	total := interactiveSessionReleaseLimit + 5
	for i := range total {
		f.sessions.listed = append(f.sessions.listed, agentManagerSession(fmt.Sprintf("orphan-%02d", i), retentionNow.Add(-48*time.Hour)))
	}

	first := f.reconciler.RunOnce(context.Background())
	if first.InteractiveSessionsReleased != interactiveSessionReleaseLimit || len(f.sessions.archivedIDs()) != interactiveSessionReleaseLimit {
		t.Fatalf("first cycle released %d (stats %d), want limit %d", len(f.sessions.archivedIDs()), first.InteractiveSessionsReleased, interactiveSessionReleaseLimit)
	}
	f.sessions.listed = f.sessions.listed[interactiveSessionReleaseLimit:] // web-console no longer lists released sessions

	second := f.reconciler.RunOnce(context.Background())
	if second.InteractiveSessionsReleased != total-interactiveSessionReleaseLimit || len(f.sessions.archivedIDs()) != total {
		t.Fatalf("second cycle released %d, want the remaining %d", second.InteractiveSessionsReleased, total-interactiveSessionReleaseLimit)
	}
}

func TestReconcilerInteractiveSessionInventoryOutageDoesNotStopCycle(t *testing.T) {
	f := newRetentionFixture(t, time.Hour)
	f.sessions.listErr = errors.New("web-console unavailable")
	pending := &domain.Run{ID: uuid.New(), TaskID: f.task.ID, Status: domain.RunStatusPending, Phase: domain.RunPhaseQueued}
	if err := f.repos.Runs.Create(context.Background(), pending); err != nil {
		t.Fatal(err)
	}
	f.reconciler.config.PendingThreshold = time.Nanosecond

	stats := f.reconciler.RunOnce(context.Background())

	if len(stats.Errors) != 1 || !strings.Contains(stats.Errors[0], "web-console unavailable") {
		t.Fatalf("errors = %v, want the inventory outage recorded once", stats.Errors)
	}
	if len(f.sessions.archivedIDs()) != 0 {
		t.Fatal("deleted sessions without an inventory")
	}
	stored, err := f.repos.Runs.Get(context.Background(), pending.ID)
	if err != nil || stored.Status != domain.RunStatusFailed {
		t.Fatalf("pending run = %+v err=%v, want the rest of the cycle to still reap it", stored, err)
	}
}

func TestReconcilerInteractiveSessionRetentionZeroDisablesSweep(t *testing.T) {
	f := newRetentionFixture(t, 0)
	f.interactiveRun(t, "ended", domain.RunStatusComplete, ago(100*time.Hour))
	f.sessions.listed = []webconsole.SessionInfo{agentManagerSession("ended", retentionNow.Add(-200*time.Hour))}

	f.reconciler.RunOnce(context.Background())

	if f.sessions.listCalls != 0 || len(f.sessions.archivedIDs()) != 0 {
		t.Fatalf("disabled sweep made web-console calls: list=%d delete=%v", f.sessions.listCalls, f.sessions.archivedIDs())
	}
}
