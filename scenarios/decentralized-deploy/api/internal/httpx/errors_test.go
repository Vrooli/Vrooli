package httpx_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"decentralized-deploy/internal/httpx"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	errorsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/decentralized-deploy/v1/shared"
)

func noOpErrorLogger() httpx.ErrorLogger {
	return httpx.ErrorLogFunc(func(string, ...any) {})
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
			httpx.WriteError(rec, noOpErrorLogger(), tc.status, tc.code, tc.message)

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

func TestWriteErrorMarshalFailureUsesInjectedLogger(t *testing.T) {
	var entries []string
	logger := httpx.ErrorLogFunc(func(format string, args ...any) {
		entries = append(entries, fmt.Sprintf(format, args...))
	})
	rec := httptest.NewRecorder()
	httpx.WriteError(rec, logger, http.StatusBadRequest, httpx.CodeInvalidRequest, string([]byte{0xff}))

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.JSONEq(t, `{"code":"internal","message":"error envelope marshal failed"}`, rec.Body.String())
	require.Len(t, entries, 1)
	require.Contains(t, entries[0], "httpx.WriteError: protojson marshal failed:")
}

func TestWriteProtoSuccess(t *testing.T) {
	rec := httptest.NewRecorder()
	httpx.WriteProto(rec, noOpErrorLogger(), http.StatusOK, structpb.NewStringValue("ready"))

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	require.JSONEq(t, `"ready"`, rec.Body.String())
}

func TestWriteProtoMarshalFailureUsesInjectedLogger(t *testing.T) {
	var entries []string
	logger := httpx.ErrorLogFunc(func(format string, args ...any) {
		entries = append(entries, fmt.Sprintf(format, args...))
	})
	invalid := &structpb.Value{Kind: &structpb.Value_StringValue{StringValue: string([]byte{0xff})}}
	rec := httptest.NewRecorder()
	httpx.WriteProto(rec, logger, http.StatusCreated, invalid)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	var got errorsv1.ErrorEnvelope
	require.NoError(t, protojson.Unmarshal(rec.Body.Bytes(), &got))
	require.Equal(t, httpx.CodeInternal, got.Code)
	require.Equal(t, "response marshal failed", got.Message)
	require.Len(t, entries, 1)
	require.Contains(t, entries[0], "httpx.WriteProto: protojson marshal failed:")
}
