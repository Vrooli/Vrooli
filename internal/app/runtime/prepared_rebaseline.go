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
	"time"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/cliinstall"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/coreactivation"
	"github.com/vrooli/vrooli/internal/rebaselineprogress"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

// This file owns the durable root rebaseline lifecycle: state and lock paths,
// progress and watchdog locators, rollback qualification and retained prior
// images, pause/restore state, prepared bundle binding, and completion and
// retirement order. Keep command routing, candidate admission, metrics, and
// presentation outside this source contract.

func preparedRebaselineProgressPath(transitions string) string {
	return filepath.Join(transitions, "progress.json")
}

type preparedRebaselineStatePaths struct {
	transitions string
	pause       string
	progress    string
	lock        string
}

func preparedRebaselineStatePathsForHome(home string) (preparedRebaselineStatePaths, error) {
	stateDir, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyState)
	if err != nil {
		return preparedRebaselineStatePaths{}, err
	}
	transitions := filepath.Join(stateDir, "root-rebaseline")
	return preparedRebaselineStatePaths{
		transitions: transitions,
		pause:       filepath.Join(transitions, "pause.json"),
		progress:    preparedRebaselineProgressPath(transitions),
		lock:        filepath.Join(transitions, "transition.lock"),
	}, nil
}

const (
	rebaselinePauseSchemaVersion   = 1
	rebaselineReceiptSchemaVersion = 1
)

// rebaselinePause is the legacy pause marker, retained for interrupted
// transitions started before pause intent became part of enrollment.
type rebaselinePause struct {
	SchemaVersion             int    `json:"schema_version"`
	PausedUnixNano            int64  `json:"paused_unix_nano"`
	EnrollmentUpdatedUnixNano int64  `json:"enrollment_updated_unix_nano"`
	PriorServingSHA           string `json:"prior_serving_sha256"`
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

// pause atomically records the restore intent with the disabled enrollment.
// A concurrent operator decision supersedes the observed record.
func (e rebaselineEffects) pause(ctx context.Context, home string) error {
	enrollment, exists, err := e.enrollment(home)
	if err != nil {
		return err
	}
	if !exists || !enrollment.Enabled {
		return nil
	}
	return e.compareEnrollment(ctx, home, enrollment, false, true)
}

// qualifyAndRetain qualifies the actual images, obtains the operator
// decision and retains the prior baseline. Nothing on the serving path
// changes before it returns.
func (e rebaselineEffects) qualifyAndRetain(ctx context.Context, home string, layout coreactivation.Layout, transitions string, request RebaselineRequest, servingSHA, candidateSHA, watchdogSHA string, candidate *cliv1.CliVersion, continuing bool, attempt *rootAttempt) (string, error) {
	serving, err := e.probeIdentity(ctx, layout.Executable)
	if err != nil {
		return "", fmt.Errorf("serving root CLI identity: %w", err)
	}
	qualification := coreactivation.QualifyRollbackIdentities(serving, candidate)
	if qualification == nil && continuing {
		if request.AllowContinuationDecision {
			ok, err := request.Confirm(RebaselinePlan{Continuation: true, ServingSHA: servingSHA, CandidateSHA: candidateSHA, WatchdogSHA: watchdogSHA, ServingBuild: serving.GetBuildIdentity(), ServingContract: serving.GetPersistentStateContractSha256(), CandidateBuild: candidate.GetBuildIdentity(), CandidateContract: candidate.GetPersistentStateContractSha256(), Changes: []string{"Continue the interrupted rebaseline with this rebuilt root; its persistent-state contract is unchanged."}})
			if err != nil {
				return "", err
			}
			if !ok {
				return "", fmt.Errorf("root rebaseline continuation cancelled; nothing was changed")
			}
		}
		return "", nil
	}
	if qualification == nil {
		if enrollment, exists, err := e.enrollment(home); err == nil && exists && enrollment.Enabled {
			return "", fmt.Errorf("%w; activate it with `make install`, which keeps automatic rollback", ErrRebaselineNotRequired)
		}
		return "", fmt.Errorf("%w; `make install` installs it directly because automatic root updates are not enrolled", ErrRebaselineNotRequired)
	}
	if !errors.Is(qualification, coreactivation.ErrStateContractChanged) {
		return "", fmt.Errorf("root rebaseline only crosses persistent-state contract changes: %w", coreactivation.ExplainRollbackRefusal(ctx, qualification, layout.Executable, request.Candidate))
	}
	attempt.identities(serving.GetPersistentStateContractSha256(), candidate.GetPersistentStateContractSha256())
	attempt.attributeLater(layout.Executable, request.Candidate)
	attempt.event("qualification", "state_contract_changed")
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
	confirmed, err := request.Confirm(RebaselinePlan{ServingSHA: servingSHA, CandidateSHA: candidateSHA, WatchdogSHA: watchdogSHA, ServingBuild: receipt.ServingBuild, ServingContract: receipt.ServingContract, CandidateBuild: receipt.CandidateBuild, CandidateContract: receipt.CandidateContract, Changes: receipt.Changes, RetainedDir: dir})
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

// pauseInEffect reads only the enrollment owner's atomic intent.
func (e rebaselineEffects) pauseInEffect(home string) bool {
	enrollment, exists, err := e.enrollment(home)
	return err == nil && exists && !enrollment.Enabled && enrollment.RebaselinePaused
}

// Legacy markers are not evidence of operator intent. Stop for an explicit
// enrollment decision rather than enabling updates based on a writable file.
func (e rebaselineEffects) legacyPauseMatches(home, path string) bool {
	enrollment, exists, err := e.enrollment(home)
	if err != nil || !exists || enrollment.Enabled || enrollment.RebaselinePaused {
		return false
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) > 4096 {
		return false
	}
	var marker rebaselinePause
	return json.Unmarshal(data, &marker) == nil && marker.SchemaVersion == rebaselinePauseSchemaVersion && marker.EnrollmentUpdatedUnixNano > 0 && enrollment.UpdatedUnixNano == marker.EnrollmentUpdatedUnixNano
}

func (e rebaselineEffects) restore(ctx context.Context, home, path string, progress io.Writer) error {
	enrollment, exists, err := e.enrollment(home)
	if err != nil {
		return err
	}
	if !exists || !e.pauseInEffect(home) {
		return removeIfPresent(path)
	}
	if err := e.compareEnrollment(ctx, home, enrollment, true, false); err != nil {
		if errors.Is(err, errEnrollmentSuperseded) {
			_, _ = fmt.Fprintln(progress, "   enrollment was changed after the rebaseline paused it; leaving it as the operator set it")
			return removeIfPresent(path)
		}
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

func preparedRebaselineWatchdogPath(transitions string) string {
	return filepath.Join(transitions, "watchdog-pending")
}

func preparedBundleRecoveryOperations() (func(string, string, string) (string, error), func(string, string) error) {
	return cliinstall.RetainRebaselineBundle, cliinstall.RemoveRebaselineBundle
}

func readPreparedRebaselineProgress(path string) (*rebaselineprogress.Progress, error) {
	return rebaselineprogress.Read(path)
}

func validatePreparedRebaselineProgress(progress *rebaselineprogress.Progress, candidateSHA, bundleSHA, source string, hasRecoveryCallback, hubDecision bool) error {
	if progress == nil || progress.Complete {
		return nil
	}
	if progress.Candidate != candidateSHA || progress.Bundle != bundleSHA || (progress.SourceRoot != "" && progress.SourceRoot != source) || (progress.Bundle != "" && !hasRecoveryCallback) {
		return fmt.Errorf("an unfinished rebaseline belongs to another candidate or bundle; resume the original prepared bundle before starting another transition")
	}
	if (progress.Mode == "hub" && !hubDecision) || (progress.Mode == "prepared" && hubDecision) {
		return fmt.Errorf("unfinished rebaseline must resume with its original authority mode")
	}
	if progress.Bundle != "" && progress.Mode == "" && !hubDecision {
		return fmt.Errorf("legacy rebaseline has no authority-mode evidence; resume the original prepared candidate with --hub for a verified human decision")
	}
	return nil
}

func preparedProgressIncomplete(progress *rebaselineprogress.Progress) bool {
	return progress != nil && !progress.Complete
}

func preparedProgressAllowsRollbackQualification(progress *rebaselineprogress.Progress) bool {
	return progress == nil || progress.Complete
}

func preparedProgressContinuesCandidate(progress *rebaselineprogress.Progress, candidateSHA string) bool {
	return preparedProgressIncomplete(progress) && progress.Candidate == candidateSHA
}

func preparedProgressNeedsFreshDecision(progress *rebaselineprogress.Progress, hubDecision bool) bool {
	return hubDecision && preparedProgressIncomplete(progress) && progress.Mode == ""
}

func preparedProgressMatchesWatchdog(progress *rebaselineprogress.Progress, watchdogSHA string) bool {
	return !preparedProgressIncomplete(progress) || progress.Watchdog == watchdogSHA
}

func preparedProgressWatchdog(progress *rebaselineprogress.Progress) string {
	if progress == nil {
		return ""
	}
	return progress.Watchdog
}

func newPreparedRebaselineProgress(candidateSHA, watchdogSHA, bundleSHA, source string, hubDecision bool) rebaselineprogress.Progress {
	mode := ""
	if bundleSHA != "" {
		mode = "prepared"
		if hubDecision {
			mode = "hub"
		}
	}
	return rebaselineprogress.Progress{Version: 1, Candidate: candidateSHA, Watchdog: watchdogSHA, Bundle: bundleSHA, SourceRoot: source, Mode: mode}
}

func restorePreparedRebaselineWatchdog(
	ctx context.Context,
	e rebaselineEffects,
	progress *rebaselineprogress.Progress,
	pending, snapshot, staged, source string,
) error {
	_, pendingErr := os.Lstat(pending)
	if preparedProgressIncomplete(progress) && pendingErr == nil {
		if snapshot != "" {
			snapshotSHA, err := e.imageDigest(ctx, snapshot)
			if err != nil || snapshotSHA != preparedProgressWatchdog(progress) {
				return fmt.Errorf("watchdog changed from the prepared snapshot: %w", errors.Join(err, errors.New("qualified digest differs")))
			}
		}
		if _, err := retainRebaselineFile(pending, staged); err != nil {
			return fmt.Errorf("restore pending watchdog: %w", err)
		}
		return nil
	}
	if pendingErr != nil && !errors.Is(pendingErr, os.ErrNotExist) {
		return fmt.Errorf("inspect pending watchdog: %w", pendingErr)
	}
	if snapshot != "" {
		if _, err := os.Lstat(snapshot); err == nil {
			if _, err := retainRebaselineFile(snapshot, staged); err != nil {
				return fmt.Errorf("restore qualified watchdog snapshot: %w", err)
			}
			return nil
		} else if errors.Is(err, os.ErrNotExist) {
			if err := e.buildWatchdog(source, staged); err != nil {
				return fmt.Errorf("build the watchdog for this candidate: %w", err)
			}
			if _, err := retainRebaselineFile(staged, snapshot); err != nil {
				return fmt.Errorf("retain qualified watchdog snapshot: %w", err)
			}
			return nil
		} else {
			return err
		}
	}
	if err := e.buildWatchdog(source, staged); err != nil {
		return fmt.Errorf("build the watchdog for this candidate: %w", err)
	}
	return nil
}

func selectPreparedRebaselineWatchdog(bundleDir, manifest string, progress *rebaselineprogress.Progress, pending, snapshot string) (string, string, error) {
	if progress != nil && !progress.Complete {
		if _, err := os.Stat(pending); err == nil {
			return snapshot, progress.Watchdog, nil
		} else if !os.IsNotExist(err) {
			return "", "", err
		}
	}
	if manifest == "" {
		return "", "", fmt.Errorf("rebaseline requires a prepared bundle with its watchdog; run make rebaseline")
	}
	digest, err := cliinstall.PreparedBundleWatchdogSHA(bundleDir, manifest)
	if err != nil {
		return "", "", fmt.Errorf("inspect prepared watchdog: %w", err)
	}
	if digest == "" {
		return "", "", fmt.Errorf("this older prepared bundle does not bind its watchdog; prepare a new bundle with make rebaseline")
	}
	if progress != nil && !progress.Complete && progress.Watchdog != digest {
		return "", "", fmt.Errorf("prepared watchdog differs from the unfinished rebaseline")
	}
	return filepath.Join(bundleDir, "vrooli-watchdog"), digest, nil
}

// preparePreparedRebaselineRecovery records authority and intent only after
// retaining the exact prepared bundle and watchdog. The progress record is
// the recovery point, so it is written last.
type preparedRebaselineRecoveryPreparation struct {
	BundleDir               string
	HubLocatorBound         bool
	RecoveryProgressWritten bool
}

func preparePreparedRebaselineRecovery(
	ctx context.Context,
	e rebaselineEffects,
	request RebaselineRequest,
	home string,
	admission RebaselineAdmission,
	originalCandidate, bundleSHA, stagedWatchdog, pendingWatchdog, progressPath string,
	watchdogSHA string,
	record rebaselineprogress.Progress,
) (preparedRebaselineRecoveryPreparation, error) {
	prepared := preparedRebaselineRecoveryPreparation{HubLocatorBound: !request.HubDecision}
	var err error
	if bundleSHA != "" {
		prepared.BundleDir, err = e.retainBundle(home, filepath.Dir(originalCandidate), bundleSHA)
		if err != nil {
			return prepared, fmt.Errorf("retain prepared recovery bundle: %w", err)
		}
		if request.PrepareRecovery != nil {
			if err := request.PrepareRecovery(ctx, RebaselineRecovery{Admission: admission, BundleDir: prepared.BundleDir}); err != nil {
				return prepared, fmt.Errorf("bind recovery authority: %w", err)
			}
			if request.HubDecision {
				prepared.HubLocatorBound = true
			}
		}
	}
	retainedWatchdogSHA, err := retainRebaselineFile(stagedWatchdog, pendingWatchdog)
	if err != nil || retainedWatchdogSHA != watchdogSHA {
		return prepared, fmt.Errorf("retain pending watchdog: %w", errors.Join(err, errors.New("qualified watchdog snapshot must match")))
	}
	if err := rebaselineprogress.Write(progressPath, record); err != nil {
		return prepared, fmt.Errorf("record rebaseline progress: %w", err)
	}
	prepared.RecoveryProgressWritten = true
	return prepared, nil
}

// completePreparedRebaselineRecovery makes completion durable before removing
// the retained bundle. A failed progress write leaves the bundle available to
// resume; failed cleanup leaves a completed record for safe later retirement.
func completePreparedRebaselineRecovery(e rebaselineEffects, home, bundleSHA, progressPath string, record rebaselineprogress.Progress) (bool, error) {
	record.Complete = true
	if err := rebaselineprogress.Write(progressPath, record); err != nil {
		return false, fmt.Errorf("record completed transition: %w", err)
	}
	if bundleSHA == "" {
		return true, nil
	}
	return true, e.removeBundle(home, bundleSHA)
}

// preparedRebaselineTransaction contains the compatibility-sensitive effect
// order for a prepared root transition. Presentation and telemetry are
// callbacks so their implementations remain outside the persisted-state
// source owner.
type preparedRebaselineTransaction struct {
	preparedRebaselinePlan
	ctx            context.Context
	e              rebaselineEffects
	home           string
	progress       io.Writer
	attemptEvent   func(string, string)
	stepMessage    func(string)
	retirementWarn func(error)
	refuse         func(error) error
	fail           func(step string, cause error, recoveryBundle, retained string, hubLocatorBound, recoveryProgressWritten bool) error
}

type preparedRebaselinePlan struct {
	request           RebaselineRequest
	candidate         *cliv1.CliVersion
	candidateSHA      string
	servingSHA        string
	watchdogSHA       string
	originalCandidate string
	stagedWatchdog    string
	pendingWatchdog   string
	progressPath      string
	pausePath         string
	layout            coreactivation.Layout
	admission         RebaselineAdmission
	finishing         bool
	retained          string
}

// preparePreparedRebaselinePlan owns interpretation of durable progress and
// the decisions that select qualification, watchdog recovery and continuation.
// Refusal text, progress output and attempt telemetry are supplied by callers.
func preparePreparedRebaselinePlan(input preparedRebaselineRunInput) (preparedRebaselinePlan, error) {
	ctx, e, home := input.ctx, input.effects, input.home
	statePaths, err := preparedRebaselineStatePathsForHome(home)
	if err != nil {
		return preparedRebaselinePlan{}, input.refuse(err)
	}
	transitions, work := statePaths.transitions, input.paths.work
	originalCandidate := input.paths.originalCandidate
	progressPath, pausePath := statePaths.progress, statePaths.pause
	layout, request, candidate := input.layout, input.request, input.candidate
	candidateSHA, servingSHA := input.candidateSHA, input.servingSHA
	attemptEvent := input.telemetry.attemptEvent
	progressMessage, retentionCompleted := input.telemetry.progressMessage, input.telemetry.retentionCompleted
	attempt, refuse := input.attempt, input.refuse
	plan := preparedRebaselinePlan{
		request: request, candidate: candidate, candidateSHA: candidateSHA, servingSHA: servingSHA,
		originalCandidate: originalCandidate, progressPath: progressPath, pausePath: pausePath,
		layout: layout, finishing: servingSHA == candidateSHA,
	}
	if e.legacyPauseMatches(home, pausePath) {
		return plan, refuse(fmt.Errorf("legacy rebaseline pause requires an explicit enrollment decision; inspect %s, then configure root updates as intended before retrying", pausePath))
	}
	progressState, err := readPreparedRebaselineProgress(progressPath)
	if err != nil {
		return plan, refuse(err)
	}
	enrollment, exists, err := e.enrollment(home)
	if err != nil {
		return plan, refuse(fmt.Errorf("read root update enrollment: %w", err))
	}
	paused := exists && !enrollment.Enabled && enrollment.RebaselinePaused
	if request.BundleSHA != "" && request.AfterActivation == nil {
		return plan, refuse(fmt.Errorf("prepared rebaseline requires its companion publication; resume the prepared bundle"))
	}
	if err := validatePreparedRebaselineProgress(progressState, candidateSHA, request.BundleSHA, request.Source, request.AfterActivation != nil, request.HubDecision); err != nil {
		return plan, refuse(err)
	}
	if !paused && !e.legacyPauseMatches(home, pausePath) && preparedProgressAllowsRollbackQualification(progressState) {
		serving, probeErr := e.probeIdentity(ctx, layout.Executable)
		if probeErr != nil {
			return plan, refuse(fmt.Errorf("serving root CLI identity: %w", probeErr))
		}
		if coreactivation.QualifyRollbackIdentities(serving, candidate) == nil {
			if exists && enrollment.Enabled {
				return plan, refuse(fmt.Errorf("%w; activate it with `make install`, which keeps automatic rollback", ErrRebaselineNotRequired))
			}
			return plan, refuse(fmt.Errorf("%w; `make install` installs it directly because automatic root updates are not enrolled", ErrRebaselineNotRequired))
		}
	}
	if request.RequireTransitionCaller != nil {
		if err := request.RequireTransitionCaller(); err != nil {
			return plan, refuse(err)
		}
	}
	plan.stagedWatchdog = filepath.Join(work, "vrooli-watchdog")
	plan.pendingWatchdog = preparedRebaselineWatchdogPath(transitions)
	selectedWatchdog, expectedWatchdog, err := e.selectWatchdog(filepath.Dir(originalCandidate), request.BundleSHA, progressState, plan.pendingWatchdog, request.WatchdogSnapshot)
	if err != nil {
		return plan, refuse(err)
	}
	request.WatchdogSnapshot = selectedWatchdog
	plan.request = request
	if err := restorePreparedRebaselineWatchdog(ctx, e, progressState, plan.pendingWatchdog, request.WatchdogSnapshot, plan.stagedWatchdog, request.Source); err != nil {
		return plan, refuse(err)
	}
	watchdogSHA, err := e.imageDigest(ctx, plan.stagedWatchdog)
	if err != nil {
		return plan, refuse(err)
	}
	if expectedWatchdog != "" && watchdogSHA != expectedWatchdog {
		return plan, refuse(fmt.Errorf("watchdog does not match the prepared bundle or unfinished rebaseline"))
	}
	if !preparedProgressMatchesWatchdog(progressState, watchdogSHA) {
		return plan, refuse(fmt.Errorf("watchdog differs from the unfinished rebaseline; restore the original qualified watchdog before resuming"))
	}
	if err := e.probeWorker(ctx, plan.stagedWatchdog, candidate); err != nil {
		return plan, refuse(fmt.Errorf("the watchdog built from %s does not match the candidate's contract (rebuild the candidate from the same checkout): %w", request.Source, err))
	}
	if actual, err := e.imageDigest(ctx, plan.stagedWatchdog); err != nil || actual != watchdogSHA {
		return plan, refuse(fmt.Errorf("watchdog image changed during qualification: %w", errors.Join(err, errors.New("qualified bytes must match the staged image"))))
	}
	plan.watchdogSHA = watchdogSHA
	var retained string
	switch {
	case plan.finishing:
		attemptEvent("recovery", "resumed")
		progressMessage("The candidate already serves; finishing its rebaseline.")
		if request.RequireFreshFinishingDecision && preparedProgressNeedsFreshDecision(progressState, request.HubDecision) {
			ok, err := request.Confirm(RebaselinePlan{Continuation: true, ServingSHA: servingSHA, CandidateSHA: candidateSHA, WatchdogSHA: watchdogSHA, ServingBuild: candidate.GetBuildIdentity(), CandidateBuild: candidate.GetBuildIdentity(), ServingContract: candidate.GetPersistentStateContractSha256(), CandidateContract: candidate.GetPersistentStateContractSha256(), Changes: []string{"Complete the interrupted rebaseline for the already-serving root and its retained bundle."}})
			if err != nil {
				return plan, refuse(err)
			}
			if !ok {
				return plan, refuse(fmt.Errorf("legacy rebaseline continuation cancelled; nothing was changed"))
			}
		}
	case e.pauseInEffect(home) || preparedProgressContinuesCandidate(progressState, candidateSHA):
		retained, err = e.qualifyAndRetain(ctx, home, layout, transitions, request, servingSHA, candidateSHA, watchdogSHA, candidate, true, attempt)
	default:
		retained, err = e.qualifyAndRetain(ctx, home, layout, transitions, request, servingSHA, candidateSHA, watchdogSHA, candidate, false, attempt)
	}
	if err != nil {
		return plan, refuse(err)
	}
	if retained != "" {
		retentionCompleted(retained)
	}
	if err := e.verifyPrivateCandidate(ctx, request.Candidate, candidateSHA); err != nil {
		return plan, refuse(err)
	}
	plan.retained = retained
	plan.admission = RebaselineAdmission{CandidateSHA: candidateSHA, WatchdogSHA: watchdogSHA}
	return plan, nil
}

type preparedRebaselineRunInput struct {
	ctx          context.Context
	effects      rebaselineEffects
	home         string
	paths        preparedRebaselinePaths
	layout       coreactivation.Layout
	request      RebaselineRequest
	candidate    *cliv1.CliVersion
	candidateSHA string
	servingSHA   string
	attempt      *rootAttempt
	progress     io.Writer
	telemetry    preparedRebaselineTelemetry
	refuse       func(error) error
	fail         func(step string, cause error, recoveryBundle, retained string, hubLocatorBound, recoveryProgressWritten bool) error
}

type preparedRebaselinePaths struct {
	work              string
	originalCandidate string
}

type preparedRebaselineTelemetry struct {
	attemptEvent       func(string, string)
	progressMessage    func(string)
	stepMessage        func(string)
	retirementWarn     func(error)
	retentionCompleted func(string)
}

func runPreparedRebaseline(input preparedRebaselineRunInput) (RebaselineResult, error) {
	plan, err := preparePreparedRebaselinePlan(input)
	if err != nil {
		return RebaselineResult{}, err
	}
	return (preparedRebaselineTransaction{
		preparedRebaselinePlan: plan,
		ctx:                    input.ctx, e: input.effects, home: input.home, progress: input.progress,
		attemptEvent: input.telemetry.attemptEvent, stepMessage: input.telemetry.stepMessage,
		retirementWarn: input.telemetry.retirementWarn, refuse: input.refuse, fail: input.fail,
	}).run()
}

func (tx preparedRebaselineTransaction) run() (RebaselineResult, error) {
	e, request, ctx := tx.e, tx.request, tx.ctx
	record := newPreparedRebaselineProgress(tx.candidateSHA, tx.watchdogSHA, request.BundleSHA, request.Source, request.HubDecision)
	if request.BeforeEffects != nil {
		if err := request.BeforeEffects(ctx, tx.admission); err != nil {
			return RebaselineResult{}, tx.refuse(fmt.Errorf("rebaseline authority for qualified images: %w", err))
		}
	}
	prepared, err := preparePreparedRebaselineRecovery(ctx, e, request, tx.home, tx.admission, tx.originalCandidate, request.BundleSHA, tx.stagedWatchdog, tx.pendingWatchdog, tx.progressPath, tx.watchdogSHA, record)
	recoveryBundle, hubLocatorBound, recoveryProgressWritten := prepared.BundleDir, prepared.HubLocatorBound, prepared.RecoveryProgressWritten
	if err != nil {
		return RebaselineResult{}, tx.fail("record rebaseline intent", err, recoveryBundle, tx.retained, hubLocatorBound, recoveryProgressWritten)
	}
	var releaseRoot, releaseWatchdog func()
	defer func() {
		if releaseRoot != nil {
			releaseRoot()
		}
		if releaseWatchdog != nil {
			releaseWatchdog()
		}
	}()
	steps := []struct {
		name  string
		stage string
		run   func() error
	}{
		{"pause automatic root updates", "pause", func() error {
			if tx.finishing {
				return nil
			}
			return e.pause(ctx, tx.home)
		}},
		{"fence concurrent root changes", "fence", func() error {
			var err error
			releaseWatchdog, err = e.lockWatchdog(ctx, tx.home)
			if err != nil {
				return err
			}
			releaseRoot, err = e.lockRoot(ctx, tx.layout.Executable)
			if err != nil {
				return err
			}
			if err := e.guardLocked(ctx, tx.home); err != nil {
				return err
			}
			actual, err := e.imageDigest(ctx, tx.layout.Executable)
			if err != nil {
				return err
			}
			if actual != tx.servingSHA {
				return fmt.Errorf("installed image differs from expected serving image: expected %s, observed %s", tx.servingSHA, actual)
			}
			return nil
		}},
		{"install the candidate root CLI", "install", func() error {
			if tx.finishing {
				return nil
			}
			if err := e.verifyPrivateCandidate(ctx, request.Candidate, tx.candidateSHA); err != nil {
				return err
			}
			if err := e.installRoot(request.Candidate, tx.layout.Executable, tx.servingSHA); err != nil {
				return err
			}
			installed, err := e.imageDigest(ctx, tx.layout.Executable)
			if err != nil {
				return fmt.Errorf("verify installed root CLI digest: %w", err)
			}
			if installed != tx.candidateSHA {
				return fmt.Errorf("installed root CLI is not the qualified candidate image: expected SHA-256 %s, observed %s", tx.candidateSHA, installed)
			}
			return nil
		}},
		{"publish the watchdog built for this candidate", "watchdog", func() error {
			return e.publishWatchdog(ctx, tx.home, tx.stagedWatchdog, tx.watchdogSHA)
		}},
		{"restart the runtime supervisor on the new image", "supervisor", func() error {
			quick, cancel := context.WithTimeout(ctx, 3*time.Second)
			defer cancel()
			if e.waitServing(quick, tx.home, tx.candidateSHA, tx.candidate.GetBuildIdentity()) == nil {
				return nil // already serving the candidate with a fresh lease
			}
			return e.converge(ctx, tx.home, request.Source, tx.layout.Executable)
		}},
		{"verify the serving image and a fresh supervisor lease", "verify", func() error {
			verify, cancel := context.WithTimeout(ctx, 3*time.Minute)
			defer cancel()
			return e.waitServing(verify, tx.home, tx.candidateSHA, tx.candidate.GetBuildIdentity())
		}},
		{"publish companions for the serving root", "companions", func() error {
			if request.AfterActivation == nil {
				return nil
			}
			actual, err := e.imageDigest(ctx, tx.layout.Executable)
			if err != nil {
				return err
			}
			if actual != tx.candidateSHA {
				return fmt.Errorf("installed root changed before companion publication")
			}
			return request.AfterActivation(ctx, RebaselineRecovery{Admission: tx.admission, BundleDir: recoveryBundle})
		}},
		{"restore automatic root updates", "restore", func() error {
			releaseRoot()
			releaseRoot = nil
			releaseWatchdog()
			releaseWatchdog = nil
			return e.restore(ctx, tx.home, tx.pausePath, tx.progress)
		}},
	}
	for _, step := range steps {
		tx.attemptEvent(step.stage, "started")
		tx.stepMessage(step.name)
		if err := ctx.Err(); err != nil {
			tx.attemptEvent(step.stage, "failed")
			return RebaselineResult{}, tx.fail(step.name, err, recoveryBundle, tx.retained, hubLocatorBound, recoveryProgressWritten)
		}
		if err := step.run(); err != nil {
			tx.attemptEvent(step.stage, "failed")
			return RebaselineResult{}, tx.fail(step.name, err, recoveryBundle, tx.retained, hubLocatorBound, recoveryProgressWritten)
		}
		tx.attemptEvent(step.stage, "completed")
	}
	enrollment, exists, err := e.enrollment(tx.home)
	if err != nil {
		return RebaselineResult{}, tx.fail("read final enrollment", err, recoveryBundle, tx.retained, hubLocatorBound, recoveryProgressWritten)
	}
	completed, err := completePreparedRebaselineRecovery(e, tx.home, request.BundleSHA, tx.progressPath, record)
	if err != nil && !completed {
		return RebaselineResult{}, tx.fail("record completed transition", err, recoveryBundle, tx.retained, hubLocatorBound, recoveryProgressWritten)
	}
	if err != nil {
		tx.retirementWarn(err)
	}
	return RebaselineResult{
		Build: tx.candidate.GetBuildIdentity(), Contract: tx.candidate.GetPersistentStateContractSha256(),
		RetainedDir: tx.retained, Enrolled: exists && enrollment.Enabled, Finished: tx.finishing,
	}, nil
}
