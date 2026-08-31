package hub

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/vrooli/nodeclient"
	runsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/runs"
)

type bridgeRemote struct {
	client *nodeclient.Client
}

func NewBridgeRemoteFromEnvironment() RemoteDelivery {
	client := nodeclient.New(nodeclient.Config{
		Token: firstNonEmpty(os.Getenv("VROOLI_BRIDGE_API_TOKEN"), os.Getenv("VROOLI_API_TOKEN")),
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if nodes, err := client.List(ctx, 5*time.Second); err != nil {
		slog.Warn("vrooli-bridge unavailable; remote notification delivery will retry", "error", err)
	} else {
		slog.Info("vrooli-bridge reachable for remote notification delivery", "nodes", len(nodes))
	}
	return &bridgeRemote{client: client}
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
	dispatched, err := b.client.Dispatch(ctx, nodeclient.DispatchRequest{
		NodeID: nodeID, Verb: "notification-hub notifications relay", Args: []string{"--payload-base64", encoded}, Timeout: 120 * time.Second,
	})
	if err != nil {
		return "", fmt.Errorf("bridge dispatch: %w", err)
	}
	if strings.TrimSpace(dispatched.RunID) == "" {
		return "", fmt.Errorf("bridge returned no durable run id")
	}
	run, timedOut, err := b.client.Wait(ctx, dispatched.RunID, 120*time.Second)
	if err != nil {
		return "", fmt.Errorf("wait bridge run: %w", err)
	}
	if timedOut || run == nil {
		return "", fmt.Errorf("bridge returned no completed run")
	}
	if run.GetStatus() != runsv1.RunStatus_RUN_STATUS_PASSED {
		return "", fmt.Errorf("remote relay run %s ended with status %s", dispatched.RunID, run.GetStatus().String())
	}
	return dispatched.RunID, nil
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
	machines, err := b.client.ListMachines(ctx, 10*time.Second)
	if err != nil {
		return "", fmt.Errorf("list bridge machines: %w", err)
	}
	if machines == nil {
		return "", fmt.Errorf("bridge returned no machine inventory")
	}
	for _, machine := range machines {
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

var _ RemoteDelivery = (*bridgeRemote)(nil)

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
