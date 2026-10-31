package supervision

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"agent-manager/internal/effortdecision"
	"agent-manager/internal/goalhome"

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
	decisions, _, err := r.ListEffortDecisionPage(ctx, home, effortRef, limit, nil)
	return decisions, err
}

// decisionCursor uses the immutable storage ordering, not updated answer time.
// Scope is carried with the opaque token so changing filters cannot skip rows.
type decisionCursor struct {
	Created    string `json:"created"`
	ID         string `json:"id"`
	Home       string `json:"home"`
	FilterHome string `json:"filter_home"`
	Effort     string `json:"effort"`
	Unrecorded bool   `json:"unrecorded"`
}

func (r *Repository) ListEffortDecisionPage(ctx context.Context, home, effortRef string, limit int, after *decisionCursor) ([]*pb.EffortDecision, *decisionCursor, error) {
	at, id, origin := "", "", ""
	if after != nil {
		at, id, origin = after.Created, after.ID, after.Home
	}
	rows, err := r.db.QueryxContext(ctx, `SELECT decision_json,created_at,goal_home,decision_id FROM supervision_effort_decisions
	 WHERE (?='' OR goal_home=?) AND (?='' OR effort_ref=?)
	 AND (?='' OR created_at < ? OR (created_at=? AND (decision_id > ? OR (decision_id=? AND goal_home > ?))))
	 ORDER BY created_at DESC, decision_id, goal_home LIMIT ?`, home, home, effortRef, effortRef, at, at, at, id, id, origin, limit+1)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var out []*pb.EffortDecision
	var last *decisionCursor
	for rows.Next() {
		var b, created, storedHome, storedID string
		if err = rows.Scan(&b, &created, &storedHome, &storedID); err != nil {
			return nil, nil, err
		}
		if len(out) == limit {
			return out, last, nil
		}
		d := &pb.EffortDecision{}
		if err = effortRead.Unmarshal([]byte(b), d); err != nil {
			return nil, nil, err
		}
		out = append(out, d)
		last = &decisionCursor{Created: created, ID: storedID, Home: storedHome, FilterHome: home, Effort: effortRef}
	}
	return out, nil, rows.Err()
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

// decisionForHome follows the enrolled effort when its home moves. The stored
// home remains part of the original question/event identity: changing it would
// change a pending publication's bytes and break delayed phone resolutions.
func (s *EffortService) decisionForHome(ctx context.Context, home, id string) (*pb.EffortDecision, string, error) {
	d, digest, err := s.repo.GetEffortDecision(ctx, home, id)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, "", err
	}
	ref := s.effortForGoalHome(ctx, home)
	if err == nil {
		if ref == "" || d.EffortRef == "" {
			return d, digest, nil
		}
		if ref != d.EffortRef {
			return nil, "", fmt.Errorf("decision %s belongs to a different effort: %w", id, ErrConflict)
		}
	}
	if ref == "" {
		return nil, "", ErrNotFound
	}
	return s.repo.FindEffortDecisionByEffort(ctx, ref, id)
}

// Identity lookup must not inherit the board/list window: an older answer can
// never become "not found" merely because newer decisions were added. Two rows
// suffice to detect ambiguous legacy homes, including a direct-home match.
func (r *Repository) FindEffortDecisionByEffort(ctx context.Context, ref, id string) (*pb.EffortDecision, string, error) {
	rows, err := r.db.QueryxContext(ctx, `SELECT decision_json,content_digest FROM supervision_effort_decisions WHERE effort_ref=? AND decision_id=? LIMIT 2`, ref, id)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	var found *pb.EffortDecision
	var digest string
	for rows.Next() {
		if found != nil {
			return nil, "", fmt.Errorf("decision %s has conflicting homes within %s: %w", id, ref, ErrConflict)
		}
		var data string
		if err := rows.Scan(&data, &digest); err != nil {
			return nil, "", err
		}
		found = &pb.EffortDecision{}
		if err := effortRead.Unmarshal([]byte(data), found); err != nil {
			return nil, "", err
		}
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	if found == nil {
		return nil, "", ErrNotFound
	}
	return found, digest, nil
}

// currentDecisionHome is a projection, never a rewrite of the original event.
func (s *EffortService) currentDecisionHome(ctx context.Context, d *pb.EffortDecision) string {
	if d.EffortRef != "" {
		if e, _, err := s.repo.GetEffort(ctx, d.EffortRef); err == nil && !e.Withdrawn {
			if home, ok := goalHomeDestination(e.DestinationRef); ok {
				return home
			}
		}
	}
	return d.GoalHome
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
	if enrolled := s.effortForGoalHome(ctx, home); enrolled != "" {
		if stored.EffortRef != "" && stored.EffortRef != enrolled {
			return nil, fmt.Errorf("goal home belongs to a different effort: %w", ErrConflict)
		}
		stored.EffortRef = enrolled
	}
	// Re-raising from a relocated home must find the original decision before
	// inserting anything, including an already answered decision.
	if existing, existingDigest, lookupErr := s.decisionForHome(ctx, home, stored.DecisionId); lookupErr == nil {
		comparison := proto.Clone(stored).(*pb.EffortDecision)
		comparison.GoalHome = existing.GoalHome
		if decisionContentDigest(comparison) != existingDigest {
			return nil, ErrDecisionChanged
		}
		published := false
		if existing.PublishedAt == nil && existing.State == DecisionPending {
			published = s.publishDecision(ctx, existing)
		}
		return &pb.RaiseEffortDecisionResponse{Decision: existing, Published: published}, nil
	} else if !errors.Is(lookupErr, ErrNotFound) {
		return nil, lookupErr
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
	ref := strings.TrimSpace(req.EffortRef)
	if enrolled := s.effortForGoalHome(ctx, home); home != "" && enrolled != "" {
		if ref != "" && ref != enrolled {
			return nil, errors.New("goal home belongs to a different effort")
		}
		// A directory is a location, not the decision's durable identity.
		ref, home = enrolled, ""
	}
	var after *decisionCursor
	if req.PageToken != "" {
		b, err := base64.RawURLEncoding.DecodeString(req.PageToken)
		if err != nil {
			return nil, errors.New("invalid decision page token")
		}
		after = &decisionCursor{}
		if json.Unmarshal(b, after) != nil || after.Created == "" || after.ID == "" || after.Home == "" || after.FilterHome != home || after.Effort != ref || after.Unrecorded != req.UnrecordedOnly {
			return nil, errors.New("invalid decision page token or changed filters")
		}
	}
	decisions, next, err := s.repo.ListEffortDecisionPage(ctx, home, ref, limit, after)
	if err != nil {
		return nil, err
	}
	out := &pb.ListEffortDecisionsResponse{}
	if next != nil {
		next.Unrecorded = req.UnrecordedOnly
		b, err := json.Marshal(next)
		if err != nil {
			return nil, err
		}
		out.NextPageToken = base64.RawURLEncoding.EncodeToString(b)
	}
	for _, d := range decisions {
		// Do not project conflicting historical answers into one current home.
		// Validate even already-recorded decisions before any sync can write.
		if d.EffortRef != "" {
			if _, _, err := s.repo.FindEffortDecisionByEffort(ctx, d.EffortRef, d.DecisionId); err != nil {
				return nil, err
			}
		}
		if req.UnrecordedOnly && d.RecordedAt != nil {
			continue
		}
		view := proto.Clone(d).(*pb.EffortDecision)
		view.GoalHome = s.currentDecisionHome(ctx, d)
		out.Decisions = append(out.Decisions, view)
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
	d, _, err := s.decisionForHome(ctx, home, strings.TrimSpace(req.DecisionId))
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
	detail := fmt.Sprintf("decision %s %s: %s (%s); run `agent-manager effort decision sync %s`", d.DecisionId, d.State, d.Answer, d.AnsweredVia, s.currentDecisionHome(ctx, d))
	_, _, err := s.repo.AppendEffortSignal(ctx, d.EffortRef, SignalOperatorAnswer, "", detail, SignalOperatorAnswer+":"+d.GoalHome+"|"+d.DecisionId+"|"+d.State+"|"+d.Answer+"|"+at)
	return err
}

// wakeParkedOrchestrators preserves the immediate operator-answer check-in
// through the same durable signal cursor used by periodic feedback dispatch.
// The owner rereads the authoritative signal log; answer prose is not passed
// as a command payload. Failed wakes remain queued for the next scheduler tick.
func (s *EffortService) wakeParkedOrchestrators(ctx context.Context, d *pb.EffortDecision) {
	if s.controller == nil || s.signalCursorStore == nil || d == nil || d.EffortRef == "" {
		return
	}
	waker, ok := s.controller.(EffortSignalWakeController)
	if !ok {
		return
	}
	e, _, err := s.repo.GetEffort(ctx, d.EffortRef)
	if err != nil || e == nil || e.Withdrawn {
		return
	}
	for _, subject := range orchestratorSignalSubjects(e) {
		_ = s.dispatchEffortSignalSubject(ctx, waker, e, subject)
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
	d, _, err := s.decisionForHome(ctx, home, strings.TrimSpace(req.DecisionId))
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
