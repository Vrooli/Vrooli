package main

import (
	"bytes"
	"context"
	"errors"
	"mime"
	"net/http"
	"time"

	"web-console/internal/filepreview"
)

func (s *Server) serveSQLiteSnapshot(w http.ResponseWriter, r *http.Request, entry *filepreview.Entry) {
	h := w.Header()
	h.Set("Cache-Control", "no-store")
	h.Set("X-Content-Type-Options", "nosniff")
	if entry.Kind != filepreview.KindDatabase {
		http.Error(w, "not a SQLite preview", http.StatusBadRequest)
		return
	}
	var data []byte
	var err error
	if entry.RootID != "" {
		err = s.filePreviews.Fresh(entry)
		data = entry.Data
		if int64(len(data)) > filepreview.MaxSQLiteBytes {
			err = filepreview.ErrSQLiteLimit
		}
	} else {
		data, err = filepreview.SQLiteSnapshot(r.Context(), entry.ResolvedPath, entry.SizeBytes, entry.ModTimeUnixNano)
	}
	if err != nil {
		status := http.StatusUnprocessableEntity
		var fe *filepreview.Error
		switch {
		case errors.Is(err, filepreview.ErrSQLiteLimit):
			status = http.StatusRequestEntityTooLarge
		case errors.Is(err, filepreview.ErrSQLiteBusy):
			status = http.StatusServiceUnavailable
			h.Set("Retry-After", "2")
		case errors.Is(err, context.DeadlineExceeded):
			status = http.StatusGatewayTimeout
		case errors.As(err, &fe) && fe.Code == filepreview.CodeStale:
			status = http.StatusConflict
		}
		http.Error(w, "SQLite snapshot unavailable", status)
		return
	}
	h.Set("Content-Type", "application/vnd.sqlite3")
	h.Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": sanitizeContentDispositionFilename(entry.Basename)}))
	h.Set("X-Preview-Snapshot-Time", time.Now().UTC().Format(time.RFC3339))
	servePreviewContent(w, r, entry.Basename, time.Time{}, bytes.NewReader(data))
}
