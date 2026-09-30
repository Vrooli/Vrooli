//go:build windows

package runtime

import (
	"os"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
)

func TestDetectWindowsPackageManagerOrder(t *testing.T) {
	original := hostreqkit.LookPathFn
	t.Cleanup(func() { hostreqkit.LookPathFn = original })

	for _, want := range []string{"winget", "choco", "scoop", ""} {
		t.Run(want, func(t *testing.T) {
			hostreqkit.LookPathFn = func(name string) (string, error) {
				if name == want {
					return name, nil
				}
				return "", os.ErrNotExist
			}
			if got := detectWindowsPackageManager(); got != want {
				t.Fatalf("detectWindowsPackageManager() = %q, want %q", got, want)
			}
		})
	}
}

func TestWindowsHostSupportsProvisioningAndDevelopment(t *testing.T) {
	hostreqkit.LookPathFn = func(string) (string, error) { return "", os.ErrNotExist }
	host := currentHost()
	if !host.SupportsSetup || !host.SupportsDevelop {
		t.Fatalf("host support = setup %t develop %t", host.SupportsSetup, host.SupportsDevelop)
	}
	if err := host.ValidateSetup(); err != nil {
		t.Fatalf("ValidateSetup: %v", err)
	}
	if err := host.ValidateDevelop(); err != nil {
		t.Fatalf("ValidateDevelop: %v", err)
	}
}
