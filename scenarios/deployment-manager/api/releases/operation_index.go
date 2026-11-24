package releases

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	defaultOperationIndexPageSize = 50
	maxOperationIndexPageSize     = 100
	maxOperationPageTokenBytes    = 2048
)

var ErrInvalidOperationPageToken = errors.New("invalid release-operation page token")

type OperationIndexPage struct {
	OperationIDs  []string
	NextPageToken string
}

// OperationIndexRepository is the read-only persistence capability needed by
// the profile-scoped generated operation index.
type OperationIndexRepository interface {
	ListOperationIDsByProfile(context.Context, string, string, int) (OperationIndexPage, error)
}

var _ OperationIndexRepository = (*SQLRepository)(nil)

type operationIndexCursor struct {
	Version     int       `json:"v"`
	ProfileID   string    `json:"p"`
	CreatedAt   time.Time `json:"c"`
	OperationID string    `json:"o"`
}

type operationIndexEntry struct {
	ID        string
	CreatedAt time.Time
}

// ListOperationIDsByProfile returns an immutable profile-bound operation
// index ordered by created_at DESC, id DESC. It returns identities only; the
// canonical generated standing must still be fetched with GetOperation.
func (r *SQLRepository) ListOperationIDsByProfile(ctx context.Context, profileID, pageToken string, pageSize int) (OperationIndexPage, error) {
	profileID = strings.TrimSpace(profileID)
	if profileID == "" {
		return OperationIndexPage{}, fmt.Errorf("profile ID is required")
	}
	if pageSize == 0 {
		pageSize = defaultOperationIndexPageSize
	}
	if pageSize < 1 || pageSize > maxOperationIndexPageSize {
		return OperationIndexPage{}, fmt.Errorf("page size must be between 1 and %d", maxOperationIndexPageSize)
	}
	var cursor *operationIndexCursor
	if pageToken != "" {
		decoded, err := decodeOperationIndexCursor(pageToken, profileID)
		if err != nil {
			return OperationIndexPage{}, err
		}
		cursor = &decoded
	}
	limit := pageSize + 1
	query := `SELECT release_operations.id, release_operations.created_at FROM release_operations JOIN releases AS indexed_release ON indexed_release.id = release_operations.release_id AND indexed_release.profile_id = release_operations.profile_id WHERE release_operations.profile_id = $1 ORDER BY release_operations.created_at DESC, release_operations.id DESC LIMIT $2`
	args := []interface{}{profileID, limit}
	if cursor != nil {
		query = `SELECT release_operations.id, release_operations.created_at FROM release_operations JOIN releases AS indexed_release ON indexed_release.id = release_operations.release_id AND indexed_release.profile_id = release_operations.profile_id WHERE release_operations.profile_id = $1 AND (release_operations.created_at < $2 OR (release_operations.created_at = $2 AND release_operations.id < $3)) ORDER BY release_operations.created_at DESC, release_operations.id DESC LIMIT $4`
		args = []interface{}{profileID, cursor.CreatedAt, cursor.OperationID, limit}
	}
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return OperationIndexPage{}, fmt.Errorf("list release operations by profile: %w", err)
	}
	defer rows.Close()
	entries := make([]operationIndexEntry, 0, limit)
	for rows.Next() {
		var entry operationIndexEntry
		if err := rows.Scan(&entry.ID, &entry.CreatedAt); err != nil {
			return OperationIndexPage{}, fmt.Errorf("scan release operation index: %w", err)
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return OperationIndexPage{}, fmt.Errorf("read release operation index: %w", err)
	}
	page := OperationIndexPage{OperationIDs: make([]string, 0, pageSize)}
	hasNext := len(entries) > pageSize
	if hasNext {
		entries = entries[:pageSize]
	}
	for _, entry := range entries {
		page.OperationIDs = append(page.OperationIDs, entry.ID)
	}
	if hasNext && len(entries) > 0 {
		page.NextPageToken = encodeOperationIndexCursor(operationIndexCursor{Version: 1, ProfileID: profileID, CreatedAt: entries[len(entries)-1].CreatedAt.UTC(), OperationID: entries[len(entries)-1].ID})
	}
	return page, nil
}

func encodeOperationIndexCursor(cursor operationIndexCursor) string {
	raw, _ := json.Marshal(cursor) // this fixed struct has no values that can fail JSON encoding
	return base64.RawURLEncoding.EncodeToString(raw)
}

func decodeOperationIndexCursor(token, profileID string) (operationIndexCursor, error) {
	if len(token) > maxOperationPageTokenBytes {
		return operationIndexCursor{}, ErrInvalidOperationPageToken
	}
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return operationIndexCursor{}, ErrInvalidOperationPageToken
	}
	var cursor operationIndexCursor
	if err := json.Unmarshal(raw, &cursor); err != nil || cursor.Version != 1 || cursor.ProfileID != profileID || cursor.CreatedAt.IsZero() || strings.TrimSpace(cursor.OperationID) == "" {
		return operationIndexCursor{}, ErrInvalidOperationPageToken
	}
	return cursor, nil
}
