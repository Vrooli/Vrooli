package scenarioapp

import (
	"fmt"
	"os"
	"strings"

	"github.com/vrooli/vrooli/internal/cli/scenariocli"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/orchestrator"
	"github.com/vrooli/vrooli/internal/resources"
	scenariomodel "github.com/vrooli/vrooli/internal/scenario"
)

type EnvironmentValidator interface {
	ValidateScenarioEnvironment(name string) (resources.ScenarioEnvValidationReport, error)
}

type ScenarioOperations interface {
	StartDetailed(name string, opts lifecycle.StartOptions) (orchestrator.StartResult, error)
	RestartDetailed(name string, opts lifecycle.StartOptions) (orchestrator.StartResult, error)
	Inventory() ([]orchestrator.Detail, error)
	Detail(name string) (orchestrator.Detail, error)
	StartAll() (control.StartReport, error)
	StopAll() (control.StopReport, error)
	ResolvePort(name, portName string) (orchestrator.ResolvedPort, error)
}

type PhaseRunner interface {
	Stop(name string, opts lifecycle.StopOptions) error
	RunPhaseDetailed(name, phase string, opts lifecycle.PhaseOptions) (lifecycle.PhaseResult, error)
	RunPhase(name, phase string, opts lifecycle.PhaseOptions) error
}

type Service struct {
	Scenarios ScenarioOperations
	Runner    PhaseRunner
	Validator EnvironmentValidator
	OpenURL   func(string) error
	Format    cliout.Format
}

func (s Service) Start(req scenariocli.StartRequest) ([]scenariocli.LifecycleItemOutput, error) {
	items := make([]scenariocli.LifecycleItemOutput, 0, len(req.Names))
	for _, name := range req.Names {
		result, err := s.Scenarios.StartDetailed(name, req.Options)
		if err != nil {
			return nil, err
		}

		status := "started"
		if result.AlreadyRunning {
			status = "already_running"
		}
		items = append(items, scenariocli.LifecycleItemOutput{
			Name:               result.Scenario.Slug,
			Status:             status,
			Health:             result.Details.Health,
			Ports:              envPortMap(result.Scenario.Manifest, result.AllocatedPorts),
			FailedDependencies: append([]string(nil), result.FailedDependencies...),
		})

		if req.OpenAfter {
			resolved, err := s.Scenarios.ResolvePort(name, "UI_PORT")
			if err != nil {
				return nil, err
			}
			if err := s.OpenURL(resolved.URL); err != nil {
				return nil, err
			}
		}
	}
	return items, nil
}

func (s Service) Restart(req scenariocli.RestartRequest) ([]scenariocli.LifecycleItemOutput, error) {
	result, err := s.Scenarios.RestartDetailed(req.Name, req.Options)
	if err != nil {
		return nil, err
	}

	item := scenariocli.LifecycleItemOutput{
		Name:               result.Scenario.Slug,
		Status:             "restarted",
		Health:             result.Details.Health,
		Ports:              envPortMap(result.Scenario.Manifest, result.AllocatedPorts),
		FailedDependencies: append([]string(nil), result.FailedDependencies...),
	}

	if req.OpenAfter {
		resolved, err := s.Scenarios.ResolvePort(req.Name, "UI_PORT")
		if err != nil {
			return nil, err
		}
		if err := s.OpenURL(resolved.URL); err != nil {
			return nil, err
		}
	}

	return []scenariocli.LifecycleItemOutput{item}, nil
}

func (s Service) Stop(req scenariocli.StopRequest) ([]scenariocli.LifecycleItemOutput, error) {
	if err := s.Runner.Stop(req.Name, lifecycle.StopOptions{}); err != nil {
		return nil, err
	}
	return []scenariocli.LifecycleItemOutput{{Name: req.Name, Status: "stopped"}}, nil
}

func (s Service) List(req scenariocli.ListRequest) (scenariocli.ListResponse, error) {
	inventory, err := s.Scenarios.Inventory()
	if err != nil {
		return scenariocli.ListResponse{}, err
	}

	resp := scenariocli.ListResponse{Items: make([]scenariocli.ListItemOutput, 0, len(inventory))}
	for _, item := range inventory {
		status := "available"
		if item.Details.Status == "running" {
			status = item.Details.Status
			resp.RunningCount++
		}

		listPorts := []scenariocli.ListPortOutput{}
		if req.IncludePorts && item.Details.Status == "running" {
			listPorts = scenariocli.RuntimePortOutputs(item.Details.PortBindings)
		}

		resp.Items = append(resp.Items, scenariocli.ListItemOutput{
			Name:        item.Scenario.Slug,
			Description: item.Scenario.Manifest.Service.Description,
			Version:     item.Scenario.Manifest.Service.Version,
			Status:      status,
			Tags:        scenariocli.CopyStrings(item.Scenario.Manifest.Service.Tags),
			Path:        item.Scenario.Path + string(os.PathSeparator),
			Ports:       listPorts,
		})
	}
	return resp, nil
}

func (s Service) Info(req scenariocli.InfoRequest) (scenariocli.InfoOutput, error) {
	detail, err := s.Scenarios.Detail(req.Name)
	if err != nil {
		return scenariocli.InfoOutput{}, err
	}
	return scenariocli.InfoOutput{
		Success:  true,
		Scenario: scenariocli.BuildInfoData(detail.Scenario),
		Runtime:  scenariocli.BuildRuntimeData(detail.Scenario.Manifest, detail.Runtime),
	}, nil
}

func (s Service) Status(req scenariocli.StatusRequest) (scenariocli.StatusResponse, error) {
	if req.Name == "" {
		inventory, err := s.Scenarios.Inventory()
		if err != nil {
			return scenariocli.StatusResponse{}, err
		}
		items := make([]scenariocli.StatusItemOutput, 0, len(inventory))
		for _, item := range inventory {
			items = append(items, scenariocli.BuildStatusDetail(item))
		}
		return scenariocli.StatusResponse{List: items}, nil
	}

	detail, err := s.Scenarios.Detail(req.Name)
	if err != nil {
		return scenariocli.StatusResponse{}, err
	}
	output := scenariocli.StatusSingleOutput{
		Success:  true,
		Scenario: scenariocli.BuildStatusDetail(detail),
		Info:     scenariocli.BuildInfoData(detail.Scenario),
		Runtime:  scenariocli.BuildRuntimeData(detail.Scenario.Manifest, detail.Runtime),
	}
	return scenariocli.StatusResponse{Single: &output}, nil
}

func (s Service) ValidateEnv(req scenariocli.ValidateEnvRequest) (scenariocli.ValidateEnvResponse, error) {
	report, err := s.Validator.ValidateScenarioEnvironment(req.Name)
	if err != nil {
		return scenariocli.ValidateEnvResponse{}, err
	}
	return scenariocli.ValidateEnvResponse{Report: report}, nil
}

func (s Service) Setup(req scenariocli.SetupRequest) (lifecycle.PhaseResult, error) {
	return s.Runner.RunPhaseDetailed(req.Name, "setup", req.Opts)
}

func (s Service) Test(req scenariocli.TestRequest) error {
	return s.Runner.RunPhase(req.Name, "test", req.Opts)
}

func (s Service) StartAll() (scenariocli.BatchResponse, error) {
	report, err := s.Scenarios.StartAll()
	if err != nil {
		return scenariocli.BatchResponse{}, err
	}
	started := make([]scenariocli.LifecycleItemOutput, 0, len(report.Started))
	for _, item := range report.Started {
		started = append(started, scenariocli.LifecycleItemOutput{Name: item.Name, Status: "started"})
	}
	failed := make([]scenariocli.BatchFailure, 0, len(report.Failed))
	for _, item := range report.Failed {
		failed = append(failed, scenariocli.BatchFailure{Name: item.Name, Error: item.Error})
	}
	return scenariocli.BatchResponse{Verb: "Started", Started: started, Failed: failed}, nil
}

func (s Service) StopAll() (scenariocli.BatchResponse, error) {
	report, err := s.Scenarios.StopAll()
	if err != nil {
		return scenariocli.BatchResponse{}, err
	}
	stopped := make([]string, 0, len(report.Stopped))
	for _, item := range report.Stopped {
		stopped = append(stopped, item.Name)
	}
	failed := make([]scenariocli.BatchFailure, 0, len(report.Failed))
	for _, item := range report.Failed {
		failed = append(failed, scenariocli.BatchFailure{Name: item.Name, Error: item.Error})
	}
	return scenariocli.BatchResponse{Verb: "Stopped", Stopped: stopped, Failed: failed}, nil
}

func (s Service) Port(req scenariocli.PortRequest) (scenariocli.PortResponse, error) {
	detail, err := s.Scenarios.Detail(req.ScenarioName)
	if err != nil {
		return scenariocli.PortResponse{}, err
	}
	listPorts, portsMap := scenariocli.BuildListPorts(detail.Scenario.Manifest, detail.Runtime.Records)

	if req.PortName == "" {
		if detail.Runtime.ProcessCount == 0 || len(portsMap) == 0 {
			if req.JSON {
				return scenariocli.PortResponse{List: &scenariocli.PortListOutput{
					Success:  false,
					Scenario: req.ScenarioName,
					Ports:    []scenariocli.ListPortOutput{},
					Error:    "No running processes found for scenario",
				}}, nil
			}
			return scenariocli.PortResponse{}, fmt.Errorf("no running processes found for scenario %q", req.ScenarioName)
		}
		list := &scenariocli.PortListOutput{
			Success:  true,
			Scenario: req.ScenarioName,
			Ports:    listPorts,
		}
		if req.JSON {
			list.Metadata = map[string]int{"count": len(listPorts)}
		}
		return scenariocli.PortResponse{List: list}, nil
	}

	key, port, step, ok := resolveRequestedPort(detail.Scenario.Manifest, listPorts, portsMap, req.PortName)
	if !ok {
		if req.JSON {
			return scenariocli.PortResponse{Single: &scenariocli.PortSingleOutput{
				Success:  false,
				Scenario: req.ScenarioName,
				PortName: req.PortName,
				Error:    fmt.Sprintf("No running port named %s for scenario", req.PortName),
			}}, nil
		}
		return scenariocli.PortResponse{}, fmt.Errorf("no running port named %s for scenario %q", req.PortName, req.ScenarioName)
	}

	return scenariocli.PortResponse{Single: &scenariocli.PortSingleOutput{
		Success:  true,
		Scenario: req.ScenarioName,
		PortName: key,
		Step:     step,
		Port:     port,
	}}, nil
}

func (s Service) Open(req scenariocli.OpenRequest) (scenariocli.OpenOutput, error) {
	resolved, err := s.Scenarios.ResolvePort(req.ScenarioName, req.PortName)
	if err != nil {
		return scenariocli.OpenOutput{}, err
	}
	if !req.PrintURL && !req.JSON {
		if err := s.OpenURL(resolved.URL); err != nil {
			return scenariocli.OpenOutput{}, err
		}
		return scenariocli.OpenOutput{}, nil
	}
	return scenariocli.OpenOutput{
		Success:  true,
		Scenario: req.ScenarioName,
		PortName: resolved.Name,
		Port:     resolved.Port,
		URL:      resolved.URL,
	}, nil
}

func envPortMap(manifest scenariomodel.ServiceManifest, ports map[string]int) map[string]int {
	out := make(map[string]int, len(ports))
	for portName, port := range ports {
		envVar := manifest.PortEnvVar(portName)
		if envVar == "" {
			envVar = strings.ToUpper(strings.ReplaceAll(portName, "-", "_")) + "_PORT"
		}
		out[envVar] = port
	}
	return out
}

func resolveRequestedPort(manifest scenariomodel.ServiceManifest, listPorts []scenariocli.ListPortOutput, portsMap map[string]int, requested string) (string, int, string, bool) {
	candidates := []string{requested}
	if envVar := manifest.PortEnvVar(strings.ToLower(strings.TrimSuffix(requested, "_PORT"))); envVar != "" {
		candidates = append(candidates, envVar)
	}
	normalized := strings.ToUpper(strings.TrimSpace(requested))
	if normalized != "" && normalized != requested {
		candidates = append(candidates, normalized)
		if !strings.HasSuffix(normalized, "_PORT") {
			candidates = append(candidates, normalized+"_PORT")
		}
	}

	seen := map[string]struct{}{}
	for _, key := range candidates {
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		if portValue, ok := portsMap[key]; ok {
			stepName := ""
			for _, entry := range listPorts {
				if entry.Key == key && entry.Port == portValue {
					stepName = entry.Step
					break
				}
			}
			return key, portValue, stepName, true
		}
	}
	return "", 0, "", false
}
