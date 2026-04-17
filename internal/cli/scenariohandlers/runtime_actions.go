package scenariohandlers

import (
	"io"

	scenarioapp "github.com/vrooli/vrooli/internal/app/scenario"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	. "github.com/vrooli/vrooli/internal/cli/scenariocli" //nolint:revive // scenariohandlers is a thin glue layer over scenariocli; dot-import keeps wiring readable.
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/lifecycle"
)

func StartResponse(run func(StartRequest) ([]scenarioapp.LifecycleItemOutput, error), req StartRequest) (cliout.Format, []LifecycleItemOutput, error) {
	items, err := run(req)
	return cliout.FormatHuman, toCLILifecycleItems(items), err
}

func ValidateEnvResponseFrom(run func(ValidateEnvRequest) (scenarioapp.ValidateEnvResponse, error), format cliout.Format, req ValidateEnvRequest) (ValidateEnvResponse, error) {
	_ = format
	resp, err := run(req)
	return ValidateEnvResponse{Report: resp.Report}, err
}

func StopResponse(run func(StopRequest) ([]scenarioapp.LifecycleItemOutput, error), req StopRequest) (cliout.Format, []LifecycleItemOutput, error) {
	items, err := run(req)
	return cliout.FormatHuman, toCLILifecycleItems(items), err
}

func RestartResponse(run func(RestartRequest) ([]scenarioapp.LifecycleItemOutput, error), req RestartRequest) (cliout.Format, []LifecycleItemOutput, error) {
	items, err := run(req)
	return cliout.FormatHuman, toCLILifecycleItems(items), err
}

func ListResponseFrom(format cliout.Format, run func(ListRequest) (scenarioapp.ListResponse, error), req ListRequest) (ListResponse, error) {
	resp, err := run(req)
	return toCLIListResponse(resp), err
}

func InfoResponseFrom(format cliout.Format, run func(InfoRequest) (scenarioapp.InfoOutput, error), req InfoRequest) (InfoOutput, error) {
	resp, err := run(req)
	return toCLIInfoOutput(resp), err
}

func StatusResponseFrom(format cliout.Format, run func(StatusRequest) (scenarioapp.StatusResponse, error), req StatusRequest) (StatusResponse, error) {
	resp, err := run(req)
	return toCLIStatusResponse(resp), err
}

func SetupResponseFrom(run func(SetupRequest) (lifecycle.PhaseResult, error), req SetupRequest) (cliout.Format, lifecycle.PhaseResult, error) {
	result, err := run(req)
	return cliout.FormatHuman, result, err
}

func TestResponseFrom(run func(TestRequest) error, req TestRequest) (cliout.Format, struct{}, error) {
	return cliout.FormatHuman, struct{}{}, run(req)
}

func BatchStartResponseFrom(format cliout.Format, run func() (scenarioapp.BatchResponse, error)) (BatchResponse, error) {
	resp, err := run()
	return toCLIBatchResponse(resp), err
}

func BatchStopResponseFrom(format cliout.Format, run func() (scenarioapp.BatchResponse, error)) (BatchResponse, error) {
	resp, err := run()
	return toCLIBatchResponse(resp), err
}

func PortResponseFrom(format cliout.Format, run func(PortRequest) (scenarioapp.PortResponse, error), req PortRequest) (PortResponse, error) {
	resp, err := run(req)
	if err != nil {
		if req.PortName == "" {
			return PortResponse{}, rootcli.RuntimeErrorf("Inspect the scenario status or start the scenario first", err.Error())
		}
		return PortResponse{}, rootcli.RuntimeErrorf("Start the scenario before querying runtime ports", err.Error())
	}
	return toCLIPortResponse(resp), nil
}

func OpenResponseFrom(run func(OpenRequest) (scenarioapp.OpenOutput, error), req OpenRequest) (cliout.Format, OpenOutput, error) {
	resp, err := run(req)
	if err != nil {
		return "", OpenOutput{}, err
	}
	if !req.PrintURL && !req.JSON {
		return cliout.FormatHuman, OpenOutput{}, nil
	}
	return cliout.FormatHuman, toCLIOpenOutput(resp), nil
}

func RenderSetupPhaseResult(w io.Writer, format cliout.Format, result lifecycle.PhaseResult) error {
	return RenderSetupResponse(w, format, result)
}

func toCLILifecycleItems(items []scenarioapp.LifecycleItemOutput) []LifecycleItemOutput {
	out := make([]LifecycleItemOutput, 0, len(items))
	for _, item := range items {
		out = append(out, LifecycleItemOutput{
			Name:               item.Name,
			Status:             item.Status,
			Health:             item.Health,
			Ports:              CopyIntMap(item.Ports),
			Endpoints:          toCLIEndpoints(item.Endpoints),
			FailedDependencies: CopyStrings(item.FailedDependencies),
			FailedResources:    CopyStrings(item.FailedResources),
		})
	}
	return out
}

func toCLIEndpoints(items []scenarioapp.EndpointOutput) []EndpointOutput {
	out := make([]EndpointOutput, 0, len(items))
	for _, item := range items {
		out = append(out, EndpointOutput{
			Name:        item.Name,
			Key:         item.Key,
			Description: item.Description,
			Port:        item.Port,
			URL:         item.URL,
		})
	}
	return out
}

func toCLIListPorts(items []scenarioapp.ListPortOutput) []ListPortOutput {
	out := make([]ListPortOutput, 0, len(items))
	for _, item := range items {
		out = append(out, ListPortOutput{Key: item.Key, Step: item.Step, Port: item.Port})
	}
	return out
}

func toCLIListResponse(resp scenarioapp.ListResponse) ListResponse {
	items := make([]ListItemOutput, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, ListItemOutput{
			Name:        item.Name,
			Description: item.Description,
			Version:     item.Version,
			Status:      item.Status,
			Tags:        CopyStrings(item.Tags),
			Path:        item.Path,
			Ports:       toCLIListPorts(item.Ports),
		})
	}
	return ListResponse{
		Items:        items,
		RunningCount: resp.RunningCount,
		Failures:     append(resp.Failures[:0:0], resp.Failures...),
	}
}

func toCLIInfoOutput(resp scenarioapp.InfoOutput) InfoOutput {
	return InfoOutput{
		Success: resp.Success,
		Scenario: InfoScenarioData{
			Name:             resp.Scenario.Name,
			DisplayName:      resp.Scenario.DisplayName,
			Description:      resp.Scenario.Description,
			Version:          resp.Scenario.Version,
			Type:             resp.Scenario.Type,
			Category:         resp.Scenario.Category,
			Tags:             CopyStrings(resp.Scenario.Tags),
			Path:             resp.Scenario.Path,
			ServicePath:      resp.Scenario.ServicePath,
			SandboxRedirect:  resp.Scenario.SandboxRedirect,
			ConfigVersion:    resp.Scenario.ConfigVersion,
			LifecycleVersion: resp.Scenario.LifecycleVersion,
			Ports:            append(resp.Scenario.Ports[:0:0], resp.Scenario.Ports...),
			Phases:           append(resp.Scenario.Phases[:0:0], resp.Scenario.Phases...),
		},
		Runtime: InfoRuntimeData{
			Status:      resp.Runtime.Status,
			Processes:   resp.Runtime.Processes,
			Runtime:     resp.Runtime.Runtime,
			StartedAt:   resp.Runtime.StartedAt,
			Ports:       CopyIntMap(resp.Runtime.Ports),
			ProcessInfo: CopyProcessRecords(resp.Runtime.ProcessInfo),
			ListPorts:   toCLIListPorts(resp.Runtime.ListPorts),
		},
	}
}

func toCLIStatusItem(item scenarioapp.StatusItemOutput) StatusItemOutput {
	return StatusItemOutput{
		Name:        item.Name,
		DisplayName: item.DisplayName,
		Description: item.Description,
		Tags:        CopyStrings(item.Tags),
		Status:      item.Status,
		Processes:   item.Processes,
		Runtime:     item.Runtime,
		StartedAt:   item.StartedAt,
		Ports:       CopyIntMap(item.Ports),
		Health:      item.Health,
	}
}

func toCLIStatusResponse(resp scenarioapp.StatusResponse) StatusResponse {
	out := StatusResponse{}
	if resp.Single != nil {
		out.Single = &StatusSingleOutput{
			Success:  resp.Single.Success,
			Scenario: toCLIStatusItem(resp.Single.Scenario),
			Info:     toCLIInfoOutput(scenarioapp.InfoOutput{Scenario: resp.Single.Info}).Scenario,
			Runtime:  toCLIInfoOutput(scenarioapp.InfoOutput{Runtime: resp.Single.Runtime}).Runtime,
		}
		return out
	}
	out.List = make([]StatusItemOutput, 0, len(resp.List))
	for _, item := range resp.List {
		out.List = append(out.List, toCLIStatusItem(item))
	}
	out.Failures = append(resp.Failures[:0:0], resp.Failures...)
	return out
}

func toCLIBatchResponse(resp scenarioapp.BatchResponse) BatchResponse {
	started := make([]LifecycleItemOutput, 0, len(resp.Started))
	for _, item := range resp.Started {
		started = append(started, LifecycleItemOutput{
			Name:               item.Name,
			Status:             item.Status,
			Health:             item.Health,
			Ports:              CopyIntMap(item.Ports),
			Endpoints:          toCLIEndpoints(item.Endpoints),
			FailedDependencies: CopyStrings(item.FailedDependencies),
			FailedResources:    CopyStrings(item.FailedResources),
		})
	}
	failed := make([]BatchFailure, 0, len(resp.Failed))
	for _, item := range resp.Failed {
		failed = append(failed, BatchFailure{Name: item.Name, Error: item.Error})
	}
	return BatchResponse{Verb: resp.Verb, Started: started, Stopped: CopyStrings(resp.Stopped), Failed: failed}
}

func toCLIPortResponse(resp scenarioapp.PortResponse) PortResponse {
	out := PortResponse{}
	if resp.Single != nil {
		out.Single = &PortSingleOutput{
			Success:  resp.Single.Success,
			Scenario: resp.Single.Scenario,
			PortName: resp.Single.PortName,
			Step:     resp.Single.Step,
			Port:     resp.Single.Port,
			Error:    resp.Single.Error,
		}
	}
	if resp.List != nil {
		out.List = &PortListOutput{
			Success:  resp.List.Success,
			Scenario: resp.List.Scenario,
			Ports:    toCLIListPorts(resp.List.Ports),
			Metadata: mapCopy(resp.List.Metadata),
			Error:    resp.List.Error,
		}
	}
	return out
}

func toCLIOpenOutput(resp scenarioapp.OpenOutput) OpenOutput {
	return OpenOutput{
		Success:  resp.Success,
		Scenario: resp.Scenario,
		PortName: resp.PortName,
		Port:     resp.Port,
		URL:      resp.URL,
	}
}

func mapCopy(values map[string]int) map[string]int {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]int, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func HealFromSandboxResponseFrom(run func(HealFromSandboxRequest) (scenarioapp.HealFromSandboxResponse, error), req HealFromSandboxRequest) (cliout.Format, HealFromSandboxResponse, error) {
	resp, err := run(req)
	if err != nil {
		return "", HealFromSandboxResponse{}, err
	}
	return cliout.FormatHuman, HealFromSandboxResponse{
		Affected:     CopyStrings(resp.Affected),
		DryRun:       resp.DryRun,
		StoppedCount: resp.StoppedCount,
	}, nil
}

func NewStartService(ops scenarioapp.ScenarioOperations, openURL func(string) error) scenarioapp.Service {
	return scenarioapp.Service{Scenarios: ops, OpenURL: openURL}
}

func NewRunnerService(runner scenarioapp.PhaseRunner) scenarioapp.Service {
	return scenarioapp.Service{Runner: runner}
}

func NewValidatorService(validator scenarioapp.EnvironmentValidator) scenarioapp.Service {
	return scenarioapp.Service{Validator: validator}
}
