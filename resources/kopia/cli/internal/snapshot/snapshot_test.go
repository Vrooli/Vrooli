package snapshot_test

import (
	"context"
	"path/filepath"
	"resource-kopia/cli/internal/invariant"
	"resource-kopia/cli/internal/registry"
	"resource-kopia/cli/internal/repoctx"
	"resource-kopia/cli/internal/snapshot"
	"resource-kopia/cli/internal/vault"
	"testing"

	kexecmocks "resource-kopia/cli/internal/kexec/mocks"

	vaultmocks "resource-kopia/cli/internal/vault/mocks"
)

const cfg = "/cfg/nightly/repository.config"

func newSvc(t *testing.T) (snapshot.Service, *kexecmocks.FakeRunner) {
	t.Helper()
	run := &kexecmocks.FakeRunner{}
	v := vaultmocks.NewFakeVault()
	v.Seed(vault.PassphrasePath("nightly"), "passphrase", "passphrase-value-abcdefghijklmnop")
	reg := registry.New(filepath.Join(t.TempDir(), "registry.json"))
	if err := reg.Upsert(registry.Entry{Name: "nightly", Backend: registry.BackendFilesystem, ConfigFile: cfg, Path: "/p"}); err != nil {
		t.Fatal(err)
	}
	return snapshot.Service{Runner: run, Resolver: repoctx.Resolver{Registry: reg, Vault: v}}, run
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

func TestSnapshotArgvTranslation(t *testing.T) {
	cases := []struct {
		name string
		run  func(s snapshot.Service) error
		want []string
	}{
		{
			name: "create with json",
			run: func(s snapshot.Service) error {
				return s.Create(context.Background(), []string{"--repo", "nightly", "--path", "/data/pg", "--json"})
			},
			want: []string{"--config-file", cfg, "snapshot", "create", "/data/pg", "--json"},
		},
		{
			name: "list filtered",
			run: func(s snapshot.Service) error {
				return s.List(context.Background(), []string{"--repo", "nightly", "--path", "/data/pg", "--json"})
			},
			want: []string{"--config-file", cfg, "snapshot", "list", "/data/pg", "--json"},
		},
		{
			name: "list unfiltered",
			run:  func(s snapshot.Service) error { return s.List(context.Background(), []string{"--repo", "nightly"}) },
			want: []string{"--config-file", cfg, "snapshot", "list"},
		},
		{
			name: "restore",
			run: func(s snapshot.Service) error {
				return s.Restore(context.Background(), []string{"--repo", "nightly", "--snapshot", "abc123", "--target", "/restore/pg"})
			},
			want: []string{"--config-file", cfg, "snapshot", "restore", "abc123", "/restore/pg"},
		},
		{
			name: "verify single with percent",
			run: func(s snapshot.Service) error {
				return s.Verify(context.Background(), []string{"--repo", "nightly", "--snapshot", "abc123", "--verify-files-percent", "100"})
			},
			want: []string{"--config-file", cfg, "snapshot", "verify", "--verify-files-percent", "100", "abc123"},
		},
		{
			name: "verify repo-wide",
			run:  func(s snapshot.Service) error { return s.Verify(context.Background(), []string{"--repo", "nightly"}) },
			want: []string{"--config-file", cfg, "snapshot", "verify"},
		},
		{
			name: "delete forces confirmation",
			run: func(s snapshot.Service) error {
				return s.Delete(context.Background(), []string{"--repo", "nightly", "--snapshot", "abc123"})
			},
			want: []string{"--config-file", cfg, "snapshot", "delete", "abc123", "--delete"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, run := newSvc(t)
			if err := tc.run(s); err != nil {
				t.Fatalf("handler error = %v", err)
			}
			call, _ := run.LastCall()
			if !eq(call.Args, tc.want) {
				t.Fatalf("argv =\n  %v\nwant\n  %v", call.Args, tc.want)
			}
			if call.Env[repoctx.EnvPassword] == "" {
				t.Fatal("passphrase must be injected via env")
			}
			if tok, found := invariant.FindEncryptionFlag(run.AllArgs()); found {
				t.Fatalf("encryption flag leaked: %q", tok)
			}
		})
	}
}

func TestSnapshotRequiresFlags(t *testing.T) {
	s, _ := newSvc(t)
	if err := s.Create(context.Background(), []string{"--repo", "nightly"}); err == nil {
		t.Fatal("create without --path should fail")
	}
	if err := s.Restore(context.Background(), []string{"--repo", "nightly", "--snapshot", "x"}); err == nil {
		t.Fatal("restore without --target should fail")
	}
	if err := s.Create(context.Background(), []string{"--path", "/x"}); err == nil {
		t.Fatal("create without --repo should fail")
	}
}
