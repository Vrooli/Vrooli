package backlog

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"swarm-manager/internal/testutil"
)

// setupValidateGlobsHandler creates a handler with a nested rootDir that
// simulates the real layout: <projectRoot>/scenarios/swarm-manager.
// This makes filepath.Dir(filepath.Dir(rootDir)) resolve to projectRoot.
func setupValidateGlobsHandler(t *testing.T) (*Handler, string) {
	t.Helper()
	projectRoot := t.TempDir()
	// Mimic the real layout: rootDir = projectRoot/scenarios/swarm-manager
	rootDir := filepath.Join(projectRoot, "scenarios", "swarm-manager")
	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, dir := range backlogKindDirs {
		testutil.MakeDir(t, filepath.Join(rootDir, dir))
	}
	disableAutoWorkshopSettings(t, rootDir)
	return NewHandler(rootDir), projectRoot
}

func TestValidateGlobsHandler(t *testing.T) {
	h, projectRoot := setupValidateGlobsHandler(t)

	// Create a fake file so the matching-pattern test can find something.
	testDir := filepath.Join(projectRoot, "testdir")
	if err := os.MkdirAll(testDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(testDir, "hello.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		body       validateGlobsRequest
		wantStatus int
		check      func(t *testing.T, resp validateGlobsResponse)
	}{
		{
			name:       "empty patterns returns empty results",
			body:       validateGlobsRequest{Patterns: []string{}},
			wantStatus: http.StatusOK,
			check: func(t *testing.T, resp validateGlobsResponse) {
				if len(resp.Results) != 0 {
					t.Errorf("expected 0 results, got %d", len(resp.Results))
				}
			},
		},
		{
			name:       "invalid syntax returns valid=false",
			body:       validateGlobsRequest{Patterns: []string{"[unclosed"}},
			wantStatus: http.StatusOK,
			check: func(t *testing.T, resp validateGlobsResponse) {
				if len(resp.Results) != 1 {
					t.Fatalf("expected 1 result, got %d", len(resp.Results))
				}
				if resp.Results[0].Valid {
					t.Error("expected valid=false for invalid syntax")
				}
				if resp.Results[0].Error == "" {
					t.Error("expected error message for invalid syntax")
				}
			},
		},
		{
			name:       "absolute path returns valid=false",
			body:       validateGlobsRequest{Patterns: []string{"/etc/passwd"}},
			wantStatus: http.StatusOK,
			check: func(t *testing.T, resp validateGlobsResponse) {
				if len(resp.Results) != 1 {
					t.Fatalf("expected 1 result, got %d", len(resp.Results))
				}
				if resp.Results[0].Valid {
					t.Error("expected valid=false for absolute path")
				}
			},
		},
		{
			name:       "valid non-matching pattern returns warning",
			body:       validateGlobsRequest{Patterns: []string{"nonexistent-dir-xyz/**"}},
			wantStatus: http.StatusOK,
			check: func(t *testing.T, resp validateGlobsResponse) {
				if len(resp.Results) != 1 {
					t.Fatalf("expected 1 result, got %d", len(resp.Results))
				}
				r := resp.Results[0]
				if !r.Valid {
					t.Error("expected valid=true for syntactically valid pattern")
				}
				if r.MatchCount != 0 {
					t.Errorf("expected matchCount=0, got %d", r.MatchCount)
				}
				if r.Warning == "" {
					t.Error("expected warning for non-matching pattern")
				}
			},
		},
		{
			name:       "valid matching pattern returns count",
			body:       validateGlobsRequest{Patterns: []string{"testdir/*.txt"}},
			wantStatus: http.StatusOK,
			check: func(t *testing.T, resp validateGlobsResponse) {
				if len(resp.Results) != 1 {
					t.Fatalf("expected 1 result, got %d", len(resp.Results))
				}
				r := resp.Results[0]
				if !r.Valid {
					t.Errorf("expected valid=true, got false (error: %s)", r.Error)
				}
				if r.MatchCount < 1 {
					t.Errorf("expected matchCount >= 1, got %d", r.MatchCount)
				}
				if r.Warning != "" {
					t.Errorf("expected no warning for matching pattern, got %q", r.Warning)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyBytes, err := json.Marshal(tt.body)
			if err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequest(http.MethodPost, "/api/v1/backlog/validate-globs", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			h.ValidateGlobs(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", w.Code, tt.wantStatus, w.Body.String())
			}

			if tt.check != nil {
				var resp validateGlobsResponse
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				tt.check(t, resp)
			}
		})
	}
}
