package api

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	scenarioapp "github.com/vrooli/vrooli/internal/app/scenario"
	"github.com/vrooli/vrooli/internal/process"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	cliv1connect "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1/cliv1connect"
	"google.golang.org/protobuf/types/known/structpb"
)

// scenarioControlPlaneHandler is the typed control-plane adapter. It delegates
// to the same internal application service used by the root CLI, keeping
// lifecycle policy and host remediation in one place.
type scenarioControlPlaneHandler struct {
	cliv1connect.UnimplementedScenarioControlPlaneServiceHandler
	app *App
}

func (h *scenarioControlPlaneHandler) ListScenarios(_ context.Context, req *connect.Request[cliv1.ListScenariosRequest]) (*connect.Response[cliv1.ScenarioListResponse], error) {
	if h == nil || h.app == nil {
		return nil, controlPlaneInternalError("control-plane app is unavailable")
	}
	result, err := (scenarioapp.Service{Scenarios: h.app.Scenarios}).List(scenarioapp.ListRequest{IncludePorts: req.Msg.GetIncludePorts()})
	if err != nil {
		return nil, controlPlaneInternalError("list scenarios", err)
	}
	items := make([]*cliv1.Scenario, 0, len(result.Items))
	for _, item := range result.Items {
		ports := make([]*cliv1.ScenarioPort, 0, len(item.Ports))
		for _, port := range item.Ports {
			ports = append(ports, &cliv1.ScenarioPort{Key: port.Key, Step: port.Step, Port: int32(port.Port), ListenerStatus: port.ListenerStatus})
		}
		items = append(items, &cliv1.Scenario{
			Name: item.Name, Description: item.Description, Version: item.Version,
			Status: item.Status, Tags: item.Tags, Path: item.Path, Ports: ports,
		})
	}
	failures := make([]*cliv1.DiscoveryFailure, 0, len(result.Failures))
	for _, failure := range result.Failures {
		failures = append(failures, &cliv1.DiscoveryFailure{Kind: failure.Kind, Name: failure.Name, Path: failure.Path, Stage: failure.Stage, Error: failure.Error})
	}
	return connect.NewResponse(&cliv1.ScenarioListResponse{
		Success:   true,
		Summary:   &cliv1.ScenarioListSummary{TotalScenarios: int32(len(items)), Running: int32(result.RunningCount), Available: int32(len(items) - result.RunningCount)},
		Scenarios: items, DiscoveryFailures: failures,
	}), nil
}

func (h *scenarioControlPlaneHandler) GetScenarioStatus(_ context.Context, req *connect.Request[cliv1.GetScenarioStatusRequest]) (*connect.Response[cliv1.ScenarioStatusSingle], error) {
	if err := requireScenarioName(req.Msg.GetName()); err != nil {
		return nil, err
	}
	result, err := (scenarioapp.Service{Scenarios: h.app.Scenarios}).Status(scenarioapp.StatusRequest{Name: req.Msg.GetName()})
	if err != nil {
		return nil, controlPlaneInternalError("get scenario status", err)
	}
	if result.Single == nil {
		return nil, controlPlaneInternalError("get scenario status returned no single scenario")
	}
	return connect.NewResponse(statusSingleMessage(*result.Single)), nil
}

func (h *scenarioControlPlaneHandler) GetScenarioLogs(_ context.Context, req *connect.Request[cliv1.GetScenarioLogsRequest]) (*connect.Response[cliv1.ScenarioLogsResponse], error) {
	if err := requireScenarioName(req.Msg.GetName()); err != nil {
		return nil, err
	}
	if _, exists, err := h.app.Scenarios.Status(req.Msg.GetName()); err != nil || !exists {
		if err != nil {
			return nil, controlPlaneInternalError("get scenario logs", err)
		}
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("scenario %q not found", req.Msg.GetName()))
	}
	logPath, err := process.ScenarioLifecycleLogPath(h.app.Home, req.Msg.GetName())
	if err != nil {
		return nil, controlPlaneInternalError("resolve scenario log path", err)
	}
	tailLines := req.Msg.GetTailLines()
	if tailLines <= 0 {
		tailLines = 50
	}
	logs, err := h.app.readTail(logPath, strconv.FormatInt(int64(tailLines), 10))
	if err != nil {
		return nil, controlPlaneInternalError("read scenario logs", err)
	}
	return connect.NewResponse(&cliv1.ScenarioLogsResponse{Success: true, Scenario: req.Msg.GetName(), Logs: logs, TailLines: tailLines}), nil
}

func (h *scenarioControlPlaneHandler) StartScenario(_ context.Context, req *connect.Request[cliv1.StartScenarioRequest]) (*connect.Response[cliv1.ScenarioLifecycleResponse], error) {
	if err := requireScenarioName(req.Msg.GetName()); err != nil {
		return nil, err
	}
	items, err := (scenarioapp.Service{Scenarios: h.app.Scenarios}).Start(scenarioapp.StartRequest{Names: []string{req.Msg.GetName()}})
	if err != nil {
		return nil, controlPlaneInternalError("start scenario", err)
	}
	return connect.NewResponse(lifecycleResponse(items)), nil
}

func (h *scenarioControlPlaneHandler) StopScenario(_ context.Context, req *connect.Request[cliv1.StopScenarioRequest]) (*connect.Response[cliv1.ScenarioLifecycleResponse], error) {
	if err := requireScenarioName(req.Msg.GetName()); err != nil {
		return nil, err
	}
	runner, err := h.app.Services.LifecycleRunner()
	if err != nil {
		return nil, controlPlaneInternalError("create lifecycle runner", err)
	}
	items, err := (scenarioapp.Service{Scenarios: h.app.Scenarios, Runner: runner}).Stop(scenarioapp.StopRequest{Name: req.Msg.GetName()})
	if err != nil {
		return nil, controlPlaneInternalError("stop scenario", err)
	}
	return connect.NewResponse(lifecycleResponse(items)), nil
}

func (h *scenarioControlPlaneHandler) RestartScenario(_ context.Context, req *connect.Request[cliv1.RestartScenarioRequest]) (*connect.Response[cliv1.ScenarioLifecycleResponse], error) {
	if err := requireScenarioName(req.Msg.GetName()); err != nil {
		return nil, err
	}
	items, err := (scenarioapp.Service{Scenarios: h.app.Scenarios}).Restart(scenarioapp.RestartRequest{Name: req.Msg.GetName()})
	if err != nil {
		return nil, controlPlaneInternalError("restart scenario", err)
	}
	return connect.NewResponse(lifecycleResponse(items)), nil
}

func (h *scenarioControlPlaneHandler) SetupScenario(_ context.Context, req *connect.Request[cliv1.SetupScenarioRequest]) (*connect.Response[cliv1.ScenarioSetupResponse], error) {
	if err := requireScenarioName(req.Msg.GetName()); err != nil {
		return nil, err
	}
	runner, err := h.app.Services.LifecycleRunner()
	if err != nil {
		return nil, controlPlaneInternalError("create lifecycle runner", err)
	}
	result, err := (scenarioapp.Service{Runner: runner}).Setup(scenarioapp.SetupRequest{Name: req.Msg.GetName()})
	if err != nil {
		return nil, controlPlaneInternalError("run scenario setup", err)
	}
	return connect.NewResponse(&cliv1.ScenarioSetupResponse{
		Success: true, Phase: "setup", Status: string(result.Status), Defined: result.Defined,
		Steps: &cliv1.ScenarioSetupSteps{Executed: int32(result.ExecutedSteps), Skipped: int32(result.SkippedSteps)},
	}), nil
}

func requireScenarioName(name string) error {
	if strings.TrimSpace(name) == "" {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("scenario name is required"))
	}
	return nil
}

func controlPlaneInternalError(operation string, cause ...error) error {
	if len(cause) > 0 && cause[0] != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("%s: %w", operation, cause[0]))
	}
	return connect.NewError(connect.CodeInternal, fmt.Errorf("%s", operation))
}

func lifecycleResponse(items []scenarioapp.LifecycleItemOutput) *cliv1.ScenarioLifecycleResponse {
	response := &cliv1.ScenarioLifecycleResponse{Success: true, Scenarios: make([]*cliv1.ScenarioLifecycleItem, 0, len(items))}
	for _, item := range items {
		ports := make(map[string]int32, len(item.Ports))
		for key, port := range item.Ports {
			ports[key] = int32(port)
		}
		endpoints := make([]*cliv1.ScenarioEndpoint, 0, len(item.Endpoints))
		for _, endpoint := range item.Endpoints {
			endpoints = append(endpoints, &cliv1.ScenarioEndpoint{Name: endpoint.Name, Key: endpoint.Key, Description: endpoint.Description, Port: int32(endpoint.Port), Url: endpoint.URL})
		}
		response.Scenarios = append(response.Scenarios, &cliv1.ScenarioLifecycleItem{
			Name: item.Name, Status: item.Status, Health: item.Health, Ports: ports, Endpoints: endpoints,
			FailedDependencies: item.FailedDependencies, FailedResources: item.FailedResources, Verdict: item.Verdict,
		})
	}
	return response
}

func statusSingleMessage(output scenarioapp.StatusSingleOutput) *cliv1.ScenarioStatusSingle {
	item := output.Scenario
	startedAt := formatTimestamp(item.StartedAt)
	ports := make(map[string]int32, len(item.Ports))
	for key, port := range item.Ports {
		ports[key] = int32(port)
	}
	portBindings := make([]*cliv1.ScenarioPort, 0, len(item.PortBindings))
	for _, port := range item.PortBindings {
		portBindings = append(portBindings, &cliv1.ScenarioPort{
			Key: port.Key, Step: port.Step, Port: int32(port.Port), ListenerStatus: port.ListenerStatus,
		})
	}
	health := structpb.NewNullValue()
	if item.Health != nil {
		if value, err := structpb.NewValue(item.Health); err == nil {
			health = value
		}
	}
	processRecords := make([]*cliv1.ScenarioProcessRecord, 0, len(output.Runtime.ProcessInfo))
	for _, record := range output.Runtime.ProcessInfo {
		processRecords = append(processRecords, &cliv1.ScenarioProcessRecord{
			Pid: int32(record.PID), Pgid: int32(record.PGID), ProcessId: record.ProcessID,
			Phase: record.Phase, Scenario: record.Scenario, Step: record.Step,
			Command: record.Command, WorkingDir: record.WorkingDir, LogFile: record.LogFile,
			Port: int32(record.Port), StartedAt: formatTimestampValue(record.StartedAt), Status: record.Status,
		})
	}
	return &cliv1.ScenarioStatusSingle{
		Success: true,
		Scenario: &cliv1.ScenarioStatusItem{
			Name: item.Name, DisplayName: item.DisplayName, Description: item.Description, Tags: item.Tags,
			Status: item.Status, Processes: int32(item.Processes), Runtime: item.Runtime, StartedAt: startedAt,
			Ports: ports, PortBindings: portBindings, HealthStatus: health,
		},
		Info:    &cliv1.ScenarioInfoData{Name: output.Info.Name, DisplayName: output.Info.DisplayName, Description: output.Info.Description, Version: output.Info.Version, Type: output.Info.Type, Category: output.Info.Category, Tags: output.Info.Tags, Path: output.Info.Path, ServicePath: output.Info.ServicePath, SandboxRedirected: output.Info.SandboxRedirect, ConfigVersion: output.Info.ConfigVersion, LifecycleVersion: output.Info.LifecycleVersion},
		Runtime: &cliv1.ScenarioRuntimeData{Status: output.Runtime.Status, Processes: int32(output.Runtime.Processes), Runtime: output.Runtime.Runtime, StartedAt: formatTimestamp(output.Runtime.StartedAt), Ports: ports, ProcessRecords: processRecords, ListPorts: portBindings},
	}
}

func formatTimestamp(value *time.Time) string {
	if value == nil {
		return ""
	}
	return formatTimestampValue(*value)
}

func formatTimestampValue(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

var _ cliv1connect.ScenarioControlPlaneServiceHandler = (*scenarioControlPlaneHandler)(nil)
