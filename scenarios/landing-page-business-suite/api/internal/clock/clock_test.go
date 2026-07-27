package clock

import (
	"testing"
	"time"
)

func TestSystemNowReturnsCurrentWallClockTime(t *testing.T) {
	before := time.Now().Add(-time.Second)
	now := System{}.Now()
	after := time.Now().Add(time.Second)
	if now.Before(before) || now.After(after) {
		t.Fatalf("System.Now() = %s, outside [%s, %s]", now, before, after)
	}
}
