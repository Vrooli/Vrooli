# Architecture — Image Tools

This document is the scenario's system map. It explains the invariant
shape inherited from the `react-vite` template, then points to the
specialized documents that own product domains, workflows, data,
integrations, deployment, operations, and business strategy.

Keep this file high-signal. Do not turn it into a warehouse for every
domain, endpoint, workflow, or decision. If a concern has a dedicated
document below, update that document and link it here.

## Purpose Of This Document

This document owns:

- the scenario's system shape,
- the role of each surface,
- how contracts and data flow between surfaces,
- the shared infrastructure boundary,
- extension rules for future code,
- architecture maturity and intentional deviations.

This document does not own:

- product capability inventory: [`DOMAINS.md`](DOMAINS.md),
- temporal and user/system workflows: [`FLOWS.md`](FLOWS.md),
- storage details and retention: [`DATA.md`](DATA.md),
- resource and scenario dependencies: [`INTEGRATIONS.md`](INTEGRATIONS.md),
- test seams and fakes: [`../internal/SEAMS.md`](../internal/SEAMS.md),
- test strategy: [`../internal/TESTING.md`](../internal/TESTING.md),
- deployment and operations: [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md),
- commercial strategy: [`../business/MONETIZATION.md`](../business/MONETIZATION.md).

## Scenario Shape

image-tools is one product — a permanent, reusable image capability —
expressed through three coordinated surfaces over one canonical contract
layer: a Go API server (Connect-RPC, proto-owned contracts, plus a REST
multipart edge for opaque binary uploads), a React+Vite+TypeScript+
Tailwind UI, and a Go CLI. The load-bearing tenet is
**headless-completeness**: every operation — deterministic edit, AI
generation, AI enhancement, image analysis — runs fully from the CLI with
no UI and no ComfyUI dependency. The UI is an *enhancer* (mobile capture,
drag-and-drop, clipboard paste, before/after slider, mask-painting,
progress-with-cancel), never a gate. Three internal structures sit behind
the operation domains and give the scenario its character:

- a **backend-provider abstraction** (`backends`) — every op has ≥1
  standalone non-ComfyUI provider, with a Local-GPU → Local-CPU →
  BYOK-cloud fallback ladder;
- a **declarative model registry + hardware-aware selector** (`models`)
  that picks the best-fit enabled model for the probed host;
- a **durable, server-owned async job queue** (`jobs`) that serializes
  heavy GPU work and survives client disconnect.

The same three-surface, one-contract diagram below holds; image bytes flow
across the REST multipart edge while all metadata stays proto-typed.

```
                       ┌─────────────────────────────┐
                       │  Generated proto types      │
                       │  packages/proto/schemas/    │
                       │   image-tools/v1/...    │
                       └──────────────┬──────────────┘
                                      │ canonical wire shape
              ┌───────────────────────┼───────────────────────┐
              │                       │                       │
              ▼                       ▼                       ▼
        ┌──────────┐            ┌──────────┐            ┌──────────┐
        │   ui/    │ Connect-JSON│  api/   │ Connect-JSON│  cli/   │
        │ React    │ ◀────────▶ │   Go     │ ◀────────▶ │   Go     │
        │ + Vite   │            │ HTTP     │            │ cli-core │
        └──────────┘            └────┬─────┘            └──────────┘
                                     │
                                     ▼
                                ┌─────────┐
                                │ SQLite  │
                                │ (local) │
                                └─────────┘
```

| Surface | Role | Owns | Does Not Own |
|---|---|---|---|
| API (`api/`) | Scenario core | Business rules, persistence, integrations, transport edge | Browser state, CLI formatting |
| UI (`ui/`) | Browser presentation | Components, i18n, accessibility, browser interaction | Business rules, persistence policy |
| CLI (`cli/`) | Operator/agent wrapper | Argument parsing, output formatting, API invocation | Business rules, duplicated validation |
| Contracts (`packages/proto/schemas/image-tools/`) | Wire shape | Proto messages/services and generated clients | Hand-written route/type mirrors |

The load-bearing principle: the API is the only surface that contains
business logic. UI and CLI translate user/operator intent into API
calls. Proto types flow from one source of truth so wire-shape drift
between surfaces is impossible.

Internally the API is layered: operation domains (`ops`, `generation`,
`enhancement`, `analysis`) express *what* an operation does; the
`backends` abstraction resolves *which provider* runs it (subject to the
fallback ladder); `models` answers *which weights* fit the probed host;
`jobs` owns *when and in what order* heavy work executes; and `storage`
owns *where* inputs and outputs live. Operation domains never talk to
providers, model weights, or the queue directly — they go through these
seams, which is what keeps the headless-completeness tenet enforceable and
each domain easy to test and delete.

## System Boundaries

The scenario owns:

- source code under `api/`, `ui/`, and `cli/`,
- generated-scenario docs under `docs/`,
- scenario lifecycle metadata under `.vrooli/`,
- scenario-specific requirements under `requirements/`,
- scenario proto schemas relocated to
  `packages/proto/schemas/image-tools/`.

The scenario does not own:

- shared package implementation under `packages/`,
- Vrooli resource implementation (ComfyUI, per-op model resources),
- scenario dependencies it calls,
- generated proto outputs under `packages/proto/gen/`,
- **host capability/capacity probing** — owned by the platform
  `internal/hostinventory` collector and surfaced by the root `vrooli` CLI
  host-inventory contract; image-tools consumes `vrooli host inventory --json`
  via `packages/vrooli-cli-go` behind the `capabilities` seam and never
  reimplements OS/GPU detection,
- **persistent blob storage implementation** — owned by api-core
  storage/blobstore; image-tools owns only the consuming `storage` seam,
- **palette extraction** — delegated to the palette-gen scenario,
- **document→text** — owned by text-tools; image-tools owns image→text/OCR
  (the boundary is clear and non-overlapping).

Two further hard boundaries follow from PRD non-goals: no face
recognition / identity matching, no face-swap / deepfake / virtual
try-on, and no video or motion content (that belongs to
rich-media-studio). SVG is an import/export raster format only — no vector
editing.

Document dependency and resource decisions in
[`INTEGRATIONS.md`](INTEGRATIONS.md), not here.

## Contracts And Data Flow

Wire shapes do not live in TypeScript interfaces, Go structs, or
hand-written JSON schemas. They live in `.proto` files. For
proto-typed API calls, the `.proto` file also declares the service
block that generates Connect handlers and clients.

```
packages/proto/schemas/image-tools/v1/<domain>/<file>.proto
       │
       ▼
       make generate
       │
       ├──▶ packages/proto/gen/go/image-tools/v1/...              (api, cli)
       ├──▶ packages/proto/gen/go/image-tools/v1/...connect       (Connect-Go)
       ├──▶ packages/proto/gen/typescript/js/image-tools/v1/...   (ui)
       └──▶ packages/proto/gen/python/image_tools/v1/...    (future tools)
```

Use Connect-RPC by default:

- UI to API for proto-typed payloads,
- CLI to API for proto-typed payloads,
- API to API / inter-scenario calls with Vrooli-owned protos.

REST is allowed only for four enumerated reasons, defined as
`RESTReason` constants in `api/internal/module/module.go`:

| Reason | When it applies |
|---|---|
| `RESTReasonMultipartUpload` | Opaque file bytes via `multipart/form-data`. The notes attachments endpoint is the worked example. |
| `RESTReasonWebhookReceiver` | Endpoint shape is dictated by a third-party system (Stripe, GitHub, etc.) we do not own. |
| `RESTReasonThirdPartyShape` | Request or response is an externally-defined contract (OAuth callbacks, OpenAPI passthrough). |
| `RESTReasonOpsProbe` | Lifecycle systems, load balancers, and `curl` must reach the endpoint without a generated client (plain `GET /health`, static iframe-facing HTML wrappers). |

Mechanical enforcement: `cmd/gen-endpoints` rejects any
`EndpointDescriptor.Path` that is not a generated Connect procedure
constant (i.e. does not start with `/vrooli.`) unless the descriptor
carries a `RESTException` with one of the four reasons. A REST
endpoint without that tag fails `make endpoints`, which fails
`make test`, which fails CI. The fix is either to author a proto
service method (the preferred path) or to tag the exception
explicitly. There is no "internal endpoint, REST is fine" path —
that rationalization is exactly what the validation pass prevents.

Note: even for REST exceptions, the **payload shape** stays
proto-typed wherever possible. The notes attachments handler returns
the proto `UploadAttachmentResponse` message; only the request
transport is multipart. Drift between API/UI/CLI is eliminated as
long as the wire payload type is shared.

### Image bytes on the REST multipart edge

image-tools is binary-heavy, so every operation that accepts a source
image, mask, or overlay accepts those bytes over a
`RESTReasonMultipartUpload` edge while its parameters and its result
metadata stay proto-typed. Opaque image bytes are intentionally *not*
carried inside proto payloads; the proto messages reference stored blobs
by id/handle. This is the scenario's one durable transport deviation
(recorded in *Intentional Deviations*) and applies uniformly across
`ops`, `generation`, `enhancement`, and `analysis`.

### The three internal seams

Three contracts flow *inside* the API rather than over the wire, and each
is wired once in production and substitutable in tests
(see [`../internal/SEAMS.md`](../internal/SEAMS.md)):

- **Backend provider interface** (`backends`): operation domains submit an
  op request; the provider registry resolves a concrete provider and
  applies the Local-GPU → Local-CPU → BYOK-cloud fallback ladder, emitting
  user-visible messaging at each tier transition. Registration enforces
  ≥1 standalone (non-ComfyUI) provider per operation.
- **Durable job queue** (`jobs`): heavy/AI operations are submitted as
  server-owned jobs returning a job-id + ETA immediately; the queue
  serializes GPU work and streams progress over SSE. Clients never poll;
  the CLI uses a block-once wait verb (mirroring test-genie's run
  lifecycle).
- **Storage seam** (`storage`): inputs and outputs cross an api-core
  blobstore boundary outside the repo, with an overridable per-request
  save location and a decompression-bomb / large-image guard at ingestion.

## Shared Infrastructure

Shared infrastructure is allowed only when the code is
business-vocabulary-free and used by unrelated domains or surfaces.

| Package/Folder | Purpose | Why Not Domain-Owned | Consumers |
|---|---|---|---|
| `api/internal/server/` | Compose modules and middleware into one HTTP server. | Server lifecycle is not a product capability. | API entrypoint and handler modules. |
| `api/internal/module/` | Shared module and endpoint descriptor types. | Domain modules return this common shape. | Handler packages, server, endpoint codegen. |
| `api/internal/modules/` | Thin registry for schemas and endpoints. | Boot/codegen need central lists; logic stays domain-owned. | `main.go`, `gen-endpoints`. |
| `api/internal/database/` | System schema and DB reachability seam. | Cross-cutting DB infrastructure, not one domain's data. | API boot, health. |
| `api/internal/clock/` | Deterministic time seam. | Time is cross-cutting and test-substitutable. | Middleware, repositories. |
| `api/internal/testutil/` | Cross-domain test harnesses and fakes. | Used by unrelated domains; domain fakes stay domain-local. | API tests. |
| `ui/src/test-utils/` | Cross-feature render helpers, a11y helpers, and model tests. | Used by unrelated UI features. | UI tests. |
| `internal/hostinventory` (platform) | Source of host capability/capacity facts (GPU, VRAM, RAM, OS/arch). | OS/hardware detection is platform-owned, not a product capability. | Surfaced by the root `vrooli` CLI host-inventory contract; consumed by `models` selection and `backends` fallback. |
| api-core storage/blobstore (platform) | Persist opaque image bytes outside the repo with an overridable save location. | Blob persistence is cross-cutting platform infrastructure. | `storage` seam, every operation domain. |
| root `vrooli` CLI host inventory (`cli/v1`) | Typed host-facts contract (`vrooli host inventory --json`) over the shared `internal/hostinventory` collector. | Detection is platform-owned; image-tools is a read-only consumer, not a peer-scenario caller. | image-tools `capabilities` seam (`CapabilityProbe`/`CLIProbe` via `packages/vrooli-cli-go`), feeding `models` selector and `backends` fallback messaging. |

If shared infrastructure starts using product vocabulary, move that
piece back into the owning domain or split a new domain first.

## Extension Rules

Add product behavior by adding or updating the owning domain, not by
growing generic buckets.

For a normal proto-backed domain:

1. Add proto messages and service methods under
   `packages/proto/schemas/image-tools/v1/<domain>/`.
2. Add API domain code under `api/internal/<domain>/`.
3. Add transport code under `api/handlers/<domain>/`.
4. Register schemas/endpoints in `api/internal/modules/registry.go`
   and mount the module in `api/main.go`.
5. Add CLI commands under `cli/domains/<domain>/`.
6. Add UI API wrappers under `ui/src/api/<domain>.ts` and UI feature
   code under `ui/src/features/<domain>/`.
7. Update selectors, strings, endpoints, tests, and the docs contract
   in `docs/manifest.json`.

### Adding an operation

Adding a new image operation is the most common extension and has a fixed
recipe that keeps the headless-completeness and ≥1-standalone invariants
intact:

1. **Proto first** — add the operation's request/result messages and
   service method under
   `packages/proto/schemas/image-tools/v1/<domain>/`; image bytes stay
   blob-referenced, not inlined.
2. **Backend provider(s)** — implement at least one standalone
   (non-ComfyUI) provider in `backends`; optionally add ComfyUI and
   BYOK-cloud providers. The CPU fallback path is mandatory for AI ops.
   Registration fails if no standalone provider exists.
3. **Model registry entry** — for AI ops, add the declarative `models`
   entry (hardware requirements, capability labels, license, checksum,
   default-for-op marker) so the selector can place it on a probed host.
4. **Handler / CLI / UI** — wire the API handler, the CLI command (full
   headless parity), and the UI feature (interactive widget only where the
   op genuinely benefits, always with an accessible fallback).
5. **Jobs + measures** — route heavy ops through `jobs`; declare the op's
   measure block so latency/throughput/queue-wait/fallback-usage are
   observable.

For detailed product ownership, update [`DOMAINS.md`](DOMAINS.md).
For persistence and retention, update [`DATA.md`](DATA.md). For
temporal behavior, update [`FLOWS.md`](FLOWS.md). For provider and model
resources, update [`INTEGRATIONS.md`](INTEGRATIONS.md).

## Architecture Maturity

This scenario is **pre-implementation**: the PRD scope is decided and the
domain map is authored, but the operation domains, backend abstraction,
model registry, job queue, and storage seam are not yet built. The table
reflects the inherited `react-vite` template maturity plus the planned
target shape. Update it as each domain becomes green.

| Area | Maturity | Evidence | Remaining Drift |
|---|---|---|---|
| API | Template-ready, domains planned | Module registry, per-domain schema, documented seams from `react-vite`. | All eleven product domains (`ops`/`generation`/`enhancement`/`analysis`/`models`/`backends`/`jobs`/`storage`/`recipes`/`automation`/`measures`) are planned, not built; `notes` example must be removed. |
| UI | Template-ready, features planned | Feature folders, typed API clients, selector/i18n registries, modeltest helpers. | Operation forms, before/after slider, mask-painting canvas, progress-with-cancel, and model-management settings are planned. |
| CLI | Template-ready, parity planned | Domain command groups wrap API calls and render reports. | Full headless parity across every op and the block-once job wait verb are planned. |
| Docs | Contract-ready | Manifest v2 registers docs, maturity, stages, and validation hints; concept docs authored to PRD. | Reference docs and per-domain deep docs fill in as domains land. |
| Infra seams | Designed, not wired | `backends` ladder, `models` selector, `jobs` queue, `storage` blobstore boundary specified in this doc; `capabilities` seam over the root CLI host-inventory contract built (Phase 0 done). | The `capabilities` host-facts prerequisite (root CLI `cli/v1` host-inventory contract + the image-tools `capabilities` seam) is satisfied; remaining seams wire as domains land. |

Use `docs/manifest.json` as the documentation contract. The declared
`maturity` values are expected to be maintained by agents and later
grounded by Knowledge Observatory validation.

## Intentional Deviations

Record deviations from the template or from Vrooli scenario standards
when they are deliberate and durable.

| Date | Deviation | Reason | Revisit Trigger |
|---|---|---|---|
| 2026-06-16 | Opaque image bytes travel over a REST `multipart/form-data` edge instead of inside proto payloads; only metadata is proto-typed and blobs are referenced by handle. | Inlining megabyte-to-hundred-megabyte image bytes in Connect-JSON is wasteful and brittle; the multipart edge is the standard `RESTReasonMultipartUpload` path. | Revisit only if proto/Connect gains efficient first-class binary streaming that obsoletes the multipart edge. |
| 2026-06-16 | Heavy AI providers (ComfyUI, per-op model resources) and BYOK cloud are declared `required:false` and spun up on demand rather than as hard dependencies. | Deterministic ops must work zero-download; AI must be opt-in and CPU-capable so the scenario runs on a laptop or a GPU server. | Revisit if a profile mandates a fixed AI tier always present. |
| 2026-06-16 | GPU detection is NVIDIA-first; AMD/Intel/Apple-Silicon are deferred to platform `internal/hostinventory` hardening surfaced through the root CLI host-inventory contract (OT-P2-003). | The detection is owned upstream and AMD/Intel/Apple support is real engineering deferred to P2. | Revisit when `internal/hostinventory` lands ROCm/Intel/Apple-Silicon probing. |

## Documentation Architecture

Scenario docs follow the same ownership rule as code: one durable
question, one canonical home.

| Concern | Canonical Document |
|---|---|
| System map and extension rules | `docs/concepts/ARCHITECTURE.md` |
| Product capabilities and bounded contexts | `docs/concepts/DOMAINS.md` |
| Workflows and state transitions | `docs/concepts/FLOWS.md` |
| Data ownership, retention, and migrations | `docs/concepts/DATA.md` |
| Resources, scenarios, and external services | `docs/concepts/INTEGRATIONS.md` |
| Monetization and packaging | `docs/business/MONETIZATION.md` |
| Go-to-market strategy | `docs/business/GO-TO-MARKET.md` |
| Deployment tiers and readiness | `docs/operations/DEPLOYMENT.md` |
| Operator procedures | `docs/operations/RUNBOOK.md` |
| Telemetry, metrics, and alerts | `docs/operations/OBSERVABILITY.md` |
| Seams and test doubles | `docs/internal/SEAMS.md` |
| Testing strategy | `docs/internal/TESTING.md` |
| Known drift and deferred work | `docs/internal/PROBLEMS.md` |
| Change history | `docs/internal/PROGRESS.md` |

Every durable scenario document should be registered in
`docs/manifest.json`. Put deep domain-specific documentation under
`docs/domains/<domain>/` when `DOMAINS.md` would become noisy.

## Cross-References

- [`START-HERE.md`](../START-HERE.md) — first implementation workflow
- [`QUICKSTART.md`](../QUICKSTART.md) — clone-to-running flow
- [`DOMAINS.md`](DOMAINS.md) — bounded contexts and ownership
- [`FLOWS.md`](FLOWS.md) — workflow and state-transition map
- [`DATA.md`](DATA.md) — data ownership and storage
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — dependency contracts
- [`../business/MONETIZATION.md`](../business/MONETIZATION.md) — commercial story
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — seam registry
- [`../internal/TESTING.md`](../internal/TESTING.md) — test patterns
- [`../internal/ERROR-HANDLING.md`](../internal/ERROR-HANDLING.md) — error semantics
- [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md) — known issues / tech debt
- [`../internal/PROGRESS.md`](../internal/PROGRESS.md) — lifecycle log
