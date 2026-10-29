package orchestration

import "testing"

func TestFormatWallClockIsHumanReadable(t *testing.T) {
	for ms, want := range map[int64]string{0: "0s", 450: "450ms", 1499: "1s", 90500: "1m31s", 3600000: "1h0m0s"} {
		if got := formatWallClock(ms); got != want {
			t.Errorf("formatWallClock(%d) = %q, want %q", ms, got, want)
		}
	}
}
