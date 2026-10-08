// Package livesearch is the CLI's live web-search command surface. It mirrors
// the API's Connect-RPC LiveSearchService one command per RPC, built from
// cli/manifest.json via cliapp.LoadFromManifest (the single source of truth for
// the command-line shape) with one handler per subcommand in handlers.go.
package livesearch

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

// GroupName is the manifest group name this package owns.
const GroupName = "search"

// CommandName is the single command within the search group. It doubles as the
// group's default subcommand so `web-search search <query>` dispatches to it.
const CommandName = "search"

// Register builds the live-search subcommand group from the embedded manifest
// and wires every Connect-RPC binding to a handler in handlers.go.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"LiveSearchService.Search": h.search,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("livesearch: load from manifest: %w", err)
	}
	// The live-search surface is a single verb: `web-search search <query>`.
	// Routing the group's lone command as the default subcommand lets the bare
	// `search <query>` form dispatch to it (the query positional is args[0]),
	// while `search help` still prints the group help.
	group.DefaultSubcommand = CommandName
	return group, nil
}
