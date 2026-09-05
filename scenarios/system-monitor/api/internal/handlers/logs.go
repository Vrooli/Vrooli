package handlers

import (
	"context"
	"encoding/base64"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/httputil"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/services/journal"
)

const (
	logsDefaultLimit = 200
	logsMaxLimit     = 500
)

// JournalReader is the subset of *journal.Reader the logs handler needs.
type JournalReader interface {
	QueryLogs(ctx context.Context, opts journal.QueryOpts) ([]journal.LogEntry, error)
	ListBoots(ctx context.Context) ([]journal.BootRecord, error)
	Available(ctx context.Context) bool
}

// LogsHandler serves /api/v1/logs and friends.
type LogsHandler struct {
	reader JournalReader
	log    *slog.Logger
	now    func() time.Time
}

// NewLogsHandler builds the handler.
func NewLogsHandler(reader JournalReader, log *slog.Logger) *LogsHandler {
	return &LogsHandler{reader: reader, log: log, now: time.Now}
}

// LogsResponse is the response body for GET /api/v1/logs.
type LogsResponse struct {
	Available   bool               `json:"available"`
	Reason      string             `json:"reason,omitempty"`
	Entries     []journal.LogEntry `json:"entries,omitempty"`
	NextCursor  string             `json:"nextCursor,omitempty"`
	Direction   string             `json:"direction,omitempty"`
	Limit       int                `json:"limit,omitempty"`
	GeneratedAt time.Time          `json:"generatedAt"`
}

// Logs handles GET /api/v1/logs.
func (h *LogsHandler) Logs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	now := h.now().UTC()

	limit := parseLimit(q.Get("limit"))
	direction := q.Get("direction")
	if direction == "" {
		if q.Get("cursor") != "" {
			direction = "forward"
		} else {
			direction = "reverse"
		}
	}

	opts := journal.QueryOpts{
		Unit:     q["unit"],
		Kernel:   strings.EqualFold(q.Get("kernel"), "true"),
		Since:    normalizeJournalTime(q.Get("since"), now),
		Until:    normalizeJournalTime(q.Get("until"), now),
		Boot:     q.Get("boot"),
		Priority: q.Get("priority"),
		Grep:     q.Get("grep"),
		Tail:     limit,
		Reverse:  direction == "reverse",
	}

	if cursor := q.Get("cursor"); cursor != "" {
		decoded, err := base64.RawURLEncoding.DecodeString(cursor)
		if err != nil {
			httputil.JSONWithStatus(w, http.StatusBadRequest, LogsResponse{ //nolint:errcheck
				Reason:      "invalid cursor: " + err.Error(),
				GeneratedAt: now,
			})
			return
		}
		opts.AfterCursor = string(decoded)
	}

	if h.reader == nil {
		httputil.JSON(w, LogsResponse{Reason: "journal reader not configured", GeneratedAt: now}) //nolint:errcheck
		return
	}

	entries, err := h.reader.QueryLogs(r.Context(), opts)
	if err != nil {
		httputil.JSON(w, LogsResponse{ //nolint:errcheck
			Reason:      "journal query failed: " + err.Error(),
			GeneratedAt: now,
		})
		return
	}

	resp := LogsResponse{
		Available:   true,
		Entries:     entries,
		Direction:   direction,
		Limit:       limit,
		GeneratedAt: now,
	}
	if n := len(entries); n > 0 {
		if last := entries[n-1].Cursor; last != "" {
			resp.NextCursor = base64.RawURLEncoding.EncodeToString([]byte(last))
		}
	}
	httputil.JSON(w, resp) //nolint:errcheck
}

// UnitsResponse is the response for GET /api/v1/logs/units.
type UnitsResponse struct {
	Available   bool      `json:"available"`
	Reason      string    `json:"reason,omitempty"`
	Units       []string  `json:"units,omitempty"`
	GeneratedAt time.Time `json:"generatedAt"`
}

// Units handles GET /api/v1/logs/units.
//
// Returns the deduplicated set of unit names appearing in the most recent
// 1000 log entries. Cheap-and-good-enough — the dashboard only uses this
// for autocomplete.
func (h *LogsHandler) Units(w http.ResponseWriter, r *http.Request) {
	now := h.now().UTC()
	if h.reader == nil {
		httputil.JSON(w, UnitsResponse{Reason: "journal reader not configured", GeneratedAt: now}) //nolint:errcheck
		return
	}
	entries, err := h.reader.QueryLogs(r.Context(), journal.QueryOpts{Tail: 1000, Reverse: true})
	if err != nil {
		httputil.JSON(w, UnitsResponse{Reason: "journal query failed: " + err.Error(), GeneratedAt: now}) //nolint:errcheck
		return
	}
	seen := map[string]struct{}{}
	for _, e := range entries {
		switch {
		case e.Unit != "":
			seen[e.Unit] = struct{}{}
		case e.UserUnit != "":
			seen[e.UserUnit] = struct{}{}
		}
	}
	units := make([]string, 0, len(seen))
	for u := range seen {
		units = append(units, u)
	}
	httputil.JSON(w, UnitsResponse{Available: true, Units: units, GeneratedAt: now}) //nolint:errcheck
}

// BootsResponse is the response for GET /api/v1/logs/boots.
type BootsResponse struct {
	Available   bool                 `json:"available"`
	Reason      string               `json:"reason,omitempty"`
	Boots       []journal.BootRecord `json:"boots,omitempty"`
	GeneratedAt time.Time            `json:"generatedAt"`
}

// Boots handles GET /api/v1/logs/boots.
func (h *LogsHandler) Boots(w http.ResponseWriter, r *http.Request) {
	now := h.now().UTC()
	if h.reader == nil {
		httputil.JSON(w, BootsResponse{Reason: "journal reader not configured", GeneratedAt: now}) //nolint:errcheck
		return
	}
	boots, err := h.reader.ListBoots(r.Context())
	if err != nil {
		httputil.JSON(w, BootsResponse{Reason: "list-boots failed: " + err.Error(), GeneratedAt: now}) //nolint:errcheck
		return
	}
	httputil.JSON(w, BootsResponse{Available: true, Boots: boots, GeneratedAt: now}) //nolint:errcheck
}

// normalizeJournalTime accepts either a Go duration ("5m", "1h30m", "30s")
// — interpreted as a window before now — or any other string, which is
// passed through to journalctl --since/--until verbatim. Empty stays empty.
// journalctl rejects bare Go-style durations like "5m"; convert those to
// absolute UTC timestamps so the dashboard's "last 5 minutes" UX works.
func normalizeJournalTime(s string, now time.Time) string {
	if s == "" {
		return ""
	}
	if d, err := time.ParseDuration(s); err == nil && d > 0 {
		// journalctl --since/--until interprets bare timestamps in local
		// time. Pass an RFC3339 with explicit offset so we don't depend
		// on the running process's timezone matching the host's.
		return now.Add(-d).Format(time.RFC3339)
	}
	return s
}

func parseLimit(s string) int {
	if s == "" {
		return logsDefaultLimit
	}
	v, err := strconv.Atoi(s)
	if err != nil || v <= 0 {
		return logsDefaultLimit
	}
	if v > logsMaxLimit {
		return logsMaxLimit
	}
	return v
}
