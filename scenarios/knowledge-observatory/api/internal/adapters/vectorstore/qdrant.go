package vectorstore

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

	"knowledge-observatory/internal/ports"
)

type Qdrant struct {
	BaseURL string
	APIKey  string
	Client  *http.Client
}

func (q *Qdrant) client() *http.Client {
	if q != nil && q.Client != nil {
		return q.Client
	}
	return &http.Client{Timeout: 15 * time.Second}
}

func (q *Qdrant) do(req *http.Request) (*http.Response, error) {
	if key := strings.TrimSpace(q.APIKey); key != "" {
		req.Header.Set("api-key", key)
	}
	resp, err := q.client().Do(req)
	if err != nil {
		return nil, fmt.Errorf("qdrant request failed: %w", err)
	}
	return resp, nil
}

func (q *Qdrant) baseURL() (string, error) {
	base := strings.TrimRight(strings.TrimSpace(q.BaseURL), "/")
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

func (q *Qdrant) EnsureCollection(ctx context.Context, collection string, vectorSize int) error {
	collection = strings.TrimSpace(collection)
	if collection == "" {
		return fmt.Errorf("collection is required")
	}
	if vectorSize <= 0 {
		return fmt.Errorf("vectorSize must be > 0")
	}
	base, err := q.baseURL()
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
	getResp, err := q.do(getReq)
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
	putResp, err := q.do(putReq)
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

func (q *Qdrant) UpsertPoint(ctx context.Context, collection string, id string, vector []float64, payload map[string]interface{}) error {
	collection = strings.TrimSpace(collection)
	id = strings.TrimSpace(id)
	if collection == "" || id == "" {
		return fmt.Errorf("collection and id are required")
	}

	base, err := q.baseURL()
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

	resp, err := q.do(req)
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

func (q *Qdrant) DeletePoint(ctx context.Context, collection string, id string) error {
	collection = strings.TrimSpace(collection)
	id = strings.TrimSpace(id)
	if collection == "" || id == "" {
		return fmt.Errorf("collection and id are required")
	}

	base, err := q.baseURL()
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

	resp, err := q.do(req)
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
	Filter         interface{} `json:"filter,omitempty"`
	ScoreThreshold *float64    `json:"score_threshold,omitempty"`
}

type searchResponse struct {
	Result []struct {
		ID      interface{}            `json:"id"`
		Score   float64                `json:"score"`
		Payload map[string]interface{} `json:"payload"`
	} `json:"result"`
}

type matchAny struct {
	Any []string `json:"any"`
}

type matchClause struct {
	Match matchAny `json:"match"`
	Key   string   `json:"key"`
}

type rangeClause struct {
	Range map[string]int64 `json:"range"`
	Key   string           `json:"key"`
}

type qdrantFilter struct {
	Must []interface{} `json:"must,omitempty"`
}

func buildFilter(filter *ports.VectorFilter) *qdrantFilter {
	if filter == nil {
		return nil
	}

	must := make([]interface{}, 0, 4)
	if len(filter.Namespaces) > 0 {
		must = append(must, matchClause{Key: "namespace", Match: matchAny{Any: filter.Namespaces}})
	}
	if len(filter.Visibility) > 0 {
		must = append(must, matchClause{Key: "visibility", Match: matchAny{Any: filter.Visibility}})
	}
	if len(filter.Tags) > 0 {
		must = append(must, matchClause{Key: "tags", Match: matchAny{Any: filter.Tags}})
	}

	if filter.IngestedAfterMS != nil || filter.IngestedBeforeMS != nil {
		rng := map[string]int64{}
		if filter.IngestedAfterMS != nil {
			rng["gte"] = *filter.IngestedAfterMS
		}
		if filter.IngestedBeforeMS != nil {
			rng["lte"] = *filter.IngestedBeforeMS
		}
		must = append(must, rangeClause{Key: "ingested_at_unix_ms", Range: rng})
	}

	if len(must) == 0 {
		return nil
	}
	return &qdrantFilter{Must: must}
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

func (q *Qdrant) Search(ctx context.Context, collection string, vector []float64, limit int, threshold float64, filter *ports.VectorFilter) ([]ports.VectorSearchResult, error) {
	collection = strings.TrimSpace(collection)
	if collection == "" {
		return nil, fmt.Errorf("collection is required")
	}
	base, err := q.baseURL()
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
	if built := buildFilter(filter); built != nil {
		reqObj.Filter = built
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

	resp, err := q.do(req)
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

	out := make([]ports.VectorSearchResult, 0, len(decoded.Result))
	for _, r := range decoded.Result {
		out = append(out, ports.VectorSearchResult{
			ID:      stringifyID(r.ID),
			Score:   r.Score,
			Payload: r.Payload,
		})
	}
	return out, nil
}

type scrollRequest struct {
	Limit       int         `json:"limit"`
	WithPayload bool        `json:"with_payload"`
	WithVectors bool        `json:"with_vectors"`
	Filter      interface{} `json:"filter,omitempty"`
	Offset      interface{} `json:"offset,omitempty"`
}

type scrollResponse struct {
	Result struct {
		Points []struct {
			ID      interface{}            `json:"id"`
			Vector  []float64              `json:"vector"`
			Payload map[string]interface{} `json:"payload"`
		} `json:"points"`
		NextPageOffset interface{} `json:"next_page_offset"`
	} `json:"result"`
}

func (q *Qdrant) SamplePoints(ctx context.Context, collection string, limit int) ([]ports.VectorPoint, error) {
	collection = strings.TrimSpace(collection)
	if collection == "" {
		return nil, fmt.Errorf("collection is required")
	}
	if limit <= 0 {
		limit = 50
	}

	base, err := q.baseURL()
	if err != nil {
		return nil, err
	}
	u, err := url.Parse(base)
	if err != nil {
		return nil, fmt.Errorf("invalid qdrant url: %w", err)
	}
	u.Path = fmt.Sprintf("%s/collections/%s/points/scroll", strings.TrimRight(u.Path, "/"), collection)

	reqObj := scrollRequest{
		Limit:       limit,
		WithPayload: true,
		WithVectors: true,
	}
	body, err := json.Marshal(reqObj)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal scroll request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", u.String(), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := q.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("qdrant scroll returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var decoded scrollResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("failed to decode scroll response: %w", err)
	}

	out := make([]ports.VectorPoint, 0, len(decoded.Result.Points))
	for _, p := range decoded.Result.Points {
		out = append(out, ports.VectorPoint{
			ID:      stringifyID(p.ID),
			Vector:  p.Vector,
			Payload: p.Payload,
		})
	}
	return out, nil
}

type collectionsResponse struct {
	Result struct {
		Collections []struct {
			Name string `json:"name"`
		} `json:"collections"`
	} `json:"result"`
}

func (q *Qdrant) ListCollections(ctx context.Context) ([]string, error) {
	base, err := q.baseURL()
	if err != nil {
		return nil, err
	}
	u, err := url.Parse(base)
	if err != nil {
		return nil, fmt.Errorf("invalid qdrant url: %w", err)
	}
	u.Path = fmt.Sprintf("%s/collections", strings.TrimRight(u.Path, "/"))

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create collections request: %w", err)
	}

	resp, err := q.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("qdrant collections returned status %d", resp.StatusCode)
	}

	var decoded collectionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("failed to decode qdrant collections response: %w", err)
	}

	out := make([]string, 0, len(decoded.Result.Collections))
	for _, c := range decoded.Result.Collections {
		name := strings.TrimSpace(c.Name)
		if name != "" {
			out = append(out, name)
		}
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

func (q *Qdrant) CountPoints(ctx context.Context, collection string) (int, error) {
	base, err := q.baseURL()
	if err != nil {
		return 0, err
	}
	u, err := url.Parse(base)
	if err != nil {
		return 0, fmt.Errorf("invalid qdrant url: %w", err)
	}
	u.Path = fmt.Sprintf("%s/collections/%s/points/count", strings.TrimRight(u.Path, "/"), collection)

	body, err := json.Marshal(countRequest{Exact: true})
	if err != nil {
		return 0, fmt.Errorf("failed to marshal count request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", u.String(), bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("failed to create count request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := q.do(req)
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
