package signals

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/vrooli/api-core/blobstore"
	"signal-inbox/internal/httpx"

	signalsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/signal-inbox/v1/signals"
)

const maxImageUploadBytes int64 = 32 << 20

func NewImageUploadHandler(store blobstore.BlobStore, logger *log.Logger) http.Handler {
	if logger == nil {
		logger = log.Default()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxImageUploadBytes)
		if err := r.ParseMultipartForm(maxImageUploadBytes); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "invalid multipart image upload")
			return
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "file field is required")
			return
		}
		defer file.Close()
		if header.Size <= 0 {
			httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "image is empty")
			return
		}
		mime := strings.TrimSpace(header.Header.Get("Content-Type"))
		if !strings.HasPrefix(strings.ToLower(mime), "image/") {
			httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "file must have an image content type")
			return
		}
		data, err := io.ReadAll(io.LimitReader(file, maxImageUploadBytes+1))
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "read image failed")
			return
		}
		if int64(len(data)) > maxImageUploadBytes {
			httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "image exceeds maximum size")
			return
		}
		key := imagePayloadKey(header.Filename, data)
		if err := store.Put(r.Context(), key, bytes.NewReader(data), mime); err != nil {
			logger.Printf("signals image upload %q: %v", key, err)
			httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "store image failed")
			return
		}
		httpx.WriteProto(w, http.StatusCreated, &signalsv1.UploadImageResponse{Image: &signalsv1.ImageUpload{PayloadRef: key, ContentType: mime, SizeBytes: header.Size}})
	})
}

func imagePayloadKey(_ string, data []byte) string {
	sum := sha256.Sum256(data)
	return fmt.Sprintf("signals/uploads/%s", hex.EncodeToString(sum[:]))
}
