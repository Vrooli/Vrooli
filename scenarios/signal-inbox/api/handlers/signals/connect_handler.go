package signals

import (
	"context"
	"errors"
	"log"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/signal-inbox/v1/shared"
	signalsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/signal-inbox/v1/signals"
	internal "signal-inbox/internal/signals"
)

type (
	Deps struct {
		Service internal.Service
		Logger  *log.Logger
	}
	connectHandler struct{ deps Deps }
)

func NewConnectHandler(deps Deps) *connectHandler {
	if deps.Logger == nil {
		deps.Logger = log.Default()
	}
	return &connectHandler{deps: deps}
}

func (h *connectHandler) CaptureSignal(ctx context.Context, req *connect.Request[signalsv1.CaptureSignalRequest]) (*connect.Response[signalsv1.CaptureSignalResponse], error) {
	in := internal.CaptureInput{CaptureNote: req.Msg.CaptureNote, Tags: req.Msg.Tags}
	switch source := req.Msg.Source.(type) {
	case *signalsv1.CaptureSignalRequest_Url:
		in.URL = source.Url
	case *signalsv1.CaptureSignalRequest_Text:
		in.Text = source.Text
	case *signalsv1.CaptureSignalRequest_ImagePayloadRef:
		in.ImagePayloadRef = source.ImagePayloadRef
	}
	result, err := h.deps.Service.Capture(ctx, in)
	if err != nil {
		return nil, toConnectError(err)
	}
	return connect.NewResponse(&signalsv1.CaptureSignalResponse{Signal: domainToProto(result.Signal), Duplicate: result.Duplicate}), nil
}

func (h *connectHandler) GetSignal(ctx context.Context, req *connect.Request[signalsv1.GetSignalRequest]) (*connect.Response[signalsv1.GetSignalResponse], error) {
	signal, err := h.deps.Service.Get(ctx, req.Msg.Id)
	if err != nil {
		return nil, toConnectError(err)
	}
	return connect.NewResponse(&signalsv1.GetSignalResponse{Signal: domainToProto(signal)}), nil
}

func (h *connectHandler) ListSignals(ctx context.Context, req *connect.Request[signalsv1.ListSignalsRequest]) (*connect.Response[signalsv1.ListSignalsResponse], error) {
	signals, err := h.deps.Service.List(ctx, int(req.Msg.PageSize))
	if err != nil {
		return nil, toConnectError(err)
	}
	resp := &signalsv1.ListSignalsResponse{Signals: make([]*sharedv1.Signal, 0, len(signals))}
	for _, signal := range signals {
		resp.Signals = append(resp.Signals, domainToProto(signal))
	}
	return connect.NewResponse(resp), nil
}

func domainToProto(signal internal.Signal) *sharedv1.Signal {
	return &sharedv1.Signal{Id: signal.ID, SourceKind: sourceKindToProto(signal.SourceKind), SourceIdentity: signal.SourceIdentity, SourceUrl: signal.SourceURL, CapturedAt: timestamppb.New(signal.CapturedAt), RawPayloadRef: signal.RawPayloadRef, ExtractedContent: signal.ExtractedContent, ContentHash: signal.ContentHash, NeedsAttention: signal.NeedsAttention, CaptureNote: signal.CaptureNote, Tags: signal.Tags}
}

func sourceKindToProto(kind internal.SourceKind) sharedv1.SourceKind {
	switch kind {
	case internal.SourceKindURL:
		return sharedv1.SourceKind_SOURCE_KIND_URL
	case internal.SourceKindText:
		return sharedv1.SourceKind_SOURCE_KIND_TEXT
	case internal.SourceKindImage:
		return sharedv1.SourceKind_SOURCE_KIND_IMAGE
	default:
		return sharedv1.SourceKind_SOURCE_KIND_UNSPECIFIED
	}
}

func toConnectError(err error) error {
	var invalid internal.ErrInvalidSignal
	if errors.As(err, &invalid) {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	var missing internal.ErrSignalNotFound
	if errors.As(err, &missing) {
		return connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}
