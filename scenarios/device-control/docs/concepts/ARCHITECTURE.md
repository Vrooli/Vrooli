# Architecture — Device Control

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

## Where This Scenario Sits

Device Control owns exactly one layer of a four-layer stack. Each layer
answers one question, and none answers two. The boundaries are the
reason most of this scenario's dependencies exist; the rationale is
`D-008` in [`../internal/DECISIONS.md`](../internal/DECISIONS.md).

| Layer | Question it answers | Owner |
|---|---|---|
| Delivery ramps | "Is this artifact shippable for this platform?" | `scenario-to-ios`, `scenario-to-android`, `scenario-to-desktop` |
| Web-content automation | "What should the app's UI do?" | `browser-automation-studio` |
| **Device operation** | **"What can I do to this device, and did it work?"** | **this scenario** |
| Reach and trust | "What do I control, and may I address it?" | `vrooli-bridge` |

Two consequences shape every design choice below. This scenario never
learns what an artifact or a release is — a ramp's `Driver` translates
ramp intent into device verbs, one way only
([`../internal/SEAMS.md`](../internal/SEAMS.md#strategy-is-not-a-ramp-driver)).
And it never owns device identity — a phone is a bridge *attached
device*, reachable only through a host node, so a second registry here
would split the single answer to "what do I control."

## Scenario Shape

A scenario is one product expressed through three coordinated surfaces
and one canonical contract layer. Device Control adds one layer the
plain template does not have: a strategy-adapter tier beneath the API,
which is how a verb reaches a physical or virtual device.

```
                       ┌─────────────────────────────┐
                       │  Generated proto types      │
                       │  packages/proto/schemas/    │
                       │   device-control/v1/...    │
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
                        ┌────────────┴────────────┐
                        ▼                         ▼
                  ┌──────────┐        ┌───────────────────────┐
                  │  SQLite  │        │  Strategy adapters    │
                  │ (local)  │        │  android-adb,         │
                  └──────────┘        │  ios-simctl,          │
                                      │  ios-xcuitest,        │
                                      │  ios-mirror,          │
                                      │  host-desktop         │
                                      └───────────┬───────────┘
                                                  │ device verbs
                                                  ▼
                                      ┌───────────────────────┐
                                      │    vrooli-bridge      │
                                      │  reach · trust · audit│
                                      └───────────┬───────────┘
                                                  ▼
                                        host node ─▶ attached device
```

| Surface | Role | Owns | Does Not Own |
|---|---|---|---|
| API (`api/`) | Scenario core | Business rules, persistence, integrations, transport edge | Browser state, CLI formatting |
| UI (`ui/`) | Browser presentation | Components, i18n, accessibility, browser interaction | Business rules, persistence policy |
| CLI (`cli/`) | Operator/agent wrapper | Argument parsing, output formatting, API invocation | Business rules, duplicated validation |
| Strategies (`api/internal/strategies/`) | Control mechanism | One transport's floor operations and declared capabilities | Flow semantics, leases, evidence policy, ramp vocabulary |
| Contracts (`packages/proto/schemas/device-control/`) | Wire shape | Proto messages/services and generated clients | Hand-written route/type mirrors |

The load-bearing principle: the API is the only surface that contains
business logic. UI and CLI translate user/operator intent into API
calls. Proto types flow from one source of truth so wire-shape drift
between surfaces is impossible.

The CLI carries one extra obligation here. It is the agent-facing
control surface, so a capability reachable only from the API or UI is
invisible to agent mode and treated as incomplete (`D-007`). CLI parity
is a functional requirement, not ergonomics.

Two invariants hold across the whole shape and are enforced at the
API/service layer, never in a surface:

- **Capabilities are probed, never inferred** from device kind
  (`D-002`). The registry in
  [`../reference/capabilities.md`](../reference/capabilities.md) records
  expectations; only a probe decides.
- **No verb reaches a strategy without a bridge-authorized reach and a
  held, unexpired lease** (`D-006`). Several strategies are physically
  single-session, so a missing lease corrupts evidence rather than
  erroring.

## System Boundaries

The scenario owns:

- source code under `api/`, `ui/`, and `cli/`,
- generated-scenario docs under `docs/`,
- scenario lifecycle metadata under `.vrooli/`,
- scenario-specific requirements under `requirements/`,
- scenario proto schemas relocated to
  `packages/proto/schemas/device-control/`.

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
packages/proto/schemas/device-control/v1/<domain>/<file>.proto
       │
       ▼
       make generate
       │
       ├──▶ packages/proto/gen/go/device-control/v1/...              (api, cli)
       ├──▶ packages/proto/gen/go/device-control/v1/...connect       (Connect-Go)
       ├──▶ packages/proto/gen/typescript/js/device-control/v1/...   (ui)
       └──▶ packages/proto/gen/python/device_control/v1/...    (future tools)
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
   `packages/proto/schemas/device-control/v1/<domain>/`.
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

### Adding a control strategy

Strategies are the scenario's primary extension point, and they follow
a different path from a domain. Adding one must require **no engine
change** — that property is what `DVC-P0-002`'s conformance suite
exists to protect.

1. Implement the three floor operations — `Observe`, `Actuate`,
   `Describe` — against the `Strategy` seam.
2. Declare optional capabilities using the canonical IDs in
   [`../reference/capabilities.md`](../reference/capabilities.md). Never
   invent an ID; an unlisted capability cannot be required by a step or
   reported by a probe.
3. Supply the probe that verifies each declared capability on a real
   target. Declaration and verification are deliberately two seams
   ([`../internal/SEAMS.md`](../internal/SEAMS.md#why-describe-and-capabilityprober-are-two-seams)).
4. Run `device-control strategy verify <id>` and resolve any
   declared-but-unprovable delta.

If a strategy needs a capability the registry does not list, the
registry is incomplete — add the capability *and* the construct that
exercises it in the same change (`D-012`). Full walkthrough:
[`../guides/adding-a-strategy.md`](../guides/adding-a-strategy.md).

## Architecture Maturity

| Area | Maturity | Evidence | Remaining Drift |
|---|---|---|---|
| Design | Authored | PRD operational targets, capability registry, decision log, and planned seam register are complete and mutually consistent. | Two design decisions remain open — lease enforcement point and the `ios-mirror` non-promotability mechanism. See [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md). |
| API | Template-only | Generated `react-vite` shape: module registry, per-domain schema, documented seams. | No scenario domain implemented yet. The six domains in [`DOMAINS.md`](DOMAINS.md) and the seams in [`../internal/SEAMS.md`](../internal/SEAMS.md) are planned, not built. |
| UI | Template-only | Feature folders, typed API clients, selector/i18n registries. | Fleet view, live session view, flow authoring, and run review are unbuilt. |
| CLI | Template-only | Domain command groups wrap API calls and render reports. | CLI parity is a P0 functional requirement (`D-007`); no scenario command exists yet. |
| Docs | Contract-ready | Manifest v2 registers docs, maturity, stages, and validation hints. Scenario-specific docs authored 2026-08-10. | Requirement validation entries are typed `manual` while tests are unwritten; convert as implementation lands. |
| Strategies | Not started | Registry, floor, profiles, and expected matrix specified. | No adapter implemented. `android-adb` (`DVC-P0-011`) is the first and proves the floor, ladder, and conformance suite. |

Use `docs/manifest.json` as the documentation contract. The declared
`maturity` values are expected to be maintained by agents and later
grounded by Knowledge Observatory validation.

## Intentional Deviations

Record deviations from the template or from Vrooli scenario standards
when they are deliberate and durable.

| Date | Deviation | Reason | Revisit Trigger |
|---|---|---|---|
| 2026-08-10 | A strategy-adapter tier sits beneath the API, outside the template's surface model. | The template shape assumes the API's lowest layer is persistence. Here the API also drives external hardware through pluggable adapters, which is the scenario's primary extension point rather than an implementation detail of one domain. | A strategy stops being substitutable, or adapters collapse into a single domain. |
| 2026-08-10 | Device identity is read from another scenario rather than owned locally. | `vrooli-bridge` owns the fleet registry; this scenario mirrors device records and owns only probing and health. Normally a scenario owns its primary entity's identity. | Bridge changes its fleet model, or a device class fits neither side (`D-008`). |

## Documentation Architecture

Scenario docs follow the same ownership rule as code: one durable
question, one canonical home.

| Concern | Canonical Document |
|---|---|
| System map and extension rules | `docs/concepts/ARCHITECTURE.md` |
| Device capability registry and step requirements | `docs/reference/capabilities.md` |
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
- [`../reference/capabilities.md`](../reference/capabilities.md) — capability registry, step requirements, resolution rungs
- [`../internal/DECISIONS.md`](../internal/DECISIONS.md) — durable decisions behind this shape
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
