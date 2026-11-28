package agentharness

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestResolveNativeFilePathFollowsASessionLinkToTheSharedFile(t *testing.T) {
	dir := t.TempDir()
	shared := filepath.Join(dir, "shared", "config.toml")
	session := filepath.Join(dir, "session", "config.toml")
	for _, d := range []string{filepath.Dir(shared), filepath.Dir(session)} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(shared, []byte("model = 'x'\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(shared, session); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	want, _ := filepath.EvalSymlinks(shared)
	for _, tc := range []struct{ in, want string }{
		{session, want},
		{shared, shared},
		{filepath.Join(dir, "missing.toml"), filepath.Join(dir, "missing.toml")},
	} {
		got, err := ResolveNativeFilePath(tc.in)
		if err != nil || got != tc.want {
			t.Errorf("ResolveNativeFilePath(%s) = %q, %v; want %q", tc.in, got, err, tc.want)
		}
	}
}

func TestBackupNativeFileKeepsTheFirstImageAndTheLatestPrivately(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if got, err := BackupNativeFile(PermissionFileSnapshot{Path: path}, "permissions"); err != nil || got != "" {
		t.Fatalf("missing file backup = %q, %v; want none", got, err)
	}
	for _, content := range []string{"# pristine\n", "second\n"} {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		before, err := ReadPermissionFile(path)
		if err != nil {
			t.Fatal(err)
		}
		bak, err := BackupNativeFile(before, "permissions")
		if err != nil {
			t.Fatal(err)
		}
		if bak != path+".vrooli-permissions.bak" {
			t.Fatalf("backup path = %s", bak)
		}
		if got, _ := os.ReadFile(bak); string(got) != content {
			t.Fatalf(".bak = %q, want latest pre-image %q", got, content)
		}
	}
	orig := path + ".vrooli-permissions.orig"
	if got, _ := os.ReadFile(orig); string(got) != "# pristine\n" {
		t.Fatalf(".orig = %q, want the first pre-image", got)
	}
	if runtime.GOOS != "windows" {
		for _, p := range []string{orig, path + ".vrooli-permissions.bak"} {
			if info, err := os.Stat(p); err != nil || info.Mode().Perm() != 0o600 {
				t.Fatalf("%s is not private: %v %v", p, info, err)
			}
		}
	}
}
