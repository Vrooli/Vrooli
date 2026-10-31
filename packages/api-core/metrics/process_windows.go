//go:build windows

package metrics

import "os"

// processPeakRSSBytes is unavailable on Windows: ProcessState.SysUsage has
// only creation, exit, kernel, and user Filetime values, not child memory.
func processPeakRSSBytes(*os.ProcessState) (int64, bool) { return 0, false }
