package snapshot

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "snapshot"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifest(manifest, GroupName, map[string]func(cliapp.RunContext) error{
		"SnapshotService.RunSnapshot":          h.run,
		"SnapshotService.ListSnapshots":        h.list,
		"SnapshotService.GetSnapshot":          h.get,
		"SnapshotService.ExportSnapshotReport": h.export,
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("snapshot: load from manifest: %w", err)
	}
	return group, nil
}
