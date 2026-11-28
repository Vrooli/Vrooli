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

	sharedsearch "github.com/vrooli/ai-go/search"
	"github.com/vrooli/api-core/storage"
)

// SearchResult represents a single vector search result.
type SearchResult struct {
	ID      string
	Score   float64
	Payload map[string]interface{}
}

// ScrollItem is the per-point projection returned by VectorStore.ScrollIDs —
// just enough to decide if the point matches what's on disk.
type ScrollItem struct {
	PayloadHash string
}

// MaxDeleteBatch caps the number of point IDs sent in a single
// /points/delete request to keep payloads small and predictable.
const MaxDeleteBatch = 256

// scrollPageLimit is the per-request page size used by ScrollIDs. Sized to
// fit comfortably in a single qdrant response.
const scrollPageLimit = 256

// VectorStore is the qdrant-shaped interface used by the aisearch package.
// Concrete implementations are unexported; tests substitute fakes.
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

// qdrantVectorStore is the qdrant-backed VectorStore. Unexported by design.
type qdrantVectorStore struct {
	baseURL       string
	apiKey        string
	collection    string
	embeddingRole string
	vectorSize    int
	client        *http.Client
}

var resolveEmbeddingPolicy = sharedsearch.ResolveEmbeddingPolicy

func defaultSkillsCollection() (string, error) {
	ns, err := storage.ResolveNamespace(storage.NamespaceConfig{FallbackScenario: "prompt-manager"})
	if err != nil {
		return "", err
	}
	return ns.Collection("skills")
}

// NewVectorStore creates a new Qdrant-backed VectorStore.
//
// Deprecated: production callers should use NewVectorStoreForRole so collection
// dimensions stay owned by resource-ollama policy metadata.
func NewVectorStore(baseURL, apiKey, collection string, vectorSize int) VectorStore {
	if collection == "" {
		var err error
		collection, err = defaultSkillsCollection()
		if err != nil {
			panic(fmt.Sprintf("resolve skills collection: %v", err))
		}
	}
	return &qdrantVectorStore{
		baseURL:    baseURL,
		apiKey:     apiKey,
		collection: collection,
		vectorSize: vectorSize,
		client:     &http.Client{Timeout: 15 * time.Second},
	}
}

// NewVectorStoreForRole creates a Qdrant-backed VectorStore whose collection
// dimensions are resolved from the configured Ollama embedding role at
// EnsureCollection time.
func NewVectorStoreForRole(baseURL, apiKey, collection, embeddingRole string) VectorStore {
	if collection == "" {
		var err error
		collection, err = defaultSkillsCollection()
		if err != nil {
			panic(fmt.Sprintf("resolve skills collection: %v", err))
		}
	}
	embeddingRole = strings.TrimSpace(embeddingRole)
	if embeddingRole == "" {
		embeddingRole = "embedding.default"
	}
	return &qdrantVectorStore{
		baseURL:       baseURL,
		apiKey:        apiKey,
		collection:    collection,
		embeddingRole: embeddingRole,
		client:        &http.Client{Timeout: 15 * time.Second},
	}
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

func (v *qdrantVectorStore) base() (string, error) {
	base := strings.TrimRight(strings.TrimSpace(v.baseURL), "/")
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
	collection := strings.TrimSpace(v.collection)
	if collection == "" {
		return fmt.Errorf("collection is required")
	}
	base, err := v.base()
	if err != nil {
		return err
	}
	vectorSize, err := v.resolvedVectorSize(ctx)
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
	create.Vectors.Size = vectorSize
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

func (v *qdrantVectorStore) resolvedVectorSize(ctx context.Context) (int, error) {
	if v.vectorSize > 0 {
		return v.vectorSize, nil
	}
	role := strings.TrimSpace(v.embeddingRole)
	if role == "" {
		role = "embedding.default"
	}
	policy, err := resolveEmbeddingPolicy(ctx, role)
	if err != nil {
		return 0, fmt.Errorf("resolve embedding policy for %s: %w", role, err)
	}
	if policy.Dimensions <= 0 {
		return 0, fmt.Errorf("embedding role %s resolved without dimensions", role)
	}
	return policy.Dimensions, nil
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
	collection := strings.TrimSpace(v.collection)
	id = strings.TrimSpace(id)
	if collection == "" || id == "" {
		return fmt.Errorf("collection and id are required")
	}

	base, err := v.base()
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
	collection := strings.TrimSpace(v.collection)
	id = strings.TrimSpace(id)
	if collection == "" || id == "" {
		return fmt.Errorf("collection and id are required")
	}

	base, err := v.base()
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

// Search performs a vector similarity search.
func (v *qdrantVectorStore) Search(ctx context.Context, vector []float64, limit int, threshold float64) ([]SearchResult, error) {
	collection := strings.TrimSpace(v.collection)
	if collection == "" {
		return nil, fmt.Errorf("collection is required")
	}
	base, err := v.base()
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
	base, err := v.base()
	if err != nil {
		return 0, err
	}
	u, err := url.Parse(base)
	if err != nil {
		return 0, fmt.Errorf("invalid qdrant url: %w", err)
	}
	u.Path = fmt.Sprintf("%s/collections/%s/points/count", strings.TrimRight(u.Path, "/"), v.collection)

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
	base, err := v.base()
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

type scrollRequest struct {
	Limit       int         `json:"limit"`
	WithPayload []string    `json:"with_payload"`
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

// ScrollIDs walks the collection, returning every point ID along with its
// payload_hash projection (or empty string if absent). 404 → empty map. Used
// by the reconciler to detect ghost points without changing qdrant state.
func (v *qdrantVectorStore) ScrollIDs(ctx context.Context) (map[string]ScrollItem, error) {
	collection := strings.TrimSpace(v.collection)
	if collection == "" {
		return nil, fmt.Errorf("collection is required")
	}
	base, err := v.base()
	if err != nil {
		return nil, err
	}
	u, err := url.Parse(base)
	if err != nil {
		return nil, fmt.Errorf("invalid qdrant url: %w", err)
	}
	u.Path = fmt.Sprintf("%s/collections/%s/points/scroll", strings.TrimRight(u.Path, "/"), collection)
	endpoint := u.String()

	out := make(map[string]ScrollItem)
	var offset interface{}
	for {
		reqObj := scrollRequest{
			Limit:       scrollPageLimit,
			WithPayload: []string{payloadHashKey},
			WithVectors: false,
			Offset:      offset,
		}
		body, err := json.Marshal(reqObj)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal scroll request: %w", err)
		}
		req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
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
			out[id] = ScrollItem{PayloadHash: hash}
		}

		if decoded.Result.NextPageOffset == nil {
			break
		}
		offset = decoded.Result.NextPageOffset
	}
	return out, nil
}

// BatchDelete removes points by ID, chunking into MaxDeleteBatch-sized
// requests so a large drift cleanup never sends an oversized payload. Empty
// slice is a no-op.
func (v *qdrantVectorStore) BatchDelete(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	collection := strings.TrimSpace(v.collection)
	if collection == "" {
		return fmt.Errorf("collection is required")
	}
	base, err := v.base()
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
	endpoint := u.String()

	for start := 0; start < len(ids); start += MaxDeleteBatch {
		end := start + MaxDeleteBatch
		if end > len(ids) {
			end = len(ids)
		}
		chunk := ids[start:end]

		body, err := json.Marshal(map[string]interface{}{"points": chunk})
		if err != nil {
			return fmt.Errorf("failed to marshal batch delete request: %w", err)
		}
		req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("failed to create batch delete request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := v.do(req)
		if err != nil {
			return fmt.Errorf("batch delete chunk %d-%d failed: %w", start, end, err)
		}
		if resp.StatusCode != http.StatusOK {
			raw, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			return fmt.Errorf("batch delete chunk %d-%d returned status %d: %s", start, end, resp.StatusCode, strings.TrimSpace(string(raw)))
		}
		_ = resp.Body.Close()
	}
	return nil
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
