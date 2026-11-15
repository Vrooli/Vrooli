// Package access serves the caller's write access, so a client can offer or
// withhold a write before the caller fills it in, and signs the owner's
// browser in and out through the platform's sign-in edge. The decision comes
// from the same gate that refuses writes; this handler only reports it.
package access

import (
	"context"
	"errors"
	"net/http"

	"money-ledger/internal/access"
	"money-ledger/internal/module"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/browsersession"
	accesspb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/access"
	accessconnect "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/access/access_v1connect"
)

// Resolver decides who a request or a new browser session is; *access.Gate
// implements it.
type Resolver interface {
	Resolve(context.Context) access.Caller
	ResolveSession(ctx context.Context, accessToken string) access.Caller
}

// Sessions is the browser sign-in edge; *browsersession.Edge implements it.
type Sessions interface {
	SignIn(ctx context.Context, in, out http.Header, credentials browsersession.Credentials) (browsersession.Outcome, error)
	Renew(ctx context.Context, in, out http.Header) (browsersession.Outcome, error)
	SignOut(ctx context.Context, in, out http.Header) error
}

type service struct {
	resolver Resolver
	sessions Sessions
}

func Module(resolver Resolver, sessions Sessions) module.Module {
	s := &service{resolver: resolver, sessions: sessions}
	return module.Module{Name: "access", Mount: func(r *mux.Router) {
		p, h := accessconnect.NewAccessServiceHandler(s)
		r.PathPrefix(p).Handler(h)
	}, Endpoints: Endpoints}
}

func (s *service) GetWriteAccess(ctx context.Context, req *connect.Request[accesspb.GetWriteAccessRequest]) (*connect.Response[accesspb.GetWriteAccessResponse], error) {
	response := connect.NewResponse(&accesspb.GetWriteAccessResponse{})
	caller := s.resolver.Resolve(ctx)
	hasSession := browsersession.HasSession(req.Header())
	if expired(caller) && hasSession {
		// The access cookie has lapsed or is missing; the refresh cookie may
		// still renew it. A refused renewal clears the cookies, so the page
		// stops presenting a dead session.
		if renewed, err := s.sessions.Renew(ctx, req.Header(), response.Header()); err == nil {
			caller = s.resolver.ResolveSession(ctx, renewed.AccessToken)
		} else if errors.Is(err, browsersession.ErrInvalidCredentials) {
			hasSession = false
		}
	}
	response.Msg = report(caller, hasSession)
	return response, nil
}

func (s *service) SignIn(ctx context.Context, req *connect.Request[accesspb.SignInRequest]) (*connect.Response[accesspb.SignInResponse], error) {
	response := connect.NewResponse(&accesspb.SignInResponse{})
	outcome, err := s.sessions.SignIn(ctx, req.Header(), response.Header(), browsersession.Credentials{
		Email: req.Msg.GetEmail(), Password: req.Msg.GetPassword(),
		TOTPCode: req.Msg.GetTotpCode(), RecoveryCode: req.Msg.GetRecoveryCode(), MFAChallenge: req.Msg.GetMfaChallenge(),
	})
	if err != nil {
		return nil, browsersession.ConnectError(err)
	}
	if outcome.MFARequired {
		response.Msg = &accesspb.SignInResponse{MfaRequired: true, MfaChallenge: outcome.MFAChallenge}
		return response, nil
	}
	response.Msg.Access = report(s.resolver.ResolveSession(ctx, outcome.AccessToken), true)
	return response, nil
}

func (s *service) SignOut(ctx context.Context, req *connect.Request[accesspb.SignOutRequest]) (*connect.Response[accesspb.SignOutResponse], error) {
	response := connect.NewResponse(&accesspb.SignOutResponse{})
	response.Msg.RevocationConfirmed = s.sessions.SignOut(ctx, req.Header(), response.Header()) == nil
	return response, nil
}

// expired reports a caller a renewed session could change: one that presented
// nothing usable.
func expired(caller access.Caller) bool {
	return caller.Refusal == access.RefusedSignedOut || caller.Refusal == access.RefusedUnverified
}

func report(caller access.Caller, browserSession bool) *accesspb.GetWriteAccessResponse {
	if caller.Writer() {
		return &accesspb.GetWriteAccessResponse{CanWrite: true, Actor: string(caller.Actor), Reason: "writes are recorded as " + string(caller.Actor), SignedInAs: caller.SignedInAs, BrowserSession: browserSession}
	}
	return &accesspb.GetWriteAccessResponse{Refusal: refusals[caller.Refusal], Reason: caller.Reason, SignedInAs: caller.SignedInAs, BrowserSession: browserSession}
}

var refusals = map[access.Refusal]accesspb.WriteRefusal{
	access.RefusedSignedOut:         accesspb.WriteRefusal_WRITE_REFUSAL_SIGNED_OUT,
	access.RefusedUnverified:        accesspb.WriteRefusal_WRITE_REFUSAL_UNVERIFIED,
	access.RefusedNotOwner:          accesspb.WriteRefusal_WRITE_REFUSAL_NOT_OWNER,
	access.RefusedAgent:             accesspb.WriteRefusal_WRITE_REFUSAL_AGENT,
	access.RefusedUndeclaredService: accesspb.WriteRefusal_WRITE_REFUSAL_UNDECLARED_SERVICE,
	access.RefusedConflict:          accesspb.WriteRefusal_WRITE_REFUSAL_CONFLICT,
}

var Endpoints = []module.EndpointDescriptor{
	{ID: "access_get_write_access", Method: http.MethodPost, Path: accessconnect.AccessServiceGetWriteAccessProcedure, Summary: "Report whether the caller may write and, if not, why; renews an expired browser session", Category: "access"},
	{ID: "access_sign_in", Method: http.MethodPost, Path: accessconnect.AccessServiceSignInProcedure, Summary: "Sign the owner's browser in through scenario-authenticator and set the session cookies", Category: "access"},
	{ID: "access_sign_out", Method: http.MethodPost, Path: accessconnect.AccessServiceSignOutProcedure, Summary: "Clear the browser session cookies and revoke the session", Category: "access"},
}
