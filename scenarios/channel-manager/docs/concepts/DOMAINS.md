# Domains — Channel Manager

This document is the canonical map of product capabilities, bounded
contexts, and ownership for this scenario. Keep it current whenever a
domain is added, renamed, split, merged, or removed.

The scaffold also ships one clearly fenced worked example domain (never
product scope) as a copyable reference; `template-manager detemplate
<scenario>` removes every fenced example once the real domains are green.

**Status: implemented.** The account-operations domain is intentionally
consolidated in `channelmanager`: its identity, platform-policy, warming,
queue, signal, release, and ledger concepts share one durable action lifecycle
and SQLite state model. The earlier design-only folders (`identities`,
`platforms`, `warming`, `queue`, and `signals`) were never shipped and are not
ownership boundaries in the current codebase.

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
| channelmanager | Own account identity policy and its complete action lifecycle. | Every operation affecting a managed account must be governed by the same identity, descriptor, cadence, evidence, and audit rules. | Identities, attestations, bindings, queued actions, action projection, append-only ledger events, releases, metric delivery, observations, baselines, and flags. | service | workflow, policy, reporting | Identity, Platform, Action, Release, Observation | `api/internal/channelmanager/`, `api/handlers/channelmanager/`, `api/integrations/bas/`, `api/integrations/contentdesk/`, `cli/domains/channelmanager/`, `ui/src/api/`, `ui/src/pages/`, `ui/src/layout/`, `data/platforms/`, `.vrooli/browser-automation-studio/` |

## Domain Details

## Conceptual responsibilities inside `channelmanager`

The following sections remain the canonical conceptual map. They are not
separate source-code packages: each is a responsibility within the one
domain-owned service because every action must share the same identity and
audit record.

### identities

- Purpose: hold one row per account the fleet operates, plus the environment that
  account depends on.
- Primary archetype: CRUD, with a policy edge for lane grants.
- Owns: platform id, handle, purpose tag (`brand` or `persona-actor`), optional
  persona reference into `asset-studio`, environment reference, credential-authority reference
  reference, lane grants, lifecycle status.
- Does not own: the credential itself (credential authority), what a lane means
  (`content-desk`), or the persona's visual identity (`asset-studio`).
- Invariant: **no credential value is ever persisted.** An identity carries an authority
  path; the value is read at execution time and never stored, logged, or returned.
- Invariant: purpose tag is load-bearing, not descriptive. It selects the warming
  program, decides whether disclosure rules apply, and changes which flags matter.
- Environment note: device fingerprint, proxy region, and geo consistency are
  **recorded and checked, never provisioned.** The scenario cannot detect a leak in
  an environment it did not create, which is why the precondition gate is manual
  attestation rather than an automated probe (D-006).
- Requirements: `CHANMGR-P0-001`, `CHANMGR-P0-002`, `CHANMGR-P1-010`.

### platforms

- Purpose: describe what each platform allows so that no platform behaviour is
  hard-coded.
- Primary archetype: policy / reference data.
- Owns: supported formats and their limits, cadence ceilings per action kind,
  media constraints, disclosure requirements, and which executors exist for which
  actions.
- Does not own: channel strategy or account purpose — those are canon in
  `docs/marketing/strategy/CHANNELS.md` and are read-only input here.
- Invariant: descriptor files are immutable boot-time inputs. They are loaded and
  validated on startup; runtime state never copies, reseeds, or rewrites them.
- Invariant: **the ceiling always wins.** A warming program that asks for more
  actions than the platform ceiling allows is clamped or rejected at plan
  generation, so a badly written program fails safe rather than at the platform.
- Invariant: **the descriptor abstraction is not proven by one descriptor.**
  `CHANMGR-P0-003` claims that adding a platform requires no code change; a claim
  validated against a single instance is a hypothesis with a JSON file attached.
  The requirement is met only when at least two *structurally different* platforms
  are described — one text-led and one video-led, with different action kinds — and
  neither required a code change. TikTok and X are the intended pair.
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
- Invariant: **scheduling under a reached ceiling is a strict ordering, not
  first-come.** A release outranks maintenance engagement; maintenance defers
  rather than drops, and a deferred action whose phase window closes before it runs
  is recorded as a miss rather than silently skipped (D-011). Both plausible
  behaviours look like a working queue from the outside, which is why the ordering
  is declared here rather than settled in the scheduler.
- Requirements: `CHANMGR-P0-006`, `CHANMGR-P0-007`, `CHANMGR-P0-008`,
  `CHANMGR-P0-013`, `CHANMGR-P0-014`, `CHANMGR-P0-015`, `CHANMGR-P1-001`,
  `CHANMGR-P1-002`, `CHANMGR-P1-003`, `CHANMGR-P1-005`, `CHANMGR-P1-008`.

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
- Capture note: **at P0 every observation is entered by hand.** No executor reads
  metrics back from a platform, so `CHANMGR-P0-016` is satisfied by a CLI verb and a
  console entry surface, not by a collector. This is not a gap — a pasted number is a
  real data point, and the baseline is the whole value. Automated capture arrives
  with the browser executor and platform APIs, and never becomes the only path.
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

## Cross-Domain Requirements

Two requirements are satisfied by composing several domains rather than by any one
of them. They are listed here so that no requirement is left without an owner, and
so that neither is mistaken for a UI detail that can be deferred.

| Requirement | Composed from | Ownership rule |
|---|---|---|
| `CHANMGR-P0-019` — operator console | Roster from `identities`, the day's due actions from `queue`, program progress from `warming`, history and flags from `signals`. | No domain owns the console; each domain owns the read model behind its panel. A panel with no query behind it is a domain gap, not a UI gap. At P0 the console is also the **only** write surface for two things: manual action completion with evidence (`CHANMGR-P0-015`) and observation entry (`CHANMGR-P0-016`), so it is load-bearing rather than presentational. |
| `CHANMGR-P1-009` — per-platform post preview | Limits, media constraints, and disclosure rules from `platforms`; the pending action and its payload from `queue`. | `platforms` owns what the rules are; `queue` owns rendering the pending action against them. Neither owns the preview alone. |

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
| A `credentials` domain | **Rejected.** | The credential authority owns credentials. This scenario owns a reference and nothing more; a domain here would invite storing the value. |
| A `campaigns` or `drafts` domain | **Rejected.** | Content, campaigns, claims, and approval belong to `content-desk`. This scenario receives an approved draft and knows nothing about how it was produced. |
| A `personas` domain | **Rejected.** | Persona identity — appearance, voice, scene — is `asset-studio`. An identity here holds a reference to one. |

## Non-Domains

These are important but should not become product domains:

- `api/internal/server/` — HTTP composition substrate.
- `api/internal/module/` — shared module descriptor type.
- `api/internal/modules/` — thin registry for boot/codegen.
- `api/internal/database/` — cross-cutting database infrastructure.
- `api/internal/testutil/` — cross-domain test harnesses.
- `api/handlers/health/` — operational readiness probe, not account health.
- `ui/src/features/health/` — readiness display, not account health.
- `packages/proto/schemas/channel-manager/v1/shared/health.proto` — generic readiness wire shape.
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
