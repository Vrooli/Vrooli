package scenariocli

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/resources"
)

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
	CommandTemplate        CommandID = "template"
	CommandGenerate        CommandID = "generate"
	CommandCompleteness    CommandID = "completeness"
	CommandHealFromSandbox CommandID = "heal-from-sandbox"
)

type helpOnlyError struct {
	text string
}

func (e helpOnlyError) Error() string    { return e.text }
func (e helpOnlyError) HelpText() string { return e.text }

func commandHelpOnly(text string) error {
	return helpOnlyError{text: text}
}

func usageErrorf(helpTarget, format string, args ...any) error {
	return fmt.Errorf(format, args...)
}

func unknownOptionError(command, option string) error {
	return fmt.Errorf("unknown option for %s: %s", command, option)
}

func CommandSpecs() []commandtree.Spec[CommandID] {
	return []commandtree.Spec[CommandID]{
		{Name: string(CommandList), Group: "Read-only Commands", Summary: "List discovered scenarios", Handler: CommandList, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}},
		{Name: string(CommandInfo), Group: "Read-only Commands", Summary: "Show scenario metadata and runtime summary", Handler: CommandInfo, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}},
		{Name: string(CommandStatus), Group: "Read-only Commands", Summary: "Show scenario runtime status", Handler: CommandStatus, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}},
		{Name: string(CommandValidateEnv), Group: "Read-only Commands", Summary: "Validate resource-derived environment injection for a scenario", Handler: CommandValidateEnv, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}},
		{Name: string(CommandRun), Group: "Lifecycle and Utility Commands", Summary: "Run a scenario directly (alias of start)", Handler: CommandRun, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}},
		{Name: string(CommandStart), Group: "Lifecycle and Utility Commands", Summary: "Start a scenario", Handler: CommandStart, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}},
		{Name: string(CommandStartAll), Group: "Lifecycle and Utility Commands", Summary: "Start all available scenarios", Handler: CommandStartAll, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}},
		{Name: string(CommandSetup), Group: "Lifecycle and Utility Commands", Summary: "Run the setup lifecycle", Handler: CommandSetup, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}},
		{Name: string(CommandRestart), Group: "Lifecycle and Utility Commands", Summary: "Restart a scenario", Handler: CommandRestart, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}},
		{Name: string(CommandStop), Group: "Lifecycle and Utility Commands", Summary: "Stop a running scenario", Handler: CommandStop, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}},
		{Name: string(CommandStopAll), Group: "Lifecycle and Utility Commands", Summary: "Stop all running scenarios", Handler: CommandStopAll, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}},
		{Name: string(CommandTest), Group: "Lifecycle and Utility Commands", Summary: "Run scenario tests", Handler: CommandTest, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}},
		{Name: string(CommandLogs), Group: "Lifecycle and Utility Commands", Summary: "View logs for a scenario", Handler: CommandLogs, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}},
		{Name: string(CommandOpen), Group: "Lifecycle and Utility Commands", Summary: "Open a scenario in the browser", Handler: CommandOpen, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}},
		{Name: string(CommandPort), Group: "Lifecycle and Utility Commands", Summary: "Show running port assignments", Handler: CommandPort, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}},
		{Name: string(CommandUISmoke), Group: "Lifecycle and Utility Commands", Summary: "Run the Browserless UI smoke harness", Handler: CommandUISmoke, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}},
		{Name: string(CommandRequirements), Group: "Lifecycle and Utility Commands", Summary: "Manage scenario requirements", Handler: CommandRequirements, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}},
		{Name: string(CommandTemplate), Group: "Lifecycle and Utility Commands", Summary: "Manage scenario templates", Handler: CommandTemplate, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}},
		{Name: string(CommandGenerate), Group: "Lifecycle and Utility Commands", Summary: "Scaffold a scenario from a template", Handler: CommandGenerate, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}},
		{Name: string(CommandCompleteness), Group: "Lifecycle and Utility Commands", Summary: "Calculate a completeness score", Handler: CommandCompleteness, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}},
		{Name: string(CommandHealFromSandbox), Group: "Lifecycle and Utility Commands", Summary: "Relaunch sandbox-rooted scenario processes", Handler: CommandHealFromSandbox, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}},
	}
}

func HelpOnlyWithoutRoot(args []string) bool { return commandtree.WantsHelp(args) }

func RenderCommandHelp(w io.Writer) {
	_, _ = fmt.Fprintln(w, "Vrooli Scenario Commands")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Usage:")
	_, _ = fmt.Fprintln(w, "  vrooli scenario <subcommand> [options]")
	_, _ = fmt.Fprintln(w)
	commandtree.RenderGroups(w, commandtree.VisibleEntries(CommandSpecs(), ""))
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
		Name string
		Opts lifecycle.PhaseOptions
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
	name, jsonFlag, err := ParseOptionalScenarioNameAndJSON(command, defaultJSON, args)
	if err != nil {
		return "", false, err
	}
	if name == "" {
		return "", false, usageErrorf("scenario "+command, "scenario %s requires a scenario name", command)
	}
	return name, jsonFlag, nil
}

func ParseOptionalScenarioNameAndJSON(command string, defaultJSON bool, args []string) (string, bool, error) {
	name := ""
	jsonFlag := defaultJSON
	for _, arg := range args {
		switch arg {
		case "--json":
			jsonFlag = true
		case "--help", "-h":
			return "", false, commandHelpOnly("")
		default:
			if strings.HasPrefix(arg, "-") {
				return "", false, unknownOptionError("scenario "+command, arg)
			}
			if name != "" {
				return "", false, usageErrorf("scenario "+command, "scenario %s accepts at most one scenario name", command)
			}
			name = arg
		}
	}
	return name, jsonFlag, nil
}

func ParseScenarioStartArgs(defaultJSON bool, args []string) ([]string, lifecycle.StartOptions, bool, bool, error) {
	names := []string{}
	jsonFlag := defaultJSON
	openAfter := false
	opts := lifecycle.StartOptions{}
	for index := 0; index < len(args); index++ {
		arg := args[index]
		switch arg {
		case "--json":
			jsonFlag = true
		case "--open":
			openAfter = true
		case "--best-effort":
			opts.BestEffort = true
		case "--clean-stale":
			opts.CleanStale = true
		case "--path":
			if index+1 >= len(args) {
				return nil, lifecycle.StartOptions{}, false, false, usageErrorf("scenario start", "scenario start --path requires a value")
			}
			index++
			opts.CustomPath = args[index]
		case "--help", "-h":
			return nil, lifecycle.StartOptions{}, false, false, commandHelpOnly("")
		default:
			if strings.HasPrefix(arg, "-") {
				return nil, lifecycle.StartOptions{}, false, false, unknownOptionError("scenario start", arg)
			}
			names = append(names, arg)
		}
	}
	return names, opts, jsonFlag, openAfter, nil
}

func ParseScenarioSingleStartArgs(command string, defaultJSON bool, args []string) (string, lifecycle.StartOptions, bool, bool, error) {
	names, opts, jsonFlag, openAfter, err := ParseScenarioStartArgs(defaultJSON, args)
	if err != nil {
		return "", lifecycle.StartOptions{}, false, false, err
	}
	if len(names) == 0 {
		return "", lifecycle.StartOptions{}, false, false, usageErrorf("scenario "+command, "scenario %s requires a scenario name", command)
	}
	if len(names) > 1 {
		return "", lifecycle.StartOptions{}, false, false, usageErrorf("scenario "+command, "scenario %s accepts exactly one scenario name", command)
	}
	return names[0], opts, jsonFlag, openAfter, nil
}

func ParseStartRequest(globalsJSON bool, args []string) (StartRequest, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return StartRequest{}, commandHelpOnly(StartHelpText)
		}
	}
	names, opts, jsonFlag, openAfter, err := ParseScenarioStartArgs(globalsJSON, args)
	if err != nil {
		return StartRequest{}, err
	}
	if len(names) == 0 {
		return StartRequest{}, usageErrorf("scenario start", "scenario start requires at least one scenario name")
	}
	if opts.CustomPath != "" && len(names) != 1 {
		return StartRequest{}, usageErrorf("scenario start", "scenario start with --path accepts exactly one scenario name")
	}
	return StartRequest{Names: names, Options: opts, JSON: jsonFlag, OpenAfter: openAfter}, nil
}

func ParseStopRequest(globalsJSON bool, args []string) (StopRequest, error) {
	name, jsonFlag, err := ParseScenarioNameAndJSON("stop", globalsJSON, args)
	if err != nil {
		return StopRequest{}, err
	}
	return StopRequest{Name: name, JSON: jsonFlag}, nil
}

func ParseRestartRequest(globalsJSON bool, args []string) (RestartRequest, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return RestartRequest{}, commandHelpOnly(RestartHelpText)
		}
	}
	name, opts, jsonFlag, openAfter, err := ParseScenarioSingleStartArgs("restart", globalsJSON, args)
	if err != nil {
		return RestartRequest{}, err
	}
	return RestartRequest{Name: name, Options: opts, JSON: jsonFlag, OpenAfter: openAfter}, nil
}

func ParseListRequest(globalsJSON bool, args []string) (ListRequest, error) {
	req := ListRequest{JSON: globalsJSON}
	for _, arg := range args {
		switch arg {
		case "--json":
			req.JSON = true
		case "--include-ports":
			req.IncludePorts = true
		case "--help", "-h":
			return ListRequest{}, commandHelpOnly(ListHelpText)
		default:
			return ListRequest{}, unknownOptionError("scenario list", arg)
		}
	}
	return req, nil
}

func ParseInfoRequest(globalsJSON bool, args []string) (InfoRequest, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return InfoRequest{}, commandHelpOnly(InfoHelpText)
		}
	}
	name, jsonFlag, err := ParseScenarioNameAndJSON("info", globalsJSON, args)
	if err != nil {
		return InfoRequest{}, err
	}
	return InfoRequest{Name: name, JSON: jsonFlag}, nil
}

func ParseStatusRequest(globalsJSON bool, args []string) (StatusRequest, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return StatusRequest{}, commandHelpOnly(StatusHelpText)
		}
	}
	name, jsonFlag, err := ParseOptionalScenarioNameAndJSON("status", globalsJSON, args)
	if err != nil {
		return StatusRequest{}, err
	}
	return StatusRequest{Name: name, JSON: jsonFlag}, nil
}

func ParseValidateEnvRequest(globalsJSON bool, args []string) (ValidateEnvRequest, error) {
	name, jsonFlag, err := ParseScenarioNameAndJSON("validate-env", globalsJSON, args)
	if err != nil {
		return ValidateEnvRequest{}, err
	}
	return ValidateEnvRequest{Name: name, JSON: jsonFlag}, nil
}

func RenderValidateEnvResponse(w io.Writer, format cliout.Format, resp ValidateEnvResponse) error {
	if format == cliout.FormatJSON {
		return cliout.WriteJSON(w, map[string]any{"success": resp.Report.Passed, "report": resp.Report})
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
				return "", lifecycle.PhaseOptions{}, usageErrorf("scenario "+command, "scenario %s --path requires a value", command)
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
		return "", lifecycle.PhaseOptions{}, usageErrorf("scenario "+command, "scenario %s requires a scenario name", command)
	}
	return name, opts, nil
}

func ParseSetupRequest(globalsJSON bool, args []string) (SetupRequest, error) {
	jsonFlag := globalsJSON
	for _, arg := range args {
		switch arg {
		case "--help", "-h":
			return SetupRequest{}, commandHelpOnly(SetupHelpText)
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

func ParseTestArgs(globalsJSON, globalsVerbose bool, args []string) (string, lifecycle.PhaseOptions, error) {
	name := ""
	selection := ""
	opts := lifecycle.PhaseOptions{}
	remaining := []string{}
	for index := 0; index < len(args); index++ {
		arg := args[index]
		switch arg {
		case "--path":
			if index+1 >= len(args) {
				return "", lifecycle.PhaseOptions{}, usageErrorf("scenario test", "scenario test --path requires a value")
			}
			index++
			opts.CustomPath = args[index]
		case "--allow-skip-missing-runtime":
			opts.AllowSkipMissingRuntime = true
		case "--manage-runtime":
			opts.ManageRuntime = true
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
		return "", lifecycle.PhaseOptions{}, usageErrorf("scenario test", "scenario test requires a scenario name")
	}
	if selection != "" {
		valid := map[string]string{"structure": "structure", "dependencies": "dependencies", "unit": "unit", "integration": "integration", "business": "business", "performance": "performance", "all": "all", "e2e": "integration"}
		mapped, ok := valid[selection]
		if !ok {
			return "", lifecycle.PhaseOptions{}, usageErrorf("scenario test", "invalid test selector: %s", selection)
		}
		remaining = append([]string{mapped}, remaining...)
	}
	if globalsJSON && !containsArg(remaining, "--json") {
		remaining = append(remaining, "--json")
	}
	if globalsVerbose && !containsArg(remaining, "--verbose") {
		remaining = append(remaining, "--verbose")
	}
	opts.Args = remaining
	return name, opts, nil
}

func ParseTestRequest(globalsJSON, globalsVerbose bool, args []string) (TestRequest, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return TestRequest{}, commandHelpOnly(TestHelpText)
		}
	}
	name, opts, err := ParseTestArgs(globalsJSON, globalsVerbose, args)
	if err != nil {
		return TestRequest{}, err
	}
	return TestRequest{Name: name, Opts: opts}, nil
}

func ParseStartAllRequest(globalsJSON bool, args []string) (StartAllRequest, error) {
	req := StartAllRequest{JSON: globalsJSON}
	for _, arg := range args {
		switch arg {
		case "--json":
			req.JSON = true
		case "--help", "-h":
			return StartAllRequest{}, commandHelpOnly(StartAllHelpText)
		default:
			return StartAllRequest{}, unknownOptionError("scenario start-all", arg)
		}
	}
	return req, nil
}

func ParseStopAllRequest(globalsJSON bool, args []string) (StopAllRequest, error) {
	req := StopAllRequest{JSON: globalsJSON}
	for _, arg := range args {
		switch arg {
		case "--json":
			req.JSON = true
		case "--help", "-h":
			return StopAllRequest{}, commandHelpOnly(StopAllHelpText)
		default:
			return StopAllRequest{}, unknownOptionError("scenario stop-all", arg)
		}
	}
	return req, nil
}

func ParsePortRequest(globalsJSON bool, args []string) (PortRequest, error) {
	req := PortRequest{JSON: globalsJSON}
	for _, arg := range args {
		switch {
		case arg == "--json":
			req.JSON = true
		case arg == "--help" || arg == "-h":
			return PortRequest{}, commandHelpOnly(PortHelpText)
		case strings.HasPrefix(arg, "-"):
			return PortRequest{}, unknownOptionError("scenario port", arg)
		case req.ScenarioName == "":
			req.ScenarioName = arg
		case req.PortName == "":
			req.PortName = arg
		default:
			return PortRequest{}, usageErrorf("scenario port", "scenario port accepts at most two positional arguments")
		}
	}
	if req.ScenarioName == "" {
		return PortRequest{}, usageErrorf("scenario port", "scenario port requires a scenario name")
	}
	return req, nil
}

func ParseOpenRequest(globalsJSON bool, args []string) (OpenRequest, error) {
	req := OpenRequest{PortName: "UI_PORT", JSON: globalsJSON}
	for index := 0; index < len(args); index++ {
		arg := args[index]
		switch arg {
		case "--help", "-h":
			return OpenRequest{}, commandHelpOnly(OpenHelpText)
		case "--port":
			if index+1 >= len(args) {
				return OpenRequest{}, usageErrorf("scenario open", "scenario open --port requires a value")
			}
			index++
			req.PortName = args[index]
		case "--print-url":
			req.PrintURL = true
		case "--json":
			req.JSON = true
		default:
			if strings.HasPrefix(arg, "-") {
				return OpenRequest{}, unknownOptionError("scenario open", arg)
			}
			if req.ScenarioName != "" {
				return OpenRequest{}, usageErrorf("scenario open", "scenario open accepts exactly one scenario name")
			}
			req.ScenarioName = arg
		}
	}
	if req.ScenarioName == "" {
		return OpenRequest{}, usageErrorf("scenario open", "scenario open requires a scenario name")
	}
	return req, nil
}

func RenderRequirementsResponse(w io.Writer, format cliout.Format, _ struct{}) error { return nil }

func ParseRequirementsRequest(args []string) (RequirementsRequest, error) {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		return RequirementsRequest{}, commandHelpOnly(RequirementsHelpText())
	}
	req := RequirementsRequest{Args: append([]string(nil), args...)}
	if args[0] == "snapshot" {
		req.Snapshot = true
	}
	return req, nil
}

func RequirementsHelpText() string {
	return RequirementsHelpTextBody
}

func ParseHealFromSandboxRequest(defaultMergedPath string, args []string) (HealFromSandboxRequest, error) {
	req := HealFromSandboxRequest{MergedPath: strings.TrimSpace(defaultMergedPath)}
	for index := 0; index < len(args); index++ {
		switch args[index] {
		case "--merged-path":
			if index+1 >= len(args) {
				return HealFromSandboxRequest{}, usageErrorf("scenario heal-from-sandbox", "scenario heal-from-sandbox --merged-path requires a value")
			}
			index++
			req.MergedPath = args[index]
		case "--dry-run":
			req.DryRun = true
		case "--help", "-h":
			return HealFromSandboxRequest{}, commandHelpOnly(HealFromSandboxHelpText)
		default:
			return HealFromSandboxRequest{}, unknownOptionError("scenario heal-from-sandbox", args[index])
		}
	}
	if strings.TrimSpace(req.MergedPath) == "" {
		return HealFromSandboxRequest{}, usageErrorf("scenario heal-from-sandbox", "heal-from-sandbox requires SANDBOX_MERGED_DIR or --merged-path")
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
