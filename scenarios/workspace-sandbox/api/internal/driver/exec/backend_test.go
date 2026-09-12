//go:build linux

package exec

import (
	"strings"
	"testing"

	"workspace-sandbox/internal/driver"
	"workspace-sandbox/internal/testutil/mocks/procmocks"
)

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
