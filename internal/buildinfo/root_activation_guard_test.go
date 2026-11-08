package buildinfo

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/repo-contract-go/repocontracttest"
	"github.com/vrooli/vrooli/internal/coreactivation"
)

func TestRootInstallGuardRejectsUnknownOperationStateAndPreservesScratchInstalls(t *testing.T) {
	home := useTestRebuildHome(t)
	bin, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyBin)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(bin, 0o700); err != nil {
		t.Fatal(err)
	}
	s, err := coreactivation.NewStore(home)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(s.Directory(), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s.Directory(), "current.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := GuardRootActivation(context.Background(), filepath.Join(bin, "vrooli")); err == nil {
		t.Fatal("canonical root installation ignored unknown operation state")
	}
	scratch := filepath.Join(t.TempDir(), "vrooli")
	if err := GuardRootActivation(context.Background(), scratch); err != nil {
		t.Fatalf("scratch binary confused with canonical root: %v", err)
	}
	alias := filepath.Join(t.TempDir(), "bin-alias")
	if err := os.Symlink(bin, alias); err != nil {
		t.Fatal(err)
	}
	if err := GuardRootActivation(context.Background(), filepath.Join(alias, "vrooli")); err == nil {
		t.Fatal("canonical parent alias bypassed activation guard")
	}
}

func TestSelfRebuildPreservesServingImageWhenFallbackRetentionFails(t *testing.T) {
	useTestRebuildHome(t)
	oldFingerprint, oldCommit, oldTime := Fingerprint, GitCommit, BuildTime
	oldBuild, oldExec, oldExecutable := goBuildFn, execFn, executablePathFn
	t.Cleanup(func() {
		Fingerprint, GitCommit, BuildTime = oldFingerprint, oldCommit, oldTime
		goBuildFn, execFn, executablePathFn = oldBuild, oldExec, oldExecutable
	})
	root := t.TempDir()
	repocontracttest.WriteRepoContract(t, root, "scenarios")
	writeStateContractFixture(t, root)
	writeTestFile(t, root, "go.mod", "module example.com/test\n\ngo 1.21\n")
	writeTestFile(t, root, "cmd/vrooli/main.go", "package main\n")
	writeTestFile(t, root, "internal/buildinfo/buildinfo.go", "package buildinfo\n")
	t.Setenv(SourceRootEnvVar, root)
	t.Setenv(FingerprintPathsEnvVar, "cmd/vrooli,internal")
	t.Setenv(BuildTargetEnvVar, "./cmd/vrooli")
	executable := filepath.Join(t.TempDir(), "vrooli")
	if err := os.WriteFile(executable, []byte("serving-prior"), 0o755); err != nil {
		t.Fatal(err)
	}
	executablePathFn = func() (string, error) { return executable, nil }
	Fingerprint = "stale"
	GitCommit = "fixture"
	blocked := filepath.Join(filepath.Dir(filepath.Dir(executable)), "libexec")
	if err := os.WriteFile(blocked, []byte("do not replace"), 0o600); err != nil {
		t.Fatal(err)
	}
	var staged string
	goBuildFn = func(_ string, args []string) error {
		for i := 0; i+1 < len(args); i++ {
			if args[i] == "-o" {
				staged = args[i+1]
				return os.WriteFile(staged, []byte("new-candidate"), 0o755)
			}
		}
		t.Fatal("missing build output")
		return nil
	}
	executed := false
	execFn = func(string, []string, []string) error { executed = true; return nil }
	err := RebuildAndReexec([]string{"version"})
	if err == nil || !strings.Contains(err.Error(), "retain root fallback") {
		t.Fatalf("fallback failure did not stop publication: %v", err)
	}
	if executed {
		t.Fatal("self-rebuild executed candidate after failed retention")
	}
	if data, err := os.ReadFile(executable); err != nil || string(data) != "serving-prior" {
		t.Fatal("serving image changed despite failed fallback retention")
	}
	if staged == "" {
		t.Fatal("failure occurred before candidate staging")
	}
	if _, err := os.Lstat(staged); !os.IsNotExist(err) {
		t.Fatal("failed retained candidate was not retired")
	}
}
