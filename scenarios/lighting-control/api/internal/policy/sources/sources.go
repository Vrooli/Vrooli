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
	agentdomainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	conditionconnect "github.com/vrooli/vrooli/packages/proto/gen/go/infrastructure-manager/v1/condition/condition_v1connect"
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
	resolve            scenarioResolver
	client             *http.Client
	now                func() time.Time
	newMetricsReader   func(*http.Client, string) metricsReader
	newRunsReader      func(*http.Client, string) runsReader
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
		newRunsReader: func(client *http.Client, base string) runsReader {
			return agentconnect.NewAgentManagerServiceClient(client, base)
		},
		newConditionReader: func(client *http.Client, base string) conditionReader {
			return conditionconnect.NewConditionServiceClient(client, base)
		},
	}
}

// Read fetches both sources concurrently and timestamps their receipt locally.
// Agent Manager currently has no source-side snapshot timestamp.
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
	client := s.newRunsReader(s.client, strings.TrimRight(base, "/"))
	seen := map[string]bool{}
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
			for _, run := range response.Msg.GetRuns() {
				if run.GetId() == "" || run.GetImportedAt() != nil || run.GetImportSourceSessionId() != "" || run.GetStatus() != status {
					continue
				}
				seen[run.GetId()] = true
			}
			if !response.Msg.GetHasMore() {
				break
			}
			if page == maxRunPages-1 {
				input.State = policyv1.EvidenceState_EVIDENCE_STATE_DEGRADED
				input.Reason = "active run list exceeded the bounded 5000-record page limit"
				input.Measurements["active_runs"] = float64(len(seen))
				return input
			}
		}
	}
	input.Measurements["active_runs"] = float64(len(seen))
	input.ObservedAt = s.now().UTC()
	input.State = policyv1.EvidenceState_EVIDENCE_STATE_FRESH
	input.Reason = "unique native pending, starting, and running records; historical imports excluded"
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
