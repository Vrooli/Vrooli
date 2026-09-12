// download_handler.go is the transfer domain's streaming-download REST edge.
// Download is a REST exception because the response body is opaque file bytes
// (optionally many GB) streamed with the original filename — a payload no proto
// message can carry. Metadata still flows proto-typed via GetItem; this endpoint
// is purely the byte channel.
package transfer

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"device-sync-hub/internal/deviceauth"
	"device-sync-hub/internal/httpx"
	internaltransfer "device-sync-hub/internal/transfer"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/blobstore"
)

// unsafeHeaderChars strips characters that cannot appear unescaped in a quoted
// Content-Disposition filename (control chars, quotes, backslash, path seps).
var unsafeHeaderChars = regexp.MustCompile(`[^\x20-\x7e]|["\\/]`)

// urlEncode percent-encodes s for the RFC 5987 `filename*=UTF-8”` form, where
// a space must be %20 (not the +/x-www-form-urlencoded form QueryEscape emits).
func urlEncode(s string) string {
	return strings.ReplaceAll(url.QueryEscape(s), "+", "%20")
}

type downloadHandler struct {
	deps DownloadDeps
}

// DownloadDeps wires the download handler's seams.
type DownloadDeps struct {
	Service internaltransfer.Service
	Store   blobstore.BlobStore
}

func newDownloadHandler(d DownloadDeps) *downloadHandler { return &downloadHandler{deps: d} }

func (h *downloadHandler) handleDownload(w http.ResponseWriter, r *http.Request) {
	dev, err := deviceauth.RequireDevice(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, httpx.CodeUnauthenticated, "a trusted device token is required")
		return
	}
	id := mux.Vars(r)["id"]
	item, err := h.deps.Service.Get(r.Context(), dev.OwnerID, dev.ID, id)
	if err != nil {
		var notFound internaltransfer.ErrItemNotFound
		if errors.As(err, &notFound) {
			httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, "item not found")
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "download failed")
		return
	}

	// Text items carry their body inline — no blob fetch.
	if item.Kind == internaltransfer.KindText {
		w.Header().Set("Content-Type", item.MIME)
		w.Header().Set("Content-Disposition", contentDisposition("inline", textDownloadName(item)))
		w.Header().Set("Content-Length", strconv.Itoa(len(item.Text)))
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, item.Text)
		h.markDelivered(r, item)
		return
	}

	// ?thumb=1 streams the generated thumbnail instead of the original bytes.
	wantThumb := r.URL.Query().Get("thumb") == "1"
	key := item.BlobKey
	if wantThumb {
		if item.ThumbKey == "" {
			httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, "no thumbnail for this item")
			return
		}
		key = item.ThumbKey
	}

	body, mime, err := h.deps.Store.Get(r.Context(), key)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, "item content not found")
		return
	}
	defer body.Close()

	if mime == "" {
		mime = item.MIME
	}
	w.Header().Set("Content-Type", mime)
	if wantThumb {
		w.Header().Set("Content-Disposition", contentDisposition("inline", item.Name))
	} else {
		w.Header().Set("Content-Disposition", contentDisposition("attachment", item.Name))
		w.Header().Set("Content-Length", strconv.FormatInt(item.SizeBytes, 10))
	}
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, body); err != nil {
		// The client likely hung up mid-stream; nothing actionable, headers are
		// already sent. Do not mark delivered on a broken transfer.
		return
	}
	if !wantThumb {
		h.markDelivered(r, item)
	}
}

// markDelivered flags a successfully-pulled Live item so the purge sweep
// removes it. Only Live items care; Held/Pinned ignore delivery.
func (h *downloadHandler) markDelivered(r *http.Request, item internaltransfer.Item) {
	if item.Retention == internaltransfer.RetentionLive {
		h.deps.Service.MarkDelivered(r.Context(), item.OwnerID, item.ID)
	}
}

func textDownloadName(item internaltransfer.Item) string {
	if n := strings.TrimSpace(item.Name); n != "" {
		return n
	}
	return "snippet-" + item.ID + ".txt"
}

// contentDisposition builds a header value that is safe for arbitrary filenames:
// a sanitized ASCII fallback plus an RFC 5987 UTF-8 form for modern clients.
func contentDisposition(disp, filename string) string {
	filename = strings.TrimSpace(filename)
	if filename == "" {
		return disp
	}
	ascii := unsafeHeaderChars.ReplaceAllString(filename, "_")
	return disp + `; filename="` + ascii + `"; filename*=UTF-8''` + urlEncode(filename)
}
