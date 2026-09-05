package control

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"connectrpc.com/connect"
	"device-control/internal/execution"
	internalflows "device-control/internal/flows"
	"device-control/internal/sessions"
	desktopv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop"
	"google.golang.org/protobuf/proto"
)

// RunFlow preserves the caller context for every owner operation. Holding p.mu
// only for admission lets Stop revoke the lease between bounded native calls.
func (p *desktopOwnerRPC) RunFlow(ctx context.Context, r *connect.Request[desktopv1.OwnerRunFlowRequest]) (*connect.Response[desktopv1.FlowRecord], error) {
	return p.runFlow(ctx, r, nil)
}

func (p *desktopOwnerRPC) runFlow(ctx context.Context, r *connect.Request[desktopv1.OwnerRunFlowRequest], saved *internalflows.DesktopFlowRevision) (*connect.Response[desktopv1.FlowRecord], error) {
	request := r.Msg
	if request.Flow == nil || request.Session == nil || request.RunId == "" || len(request.RunId) > 120 || strings.ContainsRune(request.RunId, ':') || request.ApplicationId == "" || request.ApplicationRevision == "" {
		return nil, ownerError(sessions.ErrDesktopAdmission)
	}
	source := request.Flow
	flow := execution.Flow{ID: source.Id, Name: source.Name, Transport: source.Transport, RequireUnlocked: source.RequireUnlocked, AuthProfileID: source.AuthProfileId, AllowUnredactedCapture: source.AllowUnredactedCapture}
	for _, step := range source.Steps {
		if step == nil {
			return nil, ownerError(sessions.ErrDesktopAdmission)
		}
		flow.Steps = append(flow.Steps, execution.Step{ID: step.Id, Kind: step.Kind, Target: step.Target, TimeoutMS: step.TimeoutMs, RequiredCapabilities: step.RequiredCapabilities, Arguments: step.Arguments.AsMap()})
	}
	if internalflows.ValidateDesktopFlow(flow) != nil {
		return nil, ownerError(sessions.ErrDesktopAdmission)
	}
	if p.owner.service.desktopRuns == nil {
		return nil, ownerError(sessions.ErrDesktopAdmission)
	}
	encoded, err := (proto.MarshalOptions{Deterministic: true}).Marshal(request)
	if err != nil {
		return nil, ownerError(err)
	}
	if saved != nil {
		encoded, err = json.Marshal(struct {
			Request []byte
			Saved   *internalflows.DesktopFlowRevision
		}{encoded, saved})
		if err != nil {
			return nil, ownerError(err)
		}
	}
	hash := sha256.Sum256(encoded)
	digest := hex.EncodeToString(hash[:])
	p.mu.Lock()
	active, err := p.resolve(ctx, request.Session)
	if err != nil {
		p.mu.Unlock()
		return nil, ownerError(err)
	}
	if !active.grant.Lease.Control || !p.owner.sessionActorAllowed(ctx, active.grant.Lease.Actor, true) {
		p.mu.Unlock()
		return nil, ownerError(sessions.ErrDesktopAdmission)
	}
	p.mu.Unlock()
	claim, err := p.helper.ClaimFlow(ctx, helperRequest(&desktopv1.ClaimFlowRequest{Lease: helperLease(active.grant), RunId: request.RunId, Digest: digest, Steps: uint32(len(flow.Steps))}, active.token))
	if err != nil {
		return nil, ownerError(err)
	}
	if !claim.Msg.Fresh {
		if err := p.recordDesktopRun(ctx, active, flow, digest, request.RunId, claim.Msg.Record, saved); err != nil {
			return nil, ownerError(err)
		}
		return connect.NewResponse(claim.Msg.Record), nil
	}
	binding := &desktopv1.OwnerObserveRequest{Session: request.Session, ApplicationId: request.ApplicationId, ApplicationRevision: request.ApplicationRevision}
	result, runErr := internalflows.ExecuteDesktopFlowWithID(ctx, p, binding, flow, request.RunId)
	disposition := "incomplete"
	if runErr == nil && result.Disposition == "passed" && !result.Incomplete {
		disposition = "passed"
	}
	finishctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()
	p.mu.Lock()
	current, err := p.resolve(finishctx, request.Session)
	p.mu.Unlock()
	if err != nil {
		return nil, ownerError(err)
	}
	finished, err := p.helper.FinishFlow(finishctx, helperRequest(&desktopv1.FinishFlowRequest{Lease: helperLease(current.grant), RunId: request.RunId, Digest: digest, Disposition: disposition}, current.token))
	if err != nil {
		return nil, ownerError(err)
	}
	if err := p.recordDesktopRun(finishctx, current, flow, digest, request.RunId, finished.Msg, saved); err != nil {
		return nil, ownerError(err)
	}
	return connect.NewResponse(finished.Msg), nil
}

func (p *desktopOwnerRPC) recordDesktopRun(ctx context.Context, active ownerDesktopSession, flow execution.Flow, digest, runID string, record *desktopv1.FlowRecord, saved *internalflows.DesktopFlowRevision) error {
	if record == nil || record.RunId != runID || record.Digest != digest || int(record.Steps) != len(flow.Steps) {
		return sessions.ErrDesktopAdmission
	}
	if record.Disposition == "claimed" {
		return nil // No terminal evidence exists after interrupted execution.
	}
	return p.owner.service.desktopRuns.Put(ctx, internalflows.DesktopRun{
		SavedRevision: saved,
		Scope:         internalflows.DesktopRunScope{Actor: active.grant.Lease.Actor, DeviceID: p.owner.deviceID, Surface: active.grant.Lease.Ref.Surface, DesktopSessionID: active.grant.Lease.Ref.DesktopSessionID, LeaseID: active.grant.Lease.Ref.SessionID, RunID: runID},
		Digest:        digest, Flow: flow, Disposition: record.Disposition, Confirmed: record.Confirmed,
	})
}
