// Package durability exposes the swarm-owned evidence read lane. It projects
// status transitions and record lineage without reading narrative outcomes.
package durability

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"

	"swarm-manager/internal/eventlog"
	"swarm-manager/internal/httputil"
	"swarm-manager/internal/records"

	"github.com/gorilla/mux"
)

// recordScanLimit bounds the rework scan. A silently truncated scan would make
// missing evidence look like clean evidence, so hitting it is logged.
const recordScanLimit = 10000

// durabilityEventTypes are the externally-authored signals this lane reads.
// Keep this list and the provenance-carrying emitters in sync: an event type
// added here that still emits without request context reads as unattributed.
var durabilityEventTypes = []eventlog.EventType{
	eventlog.EventBacklogStatusChanged,
	eventlog.EventReviewFailed,
	eventlog.EventRecordSuperseded,
}

type Evidence struct {
	Kind      string `json:"kind"`
	Reference string `json:"reference"`
	At        string `json:"at"`
	Lane      string `json:"lane"`
}

type Handler struct {
	events  eventlog.Repository
	records records.Store
}

func NewHandler(events eventlog.Repository, store records.Store) *Handler {
	return &Handler{events: events, records: store}
}

func (h *Handler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/v1/durability/evidence", h.get).Methods(http.MethodGet)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.events == nil || h.records == nil {
		writeError(w, http.StatusServiceUnavailable, "durability evidence is unavailable")
		return
	}
	runID := strings.TrimSpace(r.URL.Query().Get("run_id"))
	subjects := splitValues(r.URL.Query().Get("subjects"))
	startedAt, err := parseTime(r.URL.Query().Get("started_at"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	evidence, err := h.read(r.Context(), runID, subjects, startedAt)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	_ = httputil.JSON(w, map[string]any{"evidence": evidence})
}

func (h *Handler) read(ctx context.Context, runID string, subjects []string, startedAt time.Time) ([]Evidence, error) {
	all, err := h.events.QueryByTypesSince(ctx, durabilityEventTypes, startedAt)
	if err != nil {
		return nil, err
	}
	out := make([]Evidence, 0)
	for _, event := range all {
		if !eventMatches(event, runID, subjects) {
			continue
		}
		kind := "pushback"
		if event.EventType == eventlog.EventRecordSuperseded {
			kind = "rework"
		} else if event.EventType == eventlog.EventBacklogStatusChanged && !pushbackTransition(event.Metadata) {
			continue
		}
		out = append(out, Evidence{Kind: kind, Reference: "swarm-manager://events/" + strings.TrimSpace(event.EntityID) + "/" + string(event.EventType), At: event.Timestamp.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"), Lane: eventLane(event.VerificationStatus, event.HarnessSessionID)})
	}
	// Filter by kind in the store rather than the loop. Listing every kind under
	// a flat cap silently dropped most of the corpus before the fix records were
	// even reached, which reads as "no rework found" instead of "not looked".
	recs, err := h.records.List(records.ListFilter{Kind: records.KindFix, IncludeStubs: true, Limit: recordScanLimit})
	if err != nil {
		return nil, err
	}
	if len(recs) == recordScanLimit {
		slog.Warn("durability record scan hit its limit; rework evidence may be incomplete", "limit", recordScanLimit)
	}
	for _, record := range recs {
		if record.Kind != records.KindFix || !recordMatches(record, runID, subjects) || (!startedAt.IsZero() && !record.CreatedAt.After(startedAt)) {
			continue
		}
		out = append(out, Evidence{Kind: "rework", Reference: "swarm-manager://records/" + record.ID, At: record.CreatedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"), Lane: recordsLane(record)})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].At < out[j].At })
	return out, nil
}

func pushbackTransition(raw json.RawMessage) bool {
	var payload eventlog.StatusChangePayload
	if json.Unmarshal(raw, &payload) != nil {
		return false
	}
	for _, status := range []string{payload.From, payload.To} {
		switch strings.ToLower(strings.TrimSpace(status)) {
		case "failed", "rejected", "blocked", "review_pending", "needs_changes":
			return true
		}
	}
	return false
}

func eventMatches(event eventlog.Event, runID string, subjects []string) bool {
	if runID != "" && strings.Contains(string(event.Metadata), runID) {
		return true
	}
	for _, subject := range subjects {
		if strings.Contains(event.EntityID, subject) || strings.Contains(string(event.Metadata), subject) {
			return true
		}
	}
	return runID == "" && len(subjects) == 0
}

// recordMatches reports whether a fix record concerns the same work as the
// requested subjects.
//
// Subjects are "kind:value" tokens (path:scenarios/x/y.go, tool:swarm-manager),
// so the record's own identifiers are looked for INSIDE the token value. The
// reverse comparison asked whether a scenario name contained a whole subject
// token and therefore never matched anything.
//
// An empty subject set is scoped by run: an unscoped browse still returns the
// corpus, but a request scoped to one run matches nothing rather than every fix
// record. Matching everything turned an unsearchable run into "rework found";
// matching nothing lets the caller report that no subject was searchable, which
// is what actually happened. This mirrors eventMatches' convention.
func recordMatches(record records.Record, runID string, subjects []string) bool {
	if len(subjects) == 0 {
		return runID == ""
	}
	scenario := strings.TrimSpace(record.Scenario)
	backlogRef := strings.TrimSpace(record.BacklogRef)
	for _, subject := range subjects {
		value := subjectValue(subject)
		if value == "" {
			continue
		}
		if scenario != "" && mentionsSegment(value, scenario) {
			return true
		}
		if backlogRef != "" && strings.Contains(value, backlogRef) {
			return true
		}
	}
	return false
}

// subjectValue strips the "kind:" prefix from a subject token, keeping the
// value that identifies the work — a repository path or an invoked command.
func subjectValue(subject string) string {
	subject = strings.TrimSpace(subject)
	if _, value, found := strings.Cut(subject, ":"); found {
		return strings.TrimSpace(value)
	}
	return subject
}

// mentionsSegment reports whether name appears as a whole path or word segment
// of value. Whole-segment matching keeps a short scenario name from matching an
// unrelated substring, and keeps "agent-manager" from matching a sibling like
// "agent-manager-retired". Splitting on spaces as well as slashes lets a
// command subject ("tool:swarm-manager backlog list") match its owning scenario.
func mentionsSegment(value, name string) bool {
	segments := strings.FieldsFunc(value, func(r rune) bool {
		return r == '/' || r == ' ' || r == '\t'
	})
	for _, segment := range segments {
		if strings.EqualFold(segment, name) {
			return true
		}
	}
	return false
}

func eventLane(verification, session string) string {
	if strings.EqualFold(strings.TrimSpace(verification), "verified") {
		return "verified"
	}
	if strings.TrimSpace(session) != "" {
		return "observed"
	}
	return "unlinked"
}

func recordsLane(record records.Record) string {
	if strings.TrimSpace(record.CreatedBy) != "" {
		return "observed"
	}
	return "unlinked"
}

func splitValues(raw string) []string {
	var values []string
	if strings.TrimSpace(raw) != "" && json.Unmarshal([]byte(raw), &values) == nil {
		return values
	}
	for _, value := range strings.Split(raw, ",") {
		if value = strings.TrimSpace(value); value != "" {
			values = append(values, value)
		}
	}
	return values
}

func parseTime(raw string) (time.Time, error) {
	if strings.TrimSpace(raw) == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("started_at must be RFC3339: %w", err)
	}
	return parsed, nil
}

func writeError(w http.ResponseWriter, status int, message string) {
	_ = httputil.JSONWithStatus(w, status, map[string]any{"error": message})
}
