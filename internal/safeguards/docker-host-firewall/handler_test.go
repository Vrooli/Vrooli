//nolint:goconst // test data deliberately reuses stable command fixtures.
package dockerhostfirewall

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

var stubLookups = dockerFirewallStubLookups

func dockerFirewallStubLookups(t *testing.T) func() {
	t.Helper()
	origLookPath := hostreqkit.LookPathFn
	origReadFile := hostreqkit.ReadFileFn
	origCombinedOutput := hostreqkit.CombinedOutputFn
	origRunCommand := hostreqkit.RunCommandFn
	return func() {
		hostreqkit.LookPathFn = origLookPath
		hostreqkit.ReadFileFn = origReadFile
		hostreqkit.CombinedOutputFn = origCombinedOutput
		hostreqkit.RunCommandFn = origRunCommand
	}
}

var newTestHandler = dockerFirewallTestHandler

func dockerFirewallTestHandler() hostreqkit.Handler {
	return NewHandler(hostreqkit.SafeguardManifest{
		Name:    "docker_host_firewall",
		Handler: "docker_host_firewall",
	})
}

var linuxReq = dockerFirewallLinuxReq

func dockerFirewallLinuxReq() hostreqspec.ResolvedRequirement {
	return hostreqspec.ResolvedRequirement{
		Name:     "docker_host_firewall",
		Kind:     hostreqspec.KindSafeguard,
		Required: true,
	}
}

var linuxHost = dockerFirewallLinuxHost

func dockerFirewallLinuxHost() hostreqkit.Host {
	return hostreqkit.Host{
		OS:             "linux",
		PackageManager: "apt-get",
	}
}

func TestInspectNoIptablesUnsupported(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	hostreqkit.LookPathFn = func(name string) (string, error) {
		return "", os.ErrNotExist
	}

	h := newTestHandler()
	status := h.Inspect(linuxHost(), linuxReq())
	if status.SupportClass != hostreqkit.SupportUnsupported {
		t.Fatalf("SupportClass = %q", status.SupportClass)
	}
}

func TestInspectNoDockerNotApplicable(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	hostreqkit.LookPathFn = func(name string) (string, error) {
		if name == "iptables" {
			return "/usr/sbin/iptables", nil
		}
		return "", os.ErrNotExist
	}

	h := newTestHandler()
	status := h.Inspect(linuxHost(), linuxReq())
	if status.SupportClass != hostreqkit.SupportNotApplicable {
		t.Fatalf("SupportClass = %q", status.SupportClass)
	}
}

func TestInspectChainExistsAndWired(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	hostreqkit.LookPathFn = func(name string) (string, error) {
		if name == "iptables" || name == "docker" {
			return "/usr/bin/" + name, nil
		}
		return "", os.ErrNotExist
	}

	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		return []byte("OK"), nil
	}

	h := newTestHandler()
	status := h.Inspect(linuxHost(), linuxReq())
	if !status.Applied {
		t.Fatalf("expected Applied = true, notes: %v", status.Notes)
	}
}

func TestInspectChainMissing(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	hostreqkit.LookPathFn = func(name string) (string, error) {
		if name == "iptables" || name == "docker" {
			return "/usr/bin/" + name, nil
		}
		return "", os.ErrNotExist
	}

	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		return nil, fmt.Errorf("iptables: No chain/target/match by that name")
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

func TestApplyCreatesChainAndWires(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	hostreqkit.LookPathFn = func(name string) (string, error) {
		return "/usr/bin/" + name, nil
	}

	created := false
	wired := false
	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		for i, arg := range args {
			if arg == "-L" && i+1 < len(args) && args[i+1] == chainName {
				if created {
					return []byte("OK"), nil
				}
				return nil, fmt.Errorf("no chain")
			}
			if arg == "-C" && i+1 < len(args) && args[i+1] == dockerUserChain {
				if wired {
					return []byte("OK"), nil
				}
				return nil, fmt.Errorf("no rule")
			}
		}
		return nil, fmt.Errorf("unexpected iptables call: %v", args)
	}

	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		for _, arg := range args {
			if arg == "-N" {
				created = true
			}
			if arg == "-I" {
				wired = true
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
	if !status.Applied {
		t.Fatalf("expected Applied = true, notes: %v", status.Notes)
	}
	if status.ExecutionState != hostreqkit.ExecutionApplied {
		t.Fatalf("ExecutionState = %q", status.ExecutionState)
	}
}

func TestApplyChainCreationFailure(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	hostreqkit.LookPathFn = func(name string) (string, error) {
		return "/usr/bin/" + name, nil
	}

	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		return nil, fmt.Errorf("no chain")
	}

	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		return fmt.Errorf("iptables: Permission denied")
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
