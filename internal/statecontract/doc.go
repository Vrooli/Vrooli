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
// only through declared constants, stored vocabulary written, compared and
// switched on only through named constants, no cross-module codec calls); a
// failed check makes the executable unqualified, with the reason in its
// refusal. Stored-vocabulary candidates are kept only when their field is a
// string field of a declared envelope (LiteralViolations).
//
// Function bodies, comments, tests and classified constants of precise owners
// do not participate, so a behaviour-only change keeps the contract. A change
// that alters what the prior release reads changes the owner's digest, and
// Diff names the owner and the stored type, field, constant or fact.
//
// The contract cannot see a function body that starts writing a new value
// into an unchanged field through a variable or a function result, or a
// literal on a field whose readers it cannot resolve. Such a change must bump
// the owner's declared format version.
package statecontract
