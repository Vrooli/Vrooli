package experimentarms

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	aisearch "github.com/vrooli/ai-go/search"
)

func TestSQLFleetGenerationCatalogCASAndRestart(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "qdrant-fleet.sqlite")
	db, err := sql.Open("sqlite", sqliteDSN(path))
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := aisearch.NewSQLiteFleetGenerationCatalog(db)
	if err != nil {
		t.Fatal(err)
	}
	if err := catalog.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	operation := aisearch.FleetAliasOperation{OwnerID: "owner-a", EndpointID: "http://qdrant.test", Alias: "docs", OperationID: "promote:arm-a:collection-a", Kind: aisearch.FleetPromote, ArmID: "arm-a", Target: "collection-a", StartedAt: time.Now().UTC()}
	if err := catalog.AcquireFleetAliasOperation(ctx, operation); err != nil {
		t.Fatal(err)
	}
	record := aisearch.FleetGeneration{OwnerID: "owner-a", Alias: "docs", ArmID: "arm-a", RecipeID: "sha256:recipe", SnapshotID: "snapshot-a", SnapshotWatermark: "sequence:4", Dimensions: 768, IndexSpecDigest: "sha256:index", Collection: "docs--fleet--arm-a--digest", State: aisearch.FleetBuilding, Version: 1}
	if err := catalog.SaveFleetGenerationCAS(ctx, record, 0); err != nil {
		t.Fatal(err)
	}
	if err := catalog.SaveFleetGenerationCAS(ctx, record, 0); !errors.Is(err, aisearch.ErrFleetStateConflict) {
		t.Fatalf("stale CAS error=%v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = sql.Open("sqlite", sqliteDSN(path))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	reopened, err := aisearch.NewSQLiteFleetGenerationCatalog(db)
	if err != nil {
		t.Fatal(err)
	}
	if err := reopened.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	if err := reopened.AcquireFleetAliasOperation(ctx, operation); err != nil {
		t.Fatalf("same durable operation could not resume after reopen: %v", err)
	}
	conflicting := operation
	conflicting.OwnerID = "owner-b"
	conflicting.OperationID = "promote:arm-b:collection-b"
	conflicting.ArmID = "arm-b"
	conflicting.Target = "collection-b"
	if err := reopened.AcquireFleetAliasOperation(ctx, conflicting); !errors.Is(err, aisearch.ErrFleetStateConflict) {
		t.Fatalf("concurrent alias operation was not fenced: %v", err)
	}
	got, ok, err := reopened.LoadFleetGeneration(ctx, record.ArmID)
	if err != nil || !ok {
		t.Fatalf("reopen read: ok=%v err=%v", ok, err)
	}
	if got != record {
		t.Fatalf("reopened record mismatch: got=%+v want=%+v", got, record)
	}
	record.State = aisearch.FleetCandidate
	record.Version++
	record.Cursor = "page:next"
	if err := reopened.SaveFleetGenerationCAS(ctx, record, 1); err != nil {
		t.Fatal(err)
	}
	byCollection, ok, err := reopened.LoadFleetGenerationByCollection(ctx, record.Collection)
	if err != nil || !ok || byCollection != record {
		t.Fatalf("collection lookup: record=%+v ok=%v err=%v", byCollection, ok, err)
	}
	listed, err := reopened.ListFleetGenerations(ctx, record.OwnerID, record.Alias)
	if err != nil || len(listed) != 1 || listed[0] != record {
		t.Fatalf("owner listing: records=%+v err=%v", listed, err)
	}
	receipt := aisearch.FleetAdmissionReceipt{OperationID: "stage-op-1", ReceiptID: "capacity-receipt-1", Operation: aisearch.FleetStage, ArmID: record.ArmID, TargetID: record.Collection, PeakBytes: 4096, ObservedAt: time.Now().UTC(), ExpiresAt: time.Now().Add(time.Minute).UTC()}
	if err := reopened.RecordFleetAdmissionReceipt(ctx, receipt); err != nil {
		t.Fatal(err)
	}
	if err := reopened.RecordFleetAdmissionReceipt(ctx, receipt); err != nil {
		t.Fatalf("idempotent admission receipt: %v", err)
	}
	receipts, err := reopened.ListFleetAdmissionReceipts(ctx, record.ArmID)
	if err != nil || len(receipts) != 1 || receipts[0] != receipt {
		t.Fatalf("admission receipt persistence: receipts=%+v err=%v", receipts, err)
	}
	if err := reopened.ReleaseFleetAliasOperation(ctx, operation); err != nil {
		t.Fatal(err)
	}
	if err := reopened.AcquireFleetAliasOperation(ctx, conflicting); err != nil {
		t.Fatalf("next alias operation remained blocked after successful release: %v", err)
	}
}
