package procmetrics

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// mockProcReader provides canned /proc responses for testing.
type mockProcReader struct {
	mu       sync.Mutex
	utime    int64
	stime    int64
	rss      int64
	peak     int64
	threads  int
	alive    bool
	statErr  error
	statFn   func() (int64, int64, error)
	statusFn func() (int64, int64, int, error)
}

func (m *mockProcReader) ReadStat(_ int) (int64, int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.statFn != nil {
		return m.statFn()
	}
	if m.statErr != nil {
		return 0, 0, m.statErr
	}
	return m.utime, m.stime, nil
}

func (m *mockProcReader) ReadStatus(_ int) (int64, int64, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.statusFn != nil {
		return m.statusFn()
	}
	return m.rss, m.peak, m.threads, nil
}

func (m *mockProcReader) IsAlive(_ int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.alive
}

// mockWindowDetector returns configurable window detection results.
type mockWindowDetector struct {
	mu       sync.Mutex
	visible  bool
	geometry *WindowGeometry
	err      error
	calls    int32
}

func (m *mockWindowDetector) HasVisibleWindow(_ context.Context, _ int, _ string) (bool, error) {
	atomic.AddInt32(&m.calls, 1)
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.visible, m.err
}

func (m *mockWindowDetector) LargestVisibleWindow(_ context.Context, _ int, _ string) (*WindowGeometry, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.geometry, m.err
}

func (m *mockWindowDetector) setVisible(v bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.visible = v
}

func (m *mockWindowDetector) setGeometry(w, h int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.geometry = &WindowGeometry{Width: w, Height: h}
}

func silentLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestMonitor_CollectsSamples(t *testing.T) {
	proc := &mockProcReader{
		alive:   true,
		utime:   100,
		stime:   50,
		rss:     1024 * 1024,
		peak:    2 * 1024 * 1024,
		threads: 4,
	}
	window := &mockWindowDetector{}

	m := NewDefaultMonitor(proc, window, silentLogger())
	if err := m.Start(context.Background(), 1, "", 0, 0); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Let it collect a few samples.
	time.Sleep(2500 * time.Millisecond)
	m.Stop()

	report := m.Report()
	if len(report.Samples) < 2 {
		t.Fatalf("expected >= 2 samples, got %d", len(report.Samples))
	}

	for i, s := range report.Samples {
		if s.RSSBytes != 1024*1024 {
			t.Errorf("sample %d: rss = %d, want %d", i, s.RSSBytes, 1024*1024)
		}
		if s.Threads != 4 {
			t.Errorf("sample %d: threads = %d, want 4", i, s.Threads)
		}
	}
}

func TestMonitor_DetectsWindowNoSizeCheck(t *testing.T) {
	proc := &mockProcReader{
		alive:   true,
		utime:   100,
		stime:   50,
		rss:     1024 * 1024,
		peak:    2 * 1024 * 1024,
		threads: 1,
	}
	window := &mockWindowDetector{visible: false}

	m := NewDefaultMonitor(proc, window, silentLogger())
	// No expected size → any visible window counts as both splash and ready.
	if err := m.Start(context.Background(), 1, ":99", 0, 0); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Simulate window appearing after a short delay.
	time.Sleep(400 * time.Millisecond)
	window.setVisible(true)
	time.Sleep(400 * time.Millisecond)

	report := m.Report()
	if report.Startup.SplashVisibleAt == nil {
		t.Fatal("expected SplashVisibleAt to be set")
	}
	if report.Startup.SplashDurationMs == nil {
		t.Fatal("expected SplashDurationMs to be set")
	}
	if report.Startup.ReadyAt == nil {
		t.Fatal("expected ReadyAt to be set (no size check → splash = ready)")
	}
	if report.Startup.ReadyMs == nil {
		t.Fatal("expected ReadyMs to be set")
	}
	if *report.Startup.ReadyMs <= 0 {
		t.Errorf("ReadyMs = %d, want > 0", *report.Startup.ReadyMs)
	}

	m.Stop()
}

func TestMonitor_TwoPhaseDetection(t *testing.T) {
	proc := &mockProcReader{
		alive:   true,
		utime:   100,
		stime:   50,
		rss:     1024 * 1024,
		peak:    2 * 1024 * 1024,
		threads: 1,
	}
	// Start with visible=true but small geometry (splash screen).
	window := &mockWindowDetector{
		visible:  true,
		geometry: &WindowGeometry{Width: 300, Height: 200},
	}

	m := NewDefaultMonitor(proc, window, silentLogger())
	// Expected size 1280x720 → threshold 1152x648 (90%).
	if err := m.Start(context.Background(), 1, ":99", 1280, 720); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Give time for splash to be detected.
	time.Sleep(500 * time.Millisecond)

	report := m.Report()
	if report.Startup.SplashVisibleAt == nil {
		t.Fatal("expected SplashVisibleAt to be set (splash screen visible)")
	}
	if report.Startup.ReadyAt != nil {
		t.Fatal("expected ReadyAt to be nil (window too small)")
	}

	// Now simulate the main window appearing (large enough).
	window.setGeometry(1280, 720)
	time.Sleep(500 * time.Millisecond)

	report = m.Report()
	if report.Startup.ReadyAt == nil {
		t.Fatal("expected ReadyAt to be set after large window appeared")
	}
	if *report.Startup.ReadyMs <= *report.Startup.SplashDurationMs {
		t.Errorf("ReadyMs (%d) should be > SplashDurationMs (%d)",
			*report.Startup.ReadyMs, *report.Startup.SplashDurationMs)
	}

	m.Stop()
}

func TestMonitor_ProcessExitStopsPolling(t *testing.T) {
	proc := &mockProcReader{
		alive:   true,
		utime:   100,
		stime:   50,
		rss:     1024 * 1024,
		peak:    2 * 1024 * 1024,
		threads: 1,
	}
	window := &mockWindowDetector{}

	m := NewDefaultMonitor(proc, window, silentLogger())
	if err := m.Start(context.Background(), 1, "", 0, 0); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Simulate process dying after 1 sample.
	time.Sleep(1500 * time.Millisecond)
	proc.mu.Lock()
	proc.alive = false
	proc.mu.Unlock()

	// Monitor should finish on its own.
	select {
	case <-m.Done():
		// Good — monitor detected process exit.
	case <-time.After(3 * time.Second):
		t.Fatal("monitor did not stop after process exit")
	}

	report := m.Report()
	if report.Summary == nil {
		t.Fatal("expected summary to be computed after process exit")
	}
}

func TestMonitor_StopIsIdempotent(t *testing.T) {
	proc := &mockProcReader{alive: true, rss: 1024, peak: 2048, threads: 1}
	window := &mockWindowDetector{}

	m := NewDefaultMonitor(proc, window, silentLogger())
	if err := m.Start(context.Background(), 1, "", 0, 0); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	// Call Stop multiple times — should not panic.
	m.Stop()
	m.Stop()
	m.Stop()
}

func TestMonitor_ComputesSummary(t *testing.T) {
	callCount := int64(0)
	proc := &mockProcReader{
		alive:   true,
		peak:    4 * 1024 * 1024,
		threads: 2,
		statFn: func() (int64, int64, error) {
			c := atomic.AddInt64(&callCount, 1)
			// Simulate increasing CPU time.
			return c * 50, c * 20, nil
		},
		statusFn: func() (int64, int64, int, error) {
			c := atomic.LoadInt64(&callCount)
			// RSS varies between 1MB and 3MB.
			rss := int64(1024*1024) + (c%3)*1024*1024
			return rss, 4 * 1024 * 1024, 2 + int(c%3), nil
		},
	}
	window := &mockWindowDetector{}

	m := NewDefaultMonitor(proc, window, silentLogger())
	if err := m.Start(context.Background(), 1, "", 0, 0); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	time.Sleep(3500 * time.Millisecond)
	m.Stop()

	report := m.Report()
	if report.Summary == nil {
		t.Fatal("expected summary")
	}
	if report.Summary.SampleCount < 3 {
		t.Fatalf("expected >= 3 samples in summary, got %d", report.Summary.SampleCount)
	}
	if report.Summary.PeakRSSBytes <= 0 {
		t.Error("expected PeakRSSBytes > 0")
	}
	if report.Summary.AvgRSSBytes <= 0 {
		t.Error("expected AvgRSSBytes > 0")
	}
	if report.Summary.MaxThreads < 2 {
		t.Errorf("expected MaxThreads >= 2, got %d", report.Summary.MaxThreads)
	}
}

func TestMonitor_ReportSafeDuringRun(t *testing.T) {
	proc := &mockProcReader{alive: true, rss: 1024, peak: 2048, threads: 1}
	window := &mockWindowDetector{}

	m := NewDefaultMonitor(proc, window, silentLogger())
	if err := m.Start(context.Background(), 1, "", 0, 0); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Call Report concurrently while monitor is running.
	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r := m.Report()
			if r == nil {
				t.Error("Report returned nil")
			}
		}()
	}
	wg.Wait()
	m.Stop()
}

func TestMonitor_StartFailsForDeadProcess(t *testing.T) {
	proc := &mockProcReader{alive: false}
	window := &mockWindowDetector{}

	m := NewDefaultMonitor(proc, window, silentLogger())
	err := m.Start(context.Background(), 1, ":99", 0, 0)
	if err == nil {
		t.Fatal("expected error for dead process")
	}
}

func TestMonitor_NoDisplaySkipsWindowDetection(t *testing.T) {
	proc := &mockProcReader{alive: true, rss: 1024, peak: 2048, threads: 1}
	window := &mockWindowDetector{}

	m := NewDefaultMonitor(proc, window, silentLogger())
	if err := m.Start(context.Background(), 1, "", 0, 0); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	time.Sleep(500 * time.Millisecond)
	m.Stop()

	if atomic.LoadInt32(&window.calls) != 0 {
		t.Error("expected no window detection calls when display is empty")
	}

	report := m.Report()
	if report.Startup.SplashVisibleAt != nil {
		t.Error("expected nil SplashVisibleAt when display is empty")
	}
	if report.Startup.ReadyAt != nil {
		t.Error("expected nil ReadyAt when display is empty")
	}
}
