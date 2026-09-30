package credentials

import "context"

// ReloadOutcome is what a daemon-reload attempt did, in terms the ladder can
// render without knowing which platform produced it.
//
// It carries Action — the command actually executed — because a repair that
// mutates an operator's host and does not say how is indistinguishable from one
// that guessed.
type ReloadOutcome struct {
	Status string
	Detail string
	Action string
	// Remedy is what the operator must do when Status is RungBlocked. It is
	// empty for every other status. A blocked rung with no remedy is a dead end
	// and this package treats producing one as a defect.
	Remedy []string
}

// reloadCredentialDaemon restarts the platform's credential-store service so it
// re-reads its on-disk state and drops a wedged session.
//
// It is implemented per platform. Every implementation must probe for the tools
// and units it needs rather than assuming them: a Linux host may have no
// systemd, no user manager, or no GNOME Keyring unit, and none of those is an
// error — they are different hosts, and the ladder reports them as such.
//
// An implementation must never return RungRepaired on the strength of having
// run a command. The caller re-probes the live store afterwards and that probe,
// not the exit code, decides whether anything was fixed.
var reloadCredentialDaemon = func(ctx context.Context) ReloadOutcome {
	return platformReloadCredentialDaemon(ctx)
}

// relogRemedy is the terminal rung of the whole ladder, and the only honest
// answer for a class of faults: re-authenticating a desktop session needs a
// credential no automated process holds or should hold.
//
// It is stated explicitly rather than omitted. A repair command that ends with
// no remedy reads as "nothing more can be done and nobody knows why"; this
// says which wall was hit and that the wall is deliberate.
func relogRemedy(cause string) []string {
	return []string{
		cause + ".",
		"Log out and back in (or reboot). A fresh login restarts the credential daemon and, on a passphrase-protected keyring, unlocks it through PAM.",
		"There is no Vrooli command for this rung: re-authenticating a desktop session requires a credential no automated process holds.",
	}
}
