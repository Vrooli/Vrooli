# Domains — Content Desk

This document is the canonical map of product capabilities, bounded
contexts, and ownership for this scenario. Keep it current whenever a
domain is added, renamed, split, merged, or removed.

The scaffold also ships one clearly fenced worked example domain (never
product scope) as a copyable reference; `template-manager detemplate
<scenario>` removes every fenced example once the real domains are green.

**Status: designed, not implemented.** The domain map below is the output of
a design conversation recorded in [`../internal/DECISIONS.md`](../internal/DECISIONS.md).
Only `health` exists in code today. Each row states the boundary the
implementation is expected to honour.

The load-bearing split is between **what is true** and **what is allowed**.
`claims` answers whether an assertion is true and owns the verification gate;
`review` answers whether a draft is permitted to ship and owns craft and
policy scoring. Conflating them is the most likely way this scenario decays:
a truth gate that keeps failing on unfalsifiable statements gets switched off,
and a policy checker that pretends to verify facts gives false assurance.

## Purpose Of This Document

Use this document to answer:

- What product capabilities does this scenario expose?
- Which domain owns each concept, table, proto, endpoint, UI feature,
  CLI command, and test surface?
- Which concepts are shared, deferred, or deliberately not domains?

System-level architecture belongs in [`ARCHITECTURE.md`](ARCHITECTURE.md).
Workflow details belong in [`FLOWS.md`](FLOWS.md). Storage details
belong in [`DATA.md`](DATA.md).

## Domain Inventory

| Domain | Responsibility | Purpose | Owns Data | Primary Archetype | Secondary Traits | Glossary | Source Paths |
|---|---|---|---|---|---|---|---|
| health | Report runtime readiness and dependency reachability. | Expose API/database readiness and show the UI can read live backend state. | No product data. | reporting | query | HealthHandler | `api/handlers/health/`, `ui/src/features/health/`, `packages/proto/schemas/content-desk/v1/shared/health.proto` |
| campaigns | Own the campaign record, its evidence gate, and its artifact slot budget. | The work context a producer draws from; the budget is what keeps in-flight work and operator review load bounded. | Campaigns, slots, evidence refs. | crud | policy | Campaign, Slot, Hypothesis | `api/internal/campaigns/` |
| artifacts | Own the draft object and its lifecycle. | One place a draft lives from request to publication, with illegal transitions refused rather than discouraged. | Drafts, revisions, requests, approval attribution. | crud | workflow | Draft, Request, Revision | `api/internal/artifacts/` |
| claims | Own the claim library, evidence strength, and the verification gate. | Makes the honesty doctrine mechanical: a draft cannot be approved while it cites something unverified. | Claims, evidence, checks, citations. | policy | service | Claim, Evidence, Check, Citation | `api/internal/claims/` |
| posttypes | Own the post-type registry and the activation gate. | Turns the v0→v1 activation rule from prose into something executable. | Type definitions, activation state, failure-mode sets. | policy | reporting | PostType, Criterion, FailureMode | `api/internal/posttypes/` |
| review | Own review runs, per-mode verdicts, and challenge state. | Craft and policy scoring, including the AI-UGC guardrails, recorded as structured output rather than prose. | Review runs, verdicts, challenges, resolutions. | service | policy | ReviewRun, Verdict, Challenge | `api/internal/review/` |
| ledger | Own publish history, coverage, subject familiarity, and narration state. | The read surface later drafts depend on, and the first real input the marketing learning loop has ever had. | Publish records, coverage, mentions, narrated items. | reporting | query | PublishRecord, Coverage, Mention | `api/internal/ledger/` |

## Domain Details

### campaigns

- Purpose: hold the campaign record and bound the work it can generate.
- Primary archetype: CRUD with a policy gate.
- Owns: campaign fields (theme, audiences, channels, acquisition and retention
  hypotheses, linked SKUs, status); the evidence references that justify it;
  the artifact slot budget and slot occupancy.
- Does not own: the decision to launch a campaign. That stays an operator-reviewed
  `campaign-launch-proposal`; this domain stores the resulting record.
- Invariant: a campaign cannot leave `proposed` without at least one evidence
  reference. Campaign sprawl is a named marketing risk, and an evidence gate is
  the structural version of "prevent sprawl when signal is weak".
- Invariant: slots are a hard cap, not a target. A draft beyond budget is
  refused. This is the only mechanism that bounds operator review load, which
  becomes the binding constraint once agents are freer.
- Requirements: `CONTENTD-P0-001`, `CONTENTD-P0-002`.

### artifacts

- Purpose: own the draft from request through publication.
- Primary archetype: CRUD with a modelled workflow.
- Owns: draft body, hook, type, lane, audience, channel, revisions, the inbound
  request queue, and approval attribution.
- Does not own: whether the draft is true (`claims`) or permitted (`review`).
- Invariant: the lifecycle is declared in a flow contract and the transition
  table is generated from it. `drafted → approved` without verification,
  approval by a non-operator, and any transition out of `published` are all
  refused by the model rather than by a handler check.
- Invariant: approval is never automated. It is the last human check before a
  public claim.
- Requirements: `CONTENTD-P0-003`, `CONTENTD-P0-007`, `CONTENTD-P2-002`.

### claims

- Purpose: decide whether an assertion in a draft is true, and block approval
  when it is not.
- Primary archetype: policy.
- Owns: claim records, their kind, evidence (citation or re-runnable check),
  verification state, expiry, and the many-to-many citations from drafts.
- Does not own: policy compliance. Persona traits, disclosure, and credential
  rules are unfalsifiable-by-design or normative, and belong to `review`.
- Invariant: claims are a shared library, not draft annotations. One claim may
  be cited by many drafts; verify once, cite many. This is what later makes it
  possible to ask which published posts now carry a false statement.
- Invariant: implication claims never enter the truth gate. Routing them here
  would make the gate fail on statements that cannot be verified even in
  principle, and a gate that fails constantly gets disabled.
- Degradation rule: inference is not required. Author-declared claims are the
  P0 path; assisted extraction is a P1 cross-check that must never become the
  authority.
- Requirements: `CONTENTD-P0-004`, `CONTENTD-P0-005`, `CONTENTD-P0-006`,
  `CONTENTD-P1-001`, `CONTENTD-P1-003`, `CONTENTD-P1-007`.

### posttypes

- Purpose: hold the registry of post types and evaluate activation.
- Primary archetype: policy / reporting.
- Owns: type definitions (medium, paired-skill reference, required fields),
  activation criteria and their evaluated state, and the per-type failure-mode
  sets that `review` scores against.
- Does not own: the strategic canon for a post type. Purpose, audience, and
  conversion goal stay in operator-curated marketing docs; this domain holds the
  executable half.
- Invariant: an inactive type blocks approval. The activation rule already exists
  in canon as four checkable criteria that nothing can execute; making it
  runnable is the entire point of this domain.
- Requirements: `CONTENTD-P0-008`.

### review

- Purpose: decide whether a draft is permitted to ship.
- Primary archetype: service with a policy surface.
- Owns: review runs, per-failure-mode verdicts with evidence, challenge reports,
  resolution state, and staleness sweeps over pending decisions.
- Does not own: factual verification (`claims`). It does own the AI-UGC
  guardrails — credential claims by a persona, real-person likeness, fabricated
  customer testimonials, and missing disclosure — because those are normative
  rules, not truth conditions.
- Invariant: a run with any failed mode leaves the draft blocked. Review output
  is structured, not prose, so the contrarian's judgement lands somewhere a gate
  can read.
- Requirements: `CONTENTD-P0-009`, `CONTENTD-P2-004`.

### ledger

- Purpose: record what shipped and answer the questions later drafts depend on.
- Primary archetype: reporting / query.
- Owns: publish records with series linkage, coverage by campaign/lane/channel/SKU,
  subject familiarity per audience, and the narration log per subject.
- Does not own: publishing itself. Releasing to a platform belongs to the
  Channel Manager; this domain records the result it returns.
- Invariant: the ledger is append-oriented. A correction is a new record, so
  publish history stays auditable.
- Requirements: `CONTENTD-P0-010`, `CONTENTD-P0-011`, `CONTENTD-P0-012`,
  `CONTENTD-P0-013`, `CONTENTD-P1-002`, `CONTENTD-P1-004`, `CONTENTD-P2-001`.

### health

- Purpose: expose API/database readiness and show the UI can read live
  backend state.
- Primary archetype: reporting / query.
- Secondary traits: operational health.
- Owns: health response construction and dependency status mapping.
- Does not own: product data, business rules, or scenario-specific
  domain behavior.
- API: `api/handlers/health/`.
- CLI: built-in `status` command is provided through cli-core.
- UI: `ui/src/features/health/HealthCard.tsx`.
- Storage: none; probes configured database reachability.
- Requirements: starter scaffold health only.
- Tests: handler, module, UI feature, and accessibility tests.
- Related docs: [`../reference/api-endpoints.md`](../reference/api-endpoints.md).

## Shared Concepts

| Concept | Meaning | Owner |
|---|---|---|
| Domain | Product capability boundary that should be easy to find, test, and delete. | `DOMAINS.md` defines the map; code owns implementation. |
| Surface | API, UI, CLI, or contract layer exposing the same product capability. | `ARCHITECTURE.md`. |
| Seam | Test-substitutable boundary wired once in production. | `../internal/SEAMS.md`. |
| Requirement | Implementation-facing measurement tied back to the PRD. | `requirements/`. |

## Deferred Domains

Add future or intentionally deferred capabilities here only when they
are real enough to affect architecture or requirements.

| Candidate Domain | Why Deferred | Revisit Trigger |
|---|---|---|
| `performance` — engagement telemetry per published post | No social accounts exist and no measurement source is wired, so there is nothing to ingest and nothing against which to validate an ingestion path. | `CONTENTD-P2-001`: once accounts are live and producing measurable posts. |
| `variants` — per-channel derivation from one approved draft | Premature until the single-draft path has run end to end. Deriving variants before knowing what a verified draft looks like would fix the wrong shape. | `CONTENTD-P2-002`: once the P0 loop has published repeatedly. |

## Explicitly Not Domains (Decided)

| Considered | Verdict | Reason |
|---|---|---|
| `generation` — producing draft copy | **Rejected.** | Generation prompts live in the paired `x-<type>` skills. Absorbing them would break the doc-plus-skill discipline that keeps strategic reasoning and executable procedure separate, and would freeze prompt iteration behind a deploy. |
| `accounts` / `warming` — identity lifecycle and anti-shadowban cadence | **Rejected.** | Marketing canon already routes account handles, credentials, activation, warming, and cadence to Channel Manager. Warming state and the post queue are tightly coupled — the queue must refuse an account that is not warm — so splitting them would put a cross-scenario call on every publish. |
| `assets` — character, scene, and product definitions rendered to images and video | **Rejected.** | Structured rich-media data is operator-curated canon today and belongs to a future asset-production scenario that wraps existing image and browser-automation capabilities. Different clock, different failure mode: visual drift and generation spend, not a false claim. |
| `identity` / `credentials` | **Rejected.** | No credential ever touches this scenario's storage. Account records elsewhere hold a vault reference; the desk holds none. |

## Non-Domains

These are important but should not become product domains:

- `api/internal/server/` — HTTP composition substrate.
- `api/internal/module/` — shared module descriptor type.
- `api/internal/modules/` — thin registry for boot/codegen.
- `api/internal/database/` — cross-cutting database infrastructure.
- `api/internal/testutil/` — cross-domain test harnesses.
- `ui/src/components/` — shared presentation primitives.
- `ui/src/test-utils/` — cross-feature testing support.

If one of these starts using product vocabulary, split the product
piece into an owning domain instead of growing infrastructure.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system shape and extension rules
- [`FLOWS.md`](FLOWS.md) — workflows and state transitions
- [`DATA.md`](DATA.md) — data ownership and storage
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — dependency contracts
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — boundary registry
- [`../internal/TESTING.md`](../internal/TESTING.md) — test strategy
