package natprotection

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

var stubLookups = natProtectionStubLookups

func natProtectionStubLookups(t *testing.T) func() {
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

var newTestHandler = natProtectionTestHandler

func natProtectionTestHandler() hostreqkit.Handler {
	return NewHandler(hostreqkit.SafeguardManifest{
		Name:    "nat_protection",
		Handler: "nat_protection",
	})
}

var linuxReq = natProtectionLinuxReq

func natProtectionLinuxReq() hostreqspec.ResolvedRequirement {
	return hostreqspec.ResolvedRequirement{
		Name:     "nat_protection",
		Kind:     hostreqspec.KindSafeguard,
		Required: true,
	}
}

var linuxHost = natProtectionLinuxHost

func natProtectionLinuxHost() hostreqkit.Host {
	return hostreqkit.Host{
		OS:             "linux",
		PackageManager: "apt-get",
	}
}

// iptables output with two REDIRECT rules.
const iptablesWithRedirects = `-P OUTPUT ACCEPT
-A OUTPUT -p tcp -m tcp --dport 443 -j REDIRECT --to-ports 8085
-A OUTPUT -p tcp -m tcp --dport 80 -j REDIRECT --to-ports 8080
`

// ss output showing port 8080 listening but NOT 8085.
const ssPortListening = `State    Recv-Q   Send-Q     Local Address:Port     Peer Address:Port
LISTEN   0        128              0.0.0.0:8080          0.0.0.0:*
LISTEN   0        128              0.0.0.0:22            0.0.0.0:*
`

// ss output showing both ports listening (no dead redirects).
const ssBothListening = `State    Recv-Q   Send-Q     Local Address:Port     Peer Address:Port
LISTEN   0        128              0.0.0.0:8080          0.0.0.0:*
LISTEN   0        128              0.0.0.0:8085          0.0.0.0:*
`

// ss output showing nothing listening (both redirects dead).
const ssNoneListening = `State    Recv-Q   Send-Q     Local Address:Port     Peer Address:Port
LISTEN   0        128              0.0.0.0:22            0.0.0.0:*
`

func TestInspectNoIptablesNotApplicable(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	hostreqkit.LookPathFn = func(name string) (string, error) {
		return "", os.ErrNotExist
	}

	h := newTestHandler()
	status := h.Inspect(linuxHost(), linuxReq())
	if status.SupportClass != hostreqkit.SupportNotApplicable {
		t.Fatalf("SupportClass = %q", status.SupportClass)
	}
}

func TestInspectNoDeadRedirects(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	hostreqkit.LookPathFn = func(name string) (string, error) {
		return "/usr/sbin/" + name, nil
	}
	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		if name == "iptables" {
			return []byte(iptablesWithRedirects), nil
		}
		if name == "ss" {
			return []byte(ssBothListening), nil
		}
		return nil, fmt.Errorf("unexpected command: %s", name)
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

func TestInspectNoRedirectRules(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	hostreqkit.LookPathFn = func(name string) (string, error) {
		return "/usr/sbin/" + name, nil
	}
	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		if name == "iptables" {
			return []byte("-P OUTPUT ACCEPT\n"), nil
		}
		if name == "ss" {
			return []byte(""), nil
		}
		return nil, fmt.Errorf("unexpected command: %s", name)
	}

	h := newTestHandler()
	status := h.Inspect(linuxHost(), linuxReq())
	if !status.Applied {
		t.Fatalf("expected Applied = true, notes: %v", status.Notes)
	}
}

func TestInspectDeadRedirectsFound(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	hostreqkit.LookPathFn = func(name string) (string, error) {
		return "/usr/sbin/" + name, nil
	}
	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		if name == "iptables" {
			return []byte(iptablesWithRedirects), nil
		}
		if name == "ss" {
			return []byte(ssPortListening), nil // 8080 listening, 8085 not
		}
		return nil, fmt.Errorf("unexpected command: %s", name)
	}

	h := newTestHandler()
	status := h.Inspect(linuxHost(), linuxReq())
	if status.Applied {
		t.Fatal("expected Applied = false")
	}
	found := false
	for _, note := range status.Notes {
		if strings.Contains(note, "8085") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected note about port 8085, got: %v", status.Notes)
	}
}

func TestInspectAllRedirectsDead(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	hostreqkit.LookPathFn = func(name string) (string, error) {
		return "/usr/sbin/" + name, nil
	}
	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		if name == "iptables" {
			return []byte(iptablesWithRedirects), nil
		}
		if name == "ss" {
			return []byte(ssNoneListening), nil
		}
		return nil, fmt.Errorf("unexpected command: %s", name)
	}

	h := newTestHandler()
	status := h.Inspect(linuxHost(), linuxReq())
	if status.Applied {
		t.Fatal("expected Applied = false")
	}
	count := 0
	for _, note := range status.Notes {
		if strings.Contains(note, "dead redirect") {
			count++
		}
	}
	if count != 2 {
		t.Fatalf("expected 2 dead redirect notes, got %d: %v", count, status.Notes)
	}
}

func TestApplyRemovesDeadRedirects(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	hostreqkit.LookPathFn = func(name string) (string, error) {
		return "/usr/sbin/" + name, nil
	}

	// Track iptables -D calls.
	var removals []string
	iptablesCallNum := 0
	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		if name == "iptables" {
			iptablesCallNum++
			// First call is from findDeadRedirects before removal.
			// Second call is from the verification findDeadRedirects.
			if iptablesCallNum <= 1 {
				return []byte(iptablesWithRedirects), nil
			}
			// After removal, return clean output.
			return []byte("-P OUTPUT ACCEPT\n"), nil
		}
		if name == "ss" {
			return []byte(ssNoneListening), nil // both dead
		}
		return nil, fmt.Errorf("unexpected: %s", name)
	}

	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		removals = append(removals, strings.Join(args, " "))
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
	if len(removals) != 2 {
		t.Fatalf("expected 2 removal calls, got %d: %v", len(removals), removals)
	}
}

func TestApplyPartialRemovalFailure(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	hostreqkit.LookPathFn = func(name string) (string, error) {
		return "/usr/sbin/" + name, nil
	}

	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		if name == "iptables" {
			// Both calls return rules (removal didn't stick).
			return []byte(iptablesWithRedirects), nil
		}
		if name == "ss" {
			return []byte(ssNoneListening), nil
		}
		return nil, fmt.Errorf("unexpected: %s", name)
	}

	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		return fmt.Errorf("permission denied")
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

func TestApplyNoDeadRedirectsAtApplyTime(t *testing.T) {
	restore := stubLookups(t)
	defer restore()

	hostreqkit.LookPathFn = func(name string) (string, error) {
		return "/usr/sbin/" + name, nil
	}
	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		if name == "iptables" {
			return []byte("-P OUTPUT ACCEPT\n"), nil
		}
		if name == "ss" {
			return []byte(""), nil
		}
		return nil, fmt.Errorf("unexpected: %s", name)
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
	if status.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Fatalf("ExecutionState = %q", status.ExecutionState)
	}
}

func TestPortListeningAvoidsFalsePositive(t *testing.T) {
	// Port 80 should not match :8080
	ssOutput := "LISTEN   0   128   0.0.0.0:8080   0.0.0.0:*\n"
	if portListening(ssOutput, "80") {
		t.Fatal("port 80 should not match :8080")
	}
	if !portListening(ssOutput, "8080") {
		t.Fatal("port 8080 should match :8080")
	}
}

func TestPortListeningEndOfLine(t *testing.T) {
	ssOutput := "LISTEN   0   128   0.0.0.0:443"
	if !portListening(ssOutput, "443") {
		t.Fatal("port 443 should match at end of line")
	}
}
