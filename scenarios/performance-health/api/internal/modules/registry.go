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
	"performance-health/internal/module"

	apidb "github.com/vrooli/api-core/database"
	"google.golang.org/protobuf/reflect/protoreflect"

	analysisH "performance-health/handlers/analysis"
	auditH "performance-health/handlers/audit"
	benchmarkH "performance-health/handlers/benchmark"
	budgetsH "performance-health/handlers/budgets"
	fleetH "performance-health/handlers/fleet"
	healthH "performance-health/handlers/health"
	lighthouseH "performance-health/handlers/lighthouse"
	startupH "performance-health/handlers/startup"
	trendH "performance-health/handlers/trend"
	validationH "performance-health/handlers/validation"
	localdb "performance-health/internal/database"

	analysisv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/analysis"
	auditv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/audit"
	benchmarkv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/benchmark"
	budgetsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/budgets"
	fleetv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/fleet"
	lighthousev1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/lighthouse"
	readinessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/readiness"
	startupv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/startup"
	trendv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/trend"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

// AllEndpoints returns every domain's static endpoint descriptors in a
// stable order (system endpoints first, then domains alphabetically).
// The stable order is what makes the diff-exit-code CI check on
// .vrooli/endpoints.json meaningful.
func AllEndpoints() []module.EndpointDescriptor {
	out := make([]module.EndpointDescriptor, 0)
	out = append(out, healthH.Endpoints...)
	out = append(out, analysisH.Endpoints...)
	out = append(out, auditH.Endpoints...)
	out = append(out, benchmarkH.Endpoints...)
	out = append(out, budgetsH.Endpoints...)
	out = append(out, fleetH.Endpoints...)
	out = append(out, lighthouseH.Endpoints...)
	out = append(out, startupH.Endpoints...)
	out = append(out, trendH.Endpoints...)
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
		{Module: "analysis", File: analysisv1.File_performance_health_v1_analysis_analysis_proto},
		{Module: "audit", File: auditv1.File_performance_health_v1_audit_audit_proto},
		{Module: "benchmark", File: benchmarkv1.File_performance_health_v1_benchmark_benchmark_proto},
		{Module: "budgets", File: budgetsv1.File_performance_health_v1_budgets_budgets_proto},
		{Module: "fleet", File: fleetv1.File_performance_health_v1_fleet_fleet_proto},
		{Module: "lighthouse", File: lighthousev1.File_performance_health_v1_lighthouse_lighthouse_proto},
		{Module: "startup", File: startupv1.File_performance_health_v1_startup_startup_proto},
		{Module: "trend", File: trendv1.File_performance_health_v1_trend_trend_proto},
		{Module: "validation", File: readinessv1.File_performance_health_v1_readiness_readiness_proto},
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
		apidb.SchemaProviderFunc(analysisH.Schema),
		apidb.SchemaProviderFunc(auditH.Schema),
		apidb.SchemaProviderFunc(benchmarkH.Schema),
		apidb.SchemaProviderFunc(budgetsH.Schema),
		apidb.SchemaProviderFunc(fleetH.Schema),
		apidb.SchemaProviderFunc(lighthouseH.Schema),
		apidb.SchemaProviderFunc(startupH.Schema),
		apidb.SchemaProviderFunc(trendH.Schema),
		apidb.SchemaProviderFunc(validationH.Schema),
	}
}
