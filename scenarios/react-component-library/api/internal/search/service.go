package search

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	searchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/search"
)

// AvailabilityFunc resolves whether one catalog asset is built and gives the
// reader a short label for it. The search package intentionally does not import
// the components service: availability is an injected seam so Reindex can be
// tested without a registry and so the catalog owner remains the only resolver.
type AvailabilityFunc func(ctx context.Context, catalogID string) (built bool, label string, err error)

// Semantic is the seam for the shared packages/ai-go/search hybrid engine.
// This package deliberately ships no embedding/vector implementation. Until the
// shared engine is wired, the leg is reported DISABLED and lexical results are
// returned unchanged — never a fabricated "semantic" result.
type Semantic interface {
	Enabled() bool
}

// Service is the lexical retrieval service over the search projection.
type Service struct {
	db           *sql.DB
	scenarioRoot string
	availability AvailabilityFunc
	semantic     Semantic

	// reindexMu serializes rebuilds. Search reads keep seeing the previous
	// projection until the rebuild transaction commits.
	reindexMu sync.Mutex
}

// NewService builds the service for a scenario. scenarioRoot is the scenario
// directory (the parent of catalog/), not the repository root; callers that
// have only the repo root should pass `<repoRoot>/scenarios/react-component-library`.
func NewService(db *sql.DB, scenarioRoot string, availability AvailabilityFunc) *Service {
	return &Service{db: db, scenarioRoot: scenarioRoot, availability: availability}
}

// Reindex is the boot/setup helper for callers that do not otherwise need a
// Service value. It uses the same implementation as (*Service).Reindex so main
// wiring cannot drift from the RPC.
func Reindex(ctx context.Context, db *sql.DB, scenarioRoot string, availability AvailabilityFunc, force bool) (int, time.Time, error) {
	return NewService(db, scenarioRoot, availability).Reindex(ctx, force)
}

// WithSemantic attaches the optional semantic leg. A nil leg means DISABLED.
func (s *Service) WithSemantic(semantic Semantic) *Service {
	if s != nil {
		s.semantic = semantic
	}
	return s
}

// DB exposes the handle for wiring tests; production callers already own it.
func (s *Service) DB() *sql.DB {
	if s == nil {
		return nil
	}
	return s.db
}

const (
	defaultSearchLimit  = 20
	maxSearchLimit      = 100
	defaultSuggestLimit = 8
	maxSuggestLimit     = 25
)

// Search answers one ranked query. It applies the requested kind filter before
// the limit and reports the pre-limit match count, so the UI can say "showing
// 20 of 137" honestly.
func (s *Service) Search(ctx context.Context, req *searchv1.SearchRequest) (*searchv1.SearchResponse, error) {
	started := time.Now()
	if s == nil || s.db == nil {
		return nil, ErrNotConfigured
	}
	if req == nil {
		req = &searchv1.SearchRequest{}
	}
	plan := planQuery(req.GetQuery())
	kinds := requestedKinds(req.GetKinds())

	var candidates []candidate
	var err error
	browse := plan.empty()
	if browse {
		candidates, err = s.browse(ctx)
	} else {
		candidates, err = s.match(ctx, plan)
	}
	if err != nil {
		return nil, err
	}
	candidates = dedupeCandidates(candidates)
	if len(kinds) > 0 {
		candidates = filterKinds(candidates, kinds)
	} else if browse {
		// The curated browse set is the five top-level destinations. Displays,
		// states and evidence are addressable through their parent page or the
		// Review queue, so they are omitted unless a caller explicitly filters
		// for them.
		candidates = filterKinds(candidates, defaultBrowseKinds)
	}

	total := len(candidates)
	facets := buildFacets(candidates)
	if browse {
		sortBrowse(candidates)
	} else {
		sortRanked(candidates)
	}
	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = defaultSearchLimit
	}
	if limit > maxSearchLimit {
		limit = maxSearchLimit
	}
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}

	status, err := s.Status(ctx)
	if err != nil {
		return nil, err
	}
	results := make([]*searchv1.SearchResult, 0, len(candidates))
	for _, cand := range candidates {
		results = append(results, &searchv1.SearchResult{
			Id:           cand.doc.ID,
			Kind:         cand.doc.Kind,
			Title:        cand.doc.Title,
			Subtitle:     cand.doc.Subtitle,
			Snippet:      snippet(cand.doc.Body, plan.words, 160),
			Href:         cand.doc.Href,
			Score:        cand.score,
			Availability: cand.doc.Availability,
			Badges:       cand.doc.Badges,
			Relations:    cand.doc.Relations,
		})
	}
	return &searchv1.SearchResponse{
		Query:   req.GetQuery(),
		Results: results,
		Facets:  facets,
		Total:   int32(total),
		TookMs:  time.Since(started).Milliseconds(),
		Status:  status,
	}, nil
}

// Suggest completes from indexed titles and tags. It intentionally queries the
// projection rather than the query grammar: a completion is a known string from
// the corpus, not a new search expression.
func (s *Service) Suggest(ctx context.Context, req *searchv1.SuggestRequest) (*searchv1.SuggestResponse, error) {
	if s == nil || s.db == nil {
		return nil, ErrNotConfigured
	}
	if req == nil {
		req = &searchv1.SuggestRequest{}
	}
	prefix := strings.ToLower(strings.TrimSpace(req.GetQuery()))
	if prefix == "" {
		return &searchv1.SuggestResponse{}, nil
	}
	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = defaultSuggestLimit
	}
	if limit > maxSuggestLimit {
		limit = maxSuggestLimit
	}
	rows, err := s.db.QueryContext(ctx, `SELECT title, kind, badges_json FROM search_docs`)
	if err != nil {
		if isMissingTable(err) {
			return nil, ErrIndexNotInitialized
		}
		return nil, fmt.Errorf("read search suggestions: %w", err)
	}
	defer rows.Close()

	type suggestion struct {
		text   string
		prefix bool
		hits   int
		kind   searchv1.SearchKind
	}
	seen := map[string]*suggestion{}
	add := func(text string, kind searchv1.SearchKind) {
		text = strings.TrimSpace(text)
		if text == "" {
			return
		}
		lower := strings.ToLower(text)
		startsWith := strings.HasPrefix(lower, prefix)
		if !startsWith && !strings.Contains(lower, prefix) {
			return
		}
		key := lower
		item, ok := seen[key]
		if !ok {
			seen[key] = &suggestion{text: text, prefix: startsWith, hits: 1, kind: kind}
			return
		}
		item.hits++
		if startsWith {
			item.prefix = true
		}
		if kindPrior(KindText(kind)) > kindPrior(KindText(item.kind)) {
			item.kind = kind
		}
	}
	for rows.Next() {
		var title, kindText, badgesJSON string
		if err := rows.Scan(&title, &kindText, &badgesJSON); err != nil {
			return nil, fmt.Errorf("scan search suggestion: %w", err)
		}
		kind := ParseKind(kindText)
		add(title, kind)
		for _, badge := range unmarshalStrings(badgesJSON) {
			add(badge, kind)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read search suggestions: %w", err)
	}

	items := make([]*suggestion, 0, len(seen))
	for _, item := range seen {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].prefix != items[j].prefix {
			return items[i].prefix
		}
		if items[i].hits != items[j].hits {
			return items[i].hits > items[j].hits
		}
		if len(items[i].text) != len(items[j].text) {
			return len(items[i].text) < len(items[j].text)
		}
		return strings.ToLower(items[i].text) < strings.ToLower(items[j].text)
	})
	if len(items) > limit {
		items = items[:limit]
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.text)
	}
	return &searchv1.SuggestResponse{Suggestions: out}, nil
}

// Status reports the projection's own state. indexed is false for an empty
// projection even when a successful reindex wrote indexed_at: an index that can
// answer nothing is not "indexed" from the reader's point of view.
func (s *Service) Status(ctx context.Context) (*searchv1.SearchStatus, error) {
	if s == nil || s.db == nil {
		return nil, ErrNotConfigured
	}
	state, err := s.readState(ctx)
	if err != nil {
		return nil, err
	}
	count := 0
	if raw := strings.TrimSpace(state[stateDocumentCount]); raw != "" {
		if parsed, parseErr := strconv.Atoi(raw); parseErr == nil {
			count = parsed
		}
	}
	indexedAt := strings.TrimSpace(state[stateIndexedAt])
	semantic := searchv1.SemanticLegState_SEMANTIC_LEG_STATE_DISABLED
	if s.semantic != nil && s.semantic.Enabled() {
		semantic = searchv1.SemanticLegState_SEMANTIC_LEG_STATE_READY
	}
	return &searchv1.SearchStatus{
		Indexed:       count > 0 && indexedAt != "",
		DocumentCount: int32(count),
		IndexedAt:     indexedAt,
		Stale: strings.TrimSpace(state[stateLastError]) != "" ||
			strings.TrimSpace(state[stateProjectionVersion]) != projectionVersion,
		LastError:   state[stateLastError],
		SemanticLeg: semantic,
	}, nil
}

func (s *Service) readState(ctx context.Context) (map[string]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT key, value FROM search_state`)
	if err != nil {
		if isMissingTable(err) {
			// Status must stay answerable before the first index exists.
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("read search state: %w", err)
	}
	defer rows.Close()
	state := map[string]string{}
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("scan search state: %w", err)
		}
		state[key] = value
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read search state: %w", err)
	}
	return state, nil
}

type candidate struct {
	doc   Document
	score float64
}

type rawRow struct {
	doc  Document
	rank float64
}

const searchSelect = `
SELECT d.id, d.kind, d.subject_id, d.title, d.subtitle, d.body, d.href,
       d.availability, d.badges_json, d.relations_json, d.updated_at, `

// match runs the AND query first and falls back to OR only when AND returned
// nothing. Prefix matching is applied per token, so "land" already finds
// "LandingPage" without the fallback; the fallback exists for multi-word queries
// where one word is absent (for example "onboarding wizard").
func (s *Service) match(ctx context.Context, plan queryPlan) ([]candidate, error) {
	rows, err := s.ftsRows(ctx, plan.andQuery)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 && plan.orQuery != "" && plan.orQuery != plan.andQuery {
		if rows, err = s.ftsRows(ctx, plan.orQuery); err != nil {
			return nil, err
		}
	}
	if len(rows) == 0 {
		return nil, nil
	}
	lowerTitle := make([]string, len(rows))
	for i := range rows {
		lowerTitle[i] = strings.ToLower(strings.TrimSpace(rows[i].doc.Title))
	}
	out := make([]candidate, 0, len(rows))
	for i, row := range rows {
		base := -row.rank
		if base <= 0 {
			base = 1
		}
		title := lowerTitle[i]
		score := base * kindPrior(KindText(row.doc.Kind)) * availabilityFactor(AvailabilityText(row.doc.Availability))
		switch {
		case title == plan.normalized:
			score *= 1.6
		case strings.HasPrefix(title, plan.normalized):
			score *= 1.3
		}
		if strings.Contains(title, plan.normalized) {
			score *= 1.25
		}
		out = append(out, candidate{doc: row.doc, score: score})
	}
	return out, nil
}

// browse returns the curated empty-query set. It reads search_docs directly
// because there is no MATCH expression to rank against; the ordering is applied
// in Go by kind and availability.
func (s *Service) browse(ctx context.Context) ([]candidate, error) {
	rows, err := s.db.QueryContext(ctx, searchSelect+`0.0 AS rank FROM search_docs d`)
	if err != nil {
		if isMissingTable(err) {
			return nil, ErrIndexNotInitialized
		}
		return nil, fmt.Errorf("browse search documents: %w", err)
	}
	defer rows.Close()
	raw, err := scanRows(rows)
	if err != nil {
		return nil, err
	}
	out := make([]candidate, 0, len(raw))
	for _, row := range raw {
		out = append(out, candidate{
			doc:   row.doc,
			score: kindPrior(KindText(row.doc.Kind)) * availabilityFactor(AvailabilityText(row.doc.Availability)),
		})
	}
	return out, nil
}

func (s *Service) ftsRows(ctx context.Context, expression string) ([]rawRow, error) {
	if strings.TrimSpace(expression) == "" {
		return nil, nil
	}
	rows, err := s.db.QueryContext(ctx, searchSelect+`bm25(search_fts, 0.0, 0.0, 10.0, 5.0, 1.0) AS rank
FROM search_fts JOIN search_docs d ON d.id = search_fts.id
WHERE search_fts MATCH ?`, expression)
	if err != nil {
		if isMissingTable(err) {
			return nil, ErrIndexNotInitialized
		}
		return nil, fmt.Errorf("search index query: %w", err)
	}
	defer rows.Close()
	return scanRows(rows)
}

func scanRows(rows *sql.Rows) ([]rawRow, error) {
	out := make([]rawRow, 0, 32)
	for rows.Next() {
		var (
			id, kindText, subjectID, title, subtitle, body, href string
			availability, badgesJSON, relationsJSON, updated     string
			rank                                                 float64
		)
		if err := rows.Scan(&id, &kindText, &subjectID, &title, &subtitle, &body, &href,
			&availability, &badgesJSON, &relationsJSON, &updated, &rank); err != nil {
			return nil, fmt.Errorf("scan search row: %w", err)
		}
		updatedAt, _ := time.Parse(time.RFC3339Nano, updated)
		out = append(out, rawRow{
			doc: Document{
				ID:           id,
				SubjectID:    subjectID,
				Kind:         ParseKind(kindText),
				Title:        title,
				Subtitle:     subtitle,
				Body:         body,
				Href:         href,
				Availability: ParseAvailability(availability),
				Badges:       unmarshalStrings(badgesJSON),
				Relations:    unmarshalStrings(relationsJSON),
				UpdatedAt:    updatedAt,
			},
			rank: rank,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read search rows: %w", err)
	}
	return out, nil
}

// dedupeCandidates keeps the highest-scoring row per kind+subject. A subject can
// legitimately appear under two kinds (an asset and the page type it realizes),
// but never twice under the same kind.
func dedupeCandidates(candidates []candidate) []candidate {
	best := make(map[string]int, len(candidates))
	out := make([]candidate, 0, len(candidates))
	for _, cand := range candidates {
		key := dedupeKey(cand.doc.Kind, cand.doc.SubjectID)
		if index, ok := best[key]; ok {
			if cand.score > out[index].score {
				out[index] = cand
			}
			continue
		}
		best[key] = len(out)
		out = append(out, cand)
	}
	return out
}

func requestedKinds(kinds []searchv1.SearchKind) map[string]bool {
	if len(kinds) == 0 {
		return nil
	}
	out := map[string]bool{}
	for _, kind := range kinds {
		if text := KindText(kind); text != "" {
			out[text] = true
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func filterKinds(candidates []candidate, kinds map[string]bool) []candidate {
	out := candidates[:0]
	for _, cand := range candidates {
		if kinds[KindText(cand.doc.Kind)] {
			out = append(out, cand)
		}
	}
	return out
}

// defaultBrowseKinds is the curated empty-query set. It mirrors the program's
// five browse destinations; display/state/evidence stay reachable through their
// parent or through an explicit kind filter.
var defaultBrowseKinds = map[string]bool{
	kindWorkspace: true,
	kindPage:      true,
	kindPageType:  true,
	kindAsset:     true,
	kindCommand:   true,
}

func sortRanked(candidates []candidate) {
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		left := strings.ToLower(candidates[i].doc.Title)
		right := strings.ToLower(candidates[j].doc.Title)
		if left != right {
			return left < right
		}
		return candidates[i].doc.ID < candidates[j].doc.ID
	})
}

// sortBrowse preserves the curated kind order. Availability and title break ties
// inside a kind; score is not the ordering key here because a planned workspace
// must still appear before an available command.
func sortBrowse(candidates []candidate) {
	sort.SliceStable(candidates, func(i, j int) bool {
		left, right := candidates[i].doc, candidates[j].doc
		if l, r := browseKindRank(KindText(left.Kind)), browseKindRank(KindText(right.Kind)); l != r {
			return l < r
		}
		if l, r := availabilityRank(AvailabilityText(left.Availability)), availabilityRank(AvailabilityText(right.Availability)); l != r {
			return l < r
		}
		lt, rt := strings.ToLower(left.Title), strings.ToLower(right.Title)
		if lt != rt {
			return lt < rt
		}
		return left.ID < right.ID
	})
}

func buildFacets(candidates []candidate) []*searchv1.SearchFacet {
	kindCounts := map[string]int32{}
	availabilityCounts := map[string]int32{}
	for _, cand := range candidates {
		kindCounts[KindText(cand.doc.Kind)]++
		availabilityCounts[AvailabilityText(cand.doc.Availability)]++
	}
	return []*searchv1.SearchFacet{
		facet("kind", "Kind", kindCounts, kindFacetOrder),
		facet("availability", "Availability", availabilityCounts, availabilityFacetOrder),
	}
}

var (
	kindFacetOrder         = []string{kindWorkspace, kindPage, kindDisplay, kindState, kindPageType, kindAsset, kindCommand, kindEvidence}
	availabilityFacetOrder = []string{availabilityAvailable, availabilityDraft, availabilityPlanned, availabilityUnavailable}
)

func facet(key, label string, counts map[string]int32, order []string) *searchv1.SearchFacet {
	out := &searchv1.SearchFacet{Key: key, Label: label}
	for _, value := range order {
		count := counts[value]
		if count == 0 {
			continue
		}
		valueLabel := KindLabel(value)
		if key == "availability" {
			valueLabel = AvailabilityLabel(value)
		}
		out.Values = append(out.Values, &searchv1.SearchFacetValue{Value: value, Label: valueLabel, Count: count})
	}
	return out
}

// Reindex rebuilds the whole projection from the owned sources in one
// transaction. A failed rebuild rolls back and records last_error, leaving the
// previous index queryable. force=false is a no-op for a current, error-free
// index.
func (s *Service) Reindex(ctx context.Context, force bool) (int, time.Time, error) {
	if s == nil || s.db == nil {
		return 0, time.Time{}, ErrNotConfigured
	}
	s.reindexMu.Lock()
	defer s.reindexMu.Unlock()

	if !force {
		status, err := s.Status(ctx)
		if err != nil {
			return 0, time.Time{}, err
		}
		if status.GetIndexed() && !status.GetStale() {
			indexedAt, _ := time.Parse(time.RFC3339Nano, status.GetIndexedAt())
			return int(status.GetDocumentCount()), indexedAt, nil
		}
	}

	now := time.Now().UTC()
	// Catalog + availability are resolved before the transaction. They read
	// files and an injected resolver, and holding SQLite's single writer while
	// doing that would serialize the whole workbench behind a reindex.
	catalogDocs, warnings, err := s.catalogDocuments(ctx, now)
	if err != nil {
		return s.failReindex(ctx, err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return s.failReindex(ctx, fmt.Errorf("begin search reindex: %w", err))
	}
	// The rollback is explicit on every failure path before failReindex writes
	// last_error: SQLite allows one connection here, so recording the failure
	// while the transaction still holds that connection would deadlock.
	abort := func(cause error) (int, time.Time, error) {
		_ = tx.Rollback()
		return s.failReindex(ctx, cause)
	}
	defer func() { _ = tx.Rollback() }()

	docs := catalogDocs
	workspaceDocs, err := s.workspaceDocuments(ctx, tx, now)
	if err != nil {
		return abort(err)
	}
	docs = append(docs, workspaceDocs...)
	docs = append(docs, commandDocuments(now)...)
	findingDocs, err := s.findingDocuments(ctx, tx, now)
	if err != nil {
		return abort(err)
	}
	docs = append(docs, findingDocs...)

	normalized, err := normalizeDocuments(docs, now)
	if err != nil {
		return abort(err)
	}
	if err := writeDocuments(ctx, tx, normalized, now, strings.Join(warnings, "; ")); err != nil {
		return abort(err)
	}
	if err := tx.Commit(); err != nil {
		return s.failReindex(ctx, fmt.Errorf("commit search reindex: %w", err))
	}
	return len(normalized), now, nil
}

// failReindex records the failure after the transaction has rolled back. If the
// state write also fails (for example the schema is absent), the original error
// is still the one reported.
func (s *Service) failReindex(ctx context.Context, cause error) (int, time.Time, error) {
	writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	_ = setState(writeCtx, s.db, stateLastError, cause.Error())
	return 0, time.Time{}, cause
}
