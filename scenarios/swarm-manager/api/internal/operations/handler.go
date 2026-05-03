package operations

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"

	"github.com/gorilla/mux"
)

// Handler exposes the operations aggregate over HTTP.
type Handler struct {
	aggregator *Aggregator
}

// NewHandler returns a Handler bound to the given aggregator.
func NewHandler(aggregator *Aggregator) *Handler {
	return &Handler{aggregator: aggregator}
}

// RegisterRoutes wires the operations endpoints onto the given router.
//
// Today only GET /api/v1/operations is registered. POST
// /api/v1/operations/bulk-stop lands in P7b.
func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/operations", h.Get).Methods("GET")
}

// Get parses query-string filters and returns the aggregated view.
//
// Query parameters:
//   - window: ISO-8601 duration (e.g. "PT3H"), default PT3H, max PT24H.
//   - status: repeatable; lower-cased match against ActivityRow.Status.
//   - lane: repeatable; lower-cased match against ActivityRow.Lane.
//   - mode: repeatable; matches ActivityRow.Mode (operating-mode runs).
//   - owner_type: repeatable; matches ActivityRow.OwnerType.
//   - q: substring search over OwnerTitle / OwnerName / RunID.
//
// All comparisons are case-insensitive. Repeated keys are read with
// r.URL.Query()[k] which returns []string, so callers can pass either
// `status=running&status=needs_review` or `status[]=running&status[]=needs_review`
// and both forms work — gorilla/mux preserves the bracket suffix in the
// key, so we look up both shapes.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	filters, derr := h.parseFilters(r)
	if derr != nil {
		apierr.MapError(w, "[operations] get", derr)
		return
	}

	view, err := h.aggregator.Aggregate(r.Context(), filters)
	if err != nil {
		apierr.MapError(w, "[operations] get", apierr.Internal("failed to aggregate operations: %v", err))
		return
	}

	if err := httputil.JSON(w, view); err != nil {
		apierr.MapError(w, "[operations] get", apierr.Internal("failed to encode operations response"))
	}
}

// parseFilters reads the query string into a Filters struct. Returns a
// typed apierr.DomainError for validation failures so MapError can render
// 400 with the user-facing message intact.
func (h *Handler) parseFilters(r *http.Request) (Filters, *apierr.DomainError) {
	q := r.URL.Query()

	window := DefaultWindow
	if raw := strings.TrimSpace(q.Get("window")); raw != "" {
		dur, err := parseISO8601Duration(raw)
		if err != nil {
			return Filters{}, apierr.BadRequest("invalid window: %v", err)
		}
		if dur < MinWindow {
			return Filters{}, apierr.BadRequest("window must be at least %s", MinWindow)
		}
		if dur > MaxWindow {
			return Filters{}, apierr.BadRequest("window must not exceed %s", MaxWindow)
		}
		window = dur
	}

	statuses := readRepeated(q, "status")
	lanes := readRepeated(q, "lane")
	modes := readRepeated(q, "mode")
	ownerTypes := readRepeated(q, "owner_type")

	for _, lane := range lanes {
		if !agentactivity.IsValidLane(agentactivity.Lane(strings.ToLower(strings.TrimSpace(lane)))) {
			return Filters{}, apierr.BadRequest("invalid lane %q (expected one of investigate|execute|review|reconcile)", lane)
		}
	}

	return Filters{
		Window:     window,
		Statuses:   statuses,
		Lanes:      lanes,
		Modes:      modes,
		OwnerTypes: ownerTypes,
		Q:          strings.TrimSpace(q.Get("q")),
	}, nil
}

// readRepeated reads both `key` and `key[]` query forms and returns the
// concatenated list with empty entries removed. Lower-cased trim happens
// downstream in matchesFilter.
func readRepeated(q map[string][]string, key string) []string {
	combined := append([]string{}, q[key]...)
	combined = append(combined, q[key+"[]"]...)
	out := make([]string, 0, len(combined))
	for _, v := range combined {
		if t := strings.TrimSpace(v); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// parseISO8601Duration accepts a small subset of ISO-8601 durations
// sufficient for the operations window: PT-prefixed time-only durations
// with hour, minute, and second components (e.g. "PT3H", "PT1H30M",
// "PT45M", "PT90S"). Date components (Y/M/W/D) and decimals are not
// accepted — three hours is the longest unit the operator picks here, so
// the grammar is intentionally tight to surface typos as 400s.
func parseISO8601Duration(s string) (time.Duration, error) {
	upper := strings.ToUpper(strings.TrimSpace(s))
	if !strings.HasPrefix(upper, "PT") {
		return 0, fmt.Errorf("expected ISO-8601 PT-prefixed duration, got %q", s)
	}
	body := upper[2:]
	if body == "" {
		return 0, fmt.Errorf("empty duration body")
	}

	var total time.Duration
	var (
		num     int64
		hasNum  bool
		seenH   bool
		seenM   bool
		seenSec bool
	)
	for _, ch := range body {
		switch {
		case ch >= '0' && ch <= '9':
			num = num*10 + int64(ch-'0')
			hasNum = true
		case ch == 'H':
			if !hasNum {
				return 0, fmt.Errorf("H must follow a number")
			}
			if seenH || seenM || seenSec {
				return 0, fmt.Errorf("H must precede M and S")
			}
			total += time.Duration(num) * time.Hour
			num = 0
			hasNum = false
			seenH = true
		case ch == 'M':
			if !hasNum {
				return 0, fmt.Errorf("M must follow a number")
			}
			if seenM || seenSec {
				return 0, fmt.Errorf("M must precede S and appear at most once")
			}
			total += time.Duration(num) * time.Minute
			num = 0
			hasNum = false
			seenM = true
		case ch == 'S':
			if !hasNum {
				return 0, fmt.Errorf("S must follow a number")
			}
			if seenSec {
				return 0, fmt.Errorf("S must appear at most once")
			}
			total += time.Duration(num) * time.Second
			num = 0
			hasNum = false
			seenSec = true
		default:
			return 0, fmt.Errorf("unexpected character %q in duration", ch)
		}
	}
	if hasNum {
		return 0, fmt.Errorf("trailing number missing unit (H, M, or S)")
	}
	if total <= 0 {
		return 0, fmt.Errorf("duration must be positive")
	}
	return total, nil
}
