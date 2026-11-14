# Domains — Business Health

This document is the canonical map of product capabilities, bounded
contexts, and ownership for this scenario. Keep it current whenever a
domain is added, renamed, split, merged, or removed.

The `contract` domain is the keystone: every other domain composes its
extraction and check engine. `search` federates the same extracted
intent fleet-wide.

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
| health | Report runtime readiness and dependency reachability. | Expose API/database readiness and show the UI can read live backend state. | No product data. | reporting | query | HealthHandler | `api/handlers/health/`, `ui/src/features/health/`, `packages/proto/schemas/business-health/v1/shared/health.proto` |
| contract | Validate a scenario's business contract: extract PRD + registry claims (via intent-go only), run the check set, compose read-only evidence into the traceability matrix, map findings into the shared maturity envelope, and answer both validation service mounts. | Give agents and test-genie one authoritative verdict on whether a scenario's stated intent is conformant, linked, and honest. | The manual-validations ledger; reads target scenario trees otherwise. | service | validation, reporting | Contract, Check, Finding, Assessment, Matrix, Evidence | `api/internal/extraction/`, `api/internal/checks/`, `api/internal/assessment/`, `api/internal/evidence/`, `api/internal/matrix/`, `api/handlers/validation/`, `cli/domains/contract/`, `packages/proto/schemas/business-health/v1/contract/contract.proto` |
| remediation | Deterministic fixers on the shared autofix registry behind the PreviewFix/ApplyFix RPCs (dry-run by default). | Make every `fix_class: auto` finding mechanically repairable without judgment calls. | No product data; edits target scenario trees on explicit apply. | mutation | service | Fixer, Candidate, Preview, Apply | `api/internal/autofix/`, `cli/domains/fix/` |
| wizard | Deterministic interview-driven scaffolding of conformant PRD.md + requirements/ skeletons. | Author contracts that validate clean by construction — no embedded AI, resumable, diff-preview first. | Session state under `data/`. | orchestration | mutation | Session, Question, Answer, Scaffold | `api/handlers/wizard/`, `packages/proto/schemas/business-health/v1/wizard/wizard.proto` |
| fleet | Compute-on-read business-contract debt sweep across scenarios/* with as-of stamps. | Rank the fleet worst-first by contract debt so remediation goes where it matters. | No stored state; computed on read. | aggregation | reporting | DebtScore, Laggard, AsOf | `api/internal/fleet/`, `api/handlers/fleet/`, `cli/domains/fleet/`, `packages/proto/schemas/business-health/v1/fleet/fleet.proto` |
| search | The `business-health.intent` search leaf: fleet-wide intent corpus (one doc per PRD purpose, operational target, requirement) on the shared retrieval engine, plus the token-gated search-hub control plane. | Make every scenario's stated intent semantically discoverable — capability dedup (cell #34) and pointer-only "why" lookups (cell #22). | The Qdrant collection `business-health-intent` (no scenario SQL). | service | query | IntentRecord, SourceDoc, Reconciler, ControlToken | `api/internal/aisearch/`, `api/handlers/search/`, `api/handlers/searchcontrol/`, `cli/domains/search/`, `cli/domains/reindex/`, `packages/proto/schemas/business-health/v1/search/search.proto` |

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

### contract

- Purpose: one authoritative verdict on a scenario's business contract —
  PRD.md template conformance, requirements-registry structure, intent
  linkage in both directions, evidence-backed honesty.
- Primary archetype: service / validation.
- Owns: the extraction doorway (`internal/extraction`, the only intent-go
  composition point — the single-parser ratchet), the check engine and
  registry (`internal/checks`), the finding→assessment mapping
  (`internal/assessment`), and both validation service mounts
  (`handlers/validation`: shared `ScenarioValidationService` for
  test-genie, native `ContractService` for this scenario's own surfaces).
- Does not own: run evidence (test-genie is the single writer; this
  domain reads sync artifacts only), fixer implementations (remediation),
  or maturity mechanics (delegated to `packages/maturity-go`).
- API: `api/handlers/validation/`. CLI: `validate scenario`,
  `matrix show`, `drift show`, `manual-log add` (`cli/domains/contract/`).
- Findings vocabulary: frozen in `.vrooli/maturity.json`; each code has a
  remediation doc under `docs/findings/<code>.md`.
- Live end-to-end: checks, matrix, drift, and the manual-validations ledger.

### remediation

- Purpose: deterministic, judgment-free repair of `fix_class: auto`
  findings (template-section scaffold, registry creation, status
  normalization, prd_ref stubs).
- Primary archetype: mutation / autofix.
- Owns: the fixer registry (`internal/autofix`) mounted behind the shared
  `PreviewFix`/`ApplyFix` RPCs; preview is dry-run diffs, apply is the
  explicit write half; second apply is a no-op.
- Does not own: deciding WHICH findings are auto-fixable — that is the
  maturity.json `fix_class` declaration, and it never reclassifies to
  dodge a gap.
- CLI: `fix preview`, `fix apply` (`cli/domains/fix/`).
- All six fix_class:auto codes have implemented, idempotency-tested fixers.

### wizard

- Purpose: interview-driven scaffolding of conformant contracts; the
  question model derives from the same intent-go section definitions the
  validator checks, so the two can never disagree.
- Primary archetype: orchestration / scaffolding.
- Owns: session lifecycle (resumable, under `data/`), question model,
  scaffold rendering, diff preview.
- Does not own: judgment about product intent (answers come from the
  calling agent or operator; no LLM/network calls anywhere in the path).
- API: `api/handlers/wizard/` (session RPCs for the UI). CLI: an
  interactive local flow driving the same engine in-process.
- Live: session RPCs + CLI interactive/--answers flows; round-trip property test guarantees zero-findings output.

### fleet

- Purpose: fleet-wide business-contract debt, worst-first, computed on
  read with as-of stamps (test-genie fleet-status honesty rules).
- Primary archetype: aggregation / reporting.
- Owns: the sweep, the debt score, laggard classification
  (starter-registry, template-version, unproven-claims).
- Does not own: per-scenario validation logic (it reuses the contract
  domain's engine per target).
- API: `api/handlers/fleet/`. CLI: `fleet scan` (`cli/domains/fleet/`).
- Live: ScanFleet grades the whole fleet compute-on-read (sub-second across ~113 scenarios).

### search

- Purpose: the `business-health.intent` federated leaf — semantic search
  over every scenario's PRD purpose, operational targets, and
  requirements (the D5 type-faceted single corpus).
- Primary archetype: service / query.
- Owns: the fleet intent source (composed from `internal/extraction`, the
  intent-go ratchet), embed-text composition, the tuned engine
  (`.vrooli/search.json` is the tuning SSOT), the scenario-local
  SearchService, the shared token-gated SearchControlService mount, and
  the wizard's capability-dedup Hinter.
- Does not own: the retrieval engine itself (`packages/ai-go/search`),
  the control proto (search-hub's shared contract), or ranking policy
  changes (the search-hub sweep writes tuning back through WriteConfig).
- Pointer-only rule (cell #22): hits carry anchors into PRD/requirements
  artifacts; no synthesized rationale.
- CLI: `search query|status`, `reindex run|status|cancel`.
- Evals: the `tests` block in `.vrooli/search.json` is the gold corpus;
  `internal/aisearch/recall_test.go` gates it (live via
  `BUSINESS_HEALTH_AISEARCH_LIVE=1`).

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
| None yet. | Generated scaffold. | Add after PRD-specific requirements identify future capability boundaries. |

## Non-Domains

These are important but should not become product domains:

- `api/internal/server/` — HTTP composition substrate.
- `api/internal/module/` — shared module descriptor type.
- `api/internal/modules/` — thin registry for boot/codegen.
- `api/internal/database/` — cross-cutting database infrastructure.
- `api/internal/testutil/` — cross-domain test harnesses.
- `ui/src/components/` — shared presentation primitives.
- `ui/src/test-utils/` — cross-feature testing support.

If any of these starts using product vocabulary, split the product
piece into an owning domain instead of growing infrastructure.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system shape and extension rules
- [`FLOWS.md`](FLOWS.md) — workflows and state transitions
- [`DATA.md`](DATA.md) — data ownership and storage
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — dependency contracts
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — boundary registry
- [`../internal/TESTING.md`](../internal/TESTING.md) — test strategy
