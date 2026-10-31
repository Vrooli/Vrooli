package devices

import (
	devicesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/devices/devices_v1connect"
	"web-console/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "devices_list", Path: devicesconnect.DeviceServiceListProcedure, Method: "POST", Summary: "List connected devices", Description: "Returns the live browser devices and the sessions they are viewing.", Category: "devices"},
	{ID: "devices_disconnect", Path: devicesconnect.DeviceServiceDisconnectProcedure, Method: "POST", Summary: "Disconnect a device connection", Description: "Closes one named connection owned by another device.", Category: "devices"},
	{ID: "devices_give_control", Path: devicesconnect.DeviceServiceGiveControlProcedure, Method: "POST", Summary: "Give terminal control to a device", Description: "Transfers terminal-size and input authority to a live device connection.", Category: "devices"},
}
