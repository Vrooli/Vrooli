// Package events exposes the canonical metadata for the EventsService
// Connect-RPC endpoints. gen-endpoints consumes this slice to keep
// `.vrooli/endpoints.json` in sync with the wire surface.
package events

import (
	eventsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/events/events_v1connect"

	"web-console/internal/module"
)

var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "events_list",
		Path:        eventsconnect.EventsServiceListProcedure,
		Method:      "POST",
		Summary:     "List recent lifecycle events",
		Description: "Returns the most recent events from the in-memory ring buffer, newest last. The server caps the limit at 1000.",
		Category:    "events",
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"events": "Event[]",
				"total":  "int32",
			},
		},
	},
}
