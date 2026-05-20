// Package ai wires the BAS `ai` CLI group to the proto AIService via the
// generic protodispatch dispatcher. AIService and VisionNavigationService
// share a single proto file (ai.proto); VisionNavigation is wired by the
// sibling vision_navigation package.
//
// Part of Phase 10 of the BAS proto+Connect migration
// (plans:bas-migration-to-proto-connect-rpc). The prior hand-coded
// `ai navigate`/`ai navigators` command surface has been replaced by the
// manifest-driven `ai` and `vision-navigation` subgroups; navigation
// commands now live under `vision-navigation start`/`abort`/`status`.
package ai

import (
	"browser-automation-studio/cli/internal/protodispatch"
	"fmt"

	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/ai"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group this register package owns.
const GroupName = "ai"

// Register builds the ai subcommand group by binding every manifest
// command to a generic protodispatch handler resolved against the
// generated AIService descriptor.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	svc := aiv1.File_browser_automation_studio_v1_ai_ai_proto.Services().ByName("AIService")
	if svc == nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("%s: AIService descriptor not found", GroupName)
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
