package httpx_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"web-search/internal/httpx"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	errorsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-search/v1/errors"
)

// TestWriteError exercises the canonical non-2xx writer end-to-end:
// status code, content-type, and the proto-typed body decoded back via
// protojson. Handlers rely on this round-trip — if the writer drifts
// from the proto-side schema, every error path silently lies on the
// wire.
func TestWriteError(t *testing.T) {
	cases := []struct {
		name        string
		status      int
		code        string
		message     string
		wantStatus  int
		wantCode    string
		wantMessage string
	}{
		{
			name:        "invalid_request",
			status:      http.StatusBadRequest,
			code:        httpx.CodeInvalidRequest,
			message:     "title required",
			wantStatus:  http.StatusBadRequest,
			wantCode:    "invalid_request",
			wantMessage: "title required",
		},
		{
			name:        "not_found",
			status:      http.StatusNotFound,
			code:        httpx.CodeNotFound,
			message:     "note 7 not found",
			wantStatus:  http.StatusNotFound,
			wantCode:    "not_found",
			wantMessage: "note 7 not found",
		},
		{
			name:        "internal",
			status:      http.StatusInternalServerError,
			code:        httpx.CodeInternal,
			message:     "transient store failure",
			wantStatus:  http.StatusInternalServerError,
			wantCode:    "internal",
			wantMessage: "transient store failure",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			httpx.WriteError(rec, tc.status, tc.code, tc.message)

			require.Equal(t, tc.wantStatus, rec.Code)
			require.Equal(t, "application/json", rec.Header().Get("Content-Type"))

			var got errorsv1.ErrorEnvelope
			err := protojson.Unmarshal(rec.Body.Bytes(), &got)
			require.NoError(t, err, "envelope must round-trip through protojson")
			require.Equal(t, tc.wantCode, got.Code)
			require.Equal(t, tc.wantMessage, got.Message)
		})
	}
}
