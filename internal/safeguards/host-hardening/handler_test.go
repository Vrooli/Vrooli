package hosthardening

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqkit/hostreqkittest"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

var stubLookups = hostreqkittest.StubLookups

var newTestHandler = hostHardeningTestHandler

func hostHardeningTestHandler() hostreqkit.Handler {
	return NewHandler(hostreqkit.SafeguardManifest{
		Name:    "host_hardening",
		Handler: "host_hardening",
	})
}

var linuxReq = hostHardeningLinuxReq

func hostHardeningLinuxReq() hostreqspec.ResolvedRequirement {
	return hostreqspec.ResolvedRequirement{
		Name:     "host_hardening",
		Kind:     hostreqspec.KindSafeguard,
		Required: true,
	}
}

var linuxHost = hostHardeningLinuxHost

func hostHardeningLinuxHost() hostreqkit.Host {
	return hostreqkit.Host{
		OS:             "linux",
		PackageManager: "apt-get",
		SupportsSysctl: true,
	}
}

// defaultPolicy is what an unconfigured requirement resolves to.
func defaultPolicy() policy { return resolvePolicy(nil) }

// withArmedKdump serves the crash-kernel probe as armed and delegates the rest.
// Tests about sysctl behaviour would otherwise be short-circuited by the
// prerequisite check, which has its own dedicated tests.
func withArmedKdump(next func(string) ([]byte, error)) func(string) ([]byte, error) {
	return func(path string) ([]byte, error) {
		if path == kexecCrashLoadedPath {
			return []byte("1\n"), nil
		}
		return next(path)
	}
}

// readSysctlsAtTarget makes ReadFileFn return the desired live values for
// every managed sysctl under the given policy. Tests that exercise drift
// override this.
func readSysctlsAtTarget(pol policy) func(string) ([]byte, error) {
	return withArmedKdump(func(path string) ([]byte, error) {
		switch path {
		case sysctlPath:
			return []byte(buildSysctlContent(pol)), nil
		case journaldPath:
			return []byte(buildJournaldContent()), nil
		}
		for _, setting := range managedSysctls(pol) {
			procPath := "/proc/sys/" + strings.ReplaceAll(setting.Name, ".", "/")
			if path == procPath {
				return []byte(fmt.Sprintf("%d\n", setting.Value)), nil
			}
		}
		return nil, os.ErrNotExist
	})
}

func TestInspectAllInPlace(t *testing.T) {
	restore := stubLookups(t)
	defer restore()
	hostreqkit.ReadFileFn = readSysctlsAtTarget(defaultPolicy())

	h := newTestHandler()
	status := h.Inspect(linuxHost(), linuxReq())
	if !status.Applied {
		t.Fatalf("expected Applied = true; notes: %v", status.Notes)
	}
	if status.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Fatalf("ExecutionState = %q", status.ExecutionState)
	}
}

func TestInspectSysctlDrift(t *testing.T) {
	restore := stubLookups(t)
	defer restore()
	target := readSysctlsAtTarget(defaultPolicy())
	hostreqkit.ReadFileFn = withArmedKdump(func(path string) ([]byte, error) {
		// Pretend kernel.panic_on_oops is still 0 even though the drop-in is
		// in place — drift after a one-shot sysctl -w from somewhere.
		if path == "/proc/sys/kernel/panic_on_oops" {
			return []byte("0\n"), nil
		}
		return target(path)
	})

	h := newTestHandler()
	status := h.Inspect(linuxHost(), linuxReq())
	if status.Applied {
		t.Fatal("expected Applied = false when a sysctl drifts")
	}
	joined := strings.Join(status.Notes, " | ")
	if !strings.Contains(joined, "kernel.panic_on_oops=1 (current: 0)") {
		t.Errorf("expected drift to surface in notes; got %v", status.Notes)
	}
}

func TestInspectMissingPolicyFiles(t *testing.T) {
	for _, missingPath := range []string{sysctlPath, journaldPath} {
		t.Run(missingPath, func(t *testing.T) {
			restore := stubLookups(t)
			defer restore()
			target := readSysctlsAtTarget(defaultPolicy())
			hostreqkit.ReadFileFn = withArmedKdump(func(path string) ([]byte, error) {
				if path == missingPath {
					return nil, os.ErrNotExist
				}
				return target(path)
			})

			status := newTestHandler().Inspect(linuxHost(), linuxReq())
			if status.Applied {
				t.Fatalf("expected Applied = false when %s is missing", missingPath)
			}
			if !strings.Contains(strings.Join(status.Notes, " | "), missingPath+" needs update") {
				t.Errorf("expected missing-path note for %s; got %v", missingPath, status.Notes)
			}
		})
	}
}

func TestApplyDryRunDoesNotMutate(t *testing.T) {
	restore := stubLookups(t)
	defer restore()
	hostreqkit.ReadFileFn = withArmedKdump(func(string) ([]byte, error) { return nil, os.ErrNotExist })

	calls := 0
	hostreqkit.RunCommandFn = func(string, []string, hostreqkit.EnsureOptions) error {
		calls++
		return nil
	}

	h := newTestHandler()
	status := h.Inspect(linuxHost(), linuxReq())
	out, err := h.Apply(linuxHost(), status, hostreqkit.EnsureOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Apply error: %v", err)
	}
	if out.ExecutionState != hostreqkit.ExecutionWouldApply {
		t.Fatalf("ExecutionState = %q, want WouldApply", out.ExecutionState)
	}
	if calls != 0 {
		t.Errorf("dry-run ran %d commands; want 0", calls)
	}
}

func TestApplyAlreadyAppliedShortCircuits(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	calls := 0
	hostreqkit.RunCommandFn = func(string, []string, hostreqkit.EnsureOptions) error {
		calls++
		return nil
	}

	h := newTestHandler()
	out, err := h.Apply(linuxHost(), hostreqkit.ItemStatus{
		SupportClass: hostreqkit.SupportSupported,
		Applied:      true,
	}, hostreqkit.EnsureOptions{})
	if err != nil {
		t.Fatalf("Apply error: %v", err)
	}
	if out.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Fatalf("ExecutionState = %q", out.ExecutionState)
	}
	if calls != 0 {
		t.Errorf("already-applied path ran %d commands; want 0", calls)
	}
}

func TestApplyRunsSysctlAndRestartsJournald(t *testing.T) {
	restore := stubLookups(t)
	defer restore()
	// Files do not exist yet; sysctls drift; Apply should write everything
	// and run both commands.
	hostreqkit.ReadFileFn = withArmedKdump(func(string) ([]byte, error) { return nil, os.ErrNotExist })

	type call struct {
		name string
		args []string
	}
	var calls []call
	hostreqkit.RunCommandFn = func(name string, args []string, _ hostreqkit.EnsureOptions) error {
		calls = append(calls, call{name: name, args: append([]string(nil), args...)})
		return nil
	}

	h := newTestHandler()
	status := h.Inspect(linuxHost(), linuxReq())
	out, err := h.Apply(linuxHost(), status, hostreqkit.EnsureOptions{SudoMode: "ask"})
	if err != nil {
		t.Fatalf("Apply error: %v", err)
	}
	if out.ExecutionState != hostreqkit.ExecutionApplied {
		t.Fatalf("ExecutionState = %q, want Applied; notes: %v", out.ExecutionState, out.Notes)
	}

	// SudoMode "ask" prefixes commands with `sudo`, so the program name
	// becomes "sudo" and the original command shifts into args[0].
	sawSysctl := false
	sawRestart := false
	for _, c := range calls {
		flat := append([]string{c.name}, c.args...)
		joined := strings.Join(flat, " ")
		if strings.Contains(joined, "sysctl --system") {
			sawSysctl = true
		}
		if strings.Contains(joined, "systemctl restart systemd-journald") {
			sawRestart = true
		}
	}
	if !sawSysctl {
		t.Errorf("expected `sysctl --system` call; got %+v", calls)
	}
	if !sawRestart {
		t.Errorf("expected `systemctl restart systemd-journald` call; got %+v", calls)
	}
}

func TestApplySysctlFailureSurfaced(t *testing.T) {
	restore := stubLookups(t)
	defer restore()
	hostreqkit.ReadFileFn = withArmedKdump(func(string) ([]byte, error) { return nil, os.ErrNotExist })

	hostreqkit.RunCommandFn = func(name string, args []string, _ hostreqkit.EnsureOptions) error {
		// With SudoMode "ask" the program is `sudo`; check args for the
		// actual command we want to fail.
		flat := append([]string{name}, args...)
		if strings.Contains(strings.Join(flat, " "), "sysctl --system") {
			return errors.New("sysctl: invalid value")
		}
		return nil
	}

	h := newTestHandler()
	status := h.Inspect(linuxHost(), linuxReq())
	out, err := h.Apply(linuxHost(), status, hostreqkit.EnsureOptions{SudoMode: "ask"})
	if err != nil {
		t.Fatalf("Apply error: %v", err)
	}
	if out.ExecutionState != hostreqkit.ExecutionFailed {
		t.Fatalf("ExecutionState = %q, want Failed", out.ExecutionState)
	}
	if !strings.Contains(strings.Join(out.Notes, " | "), "sysctl --system` failed") {
		t.Errorf("expected sysctl-failure note; got %v", out.Notes)
	}
}

func TestBuildSysctlContentIncludesAllParams(t *testing.T) {
	content := buildSysctlContent(defaultPolicy())
	for _, p := range managedSysctls(defaultPolicy()) {
		want := fmt.Sprintf("%s = %d", p.Name, p.Value)
		if !strings.Contains(content, want) {
			t.Errorf("missing %q", want)
		}
	}
}

func TestBuildJournaldContentHasRateLimit(t *testing.T) {
	content := buildJournaldContent()
	for _, want := range []string{"[Journal]", "RateLimitIntervalSec=30s", "RateLimitBurst=10000"} {
		if !strings.Contains(content, want) {
			t.Errorf("missing %q", want)
		}
	}
}
