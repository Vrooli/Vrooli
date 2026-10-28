package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	"agent-manager/internal/supervision"
	"connectrpc.com/connect"

	"github.com/vrooli/api-core/authn"
	requestidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/provenance"

	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

type effortActionAuthorizer interface {
	AuthorizeEffortAction(context.Context, string, pb.WatchAuthority) (supervision.EffortActor, error)
}

func effortToken(header http.Header, authority pb.WatchAuthority) string {
	if authority == pb.WatchAuthority_WATCH_AUTHORITY_FAMILY_PARENT {
		if token := header.Get("X-Agent-Identity-Token"); token != "" {
			return token
		}
	}
	return bearerToken(header.Get("Authorization"))
}

func (h *AgentManagerConnectHandler) effortService() (*supervision.EffortService, error) {
	if h.supervision == nil || h.supervision.Efforts == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("effort supervision unavailable"))
	}
	return h.supervision.Efforts, nil
}

func (h *AgentManagerConnectHandler) effortActor(ctx context.Context, token string, authority pb.WatchAuthority) (supervision.EffortActor, error) {
	if h.watchActionAuth == nil {
		return supervision.EffortActor{}, connect.NewError(connect.CodeUnavailable, errors.New("owner authorizer unavailable"))
	}
	if auth, ok := h.watchActionAuth.(effortActionAuthorizer); ok {
		actor, err := auth.AuthorizeEffortAction(ctx, token, authority)
		if err == nil {
			return actor, nil
		}
		code := connect.CodePermissionDenied
		if errors.Is(err, ErrWatchActionUnauthenticated) {
			code = connect.CodeUnauthenticated
		}
		return supervision.EffortActor{}, connect.NewError(code, err)
	}
	r := &pb.RequestCohortWatchActionRequest{Authority: authority}
	if err := h.watchActionAuth.AuthorizeWatchAction(ctx, token, r); err != nil {
		code := connect.CodePermissionDenied
		if errors.Is(err, ErrWatchActionUnauthenticated) {
			code = connect.CodeUnauthenticated
		}
		return supervision.EffortActor{}, connect.NewError(code, err)
	}
	return supervision.EffortActor{ID: r.RequestedBy, Operator: authority == pb.WatchAuthority_WATCH_AUTHORITY_OPERATOR}, nil
}

func (h *AgentManagerConnectHandler) ListEfforts(ctx context.Context, req *connect.Request[pb.ListEffortsRequest]) (*connect.Response[pb.ListEffortsResponse], error) {
	s, err := h.effortService()
	if err != nil {
		return nil, err
	}
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request required"))
	}
	out, err := s.List(ctx, req.Msg)
	if err != nil {
		return nil, watchConnectError(err)
	}
	return connect.NewResponse(out), nil
}

func (h *AgentManagerConnectHandler) GetEffortBoard(ctx context.Context, req *connect.Request[pb.GetEffortBoardRequest]) (*connect.Response[pb.EffortBoard], error) {
	s, err := h.effortService()
	if err != nil {
		return nil, err
	}
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request required"))
	}
	out, err := s.Board(ctx, req.Msg)
	if err != nil {
		return nil, watchConnectError(err)
	}
	return connect.NewResponse(out), nil
}

func (h *AgentManagerConnectHandler) EnrollEffort(ctx context.Context, req *connect.Request[pb.EnrollEffortRequest]) (*connect.Response[pb.EffortEnrollment], error) {
	s, err := h.effortService()
	if err != nil {
		return nil, err
	}
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request required"))
	}
	authority := pb.WatchAuthority_WATCH_AUTHORITY_OPERATOR
	actor, delegated, err := deliveryGrantActor(ctx, req.Msg.GetEnrollment().GetEffortRef())
	if p, ok := requestidentity.PrincipalFromContext(ctx); ok && p.Kind == requestidentity.ActorAgent && (err != nil || req.Msg.GetEnrollment().GetDelivery() == nil) {
		for _, subject := range req.Msg.GetEnrollment().GetSubjects() {
			if subject.GetOwner() == "prompt-manager" && subject.GetKind() == "team-member" {
				// Team setup remains available when the caller also holds a delivery-only
				// grant. The latter cannot authorize an initial enrollment by itself.
				teamActor, teamErr := h.teamEffortActor(ctx, req.Msg.GetEnrollment())
				if teamErr == nil {
					actor, delegated, err = teamActor, true, nil
				} else if err != nil {
					err = teamErr
				}
				break
			}
		}
	}
	if err != nil {
		return nil, err
	}
	if !delegated {
		actor, err = h.effortActor(ctx, effortToken(req.Header(), authority), authority)
		if err != nil {
			return nil, err
		}
	}
	out, err := s.Enroll(ctx, req.Msg, actor)
	if err != nil {
		return nil, watchConnectError(err)
	}
	return connect.NewResponse(out), nil
}

func (h *AgentManagerConnectHandler) ReconcileEffortMetadata(ctx context.Context, req *connect.Request[pb.ReconcileEffortMetadataRequest]) (*connect.Response[pb.EffortEnrollment], error) {
	s, err := h.effortService()
	if err != nil {
		return nil, err
	}
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request required"))
	}
	if len(req.Header().Values(supervision.ExactProofHeader)) > 0 || len(req.Header().Values(supervision.ExactSignatureHeader)) > 0 {
		if len(req.Header().Values(supervision.ExactProofHeader)) != 1 || len(req.Header().Values(supervision.ExactSignatureHeader)) != 1 {
			return nil, connect.NewError(connect.CodePermissionDenied, supervision.ErrExactAuthority)
		}
		var err error
		ctx, err = h.exactVerifier.Verify(ctx, req.Msg, req.Header().Get(supervision.ExactProofHeader), req.Header().Get(supervision.ExactSignatureHeader))
		if err != nil {
			return nil, connect.NewError(connect.CodePermissionDenied, supervision.ErrExactAuthority)
		}
	}
	authority := req.Msg.GetAuthority()
	if authority == pb.WatchAuthority_WATCH_AUTHORITY_UNSPECIFIED {
		authority = pb.WatchAuthority_WATCH_AUTHORITY_OPERATOR
	}
	actor, delegated, err := effortGrantActor(ctx, req.Msg.GetEnrollment().GetEffortRef(), authority)
	if err != nil {
		return nil, err
	}
	if !delegated {
		actor, err = h.effortActor(ctx, effortToken(req.Header(), authority), authority)
		if err != nil {
			return nil, err
		}
		if authority == pb.WatchAuthority_WATCH_AUTHORITY_FAMILY_PARENT {
			actor.MetadataReconciler = true
		}
	}
	out, err := s.ReconcileMetadata(ctx, req.Msg, actor)
	if err != nil {
		return nil, watchConnectError(err)
	}
	return connect.NewResponse(out), nil
}

// EffortGrantTarget is the interactive-grant object for one effort:
// agent-manager:effort/<name> for effort ref effort:<name>.
func EffortGrantTarget(effortRef string) string {
	return "agent-manager:effort/" + strings.TrimPrefix(strings.TrimSpace(effortRef), "effort:")
}

// deliveryGrantActor always handles agent callers, including those without a
// grant, so an agent cannot fall back to a stored human operator credential.
func deliveryGrantActor(ctx context.Context, effortRef string) (supervision.EffortActor, bool, error) {
	p, _ := requestidentity.PrincipalFromContext(ctx)
	if p.Kind != requestidentity.ActorAgent {
		return supervision.EffortActor{}, false, nil
	}
	target := EffortGrantTarget(effortRef)
	if strings.TrimSpace(effortRef) == "" {
		return supervision.EffortActor{}, true, connect.NewError(connect.CodeInvalidArgument, errors.New("effort reference required"))
	}
	if _, err := authn.RequireOperation(ctx, "agent-manager:write", "effort.delivery", target); err != nil {
		return supervision.EffortActor{}, true, connect.NewError(connect.CodePermissionDenied, errors.New("interactive grant does not allow effort.delivery on "+target))
	}
	grant := p.Grant
	proof, err := provenance.ReverifyOperationProof(ctx)
	if err != nil {
		return supervision.EffortActor{}, true, connect.NewError(connect.CodePermissionDenied, errors.New("delivery grant could not be reverified"))
	}
	if proof != nil {
		grant = proof.Grant
	}
	if grant == nil || grant.Approver == "" {
		return supervision.EffortActor{}, true, connect.NewError(connect.CodePermissionDenied, errors.New("owner-approved delivery grant required"))
	}
	log.Printf("agent-manager delivery delegated caller=%s owner=%s grant=%s approver=%s target=%s", p.Subject, p.Owner, grant.ID, grant.Approver, target)
	return supervision.EffortActor{ID: p.Subject, DeliveryManager: true, ApprovedBy: grant.Approver}, true, nil
}

// effortGrantActor admits a live session holding an owner-approved
// interactive grant with effort.reconcile on this exact effort. It reports
// delegated=false for callers without an interactive grant, which keep the
// owner and orchestrator paths. The service still refuses any change beyond
// owner-facing metadata, so the grant cannot touch authority or subjects.
func effortGrantActor(ctx context.Context, effortRef string, authority pb.WatchAuthority) (supervision.EffortActor, bool, error) {
	if actor, present, err := supervision.ExactEffortActor(ctx, effortRef); present {
		if err != nil || authority != pb.WatchAuthority_WATCH_AUTHORITY_OPERATOR {
			return supervision.EffortActor{}, true, connect.NewError(connect.CodePermissionDenied, supervision.ErrExactAuthority)
		}
		return actor, true, nil
	}
	p, ok := requestidentity.PrincipalFromContext(ctx)
	if !ok || p.Kind != requestidentity.ActorAgent {
		return supervision.EffortActor{}, false, nil
	}
	// Supplemental operation authority deliberately stays outside p.Grant.
	// Recognize both supported credential shapes, then let the canonical
	// evaluator decide coverage without unioning ordinary identity scopes.
	grant := p.Grant
	proof, proofErr := provenance.ReverifyOperationProof(ctx)
	if proofErr != nil {
		return supervision.EffortActor{}, true, connect.NewError(connect.CodePermissionDenied, errors.New("interactive effort grant could not be reverified"))
	}
	if proof != nil {
		grant = proof.Grant
	}
	if grant == nil {
		return supervision.EffortActor{}, false, nil
	}
	if authority != pb.WatchAuthority_WATCH_AUTHORITY_OPERATOR {
		return supervision.EffortActor{}, true, connect.NewError(connect.CodePermissionDenied, errors.New("an interactive grant acts only with delegated operator authority"))
	}
	if strings.TrimSpace(effortRef) == "" {
		return supervision.EffortActor{}, true, connect.NewError(connect.CodeInvalidArgument, errors.New("effort reference required"))
	}
	target := EffortGrantTarget(effortRef)
	if _, err := authn.RequireOperation(ctx, "agent-manager:write", "effort.reconcile", target); err != nil {
		return supervision.EffortActor{}, true, connect.NewError(connect.CodePermissionDenied, errors.New("interactive grant does not allow effort.reconcile on "+target))
	}
	log.Printf("agent-manager effort metadata delegated caller=%s owner=%s grant=%s approver=%s target=%s", p.Subject, p.Owner, grant.ID, grant.Approver, target)
	return supervision.EffortActor{ID: p.Subject, Delegated: true, ApprovedBy: grant.Approver}, true, nil
}

func (h *AgentManagerConnectHandler) WithdrawEffort(ctx context.Context, req *connect.Request[pb.WithdrawEffortRequest]) (*connect.Response[pb.EffortEnrollment], error) {
	s, err := h.effortService()
	if err != nil {
		return nil, err
	}
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request required"))
	}
	authority := pb.WatchAuthority_WATCH_AUTHORITY_OPERATOR
	actor, err := h.effortActor(ctx, effortToken(req.Header(), authority), authority)
	if err != nil {
		return nil, err
	}
	out, err := s.Withdraw(ctx, req.Msg, actor)
	if err != nil {
		return nil, watchConnectError(err)
	}
	return connect.NewResponse(out), nil
}

func (h *AgentManagerConnectHandler) ReconcileEffortDiscovery(ctx context.Context, req *connect.Request[pb.ReconcileEffortDiscoveryRequest]) (*connect.Response[pb.EffortDiscovery], error) {
	s, err := h.effortService()
	if err != nil {
		return nil, err
	}
	if req == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("request required"))
	}
	// Explicit scan is authenticated; ordinary board reads stay free of enrollment effects.
	if _, err = h.effortActor(ctx, bearerToken(req.Header().Get("Authorization")), pb.WatchAuthority_WATCH_AUTHORITY_OPERATOR); err != nil {
		return nil, err
	}
	out, err := s.ReconcileDiscovery(ctx)
	if err != nil {
		return nil, watchConnectError(err)
	}
	return connect.NewResponse(out), nil
}
