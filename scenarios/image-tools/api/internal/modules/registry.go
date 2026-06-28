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
	"image-tools/internal/module"

	apidb "github.com/vrooli/api-core/database"
	"google.golang.org/protobuf/reflect/protoreflect"

	adaptersH "image-tools/handlers/adapters"
	aiH "image-tools/handlers/ai"
	analysisH "image-tools/handlers/analysis"
	diffH "image-tools/handlers/diff"
	healthH "image-tools/handlers/health"
	jobsH "image-tools/handlers/jobs"
	looksH "image-tools/handlers/looks"
	modelsH "image-tools/handlers/models"
	opsH "image-tools/handlers/ops"
	safetyH "image-tools/handlers/safety"
	selectionH "image-tools/handlers/selection"
	localdb "image-tools/internal/database"
	internalmeasures "image-tools/internal/measures"

	adaptersv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/adapters"
	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ai"
	analysisv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/analysis"
	diffv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/diff"
	jobsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/jobs"
	looksv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/looks"
	modelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/models"
	opsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ops"
	safetyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/safety"
	selectionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/selection"
)

// AllEndpoints returns every domain's static endpoint descriptors in a
// stable order (system endpoints first, then domains alphabetically).
// The stable order is what makes the diff-exit-code CI check on
// .vrooli/endpoints.json meaningful.
func AllEndpoints() []module.EndpointDescriptor {
	out := make([]module.EndpointDescriptor, 0)
	out = append(out, healthH.Endpoints...)
	out = append(out, adaptersH.Endpoints...)
	out = append(out, aiH.Endpoints...)
	out = append(out, analysisH.Endpoints...)
	out = append(out, diffH.Endpoints...)
	out = append(out, jobsH.Endpoints...)
	out = append(out, looksH.Endpoints...)
	out = append(out, modelsH.Endpoints...)
	out = append(out, opsH.Endpoints...)
	out = append(out, safetyH.Endpoints...)
	out = append(out, selectionH.Endpoints...)
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
		{Module: "adapters", File: adaptersv1.File_image_tools_v1_adapters_adapters_proto},
		{Module: "ai", File: aiv1.File_image_tools_v1_ai_ai_proto},
		{Module: "analysis", File: analysisv1.File_image_tools_v1_analysis_analysis_proto},
		{Module: "diff", File: diffv1.File_image_tools_v1_diff_diff_proto},
		{Module: "jobs", File: jobsv1.File_image_tools_v1_jobs_jobs_proto},
		{Module: "looks", File: looksv1.File_image_tools_v1_looks_looks_proto},
		{Module: "models", File: modelsv1.File_image_tools_v1_models_models_proto},
		{Module: "ops", File: opsv1.File_image_tools_v1_ops_ops_proto},
		{Module: "safety", File: safetyv1.File_image_tools_v1_safety_safety_proto},
		{Module: "selection", File: selectionv1.File_image_tools_v1_selection_selection_proto},
	}
}

// AllSchemas returns every domain's schema provider plus the system
// schema (always first; cross-cutting infrastructure runs before any
// domain table). Consumed by main.go's database.EnsureSchemas call.
//
// Order matters: system → health → jobs → models → … (domains alphabetical).
// Postgres scenarios that put `CREATE EXTENSION ...` in system.sql rely
// on system running before any domain that references the extension.
func AllSchemas() []apidb.SchemaProvider {
	return []apidb.SchemaProvider{
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(healthH.Schema),
		apidb.SchemaProviderFunc(adaptersH.Schema),
		apidb.SchemaProviderFunc(aiH.Schema),
		apidb.SchemaProviderFunc(analysisH.Schema),
		apidb.SchemaProviderFunc(jobsH.Schema),
		apidb.SchemaProviderFunc(looksH.Schema),
		apidb.SchemaProviderFunc(internalmeasures.Schema),
		apidb.SchemaProviderFunc(modelsH.Schema),
		apidb.SchemaProviderFunc(opsH.Schema),
		apidb.SchemaProviderFunc(safetyH.Schema),
	}
}
