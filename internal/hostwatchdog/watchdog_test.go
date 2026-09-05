package hostwatchdog

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestTickReportsFloorOnlyAfterSustain(t *testing.T) {
	state := filepath.Join(t.TempDir(), "state.json")
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var reports int
	cfg := Config{Mount: t.TempDir(), FloorBytes: 100, Sustain: time.Minute, StatePath: state, Now: func() time.Time { return now }, FreeSpace: func(string) (uint64, float64, error) { return 99, 95, nil }, ReportPressure: func(context.Context, Report) error { reports++; return nil }}
	first, err := Tick(context.Background(), cfg)
	if err != nil || first.Sustained {
		t.Fatalf("first tick = %+v, %v", first, err)
	}
	now = now.Add(time.Minute)
	second, err := Tick(context.Background(), cfg)
	if err != nil || !second.Sustained || reports != 1 {
		t.Fatalf("sustained tick = %+v, reports=%d, err=%v", second, reports, err)
	}
}
