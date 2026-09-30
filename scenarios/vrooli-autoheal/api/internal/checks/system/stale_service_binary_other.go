//go:build !linux

package system

import "context"

// linuxProcResolver has no non-Linux implementation. macOS and Windows expose
// no equivalent of /proc/<pid>/exe's "(deleted)" marker, so the check reports
// NotApplicable rather than guessing from binary timestamps — an mtime
// comparison would fire on every rebuild whether or not the process was stale.
type linuxProcResolver struct{}

func (linuxProcResolver) Resolve(context.Context, string) (string, bool, bool) {
	return "", false, false
}

func restartUserUnit(context.Context, string) (string, error) {
	return "", nil
}
