package hosthandlers

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliapp"
	climanifest "github.com/vrooli/vrooli/cli"
	hostapp "github.com/vrooli/vrooli/internal/app/host"
	"github.com/vrooli/vrooli/internal/cli/commandtree"
	"github.com/vrooli/vrooli/internal/cli/manifestdispatch"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	hostruntime "github.com/vrooli/vrooli/internal/runtime"
	"github.com/vrooli/vrooli/internal/volumeremediation"
	"github.com/vrooli/vrooli/internal/workloadowner"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

type hostService struct {
	run func([]string) error
}

const hostMemoryBytesPerMegabyte = 1024

var hostCommandNames = []string{"cron", "inventory", "desktop-session", "install", "safeguard", "volume", "storage"}

// RegisteredCommandPaths returns the child paths bound by the host handler.
func RegisteredCommandPaths() []string {
	paths := make([]string, 0, len(hostCommandNames)+1)
	for _, name := range hostCommandNames {
		paths = append(paths, "host "+name)
	}
	return append(paths, "workload list")
}

// RootHandler dispatches `vrooli host` through the host application.
func RootHandler[C any](deps rootcli.HandlerDeps[C]) rootcli.Handler[C] {
	return rootcli.BindService(deps.Stdout,
		func(C) (cliout.Format, error) { return cliout.FormatHuman, nil },
		func(ctx C, _ cliout.Format) (hostService, error) {
			commandCtx := &rootcli.CommandContext{Root: deps.Root(ctx), Globals: deps.Globals(ctx), Stdout: deps.Stdout(ctx), Stderr: deps.Stderr(ctx), Context: rootcli.ResolveOperationContext(deps, ctx)}
			return hostService{run: func(args []string) error {
				group, err := cliapp.LoadFromManifest(climanifest.Bytes(), "host", hostBindings(commandCtx, hostCommandNames))
				if err != nil {
					return err
				}
				core := cliapp.NewApp(cliapp.AppOptions{Name: "vrooli host", Commands: []cliapp.CommandGroup{{Commands: group.Subcommands}}})
				return core.RunWithWriters(manifestdispatch.WithJSON(args, commandCtx.Globals.JSON), deps.Stdout(ctx), deps.Stderr(ctx))
			}}, nil
		},
		func(_ C, args []string) ([]string, error) { return args, nil },
		func(service hostService, args []string) (struct{}, error) { return struct{}{}, service.run(args) },
		func(io.Writer, cliout.Format, struct{}) error { return nil },
	)
}

// WorkloadHandler dispatches `vrooli workload` through the host application.
func WorkloadHandler[C any](deps rootcli.HandlerDeps[C]) rootcli.Handler[C] {
	return rootcli.BindService(deps.Stdout,
		func(C) (cliout.Format, error) { return cliout.FormatHuman, nil },
		func(ctx C, _ cliout.Format) (hostService, error) {
			commandCtx := &rootcli.CommandContext{Root: deps.Root(ctx), Globals: deps.Globals(ctx), Stdout: deps.Stdout(ctx), Stderr: deps.Stderr(ctx), Context: rootcli.ResolveOperationContext(deps, ctx)}
			return hostService{run: func(args []string) error { return runWorkload(commandCtx, args) }}, nil
		},
		func(_ C, args []string) ([]string, error) { return args, nil },
		func(service hostService, args []string) (struct{}, error) { return struct{}{}, service.run(args) },
		func(io.Writer, cliout.Format, struct{}) error { return nil },
	)
}

func runWorkload(ctx *rootcli.CommandContext, args []string) error {
	spec := commandtree.Spec[string]{
		Name: "list", Args: commandtree.ArgSchema{Options: []commandtree.OptionArg{{Name: "--json"}, {Name: "--posture", ValueName: "posture"}}}, Handler: "list",
	}
	if len(args) == 0 || commandtree.WantsHelp(args) {
		commandtree.RenderHelp(ctx.Stdout, commandtree.Help{Title: "Vrooli Workloads", Description: "Observe host workloads and classify them against Vrooli declarations.", Usage: "vrooli workload list [--posture whole_host|vrooli_only] [--json]"}, []commandtree.Spec[string]{spec})
		return nil
	}
	if args[0] != "list" {
		return fmt.Errorf("unknown workload command %q", args[0])
	}
	parsed, err := commandtree.ParseArgs("workload list", "", spec.Args, args[1:])
	if err != nil {
		return err
	}
	posture := workloadowner.VrooliOnly
	if value := strings.TrimSpace(parsed.FlagValue("--posture")); value != "" {
		posture = workloadowner.Posture(value)
	}
	report, err := (hostapp.Service{}).CollectWorkloads(ctx.OperationContext(), hostapp.WorkloadOptions{Root: ctx.Root, Posture: posture, PostureExplicit: parsed.FlagValue("--posture") != ""})
	if err != nil {
		return err
	}
	if ctx.Globals.JSON || parsed.HasFlag("--json") {
		return cliout.WriteJSONValue(ctx.Stdout, report)
	}
	rows := make([][]string, 0, len(report.Report.Declared)+len(report.Report.Findings)+len(report.Report.Informational)+len(report.Unread))
	for _, finding := range append(append(append([]workloadowner.Finding{}, report.Report.Declared...), report.Report.Findings...), report.Report.Informational...) {
		rows = append(rows, []string{string(finding.Class), finding.Kind, finding.Name, finding.Reason})
	}
	for _, note := range report.Unread {
		rows = append(rows, []string{"unread", note})
	}
	return cliout.WriteSection(ctx.Stdout, cliout.Section{Rows: rows})
}

func hostBindings(ctx *rootcli.CommandContext, names []string) map[string]func(cliapp.RunContext) error {
	bindings := make(map[string]func(cliapp.RunContext) error, len(names))
	for _, name := range names {
		command := name
		bindings[name] = func(command string) func(cliapp.RunContext) error {
			return func(runCtx cliapp.RunContext) error {
				switch command {
				case "cron":
					return runHostCron(runCtx, ctx)
				case "inventory":
					return runHostInventory(runCtx, ctx)
				case "desktop-session":
					return runDesktopSession(runCtx, ctx)
				case "install":
					return runHostInstall(runCtx, ctx)
				case "safeguard":
					return runHostSafeguard(runCtx, ctx)
				case "storage":
					return runHostStorage(runCtx, ctx)
				case "volume":
					return runHostVolume(runCtx, ctx)
				}
				return fmt.Errorf("host command %q has no handler", command)
			}
		}(command)
	}
	return bindings
}

func runHostInstall(runCtx cliapp.RunContext, parent *rootcli.CommandContext) error {
	tool := strings.TrimSpace(runCtx.Positional("tool"))
	if tool == "" {
		return rootcli.UsageErrorf("host install", "a tool name is required")
	}
	sudoMode := strings.ToLower(strings.TrimSpace(runCtx.Flag("sudo-mode")))
	if sudoMode != "" && sudoMode != "ask" && sudoMode != "skip" && sudoMode != "error" {
		return rootcli.UsageErrorf("host install", "invalid --sudo-mode %q (want ask, skip, or error)", sudoMode)
	}
	jsonOutput := parent.Globals.JSON || runCtx.JSON()
	status, err := (hostapp.Service{}).Install(parent.OperationContext(), tool, hostapp.InstallOptions{DryRun: runCtx.BoolFlag("dry-run"), SudoMode: sudoMode})
	if err != nil {
		return fmt.Errorf("host install %q: %w", tool, err)
	}
	if jsonOutput {
		if err := cliout.WriteProtoJSON(runCtx.Stdout(), hostapp.HostInstallStatusResponse(status)); err != nil {
			return err
		}
	} else {
		hostapp.RenderHostInstallText(runCtx.Stdout(), status)
	}
	if !hostapp.HostInstallOK(status) {
		return rootcli.ExitCodeError{Code: 1, Silent_: true}
	}
	return nil
}

func runHostSafeguard(runCtx cliapp.RunContext, parent *rootcli.CommandContext) error {
	name := strings.TrimSpace(runCtx.Positional("name"))
	if name == "" {
		return rootcli.UsageErrorf("host safeguard", "a safeguard name is required")
	}
	sudoMode := strings.ToLower(strings.TrimSpace(runCtx.Flag("sudo-mode")))
	if sudoMode != "" && sudoMode != "ask" && sudoMode != "skip" && sudoMode != "error" {
		return rootcli.UsageErrorf("host safeguard", "invalid --sudo-mode %q (want ask, skip, or error)", sudoMode)
	}
	jsonOutput := parent.Globals.JSON || runCtx.JSON()
	value, err := (hostapp.Service{}).Safeguard(parent.OperationContext(), name, hostapp.SafeguardOptions{DryRun: runCtx.BoolFlag("dry-run"), MaintenanceWindow: runCtx.BoolFlag("maintenance-window"), SudoMode: sudoMode})
	if err != nil {
		return fmt.Errorf("host safeguard %q: %w", name, err)
	}
	if strings.EqualFold(name, "list") {
		return renderSafeguardListValue(runCtx.Stdout(), value, jsonOutput)
	}
	if strings.EqualFold(name, "portability-backlog") || strings.EqualFold(name, "portability_backlog") {
		return renderPortabilityBacklogValue(runCtx.Stdout(), value, jsonOutput)
	}
	if strings.EqualFold(name, "invariant-coverage") || strings.EqualFold(name, "invariant_coverage") {
		return renderInvariantCoverageValue(runCtx.Stdout(), value, jsonOutput)
	}
	if strings.EqualFold(name, "audit") || strings.EqualFold(name, "host-state-audit") {
		return renderHostStateAuditValue(runCtx.Stdout(), value, jsonOutput)
	}
	status, ok := value.(hostreqkit.ItemStatus)
	if !ok {
		return fmt.Errorf("host safeguard %q returned an unexpected result", name)
	}
	if jsonOutput {
		if err := cliout.WriteJSONValue(runCtx.Stdout(), status); err != nil {
			return err
		}
	} else {
		hostapp.RenderHostInstallText(runCtx.Stdout(), status)
	}
	if !hostapp.HostSafeguardOK(status) {
		return rootcli.ExitCodeError{Code: 1, Silent_: true}
	}
	return nil
}

func runHostStorage(runCtx cliapp.RunContext, parent *rootcli.CommandContext) error {
	action := ""
	allowUnknown := false
	for _, arg := range manifestdispatch.LegacyArgs(runCtx) {
		switch {
		case arg == "--allow-unknown-physical-device":
			allowUnknown = true
		case strings.HasPrefix(arg, "--"):
			if arg != "--json" {
				return rootcli.UsageErrorf("host storage", "unknown option %q", arg)
			}
		default:
			if action != "" {
				return rootcli.UsageErrorf("host storage", "unexpected argument %q", arg)
			}
			action = strings.ToLower(strings.TrimSpace(arg))
		}
	}
	if action != "candidates" {
		return rootcli.UsageErrorf("host storage", "the action must be candidates")
	}
	result, err := (hostapp.Service{}).DiscoverStorageCandidates(parent.OperationContext(), allowUnknown)
	if err != nil {
		return err
	}
	if parent.Globals.JSON || runCtx.JSON() {
		return cliout.WriteJSONValue(runCtx.Stdout(), result)
	}
	rows := make([][]string, 0, len(result.Candidates))
	for _, candidate := range result.Candidates {
		rows = append(rows, []string{candidate.Status, candidate.Kind, candidate.Location, candidate.Remediation})
	}
	return cliout.WriteSection(runCtx.Stdout(), cliout.Section{Rows: rows})
}

var hostVolumeActions = map[string]volumeremediation.Action{
	"inspect":  volumeremediation.ActionInspect,
	"check":    volumeremediation.ActionCheck,
	"repair":   volumeremediation.ActionRepair,
	"unmount":  volumeremediation.ActionUnmount,
	"mount-rw": volumeremediation.ActionMountReadWrite,
}

func runHostVolume(runCtx cliapp.RunContext, parent *rootcli.CommandContext) error {
	parsed := commandtree.NewFlagSet("host volume")
	device := parsed.String("device", "", "device")
	filesystem := parsed.String("filesystem", "", "filesystem")
	uuid := parsed.String("uuid", "", "uuid")
	serial := parsed.String("serial", "", "serial")
	mountpoint := parsed.String("mountpoint", "", "mountpoint")
	acknowledge := parsed.Bool("acknowledge-data-loss", false, "acknowledge data loss")
	dryRun := parsed.Bool("dry-run", false, "dry run")
	legacy := manifestdispatch.LegacyArgs(runCtx)
	if err := parsed.Parse(legacy); err != nil {
		return rootcli.UsageErrorf("host volume", "%s", err.Error())
	}
	if parsed.NArg() != 1 {
		return rootcli.UsageErrorf("host volume", "an action is required (inspect, check, repair, unmount, or mount-rw)")
	}
	action, ok := hostVolumeActions[strings.ToLower(strings.TrimSpace(parsed.Arg(0)))]
	if !ok {
		return rootcli.UsageErrorf("host volume", "unknown action %q (want inspect, check, repair, unmount, or mount-rw)", parsed.Arg(0))
	}
	if strings.TrimSpace(*device) == "" {
		return rootcli.UsageErrorf("host volume", "--device is required")
	}
	request := volumeremediation.Request{Action: action, Device: volumeremediation.Device{Path: strings.TrimSpace(*device), Filesystem: strings.TrimSpace(*filesystem), UUID: strings.TrimSpace(*uuid), Serial: strings.TrimSpace(*serial), Mountpoint: strings.TrimSpace(*mountpoint)}, DesiredMountpoint: strings.TrimSpace(*mountpoint), AcknowledgeDataLoss: *acknowledge, DryRun: *dryRun}
	result, execErr := (hostapp.Service{}).ExecuteVolume(parent.OperationContext(), request)
	response := hostapp.HostVolumeResponse(result, execErr)
	if parent.Globals.JSON || runCtx.JSON() {
		if err := cliout.WriteProtoJSON(runCtx.Stdout(), response); err != nil {
			return err
		}
	} else {
		hostapp.RenderHostVolumeText(runCtx.Stdout(), response)
	}
	if hostapp.HostVolumeOK(result.Status) {
		return nil
	}
	return rootcli.ExitCodeError{Code: 1, Silent_: true}
}

func renderSafeguardListValue(out io.Writer, value any, jsonOutput bool) error {
	entries, ok := value.([]hostapp.SafeguardListEntry)
	if !ok {
		return fmt.Errorf("safeguard list returned an unexpected result")
	}
	if jsonOutput {
		return cliout.WriteJSONValue(out, entries)
	}
	rows := make([][]string, 0, len(entries))
	for _, entry := range entries {
		rows = append(rows, []string{entry.Name, entry.Capability, entry.CapabilityRole, strings.Join(entry.Platforms, ","), entry.ObservedState})
	}
	return cliout.WriteSection(out, cliout.Section{Rows: rows})
}

func renderPortabilityBacklogValue(out io.Writer, value any, jsonOutput bool) error {
	report, ok := value.(hostruntime.PortabilityBacklogReport)
	if !ok {
		return fmt.Errorf("safeguard portability backlog returned an unexpected result")
	}
	if jsonOutput {
		return cliout.WriteJSONValue(out, report)
	}
	_, _ = fmt.Fprintf(out, "not_implemented safeguards: %d/%d\n", report.NotImplemented, report.Total)
	for _, entry := range report.Entries {
		_, _ = fmt.Fprintf(out, "- %s\n", entry)
	}
	return nil
}

func renderInvariantCoverageValue(out io.Writer, value any, jsonOutput bool) error {
	report, ok := value.(hostreqkit.Coverage)
	if !ok {
		return fmt.Errorf("safeguard invariant coverage returned an unexpected result")
	}
	if jsonOutput {
		return cliout.WriteJSONValue(out, report)
	}
	_, err := fmt.Fprintf(out, "sites walked: %d; invariants declared: %d; invariants evaluated: %d\n", report.SitesWalked, report.InvariantsDeclared, report.InvariantsEvaluated)
	for _, gap := range report.Gaps {
		if _, writeErr := fmt.Fprintf(out, "gap: %s\n", gap); writeErr != nil {
			return writeErr
		}
	}
	return err
}

func renderHostStateAuditValue(out io.Writer, value any, jsonOutput bool) error {
	report, ok := value.(hostinventory.AuditReport)
	if !ok {
		return fmt.Errorf("safeguard host-state audit returned an unexpected result")
	}
	if jsonOutput {
		return cliout.WriteJSONValue(out, report)
	}
	rows := make([][]string, 0, len(report.Findings))
	for _, finding := range report.Findings {
		rows = append(rows, []string{string(finding.Kind), finding.Path, finding.Reason})
	}
	return cliout.WriteSection(out, cliout.Section{Rows: rows})
}

func runHostCron(runCtx cliapp.RunContext, parent *rootcli.CommandContext) error {
	if action := runCtx.Positional("action"); action != "" && action != "audit" {
		return fmt.Errorf("unknown host cron action %q", action)
	}
	report, err := (hostapp.Service{}).AuditCron(parent.OperationContext(), parent.Root)
	if err != nil {
		return err
	}
	if parent.Globals.JSON || runCtx.JSON() {
		return cliout.WriteJSONValue(runCtx.Stdout(), report)
	}
	rows := make([][]string, 0, len(report.Findings)+1)
	rows = append(rows, []string{"status", report.Status, report.Detail})
	for _, finding := range report.Findings {
		rows = append(rows, []string{finding.Code, finding.Declaration, finding.Message})
	}
	return cliout.WriteSection(runCtx.Stdout(), cliout.Section{Rows: rows})
}

func runHostInventory(runCtx cliapp.RunContext, parent *rootcli.CommandContext) error {
	snapshot, err := (hostapp.Service{}).CollectInventory(parent.OperationContext())
	if err != nil {
		return err
	}
	if field := strings.TrimSpace(runCtx.Flag("field")); field != "" {
		value, err := (hostapp.Service{}).InventoryField(snapshot, field)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(runCtx.Stdout(), value)
		return err
	}
	if parent.Globals.JSON || runCtx.JSON() {
		return cliout.WriteProtoJSON(runCtx.Stdout(), hostapp.HostSnapshotResponse(snapshot))
	}
	_, _ = fmt.Fprintf(runCtx.Stdout(), "OS: %s/%s\n", snapshot.OS, snapshot.Arch)
	_, _ = fmt.Fprintf(runCtx.Stdout(), "CPU cores: %d\n", snapshot.CPU.Cores)
	_, _ = fmt.Fprintf(runCtx.Stdout(), "Memory total: %d MB\n", snapshot.Memory.TotalBytes/hostMemoryBytesPerMegabyte/hostMemoryBytesPerMegabyte)
	_, _ = fmt.Fprintf(runCtx.Stdout(), "NVIDIA GPU: %s\n", cliout.BoolLabel(snapshot.HasNvidiaGPU()))
	_, _ = fmt.Fprintf(runCtx.Stdout(), "Docker NVIDIA runtime: %s\n", cliout.BoolLabel(snapshot.HasDockerNvidiaRuntime()))
	return nil
}

func runDesktopSession(runCtx cliapp.RunContext, parent *rootcli.CommandContext) error {
	peerPID := 0
	if raw := runCtx.Flag("peer-pid"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 {
			return fmt.Errorf("peer-pid must be a non-negative integer")
		}
		peerPID = value
	}
	facts, err := (hostapp.Service{}).InspectDesktopSession(parent.OperationContext(), runCtx.Flag("session-id"), peerPID)
	if err != nil {
		return err
	}
	jsonOutput := parent.Globals.JSON || runCtx.JSON()
	if jsonOutput {
		return cliout.WriteProtoJSON(runCtx.Stdout(), &cliv1.CliDesktopSessionFacts{SessionId: facts.SessionID, Uid: facts.UID, PeerPid: int64(facts.PeerPID), PeerUid: facts.PeerUID, Type: facts.Type, Active: facts.Active, Locked: facts.Locked, Remote: facts.Remote, Matched: facts.Matched, Reason: facts.Reason, ObservedAt: facts.ObservedAt.UTC().Format(time.RFC3339Nano)})
	}
	_, err = fmt.Fprintf(runCtx.Stdout(), "Session %s: %s\n", facts.SessionID, facts.Reason)
	return err
}
