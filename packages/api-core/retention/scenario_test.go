package retention

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"
)

const autohealManifest = `{
  "retention": {
    "budgets": {
      "system_events": {
        "target": {
          "kind": "sqlite_table",
          "database": "autoheal.sqlite",
          "table": "system_events",
          "time_column": "occurred_at"
        },
        "max_age": "30d",
        "max_bytes": "2GiB",
        "pruner": "builtin",
        "rationale": "Host event ingest driven by conditions Vrooli does not control."
      }
    }
  }
}`

// isolate points the variant-aware namespace resolution at an explicit
// scenario/variant pair for the duration of a test.
func isolate(t *testing.T, namespace, variant string) {
	t.Helper()
	t.Setenv(storage.EnvStorageNamespace, namespace)
	t.Setenv(storage.EnvVariant, variant)
	t.Setenv(storage.EnvScenario, "vrooli-autoheal")
}

func TestNewForScenarioWithNoRetentionBlock(t *testing.T) {
	// A component that declares nothing must keep working unchanged, and calling
	// Start/Stop on it must be safe.
	isolate(t, "vrooli-autoheal", "live")
	m, err := NewForScenario(ScenarioConfig{
		Manifest:     []byte(`{"service":{"name":"x"}}`),
		RootOverride: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("NewForScenario: %v", err)
	}
	if len(m.Engine().Budgets()) != 0 {
		t.Fatalf("got %d budgets, want 0", len(m.Engine().Budgets()))
	}
	results, err := m.Engine().Run(context.Background())
	if err != nil || len(results) != 0 {
		t.Fatalf("Run = %v, %v; want no results and no error", results, err)
	}
	m.Start(context.Background())
	m.Stop()
}

func TestNewForScenarioMissingManifestIsNotAnError(t *testing.T) {
	isolate(t, "vrooli-autoheal", "live")
	m, err := NewForScenario(ScenarioConfig{
		StartDir:     t.TempDir(),
		RootOverride: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("NewForScenario: %v", err)
	}
	if len(m.Engine().Budgets()) != 0 {
		t.Fatal("a component with no manifest must have no budgets")
	}
}

func TestNewForScenarioDiscoversTheManifestByWalkingUp(t *testing.T) {
	isolate(t, "vrooli-autoheal", "live")
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "service.json"), []byte(autohealManifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	deep := filepath.Join(root, "api", "internal", "systemevents")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatalf("mkdir deep: %v", err)
	}

	m, err := NewForScenario(ScenarioConfig{
		StartDir:     deep,
		RootOverride: t.TempDir(),
		OpenDatabase: openTestDatabase,
	})
	if err != nil {
		t.Fatalf("NewForScenario: %v", err)
	}
	if len(m.Engine().Budgets()) != 1 {
		t.Fatalf("got %d budgets, want the one declared in the discovered manifest", len(m.Engine().Budgets()))
	}
}

// openTestDatabase opens (creating if needed) a SQLite database with the
// system_events shape the autoheal manifest declares.
func openTestDatabase(path string) (Execer, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS system_events (
		id INTEGER PRIMARY KEY, occurred_at TEXT NOT NULL, payload TEXT NOT NULL)`); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func TestBuiltinBudgetNeedsNoRegisteredPrunerAndNoScenarioCode(t *testing.T) {
	// This is the property that makes adoption realistic across 125 components:
	// the manifest above is the entire integration.
	isolate(t, "vrooli-autoheal", "live")
	root := t.TempDir()
	m, err := NewForScenario(ScenarioConfig{
		Manifest:     []byte(autohealManifest),
		RootOverride: root,
		OpenDatabase: openTestDatabase,
		Now:          func() time.Time { return fixtureClock },
	})
	if err != nil {
		t.Fatalf("NewForScenario: %v", err)
	}
	if len(m.Engine().Budgets()) != 1 {
		t.Fatalf("got %d budgets, want 1", len(m.Engine().Budgets()))
	}

	dbPath, ok := m.ResolvedPath("system_events")
	if !ok {
		t.Fatal("the budget resolved to no path")
	}
	db, err := openTestDatabase(dbPath)
	if err != nil {
		t.Fatalf("open resolved database: %v", err)
	}
	// Rows well past the 30-day horizon must be gone after one cycle, with no
	// Go code in the component beyond this constructor.
	for i := range 20 {
		at := fixtureClock.Add(-time.Duration(40+i) * 24 * time.Hour)
		if _, err := db.ExecContext(context.Background(),
			`INSERT INTO system_events (occurred_at, payload) VALUES (?, ?)`, at.Format(time.RFC3339Nano), "p"); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}
	if _, err := m.Engine().Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}

	raw, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer raw.Close()
	var remaining int64
	if err := raw.QueryRow(`SELECT COUNT(*) FROM system_events`).Scan(&remaining); err != nil {
		t.Fatalf("count: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("%d rows survived a 30d horizon at 40+ days old", remaining)
	}
}

func TestShadowVariantResolvesUnderItsOwnNamespace(t *testing.T) {
	root := t.TempDir()

	isolate(t, "vrooli-autoheal", "live")
	live, err := NewForScenario(ScenarioConfig{Manifest: []byte(autohealManifest), RootOverride: root, OpenDatabase: openTestDatabase})
	if err != nil {
		t.Fatalf("live NewForScenario: %v", err)
	}
	livePath, _ := live.ResolvedPath("system_events")

	isolate(t, "vrooli-autoheal_shadow", "shadow")
	shadow, err := NewForScenario(ScenarioConfig{Manifest: []byte(autohealManifest), RootOverride: root, OpenDatabase: openTestDatabase})
	if err != nil {
		t.Fatalf("shadow NewForScenario: %v", err)
	}
	shadowPath, _ := shadow.ResolvedPath("system_events")

	if livePath == shadowPath {
		t.Fatalf("live and shadow both resolved to %q; a shadow would prune live's data", livePath)
	}
	if !strings.Contains(shadowPath, "vrooli-autoheal_shadow") {
		t.Fatalf("shadow resolved to %q, want a path under the shadow namespace", shadowPath)
	}
	if strings.Contains(filepath.Dir(livePath), "_shadow") {
		t.Fatalf("live resolved under a shadow namespace: %q", livePath)
	}
}

func TestBoundBytesCycleEmitsAFindingNamingTheBudget(t *testing.T) {
	isolate(t, "vrooli-autoheal", "live")
	var mu sync.Mutex
	var findings []Finding

	spec := specFor("system_events", Budget{MaxAge: 30 * 24 * time.Hour, MaxBytes: 2 << 30}, PrunerBuiltin)
	spec.Rationale = "Host event ingest driven by conditions Vrooli does not control."

	m := &Manager{scenario: "vrooli-autoheal", paths: map[string]string{"system_events": "/data/autoheal.sqlite"}, log: discardLogger()}
	observe := m.observe(func(f Finding) {
		mu.Lock()
		findings = append(findings, f)
		mu.Unlock()
	})

	// A cycle that pruned hard and is still bound by its ceiling.
	observe(Result{
		Budget: "system_events", Deleted: 843_000_000, BoundBy: BoundBytes,
		After: Usage{Bytes: 2 << 30, Items: 3_000_000},
	}, spec)
	// A cycle bound by age must NOT raise one; a finding on every cycle is a
	// finding nobody reads.
	observe(Result{Budget: "system_events", Deleted: 10, BoundBy: BoundAge}, spec)
	observe(Result{Budget: "system_events", BoundBy: BoundNone}, spec)

	mu.Lock()
	defer mu.Unlock()
	if len(findings) != 1 {
		t.Fatalf("got %d findings, want exactly 1 (only the BoundBytes cycle)", len(findings))
	}
	f := findings[0]
	if f.Budget != "system_events" || f.Scenario != "vrooli-autoheal" {
		t.Errorf("finding = %+v, want it to name the budget and the scenario", f)
	}
	if f.MaxBytes != 2<<30 || f.UsedBytes != 2<<30 || f.Deleted != 843_000_000 {
		t.Errorf("finding = %+v, want the declared ceiling and the measured usage", f)
	}
	if f.Rationale == "" || f.Target == "" {
		t.Errorf("finding = %+v, want the rationale and target so it names the producer", f)
	}
	if !strings.Contains(f.String(), "system_events") || !strings.Contains(f.String(), "2GiB") {
		t.Errorf("finding string %q does not name the budget and its ceiling", f.String())
	}
}

func TestManagerLogsSkippedCompaction(t *testing.T) {
	// A skipped compaction that is not surfaced is a silent one, and the space
	// it left behind is not explained by anything else.
	recorder := &recordingHandler{}
	m := &Manager{scenario: "vrooli-autoheal", paths: map[string]string{}, log: newRecordingLogger(recorder)}
	m.observe(nil)(Result{
		Budget: "system_events", BoundBy: BoundAge,
		CompactSkipped: true, CompactSkipReason: "needs 453GiB free, only 226GiB available",
	}, specFor("system_events", Budget{MaxAge: time.Hour}, PrunerBuiltin))

	if !recorder.contains("compact_skipped") || !recorder.contains("453GiB") {
		t.Fatalf("cycle log did not surface the skipped compaction: %v", recorder.lines())
	}
}

func TestNewForScenarioFailsOnUnregisteredCustomPruner(t *testing.T) {
	isolate(t, "architecture-cartographer", "live")
	manifest := `{"retention":{"budgets":{"graph_snapshots":{
	  "target":{"kind":"directory","path":"snapshots"},"max_bytes":"5GiB","pruner":"custom"}}}}`
	_, err := NewForScenario(ScenarioConfig{Manifest: []byte(manifest), RootOverride: t.TempDir(), Registry: NewRegistry()})
	if !errors.Is(err, ErrPrunerNotRegistered) {
		t.Fatalf("error = %v, want ErrPrunerNotRegistered", err)
	}
}

func TestNewForScenarioWiresARegisteredCustomPruner(t *testing.T) {
	isolate(t, "architecture-cartographer", "live")
	manifest := `{"retention":{"budgets":{"graph_snapshots":{
	  "target":{"kind":"directory","path":"snapshots"},"max_bytes":"5GiB","pruner":"custom"}}}}`

	pruner := &fakePruner{before: Usage{Bytes: 10 << 30}, after: Usage{Bytes: 1 << 30}, result: Result{Deleted: 2, BoundBy: BoundAge, After: Usage{Bytes: 1 << 30}}}
	registry := NewRegistry()
	if err := registry.Register("graph_snapshots", pruner); err != nil {
		t.Fatalf("Register: %v", err)
	}

	m, err := NewForScenario(ScenarioConfig{Manifest: []byte(manifest), RootOverride: t.TempDir(), Registry: registry})
	if err != nil {
		t.Fatalf("NewForScenario: %v", err)
	}
	if _, err := m.Engine().Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if pruner.gotBudget.MaxBytes != 5<<30 {
		t.Fatalf("custom pruner got %+v, want the manifest budget", pruner.gotBudget)
	}
}

func TestManagerStartAndStopAreIdempotent(t *testing.T) {
	isolate(t, "vrooli-autoheal", "live")
	m, err := NewForScenario(ScenarioConfig{
		Manifest:     []byte(autohealManifest),
		RootOverride: t.TempDir(),
		OpenDatabase: openTestDatabase,
		Clock:        newFakeClock(fixtureClock),
		Interval:     time.Hour,
		Logger:       newRecordingLogger(&recordingHandler{}),
	})
	if err != nil {
		t.Fatalf("NewForScenario: %v", err)
	}
	ctx := context.Background()
	m.Start(ctx)
	m.Start(ctx)
	m.Stop()
	m.Stop()
}

func TestManagerWritesDurableEnforcementReceipt(t *testing.T) {
	now := time.Date(2026, 8, 2, 23, 0, 0, 0, time.UTC)
	receiptPath := filepath.Join(t.TempDir(), "retention", "enforcement-receipt.json")
	m := &Manager{
		scenario:    "demo",
		receiptPath: receiptPath,
		now:         func() time.Time { return now },
		log:         discardLogger(),
	}

	m.recordEnforcementReceipt(nil)
	var receipt EnforcementReceipt
	data, err := os.ReadFile(receiptPath)
	if err != nil {
		t.Fatalf("read successful receipt: %v", err)
	}
	if err := json.Unmarshal(data, &receipt); err != nil {
		t.Fatalf("decode successful receipt: %v", err)
	}
	if receipt.Owner != "demo" || !receipt.LastCycleTime.Equal(now) || receipt.LastEnforcementTime == nil || !receipt.LastEnforcementTime.Equal(now) {
		t.Fatalf("successful receipt = %+v, want owner and both timestamps at %s", receipt, now)
	}

	m.recordEnforcementReceipt(errors.New("temporary prune failure"))
	data, err = os.ReadFile(receiptPath)
	if err != nil {
		t.Fatalf("read failed receipt: %v", err)
	}
	if err := json.Unmarshal(data, &receipt); err != nil {
		t.Fatalf("decode failed receipt: %v", err)
	}
	if receipt.LastEnforcementTime == nil || !receipt.LastEnforcementTime.Equal(now) || receipt.LastError != "temporary prune failure" {
		t.Fatalf("failed receipt = %+v, want the prior enforcement timestamp and the cycle error", receipt)
	}
}

func TestDiscoverOwnersAndNewForOwnerLoadNativeManifest(t *testing.T) {
	root := t.TempDir()
	manifest := filepath.Join(root, "resources", "demo", "resource.json")
	if err := os.MkdirAll(filepath.Dir(manifest), 0o755); err != nil {
		t.Fatal(err)
	}
	data := `{"name":"demo","retention":{"budgets":{"cache":{"target":{"kind":"file","class":"cache","path":"cache.bin"},"max_bytes":"1KiB"}}}}`
	if err := os.WriteFile(manifest, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	discovery, err := DiscoverOwners(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(discovery.Configs) != 1 || discovery.Configs[0].Kind != OwnerResource {
		t.Fatalf("discovery = %+v", discovery)
	}
	m, err := NewForOwner(OwnerConfig{Kind: OwnerResource, ID: "demo", ScenarioConfig: ScenarioConfig{ManifestPath: manifest, RootOverride: filepath.Join(root, "state")}})
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Budgets()) != 1 || m.ownerKind != OwnerResource || m.ownerID != "demo" {
		t.Fatalf("manager owner/budgets = %q/%q/%+v", m.ownerKind, m.ownerID, m.Budgets())
	}
}
