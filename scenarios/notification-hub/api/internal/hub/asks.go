package hub

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/operatordecision"
	"notification-hub/internal/dbvalue"
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
	AskDismissed = "dismissed"
)

const (
	// DefaultAskUrgency is the urgency of an ask whose requester names none. It
	// matches the CLI, so the same decision has the same urgency whichever route
	// raised it. A producer whose work is blocked on the answer says "high".
	DefaultAskUrgency = "normal"
	// DefaultAskDeadline is how long an ask that names no deadline waits for its
	// answer.
	DefaultAskDeadline = 24 * time.Hour
	// DefaultAskSensitivity labels an ask whose requester names none: the
	// operator's own phone is the only audience, so the label that fits an
	// ordinary decision.
	DefaultAskSensitivity = "private"
	// DefaultAskDefaultFloor is the minimum time between the first delivered
	// receipt and applying a default answer.
	DefaultAskDefaultFloor = 12 * time.Hour
	// AskResolvedEventType tells the requester an ask was answered, defaulted,
	// expired, withdrawn, or dismissed. It is published once per transition.
	AskResolvedEventType = operatordecision.AskResolvedEventType
	// DefaultAnsweredBy marks an answer the sweep applied.
	DefaultAnsweredBy = "default"

	maxAskOptions                 = 8
	maxAskOptionLabel             = 200
	maxAskCorrelationLen          = 1024
	maxBigPictureCodePoints       = 4_000
	maxTechnicalDetailsCodePoints = 12_000
	operatorNotReached            = "operator not reached: no delivered receipt yet; default withheld"
	// neverDeliveredExpiredReason closes an ask whose default was withheld for
	// lack of a delivered receipt for a week after its deadline.
	neverDeliveredExpiredReason = "expired: never delivered to the operator, 7 days after its deadline; default not applied"
)

var (
	ErrAskResolved = errors.New("ask is already resolved")
	// ErrAskChanged reports an answer checked against a revision of the ask that
	// the requester has since replaced. Nothing was stored; the answerer should
	// reload the ask and review the answers against its new text and form.
	ErrAskChanged          = errors.New("ask_changed: the requester updated this ask while you were answering; reload it and check your answers")
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
	// ProviderDecisionContext is an additive typed binding for provider rule
	// decisions. Its canonical JSON is persisted inside the ask's correlation
	// record and is compared during idempotent retries.
	ProviderDecisionContext *ProviderDecisionContext
	// SubjectKey names the ongoing situation the ask is about. With a key the
	// hub keeps at most one open ask per (Source, SourceEventType, SubjectKey):
	// a repeat re-asserts it, a text change updates it in place, a changed
	// answer menu replaces it. See ask_subjects.go.
	SubjectKey string
	// ReassertWithin closes the open ask when the producer has not re-sent it
	// for this long. It needs SubjectKey.
	ReassertWithin time.Duration
	// AssertedAt is when the producer last saw the decision was still needed.
	// It orders a request against the subject's withdrawals; zero means now.
	AssertedAt time.Time
	// AcceptsContextRequests is the asker's declaration that it will answer a
	// context request by revising this ask under the same SubjectKey. The Hub
	// offers the operator the button only for an ask that declared it, and drops
	// the declaration (with a warning) from an ask without a SubjectKey.
	AcceptsContextRequests bool
	// Keep lists the fields the request did not say. A revision keeps the open
	// ask's value for each, so it says only what changes; a request with no open
	// ask takes each field's default (and needs a question and answers). The zero
	// value says every field, which is what an event producer sends. See
	// keepOpenTerms.
	Keep AskKeep
	// ClearAttachments and ClearQuestions make a revision drop every current
	// attachment (the explainer included) or form question before applying the
	// ones it sends. Without them a revision keeps what it does not mention.
	ClearAttachments, ClearQuestions bool
	// RemoveAttachments names current attachments a revision drops, leaving the
	// others. It applies after ClearAttachments and before the attachments sent.
	RemoveAttachments []string
	// RemoveQuestions names the ids of current form questions a revision drops,
	// leaving the others. It applies after ClearQuestions and before the
	// questions sent.
	RemoveQuestions []string
	// ContextResponse is the requester's written answer to the operator's open
	// context questions. It is not a term of the ask: the revision stores it with
	// each question it answers. sealAsk drops it, with a warning, when no question
	// is open.
	ContextResponse string
	// Questions is the answer form shown under the main options. normalizeQuestions
	// validates it; a question that breaks a rule is dropped with a warning.
	Questions []AskQuestion
	// Attachments are the files and folders the operator opens inside the Hub.
	// prepareAttachments validates them against the blob inventory; one the Hub
	// cannot accept is dropped with a warning.
	Attachments []AskAttachment
	// VerifiedProducer is set by the handler that verified who is asking (an
	// agent run or the owner). Attachments carry host paths the Hub later probes
	// and bytes it stores, so an ask without a verified producer has them all
	// dropped with a warning.
	VerifiedProducer bool
	// CreatedBy is the verified identity that sends the ask (run:<id> or
	// owner:<subject>); empty when the sender has none. See AskAudience for who
	// may read the ask back.
	CreatedBy string
	// CreatedConversation is the verified conversation the sending run belongs
	// to (shared by its recovered and child runs); empty when the sender is not
	// a run or its token carries no conversation claim.
	CreatedConversation string
	// CreatedSubject is the verified owner account behind the sender; empty when
	// it has none. It is what makes a subject key a capability for a run of the
	// same owner.
	CreatedSubject string
	// Sender is who the requester says is asking ("Research Desk"). Unverified
	// display text; empty on a revision keeps the ask's sender, and on a new ask
	// the Hub names it (DeriveSender).
	Sender string
	// DefaultSender is a name the caller's tooling suggests (the label of the
	// session it ran in). It names a new ask that has no Sender and nothing else:
	// a revision never renames from it, and a replacement keeps the name of the
	// ask it replaces. Unlike Sender it is repaired, not refused.
	DefaultSender string
	// sourceName is the name the Hub has registered for Source, read by
	// prepareAsk before any transaction opens. The store has one connection, so
	// a lookup made while a write transaction is open would wait on itself.
	sourceName string
}

// AskKeep names the fields of an AskSpec its request left unsaid. Each is kept
// from the open ask the request revises.
type AskKeep struct {
	Question, Options                      bool
	Recommended, RecommendationReason      bool
	DefaultAnswer, Reversible              bool
	Urgency, ContextURL                    bool
	BigPicture, TechnicalDetails           bool
	ReassertWithin, AcceptsContextRequests bool
}

// Any reports whether the request left any field unsaid.
func (k AskKeep) Any() bool { return k != AskKeep{} }

// AskRecord is the read model of one ask. Reading it never changes it.
type AskRecord struct {
	ID                       string
	Title                    string
	NotificationID           string
	Recipient                string
	Question                 string
	Options                  []AskOption
	Recommended              string
	RecommendationReason     string
	DefaultAnswer            string
	Reversible               bool
	Urgency                  string
	ContextURL               string
	BigPicture               string
	TechnicalDetails         string
	Deadline                 string
	State                    string
	Reason                   string
	Answer                   string
	AnswerLabel              string
	AnsweredByIdentityKind   string
	AnsweredByIdentitySource string
	Note                     string
	AnsweredBy               string
	AnsweredAt               string
	Late                     bool
	DefaultAppliedAt         string
	FirstDeliveredAt         string
	Source                   string
	SourceEventType          string
	RequesterRef             string
	Correlation              map[string]string
	ProviderDecisionContext  *ProviderDecisionContext
	ResolvedAt               string
	CreatedAt                string
	UpdatedAt                string
	// SubjectKey is the producer's subject key; empty for a one-off decision.
	SubjectKey string
	// LastAssertedAt is when the producer last confirmed the decision.
	LastAssertedAt string
	// ReassertWithinSeconds is the producer's re-send cadence bound; 0 means none.
	ReassertWithinSeconds int
	// RevisionCount is the number of in-place updates the ask received.
	RevisionCount int
	// SupersededByAskID names the decision that replaced this one.
	SupersededByAskID string
	// CurrentEventID is the event id of the revision the ask shows now.
	CurrentEventID string
	// CreatedBy is the verified identity that sent the ask; empty when it had
	// none, and for event-sourced asks.
	CreatedBy string
	// CreatedConversation and CreatedSubject are the sender's verified
	// conversation and owner account; see AskSpec.
	CreatedConversation string
	CreatedSubject      string
	// Sender is who the operator reads as asking; see DeriveSender. Never empty
	// on a record read from the store.
	Sender string
	// Questions is the answer form of the ask's current revision.
	Questions []AskQuestion
	// AcceptsContextRequests records that the asker declared it answers context
	// requests; see CanRequestContext.
	AcceptsContextRequests bool
	// Responses are the operator's answers to Questions, in question order.
	Responses []AskQuestionResponse
	// AttachmentNotes are the notes the operator pinned to attachments in the
	// answer.
	AttachmentNotes []AskAttachmentNote
	// AnswerSummary is the plain-text rendering of the whole answer; empty until
	// an operator answered.
	AnswerSummary string
	// OpenContextRequestCount counts the recipient's context requests the
	// requester has not addressed by revising the ask.
	OpenContextRequestCount int
	// AttachmentCount is how many attachments the current revision has; the
	// attachments themselves come from Service.AskAttachments.
	AttachmentCount int
	// AttachmentsDigest identifies what the producer attached, for comparing a
	// resend with the open ask.
	AttachmentsDigest string
}

// Open reports whether the ask still waits for an answer.
func (r AskRecord) Open() bool { return r.State == AskPending || r.State == AskEscalated }

// CanRequestContext reports whether the operator may ask this ask's requester a
// question: the ask is open and its requester declared it answers by revising the
// ask. Without that, the question would go to a sender that cannot reply.
func (r AskRecord) CanRequestContext() bool { return r.Open() && r.AcceptsContextRequests }

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
	// Responses answer the ask's Questions. An unknown question id or a response
	// of the wrong shape is refused; required questions bind only when Answer is
	// one of the requester's options.
	Responses []AskQuestionResponse
	// AttachmentNotes are notes pinned to the ask's attachments; one that names
	// an attachment (or folder entry) the ask does not have is refused.
	AttachmentNotes          []AskAttachmentNote
	AnsweredBy               string
	AnsweredByIdentityKind   string
	AnsweredByIdentitySource string
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
	spec.ContextResponse = normalizeExplanation(spec.ContextResponse)
	sender, err := normalizeSender(spec.Sender)
	if err != nil {
		return spec, err
	}
	spec.Sender = sender
	spec.DefaultSender = repairSender(spec.DefaultSender)
	decisionContext, err := normalizeProviderDecisionContext(spec.ProviderDecisionContext)
	if err != nil {
		return spec, err
	}
	spec.ProviderDecisionContext = decisionContext
	if err := normalizeSubject(&spec); err != nil {
		return spec, err
	}
	if !utf8.ValidString(spec.BigPicture) || utf8.RuneCountInString(spec.BigPicture) > maxBigPictureCodePoints {
		return spec, fmt.Errorf("%w: big_picture must be valid UTF-8 and at most %d Unicode code points", ErrInvalidArgument, maxBigPictureCodePoints)
	}
	if !utf8.ValidString(spec.TechnicalDetails) || utf8.RuneCountInString(spec.TechnicalDetails) > maxTechnicalDetailsCodePoints {
		return spec, fmt.Errorf("%w: technical_details must be valid UTF-8 and at most %d Unicode code points", ErrInvalidArgument, maxTechnicalDetailsCodePoints)
	}
	if !utf8.ValidString(spec.ContextResponse) || utf8.RuneCountInString(spec.ContextResponse) > maxContextResponseCodePoints {
		return spec, fmt.Errorf("%w: context_response must be valid UTF-8 and at most %d Unicode code points", ErrInvalidArgument, maxContextResponseCodePoints)
	}
	if spec.Source == "" {
		spec.Source = spec.Recipient
	}
	if spec.Keep.Any() {
		// The request leaves fields to the open ask it revises; the merge reads
		// that ask (found by Source and SubjectKey) and then normalizes the whole.
		return spec, nil
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
		if key == "other" {
			return spec, fmt.Errorf("%w: answer key %q is reserved for the recipient's free-form answer", ErrInvalidArgument, key)
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
	return s.askWithSpec(ctx, spec, "")
}

// AskFrom accepts a direct decision ask without requiring producer authority.
// The source key selects the producer's device-push bucket; it never affects
// whether the ask is stored.
func (s *Service) AskFrom(ctx context.Context, source string, spec AskSpec) (AskRecord, Notification, error) {
	return s.askWithSpec(ctx, spec, source)
}

func (s *Service) askWithSpec(ctx context.Context, spec AskSpec, pushSource string) (AskRecord, Notification, error) {
	result, err := s.AskDecision(ctx, spec, pushSource)
	if err == nil && result.Outcome == AskStale {
		err = ErrAskStale
	}
	return result.Ask, result.Notification, err
}

// AskDecision opens, re-asserts, updates or replaces an ask and reports which
// happened. An ask with a SubjectKey follows the decision-subject rules; any
// other ask is idempotent on (Source, IdempotencyKey) and its Outcome is empty.
// pushSource is "" for event-sourced asks and the producer key for direct asks.
func (s *Service) AskDecision(ctx context.Context, spec AskSpec, pushSource string) (AskResult, error) {
	spec, warnings, err := s.prepareAsk(ctx, spec, false)
	if err != nil {
		return AskResult{}, err
	}
	// What the request says is read before it is sealed (sealing clears Keep), and
	// the open ask is read before the revision changes it: together they tell which
	// findings of the packet check are news.
	revision := s.beginRevision(ctx, spec, s.ReadAttachmentBlob)
	result, err := s.openAsk(ctx, spec, pushSource)
	if err == nil && result.packet != nil {
		result.Warnings = append(result.Warnings, revision.markCarriedOver(s.checkPacket(ctx, *result.packet, s.ReadAttachmentBlob))...)
	}
	result.Warnings = append(warnings, result.Warnings...)
	return result, err
}

// prepareAsk is everything an ask goes through before it is stored and needs no
// database read of the ask itself: validation and the request's own attachments.
// Laying them over the open ask's packet (sealAsk) needs that ask, so it
// happens where the ask is written. A packet part never fails an ask: what the
// Hub cannot accept is dropped and reported in the warnings.
//
// check is a dry run: it stores and touches nothing, and takes the request's
// files as stored with the sha256 and size the request states.
func (s *Service) prepareAsk(ctx context.Context, spec AskSpec, check bool) (AskSpec, []PacketWarning, error) {
	spec, err := normalizeAskSpec(spec)
	if err != nil {
		return spec, nil, err
	}
	spec.sourceName = s.resolveDisplayName(ctx, spec.Source)
	return s.prepareAttachments(ctx, spec, check)
}

// AskCheck is what a request would do, found without storing anything.
type AskCheck struct {
	// Warnings are the packet check's findings, as Ask would report them.
	Warnings []PacketWarning
	// PushTitle and PushBody are the notification the operator would receive.
	PushTitle, PushBody string
	// RevisesAskID is the open ask a keyed request would revise; empty when it
	// would create one.
	RevisesAskID string
	// Sender is the name the operator would see on the ask, and SenderSource
	// says where it came from. They come from the rule a real send uses.
	Sender       string
	SenderSource SenderSource
}

// CheckInput is what a dry run is given besides the request: the content of the
// files the check reads, by SHA-256. A dry run uploads nothing, so the Hub reads
// these in place of a stored copy, keeps none of them, and takes every
// attachment of the request as stored.
type CheckInput struct {
	Contents map[string][]byte
}

// reader reads a blob from the input's contents, and from the Hub's store when
// the input does not hold it.
func (in CheckInput) reader(stored BlobReader) BlobReader {
	return func(ctx context.Context, sha string, max int64) ([]byte, error) {
		if data, ok := in.Contents[sha]; ok {
			if int64(len(data)) > max {
				return nil, ErrBlobTooLarge
			}
			return data, nil
		}
		return stored(ctx, sha, max)
	}
}

// CheckAsk runs an ask through validation and the packet check and reports what
// the operator would see, changing nothing: no ask, notification, attachment or
// blob row is written. A revision is checked as it would be stored, merged over
// the open ask of its subject, and its findings are marked the way Ask marks them.
func (s *Service) CheckAsk(ctx context.Context, spec AskSpec, in CheckInput) (AskCheck, error) {
	read := in.reader(s.ReadAttachmentBlob)
	spec, warnings, err := s.prepareAsk(ctx, spec, true)
	if err != nil {
		return AskCheck{}, err
	}
	open, found, err := s.openSubjectAsk(ctx, spec)
	if err != nil {
		return AskCheck{}, err
	}
	var openAsk *AskRecord
	var current []AskAttachment
	if found {
		openAsk = &open
		if current, err = s.loadAttachments(ctx, open.ID); err != nil {
			return AskCheck{}, err
		}
	}
	revision := s.beginRevision(ctx, spec, read)
	spec, sealed, err := sealAsk(ctx, spec, openAsk, current, read)
	if err != nil {
		return AskCheck{}, err
	}
	findings := revision.markCarriedOver(s.checkPacket(ctx, spec, read))
	check := AskCheck{Warnings: append(append(warnings, sealed...), findings...), PushTitle: spec.Title, PushBody: spec.Body}
	kept := ""
	if found {
		check.RevisesAskID = open.ID
		kept = open.Sender
	}
	check.Sender, check.SenderSource = senderOf(spec, kept)
	return check, nil
}

func (s *Service) openAsk(ctx context.Context, spec AskSpec, pushSource string) (AskResult, error) {
	if spec.SubjectKey != "" {
		return s.askSubject(ctx, spec, pushSource)
	}
	// Nothing to revise: the packet is what the request sends.
	spec, warnings, err := sealAsk(ctx, spec, nil, nil, s.ReadAttachmentBlob)
	if err != nil {
		return AskResult{}, err
	}
	record, n, err := s.askPlain(ctx, spec, pushSource)
	return AskResult{Ask: record, Notification: n, Warnings: warnings, packet: &spec}, err
}

func (s *Service) askPlain(ctx context.Context, spec AskSpec, pushSource string) (AskRecord, Notification, error) {
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
	if err := s.requestLifecycleGateTx(ctx, tx, spec); err != nil {
		return AskRecord{}, Notification{}, err
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
	enc, err := encodeAskRow(spec)
	if err != nil {
		return AskRecord{}, Notification{}, err
	}
	id := uuid.NewString()
	now := storedTime(s.clock.Now())
	spec.Sender, _ = senderOf(spec, "")
	if err := insertAskTx(ctx, tx, id, n.ID, spec, enc, now, ""); err != nil {
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
	if err := insertAttachmentsTx(ctx, tx, id, spec.Attachments); err != nil {
		return AskRecord{}, Notification{}, err
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
	if pushSource == "" {
		s.dispatch(ctx, n.ID)
	} else {
		s.dispatchDirect(ctx, n.ID, pushSource)
	}
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
	if existing.Source != spec.Source || existing.RequesterRef != spec.IdempotencyKey || existing.SubjectKey != spec.SubjectKey ||
		existing.Recipient != spec.Recipient || notification.Title != spec.Title ||
		notification.Urgency != spec.Urgency || notification.SensitivityLabel != spec.SensitivityLabel ||
		existing.Question != spec.Question ||
		existing.Recommended != spec.Recommended || existing.RecommendationReason != spec.RecommendationReason ||
		existing.DefaultAnswer != spec.DefaultAnswer || existing.Reversible != spec.Reversible ||
		existing.Urgency != spec.Urgency || existing.ContextURL != spec.ContextURL ||
		existing.BigPicture != spec.BigPicture || existing.TechnicalDetails != spec.TechnicalDetails ||
		existing.SourceEventType != spec.SourceEventType || len(existing.Options) != len(spec.Options) ||
		!sameProviderDecisionContext(existing.ProviderDecisionContext, spec.ProviderDecisionContext) ||
		!sameQuestions(existing.Questions, spec.Questions) ||
		!sameCorrelation(existing.Correlation, spec.Correlation) {
		return false
	}
	for index := range existing.Options {
		if existing.Options[index] != spec.Options[index] {
			return false
		}
	}
	if spec.DeadlineSpecified && existing.Deadline != storedTime(spec.Deadline) {
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
  a.subject_key, a.last_asserted_at, a.reassert_within_seconds, a.revision_count, a.superseded_by_ask_id, a.current_event_id,
  a.questions_json, a.accepts_context_requests, COALESCE(ans.responses_json, ''), COALESCE(ans.attachment_notes_json, ''),
  (SELECT COUNT(*) FROM ask_context_requests c WHERE c.ask_id = a.id AND c.addressed_at = ''),
  a.attachment_count, a.attachments_digest, a.created_by, a.created_conversation, a.created_subject, a.sender,
  COALESCE(ans.answer, ''), COALESCE(ans.answered_by, ''), COALESCE(ans.answered_at, ''), COALESCE(ans.note, ''), COALESCE(ans.late, 0),
  COALESCE(ai.identity_kind, ''), COALESCE(ai.identity_source, ''),
  COALESCE((SELECT MIN(r.delivered_at) FROM receipts r WHERE r.notification_id = a.notification_id), ''),
  COALESCE((SELECT n.requested_by FROM notifications n WHERE n.id = a.notification_id), ''),
  COALESCE((SELECT n.title FROM notifications n WHERE n.id = a.notification_id), '')
FROM asks a LEFT JOIN answers ans ON ans.ask_id = a.id LEFT JOIN answer_identity ai ON ai.ask_id = a.id`

func scanAsk(row rowScanner) (AskRecord, error) {
	var record AskRecord
	var allowedJSON, optionsJSON, correlationJSON, questionsJSON, responsesJSON, notesJSON string
	var reversible, late, acceptsContext int
	err := row.Scan(&record.ID, &record.NotificationID, &record.Question, &allowedJSON, &optionsJSON, &record.Recommended, &record.RecommendationReason, &record.DefaultAnswer, &reversible, &record.Urgency, &record.ContextURL, &record.BigPicture, &record.TechnicalDetails, &record.Source, &record.SourceEventType, &record.RequesterRef, &correlationJSON, &record.Deadline, &record.State, &record.Reason, &record.DefaultAppliedAt, &record.ResolvedAt, &record.CreatedAt, &record.UpdatedAt,
		&record.SubjectKey, &record.LastAssertedAt, &record.ReassertWithinSeconds, &record.RevisionCount, &record.SupersededByAskID, &record.CurrentEventID,
		&questionsJSON, &acceptsContext, &responsesJSON, &notesJSON, &record.OpenContextRequestCount,
		&record.AttachmentCount, &record.AttachmentsDigest, &record.CreatedBy, &record.CreatedConversation, &record.CreatedSubject, &record.Sender,
		&record.Answer, &record.AnsweredBy, &record.AnsweredAt, &record.Note, &late, &record.AnsweredByIdentityKind, &record.AnsweredByIdentitySource, &record.FirstDeliveredAt, &record.Recipient, &record.Title)
	if errors.Is(err, sql.ErrNoRows) {
		return AskRecord{}, ErrNotFound
	}
	if err != nil {
		return AskRecord{}, err
	}
	record.Reversible = reversible != 0
	record.Late = late != 0
	if record.Sender == "" {
		record.Sender = DeriveSender("", record.Source, record.SourceEventType, record.CreatedBy, record.Recipient, nil)
	}
	record.AcceptsContextRequests = acceptsContext != 0
	record.Questions = decodeQuestions(questionsJSON)
	record.Responses = decodeResponses(responsesJSON)
	record.AttachmentNotes = decodeAttachmentNotes(notesJSON)
	if record.AnsweredByIdentityKind == "" {
		record.AnsweredByIdentityKind = AnswerIdentityKindUnknown
	}
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
	if encoded := record.Correlation[providerDecisionCorrelationKey]; encoded != "" {
		record.ProviderDecisionContext = decodeProviderDecisionContext(encoded)
		delete(record.Correlation, providerDecisionCorrelationKey)
	}
	if record.State == AskDefaulted {
		record.Answer = record.DefaultAnswer
		record.AnsweredBy = DefaultAnsweredBy
		record.AnsweredAt = record.DefaultAppliedAt
		record.AnsweredByIdentityKind = "system_default"
		record.AnsweredByIdentitySource = ""
	}
	if record.Answer != "" {
		record.AnswerLabel = optionLabel(record.Options, record.Answer)
		if record.Answer == "other" && allowsOtherAnswer(record) {
			record.AnswerLabel = "Other"
		}
	}
	record.AnswerSummary = AnswerSummary(record)
	return record, nil
}

// GetAsk reads one ask. It has no side effects: an ask past its deadline is
// reported as it is stored, and only the background sweep changes state.
func (s *Service) GetAsk(ctx context.Context, id string) (AskRecord, error) {
	return scanAsk(s.db.QueryRowContext(ctx, askSelect+` WHERE a.id = ?`, strings.TrimSpace(id)))
}

// GetAskByRequester reads one ask by the immutable requester identity. The
// source, event type, and requester reference must all match; this is not a
// list operation and does not change expired or pending asks.
func (s *Service) GetAskByRequester(ctx context.Context, source, eventType, requesterRef string) (AskRecord, error) {
	source = strings.TrimSpace(source)
	eventType = strings.TrimSpace(eventType)
	requesterRef = strings.TrimSpace(requesterRef)
	if source == "" || eventType == "" || requesterRef == "" {
		return AskRecord{}, ErrNotFound
	}
	record, err := s.askByRequester(ctx, source, requesterRef)
	if err != nil {
		return AskRecord{}, err
	}
	if record.SourceEventType != eventType {
		return AskRecord{}, ErrNotFound
	}
	return record, nil
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
	if answer == "other" && allowsOtherAnswer(record) {
		found = true
		if strings.TrimSpace(in.Note) == "" {
			return AskRecord{}, fmt.Errorf("%w: an Other answer needs a note describing the alternative", ErrInvalidArgument)
		}
	}
	if !found {
		return AskRecord{}, fmt.Errorf("%w: answer is not one of the allowed answers", ErrInvalidArgument)
	}
	responses, err := normalizeResponses(record.Questions, answer, in.Responses)
	if err != nil {
		return AskRecord{}, err
	}
	notes, err := s.attachmentNotesFor(ctx, record.ID, in.AttachmentNotes)
	if err != nil {
		return AskRecord{}, err
	}
	// answered_by records the ACTUAL verified human; it falls back to the
	// recipient-check actor when the request carried no per-human identity.
	answeredBy := strings.TrimSpace(in.AnsweredBy)
	if answeredBy == "" {
		answeredBy = in.Actor
	}
	identityKind := strings.TrimSpace(in.AnsweredByIdentityKind)
	identitySource := strings.TrimSpace(in.AnsweredByIdentitySource)
	if identityKind != AnswerIdentityKindHuman || answeredBy == "" ||
		(identitySource != AnswerIdentitySourceAccess && identitySource != AnswerIdentitySourceLocal) {
		identityKind, identitySource = AnswerIdentityKindUnknown, ""
	}
	if hook := s.beforeAnswerCommit.Load(); hook != nil {
		(*hook)()
	}
	return s.commitAnswer(ctx, record, answerPlan{
		answer: answer, note: in.Note, late: late, responses: responses, notes: notes,
		answeredBy: answeredBy, identityKind: identityKind, identitySource: identitySource,
	})
}

// answerPlan is a validated answer, checked against one revision of the ask.
type answerPlan struct {
	answer, note                             string
	late                                     bool
	responses                                []AskQuestionResponse
	notes                                    []AskAttachmentNote
	answeredBy, identityKind, identitySource string
}

// commitAnswer stores an answer that was validated against record. The ask must
// still be in the state and at the revision record had: if the requester revised
// it in between, the answers were checked against a form that is gone, so
// nothing is stored and the caller gets ErrAskChanged.
func (s *Service) commitAnswer(ctx context.Context, record AskRecord, plan answerPlan) (AskRecord, error) {
	now := storedTime(s.clock.Now())
	reason := "answered by the operator"
	if plan.late {
		reason = "operator answer after the default supersedes it"
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return AskRecord{}, err
	}
	defer func() { _ = tx.Rollback() }()
	result, err := tx.ExecContext(ctx, `UPDATE asks SET state='answered', reason=?, resolved_at=?, resolution_published_at='', updated_at=? WHERE id=? AND state=? AND revision_count=?`, reason, now, now, record.ID, record.State, record.RevisionCount)
	if err != nil {
		return AskRecord{}, err
	}
	if changed, _ := result.RowsAffected(); changed != 1 {
		current, readErr := scanAsk(tx.QueryRowContext(ctx, askSelect+` WHERE a.id = ?`, record.ID))
		if readErr == nil && current.State == record.State && current.RevisionCount != record.RevisionCount {
			return AskRecord{}, ErrAskChanged
		}
		return AskRecord{}, fmt.Errorf("%w: ask changed while answering", ErrAskResolved)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO answers (id, ask_id, answer, answered_by, answered_at, note, late, responses_json, attachment_notes_json) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, uuid.NewString(), record.ID, plan.answer, plan.answeredBy, now, plan.note, dbvalue.BoolInt(plan.late), encodeResponses(plan.responses), encodeAttachmentNotes(plan.notes)); err != nil {
		return AskRecord{}, fmt.Errorf("%w: ask already has an answer", ErrAskResolved)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO answer_identity (ask_id, identity_kind, identity_source) VALUES (?, ?, ?)`, record.ID, plan.identityKind, plan.identitySource); err != nil {
		return AskRecord{}, fmt.Errorf("persist answer identity: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return AskRecord{}, err
	}
	s.kickResolutions(ctx)
	return s.GetAsk(ctx, record.ID)
}

// Every decision supports a recipient-authored alternative. The distinct
// "other" key is never an alias for a source-defined answer such as "approve";
// consumers must treat it as a non-approval response and use its note as context.
func allowsOtherAnswer(record AskRecord) bool {
	return true
}

// DismissAsk closes an open recipient ask without writing an answers row.
// Repeating the same dismissal is idempotent; terminal answers/defaults remain
// authoritative and cannot be erased by a later dismissal.
func (s *Service) DismissAsk(ctx context.Context, askID, actor, note string) (AskRecord, error) {
	record, err := s.GetAsk(ctx, askID)
	if err != nil {
		return AskRecord{}, err
	}
	if strings.TrimSpace(actor) == "" || strings.TrimSpace(actor) != record.Recipient {
		return AskRecord{}, ErrNotFound
	}
	if record.State == AskDismissed {
		return record, nil
	}
	if !record.Open() {
		return AskRecord{}, fmt.Errorf("%w: ask is already %s", ErrAskResolved, record.State)
	}
	note = strings.TrimSpace(note)
	if len(note) > 2_000 {
		return AskRecord{}, fmt.Errorf("%w: dismissal note exceeds 2000 bytes", ErrInvalidArgument)
	}
	reason := "dismissed by recipient"
	if note != "" {
		reason += ": " + note
	}
	now := storedTime(s.clock.Now())
	result, err := s.db.ExecContext(ctx, `UPDATE asks SET state='dismissed', reason=?, resolved_at=?, resolution_published_at='', updated_at=? WHERE id=? AND state IN ('pending','escalated')`, reason, now, now, record.ID)
	if err != nil {
		return AskRecord{}, err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return AskRecord{}, err
	}
	if changed != 1 {
		return AskRecord{}, fmt.Errorf("%w: ask changed while dismissing", ErrAskResolved)
	}
	s.kickResolutions(ctx)
	return s.GetAsk(ctx, record.ID)
}

// WithdrawDirectAsk lets a direct requester close a pending ask it may read: a
// reader outside the ask's audience is told the ask does not exist, exactly as
// a read is. source is the source the requester's asks are stored under
// (identity.Producer.AskSource). Provider-owned event decisions retain their
// source-specific withdrawal contracts so an ask ID alone cannot cancel another
// owner's workflow.
func (s *Service) WithdrawDirectAsk(ctx context.Context, askID, source, reason string, audience AskAudience) (AskRecord, error) {
	record, err := s.GetAsk(ctx, askID)
	if err != nil {
		return AskRecord{}, err
	}
	source = strings.TrimSpace(source)
	if source == "" || source != record.Source || record.SourceEventType != "" || !audience.Sees(record) {
		return AskRecord{}, ErrNotFound
	}
	if record.State == AskWithdrawn {
		return record, nil
	}
	if !record.Open() {
		return AskRecord{}, fmt.Errorf("%w: ask is already %s", ErrAskResolved, record.State)
	}
	reason = strings.TrimSpace(reason)
	if reason == "" || len(reason) > 2_000 {
		return AskRecord{}, fmt.Errorf("%w: a withdrawal reason of 1-2000 bytes is required", ErrInvalidArgument)
	}
	now := storedTime(s.clock.Now())
	result, err := s.db.ExecContext(ctx, `UPDATE asks SET state='withdrawn', reason=?, resolved_at=?, resolution_published_at='', updated_at=? WHERE id=? AND source=? AND source_event_type='' AND state IN ('pending','escalated')`, reason, now, now, record.ID, source)
	if err != nil {
		return AskRecord{}, err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return AskRecord{}, err
	}
	if changed != 1 {
		return AskRecord{}, fmt.Errorf("%w: ask changed while withdrawing", ErrAskResolved)
	}
	s.kickResolutions(ctx)
	return s.GetAsk(ctx, record.ID)
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
	nowText := storedTime(now)
	if record.FirstDeliveredAt == "" {
		// A default is withheld until the operator was reached, but an ask that
		// never reached them must not stay open forever.
		if deadline, parseErr := time.Parse(time.RFC3339Nano, record.Deadline); parseErr == nil && !now.Before(deadline.Add(neverDeliveredExpiry)) {
			result, execErr := s.db.ExecContext(ctx, `UPDATE asks SET state='expired', reason=?, resolved_at=?, resolution_published_at='', updated_at=? WHERE id=? AND state IN ('pending','escalated')`, neverDeliveredExpiredReason, nowText, nowText, record.ID)
			if execErr != nil {
				return execErr
			}
			if changed, _ := result.RowsAffected(); changed == 1 {
				s.kickResolutions(ctx)
			}
			return nil
		}
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

// recordPath is the page a push for a notification opens: the Hub's own record
// of it, as a path relative to the UI's service-worker scope (so it keeps the
// base path the app is served under). An ask opens its decision; any other
// notification opens its delivery record.
func (s *Service) recordPath(ctx context.Context, notificationID string) string {
	var id string
	if err := s.db.QueryRowContext(ctx, `SELECT id FROM asks WHERE notification_id = ?`, notificationID).Scan(&id); err == nil {
		return "asks/" + url.PathEscape(id)
	}
	return "delivery/" + url.PathEscape(notificationID)
}

// kickResolutions runs after a transition that may need publishing: it wakes
// Wait callers, then drains the ask outboxes in the background.
func (s *Service) kickResolutions(ctx context.Context) {
	s.askChanges.notify()
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
		if err := s.publishAskEvents(publishCtx); err != nil {
			s.log.Printf("publish ask events: %v", err)
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
	var afterTime, afterID string
	err := s.db.QueryRowContext(ctx, `SELECT resolved_at, ask_id FROM resolution_outbox_cursor WHERE singleton=1`).Scan(&afterTime, &afterID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, resolved_at FROM asks WHERE resolved_at != '' AND resolution_published_at = '' AND (?='' OR resolved_at > ? OR (resolved_at=? AND id>?)) ORDER BY resolved_at ASC, id ASC LIMIT 50`, afterTime, afterTime, afterTime, afterID)
	if err != nil {
		return err
	}
	var ids []string
	lastTime := ""
	for rows.Next() {
		var id, resolved string
		if scanErr := rows.Scan(&id, &resolved); scanErr != nil {
			rows.Close()
			return scanErr
		}
		ids = append(ids, id)
		lastTime = resolved
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	var publishErrors []error
	for _, id := range ids {
		record, err := s.GetAsk(ctx, id)
		if err != nil {
			return err
		}
		if err := publisher.PublishDomainEvent(ctx, AskResolvedEvent(record)); err != nil {
			publishErrors = append(publishErrors, fmt.Errorf("publish %s for ask %s: %w", AskResolvedEventType, id, err))
			continue
		}
		if _, err := s.db.ExecContext(ctx, `UPDATE asks SET resolution_published_at=? WHERE id=? AND resolved_at=?`, storedTime(s.clock.Now()), id, record.ResolvedAt); err != nil {
			return err
		}
	}
	if len(ids) == 50 {
		_, err = s.db.ExecContext(ctx, `INSERT INTO resolution_outbox_cursor(singleton, resolved_at, ask_id) VALUES(1,?,?) ON CONFLICT(singleton) DO UPDATE SET resolved_at=excluded.resolved_at, ask_id=excluded.ask_id`, lastTime, ids[len(ids)-1])
	} else {
		_, err = s.db.ExecContext(ctx, `DELETE FROM resolution_outbox_cursor WHERE singleton=1`)
	}
	return errors.Join(append(publishErrors, err)...)
}

// AskResolvedEvent is the requester notification for one resolution. The
// payload is facts only; consumers match on source and requester_ref (the
// event id or idempotency key that opened the ask) or on correlation.
func AskResolvedEvent(record AskRecord) eventbus.DomainEvent {
	occurred, _ := time.Parse(time.RFC3339Nano, record.ResolvedAt)
	event := eventbus.DomainEvent{
		Source:    "notification-hub",
		EventType: AskResolvedEventType,
		Occurred:  occurred.UTC(),
		Payload: map[string]any{
			"ask_id":                      record.ID,
			"state":                       record.State,
			"answer":                      record.Answer,
			"answer_label":                record.AnswerLabel,
			"note":                        record.Note,
			"answered_by":                 record.AnsweredBy,
			"answered_by_identity_kind":   record.AnsweredByIdentityKind,
			"answered_by_identity_source": record.AnsweredByIdentitySource,
			"answered_at":                 record.AnsweredAt,
			"by_default":                  record.State == AskDefaulted,
			"late":                        record.Late,
			"default_answer":              record.DefaultAnswer,
			"reason":                      record.Reason,
			"source":                      record.Source,
			"source_event_type":           record.SourceEventType,
			"requester_ref":               record.RequesterRef,
			"correlation":                 correlationPayload(record.Correlation),
			"resolved_at":                 record.ResolvedAt,
			"subject_key":                 record.SubjectKey,
			"superseded_by_ask_id":        record.SupersededByAskID,
			"responses":                   ResponsePayload(record.Responses),
			"attachment_notes":            AttachmentNotePayload(record.AttachmentNotes),
			"answer_summary":              record.AnswerSummary,
		},
	}
	if record.ProviderDecisionContext != nil {
		event.Payload["provider_decision_context"] = map[string]any{
			"schema_version":         record.ProviderDecisionContext.SchemaVersion,
			"canonical_terms_sha256": record.ProviderDecisionContext.CanonicalTermsSHA256,
			"scope":                  record.ProviderDecisionContext.Scope,
		}
	}
	return event
}
