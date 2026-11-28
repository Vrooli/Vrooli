package experimentarms

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	aisearch "github.com/vrooli/ai-go/search"
)

type fleetIntegrationSource struct{ watermark string }

func (s *fleetIntegrationSource) PinFleetSnapshot(context.Context, string) (aisearch.FleetSnapshot, error) {
	return aisearch.FleetSnapshot{}, errors.New("not used by this cutover integration test")
}
func (s *fleetIntegrationSource) ReadFleetSnapshotPage(context.Context, string, aisearch.FleetSnapshot, string, int) ([]aisearch.FleetPoint, string, bool, error) {
	return nil, "", false, errors.New("not used by this cutover integration test")
}
func (s *fleetIntegrationSource) ReadFleetChanges(_ context.Context, _, _, after string, _ int) ([]aisearch.FleetChange, string, error) {
	return nil, after, nil
}
func (s *fleetIntegrationSource) CompareFleetSequence(_ context.Context, before, after string) (int, error) {
	return strings.Compare(after, before), nil
}
func (s *fleetIntegrationSource) FleetWatermark(context.Context) (string, error) {
	return s.watermark, nil
}
func (s *fleetIntegrationSource) ExpectedFleetPointCount(context.Context, string, string, string) (uint64, error) {
	return 1, nil
}
func (s *fleetIntegrationSource) AcquireFleetCutover(_ context.Context, expected string) (aisearch.FleetCutoverFence, error) {
	if expected != s.watermark {
		return nil, errors.New("watermark mismatch")
	}
	return fleetIntegrationFence{s: s}, nil
}

type fleetIntegrationFence struct{ s *fleetIntegrationSource }

func (f fleetIntegrationFence) CurrentWatermark(context.Context) (string, error) {
	return f.s.watermark, nil
}
func (fleetIntegrationFence) Release(context.Context) error { return nil }

type fleetIntegrationAliasBoundary struct {
	mu        sync.Mutex
	byAlias   map[string]aisearch.FleetAliasOperation
	onRelease func(aisearch.FleetAliasOperation) error
}
type fleetIntegrationAliasFence struct {
	boundary    *fleetIntegrationAliasBoundary
	key         string
	operationID string
	operation   aisearch.FleetAliasOperation
}

func (b *fleetIntegrationAliasBoundary) AcquireFleetAliasMutation(_ context.Context, operation aisearch.FleetAliasOperation) (aisearch.FleetAliasMutationFence, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.byAlias == nil {
		b.byAlias = make(map[string]aisearch.FleetAliasOperation)
	}
	key := operation.EndpointID + "/" + operation.Alias
	if existing, ok := b.byAlias[key]; ok {
		if existing.OperationID != operation.OperationID || existing.OwnerID != operation.OwnerID || existing.Target != operation.Target {
			return nil, aisearch.ErrFleetStateConflict
		}
	} else {
		b.byAlias[key] = operation
	}
	return &fleetIntegrationAliasFence{boundary: b, key: key, operationID: operation.OperationID, operation: operation}, nil
}
func (f *fleetIntegrationAliasFence) Release(context.Context) error {
	f.boundary.mu.Lock()
	current, ok := f.boundary.byAlias[f.key]
	if !ok || current.OperationID != f.operationID {
		f.boundary.mu.Unlock()
		return aisearch.ErrFleetStateConflict
	}
	onRelease := f.boundary.onRelease
	if onRelease != nil {
		if err := onRelease(f.operation); err != nil {
			f.boundary.mu.Unlock()
			return err
		}
	}
	delete(f.boundary.byAlias, f.key)
	f.boundary.mu.Unlock()
	return nil
}

type fleetIntegrationGate struct {
	mu      sync.Mutex
	entered chan struct{}
	resume  <-chan struct{}
	once    sync.Once
}

func (g *fleetIntegrationGate) AdmitQdrantFleetOperation(_ context.Context, _ aisearch.FleetOperation, _ string, target string, peak int64) (aisearch.FleetAdmission, error) {
	g.mu.Lock()
	entered, resume := g.entered, g.resume
	g.mu.Unlock()
	if resume != nil {
		g.once.Do(func() { close(entered) })
		<-resume
	}
	now := time.Now().UTC()
	return aisearch.FleetAdmission{OperationID: fmt.Sprintf("integration-%d", now.UnixNano()), ReceiptID: "fresh-integration-receipt", TargetID: target, PeakBytes: peak, ObservedAt: now, ExpiresAt: now.Add(time.Minute)}, nil
}

func TestFleetAdapterRestartReplayAndConcurrentAdapters(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "fleet-adapter.sqlite")
	db1, err := sql.Open("sqlite", sqliteDSN(path))
	if err != nil {
		t.Fatal(err)
	}
	db1.SetMaxOpenConns(1)
	catalog1, err := aisearch.NewSQLiteFleetGenerationCatalog(db1)
	if err != nil {
		t.Fatal(err)
	}
	if err := catalog1.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	old := integrationGeneration("arm-old", "old-collection", aisearch.FleetActive, 1, "old-recipe")
	armA := integrationGeneration("arm-a", "collection-a", aisearch.FleetCandidate, 1, "recipe-a")
	armB := integrationGeneration("arm-b", "collection-b", aisearch.FleetCandidate, 1, "recipe-b")
	for _, record := range []aisearch.FleetGeneration{old, armA, armB} {
		if err := catalog1.SaveFleetGenerationCAS(ctx, record, 0); err != nil {
			t.Fatal(err)
		}
	}

	var mu sync.Mutex
	collections := map[string]bool{"old-collection": true, "collection-a": true, "collection-b": true}
	deleteCalls := map[string]int{}
	aliases := map[string]string{"owner-a_docs": "old-collection"}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if r.URL.Path == "/collections/aliases" && r.Method == http.MethodGet {
			items := make([]map[string]string, 0, len(aliases))
			for alias, collection := range aliases {
				items = append(items, map[string]string{"alias_name": alias, "collection_name": collection})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{"aliases": items}})
			return
		}
		if r.URL.Path == "/collections/aliases" && r.Method == http.MethodPost {
			var body struct {
				Actions []map[string]map[string]string `json:"actions"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			for _, action := range body.Actions {
				if remove, ok := action["delete_alias"]; ok {
					delete(aliases, remove["alias_name"])
				}
				if add, ok := action["create_alias"]; ok {
					aliases[add["alias_name"]] = add["collection_name"]
				}
			}
			w.WriteHeader(http.StatusOK)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/collections/") {
			name := strings.TrimPrefix(r.URL.Path, "/collections/")
			if strings.HasSuffix(r.URL.Path, "/points/count") && r.Method == http.MethodPost {
				_ = json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{"count": uint64(1)}})
				return
			}
			switch r.Method {
			case http.MethodGet:
				if !collections[name] {
					http.NotFound(w, r)
					return
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{"disk_data_size": int64(256)}})
				return
			case http.MethodDelete:
				if !collections[name] {
					http.NotFound(w, r)
					return
				}
				delete(collections, name)
				deleteCalls[name]++
				w.WriteHeader(http.StatusOK)
				return
			}
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	source := &fleetIntegrationSource{watermark: "w1"}
	gate := &fleetIntegrationGate{}
	aliasBoundary := &fleetIntegrationAliasBoundary{}
	options := func(catalog aisearch.FleetGenerationCatalog, admission aisearch.FleetAdmissionGate) aisearch.QdrantFleetOptions {
		return aisearch.QdrantFleetOptions{BaseURL: server.URL, OwnerID: "owner-a", Alias: "owner-a_docs", Client: server.Client(), Admission: admission, AliasBoundary: aliasBoundary, Source: source, Catalog: catalog, RetiredMaxAge: time.Nanosecond}
	}
	adapter1, err := aisearch.NewQdrantFleetAdapter(options(catalog1, gate))
	if err != nil {
		t.Fatal(err)
	}
	db2, err := sql.Open("sqlite", sqliteDSN(path))
	if err != nil {
		t.Fatal(err)
	}
	db2.SetMaxOpenConns(1)
	defer db2.Close()
	catalog2, err := aisearch.NewSQLiteFleetGenerationCatalog(db2)
	if err != nil {
		t.Fatal(err)
	}
	if err := catalog2.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	gate.entered = make(chan struct{})
	resume := make(chan struct{})
	gate.resume = resume
	adapter2, err := aisearch.NewQdrantFleetAdapter(options(catalog2, &fleetIntegrationGate{}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db1.ExecContext(ctx, `CREATE TRIGGER fail_promote_after_alias BEFORE UPDATE ON qdrant_fleet_generations WHEN NEW.arm_id = 'arm-a' AND NEW.version = 3 BEGIN SELECT RAISE(FAIL, 'injected promote catalog interruption'); END`); err != nil {
		t.Fatal(err)
	}
	var promoteErr error
	var done sync.WaitGroup
	done.Add(1)
	go func() {
		defer done.Done()
		_, promoteErr = adapter1.Promote(ctx, "arm-a", 1024)
	}()
	<-gate.entered
	if _, err := adapter2.Promote(ctx, "arm-b", 1024); !errors.Is(err, aisearch.ErrFleetStateConflict) {
		t.Fatalf("second adapter bypassed durable alias lock: %v", err)
	}
	close(resume)
	done.Wait()
	if promoteErr == nil || !strings.Contains(promoteErr.Error(), "alias switched but generation catalog write failed") {
		t.Fatalf("expected interrupted promotion after alias mutation, got %v", promoteErr)
	}
	mu.Lock()
	if aliases["owner-a_docs"] != "collection-a" {
		t.Fatalf("promotion failpoint missed alias effect: %q", aliases["owner-a_docs"])
	}
	mu.Unlock()
	if _, err := db1.ExecContext(ctx, `DROP TRIGGER fail_promote_after_alias`); err != nil {
		t.Fatal(err)
	}
	if err := db2.Close(); err != nil {
		t.Fatal(err)
	}
	if err := db1.Close(); err != nil {
		t.Fatal(err)
	}
	db3, err := sql.Open("sqlite", sqliteDSN(path))
	if err != nil {
		t.Fatal(err)
	}
	db3.SetMaxOpenConns(1)
	catalog3, err := aisearch.NewSQLiteFleetGenerationCatalog(db3)
	if err != nil {
		t.Fatal(err)
	}
	if err := catalog3.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	restarted, err := aisearch.NewQdrantFleetAdapter(options(catalog3, &fleetIntegrationGate{}))
	if err != nil {
		t.Fatal(err)
	}
	active, err := restarted.Promote(ctx, "arm-a", 1024)
	if err != nil || active.State != aisearch.FleetActive {
		t.Fatalf("restarted promotion recovery: generation=%+v err=%v", active, err)
	}
	old, _, err = catalog3.LoadFleetGeneration(ctx, "arm-old")
	if err != nil || old.State != aisearch.FleetRetired {
		t.Fatalf("previous generation was not retired after replay: %+v err=%v", old, err)
	}
	if _, err := db3.ExecContext(ctx, `CREATE TRIGGER fail_rollback_after_alias BEFORE UPDATE ON qdrant_fleet_generations WHEN NEW.arm_id = 'arm-old' AND NEW.version = 6 BEGIN SELECT RAISE(FAIL, 'injected rollback catalog interruption'); END`); err != nil {
		t.Fatal(err)
	}
	if _, err := restarted.Rollback(ctx, "arm-a", 1024); err == nil || !strings.Contains(err.Error(), "alias rolled back but previous generation catalog update failed") {
		t.Fatalf("expected interrupted rollback after alias mutation, got %v", err)
	}
	mu.Lock()
	if aliases["owner-a_docs"] != "old-collection" {
		t.Fatalf("rollback failpoint missed alias effect: %q", aliases["owner-a_docs"])
	}
	mu.Unlock()
	if _, err := db3.ExecContext(ctx, `DROP TRIGGER fail_rollback_after_alias`); err != nil {
		t.Fatal(err)
	}
	if err := db3.Close(); err != nil {
		t.Fatal(err)
	}
	db4, err := sql.Open("sqlite", sqliteDSN(path))
	if err != nil {
		t.Fatal(err)
	}
	db4.SetMaxOpenConns(1)
	defer db4.Close()
	catalog4, err := aisearch.NewSQLiteFleetGenerationCatalog(db4)
	if err != nil {
		t.Fatal(err)
	}
	if err := catalog4.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	restarted, err = aisearch.NewQdrantFleetAdapter(options(catalog4, &fleetIntegrationGate{}))
	if err != nil {
		t.Fatal(err)
	}
	rolledBack, err := restarted.Rollback(ctx, "arm-a", 1024)
	if err != nil || rolledBack.State != aisearch.FleetRetired {
		t.Fatalf("restarted rollback recovery: generation=%+v err=%v", rolledBack, err)
	}
	old, _, err = catalog4.LoadFleetGeneration(ctx, "arm-old")
	if err != nil {
		t.Fatal(err)
	}
	old.RollbackTo = ""
	old.Version++
	if err := catalog4.SaveFleetGenerationCAS(ctx, old, old.Version-1); err != nil {
		t.Fatal(err)
	}
	armA, _, err = catalog4.LoadFleetGeneration(ctx, "arm-a")
	if err != nil {
		t.Fatal(err)
	}
	armA.RetentionExpiresAt = time.Now().Add(-time.Second).UTC()
	armA.Version++
	if err := catalog4.SaveFleetGenerationCAS(ctx, armA, armA.Version-1); err != nil {
		t.Fatal(err)
	}
	if _, err := db4.ExecContext(ctx, `CREATE TRIGGER fail_collect_after_delete BEFORE UPDATE ON qdrant_fleet_generations WHEN NEW.arm_id = 'arm-a' AND NEW.version = 8 BEGIN SELECT RAISE(FAIL, 'injected collect catalog interruption'); END`); err != nil {
		t.Fatal(err)
	}
	if err := restarted.Collect(ctx, []string{"arm-a"}, 1024); err == nil {
		t.Fatal("expected collection catalog interruption after Qdrant deletion")
	}
	mu.Lock()
	if collections["collection-a"] {
		t.Fatal("collect failpoint did not happen after physical delete")
	}
	mu.Unlock()
	if _, err := db4.ExecContext(ctx, `DROP TRIGGER fail_collect_after_delete`); err != nil {
		t.Fatal(err)
	}
	if err := db4.Close(); err != nil {
		t.Fatal(err)
	}
	db5, err := sql.Open("sqlite", sqliteDSN(path))
	if err != nil {
		t.Fatal(err)
	}
	defer db5.Close()
	catalog5, err := aisearch.NewSQLiteFleetGenerationCatalog(db5)
	if err != nil {
		t.Fatal(err)
	}
	if err := catalog5.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	aliasBoundary.onRelease = func(operation aisearch.FleetAliasOperation) error {
		if operation.Kind != aisearch.FleetCollect {
			return nil
		}
		var count int
		if err := db5.QueryRowContext(ctx, `SELECT COUNT(*) FROM qdrant_fleet_alias_operations WHERE endpoint_id = ? AND alias_name = ?`, operation.EndpointID, operation.Alias).Scan(&count); err != nil {
			return err
		}
		if count != 0 {
			return fmt.Errorf("durable collect operation row still exists when alias fence was released (count=%d)", count)
		}
		return nil
	}
	restarted, err = aisearch.NewQdrantFleetAdapter(options(catalog5, &fleetIntegrationGate{}))
	if err != nil {
		t.Fatal(err)
	}
	if err := restarted.Collect(ctx, []string{"arm-a"}, 1024); err != nil {
		t.Fatalf("restarted collect recovery: %v", err)
	}
	armA, _, err = catalog5.LoadFleetGeneration(ctx, "arm-a")
	if err != nil || armA.State != aisearch.FleetDeleted || armA.DiskReclaimedBytesKnown {
		t.Fatalf("deleted generation must retain unknown reclaim evidence: %+v err=%v", armA, err)
	}

	// Exercise the later crash window: Qdrant DELETE and the FleetDeleted CAS
	// succeeded, but releasing the active owner fence failed. The fresh adapter
	// must reconcile the absent collection and drain the exact durable rows.
	armB, ok, err := catalog5.LoadFleetGeneration(ctx, "arm-b")
	if err != nil || !ok {
		t.Fatalf("load second retired arm: found=%v err=%v", ok, err)
	}
	armB.State = aisearch.FleetRetired
	armB.RetentionExpiresAt = time.Now().Add(-time.Second).UTC()
	armB.Version++
	if err := catalog5.SaveFleetGenerationCAS(ctx, armB, armB.Version-1); err != nil {
		t.Fatal(err)
	}
	durableBoundary, err := aisearch.NewSQLiteFleetAliasMutationBoundary(db5)
	if err != nil {
		t.Fatal(err)
	}
	if err := durableBoundary.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := db5.ExecContext(ctx, `CREATE TRIGGER fail_collect_fence_release BEFORE DELETE ON qdrant_fleet_alias_fences WHEN OLD.operation_id = 'collect:arm-b:collection-b' BEGIN SELECT RAISE(FAIL, 'injected crash after deleted CAS before fence release'); END`); err != nil {
		t.Fatal(err)
	}
	replayOptions := options(catalog5, &fleetIntegrationGate{})
	replayOptions.AliasBoundary = durableBoundary
	restarted, err = aisearch.NewQdrantFleetAdapter(replayOptions)
	if err != nil {
		t.Fatal(err)
	}
	if err := restarted.Collect(ctx, []string{"arm-b"}, 1024); err == nil || !strings.Contains(err.Error(), "release owner-wide") {
		t.Fatalf("expected active fence release interruption after deleted CAS, got %v", err)
	}
	armB, _, err = catalog5.LoadFleetGeneration(ctx, "arm-b")
	if err != nil || armB.State != aisearch.FleetDeleted {
		t.Fatalf("collect failpoint must persist the deleted state: %+v err=%v", armB, err)
	}
	mu.Lock()
	if collections[armB.Collection] {
		mu.Unlock()
		t.Fatalf("collect failpoint did not delete %q", armB.Collection)
	}
	deletedCount := deleteCalls[armB.Collection]
	mu.Unlock()
	if _, err := db5.ExecContext(ctx, `DROP TRIGGER fail_collect_fence_release`); err != nil {
		t.Fatal(err)
	}
	if _, err := db5.ExecContext(ctx, `UPDATE qdrant_fleet_alias_fences SET holder_pid = ?, holder_start = ? WHERE endpoint_id = ? AND alias_name = ?`, 2147483647, "dead-process", server.URL, "owner-a_docs"); err != nil {
		t.Fatal(err)
	}
	if err := db5.Close(); err != nil {
		t.Fatal(err)
	}
	db6, err := sql.Open("sqlite", sqliteDSN(path))
	if err != nil {
		t.Fatal(err)
	}
	defer db6.Close()
	catalog6, err := aisearch.NewSQLiteFleetGenerationCatalog(db6)
	if err != nil {
		t.Fatal(err)
	}
	if err := catalog6.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	boundary6, err := aisearch.NewSQLiteFleetAliasMutationBoundary(db6)
	if err != nil {
		t.Fatal(err)
	}
	if err := boundary6.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	replayOptions = options(catalog6, &fleetIntegrationGate{})
	replayOptions.AliasBoundary = boundary6
	restarted, err = aisearch.NewQdrantFleetAdapter(replayOptions)
	if err != nil {
		t.Fatal(err)
	}
	if err := restarted.Collect(ctx, []string{"arm-b"}, 1024); err != nil {
		t.Fatalf("deleted-state collect replay did not drain stale fence/operation: %v", err)
	}
	armB, _, err = catalog6.LoadFleetGeneration(ctx, "arm-b")
	if err != nil || armB.State != aisearch.FleetDeleted {
		t.Fatalf("deleted replay changed terminal generation state: %+v err=%v", armB, err)
	}
	var count int
	if err := db6.QueryRowContext(ctx, `SELECT COUNT(*) FROM qdrant_fleet_alias_operations WHERE endpoint_id = ? AND alias_name = ?`, server.URL, "owner-a_docs").Scan(&count); err != nil || count != 0 {
		t.Fatalf("deleted replay left durable operation row count=%d err=%v", count, err)
	}
	if err := db6.QueryRowContext(ctx, `SELECT COUNT(*) FROM qdrant_fleet_alias_fences WHERE endpoint_id = ? AND alias_name = ?`, server.URL, "owner-a_docs").Scan(&count); err != nil || count != 0 {
		t.Fatalf("deleted replay left active fence row count=%d err=%v", count, err)
	}
	mu.Lock()
	deletesAfterReplay := deleteCalls[armB.Collection]
	mu.Unlock()
	if deletesAfterReplay != deletedCount {
		t.Fatalf("terminal-state replay issued another collection delete: before=%d after=%d", deletedCount, deletesAfterReplay)
	}
}

func integrationGeneration(armID, collection string, state aisearch.FleetGenerationState, version int64, recipeID string) aisearch.FleetGeneration {
	record := aisearch.FleetGeneration{OwnerID: "owner-a", Alias: "owner-a_docs", ArmID: armID, RecipeID: recipeID, SnapshotID: "snapshot-" + armID, SnapshotWatermark: "w1", CatchupWatermark: "w1", Dimensions: 2, IndexSpecDigest: "sha256:index", Collection: collection, State: state, Version: version, DataVersion: 1}
	if state == aisearch.FleetCandidate {
		record.ValidationDataVersion = 1
		record.ValidationWatermark = "w1"
		record.ValidationExpected = 1
		record.ValidationActual = 1
		record.ValidationValid = true
		validation := aisearch.FleetValidation{ArmID: record.ArmID, RecipeID: record.RecipeID, Watermark: record.CatchupWatermark, DataVersion: record.DataVersion, ExpectedPointCount: 1, ActualPointCount: 1, Valid: true}
		encoded, _ := json.Marshal(validation)
		digest := sha256.Sum256(encoded)
		record.ValidationDigest = "sha256:" + hex.EncodeToString(digest[:])
	}
	return record
}
