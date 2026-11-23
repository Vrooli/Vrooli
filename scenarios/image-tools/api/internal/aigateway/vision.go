package aigateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image/jpeg"
	"io"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/disintegration/imaging"
	"github.com/google/uuid"
	"github.com/vrooli/api-core/discovery"

	"image-tools/internal/backends"
	"image-tools/internal/models"
	"image-tools/internal/technique"

	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"
	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing/routing_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

// A general vision-language model answers a question about the image in words.
// This file is the whole of image-tools' knowledge of that: one prompt per
// operation, one parser per answer shape, and the gateway call. Which model
// answers is not here. The registry row names a gateway role (runtime.gateway_role)
// and the gateway's resource policy maps the role to an Ollama model on this
// machine or a hosted one, so a new vision model is a row, not code.

// visionMaxImageEdge bounds the long side of the image sent to the model. Vision
// models resize internally; sending a 24 MP original costs bandwidth and, on a
// hosted route, money for nothing. Boxes come back normalised, so they are
// scaled by the original size and the downscale never shifts them.
const visionMaxImageEdge = 1536

// visionPrompts is the question asked for each vision operation.
var visionPrompts = map[string]string{
	"caption":          "Describe this image in one or two sentences. Name the main subject, its colours, the setting and any readable text. Output only the description.",
	"tagging":          `List 15 short descriptive tags for this image: subject, colours, setting, objects, mood. Output only a JSON array of strings.`,
	"object_detection": `Detect every distinct object you can see. Output only a JSON array of {"label": "<object>", "bbox_2d": [x1, y1, x2, y2]} with one entry per object and coordinates on a 0 to 1000 grid measured from the top-left corner of the image.`,
}

// visionTokenBudget is the output cap per operation. A model that thinks before
// it answers spends part of it on thinking, so detection, which lists many boxes,
// gets the most.
var visionTokenBudget = map[string]int32{"caption": 1500, "tagging": 1500, "object_detection": 6000}

// visionTimeout is how long one call may take. A cold local model must load into
// GPU memory first, which on a shared GPU can take minutes.
const visionTimeout = 10 * time.Minute

// VisionGateway is the narrow seam for asking a gateway vision role a question
// about one image; tests substitute a fake.
type VisionGateway interface {
	Ask(context.Context, VisionRequest) (VisionAnswer, error)
	// Reachable reports whether the gateway can be called at all.
	Reachable(context.Context) error
}

// VisionRequest is one question about one image.
type VisionRequest struct {
	Operation string
	Role      string
	// Local keeps the call on this machine (the gateway's local-only profile);
	// otherwise it runs on a hosted model (remote-only).
	Local           bool
	Prompt          string
	Image           []byte
	MediaType       string
	MaxOutputTokens int32
	// MaxCostUSD is the per-call spend ceiling for a hosted route; zero means none.
	MaxCostUSD float64
}

// VisionAnswer is the model's reply and what the gateway recorded about the call.
type VisionAnswer struct {
	Text        string
	Model       string
	Provider    string
	EventID     string
	Status      string
	LatencyMS   int64
	CostUSD     float64
	CostKnown   bool
	InputTokens int64
	OutputToken int64
	Capacity    string
}

type visionClient struct {
	httpClient *http.Client
	resolve    func(context.Context) (string, error)
}

// NewVisionClient builds the AI Gateway client for vision questions. It
// discovers the gateway at call time, like the media client.
func NewVisionClient() VisionGateway {
	return &visionClient{
		httpClient: &http.Client{Timeout: visionTimeout + time.Minute},
		resolve: func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, aiGatewayScenario)
		},
	}
}

func (c *visionClient) baseURL(ctx context.Context) (string, error) {
	if c == nil || c.resolve == nil || c.httpClient == nil {
		return "", fmt.Errorf("ai: AI Gateway vision client is not configured")
	}
	base, err := c.resolve(ctx)
	if err != nil {
		return "", fmt.Errorf("ai: resolve AI Gateway: %w", err)
	}
	if base = strings.TrimRight(strings.TrimSpace(base), "/"); base == "" {
		return "", fmt.Errorf("ai: AI Gateway discovery returned an empty URL")
	}
	return base, nil
}

func (c *visionClient) Reachable(ctx context.Context) error {
	_, err := c.baseURL(ctx)
	return err
}

func (c *visionClient) Ask(ctx context.Context, req VisionRequest) (VisionAnswer, error) {
	var answer VisionAnswer
	base, err := c.baseURL(ctx)
	if err != nil {
		return answer, err
	}
	if strings.TrimSpace(req.Role) == "" || len(req.Image) == 0 {
		return answer, fmt.Errorf("ai: a vision request needs a gateway role and an image")
	}
	profile, privacy := sharedv1.Profile_PROFILE_REMOTE_ONLY, sharedv1.PrivacyClass_PRIVACY_CLASS_INTERNAL
	if req.Local {
		// A local route never leaves the machine, so the strictest class is safe.
		profile, privacy = sharedv1.Profile_PROFILE_LOCAL_ONLY, sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL
	}
	gateway := &sharedv1.GatewayRequest{
		Kind:              sharedv1.RequestKind_REQUEST_KIND_TEXT_GENERATION,
		Role:              strings.TrimSpace(req.Role),
		Profile:           profile,
		PrivacyClass:      privacy,
		PrivacyProvenance: "image-tools.user_image_vision_question",
		Operation:         req.Operation,
		Scenario:          "image-tools",
		TimeoutMs:         int32(visionTimeout / time.Millisecond),
		MaxCostUsd:        req.MaxCostUSD,
		MaxOutputTokens:   req.MaxOutputTokens,
		RequestId:         uuid.NewString(),
		Attachments: []*sharedv1.Attachment{{
			Modality:  sharedv1.Modality_MODALITY_IMAGE,
			MediaType: req.MediaType,
			Bytes:     uint64(len(req.Image)),
			Payload:   &sharedv1.Attachment_InlineBytes{InlineBytes: req.Image},
		}},
	}
	client := routingconnect.NewRoutingServiceClient(c.httpClient, base)
	resp, err := client.ExecuteRoute(ctx, connect.NewRequest(&routingv1.ExecuteRouteRequest{Request: gateway, InputText: req.Prompt}))
	if err != nil {
		return answer, fmt.Errorf("ai: AI Gateway vision call: %w", err)
	}
	msg := resp.Msg
	evidence := msg.GetEvidence()
	answer = VisionAnswer{
		Text:        msg.GetOutputText(),
		Model:       evidence.GetSelectedModel(),
		Provider:    evidence.GetSelectedProvider(),
		EventID:     evidence.GetEventId(),
		Status:      evidence.GetStatus(),
		LatencyMS:   evidence.GetLatencyMs(),
		InputTokens: evidence.GetInputTokens(),
		OutputToken: evidence.GetOutputTokens(),
		Capacity:    evidence.GetCapacityVerdict(),
	}
	if cost := evidence.GetCostEstimate(); cost > 0 && !req.Local {
		answer.CostUSD, answer.CostKnown = cost, true
	}
	if !msg.GetValid() {
		return answer, fmt.Errorf("ai: AI Gateway refused the vision request: %s", describeIssues(msg.GetIssues()))
	}
	if evidence.GetStatus() != "succeeded" {
		reasons := append(append([]string{}, evidence.GetFailureReasons()...), evidence.GetRejectionReason())
		return answer, fmt.Errorf("ai: AI Gateway vision route %q ended %q (%s)", req.Role, evidence.GetStatus(), strings.Join(nonEmpty(reasons), ", "))
	}
	return answer, nil
}

func describeIssues(issues []*sharedv1.ValidationIssue) string {
	parts := make([]string, 0, len(issues))
	for _, issue := range issues {
		parts = append(parts, issue.GetField()+": "+issue.GetMessage())
	}
	return strings.Join(parts, "; ")
}

func nonEmpty(in []string) []string {
	out := in[:0:0]
	for _, s := range in {
		if strings.TrimSpace(s) != "" {
			out = append(out, s)
		}
	}
	return out
}

// executeVision answers one vision operation through the gateway and writes the
// operation's output JSON. local says whether the row's route stays on this
// machine. It is shared by the local provider and the hosted one, so the two
// differ only in the profile they ask the gateway for.
func executeVision(ctx context.Context, gateway VisionGateway, backend string, local bool, req backends.Request) (backends.Result, error) {
	if req.Output.LocalPath == "" {
		return backends.Result{}, fmt.Errorf("ai: backend %q requires a local output path", backend)
	}
	if gateway == nil {
		return backends.Result{}, fmt.Errorf("ai: AI Gateway vision client is not configured")
	}
	prompt, ok := visionPrompts[req.Operation]
	if !ok {
		return backends.Result{}, fmt.Errorf("ai: backend %q does not serve %q", backend, req.Operation)
	}
	role := strings.TrimSpace(req.Model.Runtime.GatewayRole)
	if role == "" {
		return backends.Result{}, fmt.Errorf("ai: model %q has no runtime.gateway_role", req.Model.ID)
	}
	if custom := strings.TrimSpace(req.Params["prompt"]); custom != "" && req.Operation == "caption" {
		prompt = custom
	}
	source, err := technique.Input0(req)
	if err != nil {
		return backends.Result{}, fmt.Errorf("ai: vision source image: %w", err)
	}
	image, width, height, mediaType, err := prepareVisionImage(source)
	if err != nil {
		return backends.Result{}, err
	}
	answer, err := gateway.Ask(ctx, VisionRequest{
		Operation: req.Operation, Role: role, Local: local, Prompt: prompt, Image: image, MediaType: mediaType,
		MaxOutputTokens: visionTokenBudget[req.Operation], MaxCostUSD: maxCostForRequest(req),
	})
	attempt := visionAttempt(answer, err == nil)
	if err != nil {
		return backends.Result{Meta: attempt}, err
	}
	text := completionText(answer.Text)
	payload, err := visionPayload(req.Operation, backend, firstNonEmpty(answer.Model, req.Model.ID), text, req.Model.Runtime, width, height)
	if err != nil {
		return backends.Result{Meta: attempt}, fmt.Errorf("ai: %s answer from role %q is unusable: %w", req.Operation, role, err)
	}
	if err := os.WriteFile(req.Output.LocalPath, payload, 0o644); err != nil {
		return backends.Result{Meta: attempt}, fmt.Errorf("ai: write %s output: %w", req.Operation, err)
	}
	meta := map[string]string{
		"backend":            backend,
		"gateway":            "ai-gateway",
		"role":               role,
		"gateway_provider":   answer.Provider,
		"gateway_latency_ms": strconv.FormatInt(answer.LatencyMS, 10),
	}
	if answer.Model != "" {
		meta["model"] = answer.Model
	}
	if answer.Capacity != "" {
		meta["gateway_capacity_verdict"] = answer.Capacity
	}
	if answer.InputTokens > 0 || answer.OutputToken > 0 {
		meta["gateway_tokens"] = strconv.FormatInt(answer.InputTokens, 10) + "+" + strconv.FormatInt(answer.OutputToken, 10)
	}
	for k, v := range attempt {
		meta[k] = v
	}
	if answer.CostKnown {
		meta[MetaCostUSD] = strconv.FormatFloat(answer.CostUSD, 'f', -1, 64)
		meta[MetaCostProvenance] = ProviderReportedCostProvenance
	}
	tier := backends.TierBYOK
	if local {
		tier = backends.TierGatewayLocal
	}
	return backends.Result{OutputRef: req.Output.LocalPath, Tier: tier, Meta: meta}, nil
}

// visionAttempt is the per-call execution evidence the job's cost ledger reads.
func visionAttempt(answer VisionAnswer, ok bool) map[string]string {
	if answer.EventID == "" && answer.Status == "" {
		return nil
	}
	outcome := "failed"
	if ok {
		outcome = "succeeded"
	}
	return attemptMeta(mediaGenerationResult{
		Outcome: outcome, ExecutionID: answer.EventID, GatewayStatus: answer.Status,
		CostUSD: answer.CostUSD, CostKnown: answer.CostKnown,
	}, "")
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// prepareVisionImage reads the source, applies its EXIF orientation, bounds its
// size and returns the bytes to send plus the oriented original's dimensions.
// A small image already in a format the models take is sent untouched.
func prepareVisionImage(path string) (data []byte, width, height int, mediaType string, err error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, 0, 0, "", fmt.Errorf("ai: read vision source: %w", err)
	}
	decoded, err := imaging.Decode(bytes.NewReader(raw), imaging.AutoOrientation(true))
	if err != nil {
		return nil, 0, 0, "", fmt.Errorf("ai: decode vision source: %w", err)
	}
	bounds := decoded.Bounds()
	width, height = bounds.Dx(), bounds.Dy()
	sent := decoded
	if max(width, height) > visionMaxImageEdge {
		if width >= height {
			sent = imaging.Resize(decoded, visionMaxImageEdge, 0, imaging.Lanczos)
		} else {
			sent = imaging.Resize(decoded, 0, visionMaxImageEdge, imaging.Lanczos)
		}
	}
	var out bytes.Buffer
	if err := jpeg.Encode(&out, sent, &jpeg.Options{Quality: 92}); err != nil {
		return nil, 0, 0, "", fmt.Errorf("ai: encode vision image: %w", err)
	}
	return out.Bytes(), width, height, "image/jpeg", nil
}

// Detection is one labelled box in source pixels, the shape the Florence-2
// operation already reports.
type Detection struct {
	Label string     `json:"label"`
	Box   [4]float64 `json:"box"`
}

// visionPayload turns the model's text into the operation's output JSON.
func visionPayload(op, backend, model, text string, rt models.Runtime, width, height int) ([]byte, error) {
	var payload any
	switch op {
	case "caption":
		caption := strings.TrimSpace(stripThinking(text))
		if caption == "" {
			return nil, fmt.Errorf("the model returned an empty caption")
		}
		payload = map[string]any{"backend": backend, "model": model, "caption": caption}
	case "tagging":
		tags := ParseTags(text)
		if len(tags) == 0 {
			return nil, fmt.Errorf("no tags found in %q", clip(text))
		}
		items := make([]map[string]string, len(tags))
		for i, tag := range tags {
			items[i] = map[string]string{"tag": tag}
		}
		payload = map[string]any{"backend": backend, "model": model, "tags": items, "count": len(items)}
	case "object_detection":
		found, err := ParseDetections(text, rt.BoxOrder, rt.BoxScale, width, height)
		if err != nil {
			return nil, err
		}
		payload = map[string]any{"backend": backend, "model": model, "objects": found, "count": len(found)}
	default:
		return nil, fmt.Errorf("unsupported vision operation %q", op)
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func clip(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 160 {
		return s[:160] + "..."
	}
	return s
}

// completionText unwraps the reply the gateway returns as a string. A hosted
// route returns an OpenAI-style chat completion, a local Ollama route returns
// Ollama's generate object, and a role served some other way returns plain text.
func completionText(output string) string {
	trimmed := strings.TrimSpace(output)
	if !strings.HasPrefix(trimmed, "{") {
		return trimmed
	}
	var chat struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Response string `json:"response"`
	}
	if json.Unmarshal([]byte(trimmed), &chat) != nil {
		return trimmed
	}
	if len(chat.Choices) > 0 && strings.TrimSpace(chat.Choices[0].Message.Content) != "" {
		return chat.Choices[0].Message.Content
	}
	if strings.TrimSpace(chat.Response) != "" {
		return chat.Response
	}
	return trimmed
}

// stripThinking drops a leading <think>...</think> block some local models emit.
func stripThinking(text string) string {
	if end := strings.Index(text, "</think>"); end >= 0 {
		return text[end+len("</think>"):]
	}
	return text
}

// extractJSON finds the first complete JSON array or object in text, ignoring a
// Markdown fence and any prose around it.
func extractJSON(text string) (json.RawMessage, error) {
	text = stripThinking(text)
	start := strings.IndexAny(text, "[{")
	if start < 0 {
		return nil, fmt.Errorf("no JSON in %q", clip(text))
	}
	decoder := json.NewDecoder(strings.NewReader(text[start:]))
	var raw json.RawMessage
	if err := decoder.Decode(&raw); err != nil && err != io.EOF {
		return nil, fmt.Errorf("answer is not valid JSON (%v): %q", err, clip(text))
	}
	return raw, nil
}

// ParseTags reads a tag list: a JSON array of strings or of {"tag": ...} objects,
// or, failing that, comma or line separated words.
func ParseTags(text string) []string {
	var tags []string
	if raw, err := extractJSON(text); err == nil {
		var strs []string
		if json.Unmarshal(raw, &strs) == nil {
			tags = strs
		} else {
			var objs []map[string]any
			if json.Unmarshal(raw, &objs) == nil {
				for _, o := range objs {
					for _, key := range []string{"tag", "label", "name"} {
						if s, ok := o[key].(string); ok {
							tags = append(tags, s)
							break
						}
					}
				}
			}
		}
	}
	if len(tags) == 0 && !strings.ContainsAny(text, "[{") {
		tags = strings.FieldsFunc(stripThinking(text), func(r rune) bool { return r == ',' || r == '\n' })
	}
	seen := map[string]bool{}
	out := tags[:0:0]
	for _, tag := range tags {
		tag = strings.ToLower(strings.Trim(strings.TrimSpace(tag), "-*•\"'` "))
		if tag != "" && !seen[tag] {
			seen[tag] = true
			out = append(out, tag)
		}
	}
	return out
}

// boxKeys are the field names vision models use for a box.
var boxKeys = []string{"bbox_2d", "box_2d", "bbox", "box", "bounding_box"}

// ParseDetections reads a model's detection answer into boxes in source pixels.
// order ("xyxy" or "yxyx") and scale (a 0..scale grid, or 0 for pixels) come from
// the model's registry row, because models disagree on both. A box that is not
// four finite numbers, or that has no area once clamped to the image, is dropped.
func ParseDetections(text, order string, scale, width, height int) ([]Detection, error) {
	raw, err := extractJSON(text)
	if err != nil {
		return nil, err
	}
	var entries []map[string]any
	if err := json.Unmarshal(raw, &entries); err != nil {
		var wrapped map[string]json.RawMessage
		if json.Unmarshal(raw, &wrapped) != nil {
			return nil, fmt.Errorf("detections are neither a list nor an object holding one: %q", clip(text))
		}
		for _, key := range []string{"objects", "detections", "boxes", "results"} {
			if inner, ok := wrapped[key]; ok && json.Unmarshal(inner, &entries) == nil {
				break
			}
		}
	}
	sx, sy := 1.0, 1.0
	if scale > 0 {
		sx, sy = float64(width)/float64(scale), float64(height)/float64(scale)
	}
	found := make([]Detection, 0, len(entries))
	for _, entry := range entries {
		label := ""
		for _, key := range []string{"label", "name", "class", "category"} {
			if s, ok := entry[key].(string); ok && strings.TrimSpace(s) != "" {
				label = strings.TrimSpace(s)
				break
			}
		}
		values, ok := boxValues(entry)
		if label == "" || !ok {
			continue
		}
		x1, y1, x2, y2 := values[0], values[1], values[2], values[3]
		if order == models.BoxOrderYXYX {
			x1, y1, x2, y2 = values[1], values[0], values[3], values[2]
		}
		x1, x2 = math.Min(x1, x2)*sx, math.Max(x1, x2)*sx
		y1, y2 = math.Min(y1, y2)*sy, math.Max(y1, y2)*sy
		x1, x2 = clamp(x1, 0, float64(width)), clamp(x2, 0, float64(width))
		y1, y2 = clamp(y1, 0, float64(height)), clamp(y2, 0, float64(height))
		if x2-x1 < 1 || y2-y1 < 1 {
			continue
		}
		found = append(found, Detection{Label: label, Box: [4]float64{round1(x1), round1(y1), round1(x2), round1(y2)}})
	}
	if len(found) == 0 {
		return nil, fmt.Errorf("no usable boxes in %q", clip(text))
	}
	return found, nil
}

func boxValues(entry map[string]any) ([4]float64, bool) {
	var out [4]float64
	for _, key := range boxKeys {
		list, ok := entry[key].([]any)
		if !ok || len(list) != 4 {
			continue
		}
		for i, v := range list {
			f, ok := v.(float64)
			if !ok || math.IsNaN(f) || math.IsInf(f, 0) {
				return out, false
			}
			out[i] = f
		}
		return out, true
	}
	return out, false
}

func clamp(v, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, v)) }
func round1(v float64) float64        { return math.Round(v*10) / 10 }

// visionProvider is the local rung: a general vision model the gateway reaches
// through Ollama on this machine. It is the same executor the hosted provider
// uses, with the local-only profile.
type visionProvider struct {
	gateway VisionGateway
	// slot admits one local call at a time. The gateway's Ollama resource runs one
	// request at once (default_parallel_requests 1) and fails a request that waits
	// behind another for longer than its own 60 s admission window, which also
	// opens the provider's circuit breaker for everyone. Waiting here, in the
	// job's own context, costs nothing and fails nothing.
	slot chan struct{}
}

// NewVisionProvider builds the local gateway vision provider.
func NewVisionProvider(gateway VisionGateway) backends.Provider {
	return &visionProvider{gateway: gateway, slot: make(chan struct{}, 1)}
}

func (p *visionProvider) Name() string { return models.BackendGatewayVision }
func (p *visionProvider) Operations() []string {
	return append([]string(nil), models.VisionOperations...)
}
func (p *visionProvider) Standalone() bool { return true }
func (p *visionProvider) IsCloud() bool    { return false }

// GPUCapable is false on purpose: the gateway's capacity broker places the model
// on the GPU or not; image-tools holds no GPU claim for a call it only waits on.
func (p *visionProvider) GPUCapable() bool { return false }

// LocalTier reports the gateway-local tier: the call runs on this machine, on
// hardware the gateway manages, not on a CPU or GPU this scenario reserves.
func (p *visionProvider) LocalTier() backends.Tier { return backends.TierGatewayLocal }

func (p *visionProvider) Available(ctx context.Context) bool {
	return p.Availability(ctx).Available
}

func (p *visionProvider) Availability(ctx context.Context) backends.Availability {
	provision := "start ai-gateway and its Ollama resource; the row's gateway role chooses the model"
	if p == nil || p.gateway == nil {
		return backends.Availability{Detail: "AI Gateway vision client is not configured", Provision: provision}
	}
	if err := p.gateway.Reachable(ctx); err != nil {
		return backends.Availability{Detail: err.Error(), Provision: provision}
	}
	return backends.Availability{Available: true, Detail: "AI Gateway is reachable; the model is the gateway role's local choice", Provision: provision}
}

func (p *visionProvider) Execute(ctx context.Context, req backends.Request) (backends.Result, error) {
	select {
	case p.slot <- struct{}{}:
		defer func() { <-p.slot }()
	case <-ctx.Done():
		return backends.Result{}, fmt.Errorf("ai: waiting for the local vision model: %w", ctx.Err())
	}
	return executeVision(ctx, p.gateway, models.BackendGatewayVision, true, req)
}
