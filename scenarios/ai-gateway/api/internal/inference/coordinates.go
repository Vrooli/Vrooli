package inference

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

const (
	CoordinateConventionNormalized1000 = "normalized_1000"
	CoordinateConventionAbsolutePixels = "absolute_pixels"
)

// IsVisualLocateRole reports whether the request uses the canonical visual
// grounding contract. browser.locate is the browser-facing alias; both roles
// share the same schema, attachment checks, coordinate normalization and
// privacy/source routing.
func IsVisualLocateRole(role string) bool {
	role = strings.TrimSpace(role)
	return role == "locate.visual" || role == "browser.locate"
}

type visualResult struct {
	Found      bool      `json:"found"`
	Bounds     []float64 `json:"bounds"`
	Confidence float64   `json:"confidence"`
}

// NormalizeLocateVisualJSON converts the model-declared coordinate convention
// to the gateway's canonical relative bounds. It deliberately takes the
// convention as an input: numeric magnitude is not a safe discriminator.
func NormalizeLocateVisualJSON(raw, convention string, attachments []*sharedv1.Attachment) (string, error) {
	result, err := parseLocateVisualResult(raw)
	if err != nil {
		return "", err
	}
	width, height, err := visualDimensions(attachments)
	if err != nil {
		return "", err
	}
	bounds, err := normalizeLocateBounds(result.Bounds, convention, width, height)
	if err != nil {
		return "", err
	}
	result.Bounds = []float64{bounds[0], bounds[1], bounds[2], bounds[3]}
	encoded, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("marshal normalized locate.visual response: %w", err)
	}
	return string(encoded), nil
}

func parseLocateVisualResult(raw string) (visualResult, error) {
	var result visualResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return visualResult{}, fmt.Errorf("locate.visual response is not valid JSON: %w", err)
	}
	if len(result.Bounds) != 4 {
		return visualResult{}, fmt.Errorf("locate.visual bounds must contain exactly four numbers")
	}
	if !finite(result.Bounds) || math.IsNaN(result.Confidence) || math.IsInf(result.Confidence, 0) || result.Confidence < 0 || result.Confidence > 1 {
		return visualResult{}, fmt.Errorf("locate.visual response contains a non-finite or out-of-range value")
	}
	return result, nil
}

func normalizeLocateBounds(values []float64, convention string, width, height float64) ([4]float64, error) {
	var bounds [4]float64
	switch strings.TrimSpace(convention) {
	case CoordinateConventionNormalized1000:
		for i, value := range values {
			bounds[i] = value / 1000
		}
	case CoordinateConventionAbsolutePixels:
		bounds[0] = values[0] / width
		bounds[1] = values[1] / height
		bounds[2] = values[2] / width
		bounds[3] = values[3] / height
	default:
		return [4]float64{}, fmt.Errorf("locate.visual model declared unsupported coordinate convention %q", convention)
	}
	for _, value := range bounds {
		if value < 0 || value > 1 || math.IsNaN(value) || math.IsInf(value, 0) {
			return [4]float64{}, fmt.Errorf("locate.visual bounds fall outside the submitted image")
		}
	}
	return bounds, nil
}

func visualDimensions(attachments []*sharedv1.Attachment) (float64, float64, error) {
	for _, attachment := range attachments {
		if attachment == nil || attachment.GetModality() != sharedv1.Modality_MODALITY_IMAGE {
			continue
		}
		if attachment.GetWidth() <= 0 || attachment.GetHeight() <= 0 {
			return 0, 0, fmt.Errorf("locate.visual requires positive image dimensions")
		}
		return float64(attachment.GetWidth()), float64(attachment.GetHeight()), nil
	}
	return 0, 0, fmt.Errorf("locate.visual requires an image attachment")
}

func finite(values []float64) bool {
	for _, value := range values {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return false
		}
	}
	return true
}
