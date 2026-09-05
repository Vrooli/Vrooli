package supervision

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"agent-manager/internal/eventlog"
	"agent-manager/internal/invocationreadmodel"
	"agent-manager/internal/repository"
	"agent-manager/internal/runsignal"

	"github.com/google/uuid"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type SubjectSummary struct {
	RunID               string
	Status              string
	Terminal            bool
	FrictionScore       float64
	EvidenceIDs         []string
	Friction            []FrictionSummary
	FrictionUnavailable bool
	FrictionThrough     time.Time
}

type SubjectResolver interface {
	Resolve(context.Context, []*domainpb.WatchSubject) ([]SubjectSummary, error)
}

type FrictionSummary struct {
	EvidenceID  string  `json:"evidence_id"`
	Score       float64 `json:"score"`
	Pattern     string  `json:"pattern"`
	Fingerprint string  `json:"fingerprint"`
	Owner       string  `json:"owner"`
}
type FrictionSource interface {
	Episodes(context.Context, invocationreadmodel.Filter, int) ([]invocationreadmodel.Episode, error)
	Watermark(context.Context, string) (*invocationreadmodel.Watermark, error)
}
type RunSubjectResolver struct {
	runs     repository.RunRepository
	friction FrictionSource
	now      func() time.Time
}

func NewRunSubjectResolver(runs repository.RunRepository, sources ...FrictionSource) *RunSubjectResolver {
	r := &RunSubjectResolver{runs: runs, now: time.Now}
	if len(sources) > 0 {
		r.friction = sources[0]
	}
	return r
}

func (r *RunSubjectResolver) Resolve(ctx context.Context, subjects []*domainpb.WatchSubject) ([]SubjectSummary, error) {
	summaries := make([]SubjectSummary, 0, len(subjects))
	for _, subject := range subjects {
		runID, err := uuid.Parse(subject.GetRunId())
		if err != nil {
			return nil, err
		}
		run, err := r.runs.Get(ctx, runID)
		if err != nil {
			return nil, err
		}
		summary := SubjectSummary{RunID: subject.GetRunId(), Status: string(run.Status), Terminal: run.Status.IsTerminal(), EvidenceIDs: []string{"run:" + subject.GetRunId()}}
		summary.FrictionUnavailable = true
		if r.friction != nil {
			watermark, readErr := r.friction.Watermark(ctx, subject.GetRunId())
			if readErr == nil && watermark != nil && watermark.EpisodeClassifierVersion == runsignal.EpisodeClassifierVersion {
				from := r.now().UTC().Add(-15 * time.Minute)
				episodes, err := r.friction.Episodes(ctx, invocationreadmodel.Filter{RunID: subject.GetRunId(), From: &from}, 4)
				if err == nil {
					summary.FrictionUnavailable = false
					summary.FrictionThrough = watermark.LastEventAt
					for _, episode := range episodes {
						score := 0.0
						switch episode.Severity {
						case "critical":
							score = 1
						case "high":
							score = .9
						case "medium":
							score = .6
						case "low":
							score = .3
						default:
							continue
						}
						summary.FrictionScore = max(summary.FrictionScore, score)
						summary.Friction = append(summary.Friction, FrictionSummary{EvidenceID: episode.EpisodeID, Score: score, Pattern: episode.Pattern, Fingerprint: episode.Fingerprint, Owner: episode.SuspectedOwnerScenario})
						summary.EvidenceIDs = append(summary.EvidenceIDs, episode.EpisodeID)
					}
				}
			}
		}
		summaries = append(summaries, summary)
	}
	return summaries, nil
}

type EvaluationInput struct {
	Watch          *domainpb.CohortWatch
	Events         []eventlog.CohortEvent
	Subjects       []SubjectSummary
	Now            time.Time
	Reset          bool
	ResetFrom      int64
	ResetTo        int64
	ProposedCursor string
}

type Evaluator interface {
	Evaluate(context.Context, EvaluationInput) (*domainpb.WatchDecision, error)
}

type TriggerEvaluator struct{}

func (TriggerEvaluator) Evaluate(_ context.Context, input EvaluationInput) (*domainpb.WatchDecision, error) {
	decision := &domainpb.WatchDecision{Disposition: domainpb.WatchDisposition_WATCH_DISPOSITION_QUIET, Classification: "quiet"}
	for _, event := range input.Events {
		decision.EvidenceIds = append(decision.EvidenceIds, event.ID.String())
	}
	for _, subject := range input.Subjects {
		decision.EvidenceIds = append(decision.EvidenceIds, subject.EvidenceIDs...)
	}
	if input.Reset {
		decision.Disposition = domainpb.WatchDisposition_WATCH_DISPOSITION_CURSOR_RESET
		decision.Classification = fmt.Sprintf("retention_generation_changed:%d:%d", input.ResetFrom, input.ResetTo)
		decision.RecommendedAction = domainpb.WatchActionKind_WATCH_ACTION_KIND_WAKE_PARENT
		return decision, nil
	}
	triggers := input.Watch.GetSpec().GetTriggers()
	allTerminal := len(input.Subjects) > 0
	for _, subject := range input.Subjects {
		allTerminal = allTerminal && subject.Terminal
		if triggers.GetFrictionScore() > 0 && subject.FrictionScore >= triggers.GetFrictionScore() {
			decision.Disposition, decision.Classification = domainpb.WatchDisposition_WATCH_DISPOSITION_SIGNAL, "friction_threshold"
		}
	}
	if triggers.GetTerminal() && allTerminal {
		decision.Disposition, decision.Classification = domainpb.WatchDisposition_WATCH_DISPOSITION_TERMINAL, "cohort_terminal"
	} else if triggers.GetDeadline().IsValid() && !input.Now.Before(triggers.GetDeadline().AsTime()) {
		decision.Disposition, decision.Classification = domainpb.WatchDisposition_WATCH_DISPOSITION_SIGNAL, "deadline"
	} else if triggers.GetEventCount() > 0 && len(input.Events) >= int(triggers.GetEventCount()) {
		decision.Disposition, decision.Classification = domainpb.WatchDisposition_WATCH_DISPOSITION_SIGNAL, "event_count"
	} else if triggers.GetQuietTime().IsValid() && triggers.GetQuietTime().AsDuration() > 0 {
		lastActivity := input.Watch.GetUpdatedAt().AsTime()
		if len(input.Events) > 0 {
			lastActivity = input.Events[len(input.Events)-1].Timestamp
		}
		if !input.Now.Before(lastActivity.Add(triggers.GetQuietTime().AsDuration())) {
			decision.Disposition, decision.Classification = domainpb.WatchDisposition_WATCH_DISPOSITION_SIGNAL, "quiet_time"
		}
	}

	nextWake := input.Now.Add(30 * time.Second)
	if triggers.GetQuietTime().IsValid() && triggers.GetQuietTime().AsDuration() > 0 {
		nextWake = input.Now.Add(triggers.GetQuietTime().AsDuration())
	}
	if triggers.GetDeadline().IsValid() && triggers.GetDeadline().AsTime().Before(nextWake) {
		nextWake = triggers.GetDeadline().AsTime()
	}
	decision.NextWakeAt = timestamppb.New(nextWake.UTC())
	switch decision.GetDisposition() {
	case domainpb.WatchDisposition_WATCH_DISPOSITION_QUIET:
		decision.RecommendedAction = domainpb.WatchActionKind_WATCH_ACTION_KIND_PARK
	case domainpb.WatchDisposition_WATCH_DISPOSITION_SIGNAL:
		decision.RecommendedAction = domainpb.WatchActionKind_WATCH_ACTION_KIND_ESCALATE
	case domainpb.WatchDisposition_WATCH_DISPOSITION_TERMINAL:
		decision.RecommendedAction = domainpb.WatchActionKind_WATCH_ACTION_KIND_WAKE_PARENT
	}
	return decision, nil
}

type Processor struct {
	service   *Service
	resolver  SubjectResolver
	evaluator Evaluator
	now       func() time.Time
}

func NewProcessor(service *Service, resolver SubjectResolver, evaluator Evaluator) *Processor {
	if evaluator == nil {
		evaluator = TriggerEvaluator{}
	}
	return &Processor{service: service, resolver: resolver, evaluator: evaluator, now: time.Now}
}

func (p *Processor) Process(ctx context.Context, watchID string) (*domainpb.CohortWatch, error) {
	watch, before, err := p.service.watches.Get(ctx, watchID)
	if err != nil {
		return nil, err
	}
	if watch.GetStatus() != domainpb.WatchStatus_WATCH_STATUS_ACTIVE {
		return watch, nil
	}
	_, currentFilterDigest, err := canonicalSpec(watch.GetSpec())
	if err != nil {
		return nil, err
	}
	if currentFilterDigest != before.FilterDigest {
		return nil, errors.New("watch cursor filter binding does not match persisted subjects")
	}
	retention, err := p.service.events.RetentionState(ctx)
	if err != nil {
		return nil, err
	}
	summaries := []SubjectSummary{}
	if p.resolver != nil {
		summaries, err = p.resolver.Resolve(ctx, watch.GetSpec().GetSubjects())
		if err != nil {
			return nil, err
		}
	}
	input := EvaluationInput{Watch: watch, Subjects: summaries, Now: p.now().UTC(), Reset: retention.Generation != before.RetentionGeneration, ResetFrom: before.RetentionGeneration, ResetTo: retention.Generation}
	after := before
	if input.Reset {
		after.RowID, after.RetentionGeneration = retention.HighRowID, retention.Generation
	} else {
		runIDs, err := subjectRunIDs(watch.GetSpec().GetSubjects())
		if err != nil {
			return nil, err
		}
		limit := int(watch.GetSpec().GetTriggers().GetEventCount())
		if limit <= 0 {
			limit = 64
		}
		input.Events, err = p.service.events.ReadCohort(ctx, runIDs, before.RowID, limit)
		if err != nil {
			return nil, err
		}
		if len(input.Events) > 0 {
			after.RowID = input.Events[len(input.Events)-1].Rowid
		}
	}
	for i := range input.Subjects {
		for _, event := range input.Events {
			if event.RunID.String() == input.Subjects[i].RunID && event.Timestamp.After(input.Subjects[i].FrictionThrough) {
				input.Subjects[i].FrictionUnavailable = true
			}
		}
	}
	after.Token = uuid.NewString()
	input.ProposedCursor = after.Token
	decision, err := p.evaluator.Evaluate(ctx, input)
	if err != nil {
		return nil, err
	}
	if decision == nil {
		return nil, errors.New("watch evaluator returned no decision")
	}
	// Dependency outages and explicit abstention must not consume evidence. The
	// same durable cursor will be replayed when evaluation becomes available.
	if decision.GetDisposition() == domainpb.WatchDisposition_WATCH_DISPOSITION_UNAVAILABLE {
		after = before
	}
	decision.IdempotencyKey = decisionKey(watchID, before.Token, after.RowID, decision.GetDisposition())
	updated, err := p.service.watches.CommitDecision(ctx, watchID, watch.GetRevision(), before, decision, after, input)
	if err == nil {
		p.service.notify(watchID)
		if p.service.actions != nil {
			_, _ = p.service.actions.RecoverPending(ctx)
			if action := updated.GetLastDecision().GetRecommendedAction(); action != domainpb.WatchActionKind_WATCH_ACTION_KIND_UNSPECIFIED && updated.GetSpec().GetParentRunId() != "" {
				_, _ = p.service.actions.Request(ctx, &domainpb.RequestCohortWatchActionRequest{
					WatchId: updated.GetWatchId(), ExpectedWatchRevision: updated.GetRevision(),
					IdempotencyKey: "watch-decision-action:" + updated.GetLastDecision().GetDecisionId(),
					Kind:           action, TargetRunId: updated.GetSpec().GetParentRunId(),
					RequestedBy: "agent-manager", Authority: domainpb.WatchAuthority_WATCH_AUTHORITY_SYSTEM,
					Rationale: updated.GetLastDecision().GetClassification(), Message: decisionEvidenceMessage(updated.GetLastDecision()),
				})
			}
		}
	}
	return updated, err
}

func decisionEvidenceMessage(decision *domainpb.WatchDecision) string {
	if decision == nil {
		return ""
	}
	evidence := append([]string(nil), decision.GetEvidenceIds()...)
	if len(evidence) > 20 {
		evidence = evidence[:20]
	}
	return fmt.Sprintf("classification=%s evidence=%s", decision.GetClassification(), strings.Join(evidence, ","))
}

func subjectRunIDs(subjects []*domainpb.WatchSubject) ([]uuid.UUID, error) {
	ids := make([]uuid.UUID, 0, len(subjects))
	for _, subject := range subjects {
		id, err := uuid.Parse(subject.GetRunId())
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func decisionKey(watchID, cursor string, rowID int64, disposition domainpb.WatchDisposition) string {
	parts := []string{watchID, cursor, fmt.Sprint(rowID), fmt.Sprint(int32(disposition))}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return "watch-decision:" + hex.EncodeToString(sum[:])
}
