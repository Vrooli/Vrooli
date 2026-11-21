//go:build !(linux && (amd64 || arm64))

package agentharness

import "time"

// birthTime is unknown off Linux, so the birth-time resolver never vouches.
func birthTime(string) (time.Time, bool) { return time.Time{}, false }
