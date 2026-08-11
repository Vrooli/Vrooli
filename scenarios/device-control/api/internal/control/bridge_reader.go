package control

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	attachedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/attached_devices"
	attachedconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/attached_devices/attached_devices_v1connect"
)

type bridgeAttachedReader struct {
	client attachedconnect.AttachedDeviceServiceClient
}

func NewBridgeAttachedReader(httpClient *http.Client, resolveURL func(context.Context, string) (string, error)) AttachedReader {
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	if resolveURL == nil {
		resolveURL = discovery.ResolveScenarioURLDefault
	}
	return &lazyBridgeAttachedReader{httpClient: httpClient, resolveURL: resolveURL}
}

type lazyBridgeAttachedReader struct {
	httpClient *http.Client
	resolveURL func(context.Context, string) (string, error)
}

func (r *lazyBridgeAttachedReader) List(ctx context.Context) ([]AttachedDevice, error) {
	base, err := r.resolveURL(ctx, "vrooli-bridge")
	if err != nil {
		return nil, err
	}
	client := attachedconnect.NewAttachedDeviceServiceClient(r.httpClient, strings.TrimRight(base, "/"))
	resp, err := client.ListAttachedDevices(ctx, connect.NewRequest(&attachedv1.ListAttachedDevicesRequest{}))
	if err != nil {
		return nil, fmt.Errorf("list bridge attached devices: %w", err)
	}
	out := make([]AttachedDevice, 0, len(resp.Msg.Devices))
	for _, d := range resp.Msg.Devices {
		out = append(out, AttachedDevice{ID: d.Id, Name: d.Name, HostNodeID: d.HostNodeId, Kind: d.Kind, Transport: d.Transport, Serial: d.Serial, OSVersion: d.OsVersion, TrustState: d.TrustState, Reachability: d.Reachability, HealthReason: d.HealthReason})
	}
	return out, nil
}
