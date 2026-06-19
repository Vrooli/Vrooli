package scenariocli

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/internal/cli/clipolicy"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/resources"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

// instanceOption is the shared `--instance <variant>` flag that lets every
// scenario-addressing command target a named instance (e.g. a shadow). It is
// sugar for the canonical `name@variant` argument; both resolve through the one
// scenarioruntime.ParseInstanceKey parser (§1a), and supplying them with
// conflicting values is a hard error.
func instanceOption() commandtree.OptionArg {
	return commandtree.OptionArg{Name: "--instance", ValueName: "variant", Description: "Target a named instance/variant (e.g. shadow); equivalent to name@variant"}
}

// resolveInstanceArg folds a scenario-name argument and the --instance flag into
// the canonical instance slug ("scenario" for live, "scenario@variant"
// otherwise). It is the single CLI-side resolution point so downstream layers
// receive one unambiguous identifier rather than re-splitting the name.
func resolveInstanceArg(command, name, instanceFlag string) (string, error) {
	key, err := scenarioruntime.ParseInstanceKey(name, instanceFlag)
	if err != nil {
		return "", clipolicy.UsageErrorf("scenario "+command, "%s", err.Error())
	}
	return key.Slug(), nil
}

type CommandID string

const (
	CommandList            CommandID = "list"
	CommandInfo            CommandID = "info"
	CommandStatus          CommandID = "status"
	CommandValidateEnv     CommandID = "validate-env"
	CommandRun             CommandID = "run"
	CommandStart           CommandID = "start"
	CommandStartAll        CommandID = "start-all"
	CommandSetup           CommandID = "setup"
	CommandRestart         CommandID = "restart"
	CommandStop            CommandID = "stop"
	CommandStopAll         CommandID = "stop-all"
	CommandTest            CommandID = "test"
	CommandLogs            CommandID = "logs"
	CommandOpen            CommandID = "open"
	CommandPort            CommandID = "port"
	CommandUISmoke         CommandID = "ui-smoke"
	CommandRequirements    CommandID = "requirements"
	CommandDesign          CommandID = "design"
	CommandTemplate        CommandID = "template"
	CommandGenerate        CommandID = "generate"
	CommandOrient          CommandID = "orient"
	CommandDetemplate      CommandID = "detemplate"
	CommandCompleteness    CommandID = "completeness"
	CommandHealFromSandbox CommandID = "heal-from-sandbox"
)

func CommandSpecs() []commandtree.Spec[CommandID] {
	return []commandtree.Spec[CommandID]{
		{
			Name: string(CommandList), Group: "Read-only Commands", Summary: "List discovered scenarios", Handler: CommandList, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot},
			Args: commandtree.ArgSchema{Options: []commandtree.OptionArg{commandtree.JSONOption(), {Name: "--include-ports", Description: "Include port mappings in the list output"}}},
		},
		{
			Name: string(CommandInfo), Group: "Read-only Commands", Summary: "Show scenario metadata and runtime summary", Handler: CommandInfo, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot},
			Args: commandtree.ArgSchema{Positionals: []commandtree.PositionalArg{{Name: "scenario name", Required: true}}, Options: []commandtree.OptionArg{commandtree.JSONOption()}},
		},
		{
			Name: string(CommandStatus), Group: "Read-only Commands", Summary: "Show scenario runtime status", Handler: CommandStatus, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot},
			Args: commandtree.ArgSchema{Positionals: []commandtree.PositionalArg{{Name: "scenario name"}}, Options: []commandtree.OptionArg{commandtree.JSONOption(), instanceOption()}},
		},
		{
			Name: string(CommandValidateEnv), Group: "Read-only Commands", Summary: "Validate resource-derived environment injection for a scenario", Handler: CommandValidateEnv, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot},
			Args: commandtree.ArgSchema{Positionals: []commandtree.PositionalArg{{Name: "scenario name", Required: true}}, Options: []commandtree.OptionArg{commandtree.JSONOption()}},
		},
		{Name: string(CommandRun), Group: "Lifecycle and Utility Commands", Summary: "Run a scenario directly (alias of start)", Handler: CommandRun, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}},
		{
			Name: string(CommandStart), Group: "Lifecycle and Utility Commands", Summary: "Start a scenario", Handler: CommandStart, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot},
			Args: commandtree.ArgSchema{
				Positionals: []commandtree.PositionalArg{{Name: "scenario name", Required: true, Repeatable: true}},
				Options:     []commandtree.OptionArg{{Name: "--path", ValueName: "path"}, {Name: "--best-effort"}, {Name: "--clean-stale"}, {Name: "--open"}, commandtree.JSONOption(), instanceOption()},
			},
		},
		{
			Name: string(CommandStartAll), Group: "Lifecycle and Utility Commands", Summary: "Start all available scenarios", Handler: CommandStartAll, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot},
			Args: commandtree.ArgSchema{Options: []commandtree.OptionArg{commandtree.JSONOption()}},
		},
		{
			Name: string(CommandSetup), Group: "Lifecycle and Utility Commands", Summary: "Run the setup lifecycle", Handler: CommandSetup, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot},
			Args: commandtree.ArgSchema{Positionals: []commandtree.PositionalArg{{Name: "scenario name", Required: true}}, Options: []commandtree.OptionArg{{Name: "--path", ValueName: "path"}}},
		},
		{
			Name: string(CommandRestart), Group: "Lifecycle and Utility Commands", Summary: "Restart a scenario", Handler: CommandRestart, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot},
			Args: commandtree.ArgSchema{
				Positionals: []commandtree.PositionalArg{{Name: "scenario name", Required: true}},
				Options:     []commandtree.OptionArg{{Name: "--path", ValueName: "path"}, {Name: "--best-effort"}, {Name: "--clean-stale"}, {Name: "--open"}, commandtree.JSONOption(), instanceOption()},
			},
		},
		{
			Name: string(CommandStop), Group: "Lifecycle and Utility Commands", Summary: "Stop a running scenario", Handler: CommandStop, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot},
			Args: commandtree.ArgSchema{Positionals: []commandtree.PositionalArg{{Name: "scenario name", Required: true}}, Options: []commandtree.OptionArg{commandtree.JSONOption(), instanceOption()}},
		},
		{
			Name: string(CommandStopAll), Group: "Lifecycle and Utility Commands", Summary: "Stop all running scenarios", Handler: CommandStopAll, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot},
			Args: commandtree.ArgSchema{Options: []commandtree.OptionArg{commandtree.JSONOption()}},
		},
		{Name: string(CommandTest), Group: "Lifecycle and Utility Commands", Summary: "Run scenario tests", Handler: CommandTest, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}},
		{Name: string(CommandLogs), Group: "Lifecycle and Utility Commands", Summary: "View logs for a scenario", Handler: CommandLogs, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}},
		{
			Name: string(CommandOpen), Group: "Lifecycle and Utility Commands", Summary: "Open a scenario in the browser", Handler: CommandOpen, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot},
			Args: commandtree.ArgSchema{
				Positionals: []commandtree.PositionalArg{{Name: "scenario name", Required: true}},
				Options:     []commandtree.OptionArg{{Name: "--port", ValueName: "name"}, {Name: "--print-url"}, commandtree.JSONOption()},
			},
		},
		{
			Name: string(CommandPort), Group: "Lifecycle and Utility Commands", Summary: "Show running port assignments", Handler: CommandPort, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot},
			Args: commandtree.ArgSchema{
				Positionals: []commandtree.PositionalArg{{Name: "scenario name", Required: true}, {Name: "port name"}},
				Options:     []commandtree.OptionArg{commandtree.JSONOption(), instanceOption()},
			},
		},
		{Name: string(CommandUISmoke), Group: "Lifecycle and Utility Commands", Summary: "Run the Browserless UI smoke harness", Handler: CommandUISmoke, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}},
		{Name: string(CommandRequirements), Group: "Lifecycle and Utility Commands", Summary: "Manage scenario requirements", Handler: CommandRequirements, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}},
		{Name: string(CommandDesign), Group: "Lifecycle and Utility Commands", Summary: "Manage scenario design kits", Handler: CommandDesign, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}},
		{Name: string(CommandTemplate), Group: "Lifecycle and Utility Commands", Summary: "Manage scenario templates", Handler: CommandTemplate, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}},
		{Name: string(CommandGenerate), Group: "Lifecycle and Utility Commands", Summary: "Scaffold a scenario from a template", Handler: CommandGenerate, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}},
		{
			Name: string(CommandOrient), Group: "Lifecycle and Utility Commands", Summary: "Show or finalize generated scenario orientation progress", Handler: CommandOrient, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot},
			Args: commandtree.ArgSchema{
				Positionals: []commandtree.PositionalArg{{Name: "scenario name", Required: true}},
				Options:     []commandtree.OptionArg{commandtree.JSONOption(), {Name: "--finalize", Description: "Remove temporary orientation metadata after required checks pass"}},
			},
		},
		{
			Name: string(CommandDetemplate), Group: "Lifecycle and Utility Commands", Summary: "Remove the template's example domain from a generated scenario", Handler: CommandDetemplate, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot},
			Args: commandtree.ArgSchema{
				Positionals: []commandtree.PositionalArg{{Name: "scenario name", Required: true}},
				Options:     []commandtree.OptionArg{commandtree.JSONOption(), {Name: "--dry-run", Description: "Preview removals without writing, deleting, or running finalizers"}},
			},
		},
		{Name: string(CommandCompleteness), Group: "Lifecycle and Utility Commands", Summary: "Calculate a completeness score", Handler: CommandCompleteness, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}},
		{
			Name: string(CommandHealFromSandbox), Group: "Lifecycle and Utility Commands", Summary: "Relaunch sandbox-rooted scenario processes", Handler: CommandHealFromSandbox, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot},
			Args: commandtree.ArgSchema{Options: []commandtree.OptionArg{{Name: "--merged-path", ValueName: "path"}, {Name: "--dry-run"}}},
		},
	}
}

func HelpOnlyWithoutRoot(args []string) bool { return commandtree.WantsHelp(args) }

func RenderCommandHelp(w io.Writer) {
	commandtree.RenderHelp(w, commandtree.Help{
		Title:        "Vrooli Scenario Commands",
		Usage:        "vrooli scenario <subcommand> [options]",
		DefaultGroup: "Scenario Management",
	}, CommandSpecs())
}

func commandSpec(id CommandID) commandtree.Spec[CommandID] {
	for _, spec := range CommandSpecs() {
		if spec.Handler == id {
			return spec
		}
	}
	panic("unknown scenario command spec: " + string(id))
}

func commandHelpText(id CommandID) string {
	spec := commandSpec(id)
	return commandtree.SpecHelpText("", "vrooli scenario "+spec.Name, spec)
}

type StartRequest struct {
	Names     []string
	Options   lifecycle.StartOptions
	JSON      bool
	OpenAfter bool
}
type (
	StopRequest struct {
		Name string
		JSON bool
	}
	RestartRequest struct {
		Name      string
		Options   lifecycle.StartOptions
		JSON      bool
		OpenAfter bool
	}
)

type (
	ListRequest struct{ JSON, IncludePorts bool }
	InfoRequest struct {
		Name string
		JSON bool
	}
	StatusRequest struct {
		Name string
		JSON bool
	}
	ValidateEnvRequest struct {
		Name string
		JSON bool
	}
	SetupRequest struct {
		Name string
		Opts lifecycle.PhaseOptions
		JSON bool
	}
	TestRequest struct {
		Name     string
		Selector string
		Opts     lifecycle.PhaseOptions
		// JSON emits the typed vrooli.cli.v1.TestPhaseResult pass/fail summary.
		JSON bool
	}
	StartAllRequest struct{ JSON bool }
	StopAllRequest  struct{ JSON bool }
	PortRequest     struct {
		ScenarioName, PortName string
		JSON                   bool
	}
	OpenRequest struct {
		ScenarioName, PortName string
		PrintURL, JSON         bool
	}
	RequirementsRequest struct {
		Snapshot bool
		Args     []string
	}
	HealFromSandboxRequest struct {
		MergedPath string
		DryRun     bool
	}
	HealFromSandboxResponse struct {
		Affected     []string
		DryRun       bool
		StoppedCount int
	}
)

type ValidateEnvResponse struct {
	Report resources.ScenarioEnvValidationReport
}

func ParseScenarioNameAndJSON(command string, defaultJSON bool, args []string) (string, bool, error) {
	name, jsonFlag, err := parseOptionalScenarioNameAndJSONWithHelp(command, defaultJSON, "", args)
	if err != nil {
		return "", false, err
	}
	if name == "" {
		return "", false, clipolicy.UsageErrorf("scenario "+command, "scenario %s requires a scenario name", command)
	}
	return name, jsonFlag, nil
}

func ParseOptionalScenarioNameAndJSON(command string, defaultJSON bool, args []string) (string, bool, error) {
	return parseOptionalScenarioNameAndJSONWithHelp(command, defaultJSON, "", args)
}

func parseOptionalScenarioNameAndJSONWithHelp(command string, defaultJSON bool, helpText string, args []string) (string, bool, error) {
	parsed, err := commandtree.ParseArgs("scenario "+command, helpText, commandtree.ArgSchema{
		Positionals: []commandtree.PositionalArg{{Name: "scenario name"}},
		Options: []commandtree.OptionArg{
			{Name: "--json"},
		},
	}, args)
	if err != nil {
		return "", false, err
	}
	name := ""
	if len(parsed.Positionals) == 1 {
		name = parsed.Positionals[0]
	}
	return name, defaultJSON || parsed.HasFlag("--json"), nil
}

func ParseScenarioStartArgs(defaultJSON bool, args []string) ([]string, lifecycle.StartOptions, bool, bool, error) {
	spec := commandSpec(CommandStart)
	parsed, err := commandtree.ParseArgs("scenario start", commandHelpText(CommandStart), spec.Args, args)
	if err != nil {
		return nil, lifecycle.StartOptions{}, false, false, err
	}
	opts := lifecycle.StartOptions{
		BestEffort: parsed.HasFlag("--best-effort"),
		CleanStale: parsed.HasFlag("--clean-stale"),
		CustomPath: parsed.FlagValue("--path"),
	}
	instanceFlag := parsed.FlagValue("--instance")
	names := make([]string, 0, len(parsed.Positionals))
	for _, positional := range parsed.Positionals {
		slug, err := resolveInstanceArg("start", positional, instanceFlag)
		if err != nil {
			return nil, lifecycle.StartOptions{}, false, false, err
		}
		names = append(names, slug)
	}
	return names, opts, defaultJSON || parsed.HasFlag("--json"), parsed.HasFlag("--open"), nil
}

func ParseScenarioSingleStartArgs(command string, defaultJSON bool, args []string) (string, lifecycle.StartOptions, bool, bool, error) {
	names, opts, jsonFlag, openAfter, err := ParseScenarioStartArgs(defaultJSON, args)
	if err != nil {
		return "", lifecycle.StartOptions{}, false, false, err
	}
	if len(names) == 0 {
		return "", lifecycle.StartOptions{}, false, false, clipolicy.UsageErrorf("scenario "+command, "scenario %s requires a scenario name", command)
	}
	if len(names) > 1 {
		return "", lifecycle.StartOptions{}, false, false, clipolicy.UsageErrorf("scenario "+command, "scenario %s accepts exactly one scenario name", command)
	}
	return names[0], opts, jsonFlag, openAfter, nil
}

func ParseStartRequest(globalsJSON bool, args []string) (StartRequest, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return StartRequest{}, clipolicy.CommandHelpOnly(commandHelpText(CommandStart))
		}
	}
	names, opts, jsonFlag, openAfter, err := ParseScenarioStartArgs(globalsJSON, args)
	if err != nil {
		return StartRequest{}, err
	}
	if len(names) == 0 {
		return StartRequest{}, clipolicy.UsageErrorf("scenario start", "scenario start requires at least one scenario name")
	}
	if opts.CustomPath != "" && len(names) != 1 {
		return StartRequest{}, clipolicy.UsageErrorf("scenario start", "scenario start with --path accepts exactly one scenario name")
	}
	return StartRequest{Names: names, Options: opts, JSON: jsonFlag, OpenAfter: openAfter}, nil
}

func ParseStopRequest(globalsJSON bool, args []string) (StopRequest, error) {
	spec := commandSpec(CommandStop)
	parsed, err := commandtree.ParseArgs("scenario stop", commandHelpText(CommandStop), spec.Args, args)
	if err != nil {
		return StopRequest{}, err
	}
	if len(parsed.Positionals) == 0 {
		return StopRequest{}, clipolicy.UsageErrorf("scenario stop", "scenario stop requires a scenario name")
	}
	slug, err := resolveInstanceArg("stop", parsed.Positionals[0], parsed.FlagValue("--instance"))
	if err != nil {
		return StopRequest{}, err
	}
	return StopRequest{Name: slug, JSON: globalsJSON || parsed.HasFlag("--json")}, nil
}

func ParseRestartRequest(globalsJSON bool, args []string) (RestartRequest, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return RestartRequest{}, clipolicy.CommandHelpOnly(commandHelpText(CommandRestart))
		}
	}
	name, opts, jsonFlag, openAfter, err := ParseScenarioSingleStartArgs("restart", globalsJSON, args)
	if err != nil {
		return RestartRequest{}, err
	}
	return RestartRequest{Name: name, Options: opts, JSON: jsonFlag, OpenAfter: openAfter}, nil
}

func ParseListRequest(globalsJSON bool, args []string) (ListRequest, error) {
	spec := commandSpec(CommandList)
	parsed, err := commandtree.ParseArgs("scenario list", commandHelpText(CommandList), spec.Args, args)
	if err != nil {
		return ListRequest{}, err
	}
	return ListRequest{
		JSON:         globalsJSON || parsed.HasFlag("--json"),
		IncludePorts: parsed.HasFlag("--include-ports"),
	}, nil
}

func ParseStatusRequest(globalsJSON bool, args []string) (StatusRequest, error) {
	spec := commandSpec(CommandStatus)
	parsed, err := commandtree.ParseArgs("scenario status", commandHelpText(CommandStatus), spec.Args, args)
	if err != nil {
		return StatusRequest{}, err
	}
	name := ""
	if len(parsed.Positionals) == 1 {
		if name, err = resolveInstanceArg("status", parsed.Positionals[0], parsed.FlagValue("--instance")); err != nil {
			return StatusRequest{}, err
		}
	} else if parsed.FlagValue("--instance") != "" {
		return StatusRequest{}, clipolicy.UsageErrorf("scenario status", "scenario status --instance requires a scenario name")
	}
	return StatusRequest{Name: name, JSON: globalsJSON || parsed.HasFlag("--json")}, nil
}

func ParseValidateEnvRequest(globalsJSON bool, args []string) (ValidateEnvRequest, error) {
	name, jsonFlag, err := parseOptionalScenarioNameAndJSONWithHelp("validate-env", globalsJSON, commandHelpText(CommandValidateEnv), args)
	if err != nil {
		return ValidateEnvRequest{}, err
	}
	if name == "" {
		return ValidateEnvRequest{}, clipolicy.UsageErrorf("scenario validate-env", "scenario validate-env requires a scenario name")
	}
	return ValidateEnvRequest{Name: name, JSON: jsonFlag}, nil
}

func RenderValidateEnvResponse(w io.Writer, format cliout.Format, resp ValidateEnvResponse) error {
	if format == cliout.FormatJSON {
		return writeScenarioEnvValidationJSON(w, resp.Report)
	}
	status := "passed"
	if !resp.Report.Passed {
		status = "failed"
	}
	if _, err := fmt.Fprintf(w, "Scenario env validation %s for %s\n", status, resp.Report.Scenario); err != nil {
		return err
	}
	for _, issue := range resp.Report.Issues {
		if _, err := fmt.Fprintf(w, "- [%s] %s\n", issue.Severity, issue.Message); err != nil {
			return err
		}
	}
	return nil
}

func ParsePhaseArgs(command string, args []string) (string, lifecycle.PhaseOptions, error) {
	name := ""
	opts := lifecycle.PhaseOptions{}
	for index := 0; index < len(args); index++ {
		arg := args[index]
		switch arg {
		case "--path":
			if index+1 >= len(args) {
				return "", lifecycle.PhaseOptions{}, clipolicy.UsageErrorf("scenario "+command, "scenario %s --path requires a value", command)
			}
			index++
			opts.CustomPath = args[index]
		default:
			if strings.HasPrefix(arg, "-") {
				opts.Args = append(opts.Args, arg)
				continue
			}
			if name == "" {
				name = arg
			} else {
				opts.Args = append(opts.Args, arg)
			}
		}
	}
	if name == "" {
		return "", lifecycle.PhaseOptions{}, clipolicy.UsageErrorf("scenario "+command, "scenario %s requires a scenario name", command)
	}
	return name, opts, nil
}

func ParseInfoRequest(globalsJSON bool, args []string) (InfoRequest, error) {
	name, jsonFlag, err := parseOptionalScenarioNameAndJSONWithHelp("info", globalsJSON, commandHelpText(CommandInfo), args)
	if err != nil {
		return InfoRequest{}, err
	}
	if name == "" {
		return InfoRequest{}, clipolicy.UsageErrorf("scenario info", "scenario info requires a scenario name")
	}
	return InfoRequest{Name: name, JSON: jsonFlag}, nil
}

func ParseSetupRequest(globalsJSON bool, args []string) (SetupRequest, error) {
	jsonFlag := globalsJSON
	for _, arg := range args {
		switch arg {
		case "--help", "-h":
			return SetupRequest{}, clipolicy.CommandHelpOnly(commandHelpText(CommandSetup))
		case "--json":
			jsonFlag = true
		}
	}
	name, opts, err := ParsePhaseArgs("setup", args)
	if err != nil {
		return SetupRequest{}, err
	}
	return SetupRequest{Name: name, Opts: opts, JSON: jsonFlag}, nil
}

func ParseTestArgs(globalsJSON, globalsVerbose bool, args []string) (TestRequest, error) {
	name := ""
	selection := ""
	req := TestRequest{JSON: globalsJSON}
	opts := lifecycle.PhaseOptions{}
	remaining := []string{}
	for index := 0; index < len(args); index++ {
		arg := args[index]
		switch arg {
		case "--path":
			if index+1 >= len(args) {
				return TestRequest{}, clipolicy.UsageErrorf("scenario test", "scenario test --path requires a value")
			}
			index++
			opts.CustomPath = args[index]
		case "--allow-skip-missing-runtime":
			opts.AllowSkipMissingRuntime = true
		case "--manage-runtime":
			opts.ManageRuntime = true
		case "--json":
			// Wrapper-level JSON (typed TestPhaseResult pass/fail summary) — NOT
			// forwarded to the test-genie child; that stays terse/human by default.
			req.JSON = true
		default:
			if strings.HasPrefix(arg, "-") {
				remaining = append(remaining, arg)
				continue
			}
			if name == "" {
				name = arg
			} else if selection == "" {
				selection = arg
			} else {
				remaining = append(remaining, arg)
			}
		}
	}
	if name == "" {
		return TestRequest{}, clipolicy.UsageErrorf("scenario test", "scenario test requires a scenario name")
	}
	if selection != "" {
		valid := map[string]string{"structure": "structure", "dependencies": "dependencies", "unit": "unit", "integration": "integration", "business": "business", "performance": "performance", "all": "all", "e2e": "integration"}
		mapped, ok := valid[selection]
		if !ok {
			return TestRequest{}, clipolicy.UsageErrorf("scenario test", "invalid test selector: %s", selection)
		}
		req.Selector = mapped
		remaining = append([]string{mapped}, remaining...)
	}
	if globalsVerbose && !containsArg(remaining, "--verbose") {
		remaining = append(remaining, "--verbose")
	}
	opts.Args = remaining
	req.Name = name
	req.Opts = opts
	return req, nil
}

func ParseTestRequest(globalsJSON, globalsVerbose bool, args []string) (TestRequest, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return TestRequest{}, clipolicy.CommandHelpOnly(TestHelpText())
		}
	}
	return ParseTestArgs(globalsJSON, globalsVerbose, args)
}

func ParseStartAllRequest(globalsJSON bool, args []string) (StartAllRequest, error) {
	spec := commandSpec(CommandStartAll)
	parsed, err := commandtree.ParseArgs("scenario start-all", commandHelpText(CommandStartAll), spec.Args, args)
	if err != nil {
		return StartAllRequest{}, err
	}
	return StartAllRequest{JSON: globalsJSON || parsed.HasFlag("--json")}, nil
}

func ParseStopAllRequest(globalsJSON bool, args []string) (StopAllRequest, error) {
	spec := commandSpec(CommandStopAll)
	parsed, err := commandtree.ParseArgs("scenario stop-all", commandHelpText(CommandStopAll), spec.Args, args)
	if err != nil {
		return StopAllRequest{}, err
	}
	return StopAllRequest{JSON: globalsJSON || parsed.HasFlag("--json")}, nil
}

func ParsePortRequest(globalsJSON bool, args []string) (PortRequest, error) {
	spec := commandSpec(CommandPort)
	parsed, err := commandtree.ParseArgs("scenario port", commandHelpText(CommandPort), spec.Args, args)
	if err != nil {
		return PortRequest{}, err
	}
	slug, err := resolveInstanceArg("port", parsed.Positionals[0], parsed.FlagValue("--instance"))
	if err != nil {
		return PortRequest{}, err
	}
	req := PortRequest{
		ScenarioName: slug,
		JSON:         globalsJSON || parsed.HasFlag("--json"),
	}
	if len(parsed.Positionals) > 1 {
		req.PortName = parsed.Positionals[1]
	}
	return req, nil
}

func ParseOpenRequest(globalsJSON bool, args []string) (OpenRequest, error) {
	spec := commandSpec(CommandOpen)
	parsed, err := commandtree.ParseArgs("scenario open", commandHelpText(CommandOpen), spec.Args, args)
	if err != nil {
		return OpenRequest{}, err
	}
	req := OpenRequest{
		ScenarioName: parsed.Positionals[0],
		PortName:     "UI_PORT",
		PrintURL:     parsed.HasFlag("--print-url"),
		JSON:         globalsJSON || parsed.HasFlag("--json"),
	}
	if value := parsed.FlagValue("--port"); value != "" {
		req.PortName = value
	}
	return req, nil
}

func ParseOrientationRequest(globalsJSON bool, args []string) (OrientationRequest, error) {
	spec := commandSpec(CommandOrient)
	parsed, err := commandtree.ParseArgs("scenario orient", commandHelpText(CommandOrient), spec.Args, args)
	if err != nil {
		return OrientationRequest{}, err
	}
	return OrientationRequest{
		Name:     parsed.Positionals[0],
		JSON:     globalsJSON || parsed.HasFlag("--json"),
		Finalize: parsed.HasFlag("--finalize"),
	}, nil
}

func ParseDetemplateRequest(globalsJSON bool, args []string) (DetemplateRequest, error) {
	spec := commandSpec(CommandDetemplate)
	parsed, err := commandtree.ParseArgs("scenario detemplate", commandHelpText(CommandDetemplate), spec.Args, args)
	if err != nil {
		return DetemplateRequest{}, err
	}
	return DetemplateRequest{
		Name:   parsed.Positionals[0],
		JSON:   globalsJSON || parsed.HasFlag("--json"),
		DryRun: parsed.HasFlag("--dry-run"),
	}, nil
}

func RenderOrientationResponse(w io.Writer, format cliout.Format, report OrientationReport) error {
	if format == cliout.FormatJSON {
		return writeScenarioOrientationJSON(w, report)
	}
	if report.Finalized {
		_, _ = fmt.Fprintf(w, "Orientation finalized for %s\n", report.Scenario)
		if strings.TrimSpace(report.Message) != "" {
			_, _ = fmt.Fprintln(w, report.Message)
		}
		return nil
	}
	if strings.TrimSpace(report.Message) != "" && len(report.Steps) == 0 {
		_, _ = fmt.Fprintln(w, report.Message)
		return nil
	}
	_, _ = fmt.Fprintf(w, "Orientation for %s\n", report.Scenario)
	if report.Template.ID != "" {
		_, _ = fmt.Fprintf(w, "Template: %s", report.Template.ID)
		if report.Template.Version != "" {
			_, _ = fmt.Fprintf(w, " (%s)", report.Template.Version)
		}
		_, _ = fmt.Fprintln(w)
	}
	if report.Design.ID != "" {
		_, _ = fmt.Fprintf(w, "Design: %s", report.Design.ID)
		if report.Design.Version != "" {
			_, _ = fmt.Fprintf(w, " (%s)", report.Design.Version)
		}
		if report.Design.Adapter != "" {
			_, _ = fmt.Fprintf(w, " adapter=%s", report.Design.Adapter)
		}
		_, _ = fmt.Fprintln(w)
	}
	if report.StartDocument != "" {
		_, _ = fmt.Fprintf(w, "Start document: %s\n", report.StartDocument)
	}
	_, _ = fmt.Fprintf(w, "Progress: %d/%d required steps complete\n", report.Completed, report.Required)
	for _, step := range report.Steps {
		marker := "[ ]"
		if step.Complete {
			marker = "[x]"
		}
		_, _ = fmt.Fprintf(w, "  %s %s", marker, step.ID)
		if step.Title != "" {
			_, _ = fmt.Fprintf(w, " - %s", step.Title)
		}
		if !step.Required {
			_, _ = fmt.Fprint(w, " (optional)")
		}
		_, _ = fmt.Fprintln(w)
		for _, check := range step.Checks {
			checkMarker := "fail"
			if check.Skipped {
				checkMarker = "skip"
			} else if check.Passed {
				checkMarker = "pass"
			}
			label := check.Label
			if label == "" {
				label = check.Kind
			}
			_, _ = fmt.Fprintf(w, "      %s  %s", checkMarker, label)
			if check.Message != "" {
				_, _ = fmt.Fprintf(w, " - %s", check.Message)
			}
			_, _ = fmt.Fprintln(w)
		}
	}
	if report.NextStep != nil {
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintf(w, "Next step: %s", report.NextStep.ID)
		if report.NextStep.Title != "" {
			_, _ = fmt.Fprintf(w, " - %s", report.NextStep.Title)
		}
		_, _ = fmt.Fprintln(w)
	}
	if report.FinalizeRequired {
		_, _ = fmt.Fprintln(w)
		_, _ = fmt.Fprintln(w, "Run with --finalize after required steps pass.")
	}
	return nil
}

func RenderRequirementsResponse(w io.Writer, format cliout.Format, _ struct{}) error { return nil }

func ParseRequirementsRequest(args []string) (RequirementsRequest, error) {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		return RequirementsRequest{}, clipolicy.CommandHelpOnly(RequirementsHelpText())
	}
	req := RequirementsRequest{Args: append([]string(nil), args...)}
	if args[0] == "snapshot" {
		req.Snapshot = true
	}
	return req, nil
}

func ParseHealFromSandboxRequest(defaultMergedPath string, args []string) (HealFromSandboxRequest, error) {
	spec := commandSpec(CommandHealFromSandbox)
	parsed, err := commandtree.ParseArgs("scenario heal-from-sandbox", commandHelpText(CommandHealFromSandbox), spec.Args, args)
	if err != nil {
		return HealFromSandboxRequest{}, err
	}
	req := HealFromSandboxRequest{
		MergedPath: strings.TrimSpace(defaultMergedPath),
		DryRun:     parsed.HasFlag("--dry-run"),
	}
	if value := parsed.FlagValue("--merged-path"); value != "" {
		req.MergedPath = value
	}
	if strings.TrimSpace(req.MergedPath) == "" {
		return HealFromSandboxRequest{}, clipolicy.UsageErrorf("scenario heal-from-sandbox", "heal-from-sandbox requires SANDBOX_MERGED_DIR or --merged-path")
	}
	return req, nil
}

func RenderHealFromSandboxResponse(w io.Writer, format cliout.Format, resp HealFromSandboxResponse) error {
	if len(resp.Affected) == 0 {
		return nil
	}
	if resp.DryRun {
		_, _ = fmt.Fprintf(w, "heal-from-sandbox: dry-run mode, would stop and restart: %s\n", strings.Join(resp.Affected, ", "))
		return nil
	}
	_, _ = fmt.Fprintf(w, "heal-from-sandbox: stopped and relaunched %d scenario(s)\n", len(resp.Affected))
	return nil
}

func containsArg(args []string, target string) bool {
	for _, arg := range args {
		if arg == target {
			return true
		}
	}
	return false
}

func FormatScenarioTemplateRequiredFlags(requiredVars map[string]TemplateVar) string {
	parts := make([]string, 0, len(requiredVars))
	keys := make([]string, 0, len(requiredVars))
	for key := range requiredVars {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		variable := requiredVars[key]
		flag := variable.Flag
		if flag == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf(" --%s <%s>", flag, strings.ToLower(key)))
	}
	return strings.Join(parts, "")
}
