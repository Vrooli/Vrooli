package onboard

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	onboardv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/onboard"
)

// allFailureCodes is the taxonomy the orchestrator can record on a FAILED op
// (mirrors api/internal/onboard/types.go). Every one must render its own
// distinct, actionable guidance line.
var allFailureCodes = []string{
	failSSHSetup,
	failScriptPush,
	failPairingIssue,
	failBootstrapUsage,
	failUnsupportedPlatform,
	failPairing,
	failBootstrap,
	failVerifyOnline,
	failInterrupted,
	failInternal,
}

func TestFailureGuidance_EveryCodeIsDistinctAndActionable(t *testing.T) {
	seen := map[string]string{}
	generic := failureGuidance(&onboardv1.OnboardingOp{FailureReason: "totally-unknown-code"})
	for _, code := range allFailureCodes {
		msg := failureGuidance(&onboardv1.OnboardingOp{FailureReason: code, Host: "h", User: "root"})
		require.NotEmpty(t, msg, "code %q produced no guidance", code)
		// Distinct from the generic fallback (i.e. actually mapped).
		require.NotEqual(t, generic, msg, "code %q fell through to the generic message", code)
		// No two codes share a message.
		if prev, ok := seen[msg]; ok {
			t.Fatalf("codes %q and %q share the same message", prev, code)
		}
		seen[msg] = code
	}
	require.Len(t, seen, len(allFailureCodes))
}

func TestFailureGuidance_NamesTheTargetAndNextStep(t *testing.T) {
	msg := failureGuidance(&onboardv1.OnboardingOp{FailureReason: failSSHSetup, Host: "10.0.0.5", User: "deploy"})
	require.Contains(t, msg, "deploy@10.0.0.5")
	require.Contains(t, msg, "onboard start") // every retryable failure points at the retry
}

func TestFailureGuidance_UnsupportedPlatformMentionsMacGate(t *testing.T) {
	msg := failureGuidance(&onboardv1.OnboardingOp{FailureReason: failUnsupportedPlatform, Host: "mac", User: "admin"})
	require.True(t, strings.Contains(strings.ToLower(msg), "macos"), "unsupported-platform guidance should name the macOS gate: %q", msg)
}

func TestFailureGuidance_EmptyReasonIsHonest(t *testing.T) {
	msg := failureGuidance(&onboardv1.OnboardingOp{})
	require.Contains(t, msg, "Onboarding failed")
	require.NotContains(t, msg, "()") // no dangling empty-code parenthetical
}

func TestDiagnosticsBlock_FramesNodeOutputOnlyWhenPresent(t *testing.T) {
	// No node output → no block (control-plane-side failures produce none).
	require.Empty(t, diagnosticsBlock(&onboardv1.OnboardingOp{FailureReason: failSSHSetup}))
	require.Empty(t, diagnosticsBlock(nil))

	op := &onboardv1.OnboardingOp{
		FailureReason: failBootstrap,
		FailureDetail: "make[1]: *** [setup] Error 2\nvrooli setup failed: postgres not reachable",
	}
	lines := diagnosticsBlock(op)
	require.NotEmpty(t, lines)
	joined := strings.Join(lines, "\n")
	// The concrete cause is present, each output line indented under a frame.
	require.Contains(t, joined, "node output")
	require.Contains(t, joined, "  make[1]: *** [setup] Error 2")
	require.Contains(t, joined, "  vrooli setup failed: postgres not reachable")
}

func TestTerminalSummary_FailedOpShowsNodeOutputBeforeGuidance(t *testing.T) {
	op := &onboardv1.OnboardingOp{
		Id:            "op-1",
		Host:          "h",
		User:          "root",
		ExitCode:      1,
		State:         onboardv1.OnboardingState_ONBOARDING_STATE_FAILED,
		FailureReason: failBootstrap,
		FailureDetail: "FINAL ERROR: make setup failed",
	}
	joined := strings.Join(terminalSummaryLines(op), "\n")
	require.Contains(t, joined, "FINAL ERROR: make setup failed", "the operator must see the concrete node error, not just the reason code")
	// The concrete output appears above the taxonomy guidance.
	require.Less(t, strings.Index(joined, "FINAL ERROR"), strings.Index(joined, "idempotent"))
}
