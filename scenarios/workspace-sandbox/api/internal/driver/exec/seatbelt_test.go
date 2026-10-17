package exec

import (
	"reflect"
	"strings"
	"testing"

	"workspace-sandbox/internal/types"
)

// seatbeltSandbox returns a sandbox with fixed overlay dirs so the profile
// and argv golden strings are deterministic (no t.TempDir randomness).
func seatbeltSandbox() *types.Sandbox {
	return &types.Sandbox{
		MergedDir: "/var/lib/workspace-sandbox/abc/merged",
		UpperDir:  "/var/lib/workspace-sandbox/abc/upper",
		WorkDir:   "/var/lib/workspace-sandbox/abc/work",
	}
}

// TestSeatbeltProfile_NetworkDenied pins the full profile string for the
// default (network-denied) case, including the /private symlink twins for the
// /var-rooted overlay dirs and the system temp roots.
func TestSeatbeltProfile_NetworkDenied(t *testing.T) {
	cfg := DefaultBwrapConfig() // AllowNetwork=false
	got := SeatbeltProfile(seatbeltSandbox(), cfg)

	want := `(version 1)
(allow default)
(deny file-write*)
(allow file-write*
    (subpath "/var/lib/workspace-sandbox/abc/merged")
    (subpath "/private/var/lib/workspace-sandbox/abc/merged")
    (subpath "/var/lib/workspace-sandbox/abc/upper")
    (subpath "/private/var/lib/workspace-sandbox/abc/upper")
    (subpath "/var/lib/workspace-sandbox/abc/work")
    (subpath "/private/var/lib/workspace-sandbox/abc/work")
    (subpath "/tmp")
    (subpath "/private/tmp")
    (subpath "/var/folders")
    (subpath "/private/var/folders")
    (literal "/dev/null")
    (regex #"^/dev/tty"))
(deny network*)
`
	if got != want {
		t.Errorf("profile mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

// TestSeatbeltProfile_NetworkAllowed asserts the deny network* line is absent
// when the profile permits network (localhost or full both map to
// AllowNetwork=true, matching the bwrap path's --unshare-net toggle).
func TestSeatbeltProfile_NetworkAllowed(t *testing.T) {
	cfg := DefaultBwrapConfig()
	cfg.AllowNetwork = true
	got := SeatbeltProfile(seatbeltSandbox(), cfg)

	if strings.Contains(got, "network*") {
		t.Errorf("expected no network deny rule when AllowNetwork=true; got:\n%s", got)
	}
	// Filesystem write-containment is still enforced regardless of network.
	if !strings.Contains(got, "(deny file-write*)") {
		t.Errorf("expected file-write containment even with network allowed; got:\n%s", got)
	}
}

// TestSeatbeltProfile_HomeOverlayAndEscaping covers the per-sandbox HOME
// overlay bind and SBPL string escaping of a path containing a double quote.
func TestSeatbeltProfile_HomeOverlayAndEscaping(t *testing.T) {
	s := seatbeltSandbox()
	s.HomeMergedDir = `/Users/dev/ws "quoted"/home`
	cfg := DefaultBwrapConfig()
	cfg.HostHome = "/Users/dev"

	got := SeatbeltProfile(s, cfg)

	// The HOME overlay merged dir is granted write access, with the quote
	// escaped so the s-expression stays well-formed.
	wantLine := `    (subpath "/Users/dev/ws \"quoted\"/home")`
	if !strings.Contains(got, wantLine) {
		t.Errorf("expected escaped home overlay subpath %q in profile:\n%s", wantLine, got)
	}
	// A /Users path has no /private twin.
	if strings.Contains(got, "/private/Users") {
		t.Errorf("did not expect a /private twin for a /Users path; got:\n%s", got)
	}
}

// TestSeatbeltProfile_NoHomeOverlay confirms the HOME overlay line is omitted
// when the sandbox has no home overlay mounted.
func TestSeatbeltProfile_NoHomeOverlay(t *testing.T) {
	s := seatbeltSandbox() // HomeMergedDir empty
	cfg := DefaultBwrapConfig()
	cfg.HostHome = "/Users/dev"
	got := SeatbeltProfile(s, cfg)
	if strings.Contains(got, "/home") {
		t.Errorf("expected no home overlay subpath when HomeMergedDir empty; got:\n%s", got)
	}
}

// TestBuildSeatbeltCommand_NoLimits pins the sandbox-exec argv when no
// resource limits are configured: the target runs directly under the profile,
// with no shim prefix.
func TestBuildSeatbeltCommand_NoLimits(t *testing.T) {
	cfg := DefaultBwrapConfig()
	exe, args := BuildSeatbeltCommand("/opt/api/workspace-sandbox-api", seatbeltSandbox(), cfg, "/bin/echo", "hi")

	if exe != "sandbox-exec" {
		t.Errorf("executable: got %q, want sandbox-exec", exe)
	}
	if len(args) < 4 || args[0] != "-p" {
		t.Fatalf("expected [-p <profile> …], got %v", args)
	}
	// After the profile, the target command follows immediately (no shim).
	tail := args[2:]
	want := []string{"/bin/echo", "hi"}
	if !reflect.DeepEqual(tail, want) {
		t.Errorf("target argv: got %v, want %v", tail, want)
	}
}

// TestBuildSeatbeltCommand_WithLimits pins the shim prefix that wraps the
// target when resource limits are configured, mirroring how BuildExecCommand
// prepends prlimit ahead of bwrap on Linux.
func TestBuildSeatbeltCommand_WithLimits(t *testing.T) {
	cfg := DefaultBwrapConfig()
	cfg.ResourceLimits = ResourceLimits{
		MemoryLimitMB: 512,
		CPUTimeSec:    30,
		MaxProcesses:  64,
		MaxOpenFiles:  256,
		TimeoutSec:    120, // context-enforced, must NOT appear in the shim argv
	}
	const shim = "/opt/api/workspace-sandbox-api"
	exe, args := BuildSeatbeltCommand(shim, seatbeltSandbox(), cfg, "node", "server.js")

	if exe != "sandbox-exec" {
		t.Errorf("executable: got %q, want sandbox-exec", exe)
	}
	// args = [-p <profile> <shim> rlimit-exec --as=… --cpu=… --nproc=… --nofile=… -- node server.js]
	tail := args[2:]
	want := []string{
		shim, "rlimit-exec",
		"--as=536870912", // 512 MiB in bytes
		"--cpu=30",
		"--nproc=64",
		"--nofile=256",
		"--",
		"node", "server.js",
	}
	if !reflect.DeepEqual(tail, want) {
		t.Errorf("shim-wrapped argv:\n got  %v\n want %v", tail, want)
	}
	for _, a := range tail {
		if strings.Contains(a, "120") {
			t.Errorf("TimeoutSec must be context-enforced, not in shim argv; found %q", a)
		}
	}
}

// TestBuildRlimitShimArgs pins the shim flag assembly, including the nil
// result when no limits are set.
func TestBuildRlimitShimArgs(t *testing.T) {
	if got := BuildRlimitShimArgs("/shim", ResourceLimits{}); got != nil {
		t.Errorf("expected nil shim args when no limits set, got %v", got)
	}
	if got := BuildRlimitShimArgs("/shim", ResourceLimits{TimeoutSec: 99}); got != nil {
		t.Errorf("TimeoutSec alone must not trigger the shim (it is context-enforced), got %v", got)
	}
	got := BuildRlimitShimArgs("/shim", ResourceLimits{MemoryLimitMB: 1, MaxOpenFiles: 8})
	want := []string{"/shim", "rlimit-exec", "--as=1048576", "--nofile=8", "--"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("shim args: got %v, want %v", got, want)
	}
}
