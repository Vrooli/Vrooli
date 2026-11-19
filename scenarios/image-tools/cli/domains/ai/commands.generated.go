// AUTO-GENERATED — do not edit by hand.
//
// Source : api/internal/capability/manifests.seed.json
// Codegen: api/cmd/gen-capability (invoked by `make capability-gen`)
// Drift  : api/internal/capability/render_test.go fails the build if this file is stale.
//
// Data only: the submit command builder and flag definitions live in register.go.

package ai

var generatedSubmitSpecs = []submitSpec{
	{Verb: "generate", Op: "text_to_image", Desc: "Generate an image from a text prompt", NeedsInput: false, NeedsMask: false, Flags: []string{"prompt", "negative", "seed", "width", "height", "steps", "cfg-scale", "variations", "auto-scan"}},
	{Verb: "img2img", Op: "image_to_image", Desc: "Transform an input image guided by a prompt", NeedsInput: true, NeedsMask: false, Flags: []string{"prompt", "negative", "seed", "width", "height", "steps", "cfg-scale", "variations", "auto-scan", "strength"}},
	{Verb: "edit", Op: "edit_instruct", Desc: "Edit an image from a natural-language instruction (identity-preserving)", NeedsInput: true, NeedsMask: false, Flags: []string{"prompt", "negative", "seed", "width", "height", "steps", "cfg-scale", "variations", "auto-scan", "strength"}},
	{Verb: "inpaint", Op: "inpaint", Desc: "Regenerate a masked region from a prompt", NeedsInput: true, NeedsMask: true, Flags: []string{"prompt", "negative", "seed", "width", "height", "steps", "cfg-scale", "variations", "auto-scan", "mask", "crop-to-mask", "crop-context-pixels", "crop-max-edge-px", "neutralize-masked-pixels"}},
	{Verb: "outpaint", Op: "outpaint", Desc: "Expand an image beyond its borders, generating the new region from a prompt", NeedsInput: true, NeedsMask: true, Flags: []string{"prompt", "negative", "seed", "width", "height", "steps", "cfg-scale", "variations", "auto-scan", "mask"}},
	{Verb: "object-removal", Op: "object_removal", Desc: "Remove a masked object and fill the gap", NeedsInput: true, NeedsMask: true, Flags: []string{"mask", "mask-expansion", "remove-region"}},
	{Verb: "bg-removal", Op: "background_removal", Desc: "Remove the background to transparency", NeedsInput: true, NeedsMask: false, Flags: []string{}},
	{Verb: "background-replace", Op: "background_replace", Desc: "Replace the background behind a masked subject from a prompt", NeedsInput: true, NeedsMask: true, Flags: []string{"prompt", "negative", "seed", "width", "height", "steps", "cfg-scale", "variations", "auto-scan", "mask"}},
	{Verb: "face-swap", Op: "face_swap", Desc: "Transfer a reference face onto a subject (reference supplied as a conditioning image)", NeedsInput: true, NeedsMask: false, Flags: []string{"prompt", "negative", "seed", "width", "height", "steps", "cfg-scale", "variations", "auto-scan"}},
	{Verb: "garment-replace", Op: "garment_replace", Desc: "Replace or restyle clothing, hair, or makeup on a masked region from a prompt", NeedsInput: true, NeedsMask: true, Flags: []string{"prompt", "negative", "seed", "width", "height", "steps", "cfg-scale", "variations", "auto-scan", "mask"}},
	{Verb: "upscale", Op: "upscale", Desc: "Super-resolve / enlarge an image", NeedsInput: true, NeedsMask: false, Flags: []string{"scale"}},
	{Verb: "denoise", Op: "denoise", Desc: "Reduce noise / deblur an image", NeedsInput: true, NeedsMask: false, Flags: []string{}},
	{Verb: "deblur", Op: "deblur", Desc: "Sharpen and remove motion/defocus blur from an image", NeedsInput: true, NeedsMask: false, Flags: []string{}},
	{Verb: "naturalize", Op: "naturalize", Desc: "Reintroduce realistic texture/grain to over-smoothed (restored/upscaled) images", NeedsInput: true, NeedsMask: false, Flags: []string{"realism", "face-aware"}},
	{Verb: "colorize", Op: "colorize", Desc: "Add realistic colour to a grayscale / black-and-white image", NeedsInput: true, NeedsMask: false, Flags: []string{}},
	{Verb: "old-photo-restore", Op: "old_photo_restore", Desc: "Repair scratches, fading, and damage on old photographs", NeedsInput: true, NeedsMask: false, Flags: []string{}},
	{Verb: "face-restore", Op: "face_restore", Desc: "Restore degraded faces in a photo", NeedsInput: true, NeedsMask: false, Flags: []string{}},
	{Verb: "segment", Op: "segment", Desc: "Segment objects/regions in an image", NeedsInput: true, NeedsMask: false, Flags: []string{"selection"}},
	{Verb: "depth", Op: "depth_map", Desc: "Estimate a per-pixel depth map from a single image", NeedsInput: true, NeedsMask: false, Flags: []string{}},
	{Verb: "geometry", Op: "scene_geometry", Desc: "Estimate a camera-calibrated floating-point scene geometry bundle", NeedsInput: true, NeedsMask: false, Flags: []string{"support-roi"}},
	{Verb: "image-to-object", Op: "image_to_object", Desc: "Reconstruct a textured 3D prop mesh from one image and optional foreground mask", NeedsInput: true, NeedsMask: false, Flags: []string{"mask", "mesh-resolution"}},
	{Verb: "normal-map", Op: "normal_map", Desc: "Convert image luminance/depth into a tangent-space normal map", NeedsInput: true, NeedsMask: false, Flags: []string{}},
	{Verb: "caption", Op: "caption", Desc: "Describe an image in natural language", NeedsInput: true, NeedsMask: false, Flags: []string{"prompt"}},
}
