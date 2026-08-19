package channels

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	channelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/persona/v1/channels"
	"google.golang.org/protobuf/types/known/timestamppb"
	domain "persona/internal/channels"
)

type connectHandler struct{ service domain.Service }

func NewConnectHandler(service domain.Service) *connectHandler {
	return &connectHandler{service: service}
}

func (h *connectHandler) BindChannel(ctx context.Context, req *connect.Request[channelsv1.BindChannelRequest]) (*connect.Response[channelsv1.BindChannelResponse], error) {
	channel, err := h.service.Bind(ctx, domain.ChannelInput{PersonaID: req.Msg.GetPersonaId(), Kind: fromKind(req.Msg.GetKind()), Address: req.Msg.GetAddress(), CredentialRef: req.Msg.GetCredentialRef(), Adapter: req.Msg.GetAdapter()})
	if err != nil {
		return nil, channelError(err)
	}
	return connect.NewResponse(&channelsv1.BindChannelResponse{Channel: toProto(channel)}), nil
}

func (h *connectHandler) ListChannels(ctx context.Context, req *connect.Request[channelsv1.ListChannelsRequest]) (*connect.Response[channelsv1.ListChannelsResponse], error) {
	items, err := h.service.List(ctx, req.Msg.GetPersonaId())
	if err != nil {
		return nil, channelError(err)
	}
	out := &channelsv1.ListChannelsResponse{Channels: make([]*channelsv1.Channel, 0, len(items))}
	for _, item := range items {
		out.Channels = append(out.Channels, toProto(item))
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) SendMessage(ctx context.Context, req *connect.Request[channelsv1.SendMessageRequest]) (*connect.Response[channelsv1.SendMessageResponse], error) {
	message, err := h.service.SendMessage(ctx, req.Msg.GetPersonaId(), req.Msg.GetChannelId(), domain.MessageInput{Recipient: req.Msg.GetRecipient(), Subject: req.Msg.GetSubject(), Body: req.Msg.GetBody()})
	if err != nil {
		return nil, channelError(err)
	}
	return connect.NewResponse(&channelsv1.SendMessageResponse{ChannelId: req.Msg.GetChannelId(), MessageId: message.ID, FromAddress: message.FromAddress}), nil
}

func (h *connectHandler) RetrieveCode(ctx context.Context, req *connect.Request[channelsv1.RetrieveCodeRequest]) (*connect.Response[channelsv1.RetrieveCodeResponse], error) {
	code, err := h.service.RetrieveCode(ctx, req.Msg.GetPersonaId(), req.Msg.GetChannelId(), req.Msg.GetPurpose())
	if err != nil {
		return nil, channelError(err)
	}
	return connect.NewResponse(&channelsv1.RetrieveCodeResponse{ChannelId: req.Msg.GetChannelId(), Code: code.Value, ExpiresAt: timestamppb.New(code.ExpiresAt), Adapter: code.Adapter}), nil
}

func channelError(err error) error {
	code := connect.CodeInternal
	if errors.Is(err, domain.ErrMissingPersona) || errors.Is(err, domain.ErrMissingChannel) || errors.Is(err, domain.ErrInvalidChannel) || errors.Is(err, domain.ErrInvalidMessage) {
		code = connect.CodeInvalidArgument
	}
	if errors.Is(err, domain.ErrAdapterUnavailable) {
		code = connect.CodeUnavailable
	}
	if errors.Is(err, domain.ErrChannelOwnership) {
		code = connect.CodePermissionDenied
	}
	return connect.NewError(code, err)
}

func fromKind(k channelsv1.ChannelKind) domain.Kind {
	switch k {
	case channelsv1.ChannelKind_CHANNEL_KIND_EMAIL:
		return domain.KindEmail
	case channelsv1.ChannelKind_CHANNEL_KIND_SMS:
		return domain.KindSMS
	case channelsv1.ChannelKind_CHANNEL_KIND_DEVICE:
		return domain.KindDevice
	default:
		return ""
	}
}

func toKind(k domain.Kind) channelsv1.ChannelKind {
	switch k {
	case domain.KindEmail:
		return channelsv1.ChannelKind_CHANNEL_KIND_EMAIL
	case domain.KindSMS:
		return channelsv1.ChannelKind_CHANNEL_KIND_SMS
	case domain.KindDevice:
		return channelsv1.ChannelKind_CHANNEL_KIND_DEVICE
	default:
		return channelsv1.ChannelKind_CHANNEL_KIND_UNSPECIFIED
	}
}

func toProto(c domain.Channel) *channelsv1.Channel {
	return &channelsv1.Channel{Id: c.ID, PersonaId: c.PersonaID, Kind: toKind(c.Kind), Address: c.Address, CredentialRef: c.CredentialRef, Adapter: c.Adapter, Enabled: c.Enabled, CreatedAt: timestamppb.New(c.CreatedAt)}
}
