package main

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

type metricsRegistry struct {
	activeSessions   atomic.Int64
	totalSessions    atomic.Int64
	panicStops       atomic.Int64
	idleTimeOuts     atomic.Int64
	ttlExpirations   atomic.Int64
	httpUpgradesFail atomic.Int64
}

func newMetricsRegistry() *metricsRegistry {
	return &metricsRegistry{}
}

func (m *metricsRegistry) serveHTTP(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	fmt.Fprintf(w, "# HELP codex_console_active_sessions Current active Codex sessions\n")
	fmt.Fprintf(w, "# TYPE codex_console_active_sessions gauge\n")
	fmt.Fprintf(w, "codex_console_active_sessions %d\n", m.activeSessions.Load())

	fmt.Fprintf(w, "# HELP codex_console_total_sessions Total Codex sessions created\n")
	fmt.Fprintf(w, "# TYPE codex_console_total_sessions counter\n")
	fmt.Fprintf(w, "codex_console_total_sessions %d\n", m.totalSessions.Load())

	fmt.Fprintf(w, "# HELP codex_console_panic_stops Sessions terminated via panic-stop\n")
	fmt.Fprintf(w, "# TYPE codex_console_panic_stops counter\n")
	fmt.Fprintf(w, "codex_console_panic_stops %d\n", m.panicStops.Load())

	fmt.Fprintf(w, "# HELP codex_console_idle_timeouts Sessions terminated due to idle timeout\n")
	fmt.Fprintf(w, "# TYPE codex_console_idle_timeouts counter\n")
	fmt.Fprintf(w, "codex_console_idle_timeouts %d\n", m.idleTimeOuts.Load())

	fmt.Fprintf(w, "# HELP codex_console_ttl_expirations Sessions terminated after TTL expiry\n")
	fmt.Fprintf(w, "# TYPE codex_console_ttl_expirations counter\n")
	fmt.Fprintf(w, "codex_console_ttl_expirations %d\n", m.ttlExpirations.Load())

	fmt.Fprintf(w, "# HELP codex_console_failed_upgrades WebSocket upgrade failures\n")
	fmt.Fprintf(w, "# TYPE codex_console_failed_upgrades counter\n")
	fmt.Fprintf(w, "codex_console_failed_upgrades %d\n", m.httpUpgradesFail.Load())
}
