package supervision

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/effortdecision"
	"agent-manager/internal/goalhome"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/operatordecision"
	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Decision states and answer channels; see effortdecision.
const (
	DecisionPending   = effortdecision.Pending
	DecisionAnswered  = effortdecision.Answered
	DecisionDefaulted = effortdecision.Defaulted
	DecisionExpired   = effortdecision.Expired

	DecisionViaPhone   = effortdecision.ViaPhone
	DecisionViaDesktop = effortdecision.ViaDesktop
	DecisionViaDefault = effortdecision.ViaDefault

	// AskResolvedEventType is notification-hub's resolution event. It re-exports
	// the shared substrate constant so every resolver agrees on the one value.
	AskResolvedEventType = operatordecision.AskResolvedEventType
)

var (
	// ErrDecisionChanged refuses a re-raise whose content differs; a changed
	// question is a new decision with a new id.
	ErrDecisionChanged = errors.New("a decision with this id already exists with different content; raise the changed question under a new id")
	// ErrDecisionAnswered refuses a desktop answer that contradicts an
	// operator answer already recorded.
	ErrDecisionAnswered = errors.New("the decision already has a different operator answer")
	// ErrDecisionNotAnswered refuses marking an unanswered or newer answer recorded.
	ErrDecisionNotAnswered = errors.New("the decision has no answer at that time; read it again")
)

// DecisionPublisher publishes the decision_requested event
// (api-core eventbus.Client in production). It is an alias for the shared
// operatordecision.Publisher.
type DecisionPublisher = operatordecision.Publisher

// SetDecisionPublisher wires the Vrooli Events publisher. Without one,
// decisions are stored and stay unpublished until a publisher is set.
func (s *EffortService) SetDecisionPublisher(p DecisionPublisher) { s.decisionPublisher = p }

// DecisionResolution is notification-hub's ask.resolved payload. It is an alias
// for the shared operatordecision.DecisionResolution, so the webhook decodes
// once into the substrate type and every resolver reads the same struct.
type DecisionResolution = operatordecision.DecisionResolution

// --- storage ---------------------------------------------------------------

func (r *Repository) InsertEffortDecision(ctx context.Context, d *pb.EffortDecision, digest string) (bool, error) {
	b, err := effortJSON.Marshal(d)
	if err != nil {
		return false, err
	}
	now := formatTime(r.now())
	result, err := r.db.ExecContext(ctx, `INSERT INTO supervision_effort_decisions(goal_home,decision_id,effort_ref,event_id,content_digest,decision_json,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?) ON CONFLICT(goal_home,decision_id) DO NOTHING`, d.GoalHome, d.DecisionId, d.EffortRef, d.EventId, digest, string(b), now, now)
	if err != nil {
		return false, err
	}
	n, _ := result.RowsAffected()
	return n == 1, nil
}

func (r *Repository) GetEffortDecision(ctx context.Context, home, id string) (*pb.EffortDecision, string, error) {
	var b, digest string
	err := r.db.QueryRowContext(ctx, `SELECT decision_json,content_digest FROM supervision_effort_decisions WHERE goal_home=? AND decision_id=?`, home, id).Scan(&b, &digest)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, "", ErrNotFound
	}
	if err != nil {
		return nil, "", err
	}
	d := &pb.EffortDecision{}
	return d, digest, effortRead.Unmarshal([]byte(b), d)
}

func (r *Repository) FindEffortDecisionByEvent(ctx context.Context, eventID string) (*pb.EffortDecision, error) {
	var home, id string
	err := r.db.QueryRowContext(ctx, `SELECT goal_home,decision_id FROM supervision_effort_decisions WHERE event_id=?`, eventID).Scan(&home, &id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	d, _, err := r.GetEffortDecision(ctx, home, id)
	return d, err
}

func (r *Repository) SaveEffortDecision(ctx context.Context, d *pb.EffortDecision) error {
	b, err := effortJSON.Marshal(d)
	if err != nil {
		return err
	}
	result, err := r.db.ExecContext(ctx, `UPDATE supervision_effort_decisions SET effort_ref=?,decision_json=?,updated_at=? WHERE goal_home=? AND decision_id=?`, d.EffortRef, string(b), formatTime(r.now()), d.GoalHome, d.DecisionId)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n != 1 {
		return ErrNotFound
	}
	return nil
}

// ListEffortDecisions lists decisions newest first, filtered by goal home
// and/or effort when given.
func (r *Repository) ListEffortDecisions(ctx context.Context, home, effortRef string, limit int) ([]*pb.EffortDecision, error) {
	rows, err := r.db.QueryxContext(ctx, `SELECT decision_json FROM supervision_effort_decisions WHERE (?='' OR goal_home=?) AND (?='' OR effort_ref=?) ORDER BY created_at DESC, decision_id LIMIT ?`, home, home, effortRef, effortRef, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*pb.EffortDecision
	for rows.Next() {
		var b string
		if err = rows.Scan(&b); err != nil {
			return nil, err
		}
		d := &pb.EffortDecision{}
		if err = effortRead.Unmarshal([]byte(b), d); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// --- service ---------------------------------------------------------------

// NormalizeGoalHome cleans a repository-relative goal-home path.
func NormalizeGoalHome(home string) (string, error) { return effortdecision.NormalizeGoalHome(home) }

// DecisionView converts the record to the goal-home view the renderers use.
func DecisionView(d *pb.EffortDecision) goalhome.Decision { return effortdecision.View(d) }

// decisionContentDigest covers what the operator is asked, so an identical
// re-raise is a no-op and a changed one is refused.
func decisionContentDigest(d *pb.EffortDecision) string {
	cut := &pb.EffortDecision{
		GoalHome: d.GoalHome, DecisionId: d.DecisionId, Question: d.Question, Options: d.Options, Recommended: d.Recommended,
		RecommendationReason: d.RecommendationReason, DefaultAnswer: d.DefaultAnswer, Reversible: d.Reversible,
		Urgency: d.Urgency, Note: d.Note, BigPicture: d.BigPicture, TechnicalDetails: d.TechnicalDetails,
		ContextUrl: d.ContextUrl, Deadline: d.Deadline,
	}
	return effortDigest(cut)
}

func decisionEvent(d *pb.EffortDecision) eventbus.DomainEvent {
	return effortdecision.RequestedEvent(d)
}

func (s *EffortService) effortForGoalHome(ctx context.Context, home string) string {
	efforts, err := s.repo.ListEfforts(ctx, "", 1001)
	if err != nil {
		return ""
	}
	for _, e := range efforts {
		if e.Withdrawn {
			continue
		}
		if destination, ok := goalHomeDestination(e.GetDestinationRef()); ok && destination == home {
			return e.EffortRef
		}
	}
	return ""
}

// RaiseDecision stores one decision per (goal home, decision id) and
// publishes its event once. An identical re-raise returns the stored record
// (and retries an unpublished event); a changed one is refused.
func (s *EffortService) RaiseDecision(ctx context.Context, req *pb.RaiseEffortDecisionRequest, raisedBy string) (*pb.RaiseEffortDecisionResponse, error) {
	if req == nil || req.Decision == nil {
		return nil, errors.New("decision required")
	}
	d := proto.Clone(req.Decision).(*pb.EffortDecision)
	home, err := NormalizeGoalHome(d.GoalHome)
	if err != nil {
		return nil, err
	}
	d.GoalHome = home
	d.DecisionId = strings.TrimSpace(d.DecisionId)
	d.Question = strings.TrimSpace(d.Question)
	if err := DecisionView(d).Validate(); err != nil {
		return nil, err
	}
	// Answer and lifecycle fields are owned here, never by the caller.
	now := s.now().UTC().Truncate(time.Millisecond)
	stored := &pb.EffortDecision{
		EffortRef: strings.TrimSpace(d.EffortRef), GoalHome: d.GoalHome, DecisionId: d.DecisionId, Question: d.Question,
		Options: d.Options, Recommended: d.Recommended, RecommendationReason: d.RecommendationReason,
		DefaultAnswer: d.DefaultAnswer, Reversible: d.Reversible, Urgency: d.Urgency, Note: d.Note,
		BigPicture: strings.TrimSpace(d.BigPicture), TechnicalDetails: strings.TrimSpace(d.TechnicalDetails),
		ContextUrl: d.ContextUrl, Deadline: d.Deadline, State: DecisionPending, RaisedByRunId: raisedBy,
		CreatedAt: timestamppb.New(now),
	}
	s.decisionMu.Lock()
	defer s.decisionMu.Unlock()
	if stored.EffortRef == "" {
		stored.EffortRef = s.effortForGoalHome(ctx, home)
	}
	stored.EventId = effortdecision.EventID(decisionEvent(stored))
	digest := decisionContentDigest(stored)
	inserted, err := s.repo.InsertEffortDecision(ctx, stored, digest)
	if err != nil {
		return nil, err
	}
	if !inserted {
		existing, existingDigest, err := s.repo.GetEffortDecision(ctx, home, stored.DecisionId)
		if err != nil {
			return nil, err
		}
		if existingDigest != digest {
			return nil, ErrDecisionChanged
		}
		stored = existing
	}
	published := false
	if stored.PublishedAt == nil {
		published = s.publishDecision(ctx, stored)
	}
	return &pb.RaiseEffortDecisionResponse{Decision: stored, Created: inserted, Published: published}, nil
}

// publishDecision publishes the event and records the outcome on the
// record. Callers hold decisionMu.
func (s *EffortService) publishDecision(ctx context.Context, d *pb.EffortDecision) bool {
	if s.decisionPublisher == nil {
		d.PublishError = "Vrooli Events publisher unavailable"
	} else if err := s.decisionPublisher.PublishDomainEvent(ctx, decisionEvent(d)); err != nil {
		d.PublishError = err.Error()
	} else {
		d.PublishError = ""
		d.PublishedAt = timestamppb.New(s.now().UTC())
	}
	if err := s.repo.SaveEffortDecision(ctx, d); err != nil {
		d.PublishError = "record publish outcome: " + err.Error()
		return false
	}
	return d.PublishedAt != nil
}

// publishPendingDecisions retries pending decisions whose event was never
// published. The event id is deterministic, so a retry after a lost
// acknowledgement is deduplicated by Vrooli Events and the hub.
func (s *EffortService) publishPendingDecisions(ctx context.Context) error {
	if s.decisionPublisher == nil {
		return nil
	}
	s.decisionMu.Lock()
	defer s.decisionMu.Unlock()
	decisions, err := s.repo.ListEffortDecisions(ctx, "", "", 200)
	if err != nil {
		return err
	}
	for _, d := range decisions {
		if d.PublishedAt == nil && d.State == DecisionPending {
			s.publishDecision(ctx, d)
		}
	}
	return nil
}

// ListDecisions lists decisions for a goal home or effort.
func (s *EffortService) ListDecisions(ctx context.Context, req *pb.ListEffortDecisionsRequest) (*pb.ListEffortDecisionsResponse, error) {
	if req == nil {
		req = &pb.ListEffortDecisionsRequest{}
	}
	home := ""
	if strings.TrimSpace(req.GoalHome) != "" {
		var err error
		if home, err = NormalizeGoalHome(req.GoalHome); err != nil {
			return nil, err
		}
	}
	limit := int(req.Limit)
	if limit <= 0 || limit > 200 {
		limit = 200
	}
	decisions, err := s.repo.ListEffortDecisions(ctx, home, strings.TrimSpace(req.EffortRef), limit)
	if err != nil {
		return nil, err
	}
	out := &pb.ListEffortDecisionsResponse{}
	for _, d := range decisions {
		if req.UnrecordedOnly && d.RecordedAt != nil {
			continue
		}
		out.Decisions = append(out.Decisions, d)
	}
	return out, nil
}

// AnswerDecision records a desktop operator answer. A default is superseded
// (late); a different earlier operator answer is refused.
func (s *EffortService) AnswerDecision(ctx context.Context, req *pb.AnswerEffortDecisionRequest, actor EffortActor) (*pb.EffortDecision, error) {
	d, answered, err := s.answerDecision(ctx, req, actor)
	if err == nil && answered {
		s.wakeParkedOrchestrators(ctx, d)
	}
	return d, err
}

func (s *EffortService) answerDecision(ctx context.Context, req *pb.AnswerEffortDecisionRequest, actor EffortActor) (*pb.EffortDecision, bool, error) {
	if !actor.Operator || actor.ID == "" {
		return nil, false, errors.New("operator authority required to answer a decision")
	}
	if req == nil {
		return nil, false, errors.New("answer required")
	}
	home, err := NormalizeGoalHome(req.GoalHome)
	if err != nil {
		return nil, false, err
	}
	s.decisionMu.Lock()
	defer s.decisionMu.Unlock()
	d, _, err := s.repo.GetEffortDecision(ctx, home, strings.TrimSpace(req.DecisionId))
	if err != nil {
		return nil, false, err
	}
	answer := strings.TrimSpace(req.Answer)
	if !hasDecisionOption(d, answer) {
		return nil, false, fmt.Errorf("answer %q is not one of the options", answer)
	}
	if d.State == DecisionAnswered {
		if d.Answer == answer {
			// The decision write may have succeeded before the receiver signal
			// failed. Preserve the original answer fields and repair that effect.
			return d, true, s.signalDecisionAnswer(ctx, d)
		}
		return nil, false, ErrDecisionAnswered
	}
	d.Late = d.State == DecisionDefaulted
	s.setDecisionAnswer(d, DecisionAnswered, answer, req.Note, DecisionViaDesktop, s.now().UTC(), "answered on the desktop by "+actor.ID)
	if err := s.repo.SaveEffortDecision(ctx, d); err != nil {
		return nil, false, err
	}
	return d, true, s.signalDecisionAnswer(ctx, d)
}

// ResolveDecisionFromHub applies notification-hub's ask.resolved event. It
// returns nil, nil for an event about an ask this effort log did not raise.
// Duplicate deliveries change nothing. An operator answer is never replaced
// by a later default; a later operator answer supersedes a default.
func (s *EffortService) ResolveDecisionFromHub(ctx context.Context, resolution DecisionResolution) (*pb.EffortDecision, error) {
	d, answered, err := s.resolveDecisionFromHub(ctx, resolution)
	if err == nil && answered {
		s.wakeParkedOrchestrators(ctx, d)
	}
	return d, err
}

func (s *EffortService) resolveDecisionFromHub(ctx context.Context, resolution DecisionResolution) (*pb.EffortDecision, bool, error) {
	if resolution.SourceEventType != "" && resolution.SourceEventType != goalhome.DecisionRequestedEventType {
		return nil, false, nil
	}
	s.decisionMu.Lock()
	defer s.decisionMu.Unlock()
	var d *pb.EffortDecision
	var err error
	if home, id := resolution.Correlation["goal_home"], resolution.Correlation["decision_id"]; home != "" && id != "" {
		d, _, err = s.repo.GetEffortDecision(ctx, home, id)
	} else if resolution.RequesterRef != "" {
		d, err = s.repo.FindEffortDecisionByEvent(ctx, resolution.RequesterRef)
	} else {
		return nil, false, nil
	}
	if errors.Is(err, ErrNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	// The correlation identifies the logical decision, while RequesterRef binds
	// the Hub resolution to the exact immutable request event that opened it.
	// Hub always carries this reference; a missing or stale reference must not
	// answer the current decision generation.
	if resolution.RequesterRef == "" || resolution.RequesterRef != d.GetEventId() {
		return nil, false, nil
	}
	if resolution.AskID != "" {
		d.AskId = resolution.AskID
	}
	answeredAt := s.now().UTC()
	if at, parseErr := time.Parse(time.RFC3339Nano, resolution.AnsweredAt); parseErr == nil {
		answeredAt = at.UTC()
	}
	changed := false
	answerSignalNeeded := false
	switch resolution.State {
	case "expired":
		if d.State == DecisionPending {
			d.State, d.ResolutionReason = DecisionExpired, resolution.Reason
			changed = true
		}
	case "defaulted":
		if d.State == DecisionPending || d.State == DecisionExpired {
			s.setDecisionAnswer(d, DecisionDefaulted, resolution.Answer, "", DecisionViaDefault, answeredAt, resolution.Reason)
			d.AnswerLabel = resolution.AnswerLabel
			changed = true
			answerSignalNeeded = true
		} else if d.State == DecisionDefaulted && d.Answer == resolution.Answer {
			// A previous default may be durable even though its signal append
			// failed. Redelivery repairs the append without replacing its time.
			answerSignalNeeded = true
		}
	case "answered":
		same := d.State == DecisionAnswered && d.Answer == resolution.Answer
		if same {
			// The decision write may have succeeded while the signal append failed.
			// Reuse the signal's existing dedupe key to repair that partial effect.
			answerSignalNeeded = true
		} else if hasDecisionOption(d, resolution.Answer) {
			d.Late = resolution.Late || d.State == DecisionDefaulted
			s.setDecisionAnswer(d, DecisionAnswered, resolution.Answer, resolution.Note, DecisionViaPhone, answeredAt, "answered via phone by "+resolution.AnsweredBy)
			d.AnswerLabel = resolution.AnswerLabel
			changed = true
			answerSignalNeeded = true
		}
	default:
		return d, false, nil
	}
	if changed {
		if err := s.repo.SaveEffortDecision(ctx, d); err != nil {
			return nil, false, err
		}
	}
	if answerSignalNeeded {
		return d, true, s.signalDecisionAnswer(ctx, d)
	}
	return d, false, nil
}

func (s *EffortService) setDecisionAnswer(d *pb.EffortDecision, state, answer, note, via string, at time.Time, reason string) {
	d.State, d.Answer, d.AnswerNote, d.AnsweredVia = state, answer, note, via
	d.AnswerLabel = DecisionView(d).Label(answer)
	d.AnsweredAt = timestamppb.New(at)
	d.ResolutionReason = reason
	d.RecordedAt = nil
}

func hasDecisionOption(d *pb.EffortDecision, key string) bool {
	for _, option := range d.GetOptions() {
		if option.GetKey() == key {
			return true
		}
	}
	return false
}

// signalDecisionAnswer appends one operator_answer signal per answer.
func (s *EffortService) signalDecisionAnswer(ctx context.Context, d *pb.EffortDecision) error {
	if d.EffortRef == "" {
		d.EffortRef = s.effortForGoalHome(ctx, d.GoalHome)
		if d.EffortRef == "" {
			return nil
		}
		if err := s.repo.SaveEffortDecision(ctx, d); err != nil {
			return err
		}
	}
	at := d.GetAnsweredAt().AsTime().UTC().Format(time.RFC3339Nano)
	detail := fmt.Sprintf("decision %s %s: %s (%s); run `agent-manager effort decision sync %s`", d.DecisionId, d.State, d.Answer, d.AnsweredVia, d.GoalHome)
	_, _, err := s.repo.AppendEffortSignal(ctx, d.EffortRef, SignalOperatorAnswer, "", detail, SignalOperatorAnswer+":"+d.GoalHome+"|"+d.DecisionId+"|"+d.State+"|"+d.Answer+"|"+at)
	return err
}

// wakeParkedOrchestrators wakes the effort's parked orchestrator runs with
// the answer, so a parked orchestrator records it at once. Event-driven
// check-ins start on the operator_answer signal instead; a run that is not
// parked is left alone. A failed wake leaves the signal for the next check-in.
func (s *EffortService) wakeParkedOrchestrators(ctx context.Context, d *pb.EffortDecision) {
	if s.controller == nil || d == nil || d.EffortRef == "" {
		return
	}
	e, _, err := s.repo.GetEffort(ctx, d.EffortRef)
	if err != nil || e == nil || e.Withdrawn {
		return
	}
	result := fmt.Sprintf("OPERATOR_DECISION %s %s: %s; run `agent-manager effort decision sync %s` first", d.DecisionId, d.State, d.Answer, d.GoalHome)
	for _, subject := range e.GetSubjects() {
		if subject.GetRole() != "orchestrator" || subject.GetRunId() == "" {
			continue
		}
		id, err := uuid.Parse(subject.GetRunId())
		if err != nil {
			continue
		}
		if run, err := s.controller.GetRun(ctx, id); err != nil || run == nil || run.Status != domain.RunStatusParked {
			continue
		}
		_ = s.controller.WakeRun(ctx, id, result)
	}
}

// MarkDecisionRecorded records that sync wrote the answer given at
// answered_at into FEEDBACK.md. It refuses a stale answered_at, so an answer
// that arrived after the sync read is never marked recorded.
func (s *EffortService) MarkDecisionRecorded(ctx context.Context, req *pb.MarkEffortDecisionRecordedRequest) (*pb.EffortDecision, error) {
	if req == nil {
		return nil, errors.New("request required")
	}
	home, err := NormalizeGoalHome(req.GoalHome)
	if err != nil {
		return nil, err
	}
	s.decisionMu.Lock()
	defer s.decisionMu.Unlock()
	d, _, err := s.repo.GetEffortDecision(ctx, home, strings.TrimSpace(req.DecisionId))
	if err != nil {
		return nil, err
	}
	if (d.State != DecisionAnswered && d.State != DecisionDefaulted) || req.AnsweredAt == nil || d.AnsweredAt == nil || !d.AnsweredAt.AsTime().Equal(req.AnsweredAt.AsTime()) {
		return nil, ErrDecisionNotAnswered
	}
	if d.RecordedAt == nil {
		d.RecordedAt = timestamppb.New(s.now().UTC())
		if err := s.repo.SaveEffortDecision(ctx, d); err != nil {
			return nil, err
		}
	}
	return d, nil
}
