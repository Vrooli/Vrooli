package ops

import (
	"fmt"
	"strconv"

	"github.com/vrooli/cli-core/cliapp"
	"google.golang.org/protobuf/reflect/protoreflect"

	opsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ops"
)

// flagKind is the scalar shape of a wire field. The generated table names one
// of these per flag; adding a kind here is the only change a new scalar type
// needs.
type flagKind uint8

const (
	flagString flagKind = iota
	flagInt32
	flagInt64
	flagUint32
	flagDouble
	flagBool
	flagStringList
	flagInt32List
)

// flagSpec binds a CLI flag to a field of the operation's OpParams message.
type flagSpec struct {
	Name  string
	Field string
	Kind  flagKind
}

// runSpec is one generated deterministic run command.
type runSpec struct {
	Op    string
	Desc  string
	Flags []flagSpec
}

// flagConnector adapts a flagKind to the CLI's flag definition. Parameters are
// positional in the CLI; an option the parser never binds cannot be read, so
// every scalar field of every operation message gets a flag.
func flagConnector(f flagSpec) cliapp.Flag {
	def := cliapp.Flag{Name: f.Name, Description: flagDescriptions[f.Name]}
	switch f.Kind {
	case flagBool:
		def.Bool = true
	case flagStringList, flagInt32List:
		def.Repeated = true
	}
	return def
}

// buildParams converts the parsed flags into the typed OpParams message for one
// operation. It walks the protobuf descriptor rather than naming fields, so a
// new parameter is reachable as soon as it exists on the wire.
//
// Unset flags are written as zero values, which protojson omits, so the wire
// payload is identical to the hand-built one this replaced.
func buildParams(spec runSpec, ctx cliapp.RunContext) (*opsv1.OpParams, error) {
	out := &opsv1.OpParams{}
	msg := out.ProtoReflect()
	oneof := msg.Descriptor().Oneofs().ByName("op")
	if oneof == nil {
		return nil, fmt.Errorf("OpParams has no op oneof")
	}
	member := oneof.Fields().ByName(protoreflect.Name(spec.Op))
	if member == nil {
		return nil, fmt.Errorf("operation %q has no parameter message", spec.Op)
	}
	target := msg.Mutable(member).Message()
	for _, f := range spec.Flags {
		field := target.Descriptor().Fields().ByName(protoreflect.Name(f.Field))
		if field == nil {
			return nil, fmt.Errorf("operation %q has no parameter %q", spec.Op, f.Field)
		}
		switch f.Kind {
		case flagString:
			target.Set(field, protoreflect.ValueOfString(ctx.Flag(f.Name)))
		case flagInt32:
			target.Set(field, protoreflect.ValueOfInt32(int32(parseInt(ctx.Flag(f.Name), 32))))
		case flagInt64:
			target.Set(field, protoreflect.ValueOfInt64(parseInt(ctx.Flag(f.Name), 64)))
		case flagUint32:
			target.Set(field, protoreflect.ValueOfUint32(uint32(parseInt(ctx.Flag(f.Name), 32))))
		case flagDouble:
			target.Set(field, protoreflect.ValueOfFloat64(parseFloat(ctx.Flag(f.Name))))
		case flagBool:
			target.Set(field, protoreflect.ValueOfBool(ctx.BoolFlag(f.Name)))
		case flagStringList:
			list := target.Mutable(field).List()
			for _, v := range ctx.FlagValues(f.Name) {
				list.Append(protoreflect.ValueOfString(v))
			}
		case flagInt32List:
			list := target.Mutable(field).List()
			for _, v := range ctx.FlagValues(f.Name) {
				list.Append(protoreflect.ValueOfInt32(int32(parseInt(v, 32))))
			}
		default:
			return nil, fmt.Errorf("operation %q parameter %q has unknown flag kind %d", spec.Op, f.Field, f.Kind)
		}
	}
	return out, nil
}

// flagDescriptions is the CLI's copy for the wire parameters. Descriptions are
// presentation, so they live here rather than in the wire contract; the shape
// (which flags exist) is generated from ops.proto.
var flagDescriptions = map[string]string{
	// Shared geometry.
	"width":      "Width in px (0 = derive from aspect)",
	"height":     "Height in px (0 = derive from aspect)",
	"fit":        "fit|fill|stretch",
	"gravity":    "Anchor for a fill crop or placement (center, top-left, …)",
	"x":          "Left offset in px",
	"axis":       "horizontal|vertical",
	"y":          "Top offset in px",
	"angle":      "Angle in degrees, counter-clockwise",
	"expand":     "Expand the canvas to fit a rotated image",
	"background": "Fill colour for exposed areas (hex)",
	// Screenshot cleanup.
	"tolerance":        "Per-channel match tolerance as a fraction of full scale (default 0.04)",
	"max-trim-percent": "Per-side trim ceiling as a fraction of the dimension (default 0.25)",
	"padding-px":       "Margin kept around the detected content, in pixels",
	// Tonal.
	"brightness":          "Brightness delta percent",
	"contrast":            "Contrast delta percent",
	"gamma":               "Gamma (1.0 = identity)",
	"saturation":          "Saturation delta percent",
	"hue":                 "Hue rotation in degrees",
	"filter":              "grayscale|sepia|invert|blur|sharpen",
	"amount":              "Effect strength (blur/sharpen sigma, canny smoothing)",
	"low-threshold":       "Canny hysteresis lower bound on the 0..255 gradient (0 = 50)",
	"high-threshold":      "Canny hysteresis upper bound on the 0..255 gradient (0 = 150)",
	"dark":                "Dark ink hex colour",
	"light":               "Light ink hex colour",
	"mid":                 "Optional mid ink hex colour",
	"mid-low":             "Mid ink lower luminance band",
	"mid-high":            "Mid ink upper luminance band",
	"normalize":           "Auto-level the source p1-p99 tonal range before mapping",
	"levels":              "Number of quantisation levels (2-256)",
	"lpi":                 "Screen lines across the image width",
	"dot":                 "circle|square",
	"seed":                "Deterministic seed",
	"contrast-multiplier": "Grain contrast multiplier",
	"opacity":             "Opacity 0..1",
	"direction":           "top|bottom|left|right",
	"color":               "Colour (hex)",
	"region-x":            "Region left, as a fraction of the frame",
	"region-y":            "Region top, as a fraction of the frame",
	"region-width":        "Region width, as a fraction of the frame",
	"region-height":       "Region height, as a fraction of the frame",
	"region-feather":      "Region feather, as a fraction of the short edge",
	// Relative spatial parameters.
	"spacing":        "Spacing in pixels",
	"spacing-rel":    "Spacing as a fraction of the short edge (wins over --spacing)",
	"radius":         "Radius in pixels",
	"radius-rel":     "Radius as a fraction of the short edge (wins over --radius)",
	"blade-count":    "Aperture blade count",
	"distance":       "Distance in pixels",
	"distance-rel":   "Distance as a fraction of the short edge (wins over --distance)",
	"amplitude":      "Amplitude in pixels",
	"amplitude-rel":  "Amplitude as a fraction of the short edge (wins over --amplitude)",
	"threshold":      "Threshold 0..1",
	"exponent":       "Curve exponent",
	"block-size":     "Block size in pixels",
	"block-size-rel": "Block size as a fraction of the short edge, snapped to the 7px glyph advance",
	// Format.
	"format":       "Output format (png|jpeg|gif|webp|tiff|bmp|avif, or ico|icns for icon containers)",
	"quality":      "Quality 1-100 for lossy formats",
	"lossless":     "Lossless encoding (webp)",
	"target-bytes": "Compress to at most this many bytes",
	// Watermark.
	"text":      "Text watermark",
	"position":  "Anchor (bottom-right, center, …)",
	"font-size": "Text size in px",
	// Metadata.
	"strip-all":   "Strip all metadata",
	"strip-gps":   "Strip metadata (removes GPS)",
	"auto-orient": "Apply EXIF orientation to pixels",
	// Vector bridge.
	"colors":                         "Palette entries for k-means (default 4)",
	"keep-colors":                    "Pin a layer to this hex colour (repeatable); assigns within delta-E 12",
	"drop-background-layers":         "Drop the layer touching the image border",
	"clip-to-largest-rounded-region": "Clip to the enclosing tile hull",
	"inset-px":                       "Clip erosion in px (default 2% of the short edge)",
	"tolerance-px":                   "Simplification tolerance in px (default 0.8)",
	"smoothing":                      "Fit cubic Beziers to the simplified contours",
	"min-area-px":                    "Drop contours smaller than this area (default 40)",
	"sizes":                          "Exact pixel size (repeatable)",
}

// parseInt parses a decimal flag value, treating empty or malformed input as the
// proto default (0). Flag-level validation is the server's job: it owns the
// per-operation bounds and reports them in one place.
func parseInt(s string, bits int) int64 {
	n, err := strconv.ParseInt(s, 10, bits)
	if err != nil {
		return 0
	}
	return n
}

func parseFloat(s string) float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}
