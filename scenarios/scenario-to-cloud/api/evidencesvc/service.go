// Package evidencesvc is the Connect implementation of
// vrooli.scenario_to_cloud.v1.evidence.EvidenceService. The application
// logic lives on the API server (Publisher); this package adapts requests
// and maps failures onto the typed apierrors.Error.
package evidencesvc

import (
	"context"
	"net/http"

	"connectrpc.com/connect"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/deploymentsvc"
	"scenario-to-cloud/evidence"

	evidencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/evidence"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/evidence/evidencev1connect"
)

// Publisher is the transport-free publication surface the server provides.
type Publisher interface {
	ReleaseEvidence(ctx context.Context, releaseDigest, deploymentID string) (*evidence.Summary, error)
	RequestPublication(ctx context.Context, deploymentID, requestKey string, identity evidence.ReviewIdentity) (*evidence.Publication, *evidence.Summary, error)
	ApplyPublication(ctx context.Context, deploymentID, requestKey, reviewRef, planDigest string) (*evidence.Publication, *evidence.Summary, error)
	GetPublication(ctx context.Context, deploymentID, requestKey string) (*evidence.Publication, *evidence.Summary, error)
}

// Service implements evidencev1connect.EvidenceServiceHandler.
type Service struct {
	evidencev1connect.UnimplementedEvidenceServiceHandler
	publisher Publisher
}

// New builds the service.
func New(publisher Publisher) *Service { return &Service{publisher: publisher} }

// Handler returns the Connect mount path and handler.
func (s *Service) Handler(opts ...connect.HandlerOption) (string, http.Handler) {
	return evidencev1connect.NewEvidenceServiceHandler(s, opts...)
}

func fail(err error) error {
	if err == nil {
		return nil
	}
	if apierrors.As(err) == nil {
		err = apierrors.Internal("evidence service failed", err)
	}
	return deploymentsvc.ConnectError(err)
}

// GetReleaseEvidence implements the RPC.
func (s *Service) GetReleaseEvidence(ctx context.Context, req *connect.Request[evidencev1.GetReleaseEvidenceRequest]) (*connect.Response[evidencev1.ReleaseEvidence], error) {
	summary, err := s.publisher.ReleaseEvidence(ctx, req.Msg.GetReleaseDigest(), req.Msg.GetDeploymentId())
	if err != nil {
		return nil, fail(err)
	}
	return connect.NewResponse(summary.Proto()), nil
}

// RequestPublication implements the RPC.
func (s *Service) RequestPublication(ctx context.Context, req *connect.Request[evidencev1.PublicationRequest]) (*connect.Response[evidencev1.Publication], error) {
	pub, summary, err := s.publisher.RequestPublication(ctx, req.Msg.GetDeploymentId(), req.Msg.GetRequestKey(), evidence.IdentityFromProto(req.Msg.GetIdentity()))
	if err != nil {
		return nil, fail(err)
	}
	return connect.NewResponse(pub.Proto(summary)), nil
}

// ApplyPublication implements the RPC.
func (s *Service) ApplyPublication(ctx context.Context, req *connect.Request[evidencev1.ApplyPublicationRequest]) (*connect.Response[evidencev1.Publication], error) {
	pub, summary, err := s.publisher.ApplyPublication(ctx, req.Msg.GetDeploymentId(), req.Msg.GetRequestKey(), req.Msg.GetReviewRef(), req.Msg.GetPlanDigest())
	if err != nil {
		return nil, fail(err)
	}
	return connect.NewResponse(pub.Proto(summary)), nil
}

// GetPublication implements the RPC.
func (s *Service) GetPublication(ctx context.Context, req *connect.Request[evidencev1.GetPublicationRequest]) (*connect.Response[evidencev1.Publication], error) {
	pub, summary, err := s.publisher.GetPublication(ctx, req.Msg.GetDeploymentId(), req.Msg.GetRequestKey())
	if err != nil {
		return nil, fail(err)
	}
	return connect.NewResponse(pub.Proto(summary)), nil
}
