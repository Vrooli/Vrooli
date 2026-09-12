package conversationsearch

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

const maximumRegexCandidateRead = 2001

// RegexCandidates returns a bounded, deterministic projection scan. Query is
// never interpreted as SQL; only a conservatively extracted mandatory literal
// may enter the parameterized FTS prefilter.
func (r *SQLiteRepository) RegexCandidates(ctx context.Context, request CandidateQuery) (RegexCandidatePage, error) {
	if request.Limit <= 0 || request.Limit >= maximumRegexCandidateRead {
		return RegexCandidatePage{}, fmt.Errorf("regex candidate limit must be between 1 and %d", maximumRegexCandidateRead-1)
	}
	if request.ByteLimit <= 0 {
		return RegexCandidatePage{}, errors.New("regex candidate byte limit must be positive")
	}
	where := make([]string, 0, 16)
	args := make([]any, 0, 16)
	from := "conversation_search_documents d"
	if request.PrefilterLiteral != "" {
		expression, err := buildFTSExpression(request.PrefilterLiteral)
		if err != nil {
			return RegexCandidatePage{}, fmt.Errorf("build regex literal prefilter: %w", err)
		}
		from = "conversation_search_fts JOIN conversation_search_documents d ON d.rowid = conversation_search_fts.rowid"
		where = append(where, "conversation_search_fts MATCH ?")
		args = append(args, expression)
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

	query := "SELECT d.* FROM " + from + " WHERE " + strings.Join(where, " AND ") +
		" ORDER BY d.occurred_at DESC, d.document_id ASC LIMIT ?"
	args = append(args, request.Limit+1)
	rows, err := r.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return RegexCandidatePage{}, fmt.Errorf("query bounded regex candidates: %w", err)
	}
	defer rows.Close()

	page := RegexCandidatePage{Documents: make([]Document, 0, request.Limit)}
	for rows.Next() {
		var row documentRow
		if err := rows.StructScan(&row); err != nil {
			return RegexCandidatePage{}, fmt.Errorf("scan bounded regex candidate: %w", err)
		}
		if len(page.Documents) == request.Limit {
			page.HasMore = true
			page.LimitReason = RegexLimitCandidates
			break
		}
		if page.ScannedBytes+len(row.Content) > request.ByteLimit {
			page.HasMore = true
			page.LimitReason = RegexLimitBytes
			break
		}
		document, err := row.document()
		if err != nil {
			return RegexCandidatePage{}, err
		}
		page.Documents = append(page.Documents, document)
		page.ScannedBytes += len(row.Content)
	}
	if err := rows.Err(); err != nil {
		return RegexCandidatePage{}, fmt.Errorf("iterate bounded regex candidates: %w", err)
	}
	return page, nil
}

var _ RegexCandidateRepository = (*SQLiteRepository)(nil)
