# Domains — Channel Manager

This document is the canonical map of product capabilities, bounded
contexts, and ownership for this scenario. Keep it current whenever a
domain is added, renamed, split, merged, or removed.

The scaffold also ships one clearly fenced worked example domain (never
product scope) as a copyable reference; `template-manager detemplate
<scenario>` removes every fenced example once the real domains are green.

**Status: designed, not implemented.** The domain map below is the output of a
design pass completed 2026-07-28. Only `health` exists in code today. Each row
states the boundary the implementation is expected to honour.

The load-bearing split is between **descriptors** (versioned JSON that declares
how a platform behaves and how an identity is warmed) and **state** (what an
identity has actually done and what happened). Descriptors are authored, reviewed,
and replayable; state is accumulated and never rewritten. If a change would let
runtime state edit a descriptor, it belongs in a proposal instead.

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
| health | Report runtime readiness and dependency reachability. | Expose API/database readiness and show the UI can read live backend state. | No product data. | reporting | query | HealthHandler | `api/handlers/health/`, `ui/src/features/health/`, `packages/proto/schemas/channel-manager/v1/shared/health.proto` |
| identities | Own the account roster, its environment record, credential references, and lane grants. | One row per account the fleet operates; everything else in the scenario is scoped to an identity. | Identities, environments, precondition satisfaction, lane grants. | crud | policy | Identity, Environment, PurposeTag, LaneGrant | `api/internal/identities/` |
| platforms | Own the declarative per-platform capability descriptors. | Storage shape, limits, cadence ceilings, and disclosure rules differ per platform; adding one should be a descriptor, not a build. | Platform descriptors and their seeded cache. | policy | reference | PlatformDescriptor, ActionKind, Ceiling | `api/internal/platforms/` |
| warming | Own warming programs, generated plans, phase progression, gates, and graduation. | An account earns distribution before it earns a lane. This domain is how that is declared, scheduled, and judged. | Programs, plans, rolls, phase state, gate evaluations, observations. | service | scheduler | Program, Phase, Gate, Graduation, Provenance | `api/internal/warming/` |
| queue | Own the unified action queue, sessions, cadence accounting, execution dispatch, and the release handoff. | Every platform action for an identity passes through here, because cadence is counted per account and not per workflow. | Queued actions, sessions, action records, release records. | workflow | service | Action, Session, Cadence, Release, Executor | `api/internal/queue/` |
| signals | Own observed distribution metrics, rolling baselines, and flags. | Decay is only detectable against a baseline, and it is always inferred rather than observed. | Observations, baselines, flags. | reporting | query | Observation, Baseline, Flag | `api/internal/signals/` |

<!-- EXAMPLE-DOMAIN:notes START -->
### Example domain — `notes` (removed by `template-manager detemplate`)

The template ships `notes` as a worked CRUD vertical slice with a binary
upload exception. Copy its shape for your own domains, then remove it.

| Domain | Responsibility | Purpose | Owns Data | Primary Archetype | Secondary Traits | Glossary | Source Paths |
|---|---|---|---|---|---|---|---|
| notes | Provide the worked CRUD reference with attachment upload exception. | Demonstrate the expected vertical slice for a real domain. | Notes and attachment metadata. | crud | service | Note, Attachment | `api/internal/notes/`, `api/handlers/notes/`, `cli/domains/notes/`, `ui/src/features/notes/`, `packages/proto/schemas/channel-manager/v1/notes/` |

- Purpose: demonstrate the expected vertical slice for a real domain.
- Primary archetype: CRUD / entity.
- Secondary traits: binary/blob attachment upload, upload workflow.
- Owns: note records, attachment metadata, note validation, note
  service/repository seams, UI note interactions, CLI notes commands.
- Does not own: product scope for a generated scenario.
- API: `api/internal/notes/`, `api/handlers/notes/`.
- CLI: `cli/domains/notes/`.
- UI: `ui/src/features/notes/`, `ui/src/api/notes.ts`.
- Storage: domain-owned SQLite schema in `api/internal/notes/schema.sql`.
- Requirements: template starter only; replace with PRD-specific
  requirements.
- Tests: repository, service, handler, CLI, UI, accessibility, and
  workflow tests.
- Related docs: [`FLOWS.md`](FLOWS.md), [`DATA.md`](DATA.md),
  [`../internal/SEAMS.md`](../internal/SEAMS.md).
<!-- EXAMPLE-DOMAIN:notes END -->

## Domain Details

### identities

- Purpose: hold one row per account the fleet operates, plus the environment that
  account depends on.
- Primary archetype: CRUD, with a policy edge for lane grants.
- Owns: platform id, handle, purpose tag (`brand` or `persona-actor`), optional
  persona reference into `asset-studio`, environment reference, vault credential
  reference, lane grants, lifecycle status.
- Does not own: the credential itself (`vault`), what a lane means
  (`content-desk`), or the persona's visual identity (`asset-studio`).
- Invariant: **no credential value is ever persisted.** An identity carries a vault
  path; the value is read at execution time and never stored, logged, or returned.
- Invariant: purpose tag is load-bearing, not descriptive. It selects the warming
  program, decides whether disclosure rules apply, and changes which flags matter.
- Environment note: device fingerprint, proxy region, and geo consistency are
  **recorded and checked, never provisioned.** The scenario cannot detect a leak in
  an environment it did not create, which is why the precondition gate is manual
  attestation rather than an automated probe (D-006).
- Requirements: `CHANMGR-P0-001`, `CHANMGR-P0-002`.

### platforms

- Purpose: describe what each platform allows so that no platform behaviour is
  hard-coded.
- Primary archetype: policy / reference data.
- Owns: supported formats and their limits, cadence ceilings per action kind,
  media constraints, disclosure requirements, and which executors exist for which
  actions.
- Does not own: channel strategy or account purpose — those are canon in
  `docs/marketing/strategy/CHANNELS.md` and are read-only input here.
- Invariant: the descriptor file is the source of truth and the table is a seeded
  cache. Reseeding replaces rows rather than duplicating them.
- Invariant: **the ceiling always wins.** A warming program that asks for more
  actions than the platform ceiling allows is clamped or rejected at plan
  generation, so a badly written program fails safe rather than at the platform.
- Requirements: `CHANMGR-P0-003`, `CHANMGR-P1-004`.

### warming

- Purpose: take an identity from newly created to eligible for a lane, and prove
  it earned the transition.
- Primary archetype: service with a scheduled pass.
- Owns: program descriptors, generated plans and their recorded rolls, phase
  progression, forbidden-action sets, gate evaluation, graduation criteria,
  maintenance policy, and the append-only observation log.
- Does not own: execution (`queue`) or measurement (`signals`). It declares what
  should happen and judges what did.
- Invariant: **graduation is earned, never elapsed.** Days passing does not qualify
  an identity; passing every criterion does.
- Invariant: **warming can fail terminally.** Quarantine is a real outcome, and an
  identity that keeps running a program it already failed is the expensive case
  this domain exists to prevent.
- Invariant: **graduation is not the end.** Maintenance engagement continues
  indefinitely and competes for the same cadence budget as posting.
- Honesty rule: every program carries provenance, and the programs shipped at P0
  are marked `speculative` because they are operator folklore rather than platform
  documentation (D-002).
- Requirements: `CHANMGR-P0-004`, `CHANMGR-P0-005`, `CHANMGR-P0-009`,
  `CHANMGR-P0-010`, `CHANMGR-P0-011`, `CHANMGR-P0-012`, `CHANMGR-P0-018`,
  `CHANMGR-P1-006`.

### queue

- Purpose: be the single path through which any platform action happens.
- Primary archetype: workflow.
- Owns: queued actions and their lifecycle, session grouping, cadence accounting,
  executor dispatch, action records with evidence, and the release handoff from
  `content-desk`.
- Does not own: what a warming program wants (`warming`) or whether a draft is
  publishable (`content-desk`).
- Invariant: **one queue per identity, across all action kinds.** Separate queues
  would let an identity breach its daily budget by warming and posting at once.
- Invariant: **the action record is identical regardless of executor.** Manual,
  browser, and API executions write the same fields, which is what makes
  automating later a swap rather than a rewrite (D-003).
- Invariant: release is idempotent by key. A retry returns the original result
  and never creates a second publish record — the contract `content-desk` is
  already written against.
- Requirements: `CHANMGR-P0-006`, `CHANMGR-P0-007`, `CHANMGR-P0-008`,
  `CHANMGR-P0-013`, `CHANMGR-P0-014`, `CHANMGR-P0-015`, `CHANMGR-P1-001`,
  `CHANMGR-P1-002`, `CHANMGR-P1-003`, `CHANMGR-P1-005`.

### signals

- Purpose: know what an identity's distribution normally looks like, so a change
  is detectable.
- Primary archetype: reporting / query.
- Owns: metric observations, rolling baselines, decay flags and their evidence.
- Does not own: any reaction to a flag. Pausing is the only effect, and it is
  mechanical.
- Invariant: **the domain reports measurements, never verdicts.** Platform
  penalties are unobservable; only reach, impressions, engagement, follower delta,
  and audience geography are real. No message asserts that an account has been
  penalized (D-004).
- Invariant: **flags pause; they never compensate.** Every plausible automatic
  response to suspected decay — posting more, posting less, changing niche — makes
  the situation worse or destroys the evidence.
- Baseline note: a baseline below the configured minimum observation count is
  reported as not-established rather than computed from thin data, which is why
  the first-post gate depends on baseline days.
- Requirements: `CHANMGR-P0-016`, `CHANMGR-P0-017`, `CHANMGR-P1-007`,
  `CHANMGR-P2-002`.

### health

- Purpose: expose API/database readiness and show the UI can read live
  backend state.
- Primary archetype: reporting / query.
- Secondary traits: operational health.
- Owns: health response construction and dependency status mapping.
- Does not own: product data, business rules, or scenario-specific
  domain behavior. In particular it does not own *account* health, which is
  `signals` — the two were deliberately named apart to prevent the collision.
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
| Identity | One account on one platform. The scoping unit for everything here. | `identities`. |
| Action | One thing done on a platform as an identity. The atom the queue schedules. | `queue`. |
| Lane | A content category an identity may carry (`oss`, `subscription`, `persona`). Defined by marketing canon; granted here. | `identities` grants; `content-desk` consumes. |

## Deferred Domains

Add future or intentionally deferred capabilities here only when they
are real enough to affect architecture or requirements.

| Candidate Domain | Why Deferred | Revisit Trigger |
|---|---|---|
| `environments` — provisioning device fingerprints, proxies, and regions | The infrastructure sits upstream of this scenario and involves third-party services with their own cost and terms. Recording and checking an environment is cheap; provisioning one is a different product. | If environment drift becomes a recurring cause of quarantine and manual attestation proves insufficient. |
| `experiments` — A/B testing content variants per identity | Requires a corpus of published posts with performance data, which does not exist. Proposing variants before any baseline exists would measure noise. | Once `CHANMGR-P1-007` has returned performance data for enough published posts to compare against. |

## Explicitly Not Domains (Decided)

| Considered | Verdict | Reason |
|---|---|---|
| A separate `warming-executor` domain | **Rejected.** | Execution is cross-cutting: warming steps and posts both need it. It is a seam with pluggable implementations, registered in `SEAMS.md`, not a product capability. |
| A `credentials` domain | **Rejected.** | `vault` owns credentials. This scenario owns a reference and nothing more; a domain here would invite storing the value. |
| A `campaigns` or `drafts` domain | **Rejected.** | Content, campaigns, claims, and approval belong to `content-desk`. This scenario receives an approved draft and knows nothing about how it was produced. |
| A `personas` domain | **Rejected.** | Persona identity — appearance, voice, scene — is `asset-studio`. An identity here holds a reference to one. |

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
