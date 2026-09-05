package handoffrules

import (
	handoffrulesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/handoffrules/handoffrules_v1connect"

	"web-console/internal/module"
)

// Endpoints describes the capture-rule module's public surface. Connect-RPC
// method paths reference generated *Procedure constants so adding or renaming
// an RPC in handoffrules.proto breaks this file at compile time.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "handoffrules_list_rules",
		Path:        handoffrulesconnect.HandoffRulesServiceListRulesProcedure,
		Method:      "POST",
		Summary:     "List handoff capture rules",
		Description: "Returns every capture rule. A rule decides when a handoff is suggested; it never sends one.",
		Category:    "handoffrules",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"rules": "[]Rule"},
		},
	},
	{
		ID:          "handoffrules_upsert_rule",
		Path:        handoffrulesconnect.HandoffRulesServiceUpsertRuleProcedure,
		Method:      "POST",
		Summary:     "Create or update a handoff capture rule",
		Description: "Upserts a rule by id. A blank id is assigned server-side. Rejects a blank name, a blank pattern, and an unknown source.",
		Category:    "handoffrules",
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"rule": "Rule"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Blank name, blank pattern, or an unknown source"},
		},
	},
	{
		ID:          "handoffrules_delete_rule",
		Path:        handoffrulesconnect.HandoffRulesServiceDeleteRuleProcedure,
		Method:      "POST",
		Summary:     "Delete a handoff capture rule",
		Description: "Idempotent: succeeds whether the id exists or not. Every rule is deletable, including a shipped example.",
		Category:    "handoffrules",
	},
}
