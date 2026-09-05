package domains

import (
	"vrooli-events/cli/domains/capture"
	"vrooli-events/cli/domains/events"
	"vrooli-events/cli/domains/store"

	"github.com/vrooli/cli-core/cliapp"
)

func CommandGroups(core *cliapp.ScenarioApp) []cliapp.CommandGroup {
	return []cliapp.CommandGroup{
		capture.Register(core),
		events.Register(core),
		store.Register(core),
	}
}

// SubcommandGroups assembles the manifest-backed primitive evidence artifact.
// The live app uses CommandGroups because the events manifest is intentionally
// flat; this helper keeps evidence generation coupled to the same manifest and
// handler rather than duplicating primitive metadata in the test.
func SubcommandGroups(core *cliapp.ScenarioApp, manifest []byte) ([]cliapp.SubcommandGroup, error) {
	group, err := events.RegisterManifest(core, manifest)
	if err != nil {
		return nil, err
	}
	return []cliapp.SubcommandGroup{group}, nil
}
