// Package modules is the single registration point for the scenario's
// API modules' static metadata. Both api/main.go and
// api/cmd/gen-endpoints/main.go import this package to enumerate
// domains uniformly.
//
// The runtime Module(...) constructors stay inline in main.go's
// server.New(...) call — they need live deps (db handle, clock, logger)
// and abstracting them is needless ceremony. This package only handles
// the static side: the Endpoints slice each handler exports for
// codegen, and the Schema() function each handler re-exports for
// EnsureSchemas.
//
// Adding a domain: add two lines below — one in AllEndpoints, one in
// AllSchemas. The runtime constructor lands in main.go's server.New
// call as a third line. Three central lines per new domain, no other
// central registry mutations.
package modules

import (
	"device-control/internal/module"

	capsH "device-control/handlers/capabilities"
	controlH "device-control/handlers/control"

	apidb "github.com/vrooli/api-core/database"
	authv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/auth"
	authconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/auth/auth_v1connect"
	devicesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/devices"
	devicesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/devices/devices_v1connect"
	evidencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/evidence"
	evidenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/evidence/evidence_v1connect"
	flowsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/flows"
	flowsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/flows/flows_v1connect"
	sessionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/sessions"
	sessionsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/sessions/sessions_v1connect"
	strategiesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/strategies"
	strategiesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/strategies/strategies_v1connect"
	"google.golang.org/protobuf/reflect/protoreflect"

	healthH "device-control/handlers/health"
	localdb "device-control/internal/database"
)

// AllEndpoints returns every domain's static endpoint descriptors in a
// stable order (system endpoints first, then domains alphabetically).
// The stable order is what makes the diff-exit-code CI check on
// .vrooli/endpoints.json meaningful.
func AllEndpoints() []module.EndpointDescriptor {
	out := make([]module.EndpointDescriptor, 0)
	out = append(out, healthH.Endpoints...)
	out = append(out, capsH.Endpoints...)
	out = append(out, controlH.Endpoints...)
	out = append(out,
		module.EndpointDescriptor{ID: "auth_rpc_list", Path: authconnect.AuthenticationServiceListProfilesProcedure, Method: "POST", Summary: "List authentication profiles", Category: "auth"},
		module.EndpointDescriptor{ID: "auth_rpc_get", Path: authconnect.AuthenticationServiceGetProfileProcedure, Method: "POST", Summary: "Get authentication profile", Category: "auth"},
		module.EndpointDescriptor{ID: "auth_rpc_create", Path: authconnect.AuthenticationServiceCreateProfileProcedure, Method: "POST", Summary: "Create authentication profile", Category: "auth"},
		module.EndpointDescriptor{ID: "auth_rpc_update", Path: authconnect.AuthenticationServiceUpdateProfileProcedure, Method: "POST", Summary: "Update authentication profile", Category: "auth"},
		module.EndpointDescriptor{ID: "auth_rpc_revoke", Path: authconnect.AuthenticationServiceRevokeProfileProcedure, Method: "POST", Summary: "Revoke authentication profile", Category: "auth"},
		module.EndpointDescriptor{ID: "auth_rpc_test", Path: authconnect.AuthenticationServiceTestProfileProcedure, Method: "POST", Summary: "Test authentication profile", Category: "auth"},
		module.EndpointDescriptor{ID: "auth_rpc_unlock", Path: authconnect.AuthenticationServiceUnlockDeviceProcedure, Method: "POST", Summary: "Unlock a device", Category: "auth"},
		module.EndpointDescriptor{ID: "device_rpc_list", Path: devicesconnect.DeviceServiceListDevicesProcedure, Method: "POST", Summary: "List devices", Category: "devices"},
		module.EndpointDescriptor{ID: "device_rpc_connect", Path: devicesconnect.DeviceServiceConnectDeviceProcedure, Method: "POST", Summary: "Show device onboarding", Category: "devices"},
		module.EndpointDescriptor{ID: "device_rpc_reconnect", Path: devicesconnect.DeviceServiceReconnectDeviceProcedure, Method: "POST", Summary: "Reconnect a wireless device", Category: "devices"},
		module.EndpointDescriptor{ID: "strategy_rpc_list", Path: strategiesconnect.StrategyServiceListStrategiesProcedure, Method: "POST", Summary: "List strategies", Category: "strategies"},
		module.EndpointDescriptor{ID: "strategy_rpc_verify", Path: strategiesconnect.StrategyServiceVerifyStrategyProcedure, Method: "POST", Summary: "Verify strategy", Category: "strategies"},
		module.EndpointDescriptor{ID: "session_rpc_list", Path: sessionsconnect.SessionServiceListSessionsProcedure, Method: "POST", Summary: "List sessions", Category: "sessions"},
		module.EndpointDescriptor{ID: "session_rpc_acquire", Path: sessionsconnect.SessionServiceAcquireSessionProcedure, Method: "POST", Summary: "Acquire session", Category: "sessions"},
		module.EndpointDescriptor{ID: "session_rpc_kill", Path: sessionsconnect.SessionServiceKillSessionProcedure, Method: "POST", Summary: "Kill session", Category: "sessions"},
		module.EndpointDescriptor{ID: "session_rpc_release", Path: sessionsconnect.SessionServiceReleaseSessionProcedure, Method: "POST", Summary: "Release session", Category: "sessions"},
		module.EndpointDescriptor{ID: "flow_rpc_validate", Path: flowsconnect.FlowServiceValidateFlowProcedure, Method: "POST", Summary: "Validate flow", Category: "flows"},
		module.EndpointDescriptor{ID: "flow_rpc_run", Path: flowsconnect.FlowServiceRunFlowProcedure, Method: "POST", Summary: "Run flow", Category: "flows"},
		module.EndpointDescriptor{ID: "evidence_rpc_list", Path: evidenceconnect.EvidenceServiceListAuditProcedure, Method: "POST", Summary: "List audit", Category: "evidence"},
	)
	return out
}

// ProtoFileEntry pairs a domain module's name with the proto
// FileDescriptor whose RPCs that module exposes via Connect-RPC. The
// global parity test in registry_test.go walks every entry and asserts
// each rpc method in the FileDescriptor has exactly one matching
// EndpointDescriptor in AllEndpoints().
//
// Adding a Connect-RPC domain: append one line below. The global parity
// test then covers it automatically — there is no per-domain parity
// test to write.
//
// REST-exception-only domains (none in the template today) are simply
// not listed here; the global test never inspects them, and the
// gen-endpoints validateTransport pass enforces their RESTException
// tags at codegen time.
type ProtoFileEntry struct {
	Module string
	File   protoreflect.FileDescriptor
}

// AllProtoFiles returns the proto FileDescriptor backing each
// Connect-mounted domain module, in registration order.
func AllProtoFiles() []ProtoFileEntry {
	return []ProtoFileEntry{
		{Module: "auth", File: authv1.File_device_control_v1_auth_auth_proto},
		{Module: "devices", File: devicesv1.File_device_control_v1_devices_devices_proto},
		{Module: "strategies", File: strategiesv1.File_device_control_v1_strategies_strategies_proto},
		{Module: "sessions", File: sessionsv1.File_device_control_v1_sessions_sessions_proto},
		{Module: "flows", File: flowsv1.File_device_control_v1_flows_flows_proto},
		{Module: "evidence", File: evidencev1.File_device_control_v1_evidence_evidence_proto},
	}
}

// AllSchemas returns every domain's schema provider plus the system
// schema (always first; cross-cutting infrastructure runs before any
// domain table). Consumed by main.go's database.EnsureSchemas call.
//
// Order matters: system → health → … (domains alphabetical).
// Postgres scenarios that put `CREATE EXTENSION ...` in system.sql rely
// on system running before any domain that references the extension.
func AllSchemas() []apidb.SchemaProvider {
	return []apidb.SchemaProvider{
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(healthH.Schema),
	}
}
