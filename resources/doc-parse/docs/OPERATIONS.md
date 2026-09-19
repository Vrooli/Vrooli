# Operations

`doc-parse` is a `native-cli` resource whose stable operator surface is a Go
command backed by a repo-built Rust parser shim. The selected Phase 1 runtime
is one WASI module executed in-process by Wazero; native Rust artifacts are the
declared fallback.

## Architecture Boundary

Keep responsibilities split cleanly:

- `resource.json` owns declarative install, invoke, freshness, portability, and exported environment metadata.
- `cli/` owns the binary entrypoint, artifact resolution, and install/build
  surface.
- `cli/internal/app` owns operator-facing command registration and CLI wiring.
- `cli/internal/domain` owns parser invocation, capability checks, and terminal
  state normalization.
- `src/main.rs` is the Rust parser shim. It communicates over JSONL so the
  WASI and native execution paths have the same contract.

Do not turn `cli/main.go` into the primary implementation surface. If the resource grows richer commands, wire them in `cli/internal/app` and implement the behavior in `cli/internal/domain` or sibling packages under `cli/internal/...`.

## Operator Checklist

- Keep mutable state outside the repo and resolve it through canonical resource storage paths.
- Keep manifest loading and source-root resolution in shared helpers under `cli/internal/discovery` and `cli/internal/version`.
- Route build/install behavior through `cli/internal/install`.
- Keep resource-specific business logic under `cli/internal/domain` instead of reintroducing shell glue.
- Keep `resource.json` as the declarative contract for command/install/invoke/freshness behavior.

## Build and packaging boundary

Rust is built in the containerized build lane with the exact versions in
`Cargo.lock`; no scenario or resource-local installer may invoke a raw package
manager. Run `go run ./build` from `resources/doc-parse/build` in the build
environment. The build produces the selected WASI parser artifact and a
checksum; the native parser remains available as a fallback build output. The
Go resource command must resolve an artifact from its manifest, verify its
checksum, reject unsupported OS/architecture pairs, and report the artifact
identity in `health` and `version` output.

The desktop deployment profiles are conditional: the manifest and checksum
contract are verified, while native target-host execution remains an explicit
follow-up because this phase exercises Linux amd64 only.

## Evidence

Run the portable comparison harness against the corpus under
`testdata/corpus`. A successful run requires byte-for-byte equality for every
fixture, including malformed and password-protected terminal states. See
[`measurements.md`](measurements.md) for the recorded 42-fixture result.
