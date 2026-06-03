package aisearch

import (
	"bytes"
	"context"
	"encoding/json"
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
}

type sparseVectorParams struct {
	Modifier string `json:"modifier,omitempty"`
}

type createCollectionRequest struct {
	Vectors       map[string]namedVectorParams  `json:"vectors"`
	SparseVectors map[string]sparseVectorParams `json:"sparse_vectors,omitempty"`
}

// EnsureCollection creates the collection (named dense vector, optional named
// sparse vector with idf modifier) if it does not already exist. Idempotent.
func (v *qdrantVectorStore) EnsureCollection(ctx context.Context, spec CollectionSpec) error {
	size := spec.DenseSize
	if size <= 0 {
		size = DefaultVectorSize
	}
	distance := strings.TrimSpace(spec.DenseDistance)
	if distance == "" {
		distance = DefaultDenseDistance
	}
	endpoint, err := v.endpoint("")
	if err != nil {
		return err
	}

	getReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	getResp, err := v.do(getReq)
	if err != nil {
		return err
	}
	_ = getResp.Body.Close()
	if getResp.StatusCode == http.StatusOK {
		return nil
	}
	if getResp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("qdrant collection info returned status %d", getResp.StatusCode)
	}

	create := createCollectionRequest{
		Vectors: map[string]namedVectorParams{
			denseVectorName: {Size: size, Distance: distance},
		},
	}
	if spec.Sparse {
		modifier := strings.TrimSpace(spec.SparseModifier)
		if modifier == "" {
			modifier = DefaultSparseModifier
		}
		create.SparseVectors = map[string]sparseVectorParams{
			sparseVectorName: {Modifier: modifier},
		}
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
	return nil
}

// --- upsert -----------------------------------------------------------------

type sparseVectorJSON struct {
	Indices []uint32  `json:"indices"`
	Values  []float32 `json:"values"`
}

// Upsert inserts or updates one point. The vector is always a named-vector map
// ({"dense": [...]}); when Sparse is set the named "sparse" vector is added.
func (v *qdrantVectorStore) Upsert(ctx context.Context, point Point) error {
	id := strings.TrimSpace(point.ID)
	if id == "" {
		return fmt.Errorf("point id is required")
	}
	endpoint, err := v.endpoint("/points")
	if err != nil {
		return err
	}
	endpoint = withWait(endpoint)

	vec := map[string]any{denseVectorName: point.Dense}
	if point.Sparse != nil {
		vec[sparseVectorName] = sparseVectorJSON{Indices: point.Sparse.Indices, Values: point.Sparse.Values}
	}
	reqBody := map[string]any{
		"points": []map[string]any{
			{"id": id, "vector": vec, "payload": point.Payload},
		},
	}
	return v.writeJSON(ctx, http.MethodPut, endpoint, reqBody, "upsert")
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

// CountPoints returns the exact number of points in the collection.
func (v *qdrantVectorStore) CountPoints(ctx context.Context) (int, error) {
	endpoint, err := v.endpoint("/points/count")
	if err != nil {
		return 0, err
	}
	body, err := json.Marshal(map[string]any{"exact": true})
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
			WithPayload: []string{payloadHashKey, sourceIDKey, sourceHashKey, chunkTotalKey},
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
