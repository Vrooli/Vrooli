package conversationsearch

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode"
)

const maxCandidatePageSize = 101

func (r *SQLiteRepository) LexicalCandidates(ctx context.Context, request CandidateQuery) ([]Candidate, error) {
	if request.Limit <= 0 || request.Limit > maxCandidatePageSize {
		return nil, fmt.Errorf("candidate limit must be between 1 and %d", maxCandidatePageSize)
	}
	if request.Sort == 0 {
		request.Sort = SearchSortRelevance
	}
	where := make([]string, 0, 16)
	args := make([]any, 0, 16)
	from := "conversation_search_documents d"
	score := "0.0"
	if strings.TrimSpace(request.Query) != "" {
		expression, err := buildFTSExpressionFor(request.Query, request.MatchAllTerms)
		if err != nil {
			return nil, err
		}
		from = "conversation_search_fts JOIN conversation_search_documents d ON d.rowid = conversation_search_fts.rowid"
		score = "-bm25(conversation_search_fts)"
		where = append(where, "conversation_search_fts MATCH ?")
		args = append(args, expression)
	} else if request.Sort == SearchSortRelevance {
		return nil, errors.New("lexical query is required for relevance sorting")
	}
	if !request.IncludeHidden {
		where = append(where, "d.visible = 1")
	}
	classes := request.ContentClasses
	if len(classes) == 0 {
		classes = []ContentClass{ContentClassProse, ContentClassQuotedProse}
	}
	where, args = appendIntegerFilter(where, args, "d.content_class", classes)
	where, args = appendStringFilter(where, args, "d.role", request.Roles)
	where, args = appendStringFilter(where, args, "d.harness", request.Harnesses)
	where, args = appendStringFilter(where, args, "d.provider_origin", request.ProviderOrigins)
	where, args = appendStringFilter(where, args, "d.project_scope", request.ProjectScopes)
	where, args = appendStringFilter(where, args, "d.cwd_scope", request.CWDScopes)
	where, args = appendStringFilter(where, args, "d.runner", request.Runners)
	where, args = appendStringFilter(where, args, "d.model", request.Models)
	where, args = appendStringFilter(where, args, "d.profile", request.Profiles)
	where, args = appendStringFilter(where, args, "d.run_status", request.RunStatuses)
	where, args = appendJSONFilter(where, args, "d.tags_json", request.Tags)
	where, args = appendJSONFilter(where, args, "d.workloads_json", request.Workloads)
	if request.OccurredAfter != nil {
		where = append(where, "d.occurred_at >= ?")
		args = append(args, formatTime(*request.OccurredAfter))
	}
	if request.OccurredBefore != nil {
		where = append(where, "d.occurred_at <= ?")
		args = append(args, formatTime(*request.OccurredBefore))
	}

	query := `WITH ranked AS (
        SELECT d.*, ` + score + ` AS search_score
        FROM ` + from + `
        WHERE ` + strings.Join(where, " AND ") + `
    ) SELECT * FROM ranked`
	order, cursorClause, cursorArgs, err := lexicalOrder(request.Sort, request.After)
	if err != nil {
		return nil, err
	}
	if cursorClause != "" {
		query += " WHERE " + cursorClause
		args = append(args, cursorArgs...)
	}
	query += " ORDER BY " + order + " LIMIT ?"
	args = append(args, request.Limit)

	rows, err := r.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query lexical conversation candidates: %w", err)
	}
	defer rows.Close()
	candidates := make([]Candidate, 0, request.Limit)
	for rows.Next() {
		var row lexicalCandidateRow
		if err := rows.StructScan(&row); err != nil {
			return nil, fmt.Errorf("scan lexical conversation candidate: %w", err)
		}
		document, err := row.documentRow.document()
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, Candidate{Document: document, Score: row.Score, Rank: len(candidates) + 1})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate lexical conversation candidates: %w", err)
	}
	return candidates, nil
}

func (r *SQLiteRepository) ContextDocuments(ctx context.Context, runID string, sequence int64, before, after int) ([]Document, error) {
	if runID == "" || sequence < 0 || before < 0 || after < 0 || before > 20 || after > 20 {
		return nil, errors.New("context requires a run, non-negative sequence, and before/after bounds no greater than 20")
	}
	previous, err := r.contextSide(ctx, runID, sequence, before, "<", "DESC")
	if err != nil {
		return nil, err
	}
	for left, right := 0, len(previous)-1; left < right; left, right = left+1, right-1 {
		previous[left], previous[right] = previous[right], previous[left]
	}
	current, err := r.contextSide(ctx, runID, sequence, 1, "=", "ASC")
	if err != nil {
		return nil, err
	}
	next, err := r.contextSide(ctx, runID, sequence, after, ">", "ASC")
	if err != nil {
		return nil, err
	}
	return append(append(previous, current...), next...), nil
}

func (r *SQLiteRepository) contextSide(ctx context.Context, runID string, sequence int64, limit int, comparator, direction string) ([]Document, error) {
	if limit == 0 {
		return nil, nil
	}
	if comparator != "<" && comparator != "=" && comparator != ">" {
		return nil, errors.New("invalid internal context comparator")
	}
	if direction != "ASC" && direction != "DESC" {
		return nil, errors.New("invalid internal context direction")
	}
	query := `SELECT * FROM conversation_search_documents
        WHERE source_run_id = ? AND visible = 1 AND chunk_index = 0 AND event_sequence ` + comparator + ` ?
        ORDER BY event_sequence ` + direction + `, document_id ASC LIMIT ?`
	rows, err := r.db.QueryxContext(ctx, query, runID, sequence, limit)
	if err != nil {
		return nil, fmt.Errorf("query conversation context: %w", err)
	}
	defer rows.Close()
	documents := make([]Document, 0, limit)
	for rows.Next() {
		var row documentRow
		if err := rows.StructScan(&row); err != nil {
			return nil, fmt.Errorf("scan conversation context: %w", err)
		}
		document, err := row.document()
		if err != nil {
			return nil, err
		}
		documents = append(documents, document)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate conversation context: %w", err)
	}
	return documents, nil
}

type lexicalCandidateRow struct {
	documentRow
	Score float64 `db:"search_score"`
}

func buildFTSExpression(query string) (string, error) {
	return buildFTSExpressionFor(query, false)
}

func buildFTSExpressionFor(query string, matchAll bool) (string, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return "", errors.New("lexical query is required")
	}
	var allTerms, terms []string
	var token strings.Builder
	quoted := false
	flush := func() {
		value := strings.TrimSpace(token.String())
		token.Reset()
		if value != "" {
			term := `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
			allTerms = append(allTerms, term)
			if strings.ContainsRune(value, ' ') || !isLexicalStopword(value) {
				terms = append(terms, term)
			}
		}
	}
	for _, character := range query {
		switch {
		case character == '"':
			if quoted {
				flush()
				quoted = false
			} else {
				flush()
				quoted = true
			}
		case unicode.IsSpace(character) && !quoted:
			flush()
		default:
			token.WriteRune(character)
		}
	}
	flush()
	if len(allTerms) == 0 {
		return "", errors.New("lexical query has no searchable terms")
	}
	if len(terms) == 0 {
		terms = allTerms
	}
	operator := " OR "
	if matchAll {
		operator = " AND "
	}
	return strings.Join(terms, operator), nil
}

func isLexicalStopword(value string) bool {
	value = strings.ToLower(strings.Trim(value, `"'.,;:!?()[]{} `))
	switch value {
	case "a", "an", "and", "about", "agent", "conversation", "discussed", "earlier", "find", "for", "how", "in", "me", "my", "of", "on", "or", "previous", "prior", "recall", "run", "that", "the", "to", "what", "when", "where", "which", "who", "why", "with":
		return true
	default:
		return false
	}
}

func appendStringFilter(where []string, args []any, column string, values []string) ([]string, []any) {
	values = compactStrings(values)
	if len(values) == 0 {
		return where, args
	}
	where = append(where, column+" IN ("+placeholders(len(values))+")")
	for _, value := range values {
		args = append(args, value)
	}
	return where, args
}

func appendIntegerFilter(where []string, args []any, column string, values []ContentClass) ([]string, []any) {
	if len(values) == 0 {
		return where, args
	}
	where = append(where, column+" IN ("+placeholders(len(values))+")")
	for _, value := range values {
		args = append(args, value)
	}
	return where, args
}

func appendJSONFilter(where []string, args []any, column string, values []string) ([]string, []any) {
	values = compactStrings(values)
	if len(values) == 0 {
		return where, args
	}
	where = append(where, "EXISTS (SELECT 1 FROM json_each("+column+") facet WHERE facet.value IN ("+placeholders(len(values))+"))")
	for _, value := range values {
		args = append(args, value)
	}
	return where, args
}

func placeholders(count int) string { return strings.TrimSuffix(strings.Repeat("?,", count), ",") }

func lexicalOrder(sort SearchSort, cursor *CandidateCursor) (order, clause string, args []any, err error) {
	switch sort {
	case SearchSortRelevance:
		order = "search_score DESC, occurred_at DESC, document_id ASC"
		if cursor != nil {
			clause = `(search_score < ? OR (search_score = ? AND occurred_at < ?) OR
                (search_score = ? AND occurred_at = ? AND document_id > ?))`
			args = []any{cursor.Score, cursor.Score, formatTime(cursor.OccurredAt), cursor.Score, formatTime(cursor.OccurredAt), cursor.DocumentID}
		}
	case SearchSortNewest:
		order = "occurred_at DESC, document_id ASC"
		if cursor != nil {
			clause = `(occurred_at < ? OR (occurred_at = ? AND document_id > ?))`
			args = []any{formatTime(cursor.OccurredAt), formatTime(cursor.OccurredAt), cursor.DocumentID}
		}
	case SearchSortOldest:
		order = "occurred_at ASC, document_id ASC"
		if cursor != nil {
			clause = `(occurred_at > ? OR (occurred_at = ? AND document_id > ?))`
			args = []any{formatTime(cursor.OccurredAt), formatTime(cursor.OccurredAt), cursor.DocumentID}
		}
	default:
		return "", "", nil, fmt.Errorf("unsupported search sort %d", sort)
	}
	if cursor != nil && (cursor.OccurredAt.IsZero() || cursor.DocumentID == "") {
		return "", "", nil, errors.New("candidate cursor requires occurred time and document id")
	}
	return order, clause, args, nil
}

var _ CandidateRepository = (*SQLiteRepository)(nil)
