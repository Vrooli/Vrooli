package runtimesupervisor

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestHostPressureProviderFailsClosedWhenEvidenceIsUnavailable(t *testing.T) {
	p := NewHostPressureProvider(10)
	p.goos = "linux"
	p.readFile = func(string) ([]byte, error) { return nil, errors.New("unavailable") }
	if got := p.Snapshot(context.Background()); got.Known {
		t.Fatalf("Snapshot() = %#v, want unknown", got)
	}
}

func TestHostPressureProviderDetectsPSIAndOOMAdvance(t *testing.T) {
	files := map[string]string{
		"/proc/pressure/memory": "some avg10=12.50 avg60=1.00 avg300=0.50 total=2\nfull avg10=0.00 avg60=0.00 avg300=0.00 total=0\n",
		"/proc/vmstat":          "oom_kill 4\n",
	}
	p := NewHostPressureProvider(10)
	p.goos = "linux"
	p.now = func() time.Time { return time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC) }
	p.readFile = func(name string) ([]byte, error) { return []byte(files[name]), nil }
	first := p.Snapshot(context.Background())
	if !first.Known || !first.UnderPressure || !strings.Contains(first.Reason, "some.avg10=12.50") {
		t.Fatalf("first Snapshot() = %#v", first)
	}
	files["/proc/pressure/memory"] = "some avg10=0.00 avg60=0.00 avg300=0.00 total=2\n"
	files["/proc/vmstat"] = "oom_kill 5\n"
	second := p.Snapshot(context.Background())
	if !second.UnderPressure || !strings.Contains(second.Reason, "advanced") {
		t.Fatalf("second Snapshot() = %#v, want OOM advance pressure", second)
	}
}
