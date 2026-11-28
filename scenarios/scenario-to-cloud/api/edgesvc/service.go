// Package edgesvc serves the typed edge observation over Connect.
package edgesvc

import (
	"context"
	"net/http"

	"connectrpc.com/connect"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/deploymentsvc"
	"scenario-to-cloud/edge"

	edgev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/edge"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/edge/edgev1connect"
)

// Producer makes a fresh edge observation for one deployment.
type Producer interface {
	ObserveEdge(ctx context.Context, deploymentID string) (*edgev1.EdgeObservation, error)
}

// Service implements edgev1connect.EdgeServiceHandler over a Producer.
type Service struct {
	producer Producer
}

// NewService builds the Connect service.
func NewService(producer Producer) *Service {
	return &Service{producer: producer}
}

// Handler returns the Connect mount path and handler.
func (s *Service) Handler(opts ...connect.HandlerOption) (string, http.Handler) {
	return edgev1connect.NewEdgeServiceHandler(s, opts...)
}

// GetEdgeObservation produces a fresh observation for one deployment.
func (s *Service) GetEdgeObservation(ctx context.Context, req *connect.Request[edgev1.GetEdgeObservationRequest]) (*connect.Response[edgev1.GetEdgeObservationResponse], error) {
	id := req.Msg.GetDeploymentId()
	if id == "" {
		return nil, deploymentsvc.ConnectError(apierrors.New(apierrors.CodeInvalidRequest, "deployment_id is required"))
	}
	obs, err := s.producer.ObserveEdge(ctx, id)
	if err != nil {
		return nil, deploymentsvc.ConnectError(err)
	}
	return connect.NewResponse(edge.Response(obs)), nil
}
