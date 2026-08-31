// Package components is the CLI's component-registry surface. Mirrors
// the API's Connect-RPC ComponentsService. Command surface loads from
// cli/manifest.json via cliapp.LoadFromManifest.
package components

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "components"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"ComponentsService.IndexComponents":              h.index,
		"ComponentsService.ListComponents":               h.list,
		"ComponentsService.GetComponent":                 h.get,
		"ComponentsService.GetComponentByLibraryId":      h.getByLibraryID,
		"ComponentsService.InitializeComponent":          h.init,
		"ComponentsService.IngestComponent":              h.ingest,
		"ComponentsService.BeginComponentVersion":        h.versionBegin,
		"ComponentsService.CheckComponentVersion":        h.versionCheck,
		"ComponentsService.PublishComponentVersion":      h.versionPublish,
		"ComponentsService.CreateComponentVersion":       h.versionCreate,
		"ComponentsService.UpdateComponentManifest":      h.manifestUpdate,
		"ComponentsService.GetComponentContent":          h.contentGet,
		"ComponentsService.UpdateComponentContent":       h.contentSet,
		"ComponentsService.ListComponentVersions":        h.versions,
		"ComponentsService.GetComponentVersionContent":   h.showVersion,
		"ComponentsService.ListComponentStories":         h.stories,
		"ComponentsService.ListDesignStyles":             h.styles,
		"ComponentsService.ValidateStyleFit":             h.validateStyleFit,
		"ComponentTestsService.RunComponentTest":         h.testRun,
		"ComponentTestsService.RerunComponentTest":       h.testRerun,
		"ComponentTestsService.GetComponentTestReport":   h.testShow,
		"ComponentTestsService.ListComponentTestReports": h.testList,
		"ComponentTestsService.SweepComponentTests":      h.sweep,
		// Local binding: the manifest declares this command with
		// binding.kind "local" and no handler name, so the loader keys it by
		// the command name. It must be registered here like any other — a
		// manifest command with no entry in this map is a startup panic, not
		// a missing subcommand.
		"republish-dependents": h.republishDependents,
		"migrate-specifiers":   h.migrateSpecifiers,
		"republish-plan":       h.republishPlan,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("components: load from manifest: %w", err)
	}
	return group, nil
}
