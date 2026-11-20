package soak

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestVirtualCorpusOnlyServesTheFixedFixture proves the accelerated soak's
// fixed corpus route cannot be pointed at caller-selected files: only the named
// clip resolves, and it serves exactly the checked-in qualification fixture.
func TestVirtualCorpusOnlyServesTheFixedFixture(t *testing.T) {
	root := scenarioRoot(t)
	t.Setenv("VROOLI_ROOT", "")
	t.Setenv("SCENARIO_ROOT", root)

	for _, target := range []string{
		"/api/v1/validation/soak/virtual-corpus",
		"/api/v1/validation/soak/virtual-corpus?clip=other",
		"/api/v1/validation/soak/virtual-corpus?clip=../../../../etc/passwd",
	} {
		recorder := httptest.NewRecorder()
		serveVirtualCorpus(recorder, httptest.NewRequest(http.MethodGet, target, nil))
		require.Equal(t, http.StatusNotFound, recorder.Code, "target %s must not serve arbitrary clips", target)
	}

	recorder := httptest.NewRecorder()
	serveVirtualCorpus(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/validation/soak/virtual-corpus?clip=quick-brown-fox", nil))
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "audio/wav", recorder.Header().Get("Content-Type"))
	require.Equal(t, "quick-brown-fox", recorder.Header().Get("X-Vrooli-Corpus-Id"))
	require.Equal(t, "the quick brown fox jumps.", recorder.Header().Get("X-Vrooli-Corpus-Reference"))

	want, err := os.ReadFile(filepath.Join(root, "bas", "fixtures", "dictation-reference.wav"))
	require.NoError(t, err)
	require.Equal(t, want, recorder.Body.Bytes(), "the route must serve the canonical qualification fixture byte-for-byte")
}

func scenarioRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, err := os.Stat(filepath.Join(dir, "bas", "fixtures", "dictation-reference.wav")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		require.NotEqual(t, dir, parent, "could not find the audio-tools scenario root above the test working directory")
		dir = parent
	}
}
