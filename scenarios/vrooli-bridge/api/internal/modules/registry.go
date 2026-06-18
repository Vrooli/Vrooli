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
	"vrooli-bridge/internal/module"

	apidb "github.com/vrooli/api-core/database"
	"google.golang.org/protobuf/reflect/protoreflect"

	auditH "vrooli-bridge/handlers/audit"
	channelH "vrooli-bridge/handlers/channel"
	dispatchH "vrooli-bridge/handlers/dispatch"
	healthH "vrooli-bridge/handlers/health"
	pairingH "vrooli-bridge/handlers/pairing"
	provisionH "vrooli-bridge/handlers/provision"
	registryH "vrooli-bridge/handlers/registry"
	runsH "vrooli-bridge/handlers/runs"
	localdb "vrooli-bridge/internal/database"

	auditv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/audit"
	dispatchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/dispatch"
	pairingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/pairing"
	presencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/presence"
	provisionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/provision"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/registry"
	runsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/runs"
)

// AllEndpoints returns every domain's static endpoint descriptors in a
// stable order (system endpoints first, then domains alphabetically).
// The stable order is what makes the diff-exit-code CI check on
// .vrooli/endpoints.json meaningful.
func AllEndpoints() []module.EndpointDescriptor {
	out := make([]module.EndpointDescriptor, 0)
	out = append(out, healthH.Endpoints...)
	out = append(out, auditH.Endpoints...)
	out = append(out, channelH.Endpoints...)
	out = append(out, dispatchH.Endpoints...)
	out = append(out, pairingH.Endpoints...)
	out = append(out, provisionH.Endpoints...)
	out = append(out, registryH.Endpoints...)
	out = append(out, runsH.Endpoints...)
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
		{Module: "audit", File: auditv1.File_vrooli_bridge_v1_audit_audit_proto},
		{Module: "channel", File: presencev1.File_vrooli_bridge_v1_presence_presence_proto},
		{Module: "dispatch", File: dispatchv1.File_vrooli_bridge_v1_dispatch_dispatch_proto},
		{Module: "pairing", File: pairingv1.File_vrooli_bridge_v1_pairing_pairing_proto},
		{Module: "provision", File: provisionv1.File_vrooli_bridge_v1_provision_provision_proto},
		{Module: "registry", File: registryv1.File_vrooli_bridge_v1_registry_registry_proto},
		{Module: "runs", File: runsv1.File_vrooli_bridge_v1_runs_runs_proto},
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
		apidb.SchemaProviderFunc(auditH.Schema),
		apidb.SchemaProviderFunc(channelH.Schema),
		apidb.SchemaProviderFunc(pairingH.Schema),
		apidb.SchemaProviderFunc(provisionH.Schema),
		apidb.SchemaProviderFunc(registryH.Schema),
		apidb.SchemaProviderFunc(runsH.Schema),
	}
}
