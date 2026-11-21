package utilizationhistory

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"agent-manager/internal/runtimeactivity"
	coredb "github.com/vrooli/api-core/database"
	_ "modernc.org/sqlite"
)

func openHistoryDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "history.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := coredb.EnsureSchemas(context.Background(), db, coredb.SchemaProviderFunc(Schema)); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestRepositoryRetainsProviderRunIdentitiesAndCoverage(t *testing.T) {
	db := openHistoryDB(t)
	repo := NewRepository(db)
	at := time.Date(2026, 11, 16, 20, 0, 0, 0, time.UTC)
	want := Sample{ObservedAt: at, Complete: true, ProviderRunIDs: map[string][]string{"openai": {"run-a", "run-b"}}, UnmappedRunIDs: []string{}, Unknown: []string{}}
	if err := repo.Record(t.Context(), want); err != nil {
		t.Fatal(err)
	}
	got, err := repo.Read(t.Context(), at.Add(-time.Second), at.Add(time.Second))
	if err != nil || len(got) != 1 || !got[0].Complete || got[0].ProviderRunIDs["openai"][1] != "run-b" {
		t.Fatalf("read samples=%+v err=%v", got, err)
	}
	if err := repo.Prune(t.Context(), at.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	got, err = repo.Read(t.Context(), at.Add(-time.Second), at.Add(time.Second))
	if err != nil || len(got) != 0 {
		t.Fatalf("prune samples=%+v err=%v", got, err)
	}
}

type fixedActivity struct{ snapshot runtimeactivity.Snapshot }

func (f fixedActivity) Read(context.Context) runtimeactivity.Snapshot { return f.snapshot }

func TestSamplerMapsOnlyKnownProviderRunIdentities(t *testing.T) {
	db := openHistoryDB(t)
	if _, err := db.Exec(`CREATE TABLE runs(id TEXT PRIMARY KEY,resolved_config TEXT,harness_kind TEXT);
	INSERT INTO runs VALUES
	('codex-run','{"runnerType":"codex"}',''),
	('claude-run','{"runnerType":"claude-code"}',''),
	('attached-codex','{}','codex'),
	('attached-claude','{}','claude-code'),
	('precedence','{"runnerType":"codex"}','claude-code'),
	('opencode-openai','{"runnerType":"opencode","model":"openai/gpt-5"}','opencode'),
	('opencode-anthropic','{"runnerType":"opencode","model":"anthropic/claude-sonnet-4"}','opencode'),
	('opencode-proxy','{"runnerType":"opencode","model":"openrouter/openai/gpt-5"}','opencode'),
	('other-run','{"runnerType":"opencode"}','opencode');`); err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 11, 16, 20, 0, 0, 0, time.UTC)
	sampler := NewSampler(fixedActivity{runtimeactivity.Snapshot{ObservedAt: at, Complete: true, PhysicalRunIDs: []string{"codex-run", "claude-run", "attached-codex", "attached-claude", "precedence", "opencode-openai", "opencode-anthropic", "opencode-proxy", "other-run"}}}, db, NewRepository(db), time.Minute)
	sampler.Now = func() time.Time { return at }
	if err := sampler.Capture(t.Context()); err != nil {
		t.Fatal(err)
	}
	got, err := NewRepository(db).Read(t.Context(), at.Add(-time.Second), at.Add(time.Second))
	if err != nil || len(got) != 1 {
		t.Fatalf("samples=%+v err=%v", got, err)
	}
	if len(got[0].ProviderRunIDs["openai"]) != 4 || len(got[0].ProviderRunIDs["anthropic"]) != 3 || got[0].ProviderRunIDs["openai"][0] != "attached-codex" || got[0].ProviderRunIDs["anthropic"][0] != "attached-claude" || len(got[0].UnmappedRunIDs) != 2 || got[0].UnmappedRunIDs[0] != "opencode-proxy" || got[0].UnmappedRunIDs[1] != "other-run" || !got[0].Complete {
		t.Fatalf("provider attribution or unknown runner mapping incorrect: %+v", got[0])
	}
}

func TestSamplerMarksMissingRunIdentityAsIncomplete(t *testing.T) {
	db := openHistoryDB(t)
	if _, err := db.Exec(`CREATE TABLE runs(id TEXT PRIMARY KEY,resolved_config TEXT,harness_kind TEXT);`); err != nil {
		t.Fatal(err)
	}
	at := time.Now().UTC()
	sampler := NewSampler(fixedActivity{runtimeactivity.Snapshot{ObservedAt: at, Complete: true, PhysicalRunIDs: []string{"deleted-run"}}}, db, NewRepository(db), time.Minute)
	if err := sampler.Capture(t.Context()); err != nil {
		t.Fatal(err)
	}
	got, err := NewRepository(db).Read(t.Context(), at.Add(-time.Second), at.Add(time.Second))
	if err != nil || len(got) != 1 || got[0].Complete || len(got[0].Unknown) == 0 {
		t.Fatalf("missing run mapping did not preserve unknown: %+v err=%v", got, err)
	}
}
