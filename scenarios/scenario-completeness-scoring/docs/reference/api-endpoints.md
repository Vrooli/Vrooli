# API Endpoints — Scenario Completeness Scoring

Human-readable reference for the API. The machine-readable
source of truth is [`.vrooli/endpoints.json`](../../.vrooli/endpoints.json) —
doc generators, Postman collection builders, and SDK stubs read it
directly. The CI gate fails if the JSON drifts from the registered
handlers or from the CLI commands it claims to mirror.

Wire shapes for every endpoint live in
`packages/proto/schemas/scenario-completeness-scoring/v1/<domain>/<file>.proto`.
Proto-typed calls use generated Connect-RPC handlers and clients.
Tests, handlers, UI clients, and CLI handlers all consume generated
types — no hand-written struct mirror exists to drift.

Connect-RPC errors use Connect's canonical error envelope and code set.
REST exceptions, such as multipart uploads, use the template error
envelope (`packages/proto/schemas/scenario-completeness-scoring/v1/errors/errors.proto`):

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
| **CLI** | `scenario-completeness-scoring status` |

```bash
curl "http://localhost:${API_PORT}/health"
```

The proto type lives at `packages/proto/schemas/scenario-completeness-scoring/v1/health/health.proto`
and mirrors `api-core/health.Response` field-for-field.

---

## Scoring

### `POST /vrooli.scenario_completeness_scoring.v1.scoring.ScoreService/GetScore`

Compute the full cached status payload for one scenario: maturity rung
"as of digest td:…", 0–100 composite with classification and per-group
breakdown, prioritized recommendations with point impact, the phased
action plan, per-phase freshness verdicts with a copy-pastable refresh
command, optional best-effort importance enrichment, and any collector
degradations. Zero network on the server's core score path; importance
is the only optional network touch and is omitted on miss. Warm latency
budget is <1s.

| | |
|---|---|
| **Auth** | None |
| **Request** | `GetScoreRequest { scenario: string }` — directory name under the scenarios root |
| **Response** | `GetScoreResponse { scenario, category, maturity, composite, freshness, recommendations[], action_plan[], degradations[], calculated_at, importance? }` |
| **Errors** | `not_found` — unknown scenario<br>`internal` — unexpected assembly failure (collector failures degrade instead of erroring) |
| **CLI** | `scenario-completeness-scoring score get <scenario> [--json]` |

```bash
curl -X POST "http://localhost:${API_PORT}/vrooli.scenario_completeness_scoring.v1.scoring.ScoreService/GetScore" \
  -H 'Content-Type: application/json' \
  -d '{"scenario":"web-search"}'
```

UI and CLI code should normally use the generated client instead of
calling this path by hand. The full payload shape is documented
field-by-field in
`packages/proto/schemas/scenario-completeness-scoring/v1/scoring/scoring.proto`.

---

## Adding a new endpoint

For an endpoint inside an existing domain:

1. Add or extend the `.proto` messages and service in
   `packages/proto/schemas/scenario-completeness-scoring/v1/<domain>/`, then run
   `make generate`.
2. Implement the generated handler method in
   `handlers/<domain>/connect_handler.go`; keep it thin.
3. Update endpoint metadata in `handlers/<domain>/module.go`.
4. If the endpoint has a CLI mirror, update
   `api/cmd/gen-endpoints/cli_commands_seed.json`.
5. Run `make endpoints`; do not edit
   [`.vrooli/endpoints.json`](../../.vrooli/endpoints.json) by hand.
6. Update this document and add tests for the touched layers.
7. Add a row to [`internal/SEAMS.md`](../internal/SEAMS.md) if you
   introduced a new interface that production wires once and tests
   substitute.

The CI gate enforces endpoint-manifest freshness and command-seed
consistency.

## Cross-references

- [`cli-commands.md`](cli-commands.md) — CLI commands that mirror these endpoints
- [`configuration.md`](configuration.md) — env vars (e.g., `API_PORT`)
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md#proto-as-the-canonical-contract) — proto bridge details
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — handler/service/repository seams
- [`../internal/TESTING.md`](../internal/TESTING.md) — endpoint test patterns
