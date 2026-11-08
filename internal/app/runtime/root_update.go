package runtimeapp

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"runtime"
	"time"

	platform "github.com/vrooli/platform-go"
	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/coreactivation"
	"github.com/vrooli/vrooli/internal/process"
	watchdog "github.com/vrooli/vrooli/internal/safeguards/emergency-watchdog"
	supervisor "github.com/vrooli/vrooli/internal/safeguards/runtime-supervisor"
)

func rootUpdateLayout(home string) (coreactivation.Layout, error) {
	bin, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyBin)
	if err != nil {
		return coreactivation.Layout{}, err
	}
	logs, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyLogs)
	if err != nil {
		return coreactivation.Layout{}, err
	}
	unit, _ := platform.CoreUnitByID(platform.CoreUnitRuntimeSupervisor)
	return coreactivation.Layout{Home: home, Executable: filepath.Join(bin, "vrooli"), UnitPath: platform.RuntimeSupervisorUnitPath("linux", home), UnitName: unit.Systemd, LogPath: filepath.Join(logs, "runtime-supervisor.log")}, nil
}

// SubmitRootUpdate prepares, admits and dispatches once. Source is needed only
// to acquire/build a candidate upstream; this function accepts its absolute
// binding hint even when that checkout is unavailable. It returns the receipt
// on ambiguous dispatch; callers may wait/reconcile that identity, never replay.
type RootUpdateRequest struct {
	Source, Candidate                string
	ExplicitRetry, RequireEnrollment bool
}

func SubmitRootUpdate(ctx context.Context, home string, request RootUpdateRequest) (coreactivation.Record, error) {
	source, candidate, explicitRetry := request.Source, request.Candidate, request.ExplicitRetry
	var result coreactivation.Record
	if ctx == nil || runtime.GOOS != "linux" || !filepath.IsAbs(source) || filepath.Clean(source) != source || !filepath.IsAbs(candidate) || filepath.Clean(candidate) != candidate {
		return result, coreactivation.ErrConflict
	}
	store, err := coreactivation.NewStore(home)
	if err != nil {
		return result, err
	}
	err = store.Prepare(ctx, func(p *coreactivation.Preparer) error {
		releaseWatchdog, err := watchdog.AcquirePublicationLock(ctx, home)
		if err != nil {
			return err
		}
		defer releaseWatchdog()
		layout, err := rootUpdateLayout(home)
		if err != nil {
			return err
		}
		releaseRoot, err := buildinfo.AcquireBinaryInstallLockContext(ctx, layout.Executable)
		if err != nil {
			return err
		}
		defer releaseRoot()
		if err := store.GuardOrdinaryMutation(ctx); err != nil {
			return err
		}
		if err := coreactivation.ValidateRootExecutableRole(layout.Executable); err != nil {
			return err
		}
		if err := coreactivation.ValidateRootExecutableRole(candidate); err != nil {
			return err
		}
		priorIdentity, err := coreactivation.ProbeRootIdentity(ctx, layout.Executable)
		if err != nil {
			return err
		}
		candidateIdentity, err := coreactivation.ProbeRootIdentity(ctx, candidate)
		if err != nil {
			return err
		}
		if err := coreactivation.QualifyRollbackIdentities(priorIdentity, candidateIdentity); err != nil {
			return coreactivation.ExplainRollbackRefusal(ctx, err, layout.Executable, candidate)
		}
		worker, err := coreactivation.InstalledWatchdogPath(home)
		if err != nil {
			return err
		}
		workerIdentity, err := coreactivation.ProbeWorkerIdentity(ctx, worker, priorIdentity)
		if err != nil {
			return err
		}
		priorUnit, err := coreactivation.ReadDefinition(layout.UnitPath)
		if err != nil {
			return err
		}
		candidateUnit, err := coreactivation.ProbeRootDefinition(ctx, candidate, layout, source)
		if err != nil {
			return err
		}
		native, err := platform.InspectCoreSupervisor(ctx)
		if err != nil {
			return err
		}
		overrides, err := coreactivation.NativeOverridesDigest(ctx, native.DropInPaths)
		if err != nil {
			return err
		}
		incarnation, err := process.Incarnation(native.PID)
		if err != nil {
			return err
		}
		var token [16]byte
		if _, err := rand.Read(token[:]); err != nil {
			return err
		}
		id := hex.EncodeToString(token[:])
		digests, err := p.Freeze(ctx, id, layout, coreactivation.InputSources{Candidate: candidate, Prior: layout.Executable, Worker: worker, CandidateUnit: candidateUnit, PriorUnit: priorUnit})
		if err != nil {
			return err
		}
		result = coreactivation.Record{ID: id, Home: home, SourceRoot: source, Automatic: request.RequireEnrollment, CandidateSHA: digests.Candidate, PriorSHA: digests.Prior, WorkerSHA: digests.Worker, LayoutSHA: digests.Layout, CandidateUnitSHA: digests.CandidateUnit, PriorUnitSHA: digests.PriorUnit, CandidateBuild: candidateIdentity.GetBuildIdentity(), PriorBuild: priorIdentity.GetBuildIdentity(), CandidateStateContract: candidateIdentity.GetPersistentStateContractSha256(), PriorStateContract: priorIdentity.GetPersistentStateContractSha256(), PriorPID: native.PID, PriorIncarnation: incarnation, PriorNativeInvocation: native.InvocationID, NativeOverridesDigest: overrides}
		assets := coreactivation.Assets{Store: store}
		frozen, err := assets.Image(ctx, result, "candidate")
		if err != nil {
			return err
		}
		definition, err := coreactivation.ProbeRootDefinition(ctx, frozen, layout, source)
		if err != nil || !bytes.Equal(definition, candidateUnit) {
			return fmt.Errorf("candidate definition changed while freezing native inputs")
		}
		frozenWorker, err := assets.Image(ctx, result, "worker")
		if err != nil {
			return err
		}
		if _, err := coreactivation.ProbeWorkerIdentity(ctx, frozenWorker, priorIdentity); err != nil {
			return err
		}
		if err := watchdog.QualifyRootRecovery(ctx, home, digests.Worker, workerIdentity.BuildIdentity); err != nil {
			return err
		}
		created := time.Now()
		result.CreatedUnixNano = created.UnixNano()
		result.DeadlineUnixNano = created.Add(5 * time.Minute).UnixNano()
		spec := platform.OperationSpec{Path: frozenWorker, Args: []string{"--core-update-execute", id, store.Directory()}, Env: []string{"HOME=" + home, "PATH=" + platform.DefaultPath("linux", home)}, Dir: home, Limits: platform.OperationLimits{MemoryMaxBytes: 512 << 20, TasksMax: 128, CPUWeight: 400, WallTime: 3 * time.Minute}}
		result.WorkerIntent, err = platform.NewOperationIntent("core-activation", id, spec)
		if err != nil {
			return err
		}
		// Reacquire installation under the operation lock inside Begin. Keeping
		// root -> operation here would deadlock independent restoration.
		releaseRoot()
		engine := coreactivation.Engine{Store: store, Effects: supervisor.ActivationEffects{Store: store}}
		admitted, beginErr := engine.Begin(ctx, result, explicitRetry)
		if beginErr != nil {
			// A rejected retry can return the retained terminal receipt. Do not
			// advertise this preparation's unadmitted identity as an operation.
			if admitted.ID != "" {
				result = admitted
			}
			if current, exists, readErr := store.Current(); readErr == nil && exists && current.ID == id {
				result = current
			}
			return beginErr
		}
		result = admitted
		_, err = engine.Dispatch(ctx, result.ID, spec)
		current, exists, readErr := store.Current()
		if readErr == nil && exists && current.ID == result.ID {
			result = current
		}
		return err
	})
	return result, err
}

// WaitRootUpdate observes the same server/native-owned request. Cancellation
// stops this wait; it does not abort, resubmit or repair the admitted update.
func WaitRootUpdate(ctx context.Context, home, id string) (coreactivation.Record, error) {
	store, err := coreactivation.NewStore(home)
	if err != nil {
		return coreactivation.Record{}, err
	}
	for {
		r, exists, err := store.Current()
		if err != nil {
			return r, err
		}
		if !exists || r.ID != id {
			return r, coreactivation.ErrConflict
		}
		if r.Terminal() {
			if r.Phase != coreactivation.Verified {
				return r, fmt.Errorf("root update %s ended %s (%s)", r.ID, r.Phase, r.FailureClass)
			}
			return r, nil
		}
		timer := time.NewTimer(time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return r, ctx.Err()
		case <-timer.C:
		}
	}
}

func ConfigureRootUpdates(ctx context.Context, home string, enabled bool) (coreactivation.Enrollment, error) {
	if ctx == nil || runtime.GOOS != "linux" {
		return coreactivation.Enrollment{}, coreactivation.ErrConflict
	}
	store, err := coreactivation.NewStore(home)
	if err != nil {
		return coreactivation.Enrollment{}, err
	}
	releaseWatchdog, err := watchdog.AcquirePublicationLock(ctx, home)
	if err != nil {
		return coreactivation.Enrollment{}, err
	}
	defer releaseWatchdog()
	return store.ConfigureEnrollment(ctx, enabled, func(ctx context.Context) (func(), error) {
		layout, err := rootUpdateLayout(home)
		if err != nil {
			return nil, err
		}
		release, err := buildinfo.AcquireBinaryInstallLockContext(ctx, layout.Executable)
		if err != nil {
			return nil, err
		}
		fail := func(err error) (func(), error) { release(); return nil, err }
		if enabled {
			if err := coreactivation.ValidateRootExecutableRole(layout.Executable); err != nil {
				return fail(err)
			}
			prior, err := coreactivation.ProbeRootIdentity(ctx, layout.Executable)
			if err != nil {
				return fail(err)
			}
			worker, err := coreactivation.InstalledWatchdogPath(home)
			if err != nil {
				return fail(err)
			}
			identity, err := coreactivation.ProbeWorkerIdentity(ctx, worker, prior)
			if err != nil {
				return fail(err)
			}
			sha, err := coreactivation.ImageDigest(ctx, worker)
			if err != nil {
				return fail(err)
			}
			if err := watchdog.QualifyRootRecovery(ctx, home, sha, identity.BuildIdentity); err != nil {
				return fail(err)
			}
			native, err := platform.InspectCoreSupervisor(ctx)
			if err != nil {
				return fail(err)
			}
			if native.ActiveState != "active" || native.PID <= 0 || native.FragmentPath != layout.UnitPath || native.NeedDaemonReload != "no" {
				return fail(fmt.Errorf("native root owner is not ready for update enrollment"))
			}
			installed, err := coreactivation.ImageDigest(ctx, layout.Executable)
			if err != nil {
				return fail(err)
			}
			running, err := coreactivation.RunningImageDigest(ctx, native.PID)
			if err != nil || installed != running {
				return fail(coreactivation.ErrConflict)
			}
		}
		return release, nil
	})
}
