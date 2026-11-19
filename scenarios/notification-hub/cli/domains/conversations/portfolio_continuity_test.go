package conversations

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRequestPortfolioContinuityUsesExactAskIDAndChecksStoredRecipient(t *testing.T) {
	const askID = "9cfbaf17-75ff-4a75-9e3d-27390bea8b9f"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/api/v1/conversations/asks/"+askID+"/portfolio-continuity", r.URL.Path)
		_, _ = fmt.Fprint(w, `{"ask_id":"`+askID+`","source":"portfolio-steward","recipient":"alice","subject_key":"portfolio-steward:roadmap:2026-11-16","attested":"true"}`)
	}))
	defer server.Close()

	result, err := requestPortfolioContinuity(server.Client(), server.URL, askID)
	require.NoError(t, err)
	require.Equal(t, askID, result.AskID)
	require.Equal(t, "alice", result.Recipient)
}

func TestRequestPortfolioContinuityDoesNotExposeRefusalBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "credential=must-not-be-printed", http.StatusNotFound)
	}))
	defer server.Close()

	_, err := requestPortfolioContinuity(server.Client(), server.URL, "some-ask")
	require.Error(t, err)
	require.Contains(t, err.Error(), "status 404")
	require.NotContains(t, err.Error(), "credential")
}
