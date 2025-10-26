package timeutil

import "time"

// NowRFC3339 returns the current time formatted as RFC3339.
// This helper ensures consistent timestamp formatting across the application.
func NowRFC3339() string {
	return time.Now().Format(time.RFC3339)
}
