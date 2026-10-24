package convergence

import (
	"context"
	"testing"
	"time"

	internalconv "meta-optimization-manager/internal/convergence"

	"connectrpc.com/connect"

	convergencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/meta-optimization-manager/v1/convergence"
)

type fakeService struct {
	status internalconv.Status
	fit    []internalconv.TemplateFitness
	refs   []internalconv.ReferenceHealth
	trend  []internalconv.FitnessTrendPoint
	lastEl internalconv.ReferenceEligibility
}

func (f *fakeService) GetConvergenceStatus(context.Context) (internalconv.Status, error) {
	return f.status, nil
}

func (f *fakeService) GetTemplateFitness(_ context.Context, _ string) ([]internalconv.TemplateFitness, error) {
	return f.fit, nil
}

func (f *fakeService) ListReferences(_ context.Context, e internalconv.ReferenceEligibility) ([]internalconv.ReferenceHealth, error) {
	f.lastEl = e
	return f.refs, nil
}

func (f *fakeService) GetConvergenceTrend(_ context.Context, _ string) ([]internalconv.FitnessTrendPoint, error) {
	return f.trend, nil
}

func TestHandlerStatus(t *testing.T) {
	svc := &fakeService{status: internalconv.Status{
		Templates:  []internalconv.TemplateFitness{{Template: "react-vite", PerReplicaCost: 900, Tier: internalconv.TierStrong}},
		References: []internalconv.ReferenceHealth{{Scenario: "reference-react-vite", StabilityDays: 61, Eligibility: internalconv.EligibilityCandidate, LastTemplateSync: time.Now()}},
	}}
	h := NewConnectHandler(Deps{Service: svc})
	resp, err := h.GetConvergenceStatus(context.Background(), connect.NewRequest(&convergencev1.GetConvergenceStatusRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Msg.GetTemplates()) != 1 || resp.Msg.GetTemplates()[0].GetTier() != convergencev1.FitnessTier_FITNESS_TIER_STRONG {
		t.Fatalf("templates = %+v", resp.Msg.GetTemplates())
	}
	if len(resp.Msg.GetReferences()) != 1 || resp.Msg.GetReferences()[0].GetEligibility() != convergencev1.ReferenceEligibility_REFERENCE_ELIGIBILITY_CANDIDATE {
		t.Fatalf("references = %+v", resp.Msg.GetReferences())
	}
}

func TestHandlerListReferencesThreadsFilter(t *testing.T) {
	svc := &fakeService{}
	h := NewConnectHandler(Deps{Service: svc})
	_, err := h.ListReferences(context.Background(), connect.NewRequest(&convergencev1.ListReferencesRequest{
		Eligibility: convergencev1.ReferenceEligibility_REFERENCE_ELIGIBILITY_ELIGIBLE,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if svc.lastEl != internalconv.EligibilityEligible {
		t.Fatalf("eligibility not threaded: %v", svc.lastEl)
	}
}

func TestHandlerTrend(t *testing.T) {
	svc := &fakeService{trend: []internalconv.FitnessTrendPoint{{Template: "react-vite", PerReplicaCost: 900, CoordinatedEditCount: 5, At: time.Now()}}}
	h := NewConnectHandler(Deps{Service: svc})
	resp, err := h.GetConvergenceTrend(context.Background(), connect.NewRequest(&convergencev1.GetConvergenceTrendRequest{Template: "react-vite"}))
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Msg.GetPoints()) != 1 || resp.Msg.GetPoints()[0].GetPerReplicaCost() != 900 {
		t.Fatalf("points = %+v", resp.Msg.GetPoints())
	}
}
