package bundles

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestCommandsCoverBundleLifecycle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/bundles/assemble":
			_, _ = io.WriteString(w, `{"status":"assembled","schema":"v1","manifest":{"app":{"name":"demo"},"services":[]}}`)
		case "/api/v1/bundles/export":
			_, _ = io.WriteString(w, `{"status":"exported","schema":"v1","scenario":"demo","tier":"tier-2-desktop","manifest":{"app":{"name":"demo"},"services":[]},"checksum":"abc","generated_at":"now"}`)
		case "/api/v1/bundles/validate":
			_, _ = io.WriteString(w, `{"status":"valid","schema":"v1"}`)
		default:
			_, _ = io.WriteString(w, `{}`)
		}
	}))
	defer server.Close()
	cmd := New(testAPIClient(server.URL))
	if err := cmd.Run([]string{"help"}); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Assemble([]string{"demo", "--tier", "tier3", "--profile", "p1", "--include-secrets=false", "--format", "json"}); err != nil {
		t.Fatalf("assemble json: %v", err)
	}
	assembled := filepath.Join(t.TempDir(), "assembled.json")
	if err := cmd.Assemble([]string{"demo", "--output", assembled}); err != nil {
		t.Fatalf("assemble file: %v", err)
	}
	if _, err := os.Stat(assembled); err != nil {
		t.Fatal(err)
	}
	exported := filepath.Join(t.TempDir(), "exported.json")
	if err := cmd.Export([]string{"demo", "--profile", "p1", "--tier", "enterprise", "--output", exported}); err != nil {
		t.Fatalf("export file: %v", err)
	}
	if err := cmd.Export([]string{"demo", "--manifest-only"}); err != nil {
		t.Fatalf("export manifest only: %v", err)
	}
	if err := cmd.Export([]string{"demo", "--format", "json"}); err != nil {
		t.Fatalf("export json: %v", err)
	}
	if err := cmd.Validate([]string{exported}); err != nil {
		t.Fatalf("validate table: %v", err)
	}
	if err := cmd.Validate([]string{exported, "--format", "json"}); err != nil {
		t.Fatalf("validate json: %v", err)
	}
	for _, tier := range []string{"desktop", "2", "tier-2", "mobile", "3", "saas", "4", "enterprise", "5", "tier-9", "unknown"} {
		if normalizeTier(tier) == "" {
			t.Errorf("empty normalized tier for %q", tier)
		}
	}
}

func TestCommandsRejectBundleInput(t *testing.T) {
	cmd := New(testAPIClient("http://127.0.0.1:1"))
	for _, args := range [][]string{{"unknown"}, {"assemble"}, {"export"}, {"validate"}, {"validate", filepath.Join(t.TempDir(), "missing.json")}} {
		if err := cmd.Run(args); err == nil {
			t.Errorf("expected error for %v", args)
		}
	}
	bad := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(bad, []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Validate([]string{bad}); err == nil {
		t.Fatal("expected invalid JSON error")
	}
}
