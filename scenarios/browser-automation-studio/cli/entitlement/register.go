// Package entitlement wires the BAS `entitlement` CLI group to the proto
// EntitlementService via the generic protodispatch dispatcher. The manifest
// (cli/manifest.json) is the single source of truth for command shape.
//
// Part of Phase 10 of the BAS proto+Connect migration
// (plans:bas-migration-to-proto-connect-rpc).
package entitlement

import (
	"fmt"

	pbsvc "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/entitlement"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group this register package owns.
const GroupName = "entitlement"

// Register builds the entitlement subcommand group by binding every manifest
// command to a generic protodispatch handler resolved against the
// generated EntitlementService descriptor.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	svc := pbsvc.File_browser_automation_studio_v1_entitlement_entitlement_proto.Services().ByName("EntitlementService")
	if svc == nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("%s: EntitlementService descriptor not found", GroupName)
	}
	group, err := cliapp.LoadProtoGroupFromManifest(core, svc.FullName(), manifest, GroupName, cliapp.ProtoBindingOptions{})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("%s: load from manifest: %w", GroupName, err)
	}
	return group, nil
}
