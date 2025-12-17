package search

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"knowledge-observatory/internal/ports"
)

type Service struct {
	VectorStore ports.VectorStore
	Embedder    ports.Embedder
}

type Request struct {
	Query      string
	Collection string
	Limit      int
	Threshold  float64
}

type Result struct {
	ID       string
	Score    float64
	Content  string
	Metadata map[string]interface{}
}

type Response struct {
	Results []Result
	Query   string
	TookMS  int64
}

const (
	defaultSearchLimit     = 10
	maxSearchLimit         = 100
	defaultSearchThreshold = 0.3
)

func (r *Request) NormalizeAndValidate() error {
	r.Query = strings.TrimSpace(r.Query)
	r.Collection = strings.TrimSpace(r.Collection)

	if r.Query == "" {
		return errors.New("Query parameter is required")
	}
	if r.Limit <= 0 {
		r.Limit = defaultSearchLimit
	}
	if r.Limit > maxSearchLimit {
		r.Limit = maxSearchLimit
	}
	if r.Threshold <= 0 {
		r.Threshold = defaultSearchThreshold
	}
	return nil
}

func extractContentFromPayload(payload map[string]interface{}) string {
	if payload == nil {
		return ""
	}
	if c, ok := payload["content"].(string); ok {
		return c
	}
	if c, ok := payload["text"].(string); ok {
		return c
	}
	return ""
}

func (s *Service) Search(ctx context.Context, req Request) (Response, error) {
	start := time.Now()

	if err := req.NormalizeAndValidate(); err != nil {
		return Response{}, err
	}
	if s == nil || s.VectorStore == nil || s.Embedder == nil {
		return Response{}, errors.New("search service dependencies are not configured")
	}

	embedding, err := s.Embedder.Embed(ctx, req.Query)
	if err != nil {
		return Response{}, fmt.Errorf("embedding generation failed: %w", err)
	}

	collections, err := s.VectorStore.ListCollections(ctx)
	if err != nil {
		return Response{}, fmt.Errorf("failed to list collections: %w", err)
	}

	if req.Collection != "" {
		found := false
		for _, c := range collections {
			if c == req.Collection {
				found = true
				break
			}
		}
		if !found {
			return Response{}, fmt.Errorf("collection %s not found", req.Collection)
		}
		collections = []string{req.Collection}
	}

	all := make([]Result, 0, req.Limit*len(collections))
	for _, coll := range collections {
		found, err := s.VectorStore.Search(ctx, coll, embedding, req.Limit, req.Threshold)
		if err != nil {
			continue
		}
		for _, r := range found {
			all = append(all, Result{
				ID:       r.ID,
				Score:    r.Score,
				Content:  extractContentFromPayload(r.Payload),
				Metadata: r.Payload,
			})
		}
	}

	sort.SliceStable(all, func(i, j int) bool {
		return all[i].Score > all[j].Score
	})
	if len(all) > req.Limit {
		all = all[:req.Limit]
	}

	return Response{
		Results: all,
		Query:   req.Query,
		TookMS:  time.Since(start).Milliseconds(),
	}, nil
}

