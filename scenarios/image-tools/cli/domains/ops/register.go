package ops

import (
	"fmt"

	opsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ops"

	"github.com/vrooli/cli-core/cliapp"
)

// Register builds the ops subcommand group. `list` is bound from the manifest
// to the OpsService.ListOperations Connect RPC; the per-operation run commands
// drive the REST multipart edge and are appended directly (cli-manifest binds
// only Connect calls), documented in the manifest's `omitted` array.
func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	bindings := map[string]func(cliapp.RunContext) error{
		"OpsService.ListOperations": h.list,
	}
	group, err := cliapp.LoadFromManifest(manifest, GroupName, bindings)
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("ops: load from manifest: %w", err)
	}
	group.Subcommands = append(group.Subcommands, h.runCommands()...)
	return group, nil
}

// inputArg is the shared required input positional.
func inputArg() cliapp.Positional {
	return cliapp.Positional{Name: "input", Required: true, Description: "Path to the input image"}
}

func outFlag() cliapp.Flag {
	return cliapp.Flag{Name: "out", Description: "Path to write the result (required for image outputs)"}
}

// runCommands returns the per-operation REST run commands.
func (h *handlers) runCommands() []cliapp.Command {
	cmd := func(name, desc string, build func(cliapp.RunContext) *opsv1.OpParams, flags ...cliapp.Flag) cliapp.Command {
		return cliapp.Command{
			Name:        name,
			Description: desc,
			NeedsAPI:    true,
			Args:        cliapp.ArgSchema{Positionals: []cliapp.Positional{inputArg()}, Flags: append([]cliapp.Flag{outFlag()}, flags...)},
			RunCtx:      h.run(name, build),
		}
	}
	commands := []cliapp.Command{
		cmd("resize", "Resize an image (fit/fill/stretch)", h.resizeParams,
			cliapp.Flag{Name: "width", Description: "Target width in px"},
			cliapp.Flag{Name: "height", Description: "Target height in px"},
			cliapp.Flag{Name: "fit", Description: "fit|fill|stretch"},
			cliapp.Flag{Name: "gravity", Description: "Anchor for fill crop (center, top-left, …)"},
		),
		cmd("crop", "Crop a rectangle or an anchored region", h.cropParams,
			cliapp.Flag{Name: "x", Description: "Left offset in px"},
			cliapp.Flag{Name: "y", Description: "Top offset in px"},
			cliapp.Flag{Name: "width", Description: "Width in px"},
			cliapp.Flag{Name: "height", Description: "Height in px"},
			cliapp.Flag{Name: "gravity", Description: "Anchor for an anchored crop"},
		),
		cmd("rotate", "Rotate by an angle (90° steps lossless)", h.rotateParams,
			cliapp.Flag{Name: "angle", Description: "Degrees counter-clockwise"},
			cliapp.Flag{Name: "background", Description: "Fill color for exposed corners (hex)"},
		),
		cmd("flip", "Mirror horizontally or vertically", h.flipParams,
			cliapp.Flag{Name: "axis", Description: "horizontal|vertical"},
		),
		cmd("deskew", "Auto-straighten a skewed scan/document", h.deskewParams,
			cliapp.Flag{Name: "background", Description: "Fill color for exposed corners (hex)"},
		),
		cmd("thumbnail", "Produce a fill-cropped thumbnail", h.thumbnailParams,
			cliapp.Flag{Name: "width", Description: "Thumbnail width in px"},
			cliapp.Flag{Name: "height", Description: "Thumbnail height in px"},
		),
		cmd("canvas", "Pad/extend onto a background canvas", h.canvasParams,
			cliapp.Flag{Name: "width", Description: "Canvas width in px"},
			cliapp.Flag{Name: "height", Description: "Canvas height in px"},
			cliapp.Flag{Name: "background", Description: "Background color (hex)"},
			cliapp.Flag{Name: "gravity", Description: "Where to place the image"},
		),
		cmd("adjust", "Adjust brightness/contrast/gamma/saturation/hue", h.adjustParams,
			cliapp.Flag{Name: "brightness", Description: "Brightness delta percent"},
			cliapp.Flag{Name: "contrast", Description: "Contrast delta percent"},
			cliapp.Flag{Name: "gamma", Description: "Gamma (1.0 = identity)"},
			cliapp.Flag{Name: "saturation", Description: "Saturation delta percent"},
			cliapp.Flag{Name: "hue", Description: "Hue rotation in degrees"},
		),
		cmd("filter", "Apply grayscale/sepia/invert/blur/sharpen", h.filterParams,
			cliapp.Flag{Name: "filter", Description: "grayscale|sepia|invert|blur|sharpen"},
			cliapp.Flag{Name: "amount", Description: "Sigma for blur/sharpen"},
		),
		cmd("duotone", "Map perceptual lightness onto a two- or three-ink ramp", h.duotoneParams,
			cliapp.Flag{Name: "dark", Description: "Dark ink hex color"}, cliapp.Flag{Name: "light", Description: "Light ink hex color"}, cliapp.Flag{Name: "mid", Description: "Optional mid ink hex color"}, cliapp.Flag{Name: "mid-low", Description: "Mid ink lower luminance band"}, cliapp.Flag{Name: "mid-high", Description: "Mid ink upper luminance band"}),
		cmd("posterize", "Quantize perceptual lightness to fixed levels", h.posterizeParams,
			cliapp.Flag{Name: "levels", Description: "Number of levels (2-256)"}, cliapp.Flag{Name: "dark", Description: "Dark ink hex color"}, cliapp.Flag{Name: "light", Description: "Light ink hex color"}),
		cmd("halftone", "Render perceptual lightness on a rotated dot screen", h.halftoneParams,
			cliapp.Flag{Name: "lpi", Description: "Screen lines across the image width (resolution-independent)"}, cliapp.Flag{Name: "angle", Description: "Screen angle"}, cliapp.Flag{Name: "dot", Description: "circle|square"}, cliapp.Flag{Name: "dark", Description: "Dark ink hex color"}, cliapp.Flag{Name: "light", Description: "Light ink hex color"}),
		cmd("dither_ordered", "Apply Bayer ordered dither", h.ditherOrderedParams, cliapp.Flag{Name: "dark", Description: "Dark ink hex color"}, cliapp.Flag{Name: "light", Description: "Light ink hex color"}),
		cmd("dither_diffusion", "Apply Floyd-Steinberg error diffusion", h.ditherDiffusionParams, cliapp.Flag{Name: "dark", Description: "Dark ink hex color"}, cliapp.Flag{Name: "light", Description: "Light ink hex color"}),
		cmd("grain", "Add seeded film grain", h.grainParams, cliapp.Flag{Name: "seed", Description: "Deterministic seed"}, cliapp.Flag{Name: "amount", Description: "Noise amount 0..1"}, cliapp.Flag{Name: "contrast-multiplier", Description: "Contrast multiplier"}),
		cmd("scrim", "Apply a directional contrast scrim", h.scrimParams, cliapp.Flag{Name: "color", Description: "Scrim hex color"}, cliapp.Flag{Name: "opacity", Description: "Maximum opacity 0..1"}, cliapp.Flag{Name: "direction", Description: "top|bottom|left|right"}),
		cmd("convert", "Convert to another image format", h.convertParams,
			cliapp.Flag{Name: "format", Description: "png|jpeg|gif|webp|tiff|bmp|avif"},
			cliapp.Flag{Name: "quality", Description: "Quality 1-100 for lossy formats"},
			cliapp.Flag{Name: "lossless", Bool: true, Description: "Lossless encoding (webp)"},
		),
		cmd("compress", "Re-encode at a quality or to a target size", h.compressParams,
			cliapp.Flag{Name: "format", Description: "Target lossy format"},
			cliapp.Flag{Name: "quality", Description: "Quality 1-100"},
			cliapp.Flag{Name: "lossless", Bool: true, Description: "Lossless encoding (webp)"},
			cliapp.Flag{Name: "target-bytes", Description: "Compress to <= this many bytes"},
		),
		cmd("overlay", "Composite a text or image watermark", h.overlayParams,
			cliapp.Flag{Name: "text", Description: "Text watermark"},
			cliapp.Flag{Name: "overlay", Description: "Path to an image watermark"},
			cliapp.Flag{Name: "position", Description: "Anchor (bottom-right, center, …)"},
			cliapp.Flag{Name: "opacity", Description: "Image overlay opacity 0..1"},
			cliapp.Flag{Name: "color", Description: "Text color (hex)"},
			cliapp.Flag{Name: "font-size", Description: "Text size in px"},
		),
		cmd("metadata", "Read, strip, or auto-orient metadata", h.metadataParams,
			cliapp.Flag{Name: "strip-all", Bool: true, Description: "Strip all metadata"},
			cliapp.Flag{Name: "strip-gps", Bool: true, Description: "Strip metadata (removes GPS)"},
			cliapp.Flag{Name: "auto-orient", Bool: true, Description: "Apply EXIF orientation to pixels"},
		),
	}
	commands = append(commands,
		cmd("line_screen", "Apply a lightness-modulated line screen", h.lineScreenParams, cliapp.Flag{Name: "spacing", Description: "Line spacing in pixels"}, cliapp.Flag{Name: "spacing-rel", Description: "Line spacing as a fraction of the short edge; wins over --spacing"}, cliapp.Flag{Name: "angle", Description: "Screen angle in degrees"}),
		cmd("stipple", "Apply a seeded jittered stipple screen", h.stippleParams, cliapp.Flag{Name: "spacing", Description: "Stipple spacing in pixels"}, cliapp.Flag{Name: "spacing-rel", Description: "Stipple spacing as a fraction of the short edge; wins over --spacing"}, cliapp.Flag{Name: "seed", Description: "Deterministic seed"}),
		cmd("engraving", "Render tonal hatching and cross-hatching", h.engravingParams, cliapp.Flag{Name: "spacing", Description: "Hatch spacing in pixels"}, cliapp.Flag{Name: "spacing-rel", Description: "Hatch spacing as a fraction of the short edge; wins over --spacing"}),
		cmd("aberration", "Separate channels radially at image edges", h.aberrationParams, cliapp.Flag{Name: "amplitude", Description: "Channel separation in pixels"}, cliapp.Flag{Name: "distance-rel", Description: "Channel separation as a fraction of the short edge; wins over --amplitude"}),
		cmd("bloom", "Lift bright pixels into a soft bloom", h.bloomParams, cliapp.Flag{Name: "radius", Description: "Bloom radius in pixels"}, cliapp.Flag{Name: "radius-rel", Description: "Bloom radius as a fraction of the short edge; wins over --radius"}, cliapp.Flag{Name: "threshold", Description: "Highlight threshold 0..1"}),
		cmd("curve", "Apply a deterministic tonal curve", h.curveParams, cliapp.Flag{Name: "exponent", Description: "Curve exponent"}),
		cmd("defocus", "Apply aperture-style defocus blur", h.defocusParams, cliapp.Flag{Name: "radius", Description: "Blur radius in pixels"}, cliapp.Flag{Name: "radius-rel", Description: "Blur radius as a fraction of the short edge; wins over --radius"}, cliapp.Flag{Name: "blade-count", Description: "Aperture blade count"}),
		cmd("motion_blur", "Blur along a declared motion vector", h.motionBlurParams, cliapp.Flag{Name: "distance", Description: "Blur distance in pixels"}, cliapp.Flag{Name: "distance-rel", Description: "Blur distance as a fraction of the short edge; wins over --distance"}, cliapp.Flag{Name: "angle", Description: "Motion angle in degrees"}),
		cmd("ascii_mosaic", "Rebuild the image as tone-matched ASCII glyphs", h.asciiMosaicParams, cliapp.Flag{Name: "block-size", Description: "Mosaic block size in pixels"}, cliapp.Flag{Name: "block-size-rel", Description: "Block size as a fraction of the short edge, snapped to the 7px glyph advance"}),
		cmd("pixel_sort", "Sort bright runs along an axis", h.pixelSortParams, cliapp.Flag{Name: "threshold", Description: "Luminance threshold 0..1"}, cliapp.Flag{Name: "axis", Description: "horizontal|vertical"}),
		cmd("displacement", "Offset pixels through a deterministic field", h.displacementParams, cliapp.Flag{Name: "amplitude", Description: "Displacement amplitude in pixels"}, cliapp.Flag{Name: "amplitude-rel", Description: "Displacement amplitude as a fraction of the short edge; wins over --amplitude"}, cliapp.Flag{Name: "spacing", Description: "Displacement field wavelength in pixels"}, cliapp.Flag{Name: "spacing-rel", Description: "Field wavelength as a fraction of the short edge; wins over --spacing"}, cliapp.Flag{Name: "seed", Description: "Deterministic seed"}),
	)
	return commands
}
