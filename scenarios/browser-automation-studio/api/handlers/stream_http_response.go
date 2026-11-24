package handlers

import (
	"fmt"
	"io"
	"net/http"
)

// streamHTTPResponse owns the common HTTP response lifecycle for binary
// streams. Callers retain their source lookup, MIME policy and error wording.
func streamHTTPResponse(w http.ResponseWriter, body io.ReadCloser, contentType string, contentLength int64, cacheControl string, onCopyError func(error)) {
	defer body.Close()

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", contentLength))
	w.Header().Set("Cache-Control", cacheControl)

	if _, err := io.Copy(w, body); err != nil && onCopyError != nil {
		onCopyError(err)
	}
}
