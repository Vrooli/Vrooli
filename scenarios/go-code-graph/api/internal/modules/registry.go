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
package modules

import (
	"go-code-graph/internal/module"

	apidb "github.com/vrooli/api-core/database"
	"google.golang.org/protobuf/reflect/protoreflect"

	graphH "go-code-graph/handlers/graph"
	healthH "go-code-graph/handlers/health"
	rewriteH "go-code-graph/handlers/rewrite"
	localdb "go-code-graph/internal/database"

	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/go-code-graph/v1/graph"
)

// AllEndpoints returns every domain's static endpoint descriptors in a
// stable order (system endpoints first, then domains alphabetically).
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
// Connect-mounted FileDescriptor. graph and rewrite are split at the
// handlers/<dom>/ layer but ride a single GoCodeGraphService declared
// in graph.proto, so this list has one entry covering both.
func AllProtoFiles() []ProtoFileEntry {
	return []ProtoFileEntry{
		{Module: "go_code_graph", File: graphv1.File_go_code_graph_v1_graph_graph_proto},
	}
}

// AllSchemas returns every domain's schema provider plus the system
// schema (always first; cross-cutting infrastructure runs before any
// domain table). Consumed by main.go's database.EnsureSchemas call.
//
// graph and rewrite are stateless in v1. Their Schema() helpers return
// "" — included here so a future stateful turn (REQ-P1-002 Operation
// Log) is a one-line schema swap.
func AllSchemas() []apidb.SchemaProvider {
	return []apidb.SchemaProvider{
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(healthH.Schema),
		apidb.SchemaProviderFunc(graphH.Schema),
		apidb.SchemaProviderFunc(rewriteH.Schema),
	}
}
