package scenarioapp

import (
	"time"

	"github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/orchestrator"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/resources"
	scenariomodel "github.com/vrooli/vrooli/internal/scenario"
)

type StartRequest struct {
	Names     []string
	Options   lifecycle.StartOptions
	JSON      bool
	OpenAfter bool
}

type StopRequest struct {
	Name string
	JSON bool
}

type RestartRequest struct {
	Name      string
	Options   lifecycle.StartOptions
	JSON      bool
	OpenAfter bool
}

type ListRequest struct {
	JSON         bool
	IncludePorts bool
}

type InfoRequest struct {
	Name string
	JSON bool
}

type StatusRequest struct {
	Name string
	JSON bool
}

type ValidateEnvRequest struct {
	Name string
	JSON bool
}

type SetupRequest struct {
	Name string
	Opts lifecycle.PhaseOptions
	JSON bool
}

type TestRequest struct {
	Name string
	Opts lifecycle.PhaseOptions
}

type (
	StartAllRequest struct{ JSON bool }
	StopAllRequest  struct{ JSON bool }
)

type PortRequest struct {
	ScenarioName string
	PortName     string
	JSON         bool
}

type OpenRequest struct {
	ScenarioName string
	PortName     string
	PrintURL     bool
	JSON         bool
}

type RequirementsRequest struct {
	Snapshot bool
	Args     []string
}

type HealFromSandboxRequest struct {
	MergedPath string
	DryRun     bool
}

type HealFromSandboxResponse struct {
	Affected     []string
	DryRun       bool
	StoppedCount int
}

type ValidateEnvResponse struct {
	Report resources.ScenarioEnvValidationReport
}

type ListPortOutput struct {
	Key  string `json:"key"`
	Step string `json:"step,omitempty"`
	Port int    `json:"port"`
}

type ListItemOutput struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Version     string           `json:"version,omitempty"`
	Status      string           `json:"status"`
	Tags        []string         `json:"tags"`
	Path        string           `json:"path"`
	Ports       []ListPortOutput `json:"ports"`
}

type StatusItemOutput struct {
	Name        string         `json:"name"`
	DisplayName string         `json:"display_name,omitempty"`
	Description string         `json:"description,omitempty"`
	Tags        []string       `json:"tags"`
	Status      string         `json:"status"`
	Processes   int            `json:"processes"`
	Runtime     string         `json:"runtime"`
	StartedAt   *time.Time     `json:"started_at,omitempty"`
	Ports       map[string]int `json:"ports"`
	Health      any            `json:"health_status"`
}

type InfoOutput struct {
	Success  bool             `json:"success"`
	Scenario InfoScenarioData `json:"scenario"`
	Runtime  InfoRuntimeData  `json:"runtime"`
}

type InfoScenarioData struct {
	Name             string                       `json:"name"`
	DisplayName      string                       `json:"display_name,omitempty"`
	Description      string                       `json:"description,omitempty"`
	Version          string                       `json:"version,omitempty"`
	Type             string                       `json:"type,omitempty"`
	Category         string                       `json:"category,omitempty"`
	Tags             []string                     `json:"tags"`
	Path             string                       `json:"path"`
	ServicePath      string                       `json:"service_path"`
	SandboxRedirect  bool                         `json:"sandbox_redirected"`
	ConfigVersion    string                       `json:"config_version,omitempty"`
	LifecycleVersion string                       `json:"lifecycle_version,omitempty"`
	Ports            []scenariomodel.PortSummary  `json:"ports"`
	Phases           []scenariomodel.PhaseSummary `json:"phases"`
}

type InfoRuntimeData struct {
	Status      string           `json:"status"`
	Processes   int              `json:"processes"`
	Runtime     string           `json:"runtime"`
	StartedAt   *time.Time       `json:"started_at,omitempty"`
	Ports       map[string]int   `json:"ports"`
	ProcessInfo []process.Record `json:"process_records"`
	ListPorts   []ListPortOutput `json:"list_ports"`
}

type StatusSingleOutput struct {
	Success  bool             `json:"success"`
	Scenario StatusItemOutput `json:"scenario"`
	Info     InfoScenarioData `json:"info"`
	Runtime  InfoRuntimeData  `json:"runtime"`
}

type LifecycleItemOutput struct {
	Name               string           `json:"name"`
	Status             string           `json:"status"`
	Health             string           `json:"health,omitempty"`
	Ports              map[string]int   `json:"ports,omitempty"`
	Endpoints          []EndpointOutput `json:"endpoints,omitempty"`
	FailedDependencies []string         `json:"failed_dependencies,omitempty"`
	FailedResources    []string         `json:"failed_resources,omitempty"`
}

type EndpointOutput struct {
	Name        string `json:"name"`
	Key         string `json:"key"`
	Description string `json:"description,omitempty"`
	Port        int    `json:"port"`
	URL         string `json:"url"`
}

type BatchFailure struct {
	Name  string `json:"name"`
	Error string `json:"error"`
}

type BatchResponse struct {
	Verb    string
	Started []LifecycleItemOutput
	Stopped []string
	Failed  []BatchFailure
}

type ListResponse struct {
	Items        []ListItemOutput
	RunningCount int
}

type StatusResponse struct {
	Single *StatusSingleOutput
	List   []StatusItemOutput
}

type PortSingleOutput struct {
	Success  bool   `json:"success"`
	Scenario string `json:"scenario"`
	PortName string `json:"port_name"`
	Step     string `json:"step,omitempty"`
	Port     int    `json:"port,omitempty"`
	Error    string `json:"error,omitempty"`
}

type PortListOutput struct {
	Success  bool             `json:"success"`
	Scenario string           `json:"scenario"`
	Ports    []ListPortOutput `json:"ports"`
	Metadata map[string]int   `json:"metadata,omitempty"`
	Error    string           `json:"error,omitempty"`
}

type PortResponse struct {
	Single *PortSingleOutput
	List   *PortListOutput
}

type OpenOutput struct {
	Success  bool   `json:"success"`
	Scenario string `json:"scenario"`
	PortName string `json:"port_name"`
	Port     int    `json:"port"`
	URL      string `json:"url"`
}

func BuildStatusItem(item scenariomodel.Scenario, runtime process.ScenarioRuntime) StatusItemOutput {
	return BuildStatusDetail(orchestrator.Detail{
		Scenario: item,
		Runtime:  runtime,
		Details:  scenariomodel.DescribeRuntime(item.Manifest, runtime),
	})
}

func BuildStatusDetail(detail orchestrator.Detail) StatusItemOutput {
	health := any(nil)
	if detail.Details.Health != "" {
		health = detail.Details.Health
	}
	return StatusItemOutput{
		Name:        detail.Scenario.Slug,
		DisplayName: detail.Scenario.Manifest.Service.DisplayName,
		Description: detail.Scenario.Manifest.Service.Description,
		Tags:        CopyStrings(detail.Scenario.Manifest.Service.Tags),
		Status:      detail.Details.Status,
		Processes:   detail.Details.Processes,
		Runtime:     detail.Details.Runtime,
		StartedAt:   detail.Details.StartedAt,
		Ports:       CopyIntMap(detail.Details.Ports),
		Health:      health,
	}
}

func BuildInfoData(item scenariomodel.Scenario) InfoScenarioData {
	return InfoScenarioData{
		Name:             item.Slug,
		DisplayName:      item.Manifest.Service.DisplayName,
		Description:      item.Manifest.Service.Description,
		Version:          item.Manifest.Service.Version,
		Type:             item.Manifest.Service.Type,
		Category:         item.Manifest.Service.Category,
		Tags:             CopyStrings(item.Manifest.Service.Tags),
		Path:             item.Path,
		ServicePath:      item.ServicePath,
		SandboxRedirect:  item.Redirected,
		ConfigVersion:    item.Manifest.Version,
		LifecycleVersion: item.Manifest.Lifecycle.Version,
		Ports:            item.Manifest.SortedPorts(),
		Phases:           item.Manifest.PhaseSummaries(),
	}
}

func BuildRuntimeData(manifest scenariomodel.ServiceManifest, runtime process.ScenarioRuntime) InfoRuntimeData {
	details := scenariomodel.DescribeRuntime(manifest, runtime)
	return InfoRuntimeData{
		Status:      details.Status,
		Processes:   details.Processes,
		Runtime:     details.Runtime,
		StartedAt:   details.StartedAt,
		Ports:       CopyIntMap(details.Ports),
		ProcessInfo: CopyProcessRecords(details.ProcessInfo),
		ListPorts:   RuntimePortOutputs(details.PortBindings),
	}
}

func RuntimePortOutputs(bindings []scenariomodel.RuntimePortBinding) []ListPortOutput {
	listPorts := make([]ListPortOutput, 0, len(bindings))
	for _, binding := range bindings {
		listPorts = append(listPorts, ListPortOutput{
			Key:  binding.Key,
			Step: binding.Step,
			Port: binding.Port,
		})
	}
	return listPorts
}

func BuildListPorts(manifest scenariomodel.ServiceManifest, records []process.Record) ([]ListPortOutput, map[string]int) {
	bindings, ports := scenariomodel.RuntimePortBindings(manifest, records)
	return RuntimePortOutputs(bindings), CopyIntMap(ports)
}

func CopyIntMap(src map[string]int) map[string]int {
	if len(src) == 0 {
		return map[string]int{}
	}
	dup := make(map[string]int, len(src))
	for key, value := range src {
		dup[key] = value
	}
	return dup
}

func CopyStrings(values []string) []string {
	return append([]string(nil), values...)
}

func CopyProcessRecords(values []process.Record) []process.Record {
	return append([]process.Record(nil), values...)
}

func BatchResponseFromStartReport(report control.StartReport) BatchResponse {
	started := make([]LifecycleItemOutput, 0, len(report.Started))
	for _, item := range report.Started {
		started = append(started, LifecycleItemOutput{Name: item.Name, Status: "started"})
	}
	failed := make([]BatchFailure, 0, len(report.Failed))
	for _, item := range report.Failed {
		failed = append(failed, BatchFailure{Name: item.Name, Error: item.Error})
	}
	return BatchResponse{Verb: "Started", Started: started, Failed: failed}
}

func BatchResponseFromStopReport(report control.StopReport) BatchResponse {
	stopped := make([]string, 0, len(report.Stopped))
	for _, item := range report.Stopped {
		stopped = append(stopped, item.Name)
	}
	failed := make([]BatchFailure, 0, len(report.Failed))
	for _, item := range report.Failed {
		failed = append(failed, BatchFailure{Name: item.Name, Error: item.Error})
	}
	return BatchResponse{Verb: "Stopped", Stopped: stopped, Failed: failed}
}
