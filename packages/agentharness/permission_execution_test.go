package agentharness

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const executionDocument = `{"schema_version":"v2","rules":[],"execution":{"filesystem":{"workspace":"write"},"network":{"enabled":true,"domains":{"localhost":"allow","127.0.0.1":"allow"}},"approval":{"policy":"on-request","reviewer":"user"}}}`

func TestExecutionDocumentVersionsAndCapabilities(t *testing.T) {
	d, err := parsePermissionDocument([]byte(executionDocument), "fixture")
	if err != nil {
		t.Fatal(err)
	}
	if err = RequireExecutionCapability(d, "codex"); err != nil {
		t.Fatal(err)
	}
	for _, runner := range []string{"claude-code", "opencode", "grok", "antigravity", "future"} {
		if err = RequireExecutionCapability(d, runner); err == nil {
			t.Fatalf("%s silently accepts unsupported execution intent", runner)
		}
	}
	for _, invalid := range []string{
		strings.Replace(executionDocument, `"v2"`, `"v1"`, 1),
		strings.Replace(executionDocument, `"v2"`, `"v3"`, 1),
		strings.Replace(executionDocument, `"localhost"`, `"*"`, 1),
		strings.Replace(executionDocument, `"localhost"`, `"localhost:8080"`, 1),
		strings.Replace(executionDocument, `"enabled":true`, `"enabled":false`, 1),
		strings.Replace(executionDocument, `"enabled":true`, `"enabled":null`, 1),
		strings.Replace(executionDocument, `"reviewer":"user"`, `"reviewer":"auto_review"`, 1),
	} {
		if strings.Contains(invalid, `"reviewer":"auto_review"`) {
			invalid = strings.Replace(invalid, `"on-request"`, `"never"`, 1)
		}
		if _, err = parsePermissionDocument([]byte(invalid), "fixture"); err == nil {
			t.Fatalf("accepted invalid document %s", invalid)
		}
	}
}

func TestPermissionFilePublicationRejectsStalePreviewAndSymlink(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config")
	if err := os.WriteFile(path, []byte("private"), 0o600); err != nil {
		t.Fatal(err)
	}
	before, err := ReadPermissionFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = PublishPermissionFile(before, []byte("updated")); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(path)
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatal("private file mode widened")
	}
	if err = PublishPermissionFile(before, []byte("stale")); err == nil {
		t.Fatal("accepted stale snapshot")
	}
	if err = os.Symlink(path, path+".link"); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err = ReadPermissionFile(path + ".link"); err == nil {
		t.Fatal("accepted symlink")
	}
}

func TestPermissionFileNewFilesArePrivate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "new")
	before, err := ReadPermissionFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = PublishPermissionFile(before, []byte("data")); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(path)
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatal("new file not private")
	}
}

func TestMissingPermissionStateIsNotAnEmptyDocument(t *testing.T) {
	state, err := LoadPermissionState(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil || state != nil {
		t.Fatalf("missing state: %+v, %v", state, err)
	}
}
