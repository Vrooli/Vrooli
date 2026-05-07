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

// Default values applied by NewVectorStore when callers pass empty/zero. Kept
// as exported constants so behavior tests (which exercise the constructor via
// observable HTTP traffic) can pin the same values without importing private
// state.
const (
	DefaultCollectionName = "swarm-manager"
	DefaultVectorSize     = 768

	// MaxDeleteBatch caps the number of point IDs sent in a single
	// /points/delete request. Qdrant accepts arbitrary-size batches; the cap
	// keeps request bodies bounded and lets BatchDelete chunk transparently.
	MaxDeleteBatch = 256

	// scrollPageLimit is the per-request page size used by ScrollIDs. Tuned
	// to keep response bodies under ~1MB at typical payload widths.
	scrollPageLimit = 256
)

// ScrollItem is the per-point projection returned by ScrollIDs. It carries
// only the reconciler-relevant fields — never the vector — so a full
// enumeration of a 10K-point collection is bounded at a few hundred KB.
type ScrollItem struct {
	// PayloadHash is the stored "skip-if-unchanged" fingerprint. Empty when
	// the point is a legacy upsert from before payload_hash was introduced;
	// the reconciler treats absent hash as "force re-embed once."
	PayloadHash string

	// Archived preserves the indexed archive flag so the reconciler can audit
	// archive-flag drift without a second round-trip.
	Archived bool
}

// VectorStore is the qdrant-shaped seam the reconciler and write-through hooks
// consume. The production implementation talks to a Qdrant HTTP API; tests
// substitute pure-Go fakes implementing this surface without httptest.
//
// Decision boundary: "where does the index live?" — the single named seam for
// every read and write to the vector store.
type VectorStore interface {
	EnsureCollection(ctx context.Context) error
	Upsert(ctx context.Context, id string, vector []float64, payload map[string]interface{}) error
	Delete(ctx context.Context, id string) error
	BatchDelete(ctx context.Context, ids []string) error
	Search(ctx context.Context, vector []float64, limit int, threshold float64) ([]SearchResult, error)
	CountPoints(ctx context.Context) (int, error)
	ScrollIDs(ctx context.Context) (map[string]ScrollItem, error)
	Available(ctx context.Context) bool
}

// qdrantVectorStore is the production VectorStore over Qdrant's HTTP API.
type qdrantVectorStore struct {
	BaseURL    string
	APIKey     string
	Collection string
	VectorSize int
	Client     *http.Client
}

// NewVectorStore creates a Qdrant-backed VectorStore. Empty collection and
// non-positive vectorSize fall back to DefaultCollectionName and
// DefaultVectorSize respectively. The returned value implements VectorStore;
// callers should hold it as VectorStore at consumption sites.
func NewVectorStore(baseURL, apiKey, collection string, vectorSize int) VectorStore {
	if collection == "" {
		collection = DefaultCollectionName
	}
	if vectorSize <= 0 {
		vectorSize = DefaultVectorSize
	}
	return &qdrantVectorStore{
		BaseURL:    baseURL,
		APIKey:     apiKey,
		Collection: collection,
		VectorSize: vectorSize,
		Client:     &http.Client{Timeout: 15 * time.Second},
	}
}

func (v *qdrantVectorStore) client() *http.Client {
	if v != nil && v.Client != nil {
		return v.Client
	}
	return &http.Client{Timeout: 15 * time.Second}
}

func (v *qdrantVectorStore) do(req *http.Request) (*http.Response, error) {
	if key := strings.TrimSpace(v.APIKey); key != "" {
		req.Header.Set("api-key", key)
	}
	resp, err := v.client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("qdrant request failed: %w", err)
	}
	return resp, nil
}

func (v *qdrantVectorStore) baseURL() (string, error) {
	base := strings.TrimRight(strings.TrimSpace(v.BaseURL), "/")
	if base == "" {
		return "", fmt.Errorf("qdrant base url is required")
	}
	return base, nil
}

type createCollectionRequest struct {
	Vectors struct {
		Size     int    `json:"size"`
		Distance string `json:"distance"`
	} `json:"vectors"`
}

// EnsureCollection creates the collection if it doesn't exist.
func (v *qdrantVectorStore) EnsureCollection(ctx context.Context) error {
	collection := strings.TrimSpace(v.Collection)
	if collection == "" {
		return fmt.Errorf("collection is required")
	}
	if v.VectorSize <= 0 {
		return fmt.Errorf("vectorSize must be > 0")
	}
	base, err := v.baseURL()
	if err != nil {
		return err
	}

	u, err := url.Parse(base)
	if err != nil {
		return fmt.Errorf("invalid qdrant url: %w", err)
	}
	u.Path = fmt.Sprintf("%s/collections/%s", strings.TrimRight(u.Path, "/"), collection)

	getReq, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
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

	var create createCollectionRequest
	create.Vectors.Size = v.VectorSize
	create.Vectors.Distance = "Cosine"
	body, err := json.Marshal(create)
	if err != nil {
		return fmt.Errorf("failed to marshal create request: %w", err)
	}

	putReq, err := http.NewRequestWithContext(ctx, "PUT", u.String(), bytes.NewReader(body))
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

type upsertRequest struct {
	Points []struct {
		ID      string                 `json:"id"`
		Vector  []float64              `json:"vector"`
		Payload map[string]interface{} `json:"payload,omitempty"`
	} `json:"points"`
}

// Upsert inserts or updates a vector point.
func (v *qdrantVectorStore) Upsert(ctx context.Context, id string, vector []float64, payload map[string]interface{}) error {
	collection := strings.TrimSpace(v.Collection)
	id = strings.TrimSpace(id)
	if collection == "" || id == "" {
		return fmt.Errorf("collection and id are required")
	}

	base, err := v.baseURL()
	if err != nil {
		return err
	}

	u, err := url.Parse(base)
	if err != nil {
		return fmt.Errorf("invalid qdrant url: %w", err)
	}
	u.Path = fmt.Sprintf("%s/collections/%s/points", strings.TrimRight(u.Path, "/"), collection)
	values := u.Query()
	values.Set("wait", "true")
	u.RawQuery = values.Encode()

	reqBody := upsertRequest{
		Points: []struct {
			ID      string                 `json:"id"`
			Vector  []float64              `json:"vector"`
			Payload map[string]interface{} `json:"payload,omitempty"`
		}{
			{ID: id, Vector: vector, Payload: payload},
		},
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal upsert request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", u.String(), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := v.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("qdrant upsert returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	return nil
}

// Delete removes a vector point.
func (v *qdrantVectorStore) Delete(ctx context.Context, id string) error {
	collection := strings.TrimSpace(v.Collection)
	id = strings.TrimSpace(id)
	if collection == "" || id == "" {
		return fmt.Errorf("collection and id are required")
	}

	base, err := v.baseURL()
	if err != nil {
		return err
	}

	u, err := url.Parse(base)
	if err != nil {
		return fmt.Errorf("invalid qdrant url: %w", err)
	}
	u.Path = fmt.Sprintf("%s/collections/%s/points/delete", strings.TrimRight(u.Path, "/"), collection)
	values := u.Query()
	values.Set("wait", "true")
	u.RawQuery = values.Encode()

	body, err := json.Marshal(map[string]interface{}{
		"points": []string{id},
	})
	if err != nil {
		return fmt.Errorf("failed to marshal delete request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", u.String(), bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := v.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("qdrant delete returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	return nil
}

type searchRequest struct {
	Vector         []float64 `json:"vector"`
	Limit          int       `json:"limit"`
	WithPayload    bool      `json:"with_payload"`
	ScoreThreshold *float64  `json:"score_threshold,omitempty"`
}

type searchResponse struct {
	Result []struct {
		ID      interface{}            `json:"id"`
		Score   float64                `json:"score"`
		Payload map[string]interface{} `json:"payload"`
	} `json:"result"`
}

// SearchResult represents a single vector search result.
type SearchResult struct {
	ID      string
	Score   float64
	Payload map[string]interface{}
}

// Search performs a vector similarity search.
func (v *qdrantVectorStore) Search(ctx context.Context, vector []float64, limit int, threshold float64) ([]SearchResult, error) {
	collection := strings.TrimSpace(v.Collection)
	if collection == "" {
		return nil, fmt.Errorf("collection is required")
	}
	base, err := v.baseURL()
	if err != nil {
		return nil, err
	}
	u, err := url.Parse(base)
	if err != nil {
		return nil, fmt.Errorf("invalid qdrant url: %w", err)
	}
	u.Path = fmt.Sprintf("%s/collections/%s/points/search", strings.TrimRight(u.Path, "/"), collection)

	reqObj := searchRequest{
		Vector:      vector,
		Limit:       limit,
		WithPayload: true,
	}
	if threshold > 0 {
		reqObj.ScoreThreshold = &threshold
	}
	body, err := json.Marshal(reqObj)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal search request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", u.String(), bytes.NewReader(body))
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
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("qdrant search returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var decoded searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %w", err)
	}

	out := make([]SearchResult, 0, len(decoded.Result))
	for _, r := range decoded.Result {
		out = append(out, SearchResult{
			ID:      stringifyID(r.ID),
			Score:   r.Score,
			Payload: r.Payload,
		})
	}
	return out, nil
}

type countRequest struct {
	Exact bool `json:"exact"`
}

type countResponse struct {
	Result struct {
		Count int `json:"count"`
	} `json:"result"`
}

// CountPoints returns the number of points in the collection.
func (v *qdrantVectorStore) CountPoints(ctx context.Context) (int, error) {
	base, err := v.baseURL()
	if err != nil {
		return 0, err
	}
	u, err := url.Parse(base)
	if err != nil {
		return 0, fmt.Errorf("invalid qdrant url: %w", err)
	}
	u.Path = fmt.Sprintf("%s/collections/%s/points/count", strings.TrimRight(u.Path, "/"), v.Collection)

	body, err := json.Marshal(countRequest{Exact: true})
	if err != nil {
		return 0, fmt.Errorf("failed to marshal count request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", u.String(), bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("failed to create count request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := v.do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	// A missing collection is a valid "empty" state on first boot; the
	// reconcile lifecycle creates the collection before its first upsert.
	// Treating 404 as zero lets the reconciler and GetStatus both report
	// accurately without requiring the collection to be pre-created.
	if resp.StatusCode == http.StatusNotFound {
		return 0, nil
	}
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("qdrant count returned status %d", resp.StatusCode)
	}

	var decoded countResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return 0, fmt.Errorf("failed to decode count response: %w", err)
	}
	return decoded.Result.Count, nil
}

// Available checks if Qdrant is reachable.
func (v *qdrantVectorStore) Available(ctx context.Context) bool {
	base, err := v.baseURL()
	if err != nil {
		return false
	}

	u, err := url.Parse(base)
	if err != nil {
		return false
	}
	u.Path = fmt.Sprintf("%s/collections", strings.TrimRight(u.Path, "/"))

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
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

func stringifyID(id interface{}) string {
	switch v := id.(type) {
	case string:
		return v
	case float64:
		return fmt.Sprintf("%.0f", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// scrollRequest mirrors Qdrant's /points/scroll body. Offset is interface{}
// because Qdrant returns string offsets for UUID-keyed collections and numeric
// offsets for integer-keyed ones; we pass whatever shape the previous response
// gave us back verbatim.
type scrollRequest struct {
	Limit       int         `json:"limit"`
	WithPayload bool        `json:"with_payload"`
	WithVectors bool        `json:"with_vectors"`
	Offset      interface{} `json:"offset,omitempty"`
}

type scrollResponse struct {
	Result struct {
		Points []struct {
			ID      interface{}            `json:"id"`
			Payload map[string]interface{} `json:"payload"`
		} `json:"points"`
		NextPageOffset interface{} `json:"next_page_offset"`
	} `json:"result"`
}

// ScrollIDs enumerates every point in the collection, returning a map keyed by
// stringified point ID with each point's payload_hash and archived flag.
//
// Pagination loops on next_page_offset until nil. A 404 (collection missing)
// returns an empty map without error — same gentle-degradation contract as
// CountPoints, so the reconciler can run before EnsureCollection at boot.
//
// The returned map omits the vector itself, keeping memory bounded even for
// large collections.
func (v *qdrantVectorStore) ScrollIDs(ctx context.Context) (map[string]ScrollItem, error) {
	collection := strings.TrimSpace(v.Collection)
	if collection == "" {
		return nil, fmt.Errorf("collection is required")
	}
	base, err := v.baseURL()
	if err != nil {
		return nil, err
	}
	u, err := url.Parse(base)
	if err != nil {
		return nil, fmt.Errorf("invalid qdrant url: %w", err)
	}
	u.Path = fmt.Sprintf("%s/collections/%s/points/scroll", strings.TrimRight(u.Path, "/"), collection)

	out := make(map[string]ScrollItem)
	var offset interface{}
	for {
		reqObj := scrollRequest{
			Limit:       scrollPageLimit,
			WithPayload: true,
			WithVectors: false,
			Offset:      offset,
		}
		body, err := json.Marshal(reqObj)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal scroll request: %w", err)
		}
		req, err := http.NewRequestWithContext(ctx, "POST", u.String(), bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("failed to create scroll request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := v.do(req)
		if err != nil {
			return nil, err
		}
		// Per page: read fully, decode, then close before the next iteration.
		if resp.StatusCode == http.StatusNotFound {
			_ = resp.Body.Close()
			return out, nil
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
			item := ScrollItem{}
			if p.Payload != nil {
				if h, ok := p.Payload["payload_hash"].(string); ok {
					item.PayloadHash = h
				}
				if a, ok := p.Payload["archived"].(bool); ok {
					item.Archived = a
				}
			}
			out[id] = item
		}

		if decoded.Result.NextPageOffset == nil {
			return out, nil
		}
		offset = decoded.Result.NextPageOffset
	}
}

// BatchDelete removes a batch of point IDs in one (or more) Qdrant requests.
// IDs are chunked at MaxDeleteBatch to keep request bodies bounded; per-chunk
// failures are aggregated via errors.Join so a partial failure surfaces every
// failing chunk to the caller in one error.
//
// Empty ids is a no-op (returns nil) — lets the reconciler's Apply phase pass
// "ToDelete" slices unconditionally without precondition checks.
func (v *qdrantVectorStore) BatchDelete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	collection := strings.TrimSpace(v.Collection)
	if collection == "" {
		return fmt.Errorf("collection is required")
	}
	base, err := v.baseURL()
	if err != nil {
		return err
	}
	u, err := url.Parse(base)
	if err != nil {
		return fmt.Errorf("invalid qdrant url: %w", err)
	}
	u.Path = fmt.Sprintf("%s/collections/%s/points/delete", strings.TrimRight(u.Path, "/"), collection)
	values := u.Query()
	values.Set("wait", "true")
	u.RawQuery = values.Encode()

	var errs []error
	for start := 0; start < len(ids); start += MaxDeleteBatch {
		end := start + MaxDeleteBatch
		if end > len(ids) {
			end = len(ids)
		}
		chunk := ids[start:end]

		body, err := json.Marshal(map[string]interface{}{"points": chunk})
		if err != nil {
			errs = append(errs, fmt.Errorf("marshal delete chunk [%d:%d]: %w", start, end, err))
			continue
		}
		req, err := http.NewRequestWithContext(ctx, "POST", u.String(), bytes.NewReader(body))
		if err != nil {
			errs = append(errs, fmt.Errorf("create delete chunk [%d:%d]: %w", start, end, err))
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := v.do(req)
		if err != nil {
			errs = append(errs, fmt.Errorf("delete chunk [%d:%d]: %w", start, end, err))
			continue
		}
		if resp.StatusCode != http.StatusOK {
			raw, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			errs = append(errs, fmt.Errorf("delete chunk [%d:%d] returned status %d: %s", start, end, resp.StatusCode, strings.TrimSpace(string(raw))))
			continue
		}
		_ = resp.Body.Close()
	}
	return errors.Join(errs...)
}
