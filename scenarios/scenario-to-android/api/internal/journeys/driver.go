// Package journeys adapts scenario-to-android's device-control and BAS
// clients to the delivery-ramp Driver seam. Device verbs are deliberately
// interfaces: the production implementation is the device-control client,
// while tests can prove lease and evidence invariants without adb.
package journeys

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
)

type Lease struct {
	ID        string
	DeviceID  string
	Token     string
	ExpiresAt time.Time
}

type ActionResult struct {
	Observed string
	Evidence []deliveryramp.EvidenceReference
}

type RecordingArtifact struct {
	Reference  deliveryramp.EvidenceReference
	Path       string
	StartMs    int64
	EndMs      int64
	HasOffsets bool
}

// ChapterRecorder is optional so older/native-only test doubles can retain
// the run-level recording contract while production device-control clients
// provide one bounded recording for each conformance chapter.
type ChapterRecorder interface {
	StartChapterRecording(context.Context, Lease, string) error
	StopChapterRecording(context.Context, Lease, string) (RecordingArtifact, error)
}

// ReviewRecorder finalizes the producer-owned chapter recordings into one
// review artifact. It is optional so the journey remains testable with small
// device fakes, while the production device-control client owns the bytes.
type ReviewRecorder interface {
	FinalizeReviewRecording(context.Context, Lease, []deliveryramp.EvidenceReference) (ReviewRecording, error)
}

type ReviewRecording struct {
	Reference deliveryramp.EvidenceReference
	Path      string
}

type LogCapture interface {
	StartLogCapture(context.Context, Lease) error
	StopLogCapture(context.Context, Lease) (LogCaptureArtifact, error)
}

// ClockSampler is optional for compatibility with older test doubles. The
// production HTTP client implements it so journey evidence contains explicit
// start/end host-to-device clock calibration samples.
type ClockSampler interface {
	SampleClock(context.Context, Lease) (ClockSample, error)
}

type DeviceClient interface {
	Acquire(context.Context, string, string, time.Duration) (Lease, error)
	ValidateLease(context.Context, Lease) error
	Execute(context.Context, Lease, string, map[string]string) (ActionResult, error)
	StartRecording(context.Context, Lease) error
	StopRecording(context.Context, Lease) (RecordingArtifact, error)
	Release(context.Context, Lease) error
}

// Authenticator lets a journey use a device-control-owned credential profile
// for its unlock step. The credential itself never crosses this boundary.
type Authenticator interface {
	Unlock(context.Context, Lease, string) error
}

// WebViewAttacher is optional on DeviceClient so existing native-only test
// doubles remain small. The production device-control client implements it;
// the driver then supplies BAS with a lease-scoped forwarded CDP endpoint.
type WebViewAttacher interface {
	AttachWebView(context.Context, Lease, string) (WebViewAttachment, error)
}

type TransportWebViewAttacher interface {
	AttachWebViewWithTransport(context.Context, Lease, string, string) (WebViewAttachment, error)
}

type WebViewAttachment struct {
	CDPEndpoint string
	RendererID  string
	RendererURL string
}

// DeviceStateSnapshot contains only the state required to decide whether an
// Android app journey can restore the target after testing. Device Control
// remains the source of these observations and owns all mutations.
type DeviceStateSnapshot struct {
	ForegroundPackage string `json:"foreground_package"`
	ScreenState       string `json:"screen_state"`
	LockState         string `json:"lock_state"`
}

type DeviceStateReader interface {
	ReadState(context.Context, string) (DeviceStateSnapshot, error)
}

type InstalledPackageState struct {
	Installed   bool
	Version     string
	VersionCode int64
}

type PackageStateReader interface {
	ReadPackageState(context.Context, Lease, string) (InstalledPackageState, error)
}

type BASRequest struct {
	TargetID         string
	Scenario         string
	Artifact         deliveryramp.Artifact
	StepID           string
	RunID            string
	Arguments        map[string]string
	CDPEndpoint      string
	RendererID       string
	RendererURL      string
	FlowPath         string
	IsolationLeaseID string
}

type BASResult struct {
	Completed   bool
	ExecutionID string
	Evidence    []deliveryramp.EvidenceReference
}

type BASClient interface {
	ValidateFlow(context.Context, BASRequest) error
	Execute(context.Context, BASRequest) (BASResult, error)
}

type Driver struct {
	Devices         DeviceClient
	BAS             BASClient
	WebView         WebViewAttacher
	Actor           string
	LeaseTTL        time.Duration
	DeviceTransport string
}

var _ deliveryramp.Driver = Driver{}

func (d Driver) Execute(ctx context.Context, request deliveryramp.DriverRequest) (result deliveryramp.JourneyResult, err error) {
	runStarted := time.Now()
	result = deliveryramp.JourneyResult{
		SchemaVersion: deliveryramp.JourneySchemaVersion, EvidenceVersion: deliveryramp.JourneyEvidenceVersion,
		SmokeTestID: request.RunID, PlanID: request.Plan.ID, Profile: request.Plan.Profile,
		Capability: request.Plan.Capability, TargetID: request.Cell.Target.ID, CellID: request.Cell.ID,
		ScenarioName: artifactScenario(request.Artifact, request.Cell.Target.Label), Platform: request.Cell.Target.Platform,
		Disposition: deliveryramp.DispositionNotRun, Steps: make([]deliveryramp.JourneyStep, 0, len(request.Plan.Steps)),
		CreatedAt: runStarted.UTC(),
	}
	if d.Devices == nil {
		return result, fmt.Errorf("device-control client is unavailable")
	}
	if !request.Cell.Target.Available {
		return result, fmt.Errorf("Android target %q is unavailable: %s", request.Cell.Target.ID, request.Cell.Target.Reason)
	}
	if strings.TrimSpace(request.Artifact.ImmutableRef) == "" {
		return result, fmt.Errorf("Android artifact immutable identity is required")
	}
	installMode := scenarioAppInstallMode(request.Plan.ID)
	focusedOffline := request.Plan.ID == OfflineTransitionPlanID
	if focusedOffline {
		if strings.TrimSpace(request.Cell.Target.ID) == "" || request.Cell.Target.Platform != Platform || d.DeviceTransport != "usb" {
			return result, fmt.Errorf("offline-transition journey requires an exact Android target on USB")
		}
		if artifactScenario(request.Artifact, "") != Fixture {
			return result, fmt.Errorf("offline-transition journey requires the hello-mobile test fixture; its BAS flow writes the test sentinel")
		}
		attacher := d.WebView
		if attacher == nil {
			attacher, _ = d.Devices.(WebViewAttacher)
		}
		if _, ok := attacher.(TransportWebViewAttacher); !ok {
			return result, fmt.Errorf("offline-transition journey requires USB-routed WebView attachment")
		}
		for _, spec := range request.Plan.Steps {
			if missing := missingCapability(spec.Arguments["required_capabilities"], request.Cell.Target.Capabilities); missing != "" {
				return result, fmt.Errorf("offline-transition journey requires target capability %q before mutation", missing)
			}
		}
		defer func() {
			result.CompletedAt = time.Now().UTC()
			if err != nil {
				result.Disposition = deliveryramp.DispositionFailed
				result.DegradedReason = err.Error()
			}
		}()
	}
	if installMode != "" {
		if d.BAS == nil {
			return failBASPreflight(&result, "BAS client is unavailable; refusing scenario app installation without a runnable workflow")
		}
		for _, spec := range request.Plan.Steps {
			if !strings.EqualFold(spec.Action, "bas-flow") {
				continue
			}
			flowPath := strings.TrimSpace(spec.Arguments["flow_path"])
			if flowPath == "" {
				flowPath = DefaultScenarioBASFlowPath
			}
			basRequest := BASRequest{
				TargetID: request.Cell.Target.ID, Scenario: artifactScenario(request.Artifact, request.Cell.Target.Label),
				Artifact: request.Artifact, StepID: spec.ID, Arguments: spec.Arguments, FlowPath: flowPath,
			}
			flowStep := deliveryramp.JourneyStep{
				ID: "install-preflight/bas-flow/" + spec.ID, Name: "install-preflight/bas-flow/" + spec.ID,
				Purpose: "verify the scenario BAS workflow is registered and readable before artifact installation",
				Action:  "bas-flow", Disposition: deliveryramp.StepPassed, StartedAt: time.Now().UTC(),
				ExpectedState: "registered BAS flow " + strings.TrimSpace(spec.Arguments["flow_reference"]),
				AssertionID:   "install-preflight/bas-flow/" + spec.ID + "/available", AssertionStatus: "observed",
			}
			if err := d.BAS.ValidateFlow(ctx, basRequest); err != nil {
				flowStep.Disposition = deliveryramp.StepFailed
				flowStep.Error = "safe Android install refused before mutation: BAS flow preflight failed: " + err.Error()
				flowStep.CompletedAt = time.Now().UTC()
				result.Steps = append(result.Steps, flowStep)
				result.Disposition = deliveryramp.DispositionFailed
				result.DegradedReason = flowStep.Error
				return result, nil
			}
			flowStep.CompletedAt = time.Now().UTC()
			flowStep.ObservedState = "flow file is readable and registered in the scenario BAS registry"
			result.Steps = append(result.Steps, flowStep)
		}
	}
	ttl := d.LeaseTTL
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	lease, err := d.Devices.Acquire(ctx, request.Cell.Target.ID, d.Actor, ttl)
	if err != nil {
		return result, fmt.Errorf("acquire device-control lease: %w", err)
	}
	defer func() { _ = d.Devices.Release(context.Background(), lease) }()
	packageName := ""
	var preUpdatePackageState InstalledPackageState
	var candidateUpdateVersionCode int64
	var initialDeviceState DeviceStateSnapshot
	var restoreDeviceState bool
	if installMode != "" {
		var validationErr error
		packageName, validationErr = validateScenarioInstallArtifact(request.Artifact)
		if validationErr != nil {
			return result, validationErr
		}
		preflightStep := deliveryramp.JourneyStep{ID: "install-preflight/package-state", Name: "install-preflight/package-state", Purpose: "verify the package precondition before scenario artifact installation", Action: "package-state", Disposition: deliveryramp.StepPassed, StartedAt: time.Now().UTC()}
		var preflightErr error
		if installMode == "update" {
			packageReader, ok := d.Devices.(PackageStateReader)
			if !ok {
				return failInstallPackagePreflight(&result, "Device Control package version readback is unavailable; refusing update")
			}
			installed, readErr := packageReader.ReadPackageState(ctx, lease, packageName)
			candidateCode, codeErr := strconv.ParseInt(strings.TrimSpace(request.Artifact.Metadata["version_code"]), 10, 64)
			if readErr != nil {
				return failInstallPackagePreflight(&result, "Device Control could not verify the installed package version: "+readErr.Error())
			}
			if codeErr != nil || candidateCode <= 0 {
				return failInstallPackagePreflight(&result, "update artifact has no valid positive version code")
			}
			if !installed.Installed {
				return failInstallPackagePreflight(&result, "scenario update refused because the package is not installed")
			}
			if installed.VersionCode <= 0 {
				return failInstallPackagePreflight(&result, "scenario update refused because the installed version code is unknown")
			}
			if candidateCode <= installed.VersionCode {
				return failInstallPackagePreflight(&result, fmt.Sprintf("scenario update version code %d must be greater than installed version code %d", candidateCode, installed.VersionCode))
			}
			preUpdatePackageState = installed
			candidateUpdateVersionCode = candidateCode
			preflightStep.Purpose = "verify the installed package has a known lower version code before updating"
			preflightStep.ExpectedState = fmt.Sprintf("installed versionCode %d is less than candidate versionCode %d", installed.VersionCode, candidateCode)
			preflightStep.ObservedState = fmt.Sprintf("installed=%t; versionName=%s; versionCode=%d", installed.Installed, installed.Version, installed.VersionCode)
			preflightStep.AssertionID = "install-preflight/version-upgrade"
			preflightStep.AssertionSurfaceObserved = true
		} else if installMode == "verify" {
			packageReader, ok := d.Devices.(PackageStateReader)
			if !ok {
				return failInstallPackagePreflight(&result, "Device Control package-state readback is unavailable; refusing installed-app verification")
			}
			installed, readErr := packageReader.ReadPackageState(ctx, lease, packageName)
			if readErr != nil {
				return failInstallPackagePreflight(&result, "Device Control could not verify the installed scenario package: "+readErr.Error())
			}
			if !installed.Installed {
				return failInstallPackagePreflight(&result, "scenario app verification refused because the package is not installed")
			}
			wantVersion := strings.TrimSpace(request.Artifact.Metadata["version_name"])
			if strings.TrimSpace(installed.Version) == "" || wantVersion == "" || installed.Version != wantVersion {
				return failInstallPackagePreflight(&result, fmt.Sprintf("scenario app verification refused because installed version name %q does not match expected %q", installed.Version, wantVersion))
			}
			candidateCode, codeErr := strconv.ParseInt(strings.TrimSpace(request.Artifact.Metadata["version_code"]), 10, 64)
			if codeErr != nil || candidateCode <= 0 {
				return failInstallPackagePreflight(&result, "verification artifact has no valid positive version code")
			}
			if installed.VersionCode > 0 && installed.VersionCode != candidateCode {
				return failInstallPackagePreflight(&result, fmt.Sprintf("scenario app verification refused because installed version code %d does not match expected %d", installed.VersionCode, candidateCode))
			}
			if focusedOffline && installed.VersionCode <= 0 {
				return failInstallPackagePreflight(&result, "offline-transition journey requires a known matching installed version code")
			}
			preflightStep.Purpose = "verify the existing package before BAS without reinstalling or changing saved data"
			preflightStep.ExpectedState = fmt.Sprintf("package %s versionName %s; versionCode %d when reported", packageName, wantVersion, candidateCode)
			if installed.VersionCode > 0 {
				preflightStep.ObservedState = fmt.Sprintf("installed=true; versionName=%s; versionCode=%d", installed.Version, installed.VersionCode)
			} else {
				preflightStep.ObservedState = fmt.Sprintf("installed=true; versionName=%s; versionCode=unavailable", installed.Version)
			}
			preflightStep.AssertionID = "install-preflight/package-identity"
			preflightStep.AssertionSurfaceObserved = true
		} else {
			preflight, preflightErr := d.Devices.Execute(ctx, lease, "package-state", map[string]string{
				"step_id": "install-preflight/package-state", "package": packageName, "expected": "absent",
			})
			if preflightErr != nil {
				preflightStep.Disposition = deliveryramp.StepFailed
				preflightStep.Error = fmt.Sprintf("safe Android install refused before mutation: %s", preflightErr)
				preflightStep.CompletedAt = time.Now().UTC()
				result.Steps = append(result.Steps, preflightStep)
				result.Disposition = deliveryramp.DispositionFailed
				result.DegradedReason = preflightStep.Error
				return result, nil
			}
			preflightStep.Purpose = "refuse installation when the target package already exists"
			preflightStep.ExpectedState = "scenario package is absent"
			preflightStep.AssertionID = "install-preflight/package-absent"
			preflightStep.Evidence = append(preflightStep.Evidence, preflight.Evidence...)
		}
		preflightStep.CompletedAt = time.Now().UTC()
		stateReader, ok := d.Devices.(DeviceStateReader)
		if !ok {
			return failAppInstallPreflight(&result, "Device Control state readback is unavailable; refusing scenario app operation")
		}
		initialDeviceState, preflightErr = stateReader.ReadState(ctx, lease.DeviceID)
		if preflightErr != nil {
			return failAppInstallPreflight(&result, "Device Control could not verify the pre-test screen and lock state: "+preflightErr.Error())
		}
		if err := validateRestorableAndroidState(initialDeviceState, request.Artifact.Metadata["auth_profile_id"]); err != nil {
			return failAppInstallPreflight(&result, "safe Android scenario-app operation refused before mutation: "+err.Error())
		}
		stateStep := deliveryramp.JourneyStep{
			ID: "install-preflight/device-state", Name: "install-preflight/device-state",
			Purpose: "record the verified pre-test Android display and lock state",
			Action:  "device-state", Disposition: deliveryramp.StepPassed,
			StartedAt: time.Now().UTC(), CompletedAt: time.Now().UTC(),
			ExpectedState: fmt.Sprintf("screen=%s; lock=%s; foreground=%s", initialDeviceState.ScreenState, initialDeviceState.LockState, initialDeviceState.ForegroundPackage),
			AssertionID:   "install-preflight/device-state-known", AssertionStatus: "observed",
		}
		result.Steps = append(result.Steps, stateStep)
		preflightStep.AssertionStatus = "observed"
		result.Steps = append(result.Steps, preflightStep)

		// Register restoration before temporary cleanup so the package is removed
		// first, then the original display/lock state is restored and verified.
		defer func() {
			if !restoreDeviceState {
				return
			}
			restoredAt := time.Now().UTC()
			restoreStep := deliveryramp.JourneyStep{
				ID: "device-cleanup/restore-screen-state", Name: "device-cleanup/restore-screen-state",
				Purpose: "restore and verify the pre-test Android display and lock state",
				Action:  "screen-restore", StartedAt: restoredAt,
				ExpectedState: fmt.Sprintf("screen=%s; lock=%s", initialDeviceState.ScreenState, initialDeviceState.LockState),
				AssertionID:   "device-cleanup/screen-state-restored",
			}
			restoreErr := restoreAndroidScreenState(context.Background(), d.Devices, lease, initialDeviceState, request.Artifact.Metadata["auth_profile_id"])
			restoreStep.CompletedAt = time.Now().UTC()
			if restoreErr != nil {
				restoreStep.Disposition = deliveryramp.StepFailed
				restoreStep.AssertionStatus = "failed"
				restoreStep.Error = "Android screen-state restoration failed: " + restoreErr.Error()
				result.Disposition = deliveryramp.DispositionFailed
				result.DegradedReason = restoreStep.Error
				if err == nil {
					err = errors.New(restoreStep.Error)
				} else {
					err = errors.Join(err, errors.New(restoreStep.Error))
				}
			} else {
				restoreStep.Disposition = deliveryramp.StepPassed
				restoreStep.AssertionStatus = "observed"
			}
			result.Steps = append(result.Steps, restoreStep)
		}()

		if installMode == "verify" {
			// Verification must exercise the installed package as-is. It never
			// replaces or removes the app; only the prior screen/lock state is
			// restored after BAS completes.
			restoreDeviceState = true
			if focusedOffline {
				defer func() {
					readbackCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
					defer cancel()
					state, readErr := d.Devices.(PackageStateReader).ReadPackageState(readbackCtx, lease, packageName)
					step := deliveryramp.JourneyStep{ID: "device-cleanup/package-identity", Name: "device-cleanup/package-identity", ChapterID: "offline_transition", Action: "package-state", Purpose: "verify the existing fixture package remains at the expected version without reinstalling or uninstalling", StartedAt: time.Now().UTC(), ExpectedState: fmt.Sprintf("package %s versionName %s versionCode %s", packageName, request.Artifact.Metadata["version_name"], request.Artifact.Metadata["version_code"]), AssertionID: "device-cleanup/package-identity", Disposition: deliveryramp.StepPassed, AssertionStatus: "observed"}
					step.ObservedState = fmt.Sprintf("installed=%t; versionName=%s; versionCode=%d", state.Installed, state.Version, state.VersionCode)
					if readErr == nil && (!state.Installed || state.Version != request.Artifact.Metadata["version_name"] || strconv.FormatInt(state.VersionCode, 10) != request.Artifact.Metadata["version_code"]) {
						readErr = fmt.Errorf("installed package identity changed or is unknown: %s", step.ObservedState)
					}
					if readErr != nil {
						step.Disposition, step.AssertionStatus = deliveryramp.StepFailed, "failed"
						step.Error = "offline-transition terminal package readback failed: " + readErr.Error()
						result.Disposition = deliveryramp.DispositionFailed
						err = errors.Join(err, errors.New(step.Error))
					}
					step.CompletedAt = time.Now().UTC()
					result.Steps = append(result.Steps, step)
				}()
			}
		} else {
			if installMode == "update" {
				// Capture the app-owned sentinel on the installed version before
				// replacement. The registered flow is observer-only and has already
				// passed the same registry/path preflight as the post-update flow.
				restoreDeviceState = true
				if initialDeviceState.ScreenState == "off" {
					if _, wakeErr := d.Devices.Execute(ctx, lease, "screen", map[string]string{"step_id": "update-preflight/screen-wake", "target": "wake"}); wakeErr != nil {
						return failInstallPackagePreflight(&result, "could not wake display for pre-update saved-state assertion: "+wakeErr.Error())
					}
				}
				if initialDeviceState.LockState == "locked" {
					profileID := strings.TrimSpace(request.Artifact.Metadata["auth_profile_id"])
					authenticator, ok := d.Devices.(Authenticator)
					if profileID == "" || !ok {
						return failInstallPackagePreflight(&result, "cannot unlock installed app for pre-update saved-state assertion")
					}
					if unlockErr := authenticator.Unlock(ctx, lease, profileID); unlockErr != nil {
						return failInstallPackagePreflight(&result, "could not unlock installed app for pre-update saved-state assertion: "+unlockErr.Error())
					}
				}
				if _, launchErr := d.Devices.Execute(ctx, lease, "launch", map[string]string{"step_id": "update-preflight/launch", "package": packageName}); launchErr != nil {
					return failInstallPackagePreflight(&result, "could not launch installed app for pre-update saved-state assertion: "+launchErr.Error())
				}
				basRequest := BASRequest{TargetID: request.Cell.Target.ID, Scenario: artifactScenario(request.Artifact, request.Cell.Target.Label), Artifact: request.Artifact, StepID: "update-preflight/saved-state", RunID: request.RunID, Arguments: map[string]string{"flow_reference": UpdatePreservationFlowID}, FlowPath: UpdatePreservationFlowPath, IsolationLeaseID: lease.ID}
				if attacher, ok := d.Devices.(WebViewAttacher); ok {
					endpoint, attachErr := attacher.AttachWebView(ctx, lease, packageName)
					if attachErr != nil {
						return failInstallPackagePreflight(&result, "could not attach installed app for pre-update saved-state assertion: "+attachErr.Error())
					}
					basRequest.CDPEndpoint, basRequest.RendererID, basRequest.RendererURL = endpoint.CDPEndpoint, endpoint.RendererID, endpoint.RendererURL
				}
				assertionStarted := time.Now().UTC()
				assertion := deliveryramp.JourneyStep{ID: basRequest.StepID, Name: basRequest.StepID, Purpose: "verify existing app-owned state before replacing the installed artifact", Action: "bas-flow", StartedAt: assertionStarted, ExpectedState: "registered saved-state sentinel exists on the installed version", AssertionID: "update-preflight/saved-state-present"}
				basResult, basErr := d.BAS.Execute(ctx, basRequest)
				assertion.CompletedAt = time.Now().UTC()
				assertion.Evidence = append(assertion.Evidence, basResult.Evidence...)
				if basErr != nil || !basResult.Completed {
					assertion.Disposition, assertion.AssertionStatus = deliveryramp.StepFailed, "failed"
					if basErr != nil {
						assertion.Error = "pre-update saved-state assertion failed: " + basErr.Error()
					} else {
						assertion.Error = "pre-update saved-state assertion did not complete"
					}
					result.Steps = append(result.Steps, assertion)
					result.Disposition, result.DegradedReason = deliveryramp.DispositionFailed, assertion.Error
					return result, nil
				}
				assertion.Disposition, assertion.AssertionStatus = deliveryramp.StepPassed, "observed"
				assertion.ObservedState = "registered saved-state assertion passed on the installed version"
				result.Steps = append(result.Steps, assertion)
			}
			installAttempted := false
			if installMode == "temporary" {
				defer func() {
					if !installAttempted {
						return
					}
					cleanupStarted := time.Now().UTC()
					_, uninstallErr := d.Devices.Execute(context.Background(), lease, "uninstall", map[string]string{"step_id": "temporary-cleanup/uninstall", "package": packageName})
					_, verifyErr := d.Devices.Execute(context.Background(), lease, "package-state", map[string]string{"step_id": "temporary-cleanup/verify-absent", "package": packageName, "expected": "absent"})
					cleanupStep := deliveryramp.JourneyStep{ID: "temporary-cleanup/verify-absent", Name: "temporary-cleanup/verify-absent", Purpose: "remove the temporary package and verify its absence", Action: "package-state", StartedAt: cleanupStarted, CompletedAt: time.Now().UTC(), ExpectedState: "scenario package is absent", AssertionID: "temporary-cleanup/package-absent", AssertionStatus: "observed"}
					if verifyErr != nil {
						cleanupStep.Disposition = deliveryramp.StepFailed
						cleanupStep.Error = "temporary app cleanup could not verify package absence: " + verifyErr.Error()
						result.Disposition = deliveryramp.DispositionFailed
						result.DegradedReason = cleanupStep.Error
						if err == nil {
							err = errors.New(cleanupStep.Error)
						} else {
							err = errors.Join(err, errors.New(cleanupStep.Error))
						}
					} else {
						cleanupStep.Disposition = deliveryramp.StepPassed
						if uninstallErr != nil {
							cleanupStep.DegradedReason = "uninstall reported an error, but package-state verified absence: " + uninstallErr.Error()
						}
					}
					if result.Steps == nil {
						result.Steps = make([]deliveryramp.JourneyStep, 0, len(request.Plan.Steps)+3)
					}
					result.Steps = append(result.Steps, cleanupStep)
				}()
			}
			installAttempted = installMode == "temporary"
			installStarted := time.Now().UTC()
			installResult, installErr := d.Devices.Execute(ctx, lease, "install-artifact", map[string]string{
				"step_id": "artifact-install/install", "artifact_ref": request.Artifact.ImmutableRef,
				"package": packageName, "version_name": request.Artifact.Metadata["version_name"],
				"timeout_ms": strconv.FormatInt(actionTimeoutMS("install"), 10),
			})
			installStep := deliveryramp.JourneyStep{ID: "artifact-install/install", Name: "artifact-install/install", Purpose: "install the exact immutable Android artifact through Device Control", Action: "install-artifact", StartedAt: installStarted, CompletedAt: time.Now().UTC(), ExpectedState: request.Artifact.ImmutableRef}
			if installErr != nil {
				installStep.Disposition = deliveryramp.StepFailed
				installStep.Error = installErr.Error()
				result.Steps = append(result.Steps, installStep)
				result.Disposition = deliveryramp.DispositionFailed
				result.DegradedReason = installStep.Error
				if installMode == "update" {
					appendUpdateFailurePackageReadback(&result, d.Devices, lease, packageName, preUpdatePackageState, candidateUpdateVersionCode)
				}
				return result, nil
			}
			installStep.Disposition = deliveryramp.StepPassed
			installStep.AssertionID = "artifact-install/verified"
			installStep.AssertionStatus = "observed"
			installStep.Evidence = append(installStep.Evidence, installResult.Evidence...)
			result.Steps = append(result.Steps, installStep)
			restoreDeviceState = true
		}
	}
	clockSampler, hasClockSampler := d.Devices.(ClockSampler)
	if hasClockSampler {
		sample, sampleErr := clockSampler.SampleClock(ctx, lease)
		if sampleErr != nil {
			return result, fmt.Errorf("sample device clock at journey start: %w", sampleErr)
		}
		result.ClockOffsetStart = clockOffsetSample(sample)
		// Register this defer before log capture teardown so the logcat stop
		// runs first and the end calibration is the final device observation.
		defer func() {
			end, endErr := clockSampler.SampleClock(context.Background(), lease)
			if endErr != nil {
				if err == nil {
					err = fmt.Errorf("sample device clock at journey end: %w", endErr)
				} else {
					err = errors.Join(err, fmt.Errorf("sample device clock at journey end: %w", endErr))
				}
				result.Disposition = deliveryramp.DispositionFailed
				return
			}
			result.ClockOffsetEnd = clockOffsetSample(end)
		}()
	}
	recordDeviceLogs := installMode != "verify"
	logCapture, hasLogCapture := d.Devices.(LogCapture)
	if hasLogCapture && recordDeviceLogs {
		if err := logCapture.StartLogCapture(ctx, lease); err != nil {
			return result, fmt.Errorf("start device logcat capture: %w", err)
		}
		defer func() {
			capture, stopErr := logCapture.StopLogCapture(context.Background(), lease)
			if stopErr == nil {
				if capture.Reference.ID != "" && len(result.Steps) > 0 {
					// Logcat is run-scoped evidence. Attach the producer-owned
					// artifact to the first step so the durable matrix retains a
					// verifiable reference without inventing a separate timeline
					// file.
					result.Steps[0].Evidence = append(result.Steps[0].Evidence, capture.Reference)
				}
				result.Events = append(result.Events, parseLogcatEvents(capture.Lines)...)
				finalizeJourneyAssertions(&result)
			}
		}()
	}
	chapterRecorder, perChapter := d.Devices.(ChapterRecorder)
	if installMode == "verify" {
		// The installed-app check is read-only with respect to package data and
		// avoids Device Control's external screen-recording lifecycle until its
		// uncertain recording handles can be reconciled by their owner.
		perChapter = false
	}
	reviewRecorder, hasReviewRecorder := d.Devices.(ReviewRecorder)
	chapterReferences := make([]deliveryramp.EvidenceReference, 0)
	videoUnavailable := make(map[string]string)
	activeChapter := ""
	chapterStartStep := -1
	chapterActive := false
	stopChapter := func(stopCtx context.Context) error {
		if !chapterActive {
			return nil
		}
		recording, stopErr := chapterRecorder.StopChapterRecording(stopCtx, lease, activeChapter)
		chapterActive = false
		if stopErr != nil {
			videoUnavailable[activeChapter] = fmt.Sprintf("stop recording for chapter %q: %s", activeChapter, stopErr)
			for index := chapterStartStep; index < len(result.Steps); index++ {
				if result.Steps[index].ChapterID == activeChapter || index >= chapterStartStep {
					result.Steps[index].VideoDisposition = deliveryramp.StepUnavailable
					result.Steps[index].VideoError = videoUnavailable[activeChapter]
				}
			}
			return nil
		}
		if !recording.HasOffsets || recording.StartMs < 0 || recording.EndMs < recording.StartMs {
			videoUnavailable[activeChapter] = fmt.Sprintf("chapter %q recording is missing bounded start/end offsets", activeChapter)
			for index := chapterStartStep; index < len(result.Steps); index++ {
				result.Steps[index].VideoDisposition = deliveryramp.StepUnavailable
				result.Steps[index].VideoError = videoUnavailable[activeChapter]
			}
			return nil
		}
		if recording.Reference.ID == "" || recording.Reference.Checksum == "" || !recording.Reference.Redacted {
			videoUnavailable[activeChapter] = fmt.Sprintf("chapter %q recording reference is incomplete or not redacted", activeChapter)
			for index := chapterStartStep; index < len(result.Steps); index++ {
				result.Steps[index].VideoDisposition = deliveryramp.StepUnavailable
				result.Steps[index].VideoError = videoUnavailable[activeChapter]
			}
			return nil
		}
		chapterReferences = append(chapterReferences, recording.Reference)
		if chapterStartStep >= 0 && chapterStartStep < len(result.Steps) {
			result.Steps[chapterStartStep].ChapterID = activeChapter
			result.Steps[chapterStartStep].Evidence = append(result.Steps[chapterStartStep].Evidence, recording.Reference)
			start, end := recording.StartMs, recording.EndMs
			result.Steps[chapterStartStep].VideoStartOffsetMs = &start
			result.Steps[len(result.Steps)-1].VideoEndOffsetMs = &end
			for index := chapterStartStep; index < len(result.Steps); index++ {
				result.Steps[index].ChapterID = activeChapter
			}
		}
		return nil
	}
	if !perChapter && installMode != "verify" {
		if err := d.Devices.StartRecording(ctx, lease); err != nil {
			return result, fmt.Errorf("start device recording before launch: %w", err)
		}
	}
	networkRestoreNeeded := false
	defer func() {
		if chapterActive {
			_ = stopChapter(context.Background())
		}
	}()
	defer func() {
		if !networkRestoreNeeded {
			return
		}
		step := deliveryramp.JourneyStep{
			ID: "device-cleanup/network-online", Name: "device-cleanup/network-online",
			Purpose: "restore connectivity after an interrupted offline-transition chapter",
			Action:  "network", Disposition: deliveryramp.StepPassed,
			ExpectedState: "network online", StartedAt: time.Now().UTC(),
		}
		_, restoreErr := d.Devices.Execute(context.Background(), lease, "network", map[string]string{
			"step_id": step.ID, "target": "online", "value": "online",
		})
		step.CompletedAt = time.Now().UTC()
		if restoreErr != nil {
			step.Disposition = deliveryramp.StepFailed
			step.Error = fmt.Sprintf("device-control network recovery failed: %s", restoreErr)
			result.Disposition = deliveryramp.DispositionFailed
			result.DegradedReason = step.Error
			if err == nil {
				err = fmt.Errorf("restore network after offline-transition failure: %w", restoreErr)
			} else {
				err = errors.Join(err, fmt.Errorf("restore network after offline-transition failure: %w", restoreErr))
			}
		}
		result.Steps = append(result.Steps, step)
	}()
	unsupportedChapters := make(map[string]string)
	for index, spec := range request.Plan.Steps {
		if err := d.Devices.ValidateLease(ctx, lease); err != nil {
			return result, fmt.Errorf("device-control lease lost before step %q: %w", spec.ID, err)
		}
		step := deliveryramp.JourneyStep{ID: spec.ID, Name: spec.ID, ChapterID: spec.Arguments["chapter_id"], Purpose: spec.Purpose, Action: spec.Action, Disposition: deliveryramp.StepPassed, Readiness: spec.Readiness, Settle: spec.Settle, StartedAt: time.Now().UTC(), MonotonicStartMs: time.Since(runStarted).Milliseconds()}
		if spec.Assertion != nil {
			step.AssertionID = spec.Assertion.ID
			step.ExpectedState = spec.Assertion.Expected
			step.AssertionStatus = "pending"
		}
		chapterID := spec.Arguments["chapter_id"]
		if perChapter && chapterActive && chapterID != activeChapter {
			if stopErr := stopChapter(ctx); stopErr != nil {
				return result, stopErr
			}
		}
		if strings.EqualFold(spec.Action, "not-applicable") {
			reason := strings.TrimSpace(spec.Arguments["reason"])
			if reason == "" {
				return result, fmt.Errorf("not-applicable step %q omitted its reason", spec.ID)
			}
			step.Disposition = deliveryramp.StepNotRun
			step.ObservedState = reason
			step.DegradedReason = reason
			step.CompletedAt = time.Now().UTC()
			step.MonotonicEndMs = time.Since(runStarted).Milliseconds()
			result.Events = append(result.Events, deliveryramp.JourneyEvent{
				Type: "chapter-not-applicable", StepID: step.ID, Observed: reason, Reason: reason,
				StartedAt: step.StartedAt, CompletedAt: step.CompletedAt,
				MonotonicStartMs: step.MonotonicStartMs, MonotonicEndMs: step.MonotonicEndMs, Source: "scenario-to-android",
			})
			result.Steps = append(result.Steps, step)
			continue
		}
		if reason, unsupported := unsupportedChapters[chapterID]; unsupported {
			step.Disposition = deliveryramp.StepUnavailable
			step.Error = reason
			step.DegradedReason = reason
			step.CompletedAt = time.Now().UTC()
			result.Steps = append(result.Steps, step)
			if result.Disposition == deliveryramp.DispositionNotRun {
				result.Disposition = deliveryramp.DispositionUnavailable
			}
			continue
		}
		if missing := missingCapability(spec.Arguments["required_capabilities"], request.Cell.Target.Capabilities); missing != "" {
			reason := fmt.Sprintf("chapter %q unavailable: target lacks required capability %q", chapterID, missing)
			unsupportedChapters[chapterID] = reason
			step.Disposition = deliveryramp.StepUnavailable
			step.Error = reason
			step.DegradedReason = reason
			step.CompletedAt = time.Now().UTC()
			result.Steps = append(result.Steps, step)
			if result.Disposition == deliveryramp.DispositionNotRun {
				result.Disposition = deliveryramp.DispositionUnavailable
			}
			continue
		}
		if perChapter && !chapterActive && videoUnavailable[chapterID] == "" {
			activeChapter = chapterID
			chapterStartStep = len(result.Steps)
			if err := chapterRecorder.StartChapterRecording(ctx, lease, activeChapter); err != nil {
				videoUnavailable[activeChapter] = fmt.Sprintf("start recording for chapter %q: %s", activeChapter, err)
				activeChapter = ""
				chapterStartStep = -1
			} else {
				chapterActive = true
			}
		}
		if strings.EqualFold(spec.Action, "bas-flow") {
			if d.BAS == nil {
				return result, fmt.Errorf("BAS client is unavailable for step %q", spec.ID)
			}
			basRequest := BASRequest{TargetID: request.Cell.Target.ID, Scenario: artifactScenario(request.Artifact, request.Cell.Target.Label), Artifact: request.Artifact, StepID: spec.ID, RunID: request.RunID, Arguments: spec.Arguments, FlowPath: strings.TrimSpace(spec.Arguments["flow_path"]), IsolationLeaseID: lease.ID}
			attacher := d.WebView
			if attacher == nil {
				attacher, _ = d.Devices.(WebViewAttacher)
			}
			packageName := ""
			if attacher != nil {
				if request.Artifact.Metadata != nil {
					packageName = request.Artifact.Metadata["package_name"]
				}
				if strings.TrimSpace(packageName) == "" {
					return result, fmt.Errorf("Android BAS flow %q requires an artifact package identity", spec.ID)
				}
			}
			var basResult BASResult
			var basErr error
			for attempt := 0; attempt < 2; attempt++ {
				if attacher != nil {
					var endpoint WebViewAttachment
					var attachErr error
					if strings.EqualFold(chapterID, "offline_transition") {
						if routed, ok := attacher.(TransportWebViewAttacher); ok {
							endpoint, attachErr = routed.AttachWebViewWithTransport(ctx, lease, packageName, "usb")
						} else {
							endpoint, attachErr = attacher.AttachWebView(ctx, lease, packageName)
						}
					} else {
						endpoint, attachErr = attacher.AttachWebView(ctx, lease, packageName)
					}
					if attachErr != nil {
						return result, fmt.Errorf("attach Android WebView for flow %q: %w", spec.ID, attachErr)
					}
					basRequest.CDPEndpoint = endpoint.CDPEndpoint
					basRequest.RendererID = endpoint.RendererID
					basRequest.RendererURL = endpoint.RendererURL
				}
				basResult, basErr = d.BAS.Execute(ctx, basRequest)
				if basErr == nil {
					break
				}
				transientClosure := isTransientWebViewClosure(basErr) || isTransientBASExecutionError(basErr)
				if !transientClosure || attempt == 1 {
					break
				}
				result.Events = append(result.Events, deliveryramp.JourneyEvent{Type: "bas-retry", StepID: step.ID, Observed: fmt.Sprintf("BAS attempt %d failed transiently; execution_id=%s; reason=%s", attempt+1, basResult.ExecutionID, basErr), StartedAt: step.StartedAt, CompletedAt: time.Now().UTC(), MonotonicStartMs: step.MonotonicStartMs, MonotonicEndMs: time.Since(runStarted).Milliseconds(), Source: "browser-automation-studio"})
				if err := waitForWebViewRetry(ctx); err != nil {
					return result, fmt.Errorf("BAS flow %q retry wait: %w", spec.ID, err)
				}
			}
			if basErr != nil {
				result.Events = append(result.Events, deliveryramp.JourneyEvent{Type: "bas-failure", StepID: step.ID, Observed: fmt.Sprintf("BAS execution failed; execution_id=%s; reason=%s", basResult.ExecutionID, basErr), StartedAt: step.StartedAt, CompletedAt: time.Now().UTC(), MonotonicStartMs: step.MonotonicStartMs, MonotonicEndMs: time.Since(runStarted).Milliseconds(), Source: "browser-automation-studio"})
				return result, fmt.Errorf("BAS flow %q failed: %w", spec.ID, basErr)
			}
			if !basResult.Completed {
				return result, fmt.Errorf("BAS flow %q did not complete", spec.ID)
			}
			step.Evidence = append(step.Evidence, basResult.Evidence...)
			// BAS completion is a producer-owned observation of the WebView
			// state, backed by the flow's evidence. Preserve it as an event so
			// the chapter assertion can be correlated even when Android does
			// not emit a second ActivityTaskManager/Capacitor line after an
			// already-running WebView is attached.
			result.Events = append(result.Events, deliveryramp.JourneyEvent{
				Type:             "bas-observation",
				StepID:           step.ID,
				Observed:         basExecutionObservation(basResult.ExecutionID),
				StartedAt:        step.StartedAt,
				CompletedAt:      time.Now().UTC(),
				MonotonicStartMs: step.MonotonicStartMs,
				MonotonicEndMs:   time.Since(runStarted).Milliseconds(),
				Source:           "browser-automation-studio",
			})
		} else {
			arguments := map[string]string{"step_id": spec.ID, "target": spec.Arguments["target"], "reference": spec.Arguments["reference"]}
			if timeoutMS := strings.TrimSpace(spec.Arguments["timeout_ms"]); timeoutMS != "" {
				arguments["timeout_ms"] = timeoutMS
			}
			if target := strings.TrimSpace(spec.Arguments["target"]); target != "" {
				arguments["value"] = target
			}
			if strings.EqualFold(spec.Action, "package-state") {
				// device-control distinguishes the package identity from the
				// expected installation state. The conformance step encodes the
				// latter in target, so preserve it under the protocol's expected
				// argument instead of silently sending an empty value.
				arguments["expected"] = spec.Arguments["target"]
			}
			if strings.EqualFold(spec.Action, "grant-permission") || strings.EqualFold(spec.Action, "revoke-permission") {
				arguments["permission"] = spec.Arguments["target"]
			}
			deviceAction := spec.Action
			if request.Artifact.Metadata != nil {
				arguments["package"] = request.Artifact.Metadata["package_name"]
				if strings.EqualFold(spec.Action, "install") {
					deviceAction = "install-artifact"
					arguments["artifact_ref"] = request.Artifact.ImmutableRef
					arguments["version_name"] = request.Artifact.Metadata["version_name"]
				}
			}
			var actionResult ActionResult
			var actionErr error
			if installMode != "" && strings.EqualFold(spec.Action, "screen") && strings.EqualFold(spec.Arguments["target"], "unlock") && initialDeviceState.LockState == "unlocked" && strings.TrimSpace(request.Artifact.Metadata["auth_profile_id"]) == "" {
				actionResult = ActionResult{Observed: "already unlocked"}
			} else if strings.EqualFold(spec.Action, "screen") && strings.EqualFold(spec.Arguments["target"], "unlock") {
				profileID := ""
				if request.Artifact.Metadata != nil {
					profileID = strings.TrimSpace(request.Artifact.Metadata["auth_profile_id"])
				}
				if profileID != "" {
					authenticator, ok := d.Devices.(Authenticator)
					if !ok {
						actionErr = fmt.Errorf("device-control client cannot use Android auth profile %q", profileID)
					} else {
						actionErr = authenticator.Unlock(ctx, lease, profileID)
						actionResult = ActionResult{Observed: "unlocked"}
					}
				}
			}
			if actionErr == nil && actionResult.Observed == "" {
				if strings.EqualFold(chapterID, "offline_transition") && strings.EqualFold(spec.Action, "network") && strings.EqualFold(spec.Arguments["target"], "offline") {
					networkRestoreNeeded = true
				}
				actionResult, actionErr = d.Devices.Execute(ctx, lease, deviceAction, arguments)
				if actionErr == nil && strings.EqualFold(spec.Action, "network") && strings.EqualFold(spec.Arguments["target"], "online") {
					networkRestoreNeeded = false
				}
			}
			if actionErr != nil {
				step.Disposition = deliveryramp.StepFailed
				step.Error = fmt.Sprintf("device-control step %q failed: %s", spec.ID, actionErr)
				result.Disposition = deliveryramp.DispositionFailed
				result.DegradedReason = step.Error
			}
			if actionErr == nil {
				step.Evidence = append(step.Evidence, actionResult.Evidence...)
				result.Events = append(result.Events, deliveryramp.JourneyEvent{Type: "device-action", StepID: step.ID, Observed: spec.Action, StartedAt: step.StartedAt, CompletedAt: time.Now().UTC(), MonotonicStartMs: step.MonotonicStartMs, MonotonicEndMs: time.Since(runStarted).Milliseconds(), Source: "device-control"})
			}
		}
		step.CompletedAt = time.Now().UTC()
		step.MonotonicEndMs = time.Since(runStarted).Milliseconds()
		if reason := videoUnavailable[chapterID]; reason != "" {
			step.VideoDisposition = deliveryramp.StepUnavailable
			step.VideoError = reason
		}
		result.Steps = append(result.Steps, step)
		if perChapter && (index == len(request.Plan.Steps)-1 || request.Plan.Steps[index+1].Arguments["chapter_id"] != activeChapter) {
			if stopErr := stopChapter(ctx); stopErr != nil {
				return result, stopErr
			}
		}
		if step.Disposition == deliveryramp.StepFailed {
			break
		}
	}
	if perChapter {
		if hasReviewRecorder && len(chapterReferences) > 0 {
			review, reviewErr := reviewRecorder.FinalizeReviewRecording(ctx, lease, chapterReferences)
			if reviewErr != nil {
				result.DegradedReason = fmt.Sprintf("review recording unavailable: %s", reviewErr)
				result.Disposition = deliveryramp.DispositionDegraded
			} else if review.Reference.ID == "" || review.Reference.Checksum == "" || !review.Reference.Redacted {
				result.DegradedReason = "review recording reference is incomplete or not redacted"
				result.Disposition = deliveryramp.DispositionDegraded
			} else if !filepath.IsAbs(review.Path) {
				result.DegradedReason = "review recording path is missing or not absolute"
				result.Disposition = deliveryramp.DispositionDegraded
			} else {
				result.ReviewRecording = &review.Reference
				result.ReviewRecordingPath = review.Path
			}
		}
		if result.Disposition == deliveryramp.DispositionNotRun {
			result.Disposition = deliveryramp.DispositionPass
		}
		result.CompletedAt = time.Now().UTC()
		return result, nil
	}
	if installMode == "verify" {
		finalizeJourneyAssertions(&result)
		if result.Disposition == deliveryramp.DispositionNotRun {
			result.Disposition = deliveryramp.DispositionPass
		}
		result.CompletedAt = time.Now().UTC()
		return result, nil
	}
	recording, err := d.Devices.StopRecording(ctx, lease)
	if err != nil {
		return result, fmt.Errorf("stop device recording: %w", err)
	}
	if !recording.HasOffsets || recording.StartMs < 0 || recording.EndMs < recording.StartMs {
		return result, fmt.Errorf("recording is missing bounded start/end offsets")
	}
	if recording.Reference.ID == "" || recording.Reference.Checksum == "" || !recording.Reference.Redacted {
		return result, fmt.Errorf("recording reference is incomplete or not redacted")
	}
	if len(result.Steps) > 0 {
		result.Steps[len(result.Steps)-1].Evidence = append(result.Steps[len(result.Steps)-1].Evidence, recording.Reference)
	}
	if len(result.Steps) > 0 {
		start, end := recording.StartMs, recording.EndMs
		result.Steps[0].VideoStartOffsetMs = &start
		result.Steps[len(result.Steps)-1].VideoEndOffsetMs = &end
	}
	if result.Disposition == deliveryramp.DispositionNotRun {
		result.Disposition = deliveryramp.DispositionPass
	}
	result.CompletedAt = time.Now().UTC()
	return result, nil
}

func appendUpdateFailurePackageReadback(result *deliveryramp.JourneyResult, devices DeviceClient, lease Lease, packageName string, previous InstalledPackageState, candidateVersionCode int64) {
	step := deliveryramp.JourneyStep{
		ID: "update-recovery/installed-package-state", Name: "update-recovery/installed-package-state",
		Purpose: "read back the durable package after an update error without retrying an uncertain install",
		Action:  "package-state", StartedAt: time.Now().UTC(),
		ExpectedState: fmt.Sprintf("retain prior versionCode %d or identify candidate versionCode %d", previous.VersionCode, candidateVersionCode),
		AssertionID:   "update-recovery/installed-version-readback",
	}
	reader, ok := devices.(PackageStateReader)
	if !ok {
		step.Disposition = deliveryramp.StepUnavailable
		step.AssertionStatus = "unknown"
		step.Error = "Device Control package version readback is unavailable after update error"
	} else {
		readbackCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		state, err := reader.ReadPackageState(readbackCtx, lease, packageName)
		if err != nil {
			step.Disposition = deliveryramp.StepFailed
			step.AssertionStatus = "unknown"
			step.Error = "Device Control could not determine the installed package after update error: " + err.Error()
		} else {
			step.ObservedState = fmt.Sprintf("installed=%t; versionName=%s; versionCode=%d", state.Installed, state.Version, state.VersionCode)
			step.AssertionStatus = "observed"
			switch {
			case !state.Installed:
				step.Disposition = deliveryramp.StepFailed
				step.Error = "scenario package is absent after update error"
			case state.VersionCode <= 0:
				step.Disposition = deliveryramp.StepUnavailable
				step.AssertionStatus = "unknown"
				step.Error = "scenario package is present after update error, but its installed version code is unknown"
			case state.VersionCode == previous.VersionCode:
				step.Disposition = deliveryramp.StepPassed
				step.DegradedReason = "the prior durable version remains installed; update was not retried"
			case state.VersionCode == candidateVersionCode:
				step.Disposition = deliveryramp.StepPassed
				step.DegradedReason = "the candidate version is installed despite the update error; post-update BAS verification is still required and no retry was attempted"
			default:
				step.Disposition = deliveryramp.StepFailed
				step.Error = "installed package version matches neither the prior nor candidate version"
			}
		}
	}
	step.CompletedAt = time.Now().UTC()
	result.Steps = append(result.Steps, step)
	if step.Error != "" {
		result.DegradedReason += "; " + step.Error
	} else if step.DegradedReason != "" {
		result.DegradedReason += "; " + step.DegradedReason
	}
}

func finalizeJourneyAssertions(result *deliveryramp.JourneyResult) {
	if result == nil {
		return
	}
	result.Steps = correlateLogcatEvents(result.Steps, result.Events, result.ClockOffsetStart)
	if assertionFailures := assertionFailureCount(result.Steps); assertionFailures > 0 {
		result.Disposition = deliveryramp.DispositionFailed
		assertionReason := fmt.Sprintf("%d journey assertions were not observed in device evidence: %s", assertionFailures, failedAssertionSummary(result.Steps))
		if strings.TrimSpace(result.DegradedReason) == "" {
			result.DegradedReason = assertionReason
		} else {
			result.DegradedReason += "; " + assertionReason
		}
	}
}

func failAppInstallPreflight(result *deliveryramp.JourneyResult, message string) (deliveryramp.JourneyResult, error) {
	step := deliveryramp.JourneyStep{
		ID: "install-preflight/device-state", Name: "install-preflight/device-state",
		Purpose: "refuse the scenario-app operation when pre-test device state cannot be safely restored",
		Action:  "device-state", Disposition: deliveryramp.StepFailed,
		StartedAt: time.Now().UTC(), CompletedAt: time.Now().UTC(), Error: message,
	}
	result.Steps = append(result.Steps, step)
	result.Disposition = deliveryramp.DispositionFailed
	result.DegradedReason = message
	return *result, nil
}

func failBASPreflight(result *deliveryramp.JourneyResult, message string) (deliveryramp.JourneyResult, error) {
	step := deliveryramp.JourneyStep{
		ID: "install-preflight/bas-workflows", Name: "install-preflight/bas-workflows",
		Purpose: "refuse installation when a required scenario BAS workflow is unavailable",
		Action:  "bas-flow", Disposition: deliveryramp.StepFailed,
		StartedAt: time.Now().UTC(), CompletedAt: time.Now().UTC(), Error: message,
	}
	result.Steps = append(result.Steps, step)
	result.Disposition = deliveryramp.DispositionFailed
	result.DegradedReason = message
	return *result, nil
}

func failInstallPackagePreflight(result *deliveryramp.JourneyResult, message string) (deliveryramp.JourneyResult, error) {
	step := deliveryramp.JourneyStep{
		ID: "install-preflight/package-state", Name: "install-preflight/package-state",
		Purpose: "refuse an update when the installed package or version cannot be verified",
		Action:  "package-state", Disposition: deliveryramp.StepFailed,
		StartedAt: time.Now().UTC(), CompletedAt: time.Now().UTC(), Error: message,
	}
	result.Steps = append(result.Steps, step)
	result.Disposition = deliveryramp.DispositionFailed
	result.DegradedReason = message
	return *result, nil
}

func validateRestorableAndroidState(state DeviceStateSnapshot, authProfileID string) error {
	if state.ScreenState != "on" && state.ScreenState != "off" {
		return fmt.Errorf("pre-test screen state %q is unknown", state.ScreenState)
	}
	if state.LockState != "locked" && state.LockState != "unlocked" {
		return fmt.Errorf("pre-test lock state %q is unknown", state.LockState)
	}
	if state.ScreenState == "off" && state.LockState == "unlocked" {
		return fmt.Errorf("screen-off/unlocked state cannot be restored reliably")
	}
	if state.LockState == "locked" && strings.TrimSpace(authProfileID) == "" {
		return fmt.Errorf("a Device Control auth profile is required to test a locked device")
	}
	return nil
}

func restoreAndroidScreenState(ctx context.Context, devices DeviceClient, lease Lease, initial DeviceStateSnapshot, authProfileID string) error {
	if initial.LockState == "locked" {
		if _, err := devices.Execute(ctx, lease, "screen", map[string]string{"step_id": "device-cleanup/screen-sleep", "target": "sleep"}); err != nil {
			return fmt.Errorf("lock screen before wake: %w", err)
		}
		if _, err := waitForAndroidScreenLockState(ctx, devices, lease, "off", "locked"); err != nil {
			return fmt.Errorf("verify screen sleep before restore: %w", err)
		}
		if initial.ScreenState == "on" {
			if _, err := devices.Execute(ctx, lease, "screen", map[string]string{"step_id": "device-cleanup/screen-wake", "target": "wake"}); err != nil {
				return fmt.Errorf("restore screen-on state: %w", err)
			}
			if _, err := waitForAndroidScreenLockState(ctx, devices, lease, "on", "locked"); err != nil {
				return fmt.Errorf("verify screen-on restore: %w", err)
			}
		}
	}
	reader, ok := devices.(DeviceStateReader)
	if !ok {
		return fmt.Errorf("Device Control state readback is unavailable")
	}
	current, err := reader.ReadState(ctx, lease.DeviceID)
	if err != nil {
		return fmt.Errorf("verify restored state: %w", err)
	}
	if initial.ScreenState == "on" && current.ScreenState != "on" {
		if _, err := devices.Execute(ctx, lease, "screen", map[string]string{"step_id": "device-cleanup/screen-wake", "target": "wake"}); err != nil {
			return fmt.Errorf("restore screen-on state: %w", err)
		}
		current, err = reader.ReadState(ctx, lease.DeviceID)
		if err != nil {
			return fmt.Errorf("verify screen-on state: %w", err)
		}
	}
	if initial.LockState == "unlocked" && current.LockState != "unlocked" {
		if strings.TrimSpace(authProfileID) == "" {
			return fmt.Errorf("restoring the initially unlocked state requires an auth profile")
		}
		authenticator, ok := devices.(Authenticator)
		if !ok {
			return fmt.Errorf("Device Control auth profile support is unavailable")
		}
		if err := authenticator.Unlock(ctx, lease, authProfileID); err != nil {
			return fmt.Errorf("restore unlocked state: %w", err)
		}
		current, err = reader.ReadState(ctx, lease.DeviceID)
		if err != nil {
			return fmt.Errorf("verify unlocked state: %w", err)
		}
	}
	if current.ScreenState != initial.ScreenState || current.LockState != initial.LockState {
		return fmt.Errorf("restored screen=%s lock=%s; want screen=%s lock=%s", current.ScreenState, current.LockState, initial.ScreenState, initial.LockState)
	}
	return nil
}

func waitForAndroidScreenLockState(ctx context.Context, devices DeviceClient, lease Lease, screen, lock string) (DeviceStateSnapshot, error) {
	reader, ok := devices.(DeviceStateReader)
	if !ok {
		return DeviceStateSnapshot{}, fmt.Errorf("Device Control state readback is unavailable")
	}
	var current DeviceStateSnapshot
	for attempt := 0; attempt < 11; attempt++ {
		state, err := reader.ReadState(ctx, lease.DeviceID)
		if err != nil {
			return DeviceStateSnapshot{}, fmt.Errorf("read state while waiting for screen=%s lock=%s: %w", screen, lock, err)
		}
		current = state
		if state.ScreenState == screen && state.LockState == lock {
			return state, nil
		}
		if attempt == 10 {
			break
		}
		timer := time.NewTimer(500 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return current, ctx.Err()
		case <-timer.C:
		}
	}
	return current, fmt.Errorf("screen=%s lock=%s after settling; want screen=%s lock=%s", current.ScreenState, current.LockState, screen, lock)
}

func basExecutionObservation(executionID string) string {
	if id := strings.TrimSpace(executionID); id != "" {
		return "BAS flow completed; execution_id=" + id
	}
	return "BAS flow completed; execution_id unavailable"
}

func isTransientWebViewClosure(err error) bool {
	if isTransientBASExecutionError(err) {
		return true
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "target page, context or browser has been closed") || strings.Contains(message, "target page or context has been closed")
}

func waitForWebViewRetry(ctx context.Context) error {
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func missingCapability(declared string, available []string) string {
	for _, required := range strings.Split(declared, ",") {
		required = strings.ToLower(strings.TrimSpace(required))
		if required == "" {
			continue
		}
		if capabilityPresent(required, available) {
			continue
		}
		return required
	}
	return ""
}

func capabilityPresent(required string, available []string) bool {
	for _, raw := range available {
		value := strings.ToLower(strings.TrimSpace(raw))
		if value == required || strings.ReplaceAll(value, "_", "-") == required {
			return true
		}
		if required == "network-control" && strings.Contains(value, "network_control") {
			return true
		}
		if required != "network-control" && required != "webview-attach" && (value == "device-control" || strings.Contains(value, "native_window")) {
			return true
		}
		if required == "webview-attach" && (value == "android-webview" || strings.Contains(value, "electron_cdp")) {
			return true
		}
	}
	return false
}

func artifactScenario(artifact deliveryramp.Artifact, fallback string) string {
	if artifact.Metadata != nil && strings.TrimSpace(artifact.Metadata["scenario_name"]) != "" {
		return artifact.Metadata["scenario_name"]
	}
	return fallback
}

func parseLogcatEvents(lines []string) []deliveryramp.JourneyEvent {
	events := make([]deliveryramp.JourneyEvent, 0)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		fields := strings.Fields(trimmed)
		if len(fields) == 0 {
			continue
		}
		timestamp, err := strconv.ParseFloat(fields[0], 64)
		if err != nil {
			continue
		}
		lower := strings.ToLower(trimmed)
		typeName := ""
		switch {
		case strings.Contains(lower, "activitytaskmanager:") && strings.Contains(lower, " start "):
			typeName = "activity-start"
		case strings.Contains(lower, "activitytaskmanager:") && strings.Contains(lower, " displayed "):
			typeName = "first-frame"
		case strings.Contains(lower, "capacitor") && (strings.Contains(lower, "load") || strings.Contains(lower, "ready")):
			typeName = "webview-ready"
		}
		if typeName == "" {
			continue
		}
		seconds := int64(timestamp)
		nanos := int64((timestamp - float64(seconds)) * 1e9)
		observedAt := time.Unix(seconds, nanos).UTC()
		events = append(events, deliveryramp.JourneyEvent{Type: typeName, Observed: trimmed, StartedAt: observedAt, CompletedAt: observedAt, DeviceTimestamp: timestamp, Source: "logcat", Raw: trimmed})
	}
	return events
}

func correlateLogcatEvents(steps []deliveryramp.JourneyStep, events []deliveryramp.JourneyEvent, samples ...*deliveryramp.ClockOffsetSample) []deliveryramp.JourneyStep {
	var offsetMs int64
	if len(samples) > 0 && samples[0] != nil {
		offsetMs = samples[0].OffsetMs
	}
	for eventIndex := range events {
		for stepIndex := range steps {
			step := &steps[stepIndex]
			event := events[eventIndex]
			hostEventTime := event.StartedAt.Add(time.Duration(offsetMs) * time.Millisecond)
			if event.Type == "device-action" || event.Type == "bas-observation" {
				hostEventTime = event.StartedAt
			}
			if hostEventTime.Before(step.StartedAt.Add(-2*time.Second)) || hostEventTime.After(step.CompletedAt.Add(2*time.Second)) {
				continue
			}
			if !eventMatchesStep(step, event) {
				continue
			}
			events[eventIndex].StepID = step.ID
			events[eventIndex].MonotonicStartMs = step.MonotonicStartMs
			events[eventIndex].MonotonicEndMs = step.MonotonicEndMs
			if step.AssertionStatus == "" || step.AssertionStatus == "pending" {
				step.AssertionStatus = "observed"
			}
			break
		}
	}
	for index := range steps {
		step := &steps[index]
		if step.AssertionID == "" || step.Disposition != deliveryramp.StepPassed || step.AssertionStatus == "observed" {
			continue
		}
		step.AssertionStatus = "failed"
		step.Error = fmt.Sprintf("expected device evidence %q was not observed", step.ExpectedState)
		step.Disposition = deliveryramp.StepFailed
	}
	return steps
}

func eventMatchesStep(step *deliveryramp.JourneyStep, event deliveryramp.JourneyEvent) bool {
	if event.Type == "bas-observation" {
		// BAS observations carry the exact step identity assigned by the
		// driver. Prefer that identity over the clock tolerance below: adjacent
		// flows (for example offline then online) can otherwise be attributed to
		// the preceding step when their completion timestamps are close.
		return event.StepID == step.ID && strings.EqualFold(step.Action, "bas-flow")
	}
	if event.Type == "device-action" {
		switch strings.ToLower(step.Action) {
		case "launch", "observe", "bas-flow":
			return false
		}
		return event.StepID == step.ID
	}
	switch strings.ToLower(step.Action) {
	case "launch":
		return event.Type == "activity-start" || event.Type == "first-frame"
	case "observe", "bas-flow":
		return event.Type == "first-frame" || event.Type == "webview-ready"
	default:
		return event.Type == "device-action"
	}
}

func assertionFailureCount(steps []deliveryramp.JourneyStep) int {
	count := 0
	for _, step := range steps {
		if step.AssertionStatus == "failed" {
			count++
		}
	}
	return count
}

func failedAssertionSummary(steps []deliveryramp.JourneyStep) string {
	failed := make([]string, 0)
	for _, step := range steps {
		if step.AssertionStatus == "failed" {
			failed = append(failed, fmt.Sprintf("%s (%s)", step.ID, step.ExpectedState))
		}
	}
	return strings.Join(failed, ", ")
}

func clockOffsetSample(sample ClockSample) *deliveryramp.ClockOffsetSample {
	return &deliveryramp.ClockOffsetSample{CapturedAt: sample.HostTime, HostTime: sample.HostTime, DeviceTime: sample.DeviceTime, OffsetMs: sample.OffsetMs, UncertaintyMs: sample.UncertaintyMs, Evidence: sample.Evidence}
}
