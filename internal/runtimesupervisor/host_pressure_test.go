//nolint:goconst // test data deliberately reuses stable platform fixtures.
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

// A host whose run queue is deep enough to stall most work cannot act on a
// restart, whatever its memory looks like. This is the 2026-08-19 shape: load
// 110 on 32 CPUs with memory at 40% used, so memory PSI never tripped.
func TestHostPressureProviderDetectsCPUSaturationWithHealthyMemory(t *testing.T) {
	files := map[string]string{
		"/proc/pressure/memory": "some avg10=0.30 avg60=0.20 avg300=0.10 total=2\nfull avg10=0.00 avg60=0.00 avg300=0.00 total=0\n",
		"/proc/pressure/cpu":    "some avg10=87.40 avg60=71.20 avg300=44.10 total=9\nfull avg10=0.00 avg60=0.00 avg300=0.00 total=0\n",
		"/proc/vmstat":          "oom_kill 4\n",
	}
	p := NewHostPressureProviderWithCPU(10, 50)
	p.goos = "linux"
	p.readFile = func(name string) ([]byte, error) { return []byte(files[name]), nil }

	got := p.Snapshot(context.Background())

	if !got.Known {
		t.Fatalf("Snapshot() = %#v, want known", got)
	}
	if !got.UnderPressure {
		t.Fatalf("CPU saturation must count as pressure; got %#v", got)
	}
	if !strings.Contains(got.Reason, "cpu saturated") {
		t.Errorf("reason should name the tripped signal, got %q", got.Reason)
	}
}

// Ordinary contention on a build host must not gate recovery — otherwise the
// gate is on permanently and restarts never happen.
func TestHostPressureProviderToleratesOrdinaryCPUContention(t *testing.T) {
	files := map[string]string{
		"/proc/pressure/memory": "some avg10=0.30 avg60=0.20 avg300=0.10 total=2\n",
		"/proc/pressure/cpu":    "some avg10=12.00 avg60=9.00 avg300=7.00 total=9\n",
		"/proc/vmstat":          "oom_kill 4\n",
	}
	p := NewHostPressureProviderWithCPU(10, 50)
	p.goos = "linux"
	p.readFile = func(name string) ([]byte, error) { return []byte(files[name]), nil }

	if got := p.Snapshot(context.Background()); got.UnderPressure {
		t.Fatalf("moderate CPU contention must not gate recovery; got %#v", got)
	}
}

// An unreadable CPU PSI file degrades that one signal. Memory evidence is still
// actionable, so the snapshot must stay known rather than disabling gating.
func TestHostPressureProviderDegradesGracefullyWithoutCPUPSI(t *testing.T) {
	files := map[string]string{
		"/proc/pressure/memory": "some avg10=42.00 avg60=30.00 avg300=10.00 total=2\n",
		"/proc/vmstat":          "oom_kill 4\n",
	}
	p := NewHostPressureProviderWithCPU(10, 50)
	p.goos = "linux"
	p.readFile = func(name string) ([]byte, error) {
		body, ok := files[name]
		if !ok {
			return nil, errors.New("no such file")
		}
		return []byte(body), nil
	}

	got := p.Snapshot(context.Background())

	if !got.Known {
		t.Fatalf("missing CPU PSI must not make the whole snapshot unknown; got %#v", got)
	}
	if !got.UnderPressure {
		t.Errorf("memory pressure should still register; got %#v", got)
	}
	if !strings.Contains(got.Reason, "cpu PSI unavailable") {
		t.Errorf("reason should state the missing signal, got %q", got.Reason)
	}
}

// The default constructor must carry the CPU threshold, so production wiring
// that calls NewHostPressureProvider still gets the new signal.
func TestDefaultConstructorEnablesCPUSignal(t *testing.T) {
	p := NewHostPressureProvider(10)
	if p.cpuSomeAvg10Threshold != DefaultPressureCPUSomeAvg10 {
		t.Fatalf("cpu threshold = %v, want %v", p.cpuSomeAvg10Threshold, DefaultPressureCPUSomeAvg10)
	}
}
