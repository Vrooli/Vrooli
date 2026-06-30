package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"

	filePreviewH "web-console/handlers/file_preview"
	"web-console/internal/filepreview"
)

// filePreviewAdapter implements filePreviewH.Service against the server's
// session manager, the filepreview resolver, and the preview-id store.
type filePreviewAdapter struct {
	srv *Server
}

func newFilePreviewAdapter(s *Server) *filePreviewAdapter {
	return &filePreviewAdapter{srv: s}
}

// Resolve looks up the session, resolves the path into a Target, issues an
// opaque session-bound preview id, and returns the rich metadata plus a
// same-origin blob URL.
func (a *filePreviewAdapter) Resolve(ctx context.Context, in filePreviewH.ResolveInput) (filePreviewH.ResolveResult, error) {
	sess, ok := a.srv.sessions.Get(in.SessionID)
	if !ok {
		return filePreviewH.ResolveResult{}, fmt.Errorf("session %q: %w", sanitizeID(in.SessionID), filePreviewH.ErrSessionNotFound)
	}

	var cwd string
	cwd, cwdErr := sess.CurrentDir(ctx)
	if cwdErr != nil {
		cwd = ""
	}

	target, err := a.srv.filePreviewResolver.Resolve(cwd, cwdErr, in.Path)
	if err != nil {
		return filePreviewH.ResolveResult{}, mapFilePreviewError(err)
	}

	previewID, expiry, issueErr := a.srv.filePreviews.Issue(in.SessionID, target)
	if issueErr != nil {
		return filePreviewH.ResolveResult{}, fmt.Errorf("issue preview id: %w", issueErr)
	}

	return filePreviewH.ResolveResult{
		PreviewID:            previewID,
		InputPath:            target.InputPath,
		ResolvedPath:         target.ResolvedPath,
		Basename:             target.Basename,
		Line:                 target.Line,
		HasLine:              target.HasLine,
		ResolutionBasis:      target.ResolutionBasis,
		Kind:                 string(target.Kind),
		MIMEType:             target.MIMEType,
		SizeBytes:            target.SizeBytes,
		ModTimeUnixNano:      target.ModTimeUnixNano,
		CanPreview:           target.CanPreview,
		CanDownload:          target.CanDownload,
		SupportsRange:        target.SupportsRange,
		TextContentAvailable: target.TextContentAvailable,
		BlobURL:              filePreviewBlobURL(in.SessionID, previewID),
		ExpiresUnixNano:      expiry.UnixNano(),
		Warnings:             target.Warnings,
	}, nil
}

// GetTextContent serves bounded UTF-8 content for a previously-resolved text
// preview, keyed by the opaque session-bound preview id.
func (a *filePreviewAdapter) GetTextContent(_ context.Context, sessionID, previewID string) (filePreviewH.TextResult, error) {
	entry, err := a.srv.filePreviews.Lookup(sessionID, previewID)
	if err != nil {
		return filePreviewH.TextResult{}, fmt.Errorf("preview %q: %w", sanitizeID(previewID), filePreviewH.ErrNotFound)
	}
	if !entry.TextContentAvailable {
		return filePreviewH.TextResult{}, fmt.Errorf("preview kind has no inline text content: %w", filePreviewH.ErrPreviewUnavailable)
	}
	content, truncated, readErr := a.srv.filePreviewResolver.ReadText(&filepreview.Target{
		ResolvedPath:         entry.ResolvedPath,
		Kind:                 entry.Kind,
		TextContentAvailable: entry.TextContentAvailable,
	})
	if readErr != nil {
		return filePreviewH.TextResult{}, mapFilePreviewError(readErr)
	}
	return filePreviewH.TextResult{
		ResolvedPath: entry.ResolvedPath,
		Kind:         string(entry.Kind),
		MIMEType:     entry.MIMEType,
		Content:      content,
		Truncated:    truncated,
	}, nil
}

// filePreviewBlobURL builds the same-origin relative blob path for a preview id.
func filePreviewBlobURL(sessionID, previewID string) string {
	return fmt.Sprintf("/api/v1/sessions/%s/file-previews/%s/blob", sessionID, previewID)
}

// handleFilePreviewBlob streams the bytes of a resolved preview id with HTTP
// Range support. It accepts only opaque preview ids (never raw paths), binds
// them to the session, re-stats the file to reject stale/changed/deleted
// targets, and sets safe headers (no-store, nosniff, explicit Content-Type).
func (s *Server) handleFilePreviewBlob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]
	previewID := vars["previewId"]

	entry, err := s.filePreviews.Lookup(sessionID, previewID)
	if err != nil {
		// Unknown, expired, or session-mismatched id — never disclose which.
		http.Error(w, "preview not found", http.StatusNotFound)
		return
	}

	info, statErr := os.Stat(entry.ResolvedPath)
	if statErr != nil {
		if os.IsNotExist(statErr) {
			http.Error(w, "file no longer exists", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to stat file", http.StatusInternalServerError)
		return
	}
	if info.IsDir() {
		http.Error(w, "preview target is a directory", http.StatusConflict)
		return
	}
	// Reject if the file changed since resolve — a swapped file must not be
	// served under a stale id. Force a reopen/refresh.
	if info.Size() != entry.SizeBytes || info.ModTime().UnixNano() != entry.ModTimeUnixNano {
		http.Error(w, "file changed since preview was opened; reopen to refresh", http.StatusConflict)
		return
	}

	file, openErr := os.Open(entry.ResolvedPath)
	if openErr != nil {
		if os.IsNotExist(openErr) {
			http.Error(w, "file no longer exists", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to open file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	disposition := "inline"
	if !entry.CanPreview {
		// Unsupported kinds are download-only.
		disposition = "attachment"
	}

	h := w.Header()
	h.Set("Content-Type", entry.MIMEType)
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Cache-Control", "no-store")
	h.Set("Content-Disposition", fmt.Sprintf("%s; filename=%q", disposition, sanitizeContentDispositionFilename(entry.Basename)))

	// ServeContent honors the already-set Content-Type, adds Accept-Ranges,
	// and handles Range (206/Content-Range/416) and HEAD automatically.
	http.ServeContent(w, r, entry.Basename, info.ModTime(), file)
}

// sanitizeContentDispositionFilename strips quotes and control characters so
// the basename is safe inside a quoted Content-Disposition filename.
func sanitizeContentDispositionFilename(name string) string {
	return strings.Map(func(rr rune) rune {
		if rr < 32 || rr == 127 || rr == '"' || rr == '\\' {
			return '_'
		}
		return rr
	}, name)
}

// mapFilePreviewError translates filepreview.Error codes into the file_preview
// handler sentinels so the Connect handler picks the right code.
func mapFilePreviewError(err error) error {
	var fe *filepreview.Error
	if !errors.As(err, &fe) {
		return err
	}
	switch fe.Code {
	case filepreview.CodeInvalid:
		return fmt.Errorf("%s: %w", fe.Message, filePreviewH.ErrInvalidArgument)
	case filepreview.CodeNotAllowed:
		return fmt.Errorf("%s: %w", fe.Message, filePreviewH.ErrPermissionDenied)
	case filepreview.CodeNotPreviewable:
		return fmt.Errorf("%s: %w", fe.Message, filePreviewH.ErrPreviewUnavailable)
	case filepreview.CodeNotFound, filepreview.CodeUnresolvable:
		return fmt.Errorf("%s: %w", fe.Message, filePreviewH.ErrNotFound)
	default:
		return fmt.Errorf("%s", fe.Message)
	}
}
