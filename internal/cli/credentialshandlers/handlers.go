package credentialshandlers

import (
	"io"

	"github.com/vrooli/cli-core/cliapp"
	climanifest "github.com/vrooli/vrooli/cli"
	credentialsapp "github.com/vrooli/vrooli/internal/app/credentials"
	"github.com/vrooli/vrooli/internal/cli/credentialscli"
	"github.com/vrooli/vrooli/internal/cli/manifestdispatch"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cliout"
)

// HandlerDeps supplies only the root-context values the credential boundary needs.
type HandlerDeps[C any] struct {
	Root    func(C) string
	Globals func(C) rootcli.GlobalOptions
	Stdin   func(C) io.Reader
	Stdout  func(C) io.Writer
	Stderr  func(C) io.Writer
}

type credentialService struct {
	run func([]string) error
}

var (
	credentialRootNames  = []string{"doctor", "list", "delete", "provision", "status"}
	credentialGroupNames = map[string][]string{
		"store":    {"status", "init", "unlock", "lock", "rewrap", "change-passphrase"},
		"keyring":  {"status", "inspect", "repair", "unlock"},
		"recovery": {"export", "verify", "restore"},
	}
	breakGlassCommandNames = []string{"provision", "issue", "status", "rotate", "reset"}
)

// RegisteredCommandPaths returns the child paths bound by the credential handlers.
func RegisteredCommandPaths() []string {
	groupCommandCount := 0
	for _, names := range credentialGroupNames {
		groupCommandCount += len(names)
	}
	paths := make([]string, 0, len(credentialRootNames)+len(breakGlassCommandNames)+groupCommandCount)
	for _, name := range credentialRootNames {
		paths = append(paths, "credentials "+name)
	}
	for _, group := range []string{"store", "keyring", "recovery"} {
		for _, name := range credentialGroupNames[group] {
			paths = append(paths, "credentials "+group+" "+name)
		}
	}
	for _, name := range breakGlassCommandNames {
		paths = append(paths, "break-glass "+name)
	}
	return paths
}

// RootHandler dispatches `vrooli credentials`.
func RootHandler[C any](deps HandlerDeps[C]) rootcli.Handler[C] {
	return rootcli.BindService(deps.Stdout,
		func(C) (cliout.Format, error) { return cliout.FormatHuman, nil },
		func(ctx C, _ cliout.Format) (credentialService, error) {
			commandCtx := &credentialscli.Context{
				Root: deps.Root(ctx), Globals: deps.Globals(ctx), Stdin: deps.Stdin(ctx),
				Stdout: deps.Stdout(ctx), Stderr: deps.Stderr(ctx),
			}
			app := &credentialsapp.App{}
			return credentialService{run: func(args []string) error {
				if len(args) == 0 || manifestdispatch.WantsHelp(args) {
					return credentialscli.Run(commandCtx, args)
				}
				groups, err := credentialGroups(app, commandCtx)
				if err != nil {
					return err
				}
				core := cliapp.NewApp(cliapp.AppOptions{Name: "vrooli credentials", SubcommandGroups: groups.subgroups, Commands: []cliapp.CommandGroup{{Commands: groups.root.Subcommands}}})
				return core.RunWithWriters(manifestdispatch.WithJSON(args, commandCtx.Globals.JSON), deps.Stdout(ctx), deps.Stderr(ctx))
			}}, nil
		},
		func(_ C, args []string) ([]string, error) { return args, nil },
		func(service credentialService, args []string) (struct{}, error) { return struct{}{}, service.run(args) },
		func(io.Writer, cliout.Format, struct{}) error { return nil },
	)
}

// BreakGlassHandler dispatches `vrooli break-glass`.
func BreakGlassHandler[C any](deps HandlerDeps[C]) rootcli.Handler[C] {
	return rootcli.BindService(deps.Stdout,
		func(C) (cliout.Format, error) { return cliout.FormatHuman, nil },
		func(ctx C, _ cliout.Format) (credentialService, error) {
			commandCtx := &credentialscli.Context{
				Root: deps.Root(ctx), Globals: deps.Globals(ctx), Stdin: deps.Stdin(ctx),
				Stdout: deps.Stdout(ctx), Stderr: deps.Stderr(ctx),
			}
			app := &credentialsapp.App{}
			return credentialService{run: func(args []string) error {
				if len(args) == 0 || manifestdispatch.WantsHelp(args) {
					return credentialscli.RunBreakGlass(commandCtx, args)
				}
				group, err := cliapp.LoadFromManifest(climanifest.Bytes(), "break-glass", breakGlassBindings(app, commandCtx))
				if err != nil {
					return err
				}
				core := cliapp.NewApp(cliapp.AppOptions{Name: "vrooli break-glass", Commands: []cliapp.CommandGroup{{Commands: group.Subcommands}}})
				return core.RunWithWriters(manifestdispatch.WithJSON(args, commandCtx.Globals.JSON), deps.Stdout(ctx), deps.Stderr(ctx))
			}}, nil
		},
		func(_ C, args []string) ([]string, error) { return args, nil },
		func(service credentialService, args []string) (struct{}, error) { return struct{}{}, service.run(args) },
		func(io.Writer, cliout.Format, struct{}) error { return nil },
	)
}

type credentialGroupsResult struct {
	root      cliapp.SubcommandGroup
	subgroups []cliapp.SubcommandGroup
}

func credentialGroups(app *credentialsapp.App, ctx *credentialscli.Context) (credentialGroupsResult, error) {
	root, err := cliapp.LoadFromManifest(climanifest.Bytes(), "credentials", credentialBindings(app, ctx, nil, credentialRootNames))
	if err != nil {
		return credentialGroupsResult{}, err
	}
	paths := []string{"store", "keyring", "recovery"}
	result := credentialGroupsResult{root: root, subgroups: make([]cliapp.SubcommandGroup, 0, len(paths))}
	for _, path := range paths {
		names := credentialGroupNames[path]
		group, loadErr := cliapp.LoadFromManifest(climanifest.Bytes(), "credentials/"+path, credentialBindings(app, ctx, []string{path}, names))
		if loadErr != nil {
			return credentialGroupsResult{}, loadErr
		}
		result.subgroups = append(result.subgroups, group)
	}
	return result, nil
}

func credentialBindings(app *credentialsapp.App, ctx *credentialscli.Context, prefix []string, names []string) map[string]func(cliapp.RunContext) error {
	bindings := map[string]func(cliapp.RunContext) error{}
	for _, name := range names {
		path := append(append([]string(nil), prefix...), name)
		bindings[name] = func(path []string) func(cliapp.RunContext) error {
			return func(runCtx cliapp.RunContext) error {
				globals := ctx.Globals
				globals.JSON = globals.JSON || runCtx.JSON()
				commandCtx := *ctx
				commandCtx.Globals = globals
				return app.Run(&commandCtx, append(path, manifestdispatch.LegacyArgs(runCtx)...))
			}
		}(path)
	}
	return bindings
}

func breakGlassBindings(app *credentialsapp.App, ctx *credentialscli.Context) map[string]func(cliapp.RunContext) error {
	bindings := map[string]func(cliapp.RunContext) error{}
	for _, name := range breakGlassCommandNames {
		command := name
		bindings[name] = func(runCtx cliapp.RunContext) error {
			globals := ctx.Globals
			globals.JSON = globals.JSON || runCtx.JSON()
			commandCtx := *ctx
			commandCtx.Globals = globals
			return app.RunBreakGlass(&commandCtx, append([]string{command}, manifestdispatch.LegacyArgs(runCtx)...))
		}
	}
	return bindings
}
