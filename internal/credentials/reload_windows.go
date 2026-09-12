//go:build windows

package credentials

import "context"

// Windows stores credentials in the Credential Manager, backed by the Data
// Protection API and keyed to the interactive logon session. There is no
// separate daemon whose restart would clear a fault: a credential that cannot
// be read is one written under a different user or profile, and restarting a
// service cannot change that.
//
// The rung reports not-applicable rather than inventing an actuation, because a
// repair that runs a plausible-looking command on the wrong platform is worse
// than one that declines.
func platformReloadCredentialDaemon(context.Context) ReloadOutcome {
	return ReloadOutcome{
		Status: RungNotApplicable,
		Detail: "Windows serves credentials from the Credential Manager through DPAPI, bound to the interactive logon session; no credential daemon exists to restart",
	}
}
