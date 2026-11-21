package search

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	searchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/search"
)

// Document is one row of the search projection. It is exported because the
// projection is also the test seam: tests seed a known corpus through
// ReplaceDocuments instead of standing up the whole workbench.
type Document struct {
	// ID is the projection identity, namespaced by kind so the same subject id
	// cannot collide across kinds. See DocumentID.
	ID string
	// SubjectID is the identity of the owning object (catalog id, workspace id,
	// page id, command id, finding id).
	SubjectID    string
	Kind         searchv1.SearchKind
	Title        string
	Subtitle     string
	Body         string
	Href         string
	Availability searchv1.SearchAvailability
	Badges       []string
	Relations    []string
	UpdatedAt    time.Time
}

// DocumentID is the canonical projection id for one subject. It is what
// search_fts joins back to and what deduplication keys on, so rebuilds are
// stable and a subject cannot be presented twice.
func DocumentID(kind searchv1.SearchKind, subjectID string) string {
	return KindText(kind) + ":" + strings.TrimSpace(subjectID)
}

// dedupeKey matches DocumentID, but is kept separate so the intent ("one row
// per kind and subject") is readable at the ranking site.
func dedupeKey(kind searchv1.SearchKind, subjectID string) string {
	return KindText(kind) + "|" + subjectID
}

// ReplaceDocuments atomically replaces the whole projection. It is the
// production write path used by Reindex and the supported test seam for
// golden-query fixtures. A failure leaves the previous projection untouched.
func (s *Service) ReplaceDocuments(ctx context.Context, docs []Document) error {
	if s == nil || s.db == nil {
		return ErrNotConfigured
	}
	now := time.Now().UTC()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin search replace: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	normalized, err := normalizeDocuments(docs, now)
	if err != nil {
		return err
	}
	if err := writeDocuments(ctx, tx, normalized, now, ""); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit search replace: %w", err)
	}
	return nil
}

// normalizeDocuments fills in the columns the schema declares NOT NULL and
// rejects rows that cannot be addressed. It runs before the write so a bad
// fixture fails without touching the live index.
func normalizeDocuments(docs []Document, now time.Time) ([]Document, error) {
	out := make([]Document, 0, len(docs))
	for i, doc := range docs {
		subject := strings.TrimSpace(doc.SubjectID)
		if subject == "" {
			subject = strings.TrimSpace(doc.ID)
		}
		if subject == "" {
			return nil, fmt.Errorf("search document %d has no subject id", i)
		}
		kind := doc.Kind
		if KindText(kind) == "" {
			return nil, fmt.Errorf("search document %q has no kind", subject)
		}
		title := strings.TrimSpace(doc.Title)
		if title == "" {
			title = subject
		}
		id := strings.TrimSpace(doc.ID)
		if id == "" {
			id = DocumentID(kind, subject)
		}
		updated := doc.UpdatedAt
		if updated.IsZero() {
			updated = now
		}
		doc.ID = id
		doc.SubjectID = subject
		doc.Title = title
		doc.UpdatedAt = updated.UTC()
		if doc.Availability == searchv1.SearchAvailability_SEARCH_AVAILABILITY_UNSPECIFIED {
			doc.Availability = searchv1.SearchAvailability_SEARCH_AVAILABILITY_AVAILABLE
		}
		out = append(out, doc)
	}
	return out, nil
}

// writeDocuments is the single writer shared by Reindex and ReplaceDocuments.
// The caller owns the transaction so the index swap is atomic.
func writeDocuments(ctx context.Context, tx *sql.Tx, docs []Document, now time.Time, warnings string) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM search_fts`); err != nil {
		return fmt.Errorf("clear search_fts: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM search_docs`); err != nil {
		return fmt.Errorf("clear search_docs: %w", err)
	}
	docStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO search_docs
			(id, kind, subject_id, title, subtitle, body, href, availability, badges_json, relations_json, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare search_docs insert: %w", err)
	}
	defer docStmt.Close()
	ftsStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO search_fts (id, kind, title, subtitle, body) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare search_fts insert: %w", err)
	}
	defer ftsStmt.Close()

	for _, doc := range docs {
		kind := KindText(doc.Kind)
		badges, err := marshalStrings(doc.Badges)
		if err != nil {
			return fmt.Errorf("encode badges for %s: %w", doc.ID, err)
		}
		relations, err := marshalStrings(doc.Relations)
		if err != nil {
			return fmt.Errorf("encode relations for %s: %w", doc.ID, err)
		}
		if _, err := docStmt.ExecContext(ctx,
			doc.ID, kind, doc.SubjectID, doc.Title, doc.Subtitle, doc.Body, doc.Href,
			AvailabilityText(doc.Availability), badges, relations,
			doc.UpdatedAt.UTC().Format(time.RFC3339Nano),
		); err != nil {
			return fmt.Errorf("insert search document %s: %w", doc.ID, err)
		}
		if _, err := ftsStmt.ExecContext(ctx, doc.ID, kind, doc.Title, doc.Subtitle, doc.Body); err != nil {
			return fmt.Errorf("insert search fts row %s: %w", doc.ID, err)
		}
	}
	for key, value := range map[string]string{
		stateIndexedAt:         now.UTC().Format(time.RFC3339Nano),
		stateDocumentCount:     fmt.Sprintf("%d", len(docs)),
		stateLastError:         "",
		stateWarnings:          warnings,
		stateProjectionVersion: projectionVersion,
	} {
		if err := setState(ctx, tx, key, value); err != nil {
			return err
		}
	}
	return nil
}

func setState(ctx context.Context, execer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}, key, value string,
) error {
	if _, err := execer.ExecContext(ctx, `
		INSERT INTO search_state (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value); err != nil {
		return fmt.Errorf("write search state %s: %w", key, err)
	}
	return nil
}

func marshalStrings(values []string) (string, error) {
	cleaned := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			cleaned = append(cleaned, value)
		}
	}
	data, err := json.Marshal(cleaned)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func unmarshalStrings(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var values []string
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		// A malformed projection row is a defect, but it must not take search
		// down. Returning nil keeps the rest of the result usable and the
		// required Reindex will rewrite the row.
		return nil
	}
	out := values[:0]
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			out = append(out, value)
		}
	}
	return out
}

// isMissingTable reports the SQLite error for a projection read before
// EnsureSchemas has run. It is used to give the RPC a FailedPrecondition
// instead of a generic internal error.
func isMissingTable(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "no such table") || strings.Contains(message, "no such module: fts5")
}

var (
	// ErrNotConfigured means the service was built without a database handle.
	ErrNotConfigured = errors.New("search service is not configured")
	// ErrIndexNotInitialized means the projection tables do not exist yet.
	ErrIndexNotInitialized = errors.New("search index is not initialized")
)
