// Package releases owns the iOS adapter for the provider-neutral durable
// validation matrix. Target discovery and journey execution are injected by
// the composition root so this domain does not own device-control or bridge
// policy.
package releases

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
	validationmatrix "github.com/vrooli/vrooli/packages/delivery-ramp-go/validationmatrix"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
)

type Catalog struct {
	Probe   deliveryramp.Prober
	Journey validationmatrix.JourneySelection
}

func (c Catalog) Resolve(ctx context.Context, scenario string) (validationmatrix.CatalogSnapshot, error) {
	if strings.TrimSpace(scenario) == "" {
		return validationmatrix.CatalogSnapshot{}, fmt.Errorf("iOS validation scenario is required")
	}
	if c.Probe == nil {
		return validationmatrix.CatalogSnapshot{}, fmt.Errorf("iOS target prober is not configured")
	}
	inventory, err := c.Probe.Probe(ctx, deliveryramp.ProbeRequest{})
	if err != nil {
		return validationmatrix.CatalogSnapshot{}, err
	}
	journey := c.Journey
	if strings.TrimSpace(journey.JourneyID) == "" {
		return validationmatrix.CatalogSnapshot{}, fmt.Errorf("iOS validation journey is not configured")
	}
	targets := make([]validationmatrix.TargetSelection, 0, len(inventory.Targets))
	for _, target := range inventory.Targets {
		reason := strings.TrimSpace(target.Reason)
		descriptor := &domainv1.ValidationTargetDescriptor{
			TargetId:     target.ID,
			DisplayName:  target.Label,
			Available:    target.Available,
			Capabilities: capabilityEnums(target.Capabilities),
		}
		if reason != "" {
			descriptor.Reason = &reason
		}
		kind := validationmatrix.TargetLocal
		if target.Transport.Kind == deliveryramp.TransportBridge {
			kind = validationmatrix.TargetBridge
		}
		targets = append(targets, validationmatrix.TargetSelection{Descriptor: descriptor, Kind: kind})
	}
	return validationmatrix.CatalogSnapshot{Journeys: []validationmatrix.JourneySelection{journey}, Targets: targets}, nil
}

type JourneyRunner func(context.Context, deliveryramp.DriverRequest) (deliveryramp.JourneyResult, error)

type Executor struct {
	JourneyPlan deliveryramp.JourneyPlan
	RunJourney  JourneyRunner
}

func (e Executor) Execute(ctx context.Context, request validationmatrix.CellRequest) validationmatrix.CellResult {
	if request.Cell == nil || request.Target == nil {
		return unavailable("iOS matrix cell has no target descriptor")
	}
	if !request.Target.GetAvailable() {
		reason := strings.TrimSpace(request.Target.GetReason())
		if reason == "" {
			reason = "iOS target is unavailable"
		}
		return unavailable(reason)
	}
	if e.RunJourney == nil {
		return unavailable("iOS journey runner is not configured")
	}
	plan := e.JourneyPlan
	if plan.ID == "" {
		return unavailable("iOS validation journey plan is not configured")
	}
	result, err := e.RunJourney(ctx, deliveryramp.DriverRequest{
		RunID:    request.RunID,
		Cell:     deliveryramp.Cell{ID: request.Cell.GetCellId(), Target: targetFromDescriptor(request.Target), ProfileID: request.Cell.GetEnvironmentProfile().String(), Required: request.Cell.GetRequired()},
		Artifact: deliveryramp.Artifact{ImmutableRef: request.ArtifactDigest, LocalPath: request.ArtifactPath, Kind: "ios-artifact"},
		Plan:     plan,
	})
	if err != nil {
		return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_FAILED, Reason: err.Error(), Evidence: journeyEvidence(result, request)}
	}
	switch result.Disposition {
	case deliveryramp.DispositionPass:
		return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_PASS, Reason: "iOS journey completed", Evidence: journeyEvidence(result, request)}
	case deliveryramp.DispositionDegraded:
		return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_DEGRADED, Reason: reasonFor(result), Evidence: journeyEvidence(result, request)}
	case deliveryramp.DispositionUnavailable:
		return unavailableWithEvidence(reasonFor(result), journeyEvidence(result, request))
	default:
		return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_FAILED, Reason: reasonFor(result), Evidence: journeyEvidence(result, request)}
	}
}

func unavailable(reason string) validationmatrix.CellResult {
	return unavailableWithEvidence(reason, nil)
}

func unavailableWithEvidence(reason string, evidence []*domainv1.LayeredEvidence) validationmatrix.CellResult {
	return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_UNAVAILABLE, Reason: strings.TrimSpace(reason), Evidence: evidence}
}

func reasonFor(result deliveryramp.JourneyResult) string {
	if strings.TrimSpace(result.DegradedReason) != "" {
		return result.DegradedReason
	}
	for _, step := range result.Steps {
		if strings.TrimSpace(step.Error) != "" {
			return step.Error
		}
	}
	return string(result.Disposition)
}

func journeyEvidence(result deliveryramp.JourneyResult, request validationmatrix.CellRequest) []*domainv1.LayeredEvidence {
	value := fmt.Sprintf("target=%s journey=%s run=%s", request.Target.GetTargetId(), request.Journey.JourneyID, request.RunID)
	mediaType := "text/plain"
	return []*domainv1.LayeredEvidence{{
		Kind:       domainv1.LayeredEvidence_KIND_MACHINE_ASSERTION,
		EvidenceId: "assertion-" + request.Cell.GetCellId(),
		Uri:        "validation://ios/" + request.Cell.GetCellId(),
		Sha256:     hashText(value),
		MediaType:  &mediaType,
		Redacted:   true,
	}}
}

func targetFromDescriptor(descriptor *domainv1.ValidationTargetDescriptor) deliveryramp.Target {
	targetID := descriptor.GetTargetId()
	deviceKind, mode := "host", "xcode"
	if strings.Contains(strings.ToLower(targetID), "simulator") {
		deviceKind, mode = "emulator", "simulator"
	}
	return deliveryramp.Target{ID: targetID, Label: descriptor.GetDisplayName(), Platform: "ios", Available: descriptor.GetAvailable(), DeviceKind: deviceKind, Mode: mode, Capabilities: capabilityNames(descriptor.GetCapabilities())}
}

func capabilityEnums(values []string) []domainv1.ValidationTargetCapability {
	result := make([]domainv1.ValidationTargetCapability, 0, len(values))
	for _, value := range values {
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "xcodebuild", "simctl", "macos-bridge-node":
			result = append(result, domainv1.ValidationTargetCapability_VALIDATION_TARGET_CAPABILITY_NATIVE_WINDOW)
		case "ios-webdriver":
			result = append(result, domainv1.ValidationTargetCapability_VALIDATION_TARGET_CAPABILITY_ELECTRON_CDP)
		}
	}
	return result
}

func capabilityNames(values []domainv1.ValidationTargetCapability) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, value.String())
	}
	return result
}

func hashText(value string) string {
	hash := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(hash[:])
}

var (
	_ validationmatrix.CatalogResolver = Catalog{}
	_ validationmatrix.CellTransport   = Executor{}
)
