package surfaces

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/targetmodel"
	instancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/compute-manager/v1/instance"
	instanceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/compute-manager/v1/instance/instance_v1connect"
)

// ComputeProvider exposes screenless compute nodes as capability panels. It
// deliberately never emits a desktop surface: a bridge machine id can be
// shown as a relationship, but a compute lease does not grant a GUI.
type ComputeProvider struct {
	ResolveURL func(context.Context, string) (string, error)
	Client     connect.HTTPClient
	Now        func() time.Time
}

func (p ComputeProvider) List(ctx context.Context) (ProviderResult, error) {
	resolve := p.ResolveURL
	if resolve == nil {
		resolve = discovery.ResolveScenarioURLDefault
	}
	client := p.Client
	if client == nil {
		client = &http.Client{Timeout: 2 * time.Second}
	}
	now := time.Now
	if p.Now != nil {
		now = p.Now
	}
	base, err := resolve(ctx, "compute-manager")
	if err != nil {
		return ProviderResult{}, err
	}
	response, err := instanceconnect.NewInstanceServiceClient(client, base, connect.WithReadMaxBytes(2<<20)).ListInstances(ctx, connect.NewRequest(&instancev1.ListInstancesRequest{}))
	if err != nil {
		return ProviderResult{}, err
	}
	if len(response.Msg.Instances) > 256 {
		return ProviderResult{}, fmt.Errorf("compute inventory exceeds bound")
	}
	observed := now().UTC()
	result := ProviderResult{Surfaces: make([]targetmodel.SurfaceDescriptor, 0, len(response.Msg.Instances))}
	for _, instance := range response.Msg.Instances {
		if instance == nil || strings.TrimSpace(instance.Id) == "" {
			return ProviderResult{}, fmt.Errorf("invalid compute instance")
		}
		label := strings.TrimSpace(instance.Provider)
		if label == "" {
			label = "Compute node"
		}
		if instance.Region != "" {
			label += " · " + instance.Region
		}
		if instance.Size != "" {
			label += " · " + instance.Size
		}
		state := targetmodel.CapabilityReady
		reason := "compute_instance_observed"
		if instance.State != instancev1.InstanceState_INSTANCE_STATE_RUNNING {
			state = targetmodel.CapabilityUnknown
			reason = "compute_instance_not_running"
		}
		result.Surfaces = append(result.Surfaces, targetmodel.SurfaceDescriptor{
			Ref:              targetmodel.SurfaceRef{Target: targetmodel.TargetRef{OwnerScenario: "compute-manager", ResourceID: instance.Id, HostNodeID: instance.BridgeMachineId}, OwnerScenario: "compute-manager", SurfaceID: instance.Id},
			Kind:             targetmodel.SurfaceDevicePanel,
			DisplayLabel:     label,
			ProtocolVersions: []string{"vrooli.compute.v1"},
			Capabilities: []targetmodel.CapabilityFact{
				{Capability: "compute.instance", State: state, ReasonCode: reason, EvidenceID: "compute-" + instance.Id, ObservedAt: observed, ExpiresAt: observed.Add(30 * time.Second)},
			},
		})
	}
	return result, nil
}
