package hosthandlers

import (
	"io"

	"github.com/vrooli/cli-core/cliapp"
	climanifest "github.com/vrooli/vrooli/cli"
	hostapp "github.com/vrooli/vrooli/internal/app/host"
	"github.com/vrooli/vrooli/internal/cli/hostcli"
	"github.com/vrooli/vrooli/internal/cli/manifestdispatch"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
)

// HandlerDeps supplies the root-context values needed by host commands.
type HandlerDeps[C any] struct {
	Root    func(C) string
	Globals func(C) rootcli.GlobalOptions
	Stdout  func(C) io.Writer
	Stderr  func(C) io.Writer
}

// RootHandler dispatches `vrooli host` through the host application.
func RootHandler[C any](deps HandlerDeps[C]) rootcli.Handler[C] {
	return func(ctx C, args []string) error {
		commandCtx := &hostcli.Context{Root: deps.Root(ctx), Globals: deps.Globals(ctx), Stdout: deps.Stdout(ctx), Stderr: deps.Stderr(ctx)}
		app := &hostapp.App{}
		if len(args) == 0 || manifestdispatch.WantsHelp(args) {
			return hostcli.Run(app, commandCtx, args)
		}
		group, err := cliapp.LoadFromManifest(climanifest.Bytes(), "host", hostBindings(app, commandCtx, []string{"inventory", "install", "safeguard", "volume", "storage"}))
		if err != nil {
			return err
		}
		core := cliapp.NewApp(cliapp.AppOptions{Name: "vrooli host", Commands: []cliapp.CommandGroup{{Commands: group.Subcommands}}})
		return core.RunWithWriters(manifestdispatch.WithJSON(args, commandCtx.Globals.JSON), deps.Stdout(ctx), deps.Stderr(ctx))
	}
}

// WorkloadHandler dispatches `vrooli workload` through the host application.
func WorkloadHandler[C any](deps HandlerDeps[C]) rootcli.Handler[C] {
	return func(ctx C, args []string) error {
		app := &hostapp.App{}
		return hostcli.RunWorkload(app, &hostcli.Context{Root: deps.Root(ctx), Globals: deps.Globals(ctx), Stdout: deps.Stdout(ctx), Stderr: deps.Stderr(ctx)}, args)
	}
}

func hostBindings(app *hostapp.App, ctx *hostcli.Context, names []string) map[string]func(cliapp.RunContext) error {
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
