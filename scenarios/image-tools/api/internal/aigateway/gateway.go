// Package aigateway is image-tools' remote-image provider. It is the only code
// in the scenario that knows the AI Gateway media RPC: all provider choice,
// credentials, retry policy and cost accounting stay behind that boundary, and
// this package owns the wire format of the execution evidence it returns.
package aigateway

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math"
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
const (
	MetaCostUSD                    = "cost_usd"
	MetaCostProvenance             = "cost_provenance"
	ProviderReportedCostProvenance = "provider_reported_v1"
	MetaExecutionAttempt           = "_gateway_execution_attempt_v1"

	// ParamMaxCostUSD is the engine-set backend parameter carrying the per-image
	// USD ceiling for one cloud generation. Gateway forwards it to the provider
	// owner, which turns it into a bound on what the provider reserves up front.
	ParamMaxCostUSD = "max_cost_usd"
)

type Attempt struct {
	Outcome     string   `json:"outcome"`
	ExecutionID string   `json:"gateway_execution_id,omitempty"`
	Status      string   `json:"gateway_status,omitempty"`
	CostState   string   `json:"cost_state"`
	CostUSD     *float64 `json:"cost_usd,omitempty"`
}

type EvidenceVariation struct {
	Index       int      `json:"index"`
	Outcome     string   `json:"outcome"`
	ExecutionID string   `json:"gateway_execution_id,omitempty"`
	Status      string   `json:"gateway_status,omitempty"`
	CostState   string   `json:"cost_state"`
	CostUSD     *float64 `json:"cost_usd,omitempty"`
}

type Evidence struct {
	SchemaVersion       int                 `json:"schema_version"`
	Complete            bool                `json:"complete"`
	RequestedVariations int                 `json:"requested_variations"`
	Variations          []EvidenceVariation `json:"variations"`
}

// parseCostUSD reads the cost a backend reported. The second return separates
// "reported nothing" from "reported zero", which is the same distinction the
// writer above is careful to preserve.
func ParseCostUSD(meta map[string]string) (float64, bool) {
	if meta[MetaCostProvenance] != ProviderReportedCostProvenance {
		return 0, false
	}
	raw, ok := meta[MetaCostUSD]
	if !ok {
		return 0, false
	}
	cost, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil || math.IsNaN(cost) || math.IsInf(cost, 0) || cost < 0 {
		return 0, false
	}
	return cost, true
}

// mediaGenerationRequest is the provider-neutral request image-tools sends to
// AI Gateway. The output path remains owned by image-tools: Gateway writes only
// to this caller-supplied path for the duration of the request and stores no
// media bytes in its receipt database.
type mediaGenerationRequest struct {
	Operation string
	Role      string
	Prompt    string
	// InputFiles are the images sent with the prompt, in order: the image being
	// edited first, then any reference (a face_swap source face).
	InputFiles []string
	OutputPath string
	// MaxCostUSD is the per-image ceiling for this call; zero means none was set.
	MaxCostUSD float64
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
	CostUSD          float64
	CostKnown        bool
	ExecutionID      string
	GatewayStatus    string
	GatewayErrorCode string
	Outcome          string
}

func attemptMeta(result mediaGenerationResult, outcomeOverride string) map[string]string {
	if result.Outcome == "" && result.ExecutionID == "" {
		return nil
	}
	outcome := result.Outcome
	if outcomeOverride != "" {
		outcome = outcomeOverride
	}
	entry := Attempt{
		Outcome:     outcome,
		ExecutionID: strings.TrimSpace(result.ExecutionID),
		Status:      strings.TrimSpace(result.GatewayStatus),
		CostState:   "unknown",
	}
	if result.CostKnown && outcome != "outcome_unknown" {
		cost := result.CostUSD
		entry.CostState = "known"
		entry.CostUSD = &cost
	}
	encoded, err := json.Marshal(entry)
	if err != nil {
		return nil
	}
	return map[string]string{MetaExecutionAttempt: string(encoded)}
}

// MediaGateway is the narrow remote-media seam; tests substitute a fake.
type MediaGateway interface {
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

func NewMediaClient() MediaGateway {
	return &aiGatewayMediaClient{
		httpClient: &http.Client{Timeout: 10 * time.Minute},
		resolve: func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, aiGatewayScenario)
		},
	}
}

func (c *aiGatewayMediaClient) Generate(ctx context.Context, req mediaGenerationRequest) (mediaGenerationResult, error) {
	result := mediaGenerationResult{Outcome: "failed"}
	if c == nil || c.resolve == nil || c.httpClient == nil {
		return result, fmt.Errorf("ai: AI Gateway media client is not configured")
	}
	baseURL, err := c.resolve(ctx)
	if err != nil {
		return result, fmt.Errorf("ai: resolve AI Gateway: %w", err)
	}
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return result, fmt.Errorf("ai: AI Gateway discovery returned an empty URL")
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
		MaxCostUsd:   req.MaxCostUSD,
	}
	if request.Role == "" || request.Operation == "" {
		return result, fmt.Errorf("ai: AI Gateway media request requires operation and role")
	}

	submit := &routingv1.SubmitMediaRequest{
		Request:         request,
		Prompt:          req.Prompt,
		OutputCount:     1,
		IdempotencyKey:  "image-tools-" + uuid.NewString(),
		OutputReference: req.OutputPath,
	}
	for _, input := range req.InputFiles {
		if input = strings.TrimSpace(input); input != "" {
			submit.Inputs = append(submit.Inputs, &routingv1.MediaInput{Reference: input, MediaType: mediaTypeForFile(input)})
		}
	}

	client := routingconnect.NewRoutingServiceClient(c.httpClient, baseURL)
	submitted, err := client.SubmitMedia(ctx, connect.NewRequest(submit))
	if err != nil {
		result.Outcome = "outcome_unknown"
		return result, fmt.Errorf("ai: submit AI Gateway media request: %w", err)
	}
	if submitted.Msg == nil || submitted.Msg.GetExecution() == nil {
		result.Outcome = "outcome_unknown"
		return result, fmt.Errorf("ai: AI Gateway returned no media execution")
	}
	executionID := submitted.Msg.GetExecution().GetExecutionId()
	if executionID == "" {
		result.Outcome = "outcome_unknown"
		return result, fmt.Errorf("ai: AI Gateway returned an empty media execution id")
	}
	result.ExecutionID = executionID

	// WaitMediaExecution is the durable server-owned wait edge. This is one
	// blocking call, not client-side polling; the gateway owns provider latency,
	// retries, cancellation, and terminal receipt state.
	waited, err := client.WaitMediaExecution(ctx, connect.NewRequest(&routingv1.WaitMediaExecutionRequest{ExecutionId: executionID}))
	if err != nil {
		result.Outcome = "outcome_unknown"
		return result, fmt.Errorf("ai: wait AI Gateway media execution %q: %w", executionID, err)
	}
	if waited.Msg == nil || waited.Msg.GetExecution() == nil {
		result.Outcome = "outcome_unknown"
		return result, fmt.Errorf("ai: AI Gateway returned no terminal media execution")
	}
	execution := waited.Msg.GetExecution()
	result.GatewayStatus = execution.GetStatus().String()
	result.GatewayErrorCode = execution.GetErrorCode()
	if execution.ActualCostUsd != nil && execution.GetErrorCode() != "outcome_unknown" {
		result.CostUSD = execution.GetActualCostUsd()
		result.CostKnown = true
	}
	if execution.GetStatus() != routingv1.MediaExecutionStatus_MEDIA_EXECUTION_STATUS_SUCCEEDED {
		result.Outcome = "failed"
		if execution.GetErrorCode() == "outcome_unknown" {
			result.Outcome = "outcome_unknown"
			result.CostUSD = 0
			result.CostKnown = false
		}
		detail := strings.TrimSpace(execution.GetErrorMessage())
		if detail == "" {
			detail = execution.GetStatus().String()
		}
		return result, fmt.Errorf("ai: AI Gateway media execution %q ended in %s: %s", executionID, execution.GetStatus().String(), detail)
	}
	result.Outcome = "succeeded"
	if _, err := os.Stat(req.OutputPath); err != nil {
		return result, fmt.Errorf("ai: AI Gateway succeeded without writing %q: %w", req.OutputPath, err)
	}

	mediaType := ""
	for _, output := range execution.GetOutputs() {
		if output.GetReference() == req.OutputPath || output.GetReference() == "" {
			mediaType = strings.TrimSpace(output.GetMediaType())
			break
		}
	}
	result.MediaType = mediaType
	result.Model = strings.TrimSpace(execution.GetResolvedModel())
	result.Warnings = append([]string(nil), execution.GetWarnings()...)
	return result, nil
}

// aiGatewayProvider is image-tools' final BYOK rung. It retains the existing
// model/backend vocabulary for selection and telemetry, while all remote
// transport and credential/model policy are delegated to AI Gateway.
type aiGatewayProvider struct {
	media  MediaGateway
	vision VisionGateway
}

// NewProvider builds the hosted rung. media serves its image operations and
// vision serves the general vision models (caption, object detection, tagging).
func NewProvider(media MediaGateway, vision VisionGateway) backends.Provider {
	return &aiGatewayProvider{media: media, vision: vision}
}

func (p *aiGatewayProvider) Name() string { return models.BackendOpenRouter }
func (p *aiGatewayProvider) Operations() []string {
	ops := []string{"text_to_image", "image_to_image", "edit_instruct", "face_swap", "colorize"}
	if p != nil && p.vision != nil {
		ops = append(ops, models.VisionOperations...)
	}
	return ops
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

// maxCostForRequest reads the engine-set per-image ceiling. An absent, invalid
// or non-positive value means no ceiling, never a zero-dollar one.
func maxCostForRequest(req backends.Request) float64 {
	cost, err := strconv.ParseFloat(strings.TrimSpace(req.Params[ParamMaxCostUSD]), 64)
	if err != nil || math.IsNaN(cost) || math.IsInf(cost, 0) || cost <= 0 {
		return 0
	}
	return cost
}

// FaceSwapInstruction is the edit instruction a cloud face swap sends when the
// caller gave none (see technique.FaceSwapInstruction).
const FaceSwapInstruction = technique.FaceSwapInstruction

func roleForRequest(req backends.Request) string {
	if role := strings.TrimSpace(req.Params["openrouter_role"]); role != "" {
		return role
	}
	if req.Operation == "text_to_image" {
		return "image.generate.default"
	}
	if req.Operation == "face_swap" || req.Operation == "colorize" {
		// Both keep the subject recognisable: a swap changes who is shown, a
		// colourisation must not change anything but colour.
		return "image.edit.identity"
	}
	return "image.edit.default"
}

func (p *aiGatewayProvider) Execute(ctx context.Context, req backends.Request) (backends.Result, error) {
	if models.IsVisionOperation(req.Operation) {
		return executeVision(ctx, p.vision, models.BackendOpenRouter, false, req)
	}
	if req.Output.LocalPath == "" {
		return backends.Result{}, fmt.Errorf("ai: AI Gateway image backend requires a local output path")
	}
	prompt := strings.TrimSpace(req.Params["prompt"])
	switch {
	case prompt != "":
	case req.Operation == "face_swap":
		prompt = FaceSwapInstruction
	case req.Operation == "colorize":
		prompt = technique.ColorizeInstruction
	}
	if prompt == "" {
		return backends.Result{}, fmt.Errorf("ai: AI Gateway image generation requires a prompt")
	}
	if p == nil || p.media == nil {
		return backends.Result{}, fmt.Errorf("ai: AI Gateway media client is not configured")
	}
	var inputFiles []string
	if req.Operation != "text_to_image" {
		source, err := technique.Input0(req)
		if err != nil {
			return backends.Result{}, fmt.Errorf("ai: AI Gateway source image: %w", err)
		}
		inputFiles = append(inputFiles, source)
		if req.Operation == "face_swap" {
			reference, err := technique.ReferencePath(req)
			if err != nil {
				return backends.Result{}, fmt.Errorf("ai: AI Gateway face reference: %w", err)
			}
			inputFiles = append(inputFiles, reference)
		}
	}
	result, err := p.media.Generate(ctx, mediaGenerationRequest{
		Operation:  req.Operation,
		Role:       roleForRequest(req),
		Prompt:     prompt,
		InputFiles: inputFiles,
		OutputPath: req.Output.LocalPath,
		MaxCostUSD: maxCostForRequest(req),
	})
	if err != nil {
		outcomeOverride := ""
		if result.Outcome == "succeeded" {
			outcomeOverride = "failed"
		}
		return backends.Result{Meta: attemptMeta(result, outcomeOverride)}, fmt.Errorf("ai: AI Gateway image generation: %w", err)
	}
	if info, statErr := os.Stat(req.Output.LocalPath); statErr != nil || info.Size() == 0 {
		if statErr == nil {
			statErr = io.ErrUnexpectedEOF
		}
		return backends.Result{Meta: attemptMeta(result, "failed")}, fmt.Errorf("ai: AI Gateway output is empty: %w", statErr)
	}
	if req.Operation == "colorize" {
		// A colourisation keeps the picture and changes its colour. A general editor
		// that answers with a picture of another shape has replaced the photo (it
		// returned a different woman in a different scene for a portrait), so the
		// result is refused rather than delivered as a colourised copy.
		if err := sameShape(inputFiles[0], req.Output.LocalPath); err != nil {
			return backends.Result{Meta: attemptMeta(result, "failed")}, fmt.Errorf("ai: AI Gateway colorize: %w", err)
		}
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
	if result.CostKnown {
		meta[MetaCostUSD] = strconv.FormatFloat(result.CostUSD, 'f', -1, 64)
		meta[MetaCostProvenance] = ProviderReportedCostProvenance
	}
	if len(result.Warnings) > 0 {
		meta["warnings"] = strings.Join(result.Warnings, "; ")
	}
	for key, value := range attemptMeta(result, "succeeded") {
		meta[key] = value
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

// shapeTolerance is how far the output's aspect ratio may differ from the
// input's. Providers round dimensions to their own grid (768x1024 comes back
// 896x1200, a 0.4% difference); a different orientation is a different picture.
const shapeTolerance = 0.03

// sameShape checks that the output has the input's aspect ratio.
func sameShape(inputPath, outputPath string) error {
	in, err := imageSize(inputPath)
	if err != nil {
		return fmt.Errorf("read the source size: %w", err)
	}
	out, err := imageSize(outputPath)
	if err != nil {
		return fmt.Errorf("read the result size: %w", err)
	}
	inRatio, outRatio := float64(in[0])/float64(in[1]), float64(out[0])/float64(out[1])
	if diff := math.Abs(outRatio-inRatio) / inRatio; diff > shapeTolerance {
		return fmt.Errorf("the provider returned a %dx%d picture for a %dx%d photo, so it did not edit the photo you sent; no result is kept", out[0], out[1], in[0], in[1])
	}
	return nil
}

func imageSize(path string) ([2]int, error) {
	f, err := os.Open(path)
	if err != nil {
		return [2]int{}, err
	}
	defer func() { _ = f.Close() }()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return [2]int{}, err
	}
	return [2]int{cfg.Width, cfg.Height}, nil
}
