package protocodegen

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestBufGenYamlContainsNoRemotePlugins enforces CD-1 from
// docs/plans/proto-codegen-local-and-bsr-login-implementation-plan.md:
// every plugin in packages/proto/buf.gen.yaml must be `local:` or
// `protoc_builtin:`. A regression to `remote:` re-introduces BSR
// rate-limit and offline-availability failures, so this is a hard guard.
func TestBufGenYamlContainsNoRemotePlugins(t *testing.T) {
	path := bufGenYamlPath(t)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	for i, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if !strings.HasPrefix(line, "- remote:") && !strings.HasPrefix(line, "remote:") {
			continue
		}
		t.Fatalf(
			"%s line %d: %q references a BSR remote plugin; codegen must use `local:` or `protoc_builtin:` (see CD-1 in docs/plans/proto-codegen-local-and-bsr-login-implementation-plan.md)",
			path, i+1, line,
		)
	}
}

// bufGenYamlPath resolves packages/proto/buf.gen.yaml relative to this
// test file rather than the working directory, so the test passes whether
// it runs via `go test ./...`, from the package dir, or from a workspace.
func bufGenYamlPath(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	// internal/protocodegen/buf_gen_guard_test.go → repo root → packages/proto/buf.gen.yaml
	repoRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
	return filepath.Join(repoRoot, "packages", "proto", "buf.gen.yaml")
}
