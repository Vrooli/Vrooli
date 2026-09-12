package httpx_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/httpx"
	"audio-tools/internal/logx"

	errorsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/errors"
)

func TestWriteProto(t *testing.T) {
	rec := httptest.NewRecorder()
	httpx.WriteProto(rec, http.StatusOK, &errorsv1.ErrorEnvelope{Code: "ok", Message: "msg"})
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	require.Contains(t, rec.Body.String(), "ok")
}

func TestSetPackageLogger(t *testing.T) {
	prev := httpx.SetPackageLogger(logx.Std{})
	t.Cleanup(func() { httpx.SetPackageLogger(prev) })
	require.NotNil(t, prev)
}
