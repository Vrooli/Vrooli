package workflows

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/vrooli/api-core/discovery"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const ProfileKey = "react-component-library/catalog-maintainer"

// AgentManagerDispatcher resolves Agent Manager on each operation through the
// lifecycle discovery seam. It talks in generated protobuf messages over the
// published HTTP surface, never shells out from a handler.
type AgentManagerDispatcher struct {
	Resolver *discovery.Resolver
	Client   *http.Client
}

func NewAgentManagerDispatcher() *AgentManagerDispatcher {
	return &AgentManagerDispatcher{Resolver: discovery.NewResolver(discovery.ResolverConfig{}), Client: &http.Client{Timeout: 20 * time.Second}}
}

func (d *AgentManagerDispatcher) Dispatch(ctx context.Context, in StartInput) (DispatchResult, error) {
	base, err := d.baseURL(ctx)
	if err != nil {
		return DispatchResult{}, err
	}
	task := &apipb.CreateTaskRequest{Task: &domainpb.Task{
		Title:       workflowTitle(in),
		Description: workflowPrompt(in),
		ScopePath:   workflowScope(in),
		ProjectRoot: workflowScope(in),
		CreatedBy:   "react-component-library",
	}}
	var taskResp apipb.CreateTaskResponse
	if err := d.post(ctx, base, "/api/v1/tasks", task, &taskResp); err != nil {
		return DispatchResult{}, err
	}
	if taskResp.Task == nil || taskResp.Task.Id == "" {
		return DispatchResult{}, fmt.Errorf("agent-manager task response missing id")
	}
	mode := domainpb.RunMode_RUN_MODE_IN_PLACE
	run := &apipb.CreateRunRequest{
		TaskId:         taskResp.Task.Id,
		RunMode:        &mode,
		IdempotencyKey: &in.IdempotencyKey,
		ProfileRef:     &apipb.ProfileRef{ProfileKey: ProfileKey},
		Tag:            ptr("rcl/" + string(in.Kind) + "/" + in.IdempotencyKey),
	}
	var runResp apipb.CreateRunResponse
	if err := d.post(ctx, base, "/api/v1/runs", run, &runResp); err != nil {
		return DispatchResult{}, err
	}
	if runResp.Run == nil || runResp.Run.Id == "" {
		return DispatchResult{}, fmt.Errorf("agent-manager run response missing id")
	}
	return DispatchResult{TaskID: taskResp.Task.Id, RunID: runResp.Run.Id, Status: statusFromAgent(runResp.Run.Status.String()), QueueDepth: int(runResp.QueueDepth)}, nil
}

func (d *AgentManagerDispatcher) Snapshot(ctx context.Context, runID string, after int64) (RunSnapshot, error) {
	base, err := d.baseURL(ctx)
	if err != nil {
		return RunSnapshot{}, err
	}
	var run apipb.GetRunResponse
	if err := d.get(ctx, base, "/api/v1/runs/"+runID, &run); err != nil {
		return RunSnapshot{}, err
	}
	if run.Run == nil {
		return RunSnapshot{}, fmt.Errorf("agent-manager run response missing run")
	}
	snap := RunSnapshot{Status: statusFromAgent(run.Run.Status.String()), Error: run.Run.ErrorMsg}
	if run.Run.Summary != nil {
		snap.Summary = run.Run.Summary.Description
	}
	var events apipb.GetRunEventsResponse
	if err := d.get(ctx, base, fmt.Sprintf("/api/v1/runs/%s/events?after_sequence=%d", runID, after), &events); err == nil {
		for _, event := range events.Events {
			if event.Sequence > snap.LastEventSequence {
				snap.LastEventSequence = event.Sequence
			}
		}
	}
	return snap, nil
}

func (d *AgentManagerDispatcher) Stop(ctx context.Context, runID string) (RunSnapshot, error) {
	base, err := d.baseURL(ctx)
	if err != nil {
		return RunSnapshot{}, err
	}
	var out apipb.StopRunResponse
	if err := d.post(ctx, base, "/api/v1/runs/"+runID+"/stop", &apipb.StopRunRequest{RunId: runID}, &out); err != nil {
		return RunSnapshot{}, err
	}
	snapshot := RunSnapshot{Status: StatusStopped}
	if out.Run != nil {
		snapshot.Status = statusFromAgent(out.Run.Status.String())
		snapshot.Error = out.Run.ErrorMsg
		if out.Run.Summary != nil {
			snapshot.Summary = out.Run.Summary.Description
		}
	}
	return snapshot, nil
}

func (d *AgentManagerDispatcher) baseURL(ctx context.Context) (string, error) {
	if d.Resolver == nil {
		d.Resolver = discovery.NewResolver(discovery.ResolverConfig{})
	}
	base, err := d.Resolver.ResolveScenarioURLDefault(ctx, "agent-manager")
	if err != nil {
		return "", fmt.Errorf("agent-manager unavailable: %w", err)
	}
	return strings.TrimRight(base, "/"), nil
}
func (d *AgentManagerDispatcher) post(ctx context.Context, base, path string, in, out proto.Message) error {
	return d.request(ctx, http.MethodPost, base, path, in, out)
}

func (d *AgentManagerDispatcher) get(ctx context.Context, base, path string, out proto.Message) error {
	return d.request(ctx, http.MethodGet, base, path, nil, out)
}

func (d *AgentManagerDispatcher) request(ctx context.Context, method, base, path string, in, out proto.Message) error {
	if d.Client == nil {
		d.Client = &http.Client{Timeout: 20 * time.Second}
	}
	var body io.Reader
	if in != nil {
		encoded, err := (protojson.MarshalOptions{UseProtoNames: true}).Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, base+path, body)
	if err != nil {
		return err
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	response, err := d.Client.Do(req)
	if err != nil {
		return fmt.Errorf("agent-manager request: %w", err)
	}
	defer response.Body.Close()
	raw, readErr := io.ReadAll(response.Body)
	if readErr != nil {
		return fmt.Errorf("read agent-manager response: %w", readErr)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("agent-manager status %d: %s", response.StatusCode, string(raw))
	}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(raw, out); err != nil {
		return fmt.Errorf("decode agent-manager response: %w", err)
	}
	return nil
}
func workflowTitle(in StartInput) string { return "RCL " + string(in.Kind) + " workflow" }

func workflowScope(in StartInput) string {
	if in.Kind == KindExtract {
		return "scenarios/" + in.SourceScenario
	}
	return "scenarios/" + in.TargetScenario
}

func workflowPrompt(in StartInput) string {
	if in.Kind == KindExtract {
		return fmt.Sprintf("Inspect only %s in scenario %s. Preserve behavior exactly. Use the React Component Library direct ingest API for the requested source; do not write files directly, do not acknowledge behavior loss, and report any parity failure for human review.", in.SourcePath, in.SourceScenario)
	}
	return fmt.Sprintf("Adopt asset %s version %s into scenario %s only through the React Component Library direct apply or reapply API. Preserve validation and overwrite controls; do not infer confirmation or claim success unless RCL returns an authoritative adoption result.", in.AssetID, in.RequestedVersion, in.TargetScenario)
}

func ptr(s string) *string { return &s }

func statusFromAgent(s string) Status {
	switch {
	case strings.Contains(s, "QUEUED"):
		return StatusQueued
	case strings.Contains(s, "RUNNING"):
		return StatusRunning
	case strings.Contains(s, "PARKED"):
		return StatusParked
	case strings.Contains(s, "SUCCEEDED"), strings.Contains(s, "COMPLETED"):
		return StatusSucceeded
	case strings.Contains(s, "STOPPED"), strings.Contains(s, "CANCELLED"):
		return StatusStopped
	default:
		return StatusFailed
	}
}
