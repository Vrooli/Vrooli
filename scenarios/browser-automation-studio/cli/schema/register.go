// Package schema wires the BAS `schema` CLI group to the proto
// SchemaService via the generic protodispatch dispatcher.
//
// Part of Phase 10 of the BAS proto+Connect migration
// (plans:bas-migration-to-proto-connect-rpc). The prior hand-coded
// `schema workflow`/`schema steps`/`schema node-types` handlers used
// generated Connect clients directly with custom rendering; this register
// replaces them with manifest-driven dispatch.
package schema

import (
	"fmt"


	schemav1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/schema"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group this register package owns.
const GroupName = "schema"

// Register builds the schema subcommand group by binding every manifest
// command to a generic protodispatch handler resolved against the
// generated SchemaService descriptor.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	svc := schemav1.File_browser_automation_studio_v1_schema_schema_proto.Services().ByName("SchemaService")
	if svc == nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("%s: SchemaService descriptor not found", GroupName)
	}
	bindings, err := cliapp.ProtoBindings(core, svc.FullName(), cliapp.ProtoBindingOptions{})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("%s: %w", GroupName, err)
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("%s: load from manifest: %w", GroupName, err)
	}
	return group, nil
}
