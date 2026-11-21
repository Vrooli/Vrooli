package triage

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/schedule"
)

type Service struct {
	repo  Repository
	clock schedule.Clock
}

func NewService(repo Repository, clk schedule.Clock) *Service { return &Service{repo, clk} }
func (s *Service) Get(ctx context.Context, id string) (Disposition, []Annotation, error) {
	d, ok, e := s.repo.GetDisposition(ctx, id)
	if e != nil {
		return Disposition{}, nil, e
	}
	if !ok {
		d = Disposition{SignalID: id, State: New, UpdatedAt: s.clock.Now().UTC()}
	}
	a, e := s.repo.ListAnnotations(ctx, id)
	return d, a, e
}

func (s *Service) Set(ctx context.Context, id string, to State, revisit *time.Time) (Disposition, error) {
	current, ok, e := s.repo.GetDisposition(ctx, id)
	if e != nil {
		return Disposition{}, e
	}
	from := New
	if ok {
		from = current.State
	}
	if !allowed(from, to) {
		return Disposition{}, ErrInvalidTransition{from, to}
	}
	return s.repo.UpsertDisposition(ctx, Disposition{SignalID: id, State: to, RevisitAt: revisit, UpdatedAt: s.clock.Now().UTC()})
}

func (s *Service) Annotate(ctx context.Context, id string, author Author, body string, outcome *Outcome) (Annotation, error) {
	if author != Operator && author != Agent && author != System {
		return Annotation{}, ErrInvalidTriage{"invalid annotation author"}
	}
	if strings.TrimSpace(body) == "" && outcome == nil {
		return Annotation{}, ErrInvalidTriage{"annotation body or outcome is required"}
	}
	if outcome != nil && (outcome.TargetID == "" || !validOutcome(outcome.Kind)) {
		return Annotation{}, ErrInvalidTriage{"invalid outcome link"}
	}
	return s.repo.AppendAnnotation(ctx, Annotation{ID: uuid.NewString(), SignalID: id, Author: author, Body: strings.TrimSpace(body), Outcome: outcome, CreatedAt: s.clock.Now().UTC()})
}

func allowed(from, to State) bool {
	if from == to {
		return true
	}
	switch from {
	case New:
		return to == Triaged || to == Dropped
	case Triaged:
		return to == Routed || to == Done || to == Dropped
	case Routed:
		return to == Done || to == Triaged
	case Done, Dropped:
		return to == Triaged
	}
	return false
}

func validOutcome(k OutcomeKind) bool {
	return k == OutcomeScenario || k == OutcomeBacklog || k == OutcomeIdeaPipeline || k == OutcomeKnowledgeTopic
}
