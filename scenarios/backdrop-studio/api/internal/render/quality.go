package render

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	_ "image/png" // the treatment chain and every source lane deliver PNG

	"backdrop-studio/internal/catalog"
	"backdrop-studio/internal/perceptual"
)

// QualityRejectedError is what a candidate that failed the perceptual gate
// returns. It is a distinct type rather than a formatted string so the handler
// can map it to FailedPrecondition — the render worked, the result is not
// usable, and retrying with the same style and seed will fail the same way.
type QualityRejectedError struct {
	StyleID string
	Seed    int64
	Verdict perceptual.Verdict
}

func (e *QualityRejectedError) Error() string {
	return fmt.Sprintf("style %q at seed %d did not survive its own treatment: %s", e.StyleID, e.Seed, e.Verdict.Error())
}

// scoreCandidate measures the treated candidate against the source it was made
// from, at the style's effective bar.
func scoreCandidate(source, treated []byte, style catalog.Style) (perceptual.Verdict, error) {
	src, err := decodeImage(source)
	if err != nil {
		return perceptual.Verdict{}, fmt.Errorf("decode source: %w", err)
	}
	out, err := decodeImage(treated)
	if err != nil {
		return perceptual.Verdict{}, fmt.Errorf("decode treated candidate: %w", err)
	}
	bar := style.EffectiveQuality()
	return perceptual.Score(src, out, reservedRegions(style), perceptual.Thresholds{
		MinSubjectSurvival:     bar.MinSubjectSurvival,
		MinTonalOccupancy:      bar.MinTonalOccupancy,
		MinFrequencyModulation: bar.MinFrequencyModulation,
		MaxReservedQuiet:       bar.MaxReservedQuiet,
	}), nil
}

// reservedRegions converts the style's declared regions into the plain
// rectangles the perceptual package takes. Only regions that will actually hold
// something are passed: a decorative region is not a place text has to be
// readable, so measuring texture inside it would reject candidates for a
// problem that does not exist.
func reservedRegions(style catalog.Style) []perceptual.Region {
	out := make([]perceptual.Region, 0, len(style.Regions))
	for _, r := range style.Regions {
		if r.Kind == "decorative" || r.Width <= 0 || r.Height <= 0 {
			continue
		}
		out = append(out, perceptual.Region{X: r.X, Y: r.Y, Width: r.Width, Height: r.Height})
	}
	return out
}

func decodeImage(data []byte) (image.Image, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("no image bytes")
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	return img, err
}

// qualityJSON renders a verdict for the candidate record. It is stored as JSON
// rather than as typed columns because the metric set is expected to grow, and
// an operator reading `render get` needs the numbers more than the schema needs
// the rigidity.
func qualityJSON(v perceptual.Verdict) string {
	raw, err := json.Marshal(v)
	if err != nil {
		// A verdict that cannot be encoded is a bug in this file, not a
		// condition worth failing a good render over.
		return ""
	}
	return string(raw)
}
