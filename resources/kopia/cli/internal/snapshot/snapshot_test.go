package snapshot_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/resources/kopia/cli/internal/invariant"
	"github.com/vrooli/vrooli/resources/kopia/cli/internal/kexec"
	"github.com/vrooli/vrooli/resources/kopia/cli/internal/registry"
	"github.com/vrooli/vrooli/resources/kopia/cli/internal/repoctx"
	"github.com/vrooli/vrooli/resources/kopia/cli/internal/snapshot"

	kexecmocks "github.com/vrooli/vrooli/resources/kopia/cli/internal/kexec/mocks"

	credentialmocks "github.com/vrooli/vrooli/resources/kopia/cli/internal/credentials/mocks"
)

const cfg = "/cfg/nightly/repository.config"

func newSvc(t *testing.T) (snapshot.Service, *kexecmocks.FakeRunner) {
	t.Helper()
	run := &kexecmocks.FakeRunner{}
	v := credentialmocks.NewFakeStore()
	v.SeedPassphrase("nightly", "passphrase-value-abcdefghijklmnop")
	reg := registry.New(filepath.Join(t.TempDir(), "registry.json"))
	if err := reg.Upsert(registry.Entry{Name: "nightly", Backend: registry.BackendFilesystem, ConfigFile: cfg, Path: "/p"}); err != nil {
		t.Fatal(err)
	}
	return snapshot.Service{Runner: run, Resolver: repoctx.Resolver{Registry: reg, Credentials: v}}, run
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
		{
			name: "create with description override-source and repeated tags",
			run: func(s snapshot.Service) error {
				return s.Create(context.Background(), []string{
					"--repo", "nightly", "--path", "/data/pg",
					"--description", "DBM target acme/db run r1",
					"--override-source", "dbm://acme/db",
					"--tags", "dbm:true", "--tags", "dbm.run_id:r1",
					"--json",
				})
			},
			want: []string{
				"--config-file", cfg, "snapshot", "create", "/data/pg",
				"--description", "DBM target acme/db run r1",
				"--override-source", "dbm://acme/db",
				"--tags", "dbm:true", "--tags", "dbm.run_id:r1",
				"--json",
			},
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

func TestSnapshotCreateRejectsMalformedTag(t *testing.T) {
	s, run := newSvc(t)
	err := s.Create(context.Background(), []string{"--repo", "nightly", "--path", "/data/pg", "--tags", "nocolon"})
	if err == nil {
		t.Fatal("expected error for tag without ':'")
	}
	if !strings.Contains(err.Error(), "key:value") {
		t.Fatalf("error should explain key:value form, got %v", err)
	}
	// No kopia process should have run for a rejected tag.
	if _, ok := run.LastCall(); ok {
		t.Fatal("kopia should not be invoked when a tag is malformed")
	}
}

func TestBrowseRestoresSnapshotToTempAndListsOneDirectoryLevel(t *testing.T) {
	s, run := newSvc(t)
	var restoredTo string
	run.Responder = func(c kexec.Call) ([]byte, error) {
		if !slices.Equal(c.Args[:4], []string{"--config-file", cfg, "snapshot", "restore"}) {
			t.Fatalf("unexpected argv prefix: %v", c.Args)
		}
		if c.Args[4] != "abc123" {
			t.Fatalf("snapshot arg = %q, want abc123", c.Args[4])
		}
		restoredTo = c.Args[5]
		mustWriteFile(t, filepath.Join(restoredTo, "root.txt"), "alpha\n")
		mustWriteFile(t, filepath.Join(restoredTo, "nested", "child.txt"), "beta\n")
		mustWriteFile(t, filepath.Join(restoredTo, "nested", "second.txt"), "gamma\n")
		return nil, nil
	}

	var out strings.Builder
	s.Out = &out
	if err := s.Browse(context.Background(), []string{"--repo", "nightly", "--snapshot", "abc123", "--path", "nested", "--json"}); err != nil {
		t.Fatalf("Browse: %v", err)
	}
	if restoredTo == "" {
		t.Fatal("snapshot was not restored to a temp dir")
	}
	if _, err := os.Stat(restoredTo); !os.IsNotExist(err) {
		t.Fatalf("browse temp dir should be cleaned up; stat err = %v", err)
	}
	var entries []struct {
		Path      string `json:"path"`
		SizeBytes int64  `json:"sizeBytes"`
		Type      string `json:"type"`
		IsDir     bool   `json:"isDir"`
	}
	if err := json.Unmarshal([]byte(out.String()), &entries); err != nil {
		t.Fatalf("parse browse json: %v\n%s", err, out.String())
	}
	if len(entries) != 2 {
		t.Fatalf("entries = %v, want 2 files", entries)
	}
	if entries[0].Path != "nested/child.txt" || entries[0].Type != "f" || entries[0].IsDir {
		t.Fatalf("entry[0] = %+v", entries[0])
	}
	if entries[1].Path != "nested/second.txt" || entries[1].SizeBytes != int64(len("gamma\n")) {
		t.Fatalf("entry[1] = %+v", entries[1])
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
	if err := s.Browse(context.Background(), []string{"--repo", "nightly", "--snapshot", "x"}); err == nil {
		t.Fatal("browse without --json should fail")
	}
	if err := s.Browse(context.Background(), []string{"--repo", "nightly", "--json"}); err == nil {
		t.Fatal("browse without --snapshot should fail")
	}
	if err := s.Browse(context.Background(), []string{"--repo", "nightly", "--snapshot", "x", "--path", "../escape", "--json"}); err == nil {
		t.Fatal("browse with escaping path should fail")
	}
}

func mustWriteFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o640); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
