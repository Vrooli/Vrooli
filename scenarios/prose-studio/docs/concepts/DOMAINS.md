# Domains — Prose Studio

This document is the canonical map of product capabilities, bounded
contexts, and ownership for this scenario. Keep it current whenever a
domain is added, renamed, split, merged, or removed.

Prose Studio owns eight product domains plus the scaffold's `health`. The
inventory below is the real map; the fenced `notes` example remains only as a
copyable vertical-slice reference and is removed by
`template-manager detemplate prose-studio` once the first real domain is green.

The organising idea: **a candidate set is the unit of work, not a single
output.** `generation` produces a set, `measurement` describes it, `selection`
picks from it without ranking by taste, `sessions` converges it across rounds,
and `documents` composes many converged sections into one long-form piece.
`styles` and `profiles` are the data that condition all of it, and
`declarations` is how a consuming scenario supplies that data from its own
repository folder without this scenario knowing the consumer exists.

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
| health | Report runtime readiness and dependency reachability. | Expose API/database readiness and show the UI can read live backend state. | No product data. | reporting | query | HealthHandler | `api/handlers/health/`, `ui/src/features/health/`, `packages/proto/schemas/prose-studio/v1/shared/health.proto` |
| styles | Own the writing-voice record and its resolution. | A named, versioned voice that many consumers reference and that freezes once something ships against it. | `style`, `style_version`. | crud | service, versioning | Style, StyleVersion, Exemplar, AntiPattern, Target, Lexicon | `api/internal/styles/`, `api/handlers/styles/`, `cli/domains/styles/`, `ui/src/features/styles/` |
| profiles | Own the generation configuration record and the kind registry. | Everything that shapes a generation — sampler, constraints, policy, budget, role — as one addressable, versioned record. | `profile`, `profile_version`, `axis_space` (P1). | crud | service, registry | Profile, ProfileVersion, SamplerStrategy, SelectionPolicy, ContextPolicy, Budget | `api/internal/profiles/`, `api/handlers/profiles/`, `cli/domains/profiles/`, `ui/src/features/profiles/` |
| declarations | Own consumer-declared records: discovery, parsing, hashing, collision, lifecycle. | A consuming scenario owns its own voice as files in its own repo folder, with zero integration code. | `declaration` (registration state only; the records themselves stay owned by `styles`/`profiles`). | service | ingestion, validation | Declaration, DeclarationKey, ContentHash, RegistrationStatus | `api/internal/declarations/`, `api/handlers/declarations/`, `cli/domains/declarations/`, `ui/src/features/declarations/` |
| measurement | Own deterministic text measurement and comparability rules. | Diversity and readability stated as reproducible numbers rather than asserted, and only compared when comparable. | No tables; writes measurements onto candidates owned by `generation`. | service | computation | Measurement, MetricTier, DiversityBasis, ComparabilityKey | `api/internal/measurement/`, `packages/textmetrics/` |
| generation | Own candidates, rounds, sampler execution, and gateway calls. | The act of producing k candidates with full provenance, and the only place that talks to ai-gateway. | `round`, `candidate`, `verbalized_hint`. | workflow | service, integration | Candidate, Round, Strategy, Provenance, VerbalizedHint | `api/internal/generation/`, `api/handlers/generation/`, `cli/domains/generation/` |
| selection | Own eligibility gating and policy-based choice. | Quality as a deterministic floor and choice by rarity or coverage, so nothing ranks by taste. | `selection_event`. | service | policy | SelectionPolicy, Eligibility, Constraint, SelectionEvent | `api/internal/selection/`, `api/handlers/selection/` |
| sessions | Own the append-only convergence graph and its verbs. | Converging a candidate set across rounds without losing what was rejected or why. | `session`. | workflow | state-machine | Session, Pin, Reject, Reroll, Commit, Abandon | `api/internal/sessions/`, `api/handlers/sessions/`, `cli/domains/sessions/`, `ui/src/features/variation/` |
| documents | Own long-form structure: outline, sections, context budget, coherence, assembly. | A blog post built from converged sections, with feasibility known before work starts. | `document`, `section`, `context_snapshot`. | workflow | composition | Document, Outline, Section, ContextPolicy, ContextSnapshot, Coherence | `api/internal/documents/`, `api/handlers/documents/`, `cli/domains/documents/`, `ui/src/features/documents/` |

<!-- EXAMPLE-DOMAIN:notes START -->
### Example domain — `notes` (removed by `template-manager detemplate`)

The template ships `notes` as a worked CRUD vertical slice with a binary
upload exception. Copy its shape for your own domains, then remove it.

| Domain | Responsibility | Purpose | Owns Data | Primary Archetype | Secondary Traits | Glossary | Source Paths |
|---|---|---|---|---|---|---|---|
| notes | Provide the worked CRUD reference with attachment upload exception. | Demonstrate the expected vertical slice for a real domain. | Notes and attachment metadata. | crud | service | Note, Attachment | `api/internal/notes/`, `api/handlers/notes/`, `cli/domains/notes/`, `ui/src/features/notes/`, `packages/proto/schemas/prose-studio/v1/notes/` |

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

### Build order

Domains are listed in dependency order. A domain is built only after every
domain it reads exists, per the ordering rule in
[`ARCHITECTURE.md`](ARCHITECTURE.md) §Extension Rules — proto, API, transport,
CLI, then UI, finishing one domain before starting the next.

```
packages/textmetrics ──> measurement ──┐
                                       v
styles ──> profiles ──> generation ──> selection ──> sessions ──> documents
   ^          ^
   └── declarations ──┘   (writes through the styles/profiles write paths;
                           owns registration state, never the records)
```

`declarations` deliberately does not own style or profile rows. Shared data gets
exactly one owning domain, so `styles` owns every style regardless of whether a
file or an operator created it; `declarations` owns only the file lifecycle and
drives the same write paths with a declared-source marker.

### styles

- Purpose: hold a writing voice as a versioned, addressable record so many consumers reference one voice rather than each re-describing it.
- Primary archetype: CRUD / entity. Secondary traits: composition, versioning.
- Owns: style and style_version records; exemplars, directives, anti-patterns, lexicon, targets, axis defaults; single-parent `extends` with write-time cycle detection; the declared merge order; conflict reporting when merged targets have an empty intersection; text conformance reporting with span locations.
- Does not own: how a style is used at generation time, or which consumer declared it.
- Storage: domain-owned SQLite schema. A style version freezes the moment a committed output references it.
- Requirements: `PS-P0` style record and resolution, style version immutability, text conformance reporting.

### profiles

- Purpose: make every knob that shapes a generation one addressable, versioned record, so adding a new way of writing is data rather than a deploy.
- Primary archetype: CRUD / entity. Secondary traits: registry.
- Owns: profile and profile_version records covering style references, sampler kind and parameters, constraints, selection policy and parameters, measurement tiers, context policy, budget, and gateway role; profile resolution returning both the merged effective configuration and the exact instruction text before it is sent; the introspectable kind registry that returns each strategy, policy, transform, and metric kind with its parameter schema.
- Does not own: the strategy and policy *algorithms* themselves, which stay in code as closed vocabularies because each carries invariants the gates depend on. Adding a new *kind* is a code change; adding a new *configuration of an existing kind* is data.
- Storage: domain-owned SQLite schema.
- Requirements: `PS-P0` profile record and inspectable resolution, introspectable kind registry.

### declarations

- Purpose: let a consuming scenario own its own voice as version-controlled files inside its own folder, so a team edits a file rather than requesting a change here.
- Primary archetype: service. Secondary traits: ingestion, validation.
- Owns: discovery by startup scan over the declaration glob plus an explicit reindex verb; parsing; content-hash versioning; namespaced key allocation with `local/` reserved for operator-authored records; key-collision refusal naming both paths; registration lifecycle across registered, invalid, and unregistered; one-way export of an operator-authored record to file content; the standalone validator exposed over RPC and CLI.
- Does not own: style or profile records themselves, and nothing about any specific consumer. Registration is the only coupling; no consumer name appears in this scenario's code.
- Storage: registration state only. A declared record's row is a projection marked `authority: file`, checked on every write path; API writes to it are refused naming the file. Deleting a file marks the record unregistered rather than deleting it, so historical provenance stays resolvable.
- Requirements: `PS-P0` consumer-owned declarations, file-as-authority, standalone declaration validator.

### measurement

- Purpose: state diversity and readability as reproducible numbers, and refuse to compare two sets that are not comparable.
- Primary archetype: service. Secondary traits: computation.
- Owns: the deterministic metric tier — compression ratio, self-repetition, lexical homogenization, distinct-n, burstiness, three readability grades, type-token ratio, lexicon flags — per candidate and per set; the pairwise similarity matrix that coverage binning and rarest-above-threshold both consume; the comparability rule that two sets compare only when their effective sampling keys and effective max-output-token values and sources match.
- Does not own: any inferential judgement. An optional judge tier (P1) may gate pass or fail; it may never rank.
- Storage: none of its own. Computation lives in `packages/textmetrics`, a shared package consumable with no runtime dependency on this scenario's API.
- Requirements: `PS-P0` deterministic measurement at birth, comparability enforcement, shared metrics package.

### generation

- Purpose: produce k candidates with enough provenance to reproduce and compare them later.
- Primary archetype: workflow. Secondary traits: service, integration.
- Owns: round and candidate records; sampler strategy execution for the direct and verbalized-distribution kinds; the single ai-gateway integration point; per-request derivation of max output tokens from profile data; full candidate provenance including resolved model reference and that model's declared context window; the verbalized ordering signal stored as an ordinal marked uncalibrated; whole-set cost attribution.
- Does not own: whether a candidate is eligible or chosen. It produces the set; `selection` decides.
- Storage: domain-owned SQLite schema.
- Requirements: `PS-P0` sampler strategies, candidate provenance, whole-set cost attribution, uncalibrated hint containment, gateway-only inference, machine-generation disclosure.

### selection

- Purpose: separate a deterministic quality floor from the choice made among candidates that clear it.
- Primary archetype: service. Secondary traits: policy.
- Owns: deterministic constraint evaluation marking violating candidates ineligible with a named reason while retaining them in the set; the five shipped policies; the agent-mode and human-mode defaults; the selection event recorded on every commit with measurements snapshotted at choice time and the considered candidates retained.
- Does not own: any ranking by taste. No judge-based ranker exists at any priority, enforced by a static check that no policy or query reads the uncalibrated hint.
- Storage: selection_event, with a reserved outcome reference so external-outcome scoring is possible later without a migration.
- Requirements: `PS-P0` constraint gating that retains, selection policies and defaults, no taste-based ranking, selection recording without learning.

### sessions

- Purpose: let a human or an agent converge a candidate set across rounds without losing history.
- Primary archetype: workflow. Secondary traits: state machine.
- Owns: the session as an append-only graph of rounds and candidates linked by derivation, mutating no record in place; the verbs generate, reroll, pin, unpin, reject, refine, commit, and abandon; reroll semantics that regenerate only unpinned slots while conditioning explicitly on the pinned and rejected sets; the per-session budget ceiling; the single-call generate wrapper that gives an agent one decided result over the same code path rather than a second one.
- Does not own: long-form structure. A section owns a session; a document owns sections.
- Storage: domain-owned SQLite schema.
- Requirements: `PS-P0` session as append-only history, reroll with negative conditioning, single-call agent path.

### documents

- Purpose: build a long-form document from converged sections, with feasibility known before work starts rather than discovered partway through.
- Primary archetype: workflow. Secondary traits: composition.
- Owns: document, section, and context snapshot records; the outline represented as an ordinary candidate so outline variation reuses the whole sampling and selection stack with no new code; section profile resolution by inheritance from the document; section context assembled from the outline, the committed text of prior sections, the declared intents of following sections, and the resolved profile; the declared context policy that caps accumulation; the static feasibility check at profile-validation time against the profile's worst-case section, not its first; the dynamic pre-call check per section; cross-section repetition and style-drift measurement; assembly to ordered text plus structure.
- Does not own: rendering to any presentation format, which belongs to `document-manager`'s generation spine.
- Storage: domain-owned SQLite schema.
- Requirements: `PS-P0` long-form composition level, section context and budget, feasibility before work, document coherence and assembly.

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
| `axis_spaces` (specified diversity) | P1. Shipping it inside `profiles` first keeps the record count down; it becomes its own domain only if cell planning and coverage binning grow beyond a profile field. Until it lands, this scenario can *measure* set diversity but cannot *guarantee* it. | Cell planning acquires its own lifecycle, or a second consumer needs axis spaces independent of a profile. |
| `conformance` (the test-genie phase wrapper) | P1 by operator decision. The validation logic ships at P0 inside `declarations` and is callable over RPC and CLI; only automatic invocation during a consuming scenario's own suite waits, because the phase descriptor needs a North Star, a gated L0–L4 ladder, a runtime-computed assessment, and structured remediation docs. | The declaration validator is stable and a second consumer has declared records worth gating a build on. |
| `transforms` | P1. Reading-level, elaboration, and simplification are provenance-recorded operations from P0 but are not a typed registry until there is more than one real caller. | A consumer needs to compose transforms or select among them by parameter. |
| `learning` | P2 and deliberately constrained. Configuration-level bandits over durable entities are legitimate; a preference model over candidates is not, because candidates are ephemeral and candidate-level ranking is ill-posed rather than merely data-poor. | Months of real selection data exist *and* a floor-then-rarity formulation is specified. |

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
