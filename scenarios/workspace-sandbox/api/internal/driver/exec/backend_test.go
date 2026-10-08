//go:build linux

package exec

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"workspace-sandbox/internal/driver"
	"workspace-sandbox/internal/testutil/mocks/procmocks"
	"workspace-sandbox/internal/types"
)

func TestPolicyFilesRefuseUnsafeLaunches(t *testing.T) {
	for _, name := range []string{"valid", "changed bytes", "missing identity", "symlink", "hardlink", "tracking", "shared pid", "writable alias", "workspace target", "masked target", "overlap"} {
		t.Run(name, func(t *testing.T) {
			sb := newSandboxFor(t)
			source := filepath.Join(t.TempDir(), "requirements.toml")
			if err := os.WriteFile(source, []byte("policy"), 0600); err != nil {
				t.Fatal(err)
			}
			cfg := DefaultBwrapConfig()
			cfg.PolicyFiles = []types.PolicyFile{{Source: source, Target: "/etc/fixture/requirements.toml", SHA256: fmt.Sprintf("%x", sha256.Sum256([]byte("policy")))}}
			level := driver.ContainmentRequired
			switch name {
			case "changed bytes":
				if err := os.WriteFile(source, []byte("changed"), 0600); err != nil {
					t.Fatal(err)
				}
			case "missing identity":
				cfg.PolicyFiles[0].SHA256 = ""
			case "symlink":
				link := source + ".link"
				if err := os.Symlink(source, link); err != nil {
					t.Fatal(err)
				}
				cfg.PolicyFiles[0].Source = link
			case "hardlink":
				if err := os.Link(source, source+".link"); err != nil {
					t.Fatal(err)
				}
			case "tracking":
				level = driver.ContainmentNone
			case "shared pid":
				cfg.SharePID = true
			case "writable alias":
				cfg.ReadWriteBinds[filepath.Dir(source)] = "/alias"
			case "workspace target":
				cfg.PolicyFiles[0].Target = "/workspace/policy"
			case "masked target":
				cfg.MaskPaths = []string{"/etc/fixture"}
			case "overlap":
				cfg.PolicyFiles = append(cfg.PolicyFiles, cfg.PolicyFiles[0])
			}
			starter := procmocks.NewFakeStarter()
			starter.SetLookPath("bwrap", "/usr/bin/bwrap")
			_, _, err := buildStartOpts(starter, sb, level, cfg, "/bin/true")
			if (err == nil) != (name == "valid") {
				t.Fatalf("%s: error = %v", name, err)
			}
		})
	}
}

func TestWorkspaceWritePolicyRefusesUnsafeLaunches(t *testing.T) {
	for _, paths := range [][]string{{"."}, {"../outside"}, {"/tmp"}, {"src/**"}, {"src/../controls"}, {"src", "src/file"}, {"missing"}, {"link"}, {"link/file"}} {
		t.Run(strings.Join(paths, ","), func(t *testing.T) {
			sb := newSandboxFor(t)
			if err := os.MkdirAll(filepath.Join(sb.MergedDir, "src"), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink("src", filepath.Join(sb.MergedDir, "link")); err != nil {
				t.Fatal(err)
			}
			sb.Behavior.WritePolicy = &types.WorkspaceWritePolicy{Paths: paths}
			starter := procmocks.NewFakeStarter()
			starter.SetLookPath("bwrap", "/usr/bin/bwrap")
			if _, _, err := buildStartOpts(starter, sb, driver.ContainmentRequired, DefaultBwrapConfig(), "/bin/true"); err == nil {
				t.Fatal("unsafe grant must refuse launch")
			}
		})
	}
	sb := newSandboxFor(t)
	sb.Behavior.WritePolicy = &types.WorkspaceWritePolicy{}
	starter := procmocks.NewFakeStarter()
	starter.SetLookPath("bwrap", "/usr/bin/bwrap")
	for _, level := range []driver.ContainmentLevel{driver.ContainmentNone, driver.ContainmentPreferred} {
		if _, _, err := buildStartOpts(starter, sb, level, DefaultBwrapConfig(), "/bin/true"); err == nil {
			t.Fatal("write policy must never use a weaker containment level")
		}
	}
	cfg := DefaultBwrapConfig()
	cfg.ReadWriteBinds[sb.MergedDir] = "/escape"
	if _, _, err := buildStartOpts(starter, sb, driver.ContainmentRequired, cfg, "/bin/true"); err == nil {
		t.Fatal("a writable profile alias must not bypass the policy")
	}
	cfg = DefaultBwrapConfig()
	cfg.SharePID = true
	if _, _, err := buildStartOpts(starter, sb, driver.ContainmentRequired, cfg, "/bin/true"); err == nil {
		t.Fatal("a shared host PID namespace must not expose alternate filesystem roots")
	}
}

func TestWorkspaceWritePolicyRefusesExistingHardlinkAlias(t *testing.T) {
	sb := newSandboxFor(t)
	if err := os.Mkdir(filepath.Join(sb.MergedDir, "src"), 0700); err != nil {
		t.Fatal(err)
	}
	control := filepath.Join(sb.MergedDir, "owner-control")
	if err := os.WriteFile(control, []byte("owner"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(control, filepath.Join(sb.MergedDir, "src", "alias")); err != nil {
		t.Fatal(err)
	}
	sb.Behavior.WritePolicy = &types.WorkspaceWritePolicy{Paths: []string{"src"}}
	starter := procmocks.NewFakeStarter()
	starter.SetLookPath("bwrap", "/usr/bin/bwrap")
	if _, _, err := buildStartOpts(starter, sb, driver.ContainmentRequired, DefaultBwrapConfig(), "/bin/true"); err == nil || !strings.Contains(err.Error(), "hardlink") {
		t.Fatalf("a writable inode alias of an owner file must refuse launch: %v", err)
	}
}

// TestBuildStartOpts_LevelDispatch pins the ContainmentLevel -> backend
// dispatch on Linux, where the platform containment backend is bwrap:
//
//   - ContainmentNone always runs direct in s.MergedDir.
//   - ContainmentPreferred requires the backend and fails closed when it is
//     unavailable.
//   - ContainmentRequired uses the backend when available, else hard-errors.
//
// bwrap availability is simulated purely via the FakeStarter LookPath
// table, so the test is deterministic regardless of what is installed.
func TestBuildStartOpts_LevelDispatch(t *testing.T) {
	sb := newSandboxFor(t)
	const cmd = "/bin/echo"

	cases := []struct {
		name        string
		level       driver.ContainmentLevel
		bwrapOnPath bool
		wantErr     bool
		wantDirect  bool   // true = direct exec (Path==cmd, Dir==MergedDir); false = bwrap backend
		wantBackend string // effective backend id returned alongside the opts
	}{
		{"none/no-bwrap", driver.ContainmentNone, false, false, true, "none"},
		{"none/has-bwrap", driver.ContainmentNone, true, false, true, "none"},
		{"preferred/no-bwrap-errors", driver.ContainmentPreferred, false, true, false, ""},
		{"preferred/has-bwrap-uses-backend", driver.ContainmentPreferred, true, false, false, "bwrap"},
		{"required/no-backend-errors", driver.ContainmentRequired, false, true, false, ""},
		{"required/has-bwrap-uses-backend", driver.ContainmentRequired, true, false, false, "bwrap"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			starter := procmocks.NewFakeStarter()
			if tc.bwrapOnPath {
				starter.SetLookPath("bwrap", "/usr/bin/bwrap")
			}
			opts, backendID, err := buildStartOpts(starter, sb, tc.level, DefaultBwrapConfig(), cmd, "hi")
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for required-without-backend, got opts %+v", opts)
				}
				if !strings.Contains(err.Error(), "bwrap") {
					t.Errorf("error should name the missing backend, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("buildStartOpts: %v", err)
			}
			if backendID != tc.wantBackend {
				t.Errorf("effective backend: got %q, want %q", backendID, tc.wantBackend)
			}
			if tc.wantDirect {
				if opts.Path != cmd {
					t.Errorf("direct path: got %q, want %q", opts.Path, cmd)
				}
				if opts.Dir != sb.MergedDir {
					t.Errorf("direct dir: got %q, want %q", opts.Dir, sb.MergedDir)
				}
				return
			}
			if opts.Path != "/usr/bin/bwrap" {
				t.Errorf("backend path: got %q, want /usr/bin/bwrap", opts.Path)
			}
			if opts.Dir != "" {
				t.Errorf("bwrap backend sets chdir via argv, not StartOpts.Dir; got Dir=%q", opts.Dir)
			}
		})
	}
}
