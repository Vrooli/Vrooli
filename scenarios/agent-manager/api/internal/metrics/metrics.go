// Package metrics provides Prometheus metrics for agent-manager.
package metrics

import (
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds all prometheus metrics for agent-manager.
type Metrics struct {
	// Run metrics
	RunsTotal       *prometheus.CounterVec
	RunsActive      prometheus.Gauge
	RunDuration     *prometheus.HistogramVec
	RunsByStatus    *prometheus.GaugeVec
	RunsByRunner    *prometheus.GaugeVec
	RunStopTotal    *prometheus.CounterVec
	RunStopDuration *prometheus.HistogramVec

	// Task metrics
	TasksTotal       *prometheus.CounterVec
	TasksActive      prometheus.Gauge
	TasksByStatus    *prometheus.GaugeVec
	TaskProcessing   *prometheus.HistogramVec

	// Runner metrics
	RunnerAvailability *prometheus.GaugeVec
	RunnerErrors       *prometheus.CounterVec

	// Event metrics
	EventsTotal     *prometheus.CounterVec
	EventsPerSecond prometheus.Gauge

	// WebSocket metrics
	WebSocketConnections prometheus.Gauge
	WebSocketMessages    *prometheus.CounterVec

	// API metrics
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec
	HTTPResponseSize    *prometheus.HistogramVec

	// Cost tracking
	TotalCostUSD *prometheus.CounterVec
	TokensUsed   *prometheus.CounterVec
}

var (
	instance *Metrics
	once     sync.Once
)

// Get returns the singleton Metrics instance.
func Get() *Metrics {
	once.Do(func() {
		instance = newMetrics()
	})
	return instance
}

func newMetrics() *Metrics {
	m := &Metrics{
		// Run metrics
		RunsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "agent_manager_runs_total",
				Help: "Total number of runs created",
			},
			[]string{"runner_type", "run_mode"},
		),
		RunsActive: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "agent_manager_runs_active",
				Help: "Number of currently active runs",
			},
		),
		RunDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "agent_manager_run_duration_seconds",
				Help:    "Duration of run executions",
				Buckets: []float64{1, 5, 15, 30, 60, 120, 300, 600, 1800, 3600},
			},
			[]string{"runner_type", "status"},
		),
		RunsByStatus: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "agent_manager_runs_by_status",
				Help: "Number of runs by status",
			},
			[]string{"status"},
		),
		RunsByRunner: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "agent_manager_runs_by_runner",
				Help: "Number of runs by runner type",
			},
			[]string{"runner_type"},
		),
		RunStopTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "agent_manager_run_stops_total",
				Help: "Total number of run stop operations",
			},
			[]string{"method", "success"},
		),
		RunStopDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "agent_manager_run_stop_duration_seconds",
				Help:    "Duration of run stop operations",
				Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30},
			},
			[]string{"method"},
		),

		// Task metrics
		TasksTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "agent_manager_tasks_total",
				Help: "Total number of tasks created",
			},
			[]string{},
		),
		TasksActive: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "agent_manager_tasks_active",
				Help: "Number of currently active tasks",
			},
		),
		TasksByStatus: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "agent_manager_tasks_by_status",
				Help: "Number of tasks by status",
			},
			[]string{"status"},
		),
		TaskProcessing: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "agent_manager_task_processing_seconds",
				Help:    "Task processing time",
				Buckets: []float64{1, 5, 15, 30, 60, 300, 600, 1800},
			},
			[]string{"status"},
		),

		// Runner metrics
		RunnerAvailability: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "agent_manager_runner_available",
				Help: "Whether a runner is available (1=yes, 0=no)",
			},
			[]string{"runner_type"},
		),
		RunnerErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "agent_manager_runner_errors_total",
				Help: "Total runner errors",
			},
			[]string{"runner_type", "error_type"},
		),

		// Event metrics
		EventsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "agent_manager_events_total",
				Help: "Total number of events emitted",
			},
			[]string{"event_type"},
		),
		EventsPerSecond: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "agent_manager_events_per_second",
				Help: "Current events per second rate",
			},
		),

		// WebSocket metrics
		WebSocketConnections: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "agent_manager_websocket_connections",
				Help: "Number of active WebSocket connections",
			},
		),
		WebSocketMessages: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "agent_manager_websocket_messages_total",
				Help: "Total WebSocket messages sent",
			},
			[]string{"type"},
		),

		// API metrics
		HTTPRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "agent_manager_http_requests_total",
				Help: "Total HTTP requests",
			},
			[]string{"method", "path", "status"},
		),
		HTTPRequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "agent_manager_http_request_duration_seconds",
				Help:    "HTTP request duration",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path"},
		),
		HTTPResponseSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "agent_manager_http_response_size_bytes",
				Help:    "HTTP response size in bytes",
				Buckets: []float64{100, 1000, 10000, 100000, 1000000},
			},
			[]string{"method", "path"},
		),

		// Cost tracking
		TotalCostUSD: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "agent_manager_cost_usd_total",
				Help: "Total cost in USD",
			},
			[]string{"runner_type"},
		),
		TokensUsed: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "agent_manager_tokens_total",
				Help: "Total tokens used",
			},
			[]string{"runner_type", "type"}, // type: input, output, cache_read, cache_creation
		),
	}

	// Register all metrics
	prometheus.MustRegister(
		m.RunsTotal,
		m.RunsActive,
		m.RunDuration,
		m.RunsByStatus,
		m.RunsByRunner,
		m.RunStopTotal,
		m.RunStopDuration,
		m.TasksTotal,
		m.TasksActive,
		m.TasksByStatus,
		m.TaskProcessing,
		m.RunnerAvailability,
		m.RunnerErrors,
		m.EventsTotal,
		m.EventsPerSecond,
		m.WebSocketConnections,
		m.WebSocketMessages,
		m.HTTPRequestsTotal,
		m.HTTPRequestDuration,
		m.HTTPResponseSize,
		m.TotalCostUSD,
		m.TokensUsed,
	)

	return m
}

// Handler returns the prometheus HTTP handler.
func Handler() http.Handler {
	return promhttp.Handler()
}

// RecordRunCreated records a new run being created.
func (m *Metrics) RecordRunCreated(runnerType, runMode string) {
	m.RunsTotal.WithLabelValues(runnerType, runMode).Inc()
	m.RunsActive.Inc()
}

// RecordRunCompleted records a run completion.
func (m *Metrics) RecordRunCompleted(runnerType, status string, duration time.Duration) {
	m.RunsActive.Dec()
	m.RunDuration.WithLabelValues(runnerType, status).Observe(duration.Seconds())
}

// RecordRunStop records a run stop operation.
func (m *Metrics) RecordRunStop(method string, success bool, duration time.Duration) {
	successStr := "true"
	if !success {
		successStr = "false"
	}
	m.RunStopTotal.WithLabelValues(method, successStr).Inc()
	m.RunStopDuration.WithLabelValues(method).Observe(duration.Seconds())
}

// RecordRunnerAvailability records whether a runner is available.
func (m *Metrics) RecordRunnerAvailability(runnerType string, available bool) {
	val := 0.0
	if available {
		val = 1.0
	}
	m.RunnerAvailability.WithLabelValues(runnerType).Set(val)
}

// RecordRunnerError records a runner error.
func (m *Metrics) RecordRunnerError(runnerType, errorType string) {
	m.RunnerErrors.WithLabelValues(runnerType, errorType).Inc()
}

// RecordEvent records an event being emitted.
func (m *Metrics) RecordEvent(eventType string) {
	m.EventsTotal.WithLabelValues(eventType).Inc()
}

// RecordWebSocketConnection records a WebSocket connection change.
func (m *Metrics) RecordWebSocketConnection(delta int) {
	if delta > 0 {
		m.WebSocketConnections.Add(float64(delta))
	} else {
		m.WebSocketConnections.Sub(float64(-delta))
	}
}

// RecordWebSocketMessage records a WebSocket message.
func (m *Metrics) RecordWebSocketMessage(msgType string) {
	m.WebSocketMessages.WithLabelValues(msgType).Inc()
}

// RecordHTTPRequest records an HTTP request.
func (m *Metrics) RecordHTTPRequest(method, path, status string, duration time.Duration, size int) {
	m.HTTPRequestsTotal.WithLabelValues(method, path, status).Inc()
	m.HTTPRequestDuration.WithLabelValues(method, path).Observe(duration.Seconds())
	m.HTTPResponseSize.WithLabelValues(method, path).Observe(float64(size))
}

// RecordCost records cost incurred.
func (m *Metrics) RecordCost(runnerType string, costUSD float64) {
	m.TotalCostUSD.WithLabelValues(runnerType).Add(costUSD)
}

// RecordTokens records token usage.
func (m *Metrics) RecordTokens(runnerType string, input, output, cacheRead, cacheCreation int) {
	m.TokensUsed.WithLabelValues(runnerType, "input").Add(float64(input))
	m.TokensUsed.WithLabelValues(runnerType, "output").Add(float64(output))
	m.TokensUsed.WithLabelValues(runnerType, "cache_read").Add(float64(cacheRead))
	m.TokensUsed.WithLabelValues(runnerType, "cache_creation").Add(float64(cacheCreation))
}

// UpdateStatusCounts updates the status gauge counts (call periodically).
func (m *Metrics) UpdateStatusCounts(runCounts, taskCounts map[string]int) {
	for status, count := range runCounts {
		m.RunsByStatus.WithLabelValues(status).Set(float64(count))
	}
	for status, count := range taskCounts {
		m.TasksByStatus.WithLabelValues(status).Set(float64(count))
	}
}
