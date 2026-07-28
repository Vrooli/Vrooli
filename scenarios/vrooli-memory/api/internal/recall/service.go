// Package recall implements the derived, read-only memory view. It owns no
// tables; journal and forest remain the authorities for its candidates.
package recall

import (
	"context"
	"sort"
	"strings"
)

type (
	Source interface {
		Nodes(context.Context) ([]Node, error)
	}
	QueryEmbedder interface {
		EmbedQuery(context.Context, string) ([]float64, error)
	}
	Service struct {
		source   Source
		embedder QueryEmbedder
		config   Config
	}
)

func NewService(source Source, embedder QueryEmbedder, config Config) *Service {
	if config.WakeBudget <= 0 {
		config.WakeBudget = 40
	}
	return &Service{source: source, embedder: embedder, config: config}
}

func (s *Service) Recall(ctx context.Context, query string, limit int) ([]Hit, error) {
	if limit <= 0 {
		limit = 10
	}
	nodes, err := s.source.Nodes(ctx)
	if err != nil {
		return nil, err
	}
	q, err := s.embedder.EmbedQuery(ctx, query)
	if err != nil {
		return nil, err
	}
	hits := make([]Hit, 0, len(nodes))
	for _, n := range nodes {
		hits = append(hits, Hit{Node: n, Score: cosine(q, n.Vector)})
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].Score > hits[j].Score })
	return collapse(hits, limit), nil
}

func (s *Service) Wake(ctx context.Context, budget int) (Wake, error) {
	if budget <= 0 {
		budget = s.config.WakeBudget
	}
	nodes, err := s.source.Nodes(ctx)
	if err != nil {
		return Wake{}, err
	}
	var pinned, frontier []Node
	for _, n := range nodes {
		if n.Pinned {
			pinned = append(pinned, n)
			continue
		}
		if n.Frontier {
			frontier = append(frontier, n)
		}
	}
	sort.SliceStable(pinned, func(i, j int) bool { return pinned[i].CreatedAt.Before(pinned[j].CreatedAt) })
	sort.SliceStable(frontier, func(i, j int) bool { return frontier[i].CreatedAt.After(frontier[j].CreatedAt) })
	out := Wake{Budget: budget}
	used := 0
	for _, n := range pinned {
		out.Hits = append(out.Hits, Hit{Node: n})
		used += lines(n.Text)
	}
	if used > budget {
		out.Overflow = true
		return out, nil
	}
	for _, n := range frontier {
		cost := lines(n.Text)
		if used+cost > budget {
			break
		}
		out.Hits = append(out.Hits, Hit{Node: n})
		used += cost
	}
	return out, nil
}

func (s *Service) Zoom(ctx context.Context, id string) ([]Node, error) {
	nodes, err := s.source.Nodes(ctx)
	if err != nil {
		return nil, err
	}
	var out []Node
	for _, n := range nodes {
		if n.ParentID == id {
			out = append(out, n)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

func collapse(hits []Hit, limit int) []Hit {
	selected := make([]Hit, 0, limit)
	for _, hit := range hits {
		if len(selected) == limit {
			break
		}
		ancestor := -1
		for i, existing := range selected {
			if isDescendant(hit.Node, existing.Node.ID, hits) {
				ancestor = i
				break
			}
		}
		if ancestor >= 0 {
			selected[ancestor].Descendants = append(selected[ancestor].Descendants, hit.Node)
			continue
		}
		selected = append(selected, hit)
	}
	return selected
}

func isDescendant(n Node, ancestor string, hits []Hit) bool {
	parent := n.ParentID
	for parent != "" {
		if parent == ancestor {
			return true
		}
		next := ""
		for _, h := range hits {
			if h.Node.ID == parent {
				next = h.Node.ParentID
				break
			}
		}
		parent = next
	}
	return false
}

func cosine(a, b []float64) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, aa, bb float64
	for i := range a {
		dot += a[i] * b[i]
		aa += a[i] * a[i]
		bb += b[i] * b[i]
	}
	if aa == 0 || bb == 0 {
		return 0
	}
	return dot / (sqrt(aa) * sqrt(bb))
}

func sqrt(v float64) float64 {
	if v == 0 {
		return 0
	}
	x := v
	for i := 0; i < 12; i++ {
		x = (x + v/x) / 2
	}
	return x
}

func lines(text string) int {
	n := strings.Count(strings.TrimSpace(text), "\n") + 1
	if strings.TrimSpace(text) == "" {
		return 0
	}
	return n
}
