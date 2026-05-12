package scenariocli

import (
	"fmt"
	"io"
	"strings"

	"github.com/vrooli/vrooli/internal/cli/clipolicy"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
)

type TemplateCommandID string

const (
	TemplateCommandList     TemplateCommandID = "list"
	TemplateCommandShow     TemplateCommandID = "show"
	TemplateCommandValidate TemplateCommandID = "validate"
	TemplateCommandCleanup  TemplateCommandID = "cleanup"
	TemplateCommandDrift    TemplateCommandID = "drift"
)

type DesignCommandID string

const (
	DesignCommandList     DesignCommandID = "list"
	DesignCommandShow     DesignCommandID = "show"
	DesignCommandValidate DesignCommandID = "validate"
)

func templateCommandSpecs() []commandtree.Spec[TemplateCommandID] {
	return []commandtree.Spec[TemplateCommandID]{
		{
			Name:    string(TemplateCommandList),
			Summary: "List scenario templates",
			Handler: TemplateCommandList,
		},
		{
			Name:    string(TemplateCommandShow),
			Summary: "Show scenario template details",
			Args: commandtree.ArgSchema{
				Positionals: []commandtree.PositionalArg{{Name: "template name", Required: true}},
			},
			Handler: TemplateCommandShow,
		},
		{
			Name:    string(TemplateCommandValidate),
			Summary: "Validate scenario templates",
			Args: commandtree.ArgSchema{
				Options: []commandtree.OptionArg{
					{Name: "--mode", ValueName: "shallow|deep", Description: "Validation depth; defaults to shallow"},
					{Name: "--template", ValueName: "name", Description: "Validate only one template"},
					{Name: "--retain-temp", Description: "Keep temporary generated output for debugging"},
					{Name: "--test-preset", ValueName: "name", Description: "test-genie preset for deep validation"},
					{Name: "--warning-policy", ValueName: "ignore|report|fail", Description: "Deep validation warning handling; defaults to report for deep mode"},
				},
			},
			Handler: TemplateCommandValidate,
		},
		{
			Name:    string(TemplateCommandDrift),
			Summary: "Compare a generated scenario's recorded template hashes against the current template",
			Args: commandtree.ArgSchema{
				Positionals: []commandtree.PositionalArg{{Name: "scenario"}},
				Options: []commandtree.OptionArg{
					{Name: "--all", Description: "Audit every generated scenario with recorded provenance"},
					{Name: "--verbose", Description: "List per-file differences when content drift is detected"},
					{Name: "--json", Description: "Emit a JSON report instead of human-readable output"},
				},
			},
			Handler: TemplateCommandDrift,
		},
		{
			Name:    string(TemplateCommandCleanup),
			Summary: "Clean retained or stale deep template validation workspaces",
			Args: commandtree.ArgSchema{
				Options: []commandtree.OptionArg{
					{Name: "--dry-run", Description: "Preview cleanup without deleting files"},
					{Name: "--older-than", ValueName: "duration", Description: "Clean broad matches older than this Go duration; defaults to 24h"},
					{Name: "--include-retained", Description: "Allow broad cleanup to remove retained debugging runs"},
					{Name: "--run", ValueName: "run-id", Description: "Clean one explicit run id regardless of age or retained status"},
				},
			},
			Handler: TemplateCommandCleanup,
		},
	}
}

func templateGenerateArgSchema() commandtree.ArgSchema {
	return commandtree.ArgSchema{
		Positionals: []commandtree.PositionalArg{{Name: "template", Required: true}},
		Options: []commandtree.OptionArg{
			{Name: "--id", ValueName: "slug", Description: "Scenario identifier"},
			{Name: "--display-name", ValueName: "name", Description: "Human-friendly scenario name"},
			{Name: "--description", ValueName: "text", Description: "Scenario description"},
			{Name: "--design", ValueName: "kit-id|none", Description: "Design kit to install into the generated scenario"},
			{Name: "--dest", ValueName: "path", Description: "Destination directory"},
			{Name: "--var", ValueName: "KEY=VALUE", Repeatable: true, Description: "Additional placeholder override"},
			{Name: "--force", Description: "Overwrite destination if it already exists"},
			{Name: "--dry-run", Description: "Print planned actions without writing files"},
			{Name: "--run-hooks", Description: "Execute template post hooks after generation"},
		},
	}
}

func designCommandSpecs() []commandtree.Spec[DesignCommandID] {
	return []commandtree.Spec[DesignCommandID]{
		{
			Name:    string(DesignCommandList),
			Summary: "List scenario design kits",
			Handler: DesignCommandList,
		},
		{
			Name:    string(DesignCommandShow),
			Summary: "Show scenario design kit details",
			Args: commandtree.ArgSchema{
				Positionals: []commandtree.PositionalArg{{Name: "kit id", Required: true}},
			},
			Handler: DesignCommandShow,
		},
		{
			Name:    string(DesignCommandValidate),
			Summary: "Validate scenario design kits",
			Args: commandtree.ArgSchema{
				Positionals: []commandtree.PositionalArg{{Name: "kit id"}},
				Options:     []commandtree.OptionArg{{Name: "--all", Description: "Validate every design kit"}},
			},
			Handler: DesignCommandValidate,
		},
	}
}

func ParseTemplateListRequest(args []string) (TemplateListRequest, error) {
	if _, err := commandtree.ParseArgs("scenario template list", templateCommandHelpText(TemplateCommandList), templateCommandSpec(TemplateCommandList).Args, args); err != nil {
		return TemplateListRequest{}, err
	}
	return TemplateListRequest{}, nil
}

func ParseTemplateShowRequest(args []string) (TemplateShowRequest, error) {
	parsed, err := commandtree.ParseArgs("scenario template show", templateCommandHelpText(TemplateCommandShow), templateCommandSpec(TemplateCommandShow).Args, args)
	if err != nil {
		return TemplateShowRequest{}, err
	}
	return TemplateShowRequest{Name: parsed.Positionals[0]}, nil
}

func ParseTemplateValidateRequest(args []string) (TemplateValidateRequest, error) {
	parsed, err := commandtree.ParseArgs("scenario template validate", templateCommandHelpText(TemplateCommandValidate), templateCommandSpec(TemplateCommandValidate).Args, args)
	if err != nil {
		return TemplateValidateRequest{}, err
	}
	req := TemplateValidateRequest{
		Mode:          TemplateValidationModeShallow,
		TestPreset:    DefaultTemplateValidationTestPreset,
		WarningPolicy: TemplateValidationWarningPolicyIgnore,
	}
	if mode := strings.TrimSpace(parsed.FlagValue("--mode")); mode != "" {
		req.Mode = TemplateValidationMode(mode)
	}
	switch req.Mode {
	case TemplateValidationModeShallow, TemplateValidationModeDeep:
	default:
		return TemplateValidateRequest{}, fmt.Errorf("unknown validation mode %q; expected shallow or deep", req.Mode)
	}
	req.TemplateName = strings.TrimSpace(parsed.FlagValue("--template"))
	req.RetainTemp = parsed.HasFlag("--retain-temp")
	if preset := strings.TrimSpace(parsed.FlagValue("--test-preset")); preset != "" {
		req.TestPreset = preset
	}
	if req.Mode == TemplateValidationModeDeep {
		req.WarningPolicy = TemplateValidationWarningPolicyReport
	}
	if policy := strings.TrimSpace(parsed.FlagValue("--warning-policy")); policy != "" {
		req.WarningPolicy = TemplateValidationWarningPolicy(policy)
	}
	switch req.WarningPolicy {
	case TemplateValidationWarningPolicyIgnore, TemplateValidationWarningPolicyReport, TemplateValidationWarningPolicyFail:
	default:
		return TemplateValidateRequest{}, fmt.Errorf("unknown warning policy %q; expected ignore, report, or fail", req.WarningPolicy)
	}
	if req.Mode == TemplateValidationModeShallow && parsed.HasFlag("--test-preset") {
		return TemplateValidateRequest{}, fmt.Errorf("--test-preset is only valid with --mode deep")
	}
	if req.Mode == TemplateValidationModeShallow && req.RetainTemp {
		return TemplateValidateRequest{}, fmt.Errorf("--retain-temp is only valid with --mode deep")
	}
	return req, nil
}

func ParseTemplateDriftRequest(args []string) (TemplateDriftRequest, error) {
	parsed, err := commandtree.ParseArgs("scenario template drift", templateCommandHelpText(TemplateCommandDrift), templateCommandSpec(TemplateCommandDrift).Args, args)
	if err != nil {
		return TemplateDriftRequest{}, err
	}
	req := TemplateDriftRequest{
		All:     parsed.HasFlag("--all"),
		Verbose: parsed.HasFlag("--verbose"),
		JSON:    parsed.HasFlag("--json"),
	}
	if len(parsed.Positionals) > 0 {
		req.Scenario = strings.TrimSpace(parsed.Positionals[0])
	}
	if req.Scenario == "" && !req.All {
		return TemplateDriftRequest{}, fmt.Errorf("scenario template drift requires a scenario name or --all")
	}
	if req.Scenario != "" && req.All {
		return TemplateDriftRequest{}, fmt.Errorf("scenario template drift accepts either a scenario name or --all, not both")
	}
	return req, nil
}

func ParseTemplateCleanupRequest(args []string) (TemplateCleanupRequest, error) {
	parsed, err := commandtree.ParseArgs("scenario template cleanup", templateCommandHelpText(TemplateCommandCleanup), templateCommandSpec(TemplateCommandCleanup).Args, args)
	if err != nil {
		return TemplateCleanupRequest{}, err
	}
	return TemplateCleanupRequest{
		DryRun:          parsed.HasFlag("--dry-run"),
		OlderThan:       strings.TrimSpace(parsed.FlagValue("--older-than")),
		IncludeRetained: parsed.HasFlag("--include-retained"),
		RunID:           strings.TrimSpace(parsed.FlagValue("--run")),
	}, nil
}

func templateCommandHelpText(id TemplateCommandID) string {
	spec := templateCommandSpec(id)
	return commandtree.SpecHelpText("", "vrooli scenario template "+spec.Name, spec)
}

func ParseDesignListRequest(args []string) (DesignListRequest, error) {
	if _, err := commandtree.ParseArgs("scenario design list", DesignCommandHelpText(), designCommandSpec(DesignCommandList).Args, args); err != nil {
		return DesignListRequest{}, err
	}
	return DesignListRequest{}, nil
}

func ParseDesignShowRequest(args []string) (DesignShowRequest, error) {
	parsed, err := commandtree.ParseArgs("scenario design show", DesignCommandHelpText(), designCommandSpec(DesignCommandShow).Args, args)
	if err != nil {
		return DesignShowRequest{}, err
	}
	return DesignShowRequest{ID: parsed.Positionals[0]}, nil
}

func ParseDesignValidateRequest(args []string) (DesignValidateRequest, error) {
	parsed, err := commandtree.ParseArgs("scenario design validate", DesignCommandHelpText(), designCommandSpec(DesignCommandValidate).Args, args)
	if err != nil {
		return DesignValidateRequest{}, err
	}
	req := DesignValidateRequest{}
	if len(parsed.Positionals) > 0 {
		req.ID = parsed.Positionals[0]
	}
	req.All = parsed.HasFlag("--all")
	if req.ID == "" && !req.All {
		req.All = true
	}
	if req.ID != "" && req.All {
		return DesignValidateRequest{}, fmt.Errorf("use either a kit id or --all, not both")
	}
	return req, nil
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
			return GenerateRequest{}, clipolicy.CommandHelpOnly(TemplateGenerateHelpText())
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
		case arg == "--design":
			if index+1 >= len(args) {
				return GenerateOptions{}, fmt.Errorf("scenario generate --design requires a value")
			}
			index++
			opts.Design = args[index]
		case strings.HasPrefix(arg, "--design="):
			opts.Design = strings.TrimPrefix(arg, "--design=")
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
	commandtree.WriteHelp(w, TemplateCommandHelpText())
}

func RenderDesignHelp(w io.Writer) {
	commandtree.WriteHelp(w, DesignCommandHelpText())
}

func RenderGenerateHelp(w io.Writer) {
	commandtree.WriteHelp(w, TemplateGenerateHelpText())
}

func templateCommandSpec(id TemplateCommandID) commandtree.Spec[TemplateCommandID] {
	for _, spec := range templateCommandSpecs() {
		if spec.Handler == id {
			return spec
		}
	}
	panic("unknown template command spec: " + string(id))
}

func designCommandSpec(id DesignCommandID) commandtree.Spec[DesignCommandID] {
	for _, spec := range designCommandSpecs() {
		if spec.Handler == id {
			return spec
		}
	}
	panic("unknown design command spec: " + string(id))
}
