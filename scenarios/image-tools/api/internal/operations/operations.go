// Package operations is the single declarative source of truth for image-tools'
// operation vocabulary: every user-facing image verb, its category, and its I/O
// contract (requires_image / requires_mask / prompt_driven). It is the one place
// an operation is *declared*; the model registry (internal/models), the AI engine
// catalog (internal/ai), the compound-op substrate (internal/looks), and the
// analysis catalog (internal/analysis) all *read* it and never re-declare it.
//
// Before this package the vocabulary lived in three hand-synced copies — the
// seed's operations_vocabulary, the Op table in ai/catalog.go, and per-model
// operations[] — which drifted by hand (W1). Collapsing them to one table makes a
// consumer's membership a category *filter* over data instead of a second list;
// a conformance test fails the build on any unresolved op reference. See
// docs/internal/TECHNIQUE-SUBSTRATE.md (Phase 0) and DECISIONS.md (2026-06-25).
package operations

import "sort"

// Category buckets an operation for discovery and to drive each consumer's
// membership. The AI engine owns generation+enhancement (the ops it builds
// runners for); restoration is forward-declared + provider-served but not yet
// AI-engine-wired; analysis is owned by internal/analysis.
type Category string

const (
	// CategoryGeneration covers text-to-image and image-editing generation.
	CategoryGeneration Category = "generation"
	// CategoryEnhancement covers super-resolution / restoration-style enhancement
	// wired into the AI engine.
	CategoryEnhancement Category = "enhancement"
	// CategoryRestoration covers degradation-repair ops that are forward-declared
	// in the vocabulary and provider-served but NOT yet wired into the AI engine's
	// runner set (kept out of the AI discovery catalog).
	CategoryRestoration Category = "restoration"
	// CategoryAnalysis covers extraction/inference ops owned by internal/analysis.
	CategoryAnalysis Category = "analysis"
)

func (c Category) valid() bool {
	switch c {
	case CategoryGeneration, CategoryEnhancement, CategoryRestoration, CategoryAnalysis:
		return true
	default:
		return false
	}
}

// Operation is one entry in the vocabulary SSOT.
type Operation struct {
	// Name is the canonical op name and the {operation} path segment.
	Name string
	// Category buckets the op (see the Category constants).
	Category Category
	// Summary is a one-line human description (surfaced in discovery).
	Summary string
	// RequiresImage is true when the op edits/consumes an input image (false for
	// the prompt-only text_to_image).
	RequiresImage bool
	// RequiresMask is true when the op needs a mask image (inpaint, outpaint,
	// object_removal, background_replace).
	RequiresMask bool
	// PromptDriven is true when a text prompt is a primary input.
	PromptDriven bool
}

// table is the canonical operation vocabulary in declaration order (the order
// internal/models exposes via Operations()). Adding an op is one row here; every
// consumer picks it up by category filter, and the conformance test enforces that
// no model/backend/Look/analysis reference names an op absent from this table.
var table = []Operation{
	// generation
	{Name: "text_to_image", Category: CategoryGeneration, Summary: "Generate an image from a text prompt", PromptDriven: true},
	{Name: "image_to_image", Category: CategoryGeneration, Summary: "Transform an input image guided by a prompt", RequiresImage: true, PromptDriven: true},
	{Name: "edit_instruct", Category: CategoryGeneration, Summary: "Edit an image from a natural-language instruction (identity-preserving)", RequiresImage: true, PromptDriven: true},
	{Name: "inpaint", Category: CategoryGeneration, Summary: "Regenerate a masked region from a prompt", RequiresImage: true, RequiresMask: true, PromptDriven: true},
	{Name: "outpaint", Category: CategoryGeneration, Summary: "Expand an image beyond its borders, generating the new region from a prompt", RequiresImage: true, RequiresMask: true, PromptDriven: true},
	{Name: "object_removal", Category: CategoryGeneration, Summary: "Remove a masked object and fill the gap", RequiresImage: true, RequiresMask: true},
	{Name: "background_removal", Category: CategoryEnhancement, Summary: "Remove the background to transparency", RequiresImage: true},
	{Name: "background_replace", Category: CategoryGeneration, Summary: "Replace the background behind a masked subject from a prompt", RequiresImage: true, RequiresMask: true, PromptDriven: true},
	// enhancement
	{Name: "upscale", Category: CategoryEnhancement, Summary: "Super-resolve / enlarge an image", RequiresImage: true},
	{Name: "denoise", Category: CategoryEnhancement, Summary: "Reduce noise / deblur an image", RequiresImage: true},
	// restoration (forward-declared; not AI-engine-wired)
	{Name: "deblur", Category: CategoryRestoration, Summary: "Sharpen and remove motion/defocus blur from an image", RequiresImage: true},
	{Name: "naturalize", Category: CategoryEnhancement, Summary: "Reintroduce realistic texture/grain to over-smoothed (restored/upscaled) images", RequiresImage: true},
	{Name: "colorize", Category: CategoryEnhancement, Summary: "Add realistic colour to a grayscale / black-and-white image", RequiresImage: true},
	{Name: "old_photo_restore", Category: CategoryRestoration, Summary: "Repair scratches, fading, and damage on old photographs", RequiresImage: true},
	{Name: "face_restore", Category: CategoryRestoration, Summary: "Restore degraded faces in a photo", RequiresImage: true},
	// analysis
	{Name: "segment", Category: CategoryAnalysis, Summary: "Segment objects/regions in an image", RequiresImage: true},
	{Name: "depth_map", Category: CategoryEnhancement, Summary: "Estimate a per-pixel depth map from a single image", RequiresImage: true},
	{Name: "normal_map", Category: CategoryEnhancement, Summary: "Convert image luminance/depth into a tangent-space normal map", RequiresImage: true},
	{Name: "ocr", Category: CategoryAnalysis, Summary: "Extract text from an image (OCR)", RequiresImage: true},
	{Name: "nsfw_classify", Category: CategoryAnalysis, Summary: "Classify an image for NSFW / unsafe content", RequiresImage: true},
	{Name: "caption", Category: CategoryAnalysis, Summary: "Describe an image in natural language", RequiresImage: true},
	{Name: "object_detection", Category: CategoryAnalysis, Summary: "Detect and locate objects with bounding boxes", RequiresImage: true},
	{Name: "tagging", Category: CategoryAnalysis, Summary: "Predict descriptive tags/labels for an image", RequiresImage: true},
	{Name: "face_detection", Category: CategoryAnalysis, Summary: "Detect face bounding boxes / landmarks", RequiresImage: true},
	{Name: "quality_assessment", Category: CategoryAnalysis, Summary: "Assess no-reference image quality (sharpness, exposure, contrast)", RequiresImage: true},
	{Name: "duplicate_detect", Category: CategoryAnalysis, Summary: "Compute perceptual fingerprints to find near-duplicate images", RequiresImage: true},
	{Name: "embedding", Category: CategoryAnalysis, Summary: "Compute a vector embedding of an image", RequiresImage: true},
	{Name: "qr_barcode_read", Category: CategoryAnalysis, Summary: "Read QR codes / barcodes from an image", RequiresImage: true},
}

// byName indexes the table for O(1) lookup; init validates structural integrity
// (non-empty unique names, valid categories) so a bad edit fails at package init.
var byName = func() map[string]Operation {
	m := make(map[string]Operation, len(table))
	for _, op := range table {
		if op.Name == "" {
			panic("operations: table has an entry with an empty name")
		}
		if !op.Category.valid() {
			panic("operations: op " + op.Name + " has invalid category " + string(op.Category))
		}
		if _, dup := m[op.Name]; dup {
			panic("operations: duplicate op name " + op.Name)
		}
		m[op.Name] = op
	}
	return m
}()

// All returns the full vocabulary in declaration order.
func All() []Operation { return append([]Operation(nil), table...) }

// Names returns every op name in declaration order.
func Names() []string {
	out := make([]string, len(table))
	for i, op := range table {
		out[i] = op.Name
	}
	return out
}

// Get returns the operation and whether it is in the vocabulary.
func Get(name string) (Operation, bool) {
	op, ok := byName[name]
	return op, ok
}

// Has reports whether name is a known operation.
func Has(name string) bool {
	_, ok := byName[name]
	return ok
}

// ByCategory returns every operation in any of the given categories, in
// declaration order. With no categories it returns nothing.
func ByCategory(cats ...Category) []Operation {
	want := make(map[Category]struct{}, len(cats))
	for _, c := range cats {
		want[c] = struct{}{}
	}
	var out []Operation
	for _, op := range table {
		if _, ok := want[op.Category]; ok {
			out = append(out, op)
		}
	}
	return out
}

// NamesByCategory returns the names in any of the given categories, sorted
// (stable for callers that want a deterministic set rather than seed order).
func NamesByCategory(cats ...Category) []string {
	ops := ByCategory(cats...)
	out := make([]string, len(ops))
	for i, op := range ops {
		out[i] = op.Name
	}
	sort.Strings(out)
	return out
}
