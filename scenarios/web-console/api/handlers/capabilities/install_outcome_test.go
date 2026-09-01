package capabilities

import (
	"context"
	"testing"

	internalcaps "web-console/internal/capabilities"
)

// An install used to be reported as successful because the installer exited 0.
// It is reported as successful only when the machine says the capability is
// there, because those are different claims and the UI renders a completed
// install from this one field.
func TestStampInstallOutcomeRequiresTheTargetToConfirm(t *testing.T) {
	adapter := &Adapter{ConfirmInstall: func(context.Context, string, string) (string, string) {
		return "ready", "1.2.3"
	}}
	result := adapter.stampInstallOutcome(context.Background(), "bridge-node:mac", internalcaps.LifecycleActionResult{
		Success: true, CapabilityID: "codex", Status: "RELAY_CALL_OUTCOME_COMPLETED",
	})
	if !result.Success || result.Status != internalcaps.InstallStatusInstalled {
		t.Fatalf("result = %+v, want a confirmed install", result)
	}
	if result.Message != "Installed (1.2.3)." {
		t.Fatalf("message = %q", result.Message)
	}
}

// The exact shape the operator hit: the relay completed, the machine still
// does not report the agent, and the card announced "Installed" anyway.
func TestStampInstallOutcomeReportsUnconfirmedWhenTheTargetStillLacksTheCapability(t *testing.T) {
	adapter := &Adapter{ConfirmInstall: func(context.Context, string, string) (string, string) {
		return "missing", ""
	}}
	result := adapter.stampInstallOutcome(context.Background(), "bridge-node:mac", internalcaps.LifecycleActionResult{
		Success: true, CapabilityID: "codex",
	})
	if result.Success {
		t.Fatal("Success = true for an install the machine never confirmed")
	}
	if result.Status != internalcaps.InstallStatusUnconfirmed {
		t.Fatalf("status = %q, want %q", result.Status, internalcaps.InstallStatusUnconfirmed)
	}
	if result.Message == "" {
		t.Fatal("an unconfirmed install must say what is unknown")
	}
}

// Unconfirmed is not failed. A failing installer keeps its own message,
// because that message is the only thing that says what went wrong.
func TestStampInstallOutcomeKeepsTheInstallerFailureMessage(t *testing.T) {
	adapter := &Adapter{ConfirmInstall: func(context.Context, string, string) (string, string) {
		t.Fatal("confirmation must not run for a failed install")
		return "", ""
	}}
	result := adapter.stampInstallOutcome(context.Background(), "bridge-node:mac", internalcaps.LifecycleActionResult{
		Success: false, CapabilityID: "codex", Message: "npm ERR! code EACCES",
	})
	if result.Status != internalcaps.InstallStatusFailed || result.Message != "npm ERR! code EACCES" {
		t.Fatalf("result = %+v", result)
	}
}

// A capability that cannot exist on the target is already a final answer;
// re-deriving it from a probe that will never report it would turn a clear
// "not supported here" into an ambiguous "unconfirmed".
func TestStampInstallOutcomeLeavesNotApplicableAlone(t *testing.T) {
	adapter := &Adapter{}
	result := adapter.stampInstallOutcome(context.Background(), "bridge-node:mac", internalcaps.LifecycleActionResult{
		CapabilityID: "codex", Status: internalcaps.InstallStatusNotApplicable, Message: "windows/arm64 is not published",
	})
	if result.Status != internalcaps.InstallStatusNotApplicable || result.Message != "windows/arm64 is not published" {
		t.Fatalf("result = %+v", result)
	}
}

// With no confirmation seam wired, the honest answer is "unknown". Defaulting
// to success here is exactly the bug, one wiring mistake away.
func TestStampInstallOutcomeWithoutAConfirmerNeverClaimsSuccess(t *testing.T) {
	adapter := &Adapter{}
	result := adapter.stampInstallOutcome(context.Background(), "bridge-node:mac", internalcaps.LifecycleActionResult{
		Success: true, CapabilityID: "codex",
	})
	if result.Success || result.Status != internalcaps.InstallStatusUnconfirmed {
		t.Fatalf("result = %+v", result)
	}
}
