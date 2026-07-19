package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/api-core/provenance"
	domain "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/broker"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/convert"
	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/store"
	"google.golang.org/protobuf/encoding/protojson"
)

var (
	protoMarshaler   = protojson.MarshalOptions{EmitUnpopulated: false}
	protoUnmarshaler = protojson.UnmarshalOptions{DiscardUnknown: true}
)

// --- Error codes (centralized so new error categories are added in one place) ---

const (
	ErrCodeReadError      = "READ_ERROR"
	ErrCodeInvalidBody    = "INVALID_BODY"
	ErrCodeMissingField   = "MISSING_FIELD"
	ErrCodeConversion     = "CONVERSION_ERROR"
	ErrCodeStoreWrite     = "STORE_ERROR"
	ErrCodeDuplicate      = "DUPLICATE_EVENT"
	ErrCodeStoreRead      = "QUERY_ERROR"
	ErrCodeMarshal        = "MARSHAL_ERROR"
	ErrCodeSSEUnsupported = "SSE_UNSUPPORTED"
	ErrCodeInvalidParam   = "INVALID_PARAM"
)

const receiptEventType = "vrooli.receipt.observed.v1"

// validateReceipt keeps the generic event store safe for the platform receipt
// projection. Receipts are still ordinary durable events, but their bounded
// safe projection and correlation fields are validated centrally rather than
// trusted as arbitrary scenario metadata.
func validateReceipt(env *domain.EventEnvelope) *envelopeValidationError {
	if env.EventType != receiptEventType {
		return nil
	}
	if strings.TrimSpace(env.CorrelationId) == "" {
		return &envelopeValidationError{Field: "correlationId", Message: "receipt correlationId (verified run id) is required"}
	}
	if env.Metadata == nil || strings.TrimSpace(env.Metadata["operation"]) == "" {
		return &envelopeValidationError{Field: "metadata.operation", Message: "receipt operation is required"}
	}
	if strings.TrimSpace(env.Metadata["outcome"]) == "" {
		return &envelopeValidationError{Field: "metadata.outcome", Message: "receipt outcome is required"}
	}
	if projection := env.Metadata["safe_projection"]; len(projection) > 64*1024 {
		return &envelopeValidationError{Field: "metadata.safe_projection", Message: "receipt safe projection exceeds 64KiB"}
	}
	return nil
}

// validateReceiptProjection ensures a verified producer cannot bypass the
// centrally declared receipt allow-list by placing arbitrary values in
// safe_projection. The service, not scenario code, remains the authority.
func (s *Server) validateReceiptProjection(ctx context.Context, env *domain.EventEnvelope) *envelopeValidationError {
	if env.EventType != receiptEventType {
		return nil
	}
	rule, err := s.policyStore.MatchReceiptProjection(ctx, env.SourceScenario, env.TargetScenario, env.Metadata["operation"])
	if err != nil {
		return &envelopeValidationError{Field: "receipt_projection", Message: "failed to evaluate receipt projection policy"}
	}
	if rule == nil {
		return &envelopeValidationError{Field: "receipt_projection", Message: "no approved receipt projection rule matches this operation"}
	}
	if rule.SamplePerTenK == 0 || (rule.SamplePerTenK < 10000 && receiptSample(env.EventId) >= rule.SamplePerTenK) {
		return &envelopeValidationError{Field: "receipt_projection", Message: "receipt excluded by projection sampling policy"}
	}
	retentionDays, err := strconv.Atoi(env.Metadata["retention_days"])
	if err != nil || retentionDays != rule.RetentionDays {
		return &envelopeValidationError{Field: "metadata.retention_days", Message: "receipt retention must match the approved projection policy"}
	}
	projection := env.Metadata["safe_projection"]
	if len(projection) > rule.MaxBytes {
		return &envelopeValidationError{Field: "metadata.safe_projection", Message: "receipt safe projection exceeds approved policy limit"}
	}
	if projection == "" {
		return nil
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(projection), &fields); err != nil {
		return &envelopeValidationError{Field: "metadata.safe_projection", Message: "receipt safe projection must be a JSON object"}
	}
	allowed := make(map[string]bool, len(rule.ResponseFields))
	for _, field := range rule.ResponseFields {
		allowed[field] = true
	}
	for _, field := range rule.RedactFields {
		delete(allowed, field)
	}
	for field := range fields {
		if !allowed[field] {
			return &envelopeValidationError{Field: "metadata.safe_projection", Message: "receipt safe projection contains a field not approved by policy"}
		}
	}
	return nil
}

func receiptSample(eventID string) int {
	var sum uint32 = 2166136261
	for i := 0; i < len(eventID); i++ {
		sum = (sum ^ uint32(eventID[i])) * 16777619
	}
	return int(sum % 10000)
}

// envelopeValidationError describes a missing or invalid field in an inbound EventEnvelope.
// Keeping validation rules as data (not inline if-chains) means adding a new required
// field is a single-line table entry rather than a scattered code change.
type envelopeValidationError struct {
	Field   string
	Message string
}

// validateEnvelope checks that the EventEnvelope carries the minimum required fields.
// Returns nil when valid, or a description of the first violated rule.
func validateEnvelope(env *domain.EventEnvelope) *envelopeValidationError {
	rules := []struct {
		value   string
		field   string
		message string
	}{
		{env.EventId, "eventId", "eventId is required"},
		{env.EventType, "eventType", "eventType is required"},
		{env.SourceScenario, "sourceScenario", "sourceScenario is required"},
	}
	for _, r := range rules {
		if r.value == "" {
			return &envelopeValidationError{Field: r.field, Message: r.message}
		}
	}
	return nil
}

// parseQueryFilters builds store.QueryFilters from URL query parameters.
// Returns a non-nil error string (code + message) if a parameter is malformed.
func parseQueryFilters(q map[string][]string) (store.QueryFilters, *paramError) {
	get := func(key string) string {
		if vs, ok := q[key]; ok && len(vs) > 0 {
			return vs[0]
		}
		return ""
	}

	filters := store.QueryFilters{
		EventType:     get("type"),
		Source:        get("source"),
		Target:        get("target"),
		CorrelationID: get("correlation_id"),
	}

	if v := get("since"); v != "" {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return filters, &paramError{Param: "since", Message: "since must be an integer"}
		}
		filters.Since = n
	}
	if v := get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return filters, &paramError{Param: "limit", Message: "limit must be an integer"}
		}
		filters.Limit = n
	}

	return filters, nil
}

// paramError describes a malformed query parameter.
type paramError struct {
	Param   string
	Message string
}

// isDryRun checks whether the request carries the X-Dry-Run header.
func isDryRun(r *http.Request) bool {
	return r.Header.Get("X-Dry-Run") == "true"
}

// DOC: docs/reference/api-endpoints.md#event-ingestion
// DOC: docs/internal/ERROR-SEMANTICS.md
func (s *Server) handleIngest(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, s.config.MaxBodyBytes))
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeReadError, "failed to read request body")
		return
	}

	var env domain.EventEnvelope
	if err := protoUnmarshaler.Unmarshal(body, &env); err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidBody, "invalid request body: "+err.Error())
		return
	}

	if ve := validateEnvelope(&env); ve != nil {
		writeError(w, http.StatusBadRequest, ErrCodeMissingField, ve.Message)
		return
	}
	if ve := validateReceipt(&env); ve != nil {
		writeError(w, http.StatusBadRequest, ErrCodeMissingField, ve.Message)
		return
	}
	if env.EventType == receiptEventType && !provenance.FromContext(r.Context()).IsVerifiedAgent() {
		p := provenance.FromContext(r.Context())
		log.Printf("rejected receipt without verified Agent Manager provenance: actor=%s verification=%s", p.Actor, p.VerificationStatus)
		writeError(w, http.StatusUnauthorized, ErrCodeValidation, "receipt run correlation requires verified Agent Manager identity")
		return
	}
	if ve := s.validateReceiptProjection(r.Context(), &env); ve != nil {
		writeError(w, http.StatusBadRequest, ErrCodeValidation, ve.Message)
		return
	}

	event, err := convert.EnvelopeToEvent(&env)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeConversion, err.Error())
		return
	}
	if env.EventType == receiptEventType {
		retentionDays, _ := strconv.Atoi(env.Metadata["retention_days"])
		if retentionDays > 0 {
			expiresAt := time.Now().UTC().Add(time.Duration(retentionDays) * 24 * time.Hour)
			event.ExpiresAt = &expiresAt
		}
	}

	// Dry-run: full validation passes, but skip persistence and broadcast.
	if isDryRun(r) {
		writeJSON(w, 0, map[string]any{
			"dry_run":        true,
			"eventId":        env.EventId,
			"eventType":      env.EventType,
			"sourceScenario": env.SourceScenario,
		})
		return
	}

	id, err := s.store.Insert(r.Context(), event)
	if err != nil {
		if errors.Is(err, store.ErrDuplicateEvent) {
			writeError(w, http.StatusConflict, ErrCodeDuplicate, fmt.Sprintf("event_id %q already exists", env.EventId))
			return
		}
		writeError(w, http.StatusInternalServerError, ErrCodeStoreWrite, "failed to store event")
		log.Printf("insert error: %v", err)
		return
	}

	event.ID = id

	// Broadcast asynchronously so ingestion latency is not affected by slow subscribers.
	go func() {
		envOut, err := convert.EventToEnvelope(event)
		if err != nil {
			log.Printf("convert for broadcast: %v", err)
			return
		}
		data, err := protoMarshaler.Marshal(envOut)
		if err != nil {
			log.Printf("marshal for broadcast: %v", err)
			return
		}
		s.broker.Publish(event, string(data))
	}()

	writeJSON(w, http.StatusAccepted, map[string]any{"id": id, "eventId": env.EventId})
}

// DOC: docs/reference/api-endpoints.md#event-query
func (s *Server) handleQuery(w http.ResponseWriter, r *http.Request) {
	filters, pe := parseQueryFilters(r.URL.Query())
	if pe != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidParam, pe.Message)
		return
	}

	events, err := s.store.Query(r.Context(), filters)
	if err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodeStoreRead, "failed to query events")
		log.Printf("query error: %v", err)
		return
	}

	writeEventList(w, events)
}

// writeEventList serializes a slice of store.Event as a JSON array of proto EventEnvelopes.
func writeEventList(w http.ResponseWriter, events []store.Event) {
	w.Header().Set("Content-Type", "application/json")

	if len(events) == 0 {
		_, _ = w.Write([]byte("[]"))
		return
	}

	_, _ = w.Write([]byte("["))
	for i, e := range events {
		if i > 0 {
			_, _ = w.Write([]byte(","))
		}
		env, err := convert.EventToEnvelope(e)
		if err != nil {
			writeError(w, http.StatusInternalServerError, ErrCodeConversion, err.Error())
			return
		}
		data, err := protoMarshaler.Marshal(env)
		if err != nil {
			writeError(w, http.StatusInternalServerError, ErrCodeMarshal, err.Error())
			return
		}
		_, _ = w.Write(data)
	}
	_, _ = w.Write([]byte("]"))
}

// DOC: docs/reference/api-endpoints.md#sse-subscribe
// DOC: docs/internal/TEMPORAL-FLOWS.md
func (s *Server) handleSubscribe(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, ErrCodeSSEUnsupported, "streaming not supported")
		return
	}

	q := r.URL.Query()
	opts := broker.SubscribeOpts{
		EventTypePattern:      q.Get("type"),
		SourceScenarioPattern: q.Get("source"),
		TargetScenarioPattern: q.Get("target"),
	}

	ch, subCtx, cleanup := s.broker.Subscribe(r.Context(), opts)
	defer cleanup()

	writeSSEHeaders(w, flusher, s.config.SSERetryMs)

	// Replay missed events so subscribers that reconnect (via Last-Event-ID)
	// don't lose events that arrived while they were disconnected.
	s.replayMissedEvents(w, r, flusher)

	// Stream live events until the client disconnects or the broker shuts down.
	s.streamLiveEvents(w, flusher, ch, subCtx, r.Context())
}

// replayMissedEvents sends stored events newer than the client's Last-Event-ID,
// enabling gap-free reconnection for SSE subscribers.
func (s *Server) replayMissedEvents(w http.ResponseWriter, r *http.Request, flusher http.Flusher) {
	lastIDStr := r.Header.Get("Last-Event-ID")
	if lastIDStr == "" {
		return
	}

	lastID, err := strconv.ParseInt(lastIDStr, 10, 64)
	if err != nil {
		log.Printf("replay: invalid Last-Event-ID %q: %v", lastIDStr, err)
		return
	}

	events, err := s.store.GetSince(r.Context(), lastID, s.config.ReplayLimit)
	if err != nil {
		log.Printf("replay: GetSince(lastID=%d) failed: %v", lastID, err)
		return
	}

	replayed, skipped := 0, 0
	for _, e := range events {
		env, err := convert.EventToEnvelope(e)
		if err != nil {
			skipped++
			log.Printf("replay: convert event %d: %v", e.ID, err)
			continue
		}
		data, err := protoMarshaler.Marshal(env)
		if err != nil {
			skipped++
			log.Printf("replay: marshal event %d: %v", e.ID, err)
			continue
		}
		fmt.Fprintf(w, "id: %d\nevent: %s\ndata: %s\n\n", e.ID, e.EventType, data)
		replayed++
	}
	if skipped > 0 {
		log.Printf("replay: %d replayed, %d skipped (from Last-Event-ID=%d)", replayed, skipped, lastID)
	}
	flusher.Flush()
}

// streamLiveEvents is the main SSE event loop. It writes heartbeats (as SSE
// comments) and data events until one of the contexts is cancelled.
func (s *Server) streamLiveEvents(w http.ResponseWriter, flusher http.Flusher, ch <-chan broker.SSEMessage, subCtx, reqCtx context.Context) {
	for {
		select {
		case <-subCtx.Done():
			return
		case <-reqCtx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			writeSSEMessage(w, msg)
			flusher.Flush()
		}
	}
}

// writeSSEHeaders sets the standard SSE response headers and writes the retry
// directive. Both event and policy SSE endpoints share this setup.
func writeSSEHeaders(w http.ResponseWriter, flusher http.Flusher, retryMs int) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // nginx-specific: disable proxy buffering
	fmt.Fprintf(w, "retry: %d\n\n", retryMs)
	flusher.Flush()
}

// writeSSEMessage formats a single SSE frame. Heartbeats are written as SSE
// comments (lines starting with ":") so they don't trigger client-side event
// handlers but still keep the TCP connection alive.
func writeSSEMessage(w http.ResponseWriter, msg broker.SSEMessage) {
	if msg.Event == "heartbeat" {
		if msg.Data != "" {
			fmt.Fprintf(w, ": heartbeat %s\n\n", msg.Data)
		} else {
			fmt.Fprintf(w, ": heartbeat\n\n")
		}
	} else {
		fmt.Fprintf(w, "id: %d\nevent: %s\ndata: %s\n\n", msg.ID, msg.Event, msg.Data)
	}
}

// DOC: docs/reference/api-endpoints.md#health
// handleHealth reports store reachability and key counters. Returning 503 when
// the store is unreachable lets load balancers route traffic away from this instance.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC().Format(time.RFC3339)

	stats, err := s.store.Stats(r.Context())
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"status":    "unhealthy",
			"service":   "vrooli-events",
			"timestamp": now,
			"readiness": false,
			"error":     err.Error(),
		})
		return
	}

	writeJSON(w, 0, map[string]any{
		"status":      "healthy",
		"service":     "vrooli-events",
		"timestamp":   now,
		"readiness":   true,
		"subscribers": s.broker.SubscriberCount(),
		"store": map[string]any{
			"totalEvents":       stats.TotalEvents,
			"totalPayloadBytes": stats.TotalPayloadBytes,
		},
	})
}

// orEmpty returns s if non-nil, otherwise an empty slice of the same type.
// Ensures JSON serialization produces "[]" instead of "null" for list endpoints.
func orEmpty[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

// writeJSON sends a JSON response with the given status code.
// Centralizes Content-Type header and encoding so handlers stay focused on logic.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	if status != 0 {
		w.WriteHeader(status)
	}
	_ = json.NewEncoder(w).Encode(v)
}

// writeError sends a structured JSON error response.
// All error responses share the same shape {error, code} so clients can
// switch on the machine-readable code field rather than parsing messages.
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]string{
		"error": message,
		"code":  code,
	})
}

// decodeJSONBody reads and JSON-decodes the request body into v.
// Returns true on success; on failure it writes a 400 error and returns false.
func decodeJSONBody(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidBody, "invalid request body: "+err.Error())
		return false
	}
	return true
}

// requireByID parses a path parameter as an int64 ID, fetches the entity via
// fetch, and handles 400/404/500 error responses. On success it returns the ID,
// entity, and true. Callers can skip to business logic without the repeated
// parse → fetch → ErrNoRows boilerplate.
func requireByID[T any](w http.ResponseWriter, r *http.Request, param string, fetch func(context.Context, int64) (T, error), readCode, entity string) (int64, T, bool) {
	var zero T
	id, err := parsePathID(r, param)
	if err != nil {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidParam, "invalid "+entity+" ID")
		return 0, zero, false
	}
	val, err := fetch(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, ErrCodeNotFound, entity+" not found")
			return 0, zero, false
		}
		writeError(w, http.StatusInternalServerError, readCode, "failed to get "+entity)
		log.Printf("%s get error: %v", entity, err)
		return 0, zero, false
	}
	return id, val, true
}
