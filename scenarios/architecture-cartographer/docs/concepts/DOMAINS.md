# Domains — Architecture Cartographer

This document is the canonical map of product capabilities, bounded
contexts, and ownership for this scenario. Keep it current whenever a
domain is added, renamed, split, merged, or removed.

The template's `notes` example domain has been removed (Phase 0 of the
implementation plan). All inventory rows below describe the cartographer's
own product domains.

> **Contract status — proposed v2 shape.** This document has been migrated to
> the *authored-claims-only* contract proposed in the intent-alignment work.
> The inventory table is still backward-compatible with today's parser (which
> reads only `Domain` / `Primary Archetype` / `Glossary` / `Source Paths`). The
> substrate is now split into **Transport Zones**, **Shared Substrate**, and
> **Not Owned**, superseding the single `## Non-Domains` section. The v2 reader
> (for `Responsibility`, `Secondary Traits`, `Surface Exceptions`, and the split
> substrate) and the DOMAINS.md **quality gate** (coverage, overlap,
> concentration, name quality) are not yet implemented; until they ship, those
> sections are human-facing.

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
It is **authored-claims-only**: every column is either an irreducible human
judgment (name, responsibility, archetype, owned vocabulary) or a claim the
audit verifies against the code graph (source paths). Machine-derivable facts —
which surfaces a domain exposes, which requirements validate it — are *not*
authored here; the audit derives them and renders them in a generated view, so
there is no second source of truth to drift.

The parser is header-driven (columns may be reordered):

- **Domain** (required) — unique product-capability name.
- **Responsibility** (required) — one sentence; the semantic anchor the Tier 2/3
  intent checks compare the domain's code against.
- **Primary Archetype** (required) — a single value from the controlled archetype
  vocabulary (pinned in [`../reference/domains-contract.md`](../reference/domains-contract.md)).
- **Secondary Traits** (optional) — additional archetype traits, comma-separated.
- **Glossary** (optional) — the vocabulary this domain owns and that no other
  domain should use (the `glossary_drift` signal checks the "nowhere else" half).
- **Source Paths** (required) — repo-relative globs the domain owns; the audit
  checks these for coverage, overlap, concentration, and on-disk reality.

| Domain | Responsibility | Primary Archetype | Secondary Traits | Glossary | Source Paths |
|---|---|---|---|---|---|
| health | Report runtime readiness and dependency reachability. | reporting | — | HealthHandler | `api/handlers/health/`, `ui/src/features/health/`, `packages/proto/schemas/architecture-cartographer/v1/health/` |
| graph | Build the ground-truth code graph for a target scenario by delegating to language code-graph scenarios. | service | orchestration | GraphSnapshot, FileNode, PackageNode, SymbolNode, ImportEdge, CodeGraphAdapter, Chunk | `api/internal/graph/`, `api/handlers/graph/`, `cli/domains/graph/`, `ui/src/features/graph/`, `packages/proto/schemas/architecture-cartographer/v1/graph/` |
| domains | Derive the intended domain map from on-disk sources via a trust ladder and report cross-surface convergence. | service | orchestration | DerivedDomainMap, DerivedDomain, DomainSourceExtractor, Extraction, ScenarioLocator, DomainSource, DomainDraft, ProposedDomain | `api/internal/domains/`, `api/handlers/domains/`, `cli/domains/domains/`, `ui/src/features/domains/`, `packages/proto/schemas/architecture-cartographer/v1/domains/` |
| conflicts | Detect drift between the actual graph and the derived domain map and emit typed conflicts (detection only — no lifecycle). | service | classification | Conflict, Fix, Detector, Resolver, AnalyticsRecorder | `api/internal/conflicts/`, `api/handlers/conflicts/`, `cli/domains/conflicts/`, `ui/src/features/conflicts/`, `packages/proto/schemas/architecture-cartographer/v1/conflicts/` |
| signals | Score chunk-to-domain assignments via pluggable deterministic signals and aggregate them into explainable verdicts. | service | scoring | Signal, Score, Verdict, Tier, SignalDescriptor, Aggregator, GraphContext | `api/internal/signals/`, `api/handlers/signals/`, `cli/domains/signals/`, `packages/proto/schemas/architecture-cartographer/v1/signals/` |
| apply | Emit per-domain migration plans and execute file moves and import rewrites under a build-green guardrail. | service | mutation | Plan, Operation, OperationKind, ApplyRun, ApplyStatus, BuildBaseline, BuildGuard, Recipe | `api/internal/apply/`, `api/handlers/apply/`, `cli/domains/apply/`, `ui/src/features/apply/`, `packages/proto/schemas/architecture-cartographer/v1/apply/` |
| analytics | Persist conflict events, resolutions, verdicts, overrides, and build deltas; serve history and stats. | reporting | — | Event, EventKind, Placement, Override, StatsSummary | `api/internal/analytics/`, `api/handlers/analytics/`, `cli/domains/analytics/`, `ui/src/features/analytics/`, `packages/proto/schemas/architecture-cartographer/v1/analytics/` |
| audit | CI-shaped orchestrator: one call runs graph extract → domains derivation → conflicts detection and returns a deterministic exit-code summary. | service | orchestration | Outcome, Report, RunInput, ConflictSummary, DerivedDomainSummary, GraphSummary, CoverageSummary | `api/internal/audit/`, `api/handlers/audit/`, `cli/domains/audit/`, `packages/proto/schemas/architecture-cartographer/v1/audit/` |
| campaign | Stateful tracker for a scenario-improvement effort: ingest a findings audit, drive each finding through a lifecycle, and reconcile re-audits by stable id (ingest only — never detects). | service | orchestration | Campaign, Finding, FindingStatus, CampaignLifecycle, ReauditResult, Repository, AnalyticsRecorder | `api/internal/campaign/`, `api/handlers/campaign/`, `cli/domains/campaign/`, `packages/proto/schemas/architecture-cartographer/v1/campaign/` |

## Surface Exceptions

The audit derives which surfaces (API / CLI / UI) a domain exposes from its
source paths; this section declares the surfaces a domain *intentionally* does
not expose, so a derived absence is not reported as a gap. `permanent` absences
are by design; `deferred` absences are tracked as future work — they may be
nudged but never gate.

| Domain | Absent Surface | Status | Why |
|---|---|---|---|
| health | CLI | permanent | Readiness is the cli-core built-in `status`; no dedicated command group. |
| signals | UI | permanent | Scoring is an internal/CLI capability; there is no human dashboard. |
| audit | UI | deferred | CI surface; the report UI is a later cut (the greenfield rule forbids a placeholder page). |
| campaign | UI | deferred | The lifecycle UI is a separate cut. |

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
  `coupling_smell`, `surface_coherence`, `cross_scenario`, and
  `domains_doc_parse_warning`.
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

## Transport Zones

Domain-adjacent edges that route requests into domains but are not themselves
products. A requirement whose `validation[].ref` lands here resolves to
`intent.req_transport_owned` (info), not an orphan.

_None._ In this scenario each domain's transport edge lives under that domain's
own `api/handlers/<domain>/` path and is therefore domain-owned, not a shared
zone.

## Shared Substrate

Vocabulary-free infrastructure used across domains. These are important but must
never become product domains:

- `api/internal/server/` — HTTP composition substrate.
- `api/internal/app/` — production bootstrap and domain service wiring.
- `api/internal/module/` — shared module descriptor type.
- `api/internal/modules/` — thin registry for boot/codegen.
- `api/internal/database/` — cross-cutting database infrastructure.
- `api/internal/testutil/` — cross-domain test harnesses.
- `api/internal/config/` — cartographer-global control-surface levers.
- `api/internal/observability/` — dev-only profiling and diagnostics substrate.
- `api/internal/suppressions/` — in-repo `// arch:allow` marker scanning/writing substrate.
- `ui/src/components/` — shared presentation primitives.
- `ui/src/test-utils/` — cross-feature testing support.

If one of these starts using product vocabulary, split the product piece into an
owning domain instead of growing infrastructure.

## Non-Domains

Machine-readable compatibility section for the current `DomainsDocExtractor`.
It mirrors the shared-substrate paths above until the v2 substrate reader ships.

- `api/internal/server/` — HTTP composition substrate.
- `api/internal/app/` — production bootstrap and domain service wiring.
- `api/internal/module/` — shared module descriptor type.
- `api/internal/modules/` — thin registry for boot/codegen.
- `api/internal/database/` — cross-cutting database infrastructure.
- `api/internal/testutil/` — cross-domain test harnesses.
- `api/internal/config/` — cartographer-global control-surface levers.
- `api/internal/observability/` — dev-only profiling and diagnostics substrate.
- `api/internal/suppressions/` — in-repo `// arch:allow` marker scanning/writing substrate.
- `ui/src/components/` — shared presentation primitives.
- `ui/src/test-utils/` — cross-feature testing support.

## Not Owned

Generated, vendored, or fixture paths that are exempt from the coverage check —
no domain is required to own them.

- `packages/proto/gen/**` — generated protobuf bindings (Go / TypeScript / Python).
- `**/testdata/**` — test fixtures and golden files.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system shape and extension rules
- [`SIGNAL_LADDER.md`](SIGNAL_LADDER.md) — pluggable signal contract
- [`FLOWS.md`](FLOWS.md) — workflows and state transitions
- [`DATA.md`](DATA.md) — data ownership and storage
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — dependency contracts
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — boundary registry
- [`../internal/TESTING.md`](../internal/TESTING.md) — test strategy
