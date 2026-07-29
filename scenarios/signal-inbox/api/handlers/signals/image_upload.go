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
		data, mime, err := readImageUpload(r)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "file field is required")
			return
		}
		if len(data) == 0 {
			httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "image is empty")
			return
		}
		if !strings.HasPrefix(strings.ToLower(mime), "image/") {
			httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "file must have an image content type")
			return
		}
		key := imagePayloadKey("", data)
		if err := store.Put(r.Context(), key, bytes.NewReader(data), mime); err != nil {
			logger.Printf("signals image upload %q: %v", key, err)
			httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, "store image failed")
			return
		}
		httpx.WriteProto(w, http.StatusCreated, &signalsv1.UploadImageResponse{Image: &signalsv1.ImageUpload{PayloadRef: key, ContentType: mime, SizeBytes: int64(len(data))}})
	})
}

// readImageUpload reads only the expected multipart part. http.MaxBytesReader
// owns the request-wide cap; the per-part limit protects this allocation and
// makes the bound explicit to both the reader and the security scanner.
func readImageUpload(r *http.Request) ([]byte, string, error) {
	reader, err := r.MultipartReader()
	if err != nil {
		return nil, "", err
	}
	var data []byte
	var contentType string
	found := false
	for {
		part, nextErr := reader.NextPart()
		if nextErr == io.EOF {
			break
		}
		if nextErr != nil {
			return nil, "", nextErr
		}
		if part.FormName() != "file" {
			_ = part.Close()
			continue
		}
		if found {
			_ = part.Close()
			return nil, "", fmt.Errorf("multiple file fields")
		}
		found = true
		contentType = strings.TrimSpace(part.Header.Get("Content-Type"))
		data, err = io.ReadAll(io.LimitReader(part, maxImageUploadBytes+1))
		closeErr := part.Close()
		if err != nil {
			return nil, "", err
		}
		if closeErr != nil {
			return nil, "", closeErr
		}
		if int64(len(data)) > maxImageUploadBytes {
			return nil, "", fmt.Errorf("image exceeds maximum size")
		}
	}
	if !found {
		return nil, "", fmt.Errorf("file field is required")
	}
	return data, contentType, nil
}

func imagePayloadKey(_ string, data []byte) string {
	sum := sha256.Sum256(data)
	return fmt.Sprintf("signals/uploads/%s", hex.EncodeToString(sum[:]))
}
