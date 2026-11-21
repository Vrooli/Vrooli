// AUTO-GENERATED — do not edit by hand.
//
// Source : packages/proto/schemas/image-tools/v1/ops/ops.proto (OpParams)
// Codegen: api/cmd/gen-capability (invoked by `make capability-gen`)
// Drift  : api/internal/capability/render_ops_test.go fails the build if this file is stale.
//
// One spec per deterministic operation reachable over the REST run edge. The
// command construction, flag definitions, and parameter binding live in
// register.go and params.go; this file is shape only.

package ops

var generatedRunSpecs = []runSpec{
	{Op: "aberration", Desc: "Separate color channels at edges", Flags: []flagSpec{
		{Name: "amplitude", Field: "amplitude", Kind: flagDouble},
		{Name: "distance", Field: "distance", Kind: flagInt32},
		{Name: "distance-rel", Field: "distance_rel", Kind: flagDouble},
	}},
	{Op: "adjust", Desc: "Brightness/contrast/gamma/saturation/hue", Flags: []flagSpec{
		{Name: "brightness", Field: "brightness", Kind: flagDouble},
		{Name: "contrast", Field: "contrast", Kind: flagDouble},
		{Name: "gamma", Field: "gamma", Kind: flagDouble},
		{Name: "saturation", Field: "saturation", Kind: flagDouble},
		{Name: "hue", Field: "hue", Kind: flagDouble},
	}},
	{Op: "ascii_mosaic", Desc: "Rebuild the image as tone-matched ASCII glyphs", Flags: []flagSpec{
		{Name: "block-size", Field: "block_size", Kind: flagInt32},
		{Name: "dark", Field: "dark", Kind: flagString},
		{Name: "light", Field: "light", Kind: flagString},
		{Name: "normalize", Field: "normalize", Kind: flagBool},
		{Name: "block-size-rel", Field: "block_size_rel", Kind: flagDouble},
	}},
	{Op: "bloom", Desc: "Lift highlights into a soft bloom", Flags: []flagSpec{
		{Name: "radius", Field: "radius", Kind: flagInt32},
		{Name: "threshold", Field: "threshold", Kind: flagDouble},
		{Name: "radius-rel", Field: "radius_rel", Kind: flagDouble},
	}},
	{Op: "canny", Desc: "Deterministic Canny edge map (ControlNet conditioning preprocessor)", Flags: []flagSpec{
		{Name: "amount", Field: "amount", Kind: flagDouble},
		{Name: "low-threshold", Field: "low_threshold", Kind: flagDouble},
		{Name: "high-threshold", Field: "high_threshold", Kind: flagDouble},
	}},
	{Op: "canvas", Desc: "Pad/extend onto a background canvas", Flags: []flagSpec{
		{Name: "width", Field: "width", Kind: flagInt32},
		{Name: "height", Field: "height", Kind: flagInt32},
		{Name: "background", Field: "background", Kind: flagString},
		{Name: "gravity", Field: "gravity", Kind: flagString},
	}},
	{Op: "compress", Desc: "Re-encode at a quality or to a target file size", Flags: []flagSpec{
		{Name: "format", Field: "format", Kind: flagString},
		{Name: "quality", Field: "quality", Kind: flagInt32},
		{Name: "lossless", Field: "lossless", Kind: flagBool},
		{Name: "target-bytes", Field: "target_bytes", Kind: flagInt64},
	}},
	{Op: "convert", Desc: "Convert to another image format", Flags: []flagSpec{
		{Name: "format", Field: "format", Kind: flagString},
		{Name: "quality", Field: "quality", Kind: flagInt32},
		{Name: "lossless", Field: "lossless", Kind: flagBool},
	}},
	{Op: "crop", Desc: "Crop a rectangle or a gravity-anchored region", Flags: []flagSpec{
		{Name: "x", Field: "x", Kind: flagInt32},
		{Name: "y", Field: "y", Kind: flagInt32},
		{Name: "width", Field: "width", Kind: flagInt32},
		{Name: "height", Field: "height", Kind: flagInt32},
		{Name: "gravity", Field: "gravity", Kind: flagString},
	}},
	{Op: "curve", Desc: "Apply a deterministic tonal curve", Flags: []flagSpec{
		{Name: "exponent", Field: "exponent", Kind: flagDouble},
	}},
	{Op: "defocus", Desc: "Apply a deterministic aperture blur", Flags: []flagSpec{
		{Name: "radius", Field: "radius", Kind: flagInt32},
		{Name: "blade-count", Field: "blade_count", Kind: flagInt32},
		{Name: "radius-rel", Field: "radius_rel", Kind: flagDouble},
	}},
	{Op: "deskew", Desc: "Auto-straighten a skewed scan/document", Flags: []flagSpec{
		{Name: "background", Field: "background", Kind: flagString},
	}},
	{Op: "displacement", Desc: "Offset pixels through a deterministic displacement map", Flags: []flagSpec{
		{Name: "amplitude", Field: "amplitude", Kind: flagDouble},
		{Name: "seed", Field: "seed", Kind: flagInt64},
		{Name: "spacing", Field: "spacing", Kind: flagDouble},
		{Name: "spacing-rel", Field: "spacing_rel", Kind: flagDouble},
		{Name: "amplitude-rel", Field: "amplitude_rel", Kind: flagDouble},
	}},
	{Op: "dither_diffusion", Desc: "Apply deterministic Floyd-Steinberg error diffusion", Flags: []flagSpec{
		{Name: "dark", Field: "dark", Kind: flagString},
		{Name: "light", Field: "light", Kind: flagString},
		{Name: "normalize", Field: "normalize", Kind: flagBool},
	}},
	{Op: "dither_ordered", Desc: "Apply a deterministic Bayer ordered dither", Flags: []flagSpec{
		{Name: "dark", Field: "dark", Kind: flagString},
		{Name: "light", Field: "light", Kind: flagString},
		{Name: "normalize", Field: "normalize", Kind: flagBool},
	}},
	{Op: "duotone", Desc: "Map perceptual lightness onto a deterministic two- or three-ink ramp", Flags: []flagSpec{
		{Name: "dark", Field: "dark", Kind: flagString},
		{Name: "light", Field: "light", Kind: flagString},
		{Name: "mid", Field: "mid", Kind: flagString},
		{Name: "mid-low", Field: "mid_low", Kind: flagDouble},
		{Name: "mid-high", Field: "mid_high", Kind: flagDouble},
		{Name: "normalize", Field: "normalize", Kind: flagBool},
	}},
	{Op: "engraving", Desc: "Render tonal value as hatching density", Flags: []flagSpec{
		{Name: "spacing", Field: "spacing", Kind: flagDouble},
		{Name: "dark", Field: "dark", Kind: flagString},
		{Name: "light", Field: "light", Kind: flagString},
		{Name: "normalize", Field: "normalize", Kind: flagBool},
		{Name: "spacing-rel", Field: "spacing_rel", Kind: flagDouble},
	}},
	{Op: "filter", Desc: "Grayscale/sepia/invert/blur/sharpen", Flags: []flagSpec{
		{Name: "filter", Field: "filter", Kind: flagString},
		{Name: "amount", Field: "amount", Kind: flagDouble},
	}},
	{Op: "flip", Desc: "Mirror horizontally or vertically", Flags: []flagSpec{
		{Name: "axis", Field: "axis", Kind: flagString},
	}},
	{Op: "grain", Desc: "Add seeded film grain", Flags: []flagSpec{
		{Name: "seed", Field: "seed", Kind: flagInt64},
		{Name: "amount", Field: "amount", Kind: flagDouble},
		{Name: "contrast-multiplier", Field: "contrast_multiplier", Kind: flagDouble},
	}},
	{Op: "halftone", Desc: "Render perceptual lightness on a rotated dot screen", Flags: []flagSpec{
		{Name: "lpi", Field: "lpi", Kind: flagInt32},
		{Name: "angle", Field: "angle", Kind: flagDouble},
		{Name: "dot", Field: "dot", Kind: flagString},
		{Name: "dark", Field: "dark", Kind: flagString},
		{Name: "light", Field: "light", Kind: flagString},
		{Name: "normalize", Field: "normalize", Kind: flagBool},
	}},
	{Op: "icon_container", Desc: "Pack PNG renders into an ICO or ICNS container", Flags: []flagSpec{
		{Name: "format", Field: "format", Kind: flagString},
		{Name: "sizes", Field: "sizes", Kind: flagInt32List},
	}},
	{Op: "line_screen", Desc: "Render perceptual lightness as a line screen", Flags: []flagSpec{
		{Name: "spacing", Field: "spacing", Kind: flagDouble},
		{Name: "angle", Field: "angle", Kind: flagDouble},
		{Name: "dark", Field: "dark", Kind: flagString},
		{Name: "light", Field: "light", Kind: flagString},
		{Name: "normalize", Field: "normalize", Kind: flagBool},
		{Name: "spacing-rel", Field: "spacing_rel", Kind: flagDouble},
	}},
	{Op: "metadata", Desc: "Read, strip, or auto-orient image metadata", Flags: []flagSpec{
		{Name: "strip-all", Field: "strip_all", Kind: flagBool},
		{Name: "strip-gps", Field: "strip_gps", Kind: flagBool},
		{Name: "auto-orient", Field: "auto_orient", Kind: flagBool},
	}},
	{Op: "motion_blur", Desc: "Blur along a deterministic motion axis", Flags: []flagSpec{
		{Name: "distance", Field: "distance", Kind: flagInt32},
		{Name: "angle", Field: "angle", Kind: flagDouble},
		{Name: "distance-rel", Field: "distance_rel", Kind: flagDouble},
	}},
	{Op: "overlay", Desc: "Composite a watermark image or text", Flags: []flagSpec{
		{Name: "text", Field: "text", Kind: flagString},
		{Name: "position", Field: "position", Kind: flagString},
		{Name: "opacity", Field: "opacity", Kind: flagDouble},
		{Name: "color", Field: "color", Kind: flagString},
		{Name: "font-size", Field: "font_size", Kind: flagDouble},
	}},
	{Op: "pixel_sort", Desc: "Sort bright runs by lightness along an axis", Flags: []flagSpec{
		{Name: "threshold", Field: "threshold", Kind: flagDouble},
		{Name: "axis", Field: "axis", Kind: flagString},
	}},
	{Op: "posterize", Desc: "Quantize perceptual lightness to a fixed number of levels", Flags: []flagSpec{
		{Name: "levels", Field: "levels", Kind: flagInt32},
		{Name: "dark", Field: "dark", Kind: flagString},
		{Name: "light", Field: "light", Kind: flagString},
		{Name: "normalize", Field: "normalize", Kind: flagBool},
	}},
	{Op: "rasterize", Desc: "Rasterize an SVG at an exact pixel size", Flags: []flagSpec{
		{Name: "width", Field: "width", Kind: flagInt32},
		{Name: "height", Field: "height", Kind: flagInt32},
		{Name: "background", Field: "background", Kind: flagString},
	}},
	{Op: "resize", Desc: "Resize (fit/fill/stretch) preserving or forcing aspect", Flags: []flagSpec{
		{Name: "width", Field: "width", Kind: flagInt32},
		{Name: "height", Field: "height", Kind: flagInt32},
		{Name: "fit", Field: "fit", Kind: flagString},
		{Name: "gravity", Field: "gravity", Kind: flagString},
	}},
	{Op: "rotate", Desc: "Rotate by 90° steps or an arbitrary angle", Flags: []flagSpec{
		{Name: "angle", Field: "angle", Kind: flagDouble},
		{Name: "expand", Field: "expand", Kind: flagBool},
		{Name: "background", Field: "background", Kind: flagString},
	}},
	{Op: "screenshot_cleanup", Desc: "Trim uniform screenshot chrome, frame, and letterbox bands", Flags: []flagSpec{
		{Name: "tolerance", Field: "tolerance", Kind: flagDouble},
		{Name: "max-trim-percent", Field: "max_trim_percent", Kind: flagDouble},
		{Name: "padding-px", Field: "padding_px", Kind: flagInt32},
	}},
	{Op: "scrim", Desc: "Apply a directional contrast scrim", Flags: []flagSpec{
		{Name: "color", Field: "color", Kind: flagString},
		{Name: "opacity", Field: "opacity", Kind: flagDouble},
		{Name: "direction", Field: "direction", Kind: flagString},
		{Name: "region-x", Field: "region_x", Kind: flagDouble},
		{Name: "region-y", Field: "region_y", Kind: flagDouble},
		{Name: "region-width", Field: "region_width", Kind: flagDouble},
		{Name: "region-height", Field: "region_height", Kind: flagDouble},
		{Name: "region-feather", Field: "region_feather", Kind: flagDouble},
	}},
	{Op: "stipple", Desc: "Render seeded jittered stipple", Flags: []flagSpec{
		{Name: "spacing", Field: "spacing", Kind: flagDouble},
		{Name: "seed", Field: "seed", Kind: flagInt64},
		{Name: "dark", Field: "dark", Kind: flagString},
		{Name: "light", Field: "light", Kind: flagString},
		{Name: "normalize", Field: "normalize", Kind: flagBool},
		{Name: "spacing-rel", Field: "spacing_rel", Kind: flagDouble},
	}},
	{Op: "thumbnail", Desc: "Produce a fill-cropped thumbnail", Flags: []flagSpec{
		{Name: "width", Field: "width", Kind: flagInt32},
		{Name: "height", Field: "height", Kind: flagInt32},
	}},
	{Op: "vectorize", Desc: "Trace a flat raster mark into a clean SVG", Flags: []flagSpec{
		{Name: "colors", Field: "colors", Kind: flagInt32},
		{Name: "keep-colors", Field: "keep_colors", Kind: flagStringList},
		{Name: "drop-background-layers", Field: "drop_background_layers", Kind: flagBool},
		{Name: "clip-to-largest-rounded-region", Field: "clip_to_largest_rounded_region", Kind: flagBool},
		{Name: "inset-px", Field: "inset_px", Kind: flagDouble},
		{Name: "tolerance-px", Field: "tolerance_px", Kind: flagDouble},
		{Name: "smoothing", Field: "smoothing", Kind: flagBool},
		{Name: "min-area-px", Field: "min_area_px", Kind: flagDouble},
	}},
}
