package projectcli

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/vrooli/vrooli/internal/cli/clipolicy"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/onboardinghandoff"
	"github.com/vrooli/vrooli/internal/project"
	projectsetup "github.com/vrooli/vrooli/internal/setup"
	"github.com/vrooli/vrooli/internal/vroolierr"
)

const projectLifecycleExplain = "explain"

const (
	lifecycleHelp   = "--help"
	lifecycleStatus = "status"
)

func HelpOnlyWithoutRoot(args []string) bool {
	return len(args) == 0 || commandtree.WantsHelp(args)
}

type NoArgsRequest struct{}

type DoctorRequest struct {
	RepairFilePermissions bool
}

type StatusRequest struct {
	ResourcesOnly bool
	ScenariosOnly bool
	Fast          bool
}

type StopRequest struct {
	Targets []string
}

type OrphansRequest struct {
	Kill   bool
	DryRun bool
}

type LocksRequest struct {
	Clean bool
	// ShowAll includes expired claims in human-readable output. JSON output
	// is never filtered (machine consumers depend on the full set), so this
	// only affects rendering.
	ShowAll bool
}

type DiagnosePortRequest struct {
	Port         int
	ScenarioName string
}

type CleanupRequest struct {
	Target string
	Args   []string
}

type TemplateValidationCleanupRequest struct {
	DryRun          bool
	OlderThan       string
	IncludeRetained bool
	RunID           string
}

type LifecycleRequest struct {
	Subcommand string
	Args       []string
}

type ProjectPhaseRequest struct {
	Args []string
}

func ParseStatusRequest(args []string) (StatusRequest, error) {
	parsed, err := commandtree.ParseArgs(lifecycleStatus, StatusHelpText(), statusArgSchema(), args)
	if err != nil {
		return StatusRequest{}, err
	}
	req := StatusRequest{
		ResourcesOnly: parsed.HasFlag("--resources"),
		ScenariosOnly: parsed.HasFlag("--scenarios"),
		Fast:          true,
	}
	if parsed.HasFlag("--no-fast") {
		req.Fast = false
	}
	if parsed.HasFlag("--fast") {
		req.Fast = true
	}
	if req.ResourcesOnly && req.ScenariosOnly {
		return StatusRequest{}, clipolicy.UsageErrorf(lifecycleStatus, "status accepts only one of --resources or --scenarios")
	}
	return req, nil
}

func ParseDoctorRequest(args []string) (DoctorRequest, error) {
	parsed, err := commandtree.ParseArgs("doctor", DoctorHelpText(), commandtree.ArgSchema{Options: []commandtree.OptionArg{
		{Name: "--repair-file-permissions", Description: "Repair bounded runtime-home ownership mismatches after explicit opt-in"},
	}}, args)
	if err != nil {
		return DoctorRequest{}, err
	}
	return DoctorRequest{RepairFilePermissions: parsed.HasFlag("--repair-file-permissions")}, nil
}

func ParseStopRequest(args []string) (StopRequest, error) {
	parsed, err := commandtree.ParseArgs("stop", StopHelpText(), stopArgSchema(), args)
	if err != nil {
		return StopRequest{}, err
	}
	return StopRequest{Targets: append([]string(nil), parsed.Positionals...)}, nil
}

func ParseOrphansRequest(args []string) (OrphansRequest, error) {
	parsed, err := commandtree.ParseArgs("orphans", OrphansHelpText(), commandtree.ArgSchema{
		Positionals: []commandtree.PositionalArg{{Name: "action"}},
		Options: []commandtree.OptionArg{
			{Name: "--dry-run", Description: "With `kill`: list processes that would be terminated without sending signals"},
		},
	}, args)
	if err != nil {
		return OrphansRequest{}, err
	}
	req := OrphansRequest{DryRun: parsed.HasFlag("--dry-run")}
	if len(parsed.Positionals) == 1 {
		switch parsed.Positionals[0] {
		case "kill":
			req.Kill = true
		case lifecycleHelp:
			return OrphansRequest{}, clipolicy.CommandHelpOnly(OrphansHelpText())
		default:
			return OrphansRequest{}, clipolicy.UnknownOptionError("orphans", parsed.Positionals[0])
		}
	}
	if req.DryRun && !req.Kill {
		return OrphansRequest{}, clipolicy.UsageErrorf("orphans", "--dry-run is only meaningful with `kill`")
	}
	return req, nil
}

func ParseLocksRequest(args []string) (LocksRequest, error) {
	parsed, err := commandtree.ParseArgs("locks", LocksHelpText(), commandtree.ArgSchema{
		Positionals: []commandtree.PositionalArg{{Name: "action"}},
		Options: []commandtree.OptionArg{
			{Name: "--all", Description: "Include expired claims in the table (JSON always includes them)"},
		},
	}, args)
	if err != nil {
		return LocksRequest{}, err
	}
	req := LocksRequest{ShowAll: parsed.HasFlag("--all")}
	if len(parsed.Positionals) == 1 {
		switch parsed.Positionals[0] {
		case "clean":
			req.Clean = true
			if req.ShowAll {
				return LocksRequest{}, clipolicy.UsageErrorf("locks", "--all is only meaningful when listing claims, not with `clean`")
			}
		case lifecycleHelp:
			return LocksRequest{}, clipolicy.CommandHelpOnly(LocksHelpText())
		default:
			return LocksRequest{}, clipolicy.UnknownOptionError("locks", parsed.Positionals[0])
		}
	}
	return req, nil
}

func ParseDiagnosePortRequest(args []string) (DiagnosePortRequest, error) {
	parsed, err := commandtree.ParseArgs("diagnose-port", DiagnosePortHelpText(), diagnosePortArgSchema(), args)
	if err != nil {
		return DiagnosePortRequest{}, err
	}
	port, err := strconv.Atoi(strings.TrimSpace(parsed.Positionals[0]))
	if err != nil || port < 1 || port > 65535 {
		return DiagnosePortRequest{}, clipolicy.UsageErrorf("diagnose-port", "invalid port %q: must be an integer in [1, 65535]", parsed.Positionals[0])
	}
	req := DiagnosePortRequest{Port: port}
	if len(parsed.Positionals) > 1 {
		req.ScenarioName = parsed.Positionals[1]
	}
	return req, nil
}

func ParseCleanupRequest(args []string) (CleanupRequest, error) {
	parsed, err := commandtree.ParseArgs("cleanup", CleanupHelpText, commandtree.ArgSchema{
		Positionals: []commandtree.PositionalArg{
			{Name: "target", Required: true},
			{Name: "argument", Repeatable: true},
		},
		Options: []commandtree.OptionArg{
			{Name: "--dry-run", Description: "For `orphans`: list the processes that would be killed without sending signals"},
			{Name: "--older-than", ValueName: "duration", Description: "For `template-validation`: clean broad matches older than this Go duration"},
			{Name: "--include-retained", Description: "For `template-validation`: allow broad cleanup to remove retained debugging runs"},
			{Name: "--run", ValueName: "run-id", Description: "For `template-validation`: clean one explicit run id"},
		},
	}, args)
	if err != nil {
		return CleanupRequest{}, err
	}
	target := strings.TrimSpace(parsed.Positionals[0])
	forwarded := append([]string(nil), parsed.Positionals[1:]...)
	if parsed.HasFlag("--dry-run") {
		forwarded = append(forwarded, "--dry-run")
	}
	if value := strings.TrimSpace(parsed.FlagValue("--older-than")); value != "" {
		forwarded = append(forwarded, "--older-than", value)
	}
	if parsed.HasFlag("--include-retained") {
		forwarded = append(forwarded, "--include-retained")
	}
	if value := strings.TrimSpace(parsed.FlagValue("--run")); value != "" {
		forwarded = append(forwarded, "--run", value)
	}
	switch target {
	case lifecycleHelp:
		return CleanupRequest{}, clipolicy.CommandHelpOnly(CleanupHelpText)
	case "orphans", "locks", "template-validation":
		return CleanupRequest{Target: target, Args: forwarded}, nil
	default:
		return CleanupRequest{}, &vroolierr.Error{
			Err:         fmt.Errorf("unknown cleanup target: %s", target),
			Category:    clipolicy.ErrorCategoryUsage,
			Hint:        clipolicy.UsageHint("cleanup"),
			Suggestions: []string{"orphans", "locks", "template-validation"},
		}
	}
}

func ParseTemplateValidationCleanupRequest(args []string) (TemplateValidationCleanupRequest, error) {
	parsed, err := commandtree.ParseArgs("cleanup template-validation", CleanupHelpText, commandtree.ArgSchema{
		Options: []commandtree.OptionArg{
			{Name: "--dry-run", Description: "Preview cleanup without deleting files"},
			{Name: "--older-than", ValueName: "duration", Description: "Clean broad matches older than this Go duration; defaults to 24h"},
			{Name: "--include-retained", Description: "Allow broad cleanup to remove retained debugging runs"},
			{Name: "--run", ValueName: "run-id", Description: "Clean one explicit run id regardless of age or retained status"},
		},
	}, args)
	if err != nil {
		return TemplateValidationCleanupRequest{}, err
	}
	return TemplateValidationCleanupRequest{
		DryRun:          parsed.HasFlag("--dry-run"),
		OlderThan:       strings.TrimSpace(parsed.FlagValue("--older-than")),
		IncludeRetained: parsed.HasFlag("--include-retained"),
		RunID:           strings.TrimSpace(parsed.FlagValue("--run")),
	}, nil
}

func ParseSetupOptions(args []string) (projectsetup.Options, error) {
	if sub, rest, ok := extractSetupSubcommand(args); ok {
		switch sub {
		case lifecycleStatus:
			opts, err := parseLifecycleOptions("setup status", rest, SetupStatusHelpText())
			if err != nil {
				return projectsetup.Options{}, err
			}
			opts.Subcommand = lifecycleStatus
			return opts, nil
		case projectLifecycleExplain:
			parsed, err := commandtree.ParseArgs("setup explain", SetupExplainHelpText(), setupExplainArgSchema(), rest)
			if err != nil {
				return projectsetup.Options{}, err
			}
			if len(parsed.Positionals) != 1 {
				return projectsetup.Options{}, clipolicy.UsageErrorf("setup explain", "setup explain requires exactly one requirement name")
			}
			return projectsetup.Options{
				Subcommand:  "explain",
				ExplainName: strings.TrimSpace(parsed.Positionals[0]),
				Verbose:     parsed.HasFlag("--verbose"),
			}, nil
		}
	}
	return parseLifecycleOptions("setup", args, SetupHelpText())
}

// extractSetupSubcommand detects the leading positional lifecycleStatus or `explain`.
// Args before any flag are inspected; once a `--flag` appears we stop scanning.
func extractSetupSubcommand(args []string) (string, []string, bool) {
	for i, arg := range args {
		if strings.HasPrefix(arg, "-") {
			return "", nil, false
		}
		switch arg {
		case lifecycleStatus, "explain":
			rest := append([]string(nil), args[:i]...)
			rest = append(rest, args[i+1:]...)
			return arg, rest, true
		}
	}
	return "", nil, false
}

func setupExplainArgSchema() commandtree.ArgSchema {
	return commandtree.ArgSchema{
		Positionals: []commandtree.PositionalArg{{Name: "name", Required: true, Description: "Requirement name"}},
		Options: []commandtree.OptionArg{
			{Name: "--verbose", Description: "Currently a no-op for explain (kept for symmetry)"},
		},
	}
}

func SetupStatusHelpText() string {
	return commandtree.HelpText("", "vrooli setup status", "Inspect host requirements without applying changes.", commandtree.Help{}, lifecycleOptionsSchema())
}

func SetupExplainHelpText() string {
	return commandtree.HelpText("", "vrooli setup explain <name>", "Show full reasons, notes, and provenance for one requirement.", commandtree.Help{}, setupExplainArgSchema())
}

func ParseDevelopOptions(args []string) (projectsetup.Options, error) {
	return parseLifecycleOptions("develop", args, DevelopHelpText())
}

func ParseBuildRequest(args []string) (NoArgsRequest, error) {
	if _, err := commandtree.ParseArgs("build", BuildHelpText(), commandtree.ArgSchema{}, args); err != nil {
		return NoArgsRequest{}, err
	}
	return NoArgsRequest{}, nil
}

func ParseProjectPhaseRequest(phase string, args []string) (ProjectPhaseRequest, error) {
	if len(args) > 0 && (args[0] == lifecycleHelp || args[0] == "-h") {
		return ProjectPhaseRequest{}, clipolicy.CommandHelpOnly(ProjectPhaseHelpText(phase))
	}
	return ProjectPhaseRequest{Args: append([]string(nil), args...)}, nil
}

func ParseLifecycleRequest(args []string) (LifecycleRequest, error) {
	if len(args) == 0 || args[0] == lifecycleHelp || args[0] == "-h" {
		return LifecycleRequest{}, clipolicy.CommandHelpOnly(LifecycleHelpText())
	}
	switch args[0] {
	case "protect":
		return LifecycleRequest{Subcommand: "protect", Args: append([]string(nil), args[1:]...)}, nil
	default:
		return LifecycleRequest{}, clipolicy.UsageErrorf("lifecycle", "unknown lifecycle subcommand: %s", args[0])
	}
}

func ParseLifecycleProtectArgs(args []string) ([]string, error) {
	if len(args) == 0 {
		return nil, clipolicy.CommandHelpOnly(LifecycleProtectHelpText())
	}
	if args[0] == lifecycleHelp || args[0] == "-h" {
		return nil, clipolicy.CommandHelpOnly(LifecycleProtectHelpText())
	}
	if args[0] != "--" {
		return nil, clipolicy.UsageErrorf("lifecycle protect", "lifecycle protect requires '--' before the protected command")
	}
	if len(args) == 1 {
		return nil, clipolicy.UsageErrorf("lifecycle protect", "lifecycle protect requires a command after '--'")
	}
	return append([]string(nil), args[1:]...), nil
}

func parseLifecycleOptions(command string, args []string, helpText string) (projectsetup.Options, error) {
	parsed, err := commandtree.ParseArgs(command, helpText, lifecycleOptionsSchema(), args)
	if err != nil {
		return projectsetup.Options{}, err
	}
	opts := projectsetup.Options{DryRun: parsed.HasFlag("--dry-run")}
	opts.BootstrapOnly = parsed.HasFlag("--bootstrap-only")
	opts.CredentialPassphraseStdin = parsed.HasFlag("--credential-passphrase-stdin")
	if value := strings.ToLower(strings.TrimSpace(parsed.FlagValue("--sudo-mode"))); value != "" {
		switch value {
		case "ask", "skip", "error":
			opts.SudoMode = value
		default:
			return projectsetup.Options{}, fmt.Errorf("invalid value for --sudo-mode: %s", value)
		}
	}
	if value := strings.ToLower(strings.TrimSpace(parsed.FlagValue("--environment"))); value != "" {
		switch value {
		case "development", "production", "minimal":
			opts.Environment = value
		default:
			return projectsetup.Options{}, fmt.Errorf("invalid value for --environment: %s", value)
		}
	}
	if value := strings.ToLower(strings.TrimSpace(parsed.FlagValue("--resources"))); value != "" {
		opts.Resources = value
	}
	if value := strings.ToLower(strings.TrimSpace(parsed.FlagValue("--scenarios"))); value != "" {
		opts.Scenarios = value
	}
	if value := strings.ToLower(strings.TrimSpace(parsed.FlagValue("--yes"))); value != "" {
		opts.Yes = value
	}
	if value := strings.TrimSpace(parsed.FlagValue("--result-file")); value != "" {
		opts.ResultPath = value
	}
	if value := strings.TrimSpace(parsed.FlagValue("--onboarding")); value != "" {
		mode, modeErr := onboardinghandoff.ParseMode(value)
		if modeErr != nil {
			return projectsetup.Options{}, modeErr
		}
		opts.Onboarding = mode
	}
	if parsed.HasFlag("--include-optional") {
		opts.IncludeOptional = true
	}
	if parsed.HasFlag("--maintenance-window") {
		opts.MaintenanceWindow = true
	}
	return opts, nil
}

const CleanupHelpText = "vrooli cleanup - Clean up orphaned processes, stale registry claims, and template validation workspaces\n\nUsage:\n  vrooli cleanup orphans [--dry-run]                  Kill orphaned Vrooli processes\n  vrooli cleanup locks                                Expire stale registry leases/claims\n  vrooli cleanup template-validation [options]        Clean marker-backed deep template validation workspaces\n\nOptions:\n  --dry-run                 Preview cleanup without deleting files\n  --older-than <duration>   For `template-validation`: clean broad matches older than this Go duration (default 24h)\n  --include-retained        For `template-validation`: include retained debugging runs in broad cleanup\n  --run <run-id>            For `template-validation`: clean one explicit run id\n  --help, -h                Show this help message\n\nExamples:\n  vrooli cleanup orphans --dry-run                    # Preview which Vrooli processes would be killed\n  vrooli cleanup orphans                              # Kill orphaned Vrooli processes (SIGTERM, then SIGKILL)\n  vrooli cleanup locks                                # Expire stale registry leases and claims\n  vrooli cleanup template-validation --dry-run        # Preview stale template validation workspaces\n  vrooli cleanup template-validation --older-than 24h # Clean stale non-retained validation workspaces\n"

func statusArgSchema() commandtree.ArgSchema {
	return commandtree.ArgSchema{
		Options: []commandtree.OptionArg{
			commandtree.JSONOption(),
			{Name: "--resources", Description: "Show only resources"},
			{Name: "--scenarios", Description: "Show only scenarios"},
			{Name: "--fast", Description: "Use fast status probes"},
			{Name: "--no-fast", Description: "Disable fast status probes"},
		},
	}
}

func stopArgSchema() commandtree.ArgSchema {
	return commandtree.ArgSchema{
		Positionals: []commandtree.PositionalArg{{Name: "target", Repeatable: true}},
		Options:     []commandtree.OptionArg{commandtree.JSONOption()},
	}
}

func diagnosePortArgSchema() commandtree.ArgSchema {
	return commandtree.ArgSchema{
		Positionals: []commandtree.PositionalArg{
			{Name: "port", Required: true},
			{Name: "scenario"},
		},
		Options: []commandtree.OptionArg{commandtree.JSONOption(), {Name: "--repair-file-permissions", Description: "Repair bounded runtime-home ownership mismatches after explicit opt-in"}},
	}
}

func lifecycleOptionsSchema() commandtree.ArgSchema {
	return commandtree.ArgSchema{
		Options: []commandtree.OptionArg{
			{Name: "--dry-run", Description: "Preview actions without mutating the host"},
			{Name: "--bootstrap-only", Description: "Apply only host requirements needed to build a native CLI; internal bootstrap use"},
			{Name: "--credential-passphrase-stdin", Description: "Read the credential-store passphrase from standard input; internal automation use"},
			{Name: "--sudo-mode", ValueName: "mode", Description: "Sudo policy (ask|skip|error)"},
			{Name: "--maintenance-window", Description: "Acknowledge graphical/remote-session interruption risk"},
			{Name: "--environment", Aliases: []string{"--env"}, ValueName: "name", Description: "Environment profile (development|production|minimal)"},
			{Name: "--resources", ValueName: "value", Description: "Resource selection (enabled|none|comma,list)"},
			{Name: "--scenarios", ValueName: "value", Description: "Scenario selection (none|all|comma,list)"},
			{Name: "--yes", Aliases: []string{"-y"}, ValueName: "value", Description: "Confirmation policy forwarded to setup steps"},
			{Name: "--result-file", ValueName: "path", Description: "Write the versioned terminal setup result JSON to path"},
			{Name: "--onboarding", ValueName: "mode", Description: "Configuration handoff (auto|browser|cli|url|none)"},
			{Name: "--include-optional", Description: "Apply optional safeguards too (default: skip optional items)"},
		},
	}
}

func StatusHelpText() string {
	return commandtree.HelpText("", "vrooli status", "Show system health and status overview.", commandtree.Help{}, statusArgSchema())
}

func DoctorHelpText() string {
	return commandtree.HelpText("", "vrooli doctor", "Run environment and tool diagnostics.", commandtree.Help{}, commandtree.ArgSchema{
		Options: []commandtree.OptionArg{commandtree.JSONOption()},
	})
}

func StopHelpText() string {
	return commandtree.HelpText("", "vrooli stop", "Stop all or specific components.", commandtree.Help{}, stopArgSchema())
}

func OrphansHelpText() string {
	return commandtree.HelpText("", "vrooli orphans", "Inspect or clean orphaned Vrooli processes.", commandtree.Help{}, commandtree.ArgSchema{
		Positionals: []commandtree.PositionalArg{{Name: "action"}},
		Options: []commandtree.OptionArg{
			commandtree.JSONOption(),
			{Name: "--dry-run", Description: "With `kill`: list the processes that would be terminated without actually sending signals"},
		},
	})
}

func LocksHelpText() string {
	return commandtree.HelpText("", "vrooli locks", "Inspect runtime registry port claims.", commandtree.Help{}, commandtree.ArgSchema{
		Positionals: []commandtree.PositionalArg{{Name: "action"}},
		Options: []commandtree.OptionArg{
			commandtree.JSONOption(),
			{Name: "--all", Description: "Include expired claims in the table (JSON always includes them)"},
		},
	})
}

func DiagnosePortHelpText() string {
	return commandtree.HelpText("", "vrooli diagnose-port", "Diagnose port conflicts using registry claims, listener evidence, and legacy artifact hints.", commandtree.Help{}, diagnosePortArgSchema())
}

func BuildHelpText() string {
	return commandtree.HelpText("", "vrooli build", "Build the project-level Go binaries into .vrooli/build.", commandtree.Help{}, commandtree.ArgSchema{})
}

func SetupHelpText() string {
	return commandtree.HelpText("", "vrooli setup", "Initialize the development environment.", commandtree.Help{
		Notes: []string{
			"Subcommands:",
			"  vrooli setup status            Inspect host requirements without applying changes",
			"  vrooli setup explain <name>    Show full reasons / notes / declarer for one requirement",
			"",
			"Privilege & opt-in flags:",
			"  By default `vrooli setup` is non-interactive: items requiring privilege are",
			"    listed in the 'Needs privilege' group, not installed. To install them,",
			"    re-run as `vrooli setup --sudo-mode=ask`, which lets the in-process",
			"    `sudo` wrapper prompt for a password (interactive runs only). An",
			"    already-elevated invocation also installs them, but it runs the whole",
			"    of setup as root and is outside the onboarding flow.",
			"  --include-optional applies optional safeguards too. By default optional items",
			"    are listed but not installed (visible in the 'Optional' group).",
			"  --maintenance-window acknowledges graphical/remote-session interruption risk",
			"    for safeguards whose apply operation may interrupt the active session.",
			"  --onboarding=auto|browser|cli|url|none selects the configuration handoff.",
			"    `auto` opens a browser only when the invoking session can show one;",
			"    otherwise it prints the URL and an exact resume command.",
			"",
			"Pass --verbose (global) to switch the apply / status output to the legacy per-item block format.",
		},
	}, lifecycleOptionsSchema())
}

func DevelopHelpText() string {
	return commandtree.HelpText("", "vrooli develop", "Start development servers with optional auto-setup.", commandtree.Help{}, lifecycleOptionsSchema())
}

func ProjectPhaseHelpText(phase string) string {
	return commandtree.HelpText("", "vrooli "+phase, fmt.Sprintf("Run the %s lifecycle when defined.", phase), commandtree.Help{}, commandtree.ArgSchema{})
}

func LifecycleHelpText() string {
	return commandtree.HelpText("", "vrooli lifecycle", "Internal lifecycle command plumbing.", commandtree.Help{
		Usage: "vrooli lifecycle protect -- <command> [args...]",
	}, commandtree.ArgSchema{})
}

func LifecycleProtectHelpText() string {
	return commandtree.HelpText("", "vrooli lifecycle protect", "Run a command under lifecycle protection.", commandtree.Help{
		Usage: "vrooli lifecycle protect -- <command> [args...]",
	}, commandtree.ArgSchema{})
}

func LifecycleProtectErrorMessage() string {
	return "This UI must be run through the Vrooli lifecycle system.\n\nInstead, use:\n   vrooli scenario start <scenario-name>\n\nThe lifecycle system provides environment variables, port allocation,\nand dependency management automatically. Direct execution is not supported.\n"
}

func RenderStatusResponse(w io.Writer, format cliout.Format, resp StatusResponse) error {
	return RenderStatusReport(w, format, resp)
}

func RenderDoctorResponse(w io.Writer, format cliout.Format, resp project.DoctorReport) error {
	return RenderDoctorReport(w, format, resp)
}

func RenderStopResponse(w io.Writer, format cliout.Format, report control.StopReport) error {
	return RenderStopReport(w, format, report)
}

func RenderOrphansHelp(w io.Writer)   { commandtree.WriteHelp(w, OrphansHelpText()) }
func RenderLocksHelp(w io.Writer)     { commandtree.WriteHelp(w, LocksHelpText()) }
func RenderCleanupHelp(w io.Writer)   { commandtree.WriteHelp(w, CleanupHelpText) }
func RenderBuildHelp(w io.Writer)     { commandtree.WriteHelp(w, BuildHelpText()) }
func RenderSetupHelp(w io.Writer)     { commandtree.WriteHelp(w, SetupHelpText()) }
func RenderDevelopHelp(w io.Writer)   { commandtree.WriteHelp(w, DevelopHelpText()) }
func RenderLifecycleHelp(w io.Writer) { commandtree.WriteHelp(w, LifecycleHelpText()) }
func RenderLifecycleProtectHelp(w io.Writer) {
	commandtree.WriteHelp(w, LifecycleProtectHelpText())
}

func RenderProjectPhaseHelp(w io.Writer, phase string) {
	commandtree.WriteHelp(w, ProjectPhaseHelpText(phase))
}
