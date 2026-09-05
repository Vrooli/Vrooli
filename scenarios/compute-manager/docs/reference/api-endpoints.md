# API Endpoints: Compute Manager

Human-readable reference for the API. The machine-readable source of
truth is [`.vrooli/endpoints.json`](../../.vrooli/endpoints.json), which
doc generators, Postman collection builders and SDK stubs read directly.
The CI gate fails if that JSON drifts from the registered handlers or
from the CLI commands it claims to mirror.

## Read this first: implementation status

**Compute Manager is partially implemented.** The scenario was generated
from the `react-vite` template, but the instance, intent, meter and
reconciliation domain contracts and handlers now exist alongside the
health and capabilities endpoints. Provider-live enrollment and several
operator-only/post-launch surfaces remain open.

This document is therefore split into two parts, and they must not be
confused:

| Part | What it describes | Trust level |
|---|---|---|
| Part 1: The surface that exists today | Endpoints that are registered, mounted and reachable right now | Verified against `api/handlers/` and `.vrooli/endpoints.json` |
| Part 2: Remaining/planned surface | Domain methods not yet exposed through the current API/CLI surface | Proposal or deferred work; verify against generated endpoints |

The generated endpoint inventory is authoritative for what is callable.
When a remaining planned method becomes real, update this reference and
regenerate the endpoint inventory in the same change.

Wire shapes for every implemented endpoint live in
`packages/proto/schemas/compute-manager/v1/<domain>/<file>.proto`.
Proto-typed calls use generated Connect-RPC handlers and clients. Tests,
handlers, UI clients and CLI handlers all consume generated types, so no
hand-written struct mirror exists to drift.

Connect-RPC errors use Connect's canonical error envelope and code set.
REST exceptions, such as multipart uploads, use the template error
envelope (`packages/proto/schemas/compute-manager/v1/shared/errors.proto`):

```json
{ "code": "<canonical_code>", "message": "<human readable>", "details": [...] }
```

Canonical REST codes used today: `invalid_request` (400), `not_found`
(404), `internal` (500). Add to the proto enum when a new REST-exception
failure mode appears.

---

## Part 1: The surface that exists today

Everything in this part is registered in
`api/internal/modules/registry.go` and mounted by
`api/internal/server/server.go`.

### System

#### `GET /health`

Service health check. Returns API readiness plus dependency status.
Also mounted at `/api/v1/health` for client callers. This is an
operational REST exception by design: lifecycle systems, load balancers
and curl probes must be able to read it without a Connect client.

| | |
|---|---|
| **Auth** | None |
| **Response** | `Response { status: string, readiness: bool, service: string, timestamp: string, version: string, uptime_seconds: int64, dependencies: map<string, DependencyStatus> }` |
| **Errors** | None. The endpoint answers 200 with an unhealthy status, or 503 when a critical dependency check fails |
| **CLI** | `compute-manager status` |

```bash
curl "http://localhost:${API_PORT}/health"
```

The handler is built in `api/handlers/health/handler.go` on
`api-core/health`. It registers exactly one dependency check today,
`database`, backed by the local `database.Pinger` seam and marked
critical, so a failed SQLite ping flips the response to `unhealthy` with
HTTP 503. The proto type lives at
`packages/proto/schemas/compute-manager/v1/shared/health.proto` and
mirrors `api-core/health.Response` field for field.

The dependency map currently reports the declared scenario dependencies;
provider and business-suite call health remain separate integration
signals. A green `/health` proves the database is reachable, not that
live provider enrollment has succeeded.

#### `GET /api/v1/capabilities/describe`

Describes the scenario's declared dependencies and the recovery actions
currently associated with them, read from the capabilities registry in
`api/internal/capabilities/registry.go`.

| | |
|---|---|
| **Auth** | None |
| **Response** | Registry-rendered JSON |
| **Errors** | `503` when the registry is unconfigured or fails to render |
| **CLI** | None |

```bash
curl "http://localhost:${API_PORT}/api/v1/capabilities/describe"
```

All four scenario dependencies named in the PRD are declared in
`.vrooli/service.json` under `dependencies.scenarios`, and
`api/internal/capabilities/registry.go` declares one capability per
dependency: `landing-page-business-suite`, `vrooli-bridge`, `treasury`
and `offer-desk`. Each carries the manifest's `required` and
`startup_policy` values plus a `vrooli scenario start <slug>` operator
action, and each is probed by a checker that shells out to `vrooli
scenario status <slug> --json`.

What the response does **not** describe is provider adapter state or
domain-level lifecycle state. The registry describes declared dependencies,
not working integrations: an
`available` verdict means the dependency scenario is running, never that
this scenario can call it.

---

## Part 2: Remaining and planned surface

The proto contracts and several handlers for these services now exist.
Methods absent from the generated endpoint inventory remain planned.
Treat the generated bindings and endpoint inventory as the callable
contract.

Each service section maps to exactly one domain, and each domain owns
the operational targets listed against it.

### `intent`, implemented `IntentService` (partial surface)

Owns `OT-P0-002`. The durable record that a request was made, written
with its idempotency key before any provider client is reachable.

| Planned method | Purpose | Notes |
|---|---|---|
| `ListIntents` | Read intents, filtered by state | Read only |
| `GetIntent` | Read one intent by id or idempotency key | Replaying a known key returns the original intent |

There is no `CreateIntent` on the wire. Intent is written by the
instance request path, because the ordering guarantee is worthless if a
caller can request an instance without one.

### `instance`, implemented `InstanceService`

Owns `OT-P0-001`. The lifecycle state machine over `requested`,
`creating`, `running`, `draining` and `destroyed`, plus the two states
reconciliation assigns, `orphaned` and `unknown`.

| Planned method | Purpose | Notes |
|---|---|---|
| `RequestInstance` | Ask for capacity | Writes the intent, reserves credit, then calls the provider. Carries an idempotency key |
| `GetInstance` | Read one instance with its elapsed cost and remaining lifetime | Feeds the operator dashboard row |
| `ListInstances` | Read live inventory | Backs `OT-P1-005` |
| `ExtendInstance` | Move an expiry later | Refused for an instance already destroyed |
| `DestroyInstance` | Terminate and settle | The only terminator |

No caller names a provider. A request describes what it needs, and
provider selection happens behind the adapter interface.

### `provider`, planned `ProviderService`

Owns `OT-P0-001`, `OT-P0-007` and `OT-P1-004`. The adapter interface has
exactly four methods internally: create, describe, list and destroy.

| Planned method | Purpose | Notes |
|---|---|---|
| `ListProviders` | Enumerate registered adapters by identifier | Selection is by identifier alone |
| `DescribeProvider` | Return an adapter's declared billing facts | Rounding behaviour, minimum billable unit, whether a stopped instance bills, whether inbound traffic counts against the transfer allowance |

Billing facts are data an adapter declares, never assumptions buried in
adapter code, because product decisions depend on them.

### `meter`, implemented `MeterService`

Owns `OT-P0-006` and `OT-P1-002`. Reserve, provision, re-reserve on a
heartbeat, settle on teardown, release on failure.

| Planned method | Purpose | Notes |
|---|---|---|
| `GetUsage` | Metered usage for an instance or a tenant | Computed from transitions this scenario caused |
| `ListReservations` | Open and settled reservations | Identifiers only. The business suite holds the wallet |
| `GetCeiling` | The tenant ceiling computed from our own meter | A refusal names the ceiling that was reached |

Hosted compute is cost-bearing, so enforcement runs server-side before
the machine boots and the tier is never read from the request. A refused
reservation short-circuits with no provider call, and out of credit must
be distinguishable from a server error.

### `reconcile`, implemented `ReconcileService` (partial surface)

Owns `OT-P0-003` and `OT-P1-003`. Compares provider inventory against
local records in both directions.

| Planned method | Purpose | Notes |
|---|---|---|
| `RunSweep` | Trigger a bidirectional sweep on demand | Emits findings and mutates no instance |
| `ListFindings` | Read divergences | Present at the provider and absent locally is unaccounted. Present locally and absent at the provider closes the usage window |
| `GetFinding` | Read one finding with its evidence | Operator acts on it, the sweep does not |
| `CompareCost` | Compare metered usage against a provider statement | Alarms beyond a threshold. Provider billing lags, so it is a signal and never a control |

This service reports and never destroys. There is no `ResolveFinding`
that destroys anything, because a reconciler bug must not be able to
delete a paying customer's node. A local-only observation may settle its
known usage window through the meter-owned callback. Mark, then let an
operator handle any provider-side action. The
operator procedure is
[Quarantine An Unaccounted Instance](../operations/RUNBOOK.md#quarantine-an-unaccounted-instance).

### `expiry`, planned `ExpiryService`

Owns `OT-P0-004`. Two enforcement points for one guarantee.

| Planned method | Purpose | Notes |
|---|---|---|
| `RunSweep` | Trigger the expiry sweep on demand | The background loop runs this on an interval |
| `ListExpiring` | Instances approaching their expiry | Drives the dashboard's expiring-soon state |

The second enforcement point is not an endpoint at all. It is a timer
rendered into the instance's own first-boot configuration that powers
the instance off at its expiry without contacting the control plane, so
the fleet drains while this scenario is down.

### `enroll`, planned `EnrollService`

Owns `OT-P0-005` and `OT-P1-001`. Delegation to `vrooli-bridge`, and
nothing more.

| Planned method | Purpose | Notes |
|---|---|---|
| `GetEnrollmentState` | Whether an instance became a trusted node | An un-enrolled instance stays visible and flagged |
| `RetryEnrollment` | Re-run a queued enrollment | Bridge unavailability degrades rather than refuses |
| `AdoptHost` | Enroll a host the operator already owns | Records no instance, no intent and no reservation, so it is never metered |

This domain contains no SSH implementation and never will. It renders a
first-boot configuration carrying the bridge's onboarding public key,
creates the bridge Machine record with the instance address as a
locator, and starts bridge onboarding.

**Upstream prerequisite.** Unattended enrollment cannot be built until
`vrooli-bridge` publishes its onboarding public key. Bridge can read the
key internally today but exposes it nowhere. That is the single new wire
contract this scenario needs from another scenario, and it is not
Compute Manager's to add.

### Endpoints that will never exist

| Never | Why |
|---|---|
| `PauseInstance`, `StopInstance`, `SuspendInstance`, `HaltInstance`, `ShutdownInstance` | `OT-P0-007`. A stopped instance still bills at the full rate on five of the seven providers surveyed, so a pause control costs full price for zero value. Destroy is the only stop, and a structural test asserts no such method, handler or domain function exists anywhere in the scenario |
| Any wallet, plan, entitlement or invoice endpoint | Owned by `landing-page-business-suite` |
| Any node identity, pairing, scope or dispatch endpoint | Owned by `vrooli-bridge` |
| Any hostname, DNS or ingress endpoint | Owned by `tunnel-manager` |
| Any endpoint that deploys a scenario onto a machine | Owned by `scenario-to-cloud` and `deployment-manager` |
| Any second `Machine` object | Bridge owns Machine. This scenario owns Instance, and one bridge Machine may point at the instance backing it |

## Adding a new endpoint

For a new domain, take the worked vertical slice from
`templates/scenarios/react-vite/`, because the copy this scenario shipped
with was removed by `template-manager detemplate compute-manager`.

For an endpoint inside an existing domain:

1. Add or extend the `.proto` messages and service in
   `packages/proto/schemas/compute-manager/v1/<domain>/`, then run
   `make generate`.
2. Implement the generated handler method in
   `api/handlers/<domain>/connect_handler.go`, and keep it thin.
3. Update endpoint metadata in `api/handlers/<domain>/module.go`, and add
   the domain's two lines to `api/internal/modules/registry.go`.
4. If the endpoint has a CLI mirror, bind it in `cli/manifest.json`, or
   list it in `omitted[]` with a reason. That manifest is the single
   source of truth for the CLI surface.
5. Run `make endpoints`. Do not edit
   [`.vrooli/endpoints.json`](../../.vrooli/endpoints.json) by hand.
6. Move the endpoint's row out of Part 2 and into Part 1 of this
   document in the same change, and add tests for the touched layers.
7. Add a row to [`../internal/SEAMS.md`](../internal/SEAMS.md) if you
   introduced a new interface that production wires once and tests
   substitute.

The CI gate enforces endpoint-manifest freshness and the API to CLI
mapping contract, so every Connect endpoint is either bound or omitted
in `cli/manifest.json`.

## Cross-references

- [`cli-commands.md`](cli-commands.md): CLI commands that mirror these endpoints
- [`configuration.md`](configuration.md): environment variables, credentials and interval settings
- [`../concepts/DOMAINS.md`](../concepts/DOMAINS.md): the seven domains the planned services map to
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md): what each dependency does when it is unavailable
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md): proto as the canonical contract
- [`../internal/SEAMS.md`](../internal/SEAMS.md): handler, service and repository seams
- [`../internal/TESTING.md`](../internal/TESTING.md): endpoint test patterns
- [`../operations/RUNBOOK.md`](../operations/RUNBOOK.md): operator procedures the planned endpoints support
