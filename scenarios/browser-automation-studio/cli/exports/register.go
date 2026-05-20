// Package exports wires the BAS `exports` CLI group to the proto
// ExportsService via the generic protodispatch dispatcher. The manifest
// (cli/manifest.json) is the single source of truth for command shape.
//
// Part of Phase 10 of the BAS proto+Connect migration
// (plans:bas-migration-to-proto-connect-rpc).
package exports

import (
	"browser-automation-studio/cli/internal/protodispatch"
	"fmt"

	pbsvc "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/exports"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group this register package owns.
const GroupName = "exports"

// Register builds the exports subcommand group by binding every manifest
// command to a generic protodispatch handler resolved against the
// generated ExportsService descriptor.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	svc := pbsvc.File_browser_automation_studio_v1_exports_exports_proto.Services().ByName("ExportsService")
	if svc == nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("%s: ExportsService descriptor not found", GroupName)
	}
	bindings, err := protodispatch.Bindings(core, svc.FullName())
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("%s: %w", GroupName, err)
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("%s: load from manifest: %w", GroupName, err)
	}
	return group, nil
}
