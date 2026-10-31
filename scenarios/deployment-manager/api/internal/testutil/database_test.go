package testutil

import (
	"testing"
	"time"
)

func TestOpenSQLiteAndClock(t *testing.T) {
	db := OpenSQLite(t)
	if err := db.Ping(); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, time.August, 4, 20, 0, 0, 0, time.UTC)
	if got := (Clock{NowValue: now}).Now(); !got.Equal(now) {
		t.Fatalf("clock = %v", got)
	}
}
