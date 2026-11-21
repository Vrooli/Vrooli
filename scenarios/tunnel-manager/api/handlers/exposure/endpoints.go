package exposure

import (
	"tunnel-manager/internal/module"

	exposureconnect "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/exposure/exposure_v1connect"
)

// Endpoints describes the exposure module's public surface. Connect-RPC
// method paths reference generated *Procedure constants, so adding or
// renaming an RPC in exposure.proto breaks this file at compile time. The
// global proto↔Connect parity test asserts every rpc has exactly one entry.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "exposure_expose",
		Path:        exposureconnect.ExposureServiceExposeProcedure,
		Method:      "POST",
		Summary:     "Expose a scenario",
		Description: "Ensures the scenario has a LEASED route, is running, and has live ingress; returns the lease and reachable URL. Idempotent — re-exposing extends the existing lease rather than duplicating ingress.",
		Category:    "exposure",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"scenario":     "string (required)",
				"ttl_seconds":  "int64 (0 = default ~1 week)",
				"requested_by": "string (optional caller identity)",
			},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"lease": "Lease", "public_url": "string"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing scenario"},
			{Status: 412, Code: "failed_precondition", Description: "Scenario has no resolvable fixed UI port"},
			{Status: 500, Code: "internal", Description: "Route, run, or ingress failure"},
		},
		Examples: []module.Example{
			{Name: "Expose for a day", Curl: "curl http://localhost:${API_PORT}/vrooli.tunnel_manager.v1.exposure.ExposureService/Expose -H 'Content-Type: application/json' -d '{\"scenario\":\"web-console\",\"ttl_seconds\":86400}'"},
		},
	},
	{
		ID:          "exposure_extend_lease",
		Path:        exposureconnect.ExposureServiceExtendLeaseProcedure,
		Method:      "POST",
		Summary:     "Extend a lease",
		Description: "Pushes a lease's expiry out by ttl_seconds from now.",
		Category:    "exposure",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"lease_id": "string (required)", "ttl_seconds": "int64 (0 = default)"},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"lease": "Lease"},
		},
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "No lease with that id"},
			{Status: 500, Code: "internal", Description: "Repository write failure"},
		},
		Examples: []module.Example{
			{Name: "Extend", Curl: "curl http://localhost:${API_PORT}/vrooli.tunnel_manager.v1.exposure.ExposureService/ExtendLease -H 'Content-Type: application/json' -d '{\"lease_id\":\"abc\",\"ttl_seconds\":3600}'"},
		},
	},
	{
		ID:          "exposure_revoke_lease",
		Path:        exposureconnect.ExposureServiceRevokeLeaseProcedure,
		Method:      "POST",
		Summary:     "Revoke a lease",
		Description: "Ends a lease immediately and retracts ingress unless the scenario is also CORE.",
		Category:    "exposure",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"lease_id": "string (required)"},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"retracted": "bool (false when scenario is also CORE)"},
		},
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "No lease with that id"},
			{Status: 500, Code: "internal", Description: "Repository or ingress failure"},
		},
		Examples: []module.Example{
			{Name: "Revoke", Curl: "curl http://localhost:${API_PORT}/vrooli.tunnel_manager.v1.exposure.ExposureService/RevokeLease -H 'Content-Type: application/json' -d '{\"lease_id\":\"abc\"}'"},
		},
	},
	{
		ID:          "exposure_unexpose",
		Path:        exposureconnect.ExposureServiceUnexposeProcedure,
		Method:      "POST",
		Summary:     "Unexpose a scenario",
		Description: "Revokes a scenario's active lease by name, retracting ingress and the TM-created DNS record unless the scenario is also CORE.",
		Category:    "exposure",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"scenario": "string (required)"},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"retracted": "bool (false when scenario is also CORE)", "lease_id": "string"},
		},
		Errors: []module.ErrorDesc{
			{Status: 404, Code: "not_found", Description: "No active lease for that scenario"},
			{Status: 500, Code: "internal", Description: "Repository or ingress failure"},
		},
		Examples: []module.Example{
			{Name: "Unexpose", Curl: "curl http://localhost:${API_PORT}/vrooli.tunnel_manager.v1.exposure.ExposureService/Unexpose -H 'Content-Type: application/json' -d '{\"scenario\":\"react-component-library\"}'"},
		},
	},
	{
		ID:          "exposure_list_leases",
		Path:        exposureconnect.ExposureServiceListLeasesProcedure,
		Method:      "POST",
		Summary:     "List leases",
		Description: "Returns leases newest-first, optionally filtered by status.",
		Category:    "exposure",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"status": "LeaseStatus (unset = all)"},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"leases": "array<Lease>"},
		},
		Errors: []module.ErrorDesc{
			{Status: 500, Code: "internal", Description: "Repository read failure"},
		},
		Examples: []module.Example{
			{Name: "List active", Curl: "curl http://localhost:${API_PORT}/vrooli.tunnel_manager.v1.exposure.ExposureService/ListLeases -H 'Content-Type: application/json' -d '{\"status\":\"LEASE_STATUS_ACTIVE\"}'"},
		},
	},
	{
		ID:          "exposure_list_exposures",
		Path:        exposureconnect.ExposureServiceListExposuresProcedure,
		Method:      "POST",
		Summary:     "List exposures",
		Description: "Returns the reconciled exposure state per scenario (tier, route, lease, reachability) — the data behind the Exposure UI surface.",
		Category:    "exposure",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"exposures": "array<Exposure>"},
		},
		Errors: []module.ErrorDesc{
			{Status: 500, Code: "internal", Description: "Manifest read failure"},
		},
		Examples: []module.Example{
			{Name: "List exposures", Curl: "curl http://localhost:${API_PORT}/vrooli.tunnel_manager.v1.exposure.ExposureService/ListExposures -H 'Content-Type: application/json' -d '{}'"},
		},
	},
	{
		ID:          "exposure_is_exposed",
		Path:        exposureconnect.ExposureServiceIsExposedProcedure,
		Method:      "POST",
		Summary:     "Check exposure",
		Description: "Reports whether a scenario is currently reachable and, if so, its public URL — the read backing app-monitor open-in-new-tab.",
		Category:    "exposure",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"scenario": "string (required)"},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"exposed": "bool", "public_url": "string"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing scenario"},
			{Status: 500, Code: "internal", Description: "Manifest read failure"},
		},
		Examples: []module.Example{
			{Name: "Is exposed", Curl: "curl http://localhost:${API_PORT}/vrooli.tunnel_manager.v1.exposure.ExposureService/IsExposed -H 'Content-Type: application/json' -d '{\"scenario\":\"web-console\"}'"},
		},
	},
	{
		ID:          "exposure_reconcile",
		Path:        exposureconnect.ExposureServiceReconcileProcedure,
		Method:      "POST",
		Summary:     "Reconcile exposure",
		Description: "Re-derives CORE routes from the coreset closure and reaps expired leases. Idempotent; also run on a timer.",
		Category:    "exposure",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"core_ensured": "int32", "leases_reaped": "int32"},
		},
		Errors: []module.ErrorDesc{
			{Status: 500, Code: "internal", Description: "Manifest or repository failure"},
		},
		Examples: []module.Example{
			{Name: "Reconcile", Curl: "curl http://localhost:${API_PORT}/vrooli.tunnel_manager.v1.exposure.ExposureService/Reconcile -H 'Content-Type: application/json' -d '{}'"},
		},
	},
}
