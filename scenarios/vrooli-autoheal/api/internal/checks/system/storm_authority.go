package system

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	platformgo "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

// The storm authority is the one component allowed to act on an attributed
// fork storm: it freezes (never kills) the agent session scope the watchdog
// named, under the runtime recovery gate, with a decision row, reversibly.
// The watchdog senses; this check reads; the authority decides.
const (
	// ContainStormActionID is the recovery action id on the watchdog report check.
	ContainStormActionID = "contain-storm"
	// ContainStormAutomatic runs the action from the auto-heal pass after a
	// sustained finding; ContainStormProposeOnly keeps it operator-only.
	ContainStormAutomatic   = "automatic"
	ContainStormProposeOnly = "propose_only"

	// agentScopeSegment is the cgroup path segment every containable scope
	// carries; anything else is a supervisor, a resource or the desktop.
	agentScopeSegment = "/vrooli-agents.slice/vrooli-agent-"

	// stormDecisionScenario is the scenario column of every storm decision
	// row; the scope name travels in the details and the reason.
	stormDecisionScenario = "vrooli-agents"
	// noEpochID is the epoch column when the runtime registry holds no
	// pressure epoch: the row still records that the gate was consulted.
	noEpochID = "no-epoch"

	// Decision states for storm rows; the restart controller's states do
	// not describe a freeze.
	StormDecisionContained = "contained"
	StormDecisionThawed    = "thawed"
	StormDecisionRefused   = "refused"
)

// StormTarget is the attributed parent a finding names.
type StormTarget struct {
	Finding    string `json:"finding"`
	PID        int64  `json:"pid"`
	Name       string `json:"name"`
	Children   int    `json:"children"`
	ScopePath  string `json:"scope_path"`
	ScopeName  string `json:"scope_name"`
	WorkingDir string `json:"working_dir,omitempty"`
}

// StormDecision is one row the authority writes.
type StormDecision struct {
	EpochID        string
	State          string
	Reason         string
	IdempotencyKey string
	Target         StormTarget
	At             time.Time
}

// StormOutcome is what a contain call did.
type StormOutcome struct {
	Scope       platformgo.ScopeRef
	Target      StormTarget
	EpochID     string
	Decision    StormDecision
	ThawCommand string
	FrozenAt    time.Time
}

// StormAuthority holds the seams the action runs through. Every field is
// injectable so the freeze, the gate and the decision row are tested without
// a cgroup; NewProductionStormAuthority binds the real ones.
type StormAuthority struct {
	Mode string
	// Gate answers whether recovery coordination is readable and which
	// pressure epoch is open. A gate that cannot be read refuses: losing
	// coordination under a storm must not create two authorities.
	Gate func(ctx context.Context) (epochID string, allowed bool, reason string)
	// Record writes one decision row; the idempotency key makes a repeated
	// freeze of the same scope in the same epoch one row.
	Record     func(ctx context.Context, decision StormDecision) error
	Freeze     func(ref platformgo.ScopeRef) error
	Thaw       func(ref platformgo.ScopeRef) error
	Frozen     func(ref platformgo.ScopeRef) (bool, error)
	WorkingDir func(pid int) (string, error)
	Now        func() time.Time
}

// NewProductionStormAuthority binds the authority to this host: the runtime
// registry under home for the gate and the decision rows, platform-go for
// the freeze. An empty mode is automatic (D5).
func NewProductionStormAuthority(mode string) *StormAuthority {
	home, _ := os.UserHomeDir()
	return &StormAuthority{
		Mode:       normalizeContainStormMode(mode),
		Gate:       registryGate(home),
		Record:     registryRecorder(home),
		Freeze:     platformgo.FreezeScope,
		Thaw:       platformgo.ThawScope,
		Frozen:     platformgo.ScopeFrozen,
		WorkingDir: platformgo.ProcessWorkingDir,
		Now:        time.Now,
	}
}

func normalizeContainStormMode(mode string) string {
	if strings.TrimSpace(strings.ToLower(mode)) == ContainStormProposeOnly {
		return ContainStormProposeOnly
	}
	return ContainStormAutomatic
}

// ProposeOnly reports whether the action must wait for an operator.
func (a *StormAuthority) ProposeOnly() bool {
	return a != nil && a.Mode == ContainStormProposeOnly
}

// AgentScopeName returns the scope unit name of a cgroup path under the
// agent slice, and false for any other path. It is the only test a freeze
// target passes; the name is never matched, the path is.
func AgentScopeName(cgroupPath string) (string, bool) {
	clean := path.Clean("/" + strings.TrimSpace(cgroupPath))
	if !strings.Contains(clean, agentScopeSegment) {
		return "", false
	}
	base := path.Base(clean)
	if !strings.HasPrefix(base, "vrooli-agent-") || !strings.HasSuffix(base, ".scope") {
		return "", false
	}
	return base, true
}

// ScopeRefForPath builds the cgroup scope ref platform-go freezes.
func ScopeRefForPath(cgroupPath string) platformgo.ScopeRef {
	name, _ := AgentScopeName(cgroupPath)
	return platformgo.ScopeRef{Name: strings.TrimSuffix(name, ".scope"), Kind: platformgo.ScopeKindCgroup, Path: path.Clean("/" + strings.TrimSpace(cgroupPath))}
}

// Contain freezes the target's scope. It refuses a target outside the agent
// slice before touching the gate, refuses when the gate cannot be read,
// freezes through platform-go (which refuses again on the path), and records
// the decision either way so a refusal is as durable as a freeze.
func (a *StormAuthority) Contain(ctx context.Context, target StormTarget) (StormOutcome, error) {
	if a == nil {
		return StormOutcome{}, fmt.Errorf("contain-storm: no storm authority is configured")
	}
	now := a.now()
	scopeName, ok := AgentScopeName(target.ScopePath)
	if !ok {
		return StormOutcome{}, fmt.Errorf("contain-storm: refusing %s (pid %d): scope %q is not an agent session scope under vrooli-agents.slice; supervisors and resources are never frozen", target.Name, target.PID, target.ScopePath)
	}
	target.ScopeName = scopeName
	if a.WorkingDir != nil && target.WorkingDir == "" {
		if dir, err := a.WorkingDir(int(target.PID)); err == nil {
			target.WorkingDir = dir
		}
	}
	epochID, allowed, reason := a.gate(ctx)
	ref := ScopeRefForPath(target.ScopePath)
	thaw := "vrooli agent thaw " + strings.TrimSuffix(scopeName, ".scope")
	if !allowed {
		decision := StormDecision{EpochID: epochID, State: StormDecisionRefused, Reason: "runtime recovery gate closed: " + reason, IdempotencyKey: fmt.Sprintf("%s/%s/contain-storm/refused/%d", epochID, scopeName, now.Unix()), Target: target, At: now}
		_ = a.record(ctx, decision)
		return StormOutcome{Scope: ref, Target: target, EpochID: epochID, Decision: decision, ThawCommand: thaw}, fmt.Errorf("contain-storm: refused: %s", reason)
	}
	if err := a.Freeze(ref); err != nil {
		decision := StormDecision{EpochID: epochID, State: StormDecisionRefused, Reason: err.Error(), IdempotencyKey: fmt.Sprintf("%s/%s/contain-storm/refused/%d", epochID, scopeName, now.Unix()), Target: target, At: now}
		_ = a.record(ctx, decision)
		return StormOutcome{Scope: ref, Target: target, EpochID: epochID, Decision: decision, ThawCommand: thaw}, fmt.Errorf("contain-storm: %w", err)
	}
	decision := StormDecision{
		EpochID:        epochID,
		State:          StormDecisionContained,
		Reason:         fmt.Sprintf("%s: froze %s (%s pid %d, %d children) in %s; thaw with `%s`", target.Finding, scopeName, target.Name, target.PID, target.Children, target.WorkingDir, thaw),
		IdempotencyKey: fmt.Sprintf("%s/%s/contain-storm", epochID, scopeName),
		Target:         target,
		At:             now,
	}
	if err := a.record(ctx, decision); err != nil {
		return StormOutcome{Scope: ref, Target: target, EpochID: epochID, Decision: decision, ThawCommand: thaw, FrozenAt: now}, fmt.Errorf("contain-storm: froze %s but could not record the decision: %w", scopeName, err)
	}
	return StormOutcome{Scope: ref, Target: target, EpochID: epochID, Decision: decision, ThawCommand: thaw, FrozenAt: now}, nil
}

func (a *StormAuthority) gate(ctx context.Context) (string, bool, string) {
	if a.Gate == nil {
		return noEpochID, false, "no recovery gate is configured"
	}
	return a.Gate(ctx)
}

func (a *StormAuthority) record(ctx context.Context, decision StormDecision) error {
	if a.Record == nil {
		return fmt.Errorf("no decision recorder is configured")
	}
	return a.Record(ctx, decision)
}

func (a *StormAuthority) now() time.Time {
	if a.Now != nil {
		return a.Now()
	}
	return time.Now()
}

// registryGate reads the runtime registry's pressure epoch. The registry
// unreadable is the closed gate; an open epoch is recorded, not refused: a
// storm is exactly when the supervisor holds an epoch, and freezing the
// culprit complements its restart hold. When no epoch exists the authority
// anchors its decision to one it creates already cleared, so the row has an
// epoch to reference without gating anyone's restarts.
func registryGate(home string) func(context.Context) (string, bool, string) {
	return func(ctx context.Context) (string, bool, string) {
		store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home})
		if err != nil {
			return noEpochID, false, fmt.Sprintf("runtime recovery ownership unavailable: %v", err)
		}
		defer store.Close()
		epochID, err := EnsureStormEpoch(ctx, store)
		if err != nil {
			return noEpochID, false, fmt.Sprintf("runtime recovery ownership unavailable: %v", err)
		}
		return epochID, true, ""
	}
}

// stormEpochSource names the anchor epoch the authority creates when the
// supervisor holds none.
const stormEpochSource = "vrooli-autoheal/contain-storm"

// EnsureStormEpoch returns the latest pressure epoch's id, creating a cleared
// anchor epoch when the registry holds none. A cleared epoch is closed to
// every controller (isOpenPressureEpoch is false), so the anchor never gates
// a restart; it only gives the decision row its foreign key.
func EnsureStormEpoch(ctx context.Context, store scenarioruntime.RecoveryRepository) (string, error) {
	epochs, err := store.ListPressureEpochs(ctx, 1)
	if err != nil {
		return "", err
	}
	if len(epochs) > 0 {
		return epochs[0].EpochID, nil
	}
	now := time.Now().UTC()
	created, err := store.CreatePressureEpoch(ctx, scenarioruntime.PressureEpoch{
		Status: scenarioruntime.PressureEpochCleared, Source: stormEpochSource, DetectedAt: now, ClearedAt: &now,
		DetailsJSON: `{"anchor":"storm authority decision anchor; no supervisor epoch was open"}`,
	})
	if err != nil {
		return "", err
	}
	return created.EpochID, nil
}

// ListStormDecisions reads the authority's rows from the registry under home.
func ListStormDecisions(ctx context.Context, home string, limit int) ([]scenarioruntime.RecoveryDecision, error) {
	store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home})
	if err != nil {
		return nil, err
	}
	defer store.Close()
	return store.ListRecoveryDecisions(ctx, scenarioruntime.RecoveryDecisionFilter{Scenario: stormDecisionScenario, Limit: limit})
}

// registryRecorder writes storm decisions as runtime_recovery_decisions rows
// so `vrooli-autoheal storm status` and the supervisor read one ledger.
func registryRecorder(home string) func(context.Context, StormDecision) error {
	return func(ctx context.Context, decision StormDecision) error {
		store, err := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: home})
		if err != nil {
			return err
		}
		defer store.Close()
		return RecordStormDecision(ctx, store, decision)
	}
}

// RecordStormDecision writes one storm decision through a registry store.
func RecordStormDecision(ctx context.Context, store scenarioruntime.RecoveryRepository, decision StormDecision) error {
	details, _ := json.Marshal(decision.Target)
	_, err := store.RecordRecoveryDecision(ctx, scenarioruntime.RecoveryDecision{
		EpochID:        decision.EpochID,
		Scenario:       stormDecisionScenario,
		State:          decision.State,
		Reason:         decision.Reason,
		IdempotencyKey: decision.IdempotencyKey,
		CreatedAt:      decision.At,
		DetailsJSON:    string(details),
	})
	return err
}
