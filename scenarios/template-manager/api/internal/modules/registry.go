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
	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/module"

	apidb "github.com/vrooli/api-core/database"
	"google.golang.org/protobuf/reflect/protoreflect"

	debtH "github.com/vrooli/vrooli/scenarios/template-manager/api/handlers/debt"
	guidanceH "github.com/vrooli/vrooli/scenarios/template-manager/api/handlers/guidance"
	healthH "github.com/vrooli/vrooli/scenarios/template-manager/api/handlers/health"
	lifecycleH "github.com/vrooli/vrooli/scenarios/template-manager/api/handlers/lifecycle"
	measuresH "github.com/vrooli/vrooli/scenarios/template-manager/api/handlers/measures"
	monitorH "github.com/vrooli/vrooli/scenarios/template-manager/api/handlers/monitor"
	registryH "github.com/vrooli/vrooli/scenarios/template-manager/api/handlers/registry"
	resourceTemplateH "github.com/vrooli/vrooli/scenarios/template-manager/api/handlers/resource_template"
	validationH "github.com/vrooli/vrooli/scenarios/template-manager/api/handlers/validation"
	localdb "github.com/vrooli/vrooli/scenarios/template-manager/api/internal/database"

	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	debtv1 "github.com/vrooli/vrooli/packages/proto/gen/go/template-manager/v1/debt"
	guidancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/template-manager/v1/guidance"
	lifecyclev1 "github.com/vrooli/vrooli/packages/proto/gen/go/template-manager/v1/lifecycle"
	measuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/template-manager/v1/measures"
	monitorv1 "github.com/vrooli/vrooli/packages/proto/gen/go/template-manager/v1/monitor"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/template-manager/v1/registry"
	resourceTemplatev1 "github.com/vrooli/vrooli/packages/proto/gen/go/template-manager/v1/resource_template"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/template-manager/v1/validation"
)

// AllEndpoints returns every domain's static endpoint descriptors in a
// stable order (system endpoints first, then domains alphabetically).
// The stable order is what makes the diff-exit-code CI check on
// .vrooli/endpoints.json meaningful.
func AllEndpoints() []module.EndpointDescriptor {
	out := make([]module.EndpointDescriptor, 0)
	out = append(out, healthH.Endpoints...)
	out = append(out, registryH.Endpoints...)
	out = append(out, validationH.Endpoints...)
	out = append(out, debtH.Endpoints...)
	out = append(out, guidanceH.Endpoints...)
	out = append(out, lifecycleH.Endpoints...)
	out = append(out, measuresH.Endpoints...)
	out = append(out, monitorH.Endpoints...)
	out = append(out, resourceTemplateH.Endpoints...)
	out = append(out, validationH.ScenarioValidationEndpoints...)
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
	// Services optionally narrows the parity contract to the named
	// services when the proto file declares more services than this
	// module implements (e.g. opt-in provider contracts that sit
	// beside the base service in the same file). Empty means every
	// service in File must have full endpoint parity.
	Services []protoreflect.Name
}

// AllProtoFiles returns the proto FileDescriptor backing each
// Connect-mounted domain module, in registration order.
func AllProtoFiles() []ProtoFileEntry {
	return []ProtoFileEntry{
		{Module: "registry", File: registryv1.File_template_manager_v1_registry_registry_proto},
		{Module: "validation", File: validationv1.File_template_manager_v1_validation_validation_proto},
		{Module: "debt", File: debtv1.File_template_manager_v1_debt_debt_proto},
		{Module: "guidance", File: guidancev1.File_template_manager_v1_guidance_guidance_proto},
		{Module: "lifecycle", File: lifecyclev1.File_template_manager_v1_lifecycle_lifecycle_proto},
		{Module: "measures", File: measuresv1.File_template_manager_v1_measures_measures_proto},
		{Module: "monitor", File: monitorv1.File_template_manager_v1_monitor_monitor_proto},
		{Module: "resource_template", File: resourceTemplatev1.File_template_manager_v1_resource_template_resource_template_proto},
		// scenario-validation/v1 also declares DurableValidationRunService,
		// an opt-in provider contract this module does not implement;
		// parity is scoped to the static validation service it mounts.
		{Module: "scenario-validation", File: scenariovalidationv1.File_scenario_validation_v1_validation_proto, Services: []protoreflect.Name{"ScenarioValidationService"}},
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
		apidb.SchemaProviderFunc(registryH.Schema),
	}
}
