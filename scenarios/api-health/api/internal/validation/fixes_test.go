package validation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFixRegistryPreviewApplyAndIdempotency(t *testing.T) {
	root := apiFixFixture(t)
	registry := NewFixRegistry()

	preview, err := registry.Preview(root, []string{CodeServiceHealthMissing, CodeRawStatusCode, CodeContentTypeMissing, CodeResponseBodyUnclosed})
	require.NoError(t, err)
	require.Len(t, preview, 4)
	require.False(t, preview[0].Applied)
	require.NotContains(t, readFile(t, filepath.Join(root, ".vrooli", "service.json")), "${API_PORT}")
	require.Contains(t, readFile(t, filepath.Join(root, "api", "handler.go")), "WriteHeader(404)")

	applied, err := registry.Apply(root, []string{CodeServiceHealthMissing, CodeRawStatusCode, CodeContentTypeMissing, CodeResponseBodyUnclosed})
	require.NoError(t, err)
	require.Len(t, applied, 4)
	for _, candidate := range applied {
		require.True(t, candidate.Applied)
		require.NotEmpty(t, candidate.Before)
		require.NotEmpty(t, candidate.After)
	}
	service := readFile(t, filepath.Join(root, ".vrooli", "service.json"))
	require.Contains(t, service, `"api": "/health"`)
	source := readFile(t, filepath.Join(root, "api", "handler.go"))
	require.Contains(t, source, "http.StatusNotFound")
	require.Contains(t, source, `w.Header().Set("Content-Type", "application/json")`)
	require.Contains(t, source, "defer resp.Body.Close()")

	second, err := registry.Apply(root, []string{CodeServiceHealthMissing, CodeRawStatusCode, CodeContentTypeMissing, CodeResponseBodyUnclosed})
	require.NoError(t, err)
	require.Empty(t, second)
}

func TestFixRegistryAddsHealthEndpointDescriptor(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".vrooli", "endpoints.json"), `{"endpoints":[]}`)

	preview, err := NewFixRegistry().Preview(root, []string{CodeHealthEndpointMissing})
	require.NoError(t, err)
	require.Len(t, preview, 1)
	require.Contains(t, preview[0].After, `"/health"`)
	require.Contains(t, preview[0].After, `"rest_exception": true`)
}

func apiFixFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".vrooli", "service.json"), `{"ports":{"ui":{}},"lifecycle":{"health":{"endpoints":{"api":"/healthz"}}}}`)
	writeFile(t, filepath.Join(root, "api", "handler.go"), `package main

import (
	"encoding/json"
	"net/http"
)

func handler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(404)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "missing"})
	resp, err := http.Get("https://example.test")
	if err != nil {
		return
	}
	_ = resp.StatusCode
}
`)
	return root
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(raw)
}
