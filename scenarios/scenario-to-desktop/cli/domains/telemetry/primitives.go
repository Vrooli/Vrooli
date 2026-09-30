package telemetry

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"google.golang.org/protobuf/types/known/structpb"
)

// downloadResult is deliberately a small action response: the bytes themselves
// are a raw JSONL export, not a structured CLI document. The action renderer
// keeps metadata output stable while the file remains available to callers.
type downloadResult struct {
	ScenarioName string `json:"scenario_name"`
	OutputPath   string `json:"output_path,omitempty"`
	Bytes        int    `json:"bytes"`
}

func (c *Commands) ingestPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoMutation(func(ctx cliapp.OperationContext) (*domainv1.IngestTelemetryResponse, error) {
		events, err := telemetryEvents(ctx.Flag("file"))
		if err != nil {
			return nil, err
		}
		source := strings.TrimSpace(ctx.Flag("source"))
		response, err := c.rpc.IngestTelemetry(context.Background(), connect.NewRequest(&domainv1.IngestTelemetryRequest{
			ScenarioName: strings.TrimSpace(ctx.Positional("scenario")),
			Source:       &source,
			Events:       events,
		}))
		if err != nil {
			return nil, cliapp.WrapAPIError("ingest telemetry", err, nil)
		}
		return response.Msg, nil
	}, func(ctx cliapp.OperationContext, response *domainv1.IngestTelemetryResponse) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{fmt.Sprintf("Ingested %d event(s) for %s", response.GetEventsIngested(), ctx.Positional("scenario"))}}
	})
}

func (c *Commands) summaryPrimitive() cliapp.PrimitiveHandler {
	return c.payloadPrimitive("get telemetry summary", c.rpc.GetTelemetrySummary)
}

func (c *Commands) insightsPrimitive() cliapp.PrimitiveHandler {
	return c.payloadPrimitive("get telemetry insights", c.rpc.GetTelemetryInsights)
}

func (c *Commands) payloadPrimitive(operation string, call func(context.Context, *connect.Request[domainv1.TelemetryScenarioRequest]) (*connect.Response[domainv1.TelemetryPayloadResponse], error)) cliapp.PrimitiveHandler {
	return cliapp.ProtoList(func(ctx cliapp.OperationContext) (*domainv1.TelemetryPayloadResponse, error) {
		response, err := call(context.Background(), connect.NewRequest(&domainv1.TelemetryScenarioRequest{ScenarioName: strings.TrimSpace(ctx.Positional("scenario"))}))
		if err != nil {
			return nil, cliapp.WrapAPIError(operation, err, nil)
		}
		return response.Msg, nil
	}, func(_ cliapp.OperationContext, response *domainv1.TelemetryPayloadResponse) cliapp.ListReport {
		return cliapp.ListReport{Summary: []string{"Telemetry payload retrieved"}, Results: []string{fmt.Sprintf("Fields: %d", len(response.GetPayload().GetFields()))}}
	})
}

func (c *Commands) tailPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoList(func(ctx cliapp.OperationContext) (*domainv1.TelemetryPayloadResponse, error) {
		limit, err := strconv.ParseInt(strings.TrimSpace(ctx.Flag("limit")), 10, 32)
		if err != nil || limit <= 0 {
			return nil, fmt.Errorf("--limit must be a positive integer")
		}
		limitValue := int32(limit)
		response, err := c.rpc.GetTelemetryTail(context.Background(), connect.NewRequest(&domainv1.TelemetryTailRequest{ScenarioName: strings.TrimSpace(ctx.Positional("scenario")), Limit: &limitValue}))
		if err != nil {
			return nil, cliapp.WrapAPIError("get telemetry tail", err, nil)
		}
		return response.Msg, nil
	}, func(_ cliapp.OperationContext, response *domainv1.TelemetryPayloadResponse) cliapp.ListReport {
		return cliapp.ListReport{Summary: []string{"Telemetry tail retrieved"}, Results: []string{fmt.Sprintf("Fields: %d", len(response.GetPayload().GetFields()))}}
	})
}

func (c *Commands) downloadPrimitive() cliapp.PrimitiveHandler {
	return cliapp.Action(func(ctx cliapp.OperationContext) (downloadResult, error) {
		scenario := strings.TrimSpace(ctx.Positional("scenario"))
		body, err := c.deps.Get("/deployment/telemetry/"+scenario+"/download", nil)
		if err != nil {
			return downloadResult{}, err
		}
		result := downloadResult{ScenarioName: scenario, Bytes: len(body)}
		if outputPath := strings.TrimSpace(ctx.Flag("output")); outputPath != "" {
			if err := os.WriteFile(outputPath, body, 0o644); err != nil {
				return downloadResult{}, fmt.Errorf("write telemetry file: %w", err)
			}
			result.OutputPath = outputPath
		}
		return result, nil
	}, func(_ cliapp.OperationContext, result downloadResult) cliapp.MutationReport {
		line := fmt.Sprintf("Downloaded %d bytes for %s", result.Bytes, result.ScenarioName)
		if result.OutputPath != "" {
			line += " to " + result.OutputPath
		}
		return cliapp.MutationReport{Result: []string{line}}
	})
}

func (c *Commands) deletePrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoMutation(func(ctx cliapp.OperationContext) (*domainv1.TelemetryDeleteResponse, error) {
		response, err := c.rpc.DeleteTelemetry(context.Background(), connect.NewRequest(&domainv1.TelemetryScenarioRequest{ScenarioName: strings.TrimSpace(ctx.Positional("scenario"))}))
		if err != nil {
			return nil, cliapp.WrapAPIError("delete telemetry", err, nil)
		}
		return response.Msg, nil
	}, func(_ cliapp.OperationContext, response *domainv1.TelemetryDeleteResponse) cliapp.MutationReport {
		return cliapp.MutationReport{Result: []string{fmt.Sprintf("Telemetry deleted for %s", response.GetScenarioName())}}
	})
}

func telemetryEvents(path string) ([]*structpb.Struct, error) {
	data, err := os.ReadFile(strings.TrimSpace(path))
	if err != nil {
		return nil, fmt.Errorf("read telemetry file: %w", err)
	}
	events := make([]*structpb.Struct, 0)
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var event map[string]any
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}
		value, err := structpb.NewStruct(event)
		if err != nil {
			return nil, fmt.Errorf("invalid telemetry event: %w", err)
		}
		events = append(events, value)
	}
	if len(events) == 0 {
		return nil, fmt.Errorf("no valid events found in file")
	}
	return events, nil
}
