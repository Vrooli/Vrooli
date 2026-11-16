package hosthome_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agent-manager/internal/testutil/hosthome"
)

func writeAuth(t *testing.T, root, content string) string {
	t.Helper()
	dir := filepath.Join(root, "003", "run-id", "runtime", "codex")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "auth.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestFindLeaksFlagsNonDummyCredentialWithoutPrintingIt(t *testing.T) {
	const secret = `{"tokens":{"refresh_token":"not-a-real-token-but-treated-as-one"}}`
	root := t.TempDir()
	path := writeAuth(t, root, secret)
	leaks := hosthome.FindLeaks(root, false)
	if len(leaks) != 1 || leaks[0].Path != path || leaks[0].Fingerprint == "" {
		t.Fatalf("leaks = %+v, want one non-dummy leak at %s", leaks, path)
	}
	if strings.Contains(leaks[0].String(), "not-a-real-token") {
		t.Fatalf("leak report prints credential content: %s", leaks[0])
	}
}

func TestFindLeaksAllowsLiveDummyButFlagsSurvivor(t *testing.T) {
	root := t.TempDir()
	path := writeAuth(t, root, hosthome.DummyAuth)
	if leaks := hosthome.FindLeaks(root, false); len(leaks) != 0 {
		t.Fatalf("dummy credential during a test is allowed, got %+v", leaks)
	}
	leaks := hosthome.FindLeaks(root, true)
	if len(leaks) != 1 || leaks[0].Path != path {
		t.Fatalf("leaks = %+v, want the surviving dummy credential flagged", leaks)
	}
}

func TestFindLeaksIgnoresUnrelatedAuthFiles(t *testing.T) {
	root := t.TempDir()
	other := filepath.Join(root, "opencode", "auth.json")
	if err := os.MkdirAll(filepath.Dir(other), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(other, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if leaks := hosthome.FindLeaks(root, true); len(leaks) != 0 {
		t.Fatalf("leaks = %+v, want none for non-run-scoped auth files", leaks)
	}
}

func TestMainIsolatesHostLogin(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(home, ".codex", "auth.json"))
	if err != nil || string(data) != hosthome.DummyAuth {
		t.Fatalf("HOME login is not the dummy login (err=%v)", err)
	}
	if os.Getenv("CODEX_HOME") != filepath.Join(home, ".codex") || !strings.HasPrefix(os.TempDir(), filepath.Dir(home)) {
		t.Fatalf("CODEX_HOME/TMPDIR are not isolated under %s", filepath.Dir(home))
	}
}
