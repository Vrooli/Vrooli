package setup

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/logx"
	"github.com/vrooli/vrooli/internal/tuning"
)

// SetupPhase is the stable machine-facing identity of a setup stage. Labels
// are presentation; callers should persist and test IDs.
type SetupPhase string

const (
	PhaseValidation             SetupPhase = "validation"
	PhaseProject                SetupPhase = "project"
	PhaseFilesystem             SetupPhase = "filesystem"
	PhaseResolution             SetupPhase = "resolution"
	PhaseBootstrap              SetupPhase = "bootstrap"
	PhaseRequirements           SetupPhase = "requirements"
	PhaseGeneratedPackages      SetupPhase = "generated-packages"
	PhaseCredentials            SetupPhase = "credentials"
	PhaseCredentialCapabilities SetupPhase = "credential-capabilities"
	PhasePrivilegeBroker        SetupPhase = "privilege-broker"
	PhaseGit                    SetupPhase = "git"
	PhaseResources              SetupPhase = "resources"
	PhaseCLI                    SetupPhase = "cli"
	PhaseFinalize               SetupPhase = "finalize"
	PhaseCompletion             SetupPhase = "completion"
)

type phaseInfo struct {
	ID    SetupPhase
	Label string
}

var setupPhases = []phaseInfo{
	{PhaseValidation, "Validate host"},
	{PhaseProject, "Load project"},
	{PhaseFilesystem, "Prepare filesystem"},
	{PhaseResolution, "Resolve requirements"},
	{PhaseBootstrap, "Bootstrap tools"},
	{PhaseRequirements, "Apply host requirements"},
	{PhaseGeneratedPackages, "Generate packages"},
	{PhaseCredentials, "Configure credentials"},
	{PhaseCredentialCapabilities, "Discover capabilities"},
	{PhasePrivilegeBroker, "Install privilege broker"},
	{PhaseGit, "Configure Git"},
	{PhaseResources, "Reconcile resources"},
	{PhaseCLI, "Synchronize resource CLIs"},
	{PhaseFinalize, "Refresh project CLIs"},
	{PhaseCompletion, "Complete setup"},
}

type ProgressEventKind string

const (
	EventSetupStarted     ProgressEventKind = "setup_started"
	EventPhaseStarted     ProgressEventKind = "phase_started"
	EventOperationChanged ProgressEventKind = "operation_changed"
	EventHeartbeat        ProgressEventKind = "heartbeat"
	EventPhaseCompleted   ProgressEventKind = "phase_completed"
	EventPhaseFailed      ProgressEventKind = "phase_failed"
	EventSetupCompleted   ProgressEventKind = "setup_completed"
	EventSetupInterrupted ProgressEventKind = "setup_interrupted"
)

// ProgressEvent is intentionally safe to serialize. Error text is never
// included: errors can contain command arguments, paths, or secrets.
type ProgressEvent struct {
	At                  time.Time         `json:"at"`
	Kind                ProgressEventKind `json:"kind"`
	Phase               SetupPhase        `json:"phase,omitempty"`
	PhaseLabel          string            `json:"phase_label,omitempty"`
	PhaseIndex          int               `json:"phase_index,omitempty"`
	PhaseCount          int               `json:"phase_count,omitempty"`
	Operation           string            `json:"operation,omitempty"`
	Elapsed             time.Duration     `json:"-"`
	LastUpdateAge       time.Duration     `json:"-"`
	ElapsedMilliseconds int64             `json:"elapsed_ms,omitempty"`
	LastUpdateAgeMillis int64             `json:"last_update_age_ms,omitempty"`
	DryRun              bool              `json:"dry_run,omitempty"`
}

type activeSetupState struct {
	Version    string     `json:"version"`
	RunID      string     `json:"run_id"`
	Status     string     `json:"status"`
	PID        int        `json:"pid"`
	Host       string     `json:"host,omitempty"`
	StartedAt  time.Time  `json:"started_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	Phase      SetupPhase `json:"phase,omitempty"`
	PhaseLabel string     `json:"phase_label,omitempty"`
	Operation  string     `json:"operation,omitempty"`
}

type ProgressSink interface{ Publish(ProgressEvent) }

type writerSink struct {
	w     io.Writer
	json  bool
	quiet bool
	mu    sync.Mutex
}

func (s *writerSink) Publish(ev ProgressEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.quiet {
		return
	}
	ev.Operation = redactProgressText(ev.Operation)
	if s.json {
		ev.ElapsedMilliseconds = ev.Elapsed.Milliseconds()
		ev.LastUpdateAgeMillis = ev.LastUpdateAge.Milliseconds()
		b, err := json.Marshal(ev)
		if err == nil {
			_, _ = fmt.Fprintf(s.w, "%s\n", b)
		}
		return
	}
	line := renderSetupProgress(ev)
	if line != "" {
		_, _ = io.WriteString(s.w, line)
	}
}

func renderSetupProgress(ev ProgressEvent) string {
	phase := ev.PhaseLabel
	if phase == "" {
		phase = string(ev.Phase)
	}
	position := ""
	if ev.PhaseIndex > 0 && ev.PhaseCount > 0 {
		position = fmt.Sprintf("%d/%d · ", ev.PhaseIndex, ev.PhaseCount)
	}
	switch ev.Kind {
	case EventSetupStarted:
		return "SETUP  · Starting project setup\n"
	case EventPhaseStarted:
		return fmt.Sprintf("SETUP  · %s%s\n", position, phase)
	case EventOperationChanged:
		ev.Operation = redactProgressText(ev.Operation)
		if ev.Operation == "" {
			return ""
		}
		return fmt.Sprintf("         ↳ %s\n", ev.Operation)
	case EventHeartbeat:
		ev.Operation = redactProgressText(ev.Operation)
		if ev.Operation != "" {
			return fmt.Sprintf("         · still working: %s (%s elapsed)\n", ev.Operation, formatDuration(ev.Elapsed))
		}
		return fmt.Sprintf("         · still working (%s elapsed)\n", formatDuration(ev.Elapsed))
	case EventPhaseCompleted:
		return fmt.Sprintf("         ✓ %s (%s)\n", phase, formatDuration(ev.Elapsed))
	case EventPhaseFailed:
		return fmt.Sprintf("         ! %s stopped after %s\n", phase, formatDuration(ev.Elapsed))
	case EventSetupCompleted:
		return "SETUP  · Complete\n"
	case EventSetupInterrupted:
		return "SETUP  · Interrupted; the last reported phase is recorded above\n"
	default:
		return ""
	}
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return "<1s"
	}
	return d.Round(time.Second).String()
}

var sensitiveProgressValue = regexp.MustCompile(`(?i)(password|passphrase|secret|token|api[_-]?key)(\s*[:=]\s*)[^\s,;]+`)

func redactProgressText(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return sensitiveProgressValue.ReplaceAllString(value, `$1$2[redacted]`)
}

type progressOptions struct {
	Now            func() time.Time
	FirstHeartbeat time.Duration
	HeartbeatEvery time.Duration
	DryRun         bool
	JSON           bool
	StatePath      string
}

// heartbeatDue is the pure scheduling rule used by the live timer and tests.
// It deliberately has no wall-clock or terminal dependencies.
func heartbeatDue(now, phaseStarted, lastHeartbeat time.Time, first, every time.Duration) bool {
	if phaseStarted.IsZero() || now.Before(phaseStarted) {
		return false
	}
	if lastHeartbeat.IsZero() {
		return !now.Before(phaseStarted.Add(first))
	}
	return !now.Before(lastHeartbeat.Add(every))
}

type progressCoordinator struct {
	sink           ProgressSink
	now            func() time.Time
	firstHeartbeat time.Duration
	heartbeatEvery time.Duration
	dryRun         bool
	statePath      string
	runID          string
	host           string
	pid            int
	mu             sync.Mutex
	phase          phaseInfo
	runStarted     time.Time
	phaseStarted   time.Time
	lastUpdate     time.Time
	operation      string
	stop           chan struct{}
	done           chan struct{}
}

func newProgressCoordinator(w io.Writer, opts progressOptions) *progressCoordinator {
	if w == nil {
		w = io.Discard
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	first := opts.FirstHeartbeat
	if first <= 0 {
		first = tuning.ControlPlaneClientTimeout()
	}
	every := opts.HeartbeatEvery
	if every <= 0 {
		every = tuning.SetupProgressObservationInterval()
	}
	format := strings.ToLower(strings.TrimSpace(os.Getenv("VROOLI_SETUP_PROGRESS_FORMAT")))
	jsonOutput := opts.JSON || format == string(logx.FormatJSON) || format == "ndjson"
	quiet := format == "quiet" || strings.EqualFold(strings.TrimSpace(os.Getenv("VROOLI_SETUP_PROGRESS")), "quiet")
	host, _ := os.Hostname()
	return &progressCoordinator{sink: &writerSink{w: w, json: jsonOutput, quiet: quiet}, now: now, firstHeartbeat: first, heartbeatEvery: every, dryRun: opts.DryRun, statePath: opts.StatePath, runID: fmt.Sprintf("setup-%d-%d", os.Getpid(), now().UnixNano()), host: host, pid: os.Getpid(), phase: setupPhases[0]}
}

func (p *progressCoordinator) publish(kind ProgressEventKind) {
	p.mu.Lock()
	ev := ProgressEvent{At: p.now(), Kind: kind, DryRun: p.dryRun}
	started := p.phaseStarted
	if started.IsZero() {
		started = p.runStarted
	}
	if !started.IsZero() {
		ev.Elapsed = ev.At.Sub(started)
	}
	if !p.lastUpdate.IsZero() {
		ev.LastUpdateAge = ev.At.Sub(p.lastUpdate)
	}
	ev.Phase, ev.PhaseLabel, ev.Operation = p.phase.ID, p.phase.Label, redactProgressText(p.operation)
	for i, phase := range setupPhases {
		if phase.ID == p.phase.ID {
			ev.PhaseIndex = i + 1
			break
		}
	}
	ev.PhaseCount = len(setupPhases)
	p.lastUpdate = ev.At
	state := activeSetupState{Version: "v1", RunID: p.runID, Status: "running", PID: p.pid, Host: p.host, StartedAt: p.runStarted, UpdatedAt: ev.At, Phase: ev.Phase, PhaseLabel: ev.PhaseLabel, Operation: ev.Operation}
	p.mu.Unlock()
	p.writeState(state)
	// Rendering is best effort. A broken diagnostics pipe must never change
	// setup's host-remediation result.
	func() { defer func() { _ = recover() }(); p.sink.Publish(ev) }()
}

func (p *progressCoordinator) writeState(state activeSetupState) {
	if strings.TrimSpace(p.statePath) == "" {
		return
	}
	b, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return
	}
	if _, err := config.EnsureOwnedDir(filepath.Dir(p.statePath)); err != nil {
		return
	}
	_ = config.WriteOwnedFileAtomic(p.statePath, append(b, '\n'), tuning.PermSecret)
}

func (p *progressCoordinator) Start() {
	p.runStarted = p.now()
	p.publish(EventSetupStarted)
}

func (p *progressCoordinator) StartPhase(id SetupPhase) {
	if p.stop != nil {
		p.StopPhase()
	}
	for _, phase := range setupPhases {
		if phase.ID == id {
			p.phase = phase
			break
		}
	}
	p.operation = ""
	p.phaseStarted = p.now()
	p.lastUpdate = p.phaseStarted
	p.publish(EventPhaseStarted)
	p.stop, p.done = make(chan struct{}), make(chan struct{})
	go p.heartbeatLoop(p.stop, p.done)
}

func (p *progressCoordinator) CurrentPhase() SetupPhase {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.phase.ID
}

func (p *progressCoordinator) Operation(label string) {
	p.mu.Lock()
	changed := strings.TrimSpace(label) != p.operation
	p.operation = strings.TrimSpace(label)
	p.mu.Unlock()
	if changed {
		p.publish(EventOperationChanged)
	}
}

func (p *progressCoordinator) heartbeatLoop(stop <-chan struct{}, done chan<- struct{}) {
	timer := time.NewTimer(p.firstHeartbeat)
	defer func() {
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		close(done)
	}()
	select {
	case <-stop:
		return
	case <-timer.C:
	}
	for {
		p.publish(EventHeartbeat)
		timer.Reset(p.heartbeatEvery)
		select {
		case <-stop:
			return
		case <-timer.C:
		}
	}
}

func (p *progressCoordinator) StopPhase() {
	if p.stop == nil {
		return
	}
	close(p.stop)
	<-p.done
	p.stop, p.done = nil, nil
}

func (p *progressCoordinator) CompletePhase() { p.StopPhase(); p.publish(EventPhaseCompleted) }
func (p *progressCoordinator) FailPhase()     { p.StopPhase(); p.publish(EventPhaseFailed) }
func (p *progressCoordinator) Finish(err error) {
	if err != nil {
		p.FailPhase()
		p.writeTerminalState("failed")
		return
	}
	p.StopPhase()
	p.publish(EventSetupCompleted)
	p.writeTerminalState("completed")
}

func (p *progressCoordinator) writeTerminalState(status string) {
	p.mu.Lock()
	state := activeSetupState{Version: "v1", RunID: p.runID, Status: status, PID: p.pid, Host: p.host, StartedAt: p.runStarted, UpdatedAt: p.now(), Phase: p.phase.ID, PhaseLabel: p.phase.Label, Operation: redactProgressText(p.operation)}
	p.mu.Unlock()
	p.writeState(state)
}

func readActiveSetupState(path string) (activeSetupState, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return activeSetupState{}, err
	}
	var state activeSetupState
	if err := json.Unmarshal(b, &state); err != nil {
		return activeSetupState{}, err
	}
	return state, nil
}

func renderActiveSetupState(w io.Writer, path string, now time.Time) {
	state, err := readActiveSetupState(path)
	if err != nil || state.RunID == "" {
		return
	}
	age := now.Sub(state.UpdatedAt).Round(time.Second)
	if age < 0 {
		age = 0
	}
	status := state.Status
	if status == "running" && age > tuning.SetupProgressStaleThreshold() && !processIdentityAlive(state.PID, state.Host) {
		status = "possibly stale"
	}
	_, _ = fmt.Fprintf(w, "[INFO]    Last setup run: %s (%s, updated %s ago)\n", state.RunID, status, formatDuration(age))
	if state.PhaseLabel != "" {
		_, _ = fmt.Fprintf(w, "[INFO]    Last setup phase: %s", state.PhaseLabel)
		if state.Operation != "" {
			_, _ = fmt.Fprintf(w, " — %s", state.Operation)
		}
		_, _ = io.WriteString(w, "\n")
	}
}

func progressWriter(stderr, stdout io.Writer) io.Writer {
	if stderr != nil {
		return stderr
	}
	return stdout
}
