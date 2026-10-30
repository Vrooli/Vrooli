package database

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

func TestListByWebConsoleSessionIDsReturnsEveryReferencingRun(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repos := NewRepositories(db, db.log)
	ctx := context.Background()
	task := &domain.Task{ID: uuid.New(), Title: "web-console lookup", ScopePath: "src/", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	ended := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	create := func(sessionID string, status domain.RunStatus) *domain.Run {
		t.Helper()
		run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Tag: uuid.NewString(), Status: status, Phase: domain.RunPhaseCompleted, ExecutionMode: domain.ExecutionModeInteractive, WebConsoleSessionID: sessionID, EndedAt: &ended}
		if err := repos.Runs.Create(ctx, run); err != nil {
			t.Fatal(err)
		}
		return run
	}
	first := create("wc-shared", domain.RunStatusComplete)
	second := create("wc-shared", domain.RunStatusFailed)
	single := create("wc-single", domain.RunStatusCancelled)
	create("wc-unrequested", domain.RunStatusComplete)
	create("", domain.RunStatusComplete)

	runs, err := repos.Runs.ListByWebConsoleSessionIDs(ctx, []string{"wc-shared", "wc-single", "wc-single", "wc-missing", ""})
	if err != nil {
		t.Fatal(err)
	}
	got := map[uuid.UUID]*domain.Run{}
	for _, run := range runs {
		got[run.ID] = run
	}
	if len(runs) != 3 || got[first.ID] == nil || got[second.ID] == nil || got[single.ID] == nil {
		t.Fatalf("runs = %d %v, want exactly the two shared-session runs and the single-session run", len(runs), got)
	}
	if r := got[single.ID]; r.WebConsoleSessionID != "wc-single" || r.Status != domain.RunStatusCancelled || r.EndedAt == nil || !r.EndedAt.Equal(ended) {
		t.Fatalf("single-session run = %+v, want session, status, and end time round-tripped", r)
	}

	none, err := repos.Runs.ListByWebConsoleSessionIDs(ctx, nil)
	if err != nil || len(none) != 0 {
		t.Fatalf("empty lookup = %v err=%v, want no runs", none, err)
	}
}

func TestListByWebConsoleSessionIDsSpansLookupChunks(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repos := NewRepositories(db, db.log)
	ctx := context.Background()
	task := &domain.Task{ID: uuid.New(), Title: "web-console chunked lookup", ScopePath: "src/", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	ids := make([]string, webConsoleSessionLookupChunk+3)
	for i := range ids {
		ids[i] = fmt.Sprintf("wc-%04d", i)
	}
	for _, sessionID := range []string{ids[0], ids[webConsoleSessionLookupChunk+2]} {
		run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Tag: uuid.NewString(), Status: domain.RunStatusComplete, Phase: domain.RunPhaseCompleted, WebConsoleSessionID: sessionID}
		if err := repos.Runs.Create(ctx, run); err != nil {
			t.Fatal(err)
		}
	}

	runs, err := repos.Runs.ListByWebConsoleSessionIDs(ctx, ids)
	if err != nil {
		t.Fatal(err)
	}
	var found []string
	for _, run := range runs {
		found = append(found, run.WebConsoleSessionID)
	}
	slices.Sort(found)
	if want := []string{ids[0], ids[webConsoleSessionLookupChunk+2]}; !slices.Equal(found, want) {
		t.Fatalf("sessions found = %v, want runs from both lookup chunks %v", found, want)
	}
}

func TestFindImportProvenancePreservesAmbiguityAndHarnessIdentity(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repos := NewRepositories(db, db.log)
	ctx := context.Background()
	task := &domain.Task{ID: uuid.New(), Title: "import lookup", ScopePath: "src/", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	create := func(harness, session string) uuid.UUID {
		run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Tag: uuid.NewString(), Status: domain.RunStatusComplete, Phase: domain.RunPhaseCompleted, ExecutionMode: domain.ExecutionModeImported, ImportSourceHarness: harness, ImportSourceSessionID: session}
		if err := repos.Runs.Create(ctx, run); err != nil {
			t.Fatal(err)
		}
		return run.ID
	}
	first := create("codex", "same")
	create("claude-code", "same")
	got, err := repos.Runs.FindImportProvenance(ctx, "resource:codex/sessions", "same")
	if err != nil || len(got) != 1 || got[0] != first {
		t.Fatalf("single match = %v, %v", got, err)
	}
	second := create("resource:codex/sessions", "same")
	got, err = repos.Runs.FindImportProvenance(ctx, "codex", "same")
	if err != nil || len(got) != 2 || !slices.Contains(got, first) || !slices.Contains(got, second) {
		t.Fatalf("ambiguous matches = %v, %v", got, err)
	}
	create("resource:codex/alternate", "same")
	got, err = repos.Runs.FindImportProvenance(ctx, "codex", "same")
	if err != nil || len(got) != 2 {
		t.Fatalf("ambiguous matches must be bounded: %v, %v", got, err)
	}
	got, err = repos.Runs.FindImportProvenance(ctx, "codex", "missing")
	if err != nil || len(got) != 0 {
		t.Fatalf("missing match = %v, %v", got, err)
	}
	if _, err = repos.Runs.FindImportProvenance(ctx, "codex", ""); err == nil {
		t.Fatal("empty identity accepted")
	}
}
