package backlog

import (
	"connectrpc.com/connect"
	"context"
	"errors"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/connectx"
	sharedidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/cli-core/cliutil"
	api "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api/apiconnect"
	"net/http"
	"strings"
)

// DevelopmentService has no runner, dispatcher, reservation, grant or stop
// dependency. It adapts the canonical owner; unavailable setup fails closed.
type DevelopmentService struct{ Owner *DevelopmentOwner }

func developmentTransportIdentity(ctx context.Context, write bool) error {
	if _, err := authn.RequireCapability(ctx, "swarm-manager:read"); !write && err != nil {
		return connect.NewError(connect.CodeUnauthenticated, err)
	}
	if write {
		if _, err := authn.RequireHuman(ctx); err != nil {
			return connect.NewError(connect.CodeUnauthenticated, err)
		}
		if _, err := authn.RequireCapability(ctx, "swarm-manager:write"); err != nil {
			return connect.NewError(connect.CodePermissionDenied, err)
		}
	}
	return nil
}
func developmentTransportError(err error) error {
	if err == nil {
		return nil
	}
	return connect.NewError(connect.CodeFailedPrecondition, errors.New("governed development request refused"))
}
func (s *DevelopmentService) GetDevelopment(ctx context.Context, r *connect.Request[api.GetDevelopmentRequest]) (*connect.Response[api.GetDevelopmentResponse], error) {
	if err := developmentTransportIdentity(ctx, false); err != nil {
		return nil, err
	}
	if s == nil || s.Owner == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("governed development owner is not installed"))
	}
	if r == nil || r.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, ErrDevelopmentRefused)
	}
	out, err := s.Owner.GetDevelopment(ctx, r.Msg.EffortId)
	if err != nil {
		return nil, developmentTransportError(err)
	}
	return connect.NewResponse(&api.GetDevelopmentResponse{Development: out}), nil
}
func (s *DevelopmentService) GetDevelopmentArtifact(ctx context.Context, r *connect.Request[api.GetDevelopmentArtifactRequest]) (*connect.Response[api.GetDevelopmentArtifactResponse], error) {
	if err := developmentTransportIdentity(ctx, false); err != nil {
		return nil, err
	}
	if s == nil || s.Owner == nil {
		return nil, connect.NewError(connect.CodeUnavailable, ErrDevelopmentRefused)
	}
	if r == nil || r.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, ErrDevelopmentRefused)
	}
	out, err := s.Owner.GetDevelopmentArtifact(ctx, r.Msg.Reference, r.Msg.ArtifactId)
	if err != nil {
		return nil, developmentTransportError(err)
	}
	return connect.NewResponse(out), nil
}
func (s *DevelopmentService) ApproveDevelopment(ctx context.Context, r *connect.Request[api.ApproveDevelopmentRequest]) (*connect.Response[api.DevelopmentDecisionResponse], error) {
	if err := developmentTransportIdentity(ctx, true); err != nil {
		return nil, err
	}
	if s == nil || s.Owner == nil {
		return nil, connect.NewError(connect.CodeUnavailable, ErrDevelopmentRefused)
	}
	if r == nil || r.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, ErrDevelopmentRefused)
	}
	out, err := s.Owner.ApproveDevelopment(ctx, r.Msg)
	if err != nil {
		return nil, developmentTransportError(err)
	}
	return connect.NewResponse(out), nil
}
func (s *DevelopmentService) RevokeDevelopment(ctx context.Context, r *connect.Request[api.RevokeDevelopmentRequest]) (*connect.Response[api.DevelopmentDecisionResponse], error) {
	if err := developmentTransportIdentity(ctx, true); err != nil {
		return nil, err
	}
	if s == nil || s.Owner == nil {
		return nil, connect.NewError(connect.CodeUnavailable, ErrDevelopmentRefused)
	}
	if r == nil || r.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, ErrDevelopmentRefused)
	}
	out, err := s.Owner.RevokeDevelopment(ctx, r.Msg)
	if err != nil {
		return nil, developmentTransportError(err)
	}
	return connect.NewResponse(out), nil
}
func (s *DevelopmentService) AcceptDevelopment(ctx context.Context, r *connect.Request[api.AcceptDevelopmentRequest]) (*connect.Response[api.DevelopmentDecisionResponse], error) {
	if err := developmentTransportIdentity(ctx, true); err != nil {
		return nil, err
	}
	if s == nil || s.Owner == nil {
		return nil, connect.NewError(connect.CodeUnavailable, ErrDevelopmentRefused)
	}
	if r == nil || r.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, ErrDevelopmentRefused)
	}
	out, err := s.Owner.AcceptDevelopment(ctx, r.Msg)
	if err != nil {
		return nil, developmentTransportError(err)
	}
	return connect.NewResponse(out), nil
}
func (s *DevelopmentService) PreviewDevelopment(ctx context.Context, r *api.PreviewDevelopmentRequest) (*api.PreviewDevelopmentResponse, error) {
	if err := developmentTransportIdentity(ctx, false); err != nil {
		return nil, err
	}
	if s == nil || s.Owner == nil {
		return nil, connect.NewError(connect.CodeUnavailable, ErrDevelopmentRefused)
	}
	out, err := s.Owner.PreviewDevelopment(ctx, r)
	return out, developmentTransportError(err)
}
func RegisterDevelopmentRoutes(router *mux.Router, owner *DevelopmentOwner) {
	path, handler := apiconnect.NewDevelopmentServiceHandler(&DevelopmentService{Owner: owner})
	connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: handler})
}

// DevelopmentCallerMiddleware runs after the existing authentication and run
// provenance middleware. It reconciles only the five Development RPCs and
// Development preview, leaving sibling route identity semantics unchanged.
// A provenance principal is derived only from the server-verified Agent Manager
// result; caller attribution headers, loopback and absent proof never suffice.
func DevelopmentCallerMiddleware(scenarioCookieName string) func(http.Handler) http.Handler {
	scenarioCookieName = strings.TrimSpace(scenarioCookieName)
	if scenarioCookieName == "" {
		scenarioCookieName = "vrooli_access_token"
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case apiconnect.DevelopmentServiceGetDevelopmentProcedure,
				apiconnect.DevelopmentServiceGetDevelopmentArtifactProcedure,
				apiconnect.DevelopmentServiceApproveDevelopmentProcedure,
				apiconnect.DevelopmentServiceRevokeDevelopmentProcedure,
				apiconnect.DevelopmentServiceAcceptDevelopmentProcedure,
				apiconnect.TransitionServicePreviewDevelopmentProcedure:
			default:
				next.ServeHTTP(w, r)
				return
			}
			ctx := r.Context()
			principal, hasPrincipal := sharedidentity.PrincipalFromContext(ctx)
			agent := provenance.FromContext(ctx)
			offeredRun := len(r.Header.Values(cliutil.HeaderAgentIdentityToken)) != 0
			ambiguousHeaders := false
			for _, name := range []string{cliutil.HeaderAgentIdentityToken, "Authorization", "Cf-Access-Jwt-Assertion", "CF-Access-Client-Id", "CF-Access-Client-Secret"} {
				if len(r.Header.Values(name)) > 1 {
					ambiguousHeaders = true
				}
			}
			rejected := ambiguousHeaders || agent.VerificationStatus == provenance.VerificationInvalid || agent.VerificationStatus == provenance.VerificationUnavailable || (offeredRun && !agent.IsVerifiedAgent())
			if agent.IsVerifiedAgent() {
				// Separate verified channels are not an identity mapping. A bad human
				// provider result likewise cannot fall through to a valid run token.
				offeredOtherProof := len(r.Header.Values("Authorization")) != 0 || len(r.Header.Values("Cf-Access-Jwt-Assertion")) != 0 || len(r.Header.Values("CF-Access-Client-Id")) != 0 || len(r.Header.Values("CF-Access-Client-Secret")) != 0
				for _, cookie := range r.Cookies() {
					if cookie.Name == scenarioCookieName || cookie.Name == "vrooli_access_token" {
						offeredOtherProof = true
					}
				}
				if offeredOtherProof || (hasPrincipal && (principal.Kind != sharedidentity.ActorAgent || principal.Source != sharedidentity.SourceAgentProvenance || principal.Subject != agent.RunID)) {
					rejected = true
				}
				if status, ok := sharedidentity.StatusFromContext(ctx); ok && status.FailureClass != "" && status.FailureClass != sharedidentity.FailureMissing {
					rejected = true
				}
				if !rejected {
					ctx = sharedidentity.WithPrincipal(ctx, sharedidentity.Principal{Kind: sharedidentity.ActorAgent, Subject: agent.RunID, Verified: true, Source: sharedidentity.SourceAgentProvenance, Scopes: append([]string(nil), agent.Scopes...)})
				}
			} else if hasPrincipal && principal.Source == sharedidentity.SourceAgentProvenance {
				rejected = true
			}
			if rejected {
				ctx = sharedidentity.WithPrincipal(ctx, sharedidentity.Principal{})
				ctx = sharedidentity.WithStatus(ctx, sharedidentity.Status{State: sharedidentity.StateConflict, FailureClass: sharedidentity.FailureConflict, Source: sharedidentity.SourceUnknown})
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
