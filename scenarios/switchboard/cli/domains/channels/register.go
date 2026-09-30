package channels

import (
	"connectrpc.com/connect"
	"context"
	"fmt"
	"github.com/vrooli/cli-core/cliapp"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/switchboard/v1/channels"
	connectv1 "github.com/vrooli/vrooli/packages/proto/gen/go/switchboard/v1/channels/channels_v1connect"
)

const GroupName = "channels"

type handlers struct {
	client connectv1.ChannelServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, base := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: connectv1.NewChannelServiceClient(httpClient, base)}
}
func (h *handlers) list(cliapp.OperationContext) (*v1.ListChannelsResponse, error) {
	r, e := h.client.ListChannels(context.Background(), connect.NewRequest(&v1.ListChannelsRequest{}))
	if e != nil {
		return nil, e
	}
	return r.Msg, nil
}
func (h *handlers) get(ctx cliapp.OperationContext) (*v1.GetChannelResponse, error) {
	r, e := h.client.GetChannel(context.Background(), connect.NewRequest(&v1.GetChannelRequest{Id: ctx.Positional("id")}))
	if e != nil {
		return nil, e
	}
	return r.Msg, nil
}
func (h *handlers) bindings(ctx cliapp.OperationContext) (*v1.ListBindingsResponse, error) {
	r, e := h.client.ListBindings(context.Background(), connect.NewRequest(&v1.ListBindingsRequest{AgentId: ctx.Flag("agent-id")}))
	if e != nil {
		return nil, e
	}
	return r.Msg, nil
}
func (h *handlers) create(ctx cliapp.OperationContext) (*v1.CreateBindingResponse, error) {
	r, e := h.client.CreateBinding(context.Background(), connect.NewRequest(&v1.CreateBindingRequest{AgentId: ctx.Flag("agent-id"), ChannelId: ctx.Flag("channel-id"), Address: ctx.Flag("address"), ThreadKey: ctx.Flag("thread-key")}))
	if e != nil {
		return nil, e
	}
	return r.Msg, nil
}
func (h *handlers) send(ctx cliapp.OperationContext) (*v1.SendMessageResponse, error) {
	r, e := h.client.SendMessage(context.Background(), connect.NewRequest(&v1.SendMessageRequest{ChannelId: ctx.Flag("channel-id"), ThreadKey: ctx.Flag("thread-key"), Text: ctx.Flag("text")}))
	if e != nil {
		return nil, e
	}
	return r.Msg, nil
}
func reportList(_ cliapp.OperationContext, r *v1.ListChannelsResponse) cliapp.ListReport {
	out := []string{}
	for _, c := range r.Channels {
		out = append(out, fmt.Sprintf("%s — %s (%s)", c.Id, c.DisplayName, c.Availability))
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d channel(s).", len(out))}, ResultsHeading: "Channels", Results: out}
}
func reportGet(_ cliapp.OperationContext, r *v1.GetChannelResponse) cliapp.ListReport {
	return cliapp.ListReport{ResultsHeading: "Channel", Results: []string{r.Channel.Id + " — " + r.Channel.DisplayName}}
}
func reportBindings(_ cliapp.OperationContext, r *v1.ListBindingsResponse) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d binding(s).", len(r.Bindings))}}
}
func reportCreate(_ cliapp.OperationContext, r *v1.CreateBindingResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{"Created binding " + r.Binding.Id + "."}}
}
func reportSend(_ cliapp.OperationContext, r *v1.SendMessageResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Message accepted: %t.", r.Accepted)}}
}
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	b := map[string]cliapp.PrimitiveHandler{"ChannelService.ListChannels": cliapp.ProtoList(h.list, reportList), "ChannelService.GetChannel": cliapp.ProtoList(h.get, reportGet), "ChannelService.ListBindings": cliapp.ProtoList(h.bindings, reportBindings), "ChannelService.CreateBinding": cliapp.ProtoMutation(h.create, reportCreate), "ChannelService.SendMessage": cliapp.ProtoMutation(h.send, reportSend)}
	g, e := cliapp.LoadFromManifestPrimitives(manifest, GroupName, b)
	if e != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("channels: %w", e)
	}
	return g, nil
}
