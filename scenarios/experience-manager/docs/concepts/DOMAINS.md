# Domains — Experience Manager

This document is the canonical map of product capabilities, bounded
contexts, and ownership for this scenario. Keep it current whenever a
domain is added, renamed, split, merged, or removed.

`health` is the one real domain the scaffold ships. Add your scenario's
domains to the inventory below as you build them. The scaffold also ships
one clearly fenced worked example domain (never product scope) as a
copyable reference; `vrooli scenario detemplate <scenario>` removes every
fenced example once your real domains are green.

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
| health | Report runtime readiness and dependency reachability. | Expose API/database readiness and show the UI can read live backend state. | No product data. | reporting | query | HealthHandler | `api/handlers/health/`, `ui/src/features/health/`, `packages/proto/schemas/experience-manager/v1/shared/health.proto` |
| spec | Parse and validate `experience/` specs. | Make UX intent a machine-checkable contract. | No product data in Phase 1. | validation | query | ExperienceSpec, Finding | `api/internal/spec/`, `.vrooli/schemas/scenario-experience-spec.schema.json` |
| checks | Compose experience finding checks and severity policy. | Keep parser output and provider findings behind a stable engine seam. | No product data. | validation | scoring | Check, Engine, Registry | `api/internal/checks/` |
| reconcile | Join parsed machine-tier claims to captured accessibility-tree evidence. | Close the INTENT-STRUCTURE edge without owning a browser harness. | Per-claim reconciliation evidence. | validation | reporting | AccessibilitySnapshot, ClaimVerdict | `api/internal/reconcile/` |
| attest | Record and check manual-tier claim attestations. | Keep human-reviewed experience claims honest with explicit expiry semantics. | Append-only manual attestations. | mutation | evidence | ManualAttestation | `api/internal/attestation/`, `cli/domains/contract/` |
| fleet | Compute experience coverage and debt across scenarios. | Make experience-spec adoption visible without persisted fleet state. | No persisted data. | reporting | query | FleetScenario, DebtScore | `api/internal/fleet/`, `ui/src/pages/ExperiencePages.tsx`, `cli/domains/contract/` |
| studio | Persist form-shaped spec authoring sessions and apply parser-clean drafts. | Let agents and the UI author `experience/` files through a validated contract instead of hand-editing JSON. | Authoring sessions and page drafts. | mutation | service | AuthoringSession, PageForm, FileDiff | `api/internal/authoring/`, `api/handlers/studio/`, `cli/domains/studio/` |
| render | Produce deterministic wireframes and variant comparisons from parsed page specs. | Make experience specs visually workshoppable before implementation. | Wireframe artifacts under coverage output. | reporting | authoring | WireframeRender, VariantPreview | `api/internal/render/`, `api/handlers/studio/`, `cli/domains/contract/` |
| assessment | Map experience findings into shared maturity assessments. | Make local experience maturity consumable by Test Genie and fleet reports. | No product data. | scoring | provider | MaturityAssessment | `api/internal/assessment/` |
| autofix | Register deterministic experience remediations. | Keep write-capable repairs explicit and dry-run-first. | Fix candidates only. | mutation | validation | FixCandidate, Fixer | `api/internal/autofix/` |
| validation | Mount shared and native validation services. | Expose experience validation to Test Genie, CLI, UI, and agents. | No product data in Phase 1. | provider | validation | ScenarioValidationService, ContractService | `api/handlers/validation/`, `cli/domains/contract/`, `cli/domains/provider/`, `cli/domains/fix/`, `packages/proto/schemas/experience-manager/v1/contract/contract.proto` |

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
- UI: health is exposed through the backend contract and shell-level readiness
  checks; the unused scaffolded HealthCard surface was removed.
- Storage: none; probes configured database reachability.
- Requirements: starter scaffold health only.
- Tests: handler, module, and shell accessibility tests.
- Related docs: [`../reference/api-endpoints.md`](../reference/api-endpoints.md).

### spec

- Purpose: parse `experience/` folders and own the scenario-experience-spec/v1
  contract model.
- Primary archetype: validation.
- Owns: parser-facing reports, findings, and target spec vocabulary.
- Does not own: service transport, CLI rendering, or live UI capture.
- API: `api/internal/spec/`.
- CLI: consumed through the validation domain.
- Storage: none.
- Requirements: OT-P0-001.
- Tests: parser golden fixtures land with Phase 2.

### checks

- Purpose: compose registered experience checks and enforce the advisory cap at
  ERROR.
- Primary archetype: validation / scoring.
- Owns: check registry, engine seam, severity normalization.
- Does not own: JSON parsing or maturity rendering.
- API: `api/internal/checks/`.
- CLI: none directly.
- Storage: none.
- Requirements: OT-P0-001, OT-P0-002.

### reconcile

- Purpose: compare active page claims with `bas-accessibility-snapshot/v1`
  evidence from Browser Automation Studio.
- Primary archetype: validation / evidence.
- Owns: AX snapshot parsing, binding-to-node matching, and structure claim
  verdicts for declared accessible names, keyboard reachability, reading order,
  and state affordances.
- Does not own: browser automation, screenshot capture, JSON spec parsing, or
  studio authoring.
- API: `api/internal/reconcile/`, registered through `api/internal/checks/`.
- CLI: rendered through validation commands.
- Storage: `api/internal/reconcile/schema.sql` owns the
  `reconcile_evidence` table. The reconciler stores one row per active
  default-state machine claim verdict, including skipped rows when BAS capture
  is unavailable.
- Requirements: OT-P0-003.

### attest

- Purpose: record manual-tier claim evidence with author, rationale, and expiry.
- Primary archetype: mutation / evidence.
- Owns: append-only manual attestation storage and expiry findings.
- Does not own: parser shape, machine reconciliation, or fleet scoring.
- API: `api/internal/attestation/`, registered through `api/internal/checks/`.
- CLI: `spec attest` appends the sole external write path.
- Storage: `api/internal/attestation/schema.sql` owns the
  `manual_attestations` table. Rows are append-only; refreshing evidence appends
  a new row instead of mutating old rows.
- Requirements: OT-P1-004.

### fleet

- Purpose: compute experience depth and debt across the scenario tree on read.
- Primary archetype: reporting / query.
- Owns: scenario sweep, depth summaries, debt sorting, and Fleet page data.
- Does not own: persistent caches, parser rules, or scenario generation.
- API: `api/internal/fleet/`, exposed through `ContractService.ListFleet`.
- CLI: `spec fleet`.
- UI: Fleet page reads the live sweep and falls back to static rows only when
  the API is unavailable.
- Storage: none.
- Requirements: OT-P1-005.

### studio

- Purpose: provide the form-shaped authoring API/CLI for `experience/` specs.
- Primary archetype: mutation / authoring.
- Owns: persisted authoring sessions, page-form drafts, preview diffs,
  parser-clean apply semantics, spec list/show, and binding suggestions.
- Does not own: parser rules, live UI implementation, wireframe rendering, or
  deterministic autofix.
- API: `api/handlers/studio/`, `api/internal/authoring/`.
- CLI: `author start|submit|preview|apply|discard`, plus read-only
  `spec list|show|suggest-bindings`.
- Storage: `api/internal/authoring/schema.sql` owns `authoring_sessions` and
  `authoring_pages`.
- Requirements: OT-P0-004.
- Tests: authoring round-trip applies a submitted page and re-parses with zero
  contract error findings.

### render

- Purpose: render one parsed page spec or a side-by-side variant set into stable
  HTML for workshop review.
- Primary archetype: reporting / authoring.
- Owns: wireframe HTML generation, variant comparison layout, claim annotation
  layout, stable artifact paths, and graceful image-mode degradation.
- Does not own: parser rules, AI image generation, BAS execution, or persisted
  authoring sessions.
- API: `api/internal/render/`, exposed through `StudioSessionService.RenderSpec`.
- CLI: `spec render`, `spec compare-variants`, `spec promote-variant`.
- Storage: writes deterministic review artifacts to `coverage/wireframes/`.
- Requirements: OT-P1-001.
- Tests: byte-stability, variant artifact writing, promotion validation, and
  image-mode degradation.

### assessment

- Purpose: translate neutral findings into the shared maturity assessment
  envelope.
- Primary archetype: scoring.
- Owns: maturity builder and finding-to-capability mapping.
- Does not own: the Test Genie phase descriptor itself.
- API: `api/internal/assessment/`.
- CLI: rendered through validation commands.
- Storage: none.
- Requirements: OT-P0-002.

### autofix

- Purpose: host deterministic fixers on the shared maturity-go registry.
- Primary archetype: mutation.
- Owns: fixer registration, sequential apply-order policy, BAS scaffold stubs,
  binding placeholder repair, index normalization, and finding-doc stubs.
- Does not own: judgment-heavy authoring decisions.
- API: `api/internal/autofix/`.
- CLI: `spec scaffold`, `fix preview`, `fix apply`.
- Storage: writes only when a fixer is explicitly applied.
- Requirements: OT-P1-003.

### validation

- Purpose: mount the shared `ScenarioValidationService` and native
  `ContractService` off one validation pipeline.
- Primary archetype: provider.
- Owns: Connect-RPC handlers, endpoint descriptors, CLI command bindings.
- Does not own: parser rules or maturity ladder semantics.
- API: `api/handlers/validation/`.
- CLI: `spec validate`, `provider validate`, `fix preview`, `fix apply`.
- Storage: none in Phase 1.
- Requirements: OT-P0-001, OT-P0-002.

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
| `scaffold` | Derive `bas/cases` stubs from spec entries for workflow-health governance; spec↔case reference-integrity findings both directions. | OT-P1-002. |
| `perception` | Pixel-side parsing + saliency importance vs. declared communication priorities; advisory before gating, off the CI hot path. Engine home (here vs. image-tools) deliberately undecided. | OT-P2-001. |

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
