package conversationsearch

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	aisearch "github.com/vrooli/ai-go/search"
)

// SemanticSource exposes the canonical SQLite projection through the shared
// bounded source contract. It never exposes transcript paths or unnormalized
// attachment data.
type SemanticSource struct {
	repository SourceRepository
}

// StagedSemanticSource reads the immutable SQLite shadow that was already
// validated by the lexical indexer. Using the same staged snapshot prevents a
// long vector build (or its restart) from drifting onto newer canonical rows.
type StagedSemanticSource struct {
	repository   *SQLiteRepository
	generationID string
}

// semanticPageCursor can resume inside a canonical source page. The canonical
// repository bounds rows, but normalization may split one row into several
// documents, so forwarding its cursor directly can violate PagedSource's hard
// document limit or skip the overflow.
type semanticPageCursor struct {
	Source *SourceCursor `json:"source,omitempty"`
	Offset int           `json:"offset,omitempty"`
}

func NewSemanticSource(repository SourceRepository) *SemanticSource {
	return &SemanticSource{repository: repository}
}

func NewStagedSemanticSource(repository *SQLiteRepository, generationID string) *StagedSemanticSource {
	return &StagedSemanticSource{repository: repository, generationID: generationID}
}

func (s *StagedSemanticSource) LoadPage(ctx context.Context, request aisearch.PageRequest) (aisearch.SourcePage, error) {
	if s == nil || s.repository == nil || strings.TrimSpace(s.generationID) == "" {
		return aisearch.SourcePage{}, fmt.Errorf("staged conversation semantic source is not configured")
	}
	if request.Limit <= 0 || request.Limit > maxSourcePageSize {
		return aisearch.SourcePage{}, fmt.Errorf("staged semantic source limit must be between 1 and %d", maxSourcePageSize)
	}
	after := ""
	if request.Cursor != "" {
		decoded, err := base64.RawURLEncoding.DecodeString(request.Cursor)
		if err != nil {
			return aisearch.SourcePage{}, fmt.Errorf("decode staged semantic cursor: %w", err)
		}
		after = string(decoded)
	}
	var rows []documentRow
	query := `SELECT ` + projectionDocumentColumns + `
FROM conversation_search_generation_documents
WHERE generation_id=? AND visible=1 AND content_class IN (?, ?) AND document_id>?
ORDER BY document_id LIMIT ?`
	if err := s.repository.db.SelectContext(ctx, &rows, query, s.generationID, ContentClassProse, ContentClassQuotedProse, after, request.Limit+1); err != nil {
		return aisearch.SourcePage{}, fmt.Errorf("load staged semantic page: %w", err)
	}
	hasMore := len(rows) > request.Limit
	if hasMore {
		rows = rows[:request.Limit]
	}
	page := aisearch.SourcePage{Documents: make([]aisearch.SourceDoc, 0, len(rows)), Done: !hasMore}
	for _, row := range rows {
		document, err := row.document()
		if err != nil {
			return aisearch.SourcePage{}, err
		}
		page.Documents = append(page.Documents, semanticSourceDoc(document))
	}
	if hasMore {
		page.NextCursor = base64.RawURLEncoding.EncodeToString([]byte(rows[len(rows)-1].DocumentID))
	}
	return page, nil
}

func (s *SemanticSource) LoadPage(ctx context.Context, request aisearch.PageRequest) (aisearch.SourcePage, error) {
	if s == nil || s.repository == nil {
		return aisearch.SourcePage{}, fmt.Errorf("conversation semantic source is not configured")
	}
	if request.Limit <= 0 {
		return aisearch.SourcePage{}, fmt.Errorf("conversation semantic source requires a positive page limit")
	}
	state := semanticPageCursor{}
	if request.Cursor != "" {
		decoded, err := base64.RawURLEncoding.DecodeString(request.Cursor)
		if err != nil {
			return aisearch.SourcePage{}, fmt.Errorf("decode source cursor: %w", err)
		}
		if err := json.Unmarshal(decoded, &state); err != nil {
			return aisearch.SourcePage{}, fmt.Errorf("parse source cursor: %w", err)
		}
		if state.Offset < 0 {
			return aisearch.SourcePage{}, fmt.Errorf("parse source cursor: negative document offset")
		}
	} else if snapshotSource, ok := s.repository.(SnapshotSource); ok {
		snapshot, err := snapshotSource.SnapshotCursor(ctx)
		if err != nil {
			return aisearch.SourcePage{}, err
		}
		state.Source = snapshot
	}
	for {
		page, err := s.repository.LoadSourcePage(ctx, state.Source, request.Limit)
		if err != nil {
			return aisearch.SourcePage{}, err
		}
		if state.Offset > len(page.Documents) {
			return aisearch.SourcePage{}, fmt.Errorf("source cursor offset %d exceeds page size %d", state.Offset, len(page.Documents))
		}
		indexable := make([]Document, 0, len(page.Documents))
		for _, document := range page.Documents {
			if semanticIndexableContent(document.ContentClass) {
				indexable = append(indexable, document)
			}
		}
		if state.Offset > len(indexable) {
			return aisearch.SourcePage{}, fmt.Errorf("source cursor offset %d exceeds indexable page size %d", state.Offset, len(indexable))
		}
		documents := indexable[state.Offset:]
		if len(documents) == 0 && page.NextCursor != nil {
			state = semanticPageCursor{Source: page.NextCursor}
			continue
		}
		outputCount := len(documents)
		if outputCount > request.Limit {
			outputCount = request.Limit
		}
		output := aisearch.SourcePage{Documents: make([]aisearch.SourceDoc, 0, outputCount)}
		for _, document := range documents[:outputCount] {
			output.Documents = append(output.Documents, semanticSourceDoc(document))
		}
		switch {
		case outputCount < len(documents):
			output.NextCursor, err = encodeSemanticCursor(semanticPageCursor{Source: state.Source, Offset: state.Offset + outputCount})
		case page.NextCursor != nil:
			output.NextCursor, err = encodeSemanticCursor(semanticPageCursor{Source: page.NextCursor})
		default:
			output.Done = true
		}
		if err != nil {
			return aisearch.SourcePage{}, err
		}
		return output, nil
	}
}

func semanticIndexableContent(class ContentClass) bool {
	return class == ContentClassProse || class == ContentClassQuotedProse
}

func encodeSemanticCursor(cursor semanticPageCursor) (string, error) {
	encoded, err := json.Marshal(cursor)
	if err != nil {
		return "", fmt.Errorf("encode source cursor: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(encoded), nil
}

// LoadAll deliberately refuses to materialize conversation history. It exists
// only so the shared binding can carry this source through its schema guard;
// indexing uses the PagedSource contract above.
func (s *SemanticSource) LoadAll(context.Context) ([]aisearch.SourceDoc, error) {
	return nil, fmt.Errorf("conversation history must be loaded through the bounded paged source")
}

func semanticSourceDoc(document Document) aisearch.SourceDoc {
	return aisearch.SourceDoc{
		ID:          document.DocumentID,
		Kind:        "conversation",
		ContentHash: document.ContentHash,
		Body:        document.Content,
		Meta: map[string]any{
			"document_id": document.DocumentID, "source_id": document.DocumentID,
			"run_id": document.SourceRunID, "event_id": document.SourceEventID,
			"role": document.Role, "occurred_at_unix": float64(document.OccurredAt.Unix()),
			"content_class": int(document.ContentClass), "harness": document.Harness,
			"provider_origin": document.ProviderOrigin, "project_scope": document.ProjectScope,
			"cwd_scope": document.CWDScope, "runner": document.Runner, "model": document.Model,
			"profile": document.Profile, "run_status": document.RunStatus, "run_label": document.RunLabel,
			"tags": document.Tags, "workloads": document.Workloads,
		},
	}
}

type conversationEmbeddingComposer struct{}

const (
	maxEmbeddingContextFieldBytes = 128
	maxEmbeddingInputBytes        = 1536
)

// Compose enriches the raw conversational passage only with stable, useful
// context. Volatile identifiers, source paths, tool metadata, and evidence
// payloads are intentionally excluded.
func (conversationEmbeddingComposer) Compose(chunk aisearch.Chunk) string {
	contextParts := make([]string, 0, 3)
	for _, field := range []string{"role", "run_label", "project_scope"} {
		if value, ok := chunk.Meta[field].(string); ok && strings.TrimSpace(value) != "" {
			value = strings.TrimSpace(value)
			if len(value) > maxEmbeddingContextFieldBytes {
				value = value[:runeBoundaryAtOrBefore(value, maxEmbeddingContextFieldBytes)]
			}
			contextParts = append(contextParts, value)
		}
	}
	text := chunk.Body
	if len(contextParts) == 0 {
		return boundEmbeddingInput(text)
	}
	text = strings.Join(contextParts, " · ") + "\n\n" + chunk.Body
	return boundEmbeddingInput(text)
}

func boundEmbeddingInput(value string) string {
	if len(value) <= maxEmbeddingInputBytes {
		return value
	}
	return value[:runeBoundaryAtOrBefore(value, maxEmbeddingInputBytes)]
}

var (
	_ aisearch.Source      = (*SemanticSource)(nil)
	_ aisearch.PagedSource = (*SemanticSource)(nil)
)
