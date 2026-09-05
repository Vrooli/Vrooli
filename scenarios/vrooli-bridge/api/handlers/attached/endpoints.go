package attached

import "vrooli-bridge/internal/module"

var Endpoints = []module.EndpointDescriptor{
	{ID: "attached_pair", Path: "/vrooli.vrooli_bridge.v1.attached_devices.AttachedDeviceService/PairAttachedDevice", Method: "POST", Summary: "Pair an attached device through a trusted host", Category: "attached-devices"},
	{ID: "attached_list", Path: "/vrooli.vrooli_bridge.v1.attached_devices.AttachedDeviceService/ListAttachedDevices", Method: "POST", Summary: "List attached devices without hiding offline hosts", Category: "attached-devices"},
	{ID: "attached_revoke", Path: "/vrooli.vrooli_bridge.v1.attached_devices.AttachedDeviceService/RevokeAttachedDevice", Method: "POST", Summary: "Revoke an attached device", Category: "attached-devices"},
}
