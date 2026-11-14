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
	"network-manager/internal/module"

	apidb "github.com/vrooli/api-core/database"
	"google.golang.org/protobuf/reflect/protoreflect"

	adaptersH "network-manager/handlers/adapters"
	healthH "network-manager/handlers/health"
	homeintegrationH "network-manager/handlers/homeintegration"
	inventoryH "network-manager/handlers/inventory"
	monitoringH "network-manager/handlers/monitoring"
	optimizationH "network-manager/handlers/optimization"
	policyH "network-manager/handlers/policy"
	privacyH "network-manager/handlers/privacy"
	resolverH "network-manager/handlers/resolver"
	snapshotH "network-manager/handlers/snapshot"
	localdb "network-manager/internal/database"

	adaptersv1 "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/adapters"
	homeintegrationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/home_integration"
	inventoryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/inventory"
	monitoringv1 "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/monitoring"
	optimizationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/optimization"
	policyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/policy"
	privacyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/privacy"
	resolverv1 "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/resolver"
	snapshotv1 "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/snapshot"
)

// AllEndpoints returns every domain's static endpoint descriptors in a
// stable order (system endpoints first, then domains alphabetically).
// The stable order is what makes the diff-exit-code CI check on
// .vrooli/endpoints.json meaningful.
func AllEndpoints() []module.EndpointDescriptor {
	out := make([]module.EndpointDescriptor, 0)
	out = append(out, healthH.Endpoints...)
	out = append(out, adaptersH.Endpoints...)
	out = append(out, homeintegrationH.Endpoints...)
	out = append(out, inventoryH.Endpoints...)
	out = append(out, monitoringH.Endpoints...)
	out = append(out, optimizationH.Endpoints...)
	out = append(out, policyH.Endpoints...)
	out = append(out, privacyH.Endpoints...)
	out = append(out, resolverH.Endpoints...)
	out = append(out, snapshotH.Endpoints...)
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
		{Module: "adapters", File: adaptersv1.File_network_manager_v1_adapters_adapters_proto},
		{Module: "home_integration", File: homeintegrationv1.File_network_manager_v1_home_integration_home_integration_proto},
		{Module: "inventory", File: inventoryv1.File_network_manager_v1_inventory_inventory_proto},
		{Module: "monitoring", File: monitoringv1.File_network_manager_v1_monitoring_monitoring_proto},
		{Module: "optimization", File: optimizationv1.File_network_manager_v1_optimization_optimization_proto},
		{Module: "policy", File: policyv1.File_network_manager_v1_policy_policy_proto},
		{Module: "privacy", File: privacyv1.File_network_manager_v1_privacy_privacy_proto},
		{Module: "resolver", File: resolverv1.File_network_manager_v1_resolver_resolver_proto},
		{Module: "snapshot", File: snapshotv1.File_network_manager_v1_snapshot_snapshot_proto},
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
		apidb.SchemaProviderFunc(adaptersH.Schema),
		apidb.SchemaProviderFunc(homeintegrationH.Schema),
		apidb.SchemaProviderFunc(inventoryH.Schema),
		apidb.SchemaProviderFunc(monitoringH.Schema),
		apidb.SchemaProviderFunc(optimizationH.Schema),
		apidb.SchemaProviderFunc(policyH.Schema),
		apidb.SchemaProviderFunc(privacyH.Schema),
		apidb.SchemaProviderFunc(resolverH.Schema),
		apidb.SchemaProviderFunc(snapshotH.Schema),
	}
}
