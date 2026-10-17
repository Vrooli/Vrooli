package flows

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"connectrpc.com/connect"

	desktopv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop"
)

var (
	ErrDesktopAbsent    = errors.New("desktop target absent")
	ErrDesktopAmbiguous = errors.New("desktop target ambiguous")
	ErrDesktopStale     = errors.New("desktop evidence stale or inconsistent")
)

// DesktopResolveCall is bound to an admitted session by the execution owner.
// Saved flows retain selector intent, never this call or helper credentials.
type DesktopResolveCall func(context.Context, *desktopv1.SemanticSelector) (*desktopv1.ResolveResponse, error)

// ResolveDesktopField preserves semantic identity rather than pointer bounds.
func ResolveDesktopField(ctx context.Context, o *desktopv1.ObserveResponse, windowName, fieldName string, resolve DesktopResolveCall) (*desktopv1.TextAction, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if o == nil || o.Semantic == nil || resolve == nil || windowName == "" || fieldName == "" {
		return nil, ErrInvalidRequest
	}
	s := o.Semantic
	if s.ExpiresAt == nil || !s.ExpiresAt.IsValid() || !time.Now().Before(s.ExpiresAt.AsTime()) || s.Revision == "" || o.GeometryRevision == "" {
		return nil, ErrDesktopStale
	}
	window := ""
	for _, e := range s.Elements {
		if e.GetElementId() != "" && e.WindowId == e.ElementId && e.Name == windowName {
			if window != "" {
				return nil, ErrDesktopAmbiguous
			}
			window = e.ElementId
		}
	}
	if window == "" {
		return nil, ErrDesktopAbsent
	}
	r, err := resolve(ctx, &desktopv1.SemanticSelector{ObservationRevision: s.Revision, WindowId: window, Name: fieldName, EditableOnly: true})
	if err != nil {
		return nil, err
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if r == nil || r.ObservationRevision != s.Revision || r.GeometryRevision != o.GeometryRevision || r.ExpiresAt == nil || !r.ExpiresAt.IsValid() || r.ExpiresAt.AsTime().After(s.ExpiresAt.AsTime()) || !time.Now().Before(r.ExpiresAt.AsTime()) {
		return nil, ErrDesktopStale
	}
	switch r.Disposition {
	case desktopv1.ResolveResponse_DISPOSITION_ABSENT:
		if len(r.ElementIds) != 0 {
			return nil, ErrDesktopStale
		}
		return nil, ErrDesktopAbsent
	case desktopv1.ResolveResponse_DISPOSITION_AMBIGUOUS:
		if len(r.ElementIds) < 2 {
			return nil, ErrDesktopStale
		}
		return nil, ErrDesktopAmbiguous
	case desktopv1.ResolveResponse_DISPOSITION_UNIQUE:
		if len(r.ElementIds) != 1 {
			return nil, ErrDesktopStale
		}
	default:
		return nil, ErrDesktopStale
	}
	for _, e := range s.Elements {
		if e.GetElementId() == r.ElementIds[0] && e.WindowId == window && e.Name == fieldName && e.Editable {
			return &desktopv1.TextAction{ElementId: e.ElementId, ObservationRevision: s.Revision}, nil
		}
	}
	return nil, ErrDesktopStale
}

// DesktopStepOwner is bound to the existing admitted owner session. Implementors
// own authentication metadata; this runner neither acquires nor changes leases.
type DesktopStepOwner interface {
	Observe(context.Context, *connect.Request[desktopv1.OwnerObserveRequest]) (*connect.Response[desktopv1.ObserveResponse], error)
	Resolve(context.Context, *connect.Request[desktopv1.OwnerResolveRequest]) (*connect.Response[desktopv1.ResolveResponse], error)
	Act(context.Context, *connect.Request[desktopv1.OwnerActRequest]) (*connect.Response[desktopv1.ActResponse], error)
}

// ExecuteDesktopText executes one semantic text step. Every attempt has a caller
// supplied command ID. Unknown receipts and transport errors are returned as-is;
// the flow executor must stop rather than retry with a new observation/command.
func ExecuteDesktopText(ctx context.Context, owner DesktopStepOwner, observation *desktopv1.OwnerObserveRequest, commandID, windowName, fieldName, text string, position int32) (*desktopv1.ActResponse, error) {
	return executeDesktopText(ctx, owner, observation, commandID, windowName, fieldName, text, position, false)
}

func executeDesktopText(ctx context.Context, owner DesktopStepOwner, observation *desktopv1.OwnerObserveRequest, commandID, windowName, fieldName, text string, position int32, assertion bool) (*desktopv1.ActResponse, error) {
	if owner == nil || observation == nil || observation.Session == nil || observation.ApplicationId == "" || observation.ApplicationRevision == "" || observation.ProcessId != 0 || commandID == "" || len(commandID) > 128 || windowName == "" || fieldName == "" || (!assertion && text == "") || len(text) > 16*1024 || !utf8.ValidString(text) || strings.ContainsRune(text, 0) || position < 0 {
		return nil, ErrInvalidRequest
	}
	captured, err := owner.Observe(ctx, connect.NewRequest(observation))
	if err != nil {
		return nil, err
	}
	if captured == nil {
		return nil, ErrDesktopStale
	}
	target, err := ResolveDesktopField(ctx, captured.Msg, windowName, fieldName, func(ctx context.Context, s *desktopv1.SemanticSelector) (*desktopv1.ResolveResponse, error) {
		response, err := owner.Resolve(ctx, connect.NewRequest(&desktopv1.OwnerResolveRequest{Session: observation.Session, Selector: s}))
		if err != nil {
			return nil, err
		}
		if response == nil {
			return nil, ErrDesktopStale
		}
		return response.Msg, nil
	})
	if err != nil {
		return nil, err
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	target.Text, target.Position = text, position
	action := &desktopv1.Action{Action: &desktopv1.Action_Text{Text: target}}
	if assertion {
		action = &desktopv1.Action{Action: &desktopv1.Action_AssertText{AssertText: &desktopv1.AssertTextAction{ExpectedText: text, ElementId: target.ElementId, ObservationRevision: target.ObservationRevision}}}
	}
	result, err := owner.Act(ctx, connect.NewRequest(&desktopv1.OwnerActRequest{Session: observation.Session, CommandId: commandID, GeometryRevision: captured.Msg.GeometryRevision, Action: action}))
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, ErrDesktopStale
	}
	return result.Msg, nil
}
