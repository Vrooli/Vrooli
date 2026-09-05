package capabilityhandlers

import (
	"io"

	"github.com/vrooli/cli-core/cliapp"
	climanifest "github.com/vrooli/vrooli/cli"
	capabilityapp "github.com/vrooli/vrooli/internal/app/capability"
	"github.com/vrooli/vrooli/internal/cli/capabilitycli"
	"github.com/vrooli/vrooli/internal/cli/manifestdispatch"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cliout"
)

// HandlerDeps supplies root-context values needed by capability commands.
type HandlerDeps[C any] struct {
	Root    func(C) string
	Globals func(C) rootcli.GlobalOptions
	Stdin   func(C) io.Reader
	Stdout  func(C) io.Writer
	Stderr  func(C) io.Writer
}

type capabilityService struct {
	run func([]string) error
}

var capabilityCommandNames = []string{"ledger", "fleet"}

// RegisteredCommandPaths returns the child paths bound by the capability handler.
func RegisteredCommandPaths() []string {
	paths := make([]string, 0, len(capabilityCommandNames))
	for _, name := range capabilityCommandNames {
		paths = append(paths, "capability "+name)
	}
	return paths
}

// RootHandler dispatches `vrooli capability` through the capability app.
func RootHandler[C any](deps HandlerDeps[C]) rootcli.Handler[C] {
	return rootcli.BindService(deps.Stdout,
		func(C) (cliout.Format, error) { return cliout.FormatHuman, nil },
		func(ctx C, _ cliout.Format) (capabilityService, error) {
			commandCtx := &capabilitycli.Context{Root: deps.Root(ctx), Globals: deps.Globals(ctx), Stdin: deps.Stdin(ctx), Stdout: deps.Stdout(ctx), Stderr: deps.Stderr(ctx)}
			app := &capabilityapp.App{}
			return capabilityService{run: func(args []string) error {
				if len(args) == 0 || manifestdispatch.WantsHelp(args) {
					return capabilitycli.Run(app, commandCtx, args)
				}
				if args[0] != "ledger" && args[0] != "fleet" {
					return capabilitycli.Run(app, commandCtx, args)
				}
				group, err := cliapp.LoadFromManifest(climanifest.Bytes(), "capability", capabilityBindings(app, commandCtx, capabilityCommandNames))
				if err != nil {
					return err
				}
				core := cliapp.NewApp(cliapp.AppOptions{Name: "vrooli capability", Commands: []cliapp.CommandGroup{{Commands: group.Subcommands}}})
				return core.RunWithWriters(manifestdispatch.WithJSON(args, commandCtx.Globals.JSON), deps.Stdout(ctx), deps.Stderr(ctx))
			}}, nil
		},
		func(_ C, args []string) ([]string, error) { return args, nil },
		func(service capabilityService, args []string) (struct{}, error) { return struct{}{}, service.run(args) },
		func(io.Writer, cliout.Format, struct{}) error { return nil },
	)
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
