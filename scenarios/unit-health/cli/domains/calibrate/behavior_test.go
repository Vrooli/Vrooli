package calibrate

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"
	cliapptest "github.com/vrooli/cli-core/cliapptest"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/unit-health/v1/validation"
)

type calibrationClient struct {
	response *validationv1.RunCalibrationResponse
}

func (calibrationClient) ValidateScenario(context.Context, *connect.Request[validationv1.ValidateScenarioRequest]) (*connect.Response[validationv1.ValidateScenarioResponse], error) {
	return nil, nil
}

func (c calibrationClient) RunCalibration(context.Context, *connect.Request[validationv1.RunCalibrationRequest]) (*connect.Response[validationv1.RunCalibrationResponse], error) {
	return connect.NewResponse(c.response), nil
}

func (calibrationClient) ReadTestBody(context.Context, *connect.Request[validationv1.ReadTestBodyRequest]) (*connect.Response[validationv1.ReadTestBodyResponse], error) {
	return nil, nil
}

func (calibrationClient) RunMutationPilot(context.Context, *connect.Request[validationv1.RunMutationPilotRequest]) (*connect.Response[validationv1.RunMutationPilotResponse], error) {
	return nil, nil
}

func TestCalibrationHandlerRendersCorpusCasesAndHoldout(t *testing.T) {
	h := newHandlers(&cliapp.ScenarioApp{})
	h.client = calibrationClient{response: &validationv1.RunCalibrationResponse{RunId: "run-1", Partition: "development", Corpus: &validationv1.CorpusInventory{Implemented: 2, Retired: 1, Specified: 3, DevelopmentFloor: "2/2@today", Families: []*validationv1.FamilyCount{{Family: "Go", Implemented: 2, Retired: 0, Specified: 2}}}, Cases: []*validationv1.CaseOutcome{{Id: "C1", RuleId: "rule", Matched: true}, {Id: "C2", RuleId: "rule", Status: "mismatch", Differences: []string{"missing"}}}, Holdout: []*validationv1.HoldoutComparison{{HoldoutId: "h1", RuleId: "rule", Labelled: 2, Observed: 2, FpRate: .1, FnRate: .2, PromotionAllowed: false}}, Limitations: []string{"owner decision required"}}}
	schema := cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "partition"}, {Name: "holdout"}, {Name: "rule"}, {Name: "include-native", Bool: true}}}
	ctx, out := cliapptest.NewCapturedRunContext(nil, schema, cliapptest.TestRunContextOptions{Flags: map[string]string{"partition": "development"}})
	if err := h.run(ctx); err != nil {
		t.Fatal(err)
	}
	if out.Len() == 0 {
		t.Fatal("calibration handler rendered no output")
	}
	if lines := corpusLines(nil); len(lines) != 1 {
		t.Fatalf("nil corpus lines = %v", lines)
	}
}

func TestCalibrationDefaultsPartitionAndReportsNilResponse(t *testing.T) {
	h := newHandlers(nil)
	h.client = calibrationClient{response: nil}
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Flags: map[string]string{}, Core: nil})
	if err := h.corpus(ctx); err == nil {
		t.Fatal("nil calibration response accepted")
	}
	if got := calibrationCaseLines(nil); len(got) != 0 {
		t.Fatalf("nil case lines = %v", got)
	}
}
