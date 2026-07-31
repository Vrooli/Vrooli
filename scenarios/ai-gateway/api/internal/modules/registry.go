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
	"ai-gateway/internal/module"

	apidb "github.com/vrooli/api-core/database"
	"google.golang.org/protobuf/reflect/protoreflect"

	conformanceH "ai-gateway/handlers/conformance"
	gatewayH "ai-gateway/handlers/gateway"
	healthH "ai-gateway/handlers/health"
	inventoryH "ai-gateway/handlers/inventory"
	measuresH "ai-gateway/handlers/measures"
	routingH "ai-gateway/handlers/routing"
	localdb "ai-gateway/internal/database"

	conformancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/conformance"
	gatewayv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/gateway"
	inventoryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inventory"
	measuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/measures"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

// AllEndpoints returns every domain's static endpoint descriptors in a
// stable order (system endpoints first, then domains alphabetically).
// The stable order is what makes the diff-exit-code CI check on
// .vrooli/endpoints.json meaningful.
func AllEndpoints() []module.EndpointDescriptor {
	out := make([]module.EndpointDescriptor, 0)
	out = append(out, healthH.Endpoints...)
	out = append(out, conformanceH.Endpoints...)
	out = append(out, gatewayH.Endpoints...)
	out = append(out, inventoryH.Endpoints...)
	out = append(out, measuresH.Endpoints...)
	out = append(out, routingH.Endpoints...)
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
	// Services narrows a shared proto file to the services this module mounts.
	// Empty preserves the legacy meaning of "all services in File".
	Services []protoreflect.Name
	Module   string
	File     protoreflect.FileDescriptor
}

// AllProtoFiles returns the proto FileDescriptor backing each
// Connect-mounted domain module, in registration order.
func AllProtoFiles() []ProtoFileEntry {
	return []ProtoFileEntry{
		{Module: "conformance", File: conformancev1.File_ai_gateway_v1_conformance_conformance_proto},
		{Module: "conformance", File: scenariovalidationv1.File_scenario_validation_v1_validation_proto, Services: []protoreflect.Name{"ScenarioValidationService"}},
		{Module: "gateway", File: gatewayv1.File_ai_gateway_v1_gateway_gateway_proto},
		{Module: "inventory", File: inventoryv1.File_ai_gateway_v1_inventory_inventory_proto},
		{Module: "measures", File: measuresv1.File_ai_gateway_v1_measures_measures_proto},
		{Module: "routing", File: routingv1.File_ai_gateway_v1_routing_routing_proto},
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
		apidb.SchemaProviderFunc(conformanceH.Schema),
		apidb.SchemaProviderFunc(gatewayH.Schema),
		apidb.SchemaProviderFunc(inventoryH.Schema),
		apidb.SchemaProviderFunc(routingH.Schema),
	}
}
