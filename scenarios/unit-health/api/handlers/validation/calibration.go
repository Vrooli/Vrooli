package validation

import (
	"context"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/unit-health/v1/validation"
	"unit-health/internal/testquality/calibration"
)

// RunCalibration executes a governed, non-promoting calibration read over the
// authored corpus or an explicitly named reviewed holdout.
func (h *Handler) RunCalibration(ctx context.Context, req *connect.Request[validationv1.RunCalibrationRequest]) (*connect.Response[validationv1.RunCalibrationResponse], error) {
	if h == nil || h.calibrationRoot == "" {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("calibration corpus is not configured"))
	}
	partition := req.Msg.GetPartition()
	if partition == "" {
		partition = "development"
	}
	report, err := calibration.RunPartition(ctx, h.calibrationRoot, partition, req.Msg.GetHoldoutId(), req.Msg.GetRuleId(), req.Msg.GetIncludeNative())
	if err != nil {
		code := connect.CodeInvalidArgument
		if errors.Is(err, calibration.ErrHoldoutNotFound) {
			code = connect.CodeNotFound
		}
		return nil, connect.NewError(code, err)
	}
	return connect.NewResponse(&validationv1.RunCalibrationResponse{
		RunId:       fmt.Sprintf("calibration-%d", time.Now().UnixNano()),
		Partition:   report.Partition,
		Corpus:      calibrationCorpusToProto(report.Corpus),
		Cases:       calibrationCasesToProto(report.Cases),
		Holdout:     calibrationHoldoutToProto(report.Holdout),
		Limitations: append([]string(nil), report.Limitations...),
	}), nil
}

func calibrationCorpusToProto(in calibration.CorpusInventory) *validationv1.CorpusInventory {
	out := &validationv1.CorpusInventory{
		Specified:               uint32(maxZero(in.Specified)),
		Implemented:             uint32(maxZero(in.Implemented)),
		Retired:                 uint32(maxZero(in.Retired)),
		SpecCodesWithoutEmitter: uint32(maxZero(in.SpecCodesWithoutEmitter)),
		DevelopmentFloor:        in.DevelopmentFloor,
		Families:                make([]*validationv1.FamilyCount, 0, len(in.Families)),
	}
	for _, family := range in.Families {
		out.Families = append(out.Families, &validationv1.FamilyCount{
			Family: family.Family, Specified: uint32(maxZero(family.Specified)),
			Implemented: uint32(maxZero(family.Implemented)), Retired: uint32(maxZero(family.Retired)),
		})
	}
	return out
}

func calibrationCasesToProto(in []calibration.CaseOutcome) []*validationv1.CaseOutcome {
	out := make([]*validationv1.CaseOutcome, 0, len(in))
	for _, row := range in {
		out = append(out, &validationv1.CaseOutcome{Id: row.ID, RuleId: row.RuleID, Matched: row.Matched, Differences: append([]string(nil), row.Differences...), Status: row.Status})
	}
	return out
}

func calibrationHoldoutToProto(in []calibration.HoldoutOutcome) []*validationv1.HoldoutComparison {
	out := make([]*validationv1.HoldoutComparison, 0, len(in))
	for _, row := range in {
		out = append(out, &validationv1.HoldoutComparison{
			RuleId: row.RuleID, HoldoutId: row.HoldoutID, Labelled: uint32(maxZero(row.Labelled)),
			Observed: uint32(maxZero(row.Observed)), FalsePositives: uint32(maxZero(row.FalsePositives)),
			FalseNegatives: uint32(maxZero(row.FalseNegatives)), Unknown: uint32(maxZero(row.Unknown)),
			FpRate: row.FPRate, FnRate: row.FNRate, Budget: row.Budget,
			WithinBudget: row.WithinBudget, PromotionAllowed: row.PromotionAllowed,
		})
	}
	return out
}

func maxZero(value int) int {
	if value < 0 {
		return 0
	}
	return value
}
