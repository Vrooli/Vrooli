package intent

import (
	"compute-manager/internal/module"
	intentconnect "github.com/vrooli/vrooli/packages/proto/gen/go/compute-manager/v1/intent/intent_v1connect"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "intent_list", Path: intentconnect.IntentServiceListIntentsProcedure, Method: "POST", Summary: "List provisioning intents", Category: "intent"},
	{ID: "intent_get", Path: intentconnect.IntentServiceGetIntentProcedure, Method: "POST", Summary: "Get a provisioning intent", Category: "intent"},
}
