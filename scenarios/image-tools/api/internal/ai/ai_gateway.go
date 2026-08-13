package ai

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"image-tools/internal/backends"
	"image-tools/internal/models"
	"image-tools/internal/technique"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/vrooli/api-core/discovery"

	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"
	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing/routing_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

const aiGatewayScenario = "ai-gateway"

// MetaCostUSD is the backend-result metadata key carrying what a route cost.
// It is a named constant because the producer here and the consumer on the
// submit edge are in different packages, and a string literal in two places is
// a contract nobody can see.
const MetaCostUSD = "cost_usd"

// mediaGenerationRequest is the provider-neutral request image-tools sends to
// AI Gateway. The output path remains owned by image-tools: Gateway writes only
// to this caller-supplied path for the duration of the request and stores no
// media bytes in its receipt database.
type mediaGenerationRequest struct {
	Operation  string
	Role       string
	Prompt     string
	InputFile  string
	OutputPath string
}

type mediaGenerationResult struct {
	MediaType string
	Model     string
	Warnings  []string
	// CostUSD is what Gateway's receipt says the call actually cost.
	//
	// It was being dropped one stack frame after it arrived. Gateway records it
	// on the terminal MediaExecution because it is the only party that knows,
	// and a caller deciding whether a catalog is affordable to render has
	// nowhere else to look — so a consumer that discards it forces every
	// downstream cost question to be answered by guessing. Zero is a genuine
	// measurement for a route that cost nothing, not a missing value.
	CostUSD float64
}

type mediaGateway interface {
	Generate(context.Context, mediaGenerationRequest) (mediaGenerationResult, error)
}

// aiGatewayMediaClient is the only image-tools implementation that knows the
// AI Gateway media RPC. It discovers the gateway at call time and uses the
// generated Connect client; image-tools has no provider URL, credential, or
// concrete remote model configuration.
type aiGatewayMediaClient struct {
	httpClient *http.Client
	resolve    func(context.Context) (string, error)
}

func newAIGatewayMediaClient() *aiGatewayMediaClient {
	return &aiGatewayMediaClient{
		httpClient: &http.Client{Timeout: 10 * time.Minute},
		resolve: func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, aiGatewayScenario)
		},
	}
}

func newAIGatewayMediaClientWithResolver(resolve func(context.Context) (string, error), httpClient *http.Client) *aiGatewayMediaClient {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Minute}
	}
	return &aiGatewayMediaClient{httpClient: httpClient, resolve: resolve}
}

func (c *aiGatewayMediaClient) Generate(ctx context.Context, req mediaGenerationRequest) (mediaGenerationResult, error) {
	if c == nil || c.resolve == nil || c.httpClient == nil {
		return mediaGenerationResult{}, fmt.Errorf("ai: AI Gateway media client is not configured")
	}
	baseURL, err := c.resolve(ctx)
	if err != nil {
		return mediaGenerationResult{}, fmt.Errorf("ai: resolve AI Gateway: %w", err)
	}
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return mediaGenerationResult{}, fmt.Errorf("ai: AI Gateway discovery returned an empty URL")
	}

	request := &sharedv1.GatewayRequest{
		Kind:         sharedv1.RequestKind_REQUEST_KIND_IMAGE_GENERATION,
		Role:         strings.TrimSpace(req.Role),
		Profile:      sharedv1.Profile_PROFILE_REMOTE_ONLY,
		PrivacyClass: sharedv1.PrivacyClass_PRIVACY_CLASS_INTERNAL,
		Operation:    strings.TrimSpace(req.Operation),
		Scenario:     "image-tools",
		TimeoutMs:    10 * 60 * 1000,
		RequestId:    uuid.NewString(),
	}
	if request.Role == "" || request.Operation == "" {
		return mediaGenerationResult{}, fmt.Errorf("ai: AI Gateway media request requires operation and role")
	}

	submit := &routingv1.SubmitMediaRequest{
		Request:         request,
		Prompt:          req.Prompt,
		OutputCount:     1,
		IdempotencyKey:  "image-tools-" + uuid.NewString(),
		OutputReference: req.OutputPath,
	}
	if input := strings.TrimSpace(req.InputFile); input != "" {
		submit.Inputs = []*routingv1.MediaInput{{
			Reference: input,
			MediaType: mediaTypeForFile(input),
		}}
	}

	client := routingconnect.NewRoutingServiceClient(c.httpClient, baseURL)
	submitted, err := client.SubmitMedia(ctx, connect.NewRequest(submit))
	if err != nil {
		return mediaGenerationResult{}, fmt.Errorf("ai: submit AI Gateway media request: %w", err)
	}
	if submitted.Msg == nil || submitted.Msg.GetExecution() == nil {
		return mediaGenerationResult{}, fmt.Errorf("ai: AI Gateway returned no media execution")
	}
	executionID := submitted.Msg.GetExecution().GetExecutionId()
	if executionID == "" {
		return mediaGenerationResult{}, fmt.Errorf("ai: AI Gateway returned an empty media execution id")
	}

	// WaitMediaExecution is the durable server-owned wait edge. This is one
	// blocking call, not client-side polling; the gateway owns provider latency,
	// retries, cancellation, and terminal receipt state.
	waited, err := client.WaitMediaExecution(ctx, connect.NewRequest(&routingv1.WaitMediaExecutionRequest{ExecutionId: executionID}))
	if err != nil {
		return mediaGenerationResult{}, fmt.Errorf("ai: wait AI Gateway media execution %q: %w", executionID, err)
	}
	if waited.Msg == nil || waited.Msg.GetExecution() == nil {
		return mediaGenerationResult{}, fmt.Errorf("ai: AI Gateway returned no terminal media execution")
	}
	execution := waited.Msg.GetExecution()
	if execution.GetStatus() != routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_SUCCEEDED {
		detail := strings.TrimSpace(execution.GetErrorMessage())
		if detail == "" {
			detail = execution.GetStatus().String()
		}
		return mediaGenerationResult{}, fmt.Errorf("ai: AI Gateway media execution %q ended in %s: %s", executionID, execution.GetStatus().String(), detail)
	}
	if _, err := os.Stat(req.OutputPath); err != nil {
		return mediaGenerationResult{}, fmt.Errorf("ai: AI Gateway succeeded without writing %q: %w", req.OutputPath, err)
	}

	mediaType := ""
	for _, output := range execution.GetOutputs() {
		if output.GetReference() == req.OutputPath || output.GetReference() == "" {
			mediaType = strings.TrimSpace(output.GetMediaType())
			break
		}
	}
	return mediaGenerationResult{
		MediaType: mediaType,
		Model:     strings.TrimSpace(execution.GetResolvedModel()),
		Warnings:  append([]string(nil), execution.GetWarnings()...),
		CostUSD:   execution.GetActualCostUsd(),
	}, nil
}

// aiGatewayProvider is image-tools' final BYOK rung. It retains the existing
// model/backend vocabulary for selection and telemetry, while all remote
// transport and credential/model policy are delegated to AI Gateway.
type aiGatewayProvider struct {
	media mediaGateway
}

func newAIGatewayProvider(media mediaGateway) *aiGatewayProvider {
	return &aiGatewayProvider{media: media}
}

func (p *aiGatewayProvider) Name() string { return models.BackendOpenRouter }
func (p *aiGatewayProvider) Operations() []string {
	return []string{"text_to_image", "image_to_image", "edit_instruct"}
}
func (p *aiGatewayProvider) Standalone() bool { return true }
func (p *aiGatewayProvider) IsCloud() bool    { return true }
func (p *aiGatewayProvider) GPUCapable() bool { return false }
func (p *aiGatewayProvider) Available(context.Context) bool {
	return p != nil && p.media != nil
}

func (p *aiGatewayProvider) Availability(ctx context.Context) backends.Availability {
	if p.Available(ctx) {
		return backends.Availability{Available: true, Detail: "AI Gateway remote image generation", Provision: "configure the provider credential through Vrooli credentials"}
	}
	return backends.Availability{Available: false, Detail: "AI Gateway media client is not configured", Provision: "start ai-gateway and configure its resource credential"}
}

func roleForRequest(req backends.Request) string {
	if role := strings.TrimSpace(req.Params["openrouter_role"]); role != "" {
		return role
	}
	if req.Operation == "text_to_image" {
		return "image.generate.default"
	}
	return "image.edit.default"
}

func (p *aiGatewayProvider) Execute(ctx context.Context, req backends.Request) (backends.Result, error) {
	if req.Output.LocalPath == "" {
		return backends.Result{}, fmt.Errorf("ai: AI Gateway image backend requires a local output path")
	}
	prompt := strings.TrimSpace(req.Params["prompt"])
	if prompt == "" {
		return backends.Result{}, fmt.Errorf("ai: AI Gateway image generation requires a prompt")
	}
	if p == nil || p.media == nil {
		return backends.Result{}, fmt.Errorf("ai: AI Gateway media client is not configured")
	}
	inputFile := ""
	if req.Operation != "text_to_image" {
		var err error
		inputFile, err = technique.Input0(req)
		if err != nil {
			return backends.Result{}, fmt.Errorf("ai: AI Gateway source image: %w", err)
		}
	}
	result, err := p.media.Generate(ctx, mediaGenerationRequest{
		Operation:  req.Operation,
		Role:       roleForRequest(req),
		Prompt:     prompt,
		InputFile:  inputFile,
		OutputPath: req.Output.LocalPath,
	})
	if err != nil {
		return backends.Result{}, fmt.Errorf("ai: AI Gateway image generation: %w", err)
	}
	if info, statErr := os.Stat(req.Output.LocalPath); statErr != nil || info.Size() == 0 {
		if statErr == nil {
			statErr = io.ErrUnexpectedEOF
		}
		return backends.Result{}, fmt.Errorf("ai: AI Gateway output is empty: %w", statErr)
	}
	meta := map[string]string{
		"backend": models.BackendOpenRouter,
		"gateway": "ai-gateway",
		"role":    roleForRequest(req),
	}
	if result.Model != "" {
		meta["model"] = result.Model
	}
	if result.MediaType != "" {
		meta["media_type"] = result.MediaType
	}
	// Always recorded, including when it is zero: "this route cost nothing" and
	// "nobody asked what this route cost" are different facts, and a caller
	// deciding whether to escalate needs to tell them apart.
	meta[MetaCostUSD] = strconv.FormatFloat(result.CostUSD, 'f', -1, 64)
	if len(result.Warnings) > 0 {
		meta["warnings"] = strings.Join(result.Warnings, "; ")
	}
	return backends.Result{OutputRef: req.Output.LocalPath, Tier: backends.TierBYOK, Meta: meta}, nil
}

func mediaTypeForFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return "application/octet-stream"
	}
	trimmed := strings.TrimSpace(string(data[:min(len(data), 256)]))
	if strings.HasPrefix(trimmed, "<svg") || (strings.HasPrefix(trimmed, "<?xml") && strings.Contains(trimmed, "<svg")) {
		return "image/svg+xml"
	}
	return http.DetectContentType(data)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
