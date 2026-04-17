package backlog

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"swarm-manager/internal/testutil"
	"testing"
)

// setupValidateGlobsHandler creates a handler with a nested rootDir inside a
// valid repo-contract fixture.
func setupValidateGlobsHandler(t *testing.T) (*Handler, string) {
	t.Helper()
	projectRoot := t.TempDir()

	writeRepoContractFixture(t, projectRoot)

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

func writeRepoContractFixture(t *testing.T, root string) {
	t.Helper()

	for _, dir := range []string{".vrooli", "scenarios", "resources", "packages", "cmd", "internal"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module fixture\n\ngo 1.24.0\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	contract := `{
  "$schema": "schemas/repo-contract.schema.json",
  "version": "1.0.0",
  "platform": {"mode": "cross_platform_go_native", "legacy_project_bash_supported": false},
  "root": {"markers": {"required_dirs": [".vrooli", "scenarios", "resources", "packages", "cmd", "internal"], "required_files": ["go.mod"]}},
  "layout": {"project_config_dir": ".vrooli", "scenario_dir": "scenarios", "resource_dir": "resources", "package_dir": "packages", "command_dir": "cmd", "internal_dir": "internal", "docs_dir": "docs"},
  "scenario": {"required_files": [".vrooli/service.json"], "well_known_paths": {"service": ".vrooli/service.json", "api": "api", "ui": "ui", "cli": "cli", "docs": "docs", "requirements": "requirements", "initialization": "initialization"}},
  "resource": {"manifest": "resource.json", "well_known_paths": {"docs": "docs", "initialization": "initialization"}},
  "globs": {"syntax": "doublestar", "root_relative": true, "case_sensitive": true, "allow_absolute": false, "path_format": "slash_normalized"},
  "environment": {"variables": {"repo_root": "VROOLI_ROOT", "source_root": "VROOLI_SOURCE_ROOT", "sandbox_id": "VROOLI_SANDBOX_ID", "sandbox_merged": "VROOLI_SANDBOX_MERGED", "sandbox_scope": "VROOLI_SANDBOX_SCOPE"}},
  "sandbox": {"full_repo_scopes": ["", ".", "/"], "scenario_scope_prefix": "scenarios/"},
  "profiles": {"mini_vrooli_bundle": {"description": "fixture profile", "parameters": ["scenario", "resources[*]"], "include": [".vrooli", "cmd", "internal", "packages", "scenarios/{scenario}", "resources/{resources[*]}"], "optional_include": ["docs", "go.mod"], "exclude": [".git/**"]}}
}`
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "repo-contract.json"), []byte(contract), 0o644); err != nil {
		t.Fatalf("write repo-contract.json: %v", err)
	}
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
