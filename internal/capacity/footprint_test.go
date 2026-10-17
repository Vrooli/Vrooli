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
	if len(rows) != 1 || rows[0].PeakBytes != 11<<30 || rows[0].Source != FootprintSourceMeasured {
		t.Fatalf("num_parallel=2 rows = %+v, want monotonic 11 GiB measured row", rows)
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
