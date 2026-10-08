package hub

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/eventbus"
)

// Ask states. pending and escalated are open; answered, defaulted, and
// expired are resolved. A defaulted ask still accepts one late operator
// answer, which supersedes the default.
const (
	AskPending   = "pending"
	AskEscalated = "escalated"
	AskAnswered  = "answered"
	AskDefaulted = "defaulted"
	AskExpired   = "expired"
)

const (
	// DefaultAskUrgency keeps asks inside quiet hours unless a requester
	// explicitly asks for critical.
	DefaultAskUrgency = "high"
	// DefaultAskDefaultFloor is the minimum time between the first delivered
	// receipt and applying a default answer.
	DefaultAskDefaultFloor = 12 * time.Hour
	// AskResolvedEventType tells the requester an ask was answered, defaulted,
	// or expired. It is published once per transition.
	AskResolvedEventType = "notification_hub.ask.resolved.v1"
	// DefaultAnsweredBy marks an answer the sweep applied.
	DefaultAnsweredBy = "default"

	maxAskOptions        = 8
	maxAskOptionLabel    = 200
	maxAskCorrelationLen = 1024
	operatorNotReached   = "operator not reached: no delivered receipt yet; default withheld"
)

var (
	ErrAskResolved   = errors.New("ask is already resolved")
	askOptionKeyRule = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,63}$`)
	validUrgencies   = map[string]bool{"low": true, "normal": true, "high": true, "critical": true}
)

// AskOption is one structured answer. Key is what the requester receives;
// Label is what the operator sees.
type AskOption struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

// AskSpec is the full ask contract. Recipient, Question, Options, Deadline,
// SensitivityLabel, and IdempotencyKey are required; everything else is
// optional.
type AskSpec struct {
	Recipient            string
	Title                string
	Body                 string
	Question             string
	Options              []AskOption
	Recommended          string
	RecommendationReason string
	DefaultAnswer        string
	Reversible           bool
	Urgency              string
	ContextURL           string
	Deadline             time.Time
	SensitivityLabel     string
	IdempotencyKey       string
	// Source names the requester (event source scenario, or the owner
	// subject for direct asks). Together with IdempotencyKey it identifies
	// one ask, so a redelivered event never creates a second ask.
	Source          string
	SourceEventType string
	// Correlation is echoed verbatim in the ask-resolved event so the
	// requester can bind the answer without a lookup.
	Correlation map[string]string
}

// AskRecord is the read model of one ask. Reading it never changes it.
type AskRecord struct {
	ID                   string
	NotificationID       string
	Recipient            string
	Question             string
	Options              []AskOption
	Recommended          string
	RecommendationReason string
	DefaultAnswer        string
	Reversible           bool
	Urgency              string
	ContextURL           string
	Deadline             string
	State                string
	Reason               string
	Answer               string
	AnswerLabel          string
	Note                 string
	AnsweredBy           string
	AnsweredAt           string
	Late                 bool
	DefaultAppliedAt     string
	FirstDeliveredAt     string
	Source               string
	SourceEventType      string
	RequesterRef         string
	Correlation          map[string]string
	ResolvedAt           string
	CreatedAt            string
	UpdatedAt            string
}

// Open reports whether the ask still waits for an answer.
func (r AskRecord) Open() bool { return r.State == AskPending || r.State == AskEscalated }

// AnswerInput records an operator answer. Note is stored verbatim.
type AnswerInput struct {
	AskID, Answer, Actor, Note string
}

// ResolutionPublisher is the Vrooli Events publisher seam
// (api-core/eventbus.Client in production).
type ResolutionPublisher interface {
	PublishDomainEvent(context.Context, eventbus.DomainEvent) error
}

type askResolutions struct {
	mu        sync.Mutex
	publisher ResolutionPublisher
	running   bool
	// drain serializes outbox drains so the sweep and a post-answer kick never
	// publish the same resolution twice.
	drain sync.Mutex
}

// SetResolutionPublisher wires the ask-resolved event publisher. Without
// one, resolutions stay in the outbox until a publisher is set.
func (s *Service) SetResolutionPublisher(publisher ResolutionPublisher) {
	s.resolutions.mu.Lock()
	s.resolutions.publisher = publisher
	s.resolutions.mu.Unlock()
}

// SetAskDefaultFloor sets the minimum time between the first delivered
// receipt and applying a default. Values <= 0 restore the 12 h floor.
func (s *Service) SetAskDefaultFloor(floor time.Duration) {
	if floor <= 0 {
		floor = DefaultAskDefaultFloor
	}
	s.mu.Lock()
	s.askDefaultFloor = floor
	s.mu.Unlock()
}

func (s *Service) defaultFloor() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.askDefaultFloor <= 0 {
		return DefaultAskDefaultFloor
	}
	return s.askDefaultFloor
}

func normalizeAskSpec(spec AskSpec) (AskSpec, error) {
	spec.Recipient = strings.TrimSpace(spec.Recipient)
	spec.Question = strings.TrimSpace(spec.Question)
	spec.IdempotencyKey = strings.TrimSpace(spec.IdempotencyKey)
	spec.SensitivityLabel = strings.TrimSpace(spec.SensitivityLabel)
	spec.Recommended = strings.TrimSpace(spec.Recommended)
	spec.DefaultAnswer = strings.TrimSpace(spec.DefaultAnswer)
	spec.Urgency = strings.ToLower(strings.TrimSpace(spec.Urgency))
	if spec.Deadline.IsZero() || len(spec.Options) == 0 || spec.Question == "" {
		return spec, fmt.Errorf("%w: question, deadline, and allowed answers are required", ErrInvalidArgument)
	}
	if len(spec.Options) > maxAskOptions {
		return spec, fmt.Errorf("%w: at most %d answers are allowed", ErrInvalidArgument, maxAskOptions)
	}
	seen := make(map[string]bool, len(spec.Options))
	options := make([]AskOption, 0, len(spec.Options))
	for _, option := range spec.Options {
		key := strings.TrimSpace(option.Key)
		label := strings.TrimSpace(option.Label)
		if !askOptionKeyRule.MatchString(key) {
			return spec, fmt.Errorf("%w: answer key %q must be 1-64 letters, digits, '.', '_', ':' or '-'", ErrInvalidArgument, key)
		}
		if seen[key] {
			return spec, fmt.Errorf("%w: answer key %q is repeated", ErrInvalidArgument, key)
		}
		seen[key] = true
		if label == "" {
			label = key
		}
		if len(label) > maxAskOptionLabel {
			return spec, fmt.Errorf("%w: answer label for %q exceeds %d characters", ErrInvalidArgument, key, maxAskOptionLabel)
		}
		options = append(options, AskOption{Key: key, Label: label})
	}
	spec.Options = options
	if spec.Recommended != "" && !seen[spec.Recommended] {
		return spec, fmt.Errorf("%w: recommended answer %q is not one of the answers", ErrInvalidArgument, spec.Recommended)
	}
	if spec.DefaultAnswer != "" {
		if !seen[spec.DefaultAnswer] {
			return spec, fmt.Errorf("%w: default answer %q is not one of the answers", ErrInvalidArgument, spec.DefaultAnswer)
		}
		if !spec.Reversible {
			return spec, fmt.Errorf("%w: a default answer is allowed only for a reversible decision", ErrInvalidArgument)
		}
	}
	if spec.Urgency == "" {
		spec.Urgency = DefaultAskUrgency
	}
	if !validUrgencies[spec.Urgency] {
		return spec, fmt.Errorf("%w: urgency must be low, normal, high, or critical", ErrInvalidArgument)
	}
	if spec.Source == "" {
		spec.Source = spec.Recipient
	}
	if spec.Title == "" {
		spec.Title = "Decision required"
	}
	if spec.Body == "" {
		spec.Body = renderAskBody(spec)
	}
	return spec, nil
}

// renderAskBody is the hub-owned copy for a structured ask: the question,
// then the recommendation and the default, so the push alone is enough to
// decide.
func renderAskBody(spec AskSpec) string {
	lines := []string{spec.Question}
	if spec.Recommended != "" {
		line := "Recommended: " + optionLabel(spec.Options, spec.Recommended)
		if reason := strings.TrimSpace(spec.RecommendationReason); reason != "" {
			line += " — " + reason
		}
		lines = append(lines, line)
	}
	if spec.DefaultAnswer != "" {
		lines = append(lines, "Default if unanswered: "+optionLabel(spec.Options, spec.DefaultAnswer)+" (not before "+spec.Deadline.UTC().Format("Jan 2 15:04 MST")+")")
	}
	return strings.Join(lines, "\n")
}

func optionLabel(options []AskOption, key string) string {
	for _, option := range options {
		if option.Key == key {
			return option.Label
		}
	}
	return key
}

// AskWithSpec opens an ask. It is idempotent on (Source, IdempotencyKey):
// a repeated request returns the existing ask and sends nothing new.
func (s *Service) AskWithSpec(ctx context.Context, spec AskSpec) (AskRecord, Notification, error) {
	spec, err := normalizeAskSpec(spec)
	if err != nil {
		return AskRecord{}, Notification{}, err
	}
	if existing, err := s.askByRequester(ctx, spec.Source, spec.IdempotencyKey); err == nil {
		n, _, getErr := s.Get(ctx, existing.NotificationID)
		return existing, n, getErr
	} else if !errors.Is(err, ErrNotFound) {
		return AskRecord{}, Notification{}, err
	}
	n, created, err := s.persistNotification(ctx, SendInput{RequestedBy: spec.Recipient, Title: spec.Title, Body: spec.Body, Urgency: spec.Urgency, SensitivityLabel: spec.SensitivityLabel, IdempotencyKey: spec.IdempotencyKey})
	if err != nil {
		return AskRecord{}, Notification{}, err
	}
	if !created {
		if existing, lookupErr := s.askByNotification(ctx, n.ID); lookupErr == nil {
			return existing, n, nil
		}
	}
	allowed := make([]string, 0, len(spec.Options))
	for _, option := range spec.Options {
		allowed = append(allowed, option.Key)
	}
	allowedJSON, _ := json.Marshal(allowed)
	optionsJSON, _ := json.Marshal(spec.Options)
	correlationJSON := ""
	if len(spec.Correlation) > 0 {
		encoded, _ := json.Marshal(spec.Correlation)
		if len(encoded) > maxAskCorrelationLen {
			return AskRecord{}, Notification{}, fmt.Errorf("%w: correlation exceeds %d bytes", ErrInvalidArgument, maxAskCorrelationLen)
		}
		correlationJSON = string(encoded)
	}
	id := uuid.NewString()
	now := s.clock.Now().UTC().Format(time.RFC3339Nano)
	if _, err := s.db.ExecContext(ctx, `INSERT INTO asks (id, notification_id, question, allowed_answers, deadline, state, reason, created_at, updated_at, options_json, recommended, recommendation_reason, default_answer, reversible, urgency, context_url, source, source_event_type, requester_ref, correlation_json) VALUES (?, ?, ?, ?, ?, 'pending', '', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, n.ID, spec.Question, string(allowedJSON), spec.Deadline.UTC().Format(time.RFC3339Nano), now, now, string(optionsJSON), spec.Recommended, strings.TrimSpace(spec.RecommendationReason), spec.DefaultAnswer, boolInt(spec.Reversible), spec.Urgency, strings.TrimSpace(spec.ContextURL), spec.Source, strings.TrimSpace(spec.SourceEventType), spec.IdempotencyKey, correlationJSON); err != nil {
		if existing, lookupErr := s.askByNotification(ctx, n.ID); lookupErr == nil {
			return existing, n, nil
		}
		return AskRecord{}, Notification{}, fmt.Errorf("persist ask: %w", err)
	}
	// The ask row exists before routing starts, so the push can deep-link to
	// the ask page.
	if created {
		s.dispatch(n.ID)
	}
	record, err := s.GetAsk(ctx, id)
	return record, n, err
}

const askSelect = `SELECT a.id, a.notification_id, a.question, a.allowed_answers, a.options_json, a.recommended, a.recommendation_reason, a.default_answer, a.reversible, a.urgency, a.context_url, a.source, a.source_event_type, a.requester_ref, a.correlation_json, a.deadline, a.state, a.reason, a.default_applied_at, a.resolved_at, a.created_at, a.updated_at,
  COALESCE(ans.answer, ''), COALESCE(ans.answered_by, ''), COALESCE(ans.answered_at, ''), COALESCE(ans.note, ''), COALESCE(ans.late, 0),
  COALESCE((SELECT MIN(r.delivered_at) FROM receipts r WHERE r.notification_id = a.notification_id), ''),
  COALESCE((SELECT n.requested_by FROM notifications n WHERE n.id = a.notification_id), '')
FROM asks a LEFT JOIN answers ans ON ans.ask_id = a.id`

func scanAsk(row rowScanner) (AskRecord, error) {
	var record AskRecord
	var allowedJSON, optionsJSON, correlationJSON string
	var reversible, late int
	err := row.Scan(&record.ID, &record.NotificationID, &record.Question, &allowedJSON, &optionsJSON, &record.Recommended, &record.RecommendationReason, &record.DefaultAnswer, &reversible, &record.Urgency, &record.ContextURL, &record.Source, &record.SourceEventType, &record.RequesterRef, &correlationJSON, &record.Deadline, &record.State, &record.Reason, &record.DefaultAppliedAt, &record.ResolvedAt, &record.CreatedAt, &record.UpdatedAt,
		&record.Answer, &record.AnsweredBy, &record.AnsweredAt, &record.Note, &late, &record.FirstDeliveredAt, &record.Recipient)
	if errors.Is(err, sql.ErrNoRows) {
		return AskRecord{}, ErrNotFound
	}
	if err != nil {
		return AskRecord{}, err
	}
	record.Reversible = reversible != 0
	record.Late = late != 0
	if strings.TrimSpace(optionsJSON) != "" {
		_ = json.Unmarshal([]byte(optionsJSON), &record.Options)
	}
	if len(record.Options) == 0 {
		// Asks written before labels existed: the key is the label.
		var allowed []string
		_ = json.Unmarshal([]byte(allowedJSON), &allowed)
		for _, key := range allowed {
			record.Options = append(record.Options, AskOption{Key: key, Label: key})
		}
	}
	if strings.TrimSpace(correlationJSON) != "" {
		_ = json.Unmarshal([]byte(correlationJSON), &record.Correlation)
	}
	if record.State == AskDefaulted {
		record.Answer = record.DefaultAnswer
		record.AnsweredBy = DefaultAnsweredBy
		record.AnsweredAt = record.DefaultAppliedAt
	}
	if record.Answer != "" {
		record.AnswerLabel = optionLabel(record.Options, record.Answer)
	}
	return record, nil
}

// GetAsk reads one ask. It has no side effects: an ask past its deadline is
// reported as it is stored, and only the background sweep changes state.
func (s *Service) GetAsk(ctx context.Context, id string) (AskRecord, error) {
	return scanAsk(s.db.QueryRowContext(ctx, askSelect+` WHERE a.id = ?`, strings.TrimSpace(id)))
}

// ListAsks returns the recipient's asks newest first. openOnly limits the
// list to asks that still wait for an answer.
func (s *Service) ListAsks(ctx context.Context, recipient string, openOnly bool, limit int) ([]AskRecord, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	query := askSelect + ` WHERE a.notification_id IN (SELECT id FROM notifications WHERE requested_by = ?)`
	if openOnly {
		query += ` AND a.state IN ('pending', 'escalated')`
	}
	rows, err := s.db.QueryContext(ctx, query+` ORDER BY a.created_at DESC LIMIT ?`, recipient, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AskRecord
	for rows.Next() {
		record, scanErr := scanAsk(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, record)
	}
	return out, rows.Err()
}

// DefaultEligibleAt is when the sweep may apply the ask's default:
// max(deadline, first delivered receipt + floor). It is "" without a
// default, for a non-reversible ask, or before anything was delivered.
func (s *Service) DefaultEligibleAt(record AskRecord) string {
	if record.DefaultAnswer == "" || !record.Reversible || record.FirstDeliveredAt == "" {
		return ""
	}
	delivered, err := time.Parse(time.RFC3339Nano, record.FirstDeliveredAt)
	if err != nil {
		return ""
	}
	deadline, err := time.Parse(time.RFC3339Nano, record.Deadline)
	if err != nil {
		return ""
	}
	eligible := delivered.Add(s.defaultFloor())
	if deadline.After(eligible) {
		eligible = deadline
	}
	return eligible.UTC().Format(time.RFC3339)
}

func (s *Service) askByRequester(ctx context.Context, source, ref string) (AskRecord, error) {
	if strings.TrimSpace(ref) == "" {
		return AskRecord{}, ErrNotFound
	}
	return scanAsk(s.db.QueryRowContext(ctx, askSelect+` WHERE a.source = ? AND a.requester_ref = ? ORDER BY a.created_at ASC LIMIT 1`, source, ref))
}

func (s *Service) askByNotification(ctx context.Context, notificationID string) (AskRecord, error) {
	return scanAsk(s.db.QueryRowContext(ctx, askSelect+` WHERE a.notification_id = ?`, notificationID))
}

// AnswerAsk records an operator answer. An open ask becomes answered; a
// defaulted ask accepts one late answer that supersedes the default.
func (s *Service) AnswerAsk(ctx context.Context, in AnswerInput) (AskRecord, error) {
	record, err := s.GetAsk(ctx, in.AskID)
	if err != nil {
		return AskRecord{}, err
	}
	if in.Actor != record.Recipient {
		// Only the person the ask was sent to may answer it.
		return AskRecord{}, ErrNotFound
	}
	late := false
	switch record.State {
	case AskPending, AskEscalated:
	case AskDefaulted:
		late = true
	default:
		return AskRecord{}, fmt.Errorf("%w: ask is already %s", ErrAskResolved, record.State)
	}
	answer := strings.TrimSpace(in.Answer)
	found := false
	for _, option := range record.Options {
		if option.Key == answer {
			found = true
		}
	}
	if !found {
		return AskRecord{}, fmt.Errorf("%w: answer is not one of the allowed answers", ErrInvalidArgument)
	}
	now := s.clock.Now().UTC().Format(time.RFC3339Nano)
	reason := "answered by the operator"
	if late {
		reason = "operator answer after the default supersedes it"
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return AskRecord{}, err
	}
	defer func() { _ = tx.Rollback() }()
	result, err := tx.ExecContext(ctx, `UPDATE asks SET state='answered', reason=?, resolved_at=?, resolution_published_at='', updated_at=? WHERE id=? AND state=?`, reason, now, now, record.ID, record.State)
	if err != nil {
		return AskRecord{}, err
	}
	if changed, _ := result.RowsAffected(); changed != 1 {
		return AskRecord{}, fmt.Errorf("%w: ask changed while answering", ErrAskResolved)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO answers (id, ask_id, answer, answered_by, answered_at, note, late) VALUES (?, ?, ?, ?, ?, ?, ?)`, uuid.NewString(), record.ID, answer, in.Actor, now, in.Note, boolInt(late)); err != nil {
		return AskRecord{}, fmt.Errorf("%w: ask already has an answer", ErrAskResolved)
	}
	if err := tx.Commit(); err != nil {
		return AskRecord{}, err
	}
	s.kickResolutions(ctx)
	return s.GetAsk(ctx, record.ID)
}

// Wait blocks until the ask is resolved or the caller's deadline passes. It
// never changes the ask: when the caller's deadline passes first it returns
// the open state, and the ask keeps waiting for the operator.
func (s *Service) Wait(ctx context.Context, askID string, deadline time.Time) (string, string, string, error) {
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for {
		record, err := s.GetAsk(ctx, askID)
		if err != nil {
			return "", "", "", err
		}
		if !record.Open() {
			return record.State, record.Answer, record.Reason, nil
		}
		if !deadline.IsZero() && !s.clock.Now().Before(deadline) {
			return record.State, "", "caller deadline reached; the ask is still open", nil
		}
		select {
		case <-ctx.Done():
			return "", "", "", ctx.Err()
		case <-ticker.C:
		}
	}
}

// applyDefaultIfDue is the sweep rule for an overdue reversible ask with a
// default: apply it only after the deadline and at least the floor after the
// first delivered receipt. Without a delivered receipt the ask stays open and
// says the operator was not reached.
func (s *Service) applyDefaultIfDue(ctx context.Context, askID string) error {
	record, err := s.GetAsk(ctx, askID)
	if err != nil {
		return err
	}
	if !record.Open() || record.DefaultAnswer == "" || !record.Reversible {
		return nil
	}
	now := s.clock.Now().UTC()
	nowText := now.Format(time.RFC3339Nano)
	if record.FirstDeliveredAt == "" {
		return s.setAskReason(ctx, record, operatorNotReached, nowText)
	}
	delivered, err := time.Parse(time.RFC3339Nano, record.FirstDeliveredAt)
	if err != nil {
		return fmt.Errorf("parse first delivered receipt: %w", err)
	}
	deadline, err := time.Parse(time.RFC3339Nano, record.Deadline)
	if err != nil {
		return fmt.Errorf("parse ask deadline: %w", err)
	}
	eligible := delivered.Add(s.defaultFloor())
	if deadline.After(eligible) {
		eligible = deadline
	}
	if now.Before(eligible) {
		return s.setAskReason(ctx, record, "default "+record.DefaultAnswer+" applies at "+eligible.UTC().Format(time.RFC3339), nowText)
	}
	result, err := s.db.ExecContext(ctx, `UPDATE asks SET state='defaulted', reason=?, default_applied_at=?, resolved_at=?, resolution_published_at='', updated_at=? WHERE id=? AND state IN ('pending','escalated')`, "default applied after the deadline without an operator answer", nowText, nowText, nowText, record.ID)
	if err != nil {
		return err
	}
	if changed, _ := result.RowsAffected(); changed == 1 {
		s.kickResolutions(ctx)
	}
	return nil
}

func (s *Service) setAskReason(ctx context.Context, record AskRecord, reason, now string) error {
	if record.Reason == reason {
		return nil
	}
	_, err := s.db.ExecContext(ctx, `UPDATE asks SET reason=?, updated_at=? WHERE id=? AND state IN ('pending','escalated')`, reason, now, record.ID)
	return err
}

// askPath is the deep link a push opens for an ask notification, relative to
// the UI's service-worker scope. Non-ask notifications return "".
func (s *Service) askPath(ctx context.Context, notificationID string) string {
	var id string
	if err := s.db.QueryRowContext(ctx, `SELECT id FROM asks WHERE notification_id = ?`, notificationID).Scan(&id); err != nil {
		return ""
	}
	return "asks/" + id
}

func (s *Service) kickResolutions(ctx context.Context) {
	s.resolutions.mu.Lock()
	if s.resolutions.running || s.resolutions.publisher == nil {
		s.resolutions.mu.Unlock()
		return
	}
	s.resolutions.running = true
	s.resolutions.mu.Unlock()
	go func() {
		defer func() {
			s.resolutions.mu.Lock()
			s.resolutions.running = false
			s.resolutions.mu.Unlock()
		}()
		publishCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
		defer cancel()
		if err := s.PublishPendingResolutions(publishCtx); err != nil {
			s.log.Printf("publish ask resolutions: %v", err)
		}
	}()
}

// PublishPendingResolutions drains the ask-resolved outbox. Each event's id
// is derived from its content, so a retry after a lost acknowledgement is
// deduplicated by vrooli-events rather than delivered twice.
func (s *Service) PublishPendingResolutions(ctx context.Context) error {
	s.resolutions.mu.Lock()
	publisher := s.resolutions.publisher
	s.resolutions.mu.Unlock()
	if publisher == nil {
		return nil
	}
	s.resolutions.drain.Lock()
	defer s.resolutions.drain.Unlock()
	rows, err := s.db.QueryContext(ctx, `SELECT id FROM asks WHERE resolved_at != '' AND resolution_published_at = '' ORDER BY resolved_at ASC LIMIT 50`)
	if err != nil {
		return err
	}
	var ids []string
	for rows.Next() {
		var id string
		if scanErr := rows.Scan(&id); scanErr != nil {
			rows.Close()
			return scanErr
		}
		ids = append(ids, id)
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, id := range ids {
		record, err := s.GetAsk(ctx, id)
		if err != nil {
			return err
		}
		if err := publisher.PublishDomainEvent(ctx, AskResolvedEvent(record)); err != nil {
			return fmt.Errorf("publish %s for ask %s: %w", AskResolvedEventType, id, err)
		}
		if _, err := s.db.ExecContext(ctx, `UPDATE asks SET resolution_published_at=? WHERE id=? AND resolved_at=?`, s.clock.Now().UTC().Format(time.RFC3339Nano), id, record.ResolvedAt); err != nil {
			return err
		}
	}
	return nil
}

// AskResolvedEvent is the requester notification for one resolution. The
// payload is facts only; consumers match on source and requester_ref (the
// event id or idempotency key that opened the ask) or on correlation.
func AskResolvedEvent(record AskRecord) eventbus.DomainEvent {
	occurred, _ := time.Parse(time.RFC3339Nano, record.ResolvedAt)
	correlation := make(map[string]any, len(record.Correlation))
	for key, value := range record.Correlation {
		correlation[key] = value
	}
	return eventbus.DomainEvent{
		Source:    "notification-hub",
		EventType: AskResolvedEventType,
		Occurred:  occurred.UTC(),
		Payload: map[string]any{
			"ask_id":            record.ID,
			"state":             record.State,
			"answer":            record.Answer,
			"answer_label":      record.AnswerLabel,
			"note":              record.Note,
			"answered_by":       record.AnsweredBy,
			"answered_at":       record.AnsweredAt,
			"by_default":        record.State == AskDefaulted,
			"late":              record.Late,
			"default_answer":    record.DefaultAnswer,
			"reason":            record.Reason,
			"source":            record.Source,
			"source_event_type": record.SourceEventType,
			"requester_ref":     record.RequesterRef,
			"correlation":       correlation,
			"resolved_at":       record.ResolvedAt,
		},
	}
}
