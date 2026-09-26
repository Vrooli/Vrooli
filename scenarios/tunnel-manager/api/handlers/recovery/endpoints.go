package recovery

import (
	"tunnel-manager/internal/module"

	recoveryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/recovery/recovery_v1connect"
)

// Endpoints is the machine-readable description of the recovery module's
// public surface. Connect-RPC method paths reference the generated
// *Procedure constants, so adding or renaming an RPC in recovery.proto
// breaks this file at compile time. The global proto↔Connect parity test
// asserts every rpc has exactly one entry here.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "recovery_get_state",
		Path:        recoveryconnect.RecoveryServiceGetStateProcedure,
		Method:      "POST",
		Summary:     "Get recovery state",
		Description: "Returns the live recovery state-machine snapshot (status, consecutive failures, backoff level, circuit-breaker state) through the generated Connect-RPC Recovery service.",
		Category:    "recovery",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"state": "RecoveryState"},
		},
		Errors: []module.ErrorDesc{
			{Status: 500, Code: "internal", Description: "State read failure"},
		},
		Examples: []module.Example{
			{Name: "Get state", Curl: "curl http://localhost:${API_PORT}/vrooli.tunnel_manager.v1.recovery.RecoveryService/GetState -H 'Content-Type: application/json' -d '{}'"},
		},
	},
	{
		ID:          "recovery_list_events",
		Path:        recoveryconnect.RecoveryServiceListEventsProcedure,
		Method:      "POST",
		Summary:     "List recovery events",
		Description: "Returns the recovery event log newest-first, capped at limit (default 50), through the generated Connect-RPC Recovery service.",
		Category:    "recovery",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"limit": "int32 (0 = default 50)"},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"events": "array<RecoveryEvent>"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Negative limit"},
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		Examples: []module.Example{
			{Name: "List events", Curl: "curl http://localhost:${API_PORT}/vrooli.tunnel_manager.v1.recovery.RecoveryService/ListEvents -H 'Content-Type: application/json' -d '{\"limit\":20}'"},
		},
	},
	{
		ID:          "recovery_recover",
		Path:        recoveryconnect.RecoveryServiceRecoverProcedure,
		Method:      "POST",
		Summary:     "Trigger recovery",
		Description: "Triggers a manual cloudflared recovery attempt. Idempotent while a recovery is already in flight (returns SKIPPED); rejected while the circuit breaker is open unless force is true.",
		Category:    "recovery",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"force": "bool (bypass circuit breaker)"},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"outcome": "EventOutcome", "event": "RecoveryEvent"},
		},
		Errors: []module.ErrorDesc{
			{Status: 500, Code: "internal", Description: "Restart or persistence failure"},
		},
		Examples: []module.Example{
			{Name: "Force recover", Curl: "curl http://localhost:${API_PORT}/vrooli.tunnel_manager.v1.recovery.RecoveryService/Recover -H 'Content-Type: application/json' -d '{\"force\":true}'"},
		},
	},
}
