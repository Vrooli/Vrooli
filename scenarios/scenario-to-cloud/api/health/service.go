package health

import (
	"context"
	"net/http"

	"connectrpc.com/connect"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/deploymentsvc"

	healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/health"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/health/healthv1connect"
)

// Producer makes a fresh observation for one deployment. The server
// implements it over the live-state, DNS and TLS inspections.
type Producer interface {
	Observe(ctx context.Context, deploymentID string) (*healthv1.HealthObservation, error)
}

// Service implements healthv1connect.HealthServiceHandler over a Producer.
type Service struct {
	producer Producer
}

// NewService builds the Connect service.
func NewService(producer Producer) *Service {
	return &Service{producer: producer}
}

// Handler returns the Connect mount path and handler.
func (s *Service) Handler(opts ...connect.HandlerOption) (string, http.Handler) {
	return healthv1connect.NewHealthServiceHandler(s, opts...)
}

// GetHealthObservation produces a fresh observation for one deployment.
func (s *Service) GetHealthObservation(ctx context.Context, req *connect.Request[healthv1.GetHealthObservationRequest]) (*connect.Response[healthv1.GetHealthObservationResponse], error) {
	id := req.Msg.GetDeploymentId()
	if id == "" {
		return nil, deploymentsvc.ConnectError(apierrors.New(apierrors.CodeInvalidRequest, "deployment_id is required"))
	}
	obs, err := s.producer.Observe(ctx, id)
	if err != nil {
		return nil, deploymentsvc.ConnectError(err)
	}
	return connect.NewResponse(Response(obs)), nil
}
