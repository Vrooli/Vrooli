package bindings

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenTestGenieArtifactStreamResolvesOpaqueIdentityAndUsesExactPath(t *testing.T) {
	const artifactID = "artifact_0123456789abcdef0123456789abcdef"
	const raw = "synthetic stream fixture"
	streamed := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "GetRunArtifact") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"artifact":{"id":"`+artifactID+`","kind":"findings","sizeBytes":`+strconv.Itoa(len(raw))+`,"accessCapability":"ARTIFACT_ACCESS_CAPABILITY_STREAM","accessPath":"/api/v1/scenarios/demo/runs/run-1/artifacts/`+artifactID+`"},"legacyDiscovered":false}`)
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/scenarios/demo/runs/run-1/artifacts/"+artifactID {
			streamed = true
			_, _ = io.WriteString(w, raw)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	registry, err := Load(repoRoot(t))
	require.NoError(t, err)
	registry.SetReachabilityResolver(func(context.Context, string) (string, error) { return server.URL, nil })
	artifact, body, err := registry.OpenTestGenieArtifactStream(context.Background(), "test-genie.run-evidence", "unused-digest", "demo", "run-1", artifactID, nil, InvocationMetadata{SessionID: "session", ProgramID: "program", Provenance: "PROVENANCE_TEST"}, server.Client())
	require.NoError(t, err)
	defer body.Close()
	got, err := io.ReadAll(body)
	require.NoError(t, err)
	require.True(t, streamed)
	require.Equal(t, raw, string(got))
	require.Equal(t, int64(len(raw)), artifact.SizeBytes)
	require.Equal(t, artifactID, artifact.ArtifactID)
}
