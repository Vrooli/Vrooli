package transcriptindex

import "time"

// SetRepairRetryDelayForTest shortens the busy-retry delay; it returns the
// restore function.
func SetRepairRetryDelayForTest(delay time.Duration) func() {
	previous := repairRetryDelay
	repairRetryDelay = delay
	return func() { repairRetryDelay = previous }
}
