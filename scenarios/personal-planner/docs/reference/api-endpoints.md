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

> **Status: incrementally implemented.** `health`, work, calendar, workspace,
> focus, goals, review, and the provider-connection read model are now real
> Connect-RPC surfaces. The remaining
> rows are still target contract work; this section describes the **target**
> surface derived from the
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
| work | `CreateWork`, `GetWork`, `ListWorkItems`, `GetTodayPlan`, `UpdateWork`, `ArchiveWork`, `ReviseEffort`, `UpdateRemaining`, `CompleteWork`, `ReopenWork`, `Capture` | Estimate / remaining / actual never collapse; completion is explicit; Today/Plan projections are labeled until schedule data exists. |
| calendar | `ListTodayAllocations`, `ListAllocations`, `CreateAllocation`, `CarryForwardAllocation`, `ListRoutines`, `CreateRoutine`, `ListRoutineOccurrences`, `SkipRoutineOccurrence`, `RescheduleRoutineOccurrence` | Accepted allocations are durable, non-overlapping, and distinct from work recommendations. Carry-forward preserves the original as history, creates one new accepted placement transactionally, and is idempotent on retry. Native routines expand in local time and support revision-checked overrides. Today capacity reports active imported busy minutes/events and unions them with accepted work without double-counting overlap; provider recurrence remains open. |
| goals | `ListGoals`, `CreateGoal`, `UpdateGoalProgress`, `ListMilestones`, `CreateMilestone`, `UpdateMilestoneStatus` | Goals choose manual or milestone-derived progress; milestones carry criteria, due dates, optional validated work-item links, prerequisite edges, and revision-safe completion blocked until prerequisites are complete. |
| commitments | `CreateDraft`, `AcceptPromise`, `ProposeRevision`, `AcceptRevision`, `Withdraw`, `ListHistory`, `InspectCommitment` | A forecast update can never invoke revision acceptance. |
| capacity | `Query`, `Explain` | Returns known + unknown quantities and fragmentation, not one ambiguous percentage. |
| planning | `GenerateProposal`, `GetProposal`, `CompareProposal`, `RejectProposal`, `ApplyProposal`, `UndoApplication`, `WhatIf` | Generation may be async; apply reloads the stored proposal, revalidates, and commits atomically. |
| forecasts | `GetLatest`, `RequestRecompute`, `GetSnapshot`, `GetHistory`, `GetExplanation` | Freshness + model/scenario labels accompany every result; no accepted allocations created. |
| focus | `StartFocus`, `PauseFocus`, `ResumeFocus`, `EndFocus`, `GetCurrentSession`, `RecordManualActual`, `ListActuals`, `ListActualCorrections`, `CorrectActual` | One authoritative current session; transitions carry an expected session revision. Manual/approximate actuals are local-date records, and bounded correction history exposes preserved before/after provenance. |
| review | `GetDailySummary`, `GetWeeklySummary`, `GetReflection`, `SaveReflection`, `ListInsights`, `InspectInsight`, `AcceptAdjustment`, `DismissInsight`, `ResetDerivedProfile` | Daily/weekly summaries combine focus-session and manual-actual time, keep focus-session counts separate, and label planned/unrecorded coverage honestly. The current UI offers selective carry-forward through Calendar with a target-day capacity preview and durable optional daily reflection; learning insights remain open. |
| integrations (sources) | `RegisterSource`, `IngestIntent`, `IngestChange`, `QueryScheduleProjection` | Registration is privileged; app scope cannot become whole-workspace access by default. |
| integrations (providers) | `ListConnections`, `CreateFixtureConnection`, `SyncConnection`, `DisconnectConnection` | The provider-neutral read model and synthetic fixture adapter are real; live OAuth/callback, credential-owner storage, calendar selection, imported-event projections, and provider write endpoints remain open. Secrets are never returned to the UI. |
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
