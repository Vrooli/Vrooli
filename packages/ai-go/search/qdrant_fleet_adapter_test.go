package aisearch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

type fleetTestGate struct {
	mu          sync.Mutex
	deny        bool
	calls       int
	badProof    bool
	block       <-chan struct{}
	entered     chan struct{}
	enteredOnce sync.Once
}

type fleetTestAliasBoundary struct {
	mu      sync.Mutex
	byAlias map[string]FleetAliasOperation
}
type fleetTestAliasFence struct {
	boundary    *fleetTestAliasBoundary
	key         string
	operationID string
}

func (b *fleetTestAliasBoundary) AcquireFleetAliasMutation(_ context.Context, operation FleetAliasOperation) (FleetAliasMutationFence, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.byAlias == nil {
		b.byAlias = make(map[string]FleetAliasOperation)
	}
	key := operation.EndpointID + "/" + operation.Alias
	if existing, ok := b.byAlias[key]; ok {
		if existing.OperationID != operation.OperationID || existing.OwnerID != operation.OwnerID || existing.Target != operation.Target {
			return nil, ErrFleetStateConflict
		}
	} else {
		b.byAlias[key] = operation
	}
	return &fleetTestAliasFence{boundary: b, key: key, operationID: operation.OperationID}, nil
}
func (f *fleetTestAliasFence) Release(context.Context) error {
	f.boundary.mu.Lock()
	defer f.boundary.mu.Unlock()
	if current, ok := f.boundary.byAlias[f.key]; !ok || current.OperationID != f.operationID {
		return ErrFleetStateConflict
	}
	delete(f.boundary.byAlias, f.key)
	return nil
}

func (g *fleetTestGate) AdmitQdrantFleetOperation(_ context.Context, _ FleetOperation, _ string, target string, peak int64) (FleetAdmission, error) {
	g.mu.Lock()
	g.calls++
	calls := g.calls
	deny, badProof, block, entered := g.deny, g.badProof, g.block, g.entered
	g.mu.Unlock()
	if block != nil {
		g.enteredOnce.Do(func() { close(entered) })
		<-block
	}
	if deny {
		return FleetAdmission{}, errors.New("capacity denied")
	}
	proof := FleetAdmission{OperationID: fmt.Sprintf("op-test-%d", calls), ReceiptID: fmt.Sprintf("receipt-test-%d", calls), TargetID: target, PeakBytes: peak, ObservedAt: time.Now().Add(-time.Second), ExpiresAt: time.Now().Add(time.Minute)}
	if badProof {
		proof.TargetID = "other-target"
	}
	return proof, nil
}

type fleetTestCatalog struct {
	mu             sync.Mutex
	byArm          map[string]FleetGeneration
	receipts       map[string]FleetAdmissionReceipt
	locks          map[string]FleetAliasOperation
	semanticOps    map[string]FleetSemanticOperationRecord
	sourceVersions map[string]FleetSemanticSourceVersion
	failArm        string
	failState      FleetGenerationState
}

func newFleetTestCatalog() *fleetTestCatalog {
	return &fleetTestCatalog{byArm: map[string]FleetGeneration{}, receipts: map[string]FleetAdmissionReceipt{}, locks: map[string]FleetAliasOperation{}, semanticOps: map[string]FleetSemanticOperationRecord{}, sourceVersions: map[string]FleetSemanticSourceVersion{}}
}

func fleetSemanticOperationKey(record FleetSemanticOperationRecord) string {
	return record.EndpointID + "/" + record.OwnerID + "/" + record.Alias + "/" + record.OperationID
}

func (c *fleetTestCatalog) LoadFleetSemanticOperation(_ context.Context, identity FleetSemanticOperationRecord) (FleetSemanticOperationRecord, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	record, ok := c.semanticOps[fleetSemanticOperationKey(identity)]
	if !ok {
		return FleetSemanticOperationRecord{}, false, nil
	}
	if !sameSemanticRequestIdentity(record, identity) {
		return FleetSemanticOperationRecord{}, false, ErrFleetSemanticOperationConflict
	}
	record.ResultJSON = append([]byte(nil), record.ResultJSON...)
	return record, true, nil
}

func (c *fleetTestCatalog) BeginFleetSemanticOperation(_ context.Context, record FleetSemanticOperationRecord) (FleetSemanticOperationRecord, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := fleetSemanticOperationKey(record)
	if current, ok := c.semanticOps[key]; ok {
		if !sameSemanticRequestIdentity(current, record) || current.ExpectedVersion != record.ExpectedVersion || current.Operation != record.Operation || current.ArmID != record.ArmID || current.TenantID != record.TenantID {
			return FleetSemanticOperationRecord{}, ErrFleetSemanticOperationConflict
		}
		if current.State == FleetSemanticOperationCompleted {
			return current, nil
		}
		current.Admission = record.Admission
		c.semanticOps[key] = current
		return current, nil
	}
	record.State = FleetSemanticOperationStarted
	c.semanticOps[key] = record
	return record, nil
}

func (c *fleetTestCatalog) CompleteFleetSemanticOperation(_ context.Context, identity FleetSemanticOperationRecord, resultJSON []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := fleetSemanticOperationKey(identity)
	current, ok := c.semanticOps[key]
	if !ok || !sameSemanticRequestIdentity(current, identity) {
		return ErrFleetSemanticOperationConflict
	}
	if current.State == FleetSemanticOperationCompleted {
		if string(current.ResultJSON) != string(resultJSON) {
			return ErrFleetSemanticOperationConflict
		}
		return nil
	}
	current.State = FleetSemanticOperationCompleted
	current.ResultJSON = append([]byte(nil), resultJSON...)
	c.semanticOps[key] = current
	return nil
}

func fleetSemanticSourceVersionKey(version FleetSemanticSourceVersion) string {
	return version.EndpointID + "/" + version.OwnerID + "/" + version.Alias + "/" + version.TenantID + "/" + version.SourceID
}

func sameFleetSemanticSourceVersion(left, right FleetSemanticSourceVersion) bool {
	return fleetSemanticSourceVersionKey(left) == fleetSemanticSourceVersionKey(right) && left.SourceSequence == right.SourceSequence && left.OperationID == right.OperationID && left.RequestSHA256 == right.RequestSHA256
}

func (c *fleetTestCatalog) CheckFleetSemanticSourceSequences(_ context.Context, versions []FleetSemanticSourceVersion) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, version := range versions {
		current, ok := c.sourceVersions[fleetSemanticSourceVersionKey(version)]
		if !ok || sameFleetSemanticSourceVersion(current, version) {
			continue
		}
		if current.State == FleetSemanticOperationStarted || version.SourceSequence <= current.SourceSequence {
			return ErrFleetSemanticSequenceStale
		}
	}
	return nil
}

func (c *fleetTestCatalog) ReserveFleetSemanticSourceSequences(_ context.Context, versions []FleetSemanticSourceVersion) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, version := range versions {
		key := fleetSemanticSourceVersionKey(version)
		current, ok := c.sourceVersions[key]
		if ok && !sameFleetSemanticSourceVersion(current, version) {
			if current.State == FleetSemanticOperationStarted || version.SourceSequence <= current.SourceSequence {
				return ErrFleetSemanticSequenceStale
			}
		}
		version.State = FleetSemanticOperationStarted
		c.sourceVersions[key] = version
	}
	return nil
}

func (c *fleetTestCatalog) CompleteFleetSemanticSourceSequences(_ context.Context, operation FleetSemanticOperationRecord, versions []FleetSemanticSourceVersion) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, version := range versions {
		current, ok := c.sourceVersions[fleetSemanticSourceVersionKey(version)]
		if !ok || !sameFleetSemanticSourceVersion(current, version) || current.OperationID != operation.OperationID {
			return ErrFleetSemanticOperationUnresolved
		}
		current.State = FleetSemanticOperationCompleted
		c.sourceVersions[fleetSemanticSourceVersionKey(version)] = current
	}
	return nil
}

func (c *fleetTestCatalog) AcquireFleetAliasOperation(_ context.Context, operation FleetAliasOperation) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := operation.EndpointID + "/" + operation.Alias
	if current, ok := c.locks[key]; ok {
		if current.OperationID == operation.OperationID && current.Kind == operation.Kind && current.ArmID == operation.ArmID && current.Target == operation.Target {
			return nil
		}
		return ErrFleetStateConflict
	}
	c.locks[key] = operation
	return nil
}

func (c *fleetTestCatalog) ReleaseFleetAliasOperation(_ context.Context, operation FleetAliasOperation) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := operation.EndpointID + "/" + operation.Alias
	current, ok := c.locks[key]
	if !ok || current.OperationID != operation.OperationID {
		return ErrFleetStateConflict
	}
	delete(c.locks, key)
	return nil
}
func (c *fleetTestCatalog) LoadFleetGeneration(_ context.Context, id string) (FleetGeneration, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	value, ok := c.byArm[id]
	return value, ok, nil
}
func (c *fleetTestCatalog) LoadFleetGenerationByCollection(_ context.Context, collection string) (FleetGeneration, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, value := range c.byArm {
		if value.Collection == collection {
			return value, true, nil
		}
	}
	return FleetGeneration{}, false, nil
}
func (c *fleetTestCatalog) SaveFleetGenerationCAS(_ context.Context, value FleetGeneration, expected int64) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.failArm == value.ArmID && c.failState == value.State {
		c.failArm = ""
		c.failState = ""
		return errors.New("injected catalog interruption")
	}
	old, ok := c.byArm[value.ArmID]
	if (!ok && expected != 0) || (ok && old.Version != expected) {
		return ErrFleetStateConflict
	}
	c.byArm[value.ArmID] = value
	return nil
}
func (c *fleetTestCatalog) ListFleetGenerations(_ context.Context, owner, alias string) ([]FleetGeneration, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var records []FleetGeneration
	for _, value := range c.byArm {
		if value.OwnerID == owner && value.Alias == alias {
			records = append(records, value)
		}
	}
	return records, nil
}
func (c *fleetTestCatalog) RecordFleetAdmissionReceipt(_ context.Context, receipt FleetAdmissionReceipt) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if existing, ok := c.receipts[receipt.OperationID]; ok && existing != receipt {
		return ErrFleetIdentityConflict
	}
	c.receipts[receipt.OperationID] = receipt
	return nil
}
func (c *fleetTestCatalog) ListFleetAdmissionReceipts(_ context.Context, armID string) ([]FleetAdmissionReceipt, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var receipts []FleetAdmissionReceipt
	for _, receipt := range c.receipts {
		if receipt.ArmID == armID {
			receipts = append(receipts, receipt)
		}
	}
	return receipts, nil
}

type fleetTestSource struct {
	watermark  string
	pages      map[string][]FleetPoint
	changes    []FleetChange
	pointCount uint64
}

func (s *fleetTestSource) PinFleetSnapshot(_ context.Context, id string) (FleetSnapshot, error) {
	return FleetSnapshot{ID: id, Watermark: s.watermark}, nil
}
func (s *fleetTestSource) ReadFleetSnapshotPage(_ context.Context, _ string, _ FleetSnapshot, cursor string, _ int) ([]FleetPoint, string, bool, error) {
	if cursor == "" {
		return s.pages["first"], "done", true, nil
	}
	return nil, cursor, true, nil
}
func (s *fleetTestSource) ReadFleetChanges(_ context.Context, _, snapshot, after string, _ int) ([]FleetChange, string, error) {
	if snapshot == "" {
		return nil, after, errors.New("snapshot required")
	}
	var out []FleetChange
	for _, change := range s.changes {
		if change.Sequence > after {
			out = append(out, change)
		}
	}
	if len(out) == 0 {
		return nil, after, nil
	}
	return out, out[len(out)-1].Sequence, nil
}
func (s *fleetTestSource) CompareFleetSequence(_ context.Context, before, after string) (int, error) {
	return strings.Compare(after, before), nil
}
func (s *fleetTestSource) FleetWatermark(context.Context) (string, error) { return s.watermark, nil }
func (s *fleetTestSource) ExpectedFleetPointCount(context.Context, string, string, string) (uint64, error) {
	return s.pointCount, nil
}
func (s *fleetTestSource) AcquireFleetCutover(_ context.Context, expected string) (FleetCutoverFence, error) {
	if s.watermark != expected {
		return nil, errors.New("watermark conflict")
	}
	return fleetTestFence{s: s}, nil
}

type fleetTestFence struct{ s *fleetTestSource }

func (f fleetTestFence) CurrentWatermark(context.Context) (string, error) { return f.s.watermark, nil }
func (fleetTestFence) Release(context.Context) error                      { return nil }

func TestQdrantFleetStageCatchupPromoteRollbackAndProtectedGC(t *testing.T) {
	var mu sync.Mutex
	collections := map[string]bool{}
	points := map[string]map[string]FleetPoint{}
	aliases := map[string]string{"owner_docs": "old-generation"}
	collections["old-generation"] = true
	points["old-generation"] = map[string]FleetPoint{"old-point": {ID: "old-point", Vector: []float32{1, 0}, Payload: map[string]any{"source_id": "s1"}}}
	deleted := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		path := r.URL.Path
		if path == "/collections/aliases" && r.Method == http.MethodGet {
			items := []map[string]string{}
			for name, collection := range aliases {
				items = append(items, map[string]string{"alias_name": name, "collection_name": collection})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{"aliases": items}})
			return
		}
		if strings.HasPrefix(path, "/collections/") {
			parts := strings.Split(path, "/")
			name := parts[2]
			if len(parts) == 3 {
				switch r.Method {
				case http.MethodGet:
					if !collections[name] {
						http.NotFound(w, r)
						return
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{"config": map[string]any{"params": map[string]any{"vectors": map[string]any{"dense": map[string]any{"size": 2, "distance": "Cosine"}}, "sparse_vectors": map[string]any{}}}}})
					return
				case http.MethodPut:
					collections[name] = true
					points[name] = map[string]FleetPoint{}
					w.WriteHeader(http.StatusOK)
					return
				case http.MethodDelete:
					if !collections[name] {
						http.NotFound(w, r)
						return
					}
					delete(collections, name)
					delete(points, name)
					deleted = append(deleted, name)
					w.WriteHeader(http.StatusOK)
					return
				}
			}
			if len(parts) >= 4 && parts[3] == "points" && r.Method == http.MethodPut {
				var body struct {
					Points []struct {
						ID      string          `json:"id"`
						Vector  json.RawMessage `json:"vector"`
						Payload map[string]any  `json:"payload"`
					} `json:"points"`
				}
				_ = json.NewDecoder(r.Body).Decode(&body)
				for _, p := range body.Points {
					var vector []float64
					var named map[string][]float64
					if json.Unmarshal(p.Vector, &named) == nil {
						vector = named["dense"]
					} else {
						_ = json.Unmarshal(p.Vector, &vector)
					}
					converted := make([]float32, len(vector))
					for i, v := range vector {
						converted[i] = float32(v)
					}
					points[name][p.ID] = FleetPoint{ID: p.ID, Vector: converted, Payload: p.Payload}
				}
				w.WriteHeader(http.StatusOK)
				return
			}
			if len(parts) >= 5 && parts[3] == "points" && r.Method == http.MethodGet {
				point, ok := points[name][parts[4]]
				if !ok {
					http.NotFound(w, r)
					return
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{"payload": point.Payload}})
				return
			}
			if len(parts) >= 5 && parts[3] == "points" && parts[4] == "count" && r.Method == http.MethodPost {
				var body struct {
					Filter struct {
						Must []struct {
							Key   string `json:"key"`
							Match struct {
								Value string `json:"value"`
							} `json:"match"`
						} `json:"must"`
					} `json:"filter"`
				}
				_ = json.NewDecoder(r.Body).Decode(&body)
				var count uint64
				for _, point := range points[name] {
					recipe, _ := point.Payload["embedding_recipe_id"].(string)
					if recipe == body.Filter.Must[0].Match.Value {
						count++
					}
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"result": map[string]any{"count": count}})
				return
			}
		}
		if path == "/collections/aliases" && r.Method == http.MethodPost {
			var body struct {
				Actions []map[string]map[string]string `json:"actions"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			for _, action := range body.Actions {
				if del, ok := action["delete_alias"]; ok {
					delete(aliases, del["alias_name"])
				}
				if create, ok := action["create_alias"]; ok {
					aliases[create["alias_name"]] = create["collection_name"]
				}
			}
			w.WriteHeader(http.StatusOK)
			return
		}
		if strings.HasSuffix(path, "/points/delete") {
			var body struct {
				Filter struct {
					Must []struct {
						Key   string `json:"key"`
						Match struct {
							Any []string `json:"any"`
						} `json:"match"`
					} `json:"must"`
				} `json:"filter"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			name := strings.Split(path, "/")[2]
			for id, p := range points[name] {
				for _, condition := range body.Filter.Must {
					if condition.Key == "source_id" {
						source, _ := p.Payload["source_id"].(string)
						for _, target := range condition.Match.Any {
							if source == target {
								delete(points[name], id)
							}
						}
					}
				}
			}
			w.WriteHeader(http.StatusOK)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	gate := &fleetTestGate{}
	source := &fleetTestSource{watermark: "w1", pointCount: 1, pages: map[string][]FleetPoint{"first": {{ID: "p1", Vector: []float32{1, 0}, Payload: map[string]any{"source_id": "s1"}}}}}
	catalog := newFleetTestCatalog()
	oldGeneration := FleetGeneration{OwnerID: "owner", Alias: "owner_docs", ArmID: "arm-old", RecipeID: "sha256:old-recipe", SnapshotID: "snapshot-0", SnapshotWatermark: "w1", CatchupWatermark: "w1", Dimensions: 2, IndexSpecDigest: "sha256:old-index", Collection: "old-generation", State: FleetActive, Version: 1}
	if err := catalog.SaveFleetGenerationCAS(context.Background(), oldGeneration, 0); err != nil {
		t.Fatal(err)
	}
	opts := QdrantFleetOptions{BaseURL: server.URL, OwnerID: "owner", Alias: "owner_docs", Client: server.Client(), Admission: gate, AliasBoundary: &fleetTestAliasBoundary{}, Source: source, Catalog: catalog, RetiredMaxAge: time.Nanosecond}
	adapter, err := NewQdrantFleetAdapter(opts)
	if err != nil {
		t.Fatal(err)
	}
	if err := adapter.setAlias(context.Background(), "candidate-unexpected", "foreign-collection", true); err == nil {
		t.Fatal("alias setter accepted an unexpected prior mapping")
	}
	if aliases["owner_docs"] != "old-generation" {
		t.Fatalf("alias changed despite unexpected prior mapping: %q", aliases["owner_docs"])
	}
	recipe := testFleetRecipe(t)
	first, err := adapter.Stage(context.Background(), "arm-a", recipe, "snapshot-1", 128)
	if err != nil {
		t.Fatal(err)
	}
	if first.State != FleetCandidate || first.CatchupWatermark != "w1" {
		t.Fatalf("stage state: %+v", first)
	}
	if _, err := adapter.Stage(context.Background(), "arm-a", recipe, "snapshot-1", 128); err != nil {
		t.Fatalf("idempotent stage retry: %v", err)
	}
	source.watermark = "w2"
	source.changes = []FleetChange{{Sequence: "w2", Upserts: []FleetPoint{{ID: "replay", Vector: []float32{0, 1}, Payload: map[string]any{"source_id": "replay"}}}}, {Sequence: "w2", Upserts: []FleetPoint{{ID: "replay-again", Vector: []float32{0, 1}, Payload: map[string]any{"source_id": "replay-again"}}}}}
	if _, err := adapter.CatchUp(context.Background(), "arm-a", 128); err == nil || !strings.Contains(err.Error(), "duplicate or not after") {
		t.Fatalf("duplicate sequence page accepted: %v", err)
	}
	if _, exists := points[first.Collection]["replay"]; exists {
		t.Fatal("point write occurred before duplicate page rejection")
	}
	source.changes = []FleetChange{{Sequence: "w2", Deletes: []string{"s1"}, Upserts: []FleetPoint{{ID: "p2", Vector: []float32{0, 1}, Payload: map[string]any{"source_id": "s2"}}}}}
	updated, err := adapter.CatchUp(context.Background(), "arm-a", 128)
	if err != nil {
		t.Fatal(err)
	}
	if updated.CatchupWatermark != "w2" {
		t.Fatalf("catch-up watermark=%q", updated.CatchupWatermark)
	}
	if _, err := adapter.Promote(context.Background(), "arm-a", 128); err == nil || !strings.Contains(err.Error(), "validation receipt") {
		t.Fatalf("promotion bypassed structural validation: %v", err)
	}
	source.pointCount = 2
	invalidValidation, err := adapter.Validate(context.Background(), "arm-a")
	if err != nil || invalidValidation.Valid {
		t.Fatalf("mismatched structural validation=%+v err=%v", invalidValidation, err)
	}
	if _, err := adapter.Promote(context.Background(), "arm-a", 128); err == nil || !strings.Contains(err.Error(), "validation receipt") {
		t.Fatalf("promotion accepted failed validation receipt: %v", err)
	}
	source.pointCount = 1
	validation, err := adapter.Validate(context.Background(), "arm-a")
	if err != nil || !validation.Valid || validation.ExpectedPointCount != 1 || validation.ActualPointCount != 1 {
		t.Fatalf("candidate structural validation=%+v err=%v", validation, err)
	}
	other, err := adapter.Stage(context.Background(), "arm-b", recipe, "snapshot-b", 128)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := adapter.Validate(context.Background(), "arm-b"); err != nil {
		t.Fatal(err)
	}
	gate.deny = true
	if _, err := adapter.Promote(context.Background(), "arm-a", 128); err == nil || !strings.Contains(err.Error(), "capacity denied") {
		t.Fatalf("expected interrupted promotion after durable lock, got %v", err)
	}
	if _, err := adapter.Promote(context.Background(), "arm-b", 128); !errors.Is(err, ErrFleetStateConflict) {
		t.Fatalf("distinct adapter operation bypassed owner+alias lock: %v", err)
	}
	gate.deny = false
	catalog.failArm = "arm-a"
	catalog.failState = FleetActive
	entered := make(chan struct{})
	unblock := make(chan struct{})
	gate.mu.Lock()
	gate.entered = entered
	gate.block = unblock
	gate.enteredOnce = sync.Once{}
	gate.mu.Unlock()
	var promoteErr error
	var promoteDone sync.WaitGroup
	promoteDone.Add(1)
	go func() {
		defer promoteDone.Done()
		_, promoteErr = adapter.Promote(context.Background(), "arm-a", 128)
	}()
	<-entered
	concurrentAdapter, err := NewQdrantFleetAdapter(opts)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := concurrentAdapter.Promote(context.Background(), "arm-b", 128); !errors.Is(err, ErrFleetStateConflict) {
		t.Fatalf("concurrent adapter bypassed owner+alias lock: %v", err)
	}
	close(unblock)
	promoteDone.Wait()
	if promoteErr == nil || !strings.Contains(promoteErr.Error(), "alias switched but generation catalog write failed") {
		t.Fatalf("expected failure after alias switch, got %v", promoteErr)
	}
	gate.mu.Lock()
	gate.block = nil
	gate.entered = nil
	gate.mu.Unlock()
	if aliases["owner_docs"] != first.Collection {
		t.Fatalf("promotion failpoint did not occur after alias mutation: %q", aliases["owner_docs"])
	}
	if _, err := adapter.Promote(context.Background(), "arm-b", 128); !errors.Is(err, ErrFleetStateConflict) {
		t.Fatalf("other promotion bypassed interrupted alias operation: %v", err)
	}
	restartedAdapter, err := NewQdrantFleetAdapter(opts)
	if err != nil {
		t.Fatal(err)
	}
	active, err := restartedAdapter.Promote(context.Background(), "arm-a", 128)
	if err != nil {
		t.Fatal(err)
	}
	if aliases["owner_docs"] != active.Collection {
		t.Fatalf("active alias=%q; want %q", aliases["owner_docs"], active.Collection)
	}
	if other.Collection == active.Collection {
		t.Fatal("test candidates unexpectedly share a physical collection")
	}
	if _, exists := points[active.Collection]["p2"]; !exists || points[active.Collection]["p1"].ID != "" {
		t.Fatalf("tombstone was not applied: %#v", points[active.Collection])
	}
	catalog.failArm = "arm-old"
	catalog.failState = FleetActive
	aliases["owner_docs"] = "foreign-out-of-band"
	if _, err := restartedAdapter.Rollback(context.Background(), "arm-a", 128); err == nil || !strings.Contains(err.Error(), "unexpected collection") {
		t.Fatalf("rollback accepted an out-of-band alias target: %v", err)
	}
	if aliases["owner_docs"] != "foreign-out-of-band" {
		t.Fatalf("rollback mutated unexpected alias target: %q", aliases["owner_docs"])
	}
	aliases["owner_docs"] = active.Collection
	if _, err := restartedAdapter.Rollback(context.Background(), "arm-a", 128); err == nil || !strings.Contains(err.Error(), "alias rolled back but previous generation catalog update failed") {
		t.Fatalf("expected rollback interruption after alias switch, got %v", err)
	}
	if aliases["owner_docs"] != "old-generation" {
		t.Fatalf("rollback failpoint did not occur after alias mutation: %q", aliases["owner_docs"])
	}
	restartedAdapter, err = NewQdrantFleetAdapter(opts)
	if err != nil {
		t.Fatal(err)
	}
	rolledBack, err := restartedAdapter.Rollback(context.Background(), "arm-a", 128)
	if err != nil {
		t.Fatal(err)
	}
	if rolledBack.State != FleetRetired || aliases["owner_docs"] != "old-generation" {
		t.Fatalf("rollback record=%+v active alias=%q", rolledBack, aliases["owner_docs"])
	}
	if _, exists := points["old-generation"]["p2"]; !exists {
		t.Fatalf("rollback target missed catch-up upsert: %#v", points["old-generation"])
	}
	if _, exists := points["old-generation"]["old-point"]; exists {
		t.Fatalf("rollback target missed source tombstone: %#v", points["old-generation"])
	}
	observation, err := adapter.ObserveStorage(context.Background(), "arm-a")
	if err != nil || observation.QdrantDiskBytesKnown || observation.DiskObservedAt.IsZero() {
		t.Fatalf("missing Qdrant byte stats must persist explicit unknown: generation=%+v err=%v", observation, err)
	}
	retired, _, err := catalog.LoadFleetGeneration(context.Background(), "arm-a")
	if err != nil {
		t.Fatal(err)
	}
	retired.Protected = true
	retired.Version++
	if err := catalog.SaveFleetGenerationCAS(context.Background(), retired, retired.Version-1); err != nil {
		t.Fatal(err)
	}
	if err := adapter.Collect(context.Background(), []string{"arm-a"}, 128); !errors.Is(err, ErrFleetStateConflict) {
		t.Fatalf("explicitly protected generation collection error=%v", err)
	}
	retired, _, err = catalog.LoadFleetGeneration(context.Background(), "arm-a")
	if err != nil {
		t.Fatal(err)
	}
	retired.Protected = false
	retired.Version++
	if err := catalog.SaveFleetGenerationCAS(context.Background(), retired, retired.Version-1); err != nil {
		t.Fatal(err)
	}
	if err := adapter.Collect(context.Background(), []string{"arm-a"}, 128); err == nil || !strings.Contains(err.Error(), "protects it") {
		t.Fatalf("rollback-reference protected generation collection error=%v", err)
	}
	oldGeneration, _, err = catalog.LoadFleetGeneration(context.Background(), "arm-old")
	if err != nil {
		t.Fatal(err)
	}
	oldGeneration.RollbackTo = ""
	oldGeneration.Version++
	if err := catalog.SaveFleetGenerationCAS(context.Background(), oldGeneration, oldGeneration.Version-1); err != nil {
		t.Fatal(err)
	}
	catalog.failArm = "arm-a"
	catalog.failState = FleetDeleted
	if err := restartedAdapter.Collect(context.Background(), []string{"arm-a"}, 128); err == nil || !strings.Contains(err.Error(), "injected catalog interruption") {
		t.Fatalf("expected delete/catalog interruption, got %v", err)
	}
	if collections[active.Collection] {
		t.Fatal("collect failpoint did not occur after physical delete")
	}
	restartedAdapter, err = NewQdrantFleetAdapter(opts)
	if err != nil {
		t.Fatal(err)
	}
	if err := restartedAdapter.Collect(context.Background(), []string{"arm-a"}, 128); err != nil {
		t.Fatalf("collect replay did not reconcile already deleted collection: %v", err)
	}
	receipts, err := catalog.ListFleetAdmissionReceipts(context.Background(), "arm-a")
	if err != nil || len(receipts) == 0 || receipts[0].PeakBytes != 128 {
		t.Fatalf("per-operation admission receipts were not durable: receipts=%+v err=%v", receipts, err)
	}
}

func TestQdrantFleetRejectsMissingOrStaleAdmissionBeforeWrites(t *testing.T) {
	gate := &fleetTestGate{badProof: true}
	source := &fleetTestSource{watermark: "w1", pointCount: 1, pages: map[string][]FleetPoint{"first": {{ID: "p1", Vector: []float32{1, 0}, Payload: map[string]any{"source_id": "s1"}}}}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected Qdrant mutation/read after invalid admission: %s %s", r.Method, r.URL.Path)
		http.NotFound(w, r)
	}))
	defer server.Close()
	adapter, err := NewQdrantFleetAdapter(QdrantFleetOptions{BaseURL: server.URL, OwnerID: "owner", Alias: "owner_docs", Client: server.Client(), Admission: gate, Source: source, Catalog: newFleetTestCatalog()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := adapter.Stage(context.Background(), "arm-a", testFleetRecipe(t), "snapshot-1", 100); !errors.Is(err, ErrFleetAdmissionRequired) {
		t.Fatalf("invalid admission error=%v", err)
	}
}

func TestQdrantFleetRequiresAdmissionAndSourceIdentity(t *testing.T) {
	if _, err := NewQdrantFleetAdapter(QdrantFleetOptions{OwnerID: "owner", Alias: "owner_docs", Source: &fleetTestSource{}, Catalog: newFleetTestCatalog()}); !errors.Is(err, ErrFleetAdmissionRequired) {
		t.Fatalf("missing gate error=%v", err)
	}
	if _, err := NewQdrantFleetAdapter(QdrantFleetOptions{OwnerID: "search-hub.experimentarms", Alias: "agent-manager_conversation-search", Admission: &fleetTestGate{}, Source: &fleetTestSource{}, Catalog: newFleetTestCatalog()}); err == nil || !strings.Contains(err.Error(), "must be namespaced beneath owner") {
		t.Fatalf("cross-owner alias namespace was accepted: %v", err)
	}
	if err := validateFleetPoints([]FleetPoint{{ID: "p1", Vector: []float32{1, 0}, Payload: map[string]any{}}}, 2, false); err == nil {
		t.Fatal("point without source identity was accepted")
	}
	if err := validateFleetPoints([]FleetPoint{{ID: "p1", Vector: []float32{1, 0}, Sparse: &SparseVector{Indices: []uint32{1}, Values: []float32{1}}, Payload: map[string]any{"source_id": "s1"}}}, 2, false); err == nil {
		t.Fatal("unexpected sparse vector was accepted for a dense-only collection")
	}
	if err := validateFleetPoints([]FleetPoint{{ID: "p1", Vector: []float32{1, 0}, Sparse: &SparseVector{Indices: []uint32{1}, Values: []float32{1}}, Payload: map[string]any{"source_id": "s1"}}}, 2, true); err != nil {
		t.Fatalf("valid sparse point rejected: %v", err)
	}
	sequenceAdapter := &QdrantFleetAdapter{source: &fleetTestSource{}}
	if _, err := sequenceAdapter.validateFleetChangePage(context.Background(), "w1", []FleetChange{{Sequence: "w3"}, {Sequence: "w2"}}, "w2"); err == nil {
		t.Fatal("out-of-order change page was accepted")
	}
}

func TestCanonicalQdrantEndpointIDNormalizesSharedLockIdentity(t *testing.T) {
	got, err := CanonicalQdrantEndpointID("HTTP://LOCALHOST:80/qdrant///")
	if err != nil {
		t.Fatal(err)
	}
	if got != "http://localhost/qdrant" {
		t.Fatalf("canonical endpoint=%q, want http://localhost/qdrant", got)
	}
	for _, raw := range []string{"http://user:secret@localhost:6333", "http://localhost:6333/?api_key=secret", "file:///qdrant"} {
		if _, err := CanonicalQdrantEndpointID(raw); err == nil {
			t.Errorf("endpoint with forbidden or unsupported identity %q was accepted", raw)
		}
	}
}

func TestQdrantFleetAliasMutationsFailClosedWithoutOwnerBoundary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("Qdrant request reached without owner-wide alias boundary: %s %s", r.Method, r.URL.Path)
		http.Error(w, "unexpected", http.StatusInternalServerError)
	}))
	defer server.Close()
	catalog := newFleetTestCatalog()
	record := FleetGeneration{OwnerID: "owner", Alias: "owner_docs", ArmID: "arm-a", RecipeID: "recipe-a", SnapshotID: "snapshot-a", SnapshotWatermark: "w1", CatchupWatermark: "w1", Dimensions: 2, IndexSpecDigest: "sha256:index", Collection: "collection-a", State: FleetCandidate, Version: 1, DataVersion: 1, ValidationDataVersion: 1, ValidationWatermark: "w1", ValidationExpected: 1, ValidationActual: 1, ValidationValid: true}
	record.ValidationDigest = fleetValidationDigest(FleetValidation{ArmID: record.ArmID, RecipeID: record.RecipeID, Watermark: record.CatchupWatermark, DataVersion: record.DataVersion, ExpectedPointCount: 1, ActualPointCount: 1, Valid: true})
	if err := catalog.SaveFleetGenerationCAS(context.Background(), record, 0); err != nil {
		t.Fatal(err)
	}
	adapter, err := NewQdrantFleetAdapter(QdrantFleetOptions{BaseURL: server.URL, OwnerID: "owner", Alias: "owner_docs", Client: server.Client(), Admission: &fleetTestGate{}, Source: &fleetTestSource{watermark: "w1"}, Catalog: catalog})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := adapter.Promote(context.Background(), "arm-a", 128); !errors.Is(err, ErrFleetAliasBoundaryRequired) {
		t.Fatalf("promotion without owner-wide alias boundary error=%v", err)
	}
}

func testFleetRecipe(t *testing.T) EmbeddingRecipe {
	t.Helper()
	recipe, err := NewEmbeddingRecipe(EmbeddingRecipeInput{SchemaVersion: "1", ModelID: "test-model", ModelRevision: "revision-1", ArtifactSHA256: strings.Repeat("a", 64), TokenizerRevision: "tokenizer-1", Runtime: "test-runtime", RuntimeBuild: "build-1", Precision: "fp32", QueryInstruction: "none", DocumentInstruction: "none", Pooling: "mean", Normalization: "l2", Dimensions: 2, OutputDType: "float32", Truncation: "truncate-1", ChunkerVersion: "chunk-1", ComposerVersion: "composer-1"})
	if err != nil {
		t.Fatal(err)
	}
	return recipe
}
