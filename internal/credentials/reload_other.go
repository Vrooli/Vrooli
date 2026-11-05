//go:build !linux && !darwin && !windows

package credentials

import (
	"context"
	"runtime"
)

// A platform with no adapter in this package reports that fact rather than
// falling through to a default. "Vrooli has not been taught this host" is a
// true and useful answer; silently returning healthy is neither.
func platformReloadCredentialDaemon(context.Context) ReloadOutcome {
	return ReloadOutcome{
		Status: RungNotApplicable,
		Detail: "no credential-daemon reload is implemented for " + runtime.GOOS + "; Vrooli has no host facts for this platform's credential store",
	}
}
