package scenariohandlers

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"time"

	scenarioapp "github.com/vrooli/vrooli/internal/app/scenario"
	"github.com/vrooli/vrooli/internal/cli/rootcli"
	. "github.com/vrooli/vrooli/internal/cli/scenariocli" //nolint:revive // scenariohandlers is a thin glue layer over scenariocli; dot-import keeps wiring readable.
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/tuning"
)

const (
	runtimeActionsParameterA = 124
)

func StartResponse(run func(StartRequest) ([]scenarioapp.LifecycleItemOutput, error), req StartRequest) (cliout.Format, []LifecycleItemOutput, error) {
	items, err := run(req)
	return cliout.FormatHuman, toCLILifecycleItems(items), err
}

func ValidateEnvResponseFrom(run func(ValidateEnvRequest) (scenarioapp.ValidateEnvResponse, error), format cliout.Format, req ValidateEnvRequest) (ValidateEnvResponse, error) {
	_ = format
	resp, err := run(req)
	return resp, err
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
			return PortResponse{}, rootcli.RuntimeErrorf("Inspect the scenario status or start the scenario first", "%s", err.Error())
		}
		return PortResponse{}, rootcli.RuntimeErrorf("Start the scenario before querying runtime ports", "%s", err.Error())
	}
	return toCLIPortResponse(resp), nil
}

func OpenResponseFrom(run func(OpenRequest) (scenarioapp.OpenOutput, error), req OpenRequest) (OpenOutput, error) {
	resp, err := run(req)
	if err != nil {
		return OpenOutput{}, err
	}
	if !req.PrintURL && !req.JSON {
		return OpenOutput{}, nil
	}
	return toCLIOpenOutput(resp), nil
}

func RenderSetupPhaseResult(w io.Writer, format cliout.Format, result lifecycle.PhaseResult) error {
	return RenderSetupResponse(w, format, result)
}

// runWithStartCeiling bounds a blocking start/restart with the --timeout
// ceiling. On expiry the CLI detaches (exit 124): the in-process orchestration
// dies with this process, but the operation record stays honest — the
// initiator pid goes dead, readers report the operation abandoned, and the
// next `scenario start`/`scenario wait` resumes or attaches. This asymmetry
// vs a server-owned run is documented in cli-commands.md.
func runWithStartCeiling(timeoutSeconds int, stderr io.Writer, reattachName string, run func(context.Context) ([]scenarioapp.LifecycleItemOutput, error)) ([]scenarioapp.LifecycleItemOutput, error) {
	operationCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	if timeoutSeconds <= 0 {
		return run(operationCtx)
	}
	operationCtx, cancel = context.WithTimeout(operationCtx, tuning.ScenarioActionTimeout(time.Duration(timeoutSeconds)*time.Second))
	defer cancel()
	type result struct {
		items []scenarioapp.LifecycleItemOutput
		err   error
	}
	ch := make(chan result, 1)
	go func() {
		items, err := run(operationCtx)
		ch <- result{items, err}
	}()
	timer := time.NewTimer(time.Duration(timeoutSeconds) * time.Second)
	defer timer.Stop()
	select {
	case r := <-ch:
		return r.items, r.err
	case <-timer.C:
		cancel()
		fmt.Fprintf(stderr, "scenario start: --timeout ceiling (%ds) elapsed; detaching. The orchestration stops with this process, but the operation record stays honest — resume with `vrooli scenario start %s` or attach with `vrooli scenario wait %s --json`.\n", timeoutSeconds, reattachName, reattachName)
		return nil, VerdictExitError{Code: runtimeActionsParameterA}
	}
}

func toCLILifecycleItems(items []scenarioapp.LifecycleItemOutput) []LifecycleItemOutput {
	out := make([]LifecycleItemOutput, 0, len(items))
	for _, item := range items {
		out = append(out, LifecycleItemOutput{
			Name:               item.Name,
			Status:             item.Status,
			Health:             item.Health,
			Ports:              scenarioapp.CopyIntMap(item.Ports),
			Endpoints:          toCLIEndpoints(item.Endpoints),
			FailedDependencies: scenarioapp.CopyStrings(item.FailedDependencies),
			FailedResources:    scenarioapp.CopyStrings(item.FailedResources),
			Verdict:            item.Verdict,
			Operation:          item.Operation,
		})
	}
	return out
}

func toCLIEndpoints(items []scenarioapp.EndpointOutput) []EndpointOutput {
	return append([]EndpointOutput(nil), items...)
}

func toCLIListPorts(items []scenarioapp.ListPortOutput) []ListPortOutput {
	return append([]ListPortOutput(nil), items...)
}

func toCLIListResponse(resp scenarioapp.ListResponse) ListResponse {
	items := make([]ListItemOutput, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, ListItemOutput{
			Name:        item.Name,
			Description: item.Description,
			Version:     item.Version,
			Status:      item.Status,
			Tags:        scenarioapp.CopyStrings(item.Tags),
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
			Tags:             scenarioapp.CopyStrings(resp.Scenario.Tags),
			Path:             resp.Scenario.Path,
			ServicePath:      resp.Scenario.ServicePath,
			SandboxRedirect:  resp.Scenario.SandboxRedirect,
			ConfigVersion:    resp.Scenario.ConfigVersion,
			LifecycleVersion: resp.Scenario.LifecycleVersion,
			Ports:            append(resp.Scenario.Ports[:0:0], resp.Scenario.Ports...),
			Phases:           append(resp.Scenario.Phases[:0:0], resp.Scenario.Phases...),
			Generation:       resp.Scenario.Generation,
		},
		Runtime: InfoRuntimeData{
			Status:      resp.Runtime.Status,
			Processes:   resp.Runtime.Processes,
			Runtime:     resp.Runtime.Runtime,
			StartedAt:   resp.Runtime.StartedAt,
			Ports:       scenarioapp.CopyIntMap(resp.Runtime.Ports),
			ProcessInfo: scenarioapp.CopyProcessRecords(resp.Runtime.ProcessInfo),
			ListPorts:   toCLIListPorts(resp.Runtime.ListPorts),
			HealthError: resp.Runtime.HealthError,
		},
	}
}

func toCLIStatusItem(item scenarioapp.StatusItemOutput) StatusItemOutput {
	return StatusItemOutput{
		Name:           item.Name,
		DisplayName:    item.DisplayName,
		Description:    item.Description,
		Tags:           scenarioapp.CopyStrings(item.Tags),
		Status:         item.Status,
		Processes:      item.Processes,
		Runtime:        item.Runtime,
		StartedAt:      item.StartedAt,
		Ports:          scenarioapp.CopyIntMap(item.Ports),
		PortBindings:   toCLIListPorts(item.PortBindings),
		Health:         item.Health,
		HealthError:    item.HealthError,
		StartOperation: item.StartOperation,
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
			Ports:              scenarioapp.CopyIntMap(item.Ports),
			Endpoints:          toCLIEndpoints(item.Endpoints),
			FailedDependencies: scenarioapp.CopyStrings(item.FailedDependencies),
			FailedResources:    scenarioapp.CopyStrings(item.FailedResources),
		})
	}
	failed := append([]BatchFailure(nil), resp.Failed...)
	return BatchResponse{Verb: resp.Verb, Started: started, Stopped: scenarioapp.CopyStrings(resp.Stopped), Failed: failed}
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
	return resp
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

func HealFromSandboxResponseFrom(run func(HealFromSandboxRequest) (scenarioapp.HealFromSandboxResponse, error), req HealFromSandboxRequest) (HealFromSandboxResponse, error) {
	resp, err := run(req)
	if err != nil {
		return HealFromSandboxResponse{}, err
	}
	return HealFromSandboxResponse{
		Affected:     scenarioapp.CopyStrings(resp.Affected),
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
