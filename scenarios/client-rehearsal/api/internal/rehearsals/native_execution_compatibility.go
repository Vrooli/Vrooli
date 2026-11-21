package rehearsals

// NativeTriggerRoute states how a trigger enters n8n through the owner CLI.
// Unsupported trigger families are represented explicitly so the service
// never silently treats an event, poll, error, or child workflow as manual.
type NativeTriggerRoute string

const (
	NativeRouteManual      NativeTriggerRoute = "manual-cli-start"
	NativeRouteScheduled   NativeTriggerRoute = "schedule-one-shot-cli-start"
	NativeRouteWebhook     NativeTriggerRoute = "owner-webhook-dispatch"
	NativeRouteUnsupported NativeTriggerRoute = "unsupported"
)

type NativeTriggerCompatibility struct {
	Kind            TriggerKind
	Route           NativeTriggerRoute
	InputRoute      string
	Supported       bool
	Incompatibility string
}

func nativeTriggerCompatibility(kind TriggerKind) NativeTriggerCompatibility {
	switch kind {
	case TriggerManual:
		return NativeTriggerCompatibility{Kind: kind, Route: NativeRouteManual, InputRoute: "task-local-set-node", Supported: true}
	case TriggerScheduled:
		return NativeTriggerCompatibility{Kind: kind, Route: NativeRouteScheduled, InputRoute: "task-local-set-node", Supported: true}
	case TriggerWebhook:
		return NativeTriggerCompatibility{Kind: kind, Route: NativeRouteWebhook, InputRoute: "owner-dispatched-event-json", Supported: true}
	case TriggerPolling:
		return NativeTriggerCompatibility{Kind: kind, Route: NativeRouteUnsupported, Incompatibility: "polling triggers require an active poll scheduler; the n8n CLI start route cannot produce a poll event"}
	case TriggerEvent:
		return NativeTriggerCompatibility{Kind: kind, Route: NativeRouteUnsupported, Incompatibility: "instance event triggers require an n8n event-bus event that the task owner CLI does not expose"}
	case TriggerError:
		return NativeTriggerCompatibility{Kind: kind, Route: NativeRouteUnsupported, Incompatibility: "Error Trigger requires an upstream failed execution context; it cannot be started as a standalone workflow"}
	case TriggerSubworkflow:
		return NativeTriggerCompatibility{Kind: kind, Route: NativeRouteUnsupported, Incompatibility: "Execute Workflow Trigger requires a parent workflow execution context"}
	case TriggerChat:
		return NativeTriggerCompatibility{Kind: kind, Route: NativeRouteUnsupported, Incompatibility: "Chat Trigger requires the n8n chat session runtime, which the task owner CLI does not expose"}
	default:
		return NativeTriggerCompatibility{Kind: kind, Route: NativeRouteUnsupported, Incompatibility: "trigger family is not mapped to a supported native n8n owner route"}
	}
}
