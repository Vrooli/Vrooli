// Package hooks owns the descriptors for Claude Code lifecycle hook
// endpoints. These are inbound webhooks called by the Claude Code CLI —
// a tool we do not control — so they cannot use a generated Connect
// client and stay REST under the webhook_receiver exception. The actual
// HTTP handlers continue to live in api/main.go for now; this package
// only exposes the canonical metadata so gen-endpoints validates them
// against the RESTException rule.
package hooks

import "web-console/internal/module"

var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "hooks_stop",
		Path:        "/api/v1/hooks/stop",
		Method:      "POST",
		Summary:     "Claude Code Stop hook receiver",
		Description: "Inbound webhook from the Claude Code CLI fired on the Stop lifecycle event. Routes the assistant's final response into the conversation store and (when present) records agent identity for session recovery. Authenticated by the X-Hook-Token header.",
		Category:    "hooks",
		RESTException: &module.RESTException{
			Reason: module.RESTReasonWebhookReceiver,
			Note:   "Called by the Claude Code CLI, which we do not control. Request shape is dictated by Claude Code and cannot be wrapped in a Connect client.",
			ProtoPayloads: &module.RESTProtoPayloads{
				Request:  module.RESTPayload{Transport: "json", Conformance: "external_shape"},
				Response: module.RESTPayload{Transport: "json", Conformance: "external_shape"},
				Error:    module.RESTPayload{Transport: "json", Conformance: "external_shape"},
			},
		},
	},
	{
		ID:          "hooks_prompt_submit",
		Path:        "/api/v1/hooks/prompt-submit",
		Method:      "POST",
		Summary:     "Claude Code UserPromptSubmit hook receiver",
		Description: "Inbound webhook from the Claude Code CLI fired when the user submits a prompt. Appends the prompt to the conversation store so the UI sees user turns even when typed through the agent CLI rather than the web console. Authenticated by the X-Hook-Token header.",
		Category:    "hooks",
		RESTException: &module.RESTException{
			Reason: module.RESTReasonWebhookReceiver,
			Note:   "Called by the Claude Code CLI, which we do not control. Request shape is dictated by Claude Code and cannot be wrapped in a Connect client.",
			ProtoPayloads: &module.RESTProtoPayloads{
				Request:  module.RESTPayload{Transport: "json", Conformance: "external_shape"},
				Response: module.RESTPayload{Transport: "json", Conformance: "external_shape"},
				Error:    module.RESTPayload{Transport: "json", Conformance: "external_shape"},
			},
		},
	},
}
