# Architecture — Compute Manager

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

> **Status: partially implemented.** The scenario was generated from the
> `react-vite` template on 2026-09-03. The provisioning, metering,
> reconciliation, expiry, enrollment, CLI and inventory slices now exist;
> provider-live proof and several post-launch controls remain open.

Compute Manager acquires, tracks and retires remote compute, and knows what it
costs. It owns **capacity and cost, and nothing else.**

The scenario is the standard surface shape plus two unattended loops:

- **API** (`api/`) - Go with Connect-RPC over proto-owned wire contracts. The
  only place business logic lives.
- **CLI** (`cli/`) - full headless parity, with every verb manifest-declared so
  it is dispatchable and governable through `vrooli-bridge`. A verb that exists
  only as Go code cannot be relayed.
- **UI** (`ui/`) - React with Vite and TypeScript. An inventory-first operator
  surface whose job is to make cost and expiry impossible to miss.
- **Storage** - one scenario-owned SQLite database. No external service, no
  cache, no queue.
- **Reconciler and expiry sweeper** - two scheduled loops that run with no
  operator present. Both are idempotent, resumable and bounded.

There is no local resource dependency of any kind. A scenario that provisions
infrastructure looks like it should need infrastructure; it does not. The
machines it creates live at a provider, not here.


## System Boundaries

The boundary is the whole design. Four neighbours already own everything
adjacent to capacity, and this scenario delegates to all four in a single
provisioning flow rather than absorbing any of them.

| Concern | Owner | This scenario's part |
|---|---|---|
| Node identity, pairing, scopes, dispatch | `vrooli-bridge` | Creates the Machine record and starts onboarding |
| SSH, first touch, bootstrap | `vrooli-bridge` | **None.** Contains no SSH code and never will |
| Credit, entitlements, wallets, invoices | `landing-page-business-suite` | Reserves and settles; stores only reservation identifiers |
| Agent spend limits | `treasury` | Consulted on the agent-initiated path only |
| Public hostnames, DNS, ingress | `tunnel-manager` | None |
| Deploying scenarios onto a machine | `scenario-to-cloud`, `deployment-manager` | None. Delivers a node, not a workload |
| The sellable definition | `offer-desk` | Declares the meter; publishes an offer |

Two boundary rules are load-bearing:

**The object is an Instance, not a Machine.** Bridge already owns a Machine,
meaning durable operator intent for a node. A second Machine in the fleet would
make every cross-scenario conversation ambiguous. The two are joined by pointer,
never by name.

**Bridge trust is never copied here.** This database stores a bridge machine
identifier and nothing else. No node key, no pairing state, no scope list.


## Contracts And Data Flow

Wire contracts live in proto. The service block in the proto is the transport
contract, and generated Connect handlers and clients are the boundary. REST
remains intentional only where a payload is genuinely not proto-typed.

The ordering rules below are the substance of this scenario. Most of its failure
modes are ordering mistakes rather than logic mistakes.

**Record intent, then reserve credit, then call the provider.** Writing the
intent first means a crash during reservation leaves a recoverable record
rather than an invisible hold. A refusal leaves a cheap `refused` intent, while
a create that succeeds with a lost response remains recoverable. This is the
invariant the reconciler depends on.

**Enrollment comes after running and is allowed to fail.** The instance is real,
metered and expiring whether or not bridge is reachable.

**Usage derives from transitions this scenario caused**, never from a loop that
observes what is running. A dead observer stops billing while the provider keeps
charging.

**The reconciler reports and never destroys.** Findings are rows an operator
acts on. Marking precedes sweeping, so a reconciler defect cannot destroy a
running node. For a local record whose provider instance is already gone, the
reconciler may invoke the meter's explicit settlement callback to close the
known usage window; it never calls the provider destroy operation.

The provider boundary is four methods: create, describe, list, destroy. There is
no stop, because a stopped instance still bills at the full rate on most
providers. Each adapter also declares its provider's billing facts as data
rather than hiding them.

See [`FLOWS.md`](FLOWS.md) for the sequences and state machines, and
[`DATA.md`](DATA.md) for what each step writes.


## Shared Infrastructure

Shared infrastructure is allowed only when the code is
business-vocabulary-free and used by unrelated domains or surfaces.

| Package/Folder | Purpose | Why Not Domain-Owned | Consumers |
|---|---|---|---|
| `api/internal/server/` | Compose modules and middleware into one HTTP server. | Server lifecycle is not a product capability. | API entrypoint and handler modules. |
| `api/internal/module/` | Shared module and endpoint descriptor types. | Domain modules return this common shape. | Handler packages, server, endpoint codegen. |
| `api/internal/modules/` | Thin registry for schemas and endpoints. | Boot/codegen need central lists; logic stays domain-owned. | `main.go`, `gen-endpoints`. |
| `api/internal/database/` | System schema and DB reachability seam. | Cross-cutting DB infrastructure, not one domain's data. | API boot, health. |
| `api/internal/middleware/` | Request logging, recovery and correlation. | Cross-cutting HTTP concerns, not one domain's behaviour. | Server composition. |
| `api/internal/httpc/` | Outbound HTTP client construction with bounded timeouts. | Every outbound seam needs the same budget discipline. | Provider, metering and bridge clients. |
| `api/internal/httpx/` | Inbound request decoding and error rendering helpers. | Shared across handlers; carries no product vocabulary. | Handler packages. |
| `api/internal/capabilities/` | Capability descriptor registry for the describe endpoint. | Platform-level introspection, not a product capability. | Capabilities handler. |
| `api/internal/testutil/` | Cross-domain test harnesses and fakes. | Used by unrelated domains; domain fakes stay domain-local. | API tests. |
| `ui/src/test-utils/` | Cross-feature render helpers, a11y helpers, and model tests. | Used by unrelated UI features. | UI tests. |

A deterministic clock seam is specified in `../internal/SEAMS.md` and does
not exist yet. The expiry sweeper and the reconciler schedule both depend
on it, so it lands with the first of those.

If shared infrastructure starts using product vocabulary, move that
piece back into the owning domain or split a new domain first.

## Extension Rules

Add product behavior by adding or updating the owning domain, not by
growing generic buckets.

For a normal proto-backed domain:

1. Add proto messages and service methods under
   `packages/proto/schemas/compute-manager/v1/<domain>/`.
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

Honest reading: the contract is authored and nothing is implemented.

| Area | Maturity | Evidence | Remaining Drift |
|---|---|---|---|
| Product contract | In progress | PRD with 15 operational targets; requirements validation reports zero structural findings | P0/P1 claims remain planned until their full evidence set and live proof are complete |
| Domain design | In progress | Provider, intent, instance, meter, reconcile, expiry, enroll and provision packages exist | Post-launch ceiling and daily cost domains remain unimplemented |
| API | Partial implementation | Proto-backed lifecycle, adoption, reconciliation, metering and health endpoints are registered | Provider-live proof and remaining post-launch endpoints are open |
| CLI | Partial implementation | Manifest-declared instance, intent, meter and reconcile commands are wired | Full operator workflow evidence remains open |
| UI | Partial implementation | Inventory, findings and instance-detail routes render with component and accessibility tests | Browser health evidence and remaining visual findings are open |
| Docs | In progress | Operational and contract docs are being aligned with implemented behavior | Several generated design-era statements still require cleanup |
| Storage | Implemented slice | Scenario-owned SQLite schema is loaded through the domain schema provider | Retention/pruning and production migration evidence remain open |

The remaining implementation slices should extend the fake-provider spine with
the post-launch controls and then prove the provider adapter against a real
credentialed environment. The expensive failure modes are reachable without a
real API key; provider billing and enrollment behavior are not.


## Intentional Deviations

| Date | Deviation | Reason | Revisit Trigger |
|---|---|---|---|
| 2026-09-03 | The provider interface is four methods, not the ten-method shape used by the existing local-virtual-machine provider seam elsewhere in the fleet. | Stop, snapshot and reset were shaped by local virtual machines where stopping is free. On most cloud providers a stopped instance still bills at the full rate. | A provider whose stopped instances genuinely cost nothing, and a workload that needs suspension rather than destruction. |
| 2026-09-03 | Requirement validations name the test path they will occupy and stay `not_implemented` until that test exists. | A validation marked `implemented` against a file that does not exist is the fabricated claim the validator rejects. | Each entry flips as its test lands. |
| 2026-09-03 | Every experience page is `draft` and every claim is tier `aspirational`. | Machine-tier claims gate CI and require stable selectors. There is no UI to select against. | Claims promote to machine tier as the real surface is built. |
| 2026-09-03 | The template example region pin was raised from `experience-surface@1.0.0` to `1.0.3` before the example was removed. | The template pins a version that no longer resolves, so a generated scenario fails experience validation out of the box. | Template fixes its pin. |


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
