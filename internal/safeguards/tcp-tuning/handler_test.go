//nolint:goconst // test data deliberately reuses stable sysfs fixtures.
package tcptuning

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqkit/hostreqkittest"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

var stubLookups = hostreqkittest.StubLookups

var newTestHandler = tcpTuningTestHandler

func tcpTuningTestHandler() hostreqkit.Handler {
	return NewHandler(hostreqkit.SafeguardManifest{
		Name:    "tcp_tuning",
		Handler: "tcp_tuning",
	})
}

var linuxReq = tcpTuningLinuxReq

func tcpTuningLinuxReq() hostreqspec.ResolvedRequirement {
	return hostreqspec.ResolvedRequirement{
		Name:     "tcp_tuning",
		Kind:     hostreqspec.KindSafeguard,
		Required: true,
	}
}

var linuxHost = tcpTuningLinuxHost

func tcpTuningLinuxHost() hostreqkit.Host {
	return hostreqkit.Host{
		OS:             "linux",
		PackageManager: "apt-get",
		SupportsSysctl: true,
	}
}

func TestInspectAllParametersMet(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		switch path {
		case "/proc/sys/net/ipv4/tcp_ecn":
			return []byte("0\n"), nil
		case "/proc/sys/net/ipv4/tcp_mtu_probing":
			return []byte("1\n"), nil
		case "/proc/sys/net/ipv4/tcp_base_mss":
			return []byte("1024\n"), nil
		case configPath:
			return []byte(buildConfigContent()), nil
		}
		return nil, os.ErrNotExist
	}

	h := newTestHandler()
	status := h.Inspect(linuxHost(), linuxReq())
	if !status.Applied {
		t.Fatalf("expected Applied = true, notes: %v", status.Notes)
	}
	if status.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Fatalf("ExecutionState = %q", status.ExecutionState)
	}
}

func TestInspectParametersMismatch(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		switch path {
		case "/proc/sys/net/ipv4/tcp_ecn":
			return []byte("2\n"), nil // default is 2, we want 0
		case "/proc/sys/net/ipv4/tcp_mtu_probing":
			return []byte("0\n"), nil // default is 0, we want 1
		case "/proc/sys/net/ipv4/tcp_base_mss":
			return []byte("512\n"), nil // default is 512, we want 1024
		}
		return nil, os.ErrNotExist
	}

	h := newTestHandler()
	status := h.Inspect(linuxHost(), linuxReq())
	if status.Applied {
		t.Fatal("expected Applied = false")
	}
	found := false
	for _, note := range status.Notes {
		if strings.Contains(note, "pending") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected pending note, got: %v", status.Notes)
	}
}

func TestInspectConfigFileMismatch(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		switch path {
		case "/proc/sys/net/ipv4/tcp_ecn":
			return []byte("0\n"), nil
		case "/proc/sys/net/ipv4/tcp_mtu_probing":
			return []byte("1\n"), nil
		case "/proc/sys/net/ipv4/tcp_base_mss":
			return []byte("1024\n"), nil
		case configPath:
			return []byte("# wrong content\n"), nil
		}
		return nil, os.ErrNotExist
	}

	h := newTestHandler()
	status := h.Inspect(linuxHost(), linuxReq())
	if status.Applied {
		t.Fatal("expected Applied = false when config file doesn't match")
	}
}

func TestApplyRunsFullFlow(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	hostreqkit.LookPathFn = func(name string) (string, error) {
		return "/usr/bin/" + name, nil
	}

	var calls []string
	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		calls = append(calls, name+" "+strings.Join(args, " "))
		return nil
	}

	h := newTestHandler()
	status, err := h.Apply(linuxHost(), hostreqkit.ItemStatus{
		SupportClass: hostreqkit.SupportSupported,
	}, hostreqkit.EnsureOptions{SudoMode: "ask"})
	if err != nil {
		t.Fatal(err)
	}
	if !status.Applied {
		t.Fatalf("expected Applied = true, notes: %v", status.Notes)
	}
	if status.ExecutionState != hostreqkit.ExecutionApplied {
		t.Fatalf("ExecutionState = %q", status.ExecutionState)
	}

	foundSysctl := false
	for _, call := range calls {
		if strings.Contains(call, "sysctl") && strings.Contains(call, "--system") {
			foundSysctl = true
			break
		}
	}
	if !foundSysctl {
		t.Fatalf("expected sysctl --system call, got: %v", calls)
	}
}

func TestApplySysctlFailure(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	hostreqkit.LookPathFn = func(name string) (string, error) {
		return "/usr/bin/" + name, nil
	}

	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		for _, arg := range args {
			if arg == "sysctl" {
				return fmt.Errorf("sysctl: permission denied")
			}
		}
		return nil
	}

	h := newTestHandler()
	status, err := h.Apply(linuxHost(), hostreqkit.ItemStatus{
		SupportClass: hostreqkit.SupportSupported,
	}, hostreqkit.EnsureOptions{SudoMode: "ask"})
	if err != nil {
		t.Fatal(err)
	}
	if status.ExecutionState != hostreqkit.ExecutionFailed {
		t.Fatalf("ExecutionState = %q, want failed", status.ExecutionState)
	}
}
