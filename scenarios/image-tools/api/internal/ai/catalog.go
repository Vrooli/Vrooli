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

import (
	"sort"

	"image-tools/internal/operations"
)

// Category and Op are views over the operation vocabulary SSOT
// (internal/operations). The AI engine owns exactly the generation + enhancement
// ops — the ones it builds runners for — so its catalog is a category filter over
// the one table rather than a re-declared list (W1 collapse, see
// docs/internal/TECHNIQUE-SUBSTRATE.md).
type Category = operations.Category

const (
	// CategoryGeneration covers text-to-image and image-editing generation.
	CategoryGeneration = operations.CategoryGeneration
	// CategoryEnhancement covers super-resolution / restoration enhancement.
	CategoryEnhancement = operations.CategoryEnhancement
)

// Op is one model-backed AI operation (an alias of the vocabulary entry). Name
// matches the registry operation vocabulary so the selector and provider registry
// agree.
type Op = operations.Operation

// catalog is the AI-op table derived from the vocabulary SSOT: every generation +
// enhancement operation (the ops the runner factory builds runners for). It is
// consumed by the runner factory (execution), the AIService (discovery), and the
// CLI surface. Each op's model is forward-declared in registry.seed.json and the
// op gates honestly (HTTP 409) until its backend program + weights are installed.
var catalog = func() map[string]Op {
	ops := operations.ByCategory(CategoryGeneration, CategoryEnhancement)
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
