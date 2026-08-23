// Command vrooli-watchdog is the portable, report-only decision surface for
// host pressure. It has no shell or scenario dependency in its sensing path.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/internal/hostpressure"
	"github.com/vrooli/vrooli/internal/operatorstate"
	"github.com/vrooli/vrooli/internal/resources"
	"github.com/vrooli/vrooli/internal/workloadowner"
)

type output struct {
	CapturedAt time.Time                     `json:"captured_at"`
	Readings   hostpressure.PressureSnapshot `json:"readings"`
	Findings   []string                      `json:"findings"`
	Actions    []string                      `json:"actions,omitempty"`
	Evidence   map[string][]string           `json:"evidence"`
	Thresholds thresholdSource               `json:"thresholds"`
	Workloads  *workloadowner.Report         `json:"workload_report,omitempty"`
}

type thresholdSource struct {
	CPUPressurePercent float64 `json:"cpu_pressure_percent"`
	StrandedMemoryMB   float64 `json:"stranded_memory_mb"`
	ForksPerSecond     float64 `json:"forks_per_second"`
	CrashLoopsPerHour  float64 `json:"crash_loops_per_hour"`
	Source             string  `json:"source"`
}

type forkState struct {
	Counter  hostpressure.Reading `json:"counter"`
	Captured time.Time            `json:"captured_at"`
}

const (
	// These are compatibility fallbacks for the legacy emergency watchdog
	// behavior. Pressure and ownership bars come from the setpoint below.
	maxDiskFloorMB        = 10240
	defaultUnitSustain    = 600
	maxCPUPressurePercent = 50.0
	maxStrandedMemoryMB   = 17200.0
	maxForksPerSecond     = 200.0
	maxCrashLoopsPerHour  = 2700.0
)

func main() {
	fixtures := flag.String("fixtures", "", "read from a captured fixture directory")
	reportOnly := flag.Bool("report-only", false, "report findings without taking action")
	reclaim := flag.Bool("reclaim", false, "reclaim one eligible stranded managed resource (explicit operator action)")
	version := flag.Bool("version", false, "print the watchdog build version")
	flag.Parse()
	if *version {
		fmt.Println(buildVersion)
		return
	}
	if !*reportOnly && !*reclaim {
		fmt.Fprintln(os.Stderr, "choose --report-only or the explicit operator action --reclaim")
		os.Exit(2)
	}
	if *reportOnly && *reclaim {
		fmt.Fprintln(os.Stderr, "--report-only and --reclaim are mutually exclusive")
		os.Exit(2)
	}
	thresholds := readThresholds()
	o := output{CapturedAt: time.Now().UTC(), Evidence: map[string][]string{}, Thresholds: thresholds}
	if *fixtures == "" {
		previous := loadForkState()
		o.Readings = hostpressure.Collect(context.Background(), hostpressure.Options{Previous: previous})
		saveForkState(o.Readings)
		addLiveFindings(&o, thresholds)
		addLiveWatchdogFindings(&o, thresholds)
		addLiveWorkloadFindings(&o, thresholds)
	} else {
		if *reclaim {
			fmt.Fprintln(os.Stderr, "--reclaim requires live host readings; fixtures are report-only")
			os.Exit(2)
		}
		if err := fromFixture(*fixtures, &o); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	if *reclaim {
		message, err := reclaimOne(context.Background(), o.Readings, thresholds)
		if err != nil {
			add(&o, "reclaim", "reclaim refused: "+err.Error(), []string{"control-plane lifecycle policy"})
		} else {
			o.Actions = append(o.Actions, message)
		}
	}
	data, _ := json.MarshalIndent(o, "", "  ")
	fmt.Println(string(data))
}

const minimumReclaimSwapBytes = 512 * 1024 * 1024

func reclaimOne(ctx context.Context, snapshot hostpressure.PressureSnapshot, thresholds thresholdSource) (string, error) {
	root, err := repocontract.ResolveRepoRoot()
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
		SwapToResident: 2,
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
	CapturedAt     time.Time `json:"captured_at"`
	Workload       string    `json:"workload"`
	Class          string    `json:"class"`
	Posture        string    `json:"posture"`
	Evidence       []string  `json:"evidence"`
	Reason         string    `json:"reason"`
	ProposedAction string    `json:"proposed_action"`
}

func workloadCachePath() string {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return filepath.Join(os.TempDir(), "vrooli-watchdog-workloads.json")
	}
	return filepath.Join(home, ".vrooli", "state", "watchdog-workloads.json")
}

func addLiveWorkloadFindings(o *output, thresholds thresholdSource) {
	root, rootErr := repocontract.ResolveRepoRoot()
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
			if makeErr := os.MkdirAll(filepath.Dir(cachePath), 0o700); makeErr == nil {
				_ = os.WriteFile(cachePath, encoded, 0o600)
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

func disposalProposalPath() string {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return filepath.Join(os.TempDir(), "vrooli-watchdog-disposal-proposals.jsonl")
	}
	return filepath.Join(home, ".vrooli", "state", "watchdog-disposal-proposals.jsonl")
}

func writeDisposalProposal(proposal disposalProposal) {
	if strings.TrimSpace(proposal.Workload) == "" || proposal.Class != string(workloadowner.Abandoned) {
		return
	}
	path := disposalProposalPath()
	if info, err := os.Stat(path); err == nil && info.Size() > 1<<20 {
		_ = os.Rename(path, path+".previous")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return
	}
	b, err := json.Marshal(proposal)
	if err != nil {
		return
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
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
	} else if sustainedFailure("last-disk", available < maxDiskFloorMB, 120*time.Second) {
		add(o, "disk-space", fmt.Sprintf("%d MB available is below the %d MB emergency floor", available, maxDiskFloorMB), []string{"statfs:" + watchMount(), fmt.Sprintf("used_percent=%.1f", used)})
	}

	var down []string
	for _, unit := range declaredUnits() {
		if active, evidence := unitActive(unit); !active && !strings.Contains(evidence, "unread") {
			down = append(down, unit+" ("+evidence+")")
		}
	}
	if len(down) > 0 && sustainedFailure("last-fail", true, defaultUnitSustain*time.Second) {
		reason := "declared units are not active: " + strings.Join(down, ", ")
		if cpu, ok := o.Readings.CPUPressure.Number(); ok && cpu >= thresholds.CPUPressurePercent {
			reason += "; escalation held by CPU saturation brake"
		}
		add(o, "unit-liveness", reason, []string{"declared units: vrooli-runtime-supervisor.service, vrooli-autoheal.service"})
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

func declaredUnits() []string {
	return []string{"vrooli-runtime-supervisor.service", "vrooli-autoheal.service"}
}

func unitActive(unit string) (bool, string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	switch current := strings.ToLower(runtimeGOOS()); current {
	case "linux":
		cmd := exec.CommandContext(ctx, "systemctl", "--user", "is-active", unit)
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
			if err := exec.CommandContext(ctx, "launchctl", "print", domain+"/"+strings.TrimSuffix(unit, ".service")).Run(); err == nil {
				return true, "launchctl print " + domain
			}
		}
		return false, "launchctl print unread"
	case "windows":
		if err := exec.CommandContext(ctx, "sc.exe", "query", unit).Run(); err != nil {
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
	return &hostpressure.PressureSnapshot{CapturedAt: state.Captured, ForkCounter: state.Counter}
}

func saveForkState(snapshot hostpressure.PressureSnapshot) {
	if snapshot.ForkCounter.State != hostpressure.Read || snapshot.CapturedAt.IsZero() {
		return
	}
	path := forkStatePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return
	}
	b, err := json.Marshal(forkState{Counter: snapshot.ForkCounter, Captured: snapshot.CapturedAt})
	if err != nil {
		return
	}
	tmp := path + fmt.Sprintf(".%d.tmp", os.Getpid())
	if err := os.WriteFile(tmp, b, 0o600); err == nil {
		_ = os.Rename(tmp, path)
	}
}

func addLiveFindings(o *output, thresholds thresholdSource) {
	cpuFinding := false
	if v, ok := o.Readings.CPUPressure.Number(); ok && v >= thresholds.CPUPressurePercent {
		cpuFinding = sustainedFailure("last-cpu-pressure", true, 60*time.Second)
	}
	if cpuFinding {
		v, _ := o.Readings.CPUPressure.Number()
		add(o, "cpu-pressure", fmt.Sprintf("CPU pressure %.1f%% meets or exceeds SB14 bar", v), []string{o.Readings.CPUPressure.Provenance})
	} else if _, ok := o.Readings.CPUPressure.Number(); ok {
		_ = sustainedFailure("last-cpu-pressure", false, 60*time.Second)
	}
	forkFinding := false
	if v, ok := o.Readings.ForkRate.Number(); ok && v >= thresholds.ForksPerSecond {
		forkFinding = sustainedFailure("last-fork-rate", true, 60*time.Second)
	}
	if forkFinding {
		v, _ := o.Readings.ForkRate.Number()
		add(o, "fork-rate", fmt.Sprintf("%.1f forks/s exceeds SB16 bar", v), []string{o.Readings.ForkRate.Provenance})
	} else if _, ok := o.Readings.ForkRate.Number(); ok {
		_ = sustainedFailure("last-fork-rate", false, 60*time.Second)
	}
	stranded := hostpressure.Stranded(o.Readings.Processes, 2)
	var strandedBytes uint64
	for _, p := range stranded {
		strandedBytes += p.Swapped
	}
	strandedFinding := float64(strandedBytes)/(1024*1024) >= thresholds.StrandedMemoryMB && len(stranded) > 0
	if sustainedFailure("last-stranded-memory", strandedFinding, 60*time.Second) {
		add(o, "stranded-memory", fmt.Sprintf("%.0f MB stranded across %d idle processes; top holder %s", float64(strandedBytes)/(1024*1024), len(stranded), stranded[0].Name), []string{"/proc/*/status"})
	}
}

type failureState struct {
	FirstObserved time.Time `json:"first_observed"`
}

var watchdogNow = func() time.Time { return time.Now().UTC() }

// sustainedFailure implements the watchdog's hysteresis without making a
// transient host observation look actionable. State files intentionally keep
// the legacy names for disk and unit liveness so upgrades do not erase their
// accumulated failure window.
func sustainedFailure(name string, failing bool, sustain time.Duration) bool {
	path := filepath.Join(filepath.Dir(forkStatePath()), "emergency-watchdog."+name)
	if !failing {
		_ = os.Remove(path)
		return false
	}
	now := watchdogNow()
	var state failureState
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &state)
	}
	if state.FirstObserved.IsZero() || now.Before(state.FirstObserved) {
		state.FirstObserved = now
		if data, err := json.Marshal(state); err == nil {
			if err := os.MkdirAll(filepath.Dir(path), 0o700); err == nil {
				_ = os.WriteFile(path, data, 0o600)
			}
		}
		return false
	}
	return now.Sub(state.FirstObserved) >= sustain
}

func readThresholds() thresholdSource {
	thresholds := thresholdSource{CPUPressurePercent: maxCPUPressurePercent, StrandedMemoryMB: maxStrandedMemoryMB, ForksPerSecond: maxForksPerSecond, CrashLoopsPerHour: maxCrashLoopsPerHour, Source: "compiled fallback"}
	paths := []string{}
	if configured := strings.TrimSpace(os.Getenv("VROOLI_SETPOINT_PATH")); configured != "" {
		paths = append(paths, configured)
	}
	if cwd, err := os.Getwd(); err == nil {
		paths = append(paths, filepath.Join(cwd, "scenarios/infrastructure-manager/setpoint/reliability-setpoint.json"))
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var document struct {
			Bars []struct {
				CellRef string  `json:"cell_ref"`
				Max     float64 `json:"max"`
			} `json:"bars"`
		}
		if json.Unmarshal(data, &document) != nil {
			continue
		}
		for _, bar := range document.Bars {
			switch bar.CellRef {
			case "substrate/SB14":
				if bar.Max > 0 {
					thresholds.CPUPressurePercent = bar.Max
				}
			case "substrate/SB15":
				if bar.Max > 0 {
					thresholds.StrandedMemoryMB = bar.Max
				}
			case "substrate/SB16":
				if bar.Max > 0 {
					thresholds.ForksPerSecond = bar.Max
				}
			case "availability/A2":
				if bar.Max > 0 {
					thresholds.CrashLoopsPerHour = bar.Max
				}
			}
		}
		thresholds.Source = path
		return thresholds
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

func fromFixture(root string, o *output) error {
	first := filepath.Join(root, "proc-stat-t0")
	second := filepath.Join(root, "proc-stat-t1")
	b0, e := os.ReadFile(first)
	if e != nil {
		return e
	}
	b1, e := os.ReadFile(second)
	if e != nil {
		return e
	}
	n0, ok := counter(string(b0))
	if !ok {
		return fmt.Errorf("fixture %s lacks process counter", first)
	}
	n1, ok := counter(string(b1))
	if !ok {
		return fmt.Errorf("fixture %s lacks process counter", second)
	}
	var m struct {
		Intervals map[string]float64 `json:"intervals_seconds"`
	}
	mb, e := os.ReadFile(filepath.Join(root, "manifest.json"))
	if e != nil {
		return e
	}
	if e = json.Unmarshal(mb, &m); e != nil {
		return e
	}
	elapsed := m.Intervals["proc_stat"]
	if elapsed <= 0 {
		return fmt.Errorf("fixture process interval is not positive")
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
			return parseErr
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
	stranded := hostpressure.Stranded(o.Readings.Processes, 2)
	if len(stranded) > 0 {
		add(o, "stranded-memory", fmt.Sprintf("%s holds %d swapped bytes", stranded[0].Name, stranded[0].Swapped), []string{"procs.tsv"})
	}
	if v, ok := o.Readings.ForkRate.Number(); ok && v >= o.Thresholds.ForksPerSecond {
		add(o, "fork-rate", fmt.Sprintf("%.1f forks/s observed; dominant source is the abandoned Airbyte KinD workload's crash-looping kubelet", v), []string{"proc-stat-t0", "proc-stat-t1", "manifest.json", "docker-inspect-airbyte.json", "kubelet-restarts.txt"})
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
	return nil
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
		parts := strings.SplitN(field, "=", 2)
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
		if len(f) != 5 {
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
