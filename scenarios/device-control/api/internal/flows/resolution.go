// Package flows owns target resolution and flow evidence. It deliberately
// depends on the generated ai-gateway client rather than a provider SDK.
package flows

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	inferenceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference/inference_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

const (
	VisionRung       = "vision"
	VisualAnchorRung = "visual-anchor"
	LocateVisualRole = "locate.visual"

	defaultConfidenceThreshold = 0.70
	defaultMaxDimension        = 1024
)

var (
	ErrInvalidRequest = errors.New("invalid target-resolution request")
	ErrInvalidFrame   = errors.New("invalid frame")
	ErrUnresolved     = errors.New("target unresolved")
)

// UnavailableError is a typed, provider-neutral failure. Its reason is safe
// to expose to callers; it never includes frame bytes or screen text.
type UnavailableError struct{ Reason string }

func (e *UnavailableError) Error() string {
	if e == nil || e.Reason == "" {
		return "vision route unavailable"
	}
	return "vision route unavailable: " + e.Reason
}

// Frame is the caller-owned capture after the resolver has applied its
// submission policy. OriginalWidth and OriginalHeight remain the device
// coordinate space used by the final result.
type Frame struct {
	Bytes          []byte
	MediaType      string
	Width          int
	Height         int
	OriginalWidth  int
	OriginalHeight int
}

// Request is the flow-level target intent. FallbackBounds represents a
// previously captured visual anchor in canonical normalized coordinates.
type Request struct {
	Target              string
	Frame               Frame
	ConfidenceThreshold float64
	FallbackBounds      []float64
	FallbackConfidence  float64
	Anchors             *AnchorStore
	Semantic            func(context.Context, string) (SemanticMatch, error)
}

type SemanticMatch struct {
	Bounds     []float64
	Confidence float64
}

type EvidenceEvent struct {
	Name            string   `json:"name"`
	Rung            string   `json:"rung,omitempty"`
	Confidence      *float64 `json:"confidence,omitempty"`
	Reason          string   `json:"reason,omitempty"`
	SubmittedWidth  int      `json:"submitted_width,omitempty"`
	SubmittedHeight int      `json:"submitted_height,omitempty"`
}

// Result is the durable-safe flow evidence shape. Bounds are canonical
// normalized coordinates; DeviceBounds is derived locally from the original
// capture dimensions and is never sent to the model.
type Result struct {
	Status            string          `json:"status"`
	Rung              string          `json:"rung,omitempty"`
	Bounds            []float64       `json:"bounds,omitempty"`
	DeviceBounds      []int           `json:"device_bounds,omitempty"`
	Confidence        float64         `json:"confidence,omitempty"`
	FallbackUsed      bool            `json:"fallback_used"`
	Provider          string          `json:"provider,omitempty"`
	Model             string          `json:"model,omitempty"`
	SubmittedWidth    int             `json:"submitted_width"`
	SubmittedHeight   int             `json:"submitted_height"`
	OriginalWidth     int             `json:"original_width"`
	OriginalHeight    int             `json:"original_height"`
	Evidence          []EvidenceEvent `json:"evidence"`
	UnavailableReason string          `json:"unavailable_reason,omitempty"`
}

// Gateway resolves ai-gateway lazily, so device-control remains healthy for
// flows that do not need inference while still using the generated client for
// every vision request.
type Gateway struct {
	HTTPClient *http.Client
	ResolveURL func(context.Context, string) (string, error)
}

// InferenceRunner is the generated-client seam used by Resolver. Production
// supplies Gateway; tests use a response fixture without bypassing request
// construction or response normalization.
type InferenceRunner interface {
	Run(context.Context, *connect.Request[inferencev1.RunRequest]) (*connect.Response[inferencev1.RunResponse], error)
}

func NewGateway(httpClient *http.Client, resolveURL func(context.Context, string) (string, error)) *Gateway {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Gateway{HTTPClient: httpClient, ResolveURL: resolveURL}
}

func (g *Gateway) Run(ctx context.Context, req *connect.Request[inferencev1.RunRequest]) (*connect.Response[inferencev1.RunResponse], error) {
	if g == nil || g.ResolveURL == nil {
		return nil, &UnavailableError{Reason: "ai_gateway_client_not_configured"}
	}
	baseURL, err := g.ResolveURL(ctx, "ai-gateway")
	if err != nil {
		return nil, &UnavailableError{Reason: "ai_gateway_unreachable"}
	}
	client := inferenceconnect.NewInferenceServiceClient(g.HTTPClient, baseURL)
	return client.Run(ctx, req)
}

type Resolver struct {
	Gateway InferenceRunner
	Anchors *AnchorStore
}

func NewResolver(gateway InferenceRunner) *Resolver {
	return &Resolver{Gateway: gateway, Anchors: NewAnchorStore()}
}

func (r *Resolver) Resolve(ctx context.Context, req Request) (Result, error) {
	if strings.TrimSpace(req.Target) == "" {
		return Result{}, fmt.Errorf("%w: target is required", ErrInvalidRequest)
	}
	if err := validateFrame(req.Frame); err != nil {
		return Result{}, err
	}
	threshold := req.ConfidenceThreshold
	if threshold == 0 {
		threshold = defaultConfidenceThreshold
	}
	if threshold < 0 || threshold > 1 {
		return Result{}, fmt.Errorf("%w: confidence threshold must be between 0 and 1", ErrInvalidRequest)
	}

	result := Result{
		Status:          "unresolved",
		SubmittedWidth:  req.Frame.Width,
		SubmittedHeight: req.Frame.Height,
		OriginalWidth:   req.Frame.OriginalWidth,
		OriginalHeight:  req.Frame.OriginalHeight,
		Evidence:        []EvidenceEvent{},
	}
	anchors := req.Anchors
	if anchors == nil && r != nil {
		anchors = r.Anchors
	}
	if req.Semantic != nil {
		match, semanticErr := req.Semantic(ctx, req.Target)
		if semanticErr == nil && validBounds(match.Bounds) && finite(match.Confidence) {
			result.Status, result.Rung, result.Bounds = "resolved", "semantic", append([]float64(nil), match.Bounds...)
			result.DeviceBounds, result.Confidence = deviceBounds(match.Bounds, req.Frame.OriginalWidth, req.Frame.OriginalHeight), match.Confidence
			result.Evidence = append(result.Evidence, EvidenceEvent{Name: "resolved", Rung: "semantic", Confidence: ptr(match.Confidence)})
			return result, nil
		}
		result.Evidence = append(result.Evidence, EvidenceEvent{Name: "skip", Rung: "semantic", Reason: "semantic_target_not_found"})
	} else {
		result.Evidence = append(result.Evidence, EvidenceEvent{Name: "skip", Rung: "semantic", Reason: "semantic_capability_not_provided"})
	}
	if anchor, ok := anchors.Resolve(req.Target); ok {
		result.Status, result.Rung, result.Bounds = "resolved", VisualAnchorRung, append([]float64(nil), anchor.Bounds...)
		result.DeviceBounds, result.Confidence = deviceBounds(anchor.Bounds, req.Frame.OriginalWidth, req.Frame.OriginalHeight), anchor.Confidence
		result.FallbackUsed = true
		result.Evidence = append(result.Evidence, EvidenceEvent{Name: "resolved", Rung: VisualAnchorRung, Confidence: ptr(anchor.Confidence)})
		return result, nil
	}
	result.Evidence = append(result.Evidence, EvidenceEvent{Name: "skip", Rung: VisualAnchorRung, Reason: "anchor_not_found"})
	result.Evidence = append(result.Evidence, EvidenceEvent{Name: "attempt_vision", Rung: VisionRung, SubmittedWidth: req.Frame.Width, SubmittedHeight: req.Frame.Height})

	response, err := r.runVision(ctx, req)
	if err != nil {
		var unavailable *UnavailableError
		if errors.As(err, &unavailable) {
			return r.resolveUnavailable(req, result, unavailable.Reason)
		}
		return Result{}, err
	}
	if response == nil {
		return r.resolveUnavailable(req, result, "empty_gateway_response")
	}
	if response.GetError() != nil {
		if response.GetError().GetCode() == inferencev1.InferenceErrorCode_INFERENCE_ERROR_CODE_UNAVAILABLE {
			return r.resolveUnavailable(req, result, "gateway_route_unavailable")
		}
		return Result{}, fmt.Errorf("gateway inference failed: %s", response.GetError().GetMessage())
	}
	var value struct {
		Found      bool      `json:"found"`
		Bounds     []float64 `json:"bounds"`
		Confidence float64   `json:"confidence"`
	}
	if err := json.Unmarshal([]byte(response.GetValueJson()), &value); err != nil {
		return Result{}, fmt.Errorf("decode locate.visual response: %w", err)
	}
	if !value.Found || !validBounds(value.Bounds) || !finite(value.Confidence) || value.Confidence < 0 || value.Confidence > 1 {
		return Result{}, fmt.Errorf("%w: locate.visual returned an invalid result", ErrUnresolved)
	}
	if value.Confidence < threshold {
		if fallback, ok := fallbackResult(req, result, "confidence_below_threshold"); ok {
			return fallback, nil
		}
		result.Evidence = append(result.Evidence, EvidenceEvent{
			Name: "unresolved", Rung: VisionRung, Confidence: ptr(value.Confidence), Reason: "confidence_below_threshold",
		})
		return result, fmt.Errorf("%w: confidence below caller threshold", ErrUnresolved)
	}
	result.Status = "resolved"
	result.Rung = VisionRung
	result.Bounds = value.Bounds
	result.DeviceBounds = deviceBounds(value.Bounds, req.Frame.OriginalWidth, req.Frame.OriginalHeight)
	result.Confidence = value.Confidence
	result.Provider = response.GetProvider()
	result.Model = response.GetModel()
	result.Evidence = append(result.Evidence, EvidenceEvent{Name: "resolved", Rung: VisionRung, Confidence: ptr(value.Confidence)})
	if req.Anchors != nil {
		// A successful vision result becomes a deterministic replay anchor. The
		// frame checksum is retained as provenance, while the next run uses the
		// normalized bounds and never resubmits the frame to ai-gateway.
		if _, anchorErr := req.Anchors.CreateFromFrame(req.Target, req.Target, value.Bounds, value.Confidence, req.Frame.Bytes); anchorErr != nil {
			return Result{}, fmt.Errorf("persist visual anchor: %w", anchorErr)
		}
	}
	return result, nil
}

func (r *Resolver) runVision(ctx context.Context, req Request) (*inferencev1.RunResponse, error) {
	if r == nil || r.Gateway == nil {
		return nil, &UnavailableError{Reason: "ai_gateway_client_not_configured"}
	}
	request := connect.NewRequest(&inferencev1.RunRequest{
		Role:        LocateVisualRole,
		Instruction: fmt.Sprintf("Locate the target intent %q in the supplied screenshot. Return JSON with found, bounds as [x1,y1,x2,y2], and confidence from 0..1. The gateway owns coordinate normalization; do not assume a caller coordinate convention.", req.Target),
		// The gateway's typed schema subset deliberately does not include
		// minItems/maxItems. The resolver enforces the exact four-bound
		// contract after decoding, so the provider cannot widen it silently.
		SchemaJson: `{"type":"object","required":["found","bounds","confidence"],"properties":{"found":{"type":"boolean"},"bounds":{"type":"array","items":{"type":"number"}},"confidence":{"type":"number"}}}`,
		Attachments: []*sharedv1.Attachment{{
			Modality:  sharedv1.Modality_MODALITY_IMAGE,
			MediaType: req.Frame.MediaType,
			Width:     uint32(req.Frame.Width),
			Height:    uint32(req.Frame.Height),
			Bytes:     uint64(len(req.Frame.Bytes)),
			Payload:   &sharedv1.Attachment_InlineBytes{InlineBytes: req.Frame.Bytes},
		}},
	})
	response, err := r.Gateway.Run(ctx, request)
	if err != nil {
		var unavailable *UnavailableError
		if errors.As(err, &unavailable) {
			return nil, err
		}
		return nil, &UnavailableError{Reason: "ai_gateway_request_failed"}
	}
	if response == nil {
		return nil, &UnavailableError{Reason: "empty_gateway_response"}
	}
	return response.Msg, nil
}
