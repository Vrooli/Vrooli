package validation

import (
	"context"

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
