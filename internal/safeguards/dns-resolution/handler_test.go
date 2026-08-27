package dnsresolution

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

var stubLookups = dnsResolutionStubLookups

func dnsResolutionStubLookups(t *testing.T) func() {
	t.Helper()
	origLookPath := hostreqkit.LookPathFn
	origReadFile := hostreqkit.ReadFileFn
	origCombinedOutput := hostreqkit.CombinedOutputFn
	origRunCommand := hostreqkit.RunCommandFn
	origResolve := ResolveFn
	return func() {
		hostreqkit.LookPathFn = origLookPath
		hostreqkit.ReadFileFn = origReadFile
		hostreqkit.CombinedOutputFn = origCombinedOutput
		hostreqkit.RunCommandFn = origRunCommand
		ResolveFn = origResolve
	}
}

var newTestHandler = dnsResolutionTestHandler

func dnsResolutionTestHandler() hostreqkit.Handler {
	return NewHandler(hostreqkit.SafeguardManifest{
		Name:    "dns_resolution",
		Handler: "dns_resolution",
	})
}

func linuxReq() hostreqspec.ResolvedRequirement {
	return hostreqspec.ResolvedRequirement{
		Name:     "dns_resolution",
		Kind:     hostreqspec.KindSafeguard,
		Required: true,
	}
}

var linuxHost = dnsResolutionLinuxHost

func dnsResolutionLinuxHost() hostreqkit.Host {
	return hostreqkit.Host{
		OS:              "linux",
		PackageManager:  "apt-get",
		SupportsSystemd: true,
	}
}

func TestInspectNoSystemdNotApplicable(t *testing.T) {
	h := newTestHandler()
	host := linuxHost()
	host.SupportsSystemd = false
	status := h.Inspect(host, linuxReq())
	if status.SupportClass != hostreqkit.SupportNotApplicable {
		t.Fatalf("SupportClass = %q", status.SupportClass)
	}
}

func TestInspectConfigPresent(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		if path == resolvedConfigPath {
			return []byte(resolvedContent), nil
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

func TestInspectConfigMissing(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	hostreqkit.ReadFileFn = func(string) ([]byte, error) {
		return nil, os.ErrNotExist
	}
	ResolveFn = func(string) error { return nil }

	h := newTestHandler()
	status := h.Inspect(linuxHost(), linuxReq())
	if status.Applied {
		t.Fatal("expected Applied = false")
	}
	found := false
	for _, note := range status.Notes {
		if strings.Contains(note, "not applied") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected 'not applied' note, got: %v", status.Notes)
	}
}

func TestInspectConfigMissingDNSFailing(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	hostreqkit.ReadFileFn = func(string) ([]byte, error) {
		return nil, os.ErrNotExist
	}
	ResolveFn = func(string) error { return fmt.Errorf("lookup failed") }

	h := newTestHandler()
	status := h.Inspect(linuxHost(), linuxReq())
	if status.Applied {
		t.Fatal("expected Applied = false")
	}
	foundDNS := false
	for _, note := range status.Notes {
		if strings.Contains(note, "failing") {
			foundDNS = true
			break
		}
	}
	if !foundDNS {
		t.Fatalf("expected DNS failing note, got: %v", status.Notes)
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

	foundRestart := false
	foundFlush := false
	for _, call := range calls {
		if strings.Contains(call, "systemctl") && strings.Contains(call, "restart") {
			foundRestart = true
		}
		if strings.Contains(call, "resolvectl") && strings.Contains(call, "flush") {
			foundFlush = true
		}
	}
	if !foundRestart {
		t.Fatalf("expected systemctl restart call, got: %v", calls)
	}
	if !foundFlush {
		t.Fatalf("expected resolvectl flush call, got: %v", calls)
	}
}

func TestApplyInstallContentFailure(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	hostreqkit.LookPathFn = func(name string) (string, error) {
		return "/usr/bin/" + name, nil
	}

	callCount := 0
	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		callCount++
		// First call is mkdir (succeeds), second is install (fails).
		if callCount >= 2 {
			return fmt.Errorf("permission denied")
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
