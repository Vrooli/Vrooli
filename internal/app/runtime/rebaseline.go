package runtimeapp

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/cliinstall"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/coreactivation"
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

const (
	rebaselineReceiptSchemaVersion = 1
	rebaselinePauseSchemaVersion   = 1
)

// RebaselinePlan is what the operator confirms.
type RebaselinePlan struct {
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
}

// RebaselineResult identifies the established baseline.
type RebaselineResult struct {
	Build       string
	Contract    string
	RetainedDir string
	Enrolled    bool
	Finished    bool
}

// rebaselineReceipt is audit evidence retained beside the prior images. It is
// never read back as authority.
type rebaselineReceipt struct {
	SchemaVersion     int      `json:"schema_version"`
	ID                string   `json:"id"`
	CreatedUnixNano   int64    `json:"created_unix_nano"`
	Source            string   `json:"source"`
	ServingSHA        string   `json:"serving_sha256"`
	ServingBuild      string   `json:"serving_build_identity"`
	ServingContract   string   `json:"serving_state_contract"`
	WatchdogSHA       string   `json:"watchdog_sha256,omitempty"`
	UnitSHA           string   `json:"supervisor_unit_sha256,omitempty"`
	CandidateSHA      string   `json:"candidate_sha256"`
	CandidateBuild    string   `json:"candidate_build_identity"`
	CandidateContract string   `json:"candidate_state_contract"`
	Changes           []string `json:"changes"`
}

// rebaselinePause records that a rebaseline paused enrollment. It is in
// effect only while the enrollment record is exactly the one that pause wrote
// (same update time, still disabled): an operator who enables or disables
// enrollment afterwards supersedes it. The next successful rebaseline of any
// qualified candidate restores enrollment through enrollment's own
// qualification and clears it.
type rebaselinePause struct {
	SchemaVersion             int    `json:"schema_version"`
	PausedUnixNano            int64  `json:"paused_unix_nano"`
	EnrollmentUpdatedUnixNano int64  `json:"enrollment_updated_unix_nano"`
	PriorServingSHA           string `json:"prior_serving_sha256"`
}

type rebaselineEffects struct {
	guard           func(context.Context, string) error
	validateRole    func(string) error
	imageDigest     func(context.Context, string) (string, error)
	probeIdentity   func(context.Context, string) (*cliv1.CliVersion, error)
	explain         func(context.Context, string, string) []string
	buildWatchdog   func(root, output string) error
	probeWorker     func(context.Context, string, *cliv1.CliVersion) error
	enrollment      func(string) (coreactivation.Enrollment, bool, error)
	configure       func(context.Context, string, bool) error
	installRoot     func(string, string) error
	publishWatchdog func(ctx context.Context, home, image, digest string) error
	converge        func(context.Context, string, string, string) error
	waitServing     func(context.Context, string, string, string) error
	now             func() time.Time
}

// RebaselineRoot runs the governed operator transition on this host.
func RebaselineRoot(ctx context.Context, home string, request RebaselineRequest) (RebaselineResult, error) {
	if runtime.GOOS != "linux" {
		return RebaselineResult{}, fmt.Errorf("root rebaseline is supported only for the Linux native root owner")
	}
	return defaultRebaselineEffects().run(ctx, home, request)
}

func defaultRebaselineEffects() rebaselineEffects {
	return rebaselineEffects{
		guard: func(ctx context.Context, home string) error {
			store, err := coreactivation.NewStore(home)
			if err != nil {
				return err
			}
			return store.GuardOrdinaryMutation(ctx)
		},
		validateRole:  coreactivation.ValidateRootExecutableRole,
		imageDigest:   coreactivation.ImageDigest,
		probeIdentity: coreactivation.ProbeRootIdentity,
		explain:       coreactivation.ExplainStateContractChange,
		buildWatchdog: watchdog.BuildImage,
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
		configure: func(ctx context.Context, home string, enabled bool) error {
			_, err := ConfigureRootUpdates(ctx, home, enabled)
			return err
		},
		installRoot:     cliinstall.AtomicInstall,
		publishWatchdog: watchdog.PublishImage,
		converge: func(ctx context.Context, home, source, executable string) error {
			_, err := supervisor.Converge(ctx, supervisor.ConvergeOptions{OS: runtime.GOOS, Home: home, Root: source, Executable: executable, Restart: true})
			return err
		},
		waitServing: supervisor.WaitServing,
		now:         time.Now,
	}
}

func (e rebaselineEffects) run(ctx context.Context, home string, request RebaselineRequest) (RebaselineResult, error) {
	var result RebaselineResult
	if ctx == nil || request.Confirm == nil || !filepath.IsAbs(request.Candidate) || filepath.Clean(request.Candidate) != request.Candidate || !filepath.IsAbs(request.Source) || filepath.Clean(request.Source) != request.Source {
		return result, fmt.Errorf("root rebaseline requires a context, an operator confirmation and canonical candidate/source paths")
	}
	progress := request.Progress
	if progress == nil {
		progress = io.Discard
	}
	layout, err := rootUpdateLayout(home)
	if err != nil {
		return result, err
	}
	stateDir, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyState)
	if err != nil {
		return result, err
	}
	transitions := filepath.Join(stateDir, "root-rebaseline")
	pausePath := filepath.Join(transitions, "pause.json")
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
	work, err := os.MkdirTemp(transitionsDir(transitions), "work-")
	if err != nil {
		return result, err
	}
	defer os.RemoveAll(work)
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
		return result, fmt.Errorf("%w\n%s", err, e.enrollmentState(home, pausePath))
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
	finishing := servingSHA == candidateSHA
	// Build the watchdog from the same checkout as the candidate and qualify
	// the pair before anything changes. The exact qualified bytes, bound by
	// digest, are published after the root; the live tree is not consulted
	// again.
	stagedWatchdog := filepath.Join(work, "vrooli-watchdog")
	if err := e.buildWatchdog(request.Source, stagedWatchdog); err != nil {
		return refuse(fmt.Errorf("build the watchdog for this candidate: %w", err))
	}
	if err := e.probeWorker(ctx, stagedWatchdog, candidate); err != nil {
		return refuse(fmt.Errorf("the watchdog built from %s does not match the candidate's contract (rebuild the candidate from the same checkout): %w", request.Source, err))
	}
	watchdogSHA, err := e.imageDigest(ctx, stagedWatchdog)
	if err != nil {
		return refuse(err)
	}
	retained := ""
	switch {
	case finishing:
		_, _ = fmt.Fprintln(progress, "The candidate already serves; finishing its rebaseline.")
	case e.pauseInEffect(home, pausePath):
		// An unfinished rebaseline paused enrollment and installed another
		// build. A candidate with the serving contract continues it without
		// crossing a contract change; a different contract is a new,
		// confirmed transition.
		retained, err = e.qualifyAndRetain(ctx, home, layout, transitions, request, servingSHA, candidateSHA, candidate, true)
	default:
		retained, err = e.qualifyAndRetain(ctx, home, layout, transitions, request, servingSHA, candidateSHA, candidate, false)
	}
	if err != nil {
		return refuse(err)
	}
	if err := e.verifyPrivateCandidate(ctx, request.Candidate, candidateSHA); err != nil {
		return refuse(err)
	}
	fail := func(step string, cause error) (RebaselineResult, error) {
		where := ""
		if retained != "" {
			where = fmt.Sprintf("The prior baseline is retained in %s (receipt.json lists its checksums). ", retained)
		}
		return result, fmt.Errorf("root rebaseline stopped at %q: %w\n%s%s Fix the cause and rerun `make rebaseline`; it re-qualifies from the installed images", step, cause, where, e.enrollmentState(home, pausePath))
	}
	steps := []struct {
		name string
		run  func() error
	}{
		{"pause automatic root updates", func() error {
			if finishing {
				return nil
			}
			return e.pause(ctx, home, pausePath, servingSHA)
		}},
		{"install the candidate root CLI", func() error {
			if finishing {
				return nil
			}
			if err := e.verifyPrivateCandidate(ctx, request.Candidate, candidateSHA); err != nil {
				return err
			}
			if err := e.installRoot(request.Candidate, layout.Executable); err != nil {
				return err
			}
			installed, err := e.imageDigest(ctx, layout.Executable)
			if err != nil {
				return fmt.Errorf("verify installed root CLI digest: %w", err)
			}
			if installed != candidateSHA {
				return fmt.Errorf("installed root CLI is not the qualified candidate image: expected SHA-256 %s, observed %s", candidateSHA, installed)
			}
			return nil
		}},
		{"publish the watchdog built for this candidate", func() error {
			return e.publishWatchdog(ctx, home, stagedWatchdog, watchdogSHA)
		}},
		{"restart the runtime supervisor on the new image", func() error {
			quick, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()
			if e.waitServing(quick, home, candidateSHA, candidate.GetBuildIdentity()) == nil {
				return nil // already serving the candidate with a fresh lease
			}
			return e.converge(ctx, home, request.Source, layout.Executable)
		}},
		{"verify the serving image and a fresh supervisor lease", func() error {
			verify, cancel := context.WithTimeout(ctx, 3*time.Minute)
			defer cancel()
			return e.waitServing(verify, home, candidateSHA, candidate.GetBuildIdentity())
		}},
		{"restore automatic root updates", func() error {
			return e.restore(ctx, home, pausePath, progress)
		}},
	}
	for _, step := range steps {
		_, _ = fmt.Fprintf(progress, "-> %s\n", step.name)
		if err := ctx.Err(); err != nil {
			return fail(step.name, err)
		}
		if err := step.run(); err != nil {
			return fail(step.name, err)
		}
	}
	if enrollment, exists, err := e.enrollment(home); err == nil {
		result.Enrolled = exists && enrollment.Enabled
	}
	result.Build, result.Contract, result.RetainedDir, result.Finished = candidate.GetBuildIdentity(), candidate.GetPersistentStateContractSha256(), retained, finishing
	return result, nil
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

func transitionsDir(path string) string {
	if _, err := config.EnsureOwnedDir(path); err != nil {
		return os.TempDir()
	}
	return path
}

// qualifyAndRetain qualifies the actual images, obtains the operator
// decision and retains the prior baseline. Nothing on the serving path
// changes before it returns.
func (e rebaselineEffects) qualifyAndRetain(ctx context.Context, home string, layout coreactivation.Layout, transitions string, request RebaselineRequest, servingSHA, candidateSHA string, candidate *cliv1.CliVersion, continuing bool) (string, error) {
	serving, err := e.probeIdentity(ctx, layout.Executable)
	if err != nil {
		return "", fmt.Errorf("serving root CLI identity: %w", err)
	}
	qualification := coreactivation.QualifyRollbackIdentities(serving, candidate)
	if qualification == nil && continuing {
		// Same contract as the unfinished rebaseline's installed build: no
		// contract change is crossed, so the operator is not asked again.
		return "", nil
	}
	if qualification == nil {
		if enrollment, exists, err := e.enrollment(home); err == nil && exists && enrollment.Enabled {
			return "", fmt.Errorf("the candidate keeps the serving persistent-state contract; activate it with `make install`, which keeps automatic rollback")
		}
		return "", fmt.Errorf("the candidate keeps the serving persistent-state contract; `make install` installs it directly because automatic root updates are not enrolled")
	}
	if !errors.Is(qualification, coreactivation.ErrStateContractChanged) {
		return "", fmt.Errorf("root rebaseline only crosses persistent-state contract changes: %w", coreactivation.ExplainRollbackRefusal(ctx, qualification, layout.Executable, request.Candidate))
	}
	var token [8]byte
	if _, err := rand.Read(token[:]); err != nil {
		return "", err
	}
	now := e.now()
	receipt := rebaselineReceipt{
		SchemaVersion: rebaselineReceiptSchemaVersion, ID: now.UTC().Format("20060102T150405Z") + "-" + hex.EncodeToString(token[:]),
		CreatedUnixNano: now.UnixNano(), Source: request.Source,
		ServingSHA: servingSHA, ServingBuild: serving.GetBuildIdentity(), ServingContract: serving.GetPersistentStateContractSha256(),
		CandidateSHA: candidateSHA, CandidateBuild: candidate.GetBuildIdentity(), CandidateContract: candidate.GetPersistentStateContractSha256(),
		Changes: e.explain(ctx, layout.Executable, request.Candidate),
	}
	if err := e.verifyPrivateCandidate(ctx, request.Candidate, candidateSHA); err != nil {
		return "", err
	}
	dir := filepath.Join(transitions, receipt.ID)
	confirmed, err := request.Confirm(RebaselinePlan{ServingBuild: receipt.ServingBuild, ServingContract: receipt.ServingContract, CandidateBuild: receipt.CandidateBuild, CandidateContract: receipt.CandidateContract, Changes: receipt.Changes, RetainedDir: dir})
	if err != nil {
		return "", err
	}
	if !confirmed {
		return "", fmt.Errorf("root rebaseline cancelled; nothing was changed")
	}
	if err := e.verifyPrivateCandidate(ctx, request.Candidate, candidateSHA); err != nil {
		return "", err
	}
	if _, err := config.EnsureOwnedDir(dir); err != nil {
		return "", err
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return "", err
	}
	retained, err := retainRebaselineFile(layout.Executable, filepath.Join(dir, "vrooli"))
	if err != nil || retained != servingSHA {
		return "", fmt.Errorf("retain serving root CLI: %w", errors.Join(err, errors.New("retained bytes must match the observed serving image")))
	}
	if worker, err := coreactivation.InstalledWatchdogPath(home); err == nil {
		if receipt.WatchdogSHA, err = retainRebaselineFile(worker, filepath.Join(dir, "vrooli-watchdog")); err != nil && !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("retain serving watchdog: %w", err)
		}
	}
	if receipt.UnitSHA, err = retainRebaselineFile(layout.UnitPath, filepath.Join(dir, layout.UnitName)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("retain supervisor unit: %w", err)
	}
	data, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return "", err
	}
	return dir, config.WriteOwnedFileDurable(filepath.Join(dir, "receipt.json"), append(data, '\n'), 0o600)
}

// pause disables enrollment when it is enabled now and records the
// enrollment record its disable wrote. A pause already in effect from an
// unfinished rebaseline is kept; a stale marker is cleared.
func (e rebaselineEffects) pause(ctx context.Context, home, path, servingSHA string) error {
	enrollment, exists, err := e.enrollment(home)
	if err != nil {
		return err
	}
	if !exists || !enrollment.Enabled {
		if !e.pauseInEffect(home, path) {
			_ = os.Remove(path)
		}
		return nil
	}
	if err := e.configure(ctx, home, false); err != nil {
		return err
	}
	paused, exists, err := e.enrollment(home)
	if err != nil || !exists || paused.Enabled {
		return fmt.Errorf("enrollment was not paused: %w", errors.Join(err, errors.New("the enrollment record does not show a disabled policy")))
	}
	data, err := json.Marshal(rebaselinePause{SchemaVersion: rebaselinePauseSchemaVersion, PausedUnixNano: e.now().UnixNano(), EnrollmentUpdatedUnixNano: paused.UpdatedUnixNano, PriorServingSHA: servingSHA})
	if err != nil {
		return err
	}
	return config.WriteOwnedFileDurable(path, append(data, '\n'), 0o600)
}

// pauseInEffect reports whether a rebaseline's pause is still the current
// enrollment decision: the record is exactly the disabled one it wrote.
func (e rebaselineEffects) pauseInEffect(home, path string) bool {
	data, err := os.ReadFile(path)
	if err != nil || len(data) > 4096 {
		return false
	}
	var marker rebaselinePause
	if json.Unmarshal(data, &marker) != nil || marker.SchemaVersion != rebaselinePauseSchemaVersion || marker.EnrollmentUpdatedUnixNano <= 0 {
		return false
	}
	enrollment, exists, err := e.enrollment(home)
	return err == nil && exists && !enrollment.Enabled && enrollment.UpdatedUnixNano == marker.EnrollmentUpdatedUnixNano
}

// restore re-enables enrollment when a rebaseline's pause is still in effect;
// enrollment's own qualification decides whether it may. A superseded marker
// is cleared and the operator's later decision stands.
func (e rebaselineEffects) restore(ctx context.Context, home, path string, progress io.Writer) error {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if !e.pauseInEffect(home, path) {
		_, _ = fmt.Fprintln(progress, "   enrollment was changed after the rebaseline paused it; leaving it as the operator set it")
		return removeIfPresent(path)
	}
	if err := e.configure(ctx, home, true); err != nil {
		return err
	}
	return removeIfPresent(path)
}

func removeIfPresent(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// enrollmentState is the live enrollment sentence every stop or refusal ends
// with.
func (e rebaselineEffects) enrollmentState(home, path string) string {
	enrollment, exists, err := e.enrollment(home)
	switch {
	case err != nil:
		return "Automatic root update enrollment could not be read: " + err.Error() + "."
	case exists && enrollment.Enabled:
		return "Automatic root updates are enabled."
	case e.pauseInEffect(home, path):
		return "Automatic root updates are paused by an unfinished rebaseline; `make install` installs directly, without automatic rollback, until a rebaseline completes."
	default:
		return "Automatic root updates are not enrolled."
	}
}

// retainRebaselineFile streams a bounded regular file into a private candidate
// or retained baseline and returns the digest of the bytes durably written.
func retainRebaselineFile(source, destination string) (string, error) {
	const limit = 1 << 30
	info, err := os.Lstat(source)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() || info.Size() > limit {
		return "", fmt.Errorf("%s is not a bounded regular file", source)
	}
	in, err := os.Open(source)
	if err != nil {
		return "", err
	}
	defer in.Close()
	opened, err := in.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return "", fmt.Errorf("%s changed while opening", source)
	}
	var digest string
	err = config.WriteOwnedFileDurableStream(destination, 0o700, func(out io.Writer) error {
		hash := sha256.New()
		n, err := io.Copy(io.MultiWriter(out, hash), io.LimitReader(in, limit+1))
		if err != nil {
			return err
		}
		if n > limit {
			return fmt.Errorf("%s exceeded the copy bound", source)
		}
		digest = hex.EncodeToString(hash.Sum(nil))
		return nil
	})
	return digest, err
}
