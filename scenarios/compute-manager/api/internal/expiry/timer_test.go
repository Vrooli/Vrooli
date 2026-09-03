package expiry

import (
	"strings"
	"testing"
	"time"
)

func TestRenderTimerCarriesExactExpiry(t *testing.T) {
	expiry := time.Date(2026, 9, 3, 21, 0, 0, 0, time.UTC)
	if got := RenderTimer(expiry); !strings.Contains(got, "2026-09-03 21:00:00 UTC") {
		t.Fatalf("timer = %q, missing expiry", got)
	}
}
