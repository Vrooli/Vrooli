//nolint:goconst // test data deliberately reuses stable device fixtures.
package tpmcredentialaccess

import (
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/testenv"
)

func tpmRequirement() hostreqspec.ResolvedRequirement {
	return hostreqspec.ResolvedRequirement{Name: "tpm_credential_access", Kind: hostreqspec.KindSafeguard}
}

func TestTPMAccessDecisionMatrix(t *testing.T) {
	originalDevice, originalGroup := grantableDeviceFn, accountInGroupFn
	t.Cleanup(func() { grantableDeviceFn, accountInGroupFn = originalDevice, originalGroup })
	h := NewHandler(hostreqkit.SafeguardManifest{Name: "tpm_credential_access"})

	t.Run("no device is not applicable", func(t *testing.T) {
		grantableDeviceFn = func() (string, string, bool) { return "", "", false }
		status := h.Inspect(hostreqkit.Host{OS: "linux"}, tpmRequirement())
		if status.SupportClass != hostreqkit.SupportNotApplicable {
			t.Fatalf("status = %+v, want not applicable", status)
		}
	})

	t.Run("missing account grant is pending", func(t *testing.T) {
		grantableDeviceFn = func() (string, string, bool) { return "/dev/tpmrm0", "tss", true }
		accountInGroupFn = func(string, string) (bool, error) { return false, nil }
		status := h.Inspect(hostreqkit.Host{OS: "linux"}, tpmRequirement())
		if status.Applied || status.ExecutionState != hostreqkit.ExecutionPending {
			t.Fatalf("status = %+v, want pending and unapplied", status)
		}
	})

	t.Run("existing account grant is already present", func(t *testing.T) {
		grantableDeviceFn = func() (string, string, bool) { return "/dev/tpmrm0", "tss", true }
		accountInGroupFn = func(string, string) (bool, error) { return true, nil }
		status := h.Inspect(hostreqkit.Host{OS: "linux"}, tpmRequirement())
		if !status.Applied || status.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
			t.Fatalf("status = %+v, want already present", status)
		}
	})
}

func TestTPMAccessApplyUsesPrivilegedGroupGrant(t *testing.T) {
	testenv.AsCurrentUser(t, "fixture-operator")
	originalDevice, originalGroup, originalRun := grantableDeviceFn, accountInGroupFn, hostreqkit.RunCommandFn
	t.Cleanup(func() {
		grantableDeviceFn, accountInGroupFn, hostreqkit.RunCommandFn = originalDevice, originalGroup, originalRun
	})
	grantableDeviceFn = func() (string, string, bool) { return "/dev/tpmrm0", "tss", true }
	accountInGroupFn = func(string, string) (bool, error) { return false, nil }
	var command string
	var args []string
	hostreqkit.RunCommandFn = func(name string, got []string, _ hostreqkit.EnsureOptions) error {
		command, args = name, append([]string(nil), got...)
		return nil
	}
	h := NewHandler(hostreqkit.SafeguardManifest{Name: "tpm_credential_access"})
	status := h.Inspect(hostreqkit.Host{OS: "linux"}, tpmRequirement())
	if _, err := h.Apply(hostreqkit.Host{OS: "linux"}, status, hostreqkit.EnsureOptions{SudoMode: "ask"}); err != nil {
		t.Fatal(err)
	}
	want := "usermod -aG tss " + hostreqkit.InvokingUser()
	if command != "sudo" || len(args) != 4 || strings.Join(args, " ") != want {
		t.Fatalf("grant command = %q %v, want sudo %s", command, args, want)
	}
}
