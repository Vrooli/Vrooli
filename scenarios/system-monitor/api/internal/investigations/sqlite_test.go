package investigations_test

import (
	"context"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/investigations"
	sqliterepo "github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository/sqlite"
)

func TestSaveAndListRuns(t *testing.T) {
	repo, err := sqliterepo.NewInMemoryRepository()
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	store := investigations.NewSQLiteRepository(repo.RoutedDB())
	now := time.Now().UTC().Truncate(time.Microsecond)
	run := investigations.Run{ID: "run-1", EntryID: "cpu", ExecutionMode: "native", Status: "completed", StartedAt: now, CompletedAt: now.Add(time.Second), DurationSeconds: 1, HostOS: "linux", HostArch: "amd64", ResultJSON: `{"ok":true}`, Findings: []investigations.Finding{{Severity: "info", Code: "ok", Summary: "healthy"}}}
	if err := store.SaveRun(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	got, err := store.ListRuns(context.Background(), "", time.Time{}, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || len(got[0].Findings) != 1 || got[0].StartedAt.Location() != time.UTC {
		t.Fatalf("runs = %#v", got)
	}
}

func TestPruneRunsBeforeDeletesFindings(t *testing.T) {
	repo, err := sqliterepo.NewInMemoryRepository()
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	store := investigations.NewSQLiteRepository(repo.RoutedDB())
	old := time.Now().UTC().Add(-48 * time.Hour)
	run := investigations.Run{ID: "old", EntryID: "cpu", ExecutionMode: "native", Status: "completed", StartedAt: old, CompletedAt: old.Add(time.Second), HostOS: "linux", HostArch: "amd64", Findings: []investigations.Finding{{Severity: "warning", Code: "pressure", Summary: "pressure"}}}
	if err := store.SaveRun(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	deleted, err := store.PruneRunsBefore(context.Background(), time.Now().UTC().Add(-24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 1 {
		t.Fatalf("deleted = %d, want 1", deleted)
	}
	if _, err := store.GetRun(context.Background(), "old"); err == nil {
		t.Fatal("expected pruned run to be absent")
	}
}

func TestTimestampsAreUTCRFC3339(t *testing.T) {
	repo, err := sqliterepo.NewInMemoryRepository()
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	store := investigations.NewSQLiteRepository(repo.RoutedDB())
	now := time.Date(2026, 8, 26, 12, 0, 0, 0, time.FixedZone("EDT", -4*60*60))
	if err := store.SaveRun(context.Background(), investigations.Run{ID: "tz", EntryID: "cpu", ExecutionMode: "native", Status: "completed", StartedAt: now, CompletedAt: now.Add(time.Second), HostOS: "linux", HostArch: "amd64"}); err != nil {
		t.Fatal(err)
	}
	run, err := store.GetRun(context.Background(), "tz")
	if err != nil {
		t.Fatal(err)
	}
	if run.StartedAt.Location() != time.UTC || run.StartedAt.Format(time.RFC3339) != "2026-08-26T16:00:00Z" {
		t.Fatalf("timestamp = %s", run.StartedAt.Format(time.RFC3339))
	}
}
