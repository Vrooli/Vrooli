// AUTO-GENERATED — do not edit by hand.
//
// Source : api/internal/capability/manifests.seed.json
// Codegen: api/cmd/gen-capability (invoked by `make capability-gen`)
// Drift  : api/internal/capability/render_test.go fails the build if this file is stale.
//
// Data only: the submit command builder and flag definitions live in register.go.

package ai

var generatedSubmitSpecs = []submitSpec{
	{Verb: "generate", Op: "text_to_image", Desc: "Generate an image from a text prompt", NeedsInput: false, NeedsMask: false, NeedsReference: false, Flags: []string{"prompt", "negative", "seed", "width", "height", "steps", "cfg-scale", "variations", "auto-scan"}},
	{Verb: "img2img", Op: "image_to_image", Desc: "Transform an input image guided by a prompt", NeedsInput: true, NeedsMask: false, NeedsReference: false, Flags: []string{"prompt", "negative", "seed", "width", "height", "steps", "cfg-scale", "variations", "auto-scan", "strength"}},
	{Verb: "edit", Op: "edit_instruct", Desc: "Edit an image from a natural-language instruction (identity-preserving)", NeedsInput: true, NeedsMask: false, NeedsReference: false, Flags: []string{"prompt", "negative", "seed", "width", "height", "steps", "cfg-scale", "variations", "auto-scan", "strength"}},
	{Verb: "inpaint", Op: "inpaint", Desc: "Regenerate a masked region from a prompt", NeedsInput: true, NeedsMask: true, NeedsReference: false, Flags: []string{"prompt", "negative", "seed", "width", "height", "steps", "cfg-scale", "variations", "auto-scan", "mask", "crop-to-mask", "crop-context-pixels", "crop-max-edge-px", "neutralize-masked-pixels"}},
	{Verb: "outpaint", Op: "outpaint", Desc: "Expand an image beyond its borders, generating the new region from a prompt", NeedsInput: true, NeedsMask: true, NeedsReference: false, Flags: []string{"prompt", "negative", "seed", "width", "height", "steps", "cfg-scale", "variations", "auto-scan", "mask"}},
	{Verb: "object-removal", Op: "object_removal", Desc: "Remove a masked object and fill the gap", NeedsInput: true, NeedsMask: true, NeedsReference: false, Flags: []string{"mask", "mask-expansion", "remove-region"}},
	{Verb: "bg-removal", Op: "background_removal", Desc: "Remove the background to transparency", NeedsInput: true, NeedsMask: false, NeedsReference: false, Flags: []string{}},
	{Verb: "background-replace", Op: "background_replace", Desc: "Replace the background behind a masked subject from a prompt", NeedsInput: true, NeedsMask: true, NeedsReference: false, Flags: []string{"prompt", "negative", "seed", "width", "height", "steps", "cfg-scale", "variations", "auto-scan", "mask"}},
	{Verb: "face-swap", Op: "face_swap", Desc: "Put the face from a reference image onto a target image (SFW only; local models, with a consent affirmation for non-local routes)", NeedsInput: true, NeedsMask: false, NeedsReference: true, Flags: []string{"reference", "target-face", "source-face", "all-faces", "color-match", "no-occlusion", "extent", "match", "skin-target", "face-strength", "prompt", "seed", "auto-scan"}},
	{Verb: "garment-replace", Op: "garment_replace", Desc: "Replace or restyle clothing, hair, or makeup on a masked region from a prompt", NeedsInput: true, NeedsMask: true, NeedsReference: false, Flags: []string{"prompt", "negative", "seed", "width", "height", "steps", "cfg-scale", "variations", "auto-scan", "mask"}},
	{Verb: "color-transfer", Op: "color_transfer", Desc: "Recolour a masked region of an image to match the colour of a reference (skin tone, hair colour), keeping its own shading and texture", NeedsInput: true, NeedsMask: true, NeedsReference: true, Flags: []string{"mask", "reference", "strength", "luminance", "flatten", "feather-px"}},
	{Verb: "mask-composite", Op: "mask_composite", Desc: "Lay a second image into the first through a mask, with a grown, feathered seam (the changed hair of an edited photo onto the original)", NeedsInput: true, NeedsMask: true, NeedsReference: true, Flags: []string{"mask", "reference", "grow-px", "feather-px"}},
	{Verb: "upscale", Op: "upscale", Desc: "Super-resolve / enlarge an image", NeedsInput: true, NeedsMask: false, NeedsReference: false, Flags: []string{"scale"}},
	{Verb: "denoise", Op: "denoise", Desc: "Reduce noise / deblur an image", NeedsInput: true, NeedsMask: false, NeedsReference: false, Flags: []string{}},
	{Verb: "deblur", Op: "deblur", Desc: "Sharpen and remove motion/defocus blur from an image", NeedsInput: true, NeedsMask: false, NeedsReference: false, Flags: []string{}},
	{Verb: "naturalize", Op: "naturalize", Desc: "Reintroduce realistic texture/grain to over-smoothed (restored/upscaled) images", NeedsInput: true, NeedsMask: false, NeedsReference: false, Flags: []string{"realism", "face-aware"}},
	{Verb: "colorize", Op: "colorize", Desc: "Add realistic colour to a grayscale / black-and-white image", NeedsInput: true, NeedsMask: false, NeedsReference: false, Flags: []string{}},
	{Verb: "old-photo-restore", Op: "old_photo_restore", Desc: "Repair scratches, fading, and damage on old photographs", NeedsInput: true, NeedsMask: false, NeedsReference: false, Flags: []string{}},
	{Verb: "face-restore", Op: "face_restore", Desc: "Restore degraded faces in a photo", NeedsInput: true, NeedsMask: false, NeedsReference: false, Flags: []string{}},
	{Verb: "segment", Op: "segment", Desc: "Segment objects/regions in an image", NeedsInput: true, NeedsMask: false, NeedsReference: false, Flags: []string{"selection", "text"}},
	{Verb: "depth", Op: "depth_map", Desc: "Estimate a per-pixel depth map from a single image", NeedsInput: true, NeedsMask: false, NeedsReference: false, Flags: []string{}},
	{Verb: "geometry", Op: "scene_geometry", Desc: "Estimate a camera-calibrated floating-point scene geometry bundle", NeedsInput: true, NeedsMask: false, NeedsReference: false, Flags: []string{"support-roi"}},
	{Verb: "image-to-object", Op: "image_to_object", Desc: "Reconstruct a textured 3D prop mesh from one image and optional foreground mask", NeedsInput: true, NeedsMask: false, NeedsReference: false, Flags: []string{"mask", "mesh-resolution"}},
	{Verb: "normal-map", Op: "normal_map", Desc: "Convert image luminance/depth into a tangent-space normal map", NeedsInput: true, NeedsMask: false, NeedsReference: false, Flags: []string{}},
	{Verb: "caption", Op: "caption", Desc: "Describe an image in natural language", NeedsInput: true, NeedsMask: false, NeedsReference: false, Flags: []string{"prompt"}},
	{Verb: "detect", Op: "object_detection", Desc: "Detect and locate objects with bounding boxes", NeedsInput: true, NeedsMask: false, NeedsReference: false, Flags: []string{}},
	{Verb: "tag", Op: "tagging", Desc: "Predict descriptive tags/labels for an image", NeedsInput: true, NeedsMask: false, NeedsReference: false, Flags: []string{}},
	{Verb: "embed", Op: "embedding", Desc: "Compute a vector embedding of an image", NeedsInput: true, NeedsMask: false, NeedsReference: false, Flags: []string{}},
}
