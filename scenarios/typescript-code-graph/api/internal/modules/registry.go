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
// Adding a domain: append the new handler package's Endpoints,
// ProtoFileEntry, and Schema to the three lists below, then add the
// runtime Module(...) call to main.go's server.New(...) invocation.
package modules

import (
	"typescript-code-graph/internal/module"

	apidb "github.com/vrooli/api-core/database"
	"google.golang.org/protobuf/reflect/protoreflect"

	graphH "typescript-code-graph/handlers/graph"
	healthH "typescript-code-graph/handlers/health"
	rewriteH "typescript-code-graph/handlers/rewrite"
	localdb "typescript-code-graph/internal/database"

	graphproto "github.com/vrooli/vrooli/packages/proto/gen/go/typescript-code-graph/v1/graph"
)

// AllEndpoints returns every domain's static endpoint descriptors in a
// stable order (system endpoints first, then domains alphabetically).
// The stable order is what makes the diff-exit-code CI check on
// .vrooli/endpoints.json meaningful.
//
// The rewrite RPCs live on the same Connect service as graph_extract
// (they all hang off TypeScriptCodeGraphService) but their endpoint
// descriptors live in handlers/rewrite — they describe a distinct
// product domain. Appending them here is the only registration step
// the codegen needs.
func AllEndpoints() []module.EndpointDescriptor {
	out := make([]module.EndpointDescriptor, 0)
	out = append(out, healthH.Endpoints...)
	out = append(out, graphH.Endpoints...)
	out = append(out, rewriteH.Endpoints...)
	return out
}

// ProtoFileEntry pairs a domain module's name with the proto
// FileDescriptor whose RPCs that module exposes via Connect-RPC. The
// global parity test in registry_test.go walks every entry and asserts
// each rpc method in the FileDescriptor has exactly one matching
// EndpointDescriptor in AllEndpoints().
type ProtoFileEntry struct {
	Module string
	File   protoreflect.FileDescriptor
}

// AllProtoFiles returns the proto FileDescriptor backing each
// Connect-mounted domain module, in registration order.
//
// The template-inherited health domain is REST-only; product domains
// (graph, rewrite) land here as they ship.
func AllProtoFiles() []ProtoFileEntry {
	return []ProtoFileEntry{
		{Module: "graph", File: graphproto.File_typescript_code_graph_v1_graph_graph_proto},
	}
}

// AllSchemas returns every domain's schema provider plus the system
// schema (always first; cross-cutting infrastructure runs before any
// domain table). Consumed by main.go's database.EnsureSchemas call.
//
// Order matters: system → health → … (domains alphabetical). Postgres
// scenarios that put `CREATE EXTENSION ...` in system.sql rely on
// system running before any domain that references the extension.
func AllSchemas() []apidb.SchemaProvider {
	return []apidb.SchemaProvider{
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(healthH.Schema),
		apidb.SchemaProviderFunc(graphH.Schema),
		// rewrite is stateless in v1 — Schema() returns "" — but the
		// registry includes it so a future SQLite-backed PlanStore
		// (REQ-P1-002) drops in with a one-line change.
		apidb.SchemaProviderFunc(rewriteH.Schema),
	}
}
