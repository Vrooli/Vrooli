package manifest

import (
	"context"
	"log"

	manifest "development-toolchain-validator/internal/manifest"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	manifestv1 "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/manifest"
)

// Deps wires the seams the Connect manifest handler needs.
type Deps struct {
	Service manifest.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the Connect handler for ManifestService.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) ListManifests(ctx context.Context, _ *connect.Request[manifestv1.ListManifestsRequest]) (*connect.Response[manifestv1.ListManifestsResponse], error) {
	rows, err := h.deps.Service.List(ctx)
	if err != nil {
		h.deps.Logger.Printf("manifest.ListManifests: %v", err)
		return nil, manifest.ToConnectError(err)
	}
	resp := &manifestv1.ListManifestsResponse{Manifests: make([]*manifestv1.Manifest, 0, len(rows))}
	for _, m := range rows {
		resp.Manifests = append(resp.Manifests, domainToProto(m))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) GetManifest(ctx context.Context, req *connect.Request[manifestv1.GetManifestRequest]) (*connect.Response[manifestv1.GetManifestResponse], error) {
	m, err := h.deps.Service.Get(ctx, req.Msg.SkillId, req.Msg.GoldenSlug)
	if err != nil {
		connectErr := manifest.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("manifest.GetManifest(%q,%q): %v", req.Msg.SkillId, req.Msg.GoldenSlug, err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&manifestv1.GetManifestResponse{Manifest: domainToProto(m)}), nil
}

func (h *connectHandler) UpsertManifest(ctx context.Context, req *connect.Request[manifestv1.UpsertManifestRequest]) (*connect.Response[manifestv1.UpsertManifestResponse], error) {
	if req.Msg.Manifest == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, manifest.ErrInvalidManifest{Field: "manifest", Reason: "required"})
	}
	in := protoToUpsertInput(req.Msg.Manifest)
	m, err := h.deps.Service.Upsert(ctx, in)
	if err != nil {
		connectErr := manifest.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("manifest.UpsertManifest(%q,%q): %v", in.SkillID, in.GoldenSlug, err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&manifestv1.UpsertManifestResponse{Manifest: domainToProto(m)}), nil
}

func (h *connectHandler) ClearStale(ctx context.Context, req *connect.Request[manifestv1.ClearStaleRequest]) (*connect.Response[manifestv1.ClearStaleResponse], error) {
	at, err := h.deps.Service.ClearStale(ctx, req.Msg.SkillId, req.Msg.GoldenSlug)
	if err != nil {
		connectErr := manifest.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("manifest.ClearStale(%q,%q): %v", req.Msg.SkillId, req.Msg.GoldenSlug, err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&manifestv1.ClearStaleResponse{ClearedAt: timestamppb.New(at.UTC())}), nil
}
