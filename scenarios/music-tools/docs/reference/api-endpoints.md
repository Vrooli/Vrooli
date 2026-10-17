# API Endpoints — Music Tools

> **Status (2026-09-19): composition metadata and REST edges are shipped.**
> The generated contracts cover composition, jobs, models, styles, and capacity;
> the current API runtime exposes health, styles, composition jobs, takes, pool
> transitions, and audio retrieval. Connect handlers for the infrastructure
> services are staged for the infrastructure-domain port.

Human-readable reference for the API. The machine-readable
source of truth is [`.vrooli/endpoints.json`](../../.vrooli/endpoints.json) —
doc generators, Postman collection builders, and SDK stubs read it
directly. The CI gate fails if the JSON drifts from the registered
handlers or from the CLI commands it claims to mirror.

The generated contracts in `packages/proto/schemas/music-tools/v1` define
portable styles, composition submit parameters, jobs and takes. The implemented
composition API uses REST JSON; style/submit ingress decodes generated proto JSON,
while internal durable payloads add server-owned snapshots. Errors on these REST
handlers are HTTP status plus text. Planned Connect surfaces below are not a claim
that every generated service is mounted.

---

## System

### `GET /health`

Service health check. Returns API readiness plus dependency status.
Also mounted at `/api/v1/health` for client callers.
This is an operational REST exception by design: lifecycle systems,
load balancers, and curl probes must be able to read it without a Connect
client.

| | |
|---|---|
| **Auth** | None |
| **Response** | `Response { status: string, readiness: bool, service: string, timestamp: string, version: string, uptime_seconds: int64, dependencies: map<string, DependencyStatus> }` |
| **Errors** | None — always returns 200 and reports an unhealthy status if a dependency fails |
| **CLI** | `music-tools status` |

```bash
curl "http://localhost:${API_PORT}/health"
```

The proto type lives at `packages/proto/schemas/music-tools/v1/shared/health.proto`
and mirrors `api-core/health.Response` field-for-field.

---

## Domain endpoints — `<domain>`

Each product domain exposes its endpoints under
`POST /vrooli.music_tools.v1.<domain>.<Domain>Service/<Method>`
for proto-typed Connect-RPC calls, with REST exceptions (such as
multipart uploads) mounted at explicit REST paths. Document your
domain's endpoints here as you build them — one section per RPC, with
its auth, request/response proto shapes, error codes, and CLI mirror.

The scaffold ships one fully worked CRUD vertical slice as a copyable
reference (see the fenced example below); `template-manager detemplate
<scenario>` removes it once your real domains are green.

### Composition endpoint surface

The maintained composition handlers currently expose:

| Edge | Transport | Behavior |
|---|---|---|
| `POST /api/v1/compose` | REST JSON, generated `SubmitRequest` proto JSON | Submit a batch; returns job ID, queued state and take count |
| `GET /api/v1/jobs/{id}`, `/api/v1/jobs`, `/api/v1/jobs/{id}/wait` | REST JSON | Read durable status or block once until terminal |
| `POST /api/v1/jobs/{id}/cancel` | REST JSON | Request cancellation; read the terminal job for its outcome |
| `GET /api/v1/takes`, `/api/v1/takes/{id}`, `/api/v1/takes/{id}/audio` | REST JSON / audio bytes | Enumerate peers and retrieve managed WAV output |
| `/api/v1/styles/*`, `/api/v1/pool/*` | REST JSON | Portable style and pool operations |

Generated Connect service declarations do not establish runtime mounting.
Analysis/reference-audio multipart composition remains outside this implemented
text/style composition route.

Three contract points that follow from recorded decisions rather than taste:

- **A composition job produces N artifacts.** The result references a set of
  takes, each with its own provenance — model, licence lane, applied capacity
  rung, seed, and the caption *as the model received it*. A response shape that
  assumes one output per job will have to be broken later.
- **The applied degradation rung travels with the result** (`OT-P0-003`). It is
  not a log line; a caller must be able to see that a take was produced on a
  degraded rung. The first measured rung is CPU-offloaded DiT at 3.1× the
  resident cost — see [`../internal/PERFORMANCE.md`](../internal/PERFORMANCE.md).
- **Any caption rewriting is reported.** If a planner rewrote the caller's
  brief, the rewritten text is in the report, not swallowed. See the
  2026-09-18 caption-ownership decision in
  [`../internal/DECISIONS.md`](../internal/DECISIONS.md).

---

## Implemented style JSON ingress

The current REST handlers mount `POST /api/v1/styles` (create),
`POST /api/v1/styles/import`, `GET /api/v1/styles/{id}/export` and
`GET /api/v1/styles/{id}/compile`. Create and import accept the same direct,
portable `Style` document; do not wrap it in the planned Connect
`CreateStyleRequest.style` envelope. Canonical fields are flat:

```json
{"id":"driving-rave","name":"Driving Rave","caption":"energetic instrumental rave with driving drums","bpm":200,"keyscale":"F# minor","duration":48,"inference_steps":8,"guidance_scale":1,"variant":"acestep-v15-turbo"}
```

Proto JSON camelCase aliases `inferenceSteps` and `guidanceScale` are also valid.
Legacy nested `params` remains accepted, using `bpm`, `key_scale`, `duration`,
`steps`, `guidance` and `variant` inside that object. Do not mix flat and nested
parameter representations. Both paths use the shared portable decoder;
unrecognized fields, invalid values and trailing JSON are refused rather than
discarded. One complete request must fit the existing portable-import1MiB bound;
oversized requests are refused before saving, including valid prefixes.

Records are custom (`builtin` cannot grant built-in identity), and creation
timestamps are server-owned; recognized legacy timestamps do not override them.
Built-in IDs remain immutable and cannot be shadowed. Creation still permits
incomplete drafts; compilation requires positive BPM and duration. Export/import
preserves the actual parameter tuple and caption, while import creates a fresh
custom record. Create reports malformed/invalid JSON as400 and domain conflicts
as409; import retains its existing409 decoding/domain response (body-read/size
errors are400). This source contract needs normal API adoption before claiming
the running service implements it; deterministic tests alone are not deployment.

## Adding a new endpoint

For a new domain, copy the worked vertical slice in the fenced example
above first, then replace it once your real domain is green.

For an endpoint inside an existing domain:

1. Add or extend the `.proto` messages and service in
   `packages/proto/schemas/music-tools/v1/<domain>/`, then run
   `make generate`.
2. Implement the generated handler method in
   `handlers/<domain>/connect_handler.go`; keep it thin.
3. Update endpoint metadata in `handlers/<domain>/module.go`.
4. If the endpoint has a CLI mirror, bind it (or list it in `omitted[]`
   with a reason) in `cli/manifest.json` — the single source of truth for
   the CLI surface.
5. Run `make endpoints`; do not edit
   [`.vrooli/endpoints.json`](../../.vrooli/endpoints.json) by hand.
6. Update this document and add tests for the touched layers.
7. Add a row to [`internal/SEAMS.md`](../internal/SEAMS.md) if you
   introduced a new interface that production wires once and tests
   substitute.

The CI gate enforces endpoint-manifest freshness and the API↔CLI mapping
contract (every Connect endpoint is bound or omitted in `cli/manifest.json`).

## Cross-references

- [`cli-commands.md`](cli-commands.md) — CLI commands that mirror these endpoints
- [`configuration.md`](configuration.md) — env vars (e.g., `API_PORT`)
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md#proto-as-the-canonical-contract) — proto bridge details
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — handler/service/repository seams
- [`../internal/TESTING.md`](../internal/TESTING.md) — endpoint test patterns

## Implemented lyrics and composition controls

`Style.lyrics` is optional, belongs to the style, and survives canonical flat and
supported nested-legacy style import/export. Empty or absent legacy lyrics resolve
to `[instrumental]`; other text and newlines are preserved. Use separate style IDs
for songs with vocals and instrumental beds. The current resource language is
English (`en`); this contract does not claim multilingual or listening quality.

`POST /api/v1/compose` consumes generated `SubmitRequest` proto JSON: snake_case
or canonical camelCase aliases are accepted, unknown/duplicate fields and trailing
JSON are refused before enqueueing, and the document limit is1MiB. Examples:

```json
{"style_id":"driving-rave","takes":2,"duration":48}
{"styleId":"driving-rave","takes":1,"seed":"424242","shift":1}
```

`seed` is an optional signed-int64 **batch base**, accepting proto JSON string or
integer form. Omission chooses a fresh base once at admission and saves it in the
opaque durable payload; explicit zero is repeatable. Each take derives an exact
uint32 effective seed. Optional `shift` must be finite1..5 for the current ACE
protocol; omission preserves the loaded profile default. It is sampling control,
not an instrumentality override. Compose cannot override style lyrics.

Admission snapshots resolved style lyrics. Old queued payloads without that
snapshot retain the old instrumental behavior; absent old seeds retain base zero.
Recovery does not sample another seed or reinterpret missing lyrics using a newly
recreated style. Other style fields still use the existing style lookup contract.

Job metadata provides compact input facts `lyrics_intent`, `lyrics_source`,
`lyrics_sha256`, `lyrics_bytes`, `seed_base` and `seed_base_source`; full authored
text stays in the style and durable payload. These describe requested intent.
For completed responses, `generation_profiles` is bounded JSON keyed by take ID,
containing verified actual seed, planner, steps, guidance, shift, exact sent-lyrics
hash and optional paired semantic-code hash/count. `completed_take_ids` and
`generation_profile_scope=completed-takes` make partial batches explicit.
`seed_receipt=verified-uint32` applies to those outputs. Uniform model/rung summaries
are omitted when mixed; per-take provenance remains authoritative.

Missing, invalid or mismatched seed/profile receipts refuse ingestion rather than
fall back to an old resource. Historical takes without these receipts have unknown
actual seeds: their stored planned number does not prove what upstream used.
Completed facts can persist on failure/cancel if that worker wins the first terminal
write. An already-authoritative terminal result is never amended by a stale worker;
late facts can therefore remain unknown. This fence does not provide a distributed
lease for competing running workers.

For a launch video, require source-job instrumental intent **and audition the actual
audio for lyrics/vocals**. A style ID or `[instrumental]` request cannot prove heard
absence; delete/recreate style-ID drift is not repaired by this change.

Compile/admission accepts unset sampler zeros, inference steps0..100 and finite
guidance0..20; invalid drafts can still be saved, but cannot queue generation.
SFT responses must match explicitly authored nonzero guidance, including a legacy
style with no variant precondition served by SFT. Turbo's effective guidance1 is
legitimate normalization. Request encoding errors refuse HTTP dispatch rather than
sending an empty body. These bounds describe the current resource protocol, not a
promise that every model accepts every value (turbo retains its narrower step cap).
