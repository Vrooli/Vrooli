package capacity

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/testenv"
)

func TestFootprintIsMonotonicAndIsolatedByTunablesKey(t *testing.T) {
	ctx := context.Background()
	store := testenv.NewSQLiteStore(t, "capacity.db", func(path string) (*SQLiteStore, error) {
		return NewSQLiteStore(ctx, Config{DBPath: path})
	})
	at := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	keyTwo := "num_parallel=2"
	keyFour := "num_parallel=4"
	for _, observation := range []FootprintObservation{
		{Resource: "ollama", Rung: "qwen3.5:9b", TunablesKey: keyTwo, Bytes: 11 << 30, Source: FootprintSourceManifestDefault, ObservedAt: at},
		{Resource: "ollama", Rung: "qwen3.5:9b", TunablesKey: keyTwo, Bytes: 10 << 30, Source: FootprintSourceMeasured, ObservedAt: at.Add(time.Hour)},
		{Resource: "ollama", Rung: "qwen3.5:9b", TunablesKey: keyFour, Bytes: 14 << 30, Source: FootprintSourceMeasured, ObservedAt: at.Add(2 * time.Hour)},
	} {
		if _, err := store.RecordFootprint(ctx, observation); err != nil {
			t.Fatal(err)
		}
	}
	rows, err := store.ListFootprints(ctx, FootprintFilter{Resource: "ollama", TunablesKey: &keyTwo})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].PeakBytes != 11<<30 || rows[0].Source != FootprintSourceManifestDefault {
		t.Fatalf("num_parallel=2 rows = %+v, want monotonic 11 GiB declared estimate; smaller sample does not measure that peak", rows)
	}
	if rows[0].LastSeenAt.Sub(rows[0].FirstSeenAt) != time.Hour {
		t.Fatalf("footprint did not outlive sampling half-life: %+v", rows[0])
	}
}

func TestClaimGCDoesNotDeleteFootprints(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "capacity.db")
	store, err := NewSQLiteStore(ctx, Config{DBPath: dbPath})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	at := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	if _, err := store.RecordFootprint(ctx, FootprintObservation{Resource: "ollama", Rung: "small", Bytes: 4 << 30, Source: FootprintSourceMeasured, ObservedAt: at}); err != nil {
		t.Fatal(err)
	}
	claim, err := store.CreateClaim(ctx, sampleClaim(), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.ReleaseClaim(ctx, claim.ClaimID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GCTerminalClaims(ctx, time.Now().UTC().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	rows, err := store.ListFootprints(ctx, FootprintFilter{})
	if err != nil || len(rows) != 1 {
		t.Fatalf("footprints after claim GC = %+v, err=%v; want one durable row", rows, err)
	}
}

func TestResetFootprintsCanTargetOneResource(t *testing.T) {
	ctx := context.Background()
	store := testenv.NewSQLiteStore(t, "capacity.db", func(path string) (*SQLiteStore, error) {
		return NewSQLiteStore(ctx, Config{DBPath: path})
	})
	for _, resource := range []string{"ollama", "reranker"} {
		if _, err := store.RecordFootprint(ctx, FootprintObservation{Resource: resource, Rung: "gpu", Bytes: 1, Source: FootprintSourceMeasured, ObservedAt: time.Now()}); err != nil {
			t.Fatal(err)
		}
	}
	reset, err := store.ResetFootprints(ctx, "ollama")
	if err != nil || reset != 1 {
		t.Fatalf("ResetFootprints() = %d, %v; want 1, nil", reset, err)
	}
	rows, err := store.ListFootprints(ctx, FootprintFilter{})
	if err != nil || len(rows) != 1 || rows[0].Resource != "reranker" {
		t.Fatalf("remaining footprints = %+v, %v; want reranker only", rows, err)
	}
}

func TestFootprintSourceDescribesWinningPeak(t *testing.T) {
	ctx := context.Background()
	store := testenv.NewSQLiteStore(t, "capacity.db", func(path string) (*SQLiteStore, error) { return NewSQLiteStore(ctx, Config{DBPath: path}) })
	at := time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)
	record := func(bytes int64, source string) Footprint {
		t.Helper()
		f, err := store.RecordFootprint(ctx, FootprintObservation{Resource: "model", Rung: "gpu", Bytes: bytes, Source: source, ObservedAt: at})
		if err != nil {
			t.Fatal(err)
		}
		at = at.Add(time.Minute)
		return f
	}
	seeded := record(12*gib, FootprintSourceManifestDefault)
	if seeded.Samples != 0 {
		t.Fatalf("manifest seed counted as measurement: %+v", seeded)
	}
	small := record(4*gib, FootprintSourceMeasured)
	if small.PeakBytes != 12*gib || small.Source != FootprintSourceManifestDefault || small.Samples != 1 {
		t.Fatalf("small sample relabeled unmeasured peak: %+v", small)
	}
	tie := record(12*gib, FootprintSourceMeasured)
	if tie.Source != FootprintSourceMeasured || tie.Samples != 2 {
		t.Fatalf("measured peak not recognized: %+v", tie)
	}
	laterSeed := record(11*gib, FootprintSourceManifestDefault)
	if laterSeed.Source != FootprintSourceMeasured || laterSeed.Samples != 2 {
		t.Fatalf("losing seed changed measured evidence: %+v", laterSeed)
	}
}
