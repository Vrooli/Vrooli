package aisearch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

const fixtureDenseSize = 1234

// capturingDoer records every request and replies with a scripted response.
type capturingDoer struct {
	requests []capturedReq
	respond  func(req capturedReq) (int, string)
}

type capturedReq struct {
	method string
	url    string
	body   map[string]any
}

func (c *capturingDoer) Do(req *http.Request) (*http.Response, error) {
	cap := capturedReq{method: req.Method, url: req.URL.String()}
	if req.Body != nil {
		raw, _ := io.ReadAll(req.Body)
		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &cap.body)
		}
	}
	c.requests = append(c.requests, cap)
	status, body := http.StatusOK, "{}"
	if c.respond != nil {
		status, body = c.respond(cap)
	}
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewReader([]byte(body))),
		Header:     make(http.Header),
	}, nil
}

func (c *capturingDoer) last() capturedReq { return c.requests[len(c.requests)-1] }

// createPut returns the collection-create PUT request (the one whose path ends
// at /collections/<name>), distinct from the meta-sentinel /points upsert that
// now follows every creation.
func (c *capturingDoer) createPut(t *testing.T, collection string) capturedReq {
	t.Helper()
	suffix := "/collections/" + collection
	for _, r := range c.requests {
		if r.method == http.MethodPut && strings.HasSuffix(strings.Split(r.url, "?")[0], suffix) {
			return r
		}
	}
	t.Fatalf("no create PUT for collection %q in %d requests", collection, len(c.requests))
	return capturedReq{}
}

func TestEnsureCollectionHybridShape(t *testing.T) {
	doer := &capturingDoer{respond: func(req capturedReq) (int, string) {
		if req.method == http.MethodGet {
			return http.StatusNotFound, "{}" // force creation
		}
		return http.StatusOK, "{}"
	}}
	store := NewVectorStoreWithClient("http://q", "", "vrooli-docs", doer)
	if err := store.EnsureCollection(context.Background(), CollectionSpec{DenseSize: fixtureDenseSize, Sparse: true, SparseModifier: "idf"}); err != nil {
		t.Fatal(err)
	}
	put := doer.createPut(t, "vrooli-docs")
	vectors, _ := put.body["vectors"].(map[string]any)
	if _, ok := vectors["dense"]; !ok {
		t.Fatalf("expected named dense vector, got %v", put.body["vectors"])
	}
	sparse, ok := put.body["sparse_vectors"].(map[string]any)
	if !ok {
		t.Fatalf("expected sparse_vectors block, got %v", put.body)
	}
	s, _ := sparse["sparse"].(map[string]any)
	if s["modifier"] != "idf" {
		t.Fatalf("expected idf modifier, got %v", sparse["sparse"])
	}
}

func TestEnsureCollectionAppliesStorageProfile(t *testing.T) {
	doer := &capturingDoer{respond: func(req capturedReq) (int, string) {
		if req.method == http.MethodGet {
			return http.StatusNotFound, "{}"
		}
		return http.StatusOK, "{}"
	}}
	store := NewVectorStoreWithClient("http://q", "", "profiled", doer)
	err := store.EnsureCollection(context.Background(), CollectionSpec{
		DenseSize: fixtureDenseSize,
		Sparse:    true,
		Storage: StorageProfile{
			OnDiskVectors:          true,
			OnDiskPayload:          true,
			OnDiskSparse:           true,
			OnDiskHNSW:             true,
			HNSWM:                  24,
			HNSWEFConstruct:        128,
			FullScanThreshold:      20_000,
			IndexingThreshold:      30_000,
			MaxOptimizationThreads: 3,
			UpsertBatchSize:        128,
			ScalarQuantization:     true,
			Quantile:               0.99,
			QuantizationAlwaysRAM:  true,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	put := doer.createPut(t, "profiled")
	vectors, _ := put.body["vectors"].(map[string]any)
	dense, _ := vectors["dense"].(map[string]any)
	if dense["on_disk"] != true || put.body["on_disk_payload"] != true {
		t.Fatalf("dense/payload disk profile missing: %v", put.body)
	}
	sparse, _ := put.body["sparse_vectors"].(map[string]any)
	sparseVector, _ := sparse["sparse"].(map[string]any)
	sparseIndex, _ := sparseVector["index"].(map[string]any)
	if sparseIndex["on_disk"] != true {
		t.Fatalf("sparse disk profile missing: %v", sparseVector)
	}
	hnsw, _ := put.body["hnsw_config"].(map[string]any)
	if hnsw["on_disk"] != true || hnsw["m"] != float64(24) || hnsw["ef_construct"] != float64(128) {
		t.Fatalf("HNSW profile missing: %v", hnsw)
	}
	quantization, _ := put.body["quantization_config"].(map[string]any)
	if _, ok := quantization["scalar"]; !ok {
		t.Fatalf("quantization profile missing: %v", quantization)
	}
	optimizers, _ := put.body["optimizers_config"].(map[string]any)
	if optimizers["max_optimization_threads"] != float64(3) {
		t.Fatalf("optimizer worker profile missing: %v", optimizers)
	}
}

func TestQdrantUpsertBatchBoundsRequests(t *testing.T) {
	doer := &capturingDoer{}
	store := NewVectorStoreWithClient("http://q", "", "batched", doer)
	batchStore, ok := store.(BatchVectorStore)
	if !ok {
		t.Fatal("Qdrant store must implement BatchVectorStore")
	}
	points := make([]Point, 5)
	for i := range points {
		points[i] = Point{ID: fmt.Sprintf("point-%d", i), Dense: []float64{float64(i)}}
	}
	if err := batchStore.UpsertBatch(context.Background(), points, 2); err != nil {
		t.Fatal(err)
	}
	if len(doer.requests) != 3 {
		t.Fatalf("expected bounded batches of 2,2,1; got %d requests", len(doer.requests))
	}
	for i, request := range doer.requests {
		batch, _ := request.body["points"].([]any)
		if len(batch) > 2 {
			t.Fatalf("request %d exceeded batch bound: %d", i, len(batch))
		}
	}
}

func TestEnsureCollectionRejectsInvalidStorageProfileBeforeHTTP(t *testing.T) {
	doer := &capturingDoer{}
	store := NewVectorStoreWithClient("http://q", "", "invalid-profile", doer)
	err := store.EnsureCollection(context.Background(), CollectionSpec{Storage: StorageProfile{Quantile: 1.5}})
	if err == nil || !strings.Contains(err.Error(), "quantile") {
		t.Fatalf("expected quantile validation error, got %v", err)
	}
	if len(doer.requests) != 0 {
		t.Fatalf("invalid profile must fail before HTTP, got %d request(s)", len(doer.requests))
	}
}

func TestEnsureCollectionDenseOnlyOmitsSparse(t *testing.T) {
	doer := &capturingDoer{respond: func(req capturedReq) (int, string) {
		if req.method == http.MethodGet {
			return http.StatusNotFound, "{}"
		}
		return http.StatusOK, "{}"
	}}
	store := NewVectorStoreWithClient("http://q", "", "cli-health-commands", doer)
	if err := store.EnsureCollection(context.Background(), CollectionSpec{DenseSize: fixtureDenseSize}); err != nil {
		t.Fatal(err)
	}
	put := doer.createPut(t, "cli-health-commands")
	if _, ok := put.body["sparse_vectors"]; ok {
		t.Fatalf("dense-only collection must not declare sparse_vectors: %v", put.body)
	}
	vectors, _ := put.body["vectors"].(map[string]any)
	if _, ok := vectors["dense"]; !ok {
		t.Fatal("dense-only collection must still use a NAMED dense vector (D5)")
	}
}

// matchingDenseInfo is a GET /collections/<name> body describing a named
// dense vector of the given size+distance with no sparse vector.
func matchingDenseInfo(size int, distance string) string {
	return fmt.Sprintf(`{"result":{"config":{"params":{"vectors":{"dense":{"size":%d,"distance":%q}}}}}}`, size, distance)
}

func matchingUnnamedDenseInfo(size int, distance string) string {
	return fmt.Sprintf(`{"result":{"config":{"params":{"vectors":{"size":%d,"distance":%q}}}}}`, size, distance)
}

func isCollectionInfoGet(req capturedReq) bool {
	return req.method == http.MethodGet && strings.HasSuffix(req.url, "/collections/cli-health-commands")
}

func isPointGet(req capturedReq) bool {
	return req.method == http.MethodGet && strings.Contains(req.url, "/collections/cli-health-commands/points/")
}

func TestEnsureCollectionSchemaGuardUnnamedVector(t *testing.T) {
	doer := &capturingDoer{respond: func(req capturedReq) (int, string) {
		if isCollectionInfoGet(req) {
			// Legacy single unnamed vector — the production trap.
			return http.StatusOK, matchingUnnamedDenseInfo(fixtureDenseSize, "Cosine")
		}
		return http.StatusOK, "{}"
	}}
	store := NewVectorStoreWithClient("http://q", "", "cli-health-commands", doer)
	err := store.EnsureCollection(context.Background(), CollectionSpec{DenseSize: fixtureDenseSize, DenseDistance: "Cosine"})
	if !errors.Is(err, ErrCollectionSchemaMismatch) {
		t.Fatalf("expected schema mismatch on unnamed vector, got %v", err)
	}
	for _, r := range doer.requests {
		if r.method == http.MethodPut {
			t.Fatalf("guard must not PUT (create/modify) on mismatch, saw %s %s", r.method, r.url)
		}
	}
}

func TestEnsureCollectionSchemaGuardWrongSize(t *testing.T) {
	doer := &capturingDoer{respond: func(req capturedReq) (int, string) {
		if isCollectionInfoGet(req) {
			return http.StatusOK, matchingDenseInfo(2345, "Cosine")
		}
		return http.StatusOK, "{}"
	}}
	store := NewVectorStoreWithClient("http://q", "", "cli-health-commands", doer)
	err := store.EnsureCollection(context.Background(), CollectionSpec{DenseSize: fixtureDenseSize, DenseDistance: "Cosine"})
	if !errors.Is(err, ErrCollectionSchemaMismatch) {
		t.Fatalf("expected schema mismatch on wrong size, got %v", err)
	}
}

func TestEnsureCollectionSchemaGuardWrongModel(t *testing.T) {
	doer := &capturingDoer{respond: func(req capturedReq) (int, string) {
		switch {
		case isCollectionInfoGet(req):
			return http.StatusOK, matchingDenseInfo(fixtureDenseSize, "Cosine")
		case isPointGet(req):
			return http.StatusOK, fmt.Sprintf(`{"result":{"payload":{"__aisearch_meta__":true,"model":"other-fixture-embed:latest","dense_size":%d}}}`, fixtureDenseSize)
		}
		return http.StatusOK, "{}"
	}}
	store := NewVectorStoreWithClient("http://q", "", "cli-health-commands", doer)
	err := store.EnsureCollection(context.Background(), CollectionSpec{DenseSize: fixtureDenseSize, DenseDistance: "Cosine", Model: "fixture-embed-model:latest"})
	if !errors.Is(err, ErrCollectionSchemaMismatch) {
		t.Fatalf("expected schema mismatch on wrong model, got %v", err)
	}
}

func TestEnsureCollectionSchemaGuardAcceptsImplicitLatestModelTag(t *testing.T) {
	doer := &capturingDoer{respond: func(req capturedReq) (int, string) {
		switch {
		case isCollectionInfoGet(req):
			return http.StatusOK, matchingDenseInfo(fixtureDenseSize, "Cosine")
		case isPointGet(req):
			return http.StatusOK, fmt.Sprintf(`{"result":{"payload":{"__aisearch_meta__":true,"model":"fixture-embed-model","dense_size":%d}}}`, fixtureDenseSize)
		}
		return http.StatusOK, "{}"
	}}
	store := NewVectorStoreWithClient("http://q", "", "cli-health-commands", doer)
	err := store.EnsureCollection(context.Background(), CollectionSpec{DenseSize: fixtureDenseSize, DenseDistance: "Cosine", Model: "fixture-embed-model:latest"})
	if err != nil {
		t.Fatalf("expected untagged model metadata to match explicit :latest spec, got %v", err)
	}
}

func TestEnsureCollectionSchemaGuardWrongRole(t *testing.T) {
	doer := &capturingDoer{respond: func(req capturedReq) (int, string) {
		switch {
		case isCollectionInfoGet(req):
			return http.StatusOK, matchingDenseInfo(fixtureDenseSize, "Cosine")
		case isPointGet(req):
			return http.StatusOK, fmt.Sprintf(`{"result":{"payload":{"__aisearch_meta__":true,"model":"fixture-embed-model:latest","role":"embedding.other","dense_size":%d,"policy_schema_version":"fixture-v1"}}}`, fixtureDenseSize)
		}
		return http.StatusOK, "{}"
	}}
	store := NewVectorStoreWithClient("http://q", "", "cli-health-commands", doer)
	err := store.EnsureCollection(context.Background(), CollectionSpec{
		DenseSize: fixtureDenseSize, DenseDistance: "Cosine",
		Model: "fixture-embed-model:latest", Role: "embedding.default", PolicySchemaVersion: "fixture-v1",
	})
	if !errors.Is(err, ErrCollectionSchemaMismatch) {
		t.Fatalf("expected schema mismatch on wrong role, got %v", err)
	}
}

// TestEnsureCollectionRejectsMissingRoleMetadata: a sentinel missing a field
// the spec asserts was written by an older engine version; there is no in-place
// backfill — the collection is derived data, so the remediation is a deliberate
// drop-and-reindex.
func TestEnsureCollectionRejectsMissingRoleMetadata(t *testing.T) {
	doer := &capturingDoer{respond: func(req capturedReq) (int, string) {
		switch {
		case isCollectionInfoGet(req):
			return http.StatusOK, matchingDenseInfo(fixtureDenseSize, "Cosine")
		case isPointGet(req):
			return http.StatusOK, fmt.Sprintf(`{"result":{"payload":{"__aisearch_meta__":true,"model":"fixture-embed-model:latest","dense_size":%d,"policy_schema_version":"fixture-v1"}}}`, fixtureDenseSize)
		}
		return http.StatusOK, "{}"
	}}
	store := NewVectorStoreWithClient("http://q", "", "cli-health-commands", doer)
	err := store.EnsureCollection(context.Background(), CollectionSpec{
		DenseSize: fixtureDenseSize, DenseDistance: "Cosine",
		Model: "fixture-embed-model:latest", Role: "embedding.default", PolicySchemaVersion: "fixture-v1",
	})
	if !errors.Is(err, ErrCollectionSchemaMismatch) {
		t.Fatalf("expected schema mismatch on missing role metadata, got %v", err)
	}
	assertNoSentinelWrite(t, doer)
}

func TestEnsureCollectionSchemaGuardWrongPolicySchemaVersion(t *testing.T) {
	doer := &capturingDoer{respond: func(req capturedReq) (int, string) {
		switch {
		case isCollectionInfoGet(req):
			return http.StatusOK, matchingDenseInfo(fixtureDenseSize, "Cosine")
		case isPointGet(req):
			return http.StatusOK, fmt.Sprintf(`{"result":{"payload":{"__aisearch_meta__":true,"model":"fixture-embed-model:latest","role":"embedding.default","dense_size":%d,"policy_schema_version":"fixture-v0"}}}`, fixtureDenseSize)
		}
		return http.StatusOK, "{}"
	}}
	store := NewVectorStoreWithClient("http://q", "", "cli-health-commands", doer)
	err := store.EnsureCollection(context.Background(), CollectionSpec{
		DenseSize: fixtureDenseSize, DenseDistance: "Cosine",
		Model: "fixture-embed-model:latest", Role: "embedding.default", PolicySchemaVersion: "fixture-v1",
	})
	if !errors.Is(err, ErrCollectionSchemaMismatch) {
		t.Fatalf("expected schema mismatch on wrong policy schema version, got %v", err)
	}
}

func TestEnsureCollectionMatchingIsIdempotent(t *testing.T) {
	// WS1 case (b): a matching layout with a matching sentinel already present →
	// no error and no write at all (no create, no sentinel backfill).
	doer := &capturingDoer{respond: func(req capturedReq) (int, string) {
		switch {
		case isCollectionInfoGet(req):
			return http.StatusOK, matchingDenseInfo(fixtureDenseSize, "Cosine")
		case isPointGet(req):
			return http.StatusOK, fmt.Sprintf(`{"result":{"payload":{"__aisearch_meta__":true,"model":"fixture-embed-model:latest","role":"embedding.default","dense_size":%d,"policy_schema_version":"fixture-v1"}}}`, fixtureDenseSize)
		}
		return http.StatusOK, "{}"
	}}
	store := NewVectorStoreWithClient("http://q", "", "cli-health-commands", doer)
	if err := store.EnsureCollection(context.Background(), CollectionSpec{DenseSize: fixtureDenseSize, DenseDistance: "Cosine", Model: "fixture-embed-model:latest", Role: "embedding.default", PolicySchemaVersion: "fixture-v1"}); err != nil {
		t.Fatalf("matching collection + matching sentinel must be accepted, got %v", err)
	}
	for _, r := range doer.requests {
		if r.method == http.MethodPut {
			t.Fatalf("idempotent ensure must not PUT, saw %s %s", r.method, r.url)
		}
	}
}

// TestEnsureCollectionRejectsSentinelWhenAbsent: a layout-compatible collection
// with no meta sentinel was not created by this engine version. It is rejected
// with the mismatch error (drop-and-reindex remediation) — never backfilled,
// never deleted, never recreated.
func TestEnsureCollectionRejectsSentinelWhenAbsent(t *testing.T) {
	doer := &capturingDoer{respond: func(req capturedReq) (int, string) {
		switch {
		case isCollectionInfoGet(req):
			return http.StatusOK, matchingDenseInfo(fixtureDenseSize, "Cosine")
		case isPointGet(req):
			return http.StatusNotFound, "{}" // no sentinel — pre-engine collection
		}
		return http.StatusOK, "{}"
	}}
	store := NewVectorStoreWithClient("http://q", "", "cli-health-commands", doer)
	err := store.EnsureCollection(context.Background(), CollectionSpec{DenseSize: fixtureDenseSize, DenseDistance: "Cosine", Model: "fixture-embed-model:latest", Role: "embedding.default", PolicySchemaVersion: "fixture-v1"})
	if !errors.Is(err, ErrCollectionSchemaMismatch) {
		t.Fatalf("expected schema mismatch on missing sentinel, got %v", err)
	}

	for _, r := range doer.requests {
		// The guard must never remediate on its own: no delete, no recreate.
		if strings.Contains(r.url, "/points/delete") {
			t.Fatalf("guard must not delete, saw %s %s", r.method, r.url)
		}
		if r.method == http.MethodPut && strings.HasSuffix(strings.Split(r.url, "?")[0], "/collections/cli-health-commands") {
			t.Fatalf("guard must not recreate the collection, saw %s %s", r.method, r.url)
		}
	}
	assertNoSentinelWrite(t, doer)
}

// assertNoSentinelWrite fails if any request wrote a meta-sentinel point: the
// guard is read-only on existing collections (no backfill remains in-code).
func assertNoSentinelWrite(t *testing.T, doer *capturingDoer) {
	t.Helper()
	for _, r := range doer.requests {
		if !strings.Contains(r.url, "/points") {
			continue
		}
		pts, _ := r.body["points"].([]any)
		for _, p := range pts {
			m, _ := p.(map[string]any)
			pl, _ := m["payload"].(map[string]any)
			if pl == nil {
				continue
			}
			if _, ok := pl["__aisearch_meta__"]; ok {
				t.Fatalf("guard must not write a meta sentinel, saw %s %s", r.method, r.url)
			}
		}
	}
}

// TestEnsureCollectionNameMismatchErrors covers WS2: a non-empty spec.Name that
// disagrees with the store's collection is a loud error (not a silent
// mis-target), and the guard performs no GET/PUT against the wrong collection.
func TestEnsureCollectionNameMismatchErrors(t *testing.T) {
	doer := &capturingDoer{respond: func(req capturedReq) (int, string) {
		return http.StatusOK, "{}"
	}}
	store := NewVectorStoreWithClient("http://q", "", "cli-health-commands", doer)
	err := store.EnsureCollection(context.Background(), CollectionSpec{Name: "some-other-collection", DenseSize: fixtureDenseSize, DenseDistance: "Cosine"})
	if err == nil {
		t.Fatal("expected an error when spec.Name disagrees with the store collection")
	}
	if !strings.Contains(err.Error(), "some-other-collection") || !strings.Contains(err.Error(), "cli-health-commands") {
		t.Fatalf("error must name both collections, got %v", err)
	}
	if len(doer.requests) != 0 {
		t.Fatalf("name-mismatch must fail before any HTTP call, saw %d requests", len(doer.requests))
	}
}

// TestEnsureCollectionMatchingNamePasses covers WS2: a spec.Name equal to the
// store's collection is accepted (a useful intent cross-check, not an error).
func TestEnsureCollectionMatchingNamePasses(t *testing.T) {
	doer := &capturingDoer{respond: func(req capturedReq) (int, string) {
		if isCollectionInfoGet(req) {
			return http.StatusNotFound, "{}" // force creation
		}
		return http.StatusOK, "{}"
	}}
	store := NewVectorStoreWithClient("http://q", "", "cli-health-commands", doer)
	if err := store.EnsureCollection(context.Background(), CollectionSpec{Name: "cli-health-commands", DenseSize: fixtureDenseSize, DenseDistance: "Cosine", Model: "fixture-embed-model:latest", Role: "embedding.default", PolicySchemaVersion: "fixture-v1"}); err != nil {
		t.Fatalf("matching spec.Name must be accepted, got %v", err)
	}
}

func TestEnsureCollectionCreatesAndWritesSentinel(t *testing.T) {
	doer := &capturingDoer{respond: func(req capturedReq) (int, string) {
		if isCollectionInfoGet(req) {
			return http.StatusNotFound, "{}" // force creation
		}
		return http.StatusOK, "{}"
	}}
	store := NewVectorStoreWithClient("http://q", "", "cli-health-commands", doer)
	if err := store.EnsureCollection(context.Background(), CollectionSpec{DenseSize: fixtureDenseSize, DenseDistance: "Cosine", Model: "fixture-embed-model:latest", Role: "embedding.default", PolicySchemaVersion: "fixture-v1"}); err != nil {
		t.Fatal(err)
	}
	var sawCreate, sawSentinel bool
	for _, r := range doer.requests {
		if r.method == http.MethodPut && strings.HasSuffix(strings.Split(r.url, "?")[0], "/collections/cli-health-commands") {
			sawCreate = true
		}
		if strings.Contains(r.url, "/points") {
			pts, _ := r.body["points"].([]any)
			for _, p := range pts {
				m, _ := p.(map[string]any)
				pl, _ := m["payload"].(map[string]any)
				if pl != nil {
					if _, ok := pl["__aisearch_meta__"]; ok {
						sawSentinel = true
						if pl["model"] != "fixture-embed-model:latest" {
							t.Fatalf("sentinel must record the model, got %v", pl["model"])
						}
						if pl["role"] != "embedding.default" {
							t.Fatalf("sentinel must record the role, got %v", pl["role"])
						}
						if pl["policy_schema_version"] != "fixture-v1" {
							t.Fatalf("sentinel must record policy schema version, got %v", pl["policy_schema_version"])
						}
					}
				}
			}
		}
	}
	if !sawCreate {
		t.Fatal("expected collection create PUT")
	}
	if !sawSentinel {
		t.Fatal("expected meta sentinel upsert after create")
	}
}

func TestScrollIDsExcludesMetaSentinel(t *testing.T) {
	doer := &capturingDoer{respond: func(req capturedReq) (int, string) {
		// One real chunk + the meta sentinel; sentinel must not appear in the
		// reconciler's view, so it is never treated as a ghost and deleted.
		return http.StatusOK, `{"result":{"points":[
			{"id":"real-1","payload":{"payload_hash":"sha256:aa","source_id":"cmd-1","source_hash":"sha256:bb","chunk_total":1}},
			{"id":"meta-1","payload":{"__aisearch_meta__":true,"model":"fixture-embed-model:latest"}}
		],"next_page_offset":null}}`
	}}
	store := NewVectorStoreWithClient("http://q", "", "cli-health-commands", doer)
	got, err := store.ScrollIDs(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("expected only the real point, got %d: %v", len(got), got)
	}
	if _, ok := got["meta-1"]; ok {
		t.Fatal("meta sentinel must be excluded from ScrollIDs (else reconciler ghost-deletes it)")
	}
	if _, ok := got["real-1"]; !ok {
		t.Fatal("real point must be present")
	}
}

func TestQueryExcludesMetaSentinel(t *testing.T) {
	doer := &capturingDoer{respond: func(req capturedReq) (int, string) {
		return http.StatusOK, `{"result":{"points":[
			{"id":"meta-1","score":0.99,"payload":{"__aisearch_meta__":true}},
			{"id":"real-1","score":0.80,"payload":{"body":"x"}}
		]}}`
	}}
	store := NewVectorStoreWithClient("http://q", "", "cli-health-commands", doer)
	got, err := store.Query(context.Background(), HybridQuery{Dense: []float64{0.1}, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "real-1" {
		t.Fatalf("meta sentinel must never surface as a hit, got %+v", got)
	}
}

// TestCountPointsExcludesMetaSentinel verifies the count request carries a
// must_not filter on metaMarkerKey so the meta sentinel is not counted as a
// real point (the off-by-one the review found: live reported N, real = N-1).
func TestCountPointsExcludesMetaSentinel(t *testing.T) {
	var sawMustNot bool
	doer := &capturingDoer{respond: func(req capturedReq) (int, string) {
		if strings.HasSuffix(req.url, "/points/count") {
			filter, _ := req.body["filter"].(map[string]any)
			mustNot, _ := filter["must_not"].([]any)
			for _, clause := range mustNot {
				m, _ := clause.(map[string]any)
				if m["key"] == metaMarkerKey {
					sawMustNot = true
				}
			}
			// Qdrant returns the post-filter count (sentinel excluded).
			return http.StatusOK, `{"result":{"count":2235}}`
		}
		return http.StatusOK, "{}"
	}}
	store := NewVectorStoreWithClient("http://q", "", "cli-health-commands", doer)
	got, err := store.CountPoints(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !sawMustNot {
		t.Fatalf("count request must exclude the meta sentinel via must_not on %q, body=%v", metaMarkerKey, doer.last().body)
	}
	if got != 2235 {
		t.Fatalf("expected count to reflect real points only, got %d", got)
	}
}

func TestQueryHybridFusionShape(t *testing.T) {
	doer := &capturingDoer{respond: func(req capturedReq) (int, string) {
		return http.StatusOK, `{"result":{"points":[{"id":"p1","score":0.83,"payload":{"body":"x"}}]}}`
	}}
	store := NewVectorStoreWithClient("http://q", "", "vrooli-docs", doer)
	got, err := store.Query(context.Background(), HybridQuery{
		Dense:         []float64{0.1, 0.2},
		Sparse:        &SparseVector{Indices: []uint32{1, 2}, Values: []float32{1, 1}},
		Fusion:        "rrf",
		Limit:         5,
		PrefetchLimit: 50,
		Filter:        &QueryFilter{Must: []FieldMatch{{Key: "scope", Value: "project"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "p1" || got[0].Score != 0.83 {
		t.Fatalf("unexpected query result: %+v", got)
	}
	req := doer.last()
	if !strings.HasSuffix(req.url, "/collections/vrooli-docs/points/query") {
		t.Fatalf("expected Query API endpoint, got %s", req.url)
	}
	prefetch, ok := req.body["prefetch"].([]any)
	if !ok || len(prefetch) != 2 {
		t.Fatalf("expected 2 prefetch legs, got %v", req.body["prefetch"])
	}
	usings := map[string]bool{}
	for _, leg := range prefetch {
		m := leg.(map[string]any)
		usings[m["using"].(string)] = true
		if _, ok := m["filter"].(map[string]any); !ok {
			t.Fatalf("each prefetch leg must carry the scope filter, got %v", m)
		}
	}
	if !usings["dense"] || !usings["sparse"] {
		t.Fatalf("expected dense+sparse legs, got %v", usings)
	}
	q, _ := req.body["query"].(map[string]any)
	if q["fusion"] != "rrf" {
		t.Fatalf("expected rrf fusion, got %v", req.body["query"])
	}
}

func TestQueryDenseOnlyShape(t *testing.T) {
	doer := &capturingDoer{respond: func(req capturedReq) (int, string) {
		return http.StatusOK, `{"result":{"points":[]}}`
	}}
	store := NewVectorStoreWithClient("http://q", "", "cli-health-commands", doer)
	if _, err := store.Query(context.Background(), HybridQuery{Dense: []float64{0.1}, Limit: 10}); err != nil {
		t.Fatal(err)
	}
	req := doer.last()
	if _, ok := req.body["prefetch"]; ok {
		t.Fatalf("dense-only query must not use prefetch fusion: %v", req.body)
	}
	if req.body["using"] != "dense" {
		t.Fatalf("dense-only query must address the named dense vector, got %v", req.body["using"])
	}
}

func TestUpsertNamedVectorShape(t *testing.T) {
	doer := &capturingDoer{}
	store := NewVectorStoreWithClient("http://q", "", "vrooli-docs", doer)
	err := store.Upsert(context.Background(), Point{
		ID:      "p1",
		Dense:   []float64{0.1, 0.2},
		Sparse:  &SparseVector{Indices: []uint32{5}, Values: []float32{2}},
		Payload: map[string]any{"body": "hello"},
	})
	if err != nil {
		t.Fatal(err)
	}
	req := doer.last()
	points, _ := req.body["points"].([]any)
	if len(points) != 1 {
		t.Fatalf("expected one point, got %v", req.body)
	}
	p := points[0].(map[string]any)
	vec, ok := p["vector"].(map[string]any)
	if !ok {
		t.Fatalf("vector must be a named-vector map, got %T", p["vector"])
	}
	if _, ok := vec["dense"]; !ok {
		t.Fatal("missing named dense vector")
	}
	if _, ok := vec["sparse"]; !ok {
		t.Fatal("missing named sparse vector")
	}
}

func TestSetPayloadShape(t *testing.T) {
	doer := &capturingDoer{}
	store := NewVectorStoreWithClient("http://q", "", "vrooli-docs", doer)
	if err := store.SetPayload(context.Background(), "p1", map[string]any{"source_hash": "sha256:abc"}); err != nil {
		t.Fatal(err)
	}
	req := doer.last()
	if !strings.Contains(req.url, "/points/payload") {
		t.Fatalf("expected set-payload endpoint, got %s", req.url)
	}
	pts, _ := req.body["points"].([]any)
	if len(pts) != 1 || pts[0] != "p1" {
		t.Fatalf("expected point id p1, got %v", req.body["points"])
	}
}
