package resources

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	capacityapp "github.com/vrooli/vrooli/internal/app/capacity"
	capacityengine "github.com/vrooli/vrooli/internal/capacity"
)

func recoverWhisperOpenResidency(ctx context.Context, controller *Controller, manifest ResourceManifest, supervisor *ManagedServiceSupervisor) error {
	return recoverWhisperOpenResidencyWithService(ctx, controller, manifest, supervisor, capacityapp.Service{SourceRoot: controller.Root})
}

func recoverWhisperOpenResidencyWithService(ctx context.Context, controller *Controller, manifest ResourceManifest, supervisor *ManagedServiceSupervisor, service capacityapp.Service) error {
	started, err := service.ResidencyGetOpenForOwner(ctx, manifest.Name)
	if errors.Is(err, capacityengine.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	listing, err := service.List(ctx, capacityapp.ListRequest{OwnerID: manifest.Name, AllHistory: true})
	if err != nil {
		return fmt.Errorf("sample current Whisper claim for recovery: %w", err)
	}
	claim, terminal, err := whisperRecoveryClaim(listing, manifest.Name, started.ClaimID, started.ClaimGeneration)
	if err != nil {
		return err
	}
	observation, err := service.ResidencyObserveClaim(ctx, started.ClaimID)
	if err != nil {
		return fmt.Errorf("sample exact Whisper claim for recovery: %w", err)
	}
	if observation.Claim.Generation != claim.Generation || observation.Claim.Status != claim.Status {
		return fmt.Errorf("Whisper claim changed while recovery evidence was being sampled")
	}
	now := time.Now().UTC()
	observedBytes, observedAt, sampleOK := freshWhisperVRAMSample(observation, started.ClaimID, claim.Generation, started.Transition.StartedAt, now)
	if !sampleOK {
		return fmt.Errorf("current GPU attribution is not a fresh measured sample after the transition began")
	}
	transition := started.Transition
	transition.Footprint = capacityengine.ResidencyFootprint{Bytes: observedBytes, Source: capacityengine.FootprintSourceMeasured, ObservedAt: observedAt, Verified: true}
	transition.Observed = capacityengine.ResidencyPlacement{Known: false, Reason: "recovery has no verified serving placement"}
	transition.EffectsVerified = false
	transition.CompletedAt = nil
	_, running, statusErr := supervisor.Status()
	if statusErr != nil {
		return fmt.Errorf("observe Whisper managed-service state for recovery: %w", statusErr)
	}
	if running {
		placement, placementErr := observePlacement(ctx, controller, manifest)
		if placementErr != nil {
			return fmt.Errorf("observe Whisper placement for recovery: %w", placementErr)
		}
		if placement != nil {
			transition.Observed = placement.ObservedResidencyPlacement()
		}
	} else if observedBytes == 0 {
		transition.Observed = capacityengine.ResidencyPlacement{Backend: capacityengine.ResidencyBackendUnloaded, Known: true, Reason: "managed service is absent and fresh owner-attributed GPU use is zero"}
	}
	if transition.Observed.Known {
		completedAt := time.Now().UTC()
		transition.CompletedAt = &completedAt
		transition.EffectsVerified = true
	}
	req := capacityapp.ResidencyRequest{
		OwnerID:                manifest.Name,
		ClaimID:                started.ClaimID,
		ClaimGeneration:        started.ClaimGeneration,
		CurrentClaimGeneration: claim.Generation,
		Revision:               started.Revision,
		Transition:             transition,
	}
	var recovered capacityengine.ResidencyTransitionRecord
	if started.Transition.CompletedAt != nil {
		if !transition.Observed.Known || !transition.EffectsVerified {
			return fmt.Errorf("aged completed Whisper evidence lacks a fresh, verified placement for reconciliation")
		}
		recovered, err = service.ResidencyReconcile(ctx, req)
	} else {
		recovered, err = service.ResidencyRecover(ctx, req)
	}
	if err != nil {
		return fmt.Errorf("persist recovered or refreshed Whisper transition: %w", err)
	}
	if recovered.Transition.CompletedAt == nil || !recovered.Transition.EffectsVerified {
		return fmt.Errorf("Whisper recovery remains unresolved; capacity and lifecycle stay locked")
	}
	if recovered.Transition.Desired.Backend == capacityengine.ResidencyBackendUnloaded {
		if !recovered.Transition.VerifiedUnload() {
			if recovered.Transition.Observed.Known && recovered.Transition.Observed.Backend != capacityengine.ResidencyBackendUnloaded {
				// Recovery established that the prior unload did not happen and
				// captured the still-resident backend. Keep its existing claim; the
				// caller's current start/restart may proceed through normal admission.
				return nil
			}
			return fmt.Errorf("Whisper recovery did not verify model absence; capacity remains locked")
		}
		if terminal {
			// The exact historical claim already crossed its terminal generation.
			// Recovery records the physical outcome; it must not attempt a second
			// release or treat the old generation as active again.
			return nil
		}
		if _, err := service.ResidencyRelease(ctx, capacityapp.ResidencyReleaseRequest{
			OwnerID: manifest.Name, ClaimID: recovered.ClaimID, ClaimGeneration: recovered.ClaimGeneration,
			TransitionID: recovered.Transition.ID, Revision: recovered.Revision, Model: recovered.Transition.Model,
		}); err != nil {
			return fmt.Errorf("release recovered Whisper claim after verified unload: %w", err)
		}
	}
	return nil
}

func whisperRecoveryClaim(listing capacityapp.ListOutput, ownerID, claimID string, originalGeneration int64) (*capacityapp.ClaimView, bool, error) {
	var claim *capacityapp.ClaimView
	for i := range listing.Claims {
		candidate := &listing.Claims[i]
		if candidate.ClaimID == claimID {
			if claim != nil {
				return nil, false, fmt.Errorf("open transition has duplicate exact claim history")
			}
			claim = candidate
			continue
		}
		if candidate.OwnerID == ownerID && capacityengine.IsActiveClaimStatus(candidate.Status) {
			return nil, false, fmt.Errorf("open transition cannot establish owner placement while another resource claim is active")
		}
	}
	if claim == nil || claim.OwnerID != ownerID || claim.OwnerKind != capacityengine.OwnerKindResource || claim.ResourceKind != capacityengine.ResourceKindVRAM {
		return nil, false, fmt.Errorf("open transition has no exact historical resource VRAM claim")
	}
	terminal := !capacityengine.IsActiveClaimStatus(claim.Status)
	if claim.Generation != originalGeneration && !(terminal && claim.Generation == originalGeneration+1 && (claim.Status == capacityengine.StatusReleased || claim.Status == capacityengine.StatusExpired || claim.Status == capacityengine.StatusPreempted)) {
		return nil, false, fmt.Errorf("open transition claim generation changed unexpectedly from %d to %d", originalGeneration, claim.Generation)
	}
	return claim, terminal, nil
}

type whisperResidencyTransition struct {
	service capacityapp.Service
	request capacityapp.ResidencyRequest
	started capacityengine.ResidencyTransitionRecord
}

func beginWhisperCPUTransition(ctx context.Context, controller *Controller, manifest ResourceManifest) (*whisperResidencyTransition, error) {
	return beginWhisperCPUTransitionWithService(ctx, controller, manifest, capacityapp.Service{SourceRoot: controller.Root})
}

func beginWhisperCPUTransitionWithService(ctx context.Context, controller *Controller, manifest ResourceManifest, service capacityapp.Service) (*whisperResidencyTransition, error) {
	return beginWhisperResidencyTransition(ctx, controller, manifest, service, capacityengine.ResidencyBackendCPU, "resource_lifecycle_restart")
}

func beginWhisperUnloadTransition(ctx context.Context, controller *Controller, manifest ResourceManifest) (*whisperResidencyTransition, error) {
	return beginWhisperResidencyTransition(ctx, controller, manifest, capacityapp.Service{SourceRoot: controller.Root}, capacityengine.ResidencyBackendUnloaded, "idle_unload")
}

func beginWhisperResidencyTransition(ctx context.Context, controller *Controller, manifest ResourceManifest, service capacityapp.Service, desiredBackend, reason string) (*whisperResidencyTransition, error) {
	decl := manifest.EffectiveAcceleration()
	if decl == nil || decl.Claim == nil || decl.Claim.Profile == nil || !decl.Claim.Profile.Model.Complete() {
		return nil, fmt.Errorf("Whisper residency transition requires the resource's complete pinned model identity")
	}
	listing, err := service.List(ctx, capacityapp.ListRequest{OwnerID: manifest.Name, ActiveOnly: true})
	if err != nil {
		return nil, fmt.Errorf("read current Whisper capacity claim: %w", err)
	}
	if len(listing.Claims) != 1 {
		return nil, fmt.Errorf("Whisper residency transition requires one unambiguous active resource claim, got %d", len(listing.Claims))
	}
	claim := listing.Claims[0]
	if claim.OwnerID != manifest.Name || claim.OwnerKind != capacityengine.OwnerKindResource || claim.ResourceKind != capacityengine.ResourceKindVRAM || claim.Generation <= 0 || claim.Protected || claim.ActivityState != capacityengine.ActivityIdle {
		return nil, fmt.Errorf("Whisper capacity claim is not an idle, unprotected resource VRAM claim")
	}
	if claim.ObservedAt == nil || claim.ObservedBytes < 0 {
		return nil, fmt.Errorf("Whisper CPU transition requires measured pre-transition VRAM evidence")
	}
	observedAt, err := time.Parse(time.RFC3339, *claim.ObservedAt)
	if err != nil || observedAt.After(time.Now().UTC()) || time.Since(observedAt) > 2*time.Minute {
		return nil, fmt.Errorf("Whisper pre-transition VRAM evidence is missing, future-dated, or stale")
	}
	startedAt := time.Now().UTC()
	transition := capacityengine.ResidencyTransition{
		ID:      fmt.Sprintf("whisper-cpu-%d-%d", os.Getpid(), startedAt.UnixNano()),
		Model:   decl.Claim.Profile.Model,
		Desired: capacityengine.ResidencyPlacement{Backend: desiredBackend, Known: true, Reason: "resource lifecycle requested the declared placement"},
		// The resource CLI is directly callable, so ambient environment values
		// cannot establish narrower broker-pressure provenance.
		Reason:       reason,
		StartedAt:    startedAt,
		PreFootprint: capacityengine.ResidencyFootprint{Bytes: claim.ObservedBytes, Source: capacityengine.FootprintSourceMeasured, ObservedAt: observedAt, Verified: true},
	}
	req := capacityapp.ResidencyRequest{OwnerID: manifest.Name, ClaimID: claim.ClaimID, ClaimGeneration: claim.Generation, Transition: transition}
	started, err := service.ResidencyBegin(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("persist Whisper CPU transition before placement change: %w", err)
	}
	return &whisperResidencyTransition{service: service, request: req, started: started}, nil
}

func (t *whisperResidencyTransition) completeCPU(ctx context.Context, controller *Controller, manifest ResourceManifest, supervisor *ManagedServiceSupervisor) error {
	transition := t.request.Transition
	completedAt := time.Now().UTC()
	transition.CompletedAt = &completedAt
	transition.Observed = capacityengine.ResidencyPlacement{Known: false, Reason: "CPU placement has not yet been independently observed"}
	transition.EffectsVerified = false
	transition.Footprint = capacityengine.ResidencyFootprint{ObservedAt: completedAt}
	_, running, statusErr := supervisor.Status()
	if statusErr == nil && running {
		placement, placementErr := observePlacement(ctx, controller, manifest)
		observation, sampleErr := t.service.ResidencyObserveClaim(ctx, t.request.ClaimID)
		if placementErr == nil && placement != nil {
			transition.Observed = placement.ObservedResidencyPlacement()
		}
		if sampleErr == nil {
			if observed, observedAt, ok := freshWhisperVRAMSample(observation, t.request.ClaimID, t.request.ClaimGeneration, transition.StartedAt, completedAt); ok {
				transition.Footprint = capacityengine.ResidencyFootprint{Bytes: observed, Source: capacityengine.FootprintSourceMeasured, ObservedAt: observedAt, Verified: true}
			}
		}
		// The claim is for VRAM. Process VmRSS is host RAM and cannot prove that
		// the accelerator allocation disappeared, so only a fresh owner-attributed
		// zero-byte GPU sample verifies the CPU transition.
		transition.EffectsVerified = placementErr == nil && placement != nil && whisperCPUTransitionVerified(transition.Observed, transition.Footprint)
	}
	req := t.request
	req.Revision = t.started.Revision
	req.Transition = transition
	if _, err := t.service.ResidencyComplete(ctx, req); err != nil {
		return fmt.Errorf("persist Whisper CPU transition observation: %w", err)
	}
	if !transition.EffectsVerified {
		return fmt.Errorf("Whisper CPU placement or process footprint remains unverified; activity edge stays drained")
	}
	return nil
}

func stopWhisperForCapacityUnload(ctx context.Context, controller *Controller, manifest ResourceManifest, supervisor *ManagedServiceSupervisor) error {
	if err := controlWhisperActivityEdge(ctx, "drain"); err != nil {
		return fmt.Errorf("drain Whisper activity edge before unload: %w", err)
	}
	transition, err := beginWhisperUnloadTransition(ctx, controller, manifest)
	if err != nil {
		resumeErr := controlWhisperActivityEdge(ctx, "resume")
		return errors.Join(fmt.Errorf("begin durable Whisper unload transition: %w", err), resumeErr)
	}
	stopCtx, cancel := managedServiceStopContext(ctx, manifest)
	stopErr := stopManagedService(stopCtx, manifest, supervisor)
	cancel()
	if stopErr != nil {
		return fmt.Errorf("stop Whisper model server for verified unload: %w", stopErr)
	}
	if err := transition.completeUnload(ctx, supervisor); err != nil {
		return err
	}
	if err := controlWhisperActivityEdge(ctx, "resume"); err != nil {
		return fmt.Errorf("resume stable Whisper activity edge after verified unload: %w", err)
	}
	return nil
}

func (t *whisperResidencyTransition) completeUnload(ctx context.Context, supervisor *ManagedServiceSupervisor) error {
	transition := t.request.Transition
	completedAt := time.Now().UTC()
	transition.CompletedAt = &completedAt
	transition.Observed = capacityengine.ResidencyPlacement{Known: false, Reason: "Whisper server absence has not been verified"}
	transition.EffectsVerified = false
	transition.Footprint = capacityengine.ResidencyFootprint{ObservedAt: completedAt}
	_, running, statusErr := supervisor.Status()
	if statusErr != nil {
		return fmt.Errorf("verify Whisper server exit: %w", statusErr)
	}
	observation, sampleErr := t.service.ResidencyObserveClaim(ctx, t.request.ClaimID)
	if sampleErr != nil {
		return fmt.Errorf("sample owner-attributed GPU use after Whisper unload: %w", sampleErr)
	}
	if observed, observedAt, ok := freshWhisperVRAMSample(observation, t.request.ClaimID, t.request.ClaimGeneration, transition.StartedAt, completedAt); ok {
		transition.Footprint = capacityengine.ResidencyFootprint{Bytes: observed, Source: capacityengine.FootprintSourceMeasured, ObservedAt: observedAt, Verified: true}
		if !running && observed == 0 {
			transition.Observed = capacityengine.ResidencyPlacement{Backend: capacityengine.ResidencyBackendUnloaded, Known: true, Reason: "managed server is absent and fresh owner-attributed GPU use is zero"}
			transition.EffectsVerified = true
		}
	}
	req := t.request
	req.Revision = t.started.Revision
	req.Transition = transition
	completed, err := t.service.ResidencyComplete(ctx, req)
	if err != nil {
		return fmt.Errorf("persist Whisper unload observation: %w", err)
	}
	if !completed.Transition.VerifiedUnload() {
		return fmt.Errorf("Whisper model absence and measured zero-VRAM postcondition are unverified; edge remains drained")
	}
	if _, err := t.service.ResidencyRelease(ctx, capacityapp.ResidencyReleaseRequest{
		OwnerID: t.request.OwnerID, ClaimID: t.request.ClaimID, ClaimGeneration: t.request.ClaimGeneration,
		TransitionID: completed.Transition.ID, Revision: completed.Revision, Model: completed.Transition.Model,
	}); err != nil {
		return fmt.Errorf("release exact Whisper claim after verified unload: %w", err)
	}
	return nil
}

func whisperCPUTransitionVerified(placement capacityengine.ResidencyPlacement, footprint capacityengine.ResidencyFootprint) bool {
	return placement.Known && placement.Backend == capacityengine.ResidencyBackendCPU && footprint.Verified && footprint.Source == capacityengine.FootprintSourceMeasured && footprint.Bytes == 0
}

func freshWhisperVRAMSample(observation capacityapp.ResidencyObservation, claimID string, generation int64, startedAt, now time.Time) (int64, time.Time, bool) {
	claim := observation.Claim
	if claim.ClaimID != claimID || claim.Generation != generation || claim.OwnerID != "whisper" || claim.OwnerKind != capacityengine.OwnerKindResource || claim.ResourceKind != capacityengine.ResourceKindVRAM || observation.Bytes < 0 || observation.ObservedAt.IsZero() {
		return 0, time.Time{}, false
	}
	observedAt := observation.ObservedAt.UTC()
	if !observedAt.After(startedAt) || observedAt.After(now) || now.Sub(observedAt) > 2*time.Minute {
		return 0, time.Time{}, false
	}
	return observation.Bytes, observedAt, true
}
