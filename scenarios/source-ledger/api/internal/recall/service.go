// Package recall implements the derived, read-only memory view. It owns no
// tables; journal and forest remain the authorities for its candidates.
package recall

import (
	"context"
	"sort"
	"strings"
	"unicode/utf8"

	"source-ledger/internal/policy"
)

type (
	Source interface {
		Nodes(context.Context) ([]Node, error)
		// AmbientNodes returns only pinned and frontier nodes, without vectors.
		// Wake never scores similarity, so making it pay for the whole embedding
		// corpus made session start cost grow with total memories rather than
		// with the frontier it actually renders.
		AmbientNodes(ctx context.Context, perFacetLimit int) ([]Node, error)
	}
	QueryEmbedder interface {
		EmbedQuery(context.Context, string) ([]float64, error)
	}
	Service struct {
		source   Source
		embedder QueryEmbedder
		config   Config
		registry *policy.Registry
	}
)

func NewService(source Source, embedder QueryEmbedder, config Config) *Service {
	if config.WakeBudget <= 0 {
		config.WakeBudget = DefaultWakeBudget
	}
	if config.MaxEntryLines <= 0 {
		config.MaxEntryLines = DefaultMaxEntryLines
	}
	if config.WakeBudgetChars <= 0 {
		config.WakeBudgetChars = DefaultWakeBudgetChars
	}
	if config.MaxEntryChars <= 0 {
		config.MaxEntryChars = DefaultMaxEntryChars
	}
	return &Service{source: source, embedder: embedder, config: config}
}

func (s *Service) SetPolicyRegistry(registry *policy.Registry) { s.registry = registry }

func (s *Service) configFor(ctx context.Context) (Config, error) {
	if config, ok := policy.ConfigFromContext(ctx); ok {
		return configFromPolicy(config, s.config), nil
	}
	if s.registry != nil {
		config, err := s.registry.Resolve(ctx, string(policy.ScopeFromContext(ctx)))
		if err != nil {
			return Config{}, err
		}
		return configFromPolicy(config, s.config), nil
	}
	return s.config, nil
}

func configFromPolicy(config policy.Config, fallback Config) Config {
	fallback.FrontierTarget = config.FrontierTarget
	fallback.WakeBudget = config.WakeBudget
	fallback.WakeBudgetChars = config.WakeBudgetChars
	fallback.MaxEntryLines = config.MaxEntryLines
	fallback.MaxEntryChars = config.MaxEntryChars
	if config.FacetBudgets != nil {
		fallback.FacetBudgets = config.FacetBudgets
	}
	return fallback
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
		hits = append(hits, Hit{Node: n, Score: bestSpaceScore(q, n.Vectors)})
	}
	sort.SliceStable(hits, func(i, j int) bool { return hits[i].Score > hits[j].Score })
	return collapse(hits, limit), nil
}

func (s *Service) Wake(ctx context.Context, budget int) (Wake, error) {
	config, err := s.configFor(ctx)
	if err != nil {
		return Wake{}, err
	}
	if budget <= 0 {
		budget = config.WakeBudget
	}
	// Only the newest perFacetLimit memories of a facet can ever be emitted, so
	// there is no reason to carry the rest of the frontier's bodies into memory.
	nodes, err := s.source.AmbientNodes(ctx, perFacetLimit(config))
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
	out := Wake{BudgetLines: budget, BudgetChars: config.WakeBudgetChars}
	usedLines, usedChars := 0, 0
	admit := func(n Node) {
		hit := excerpt(n, config.MaxEntryLines, config.MaxEntryChars)
		lineCost, charCost := lines(hit.Text), chars(hit.Text)
		if usedLines+lineCost > budget || usedChars+charCost > config.WakeBudgetChars {
			out.Overflow = true
			out.Refused++
			return
		}
		out.Hits = append(out.Hits, Hit{Node: hit})
		usedLines += lineCost
		usedChars += charCost
	}
	// Pins retain ordering privilege, but they are not exempt from the global
	// ceilings. The oldest pin is admitted first; later pins are lower priority
	// when the ambient view cannot contain them all.
	for _, n := range pinned {
		admit(n)
	}
	if len(config.FacetBudgets) == 0 {
		for _, n := range frontier {
			admit(n)
		}
		out.LinesUsed, out.CharsUsed = usedLines, usedChars
		return out, nil
	}
	byFacet := map[string][]Node{}
	for _, n := range frontier {
		byFacet[n.FacetID] = append(byFacet[n.FacetID], n)
	}
	facets := make([]string, 0, len(byFacet))
	for facet := range byFacet {
		facets = append(facets, facet)
	}
	sort.Strings(facets)
	for _, facet := range facets {
		// A facet's resident budget counts entries. Measuring it in lines made
		// every facet whose newest memory was longer than its ceiling emit
		// nothing at all, which is why ambient recall collapsed to one facet.
		ceiling := config.FacetBudgets[facet]
		if ceiling <= 0 {
			continue
		}
		taken := 0
		for _, n := range byFacet[facet] {
			if taken >= ceiling {
				break
			}
			before := len(out.Hits)
			admit(n)
			if len(out.Hits) == before {
				// Skip this memory rather than abandoning the facet: a later
				// entry may still fit, and one oversized memory must not cost a
				// facet its whole residency.
				continue
			}
			taken++
		}
	}
	out.LinesUsed, out.CharsUsed = usedLines, usedChars
	return out, nil
}

// perFacetLimit is the largest residency any facet declares. Pins are ordered
// before frontier nodes but are still admitted through Wake's global bounds.
func perFacetLimit(config Config) int {
	limit := 0
	for _, ceiling := range config.FacetBudgets {
		if ceiling > limit {
			limit = ceiling
		}
	}
	if limit <= 0 {
		// Without declared residencies wake walks one recency-ordered list, so
		// the line budget is the only bound that applies.
		limit = config.WakeBudget
	}
	return limit
}

func (s *Service) perFacetLimit() int { return perFacetLimit(s.config) }

// excerpt bounds one memory's contribution to the ambient view. The journal
// keeps the full text; recall returns it in full. Wake is a fixed-size index
// into memory, so it shows a bounded head and marks that it did.
func excerpt(n Node, maxEntryLines, maxEntryChars int) Node {
	if maxEntryLines <= 0 && maxEntryChars <= 0 {
		return n
	}
	n.Text = excerptText(n.Text, maxEntryLines, maxEntryChars)
	return n
}

const truncationMarker = "…"

func excerptText(text string, maxLines int, charLimits ...int) string {
	maxChars := 0
	if len(charLimits) > 0 {
		maxChars = charLimits[0]
	}
	lead, body := splitLeadingMetadata(text)
	if maxLines <= 0 {
		maxLines = int(^uint(0) >> 1)
	}
	kept := make([]string, 0, maxLines)
	if lead != "" {
		kept = append(kept, lead)
	}
	total := len(kept)
	for _, line := range strings.Split(strings.TrimSpace(body), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		total++
		if len(kept) < maxLines {
			kept = append(kept, strings.TrimRight(line, " \t"))
		}
	}
	if len(kept) == 0 {
		return ""
	}
	result := strings.Join(kept, "\n")
	truncated := total > len(kept)
	if maxChars > 0 && chars(result) > maxChars {
		truncated = true
		result = truncateRunes(result, maxChars)
	}
	if truncated {
		marker := " " + truncationMarker
		if maxChars > 0 {
			available := maxChars - chars(marker)
			if available < 0 {
				available = 0
			}
			result = truncateRunes(result, available)
		}
		result = strings.TrimRight(result, " \n") + marker
		if maxChars > 0 && chars(result) > maxChars {
			result = truncationMarker
		}
	}
	return result
}

func chars(text string) int { return utf8.RuneCountInString(text) }

func truncateRunes(text string, limit int) string {
	if limit <= 0 {
		return ""
	}
	count := 0
	for index := range text {
		if count == limit {
			return text[:index]
		}
		count++
	}
	return text
}

// splitLeadingMetadata separates a leading `---` fenced metadata block from the
// prose after it, returning the block's most descriptive field as the lead.
// Memories are stored verbatim, so a file that opens with frontmatter would
// otherwise spend its whole excerpt on a delimiter and a slug. This is a
// rendering concern only: the journal keeps every byte and recall returns them.
func splitLeadingMetadata(text string) (lead, body string) {
	trimmed := strings.TrimSpace(text)
	if !strings.HasPrefix(trimmed, "---\n") {
		return "", trimmed
	}
	rest := trimmed[len("---\n"):]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return "", trimmed
	}
	block, after := rest[:end], strings.TrimSpace(rest[end+len("\n---"):])
	// Prefer a human-written summary over the slug when the block offers both.
	for _, field := range []string{"description:", "name:"} {
		for _, line := range strings.Split(block, "\n") {
			if value := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), field)); strings.HasPrefix(strings.TrimSpace(line), field) && value != "" {
				return strings.Trim(value, `"'`), after
			}
		}
	}
	return "", after
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
			// A more relevant descendant may have been selected before its
			// ancestor. Keep that best node and suppress the ancestor: returning
			// both would violate the antichain contract for recall results.
			if isDescendant(existing.Node, hit.Node.ID, hits) {
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

// bestSpaceScore scores a query against every derived facet space and keeps the
// strongest. The spaces share a model and differ only in framing, so a memory
// whose entities space matches the query still surfaces when its topic space
// does not.
func bestSpaceScore(query []float64, vectors [][]float64) float64 {
	best := 0.0
	for _, v := range vectors {
		if score := cosine(query, v); score > best {
			best = score
		}
	}
	return best
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
