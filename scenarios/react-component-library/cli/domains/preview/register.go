// Package preview is the CLI's live-preview-bundler surface. Mirrors
// the API's Connect-RPC PreviewService (proto schema at
// packages/proto/schemas/react-component-library/v1/preview). The
// `preview bundle <id>` verb prints the transpiled ES module to stdout
// for diagnostic use; the live-preview iframe consumes the same
// service via the HTTP harness route.
package preview

import (
	"github.com/vrooli/cli-core/cliapp"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	h := newHandlers(core)
	return cliapp.SubcommandGroup{
		Name:        "preview",
		Description: "Bundle components for the live-preview iframe",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{
				Name:        "bundle",
				Description: "Print the esbuild-transpiled ES module for a component",
				Args: cliapp.ArgSchema{
					Positionals: []cliapp.Positional{
						{Name: "id", Required: true, Description: "Component id"},
					},
				},
				RunCtx: h.bundle,
			},
		},
	}
}
