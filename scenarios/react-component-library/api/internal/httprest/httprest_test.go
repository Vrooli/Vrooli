package httprest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteErrorUsesTheOneEnvelope(t *testing.T) {
	recorder := httptest.NewRecorder()
	WriteError(recorder, http.StatusNotFound, "not_found", "workspace not found")

	require.Equal(t, http.StatusNotFound, recorder.Code)
	require.Equal(t, "application/json", recorder.Header().Get("Content-Type"))

	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Equal(t, "not_found", body.Error.Code)
	require.Equal(t, "workspace not found", body.Error.Message)
}

func TestWriteJSONKeepsTheGivenStatus(t *testing.T) {
	recorder := httptest.NewRecorder()
	WriteJSON(recorder, http.StatusCreated, map[string]any{"asset": map[string]string{"id": "a"}})
	require.Equal(t, http.StatusCreated, recorder.Code)
	require.JSONEq(t, `{"asset":{"id":"a"}}`, recorder.Body.String())
}

func TestEtagMatchesAcceptsStoredFormsOnly(t *testing.T) {
	require.True(t, EtagMatches("abc", "abc"))
	require.True(t, EtagMatches(`"abc"`, "abc"))
	require.True(t, EtagMatches(`W/"abc"`, "abc"))
	require.True(t, EtagMatches(`"other", W/"abc"`, "abc"))
	require.True(t, EtagMatches("*", "abc"))
	require.False(t, EtagMatches(`"other"`, "abc"))
	require.False(t, EtagMatches("", "abc"))
}
