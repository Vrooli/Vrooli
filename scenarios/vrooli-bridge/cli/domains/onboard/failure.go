package onboard

import (
	"fmt"

	onboardv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/onboard"
)

// Failure taxonomy machine codes, mirroring the API's onboard.FailureReason
// wire values (api/internal/onboard/types.go). The CLI cannot import the API's
// internal package (separate module), so these stable strings are duplicated
// here as the branch keys — a drift-catching test asserts each renders a
// distinct, non-generic message.
const (
	failSSHSetup            = "ssh_setup_failed"
	failScriptPush          = "script_push_failed"
	failPairingIssue        = "pairing_issue_failed"
	failBootstrapUsage      = "bootstrap_usage_error"
	failUnsupportedPlatform = "unsupported_platform"
	failPairing             = "pairing_failed"
	failBootstrap           = "bootstrap_failed"
	failVerifyOnline        = "verify_online_failed"
	failInterrupted         = "interrupted_by_restart"
	failInternal            = "internal_error"
)

// failureGuidance maps a terminal FAILED op's taxonomy code to a distinct,
// operator-actionable message: what went wrong and the concrete next step. The
// op is passed so the message can name the SSH target and (for retries) point
// back at `onboard start`. Every code the orchestrator can record maps to its
// own line; an unrecognised or empty code falls back to a generic-but-honest
// message rather than pretending to know the cause.
func failureGuidance(op *onboardv1.OnboardingOp) string {
	reason := ""
	target := "the host"
	if op != nil {
		reason = op.FailureReason
		if op.Host != "" {
			user := op.User
			if user == "" {
				user = "root"
			}
			target = fmt.Sprintf("%s@%s", user, op.Host)
		}
	}
	switch reason {
	case failSSHSetup:
		return fmt.Sprintf("SSH setup failed: could not establish passwordless SSH to %s. "+
			"Confirm the host is reachable on the SSH port, the user is correct, and the SSH password was right, then re-run `onboard start` "+
			"(supply the password via --password-stdin, --prompt-password, $%s, or the UI onboard form).", target, sshPasswordEnvVar)
	case failScriptPush:
		return fmt.Sprintf("Could not copy the bootstrap script to %s over SSH. "+
			"Check remote disk space and write access to the temp dir, then re-run `onboard start`.", target)
	case failPairingIssue:
		return "The control plane could not issue a pairing code. " +
			"Check the control-plane owner credentials and health, then re-run `onboard start`."
	case failBootstrapUsage:
		return "The bootstrap script rejected its arguments (exit 2). This is a control-plane defect in how the op was built, " +
			"not a host problem — capture the op id and file a bug (report-bug → scenario-qa)."
	case failUnsupportedPlatform:
		return fmt.Sprintf("%s runs a platform that this Bridge build does not support (exit 3). "+
			"Use a supported target or update Bridge when support for that platform is available.", target)
	case failPairing:
		return "Pairing was rejected on the node (exit 4): the single-use code was already consumed or expired. " +
			"Re-run `onboard start` to reissue a fresh code."
	case failBootstrap:
		return "The bootstrap script failed on the node (exit 1). Inspect the failing step above, fix the node condition, " +
			"then re-run `onboard start` — every step is idempotent, so a re-run converges."
	case failVerifyOnline:
		return "The node bootstrapped but did not come ONLINE within the verification budget. " +
			"Check the node-agent service (autostart) and its dial-out path to the control plane, then re-run `onboard start` or raise --verify-timeout."
	case failInterrupted:
		return "The control plane restarted mid-onboarding. The op is safe to retry — re-run `onboard start` (every step is idempotent)."
	case failInternal:
		return "An unexpected control-plane error ended the op. Capture the op id and control-plane logs, then re-run `onboard start`."
	default:
		if reason != "" {
			return fmt.Sprintf("Onboarding failed (%s). Inspect the step history above, then re-run `onboard start`.", reason)
		}
		return "Onboarding failed. Inspect the step history above, then re-run `onboard start`."
	}
}
