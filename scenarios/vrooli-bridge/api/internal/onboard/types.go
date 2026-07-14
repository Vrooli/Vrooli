// Package onboard is the domain-scoped home for durable, orchestrated one-shot
// node onboarding (the phase-5 orchestration tier). It sequences the artifacts
// the earlier phases built — SSH first-touch (internal/onboard/ssh, phase 1),
// the server-side pairing code (internal/pairing, phase 3), and the idempotent
// node bootstrap script (scenarios/vrooli-bridge/bootstrap, phase 4) — into one
// server-owned OnboardingOp that drives a raw SSH host to a paired, ONLINE,
// auto-starting fleet agent.
//
// It mirrors internal/provision's durable-op idiom rather than inventing a new
// one: a durable op record + append-only step-event history are the source of
// truth a re-attaching client reads; an in-memory coordinator provides the
// block-once Wait and the live Subscribe fan-out. Where provision pushes a
// privileged command down a channel the node already holds, onboard reaches a
// host that has NO agent yet — so the orchestration itself (SSH → SCP → remote
// bootstrap → verify) runs server-side in a detached goroutine that reports
// progress against the op.
//
// Every outside-world dependency is a narrow seam declared HERE (seams.go) over
// proto-free DTOs, so the domain imports no proto and no sibling handler: the
// handler module is the single translation point to the pairing service (code
// issue), the SSH capability, and the presence/registry read (online confirm).
//
// SECRETS never enter the durable record: the owner's SSH password is
// request-scoped (StartInput carries it once; ssh.FirstTouch zeroes it) and the
// single-use pairing code is issued server-side and injected into the remote
// bootstrap over stdin — never argv, never a persisted field, never a log line.
package onboard

import (
	"fmt"
	"strings"
	"time"
)

// State is an onboarding op's lifecycle state. It mirrors the onboard.proto
// OnboardingState so the domain never imports proto; the handler translates at
// the boundary. PENDING/SSH_SETUP/PUSHING_SCRIPT/BOOTSTRAPPING/VERIFYING are
// non-terminal; SUCCEEDED/FAILED/CANCELLED are terminal.
type State int

const (
	// StateUnspecified is the zero value; a persisted op never holds it.
	StateUnspecified State = 0
	// StatePending — the op record exists; the orchestrator has not begun work.
	StatePending State = 1
	// StateSSHSetup — establishing passwordless SSH (first touch).
	StateSSHSetup State = 2
	// StatePushingScript — copying the bootstrap script to the host (SCP).
	StatePushingScript State = 3
	// StateBootstrapping — running the bootstrap script remotely.
	StateBootstrapping State = 4
	// StateVerifying — bootstrap reported success; confirming the node is ONLINE.
	StateVerifying State = 5
	// StateSucceeded — terminal: the node is paired, ONLINE, and auto-starting.
	StateSucceeded State = 6
	// StateFailed — terminal: onboarding failed (see FailureReason).
	StateFailed State = 7
	// StateCancelled — terminal: the operator cancelled the op.
	StateCancelled State = 8
)

// Terminal reports whether the state is a terminal one. Wait returns once an op
// reaches a terminal state; a late transition on a terminal op is ignored.
func (s State) Terminal() bool {
	switch s {
	case StateSucceeded, StateFailed, StateCancelled:
		return true
	default:
		return false
	}
}

// String renders the state as a short lowercase label for logs/CLI.
func (s State) String() string {
	switch s {
	case StatePending:
		return "pending"
	case StateSSHSetup:
		return "ssh_setup"
	case StatePushingScript:
		return "pushing_script"
	case StateBootstrapping:
		return "bootstrapping"
	case StateVerifying:
		return "verifying"
	case StateSucceeded:
		return "succeeded"
	case StateFailed:
		return "failed"
	case StateCancelled:
		return "cancelled"
	default:
		return "unspecified"
	}
}

// StepStatus is the disposition of one step in an onboarding op. It mirrors the
// bootstrap script's step-start/step-ok/step-skip/step-fail markers and the
// orchestrator's own phase milestones, and the onboard.proto OnboardingStepStatus.
type StepStatus int

const (
	// StepStatusUnspecified is the zero value.
	StepStatusUnspecified StepStatus = 0
	// StepStatusStarted — the step has begun.
	StepStatusStarted StepStatus = 1
	// StepStatusOK — the step completed successfully.
	StepStatusOK StepStatus = 2
	// StepStatusSkipped — the step was skipped (its work was already done).
	StepStatusSkipped StepStatus = 3
	// StepStatusFailed — the step failed.
	StepStatusFailed StepStatus = 4
)

// String renders the step status as a short lowercase label.
func (s StepStatus) String() string {
	switch s {
	case StepStatusStarted:
		return "started"
	case StepStatusOK:
		return "ok"
	case StepStatusSkipped:
		return "skipped"
	case StepStatusFailed:
		return "failed"
	default:
		return "unspecified"
	}
}

// Orchestrator-phase step ids (distinct from the bootstrap script's own step
// ids). They give the persisted trail an entry for each server-side phase the
// orchestrator drives before/around the remote script.
const (
	// StepSSHSetup — the SSH first-touch phase.
	StepSSHSetup = "ssh-setup"
	// StepPushScript — the SCP of the bootstrap script.
	StepPushScript = "push-script"
	// StepRun — the bootstrap run envelope (run-start / run-ok / run-fail).
	StepRun = "run"
	// StepVerifyOnline — the orchestrator's post-run ONLINE confirmation (distinct
	// from the bootstrap's own verify-online step).
	StepVerifyOnline = "verify-online-confirm"
)

// FailureReason is the taxonomy of terminal-failure causes recorded on a FAILED
// op. Stable machine codes phase 6/7 (CLI/UI) can branch on.
type FailureReason string

const (
	// FailureSSHSetup — could not establish passwordless SSH to the host.
	FailureSSHSetup FailureReason = "ssh_setup_failed"
	// FailureScriptPush — could not copy the bootstrap script to the host.
	FailureScriptPush FailureReason = "script_push_failed"
	// FailurePairingIssue — could not issue the server-side pairing code.
	FailurePairingIssue FailureReason = "pairing_issue_failed"
	// FailureBootstrapUsage — the bootstrap script rejected its config (exit 2).
	FailureBootstrapUsage FailureReason = "bootstrap_usage_error"
	// FailureUnsupportedPlatform — the node's platform is unsupported (exit 3).
	FailureUnsupportedPlatform FailureReason = "unsupported_platform"
	// FailurePairing — the pairing code was rejected on redeem (exit 4).
	FailurePairing FailureReason = "pairing_failed"
	// FailureBootstrap — the bootstrap script failed for another reason (exit 1).
	FailureBootstrap FailureReason = "bootstrap_failed"
	// FailureVerifyOnline — the node did not come ONLINE within the budget.
	FailureVerifyOnline FailureReason = "verify_online_failed"
	// FailureInterrupted — the control plane restarted mid-op; the op is safe to
	// retry because every step is idempotent.
	FailureInterrupted FailureReason = "interrupted_by_restart"
	// FailureInternal — an unexpected control-plane-side error.
	FailureInternal FailureReason = "internal_error"
)

// Op is the durable, server-owned record of one StartOnboarding.
type Op struct {
	ID   string
	Host string
	Port int
	User string

	NodeName       string
	TargetRevision string
	RepoURL        string

	State         State
	NodeID        string
	FailureReason FailureReason
	ExitCode      int32

	CreatedAt time.Time
	// StartedAt/FinishedAt are zero until the corresponding transition.
	StartedAt  time.Time
	FinishedAt time.Time
}

// StepEvent is one entry in an op's append-only progress history.
type StepEvent struct {
	OpID      string
	Sequence  uint64
	StepID    string
	Status    StepStatus
	Detail    string
	EmittedAt time.Time
}

// StartInput is what Service.Start accepts. Password is the transient owner SSH
// credential: it is used once (ssh.FirstTouch installs the bridge key with it,
// then zeroes it) and is NEVER copied into the Op, an event, or a log. Pass an
// owned, mutable slice you do not reuse.
type StartInput struct {
	Actor string

	Host     string
	Port     int
	User     string
	Password []byte

	NodeName             string
	TargetRevision       string
	RepoURL              string
	CheckoutDir          string
	ControlPlaneURL      string
	Capabilities         []string
	VerifyTimeoutSeconds int32
	SkipSetup            bool
	SkipPrereqs          bool

	DryRun bool
}

// Decision is the result of a Start: the created op id (empty on a dry-run),
// whether it was a dry-run, and the validated SSH target echoed back.
type Decision struct {
	OpID   string
	DryRun bool
	Host   string
	Port   int
	User   string
}

// ListFilter narrows ListOps. Zero-value fields are not applied.
type ListFilter struct {
	Host  string
	Limit int
}

// ---- Typed error sentinels (translated to Connect codes at the handler) ----

// ErrOpNotFound — no op matches the id.
type ErrOpNotFound struct{ ID string }

func (e ErrOpNotFound) Error() string { return fmt.Sprintf("onboarding op %q not found", e.ID) }

// ErrInvalid — a structural validation failure (empty required field).
type ErrInvalid struct {
	Field  string
	Reason string
}

func (e ErrInvalid) Error() string { return fmt.Sprintf("%s: %s", e.Field, e.Reason) }

// trimField normalises a request token.
func trimField(s string) string { return strings.TrimSpace(s) }
