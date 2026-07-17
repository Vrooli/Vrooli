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
		"ComponentsService.CreateComponentVersion":       h.versionCreate,
		"ComponentsService.UpdateComponentManifest":      h.manifestUpdate,
		"ComponentsService.GetComponentContent":          h.contentGet,
		"ComponentsService.UpdateComponentContent":       h.contentSet,
		"ComponentsService.ListComponentVersions":        h.versions,
		"ComponentsService.GetComponentVersionContent":   h.showVersion,
		"ComponentsService.ListComponentExamples":        h.examples,
		"ComponentsService.ListDesignStyles":             h.styles,
		"ComponentsService.ValidateStyleFit":             h.validateStyleFit,
		"ComponentTestsService.RunComponentTest":         h.testRun,
		"ComponentTestsService.RerunComponentTest":       h.testRerun,
		"ComponentTestsService.GetComponentTestReport":   h.testShow,
		"ComponentTestsService.ListComponentTestReports": h.testList,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("components: load from manifest: %w", err)
	}
	return group, nil
}
