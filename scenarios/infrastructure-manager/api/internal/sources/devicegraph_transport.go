package sources

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	devicegraphpb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/devicegraph"
	devicegraphconnect "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/devicegraph/devicegraphconnect"
)

// ConnectDeviceGraphTransport reads the device graph over the owner's typed
// Connect verb. It translates the wire vocabulary onto this package's
// transport-neutral shape and translates nothing else: the grades are the
// owner's and are carried through unchanged, because a transport that
// re-grades a reading is a second opinion pretending to be a measurement.
type ConnectDeviceGraphTransport struct {
	HTTP *http.Client
}

func (t ConnectDeviceGraphTransport) DeviceGraph(ctx context.Context, baseURL string) (DeviceGraph, error) {
	httpClient := t.HTTP
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	client := devicegraphconnect.NewDeviceGraphServiceClient(httpClient, baseURL)
	response, err := client.GetDeviceGraph(ctx, connect.NewRequest(&devicegraphpb.GetDeviceGraphRequest{}))
	if err != nil {
		return DeviceGraph{}, err
	}
	graph := response.Msg.GetGraph()
	if graph == nil {
		return DeviceGraph{}, fmt.Errorf("%s returned no graph", deviceGraphScenario)
	}
	// The owner reports its own failure to observe. Passing that through as a
	// graph would turn "the walk failed" into "this host has no hardware".
	if !graph.GetAvailable() {
		return DeviceGraph{}, fmt.Errorf("the device graph is unavailable: %s", graph.GetUnavailableReason())
	}

	out := DeviceGraph{
		CollectedAt:              graph.GetCollectedAt().AsTime(),
		Platform:                 graph.GetPlatform(),
		VirtualNetworkInterfaces: graph.GetVirtualNetworkInterfaces(),
		Warnings:                 graph.GetWarnings(),
		Devices:                  make([]GraphDevice, 0, len(graph.GetDevices())),
		Subsystems:               make([]GraphSubsystem, 0, len(graph.GetSubsystems())),
	}
	for _, device := range graph.GetDevices() {
		out.Devices = append(out.Devices, GraphDevice{
			ID:       device.GetId(),
			Class:    device.GetClass(),
			ParentID: device.GetParentId(),
			Vendor:   device.GetVendor(),
			Model:    device.GetModel(),
			Driver:   device.GetDriver(),
			SysPath:  device.GetSysPath(),
			// Attributes carry the kernel node names and per-class provenance
			// the owner recorded. They are passed through whole rather than
			// filtered to a known key list, because a filter here silently
			// drops every attribute the owner adds later.
			Attributes: device.GetAttributes(),
			Readings:   device.GetReadings(),
			Rungs:      rungStates(device.GetRungs()),
		})
	}
	for _, subsystem := range graph.GetSubsystems() {
		out.Subsystems = append(out.Subsystems, GraphSubsystem{
			Name:       subsystem.GetName(),
			Attributes: subsystem.GetAttributes(),
			Rungs:      rungStates(subsystem.GetRungs()),
		})
	}
	return out, nil
}

func rungStates(states []*devicegraphpb.RungState) map[string]RungState {
	out := make(map[string]RungState, len(states))
	for _, state := range states {
		rung := rungToken(state.GetRung())
		if rung == "" {
			continue
		}
		out[rung] = RungState{
			Rung:        rung,
			State:       gradeToken(state.GetGrade()),
			Reason:      state.GetReason(),
			Mechanism:   state.GetMechanism(),
			Remediation: state.GetRemediation(),
			ObservedAt:  state.GetObservedAt().AsTime(),
		}
	}
	return out
}

func rungToken(rung devicegraphpb.Rung) string {
	switch rung {
	case devicegraphpb.Rung_RUNG_IDENTITY:
		return "identity"
	case devicegraphpb.Rung_RUNG_TELEMETRY:
		return "telemetry"
	case devicegraphpb.Rung_RUNG_EVIDENCE:
		return "evidence"
	case devicegraphpb.Rung_RUNG_CONTROL:
		return "control"
	case devicegraphpb.Rung_RUNG_ANTICIPATION:
		return "anticipation"
	default:
		return ""
	}
}

// gradeToken maps the wire grade onto the state vocabulary. UNSPECIFIED is
// mapped to unmeasurable, never measured: a grade nobody set is not a reading
// this instrument may count as healthy.
func gradeToken(grade devicegraphpb.RungGrade) string {
	switch grade {
	case devicegraphpb.RungGrade_RUNG_GRADE_MEASURED:
		return "measured"
	case devicegraphpb.RungGrade_RUNG_GRADE_UNAVAILABLE:
		return "unavailable"
	case devicegraphpb.RungGrade_RUNG_GRADE_NOT_APPLICABLE:
		return "not_applicable"
	default:
		return "unmeasurable"
	}
}
