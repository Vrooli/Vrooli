package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type qdrantCollectionsResponse struct {
	Result struct {
		Collections []struct {
			Name string `json:"name"`
		} `json:"collections"`
	} `json:"result"`
	Status string `json:"status"`
}

func (s *Server) listQdrantCollectionsHTTP(ctx context.Context) ([]string, error) {
	baseURL, err := url.Parse(strings.TrimRight(s.qdrantURL(), "/"))
	if err != nil {
		return nil, fmt.Errorf("invalid qdrant url: %w", err)
	}
	baseURL.Path = fmt.Sprintf("%s/collections", strings.TrimRight(baseURL.Path, "/"))

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create collections request: %w", err)
	}

	resp, err := s.qdrantDo(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("qdrant collections returned status %d", resp.StatusCode)
	}

	var decoded qdrantCollectionsResponse
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

