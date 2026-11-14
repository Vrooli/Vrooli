package sources

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	agentapiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	agentconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	conditionconnect "github.com/vrooli/vrooli/packages/proto/gen/go/infrastructure-manager/v1/condition/condition_v1connect"
	policyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/policy"
	metricsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/metrics"
	metricsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/metrics/metricsconnect"
	"google.golang.org/protobuf/types/known/timestamppb"
	"lighting-control/internal/policy"
)

const (
	requestTimeout = 4 * time.Second
)

type scenarioResolver func(context.Context, string) (string, error)

type metricsReader interface {
	GetCurrentMetrics(context.Context, *connect.Request[metricsv1.GetCurrentMetricsRequest]) (*connect.Response[metricsv1.GetCurrentMetricsResponse], error)
	GetDetailedMetrics(context.Context, *connect.Request[metricsv1.GetDetailedMetricsRequest]) (*connect.Response[metricsv1.GetDetailedMetricsResponse], error)
}

type activityReader interface {
	GetRuntimeActivity(context.Context, *connect.Request[agentapiv1.GetRuntimeActivityRequest]) (*connect.Response[agentapiv1.GetRuntimeActivityResponse], error)
}

// Sources reads the owner APIs on demand. Callers choose a bounded cadence;
// this type does not start background polling or retain telemetry history.
type Sources struct {
	resolve            scenarioResolver
	client             *http.Client
	now                func() time.Time
	newMetricsReader   func(*http.Client, string) metricsReader
	newActivityReader  func(*http.Client, string) activityReader
	newConditionReader func(*http.Client, string) conditionReader
}

func New() *Sources {
	return &Sources{
		resolve: discovery.ResolveScenarioURLDefault,
		client:  &http.Client{Timeout: requestTimeout},
		now:     func() time.Time { return time.Now().UTC() },
		newMetricsReader: func(client *http.Client, base string) metricsReader {
			return metricsconnect.NewMetricsServiceClient(client, base)
		},
		newActivityReader: func(client *http.Client, base string) activityReader {
			return agentconnect.NewAgentManagerServiceClient(client, base, connect.WithReadMaxBytes(256<<10))
		},
		newConditionReader: func(client *http.Client, base string) conditionReader {
			return conditionconnect.NewConditionServiceClient(client, base)
		},
	}
}

// Read fetches owner evidence concurrently, preserving source timestamps separately from local receipt.
func (s *Sources) Read(ctx context.Context) policy.Inputs {
	ctx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	inputs := policy.Inputs{}
	var monitor, health policy.SourceInput
	var agents policy.SourceInput
	var infrastructure []policy.SourceInput
	monitorDone, agentsDone, infrastructureDone := make(chan struct{}), make(chan struct{}), make(chan struct{})
	go func() { monitor, health = s.readSystemMonitor(ctx); close(monitorDone) }()
	go func() { agents = s.readAgentManager(ctx); close(agentsDone) }()
	go func() { infrastructure = s.readHostHealth(ctx); close(infrastructureDone) }()
	<-monitorDone
	<-agentsDone
	<-infrastructureDone
	inputs.Now = s.now().UTC()
	inputs.SystemMonitor = monitor
	inputs.SystemHealth = health
	inputs.InfrastructureHealth = infrastructure
	inputs.AgentManager = agents
	return inputs
}

func (s *Sources) readSystemMonitor(ctx context.Context) (policy.SourceInput, policy.SourceInput) {
	load := policy.SourceInput{Name: "system-monitor", State: policyv1.EvidenceState_EVIDENCE_STATE_UNAVAILABLE, Measurements: map[string]float64{}}
	health := policy.SourceInput{Name: "system-monitor/health", State: policyv1.EvidenceState_EVIDENCE_STATE_UNAVAILABLE, Measurements: map[string]float64{}}
	base, err := s.resolve(ctx, "system-monitor")
	if err != nil {
		load.Reason = fmt.Sprintf("resolve system-monitor: %v", err)
		health.Reason = load.Reason
		return load, health
	}
	client := s.newMetricsReader(s.client, strings.TrimRight(base, "/"))
	current, err := client.GetCurrentMetrics(ctx, connect.NewRequest(&metricsv1.GetCurrentMetricsRequest{}))
	load.ObservedAt = s.now().UTC()
	if err != nil {
		load.Reason = fmt.Sprintf("read current metrics: %v", err)
	} else {
		snapshot := current.Msg.GetMetrics()
		load.SourceTime = timeFromProto(snapshot.GetTimestamp())
		if measured(snapshot.GetCpu()) {
			load.Measurements["cpu_usage_percent"] = snapshot.GetCpu().GetMeasured()
		}
		if measured(snapshot.GetGpu()) {
			load.Measurements["gpu_usage_percent"] = snapshot.GetGpu().GetMeasured()
		}
		load.State = policyv1.EvidenceState_EVIDENCE_STATE_FRESH
		if len(load.Measurements) == 0 {
			load.State = policyv1.EvidenceState_EVIDENCE_STATE_DEGRADED
			load.Reason = "system-monitor snapshot has no measured CPU or GPU usage"
		} else if load.SourceTime.IsZero() {
			load.State = policyv1.EvidenceState_EVIDENCE_STATE_DEGRADED
			load.Reason = "system-monitor snapshot has no source timestamp"
		}
	}
	// Pressure is independent evidence. A missing utilization reading must
	// not hide a measured pressure verdict, or lend it a newer timestamp.
	detailed, err := client.GetDetailedMetrics(ctx, connect.NewRequest(&metricsv1.GetDetailedMetricsRequest{}))
	health.ObservedAt = s.now().UTC()
	if err != nil {
		health.Reason = fmt.Sprintf("read health detail: %v", err)
		return load, health
	}
	pressure := detailed.Msg.GetMetrics().GetMemoryDetails().GetMemoryPressure()
	health.SourceTime = timeFromProto(pressure.GetObservedAt())
	health.State = policyv1.EvidenceState_EVIDENCE_STATE_DEGRADED
	if pressure == nil {
		health.Reason = "system-monitor returned no memory-pressure evidence"
		return load, health
	}
	if health.SourceTime.IsZero() {
		health.Reason = "memory-pressure evidence has no valid source timestamp"
		return load, health
	}
	levels := map[string]float64{"normal": 1, "elevated": 2, "critical": 3}
	level, known := levels[strings.ToLower(strings.TrimSpace(pressure.GetLevel()))]
	if !known {
		health.Reason = "memory-pressure verdict is unknown: " + pressure.GetLevelReason()
		return load, health
	}
	health.Measurements["memory_pressure_level"] = level
	health.State = policyv1.EvidenceState_EVIDENCE_STATE_FRESH
	health.Reason = pressure.GetLevelReason()
	return load, health
}

func (s *Sources) readAgentManager(ctx context.Context) policy.SourceInput {
	input := policy.SourceInput{Name: "agent-manager", State: policyv1.EvidenceState_EVIDENCE_STATE_UNAVAILABLE, ObservedAt: s.now().UTC(), Measurements: map[string]float64{}}
	base, err := s.resolve(ctx, "agent-manager")
	if err != nil {
		input.Reason = fmt.Sprintf("resolve agent-manager: %v", err)
		return input
	}
	client := s.newActivityReader(s.client, strings.TrimRight(base, "/"))
	response, err := client.GetRuntimeActivity(ctx, connect.NewRequest(&agentapiv1.GetRuntimeActivityRequest{}))
	input.ObservedAt = s.now().UTC()
	if err != nil {
		input.Reason = fmt.Sprintf("read physical runtime activity: %v", err)
		return input
	}
	if response == nil || response.Msg == nil {
		input.Reason = "physical runtime activity response unavailable"
		return input
	}
	snapshot := response.Msg
	input.SourceTime = timeFromProto(snapshot.GetObservedAt())
	input.State = policyv1.EvidenceState_EVIDENCE_STATE_DEGRADED
	input.Reason = snapshot.GetReason()
	input.Measurements["observed_executing_runs"] = float64(snapshot.GetObservedExecutingRuns())
	input.Measurements["logical_queued_runs"] = float64(snapshot.GetLogicalQueuedRuns())
	if snapshot.ExecutingRuns != nil && (!snapshot.GetComplete() || snapshot.GetExecutingRuns() != snapshot.GetObservedExecutingRuns()) {
		delete(input.Measurements, "observed_executing_runs")
		input.Reason = "physical activity response has contradictory count/coverage evidence"
		return input
	}
	if input.SourceTime.IsZero() {
		input.Reason = "physical activity has no valid source timestamp"
		return input
	}
	if snapshot.GetComplete() && snapshot.ExecutingRuns != nil && len(snapshot.GetUnknownReasons()) == 0 && snapshot.GetExecutingRuns() == snapshot.GetObservedExecutingRuns() {
		input.State = policyv1.EvidenceState_EVIDENCE_STATE_FRESH
		input.Measurements["active_runs"] = float64(snapshot.GetExecutingRuns())
	} else if input.Reason == "" {
		input.Reason = "physical execution coverage incomplete; zero observed is not zero activity"
	}

	return input
}

func measured(v *metricsv1.MetricValue) bool {
	if v == nil {
		return false
	}
	_, ok := v.GetState().(*metricsv1.MetricValue_Measured)
	return ok && !math.IsNaN(v.GetMeasured()) && !math.IsInf(v.GetMeasured(), 0) && v.GetMeasured() >= 0 && v.GetMeasured() <= 100
}

func timeFromProto(ts *timestamppb.Timestamp) time.Time {
	if ts == nil || ts.CheckValid() != nil {
		return time.Time{}
	}
	return ts.AsTime().UTC()
}
