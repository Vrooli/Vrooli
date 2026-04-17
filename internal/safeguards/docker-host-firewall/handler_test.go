package dockerhostfirewall

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

func stubLookups(t *testing.T) func() {
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

func newTestHandler() hostreqkit.Handler {
	return NewHandler(hostreqkit.SafeguardManifest{
		Name:    "docker_host_firewall",
		Handler: "docker_host_firewall",
	})
}

func linuxReq() hostreqspec.ResolvedRequirement {
	return hostreqspec.ResolvedRequirement{
		Name:     "docker_host_firewall",
		Kind:     hostreqspec.KindSafeguard,
		Required: true,
	}
}

func linuxHost() hostreqkit.Host {
	return hostreqkit.Host{
		OS:             "linux",
		PackageManager: "apt-get",
	}
}

func TestNameAndKind(t *testing.T) {
	h := newTestHandler()
	if h.Name() != "docker_host_firewall" {
		t.Fatalf("Name = %q", h.Name())
	}
	if h.Kind() != hostreqspec.KindSafeguard {
		t.Fatalf("Kind = %q", h.Kind())
	}
}

func TestInspectManualRequirement(t *testing.T) {
	h := newTestHandler()
	req := linuxReq()
	req.Manual = true
	status := h.Inspect(linuxHost(), req)
	if status.SupportClass != hostreqkit.SupportManualOnly {
		t.Fatalf("SupportClass = %q", status.SupportClass)
	}
}

func TestInspectNonLinuxUnsupported(t *testing.T) {
	h := newTestHandler()
	status := h.Inspect(hostreqkit.Host{OS: "darwin"}, linuxReq())
	if status.SupportClass != hostreqkit.SupportUnsupported {
		t.Fatalf("SupportClass = %q", status.SupportClass)
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

func TestApplyUnsupportedReturnsEarly(t *testing.T) {
	h := newTestHandler()
	status, err := h.Apply(linuxHost(), hostreqkit.ItemStatus{
		SupportClass: hostreqkit.SupportUnsupported,
	}, hostreqkit.EnsureOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if status.ExecutionState != hostreqkit.ExecutionUnsupported {
		t.Fatalf("ExecutionState = %q", status.ExecutionState)
	}
}

func TestApplyNotApplicableReturnsEarly(t *testing.T) {
	h := newTestHandler()
	status, err := h.Apply(linuxHost(), hostreqkit.ItemStatus{
		SupportClass: hostreqkit.SupportNotApplicable,
	}, hostreqkit.EnsureOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if status.ExecutionState != hostreqkit.ExecutionNotApplicable {
		t.Fatalf("ExecutionState = %q", status.ExecutionState)
	}
}

func TestApplyManualReturnsEarly(t *testing.T) {
	h := newTestHandler()
	status, err := h.Apply(linuxHost(), hostreqkit.ItemStatus{
		SupportClass: hostreqkit.SupportManualOnly,
	}, hostreqkit.EnsureOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if status.ExecutionState != hostreqkit.ExecutionManualActionRequired {
		t.Fatalf("ExecutionState = %q", status.ExecutionState)
	}
}

func TestApplyAlreadyAppliedSkips(t *testing.T) {
	h := newTestHandler()
	status, err := h.Apply(linuxHost(), hostreqkit.ItemStatus{
		SupportClass: hostreqkit.SupportSupported,
		Applied:      true,
	}, hostreqkit.EnsureOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if status.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Fatalf("ExecutionState = %q", status.ExecutionState)
	}
}

func TestApplyDryRun(t *testing.T) {
	h := newTestHandler()
	status, err := h.Apply(linuxHost(), hostreqkit.ItemStatus{
		SupportClass: hostreqkit.SupportSupported,
	}, hostreqkit.EnsureOptions{DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if status.ExecutionState != hostreqkit.ExecutionWouldApply {
		t.Fatalf("ExecutionState = %q", status.ExecutionState)
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
