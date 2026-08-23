package aisearch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// MaxDeleteBatch caps the point IDs sent in one /points/delete request.
const MaxDeleteBatch = 256

// scrollPageLimit is the per-request page size used by ScrollIDs.
const scrollPageLimit = 256

// denseVectorName / sparseVectorName are the named-vector keys. Every consumer
// — even dense-only ones — uses a named "dense" vector so hybrid and dense-only
// collections differ only by the presence of the sparse vector (Phase 0 D5).
const (
	denseVectorName  = "dense"
	sparseVectorName = "sparse"
)

// engineSchemaVersion is the on-disk schema generation recorded on the meta
// sentinel. Bump it when a change to the point/payload layout requires a
// deliberate re-index of every existing collection.
const engineSchemaVersion = 1

// Meta-sentinel reserved keys. A single sentinel point per collection records
// the embedding model + vector layout the collection was created with, so a
// future model/dimension swap fails loudly instead of silently embedding into
// an incompatible collection. The sentinel is excluded from search (Query) and
// from reconcile drift (ScrollIDs) by the presence of metaMarkerKey, so it is
// invisible to consumers and never ghost-deleted.
const (
	metaMarkerKey        = "__aisearch_meta__" // bool true; marks the sentinel
	metaModelKey         = "model"
	metaRoleKey          = "role"
	metaDenseSizeKey     = "dense_size"
	metaDenseDistanceKey = "dense_distance"
	metaPolicySchemaKey  = "policy_schema_version"
	metaSchemaVersionKey = "engine_schema_version"
	// metaIDPrefix derives the deterministic sentinel point ID from the
	// collection name; reserved so it can never collide with a real source key.
	metaIDPrefix = "__aisearch_meta__:"
)

// ErrCollectionSchemaMismatch is the sentinel wrapped by every schema-guard
// failure; callers test it with errors.Is. The wrapping
// *CollectionSchemaMismatchError carries the offending field + remediation.
var ErrCollectionSchemaMismatch = errors.New("aisearch: collection schema mismatch")

// CollectionSchemaMismatchError reports that a pre-existing collection's layout
// disagrees with the requested CollectionSpec. It never auto-drops — the fix is
// operator-initiated (data loss must be deliberate).
type CollectionSchemaMismatchError struct {
	Collection string
	Field      string
	Want       string
	Got        string
}

func (e *CollectionSchemaMismatchError) Error() string {
	return fmt.Sprintf(
		"aisearch: collection %q schema mismatch on %s (want %q, got %q); drop the collection and reindex (this is a deliberate, data-losing operation)",
		e.Collection, e.Field, e.Want, e.Got,
	)
}

func (e *CollectionSchemaMismatchError) Unwrap() error { return ErrCollectionSchemaMismatch }

// httpDoer is the minimal HTTP surface the store needs; injectable for tests.
type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// qdrantVectorStore is the Qdrant-backed VectorStore generalized to named
// dense+sparse vectors and the server-side hybrid Query API (RRF fusion).
type qdrantVectorStore struct {
	baseURL    string
	apiKey     string
	collection string
	client     httpDoer
}

// NewVectorStore creates a Qdrant-backed VectorStore for one collection.
func NewVectorStore(baseURL, apiKey, collection string) VectorStore {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = DefaultQdrantURL
	}
	return &qdrantVectorStore{
		baseURL:    baseURL,
		apiKey:     apiKey,
		collection: collection,
		client:     &http.Client{Timeout: 30 * time.Second},
	}
}

// NewVectorStoreWithClient injects the HTTP client (tests).
func NewVectorStoreWithClient(baseURL, apiKey, collection string, client httpDoer) VectorStore {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = DefaultQdrantURL
	}
	return &qdrantVectorStore{baseURL: baseURL, apiKey: apiKey, collection: collection, client: client}
}

func (v *qdrantVectorStore) do(req *http.Request) (*http.Response, error) {
	if key := strings.TrimSpace(v.apiKey); key != "" {
		req.Header.Set("api-key", key)
	}
	resp, err := v.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("qdrant request failed: %w", err)
	}
	return resp, nil
}

func (v *qdrantVectorStore) endpoint(suffix string) (string, error) {
	base := strings.TrimRight(strings.TrimSpace(v.baseURL), "/")
	if base == "" {
		return "", fmt.Errorf("qdrant base url is required")
	}
	if strings.TrimSpace(v.collection) == "" {
		return "", fmt.Errorf("collection is required")
	}
	u, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("invalid qdrant url: %w", err)
	}
	u.Path = fmt.Sprintf("%s/collections/%s%s", strings.TrimRight(u.Path, "/"), v.collection, suffix)
	return u.String(), nil
}

// --- collection creation ----------------------------------------------------

type namedVectorParams struct {
	Size     int    `json:"size"`
	Distance string `json:"distance"`
	OnDisk   bool   `json:"on_disk,omitempty"`
}

type sparseIndexParams struct {
	OnDisk bool `json:"on_disk,omitempty"`
}

type sparseVectorParams struct {
	Modifier string             `json:"modifier,omitempty"`
	Index    *sparseIndexParams `json:"index,omitempty"`
}

type hnswConfig struct {
	M                 int  `json:"m,omitempty"`
	EFConstruct       int  `json:"ef_construct,omitempty"`
	FullScanThreshold int  `json:"full_scan_threshold,omitempty"`
	OnDisk            bool `json:"on_disk,omitempty"`
}

type optimizerConfig struct {
	IndexingThreshold      int `json:"indexing_threshold,omitempty"`
	MaxOptimizationThreads int `json:"max_optimization_threads,omitempty"`
}

type scalarQuantization struct {
	Type      string  `json:"type"`
	Quantile  float64 `json:"quantile,omitempty"`
	AlwaysRAM bool    `json:"always_ram,omitempty"`
}

type quantizationConfig struct {
	Scalar scalarQuantization `json:"scalar"`
}

type createCollectionRequest struct {
	Vectors            map[string]namedVectorParams  `json:"vectors"`
	SparseVectors      map[string]sparseVectorParams `json:"sparse_vectors,omitempty"`
	OnDiskPayload      bool                          `json:"on_disk_payload,omitempty"`
	HNSWConfig         *hnswConfig                   `json:"hnsw_config,omitempty"`
	OptimizersConfig   *optimizerConfig              `json:"optimizers_config,omitempty"`
	QuantizationConfig *quantizationConfig           `json:"quantization_config,omitempty"`
}

// --- schema inspection ------------------------------------------------------

// collectionInfoResponse decodes the parts of GET /collections/<name> the
// schema guard needs: the dense vector params (named-map or legacy single) and
// the sparse-vector map.
type collectionInfoResponse struct {
	Result struct {
		Config struct {
			Params struct {
				Vectors       json.RawMessage            `json:"vectors"`
				SparseVectors map[string]json.RawMessage `json:"sparse_vectors"`
			} `json:"params"`
		} `json:"config"`
	} `json:"result"`
}

// collectionLayout is the normalized view of an existing collection's vector
// schema used by checkLayout.
type collectionLayout struct {
	hasDense      bool
	denseSize     int
	denseDistance string
	unnamedDense  bool // a legacy single unnamed vector is present
	hasSparse     bool
}

// inspect fetches the collection and parses its vector layout. found=false
// (no error) when the collection does not exist (404).
func (v *qdrantVectorStore) inspect(ctx context.Context) (collectionLayout, bool, error) {
	endpoint, err := v.endpoint("")
	if err != nil {
		return collectionLayout{}, false, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return collectionLayout{}, false, fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := v.do(req)
	if err != nil {
		return collectionLayout{}, false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return collectionLayout{}, false, nil
	}
	if resp.StatusCode != http.StatusOK {
		return collectionLayout{}, false, fmt.Errorf("qdrant collection info returned status %d", resp.StatusCode)
	}
	var decoded collectionInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return collectionLayout{}, false, fmt.Errorf("failed to decode collection info: %w", err)
	}
	layout := parseVectorLayout(decoded.Result.Config.Params.Vectors)
	layout.hasSparse = func() bool {
		_, ok := decoded.Result.Config.Params.SparseVectors[sparseVectorName]
		return ok
	}()
	return layout, true, nil
}

// parseVectorLayout normalizes Qdrant's vectors field, which is either a
// named-vector map ({"dense":{"size":…}}) or — for a legacy collection — a
// single unnamed vector object ({"size":…,"distance":…}).
func parseVectorLayout(raw json.RawMessage) collectionLayout {
	var layout collectionLayout
	if len(raw) == 0 {
		return layout
	}
	// Try the named-vector map first.
	var named map[string]namedVectorParams
	if err := json.Unmarshal(raw, &named); err == nil && len(named) > 0 {
		if dp, ok := named[denseVectorName]; ok && dp.Size > 0 {
			layout.hasDense = true
			layout.denseSize = dp.Size
			layout.denseDistance = dp.Distance
		}
		return layout
	}
	// Fall back to a single unnamed vector (the legacy trap).
	var single namedVectorParams
	if err := json.Unmarshal(raw, &single); err == nil && single.Size > 0 {
		layout.unnamedDense = true
		layout.denseSize = single.Size
		layout.denseDistance = single.Distance
	}
	return layout
}

// --- meta sentinel ----------------------------------------------------------

// metaPointID is the deterministic sentinel point ID for this collection.
func (v *qdrantVectorStore) metaPointID() string {
	return PointIDFor(metaIDPrefix, v.collection, 0, 1)
}

// writeMetaSentinel upserts the per-collection meta sentinel recording the
// embedding model + vector layout. The dense vector is a unit placeholder (so a
// Cosine collection never sees a zero vector); the sentinel is excluded from
// search and reconcile by metaMarkerKey.
func (v *qdrantVectorStore) writeMetaSentinel(ctx context.Context, spec CollectionSpec, size int, distance string) error {
	dense := make([]float64, size)
	if size > 0 {
		dense[0] = 1
	}
	payload := map[string]any{
		metaMarkerKey:        true,
		metaModelKey:         strings.TrimSpace(spec.Model),
		metaRoleKey:          strings.TrimSpace(spec.Role),
		metaDenseSizeKey:     size,
		metaDenseDistanceKey: distance,
		metaPolicySchemaKey:  strings.TrimSpace(spec.PolicySchemaVersion),
		metaSchemaVersionKey: engineSchemaVersion,
	}
	return v.Upsert(ctx, Point{ID: v.metaPointID(), Dense: dense, Payload: payload})
}

type collectionMeta struct {
	Model               string
	Role                string
	DenseSize           int
	DenseDistance       string
	PolicySchemaVersion string
	EngineSchemaVersion int
}

type pointRetrieveResponse struct {
	Result struct {
		Payload map[string]any `json:"payload"`
	} `json:"result"`
}

// readMeta retrieves the sentinel's recorded embedding metadata. ok=false when
// no sentinel exists (a pre-guard collection).
func (v *qdrantVectorStore) readMeta(ctx context.Context) (collectionMeta, bool, error) {
	endpoint, err := v.endpoint("/points/" + url.PathEscape(v.metaPointID()))
	if err != nil {
		return collectionMeta{}, false, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return collectionMeta{}, false, fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := v.do(req)
	if err != nil {
		return collectionMeta{}, false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return collectionMeta{}, false, nil
	}
	if resp.StatusCode != http.StatusOK {
		return collectionMeta{}, false, fmt.Errorf("qdrant point retrieve returned status %d", resp.StatusCode)
	}
	var decoded pointRetrieveResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return collectionMeta{}, false, fmt.Errorf("failed to decode point retrieve: %w", err)
	}
	if decoded.Result.Payload == nil {
		return collectionMeta{}, false, nil
	}
	meta := collectionMeta{
		Model:               stringPayload(decoded.Result.Payload, metaModelKey),
		Role:                stringPayload(decoded.Result.Payload, metaRoleKey),
		DenseSize:           intPayload(decoded.Result.Payload, metaDenseSizeKey),
		DenseDistance:       stringPayload(decoded.Result.Payload, metaDenseDistanceKey),
		PolicySchemaVersion: stringPayload(decoded.Result.Payload, metaPolicySchemaKey),
		EngineSchemaVersion: intPayload(decoded.Result.Payload, metaSchemaVersionKey),
	}
	return meta, true, nil
}

func stringPayload(payload map[string]any, key string) string {
	value, _ := payload[key].(string)
	return strings.TrimSpace(value)
}

func intPayload(payload map[string]any, key string) int {
	switch value := payload[key].(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	case json.Number:
		i, _ := value.Int64()
		return int(i)
	default:
		return 0
	}
}

// EnsureCollection creates the collection (named dense vector, optional named
// sparse vector with idf modifier) if it does not already exist. Idempotent.
//
// When the collection already exists, its on-disk layout is inspected and
// compared against spec: a named "dense" vector of the right size/distance,
// sparse presence matching spec.Sparse, and — when both the sentinel and
// spec.Model are present — the recorded embedding model. Any disagreement
// returns a *CollectionSchemaMismatchError (errors.Is ErrCollectionSchemaMismatch)
// rather than silently upserting named-vector points into an incompatible
// collection. The guard never auto-drops; remediation is operator-initiated.
func (v *qdrantVectorStore) EnsureCollection(ctx context.Context, spec CollectionSpec) error {
	if err := validateStorageProfile(spec.Storage); err != nil {
		return err
	}
	// CollectionSpec.Name is an optional cross-check; the store (NewVectorStore)
	// owns the authoritative collection name. A non-empty disagreement is a
	// mis-target (the adopter set spec.Name expecting it to choose the
	// collection) — fail loudly at boot rather than silently operating on
	// v.collection.
	if name := strings.TrimSpace(spec.Name); name != "" && name != v.collection {
		return fmt.Errorf("aisearch: CollectionSpec.Name %q != store collection %q; the store (NewVectorStore) owns the collection name", spec.Name, v.collection)
	}

	size := spec.DenseSize
	if size <= 0 {
		size = DefaultVectorSize
	}
	distance := strings.TrimSpace(spec.DenseDistance)
	if distance == "" {
		distance = DefaultDenseDistance
	}

	layout, found, err := v.inspect(ctx)
	if err != nil {
		return err
	}
	if found {
		return v.checkLayout(ctx, spec, size, distance, layout)
	}

	endpoint, err := v.endpoint("")
	if err != nil {
		return err
	}
	create := createCollectionRequest{
		Vectors: map[string]namedVectorParams{
			denseVectorName: {Size: size, Distance: distance, OnDisk: spec.Storage.OnDiskVectors},
		},
		OnDiskPayload: spec.Storage.OnDiskPayload,
	}
	if spec.Storage.OnDiskHNSW || spec.Storage.HNSWM > 0 || spec.Storage.HNSWEFConstruct > 0 || spec.Storage.FullScanThreshold > 0 {
		create.HNSWConfig = &hnswConfig{
			M:                 spec.Storage.HNSWM,
			EFConstruct:       spec.Storage.HNSWEFConstruct,
			FullScanThreshold: spec.Storage.FullScanThreshold,
			OnDisk:            spec.Storage.OnDiskHNSW,
		}
	}
	if spec.Storage.IndexingThreshold > 0 || spec.Storage.MaxOptimizationThreads > 0 {
		create.OptimizersConfig = &optimizerConfig{
			IndexingThreshold:      spec.Storage.IndexingThreshold,
			MaxOptimizationThreads: spec.Storage.MaxOptimizationThreads,
		}
	}
	if spec.Storage.ScalarQuantization {
		create.QuantizationConfig = &quantizationConfig{Scalar: scalarQuantization{
			Type:      "int8",
			Quantile:  spec.Storage.Quantile,
			AlwaysRAM: spec.Storage.QuantizationAlwaysRAM,
		}}
	}
	if spec.Sparse {
		modifier := strings.TrimSpace(spec.SparseModifier)
		if modifier == "" {
			modifier = DefaultSparseModifier
		}
		params := sparseVectorParams{Modifier: modifier}
		if spec.Storage.OnDiskSparse {
			params.Index = &sparseIndexParams{OnDisk: true}
		}
		create.SparseVectors = map[string]sparseVectorParams{sparseVectorName: params}
	}
	body, err := json.Marshal(create)
	if err != nil {
		return fmt.Errorf("failed to marshal create request: %w", err)
	}
	putReq, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	putReq.Header.Set("Content-Type", "application/json")
	putResp, err := v.do(putReq)
	if err != nil {
		return err
	}
	defer putResp.Body.Close()
	if putResp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(putResp.Body)
		return fmt.Errorf("qdrant create collection returned status %d: %s", putResp.StatusCode, strings.TrimSpace(string(raw)))
	}
	// Record the model + layout on the meta sentinel so a future model/dimension
	// swap is caught by the guard above instead of corrupting this collection.
	return v.writeMetaSentinel(ctx, spec, size, distance)
}

func validateStorageProfile(profile StorageProfile) error {
	values := []struct {
		name  string
		value int
	}{
		{name: "hnsw_m", value: profile.HNSWM},
		{name: "hnsw_ef_construct", value: profile.HNSWEFConstruct},
		{name: "full_scan_threshold", value: profile.FullScanThreshold},
		{name: "indexing_threshold", value: profile.IndexingThreshold},
		{name: "max_optimization_threads", value: profile.MaxOptimizationThreads},
		{name: "upsert_batch_size", value: profile.UpsertBatchSize},
	}
	for _, entry := range values {
		if entry.value < 0 {
			return fmt.Errorf("aisearch: storage profile %s must be non-negative", entry.name)
		}
	}
	if profile.Quantile < 0 || profile.Quantile > 1 {
		return fmt.Errorf("aisearch: storage profile quantile must be within 0..1")
	}
	return nil
}

// checkLayout compares a discovered layout against the requested spec and
// returns a *CollectionSchemaMismatchError on the first disagreement. A
// collection whose sentinel is absent (or missing a field the spec asserts)
// was not created by this engine version and is rejected the same way: vector
// stores are derived data, so the remediation is a deliberate drop-and-reindex,
// never an in-place backfill.
func (v *qdrantVectorStore) checkLayout(ctx context.Context, spec CollectionSpec, size int, distance string, layout collectionLayout) error {
	mismatch := func(field, want, got string) error {
		return &CollectionSchemaMismatchError{Collection: v.collection, Field: field, Want: want, Got: got}
	}
	if layout.unnamedDense {
		return mismatch("dense vector", "named \""+denseVectorName+"\" vector", "legacy unnamed vector")
	}
	if !layout.hasDense {
		return mismatch("dense vector", "named \""+denseVectorName+"\" vector", "absent")
	}
	if layout.denseSize != size {
		return mismatch("dense.size", fmt.Sprintf("%d", size), fmt.Sprintf("%d", layout.denseSize))
	}
	if layout.denseDistance != "" && !strings.EqualFold(layout.denseDistance, distance) {
		return mismatch("dense.distance", distance, layout.denseDistance)
	}
	if layout.hasSparse != spec.Sparse {
		return mismatch("sparse vector", fmt.Sprintf("%t", spec.Sparse), fmt.Sprintf("%t", layout.hasSparse))
	}
	meta, ok, err := v.readMeta(ctx)
	if err != nil {
		return err
	}
	if !ok {
		return mismatch("meta sentinel", "present (collection created by this engine)", "absent")
	}
	if want := strings.TrimSpace(spec.Model); want != "" && !sameModelRef(meta.Model, want) {
		return mismatch("embedding model", want, meta.Model)
	}
	if want := strings.TrimSpace(spec.Role); want != "" && meta.Role != want {
		return mismatch("embedding role", want, meta.Role)
	}
	if meta.DenseSize > 0 && meta.DenseSize != size {
		return mismatch("metadata dense_size", fmt.Sprintf("%d", size), fmt.Sprintf("%d", meta.DenseSize))
	}
	if meta.DenseDistance != "" && !strings.EqualFold(meta.DenseDistance, distance) {
		return mismatch("metadata dense_distance", distance, meta.DenseDistance)
	}
	if want := strings.TrimSpace(spec.PolicySchemaVersion); want != "" && meta.PolicySchemaVersion != want {
		return mismatch("policy schema version", want, meta.PolicySchemaVersion)
	}
	if meta.EngineSchemaVersion > 0 && meta.EngineSchemaVersion != engineSchemaVersion {
		return mismatch("engine schema version", fmt.Sprintf("%d", engineSchemaVersion), fmt.Sprintf("%d", meta.EngineSchemaVersion))
	}
	return nil
}

func sameModelRef(a, b string) bool {
	a = canonicalModelRef(a)
	b = canonicalModelRef(b)
	return a != "" && strings.EqualFold(a, b)
}

func canonicalModelRef(ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ""
	}
	if strings.Contains(ref, ":") {
		return ref
	}
	return ref + ":latest"
}

// --- upsert -----------------------------------------------------------------

type sparseVectorJSON struct {
	Indices []uint32  `json:"indices"`
	Values  []float32 `json:"values"`
}

// Upsert inserts or updates one point. The vector is always a named-vector map
// ({"dense": [...]}); when Sparse is set the named "sparse" vector is added.
func (v *qdrantVectorStore) Upsert(ctx context.Context, point Point) error {
	return v.UpsertBatch(ctx, []Point{point}, 1)
}

// UpsertBatch writes points in bounded requests. batchSize is clamped to
// MaxSourcePageSize so a malformed operator value cannot create an unbounded
// payload. The method is additive through BatchVectorStore; existing adopters
// that call Upsert retain byte-equivalent one-point writes.
func (v *qdrantVectorStore) UpsertBatch(ctx context.Context, points []Point, batchSize int) error {
	if len(points) == 0 {
		return nil
	}
	if batchSize <= 0 {
		batchSize = DefaultSourcePageSize
	}
	if batchSize > MaxSourcePageSize {
		batchSize = MaxSourcePageSize
	}
	endpoint, err := v.endpoint("/points")
	if err != nil {
		return err
	}
	endpoint = withWait(endpoint)
	for start := 0; start < len(points); start += batchSize {
		end := start + batchSize
		if end > len(points) {
			end = len(points)
		}
		requestPoints := make([]map[string]any, 0, end-start)
		for _, point := range points[start:end] {
			id := strings.TrimSpace(point.ID)
			if id == "" {
				return fmt.Errorf("point id is required")
			}
			vec := map[string]any{denseVectorName: point.Dense}
			if point.Sparse != nil {
				vec[sparseVectorName] = sparseVectorJSON{Indices: point.Sparse.Indices, Values: point.Sparse.Values}
			}
			requestPoints = append(requestPoints, map[string]any{"id": id, "vector": vec, "payload": point.Payload})
		}
		if err := v.writeJSON(ctx, http.MethodPut, endpoint, map[string]any{"points": requestPoints}, "upsert"); err != nil {
			return err
		}
	}
	return nil
}

type setPayloadRequest struct {
	Payload map[string]any `json:"payload"`
	Points  []string       `json:"points"`
}

// SetPayload refreshes a point's payload without touching its vectors.
func (v *qdrantVectorStore) SetPayload(ctx context.Context, id string, payload map[string]any) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("point id is required")
	}
	endpoint, err := v.endpoint("/points/payload")
	if err != nil {
		return err
	}
	endpoint = withWait(endpoint)
	return v.writeJSON(ctx, http.MethodPost, endpoint, setPayloadRequest{Payload: payload, Points: []string{id}}, "set payload")
}

// --- query (hybrid / dense-only) --------------------------------------------

type prefetchLeg struct {
	Query  any    `json:"query"`
	Using  string `json:"using"`
	Limit  int    `json:"limit"`
	Filter any    `json:"filter,omitempty"`
}

type queryResponse struct {
	Result struct {
		Points []struct {
			ID      any            `json:"id"`
			Score   float64        `json:"score"`
			Payload map[string]any `json:"payload"`
		} `json:"points"`
	} `json:"result"`
}

// Query runs the Qdrant Query API. With Sparse set and Fusion=="rrf" it issues
// two prefetch legs (dense + sparse), each scoped by Filter, fused server-side
// with RRF; otherwise it issues a single dense-only query (the fallback leg).
func (v *qdrantVectorStore) Query(ctx context.Context, q HybridQuery) ([]SearchResult, error) {
	endpoint, err := v.endpoint("/points/query")
	if err != nil {
		return nil, err
	}
	limit := q.Limit
	if limit <= 0 {
		limit = 10
	}
	filter := buildFilter(q.Filter)

	var body map[string]any
	if q.Sparse != nil && strings.EqualFold(q.Fusion, "rrf") {
		prefetchLimit := q.PrefetchLimit
		if prefetchLimit <= 0 {
			prefetchLimit = 50
		}
		body = map[string]any{
			"prefetch": []prefetchLeg{
				{Query: q.Dense, Using: denseVectorName, Limit: prefetchLimit, Filter: filter},
				{Query: sparseVectorJSON{Indices: q.Sparse.Indices, Values: q.Sparse.Values}, Using: sparseVectorName, Limit: prefetchLimit, Filter: filter},
			},
			"query":        map[string]any{"fusion": "rrf"},
			"limit":        limit,
			"with_payload": true,
		}
	} else {
		body = map[string]any{
			"query":        q.Dense,
			"using":        denseVectorName,
			"limit":        limit,
			"with_payload": true,
		}
		if filter != nil {
			body["filter"] = filter
		}
		if q.ScoreThreshold > 0 {
			body["score_threshold"] = q.ScoreThreshold
		}
	}

	raw, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := v.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("qdrant query returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var decoded queryResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("failed to decode query response: %w", err)
	}
	out := make([]SearchResult, 0, len(decoded.Result.Points))
	for _, p := range decoded.Result.Points {
		if _, isMeta := p.Payload[metaMarkerKey]; isMeta {
			continue // never surface the schema sentinel as a search hit
		}
		out = append(out, SearchResult{ID: stringifyID(p.ID), Score: p.Score, Payload: p.Payload})
	}
	return out, nil
}

// buildFilter renders a QueryFilter into Qdrant's must/match JSON. Nil/empty
// filters render as nil (no scoping).
func buildFilter(f *QueryFilter) map[string]any {
	if f == nil || len(f.Must) == 0 {
		return nil
	}
	must := make([]map[string]any, 0, len(f.Must))
	for _, m := range f.Must {
		if len(m.AnyOf) > 0 {
			must = append(must, map[string]any{"key": m.Key, "match": map[string]any{"any": m.AnyOf}})
			continue
		}
		must = append(must, map[string]any{"key": m.Key, "match": map[string]any{"value": m.Value}})
	}
	return map[string]any{"must": must}
}

// --- count / scroll / delete / availability ---------------------------------

type countResponse struct {
	Result struct {
		Count int `json:"count"`
	} `json:"result"`
}

// CountPoints returns the exact number of real points in the collection. The
// meta sentinel (metaMarkerKey) is excluded via a must_not filter so the count
// matches what Query/ScrollIDs surface (which post-filter the sentinel out);
// without this the reported count is off by one.
func (v *qdrantVectorStore) CountPoints(ctx context.Context) (int, error) {
	endpoint, err := v.endpoint("/points/count")
	if err != nil {
		return 0, err
	}
	body, err := json.Marshal(map[string]any{
		"exact": true,
		"filter": map[string]any{
			"must_not": []map[string]any{
				{"key": metaMarkerKey, "match": map[string]any{"value": true}},
			},
		},
	})
	if err != nil {
		return 0, fmt.Errorf("failed to marshal count request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("failed to create count request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := v.do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("qdrant count returned status %d", resp.StatusCode)
	}
	var decoded countResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return 0, fmt.Errorf("failed to decode count response: %w", err)
	}
	return decoded.Result.Count, nil
}

type scrollRequest struct {
	Limit       int      `json:"limit"`
	WithPayload []string `json:"with_payload"`
	WithVectors bool     `json:"with_vectors"`
	Offset      any      `json:"offset,omitempty"`
}

type scrollResponse struct {
	Result struct {
		Points []struct {
			ID      any            `json:"id"`
			Payload map[string]any `json:"payload"`
		} `json:"points"`
		NextPageOffset any `json:"next_page_offset"`
	} `json:"result"`
}

// ScrollIDs walks the collection, projecting each point's drift fields
// (payload_hash, source_id, source_hash). 404 → empty map. Read-only.
func (v *qdrantVectorStore) ScrollIDs(ctx context.Context) (map[string]ScrollItem, error) {
	endpoint, err := v.endpoint("/points/scroll")
	if err != nil {
		return nil, err
	}
	out := make(map[string]ScrollItem)
	var offset any
	for {
		reqObj := scrollRequest{
			Limit:       scrollPageLimit,
			WithPayload: []string{payloadHashKey, sourceIDKey, sourceHashKey, chunkTotalKey, metaMarkerKey},
			WithVectors: false,
			Offset:      offset,
		}
		body, err := json.Marshal(reqObj)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal scroll request: %w", err)
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("failed to create scroll request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := v.do(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode == http.StatusNotFound {
			_ = resp.Body.Close()
			return map[string]ScrollItem{}, nil
		}
		if resp.StatusCode != http.StatusOK {
			raw, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			return nil, fmt.Errorf("qdrant scroll returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
		}
		var decoded scrollResponse
		if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
			_ = resp.Body.Close()
			return nil, fmt.Errorf("failed to decode scroll response: %w", err)
		}
		_ = resp.Body.Close()

		for _, p := range decoded.Result.Points {
			id := stringifyID(p.ID)
			if id == "" {
				continue
			}
			if _, isMeta := p.Payload[metaMarkerKey]; isMeta {
				continue // sentinel: no source_id, excluded from drift/ghost logic
			}
			hash, _ := p.Payload[payloadHashKey].(string)
			srcID, _ := p.Payload[sourceIDKey].(string)
			srcHash, _ := p.Payload[sourceHashKey].(string)
			total := 0
			if raw, ok := p.Payload[chunkTotalKey].(float64); ok {
				total = int(raw)
			}
			out[id] = ScrollItem{PayloadHash: hash, SourceID: srcID, SourceHash: srcHash, ChunkTotal: total}
		}
		if decoded.Result.NextPageOffset == nil {
			break
		}
		offset = decoded.Result.NextPageOffset
	}
	return out, nil
}

// BatchDelete removes points by ID in MaxDeleteBatch-sized chunks. Empty → no-op.
func (v *qdrantVectorStore) BatchDelete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	endpoint, err := v.endpoint("/points/delete")
	if err != nil {
		return err
	}
	endpoint = withWait(endpoint)
	for start := 0; start < len(ids); start += MaxDeleteBatch {
		end := start + MaxDeleteBatch
		if end > len(ids) {
			end = len(ids)
		}
		if err := v.writeJSON(ctx, http.MethodPost, endpoint, map[string]any{"points": ids[start:end]}, fmt.Sprintf("batch delete %d-%d", start, end)); err != nil {
			return err
		}
	}
	return nil
}

// Available reports whether Qdrant is reachable.
func (v *qdrantVectorStore) Available(ctx context.Context) bool {
	base := strings.TrimRight(strings.TrimSpace(v.baseURL), "/")
	if base == "" {
		return false
	}
	u, err := url.Parse(base)
	if err != nil {
		return false
	}
	u.Path = fmt.Sprintf("%s/collections", strings.TrimRight(u.Path, "/"))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return false
	}
	resp, err := v.do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// --- helpers ----------------------------------------------------------------

func (v *qdrantVectorStore) writeJSON(ctx context.Context, method, endpoint string, payload any, op string) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal %s request: %w", op, err)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create %s request: %w", op, err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := v.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("qdrant %s returned status %d: %s", op, resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	return nil
}

func withWait(endpoint string) string {
	if strings.Contains(endpoint, "?") {
		return endpoint + "&wait=true"
	}
	return endpoint + "?wait=true"
}

func stringifyID(id any) string {
	switch x := id.(type) {
	case string:
		return x
	case float64:
		return fmt.Sprintf("%.0f", x)
	default:
		return fmt.Sprintf("%v", x)
	}
}
