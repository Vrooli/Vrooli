// Package ai is image-tools' core AI generation & enhancement engine
// (IMG-P0-002 / IMG-P0-003). It owns the model-backed operation catalog and the
// per-operation runners that the durable job Manager executes asynchronously on
// its GPU-serialized lane.
//
// An AI op runs through the seams the spine established in Phase 1: the
// hardware-fit model selector (internal/models), the provider abstraction with
// its Local-GPU → Local-CPU → BYOK ladder (internal/backends), the host probe
// (internal/capabilities), and the blob store (internal/storage). This package
// adds the concrete standalone providers (exec wrappers over sd / iopaint /
// realesrgan / rembg / onnxruntime, gated on binary+model availability) and the
// runner factory that wires probe → select model → select provider → execute →
// persist, plus an optional NSFW auto-scan hook on generated output.
//
// Heavy model work cannot run in CI (the backend binaries/models are absent), so
// the providers refuse cleanly via Available() and the tests drive the full
// vertical with fake providers. The headless-completeness acceptance gate is an
// attended run on a host with the CPU default models installed.
package ai

import "sort"

// Category groups AI operations for discovery/UI.
type Category string

const (
	// CategoryGeneration covers text-to-image and image-editing generation.
	CategoryGeneration Category = "generation"
	// CategoryEnhancement covers super-resolution / restoration enhancement.
	CategoryEnhancement Category = "enhancement"
)

// Op is one model-backed AI operation. Name matches the registry operation
// vocabulary (internal/models) so the selector and provider registry agree.
type Op struct {
	// Name is the canonical op name and the {operation} path segment.
	Name string
	// Category is generation or enhancement.
	Category Category
	// Summary is a one-line human description.
	Summary string
	// RequiresImage is true when the op edits an input image (false for the
	// prompt-only text_to_image).
	RequiresImage bool
	// RequiresMask is true when the op needs a mask image (inpaint, object_removal).
	RequiresMask bool
	// PromptDriven is true when a text prompt is a primary input.
	PromptDriven bool
}

// catalog is the canonical AI-op table: the P0 generation + enhancement subset
// (the breadth ops in IMG-P1-014/015 land in Phase 5). It is the single source
// of truth consumed by the runner factory (execution), the AIService (discovery),
// and the CLI surface.
var catalog = func() map[string]Op {
	ops := []Op{
		{Name: "text_to_image", Category: CategoryGeneration, Summary: "Generate an image from a text prompt", PromptDriven: true},
		{Name: "image_to_image", Category: CategoryGeneration, Summary: "Transform an input image guided by a prompt", RequiresImage: true, PromptDriven: true},
		{Name: "inpaint", Category: CategoryGeneration, Summary: "Regenerate a masked region from a prompt", RequiresImage: true, RequiresMask: true, PromptDriven: true},
		{Name: "object_removal", Category: CategoryGeneration, Summary: "Remove a masked object and fill the gap", RequiresImage: true, RequiresMask: true},
		{Name: "upscale", Category: CategoryEnhancement, Summary: "Super-resolve / enlarge an image", RequiresImage: true},
		{Name: "background_removal", Category: CategoryEnhancement, Summary: "Remove the background to transparency", RequiresImage: true},
		{Name: "denoise", Category: CategoryEnhancement, Summary: "Reduce noise / deblur an image", RequiresImage: true},
	}
	m := make(map[string]Op, len(ops))
	for _, o := range ops {
		m[o.Name] = o
	}
	return m
}()

// Names returns the registered AI-op names in stable (sorted) order.
func Names() []string {
	names := make([]string, 0, len(catalog))
	for n := range catalog {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// List returns the AI-op catalog in stable order (for discovery).
func List() []Op {
	ops := make([]Op, 0, len(catalog))
	for _, n := range Names() {
		ops = append(ops, catalog[n])
	}
	return ops
}

// Get returns the op and whether it is a registered AI operation.
func Get(name string) (Op, bool) {
	o, ok := catalog[name]
	return o, ok
}

// Has reports whether name is a registered AI operation.
func Has(name string) bool {
	_, ok := catalog[name]
	return ok
}
