package ops

import (
	"fmt"
	"image"

	"github.com/disintegration/imaging"
)

// resolveOutputFormat chooses the encode format: an explicit, encodable
// request wins; otherwise the source format if it is encodable; otherwise PNG
// (the safe lossless fallback for decode-only sources like HEIC/SVG).
func resolveOutputFormat(requested, source string) string {
	if requested != "" && CanEncode(requested) {
		return normalizeFormat(requested)
	}
	if CanEncode(source) {
		return normalizeFormat(source)
	}
	return FormatPNG
}

// compressToTarget re-encodes img into the resolved lossy format at the highest
// quality whose output is <= p.TargetBytes, via binary search over quality. It
// requires a lossy format (jpeg/webp/avif); a lossless target size is not a
// quality trade-off. Returns the best-fitting bytes, or the smallest achievable
// output (lowest quality) when even q=1 exceeds the target, with the chosen
// quality reported to the caller via the result.
func compressToTarget(img image.Image, p *Params) (data []byte, format string, quality int, err error) {
	format = resolveOutputFormat(p.Format, FormatJPEG)
	if !isLossy(format) {
		// No quality knob; fall back to JPEG for target-size compression.
		format = FormatJPEG
	}
	target := p.TargetBytes
	if target <= 0 {
		return nil, "", 0, fmt.Errorf("ops: compress target_bytes must be positive")
	}

	lo, hi := 1, 100
	bestData, bestQ := []byte(nil), -1
	// Binary search for the largest quality whose size <= target.
	for lo <= hi {
		mid := (lo + hi) / 2
		enc, encErr := Encode(img, format, EncodeOptions{Quality: mid})
		if encErr != nil {
			return nil, "", 0, encErr
		}
		if int64(len(enc)) <= target {
			bestData, bestQ = enc, mid
			lo = mid + 1
		} else {
			hi = mid - 1
		}
	}
	if bestData == nil {
		// Even q=1 exceeds the target; return the smallest we can produce.
		enc, encErr := Encode(img, format, EncodeOptions{Quality: 1})
		if encErr != nil {
			return nil, "", 0, encErr
		}
		return enc, format, 1, nil
	}
	return bestData, format, bestQ, nil
}

func isLossy(format string) bool {
	switch normalizeFormat(format) {
	case FormatJPEG, FormatWebP, FormatAVIF:
		return true
	default:
		return false
	}
}

// convertClone is the pixel transform for the convert op: it returns the image
// unchanged and lets the runner encode it into the requested format. Convert
// requires an explicit, encodable target format.
func convertClone(img image.Image, p *Params) (image.Image, error) {
	if p.Format == "" {
		return nil, fmt.Errorf("ops: convert requires a target format")
	}
	if !CanEncode(p.Format) {
		return nil, fmt.Errorf("ops: cannot convert to %q (encodable: %v)", p.Format, EncodableFormats)
	}
	return imaging.Clone(img), nil
}
