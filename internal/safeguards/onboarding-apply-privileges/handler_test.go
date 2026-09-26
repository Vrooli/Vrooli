package onboardingapplyprivileges

import (
	"os"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/testenv"
)

func TestPolicyContainsOnlyLiteralSelectedCommands(t *testing.T) {
	content, err := policyContent("operator", ConfigForRequirements("/usr/local/bin/vrooli", []string{"git", "qemu"}, []string{"clock"}))
	if err != nil {
		t.Fatal(err)
	}
	if strings.ContainsAny(content, "*?[]") {
		t.Fatalf("policy contains wildcard syntax: %q", content)
	}
	for _, want := range []string{
		"/usr/local/bin/vrooli host install git --json --sudo-mode error",
		"/usr/local/bin/vrooli host install qemu --json --sudo-mode error",
		"/usr/local/bin/vrooli host safeguard clock --json --sudo-mode error",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("policy missing %q: %s", want, content)
		}
	}
}

func TestDarwinGrantUsesSameValidatedLiteralPolicy(t *testing.T) {
	oldRoot := hostreqkit.RunningAsRootFn
	oldRead := hostreqkit.ReadFileFn
	oldWrite := hostreqkit.WriteTempFileFn
	oldRun := hostreqkit.RunCommandFn
	t.Cleanup(func() {
		hostreqkit.RunningAsRootFn = oldRoot
		hostreqkit.ReadFileFn = oldRead
		hostreqkit.WriteTempFileFn = oldWrite
		hostreqkit.RunCommandFn = oldRun
	})
	testenv.AsCurrentUser(t, "operator")
	hostreqkit.RunningAsRootFn = func() bool { return true }
	hostreqkit.ReadFileFn = func(string) ([]byte, error) { return nil, os.ErrNotExist }
	hostreqkit.WriteTempFileFn = func(content string) (string, error) {
		if strings.ContainsAny(content, "*?[]") {
			t.Fatalf("darwin policy contains wildcard syntax: %s", content)
		}
		return "/tmp/onboarding-grant", nil
	}
	var calls []string
	hostreqkit.RunCommandFn = func(name string, args []string, _ hostreqkit.EnsureOptions) error {
		calls = append(calls, name+" "+strings.Join(args, " "))
		return nil
	}
	req := hostreqspec.ResolvedRequirement{
		Name: "onboarding_apply_privileges", Kind: hostreqspec.KindSafeguard, Required: true,
		Config: ConfigForRequirements("/usr/local/bin/vrooli", []string{"git"}, nil),
	}
	h := NewHandler(hostreqkit.SafeguardManifest{Name: req.Name})
	status := h.Inspect(hostreqkit.Host{OS: "darwin"}, req)
	got, err := h.Apply(hostreqkit.Host{OS: "darwin"}, status, hostreqkit.EnsureOptions{})
	if err != nil || !got.Applied || got.ExecutionState != hostreqkit.ExecutionApplied {
		t.Fatalf("darwin apply = %+v, err=%v", got, err)
	}
	if len(calls) != 3 || !strings.Contains(strings.Join(calls, "\n"), "visudo -c -f") {
		t.Fatalf("unexpected darwin command sequence: %v", calls)
	}
}

// TestUnreadableGrantIsNotReportedMissing pins the fix for a live false blocker.
// The grant lives at /etc/sudoers.d/vrooli-onboarding-apply as 0440 root:root,
// because sudo refuses to load a drop-in with looser permissions. Every
// unprivileged inspection therefore gets a permission error reading it. The
// handler used to report that as "missing or stale", which put a required
// safeguard into MissingRequired forever, so `vrooli setup status` on a
// correctly configured host reported readiness "missing" and the completion
// gate could never pass.
func TestUnreadableGrantIsNotReportedMissing(t *testing.T) {
	requirement := hostreqspec.ResolvedRequirement{
		Name:   "onboarding_apply_privileges",
		Config: ConfigForRequirements("/usr/local/bin/vrooli", []string{"git"}, []string{"clock"}),
	}
	host := hostreqkit.Host{OS: "linux"}

	original := hostreqkit.ReadFileFn
	t.Cleanup(func() { hostreqkit.ReadFileFn = original })

	t.Run("permission denied means present-but-unverified, not missing", func(t *testing.T) {
		hostreqkit.ReadFileFn = func(string) ([]byte, error) { return nil, os.ErrPermission }
		status := handler{manifest: hostreqkit.SafeguardManifest{Name: requirement.Name}}.Inspect(host, requirement)
		if !status.Applied {
			t.Fatalf("Applied = false; an unprivileged read of a root-owned grant must not read as missing")
		}
		if status.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
			t.Fatalf("ExecutionState = %q, want %q", status.ExecutionState, hostreqkit.ExecutionAlreadyPresent)
		}
		notes := strings.Join(status.Notes, " ")
		if strings.Contains(notes, "missing or stale") {
			t.Fatalf("notes still claim the grant is missing or stale: %q", notes)
		}
		if !strings.Contains(notes, "cannot be read without privilege") {
			t.Fatalf("notes must state the probe was not authoritative, got %q", notes)
		}
	})

	t.Run("absent grant still blocks", func(t *testing.T) {
		hostreqkit.ReadFileFn = func(string) ([]byte, error) { return nil, os.ErrNotExist }
		status := handler{manifest: hostreqkit.SafeguardManifest{Name: requirement.Name}}.Inspect(host, requirement)
		if status.Applied {
			t.Fatal("Applied = true for an absent grant; the real missing case must still block")
		}
		if !strings.Contains(strings.Join(status.Notes, " "), "missing or stale") {
			t.Fatalf("absent grant must keep its remediation note, got %v", status.Notes)
		}
	})

	t.Run("drifted grant still blocks", func(t *testing.T) {
		hostreqkit.ReadFileFn = func(string) ([]byte, error) { return []byte("# something else\n"), nil }
		status := handler{manifest: hostreqkit.SafeguardManifest{Name: requirement.Name}}.Inspect(host, requirement)
		if status.Applied {
			t.Fatal("Applied = true for a drifted grant; real drift must still block")
		}
	})

	t.Run("matching grant is applied with no caveat", func(t *testing.T) {
		content, err := policyContent(hostreqkit.InvokingUser(), requirement.Config)
		if err != nil {
			t.Fatal(err)
		}
		hostreqkit.ReadFileFn = func(string) ([]byte, error) { return []byte(content), nil }
		status := handler{manifest: hostreqkit.SafeguardManifest{Name: requirement.Name}}.Inspect(host, requirement)
		if !status.Applied {
			t.Fatal("Applied = false for a matching grant")
		}
		if strings.Contains(strings.Join(status.Notes, " "), "cannot be read without privilege") {
			t.Fatalf("a readable matching grant must not carry the unverified caveat: %v", status.Notes)
		}
	})
}
