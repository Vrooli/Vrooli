package policy_test

import (
	"context"
	"path/filepath"
	"testing"

	"resource-kopia/cli/internal/invariant"
	"resource-kopia/cli/internal/policy"
	"resource-kopia/cli/internal/registry"
	"resource-kopia/cli/internal/repoctx"

	kexecmocks "resource-kopia/cli/internal/kexec/mocks"

	vaultmocks "resource-kopia/cli/internal/vault/mocks"
)

const cfg = "/cfg/offsite/repository.config"

func newSvc(t *testing.T) (policy.Service, *kexecmocks.FakeRunner) {
	t.Helper()
	run := &kexecmocks.FakeRunner{}
	v := vaultmocks.NewFakeVault()
	v.SeedPassphrase("offsite", "passphrase-value-abcdefghijklmnop")
	reg := registry.New(filepath.Join(t.TempDir(), "registry.json"))
	if err := reg.Upsert(registry.Entry{Name: "offsite", Backend: registry.BackendFilesystem, ConfigFile: cfg, Path: "/p"}); err != nil {
		t.Fatal(err)
	}
	return policy.Service{Runner: run, Resolver: repoctx.Resolver{Registry: reg, Credentials: v, Vault: v}}, run
}

func eq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestPolicySetAllGFSFlags(t *testing.T) {
	s, run := newSvc(t)
	err := s.Set(context.Background(), []string{
		"--repo", "offsite", "--path", "/data/pg",
		"--keep-latest", "3", "--keep-hourly", "24", "--keep-daily", "7",
		"--keep-weekly", "4", "--keep-monthly", "12", "--keep-annual", "2",
		"--compression", "zstd", "--snapshot-interval", "1h",
	})
	if err != nil {
		t.Fatalf("Set error = %v", err)
	}
	call, _ := run.LastCall()
	want := []string{
		"--config-file", cfg, "policy", "set", "/data/pg",
		"--keep-latest", "3", "--keep-hourly", "24", "--keep-daily", "7",
		"--keep-weekly", "4", "--keep-monthly", "12", "--keep-annual", "2",
		"--compression", "zstd", "--snapshot-interval", "1h",
	}
	if !eq(call.Args, want) {
		t.Fatalf("argv =\n  %v\nwant\n  %v", call.Args, want)
	}
}

func TestPolicySetOmitsUnsetKeeps(t *testing.T) {
	s, run := newSvc(t)
	if err := s.Set(context.Background(), []string{"--repo", "offsite", "--path", "/data/pg", "--keep-daily", "7"}); err != nil {
		t.Fatalf("Set error = %v", err)
	}
	call, _ := run.LastCall()
	want := []string{"--config-file", cfg, "policy", "set", "/data/pg", "--keep-daily", "7"}
	if !eq(call.Args, want) {
		t.Fatalf("argv = %v want %v", call.Args, want)
	}
}

func TestPolicyShowAndList(t *testing.T) {
	s, run := newSvc(t)
	if err := s.Show(context.Background(), []string{"--repo", "offsite", "--path", "/data/pg", "--json"}); err != nil {
		t.Fatalf("Show error = %v", err)
	}
	call, _ := run.LastCall()
	if !eq(call.Args, []string{"--config-file", cfg, "policy", "show", "/data/pg", "--json"}) {
		t.Fatalf("show argv = %v", call.Args)
	}

	if err := s.Show(context.Background(), []string{"--repo", "offsite", "--global", "--json"}); err != nil {
		t.Fatalf("Show global error = %v", err)
	}
	call, _ = run.LastCall()
	if !eq(call.Args, []string{"--config-file", cfg, "policy", "show", "--global", "--json"}) {
		t.Fatalf("show global argv = %v", call.Args)
	}

	if err := s.List(context.Background(), []string{"--repo", "offsite", "--json"}); err != nil {
		t.Fatalf("List error = %v", err)
	}
	call, _ = run.LastCall()
	if !eq(call.Args, []string{"--config-file", cfg, "policy", "list", "--json"}) {
		t.Fatalf("list argv = %v", call.Args)
	}
}

func TestPolicyEncryptionInvariant(t *testing.T) {
	s, run := newSvc(t)
	_ = s.Set(context.Background(), []string{"--repo", "offsite", "--path", "/p", "--keep-daily", "7", "--compression", "zstd"})
	if tok, found := invariant.FindEncryptionFlag(run.AllArgs()); found {
		t.Fatalf("encryption flag leaked: %q", tok)
	}
}

func TestPolicySetRequiresPath(t *testing.T) {
	s, _ := newSvc(t)
	if err := s.Set(context.Background(), []string{"--repo", "offsite"}); err == nil {
		t.Fatal("expected --path required error")
	}
}
