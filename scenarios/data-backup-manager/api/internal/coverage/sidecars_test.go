package coverage_test

import (
	"context"
	"testing"

	"data-backup-manager/internal/coverage"
	"data-backup-manager/internal/sources"
)

func sidecarSuggestion(name, locator string, kind sources.SourceKind) coverage.Suggestion {
	return coverage.Suggestion{ID: "id-" + name, Owner: "swarm-manager", Name: name, SourceKind: kind, Locator: locator, Sensitive: true}
}

func TestFoldSQLiteSidecars_SidecarTravelsWithItsDatabase(t *testing.T) {
	in := []coverage.Suggestion{
		sidecarSuggestion("events-db", "/data/events.db", sources.KindSQLite),
		sidecarSuggestion("events-db-wal", "/data/events.db-wal", sources.KindFilesystem),
		sidecarSuggestion("events-db-shm", "/data/events.db-shm", sources.KindFilesystem),
		sidecarSuggestion("notes", "/data/notes", sources.KindFilesystem),
	}
	got := coverage.FoldSQLiteSidecars(in, nil)
	if len(got) != 2 || got[0].Name != "events-db" || got[1].Name != "notes" {
		t.Fatalf("folded list = %+v, want the database and the unrelated item only", got)
	}
	if want := []string{"-wal", "-shm"}; len(got[0].Sidecars) != 2 || got[0].Sidecars[0] != want[0] || got[0].Sidecars[1] != want[1] {
		t.Fatalf("database sidecars = %v, want %v", got[0].Sidecars, want)
	}
	if len(got[1].Sidecars) != 0 {
		t.Fatalf("an unrelated item gained sidecars: %v", got[1].Sidecars)
	}
}

func TestFoldSQLiteSidecars_RegisteredDatabaseAlsoClaimsItsSidecar(t *testing.T) {
	in := []coverage.Suggestion{sidecarSuggestion("events-db-shm", "/data/events.db-shm", sources.KindFilesystem)}
	registered := []coverage.CatalogTarget{{ID: "t1", Locator: "/data/events.db"}}
	if got := coverage.FoldSQLiteSidecars(in, registered); len(got) != 0 {
		t.Fatalf("sidecar of a registered database stayed as an item: %+v", got)
	}
}

func TestFoldSQLiteSidecars_OrphanSidecarStaysVisible(t *testing.T) {
	in := []coverage.Suggestion{sidecarSuggestion("lost-shm", "/data/lost.db-shm", sources.KindFilesystem)}
	if got := coverage.FoldSQLiteSidecars(in, nil); len(got) != 1 {
		t.Fatalf("a sidecar with no known database was hidden: %+v", got)
	}
}

func TestReport_SensitiveListHasNoSidecarItems(t *testing.T) {
	svc := coverage.NewService(coverage.Deps{
		Suggestions: &fakeSuggestions{list: []coverage.Suggestion{
			sidecarSuggestion("events-db", "/data/events.db", sources.KindSQLite),
			sidecarSuggestion("events-db-shm", "/data/events.db-shm", sources.KindFilesystem),
		}},
		Targets: &fakeTargets{}, Plans: &fakePlans{}, Runs: &fakeRuns{}, Restores: &fakeRestores{},
	})
	rep, err := svc.Report(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Sensitive) != 1 || rep.Summary.SensitiveCount != 1 || rep.Sensitive[0].Name != "events-db" {
		t.Fatalf("sensitive = %+v (count %d), want only the database", rep.Sensitive, rep.Summary.SensitiveCount)
	}
}

func TestAcceptSensitive_RefusesAFoldedSidecar(t *testing.T) {
	targets := &fakeTargets{}
	svc := coverage.NewService(coverage.Deps{
		Suggestions: &fakeSuggestions{list: []coverage.Suggestion{
			sidecarSuggestion("events-db", "/data/events.db", sources.KindSQLite),
			sidecarSuggestion("events-db-shm", "/data/events.db-shm", sources.KindFilesystem),
		}},
		Targets: targets,
	})
	if _, err := svc.AcceptSensitive(context.Background(), []string{"id-events-db-shm"}); err == nil {
		t.Fatal("a folded sidecar was accepted as its own target")
	}
	if len(targets.registered) != 0 {
		t.Fatalf("registered %+v", targets.registered)
	}
}
