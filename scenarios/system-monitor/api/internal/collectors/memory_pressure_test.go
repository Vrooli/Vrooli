package collectors

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/darwinsys"
)

const testPageSize = 4096

// fixtureFS serves /proc paths from testdata files, swappable between samples.
type fixtureFS struct {
	files map[string]string // proc path -> testdata file ("" = absent)
}

func (f *fixtureFS) read(path string) ([]byte, error) {
	name, ok := f.files[path]
	if !ok || name == "" {
		return nil, os.ErrNotExist
	}
	return os.ReadFile("testdata/" + name)
}

type stepClock struct{ now time.Time }

func (c *stepClock) Now() time.Time { return c.now }

func newLinuxPressureCollector(fs *fixtureFS, clock *stepClock) *PressureCollector {
	c := NewPressureCollector()
	c.platform = "linux"
	c.readFile = fs.read
	c.readPaging = func(vmstat map[string]uint64) pagingReading { return linuxPagingReading(vmstat, testPageSize) }
	c.now = clock.Now
	return c
}

func readingOf(t *testing.T, data *MetricData) MemoryPressureReading {
	t.Helper()
	reading, ok := data.Values["memory_pressure"].(MemoryPressureReading)
	if !ok {
		t.Fatalf("memory_pressure payload missing: %#v", data.Values["memory_pressure"])
	}
	return reading
}

func assertEverySignalHasState(t *testing.T, reading MemoryPressureReading) {
	t.Helper()
	for _, name := range MemoryPressureSignalNames {
		signal, ok := reading.Signals[name]
		if !ok {
			t.Errorf("signal %s missing from the contract", name)
			continue
		}
		if signal.Status == "" {
			t.Errorf("signal %s has no status", name)
		}
		if signal.Status != SignalMeasured && signal.Reason == "" {
			t.Errorf("signal %s is %s without a reason", name, signal.Status)
		}
		if signal.Status != SignalMeasured && signal.Value != 0 {
			t.Errorf("signal %s is %s but carries value %v", name, signal.Status, signal.Value)
		}
	}
}

func approx(a, b float64) bool { return math.Abs(a-b) < 1e-6*math.Max(1, math.Abs(b)) }

func TestLinuxMemoryPressure_ThrashingHostIsCriticalWithSwapBytes(t *testing.T) {
	fs := &fixtureFS{files: map[string]string{"/proc/pressure/memory": "pressure_memory_thrash.txt", "/proc/vmstat": "vmstat_swapin_t0.txt"}}
	clock := &stepClock{now: time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)}
	c := newLinuxPressureCollector(fs, clock)

	first, err := c.Collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got := readingOf(t, first).Signals[SignalSwapInBytesPerSecond].Status; got != SignalNotYetSampled {
		t.Fatalf("first-sample swap-in status = %q, want not_yet_sampled", got)
	}

	fs.files["/proc/vmstat"] = "vmstat_swapin_t1.txt"
	clock.now = clock.now.Add(30 * time.Second)
	second, err := c.Collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	reading := readingOf(t, second)
	assertEverySignalHasState(t, reading)

	// 360,000 pages over 30 s = 12,000 pages/s = 49,152,000 bytes/s.
	if got := reading.Signals[SignalSwapInBytesPerSecond]; !got.Measured() || !approx(got.Value, 12000*testPageSize) {
		t.Fatalf("swap-in = %+v, want measured 12000 pages/s in bytes", got)
	}
	if got := reading.Signals[SignalSwapOutBytesPerSecond]; !got.Measured() || !approx(got.Value, 2000*testPageSize) {
		t.Fatalf("swap-out = %+v, want measured 2000 pages/s in bytes", got)
	}
	if got := reading.Signals[SignalPSIFullAvg60]; !got.Measured() || got.Value != 19.77 {
		t.Fatalf("psi full avg60 = %+v, want measured 19.77", got)
	}
	if got := reading.Signals[SignalPSISomeTotal]; !got.Measured() || got.Value != 48957885977 {
		t.Fatalf("psi some total = %+v", got)
	}
	if reading.Level != MemoryPressureCritical {
		t.Fatalf("level = %s (%s), want critical from PSI full avg60", reading.Level, reading.LevelReason)
	}
	if !strings.Contains(reading.LevelReason, "full avg60") {
		t.Errorf("level reason %q does not name the deciding signal", reading.LevelReason)
	}
	if code, _ := second.Values["memory_pressure_level_code"].(float64); code != 2 {
		t.Errorf("memory_pressure_level_code = %v, want 2", second.Values["memory_pressure_level_code"])
	}
	// The cross-platform paging alias stays in pages/s for existing consumers.
	if got, _ := second.Values["swap_in_per_second"].(float64); !approx(got, 12000) {
		t.Errorf("swap_in_per_second alias = %v, want 12000", got)
	}
}

func TestLinuxMemoryPressure_QuietHostIsNormal(t *testing.T) {
	fs := &fixtureFS{files: map[string]string{"/proc/pressure/memory": "pressure_memory_quiet.txt", "/proc/vmstat": "vmstat_swapin_t0.txt"}}
	clock := &stepClock{now: time.Unix(1000, 0)}
	c := newLinuxPressureCollector(fs, clock)
	data, err := c.Collect(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	reading := readingOf(t, data)
	if reading.Level != MemoryPressureNormal {
		t.Fatalf("level = %s (%s), want normal", reading.Level, reading.LevelReason)
	}
	if code, ok := data.Values["memory_pressure_level_code"].(float64); !ok || code != 0 {
		t.Fatalf("level code = %v, want 0", data.Values["memory_pressure_level_code"])
	}
}

func TestLinuxMemoryPressure_MissingFullClassIsUnsupportedNotZero(t *testing.T) {
	fs := &fixtureFS{files: map[string]string{"/proc/pressure/memory": "pressure_memory_some_only.txt", "/proc/vmstat": "vmstat_swapin_t0.txt"}}
	c := newLinuxPressureCollector(fs, &stepClock{now: time.Unix(1000, 0)})
	data, _ := c.Collect(context.Background())
	reading := readingOf(t, data)
	full := reading.Signals[SignalPSIFullAvg60]
	if full.Status != SignalUnsupported || full.Value != 0 || full.Reason == "" {
		t.Fatalf("absent full class = %+v, want unsupported with reason", full)
	}
	if reading.Level != MemoryPressureElevated {
		t.Fatalf("level = %s, want elevated from some avg60 11.5%% >= 10%%", reading.Level)
	}
}

func TestLinuxMemoryPressure_NoPSIFallsBackToSwapInRate(t *testing.T) {
	fs := &fixtureFS{files: map[string]string{"/proc/vmstat": "vmstat_swapin_t0.txt"}}
	clock := &stepClock{now: time.Unix(1000, 0)}
	c := newLinuxPressureCollector(fs, clock)
	first, _ := c.Collect(context.Background())
	if got := readingOf(t, first); got.Level != MemoryPressureUnknown {
		t.Fatalf("level with no PSI and no swap rate = %s, want unknown", got.Level)
	}
	if code, ok := first.Values["memory_pressure_level_code"]; ok {
		t.Fatalf("unknown level must not publish a numeric code, got %v", code)
	}

	fs.files["/proc/vmstat"] = "vmstat_swapin_t1.txt"
	clock.now = clock.now.Add(30 * time.Second)
	second, _ := c.Collect(context.Background())
	reading := readingOf(t, second)
	if got := reading.Signals[SignalPSISomeAvg60]; got.Status != SignalUnsupported {
		t.Fatalf("psi without the interface = %+v, want unsupported", got)
	}
	// 46.9 MiB/s is above the 10 MiB/s elevated boundary, below 50 MiB/s.
	if reading.Level != MemoryPressureElevated || !strings.Contains(reading.LevelSource, "PSI unavailable") {
		t.Fatalf("level = %s via %q, want elevated from swap-in only", reading.Level, reading.LevelSource)
	}
}

func TestLinuxMemoryPressure_ThresholdsComeFromSettings(t *testing.T) {
	fs := &fixtureFS{files: map[string]string{"/proc/pressure/memory": "pressure_memory_quiet.txt", "/proc/vmstat": "vmstat_swapin_t0.txt"}}
	c := newLinuxPressureCollector(fs, &stepClock{now: time.Unix(1000, 0)})
	c.SetMemoryPressureThresholds(MemoryPressureThresholds{SomeAvg60ElevatedPercent: 1.5})
	data, _ := c.Collect(context.Background())
	if got := readingOf(t, data).Level; got != MemoryPressureElevated {
		t.Fatalf("level with a 1.5%% some-avg60 boundary = %s, want elevated (quiet host reads 1.90%%)", got)
	}
}

func TestDeriveMemoryPressureLevel(t *testing.T) {
	m := func(v float64) PressureSignal { return measuredSignal(v, "test", "") }
	u := unsupportedSignal("absent", "test", "")
	cases := []struct {
		name     string
		platform string
		signals  map[string]PressureSignal
		want     string
	}{
		{"linux some avg60 at boundary", "linux", map[string]PressureSignal{SignalPSISomeAvg60: m(10), SignalPSIFullAvg60: m(0)}, MemoryPressureElevated},
		{"linux full avg60 at boundary", "linux", map[string]PressureSignal{SignalPSISomeAvg60: m(4), SignalPSIFullAvg60: m(5)}, MemoryPressureCritical},
		{"linux swap-in critical with calm PSI", "linux", map[string]PressureSignal{SignalPSISomeAvg60: m(1), SignalPSIFullAvg60: m(0), SignalSwapInBytesPerSecond: m(50 * mebibyte)}, MemoryPressureCritical},
		{"linux swap-in elevated with calm PSI", "linux", map[string]PressureSignal{SignalPSISomeAvg60: m(1), SignalPSIFullAvg60: m(0), SignalSwapInBytesPerSecond: m(10 * mebibyte)}, MemoryPressureElevated},
		{"linux nothing supported", "linux", map[string]PressureSignal{SignalPSISomeAvg60: u, SignalPSIFullAvg60: u, SignalSwapInBytesPerSecond: u}, MemoryPressureUnknown},
		{"darwin native normal", "darwin", map[string]PressureSignal{SignalNativePressureLevel: m(1)}, MemoryPressureNormal},
		{"darwin native warn", "darwin", map[string]PressureSignal{SignalNativePressureLevel: m(2)}, MemoryPressureElevated},
		{"darwin native critical", "darwin", map[string]PressureSignal{SignalNativePressureLevel: m(4)}, MemoryPressureCritical},
		{"darwin swap-in raises normal native", "darwin", map[string]PressureSignal{SignalNativePressureLevel: m(1), SignalSwapInBytesPerSecond: m(60 * mebibyte)}, MemoryPressureCritical},
		{"darwin swap-in never lowers native", "darwin", map[string]PressureSignal{SignalNativePressureLevel: m(4), SignalSwapInBytesPerSecond: m(0)}, MemoryPressureCritical},
		{"darwin unrecognized native value", "darwin", map[string]PressureSignal{SignalNativePressureLevel: m(3)}, MemoryPressureUnknown},
		{"darwin native missing, swap-in only", "darwin", map[string]PressureSignal{SignalNativePressureLevel: u, SignalSwapInBytesPerSecond: m(11 * mebibyte)}, MemoryPressureElevated},
		{"windows low-memory signaled", "windows", map[string]PressureSignal{SignalNativePressureLevel: m(1)}, MemoryPressureCritical},
		{"windows low-memory clear", "windows", map[string]PressureSignal{SignalNativePressureLevel: m(0)}, MemoryPressureNormal},
		{"windows notification unavailable", "windows", map[string]PressureSignal{SignalNativePressureLevel: u}, MemoryPressureUnknown},
		{"other platform", "plan9", map[string]PressureSignal{}, MemoryPressureUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, source, reason := DeriveMemoryPressureLevel(tc.platform, tc.signals, DefaultMemoryPressureThresholds())
			if got != tc.want {
				t.Fatalf("level = %s, want %s", got, tc.want)
			}
			if source == "" || reason == "" {
				t.Fatalf("level %s has no source/reason (%q, %q)", got, source, reason)
			}
		})
	}
}

func loadDarwinFixture(t *testing.T, name string) darwinsys.VMStatistics64 {
	t.Helper()
	raw, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	var stats darwinsys.VMStatistics64
	if err := json.Unmarshal(raw, &stats); err != nil {
		t.Fatal(err)
	}
	return stats
}

func TestDarwinMemoryPressure_FromHostStatisticsAndNativeLevel(t *testing.T) {
	stats := loadDarwinFixture(t, "darwin_vm_statistics64_t0.json")
	level := uint32(1)
	clock := &stepClock{now: time.Unix(5000, 0)}
	c := NewPressureCollector()
	c.platform = "darwin"
	c.now = clock.Now
	c.readPaging = func(map[string]uint64) pagingReading {
		return darwinPagingReading(stats, nil, level, nil, testPageSize)
	}
	first, _ := c.Collect(context.Background())
	if got := readingOf(t, first); got.Level != MemoryPressureNormal {
		t.Fatalf("recorded quiet Mac = %s (%s), want normal", got.Level, got.LevelReason)
	}

	stats = loadDarwinFixture(t, "darwin_vm_statistics64_t1.json")
	level = 2
	clock.now = clock.now.Add(10 * time.Second)
	second, _ := c.Collect(context.Background())
	reading := readingOf(t, second)
	assertEverySignalHasState(t, reading)
	if got := reading.Signals[SignalSwapInBytesPerSecond]; !got.Measured() || !approx(got.Value, 3000*testPageSize) {
		t.Fatalf("swap-in = %+v, want 3000 pages/s in bytes", got)
	}
	if got := reading.Signals[SignalPageInsPerSecond]; !got.Measured() || !approx(got.Value, 4000) {
		t.Fatalf("page-ins = %+v, want 4000/s", got)
	}
	if got := reading.Signals[SignalCompressionsPerSecond]; !got.Measured() || !approx(got.Value, 1000) {
		t.Fatalf("compressions = %+v, want 1000/s", got)
	}
	if got := reading.Signals[SignalMajorFaultsPerSecond]; got.Status != SignalUnsupported || !strings.Contains(got.Reason, "major-fault") {
		t.Fatalf("macOS major faults = %+v, want unsupported with reason", got)
	}
	if got := reading.Signals[SignalPSISomeAvg60]; got.Status != SignalUnsupported {
		t.Fatalf("macOS PSI = %+v, want unsupported", got)
	}
	// Native warn (2) -> elevated; 11.7 MiB/s swap-in also >= 10 MiB/s.
	if reading.Level != MemoryPressureElevated {
		t.Fatalf("level = %s (%s), want elevated", reading.Level, reading.LevelReason)
	}
}

func TestDarwinPagingReading_FailedSysctlIsFailedNotNormal(t *testing.T) {
	reading := darwinPagingReading(darwinsys.VMStatistics64{}, errors.New("kern_return 5"), 0, errors.New("sysctl: no such file"), testPageSize)
	if reading.supported {
		t.Fatal("paging reported supported after host_statistics64 failed")
	}
	if got := reading.gauges[SignalNativePressureLevel]; got.Status != SignalFailed {
		t.Fatalf("native level after sysctl failure = %+v, want failed", got)
	}
	level, _, _ := DeriveMemoryPressureLevel("darwin", completeSignals("darwin", map[string]PressureSignal{SignalNativePressureLevel: reading.gauges[SignalNativePressureLevel]}), DefaultMemoryPressureThresholds())
	if level != MemoryPressureUnknown {
		t.Fatalf("level with failed sources = %s, want unknown", level)
	}
}

type windowsFixture struct {
	PagesInputTotal        uint64   `json:"pages_input_total"`
	PageReadsTotal         uint64   `json:"page_reads_total"`
	PagingFileUsagePercent *float64 `json:"paging_file_usage_percent"`
	PagingFileError        string   `json:"paging_file_error"`
	CommitCharge           uint64   `json:"commit_charge"`
	CommitLimit            uint64   `json:"commit_limit"`
	LowMemory              bool     `json:"low_memory"`
}

func loadWindowsFixture(t *testing.T, name string) windowsPagingSample {
	t.Helper()
	raw, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	var f windowsFixture
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	sample := windowsPagingSample{PagesInputTotal: f.PagesInputTotal, PageReadsTotal: f.PageReadsTotal, CommitCharge: f.CommitCharge, CommitLimit: f.CommitLimit, LowMemory: f.LowMemory}
	if f.PagingFileUsagePercent != nil {
		sample.PagingFileUsagePercent = *f.PagingFileUsagePercent
	} else {
		sample.PagingFileErr = errors.New(f.PagingFileError)
	}
	return sample
}

func TestWindowsMemoryPressure_FromPDHSamples(t *testing.T) {
	sample := loadWindowsFixture(t, "windows_pdh_t0.json")
	clock := &stepClock{now: time.Unix(9000, 0)}
	c := NewPressureCollector()
	c.platform = "windows"
	c.now = clock.Now
	c.readPaging = func(map[string]uint64) pagingReading {
		reading := windowsPagingReading(sample)
		reading.pageSize = testPageSize
		return reading
	}
	first, _ := c.Collect(context.Background())
	firstReading := readingOf(t, first)
	if firstReading.Level != MemoryPressureNormal {
		t.Fatalf("level = %s, want normal while the low-memory notification is clear", firstReading.Level)
	}
	if got := firstReading.Signals[SignalCommitChargePercent]; !got.Measured() || !approx(got.Value, 75) {
		t.Fatalf("commit charge = %+v, want 75%%", got)
	}
	if got := firstReading.Signals[SignalPagingFileUsagePercent]; !got.Measured() || got.Value != 41.5 {
		t.Fatalf("paging file usage = %+v, want 41.5", got)
	}

	sample = loadWindowsFixture(t, "windows_pdh_t1.json")
	clock.now = clock.now.Add(10 * time.Second)
	second, _ := c.Collect(context.Background())
	reading := readingOf(t, second)
	assertEverySignalHasState(t, reading)
	if got := reading.Signals[SignalPageInsPerSecond]; !got.Measured() || !approx(got.Value, 5000) {
		t.Fatalf("Pages Input/sec = %+v, want 5000", got)
	}
	if got := reading.Signals[SignalPageReadsPerSecond]; !got.Measured() || !approx(got.Value, 1000) {
		t.Fatalf("Page Reads/sec = %+v, want 1000", got)
	}
	if got := reading.Signals[SignalSwapInBytesPerSecond]; got.Status != SignalUnsupported || !strings.Contains(got.Reason, "mapped file") {
		t.Fatalf("Windows swap-in = %+v, want unsupported naming mapped-file reads", got)
	}
	if got := reading.Signals[SignalPagingFileUsagePercent]; got.Status != SignalFailed || got.Value != 0 {
		t.Fatalf("missing paging-file counter = %+v, want failed without a value", got)
	}
	if reading.Level != MemoryPressureCritical {
		t.Fatalf("level = %s, want critical when the low-memory notification is signaled", reading.Level)
	}
}

func TestUnsupportedPlatformReportsUnknownNotNormal(t *testing.T) {
	c := NewPressureCollector()
	c.platform = "freebsd"
	c.now = func() time.Time { return time.Unix(1, 0) }
	c.readPaging = func(map[string]uint64) pagingReading {
		return pagingReading{reason: "paging counters are unsupported on this platform", provenance: "platform capability"}
	}
	data, _ := c.Collect(context.Background())
	reading := readingOf(t, data)
	assertEverySignalHasState(t, reading)
	if reading.Level != MemoryPressureUnknown {
		t.Fatalf("level = %s, want unknown", reading.Level)
	}
	if status, _ := data.Values["swap_in_per_second_status"].(string); status != "unsupported" {
		t.Fatalf("swap_in_per_second_status = %q, want unsupported", status)
	}
}
