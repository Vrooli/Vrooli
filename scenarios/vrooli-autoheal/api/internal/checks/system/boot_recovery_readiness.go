// Package system: boot-recovery readiness.
//
// Every earlier boot-protection surface answered "is the unit installed"; none
// answered "would the boot path work if the host rebooted right now". On
// 2026-09-02 a supervisor unit rendered two weeks earlier crash-looped 495
// times after boot because its argv no longer parsed, and every surface still
// read green. This check proves the preconditions while the host is healthy,
// each with a reason, and names the one repair that owns all of them:
// `vrooli setup`. It offers no actions because the scenario cannot perform
// any of the repairs; a check that offered them would be lying.
//
// [REQ:BOOT-RECOVERY-001]
package system

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	platformgo "github.com/vrooli/platform-go"
	"github.com/vrooli/repo-contract-go/cliinvoke"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/reporoot"
)

const (
	// BootRecoveryReadinessCheckID is the check's registry identity.
	BootRecoveryReadinessCheckID = "system-boot-recovery-readiness"

	// bootRecoveryRemediation is the single repair for every failed
	// precondition: the control plane owns the units, the loop binary, the
	// lingering policy and the validator run.
	bootRecoveryRemediation = "vrooli setup"

	// Precondition states.
	PreconditionOK           = "ok"
	PreconditionFailed       = "failed"
	PreconditionUndetermined = "undetermined"

	// Precondition names, in the order they are reported.
	PreconditionSafeguards    = "safeguards"
	PreconditionLoopPreflight = "loop-preflight"
	PreconditionUnitActive    = "unit-active"
	PreconditionLoopHeartbeat = "loop-heartbeat"
	PreconditionLingering     = "lingering"
	PreconditionValidator     = "validator"
	PreconditionContainment   = "containment"

	// containmentSafeguard converges vrooli-agents.slice, the ceiling every
	// coding-agent session is born inside; without it a build storm has
	// nothing to stop it and the boot path heals into a host that is gone.
	containmentSafeguard = "agent_session_containment"

	// The registry runs checks under a 30-second timeout. `setup status`
	// answers in about three seconds on this host; the budgets below keep
	// the whole run inside the registry's ceiling and report undetermined,
	// never a hang, when a probe overruns.
	setupStatusBudget  = 15 * time.Second
	loopSelfTestBudget = 8 * time.Second
	unitProbeBudget    = 3 * time.Second

	// loopHeartbeatWindow is how stale the loop's last tick may be before
	// the loop is considered wedged. The loop ticks every 60 seconds.
	loopHeartbeatWindow = 3 * time.Minute

	loopStatusRelPath = ".vrooli/state/vrooli-autoheal/loop-status.json"
	loopSelfTestExit  = 3

	bootPolicyDedicated = "dedicated"
)

// requiredSafeguards are the control-plane safeguards that together make the
// boot path: the loop's unit, the supervisor's unit, and the last-line timer.
var requiredSafeguards = []string{"autoheal_watchdog", "runtime_supervisor", "emergency_watchdog"}

// Precondition is one named, reasoned verdict.
type Precondition struct {
	Name   string `json:"name"`
	State  string `json:"state"`
	Reason string `json:"reason"`
}

// UnitState is what the service manager says about one core unit.
type UnitState struct {
	ActiveState string
	NRestarts   string
	Result      string
	// RestartsKnown is false where the manager exposes no restart counter
	// (launchd, the SCM); the precondition then reports activity only.
	RestartsKnown bool
}

// BootRecoveryProbes are the OS and CLI seams. Every field is replaceable so
// the verdict logic is testable without a host.
type BootRecoveryProbes struct {
	// SetupStatus returns the JSON of `vrooli setup status --json --phase readiness`.
	SetupStatus func(ctx context.Context) ([]byte, error)
	// LoopSelfTest runs the loop binary with --self-test and returns its
	// stdout and exit code. A non-nil error means the probe could not run.
	LoopSelfTest func(ctx context.Context) (stdout []byte, exitCode int, err error)
	// UnitState reads one core unit by its native name.
	UnitState func(ctx context.Context, unit string) (UnitState, error)
	// LoopStatus returns the loop's status file.
	LoopStatus func() ([]byte, error)
	// LoopBinarySHA256 hashes the loop binary on disk.
	LoopBinarySHA256 func() (string, error)
	// Lingering reports whether the user manager lingers for username.
	Lingering func(ctx context.Context, username string) (bool, error)
	// Username is the invoking user, for the lingering probe.
	Username func() (string, error)
	// AgentScopes counts the live vrooli-agent-*.scope units under the user
	// manager. A non-nil error means the manager could not be asked.
	AgentScopes func(ctx context.Context) (int, error)
	GOOS        string
	Now         func() time.Time
}

// BootRecoveryReadinessCheck proves the boot path while the host is healthy.
type BootRecoveryReadinessCheck struct {
	probes BootRecoveryProbes
}

// BootRecoveryOption configures the check.
type BootRecoveryOption func(*BootRecoveryReadinessCheck)

// WithBootRecoveryProbes replaces the seams (tests).
func WithBootRecoveryProbes(probes BootRecoveryProbes) BootRecoveryOption {
	return func(c *BootRecoveryReadinessCheck) { c.probes = mergeProbes(c.probes, probes) }
}

// NewBootRecoveryReadinessCheck builds the check with production probes.
func NewBootRecoveryReadinessCheck(opts ...BootRecoveryOption) *BootRecoveryReadinessCheck {
	c := &BootRecoveryReadinessCheck{probes: productionBootRecoveryProbes()}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *BootRecoveryReadinessCheck) ID() string    { return BootRecoveryReadinessCheckID }
func (c *BootRecoveryReadinessCheck) Title() string { return "Boot Recovery Readiness" }
func (c *BootRecoveryReadinessCheck) Description() string {
	return "Proves, while healthy, that the autoheal boot path would work: safeguards applied, loop preflight passing, core units active without restarts, loop ticking, lingering per policy, units accepted by the native validator, agent sessions contained by vrooli-agents.slice"
}

func (c *BootRecoveryReadinessCheck) Importance() string {
	return "A boot path that is only tested by rebooting fails at the one moment nobody is watching; every failed precondition here is repaired by `vrooli setup`"
}
func (c *BootRecoveryReadinessCheck) Category() checks.Category  { return checks.CategorySystem }
func (c *BootRecoveryReadinessCheck) IntervalSeconds() int       { return 3600 }
func (c *BootRecoveryReadinessCheck) Platforms() []platform.Type { return nil }

// setupStatusReport is the subset of `vrooli setup status --json` this check
// reads. Unknown fields are ignored on purpose; the report is versioned.
type setupStatusReport struct {
	Version    string `json:"version"`
	Safeguards []struct {
		Name           string         `json:"name"`
		Applied        bool           `json:"applied"`
		ExecutionState string         `json:"execution_state"`
		Notes          []string       `json:"notes"`
		Config         map[string]any `json:"config"`
		Evidence       struct {
			ValidatorVerdict *struct {
				State     string `json:"state"`
				Validator string `json:"validator"`
				Output    string `json:"output"`
			} `json:"validator_verdict"`
			// Probe is "undetermined" when a safeguard's live inspection
			// could not run (agent_session_containment sets it).
			Probe string `json:"probe"`
		} `json:"evidence"`
	} `json:"safeguards"`
}

// loopStatusFile is the subset of the loop's status file this check reads.
type loopStatusFile struct {
	LastTickAt   *time.Time `json:"last_tick_at"`
	State        string     `json:"state"`
	BinarySHA256 string     `json:"binary_sha256"`
	PID          int        `json:"pid"`
}

// loopPreflightResult mirrors the loop's PreflightResult.
type loopPreflightResult struct {
	OK     bool `json:"ok"`
	Checks []struct {
		Name   string `json:"name"`
		Status string `json:"status"`
		Reason string `json:"reason"`
	} `json:"checks"`
}

func (c *BootRecoveryReadinessCheck) Run(ctx context.Context) checks.Result {
	now := c.probes.Now()
	result := checks.Result{CheckID: c.ID(), Timestamp: now, Details: map[string]interface{}{
		"remediation": bootRecoveryRemediation,
		"evaluatedAt": now.UTC().Format(time.RFC3339),
	}}

	report, safeguards := c.safeguardsPrecondition(ctx)
	preconditions := []Precondition{
		safeguards,
		c.loopPreflightPrecondition(ctx),
		c.unitActivePrecondition(ctx),
		c.loopHeartbeatPrecondition(now),
		c.lingeringPrecondition(ctx, report),
		validatorPrecondition(report),
		c.containmentPrecondition(ctx, report),
	}

	var failed, undetermined []string
	list := make([]map[string]any, 0, len(preconditions))
	for _, p := range preconditions {
		list = append(list, map[string]any{"name": p.Name, "state": p.State, "reason": p.Reason})
		switch p.State {
		case PreconditionFailed:
			failed = append(failed, p.Name)
		case PreconditionUndetermined:
			undetermined = append(undetermined, p.Name)
		}
	}
	result.Details["preconditions"] = list
	result.Details["failedPreconditions"] = failed
	result.Details["undeterminedPreconditions"] = undetermined
	// The incident fingerprint keys on the failed set, so the same broken
	// precondition is one incident across hourly runs.
	result.Details["findingKey"] = strings.Join(failed, ",")

	switch {
	case len(failed) > 0:
		result.Status = checks.StatusCritical
		result.Message = fmt.Sprintf("Boot recovery would not work: %s failed (%s) — run `vrooli setup`", strings.Join(failed, ", "), reasonsFor(preconditions, failed))
	case len(undetermined) > 0:
		result.Status = checks.StatusUndetermined
		result.Message = fmt.Sprintf("Boot recovery readiness is undetermined: %s could not be probed (%s)", strings.Join(undetermined, ", "), reasonsFor(preconditions, undetermined))
	default:
		result.Status = checks.StatusOK
		result.Message = fmt.Sprintf("Boot recovery is ready: all %d preconditions hold", len(preconditions))
	}
	return result
}

// containmentPrecondition requires the agent slice to be applied with an
// accepted validator verdict, and counts the live agent scopes so the
// operator sees how many sessions the ceiling holds. An inspection the
// safeguard itself could not run, or a report it is missing from, is
// undetermined, never ok.
func (c *BootRecoveryReadinessCheck) containmentPrecondition(ctx context.Context, report *setupStatusReport) Precondition {
	p := Precondition{Name: PreconditionContainment}
	if report == nil {
		p.State = PreconditionUndetermined
		p.Reason = "setup status was not readable"
		return p
	}
	for _, s := range report.Safeguards {
		if s.Name != containmentSafeguard {
			continue
		}
		switch {
		case s.Evidence.Probe == "undetermined":
			p.State = PreconditionUndetermined
			p.Reason = containmentSafeguard + " could not inspect the slice: " + strings.Join(s.Notes, "; ")
		case !s.Applied:
			p.State = PreconditionFailed
			p.Reason = containmentSafeguard + " is not applied: " + strings.Join(s.Notes, "; ")
		case s.Evidence.ValidatorVerdict == nil || s.Evidence.ValidatorVerdict.State != "accepted":
			p.State = PreconditionFailed
			verdict := "absent"
			if s.Evidence.ValidatorVerdict != nil {
				verdict = s.Evidence.ValidatorVerdict.State
			}
			p.Reason = fmt.Sprintf("%s validator verdict is %s, not accepted", containmentSafeguard, verdict)
		default:
			p.State = PreconditionOK
			p.Reason = "vrooli-agents.slice applied with an accepted validator verdict"
			if c.probes.AgentScopes != nil {
				if scopes, err := c.probes.AgentScopes(ctx); err != nil {
					p.State = PreconditionUndetermined
					p.Reason += "; live agent scopes could not be counted: " + err.Error()
				} else {
					p.Reason += fmt.Sprintf("; %d live agent scope(s)", scopes)
				}
			}
		}
		return p
	}
	p.State = PreconditionUndetermined
	p.Reason = containmentSafeguard + " is not in the readiness phase; run `vrooli setup`"
	return p
}

// countAgentScopes asks the user manager for its vrooli-agent-*.scope units.
func countAgentScopes(ctx context.Context) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, unitProbeBudget)
	defer cancel()
	output, err := exec.CommandContext(ctx, "systemctl", "--user", "list-units", "vrooli-agent-*", "--plain", "--no-legend").Output()
	if err != nil {
		return 0, fmt.Errorf("systemctl --user list-units: %w", err)
	}
	count := 0
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "vrooli-agent-") {
			count++
		}
	}
	return count, nil
}

func reasonsFor(preconditions []Precondition, names []string) string {
	var parts []string
	for _, p := range preconditions {
		for _, name := range names {
			if p.Name == name {
				parts = append(parts, p.Name+": "+p.Reason)
			}
		}
	}
	return strings.Join(parts, "; ")
}

// safeguardsPrecondition runs `vrooli setup status --json --phase readiness`
// and requires every boot safeguard to be applied. The parsed report is
// returned so the lingering and validator preconditions read the same run.
func (c *BootRecoveryReadinessCheck) safeguardsPrecondition(ctx context.Context) (*setupStatusReport, Precondition) {
	p := Precondition{Name: PreconditionSafeguards}
	if c.probes.SetupStatus == nil {
		p.State, p.Reason = PreconditionUndetermined, "no setup status probe"
		return nil, p
	}
	ctx, cancel := context.WithTimeout(ctx, setupStatusBudget)
	defer cancel()
	raw, err := c.probes.SetupStatus(ctx)
	if err != nil {
		p.State, p.Reason = PreconditionUndetermined, "vrooli setup status --json --phase readiness: "+err.Error()
		return nil, p
	}
	var report setupStatusReport
	if err := json.Unmarshal(raw, &report); err != nil {
		p.State, p.Reason = PreconditionUndetermined, "setup status report is not JSON: "+err.Error()
		return nil, p
	}
	byName := map[string]int{}
	for i, s := range report.Safeguards {
		byName[s.Name] = i
	}
	var missing, unapplied []string
	for _, name := range requiredSafeguards {
		i, ok := byName[name]
		if !ok {
			missing = append(missing, name)
			continue
		}
		s := report.Safeguards[i]
		if !s.Applied {
			detail := s.ExecutionState
			if len(s.Notes) > 0 {
				detail += ": " + s.Notes[len(s.Notes)-1]
			}
			unapplied = append(unapplied, name+" ("+detail+")")
		}
	}
	switch {
	case len(missing) > 0:
		p.State, p.Reason = PreconditionFailed, "setup status does not report "+strings.Join(missing, ", ")
	case len(unapplied) > 0:
		p.State, p.Reason = PreconditionFailed, "not applied: "+strings.Join(unapplied, "; ")
	default:
		p.State, p.Reason = PreconditionOK, strings.Join(requiredSafeguards, ", ")+" applied"
	}
	return &report, p
}

// loopPreflightPrecondition runs the loop binary's own preflight. The loop
// knows what it needs to heal; this check does not second-guess it.
func (c *BootRecoveryReadinessCheck) loopPreflightPrecondition(ctx context.Context) Precondition {
	p := Precondition{Name: PreconditionLoopPreflight}
	if c.probes.LoopSelfTest == nil {
		p.State, p.Reason = PreconditionUndetermined, "no loop self-test probe"
		return p
	}
	ctx, cancel := context.WithTimeout(ctx, loopSelfTestBudget)
	defer cancel()
	stdout, exitCode, err := c.probes.LoopSelfTest(ctx)
	if err != nil {
		p.State, p.Reason = PreconditionUndetermined, "loop --self-test could not run: "+err.Error()
		return p
	}
	var preflight loopPreflightResult
	if parseErr := json.Unmarshal(stdout, &preflight); parseErr != nil {
		p.State, p.Reason = PreconditionUndetermined, fmt.Sprintf("loop --self-test exit %d with unparseable output: %v", exitCode, parseErr)
		return p
	}
	var failed []string
	for _, check := range preflight.Checks {
		if check.Status == "failed" {
			failed = append(failed, check.Name+": "+check.Reason)
		}
	}
	if !preflight.OK || exitCode == loopSelfTestExit || len(failed) > 0 {
		p.State, p.Reason = PreconditionFailed, "loop preflight failed: "+strings.Join(failed, "; ")
		if len(failed) == 0 {
			p.Reason = fmt.Sprintf("loop preflight reported ok=false (exit %d)", exitCode)
		}
		return p
	}
	if exitCode != 0 {
		p.State, p.Reason = PreconditionUndetermined, fmt.Sprintf("loop --self-test exited %d with a passing preflight", exitCode)
		return p
	}
	p.State, p.Reason = PreconditionOK, fmt.Sprintf("%d preflight checks passed", len(preflight.Checks))
	return p
}

// unitActivePrecondition requires each core daemon unit active with no
// restarts and a result other than start-limit-hit.
func (c *BootRecoveryReadinessCheck) unitActivePrecondition(ctx context.Context) Precondition {
	p := Precondition{Name: PreconditionUnitActive}
	if c.probes.UnitState == nil {
		p.State, p.Reason = PreconditionUndetermined, "no unit state probe"
		return p
	}
	var failed, unread, ok []string
	for _, unit := range platformgo.CoreUnits() {
		if unit.Kind != platformgo.KindDaemon {
			continue
		}
		name := unit.NativeName(c.probes.GOOS)
		if name == "" {
			continue
		}
		unitCtx, cancel := context.WithTimeout(ctx, unitProbeBudget)
		state, err := c.probes.UnitState(unitCtx, name)
		cancel()
		if err != nil {
			unread = append(unread, name+": "+err.Error())
			continue
		}
		switch {
		case state.ActiveState != "active":
			failed = append(failed, name+" is "+orUnknown(state.ActiveState))
		case state.Result == "start-limit-hit":
			failed = append(failed, name+" hit its start limit")
		case state.RestartsKnown && state.NRestarts != "0" && state.NRestarts != "":
			failed = append(failed, name+" restarted "+state.NRestarts+" times since it was started (Result="+orUnknown(state.Result)+")")
		default:
			ok = append(ok, name)
		}
	}
	switch {
	case len(failed) > 0:
		p.State, p.Reason = PreconditionFailed, strings.Join(failed, "; ")
	case len(unread) > 0:
		p.State, p.Reason = PreconditionUndetermined, strings.Join(unread, "; ")
	default:
		p.State, p.Reason = PreconditionOK, strings.Join(ok, ", ")+" active with zero restarts"
	}
	return p
}

func orUnknown(s string) string {
	if strings.TrimSpace(s) == "" {
		return "unknown"
	}
	return s
}

// loopHeartbeatPrecondition reads the loop's status file: the last tick must
// be recent, and the running loop must be the binary on disk (a loop that
// keeps running a replaced binary never gets any fix shipped since).
func (c *BootRecoveryReadinessCheck) loopHeartbeatPrecondition(now time.Time) Precondition {
	p := Precondition{Name: PreconditionLoopHeartbeat}
	if c.probes.LoopStatus == nil {
		p.State, p.Reason = PreconditionUndetermined, "no loop status probe"
		return p
	}
	raw, err := c.probes.LoopStatus()
	if err != nil {
		p.State, p.Reason = PreconditionUndetermined, "loop status file unreadable: "+err.Error()
		return p
	}
	var status loopStatusFile
	if err := json.Unmarshal(raw, &status); err != nil {
		p.State, p.Reason = PreconditionUndetermined, "loop status file is not JSON: "+err.Error()
		return p
	}
	if status.LastTickAt == nil || status.LastTickAt.IsZero() {
		p.State, p.Reason = PreconditionFailed, "loop has never ticked (state "+orUnknown(status.State)+")"
		return p
	}
	age := now.Sub(*status.LastTickAt)
	if age > loopHeartbeatWindow {
		p.State, p.Reason = PreconditionFailed, fmt.Sprintf("last tick %s ago, older than %s (state %s)", age.Round(time.Second), loopHeartbeatWindow, orUnknown(status.State))
		return p
	}
	if c.probes.LoopBinarySHA256 != nil && status.BinarySHA256 != "" {
		onDisk, hashErr := c.probes.LoopBinarySHA256()
		switch {
		case hashErr != nil:
			p.State, p.Reason = PreconditionUndetermined, fmt.Sprintf("last tick %s ago; loop binary on disk unreadable: %v", age.Round(time.Second), hashErr)
			return p
		case onDisk != status.BinarySHA256:
			p.State, p.Reason = PreconditionFailed, fmt.Sprintf("running loop binary %s differs from the one on disk %s; the unit must restart to pick it up", short(status.BinarySHA256), short(onDisk))
			return p
		}
	}
	p.State, p.Reason = PreconditionOK, fmt.Sprintf("last tick %s ago, running the on-disk binary", age.Round(time.Second))
	return p
}

func short(sha string) string {
	if len(sha) > 12 {
		return sha[:12]
	}
	return sha
}

// lingeringPrecondition applies only to a dedicated-host boot policy on
// Linux: a user manager that does not linger never starts before login.
func (c *BootRecoveryReadinessCheck) lingeringPrecondition(ctx context.Context, report *setupStatusReport) Precondition {
	p := Precondition{Name: PreconditionLingering}
	if c.probes.GOOS != "linux" {
		p.State, p.Reason = PreconditionOK, "not required on "+c.probes.GOOS
		return p
	}
	if report == nil {
		p.State, p.Reason = PreconditionUndetermined, "boot policy unknown: setup status was not read"
		return p
	}
	policy := ""
	for _, s := range report.Safeguards {
		if s.Name == "autoheal_watchdog" {
			if value, ok := s.Config["boot_policy"].(string); ok {
				policy = value
			}
		}
	}
	if policy != bootPolicyDedicated {
		p.State, p.Reason = PreconditionOK, "not required for boot policy "+orUnknown(policy)
		return p
	}
	if c.probes.Lingering == nil || c.probes.Username == nil {
		p.State, p.Reason = PreconditionUndetermined, "no lingering probe"
		return p
	}
	username, err := c.probes.Username()
	if err != nil || strings.TrimSpace(username) == "" {
		p.State, p.Reason = PreconditionUndetermined, "invoking user unknown"
		return p
	}
	unitCtx, cancel := context.WithTimeout(ctx, unitProbeBudget)
	defer cancel()
	lingering, err := c.probes.Lingering(unitCtx, username)
	if err != nil {
		p.State, p.Reason = PreconditionUndetermined, "loginctl: "+err.Error()
		return p
	}
	if !lingering {
		p.State, p.Reason = PreconditionFailed, "boot policy is dedicated but lingering is not enabled for "+username
		return p
	}
	p.State, p.Reason = PreconditionOK, "lingering enabled for "+username+" (dedicated policy)"
	return p
}

// validatorPrecondition reads the verdict each safeguard recorded from the
// native validator. Unavailable is unproven, which is undetermined, never ok.
func validatorPrecondition(report *setupStatusReport) Precondition {
	p := Precondition{Name: PreconditionValidator}
	if report == nil {
		p.State, p.Reason = PreconditionUndetermined, "setup status was not read"
		return p
	}
	var rejected, unavailable, accepted, missing []string
	for _, name := range requiredSafeguards {
		found := false
		for _, s := range report.Safeguards {
			if s.Name != name {
				continue
			}
			found = true
			verdict := s.Evidence.ValidatorVerdict
			switch {
			case verdict == nil:
				missing = append(missing, name)
			case verdict.State == string(platformgo.VerdictRejected):
				rejected = append(rejected, name+": "+verdict.Validator+": "+verdict.Output)
			case verdict.State == string(platformgo.VerdictUnavailable):
				unavailable = append(unavailable, name+": "+verdict.Validator+" "+verdict.Output)
			case verdict.State == string(platformgo.VerdictAccepted):
				accepted = append(accepted, name)
			default:
				unavailable = append(unavailable, name+": unknown verdict "+verdict.State)
			}
		}
		if !found {
			missing = append(missing, name)
		}
	}
	switch {
	case len(rejected) > 0:
		p.State, p.Reason = PreconditionFailed, "native validator rejected: "+strings.Join(rejected, "; ")
	case len(unavailable) > 0:
		p.State, p.Reason = PreconditionUndetermined, "native validator unavailable: "+strings.Join(unavailable, "; ")
	case len(missing) > 0:
		p.State, p.Reason = PreconditionUndetermined, "no validator verdict recorded for "+strings.Join(missing, ", ")
	default:
		p.State, p.Reason = PreconditionOK, "native validator accepted "+strings.Join(accepted, ", ")
	}
	return p
}

// --- production probes ------------------------------------------------------

func productionBootRecoveryProbes() BootRecoveryProbes {
	return BootRecoveryProbes{
		SetupStatus:      runSetupStatus,
		LoopSelfTest:     runLoopSelfTest,
		UnitState:        readUnitState,
		LoopStatus:       readLoopStatusFile,
		LoopBinarySHA256: hashLoopBinary,
		Lingering:        readLingering,
		AgentScopes:      countAgentScopes,
		Username:         currentUsername,
		GOOS:             runtime.GOOS,
		Now:              time.Now,
	}
}

func mergeProbes(base, override BootRecoveryProbes) BootRecoveryProbes {
	if override.SetupStatus != nil {
		base.SetupStatus = override.SetupStatus
	}
	if override.LoopSelfTest != nil {
		base.LoopSelfTest = override.LoopSelfTest
	}
	if override.UnitState != nil {
		base.UnitState = override.UnitState
	}
	if override.LoopStatus != nil {
		base.LoopStatus = override.LoopStatus
	}
	if override.LoopBinarySHA256 != nil {
		base.LoopBinarySHA256 = override.LoopBinarySHA256
	}
	if override.Lingering != nil {
		base.Lingering = override.Lingering
	}
	if override.AgentScopes != nil {
		base.AgentScopes = override.AgentScopes
	}
	if override.Username != nil {
		base.Username = override.Username
	}
	if override.GOOS != "" {
		base.GOOS = override.GOOS
	}
	if override.Now != nil {
		base.Now = override.Now
	}
	return base
}

// runSetupStatus invokes the control plane through the shared invoker so the
// binary resolution and pipe discipline match every other supervisor.
func runSetupStatus(ctx context.Context) ([]byte, error) {
	home, _ := os.UserHomeDir()
	binary, err := cliinvoke.Resolve(cliinvoke.ResolveOptions{RuntimeHome: filepath.Join(home, ".vrooli")})
	if err != nil {
		return nil, err
	}
	result := cliinvoke.Run(ctx, cliinvoke.Invocation{Binary: binary, Args: cliinvoke.SetupStatusReadiness(), Timeout: setupStatusBudget})
	if result.Class != cliinvoke.OK {
		return nil, result.Error()
	}
	return result.Stdout, nil
}

// loopBinaryPath is the loop binary the unit executes: the scenario's built
// CLI, next to the repository. It is the path the autoheal_watchdog safeguard
// renders into the unit.
func loopBinaryPath() (string, error) {
	root := reporoot.ResolveFromOS()
	if root == "" {
		return "", errors.New("repository root could not be resolved")
	}
	name := "vrooli-autoheal-loop"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(root, "scenarios", "vrooli-autoheal", "cli", name), nil
}

func runLoopSelfTest(ctx context.Context) ([]byte, int, error) {
	binary, err := loopBinaryPath()
	if err != nil {
		return nil, 0, err
	}
	if _, statErr := os.Stat(binary); statErr != nil {
		return nil, 0, fmt.Errorf("loop binary: %w", statErr)
	}
	cmd := exec.CommandContext(ctx, binary, "--self-test")
	cmd.Dir = filepath.Dir(binary)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	stdout, runErr := cmd.Output()
	var exitErr *exec.ExitError
	switch {
	case runErr == nil:
		return stdout, 0, nil
	case errors.As(runErr, &exitErr):
		return stdout, exitErr.ExitCode(), nil
	default:
		return nil, 0, fmt.Errorf("%w: %s", runErr, strings.TrimSpace(stderr.String()))
	}
}

// readUnitState asks the service manager. On Linux systemd exposes the
// restart counter and the last result; launchd and the SCM expose activity
// only, which the precondition reports as such.
func readUnitState(ctx context.Context, unit string) (UnitState, error) {
	switch runtime.GOOS {
	case "linux":
		out, err := exec.CommandContext(ctx, "systemctl", "--user", "show", "-p", "ActiveState", "-p", "NRestarts", "-p", "Result", unit).Output()
		if err != nil {
			return UnitState{}, fmt.Errorf("systemctl --user show %s: %w", unit, err)
		}
		state := UnitState{RestartsKnown: true}
		for _, line := range strings.Split(string(out), "\n") {
			key, value, ok := strings.Cut(strings.TrimSpace(line), "=")
			if !ok {
				continue
			}
			switch key {
			case "ActiveState":
				state.ActiveState = value
			case "NRestarts":
				state.NRestarts = value
			case "Result":
				state.Result = value
			}
		}
		if state.ActiveState == "" {
			return UnitState{}, fmt.Errorf("systemctl --user show %s reported no ActiveState", unit)
		}
		return state, nil
	default:
		result, err := platformgo.NativeServiceStatus(platformgo.NativeServiceOptions{Name: unit, User: true, Kind: platformgo.KindDaemon})
		if err != nil {
			return UnitState{}, err
		}
		state := UnitState{ActiveState: string(result.State)}
		if result.State == platformgo.ServiceStateRunning {
			state.ActiveState = "active"
		}
		return state, nil
	}
}

func readLoopStatusFile() ([]byte, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return os.ReadFile(filepath.Join(home, filepath.FromSlash(loopStatusRelPath)))
}

func hashLoopBinary() (string, error) {
	binary, err := loopBinaryPath()
	if err != nil {
		return "", err
	}
	f, err := os.Open(binary)
	if err != nil {
		return "", err
	}
	defer f.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func readLingering(ctx context.Context, username string) (bool, error) {
	if _, err := os.Stat(filepath.Join("/var/lib/systemd/linger", username)); err == nil {
		return true, nil
	}
	out, err := exec.CommandContext(ctx, "loginctl", "show-user", username, "--property=Linger").Output()
	if err != nil {
		return false, fmt.Errorf("loginctl show-user %s: %w", username, err)
	}
	return strings.TrimSpace(string(out)) == "Linger=yes", nil
}

func currentUsername() (string, error) {
	current, err := user.Current()
	if err != nil {
		return "", err
	}
	return current.Username, nil
}

var _ checks.Check = (*BootRecoveryReadinessCheck)(nil)
