//go:build darwin

package credentials

import "context"

// macOS has no user-serviceable credential daemon. The Keychain is served by
// securityd, which launchd owns and which an operator neither restarts nor
// should: a wedged Keychain is a locked Keychain, and the remedy is to unlock
// it, not to bounce the service.
//
// This rung therefore reports not-applicable with the reason, and lets the
// unlock rung carry the real remedy. Reporting a Linux-shaped failure here
// would tell a macOS operator to fix something their host does not have.
func platformReloadCredentialDaemon(context.Context) ReloadOutcome {
	return ReloadOutcome{
		Status: RungNotApplicable,
		Detail: "macOS serves credentials from securityd under launchd; there is no operator-restartable credential daemon, and a stuck Keychain is resolved by unlocking it",
	}
}
