package collectors

import (
	"context"
	"runtime"
	"testing"
	"time"
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
		collector := NewDiskCollector()
		ctx := context.Background()

		metrics, err := collector.Collect(ctx)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if metrics == nil {
			t.Fatal("Expected metrics, got nil")
		}

		if metrics.CollectorName != "disk" {
			t.Errorf("Expected collector name 'disk', got %s", metrics.CollectorName)
		}
	})

	t.Run("DiskStats", func(t *testing.T) {
		collector := NewDiskCollector()
		ctx := context.Background()

		metrics, err := collector.Collect(ctx)
		if err != nil {
			t.Fatalf("Failed to collect metrics: %v", err)
		}

		// Disk metrics should contain partition information
		if metrics.Values == nil {
			t.Fatal("Expected values map, got nil")
		}

		// Log available fields for debugging
		t.Logf("Disk metrics values: %+v", metrics.Values)
	})
}

// TestNetworkCollector_Collect tests network metrics collection
func TestNetworkCollector_Collect(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		collector := NewNetworkCollector()
		ctx := context.Background()

		metrics, err := collector.Collect(ctx)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if metrics == nil {
			t.Fatal("Expected metrics, got nil")
		}

		if metrics.CollectorName != "network" {
			t.Errorf("Expected collector name 'network', got %s", metrics.CollectorName)
		}
	})

	t.Run("NetworkInterfaces", func(t *testing.T) {
		collector := NewNetworkCollector()
		ctx := context.Background()

		metrics, err := collector.Collect(ctx)
		if err != nil {
			t.Fatalf("Failed to collect metrics: %v", err)
		}

		// Network metrics should contain interface information
		if metrics.Values == nil {
			t.Fatal("Expected values map, got nil")
		}

		t.Logf("Network metrics values: %+v", metrics.Values)
	})
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
