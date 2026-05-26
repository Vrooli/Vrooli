package maintenance_test

import (
	"context"
	"path/filepath"
	"resource-kopia/cli/internal/maintenance"
	"resource-kopia/cli/internal/registry"
	"resource-kopia/cli/internal/repoctx"
	"resource-kopia/cli/internal/vault"
	"testing"

	kexecmocks "resource-kopia/cli/internal/kexec/mocks"

	vaultmocks "resource-kopia/cli/internal/vault/mocks"
)

const cfg = "/cfg/offsite/repository.config"

func newSvc(t *testing.T) (maintenance.Service, *kexecmocks.FakeRunner) {
	t.Helper()
	run := &kexecmocks.FakeRunner{}
	v := vaultmocks.NewFakeVault()
	v.Seed(vault.PassphrasePath("offsite"), "passphrase", "passphrase-value-abcdefghijklmnop")
	reg := registry.New(filepath.Join(t.TempDir(), "registry.json"))
	if err := reg.Upsert(registry.Entry{Name: "offsite", Backend: registry.BackendFilesystem, ConfigFile: cfg, Path: "/p"}); err != nil {
		t.Fatal(err)
	}
	return maintenance.Service{Runner: run, Resolver: repoctx.Resolver{Registry: reg, Vault: v}}, run
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

func TestMaintenanceRunArgv(t *testing.T) {
	s, run := newSvc(t)
	if err := s.Run(context.Background(), []string{"--repo", "offsite"}); err != nil {
		t.Fatalf("Run error = %v", err)
	}
	call, _ := run.LastCall()
	if !eq(call.Args, []string{"--config-file", cfg, "maintenance", "run"}) {
		t.Fatalf("run argv = %v", call.Args)
	}
	if call.Env[repoctx.EnvPassword] == "" {
		t.Fatal("maintenance must inject passphrase env")
	}

	if err := s.Run(context.Background(), []string{"--repo", "offsite", "--full"}); err != nil {
		t.Fatalf("Run --full error = %v", err)
	}
	call, _ = run.LastCall()
	if !eq(call.Args, []string{"--config-file", cfg, "maintenance", "run", "--full"}) {
		t.Fatalf("run --full argv = %v", call.Args)
	}
}

func TestMaintenanceSetArgv(t *testing.T) {
	s, run := newSvc(t)
	err := s.Set(context.Background(), []string{"--repo", "offsite", "--enable-full", "true", "--enable-quick", "false", "--full-interval", "24h"})
	if err != nil {
		t.Fatalf("Set error = %v", err)
	}
	call, _ := run.LastCall()
	want := []string{"--config-file", cfg, "maintenance", "set", "--enable-full", "true", "--enable-quick", "false", "--full-interval", "24h"}
	if !eq(call.Args, want) {
		t.Fatalf("set argv =\n  %v\nwant\n  %v", call.Args, want)
	}
}
