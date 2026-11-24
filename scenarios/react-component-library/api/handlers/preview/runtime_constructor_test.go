package preview

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

func TestDefaultRuntimeHandlerRejectsUnvendoredVersionWithCors(t *testing.T) {
	handler := NewRuntimeHandler(nil)
	request := httptest.NewRequest(http.MethodGet, "/preview/runtime/react@0.0.0", nil)
	request = mux.SetURLVars(request, map[string]string{"module": "react", "version": "0.0.0"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	require.Equal(t, http.StatusNotFound, response.Code)
	require.Equal(t, "*", response.Header().Get("Access-Control-Allow-Origin"))
	require.Contains(t, response.Body.String(), "react@0.0.0 is not vendored")
}
