package capacity

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/testenv"
)

func placementReceiptFixture(t *testing.T, sampled time.Time) PlacementQualification {
	t.Helper()
	artifact := filepath.Join(t.TempDir(), "suite-result.json")
	body := []byte(`{"cases":24,"passed":true,"p95_ms":120,"device":"cuda","vram_bytes":1073741824}`)
	if err := os.WriteFile(artifact, body, 0o600); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(body)
	return PlacementQualification{
		ClaimID: "claim-1", ClaimGeneration: 3, Role: "rerank.default",
		Model: ModelIdentity{Name: "bge-reranker", Digest: "sha256:model"},
		Rung:  "compact", Device: ResidencyBackendCUDA,
		Suite: "search-hub-rerank", SuiteDigest: hex.EncodeToString(digest[:]),
		Cases: 24, Passed: true, P95LatencyMS: 120, MaxLatencyMS: 200,
		ObservedVRAM: gib, SampledAt: sampled, RunID: "test-genie-run-123",
		ArtifactURI: artifact, ArtifactSHA: hex.EncodeToString(digest[:]),
	}
}

func createPlacementClaim(t *testing.T, ctx context.Context, store *SQLiteStore, q PlacementQualification) CapacityClaim {
	t.Helper()
	claim, _ := qualifiedTestClaim(q.ClaimID, q.Role, q.SampledAt.Add(time.Minute))
	claim.DegradeProfile.Model = q.Model
	claim.Generation = q.ClaimGeneration
	created, err := store.CreateClaim(ctx, claim, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	return created
}

func TestPlacementQualificationPersistsContentAddressedArtifactLink(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 16, 3, 7, 0, 0, time.UTC)
	clk := testenv.NewClock(now)
	dbPath := filepath.Join(t.TempDir(), "capacity.db")
	store, err := NewSQLiteStore(ctx, Config{DBPath: dbPath, Clock: clk})
	if err != nil {
		t.Fatal(err)
	}
	q := placementReceiptFixture(t, now.Add(-time.Minute))
	createPlacementClaim(t, ctx, store, q)
	first, err := store.RecordPlacementQualification(ctx, q)
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.RecordPlacementQualification(ctx, q)
	if err != nil {
		t.Fatal(err)
	}
	if first.ReceiptID == "" || first.ReceiptID != second.ReceiptID {
		t.Fatalf("idempotent receipt identity mismatch: first=%+v second=%+v", first, second)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := NewSQLiteStore(ctx, Config{DBPath: dbPath, Clock: clk})
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	rows, err := reopened.ListPlacementQualifications(ctx, PlacementQualificationFilter{Role: "rerank.default", ModelDigest: q.Model.Digest, Rung: "compact"})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].ReceiptID != first.ReceiptID || rows[0].ArtifactSHA != q.ArtifactSHA {
		t.Fatalf("persisted, filtered artifact receipt missing or changed: %+v", rows)
	}
	claim, err := reopened.GetClaim(ctx, q.ClaimID)
	if err != nil {
		t.Fatal(err)
	}
	table, err := ReadPlacementTable(ctx, reopened, []CapacityClaim{claim}, now, 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if len(table) != 2 || table[0].EvidenceStatus != PlacementEvidenceQualified || table[0].CurrentClaimRung || table[1].EvidenceStatus != PlacementEvidenceMissing {
		t.Fatalf("table should show qualified compact evidence separately from unqualified current rung: %+v", table)
	}
}

func TestQualifiedPlannerConsumesPersistedReceipts(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 16, 3, 7, 0, 0, time.UTC)
	store, err := NewSQLiteStore(ctx, Config{DBPath: filepath.Join(t.TempDir(), "capacity.db"), Clock: testenv.NewClock(now)})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	q := placementReceiptFixture(t, now.Add(-time.Minute))
	q.ClaimID = "a"
	claim := createPlacementClaim(t, ctx, store, q)
	if _, err := store.RecordPlacementQualification(ctx, q); err != nil {
		t.Fatal(err)
	}
	plan, err := PlanQualifiedDownsizeFromStore(ctx, QualifiedPlacementRequest{
		RequesterPriority: PriorityInteractive, DeficitBytes: 2 * gib,
		Claims: []CapacityClaim{claim}, Policy: DefaultPolicy(), Now: now,
	}, store)
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Satisfied || len(plan.Actions) != 1 || plan.Actions[0].ClaimID != "a" || plan.Actions[0].Generation != 3 || plan.Actions[0].ToStep != "compact" {
		t.Fatalf("planner should consume the exact persisted suite receipt: %+v", plan)
	}
	if err := os.WriteFile(q.ArtifactURI, []byte("changed"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := PlanQualifiedDownsizeFromStore(ctx, QualifiedPlacementRequest{Claims: []CapacityClaim{claim}, Now: now}, store); err == nil {
		t.Fatal("planner must fail closed when a linked immutable run artifact changes")
	}
}

func TestPlacementQualificationRejectsBadArtifactAndWeakMeasurement(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 16, 3, 7, 0, 0, time.UTC)
	store, err := NewSQLiteStore(ctx, Config{DBPath: filepath.Join(t.TempDir(), "capacity.db"), Clock: testenv.NewClock(now)})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	tests := []struct {
		name   string
		mutate func(*PlacementQualification)
	}{
		{"artifact digest mismatch", func(q *PlacementQualification) { q.ArtifactSHA = hex.EncodeToString(make([]byte, sha256.Size)) }},
		{"no immutable run identity", func(q *PlacementQualification) { q.RunID = "" }},
		{"undersized suite", func(q *PlacementQualification) { q.Cases = 19 }},
		{"future sample", func(q *PlacementQualification) { q.SampledAt = now.Add(time.Second) }},
		{"CPU with VRAM", func(q *PlacementQualification) { q.Device = ResidencyBackendCPU }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			q := placementReceiptFixture(t, now.Add(-time.Minute))
			tc.mutate(&q)
			if _, err := store.RecordPlacementQualification(ctx, q); err == nil {
				t.Fatal("invalid qualification should fail closed")
			}
		})
	}
}

func TestPlacementQualificationRequiresCurrentClaimGenerationRoleAndRung(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 11, 16, 3, 7, 0, 0, time.UTC)
	store, err := NewSQLiteStore(ctx, Config{DBPath: filepath.Join(t.TempDir(), "capacity.db"), Clock: testenv.NewClock(now)})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	base := placementReceiptFixture(t, now.Add(-time.Minute))
	createPlacementClaim(t, ctx, store, base)
	tests := []struct {
		name   string
		mutate func(*PlacementQualification)
	}{
		{"stale generation", func(q *PlacementQualification) { q.ClaimGeneration++ }},
		{"different role", func(q *PlacementQualification) { q.Role = "chat.default" }},
		{"undeclared rung", func(q *PlacementQualification) { q.Rung = "unlisted" }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			q := base
			tc.mutate(&q)
			if _, err := store.RecordPlacementQualification(ctx, q); err == nil {
				t.Fatal("receipt must be rejected unless it matches the live claim incarnation and profile")
			}
		})
	}
}

func TestPlacementQualificationSchemaAddsExactGenerationFailClosed(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "capacity.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE capacity_placement_qualifications (
receipt_id TEXT PRIMARY KEY, claim_id TEXT NOT NULL, role TEXT NOT NULL,
model_digest TEXT NOT NULL, rung TEXT NOT NULL, device TEXT NOT NULL,
suite_digest TEXT NOT NULL, sampled_at TEXT NOT NULL,
qualification_json TEXT NOT NULL, recorded_at TEXT NOT NULL);
PRAGMA user_version = 4;`)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	store, err := NewSQLiteStore(ctx, Config{DBPath: dbPath, Clock: testenv.NewClock(time.Now().UTC())})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	rows, err := store.db.QueryContext(ctx, `PRAGMA table_info(capacity_placement_qualifications)`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	hasGeneration := false
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, columnType string
		var defaultValue any
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			t.Fatal(err)
		}
		if name == "claim_generation" {
			hasGeneration = true
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if !hasGeneration {
		t.Fatal("current store schema must add generation; legacy receipts stay unusable at generation zero")
	}
}
