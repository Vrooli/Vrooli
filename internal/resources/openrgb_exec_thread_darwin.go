//go:build darwin

package resources

import "runtime"

// The native manager sets the non-login account before the trusted wrapper
// starts. Context observation verifies that exact account and containment;
// Linux no-new-privileges is not invented on Darwin.
func confineOpenRGBExecThread() (func(), error) {
	runtime.LockOSThread()
	return runtime.UnlockOSThread, nil
}
