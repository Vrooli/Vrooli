# Integrations — Nutrition Planner

This document is the canonical dependency contract for resources, other scenarios, and
third-party services used by the scenario.

## Purpose Of This Document

Use this document to answer:

- What does the scenario depend on?
- Which dependencies are required versus optional?
- Which domain uses each dependency?
- What is the failure or degradation behavior?
- Where is the dependency declared or configured?

The short list below is a design outcome, not an early-stage accident. Nutrition Planner
computes nutrition, eligibility, cost, and planning deterministically on local data. Manual
entry and deterministic planning are sufficient to deliver every P0 workflow, so no required
P0 dependency is served by an external system.

## Dependency Inventory

| Dependency | Type | Required? | Used By | Contract | Failure Behavior |
|---|---|---|---|---|---|
| SQLite | embedded storage | yes (P0) | Every persistence-backed domain | Resolved in-process by `api-core/storage` from the scenario id; schemas applied by `api-core/database` | API reports unhealthy if the database is unreachable. |
| Vrooli lifecycle | local platform | yes (P0) | API, UI, CLI | `.vrooli/service.json`, Makefile targets, `vrooli scenario start` | The scenario must be started through lifecycle commands; a direct binary start bypasses ports and health checks. |
| `ai-gateway` | Vrooli resource | no (P1) | provider-and-job-adapters | Optional model-inference adapter behind a provider interface | No model configured is a supported state. Capture, editing, planning, and export continue to work manually; extraction is unavailable, not failed. |
| `notification-hub` | Vrooli resource | no (P1) | provider-and-job-adapters | Optional delivery channel for opt-in scheduled notifications | Notifications are simply not delivered; the in-app badge remains. Scheduling never depends on delivery. |
| `document-manager` | Vrooli resource | no (P1) | transfer-and-documents | Optional custody for source attachments and exported bundles | Attachments fall back to local blob storage; no core workflow is blocked. |

## Vrooli Resources

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| None required | not-applicable (P0–R2) | The application is one private workspace running against in-process SQLite. No P0 target requires a shared resource, and SQLite's single-writer model matches a personal, mostly append-only dataset. | A second concurrent writer appears, the dataset outgrows local storage, or a commercial multi-tenant deployment is selected. |
| `ai-gateway` | optional (P1) | Model inference for the capture → extract → validate → compare → review → apply pipeline. The planner itself never requires a model. | OT-P1-001 is implemented and the user configures a provider. |
| `notification-hub` | optional (P1) | Opt-in delivery for scheduled draft and quiet preparation notifications. | OT-P1-005 is implemented and the user enables an external channel. |
| `document-manager` | optional (P1) | Delegated custody for attachment bytes if local blob storage is insufficient for backups or sharing. | A complete backup must carry attachments the local BlobStore cannot durably hold. |

A configured model provider is optional at P1 and routes through `ai-gateway` and the
platform's secret store. A secret store is required before any credentialed external
adapter, and provider credentials never live in scenario config, the SQLite database, or an
export.

## Scenario Dependencies

| Scenario | Status | Reason | Contract |
|---|---|---|---|
| None required | not-applicable (P0) | Every P0 workflow — setup, capture, eligibility, planning, groceries, inventory, intake, portability — runs locally. | — |
| `ai-gateway` | optional (P1) | Model inference for assisted capture, explanation, and research proposals. | Typed proposal contract; model output is never a domain write. |
| `notification-hub` | optional (P1) | Delivery of opt-in notifications in the profile's IANA timezone. | Deduplicated schedule events by workspace, recurrence, and intended local period. |
| `document-manager` | optional (P1) | Attachment custody. | Opaque bytes behind a BlobStore seam; metadata stays with the owning domain. |

No scenario dependency is load-bearing. If every optional scenario is unreachable, the
product remains fully usable through manual entry and deterministic planning.

## Third-Party Services

No third-party service is required at any release through R2. The entries below are
**interfaces, not promised vendors**: an adapter is implemented only after its access,
terms, coverage, and maintenance cost are evaluated.

| Service | Status | Reason | Contract |
|---|---|---|---|
| USDA FoodData Central | optional (P1/R2) | A reasonable first nutrition-data provider; its data documentation distinguishes analytically based and branded-label data types. | Search/detail adapter returning pinned local source revisions, source identity, version, observed date, nutrient mapping, caching, and error classification. Credentials stay server-side; limits are respected. |
| Price sources (retailer APIs, store sites) | optional (P2) | Returns structured observations, never one universal price per ingredient. | Price adapter preserving package, amount/unit, retailer, conditions, currency, date, and source. A store with no integration still supports manual package prices. |
| Receipt / order-history sources | optional (P2) | Reduces manual capture for opt-in users only. | Opt-in, source-specific scope; stages purchase proposals, matches packages/units, and dedupes by source transaction identity plus normalized line content. Never turns old receipts into current stock. |
| Recipe URLs and readable pages | optional (P2) | Assists capture from user-supplied links. | Controlled server-side fetch (see below). The URL is stored even when fetching is unavailable. |
| Amazon, Costco, Walmart, Target, delivery services | potential only | Named retailers are potential sources, not promised supported integrations. | No adapter promises coverage. No integration in R0–R2 performs checkout or places an order. |

### Controlled URL And Attachment Ingestion

Any user-supplied URL fetch goes through a controlled server adapter. It restricts supported
schemes, rejects local/private/link-local destinations, validates resolved addresses and
every redirect, bounds response size/redirects/time, and isolates outbound credentials. This
addresses the class of issues described by the [OWASP SSRF Prevention Cheat
Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Server_Side_Request_Forgery_Prevention_Cheat_Sheet.html).

Fetched content and document text are untrusted data. A page that says "ignore previous
instructions" has no authority; scripts, hidden instructions, embedded forms, and external
links never become domain actions. Imported rich text renders through a safe allowed subset
or plain text. Attachments enforce MIME and size limits and never execute macros. Source
attribution and content-use metadata are retained; arbitrary-site scraping and unrestricted
redistribution are not promised.

## Failure Modes

| Dependency | Failure Signal | Expected Behavior | Tests |
|---|---|---|---|
| SQLite | database unreachable or `PingContext` error | `/health` reports an unhealthy dependency status; the UI shows a load error rather than a fake empty workspace. | health handler tests; boot/health E2E |
| Model provider (`ai-gateway`) | unavailable, over budget, or returns malformed output | The extraction or research job fails or waits; manual editing, planning, groceries, and export continue. A malformed field is rejected without losing the original source. A model proposal never authors a rule, price, target, or stock change. | provider-adapter tests with recorded responses; job budget tests; "providers disabled" suite |
| `notification-hub` | unavailable or delivery rejected | The scheduled run still produces its reviewable draft; the notification is dropped, not retried into duplicate reminders. | scheduling tests with a faked notification seam |
| `document-manager` | unavailable | Attachment fetches fail with a named reason; local blob fallback remains available where configured. | transfer tests with a faked attachment seam |
| Nutrition provider (e.g. FoodData Central) | HTTP error, auth failure, timeout, or ambiguous match | The lookup reports unavailable and queues the unresolved item for manual resolution. Unknown mappings are never silently assigned to a vaguely similar food; a missing nutrient stays unknown. | provider contract tests against recorded permitted responses; manual fallback tests |
| Price / receipt source | unavailable or malformed | A missing price remains unpriced; ranking may use a documented internal fallback that is never displayed as an observed price. Duplicate receipts cannot double-apply. | cost tests with faked unavailable sources; receipt dedup tests |
| Controlled URL fetch | blocked destination, oversized response, or fetch failure | The URL is preserved, fetch status is shown separately from whether the draft saved, and the user can enter the recipe manually. | SSRF and attachment-limit tests |

### Why There Are No P0 External Dependencies

The product's core promise is a deterministic, explainable plan built from the user's own
data. That data — diet rules, kitchen capabilities, recipes, prices, and stock — can be
entered manually in small increments, and the calculation must work offline from any
provider. Making a provider required would violate three specified properties at once:

1. **Determinism and reproducibility.** The planning engine must produce the same draft
   from the same inputs, seed, and algorithm version. A live external call is neither
   deterministic nor reproducible (specification §14.1, ACT-045).
2. **Honest unknowns.** A missing provider value must stay unknown, not become a fabricated
   fallback. Requiring a provider invites treating absence as data (DOM-07 #1).
3. **Low maintenance and no invented facts.** The user will not maintain integrations or
   research every food. Manual entry plus starter content is sufficient, and provider
   adapters are an enhancement, never a gate.

Consequently every external integration is optional, sits behind an interface, and degrades
to manual fallback. Third-party packages themselves flow only through Scenario Dependency
Analyzer per the repository dependency rule; this document records intent, not an
installation.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries
- [`DOMAINS.md`](DOMAINS.md) — which domain uses each dependency
- [`DATA.md`](DATA.md) — storage ownership and privacy
- [`FLOWS.md`](FLOWS.md) — job and import lifecycles
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../reference/product-specification.md`](../reference/product-specification.md) — §16, §19
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
