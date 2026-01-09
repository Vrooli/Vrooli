package graph

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
	Center     string
	Collection string
	Namespaces []string
	Visibility []string
	Tags       []string
	Depth      int
	Limit      int
	Threshold  float64
}

type Node struct {
	ID       string                 `json:"id"`
	Label    string                 `json:"label"`
	Score    *float64               `json:"score,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type Edge struct {
	Source       string  `json:"source"`
	Target       string  `json:"target"`
	Weight       float64 `json:"weight"`
	Relationship string  `json:"relationship"`
}

type Response struct {
	Center string `json:"center"`
	Nodes  []Node `json:"nodes"`
	Edges  []Edge `json:"edges"`
	TookMS int64  `json:"took_ms"`
}

const (
	defaultDepth     = 1
	maxDepth         = 3
	defaultLimit     = 25
	maxLimit         = 200
	defaultThreshold = 0.35
	maxExpandSeeds   = 10
	maxExpandPerSeed = 10
)

func (r *Request) NormalizeAndValidate() error {
	r.Center = strings.TrimSpace(r.Center)
	r.Collection = strings.TrimSpace(r.Collection)

	if r.Center == "" {
		return errors.New("center is required")
	}
	if r.Depth <= 0 {
		r.Depth = defaultDepth
	}
	if r.Depth > maxDepth {
		r.Depth = maxDepth
	}
	if r.Limit <= 0 {
		r.Limit = defaultLimit
	}
	if r.Limit > maxLimit {
		r.Limit = maxLimit
	}
	if r.Threshold <= 0 {
		r.Threshold = defaultThreshold
	}
	return nil
}

func (s *Service) Graph(ctx context.Context, req Request) (Response, error) {
	start := time.Now()

	if err := req.NormalizeAndValidate(); err != nil {
		return Response{}, err
	}
	if s == nil || s.VectorStore == nil || s.Embedder == nil {
		return Response{}, errors.New("graph service dependencies are not configured")
	}

	centerVec, err := s.Embedder.Embed(ctx, req.Center)
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

	filter := &ports.VectorFilter{
		Namespaces: req.Namespaces,
		Visibility: req.Visibility,
		Tags:       req.Tags,
	}
	if len(filter.Namespaces) == 0 && len(filter.Visibility) == 0 && len(filter.Tags) == 0 {
		filter = nil
	}

	type rawNode struct {
		Node
	}

	nodes := map[string]rawNode{}
	edges := map[string]Edge{}

	addNode := func(id, label string, score *float64, meta map[string]interface{}) {
		if strings.TrimSpace(id) == "" {
			return
		}
		if _, ok := nodes[id]; ok {
			return
		}
		nodes[id] = rawNode{Node: Node{ID: id, Label: label, Score: score, Metadata: meta}}
	}
	addEdge := func(source, target string, weight float64) {
		source = strings.TrimSpace(source)
		target = strings.TrimSpace(target)
		if source == "" || target == "" || source == target {
			return
		}
		key := source + "->" + target
		if _, ok := edges[key]; ok {
			return
		}
		edges[key] = Edge{
			Source:       source,
			Target:       target,
			Weight:       weight,
			Relationship: "semantic_similarity",
		}
	}

	addNode("center", req.Center, nil, map[string]interface{}{"type": "center"})

	seedResults := make([]ports.VectorSearchResult, 0, req.Limit)
	for _, coll := range collections {
		found, err := s.VectorStore.Search(ctx, coll, centerVec, req.Limit, req.Threshold, filter)
		if err != nil {
			continue
		}
		seedResults = append(seedResults, found...)
	}

	sort.SliceStable(seedResults, func(i, j int) bool { return seedResults[i].Score > seedResults[j].Score })
	if len(seedResults) > req.Limit {
		seedResults = seedResults[:req.Limit]
	}

	for _, r := range seedResults {
		score := r.Score
		label := extractContentFromPayload(r.Payload)
		if label == "" {
			label = r.ID
		}
		addNode(r.ID, label, &score, r.Payload)
		addEdge("center", r.ID, r.Score)
	}

	if req.Depth > 1 && len(seedResults) > 0 {
		maxSeeds := maxExpandSeeds
		if len(seedResults) < maxSeeds {
			maxSeeds = len(seedResults)
		}
		for i := 0; i < maxSeeds; i++ {
			seed := seedResults[i]
			seedLabel := extractContentFromPayload(seed.Payload)
			if seedLabel == "" {
				continue
			}
			seedVec, err := s.Embedder.Embed(ctx, seedLabel)
			if err != nil {
				continue
			}
			for _, coll := range collections {
				found, err := s.VectorStore.Search(ctx, coll, seedVec, maxExpandPerSeed, req.Threshold, filter)
				if err != nil {
					continue
				}
				for _, r := range found {
					if r.ID == seed.ID {
						continue
					}
					score := r.Score
					label := extractContentFromPayload(r.Payload)
					if label == "" {
						label = r.ID
					}
					addNode(r.ID, label, &score, r.Payload)
					addEdge(seed.ID, r.ID, r.Score)
				}
			}
		}
	}

	outNodes := make([]Node, 0, len(nodes))
	for _, n := range nodes {
		outNodes = append(outNodes, n.Node)
	}
	outEdges := make([]Edge, 0, len(edges))
	for _, e := range edges {
		outEdges = append(outEdges, e)
	}

	sort.SliceStable(outNodes, func(i, j int) bool { return outNodes[i].ID < outNodes[j].ID })
	sort.SliceStable(outEdges, func(i, j int) bool {
		if outEdges[i].Source == outEdges[j].Source {
			return outEdges[i].Target < outEdges[j].Target
		}
		return outEdges[i].Source < outEdges[j].Source
	})

	return Response{
		Center: req.Center,
		Nodes:  outNodes,
		Edges:  outEdges,
		TookMS: time.Since(start).Milliseconds(),
	}, nil
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
