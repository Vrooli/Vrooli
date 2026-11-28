# Lab / qualification boundary

Audio Tools builds separate product and lab API artifacts. The product build
does not import lab handler packages or lab endpoint metadata. The lab API
component builds with the `audio_lab` Go tag; that build adds corpus replay,
the persisted experiment lifecycle, eval report mapping, and the leased
browser soak. Both profiles remain supervisor-owned.
Bootstrap applies product schemas in the product variant and adds corpus and
experiment schemas in the lab variant. Moving an existing database to product
mode does not drop tables created by an earlier lab run. This narrows runtime
initialization. Product builds exclude the experiment worker, report mappers,
corpus and experiment services, and all lab handlers from their compile-time
dependency graph. The product API retains only neutral measure and evidence
reader contracts with empty product-profile sources. This directory owns the
written contract and executable checks for both the import and build graphs;
server composition tests enforce the route boundary.

## Lab packages

| Package | Role |
| --- | --- |
| `internal/corpus` | Lab-only corpus metadata and SQLite repository; audio payloads stay in the blob store. |
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
`internal/qualification` (trust floor shared by diagnostics and soak),
`internal/conformance` / `internal/acceptance` (evidence document formats), and
`internal/transcriptmetrics` (shared WER normalization/alignment). Do not
describe them as lab-only.

## Contract

1. **Serving code must not reach lab code.** The request-serving packages below
   must have no transitive import path to any lab package, including packages
   added later under `internal/experiment/`, `internal/eval/`, or
   `internal/soak/`:
   - `handlers/stt`, `handlers/tts`, `handlers/summarize`, `handlers/audio`,
     `handlers/diagnostics`, `handlers/models`
   - their serving dependencies: `internal/stt`, `internal/tts`,
     `internal/summarize`, `internal/audio`
   - product diagnostics, model promotion, and measure contracts:
     `internal/diagnostics`, `internal/modelpromotion`, `internal/measures`
   - the provider chains bootstrap builds: `internal/ai/chains`,
     `internal/ai/sttchain`, `internal/ai/ttschain`,
     `internal/ai/summarizechain`
2. **Composition may cross, but only explicitly.** `internal/bootstrap` (the
   composition root), `internal/server` (which mounts handler modules), and
   `internal/modules` (static endpoint/schema metadata used at startup and by
   endpoint generation) contain lab crossings in `audio_lab` constrained
   files. `boundary_test.go` pins those direct imports to an allowlist. A new
   crossing fails the test and must be an explicit edit with a reason.
3. **Lab code may import serving code.** Replay and soak must exercise the real
   pipeline. The boundary is one-directional.

`boundary_test.go` parses every non-lab handler package and the declared serving
roots with `go/parser`, following direct and transitive imports (production
files only). It also pins direct lab imports at each composition package.
`product_dependency_test.go` checks both the compiler-selected product API
dependency graph and package-pattern listing, and verifies that the `audio_lab`
profile restores the lab packages. Lab source files and their tests require
that tag, so `go test ./...` remains the product profile; lab tests and coverage
run through `make lab-test`. The checks need no network or running services.

The bootstrap keeps lightweight metadata repositories available where
product endpoints consume aggregate counts and qualification evidence. It
creates corpus/report blob stores, eval adapters, and the asynchronous
experiment worker only when the supervisor selects `VROOLI_VARIANT=lab`.
Product/live processes retain evidence and measures reads without initializing
the lab artifact stores or polling queued jobs.

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
cd api && go build -o /tmp/audio-tools-product .
cd api && go build -tags audio_lab -o /tmp/audio-tools-lab .
cd scenarios/audio-tools && make lab-test
```

Coverage floors for lab packages live in `api/coverage_policy_lab_test.go`,
separate from product floors in `api/coverage_policy_test.go`.

## Adding a lab package

1. Add its import prefix to `labImportPrefixes` in `boundary_test.go` and its
   row to the table above.
2. Add a measured coverage floor in `api/coverage_policy_lab_test.go` with at least
   five points of headroom.
3. If a composition package must import it, add it to that package's allowlist
   in `boundary_test.go` with a comment naming the mount point.
