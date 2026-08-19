package hub

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"connectrpc.com/connect"
	dispatchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/dispatch"
	dispatchconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/dispatch/dispatch_v1connect"
	machinesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/machines"
	machinesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/machines/machines_v1connect"
	runsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/runs"
	runsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/runs/runs_v1connect"
)

type bridgeRemote struct {
	machines   machinesconnect.MachineServiceClient
	dispatcher dispatchconnect.DispatchServiceClient
	runs       runsconnect.RunsServiceClient
}

func NewBridgeRemoteFromEnvironment() RemoteDelivery {
	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("VROOLI_BRIDGE_URL")), "/")
	if baseURL == "" {
		return nil
	}
	httpClient := &http.Client{Timeout: 3 * time.Minute}
	if token := strings.TrimSpace(os.Getenv("VROOLI_BRIDGE_API_TOKEN")); token != "" {
		httpClient.Transport = bearerTransport{base: http.DefaultTransport, token: token}
	}
	return &bridgeRemote{
		machines:   machinesconnect.NewMachineServiceClient(httpClient, baseURL),
		dispatcher: dispatchconnect.NewDispatchServiceClient(httpClient, baseURL),
		runs:       runsconnect.NewRunsServiceClient(httpClient, baseURL),
	}
}

func (b *bridgeRemote) Deliver(ctx context.Context, machineID string, n Notification, body string) (string, error) {
	nodeID, err := b.currentNode(ctx, machineID)
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(struct {
		Title            string `json:"title"`
		Body             string `json:"body"`
		Urgency          string `json:"urgency"`
		SensitivityLabel string `json:"sensitivity_label"`
		IdempotencyKey   string `json:"idempotency_key"`
		DedupeKey        string `json:"dedupe_key"`
		DedupeWindowSecs int64  `json:"dedupe_window_seconds"`
	}{n.Title, body, n.Urgency, n.SensitivityLabel, n.IdempotencyKey, n.DedupeKey, 0})
	if err != nil {
		return "", err
	}
	encoded := base64.RawStdEncoding.EncodeToString(payload)
	response, err := b.dispatcher.DispatchJob(ctx, connect.NewRequest(&dispatchv1.DispatchJobRequest{
		NodeId:         nodeID,
		Verb:           "notification-hub notifications relay",
		Args:           []string{"--payload-base64", encoded},
		TimeoutSeconds: 120,
	}))
	if err != nil {
		return "", fmt.Errorf("bridge dispatch: %w", err)
	}
	if response == nil || response.Msg == nil || response.Msg.GetRunId() == "" {
		return "", fmt.Errorf("bridge returned no durable run id")
	}
	waited, err := b.runs.WaitRun(ctx, connect.NewRequest(&runsv1.WaitRunRequest{Id: response.Msg.GetRunId(), TimeoutSeconds: 120}))
	if err != nil {
		return "", fmt.Errorf("wait bridge run: %w", err)
	}
	if waited == nil || waited.Msg == nil || waited.Msg.GetRun() == nil {
		return "", fmt.Errorf("bridge returned no completed run")
	}
	if waited.Msg.GetRun().GetStatus() != runsv1.RunStatus_RUN_STATUS_PASSED {
		return "", fmt.Errorf("remote relay run %s ended with status %s", response.Msg.GetRunId(), waited.Msg.GetRun().GetStatus().String())
	}
	return response.Msg.GetRunId(), nil
}

func (b *bridgeRemote) ChannelsStatus(ctx context.Context, machineID, _ string) (ChannelStatus, error) {
	_, err := b.currentNode(ctx, machineID)
	status := ChannelStatus{Channel: "remote", MachineID: machineID, Disposition: "ready", Reason: "current machine lineage has a dispatch node"}
	if err != nil {
		status.Disposition = "not_configured"
		status.Reason = err.Error()
	}
	return status, nil
}

func (b *bridgeRemote) currentNode(ctx context.Context, machineID string) (string, error) {
	response, err := b.machines.ListMachines(ctx, connect.NewRequest(&machinesv1.ListMachinesRequest{}))
	if err != nil {
		return "", fmt.Errorf("list bridge machines: %w", err)
	}
	if response == nil || response.Msg == nil {
		return "", fmt.Errorf("bridge returned no machine inventory")
	}
	for _, machine := range response.Msg.GetMachines() {
		if machine.GetId() != machineID {
			continue
		}
		for _, lineage := range machine.GetNodeLineage() {
			if lineage.GetCurrent() && lineage.GetNodeId() != "" {
				return lineage.GetNodeId(), nil
			}
		}
		return "", fmt.Errorf("machine %q has no current node lineage", machineID)
	}
	return "", fmt.Errorf("machine %q is not registered with bridge", machineID)
}

type bearerTransport struct {
	base  http.RoundTripper
	token string
}

func (t bearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.Header.Set("Authorization", "Bearer "+t.token)
	return t.base.RoundTrip(clone)
}

var _ RemoteDelivery = (*bridgeRemote)(nil)
