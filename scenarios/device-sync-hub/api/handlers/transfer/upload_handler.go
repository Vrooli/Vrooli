// upload_handler.go is the transfer domain's REST multipart exception: a file's
// bytes arrive as multipart/form-data (opaque bytes + filename), which cannot
// ride a proto Connect message. The RESPONSE stays proto-typed
// (UploadItemResponse → Item), so Go and TypeScript clients deserialize the
// same Item shape the Connect calls return. See handlers/notes/attachments_handler.go
// for the canonical pattern; this one adds device-token trust, quota
// pre-checks, retention/target form fields, and image thumbnailing.
package transfer

import (
	"bytes"
	"errors"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"

	"device-sync-hub/internal/deviceauth"
	"device-sync-hub/internal/httpx"
	internaltransfer "device-sync-hub/internal/transfer"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/blobstore"

	transferv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-sync-hub/v1/transfer"
)

// maxUploadBytes caps a single multipart upload. Large files spill to temp
// files (parseMemoryBytes is small) rather than buffering in memory, so the
// relay streams big payloads to disk without pinning RAM. Multi-GB transfers
// are the P2 WebRTC fast-path's job; this cap protects the relay.
const maxUploadBytes int64 = 2 << 30 // 2 GiB

// parseMemoryBytes is how much of a multipart form is buffered in memory before
// spilling to temp files. Keep it small so a large upload streams to disk.
const parseMemoryBytes int64 = 8 << 20 // 8 MiB

// thumbSourceCap bounds how large an image we buffer in memory to thumbnail.
// Above it, the file is stored streaming and simply gets no thumbnail.
const thumbSourceCap int64 = 24 << 20 // 24 MiB

var unsafeFileName = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

// UploadDeps wires the upload handler's seams.
type UploadDeps struct {
	Service     internaltransfer.Service
	Store       blobstore.BlobStore
	Thumbnailer internaltransfer.Thumbnailer
	Logger      *log.Logger
}

type uploadHandler struct {
	deps UploadDeps
}

func newUploadHandler(d UploadDeps) *uploadHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	if d.Thumbnailer == nil {
		d.Thumbnailer = internaltransfer.ImageThumbnailer{}
	}
	return &uploadHandler{deps: d}
}

func (h *uploadHandler) handleUpload(w http.ResponseWriter, r *http.Request) {
	dev, err := deviceauth.RequireDevice(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusUnauthorized, httpx.CodeUnauthenticated, "a trusted device token is required")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	if err := r.ParseMultipartForm(parseMemoryBytes); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "invalid multipart upload")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "file field is required")
		return
	}
	defer file.Close()
	if header.Size <= 0 {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "file is empty")
		return
	}

	// Reject doomed uploads before storing any bytes.
	if err := h.deps.Service.CheckQuota(r.Context(), dev.OwnerID, dev.ID, header.Size); err != nil {
		h.writeServiceError(w, err)
		return
	}

	mime := strings.TrimSpace(header.Header.Get("Content-Type"))
	if mime == "" {
		mime = "application/octet-stream"
	}
	name := uploadName(r.FormValue("name"), header.Filename)
	blobKey := itemBlobKey(dev.OwnerID, header.Filename)

	thumbKey, err := h.storeBytes(r, file, header.Size, mime, blobKey)
	if err != nil {
		h.deps.Logger.Printf("transfer.upload store(%q): %v", blobKey, err)
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "store file failed")
		return
	}

	item, err := h.deps.Service.CreateFile(r.Context(), internaltransfer.CreateFile{
		OwnerID:        dev.OwnerID,
		OriginDeviceID: dev.ID,
		Name:           name,
		MIME:           mime,
		SizeBytes:      header.Size,
		BlobKey:        blobKey,
		ThumbKey:       thumbKey,
		Retention:      parseRetention(r.FormValue("retention")),
		TargetDeviceID: strings.TrimSpace(r.FormValue("target_device_id")),
	})
	if err != nil {
		// Compensate: the metadata write failed, so the bytes we just stored
		// are orphaned. Remove them before returning.
		_ = h.deps.Store.Delete(r.Context(), blobKey)
		if thumbKey != "" {
			_ = h.deps.Store.Delete(r.Context(), thumbKey)
		}
		h.writeServiceError(w, err)
		return
	}

	httpx.WriteProto(w, http.StatusCreated, &transferv1.UploadItemResponse{Item: itemToProto(item)})
}

// storeBytes writes the upload to the blob store. For an image small enough to
// buffer, it also generates and stores a thumbnail and returns its key; in all
// other cases it streams the bytes straight through and returns an empty key.
func (h *uploadHandler) storeBytes(r *http.Request, file io.Reader, size int64, mime, blobKey string) (string, error) {
	if strings.HasPrefix(strings.ToLower(mime), "image/") && size <= thumbSourceCap {
		data, err := io.ReadAll(io.LimitReader(file, thumbSourceCap+1))
		if err != nil {
			return "", err
		}
		if err := h.deps.Store.Put(r.Context(), blobKey, bytes.NewReader(data), mime); err != nil {
			return "", err
		}
		if thumb, thumbMIME, ok := h.deps.Thumbnailer.Generate(data, mime); ok {
			thumbKey := thumbBlobKey(blobKey)
			if err := h.deps.Store.Put(r.Context(), thumbKey, bytes.NewReader(thumb), thumbMIME); err != nil {
				// A thumbnail is a nicety; its failure must not fail the upload.
				h.deps.Logger.Printf("transfer.upload thumbnail(%q): %v", thumbKey, err)
				return "", nil
			}
			return thumbKey, nil
		}
		return "", nil
	}
	if err := h.deps.Store.Put(r.Context(), blobKey, io.LimitReader(file, maxUploadBytes), mime); err != nil {
		return "", err
	}
	return "", nil
}

func (h *uploadHandler) writeServiceError(w http.ResponseWriter, err error) {
	var invalid internaltransfer.ErrInvalidItem
	var badTarget internaltransfer.ErrInvalidTarget
	var quota internaltransfer.ErrQuotaExceeded
	switch {
	case errors.As(err, &quota):
		httpx.WriteError(w, http.StatusRequestEntityTooLarge, httpx.CodeQuotaExceeded, err.Error())
	case errors.As(err, &invalid):
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, err.Error())
	case errors.As(err, &badTarget):
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, err.Error())
	default:
		h.deps.Logger.Printf("transfer.upload: %v", err)
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "upload failed")
	}
}

// parseRetention maps a form value to the domain retention. An empty/unknown
// value yields "" so the service applies the global default.
func parseRetention(v string) internaltransfer.Retention {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "live":
		return internaltransfer.RetentionLive
	case "held":
		return internaltransfer.RetentionHeld
	case "pinned":
		return internaltransfer.RetentionPinned
	default:
		return ""
	}
}

func uploadName(formName, filename string) string {
	if n := strings.TrimSpace(formName); n != "" {
		return n
	}
	base := filepath.Base(strings.TrimSpace(filename))
	if base == "." || base == string(filepath.Separator) || base == "" {
		return "upload"
	}
	return base
}

func itemBlobKey(ownerID, filename string) string {
	base := filepath.Base(strings.TrimSpace(filename))
	base = strings.Trim(unsafeFileName.ReplaceAllString(base, "-"), ".-")
	if base == "" {
		base = "upload"
	}
	return "transfer/" + safeOwnerSegment(ownerID) + "/" + uuid.NewString() + "-" + base
}

func thumbBlobKey(blobKey string) string { return blobKey + ".thumb.jpg" }

// safeOwnerSegment keeps blob paths filesystem-safe regardless of the owner id
// shape returned by the authenticator.
func safeOwnerSegment(ownerID string) string {
	s := strings.Trim(unsafeFileName.ReplaceAllString(ownerID, "-"), ".-")
	if s == "" {
		return "owner"
	}
	return s
}
