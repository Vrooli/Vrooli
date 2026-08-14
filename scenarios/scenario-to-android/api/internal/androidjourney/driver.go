// Package androidjourney adapts scenario-to-android's device-control and BAS
// clients to the delivery-ramp Driver seam. Device verbs are deliberately
// interfaces: the production implementation is the device-control client,
// while tests can prove lease and evidence invariants without adb.
package androidjourney

import (
	"context"
	"fmt"
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
	StartMs    int64
	EndMs      int64
	HasOffsets bool
}

type DeviceClient interface {
	Acquire(context.Context, string, string, time.Duration) (Lease, error)
	ValidateLease(context.Context, Lease) error
	Execute(context.Context, Lease, string, map[string]string) (ActionResult, error)
	StartRecording(context.Context, Lease) error
	StopRecording(context.Context, Lease) (RecordingArtifact, error)
	Release(context.Context, Lease) error
}

// WebViewAttacher is optional on DeviceClient so existing native-only test
// doubles remain small. The production device-control client implements it;
// the driver then supplies BAS with a lease-scoped forwarded CDP endpoint.
type WebViewAttacher interface {
	AttachWebView(context.Context, Lease, string) (WebViewAttachment, error)
}

type WebViewAttachment struct {
	CDPEndpoint string
	RendererID  string
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
	FlowPath         string
	IsolationLeaseID string
}

type BASResult struct {
	Completed bool
	Evidence  []deliveryramp.EvidenceReference
}

type BASClient interface {
	Execute(context.Context, BASRequest) (BASResult, error)
}

type Driver struct {
	Devices  DeviceClient
	BAS      BASClient
	WebView  WebViewAttacher
	Actor    string
	LeaseTTL time.Duration
}

var _ deliveryramp.Driver = Driver{}

func (d Driver) Execute(ctx context.Context, request deliveryramp.DriverRequest) (deliveryramp.JourneyResult, error) {
	result := deliveryramp.JourneyResult{
		SchemaVersion: deliveryramp.JourneySchemaVersion, EvidenceVersion: deliveryramp.JourneyEvidenceVersion,
		SmokeTestID: request.RunID, PlanID: request.Plan.ID, Profile: request.Plan.Profile,
		Capability: request.Plan.Capability, TargetID: request.Cell.Target.ID, CellID: request.Cell.ID,
		ScenarioName: artifactScenario(request.Artifact, request.Cell.Target.Label), Platform: request.Cell.Target.Platform,
		Disposition: deliveryramp.DispositionNotRun, Steps: make([]deliveryramp.JourneyStep, 0, len(request.Plan.Steps)),
		CreatedAt: time.Now().UTC(),
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
	ttl := d.LeaseTTL
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	lease, err := d.Devices.Acquire(ctx, request.Cell.Target.ID, d.Actor, ttl)
	if err != nil {
		return result, fmt.Errorf("acquire device-control lease: %w", err)
	}
	defer func() { _ = d.Devices.Release(context.Background(), lease) }()
	if err := d.Devices.StartRecording(ctx, lease); err != nil {
		return result, fmt.Errorf("start device recording before launch: %w", err)
	}
	unsupportedChapters := make(map[string]string)
	for _, spec := range request.Plan.Steps {
		if err := d.Devices.ValidateLease(ctx, lease); err != nil {
			return result, fmt.Errorf("device-control lease lost before step %q: %w", spec.ID, err)
		}
		step := deliveryramp.JourneyStep{ID: spec.ID, Name: spec.ID, Purpose: spec.Purpose, Action: spec.Action, Disposition: deliveryramp.StepPassed, Readiness: spec.Readiness, Settle: spec.Settle, StartedAt: time.Now().UTC()}
		chapterID := spec.Arguments["chapter_id"]
		if reason, unsupported := unsupportedChapters[chapterID]; unsupported {
			step.Disposition = deliveryramp.StepUnavailable
			step.Error = reason
			step.DegradedReason = reason
			step.CompletedAt = time.Now().UTC()
			result.Steps = append(result.Steps, step)
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
			continue
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
			if attacher != nil {
				packageName := ""
				if request.Artifact.Metadata != nil {
					packageName = request.Artifact.Metadata["package_name"]
				}
				if strings.TrimSpace(packageName) == "" {
					return result, fmt.Errorf("Android BAS flow %q requires an artifact package identity", spec.ID)
				}
				endpoint, attachErr := attacher.AttachWebView(ctx, lease, packageName)
				if attachErr != nil {
					return result, fmt.Errorf("attach Android WebView for flow %q: %w", spec.ID, attachErr)
				}
				basRequest.CDPEndpoint = endpoint.CDPEndpoint
				basRequest.RendererID = endpoint.RendererID
			}
			basResult, basErr := d.BAS.Execute(ctx, basRequest)
			if basErr != nil {
				return result, fmt.Errorf("BAS flow %q failed: %w", spec.ID, basErr)
			}
			if !basResult.Completed {
				return result, fmt.Errorf("BAS flow %q did not complete", spec.ID)
			}
			step.Evidence = append(step.Evidence, basResult.Evidence...)
		} else {
			arguments := map[string]string{"step_id": spec.ID, "target": spec.Arguments["target"], "reference": spec.Arguments["reference"]}
			if strings.EqualFold(spec.Action, "package-state") {
				// device-control distinguishes the package identity from the
				// expected installation state. The conformance step encodes the
				// latter in target, so preserve it under the protocol's expected
				// argument instead of silently sending an empty value.
				arguments["expected"] = spec.Arguments["target"]
			}
			if request.Artifact.Metadata != nil {
				arguments["package"] = request.Artifact.Metadata["package_name"]
				arguments["value"] = request.Artifact.Metadata["apk_path"]
			}
			actionResult, actionErr := d.Devices.Execute(ctx, lease, spec.Action, arguments)
			if actionErr != nil {
				return result, fmt.Errorf("device-control step %q failed: %w", spec.ID, actionErr)
			}
			step.Evidence = append(step.Evidence, actionResult.Evidence...)
		}
		step.CompletedAt = time.Now().UTC()
		result.Steps = append(result.Steps, step)
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
	result.Disposition = deliveryramp.DispositionPass
	result.CompletedAt = time.Now().UTC()
	return result, nil
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
