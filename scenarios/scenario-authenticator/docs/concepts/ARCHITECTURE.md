# Architecture — Scenario Authenticator

This document is the scenario's system map. It explains the IdP role,
the invariant shape inherited from the `react-vite` template, and the
authenticator-specific layering, then points to the specialized
documents that own product domains, workflows, data, integrations,
deployment, operations, and business strategy.

> **Status: implemented foundation.** The scenario is detemplated and the
> account, token, session, JWKS, rate-limit, audit, realm, and Connect
> handler paths described below are live. Future capabilities are marked
> planned or deferred in the requirements registry. This document describes
> the current architecture and its remaining extension points.

Keep this file high-signal. Do not turn it into a warehouse for every
domain, endpoint, workflow, or decision. If a concern has a dedicated
document below, update that document and link it here.

## The IdP Role

scenario-authenticator is the fleet's foundational **Identity Provider
(IdP)** — an OIDC/Keycloak-style split where this scenario owns *who you
are* and every adopting scenario is a **Relying Party (RP)** that owns
*what you may do here*. The IdP issues and signs tokens; the RP verifies
them **locally and statelessly against the published JWKS** and never
calls back to the authenticator on the request hot path. This split is
the load-bearing architectural decision and is detailed in
[`../../PRD.md`](../../PRD.md) Appendix A.

Two rules follow directly and constrain everything else in this document:

- **API-to-API only — no cross-origin browser calls.** A browser talks
  only to its own scenario's API. If that scenario must reach the
  authenticator, its API forwards **same-origin** (the
  device-sync-hub pattern, see [`INTEGRATIONS.md`](INTEGRATIONS.md) and
  [`FLOWS.md`](FLOWS.md)). Consumers resolve the authenticator **by
  slug** through `api-core/discovery`; there is no hardcoded URL or port.
- **Stateless local verification is the scale lever.** Because RPs
  verify RS256 signatures against the cached JWKS, the authenticator is
  off the hot path. Token verification never touches this scenario's
  SQLite store, so its single-writer database is not a fleet-wide
  throughput ceiling.

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
                       │   scenario-authenticator/v1/...    │
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
                          ┌──────────┴──────────┐
                          ▼                     ▼
                   ┌──────────────┐      ┌──────────────┐
                   │ SQLite       │      │ Redis        │
                   │ (api-core/   │      │ hot state:   │
                   │  storage     │      │ sessions,    │
                   │  seam) +     │      │ revocation,  │
                   │ keypair PEMs │      │ CSRF, rate-  │
                   │ at rest      │      │ limit coord. │
                   └──────────────┘      └──────────────┘
```

Persistence uses **SQLite through the `api-core/storage` seam** — not
shared Postgres. Removing the shared database is the reason for the
rewrite: a fleet-wide shared DB created a fleet-wide blast radius. The
seam keeps the store swappable to a managed DB for cloud scale, so
SQLite is a default, not a lock-in. **Redis is retained** as required
hot/ephemeral state (sessions, revocation, OAuth CSRF, distributed
rate-limit coordination). Schema changes are **additive migrations
only — never database recreation**; only hashes and signed material are
stored at rest, never plaintext secrets. See [`DATA.md`](DATA.md).

| Surface | Role | Owns | Does Not Own |
|---|---|---|---|
| API (`api/`) | Scenario core | Business rules, persistence, integrations, transport edge | Browser state, CLI formatting |
| UI (`ui/`) | Browser presentation | Components, i18n, accessibility, browser interaction | Business rules, persistence policy |
| CLI (`cli/`) | Operator/agent wrapper | Argument parsing, output formatting, API invocation | Business rules, duplicated validation |
| Contracts (`packages/proto/schemas/scenario-authenticator/`) | Wire shape | Proto messages/services and generated clients | Hand-written route/type mirrors |

The load-bearing principle: the API is the only surface that contains
business logic. UI and CLI translate user/operator intent into API
calls. Proto types flow from one source of truth so wire-shape drift
between surfaces is impossible.

### Domain Layering Inside The API

Every product domain (`realms`, `identity`, `tokens`, `sessions`,
`authorization`, `audit`, `mfa`, `federation`, `apikeys` — see
[`DOMAINS.md`](DOMAINS.md)) is a vertical slice with the same four
layers, so a new domain plugs in the same way every time:

```
packages/proto/schemas/scenario-authenticator/v1/<domain>/   # wire contract (SSOT)
  └─▶ api/internal/<domain>/    # business logic: service, repository, validation,
      │                          #   schema.sql, seams (the only layer with rules)
      └─▶ api/handlers/<domain>/ # thin Connect handler: relay request → service →
          │                       #   proto response; map errors to Connect codes
          ├─▶ cli/domains/<domain>/        # CLI translation: argv → Connect call → render
          └─▶ ui/src/features/<domain>/    # UI translation: same-origin Connect call → screen
```

The handler layer is deliberately thin — it does no auth logic, only
relay and error translation (the device-sync-hub identity handler is the
reference shape). Shared crypto primitives (RS256/JWKS sign+verify,
Argon2id) live in `api/internal/crypto/` as a business-vocabulary-free
library used by `tokens` and `identity`; they are **not** a domain (see
[`DOMAINS.md`](DOMAINS.md) Non-Domains). New domains plug in via the
[Extension Rules](#extension-rules) below — no domain may grow a generic
bucket instead of its own slice.

## System Boundaries

The scenario owns:

- source code under `api/`, `ui/`, and `cli/`,
- generated-scenario docs under `docs/`,
- scenario lifecycle metadata under `.vrooli/`,
- scenario-specific requirements under `requirements/`,
- scenario proto schemas relocated to
  `packages/proto/schemas/scenario-authenticator/`.

The scenario does not own:

- shared package implementation under `packages/`,
- Vrooli resource implementation,
- scenario dependencies it calls,
- generated proto outputs under `packages/proto/gen/`.

Document dependency and resource decisions in
[`INTEGRATIONS.md`](INTEGRATIONS.md), not here.

## Contracts And Data Flow

Wire shapes do not live in TypeScript interfaces, Go structs, or
hand-written JSON schemas. They live in `.proto` files. For
proto-typed API calls, the `.proto` file also declares the service
block that generates Connect handlers and clients.

```
packages/proto/schemas/scenario-authenticator/v1/<domain>/<file>.proto
       │
       ▼
       make generate
       │
       ├──▶ packages/proto/gen/go/scenario-authenticator/v1/...              (api, cli)
       ├──▶ packages/proto/gen/go/scenario-authenticator/v1/...connect       (Connect-Go)
       ├──▶ packages/proto/gen/typescript/js/scenario-authenticator/v1/...   (ui)
       └──▶ packages/proto/gen/python/scenario_authenticator/v1/...    (future tools)
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

### The Authenticator's REST Edge

Connect-RPC is the **primary** contract: `Login`, `Register`, `Refresh`,
`Logout`, `Validate`, and session/realm/role/audit management are all
proto-owned Connect procedures with full CLI parity (OT-P0-010). The
only REST endpoints are non-RPC web standards whose shape this scenario
does not own — each carries a `RESTException` so it survives endpoint
validation:

| REST endpoint | Reason | Tier | Why it cannot be a Connect RPC |
|---|---|---|---|
| `GET /.well-known/jwks.json` | `RESTReasonThirdPartyShape` / `RESTReasonOpsProbe` | P0 | A web standard RPs (and `curl`) fetch with no generated client; the public-key document is the local-verify contract. |
| OAuth2/OIDC redirect callbacks | `RESTReasonThirdPartyShape` | P1 | The redirect/callback shape is dictated by external IdPs (Google, GitHub, Microsoft). |
| SAML ACS (Assertion Consumer Service) | `RESTReasonThirdPartyShape` | P2 | The SAML POST binding is an externally-defined enterprise contract. |

There is no "internal auth endpoint, REST is fine" path. Login itself is
a Connect RPC, not REST — the carried-over `/api/v1/sessions/{id}` revoke
that device-sync-hub calls today is preserved (or delivered as its
Connect equivalent) in lockstep so the live consumer never breaks (see
[`INTEGRATIONS.md`](INTEGRATIONS.md)).

### Realm aud-Scoping Is Enforced At Both Ends

The **realm** is the multi-tenant unit (PRD Appendix B). Even the single
default realm issues `aud`-scoped tokens, and verification rejects a
token whose `aud` does not match the verifying realm — so a token minted
for realm A is rejected by realm B with no cross-tenant acceptance, even
in a one-realm deployment. This is enforced in two places that must stay
in agreement: the `tokens` domain sets `aud` at **issuance**, and the
verifier (this scenario's `Validate` and every RP's local JWKS check)
rejects on `aud` mismatch at **verification**. A drift here is a
cross-tenant token leak, so it is covered by integration tests.

## Shared Infrastructure

Shared infrastructure is allowed only when the code is
business-vocabulary-free and used by unrelated domains or surfaces.

| Package/Folder | Purpose | Why Not Domain-Owned | Consumers |
|---|---|---|---|
| `api/internal/server/` | Compose modules and middleware into one HTTP server. | Server lifecycle is not a product capability. | API entrypoint and handler modules. |
| `api/internal/module/` | Shared module and endpoint descriptor types. | Domain modules return this common shape. | Handler packages, server, endpoint codegen. |
| `api/internal/modules/` | Thin registry for schemas and endpoints. | Boot/codegen need central lists; logic stays domain-owned. | `main.go`, `gen-endpoints`. |
| `api/internal/database/` | System schema and the `api-core/storage` seam (SQLite) + Redis hot-state client. | Cross-cutting persistence infrastructure, not one domain's data. | API boot, all persistence-backed domains, health. |
| `api/internal/crypto/` | RS256 sign/verify, JWKS construction, load-or-generate keypair, Argon2id hashing. | Shared crypto primitives carried over verbatim; a library, not a product capability. | `tokens`, `identity`. |
| `api/internal/clock/` | Deterministic time seam. | Time is cross-cutting and test-substitutable. | Middleware, repositories, token TTLs. |
| `api/internal/testutil/` | Cross-domain test harnesses and fakes. | Used by unrelated domains; domain fakes stay domain-local. | API tests. |
| `ui/src/test-utils/` | Cross-feature render helpers, a11y helpers, and model tests. | Used by unrelated UI features. | UI tests. |

If shared infrastructure starts using product vocabulary, move that
piece back into the owning domain or split a new domain first.

## Extension Rules

Add product behavior by adding or updating the owning domain, not by
growing generic buckets.

For a normal proto-backed domain:

1. Add proto messages and service methods under
   `packages/proto/schemas/scenario-authenticator/v1/<domain>/`.
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

## Architecture Maturity

The scenario carries the standard lifecycle shape and a live authentication
foundation. The table below distinguishes shipped paths from future product
work; it does not describe the removed `notes` example.

| Area | Maturity | Evidence | Remaining Drift |
|---|---|---|---|
| API | Implemented foundation | Account, token, session, JWKS, realm, rate-limit, audit, and handler seams in `api/`. | Authorization, machine binding, and delegated token extensions are delivered by this plan; MFA/federation/apikeys/multi-realm remain deferred. |
| UI | Implemented shell | Health dashboard, shared shell, settings, typed client/test infrastructure, selector/i18n registries. | Admin console, self-service, and hosted login/consent screens remain future UI work (see [`UI-ARCHITECTURE.md`](UI-ARCHITECTURE.md)). |
| CLI | Implemented foundation | Auth and session command groups use generated Connect clients. | Refresh, password-change, scope, and machine-link commands are delivered by this plan. |
| Docs | Maintained foundation | Concepts, internal, operations, business, and reference docs are present and being reconciled against code. | Deferred capability sections require updates when MFA/federation/multi-realm land. |

Use `docs/manifest.json` as the documentation contract. The declared
`maturity` values are expected to be maintained by agents and later
grounded by Knowledge Observatory validation.

## Intentional Deviations

Record deviations from the template or from Vrooli scenario standards
when they are deliberate and durable.

| Date | Deviation | Reason | Revisit Trigger |
|---|---|---|---|
| 2026-06-18 | Crypto core (RS256/JWKS/claims/Argon2id) is **ported verbatim**, not regenerated. | Re-deriving auth crypto risks breaking live consumers; the carried-over invariants are correct (PRD Appendix C). | Only if the JWKS/claims contract must change — then change RPs in lockstep. |
| 2026-06-18 | Redis is a **required** resource (not the template's SQLite-only default). | Session revocation correctness and distributed rate-limit accuracy depend on shared hot state. | If a deployment can prove correctness without Redis. |
| 2026-06-18 | A small REST edge is retained alongside Connect. | JWKS, OAuth callbacks, and SAML ACS are non-RPC web standards (see REST Edge table). | If a standard becomes expressible as a Connect procedure. |

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
