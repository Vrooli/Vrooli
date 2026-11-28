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
	Catalog      *CatalogMetadata
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
		if doc.Catalog != nil {
			catalog := *doc.Catalog
			doc.Catalog = &catalog
		}
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
	if err := ensureCatalogColumn(ctx, tx); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM search_fts`); err != nil {
		return fmt.Errorf("clear search_fts: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM search_docs`); err != nil {
		return fmt.Errorf("clear search_docs: %w", err)
	}
	docStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO search_docs
			(id, kind, subject_id, title, subtitle, body, href, availability, badges_json, relations_json, updated_at, catalog_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
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
		catalog := "{}"
		if doc.Catalog != nil {
			encoded, err := json.Marshal(doc.Catalog)
			if err != nil {
				return fmt.Errorf("encode catalog metadata for %s: %w", doc.ID, err)
			}
			catalog = string(encoded)
		}
		if _, err := docStmt.ExecContext(ctx,
			doc.ID, kind, doc.SubjectID, doc.Title, doc.Subtitle, doc.Body, doc.Href,
			AvailabilityText(doc.Availability), badges, relations,
			doc.UpdatedAt.UTC().Format(time.RFC3339Nano), catalog,
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

func ensureCatalogColumn(ctx context.Context, tx *sql.Tx) error {
	rows, err := tx.QueryContext(ctx, `PRAGMA table_info(search_docs)`)
	if err != nil {
		return fmt.Errorf("inspect search_docs schema: %w", err)
	}
	hasCatalog := false
	for rows.Next() {
		var cid, notNull, primary int
		var name, kind string
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &kind, &notNull, &defaultValue, &primary); err != nil {
			rows.Close()
			return fmt.Errorf("scan search_docs schema: %w", err)
		}
		if name == "catalog_json" {
			hasCatalog = true
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("read search_docs schema: %w", err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close search_docs schema: %w", err)
	}
	if hasCatalog {
		return nil
	}
	if _, err := tx.ExecContext(ctx, `ALTER TABLE search_docs ADD COLUMN catalog_json TEXT NOT NULL DEFAULT '{}'`); err != nil {
		return fmt.Errorf("add catalog metadata to search_docs: %w", err)
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
	// ErrAskUnavailable means explicit Ask mode has no answering provider wired.
	// It prevents an Ask request from being silently answered as lexical search.
	ErrAskUnavailable = errors.New("ask search is not configured")
	// ErrAskInvalidQuery means the caller's question is outside the Ask contract.
	ErrAskInvalidQuery = errors.New("ask question is invalid")
	// ErrAskProviderUnavailable means an answering provider is configured but
	// could not be reached or did not return a usable Gateway response.
	ErrAskProviderUnavailable = errors.New("ask provider is unavailable")
	// ErrAskInvalidAnswer means the model response could not be safely tied to
	// the exact indexed evidence supplied for the request.
	ErrAskInvalidAnswer = errors.New("ask answer is invalid")
)
