package maintenance_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/vrooli/vrooli/resources/kopia/cli/internal/maintenance"
	"github.com/vrooli/vrooli/resources/kopia/cli/internal/registry"
	"github.com/vrooli/vrooli/resources/kopia/cli/internal/repoctx"

	kexecmocks "github.com/vrooli/vrooli/resources/kopia/cli/internal/kexec/mocks"

	credentialmocks "github.com/vrooli/vrooli/resources/kopia/cli/internal/credentials/mocks"
)

const cfg = "/cfg/offsite/repository.config"

func newSvc(t *testing.T) (maintenance.Service, *kexecmocks.FakeRunner) {
	t.Helper()
	run := &kexecmocks.FakeRunner{}
	v := credentialmocks.NewFakeStore()
	v.SeedPassphrase("offsite", "passphrase-value-abcdefghijklmnop")
	reg := registry.New(filepath.Join(t.TempDir(), "registry.json"))
	if err := reg.Upsert(registry.Entry{Name: "offsite", Backend: registry.BackendFilesystem, ConfigFile: cfg, Path: "/p"}); err != nil {
		t.Fatal(err)
	}
	return maintenance.Service{Runner: run, Resolver: repoctx.Resolver{Registry: reg, Credentials: v}}, run
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
