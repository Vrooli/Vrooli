// Package recall implements the derived, read-only memory view. It owns no
// tables; journal and forest remain the authorities for its candidates.
package recall

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"slices"
	"sort"
	"strings"
	"unicode/utf8"

	aisearch "github.com/vrooli/ai-go/search"

	"source-ledger/internal/classification"
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
	GenerationIdentitySource interface {
		ActiveRecipeID(context.Context) (string, bool, error)
	}
	RecipeQueryEmbedder interface {
		EmbedQueryForRecipe(context.Context, string, string) ([]float64, error)
	}
	Service struct {
		source     Source
		embedder   QueryEmbedder
		config     Config
		registry   *policy.Registry
		categories ClassificationIndex
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

// Result is one recall answer. Lexical is set when semantic scoring was
// unavailable; the hits are then ranked by term overlap and Reason says why.
type Result struct {
	Hits                  []Hit
	Lexical               bool
	Reason                string
	ClassificationPending bool
	UnverifiedCaseMatches bool
}

var errRecipeEmbedder = errors.New("Source Ledger active embedding generation requires its exact-recipe query embedder")

func (s *Service) Recall(ctx context.Context, query string, limit int) (Result, error) {
	return s.RecallCategory(ctx, query, limit, "")
}

func (s *Service) RecallCategory(ctx context.Context, query string, limit int, filter string) (Result, error) {
	scope := string(policy.ScopeFromContext(ctx))
	selection := categoryQuery{topic: query}
	if s.categories != nil || strings.TrimSpace(filter) != "" {
		var categories []classification.Category
		if s.categories != nil {
			var err error
			categories, err = s.categories.Categories(ctx, scope)
			if err != nil {
				return Result{}, err
			}
		}
		var err error
		selection, err = resolveCategoryQuery(query, filter, categories)
		if err != nil {
			return Result{}, err
		}
	}
	categoryOnly := len(selection.ids) > 0 && strings.TrimSpace(selection.topic) == ""
	if limit <= 0 {
		limit = 10
	}
	var matches map[string][]string
	var classificationPending bool
	if len(selection.ids) > 0 {
		if coverage, ok := s.categories.(classificationCoverage); ok {
			var err error
			classificationPending, err = coverage.Coverage(ctx, scope, selection.ids)
			if err != nil {
				return Result{}, err
			}
		}
		var err error
		matches, err = s.categories.MatchingEntries(ctx, scope, selection.ids)
		if err != nil {
			return Result{}, err
		}
	}
	var recipeID string
	var generationActive bool
	var err error
	identitySource, tracksGeneration := s.source.(GenerationIdentitySource)
	if tracksGeneration && !categoryOnly {
		recipeID, generationActive, err = identitySource.ActiveRecipeID(ctx)
		if err != nil {
			return Result{}, err
		}
	}
	var exact RecipeQueryEmbedder
	if generationActive {
		var ok bool
		exact, ok = s.embedder.(RecipeQueryEmbedder)
		if !ok {
			return Result{}, errRecipeEmbedder
		}
	}
	// Query embedding, stored vectors, and text can be fetched independently.
	embedCtx, cancelEmbed := context.WithTimeout(ctx, aisearch.DefaultDenseBudget)
	defer cancelEmbed()
	type embedded struct {
		vec []float64
		err error
	}
	queryVec := make(chan embedded, 1)
	if !categoryOnly {
		go func() {
			var e embedded
			if generationActive {
				e.vec, e.err = exact.EmbedQueryForRecipe(embedCtx, recipeID, selection.topic)
			} else {
				e.vec, e.err = s.embedder.EmbedQuery(embedCtx, selection.topic)
			}
			queryVec <- e
		}()
	}
	loadNodes := s.source.Nodes
	vectorSource, separateVectors := s.source.(interface {
		NodeVectors(context.Context) (map[string][][]float64, error)
	})
	if categoryOnly || separateVectors {
		loadNodes = s.textNodes
	}
	if listingSource, ok := s.source.(interface {
		CategoryNodes(context.Context, map[string][]string) ([]Node, error)
	}); ok && categoryOnly {
		loadNodes = func(ctx context.Context) ([]Node, error) {
			return listingSource.CategoryNodes(ctx, matches)
		}
	}
	type storedVectors struct {
		vectors map[string][][]float64
		err     error
	}
	stored := make(chan storedVectors, 1)
	if separateVectors && !categoryOnly {
		go func() {
			vectors, err := vectorSource.NodeVectors(embedCtx)
			stored <- storedVectors{vectors: vectors, err: err}
		}()
	}
	nodes, err := loadNodes(ctx)
	if err != nil {
		return Result{}, err
	}
	var e embedded
	if !categoryOnly {
		select {
		case e = <-queryVec:
		case <-embedCtx.Done():
			select {
			case e = <-queryVec:
			default:
				e.err = embedCtx.Err()
			}
		}
		if e.err == nil && separateVectors {
			var loaded storedVectors
			select {
			case loaded = <-stored:
			case <-embedCtx.Done():
				select {
				case loaded = <-stored:
				default:
					loaded.err = embedCtx.Err()
				}
			}
			e.err = loaded.err
			setNodeVectors(nodes, loaded.vectors)
		}
	}
	if tracksGeneration && !categoryOnly {
		observedRecipeID, observedActive, err := identitySource.ActiveRecipeID(ctx)
		if err != nil {
			return Result{}, err
		}
		if observedRecipeID != recipeID || observedActive != generationActive {
			return Result{}, errors.New("Source Ledger embedding generation changed during recall; retry the request")
		}
	}
	const categoryBoost = 0.14
	const summaryBoost = 0.08
	parents := nodeParents(nodes)
	historical := historicalAccountNodes(nodes, parents)
	var labels map[string][]string
	result := Result{Lexical: e.err != nil, ClassificationPending: classificationPending}
	if len(selection.ids) > 0 {
		labels = inheritedCategories(nodes, matches, parents, historical)
	}
	scores := make([]float64, len(nodes))
	if !categoryOnly {
		scores = lexicalScores(selection.topic, nodes)
	}
	if result.Lexical {
		result.Reason = "semantic scoring unavailable: " + e.err.Error()
	} else if !categoryOnly {
		for i, node := range nodes {
			dense := bestSpaceScore(e.vec, node.Vectors)
			// Lexical evidence should improve a paraphrase match or carry
			// a recently captured observation that has no embedding yet.
			scores[i] = max(scores[i], dense*(0.9+0.1*dense))
		}
	}
	inherited := inheritCategoryRelevance(nodes, scores, labels, parents)
	var accountScores map[string]float64
	if historical != nil {
		accountScores = accountRelevance(nodes, scores)
	}
	hits := make([]Hit, 0, len(nodes))
	for i, node := range nodes {
		if historical[node.ID] {
			continue
		}
		node.CategoryIDs = labels[node.ID]
		if node.CaseID != "" && slices.Contains(selection.ids, "defect-report") &&
			!slices.Contains(node.CategoryIDs, "defect-report") {
			node.CategoryIDs = append(node.CategoryIDs, "defect-report")
		}
		categorized := len(node.CategoryIDs) > 0
		if (selection.hard || selection.openOnly || categoryOnly) && !categorized {
			continue
		}
		if categoryOnly && node.Summary && node.CaseID == "" {
			// Forest summaries can contain many unrelated topics. The
			// matching originals and case accounts give a precise listing.
			continue
		}
		if selection.openOnly && !node.Open {
			if node.CaseID == "" && !node.Summary && (categoryOnly || scores[i] >= 0.55) {
				result.UnverifiedCaseMatches = true
			}
			continue
		}
		score := max(scores[i], accountScores[node.ID])
		if categorized && node.Summary {
			score = max(score, inherited[node.ID])
		}
		if categoryOnly {
			score = 1
		} else if score <= 0 {
			continue
		}
		if categorized && !categoryOnly {
			score += categoryBoost
			if node.Summary {
				score += summaryBoost
			}
		}
		if len(selection.ids) > 0 && !categoryOnly {
			// Preserve the score range advertised to Search Hub.
			score /= 1 + (categoryBoost + summaryBoost)
		}
		hits = append(hits, Hit{Node: node, Score: score})
	}
	sort.Slice(hits, func(i, j int) bool {
		a, b := hits[i], hits[j]
		if a.Score != b.Score {
			return a.Score > b.Score
		}
		if !categoryOnly && a.Node.Depth != b.Node.Depth {
			return a.Node.Depth > b.Node.Depth
		}
		if !a.Node.CreatedAt.Equal(b.Node.CreatedAt) {
			return a.Node.CreatedAt.After(b.Node.CreatedAt)
		}
		return a.Node.ID < b.Node.ID
	})
	result.Hits = collapseWithParents(hits, limit, parents)
	return result, nil
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
	var kept []string
	if lead != "" {
		kept = append(kept, lead)
	}
	total := len(kept)
	for _, line := range strings.Split(body, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		total++
		if maxLines <= 0 || len(kept) < maxLines {
			kept = append(kept, strings.TrimRight(line, " \t"))
		}
	}
	result := strings.Join(kept, "\n")
	if result == "" || (total == len(kept) && (maxChars <= 0 || chars(result) <= maxChars)) {
		return result
	}
	marker := " " + truncationMarker
	if maxChars > 0 && maxChars < chars(marker) {
		return truncationMarker
	}
	if maxChars > 0 {
		result = truncateRunes(result, maxChars-chars(marker))
	}
	return strings.TrimRight(result, " \n") + marker
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

// splitLeadingMetadata selects prose from structured source text before the
// excerpt budget is spent on transport metadata. JSON content and frontmatter
// are rendering concerns only: journal and recall retain every original byte.
func splitLeadingMetadata(text string) (lead, body string) {
	trimmed := strings.TrimSpace(text)
	if strings.HasPrefix(trimmed, "{") {
		var object struct {
			Content string `json:"content"`
		}
		if json.Unmarshal([]byte(trimmed), &object) == nil && strings.TrimSpace(object.Content) != "" {
			return "", strings.TrimSpace(object.Content)
		}
	}
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
			if value, ok := strings.CutPrefix(strings.TrimSpace(line), field); ok && strings.TrimSpace(value) != "" {
				return strings.Trim(strings.TrimSpace(value), `"'`), after
			}
		}
	}
	return "", after
}

func (s *Service) textNodes(ctx context.Context) ([]Node, error) {
	if source, ok := s.source.(interface {
		TextNodes(context.Context) ([]Node, error)
	}); ok {
		return source.TextNodes(ctx)
	}
	return s.source.Nodes(ctx)
}

func (s *Service) Zoom(ctx context.Context, id string) ([]Node, error) {
	if strings.HasPrefix(id, accountPrefix) {
		if source, ok := s.source.(interface {
			AccountEvidence(context.Context, string) ([]Node, error)
		}); ok {
			return source.AccountEvidence(ctx, id)
		}
	}
	nodes, err := s.textNodes(ctx)
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
	parents := make(map[string]string, len(hits))
	for _, hit := range hits {
		parents[hit.Node.ID] = hit.Node.ParentID
	}
	return collapseWithParents(hits, limit, parents)
}

func collapseWithParents(hits []Hit, limit int, parents map[string]string) []Hit {
	selected := make([]Hit, 0, min(limit, len(hits)))
	for _, hit := range hits {
		if len(selected) == limit {
			break
		}
		ancestor := -1
		for i, existing := range selected {
			if existing.Node.contains(hit.Node) || isDescendant(hit.Node, existing.Node.ID, parents) {
				ancestor = i
				break
			}
			// A more relevant descendant may have been selected before its
			// ancestor. Keep that best node and suppress the ancestor: returning
			// both would violate the antichain contract for recall results.
			if hit.Node.contains(existing.Node) || isDescendant(existing.Node, hit.Node.ID, parents) {
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

func isDescendant(n Node, ancestor string, parents map[string]string) bool {
	parent := n.ParentID
	// Forest data is expected acyclic; this bound also prevents corrupt lineage
	// from trapping a recall request indefinitely.
	for steps := 0; parent != "" && steps < len(parents); steps++ {
		if parent == ancestor {
			return true
		}
		parent = parents[parent]
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
	return dot / (math.Sqrt(aa) * math.Sqrt(bb))
}

func lines(text string) int {
	n := strings.Count(strings.TrimSpace(text), "\n") + 1
	if strings.TrimSpace(text) == "" {
		return 0
	}
	return n
}
