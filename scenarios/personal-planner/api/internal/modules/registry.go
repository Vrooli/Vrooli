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
	"personal-planner/internal/module"

	apidb "github.com/vrooli/api-core/database"
	"google.golang.org/protobuf/reflect/protoreflect"

	calendarH "personal-planner/handlers/calendar"
	focusH "personal-planner/handlers/focus"
	goalsH "personal-planner/handlers/goals"
	healthH "personal-planner/handlers/health"
	integrationsH "personal-planner/handlers/integrations"
	reviewH "personal-planner/handlers/review"
	workH "personal-planner/handlers/work"
	workspaceH "personal-planner/handlers/workspace"
	localdb "personal-planner/internal/database"

	calendarv1 "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/calendar"
	focusv1 "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/focus"
	goalsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/goals"
	integrationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/integrations"
	reviewv1 "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/review"
	workv1 "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/work"
	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/workspace"
)

// AllEndpoints returns every domain's static endpoint descriptors in a
// stable order (system endpoints first, then domains alphabetically).
// The stable order is what makes the diff-exit-code CI check on
// .vrooli/endpoints.json meaningful.
func AllEndpoints() []module.EndpointDescriptor {
	out := make([]module.EndpointDescriptor, 0)
	out = append(out, healthH.Endpoints...)
	out = append(out, focusH.Endpoints...)
	out = append(out, calendarH.Endpoints...)
	out = append(out, goalsH.Endpoints...)
	out = append(out, integrationsH.Endpoints...)
	out = append(out, reviewH.Endpoints...)
	out = append(out, workH.Endpoints...)
	out = append(out, workspaceH.Endpoints...)
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
		{Module: "focus", File: focusv1.File_personal_planner_v1_focus_focus_proto},
		{Module: "calendar", File: calendarv1.File_personal_planner_v1_calendar_calendar_proto},
		{Module: "goals", File: goalsv1.File_personal_planner_v1_goals_goals_proto},
		{Module: "integrations", File: integrationsv1.File_personal_planner_v1_integrations_integrations_proto},
		{Module: "review", File: reviewv1.File_personal_planner_v1_review_review_proto},
		{Module: "work", File: workv1.File_personal_planner_v1_work_work_proto},
		{Module: "workspace", File: workspacev1.File_personal_planner_v1_workspace_workspace_proto},
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
		apidb.SchemaProviderFunc(focusH.Schema),
		apidb.SchemaProviderFunc(calendarH.Schema),
		apidb.SchemaProviderFunc(goalsH.Schema),
		apidb.SchemaProviderFunc(integrationsH.Schema),
		apidb.SchemaProviderFunc(workH.Schema),
		apidb.SchemaProviderFunc(workspaceH.Schema),
		apidb.SchemaProviderFunc(reviewH.Schema),
	}
}
