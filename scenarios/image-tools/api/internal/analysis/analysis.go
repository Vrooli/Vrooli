// Package analysis is image-tools' image→data engine (IMG-P0-004): OCR /
// text-extraction, NSFW / safety classification, and image info / probe. Unlike
// the generation/enhancement ops these return STRUCTURED DATA rather than an
// output image, so they execute synchronously and are recorded as terminal
// durable jobs for uniform observability.
//
// `probe` is pure-Go and always works headless (no model, no GPU). `ocr` and
// `nsfw_classify` are model-backed standalone ops with CPU-capable defaults
// (tesseract; an onnxruntime NSFW classifier) — when their backend program or
// model weights are absent the op refuses with an actionable hint via
// ErrBackendUnavailable instead of failing opaquely.
//
// The NSFW path is also exposed as an in-process scanner so the AI generation
// engine's optional auto-scan hook can classify generated output without the ai
// package importing this one.
package analysis

import (
	"context"
	"errors"
	"sort"
)

// Operation names (match the registry operation vocabulary where model-backed).
const (
	OpOCR   = "ocr"
	OpNSFW  = "nsfw_classify"
	OpProbe = "probe"
)

// ErrBackendUnavailable is returned when a model-backed op's program/model is
// not installed. Callers surface it as an actionable install hint.
var ErrBackendUnavailable = errors.New("analysis: backend not available")

// ErrUnknownOperation is returned for an op outside the analysis catalog.
var ErrUnknownOperation = errors.New("analysis: unknown operation")

// OpInfo describes one analysis operation for discovery.
type OpInfo struct {
	Name           string
	Summary        string
	ModelBacked    bool
	DefaultModelID string
}

// catalog is the canonical analysis-op table.
var catalog = []OpInfo{
	{Name: OpOCR, Summary: "Extract text from an image (OCR)", ModelBacked: true, DefaultModelID: "tesseract"},
	{Name: OpNSFW, Summary: "Classify an image for NSFW / unsafe content", ModelBacked: true, DefaultModelID: "adamcodd-vit-nsfw"},
	{Name: OpProbe, Summary: "Report structured image info (dimensions, format, metadata, palette)", ModelBacked: false},
}

// List returns the analysis catalog in stable (sorted) order.
func List() []OpInfo {
	out := append([]OpInfo(nil), catalog...)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Has reports whether name is a registered analysis operation.
func Has(name string) bool {
	for _, o := range catalog {
		if o.Name == name {
			return true
		}
	}
	return false
}

// --- result types (translated to proto by handlers/analysis) ---

// Box is a pixel-space rectangle (x,y = top-left).
type Box struct {
	X, Y, Width, Height int
}

// OCRBlock is one recognized text region.
type OCRBlock struct {
	Text       string
	Confidence float64
	Box        Box
}

// OCRResult is the structured output of the OCR op.
type OCRResult struct {
	FullText string
	Blocks   []OCRBlock
	Language string
}

// NSFWCategory is one classifier label with its score.
type NSFWCategory struct {
	Label string
	Score float64
}

// NSFWResult is the structured output of the NSFW op (and the auto-scan hook).
type NSFWResult struct {
	NSFW       bool
	Score      float64
	Label      string
	Threshold  float64
	Categories []NSFWCategory
}

// DominantColor is one extracted palette swatch.
type DominantColor struct {
	Hex      string
	Fraction float64
}

// ProbeResult is the structured output of the pure-Go probe op.
type ProbeResult struct {
	Width          int
	Height         int
	Format         string
	ColorModel     string
	HasAlpha       bool
	FrameCount     int
	Megapixels     float64
	SizeBytes      int64
	HasEXIF        bool
	HasGPS         bool
	Orientation    int
	DominantColors []DominantColor
}

// cmdOutput runs a program and returns its stdout. Injected so OCR/NSFW are
// testable without the real backend binaries.
type cmdOutput func(ctx context.Context, name string, args []string) ([]byte, error)

// lookPathFunc resolves a binary on PATH. Injected for the same reason.
type lookPathFunc func(file string) (string, error)
