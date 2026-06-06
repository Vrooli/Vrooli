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
	"data-backup-manager/internal/module"

	apidb "github.com/vrooli/api-core/database"
	"google.golang.org/protobuf/reflect/protoreflect"

	auditsH "data-backup-manager/handlers/audits"
	coverageH "data-backup-manager/handlers/coverage"
	destinationsH "data-backup-manager/handlers/destinations"
	discoveryH "data-backup-manager/handlers/discovery"
	healthH "data-backup-manager/handlers/health"
	plansH "data-backup-manager/handlers/plans"
	restoresH "data-backup-manager/handlers/restores"
	runsH "data-backup-manager/handlers/runs"
	safetyH "data-backup-manager/handlers/safety"
	targetsH "data-backup-manager/handlers/targets"
	localdb "data-backup-manager/internal/database"

	auditsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/audits"
	coveragev1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/coverage"
	destinationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/destinations"
	discoveryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/discovery"
	plansv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/plans"
	restoresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/restores"
	runsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/runs"
	safetyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/safety"
	targetsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/data-backup-manager/v1/targets"
)

// AllEndpoints returns every domain's static endpoint descriptors in a
// stable order (system endpoints first, then domains alphabetically).
// The stable order is what makes the diff-exit-code CI check on
// .vrooli/endpoints.json meaningful.
func AllEndpoints() []module.EndpointDescriptor {
	out := make([]module.EndpointDescriptor, 0)
	out = append(out, healthH.Endpoints...)
	out = append(out, auditsH.Endpoints...)
	out = append(out, coverageH.Endpoints...)
	out = append(out, destinationsH.Endpoints...)
	out = append(out, discoveryH.Endpoints...)
	out = append(out, plansH.Endpoints...)
	out = append(out, restoresH.Endpoints...)
	out = append(out, runsH.Endpoints...)
	out = append(out, safetyH.Endpoints...)
	out = append(out, targetsH.Endpoints...)
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
		{Module: "audits", File: auditsv1.File_data_backup_manager_v1_audits_audits_proto},
		{Module: "coverage", File: coveragev1.File_data_backup_manager_v1_coverage_coverage_proto},
		{Module: "destinations", File: destinationsv1.File_data_backup_manager_v1_destinations_destinations_proto},
		{Module: "discovery", File: discoveryv1.File_data_backup_manager_v1_discovery_discovery_proto},
		{Module: "plans", File: plansv1.File_data_backup_manager_v1_plans_plans_proto},
		{Module: "restores", File: restoresv1.File_data_backup_manager_v1_restores_restores_proto},
		{Module: "runs", File: runsv1.File_data_backup_manager_v1_runs_runs_proto},
		{Module: "safety", File: safetyv1.File_data_backup_manager_v1_safety_safety_proto},
		{Module: "targets", File: targetsv1.File_data_backup_manager_v1_targets_targets_proto},
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
		apidb.SchemaProviderFunc(auditsH.Schema),
		apidb.SchemaProviderFunc(destinationsH.Schema),
		apidb.SchemaProviderFunc(discoveryH.Schema),
		apidb.SchemaProviderFunc(plansH.Schema),
		apidb.SchemaProviderFunc(restoresH.Schema),
		apidb.SchemaProviderFunc(runsH.Schema),
		apidb.SchemaProviderFunc(targetsH.Schema),
	}
}
