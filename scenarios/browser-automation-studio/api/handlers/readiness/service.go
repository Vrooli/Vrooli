package readiness

import (
	"context"
	"errors"
	"strings"

	"connectrpc.com/connect"

	readinessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/readiness"
)

var (
	errScenarioRequired  = errors.New("scenario is required")
	errProfileIncomplete = errors.New("readiness profile response omitted its JSON or version")
)

type service struct {
	deps Deps
}

func (s *service) GetReadinessProfile(ctx context.Context, req *connect.Request[readinessv1.GetReadinessProfileRequest]) (*connect.Response[readinessv1.GetReadinessProfileResponse], error) {
	scenario := strings.TrimSpace(req.Msg.GetScenario())
	if scenario == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errScenarioRequired)
	}
	profile, err := s.deps.Reader.GetReadinessProfile(ctx, scenario, strings.TrimSpace(req.Msg.GetPath()))
	if err != nil {
		s.deps.Logger.WithError(err).WithField("scenario", scenario).Warn("readiness profile lookup failed")
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	if strings.TrimSpace(profile.ProfileJSON) == "" || strings.TrimSpace(profile.ProfileVersion) == "" {
		return nil, connect.NewError(connect.CodeUnavailable, errProfileIncomplete)
	}
	if profile.Scenario == "" {
		profile.Scenario = scenario
	}
	return connect.NewResponse(&readinessv1.GetReadinessProfileResponse{
		Scenario:       profile.Scenario,
		ProfileJson:    profile.ProfileJSON,
		ProfileVersion: profile.ProfileVersion,
	}), nil
}
