package surfaces

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/targetmodel"
	attachedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/attached_devices"
	attachedconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/attached_devices/attached_devices_v1connect"
)

// BridgeAttachedProvider projects Bridge's durable physical-device topology
// into one surface per device. A device can be paired repeatedly as different
// transports become available; the provider keeps that identity joined and
// exposes each transport as a distinct capability.
type BridgeAttachedProvider struct {
	ResolveURL func(context.Context, string) (string, error)
	Client     connect.HTTPClient
	Now        func() time.Time
}

type contextHTTPClient struct{ client connect.HTTPClient }

func (c contextHTTPClient) Do(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	if token := bearerToken(req.Context()); token != "" {
		clone.Header.Set("Authorization", "Bearer "+token)
	}
	return c.client.Do(clone)
}

func (p BridgeAttachedProvider) List(ctx context.Context) (ProviderResult, error) {
	resolve := p.ResolveURL
	if resolve == nil {
		resolve = discovery.ResolveScenarioURLDefault
	}
	base, err := resolve(ctx, "vrooli-bridge")
	if err != nil {
		return ProviderResult{}, err
	}
	client := p.Client
	if client == nil {
		client = &http.Client{Timeout: 2 * time.Second}
	}
	response, err := attachedconnect.NewAttachedDeviceServiceClient(contextHTTPClient{client: client}, base, connect.WithReadMaxBytes(2<<20)).ListAttachedDevices(ctx, connect.NewRequest(&attachedv1.ListAttachedDevicesRequest{}))
	if err != nil {
		return ProviderResult{}, err
	}
	if response == nil || response.Msg == nil || len(response.Msg.Devices) > 256 {
		return ProviderResult{}, fmt.Errorf("attached-device inventory exceeds bound")
	}
	now := time.Now
	if p.Now != nil {
		now = p.Now
	}
	observed := now().UTC()
	out := make([]targetmodel.SurfaceDescriptor, 0, len(response.Msg.Devices))
	for _, device := range response.Msg.Devices {
		descriptor, err := attachedSurface(device, observed)
		if err != nil {
			return ProviderResult{}, err
		}
		out = append(out, descriptor)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Ref.SurfaceID < out[j].Ref.SurfaceID })
	return ProviderResult{Surfaces: out}, nil
}

func attachedSurface(device *attachedv1.AttachedDevice, observed time.Time) (targetmodel.SurfaceDescriptor, error) {
	if device == nil || strings.TrimSpace(device.Id) == "" || strings.TrimSpace(device.HostNodeId) == "" || strings.TrimSpace(device.Kind) == "" {
		return targetmodel.SurfaceDescriptor{}, fmt.Errorf("invalid attached-device descriptor")
	}
	transports := append([]string(nil), device.Transports...)
	if len(transports) == 0 && strings.TrimSpace(device.Transport) != "" {
		transports = []string{device.Transport}
	}
	transports = uniqueSorted(transports)
	label := strings.TrimSpace(device.Name)
	if label == "" {
		label = "Attached " + strings.TrimSpace(device.Kind)
	}
	if strings.ContainsAny(label, "\x00\r\n") || len(label) > 256 {
		return targetmodel.SurfaceDescriptor{}, fmt.Errorf("invalid attached-device label")
	}
	descriptor := targetmodel.SurfaceDescriptor{
		Ref: targetmodel.SurfaceRef{
			Target:        targetmodel.TargetRef{OwnerScenario: "vrooli-bridge", ResourceID: device.Id, HostNodeID: device.HostNodeId},
			OwnerScenario: "vrooli-bridge",
			SurfaceID:     device.Id,
		},
		Kind:             targetmodel.SurfaceDevicePanel,
		DisplayLabel:     label,
		ProtocolVersions: []string{"vrooli.device.v1"},
	}
	if device.TrustState == "trusted" {
		descriptor.Capabilities = append(descriptor.Capabilities, targetmodel.CapabilityFact{Capability: "device.trust", State: targetmodel.CapabilityReady, EvidenceID: "attached-" + device.Id, ObservedAt: observed, ExpiresAt: observed.Add(30 * time.Second)})
	} else {
		descriptor.Capabilities = append(descriptor.Capabilities, targetmodel.CapabilityFact{Capability: "device.trust", State: targetmodel.CapabilityDenied, ReasonCode: "trust_not_verified", ObservedAt: observed, ExpiresAt: observed.Add(30 * time.Second)})
	}
	for _, transport := range transports {
		capability := "device.transport." + transport
		state := targetmodel.CapabilityReady
		reason := ""
		evidence := "attached-" + device.Id + "-" + transport
		if device.Reachability != "reachable" {
			state, reason, evidence = targetmodel.CapabilityUnknown, "host_unreachable", ""
		}
		descriptor.Capabilities = append(descriptor.Capabilities, targetmodel.CapabilityFact{Capability: capability, State: state, ReasonCode: reason, EvidenceID: evidence, ObservedAt: observed, ExpiresAt: observed.Add(30 * time.Second)})
		descriptor.ProtocolVersions = append(descriptor.ProtocolVersions, "vrooli.device.transport."+transport+".v1")
	}
	if len(transports) == 0 {
		descriptor.Capabilities = append(descriptor.Capabilities, targetmodel.CapabilityFact{Capability: "device.transport", State: targetmodel.CapabilityUnknown, ReasonCode: "transport_unreported", ObservedAt: observed, ExpiresAt: observed.Add(30 * time.Second)})
	}
	if err := descriptor.Validate(); err != nil {
		return targetmodel.SurfaceDescriptor{}, fmt.Errorf("invalid attached-device projection: %w", err)
	}
	return descriptor, nil
}

func uniqueSorted(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
