package pipeline

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	pipelinev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/pipeline"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/shared"
)

func (c *Commands) runPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoMutation(func(ctx cliapp.OperationContext) (*pipelinev1.PipelineRunResponse, error) {
		config, err := pipelineConfigFromContext(ctx)
		if err != nil {
			return nil, err
		}
		response, err := c.rpc.Run(context.Background(), connect.NewRequest(&pipelinev1.PipelineRunRequest{Config: config}))
		if err != nil {
			return nil, cliapp.WrapAPIError("run pipeline", err, nil)
		}
		return response.Msg, nil
	}, pipelineRunReport)
}

func (c *Commands) startPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoMutation(func(ctx cliapp.OperationContext) (*pipelinev1.StartActivePipelineResponse, error) {
		config, err := pipelineConfigFromContext(ctx)
		if err != nil {
			return nil, err
		}
		response, err := c.rpc.StartActive(context.Background(), connect.NewRequest(&pipelinev1.StartActivePipelineRequest{ScenarioName: strings.TrimSpace(ctx.Positional("scenario")), ConfigOverrides: config}))
		if err != nil {
			return nil, cliapp.WrapAPIError("start active pipeline", err, nil)
		}
		return response.Msg, nil
	}, func(_ cliapp.OperationContext, response *pipelinev1.StartActivePipelineResponse) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{"Pipeline started: " + response.GetPipeline().GetPipelineId()}, NextCommand: []string{"scenario-to-desktop pipeline status " + response.GetPipeline().GetPipelineId()}}
	})
}

func pipelineRunReport(_ cliapp.OperationContext, response *pipelinev1.PipelineRunResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{"Pipeline started: " + response.GetPipelineId()}, NextCommand: []string{"scenario-to-desktop pipeline status " + response.GetPipelineId()}}
}

func (c *Commands) statusPrimitive() cliapp.PrimitiveHandler {
	return c.statusCall("get pipeline status", c.rpc.Get)
}

func (c *Commands) gatePrimitive() cliapp.PrimitiveHandler {
	return c.statusCall("get release gate", c.rpc.GetReleaseGate)
}

func (c *Commands) statusCall(operation string, call func(context.Context, *connect.Request[pipelinev1.PipelineGetRequest]) (*connect.Response[pipelinev1.PipelineStatus], error)) cliapp.PrimitiveHandler {
	return cliapp.ProtoList(func(ctx cliapp.OperationContext) (*pipelinev1.PipelineStatus, error) {
		response, err := call(context.Background(), connect.NewRequest(&pipelinev1.PipelineGetRequest{PipelineId: strings.TrimSpace(ctx.Positional("pipeline-id"))}))
		if err != nil {
			return nil, cliapp.WrapAPIError(operation, err, nil)
		}
		return response.Msg, nil
	}, pipelineStatusReport)
}

func pipelineStatusReport(_ cliapp.OperationContext, response *pipelinev1.PipelineStatus) cliapp.ListReport {
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Pipeline %s is %s (%d%%)", response.GetPipelineId(), response.GetStatus().String(), response.GetProgressPercent())}, Results: []string{fmt.Sprintf("Scenario: %s", response.GetScenarioName()), fmt.Sprintf("Stage: %s", response.GetCurrentStage().String())}}
}

func (c *Commands) resumePrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoMutation(func(ctx cliapp.OperationContext) (*pipelinev1.PipelineResumeResponse, error) {
		response, err := c.rpc.Resume(context.Background(), connect.NewRequest(&pipelinev1.PipelineResumeRequest{PipelineId: strings.TrimSpace(ctx.Positional("pipeline-id"))}))
		if err != nil {
			return nil, cliapp.WrapAPIError("resume pipeline", err, nil)
		}
		return response.Msg, nil
	}, func(_ cliapp.OperationContext, response *pipelinev1.PipelineResumeResponse) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{"Pipeline resumed: " + response.GetPipelineId()}}
	})
}

func (c *Commands) cancelPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoMutation(func(ctx cliapp.OperationContext) (*pipelinev1.PipelineCancelResponse, error) {
		response, err := c.rpc.Cancel(context.Background(), connect.NewRequest(&pipelinev1.PipelineCancelRequest{PipelineId: strings.TrimSpace(ctx.Positional("pipeline-id"))}))
		if err != nil {
			return nil, cliapp.WrapAPIError("cancel pipeline", err, nil)
		}
		return response.Msg, nil
	}, func(_ cliapp.OperationContext, response *pipelinev1.PipelineCancelResponse) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{"Pipeline cancellation: " + response.GetStatus()}}
	})
}

func (c *Commands) listPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoList(func(_ cliapp.OperationContext) (*pipelinev1.PipelineListResponse, error) {
		response, err := c.rpc.List(context.Background(), connect.NewRequest(&pipelinev1.PipelineListRequest{}))
		if err != nil {
			return nil, cliapp.WrapAPIError("list pipelines", err, nil)
		}
		return response.Msg, nil
	}, func(_ cliapp.OperationContext, response *pipelinev1.PipelineListResponse) cliapp.ListReport {
		return cliapp.ListReport{Summary: []string{fmt.Sprintf("%d pipeline(s)", len(response.GetPipelines()))}}
	})
}

func (c *Commands) activePrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoList(func(ctx cliapp.OperationContext) (*pipelinev1.ActivePipelineResponse, error) {
		response, err := c.rpc.GetActive(context.Background(), connect.NewRequest(&pipelinev1.GetActivePipelineRequest{ScenarioName: strings.TrimSpace(ctx.Positional("scenario")), AutoCreate: false}))
		if err != nil {
			return nil, cliapp.WrapAPIError("get active pipeline", err, nil)
		}
		return response.Msg, nil
	}, func(_ cliapp.OperationContext, response *pipelinev1.ActivePipelineResponse) cliapp.ListReport {
		return cliapp.ListReport{Summary: []string{"Active pipeline: " + response.GetPipeline().GetPipelineId()}}
	})
}

func (c *Commands) createPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoMutation(func(ctx cliapp.OperationContext) (*pipelinev1.CreatePipelineResponse, error) {
		response, err := c.rpc.CreateActive(context.Background(), connect.NewRequest(&pipelinev1.CreatePipelineRequest{ScenarioName: strings.TrimSpace(ctx.Positional("scenario"))}))
		if err != nil {
			return nil, cliapp.WrapAPIError("create active pipeline", err, nil)
		}
		return response.Msg, nil
	}, func(_ cliapp.OperationContext, response *pipelinev1.CreatePipelineResponse) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{"Pipeline created: " + response.GetPipeline().GetPipelineId()}}
	})
}

func (c *Commands) resetPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoMutation(func(ctx cliapp.OperationContext) (*pipelinev1.ResetPipelineResponse, error) {
		response, err := c.rpc.ResetActive(context.Background(), connect.NewRequest(&pipelinev1.ScenarioPipelineRequest{ScenarioName: strings.TrimSpace(ctx.Positional("scenario"))}))
		if err != nil {
			return nil, cliapp.WrapAPIError("reset active pipeline", err, nil)
		}
		return response.Msg, nil
	}, func(_ cliapp.OperationContext, response *pipelinev1.ResetPipelineResponse) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{fmt.Sprintf("Pipeline reset (cleared=%t)", response.GetCleared())}}
	})
}

func (c *Commands) historyPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoList(func(ctx cliapp.OperationContext) (*pipelinev1.PipelineHistoryResponse, error) {
		limit, err := positiveInt32(ctx.Flag("limit"), "limit")
		if err != nil {
			return nil, err
		}
		response, err := c.rpc.GetHistory(context.Background(), connect.NewRequest(&pipelinev1.PipelineHistoryRequest{ScenarioName: strings.TrimSpace(ctx.Positional("scenario")), Limit: &limit}))
		if err != nil {
			return nil, cliapp.WrapAPIError("get pipeline history", err, nil)
		}
		return response.Msg, nil
	}, func(_ cliapp.OperationContext, response *pipelinev1.PipelineHistoryResponse) cliapp.ListReport {
		return cliapp.ListReport{Summary: []string{fmt.Sprintf("%d historical pipeline(s)", response.GetTotal())}}
	})
}

func positiveInt32(value, name string) (int32, error) {
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 32)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("--%s must be a positive integer", name)
	}
	return int32(parsed), nil
}

func pipelineConfigFromContext(ctx cliapp.OperationContext) (*pipelinev1.PipelineConfig, error) {
	config := &pipelinev1.PipelineConfig{ScenarioName: strings.TrimSpace(ctx.Positional("scenario"))}
	for _, value := range splitValues(ctx.Flag("platforms")) {
		value = strings.SplitN(value, "-", 2)[0]
		platform, ok := map[string]sharedv1.Platform{"win": sharedv1.Platform_PLATFORM_WIN, "windows": sharedv1.Platform_PLATFORM_WIN, "mac": sharedv1.Platform_PLATFORM_MAC, "macos": sharedv1.Platform_PLATFORM_MAC, "darwin": sharedv1.Platform_PLATFORM_MAC, "linux": sharedv1.Platform_PLATFORM_LINUX}[value]
		if !ok {
			return nil, fmt.Errorf("unsupported platform %q (expected win, mac, or linux)", value)
		}
		config.Platforms = append(config.Platforms, platform)
	}
	for _, value := range splitValues(ctx.Flag("stages")) {
		stage, ok := map[string]sharedv1.StageName{"bundle": sharedv1.StageName_STAGE_NAME_BUNDLE, "preflight": sharedv1.StageName_STAGE_NAME_PREFLIGHT, "generate": sharedv1.StageName_STAGE_NAME_GENERATE, "build": sharedv1.StageName_STAGE_NAME_BUILD, "smoketest": sharedv1.StageName_STAGE_NAME_SMOKE_TEST, "smoke_test": sharedv1.StageName_STAGE_NAME_SMOKE_TEST, "deploy": sharedv1.StageName_STAGE_NAME_DEPLOY}[value]
		if !ok {
			return nil, fmt.Errorf("unsupported stage %q", value)
		}
		config.Stages = append(config.Stages, stage)
	}
	switch ctx.Flag("deployment-mode") {
	case "bundled":
		config.DeploymentMode = sharedv1.DeploymentMode_DEPLOYMENT_MODE_BUNDLED
	case "proxy":
		config.DeploymentMode = sharedv1.DeploymentMode_DEPLOYMENT_MODE_PROXY
	default:
		return nil, fmt.Errorf("unsupported deployment mode %q", ctx.Flag("deployment-mode"))
	}
	if value := strings.TrimSpace(ctx.Flag("location-mode")); value != "" {
		config.LocationMode = &value
	}
	if value := strings.TrimSpace(ctx.Flag("resource-artifact-root")); value != "" {
		config.ResourceArtifactRoot = &value
	}
	if value := strings.TrimSpace(ctx.Flag("artifact-trust-mode")); value != "" {
		config.ArtifactTrustMode = &value
	}
	if provider, updateURL, channel := strings.TrimSpace(ctx.Flag("update-provider")), strings.TrimSpace(ctx.Flag("update-url")), strings.TrimSpace(ctx.Flag("update-channel")); provider != "" || updateURL != "" || channel != "" || ctx.BoolFlag("update-auto-check") {
		if provider == "" {
			provider = "generic"
		}
		update := &sharedv1.UpdateConfig{Provider: &provider}
		if channel != "" {
			update.Channel = &channel
		}
		if ctx.BoolFlag("update-auto-check") {
			autoCheck := true
			update.AutoCheck = &autoCheck
		}
		if updateURL != "" {
			update.Generic = &sharedv1.GenericUpdateConfig{Url: updateURL}
		}
		config.UpdateConfig = update
	}
	if ctx.BoolFlag("clean") {
		clean := true
		config.Clean = &clean
	}
	if value := strings.TrimSpace(ctx.Flag("version")); value != "" {
		config.Version = &value
	}
	return config, nil
}
