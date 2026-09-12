package resourcecli

import (
	"fmt"

	resourceapp "github.com/vrooli/vrooli/internal/app/resource"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
)

type (
	NoArgsRequest          = resourceapp.NoArgsRequest
	ScaffoldRequest        = resourceapp.ScaffoldRequest
	CLISyncRequest         = resourceapp.CLISyncRequest
	NameRequest            = resourceapp.NameRequest
	StatusRequest          = resourceapp.StatusRequest
	ValidateRequest        = resourceapp.ValidateRequest
	UpstreamCheckRequest   = resourceapp.UpstreamCheckRequest
	BlueprintSearchRequest = resourceapp.BlueprintSearchRequest
)

func ParseListRequest(args []string) (NoArgsRequest, error) {
	if err := commandtree.ParseNoArgs("resource list", CommandHelpText(CommandList), args); err != nil {
		return NoArgsRequest{}, err
	}
	return NoArgsRequest{}, nil
}

func ParseCensusRequest(args []string) (NoArgsRequest, error) {
	if err := commandtree.ParseNoArgs("resource census", CommandHelpText(CommandCensus), args); err != nil {
		return NoArgsRequest{}, err
	}
	return NoArgsRequest{}, nil
}

func ParseScaffoldRequest(args []string) (ScaffoldRequest, error) {
	spec := commandSpec(CommandScaffold)
	parsed, err := commandtree.ParseArgs("resource scaffold", CommandHelpText(CommandScaffold), spec.Args, args)
	if err != nil {
		return ScaffoldRequest{}, err
	}
	name, driver := parsed.FlagValue("--name"), parsed.FlagValue("--driver")
	if name == "" || driver == "" {
		return ScaffoldRequest{}, fmt.Errorf("resource scaffold: --name and --driver are required")
	}
	return ScaffoldRequest{Name: name, Driver: driver}, nil
}

func ParseCLISyncRequest(args []string) (CLISyncRequest, error) {
	spec := commandSpec(CommandCLISync)
	parsed, err := commandtree.ParseArgs("resource cli-sync", CommandHelpText(CommandCLISync), spec.Args, args)
	if err != nil {
		return CLISyncRequest{}, err
	}
	return CLISyncRequest{DryRun: parsed.HasFlag("--dry-run")}, nil
}

func ParseValidateRequest(args []string) (ValidateRequest, error) {
	spec := commandSpec(CommandValidate)
	parsed, err := commandtree.ParseArgs("resource validate", CommandHelpText(CommandValidate), spec.Args, args)
	if err != nil {
		return ValidateRequest{}, err
	}
	req := ValidateRequest{}
	if len(parsed.Positionals) == 1 {
		req.Name = parsed.Positionals[0]
	}
	return req, nil
}

func ParseStatusRequest(args []string) (StatusRequest, error) {
	spec := commandSpec(CommandStatus)
	parsed, err := commandtree.ParseArgs("resource status", CommandHelpText(CommandStatus), spec.Args, args)
	if err != nil {
		return StatusRequest{}, err
	}
	// Asking about one resource is a question about that resource, so it runs
	// its health checks and reads its accelerator placement. Asking about the
	// whole fleet is a sweep, so it stays fast: placement reads the host once
	// per resource, which is fine for one and wasteful for twenty-nine.
	// Either default is overridable.
	req := StatusRequest{Fast: true}
	if len(parsed.Positionals) == 1 {
		req.Name = parsed.Positionals[0]
		req.Fast = false
	}
	if parsed.HasFlag("--no-fast") {
		req.Fast = false
	}
	if parsed.HasFlag("--fast") {
		req.Fast = true
	}
	return req, nil
}

func ParseUpstreamCheckRequest(args []string) (UpstreamCheckRequest, error) {
	spec := commandSpec(CommandUpstreamCheck)
	parsed, err := commandtree.ParseArgs("resource upstream-check", CommandHelpText(CommandUpstreamCheck), spec.Args, args)
	if err != nil {
		return UpstreamCheckRequest{}, err
	}
	req := UpstreamCheckRequest{}
	if len(parsed.Positionals) == 1 {
		req.Name = parsed.Positionals[0]
	}
	if parsed.HasFlag("--all") {
		req.All = true
	}
	return req, nil
}

func ParseInfoRequest(args []string) (NameRequest, error) {
	name, err := commandtree.ParseSinglePositional("resource info", CommandHelpText(CommandInfo), "resource name", args)
	if err != nil {
		return NameRequest{}, err
	}
	return NameRequest{Name: name}, nil
}

func ParseDeprecateRequest(args []string) (NameRequest, error) {
	name, err := commandtree.ParseSinglePositional("resource deprecate", CommandHelpText(CommandDeprecate), "resource name", args)
	if err != nil {
		return NameRequest{}, err
	}
	return NameRequest{Name: name}, nil
}

func ParseListDeprecatedRequest(args []string) (NoArgsRequest, error) {
	if err := commandtree.ParseNoArgs("resource list-deprecated", CommandHelpText(CommandListDeprecated), args); err != nil {
		return NoArgsRequest{}, err
	}
	return NoArgsRequest{}, nil
}

func ParseRestoreRequest(args []string) (NameRequest, error) {
	name, err := commandtree.ParseSinglePositional("resource restore", CommandHelpText(CommandRestore), "resource name", args)
	if err != nil {
		return NameRequest{}, err
	}
	return NameRequest{Name: name}, nil
}

func ParseArchiveToBlueprintRequest(args []string) (NameRequest, error) {
	name, err := commandtree.ParseSinglePositional("resource archive-to-blueprint", CommandHelpText(CommandArchiveToBlueprint), "resource name", args)
	if err != nil {
		return NameRequest{}, err
	}
	return NameRequest{Name: name}, nil
}

func ParseListBlueprintArchivedRequest(args []string) (NoArgsRequest, error) {
	if err := commandtree.ParseNoArgs("resource list-blueprint-archived", CommandHelpText(CommandListBlueprintArchived), args); err != nil {
		return NoArgsRequest{}, err
	}
	return NoArgsRequest{}, nil
}

func ParseRestoreBlueprintRequest(args []string) (NameRequest, error) {
	name, err := commandtree.ParseSinglePositional("resource restore-blueprint", CommandHelpText(CommandRestoreBlueprint), "resource name", args)
	if err != nil {
		return NameRequest{}, err
	}
	return NameRequest{Name: name}, nil
}

func ParseArchiveGCRequest(args []string) (NoArgsRequest, error) {
	if err := commandtree.ParseNoArgs("resource archive gc", ArchiveCommandHelpText(ArchiveCommandGC), args); err != nil {
		return NoArgsRequest{}, err
	}
	return NoArgsRequest{}, nil
}

func ParseArchiveBlueprintGCRequest(args []string) (NoArgsRequest, error) {
	if err := commandtree.ParseNoArgs("resource archive gc-blueprints", ArchiveCommandHelpText(ArchiveCommandGCBlueprints), args); err != nil {
		return NoArgsRequest{}, err
	}
	return NoArgsRequest{}, nil
}

func ParseBlueprintListRequest(args []string) (NoArgsRequest, error) {
	if err := commandtree.ParseNoArgs("resource blueprint list", BlueprintCommandHelpText(BlueprintCommandList), args); err != nil {
		return NoArgsRequest{}, err
	}
	return NoArgsRequest{}, nil
}

func ParseBlueprintInfoRequest(args []string) (NameRequest, error) {
	name, err := commandtree.ParseSinglePositional("resource blueprint info", BlueprintCommandHelpText(BlueprintCommandInfo), "resource name", args)
	if err != nil {
		return NameRequest{}, err
	}
	return NameRequest{Name: name}, nil
}

func ParseBlueprintSearchRequest(args []string) (BlueprintSearchRequest, error) {
	spec := blueprintCommandSpec(BlueprintCommandSearch)
	parsed, err := commandtree.ParseArgs("resource blueprint search", BlueprintCommandHelpText(BlueprintCommandSearch), spec.Args, args)
	if err != nil {
		return BlueprintSearchRequest{}, err
	}
	return BlueprintSearchRequest{Query: parsed.Positionals[0]}, nil
}

func ParseBlueprintValidateRequest(args []string) (NoArgsRequest, error) {
	if err := commandtree.ParseNoArgs("resource blueprint validate", BlueprintCommandHelpText(BlueprintCommandValidate), args); err != nil {
		return NoArgsRequest{}, err
	}
	return NoArgsRequest{}, nil
}

func ParseStartAllRequest(args []string) (NoArgsRequest, error) {
	if err := commandtree.ParseNoArgs("resource start-all", CommandHelpText(CommandStartAll), args); err != nil {
		return NoArgsRequest{}, err
	}
	return NoArgsRequest{}, nil
}

func ParseStopAllRequest(args []string) (NoArgsRequest, error) {
	if err := commandtree.ParseNoArgs("resource stop-all", CommandHelpText(CommandStopAll), args); err != nil {
		return NoArgsRequest{}, err
	}
	return NoArgsRequest{}, nil
}

func ParseSchemaValidateRequest(args []string) (NoArgsRequest, error) {
	if err := commandtree.ParseNoArgs("resource schema validate", SchemaCommandHelpText(SchemaCommandValidate), args); err != nil {
		return NoArgsRequest{}, err
	}
	return NoArgsRequest{}, nil
}

func ParseSchemaSyncRequest(args []string) (NoArgsRequest, error) {
	if err := commandtree.ParseNoArgs("resource schema sync", SchemaCommandHelpText(SchemaCommandSync), args); err != nil {
		return NoArgsRequest{}, err
	}
	return NoArgsRequest{}, nil
}
