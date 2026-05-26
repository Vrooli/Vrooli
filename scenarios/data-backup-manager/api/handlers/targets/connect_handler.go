package targets

import (
	"context"
	"log"

	"data-backup-manager/internal/sources"
	"data-backup-manager/internal/targets"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	sourcesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/sources"
	targetsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/targets"
)

// Deps wires the seams the Connect targets handler needs.
type Deps struct {
	Service targets.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the targets Connect-RPC handler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) RegisterTarget(ctx context.Context, req *connect.Request[targetsv1.RegisterTargetRequest]) (*connect.Response[targetsv1.RegisterTargetResponse], error) {
	t, err := h.deps.Service.Register(ctx, targets.RegisterInput{
		Owner:      req.Msg.Owner,
		Name:       req.Msg.Name,
		SourceKind: protoToKind(req.Msg.SourceKind),
		Locator:    req.Msg.Locator,
	})
	if err != nil {
		return nil, h.translate("RegisterTarget", err)
	}
	return connect.NewResponse(&targetsv1.RegisterTargetResponse{Target: domainToProto(t)}), nil
}

func (h *connectHandler) DeregisterTarget(ctx context.Context, req *connect.Request[targetsv1.DeregisterTargetRequest]) (*connect.Response[targetsv1.DeregisterTargetResponse], error) {
	removed, err := h.deps.Service.Deregister(ctx, req.Msg.Owner, req.Msg.Name)
	if err != nil {
		return nil, h.translate("DeregisterTarget", err)
	}
	return connect.NewResponse(&targetsv1.DeregisterTargetResponse{Removed: removed}), nil
}

func (h *connectHandler) GetTarget(ctx context.Context, req *connect.Request[targetsv1.GetTargetRequest]) (*connect.Response[targetsv1.GetTargetResponse], error) {
	t, err := h.deps.Service.Get(ctx, req.Msg.Id)
	if err != nil {
		return nil, h.translate("GetTarget", err)
	}
	return connect.NewResponse(&targetsv1.GetTargetResponse{Target: domainToProto(t)}), nil
}

func (h *connectHandler) ListTargets(ctx context.Context, req *connect.Request[targetsv1.ListTargetsRequest]) (*connect.Response[targetsv1.ListTargetsResponse], error) {
	list, err := h.deps.Service.List(ctx, req.Msg.Owner, int(req.Msg.PageSize))
	if err != nil {
		return nil, h.translate("ListTargets", err)
	}
	resp := &targetsv1.ListTargetsResponse{Targets: make([]*targetsv1.Target, 0, len(list))}
	for _, t := range list {
		resp.Targets = append(resp.Targets, domainToProto(t))
	}
	return connect.NewResponse(resp), nil
}

// translate maps a domain error to a Connect error, logging only internal ones.
func (h *connectHandler) translate(op string, err error) error {
	connectErr := targets.ToConnectError(err)
	if connect.CodeOf(connectErr) == connect.CodeInternal {
		h.deps.Logger.Printf("targets.%s: %v", op, err)
	}
	return connectErr
}

// domainToProto converts the internal Target to its wire shape.
func domainToProto(t targets.Target) *targetsv1.Target {
	pt := &targetsv1.Target{
		Id:         t.ID,
		Owner:      t.Owner,
		Name:       t.Name,
		SourceKind: kindToProto(t.SourceKind),
		Locator:    t.Locator,
	}
	if !t.CreatedAt.IsZero() {
		pt.CreatedAt = timestamppb.New(t.CreatedAt)
	}
	if !t.UpdatedAt.IsZero() {
		pt.UpdatedAt = timestamppb.New(t.UpdatedAt)
	}
	return pt
}

// protoToKind / kindToProto translate the proto SourceKind enum to the domain
// vocabulary so domain code never imports the generated enum. An unrecognised
// proto value maps to the empty (invalid) kind, which Service.Register rejects.
func protoToKind(k sourcesv1.SourceKind) sources.SourceKind {
	switch k {
	case sourcesv1.SourceKind_SOURCE_KIND_FILESYSTEM:
		return sources.KindFilesystem
	case sourcesv1.SourceKind_SOURCE_KIND_SQLITE:
		return sources.KindSQLite
	case sourcesv1.SourceKind_SOURCE_KIND_POSTGRES:
		return sources.KindPostgres
	case sourcesv1.SourceKind_SOURCE_KIND_REDIS:
		return sources.KindRedis
	case sourcesv1.SourceKind_SOURCE_KIND_QDRANT:
		return sources.KindQdrant
	case sourcesv1.SourceKind_SOURCE_KIND_OBJECT_STORAGE:
		return sources.KindObjectStorage
	default:
		return ""
	}
}

func kindToProto(k sources.SourceKind) sourcesv1.SourceKind {
	switch k {
	case sources.KindFilesystem:
		return sourcesv1.SourceKind_SOURCE_KIND_FILESYSTEM
	case sources.KindSQLite:
		return sourcesv1.SourceKind_SOURCE_KIND_SQLITE
	case sources.KindPostgres:
		return sourcesv1.SourceKind_SOURCE_KIND_POSTGRES
	case sources.KindRedis:
		return sourcesv1.SourceKind_SOURCE_KIND_REDIS
	case sources.KindQdrant:
		return sourcesv1.SourceKind_SOURCE_KIND_QDRANT
	case sources.KindObjectStorage:
		return sourcesv1.SourceKind_SOURCE_KIND_OBJECT_STORAGE
	default:
		return sourcesv1.SourceKind_SOURCE_KIND_UNSPECIFIED
	}
}
