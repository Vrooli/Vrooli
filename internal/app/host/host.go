package hostapp

import (
	"context"
	"fmt"
	"strings"

	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/internal/operatorstate"
	"github.com/vrooli/vrooli/internal/workloadowner"
)

// Run dispatches `vrooli host`.
func (app *App) Run(ctx *CommandContext, args []string) error {
	return app.runHostCommand(ctx, args)
}

// RunWorkload dispatches `vrooli workload`.
func (app *App) RunWorkload(ctx *CommandContext, args []string) error {
	return app.runWorkloadCommand(ctx, args)
}

func (app *App) runWorkloadCommand(ctx *CommandContext, args []string) error {
	if len(args) == 0 || commandWantsHelp(args) {
		commandtree.RenderHelp(ctx.Stdout, commandtree.Help{Title: "Vrooli Workloads", Description: "Observe host workloads and classify them against Vrooli declarations.", Usage: "vrooli workload list [--posture whole_host|vrooli_only] [--json]"}, []commandtree.Spec[string]{{Name: "list", Summary: "List observed workloads", Handler: "list"}})
		return nil
	}
	if args[0] != "list" {
		return rootcli.NewUnknownCommandError(args[0], []string{"list"})
	}
	posture := workloadowner.VrooliOnly
	postureExplicit := false
	for i := 1; i < len(args); i++ {
		if args[i] == "--posture" && i+1 < len(args) {
			posture = workloadowner.Posture(args[i+1])
			postureExplicit = true
			i++
		}
	}
	if !postureExplicit {
		state, stateErr := operatorstate.New(operatorstate.Config{RepoRoot: ctx.Root}).Load(context.Background())
		if stateErr != nil {
			return fmt.Errorf("load host workload posture: %w", stateErr)
		}
		if state.HostWorkloadPosture == string(workloadowner.WholeHost) {
			posture = workloadowner.WholeHost
		}
	}
	census, err := hostinventory.CollectWorkloads(context.Background())
	if err != nil {
		return err
	}
	observed := workloadowner.LiveReport{Unread: census.Unread, EvidenceNote: "classification is computed from hostinventory observations and enabled resource manifests"}
	observed.Observed = append(observed.Observed, census.Containers...)
	observed.Observed = append(observed.Observed, census.ServiceUnits...)
	observed.Observed = append(observed.Observed, census.ScheduledTasks...)
	declarations, declarationErr := workloadowner.DeclarationsFromRoot(ctx.Root)
	if declarationErr != nil {
		observed.Unread = append(observed.Unread, "declarations: "+declarationErr.Error())
	}
	observed.Report = workloadowner.Classify(observed.Observed, declarations, posture, 2700)
	workloadowner.RedactForPosture(&observed.Report)
	if ctx.Globals.JSON || containsArg(args, "--json") {
		return writeHostJSON(ctx.Stdout, observed)
	}
	rows := append([]workloadowner.Finding{}, observed.Report.Declared...)
	rows = append(rows, observed.Report.Findings...)
	rows = append(rows, observed.Report.Informational...)
	for _, finding := range rows {
		_, _ = fmt.Fprintf(ctx.Stdout, "%s\t%s\t%s\t%s\n", finding.Class, finding.Kind, finding.Name, finding.Reason)
		for _, evidence := range finding.Evidence {
			_, _ = fmt.Fprintf(ctx.Stdout, "\t evidence: %s\n", evidence)
		}
		if finding.ProposedAction != "" {
			_, _ = fmt.Fprintf(ctx.Stdout, "\t proposed_action: %s\n", finding.ProposedAction)
		}
	}
	for _, note := range observed.Unread {
		_, _ = fmt.Fprintf(ctx.Stdout, "unread\t%s\n", note)
	}
	return nil
}

func containsArg(args []string, wanted string) bool {
	for _, arg := range args {
		if arg == wanted {
			return true
		}
	}
	return false
}

func (app *App) runHostCommand(ctx *CommandContext, args []string) error {
	if len(args) == 0 || commandWantsHelp(args) {
		commandtree.RenderHelp(ctx.Stdout, commandtree.Help{Title: "Vrooli Host", Description: "Inspect local host facts through internal/hostinventory.", Usage: "vrooli host <command> [options]", DefaultGroup: "Host Commands"}, []commandtree.Spec[string]{hostInventorySpec(), hostInstallSpec(), hostSafeguardSpec(), hostVolumeSpec(), hostStorageCandidatesSpec()})
		return nil
	}
	switch args[0] {
	case "inventory":
		return app.runHostInventoryCommand(ctx, args[1:])
	case "install":
		return app.runHostInstallCommand(ctx, args[1:])
	case "safeguard":
		return app.runHostSafeguardCommand(ctx, args[1:])
	case "volume":
		return app.runHostVolumeCommand(ctx, args[1:])
	case "storage":
		return app.runHostStorageCommand(ctx, args[1:])
	default:
		return rootcli.NewUnknownCommandError(args[0], []string{"inventory", "install", "safeguard", "volume", "storage"})
	}
}

func hostInventorySpec() commandtree.Spec[string] {
	return commandtree.Spec[string]{Name: "inventory", Summary: "Collect local host inventory facts", Help: commandtree.Help{Description: "Collects CPU, memory, GPU, and Docker GPU-runtime facts using the shared Go host inventory package.", Usage: "vrooli host inventory [--json] [--field <name>]", Options: []commandtree.OptionArg{commandtree.JSONOption(), {Name: "--field", ValueName: "name", Description: "Print one shell-friendly field"}}, Examples: []string{"vrooli host inventory --json", "vrooli host inventory --field has_nvidia_gpu", "vrooli host inventory --field memory_total_mb"}}, Args: commandtree.ArgSchema{Options: []commandtree.OptionArg{commandtree.JSONOption(), {Name: "--field", ValueName: "name", Description: "Print one shell-friendly field"}}}, Handler: "inventory"}
}

func (app *App) runHostInventoryCommand(ctx *CommandContext, args []string) error {
	spec := hostInventorySpec()
	parsed, err := commandtree.ParseArgs("host inventory", commandtree.SpecHelpText("", "vrooli host inventory", spec), spec.Args, args)
	if err != nil {
		if rootcli.HandleHelp(ctx.Stdout, err) {
			return nil
		}
		return rootcli.UsageErrorf("host inventory", "%s", err.Error())
	}
	snapshot, err := hostinventory.Collect(context.Background())
	if err != nil {
		return err
	}
	if field := strings.TrimSpace(parsed.FlagValue("--field")); field != "" {
		value, err := hostInventoryField(snapshot, field)
		if err != nil {
			return rootcli.UsageErrorf("host inventory", "%s", err.Error())
		}
		_, _ = fmt.Fprintln(ctx.Stdout, value)
		return nil
	}
	if ctx.Globals.JSON || parsed.HasFlag("--json") {
		return writeHostSnapshotJSON(ctx.Stdout, snapshot)
	}
	_, _ = fmt.Fprintf(ctx.Stdout, "OS: %s/%s\n", snapshot.OS, snapshot.Arch)
	_, _ = fmt.Fprintf(ctx.Stdout, "CPU cores: %d\n", snapshot.CPU.Cores)
	_, _ = fmt.Fprintf(ctx.Stdout, "Memory total: %d MB\n", snapshot.Memory.TotalBytes/1024/1024)
	_, _ = fmt.Fprintf(ctx.Stdout, "NVIDIA GPU: %s\n", cliout.BoolLabel(snapshot.HasNvidiaGPU()))
	_, _ = fmt.Fprintf(ctx.Stdout, "Docker NVIDIA runtime: %s\n", cliout.BoolLabel(snapshot.HasDockerNvidiaRuntime()))
	if len(snapshot.GPUs) > 0 {
		_, _ = fmt.Fprintln(ctx.Stdout, "GPUs:")
		for _, gpu := range snapshot.GPUs {
			_, _ = fmt.Fprintf(ctx.Stdout, "- %s (%d MB, source=%s)\n", gpu.Name, gpu.VRAMBytes/1024/1024, gpu.Source)
		}
	}
	if len(snapshot.Warnings) > 0 {
		_, _ = fmt.Fprintln(ctx.Stdout, "Warnings:")
		for _, warning := range snapshot.Warnings {
			_, _ = fmt.Fprintf(ctx.Stdout, "- %s\n", warning)
		}
	}
	return nil
}

func hostInventoryField(snapshot hostinventory.Snapshot, field string) (string, error) {
	switch field {
	case "has_nvidia_gpu":
		return fmt.Sprintf("%t", snapshot.HasNvidiaGPU()), nil
	case "has_docker_nvidia_runtime":
		return fmt.Sprintf("%t", snapshot.HasDockerNvidiaRuntime()), nil
	case "has_docker_addressable_nvidia_gpu":
		return fmt.Sprintf("%t", snapshot.HasDockerAddressableNvidiaGPU()), nil
	case "gpu_count":
		return fmt.Sprintf("%d", len(snapshot.GPUs)), nil
	case "first_gpu_summary":
		for _, gpu := range snapshot.GPUs {
			if gpu.Name != "" {
				return fmt.Sprintf("%s,%d,%d", gpu.Name, gpu.VRAMUsedBytes/1024/1024, gpu.VRAMBytes/1024/1024), nil
			}
		}
		return "", nil
	case "cpu_cores":
		return fmt.Sprintf("%d", snapshot.CPU.Cores), nil
	case "memory_total_mb":
		return fmt.Sprintf("%d", snapshot.Memory.TotalBytes/1024/1024), nil
	case "memory_available_mb":
		return fmt.Sprintf("%d", snapshot.Memory.AvailableBytes/1024/1024), nil
	default:
		return "", fmt.Errorf("unknown host inventory field %q", field)
	}
}

func commandWantsHelp(args []string) bool {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return true
		}
	}
	return false
}
