// Package session_profiles wires the BAS `session-profiles` CLI group to the proto
// SessionProfilesService via the generic protodispatch dispatcher. The manifest
// (cli/manifest.json) is the single source of truth for command shape.
//
// Part of Phase 10 of the BAS proto+Connect migration
// (plans:bas-migration-to-proto-connect-rpc).
package session_profiles

import (
	"fmt"

	"browser-automation-studio/cli/internal/protodispatch"

	pbsvc "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/session_profiles"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group this register package owns.
const GroupName = "session-profiles"

// Register builds the session-profiles subcommand group by binding every manifest
// command to a generic protodispatch handler resolved against the
// generated SessionProfilesService descriptor.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	svc := pbsvc.File_browser_automation_studio_v1_session_profiles_session_profiles_proto.Services().ByName("SessionProfilesService")
	if svc == nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("%s: SessionProfilesService descriptor not found", GroupName)
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
