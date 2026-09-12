// Package snippets is the CLI surface for sender-owned reusable message text.
package snippets

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "snippet"

// Register builds the snippet command group from the manifest and binds every
// public SnippetsService method to its thin Connect-RPC handler.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"SnippetsService.ListSnippets":  h.list,
		"SnippetsService.UpsertSnippet": h.upsert,
		"SnippetsService.DeleteSnippet": h.delete,
		"SnippetsService.TouchSnippet":  h.touch,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("snippet: load from manifest: %w", err)
	}
	return group, nil
}
