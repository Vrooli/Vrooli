package main

// DOC: docs/concepts/ARCHITECTURE.md#integrations
import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

var errCollectionNotFound = errors.New("collection not found")

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

func (s *Server) deleteQdrantCollectionHTTP(ctx context.Context, collection string) error {
	baseURL, err := url.Parse(strings.TrimRight(s.qdrantURL(), "/"))
	if err != nil {
		return fmt.Errorf("invalid qdrant url: %w", err)
	}
	baseURL.Path = fmt.Sprintf("%s/collections/%s", strings.TrimRight(baseURL.Path, "/"), collection)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, baseURL.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to create delete collection request: %w", err)
	}

	resp, err := s.qdrantDo(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("%w: %s", errCollectionNotFound, collection)
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("qdrant delete collection returned status %d", resp.StatusCode)
	}
	return nil
}
