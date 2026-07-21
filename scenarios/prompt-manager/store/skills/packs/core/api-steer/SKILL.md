## Steer focus: API Steer

Prioritize **scenario APIs as stable, proto-first, Connect-RPC contracts** in `scenarios/{{TARGET}}/api/`. Treat the API as the canonical entrypoint into the scenario's core logic — every UI and CLI surface, and every other scenario, consumes it through generated typed clients. The four REST exceptions enumerated in `path:templates/scenarios/react-vite/api/internal/module/module.go` are the only escape hatch.

Required reading:
- `prompt-manager skill read knowledge-observatory-tools` — read and update scenario documentation through the canonical docs CLI.
- `prompt-manager skill read interoperability-steer` — proto schema, generation, casing, validation, and discovery rules that this skill assumes.

Read first when present:
- `scenarios/{{TARGET}}/docs/concepts/ARCHITECTURE.md` — durable mental model, surfaces, domain map.
- `scenarios/{{TARGET}}/docs/internal/SEAMS.md` — boundary registry (Connect handler factories, repositories, integrations).
- `scenarios/{{TARGET}}/docs/internal/PROBLEMS.md` — unresolved API drift and deferred refactors.

Optional context:
- `prompt-manager skill read screaming-architecture-audit` — when API domain boundaries are unclear or shared infrastructure has absorbed product vocabulary.
- `prompt-manager skill read cli-steer` — when designing API changes that the CLI must mirror.

---

> **Template example domain — delete on generation.** The react-vite template carries a `notes` domain as a fully-worked example (`path:templates/scenarios/react-vite/{proto,api}/...`). It is starter scaffolding to study, not boilerplate to keep — when a new scenario is generated from the template, the `notes` domain must be removed and replaced with the scenario's actual domains. In this skill the identifiers `<domain>`, `<Domain>Service`, `<Resource>`, and `<Resources>` are placeholders. Substituting them is the prompt that should remind you whether a `notes` package still lingering in your scenario is intentional.

---

### 1. Scope Boundaries

**In scope:**
- proto-first RPC method design (`Service.Method` naming, request/response messages, error mapping)
- Connect-RPC handler wiring, `Module` registration, and `EndpointDescriptor` parity with `.vrooli/endpoints.json`
- the four canonical REST exceptions (`multipart_upload`, `webhook_receiver`, `third_party_shape`, `ops_probe`) and their `RESTException` tagging
- consistency conventions: error envelopes, pagination, idempotency, auth, observability
- additive evolution of proto contracts; deprecation paths
- shared API substrate (validation, auth interceptors, error translation, request context)

**Out of scope:**
- deep domain modeling (use `screaming-architecture-audit`)
- proto package layering, casing, and generation workflow (use `interoperability-steer`)
- UI rendering or CLI argument parsing (use `cli-steer`)
- broad rewrites that change behavior without API-shape payoff
- adding new product features under the banner of API cleanup

---

### 2. API Maturity Model

Assess the API surface against this ladder. A scenario may be L4 for one domain and L1 for another.

| Level | Name | What exists | Main drift risk |
|---|---|---|---|
| 0 | Ad-hoc handlers | Routes hand-registered, error shapes vary, payloads are untyped JSON. | Every new endpoint reinvents auth/error/pagination. |
| 1 | Consistent envelope | A handler factory or middleware enforces request context, auth, and a uniform error envelope; pagination is one canonical shape. | Domain logic still lives in handlers; types are still hand-written. |
| 2 | Proto schemas per domain | Each domain has proto messages in `packages/proto/schemas/{{SCENARIO_ID}}/v1/<domain>/`; generated types exist but consumers may still hand-write. | Schemas drift from runtime shapes; consumers cast or bypass generated types. |
| 3 | Connect-RPC end to end | Every proto-owned operation is a Connect-RPC `Service.Method`; UI and CLI consume generated Connect clients; `Module.Mount` registers Connect handlers, not hand-wired REST routes. | REST endpoints linger without `RESTException` tags; `EndpointDescriptor.Path` may carry hand-authored REST literals. |
| 4 | Codegen drift gate | `cmd/gen-endpoints` `validateTransport` enforces that every `EndpointDescriptor.Path` is either a Connect procedure path or carries a `RESTException` with one of the four `RESTReason*` constants; `.vrooli/endpoints.json` is regenerated from `Module.Endpoints`. | Manual `endpoints.json` edits or new REST reasons added without proto rejection. |
| 5 | Validated boundary | `protovalidate` constraints enforce request invariants at ingress; endpoint-descriptor parity tests assert UI/CLI command ↔ RPC mapping; deprecations carry an explicit migration window and are caught by drift tests. | Remaining drift requires a new validator rule, not a docs note. |

Use the level to identify the next concrete move: tag, schema, wire, gate, or validate.

---

### 3. API Domain Archetype Decision Model

Choose the wire shape by behavior, not folder uniformity. Pick a primary archetype, then add secondary traits.

| Archetype | Use when | Canonical wire shape |
|---|---|---|
| RPC service (CRUD/command) | A domain entity is created, listed, fetched, updated, archived, or a domain command is invoked | Connect-RPC unary methods: `<Domain>Service.List<Resources>`, `<Domain>Service.Create<Resource>`, `<Domain>Service.Archive<Resource>` |
| Streaming RPC | Server pushes a sequence of typed events (logs, progress, watch) | Connect-RPC server-streaming method: `RunsService.StreamRunEvents` |
| Long-running operation | Work outlives the request; client polls or subscribes | Pair of Connect methods: `ImportsService.StartImport` + `ImportsService.GetImport` (and optionally a streaming variant) |
| Blob / multipart-upload exception | Opaque file bytes cross the wire and proto cannot express the request | REST `POST` tagged `RESTException{Reason: multipart_upload}`; response metadata remains proto-typed |
| Webhook-receiver exception | A third party (Stripe, GitHub) dictates the request shape | REST handler tagged `RESTException{Reason: webhook_receiver}`; payload validated against a hand-written schema, then translated to proto-typed domain events |
| Third-party-shape exception | OpenAPI passthrough, OAuth callback, or other contract owned by a third party | REST handler tagged `RESTException{Reason: third_party_shape}` |
| Ops-probe exception | Lifecycle systems / load balancers / curl probes need a generated-client-free reach | REST handler tagged `RESTException{Reason: ops_probe}` (`GET /health`, static HTML wrappers, etc.) |

Decision rule:

```text
Can the request and response be expressed as proto messages?
  YES -> Connect-RPC method on a Service. No exception, no debate.
  NO  -> Identify which of the four RESTReason* constants applies.
         If none apply, the answer is still "express it as proto."
Is the work long-running?
  YES -> Pair Start + Get (and consider streaming).
Is the wire shape dictated by an outside system?
  YES -> third_party_shape or webhook_receiver exception.
Does a kernel-level prober need it without a client?
  YES -> ops_probe exception.
```

`path:templates/scenarios/react-vite/api/internal/module/module.go` is the authoritative source for the four allowed reasons. `path:templates/scenarios/react-vite/api/cmd/gen-endpoints/main.go` (`validateTransport`) is the enforcement point.

---

### 4. Canonical File-Shape Guidance

For React-Vite-template scenarios, the healthy target is:

```text
packages/proto/schemas/{{SCENARIO_ID}}/v1/<domain>/<domain>.proto   # source of truth
packages/proto/gen/go/{{SCENARIO_ID}}/v1/<domain>/                  # generated; do not edit
packages/proto/gen/go/{{SCENARIO_ID}}/v1/<domain>/<domain>_v1connect/  # generated Connect server+client

scenarios/{{TARGET}}/api/
  handlers/<domain>/         # Connect service implementation + Module constructor + EndpointDescriptors
  internal/<domain>/         # domain types, service, repository, schema, workflows, mocks
  internal/module/           # Module/EndpointDescriptor/RESTException contract (template-owned)
  cmd/gen-endpoints/         # codegen + validateTransport gate
```

Preserve these invariants:
- One Connect service per domain. The service implementation lives in `handlers/<domain>/` and delegates to `internal/<domain>/`.
- `Module.Endpoints` is the single source for `.vrooli/endpoints.json`; never hand-edit the JSON.
- Each `EndpointDescriptor.Path` is either a Connect procedure path (e.g. `/{{SCENARIO_ID}}.v1.<domain>.<Domain>Service/List<Resources>`) or carries a `RESTException`.
- REST exceptions live alongside the rest of the domain's `Module`; their `Note` field justifies the choice to the next agent.
- Domain logic does not import transport packages (no `connect`, no `mux`) — handlers translate at the edge and call into the domain.

---

### 5. RPC Method Naming Convergence

Method names are part of the contract — they show up in the procedure path, the generated client, and the CLI. Make them predictable.

| Operation kind | Method pattern | Example |
|---|---|---|
| List a collection | `List<Plural>` | `<Domain>Service.List<Resources>` |
| Get one by id | `Get<Singular>` | `<Domain>Service.Get<Resource>` |
| Create one | `Create<Singular>` | `<Domain>Service.Create<Resource>` |
| Update one | `Update<Singular>` | `<Domain>Service.Update<Resource>` |
| Soft-delete / lifecycle command | `Archive<Singular>`, `Restore<Singular>`, `Cancel<Run>` | `ProjectsService.ArchiveProject` |
| Long-running start | `Start<Verb>` | `ImportsService.StartImport` |
| Long-running poll | `Get<Verb>` | `ImportsService.GetImport` |
| Server-streaming | `Stream<Plural>` or `Watch<Singular>` | `RunsService.StreamRunEvents` |
| Idempotent action | Verb that reads as a transition | `OrdersService.RefundOrder` (not `PostRefund`) |

Avoid:
- HTTP-verb leakage in method names (`PostNote`, `PutNote`).
- Resource-path leakage (`<Domain>V1Create<Resource>`).
- "Misc" service buckets that group unrelated methods.

---

### 6. Consistency Rules

Connect-RPC absorbs much of the boilerplate (status codes via `connect.Code*`, error envelopes via `connect.NewError`). The remaining consistency surface:

**Error mapping.** Translate domain errors to typed `*connect.Error` once, in a shared layer. Use stable `connect.Code` values and a structured `connect.ErrorDetail` (proto message) for field violations or retry hints.

**Pagination.** Pick one shape per scenario and use it for every list method. Cursor-shaped is the default (`page_size`, `page_token`, `next_page_token` in the request/response messages).

**Idempotency.** Connect headers (`Idempotency-Key`) survive the generated client; document which `Create*`/command methods honor them.

**Auth.** One Connect interceptor sets request context (actor, scopes, request id). Authorization checks live close to the domain, never copy-pasted across handlers.

**Observability.** Emit structured logs with request id, scenario id, RPC procedure, latency, outcome, dependency call outcomes. Never log raw secrets.

**Dry-run.** Mutating methods honor the `X-Dry-Run: true` header (set automatically by cli-core when `--dry-run` is passed). Validate normally, short-circuit before the first side effect, return a realistic typed response with a `dry_run` indicator.

---

### 7. REST Exceptions: The Four Reasons

The only mechanically-allowed reasons to keep a hand-authored REST path are the four constants in `path:templates/scenarios/react-vite/api/internal/module/module.go`:

| `RESTReason*` constant | Wire string | Use when |
|---|---|---|
| `RESTReasonMultipartUpload` | `multipart_upload` | Request body is opaque file bytes; proto cannot express `multipart/form-data`. Response metadata still flows through a proto message. |
| `RESTReasonWebhookReceiver` | `webhook_receiver` | A third-party system (Stripe, GitHub, …) dictates the request shape and we cannot ask them to switch to proto. |
| `RESTReasonThirdPartyShape` | `third_party_shape` | OpenAPI passthrough, OAuth callbacks, or other request/response shape owned by a third-party contract. |
| `RESTReasonOpsProbe` | `ops_probe` | Lifecycle systems, load balancers, and curl probes need to reach the endpoint without a generated client (e.g. `GET /health`, browser-facing static HTML). |

Rules:
- Every REST endpoint sets `EndpointDescriptor.RESTException` with one of these four `Reason` values and a free-form `Note` explaining the specific justification.
- Adding a fifth reason is a deliberate architectural decision; it requires editing the template's `module.go` constants, not a local workaround.
- `validateTransport` in `path:templates/scenarios/react-vite/api/cmd/gen-endpoints/main.go` rejects any descriptor whose `Path` is not a Connect procedure path and lacks a valid `RESTException`. Local copies inherit the same gate.

---

### 8. Audit Workflow

1. **Read the docs.** Start with `ARCHITECTURE.md`, `SEAMS.md`, `PROBLEMS.md`, and `.vrooli/endpoints.json`. Treat docs as claims to verify.
2. **Inventory domains.** List every `handlers/<domain>/` package and its `Module` constructor.
3. **Verify proto coverage.** Each domain should have a proto file under `packages/proto/schemas/{{SCENARIO_ID}}/v1/<domain>/` with one `service` block.
4. **Verify Connect wiring.** Each `Module.Mount` should register a Connect handler (or a small number of them), not hand-wired `mux.Router` REST routes — except for the four exceptions.
5. **Verify descriptor parity.** Every `EndpointDescriptor.Path` is either a Connect procedure path or carries a `RESTException` with a valid `Reason`. Run `cmd/gen-endpoints` and confirm `validateTransport` passes.
6. **Assign maturity levels.** Score each domain independently using §2.
7. **Find drift.** Look for the red flags in §9.
8. **Improve incrementally.** Prefer one domain's lift from L2 → L3 over a broad cross-cutting refactor.
9. **Update docs.** Put durable shape decisions in `ARCHITECTURE.md`, boundary/interceptor decisions in `SEAMS.md`, and unresolved drift in `PROBLEMS.md`.

Audit commands:

```bash
# Connect service implementations and Module constructors
rg "connectrpc\.com/connect|_v1connect" scenarios/{{TARGET}}/api --type go
rg "module\.Module\{" scenarios/{{TARGET}}/api/handlers --type go

# REST-exception tagging — every REST path should set one
rg "RESTException\s*:\s*&module\.RESTException" scenarios/{{TARGET}}/api/handlers --type go

# Hand-wired REST routes (red flag unless paired with a RESTException)
rg "r\.HandleFunc|router\.(GET|POST|PUT|DELETE|PATCH)" scenarios/{{TARGET}}/api --type go

# EndpointDescriptor paths — should be Connect procedure paths or REST exceptions
rg "Path:\s*\"/api/v1/" scenarios/{{TARGET}}/api/handlers --type go

# Proto-less domains (handlers package without a matching proto schema)
fd -t d . scenarios/{{TARGET}}/api/handlers
fd "\.proto$" packages/proto/schemas/{{SCENARIO_ID}}

# Codegen gate — validateTransport must pass
cd scenarios/{{TARGET}} && go run ./api/cmd/gen-endpoints --check

# Domain logic leaking into transport
rg "\bconnect\.|\bmux\." scenarios/{{TARGET}}/api/internal --type go
```

---

### 9. Red Flags

- A `handlers/<domain>/` package with no matching `packages/proto/schemas/{{SCENARIO_ID}}/v1/<domain>/`.
- `Module.Mount` registering `r.HandleFunc("/api/v1/...", ...)` for an operation whose payload could be a proto message.
- `EndpointDescriptor.Path` containing a hand-authored REST literal without a `RESTException`.
- `RESTException.Reason` set to a string that is not one of the four `RESTReason*` constants.
- Domain packages (`internal/<domain>/`) importing `connectrpc.com/connect` or `gorilla/mux`.
- Hand-written types in UI or CLI that duplicate generated proto messages.
- `.vrooli/endpoints.json` edited by hand (it must be regenerated from `Module.Endpoints`).
- Mutating Connect methods that ignore the `X-Dry-Run` header.
- New "misc" or "v2" services that have no domain vocabulary.
- Method names that leak HTTP verbs (`PostNote`, `PutNote`).

---

### 10. Safe Refactoring Guidelines

You may:
- introduce a proto schema for a domain that currently has only REST handlers, and migrate the methods one at a time
- rename Connect service methods to match the patterns in §5 (with deprecation aliases if external consumers exist)
- extract a shared Connect interceptor for auth/request-context/error mapping
- tag a legitimate REST exception with the correct `RESTReason*` and a clarifying `Note`
- regenerate `.vrooli/endpoints.json` from `Module.Endpoints`

You must:
- preserve observable behavior and wire compatibility unless the user has approved a breaking change
- keep `validateTransport` passing after every change
- update `ARCHITECTURE.md`, `SEAMS.md`, or `PROBLEMS.md` to reflect the new shape
- update CLI commands and UI clients when method names or shapes change (or record the follow-up in `PROBLEMS.md`)

Challenge yourself before a move:
- Does this make the domain's RPC surface easier to discover from the proto?
- Does it remove a hand-authored REST literal or a hand-written type?
- Would a second agent independently arrive at the same shape from the docs?

---

### 11. Documentation

Use `knowledge-observatory-tools` to read and update stable docs. Avoid creating `API_AUDIT.md` by default. One-off audit reports are acceptable only for a migration handoff; they should have a clear retirement path into `ARCHITECTURE.md`, `SEAMS.md`, or `PROBLEMS.md`.

`ARCHITECTURE.md` should include:
- the API surface map (domains, services, methods)
- API maturity by domain
- REST exceptions with their `RESTReason*` and justification
- shared API substrate (interceptors, error mapping, pagination)

`SEAMS.md` should include:
- Connect interceptors (auth, request context, error translation)
- repository/service seams within each domain
- outbound integration adapters (covered jointly with `interoperability-steer`)

`PROBLEMS.md` should include:
- domains still on REST that should move to Connect
- proto schemas that drift from runtime behavior
- deprecation windows in progress

Recommended architecture additions:

```markdown
## API Surface

| Domain | Service | Methods | Maturity | Notes |
|---|---|---|---|---|

## REST Exceptions

| Endpoint | Reason | Justification | Owner |
|---|---|---|---|
```

---

### **12. Output Expectations**

By the end of this loop, the scenario API should:
- expose every proto-owned operation as a Connect-RPC method via `Module.Endpoints`
- carry a valid `RESTException` on every remaining REST endpoint
- pass `validateTransport` in `cmd/gen-endpoints`
- have a domain-aligned proto schema for every domain that exposes business logic
- keep transport concerns out of `internal/<domain>/` packages
- record unresolved drift in `PROBLEMS.md` rather than leaving it implicit

Avoid superficial changes that rename files or shuffle handlers without improving proto/Connect coverage or the REST-exception story.

Last updated: 2026-05-12
