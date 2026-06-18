# Domains — Architecture Cartographer

This document is the canonical map of product capabilities, bounded
contexts, and ownership for this scenario. Keep it current whenever a
domain is added, renamed, split, merged, or removed.

The template's `notes` example domain has been removed (Phase 0 of the
implementation plan). All inventory rows below describe the cartographer's
own product domains.

## Purpose Of This Document

Use this document to answer:

- What product capabilities does this scenario expose?
- Which domain owns each concept, table, proto, endpoint, UI feature,
  CLI command, and test surface?
- Which concepts are shared, deferred, or deliberately not domains?

System-level architecture belongs in [`ARCHITECTURE.md`](ARCHITECTURE.md).
Workflow details belong in [`FLOWS.md`](FLOWS.md). Storage details
belong in [`DATA.md`](DATA.md). The pluggable signal architecture has
its own home in [`SIGNAL_LADDER.md`](SIGNAL_LADDER.md).

## Domain Inventory

The table below is the machine contract parsed by the `domains` domain's
`DomainsDocExtractor` — see [`../reference/domains-contract.md`](../reference/domains-contract.md).
The parser is header-driven; it reads the `Domain`, `Primary Archetype`,
`Source Paths`, and (optional) `Glossary` columns and ignores the rest.

| Domain | Purpose | Primary Archetype | Owns Data | Surfaces | Requirements | Source Paths | Glossary |
|---|---|---|---|---|---|---|---|
| health | Report runtime readiness and dependency reachability. | Reporting / query | No product data. | API, UI | Starter scaffold health. | `api/handlers/health/`, `ui/src/features/health/`, `packages/proto/schemas/architecture-cartographer/v1/health/` | HealthHandler |
| graph | Build the ground-truth code graph for a target scenario by delegating to language code-graph scenarios. | Service / orchestration | Cached graph snapshots and content hashes. | API, CLI, UI | OT-P0-001 (MOD-P0-001) | `api/internal/graph/`, `api/handlers/graph/`, `cli/domains/graph/`, `ui/src/features/graph/`, `packages/proto/schemas/architecture-cartographer/v1/graph/` | GraphSnapshot, FileNode, PackageNode, SymbolNode, ImportEdge, CodeGraphAdapter, Chunk |
| domains | Derive the intended domain map from on-disk sources (DOMAINS.md → api/internal folders → cli groups) via a trust ladder; report cross-surface convergence. Replaces the deleted architecture manifest. | Service / orchestration | No product data (stateless; derived on demand). | API, CLI, UI | OT-P0-002 (MOD-P0-002) | `api/internal/domains/`, `api/handlers/domains/`, `cli/domains/domains/`, `ui/src/features/domains/`, `packages/proto/schemas/architecture-cartographer/v1/domains/` | DerivedDomainMap, DerivedDomain, DomainSourceExtractor, Extraction, ScenarioLocator, DomainSource, DomainDraft, ProposedDomain |
| conflicts | Detection-only: detect drift between actual graph and the derived domain map and emit typed conflicts via a pluggable detector registry. No lifecycle (that is the campaign domain). | Service / classification | Conflict records (the photograph), suggested fixes. | API, CLI, UI | OT-P0-003, OT-P0-005 (MOD-P0-003, MOD-P0-005) | `api/internal/conflicts/`, `api/handlers/conflicts/`, `cli/domains/conflicts/`, `ui/src/features/conflicts/`, `packages/proto/schemas/architecture-cartographer/v1/conflicts/` | Conflict, Fix, Detector, Resolver, AnalyticsRecorder |
| signals | Score chunk-to-domain assignments via pluggable, deterministic signals; aggregate into explainable verdicts. | Service / scoring | Signal scores and verdict explanations. | API, CLI | OT-P0-004 (MOD-P0-004) | `api/internal/signals/`, `api/handlers/signals/`, `cli/domains/signals/`, `packages/proto/schemas/architecture-cartographer/v1/signals/` | Signal, Score, Verdict, Tier, SignalDescriptor, Aggregator, GraphContext |
| apply | Emit per-domain migration plans and execute file moves + import rewrites with build-green guardrail. | Service / mutation | Migration plans, apply history. | API, CLI, UI | OT-P0-007, OT-P0-008 (MOD-P0-007, MOD-P0-008) | `api/internal/apply/`, `api/handlers/apply/`, `cli/domains/apply/`, `ui/src/features/apply/`, `packages/proto/schemas/architecture-cartographer/v1/apply/` | Plan, Operation, OperationKind, ApplyRun, ApplyStatus, BuildBaseline, BuildGuard, Recipe |
| analytics | Persist conflict events, resolution outcomes, auto-placement verdicts, overrides, and build deltas; serve history + stats. | Reporting / query | Append-only event log. | API, CLI, UI | OT-P0-009 (MOD-P0-009) | `api/internal/analytics/`, `api/handlers/analytics/`, `cli/domains/analytics/`, `ui/src/features/analytics/`, `packages/proto/schemas/architecture-cartographer/v1/analytics/` | Event, EventKind, Placement, Override, StatsSummary |
| audit | CI-shaped orchestrator: one call runs graph extract (if needed) → domains derivation → conflicts detection, applies severity / type filters, and returns a deterministic exit-code summary. | Service / orchestration | No product data (pure orchestrator over the graph/domains/conflicts domains). | API, CLI | L5 readiness | `api/internal/audit/`, `api/handlers/audit/`, `cli/domains/audit/`, `packages/proto/schemas/architecture-cartographer/v1/audit/` | Outcome, Report, RunInput, ConflictSummary, DerivedDomainSummary, GraphSummary, CoverageSummary |
| campaign | Stateful tracker for a scenario-improvement effort: ingest a normalized findings audit from any surface (the shared ArchitectureFinding contract), drive each finding through a lifecycle, hand the agent a profile-ranked worklist (fast / balanced / long-term), and reconcile re-audits by stable id (validated / still-open / regressed). The project-plan half of the two-layer model; ingest-only, never runs detection. | Service / orchestration | Campaigns and tracked items (lifecycle state, resolution notes, effort). | API, CLI | Architecture-audit + campaign-tracker unification. | `api/internal/campaign/`, `api/handlers/campaign/`, `cli/domains/campaign/`, `packages/proto/schemas/architecture-cartographer/v1/campaign/` | Campaign, Finding, FindingStatus, CampaignLifecycle, ReauditResult, Repository, AnalyticsRecorder |

## Domain Details

### health

- Purpose: expose API/database readiness and show the UI can read live
  backend state.
- Primary archetype: reporting / query.
- Owns: health response construction and dependency status mapping.
- Does not own: product data, business rules, or scenario-specific
  domain behavior.
- API: `api/handlers/health/`.
- CLI: built-in `status` command is provided through cli-core.
- UI: `ui/src/features/health/HealthCard.tsx`.
- Storage: none; probes database reachability.
- Tests: handler, module, UI feature, and accessibility tests.
- Related docs: [`../reference/api-endpoints.md`](../reference/api-endpoints.md).

### graph

- Purpose: produce a deterministic, reproducible graph of files,
  packages, symbols, and import edges for any target scenario by
  delegating all parsing to `go-code-graph` and `typescript-code-graph`
  scenarios.
- Primary archetype: service / orchestration.
- Owns: graph adapter logic, normalization to an immutable snapshot
  model, content-hash-based caching.
- Does not own: parsing of any source code (parsing lives strictly in
  the language code-graph scenarios — this is the load-bearing
  constraint).
- API: `api/internal/graph/`, `api/handlers/graph/`.
- CLI: `cli/domains/graph/` — `arch-cart graph extract`, `arch-cart graph show`.
- UI: `ui/src/features/graph/` — graph viewer.
- Storage: SQLite snapshot store keyed by `(scenario, content_hash)`.
- Tests: unit (adapter mapping, normalization), integration (against
  fixture scenarios), performance (graphs of ≤200 files <5s; ≤2000
  files <30s).
- Related docs: [`INTEGRATIONS.md`](INTEGRATIONS.md),
  [`SIGNAL_LADDER.md`](SIGNAL_LADDER.md).

### domains

- Purpose: derive a target scenario's intended domain map from sources
  that already exist on disk — there is no per-scenario architecture
  manifest. Derivation walks a trust ladder (DOMAINS.md → api/internal
  folders → cli groups; a future API manifest sits above DOMAINS.md) and
  reports which sources agree (cross-surface convergence).
- Primary archetype: service / orchestration.
- Owns: the `DomainSourceExtractor` ladder, ladder resolution to a
  canonical `DerivedDomainMap` (with per-source provenance), the
  glossary index, and the structured-DOMAINS.md machine contract.
- Does not own: the act of comparing the derived map to the actual graph
  (that is the `conflicts` domain's responsibility); coupling scoring
  (that is the `signals` domain).
- API: `api/internal/domains/`, `api/handlers/domains/`.
- CLI: `cli/domains/domains/` — `arch-cart domains extract`,
  `arch-cart domains show`.
- UI: `ui/src/features/domains/` — derived map + per-source provenance.
- Storage: none; the map is derived on demand and not persisted.
- Tests: unit (each extractor with golden DOMAINS.md fixtures; ladder
  resolution + provenance; glossary), dogfood (own DOMAINS.md parses and
  is authoritative).
- Related docs: [`../reference/domains-contract.md`](../reference/domains-contract.md).

### conflicts

- Purpose: **detection-only.** Detect drift between actual graph and the
  derived domain map; classify each finding into a typed `Conflict`
  envelope; surface ranked fix suggestions with evidence and caveats. It
  is a stateless photograph of what is wrong now — it owns no lifecycle.
- Primary archetype: service / classification.
- Owns: the pluggable Detector registry, the Conflict envelope shape,
  conflict severity classification, cycle SCC detection and
  pattern classification (type-only, junk-drawer, cross-domain,
  within-domain), suggested-fix ranking.
- Does not own: signal scoring (that is the `signals` domain),
  mechanical fix execution (that is the `apply` domain), and — since the
  detection/tracking split — the per-finding **lifecycle** (assign /
  resolve / validate / regress), which lives in the `campaign` domain.
- API: `api/internal/conflicts/`, `api/handlers/conflicts/`.
- CLI: `cli/domains/conflicts/` — `arch-cart conflicts detect`,
  `arch-cart conflicts list`, `arch-cart conflicts show <id>`,
  `arch-cart conflicts validate`, `arch-cart conflicts detectors`,
  `arch-cart conflicts resolvers`. (Walking a finding through a lifecycle
  toward zero is `arch-cart campaign …`.)
- UI: `ui/src/features/conflicts/` — read-only detection workbench
  (list + detail). Lifecycle actions live in `ui/src/features/campaign/`.
- Storage: conflict records in SQLite (id, type, severity, locations,
  snapshot_id, payload) — no lifecycle columns.
- Plug-in seams: `Detector` interface (returns `[]Conflict` for a
  graph + derived-domain-map snapshot); `SurfaceProfile` selector
  (chooses which detectors apply to API/CLI/UI graph evidence);
  `Resolver` interface (executes a mechanical fix for a specific
  conflict type). Current detectors: `cycle`, `layering`, `naming`,
  `glossary_drift`, `mislocated_file`, `convergence_drift`,
  `coupling_smell`, `surface_coherence`, `file_cohesion`,
  `cross_scenario`, and `domains_doc_parse_warning`.
- Tests: unit (envelope serialization, detector invocation, cycle
  classification, severity), integration (against fixture scenarios
  with known cycles, mislocations, orphans).
- Related docs: [`SIGNAL_LADDER.md`](SIGNAL_LADDER.md),
  [`FLOWS.md`](FLOWS.md), [`../internal/SEAMS.md`](../internal/SEAMS.md).

### signals

- Purpose: score chunk-to-domain assignments via pluggable,
  deterministic signals; aggregate scores into explainable verdicts
  with `Reason` and `Evidence`.
- Primary archetype: service / scoring.
- Owns: the `Signal` interface (pure scoring function over an immutable
  graph snapshot), the Aggregator (weighted combination + confidence
  tiering), the explainability contract.
- Does not own: which signals exist by default (those are registered
  plug-ins documented in `SIGNAL_LADDER.md`); the derived domain map
  (that is the `domains` domain); signal weights/thresholds (those are
  cartographer-global config, not a per-scenario declaration).
- API: `api/internal/signals/`, `api/handlers/signals/`.
- CLI: `cli/domains/signals/` — `arch-cart signals score <chunk>`,
  `arch-cart signals explain <chunk>`.
- Storage: scores are computed on demand; verdicts are logged through
  the `analytics` domain.
- Plug-in seams: `Signal` interface; day-one signals: `path-token`,
  `import-cluster`, `importer-voting`, `test-coupling`,
  `symbol-glossary`, `git-co-edit`. All are pure functions; none
  mutate the graph.
- Tests: unit (each signal's scoring math; aggregator weighting; tier
  threshold logic; reproducibility under reordering).
- Related docs: [`SIGNAL_LADDER.md`](SIGNAL_LADDER.md) — the canonical
  reference for the signal contract and the default set.

### apply

- Purpose: emit per-domain migration plans and execute file moves +
  import rewrites with build-green guardrail enforcement.
- Primary archetype: service / mutation.
- Owns: migration plan emission, file-move execution, import-rewrite
  execution (delegated to `go-code-graph` / `typescript-code-graph`
  helpers), per-domain atomic commit semantics, build-green baseline
  capture and diff, `--force --note` audit.
- Does not own: which conflicts to resolve (the `conflicts` domain);
  the refactor recipes themselves (P1, in `recipes/`).
- API: `api/internal/apply/`, `api/handlers/apply/`.
- CLI: `cli/domains/apply/` — `arch-cart apply <domain>`,
  `arch-cart apply --all` (discouraged), `arch-cart migrate start`,
  `arch-cart migrate baseline-update`.
- UI: `ui/src/features/apply/` — per-domain progress and dry-run
  diffs.
- Storage: apply history rows in SQLite.
- Plug-in seams: `Recipe` interface (P1) for mechanical refactor
  patterns like extract-shared-types, invert-dependency, split-file.
- Tests: unit (plan emission, baseline diff math), integration
  (apply against fixture scenarios; build-green guard rejects when
  baseline breaks), regression (whole-scenario apply requires
  acknowledgment).
- Related docs: [`FLOWS.md`](FLOWS.md), [`../operations/RUNBOOK.md`](../operations/RUNBOOK.md).

### analytics

- Purpose: persist conflict events, resolution outcomes, auto-placement
  verdicts, overrides, and build status deltas. Serve history and
  stats commands.
- Primary archetype: reporting / query (append-only event log).
- Owns: analytics schema, event capture, history/stats queries,
  minimum-N threshold enforcement for displayed success rates.
- Does not own: the events themselves (each domain emits its own
  events through a shared analytics seam).
- API: `api/internal/analytics/`, `api/handlers/analytics/`.
- CLI: `cli/domains/analytics/` — `arch-cart analytics events`,
  `arch-cart analytics stats`, `arch-cart analytics placements`,
  `arch-cart analytics override-record`, `arch-cart calibrate` (P2).
- UI: `ui/src/features/analytics/` — history dashboards.
- Storage: append-only SQLite event log.
- Threshold rule: success rates are not surfaced until at least N=5
  observations of the same conflict pattern.
- Tests: unit (schema, threshold enforcement), integration (event
  capture across all domains), performance (history query bounded by
  scenario size).
- Related docs: [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md).

### audit

- Purpose: a CI-shaped surface that gates merges on architectural drift.
  One call orchestrates graph extract (if no snapshot is cached),
  domains derivation, and the conflicts detector chain; applies
  severity / type filters; returns a deterministic, machine-readable
  summary mapped to process exit codes (0 clean / 1 findings ≥ threshold
  / 2 tool error / 3 usage error).
- Primary archetype: service / orchestration.
- Owns: the orchestration sequencing, the severity threshold (`fail_on`),
  the type filters (`include_types` / `exclude_types`), and the
  `AuditOutcome` mapping.
- Does not own: drift detection (lives in `conflicts`), graph extraction
  (lives in `graph`), domain derivation (lives in `domains`). Audit is
  a thin orchestrator, not a detector.
- API: `api/internal/audit/`, `api/handlers/audit/` — `AuditService.Run`.
- CLI: `cli/domains/audit/` — `arch-cart audit run <scenario>` with
  `--fail-on={info|warn|error|blocker}`, `--include-types`,
  `--exclude-types`, `--json`.
- UI: deferred to a follow-up. The plan's greenfield rule forbids a
  placeholder page; either ship a working report or leave the route
  absent. This audit cut omits the route deliberately.
- Storage: none (stateless orchestrator).
- Tests: unit (filter / threshold / outcome math; intMap; severityRank),
  integration (against the three target scenarios with wall-clock
  budgets per the L5 plan).

### campaign

- Purpose: the stateful tracker that handholds an agent through a large
  scenario-improvement effort — the workflow that failed on
  swarm-manager because the surface area outgrew what an agent could
  track by hand. It ingests a normalized findings audit from any surface
  (the shared `ArchitectureFinding` contract emitted by test-genie),
  tracks each finding through the conflict lifecycle, hands the agent a
  profile-ranked worklist, and on each re-audit reconciles by stable id:
  findings that vanished become `validated`, findings that persist stay
  open, and findings that (re)appear are flagged as regressions.
- Primary archetype: service / orchestration (longitudinal — it is the
  process half of the two-layer model, not a point-in-time validator).
- Owns: campaigns, tracked items, lifecycle state, effort hints, and
  resolution notes (the `campaigns` + `campaign_items` tables). Ordering
  is a profile-strategy seam (`internal/campaign/ranking.go`): FAST
  ("make it work now" — gating sources first, cheapest path to green),
  BALANCED (legacy: regressions → cycles → severity), LONG_TERM
  ("best solution" — structural root-causes before symptoms).
- Does not own: detection. The tracker NEVER runs detectors and never
  calls test-genie or the health CLIs — findings arrive only by ingest
  (push), so there is no detection↔tracking cycle. Reconciliation keys on
  the `afid` stable id computed by the shared
  `packages/proto/architecture/findingid` helper, so a finding matches
  across the test-genie→cartographer boundary.
- API: `api/internal/campaign/`, `api/handlers/campaign/` —
  `CampaignService` (CreateCampaign, GetCampaignStatus,
  NextCampaignStep, ResolveItem, ApplyItem, ReauditCampaign,
  CloseCampaign).
- CLI: `cli/domains/campaign/` — `arch-cart campaign
  {create,status,next,resolve,apply,reaudit,close}`. `next` takes
  `--profile fast|balanced|long-term`; `create`/`reaudit` take
  `--from-audit <file|->`, a test-genie `--json` SuiteExecutionResult.
- UI: deferred to a follow-up (greenfield rule forbids a placeholder
  page; the campaign UI is a separate cut).
- Storage: SQLite (`campaigns`, `campaign_items`); reconciliation
  primary key is `(campaign_id, stable_id)`.
- Tests: unit (ingest / reconcile / regression-detection / profile
  worklist ordering / cosmetic-stable reconciliation), plus a live
  create → next → resolve → reaudit → close loop.

## Shared Concepts

| Concept | Meaning | Owner |
|---|---|---|
| Domain | Product capability boundary that should be easy to find, test, and delete. | `DOMAINS.md` defines the map; code owns implementation. |
| Surface | API, UI, CLI, or contract layer exposing the same product capability. | `ARCHITECTURE.md`. |
| Seam | Test-substitutable boundary wired once in production. | `../internal/SEAMS.md`. |
| Conflict | A typed drift finding with id, type, severity, locations, description, suggested_fixes, and evidence. The envelope is stable across versions. | `conflicts` domain. |
| Signal | A pure scoring function over an immutable graph snapshot that produces `(value, reason, evidence)` for one `(chunk, domain)` pair. | `signals` domain; [`SIGNAL_LADDER.md`](SIGNAL_LADDER.md). |
| Verdict | The aggregator's combined output across all signals for a given `(chunk, domain)` pair, with confidence tier (`auto_place`, `suggest`, `conflict`). | `signals` domain. |
| Chunk | A semantic unit within a file — a top-level function, type, constant, or grouped declaration — that can be reassigned to a different domain. | `graph` domain. |
| Recipe | A mechanical executor for a known refactor pattern (P1 onward). | `apply` domain. |

## Deferred Domains

| Candidate Domain | Why Deferred | Revisit Trigger |
|---|---|---|
| `recipes` | The Recipe interface ships with `apply` in v0.1 with zero recipes registered. A separate `recipes` domain only makes sense when ≥3 recipes are co-resident. | When `extract-shared-types`, `invert-dependency`, and `split-file` are all implemented (OT-P1-002). |
| `embeddings` / semantic-ranker | Deferred to P2 — see Intentional Deviation in [`ARCHITECTURE.md`](ARCHITECTURE.md). | After v1 evidence that deterministic signal residual is high. |
| `maturity-scorer` | OT-P2-006; rolls up findings into the L0–L5 scoring ladder. | After analytics has enough cross-scenario data to compute scores meaningfully. |

## Non-Domains

These are important but should not become product domains:

- `api/internal/server/` — HTTP composition substrate.
- `api/internal/app/` — production bootstrap and domain service wiring.
- `api/internal/module/` — shared module descriptor type.
- `api/internal/modules/` — thin registry for boot/codegen.
- `api/internal/database/` — cross-cutting database infrastructure.
- `api/internal/testutil/` — cross-domain test harnesses.
- `api/internal/config/` — cartographer-global control-surface levers.
- `api/internal/suppressions/` — in-repo `// arch:allow` marker scanning/writing substrate.
- `ui/src/components/` — shared presentation primitives.
- `ui/src/test-utils/` — cross-feature testing support.

If one of these starts using product vocabulary, split the product
piece into an owning domain instead of growing infrastructure.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system shape and extension rules
- [`SIGNAL_LADDER.md`](SIGNAL_LADDER.md) — pluggable signal contract
- [`FLOWS.md`](FLOWS.md) — workflows and state transitions
- [`DATA.md`](DATA.md) — data ownership and storage
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — dependency contracts
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — boundary registry
- [`../internal/TESTING.md`](../internal/TESTING.md) — test strategy
