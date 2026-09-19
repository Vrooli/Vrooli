# API Endpoints — Personal Planner

Human-readable reference for the API. The machine-readable
source of truth is [`.vrooli/endpoints.json`](../../.vrooli/endpoints.json) —
doc generators, Postman collection builders, and SDK stubs read it
directly. The CI gate fails if the JSON drifts from the registered
handlers or from the CLI commands it claims to mirror.

Wire shapes for every endpoint live in
`packages/proto/schemas/personal-planner/v1/<domain>/<file>.proto`.
Proto-typed calls use generated Connect-RPC handlers and clients.
Tests, handlers, UI clients, and CLI handlers all consume generated
types — no hand-written struct mirror exists to drift.

Connect-RPC errors use Connect's canonical error envelope and code set.
REST exceptions, such as multipart uploads, use the template error
envelope (`packages/proto/schemas/personal-planner/v1/shared/errors.proto`):

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
| **CLI** | `personal-planner status` |

```bash
curl "http://localhost:${API_PORT}/health"
```

The proto type lives at `packages/proto/schemas/personal-planner/v1/shared/health.proto`
and mirrors `api-core/health.Response` field-for-field.

---

## Planned product service surface (target contract)

> **Status: planned.** Only `health` (above) and the removable `notes`
> example (below) exist in the scaffold today. Everything in this section
> describes the **target** Connect-RPC surface derived from the
> implementation plan's logical endpoint inventory (plan §20.2). The
> route names are illustrative logical operations under the versioned
> `vrooli.personal_planner.v1.<domain>` namespace — not assertions that
> a handler already exists. They are recorded here so the API contract is
> visible before the code is written; each becomes a real subsection
> (auth, request/response proto, error codes, CLI mirror) as its domain
> lands. See [`../concepts/DOMAINS.md`](../concepts/DOMAINS.md) for the
> 13 product domains that own these operations.

### The proto-service rule and its REST exceptions

Connect-RPC proto services are the **default and required transport** for
every product operation. A literal REST path is a documented exception,
not a choice — the CI contract rejects a REST route unless it carries one
of exactly four `RESTReason` justifications:

1. **multipart upload** — a request body of opaque bytes that a
   Connect-JSON envelope cannot carry (the worked `notes` attachment
   example below);
2. **inbound webhook** — a third-party POST whose shape the caller
   controls (e.g. a provider push channel);
3. **third-party redirect / callback** — an OAuth-style browser redirect
   that must land on a plain URL (provider-calendar connection callback);
4. **operational probe** — an unauthenticated liveness/readiness endpoint
   a load balancer or `curl` must read without a Connect client (the
   `/health` route above).

Every other operation — including all mutations, queries, proposals,
forecasts, focus transitions, and sharing — is a proto RPC. Responses
return canonical stored records (not bare `success: true`), each mutation
takes actor + workspace from authenticated context, accepts an
idempotency key where effects can be retried, checks expected revisions,
and returns the resulting revision plus a trace/request ID (plan §20.1).

### Per-domain logical operations

One `<Domain>Service` per product domain, mounted at
`POST /vrooli.personal_planner.v1.<domain>.<Domain>Service/<Method>`:

| Domain (service) | Key planned operations | Notable behavior |
|---|---|---|
| workspace | `GetProfile`, `UpdateProfile`, `ListAvailabilityRules`, `UpdateAvailabilityRule`, `UpdateAvailabilityException`, `GetAppearance`, `UpdateAppearance` | Preview material capacity effects before broad policy changes; accepted settings invalidate proposals. |
| work | `CreateWork`, `GetWork`, `UpdateWork`, `ArchiveWork`, `ReviseEffort`, `UpdateRemaining`, `CompleteWork`, `ReopenWork`, `Capture` | Estimate / remaining / actual never collapse; completion is explicit; source-owned status routes through its owner. |
| calendar | `QueryRange`, `CreateEvent`, `UpdateEvent`, `CancelEvent`, `CreateRoutine`, `UpdateRoutine`, `MoveOccurrence`, `SkipOccurrence`, `AllocateDate`, `AllocateTimed`, `SplitAllocation`, `MoveAllocation`, `LockAllocation`, `ReleaseAllocation` | Split/move conserve demand (not delete-plus-create); recurring edits require instance/series scope. |
| goals | `CreateGoal`, `UpdateGoal`, `RecordOutcomeProgress`, `LinkWork`, `CreateMilestone`, `CompleteMilestone` | Achievement is never derived from elapsed time alone. |
| commitments | `CreateDraft`, `AcceptPromise`, `ProposeRevision`, `AcceptRevision`, `Withdraw`, `ListHistory`, `InspectCommitment` | A forecast update can never invoke revision acceptance. |
| capacity | `Query`, `Explain` | Returns known + unknown quantities and fragmentation, not one ambiguous percentage. |
| planning | `GenerateProposal`, `GetProposal`, `CompareProposal`, `RejectProposal`, `ApplyProposal`, `UndoApplication`, `WhatIf` | Generation may be async; apply reloads the stored proposal, revalidates, and commits atomically. |
| forecasts | `GetLatest`, `RequestRecompute`, `GetSnapshot`, `GetHistory`, `GetExplanation` | Freshness + model/scenario labels accompany every result; no accepted allocations created. |
| focus | `StartSession`, `PauseSession`, `ResumeSession`, `ChangePhase`, `FinishSession`, `GetCurrentSession` | One authoritative session per person; transitions carry an expected session revision. |
| focus (actuals) | `AddActual`, `CorrectActual`, `ClassifyActual`, `RemoveActual` | Provenance and coverage survive corrections; overlaps are checked. |
| review | `GetDailySummary`, `GetWeeklySummary`, `SaveReflection`, `ListInsights`, `InspectInsight`, `AcceptAdjustment`, `DismissInsight`, `ResetDerivedProfile` | No mandatory completion ceremony; accepting an insight is a separate auditable setting change. |
| integrations (sources) | `RegisterSource`, `IngestIntent`, `IngestChange`, `QueryScheduleProjection` | Registration is privileged; app scope cannot become whole-workspace access by default. |
| integrations (providers) | `InitiateConnection`, `HandleCallback` *(REST redirect exception)*, `ListCalendars`, `SelectCalendars`, `Sync`, `Disconnect` | No provider write endpoints in R1; secrets are never returned to the UI. |
| sharing | `PreviewProjection`, `CreateGrant`, `RedeemGrant`, `RevokeGrant`, `ListOwnerGrants`, `ViewGrantedRecords` | Separate viewer DTOs and authorization path; server-side field masks. |
| notifications | `GetPreferences`, `UpdatePreferences`, `ListNotices`, `AcknowledgeNotice` | Delivery and read state are separate. |
| workspace (data/ops) | `Export`, `ValidateImport`, `ApplyImport`, `RequestDeletion`, `GetJobStatus` | Native restore/import requires preview then explicit apply. |

Error handling follows the plan's error model (§20.5): `VALIDATION_FAILED`,
`FORBIDDEN` / `NOT_FOUND`, `REVISION_CONFLICT`, `PROPOSAL_STALE`,
`CONSTRAINT_VIOLATION`, `SESSION_ALREADY_ACTIVE`, `SOURCE_UNAVAILABLE`,
`SOURCE_CONSTRAINT_STALE`, `PROVIDER_REAUTH_REQUIRED`,
`INSUFFICIENT_INPUT`, `LIMIT_EXCEEDED`, and
`RATE_LIMITED` / `TEMPORARILY_UNAVAILABLE`. These map onto Connect's
canonical code set at the handler edge; raw provider errors, SQL detail,
private source payloads, and hidden record IDs never reach a user-facing
message.

---

## Domain endpoints — `<domain>`

Each product domain exposes its endpoints under
`POST /vrooli.personal_planner.v1.<domain>.<Domain>Service/<Method>`
for proto-typed Connect-RPC calls, with REST exceptions (such as
multipart uploads) mounted at explicit REST paths. Document your
domain's endpoints here as you build them — one section per RPC, with
its auth, request/response proto shapes, error codes, and CLI mirror.

The scaffold ships one fully worked CRUD vertical slice as a copyable
reference (see the Notes section below); `template-manager detemplate
<scenario>` removes it once your real domains are green.

---

## Notes (CRUD reference)

`notes` is the scaffold's canonical worked CRUD vertical slice, including
the one binary-upload REST exception. It is **not** product scope — it is
a copyable reference removed by `template-manager detemplate
personal-planner` once the first real domain is green. Copy its layering
(thin Connect handler, service-owned validation, generated types) when
adding a real domain, then delete it.

<!-- EXAMPLE-DOMAIN:notes START -->
### Example domain — `notes` (removed by `template-manager detemplate`)

The `notes` domain is the canonical worked example. Copy its layering
when adding the first non-trivial mutation in your scenario, then
remove it.

#### `POST /vrooli.personal_planner.v1.notes.NotesService/ListNotes`

List notes through the generated Connect-RPC service, newest-first.

| | |
|---|---|
| **Auth** | None (template default; scenarios add auth as needed) |
| **Response** | `ListNotesResponse { notes: Note[] }` (capped at 100 by `notes.Service`) |
| **Errors** | `500 internal` — repository read failure |
| **CLI** | `personal-planner notes list` |

```bash
curl -X POST "http://localhost:${API_PORT}/vrooli.personal_planner.v1.notes.NotesService/ListNotes" \
  -H 'Content-Type: application/json' \
  -d '{}'
```

UI and CLI code should normally use the generated client instead of
calling this path by hand.

#### `POST /vrooli.personal_planner.v1.notes.NotesService/CreateNote`

Create a note through the generated Connect-RPC service.

| | |
|---|---|
| **Auth** | None (template default) |
| **Request** | `CreateNoteRequest { title: string (required), body: string (optional) }` |
| **Response** | `CreateNoteResponse { note: Note }` |
| **Errors** | `invalid_argument` — missing/whitespace-only title<br>`internal` — repository write failure |
| **CLI** | `personal-planner notes create --title <title> [--body <body>]` |

```bash
curl -X POST "http://localhost:${API_PORT}/vrooli.personal_planner.v1.notes.NotesService/CreateNote" \
  -H 'Content-Type: application/json' \
  -d '{"title":"first","body":"hello"}'
```

Title validation (non-empty after whitespace trim) lives in
`internal/notes/service.go`, **not** the handler. The Connect handler
only translates `notes.ErrInvalidNote` into `invalid_argument`.

#### `POST /vrooli.personal_planner.v1.notes.NotesService/GetNote`

Fetch a note by id through the generated Connect-RPC service.

| | |
|---|---|
| **Auth** | None (template default) |
| **Request** | `GetNoteRequest { id: string }` |
| **Response** | `GetNoteResponse { note: Note }` |
| **Errors** | `not_found` — no note with that id<br>`internal` — repository read failure |
| **CLI** | `personal-planner notes get <id>` |

```bash
curl -X POST "http://localhost:${API_PORT}/vrooli.personal_planner.v1.notes.NotesService/GetNote" \
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
| **CLI** | `personal-planner notes attach <id> --file <path>` |

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

Defined in `packages/proto/schemas/personal-planner/v1/notes/notes.proto`.
<!-- EXAMPLE-DOMAIN:notes END -->

---

## Adding a new endpoint

For a new domain, copy the worked vertical slice in the fenced example
above first, then replace it once your real domain is green.

For an endpoint inside an existing domain:

1. Add or extend the `.proto` messages and service in
   `packages/proto/schemas/personal-planner/v1/<domain>/`, then run
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
