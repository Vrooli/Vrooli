package validation_run

import (
	"context"
	"log"

	vrun "development-toolchain-validator/internal/validation_run"

	"connectrpc.com/connect"

	vrunv1 "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/validation_run"
)

// Deps wires the seams the Connect validation_run handler needs.
type Deps struct {
	Service vrun.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the Connect handler for ValidationRunService.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) Start(ctx context.Context, req *connect.Request[vrunv1.StartRequest]) (*connect.Response[vrunv1.StartResponse], error) {
	r, err := h.deps.Service.Start(ctx, vrun.StartInput{
		TupleKind:  tupleKindProtoToDomain(req.Msg.TupleKind),
		SubjectID:  req.Msg.SubjectId,
		GoldenSlug: req.Msg.GoldenSlug,
		Force:      req.Msg.Force,
	})
	if err != nil {
		connectErr := vrun.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("validation_run.Start: %v", err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&vrunv1.StartResponse{Run: domainToProto(r)}), nil
}

func (h *connectHandler) Get(ctx context.Context, req *connect.Request[vrunv1.GetRequest]) (*connect.Response[vrunv1.GetResponse], error) {
	r, err := h.deps.Service.Get(ctx, req.Msg.Id)
	if err != nil {
		connectErr := vrun.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("validation_run.Get(%q): %v", req.Msg.Id, err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&vrunv1.GetResponse{Run: domainToProto(r)}), nil
}

func (h *connectHandler) ListActive(ctx context.Context, _ *connect.Request[vrunv1.ListActiveRequest]) (*connect.Response[vrunv1.ListActiveResponse], error) {
	rows, err := h.deps.Service.ListActive(ctx)
	if err != nil {
		h.deps.Logger.Printf("validation_run.ListActive: %v", err)
		return nil, vrun.ToConnectError(err)
	}
	resp := &vrunv1.ListActiveResponse{Runs: make([]*vrunv1.ValidationRun, 0, len(rows))}
	for _, r := range rows {
		resp.Runs = append(resp.Runs, domainToProto(r))
	}
	return connect.NewResponse(resp), nil
}
