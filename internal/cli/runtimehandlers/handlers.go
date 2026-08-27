package runtimehandlers

import (
	"io"

	"github.com/vrooli/cli-core/cliapp"
	climanifest "github.com/vrooli/vrooli/cli"
	runtimeapp "github.com/vrooli/vrooli/internal/app/runtime"
	"github.com/vrooli/vrooli/internal/cli/manifestdispatch"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cli/runtimecli"
	"github.com/vrooli/vrooli/internal/cliout"
)

// HandlerDeps supplies the root-context values needed by runtime commands.
type HandlerDeps[C any] struct {
	Root        func(C) string
	Globals     func(C) rootcli.GlobalOptions
	Stdin       func(C) io.Reader
	Stdout      func(C) io.Writer
	Stderr      func(C) io.Writer
	HomeDir     func(C) (string, error)
	ResolveRoot func(C) (string, error)
	Version     func(C) string
}

type runtimeService struct {
	run func([]string) error
}

// RootHandler dispatches `vrooli runtime` through the runtime application.
func RootHandler[C any](deps HandlerDeps[C]) rootcli.Handler[C] {
	return rootcli.BindService(deps.Stdout,
		func(ctx C) (cliout.Format, runtimeService, error) {
			commandCtx := &runtimecli.Context{Root: deps.Root(ctx), Globals: deps.Globals(ctx), Stdin: deps.Stdin(ctx), Stdout: deps.Stdout(ctx), Stderr: deps.Stderr(ctx), HomeDirFn: func() (string, error) { return deps.HomeDir(ctx) }}
			app := &runtimeapp.App{Version: deps.Version(ctx), ResolveRootFn: func() (string, error) { return deps.ResolveRoot(ctx) }}
			return cliout.FormatHuman, runtimeService{run: func(args []string) error {
				if len(args) == 0 || manifestdispatch.WantsHelp(args) {
					return runtimecli.Run(app, commandCtx, args)
				}
				if args[0] == "recovery" {
					return runRecoveryManifest(app, commandCtx, args[1:], deps.Stdout(ctx), deps.Stderr(ctx))
				}
				group, err := cliapp.LoadFromManifest(climanifest.Bytes(), "supervisor", runtimeBindings(app, commandCtx, []string{"supervisor"}, []string{"run", "status", "install", "uninstall"}))
				if err != nil {
					return err
				}
				core := cliapp.NewApp(cliapp.AppOptions{Name: "vrooli runtime", SubcommandGroups: []cliapp.SubcommandGroup{group}})
				return core.RunWithWriters(manifestdispatch.WithJSON(args, commandCtx.Globals.JSON), deps.Stdout(ctx), deps.Stderr(ctx))
			}}, nil
		},
		func(_ C, args []string) ([]string, error) { return args, nil },
		func(service runtimeService, args []string) (struct{}, error) { return struct{}{}, service.run(args) },
		func(io.Writer, cliout.Format, struct{}) error { return nil },
	)
}

func runRecoveryManifest(app *runtimeapp.App, ctx *runtimecli.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 || manifestdispatch.WantsHelp(args) {
		return app.Run(ctx, append([]string{"recovery"}, args...))
	}
	if args[0] == "inspect" {
		return app.Run(ctx, append([]string{"recovery"}, args...))
	}
	group, err := cliapp.LoadFromManifest(climanifest.Bytes(), "runtime/recovery/policy", runtimeBindings(app, ctx, []string{"recovery", "policy"}, []string{"set", "list"}))
	if err != nil {
		return err
	}
	core := cliapp.NewApp(cliapp.AppOptions{Name: "vrooli runtime recovery", SubcommandGroups: []cliapp.SubcommandGroup{group}})
	return core.RunWithWriters(manifestdispatch.WithJSON(args, ctx.Globals.JSON), stdout, stderr)
}

func runtimeBindings(app *runtimeapp.App, ctx *runtimecli.Context, prefix []string, names []string) map[string]func(cliapp.RunContext) error {
	bindings := make(map[string]func(cliapp.RunContext) error, len(names))
	for _, name := range names {
		command := append(append([]string(nil), prefix...), name)
		bindings[name] = func(command []string) func(cliapp.RunContext) error {
			return func(runCtx cliapp.RunContext) error {
				commandCtx := *ctx
				commandCtx.Globals.JSON = commandCtx.Globals.JSON || runCtx.JSON()
				return app.Run(&commandCtx, append(command, manifestdispatch.LegacyArgs(runCtx)...))
			}
		}(command)
	}
	return bindings
}
