package commitments

import (
	"context"
	"strings"
)

type Service interface {
	List(context.Context) ([]Commitment, error)
	Create(context.Context, CreateInput) (Commitment, error)
	UpdateState(context.Context, string, string, int64) (Commitment, error)
	Revise(context.Context, ReviseInput) (Commitment, Revision, error)
}

type service struct{ repo Repository }

func NewService(repo Repository) Service                        { return &service{repo: repo} }
func (s *service) List(c context.Context) ([]Commitment, error) { return s.repo.List(c) }

func (s *service) Create(c context.Context, in CreateInput) (Commitment, error) {
	if strings.TrimSpace(in.Result) == "" {
		return Commitment{}, ErrInvalidCommitment{"result", "required"}
	}
	if strings.TrimSpace(in.PromisedBoundary) == "" {
		return Commitment{}, ErrInvalidCommitment{"promised_boundary", "required"}
	}
	state := strings.TrimSpace(in.State)
	if state == "" {
		state = StateProposed
	}
	if !validState(state) {
		return Commitment{}, ErrInvalidCommitment{"state", "must be proposed, active, fulfilled, or cancelled"}
	}
	timezone := strings.TrimSpace(in.Timezone)
	if timezone == "" {
		timezone = "UTC"
	}
	return s.repo.Create(c, Commitment{Result: strings.TrimSpace(in.Result), DefinitionOfDone: strings.TrimSpace(in.DefinitionOfDone), PromisedBoundary: strings.TrimSpace(in.PromisedBoundary), Timezone: timezone, Beneficiary: strings.TrimSpace(in.Beneficiary), Assumptions: strings.TrimSpace(in.Assumptions), ScopeExclusions: strings.TrimSpace(in.ScopeExclusions), State: state, Risk: RiskUnknown, AcknowledgmentStatus: AcknowledgmentUnknown, Revision: 1})
}

func (s *service) UpdateState(c context.Context, id, state string, revision int64) (Commitment, error) {
	if strings.TrimSpace(id) == "" {
		return Commitment{}, ErrInvalidCommitment{"id", "required"}
	}
	if !validState(strings.TrimSpace(state)) {
		return Commitment{}, ErrInvalidCommitment{"state", "must be proposed, active, fulfilled, or cancelled"}
	}
	return s.repo.UpdateState(c, id, strings.TrimSpace(state), revision)
}

func (s *service) Revise(c context.Context, in ReviseInput) (Commitment, Revision, error) {
	if strings.TrimSpace(in.ID) == "" {
		return Commitment{}, Revision{}, ErrInvalidCommitment{"id", "required"}
	}
	if strings.TrimSpace(in.PromisedBoundary) == "" {
		return Commitment{}, Revision{}, ErrInvalidCommitment{"promised_boundary", "required"}
	}
	ack := strings.TrimSpace(in.AcknowledgmentStatus)
	if ack == "" {
		ack = AcknowledgmentUnknown
	}
	return s.repo.Revise(c, ReviseInput{ID: strings.TrimSpace(in.ID), PromisedBoundary: strings.TrimSpace(in.PromisedBoundary), Assumptions: strings.TrimSpace(in.Assumptions), ScopeExclusions: strings.TrimSpace(in.ScopeExclusions), Reason: strings.TrimSpace(in.Reason), AcknowledgmentStatus: ack, ExpectedRevision: in.ExpectedRevision})
}

func validState(state string) bool {
	return state == StateProposed || state == StateActive || state == StateFulfilled || state == StateCancelled
}
