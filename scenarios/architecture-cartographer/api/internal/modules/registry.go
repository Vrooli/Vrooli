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
	"architecture-cartographer/internal/module"

	apidb "github.com/vrooli/api-core/database"
	"google.golang.org/protobuf/reflect/protoreflect"

	analyticsH "architecture-cartographer/handlers/analytics"
	applyH "architecture-cartographer/handlers/apply"
	conflictsH "architecture-cartographer/handlers/conflicts"
	domainsH "architecture-cartographer/handlers/domains"
	graphH "architecture-cartographer/handlers/graph"
	healthH "architecture-cartographer/handlers/health"
	signalsH "architecture-cartographer/handlers/signals"

	localdb "architecture-cartographer/internal/database"

	analyticsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/analytics"
	applyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/apply"
	conflictsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/conflicts"
	domainsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/domains"
	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/graph"
	signalsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/signals"
)

// AllEndpoints returns every domain's static endpoint descriptors in a
// stable order (system endpoints first, then domains alphabetically).
func AllEndpoints() []module.EndpointDescriptor {
	out := make([]module.EndpointDescriptor, 0)
	out = append(out, healthH.Endpoints...)
	out = append(out, analyticsH.Endpoints...)
	out = append(out, applyH.Endpoints...)
	out = append(out, conflictsH.Endpoints...)
	out = append(out, domainsH.Endpoints...)
	out = append(out, graphH.Endpoints...)
	out = append(out, signalsH.Endpoints...)
	return out
}

// ProtoFileEntry pairs a domain module's name with the proto
// FileDescriptor whose RPCs that module exposes via Connect-RPC.
type ProtoFileEntry struct {
	Module string
	File   protoreflect.FileDescriptor
}

// AllProtoFiles returns the proto FileDescriptor backing each
// Connect-mounted domain module, in registration order.
func AllProtoFiles() []ProtoFileEntry {
	return []ProtoFileEntry{
		{Module: "analytics", File: analyticsv1.File_architecture_cartographer_v1_analytics_analytics_proto},
		{Module: "apply", File: applyv1.File_architecture_cartographer_v1_apply_apply_proto},
		{Module: "conflicts", File: conflictsv1.File_architecture_cartographer_v1_conflicts_conflicts_proto},
		{Module: "domains", File: domainsv1.File_architecture_cartographer_v1_domains_domains_proto},
		{Module: "graph", File: graphv1.File_architecture_cartographer_v1_graph_graph_proto},
		{Module: "signals", File: signalsv1.File_architecture_cartographer_v1_signals_signals_proto},
	}
}

// AllSchemas returns every domain's schema provider plus the system
// schema (always first; cross-cutting infrastructure runs before any
// domain table). Consumed by main.go's database.EnsureSchemas call.
//
// Order matters: system → health → … (domains alphabetical). Schemas
// that return "" (signals — stateless) are also returned so the
// registry shape stays uniform.
func AllSchemas() []apidb.SchemaProvider {
	return []apidb.SchemaProvider{
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(healthH.Schema),
		apidb.SchemaProviderFunc(analyticsH.Schema),
		apidb.SchemaProviderFunc(applyH.Schema),
		apidb.SchemaProviderFunc(conflictsH.Schema),
		apidb.SchemaProviderFunc(domainsH.Schema),
		apidb.SchemaProviderFunc(graphH.Schema),
		apidb.SchemaProviderFunc(signalsH.Schema),
	}
}
