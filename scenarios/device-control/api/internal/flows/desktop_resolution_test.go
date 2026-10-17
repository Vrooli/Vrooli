package flows

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop/desktopv1connect"

	"github.com/stretchr/testify/require"
	desktopv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestDesktopResolutionPreservesIdentity(t *testing.T) {
	expiry := timestamppb.New(time.Now().Add(4 * time.Second))
	o := &desktopv1.ObserveResponse{GeometryRevision: "g", Semantic: &desktopv1.SemanticObservation{Revision: "r", ExpiresAt: expiry, Elements: []*desktopv1.SemanticElement{{ElementId: "w", WindowId: "w", Name: "Editor"}, {ElementId: "f", WindowId: "w", Name: "Text", Editable: true}}}}
	calls := 0
	r := &desktopv1.ResolveResponse{Disposition: desktopv1.ResolveResponse_DISPOSITION_UNIQUE, ObservationRevision: "r", GeometryRevision: "g", ExpiresAt: expiry, ElementIds: []string{"f"}}
	resolve := func(_ context.Context, s *desktopv1.SemanticSelector) (*desktopv1.ResolveResponse, error) {
		calls++
		require.Equal(t, "w", s.WindowId)
		require.Equal(t, "r", s.ObservationRevision)
		require.True(t, s.EditableOnly)
		return r, nil
	}
	target, err := ResolveDesktopField(context.Background(), o, "Editor", "Text", resolve)
	require.NoError(t, err)
	require.Equal(t, "f", target.ElementId)
	o.Semantic.Elements = append(o.Semantic.Elements, &desktopv1.SemanticElement{ElementId: "w2", WindowId: "w2", Name: "Editor"})
	_, err = ResolveDesktopField(context.Background(), o, "Editor", "Text", resolve)
	require.ErrorIs(t, err, ErrDesktopAmbiguous)
	require.Equal(t, 1, calls)
	o.Semantic.Elements = o.Semantic.Elements[:2]
	r.Disposition = desktopv1.ResolveResponse_DISPOSITION_AMBIGUOUS
	r.ElementIds = []string{"f", "other"}
	_, err = ResolveDesktopField(context.Background(), o, "Editor", "Text", resolve)
	require.ErrorIs(t, err, ErrDesktopAmbiguous)
	r.Disposition = desktopv1.ResolveResponse_DISPOSITION_UNIQUE
	r.ElementIds = []string{"foreign"}
	_, err = ResolveDesktopField(context.Background(), o, "Editor", "Text", resolve)
	require.ErrorIs(t, err, ErrDesktopStale)
	r.ElementIds = []string{"f"}
	r.GeometryRevision = "changed"
	_, err = ResolveDesktopField(context.Background(), o, "Editor", "Text", resolve)
	require.ErrorIs(t, err, ErrDesktopStale)
}

type desktopStepFixture struct {
	desktopv1connect.UnimplementedDesktopOwnerServiceHandler
	acts      int
	ambiguous bool
}

func (f *desktopStepFixture) Observe(context.Context, *connect.Request[desktopv1.OwnerObserveRequest]) (*connect.Response[desktopv1.ObserveResponse], error) {
	return connect.NewResponse(&desktopv1.ObserveResponse{GeometryRevision: "g", Semantic: &desktopv1.SemanticObservation{Revision: "r", ExpiresAt: timestamppb.New(time.Now().Add(4 * time.Second)), Elements: []*desktopv1.SemanticElement{{ElementId: "w", WindowId: "w", Name: "Editor"}, {ElementId: "f", WindowId: "w", Name: "Text", Editable: true}}}}), nil
}

func (f *desktopStepFixture) Resolve(context.Context, *connect.Request[desktopv1.OwnerResolveRequest]) (*connect.Response[desktopv1.ResolveResponse], error) {
	r := &desktopv1.ResolveResponse{Disposition: desktopv1.ResolveResponse_DISPOSITION_UNIQUE, ObservationRevision: "r", GeometryRevision: "g", ExpiresAt: timestamppb.New(time.Now().Add(time.Second)), ElementIds: []string{"f"}}
	if f.ambiguous {
		r.Disposition = desktopv1.ResolveResponse_DISPOSITION_AMBIGUOUS
		r.ElementIds = append(r.ElementIds, "other")
	}
	return connect.NewResponse(r), nil
}

func (f *desktopStepFixture) Act(_ context.Context, r *connect.Request[desktopv1.OwnerActRequest]) (*connect.Response[desktopv1.ActResponse], error) {
	f.acts++
	if r.Msg.CommandId != "command" || r.Msg.Action.GetText().ElementId != "f" || r.Msg.Action.GetText().Text != "日本語 🧪" {
		return nil, ErrInvalidRequest
	}
	return connect.NewResponse(&desktopv1.ActResponse{Receipt: &desktopv1.Receipt{Outcome: "outcome_unknown"}}), nil
}

func TestDesktopTextStepStopsOnAmbiguityAndPreservesUnknown(t *testing.T) {
	f := &desktopStepFixture{ambiguous: true}
	request := &desktopv1.OwnerObserveRequest{Session: &commonv1.SessionRef{SessionId: "lease"}, ApplicationId: "app", ApplicationRevision: "catalog"}
	_, err := ExecuteDesktopText(context.Background(), f, request, "command", "Editor", "Text", "日本語 🧪", 0)
	require.ErrorIs(t, err, ErrDesktopAmbiguous)
	require.Zero(t, f.acts)
	f.ambiguous = false
	result, err := ExecuteDesktopText(context.Background(), f, request, "command", "Editor", "Text", "日本語 🧪", 0)
	require.NoError(t, err)
	require.Equal(t, "outcome_unknown", result.Receipt.Outcome)
	require.Equal(t, 1, f.acts)
}
