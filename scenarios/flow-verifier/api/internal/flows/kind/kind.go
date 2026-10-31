// Package kind defines the chassis every flow kind (temporal, navigation, …)
// plugs into. A Kind owns its schema, its on-disk parse, its verification
// algorithm, its codegen, and the descriptor that Flow Studio uses to render
// it. The flows service dispatches by looking up the registered Kind for a
// contract's `kind` field.
package kind

import "context"

// Kind is the contract every flow kind must satisfy.
//
// Implementations register themselves at package init() via Register. The
// flows service never knows about concrete kinds — it always goes through
// this interface and the registry.
type Kind interface {
	// Name is the wire identifier that appears in a contract's `kind` field
	// and on the gRPC/Connect surface. Must be globally unique.
	Name() string

	// SchemaJSON returns the embedded JSON Schema for this kind.
	SchemaJSON() []byte

	// FilenameGlobs returns the on-disk filename patterns this kind owns.
	// Discovery walks each `flow/` directory and only consults a Kind for
	// files matching its globs (e.g., `*.json`).
	FilenameGlobs() []string

	// Load parses and validates a raw contract. The contractPath is the
	// repo-relative path used in error messages; it is not consulted for
	// IO. Implementations must validate against SchemaJSON before
	// returning a Spec.
	Load(raw []byte, contractPath string) (Spec, error)

	// Verify runs the kind-specific verification algorithm (reachability,
	// model-checking, etc.) and returns a structured result.
	Verify(ctx context.Context, s Spec) (VerifyResult, error)

	// Scaffold writes a fresh contract directory to disk and returns the
	// repo-relative path of the created flow directory.
	Scaffold(opts ScaffoldOptions) (relPath string, err error)

	// Codegen emits any artifacts (TS routes, Go constants, etc.) derived
	// from the spec. Returning empty Artifacts is valid for kinds whose
	// only output is the verification report.
	Codegen(s Spec, target Language) (Artifacts, error)

	// StudioDescriptor returns the data Flow Studio needs to render this
	// kind's graph (nodes, edges, viewport/context toggles).
	StudioDescriptor(s Spec) StudioDescriptor
}

// Spec is the kind-agnostic view of a loaded contract. Concrete kinds
// embed their domain-specific data behind this interface; callers that
// need the rich shape type-assert to the concrete type.
type Spec interface {
	FlowID() string
	Domain() string
	Description() string
	ContractPath() string
	SchemaVersion() int
	Kind() string
}

// Language is the codegen emission target.
type Language string

const (
	LanguageGo         Language = "go"
	LanguageTypeScript Language = "typescript"
)

// ScaffoldOptions configures a Scaffold call. Fields a particular kind
// does not understand are ignored.
type ScaffoldOptions struct {
	Root      string
	ParentDir string
	FlowID    string
	Language  Language
}

// Artifacts is the output of Codegen: a map of repo-relative path to
// file contents.
type Artifacts struct {
	Files map[string][]byte
}

// VerifyResult is the structured result of Verify.
type VerifyResult struct {
	Passed   bool
	Findings []Finding
}

// Finding is a single verification observation — pass or fail.
type Finding struct {
	ID       string
	Passed   bool
	Severity string
	Message  string
	// Trace is an optional counter-example or witness, kind-specific in
	// shape; callers that need it type-assert.
	Trace any
}

// StudioDescriptor is the data Flow Studio uses to render a kind's
// graph. The renderer plugin name selects the React component; nodes
// and edges are arbitrary JSON-marshallable payloads the renderer
// understands.
type StudioDescriptor struct {
	Renderer string
	Nodes    any
	Edges    any
	Toggles  any
}
