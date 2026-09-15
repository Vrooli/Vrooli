//go:build !unix && !windows

package metrics

import "os"

// processPeakRSSBytes is unavailable on platforms without child rusage.
func processPeakRSSBytes(*os.ProcessState) (int64, bool) { return 0, false }
