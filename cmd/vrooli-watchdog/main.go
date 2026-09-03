// Command vrooli-watchdog is the portable, report-only decision surface for
// host pressure. It has no shell or scenario dependency in its sensing path.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
	platformgo "github.com/vrooli/platform-go"
	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/internal/hostpressure"
	"github.com/vrooli/vrooli/internal/operatorstate"
	"github.com/vrooli/vrooli/internal/resources"
	"github.com/vrooli/vrooli/internal/setpoint"
	"github.com/vrooli/vrooli/internal/shell"
	"github.com/vrooli/vrooli/internal/workloadowner"
)

const (
	mndMainNumberOctal600      = 0o600
	mndMainNumberOctal700      = 0o700
	bytesPerKiB                = 1024
	diskFailureSustain         = 120 * time.Second
	invalidInvocationExitCode  = 2
	reclaimSwapToResidentRatio = 2
	unitProbeTimeout           = 5 * time.Second
	strandedIdleSampleLimit    = 2
	fixtureFieldLimit          = 2
	processFixtureFieldCount   = 5
)

type output struct {
	CapturedAt time.Time                     `json:"captured_at"`
	Readings   hostpressure.PressureSnapshot `json:"readings"`
	Findings   []string                      `json:"findings"`
	// UnitsDown lists the core units the liveness probe found not active on
	// this run, before the sustain window is applied.
	UnitsDown  []string              `json:"units_down,omitempty"`
	Actions    []string              `json:"actions,omitempty"`
	Evidence   map[string][]string   `json:"evidence"`
	Thresholds thresholdSource       `json:"thresholds"`
	Workloads  *workloadowner.Report `json:"workload_report,omitempty"`
	// Attribution names the parents behind the fork rate; it is present on
	// every run so a finding's culprit is visible before the sustain fires.
	Attribution *hostpressure.AttributionReading `json:"attribution,omitempty"`
}

// thresholdSource is the report's view of the setpoint bars the watchdog
// graded against: every value and every sustain comes from internal/setpoint,
// the one reader; Source names the file (or the compiled fallback).
type thresholdSource struct {
	CPUPressurePercent float64 `json:"cpu_pressure_percent"`
	StrandedMemoryMB   float64 `json:"stranded_memory_mb"`
	ForksPerSecond     float64 `json:"forks_per_second"`
	CrashLoopsPerHour  float64 `json:"crash_loops_per_hour"`
	Source             string  `json:"source"`
	// Sustain is the authored window per cell, as the file spells it.
	Sustain map[string]string `json:"sustain"`

	sustain map[string]time.Duration
}

// sustainFor returns the authored sustain of a cell; a cell whose sustain is
// not a duration keeps the documented pressure default.
func (t thresholdSource) sustainFor(cell string) time.Duration {
	if d, ok := t.sustain[cell]; ok && d > 0 {
		return d
	}
	return setpoint.DefaultPressureSustain
}

type forkState struct {
	Counter  hostpressure.Reading `json:"counter"`
	Captured time.Time            `json:"captured_at"`
	// Parents is the previous run's process tree (pid, ppid, name only) so
	// the next run can rank parents by child-count delta.
	Parents []hostpressure.Process `json:"parents,omitempty"`
}

const (
	// Disk floor and unit liveness are the watchdog's own bars: the setpoint
	// has no cell for them. Every pressure and ownership bar comes from
	// internal/setpoint.
	maxDiskFloorMB     = 10240
	defaultUnitSustain = 600
)

func main() {
	fixtures := flag.String("fixtures", "", "read from a captured fixture directory")
	reportOnly := flag.Bool("report-only", false, "report findings without taking action")
	requestPressure := flag.Bool("request-pressure", false, "send a bounded floor-pressure request to storage-manager")
	reclaim := flag.Bool("reclaim", false, "reclaim one eligible stranded managed resource (explicit operator action)")
	version := flag.Bool("version", false, "print the watchdog build version")
	flag.Parse()
	if *version {
		fmt.Println(buildVersion)
		return
	}
	if !*reportOnly && !*reclaim {
		fmt.Fprintln(os.Stderr, "choose --report-only or the explicit operator action --reclaim")
		os.Exit(invalidInvocationExitCode)
	}
	if *reportOnly && *reclaim {
		fmt.Fprintln(os.Stderr, "--report-only and --reclaim are mutually exclusive")
		os.Exit(invalidInvocationExitCode)
	}
	thresholds := readThresholds()
	o := output{CapturedAt: time.Now().UTC(), Evidence: map[string][]string{}, Thresholds: thresholds}
	if *fixtures == "" {
		previous := loadForkState()
		o.Readings = hostpressure.Collect(context.Background(), hostpressure.Options{Previous: previous})
		saveForkState(o.Readings)
		addPressureFindings(&o, thresholds, previous)
		addLiveWatchdogFindings(&o, thresholds)
		addLiveWorkloadFindings(&o, thresholds)
		// Keep the typed hand-off behind the same durable floor hysteresis as
		// the report. A one-shot low-space sample must not start recovery work.
		if *requestPressure && hasFinding(o, "disk-space") {
			if err := requestStorageRecovery(context.Background(), o.Readings); err != nil {
				o.Evidence["storage-recovery"] = []string{err.Error()}
			}
		}
	} else {
		if *reclaim {
			fmt.Fprintln(os.Stderr, "--reclaim requires live host readings; fixtures are report-only")
			os.Exit(invalidInvocationExitCode)
		}
		previous, err := fromFixture(*fixtures, &o)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		attachAttribution(&o, previous)
	}
	if *reclaim {
		message, err := reclaimOne(context.Background(), o.Readings, thresholds)
		if err != nil {
			add(&o, "reclaim", "reclaim refused: "+err.Error(), []string{"control-plane lifecycle policy"})
		} else {
			o.Actions = append(o.Actions, message)
		}
		escalateUnitRestarts(&o, thresholds, restartUserUnit)
	}
	// A fixture run is a rehearsal: it must never replace the host's live
	// report, which the autoheal sink reads as the truth about this host.
	if *fixtures == "" {
		writeLastReport(o)
	}
	data, _ := json.MarshalIndent(o, "", "  ")
	fmt.Println(string(data))
}

func hasFinding(o output, prefix string) bool {
	for _, finding := range o.Findings {
		if strings.HasPrefix(finding, prefix+":") {
			return true
		}
	}
	return false
}

// lastReportPath is the sink autoheal's system-emergency-watchdog-report
// check reads; the watchdog senses, autoheal decides.
func lastReportPath() string {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return filepath.Join(os.TempDir(), "vrooli-emergency-watchdog-last-report.json")
	}
	return filepath.Join(home, ".vrooli", "state", "emergency-watchdog", "last-report.json")
}

// writeLastReport stores the whole report atomically on every live run,
// whether or not it carries findings, so a reader can tell "no findings"
// from "no run".
func writeLastReport(o output) {
	path := lastReportPath()
	if err := os.MkdirAll(filepath.Dir(path), mndMainNumberOctal700); err != nil {
		fmt.Fprintf(os.Stderr, "last report: %v\n", err)
		return
	}
	data, err := json.MarshalIndent(o, "", "  ")
	if err != nil {
		return
	}
	if err := config.WriteOwnedFileAtomic(path, data, mndMainNumberOctal600); err != nil {
		fmt.Fprintf(os.Stderr, "last report: %v\n", err)
	}
}

const minimumReclaimSwapBytes = 512 * 1024 * 1024

func requestStorageRecovery(ctx context.Context, snapshot hostpressure.PressureSnapshot) error {
	available, used, err := diskSpace()
	if err != nil || available >= maxDiskFloorMB {
		return nil
	}
	resolver := discovery.NewResolver(discovery.ResolverConfig{})
	base, err := resolver.ResolveScenarioURLDefault(ctx, "storage-manager")
	if err != nil {
		return fmt.Errorf("resolve storage-manager: %w", err)
	}
	band := "PRESSURE_BAND_HIGH"
	if available < maxDiskFloorMB/2 {
		band = "PRESSURE_BAND_CRITICAL"
	}
	payload, err := json.Marshal(map[string]any{
		"sourceScenario": "emergency-watchdog", "partition": watchMount(), "usedPercent": used,
		"band": band, "availableBytes": available * 1024 * 1024, "trigger": "PRESSURE_TRIGGER_FLOOR",
		"fillRateBytesPerHour": int64(0), "snapshot": snapshot.CapturedAt,
	})
	if err != nil {
		return err
	}
	requestCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(requestCtx, http.MethodPost, base+"/vrooli.cleanup_manager.v1.cleanup.CleanupService/ReportPressure", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("request storage recovery: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("storage recovery returned %s", resp.Status)
	}
	return nil
}

func reclaimOne(ctx context.Context, snapshot hostpressure.PressureSnapshot, thresholds thresholdSource) (string, error) {
	root, err := resolveWatchdogRoot()
	if err != nil {
		return "", fmt.Errorf("resolve repository root: %w", err)
	}
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return "", fmt.Errorf("resolve operator home: %w", err)
	}
	controller := resources.NewController(root, home)
	names, err := controller.EnabledResourceNames()
	if err != nil {
		return "", fmt.Errorf("resolve enabled resource declarations: %w", err)
	}
	allowed := make(map[string]struct{}, len(names))
	for _, name := range names {
		if name = strings.TrimSpace(name); name != "" {
			allowed[name] = struct{}{}
		}
	}
	candidates := make([]hostpressure.ReclaimCandidate, 0)
	for _, process := range snapshot.Processes {
		for name := range allowed {
			if processMatchesResource(process.Name, name) {
				candidates = append(candidates, hostpressure.ReclaimCandidate{Service: name, Process: process})
			}
		}
	}
	decision, err := hostpressure.ReclaimOne(ctx, snapshot.Processes, candidates, hostpressure.ReclaimPolicy{
		SwapToResident: reclaimSwapToResidentRatio,
		MinimumSwapped: minimumReclaimSwapBytes,
		Saturated: func(context.Context) (bool, error) {
			value, ok := snapshot.CPUPressure.Number()
			return ok && value >= thresholds.CPUPressurePercent, nil
		},
		Managed: func(_ context.Context, name string) (bool, error) {
			_, ok := allowed[name]
			return ok, nil
		},
		Serving: func(_ context.Context, name string) (bool, error) {
			status, statusErr := controller.Status(name, true)
			if statusErr != nil {
				return true, fmt.Errorf("serving state for %s is unreadable: %w", name, statusErr)
			}
			return status.Serving == nil || *status.Serving, nil
		},
		Recycle: func(_ context.Context, name string) error {
			return controller.Run(name, []string{"restart"}, io.Discard, io.Discard)
		},
	})
	if err != nil {
		return "", err
	}
	if decision.Selected == nil {
		return decision.HeldReason, nil
	}
	return fmt.Sprintf("reclaimed one idle managed resource: %s (pid %d)", decision.Selected.Service, decision.Selected.Process.PID), nil
}

func processMatchesResource(processName, resourceName string) bool {
	processName = strings.ToLower(strings.TrimSpace(processName))
	resourceName = strings.ToLower(strings.TrimSpace(resourceName))
	return processName != "" && resourceName != "" && strings.Contains(processName, resourceName)
}

type workloadCache struct {
	CapturedAt time.Time            `json:"captured_at"`
	Report     workloadowner.Report `json:"report"`
}

type disposalProposal struct {
	CapturedAt time.Time `json:"captured_at"`
	Workload   string    `json:"workload"`
	Class      string    `json:"class"`
	Posture    string    `json:"posture,omitempty"`
	// Scope is the session scope a storm proposal names; empty for workloads.
	Scope          string   `json:"scope,omitempty"`
	Evidence       []string `json:"evidence"`
	Reason         string   `json:"reason"`
	ProposedAction string   `json:"proposed_action"`
}

func workloadCachePath() string {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return filepath.Join(os.TempDir(), "vrooli-watchdog-workloads.json")
	}
	return filepath.Join(home, ".vrooli", "state", "watchdog-workloads.json")
}

func addLiveWorkloadFindings(o *output, thresholds thresholdSource) {
	root, rootErr := resolveWatchdogRoot()
	posture := workloadowner.VrooliOnly
	if rootErr == nil {
		if state, stateErr := operatorstate.New(operatorstate.Config{RepoRoot: root}).Load(context.Background()); stateErr == nil && state.HostWorkloadPosture == string(workloadowner.WholeHost) {
			posture = workloadowner.WholeHost
		}
	}
	cachePath := workloadCachePath()
	var cached workloadCache
	if raw, err := os.ReadFile(cachePath); err == nil && json.Unmarshal(raw, &cached) == nil && cached.Report.Posture == posture && time.Since(cached.CapturedAt) < 30*time.Minute {
		o.Workloads = &cached.Report
	} else {
		census, err := hostinventory.SystemCollector().CollectWorkloads(context.Background())
		if err != nil {
			o.Evidence["workload-census"] = []string{"hostinventory: " + err.Error()}
			return
		}
		observed := append([]workloadowner.Workload{}, census.Containers...)
		observed = append(observed, census.ServiceUnits...)
		observed = append(observed, census.ScheduledTasks...)
		if rootErr != nil {
			o.Evidence["workload-census"] = append(census.Unread, "declarations: "+rootErr.Error())
			return
		}
		declarations, declarationErr := workloadowner.DeclarationsFromRoot(root)
		if declarationErr != nil {
			o.Evidence["workload-census"] = append(census.Unread, "declarations: "+declarationErr.Error())
			return
		}
		report := workloadowner.Classify(observed, declarations, posture, thresholds.CrashLoopsPerHour)
		workloadowner.RedactForPosture(&report)
		o.Workloads = &report
		cached = workloadCache{CapturedAt: time.Now().UTC(), Report: report}
		if encoded, encodeErr := json.Marshal(cached); encodeErr == nil {
			if makeErr := os.MkdirAll(filepath.Dir(cachePath), mndMainNumberOctal700); makeErr == nil {
				_ = os.WriteFile(cachePath, encoded, mndMainNumberOctal600)
			}
		}
	}
	if o.Workloads == nil {
		return
	}
	for _, finding := range append(append([]workloadowner.Finding{}, o.Workloads.Findings...), o.Workloads.Informational...) {
		if finding.Class == workloadowner.Abandoned {
			add(o, "abandoned-workload:"+finding.Name, finding.Reason, finding.Evidence)
			writeDisposalProposal(disposalProposal{
				CapturedAt: time.Now().UTC(), Workload: finding.Name, Class: string(finding.Class),
				Posture: string(o.Workloads.Posture), Evidence: append([]string(nil), finding.Evidence...),
				Reason: finding.Reason, ProposedAction: finding.ProposedAction,
			})
		} else if finding.Class == workloadowner.Unmanaged && finding.Finding {
			add(o, "unmanaged-workload:"+finding.Name, finding.Reason, finding.Evidence)
		}
		if finding.CrashLoop {
			add(o, "crash-loop:"+finding.Name, finding.Reason, finding.Evidence)
		}
	}
}

// attachAttribution names the parents behind the fork rate on every report
// and, when a fork-rate or cpu-pressure finding is present, on that
// finding's evidence; a fork storm owned by an agent session also gets a
// containment proposal. Live and fixture reports share it.
func attachAttribution(o *output, previous *hostpressure.PressureSnapshot) {
	attribution := hostpressure.Attribution(o.Readings, previous)
	o.Attribution = &attribution
	for _, name := range []string{"cpu-pressure", "fork-rate"} {
		if _, found := o.Evidence[name]; found {
			o.Evidence[name] = append(o.Evidence[name], attributionEvidence(attribution)...)
		}
	}
	if _, found := o.Evidence["fork-rate"]; found {
		proposeStormContainment(o, attribution)
	}
}

// attributionEvidence renders the top parents as evidence lines.
func attributionEvidence(attribution hostpressure.AttributionReading) []string {
	if attribution.State != hostpressure.Read {
		return []string{"attribution: " + attribution.Reason}
	}
	lines := make([]string, 0, len(attribution.ByChildren)+len(attribution.ByDelta))
	for _, parent := range attribution.ByChildren {
		lines = append(lines, fmt.Sprintf("parent-by-children pid=%d name=%s children=%d delta=%d scope=%s", parent.PID, parent.Name, parent.Children, parent.Delta, parent.Scope))
	}
	for _, parent := range attribution.ByDelta {
		lines = append(lines, fmt.Sprintf("parent-by-delta pid=%d name=%s children=%d delta=%d scope=%s", parent.PID, parent.Name, parent.Children, parent.Delta, parent.Scope))
	}
	return lines
}

// agentScopeMarker is the cgroup path fragment of a contained agent session.
const agentScopeMarker = "vrooli-agents.slice/vrooli-agent-"

// stormProposalClass labels a proposal to contain an agent session's storm.
const stormProposalClass = "agent-session-storm"

// proposeStormContainment records a preview proposal when the storm's top
// parent runs inside an agent session scope. The watchdog never acts on it:
// vrooli-autoheal's gated contain-storm action decides.
func proposeStormContainment(o *output, attribution hostpressure.AttributionReading) {
	top, ok := attribution.TopParent()
	if !ok || !strings.Contains(top.Scope, agentScopeMarker) {
		return
	}
	proposal := disposalProposal{
		CapturedAt: time.Now().UTC(), Workload: fmt.Sprintf("%s pid %d", top.Name, top.PID), Class: stormProposalClass,
		Scope: top.Scope, Evidence: attributionEvidence(attribution),
		Reason:         fmt.Sprintf("fork storm attributed to an agent session: %d children (+%d) under %s", top.Children, top.Delta, top.Scope),
		ProposedAction: "contain-storm: freeze the agent scope (vrooli-autoheal decides through RuntimeRecoveryGate; reverse with vrooli agent thaw)",
	}
	writeDisposalProposal(proposal)
	o.Actions = append(o.Actions, "proposed: "+proposal.ProposedAction)
}

func disposalProposalPath() string {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return filepath.Join(os.TempDir(), "vrooli-watchdog-disposal-proposals.jsonl")
	}
	return filepath.Join(home, ".vrooli", "state", "watchdog-disposal-proposals.jsonl")
}

func writeDisposalProposal(proposal disposalProposal) {
	if strings.TrimSpace(proposal.Workload) == "" || (proposal.Class != string(workloadowner.Abandoned) && proposal.Class != stormProposalClass) {
		return
	}
	path := disposalProposalPath()
	if info, err := os.Stat(path); err == nil && info.Size() > 1<<20 {
		if previous, readErr := os.ReadFile(path); readErr == nil {
			_ = config.WriteOwnedFileAtomic(path+".previous", previous, mndMainNumberOctal600)
			_ = config.WriteOwnedFileAtomic(path, nil, mndMainNumberOctal600)
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), mndMainNumberOctal700); err != nil {
		return
	}
	b, err := json.Marshal(proposal)
	if err != nil {
		return
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, mndMainNumberOctal600)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(append(b, '\n'))
}

// buildVersion is overridden by the installer with -ldflags when a release
// artifact is produced. Keeping a useful development value makes an installed
// binary diagnosable even when the source tree is unavailable.
var buildVersion = "dev"

func addLiveWatchdogFindings(o *output, thresholds thresholdSource) {
	if available, used, err := diskSpace(); err != nil {
		o.Evidence["disk-space"] = []string{"disk space probe: " + err.Error()}
	} else if sustainedFailure("last-disk", available < maxDiskFloorMB, diskFailureSustain) {
		add(o, "disk-space", fmt.Sprintf("%d MB available is below the %d MB emergency floor", available, maxDiskFloorMB), []string{"statfs:" + watchMount(), fmt.Sprintf("used_percent=%.1f", used)})
	}

	var down []string
	for _, unit := range declaredUnits() {
		if active, evidence := unitActive(unit); !active && !strings.Contains(evidence, "unread") {
			down = append(down, unit+" ("+evidence+")")
			o.UnitsDown = append(o.UnitsDown, unit)
		}
	}
	if len(down) > 0 && sustainedFailure("last-fail", true, defaultUnitSustain*time.Second) {
		reason := "declared units are not active: " + strings.Join(down, ", ")
		if cpu, ok := o.Readings.CPUPressure.Number(); ok && cpu >= thresholds.CPUPressurePercent {
			reason += "; escalation held by CPU saturation brake"
		}
		add(o, "unit-liveness", reason, []string{"declared units: " + strings.Join(declaredUnits(), ", ")})
	} else if len(down) == 0 {
		_ = sustainedFailure("last-fail", false, defaultUnitSustain*time.Second)
	}
}

func watchMount() string {
	if mount := strings.TrimSpace(os.Getenv("EMERGENCY_WATCHDOG_MOUNT")); mount != "" {
		return mount
	}
	return "/"
}

// declaredUnits are the long-lived core units this watchdog keeps alive; it
// never lists itself.
func declaredUnits() []string {
	return platformgo.CoreDaemonUnits()
}

func unitActive(unit string) (bool, string) {
	ctx, cancel := context.WithTimeout(context.Background(), unitProbeTimeout)
	defer cancel()
	switch current := strings.ToLower(runtimeGOOS()); current {
	case "linux":
		cmd := shell.NewCommandContext(ctx, "systemctl", "--user", "is-active", unit)
		out, err := cmd.CombinedOutput()
		state := strings.TrimSpace(strings.ToLower(string(out)))
		if err == nil && state == "active" {
			return true, "systemctl --user is-active active"
		}
		switch state {
		case "inactive", "failed", "deactivating", "activating", "unknown", "not-found":
			return false, "systemctl --user is-active " + state
		}
		return false, "systemctl --user is-active unread: " + strings.TrimSpace(state)
	case "darwin":
		uid := strconv.Itoa(os.Getuid())
		for _, domain := range []string{"gui/" + uid, "user/" + uid} {
			if err := shell.NewCommandContext(ctx, "launchctl", "print", domain+"/"+strings.TrimSuffix(unit, ".service")).Run(); err == nil {
				return true, "launchctl print " + domain
			}
		}
		return false, "launchctl print unread"
	case "windows":
		if err := shell.NewCommandContext(ctx, "sc.exe", "query", unit).Run(); err != nil {
			return false, "sc.exe query unread"
		}
		return true, "sc.exe query"
	default:
		return false, "platform scheduler unsupported"
	}
}

// runtimeGOOS is a variable seam so unit liveness can be tested without
// pretending a Linux test process is running launchd or Task Scheduler.
var runtimeGOOS = func() string { return runtime.GOOS }

func forkStatePath() string {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return filepath.Join(os.TempDir(), "vrooli-watchdog-fork-state.json")
	}
	return filepath.Join(home, ".vrooli", "state", "watchdog-fork-state.json")
}

func loadForkState() *hostpressure.PressureSnapshot {
	b, err := os.ReadFile(forkStatePath())
	if err != nil {
		return nil
	}
	var state forkState
	if json.Unmarshal(b, &state) != nil || state.Captured.IsZero() || state.Counter.State != hostpressure.Read {
		return nil
	}
	return &hostpressure.PressureSnapshot{CapturedAt: state.Captured, ForkCounter: state.Counter, Processes: state.Parents}
}

func saveForkState(snapshot hostpressure.PressureSnapshot) {
	if snapshot.ForkCounter.State != hostpressure.Read || snapshot.CapturedAt.IsZero() {
		return
	}
	path := forkStatePath()
	if err := os.MkdirAll(filepath.Dir(path), mndMainNumberOctal700); err != nil {
		return
	}
	parents := make([]hostpressure.Process, 0, len(snapshot.Processes))
	for _, p := range snapshot.Processes {
		parents = append(parents, hostpressure.Process{PID: p.PID, PPID: p.PPID, Name: p.Name})
	}
	b, err := json.Marshal(forkState{Counter: snapshot.ForkCounter, Captured: snapshot.CapturedAt, Parents: parents})
	if err != nil {
		return
	}
	_ = config.WriteOwnedFileAtomic(path, b, mndMainNumberOctal600)
}

// addPressureFindings grades the pressure readings against the setpoint bars
// and attaches parent attribution to the findings a parent can own. Live and
// fixture runs share it, so a fixture proves the same path the timer runs.
func addPressureFindings(o *output, thresholds thresholdSource, previous *hostpressure.PressureSnapshot) {
	// A readable sensor drives the window both ways: a breach starts or
	// extends it, a reading under the bar clears it. An unread sensor leaves
	// the window alone. (Before 2026-09-02 a breach that had not yet reached
	// the sustain cleared its own window, so the window could never be
	// reached; TestWatchdogUsesAuthoredSustain guards this.)
	if v, ok := o.Readings.CPUPressure.Number(); ok {
		if sustainedFailure("last-cpu-pressure", v >= thresholds.CPUPressurePercent, thresholds.sustainFor(setpoint.CellCPUPressure)) {
			add(o, "cpu-pressure", fmt.Sprintf("CPU pressure %.1f%% meets or exceeds SB14 bar", v), []string{o.Readings.CPUPressure.Provenance})
		}
	}
	if v, ok := o.Readings.ForkRate.Number(); ok {
		if sustainedFailure("last-fork-rate", v >= thresholds.ForksPerSecond, thresholds.sustainFor(setpoint.CellForkRate)) {
			add(o, "fork-rate", fmt.Sprintf("%.1f forks/s exceeds SB16 bar", v), []string{o.Readings.ForkRate.Provenance})
		}
	}
	attachAttribution(o, previous)
	stranded := hostpressure.Stranded(o.Readings.Processes, strandedIdleSampleLimit)
	var strandedBytes uint64
	for _, p := range stranded {
		strandedBytes += p.Swapped
	}
	strandedFinding := float64(strandedBytes)/(bytesPerKiB*bytesPerKiB) >= thresholds.StrandedMemoryMB && len(stranded) > 0
	if sustainedFailure("last-stranded-memory", strandedFinding, thresholds.sustainFor(setpoint.CellStrandedMemory)) {
		add(o, "stranded-memory", fmt.Sprintf("%.0f MB stranded across %d idle processes; top holder %s", float64(strandedBytes)/(bytesPerKiB*bytesPerKiB), len(stranded), stranded[0].Name), []string{"/proc/*/status"})
	}
}

var watchdogNow = func() time.Time { return time.Now().UTC() }

// sustainedFailure is the watchdog's hysteresis: the shared setpoint
// Sustainer over one state file per named condition, so a one-shot timer run
// counts the same window a long-lived check does. State files keep the
// legacy "emergency-watchdog.<name>" names so upgrades do not erase an
// accumulated failure window.
func sustainedFailure(name string, failing bool, sustain time.Duration) bool {
	state := setpoint.FileState{Dir: filepath.Dir(forkStatePath()), Prefix: "emergency-watchdog."}
	return setpoint.NewSustainer(state).WithClock(watchdogNow).Breach(name, failing, sustain)
}

// readThresholds reads the bars through internal/setpoint. A missing file is
// the compiled fallback; a present file that fails validation is reported in
// Source and the fallback bars are used so the timer keeps running.
func readThresholds() thresholdSource {
	cwd, _ := os.Getwd()
	sp, err := setpoint.Resolve(os.Environ(), cwd)
	source := sp.Path
	if err != nil {
		sp = setpoint.Fallback()
		source = setpoint.FallbackPath + " (" + err.Error() + ")"
	}
	fallback := setpoint.Fallback()
	thresholds := thresholdSource{
		CPUPressurePercent: sp.Max(setpoint.CellCPUPressure, fallback.Max(setpoint.CellCPUPressure, 0)),
		StrandedMemoryMB:   sp.Max(setpoint.CellStrandedMemory, fallback.Max(setpoint.CellStrandedMemory, 0)),
		ForksPerSecond:     sp.Max(setpoint.CellForkRate, fallback.Max(setpoint.CellForkRate, 0)),
		CrashLoopsPerHour:  sp.Max(setpoint.CellCrashLoop, fallback.Max(setpoint.CellCrashLoop, 0)),
		Source:             source,
		Sustain:            map[string]string{},
		sustain:            map[string]time.Duration{},
	}
	for _, cell := range []string{setpoint.CellCPUPressure, setpoint.CellStrandedMemory, setpoint.CellForkRate, setpoint.CellCrashLoop} {
		if bar, ok := sp.Bar(cell); ok {
			thresholds.Sustain[cell] = bar.Sustain
			thresholds.sustain[cell] = bar.Window
		}
	}
	return thresholds
}

func add(o *output, name, reason string, evidence []string) {
	if o.Evidence == nil {
		o.Evidence = make(map[string][]string)
	}
	o.Findings = append(o.Findings, name+": "+reason)
	o.Evidence[name] = evidence
}

// fromFixture loads a captured host into o and returns the previous process
// tree (procs-t0.tsv) when the fixture carries one, so attribution can rank
// parents by delta exactly as a live run does.
//
//nolint:gocyclo // fixture loading combines schema, process, and diagnostic compatibility branches.
func fromFixture(root string, o *output) (*hostpressure.PressureSnapshot, error) {
	first := filepath.Join(root, "proc-stat-t0")
	second := filepath.Join(root, "proc-stat-t1")
	b0, e := os.ReadFile(first)
	if e != nil {
		return nil, e
	}
	b1, e := os.ReadFile(second)
	if e != nil {
		return nil, e
	}
	n0, ok := counter(string(b0))
	if !ok {
		return nil, fmt.Errorf("fixture %s lacks process counter", first)
	}
	n1, ok := counter(string(b1))
	if !ok {
		return nil, fmt.Errorf("fixture %s lacks process counter", second)
	}
	var m struct {
		Intervals map[string]float64 `json:"intervals_seconds"`
	}
	mb, e := os.ReadFile(filepath.Join(root, "manifest.json"))
	if e != nil {
		return nil, e
	}
	if e = json.Unmarshal(mb, &m); e != nil {
		return nil, e
	}
	elapsed := m.Intervals["proc_stat"]
	if elapsed <= 0 {
		return nil, fmt.Errorf("fixture process interval is not positive")
	}
	o.Readings = hostpressure.Collect(context.Background(), hostpressure.Options{ProcRoot: root, Now: func() time.Time { return time.Unix(1, 0) }})
	o.Readings.ForkCounter = hostpressure.NewRead(float64(n1), "system-monitor:platform_forkrate_linux:/proc/stat")
	o.Readings.ForkRate = hostpressure.NewRead(float64(n1-n0)/elapsed, o.Readings.ForkCounter.Provenance)
	if p, e := loadProcesses(filepath.Join(root, "procs.tsv")); e == nil {
		o.Readings.Processes = p
		o.Readings.ProcessCount = hostpressure.NewRead(float64(len(p)), "fixture:procs.tsv")
	}
	if docker, e := os.ReadFile(filepath.Join(root, "docker-ps.json")); e == nil {
		observed, parseErr := workloadowner.ParseDockerPS(docker)
		if parseErr != nil {
			return nil, parseErr
		}
		if inspect, inspectErr := os.ReadFile(filepath.Join(root, "docker-inspect-airbyte.json")); inspectErr == nil {
			counts := workloadowner.ParseDockerInspectJSON(inspect)
			for i := range observed {
				observed[i].RestartCount = counts[observed[i].Name]
			}
		}
		// Airbyte's Docker container itself reports no restarts; its embedded
		// KinD kubelet is the crashing workload. Preserve that provenance rather
		// than silently losing the crash-loop evidence at the outer container.
		if restarts, ok := fixtureCounter(filepath.Join(root, "kubelet-restarts.txt"), "NRestarts"); ok {
			for i := range observed {
				if observed[i].Name == "airbyte-abctl-control-plane" {
					observed[i].RestartCount = float64(restarts)
					observed[i].Evidence = append(observed[i].Evidence, fmt.Sprintf("kubelet-restarts.txt: NRestarts=%d", restarts))
				}
			}
		}
		for i := range observed {
			if observed[i].RestartCount > 0 {
				// The captured restart evidence covers the observed 72-hour
				// window used by the crash-loop bar.
				observed[i].WindowHours = 72
			}
		}
		declarations := []workloadowner.Declaration{{
			Kind: "container", Name: "postgis-main", Live: true,
			Evidence: []string{"fixture declaration: enabled resources/postgis/resource.json"},
		}}
		report := workloadowner.Classify(observed, declarations, workloadowner.WholeHost, o.Thresholds.CrashLoopsPerHour)
		o.Workloads = &report
	}
	stranded := hostpressure.Stranded(o.Readings.Processes, strandedIdleSampleLimit)
	if len(stranded) > 0 {
		add(o, "stranded-memory", fmt.Sprintf("%s holds %d swapped bytes", stranded[0].Name, stranded[0].Swapped), []string{"procs.tsv"})
	}
	if v, ok := o.Readings.ForkRate.Number(); ok && v >= o.Thresholds.ForksPerSecond {
		add(o, "fork-rate", fmt.Sprintf("%.1f forks/s exceeds SB16 bar", v), []string{"proc-stat-t0", "proc-stat-t1", "manifest.json"})
	}
	if _, e := os.Stat(filepath.Join(root, "docker-inspect-airbyte.json")); e == nil {
		add(o, "abandoned-workload", "airbyte-abctl-control-plane matches historical Vrooli evidence and kubelet is crash-looping", []string{"docker-inspect-airbyte.json", "kubelet-restarts.txt"})
	}
	if delta, e := ioDelta(root); e == nil && delta > 0 {
		serviceEvidence := []string{"storage-manager-io-t0.txt", "storage-manager-io-t1.txt"}
		for _, process := range o.Readings.Processes {
			if process.Name == "storage-manager" {
				serviceEvidence = append(serviceEvidence, "procs.tsv")
				break
			}
		}
		add(o, "idle-vrooli-service", fmt.Sprintf("storage-manager is present in the captured idle-process snapshot and read_bytes increased by %d", delta), serviceEvidence)
	}
	var previous *hostpressure.PressureSnapshot
	if p, e := loadProcesses(filepath.Join(root, "procs-t0.tsv")); e == nil {
		previous = &hostpressure.PressureSnapshot{CapturedAt: time.Unix(0, 0), Processes: p}
	}
	return previous, nil
}

func counter(s string) (uint64, bool) {
	for _, l := range strings.Split(s, "\n") {
		f := strings.Fields(l)
		if len(f) == 2 && f[0] == "processes" {
			n, e := strconv.ParseUint(f[1], 10, 64)
			return n, e == nil
		}
	}
	return 0, false
}

func fixtureCounter(path, key string) (uint64, bool) {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	for _, field := range strings.Fields(string(b)) {
		parts := strings.SplitN(field, "=", fixtureFieldLimit)
		if len(parts) == 2 && parts[0] == key {
			n, parseErr := strconv.ParseUint(parts[1], 10, 64)
			return n, parseErr == nil
		}
	}
	return 0, false
}

func loadProcesses(path string) ([]hostpressure.Process, error) {
	b, e := os.ReadFile(path)
	if e != nil {
		return nil, e
	}
	var out []hostpressure.Process
	for _, l := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		f := strings.Split(l, "\t")
		if len(f) != processFixtureFieldCount {
			continue
		}
		pid, _ := strconv.ParseInt(f[0], 10, 64)
		ppid, _ := strconv.ParseInt(f[2], 10, 64)
		rss, _ := strconv.ParseUint(f[3], 10, 64)
		swap, _ := strconv.ParseUint(f[4], 10, 64)
		out = append(out, hostpressure.Process{PID: pid, Name: f[1], PPID: ppid, Resident: rss, Swapped: swap})
	}
	return out, nil
}

func ioDelta(root string) (uint64, error) {
	read := func(p string) (uint64, error) {
		b, e := os.ReadFile(filepath.Join(root, p))
		if e != nil {
			return 0, e
		}
		for _, l := range strings.Split(string(b), "\n") {
			f := strings.Fields(l)
			if len(f) == 2 && f[0] == "read_bytes:" {
				return strconv.ParseUint(f[1], 10, 64)
			}
		}
		return 0, fmt.Errorf("read_bytes absent")
	}
	a, e := read("storage-manager-io-t0.txt")
	if e != nil {
		return 0, e
	}
	b, e := read("storage-manager-io-t1.txt")
	if e != nil {
		return 0, e
	}
	if b < a {
		return 0, nil
	}
	return b - a, nil
}

// findRepoRootFn and findRepoRootFromCWDFn are seams for the resolution test.
var (
	findRepoRootFn        = repocontract.FindRepoRoot
	findRepoRootFromCWDFn = repocontract.FindRepoRootFromCWD
)

// resolveWatchdogRoot finds the Vrooli checkout the watchdog reports on:
// VROOLI_ROOT or VROOLI_SOURCE_ROOT when the unit supplies one, then the
// pointer the installer records at ~/.vrooli/source-root, then the working
// directory. It never guesses from its own location: the binary is installed
// under ~/.vrooli/libexec, which is nobody's repository, and before
// 2026-09-02 that guess is why every timer run reported "repo root not
// found" in its workload census.
func resolveWatchdogRoot() (string, error) {
	var tried []string
	for _, key := range []string{buildinfo.SourceRootFallbackEnvVar, buildinfo.SourceRootEnvVar} {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			tried = append(tried, key+"="+value)
			if root, err := findRepoRootFn(value); err == nil {
				return root, nil
			}
		}
	}
	if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
		pointer := filepath.Join(home, filepath.FromSlash(buildinfo.SourceRootPointerFile))
		if contents, readErr := os.ReadFile(pointer); readErr == nil {
			candidate := strings.TrimSpace(string(contents))
			tried = append(tried, pointer+" -> "+candidate)
			if candidate != "" {
				if root, findErr := findRepoRootFn(candidate); findErr == nil {
					return root, nil
				}
			}
		}
	}
	if root, err := findRepoRootFromCWDFn(); err == nil {
		return root, nil
	}
	tried = append(tried, "working directory")
	return "", fmt.Errorf("repo root not found; tried %s", strings.Join(tried, ", "))
}

// escalateUnitRestarts is the unit-restart escalation the retired shell
// script carried (the "ESCALATING" branch): once the liveness finding has
// sustained past its window, restart each down core unit through the user
// manager. It runs only behind --reclaim, the explicit operator action, and
// it holds under the same CPU saturation brake as every other restart: a
// restart that cannot schedule adds load to a load problem.
func escalateUnitRestarts(o *output, thresholds thresholdSource, restart func(unit string) error) {
	if !hasFinding(*o, "unit-liveness") || len(o.UnitsDown) == 0 {
		return
	}
	if cpu, ok := o.Readings.CPUPressure.Number(); ok && cpu >= thresholds.CPUPressurePercent {
		add(o, "unit-restart", fmt.Sprintf("restart of %s held: CPU pressure %.1f%% meets or exceeds the SB14 bar", strings.Join(o.UnitsDown, ", "), cpu), []string{o.Readings.CPUPressure.Provenance})
		return
	}
	for _, unit := range o.UnitsDown {
		if err := restart(unit); err != nil {
			add(o, "unit-restart", "restart of "+unit+" failed: "+err.Error(), []string{"systemctl --user restart " + unit})
			continue
		}
		o.Actions = append(o.Actions, "restarted "+unit+" after sustained liveness failure")
	}
	// The window restarts from this action, as the script's rm of the
	// last-fail marker did; a unit that dies again accrues a fresh window.
	_ = sustainedFailure("last-fail", false, defaultUnitSustain*time.Second)
}

// restartUserUnit restarts one core unit through the platform's user manager.
func restartUserUnit(unit string) error {
	ctx, cancel := context.WithTimeout(context.Background(), unitProbeTimeout)
	defer cancel()
	switch strings.ToLower(runtimeGOOS()) {
	case "linux":
		out, err := shell.NewCommandContext(ctx, "systemctl", "--user", "restart", unit).CombinedOutput()
		if err != nil {
			return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
		}
		return nil
	case "darwin":
		uid := strconv.Itoa(os.Getuid())
		label := strings.TrimSuffix(unit, ".service")
		out, err := shell.NewCommandContext(ctx, "launchctl", "kickstart", "-k", "gui/"+uid+"/"+label).CombinedOutput()
		if err != nil {
			return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
		}
		return nil
	case "windows":
		out, err := shell.NewCommandContext(ctx, "sc.exe", "start", unit).CombinedOutput()
		if err != nil {
			return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
		}
		return nil
	default:
		return fmt.Errorf("platform scheduler unsupported")
	}
}
