package main

import (
	"context"
	"scenario-to-desktop-api/build"
	"scenario-to-desktop-api/deploy"
	"scenario-to-desktop-api/evidence"
	"scenario-to-desktop-api/generation"
	"scenario-to-desktop-api/pipeline"
	"scenario-to-desktop-api/records"
	"scenario-to-desktop-api/shared/connecterrors"
	"scenario-to-desktop-api/signing"
	"scenario-to-desktop-api/smoketest"
	"scenario-to-desktop-api/state"
	"scenario-to-desktop-api/system"
	"scenario-to-desktop-api/tasks"
	"scenario-to-desktop-api/telemetry"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain/domainconnect"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/pipeline/pipelineconnect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/shared"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/shared/sharedconnect"
	"google.golang.org/protobuf/types/known/timestamppb"

	preflightdomain "scenario-to-desktop-api/preflight"
)

type healthConnectService struct{}

func (healthConnectService) Check(_ context.Context, _ *connect.Request[sharedv1.HealthCheckRequest]) (*connect.Response[sharedv1.HealthCheckResponse], error) {
	return connect.NewResponse(&sharedv1.HealthCheckResponse{
		Status:    "healthy",
		CheckedAt: timestamppb.New(time.Now()),
		Service:   "scenario-to-desktop-api",
	}), nil
}

func (s *Server) registerConnectHandlers() {
	options := []connect.HandlerOption{connect.WithInterceptors(connecterrors.Interceptor())}
	path, handler := sharedconnect.NewHealthServiceHandler(healthConnectService{}, options...)
	s.router.PathPrefix(path).Handler(handler)

	path, handler = domainconnect.NewDocumentationServiceHandler(documentationConnectService{server: s}, options...)
	s.router.PathPrefix(path).Handler(handler)

	path, handler = domainconnect.NewOperationsServiceHandler(operationsConnectService{server: s, scenarioHandler: s.scenarioHandler}, options...)
	s.router.PathPrefix(path).Handler(handler)

	path, handler = pipelineconnect.NewPipelineServiceHandler(pipeline.NewConnectService(s.pipelineHandler), options...)
	s.router.PathPrefix(path).Handler(handler)

	path, handler = domainconnect.NewSystemServiceHandler(system.NewConnectService(s.systemHandler), options...)
	s.router.PathPrefix(path).Handler(handler)

	path, handler = domainconnect.NewConfigServiceHandler(generation.NewConnectService(s.configAnalyzer), options...)
	s.router.PathPrefix(path).Handler(handler)

	path, handler = domainconnect.NewSigningServiceHandler(signing.NewConnectService(s.signingHandler), options...)
	s.router.PathPrefix(path).Handler(handler)

	path, handler = domainconnect.NewBuildServiceHandler(build.NewConnectService(s.buildHandler), options...)
	s.router.PathPrefix(path).Handler(handler)

	path, handler = domainconnect.NewSmokeTestServiceHandler(smoketest.NewConnectService(s.smokeTestService, s.smokeTestStore, s.smokeTestCancels), options...)
	s.router.PathPrefix(path).Handler(handler)

	path, handler = domainconnect.NewPreflightServiceHandler(preflightdomain.NewConnectService(s.preflightService), options...)
	s.router.PathPrefix(path).Handler(handler)

	if s.taskSvc != nil {
		path, handler = domainconnect.NewTaskServiceHandler(tasks.NewConnectService(s.taskSvc), options...)
		s.router.PathPrefix(path).Handler(handler)
	}

	if s.liveDesktopService != nil && s.capturesService != nil {
		path, handler = domainconnect.NewEvidenceServiceHandler(evidence.NewConnectService(s.liveDesktopService, s.capturesService), options...)
		s.router.PathPrefix(path).Handler(handler)
	}

	if s.stateHandler != nil {
		path, handler = domainconnect.NewStateServiceHandler(state.NewConnectService(s.stateHandler), options...)
		s.router.PathPrefix(path).Handler(handler)
	}

	if s.telemetryHandler != nil {
		path, handler = domainconnect.NewTelemetryServiceHandler(telemetry.NewConnectService(s.telemetryHandler), options...)
		s.router.PathPrefix(path).Handler(handler)
	}

	if s.recordsHandler != nil {
		path, handler = domainconnect.NewDesktopRecordsServiceHandler(records.NewConnectService(s.recordsHandler), options...)
		s.router.PathPrefix(path).Handler(handler)
	}

	if s.deployHandler != nil {
		path, handler = domainconnect.NewDeployTargetServiceHandler(deploy.NewConnectService(s.deployHandler), options...)
		s.router.PathPrefix(path).Handler(handler)
	}
}
