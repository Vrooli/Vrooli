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
	"{{SCENARIO_ID}}/internal/httpx"
	"{{SCENARIO_ID}}/internal/notes"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/blobstore"

	notesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/{{SCENARIO_ID}}/v1/notes"
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
	uploadState := notes.InitialAttachmentUploadState()
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
		_, _ = notes.TransitionAttachmentUpload(uploadState, notes.AttachmentUploadFail)
		h.deps.Logger.Printf("notes.Attachments.Put(%q): %v", key, err)
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "store attachment failed")
		return
	}
	uploadState, err = notes.TransitionAttachmentUpload(uploadState, notes.AttachmentUploadStoreBytes)
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
		_, _ = notes.TransitionAttachmentUpload(uploadState, notes.AttachmentUploadFail)
		_ = h.deps.Store.Delete(r.Context(), key)
		status, code := attachmentErrorStatus(err)
		if status == http.StatusInternalServerError {
			h.deps.Logger.Printf("notes.Attachments.Create(%q): %v", key, err)
		}
		httpx.WriteError(w, status, code, err.Error())
		return
	}
	if _, err := notes.TransitionAttachmentUpload(uploadState, notes.AttachmentUploadRecordMetadata); err != nil {
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
