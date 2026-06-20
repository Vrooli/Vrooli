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
	"structure-health/internal/module"

	apidb "github.com/vrooli/api-core/database"
	"google.golang.org/protobuf/reflect/protoreflect"

	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	fleetv1 "github.com/vrooli/vrooli/packages/proto/gen/go/structure-health/v1/fleet"
	perfv1 "github.com/vrooli/vrooli/packages/proto/gen/go/structure-health/v1/perf"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/structure-health/v1/validation"

	fleetH "structure-health/handlers/fleet"
	healthH "structure-health/handlers/health"
	perfH "structure-health/handlers/perf"
	validationH "structure-health/handlers/validation"
	localdb "structure-health/internal/database"
)

// AllEndpoints returns every domain's static endpoint descriptors in a
// stable order (system endpoints first, then domains alphabetically).
// The stable order is what makes the diff-exit-code CI check on
// .vrooli/endpoints.json meaningful.
func AllEndpoints() []module.EndpointDescriptor {
	out := make([]module.EndpointDescriptor, 0)
	out = append(out, healthH.Endpoints...)
	out = append(out, fleetH.Endpoints...)
	out = append(out, perfH.Endpoints...)
	out = append(out, validationH.Endpoints...)
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
		{Module: "fleet", File: fleetv1.File_structure_health_v1_fleet_fleet_proto},
		{Module: "perf", File: perfv1.File_structure_health_v1_perf_perf_proto},
		{Module: "validation", File: validationv1.File_structure_health_v1_validation_validation_proto},
		{Module: "validation", File: scenariovalidationv1.File_scenario_validation_v1_validation_proto},
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
		apidb.SchemaProviderFunc(fleetH.Schema),
		apidb.SchemaProviderFunc(perfH.Schema),
		apidb.SchemaProviderFunc(validationH.Schema),
	}
}
