package control

import (
	"context"

	"connectrpc.com/connect"
	internalflows "device-control/internal/flows"
	"device-control/internal/sessions"
	"github.com/vrooli/api-core/targetmodel"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	desktopv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop"
	flowsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/flows"
	"google.golang.org/protobuf/types/known/structpb"
)

func (p *desktopOwnerRPC) libraryAdmission(ctx context.Context, session *commonv1.SessionRef, write bool) (ownerDesktopSession, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	s, err := p.resolve(ctx, session)
	if err != nil || p.owner.service.desktopRuns == nil || (write && !s.grant.Lease.Control) || !p.owner.sessionActorAllowed(ctx, s.grant.Lease.Actor, write) {
		return ownerDesktopSession{}, sessions.ErrDesktopAdmission
	}
	return s, nil
}

func (p *desktopOwnerRPC) savedScope(s ownerDesktopSession, contextKey string) internalflows.DesktopFlowScope {
	return internalflows.DesktopFlowScope{Actor: s.grant.Lease.Actor, DeviceID: p.owner.deviceID, Surface: s.grant.Lease.Ref.Surface, ContextKey: contextKey}
}

func savedDesktopProto(saved internalflows.SavedDesktopFlow) (*desktopv1.SavedDesktopFlow, error) {
	f := saved.Flow
	wire := &flowsv1.Flow{Id: f.ID, Name: f.Name, Transport: f.Transport, RequireUnlocked: f.RequireUnlocked, AuthProfileId: f.AuthProfileID, AllowUnredactedCapture: f.AllowUnredactedCapture}
	for _, step := range f.Steps {
		args, err := structpb.NewStruct(step.Arguments)
		if err != nil {
			return nil, err
		}
		wire.Steps = append(wire.Steps, &flowsv1.Step{Id: step.ID, Kind: step.Kind, Target: step.Target, TimeoutMs: step.TimeoutMS, RequiredCapabilities: step.RequiredCapabilities, Arguments: args})
	}
	return &desktopv1.SavedDesktopFlow{Id: saved.ID, Version: saved.Version, ContextKey: saved.Scope.ContextKey, Flow: wire, SourceRunId: saved.Source.RunID, SourceDigest: saved.SourceDigest, CreatedAt: saved.CreatedAt}, nil
}

func (p *desktopOwnerRPC) PromoteFlow(ctx context.Context, r *connect.Request[desktopv1.OwnerPromoteFlowRequest]) (*connect.Response[desktopv1.SavedDesktopFlow], error) {
	s, err := p.libraryAdmission(ctx, r.Msg.Session, true)
	if err != nil {
		return nil, ownerError(err)
	}
	source, err := targetmodel.SessionRefFromProto(r.Msg.SourceSession)
	if err != nil || source.Surface != s.grant.Lease.Ref.Surface {
		return nil, ownerError(sessions.ErrDesktopAdmission)
	}
	scope := internalflows.DesktopRunScope{Actor: s.grant.Lease.Actor, DeviceID: p.owner.deviceID, Surface: source.Surface, DesktopSessionID: source.DesktopSessionID, LeaseID: source.SessionID, RunID: r.Msg.SourceRunId}
	saved, err := p.owner.service.desktopRuns.Promote(ctx, scope, r.Msg.ContextKey, r.Msg.Id, r.Msg.ExpectedVersion)
	if err != nil {
		return nil, ownerError(err)
	}
	if _, err := p.libraryAdmission(ctx, r.Msg.Session, true); err != nil {
		return nil, ownerError(err)
	}
	wire, err := savedDesktopProto(saved)
	if err != nil {
		return nil, ownerError(err)
	}
	return connect.NewResponse(wire), nil
}

func (p *desktopOwnerRPC) GetSavedFlow(ctx context.Context, r *connect.Request[desktopv1.OwnerGetSavedFlowRequest]) (*connect.Response[desktopv1.SavedDesktopFlow], error) {
	s, err := p.libraryAdmission(ctx, r.Msg.Session, false)
	if err != nil {
		return nil, ownerError(err)
	}
	saved, err := p.owner.service.desktopRuns.GetSaved(ctx, p.savedScope(s, r.Msg.ContextKey), r.Msg.Id, r.Msg.Version)
	if err != nil {
		return nil, ownerError(err)
	}
	if _, err := p.libraryAdmission(ctx, r.Msg.Session, false); err != nil {
		return nil, ownerError(err)
	}
	wire, err := savedDesktopProto(saved)
	if err != nil {
		return nil, ownerError(err)
	}
	return connect.NewResponse(wire), nil
}

func (p *desktopOwnerRPC) RunSavedFlow(ctx context.Context, r *connect.Request[desktopv1.OwnerRunSavedFlowRequest]) (*connect.Response[desktopv1.FlowRecord], error) {
	s, err := p.libraryAdmission(ctx, r.Msg.Session, true)
	if err != nil {
		return nil, ownerError(err)
	}
	saved, err := p.owner.service.desktopRuns.GetSaved(ctx, p.savedScope(s, r.Msg.ContextKey), r.Msg.Id, r.Msg.Version)
	if err != nil {
		return nil, ownerError(err)
	}
	wire, err := savedDesktopProto(saved)
	if err != nil {
		return nil, ownerError(err)
	}
	return p.runFlow(ctx, connect.NewRequest(&desktopv1.OwnerRunFlowRequest{Session: r.Msg.Session, RunId: r.Msg.RunId, ApplicationId: r.Msg.ApplicationId, ApplicationRevision: r.Msg.ApplicationRevision, Flow: wire.Flow}), &internalflows.DesktopFlowRevision{ID: saved.ID, Version: saved.Version, ContextKey: saved.Scope.ContextKey})
}
