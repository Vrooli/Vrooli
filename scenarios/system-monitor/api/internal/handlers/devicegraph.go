package handlers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/nodeclient"
	devicegraphpb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/devicegraph"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/apierrors"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/collectors"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/devicegraph"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/httputil"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// errNoDeviceGraphProvider is returned when the handler is mounted without the
// shared provider. It is a precondition failure rather than an empty graph,
// because an empty graph is a claim that the host has no hardware.
var errNoDeviceGraphProvider = errors.New("the device-graph provider is not configured on this API")

// DeviceGraphHandler serves the graded hardware device graph over Connect.
//
// The graph was previously reachable only as a metric payload, which meant a
// consumer in another scenario had to parse a blob to learn whether a disk's
// SMART attributes were readable. This handler is the typed read verb, and it
// serves from the SAME cached provider the 60s collector uses — a second
// provider would mean a second host walk per reader and two answers about one
// machine.
type DeviceGraphHandler struct {
	graphs collectors.DeviceGraphProvider
	log    *slog.Logger
	bridge *nodeclient.Client
}

func NewDeviceGraphHandler(graphs collectors.DeviceGraphProvider, log *slog.Logger) *DeviceGraphHandler {
	if log == nil {
		log = slog.Default()
	}
	return &DeviceGraphHandler{
		graphs: graphs,
		log:    log,
		bridge: nodeclient.New(nodeclient.Config{
			Token:         firstNonEmpty(os.Getenv("VROOLI_BRIDGE_API_TOKEN"), os.Getenv("VROOLI_API_TOKEN")),
			TokenProvider: resolveLocalOwnerToken,
		}),
	}
}

// SetNodeClient replaces the Bridge transport for focused handler tests and
// controlled embeddings. Production uses the shared nodeclient above.
func (h *DeviceGraphHandler) SetNodeClient(client *nodeclient.Client) { h.bridge = client }

// HandleGetDeviceGraph serves the dashboard's lower-camel REST shape. Local
// reads use the same cached provider as Connect; a node query relays the
// scenario-owned device-graph verb through the shared node client.
func (h *DeviceGraphHandler) HandleGetDeviceGraph(w http.ResponseWriter, r *http.Request) {
	if nodeID := strings.TrimSpace(r.URL.Query().Get("node")); nodeID != "" {
		h.handleRemoteDeviceGraph(w, r, nodeID)
		return
	}
	if h.graphs == nil {
		handleDeviceGraphError(w, r, h.log, apierrors.Unavailable("local device graph"))
		return
	}
	handleDeviceGraphJSON(w, h.log, r, protoGraph(h.graphs.DeviceGraph(r.Context())))
}

func (h *DeviceGraphHandler) handleRemoteDeviceGraph(w http.ResponseWriter, r *http.Request, nodeID string) {
	if h.bridge == nil {
		handleDeviceGraphError(w, r, h.log, apierrors.Unavailable("remote node"))
		return
	}
	response, err := h.bridge.Call(r.Context(), nodeclient.CallRequest{
		NodeID: nodeID, Scenario: "system-monitor", Command: "system-monitor metrics devices",
		Args: []string{"--json"}, Timeout: 8 * time.Second, MaxResponse: 2 << 20,
	})
	if err != nil || response.Outcome != 1 {
		handleDeviceGraphError(w, r, h.log, apierrors.Unavailable("remote node"))
		return
	}
	var envelope devicegraphpb.GetDeviceGraphResponse
	if err := protojson.Unmarshal(response.Data, &envelope); err == nil && envelope.GetGraph() != nil {
		handleDeviceGraphJSON(w, h.log, r, envelope.GetGraph())
		return
	}
	var graph devicegraphpb.Graph
	if err := protojson.Unmarshal(response.Data, &graph); err != nil {
		handleDeviceGraphError(w, r, h.log, apierrors.Internal("decode remote device graph", fmt.Errorf("%w", err)))
		return
	}
	handleDeviceGraphJSON(w, h.log, r, &graph)
}

func handleDeviceGraphJSON(w http.ResponseWriter, log *slog.Logger, r *http.Request, graph *devicegraphpb.Graph) {
	if err := httputil.ProtoJSONCamel(w, graph); err != nil {
		handleDeviceGraphError(w, r, log, apierrors.Internal("encode device graph", err))
	}
}

func handleDeviceGraphError(w http.ResponseWriter, r *http.Request, log *slog.Logger, err *apierrors.APIError) {
	httputil.WriteAPIError(w, log, r, err)
}

func (h *DeviceGraphHandler) GetDeviceGraph(ctx context.Context, _ *connect.Request[devicegraphpb.GetDeviceGraphRequest]) (*connect.Response[devicegraphpb.GetDeviceGraphResponse], error) {
	if h.graphs == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			errNoDeviceGraphProvider)
	}
	graph := h.graphs.DeviceGraph(ctx)
	return connect.NewResponse(&devicegraphpb.GetDeviceGraphResponse{Graph: protoGraph(graph)}), nil
}

// protoGraph projects the graph onto the wire. Availability is decided here the
// same way the collector decides it: a walk that produced neither a device nor
// a graded subsystem, or one that failed its structural invariants, is a
// failure to observe and is reported as such. Serving it as an empty graph
// would be a claim that the host has no hardware.
func protoGraph(graph devicegraph.Graph) *devicegraphpb.Graph {
	out := &devicegraphpb.Graph{
		CollectedAt:              timestamppb.New(graph.CollectedAt),
		Platform:                 graph.Platform,
		VirtualNetworkInterfaces: append([]string(nil), graph.VirtualNetworkInterfaces...),
		Warnings:                 append([]string(nil), graph.Warnings...),
		Available:                true,
		Devices:                  make([]*devicegraphpb.Device, 0, len(graph.Devices)),
		Subsystems:               make([]*devicegraphpb.Subsystem, 0, len(graph.Subsystems)),
	}
	if len(graph.Devices) == 0 && len(graph.Subsystems) == 0 {
		out.Available = false
		out.UnavailableReason = "the device-graph walk produced neither a device nor a graded subsystem"
	}
	if err := (&graph).Validate(); err != nil {
		out.Available = false
		out.UnavailableReason = "device graph failed its structural invariants: " + err.Error()
	}
	for _, device := range graph.Devices {
		out.Devices = append(out.Devices, &devicegraphpb.Device{
			Id:         device.ID,
			Class:      string(device.Class),
			ParentId:   device.ParentID,
			Vendor:     device.Vendor,
			Model:      device.Model,
			Driver:     device.Driver,
			SysPath:    device.SysPath,
			Attributes: copyAttributes(device.Attributes),
			Readings:   copyReadings(device.Readings),
			Rungs:      protoRungs(device.Rungs),
		})
	}
	for _, subsystem := range graph.Subsystems {
		out.Subsystems = append(out.Subsystems, &devicegraphpb.Subsystem{
			Name:       subsystem.Name,
			Attributes: copyAttributes(subsystem.Attributes),
			Rungs:      protoRungs(subsystem.Rungs),
		})
	}
	return out
}

// protoRungs emits the ladder in dependency order rather than in map order, so
// two reads of one unchanged host produce byte-identical responses. A rung the
// grader never set is omitted rather than emitted as UNSPECIFIED, because an
// absent grade and a grade of "unspecified" are different claims and only the
// first is true here.
func protoRungs(rungs map[devicegraph.Rung]devicegraph.RungState) []*devicegraphpb.RungState {
	out := make([]*devicegraphpb.RungState, 0, len(rungs))
	for _, rung := range devicegraph.Rungs {
		state, ok := rungs[rung]
		if !ok {
			continue
		}
		out = append(out, &devicegraphpb.RungState{
			Rung:        protoRung(rung),
			Grade:       protoGrade(state.State),
			Reason:      state.Reason,
			Mechanism:   state.Mechanism,
			Remediation: state.Remediation,
			ObservedAt:  timestamppb.New(state.ObservedAt),
		})
	}
	// A rung outside the known ladder cannot be silently dropped: it would be a
	// grade the owner produced and this projection refused to report.
	extra := make([]devicegraph.Rung, 0)
	for rung := range rungs {
		if protoRung(rung) == devicegraphpb.Rung_RUNG_UNSPECIFIED {
			extra = append(extra, rung)
		}
	}
	sort.Slice(extra, func(i, j int) bool { return extra[i] < extra[j] })
	for _, rung := range extra {
		state := rungs[rung]
		out = append(out, &devicegraphpb.RungState{
			Rung:        devicegraphpb.Rung_RUNG_UNSPECIFIED,
			Grade:       protoGrade(state.State),
			Reason:      "rung " + string(rung) + " is outside the published ladder vocabulary: " + state.Reason,
			Mechanism:   state.Mechanism,
			Remediation: state.Remediation,
			ObservedAt:  timestamppb.New(state.ObservedAt),
		})
	}
	return out
}

func protoRung(rung devicegraph.Rung) devicegraphpb.Rung {
	switch rung {
	case devicegraph.RungIdentity:
		return devicegraphpb.Rung_RUNG_IDENTITY
	case devicegraph.RungTelemetry:
		return devicegraphpb.Rung_RUNG_TELEMETRY
	case devicegraph.RungEvidence:
		return devicegraphpb.Rung_RUNG_EVIDENCE
	case devicegraph.RungControl:
		return devicegraphpb.Rung_RUNG_CONTROL
	case devicegraph.RungAnticipation:
		return devicegraphpb.Rung_RUNG_ANTICIPATION
	default:
		return devicegraphpb.Rung_RUNG_UNSPECIFIED
	}
}

// protoGrade maps the grader's state onto the wire. An unrecognised state
// becomes UNMEASURABLE, never MEASURED: a state this projection does not
// understand is not a reading a consumer may count as healthy.
func protoGrade(state devicegraph.State) devicegraphpb.RungGrade {
	switch state {
	case devicegraph.StateMeasured:
		return devicegraphpb.RungGrade_RUNG_GRADE_MEASURED
	case devicegraph.StateUnmeasurable:
		return devicegraphpb.RungGrade_RUNG_GRADE_UNMEASURABLE
	case devicegraph.StateUnavailable:
		return devicegraphpb.RungGrade_RUNG_GRADE_UNAVAILABLE
	case devicegraph.StateNotApplicable:
		return devicegraphpb.RungGrade_RUNG_GRADE_NOT_APPLICABLE
	default:
		return devicegraphpb.RungGrade_RUNG_GRADE_UNMEASURABLE
	}
}

func copyAttributes(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func copyReadings(in map[string]float64) map[string]float64 {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]float64, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
