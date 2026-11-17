package runtimeapp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/artifactlock"
	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/cliinstall"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/coreactivation"
	"github.com/vrooli/vrooli/internal/rebaselineprogress"
	watchdog "github.com/vrooli/vrooli/internal/safeguards/emergency-watchdog"
	supervisor "github.com/vrooli/vrooli/internal/safeguards/runtime-supervisor"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

// A rebaseline makes a candidate whose persistent-state contract differs from
// the serving image the new root control-plane baseline. Automatic activation
// cannot do this: its rollback image could not be relied on to read what the
// candidate writes. It is an operator decision, and it composes the existing
// owners in one governed sequence:
//
//  1. freeze the candidate into a private image, qualify the pair from the
//     actual images (same runtime-home layout, only
//     the state contract differs), build the watchdog from the same checkout
//     and qualify it against the candidate, show the owner-level changes and
//     ask for confirmation;
//  2. retain the serving root image, watchdog image and supervisor unit with
//     checksums outside every rotating fallback slot;
//  3. pause automatic root updates when they are enabled now;
//  4. install the candidate through the atomic root installer;
//  5. publish the qualified watchdog image through its own owner;
//  6. converge and restart the native supervisor onto the installed image
//     (skipped when it already serves it);
//  7. verify the serving image and a fresh, fleet-renewed supervisor lease;
//  8. restore enrollment if this transition paused it; enrollment
//     qualification rechecks the watchdog/root pairing and native readiness.
//
// No caller-writable record grants anything. Every run re-derives its state
// from the installed bytes: while the candidate is not yet serving, the run
// is a fresh, confirmed transition; once it serves, a rerun only finishes
// steps 5-8, each of which is idempotent and verified against live state.

// ErrRebaselineNotRequired signals that ordinary activation can safely handle
// the qualified candidate. The caller must use the normal installer.
var ErrRebaselineNotRequired = errors.New("candidate keeps the serving persistent-state contract")

// RebaselinePlan is what the operator confirms.
type RebaselinePlan struct {
	Continuation      bool
	ServingSHA        string
	CandidateSHA      string
	WatchdogSHA       string
	ServingBuild      string
	ServingContract   string
	CandidateBuild    string
	CandidateContract string
	Changes           []string
	RetainedDir       string
}

// RebaselineRequest names an already-built governed candidate image and the
// checkout it was built from.
type RebaselineRequest struct {
	Candidate string
	Source    string
	// Confirm is the operator decision for a transition that has not yet
	// replaced the serving image. Finishing an installed candidate does not
	// cross the contract change again and is not asked.
	Confirm  func(RebaselinePlan) (bool, error)
	Progress io.Writer
	// RequireTransitionCaller gates all transition and recovery paths after the
	// ordinary same-contract install case has been ruled out.
	RequireTransitionCaller func() error
	// AfterActivation runs while the exact serving root remains fenced. It is
	// intended for bundle companions that must follow this root atomically.
	AfterActivation func(context.Context, RebaselineRecovery) error
	// BeforeEffects verifies external authority for the exact requalified images,
	// including recovery paths that do not ask for a new confirmation.
	BeforeEffects func(context.Context, RebaselineAdmission) error
	// PrepareRecovery binds additional authority data to the durable bundle.
	PrepareRecovery func(context.Context, RebaselineRecovery) error
	// HubDecision selects authenticated Hub recovery; it grants no authority.
	HubDecision bool
	// RequireFreshFinishingDecision is used for legacy progress without a
	// recorded authority mode and no existing Hub decision locator.
	RequireFreshFinishingDecision bool
	// WatchdogSnapshot retains the qualified worker across Hub resumptions.
	WatchdogSnapshot string
	// ExpectedCandidateSHA binds a prepared bundle's root to its manifest.
	ExpectedCandidateSHA string
	// BundleSHA binds progress evidence to the exact prepared bundle, when present.
	BundleSHA string
	// AllowContinuationDecision requests a fresh decision when a paused
	// transition is continued with a different same-contract candidate.
	AllowContinuationDecision bool
}

// RebaselineResult identifies the established baseline.
type RebaselineResult struct {
	Build       string
	Contract    string
	RetainedDir string
	Enrolled    bool
	Finished    bool
}

// RebaselineAdmission is the immutable identity of the images qualified for
// one transition. Recovery callbacks receive it explicitly instead of relying
// on another callback having populated shared state first.
type RebaselineAdmission struct {
	CandidateSHA string
	WatchdogSHA  string
}

// RebaselineRecovery binds admitted images to the retained bundle used to
// finish a prepared transition after interruption.
type RebaselineRecovery struct {
	Admission RebaselineAdmission
	BundleDir string
}

type rebaselineEffects struct {
	newAttempt        func(string, string) *rootAttempt
	guard             func(context.Context, string) error
	guardLocked       func(context.Context, string) error
	lockRoot          func(context.Context, string) (func(), error)
	lockWatchdog      func(context.Context, string) (func(), error)
	validateRole      func(string) error
	imageDigest       func(context.Context, string) (string, error)
	probeIdentity     func(context.Context, string) (*cliv1.CliVersion, error)
	explain           func(context.Context, string, string) []string
	buildWatchdog     func(root, output string) error
	selectWatchdog    func(string, string, *rebaselineprogress.Progress, string, string) (string, string, error)
	probeWorker       func(context.Context, string, *cliv1.CliVersion) error
	enrollment        func(string) (coreactivation.Enrollment, bool, error)
	compareEnrollment func(context.Context, string, coreactivation.Enrollment, bool, bool) error
	installRoot       func(string, string, string) error
	publishWatchdog   func(ctx context.Context, home, image, digest string) error
	converge          func(context.Context, string, string, string) error
	waitServing       func(context.Context, string, string, string) error
	retainBundle      func(string, string, string) (string, error)
	removeBundle      func(string, string) error
	now               func() time.Time
}

// RebaselineRoot runs the governed operator transition on this host.
func RebaselineRoot(ctx context.Context, home string, request RebaselineRequest) (RebaselineResult, error) {
	if runtime.GOOS != "linux" {
		return RebaselineResult{}, fmt.Errorf("root rebaseline is supported only for the Linux native root owner")
	}
	return defaultRebaselineEffects().run(ctx, home, request)
}

func defaultRebaselineEffects() rebaselineEffects {
	retainBundle, removeBundle := preparedBundleRecoveryOperations()
	return rebaselineEffects{
		guard: func(ctx context.Context, home string) error {
			store, err := coreactivation.NewStore(home)
			if err != nil {
				return err
			}
			layout, err := rootUpdateLayout(home)
			if err != nil {
				return err
			}
			release, err := buildinfo.AcquireBinaryInstallLockContext(ctx, layout.Executable)
			if err != nil {
				return err
			}
			defer release()
			return store.GuardOrdinaryMutation(ctx)
		},
		guardLocked: func(ctx context.Context, home string) error {
			store, err := coreactivation.NewStore(home)
			if err != nil {
				return err
			}
			return store.GuardOrdinaryMutation(ctx)
		},
		lockRoot:       buildinfo.AcquireBinaryInstallLockContext,
		lockWatchdog:   watchdog.AcquirePublicationLock,
		validateRole:   coreactivation.ValidateRootExecutableRole,
		imageDigest:    coreactivation.ImageDigest,
		probeIdentity:  coreactivation.ProbeRootIdentity,
		explain:        coreactivation.ExplainStateContractChange,
		buildWatchdog:  watchdog.BuildImage,
		selectWatchdog: selectPreparedRebaselineWatchdog,
		probeWorker: func(ctx context.Context, image string, root *cliv1.CliVersion) error {
			_, err := coreactivation.ProbeWorkerIdentity(ctx, image, root)
			return err
		},
		enrollment: func(home string) (coreactivation.Enrollment, bool, error) {
			store, err := coreactivation.NewStore(home)
			if err != nil {
				return coreactivation.Enrollment{}, false, err
			}
			return store.Enrollment()
		},
		compareEnrollment: func(ctx context.Context, home string, expected coreactivation.Enrollment, enabled, paused bool) error {
			_, err := configureRootUpdates(ctx, home, enabled, paused, &expected)
			return err
		},
		installRoot:     cliinstall.AtomicInstallRebaselineLocked,
		retainBundle:    retainBundle,
		removeBundle:    removeBundle,
		publishWatchdog: watchdog.PublishRebaselineImageLocked,
		converge: func(ctx context.Context, home, source, executable string) error {
			_, err := supervisor.ConvergeRebaselineLocked(ctx, supervisor.ConvergeOptions{OS: runtime.GOOS, Home: home, Root: source, Executable: executable, Restart: true})
			return err
		},
		waitServing: supervisor.WaitServing,
		now:         time.Now,
	}
}

func (e rebaselineEffects) run(ctx context.Context, home string, request RebaselineRequest) (RebaselineResult, error) {
	var result RebaselineResult
	originalCandidate := request.Candidate
	newAttempt := e.newAttempt
	if newAttempt == nil {
		newAttempt = beginRootAttempt
	}
	attempt := newAttempt(home, "rebaseline")
	attempt.candidateFallback = originalCandidate
	attempt.Attempt.Async()
	outcome := "failed"
	defer func() { attempt.finish(ctx, outcome) }()
	if ctx == nil || request.Confirm == nil || !filepath.IsAbs(request.Candidate) || filepath.Clean(request.Candidate) != request.Candidate || !filepath.IsAbs(request.Source) || filepath.Clean(request.Source) != request.Source {
		outcome = "invalid_request"
		return result, fmt.Errorf("root rebaseline requires a context, an operator confirmation and canonical candidate/source paths")
	}
	if request.HubDecision && request.BeforeEffects == nil {
		return result, fmt.Errorf("Hub recovery requires authenticated authority verification")
	}
	if request.HubDecision && request.BundleSHA != "" && request.PrepareRecovery == nil {
		return result, fmt.Errorf("Hub recovery requires a durable authority locator")
	}
	confirm := request.Confirm
	request.Confirm = func(plan RebaselinePlan) (bool, error) {
		attempt.identities(plan.ServingContract, plan.CandidateContract)
		attempt.event("confirmation", "started")
		ok, err := confirm(plan)
		switch {
		case err != nil:
			attempt.event("confirmation", "failed")
		case ok:
			attempt.event("confirmation", "confirmed")
		default:
			outcome = "declined"
			attempt.event("confirmation", "declined")
		}
		return ok, err
	}
	progress := request.Progress
	if progress == nil {
		progress = io.Discard
	}
	layout, err := rootUpdateLayout(home)
	if err != nil {
		return result, err
	}
	statePaths, err := preparedRebaselineStatePathsForHome(home)
	if err != nil {
		return result, err
	}
	transitions := statePaths.transitions
	releaseBundle, err := buildinfo.AcquireRootBundleWorkflow(ctx, home)
	if err != nil {
		return result, fmt.Errorf("lock root bundle workflow: %w", err)
	}
	defer releaseBundle()
	if err := buildinfo.GuardPreparedBundle(home, ""); err != nil {
		return result, err
	}
	release, err := artifactlock.AcquireWithPathContext(ctx, "root-rebaseline:"+home, func(string) (string, error) { return statePaths.lock, nil })
	if err != nil {
		return result, fmt.Errorf("lock root rebaseline: %w", err)
	}
	defer release()
	if err := e.guard(ctx, home); err != nil {
		return result, err
	}
	// The caller pins the original candidate to its source checkout. Freeze
	// its bytes before qualification: another build may replace that path
	// while the operator reads the prompt. All subsequent probes, explanations
	// and installation use only this run's private copy.
	candidateSHA, err := e.imageDigest(ctx, request.Candidate)
	if err != nil {
		return result, fmt.Errorf("observe candidate root CLI: %w", err)
	}
	if request.ExpectedCandidateSHA != "" && candidateSHA != request.ExpectedCandidateSHA {
		return result, fmt.Errorf("candidate root differs from prepared bundle: expected %s, observed %s", request.ExpectedCandidateSHA, candidateSHA)
	}
	if _, err := config.EnsureOwnedDir(transitions); err != nil {
		return result, fmt.Errorf("prepare private rebaseline directory: %w", err)
	}
	work, err := os.MkdirTemp(transitions, "work-")
	if err != nil {
		return result, err
	}
	defer os.RemoveAll(work)
	if err := config.ChownToInvokingUser(work); err != nil {
		return result, fmt.Errorf("own private rebaseline directory: %w", err)
	}
	stagedCandidate := filepath.Join(work, "vrooli")
	copiedSHA, err := retainRebaselineFile(request.Candidate, stagedCandidate)
	if err != nil {
		return result, fmt.Errorf("snapshot candidate root CLI: %w", err)
	}
	if copiedSHA != candidateSHA {
		return result, fmt.Errorf("candidate root CLI changed while preparing its private image: expected SHA-256 %s, observed %s; rebuild and retry", candidateSHA, copiedSHA)
	}
	request.Candidate = stagedCandidate
	servingSHA, err := e.imageDigest(ctx, layout.Executable)
	if err != nil {
		return result, fmt.Errorf("observe serving root CLI: %w", err)
	}
	// Every refusal from here on reports the live enrollment state, so no
	// message promises automatic rollback while enrollment is paused.
	refuse := func(err error) (RebaselineResult, error) {
		return result, fmt.Errorf("%w\n%s", err, e.enrollmentState(home))
	}
	if err := e.validateRole(request.Candidate); err != nil {
		return refuse(fmt.Errorf("candidate is not a governed root CLI image: %w", err))
	}
	if err := e.verifyPrivateCandidate(ctx, request.Candidate, candidateSHA); err != nil {
		return refuse(err)
	}
	candidate, err := e.probeIdentity(ctx, request.Candidate)
	if err != nil {
		return refuse(fmt.Errorf("candidate root CLI identity: %w", err))
	}
	if err := e.verifyPrivateCandidate(ctx, request.Candidate, candidateSHA); err != nil {
		return refuse(err)
	}
	fail := func(step string, cause error, recoveryBundle, retained string, hubLocatorBound, recoveryProgressWritten bool) error {
		where := ""
		if retained != "" {
			where = fmt.Sprintf("The prior baseline is retained in %s (receipt.json lists its checksums). ", retained)
		}
		run := preparedRebaselineResumeCommand(request, originalCandidate, recoveryBundle, hubLocatorBound, recoveryProgressWritten)
		return fmt.Errorf("root rebaseline stopped at %q: %w\n%s%s Fix the cause and rerun `%s`; it re-qualifies from the installed images", step, cause, where, e.enrollmentState(home), run)
	}
	result, err = runPreparedRebaseline(preparedRebaselineRunInput{
		ctx: ctx, effects: e, home: home,
		paths: preparedRebaselinePaths{
			work: work, originalCandidate: originalCandidate,
		},
		layout: layout, request: request, candidate: candidate, candidateSHA: candidateSHA, servingSHA: servingSHA,
		attempt: attempt, progress: progress,
		telemetry: preparedRebaselineTelemetry{
			attemptEvent:    attempt.event,
			progressMessage: func(message string) { _, _ = fmt.Fprintln(progress, message) },
			stepMessage:     func(name string) { _, _ = fmt.Fprintf(progress, "-> %s\n", name) },
			retirementWarn: func(err error) {
				_, _ = fmt.Fprintf(progress, "Warning: completed recovery bundle could not be retired: %v\n", err)
			},
			retentionCompleted: func(retained string) {
				attempt.serving = filepath.Join(retained, "vrooli")
				attempt.event("retention", "completed")
			},
		},
		refuse: func(err error) error { _, wrapped := refuse(err); return wrapped },
		fail:   fail,
	})
	if err != nil {
		return result, err
	}
	if result.RetainedDir != "" {
		// The verified installed candidate is available after private staging
		// cleanup; manifest identity checks still reject a later replacement.
		attempt.candidate = layout.Executable
	}
	outcome = "completed"
	return result, nil
}

func preparedRebaselineResumeCommand(request RebaselineRequest, originalCandidate, recoveryBundle string, hubLocatorBound, recoveryProgressWritten bool) string {
	if request.HubDecision {
		if recoveryBundle != "" && hubLocatorBound && recoveryProgressWritten {
			candidate := "'" + strings.ReplaceAll(filepath.Join(recoveryBundle, "vrooli"), "'", "'\\''") + "'"
			return fmt.Sprintf("%s runtime update rebaseline %s --hub", candidate, candidate)
		}
		return fmt.Sprintf("HUB_SNAPSHOT='%s' make rebaseline-hub-resume", strings.ReplaceAll(filepath.Dir(originalCandidate), "'", "'\\''"))
	}
	if request.AfterActivation == nil {
		return "make rebaseline"
	}
	candidatePath := originalCandidate
	if recoveryBundle != "" && recoveryProgressWritten {
		candidatePath = filepath.Join(recoveryBundle, "vrooli")
	}
	candidate := "'" + strings.ReplaceAll(candidatePath, "'", "'\\''") + "'"
	return fmt.Sprintf("%s runtime update rebaseline %s --prepared", candidate, candidate)
}

func (e rebaselineEffects) verifyPrivateCandidate(ctx context.Context, path, expected string) error {
	actual, err := e.imageDigest(ctx, path)
	if err != nil {
		return fmt.Errorf("verify private candidate image digest: %w", err)
	}
	if actual != expected {
		return fmt.Errorf("private candidate image changed: expected SHA-256 %s, observed %s", expected, actual)
	}
	return nil
}

// enrollmentState is the live enrollment sentence every stop or refusal ends
// with.
func (e rebaselineEffects) enrollmentState(home string) string {
	enrollment, exists, err := e.enrollment(home)
	switch {
	case err != nil:
		return "Automatic root update enrollment could not be read: " + err.Error() + "."
	case exists && enrollment.Enabled:
		return "Automatic root updates are enabled."
	case e.pauseInEffect(home):
		return "Automatic root updates are paused by an unfinished rebaseline; resume the prepared rebaseline before installing another bundle."
	default:
		return "Automatic root updates are not enrolled."
	}
}
