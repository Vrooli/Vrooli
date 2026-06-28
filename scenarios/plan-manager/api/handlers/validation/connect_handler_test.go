package validation

import (
	"context"
	"errors"
	"io"
	"log"
	"testing"

	internalplans "plan-manager/internal/plans"
	internalvalidation "plan-manager/internal/validation"

	"connectrpc.com/connect"

	"github.com/stretchr/testify/require"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/shared"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/plan-manager/v1/validation"
)

// fakeValidationService is a minimal stand-in for internalvalidation.Service.
type fakeValidationService struct {
	report internalvalidation.ReferenceReport
	scope  internalvalidation.BaselineScope
	result internalvalidation.Result
	dodMet bool
	err    error

	gotPlanID  string
	gotPhaseID string
}

func (f *fakeValidationService) ResolveReferences(_ context.Context, planID, phaseID string) (internalvalidation.ReferenceReport, error) {
	f.gotPlanID, f.gotPhaseID = planID, phaseID
	return f.report, f.err
}

func (f *fakeValidationService) ComputeStaleness(_ context.Context, planID, phaseID string) (internalvalidation.ReferenceReport, error) {
	f.gotPlanID, f.gotPhaseID = planID, phaseID
	return f.report, f.err
}

func (f *fakeValidationService) DeriveBaselineScope(_ context.Context, planID, phaseID string) (internalvalidation.BaselineScope, error) {
	f.gotPlanID, f.gotPhaseID = planID, phaseID
	return f.scope, f.err
}

func (f *fakeValidationService) RunValidation(_ context.Context, planID, phaseID string) (internalvalidation.Result, error) {
	f.gotPlanID, f.gotPhaseID = planID, phaseID
	return f.result, f.err
}

func (f *fakeValidationService) CaptureBaseline(_ context.Context, planID string) (internalvalidation.BaselineCapture, error) {
	f.gotPlanID = planID
	return internalvalidation.BaselineCapture{}, f.err
}

func (f *fakeValidationService) LastValidation(_ context.Context, planID, phaseID string) (internalvalidation.Result, bool, error) {
	f.gotPlanID, f.gotPhaseID = planID, phaseID
	return f.result, false, f.err
}

func (f *fakeValidationService) VerifyDefinitionOfDone(_ context.Context, planID string) (internalvalidation.Result, bool, error) {
	f.gotPlanID = planID
	return f.result, f.dodMet, f.err
}

var _ internalvalidation.Service = (*fakeValidationService)(nil)

func newValidationHandler(svc internalvalidation.Service) *connectHandler {
	return NewConnectHandler(Deps{Service: svc, Logger: log.New(io.Discard, "", 0)})
}

func TestResolveReferencesSuccess(t *testing.T) {
	svc := &fakeValidationService{report: internalvalidation.ReferenceReport{
		References: []internalplans.Reference{
			{ID: "r1", Kind: internalplans.ReferenceCode, Resolution: internalplans.ResolutionResolved},
		},
		Degraded: true,
	}}
	h := newValidationHandler(svc)

	resp, err := h.ResolveReferences(context.Background(), connect.NewRequest(&validationv1.ResolveReferencesRequest{PlanId: "p1", PhaseId: "ph-1"}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.GetReferences(), 1)
	require.Equal(t, sharedv1.ReferenceResolution_REFERENCE_RESOLUTION_RESOLVED, resp.Msg.GetReferences()[0].GetResolution())
	require.True(t, resp.Msg.GetDegraded())
	require.Equal(t, "p1", svc.gotPlanID)
	require.Equal(t, "ph-1", svc.gotPhaseID)
}

func TestComputeStalenessSuccess(t *testing.T) {
	svc := &fakeValidationService{report: internalvalidation.ReferenceReport{
		Overall:  internalplans.StalenessLightlyStale,
		Degraded: false,
		References: []internalplans.Reference{
			{ID: "r1", Staleness: internalplans.StalenessLightlyStale},
		},
	}}
	h := newValidationHandler(svc)

	resp, err := h.ComputeStaleness(context.Background(), connect.NewRequest(&validationv1.ComputeStalenessRequest{PlanId: "p1"}))
	require.NoError(t, err)
	require.Equal(t, sharedv1.StalenessTier_STALENESS_TIER_LIGHTLY_STALE, resp.Msg.GetOverall())
	require.Len(t, resp.Msg.GetReferences(), 1)
	require.False(t, resp.Msg.GetDegraded())
}

func TestDeriveBaselineScopeSuccess(t *testing.T) {
	svc := &fakeValidationService{scope: internalvalidation.BaselineScope{
		Commands:  []string{"git-control-tower baseline diff --scenario foo --name impl"},
		Locations: []string{"scenarios/foo"},
	}}
	h := newValidationHandler(svc)

	resp, err := h.DeriveBaselineScope(context.Background(), connect.NewRequest(&validationv1.DeriveBaselineScopeRequest{PlanId: "p1", PhaseId: "ph-1"}))
	require.NoError(t, err)
	require.Equal(t, []string{"git-control-tower baseline diff --scenario foo --name impl"}, resp.Msg.GetCommands())
	require.Equal(t, []string{"scenarios/foo"}, resp.Msg.GetLocations())
}

func TestRunValidationSuccess(t *testing.T) {
	svc := &fakeValidationService{result: internalvalidation.Result{
		ID: "v1", PlanID: "p1", Verdict: internalvalidation.VerdictPass, Staleness: internalplans.StalenessFresh,
	}}
	h := newValidationHandler(svc)

	resp, err := h.RunValidation(context.Background(), connect.NewRequest(&validationv1.RunValidationRequest{PlanId: "p1", PhaseId: "ph-1"}))
	require.NoError(t, err)
	require.Equal(t, "v1", resp.Msg.GetResult().GetId())
	require.Equal(t, sharedv1.ValidationVerdict_VALIDATION_VERDICT_PASS, resp.Msg.GetResult().GetVerdict())
}

func TestVerifyDefinitionOfDoneSuccess(t *testing.T) {
	svc := &fakeValidationService{
		result: internalvalidation.Result{ID: "v1", Verdict: internalvalidation.VerdictPass},
		dodMet: true,
	}
	h := newValidationHandler(svc)

	resp, err := h.VerifyDefinitionOfDone(context.Background(), connect.NewRequest(&validationv1.VerifyDefinitionOfDoneRequest{PlanId: "p1"}))
	require.NoError(t, err)
	require.True(t, resp.Msg.GetDodMet())
	require.Equal(t, sharedv1.ValidationVerdict_VALIDATION_VERDICT_PASS, resp.Msg.GetResult().GetVerdict())
	require.Equal(t, "p1", svc.gotPlanID)
}

// TestValidationErrorMapping asserts each validation/plans sentinel maps to the
// documented Connect code (see internal/validation/errors.go ToConnectError).
func TestValidationErrorMapping(t *testing.T) {
	t.Run("validation_phase_not_found_is_not_found", func(t *testing.T) {
		h := newValidationHandler(&fakeValidationService{err: internalvalidation.ErrPhaseNotFound{PlanID: "p", PhaseID: "ph"}})
		_, err := h.ResolveReferences(context.Background(), connect.NewRequest(&validationv1.ResolveReferencesRequest{PlanId: "p", PhaseId: "ph"}))
		require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	})
	t.Run("plan_not_found_is_not_found", func(t *testing.T) {
		h := newValidationHandler(&fakeValidationService{err: internalplans.ErrPlanNotFound{ID: "p"}})
		_, err := h.RunValidation(context.Background(), connect.NewRequest(&validationv1.RunValidationRequest{PlanId: "p"}))
		require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	})
	t.Run("invalid_plan_is_invalid_argument", func(t *testing.T) {
		h := newValidationHandler(&fakeValidationService{err: internalplans.ErrInvalidPlan{Reason: "bad"}})
		_, err := h.VerifyDefinitionOfDone(context.Background(), connect.NewRequest(&validationv1.VerifyDefinitionOfDoneRequest{PlanId: "p"}))
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	})
	t.Run("unknown_error_is_internal", func(t *testing.T) {
		h := newValidationHandler(&fakeValidationService{err: errors.New("boom")})
		_, err := h.ComputeStaleness(context.Background(), connect.NewRequest(&validationv1.ComputeStalenessRequest{PlanId: "p"}))
		require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
	})
}
