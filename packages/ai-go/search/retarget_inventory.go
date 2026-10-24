package aisearch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

// EmbeddingInventoryEntry is the read-only, operator-facing view of one
// Qdrant collection's embedding contract. It is deliberately separate from
// CollectionSpec: inventory must never imply that an observed collection is
// safe to mutate.
type EmbeddingInventoryEntry struct {
	Store           EmbeddingStore    `json:"store"`
	Metadata        EmbeddingMetadata `json:"metadata"`
	Distance        string            `json:"distance,omitempty"`
	PointCount      int64             `json:"point_count"`
	SentinelPresent bool              `json:"sentinel_present"`
}

type qdrantCollectionsResponse struct {
	Result struct {
		Collections []struct {
			Name string `json:"name"`
		} `json:"collections"`
	} `json:"result"`
}

type qdrantCollectionInfoResponse struct {
	Result struct {
		Config struct {
			Params struct {
				Vectors json.RawMessage `json:"vectors"`
			} `json:"params"`
		} `json:"config"`
	} `json:"result"`
}

type qdrantPointResponse struct {
	Result struct {
		Payload map[string]any `json:"payload"`
	} `json:"result"`
}

type qdrantCountResponse struct {
	Result struct {
		Count int64 `json:"count"`
	} `json:"result"`
}

// InventoryQdrantCollections reads all collections and their engine metadata
// sentinels. It is intentionally read-only and returns an error for transport,
// malformed, or unavailable Qdrant responses instead of reporting partial
// metadata as a trustworthy migration inventory.
func InventoryQdrantCollections(ctx context.Context, baseURL, apiKey string) ([]EmbeddingInventoryEntry, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = DefaultQdrantURL
	}
	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/collections", nil)
	if err != nil {
		return nil, fmt.Errorf("create Qdrant collection request: %w", err)
	}
	setAPIKey(req, apiKey)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list Qdrant collections: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list Qdrant collections: status %d: %s", resp.StatusCode, readBodyPrefix(resp.Body))
	}
	var collections qdrantCollectionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&collections); err != nil {
		return nil, fmt.Errorf("decode Qdrant collections: %w", err)
	}

	out := make([]EmbeddingInventoryEntry, 0, len(collections.Result.Collections))
	for _, collection := range collections.Result.Collections {
		name := strings.TrimSpace(collection.Name)
		if name == "" {
			continue
		}
		entry, err := inspectQdrantCollection(ctx, client, baseURL, apiKey, name)
		if err != nil {
			return nil, err
		}
		out = append(out, entry)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Store.ID < out[j].Store.ID })
	return out, nil
}

func inspectQdrantCollection(ctx context.Context, client *http.Client, baseURL, apiKey, collection string) (EmbeddingInventoryEntry, error) {
	info, err := qdrantGET[qdrantCollectionInfoResponse](ctx, client, baseURL+"/collections/"+url.PathEscape(collection), apiKey, "collection info")
	if err != nil {
		return EmbeddingInventoryEntry{}, err
	}
	size, distance := parseQdrantVectorLayout(info.Result.Config.Params.Vectors)
	metaID := PointIDFor(metaIDPrefix, collection, 0, 1)
	point, err := qdrantGET[qdrantPointResponse](ctx, client, baseURL+"/collections/"+url.PathEscape(collection)+"/points/"+url.PathEscape(metaID), apiKey, "metadata sentinel")
	if err != nil && !strings.Contains(err.Error(), "status 404") {
		return EmbeddingInventoryEntry{}, err
	}
	payload := point.Result.Payload
	model := stringPayload(payload, metaModelKey)
	role := stringPayload(payload, metaRoleKey)
	policy := stringPayload(payload, metaPolicySchemaKey)
	metaSize := intPayload(payload, metaDenseSizeKey)
	if metaSize > 0 {
		size = metaSize
	}
	count, err := qdrantCollectionCount(ctx, client, baseURL, apiKey, collection)
	if err != nil {
		return EmbeddingInventoryEntry{}, err
	}
	if count > 0 {
		count-- // the engine sentinel is not a corpus point
	}
	return EmbeddingInventoryEntry{
		Store:           NewQdrantEmbeddingStore(collection),
		Metadata:        EmbeddingMetadata{Role: role, Model: model, Dimensions: size, PolicySchemaVersion: policy},
		Distance:        firstNonEmpty(stringPayload(payload, metaDenseDistanceKey), distance),
		PointCount:      count,
		SentinelPresent: payload != nil && payload[metaMarkerKey] == true,
	}, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func qdrantCollectionCount(ctx context.Context, client *http.Client, baseURL, apiKey, collection string) (int64, error) {
	body := strings.NewReader(`{"exact":true}`)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/collections/"+url.PathEscape(collection)+"/points/count", body)
	if err != nil {
		return 0, fmt.Errorf("create Qdrant point-count request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	setAPIKey(req, apiKey)
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("count Qdrant collection %q: %w", collection, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("count Qdrant collection %q: status %d: %s", collection, resp.StatusCode, readBodyPrefix(resp.Body))
	}
	var decoded qdrantCountResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return 0, fmt.Errorf("decode Qdrant point count: %w", err)
	}
	return decoded.Result.Count, nil
}

func qdrantGET[T any](ctx context.Context, client *http.Client, endpoint, apiKey, operation string) (T, error) {
	var out T
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return out, fmt.Errorf("create Qdrant %s request: %w", operation, err)
	}
	setAPIKey(req, apiKey)
	resp, err := client.Do(req)
	if err != nil {
		return out, fmt.Errorf("read Qdrant %s: %w", operation, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return out, fmt.Errorf("read Qdrant %s: status %d: %s", operation, resp.StatusCode, readBodyPrefix(resp.Body))
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return out, fmt.Errorf("decode Qdrant %s: %w", operation, err)
	}
	return out, nil
}

func setAPIKey(req *http.Request, apiKey string) {
	if apiKey = strings.TrimSpace(apiKey); apiKey != "" {
		req.Header.Set("api-key", apiKey)
	}
}

func readBodyPrefix(r io.Reader) string {
	const max = 512
	b, _ := io.ReadAll(io.LimitReader(r, max))
	return strings.TrimSpace(string(b))
}

func parseQdrantVectorLayout(raw json.RawMessage) (int, string) {
	if len(raw) == 0 {
		return 0, ""
	}
	var named map[string]struct {
		Size     int    `json:"size"`
		Distance string `json:"distance"`
	}
	if json.Unmarshal(raw, &named) == nil {
		if dense, ok := named["dense"]; ok {
			return dense.Size, dense.Distance
		}
		for _, vector := range named {
			return vector.Size, vector.Distance
		}
	}
	var legacy struct {
		Size     int    `json:"size"`
		Distance string `json:"distance"`
	}
	if json.Unmarshal(raw, &legacy) == nil {
		return legacy.Size, legacy.Distance
	}
	return 0, ""
}
