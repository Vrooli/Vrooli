# API Endpoints — Scenario to Plugin

Human-readable reference for the API. The machine-readable
source of truth is [`.vrooli/endpoints.json`](../../.vrooli/endpoints.json) —
doc generators, Postman collection builders, and SDK stubs read it
directly. The CI gate fails if the JSON drifts from the registered
handlers or from the CLI commands it claims to mirror.

Wire shapes for every endpoint live in
`packages/proto/schemas/scenario-to-plugin/v1/<domain>/<file>.proto`.
Proto-typed calls use generated Connect-RPC handlers and clients.
Tests, handlers, UI clients, and CLI handlers all consume generated
types — no hand-written struct mirror exists to drift.

Connect-RPC errors use Connect's canonical error envelope and code set.
REST exceptions, such as multipart uploads, use the template error
envelope (`packages/proto/schemas/scenario-to-plugin/v1/shared/errors.proto`):

```json
{ "code": "<canonical_code>", "message": "<human readable>", "details": [...] }
```

Canonical REST codes used today: `invalid_request` (400),
`not_found` (404), `internal` (500). Add to the proto enum when a new
REST-exception failure mode appears.

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
| **CLI** | `scenario-to-plugin status` |

```bash
curl "http://localhost:${API_PORT}/health"
```

The proto type lives at `packages/proto/schemas/scenario-to-plugin/v1/shared/health.proto`
and mirrors `api-core/health.Response` field-for-field.

---

## Domain endpoints

> **Status: planned surface.** This section documents the API this
> scenario is contracted to expose, derived from `PRD.md` and
> `docs/concepts/DOMAINS.md`. No handler is implemented yet, and this
> document is marked `draft` in `docs/manifest.json` to say so.
> Regenerate it from the proto once the first domain ships.

Each product domain exposes its endpoints under
`POST /vrooli.scenario_to_plugin.v1.<domain>.<Domain>Service/<Method>`
for proto-typed Connect-RPC calls, with REST exceptions (such as
multipart uploads) mounted at explicit REST paths.

Errors use standard Connect codes. Two conventions are specific to this
scenario and are load-bearing:

- **`failed_precondition` means a gate is closed.** It is the response
  for every ordering refusal — signing before conformance, publishing
  before a release decision. The message names the closed gate and the
  remedy. It is never a generic error.
- **A refusal is a successful response, not a fault.** Callers must not
  treat `failed_precondition` as a transport problem to retry.

### `declaration` — `DeclarationService`

| Method | Purpose | Notes |
|---|---|---|
| `GetDeclaration` | Resolve and return a scenario's plugin declaration snapshot. | `not_found` when the scenario declares nothing. `invalid_argument` when the declaration fails schema validation. |
| `ListReadiness` | Fleet publish-readiness for every scenario. | Returns each prerequisite with its state and marks the first blocking one. Backs the readiness board. |
| `GetReadiness` | Publish-readiness for one scenario. | Names the blocking prerequisite explicitly; never a bare boolean. |

### `composition` — `CompositionService`

| Method | Purpose | Notes |
|---|---|---|
| `BuildPackage` | Compose an Agent Plugins tree for a scenario at a source commit. | `failed_precondition` when readiness reports a blocking prerequisite. A failed build is terminal; retry produces a new package. |
| `GetPackage` | One package with its gate-ladder position. | The primary read for "why can't I publish this". |
| `ListPackages` | Packages filtered by scenario, state, or commit. | |
| `ListComponents` | Per-component status for one package. | Components validate independently; one invalid component does not mark others invalid. |
| `GetArtifact` (REST) | `GET /api/v1/packages/{id}/artifact` | REST exception for opaque bytes, mirroring the example domain's attachment path. Metadata stays proto-typed. |

### `conformance` — `ConformanceService`

| Method | Purpose | Notes |
|---|---|---|
| `RunCheck` | Run every conformance rule against a composed package. | All rules run even after the first failure, so one call returns the complete finding list. |
| `GetRun` | One conformance run with its findings. | |
| `ListFindings` | Findings filtered by rule, severity, or file. | Each finding carries file, offset, and rule id — never skill body content. |
| `GetDriftDetail` | Drift outcome plus the pinned `cli-manifest` revision. | The revision is what distinguishes a skill regression from a CLI change. |

### `attestation` — `AttestationService`

| Method | Purpose | Notes |
|---|---|---|
| `RunAttestation` | Scan, redaction-check, sign, and attach provenance and SBOM. | `failed_precondition` when the conformance record is absent or failing. Non-resumable: a failure restarts from scan. |
| `GetAttestation` | Signature, provenance, and SBOM references for a package. | References only; bytes are fetched from the capture store. |
| `VerifyAttestation` | Re-verify a signature and provenance against the recorded digest. | Read-only. Safe to call on published versions. |

### `rehearsal` — `RehearsalService`

| Method | Purpose | Notes |
|---|---|---|
| `StartRehearsal` | Provision a sandbox, install twice, exercise documented commands. | Long-running. `failed_precondition` unless the package is `attested`. |
| `GetRehearsal` | Journey manifest, gate dispositions, per-command results. | |
| `CancelRehearsal` | Cancel a running rehearsal. | Teardown is guaranteed. Result is `unavailable`, never `failed` — the package was not judged. |
| `StreamRehearsal` | Server-stream progress events. | Redacted at capture time. |

### `distribution` — `DistributionService`

| Method | Purpose | Notes |
|---|---|---|
| `ReportVerdict` | Emit the `TargetVerdict` to `deployment-manager`. | Evidence references only; never artifact bytes. |
| `Publish` | Publish a rehearsed package to selected channels. | `failed_precondition` without a passing release decision for the same source commit. Per-channel outcomes; no implicit rollback. |
| `GetPublication` | Per-channel publication state for one package. | A channel is `published` only after retrieval confirmation. |
| `Revoke` | Withdraw a published version from every channel that received it. | Deliberately not gated on a release decision. Idempotent. |
| `GetRevocation` | Revocation state, including `revoked_partial` and the channels still carrying the artifact. | |
| `ListChannels` | Configured channel adapters and their capabilities. | Includes whether a channel supports withdrawal. |
| `GetAttribution` | Per-plugin, per-channel install and referrer counts. | Aggregates only — no IPs, user ids, or fingerprints. |

<!-- EXAMPLE-DOMAIN:notes START -->
### Example domain — `notes` (removed by `template-manager detemplate`)

The `notes` domain is the canonical worked example. Copy its layering
when adding the first non-trivial mutation in your scenario, then
remove it.

#### `POST /vrooli.scenario_to_plugin.v1.notes.NotesService/ListNotes`

List notes through the generated Connect-RPC service, newest-first.

| | |
|---|---|
| **Auth** | None (template default; scenarios add auth as needed) |
| **Response** | `ListNotesResponse { notes: Note[] }` (capped at 100 by `notes.Service`) |
| **Errors** | `500 internal` — repository read failure |
| **CLI** | `scenario-to-plugin notes list` |

```bash
curl -X POST "http://localhost:${API_PORT}/vrooli.scenario_to_plugin.v1.notes.NotesService/ListNotes" \
  -H 'Content-Type: application/json' \
  -d '{}'
```

UI and CLI code should normally use the generated client instead of
calling this path by hand.

#### `POST /vrooli.scenario_to_plugin.v1.notes.NotesService/CreateNote`

Create a note through the generated Connect-RPC service.

| | |
|---|---|
| **Auth** | None (template default) |
| **Request** | `CreateNoteRequest { title: string (required), body: string (optional) }` |
| **Response** | `CreateNoteResponse { note: Note }` |
| **Errors** | `invalid_argument` — missing/whitespace-only title<br>`internal` — repository write failure |
| **CLI** | `scenario-to-plugin notes create --title <title> [--body <body>]` |

```bash
curl -X POST "http://localhost:${API_PORT}/vrooli.scenario_to_plugin.v1.notes.NotesService/CreateNote" \
  -H 'Content-Type: application/json' \
  -d '{"title":"first","body":"hello"}'
```

Title validation (non-empty after whitespace trim) lives in
`internal/notes/service.go`, **not** the handler. The Connect handler
only translates `notes.ErrInvalidNote` into `invalid_argument`.

#### `POST /vrooli.scenario_to_plugin.v1.notes.NotesService/GetNote`

Fetch a note by id through the generated Connect-RPC service.

| | |
|---|---|
| **Auth** | None (template default) |
| **Request** | `GetNoteRequest { id: string }` |
| **Response** | `GetNoteResponse { note: Note }` |
| **Errors** | `not_found` — no note with that id<br>`internal` — repository read failure |
| **CLI** | `scenario-to-plugin notes get <id>` |

```bash
curl -X POST "http://localhost:${API_PORT}/vrooli.scenario_to_plugin.v1.notes.NotesService/GetNote" \
  -H 'Content-Type: application/json' \
  -d '{"id":"abc123"}'
```

`notes.ErrNoteNotFound` returned by the service is translated into the
typed `not_found` Connect error at the handler edge.

#### `POST /api/v1/notes/{id}/attachments`

Upload opaque file bytes through the documented REST multipart exception.
The response is still proto-typed metadata.

| | |
|---|---|
| **Auth** | None (template default) |
| **Path params** | `id` — note identifier |
| **Request** | `multipart/form-data` with `file` part |
| **Response** | `UploadAttachmentResponse { attachment: Attachment }` |
| **Errors** | `400 invalid_request` — malformed multipart or missing file<br>`404 not_found` — no note with that id<br>`500 internal` — blob or metadata persistence failure |
| **CLI** | `scenario-to-plugin notes attach <id> --file <path>` |

```bash
curl -X POST "http://localhost:${API_PORT}/api/v1/notes/abc123/attachments" \
  -F file=@./example.png
```

#### `Note` shape

| Field | Type | Notes |
|---|---|---|
| `id` | string (UUID) | Server-generated |
| `title` | string | Required, non-empty after trim |
| `body` | string | Optional |
| `created_at` | `google.protobuf.Timestamp` | Server-set on create |
| `updated_at` | `google.protobuf.Timestamp` | Server-set on create / future update |
| `attachment_keys` | `string[]` | Keys of uploaded note attachments |

Defined in `packages/proto/schemas/scenario-to-plugin/v1/notes/notes.proto`.
<!-- EXAMPLE-DOMAIN:notes END -->

---

## Adding a new endpoint

For a new domain, copy the worked vertical slice in the fenced example
above first, then replace it once your real domain is green.

For an endpoint inside an existing domain:

1. Add or extend the `.proto` messages and service in
   `packages/proto/schemas/scenario-to-plugin/v1/<domain>/`, then run
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
