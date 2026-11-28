package collections

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

// HTTPDoer is the resource-owned transport seam for read-only Qdrant inventory.
// Tests inject a fake implementation so they never connect to Qdrant.
type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

type Vector struct {
	Name     string `json:"name"`
	Size     int    `json:"dimensions"`
	Distance string `json:"distance,omitempty"`
}

type Collection struct {
	Name                string   `json:"name"`
	Aliases             []string `json:"aliases"`
	Status              string   `json:"status,omitempty"`
	Vectors             []Vector `json:"vectors"`
	PointsCount         *int64   `json:"points_count,omitempty"`
	IndexedVectorsCount *int64   `json:"indexed_vectors_count,omitempty"`
	SegmentsCount       *int64   `json:"segments_count,omitempty"`
}

type Inventory struct {
	Collections []Collection `json:"collections"`
}

type Client struct {
	BaseURL string
	APIKey  string
	HTTP    HTTPDoer
}

type collectionsResponse struct {
	Result struct {
		Collections []struct {
			Name string `json:"name"`
		} `json:"collections"`
	} `json:"result"`
}

type aliasesResponse struct {
	Result struct {
		Aliases []struct {
			AliasName      string `json:"alias_name"`
			CollectionName string `json:"collection_name"`
		} `json:"aliases"`
	} `json:"result"`
}

type collectionInfoResponse struct {
	Result struct {
		Status              string `json:"status"`
		PointsCount         *int64 `json:"points_count"`
		IndexedVectorsCount *int64 `json:"indexed_vectors_count"`
		SegmentsCount       *int64 `json:"segments_count"`
		Config              struct {
			Params struct {
				Vectors json.RawMessage `json:"vectors"`
			} `json:"params"`
		} `json:"config"`
	} `json:"result"`
}

func (c Client) List(ctx context.Context) (Inventory, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
	if baseURL == "" {
		return Inventory{}, errors.New("Qdrant base URL is required")
	}
	doer := c.HTTP
	if doer == nil {
		doer = http.DefaultClient
	}

	var listed collectionsResponse
	if err := c.getJSON(ctx, doer, baseURL+"/collections", &listed); err != nil {
		return Inventory{}, fmt.Errorf("list Qdrant collections: %w", err)
	}
	var aliases aliasesResponse
	// Qdrant exposes aliases at the root /aliases endpoint. A path beneath
	// /collections is interpreted as a collection lookup ("aliases") and
	// returns 404 on live Qdrant servers.
	if err := c.getJSON(ctx, doer, baseURL+"/aliases", &aliases); err != nil {
		return Inventory{}, fmt.Errorf("list Qdrant aliases: %w", err)
	}
	aliasesByCollection := make(map[string][]string)
	for _, alias := range aliases.Result.Aliases {
		name := strings.TrimSpace(alias.CollectionName)
		aliasName := strings.TrimSpace(alias.AliasName)
		if name != "" && aliasName != "" {
			aliasesByCollection[name] = append(aliasesByCollection[name], aliasName)
		}
	}
	for name := range aliasesByCollection {
		sort.Strings(aliasesByCollection[name])
	}

	out := Inventory{Collections: make([]Collection, 0, len(listed.Result.Collections))}
	for _, item := range listed.Result.Collections {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			return Inventory{}, errors.New("Qdrant collection inventory contains an empty collection name")
		}
		var info collectionInfoResponse
		endpoint := baseURL + "/collections/" + url.PathEscape(name)
		if err := c.getJSON(ctx, doer, endpoint, &info); err != nil {
			return Inventory{}, fmt.Errorf("inspect Qdrant collection %q: %w", name, err)
		}
		vectors, err := parseVectors(info.Result.Config.Params.Vectors)
		if err != nil {
			return Inventory{}, fmt.Errorf("inspect Qdrant collection %q vectors: %w", name, err)
		}
		out.Collections = append(out.Collections, Collection{
			Name:                name,
			Aliases:             aliasesByCollection[name],
			Status:              info.Result.Status,
			Vectors:             vectors,
			PointsCount:         info.Result.PointsCount,
			IndexedVectorsCount: info.Result.IndexedVectorsCount,
			SegmentsCount:       info.Result.SegmentsCount,
		})
	}
	sort.Slice(out.Collections, func(i, j int) bool { return out.Collections[i].Name < out.Collections[j].Name })
	return out, nil
}

func (c Client) getJSON(ctx context.Context, doer HTTPDoer, endpoint string, target any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	if key := strings.TrimSpace(c.APIKey); key != "" {
		request.Header.Set("api-key", key)
	}
	response, err := doer.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s returned HTTP %d", request.URL.Path, response.StatusCode)
	}
	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		return fmt.Errorf("decode GET %s response: %w", request.URL.Path, err)
	}
	return nil
}

func parseVectors(raw json.RawMessage) ([]Vector, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, errors.New("vector configuration is unavailable")
	}
	var unnamed struct {
		Size     int    `json:"size"`
		Distance string `json:"distance"`
	}
	if err := json.Unmarshal(raw, &unnamed); err != nil {
		return nil, err
	}
	if unnamed.Size > 0 {
		return []Vector{{Name: "default", Size: unnamed.Size, Distance: unnamed.Distance}}, nil
	}
	var named map[string]struct {
		Size     int    `json:"size"`
		Distance string `json:"distance"`
	}
	if err := json.Unmarshal(raw, &named); err != nil {
		return nil, err
	}
	if len(named) == 0 {
		return nil, errors.New("vector configuration has no dense vectors")
	}
	vectors := make([]Vector, 0, len(named))
	for name, config := range named {
		if config.Size <= 0 {
			return nil, fmt.Errorf("dense vector %q has no positive size", name)
		}
		vectors = append(vectors, Vector{Name: name, Size: config.Size, Distance: config.Distance})
	}
	sort.Slice(vectors, func(i, j int) bool { return vectors[i].Name < vectors[j].Name })
	return vectors, nil
}
