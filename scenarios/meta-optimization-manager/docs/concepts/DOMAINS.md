# Domains

## Purpose Of This Document

The durable domain map for **meta-optimization-manager**. It names the bounded contexts, what each owns, and how each maps back to a PRD operational target. The scenario is a thin, read-mostly aggregator: every domain *measures and surfaces*, none re-implements another scenario's measurement, makes the improvement, or makes a judgment call.

The cross-cutting model these domains share — the attestation contract, the entity × archetype × aspect question space, the denominator/numerator split, and the status/confidence legend — is defined once in [COVERAGE-MODEL.md](COVERAGE-MODEL.md) and not restated per domain. The sibling **Condition** axis — whether the supply `coverage` counts still works — is defined in [CONDITION-MODEL.md](CONDITION-MODEL.md). It is deliberately **not a new domain**: its population is derived from `coverage`'s live numerator, and it surfaces through `focus` as one more named gap source, exactly as the trials and agent-manager empirical lanes do. See the `focus` domain detail below.

## Domain Inventory

| Domain | Source Paths | Responsibility | Operational targets |
|--------|--------------|----------------|---------------------|
| **coverage** | `api/internal/coverage`, `api/handlers/coverage` | Read each projection's denominator (via the owner's `space --projection` verb) joined against the live registries; compute per-projection coverage + denominator-confidence; validate base-document integrity; synthesize the readiness scoreboard. | OT-P0-001, OT-P0-004, OT-P2-002 |
| **convergence** | `api/internal/convergence`, `api/handlers/convergence` | Measure the upstream generators: per-template fitness counts, gold-star generated-golden health/staleness, and the convergence trend. Surfaces numbers + candidates only. | OT-P1-002 |
| **focus** | `api/internal/focus`, `api/handlers/focus` | Maintain the gaps registry (qualitative notes/approaches) and rank gaps by impact × importance across all projections, convergence, and named empirical friction sources including program-runtime. | OT-P0-002, OT-P0-003 |
| **trials** | `api/internal/trials`, `api/handlers/trials` | Run the empirical local-model gate via agent-manager's sandboxed runner, evaluate the diff against a fixture oracle, record success/tokens/time history; track Guide-gate coverage. | OT-P1-001 |

## Domain Details

### coverage
- **Owns**: short-TTL cached coverage snapshots; the base-document-integrity findings. Does **not** own the denominators (the space docs live with their owner scenarios) or the numerators (computed live, never persisted).
- **Proto operations**: `GetStatus` (per-projection coverage + denominator-confidence + latest trial trend), `ListCells` (denominator grid rows, filterable by projection/status), `ExplainCell` (one cell: owner, status, basis × sufficiency, citations), `ValidateBaseDocs` (referenced skills/providers/phases exist; ≠1-skill Guide rows flagged).
- **API behavior**: fan-out reads to each owner's `space --projection <p> --json` verb + the live registries (`search-hub providers`, `test-genie health`, `prompt-manager graph health`, `completeness-scoring GetScore`); join denominator vs numerator; degrade gracefully (per-projection "unavailable") when a source is down.
- **CLI**: `coverage status [--json]`, `coverage cells [--projection] [--status] [--json]`, `coverage explain <cell-id> [--json]`, `coverage validate-docs [--projection] [--json]`.
- **UI**: the readiness scoreboard panel (P2).
- **Storage**: SQLite snapshot cache only.
- **Test evidence**: MOM-READINESS-001/002/003, MOM-BASEDOC-001/002.

### convergence
- **Owns**: a cached fitness-audit index (per-template counts + per-reference health verdicts + dated trend points).
- **Proto operations**: `GetConvergenceStatus` (fitness + generated-golden health summary across all templates), `GetTemplateFitness` (per-replica cost, drift-surface count, comment-only-contract count, coordinated-edit count + tier), `ListReferences` (gold-star generated-golden health/eligibility: stale-from-template, clean-on-all-tools, ≥60d stability, breadth), `GetConvergenceTrend` (per-replica-cost / coordinated-edit over dated fitness-audit records).
- **API behavior**: compute fitness counts (delegating raw code structure to code-facts / architecture-cartographer; the add/delete coordinated-edit walkthrough is the one genuinely new mechanization); read toolchain-clean results from test-genie / scenario-auditor; never re-run the toolchain. Surface numbers + flag candidates; tiering/substrate/nomination stay agentic.
- **CLI**: `convergence status [--json]`, `convergence fitness [--template] [--json]`, `convergence references [--eligibility] [--json]`, `convergence trend [--template] [--json]`.
- **UI**: the convergence-trend panel (P2).
- **Storage**: SQLite fitness-audit index.
- **Test evidence**: MOM-CONVERGENCE-001/002/003.

### focus
- **Owns**: the gaps registry — every known gap with notes/approaches/context, including cross-cutting/global gaps and explored-but-unbuilt ideas.
- **Proto operations**: `GetFocus` (ranked next-best gaps, impact × importance), `ListGaps` (filter by projection/cell/status), `GetGap` (one gap with full context), `AddGapNote` (append an explored approach — the one write verb).
- **API behavior**: aggregate gaps from coverage + convergence + the registry + the named program-runtime friction reader; rank by impact × importance; return each with its qualitative context. Sources are named and multiplexed: each degrades independently, and an unreadable source becomes a visible availability entry rather than silently removing gaps.
- **Condition source**: the registered `condition` source reads the typed Search Hub insight surface, while the `test-genie` empirical source adds per-phase condition entries from its self-health ledger. They emit findings for observed degradation and preserve source-availability gaps. By default a degraded leg does **not** change its cell's coverage status; only sustained degradation promotes to a downgrade, and dormancy never does. The signal families, the closed status vocabulary, the instrumentation-coverage reporting rule, and the ranking intent are specified in [CONDITION-MODEL.md](CONDITION-MODEL.md).
- **CLI**: `focus [--limit] [--projection] [--json]`, `gaps [--projection] [--cell] [--status] [--json]`, `gaps show <id> [--json]`, `gaps note <id> --add "<approach>"`.
- **UI**: the focus list + gaps registry panels (P2).
- **Storage**: SQLite gaps registry.
- **Test evidence**: MOM-FOCUS-001, MOM-GAPS-001/002.

### trials
- **Owns**: the trials history time-series (success-rate + tokens + wall-time + fixture-rev per run) and the per-task gate registry. Also the committed fixture corpus (`trials/fixtures/<family>/`) that defines "solved".
- **Seams**: `TaskGenerator` (Guide space → suite), `FixtureResolver` (family → committed fixture + oracle), `Runner` (agent-manager sandboxed spawn → evidence), `Evaluator` (evidence → verdict), `Repository` (history + gates). All faked in tests.
- **Proto operations**: `ListTrialTasks` (the suite, generated from the Guide space), `RunTrials` (dispatch a task/suite via agent-manager's sandboxed runner and evaluate it), `GetTrialHistory` (success/tokens/time trend), `GetTrialRun` (one run: verdict, sandbox diff, tokens, time), `GetGateCoverage` (% of Guide tasks with a live gate).
- **API behavior**: per task — resolve the fixture, reuse a recent identical run if any, reconcile MoM's declared role-only profile, dispatch the SWE task (add-feature/research/comprehend/bugfix + negative cases) through agent-manager's `run create --run-mode sandboxed`, collect the diff + metrics, then **evaluate in MoM** (deterministic fixture oracle, else agent-judge; negatives pass on correct abstention) and append metrics to history. Agent Manager owns runtime/resource selection and spawning; the verdict is MoM's.
- **CLI**: `trials list [--suite] [--json]`, `trials run [--task|--suite|--all] [--json]` (single-task default; `--all` for the full suite), `trials history [--json]`, `trials show <run-id> [--json]`, `trials coverage [--json]`.
- **UI**: the trials-trend panel (P2).
- **Storage**: SQLite trials history + gate registry.
- **Test evidence**: MOM-TRIALS-001/002/003.

## Shared Concepts

- **Attestation contract** — every answer carries the two honesty axes (basis × sufficiency); see [COVERAGE-MODEL.md](COVERAGE-MODEL.md).
- **Denominator / numerator split** — denominators (the space docs) live with their owners; numerators (coverage) are computed live, never stored here.
- **The `space --projection <p> --json` verb** — the shared read contract every denominator owner exposes; this scenario's primary upstream dependency.
- **Denominator-confidence** — every coverage number is paired with the confidence of its denominator; the scoreboard cannot imply false completeness.
- **Surfaces, does not decide** — all four domains emit numbers + candidates; the substrate / tiering / nomination / root-cause / improvement decisions stay agentic.

## Deferred Domains

- **stewardship** (skill/action lifecycle health, staleness, promotion-readiness) and **intake** (friction / run-lesson / discovery-gap routing) — deliberately *not* domains here. Their cores are judgment, and programmatizing them would weaken them; they stay agentic (see [ARCHITECTURE.md](ARCHITECTURE.md) "Intentional Deviations").
- Program-runtime is an external empirical source, not a meta-optimization-manager bounded context; its typed reader is owned by focus and degrades independently.

## Non-Domains

Shared substrate, not bounded contexts: the composition root + HTTP server (`api/internal/server`), the space-reader client and the per-scenario read clients (test-genie / graph-health / completeness-scoring / providers / code-facts / cartographer / agent-manager), storage plumbing (`api-core/storage`), and the CLI/UI translation layers. These have no business vocabulary; they serve every domain.

## Cross-References

- [ARCHITECTURE.md](ARCHITECTURE.md) — the thin-aggregator shape, boundaries, and contracts.
- [FLOWS.md](FLOWS.md) — the status / focus / trials / convergence flows.
- [DATA.md](DATA.md) — the SQLite gaps registry, trials history, and fitness-audit index.
- [COVERAGE-MODEL.md](COVERAGE-MODEL.md) — the canonical attestation contract + question-space model + legend.
- [../internal/SEAMS.md](../internal/SEAMS.md) — the read-client seams and their test doubles.
