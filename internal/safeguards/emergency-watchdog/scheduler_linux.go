//go:build linux

package emergencywatchdog

import "github.com/vrooli/vrooli/internal/hostreqkit"

func nativeSchedulerAvailable(string) bool { return false }
func nativePending(string, paths) []string { return nil }
func applyNative(_ string, _ paths, status hostreqkit.ItemStatus, _ hostreqkit.EnsureOptions) (hostreqkit.ItemStatus, error) {
	status.ExecutionState = hostreqkit.ExecutionUnsupported
	return status, nil
}
