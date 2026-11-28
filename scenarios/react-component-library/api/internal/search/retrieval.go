package search

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	aisearch "github.com/vrooli/ai-go/search"
)

type candidate struct {
	doc          Document
	score        float64
	intentRank   int
	lexicalMatch bool
}

type rawRow struct {
	doc  Document
	rank float64
}

// expandRequiredCatalogAssets follows authored catalog dependencies from a
// matched page blueprint (or a workspace/page that names that blueprint).
// These are page-type requirements, not proof that the workspace renders each
// asset. The workspace's visual node vocabulary is deliberately not translated.
func (s *Service) expandRequiredCatalogAssets(ctx context.Context, candidates []candidate) ([]candidate, error) {
	pageTypes := make(map[string]struct{})
	for _, candidate := range candidates {
		switch KindText(candidate.doc.Kind) {
		case kindPageType:
			pageTypes[candidate.doc.SubjectID] = struct{}{}
		case kindWorkspace, kindPage:
			for _, relation := range candidate.doc.Relations {
				if strings.HasPrefix(relation, "page-type:") {
					pageTypes[strings.TrimPrefix(relation, "page-type:")] = struct{}{}
				}
			}
		}
	}
	if len(pageTypes) == 0 {
		return candidates, nil
	}

	args := make([]any, 0, len(pageTypes)*2)
	placeholders := make([]string, 0, len(pageTypes)*2)
	for id := range pageTypes {
		placeholders = append(placeholders, "?")
		args = append(args, id)
	}
	rows, err := s.db.QueryContext(ctx, `SELECT subject_id, title, catalog_json FROM search_docs WHERE kind = ? AND subject_id IN (`+strings.Join(placeholders, ",")+")", append([]any{kindPageType}, args...)...)
	if err != nil {
		return nil, fmt.Errorf("read matched page type dependencies: %w", err)
	}
	assetRequirements := make(map[string][]string)
	for rows.Next() {
		var pageTypeID, pageTypeTitle, raw string
		if err := rows.Scan(&pageTypeID, &pageTypeTitle, &raw); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan matched page type dependencies: %w", err)
		}
		metadata, err := decodeCatalogMetadata(raw)
		if err != nil {
			rows.Close()
			return nil, err
		}
		for _, id := range metadata.Requires {
			assetRequirements[id] = append(assetRequirements[id], pageTypeID+"|"+pageTypeTitle)
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("read matched page type dependencies: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close matched page type dependencies: %w", err)
	}
	if len(assetRequirements) == 0 {
		return candidates, nil
	}
	args = args[:0]
	placeholders = placeholders[:0]
	for id := range assetRequirements {
		placeholders = append(placeholders, "?")
		args = append(args, id)
	}
	rows, err = s.db.QueryContext(ctx, `SELECT id, kind, subject_id, title, subtitle, body, href, availability, badges_json, relations_json, updated_at, catalog_json FROM search_docs WHERE kind = ? AND subject_id IN (`+strings.Join(placeholders, ",")+")", append([]any{kindAsset}, args...)...)
	if err != nil {
		return nil, fmt.Errorf("read required catalog assets: %w", err)
	}
	defer rows.Close()
	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		seen[candidate.doc.ID] = struct{}{}
	}
	for rows.Next() {
		var doc Document
		var kind, availability, badges, relations, updated, catalog string
		if err := rows.Scan(&doc.ID, &kind, &doc.SubjectID, &doc.Title, &doc.Subtitle, &doc.Body, &doc.Href, &availability, &badges, &relations, &updated, &catalog); err != nil {
			return nil, fmt.Errorf("scan required catalog asset: %w", err)
		}
		doc.Kind, doc.Availability = ParseKind(kind), ParseAvailability(availability)
		doc.Badges, doc.Relations = unmarshalStrings(badges), unmarshalStrings(relations)
		doc.UpdatedAt = parseTimeOr(updated, time.Time{})
		metadata, err := decodeCatalogMetadata(catalog)
		if err != nil {
			return nil, err
		}
		doc.Catalog = &metadata
		for _, source := range assetRequirements[doc.SubjectID] {
			parts := strings.SplitN(source, "|", 2)
			doc.Relations = append(doc.Relations, "page-type-requires:"+parts[0])
			if len(parts) == 2 && strings.TrimSpace(parts[1]) != "" {
				doc.Badges = append(doc.Badges, "Required by "+parts[1])
				doc.Body = joinNonEmpty("\n", "Required by "+parts[1]+" through its catalog requirements; this does not assert that the workspace renders the asset.", doc.Body)
			}
		}
		if _, duplicate := seen[doc.ID]; duplicate {
			continue
		}
		seen[doc.ID] = struct{}{}
		candidates = append(candidates, candidate{doc: doc, score: 0.000001})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read required catalog assets: %w", err)
	}
	return candidates, nil
}

const searchSelect = `
SELECT d.id, d.kind, d.subject_id, d.title, d.subtitle, d.body, d.href,
       d.availability, d.badges_json, d.relations_json, d.updated_at, `

// match runs the AND query first and falls back to OR only when AND returned
// nothing. Prefix matching is applied per token, so "land" already finds
// "LandingPage" without the fallback; the fallback exists for multi-word queries
// where one word is absent (for example "onboarding wizard").
func (s *Service) matchLexical(ctx context.Context, plan queryPlan) ([]candidate, error) {
	rows, err := s.ftsRows(ctx, plan.andQuery)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 && plan.orQuery != "" && plan.orQuery != plan.andQuery {
		if rows, err = s.ftsRows(ctx, plan.orQuery); err != nil {
			return nil, err
		}
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
		// A single-token exact component-name lookup is an explicit asset intent. Give an
		// exact asset title enough weight to beat a workspace/page that merely
		// mentions the component in its indexed content.
		if len(plan.words) == 1 && KindText(row.doc.Kind) == kindAsset && identifierTitle(row.doc.Title) == plan.normalized {
			score *= 2.0
		}
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

func (s *Service) match(ctx context.Context, plan queryPlan, kinds map[string]bool) ([]candidate, error) {
	out, err := s.matchLexical(ctx, plan)
	if err != nil {
		return nil, err
	}
	if semantic, ok := s.semantic.(aisearch.SemanticSearcher); ok && s.semantic.Enabled() {
		// Fusion is rank-based; first project the RCL outcome-first lexical
		// scoring into a stable lexical order so its degraded path preserves the
		// same precedence when the vector leg is unavailable.
		sortRanked(out)
		fused, fuseErr := (aisearch.ConcurrentFusion{
			Lexical: rankedLexical{hits: out}, Semantic: semantic,
		}).Search(ctx, aisearch.SearchQuery{Query: plan.normalized, Limit: maxSearchLimit, Filter: semanticKindFilter(kinds)})
		if fuseErr == nil {
			projected, missing, projectErr := s.projectFusedDetailed(ctx, fused.Results)
			if projectErr == nil {
				if len(fused.Degraded) > 0 || missing > 0 {
					cause := fmt.Errorf("semantic retrieval leg degraded (%s)", strings.Join(fused.Degraded, ","))
					if missing > 0 {
						cause = fmt.Errorf("semantic fusion returned %d unprojectable hit(s)", missing)
					}
					s.recordSemanticFailure(cause)
				} else {
					s.clearSemanticFailure()
				}
				return projected, nil
			}
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			s.recordSemanticFailure(projectErr)
			// The committed SQLite projection remains authoritative. A stale
			// vector hit is ignored and the healthy lexical leg remains usable.
		} else {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			s.recordSemanticFailure(fuseErr)
		}
	}
	return out, nil
}

func semanticKindFilter(kinds map[string]bool) *aisearch.QueryFilter {
	if len(kinds) == 0 {
		return nil
	}
	values := make([]any, 0, len(kinds))
	for kind := range kinds {
		values = append(values, kind)
	}
	return &aisearch.QueryFilter{Must: []aisearch.FieldMatch{{Key: "kind", AnyOf: values}}}
}

// rankedLexical adapts the already ranked FTS results to ai-go's shared fusion
// contract. Keeping FTS ranking here preserves its title, kind and availability
// signals while RRF adds semantic-only paraphrase matches.
type rankedLexical struct{ hits []candidate }

func (l rankedLexical) SearchLexical(_ context.Context, _ aisearch.SearchQuery) ([]aisearch.SearchResult, error) {
	out := make([]aisearch.SearchResult, 0, len(l.hits))
	for _, hit := range l.hits {
		out = append(out, aisearch.SearchResult{ID: hit.doc.ID, Score: hit.score})
	}
	return out, nil
}

func (s *Service) projectFused(ctx context.Context, fused []aisearch.FusedResult) ([]candidate, error) {
	projected, _, err := s.projectFusedDetailed(ctx, fused)
	return projected, err
}

func (s *Service) projectFusedDetailed(ctx context.Context, fused []aisearch.FusedResult) ([]candidate, int, error) {
	if len(fused) == 0 {
		return nil, 0, nil
	}
	// Resolve the fused IDs in one SQLite round trip. Semantic search can return
	// maxSearchLimit hits; issuing QueryRow once per hit made projection latency
	// grow linearly with SQLite scheduling and connection overhead.
	ids := make([]string, 0, len(fused))
	args := make([]any, 0, len(fused))
	for _, hit := range fused {
		ids = append(ids, hit.Result.ID)
		args = append(args, hit.Result.ID)
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(ids)), ",")
	rows, err := s.db.QueryContext(ctx, `SELECT id, kind, subject_id, title, subtitle, body, href, availability, badges_json, relations_json, updated_at FROM search_docs WHERE id IN (`+placeholders+`)`, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("project fused search results: %w", err)
	}
	documents := make(map[string]Document, len(ids))
	for rows.Next() {
		var doc Document
		var kind, availability, badges, relations, updated string
		if err := rows.Scan(&doc.ID, &kind, &doc.SubjectID, &doc.Title, &doc.Subtitle, &doc.Body, &doc.Href, &availability, &badges, &relations, &updated); err != nil {
			rows.Close()
			return nil, 0, fmt.Errorf("scan fused search projection: %w", err)
		}
		doc.Kind, doc.Availability = ParseKind(kind), ParseAvailability(availability)
		doc.Badges, doc.Relations = unmarshalStrings(badges), unmarshalStrings(relations)
		doc.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
		documents[doc.ID] = doc
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, 0, fmt.Errorf("read fused search projection: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, 0, fmt.Errorf("close fused search projection: %w", err)
	}
	out := make([]candidate, 0, len(fused))
	missing := 0
	for _, hit := range fused {
		doc, ok := documents[hit.Result.ID]
		if !ok {
			missing++
			continue
		}
		kind, availability := KindText(doc.Kind), AvailabilityText(doc.Availability)
		score := hit.Result.Score * kindPrior(kind) * availabilityFactor(availability)
		lexicalMatch := false
		for _, evidence := range hit.Evidence {
			if evidence.Leg == "lexical" {
				lexicalMatch = true
				break
			}
		}
		if !lexicalMatch && isEmptyDefaultPage(doc) {
			continue
		}
		out = append(out, candidate{doc: doc, score: score, lexicalMatch: lexicalMatch})
	}
	return out, missing, nil
}

// isEmptyDefaultPage rejects vector-only matches for pages that have only the
// editor's default name and an empty spec. Their embeddings mostly describe
// generic scaffolding, so they can otherwise consume a result slot for an
// unrelated outcome query. Named pages and pages with actual content remain
// eligible for semantic discovery.
func isEmptyDefaultPage(doc Document) bool {
	if KindText(doc.Kind) != kindPage {
		return false
	}
	name := strings.Fields(strings.ToLower(strings.TrimSpace(doc.Title)))
	if len(name) != 2 || name[0] != "page" {
		return false
	}
	for _, digit := range name[1] {
		if digit < '0' || digit > '9' {
			return false
		}
	}
	if strings.Trim(name[1], "0") == "" {
		return false
	}

	var content []string
	for _, line := range strings.Split(doc.Body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.EqualFold(line, doc.Title) || strings.HasPrefix(line, "Page id:") {
			continue
		}
		content = append(content, line)
	}
	return len(content) == 1 && (content[0] == "{}" || content[0] == "[]" || strings.EqualFold(content[0], "null"))
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
