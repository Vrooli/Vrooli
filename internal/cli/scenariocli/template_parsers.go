package scenariocli

import (
	"fmt"
	"io"
	"strings"
)

func ParseTemplateListRequest(args []string) (TemplateListRequest, error) {
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		return TemplateListRequest{}, commandHelpOnly(TemplateCommandHelpText)
	}
	if len(args) > 0 {
		return TemplateListRequest{}, unknownOptionError("scenario template", args[0])
	}
	return TemplateListRequest{}, nil
}

func ParseTemplateShowRequest(args []string) (TemplateShowRequest, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return TemplateShowRequest{}, commandHelpOnly(TemplateCommandHelpText)
		}
	}
	if len(args) == 0 {
		return TemplateShowRequest{}, fmt.Errorf("scenario template show requires a template name")
	}
	if len(args) > 1 {
		return TemplateShowRequest{}, fmt.Errorf("scenario template show accepts exactly one template name")
	}
	return TemplateShowRequest{Name: args[0]}, nil
}

func ParseGenerateRequest(
	args []string,
	stderr io.Writer,
	loadTemplate func(name string) (TemplateInfo, error),
	parseArgs func(args []string, manifest TemplateManifest, stderr io.Writer) (GenerateOptions, error),
) (GenerateRequest, error) {
	if len(args) == 0 {
		return GenerateRequest{}, fmt.Errorf("scenario generate requires a template name")
	}
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return GenerateRequest{}, commandHelpOnly(TemplateGenerateHelpText)
		}
	}
	templateName := args[0]
	info, err := loadTemplate(templateName)
	if err != nil {
		return GenerateRequest{}, err
	}
	opts, err := parseArgs(args[1:], info.Manifest, stderr)
	if err != nil {
		return GenerateRequest{}, err
	}
	if opts.Values["SCENARIO_ID"] == "" {
		return GenerateRequest{}, fmt.Errorf("missing required value: --id")
	}
	missing := make([]string, 0)
	for key, variable := range info.Manifest.RequiredVars {
		if strings.TrimSpace(opts.Values[key]) == "" {
			name := "--" + variable.Flag
			if variable.Flag == "" {
				name = key
			}
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return GenerateRequest{}, fmt.Errorf("missing required values: %s", strings.Join(missing, ", "))
	}
	return GenerateRequest{TemplateInfo: info, Options: opts}, nil
}

func ParseGenerateArgs(args []string, manifest TemplateManifest, stderr io.Writer) (GenerateOptions, error) {
	opts := GenerateOptions{Values: map[string]string{}}
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
		case arg == "--dest":
			if index+1 >= len(args) {
				return GenerateOptions{}, fmt.Errorf("scenario generate --dest requires a value")
			}
			index++
			opts.Destination = args[index]
		case strings.HasPrefix(arg, "--dest="):
			opts.Destination = strings.TrimPrefix(arg, "--dest=")
		case arg == "--force":
			opts.Force = true
		case arg == "--dry-run":
			opts.DryRun = true
		case arg == "--run-hooks":
			opts.RunHooks = true
		case arg == "--var":
			if index+1 >= len(args) {
				return GenerateOptions{}, fmt.Errorf("scenario generate --var requires KEY=VALUE")
			}
			index++
			key, value, err := ParseTemplateKeyValue(args[index])
			if err != nil {
				return GenerateOptions{}, err
			}
			opts.Values[key] = value
		case strings.HasPrefix(arg, "--var="):
			key, value, err := ParseTemplateKeyValue(strings.TrimPrefix(arg, "--var="))
			if err != nil {
				return GenerateOptions{}, err
			}
			opts.Values[key] = value
		case strings.HasPrefix(arg, "--"):
			flagName, flagValue, consumesNext, err := ParseTemplateFlag(arg, args, index)
			if err != nil {
				return GenerateOptions{}, err
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
			return GenerateOptions{}, fmt.Errorf("unexpected argument: %s", arg)
		}
	}
	return opts, nil
}

func ParseTemplateFlag(arg string, args []string, index int) (string, string, bool, error) {
	if strings.Contains(arg, "=") {
		parts := strings.SplitN(strings.TrimPrefix(arg, "--"), "=", 2)
		if parts[1] == "" {
			return "", "", false, fmt.Errorf("--%s requires a value", parts[0])
		}
		return parts[0], parts[1], false, nil
	}
	if index+1 >= len(args) {
		return "", "", false, fmt.Errorf("%s requires a value", arg)
	}
	return strings.TrimPrefix(arg, "--"), args[index+1], true, nil
}

func ParseTemplateKeyValue(value string) (string, string, error) {
	parts := strings.SplitN(value, "=", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" {
		return "", "", fmt.Errorf("invalid KEY=VALUE pair: %s", value)
	}
	return parts[0], parts[1], nil
}

func RenderTemplateHelp(w io.Writer) {
	commandtreeHelp := "Scenario Template Commands"
	_, _ = fmt.Fprintln(w, commandtreeHelp)
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Usage:")
	_, _ = fmt.Fprintln(w, "  vrooli scenario template <subcommand>")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Scenario Templates:")
	_, _ = fmt.Fprintln(w, "    list               List scenario templates")
	_, _ = fmt.Fprintln(w, "    show               Show scenario template details")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Related:")
	_, _ = fmt.Fprintln(w, "  vrooli scenario generate <template> [options]")
}

func RenderGenerateHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, TemplateGenerateHelpText)
}
