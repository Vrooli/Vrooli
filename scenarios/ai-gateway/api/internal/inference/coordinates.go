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

type visualResult struct {
	Found      bool      `json:"found"`
	Bounds     []float64 `json:"bounds"`
	Confidence float64   `json:"confidence"`
}

// NormalizeLocateVisualJSON converts the model-declared coordinate convention
// to the gateway's canonical relative bounds. It deliberately takes the
// convention as an input: numeric magnitude is not a safe discriminator.
func NormalizeLocateVisualJSON(raw, convention string, attachments []*sharedv1.Attachment) (string, error) {
	var result visualResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return "", fmt.Errorf("locate.visual response is not valid JSON: %w", err)
	}
	if len(result.Bounds) != 4 {
		return "", fmt.Errorf("locate.visual bounds must contain exactly four numbers")
	}
	if !finite(result.Bounds) || math.IsNaN(result.Confidence) || math.IsInf(result.Confidence, 0) || result.Confidence < 0 || result.Confidence > 1 {
		return "", fmt.Errorf("locate.visual response contains a non-finite or out-of-range value")
	}
	width, height, err := visualDimensions(attachments)
	if err != nil {
		return "", err
	}

	var bounds [4]float64
	switch strings.TrimSpace(convention) {
	case CoordinateConventionNormalized1000:
		for i, value := range result.Bounds {
			bounds[i] = value / 1000
		}
	case CoordinateConventionAbsolutePixels:
		bounds[0] = result.Bounds[0] / width
		bounds[1] = result.Bounds[1] / height
		bounds[2] = result.Bounds[2] / width
		bounds[3] = result.Bounds[3] / height
	default:
		return "", fmt.Errorf("locate.visual model declared unsupported coordinate convention %q", convention)
	}
	for _, value := range bounds {
		if value < 0 || value > 1 || math.IsNaN(value) || math.IsInf(value, 0) {
			return "", fmt.Errorf("locate.visual bounds fall outside the submitted image")
		}
	}
	result.Bounds = []float64{bounds[0], bounds[1], bounds[2], bounds[3]}
	encoded, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("marshal normalized locate.visual response: %w", err)
	}
	return string(encoded), nil
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
