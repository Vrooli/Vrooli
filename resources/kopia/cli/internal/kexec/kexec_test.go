package kexec

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestBinaryRunnerMaterializesAuthorityKeyAsPrivateTemporaryFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test helper is a POSIX executable")
	}
	dir := t.TempDir()
	argsFile := filepath.Join(dir, "args")
	keyPathFile := filepath.Join(dir, "keypath")
	keyBody := "#!/bin/sh\nprintf '%s' \"$*\" > '" + argsFile + "'\nprintf '%s' \"$VROOLI_KOPIA_SFTP_PRIVATE_KEY\" > '" + filepath.Join(dir, "leaked-env") + "'\nwhile [ $# -gt 0 ]; do if [ \"$1\" = --keyfile ]; then shift; keyfile=$1; break; fi; shift; done\nprintf '%s' \"$keyfile\" > '" + keyPathFile + "'\nstat -c '%a' \"$keyfile\" > '" + filepath.Join(dir, "mode") + "'\ncat \"$keyfile\" > '" + filepath.Join(dir, "key-content") + "'\n"
	bin := filepath.Join(dir, "kopia")
	if err := os.WriteFile(bin, []byte(keyBody), 0o700); err != nil {
		t.Fatal(err)
	}
	key := "private-key-material\n"
	_, err := NewBinaryRunner(bin).Run(context.Background(), Call{
		Args: []string{"repository", "create", "sftp", "--keyfile", SFTPKeyFilePlaceholder},
		Env:  map[string]string{SFTPPrivateKeyEnv: key},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "key-content")); string(got) != key {
		t.Fatalf("key file content = %q", got)
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "mode")); strings.TrimSpace(string(got)) != "600" {
		t.Fatalf("key file mode = %q, want 600", got)
	}
	path, _ := os.ReadFile(keyPathFile)
	if _, err := os.Stat(string(path)); !os.IsNotExist(err) {
		t.Fatalf("temporary key file remains at %q (err=%v)", path, err)
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "leaked-env")); len(got) != 0 {
		t.Fatal("private key was passed into child environment")
	}
}
