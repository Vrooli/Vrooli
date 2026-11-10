package conversations

import (
	"connectrpc.com/connect"
	"context"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
)

func (h *handler) ListNativeConsents(ctx context.Context, req *connect.Request[v1.ListNativeConsentsRequest]) (*connect.Response[v1.ListNativeConsentsResponse], error) {
	page, err := h.service.ListNativeConsents(ctx, h.service.ResolveRecipient(ctx), req.Msg.GetQuery(), req.Msg.GetCursor(), int(req.Msg.GetLimit()))
	if err != nil {
		return nil, connectError(err)
	}
	out := &v1.ListNativeConsentsResponse{NextCursor: page.NextCursor}
	for _, c := range page.Consents {
		out.Consents = append(out.Consents, &v1.NativeConsent{AuthorizationId: c.AuthorizationID, NotificationId: c.NotificationID, Source: c.Source, Title: c.Title, Body: c.Body, ReviewUrl: c.ReviewURL, State: c.State, ReviewExpiresAt: c.ReviewExpiresAt, ObservedAt: c.ObservedAt, CreatedAt: c.CreatedAt})
	}
	response := connect.NewResponse(out)
	response.Header().Set("Cache-Control", "no-store")
	return response, nil
}
