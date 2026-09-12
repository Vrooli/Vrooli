package workflows

import (
	"bytes"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/vrooli/browser-automation-studio/internal/compat"
)

const adhocRequestBodyLimit = 16 << 20

// normalizeAdhocRequest applies the same short-form workflow compatibility
// rules used by the other BAS ingress paths before Connect decodes the proto.
// Generic CLI clients send the flow file directly as proto JSON, so this
// boundary must normalize execution_mode before protojson sees the enum.
func normalizeAdhocRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !strings.HasSuffix(r.URL.Path, "/ExecuteAdhocWorkflow") || !strings.Contains(strings.ToLower(r.Header.Get("Content-Type")), "json") {
			next.ServeHTTP(w, r)
			return
		}

		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, adhocRequestBodyLimit))
		if err != nil {
			http.Error(w, "adhoc workflow request is too large", http.StatusRequestEntityTooLarge)
			return
		}
		normalized, err := compat.NormalizeExecuteAdhocRequest(body)
		if err != nil {
			http.Error(w, "invalid adhoc workflow request: "+err.Error(), http.StatusBadRequest)
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(normalized))
		r.ContentLength = int64(len(normalized))
		r.Header.Set("Content-Length", strconv.Itoa(len(normalized)))
		next.ServeHTTP(w, r)
	})
}
