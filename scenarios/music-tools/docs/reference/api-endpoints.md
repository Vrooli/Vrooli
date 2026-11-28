# API Endpoints — Music Tools

> **Status (2026-11-15): composition and analysis REST handlers are registered.**
> The generated contracts cover composition, jobs, models, styles, and capacity;
> the API runtime exposes health, composition jobs, analysis/separation jobs,
> results and stems. The `music-mir` model runtime remains unqualified.

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

## Private source REST endpoints

Sources are imported from local files into content-addressed managed storage;
the API never downloads or fetches a supplied `source_url`. One source is
limited to 128 MiB and the private source-owner quota is 1 GiB. Imports require
a title and may retain artist, uploader, acquisition time and an HTTPS YouTube
URL as local provenance. The source SHA-256 determines an immutable blob key.

### `POST /api/v1/sources`

Multipart fields: `file`, `title`, optional `artist`, `uploader`, `source_url`,
and `acquired_at`. Returns the verified SHA-256 and `source_blob_key` used by
the analysis job endpoints.

```bash
curl -F 'file=@reference.m4a' -F 'title=Reference' \
  -F 'source_url=https://www.youtube.com/watch?v=example' "$API/api/v1/sources"
```

### `GET /api/v1/sources/{sha256}` and `/audio`

The metadata path returns the locally stored provenance. The `/audio` path
streams private original bytes with byte-range support for review playback;
responses are `private, no-store` and the stored content is rehashed before
serving. These handlers never contact the external source URL.

## Analysis REST endpoints

Analysis and source separation are durable Music Tools jobs. They use the
shared job manager and its serialized GPU lane; a job retains admission until
the MIR resource confirms cancellation drain. The request names an existing
Music Tools managed source blob. The server binds its SHA-256 into the durable
payload so a source changed after submission fails closed. Output artifacts are
committed to managed blob storage only after digest and type checks.

### `POST /api/v1/analyze`

Queues a partial acoustic analysis. It returns HTTP 202 with `job_id`, `state`,
`operation`, and the pinned source SHA-256. When the job succeeds,
`GET /api/v1/analysis/{id}/result` returns the stored provenance envelope and
analysis object. Current runtime fields include estimated tempo/beat times, key
candidates and margin, energy curves, stem energy and low-confidence acoustic
cut candidates, plus best-effort timestamped transcript segments from the
configurable local Whisper service. Whisper output is a hypothesis, has no
word-level alignment or confidence value, and remains optional if the service
is absent. Music MIR accepts only loopback Whisper URLs, keeping reference audio
local. Semantic section and hook labels and plain-language descriptions are
not yet implemented.

```bash
curl -X POST "$API/api/v1/analyze" -H 'Content-Type: application/json' \
  -d '{"source_blob_key":"library/reference.wav"}'
```

### `POST /api/v1/separate`

Queues vocals, drums, bass and other separation. The result envelope records
the four managed blob keys; fetch an individual stem with
`GET /api/v1/analysis/{id}/stems/{name}` where `name` is one of those four
values. The local MIR adapter is CUDA-only and currently unqualified because
its governed dependency lock/runtime installation is pending.

```bash
curl -X POST "$API/api/v1/separate" -H 'Content-Type: application/json' \
  -d '{"source_blob_key":"library/reference.wav"}'
```

These endpoints are REST exceptions registered in the analysis module and
`.vrooli/endpoints.json`; CLI analyze/separate commands remain planned.
Requirements: `MUS-P0-009`–`011`, `MUS-P1-001`, `MUS-P1-003`.

### `GET /api/v1/analysis/library-sync`

Reports pending, delivered, and blocked forwarding counts plus the latest
delivery error. It does not return result payloads. After a job has durably
reached `succeeded`, Music Tools discovers it from the jobs table and stores an
outbox row pointing to the immutable result blob. A background worker forwards
that document to Music Library. Library downtime cannot affect the producer
job; retry state survives restarts, uses backoff from 10 seconds up to about
43 minutes, and retries indefinitely. Duplicate posts are idempotent by
track-SHA plus analysis-job identity. URL resolution uses api-core discovery;
`MUSIC_LIBRARY_URL` can override it. The worker quiesces during scenario
maintenance and stops during API cleanup. Invalid result identity is retained
as blocked; other delivery errors stay pending for retry.

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
| `GET /api/v1/jobs/{id}/derivation-receipt` | REST JSON | Return the owner-validated immutable source/output receipt for a succeeded `transform.remix` or `transform.sample` job; returns 412 when the job has no qualifying receipt |
| `POST /api/v1/jobs/{id}/cancel` | REST JSON | Request cancellation; read the terminal job for its outcome |
| `GET /api/v1/takes`, `/api/v1/takes/{id}`, `/api/v1/takes/{id}/audio` | REST JSON / audio bytes | Enumerate peers and retrieve managed WAV output |
| `/api/v1/styles/*`, `/api/v1/pool/*` | REST JSON | Portable style and pool operations |

Generated Connect service declarations do not establish runtime mounting.
Analysis/reference-audio multipart composition remains outside this implemented
text/style composition route.

Music Tools now registers deterministic CPU remix and sample operations on the shared durable jobs manager. The submit routes accept only managed source refs with exact SHA-256s, bounded excerpts (0.25 to 120 seconds) and declared inherited-rights scope. Sample passes receive short fades at both ends to reduce loop clicks. A successful job records the exact source/output hashes, ranges, tool version and rights scope; `ffprobe` must report an audio stream whose duration covers the requested excerpt. Consumers retrieve the receipt by job ID. The audio route re-hashes the stored WAV against that receipt before serving private byte ranges. This code path has not yet been run against audio: live behavior and audibility remain gated on the bounded operator decision and resource/rights checks.

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
incomplete drafts; compilation requires positive duration and nonnegative BPM. Provider admission validates actual structured controls: ACE needs positive BPM; SA3 uses BPM 0/no keyscale. Export/import
preserves the actual parameter tuple and caption, while import creates a fresh
custom record. Create reports malformed/invalid JSON as400 and domain conflicts
as409; import retains its existing409 decoding/domain response (body-read/size
errors are400). This source contract needs normal API adoption before claiming
the running service implements it; deterministic tests alone are not deployment.

## Configurable composition default

`GET /api/v1/styles/default` returns `{"style_id":"launch-instrumental"}` on a
fresh database. `PUT /api/v1/styles/default` accepts `{"style_id":"<saved-id>"}`
and refuses missing styles. The selection is stored per database scope; it
changes only composition requests that omit `style_id`. The CLI surfaces this
as `music-tools styles default-get` and `styles default-set <style-id>`.

These routes are implemented as REST handlers but are not yet in the generated
endpoint catalogue. Their typed Connect contract remains a follow-up.

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
for songs with vocals and instrumental beds. ACE declares `en` and `ko` for the
optional `vocal_language` input. Omission records the model-declared default (`en`);
an explicit language is recorded only after the resource returns the matching
language receipt. This is a parameter contract, not an audible-language quality claim.

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

New admissions snapshot provider and the entire compiled style, including lyrics. Old queued payloads without that
snapshot retain the old instrumental behavior; absent old seeds retain base zero.
Recovery does not sample another seed or reinterpret missing lyrics using a newly
recreated style. Only historical payloads lacking a compiled snapshot retain the legacy style lookup contract; missing provider means ACE.

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
absence; new jobs preserve their compiled snapshot across style edits or deletion.

Compile/admission accepts unset sampler zeros, inference steps0..100 and finite
guidance0..20; invalid drafts can still be saved, but cannot queue generation.
SFT responses must match explicitly authored nonzero guidance, including a legacy
style with no variant precondition served by SFT. Turbo's effective guidance1 is
legitimate normalization. Request encoding errors refuse HTTP dispatch rather than
sending an empty body. These bounds describe the current resource protocol, not a
promise that every model accepts every value (turbo retains its narrower step cap).

## Default instrumental provider and raw receipt

Blank-variant instrumentals use SA3; explicit ACE variants and authored lyrics use ACE.
SA3 refuses structured BPM/keyscale, `shift`, duration outside 1..120, and more than
32 steps before enqueueing. Use caption text for tempo/key intentions. Unset steps/CFG
resolve to 8/1 at admission. No fallback occurs on readiness or licensing refusal.
`MUSIC_LICENSE_LANE=permissive` excludes/refuses SA3.

`GET /api/v1/takes/{id}/receipt` returns exact raw SA3 resource receipt bytes, with
`X-Music-Receipt-SHA256`; ordinary missing/legacy takes do not invent a receipt.
`generation_profiles[takeID]` in durable job metadata links its receipt reference/SHA,
audio SHA and actual seed/steps/CFG. ACE profile fields apply only to ACE takes.
SA3 receipts validate the qualified composed/runtime/source recipe and PCM24 WAV;
requested instrumentality and successful validation do not establish audible quality.
Publication writes the protected raw receipt before audio. A later publication failure
never advertises a ready take; original resource output remains available and a retained
receipt can diagnose partial publication. Cancellation aborts transport without publishing
a new take; the resource owns cancellation/drain of its generation slot.

### Capability-based lyric routing

`Style.lyrics` remains backend-neutral. New blank-variant requests select the
registry's preferred model supporting their requested sung-lyrics capability. Current
SA3 cannot output intelligible vocals, so current lyric requests select ACE; this is
not an unconditional backend rule. An explicit SA3 lyric request returns a400 with
the actual limitation and immutable upstream source, without enqueueing or dropping
text. Explicit ACE variants retain their real controls. Whole-style/provider snapshots
and legacy missing-provider ACE semantics remain unchanged. Selection support is
distinct from successful runtime readiness and audible lyric adherence.

New admissions also freeze the selected exact variant, including an implicit lyric
model preference. A differently loaded runtime is refused through the existing
expected-variant/returned-variant contract; Music never switches a resident model.
Previously queued snapshots with blank variant retain their historical semantics.

### ACE job capacity lease identity

Admission metadata retains `capacity_claim_id` and `capacity_claim_generation` from
the same acquired claim. Job heartbeats send that exact positive generation; a
stale generation fails/cancels owned work rather than looking up a current claim
or assuming generation1. Heartbeats do not change generation in the canonical
protocol. Terminal cleanup joins renewal and releases the captured claim once.
Canonical release is intentionally ID-only (no generation guard/CLI flag); Music
retains the reference but does not invent a guarded-release guarantee. Missing
identity/generation refuses admission. CLI failure errors include at most4KiB of
stderr plus a truncation marker. These facts concern ACE job claims; SA3 continues
to use resource-owned residency without a second Music lease.
