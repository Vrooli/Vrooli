//go:build !linux

package privilegebroker

import "context"

// Host power actions are Linux/systemd only. The non-Linux broker binary exists
// so the package and its callers build everywhere; it reports the capability as
// unsupported rather than pretending a host shutdown happened.
func executeSystemReboot(_ context.Context, _ Executor, req Request) Result {
	return NewFailure(req.RequestID, req.Action, "reboot_unsupported")
}

func executeSystemRebootCancel(_ context.Context, _ Executor, req Request) Result {
	return NewFailure(req.RequestID, req.Action, "reboot_unsupported")
}
