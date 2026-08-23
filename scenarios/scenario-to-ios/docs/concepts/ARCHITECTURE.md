# Architecture — Scenario to iOS

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

A scenario is one product expressed through three coordinated surfaces
and one canonical contract layer.

```
                       ┌─────────────────────────────┐
                       │  Generated proto types      │
                       │  packages/proto/schemas/    │
                       │   scenario-to-ios/v1/...    │
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
| Contracts (`packages/proto/schemas/scenario-to-ios/`) | Wire shape | Proto messages/services and generated clients | Hand-written route/type mirrors |

The load-bearing principle: the API is the only surface that contains
business logic. UI and CLI translate user/operator intent into API
calls. Proto types flow from one source of truth so wire-shape drift
between surfaces is impossible.

## System Boundaries

The scenario owns:

- source code under `api/`, `ui/`, and `cli/`,
- generated-scenario docs under `docs/`,
- scenario lifecycle metadata under `.vrooli/`,
- scenario-specific requirements under `requirements/`,
- scenario proto schemas relocated to
  `packages/proto/schemas/scenario-to-ios/`.

The scenario does not own:

- shared package implementation under `packages/`,
- Vrooli resource implementation,
- scenario dependencies it calls,
- generated proto outputs under `packages/proto/gen/`.

### Layer ownership

This ramp is one layer in a four-layer system, and most of what an iOS
delivery pipeline appears to need is owned elsewhere. Reaching across a
line below is a design error, not a shortcut.

| Question | Owner | This scenario's relationship |
|---|---|---|
| What devices and nodes do I control, and may I send this? | `vrooli-bridge` | Consumes reach and durable dispatch. Load-bearing here in a way it is not for other ramps. |
| What can be done *to* a device? | `device-control` | Consumes every device verb under a held lease, across `ios-simctl`, `ios-xcuitest`, and `ios-mirror`. Implements no device driving. |
| What should the app *do* in web content? | `browser-automation-studio` | Delegates web-content chapters to a named BAS flow. Authors no mobile-specific flow. |
| How is delivery validated and evidenced? | `packages/delivery-ramp-go` | Implements only `Prober`, `Builder`, `Driver`, `Distributor`. Never reaches into spine internals. |
| May a release ship, across all ramps? | `deployment-manager` | Emits reference-only `common.v1.TargetVerdict`. Does not own the consumer-side gate. |
| Where does the Apple toolchain come from? | Nobody — it is a host capability | Verifies and instructs. Never acquires, licenses, or versions Xcode. |
| Where does signing material live? | `secrets-manager` | References an identity. Holds no certificate, profile, or key. |

### The constraint that shapes this scenario's architecture

Every other ramp can build on the host it runs on. This one cannot: the
Apple toolchain exists only on macOS, so **remote execution is the normal
path rather than a fallback**. Two consequences are structural rather
than incidental:

1. *The build host is a role, not a machine.* Any qualifying node can fill
   it. The currently available node is Intel, and Xcode 27 dropped Intel
   entirely — so it satisfies today's App Store Connect floor but will
   eventually be able to validate without being able to produce
   submittable builds. Modelling the host as a role means replacing it is
   a node registration, not a rewrite. Validation capability and release
   capability are therefore tracked as separate states.
2. *A dispatch is not a pass.* Reaching a remote node proves reachability,
   not correctness. A cell becomes `pass` only on target-owned evidence
   returned from that node — never on successful dispatch alone.

Document dependency and resource decisions in
[`INTEGRATIONS.md`](INTEGRATIONS.md), not here.

## Contracts And Data Flow

Wire shapes do not live in TypeScript interfaces, Go structs, or
hand-written JSON schemas. They live in `.proto` files. For
proto-typed API calls, the `.proto` file also declares the service
block that generates Connect handlers and clients.

```
packages/proto/schemas/scenario-to-ios/v1/<domain>/<file>.proto
       │
       ▼
       make generate
       │
       ├──▶ packages/proto/gen/go/scenario-to-ios/v1/...              (api, cli)
       ├──▶ packages/proto/gen/go/scenario-to-ios/v1/...connect       (Connect-Go)
       ├──▶ packages/proto/gen/typescript/scenario-to-ios/v1/...   (ui)
       └──▶ packages/proto/gen/python/scenario_to_ios/v1/...    (future tools)
```

Use Connect-RPC by default:

- UI to API for proto-typed payloads,
- CLI to API for proto-typed payloads,
- API to API / inter-scenario calls with Vrooli-owned protos.

REST is allowed only for four enumerated reasons, defined as
`RESTReason` constants in `api/internal/module/module.go`:

| Reason | When it applies |
|---|---|
| `RESTReasonMultipartUpload` | Opaque file bytes via `multipart/form-data`. A binary/blob attachment-upload endpoint is the canonical case. |
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
proto-typed wherever possible. A multipart attachment-upload handler
should return a proto-typed metadata message (e.g.
`UploadAttachmentResponse`); only the request transport is multipart.
Drift between API/UI/CLI is eliminated as long as the wire payload type
is shared.

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

If shared infrastructure starts using product vocabulary, move that
piece back into the owning domain or split a new domain first.

## Extension Rules

Add product behavior by adding or updating the owning domain, not by
growing generic buckets.

For a normal proto-backed domain:

1. Add proto messages and service methods under
   `packages/proto/schemas/scenario-to-ios/v1/<domain>/`.
2. Add API domain code under `api/internal/<domain>/`.
3. Add transport code under `api/handlers/<domain>/`.
4. Register schemas/endpoints in `api/internal/modules/registry.go`
   and mount the module in `api/main.go`.
5. Add CLI commands under `cli/domains/<domain>/`.
6. Add UI API wrappers under `ui/src/api/<domain>.ts` and UI feature
   code under `ui/src/features/<domain>/`.
7. Update selectors, strings, endpoints, tests, and the docs contract
   in `docs/manifest.json`.

For detailed product ownership, update [`DOMAINS.md`](DOMAINS.md).
For persistence and retention, update [`DATA.md`](DATA.md). For
temporal behavior, update [`FLOWS.md`](FLOWS.md).

## Monetization Contract For Generated Apps

> **Status: recorded, not implemented.** No in-app-purchase code exists in this
> scenario. This section exists so the tier-2 desktop work does not foreclose
> tier 3, and so a future implementer starts from decisions rather than
> re-deriving them. The engineering contract is
> [`docs/concepts/PAID_FEATURES.md`](../../../../docs/concepts/PAID_FEATURES.md).

### What is the same as every other tier

A generated iOS app is a monetized Vrooli app first and a iOS app
second. These do not change:

- **One identity.** Sign-in resolves the shared LPBS consumer session. Signing
  in on iOS signs the user in to their desktop and local installs too,
  because the durable credential is one logical identity, not a per-app account.
- **Entitlements are read from a signed lease.** Gate checks verify the lease
  locally and keep working without a connection until `not_after`.
- **Class A (cost-bearing) metering is unchanged.** LLM tokens, audio seconds,
  and hosted compute are charged on LPBS before the work returns. No store takes
  a cut of inference, and no receipt validation is involved in a Class A charge.
- **Class B (local-capacity) metering is unchanged.** The operation runs on the
  device, is checked optimistically against the lease, and syncs through a
  durable outbox.

### What differs on tier 3

| Concern | Tier 2 desktop | Tier 3 iOS |
|---|---|---|
| Purchase rail | Stripe through LPBS | App Store in-app purchase |
| Sign-in surface | Custom protocol deep link | `ASWebAuthenticationSession` |
| Token store | Electron `safeStorage` + credential authority | Keychain |
| Subscription ingest | Stripe webhook | App Store Server Notifications + receipt validation |

Only the purchase rail and the token store are genuinely new work. Everything
above the entitlement lease is shared.

### The decision that constrains LPBS

**Entitlements must be source-agnostic.** A subscription resolves identically
whether Stripe, Apple, or Google issued it. A user who subscribes on their phone
and then opens the desktop app must see their subscription.

This is not a tier-3 implementation detail — it is a schema requirement on LPBS
that must land before the first paying customer exists, because afterwards it
becomes a migration of live billing data. Two couplings in LPBS block it today:

1. `credit_transactions.stripe_event_id` is a Stripe-specific uniqueness key.
   Top-up idempotency needs a generic `(source, external_event_id)` pair.
2. Entitlement features resolve through `GetPlanByPriceID(stripe_price_id)`.
   A subscription with no Stripe price returns an **empty `features[]`**, so
   every feature gate would silently fail closed for store-sourced subscribers.

The fix is a `source` discriminator plus generic external identifiers. It is
tracked in the monetization foundation plan and is owned by LPBS, not here.

### Store policy

Store rules for digital-goods purchase and external purchase links move, and
have moved materially in recent years. Do not encode a policy claim in this
document. Confirm current App Review Guidelines rules at implementation time, and treat the
answer as an input to the purchase-rail design only — never to the entitlement
model, which stays source-agnostic regardless.

## Architecture Maturity

Generated scenarios start with a mature template shape and starter
reference domains. Replace this table as the scenario becomes real.

| Area | Maturity | Evidence | Remaining Drift |
|---|---|---|---|
| API | Reference-ready | Domain-owned vertical-slice stack, module registry, per-domain schema, documented seams. | Starter domains must be replaced with scenario-specific capabilities. |
| UI | Reference-ready | Feature folders, typed API clients, selector/i18n registries, modeltest helpers. | Real scenarios may need routing/state patterns once multiple screens exist. |
| CLI | Reference-ready | Domain command groups wrap API calls and render reports. | New domains must add commands intentionally; CLI should remain thin. |
| Docs | Contract-ready | Manifest v2 registers docs, maturity, stages, and validation hints. | Scenario-specific stubs must be filled or marked not-applicable. |

Use `docs/manifest.json` as the documentation contract. The declared
`maturity` values are expected to be maintained by agents and later
grounded by Knowledge Observatory validation.

## Intentional Deviations

Record deviations from the template or from Vrooli scenario standards
when they are deliberate and durable.

| Date | Deviation | Reason | Revisit Trigger |
|---|---|---|---|
| 2026-08-11 | None yet. | Generated from `react-vite`. | Update when the scenario intentionally diverges. |

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
