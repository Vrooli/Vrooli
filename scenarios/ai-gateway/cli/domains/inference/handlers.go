package inference

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	inferenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference/inference_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
	"google.golang.org/protobuf/proto"

	"github.com/vrooli/cli-core/cliapp"
)

// appliedSummary renders what the gateway actually sent alongside what the role
// declares. Both halves are printed because neither is sufficient alone: a
// temperature that was sent to a provider declaring "ignored" had no effect,
// and a caller reading only the sent value would record provenance that is
// false.
func appliedSummary(applied *sharedv1.AppliedSettings) string {
	if applied == nil {
		return ""
	}
	temperature := "omitted"
	if applied.TemperatureSent != nil {
		temperature = strconv.FormatFloat(applied.GetTemperatureSent(), 'g', -1, 64)
	}
	return fmt.Sprintf("temperature_sent=%s temperature_support=%s max_output_tokens=%d cap_source=%s",
		temperature, applied.GetTemperatureSupport().String(),
		applied.GetMaxOutputTokensEffective(), applied.GetMaxOutputTokensSource().String())
}

type handlers struct {
	client inferenceconnect.InferenceServiceClient
}

func renderRun(ctx cliapp.RunContext, message proto.Message) error {
	response, ok := message.(*inferencev1.RunResponse)
	if !ok {
		return fmt.Errorf("inference run renderer received %T", message)
	}
	usage := response.GetUsage()
	results := []string{fmt.Sprintf("validated=%t provider=%s model=%s input_tokens=%d output_tokens=%d cost_micros=%d", response.GetValidated(), response.GetProvider(), response.GetModel(), usage.GetInputTokens(), usage.GetOutputTokens(), usage.GetCostMicros())}
	if value := strings.TrimSpace(response.GetValueJson()); value != "" {
		results = append(results, "value="+value)
	}
	if applied := appliedSummary(response.GetApplied()); applied != "" {
		results = append(results, "applied="+applied)
	}
	if failure := response.GetError(); failure != nil {
		results = append(results, fmt.Sprintf("error=%s construct=%s message=%s", failure.GetCode().String(), failure.GetConstruct(), failure.GetMessage()))
	}
	return cliapp.RenderProtoList(ctx, response, cliapp.ListReport{Summary: []string{fmt.Sprintf("Typed inference completed: validated=%t.", response.GetValidated())}, ResultsHeading: "Inference", Results: results})
}

func renderRunBatch(ctx cliapp.RunContext, message proto.Message) error {
	response, ok := message.(*inferencev1.RunBatchResponse)
	if !ok {
		return fmt.Errorf("inference batch renderer received %T", message)
	}
	results := make([]string, 0, len(response.GetResults()))
	for index, result := range response.GetResults() {
		usage := result.GetUsage()
		item := fmt.Sprintf("item=%d validated=%t provider=%s model=%s input_tokens=%d output_tokens=%d cost_micros=%d", index, result.GetValidated(), result.GetProvider(), result.GetModel(), usage.GetInputTokens(), usage.GetOutputTokens(), usage.GetCostMicros())
		if value := strings.TrimSpace(result.GetValueJson()); value != "" {
			item += " value=" + value
		}
		if failure := result.GetError(); failure != nil {
			item += fmt.Sprintf(" error=%s construct=%s message=%s", failure.GetCode().String(), failure.GetConstruct(), failure.GetMessage())
		}
		results = append(results, item)
	}
	usage := response.GetUsage()
	return cliapp.RenderProtoList(ctx, response, cliapp.ListReport{Summary: []string{fmt.Sprintf("Typed inference batch completed: items=%d input_tokens=%d output_tokens=%d cost_micros=%d.", len(response.GetResults()), usage.GetInputTokens(), usage.GetOutputTokens(), usage.GetCostMicros())}, ResultsHeading: "Inference batch", Results: results})
}

func renderEmbed(ctx cliapp.RunContext, message proto.Message) error {
	response, ok := message.(*inferencev1.EmbedResponse)
	if !ok {
		return fmt.Errorf("inference embed renderer received %T", message)
	}
	results := []string{fmt.Sprintf("provider=%s model=%s dimension=%d vectors=%d input_tokens=%d cost_micros=%d", response.GetProvider(), response.GetModel(), response.GetDimension(), len(response.GetVectors()), response.GetUsage().GetInputTokens(), response.GetUsage().GetCostMicros())}
	if failure := response.GetError(); failure != nil {
		results = append(results, fmt.Sprintf("error=%s construct=%s message=%s", failure.GetCode().String(), failure.GetConstruct(), failure.GetMessage()))
	}
	return cliapp.RenderProtoList(ctx, response, cliapp.ListReport{Summary: []string{fmt.Sprintf("Embedding completed: vectors=%d dimension=%d.", len(response.GetVectors()), response.GetDimension())}, ResultsHeading: "Embedding", Results: results})
}

func renderDecide(ctx cliapp.RunContext, message proto.Message) error {
	response, ok := message.(*inferencev1.DecideResponse)
	if !ok {
		return fmt.Errorf("inference decide renderer received %T", message)
	}
	return cliapp.RenderProtoList(ctx, response, cliapp.ListReport{Summary: []string{decisionSummary(response)}, ResultsHeading: "Decision", Results: decisionLines(response)})
}

func renderDecideBatch(ctx cliapp.RunContext, message proto.Message) error {
	response, ok := message.(*inferencev1.DecideBatchResponse)
	if !ok {
		return fmt.Errorf("inference decide batch renderer received %T", message)
	}
	var results []string
	answered := 0
	for index, result := range response.GetResults() {
		if result.GetError() == nil {
			answered++
		}
		results = append(results, fmt.Sprintf("item=%d %s", index, decisionSummary(result)))
		for _, line := range decisionLines(result) {
			results = append(results, "  "+line)
		}
	}
	usage := response.GetUsage()
	summary := fmt.Sprintf("Decision batch completed: items=%d answered=%d input_tokens=%d cost_micros=%d.", len(response.GetResults()), answered, usage.GetInputTokens(), usage.GetCostMicros())
	return cliapp.RenderProtoList(ctx, response, cliapp.ListReport{Summary: []string{summary}, ResultsHeading: "Decisions", Results: results})
}

func decisionSummary(response *inferencev1.DecideResponse) string {
	if failure := response.GetError(); failure != nil {
		return fmt.Sprintf("error=%s construct=%s message=%s", failure.GetCode().String(), failure.GetConstruct(), failure.GetMessage())
	}
	return fmt.Sprintf("method=%s provider=%s model=%s role=%s locality=%s latency_ms=%d",
		strings.TrimPrefix(response.GetMethod().String(), "DECISION_METHOD_"), response.GetProvider(), response.GetModel(),
		response.GetResourceRole(), response.GetLocality(), response.GetLatencyMs())
}

// decisionLines renders each answer with its full distribution, then the
// candidate walk, so a reader sees why a hosted candidate was or was not used.
func decisionLines(response *inferencev1.DecideResponse) []string {
	var lines []string
	for _, answer := range response.GetAnswers() {
		line := answer.GetQuestionId() + " "
		switch answer.GetType() {
		case inferencev1.DecisionQuestionType_DECISION_QUESTION_TYPE_CHOICE:
			line += "choice=" + answer.GetChoice()
		case inferencev1.DecisionQuestionType_DECISION_QUESTION_TYPE_SCORE:
			line += "score=" + strconv.FormatFloat(answer.GetScore(), 'f', 4, 64)
		default:
			line += "noul=" + strconv.FormatFloat(answer.GetNoul(), 'f', 4, 64)
		}
		if answer.Confidence != nil {
			line += " confidence=" + strconv.FormatFloat(answer.GetConfidence(), 'f', 4, 64)
		}
		if len(answer.GetProbabilities()) > 0 {
			parts := make([]string, 0, len(answer.GetProbabilities()))
			for _, probability := range answer.GetProbabilities() {
				parts = append(parts, probability.GetOption()+":"+strconv.FormatFloat(probability.GetProbability(), 'f', 4, 64))
			}
			line += " probabilities={" + strings.Join(parts, " ") + "}"
		}
		lines = append(lines, line)
	}
	if usage := response.GetUsage(); usage != nil && response.GetError() == nil {
		lines = append(lines, fmt.Sprintf("usage input_tokens=%d output_tokens=%d cost_micros=%d", usage.GetInputTokens(), usage.GetOutputTokens(), usage.GetCostMicros()))
	}
	for _, candidate := range response.GetCandidates() {
		line := fmt.Sprintf("candidate %s/%s status=%s", candidate.GetProvider(), candidate.GetResourceRole(), strings.TrimPrefix(candidate.GetStatus().String(), "DECISION_CANDIDATE_STATUS_"))
		if candidate.GetReason() != "" {
			line += " reason=" + candidate.GetReason()
		}
		if candidate.GetLatencyMs() > 0 {
			line += fmt.Sprintf(" latency_ms=%d", candidate.GetLatencyMs())
		}
		lines = append(lines, line)
	}
	return lines
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
	request := &inferencev1.RunRequest{
		Source:      ctx.Flag("source"),
		SchemaJson:  string(schema),
		Instruction: ctx.Flag("instruction"),
		Role:        ctx.Flag("role"),
	}
	// An absent flag must stay absent rather than becoming an explicit 0: the
	// role's declared sampling only applies when the caller sends nothing, and
	// 0 is itself a meaningful temperature.
	if raw := strings.TrimSpace(ctx.Flag("temperature")); raw != "" {
		temperature, parseErr := strconv.ParseFloat(raw, 64)
		if parseErr != nil {
			return fmt.Errorf("--temperature %q must be a number: %w", raw, parseErr)
		}
		request.Sampling = &sharedv1.SamplingControls{Temperature: proto.Float64(temperature)}
	}
	if raw := strings.TrimSpace(ctx.Flag("max-output-tokens")); raw != "" {
		maxOutputTokens, parseErr := strconv.ParseInt(raw, 10, 32)
		if parseErr != nil {
			return fmt.Errorf("--max-output-tokens %q must be an integer: %w", raw, parseErr)
		}
		request.MaxOutputTokens = int32(maxOutputTokens)
	}
	resp, err := h.client.Run(context.Background(), connect.NewRequest(request))
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
	if applied := appliedSummary(resp.Msg.GetApplied()); applied != "" {
		results = append(results, "applied="+applied)
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

func (h *handlers) embed(ctx cliapp.RunContext) error {
	path := strings.TrimSpace(ctx.Flag("texts"))
	if path == "" {
		return fmt.Errorf("--texts is required")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read --texts %q: %w", path, err)
	}
	var texts []string
	if err := json.Unmarshal(raw, &texts); err != nil {
		return fmt.Errorf("decode --texts %q: %w", path, err)
	}
	role := strings.TrimSpace(ctx.Flag("role"))
	if role == "" {
		role = "embedding.default"
	}
	resp, err := h.client.Embed(context.Background(), connect.NewRequest(&inferencev1.EmbedRequest{Texts: texts, Role: role}))
	if err != nil {
		return cliapp.WrapAPIError("embed texts", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no embedding response")
	}
	return renderEmbed(ctx, resp.Msg)
}
