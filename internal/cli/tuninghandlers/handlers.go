// Package tuninghandlers binds the operator-visible tuning registry to the
// manifest-owned CLI command surface.
package tuninghandlers

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/vrooli/cli-core/cliapp"
	climanifest "github.com/vrooli/vrooli/cli"
	"github.com/vrooli/vrooli/internal/cli/manifestdispatch"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/tuning"
)

// HandlerDeps supplies the command streams and root-global options.
type HandlerDeps[C any] struct {
	Globals func(C) rootcli.GlobalOptions
	Stdout  func(C) io.Writer
	Stderr  func(C) io.Writer
}

// RegisteredCommandPaths returns the manifest-bound tuning leaves.
func RegisteredCommandPaths() []string { return []string{"tuning list"} }

// RootHandler builds `vrooli tuning` from cli/manifest.json.
func RootHandler[C any](deps HandlerDeps[C]) rootcli.Handler[C] {
	return func(ctx C, args []string) error {
		bindings := map[string]func(cliapp.RunContext) error{
			"list": func(runCtx cliapp.RunContext) error {
				levers := tuning.Levers()
				if runCtx.JSON() || deps.Globals(ctx).JSON {
					return writeJSON(deps.Stdout(ctx), levers)
				}
				for _, lever := range levers {
					_, _ = fmt.Fprintf(deps.Stdout(ctx), "%s\t%s\t%s\t%s\t%s\t%s\n", lever.Name, lever.Kind, lever.Environment, lever.CompiledDefault, lever.ResolvedValue, lever.Source)
				}
				return nil
			},
		}
		group, err := cliapp.LoadFromManifest(climanifest.Bytes(), "tuning", bindings)
		if err != nil {
			return err
		}
		core := cliapp.NewApp(cliapp.AppOptions{Name: "vrooli tuning", Commands: []cliapp.CommandGroup{{Commands: group.Subcommands}}})
		return core.RunWithWriters(manifestdispatch.WithJSON(args, deps.Globals(ctx).JSON), deps.Stdout(ctx), deps.Stderr(ctx))
	}
}

func writeJSON(w io.Writer, levers []tuning.Lever) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(levers)
}
