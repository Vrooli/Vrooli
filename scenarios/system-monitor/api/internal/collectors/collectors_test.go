package collectors

import (
	"context"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
)

// TestCPUCollector_Collect tests CPU metrics collection
func TestCPUCollector_Collect(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		collector := NewCPUCollector()
		ctx := context.Background()

		metrics, err := collector.Collect(ctx)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if metrics == nil {
			t.Fatal("Expected metrics, got nil")
		}

		// Validate metric data structure
		if metrics.CollectorName != "cpu" {
			t.Errorf("Expected collector name 'cpu', got %s", metrics.CollectorName)
		}

		if metrics.Type != "cpu" {
			t.Errorf("Expected type 'cpu', got %s", metrics.Type)
		}

		// Validate values
		if metrics.Values == nil {
			t.Fatal("Expected values map, got nil")
		}

		// Check required fields
		requiredFields := []string{"usage_percent", "cores", "load_average", "context_switches", "goroutines"}
		for _, field := range requiredFields {
			if _, exists := metrics.Values[field]; !exists {
				t.Errorf("Expected field %s in values", field)
			}
		}

		// Validate CPU usage range
		if usage, ok := metrics.Values["usage_percent"].(float64); ok {
			if usage < 0 || usage > 100 {
				t.Errorf("Invalid CPU usage: %f (expected 0-100)", usage)
			}
		}

		// Validate cores count
		if cores, ok := metrics.Values["cores"].(int); ok {
			if cores != runtime.NumCPU() {
				t.Errorf("Expected %d cores, got %d", runtime.NumCPU(), cores)
			}
		}
	})

	t.Run("ContextCancellation", func(t *testing.T) {
		collector := NewCPUCollector()
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		// Collector should still work but context is cancelled
		_, err := collector.Collect(ctx)
		// Note: Current implementation doesn't check context, but it should complete
		if err != nil {
			t.Logf("Collector returned error with cancelled context: %v", err)
		}
	})

	t.Run("MultipleCollections", func(t *testing.T) {
		collector := NewCPUCollector()
		ctx := context.Background()

		// Collect multiple times to test state management
		for i := 0; i < 3; i++ {
			metrics, err := collector.Collect(ctx)
			if err != nil {
				t.Errorf("Collection %d failed: %v", i, err)
			}
			if metrics == nil {
				t.Errorf("Collection %d returned nil metrics", i)
			}
			time.Sleep(100 * time.Millisecond)
		}
	})
}

// TestMemoryCollector_Collect tests memory metrics collection
func TestMemoryCollector_Collect(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		collector := NewMemoryCollector()
		ctx := context.Background()

		metrics, err := collector.Collect(ctx)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if metrics == nil {
			t.Fatal("Expected metrics, got nil")
		}

		if metrics.CollectorName != "memory" {
			t.Errorf("Expected collector name 'memory', got %s", metrics.CollectorName)
		}

		// Validate values
		if metrics.Values == nil {
			t.Fatal("Expected values map, got nil")
		}

		// Check for common memory fields
		expectedFields := []string{"used_percent", "total", "used", "available"}
		for _, field := range expectedFields {
			if _, exists := metrics.Values[field]; !exists {
				t.Logf("Field %s not found in values (may be platform-specific)", field)
			}
		}
	})

	t.Run("MemoryUsageRange", func(t *testing.T) {
		collector := NewMemoryCollector()
		ctx := context.Background()

		metrics, err := collector.Collect(ctx)
		if err != nil {
			t.Fatalf("Failed to collect metrics: %v", err)
		}

		// Validate memory usage is within valid range
		if usedPercent, ok := metrics.Values["used_percent"].(float64); ok {
			if usedPercent < 0 || usedPercent > 100 {
				t.Errorf("Invalid memory usage: %f (expected 0-100)", usedPercent)
			}
		}
	})
}

// TestDiskCollector_Collect tests disk metrics collection
func TestDiskCollector_Collect(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		assertCollectorBasics(t, NewDiskCollector(), "disk")
	})

	t.Run("DiskStats", func(t *testing.T) {
		metrics := mustCollectMetrics(t, NewDiskCollector())
		t.Logf("Disk metrics values: %+v", metrics.Values)
	})
}

func TestDiskCollector_NoSteadyForks(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("disk collector reads /proc/statfs only on linux")
	}

	c := NewDiskCollector()
	forks := countCommandForks(t, func() error {
		_, err := c.Collect(context.Background())
		return err
	})
	if forks != 0 {
		t.Errorf("disk collection forked %d times, want 0", forks)
	}
}

// TestNetworkCollector_Collect tests network metrics collection
func TestNetworkCollector_Collect(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		assertCollectorBasics(t, NewNetworkCollector(), "network")
	})

	t.Run("NetworkInterfaces", func(t *testing.T) {
		metrics := mustCollectMetrics(t, NewNetworkCollector())
		t.Logf("Network metrics values: %+v", metrics.Values)
	})
}

func TestNetworkCollector_NoSteadyForks(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("network collector reads /proc only on linux")
	}

	c := NewNetworkCollector()
	forks := countCommandForks(t, func() error {
		_, err := c.Collect(context.Background())
		return err
	})
	if forks != 0 {
		t.Errorf("network collection forked %d times, want 0", forks)
	}
}

func countCommandForks(t *testing.T, collect func() error) int {
	t.Helper()

	var mu sync.Mutex
	forks := 0

	orig := commandOutput
	defer func() { commandOutput = orig }()
	commandOutput = func(_ context.Context, _ time.Duration, name string, args ...string) ([]byte, error) {
		mu.Lock()
		defer mu.Unlock()
		forks++
		return []byte(""), nil
	}

	if err := collect(); err != nil {
		t.Fatalf("collect: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	return forks
}

type testCollector interface {
	Collect(context.Context) (*MetricData, error)
}

func assertCollectorBasics(t *testing.T, collector testCollector, wantName string) {
	t.Helper()

	metrics := mustCollectMetrics(t, collector)
	if metrics.CollectorName != wantName {
		t.Errorf("Expected collector name %q, got %s", wantName, metrics.CollectorName)
	}
}

func mustCollectMetrics(t *testing.T, collector testCollector) *MetricData {
	t.Helper()

	metrics, err := collector.Collect(context.Background())
	if err != nil {
		t.Fatalf("Failed to collect metrics: %v", err)
	}
	if metrics == nil {
		t.Fatal("Expected metrics, got nil")
	}
	if metrics.Values == nil {
		t.Fatal("Expected values map, got nil")
	}
	return metrics
}

func TestParseNetDevLine(t *testing.T) {
	name, stats, ok := parseNetDevLine("  eth0: 100 2 3 4 0 0 0 0 900 8 7 6 0 0 0 0")
	if !ok {
		t.Fatal("parseNetDevLine returned !ok")
	}
	if name != "eth0" {
		t.Fatalf("name = %q, want eth0", name)
	}
	if stats.bytesRecv != 100 || stats.packetsRecv != 2 || stats.errorsIn != 3 || stats.droppedIn != 4 {
		t.Fatalf("recv stats parsed incorrectly: %+v", stats)
	}
	if stats.bytesSent != 900 || stats.packetsSent != 8 || stats.errorsOut != 7 || stats.droppedOut != 6 {
		t.Fatalf("sent stats parsed incorrectly: %+v", stats)
	}
}

func TestIsEphemeralAddressPort(t *testing.T) {
	if !isEphemeralAddressPort("0100007F:7530") {
		t.Fatal("port 30000 should be counted as ephemeral")
	}
	if isEphemeralAddressPort("0100007F:0050") {
		t.Fatal("port 80 should not be counted as ephemeral")
	}
}

// TestProcessCollector_Collect tests process metrics collection
func TestProcessCollector_Collect(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		collector := NewProcessCollector()
		ctx := context.Background()

		metrics, err := collector.Collect(ctx)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if metrics == nil {
			t.Fatal("Expected metrics, got nil")
		}

		if metrics.CollectorName != "process" {
			t.Errorf("Expected collector name 'process', got %s", metrics.CollectorName)
		}
	})

	t.Run("ProcessCount", func(t *testing.T) {
		collector := NewProcessCollector()
		ctx := context.Background()

		metrics, err := collector.Collect(ctx)
		if err != nil {
			t.Fatalf("Failed to collect metrics: %v", err)
		}

		// Process metrics should contain process count
		if metrics.Values == nil {
			t.Fatal("Expected values map, got nil")
		}

		if processCount, ok := metrics.Values["total_processes"].(int); ok {
			if processCount < 1 {
				t.Errorf("Expected at least 1 process, got %d", processCount)
			}
		}
	})
}

// TestGPUCollector_Collect tests GPU metrics collection
func TestGPUCollector_Collect(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		collector := NewGPUCollector()
		ctx := context.Background()

		metrics, err := collector.Collect(ctx)
		// GPU may not be available, so error is acceptable
		if err != nil {
			t.Logf("GPU collection returned error (may be unavailable): %v", err)
			return
		}

		if metrics == nil {
			t.Fatal("Expected metrics, got nil")
		}

		if metrics.CollectorName != "gpu" {
			t.Errorf("Expected collector name 'gpu', got %s", metrics.CollectorName)
		}
	})
}

func TestAdaptGPUInventory(t *testing.T) {
	temp := 63.0
	snapshot := hostinventory.Snapshot{
		GPUs: []hostinventory.GPU{{
			Index:              0,
			UUID:               "GPU-1",
			Name:               "NVIDIA RTX 4090",
			DriverVersion:      "555.42",
			VRAMBytes:          24564 * 1024 * 1024,
			VRAMUsedBytes:      2048 * 1024 * 1024,
			UtilizationPercent: 42,
			TemperatureC:       &temp,
		}},
		GPUProcesses: []hostinventory.GPUProcess{{
			GPUUUID:     "GPU-1",
			PID:         123,
			ProcessName: "python",
			UsedBytes:   512 * 1024 * 1024,
		}},
	}

	devices, summary, driver, model := adaptGPUInventory(snapshot)
	if driver != "555.42" || model != "NVIDIA RTX 4090" {
		t.Fatalf("driver/model = %q/%q", driver, model)
	}
	if summary.DeviceCount != 1 || summary.UsedMemoryMB != 2048 || summary.TotalMemoryMB != 24564 {
		t.Fatalf("summary = %#v", summary)
	}
	if len(devices) != 1 {
		t.Fatalf("devices = %#v", devices)
	}
	want := models.GPUProcessInfo{PID: 123, ProcessName: "python", MemoryUsedMB: 512}
	if len(devices[0].Processes) != 1 || devices[0].Processes[0] != want {
		t.Fatalf("processes = %#v", devices[0].Processes)
	}
}

// TestBaseCollector tests base collector functionality
func TestBaseCollector(t *testing.T) {
	t.Run("GetName", func(t *testing.T) {
		collector := NewBaseCollector("test", 5*time.Second)
		if collector.GetName() != "test" {
			t.Errorf("Expected name 'test', got %s", collector.GetName())
		}
	})

	t.Run("Interval", func(t *testing.T) {
		interval := 10 * time.Second
		collector := NewBaseCollector("test", interval)
		if collector.interval != interval {
			t.Errorf("Expected interval %v, got %v", interval, collector.interval)
		}
	})

	t.Run("Enabled", func(t *testing.T) {
		collector := NewBaseCollector("test", 5*time.Second)
		if !collector.IsEnabled() {
			t.Error("Expected collector to be enabled by default")
		}

		collector.SetEnabled(false)
		if collector.IsEnabled() {
			t.Error("Expected collector to be disabled after SetEnabled(false)")
		}

		collector.SetEnabled(true)
		if !collector.IsEnabled() {
			t.Error("Expected collector to be enabled after SetEnabled(true)")
		}
	})
}

// TestGetTopProcessesByCPU tests top processes retrieval
func TestGetTopProcessesByCPU(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		processes, err := GetTopProcessesByCPU(5)

		if runtime.GOOS != "linux" {
			// Should return empty slice on non-Linux
			if err != nil {
				t.Errorf("Expected no error on non-Linux, got %v", err)
			}
			if processes == nil {
				t.Error("Expected empty slice, got nil")
			}
			return
		}

		// On Linux, should return processes
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if processes == nil {
			t.Fatal("Expected processes slice, got nil")
		}

		// Validate process structure
		for i, proc := range processes {
			if proc == nil {
				t.Errorf("Process %d is nil", i)
				continue
			}

			// Check required fields
			requiredFields := []string{"pid", "name", "cpu_percent", "mem_percent", "threads"}
			for _, field := range requiredFields {
				if _, exists := proc[field]; !exists {
					t.Errorf("Process %d missing field %s", i, field)
				}
			}
		}
	})

	t.Run("LimitRespected", func(t *testing.T) {
		limit := 3
		processes, err := GetTopProcessesByCPU(limit)

		if runtime.GOOS != "linux" {
			return // Skip on non-Linux
		}

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if len(processes) > limit {
			t.Errorf("Expected at most %d processes, got %d", limit, len(processes))
		}
	})
}

// Benchmark tests
func BenchmarkCPUCollector_Collect(b *testing.B) {
	collector := NewCPUCollector()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := collector.Collect(ctx)
		if err != nil {
			b.Fatalf("Collect failed: %v", err)
		}
	}
}

func BenchmarkMemoryCollector_Collect(b *testing.B) {
	collector := NewMemoryCollector()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := collector.Collect(ctx)
		if err != nil {
			b.Fatalf("Collect failed: %v", err)
		}
	}
}

func BenchmarkGetTopProcessesByCPU(b *testing.B) {
	if runtime.GOOS != "linux" {
		b.Skip("Benchmark only runs on Linux")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := GetTopProcessesByCPU(10)
		if err != nil {
			b.Fatalf("GetTopProcessesByCPU failed: %v", err)
		}
	}
}
