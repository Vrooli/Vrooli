// Package transfer is the HTTP/Connect transport edge for the transfer domain.
// It is intentionally thin: resolve the caller's TRUSTED device (device-token
// auth), decode the request, call internal/transfer.Service, translate the
// result and any typed error. All policy (validation, retention, quota, ACL)
// lives in the service; all persistence in the repository; all bytes in the
// blob store via the REST upload/download handlers.
package transfer

import (
	"context"
	"log"

	"device-sync-hub/internal/deviceauth"
	internaltransfer "device-sync-hub/internal/transfer"

	"connectrpc.com/connect"

	transferv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-sync-hub/v1/transfer"
)

// Deps wires the seams the Connect transfer handler needs.
type Deps struct {
	Service internaltransfer.Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the Connect handler for the transfer service.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) CreateTextItem(ctx context.Context, req *connect.Request[transferv1.CreateTextItemRequest]) (*connect.Response[transferv1.CreateTextItemResponse], error) {
	dev, err := deviceauth.RequireDevice(ctx)
	if err != nil {
		return nil, internaltransfer.ToConnectError(err)
	}
	item, err := h.deps.Service.CreateText(ctx, internaltransfer.CreateText{
		OwnerID:        dev.OwnerID,
		OriginDeviceID: dev.ID,
		Text:           req.Msg.Text,
		Name:           req.Msg.Name,
		Retention:      retentionFromProto(req.Msg.Retention),
		TargetDeviceID: req.Msg.TargetDeviceId,
	})
	if err != nil {
		h.deps.Logger.Printf("transfer.CreateTextItem: %v", err)
		return nil, internaltransfer.ToConnectError(err)
	}
	return connect.NewResponse(&transferv1.CreateTextItemResponse{Item: itemToProto(item)}), nil
}

func (h *connectHandler) ListItems(ctx context.Context, req *connect.Request[transferv1.ListItemsRequest]) (*connect.Response[transferv1.ListItemsResponse], error) {
	dev, err := deviceauth.RequireDevice(ctx)
	if err != nil {
		return nil, internaltransfer.ToConnectError(err)
	}
	list, err := h.deps.Service.List(ctx, dev.OwnerID, dev.ID, internaltransfer.ListFilter{
		Query: req.Msg.Query,
		Kind:  kindFromProto(req.Msg.Kind),
	})
	if err != nil {
		h.deps.Logger.Printf("transfer.ListItems: %v", err)
		return nil, internaltransfer.ToConnectError(err)
	}
	resp := &transferv1.ListItemsResponse{Items: make([]*transferv1.Item, 0, len(list))}
	for _, i := range list {
		resp.Items = append(resp.Items, itemToProto(i))
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) GetItem(ctx context.Context, req *connect.Request[transferv1.GetItemRequest]) (*connect.Response[transferv1.GetItemResponse], error) {
	dev, err := deviceauth.RequireDevice(ctx)
	if err != nil {
		return nil, internaltransfer.ToConnectError(err)
	}
	item, err := h.deps.Service.Get(ctx, dev.OwnerID, dev.ID, req.Msg.Id)
	if err != nil {
		return nil, internaltransfer.ToConnectError(err)
	}
	return connect.NewResponse(&transferv1.GetItemResponse{Item: itemToProto(item)}), nil
}

func (h *connectHandler) DeleteItem(ctx context.Context, req *connect.Request[transferv1.DeleteItemRequest]) (*connect.Response[transferv1.DeleteItemResponse], error) {
	dev, err := deviceauth.RequireDevice(ctx)
	if err != nil {
		return nil, internaltransfer.ToConnectError(err)
	}
	deleted, err := h.deps.Service.Delete(ctx, dev.OwnerID, req.Msg.Id)
	if err != nil {
		h.deps.Logger.Printf("transfer.DeleteItem: %v", err)
		return nil, internaltransfer.ToConnectError(err)
	}
	return connect.NewResponse(&transferv1.DeleteItemResponse{Id: deleted.ID}), nil
}
