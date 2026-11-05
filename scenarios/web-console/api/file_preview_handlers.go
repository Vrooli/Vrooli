package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

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

	resolved, err := a.srv.filePreviewResolver.ResolveDetailed(cwd, cwdErr, in.Path)
	if err != nil {
		return filePreviewH.ResolveResult{}, mapFilePreviewError(err)
	}

	// A relative path that matched nowhere is not an error: the client opens a
	// folder browser rooted at the returned search bases to locate it manually.
	if resolved.Unresolved {
		return filePreviewH.ResolveResult{
			InputPath:   in.Path,
			Unresolved:  true,
			SearchBases: toHandlerCandidates(resolved.SearchBases),
		}, nil
	}

	target := resolved.Selected
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
		ListingAvailable:     target.ListingAvailable,
		BlobURL:              filePreviewBlobURL(in.SessionID, previewID),
		ExpiresUnixNano:      expiry.UnixNano(),
		Warnings:             target.Warnings,
		Candidates:           toHandlerCandidates(resolved.Candidates),
	}, nil
}

// toHandlerCandidates projects resolver candidates onto the transport-neutral
// handler shape (Kind as its string form).
func toHandlerCandidates(in []filepreview.ResolveCandidate) []filePreviewH.ResolveCandidate {
	if len(in) == 0 {
		return nil
	}
	out := make([]filePreviewH.ResolveCandidate, 0, len(in))
	for _, c := range in {
		out = append(out, filePreviewH.ResolveCandidate{
			ResolvedPath:    c.ResolvedPath,
			ResolutionBasis: c.ResolutionBasis,
			Kind:            string(c.Kind),
			CanPreview:      c.CanPreview,
			Basename:        c.Basename,
		})
	}
	return out
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
	if entry.RootID != "" {
		if err := a.srv.filePreviews.Fresh(entry); err != nil {
			return filePreviewH.TextResult{}, mapFilePreviewError(err)
		}
		data := entry.Data
		if !utf8.Valid(data) {
			return filePreviewH.TextResult{}, filePreviewH.ErrPreviewUnavailable
		}
		truncated := int64(len(data)) > filepreview.DefaultMaxTextBytes
		if truncated {
			data = data[:filepreview.DefaultMaxTextBytes]
			for !utf8.Valid(data) {
				data = data[:len(data)-1]
			}
		}

		return filePreviewH.TextResult{ResolvedPath: entry.ResolvedPath, Kind: string(entry.Kind), MIMEType: entry.MIMEType, Content: string(data), Truncated: truncated}, nil
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

// ListDirectory serves one bounded page of a previously-resolved directory.
// It accepts only the opaque preview id, so a directory handle can never be
// repointed at another path — the same guarantee the blob route relies on.
func (a *filePreviewAdapter) ListDirectory(_ context.Context, in filePreviewH.ListInput) (filePreviewH.ListResult, error) {
	entry, err := a.srv.filePreviews.Lookup(in.SessionID, in.PreviewID)
	if err != nil {
		return filePreviewH.ListResult{}, fmt.Errorf("preview %q: %w", sanitizeID(in.PreviewID), filePreviewH.ErrNotFound)
	}
	if !entry.ListingAvailable {
		return filePreviewH.ListResult{}, fmt.Errorf("preview is not a directory: %w", filePreviewH.ErrPreviewUnavailable)
	}

	page, listErr := a.srv.filePreviewResolver.ListDirectory(entry.ResolvedPath, filepreview.ListOptions{
		Sort:       filepreview.Sort(in.Sort),
		ShowHidden: in.ShowHidden,
		PageSize:   in.PageSize,
		PageToken:  in.PageToken,
	})
	if listErr != nil {
		return filePreviewH.ListResult{}, mapFilePreviewError(listErr)
	}

	entries := make([]filePreviewH.ListEntry, 0, len(page.Entries))
	for _, e := range page.Entries {
		entries = append(entries, filePreviewH.ListEntry{
			Name:            e.Name,
			EntryType:       string(e.Type),
			Kind:            string(e.Kind),
			SizeBytes:       e.SizeBytes,
			ModTimeUnixNano: e.ModTimeUnixNano,
			CanPreview:      e.CanPreview,
			SymlinkTarget:   e.SymlinkTarget,
			SymlinkBroken:   e.SymlinkBroken,
			Mode:            e.Mode,
			ChildCount:      e.ChildCount,
		})
	}

	return filePreviewH.ListResult{
		ResolvedPath:  page.ResolvedPath,
		ParentPath:    page.ParentPath,
		Entries:       entries,
		TotalEntries:  page.TotalEntries,
		Truncated:     page.Truncated,
		NextPageToken: page.NextPageToken,
		EffectiveSort: string(page.EffectiveSort),
		Warnings:      page.Warnings,
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

	entry, release, err := s.filePreviews.AcquireBlob(sessionID, previewID)
	if err != nil {
		// Unknown, expired, or session-mismatched id — never disclose which.
		http.Error(w, "preview not found", http.StatusNotFound)
		return
	}

	defer release()

	if r.URL.Query().Get("representation") == "sqlite-snapshot" {
		s.serveSQLiteSnapshot(w, r, entry)
		return
	}

	if entry.RootID != "" {
		if err := s.filePreviews.Fresh(entry); err != nil {
			http.Error(w, "archive preview expired or changed; reopen the original archive", http.StatusConflict)
			return
		}
		h := w.Header()
		h.Set("Content-Type", entry.MIMEType)
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Cache-Control", "no-store")
		h.Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; sandbox")
		h.Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": sanitizeContentDispositionFilename(entry.Basename)}))
		servePreviewContent(w, r, entry.Basename, time.Time{}, bytes.NewReader(entry.Data))
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
	servePreviewContent(w, r, entry.Basename, info.ModTime(), file)
}

// thumbnailSlots bounds concurrent thumbnail decodes so a folder full of
// photos scrolled into view cannot saturate the host's CPU.
var thumbnailSlots = make(chan struct{}, 2)

// handleFilePreviewThumbnail serves a small rendition of an image. The preview
// id stays the capability: for a directory id, `entry` names one child (a bare
// name — no separators, no dot segments), which is exactly what listing that
// directory already disclosed; for a file id, `entry` must be empty. Formats
// the host cannot thumbnail answer 415 so the client keeps its icon.
func (s *Server) handleFilePreviewThumbnail(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	entry, err := s.filePreviews.Lookup(vars["id"], vars["previewId"])
	if err != nil || entry.RootID != "" {
		http.Error(w, "preview not found", http.StatusNotFound)
		return
	}
	target := entry.ResolvedPath
	name := r.URL.Query().Get("entry")
	switch {
	case entry.Kind == filepreview.KindDirectory:
		if name == "" || name == "." || name == ".." || strings.ContainsAny(name, "/\\\x00") {
			http.Error(w, "entry must name one child of the directory", http.StatusBadRequest)
			return
		}
		target = filepath.Join(entry.ResolvedPath, name)
	case name != "":
		http.Error(w, "entry is only valid for a directory preview", http.StatusBadRequest)
		return
	}

	info, statErr := os.Stat(target)
	if statErr != nil || !info.Mode().IsRegular() {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}
	// The thumbnail is a pure function of the file's identity and the size, so
	// a revalidation is answered without decoding anything.
	edge := filepreview.ThumbnailDefaultEdge
	if v := r.URL.Query().Get("size"); v != "" {
		if _, scanErr := fmt.Sscanf(v, "%d", &edge); scanErr != nil {
			http.Error(w, "size must be an integer", http.StatusBadRequest)
			return
		}
	}
	etag := fmt.Sprintf(`"t%x-%x-%d"`, info.ModTime().UnixNano(), info.Size(), edge)
	h := w.Header()
	h.Set("Cache-Control", "private, max-age=600")
	h.Set("ETag", etag)
	h.Set("X-Content-Type-Options", "nosniff")
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	select {
	case thumbnailSlots <- struct{}{}:
		defer func() { <-thumbnailSlots }()
	case <-r.Context().Done():
		return
	}
	data, contentType, thumbErr := filepreview.Thumbnail(target, edge)
	if thumbErr != nil {
		h.Del("ETag")
		h.Set("Cache-Control", "no-store")
		if errors.Is(thumbErr, filepreview.ErrThumbnailUnsupported) {
			http.Error(w, "thumbnail not available for this file", http.StatusUnsupportedMediaType)
			return
		}
		http.Error(w, "failed to read file", http.StatusInternalServerError)
		return
	}
	h.Set("Content-Type", contentType)
	h.Set("Content-Length", fmt.Sprint(len(data)))
	if r.Method == http.MethodHead {
		return
	}
	_, _ = w.Write(data)
}

// handleFilePreviewRuntime serves an HTML preview and the small set of local
// presentation resources it references. The preview id remains the only
// capability: the resource path is resolved below the HTML file's directory,
// never from a caller-supplied filesystem path.
func (s *Server) handleFilePreviewRuntime(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]
	previewID := vars["previewId"]
	resource := vars["resource"]

	entry, err := s.filePreviews.Lookup(sessionID, previewID)
	if err != nil || entry.RootID != "" || !isHTMLPreviewEntry(entry) {
		http.NotFound(w, r)
		return
	}

	root, err := filepath.Abs(filepath.Dir(entry.ResolvedPath))
	if err != nil {
		http.Error(w, "preview root unavailable", http.StatusInternalServerError)
		return
	}
	resourcePath, status := previewRuntimePath(root, entry.ResolvedPath, resource)
	if status != http.StatusOK {
		http.Error(w, http.StatusText(status), status)
		return
	}

	info, statErr := os.Stat(resourcePath)
	if statErr != nil {
		if os.IsNotExist(statErr) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "failed to stat preview resource", http.StatusInternalServerError)
		return
	}
	if !info.Mode().IsRegular() || (resource != "" && !previewRuntimeResourceAllowed(resourcePath)) {
		http.Error(w, "preview resource type is not allowed", http.StatusForbidden)
		return
	}
	if resource == "" && (info.Size() != entry.SizeBytes || info.ModTime().UnixNano() != entry.ModTimeUnixNano) {
		http.Error(w, "file changed since preview was opened; reopen to refresh", http.StatusConflict)
		return
	}

	// Resolve the root and resource through symlinks before serving. This keeps
	// a symlink inside the presentation directory from escaping its capability.
	resolvedRoot, rootErr := filepath.EvalSymlinks(root)
	resolvedResource, resourceErr := filepath.EvalSymlinks(resourcePath)
	if rootErr != nil || resourceErr != nil || !pathWithinRoot(resolvedRoot, resolvedResource) {
		http.Error(w, "preview resource is outside the presentation directory", http.StatusForbidden)
		return
	}
	// Media is streamed from disk by ServeContent, including native HEAD and
	// byte-range requests. Its total length does not determine memory use; a
	// presentation-asset limit must not reject multi-gigabyte video or audio.
	if info.Size() > previewRuntimeMaxResourceBytes && !previewRuntimeMedia(resourcePath) {
		http.Error(w, "preview resource is too large", http.StatusRequestEntityTooLarge)
		return
	}

	file, openErr := os.Open(resolvedResource)
	if openErr != nil {
		if os.IsNotExist(openErr) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "failed to open preview resource", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	h := w.Header()
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Cache-Control", "no-store")
	// Sandboxed iframes have an opaque Origin (`null`). This route is already
	// protected by the session-bound bearer preview id and never accepts
	// credentials, so wildcard CORS is safe here and lets module scripts/fonts
	// load without widening the API's credentialed CORS policy.
	h.Set("Access-Control-Allow-Origin", "*")
	h.Set("Access-Control-Allow-Methods", "GET, HEAD")
	h.Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", sanitizeContentDispositionFilename(filepath.Base(resourcePath))))
	if resource == "" || previewRuntimeHTML(resourcePath) {
		h.Set("Content-Type", "text/html; charset=utf-8")
		h.Del("X-Frame-Options")
		// The Web Console may be reached through a reverse proxy or secure
		// tunnel, so the parent origin is not knowable from local UI_PORT.
		// This document is safe to embed broadly because it is still an opaque,
		// script-only sandbox with no credentials, forms, navigation, or network.
		// Permit same-capability child HTML documents (for example, a design
		// index that opens a responsive page in an iframe) and same-origin
		// relative links. The server-side runtime route still limits every
		// resulting request to this preview directory and extension allowlist.
		h.Set("Content-Security-Policy", "default-src 'none'; base-uri 'self'; script-src 'unsafe-inline' 'self'; style-src 'unsafe-inline' 'self'; img-src 'self' data: blob:; font-src 'self' data:; media-src 'self' data: blob:; connect-src 'none'; form-action 'none'; frame-src 'self'; object-src 'none'; navigate-to 'self'")
	} else {
		h.Set("Content-Type", previewRuntimeContentType(resourcePath))
	}
	servePreviewContent(w, r, filepath.Base(resourcePath), info.ModTime(), file)
}

// Preview transfers may outlast the server's ordinary response budget. Refresh
// the write deadline for each chunk so active streams can continue while a
// blocked client is still bounded. The wrapper also keeps copies buffered in
// small chunks rather than loading the file into memory.
func servePreviewContent(w http.ResponseWriter, r *http.Request, name string, modTime time.Time, content io.ReadSeeker) {
	stream := &previewStreamWriter{ResponseWriter: w, idleTimeout: httpWriteTimeout}
	http.ServeContent(stream, r, name, modTime, content)
}

type previewStreamWriter struct {
	http.ResponseWriter
	idleTimeout time.Duration
}

func (w *previewStreamWriter) Write(p []byte) (int, error) {
	err := http.NewResponseController(w.ResponseWriter).SetWriteDeadline(time.Now().Add(w.idleTimeout))
	if err != nil && !errors.Is(err, http.ErrNotSupported) {
		return 0, err
	}
	return w.ResponseWriter.Write(p)
}

func (w *previewStreamWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

const previewRuntimeMaxResourceBytes int64 = 16 << 20

func isHTMLPreviewEntry(entry *filepreview.Entry) bool {
	if entry == nil || entry.Kind != filepreview.KindCode {
		return false
	}
	return previewRuntimeHTML(entry.Basename)
}

func previewRuntimePath(root, document, resource string) (string, int) {
	if resource == "" {
		return document, http.StatusOK
	}
	decoded := strings.ReplaceAll(resource, "\\", "/")
	if strings.HasPrefix(decoded, "/") {
		return "", http.StatusBadRequest
	}
	for _, segment := range strings.Split(decoded, "/") {
		if segment == ".." {
			return "", http.StatusBadRequest
		}
	}
	clean := path.Clean("/" + decoded)
	if clean == "/" || strings.HasPrefix(clean, "/../") || clean == "/.." {
		return "", http.StatusBadRequest
	}
	relative := strings.TrimPrefix(clean, "/")
	if relative == "" || filepath.IsAbs(relative) {
		return "", http.StatusBadRequest
	}
	return filepath.Join(root, filepath.FromSlash(relative)), http.StatusOK
}

func pathWithinRoot(root, candidate string) bool {
	rel, err := filepath.Rel(root, candidate)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

func previewRuntimeResourceAllowed(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".html", ".htm", ".css", ".js", ".mjs", ".png", ".jpg", ".jpeg", ".gif", ".webp", ".avif", ".bmp", ".ico", ".svg", ".woff", ".woff2", ".ttf", ".otf":
		return true
	default:
		return previewRuntimeMedia(name)
	}
}

func previewRuntimeMedia(name string) bool {
	return filepreview.IsMediaFilename(name)
}

func previewRuntimeHTML(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".html" || ext == ".htm"
}

func previewRuntimeContentType(name string) string {
	if ext := strings.ToLower(filepath.Ext(name)); ext == ".js" || ext == ".mjs" {
		return "text/javascript; charset=utf-8"
	}
	if ext := strings.ToLower(filepath.Ext(name)); ext == ".css" {
		return "text/css; charset=utf-8"
	}
	if m := mime.TypeByExtension(strings.ToLower(filepath.Ext(name))); m != "" {
		return m
	}
	return "application/octet-stream"
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
	if errors.Is(err, filepreview.ErrPreviewNotFound) {
		return filePreviewH.ErrNotFound
	}
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
	case filepreview.CodeStale:
		return fmt.Errorf("%s: %w", fe.Message, filePreviewH.ErrStale)
	default:
		return fmt.Errorf("%s", fe.Message)
	}
}

func (a *filePreviewAdapter) ListArchive(ctx context.Context, sessionID, previewID string) (filePreviewH.ArchiveResult, error) {
	res := a.srv.filePreviews.ListArchive(ctx, sessionID, previewID)
	out := filePreviewH.ArchiveResult{Format: res.Format, Status: res.Status}
	for _, e := range res.Entries {
		out.Entries = append(out.Entries, filePreviewH.ArchiveEntry{ID: e.ID, Name: e.Name, Status: e.Status, SizeBytes: e.SizeBytes})
	}
	return out, nil
}
func (a *filePreviewAdapter) OpenArchiveEntry(ctx context.Context, sessionID, previewID, entryID string) (filePreviewH.ArchiveOpenResult, error) {
	e, status := a.srv.filePreviews.OpenArchiveEntry(ctx, sessionID, previewID, entryID)
	out := filePreviewH.ArchiveOpenResult{Status: status}
	if e == nil {
		return out, nil
	}
	out.Target = &filePreviewH.ResolveResult{PreviewID: e.ID, InputPath: e.ResolvedPath, ResolvedPath: e.ResolvedPath, Basename: e.Basename, ResolutionBasis: "archive", Kind: string(e.Kind), MIMEType: e.MIMEType, SizeBytes: e.SizeBytes, CanPreview: e.CanPreview, CanDownload: e.CanDownload, SupportsRange: true, TextContentAvailable: e.TextContentAvailable, BlobURL: filePreviewBlobURL(sessionID, e.ID), ExpiresUnixNano: e.ExpiresAt.UnixNano(), ArchiveMember: true}
	return out, nil
}
