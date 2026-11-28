package conversations

import (
	"context"
	"time"

	"connectrpc.com/connect"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"

	"notification-hub/internal/hub"
)

// CheckAsk is a dry run of Ask: the packet check and the notification the
// operator would get, with nothing stored. It admits the same callers as Ask,
// because it reads the blobs and the open ask that an Ask would.
func (h *handler) CheckAsk(ctx context.Context, req *connect.Request[v1.CheckAskRequest]) (*connect.Response[v1.CheckAskResponse], error) {
	producer, err := h.directProducer(ctx, req.Header())
	if err != nil {
		return nil, err
	}
	spec, err := askSpecFromProto(req.Msg.GetAsk(), producer, time.Now())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	check, err := h.service.CheckAsk(ctx, spec, hub.CheckInput{Contents: req.Msg.GetFileContents()})
	if err != nil {
		return nil, connectError(err)
	}
	return connect.NewResponse(&v1.CheckAskResponse{
		Warnings: warningsToProto(check.Warnings), PushTitle: check.PushTitle, PushBody: check.PushBody, RevisesAskId: check.RevisesAskID,
		Sender: check.Sender, SenderSource: string(check.SenderSource),
	}), nil
}
