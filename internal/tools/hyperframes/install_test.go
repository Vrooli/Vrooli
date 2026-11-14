package hyperframes

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit/hostreqkittest"
)

func sourceFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, name := range []string{"package.json", "package-lock.json"} {
		b, _ := closure.ReadFile(name)
		if err := os.WriteFile(filepath.Join(dir, name), b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func fakeInstall(_ context.Context, dir string) (string, error) {
	entry := filepath.Join(dir, "node_modules", "hyperframes", "bin")
	if err := os.MkdirAll(entry, 0700); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(entry, "hyperframes.mjs"), []byte("fixture"), 0600); err != nil {
		return "", err
	}
	return "fixture", os.WriteFile(filepath.Join(dir, "node_modules", "hyperframes", "package.json"), []byte(`{"version":"`+Version+`"}`), 0600)
}

func TestProvisionPublishesFrozenCandidateAndReusesOnlyVerifiedTree(t *testing.T) {
	_, restore := hostreqkittest.StubInvokingUser(t)
	defer restore()
	source := sourceFixture(t)
	calls := 0
	install := func(ctx context.Context, dir string) (string, error) { calls++; return fakeInstall(ctx, dir) }
	if _, err := Provision(context.Background(), source, install); err != nil {
		t.Fatal(err)
	}
	dir, _, err := SelectedInstallation()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = Provision(context.Background(), source, install); err != nil || calls != 1 {
		t.Fatalf("verified reuse failed calls=%d %v", calls, err)
	}
	if err = os.WriteFile(filepath.Join(dir, "node_modules", "hyperframes", "bin", "hyperframes.mjs"), []byte("corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = Provision(context.Background(), source, install); err == nil || calls != 1 {
		t.Fatal("corrupt existing installation was replaced or reused")
	}
}

func TestFailedProvisionLeavesNoSelectedInstallation(t *testing.T) {
	_, restore := hostreqkittest.StubInvokingUser(t)
	defer restore()
	_, err := Provision(context.Background(), sourceFixture(t), func(context.Context, string) (string, error) {
		return "install failed", errors.New("network unavailable")
	})
	if err == nil {
		t.Fatal("expected failure")
	}
	if _, _, err = SelectedInstallation(); err == nil {
		t.Fatal("failed candidate selected")
	}
	root, _ := InstallRoot()
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("failed candidate not removed: %v", entries)
	}
}

func TestProvisionRefusesChangedLockBeforeInstaller(t *testing.T) {
	source := sourceFixture(t)
	_ = os.WriteFile(filepath.Join(source, "package-lock.json"), []byte(`{}`), 0600)
	called := false
	_, err := Provision(context.Background(), source, func(context.Context, string) (string, error) { called = true; return "", nil })
	if err == nil || called {
		t.Fatal("unrecognized closure reached installer")
	}
}
