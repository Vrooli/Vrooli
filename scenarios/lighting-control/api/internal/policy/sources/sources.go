package sources

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	agentapiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	agentconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	agentdomainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	policyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/policy"
	metricsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/metrics"
	metricsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/metrics/metricsconnect"
	"google.golang.org/protobuf/types/known/timestamppb"
	"lighting-control/internal/policy"
)

const (
	requestTimeout = 4 * time.Second
	runPageSize    = 500
	maxRunPages    = 10
)

type scenarioResolver func(context.Context, string) (string, error)

type metricsReader interface {
	GetCurrentMetrics(context.Context, *connect.Request[metricsv1.GetCurrentMetricsRequest]) (*connect.Response[metricsv1.GetCurrentMetricsResponse], error)
	GetDetailedMetrics(context.Context, *connect.Request[metricsv1.GetDetailedMetricsRequest]) (*connect.Response[metricsv1.GetDetailedMetricsResponse], error)
}

type runsReader interface {
	ListRuns(context.Context, *connect.Request[agentapiv1.ListRunsRequest]) (*connect.Response[agentapiv1.ListRunsResponse], error)
}

// Sources reads the owner APIs on demand. Callers choose a bounded cadence;
// this type does not start background polling or retain telemetry history.
type Sources struct {
	resolve          scenarioResolver
	client           *http.Client
	now              func() time.Time
	newMetricsReader func(*http.Client, string) metricsReader
	newRunsReader    func(*http.Client, string) runsReader
}

func New() *Sources {
	return &Sources{
		resolve: discovery.ResolveScenarioURLDefault,
		client:  &http.Client{Timeout: requestTimeout},
		now:     func() time.Time { return time.Now().UTC() },
		newMetricsReader: func(client *http.Client, base string) metricsReader {
			return metricsconnect.NewMetricsServiceClient(client, base)
		},
		newRunsReader: func(client *http.Client, base string) runsReader {
			return agentconnect.NewAgentManagerServiceClient(client, base)
		},
	}
}

// Read fetches both sources concurrently and timestamps their receipt locally.
// Agent Manager currently has no source-side snapshot timestamp.
func (s *Sources) Read(ctx context.Context) policy.Inputs {
	now := s.now().UTC()
	inputs := policy.Inputs{Now: now}
	var monitor policy.SourceInput
	var agents policy.SourceInput
	monitorDone, agentsDone := make(chan struct{}), make(chan struct{})
	go func() { monitor = s.readSystemMonitor(ctx); close(monitorDone) }()
	go func() { agents = s.readAgentManager(ctx); close(agentsDone) }()
	<-monitorDone
	<-agentsDone
	inputs.SystemMonitor = monitor
	inputs.AgentManager = agents
	return inputs
}

func (s *Sources) readSystemMonitor(ctx context.Context) policy.SourceInput {
	input := policy.SourceInput{Name: "system-monitor", State: policyv1.EvidenceState_EVIDENCE_STATE_UNAVAILABLE, ObservedAt: s.now().UTC(), Measurements: map[string]float64{}}
	base, err := s.resolve(ctx, "system-monitor")
	if err != nil {
		input.Reason = fmt.Sprintf("resolve system-monitor: %v", err)
		return input
	}
	base = strings.TrimRight(base, "/")
	client := s.newMetricsReader(s.client, base)
	current, err := client.GetCurrentMetrics(ctx, connect.NewRequest(&metricsv1.GetCurrentMetricsRequest{}))
	if err != nil {
		input.Reason = fmt.Sprintf("read current metrics: %v", err)
		return input
	}
	if current.Msg.GetMetrics() == nil {
		input.Reason = "system-monitor returned no current metric snapshot"
		return input
	}
	snapshot := current.Msg.GetMetrics()
	input.SourceTime = timeFromProto(snapshot.GetTimestamp())
	if measured(snapshot.GetCpu()) {
		input.Measurements["cpu_usage_percent"] = snapshot.GetCpu().GetMeasured()
	}
	if measured(snapshot.GetGpu()) {
		input.Measurements["gpu_usage_percent"] = snapshot.GetGpu().GetMeasured()
	}
	if len(input.Measurements) == 0 {
		input.State = policyv1.EvidenceState_EVIDENCE_STATE_DEGRADED
		input.Reason = "system-monitor snapshot has no measured CPU or GPU usage"
		return input
	}
	if input.SourceTime.IsZero() {
		input.State = policyv1.EvidenceState_EVIDENCE_STATE_DEGRADED
		input.Reason = "system-monitor snapshot has no source timestamp"
		return input
	}

	detailed, err := client.GetDetailedMetrics(ctx, connect.NewRequest(&metricsv1.GetDetailedMetricsRequest{}))
	if err != nil {
		input.State = policyv1.EvidenceState_EVIDENCE_STATE_DEGRADED
		input.Reason = fmt.Sprintf("read detailed memory pressure: %v", err)
		return input
	}
	pressure := detailed.Msg.GetMetrics().GetMemoryDetails().GetMemoryPressure()
	if pressure == nil {
		input.State = policyv1.EvidenceState_EVIDENCE_STATE_DEGRADED
		input.Reason = "system-monitor returned no memory-pressure evidence"
		return input
	}
	level := uint32(0) // unknown; never interpreted as healthy by policy.
	switch strings.ToLower(strings.TrimSpace(pressure.GetLevel())) {
	case "normal":
		level = 1
	case "elevated":
		level = 2
	case "critical":
		level = 3
	}
	input.Measurements["memory_pressure_level"] = float64(level)
	if pressure.GetObservedAt() != nil {
		pressureTime := timeFromProto(pressure.GetObservedAt())
		if !pressureTime.IsZero() && (input.SourceTime.IsZero() || pressureTime.Before(input.SourceTime)) {
			input.SourceTime = pressureTime
		}
	}
	if input.SourceTime.IsZero() {
		input.State = policyv1.EvidenceState_EVIDENCE_STATE_DEGRADED
		input.Reason = "system-monitor memory-pressure evidence has no source timestamp"
		return input
	}
	input.State = policyv1.EvidenceState_EVIDENCE_STATE_FRESH
	if level == 0 {
		input.Reason = "CPU/GPU are measured; memory-pressure verdict is unknown"
	}
	return input
}

func (s *Sources) readAgentManager(ctx context.Context) policy.SourceInput {
	input := policy.SourceInput{Name: "agent-manager", State: policyv1.EvidenceState_EVIDENCE_STATE_UNAVAILABLE, ObservedAt: s.now().UTC(), Measurements: map[string]float64{}}
	base, err := s.resolve(ctx, "agent-manager")
	if err != nil {
		input.Reason = fmt.Sprintf("resolve agent-manager: %v", err)
		return input
	}
	client := s.newRunsReader(s.client, strings.TrimRight(base, "/"))
	count := 0
	for _, status := range []agentdomainv1.RunStatus{
		agentdomainv1.RunStatus_RUN_STATUS_PENDING,
		agentdomainv1.RunStatus_RUN_STATUS_STARTING,
		agentdomainv1.RunStatus_RUN_STATUS_RUNNING,
	} {
		for page := 0; page < maxRunPages; page++ {
			limit := int32(runPageSize)
			offset := int32(page * runPageSize)
			response, err := client.ListRuns(ctx, connect.NewRequest(&agentapiv1.ListRunsRequest{Status: &status, Limit: &limit, Offset: &offset}))
			if err != nil {
				input.Reason = fmt.Sprintf("list %s runs: %v", status.String(), err)
				return input
			}
			count += len(response.Msg.GetRuns())
			if !response.Msg.GetHasMore() {
				break
			}
			if page == maxRunPages-1 {
				input.State = policyv1.EvidenceState_EVIDENCE_STATE_DEGRADED
				input.Reason = "active run list exceeded the bounded 5000-record page limit"
				input.Measurements["active_runs"] = float64(count)
				return input
			}
		}
	}
	input.Measurements["active_runs"] = float64(count)
	input.State = policyv1.EvidenceState_EVIDENCE_STATE_FRESH
	input.Reason = "active runs counted from pending, starting, and running records"
	return input
}

func measured(v *metricsv1.MetricValue) bool {
	if v == nil {
		return false
	}
	_, ok := v.GetState().(*metricsv1.MetricValue_Measured)
	return ok
}

func timeFromProto(ts *timestamppb.Timestamp) time.Time {
	if ts == nil {
		return time.Time{}
	}
	return ts.AsTime().UTC()
}
