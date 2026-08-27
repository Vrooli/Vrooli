package scenariocli

import (
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/vrooli/internal/cli/clipolicy"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/resources"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

const (
	commandsHelp = "--help"
)

// instanceOption is the shared `--instance <variant>` flag that lets every
// scenario-addressing command target a named instance (e.g. a shadow). It is
// sugar for the canonical `name@variant` argument; both resolve through the one
// scenarioruntime.ParseInstanceKey parser (§1a), and supplying them with
// conflicting values is a hard error.
func instanceOption() commandtree.OptionArg {
	return commandtree.OptionArg{Name: "--instance", ValueName: "variant", Description: "Target a named instance/variant (e.g. shadow); equivalent to name@variant"}
}

// nodeOption is the explicit node selector shared by every command that
// already accepts --instance. A node/ prefix in the scenario address wins
// over this flag. There is deliberately no ambient node environment signal.
func nodeOption() commandtree.OptionArg {
	return commandtree.OptionArg{Name: "--node", ValueName: "name", Description: "Target a connected node; address prefix wins; node is never selected implicitly"}
}

// resolveAddressArg resolves the instance axis and preserves the explicit
// node axis in the canonical address string. Existing local callers receive
// the same scenario[@variant] slug they received before node addressing was
// added. A node prefix is authoritative when both forms are present.
func resolveAddressArg(command, name, instanceFlag, nodeFlag string) (string, error) {
	prefixNode, scenario, suffixVariant, err := cliutil.SplitAddress(name)
	if err != nil {
		return "", clipolicy.UsageErrorf("scenario "+command, "%s", err.Error())
	}
	instanceName := scenario
	if suffixVariant != "" {
		instanceName += "@" + suffixVariant
	}
	key, err := scenarioruntime.ParseInstanceKey(instanceName, instanceFlag)
	if err != nil {
		return "", clipolicy.UsageErrorf("scenario "+command, "%s", err.Error())
	}
	node := strings.TrimSpace(prefixNode)
	if node == "" {
		node = strings.TrimSpace(nodeFlag)
	}
	if node == "" {
		return key.Slug(), nil
	}
	return node + "/" + key.Slug(), nil
}

type CommandID string

const (
	CommandList            CommandID = "list"
	CommandInfo            CommandID = "info"
	CommandStatus          CommandID = "status"
	CommandValidateEnv     CommandID = "validate-env"
	CommandFreshness       CommandID = "freshness"
	CommandTimings         CommandID = "timings"
	CommandRun             CommandID = "run"
	CommandStart           CommandID = "start"
	CommandStartAll        CommandID = "start-all"
	CommandSetup           CommandID = "setup"
	CommandRestart         CommandID = "restart"
	CommandStop            CommandID = "stop"
	CommandDelete          CommandID = "delete"
	CommandWait            CommandID = "wait"
	CommandStopAll         CommandID = "stop-all"
	CommandTest            CommandID = "test"
	CommandLogs            CommandID = "logs"
	CommandScreenshot      CommandID = "screenshot"
	CommandOpen            CommandID = "open"
	CommandPort            CommandID = "port"
	CommandRequirements    CommandID = "requirements"
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
			Args: commandtree.ArgSchema{Positionals: []commandtree.PositionalArg{{Name: "scenario name"}}, Options: []commandtree.OptionArg{commandtree.JSONOption(), instanceOption(), nodeOption()}},
		},
		{
			Name: string(CommandValidateEnv), Group: "Read-only Commands", Summary: "Validate resource-derived environment injection for a scenario", Handler: CommandValidateEnv, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot},
			Args: commandtree.ArgSchema{Positionals: []commandtree.PositionalArg{{Name: "scenario name", Required: true}}, Options: []commandtree.OptionArg{commandtree.JSONOption()}},
		},
		{
			Name: string(CommandFreshness), Group: "Read-only Commands", Summary: "Explain why a scenario's build artifacts are fresh or stale", Handler: CommandFreshness, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot},
			Args: commandtree.ArgSchema{
				Positionals: []commandtree.PositionalArg{{Name: "scenario name", Required: true}},
				Options:     []commandtree.OptionArg{commandtree.JSONOption(), {Name: "--explain", Description: "Print every check (fresh and stale) plus resolved dependency policies"}, {Name: "--path", ValueName: "path"}},
			},
		},
		{
			Name: string(CommandTimings), Group: "Read-only Commands", Summary: "Show retained start and restart timing aggregates", Handler: CommandTimings, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot},
			Args: commandtree.ArgSchema{Options: []commandtree.OptionArg{{Name: "--scenario", ValueName: "name", Description: "Limit timing rows to one scenario"}, commandtree.JSONOption()}},
		},
		{Name: string(CommandRun), Group: "Lifecycle and Utility Commands", Summary: "Run a scenario directly (alias of start)", Handler: CommandRun, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}},
		{
			Name: string(CommandStart), Group: "Lifecycle and Utility Commands", Summary: "Start a scenario", Handler: CommandStart, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot},
			Args: commandtree.ArgSchema{
				Positionals: []commandtree.PositionalArg{{Name: "scenario name", Required: true, Repeatable: true}},
				Options:     []commandtree.OptionArg{{Name: "--path", ValueName: "path"}, {Name: "--best-effort"}, {Name: "--clean-stale"}, {Name: "--force", Description: "Rebuild artifacts even when their inputs are fresh"}, {Name: "--open"}, {Name: "--timeout", ValueName: "seconds", Description: "Ceiling for the whole start (not the expected duration); on expiry exit 124 — the operation record stays honest and the next start/wait resumes"}, commandtree.JSONOption(), instanceOption(), nodeOption()},
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
				Options:     []commandtree.OptionArg{{Name: "--path", ValueName: "path"}, {Name: "--best-effort"}, {Name: "--clean-stale"}, {Name: "--force", Description: "Rebuild artifacts even when their inputs are fresh"}, {Name: "--open"}, {Name: "--timeout", ValueName: "seconds", Description: "Ceiling for the whole restart; on expiry exit 124 — the operation record stays honest and the next start/wait resumes"}, commandtree.JSONOption(), instanceOption(), nodeOption()},
			},
		},
		{
			Name: string(CommandWait), Group: "Lifecycle and Utility Commands", Summary: "Block once until a scenario's in-flight start finishes (anti-polling)", Handler: CommandWait, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot},
			Args: commandtree.ArgSchema{
				Positionals: []commandtree.PositionalArg{{Name: "scenario name", Required: true}},
				Options: []commandtree.OptionArg{
					{Name: "--timeout", ValueName: "seconds", Description: "Wait CEILING in seconds (not the expected duration); on expiry exit 124 and the start keeps running. Size it as ETA + 75% buffer."},
					commandtree.JSONOption(),
					instanceOption(),
					nodeOption(),
				},
			},
		},
		{
			Name: string(CommandStop), Group: "Lifecycle and Utility Commands", Summary: "Stop a running scenario", Handler: CommandStop, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot},
			Args: commandtree.ArgSchema{Positionals: []commandtree.PositionalArg{{Name: "scenario name", Required: true}}, Options: []commandtree.OptionArg{commandtree.JSONOption(), instanceOption(), nodeOption()}},
		},
		{
			Name: string(CommandDelete), Group: "Lifecycle and Utility Commands", Summary: "Delete a scenario and its installed CLI triple", Handler: CommandDelete, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot},
			Args: commandtree.ArgSchema{Positionals: []commandtree.PositionalArg{{Name: "scenario name", Required: true}}, Options: []commandtree.OptionArg{commandtree.JSONOption(), {Name: "--yes", Description: "Confirm deletion of the scenario source and installed CLI artifacts"}}},
		},
		{
			Name: string(CommandStopAll), Group: "Lifecycle and Utility Commands", Summary: "Stop all running scenarios", Handler: CommandStopAll, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot},
			Args: commandtree.ArgSchema{Options: []commandtree.OptionArg{commandtree.JSONOption()}},
		},
		{Name: string(CommandTest), Group: "Lifecycle and Utility Commands", Summary: "Run scenario tests", Handler: CommandTest, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}},
		{Name: string(CommandLogs), Group: "Lifecycle and Utility Commands", Summary: "View logs for a scenario", Handler: CommandLogs, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}},
		{
			Name: string(CommandScreenshot), Group: "Lifecycle and Utility Commands", Summary: "Capture a screenshot without a shell", Handler: CommandScreenshot, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot},
			Args: commandtree.ArgSchema{Options: []commandtree.OptionArg{{Name: "--output", ValueName: "path", Description: "Output PNG path"}, commandtree.JSONOption()}},
		},
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
				Options:     []commandtree.OptionArg{commandtree.JSONOption(), instanceOption(), nodeOption(), {Name: "--path", ValueName: "path", Description: "Resolve a running scenario started from this physical scenario directory"}},
			},
		},
		{Name: string(CommandRequirements), Group: "Lifecycle and Utility Commands", Summary: "Manage scenario requirements", Handler: CommandRequirements, Suggestable: true, RootPolicy: commandtree.RootPolicy{RequiresRoot: true, CanRunWithoutRoot: HelpOnlyWithoutRoot}},
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
	// TimeoutSeconds is the ceiling for the whole start; 0 = unbounded.
	TimeoutSeconds int
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
		// TimeoutSeconds is the ceiling for the whole restart; 0 = unbounded.
		TimeoutSeconds int
	}
)

type ScreenshotRequest struct {
	Output string
	JSON   bool
}

type WaitRequest struct {
	Name           string
	TimeoutSeconds int
	JSON           bool
}

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
		Args []string
	}
	StartAllRequest struct{ JSON bool }
	StopAllRequest  struct{ JSON bool }
	DeleteRequest   struct {
		Name string
		JSON bool
		Yes  bool
	}
	DeleteResponse struct {
		Name             string   `json:"name"`
		ScenarioPath     string   `json:"scenario_path,omitempty"`
		RemovedArtifacts int      `json:"removed_artifacts"`
		SkippedArtifacts []string `json:"skipped_artifacts,omitempty"`
	}
	PortRequest struct {
		ScenarioName, PortName string
		Path                   string
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

func RenderDeleteResponse(w io.Writer, format cliout.Format, resp DeleteResponse) error {
	return cliout.RenderJSONOr(w, format, func(w io.Writer) error { return cliout.WriteJSON(w, resp) }, func(w io.Writer) error {
		_, err := fmt.Fprintf(w, "Deleted scenario %s; removed %d installed CLI artifact(s).\n", resp.Name, resp.RemovedArtifacts)
		if err == nil && len(resp.SkippedArtifacts) > 0 {
			_, err = fmt.Fprintf(w, "Skipped locked CLI artifact(s): %s\n", strings.Join(resp.SkippedArtifacts, ", "))
		}
		return err
	})
}

func ParseScenarioNameAndJSON(command string, defaultJSON bool, args []string) (string, bool, error) {
	name, jsonFlag, err := parseOptionalScenarioNameAndJSONWithHelp(command, defaultJSON, "", args)
	if err != nil {
		return "", false, fmt.Errorf("parse scenario %s arguments: %w", command, err)
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
		return "", false, fmt.Errorf("parse scenario %s arguments: %w", command, err)
	}
	name := ""
	if len(parsed.Positionals) == 1 {
		name = parsed.Positionals[0]
	}
	return name, defaultJSON || parsed.HasFlag("--json"), nil
}

// ScenarioStartArgs is the parsed shape shared by start/restart/run.
type ScenarioStartArgs struct {
	Names          []string
	Options        lifecycle.StartOptions
	JSON           bool
	OpenAfter      bool
	TimeoutSeconds int
}

func ParseScenarioStartArgs(defaultJSON bool, args []string) (ScenarioStartArgs, error) {
	spec := commandSpec(CommandStart)
	parsed, err := commandtree.ParseArgs("scenario start", commandHelpText(CommandStart), spec.Args, args)
	if err != nil {
		return ScenarioStartArgs{}, err
	}
	out := ScenarioStartArgs{
		Options: lifecycle.StartOptions{
			BestEffort: parsed.HasFlag("--best-effort"),
			CleanStale: parsed.HasFlag("--clean-stale"),
			ForceSetup: parsed.HasFlag("--force"),
			CustomPath: parsed.FlagValue("--path"),
		},
		JSON:      defaultJSON || parsed.HasFlag("--json"),
		OpenAfter: parsed.HasFlag("--open"),
	}
	if raw := strings.TrimSpace(parsed.FlagValue("--timeout")); raw != "" {
		out.TimeoutSeconds, err = strconv.Atoi(raw)
		if err != nil || out.TimeoutSeconds <= 0 {
			return ScenarioStartArgs{}, clipolicy.UsageErrorf("scenario start", "--timeout must be a positive number of seconds, got %q", raw)
		}
	}
	instanceFlag := parsed.FlagValue("--instance")
	nodeFlag := parsed.FlagValue("--node")
	out.Names = make([]string, 0, len(parsed.Positionals))
	for _, positional := range parsed.Positionals {
		slug, err := resolveAddressArg("start", positional, instanceFlag, nodeFlag)
		if err != nil {
			return ScenarioStartArgs{}, err
		}
		out.Names = append(out.Names, slug)
	}
	return out, nil
}

func ParseScenarioSingleStartArgs(command string, defaultJSON bool, args []string) (ScenarioStartArgs, error) {
	parsed, err := ParseScenarioStartArgs(defaultJSON, args)
	if err != nil {
		return ScenarioStartArgs{}, err
	}
	if len(parsed.Names) == 0 {
		return ScenarioStartArgs{}, clipolicy.UsageErrorf("scenario "+command, "scenario %s requires a scenario name", command)
	}
	if len(parsed.Names) > 1 {
		return ScenarioStartArgs{}, clipolicy.UsageErrorf("scenario "+command, "scenario %s accepts exactly one scenario name", command)
	}
	return parsed, nil
}

func ParseStartRequest(globalsJSON bool, args []string) (StartRequest, error) {
	for _, arg := range args {
		if arg == commandsHelp || arg == "-h" {
			return StartRequest{}, clipolicy.CommandHelpOnly(commandHelpText(CommandStart))
		}
	}
	parsed, err := ParseScenarioStartArgs(globalsJSON, args)
	if err != nil {
		return StartRequest{}, err
	}
	if len(parsed.Names) == 0 {
		return StartRequest{}, clipolicy.UsageErrorf("scenario start", "scenario start requires at least one scenario name")
	}
	if parsed.Options.CustomPath != "" && len(parsed.Names) != 1 {
		return StartRequest{}, clipolicy.UsageErrorf("scenario start", "scenario start with --path accepts exactly one scenario name")
	}
	return StartRequest(parsed), nil
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
	slug, err := resolveAddressArg("stop", parsed.Positionals[0], parsed.FlagValue("--instance"), parsed.FlagValue("--node"))
	if err != nil {
		return StopRequest{}, err
	}
	return StopRequest{Name: slug, JSON: globalsJSON || parsed.HasFlag("--json")}, nil
}

func ParseDeleteRequest(globalsJSON bool, args []string) (DeleteRequest, error) {
	spec := commandSpec(CommandDelete)
	parsed, err := commandtree.ParseArgs("scenario delete", commandHelpText(CommandDelete), spec.Args, args)
	if err != nil {
		return DeleteRequest{}, err
	}
	if len(parsed.Positionals) == 0 {
		return DeleteRequest{}, clipolicy.UsageErrorf("scenario delete", "scenario delete requires a scenario name")
	}
	if !parsed.HasFlag("--yes") {
		return DeleteRequest{}, clipolicy.UsageErrorf("scenario delete", "scenario delete requires --yes")
	}
	name := strings.TrimSpace(parsed.Positionals[0])
	if name == "" || filepath.Base(name) != name || strings.ContainsAny(name, `/\\@`) {
		return DeleteRequest{}, clipolicy.UsageErrorf("scenario delete", "scenario name must be a plain live scenario name")
	}
	return DeleteRequest{Name: name, JSON: globalsJSON || parsed.HasFlag("--json"), Yes: true}, nil
}

func ParseScreenshotRequest(globalsJSON bool, args []string) (ScreenshotRequest, error) {
	spec := commandSpec(CommandScreenshot)
	parsed, err := commandtree.ParseArgs("scenario screenshot", commandHelpText(CommandScreenshot), spec.Args, args)
	if err != nil {
		return ScreenshotRequest{}, err
	}
	output := strings.TrimSpace(parsed.FlagValue("--output"))
	if output == "" {
		return ScreenshotRequest{}, clipolicy.UsageErrorf("scenario screenshot", "--output is required")
	}
	return ScreenshotRequest{Output: output, JSON: globalsJSON || parsed.HasFlag("--json")}, nil
}

func ParseRestartRequest(globalsJSON bool, args []string) (RestartRequest, error) {
	for _, arg := range args {
		if arg == commandsHelp || arg == "-h" {
			return RestartRequest{}, clipolicy.CommandHelpOnly(commandHelpText(CommandRestart))
		}
	}
	parsed, err := ParseScenarioSingleStartArgs("restart", globalsJSON, args)
	if err != nil {
		return RestartRequest{}, err
	}
	return RestartRequest{Name: parsed.Names[0], Options: parsed.Options, JSON: parsed.JSON, OpenAfter: parsed.OpenAfter, TimeoutSeconds: parsed.TimeoutSeconds}, nil
}

func ParseWaitRequest(globalsJSON bool, args []string) (WaitRequest, error) {
	spec := commandSpec(CommandWait)
	parsed, err := commandtree.ParseArgs("scenario wait", waitHelpText(), spec.Args, args)
	if err != nil {
		return WaitRequest{}, err
	}
	if len(parsed.Positionals) == 0 {
		return WaitRequest{}, clipolicy.UsageErrorf("scenario wait", "scenario wait requires a scenario name")
	}
	slug, err := resolveAddressArg("wait", parsed.Positionals[0], parsed.FlagValue("--instance"), parsed.FlagValue("--node"))
	if err != nil {
		return WaitRequest{}, err
	}
	timeoutSeconds := 0
	if raw := strings.TrimSpace(parsed.FlagValue("--timeout")); raw != "" {
		timeoutSeconds, err = strconv.Atoi(raw)
		if err != nil || timeoutSeconds <= 0 {
			return WaitRequest{}, clipolicy.UsageErrorf("scenario wait", "--timeout must be a positive number of seconds, got %q", raw)
		}
	}
	return WaitRequest{Name: slug, TimeoutSeconds: timeoutSeconds, JSON: globalsJSON || parsed.HasFlag("--json")}, nil
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
		if name, err = resolveAddressArg("status", parsed.Positionals[0], parsed.FlagValue("--instance"), parsed.FlagValue("--node")); err != nil {
			return StatusRequest{}, err
		}
	} else if parsed.FlagValue("--instance") != "" || parsed.FlagValue("--node") != "" {
		return StatusRequest{}, clipolicy.UsageErrorf("scenario status", "scenario status --instance/--node requires a scenario name")
	}
	return StatusRequest{Name: name, JSON: globalsJSON || parsed.HasFlag("--json")}, nil
}

func ParseFreshnessRequest(globalsJSON bool, args []string) (FreshnessRequest, error) {
	spec := commandSpec(CommandFreshness)
	parsed, err := commandtree.ParseArgs("scenario freshness", commandHelpText(CommandFreshness), spec.Args, args)
	if err != nil {
		return FreshnessRequest{}, err
	}
	if len(parsed.Positionals) == 0 {
		return FreshnessRequest{}, clipolicy.UsageErrorf("scenario freshness", "scenario freshness requires a scenario name")
	}
	return FreshnessRequest{
		Name:    parsed.Positionals[0],
		Path:    parsed.FlagValue("--path"),
		JSON:    globalsJSON || parsed.HasFlag("--json"),
		Explain: parsed.HasFlag("--explain"),
	}, nil
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
	return cliout.RenderJSONOr(w, format, func(w io.Writer) error { return cliout.WriteProtoJSON(w, ScenarioEnvValidationResponse(resp.Report)) }, func(w io.Writer) error {
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
	})
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
				return "", lifecycle.PhaseOptions{}, clipolicy.UsageErrorf("scenario "+command, "scenario %s does not accept %s", command, arg)
			}
			if name == "" {
				name = arg
			} else {
				return "", lifecycle.PhaseOptions{}, clipolicy.UsageErrorf("scenario "+command, "scenario %s accepts only a scenario name and --path", command)
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
		case commandsHelp, "-h":
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

func ParseTestArgs(_, _ bool, args []string) (TestRequest, error) {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return TestRequest{}, clipolicy.UsageErrorf("scenario test", "scenario test requires a scenario name")
	}
	return TestRequest{Name: args[0], Args: append([]string(nil), args[1:]...)}, nil
}

func ParseTestRequest(globalsJSON, globalsVerbose bool, args []string) (TestRequest, error) {
	if len(args) > 0 && (args[0] == commandsHelp || args[0] == "-h") {
		return TestRequest{}, clipolicy.CommandHelpOnly(TestHelpText())
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
	slug, err := resolveAddressArg("port", parsed.Positionals[0], parsed.FlagValue("--instance"), parsed.FlagValue("--node"))
	if err != nil {
		return PortRequest{}, err
	}
	req := PortRequest{
		ScenarioName: slug,
		Path:         parsed.FlagValue("--path"),
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

func RenderRequirementsResponse(w io.Writer, format cliout.Format, _ struct{}) error { return nil }

func ParseRequirementsRequest(args []string) (RequirementsRequest, error) {
	if len(args) == 0 || args[0] == "help" || args[0] == commandsHelp || args[0] == "-h" {
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
