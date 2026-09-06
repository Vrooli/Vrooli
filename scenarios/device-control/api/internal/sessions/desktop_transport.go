package sessions

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"image/png"
	"math"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/targetmodel"
	desktopv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop"
	"github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop/desktopv1connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type DesktopUnixHelper struct {
	controller *DesktopController
	authority  *SignedDesktopAuthority
}
type desktopConnectionKey struct{}

func NewDesktopUnixHelper(c *DesktopController, a *SignedDesktopAuthority) (*DesktopUnixHelper, error) {
	if c == nil || a == nil || c.authority != a {
		return nil, ErrDesktopAdmission
	}
	return &DesktopUnixHelper{controller: c, authority: a}, nil
}

// Serve accepts only an existing Unix listener from the lifecycle owner, which
// owns the socket directory, permissions and OS-session bootstrap. No TCP or
// browser-facing handler is exported. Native/helper activation precedes Serve.
func (h *DesktopUnixHelper) Serve(ctx context.Context, listener *net.UnixListener) error {
	if listener == nil {
		return ErrDesktopAdmission
	}
	child, cancel := context.WithCancel(ctx)
	defer cancel()
	path, handler := desktopv1connect.NewDesktopHelperServiceHandler(&desktopRPC{controller: h.controller}, connect.WithReadMaxBytes(128*1024))
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	var connections atomic.Int32
	server := &http.Server{
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 15 * time.Second, MaxHeaderBytes: 16 * 1024,
		ConnContext: func(ctx context.Context, c net.Conn) context.Context {
			return context.WithValue(ctx, desktopConnectionKey{}, c)
		},
		ConnState: func(c net.Conn, state http.ConnState) {
			switch state {
			case http.StateNew:
				if connections.Add(1) > 32 {
					c.Close()
				}
			case http.StateClosed, http.StateHijacked:
				connections.Add(-1)
			}
		},
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			conn, ok := r.Context().Value(desktopConnectionKey{}).(*net.UnixConn)
			cleanupRead := r.URL.Path == desktopv1connect.DesktopHelperServiceReadCleanupProcedure
			scheme := "DesktopGrant "
			if cleanupRead {
				scheme = "DesktopCleanup "
			}
			token, hasScheme := strings.CutPrefix(r.Header.Get("Authorization"), scheme)
			if !ok || !hasScheme {
				http.Error(w, "desktop admission refused", http.StatusUnauthorized)
				return
			}
			var requestCtx context.Context
			var err error
			if cleanupRead {
				requestCtx, err = h.authority.AuthenticateCleanupPeer(r.Context(), conn, token)
			} else {
				requestCtx, err = h.authority.AuthenticatePeer(r.Context(), conn, token)
			}
			if err != nil {
				http.Error(w, "desktop admission refused", http.StatusUnauthorized)
				return
			}
			mux.ServeHTTP(w, r.WithContext(requestCtx))
		}),
	}
	serving, maintaining := make(chan error, 1), make(chan error, 1)
	go func() { serving <- server.Serve(listener) }()
	go func() { maintaining <- h.controller.MaintainLease(child) }()
	var serveErr, maintainErr error
	gotServe, gotMaintain := false, false
	select {
	case <-ctx.Done():
	case serveErr = <-serving:
		gotServe = true
	case maintainErr = <-maintaining:
		gotMaintain = true
	}
	cancel()
	_ = server.Close()
	if !gotServe {
		serveErr = <-serving
	}
	if !gotMaintain {
		maintainErr = <-maintaining
	}
	if errors.Is(serveErr, http.ErrServerClosed) {
		serveErr = nil
	}
	return errors.Join(serveErr, maintainErr)
}

type desktopRPC struct{ controller *DesktopController }

func leaseFromWire(w *desktopv1.Lease) (DesktopLease, error) {
	if w == nil || w.ExpiresAt == nil || w.ExpiresAt.CheckValid() != nil {
		return DesktopLease{}, ErrDesktopAdmission
	}
	ref, err := targetmodel.SessionRefFromProto(w.Ref)
	if err != nil {
		return DesktopLease{}, ErrDesktopAdmission
	}
	return DesktopLease{Ref: ref, Actor: w.Actor, HelperID: w.HelperId, Epoch: w.Epoch, ExpiresAt: w.ExpiresAt.AsTime(), Control: w.Control}, nil
}

func leaseToWire(l DesktopLease) *desktopv1.Lease {
	return &desktopv1.Lease{Ref: l.Ref.Proto(), Actor: l.Actor, HelperId: l.HelperID, Epoch: l.Epoch, ExpiresAt: timestamppb.New(l.ExpiresAt), Control: l.Control}
}

func desktopRPCError(err error) error {
	if errors.Is(err, ErrDesktopAdmission) {
		return connect.NewError(connect.CodePermissionDenied, ErrDesktopAdmission)
	}
	return connect.NewError(connect.CodeUnavailable, errors.New("desktop operation incomplete; do not blindly retry input"))
}

func (s *desktopRPC) Open(ctx context.Context, r *connect.Request[desktopv1.OpenRequest]) (*connect.Response[desktopv1.OpenResponse], error) {
	lease, err := leaseFromWire(r.Msg.Lease)
	if err != nil {
		return nil, desktopRPCError(err)
	}
	lease, err = s.controller.Open(ctx, lease, r.Msg.ExpectedEpoch, r.Msg.Takeover)
	if err != nil {
		return nil, desktopRPCError(err)
	}
	return connect.NewResponse(&desktopv1.OpenResponse{Lease: leaseToWire(lease)}), nil
}

func (s *desktopRPC) Stop(ctx context.Context, r *connect.Request[desktopv1.StopRequest]) (*connect.Response[desktopv1.StopResponse], error) {
	lease, err := leaseFromWire(r.Msg.Lease)
	if err != nil {
		return nil, desktopRPCError(err)
	}
	if err = s.controller.Stop(ctx, lease); err != nil {
		return nil, desktopRPCError(err)
	}
	return connect.NewResponse(&desktopv1.StopResponse{}), nil
}

func validDesktopAction(action *desktopv1.Action) bool {
	if action == nil {
		return false
	}
	switch a := action.Action.(type) {
	case *desktopv1.Action_Pointer:
		p := a.Pointer
		return p != nil && p.Kind >= desktopv1.PointerAction_KIND_MOVE && p.Kind <= desktopv1.PointerAction_KIND_CLICK && p.DisplayId != "" && len(p.DisplayId) <= 256 && !math.IsNaN(p.X) && !math.IsNaN(p.Y) && !math.IsInf(p.X, 0) && !math.IsInf(p.Y, 0) && p.Button >= desktopv1.PointerAction_BUTTON_UNSPECIFIED && p.Button <= desktopv1.PointerAction_BUTTON_MIDDLE && (p.Kind == desktopv1.PointerAction_KIND_MOVE || p.Button != desktopv1.PointerAction_BUTTON_UNSPECIFIED)
	case *desktopv1.Action_Wheel:
		w := a.Wheel
		if w == nil {
			return false
		}
		h, v := int64(w.HorizontalTicks), int64(w.VerticalTicks)
		if h < 0 {
			h = -h
		}
		if v < 0 {
			v = -v
		}
		return h+v > 0 && h+v <= 20 && w.DisplayId != "" && len(w.DisplayId) <= 256 && !math.IsNaN(w.X) && !math.IsNaN(w.Y) && !math.IsInf(w.X, 0) && !math.IsInf(w.Y, 0)
	case *desktopv1.Action_Key:
		return a.Key != nil && a.Key.Kind >= desktopv1.KeyAction_KIND_DOWN && a.Key.Kind <= desktopv1.KeyAction_KIND_PRESS && a.Key.Key != "" && len(a.Key.Key) <= 128
	case *desktopv1.Action_Text:
		return a.Text != nil && a.Text.Text != "" && len(a.Text.Text) <= 16*1024 && a.Text.ElementId != "" && len(a.Text.ElementId) <= 128 && a.Text.ObservationRevision != "" && len(a.Text.ObservationRevision) <= 128 && a.Text.Position >= 0
	case *desktopv1.Action_AssertText:
		return a.AssertText != nil && len(a.AssertText.ExpectedText) <= 16*1024 && a.AssertText.ElementId != "" && len(a.AssertText.ElementId) <= 128 && a.AssertText.ObservationRevision != "" && len(a.AssertText.ObservationRevision) <= 128
	default:
		return false
	}
}

func (s *desktopRPC) Act(ctx context.Context, r *connect.Request[desktopv1.ActRequest]) (*connect.Response[desktopv1.ActResponse], error) {
	lease, err := leaseFromWire(r.Msg.Lease)
	if err != nil {
		return nil, desktopRPCError(err)
	}
	if !validDesktopAction(r.Msg.Action) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid desktop action"))
	}
	payload, err := protojson.Marshal(r.Msg.Action)
	if err != nil {
		return nil, desktopRPCError(err)
	}
	receipt, err := s.controller.Act(ctx, DesktopCommand{Lease: lease, ID: r.Msg.CommandId, GeometryRevision: r.Msg.GeometryRevision, Payload: payload})
	if err != nil {
		return nil, desktopRPCError(err)
	}
	return connect.NewResponse(&desktopv1.ActResponse{Receipt: &desktopv1.Receipt{CommandId: receipt.CommandID, Digest: receipt.Digest, Outcome: receipt.Outcome}}), nil
}

// Limit encoded capture memory independently of the native pixel bound.
type desktopPNGBuffer struct{ bytes.Buffer }

func (b *desktopPNGBuffer) Write(p []byte) (int, error) {
	if len(p) > 32*1024*1024-b.Len() {
		return 0, errors.New("capture exceeds encoded limit")
	}
	return b.Buffer.Write(p)
}

func (s *desktopRPC) Observe(ctx context.Context, r *connect.Request[desktopv1.ObserveRequest]) (*connect.Response[desktopv1.ObserveResponse], error) {
	lease, err := leaseFromWire(r.Msg.Lease)
	if err != nil {
		return nil, desktopRPCError(err)
	}
	var snapshot DesktopObservation
	if r.Msg.ApplicationId != "" || r.Msg.ApplicationRevision != "" {
		if r.Msg.ProcessId != 0 {
			return nil, desktopRPCError(ErrDesktopAdmission)
		}
		snapshot, err = s.controller.ObserveApplication(ctx, lease, r.Msg.ApplicationId, r.Msg.ApplicationRevision)
	} else {
		snapshot, err = s.controller.ObserveProcess(ctx, lease, r.Msg.ProcessId)
	}
	if err != nil {
		return nil, desktopRPCError(err)
	}
	var encoded desktopPNGBuffer
	encoder := png.Encoder{CompressionLevel: png.BestSpeed}
	if err = encoder.Encode(&encoded, snapshot.Image); err != nil {
		return nil, desktopRPCError(err)
	}
	if s.controller.authority.Authorize(ctx, lease, "observe") != nil {
		return nil, desktopRPCError(ErrDesktopAdmission)
	}
	var semantic *desktopv1.SemanticObservation
	if snapshot.Semantic != nil {
		source := snapshot.Semantic
		if !s.controller.now().Before(source.ExpiresAt) {
			return nil, desktopRPCError(ErrDesktopAdmission)
		}
		semantic = &desktopv1.SemanticObservation{Revision: source.Revision, ExpiresAt: timestamppb.New(source.ExpiresAt), ProcessId: source.ProcessID}
		for _, element := range source.Elements {
			semantic.Elements = append(semantic.Elements, &desktopv1.SemanticElement{ElementId: element.ID, Name: element.Name, Editable: element.Editable, ParentId: element.ParentID, WindowId: element.WindowID, Role: element.Role})
		}
	}
	return connect.NewResponse(&desktopv1.ObserveResponse{Semantic: semantic, Png: encoded.Bytes(), DisplayId: snapshot.DisplayID, GeometryRevision: snapshot.GeometryRevision, CapturedAt: timestamppb.New(snapshot.CapturedAt), Width: uint32(snapshot.Image.Bounds().Dx()), Height: uint32(snapshot.Image.Bounds().Dy())}), nil
}

func (s *desktopRPC) Applications(ctx context.Context, r *connect.Request[desktopv1.ApplicationsRequest]) (*connect.Response[desktopv1.ApplicationsResponse], error) {
	lease, err := leaseFromWire(r.Msg.Lease)
	if err != nil {
		return nil, desktopRPCError(err)
	}
	catalog, err := s.controller.Applications(ctx, lease)
	if err != nil {
		return nil, desktopRPCError(err)
	}
	out := &desktopv1.ApplicationsResponse{Revision: catalog.Revision, ExpiresAt: timestamppb.New(catalog.ExpiresAt)}
	for _, app := range catalog.Applications {
		out.Applications = append(out.Applications, &desktopv1.Application{ApplicationId: app.ID, Name: app.Name, ProcessId: app.ProcessID})
	}
	return connect.NewResponse(out), nil
}

func (s *desktopRPC) Resolve(ctx context.Context, r *connect.Request[desktopv1.ResolveRequest]) (*connect.Response[desktopv1.ResolveResponse], error) {
	lease, err := leaseFromWire(r.Msg.Lease)
	if err != nil {
		return nil, desktopRPCError(err)
	}
	selector := r.Msg.Selector
	if selector == nil {
		return nil, desktopRPCError(ErrDesktopAdmission)
	}
	result, err := s.controller.Resolve(ctx, lease, DesktopSelector{Revision: selector.ObservationRevision, WindowID: selector.WindowId, Name: selector.Name, EditableOnly: selector.EditableOnly})
	if err != nil {
		return nil, desktopRPCError(err)
	}
	kind := desktopv1.ResolveResponse_DISPOSITION_UNSPECIFIED
	switch result.Disposition {
	case "absent":
		kind = desktopv1.ResolveResponse_DISPOSITION_ABSENT
	case "unique":
		kind = desktopv1.ResolveResponse_DISPOSITION_UNIQUE
	case "ambiguous":
		kind = desktopv1.ResolveResponse_DISPOSITION_AMBIGUOUS
	}
	return connect.NewResponse(&desktopv1.ResolveResponse{Disposition: kind, ObservationRevision: result.Revision, GeometryRevision: result.GeometryRevision, ExpiresAt: timestamppb.New(result.ExpiresAt), ElementIds: result.ElementIDs}), nil
}

func flowRecordToWire(record DesktopFlowClaim) *desktopv1.FlowRecord {
	out := &desktopv1.FlowRecord{RunId: record.RunID, Digest: record.Digest, Steps: record.Steps, Disposition: record.Disposition, Confirmed: record.Confirmed, ClaimedAt: timestamppb.New(record.ClaimedAt)}
	if !record.FinishedAt.IsZero() {
		out.FinishedAt = timestamppb.New(record.FinishedAt)
	}
	return out
}

func (s *desktopRPC) ClaimFlow(ctx context.Context, r *connect.Request[desktopv1.ClaimFlowRequest]) (*connect.Response[desktopv1.ClaimFlowResponse], error) {
	lease, err := leaseFromWire(r.Msg.Lease)
	if err != nil {
		return nil, desktopRPCError(err)
	}
	record, fresh, err := s.controller.ClaimFlow(ctx, lease, r.Msg.RunId, r.Msg.Digest, r.Msg.Steps)
	if err != nil {
		return nil, desktopRPCError(err)
	}
	return connect.NewResponse(&desktopv1.ClaimFlowResponse{Record: flowRecordToWire(record), Fresh: fresh}), nil
}

func (s *desktopRPC) FinishFlow(ctx context.Context, r *connect.Request[desktopv1.FinishFlowRequest]) (*connect.Response[desktopv1.FlowRecord], error) {
	lease, err := leaseFromWire(r.Msg.Lease)
	if err != nil {
		return nil, desktopRPCError(err)
	}
	record, err := s.controller.FinishFlow(ctx, lease, r.Msg.RunId, r.Msg.Digest, r.Msg.Disposition)
	if err != nil {
		return nil, desktopRPCError(err)
	}
	return connect.NewResponse(flowRecordToWire(record)), nil
}

func (s *desktopRPC) ReadCleanup(ctx context.Context, r *connect.Request[desktopv1.StopRequest]) (*connect.Response[desktopv1.CleanupResponse], error) {
	lease, err := leaseFromWire(r.Msg.Lease)
	if err != nil {
		return nil, desktopRPCError(err)
	}
	receipt, err := s.controller.ReadCleanup(ctx, lease)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("cleanup evidence unavailable"))
	}
	if err != nil {
		return nil, desktopRPCError(err)
	}
	return connect.NewResponse(&desktopv1.CleanupResponse{Lease: leaseToWire(receipt.Lease), Released: receipt.Released, ObservedAt: timestamppb.New(receipt.ObservedAt)}), nil
}

func activationReferenceWire(ref DesktopActivationReference) *desktopv1.ActivationReference {
	return &desktopv1.ActivationReference{HasImage: ref.HasImage, SourceBounds: &desktopv1.DesktopBounds{X: ref.SourceBounds.X, Y: ref.SourceBounds.Y, Width: ref.SourceBounds.Width, Height: ref.SourceBounds.Height}, ContextId: ref.ID, DisplayId: ref.DisplayID, GeometryRevision: ref.GeometryRevision, PointerX: ref.PointerX, PointerY: ref.PointerY, CapturedAt: timestamppb.New(ref.CapturedAt), ExpiresAt: timestamppb.New(ref.ExpiresAt)}
}

func (s *desktopRPC) CaptureActivation(ctx context.Context, r *connect.Request[desktopv1.StopRequest]) (*connect.Response[desktopv1.ActivationReference], error) {
	lease, err := leaseFromWire(r.Msg.Lease)
	if err != nil {
		return nil, desktopRPCError(err)
	}
	ref, err := s.controller.CaptureActivationReference(ctx, lease)
	if err != nil {
		return nil, desktopRPCError(err)
	}
	return connect.NewResponse(activationReferenceWire(ref)), nil
}

func (s *desktopRPC) ReadActivation(ctx context.Context, r *connect.Request[desktopv1.ReadActivationRequest]) (*connect.Response[desktopv1.ActivationReference], error) {
	lease, err := leaseFromWire(r.Msg.Lease)
	if err != nil {
		return nil, desktopRPCError(err)
	}
	ref, err := s.controller.ReadActivation(ctx, lease, r.Msg.ContextId)
	if err != nil {
		return nil, desktopRPCError(err)
	}
	return connect.NewResponse(activationReferenceWire(ref)), nil
}

func (s *desktopRPC) CaptureCompanionActivation(ctx context.Context, r *connect.Request[desktopv1.CompanionActivationRequest]) (*connect.Response[desktopv1.ActivationReference], error) {
	lease, err := leaseFromWire(r.Msg.Lease)
	if err != nil {
		return nil, desktopRPCError(err)
	}
	var ref DesktopActivationReference
	if r.Msg.IncludeImage {
		ref, err = s.controller.CaptureCompanionActivationImage(ctx, lease, r.Msg.CompanionWindow, r.Msg.CompanionPid)
	} else {
		ref, err = s.controller.CaptureCompanionActivation(ctx, lease, r.Msg.CompanionWindow, r.Msg.CompanionPid)
	}
	if err != nil {
		return nil, desktopRPCError(err)
	}
	return connect.NewResponse(activationReferenceWire(ref)), nil
}

func (s *desktopRPC) DeleteActivation(ctx context.Context, r *connect.Request[desktopv1.ReadActivationRequest]) (*connect.Response[desktopv1.StopResponse], error) {
	lease, err := leaseFromWire(r.Msg.Lease)
	if err != nil {
		return nil, desktopRPCError(err)
	}
	if err = s.controller.DeleteActivation(ctx, lease, r.Msg.ContextId); err != nil {
		return nil, desktopRPCError(err)
	}
	return connect.NewResponse(&desktopv1.StopResponse{}), nil
}

func (s *desktopRPC) ReadActivationImage(ctx context.Context, r *connect.Request[desktopv1.ReadActivationRequest]) (*connect.Response[desktopv1.ActivationImage], error) {
	lease, err := leaseFromWire(r.Msg.Lease)
	if err != nil {
		return nil, desktopRPCError(err)
	}
	ref, pixels, err := s.controller.ReadActivationImage(ctx, lease, r.Msg.ContextId)
	if err != nil {
		return nil, desktopRPCError(err)
	}
	var encoded desktopPNGBuffer
	encoder := png.Encoder{CompressionLevel: png.BestSpeed}
	if err = encoder.Encode(&encoded, pixels); err != nil {
		return nil, desktopRPCError(err)
	}
	// Encoding may outlast expiry or concurrent deletion. Confirm the exact frozen
	// reference again before releasing pixels; this does not capture or renew.
	current, err := s.controller.ReadActivation(ctx, lease, r.Msg.ContextId)
	if err != nil || current != ref || ctx.Err() != nil {
		return nil, desktopRPCError(ErrDesktopAdmission)
	}
	return connect.NewResponse(&desktopv1.ActivationImage{Reference: activationReferenceWire(ref), Png: encoded.Bytes()}), nil
}
