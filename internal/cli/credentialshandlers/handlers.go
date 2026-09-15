package credentialshandlers

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliapp"
	climanifest "github.com/vrooli/vrooli/cli"
	credentialsapp "github.com/vrooli/vrooli/internal/app/credentials"
	"github.com/vrooli/vrooli/internal/cli/manifestdispatch"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cliout"
)

type credentialService struct {
	run func([]string) error
}

var (
	credentialStatusCommand = "status"
	credentialRootNames     = []string{"doctor", "list", "delete", "provision", credentialStatusCommand}
	credentialGroupNames    = map[string][]string{
		"store":      {credentialStatusCommand, "init", "unlock", "lock", "rewrap", "change-passphrase"},
		"keyring":    {credentialStatusCommand, "inspect", "repair", "unlock"},
		"recovery":   {"export", "verify", "restore"},
		"extensions": {"install", "uninstall", credentialStatusCommand},
	}
	breakGlassCommandNames = []string{"provision", "issue", credentialStatusCommand, "rotate", "reset"}
)

const credentialGroupPathParts = 2

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
	for _, group := range []string{"store", "keyring", "recovery", "extensions"} {
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
func RootHandler[C any](deps rootcli.HandlerDeps[C]) rootcli.Handler[C] {
	return rootcli.BindService(deps.Stdout,
		func(C) (cliout.Format, error) { return cliout.FormatHuman, nil },
		func(ctx C, _ cliout.Format) (credentialService, error) {
			commandCtx := &rootcli.CommandContext{
				Root: deps.Root(ctx), Globals: deps.Globals(ctx), Stdin: deps.Stdin(ctx),
				Stdout: deps.Stdout(ctx), Stderr: deps.Stderr(ctx), Context: rootcli.ResolveOperationContext(deps, ctx),
			}
			app := &credentialsapp.Service{}
			return credentialService{run: func(args []string) error {
				if len(args) == 0 || manifestdispatch.WantsHelp(args) {
					return writeCredentialsHelp(deps.Stdout(ctx))
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
func BreakGlassHandler[C any](deps rootcli.HandlerDeps[C]) rootcli.Handler[C] {
	return rootcli.BindService(deps.Stdout,
		func(C) (cliout.Format, error) { return cliout.FormatHuman, nil },
		func(ctx C, _ cliout.Format) (credentialService, error) {
			commandCtx := &rootcli.CommandContext{
				Root: deps.Root(ctx), Globals: deps.Globals(ctx), Stdin: deps.Stdin(ctx),
				Stdout: deps.Stdout(ctx), Stderr: deps.Stderr(ctx), Context: rootcli.ResolveOperationContext(deps, ctx),
			}
			app := &credentialsapp.Service{}
			return credentialService{run: func(args []string) error {
				if len(args) == 0 || manifestdispatch.WantsHelp(args) {
					return writeBreakGlassHelp(deps.Stdout(ctx))
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

func credentialGroups(app *credentialsapp.Service, ctx *rootcli.CommandContext) (credentialGroupsResult, error) {
	root, err := cliapp.LoadFromManifest(climanifest.Bytes(), "credentials", credentialBindings(app, ctx, nil, credentialRootNames))
	if err != nil {
		return credentialGroupsResult{}, err
	}
	paths := []string{"store", "keyring", "recovery", "extensions"}
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

func credentialBindings(app *credentialsapp.Service, ctx *rootcli.CommandContext, prefix []string, names []string) map[string]func(cliapp.RunContext) error {
	bindings := map[string]func(cliapp.RunContext) error{}
	for _, name := range names {
		path := append(append([]string(nil), prefix...), name)
		bindings[name] = func(path []string) func(cliapp.RunContext) error {
			return func(runCtx cliapp.RunContext) error {
				globals := ctx.Globals
				globals.JSON = globals.JSON || runCtx.JSON()
				commandCtx := *ctx
				commandCtx.Globals = globals
				return dispatchCredentialCommand(app, &commandCtx, path, runCtx)
			}
		}(path)
	}
	return bindings
}

func breakGlassBindings(app *credentialsapp.Service, ctx *rootcli.CommandContext) map[string]func(cliapp.RunContext) error {
	bindings := map[string]func(cliapp.RunContext) error{}
	for _, name := range breakGlassCommandNames {
		command := name
		bindings[name] = func(runCtx cliapp.RunContext) error {
			globals := ctx.Globals
			globals.JSON = globals.JSON || runCtx.JSON()
			commandCtx := *ctx
			commandCtx.Globals = globals
			return dispatchBreakGlassCommand(app, &commandCtx, command, runCtx)
		}
	}
	return bindings
}

func writeCredentialsHelp(out io.Writer) error {
	_, err := io.WriteString(out, "Usage: vrooli credentials <doctor|list|delete|provision|status>\n  vrooli credentials doctor\n  vrooli credentials list\n  vrooli credentials extensions <install|uninstall|status>\n")
	return err
}

func writeBreakGlassHelp(out io.Writer) error {
	_, err := io.WriteString(out, "Usage: vrooli break-glass <provision|issue|status|rotate|reset>\n")
	return err
}

func outputFormat(runCtx cliapp.RunContext) string {
	if runCtx.JSON() {
		return "json"
	}
	format := strings.TrimSpace(flagValue(runCtx, "format"))
	if format == "" {
		format = "text"
	}
	return format
}

func flagValue(runCtx cliapp.RunContext, name string) string {
	if !runCtx.FlagDeclared(name) {
		return ""
	}
	return runCtx.Flag(name)
}

func boolFlag(runCtx cliapp.RunContext, name string) bool {
	return runCtx.FlagDeclared(name) && runCtx.BoolFlag(name)
}

func flagValues(runCtx cliapp.RunContext, name string) []string {
	if !runCtx.FlagDeclared(name) {
		return nil
	}
	return runCtx.FlagValues(name)
}

func dispatchCredentialCommand(app *credentialsapp.Service, ctx *rootcli.CommandContext, path []string, runCtx cliapp.RunContext) error {
	if len(path) == 0 {
		return fmt.Errorf("credentials command is required")
	}
	format := outputFormat(runCtx)
	switch path[0] {
	case "doctor":
		return app.Doctor(ctx.OperationContext(), ctx.Root, ctx.Stdout, credentialsapp.DoctorOptions{Format: format, CheckWrites: boolFlag(runCtx, "check-writes")})
	case "list":
		return app.List(ctx.OperationContext(), ctx.Root, ctx.Stdout, credentialsapp.ListOptions{Format: format})
	case "delete":
		return app.Delete(ctx.OperationContext(), ctx.Stdout, credentialsapp.CredentialSelectorOptions{Identity: flagValue(runCtx, "identity"), Field: flagValue(runCtx, "field"), Yes: boolFlag(runCtx, "yes")})
	case "provision":
		return app.Provision(ctx.OperationContext(), ctx.Stdout, ctx.Stderr, credentialsapp.CredentialSelectorOptions{Identity: flagValue(runCtx, "identity"), Field: flagValue(runCtx, "field")}, ctx.Input())
	case credentialStatusCommand:
		return app.Status(ctx.OperationContext(), ctx.Stdout, credentialsapp.CredentialSelectorOptions{Identity: flagValue(runCtx, "identity"), Field: flagValue(runCtx, "field"), Format: format})
	case "store", "keyring", "recovery", "extensions":
		return dispatchCredentialGroup(app, ctx, path, runCtx)
	default:
		return fmt.Errorf("unknown credentials command %q", path[0])
	}
}

func dispatchCredentialGroup(app *credentialsapp.Service, ctx *rootcli.CommandContext, path []string, runCtx cliapp.RunContext) error {
	if len(path) < credentialGroupPathParts {
		return fmt.Errorf("credentials %s command is required", path[0])
	}
	format := outputFormat(runCtx)
	switch path[0] {
	case "store":
		switch path[1] {
		case credentialStatusCommand:
			return app.StoreStatus(ctx.OperationContext(), ctx.Stdout, format)
		case "init":
			return app.StoreInit(ctx.OperationContext(), ctx.Stdout, ctx.Stderr, format, ctx.Input())
		case "unlock":
			return app.StoreUnlock(ctx.OperationContext(), ctx.Stdout, ctx.Stderr, ctx.Input())
		case "lock":
			return app.StoreLock(ctx.OperationContext(), ctx.Stdout)
		case "rewrap":
			return app.StoreRewrap(ctx.OperationContext(), ctx.Stdout, ctx.Stderr, format, ctx.Input())
		case "change-passphrase":
			return app.StoreChangePassphrase(ctx.OperationContext(), ctx.Stdout, ctx.Stderr, ctx.Input())
		}
	case "keyring":
		opts := credentialsapp.KeyringOptions{Path: flagValue(runCtx, "path"), Format: format}
		switch path[1] {
		case credentialStatusCommand:
			return app.KeyringStatus(ctx.OperationContext(), ctx.Stdout, opts)
		case "inspect":
			return app.KeyringFile(ctx.OperationContext(), ctx.Stdout, opts, false)
		case "repair":
			threshold := time.Duration(0)
			if raw := strings.TrimSpace(flagValue(runCtx, "offer-retire-older-than")); raw != "" {
				parsed, err := time.ParseDuration(raw)
				if err != nil {
					return fmt.Errorf("offer-retire-older-than: %w", err)
				}
				threshold = parsed
			}
			return app.KeyringRepair(ctx.OperationContext(), ctx.Stdout, credentialsapp.KeyringRepairOptions{Path: opts.Path, Format: opts.Format, RetireBackup: flagValue(runCtx, "retire-backup"), OfferRetireOlderThan: threshold})
		case "unlock":
			return app.KeyringUnlock(ctx.OperationContext(), ctx.Stdout, ctx.Stderr, ctx.Input())
		}
	case "recovery":
		switch path[1] {
		case "export":
			return app.ExportRecovery(ctx.OperationContext(), ctx.Root, ctx.Stdout, ctx.Stderr, credentialsapp.RecoveryExportOptions{Entries: flagValues(runCtx, "entry"), Output: flagValue(runCtx, "output"), All: boolFlag(runCtx, "all"), Format: format}, ctx.Input())
		case "verify":
			return app.VerifyRecovery(ctx.OperationContext(), ctx.Stdout, ctx.Stderr, credentialsapp.RecoveryBundleOptions{Input: flagValue(runCtx, "input"), Format: format}, ctx.Input())
		case "restore":
			return app.RestoreRecovery(ctx.OperationContext(), ctx.Stdout, ctx.Stderr, credentialsapp.RecoveryBundleOptions{Input: flagValue(runCtx, "input")}, ctx.Input())
		}
	case "extensions":
		return app.Extensions(ctx.OperationContext(), ctx.Stdout, credentialsapp.ExtensionOptions{
			Operation: path[1], HostPath: flagValue(runCtx, "host-path"),
			ExtensionID: flagValue(runCtx, "extension-id"), Browser: flagValue(runCtx, "browser"), Format: format,
			Yes: boolFlag(runCtx, "yes"),
		})
	}
	return fmt.Errorf("unknown credentials command %q %q", path[0], path[1])
}

func dispatchBreakGlassCommand(app *credentialsapp.Service, ctx *rootcli.CommandContext, command string, runCtx cliapp.RunContext) error {
	opts := credentialsapp.BreakGlassOptions{Operation: command, Format: outputFormat(runCtx), AccountID: flagValue(runCtx, "account-id"), Audience: flagValue(runCtx, "audience"), Purpose: flagValue(runCtx, "purpose"), Target: flagValue(runCtx, "target"), Scopes: flagValue(runCtx, "scopes"), Scope: flagValue(runCtx, "scope"), OperatorID: flagValue(runCtx, "operator-id"), MachineID: flagValue(runCtx, "machine-id"), NodeID: flagValue(runCtx, "node-id"), PlanHash: flagValue(runCtx, "plan-hash"), OperationID: flagValue(runCtx, "operation-id")}
	if raw := strings.TrimSpace(flagValue(runCtx, "ttl")); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil {
			return fmt.Errorf("break-glass issue --ttl: %w", err)
		}
		opts.TTL = parsed
	}
	return app.BreakGlass(ctx.OperationContext(), ctx.Stdout, opts, ctx.Input())
}
