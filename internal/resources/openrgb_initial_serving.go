package resources

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"time"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/lightingruntime"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/tuning"
)

func validateManagedInitialServing(state ManagedServiceState) error {
	l := state.InitialLaunch
	if l == nil || state.UpdateLaunch != nil || !managedUpdateKey.MatchString(l.AttemptID) || l.AttemptID != state.InstanceID || l.NativeRef.ValidateIdentity() != nil || l.NativeRef.Intent.Owner != "resource-initial" || l.NativeRef.Intent.AttemptID != l.AttemptID || l.NativeRef.RootPID != state.PID {
		return errors.New("invalid initial native serving identity")
	}
	if !l.NativeRef.MatchesIncarnation(state.Incarnation) {
		return errors.New("initial native serving incarnation differs from root")
	}
	return nil
}
func readOpenRGBInitialServing(state ManagedServiceState) (openRGBInitialAttempt, managedUpdateExecFence, error) {
	var a openRGBInitialAttempt
	var f managedUpdateExecFence
	if err := validateManagedInitialServing(state); err != nil {
		return a, f, err
	}
	store, err := newOpenRGBInitialStore()
	if err != nil {
		return a, f, err
	}
	id, e := store.current()
	if e != nil {
		return a, f, errors.Join(errors.New("initial serving attempt is no longer current"), e)
	}
	a, err = store.read(state.InitialLaunch.AttemptID)
	if err != nil {
		return a, f, err
	}
	fences, err := newManagedInitialExecStore(a.ID)
	if err != nil {
		return a, f, err
	}
	f, err = fences.read()
	if err != nil {
		return a, f, err
	}
	if !sameManagedExecIdentity(f, a.fence()) || !sameOpenRGBExecPolicy(f.OpenRGBPolicy, &a.Policy) || f.Claim == nil || !f.Claim.Equal(state.InitialLaunch.NativeRef) || f.OpenRGBWitness == nil || f.OpenRGBWitness.Incarnation != state.Incarnation || state.ArtifactPath != a.Contract.Artifact.Path || state.ArtifactSHA256 != a.Contract.Artifact.SHA256 || state.ArtifactVersion != a.Contract.Artifact.Version || state.OwnershipTokenHash != a.TokenHash {
		return a, f, errors.New("initial serving publication differs from retained claim")
	}
	if id != state.InitialLaunch.AttemptID {
		// A normal restart submits its next admission before the wrapper
		// resolves selection. Canonical stopped metadata still identifies the
		// prior launch and its frozen catalog/policy. It is reusable metadata
		// only after that exact native claim was durably revoked and retired;
		// a live or uncertain prior must still be the current admission.
		if state.StoppedAt == nil || state.StoppedAt.IsZero() ||
			a.State != openRGBInitialStopped || a.StoppedAt == nil || a.StoppedAt.IsZero() ||
			a.QuiescentAt == nil || a.QuiescentAt.IsZero() || a.NativeRef == nil || !a.NativeRef.Equal(*f.Claim) ||
			f.State != managedUpdateExecFenceRevoked || f.RevokedAt == nil || f.RevokedAt.IsZero() ||
			f.ClaimedAt == nil || !state.StartedAt.Equal(*f.ClaimedAt) {
			return a, f, errors.New("initial serving attempt is no longer current and has no exact retired publication")
		}
	}
	return a, f, nil
}
func (c *Controller) observeOpenRGBInitialProcess(ctx context.Context, a openRGBInitialAttempt) (ManagedServiceState, error) {
	fences, err := newManagedInitialExecStore(a.ID)
	if err != nil {
		return ManagedServiceState{}, err
	}
	f, err := fences.read()
	if err != nil {
		return ManagedServiceState{}, err
	}
	if !sameManagedExecIdentity(f, a.fence()) {
		return ManagedServiceState{}, errors.New("initial OpenRGB exec fence identity changed")
	}
	if f.State == managedUpdateExecFencePrepared {
		return ManagedServiceState{}, fmt.Errorf("%w: initial OpenRGB witness is not claimed", errOpenRGBNotReady)
	}
	if f.State != managedUpdateExecFenceClaimed || f.Claim == nil || f.OpenRGBWitness == nil {
		return ManagedServiceState{}, errors.New("initial OpenRGB launch was revoked or has no witness")
	}
	ref, err := platform.LookupOwnedService(ctx, a.Intent)
	if err != nil {
		return ManagedServiceState{}, err
	}
	if !ref.Equal(*f.Claim) {
		return ManagedServiceState{}, errors.New("initial OpenRGB native invocation changed")
	}
	observed, err := platform.InspectOwnedService(ctx, ref)
	if err != nil {
		return ManagedServiceState{}, err
	}
	if observed.State != platform.OperationRunning || !observed.RootAlive || !observed.Populated {
		return ManagedServiceState{}, errors.New("initial OpenRGB native root is not running")
	}
	incarnation, err := process.Incarnation(ref.RootPID)
	if err != nil {
		return ManagedServiceState{}, err
	}
	if incarnation != f.OpenRGBWitness.Incarnation {
		return ManagedServiceState{}, errors.New("initial OpenRGB incarnation changed")
	}
	token, err := readManagedUpdateToken(a.tokenPath())
	if err != nil {
		return ManagedServiceState{}, err
	}
	if managedServiceTokenHash(token) != a.TokenHash {
		return ManagedServiceState{}, errors.New("initial OpenRGB token changed")
	}
	base, err := renderManagedServiceInvocation(a.Contract, a.Runtime, a.root(), token)
	if err != nil {
		return ManagedServiceState{}, err
	}
	observation := openRGBServingObservation{ref: ref, witness: *f.OpenRGBWitness, endpoint: netip.AddrPortFrom(netip.MustParseAddr("127.0.0.1"), uint16(a.Runtime.Port)), base: base}
	if err := c.observeOpenRGBNativePayload(ctx, a.Contract.Artifact, observation); err != nil {
		return ManagedServiceState{}, err
	}
	state := ManagedServiceState{InitialLaunch: &ManagedInitialServingLaunch{AttemptID: a.ID, NativeRef: ref}, InstanceID: a.ID, OwnershipTokenHash: a.TokenHash, PID: ref.RootPID, Incarnation: incarnation, ArtifactPath: a.Contract.Artifact.Path, ArtifactSHA256: a.Contract.Artifact.SHA256, ArtifactVersion: a.Contract.Artifact.Version, StartedAt: *f.ClaimedAt}
	return state, validateManagedInitialServing(state)
}

// Per-controller native observation edges keep startup sequencing testable
// without substituting the durable claim or readiness state machine.
type openRGBPayloadEffects struct {
	context  func(platform.ServiceRef, string) (string, error)
	listener func(netip.AddrPort, int) error
	payload  func(context.Context, ManagedServiceArtifact, openRGBServingObservation) error
}

func (c *Controller) observeOpenRGBNativePayload(ctx context.Context, artifact ManagedServiceArtifact, observation openRGBServingObservation) error {
	effects := openRGBPayloadEffects{context: openRGBServingContext, listener: observeOpenRGBHeadlessListener, payload: c.observeOpenRGBVerifiedPayload}
	if c.openRGBPayloadEffects != nil {
		effects = *c.openRGBPayloadEffects
	}
	// A durable claim precedes configuration and image handoff. Waiting for
	// the exact owner's listener is retryable; foreign/context mismatch is not.
	if err := observation.verifyNativeContextAndListener(effects.context, effects.listener); err != nil {
		return err
	}
	return effects.payload(ctx, artifact, observation)
}
func (c *Controller) observeOpenRGBVerifiedPayload(ctx context.Context, artifact ManagedServiceArtifact, observation openRGBServingObservation) error {
	if err := verifyOpenRGBProcessOnlyOwner(observation.ref, platform.VerifyDarwinNoChildrenProcess); err != nil {
		return err
	}

	stage, err := c.verifyOpenRGBRoleArtifact(ctx, artifact)
	if err != nil {
		return err
	}
	image, err := openOpenRGBServingImage(ctx, observation.ref, observation.witness)
	if err != nil {
		return err
	}
	defer image.Close()
	info, err := image.Stat()
	if err != nil {
		return err
	}
	digest, err := managedUpdateImageDigest(ctx, image)
	if err != nil {
		return err
	}
	return observation.observeAppImageStage(ctx, *stage, info, digest)
}

func (c *Controller) OpenRGBDevicePolicyVerifier() OpenRGBDevicePolicyVerifier {
	return func(ctx context.Context, check OpenRGBDevicePolicyCheck) error {
		pending, err := c.OpenRGBDeviceSetupPending(ctx)
		if err != nil {
			return err
		}
		if pending {
			return errors.New("lighting device access transaction requires recovery before launch")
		}
		record, err := c.readOpenRGBDevicePolicyRecord()
		if err != nil || record == nil {
			return errors.Join(errors.New("OpenRGB selected policy is unavailable"), err)
		}
		if record.NativePolicy != nil {
			account, e := c.openRGBNativeServiceAccount(ctx)
			if e != nil || account.Principal != record.NativePolicy.Account {
				return errors.Join(errors.New("OpenRGB native selected account changed"), e)
			}
			return verifyOpenRGBNativeDevicePolicyRequest(ctx, *record.NativePolicy, check, c.VerifyOpenRGBNativeDevicePolicy)
		}
		if err := hostreqkit.ValidateOpenRGBLinuxDetectorScope(record.Policy); err != nil {
			return err
		}
		return verifyOpenRGBDevicePolicyRequest(ctx, record.Policy, check, hostreqkit.ObserveLinuxOpenRGBDevicePolicy, c.VerifyOpenRGBNativeDevicePolicy)
	}
}

// VerifyOpenRGBNativeDevicePolicy consumes a retained final-context witness;
// it never transfers a process-external access check into a serving claim.
func (c *Controller) VerifyOpenRGBNativeDevicePolicy(ctx context.Context, check OpenRGBDevicePolicyCheck) error {
	if check.PID <= 0 {
		return errors.New("OpenRGB native device proof requires an exact live PID")
	}
	supervisor, _, err := managedServiceSupervisorFor(openRGBResourceName)
	if err != nil {
		return err
	}
	return verifyOpenRGBNativePolicy(ctx, check, supervisor.readState, c.observeOpenRGBServingDeviceWitness)
}

func verifyOpenRGBNativePolicy(ctx context.Context, check OpenRGBDevicePolicyCheck, read func() (ManagedServiceState, error), observe func(context.Context, ManagedServiceState) (openRGBExecWitness, error)) error {
	if check.PID <= 0 {
		return errors.New("OpenRGB native device proof requires an exact live PID")
	}
	state, err := read()
	if err != nil {
		return err
	}
	if state.PID != check.PID || state.Incarnation != check.Incarnation || state.ArtifactSHA256 != check.Artifact.SHA256 || state.ArtifactVersion != check.Artifact.Version || state.ArtifactPath != check.Artifact.Path || state.StoppedAt != nil {
		return errors.New("OpenRGB serving identity differs from requested device proof")
	}
	if state.InitialLaunch == nil && state.UpdateLaunch == nil {
		return errors.New("OpenRGB process has no retained native device witness")
	}
	witness, err := observe(ctx, state)
	if err != nil {
		return err
	}
	if witness.Policy.Account != check.Account || witness.Policy.ID != check.PolicyID || witness.Policy.DetectorCatalogSHA256 != check.DetectorSHA256 || witness.Policy.ArtifactSHA256 != check.Artifact.SHA256 {
		return errors.New("OpenRGB serving witness differs from selected device policy")
	}

	after, err := read()
	if err != nil {
		return err
	}
	if !sameOpenRGBInitialState(after, state) {
		return errors.New("OpenRGB canonical owner changed during device proof")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if !updateDigest(check.DeviceGenerationSHA256) || witness.DeviceGenerationSHA256 != check.DeviceGenerationSHA256 {
		return ErrOpenRGBDeviceGenerationChanged
	}
	return nil
}
func (c *Controller) observeOpenRGBServingDeviceWitness(ctx context.Context, state ManagedServiceState) (openRGBExecWitness, error) {
	var witness openRGBExecWitness
	if state.InitialLaunch != nil {
		a, f, err := readOpenRGBInitialServing(state)
		if err != nil {
			return openRGBExecWitness{}, err
		}
		if f.State != managedUpdateExecFenceClaimed {
			return openRGBExecWitness{}, errors.New("initial OpenRGB claim is revoked")
		}
		witness = *f.OpenRGBWitness
		observed, err := c.observeOpenRGBInitialProcess(ctx, a)
		if err != nil {
			return openRGBExecWitness{}, err
		}
		if !sameOpenRGBInitialState(observed, state) {
			return openRGBExecWitness{}, errors.New("initial OpenRGB state changed during device proof")
		}
	} else if state.UpdateLaunch != nil {
		l := state.UpdateLaunch
		i := ManagedUpdateLaunchIntent{InstanceID: state.InstanceID, OwnershipTokenHash: state.OwnershipTokenHash, ContractDigest: l.Contract.Digest, Runtime: l.Runtime, NativeIntent: platform.ServiceIntent(l.NativeRef.Intent)}
		fences, err := newManagedUpdateExecStore(openRGBResourceName, l.OperationID, state.InstanceID)
		if err != nil {
			return openRGBExecWitness{}, err
		}
		f, err := fences.read()
		if err != nil {
			return openRGBExecWitness{}, err
		}
		witness, err = verifyOpenRGBConsumedClaim(f, i, l.Contract, l.NativeRef, state.Incarnation)
		if err != nil {
			return openRGBExecWitness{}, err
		}
		tokenPath, err := l.Runtime.tokenPath(l.OperationID, state.InstanceID)
		if err != nil {
			return openRGBExecWitness{}, err
		}
		token, err := readManagedUpdateToken(tokenPath)
		if err != nil {
			return openRGBExecWitness{}, err
		}
		base, err := renderManagedUpdateInvocation(l.Contract, l.Runtime, l.OperationID, state.InstanceID, token)
		if err != nil {
			return openRGBExecWitness{}, err
		}
		observation, err := newOpenRGBServingObservation(f, i, l.Contract, base, l.NativeRef, state.Incarnation)
		if err != nil {
			return openRGBExecWitness{}, err
		}
		if err := c.observeOpenRGBNativePayload(ctx, l.Contract.Artifact, observation); err != nil {
			return openRGBExecWitness{}, err
		}
	} else {
		return openRGBExecWitness{}, errors.New("OpenRGB process has no retained native device witness")
	}
	return witness, nil
}

func (c *Controller) observeOpenRGBInitialAdmission(ctx context.Context, state ManagedServiceState) (managedUpdateAdmissionSource, error) {
	a, f, err := readOpenRGBInitialServing(state)
	if err != nil {
		return managedUpdateAdmissionSource{}, err
	}
	if f.State != managedUpdateExecFenceClaimed {
		return managedUpdateAdmissionSource{}, errors.New("initial OpenRGB serving claim is revoked")
	}
	current, err := c.observeOpenRGBInitialProcess(ctx, a)
	if err != nil {
		return managedUpdateAdmissionSource{}, err
	}
	if !sameOpenRGBInitialState(current, state) {
		return managedUpdateAdmissionSource{}, errors.New("initial OpenRGB serving identity changed")
	}
	image, err := c.openOpenRGBRollbackEnvelope(ctx, a.Contract.Artifact)
	if err != nil {
		return managedUpdateAdmissionSource{}, err
	}
	defer image.Close()
	retained, err := retainManagedUpdateFile(ctx, c, openRGBResourceName, state.ArtifactVersion, state.ArtifactSHA256, image)
	if err != nil {
		return managedUpdateAdmissionSource{}, err
	}
	currentContract, err := freezeManagedUpdateServingConfig(state, "", a.Contract)
	if err != nil {
		return managedUpdateAdmissionSource{}, err
	}
	contract, err := currentContract.withArtifact(retained)
	if err != nil {
		return managedUpdateAdmissionSource{}, err
	}
	data, err := json.Marshal(a.Contract.Manifest)
	if err != nil {
		return managedUpdateAdmissionSource{}, err
	}
	return managedUpdateAdmissionSource{SourceRevision: fmt.Sprintf("%x", sha256.Sum256(data)), Process: state, Contract: contract, CredentialStateDir: filepath.Join(a.root(), "state")}, nil
}

func (s *ManagedServiceSupervisor) stopManagedInitialServing(ctx context.Context, state ManagedServiceState) error {
	if err := validateManagedInitialServing(state); err != nil {
		return err
	}
	store := &openRGBInitialStore{dir: filepath.Dir(s.statePath), write: writeResourceOwnedFileDurable}
	a, err := store.read(state.InitialLaunch.AttemptID)
	if err != nil {
		return err
	}
	fences, e := newManagedInitialExecStore(a.ID)
	if e != nil {
		return e
	}
	f, e := fences.read()
	if e != nil {
		return e
	}
	if !sameManagedExecIdentity(f, a.fence()) || f.Claim == nil || !f.Claim.Equal(state.InitialLaunch.NativeRef) {
		return errors.New("initial OpenRGB stop publication differs from retained native claim")
	}
	if err := retireOpenRGBInitial(ctx, store, a); err != nil {
		return err
	}
	if state.StoppedAt == nil {
		now := time.Now().UTC()
		state.StoppedAt = &now
	}
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return writeResourceOwnedFileDurable(s.statePath, data, 0600)
}

type openRGBInitialCleanupOwner struct {
	retireBoot func(context.Context, managedUpdateExecFence, *platform.ServiceRef) (bool, error)
	revoke     func(context.Context, managedUpdateExecFence) (managedUpdateExecFence, error)
	lookup     func(context.Context, platform.ServiceIntent) (platform.ServiceRef, error)
	absent     func(context.Context, platform.ServiceIntent) (bool, error)
	stop       func(context.Context, platform.ServiceRef) (platform.OperationStopReceipt, error)
	release    func(context.Context, platform.ServiceRef) error
	released   func(context.Context, platform.ServiceRef) (bool, error)
}

func retireOpenRGBInitial(ctx context.Context, store *openRGBInitialStore, a openRGBInitialAttempt) error {
	fences, err := newManagedInitialExecStore(a.ID)
	if err != nil {
		return err
	}
	return retireOpenRGBInitialWith(ctx, store, a, openRGBInitialCleanupOwner{retireBoot: func(ctx context.Context, f managedUpdateExecFence, current *platform.ServiceRef) (bool, error) {
		return retireOpenRGBNativeBoot(ctx, fences, f, current)
	}, revoke: fences.Revoke, lookup: platform.LookupOwnedService, absent: platform.OwnedServiceIntentAbsent, stop: platform.StopOwnedService, release: platform.ReleaseOwnedService, released: platform.OwnedServiceReleased})
}
func retireOpenRGBInitialWith(ctx context.Context, store *openRGBInitialStore, a openRGBInitialAttempt, o openRGBInitialCleanupOwner) error {
	// Revoke before observing absence or stopping: late native delivery must not
	// acquire exec permission after an ambiguous dispatch acknowledgement.
	f, err := o.revoke(ctx, a.fence())
	if err != nil {
		return err
	}
	ref, err := o.lookup(ctx, a.Intent)
	if o.retireBoot != nil {
		var current *platform.ServiceRef
		if err == nil {
			current = &ref
		} else if !errors.Is(err, platform.ErrOperationOutcomeLost) {
			return err
		}
		handled, e := o.retireBoot(ctx, f, current)
		if e != nil {
			return e
		}
		if handled {
			if a.State == openRGBInitialStopped {
				return nil
			}
			_, e = store.change(ctx, a, func(a *openRGBInitialAttempt) error {
				now := time.Now().UTC()
				a.State = openRGBInitialStopped
				a.StoppedAt = &now
				return nil
			})
			return e
		}
	}
	if err != nil {
		if a.QuiescentAt != nil && a.NativeRef != nil {
			released, e := o.released(ctx, *a.NativeRef)
			if e != nil {
				return e
			}
			if !released {
				return errors.New("initial native release remains unresolved")
			}
		} else {
			absent, e := o.absent(ctx, a.Intent)
			if e != nil || !absent {
				return errors.Join(err, e)
			}
		}
	} else {
		if f.Claim != nil && !ref.Equal(*f.Claim) || a.NativeRef != nil && !ref.Equal(*a.NativeRef) {
			return errors.New("initial OpenRGB cleanup found a different native invocation")
		}
		if a.QuiescentAt == nil {
			receipt, e := o.stop(ctx, ref)
			if e != nil {
				return e
			}
			if !receipt.NativeQuiescent {
				return errors.New("initial OpenRGB native stop remains unresolved")
			}
			// Persist exact quiescence before releasing native metadata. A lost
			// release reply can then be resolved against this retained identity.
			a, e = store.change(ctx, a, func(a *openRGBInitialAttempt) error {
				now := time.Now().UTC()
				a.State = openRGBInitialQuiescent
				a.StoppedAt = nil
				a.QuiescentAt = &now
				a.NativeRef = &ref
				return nil
			})
			if e != nil {
				return e
			}
		}
		released, e := o.released(ctx, ref)
		if e != nil {
			return e
		}
		if !released {
			if e = o.release(ctx, ref); e != nil {
				return e
			}
		}
	}
	if a.State == openRGBInitialStopped {
		return nil
	}
	_, err = store.change(ctx, a, func(a *openRGBInitialAttempt) error {
		now := time.Now().UTC()
		a.State = openRGBInitialStopped
		a.StoppedAt = &now
		return nil
	})
	return err
}

func (a *OpenRGBHeadless) ensureOpenRGBInitial(ctx context.Context, launch openRGBHeadlessLaunch, supervisor *ManagedServiceSupervisor) (backend lightingruntime.Backend, outcome error) {
	store, err := newOpenRGBInitialStore()
	if err != nil {
		return lightingruntime.Backend{}, err
	}
	var attempt openRGBInitialAttempt
	// Every uncertain exit retains admission and fences/drains the exact native
	// identity. Re-entering Ensure never repeats a prior submission.
	defer func() {
		if outcome == nil || attempt.ID == "" {
			return
		}
		latest, e := store.read(attempt.ID)
		if e != nil {
			outcome = errors.Join(outcome, e)
			return
		}
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
		defer cancel()
		retireErr := retireOpenRGBInitial(cleanup, store, latest)
		if retireErr == nil {
			published, e := supervisor.readState()
			if e == nil && published.InitialLaunch != nil && published.InitialLaunch.AttemptID == latest.ID {
				retireErr = supervisor.stopManagedInitialServing(cleanup, published)
			}
		}
		outcome = errors.Join(outcome, retireErr)
	}()
	id, err := store.current()
	if err == nil {
		attempt, err = store.read(id)
		if err != nil {
			return lightingruntime.Backend{}, err
		}
		if attempt.State != openRGBInitialStopped {
			if attempt.State == openRGBInitialPrepared || attempt.State == openRGBInitialQuiescent {
				return lightingruntime.Backend{}, errors.New("initial OpenRGB pending attempt requires cleanup before retry")
			}
			if attempt.Contract.Artifact != launch.policy.Artifact || attempt.Policy.ID != launch.policy.PolicyID {
				return lightingruntime.Backend{}, errors.New("pending OpenRGB launch differs from selected artifact or policy")
			}
			return a.waitOpenRGBInitial(ctx, launch, supervisor, store, attempt)
		}
	} else if !os.IsNotExist(err) {
		return lightingruntime.Backend{}, err
	}
	if err := ensureManagedServicePortsAvailable(launch.manifest); err != nil {
		return lightingruntime.Backend{}, err
	}
	attempt, err = a.controller.prepareOpenRGBInitialAttempt(ctx, launch)
	if err != nil {
		return lightingruntime.Backend{}, err
	}
	fences, err := newManagedInitialExecStore(attempt.ID)
	if err != nil {
		return lightingruntime.Backend{}, err
	}
	if _, err := fences.Prepare(ctx, attempt.fence()); err != nil {
		return lightingruntime.Backend{}, err
	}
	if err := store.prepare(ctx, attempt); err != nil {
		return lightingruntime.Backend{}, err
	}
	attempt, err = store.submit(ctx, attempt)
	if err != nil {
		return lightingruntime.Backend{}, err
	}
	if attempt.Runtime.NativeAccount != "" {
		server, e := a.controller.startOpenRGBNativeInitialAdmission(ctx, attempt, fences)
		if e != nil {
			return lightingruntime.Backend{}, e
		}
		defer func() { outcome = errors.Join(outcome, server.Close()) }()
	}
	if _, err := platform.StartOwnedService(ctx, attempt.Intent, attempt.wrapperSpec()); err != nil {
		return lightingruntime.Backend{}, fmt.Errorf("initial OpenRGB dispatch requires reconciliation: %w", err)
	}
	return a.waitOpenRGBInitial(ctx, launch, supervisor, store, attempt)
}
func (a *OpenRGBHeadless) waitOpenRGBInitial(ctx context.Context, launch openRGBHeadlessLaunch, supervisor *ManagedServiceSupervisor, store *openRGBInitialStore, attempt openRGBInitialAttempt) (lightingruntime.Backend, error) {
	seconds := managedServiceStartTimeoutSeconds(a.controller, launch.manifest)
	if seconds <= 0 {
		seconds = 60
	}
	ctx, cancel := context.WithTimeout(ctx, tuning.ManagedServiceConfiguredTimeout(time.Duration(seconds)*time.Second))
	defer cancel()
	for {
		state, err := a.controller.observeOpenRGBInitialProcess(ctx, attempt)
		if err == nil {
			err = probeOpenRGBHeadlessSDK(ctx, launch.ready)
		}
		if err == nil {
			repeated, e := a.controller.observeOpenRGBInitialProcess(ctx, attempt)
			if e != nil {
				return lightingruntime.Backend{}, e
			}
			if !sameOpenRGBInitialState(state, repeated) {
				return lightingruntime.Backend{}, errors.New("initial OpenRGB changed during SDK readiness")
			}
			current, e := supervisor.readState()
			if e != nil && !os.IsNotExist(e) {
				return lightingruntime.Backend{}, e
			}
			if e == nil && current.StoppedAt == nil && !sameOpenRGBInitialState(current, state) {
				return lightingruntime.Backend{}, errors.New("refuse to replace a different canonical OpenRGB owner")
			}
			data, e := json.Marshal(state)
			if e != nil {
				return lightingruntime.Backend{}, e
			}
			if e = writeResourceOwnedFileDurable(supervisor.statePath, data, 0600); e != nil {
				return lightingruntime.Backend{}, fmt.Errorf("initial OpenRGB publication needs reconciliation: %w", e)
			}
			if attempt.State != openRGBInitialServing {
				if _, e = store.change(ctx, attempt, func(a *openRGBInitialAttempt) error { a.State = openRGBInitialServing; return nil }); e != nil {
					return lightingruntime.Backend{}, e
				}
			}
			policy := launch.policy
			policy.Artifact.Path = state.ArtifactPath
			policy.PID = state.PID
			policy.Incarnation = state.Incarnation
			if a.verify == nil {
				return lightingruntime.Backend{}, ErrOpenRGBDevicePolicyUnavailable
			}
			if e = a.verify(ctx, policy); e != nil {
				return lightingruntime.Backend{}, e
			}
			return lightingruntime.Backend{Endpoint: launch.endpoint, Incarnation: state.Incarnation}, nil
		}
		if !errors.Is(err, errOpenRGBNotReady) {
			return lightingruntime.Backend{}, err
		}
		select {
		case <-ctx.Done():
			return lightingruntime.Backend{}, errors.Join(err, ctx.Err())
		case <-time.After(tuning.LifecyclePollInterval()):
		}
	}
}
