//go:build windows

package ollamaresourcecontrols

import "golang.org/x/sys/windows"

// processAlivePID opens the process with query-only access. It never sends a
// signal or changes process state.
func processAlivePID(pid int) bool {
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	_ = windows.CloseHandle(handle)
	return true
}
