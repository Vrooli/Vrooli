package conversations

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetStoredAttachmentUsesExactAskAndAttachment(t *testing.T) {
	const askID = "9cfbaf17-75ff-4a75-9e3d-27390bea8b9f"
	const content = "<html><body>stored roadmap</body></html>"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/api/v1/asks/"+askID+"/attachments/decision.html", r.URL.Path)
		_, _ = fmt.Fprint(w, content)
	}))
	defer server.Close()

	got, digest, err := getStoredAttachment(server.Client(), server.URL, askID, "decision.html")
	require.NoError(t, err)
	require.Equal(t, content, string(got))
	want := sha256.Sum256([]byte(content))
	require.Equal(t, hex.EncodeToString(want[:]), digest)
}

func TestGetStoredAttachmentDoesNotExposeRefusalBodyOrAcceptPathTraversal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "credential=must-not-be-printed", http.StatusNotFound)
	}))
	defer server.Close()

	_, _, err := getStoredAttachment(server.Client(), server.URL, "ask", "page.html")
	require.Error(t, err)
	require.Contains(t, err.Error(), "status 404")
	require.NotContains(t, err.Error(), "credential")
	_, _, err = getStoredAttachment(server.Client(), server.URL, "ask", "../secret")
	require.Error(t, err)
	require.Contains(t, err.Error(), "one path segment")
}
