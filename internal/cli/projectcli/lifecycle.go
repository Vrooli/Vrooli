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
	"github.com/vrooli/vrooli/internal/project"
	projectsetup "github.com/vrooli/vrooli/internal/setup"
	"github.com/vrooli/vrooli/internal/vroolierr"
)

func HelpOnlyWithoutRoot(args []string) bool {
	return len(args) == 0 || commandtree.WantsHelp(args)
}

type NoArgsRequest struct{}

type StatusRequest struct {
	ResourcesOnly bool
	ScenariosOnly bool
	Fast          bool
}

type StopRequest struct {
	Targets []string
}

type OrphansRequest struct {
	Kill bool
}

type LocksRequest struct {
	Clean bool
}

type DiagnosePortRequest struct {
	Port         int
	ScenarioName string
}

type CleanupRequest struct {
	Target string
	Args   []string
}

type LifecycleRequest struct {
	Subcommand string
	Args       []string
}

type ProjectPhaseRequest struct {
	Args []string
}

func ParseStatusRequest(args []string) (StatusRequest, error) {
	req := StatusRequest{Fast: true}
	for _, arg := range args {
		switch arg {
		case "--help", "-h":
			return StatusRequest{}, clipolicy.CommandHelpOnly(StatusHelpText)
		case "--resources":
			req.ResourcesOnly = true
		case "--scenarios":
			req.ScenariosOnly = true
		case "--fast":
			req.Fast = true
		case "--no-fast":
			req.Fast = false
		default:
			return StatusRequest{}, clipolicy.UnknownOptionError("status", arg)
		}
	}
	if req.ResourcesOnly && req.ScenariosOnly {
		return StatusRequest{}, clipolicy.UsageErrorf("status", "status accepts only one of --resources or --scenarios")
	}
	return req, nil
}

func ParseDoctorRequest(args []string) (NoArgsRequest, error) {
	for _, arg := range args {
		switch arg {
		case "--help", "-h":
			return NoArgsRequest{}, clipolicy.CommandHelpOnly(DoctorHelpText)
		default:
			return NoArgsRequest{}, clipolicy.UnknownOptionError("doctor", arg)
		}
	}
	return NoArgsRequest{}, nil
}

func ParseStopRequest(args []string) (StopRequest, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return StopRequest{}, clipolicy.CommandHelpOnly(StopHelpText)
		}
	}
	return StopRequest{Targets: append([]string(nil), args...)}, nil
}

func ParseOrphansRequest(args []string) (OrphansRequest, error) {
	req := OrphansRequest{}
	for _, arg := range args {
		switch arg {
		case "kill":
			req.Kill = true
		case "--help", "-h", "help":
			return OrphansRequest{}, clipolicy.CommandHelpOnly(OrphansHelpText)
		default:
			return OrphansRequest{}, clipolicy.UnknownOptionError("orphans", arg)
		}
	}
	return req, nil
}

func ParseLocksRequest(args []string) (LocksRequest, error) {
	req := LocksRequest{}
	for _, arg := range args {
		switch arg {
		case "clean":
			req.Clean = true
		case "--help", "-h", "help":
			return LocksRequest{}, clipolicy.CommandHelpOnly(LocksHelpText)
		default:
			return LocksRequest{}, clipolicy.UnknownOptionError("locks", arg)
		}
	}
	return req, nil
}

func ParseDiagnosePortRequest(args []string) (DiagnosePortRequest, error) {
	if len(args) == 0 {
		return DiagnosePortRequest{}, clipolicy.UsageErrorf("diagnose-port", "usage: vrooli diagnose-port <port> [scenario] [--json]")
	}
	if args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		return DiagnosePortRequest{}, clipolicy.CommandHelpOnly(DiagnosePortHelpText)
	}
	port, err := strconv.Atoi(strings.TrimSpace(args[0]))
	if err != nil || port <= 0 {
		return DiagnosePortRequest{}, clipolicy.UsageErrorf("diagnose-port", "invalid port: %s", args[0])
	}
	req := DiagnosePortRequest{Port: port}
	if len(args) > 1 {
		req.ScenarioName = args[1]
	}
	return req, nil
}

func ParseCleanupRequest(args []string) (CleanupRequest, error) {
	if len(args) == 0 {
		return CleanupRequest{}, clipolicy.CommandHelpOnly(CleanupHelpText)
	}
	target := strings.TrimSpace(args[0])
	switch target {
	case "help", "--help", "-h":
		return CleanupRequest{}, clipolicy.CommandHelpOnly(CleanupHelpText)
	case "orphans", "locks":
		return CleanupRequest{Target: target, Args: append([]string(nil), args[1:]...)}, nil
	default:
		return CleanupRequest{}, &vroolierr.Error{
			Err:         fmt.Errorf("unknown cleanup target: %s", target),
			Category:    clipolicy.ErrorCategoryUsage,
			Hint:        clipolicy.UsageHint("cleanup"),
			Suggestions: []string{"orphans", "locks"},
		}
	}
}

func ParseSetupOptions(args []string) (projectsetup.Options, error) {
	return parseLifecycleOptions("setup", args, SetupHelpText)
}

func ParseDevelopOptions(args []string) (projectsetup.Options, error) {
	return parseLifecycleOptions("develop", args, DevelopHelpText)
}

func ParseBuildRequest(args []string) (NoArgsRequest, error) {
	for _, arg := range args {
		switch arg {
		case "--help", "-h":
			return NoArgsRequest{}, clipolicy.CommandHelpOnly(BuildHelpText)
		default:
			return NoArgsRequest{}, clipolicy.UnknownOptionError("build", arg)
		}
	}
	return NoArgsRequest{}, nil
}

func ParseProjectPhaseRequest(phase string, args []string) (ProjectPhaseRequest, error) {
	if len(args) > 0 && (args[0] == "--help" || args[0] == "-h") {
		return ProjectPhaseRequest{}, clipolicy.CommandHelpOnly(ProjectPhaseHelpText(phase))
	}
	return ProjectPhaseRequest{Args: append([]string(nil), args...)}, nil
}

func ParseLifecycleRequest(args []string) (LifecycleRequest, error) {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		return LifecycleRequest{}, clipolicy.CommandHelpOnly(LifecycleHelpText)
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
		return nil, clipolicy.CommandHelpOnly(LifecycleProtectHelpText)
	}
	if args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		return nil, clipolicy.CommandHelpOnly(LifecycleProtectHelpText)
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
	opts := projectsetup.Options{}
	for index := 0; index < len(args); index++ {
		arg := args[index]
		switch {
		case arg == "--help" || arg == "-h":
			return projectsetup.Options{}, clipolicy.CommandHelpOnly(helpText)
		case arg == "--dry-run":
			opts.DryRun = true
		case arg == "--sudo-mode":
			value, next, err := requireValue(command, arg, args, index)
			if err != nil {
				return projectsetup.Options{}, err
			}
			index = next
			value = strings.ToLower(strings.TrimSpace(value))
			switch value {
			case "ask", "skip", "error":
				opts.SudoMode = value
			default:
				return projectsetup.Options{}, fmt.Errorf("invalid value for --sudo-mode: %s", value)
			}
		case strings.HasPrefix(arg, "--sudo-mode="):
			value := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(arg, "--sudo-mode=")))
			switch value {
			case "ask", "skip", "error":
				opts.SudoMode = value
			default:
				return projectsetup.Options{}, fmt.Errorf("invalid value for --sudo-mode: %s", value)
			}
		case arg == "--environment" || arg == "--env":
			value, next, err := requireValue(command, arg, args, index)
			if err != nil {
				return projectsetup.Options{}, err
			}
			index = next
			value = strings.ToLower(strings.TrimSpace(value))
			switch value {
			case "development", "production", "minimal":
				opts.Environment = value
			default:
				return projectsetup.Options{}, fmt.Errorf("invalid value for --environment: %s", value)
			}
		case strings.HasPrefix(arg, "--environment="):
			value := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(arg, "--environment=")))
			switch value {
			case "development", "production", "minimal":
				opts.Environment = value
			default:
				return projectsetup.Options{}, fmt.Errorf("invalid value for --environment: %s", value)
			}
		case strings.HasPrefix(arg, "--env="):
			value := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(arg, "--env=")))
			switch value {
			case "development", "production", "minimal":
				opts.Environment = value
			default:
				return projectsetup.Options{}, fmt.Errorf("invalid value for --environment: %s", value)
			}
		case arg == "--resources":
			value, next, err := requireValue(command, arg, args, index)
			if err != nil {
				return projectsetup.Options{}, err
			}
			index = next
			opts.Resources = strings.ToLower(strings.TrimSpace(value))
		case strings.HasPrefix(arg, "--resources="):
			opts.Resources = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(arg, "--resources=")))
		case arg == "--scenarios":
			value, next, err := requireValue(command, arg, args, index)
			if err != nil {
				return projectsetup.Options{}, err
			}
			index = next
			opts.Scenarios = strings.ToLower(strings.TrimSpace(value))
		case strings.HasPrefix(arg, "--scenarios="):
			opts.Scenarios = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(arg, "--scenarios=")))
		case arg == "--yes" || arg == "-y":
			value, next, err := requireValue(command, arg, args, index)
			if err != nil {
				return projectsetup.Options{}, err
			}
			index = next
			opts.Yes = strings.ToLower(strings.TrimSpace(value))
		case strings.HasPrefix(arg, "--yes="):
			opts.Yes = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(arg, "--yes=")))
		default:
			return projectsetup.Options{}, clipolicy.UnknownOptionError(command, arg)
		}
	}
	return opts, nil
}

func requireValue(command, flag string, args []string, index int) (string, int, error) {
	if index+1 >= len(args) {
		return "", index, fmt.Errorf("%s requires a value for %s", command, flag)
	}
	return args[index+1], index + 1, nil
}

const (
	StatusHelpText           = "Usage: vrooli status [--resources|--scenarios] [--fast|--no-fast] [--json]"
	DoctorHelpText           = "Usage: vrooli doctor [--json]"
	StopHelpText             = "Usage: vrooli stop [all|scenarios|resources|scenario:<name>|resource:<name>|<name>...] [--json]"
	OrphansHelpText          = "Usage: vrooli orphans [kill] [--json]"
	LocksHelpText            = "Usage: vrooli locks [clean] [--json]"
	DiagnosePortHelpText     = "Usage: vrooli diagnose-port <port> [scenario] [--json]"
	BuildHelpText            = "Usage: vrooli build\n\nBuilds the project-level Go binaries into .vrooli/build.\n"
	SetupHelpText            = "Usage: vrooli setup [options]\n\nOptions:\n  --environment, --env <name>   Set environment profile (development|production|minimal)\n  --resources <value>           Resource selection (enabled|none|comma,list)\n  --scenarios <value>           Scenario selection (none|all|comma,list)\n  --sudo-mode <mode>            Sudo policy (ask|skip|error)\n  --yes <value>                 Confirmation policy forwarded to setup steps\n  --dry-run                     Preview setup actions without mutating the host\n"
	DevelopHelpText          = "Usage: vrooli develop [options]\n\nOptions:\n  --environment, --env <name>   Set environment profile for auto-setup (development|production|minimal)\n  --resources <value>           Resource selection for auto-setup (enabled|none|comma,list)\n  --scenarios <value>           Scenario selection for auto-setup (none|all|comma,list)\n  --sudo-mode <mode>            Sudo policy for auto-setup (ask|skip|error)\n  --yes <value>                 Confirmation policy forwarded to auto-setup\n  --dry-run                     Preview auto-setup actions without mutating the host\n"
	CleanupHelpText          = "vrooli cleanup - Clean up orphaned processes and stale locks\n\nUsage:\n  vrooli cleanup orphans    Kill orphaned Vrooli processes\n  vrooli cleanup locks      Clean stale port lock files\n\nOptions:\n  --help, -h    Show this help message\n\nExamples:\n  vrooli cleanup orphans    # Kill orphaned processes (interactive)\n  vrooli cleanup locks      # Remove stale lock files\n"
	LifecycleHelpText        = "Usage: vrooli lifecycle protect -- <command> [args...]\n"
	LifecycleProtectHelpText = "Usage: vrooli lifecycle protect -- <command> [args...]\n"
)

func ProjectPhaseHelpText(phase string) string {
	return fmt.Sprintf("Usage: vrooli %s\n", phase)
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

func RenderOrphansHelp(w io.Writer)   { _, _ = io.WriteString(w, OrphansHelpText+"\n") }
func RenderLocksHelp(w io.Writer)     { _, _ = io.WriteString(w, LocksHelpText+"\n") }
func RenderCleanupHelp(w io.Writer)   { _, _ = io.WriteString(w, CleanupHelpText) }
func RenderBuildHelp(w io.Writer)     { _, _ = io.WriteString(w, BuildHelpText) }
func RenderSetupHelp(w io.Writer)     { _, _ = io.WriteString(w, SetupHelpText) }
func RenderDevelopHelp(w io.Writer)   { _, _ = io.WriteString(w, DevelopHelpText) }
func RenderLifecycleHelp(w io.Writer) { _, _ = io.WriteString(w, LifecycleHelpText) }
func RenderLifecycleProtectHelp(w io.Writer) {
	_, _ = io.WriteString(w, LifecycleProtectHelpText)
}

func RenderProjectPhaseHelp(w io.Writer, phase string) {
	_, _ = io.WriteString(w, ProjectPhaseHelpText(phase))
}
