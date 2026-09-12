// Package ops is the deterministic image-operation core: pure, zero-download,
// cross-platform Go transforms (resize/crop/rotate/convert/adjust/filter/
// compose/thumbnail/compress/metadata) plus the codec layer that decodes and
// encodes every supported format. Nothing here downloads a model, touches a
// GPU, or requires ComfyUI — these are the ops that must run headless on any
// host (the IMG-P0-001 headless-completeness tenet).
//
// The package is deliberately transport-agnostic: ops operate on image.Image
// and []byte, and know nothing about jobs, storage, proto, or HTTP. The job
// runner and handlers wire those seams on top (see internal/jobrunner and
// handlers/ops).
package ops

import (
	"bytes"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"strings"

	// Standard + extended decoders register themselves via init(), so
	// image.Decode auto-detects them. x/image adds tiff/bmp/webp-decode; the
	// gen2brain packages add webp/avif/heic via embedded WASM codecs (pure Go,
	// no cgo). Encoders are called explicitly in Encode.
	_ "github.com/gen2brain/avif"
	_ "github.com/gen2brain/heic"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"

	"github.com/gen2brain/avif"
	"github.com/gen2brain/webp"
	xbmp "golang.org/x/image/bmp"
	xtiff "golang.org/x/image/tiff"
)

// Format identifiers — the canonical lowercase names used across the package,
// the proto contract, and the CLI. These match the strings image.Decode
// reports for registered decoders.
const (
	FormatPNG  = "png"
	FormatJPEG = "jpeg"
	FormatGIF  = "gif"
	FormatWebP = "webp"
	FormatTIFF = "tiff"
	FormatBMP  = "bmp"
	FormatAVIF = "avif"
	FormatHEIC = "heic"
	FormatSVG  = "svg" // input-only (vector → raster); never an output format.
)

// EncodableFormats are the formats Encode can write. HEIC is intentionally
// absent: the patent-encumbered HEVC encoder is not bundled (decode-in /
// convert-out only), and SVG is vector (raster import only).
var EncodableFormats = []string{
	FormatPNG, FormatJPEG, FormatGIF, FormatWebP, FormatTIFF, FormatBMP, FormatAVIF,
}

// DecodableFormats are the formats Decode recognizes (auto-detected, plus SVG
// by content sniff).
var DecodableFormats = []string{
	FormatPNG, FormatJPEG, FormatGIF, FormatWebP, FormatTIFF, FormatBMP, FormatAVIF, FormatHEIC, FormatSVG,
}

// Meta describes a decoded image's source facts.
type Meta struct {
	Format string // canonical format name (see Format* constants)
	Width  int
	Height int
}

// EncodeOptions controls lossy/lossless encoding. Zero value means "encoder
// default" for each field.
type EncodeOptions struct {
	// Quality in [1,100] for lossy formats (jpeg/webp/avif). 0 = encoder default.
	Quality int
	// Lossless requests lossless encoding where the format supports it (webp).
	Lossless bool
}

// codec errors.
var (
	// ErrUnsupportedEncodeFormat is returned by Encode for a format it cannot write.
	ErrUnsupportedEncodeFormat = fmt.Errorf("ops: unsupported encode format")
	// ErrDecode wraps any failure to decode source bytes into an image.
	ErrDecode = fmt.Errorf("ops: decode failed")
)

// Decode turns encoded image bytes into an image.Image plus its source Meta.
// SVG is detected by content sniff and rasterized; all other formats are
// auto-detected by the registered decoders.
func Decode(data []byte) (image.Image, Meta, error) {
	if len(data) == 0 {
		return nil, Meta{}, fmt.Errorf("%w: empty input", ErrDecode)
	}
	if looksLikeSVG(data) {
		img, err := rasterizeSVG(data, 0, 0)
		if err != nil {
			return nil, Meta{}, err
		}
		b := img.Bounds()
		return img, Meta{Format: FormatSVG, Width: b.Dx(), Height: b.Dy()}, nil
	}
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, Meta{}, fmt.Errorf("%w: %v", ErrDecode, err)
	}
	b := img.Bounds()
	return img, Meta{Format: normalizeFormat(format), Width: b.Dx(), Height: b.Dy()}, nil
}

// Encode writes img in the named format with the given options, returning the
// encoded bytes. Unknown/unencodable formats return ErrUnsupportedEncodeFormat.
func Encode(img image.Image, format string, opts EncodeOptions) ([]byte, error) {
	format = normalizeFormat(format)
	var buf bytes.Buffer
	if err := encodeTo(&buf, img, format, opts); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func encodeTo(w io.Writer, img image.Image, format string, opts EncodeOptions) error {
	switch format {
	case FormatPNG:
		enc := png.Encoder{CompressionLevel: png.DefaultCompression}
		return enc.Encode(w, img)
	case FormatJPEG:
		q := opts.Quality
		if q <= 0 {
			q = 85
		}
		return jpeg.Encode(w, img, &jpeg.Options{Quality: clampQuality(q)})
	case FormatGIF:
		return gif.Encode(w, img, nil)
	case FormatTIFF:
		return xtiff.Encode(w, img, &xtiff.Options{Compression: xtiff.Deflate})
	case FormatBMP:
		return xbmp.Encode(w, img)
	case FormatWebP:
		o := webp.Options{Quality: webpQuality(opts.Quality), Lossless: opts.Lossless}
		return webp.Encode(w, img, o)
	case FormatAVIF:
		q := opts.Quality
		if q <= 0 {
			q = 60
		}
		return avif.Encode(w, img, avif.Options{Quality: clampQuality(q), QualityAlpha: clampQuality(q)})
	default:
		return fmt.Errorf("%w: %q", ErrUnsupportedEncodeFormat, format)
	}
}

// MIMEFor returns the canonical MIME type for a format ("" if unknown).
func MIMEFor(format string) string {
	switch normalizeFormat(format) {
	case FormatPNG:
		return "image/png"
	case FormatJPEG:
		return "image/jpeg"
	case FormatGIF:
		return "image/gif"
	case FormatWebP:
		return "image/webp"
	case FormatTIFF:
		return "image/tiff"
	case FormatBMP:
		return "image/bmp"
	case FormatAVIF:
		return "image/avif"
	case FormatHEIC:
		return "image/heic"
	case FormatSVG:
		return "image/svg+xml"
	default:
		return ""
	}
}

// FormatFromExt maps a filename extension (with or without leading dot) to a
// canonical format name ("" if unrecognized).
func FormatFromExt(ext string) string {
	ext = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(ext), "."))
	switch ext {
	case "png":
		return FormatPNG
	case "jpg", "jpeg", "jpe":
		return FormatJPEG
	case "gif":
		return FormatGIF
	case "webp":
		return FormatWebP
	case "tif", "tiff":
		return FormatTIFF
	case "bmp":
		return FormatBMP
	case "avif":
		return FormatAVIF
	case "heic", "heif":
		return FormatHEIC
	case "svg":
		return FormatSVG
	default:
		return ""
	}
}

// CanEncode reports whether Encode can write the given format.
func CanEncode(format string) bool {
	format = normalizeFormat(format)
	for _, f := range EncodableFormats {
		if f == format {
			return true
		}
	}
	return false
}

// normalizeFormat folds decoder-reported aliases onto the canonical names.
func normalizeFormat(format string) string {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "jpg", "jpeg":
		return FormatJPEG
	case "tif", "tiff":
		return FormatTIFF
	case "heif", "heic":
		return FormatHEIC
	default:
		return strings.ToLower(strings.TrimSpace(format))
	}
}

func clampQuality(q int) int {
	if q < 1 {
		return 1
	}
	if q > 100 {
		return 100
	}
	return q
}

// webpQuality maps our EncodeOptions to gen2brain/webp (default 75 when unset).
func webpQuality(q int) int {
	if q <= 0 {
		return 75
	}
	return clampQuality(q)
}

func looksLikeSVG(data []byte) bool {
	// Sniff the first chunk for an <svg root, tolerating a leading XML
	// declaration / BOM / whitespace.
	head := data
	if len(head) > 1024 {
		head = head[:1024]
	}
	head = bytes.TrimPrefix(bytes.TrimSpace(head), []byte{0xEF, 0xBB, 0xBF}) // strip UTF-8 BOM
	s := strings.ToLower(string(bytes.TrimSpace(head)))
	return strings.Contains(s, "<svg") && (strings.HasPrefix(s, "<svg") || strings.HasPrefix(s, "<?xml") || strings.HasPrefix(s, "<!doctype"))
}
