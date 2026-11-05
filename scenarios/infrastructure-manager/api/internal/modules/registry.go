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
	"github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/internal/module"

	capsH "github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/handlers/capabilities"
	conditionH "github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/handlers/condition"
	coverageH "github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/handlers/coverage"
	focusH "github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/handlers/focus"
	ladderH "github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/handlers/ladder"
	portabilityH "github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/handlers/portability"

	apidb "github.com/vrooli/api-core/database"
	"google.golang.org/protobuf/reflect/protoreflect"

	healthH "github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/handlers/health"
	localdb "github.com/vrooli/vrooli/scenarios/infrastructure-manager/api/internal/database"

	conditionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/infrastructure-manager/v1/condition"
	coveragev1 "github.com/vrooli/vrooli/packages/proto/gen/go/infrastructure-manager/v1/coverage"
	focusv1 "github.com/vrooli/vrooli/packages/proto/gen/go/infrastructure-manager/v1/focus"
	ladderv1 "github.com/vrooli/vrooli/packages/proto/gen/go/infrastructure-manager/v1/ladder"
	portabilityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/infrastructure-manager/v1/portability"
)

// AllEndpoints returns every domain's static endpoint descriptors in a
// stable order (system endpoints first, then domains alphabetically).
// The stable order is what makes the diff-exit-code CI check on
// .vrooli/endpoints.json meaningful.
func AllEndpoints() []module.EndpointDescriptor {
	out := make([]module.EndpointDescriptor, 0)
	out = append(out, healthH.Endpoints...)
	out = append(out, capsH.Endpoints...)
	out = append(out, conditionH.Endpoints...)
	out = append(out, coverageH.Endpoints...)
	out = append(out, focusH.Endpoints...)
	out = append(out, ladderH.Endpoints...)
	out = append(out, portabilityH.Endpoints...)
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
		{Module: "condition", File: conditionv1.File_infrastructure_manager_v1_condition_condition_proto},
		{Module: "coverage", File: coveragev1.File_infrastructure_manager_v1_coverage_coverage_proto},
		{Module: "focus", File: focusv1.File_infrastructure_manager_v1_focus_focus_proto},
		{Module: "ladder", File: ladderv1.File_infrastructure_manager_v1_ladder_ladder_proto},
		{Module: "portability", File: portabilityv1.File_infrastructure_manager_v1_portability_portability_proto},
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
		apidb.SchemaProviderFunc(conditionH.Schema),
		apidb.SchemaProviderFunc(focusH.Schema),
	}
}
