package capabilityhandlers

import (
	"io"

	"github.com/vrooli/cli-core/cliapp"
	climanifest "github.com/vrooli/vrooli/cli"
	capabilityapp "github.com/vrooli/vrooli/internal/app/capability"
	"github.com/vrooli/vrooli/internal/cli/capabilitycli"
	"github.com/vrooli/vrooli/internal/cli/manifestdispatch"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
)

// HandlerDeps supplies root-context values needed by capability commands.
type HandlerDeps[C any] struct {
	Root    func(C) string
	Globals func(C) rootcli.GlobalOptions
	Stdin   func(C) io.Reader
	Stdout  func(C) io.Writer
	Stderr  func(C) io.Writer
}

// RootHandler dispatches `vrooli capability` through the capability app.
func RootHandler[C any](deps HandlerDeps[C]) rootcli.Handler[C] {
	return func(ctx C, args []string) error {
		commandCtx := &capabilitycli.Context{Root: deps.Root(ctx), Globals: deps.Globals(ctx), Stdin: deps.Stdin(ctx), Stdout: deps.Stdout(ctx), Stderr: deps.Stderr(ctx)}
		app := &capabilityapp.App{}
		if len(args) == 0 || manifestdispatch.WantsHelp(args) {
			return capabilitycli.Run(app, commandCtx, args)
		}
		if args[0] != "ledger" && args[0] != "fleet" {
			return capabilitycli.Run(app, commandCtx, args)
		}
		group, err := cliapp.LoadFromManifest(climanifest.Bytes(), "capability", capabilityBindings(app, commandCtx, []string{"ledger", "fleet"}))
		if err != nil {
			return err
		}
		core := cliapp.NewApp(cliapp.AppOptions{Name: "vrooli capability", Commands: []cliapp.CommandGroup{{Commands: group.Subcommands}}})
		return core.RunWithWriters(manifestdispatch.WithJSON(args, commandCtx.Globals.JSON), deps.Stdout(ctx), deps.Stderr(ctx))
	}
}

func capabilityBindings(app *capabilityapp.App, ctx *capabilitycli.Context, names []string) map[string]func(cliapp.RunContext) error {
	bindings := make(map[string]func(cliapp.RunContext) error, len(names))
	for _, name := range names {
		command := name
		bindings[name] = func(command string) func(cliapp.RunContext) error {
			return func(runCtx cliapp.RunContext) error {
				commandCtx := *ctx
				commandCtx.Globals.JSON = commandCtx.Globals.JSON || runCtx.JSON()
				return app.Run(&commandCtx, append([]string{command}, manifestdispatch.LegacyArgs(runCtx)...))
			}
		}(command)
	}
	return bindings
}
