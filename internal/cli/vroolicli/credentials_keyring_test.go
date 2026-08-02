package vroolicli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestKeyringInspectJSONContract pins the JSON envelope `credentials keyring
// inspect --format json` emits.
//
// It exists because vrooli-autoheal parses this output to decide whether a
// repair has already landed, and the two live in different modules with no
// compiler between them. When this command grew a second top-level field the
// shape changed from a bare array to an envelope, autoheal's parser kept
// reading the array, and its unit test kept passing because the test mocked the
// assumed shape rather than the emitted one. Only a live run caught it. This
// test fails on the producing side instead.
func TestKeyringInspectJSONContract(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "login.keyring")
	contents := "[keyring]\ndisplay-name=Login\nctime=0\nmtime=1\nlock-on-idle=false\nlock-after=false\n" +
		"\n[1]\nitem-type=0\ndisplay-name=Fine\nsecret=single-line\nmtime=1\nctime=1\n"
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	var stdout bytes.Buffer
	ctx := &CommandContext{Stdout: &stdout, Stderr: &bytes.Buffer{}}
	if err := credentialsKeyring(ctx, []string{"inspect", "--path", path, "--format", "json"}); err != nil {
		t.Fatalf("credentials keyring inspect: %v", err)
	}

	var payload struct {
		Reports []struct {
			Path     string `json:"path"`
			Loadable bool   `json:"loadable"`
		} `json:"reports"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("output is not the documented envelope: %v (got %s)", err, stdout.String())
	}
	if len(payload.Reports) != 1 {
		t.Fatalf("reports has %d entries, want 1: %s", len(payload.Reports), stdout.String())
	}
	if payload.Reports[0].Path != path {
		t.Errorf("path = %q, want %q", payload.Reports[0].Path, path)
	}
	if !payload.Reports[0].Loadable {
		t.Errorf("a single-line keyring must report loadable: %s", stdout.String())
	}
}

// TestKeyringInspectNeverWrites keeps the read-only half read-only. An operator
// running inspect is often deciding whether to trust this command with a write.
func TestKeyringInspectNeverWrites(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "login.keyring")
	contents := "[keyring]\ndisplay-name=Login\nctime=0\nmtime=1\nlock-on-idle=false\nlock-after=false\n" +
		"\n[9]\nitem-type=0\ndisplay-name=Vrooli managed resource\nsecret=line-one\nline-two\nmtime=1\nctime=1\n" +
		"\n[9:attribute0]\nname=service\ntype=string\nvalue=vrooli.credentials.v1\n"
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	var stdout bytes.Buffer
	ctx := &CommandContext{Stdout: &stdout, Stderr: &bytes.Buffer{}}
	if err := credentialsKeyring(ctx, []string{"inspect", "--path", path}); err != nil {
		t.Fatalf("credentials keyring inspect: %v", err)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if string(after) != contents {
		t.Fatal("inspect modified the keyring file")
	}
	if !strings.Contains(stdout.String(), "NOT loadable") {
		t.Errorf("inspect must report the defect, got: %s", stdout.String())
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("inspect created %d extra files; it must take no backup and write nothing", len(entries)-1)
	}
}
