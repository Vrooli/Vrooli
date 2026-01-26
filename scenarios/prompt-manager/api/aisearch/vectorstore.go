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

// VectorStore manages vector storage in Qdrant.
type VectorStore struct {
	BaseURL    string
	APIKey     string
	Collection string
	VectorSize int
	Client     *http.Client
}

// NewVectorStore creates a new Qdrant vector store.
func NewVectorStore(baseURL, apiKey, collection string, vectorSize int) *VectorStore {
	if collection == "" {
		collection = "prompt-manager-skills"
	}
	if vectorSize <= 0 {
		vectorSize = 768 // nomic-embed-text default
	}
	return &VectorStore{
		BaseURL:    baseURL,
		APIKey:     apiKey,
		Collection: collection,
		VectorSize: vectorSize,
		Client:     &http.Client{Timeout: 15 * time.Second},
	}
}

func (v *VectorStore) client() *http.Client {
	if v != nil && v.Client != nil {
		return v.Client
	}
	return &http.Client{Timeout: 15 * time.Second}
}

func (v *VectorStore) do(req *http.Request) (*http.Response, error) {
	if key := strings.TrimSpace(v.APIKey); key != "" {
		req.Header.Set("api-key", key)
	}
	resp, err := v.client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("qdrant request failed: %w", err)
	}
	return resp, nil
}

func (v *VectorStore) baseURL() (string, error) {
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
func (v *VectorStore) EnsureCollection(ctx context.Context) error {
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
func (v *VectorStore) Upsert(ctx context.Context, id string, vector []float64, payload map[string]interface{}) error {
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
func (v *VectorStore) Delete(ctx context.Context, id string) error {
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
	Vector         []float64   `json:"vector"`
	Limit          int         `json:"limit"`
	WithPayload    bool        `json:"with_payload"`
	ScoreThreshold *float64    `json:"score_threshold,omitempty"`
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
func (v *VectorStore) Search(ctx context.Context, vector []float64, limit int, threshold float64) ([]SearchResult, error) {
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
func (v *VectorStore) CountPoints(ctx context.Context) (int, error) {
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
func (v *VectorStore) Available(ctx context.Context) bool {
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
