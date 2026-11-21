package schedule

import (
	"testing"
	"time"
)

func TestWindowContains(t *testing.T) {
	weekdays := []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday}
	window, err := ParseWindow(weekdays, "09:00", "17:00")
	if err != nil {
		t.Fatalf("parse window: %v", err)
	}
	at := func(hour, minute int) time.Time {
		return time.Date(2026, time.November, 18, hour, minute, 0, 0, time.UTC) // Wednesday
	}
	cases := []struct {
		name string
		at   time.Time
		want bool
	}{
		{"inside", at(9, 0), true},
		{"mid-morning", at(12, 30), true},
		{"one minute before close", at(16, 59), true},
		{"at close", at(17, 0), false},
		{"before open", at(8, 59), false},
		{"after close", at(20, 0), false},
		{"weekend", time.Date(2026, time.November, 21, 10, 0, 0, 0, time.UTC), false},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if got := window.Contains(test.at, time.UTC); got != test.want {
				t.Fatalf("Contains(%s) = %v, want %v", test.at, got, test.want)
			}
		})
	}
}

func TestWindowCrossesMidnight(t *testing.T) {
	window, err := ParseWindow([]time.Weekday{time.Saturday}, "22:00", "02:00")
	if err != nil {
		t.Fatalf("parse window: %v", err)
	}
	cases := []struct {
		name string
		at   time.Time
		want bool
	}{
		{"saturday night", time.Date(2026, time.November, 21, 23, 0, 0, 0, time.UTC), true},
		{"sunday after midnight", time.Date(2026, time.November, 22, 1, 0, 0, 0, time.UTC), true},
		{"sunday just past end", time.Date(2026, time.November, 22, 2, 0, 0, 0, time.UTC), false},
		{"friday night is not saturday", time.Date(2026, time.November, 20, 23, 0, 0, 0, time.UTC), false},
		{"saturday daytime", time.Date(2026, time.November, 21, 12, 0, 0, 0, time.UTC), false},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if got := window.Contains(test.at, time.UTC); got != test.want {
				t.Fatalf("Contains(%s) = %v, want %v", test.at, got, test.want)
			}
		})
	}
}

func TestWindowNextOpen(t *testing.T) {
	window, err := ParseWindow([]time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday}, "09:00", "17:00")
	if err != nil {
		t.Fatalf("parse window: %v", err)
	}
	next, ok := window.nextOpen(time.Date(2026, time.November, 18, 18, 0, 0, 0, time.UTC), time.UTC)
	if !ok {
		t.Fatal("expected a next opening")
	}
	want := time.Date(2026, time.November, 19, 9, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Fatalf("nextOpen = %s, want %s", next, want)
	}
	fridayEvening := time.Date(2026, time.November, 20, 18, 0, 0, 0, time.UTC)
	next, ok = window.nextOpen(fridayEvening, time.UTC)
	if !ok {
		t.Fatal("expected a next opening after friday")
	}
	if next.Weekday() != time.Monday {
		t.Fatalf("nextOpen after friday = %s (%s), want Monday", next, next.Weekday())
	}
}

func TestParseWindowRejectsMalformedValues(t *testing.T) {
	cases := []struct {
		name       string
		start, end string
	}{
		{"identical", "09:00", "09:00"},
		{"missing minutes", "9", "17:00"},
		{"hour out of range", "25:01", "26:00"},
		{"garbage", "nine", "17:00"},
		{"empty", "", "17:00"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if _, err := ParseWindow(nil, test.start, test.end); err == nil {
				t.Fatalf("ParseWindow(%q, %q) unexpectedly succeeded", test.start, test.end)
			}
		})
	}
	if _, err := ParseWindow([]time.Weekday{time.Weekday(9)}, "01:00", "02:00"); err == nil {
		t.Fatal("expected an invalid weekday to be rejected")
	}
}

func TestPolicyNormalizedDefaults(t *testing.T) {
	policy, err := Policy{}.Normalized()
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if policy.Mode != ModeBalanced {
		t.Fatalf("mode = %q, want balanced", policy.Mode)
	}
	if policy.ModelDwell != DefaultModelDwell {
		t.Fatalf("model dwell = %s, want %s", policy.ModelDwell, DefaultModelDwell)
	}
	if policy.DefaultMaxBatch != DefaultMaxBatch {
		t.Fatalf("max batch = %d, want %d", policy.DefaultMaxBatch, DefaultMaxBatch)
	}
	if policy.DeferBase != DefaultDeferBase || policy.DeferCap != DefaultDeferCap {
		t.Fatalf("backoff = %s/%s, want %s/%s", policy.DeferBase, policy.DeferCap, DefaultDeferBase, DefaultDeferCap)
	}
	if policy.LeaseTTL != DefaultLeaseTTL {
		t.Fatalf("lease ttl = %s, want %s", policy.LeaseTTL, DefaultLeaseTTL)
	}
	if policy.Location != time.Local {
		t.Fatalf("location = %v, want local", policy.Location)
	}
}

func TestPolicyNormalizedRejectsInvalidValues(t *testing.T) {
	cases := map[string]Policy{
		"unknown mode":       {Mode: Mode("turbo")},
		"negative dwell":     {ModelDwell: -time.Second},
		"negative seconds":   {MaxSecondsPerHour: -1},
		"negative jobs":      {MaxJobsPerHour: -1},
		"share above one":    {MaxSharePerOwner: 1.5},
		"negative share":     {MaxSharePerOwner: -0.1},
		"negative idle load": {IdleLoadBelow: -1},
		"malformed window":   {Windows: []Window{{Start: "9", End: "17:00"}}},
		"identical window":   {Windows: []Window{{Start: "09:00", End: "09:00"}}},
	}
	for name, policy := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := policy.Normalized(); err == nil {
				t.Fatal("expected policy to be rejected")
			}
		})
	}
}

func TestPolicyBackoffGrowsAndCaps(t *testing.T) {
	policy, err := Policy{DeferBase: 30 * time.Second, DeferCap: 5 * time.Minute}.Normalized()
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	cases := []struct {
		deferrals int
		want      time.Duration
	}{
		{1, 30 * time.Second},
		{2, 60 * time.Second},
		{3, 120 * time.Second},
		{4, 240 * time.Second},
		{5, 5 * time.Minute},
		{9, 5 * time.Minute},
	}
	for _, test := range cases {
		if got := policy.backoff(test.deferrals); got != test.want {
			t.Fatalf("backoff(%d) = %s, want %s", test.deferrals, got, test.want)
		}
	}
}

func TestPolicyIdleObservation(t *testing.T) {
	policy, err := Policy{IdleLoadBelow: 1.5, IdleRAMFreeBytes: 4 << 30}.Normalized()
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	cases := []struct {
		name  string
		state *HostState
		want  bool
	}{
		{"nil observation", nil, false},
		{"quiet host", &HostState{Load1: 0.3, AvailableRAMBytes: 8 << 30}, true},
		{"busy load", &HostState{Load1: 2.0, AvailableRAMBytes: 8 << 30}, false},
		{"low memory", &HostState{Load1: 0.3, AvailableRAMBytes: 2 << 30}, false},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if got := policy.idleObservation(test.state); got != test.want {
				t.Fatalf("idleObservation = %v, want %v", got, test.want)
			}
		})
	}
	unconstrained, err := Policy{}.Normalized()
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	if !unconstrained.idleObservation(&HostState{Load1: 99, AvailableRAMBytes: 0}) {
		t.Fatal("an observation without thresholds must be idle")
	}
}

func TestParseMode(t *testing.T) {
	cases := map[string]Mode{
		"":         ModeBalanced,
		"off":      ModeOff,
		" passive": ModePassive,
		"IDLE":     ModeIdle,
		"balanced": ModeBalanced,
		"eager":    ModeEager,
	}
	for raw, want := range cases {
		got, err := ParseMode(raw)
		if err != nil {
			t.Fatalf("ParseMode(%q): %v", raw, err)
		}
		if got != want {
			t.Fatalf("ParseMode(%q) = %q, want %q", raw, got, want)
		}
	}
	if _, err := ParseMode("turbo"); err == nil {
		t.Fatal("expected an unknown mode to be rejected")
	}
}

func TestPolicyMaxBatchClamps(t *testing.T) {
	policy, err := Policy{DefaultMaxBatch: 8}.Normalized()
	if err != nil {
		t.Fatalf("normalize: %v", err)
	}
	jobs := []Job{{MaxBatch: 3}, {MaxBatch: 20}, {MaxBatch: 0}}
	if got := policy.maxBatchFor(jobs); got != 3 {
		t.Fatalf("maxBatchFor = %d, want 3", got)
	}
	if got := policy.maxBatchFor([]Job{{MaxBatch: 20}}); got != 8 {
		t.Fatalf("maxBatchFor = %d, want the policy cap 8", got)
	}
	if got := policy.maxBatchFor(nil); got != 8 {
		t.Fatalf("maxBatchFor(nil) = %d, want 8", got)
	}
}
