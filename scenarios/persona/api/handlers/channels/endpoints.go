package channels

import (
	channelsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/persona/v1/channels/channels_v1connect"
	"persona/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "channels_bind", Path: channelsconnect.ChannelsServiceBindChannelProcedure, Method: "POST", Summary: "Bind a controlled channel", Category: "channels"},
	{ID: "channels_list", Path: channelsconnect.ChannelsServiceListChannelsProcedure, Method: "POST", Summary: "List controlled channels", Category: "channels"},
	{ID: "channels_send_message", Path: channelsconnect.ChannelsServiceSendMessageProcedure, Method: "POST", Summary: "Send from a controlled channel", Category: "channels"},
	{ID: "channels_retrieve_code", Path: channelsconnect.ChannelsServiceRetrieveCodeProcedure, Method: "POST", Summary: "Retrieve a one-time code", Category: "channels"},
}
