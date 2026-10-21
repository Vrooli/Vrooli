package devices

import (
	"device-studio/internal/module"
	devicesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/devices/devices_v1connect"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "devices_candidate_save", Path: devicesconnect.DevicesServiceSaveDeviceCandidateProcedure, Method: "POST", Summary: "Save device candidate", Description: "Appends an immutable device candidate revision at the expected head. Candidate state does not imply qualification.", Category: "devices"},
	{ID: "devices_revision_get", Path: devicesconnect.DevicesServiceGetDeviceRevisionProcedure, Method: "POST", Summary: "Get device revision", Description: "Returns one immutable device candidate revision after verifying its canonical digest.", Category: "devices"},
	{ID: "devices_head_get", Path: devicesconnect.DevicesServiceGetDeviceHeadProcedure, Method: "POST", Summary: "Get device head", Description: "Returns the current immutable device candidate revision.", Category: "devices"},
	{ID: "devices_revision_list", Path: devicesconnect.DevicesServiceListDeviceRevisionsProcedure, Method: "POST", Summary: "List device revisions", Description: "Returns a bounded ascending page of immutable device revisions.", Category: "devices"},
	{ID: "devices_candidate_list", Path: devicesconnect.DevicesServiceListDeviceCandidatesProcedure, Method: "POST", Summary: "List device candidate heads", Description: "Returns one latest immutable candidate revision per device in stable identifier order.", Category: "devices"},
}
