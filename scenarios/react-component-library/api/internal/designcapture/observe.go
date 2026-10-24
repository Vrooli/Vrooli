package designcapture

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	apiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	basev1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	timelinev1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

type Observation struct {
	State     State
	Artifacts []Artifact
	Detail    string
}
type Observer interface {
	Observe(context.Context, Operation) (Observation, error)
}

// Attach performs one producer observation. It never restarts or busy-polls.
// Observation errors leave the existing durable state untouched.
func (s Service) Attach(ctx context.Context, id string, observer Observer) (Operation, error) {
	if s.Repository == nil || observer == nil {
		return Operation{}, fmt.Errorf("capture repository and producer observer are required")
	}
	op, err := s.Repository.Get(ctx, id)
	if err != nil {
		return op, err
	}
	if op.State == Completed || op.State == Failed || op.State == Cancelled {
		return op, nil
	}
	if op.State != Running && op.State != CancelRequested {
		return op, fmt.Errorf("capture has no acknowledged producer to attach")
	}
	observation, err := observer.Observe(ctx, op)
	if err != nil {
		return op, err
	}
	if observation.State == Running {
		return op, nil
	}
	if observation.State != Completed && observation.State != Failed && observation.State != Cancelled {
		return op, fmt.Errorf("invalid producer terminal state")
	}
	return s.Repository.Transition(ctx, op.ID, op.Version, observation.State, op.ProducerID, observation.Artifacts, observation.Detail)
}
func (d BASDispatcher) Observe(ctx context.Context, op Operation) (Observation, error) {
	if op.ProducerID == "" {
		return Observation{}, fmt.Errorf("producer identity is required")
	}
	raw, err := protojson.Marshal(&apiv1.GetExecutionTimelineRequest{ExecutionId: op.ProducerID})
	if err != nil {
		return Observation{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(d.BASBaseURL, "/")+"/browser_automation_studio.v1.ExecutionsService/GetExecutionTimeline", strings.NewReader(string(raw)))
	if err != nil {
		return Observation{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := d.Client
	if client == nil {
		client = http.DefaultClient
	}
	response, err := client.Do(req)
	if err != nil {
		return Observation{}, err
	}
	defer response.Body.Close()
	// BAS timelines can embed DOM and images. Bound the producer payload even
	// though only typed target evidence and opaque artifact identities are retained.
	body, err := io.ReadAll(io.LimitReader(response.Body, 16*1024*1024+1))
	if err != nil {
		return Observation{}, err
	}
	if len(body) > 16*1024*1024 {
		return Observation{}, fmt.Errorf("BAS timeline exceeds evidence limit")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return Observation{}, fmt.Errorf("BAS timeline returned HTTP %d", response.StatusCode)
	}
	var timeline timelinev1.ExecutionTimeline
	if err := protojson.Unmarshal(body, &timeline); err != nil {
		return Observation{}, err
	}
	return observeTimeline(op, &timeline)
}
func observeTimeline(op Operation, timeline *timelinev1.ExecutionTimeline) (Observation, error) {
	if timeline.GetExecutionId() != op.ProducerID {
		return Observation{}, fmt.Errorf("producer timeline identity mismatch")
	}
	switch timeline.GetStatus() {
	case basev1.ExecutionStatus_EXECUTION_STATUS_PENDING, basev1.ExecutionStatus_EXECUTION_STATUS_RUNNING:
		return Observation{State: Running}, nil
	case basev1.ExecutionStatus_EXECUTION_STATUS_FAILED:
		return Observation{State: Failed, Detail: "BAS reported capture failure."}, nil
	case basev1.ExecutionStatus_EXECUTION_STATUS_CANCELLED:
		return Observation{State: Cancelled, Detail: "BAS confirmed capture cancellation."}, nil
	case basev1.ExecutionStatus_EXECUTION_STATUS_COMPLETED:
	default:
		return Observation{}, fmt.Errorf("unrecognized BAS capture status")
	}
	if len(timeline.GetEntries()) != 3 {
		return Observation{}, fmt.Errorf("capture timeline does not contain the three required steps")
	}
	expected := []string{"open", "ready", "target"}
	for i, entry := range timeline.GetEntries() {
		if entry.GetNodeId() != expected[i] || !entry.GetContext().GetSuccess() {
			return Observation{}, fmt.Errorf("capture step %s is missing or failed", expected[i])
		}
	}
	target := timeline.GetEntries()[2]
	screenshot := target.GetTelemetry().GetScreenshot()
	if screenshot.GetArtifactId() == "" {
		return Observation{}, fmt.Errorf("capture lacks producer-owned screenshot identity")
	}
	screenshotOwned := false
	for _, a := range target.GetAggregates().GetArtifacts() {
		if a.GetId() == screenshot.GetArtifactId() && a.GetType() == basev1.ArtifactType_ARTIFACT_TYPE_SCREENSHOT {
			screenshotOwned = true
		}
	}
	if !screenshotOwned {
		return Observation{}, fmt.Errorf("screenshot is not owned by the verified target step")
	}
	var evidence *TargetEvidence
	var targetArtifact string
	for _, a := range target.GetAggregates().GetArtifacts() {
		result := a.GetPayload()["value"].GetObjectValue().GetFields()["result"].GetObjectValue().GetFields()
		if result["renderHash"] == nil {
			continue
		}
		if evidence != nil {
			return Observation{}, fmt.Errorf("ambiguous target evidence")
		}
		viewport := result["viewport"].GetObjectValue().GetFields()
		w, err := jsonNumber(viewport["width"])
		if err != nil {
			return Observation{}, err
		}
		h, err := jsonNumber(viewport["height"])
		if err != nil {
			return Observation{}, err
		}
		evidence = &TargetEvidence{RenderHash: result["renderHash"].GetStringValue(), Width: w, Height: h}
		for _, item := range result["regions"].GetListValue().GetValues() {
			fields := item.GetObjectValue().GetFields()
			r := RegionGeometry{Region: fields["region"].GetStringValue()}
			for key, destination := range map[string]*float64{"x": &r.X, "y": &r.Y, "width": &r.Width, "height": &r.Height} {
				n, err := jsonNumber(fields[key])
				if err != nil {
					return Observation{}, err
				}
				*destination = n
			}
			evidence.Regions = append(evidence.Regions, r)
		}
		targetArtifact = a.GetId()
	}
	if evidence == nil || targetArtifact == "" {
		return Observation{}, fmt.Errorf("capture lacks structured target evidence")
	}
	artifacts := []Artifact{{Kind: "screenshot", Reference: "bas:" + op.ProducerID + ":" + screenshot.GetArtifactId()}, {Kind: "target", Reference: "bas:" + op.ProducerID + ":" + targetArtifact, Evidence: evidence}}
	verified := op
	verified.State = Completed
	verified.Artifacts = artifacts
	if err := validateOperation(verified); err != nil {
		return Observation{}, err
	}
	return Observation{State: Completed, Artifacts: artifacts}, nil
}
func jsonNumber(v *commonv1.JsonValue) (float64, error) {
	switch n := v.GetKind().(type) {
	case *commonv1.JsonValue_IntValue:
		return float64(n.IntValue), nil
	case *commonv1.JsonValue_DoubleValue:
		return n.DoubleValue, nil
	}
	return 0, fmt.Errorf("capture geometry requires numeric evidence")
}

// Cancel delegates to BAS's idempotent stop operation. The response acknowledges
// delivery only; even "stopped" must not be converted into terminal evidence.
func (d BASDispatcher) Cancel(ctx context.Context, op Operation) error {
	if op.State != CancelRequested || op.ProducerID == "" {
		return fmt.Errorf("persisted cancellation intent and producer identity are required")
	}
	raw, err := protojson.Marshal(&apiv1.StopExecutionRequest{ExecutionId: op.ProducerID})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(d.BASBaseURL, "/")+"/browser_automation_studio.v1.ExecutionsService/StopExecution", strings.NewReader(string(raw)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := d.Client
	if client == nil {
		client = http.DefaultClient
	}
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 64*1024+1))
	if err != nil {
		return err
	}
	if len(body) > 64*1024 {
		return fmt.Errorf("BAS stop acknowledgement exceeds limit")
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("BAS stop returned HTTP %d", response.StatusCode)
	}
	var ack apiv1.StopExecutionResponse
	if err := protojson.Unmarshal(body, &ack); err != nil {
		return err
	}
	if ack.GetStatus() != "stopped" {
		return fmt.Errorf("BAS did not acknowledge stop request")
	}
	return nil
}
