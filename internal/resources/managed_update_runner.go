package resources

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/network"
	"github.com/vrooli/vrooli/internal/process"
	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

// AdvanceManagedServiceUpdate reconciles an already-admitted operation. Native
// dispatch occurs only immediately after the durable edge that admits it;
// reloaded starting phases observe, never redispatch. Serving roles survive the
// caller. The supervisor can resume this same request after caller loss.
func (c *Controller) AdvanceManagedServiceUpdate(ctx context.Context, resource, request string, abort bool) (ManagedServiceUpdate, error) {

	if resource == openRGBResourceName {
		return c.runOpenRGBPublicUpdate(ctx, func(ctx context.Context) (ManagedServiceUpdate, error) {
			return c.advanceManagedServiceUpdate(ctx, resource, request, abort)
		})
	}
	return c.advanceManagedServiceUpdate(ctx, resource, request, abort)
}

func (c *Controller) advanceManagedServiceUpdate(ctx context.Context, resource, request string, abort bool) (out ManagedServiceUpdate, outcome error) {
	if !managedUpdateResource.MatchString(resource) || !managedUpdateKey.MatchString(request) {
		return ManagedServiceUpdate{}, fmt.Errorf("invalid update selector")
	}
	ctx, cancel := context.WithTimeout(ctx, 180*time.Second)
	defer cancel()
	release, err := acquireManagedServiceLifecycleLock(ctx, resource)
	if err != nil {
		return ManagedServiceUpdate{}, err
	}
	defer release()
	s, err := newManagedUpdateStore(resource)
	if err != nil {
		return ManagedServiceUpdate{}, err
	}
	u, err := s.Lookup(request)
	if err != nil {
		return ManagedServiceUpdate{}, err
	}
	u, err = s.Confirm(ctx, request, u.Revision)
	if err != nil {
		return ManagedServiceUpdate{}, err
	}
	if abort && !u.Phase.Terminal() {
		u, err = s.Commit(ctx, u, UpdateBeginAbort)
		if err != nil {
			return u, err
		}
	}
	effects := c.managedUpdateEffectOwner()
	if effects.close != nil {
		defer func() { outcome = errors.Join(outcome, effects.close()) }()
	}
	if resource == openRGBResourceName && !u.Phase.Terminal() {
		if err := c.qualifyOpenRGBResumption(ctx, u); err != nil {
			return u, err
		}
	}
	fail := func(cause error, recovery bool) (ManagedServiceUpdate, error) {
		if errors.Is(cause, ErrManagedUpdateCommitUnknown) || errors.Is(cause, ErrManagedUpdateRevision) {
			return u, cause
		}
		if ctx.Err() != nil {
			return u, ctx.Err()
		}
		current, err := s.Lookup(request)
		if err != nil {
			return u, err
		}
		current, err = s.Confirm(ctx, request, current.Revision)
		if err != nil {
			return u, err
		}
		u = current
		u.Error = cause.Error()
		if len(u.Error) > 4096 {
			u.Error = u.Error[:4096]
		}
		event := UpdateFailure
		if recovery {
			event = UpdateRecoveryFailed
		}
		return s.Commit(ctx, u, event)
	}
	for step := 0; step < 32; step++ {
		if err := ctx.Err(); err != nil {
			return u, err
		}
		if u.Phase.Terminal() {
			return u, effects.release(ctx, u)
		}
		if resource == openRGBResourceName {
			if err := c.qualifyOpenRGBResumption(ctx, u); err != nil {
				return u, err
			}
		}
		switch u.Phase {
		case UpdatePrepared:
			if exclusiveStopStart(u) {
				u.OriginalProof, err = effects.original(ctx, u)
				if err != nil {
					u, err = fail(err, false)
					break
				}
				instance := "mu-" + updateOperationRoleDigest(u.ID, managedUpdateRolePrimary)
				intent, e := effects.prepare(ctx, u, u.Target, instance, u.Target.Manifest.Ports[0].Host)
				if e != nil {
					u, err = fail(e, false)
					break
				}
				u.Primary.Intent = &intent
				u, err = s.Commit(ctx, u, UpdateStopPrimary)
			} else {
				u, err = effects.candidate(ctx, u, s)
				if err != nil {
					u, err = fail(err, false)
				}
			}
		case UpdateRollbackProbeStarting, UpdateRollbackProbeVerifying, UpdateCandidateStarting, UpdateCandidateVerifying:
			u, err = effects.candidate(ctx, u, s)
			if err != nil {
				u, err = fail(err, false)
			}
		case UpdateCandidateReady:
			u.Candidate, err = effects.observe(ctx, u, u.Candidate, u.Target)
			if err != nil {
				u, err = fail(err, false)
				break
			}
			instance := "mu-" + updateOperationRoleDigest(u.ID, managedUpdateRolePrimary)
			intent, e := effects.prepare(ctx, u, u.Target, instance, u.Target.Manifest.Ports[0].Host)
			if e != nil {
				u, err = fail(e, false)
				break
			}
			u.Primary.Intent = &intent
			u, err = s.Commit(ctx, u, UpdateStopPrimary)
		case UpdatePrimaryStopping:
			if e := effects.reserve(ctx, u); e != nil {
				return u, e
			}
			if resource == openRGBResourceName {
				if e := c.openRGBPublicOwners().quiesce(ctx); e != nil {
					u, err = fail(e, false)
					break
				}
			}
			if _, e := effects.stopOriginal(ctx, u); e != nil {
				u, err = fail(e, false)
				break
			}
			if e := effects.portUnbound(u.Target.Manifest.Ports[0].Host); e != nil {
				u, err = fail(e, false)
				break
			}
			u.PrimaryStopVerified = true
			u, err = s.Commit(ctx, u, UpdatePrimaryStopped)
			if err == nil {
				_, err = effects.start(ctx, u.Primary.Intent.NativeIntent, managedUpdateWrapperSpec(u, *u.Primary.Intent))
				if err != nil {
					u, err = fail(err, false)
				}
			}
		case UpdatePrimaryStarting:
			u.Primary, err = effects.wait(ctx, u, u.Primary, u.Target)
			if err != nil {
				u, err = fail(err, false)
			} else {
				u, err = s.Commit(ctx, u, UpdatePrimaryStarted)
			}
		case UpdatePrimaryVerifying:
			u.Primary, err = effects.observe(ctx, u, u.Primary, u.Target)
			if err != nil {
				u, err = fail(err, false)
			} else {
				u, err = s.Commit(ctx, u, UpdatePrimaryVerified)
			}
		case UpdateCommitting:
			if e := effects.publish(ctx, u, managedUpdateRolePrimary); e != nil {
				if errors.Is(e, ErrManagedUpdateCommitUnknown) || errors.Is(e, ErrManagedUpdateConflict) {
					return u, e
				}
				u, err = fail(e, false)
				break
			}
			if e := effects.cleanup(ctx, u, u.Candidate); e != nil {
				return u, e
			}
			if e := effects.cleanup(ctx, u, u.RollbackProbe); e != nil {
				return u, e
			}
			u.CandidateQuiescent = true
			u.RollbackProbeQuiescent = u.RollbackProbe.Intent != nil
			u.Primary, err = effects.observe(ctx, u, u.Primary, u.Target)
			if err != nil {
				u, err = fail(err, false)
			} else {
				u, err = s.Commit(ctx, u, UpdateFinishSuccess)
			}
		case UpdatePreserving, UpdatePreserveCommitting:
			proof, e := effects.original(ctx, u)
			if e != nil {
				u, err = fail(e, true)
				break
			}
			u.OriginalProof = proof
			if u.Phase == UpdatePreserving {
				u, err = s.Commit(ctx, u, UpdateOriginalVerified)
				break
			}
			if e := effects.cleanup(ctx, u, u.Candidate); e != nil {
				return u, e
			}
			if e := effects.cleanup(ctx, u, u.RollbackProbe); e != nil {
				return u, e
			}
			u.CandidateQuiescent = true
			u.RollbackProbeQuiescent = u.RollbackProbe.Intent != nil
			u.OriginalProof, err = effects.original(ctx, u)
			if err != nil {
				u, err = fail(err, true)
			} else {
				u, err = s.Commit(ctx, u, UpdateFinishPreserved)
			}
		case UpdateRestoring:
			if resource == openRGBResourceName {
				if e := c.openRGBPublicOwners().quiesce(ctx); e != nil {
					return u, e
				}
			}
			if !u.PrimaryStopVerified {
				// Do not convert uncertain original ownership into permission to
				// stop it. Reconcile a prior admitted stop against the exact scope.
				quiescent, e := effects.originalQuiescent(ctx, u)
				if e != nil {
					return u, fmt.Errorf("original stop remains unresolved: %v", e)
				}
				if !quiescent {
					if _, e := effects.stopRestoration(ctx, u); e != nil {
						return u, e
					}
					quiescent, e = effects.originalQuiescent(ctx, u)
					if e != nil || !quiescent {
						return u, fmt.Errorf("original stop recovery remains unresolved: %v", e)
					}
				}
				u.PrimaryStopVerified = true
			}
			if e := effects.cleanup(ctx, u, u.Primary); e != nil {
				return u, e
			}
			u.PrimaryQuiescent = true
			if u.Restored.Intent != nil {
				if e := effects.cleanup(ctx, u, u.Restored); e != nil {
					return u, e
				}
				if len(u.RetiredRestores) >= managedUpdateRestoreAttempts-1 {
					return u, fmt.Errorf("restoration attempt limit reached; shadow services retained")
				}
				q := &ManagedUpdateQuiescence{InstanceID: u.Restored.Intent.InstanceID, OwnershipTokenHash: u.Restored.Intent.OwnershipTokenHash, ObservedAt: time.Now().UTC(), NativeAbsent: true, ProcessesAbsent: true, ListenerAbsent: true}
				if u.Restored.State != nil {
					q.PID = u.Restored.State.PID
					q.Incarnation = u.Restored.State.Incarnation
				}
				u.Restored.Quiescence = q
				u, err = s.Commit(ctx, u, UpdateRetireRestore)
				if err != nil {
					break
				}
			}
			if e := effects.portUnbound(u.Rollback.Manifest.Ports[0].Host); e != nil {
				return u, e
			}
			instance := "mu-" + updateOperationRoleDigest(u.ID, "restore-"+strconv.Itoa(len(u.RetiredRestores)))
			intent, e := effects.prepare(ctx, u, u.Rollback, instance, u.Rollback.Manifest.Ports[0].Host)
			if e != nil {
				return u, e
			}
			u.Restored.Intent = &intent
			u, err = s.Commit(ctx, u, UpdateLaunchRestore)
			if err == nil {
				_, err = effects.start(ctx, intent.NativeIntent, managedUpdateWrapperSpec(u, intent))
				if err != nil {
					u, err = fail(err, true)
				}
			}
		case UpdateRestoreStarting:
			u.Restored, err = effects.wait(ctx, u, u.Restored, u.Rollback)
			if err != nil {
				u, err = fail(err, true)
			} else {
				u, err = s.Commit(ctx, u, UpdateRestoreStarted)
			}
		case UpdateRestoreVerifying:
			u.Restored, err = effects.observe(ctx, u, u.Restored, u.Rollback)
			if err != nil {
				u, err = fail(err, true)
			} else {
				u, err = s.Commit(ctx, u, UpdateRestoreVerified)
			}
		case UpdateRestoreCommitting:
			if e := effects.publish(ctx, u, managedUpdateRoleRestored); e != nil {
				return u, e
			}
			if e := effects.cleanup(ctx, u, u.Candidate); e != nil {
				return u, e
			}
			if e := effects.cleanup(ctx, u, u.RollbackProbe); e != nil {
				return u, e
			}
			u.CandidateQuiescent = true
			u.RollbackProbeQuiescent = u.RollbackProbe.Intent != nil
			u.Restored, err = effects.observe(ctx, u, u.Restored, u.Rollback)
			if err != nil {
				u, err = fail(err, true)
			} else {
				u, err = s.Commit(ctx, u, UpdateFinishRestored)
			}
		case UpdateRecoveryNeeded:
			if u.RecoveryFrom != UpdateRestoring || len(u.RetiredRestores) >= managedUpdateRestoreAttempts-1 {
				return u, fmt.Errorf("recovery remains unresolved; serving/shadow identities retained")
			}
			u, err = s.Commit(ctx, u, UpdateRetryRecovery)
		default:
			return u, fmt.Errorf("unhandled managed update phase %s", u.Phase)
		}
		if err != nil {
			return u, err
		}
	}
	return u, fmt.Errorf("update reconciliation step bound reached; resume the same request")
}

func managedUpdatePortUnbound(port int) error {
	inspection := network.InspectPortListeners(port)
	if !inspection.Inspection.Available || len(inspection.Listeners) != 0 {
		return fmt.Errorf("canonical service port is occupied or ownership inspection is unavailable")
	}
	listener, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		return fmt.Errorf("canonical service port cannot be bound: %w", err)
	}
	return listener.Close()
}

func cleanupManagedUpdateRole(ctx context.Context, u ManagedServiceUpdate, p ManagedUpdateProcess) error {
	if p.Intent == nil {
		return nil
	}
	i := *p.Intent
	fences, err := newManagedUpdateExecStore(u.Resource, u.ID, i.InstanceID)
	if err != nil {
		return err
	}
	f, err := fences.Revoke(ctx, managedUpdateExecFence{SchemaVersion: managedUpdateExecFenceSchemaVersion, Resource: u.Resource, OperationID: u.ID, Intent: i, State: managedUpdateExecFencePrepared})
	if err != nil {
		return err
	}
	known := p.NativeRef
	if known == nil {
		known = f.Claim
	}
	current, lookupErr := platform.LookupOwnedService(ctx, i.NativeIntent)
	if u.Resource == openRGBResourceName {
		var observed *platform.ServiceRef
		if lookupErr == nil {
			observed = &current
		} else if !errors.Is(lookupErr, platform.ErrOperationOutcomeLost) {
			return lookupErr
		}
		handled, e := retireOpenRGBNativeBoot(ctx, fences, f, observed)
		if e != nil {
			return e
		}
		if handled {
			return fences.recordNativeRetired(ctx, f)
		}
	}
	if lookupErr == nil {
		if known != nil && known.NativeInvocationID != current.NativeInvocationID {
			return fmt.Errorf("%w: cleanup native invocation changed", ErrManagedUpdateConflict)
		}
		if known != nil {
			current = *known
		}
		receipt, err := platform.StopOwnedService(ctx, current)
		if err != nil {
			return err
		}
		if !receipt.NativeQuiescent {
			return fmt.Errorf("role native tree is not quiescent")
		}
		if err := platform.ReleaseOwnedService(ctx, current); err != nil {
			return err
		}
	} else {
		if !errors.Is(lookupErr, platform.ErrOperationOutcomeLost) {
			return lookupErr
		}
		absent, err := platform.OwnedServiceIntentAbsent(ctx, i.NativeIntent)
		if err != nil {
			return err
		}
		if !absent {
			return fmt.Errorf("role native unit absence is unconfirmed")
		}
		if known != nil {
			released, err := managedUpdateNativeReleased(ctx, *known, p.State)
			if err != nil {
				return err
			}
			if !released {
				return fmt.Errorf("role process absence is unconfirmed")
			}
		}
	}
	if u.Resource == openRGBResourceName {
		return fences.recordNativeRetired(ctx, f)
	}
	return nil
}

func observeManagedUpdateOriginal(ctx context.Context, u ManagedServiceUpdate) (*ManagedUpdateProof, error) {
	if u.Original.InitialLaunch != nil {
		a, _, err := readOpenRGBInitialServing(u.Original)
		if err != nil {
			return nil, err
		}
		owner := &Controller{Root: a.Runtime.Root, Home: a.Runtime.Home}
		state, err := owner.observeOpenRGBInitialProcess(ctx, a)
		if err != nil || !sameOpenRGBInitialState(state, u.Original) {
			return nil, errors.Join(errors.New("initial OpenRGB original changed during update qualification"), err)
		}
		return &ManagedUpdateProof{PID: state.PID, Incarnation: state.Incarnation, InstanceID: state.InstanceID, ArtifactSHA256: state.ArtifactSHA256, ObservedAt: time.Now().UTC(), OwnedListener: true, Ready: true}, nil
	}
	if u.Original.UpdateLaunch != nil {
		old, p, contract, err := managedUpdateRecordedOriginal(u)
		if err != nil {
			return nil, err
		}
		observed, err := observeManagedUpdateProcess(ctx, old, p, contract)
		return observed.Proof, err
	}
	if u.OriginalScope == nil {
		return nil, fmt.Errorf("original verification requires an exact native scope")
	}
	before, err := platform.InspectServiceScope(ctx, *u.OriginalScope)
	if err != nil {
		return nil, err
	}
	if !before.RootAlive || !before.TreePopulated {
		return nil, fmt.Errorf("original native service is not running")
	}
	if err := verifyManagedServiceOwnership(u.Original); err != nil {
		return nil, err
	}
	image, err := platform.OpenProcessExecutable(u.Original.PID)
	if err != nil {
		return nil, err
	}
	digest, err := managedUpdateImageDigest(ctx, image)
	_ = image.Close()
	if err != nil {
		return nil, err
	}
	if digest != u.Rollback.Artifact.SHA256 {
		return nil, fmt.Errorf("original image differs from retained rollback")
	}
	port := u.Rollback.Manifest.Ports[0].Host
	if err := managedUpdateOwnedListener(port, u.Original.PID); err != nil {
		return nil, err
	}
	key := resourcedeployment.ServicePortEnvironmentName(u.Rollback.Manifest.ManagedService.UpdatePolicy.CandidatePort)
	values := map[string]string{key: strconv.Itoa(port)}
	client, err := observeManagedUpdateReadiness(ctx, u.Rollback, port, values, nil)
	if err != nil {
		return nil, fmt.Errorf("original readiness failed: %w", err)
	}
	defer client.CloseIdleConnections()
	metrics, err := observeManagedUpdateMetrics(ctx, client, port, u.Rollback)
	if err != nil {
		return nil, err
	}
	incarnation, err := process.Incarnation(u.Original.PID)
	if err != nil || incarnation != u.Original.Incarnation {
		return nil, fmt.Errorf("original incarnation changed during verification")
	}
	after, err := platform.InspectServiceScope(ctx, *u.OriginalScope)
	if err != nil || !after.RootAlive || !after.TreePopulated {
		return nil, fmt.Errorf("original native owner changed during verification")
	}
	return &ManagedUpdateProof{PID: u.Original.PID, Incarnation: incarnation, InstanceID: u.Original.InstanceID, ArtifactSHA256: digest, ObservedAt: time.Now().UTC(), OwnedListener: true, Ready: true, Metrics: metrics}, nil
}
