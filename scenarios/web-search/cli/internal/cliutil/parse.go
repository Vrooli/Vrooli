// Package cliutil holds small parsing helpers shared by the CLI domains.
package cliutil

import (
	"errors"
	"strconv"
	"strings"
)

// ParseInt32 parses a flag value into an int32 suitable for limit/count
// fields. Invalid input parses to 0 (the "unset" default), and out-of-range
// values are clamped to [0, math.MaxInt32] so a huge or negative input can
// never wrap around the int32 conversion. ParseInt with bitSize 32 makes the
// conversion provably range-safe: on overflow it returns ErrRange together
// with the nearest representable value, which is exactly the clamp we want.
func ParseInt32(s string) int32 {
	v, err := strconv.ParseInt(strings.TrimSpace(s), 10, 32)
	if err != nil && !errors.Is(err, strconv.ErrRange) {
		return 0
	}
	if v < 0 {
		return 0
	}
	return int32(v)
}
