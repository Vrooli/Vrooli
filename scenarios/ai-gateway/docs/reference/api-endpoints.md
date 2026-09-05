# API Endpoints — AI Gateway

Human-readable reference for the API. The machine-readable
source of truth is [`.vrooli/endpoints.json`](../../.vrooli/endpoints.json) —
doc generators, Postman collection builders, and SDK stubs read it
directly. The CI gate fails if the JSON drifts from the registered
handlers or from the CLI commands it claims to mirror.

Wire shapes for every endpoint live in
`packages/proto/schemas/ai-gateway/v1/<domain>/<file>.proto`.
Proto-typed calls use generated Connect-RPC handlers and clients.
Tests, handlers, UI clients, and CLI handlers all consume generated
types — no hand-written struct mirror exists to drift.

Connect-RPC errors use Connect's canonical error envelope and code set.
REST exceptions, such as multipart uploads, use the template error
envelope (`packages/proto/schemas/ai-gateway/v1/shared/errors.proto`):

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
| **Errors** | None — always returns 200 with `status: "unhealthy"` if a dependency fails |
| **CLI** | `ai-gateway status` |

```bash
curl "http://localhost:${API_PORT}/health"
```

The proto type lives at `packages/proto/schemas/ai-gateway/v1/shared/health.proto`
and mirrors `api-core/health.Response` field-for-field.

---

## Product Domain Endpoints

Each product domain exposes endpoints under
`POST /vrooli.ai_gateway.v1.<domain>.<Domain>Service/<Method>` for
proto-typed Connect-RPC calls. REST exceptions are allowed only for the
template's documented exception classes.

The Phase 2 contract foundation mounted the provider-neutral service
surfaces below. Gateway request validation is implemented. Inventory
reads provider role policy through resource-owned CLI commands. Routing
now previews deterministic profile policy, executes through resource
commands, and persists redacted route evidence. Conformance exposes the
first deterministic scanner for provider boundary and gateway adoption
findings and the shared Test Genie `ScenarioValidationService` provider
contract for the `ai-conformance` phase.

### Gateway

#### `POST /vrooli.ai_gateway.v1.gateway.GatewayService/ValidateGatewayRequest`

Validates the provider-neutral gateway request envelope before any
provider adapter can run.

| | |
|---|---|
| **Auth** | None yet; policy/auth gates land with operator surfaces. |
| **Request** | `ValidateGatewayRequestRequest { request: GatewayRequest }` |
| **Response** | `ValidateGatewayRequestResponse { valid: bool, issues: repeated ValidationIssue, accepted_profiles: repeated string }` |
| **Errors** | Connect `internal` only for unexpected handler failures. Validation failures are returned as `valid=false` with field-level issues. |
| **CLI** | `ai-gateway gateway validate --role <role> [--kind text|embedding|extract] [--profile local-first]` |

The validator rejects provider-specific caller authority such as
provider names, provider URLs, concrete model slugs, credentials,
embedding dimensions, and context-window metadata. It also rejects
secret requests paired with profiles that may require remote providers.
Attachments are ephemeral: inline image bytes are checked with a header-only
dimension parser, references are opaque application identifiers, public
privacy is rejected, and count/size ceilings are named in field-level issues.

### Inventory

#### `POST /vrooli.ai_gateway.v1.inventory.InventoryService/ListProviderRoles`

Lists provider role inventory by executing bounded resource-owned
policy commands through the provider command seam:
`resource-ollama policy roles --json` and
`resource-openrouter policy roles --json`. AI Gateway normalizes role
name, capabilities, locality, status, and policy schema version, but it
does not expose provider credentials or become the concrete model
catalog authority.

| | |
|---|---|
| **Request** | `ListProviderRolesRequest { provider: string }` |
| **Response** | `ListProviderRolesResponse { roles: repeated ProviderRole, warnings: repeated string }` |
| **CLI** | `ai-gateway inventory roles [--provider ollama|openrouter]` |

#### `POST /vrooli.ai_gateway.v1.inventory.InventoryService/SmokeProvider`

Runs the same bounded resource policy command and maps missing binaries,
timeouts, malformed JSON, non-zero exits, and empty role inventories to
typed provider status fields.

| | |
|---|---|
| **Request** | `SmokeProviderRequest { provider: string }` |
| **Response** | `SmokeProviderResponse { provider: string, status: string, code: string, message: string, exit_code: int32, warnings: repeated string }` |
| **CLI** | `ai-gateway inventory smoke --provider <provider>` |

### Routing

#### `POST /vrooli.ai_gateway.v1.routing.RoutingService/PreviewRoute`

Runs the gateway validator and deterministic routing policy without
running inference. The response includes eligible and rejected
candidates, the selected provider, fallback eligibility, and policy
reasons. Profiles enforce local-only, local-first, remote-only,
quality-first, cheap-first, and privacy-sensitive locality semantics.

| | |
|---|---|
| **Request** | `PreviewRouteRequest { request: GatewayRequest }` |
| **Response** | `PreviewRouteResponse { valid: bool, issues: repeated ValidationIssue, candidates: repeated RouteCandidate, selected_provider: string, policy_reasons: repeated string, fallback_allowed: bool, route_plan_id: string }` |
| **CLI** | `ai-gateway routing preview --role <role> [--kind text|embedding|extract] [--profile local-first]` |

#### `POST /vrooli.ai_gateway.v1.routing.RoutingService/ExecuteRoute`

Executes a validated provider-neutral request through the selected
resource command. AI Gateway passes transient input via stdin to the
resource command and persists only route metadata/evidence before
returning provider output to the caller.

| | |
|---|---|
| **Request** | `ExecuteRouteRequest { request: GatewayRequest, input_text: string }` |
| **Response** | `ExecuteRouteResponse { valid: bool, issues: repeated ValidationIssue, evidence: RouteEvidence, output_text: string, policy_reasons: repeated string, applied: AppliedSettings }` |
| **Sampling** | `request.sampling.temperature` is optional. A candidate whose resolved role does not declare `honored` support is skipped; when none remain the route fails with `failure_class=unsupported_sampling` rather than silently sampling another way. A skipped candidate is never recorded against provider health. |
| **Persistence** | Fails closed if route evidence cannot be recorded. Evidence stores redaction flags and attachment metadata (count, bytes, hash, dimensions), never raw prompt/response content or image bytes. `sampling_temperature` is nullable so an omitted control stays distinguishable from a deterministic `0`. |
| **CLI** | `ai-gateway routing execute --role <role> --input <text> [--profile local-first]` — the CLI exposes no sampling flag here; see cli-commands.md. |

#### `POST /vrooli.ai_gateway.v1.routing.RoutingService/ListRouteEvidence`

Lists recent redacted route evidence events, optionally scoped by
scenario.

| | |
|---|---|
| **Request** | `ListRouteEvidenceRequest { limit: int32, scenario: string }` |
| **Response** | `ListRouteEvidenceResponse { events: repeated RouteEvidence }` |
| **CLI** | `ai-gateway routing evidence-list [--scenario <scenario>] [--limit 20]` |

#### `POST /vrooli.ai_gateway.v1.routing.RoutingService/GetRouteEvidence`

Fetches one redacted route evidence event by event ID.

| | |
|---|---|
| **Request** | `GetRouteEvidenceRequest { event_id: string }` |
| **Response** | `GetRouteEvidenceResponse { event: RouteEvidence }` |
| **CLI** | `ai-gateway routing evidence-show <event-id>` |

### Conformance

#### `POST /vrooli.ai_gateway.v1.conformance.ConformanceService/ScanScenario`

Scans a scenario tree for unsafe AI/provider coupling and gateway
adoption signals. Findings report rule ID, severity, path with line
number when available, message, and remediation. The scanner is
conservative and does not store source contents in findings.

| | |
|---|---|
| **Request** | `ScanScenarioRequest { scenario: string, path: string }` |
| **Response** | `ScanScenarioResponse { scenario: string, maturity_level: string, findings: repeated ConformanceFinding, recommendations: repeated string }` |
| **CLI** | `ai-gateway conformance scan --scenario <scenario>` or `--path <path>` |

Initial rule coverage includes direct Ollama/OpenRouter HTTP usage,
provider secret/url env vars, concrete model slugs, hard-coded context
windows, hard-coded embedding dimensions near vector code, missing
embedding metadata, direct resource command usage without visible role
policy, unreviewed exception markers, and missing gateway adoption
signals.

#### `POST /vrooli.scenario_validation.v1.ScenarioValidationService/ValidateScenario`

Runs the same scanner through the shared Test Genie validation-provider
contract. The response maps native findings into the maturity ladder
embedded in `scenarios/ai-gateway/.vrooli/test-genie.json` and packs the
native scan summary in `native_detail`.

| | |
|---|---|
| **Request** | `ValidateScenarioRequest { scenario: string, path: string, include_execution: bool }` |
| **Response** | `ValidateScenarioResponse { scenario: string, status: ValidationStatus, assessment: common.v1.MaturityAssessment, native_detail: google.protobuf.Any, metrics: common.v1.ExecutionMetrics }` |
| **CLI** | `ai-gateway validation validate --scenario <scenario> [--include-execution]` |

`PreviewFix` and `ApplyFix` are mounted for the shared contract. They
return guidance/no-op responses until deterministic safe migrations are
implemented.

CLI mirrors: `ai-gateway validation preview-fix --scenario <scenario>`
and `ai-gateway validation apply-fix --scenario <scenario>` call the
shared RPCs directly. Apply is currently a documented no-op response
from the API.

### Inference

#### `POST /vrooli.ai_gateway.v1.inference.InferenceService/Run`

Runs one schema-constrained typed inference request. The gateway submits
the schema through the provider's native structured-output field and still
validates the returned value locally.

| | |
|---|---|
| **Request** | `RunRequest { source, schema_json, instruction, role, turns, attachments, profile, sampling: SamplingControls, max_output_tokens: int32 }` |
| **Response** | `RunResponse { value_json, provider, model, validated, usage, error, applied: AppliedSettings }` |
| **Sampling** | `sampling.temperature` is optional and honoured only by roles declaring `overridable: true`. A closed role refuses with `INVALID_REQUEST` and construct `sampling.temperature`; an overridable role whose candidates all decline returns `UNSUPPORTED_SAMPLING`. Absent the field, the role's declared sampling applies, and a candidate that cannot honour *that* omits it and proceeds. |
| **Budget** | `max_output_tokens` is request-level because the budget depends on caller data a role cannot know — one role serves a single draft and a k-candidate set, so sizing for either breaks the other. It is bidirectional: a caller may send a tighter cap than the role default. |
| **Reporting** | `applied` reports `temperature_sent`, `temperature_support`, `max_output_tokens_effective`, and `max_output_tokens_source` on **every** return path including errors — a refused or truncated call is exactly when a caller needs to know the cap. There is deliberately no single "applied temperature": it would have to lie for a provider that accepts and ignores the control. |
| **CLI** | `ai-gateway inference run --role <role> --schema <path> --source <text> [--temperature <n>] [--max-output-tokens <n>]` |

#### `POST /vrooli.ai_gateway.v1.inference.InferenceService/RunBatch`

Runs an ordered batch of sources against one prompt and schema. Results
preserve request order, including failed items, and each carries its own
provider/model and usage accounting.

| | |
|---|---|
| **Request** | `RunBatchRequest { items, schema_json, instruction, role }` |
| **Response** | `RunBatchResponse { results: repeated RunResponse, usage }` |
| **Sampling** | Intentionally absent. A batch is N inputs against one prompt; per-item sampling has no caller. |
| **CLI** | `ai-gateway inference run-batch --role <role> --schema <path> --items <path>` |

---

## Adding a new endpoint

For a new domain, start from the API steer guidance and the generated
Connect handler patterns in this scenario's health module; do not revive
the removed example domain.

For an endpoint inside an existing domain:

1. Add or extend the `.proto` messages and service in
   `packages/proto/schemas/ai-gateway/v1/<domain>/`, then run
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
