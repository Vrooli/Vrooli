package cliapp

import (
	"fmt"
	"sort"
	"strings"
)

// This file is the single source of truth for the CLI command-architecture
// vocabulary. Both the cli-manifest schema/model and the cli-health classifier
// consume this vocabulary so the "known primitive/exception classes" list is
// declared in exactly one place. See
// scenarios/cli-health/docs/reference/cli-architecture-maturity.md for the
// maturity ladder this vocabulary powers.

// PrimitiveClass names a cli-core command primitive that owns the
// parse -> call -> render (or start -> follow -> render) lifecycle, so scenario
// code supplies only request mapping and human-report mapping and never
// branches on the output format. A command that declares a valid PrimitiveClass
// is renderer-separated by construction: --json is an output contract, not an
// operation selector.
type PrimitiveClass string

const (
	// PrimitiveProtoList is a single read RPC rendered via RenderProtoList
	// (Summary -> Results -> Retrieval Hints), proto JSON under --json.
	PrimitiveProtoList PrimitiveClass = "proto_list"
	// PrimitiveProtoMutation is a single write/destructive RPC rendered via
	// RenderProtoMutation (Result -> What Changed -> Next Command).
	PrimitiveProtoMutation PrimitiveClass = "proto_mutation"
	// PrimitiveOperational is a single diagnostic/status RPC rendered via the
	// operational contract (Status -> Triage -> Next Steps).
	PrimitiveOperational PrimitiveClass = "operational"
	// PrimitiveAction is a generic single-call action that does not fit the
	// list/mutation/operational shapes cleanly; cli-core still owns --json.
	PrimitiveAction PrimitiveClass = "action"
	// PrimitiveUpload owns the multipart upload lifecycle (see UploadFile).
	PrimitiveUpload PrimitiveClass = "upload"
	// PrimitivePassthrough forwards argv to a subprocess/external CLI and
	// streams its output; cli-core owns exit-code mapping.
	PrimitivePassthrough PrimitiveClass = "passthrough"
	// PrimitiveStreaming owns a server-stream follow lifecycle.
	PrimitiveStreaming PrimitiveClass = "streaming"
	// PrimitiveDurableRun owns the server-owned durable start -> follow/wait ->
	// reattach lifecycle; the output format is chosen at the end, so human,
	// --json, and --jsonl share one run-ownership path.
	PrimitiveDurableRun PrimitiveClass = "durable_run"
)

// ExceptionClass names a legitimate special-case command shape that cannot be a
// plain proto command. Declaring one (with a reason) is how a special command
// stays at the top maturity rung without pretending to be a normal proto call.
// Each exception class has a matching special-case PrimitiveClass; using the
// primitive satisfies the exception structurally.
type ExceptionClass string

const (
	// ExceptionStreaming: holds a long-lived server stream / event follow.
	ExceptionStreaming ExceptionClass = "streaming"
	// ExceptionUpload: sends multipart/file bodies through the REST upload path.
	ExceptionUpload ExceptionClass = "upload"
	// ExceptionPassthrough: forwards argv to a subprocess/external CLI.
	ExceptionPassthrough ExceptionClass = "passthrough"
	// ExceptionExternalDelegation: orchestrates another scenario/tool as a step
	// rather than this scenario's own proto method.
	ExceptionExternalDelegation ExceptionClass = "external_delegation"
	// ExceptionDurableRun: owns a server-side durable run lifecycle.
	ExceptionDurableRun ExceptionClass = "durable_run"
)

var primitiveClasses = []PrimitiveClass{
	PrimitiveProtoList,
	PrimitiveProtoMutation,
	PrimitiveOperational,
	PrimitiveAction,
	PrimitiveUpload,
	PrimitivePassthrough,
	PrimitiveStreaming,
	PrimitiveDurableRun,
}

var exceptionClasses = []ExceptionClass{
	ExceptionStreaming,
	ExceptionUpload,
	ExceptionPassthrough,
	ExceptionExternalDelegation,
	ExceptionDurableRun,
}

// specialCasePrimitives maps each special-case primitive to the exception class
// its use satisfies. Normal primitives are absent from this map.
var specialCasePrimitives = map[PrimitiveClass]ExceptionClass{
	PrimitiveUpload:      ExceptionUpload,
	PrimitivePassthrough: ExceptionPassthrough,
	PrimitiveStreaming:   ExceptionStreaming,
	PrimitiveDurableRun:  ExceptionDurableRun,
}

// declarablePrimitiveClasses are the NORMAL command primitive classes a manifest
// may declare in architecture.primitive. Special-case classes are declared as
// exceptions instead (plan decision D4), so they are excluded here even though
// they remain valid OBSERVED evidence values. This is the single source of truth
// the cli-manifest schema's architecture.primitive enum mirrors.
var declarablePrimitiveClasses = []PrimitiveClass{
	PrimitiveProtoList,
	PrimitiveProtoMutation,
	PrimitiveOperational,
	PrimitiveAction,
}

// ValidPrimitiveClasses returns every known primitive class, sorted, for
// schema/classifier vocabulary checks and error messages.
func ValidPrimitiveClasses() []PrimitiveClass {
	out := append([]PrimitiveClass(nil), primitiveClasses...)
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// DeclarablePrimitiveClasses returns every NORMAL primitive class a manifest may
// declare in architecture.primitive (special-case classes are declared as
// exceptions), sorted. The cli-manifest schema's architecture.primitive enum
// mirrors this list.
func DeclarablePrimitiveClasses() []PrimitiveClass {
	out := append([]PrimitiveClass(nil), declarablePrimitiveClasses...)
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// ValidExceptionClasses returns every known exception class, sorted.
func ValidExceptionClasses() []ExceptionClass {
	out := append([]ExceptionClass(nil), exceptionClasses...)
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// Valid reports whether p is a known primitive class.
func (p PrimitiveClass) Valid() bool {
	_, ok := indexOfPrimitive(p)
	return ok
}

// IsSpecialCase reports whether p is one of the special-case primitives
// (upload/passthrough/streaming/durable_run) rather than a normal proto/action
// command class.
func (p PrimitiveClass) IsSpecialCase() bool {
	_, ok := specialCasePrimitives[p]
	return ok
}

// RequiresProtoBinding reports whether a command declaring p must also declare a
// connect-rpc binding (the proto command classes). Special-case primitives and
// action do not require a proto binding.
func (p PrimitiveClass) RequiresProtoBinding() bool {
	switch p {
	case PrimitiveProtoList, PrimitiveProtoMutation, PrimitiveOperational:
		return true
	default:
		return false
	}
}

// SatisfiesException returns the exception class a use of p satisfies, or the
// empty string for normal command classes.
func (p PrimitiveClass) SatisfiesException() ExceptionClass {
	return specialCasePrimitives[p]
}

// Valid reports whether e is a known exception class.
func (e ExceptionClass) Valid() bool {
	for _, c := range exceptionClasses {
		if c == e {
			return true
		}
	}
	return false
}

func indexOfPrimitive(p PrimitiveClass) (int, bool) {
	for i, c := range primitiveClasses {
		if c == p {
			return i, true
		}
	}
	return 0, false
}

// CommandArchitecture is a command's declared renderer-separated architecture.
// A zero value means "unclassified/legacy" and is always valid — the metadata
// is opt-in. A command declares either a normal primitive class, or a
// special-case primitive / exception class (with a reason), never a normal
// primitive alongside an exception.
type CommandArchitecture struct {
	// Primitive is the cli-core primitive class the command is built on.
	Primitive PrimitiveClass
	// Exception, when set, declares a legitimate special-case shape for a
	// command that is not built on a normal proto primitive. ExceptionReason
	// is required whenever Exception is set.
	Exception       ExceptionClass
	ExceptionReason string
}

// IsZero reports whether no architecture metadata is declared.
func (a CommandArchitecture) IsZero() bool {
	return a.Primitive == "" && a.Exception == "" && strings.TrimSpace(a.ExceptionReason) == ""
}

// Validate enforces the architecture-metadata vocabulary and the
// primitive-versus-exception intent (plan decision D4):
//   - a declared Primitive must be a NORMAL command class
//     (proto_list/proto_mutation/operational/action); a special-case class
//     (upload/passthrough/streaming/durable_run) is declared as an exception,
//     never as architecture.primitive;
//   - a declared Exception must be in the known exception vocabulary and carry a
//     non-empty reason;
//   - a reason without an exception class is invalid;
//   - a normal primitive and an exception are mutually exclusive — a command is
//     either a normal renderer-separated primitive OR a declared special case.
//
// A zero value is valid (metadata is opt-in). Callers that opt in and malform
// the declaration fail fast here, which is what cli-health reports as
// arch.metadata_invalid.
func (a CommandArchitecture) Validate() error {
	if a.IsZero() {
		return nil
	}
	if a.Primitive != "" && !a.Primitive.Valid() {
		return fmt.Errorf("unknown primitive class %q (known: %s)", a.Primitive, joinDeclarablePrimitiveNames())
	}
	if a.Primitive != "" && a.Primitive.IsSpecialCase() {
		return fmt.Errorf("primitive %q is a special-case class and must be declared as an exception (class %q with a reason), not as architecture.primitive", a.Primitive, a.Primitive.SatisfiesException())
	}
	if a.Exception != "" && !a.Exception.Valid() {
		return fmt.Errorf("unknown exception class %q (known: %s)", a.Exception, joinExceptionNames())
	}
	if a.Exception != "" && strings.TrimSpace(a.ExceptionReason) == "" {
		return fmt.Errorf("exception class %q requires a reason", a.Exception)
	}
	if a.Exception == "" && strings.TrimSpace(a.ExceptionReason) != "" {
		return fmt.Errorf("exception reason set without an exception class")
	}
	if a.Primitive != "" && a.Exception != "" {
		return fmt.Errorf("primitive %q is a normal command class and cannot also declare exception %q — declare either a normal primitive or a special-case exception, not both", a.Primitive, a.Exception)
	}
	return nil
}

// EvidenceStatus classifies a command's architecture by comparing what a
// manifest declares (declared PrimitiveClass) against what cli-core actually
// observed when the handler was built (observed PrimitiveClass). It is the
// single vocabulary CLI Health uses to separate verified primitive adoption
// from declaration-only debt and from outright contradiction — see
// ClassifyPrimitiveEvidence and D1/D2 of the plan.
type EvidenceStatus string

const (
	// EvidenceNone means neither a declaration nor implementation evidence is
	// present: legacy/unclassified, always valid, capped below verified maturity.
	EvidenceNone EvidenceStatus = "none"
	// EvidenceObservedOnly means the handler was built with a cli-core primitive
	// but the manifest declared no primitive. The implementation is renderer-
	// separated by construction even though the declaration is missing.
	EvidenceObservedOnly EvidenceStatus = "observed_only"
	// EvidenceDeclaredOnly means the manifest declares a primitive but no matching
	// implementation evidence is available. This is advisory maturity debt during
	// rollout, never an error — a declaration cannot reach verified maturity on
	// manifest text alone.
	EvidenceDeclaredOnly EvidenceStatus = "declared_only"
	// EvidenceVerified means the declared primitive matches the primitive the
	// handler was actually built from: verified L4 primitive adoption.
	EvidenceVerified EvidenceStatus = "verified"
	// EvidenceContradiction means the declared primitive and the observed
	// implementation primitive disagree. This is always an error.
	EvidenceContradiction EvidenceStatus = "contradiction"
)

// ClassifyPrimitiveEvidence compares a manifest-declared primitive class against
// the primitive class cli-core observed when building the handler. It is the
// drift detector behind verified primitive maturity: a scenario reaches verified
// status only when declaration and implementation evidence agree, and a
// disagreement is surfaced as a contradiction rather than silently trusting the
// declaration.
func ClassifyPrimitiveEvidence(declared, observed PrimitiveClass) EvidenceStatus {
	switch {
	case declared == "" && observed == "":
		return EvidenceNone
	case declared == "" && observed != "":
		return EvidenceObservedOnly
	case declared != "" && observed == "":
		return EvidenceDeclaredOnly
	case declared == observed:
		return EvidenceVerified
	default:
		return EvidenceContradiction
	}
}

func joinDeclarablePrimitiveNames() string {
	names := make([]string, 0, len(declarablePrimitiveClasses))
	for _, p := range DeclarablePrimitiveClasses() {
		names = append(names, string(p))
	}
	return strings.Join(names, ", ")
}

func joinExceptionNames() string {
	names := make([]string, 0, len(exceptionClasses))
	for _, e := range ValidExceptionClasses() {
		names = append(names, string(e))
	}
	return strings.Join(names, ", ")
}
