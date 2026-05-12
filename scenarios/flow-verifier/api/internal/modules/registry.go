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
	"flow-verifier/internal/module"

	apidb "github.com/vrooli/api-core/database"
	"google.golang.org/protobuf/reflect/protoreflect"

	artifactsH "flow-verifier/handlers/artifacts"
	flowsH "flow-verifier/handlers/flows"
	healthH "flow-verifier/handlers/health"
	runsH "flow-verifier/handlers/runs"
	scenariosH "flow-verifier/handlers/scenarios"
	settingsH "flow-verifier/handlers/settings"
	verificationsH "flow-verifier/handlers/verifications"
	localdb "flow-verifier/internal/database"

	artifactsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/artifacts"
	flowsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/flows"
	runsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/runs"
	scenariosv1 "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/scenarios"
	settingsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/settings"
	verificationsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/verifications"
)

// AllEndpoints returns every domain's static endpoint descriptors in a
// stable order (system endpoints first, then domains alphabetically).
// The stable order is what makes the diff-exit-code CI check on
// .vrooli/endpoints.json meaningful.
func AllEndpoints() []module.EndpointDescriptor {
	out := make([]module.EndpointDescriptor, 0)
	out = append(out, artifactsH.Endpoints...)
	out = append(out, flowsH.Endpoints...)
	out = append(out, healthH.Endpoints...)
	out = append(out, runsH.Endpoints...)
	out = append(out, scenariosH.Endpoints...)
	out = append(out, settingsH.Endpoints...)
	out = append(out, verificationsH.Endpoints...)
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
func AllProtoFiles() []ProtoFileEntry {
	return []ProtoFileEntry{
		{Module: "artifacts", File: artifactsv1.File_flow_verifier_v1_artifacts_artifacts_proto},
		{Module: "flows", File: flowsv1.File_flow_verifier_v1_flows_flows_proto},
		{Module: "runs", File: runsv1.File_flow_verifier_v1_runs_runs_proto},
		{Module: "scenarios", File: scenariosv1.File_flow_verifier_v1_scenarios_scenarios_proto},
		{Module: "settings", File: settingsv1.File_flow_verifier_v1_settings_settings_proto},
		{Module: "verifications", File: verificationsv1.File_flow_verifier_v1_verifications_verifications_proto},
	}
}

// AllSchemas returns every domain's schema provider plus the system
// schema (always first; cross-cutting infrastructure runs before any
// domain table). Consumed by main.go's database.EnsureSchemas call.
//
// Order matters: system → flows → health → … (domains alphabetical).
// Postgres scenarios that put `CREATE EXTENSION ...` in system.sql rely
// on system running before any domain that references the extension.
func AllSchemas() []apidb.SchemaProvider {
	return []apidb.SchemaProvider{
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(artifactsH.Schema),
		apidb.SchemaProviderFunc(flowsH.Schema),
		apidb.SchemaProviderFunc(healthH.Schema),
		apidb.SchemaProviderFunc(runsH.Schema),
		apidb.SchemaProviderFunc(scenariosH.Schema),
		apidb.SchemaProviderFunc(settingsH.Schema),
		apidb.SchemaProviderFunc(verificationsH.Schema),
	}
}
