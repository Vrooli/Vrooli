package scenarios

import (
	"context"
	"strings"

	"connectrpc.com/connect"

	scenariosv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/scenarios"
)

// service implements scenariosconnect.ScenariosServiceHandler.
type service struct {
	deps Deps
}

func (s *service) List(
	ctx context.Context,
	_ *connect.Request[scenariosv1.ListScenariosRequest],
) (*connect.Response[scenariosv1.ListScenariosResponse], error) {
	items, err := s.deps.Discovery.List(ctx)
	if err != nil {
		s.deps.Logger.WithError(err).Error("scenarios.List failed")
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*scenariosv1.Scenario, 0, len(items))
	for _, it := range items {
		out = append(out, &scenariosv1.Scenario{
			Name:        it.Name,
			Description: it.Description,
			Status:      it.Status,
		})
	}
	return connect.NewResponse(&scenariosv1.ListScenariosResponse{Scenarios: out}), nil
}

func (s *service) GetPort(
	ctx context.Context,
	req *connect.Request[scenariosv1.GetScenarioPortRequest],
) (*connect.Response[scenariosv1.GetScenarioPortResponse], error) {
	name := strings.TrimSpace(req.Msg.GetName())
	if name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errMissingName)
	}
	resolvedURL, info, err := s.deps.Discovery.ResolveURL(ctx, name)
	if err != nil {
		s.deps.Logger.WithError(err).WithField("scenario", name).Error("scenarios.GetPort resolve failed")
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	port := int32(0)
	if info != nil {
		port = int32(info.Port)
	}
	status, _ := s.deps.Discovery.Status(ctx, name)
	return connect.NewResponse(&scenariosv1.GetScenarioPortResponse{
		Port:   port,
		Status: status,
		Url:    resolvedURL,
	}), nil
}
