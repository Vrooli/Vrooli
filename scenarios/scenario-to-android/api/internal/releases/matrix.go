// Package releases owns the Android adapter for the provider-neutral
// durable validation matrix. Target discovery and journey execution are
// injected by the composition root so this domain does not import sibling
// product domains.
package releases

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/api-core/discovery"
	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
	validationmatrix "github.com/vrooli/vrooli/packages/delivery-ramp-go/validationmatrix"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
)

type Catalog struct {
	Probe    deliveryramp.Prober
	Journey  validationmatrix.JourneySelection
	Journeys []validationmatrix.JourneySelection
}

func (c Catalog) Resolve(ctx context.Context, scenario string) (validationmatrix.CatalogSnapshot, error) {
	if strings.TrimSpace(scenario) == "" {
		return validationmatrix.CatalogSnapshot{}, fmt.Errorf("Android validation scenario is required")
	}
	inventory, err := c.Probe.Probe(ctx, deliveryramp.ProbeRequest{RequiredCapability: []string{
		deliveryramp.CapabilityDeviceControl,
		deliveryramp.CapabilityScreenRecording,
	}})
	if err != nil {
		return validationmatrix.CatalogSnapshot{}, err
	}
	journeys := append([]validationmatrix.JourneySelection(nil), c.Journeys...)
	if len(journeys) == 0 && strings.TrimSpace(c.Journey.JourneyID) != "" {
		journeys = []validationmatrix.JourneySelection{c.Journey}
	}
	if len(journeys) == 0 {
		return validationmatrix.CatalogSnapshot{}, fmt.Errorf("Android release catalog journey is not configured")
	}
	deviceControlEmulator := false
	for _, target := range inventory.Targets {
		if target.DeviceKind == "emulator" && target.ID != "android:emulator:local" {
			deviceControlEmulator = true
			break
		}
	}
	targets := make([]validationmatrix.TargetSelection, 0, len(inventory.Targets))
	for _, target := range inventory.Targets {
		// The generic local-emulator probe describes host readiness. Once
		// device-control has promoted the actual emulator, that synthetic
		// record is not executable and would create a matrix cell that cannot
		// acquire a device-control lease.
		if deviceControlEmulator && target.ID == "android:emulator:local" {
			continue
		}
		if !deviceControlEmulator && target.ID == "android:emulator:local" {
			target.Available = false
			target.Reason = "android-sdk is ready, but device-control did not report a running local emulator"
			target.NextAction = "start the governed AVD and probe again"
		}
		descriptor := &domainv1.ValidationTargetDescriptor{TargetId: target.ID, DisplayName: target.Label, Available: target.Available}
		if target.Reason != "" {
			reason := target.Reason
			descriptor.Reason = &reason
		}
		for _, capability := range target.Capabilities {
			if mapped, ok := capabilityEnum(capability); ok {
				descriptor.Capabilities = append(descriptor.Capabilities, mapped)
			}
		}
		kind := validationmatrix.TargetBridge
		if target.Transport.Kind == deliveryramp.TransportLocal {
			kind = validationmatrix.TargetLocal
		}
		targets = append(targets, validationmatrix.TargetSelection{Descriptor: descriptor, Kind: kind})
	}
	return validationmatrix.CatalogSnapshot{
		Journeys: journeys,
		Targets:  targets,
	}, nil
}

type Executor struct {
	HTTP             *http.Client
	JourneyPlan      deliveryramp.JourneyPlan
	JourneyPlans     map[string]deliveryramp.JourneyPlan
	ResolveTransport func(context.Context, string, string, *http.Client) string
	RunJourney       func(context.Context, deliveryramp.DriverRequest, string, string, string, *http.Client) (deliveryramp.JourneyResult, error)
}

const defaultAndroidPackageName = "com.vrooli.generated.app"

func (e Executor) Execute(ctx context.Context, request validationmatrix.CellRequest) validationmatrix.CellResult {
	if request.Cell == nil || request.Target == nil || !request.Target.GetAvailable() {
		return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_UNAVAILABLE, Reason: "Android target is unavailable"}
	}
	artifact, err := androidArtifact(request.ArtifactPath, request.ArtifactDigest)
	if err != nil {
		return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_UNAVAILABLE, Reason: err.Error()}
	}
	if request.Metadata != nil {
		if packageName := strings.TrimSpace(request.Metadata["package_name"]); packageName != "" {
			artifact.Metadata["package_name"] = packageName
		}
		if scenarioName := strings.TrimSpace(request.Metadata["scenario_name"]); scenarioName != "" {
			artifact.Metadata["scenario_name"] = scenarioName
		}
		if profileID := strings.TrimSpace(request.Metadata["auth_profile_id"]); profileID != "" {
			artifact.Metadata["auth_profile_id"] = profileID
		}
		for _, key := range []string{"immutable_ref", "version_name", "version_code", "signing_identity", "signing", "install_mode"} {
			if value := strings.TrimSpace(request.Metadata[key]); value != "" {
				artifact.Metadata[key] = value
			}
		}
		if immutableRef := strings.TrimSpace(request.Metadata["immutable_ref"]); immutableRef != "" {
			if !validImmutableRefForDigest(immutableRef, artifact.Checksum) {
				return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_UNAVAILABLE, Reason: "Android immutable reference does not match the matrix artifact digest"}
			}
			artifact.ImmutableRef = immutableRef
		}
	}
	if request.Metadata == nil {
		request.Metadata = make(map[string]string)
	}
	if request.Metadata["package_name"] == "" {
		request.Metadata["package_name"] = artifact.Metadata["package_name"]
	}
	deviceURL, err := resolveURL(ctx, "device-control", os.Getenv("DEVICE_CONTROL_URL"))
	if err != nil {
		return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_UNAVAILABLE, Reason: err.Error()}
	}
	deviceTransport := transportForTarget(request.Target)
	if e.ResolveTransport != nil {
		deviceTransport = e.ResolveTransport(ctx, deviceURL, request.Target.GetTargetId(), e.HTTP)
	}
	basURL, err := resolveURL(ctx, "browser-automation-studio", os.Getenv("BAS_URL"))
	if err != nil {
		return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_UNAVAILABLE, Reason: err.Error()}
	}
	client := e.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	journeyPlan := e.JourneyPlan
	if selected, ok := e.JourneyPlans[request.Journey.JourneyID]; ok {
		journeyPlan = selected
	}
	if journeyPlan.ID == "" || (request.Journey.JourneyID != "" && journeyPlan.ID != request.Journey.JourneyID) {
		return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_UNAVAILABLE, Reason: "Android journey plan is not configured for the selected journey"}
	}
	requestToRun := deliveryramp.DriverRequest{RunID: request.RunID, Cell: deliveryramp.Cell{ID: request.Cell.GetCellId(), Target: targetFromDescriptor(request.Target), ProfileID: request.Cell.GetEnvironmentProfile().String(), Required: request.Cell.GetRequired()}, Artifact: artifact, Plan: journeyPlan}
	if e.RunJourney == nil {
		return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_UNAVAILABLE, Reason: "Android journey runner is not configured"}
	}
	result, runErr := e.RunJourney(ctx, requestToRun, deviceURL, basURL, deviceTransport, client)
	report := map[string]string{}
	if result.ReviewRecordingPath != "" {
		report["review_recording_path"] = result.ReviewRecordingPath
	}
	if runErr != nil {
		return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_FAILED, Reason: runErr.Error(), Evidence: journeyEvidence(result, request), Report: report}
	}
	evidence := journeyEvidence(result, request)
	if result.Disposition != deliveryramp.DispositionPass {
		disposition := domainv1.ValidationDisposition_VALIDATION_DISPOSITION_FAILED
		if result.Disposition == deliveryramp.DispositionUnavailable {
			disposition = domainv1.ValidationDisposition_VALIDATION_DISPOSITION_UNAVAILABLE
		} else if result.Disposition == deliveryramp.DispositionDegraded {
			disposition = domainv1.ValidationDisposition_VALIDATION_DISPOSITION_DEGRADED
		}
		return validationmatrix.CellResult{Disposition: disposition, Reason: journeyFailureReason(result), Evidence: evidence, Report: report}
	}
	return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_PASS, Reason: "Android journeys journey completed", Evidence: evidence, Report: report}
}

func journeyFailureReason(result deliveryramp.JourneyResult) string {
	if strings.TrimSpace(result.DegradedReason) != "" {
		return result.DegradedReason
	}
	for _, step := range result.Steps {
		if strings.TrimSpace(step.Error) != "" {
			return step.Error
		}
	}
	if result.Disposition == deliveryramp.DispositionUnavailable {
		for _, step := range result.Steps {
			if strings.TrimSpace(step.DegradedReason) != "" {
				return step.DegradedReason
			}
		}
	}
	return string(result.Disposition)
}

func androidArtifact(configuredPath, digest string) (deliveryramp.Artifact, error) {
	path := strings.TrimSpace(configuredPath)
	if path == "" {
		path = strings.TrimSpace(os.Getenv("ANDROID_ARTIFACT_PATH"))
	}
	if path == "" {
		return deliveryramp.Artifact{}, fmt.Errorf("ANDROID_ARTIFACT_PATH is required for an Android matrix run")
	}
	apk, err := os.ReadFile(path)
	if err != nil {
		return deliveryramp.Artifact{}, fmt.Errorf("read Android artifact: %w", err)
	}
	aabPath := strings.Replace(path, string(filepath.Separator)+"apk"+string(filepath.Separator), string(filepath.Separator)+"bundle"+string(filepath.Separator), 1)
	if filepath.Base(path) == "app.apk" {
		// Retained immutable artifacts use stable app.apk/app.aab names in one
		// digest directory. Build workspace paths keep app-debug.apk and the
		// historical apk/../bundle/ layout handled below.
		aabPath = filepath.Join(filepath.Dir(path), "app.aab")
	} else {
		aabPath = strings.Replace(aabPath, "app-debug.apk", "app-debug.aab", 1)
	}
	aab, err := os.ReadFile(aabPath)
	if err != nil {
		return deliveryramp.Artifact{}, fmt.Errorf("read Android companion AAB: %w", err)
	}
	hash := sha256.New()
	_, _ = hash.Write(apk)
	_, _ = hash.Write([]byte("\x00"))
	_, _ = hash.Write(aab)
	actual := "sha256:" + hex.EncodeToString(hash.Sum(nil))
	if strings.TrimSpace(digest) != actual {
		return deliveryramp.Artifact{}, fmt.Errorf("Android artifact digest mismatch: matrix=%s file=%s", digest, actual)
	}
	return deliveryramp.Artifact{ImmutableRef: actual, LocalPath: path, Kind: "android-apk", Checksum: actual, Metadata: map[string]string{
		"apk_path": path, "package_name": firstNonEmpty(os.Getenv("ANDROID_ARTIFACT_PACKAGE"), defaultAndroidPackageName), "scenario_name": firstNonEmpty(os.Getenv("ANDROID_ARTIFACT_SCENARIO"), "hello-mobile"),
	}}, nil
}

func validImmutableRefForDigest(ref, checksum string) bool {
	parts := strings.Split(strings.TrimSpace(ref), ":")
	if len(parts) != 2 || (parts[0] != "android-debug" && parts[0] != "android-release") || len(parts[1]) != 64 {
		return false
	}
	for _, char := range parts[1] {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f')) {
			return false
		}
	}
	return "sha256:"+parts[1] == strings.TrimSpace(checksum)
}

func resolveURL(ctx context.Context, scenario, configured string) (string, error) {
	if strings.TrimSpace(configured) != "" {
		return strings.TrimRight(configured, "/"), nil
	}
	url, err := discovery.ResolveScenarioURLDefault(ctx, scenario)
	if err != nil {
		return "", fmt.Errorf("resolve %s URL: %w", scenario, err)
	}
	return strings.TrimRight(url, "/"), nil
}

func targetFromDescriptor(descriptor *domainv1.ValidationTargetDescriptor) deliveryramp.Target {
	targetID := descriptor.GetTargetId()
	deviceKind := "physical"
	mode := "physical"
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(targetID)), "android:emulator:") || strings.HasPrefix(strings.ToLower(strings.TrimSpace(targetID)), "emulator-") {
		deviceKind = "emulator"
		mode = "emulator"
	}
	return deliveryramp.Target{ID: targetID, Label: descriptor.GetDisplayName(), Platform: "android", Available: descriptor.GetAvailable(), DeviceKind: deviceKind, Mode: mode, Capabilities: capabilityNames(descriptor.GetCapabilities())}
}

func journeyEvidence(result deliveryramp.JourneyResult, request validationmatrix.CellRequest) []*domainv1.LayeredEvidence {
	evidence := make([]*domainv1.LayeredEvidence, 0)
	if result.ReviewRecording != nil && result.ReviewRecording.Redacted {
		reference := *result.ReviewRecording
		mediaType := reference.MediaType
		if mediaType == "" {
			mediaType = "video/mp4"
		}
		evidence = append(evidence, &domainv1.LayeredEvidence{Kind: domainv1.LayeredEvidence_KIND_DESKTOP_RUNTIME, EvidenceId: reference.ID, Uri: firstNonEmpty(reference.URI, "device-control://evidence/"+reference.ID), Sha256: reference.Checksum, MediaType: &mediaType, Redacted: reference.Redacted})
	}
	for _, step := range result.Steps {
		for _, reference := range step.Evidence {
			if !reference.Redacted {
				continue
			}
			mediaType := reference.MediaType
			if mediaType == "" {
				mediaType = "application/octet-stream"
			}
			kind := domainv1.LayeredEvidence_KIND_TARGET
			if strings.Contains(strings.ToLower(reference.Kind), "bas-runtime") {
				kind = domainv1.LayeredEvidence_KIND_DESKTOP_RUNTIME
			} else if strings.HasPrefix(strings.ToLower(mediaType), "video/") || strings.Contains(strings.ToLower(reference.Kind), "video") || strings.Contains(strings.ToLower(reference.Kind), "record") {
				kind = domainv1.LayeredEvidence_KIND_DESKTOP_RUNTIME
			}
			evidence = append(evidence, &domainv1.LayeredEvidence{Kind: kind, EvidenceId: reference.ID, Uri: firstNonEmpty(reference.URI, "device-control://evidence/"+reference.ID), Sha256: reference.Checksum, MediaType: &mediaType, Redacted: reference.Redacted})
		}
	}
	for _, sample := range []*deliveryramp.ClockOffsetSample{result.ClockOffsetStart, result.ClockOffsetEnd} {
		if sample == nil || sample.Evidence.ID == "" || !sample.Evidence.Redacted {
			continue
		}
		mediaType := sample.Evidence.MediaType
		if mediaType == "" {
			mediaType = "text/plain"
		}
		evidence = append(evidence, &domainv1.LayeredEvidence{Kind: domainv1.LayeredEvidence_KIND_MACHINE_ASSERTION, EvidenceId: sample.Evidence.ID, Uri: firstNonEmpty(sample.Evidence.URI, "device-control://evidence/"+sample.Evidence.ID), Sha256: sample.Evidence.Checksum, MediaType: &mediaType, Redacted: sample.Evidence.Redacted})
	}
	if targetProof := verifiedScenarioPackageEvidence(result, request); targetProof != nil {
		evidence = append(evidence, targetProof)
	}
	value := fmt.Sprintf("target=%s journey=%s run=%s", request.Target.GetTargetId(), request.Journey.JourneyID, request.RunID)
	mediaType := "text/plain"
	evidence = append(evidence, &domainv1.LayeredEvidence{Kind: domainv1.LayeredEvidence_KIND_MACHINE_ASSERTION, EvidenceId: "assertion-" + request.Cell.GetCellId(), Uri: "validation://android/" + request.Cell.GetCellId(), Sha256: hashText(value), MediaType: &mediaType, Redacted: true})
	return evidence
}

func verifiedScenarioPackageEvidence(result deliveryramp.JourneyResult, request validationmatrix.CellRequest) *domainv1.LayeredEvidence {
	mode := strings.TrimSpace(request.Metadata["install_mode"])
	if mode == "" {
		switch request.Journey.JourneyID {
		case "android-scenario-app-temporary-test-v1":
			mode = "temporary"
		case "android-scenario-app-durable-v1":
			mode = "durable"
		case "android-scenario-app-update-v1":
			mode = "update"
		case "android-scenario-app-verify-v1":
			mode = "verify"
		}
	}
	wantStep, wantState := "", ""
	switch mode {
	case "temporary":
		wantStep, wantState = "temporary-cleanup/verify-absent", "scenario package is absent"
	case "durable":
		wantStep = "verify-installed"
	case "update":
		wantStep = "verify-updated"
	case "verify":
		wantStep = "verify-installed"
	default:
		if request.Journey.JourneyID == "android-generated-app-conformance-v1" {
			wantStep, wantState = "clean_uninstall/assert-absent", "absent"
		} else {
			return nil
		}
	}
	for _, step := range result.Steps {
		stateMatches := wantState == "" || step.ExpectedState == wantState
		if wantState == "absent" {
			stateMatches = strings.Contains(strings.ToLower(step.ExpectedState), "absent")
		}
		if step.ID != wantStep || step.Action != "package-state" || step.Disposition != deliveryramp.StepPassed || step.AssertionStatus != "observed" || !stateMatches {
			continue
		}
		cellID := request.Cell.GetCellId()
		value := fmt.Sprintf("target=%s package=%s state=%s step=%s", request.Target.GetTargetId(), request.Metadata["package_name"], step.ExpectedState, step.ID)
		mediaType := "text/plain"
		return &domainv1.LayeredEvidence{
			Kind:       domainv1.LayeredEvidence_KIND_TARGET,
			EvidenceId: "target-state-" + cellID,
			Uri:        "validation://android/" + cellID + "/target-state",
			Sha256:     hashText(value), MediaType: &mediaType, Redacted: true,
		}
	}
	return nil
}

func capabilityEnum(value string) (domainv1.ValidationTargetCapability, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case deliveryramp.CapabilityDeviceControl:
		return domainv1.ValidationTargetCapability_VALIDATION_TARGET_CAPABILITY_NATIVE_WINDOW, true
	case deliveryramp.CapabilityScreenRecording:
		return domainv1.ValidationTargetCapability_VALIDATION_TARGET_CAPABILITY_PROCESS_METRICS, true
	case deliveryramp.CapabilityAndroidWebView:
		return domainv1.ValidationTargetCapability_VALIDATION_TARGET_CAPABILITY_ELECTRON_CDP, true
	case "network-control":
		return domainv1.ValidationTargetCapability_VALIDATION_TARGET_CAPABILITY_NETWORK_CONTROL, true
	default:
		return 0, false
	}
}

func capabilityNames(values []domainv1.ValidationTargetCapability) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, value.String())
	}
	return result
}

func transportForTarget(target *domainv1.ValidationTargetDescriptor) string {
	if strings.Contains(strings.ToLower(target.GetTargetId()), "emulator") {
		return "usb"
	}
	return "usb"
}

func hashText(value string) string {
	hash := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(hash[:])
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

var (
	_ validationmatrix.CatalogResolver = Catalog{}
	_ validationmatrix.CellTransport   = Executor{}
)
