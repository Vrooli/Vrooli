package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// defaultMetricsURL is the default cloudflared Prometheus metrics endpoint.
const defaultMetricsURL = "http://127.0.0.1:20241"

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// decodeJSON reads and decodes a JSON request body into dest.
// Returns true on success. On failure, writes a 400 error and returns false.
func decodeJSON(w http.ResponseWriter, r *http.Request, dest any) bool {
	if err := json.NewDecoder(r.Body).Decode(dest); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}
	return true
}

// purgeByTimestamp deletes rows older than the retention period from the given table.
// timestampCol is the column name used for the cutoff comparison.
func purgeByTimestamp(db *sql.DB, table, timestampCol string, retention time.Duration) (int64, error) {
	cutoff := time.Now().Add(-retention)
	result, err := db.Exec(
		fmt.Sprintf(`DELETE FROM %s WHERE %s < $1`, table, timestampCol),
		cutoff,
	)
	if err != nil {
		return 0, fmt.Errorf("purge %s: %w", table, err)
	}
	return result.RowsAffected()
}

// pollReady polls the tunnel's /ready endpoint until it returns "ok" or
// the timeout elapses. Returns nil on success, error on timeout.
func pollReady(ctx context.Context, timeout, interval time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		checker := NewTunnelHealthChecker()
		status := checker.Check(ctx)
		if status.Ready == "ok" {
			return nil
		}
		time.Sleep(interval)
	}
	return fmt.Errorf("cloudflared did not become ready within %v after restart", timeout)
}

// extractHostname strips the URL scheme and path, returning only the hostname.
// e.g., "https://api.example.com/path" → "api.example.com"
func extractHostname(publicURL string) string {
	host := strings.TrimPrefix(publicURL, "https://")
	host = strings.TrimPrefix(host, "http://")
	if i := strings.IndexByte(host, '/'); i >= 0 {
		host = host[:i]
	}
	return host
}
