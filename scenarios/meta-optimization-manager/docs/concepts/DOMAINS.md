# Domains

## Purpose Of This Document

The durable domain map for **meta-optimization-manager**. It names the bounded contexts, what each owns, and how each maps back to a PRD operational target. The scenario is a thin, read-mostly aggregator: every domain *measures and surfaces*, none re-implements another scenario's measurement, makes the improvement, or makes a judgment call.

The cross-cutting model these domains share — the attestation contract, the entity × archetype × aspect question space, the denominator/numerator split, and the status/confidence legend — is defined once in [COVERAGE-MODEL.md](COVERAGE-MODEL.md) and not restated per domain.

## Domain Inventory

| Domain | Responsibility | Operational targets |
|--------|----------------|---------------------|
| **coverage** | Read each projection's denominator (via the owner's `space --projection` verb) joined against the live registries; compute per-projection coverage + denominator-confidence; validate base-document integrity; synthesize the readiness scoreboard. | OT-P0-001, OT-P0-004, OT-P2-002 |
| **convergence** | Measure the upstream generators: per-template fitness counts, gold-star reference health/staleness, and the convergence trend. Surfaces numbers + candidates only. | OT-P1-002 |
| **focus** | Maintain the gaps registry (qualitative notes/approaches) and rank gaps by impact × importance across all projections + convergence. | OT-P0-002, OT-P0-003 |
| **trials** | Run the empirical local-model gate via agent-manager + workspace-sandbox; record success/tokens/time history; track Guide-gate coverage. | OT-P1-001 |

## Domain Details

### coverage
- **Owns**: short-TTL cached coverage snapshots; the base-document-integrity findings. Does **not** own the denominators (the space docs live with their owner scenarios) or the numerators (computed live, never persisted).
- **Proto operations**: `GetStatus` (per-projection coverage + denominator-confidence + latest trial trend), `ValidateBaseDocs` (referenced skills/providers/phases exist; ≠1-skill Guide rows flagged).
- **API behavior**: fan-out reads to each owner's `space --projection <p> --json` verb + the live registries (`search-hub providers`, `test-genie health`, `prompt-manager graph health`, `completeness-scoring GetScore`); join denominator vs numerator; degrade gracefully (per-projection "unavailable") when a source is down.
- **CLI**: `meta-optimization-manager coverage status [--json]`, `coverage validate-docs [--projection] [--json]`.
- **UI**: the readiness scoreboard panel (P2).
- **Storage**: SQLite snapshot cache only.
- **Test evidence**: MOM-READINESS-001/002, MOM-BASEDOC-001/002.

### convergence
- **Owns**: a cached fitness-audit index (per-template counts + per-reference health verdicts + dated trend points).
- **Proto operations**: `GetTemplateFitness` (per-replica cost, drift-surface count, comment-only-contract count, coordinated-edit count), `GetReferenceHealth` (stale-from-template, clean-on-all-tools, ≥60d stability, breadth), `GetConvergenceTrend`.
- **API behavior**: compute fitness counts (delegating raw code structure to code-facts / architecture-cartographer; the add/delete coordinated-edit walkthrough is the one genuinely new mechanization); read toolchain-clean results from test-genie / scenario-auditor; never re-run the toolchain. Surface numbers + flag candidates; tiering/substrate/nomination stay agentic.
- **CLI**: `convergence fitness [--template] [--json]`, `convergence reference-health [--json]`, `convergence trend [--json]`.
- **UI**: the convergence-trend panel (P2).
- **Storage**: SQLite fitness-audit index.
- **Test evidence**: MOM-CONVERGENCE-001/002.

### focus
- **Owns**: the gaps registry — every known gap with notes/approaches/context, including cross-cutting/global gaps and explored-but-unbuilt ideas.
- **Proto operations**: `ListGaps` (filter by projection/cell), `GetFocus` (ranked next-best gaps).
- **API behavior**: aggregate gaps from coverage + convergence + the registry; rank by impact × importance; return each with its qualitative context.
- **CLI**: `focus [--json]`, `gaps [--projection] [--cell] [--json]`.
- **UI**: the focus list + gaps registry panels (P2).
- **Storage**: SQLite gaps registry.
- **Test evidence**: MOM-FOCUS-001, MOM-GAPS-001.

### trials
- **Owns**: the trials history time-series (success-rate + tokens + wall-time per run) and the per-task gate registry.
- **Proto operations**: `RunTrials` (dispatch a task suite), `GetTrialHistory`, `GetGateCoverage` (% of Guide tasks with a live gate).
- **API behavior**: dispatch SWE tasks (add-feature/research/comprehend/bugfix + negative cases) through agent-manager (runner=opencode + local-model config) inside workspace-sandbox; evaluate by deterministic checks where possible and an agent-judge otherwise; append metrics to history.
- **CLI**: `trials run [--suite] [--json]`, `trials history [--json]`, `trials coverage [--json]`.
- **UI**: the trials-trend panel (P2).
- **Storage**: SQLite trials history + gate registry.
- **Test evidence**: MOM-TRIALS-001/002.

## Shared Concepts

- **Attestation contract** — every answer carries the two honesty axes (basis × sufficiency); see [COVERAGE-MODEL.md](COVERAGE-MODEL.md).
- **Denominator / numerator split** — denominators (the space docs) live with their owners; numerators (coverage) are computed live, never stored here.
- **The `space --projection <p> --json` verb** — the shared read contract every denominator owner exposes; this scenario's primary upstream dependency.
- **Denominator-confidence** — every coverage number is paired with the confidence of its denominator; the scoreboard cannot imply false completeness.
- **Surfaces, does not decide** — all four domains emit numbers + candidates; the substrate / tiering / nomination / root-cause / improvement decisions stay agentic.

## Deferred Domains

- **stewardship** (skill/action lifecycle health, staleness, promotion-readiness) and **intake** (friction / run-lesson / discovery-gap routing) — deliberately *not* domains here. Their cores are judgment, and programmatizing them would weaken them; they stay agentic (see [ARCHITECTURE.md](ARCHITECTURE.md) "Intentional Deviations").
- The example `notes` domain shipped by the template is deferred for removal via `vrooli scenario detemplate` once the first real domain is green.

## Non-Domains

Shared substrate, not bounded contexts: the composition root + HTTP server (`api/internal/server`), the space-reader client and the per-scenario read clients (test-genie / graph-health / completeness-scoring / providers / code-facts / cartographer / agent-manager / workspace-sandbox), storage plumbing (`api-core/storage`), and the CLI/UI translation layers. These have no business vocabulary; they serve every domain.

## Cross-References

- [ARCHITECTURE.md](ARCHITECTURE.md) — the thin-aggregator shape, boundaries, and contracts.
- [FLOWS.md](FLOWS.md) — the status / focus / trials / convergence flows.
- [DATA.md](DATA.md) — the SQLite gaps registry, trials history, and fitness-audit index.
- [COVERAGE-MODEL.md](COVERAGE-MODEL.md) — the canonical attestation contract + question-space model + legend.
- [../internal/SEAMS.md](../internal/SEAMS.md) — the read-client seams and their test doubles.
