// Package privsep is the node-agent's PRIVILEGED provisioning helper (OT-P0-006
// node side). It is STRUCTURALLY SEPARATE from the non-privileged job runner
// (package exec): the two never import each other, and only the channel's
// ProvisionCommand handler invokes this package. The everyday runner therefore
// has no path to provisioning — remote job execution can never escalate.
// (DECISIONS.md "two trust tiers": a distinct OS principal runs the privileged
// helper; in deployment the service-install adapter runs it as the dedicated
// provisioning user, see package service.)
//
// Provisioning brings the node to a target git revision: `git fetch` + checkout
// R, then an idempotent `vrooli setup`. It is built from typed argvs (never a
// shell string), is idempotent (re-provisioning a node already at R is safe —
// `vrooli setup` converges), and rolls back to the prior revision when setup
// fails so a bad revision never leaves the node stranded. Progress, the
// resulting version, and the terminal exit stream back to the control plane as
// ProvisionEvents.
package privsep

import (
	"context"
	"fmt"
	"os"
	osuser "os/user"
	"strconv"
	"strings"
	"time"

	channelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/channel"
	provisionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/provision"

	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	// setupFailExitCode is reported when `vrooli setup` fails and the node could
	// not be rolled back (degraded). startFailureExitCode is reported when a step
	// cannot start at all.
	setupFailExitCode    = 1
	startFailureExitCode = 127
)

// shellMetachars mirrors the runner's defence-in-depth check: a provisioning
// step is always a typed argv, so a revision token carrying a shell
// metacharacter can only be smuggling and is rejected before execution. (The
// control plane validates revisions too; the two are intentionally duplicated
// because the agent is a separate Go module.)
const shellMetachars = "|&;<>()$`\\\"'\n\r\t*?[]{}!#~ "

// StepRunner executes one provisioning step's argv in dir, streaming combined
// stdout/stderr to onLog line-by-line, and returns the step's exit code. It
// NEVER receives a shell string — only a pre-split argv. Production wires
// osStepRunner (step_runner.go), which runs the argv as the PRIVILEGED helper;
// tests substitute a fake.
type StepRunner interface {
	Run(ctx context.Context, argv []string, dir string, onLog func(string)) (exitCode int, err error)
}

// RevisionResolver reports the node's current checked-out git revision (the
// resulting version reported back to the control plane). Production wires a
// `git rev-parse HEAD` resolver; tests substitute a fake.
type RevisionResolver interface {
	Current(ctx context.Context, dir string) (string, error)
}

// Reporter delivers one ProvisionEvent back to the control plane. The channel
// package wires the production implementation (a signed ProvisionService
// ReportProvisionEvent call); tests substitute a collector.
type Reporter interface {
	Report(ctx context.Context, ev *provisionv1.ProvisionEvent) error
}

// Helper executes privileged provisioning ops and reports their progress. It is
// the privileged-tier counterpart to exec.Runner and shares no type with it.
type Helper struct {
	gitBin               string
	vrooliBin            string
	workDir              string
	step                 StepRunner
	revision             RevisionResolver
	reporter             Reporter
	sealingSeedPath      string
	clientUID            int
	clientHome           string
	deferredServiceNames []string
	cleanupWorkDir       string
	now                  func() time.Time
}

// Option customises a Helper.
type Option func(*Helper)

// WithClock overrides the time source (tests).
func WithClock(now func() time.Time) Option { return func(h *Helper) { h.now = now } }

// WithStepRunner overrides the privileged step-execution seam (tests).
func WithStepRunner(s StepRunner) Option { return func(h *Helper) { h.step = s } }

// WithRevisionResolver overrides the git-revision resolver (tests).
func WithRevisionResolver(r RevisionResolver) Option { return func(h *Helper) { h.revision = r } }

// WithGitBin overrides the git binary name/path.
func WithGitBin(bin string) Option { return func(h *Helper) { h.gitBin = strings.TrimSpace(bin) } }

// WithSealingSeedPath points at the node's independent X25519 private key.
// The raw key is read only for the duration of opening an operator envelope.
func WithSealingSeedPath(path string) Option {
	return func(h *Helper) { h.sealingSeedPath = strings.TrimSpace(path) }
}

// WithClientUID identifies the unprivileged runner whose user-scoped Vrooli
// state the privileged helper must operate on.
func WithClientUID(uid int) Option { return func(h *Helper) { h.clientUID = uid } }

// WithClientHome supplies the runner home when the transport already resolved
// it (for example, the fixed SSH cleanup command expands $HOME before sudo).
func WithClientHome(home string) Option {
	return func(h *Helper) { h.clientHome = strings.TrimSpace(home) }
}

// WithDeferredServiceNames marks native services whose unit files may be
// removed by a cleanup command while the paired helper/agent is still carrying
// the command's reporting path. The unit files are removed immediately; the
// transport shuts the processes down only after the terminal receipt is sent.
func WithDeferredServiceNames(names ...string) Option {
	return func(h *Helper) {
		for _, name := range names {
			if name = strings.TrimSpace(name); name != "" {
				h.deferredServiceNames = append(h.deferredServiceNames, name)
			}
		}
	}
}

// WithCleanupWorkDir overrides the directory from which the node-local CLI is
// launched during cleanup. Production defaults to the platform temporary
// directory because a frozen plan may remove the checkout in workDir; a child
// whose current directory is deleted can otherwise lose its ability to finish
// the receipt/reporting path. Tests use this seam to assert that contract.
func WithCleanupWorkDir(dir string) Option {
	return func(h *Helper) {
		if dir = strings.TrimSpace(dir); dir != "" {
			h.cleanupWorkDir = dir
		}
	}
}

func (h *Helper) resolvedClientHome() string {
	if home := strings.TrimSpace(h.clientHome); home != "" {
		return home
	}
	if h.clientUID < 0 {
		return ""
	}
	current, err := osuser.LookupId(strconv.Itoa(h.clientUID))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(current.HomeDir)
}

// NewHelper constructs a Helper. vrooliBin is the local vrooli CLI (default
// "vrooli"), workDir the node's Vrooli checkout, reporter the event sink. The
// production StepRunner/RevisionResolver default to the os-backed
// implementations (step_runner.go).
func NewHelper(vrooliBin, workDir string, reporter Reporter, opts ...Option) *Helper {
	h := &Helper{
		gitBin:         "git",
		vrooliBin:      strings.TrimSpace(vrooliBin),
		workDir:        workDir,
		clientUID:      -1,
		cleanupWorkDir: os.TempDir(),
		reporter:       reporter,
		now:            time.Now,
	}
	if h.vrooliBin == "" {
		h.vrooliBin = "vrooli"
	}
	h.step = osStepRunner{}
	h.revision = osRevisionResolver{gitBin: h.gitBin}
	for _, opt := range opts {
		opt(h)
	}
	// Keep the resolver's git binary in lockstep with WithGitBin overrides when
	// the default resolver is in use.
	if r, ok := h.revision.(osRevisionResolver); ok {
		r.gitBin = h.gitBin
		h.revision = r
	}
	return h
}

// Steps is the typed, ordered argv plan that brings the node to a target
// revision. It is pure and exported so the no-shell-path property and the exact
// command sequence are directly testable. Returns an error (not steps) when the
// target is empty or any revision token carries a shell metacharacter.
//
//	[git, fetch, --all, --tags]
//	[git, checkout, <target>]
//	[vrooli, setup]
func Steps(gitBin, vrooliBin, target string) ([][]string, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return nil, fmt.Errorf("target revision is required")
	}
	if i := strings.IndexAny(target, shellMetachars); i >= 0 {
		return nil, fmt.Errorf("unsafe revision %q: contains disallowed character %q", target, string(target[i]))
	}
	return [][]string{
		{gitBin, "fetch", "--all", "--tags"},
		{gitBin, "checkout", target},
		{vrooliBin, "setup"},
	}, nil
}

// Provision brings the node to cmd.TargetRevision, streaming ProvisionEvents
// back. It never returns the op's failure as a Go error — a failed setup is
// reported via the terminal EXIT event (and a rollback attempt), not the return
// value; the return error is reserved for a reporter transport failure the
// caller may log.
func (h *Helper) Provision(ctx context.Context, cmd *channelv1.ProvisionCommand) error {
	var seq uint64
	emit := func(ev *provisionv1.ProvisionEvent) error {
		seq++
		ev.OpId = cmd.GetOpId()
		ev.Sequence = seq
		ev.EmittedAt = timestamppb.New(h.now().UTC())
		return h.reporter.Report(ctx, ev)
	}

	steps, err := Steps(h.gitBin, h.vrooliBin, cmd.GetTargetRevision())
	if err != nil {
		_ = emit(statusEvent("rejected: " + err.Error()))
		return emit(exitEvent(setupFailExitCode))
	}

	if err := emit(statusEvent("provisioning to " + cmd.GetTargetRevision())); err != nil {
		return err
	}

	// Run fetch + checkout + setup. A non-zero exit on any step (or a start
	// failure) triggers the rollback path.
	exitCode, runErr := h.runSteps(ctx, steps, emit)
	if runErr == nil && exitCode == 0 {
		return h.finishSuccess(ctx, emit)
	}

	// Setup (or an earlier step) failed. Attempt a rollback to the prior
	// revision so a bad target never strands the node.
	return h.rollback(ctx, cmd, exitCode, runErr, emit)
}

// runSteps executes the plan in order, emitting a STATUS for each step and LOG
// events for its output. It stops at the first non-zero/erroring step and
// returns that step's exit code.
func (h *Helper) runSteps(ctx context.Context, steps [][]string, emit func(*provisionv1.ProvisionEvent) error) (int, error) {
	for _, argv := range steps {
		_ = emit(statusEvent("step: " + strings.Join(argv, " ")))
		code, err := h.step.Run(ctx, argv, h.workDir, func(chunk string) { _ = emit(logEvent(chunk)) })
		if err != nil {
			_ = emit(statusEvent("error: " + err.Error()))
			if code == 0 {
				code = startFailureExitCode
			}
			return code, err
		}
		if code != 0 {
			return code, nil
		}
	}
	return 0, nil
}

// finishSuccess resolves and reports the node's resulting revision, then emits a
// clean terminal EXIT(0) — the COMPLETED outcome.
func (h *Helper) finishSuccess(ctx context.Context, emit func(*provisionv1.ProvisionEvent) error) error {
	if rev, err := h.revision.Current(ctx, h.workDir); err == nil && strings.TrimSpace(rev) != "" {
		_ = emit(versionEvent(strings.TrimSpace(rev)))
	}
	_ = emit(statusEvent("completed"))
	return emit(exitEvent(0))
}

// rollback returns the node to the rollback revision and re-runs setup. On a
// successful rollback it reports the rollback revision as the resulting version
// and a terminal EXIT carrying the ORIGINAL failing code (so the control plane
// records ROLLED_BACK, the safe failure). With no rollback revision — or a
// rollback that itself fails — it reports the current revision and a terminal
// EXIT (FAILED / degraded).
func (h *Helper) rollback(ctx context.Context, cmd *channelv1.ProvisionCommand, failCode int, _ error, emit func(*provisionv1.ProvisionEvent) error) error {
	if failCode == 0 {
		failCode = setupFailExitCode
	}
	rollbackRev := strings.TrimSpace(cmd.GetRollbackRevision())
	if rollbackRev == "" {
		// First provision (nothing to roll back to): degraded failure. Report the
		// node's current revision for visibility.
		if rev, err := h.revision.Current(ctx, h.workDir); err == nil && strings.TrimSpace(rev) != "" {
			_ = emit(versionEvent(strings.TrimSpace(rev)))
		}
		_ = emit(statusEvent("failed: setup failed and no rollback revision is available"))
		return emit(exitEvent(failCode))
	}

	_ = emit(statusEvent("rolling back to " + rollbackRev))
	steps, err := Steps(h.gitBin, h.vrooliBin, rollbackRev)
	if err != nil {
		_ = emit(statusEvent("failed: rollback plan invalid: " + err.Error()))
		return emit(exitEvent(failCode))
	}
	// The rollback re-runs the same checkout+setup plan against the prior
	// revision. If even the rollback setup fails, the node is degraded.
	if code, rerr := h.runSteps(ctx, steps, emit); rerr != nil || code != 0 {
		_ = emit(statusEvent("failed: rollback did not restore the node"))
		return emit(exitEvent(failCode))
	}
	if rev, err := h.revision.Current(ctx, h.workDir); err == nil && strings.TrimSpace(rev) != "" {
		_ = emit(versionEvent(strings.TrimSpace(rev)))
	} else {
		// Fall back to the requested rollback revision when HEAD cannot be read.
		_ = emit(versionEvent(rollbackRev))
	}
	_ = emit(statusEvent("rolled_back"))
	return emit(exitEvent(failCode))
}

func statusEvent(status string) *provisionv1.ProvisionEvent {
	return &provisionv1.ProvisionEvent{Kind: provisionv1.ProvisionEventKind_PROVISION_EVENT_KIND_STATUS, Status: status}
}

func logEvent(chunk string) *provisionv1.ProvisionEvent {
	return &provisionv1.ProvisionEvent{Kind: provisionv1.ProvisionEventKind_PROVISION_EVENT_KIND_LOG, LogChunk: chunk}
}

func versionEvent(rev string) *provisionv1.ProvisionEvent {
	return &provisionv1.ProvisionEvent{Kind: provisionv1.ProvisionEventKind_PROVISION_EVENT_KIND_VERSION, Revision: rev}
}

func exitEvent(code int) *provisionv1.ProvisionEvent {
	return &provisionv1.ProvisionEvent{Kind: provisionv1.ProvisionEventKind_PROVISION_EVENT_KIND_EXIT, ExitCode: int32(code)} // #nosec G115 -- all production exit codes are bounded OS/helper statuses.
}
