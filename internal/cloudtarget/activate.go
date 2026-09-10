package cloudtarget

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/shell"
)

// Activation strategies. side_by_side starts the candidate beside the
// current workload; maintenance declares an interval in which the workload
// is stopped first. Both are delegated to the lifecycle owner.
const (
	StrategySideBySide  = "side_by_side"
	StrategyMaintenance = "maintenance"
)

// ActiveRelease is the durable pointer at active-release.json.
type ActiveRelease struct {
	SchemaVersion   int    `json:"schema_version"`
	ActiveRelease   string `json:"active_release"`
	PreviousRelease string `json:"previous_release,omitempty"`
	ActivatedAt     string `json:"activated_at"`
	OperationID     string `json:"operation_id"`
	Fence           uint64 `json:"fence"`
	Strategy        string `json:"strategy"`
}

// ActivationIntent is written before the runtime switch and removed after
// the pointer commit. Its presence on disk means the switch was interrupted.
type ActivationIntent struct {
	SchemaVersion int      `json:"schema_version"`
	Candidate     string   `json:"candidate"`
	Previous      string   `json:"previous,omitempty"`
	OperationID   string   `json:"operation_id"`
	Fence         uint64   `json:"fence"`
	Strategy      string   `json:"strategy"`
	Scenarios     []string `json:"scenarios"`
	StartedAt     string   `json:"started_at"`
}

// Activation is what the runtime owner receives. Ports pins the listener
// ports the edge routes to (port name -> number); they reach the lifecycle
// owner as the conventional <NAME>_PORT environment so an activation never
// moves a public upstream.
type Activation struct {
	DeploymentID  string
	ReleaseDigest string
	ReleaseDir    string
	Scenarios     []string
	Strategy      string
	Ports         map[string]int
}

// Env renders the fixed-port environment for the lifecycle owner.
func (a Activation) Env() []string {
	names := make([]string, 0, len(a.Ports))
	for name := range a.Ports {
		names = append(names, name)
	}
	sort.Strings(names)
	env := make([]string, 0, len(names))
	for _, name := range names {
		env = append(env, PortEnvName(name)+"="+fmt.Sprint(a.Ports[name]))
	}
	return env
}

// PortEnvName maps a manifest port name onto its environment variable
// (ui -> UI_PORT, playwright_driver -> PLAYWRIGHT_DRIVER_PORT).
func PortEnvName(name string) string {
	return strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(name), "-", "_")) + "_PORT"
}

// RuntimeActivator (re)starts the workload on a release tree. The default
// delegates to `vrooli scenario restart` so process naming, ports, health
// checks and demand leases stay with the lifecycle owner.
type RuntimeActivator interface {
	Activate(ctx context.Context, activation Activation) error
}

// PortEnvRunner is a shell.Runner that can also run with extra environment. The
// activator uses it to pin listener ports; a plain shell.Runner is accepted
// when no ports are pinned.
type PortEnvRunner interface {
	shell.Runner
	RunEnv(ctx context.Context, env []string, name string, args ...string) ([]byte, error)
}

// OSEnvRunner executes commands through the operating system with extra
// environment appended to the process environment.
type OSEnvRunner struct{ shell.OSRunner }

// RunEnv implements PortEnvRunner.
func (OSEnvRunner) RunEnv(ctx context.Context, env []string, name string, args ...string) ([]byte, error) {
	return shell.CombinedOutput(shell.Spec{Context: ctx, Name: name, Args: args, Env: append(os.Environ(), env...)})
}

// ScenarioRestartActivator runs the lifecycle owner through argv. Each
// scenario is restarted from its directory inside the release tree via
// `--path`, so the runtime resolves the scenario root from the immutable
// release rather than from a mutable checkout. Readiness is the lifecycle
// owner's verdict: `vrooli scenario restart` exits non-zero when the scenario
// does not become healthy, and the active pointer is committed only after
// the activator returned nil.
type ScenarioRestartActivator struct {
	Runner     shell.Runner
	Executable string
}

func (a ScenarioRestartActivator) run(ctx context.Context, env []string, executable string, args ...string) ([]byte, error) {
	runner := a.Runner
	if runner == nil {
		runner = OSEnvRunner{}
	}
	if len(env) == 0 {
		return runner.Run(ctx, executable, args...)
	}
	envRunner, ok := runner.(PortEnvRunner)
	if !ok {
		return nil, fmt.Errorf("runner cannot pin listener ports %v: it carries no environment", env)
	}
	return envRunner.RunEnv(ctx, env, executable, args...)
}

func (a ScenarioRestartActivator) Activate(ctx context.Context, activation Activation) error {
	executable := a.Executable
	if executable == "" {
		executable = "vrooli"
	}
	env := activation.Env()
	if activation.Strategy == StrategyMaintenance {
		for _, scenario := range activation.Scenarios {
			if out, err := a.run(ctx, env, executable, "scenario", "stop", scenario, "--json"); err != nil {
				return fmt.Errorf("stop %s for maintenance activation: %v: %s", scenario, err, strings.TrimSpace(string(out)))
			}
		}
	}
	for _, scenario := range activation.Scenarios {
		scenarioDir := filepath.Join(activation.ReleaseDir, "scenarios", scenario)
		if out, err := a.run(ctx, env, executable, "scenario", "restart", scenario, "--path", scenarioDir, "--json"); err != nil {
			return fmt.Errorf("restart %s from %s: %v: %s", scenario, scenarioDir, err, strings.TrimSpace(string(out)))
		}
	}
	return nil
}

// ActivateRequest switches the deployment to a staged release.
type ActivateRequest struct {
	Effect    EffectRequest
	Release   string
	Strategy  string
	Scenarios []string
	Activator RuntimeActivator
	// Ports pins the listener ports the runtime must keep.
	Ports map[string]int
	// DataBindings are the declared persistent-data bindings; LegacyCarry
	// names heuristic-preserved directories to carry without a binding;
	// LegacyRoot is the in-place workdir a converted deployment adopts data
	// from on its first release activation.
	DataBindings []DataBinding
	LegacyCarry  []string
	LegacyRoot   string
	// RestartIfActive runs the activator even when the release is already
	// active (a start/resume of the recorded release) instead of reporting
	// unchanged.
	RestartIfActive bool
}

// RollbackRequest switches the deployment back to its retained predecessor.
type RollbackRequest struct {
	Effect       EffectRequest
	To           string
	Strategy     string
	Scenarios    []string
	Activator    RuntimeActivator
	Ports        map[string]int
	DataBindings []DataBinding
}

// ReadActive returns the active pointer, nil when nothing has been activated.
func (s *Store) ReadActive(deploymentID string) (*ActiveRelease, error) {
	path, err := s.activePath(deploymentID)
	if err != nil {
		return nil, err
	}
	var active ActiveRelease
	if err := readJSON(path, &active); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fail(CodeStoreIO, "read active release: %v", err)
	}
	return &active, nil
}

func (s *Store) readIntent(deploymentID string) (*ActivationIntent, error) {
	path, err := s.intentPath(deploymentID)
	if err != nil {
		return nil, err
	}
	var intent ActivationIntent
	if err := readJSON(path, &intent); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fail(CodeStoreIO, "read activation intent: %v", err)
	}
	return &intent, nil
}

// Activate switches to a complete staged release. Ordering: intent written,
// runtime switched through the activator, pointer committed atomically,
// intent cleared. A crash before the pointer commit leaves the prior pointer
// intact and the intent on disk for reconciliation.
func (s *Store) Activate(ctx context.Context, req ActivateRequest) (EffectResult, error) {
	req.Effect.Verb = "release.activate"
	strategy, err := normalizeStrategy(req.Strategy)
	if err != nil {
		return EffectResult{}, err
	}
	if err := validDigest(req.Release); err != nil {
		return EffectResult{}, err
	}
	req.Effect.Input = map[string]any{"release": req.Release, "strategy": strategy, "scenarios": req.Scenarios, "ports": req.Ports, "data_bindings": req.DataBindings, "legacy_carry": req.LegacyCarry, "legacy_root": req.LegacyRoot, "restart": req.RestartIfActive}
	return s.RunEffect(ctx, req.Effect, func(ctx context.Context) (map[string]any, Outcome, error) {
		return s.switchRelease(ctx, switchRequest{effect: req.Effect, digest: req.Release, strategy: strategy, scenarios: req.Scenarios, activator: req.Activator, ports: req.Ports, bindings: req.DataBindings, legacyCarry: req.LegacyCarry, legacyRoot: req.LegacyRoot, restart: req.RestartIfActive})
	})
}

// switchRequest is the resolved input of one release switch.
type switchRequest struct {
	effect      EffectRequest
	digest      string
	strategy    string
	scenarios   []string
	activator   RuntimeActivator
	ports       map[string]int
	bindings    []DataBinding
	legacyCarry []string
	legacyRoot  string
	restart     bool
	rollback    bool
}

// Rollback switches back to the retained predecessor and only to it.
func (s *Store) Rollback(ctx context.Context, req RollbackRequest) (EffectResult, error) {
	req.Effect.Verb = "release.rollback"
	strategy, err := normalizeStrategy(req.Strategy)
	if err != nil {
		return EffectResult{}, err
	}
	if err := validDigest(req.To); err != nil {
		return EffectResult{}, err
	}
	req.Effect.Input = map[string]any{"to": req.To, "strategy": strategy, "scenarios": req.Scenarios, "ports": req.Ports, "data_bindings": req.DataBindings}
	return s.RunEffect(ctx, req.Effect, func(ctx context.Context) (map[string]any, Outcome, error) {
		return s.switchRelease(ctx, switchRequest{effect: req.Effect, digest: req.To, strategy: strategy, scenarios: req.Scenarios, activator: req.Activator, ports: req.Ports, bindings: req.DataBindings, rollback: true})
	})
}

func normalizeStrategy(value string) (string, error) {
	switch strings.TrimSpace(value) {
	case "", StrategySideBySide:
		return StrategySideBySide, nil
	case StrategyMaintenance:
		return StrategyMaintenance, nil
	default:
		return "", refuse(CodeInvalidArgument, "strategy %q is not side_by_side or maintenance", value)
	}
}

func (s *Store) switchRelease(ctx context.Context, req switchRequest) (map[string]any, Outcome, error) {
	effect, digest, strategy, scenarios, activator, rollback := req.effect, req.digest, req.strategy, req.scenarios, req.activator, req.rollback
	details := map[string]any{"release": digest, "strategy": strategy, "rollback": rollback}
	releaseDir, err := s.ReleaseDir(effect.DeploymentID, digest)
	if err != nil {
		return details, OutcomeFailed, err
	}
	details["release_dir"] = releaseDir
	switch state := releaseState(releaseDir); state {
	case ReleaseStateMissing:
		return details, OutcomeFailed, refuse(CodeReleaseNotStaged, "release %s is not staged on this target", digest)
	case ReleaseStateIncomplete, ReleaseStateStaging:
		return details, OutcomeFailed, refuse(CodeReleaseIncomplete, "release %s is %s and cannot be activated", digest, state)
	}
	current, err := s.ReadActive(effect.DeploymentID)
	if err != nil {
		return details, OutcomeFailed, err
	}
	previous := ""
	if current != nil {
		previous = current.ActiveRelease
		details["previous_release"] = previous
	}
	if rollback {
		if current == nil || current.PreviousRelease != digest {
			retained := ""
			if current != nil {
				retained = current.PreviousRelease
			}
			return details, OutcomeFailed, refuse(CodeRollbackNotEligible, "release %s is not the retained predecessor", digest).
				withDetails(map[string]any{"retained_predecessor": retained})
		}
	}
	if current != nil && current.ActiveRelease == digest && !req.restart {
		details["previous_release"] = current.PreviousRelease
		return details, OutcomeUnchanged, nil
	}
	if current != nil && current.ActiveRelease == digest {
		// A restart of the recorded release keeps its predecessor.
		previous = current.PreviousRelease
		details["previous_release"] = previous
	}
	if len(scenarios) == 0 {
		scenarios, err = discoverScenarios(releaseDir)
		if err != nil {
			return details, OutcomeFailed, err
		}
	}
	for _, scenario := range scenarios {
		if err := validIdentifier("scenario id", scenario); err != nil {
			return details, OutcomeFailed, err
		}
	}
	details["scenarios"] = scenarios
	if activator == nil {
		activator = ScenarioRestartActivator{}
	}
	intentPath, _ := s.intentPath(effect.DeploymentID)
	intent := ActivationIntent{SchemaVersion: SchemaVersion, Candidate: digest, Previous: previous, OperationID: effect.OperationID, Fence: effect.Fence, Strategy: strategy, Scenarios: scenarios, StartedAt: s.now().Format(time.RFC3339Nano)}
	if err := writeJSONAtomic(intentPath, intent); err != nil {
		return details, OutcomeFailed, fail(CodeStoreIO, "record activation intent: %v", err)
	}
	if err := s.fault("activate:before_runtime"); err != nil {
		return details, OutcomeFailed, fail(CodeActivationFailed, "%v", err)
	}
	previousDir := ""
	if previous != "" && previous != digest {
		previousDir, _ = s.ReleaseDir(effect.DeploymentID, previous)
	}
	dataPlan, err := s.bindPersistentData(effect.DeploymentID, releaseDir, previousDir, req.legacyRoot, scenarios, req.bindings, req.legacyCarry)
	if err != nil {
		_ = os.Remove(intentPath)
		return details, OutcomeFailed, err
	}
	details["data_bindings"] = dataPlan.Bound
	details["legacy_unmapped"] = dataPlan.Unmapped
	if err := activator.Activate(ctx, Activation{DeploymentID: effect.DeploymentID, ReleaseDigest: digest, ReleaseDir: releaseDir, Scenarios: scenarios, Strategy: strategy, Ports: req.ports}); err != nil {
		// The runtime owner reported failure, so the outcome is known: clear
		// the intent and leave the prior pointer as the truth.
		_ = os.Remove(intentPath)
		return details, OutcomeFailed, fail(CodeActivationFailed, "runtime activation failed: %v", err).withDetails(map[string]any{"prior_active_release": previous})
	}
	if err := s.fault("activate:after_runtime"); err != nil {
		return details, OutcomeFailed, fail(CodeActivationFailed, "%v", err)
	}
	pointer := ActiveRelease{SchemaVersion: SchemaVersion, ActiveRelease: digest, PreviousRelease: previous, ActivatedAt: s.now().Format(time.RFC3339Nano), OperationID: effect.OperationID, Fence: effect.Fence, Strategy: strategy}
	activePath, _ := s.activePath(effect.DeploymentID)
	if err := writeJSONAtomic(activePath, pointer); err != nil {
		return details, OutcomeFailed, fail(CodeStoreIO, "commit active release: %v", err)
	}
	if err := os.Remove(intentPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return details, OutcomeFailed, fail(CodeStoreIO, "clear activation intent: %v", err)
	}
	details["active_release"] = digest
	details["activated_at"] = pointer.ActivatedAt
	return details, OutcomeSucceeded, nil
}

// discoverScenarios lists the scenarios a release tree carries under
// scenarios/<id>/ with a service declaration.
func discoverScenarios(releaseDir string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(releaseDir, "scenarios"))
	if err != nil {
		return nil, refuse(CodeReleaseHasNoScenarios, "release carries no scenarios directory")
	}
	var scenarios []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(releaseDir, "scenarios", entry.Name(), ".vrooli", "service.json")); err == nil {
			scenarios = append(scenarios, entry.Name())
		}
	}
	if len(scenarios) == 0 {
		return nil, refuse(CodeReleaseHasNoScenarios, "release carries no scenario with a service declaration")
	}
	sort.Strings(scenarios)
	return scenarios, nil
}
