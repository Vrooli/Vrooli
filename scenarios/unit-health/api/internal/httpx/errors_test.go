package httpx_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"unit-health/internal/httpx"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	errorsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/unit-health/v1/errors"
)

type recordingLogger struct{ lines []string }

func (l *recordingLogger) Printf(format string, args ...any) {
	l.lines = append(l.lines, strings.TrimSpace(fmt.Sprintf(format, args...)))
}

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

func TestWriteProtoWithLoggerSubstitutionRecordsMarshalFailure(t *testing.T) {
	logger := &recordingLogger{}
	rec := httptest.NewRecorder()
	httpx.WriteProtoWithLogger(rec, http.StatusOK, &errorsv1.ErrorEnvelope{Message: string([]byte{0xff})}, logger)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Len(t, logger.lines, 1)
	require.Contains(t, logger.lines[0], "httpx.WriteProto: protojson marshal failed")
}

func TestWriteProtoEmitsProtoJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	httpx.WriteProto(rec, http.StatusCreated, &errorsv1.ErrorEnvelope{Code: "ok", Message: "created"})
	require.Equal(t, http.StatusCreated, rec.Code)
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	var got errorsv1.ErrorEnvelope
	require.NoError(t, protojson.Unmarshal(rec.Body.Bytes(), &got))
	require.Equal(t, "ok", got.Code)
}
