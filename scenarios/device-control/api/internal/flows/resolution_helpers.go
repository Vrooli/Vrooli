package flows

import (
	"fmt"
	"math"
	"strings"
)

func (r *Resolver) resolveUnavailable(req Request, result Result, reason string) (Result, error) {
	if fallback, ok := fallbackResult(req, result, reason); ok {
		return fallback, nil
	}
	result.Status = "unavailable"
	result.UnavailableReason = reason
	result.Evidence = append(result.Evidence, EvidenceEvent{Name: "unresolved", Rung: VisionRung, Reason: reason})
	return result, &UnavailableError{Reason: reason}
}

func fallbackResult(req Request, result Result, reason string) (Result, bool) {
	if !validBounds(req.FallbackBounds) {
		return Result{}, false
	}
	confidence := req.FallbackConfidence
	if confidence == 0 {
		confidence = 1
	}
	if confidence < 0 || confidence > 1 {
		return Result{}, false
	}
	result.Status = "resolved"
	result.Rung = VisualAnchorRung
	result.Bounds = append([]float64(nil), req.FallbackBounds...)
	result.DeviceBounds = deviceBounds(result.Bounds, req.Frame.OriginalWidth, req.Frame.OriginalHeight)
	result.Confidence = confidence
	result.FallbackUsed = true
	result.Evidence = append(result.Evidence,
		EvidenceEvent{Name: "fallback", Rung: VisualAnchorRung, Confidence: ptr(confidence), Reason: reason},
		EvidenceEvent{Name: "resolved", Rung: VisualAnchorRung, Confidence: ptr(confidence)},
	)
	return result, true
}

func validateFrame(frame Frame) error {
	if len(frame.Bytes) == 0 || frame.Width <= 0 || frame.Height <= 0 || frame.OriginalWidth <= 0 || frame.OriginalHeight <= 0 || strings.TrimSpace(frame.MediaType) == "" {
		return fmt.Errorf("%w: bytes, media type, and positive dimensions are required", ErrInvalidFrame)
	}
	return nil
}

func validBounds(bounds []float64) bool {
	if len(bounds) != 4 {
		return false
	}
	for _, value := range bounds {
		if !finite(value) || value < 0 || value > 1 {
			return false
		}
	}
	return bounds[0] <= bounds[2] && bounds[1] <= bounds[3]
}

func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }

func deviceBounds(bounds []float64, width, height int) []int {
	return []int{
		int(math.Round(bounds[0] * float64(width))),
		int(math.Round(bounds[1] * float64(height))),
		int(math.Round(bounds[2] * float64(width))),
		int(math.Round(bounds[3] * float64(height))),
	}
}

func ptr(value float64) *float64 { return &value }
