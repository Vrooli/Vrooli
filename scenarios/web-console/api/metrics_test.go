package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewMetricsRegistry(t *testing.T) {
	metrics := newMetricsRegistry()

	if metrics == nil {
		t.Fatal("Expected non-nil metrics registry")
	}
	if metrics.activeSessions.Load() != 0 {
		t.Error("Expected active sessions to start at 0")
	}
	if metrics.totalSessions.Load() != 0 {
		t.Error("Expected total sessions to start at 0")
	}
	if metrics.panicStops.Load() != 0 {
		t.Error("Expected panic stops to start at 0")
	}
	if metrics.idleTimeOuts.Load() != 0 {
		t.Error("Expected idle timeouts to start at 0")
	}
	if metrics.ttlExpirations.Load() != 0 {
		t.Error("Expected TTL expirations to start at 0")
	}
}

func TestMetricsServeHTTP(t *testing.T) {
	metrics := newMetricsRegistry()

	// Set some values
	metrics.activeSessions.Store(3)
	metrics.totalSessions.Store(10)
	metrics.panicStops.Store(2)
	metrics.idleTimeOuts.Store(1)
	metrics.ttlExpirations.Store(4)

	t.Run("PrometheusFormat", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		w := httptest.NewRecorder()

		metrics.serveHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		body := w.Body.String()

		// Check for metric names
		expectedMetrics := []string{
			"web_console_active_sessions",
			"web_console_total_sessions",
			"web_console_panic_stops",
			"web_console_idle_timeouts",
			"web_console_ttl_expirations",
		}

		for _, metric := range expectedMetrics {
			if !strings.Contains(body, metric) {
				t.Errorf("Expected metric '%s' in output", metric)
			}
		}

		// Check for specific values
		if !strings.Contains(body, "web_console_active_sessions 3") {
			t.Error("Expected active sessions value 3 in output")
		}
		if !strings.Contains(body, "web_console_total_sessions 10") {
			t.Error("Expected total sessions value 10 in output")
		}
		if !strings.Contains(body, "web_console_panic_stops 2") {
			t.Error("Expected panic stops value 2 in output")
		}
	})

	t.Run("ContentType", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		w := httptest.NewRecorder()

		metrics.serveHTTP(w, req)

		contentType := w.Header().Get("Content-Type")
		if contentType != "text/plain; version=0.0.4" {
			t.Errorf("Expected Content-Type 'text/plain; version=0.0.4', got '%s'", contentType)
		}
	})
}

func TestMetricsIncrement(t *testing.T) {
	metrics := newMetricsRegistry()

	t.Run("ActiveSessions", func(t *testing.T) {
		initial := metrics.activeSessions.Load()
		metrics.activeSessions.Add(1)
		if metrics.activeSessions.Load() != initial+1 {
			t.Error("Active sessions should increment")
		}
	})

	t.Run("TotalSessions", func(t *testing.T) {
		initial := metrics.totalSessions.Load()
		metrics.totalSessions.Add(1)
		if metrics.totalSessions.Load() != initial+1 {
			t.Error("Total sessions should increment")
		}
	})

	t.Run("PanicStops", func(t *testing.T) {
		initial := metrics.panicStops.Load()
		metrics.panicStops.Add(1)
		if metrics.panicStops.Load() != initial+1 {
			t.Error("Panic stops should increment")
		}
	})

	t.Run("IdleTimeouts", func(t *testing.T) {
		initial := metrics.idleTimeOuts.Load()
		metrics.idleTimeOuts.Add(1)
		if metrics.idleTimeOuts.Load() != initial+1 {
			t.Error("Idle timeouts should increment")
		}
	})

	t.Run("TTLExpirations", func(t *testing.T) {
		initial := metrics.ttlExpirations.Load()
		metrics.ttlExpirations.Add(1)
		if metrics.ttlExpirations.Load() != initial+1 {
			t.Error("TTL expirations should increment")
		}
	})
}

func TestMetricsDecrement(t *testing.T) {
	metrics := newMetricsRegistry()

	t.Run("ActiveSessions", func(t *testing.T) {
		metrics.activeSessions.Store(5)
		metrics.activeSessions.Add(-1)
		if metrics.activeSessions.Load() != 4 {
			t.Errorf("Expected 4 active sessions, got %d", metrics.activeSessions.Load())
		}
	})
}

func TestMetricsConcurrency(t *testing.T) {
	metrics := newMetricsRegistry()

	// Test concurrent increments
	done := make(chan bool)
	iterations := 100

	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < iterations; j++ {
				metrics.totalSessions.Add(1)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	expected := int64(10 * iterations)
	if metrics.totalSessions.Load() != expected {
		t.Errorf("Expected %d total sessions, got %d", expected, metrics.totalSessions.Load())
	}
}
