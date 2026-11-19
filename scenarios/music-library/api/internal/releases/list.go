package releases

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"time"
)

type ReleaseSummary struct {
	ReleaseID  string    `json:"release_id"`
	Revision   int64     `json:"revision"`
	Kind       string    `json:"kind"`
	Status     string    `json:"status"`
	Title      string    `json:"title"`
	CreatedAt  time.Time `json:"created_at"`
	TrackCount int       `json:"track_count"`
}

type ReleasePage struct {
	Items      []ReleaseSummary `json:"items"`
	NextCursor string           `json:"next_cursor,omitempty"`
}

type listCursor struct {
	CreatedAt string `json:"created_at"`
	ReleaseID string `json:"release_id"`
}

// List returns one stable page of latest immutable revisions, without loading
// the nested credits, owner assets, rights evidence, or audio.
func (s *Store) List(ctx context.Context, limit int, cursor string) (ReleasePage, error) {
	if limit < 1 || limit > 50 {
		return ReleasePage{}, ErrInvalid
	}
	var key listCursor
	if cursor != "" {
		decoded, err := base64.RawURLEncoding.DecodeString(cursor)
		if err != nil || json.Unmarshal(decoded, &key) != nil || key.ReleaseID == "" {
			return ReleasePage{}, ErrInvalid
		}
		if _, err := time.Parse(time.RFC3339Nano, key.CreatedAt); err != nil {
			return ReleasePage{}, ErrInvalid
		}
	}
	query := `SELECT v.release_id,v.revision,v.kind,v.status,v.title,v.created_at,COUNT(t.track_order)
FROM music_release_versions v
JOIN (SELECT release_id,MAX(revision) AS revision FROM music_release_versions GROUP BY release_id) latest
  ON latest.release_id=v.release_id AND latest.revision=v.revision
LEFT JOIN music_release_track_occurrences t ON t.release_id=v.release_id AND t.revision=v.revision`
	args := []any{}
	if cursor != "" {
		query += ` WHERE (v.created_at < ? OR (v.created_at = ? AND v.release_id < ?))`
		args = append(args, key.CreatedAt, key.CreatedAt, key.ReleaseID)
	}
	query += ` GROUP BY v.release_id,v.revision ORDER BY v.created_at DESC,v.release_id DESC LIMIT ?`
	args = append(args, limit+1)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return ReleasePage{}, err
	}
	defer rows.Close()
	page := ReleasePage{Items: []ReleaseSummary{}}
	for rows.Next() {
		var item ReleaseSummary
		var created string
		if err := rows.Scan(&item.ReleaseID, &item.Revision, &item.Kind, &item.Status, &item.Title, &created, &item.TrackCount); err != nil {
			return ReleasePage{}, err
		}
		if item.CreatedAt, err = time.Parse(time.RFC3339Nano, created); err != nil {
			return ReleasePage{}, err
		}
		page.Items = append(page.Items, item)
	}
	if err := rows.Err(); err != nil {
		return ReleasePage{}, err
	}
	if len(page.Items) > limit {
		page.Items = page.Items[:limit]
		last := page.Items[len(page.Items)-1]
		encoded, err := json.Marshal(listCursor{CreatedAt: last.CreatedAt.Format(time.RFC3339Nano), ReleaseID: last.ReleaseID})
		if err != nil {
			return ReleasePage{}, err
		}
		page.NextCursor = base64.RawURLEncoding.EncodeToString(encoded)
	}
	return page, nil
}
