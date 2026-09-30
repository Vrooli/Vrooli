package interactive

import (
	interactiveconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/interactive/interactive_v1connect"
	"vrooli-bridge/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "interactive_open_channel", Path: interactiveconnect.InteractiveDesktopServiceOpenChannelProcedure, Method: "POST", Summary: "Open interactive desktop channel", Description: "Owner-gated Bridge admission for one Device Control desktop session.", Category: "interactive-desktop"},
	{ID: "interactive_signal", Path: interactiveconnect.InteractiveDesktopServiceSignalProcedure, Method: "POST", Summary: "Signal interactive desktop channel", Description: "Owner-gated bounded WebRTC offer, answer, or ICE signaling.", Category: "interactive-desktop"},
	{ID: "interactive_close_channel", Path: interactiveconnect.InteractiveDesktopServiceCloseChannelProcedure, Method: "POST", Summary: "Close interactive desktop channel", Description: "Close one authorized desktop channel.", Category: "interactive-desktop"},
	{ID: "interactive_revoke_channel", Path: interactiveconnect.InteractiveDesktopServiceRevokeChannelProcedure, Method: "POST", Summary: "Revoke interactive desktop channel", Description: "Revoke one desktop channel without changing unrelated sessions.", Category: "interactive-desktop"},
}
