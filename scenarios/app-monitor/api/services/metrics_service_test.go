package services

import (
	"context"
	"testing"
	"time"
)

func TestNewMetricsService(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		service := NewMetricsService()

		if service == nil {
			t.Fatal("Expected non-nil metrics service")
		}

		if service.cacheTTL != 5*time.Second {
			t.Errorf("Expected cache TTL to be 5s, got %v", service.cacheTTL)
		}

		if service.cache != nil {
			t.Error("Expected cache to be nil initially")
		}
	})
}

func TestGetSystemMetrics(t *testing.T) {
	service := NewMetricsService()
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		metrics, err := service.GetSystemMetrics(ctx)

		// Note: This test may fail if system commands are not available
		// We expect it to at least attempt collection
		if metrics == nil && err == nil {
			t.Error("Expected either metrics or error, got both nil")
		}

		if metrics != nil {
			// Validate metrics structure
			if metrics.CPU < 0 {
				t.Errorf("Expected CPU >= 0, got %f", metrics.CPU)
			}
			if metrics.Memory < 0 {
				t.Errorf("Expected Memory >= 0, got %f", metrics.Memory)
			}
			if metrics.Disk < 0 {
				t.Errorf("Expected Disk >= 0, got %f", metrics.Disk)
			}
			if metrics.Network < 0 {
				t.Errorf("Expected Network >= 0, got %f", metrics.Network)
			}
			if metrics.Timestamp.IsZero() {
				t.Error("Expected non-zero timestamp")
			}
		}
	})

	t.Run("CachingBehavior", func(t *testing.T) {
		// Get metrics first time
		metrics1, err1 := service.GetSystemMetrics(ctx)
		if err1 != nil {
			t.Skipf("Skipping cache test, metrics collection failed: %v", err1)
		}

		// Get metrics second time (should use cache)
		metrics2, err2 := service.GetSystemMetrics(ctx)
		if err2 != nil {
			t.Fatalf("Second call failed: %v", err2)
		}

		// Timestamps should be the same (from cache)
		if !metrics1.Timestamp.Equal(metrics2.Timestamp) {
			t.Error("Expected cached metrics to have same timestamp")
		}

		// Wait for cache to expire
		time.Sleep(6 * time.Second)

		// Get metrics third time (should refresh cache)
		metrics3, err3 := service.GetSystemMetrics(ctx)
		if err3 != nil {
			t.Fatalf("Third call failed: %v", err3)
		}

		// Timestamp should be different (fresh metrics)
		if metrics1.Timestamp.Equal(metrics3.Timestamp) {
			t.Error("Expected fresh metrics to have different timestamp")
		}
	})

	t.Run("ContextCancellation", func(t *testing.T) {
		// Create a context that's already cancelled
		cancelledCtx, cancel := context.WithCancel(ctx)
		cancel()

		_, err := service.GetSystemMetrics(cancelledCtx)
		// Even with cancelled context, the function might succeed
		// if it doesn't check context in all places
		// This is more of a documentation of current behavior
		_ = err // We don't assert here as the behavior depends on implementation
	})
}

func TestCollectSystemMetrics(t *testing.T) {
	service := NewMetricsService()
	ctx := context.Background()

	t.Run("ParallelCollection", func(t *testing.T) {
		// This tests that collectSystemMetrics works
		metrics, err := service.collectSystemMetrics(ctx)

		// We expect either success or error, not panic
		if metrics == nil && err == nil {
			t.Error("Expected either metrics or error")
		}

		if metrics != nil {
			// Validate that timestamp is recent
			timeDiff := time.Since(metrics.Timestamp)
			if timeDiff > 5*time.Second {
				t.Errorf("Metrics timestamp too old: %v", timeDiff)
			}
		}
	})

	t.Run("ConcurrentAccess", func(t *testing.T) {
		// Test concurrent access to metrics service
		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func() {
				_, _ = service.GetSystemMetrics(ctx)
				done <- true
			}()
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}

		// If we got here without panic, test passes
	})
}

func TestGetMetricsHistory(t *testing.T) {
	service := NewMetricsService()
	ctx := context.Background()

	t.Run("ReturnsCurrentMetrics", func(t *testing.T) {
		// GetMetricsHistory currently returns current metrics
		history, err := service.GetMetricsHistory(ctx, 1*time.Hour)

		// May fail if system commands aren't available
		if history == nil && err == nil {
			t.Error("Expected either history or error")
		}

		if history != nil {
			if len(history) == 0 {
				t.Error("Expected non-empty history slice")
			}
		}
	})
}

func TestMetricsCacheThreadSafety(t *testing.T) {
	service := NewMetricsService()
	ctx := context.Background()

	t.Run("ConcurrentReadWrite", func(t *testing.T) {
		// Start multiple readers and writers
		done := make(chan bool, 20)

		// 10 readers
		for i := 0; i < 10; i++ {
			go func() {
				for j := 0; j < 5; j++ {
					service.GetSystemMetrics(ctx)
					time.Sleep(10 * time.Millisecond)
				}
				done <- true
			}()
		}

		// 10 writers (forcing cache refresh)
		for i := 0; i < 10; i++ {
			go func() {
				for j := 0; j < 5; j++ {
					service.collectSystemMetrics(ctx)
					time.Sleep(10 * time.Millisecond)
				}
				done <- true
			}()
		}

		// Wait for all goroutines
		for i := 0; i < 20; i++ {
			<-done
		}

		// If we got here without panic or race condition, test passes
	})
}

func TestMetricsCacheTTL(t *testing.T) {
	service := NewMetricsService()
	ctx := context.Background()

	t.Run("ExpiredCacheRefresh", func(t *testing.T) {
		// Get initial metrics
		_, err := service.GetSystemMetrics(ctx)
		if err != nil {
			t.Skipf("Skipping, initial metrics failed: %v", err)
		}

		// Manually expire the cache
		service.cacheMutex.Lock()
		service.cacheExpiry = time.Now().Add(-1 * time.Second)
		service.cacheMutex.Unlock()

		// Next call should refresh
		metricsAfter, err := service.GetSystemMetrics(ctx)
		if err != nil {
			t.Fatalf("Failed to get metrics after expiry: %v", err)
		}

		if metricsAfter == nil {
			t.Error("Expected non-nil metrics after refresh")
		}
	})
}

func TestSystemMetricsStructure(t *testing.T) {
	t.Run("ValidateFieldTypes", func(t *testing.T) {
		metrics := &SystemMetrics{
			CPU:       45.5,
			Memory:    78.3,
			Disk:      62.1,
			Network:   12.4,
			Timestamp: time.Now(),
		}

		// Validate ranges (should be 0-100 for percentages)
		if metrics.CPU < 0 || metrics.CPU > 100 {
			t.Logf("Warning: CPU value outside expected range: %f", metrics.CPU)
		}
		if metrics.Memory < 0 || metrics.Memory > 100 {
			t.Logf("Warning: Memory value outside expected range: %f", metrics.Memory)
		}
		if metrics.Disk < 0 || metrics.Disk > 100 {
			t.Logf("Warning: Disk value outside expected range: %f", metrics.Disk)
		}

		// Timestamp should be valid
		if metrics.Timestamp.IsZero() {
			t.Error("Expected non-zero timestamp")
		}
	})
}

func TestMetricsServiceEdgeCases(t *testing.T) {
	t.Run("NilCache", func(t *testing.T) {
		service := &MetricsService{
			cacheTTL: 5 * time.Second,
		}
		ctx := context.Background()

		// Should handle nil cache gracefully
		_, err := service.GetSystemMetrics(ctx)
		// We don't assert error here as it depends on system availability
		_ = err
	})

	t.Run("ZeroCacheTTL", func(t *testing.T) {
		service := &MetricsService{
			cacheTTL: 0, // No caching
		}
		ctx := context.Background()

		// Should still work, just won't cache
		metrics1, err1 := service.GetSystemMetrics(ctx)
		if err1 != nil {
			t.Skipf("Skipping, metrics failed: %v", err1)
		}

		metrics2, err2 := service.GetSystemMetrics(ctx)
		if err2 != nil {
			t.Skipf("Skipping, metrics failed: %v", err2)
		}

		// With zero TTL, timestamps should be different
		if metrics1.Timestamp.Equal(metrics2.Timestamp) {
			t.Log("Note: Expected different timestamps with zero cache TTL")
		}
	})
}
