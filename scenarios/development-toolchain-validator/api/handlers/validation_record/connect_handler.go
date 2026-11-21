package validation_record

import (
	"context"
	"log"

	vr "development-toolchain-validator/internal/validation_record"

	"connectrpc.com/connect"

	vrv1 "github.com/vrooli/vrooli/packages/proto/gen/go/development-toolchain-validator/v1/validation_record"
)

// Deps wires the seams the Connect validation_record handler needs.
type Deps struct {
	Service vr.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the Connect handler for the service.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) ListRecords(ctx context.Context, req *connect.Request[vrv1.ListRecordsRequest]) (*connect.Response[vrv1.ListRecordsResponse], error) {
	res, err := h.deps.Service.List(ctx, vr.ListFilter{
		GoldenSlug: req.Msg.GoldenSlug,
		SubjectID:  req.Msg.SubjectId,
		TupleKind:  tupleKindProtoToDomain(req.Msg.TupleKind),
	}, int(req.Msg.PageSize), req.Msg.PageToken)
	if err != nil {
		connectErr := vr.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("validation_record.ListRecords: %v", err)
		}
		return nil, connectErr
	}
	resp := &vrv1.ListRecordsResponse{
		Records:       make([]*vrv1.ValidationRecord, 0, len(res.Records)),
		NextPageToken: res.NextPageToken,
	}
	for _, r := range res.Records {
		resp.Records = append(resp.Records, domainToProto(r))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) GetRecord(ctx context.Context, req *connect.Request[vrv1.GetRecordRequest]) (*connect.Response[vrv1.GetRecordResponse], error) {
	r, err := h.deps.Service.Get(ctx, req.Msg.Id)
	if err != nil {
		connectErr := vr.ToConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("validation_record.GetRecord(%q): %v", req.Msg.Id, err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&vrv1.GetRecordResponse{Record: domainToProto(r)}), nil
}
