# Composition — workflows, ports, and cross-scenario interop

This document defines how operations compose: inside Image Tools, across steps,
and (target) across media scenarios such as audio-tools and Device Studio.

It is the companion to [`CAPABILITY-MODEL.md`](CAPABILITY-MODEL.md) (what a
capability is) and [`../internal/CAPABILITY-MANIFEST.md`](../internal/CAPABILITY-MANIFEST.md)
(how a capability is declared).

## Current state

`internal/looks` is the single compound-workflow substrate. It is good and should
be kept. As of 2026-07-05 it carries the port contract and validates chains:

```proto
message StepInput {
  string name = 1;   // input port from the capability manifest
  string from = 2;   // "source" | "previous" (default)
  string kind = 3;   // expected kind; empty resolves from the manifest
}

message LookStep {
  string operation = 1;
  StepKind kind = 2;   // engine lane: deterministic | AI
  map<string, string> params = 3;
  string scenario = 4; // "" / "image-tools" today
  repeated StepInput inputs = 5;
  string output_kind = 6;
}
```

`internal/looks.Validate` now rejects a chain when a step is unknown, is not
executable as a workflow step, declares a kind that disagrees with its lane,
declares a port that does not exist, or consumes an image the previous step does
not produce. Masks are treated as externally supplied and prompt text as supplied
by the Look. `CompileLook` validates before resolving, so a bad chain fails as
`InvalidArgument` instead of producing a broken run.

A note on `kind`: it is the **engine lane**, not the mechanism. `naturalize` is a
`compute` op that still runs on the model-backed job lane. The lane is now read
from the manifest's `engine` field (`ai` → AI lane, `compute` → deterministic
lane), so a step's lane is declared data rather than inferred from catalog
membership in two packages.

Remaining limitations:

1. **Cross-scenario steps are declared but not executable here.** `scenario` is
   representable; any value other than `image-tools` is rejected with an
   actionable error. The catalog another scenario publishes is readable (C3), but
   executing a peer's step is the workflow surface's job, not this scenario's.
2. **The chain is linear, not a DAG.** A step consumes the previous output or the
   workflow source; there is no fan-out/fan-in yet.
3. **No partial failure model across a chain.** Today a step fails; the Look
   fails. That is fine inside one scenario and insufficient across two.

## Target model

### Typed ports

Every capability manifest declares inputs and an output with **port kinds**:

| Port kind | Carries | Produced by (examples) |
|---|---|---|
| `image` | a raster blob reference | all image ops |
| `mask` | a single-channel blob reference | `segment`, selection |
| `text` | structured text | `ocr`, `caption` |
| `data` | JSON (detections, metrics, geometry) | `object_detection`, `quality_assessment` |
| `audio` | an audio blob reference | audio-tools TTS/synthesis |
| `video` | a video blob reference | Device Studio |
| `mesh` | a mesh bundle | `image_to_object` |

A **step** is `{scenario, operation, inputs, params}`. A **workflow** is a DAG of
steps whose edges connect a producer port to a consumer port of the same kind.
Validation is a type check, and it runs before any work is queued.

```json
{
  "id": "clean-screenshot-then-caption",
  "inputs": [{"name": "source", "kind": "image"}],
  "steps": [
    {"id": "clean", "scenario": "image-tools", "operation": "screenshot_cleanup",
     "inputs": {"image": "source"}},
    {"id": "read", "scenario": "image-tools", "operation": "ocr",
     "inputs": {"image": "clean.image"}, "outputs": {"text": "text"}},
    {"id": "narrate", "scenario": "audio-tools", "operation": "tts",
     "inputs": {"text": "read.text"}, "outputs": {"audio": "voice"}}
  ],
  "outputs": [{"name": "voice", "kind": "audio", "from": "narrate.voice"}]
}
```

### Ownership and execution

The load-bearing rule: **a scenario executes only its own steps.**

- Image Tools accepts image steps; audio-tools accepts audio steps.
- The orchestrator is the caller — a UI workflow builder, a scenario that composes
  others, or (later) a control-plane workflow runner. It holds the DAG, resolves
  each step to its owning scenario, and threads blob references.
- No scenario imports another scenario's Go packages or operation catalog. The
  dependency is a **contract**, not a code dependency. This preserves the existing
  boundary discipline (`ARCHITECTURE.md` "System Boundaries").

Why not let Image Tools orchestrate the whole DAG: it would make Image Tools a
runtime for other people's capabilities, which contradicts its role as one
capability in a fleet and would couple its release cadence to every peer.

### Safety across a workflow

- Each step carries its own `consent_weight`; a workflow's effective weight is the
  **maximum** over its steps (same elevation rule already used for adapters).
- The **executing** scenario applies its own deployment policy. A public audio
  deployment does not inherit a permissive image deployment's controls, and vice
  versa. This is correct: legal exposure is per-deployment, not per-DAG.
- Provenance/attribution, when enabled, is attached per artifact as it is produced,
  so a downstream scenario can preserve the chain rather than re-derive it.

## Cross-scenario catalog

To compose, a builder must be able to *discover* peer operations. Target: a
versioned **media-operation manifest contract** published by each scenario.

### Reuse check (important)

Two platform packages already exist and are **not** this:

| Package | What it actually does | Why it is not the catalog |
|---|---|---|
| `packages/capability-registry-go` | Projects `.vrooli/service.json` dependencies into liveness/readiness states with overlays | Answers "is postgres up?", not "what operations exist?" |
| `packages/capabilityprobe` | Probes host hardware facts (GPU/VRAM/CPU/OS) | Host facts, not capabilities |

Do not stretch either to cover operation discovery; that would conflate dependency
health with capability vocabulary and make both harder to reason about.

Also note: `scenarios/audio-tools` currently has **no operation vocabulary**
(`api/internal/` is organized by domain: `tts`, `audio`, `audioformat`). Cross-media
composition is therefore greenfield, and audio-tools should adopt the contract only
when it has a reason to publish ops — not preemptively.

### Contract location (built)

The contract lives in `packages/proto/schemas/common/v1/media_operations.proto`
(`common.v1.MediaPort`, `MediaRequirement`, `MediaOperation`,
`MediaOperationCatalog`). Image Tools publishes it through
`vrooli.image_tools.v1.catalog.CatalogService/GetOperationCatalog`, which projects
`api/internal/capability` — the same manifest set the CLI/UI generators and the
safety policy read, so published metadata cannot drift from what runs.

The scenario's own RPC exists because a shared *service* would have to be
implemented by every publisher before anyone could use it; the shared *messages*
are the contract, and each scenario exposes them on its own surface. A builder
reads one base URL per scenario. Aggregating those into one query is the control
plane's job.

The shape, as built:

```proto
// common/v1/media_operations.proto
message MediaOperation {
  string scenario = 1;          // owner slug
  string id = 2;                // operation id
  string summary = 3;
  string family = 4;            // generate | edit | enhance | restore | extract | compose
  string mechanism = 5;         // compute | infer | fetch
  repeated MediaPort inputs = 6; // {name, kind, required}
  MediaPort output = 7;
  string params_schema_json = 8; // JSON Schema as a string
  string consent_weight = 9;     // none | low | high
  repeated string backends = 10; // ordered local-first preference
  MediaRequirement requires = 11; // {process, weights, network}
}
message MediaOperationCatalog {
  string contract_version = 1;
  string scenario = 2;
  repeated MediaOperation operations = 3;
}

// image-tools/v1/catalog/catalog.proto
service CatalogService {
  rpc GetOperationCatalog(GetOperationCatalogRequest)
      returns (common.v1.MediaOperationCatalog);
}
```

A filter by `family` is applied server-side; an unknown family is rejected rather
than returning an empty catalog (which a builder would read as "this scenario
offers nothing"). `TestPublishedCatalogMirrorsManifests` proves the projection is
faithful, and `TestPublishedCatalogFamilyFilter` proves every family is
represented.

Image Tools publishes; audio/video adopt opportunistically. Image Tools must not
depend on the catalog being populated.

## Search and manage operations

Today: `ops list`, `models list/install/enable`, the model picker with readiness,
and `--explain` (which resolves operation → model → technique → tier without
running). That covers *within-scenario* management.

Missing, and where it belongs:

| Need | Owner | Why not Image Tools |
|---|---|---|
| Cross-scenario operation search | control plane / catalog browser | otherwise every media scenario grows its own browser |
| Enable/disable policy per op across scenarios | control plane | fleet policy |
| Workflow authoring + validation UI | a workflow surface (control plane or a dedicated scenario) | composition is not an image capability |
| Install orchestration across scenarios | control plane | host remediation is already control-plane-owned |

Image Tools' job is to expose precise, typed manifests and a stable per-step
contract. Anything above that is fleet-level.

## Phases

| Phase | Work | Gate | Status |
|---|---|---|---|
| C0 | `LookStep` gains `scenario` + typed ports; keep `params` as raw JSON | existing Looks still compile and run | **done** |
| C1 | Workflow validator: kind-check every edge before queueing | a deliberately mistyped DAG fails with an actionable error | **done** (linear chain) |
| C2 | First cross-step proof inside Image Tools: screenshot cleanup | headless run | **done** (see below) |
| C3 | Publish `MediaOperation` (shared `common.v1` contract) | contract published + drift-gated | **done** — the *consumer* side (a peer step) remains with the workflow surface |

C2 is the cheap, high-value proof: it needs no cross-scenario contract and it
validates the port model on real work. Do C2 before C3.

### C2 as built

`screenshot_cleanup` is a deterministic `internal/ops` operation that trims the
uniform bands a screenshot carries around its subject: window chrome, the frame a
picture was embedded in, and letterbox bars. It is bounded by design — a
`max_trim_percent` ceiling per side (default 25%) because a photograph whose
subject touches the edge can present a flat row that is content, not chrome — and
it refuses to trim at all when a band reaches that ceiling, since the scan then
stopped on the limit rather than on the content.

The scan is per edge and per corner: chrome is frequently asymmetric (a white
title bar above a dark rail), so a single frame colour sampled from the four
original corners would match neither band. Each edge is matched against its own
corner, read after the earlier edges have moved the frame.

Anything printed **on** the picture — a floating action button, a burned-in
toolbar — is not a band and is not this operation's job. It is a masked edit, so
the seeded `screenshot-cleanup` Look chains the trim into `upscale` and the
documented extension is an `inpaint` step with a supplied mask:

```json
{"id": "screenshot-cleanup",
 "steps": [
   {"operation": "screenshot_cleanup", "params": {"padding_px": "0"}},
   {"operation": "upscale", "params": {"scale": "2"}}]}
```

`TestScreenshotCleanupLookRunsHeadless` renders that chain in-process: the trim
runs on real pixels, the trimmed image crosses the `image` port into the next
step, and only the model-backed step is deferred. No API, model, or network is
involved, which is the gate C2 asked for.

## Relationship to hot-loading

Ports are what make hot-loaded capabilities composable without code. A new
manifest with `io.inputs: [image]` / `output: image` is immediately usable as a
middle step in any existing workflow. Without ports, "hot load an operation" still
requires hand-editing every workflow definition that might use it.

## Cross-references

- [`CAPABILITY-MODEL.md`](CAPABILITY-MODEL.md) — axes and mechanisms
- [`../internal/CAPABILITY-MANIFEST.md`](../internal/CAPABILITY-MANIFEST.md) — manifest fields used by ports
- [`FLOWS.md`](FLOWS.md) — job and Look lifecycle
- [`../internal/MATURATION-PLAN.md`](../internal/MATURATION-PLAN.md) — sequencing with the rest of the work
