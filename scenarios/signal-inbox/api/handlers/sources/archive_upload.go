package sources

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	sourcesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/signal-inbox/v1/sources"
	"signal-inbox/internal/httpx"
	internal "signal-inbox/internal/sources"
)

// Archive uploads use multipart because Connect byte messages are intentionally
// bounded. Both request and parser apply the same explicit in-process limit.
const maxArchiveUploadBytes int64 = 512 << 20

func NewArchiveUploadHandler(service *internal.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxArchiveUploadBytes)
		reader, err := r.MultipartReader()
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "valid multipart archive upload is required")
			return
		}
		var adapterID string
		var data []byte
		for {
			part, nextErr := reader.NextPart()
			if nextErr == io.EOF {
				break
			}
			if nextErr != nil {
				httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "invalid archive upload")
				return
			}
			switch part.FormName() {
			case "adapter_id":
				value, readErr := io.ReadAll(io.LimitReader(part, 256))
				if readErr != nil {
					_ = part.Close()
					httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "invalid adapter id")
					return
				}
				adapterID = strings.TrimSpace(string(value))
			case "file":
				if data != nil {
					_ = part.Close()
					httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "only one archive file is allowed")
					return
				}
				data, err = io.ReadAll(io.LimitReader(part, maxArchiveUploadBytes+1))
				if err != nil || int64(len(data)) > maxArchiveUploadBytes {
					_ = part.Close()
					httpx.WriteError(w, http.StatusRequestEntityTooLarge, httpx.CodeInvalidRequest, "archive exceeds 512 MB limit")
					return
				}
			}
			_ = part.Close()
		}
		if adapterID == "" || data == nil {
			httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "adapter_id and file are required")
			return
		}
		result, err := service.Import(r.Context(), adapterID, bytes.NewReader(data))
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, fmt.Sprintf("archive import failed: %v", err))
			return
		}
		httpx.WriteProto(w, http.StatusCreated, &sourcesv1.ImportArchiveResponse{Result: &sourcesv1.ImportResult{RunId: result.RunID, AdapterId: result.AdapterID, Created: result.Created, Duplicated: result.Duplicated, Failed: result.Failed}})
	})
}
