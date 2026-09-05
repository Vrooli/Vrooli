// Package corpus hosts the `audio-tools corpus ...` subtree — CRUD over the
// speech-eval clip corpus. The command surface is declared in
// cli/manifest.json (the single source of truth); Register binds each
// CorpusService method to a handler in handlers.go.
package corpus

import (
	"github.com/vrooli/cli-core/cliapp"

	"audio-tools/cli/internal/climanifest"
)

// GroupName is the manifest group name this package owns.
const GroupName = "corpus"

// Register builds the corpus subcommand group from the embedded manifest.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"CorpusService.ListClips":  h.list,
		"CorpusService.CreateClip": h.importClip,
		"CorpusService.GetClip":    h.get,
		"CorpusService.DeleteClip": h.delete,
	}
	return climanifest.LoadGroup(manifest, GroupName, bindings)
}
