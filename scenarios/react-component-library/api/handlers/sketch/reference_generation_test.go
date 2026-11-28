package sketch

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReferenceGenerationValidatesBeforeResolvingImageTools(t *testing.T) {
	calls := 0
	h := &referenceGenerationHTTP{resolve: func(context.Context) (string, error) { calls++; return "", nil }, logger: log.New(io.Discard, "", 0)}
	for _, body := range []string{"{", `{"scenario":"../demo","page":"home","prompt":"a sufficiently long reference prompt"}`, `{"scenario":"demo","page":"home","prompt":"too short"}`, `{"scenario":"demo","page":"home","prompt":"a sufficiently long reference prompt","width":63}`} {
		rec := httptest.NewRecorder()
		h.generate(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body)))
		require.Equal(t, http.StatusBadRequest, rec.Code)
	}
	require.Zero(t, calls)
}

func TestReferenceGenerationSubmitsWaitsAndStoresResult(t *testing.T) {
	var submitted map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/ai/text_to_image":
			require.NoError(t, r.ParseMultipartForm(1<<20))
			require.NoError(t, json.Unmarshal([]byte(r.FormValue("params")), &submitted))
			w.WriteHeader(http.StatusAccepted)
			_, _ = io.WriteString(w, `{"job_id":"job-1","model_id":"model","tier":"local","warnings":["fallback"]}`)
		case "/vrooli.image_tools.v1.jobs.JobsService/WaitJob":
			var payload map[string]string
			require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
			require.Equal(t, "job-1", payload["id"])
			_, _ = io.WriteString(w, `{"job":{"state":"JOB_STATE_SUCCEEDED","resultRef":"results/image.png"}}`)
		case "/api/v1/blobs/results/image.png":
			_, _ = io.WriteString(w, "png bytes")
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	root := t.TempDir()
	h := &referenceGenerationHTTP{repoRoot: root, resolve: func(context.Context) (string, error) { return server.URL, nil }, client: server.Client(), logger: log.New(io.Discard, "", 0)}
	body := `{"scenario":"demo","page":"home","prompt":"A detailed dashboard reference with clear hierarchy","seed":42}`
	rec := httptest.NewRecorder()
	h.generate(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body)))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var result generateReferenceResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
	require.Equal(t, "job-1", result.JobID)
	require.Equal(t, "model", result.ModelID)
	require.Equal(t, []string{"fallback"}, result.Warnings)
	require.Equal(t, 512, int(submitted["width"].(float64)))
	require.Equal(t, "/api/v1/reference-assets/demo/home/"+result.Name, result.URL)
	stored, err := os.ReadFile(filepath.Join(root, "scenarios", "demo", "data", "experience", "reference-assets", "home", result.Name))
	require.NoError(t, err)
	require.Equal(t, "png bytes", string(stored))
}

func TestReferenceGenerationRejectsUnsuccessfulJobsAndUnsafeResultReferences(t *testing.T) {
	for _, tc := range []struct{ state, ref, detail string }{{"JOB_STATE_FAILED", "", "provider failed"}, {"JOB_STATE_SUCCEEDED", "../secret", "invalid result reference"}} {
		t.Run(tc.state+tc.ref, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/api/v1/ai/text_to_image" {
					w.WriteHeader(http.StatusAccepted)
					_, _ = io.WriteString(w, `{"jobId":"job"}`)
					return
				}
				if r.URL.Path == "/vrooli.image_tools.v1.jobs.JobsService/WaitJob" {
					payload, _ := json.Marshal(map[string]any{"job": map[string]string{"state": tc.state, "resultRef": tc.ref, "error": tc.detail}})
					_, _ = w.Write(payload)
					return
				}
				http.NotFound(w, r)
			}))
			defer server.Close()
			h := &referenceGenerationHTTP{repoRoot: t.TempDir(), resolve: func(context.Context) (string, error) { return server.URL, nil }, client: server.Client(), logger: log.New(io.Discard, "", 0)}
			rec := httptest.NewRecorder()
			h.generate(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"scenario":"demo","page":"home","prompt":"A detailed dashboard reference with clear hierarchy"}`)))
			require.Equal(t, http.StatusBadGateway, rec.Code)
			require.Contains(t, rec.Body.String(), tc.detail)
		})
	}
}
