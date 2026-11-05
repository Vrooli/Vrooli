package validation

import (
	"context"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	"business-health/internal/evidence"
	"business-health/internal/matrix"

	contractv1 "github.com/vrooli/vrooli/packages/proto/gen/go/business-health/v1/contract"
)

// contractService is the native ContractService mount. It wraps the same
// core pipeline as the shared mount (both RPC surfaces answer from one
// validate() implementation) but needs its own type because both services
// declare a ValidateScenario RPC with different message types.
type contractService struct {
	core *connectHandler
}

func newContractService(core *connectHandler) *contractService {
	return &contractService{core: core}
}

func (s *contractService) ValidateScenario(ctx context.Context, req *connect.Request[contractv1.ValidateScenarioRequest]) (*connect.Response[contractv1.ValidateScenarioResponse], error) {
	resp, err := s.core.validateNative(ctx, req.Msg.GetScenario(), req.Msg.GetPath())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

// join loads the contract + evidence for one target and runs the matrix
// join (the single implementation every surface consumes).
func (s *contractService) join(scenario, path string) (matrix.Result, error) {
	dir, err := s.core.resolveTarget(scenario, path)
	if err != nil {
		return matrix.Result{}, connect.NewError(connect.CodeInvalidArgument, err)
	}
	contract, err := s.core.deps.Extractor.Load(scenario, dir)
	if err != nil {
		return matrix.Result{}, connect.NewError(connect.CodeInvalidArgument, err)
	}
	store, err := evidence.NewTargetStore(dir, nil)
	if err != nil {
		return matrix.Result{}, connect.NewError(connect.CodeInternal, err)
	}
	snap, hasSnap, err := store.ReadSnapshot()
	if err != nil {
		return matrix.Result{}, connect.NewError(connect.CodeInternal, err)
	}
	attestations, err := store.LatestAttestations()
	if err != nil {
		return matrix.Result{}, connect.NewError(connect.CodeInternal, err)
	}
	return matrix.Join(matrix.Inputs{
		Contract:     contract,
		Snapshot:     snap,
		HasSnapshot:  hasSnap,
		Staleness:    store.SnapshotStaleness(snap, hasSnap),
		Attestations: attestations,
		Now:          store.Now(),
	}), nil
}

func (s *contractService) GetMatrix(ctx context.Context, req *connect.Request[contractv1.GetMatrixRequest]) (*connect.Response[contractv1.GetMatrixResponse], error) {
	result, err := s.join(req.Msg.GetScenario(), req.Msg.GetPath())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&contractv1.GetMatrixResponse{
		Scenario:       result.Scenario,
		Matrix:         matrixRowsToProto(result.Rows),
		Registry:       registrySummaryToProto(result.Registry),
		DegradedReason: result.DegradedReason,
	}), nil
}

func (s *contractService) GetDrift(ctx context.Context, req *connect.Request[contractv1.GetDriftRequest]) (*connect.Response[contractv1.GetDriftResponse], error) {
	result, err := s.join(req.Msg.GetScenario(), req.Msg.GetPath())
	if err != nil {
		return nil, err
	}
	out := make([]*contractv1.DriftEntry, 0, len(result.Drift))
	for _, d := range result.Drift {
		out = append(out, &contractv1.DriftEntry{Kind: d.Kind, SubjectId: d.SubjectID, Detail: d.Detail})
	}
	return connect.NewResponse(&contractv1.GetDriftResponse{
		Scenario: result.Scenario,
		Drift:    out,
	}), nil
}

func (s *contractService) LogManualValidation(ctx context.Context, req *connect.Request[contractv1.LogManualValidationRequest]) (*connect.Response[contractv1.LogManualValidationResponse], error) {
	scenario := req.Msg.GetScenario()
	dir, err := s.core.resolveTarget(scenario, "")
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	store := evidence.NewStore(dir, nil)
	a, err := store.AppendAttestation(scenario, req.Msg.GetRequirementId(), req.Msg.GetAttestedBy(), req.Msg.GetNotes())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&contractv1.LogManualValidationResponse{
		Attestation: attestationToProto(a, false),
		LedgerPath:  store.LedgerPath(),
	}), nil
}

func matrixRowsToProto(rows []matrix.Row) []*contractv1.MatrixRow {
	out := make([]*contractv1.MatrixRow, 0, len(rows))
	for _, r := range rows {
		row := &contractv1.MatrixRow{
			OtId:              r.OTID,
			OtTitle:           r.OTTitle,
			OtChecked:         r.OTChecked,
			OtPriority:        r.OTPriority,
			RequirementId:     r.RequirementID,
			RequirementTitle:  r.RequirementTitle,
			RequirementStatus: r.RequirementStatus,
			Criticality:       r.Criticality,
			Unproven:          r.Unproven,
			UnprovenReason:    r.UnprovenReason,
		}
		for _, v := range r.Validations {
			row.Validations = append(row.Validations, &contractv1.ValidationCell{
				Type:      v.Type,
				Phase:     v.Phase,
				Status:    v.Status,
				Ref:       v.Ref,
				RefExists: v.RefExists,
			})
		}
		cell := &contractv1.EvidenceCell{
			LiveStatus: r.Evidence.OTStatus,
			Stale:      r.Evidence.SnapshotStale,
		}
		if !r.Evidence.SnapshotAt.IsZero() {
			cell.LastSyncedAt = timestamppb.New(r.Evidence.SnapshotAt)
		}
		if r.Evidence.Manual != nil {
			cell.Manual = attestationToProto(*r.Evidence.Manual, r.Evidence.ManualExpired)
		}
		row.Evidence = cell
		out = append(out, row)
	}
	return out
}

func attestationToProto(a evidence.Attestation, expired bool) *contractv1.ManualAttestation {
	return &contractv1.ManualAttestation{
		RequirementId: a.RequirementID,
		AttestedBy:    a.AttestedBy,
		AttestedAt:    timestamppb.New(a.AttestedAt),
		ExpiresAt:     timestamppb.New(a.ExpiresAt),
		Expired:       expired,
		Notes:         a.Notes,
	}
}

func registrySummaryToProto(s matrix.RegistrySummary) *contractv1.RegistrySummary {
	counts := make(map[string]int32, len(s.StatusCounts))
	for k, v := range s.StatusCounts {
		counts[k] = int32(v)
	}
	return &contractv1.RegistrySummary{
		ModuleCount:            int32(s.ModuleCount),
		RequirementCount:       int32(s.RequirementCount),
		OperationalTargetCount: int32(s.OperationalTargetCount),
		StatusCounts:           counts,
		StarterTemplate:        s.StarterTemplate,
	}
}
