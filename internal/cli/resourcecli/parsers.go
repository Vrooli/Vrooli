package resourcecli

import (
	"fmt"
	"io"
	"strings"

	"github.com/vrooli/vrooli/internal/cli/scenariocli"
	"github.com/vrooli/vrooli/internal/resources"
)

type (
	helpOnlyError struct {
		text string
	}
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

func (e helpOnlyError) Error() string    { return e.text }
func (e helpOnlyError) HelpText() string { return e.text }

func commandHelpOnly(text string) error {
	return helpOnlyError{text: text}
}

func ParseNoArgs(command, help string, args []string) (NoArgsRequest, error) {
	if len(args) > 0 {
		for _, arg := range args {
			if arg == "--help" || arg == "-h" {
				return NoArgsRequest{}, commandHelpOnly(help)
			}
		}
		return NoArgsRequest{}, fmt.Errorf("%s does not accept positional arguments", command)
	}
	return NoArgsRequest{}, nil
}

func ParseSingleName(noun, command, help string, args []string) (NameRequest, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return NameRequest{}, commandHelpOnly(help)
		}
	}
	if len(args) != 1 {
		return NameRequest{}, fmt.Errorf("%s requires exactly one %s name", command, noun)
	}
	return NameRequest{Name: args[0]}, nil
}

func ParseListRequest(args []string) (NoArgsRequest, error) {
	return ParseNoArgs("resource list", ListHelpText, args)
}

func ParseValidateRequest(args []string) (ValidateRequest, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return ValidateRequest{}, commandHelpOnly(ValidateHelpText)
		}
	}
	if len(args) > 1 {
		return ValidateRequest{}, fmt.Errorf("resource validate accepts at most one resource name")
	}
	req := ValidateRequest{}
	if len(args) == 1 {
		req.Name = args[0]
	}
	return req, nil
}

func ParseStatusRequest(args []string) (StatusRequest, error) {
	req := StatusRequest{Fast: true}
	filtered := make([]string, 0, len(args))
	for _, arg := range args {
		switch arg {
		case "--help", "-h":
			return StatusRequest{}, commandHelpOnly(StatusHelpText)
		case "--fast":
			req.Fast = true
		case "--no-fast":
			req.Fast = false
		default:
			filtered = append(filtered, arg)
		}
	}
	if len(filtered) > 1 {
		return StatusRequest{}, fmt.Errorf("resource status accepts at most one resource name")
	}
	if len(filtered) == 1 {
		req.Name = filtered[0]
	}
	return req, nil
}

func ParseInfoRequest(args []string) (NameRequest, error) {
	return ParseSingleName("resource", "resource info", InfoHelpText, args)
}

func ParseDeprecateRequest(args []string) (NameRequest, error) {
	return ParseSingleName("resource", "resource deprecate", DeprecateHelpText, args)
}

func ParseListDeprecatedRequest(args []string) (NoArgsRequest, error) {
	return ParseNoArgs("resource list-deprecated", ListDeprecatedHelpText, args)
}

func ParseRestoreRequest(args []string) (NameRequest, error) {
	return ParseSingleName("resource", "resource restore", RestoreHelpText, args)
}

func ParseArchiveToBlueprintRequest(args []string) (NameRequest, error) {
	return ParseSingleName("resource", "resource archive-to-blueprint", ArchiveToBlueprintHelpText, args)
}

func ParseListBlueprintArchivedRequest(args []string) (NoArgsRequest, error) {
	return ParseNoArgs("resource list-blueprint-archived", ListBlueprintArchivedHelpText, args)
}

func ParseRestoreBlueprintRequest(args []string) (NameRequest, error) {
	return ParseSingleName("resource", "resource restore-blueprint", RestoreBlueprintHelpText, args)
}

func ParseArchiveGCRequest(args []string) (NoArgsRequest, error) {
	return ParseNoArgs("resource archive gc", ArchiveGCHelpText, args)
}

func ParseArchiveBlueprintGCRequest(args []string) (NoArgsRequest, error) {
	return ParseNoArgs("resource archive gc-blueprints", ArchiveBlueprintGCHelpText, args)
}

func ParseBlueprintListRequest(args []string) (NoArgsRequest, error) {
	return ParseNoArgs("resource blueprint list", BlueprintListHelpText, args)
}

func ParseBlueprintInfoRequest(args []string) (NameRequest, error) {
	return ParseSingleName("resource blueprint", "resource blueprint info", BlueprintInfoHelpText, args)
}

func ParseBlueprintSearchRequest(args []string) (BlueprintSearchRequest, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return BlueprintSearchRequest{}, commandHelpOnly(BlueprintSearchHelpText)
		}
	}
	if len(args) != 1 {
		return BlueprintSearchRequest{}, fmt.Errorf("resource blueprint search requires exactly one query")
	}
	return BlueprintSearchRequest{Query: args[0]}, nil
}

func ParseBlueprintValidateRequest(args []string) (NoArgsRequest, error) {
	return ParseNoArgs("resource blueprint validate", BlueprintValidateHelpText, args)
}

func ParseStartAllRequest(args []string) (NoArgsRequest, error) {
	return ParseNoArgs("resource start-all", StartAllHelpText, args)
}

func ParseStopAllRequest(args []string) (NoArgsRequest, error) {
	return ParseNoArgs("resource stop-all", StopAllHelpText, args)
}

func ParseTemplateListRequest(args []string) (NoArgsRequest, error) {
	return ParseNoArgs("resource template list", TemplateListHelpText, args)
}

func ParseTemplateShowRequest(args []string) (TemplateNameRequest, error) {
	req, err := ParseSingleName("template", "resource template show", TemplateShowHelpText, args)
	return TemplateNameRequest{Name: req.Name}, err
}

func ParseTemplateValidateRequest(args []string) (NoArgsRequest, error) {
	return ParseNoArgs("resource template validate", TemplateValidateHelpText, args)
}

func ParseTemplateGenerateRequest(
	args []string,
	stderr io.Writer,
	resolve func(resources.ResourceTemplateGenerateRequest) (resources.ResourceTemplateInfo, error),
) (TemplateGenerateOptions, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return TemplateGenerateOptions{}, commandHelpOnly(RenderTemplateGenerateHelpText())
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
	return "Usage: vrooli resource template generate <template> [options]\n       vrooli resource template generate --from-blueprint <name> [options]\nOptions:\n  --from-blueprint <name>  Seed values from an existing blueprint\n  --dest <path>            Destination directory (defaults to resources/<name>)\n  --var KEY=VALUE          Additional placeholder override (repeatable)\n  --force                  Overwrite destination if it already exists\n  --dry-run                Print the planned actions without writing files"
}
