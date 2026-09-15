package control

import (
	"bytes"
	"context"
	"errors"
	"image/png"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"connectrpc.com/connect"
	"device-control/internal/desktophelper"
	"device-control/internal/sessions"
	"github.com/google/uuid"
	"github.com/vrooli/api-core/localprincipal"
	"github.com/vrooli/api-core/owneridentity"
	"github.com/vrooli/api-core/targetmodel"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	desktopv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop"
	"github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop/desktopv1connect"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type (
	ownerConnectionKey  struct{}
	ownerDesktopSession struct {
		grant sessions.DesktopGrant
		token string
	}
)

type desktopOwnerRPC struct {
	owner    *LocalDesktopAdmission
	helper   desktopv1connect.DesktopHelperServiceClient
	refresh  chan chan error
	mu       sync.Mutex
	sessions map[string]ownerDesktopSession
}

// Describe projects registered identity only. Reading registration does not
// capture the screen, acquire a lease, or establish permission to inject input.
func (p *desktopOwnerRPC) Describe(ctx context.Context, _ *connect.Request[desktopv1.OwnerDescribeRequest]) (*connect.Response[desktopv1.OwnerDescribeResponse], error) {
	registration, err := p.owner.registration(ctx)
	if err != nil || registration.Surface != p.owner.surface || registration.SessionID == "" {
		return nil, ownerError(sessions.ErrDesktopAdmission)
	}
	now := time.Now().UTC()
	descriptor := targetmodel.SurfaceDescriptor{Ref: p.owner.surface, Kind: targetmodel.SurfaceDesktop, DisplayLabel: "Desktop session " + registration.SessionID, DesktopSessionID: registration.SessionID, ProtocolVersions: []string{"vrooli.desktop.owner.v1"}}
	for _, capability := range []string{"desktop.observe", "desktop.pointer", "desktop.semantic"} {
		descriptor.Capabilities = append(descriptor.Capabilities, targetmodel.CapabilityFact{Capability: capability, State: targetmodel.CapabilityUnknown, ReasonCode: "owner_admission_required", ObservedAt: now, ExpiresAt: now.Add(5 * time.Second)})
	}
	wire, err := descriptor.Proto()
	if err != nil {
		return nil, ownerError(err)
	}
	return connect.NewResponse(&desktopv1.OwnerDescribeResponse{Surface: wire}), nil
}

func ownerError(err error) error {
	if errors.Is(err, sessions.ErrDesktopAdmission) {
		return connect.NewError(connect.CodePermissionDenied, err)
	}
	return connect.NewError(connect.CodeUnavailable, errors.New("desktop operation incomplete; do not blindly retry input"))
}

func (p *desktopOwnerRPC) publish(ctx context.Context) error {
	reply := make(chan error, 1)
	select {
	case p.refresh <- reply:
	case <-ctx.Done():
		return ctx.Err()
	}
	select {
	case err := <-reply:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func helperLease(g sessions.DesktopGrant) *desktopv1.Lease {
	l := g.Lease
	return &desktopv1.Lease{Ref: l.Ref.Proto(), Actor: l.Actor, HelperId: l.HelperID, Epoch: l.Epoch, ExpiresAt: timestamppb.New(l.ExpiresAt), Control: l.Control}
}

func helperRequest[T any](msg *T, token string) *connect.Request[T] {
	r := connect.NewRequest(msg)
	r.Header().Set("Authorization", "DesktopGrant "+token)
	return r
}

func (p *desktopOwnerRPC) Open(ctx context.Context, r *connect.Request[desktopv1.OwnerOpenRequest]) (*connect.Response[desktopv1.OwnerOpenResponse], error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	surface, err := targetmodel.SurfaceRefFromProto(r.Msg.Surface)
	if err != nil || surface != p.owner.surface {
		return nil, ownerError(sessions.ErrDesktopAdmission)
	}
	peer, ok := ctx.Value(ownerConnectionKey{}).(*net.UnixConn)
	if !ok {
		return nil, ownerError(sessions.ErrDesktopAdmission)
	}
	forwarded := false
	if r.Msg.RequestId != "" {
		actor, err := p.requestActor(ctx, r.Msg.Control)
		if err != nil {
			return nil, ownerError(err)
		}
		_, created, err := p.owner.admissions.ReserveOpen(ctx, r.Msg.RequestId, actor, sessions.DesktopOpenIntent{Surface: surface, Control: r.Msg.Control, TTLSeconds: r.Msg.TtlSeconds}, time.Now().Add(10*time.Second))
		if err != nil {
			return nil, ownerError(err)
		}
		if !created {
			return nil, ownerError(sessions.ErrDesktopAdmission)
		}
		defer func() {
			if !forwarded {
				cleanup, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()
				_ = p.owner.admissions.RejectOpen(cleanup, r.Msg.RequestId, actor)
			}
		}()
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	for id, s := range p.sessions {
		active, _ := p.owner.Active(ctx, s.grant.ID)
		if !active {
			delete(p.sessions, id)
		}
	}
	grant, token, err := p.owner.Acquire(ctx, peer, time.Duration(r.Msg.TtlSeconds)*time.Second, r.Msg.Control)
	if err != nil {
		return nil, ownerError(err)
	}
	success := false
	defer func() {
		if !success {
			cleanup, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			_, _ = p.owner.service.ReleaseContext(cleanup, grant.Lease.Ref.SessionID)
			_ = p.publish(cleanup)
		}
	}()
	// Persist before any helper Open can take effect, including an unknown reply.
	if r.Msg.RequestId == "" {
		if err = p.owner.admissions.Put(ctx, grant); err != nil {
			return nil, ownerError(err)
		}
	}
	if err = p.publish(ctx); err != nil {
		return nil, ownerError(err)
	}
	if r.Msg.RequestId != "" {
		forwarded, err = p.owner.admissions.BindOpen(ctx, r.Msg.RequestId, grant.Lease.Actor, grant)
		if err != nil {
			return nil, ownerError(err)
		}
		if !forwarded {
			return nil, ownerError(sessions.ErrDesktopAdmission)
		}
	}
	_, err = p.helper.Open(ctx, helperRequest(&desktopv1.OpenRequest{Lease: helperLease(grant), ExpectedEpoch: grant.Lease.Epoch - 1}, token))
	if err != nil {
		return nil, ownerError(err)
	}
	p.sessions[grant.Lease.Ref.SessionID] = ownerDesktopSession{grant: grant, token: token}
	success = true
	return connect.NewResponse(&desktopv1.OwnerOpenResponse{Session: grant.Lease.Ref.Proto(), ExpiresAt: timestamppb.New(grant.Lease.ExpiresAt), Control: grant.Lease.Control}), nil
}

// Caller identity is checked for every request by the Unix-only server below.
func (p *desktopOwnerRPC) resolve(ctx context.Context, wire *commonv1.SessionRef) (ownerDesktopSession, error) {
	ref, err := targetmodel.SessionRefFromProto(wire)
	if err != nil {
		return ownerDesktopSession{}, sessions.ErrDesktopAdmission
	}
	s, ok := p.sessions[ref.SessionID]
	if !ok || s.grant.Lease.Ref != ref || !p.owner.sessionActorAllowed(ctx, s.grant.Lease.Actor, false) {
		return ownerDesktopSession{}, sessions.ErrDesktopAdmission
	}
	active, err := p.owner.Active(ctx, s.grant.ID)
	if err != nil || !active {
		return ownerDesktopSession{}, sessions.ErrDesktopAdmission
	}
	return s, nil
}

func (p *desktopOwnerRPC) Observe(ctx context.Context, r *connect.Request[desktopv1.OwnerObserveRequest]) (*connect.Response[desktopv1.ObserveResponse], error) {
	p.mu.Lock()
	s, err := p.resolve(ctx, r.Msg.Session)
	p.mu.Unlock()
	if err != nil {
		return nil, ownerError(err)
	}
	out, err := p.helper.Observe(ctx, helperRequest(&desktopv1.ObserveRequest{Lease: helperLease(s.grant), ProcessId: r.Msg.ProcessId, ApplicationId: r.Msg.ApplicationId, ApplicationRevision: r.Msg.ApplicationRevision}, s.token))
	if err != nil {
		return nil, ownerError(err)
	}
	return connect.NewResponse(out.Msg), nil
}

func (p *desktopOwnerRPC) Act(ctx context.Context, r *connect.Request[desktopv1.OwnerActRequest]) (*connect.Response[desktopv1.ActResponse], error) {
	p.mu.Lock()
	s, err := p.resolve(ctx, r.Msg.Session)
	p.mu.Unlock()
	if err != nil {
		return nil, ownerError(err)
	}
	if !s.grant.Lease.Control || !p.owner.sessionActorAllowed(ctx, s.grant.Lease.Actor, true) {
		return nil, ownerError(sessions.ErrDesktopAdmission)
	}
	out, err := p.helper.Act(ctx, helperRequest(&desktopv1.ActRequest{Lease: helperLease(s.grant), CommandId: r.Msg.CommandId, GeometryRevision: r.Msg.GeometryRevision, Action: r.Msg.Action}, s.token))
	if err != nil {
		return nil, ownerError(err)
	}
	return connect.NewResponse(out.Msg), nil
}

func (p *desktopOwnerRPC) Stop(ctx context.Context, r *connect.Request[desktopv1.OwnerStopRequest]) (*connect.Response[desktopv1.StopResponse], error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	s, err := p.resolve(ctx, r.Msg.Session)
	if err != nil {
		return nil, ownerError(err)
	}
	_, stopErr := p.helper.Stop(ctx, helperRequest(&desktopv1.StopRequest{Lease: helperLease(s.grant)}, s.token))
	cleanup, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, releaseErr := p.owner.service.ReleaseContext(cleanup, s.grant.Lease.Ref.SessionID)
	publishErr := p.publish(cleanup)
	delete(p.sessions, s.grant.Lease.Ref.SessionID)
	if err = errors.Join(stopErr, releaseErr, publishErr); err != nil {
		return nil, ownerError(err)
	}
	return connect.NewResponse(&desktopv1.StopResponse{}), nil
}

func (a *LocalDesktopAdmission) serveDesktopOwner(ctx context.Context, listener *net.UnixListener, config desktophelper.Config) error {
	child, cancel := context.WithCancel(ctx)
	defer cancel()
	transport := &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", filepath.Join(config.StateDirectory, "helper.sock"))
	}}
	defer transport.CloseIdleConnections()
	rpc := &desktopOwnerRPC{owner: a, helper: desktopv1connect.NewDesktopHelperServiceClient(&http.Client{Transport: transport, Timeout: 10 * time.Second}, "http://desktop-helper", connect.WithReadMaxBytes(33*1024*1024)), refresh: make(chan chan error), sessions: make(map[string]ownerDesktopSession)}
	path, handler := desktopv1connect.NewDesktopOwnerServiceHandler(rpc, connect.WithReadMaxBytes(128*1024))
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	accountPath, accountHandler := desktopv1connect.NewDesktopAccountServiceHandler(rpc, connect.WithReadMaxBytes(128*1024))
	mux.Handle(accountPath, accountHandler)
	var count atomic.Int32
	server := &http.Server{
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 40 * time.Second, IdleTimeout: 15 * time.Second, MaxHeaderBytes: 16 * 1024,
		ConnContext: func(ctx context.Context, c net.Conn) context.Context {
			return context.WithValue(ctx, ownerConnectionKey{}, c)
		},
		ConnState: func(c net.Conn, state http.ConnState) {
			switch state {
			case http.StateNew:
				if count.Add(1) > 32 {
					c.Close()
				}
			case http.StateClosed, http.StateHijacked:
				count.Add(-1)
			}
		},
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			peer, ok := r.Context().Value(ownerConnectionKey{}).(*net.UnixConn)
			if !ok {
				http.Error(w, "desktop admission refused", http.StatusUnauthorized)
				return
			}
			principal, err := localprincipal.Peer(peer)
			if err != nil || principal != a.principal {
				http.Error(w, "desktop admission refused", http.StatusUnauthorized)
				return
			}
			if strings.HasPrefix(r.URL.Path, accountPath) && r.Header.Get("Authorization") == "" {
				http.Error(w, "operator authentication required", http.StatusUnauthorized)
				return
			}
			requestCtx, err := a.authenticateForwarded(r.Context(), r.Header.Get("Authorization"))
			if err != nil {
				http.Error(w, "desktop admission refused", http.StatusUnauthorized)
				return
			}
			mux.ServeHTTP(w, r.WithContext(requestCtx))
		}),
	}
	serving, publishing := make(chan error, 1), make(chan error, 1)
	go func() { serving <- server.Serve(listener) }()
	go func() {
		publishing <- desktophelper.MaintainGrantStatusWithRefresh(child, config.GrantStatusFile, a.GrantStatus, rpc.refresh)
	}()
	var serveErr, publishErr error
	gotServe, gotPublish := false, false
	select {
	case <-ctx.Done():
	case serveErr = <-serving:
		gotServe = true
	case publishErr = <-publishing:
		gotPublish = true
	}
	cancel()
	_ = server.Close()
	if !gotServe {
		serveErr = <-serving
	}
	if !gotPublish {
		publishErr = <-publishing
	}
	// Revoke ordinary owner exclusion too, so a worker restart does not strand
	// held device leases until expiry. Status is already empty on publisher exit.
	rpc.mu.Lock()
	for _, s := range rpc.sessions {
		cleanup, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_, _ = a.service.ReleaseContext(cleanup, s.grant.Lease.Ref.SessionID)
		cancel()
	}
	rpc.mu.Unlock()
	if errors.Is(serveErr, http.ErrServerClosed) {
		serveErr = nil
	}
	return errors.Join(serveErr, publishErr)
}

func (p *desktopOwnerRPC) Applications(ctx context.Context, r *connect.Request[desktopv1.OwnerApplicationsRequest]) (*connect.Response[desktopv1.ApplicationsResponse], error) {
	p.mu.Lock()
	s, err := p.resolve(ctx, r.Msg.Session)
	p.mu.Unlock()
	if err != nil {
		return nil, ownerError(err)
	}
	out, err := p.helper.Applications(ctx, helperRequest(&desktopv1.ApplicationsRequest{Lease: helperLease(s.grant)}, s.token))
	if err != nil {
		return nil, ownerError(err)
	}
	return connect.NewResponse(out.Msg), nil
}

func (p *desktopOwnerRPC) Resolve(ctx context.Context, r *connect.Request[desktopv1.OwnerResolveRequest]) (*connect.Response[desktopv1.ResolveResponse], error) {
	p.mu.Lock()
	session, err := p.resolve(ctx, r.Msg.Session)
	p.mu.Unlock()
	if err != nil {
		return nil, ownerError(err)
	}
	out, err := p.helper.Resolve(ctx, helperRequest(&desktopv1.ResolveRequest{Lease: helperLease(session.grant), Selector: r.Msg.Selector}, session.token))
	if err != nil {
		return nil, ownerError(err)
	}
	return connect.NewResponse(out.Msg), nil
}

// Historical access authenticates the actor again but never restores an active
// owner lease or helper input grant. Missing evidence remains an error.
func (p *desktopOwnerRPC) ListAdmissions(ctx context.Context, r *connect.Request[desktopv1.OwnerListAdmissionsRequest]) (*connect.Response[desktopv1.OwnerListAdmissionsResponse], error) {
	actor := p.owner.principal.String()
	if identity, ok := ctx.Value(desktopWebIdentityKey{}).(owneridentity.Identity); ok {
		actor = identity.Subject
	}
	if !p.owner.sessionActorAllowed(ctx, actor, false) {
		return nil, ownerError(sessions.ErrDesktopAdmission)
	}
	page, err := p.owner.admissions.List(ctx, p.owner.surface, actor, r.Msg.PageToken, int(r.Msg.PageSize))
	if err != nil {
		return nil, ownerError(err)
	}
	out := &desktopv1.OwnerListAdmissionsResponse{NextPageToken: page.NextPageToken}
	for _, grant := range page.Grants {
		out.Admissions = append(out.Admissions, &desktopv1.OwnerAdmission{Session: grant.Lease.Ref.Proto(), ExpiresAt: timestamppb.New(grant.Lease.ExpiresAt), Control: grant.Lease.Control})
	}
	return connect.NewResponse(out), nil
}

func (p *desktopOwnerRPC) ReadCleanup(ctx context.Context, r *connect.Request[desktopv1.OwnerStopRequest]) (*connect.Response[desktopv1.OwnerCleanupResponse], error) {
	ref, err := targetmodel.SessionRefFromProto(r.Msg.Session)
	if err != nil || ref.Surface != p.owner.surface {
		return nil, ownerError(sessions.ErrDesktopAdmission)
	}
	actor := p.owner.principal.String()
	if identity, ok := ctx.Value(desktopWebIdentityKey{}).(owneridentity.Identity); ok {
		actor = identity.Subject
	}
	if !p.owner.sessionActorAllowed(ctx, actor, false) {
		return nil, ownerError(sessions.ErrDesktopAdmission)
	}
	grant, err := p.owner.admissions.Get(ctx, ref, actor)
	if err != nil {
		return nil, ownerError(err)
	}
	active, err := p.owner.Active(ctx, grant.ID)
	if err != nil {
		return nil, ownerError(err)
	}
	grant.Revoked = !active
	token, err := sessions.SignDesktopGrant(p.owner.key, grant, grant.IssuedAt)
	if err != nil {
		return nil, ownerError(err)
	}
	request := connect.NewRequest(&desktopv1.StopRequest{Lease: helperLease(grant)})
	request.Header().Set("Authorization", "DesktopCleanup "+token)
	reply, err := p.helper.ReadCleanup(ctx, request)
	if err != nil {
		return nil, ownerError(err)
	}
	if reply == nil || reply.Msg == nil || !proto.Equal(reply.Msg.Lease, helperLease(grant)) || reply.Msg.ObservedAt == nil || reply.Msg.ObservedAt.CheckValid() != nil {
		return nil, ownerError(sessions.ErrDesktopAdmission)
	}
	return connect.NewResponse(&desktopv1.OwnerCleanupResponse{Session: ref.Proto(), Released: reply.Msg.Released, ObservedAt: reply.Msg.ObservedAt}), nil
}

func (p *desktopOwnerRPC) requestActor(ctx context.Context, control bool) (string, error) {
	actor := p.owner.principal.String()
	if identity, ok := ctx.Value(desktopWebIdentityKey{}).(owneridentity.Identity); ok {
		actor = identity.Subject
	}
	if !p.owner.sessionActorAllowed(ctx, actor, control) {
		return "", sessions.ErrDesktopAdmission
	}
	return actor, nil
}

func (p *desktopOwnerRPC) ReconcileOpen(ctx context.Context, r *connect.Request[desktopv1.OwnerOpenRequest]) (*connect.Response[desktopv1.OwnerOpenDisposition], error) {
	surface, err := targetmodel.SurfaceRefFromProto(r.Msg.Surface)
	if err != nil || surface != p.owner.surface {
		return nil, ownerError(sessions.ErrDesktopAdmission)
	}
	actor, err := p.requestActor(ctx, false)
	if err != nil {
		return nil, ownerError(err)
	}
	attempt, err := p.owner.admissions.ReconcileOpen(ctx, r.Msg.RequestId, actor, sessions.DesktopOpenIntent{Surface: surface, Control: r.Msg.Control, TTLSeconds: r.Msg.TtlSeconds})
	if err != nil {
		return nil, ownerError(err)
	}
	out := &desktopv1.OwnerOpenDisposition{RequestId: attempt.ID, State: attempt.State}
	if attempt.Grant != nil {
		out.Session = attempt.Grant.Lease.Ref.Proto()
		out.ExpiresAt = timestamppb.New(attempt.Grant.Lease.ExpiresAt)
		out.Control = attempt.Grant.Lease.Control
	}
	return connect.NewResponse(out), nil
}

func (p *desktopOwnerRPC) CaptureActivation(ctx context.Context, r *connect.Request[desktopv1.OwnerStopRequest]) (*connect.Response[desktopv1.ActivationReference], error) {
	return p.activation(ctx, r.Msg.Session, "", false, 0, 0, false)
}

func (p *desktopOwnerRPC) ReadActivation(ctx context.Context, r *connect.Request[desktopv1.OwnerReadActivationRequest]) (*connect.Response[desktopv1.ActivationReference], error) {
	return p.activation(ctx, r.Msg.Session, r.Msg.ContextId, true, 0, 0, false)
}

func (p *desktopOwnerRPC) activation(ctx context.Context, ref *commonv1.SessionRef, id string, read bool, window uint64, pid uint32, includeImage bool) (*connect.Response[desktopv1.ActivationReference], error) {
	if read {
		if _, err := uuid.Parse(id); err != nil || len(id) != 36 {
			return nil, ownerError(sessions.ErrDesktopAdmission)
		}
	}
	p.mu.Lock()
	session, err := p.resolve(ctx, ref)
	p.mu.Unlock()
	if err != nil {
		return nil, ownerError(err)
	}
	var out *connect.Response[desktopv1.ActivationReference]
	if read {
		out, err = p.helper.ReadActivation(ctx, helperRequest(&desktopv1.ReadActivationRequest{Lease: helperLease(session.grant), ContextId: id}, session.token))
	} else {
		if window != 0 {
			out, err = p.helper.CaptureCompanionActivation(ctx, helperRequest(&desktopv1.CompanionActivationRequest{Lease: helperLease(session.grant), CompanionWindow: window, CompanionPid: pid, IncludeImage: includeImage}, session.token))
		} else {
			out, err = p.helper.CaptureActivation(ctx, helperRequest(&desktopv1.StopRequest{Lease: helperLease(session.grant)}, session.token))
		}
	}
	if err != nil {
		return nil, ownerError(err)
	}
	// A helper reply cannot outlive current owner admission or silently substitute
	// another context during a read. Return a known-field projection only.
	p.mu.Lock()
	current, err := p.resolve(ctx, ref)
	p.mu.Unlock()
	if err != nil || current.grant.ID != session.grant.ID || ctx.Err() != nil {
		return nil, ownerError(sessions.ErrDesktopAdmission)
	}
	if out == nil || !validActivationReference(out.Msg, session.grant.Lease.ExpiresAt, time.Now()) || (read && out.Msg.ContextId != id) || (!read && out.Msg.HasImage != includeImage) {
		return nil, ownerError(sessions.ErrDesktopAdmission)
	}
	return connect.NewResponse(projectActivationReference(out.Msg)), nil
}

func projectActivationReference(value *desktopv1.ActivationReference) *desktopv1.ActivationReference {
	return &desktopv1.ActivationReference{HasImage: value.HasImage, SourceBounds: &desktopv1.DesktopBounds{X: value.SourceBounds.X, Y: value.SourceBounds.Y, Width: value.SourceBounds.Width, Height: value.SourceBounds.Height}, ContextId: value.ContextId, DisplayId: value.DisplayId, GeometryRevision: value.GeometryRevision, PointerX: value.PointerX, PointerY: value.PointerY, CapturedAt: timestamppb.New(value.CapturedAt.AsTime()), ExpiresAt: timestamppb.New(value.ExpiresAt.AsTime())}
}

func validActivationReference(value *desktopv1.ActivationReference, leaseExpiry, now time.Time) bool {
	if value == nil || value.SourceBounds == nil || !(sessions.DesktopBounds{X: value.SourceBounds.X, Y: value.SourceBounds.Y, Width: value.SourceBounds.Width, Height: value.SourceBounds.Height}).Valid() || value.CapturedAt == nil || value.ExpiresAt == nil || value.CapturedAt.CheckValid() != nil || value.ExpiresAt.CheckValid() != nil {
		return false
	}
	id, err := uuid.Parse(value.ContextId)
	if err != nil || id == uuid.Nil || id.String() != value.ContextId || value.DisplayId == "" || len(value.DisplayId) > 128 || value.GeometryRevision == "" || len(value.GeometryRevision) > 128 {
		return false
	}
	captured, expires := value.CapturedAt.AsTime(), value.ExpiresAt.AsTime()
	return !captured.After(now) && now.Before(expires) && expires.After(captured) && !expires.After(captured.Add(30*time.Second)) && !expires.After(leaseExpiry)
}

// Companion capture is local-only. A forwarding web identity must never be used
// as the identity of the native process owning a window on this desktop.
func (p *desktopOwnerRPC) CaptureCompanionActivation(ctx context.Context, r *connect.Request[desktopv1.OwnerCompanionActivationRequest]) (*connect.Response[desktopv1.ActivationReference], error) {
	if _, forwarded := ctx.Value(desktopWebIdentityKey{}).(owneridentity.Identity); forwarded {
		return nil, ownerError(sessions.ErrDesktopAdmission)
	}
	socket, ok := ctx.Value(ownerConnectionKey{}).(*net.UnixConn)
	if !ok || r.Msg.CompanionWindow == 0 {
		return nil, ownerError(sessions.ErrDesktopAdmission)
	}
	pid, err := localprincipal.PeerPID(socket)
	if err != nil {
		return nil, ownerError(sessions.ErrDesktopAdmission)
	}
	return p.activation(ctx, r.Msg.Session, "", false, r.Msg.CompanionWindow, pid, r.Msg.IncludeImage)
}

func (p *desktopOwnerRPC) DeleteActivation(ctx context.Context, r *connect.Request[desktopv1.OwnerReadActivationRequest]) (*connect.Response[desktopv1.StopResponse], error) {
	id, err := uuid.Parse(r.Msg.ContextId)
	if err != nil || id == uuid.Nil || id.String() != r.Msg.ContextId {
		return nil, ownerError(sessions.ErrDesktopAdmission)
	}
	p.mu.Lock()
	session, err := p.resolve(ctx, r.Msg.Session)
	p.mu.Unlock()
	if err != nil {
		return nil, ownerError(err)
	}
	_, err = p.helper.DeleteActivation(ctx, helperRequest(&desktopv1.ReadActivationRequest{Lease: helperLease(session.grant), ContextId: r.Msg.ContextId}, session.token))
	if err != nil {
		return nil, ownerError(err)
	}
	return connect.NewResponse(&desktopv1.StopResponse{}), nil
}

func (p *desktopOwnerRPC) ReadActivationImage(ctx context.Context, r *connect.Request[desktopv1.OwnerReadActivationRequest]) (*connect.Response[desktopv1.ActivationImage], error) {
	id, err := uuid.Parse(r.Msg.ContextId)
	if err != nil || id == uuid.Nil || id.String() != r.Msg.ContextId {
		return nil, ownerError(sessions.ErrDesktopAdmission)
	}
	p.mu.Lock()
	session, err := p.resolve(ctx, r.Msg.Session)
	p.mu.Unlock()
	if err != nil {
		return nil, ownerError(err)
	}
	out, err := p.helper.ReadActivationImage(ctx, helperRequest(&desktopv1.ReadActivationRequest{Lease: helperLease(session.grant), ContextId: r.Msg.ContextId}, session.token))
	if err != nil {
		return nil, ownerError(err)
	}
	if out == nil || out.Msg == nil || !validActivationImage(out.Msg, session.grant.Lease.ExpiresAt, time.Now()) || out.Msg.Reference.ContextId != r.Msg.ContextId {
		return nil, ownerError(sessions.ErrDesktopAdmission)
	}
	// Reuse owner and helper admission/read validation after pixel transfer.
	current, err := p.activation(ctx, r.Msg.Session, r.Msg.ContextId, true, 0, 0, false)
	if err != nil || !proto.Equal(current.Msg, projectActivationReference(out.Msg.Reference)) {
		return nil, ownerError(sessions.ErrDesktopAdmission)
	}
	return connect.NewResponse(&desktopv1.ActivationImage{Reference: current.Msg, Png: out.Msg.Png}), nil
}

func validActivationImage(value *desktopv1.ActivationImage, leaseExpiry, now time.Time) bool {
	if value == nil || !validActivationReference(value.Reference, leaseExpiry, now) || !value.Reference.HasImage || len(value.Png) == 0 || len(value.Png) > 32*1024*1024 {
		return false
	}
	bounds := value.Reference.SourceBounds
	if uint64(bounds.Width)*uint64(bounds.Height) > 16*1024*1024 {
		return false
	}
	config, err := png.DecodeConfig(bytes.NewReader(value.Png))
	return err == nil && config.Width == int(bounds.Width) && config.Height == int(bounds.Height)
}
