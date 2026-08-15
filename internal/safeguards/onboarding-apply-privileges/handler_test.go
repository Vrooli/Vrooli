package onboardingapplyprivileges

import (
	"os"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
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
	t.Setenv("USER", "operator")
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
