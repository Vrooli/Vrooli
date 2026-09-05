package sidecar

import (
	"strings"
	"testing"

	"github.com/vrooli/browser-automation-studio/automation/driver"
)

func TestRecoveryAdminSecretUsesConfiguredValueOrGeneratesOne(t *testing.T) {
	t.Setenv(driver.PlaywrightDriverAdminSecretEnv, "configured-secret")
	configured, err := recoveryAdminSecret()
	if err != nil {
		t.Fatalf("configured recoveryAdminSecret: %v", err)
	}
	if configured != "configured-secret" {
		t.Fatalf("configured secret = %q", configured)
	}

	t.Setenv(driver.PlaywrightDriverAdminSecretEnv, "")
	generated, err := recoveryAdminSecret()
	if err != nil {
		t.Fatalf("generated recoveryAdminSecret: %v", err)
	}
	if len(generated) < 32 || strings.ContainsAny(generated, " \t\r\n") {
		t.Fatalf("generated secret is not a usable high-entropy token: %q", generated)
	}
}
