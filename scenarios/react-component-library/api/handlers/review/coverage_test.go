package review

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	databasetest "github.com/vrooli/api-core/databasetest"
	reviewv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/review"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/shared"
	"react-component-library/internal/designcritique"
	sketchdomain "react-component-library/internal/sketch"
)

type acceptanceEvaluatorStub struct {
	validateErr error
	evaluateErr error
}

type critiqueRepositoryStub struct {
	store   *databasetest.SliceRepo[designcritique.Record]
	listErr error
}

type acceptanceStoreStub struct{}

func (acceptanceStoreStub) RecordAcceptance(context.Context, string, string, sketchdomain.AcceptanceIntent, func(context.Context) (sketchdomain.AcceptanceFacts, error)) (sketchdomain.AcceptanceDecision, error) {
	return sketchdomain.AcceptanceDecision{}, errors.New("unexpected record")
}

func (acceptanceStoreStub) ReadAcceptance(string, string, string) (sketchdomain.AcceptanceDecision, error) {
	return sketchdomain.AcceptanceDecision{}, errors.New("not found")
}

func (s critiqueRepositoryStub) Create(context.Context, string, designcritique.Review, designcritique.Assessment) (designcritique.Record, error) {
	return designcritique.Record{}, errors.New("unexpected create")
}

func (s critiqueRepositoryStub) List(context.Context, designcritique.Target, string) ([]designcritique.Record, string, error) {
	return nil, "", s.listErr
}

func (s critiqueRepositoryStub) Get(ctx context.Context, id string) (designcritique.Record, error) {
	return s.store.Get(ctx, id)
}

func newCritiqueRepositoryStub() critiqueRepositoryStub {
	return critiqueRepositoryStub{store: databasetest.NewSliceRepo(
		func(record designcritique.Record) string { return record.ID },
		func(record *designcritique.Record, id string) { record.ID = id },
		func(id string) error { return sql.ErrNoRows },
	)}
}

func (s acceptanceEvaluatorStub) ValidateAcceptanceCandidate(context.Context, *reviewv1.CheckCandidateAcceptanceRequest) error {
	return s.validateErr
}

func (s acceptanceEvaluatorStub) EvaluateAcceptance(context.Context, *reviewv1.CheckCandidateAcceptanceRequest) (*reviewv1.CandidateAcceptanceCheck, error) {
	return nil, s.evaluateErr
}

func validAcceptanceCheck() *reviewv1.CheckCandidateAcceptanceRequest {
	return &reviewv1.CheckCandidateAcceptanceRequest{
		Render:             &sharedv1.RenderCandidateRequest{Candidate: &sharedv1.CandidateReference{Scenario: "demo", DesignId: "home", Hash: "candidate"}},
		ExpectedRenderHash: "render",
	}
}

func TestCheckCandidateAcceptanceRejectsMissingInputsAndUnavailableEvaluator(t *testing.T) {
	h := NewHandlerWithPorts(nil, Ports{})
	for _, request := range []*connect.Request[reviewv1.CheckCandidateAcceptanceRequest]{nil, connect.NewRequest(&reviewv1.CheckCandidateAcceptanceRequest{})} {
		_, err := h.CheckCandidateAcceptance(context.Background(), request)
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	}
	_, err := h.CheckCandidateAcceptance(context.Background(), connect.NewRequest(validAcceptanceCheck()))
	require.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
}

func TestCheckCandidateAcceptancePropagatesEvaluatorFailure(t *testing.T) {
	want := connect.NewError(connect.CodeFailedPrecondition, errors.New("candidate changed"))
	h := NewHandlerWithPorts(nil, Ports{AcceptanceEvaluator: acceptanceEvaluatorStub{evaluateErr: want}})
	_, err := h.CheckCandidateAcceptance(context.Background(), connect.NewRequest(validAcceptanceCheck()))
	require.ErrorIs(t, err, want)
}

func TestAcceptCandidateMapsValidationAndStoreFailures(t *testing.T) {
	check := validAcceptanceCheck()
	request := connect.NewRequest(&reviewv1.AcceptCandidateRequest{Check: check, IdempotencyKey: "decision"})
	want := connect.NewError(connect.CodeFailedPrecondition, errors.New("candidate is stale"))
	h := NewHandlerWithPorts(nil, Ports{
		AcceptanceEvaluator: acceptanceEvaluatorStub{validateErr: want},
		AcceptanceStoreFor:  func(context.Context) (AcceptanceStore, error) { return acceptanceStoreStub{}, nil },
	})
	_, err := h.AcceptCandidate(context.Background(), request)
	require.ErrorIs(t, err, want)

	h = NewHandlerWithPorts(nil, Ports{
		AcceptanceEvaluator: acceptanceEvaluatorStub{},
		AcceptanceStoreFor:  func(context.Context) (AcceptanceStore, error) { return nil, errors.New("store unavailable") },
	})
	_, err = h.AcceptCandidate(context.Background(), request)
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
	for _, test := range []struct {
		name string
		err  error
		code connect.Code
	}{
		{name: "conflict", err: sketchdomain.ErrAcceptanceConflict, code: connect.CodeAborted},
		{name: "missing", err: os.ErrNotExist, code: connect.CodeNotFound},
		{name: "storage", err: sql.ErrNoRows, code: connect.CodeInternal},
	} {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.code, connect.CodeOf(acceptanceError(test.err)))
		})
	}
}

func TestCritiqueReadRPCsMapMissingDependenciesAndRepositoryFailures(t *testing.T) {
	h := NewHandlerWithPorts(nil, Ports{})
	_, err := h.GetCritique(context.Background(), connect.NewRequest(&reviewv1.GetCritiqueRequest{}))
	require.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
	h.ports.CritiquesFor = func(context.Context) (designcritique.Repository, error) { return newCritiqueRepositoryStub(), nil }
	_, err = h.ListCritiques(context.Background(), connect.NewRequest(&reviewv1.ListCritiquesRequest{}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))

	h.ports.CritiquesFor = func(context.Context) (designcritique.Repository, error) {
		repo := newCritiqueRepositoryStub()
		repo.store.GetErr = errors.New("database offline")
		return repo, nil
	}
	_, err = h.GetCritique(context.Background(), connect.NewRequest(&reviewv1.GetCritiqueRequest{Id: "id"}))
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
	h.ports.CritiquesFor = func(context.Context) (designcritique.Repository, error) {
		repo := newCritiqueRepositoryStub()
		repo.listErr = errors.New("database offline")
		return repo, nil
	}
	_, err = h.ListCritiques(context.Background(), connect.NewRequest(&reviewv1.ListCritiquesRequest{Target: &reviewv1.CritiqueTarget{Scenario: "demo"}}))
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
}
