package inference

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"connectrpc.com/connect"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	inferenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference/inference_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	client inferenceconnect.InferenceServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClientWithTimeout(core, 0)
	return &handlers{client: inferenceconnect.NewInferenceServiceClient(httpClient, baseURL)}
}

func (h *handlers) run(ctx cliapp.RunContext) error {
	schemaPath := strings.TrimSpace(ctx.Flag("schema"))
	if schemaPath == "" {
		return fmt.Errorf("--schema is required")
	}
	schema, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("read --schema %q: %w", schemaPath, err)
	}
	resp, err := h.client.Run(context.Background(), connect.NewRequest(&inferencev1.RunRequest{
		Source:      ctx.Flag("source"),
		SchemaJson:  string(schema),
		Instruction: ctx.Flag("instruction"),
		Role:        ctx.Flag("role"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("run typed inference", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no inference response")
	}
	usage := resp.Msg.GetUsage()
	results := []string{fmt.Sprintf("validated=%t provider=%s model=%s input_tokens=%d output_tokens=%d cost_micros=%d", resp.Msg.GetValidated(), resp.Msg.GetProvider(), resp.Msg.GetModel(), usage.GetInputTokens(), usage.GetOutputTokens(), usage.GetCostMicros())}
	if value := strings.TrimSpace(resp.Msg.GetValueJson()); value != "" {
		results = append(results, "value="+value)
	}
	if failure := resp.Msg.GetError(); failure != nil {
		results = append(results, fmt.Sprintf("error=%s construct=%s message=%s", failure.GetCode().String(), failure.GetConstruct(), failure.GetMessage()))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Typed inference completed: validated=%t.", resp.Msg.GetValidated())},
		ResultsHeading: "Inference",
		Results:        results,
	})
}

func (h *handlers) runBatch(ctx cliapp.RunContext) error {
	itemsPath := strings.TrimSpace(ctx.Flag("items"))
	if itemsPath == "" {
		return fmt.Errorf("--items is required")
	}
	itemsJSON, err := os.ReadFile(itemsPath)
	if err != nil {
		return fmt.Errorf("read --items %q: %w", itemsPath, err)
	}
	var items []struct {
		Source string `json:"source"`
	}
	if err := json.Unmarshal(itemsJSON, &items); err != nil {
		return fmt.Errorf("decode --items %q: %w", itemsPath, err)
	}
	schemaPath := strings.TrimSpace(ctx.Flag("schema"))
	if schemaPath == "" {
		return fmt.Errorf("--schema is required")
	}
	schema, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("read --schema %q: %w", schemaPath, err)
	}
	requestItems := make([]*inferencev1.RunBatchItem, 0, len(items))
	for _, item := range items {
		requestItems = append(requestItems, &inferencev1.RunBatchItem{Source: item.Source})
	}
	resp, err := h.client.RunBatch(context.Background(), connect.NewRequest(&inferencev1.RunBatchRequest{
		Items: requestItems, SchemaJson: string(schema), Instruction: ctx.Flag("instruction"), Role: ctx.Flag("role"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("run typed inference batch", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no inference batch response")
	}
	results := make([]string, 0, len(resp.Msg.GetResults()))
	for index, result := range resp.Msg.GetResults() {
		usage := result.GetUsage()
		itemResult := fmt.Sprintf("item=%d validated=%t provider=%s model=%s input_tokens=%d output_tokens=%d cost_micros=%d", index, result.GetValidated(), result.GetProvider(), result.GetModel(), usage.GetInputTokens(), usage.GetOutputTokens(), usage.GetCostMicros())
		if value := strings.TrimSpace(result.GetValueJson()); value != "" {
			itemResult += " value=" + value
		}
		if failure := result.GetError(); failure != nil {
			itemResult += fmt.Sprintf(" error=%s construct=%s message=%s", failure.GetCode().String(), failure.GetConstruct(), failure.GetMessage())
		}
		results = append(results, itemResult)
	}
	usage := resp.Msg.GetUsage()
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Typed inference batch completed: items=%d input_tokens=%d output_tokens=%d cost_micros=%d.", len(resp.Msg.GetResults()), usage.GetInputTokens(), usage.GetOutputTokens(), usage.GetCostMicros())},
		ResultsHeading: "Inference batch",
		Results:        results,
	})
}
