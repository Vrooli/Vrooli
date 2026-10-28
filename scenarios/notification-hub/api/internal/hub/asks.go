package hub

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/eventbus"
)

// Ask states. pending and escalated are open; answered, defaulted, expired,
// and withdrawn are resolved. A defaulted ask still accepts one late operator
// answer, which supersedes the default.
const (
	AskPending   = "pending"
	AskEscalated = "escalated"
	AskAnswered  = "answered"
	AskDefaulted = "defaulted"
	AskExpired   = "expired"
	AskWithdrawn = "withdrawn"
)

const (
	// DefaultAskUrgency keeps asks inside quiet hours unless a requester
	// explicitly asks for critical.
	DefaultAskUrgency = "high"
	// DefaultAskDefaultFloor is the minimum time between the first delivered
	// receipt and applying a default answer.
	DefaultAskDefaultFloor = 12 * time.Hour
	// AskResolvedEventType tells the requester an ask was answered, defaulted,
	// expired, or withdrawn. It is published once per transition.
	AskResolvedEventType = "notification_hub.ask.resolved.v1"
	// DefaultAnsweredBy marks an answer the sweep applied.
	DefaultAnsweredBy = "default"

	maxAskOptions                 = 8
	maxAskOptionLabel             = 200
	maxAskCorrelationLen          = 1024
	maxBigPictureCodePoints       = 4_000
	maxTechnicalDetailsCodePoints = 12_000
	operatorNotReached            = "operator not reached: no delivered receipt yet; default withheld"
)

var (
	ErrAskResolved         = errors.New("ask is already resolved")
	ErrAskIdentityConflict = errors.New("ask identity is already bound to different immutable terms")
	ErrAskWithdrawn        = errors.New("decision request was withdrawn before ask creation")
	askOptionKeyRule       = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,63}$`)
	validUrgencies         = map[string]bool{"low": true, "normal": true, "high": true, "critical": true}
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
	BigPicture           string
	TechnicalDetails     string
	Deadline             time.Time
	DeadlineSpecified    bool
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
	Title                string
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
	BigPicture           string
	TechnicalDetails     string
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
//
// Actor is the subject the recipient-only authorization check compares against
// (it must equal the ask's recipient). AnsweredBy is the ACTUAL verified human
// recorded as answered_by on the resolution; when empty it falls back to Actor,
// so a caller that has no per-human identity (bearer/cookie path) keeps the
// prior behavior. Splitting them lets a multi-human household answer a shared
// recipient's ask while recording who really answered.
type AnswerInput struct {
	AskID, Answer, Actor, Note string
	AnsweredBy                 string
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
	spec.BigPicture = normalizeExplanation(spec.BigPicture)
	spec.TechnicalDetails = normalizeExplanation(spec.TechnicalDetails)
	if !utf8.ValidString(spec.BigPicture) || utf8.RuneCountInString(spec.BigPicture) > maxBigPictureCodePoints {
		return spec, fmt.Errorf("%w: big_picture must be valid UTF-8 and at most %d Unicode code points", ErrInvalidArgument, maxBigPictureCodePoints)
	}
	if !utf8.ValidString(spec.TechnicalDetails) || utf8.RuneCountInString(spec.TechnicalDetails) > maxTechnicalDetailsCodePoints {
		return spec, fmt.Errorf("%w: technical_details must be valid UTF-8 and at most %d Unicode code points", ErrInvalidArgument, maxTechnicalDetailsCodePoints)
	}
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

func normalizeExplanation(value string) string {
	return strings.TrimSpace(strings.ReplaceAll(value, "\r\n", "\n"))
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
		if existing.SourceEventType != spec.SourceEventType {
			return AskRecord{}, Notification{}, ErrAskIdentityConflict
		}
		n, _, getErr := s.Get(ctx, existing.NotificationID)
		if getErr != nil {
			return AskRecord{}, Notification{}, getErr
		}
		if !sameAskTerms(existing, n, spec) {
			return AskRecord{}, Notification{}, ErrAskIdentityConflict
		}
		if existing.State == AskWithdrawn {
			return AskRecord{}, Notification{}, ErrAskWithdrawn
		}
		return existing, n, getErr
	} else if !errors.Is(err, ErrNotFound) {
		return AskRecord{}, Notification{}, err
	}
	// Keep the notification and Ask in one transaction. The notification
	// recovery sweep processes every pending row, so a losing identity race
	// must roll its notification back before it can become deliverable.
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return AskRecord{}, Notification{}, err
	}
	defer tx.Rollback()
	// The A239 lifecycle ledger covers only Agent Manager run-approval asks.
	// Other producers and direct/manual asks keep their existing path. A
	// mismatched Agent Manager type still cannot reuse a ref already reserved
	// by the exact run-approval withdrawal contract.
	if spec.Source == "agent-manager" && spec.IdempotencyKey != "" {
		var withdrawn bool
		if spec.SourceEventType == agentManagerRunDecisionRequestEventType {
			withdrawn, err = reserveDecisionRequestTx(ctx, tx, spec.Source, spec.IdempotencyKey, spec.SourceEventType, s.clock.Now().UTC().Format(time.RFC3339Nano))
		} else {
			withdrawn, err = checkDecisionRequestLifecycleTx(ctx, tx, spec.Source, spec.IdempotencyKey, spec.SourceEventType)
		}
		if err != nil {
			return AskRecord{}, Notification{}, err
		}
		if withdrawn {
			return AskRecord{}, Notification{}, ErrAskWithdrawn
		}
	}
	n, created, err := s.persistNotificationUsing(ctx, tx, SendInput{RequestedBy: spec.Recipient, Title: spec.Title, Body: spec.Body, Urgency: spec.Urgency, SensitivityLabel: spec.SensitivityLabel, IdempotencyKey: spec.IdempotencyKey})
	if err != nil {
		return AskRecord{}, Notification{}, err
	}
	if !created {
		if existing, lookupErr := askByNotificationTx(ctx, tx, n.ID); lookupErr == nil {
			if !sameAskTerms(existing, n, spec) {
				return AskRecord{}, Notification{}, ErrAskIdentityConflict
			}
			if err := tx.Commit(); err != nil {
				return AskRecord{}, Notification{}, err
			}
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
	if _, err := tx.ExecContext(ctx, `INSERT INTO asks (id, notification_id, question, allowed_answers, deadline, state, reason, created_at, updated_at, options_json, recommended, recommendation_reason, default_answer, reversible, urgency, context_url, big_picture, technical_details, source, source_event_type, requester_ref, correlation_json) VALUES (?, ?, ?, ?, ?, 'pending', '', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, n.ID, spec.Question, string(allowedJSON), spec.Deadline.UTC().Format(time.RFC3339Nano), now, now, string(optionsJSON), spec.Recommended, strings.TrimSpace(spec.RecommendationReason), spec.DefaultAnswer, boolInt(spec.Reversible), spec.Urgency, strings.TrimSpace(spec.ContextURL), spec.BigPicture, spec.TechnicalDetails, spec.Source, strings.TrimSpace(spec.SourceEventType), spec.IdempotencyKey, correlationJSON); err != nil {
		// Either notification idempotency or the durable requester identity
		// constraint may have selected a concurrent winner. Re-read by the
		// immutable identity first so a different recipient cannot split one
		// source event into two answerable asks.
		if existing, lookupErr := askByRequesterTx(ctx, tx, spec.Source, spec.IdempotencyKey); lookupErr == nil {
			if existing.SourceEventType != spec.SourceEventType {
				return AskRecord{}, Notification{}, ErrAskIdentityConflict
			}
			existingNotification, getErr := notificationByIDTx(ctx, tx, existing.NotificationID)
			if getErr != nil {
				return AskRecord{}, Notification{}, getErr
			}
			if !sameAskTerms(existing, existingNotification, spec) {
				return AskRecord{}, Notification{}, ErrAskIdentityConflict
			}
			if existing.State == AskWithdrawn {
				return AskRecord{}, Notification{}, ErrAskWithdrawn
			}
			if err := tx.Commit(); err != nil {
				return AskRecord{}, Notification{}, err
			}
			return existing, existingNotification, nil
		}
		if existing, lookupErr := askByNotificationTx(ctx, tx, n.ID); lookupErr == nil {
			if !sameAskTerms(existing, n, spec) {
				return AskRecord{}, Notification{}, ErrAskIdentityConflict
			}
			if err := tx.Commit(); err != nil {
				return AskRecord{}, Notification{}, err
			}
			return existing, n, nil
		}
		return AskRecord{}, Notification{}, fmt.Errorf("persist ask: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return AskRecord{}, Notification{}, err
	}
	if created {
		s.observeCatalog(ctx, n, now)
	}
	// The ask row exists before routing starts, so the push can deep-link to
	// the ask page.
	// The successful Ask insert owns routing. Under concurrent same-terms
	// retries, the notification creator can lose the Ask insert race; dispatch
	// based on `created` would then leave the winning durable Ask unrouted.
	s.dispatch(n.ID)
	record, err := s.GetAsk(ctx, id)
	return record, n, err
}

func askByRequesterTx(ctx context.Context, tx dbTx, source, ref string) (AskRecord, error) {
	if ref == "" {
		return AskRecord{}, ErrNotFound
	}
	return scanAsk(tx.QueryRowContext(ctx, askSelect+` WHERE a.source = ? AND a.requester_ref = ? ORDER BY a.created_at ASC LIMIT 1`, source, ref))
}

func askByNotificationTx(ctx context.Context, tx dbTx, notificationID string) (AskRecord, error) {
	return scanAsk(tx.QueryRowContext(ctx, askSelect+` WHERE a.notification_id = ?`, notificationID))
}

func notificationByIDTx(ctx context.Context, tx dbTx, id string) (Notification, error) {
	var n Notification
	err := tx.QueryRowContext(ctx, `SELECT id, requested_by, title, body, urgency, sensitivity_label, idempotency_key, dedupe_key, scheduled_at, digest_key, digest_window_seconds, source_scenario, category, context_url, scope_key, created_at, updated_at FROM notifications WHERE id = ?`, id).Scan(&n.ID, &n.RequestedBy, &n.Title, &n.Body, &n.Urgency, &n.SensitivityLabel, &n.IdempotencyKey, &n.DedupeKey, &n.ScheduledAt, &n.DigestKey, &n.DigestWindowSeconds, &n.SourceScenario, &n.Category, &n.ContextURL, &n.ScopeKey, &n.CreatedAt, &n.UpdatedAt)
	return n, err
}

func sameAskTerms(existing AskRecord, notification Notification, spec AskSpec) bool {
	if existing.Source != spec.Source || existing.RequesterRef != spec.IdempotencyKey ||
		existing.Recipient != spec.Recipient || notification.Title != spec.Title ||
		notification.Urgency != spec.Urgency || notification.SensitivityLabel != spec.SensitivityLabel ||
		existing.Question != spec.Question ||
		existing.Recommended != spec.Recommended || existing.RecommendationReason != spec.RecommendationReason ||
		existing.DefaultAnswer != spec.DefaultAnswer || existing.Reversible != spec.Reversible ||
		existing.Urgency != spec.Urgency || existing.ContextURL != spec.ContextURL ||
		existing.BigPicture != spec.BigPicture || existing.TechnicalDetails != spec.TechnicalDetails ||
		existing.SourceEventType != spec.SourceEventType || len(existing.Options) != len(spec.Options) ||
		!sameCorrelation(existing.Correlation, spec.Correlation) {
		return false
	}
	for index := range existing.Options {
		if existing.Options[index] != spec.Options[index] {
			return false
		}
	}
	if spec.DeadlineSpecified && existing.Deadline != spec.Deadline.UTC().Format(time.RFC3339Nano) {
		return false
	}
	if spec.Body != renderAskBody(spec) {
		return notification.Body == spec.Body
	}
	// The generated push body includes its deadline when a default answer is
	// present. For implicit deadlines, compare against the original deadline so
	// a same-content retry remains stable while the notification body stays
	// identity-bound.
	originalDeadline, err := time.Parse(time.RFC3339Nano, existing.Deadline)
	if err != nil {
		return false
	}
	stable := spec
	stable.Deadline = originalDeadline
	return notification.Body == renderAskBody(stable)
}

func sameCorrelation(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for key, value := range a {
		if b[key] != value {
			return false
		}
	}
	return true
}

const askSelect = `SELECT a.id, a.notification_id, a.question, a.allowed_answers, a.options_json, a.recommended, a.recommendation_reason, a.default_answer, a.reversible, a.urgency, a.context_url, a.big_picture, a.technical_details, a.source, a.source_event_type, a.requester_ref, a.correlation_json, a.deadline, a.state, a.reason, a.default_applied_at, a.resolved_at, a.created_at, a.updated_at,
  COALESCE(ans.answer, ''), COALESCE(ans.answered_by, ''), COALESCE(ans.answered_at, ''), COALESCE(ans.note, ''), COALESCE(ans.late, 0),
  COALESCE((SELECT MIN(r.delivered_at) FROM receipts r WHERE r.notification_id = a.notification_id), ''),
  COALESCE((SELECT n.requested_by FROM notifications n WHERE n.id = a.notification_id), ''),
  COALESCE((SELECT n.title FROM notifications n WHERE n.id = a.notification_id), '')
FROM asks a LEFT JOIN answers ans ON ans.ask_id = a.id`

func scanAsk(row rowScanner) (AskRecord, error) {
	var record AskRecord
	var allowedJSON, optionsJSON, correlationJSON string
	var reversible, late int
	err := row.Scan(&record.ID, &record.NotificationID, &record.Question, &allowedJSON, &optionsJSON, &record.Recommended, &record.RecommendationReason, &record.DefaultAnswer, &reversible, &record.Urgency, &record.ContextURL, &record.BigPicture, &record.TechnicalDetails, &record.Source, &record.SourceEventType, &record.RequesterRef, &correlationJSON, &record.Deadline, &record.State, &record.Reason, &record.DefaultAppliedAt, &record.ResolvedAt, &record.CreatedAt, &record.UpdatedAt,
		&record.Answer, &record.AnsweredBy, &record.AnsweredAt, &record.Note, &late, &record.FirstDeliveredAt, &record.Recipient, &record.Title)
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

// GetAskForRecipient performs the recipient check in the read query itself so
// callers cannot use ask IDs to inspect another configured recipient's data.
func (s *Service) GetAskForRecipient(ctx context.Context, id, recipient string) (AskRecord, error) {
	recipient = strings.TrimSpace(recipient)
	if recipient == "" {
		return AskRecord{}, ErrNotFound
	}
	return scanAsk(s.db.QueryRowContext(ctx, askSelect+` WHERE a.id = ? AND a.notification_id IN (SELECT id FROM notifications WHERE requested_by = ?)`, strings.TrimSpace(id), recipient))
}

// ListAsks is the first-page convenience used by existing internal callers.
// Browser and API callers use ListAsksPage to continue through every result.
func (s *Service) ListAsks(ctx context.Context, recipient string, openOnly bool, limit int) ([]AskRecord, error) {
	page, err := s.ListAsksPage(ctx, recipient, AskListOptions{OpenOnly: openOnly, Limit: limit})
	return page.Asks, err
}

type AskListOptions struct {
	OpenOnly    bool
	Limit       int
	Cursor      string
	Query       string
	StateFilter string
}

type AskListPage struct {
	Asks       []AskRecord
	NextCursor string
}

type askListCursor struct {
	Version   int    `json:"v"`
	Scope     string `json:"scope"`
	CreatedAt string `json:"created_at"`
	ID        string `json:"id"`
}

// ListAsksPage searches the recipient's full retained inventory before applying
// a stable keyset boundary. Reads never resolve, answer or otherwise change asks.
// Pages are live: newly created rows appear on refresh, not midway through older pages.
func (s *Service) ListAsksPage(ctx context.Context, recipient string, opts AskListOptions) (AskListPage, error) {
	if strings.TrimSpace(recipient) == "" {
		return AskListPage{}, ErrNotFound
	}
	if opts.Limit <= 0 {
		opts.Limit = 50
	}
	if opts.Limit > 200 {
		opts.Limit = 200
	}
	opts.Query = strings.TrimSpace(opts.Query)
	if !utf8.ValidString(opts.Query) || utf8.RuneCountInString(opts.Query) > 500 {
		return AskListPage{}, fmt.Errorf("%w: search must be valid UTF-8 and at most 500 characters", ErrInvalidArgument)
	}
	state := strings.TrimSpace(opts.StateFilter)
	if state == "" {
		state = "all"
	}
	if opts.OpenOnly {
		if state != "all" && state != "awaiting" && state != AskPending && state != AskEscalated {
			return AskListPage{}, fmt.Errorf("%w: open_only conflicts with state_filter", ErrInvalidArgument)
		}
		if state == "all" {
			state = "awaiting"
		}
	}
	switch state {
	case "all", "awaiting", AskPending, AskEscalated, AskAnswered, AskDefaulted, AskExpired, AskWithdrawn:
	default:
		return AskListPage{}, fmt.Errorf("%w: unknown state_filter", ErrInvalidArgument)
	}
	scopeBytes, _ := json.Marshal([]string{recipient, opts.Query, state})
	scope := fmt.Sprintf("%x", sha256.Sum256(scopeBytes))
	query := askSelect + ` WHERE a.notification_id IN (SELECT id FROM notifications WHERE requested_by = ?)`
	args := []any{recipient}
	if state == "awaiting" {
		query += ` AND a.state IN ('pending', 'escalated')`
	} else if state != "all" {
		query += ` AND a.state = ?`
		args = append(args, state)
	}
	if opts.Query != "" {
		// User wildcards are literal. Include option labels so readable answers,
		// not only their machine keys, are searchable without loading all records.
		pattern := "%" + strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`).Replace(strings.ToLower(opts.Query)) + "%"
		query += ` AND (LOWER(a.question) LIKE ? ESCAPE '\' OR LOWER(a.source) LIKE ? ESCAPE '\' OR LOWER(a.options_json) LIKE ? ESCAPE '\' OR LOWER(a.default_answer) LIKE ? ESCAPE '\' OR LOWER(a.requester_ref) LIKE ? ESCAPE '\' OR LOWER(COALESCE(ans.answer, '')) LIKE ? ESCAPE '\' OR LOWER(COALESCE(ans.note, '')) LIKE ? ESCAPE '\' OR LOWER((SELECT n.title FROM notifications n WHERE n.id = a.notification_id)) LIKE ? ESCAPE '\')`
		for range 8 {
			args = append(args, pattern)
		}
	}
	if opts.Cursor != "" {
		if len(opts.Cursor) > 2048 {
			return AskListPage{}, fmt.Errorf("%w: cursor is too long", ErrInvalidArgument)
		}
		var cursor askListCursor
		decoded, err := base64.RawURLEncoding.DecodeString(opts.Cursor)
		if err != nil || json.Unmarshal(decoded, &cursor) != nil || cursor.Version != 1 || cursor.Scope != scope || cursor.CreatedAt == "" || cursor.ID == "" {
			return AskListPage{}, fmt.Errorf("%w: cursor does not match this search", ErrInvalidArgument)
		}
		if _, err := time.Parse(time.RFC3339Nano, cursor.CreatedAt); err != nil {
			return AskListPage{}, fmt.Errorf("%w: invalid cursor time", ErrInvalidArgument)
		}
		query += ` AND (a.created_at < ? OR (a.created_at = ? AND a.id < ?))`
		args = append(args, cursor.CreatedAt, cursor.CreatedAt, cursor.ID)
	}
	args = append(args, opts.Limit+1)
	rows, err := s.db.QueryContext(ctx, query+` ORDER BY a.created_at DESC, a.id DESC LIMIT ?`, args...)
	if err != nil {
		return AskListPage{}, err
	}
	defer rows.Close()
	out := make([]AskRecord, 0, opts.Limit+1)
	for rows.Next() {
		record, scanErr := scanAsk(rows)
		if scanErr != nil {
			return AskListPage{}, scanErr
		}
		out = append(out, record)
	}
	if err := rows.Err(); err != nil {
		return AskListPage{}, err
	}
	page := AskListPage{Asks: out}
	if len(out) > opts.Limit {
		page.Asks = out[:opts.Limit]
		last := page.Asks[len(page.Asks)-1]
		encoded, _ := json.Marshal(askListCursor{Version: 1, Scope: scope, CreatedAt: last.CreatedAt, ID: last.ID})
		page.NextCursor = base64.RawURLEncoding.EncodeToString(encoded)
	}
	return page, nil
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
	// answered_by records the ACTUAL verified human; it falls back to the
	// recipient-check actor when the request carried no per-human identity.
	answeredBy := strings.TrimSpace(in.AnsweredBy)
	if answeredBy == "" {
		answeredBy = in.Actor
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO answers (id, ask_id, answer, answered_by, answered_at, note, late) VALUES (?, ?, ?, ?, ?, ?, ?)`, uuid.NewString(), record.ID, answer, answeredBy, now, in.Note, boolInt(late)); err != nil {
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
