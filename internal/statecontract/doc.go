// Package statecontract derives the persistent-state compatibility contract of
// a root control-plane executable.
//
// Automatic root activation keeps the prior executable as its rollback image,
// so the prior must be able to read anything the candidate writes before the
// activation verifies. The contract therefore describes what is actually
// stored, not the code that behaves around it:
//
//   - each persistence owner declares its stored envelopes (compiled Go types
//     written by the JSON codec or mirrored by SQL rows) and its format facts
//     (named version constants, SQL DDL, witness bytes, samples of derived
//     stored identities);
//   - the recursive compiled shape of every envelope (each exported field's
//     stored JSON key and options, its type, promotion and field order) is
//     read from the executable itself;
//   - the build stamps, per stored named type, its enum constants and, for a
//     custom JSON/text codec, its declaration, encoder source and the
//     same-package closure its codec methods reach (see
//     buildinfo.PersistentStateBuildFlag);
//   - whole-source owners (the activation machinery and native identities,
//     see WholeSources) contribute a digest of their complete source, and the
//     SQLite owners every package-level constant (ConstantSources);
//   - every other owner's untyped string constants participate unless
//     NotStoredConstants classifies them, and the time layouts it formats and
//     parses participate as a set (PreciseSources);
//   - the Go toolchain release (major.minor) binds encoding/json behaviour.
//
// The builder checks those sources when it stamps (format versions written
// only through declared constants, recognizable stored vocabulary uses, and
// codec dependency coverage); a
// failed check makes the executable unqualified, with the reason in its
// refusal. Stored-vocabulary candidates are kept only when their field is a
// string field of a declared envelope (LiteralViolations).
//
// Function bodies, comments, tests and classified constants of precise owners
// do not participate, so a behaviour-only change keeps the contract. A change
// that alters what the prior release reads changes the owner's digest, and
// Diff names the owner and the stored type, field, constant or fact.
//
// Runtime evolution tests inspect the root CLI and watchdog dependency
// closures for versioned envelopes and recognizable durable-write calls. It
// unions the host closure with Linux, macOS and Windows amd64/arm64 closures
// with cgo disabled; a cgo-only dependency on another platform is not covered.
// Calls in precise and constants-stamped packages have a per-call reviewed
// inventory; a new site fails until classified against an owner or a concrete
// non-state reason. Whole-source owners conservatively change their contract
// for any selected source change, including behavior unrelated to stored data.
// The audit is a development guard, not a proof: it cannot discover arbitrary
// write helpers, external writers, dynamically selected destinations, or a
// changed value passed through an already-classified call. A renamed or
// reordered call may require review even when its behavior is unchanged.
// Site identifiers include receiver type, function, callee and ordinal;
// duplicate identifiers fail the audit.
//
// Vocabulary qualification uses bounded source inference, not complete Go
// type or data-flow analysis. Some variable, function-result, generic,
// interface, dynamic or unresolved field uses can escape it. Changes that
// alter stored values beyond its coverage must bump the owner's declared
// format version.
package statecontract
