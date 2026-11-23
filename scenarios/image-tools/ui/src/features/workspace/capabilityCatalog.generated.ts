// AUTO-GENERATED — do not edit by hand.
//
// Source : api/internal/capability/manifests.seed.json
// Codegen: api/cmd/gen-capability (invoked by `make capability-gen`)
// Drift  : api/internal/capability/render_test.go fails the build if this file is stale.
//
// One entry per operation across both operation registries. The UI uses it to
// pick a family icon and a fallback description for operations that do not yet
// have a hand-written presentation entry.

export type CapabilityEngine = "ai" | "analysis" | "compute" | "unwired";

export interface ExtentStageMeta {
  readonly id: string;
  readonly kind: string;
  readonly labelKey: string;
  readonly descKey: string;
  readonly replaces?: string;
  readonly strengthParam?: string;
  readonly option?: ExtentOptionMeta;
}

/** A stage's choice of how it runs (skin tone toward the source or toward the body). */
export interface ExtentOptionMeta {
  readonly param: string;
  readonly default: string;
  readonly values: readonly { readonly id: string; readonly labelKey: string; readonly descKey: string }[];
}

export interface ExtentPresetMeta {
  readonly id: string;
  readonly labelKey: string;
  readonly descKey: string;
  readonly stages: readonly string[];
}

/** How much of a person an operation carries over: its stages and the presets composed from them. */
export interface ExtentMeta {
  readonly stages: readonly ExtentStageMeta[];
  readonly presets: readonly ExtentPresetMeta[];
  readonly defaultPreset: string;
}

export interface CapabilityMeta {
  readonly id: string;
  readonly engine: CapabilityEngine;
  readonly family: string;
  readonly mechanism: string;
  readonly inputKinds: readonly string[];
  readonly outputKind: string;
  readonly summary: string;
  readonly icon?: string;
  readonly labelKey?: string;
  readonly descKey?: string;
  readonly panel?: string;
  readonly extent?: ExtentMeta;
}

export const CAPABILITIES: Readonly<Record<string, CapabilityMeta>> = {
  "text_to_image": { id: "text_to_image", engine: "ai", family: "generate", mechanism: "infer", inputKinds: ["text"], outputKind: "image", summary: "Generate an image from a text prompt", panel: "create", icon: "ImagePlus", labelKey: "workspace.createOp.text_to_image.label", descKey: "workspace.createOp.text_to_image.desc" },
  "image_to_image": { id: "image_to_image", engine: "ai", family: "edit", mechanism: "infer", inputKinds: ["image", "text"], outputKind: "image", summary: "Transform an input image guided by a prompt", panel: "create", icon: "Replace", labelKey: "workspace.createOp.image_to_image.label", descKey: "workspace.createOp.image_to_image.desc" },
  "edit_instruct": { id: "edit_instruct", engine: "ai", family: "edit", mechanism: "infer", inputKinds: ["image", "text"], outputKind: "image", summary: "Edit an image from a natural-language instruction (identity-preserving)", panel: "create", icon: "Wand", labelKey: "workspace.createOp.edit_instruct.label", descKey: "workspace.createOp.edit_instruct.desc" },
  "inpaint": { id: "inpaint", engine: "ai", family: "edit", mechanism: "infer", inputKinds: ["image", "mask", "text"], outputKind: "image", summary: "Regenerate a masked region from a prompt", panel: "create", icon: "Stamp", labelKey: "workspace.createOp.inpaint.label", descKey: "workspace.createOp.inpaint.desc" },
  "outpaint": { id: "outpaint", engine: "ai", family: "edit", mechanism: "infer", inputKinds: ["image", "mask", "text"], outputKind: "image", summary: "Expand an image beyond its borders, generating the new region from a prompt", panel: "create", icon: "Expand", labelKey: "workspace.createOp.outpaint.label", descKey: "workspace.createOp.outpaint.desc" },
  "object_removal": { id: "object_removal", engine: "ai", family: "edit", mechanism: "infer", inputKinds: ["image", "mask"], outputKind: "image", summary: "Remove a masked object and fill the gap", panel: "create", icon: "Eraser", labelKey: "workspace.createOp.object_removal.label", descKey: "workspace.createOp.object_removal.desc" },
  "background_removal": { id: "background_removal", engine: "ai", family: "extract", mechanism: "infer", inputKinds: ["image"], outputKind: "image", summary: "Remove the background to transparency", panel: "enhance", icon: "Scissors", labelKey: "workspace.aiOp.background_removal.label", descKey: "workspace.aiOp.background_removal.desc" },
  "background_replace": { id: "background_replace", engine: "ai", family: "edit", mechanism: "infer", inputKinds: ["image", "mask", "text"], outputKind: "image", summary: "Replace the background behind a masked subject from a prompt", panel: "create", icon: "Images", labelKey: "workspace.createOp.background_replace.label", descKey: "workspace.createOp.background_replace.desc" },
  "face_swap": { id: "face_swap", engine: "ai", family: "generate", mechanism: "infer", inputKinds: ["image", "image"], outputKind: "image", summary: "Put the face from a reference image onto a target image (SFW only; local models, with a consent affirmation for non-local routes)", panel: "create", icon: "ScanFace", labelKey: "workspace.createOp.face_swap.label", descKey: "workspace.createOp.face_swap.desc", extent: { stages: [{ id: "face", kind: "swap", labelKey: "workspace.extent.stage.face.label", descKey: "workspace.extent.stage.face.desc", strengthParam: "face_strength" }, { id: "skin", kind: "recolour", labelKey: "workspace.extent.stage.skin.label", descKey: "workspace.extent.stage.skin.desc", option: { param: "skin_target", default: "source", values: [{ id: "body", labelKey: "workspace.extent.skinTarget.body.label", descKey: "workspace.extent.skinTarget.body.desc" }, { id: "source", labelKey: "workspace.extent.skinTarget.source.label", descKey: "workspace.extent.skinTarget.source.desc" }] } }, { id: "hair_style", kind: "restyle", labelKey: "workspace.extent.stage.hair_style.label", descKey: "workspace.extent.stage.hair_style.desc" }, { id: "hair_colour", kind: "recolour", labelKey: "workspace.extent.stage.hair_colour.label", descKey: "workspace.extent.stage.hair_colour.desc" }, { id: "head", kind: "swap", labelKey: "workspace.extent.stage.head.label", descKey: "workspace.extent.stage.head.desc", replaces: "face" }], presets: [{ id: "face", labelKey: "workspace.extent.preset.face.label", descKey: "workspace.extent.preset.face.desc", stages: ["face"] }, { id: "face_skin", labelKey: "workspace.extent.preset.face_skin.label", descKey: "workspace.extent.preset.face_skin.desc", stages: ["face", "skin"] }, { id: "full_look", labelKey: "workspace.extent.preset.full_look.label", descKey: "workspace.extent.preset.full_look.desc", stages: ["face", "skin", "hair_colour"] }, { id: "whole_head", labelKey: "workspace.extent.preset.whole_head.label", descKey: "workspace.extent.preset.whole_head.desc", stages: ["head"] }], defaultPreset: "face" } },
  "garment_replace": { id: "garment_replace", engine: "ai", family: "edit", mechanism: "infer", inputKinds: ["image", "mask", "text"], outputKind: "image", summary: "Replace or restyle clothing, hair, or makeup on a masked region from a prompt", panel: "create", icon: "Shirt", labelKey: "workspace.createOp.garment_replace.label", descKey: "workspace.createOp.garment_replace.desc" },
  "color_transfer": { id: "color_transfer", engine: "ai", family: "edit", mechanism: "compute", inputKinds: ["image", "mask", "image"], outputKind: "image", summary: "Recolour a masked region of an image to match the colour of a reference (skin tone, hair colour), keeping its own shading and texture", icon: "Palette", labelKey: "workspace.aiOp.color_transfer.label", descKey: "workspace.aiOp.color_transfer.desc" },
  "pose_chart_split": { id: "pose_chart_split", engine: "ai", family: "extract", mechanism: "compute", inputKinds: ["image"], outputKind: "data", summary: "Find the poses on a pose chart (a sheet of figures in a grid or any other arrangement) and return each pose's cell", icon: "LayoutGrid", labelKey: "workspace.aiOp.pose_chart_split.label", descKey: "workspace.aiOp.pose_chart_split.desc" },
  "pose_extract": { id: "pose_extract", engine: "ai", family: "extract", mechanism: "infer", inputKinds: ["image"], outputKind: "data", summary: "Read the pose of the people in an image as body and hand keypoints (photos and line-art figures)", icon: "PersonStanding", labelKey: "workspace.aiOp.pose_extract.label", descKey: "workspace.aiOp.pose_extract.desc" },
  "pose_generate": { id: "pose_generate", engine: "ai", family: "generate", mechanism: "infer", inputKinds: ["text", "image", "image"], outputKind: "image", summary: "Put a character, written or shown in a reference picture, in the pose of a photo, a line-art figure or one cell of a pose chart", panel: "create", icon: "PersonStanding", labelKey: "workspace.createOp.pose_generate.label", descKey: "workspace.createOp.pose_generate.desc" },
  "mask_composite": { id: "mask_composite", engine: "ai", family: "edit", mechanism: "compute", inputKinds: ["image", "mask", "image"], outputKind: "image", summary: "Lay a second image into the first through a mask, with a grown, feathered seam (the changed hair of an edited photo onto the original)", icon: "Layers", labelKey: "workspace.aiOp.mask_composite.label", descKey: "workspace.aiOp.mask_composite.desc" },
  "upscale": { id: "upscale", engine: "ai", family: "enhance", mechanism: "infer", inputKinds: ["image"], outputKind: "image", summary: "Super-resolve / enlarge an image", panel: "enhance", icon: "Sparkles", labelKey: "workspace.aiOp.upscale.label", descKey: "workspace.aiOp.upscale.desc" },
  "denoise": { id: "denoise", engine: "ai", family: "enhance", mechanism: "infer", inputKinds: ["image"], outputKind: "image", summary: "Reduce noise / deblur an image", panel: "enhance", icon: "Aperture", labelKey: "workspace.aiOp.denoise.label", descKey: "workspace.aiOp.denoise.desc" },
  "deblur": { id: "deblur", engine: "ai", family: "restore", mechanism: "infer", inputKinds: ["image"], outputKind: "image", summary: "Sharpen and remove motion/defocus blur from an image", panel: "enhance" },
  "naturalize": { id: "naturalize", engine: "ai", family: "enhance", mechanism: "compute", inputKinds: ["image"], outputKind: "image", summary: "Reintroduce realistic texture/grain to over-smoothed (restored/upscaled) images", panel: "enhance", icon: "Fingerprint", labelKey: "workspace.aiOp.naturalize.label", descKey: "workspace.aiOp.naturalize.desc" },
  "colorize": { id: "colorize", engine: "ai", family: "enhance", mechanism: "infer", inputKinds: ["image"], outputKind: "image", summary: "Add realistic colour to a grayscale / black-and-white image", panel: "enhance", icon: "Palette", labelKey: "workspace.aiOp.colorize.label", descKey: "workspace.aiOp.colorize.desc" },
  "old_photo_restore": { id: "old_photo_restore", engine: "ai", family: "restore", mechanism: "infer", inputKinds: ["image"], outputKind: "image", summary: "Repair scratches, fading, and damage on old photographs", panel: "enhance" },
  "face_restore": { id: "face_restore", engine: "ai", family: "restore", mechanism: "infer", inputKinds: ["image"], outputKind: "image", summary: "Restore degraded faces in a photo", panel: "enhance" },
  "segment": { id: "segment", engine: "ai", family: "extract", mechanism: "infer", inputKinds: ["image"], outputKind: "mask", summary: "Segment objects/regions in an image" },
  "depth_map": { id: "depth_map", engine: "ai", family: "extract", mechanism: "infer", inputKinds: ["image"], outputKind: "image", summary: "Estimate a per-pixel depth map from a single image", panel: "enhance", icon: "Layers", labelKey: "workspace.aiOp.depth_map.label", descKey: "workspace.aiOp.depth_map.desc" },
  "scene_geometry": { id: "scene_geometry", engine: "ai", family: "extract", mechanism: "infer", inputKinds: ["image"], outputKind: "data", summary: "Estimate a camera-calibrated floating-point scene geometry bundle", panel: "enhance" },
  "image_to_object": { id: "image_to_object", engine: "ai", family: "extract", mechanism: "infer", inputKinds: ["image"], outputKind: "mesh", summary: "Reconstruct a textured 3D prop mesh from one image and optional foreground mask", panel: "enhance" },
  "normal_map": { id: "normal_map", engine: "ai", family: "enhance", mechanism: "compute", inputKinds: ["image"], outputKind: "image", summary: "Convert image luminance/depth into a tangent-space normal map", panel: "enhance" },
  "ocr": { id: "ocr", engine: "analysis", family: "extract", mechanism: "infer", inputKinds: ["image"], outputKind: "data", summary: "Extract text from an image (OCR)", icon: "ScanText", labelKey: "workspace.analyzeOp.ocr.label", descKey: "workspace.analyzeOp.ocr.desc" },
  "nsfw_classify": { id: "nsfw_classify", engine: "analysis", family: "extract", mechanism: "infer", inputKinds: ["image"], outputKind: "data", summary: "Classify an image for NSFW / unsafe content", icon: "ShieldAlert", labelKey: "workspace.analyzeOp.nsfw_classify.label", descKey: "workspace.analyzeOp.nsfw_classify.desc" },
  "caption": { id: "caption", engine: "ai", family: "extract", mechanism: "infer", inputKinds: ["image"], outputKind: "text", summary: "Describe an image in natural language", panel: "analyze", icon: "Image", labelKey: "workspace.analyzeOp.caption.label", descKey: "workspace.analyzeOp.caption.desc" },
  "object_detection": { id: "object_detection", engine: "ai", family: "extract", mechanism: "infer", inputKinds: ["image"], outputKind: "data", summary: "Detect and locate objects with bounding boxes" },
  "tagging": { id: "tagging", engine: "ai", family: "extract", mechanism: "infer", inputKinds: ["image"], outputKind: "data", summary: "Predict descriptive tags/labels for an image" },
  "face_detection": { id: "face_detection", engine: "unwired", family: "extract", mechanism: "infer", inputKinds: ["image"], outputKind: "data", summary: "Detect face bounding boxes / landmarks" },
  "age_estimate": { id: "age_estimate", engine: "unwired", family: "extract", mechanism: "infer", inputKinds: ["image"], outputKind: "data", summary: "Estimate the apparent age range of one face crop as nine age bins (the minor-safety reviewer's age stage; no identity)" },
  "quality_assessment": { id: "quality_assessment", engine: "analysis", family: "extract", mechanism: "compute", inputKinds: ["image"], outputKind: "data", summary: "Assess no-reference image quality (sharpness, exposure, contrast)", icon: "Gauge", labelKey: "workspace.analyzeOp.quality_assessment.label", descKey: "workspace.analyzeOp.quality_assessment.desc" },
  "probe": { id: "probe", engine: "analysis", family: "extract", mechanism: "compute", inputKinds: ["image"], outputKind: "data", summary: "Report structured image info (dimensions, format, metadata, palette)", icon: "ScanSearch", labelKey: "workspace.analyzeOp.probe.label", descKey: "workspace.analyzeOp.probe.desc" },
  "duplicate_detect": { id: "duplicate_detect", engine: "analysis", family: "extract", mechanism: "compute", inputKinds: ["image"], outputKind: "data", summary: "Compute perceptual fingerprints to find near-duplicate images", icon: "Copy", labelKey: "workspace.analyzeOp.duplicate_detect.label", descKey: "workspace.analyzeOp.duplicate_detect.desc" },
  "embedding": { id: "embedding", engine: "ai", family: "extract", mechanism: "infer", inputKinds: ["image"], outputKind: "data", summary: "Compute a vector embedding of an image" },
  "qr_barcode_read": { id: "qr_barcode_read", engine: "unwired", family: "extract", mechanism: "compute", inputKinds: ["image"], outputKind: "data", summary: "Read QR codes / barcodes from an image" },
  "aberration": { id: "aberration", engine: "compute", family: "edit", mechanism: "compute", inputKinds: ["image"], outputKind: "image", summary: "Separate color channels at edges" },
  "adjust": { id: "adjust", engine: "compute", family: "edit", mechanism: "compute", inputKinds: ["image"], outputKind: "image", summary: "Brightness/contrast/gamma/saturation/hue", icon: "SlidersHorizontal", labelKey: "workspace.op.adjust.label", descKey: "workspace.op.adjust.desc" },
  "ascii_mosaic": { id: "ascii_mosaic", engine: "compute", family: "edit", mechanism: "compute", inputKinds: ["image"], outputKind: "image", summary: "Rebuild the image as tone-matched ASCII glyphs" },
  "bloom": { id: "bloom", engine: "compute", family: "edit", mechanism: "compute", inputKinds: ["image"], outputKind: "image", summary: "Lift highlights into a soft bloom" },
  "canny": { id: "canny", engine: "compute", family: "extract", mechanism: "compute", inputKinds: ["image"], outputKind: "image", summary: "Deterministic Canny edge map (ControlNet conditioning preprocessor)" },
  "canvas": { id: "canvas", engine: "compute", family: "edit", mechanism: "compute", inputKinds: ["image"], outputKind: "image", summary: "Pad/extend onto a background canvas", icon: "Frame", labelKey: "workspace.op.canvas.label", descKey: "workspace.op.canvas.desc" },
  "composite": { id: "composite", engine: "compute", family: "compose", mechanism: "compute", inputKinds: ["image"], outputKind: "image", summary: "Merge an ordered stack of depth plates into one raster" },
  "compress": { id: "compress", engine: "compute", family: "edit", mechanism: "compute", inputKinds: ["image"], outputKind: "image", summary: "Re-encode at a quality or to a target file size", icon: "Minimize2", labelKey: "workspace.op.compress.label", descKey: "workspace.op.compress.desc" },
  "convert": { id: "convert", engine: "compute", family: "edit", mechanism: "compute", inputKinds: ["image"], outputKind: "image", summary: "Convert to another image format", icon: "FileType2", labelKey: "workspace.op.convert.label", descKey: "workspace.op.convert.desc" },
  "crop": { id: "crop", engine: "compute", family: "edit", mechanism: "compute", inputKinds: ["image"], outputKind: "image", summary: "Crop a rectangle or a gravity-anchored region", icon: "Crop", labelKey: "workspace.op.crop.label", descKey: "workspace.op.crop.desc" },
  "curve": { id: "curve", engine: "compute", family: "edit", mechanism: "compute", inputKinds: ["image"], outputKind: "image", summary: "Apply a deterministic tonal curve" },
  "defocus": { id: "defocus", engine: "compute", family: "edit", mechanism: "compute", inputKinds: ["image"], outputKind: "image", summary: "Apply a deterministic aperture blur" },
  "deskew": { id: "deskew", engine: "compute", family: "edit", mechanism: "compute", inputKinds: ["image"], outputKind: "image", summary: "Auto-straighten a skewed scan/document", icon: "Ruler", labelKey: "workspace.op.deskew.label", descKey: "workspace.op.deskew.desc" },
  "displacement": { id: "displacement", engine: "compute", family: "edit", mechanism: "compute", inputKinds: ["image"], outputKind: "image", summary: "Offset pixels through a deterministic displacement map" },
  "dither_diffusion": { id: "dither_diffusion", engine: "compute", family: "edit", mechanism: "compute", inputKinds: ["image"], outputKind: "image", summary: "Apply deterministic Floyd-Steinberg error diffusion" },
  "dither_ordered": { id: "dither_ordered", engine: "compute", family: "edit", mechanism: "compute", inputKinds: ["image"], outputKind: "image", summary: "Apply a deterministic Bayer ordered dither" },
  "duotone": { id: "duotone", engine: "compute", family: "edit", mechanism: "compute", inputKinds: ["image"], outputKind: "image", summary: "Map perceptual lightness onto a deterministic two- or three-ink ramp" },
  "engraving": { id: "engraving", engine: "compute", family: "edit", mechanism: "compute", inputKinds: ["image"], outputKind: "image", summary: "Render tonal value as hatching density" },
  "filter": { id: "filter", engine: "compute", family: "edit", mechanism: "compute", inputKinds: ["image"], outputKind: "image", summary: "Grayscale/sepia/invert/blur/sharpen", icon: "Wand2", labelKey: "workspace.op.filter.label", descKey: "workspace.op.filter.desc" },
  "flip": { id: "flip", engine: "compute", family: "edit", mechanism: "compute", inputKinds: ["image"], outputKind: "image", summary: "Mirror horizontally or vertically", icon: "FlipHorizontal2", labelKey: "workspace.op.flip.label", descKey: "workspace.op.flip.desc" },
  "grain": { id: "grain", engine: "compute", family: "edit", mechanism: "compute", inputKinds: ["image"], outputKind: "image", summary: "Add seeded film grain" },
  "halftone": { id: "halftone", engine: "compute", family: "edit", mechanism: "compute", inputKinds: ["image"], outputKind: "image", summary: "Render perceptual lightness on a rotated dot screen" },
  "icon_container": { id: "icon_container", engine: "compute", family: "edit", mechanism: "compute", inputKinds: ["image"], outputKind: "image", summary: "Pack PNG renders into an ICO or ICNS container" },
  "line_screen": { id: "line_screen", engine: "compute", family: "edit", mechanism: "compute", inputKinds: ["image"], outputKind: "image", summary: "Render perceptual lightness as a line screen" },
  "metadata": { id: "metadata", engine: "compute", family: "extract", mechanism: "compute", inputKinds: ["image"], outputKind: "data", summary: "Read, strip, or auto-orient image metadata", icon: "Info", labelKey: "workspace.op.metadata.label", descKey: "workspace.op.metadata.desc" },
  "motion_blur": { id: "motion_blur", engine: "compute", family: "edit", mechanism: "compute", inputKinds: ["image"], outputKind: "image", summary: "Blur along a deterministic motion axis" },
  "overlay": { id: "overlay", engine: "compute", family: "compose", mechanism: "compute", inputKinds: ["image"], outputKind: "image", summary: "Composite a watermark image or text", icon: "Type", labelKey: "workspace.op.overlay.label", descKey: "workspace.op.overlay.desc" },
  "parallax": { id: "parallax", engine: "compute", family: "compose", mechanism: "compute", inputKinds: ["image"], outputKind: "image", summary: "Render a bounded layered-depth parallax PNG sequence with provenance" },
  "pixel_sort": { id: "pixel_sort", engine: "compute", family: "edit", mechanism: "compute", inputKinds: ["image"], outputKind: "image", summary: "Sort bright runs by lightness along an axis" },
  "posterize": { id: "posterize", engine: "compute", family: "edit", mechanism: "compute", inputKinds: ["image"], outputKind: "image", summary: "Quantize perceptual lightness to a fixed number of levels" },
  "rasterize": { id: "rasterize", engine: "compute", family: "edit", mechanism: "compute", inputKinds: ["vector"], outputKind: "image", summary: "Rasterize an SVG at an exact pixel size" },
  "resize": { id: "resize", engine: "compute", family: "edit", mechanism: "compute", inputKinds: ["image"], outputKind: "image", summary: "Resize (fit/fill/stretch) preserving or forcing aspect", icon: "Maximize2", labelKey: "workspace.op.resize.label", descKey: "workspace.op.resize.desc" },
  "rotate": { id: "rotate", engine: "compute", family: "edit", mechanism: "compute", inputKinds: ["image"], outputKind: "image", summary: "Rotate by 90° steps or an arbitrary angle", icon: "RotateCw", labelKey: "workspace.op.rotate.label", descKey: "workspace.op.rotate.desc" },
  "screenshot_cleanup": { id: "screenshot_cleanup", engine: "compute", family: "restore", mechanism: "compute", inputKinds: ["image"], outputKind: "image", summary: "Trim uniform screenshot chrome, frame, and letterbox bands", icon: "Crop", labelKey: "workspace.op.screenshot_cleanup.label", descKey: "workspace.op.screenshot_cleanup.desc" },
  "scrim": { id: "scrim", engine: "compute", family: "edit", mechanism: "compute", inputKinds: ["image"], outputKind: "image", summary: "Apply a directional contrast scrim" },
  "stipple": { id: "stipple", engine: "compute", family: "edit", mechanism: "compute", inputKinds: ["image"], outputKind: "image", summary: "Render seeded jittered stipple" },
  "thumbnail": { id: "thumbnail", engine: "compute", family: "edit", mechanism: "compute", inputKinds: ["image"], outputKind: "image", summary: "Produce a fill-cropped thumbnail", icon: "Image", labelKey: "workspace.op.thumbnail.label", descKey: "workspace.op.thumbnail.desc" },
  "vectorize": { id: "vectorize", engine: "compute", family: "edit", mechanism: "compute", inputKinds: ["image"], outputKind: "vector", summary: "Trace a flat raster mark into a clean SVG" },
};

export const capabilityIds: readonly string[] = Object.keys(CAPABILITIES);

export const capabilityMeta = (id: string): CapabilityMeta | undefined => CAPABILITIES[id];
