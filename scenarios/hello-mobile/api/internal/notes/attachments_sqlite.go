package notes

import (
	"context"
	"fmt"
)

const (
	insertAttachmentSQL = `
INSERT INTO attachments (key, note_id, mime_type, size_bytes, uploaded_at)
VALUES (?, ?, ?, ?, ?)
`

	listAttachmentKeysSQL = `
SELECT key
FROM attachments
WHERE note_id = ?
ORDER BY uploaded_at DESC, key DESC
`
)

func (s *sqliteRepository) CreateAttachment(ctx context.Context, a Attachment) (Attachment, error) {
	if a.UploadedAt.IsZero() {
		a.UploadedAt = s.clock.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, insertAttachmentSQL,
		a.Key,
		a.NoteID,
		a.MIMEType,
		a.SizeBytes,
		a.UploadedAt.Format(noteTimeFormat),
	)
	if err != nil {
		return Attachment{}, fmt.Errorf("insert attachment %q: %w", a.Key, err)
	}
	return a, nil
}

func (s *sqliteRepository) ListAttachmentKeys(ctx context.Context, noteID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, listAttachmentKeysSQL, noteID)
	if err != nil {
		return nil, fmt.Errorf("list attachment keys for note %q: %w", noteID, err)
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, fmt.Errorf("scan attachment key: %w", err)
		}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate attachment keys: %w", err)
	}
	return keys, nil
}

var _ AttachmentsRepository = (*sqliteRepository)(nil)
