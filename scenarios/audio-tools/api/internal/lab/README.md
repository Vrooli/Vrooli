# Lab / qualification boundary

Audio Tools ships qualification surfaces in the same API process as the
product: corpus replay, the persisted experiment lifecycle, eval report
mapping, and the leased browser soak. They are real product surfaces (the lab
console and Test Genie call them), but they are **not serving paths**. This
directory owns the written contract and the executable check that keeps that
distinction true.

## Lab packages

| Package | Role |
| --- | --- |
| `internal/eval` | Transport-free replay harness, metrics, safety gates, and report types. |
| `internal/experiment` | Persisted experiment lifecycle: SQLite repository, service, async manager. |
| `internal/experiment/evaldeps` | Composition adapters from corpus/STT/speaker ports into the eval runner. |
| `internal/experiment/recipe` | Recipe materialization: long-form assembly and augmentation. |
| `internal/experiment/report` | Report persistence helpers and qualification evidence shaping. |
| `internal/soak` | Long-form browser soak driver and evidence writer behind the leased isolation boundary. |
| `handlers/experiment` | Connect transport for `ExperimentService`. |
| `handlers/eval` | Protobuf mapping for eval reports. |
| `handlers/soak` | Leased, out-of-band browser soak endpoint. |
| `handlers/validation` | Test Genie validation-provider phase that owns the browser soak. |

Mixed-use packages are deliberately **out of scope** for this boundary:
`internal/corpus` (also backs the product corpus UI), `internal/qualification`
(trust floor shared by diagnostics and soak), and `internal/conformance` /
`internal/acceptance` (evidence document formats). Do not describe them as
lab-only.

## Contract

1. **Serving code must not reach lab code.** The request-serving packages below
   must have no transitive import path to any lab package, including packages
   added later under `internal/experiment/`, `internal/eval/`, or
   `internal/soak/`:
   - `handlers/stt`, `handlers/tts`, `handlers/summarize`, `handlers/audio`
   - their serving dependencies: `internal/stt`, `internal/tts`,
     `internal/summarize`, `internal/audio`
   - the provider chains bootstrap builds: `internal/ai/chains`,
     `internal/ai/sttchain`, `internal/ai/ttschain`,
     `internal/ai/summarizechain`
2. **Composition may cross, but only explicitly.** `internal/bootstrap` (the
   composition root) and `internal/server` (which mounts handler modules) do
   import lab packages today in order to expose the product's qualification
   surfaces. `boundary_test.go` pins those direct imports to an allowlist. A new
   crossing fails the test and must be an explicit edit with a reason. Removing
   an import without removing its allowlist entry also fails, so the list cannot
   become stale fiction.
3. **Lab code may import serving code.** Replay and soak must exercise the real
   pipeline. The boundary is one-directional.

`boundary_test.go` parses the source tree with `go/parser` (production files
only; `_test.go` files may import lab freely). It needs no network, no built
binaries, and no process environment.

## Why the boundary exists

Lab code has a different lifecycle and authority from serving code:

- It starts long-running workers, writes corpus/experiment artifacts, and can
  drive a real browser under a Test Genie lease.
- It persists evidence whose denominators and provenance matter to acceptance
  claims; sloppy coupling to request handling would let serving latency,
  timeouts, or privacy rules contaminate qualification.
- Its dependencies (corpus blobs, eval harness, soak broker) are optional at
  boot. A serving handler must keep working when those are unavailable.

Importing lab code into `handlers/stt` or `internal/tts` would make a user
transcription depend on qualification machinery, and would make the lab safe to
remove only after untangling the request path. Keeping the arrow one-directional
is what makes "this is a product surface, not a serving path" checkable.

## Verification

```bash
cd api && go test ./internal/lab/... -count=1
cd scenarios/audio-tools && make lab-test
```

Coverage floors for the lab packages live in `api/coverage_policy_test.go`
alongside the rest of the measured baseline. `make lab-test` runs the lab
packages' tests; the coverage gate runs as part of `go test ./...`.

## Adding a lab package

1. Add its import prefix to `labImportPrefixes` in `boundary_test.go` and its
   row to the table above.
2. Add a measured coverage floor in `api/coverage_policy_test.go` with at least
   five points of headroom.
3. If a composition package must import it, add it to that package's allowlist
   in `boundary_test.go` with a comment naming the mount point.
