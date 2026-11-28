package bindings

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"program-runtime/internal/sessions"
)

type bridgeErrorBody struct {
	Error      string `json:"error"`
	Class      string `json:"class"`
	Status     string `json:"status"`
	BindingID  string `json:"binding_id"`
	HTTPStatus int    `json:"http_status"`
}

func postBridge(t *testing.T, handler http.Handler, payload map[string]any) (int, bridgeErrorBody) {
	t.Helper()
	body, err := json.Marshal(payload)
	require.NoError(t, err)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/internal/program-runtime/bindings/execute", bytes.NewReader(body)))
	var decoded bridgeErrorBody
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &decoded), "body=%s", recorder.Body.String())
	return recorder.Code, decoded
}

// Every bridge error body carries the closed class the kernel raises as a
// typed exception. Before this, the body was `{"error": "<text>"}` and every
// contract program re-derived the class from the text with its own copy of
// the substring table.
func TestBridgeErrorBodyCarriesTheClosedClass(t *testing.T) {
	registry := liveRegistry(t)
	manager := sessions.NewManager(sessions.Options{})
	session, err := manager.Create(context.Background(), "bridge-failure-test", "", nil)
	require.NoError(t, err)
	handler := Bridge(registry, manager)

	t.Run("ungoverned binding is no_governed_binding", func(t *testing.T) {
		code, body := postBridge(t, handler, map[string]any{"session_id": session.ID, "binding_id": "nope/nope/nope", "args": map[string]any{}})
		require.Equal(t, http.StatusBadRequest, code)
		require.Equal(t, "no_governed_binding", body.Class)
		require.Equal(t, "failed", body.Status)
		require.Equal(t, "nope/nope/nope", body.BindingID)
		require.Contains(t, body.Error, "nope/nope/nope")
	})

	t.Run("unknown session still classifies", func(t *testing.T) {
		code, body := postBridge(t, handler, map[string]any{"session_id": "sess_missing", "binding_id": "nope/nope/nope"})
		require.Equal(t, http.StatusNotFound, code)
		require.NotEmpty(t, body.Class, "every bridge error body names a class")
		require.NotEmpty(t, body.Error)
	})

	t.Run("malformed request still classifies", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/internal/program-runtime/bindings/execute", bytes.NewReader([]byte("{"))))
		require.Equal(t, http.StatusBadRequest, recorder.Code)
		var body bridgeErrorBody
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
		require.Equal(t, "binding_error", body.Class)
		require.Equal(t, "failed", body.Status)
	})
}
