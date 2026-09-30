package validation

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"experience-manager/internal/spec"

	"connectrpc.com/connect"

	contractv1 "github.com/vrooli/vrooli/packages/proto/gen/go/experience-manager/v1/contract"
)

// contractService is the native ContractService mount. It wraps the same core
// pipeline as the shared ScenarioValidationService mount.
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

func (s *contractService) ListFleet(ctx context.Context, req *connect.Request[contractv1.ListFleetRequest]) (*connect.Response[contractv1.ListFleetResponse], error) {
	resp, err := s.core.listFleet(ctx, req.Msg.GetRepoRoot())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (s *contractService) AppendAttestation(ctx context.Context, req *connect.Request[contractv1.AppendAttestationRequest]) (*connect.Response[contractv1.AppendAttestationResponse], error) {
	resp, err := s.core.appendAttestation(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (s *contractService) ScaffoldCases(_ context.Context, req *connect.Request[contractv1.ScaffoldCasesRequest]) (*connect.Response[contractv1.ScaffoldCasesResponse], error) {
	resp, err := s.core.scaffoldCases(req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(resp), nil
}

func (s *contractService) GetReadinessProfile(_ context.Context, req *connect.Request[contractv1.GetReadinessProfileRequest]) (*connect.Response[contractv1.GetReadinessProfileResponse], error) {
	path := req.Msg.GetPath()
	if path == "" {
		path = filepath.Join(s.core.deps.RepoRoot, "scenarios", req.Msg.GetScenario())
	}
	profile, err := spec.ReadinessProfileForScenario(path)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("compile readiness profile: %w", err))
	}
	encoded, err := json.Marshal(profile)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("encode readiness profile: %w", err))
	}
	return connect.NewResponse(&contractv1.GetReadinessProfileResponse{Scenario: profile.Scenario, ProfileJson: string(encoded), ProfileVersion: profile.Version}), nil
}
