// Package androidmatrix owns the Android adapter for the provider-neutral
// durable validation matrix. Discovery remains in androidprobe and execution
// remains in androidjourney/device-control/BAS.
package androidmatrix

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/vrooli/api-core/discovery"
	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
	validationmatrix "github.com/vrooli/vrooli/packages/delivery-ramp-go/validationmatrix"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"

	"scenario-to-android/internal/androidjourney"
	"scenario-to-android/internal/androidprobe"
	"scenario-to-android/internal/conformance"
)

type Catalog struct {
	Probe androidprobe.Prober
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
	plan := conformance.AndroidPlan()
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
		Journeys: []validationmatrix.JourneySelection{{JourneyID: plan.ID, DisplayName: "Android generated-app conformance", SourcePath: "internal/conformance/plan.go", ExecutionMode: "platform", Required: true, Category: "android", Requirements: []string{"hello-mobile APK", "device-control lease", "redacted recording", "BAS WebView flow"}, Safety: validationmatrix.JourneySafety{Mutating: true, RequiresIsolation: true, RequiresConfirmation: true}}},
		Targets:  targets,
	}, nil
}

type Executor struct {
	HTTP *http.Client
}

func (e Executor) Execute(ctx context.Context, request validationmatrix.CellRequest) validationmatrix.CellResult {
	if request.Cell == nil || request.Target == nil || !request.Target.GetAvailable() {
		return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_UNAVAILABLE, Reason: "Android target is unavailable"}
	}
	artifact, err := androidArtifact(request.ArtifactDigest)
	if err != nil {
		return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_UNAVAILABLE, Reason: err.Error()}
	}
	deviceURL, err := resolveURL(ctx, "device-control", os.Getenv("DEVICE_CONTROL_URL"))
	if err != nil {
		return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_UNAVAILABLE, Reason: err.Error()}
	}
	deviceTransport := transportForTarget(request.Target)
	if observed, inventoryErr := (androidprobe.DeviceControlInventory{Resolve: func(context.Context) (string, error) { return deviceURL, nil }, Client: e.HTTP}).List(ctx); inventoryErr == nil {
		for _, item := range observed {
			if item.ID == request.Target.GetTargetId() && strings.TrimSpace(item.ADBTransport) != "" {
				deviceTransport = item.ADBTransport
				break
			}
		}
	}
	basURL, err := resolveURL(ctx, "browser-automation-studio", os.Getenv("BAS_URL"))
	if err != nil {
		return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_UNAVAILABLE, Reason: err.Error()}
	}
	client := e.HTTP
	if client == nil {
		client = http.DefaultClient
	}
	driver := androidjourney.Driver{
		Devices: &androidjourney.HTTPDeviceClient{BaseURL: deviceURL, Actor: "scenario-to-android", DeviceTransport: deviceTransport, Client: client},
		BAS:     androidjourney.HTTPBASClient{BaseURL: basURL, FlowRoot: repoRoot(), HTTP: client},
		Actor:   "scenario-to-android",
	}
	result, runErr := driver.Execute(ctx, deliveryramp.DriverRequest{RunID: request.RunID, Cell: deliveryramp.Cell{ID: request.Cell.GetCellId(), Target: targetFromDescriptor(request.Target), ProfileID: request.Cell.GetEnvironmentProfile().String(), Required: request.Cell.GetRequired()}, Artifact: artifact, Plan: conformance.AndroidPlan().JourneyPlan()})
	if runErr != nil {
		return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_FAILED, Reason: runErr.Error()}
	}
	evidence := journeyEvidence(result, request)
	if result.Disposition != deliveryramp.DispositionPass {
		return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_FAILED, Reason: string(result.Disposition), Evidence: evidence}
	}
	return validationmatrix.CellResult{Disposition: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_PASS, Reason: "Android conformance journey completed", Evidence: evidence}
}

func androidArtifact(digest string) (deliveryramp.Artifact, error) {
	path := strings.TrimSpace(os.Getenv("ANDROID_ARTIFACT_PATH"))
	if path == "" {
		return deliveryramp.Artifact{}, fmt.Errorf("ANDROID_ARTIFACT_PATH is required for an Android matrix run")
	}
	file, err := os.Open(path)
	if err != nil {
		return deliveryramp.Artifact{}, fmt.Errorf("open Android artifact: %w", err)
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return deliveryramp.Artifact{}, fmt.Errorf("hash Android artifact: %w", err)
	}
	actual := "sha256:" + hex.EncodeToString(hash.Sum(nil))
	if strings.TrimSpace(digest) != actual {
		return deliveryramp.Artifact{}, fmt.Errorf("Android artifact digest mismatch: matrix=%s file=%s", digest, actual)
	}
	return deliveryramp.Artifact{ImmutableRef: actual, LocalPath: path, Kind: "android-apk", Checksum: actual, Metadata: map[string]string{
		"apk_path": path, "package_name": firstNonEmpty(os.Getenv("ANDROID_ARTIFACT_PACKAGE"), "com.example.generated"), "scenario_name": firstNonEmpty(os.Getenv("ANDROID_ARTIFACT_SCENARIO"), "hello-mobile"),
	}}, nil
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
	return deliveryramp.Target{ID: descriptor.GetTargetId(), Label: descriptor.GetDisplayName(), Platform: "android", Available: descriptor.GetAvailable(), DeviceKind: "android", Capabilities: capabilityNames(descriptor.GetCapabilities())}
}

func journeyEvidence(result deliveryramp.JourneyResult, request validationmatrix.CellRequest) []*domainv1.LayeredEvidence {
	evidence := make([]*domainv1.LayeredEvidence, 0)
	for _, step := range result.Steps {
		for _, reference := range step.Evidence {
			mediaType := reference.MediaType
			if mediaType == "" {
				mediaType = "application/octet-stream"
			}
			kind := domainv1.LayeredEvidence_KIND_TARGET
			if strings.HasPrefix(strings.ToLower(mediaType), "video/") || strings.Contains(strings.ToLower(reference.Kind), "video") || strings.Contains(strings.ToLower(reference.Kind), "record") {
				kind = domainv1.LayeredEvidence_KIND_DESKTOP_RUNTIME
			}
			evidence = append(evidence, &domainv1.LayeredEvidence{Kind: kind, EvidenceId: reference.ID, Uri: firstNonEmpty(reference.URI, "device-control://evidence/"+reference.ID), Sha256: reference.Checksum, MediaType: &mediaType, Redacted: reference.Redacted})
		}
	}
	value := fmt.Sprintf("target=%s journey=%s run=%s", request.Target.GetTargetId(), request.Journey.JourneyID, request.RunID)
	mediaType := "text/plain"
	evidence = append(evidence, &domainv1.LayeredEvidence{Kind: domainv1.LayeredEvidence_KIND_MACHINE_ASSERTION, EvidenceId: "assertion-" + request.Cell.GetCellId(), Uri: "validation://android/" + request.Cell.GetCellId(), Sha256: hashText(value), MediaType: &mediaType, Redacted: true})
	return evidence
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

func repoRoot() string {
	if value := strings.TrimSpace(os.Getenv("VROOLI_ROOT")); value != "" {
		return value
	}
	return "."
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
