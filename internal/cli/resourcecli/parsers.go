package resourcecli

import (
	"fmt"
	"io"
	"strings"

	"github.com/vrooli/vrooli/internal/cli/clipolicy"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cli/scenariocli"
	"github.com/vrooli/vrooli/internal/resources"
)

type (
	NoArgsRequest struct{}
	NameRequest   struct {
		Name string
	}
	StatusRequest struct {
		Name string
		Fast bool
	}
	ValidateRequest struct {
		Name string
	}
	UpstreamCheckRequest struct {
		Name string // empty means all coding-agent resources
		All  bool
	}
	BlueprintSearchRequest struct {
		Query string
	}
	TemplateNameRequest struct {
		Name string
	}
	TemplateGenerateOptions struct {
		TemplateName  string
		BlueprintName string
		Destination   string
		Force         bool
		DryRun        bool
		Values        map[string]string
	}
)

func ParseListRequest(args []string) (NoArgsRequest, error) {
	if err := commandtree.ParseNoArgs("resource list", CommandHelpText(CommandList), args); err != nil {
		return NoArgsRequest{}, err
	}
	return NoArgsRequest{}, nil
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
	req := StatusRequest{Fast: true}
	if len(parsed.Positionals) == 1 {
		req.Name = parsed.Positionals[0]
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

func ParseTemplateListRequest(args []string) (NoArgsRequest, error) {
	if err := commandtree.ParseNoArgs("resource template list", TemplateCommandHelpText(TemplateCommandList), args); err != nil {
		return NoArgsRequest{}, err
	}
	return NoArgsRequest{}, nil
}

func ParseTemplateShowRequest(args []string) (TemplateNameRequest, error) {
	name, err := commandtree.ParseSinglePositional("resource template show", TemplateCommandHelpText(TemplateCommandShow), "template name", args)
	if err != nil {
		return TemplateNameRequest{}, err
	}
	return TemplateNameRequest{Name: name}, nil
}

func ParseTemplateValidateRequest(args []string) (NoArgsRequest, error) {
	if err := commandtree.ParseNoArgs("resource template validate", TemplateCommandHelpText(TemplateCommandValidate), args); err != nil {
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

func ParseTemplateGenerateRequest(
	args []string,
	stderr io.Writer,
	resolve func(resources.ResourceTemplateGenerateRequest) (resources.ResourceTemplateInfo, error),
) (TemplateGenerateOptions, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return TemplateGenerateOptions{}, clipolicy.CommandHelpOnly(RenderTemplateGenerateHelpText())
		}
	}
	return ParseTemplateGenerateArgs(args, stderr, resolve)
}

func ParseTemplateGenerateArgs(
	args []string,
	stderr io.Writer,
	resolve func(resources.ResourceTemplateGenerateRequest) (resources.ResourceTemplateInfo, error),
) (TemplateGenerateOptions, error) {
	opts := TemplateGenerateOptions{Values: map[string]string{}}

	if len(args) > 0 && !strings.HasPrefix(args[0], "--") {
		opts.TemplateName = args[0]
		args = args[1:]
	}

	for index := 0; index < len(args); index++ {
		arg := args[index]
		switch {
		case arg == "--from-blueprint":
			if index+1 >= len(args) {
				return TemplateGenerateOptions{}, fmt.Errorf("resource template generate --from-blueprint requires a value")
			}
			index++
			opts.BlueprintName = args[index]
		case strings.HasPrefix(arg, "--from-blueprint="):
			opts.BlueprintName = strings.TrimPrefix(arg, "--from-blueprint=")
		}
	}

	var manifest resources.ResourceTemplateManifest
	if opts.TemplateName != "" || opts.BlueprintName != "" {
		info, err := resolve(resources.ResourceTemplateGenerateRequest{
			TemplateName:  opts.TemplateName,
			BlueprintName: opts.BlueprintName,
		})
		if err != nil {
			return TemplateGenerateOptions{}, err
		}
		manifest = info.Manifest
		opts.TemplateName = info.Name
	}

	flagMap := make(map[string]string, len(manifest.RequiredVars)+len(manifest.OptionalVars))
	for key, variable := range manifest.RequiredVars {
		if variable.Flag != "" {
			flagMap[variable.Flag] = key
		}
	}
	for key, variable := range manifest.OptionalVars {
		if variable.Flag != "" {
			flagMap[variable.Flag] = key
		}
	}

	for index := 0; index < len(args); index++ {
		arg := args[index]
		switch {
		case arg == "--from-blueprint":
			index++
		case strings.HasPrefix(arg, "--from-blueprint="):
			continue
		case arg == "--dest" || arg == "--destination":
			if index+1 >= len(args) {
				return TemplateGenerateOptions{}, fmt.Errorf("%s requires a value", arg)
			}
			index++
			opts.Destination = args[index]
		case strings.HasPrefix(arg, "--dest="):
			opts.Destination = strings.TrimPrefix(arg, "--dest=")
		case strings.HasPrefix(arg, "--destination="):
			opts.Destination = strings.TrimPrefix(arg, "--destination=")
		case arg == "--force":
			opts.Force = true
		case arg == "--dry-run":
			opts.DryRun = true
		case arg == "--var":
			if index+1 >= len(args) {
				return TemplateGenerateOptions{}, fmt.Errorf("resource template generate --var requires KEY=VALUE")
			}
			index++
			key, value, err := scenariocli.ParseTemplateKeyValue(args[index])
			if err != nil {
				return TemplateGenerateOptions{}, err
			}
			opts.Values[key] = value
		case strings.HasPrefix(arg, "--var="):
			key, value, err := scenariocli.ParseTemplateKeyValue(strings.TrimPrefix(arg, "--var="))
			if err != nil {
				return TemplateGenerateOptions{}, err
			}
			opts.Values[key] = value
		case strings.HasPrefix(arg, "--"):
			flagName, flagValue, consumesNext, err := scenariocli.ParseTemplateFlag(arg, args, index)
			if err != nil {
				return TemplateGenerateOptions{}, err
			}
			if consumesNext {
				index++
			}
			key, ok := flagMap[flagName]
			if !ok {
				_, _ = fmt.Fprintf(stderr, "Warning: unknown flag --%s; use --var KEY=VALUE for arbitrary placeholders\n", flagName)
				continue
			}
			opts.Values[key] = flagValue
		default:
			return TemplateGenerateOptions{}, fmt.Errorf("unexpected argument: %s", arg)
		}
	}

	return opts, nil
}

func RenderTemplateGenerateHelpText() string {
	return commandtree.HelpText("", "vrooli resource template generate", "Generate files from a resource template.", commandtree.Help{
		Usage: "vrooli resource template generate <template> [options]\n  vrooli resource template generate --from-blueprint <name> [options]",
	}, TemplateGenerateArgSchema())
}
