// Package ai is image-tools' core AI generation & enhancement engine
// (IMG-P0-002 / IMG-P0-003). It owns the model-backed operation catalog and the
// per-operation runners that the durable job Manager executes asynchronously on
// its serialized local queue.
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

	"image-tools/internal/capability"
)

// Op is one AI operation, projected from its capability manifest. The manifest is
// the declaration of record; this view adds the execution-shaped accessors the
// engine and discovery need, derived from the declared ports so a port change
// cannot leave a stale flag behind.
type Op struct {
	Name          string
	Family        string
	Summary       string
	RequiresImage bool
	RequiresMask  bool
	PromptDriven  bool
}

// catalog is the AI engine's membership: every manifest whose declared engine is
// the AI job lane. There is no second list — adding an operation to this engine
// is setting engine=ai on its manifest, and the conformance test proves the
// runner set agrees with it.
var catalog = func() map[string]Op {
	ops := capability.ByEngine(capability.EngineAI)
	m := make(map[string]Op, len(ops))
	for _, manifest := range ops {
		m[manifest.ID] = project(manifest)
	}
	return m
}()

// project derives the flags the engine reads from the operation's declared ports.
func project(m capability.Manifest) Op {
	op := Op{Name: m.ID, Family: string(m.Family), Summary: m.Summary}
	for _, in := range m.IO.Inputs {
		switch in.Kind {
		case "image":
			op.RequiresImage = true
		case "mask":
			op.RequiresMask = true
		case "text":
			op.PromptDriven = true
		}
	}
	return op
}

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
