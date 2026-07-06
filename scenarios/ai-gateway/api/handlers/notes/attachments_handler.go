// attachments_handler.go is the canonical worked example of a REST
// exception in a proto/Connect-RPC scenario.
//
// Why REST here and only here: the request transport is
// multipart/form-data — opaque file bytes plus a filename. Proto
// Connect calls are JSON-over-HTTP or binary protobuf; neither can
// express a multipart upload without round-tripping bytes through
// base64, which defeats streaming and inflates payloads. The four
// mechanically-allowed REST reasons live in
// api/internal/module/module.go; this handler uses
// RESTReasonMultipartUpload.
//
// What stays proto-typed: the response. The UploadAttachmentResponse
// message is defined in notes.proto and is the source of truth for the
// metadata wire shape (id, note_id, filename, size, content_type, ...).
// The handler hand-marshals it via protojson rather than calling the
// generated Connect handler, but the message type is the same one the
// generated TS/Go clients deserialize. There is no separate REST schema.
//
// What this means for new domains: do NOT copy this pattern unless the
// request transport genuinely cannot be expressed in proto. The
// validateTransport pass in cmd/gen-endpoints rejects literal-string
// EndpointDescriptor.Path values that lack a RESTException tag, so the
// failure surfaces at codegen time — not after the second scenario has
// already drifted to REST.
package notes

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"

	"ai-gateway/internal/httpx"
	"ai-gateway/internal/notes"
	notesflow "ai-gateway/internal/notes/flow"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/blobstore"

	notesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/notes"
)

const maxMultipartFormBytes int64 = 32 << 20

var unsafeFileName = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

type AttachmentsDeps struct {
	Service notes.AttachmentsService
	Store   blobstore.BlobStore
	Logger  *log.Logger
}

func NewAttachmentsHandler(d AttachmentsDeps) http.Handler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	h := &attachmentsHandler{deps: d}
	r := mux.NewRouter()
	r.HandleFunc("/api/v1/notes/{id}/attachments", h.handleUpload).Methods(http.MethodPost)
	return r
}

type attachmentsHandler struct {
	deps AttachmentsDeps
}

func (h *attachmentsHandler) handleUpload(w http.ResponseWriter, r *http.Request) {
	uploadState := notesflow.InitialAttachmentUploadState()
	noteID := mux.Vars(r)["id"]
	r.Body = http.MaxBytesReader(w, r.Body, maxMultipartFormBytes)
	if err := r.ParseMultipartForm(maxMultipartFormBytes); err != nil {
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

	mime := strings.TrimSpace(header.Header.Get("Content-Type"))
	if mime == "" {
		mime = "application/octet-stream"
	}
	key := attachmentKey(noteID, header.Filename)
	if err := h.deps.Store.Put(r.Context(), key, io.LimitReader(file, maxMultipartFormBytes), mime); err != nil {
		_, _ = notesflow.TransitionAttachmentUpload(uploadState, notesflow.AttachmentUploadFail)
		h.deps.Logger.Printf("notes.Attachments.Put(%q): %v", key, err)
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "store attachment failed")
		return
	}
	uploadState, err = notesflow.TransitionAttachmentUpload(uploadState, notesflow.AttachmentUploadStoreBytes)
	if err != nil {
		h.deps.Logger.Printf("notes.Attachments.workflow(%q): %v", key, err)
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "invalid attachment upload state")
		return
	}

	created, err := h.deps.Service.Create(r.Context(), notes.CreateAttachmentInput{
		NoteID:    noteID,
		Key:       key,
		MIMEType:  mime,
		SizeBytes: header.Size,
	})
	if err != nil {
		_, _ = notesflow.TransitionAttachmentUpload(uploadState, notesflow.AttachmentUploadFail)
		_ = h.deps.Store.Delete(r.Context(), key)
		status, code := attachmentErrorStatus(err)
		if status == http.StatusInternalServerError {
			h.deps.Logger.Printf("notes.Attachments.Create(%q): %v", key, err)
		}
		httpx.WriteError(w, status, code, err.Error())
		return
	}
	if _, err := notesflow.TransitionAttachmentUpload(uploadState, notesflow.AttachmentUploadRecordMetadata); err != nil {
		h.deps.Logger.Printf("notes.Attachments.workflow(%q): %v", key, err)
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "invalid attachment upload state")
		return
	}

	httpx.WriteProto(w, http.StatusCreated, &notesv1.UploadAttachmentResponse{Attachment: attachmentToProto(created)})
}

func attachmentKey(noteID, name string) string {
	base := filepath.Base(strings.TrimSpace(name))
	if base == "." || base == string(filepath.Separator) || base == "" {
		base = "upload"
	}
	base = strings.Trim(unsafeFileName.ReplaceAllString(base, "-"), ".-")
	if base == "" {
		base = "upload"
	}
	return fmt.Sprintf("notes/%s/attachments/%s-%s", noteID, uuid.NewString(), base)
}

func attachmentErrorStatus(err error) (int, string) {
	var invalid notes.ErrInvalidNote
	if errors.As(err, &invalid) {
		return http.StatusBadRequest, httpx.CodeInvalidRequest
	}
	var notFound notes.ErrNoteNotFound
	if errors.As(err, &notFound) {
		return http.StatusNotFound, httpx.CodeNotFound
	}
	return http.StatusInternalServerError, httpx.CodeInternal
}
