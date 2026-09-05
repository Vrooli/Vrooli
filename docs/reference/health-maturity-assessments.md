# Health Maturity Assessments

Health provider scenarios report provider-owned local maturity through one shared validation contract:

```text
scenario-validation/v1.ScenarioValidationService.ValidateScenario
```

That RPC returns a shared layer and an optional native layer:

- `status`: the canonical provider verdict that Test Genie gates on.
- `assessment`: the mandatory `common.v1.MaturityAssessment`; all cross-scenario findings live in `assessment.findings`.
- `native_detail`: an optional provider-specific `Any` payload for the provider's own CLI/UI. Test Genie ignores it.
- `metrics`: a `common.v1.ExecutionMetrics` describing the *cost* of the validation (timing, profiling stages, best-effort CPU/RSS/GPU, and the host `CaptureEnvironment`). The whole delegated fleet now emits it (Plan 3); Test Genie persists it per run. The reliability ledger rolls it up, and a reachable provider that *drops* metrics is now a conformance hard-violation (see "Provider conformance" below).

`ExecutionMetrics` is a generic primitive (it can measure any unit of work, not just validations). The reusable collector lives in `packages/api-core/metrics`; every resource area self-describes availability through a `Reliability` enum (`RELIABLE` / `BEST_EFFORT` / `UNAVAILABLE`) so a consumer can tell "measured 0" from "couldn't measure here". Stages are the attribution carrier and nest (the profiling-span / flamegraph model): the top-level `resources` is the whole-op rollup and is *not* required to equal the sum of stages. GPU is in the schema but sampled opt-in (it shells `nvidia-smi`/`rocm-smi`), so default provider validations leave it `UNAVAILABLE`. `proto-health` is the reference provider; every delegated provider now emits real metrics end-to-end via the same 3-file recipe (a ctx-threaded `metrics.go` seam, staged validator, and a collector wired in the handler).

Implementation references:

- `packages/proto/schemas/scenario-validation/v1/validation.proto`
- `packages/proto/schemas/common/v1/maturity.proto`
- `packages/proto/schemas/common/v1/metrics.proto`
- `packages/api-core/metrics` (the reusable `Collector` primitive)
- `packages/maturity-go/assessment/assessment.go` (`BuildValidationResponse` accepts metrics; `LoadSpecFromScenario`; `RequireIdentity`)
- `scenarios/test-genie/api/internal/orchestrator/phases/validationprovider/provider.go`
- `scenarios/test-genie/api/internal/orchestrator/phases/phase_validationprovider.go`

## Provider Ownership

Each provider owns its local ladder in:

```text
scenarios/<provider>/.vrooli/test-genie.json
```

The Test Genie descriptor is the source of truth for provider phase metadata and the embedded `maturity` block: provider-local levels, finding-code mappings, semantic global impact, default severity, clean requirement, dimension, fallback policy, and recommended skill IDs. Do not duplicate the ladder in skills or scenario docs. If a local ladder is wrong, update the provider's `.vrooli/test-genie.json` descriptor and validation tests. The retired `.vrooli/maturity.json` file is no longer a fallback; if it reappears beside a descriptor, descriptor loading fails.

Providers may emit semantic global impacts such as `foundation_blocker`, `safety_blocker`, `evolvability_gap`, `hardening_gap`, `capability_gap`, `advisory`, and `unknown`, but they do not report the final global scenario maturity rung. Test Genie and status layers own aggregation.

### Capability ladders

Providers may keep the legacy single-ladder shape with top-level `levels[]`, or declare `capabilities[]` when one provider owns several independently meaningful readiness surfaces. In the multi-capability shape, each capability has its own `id`, `label`, optional `description`, and `levels[]`. Level ids only need to be unique inside one capability, so separate capabilities may both use `L0`, `L1`, and so on.

Finding mappings may set `capability_id`; when omitted, the mapping belongs to the default capability. For legacy specs the default capability is the synthetic local ladder. For multi-capability specs it is the first declared capability, so providers should set `capability_id` explicitly for clarity whenever findings span more than one ladder. A finding's `local_level_impact` is validated against the owning capability's levels.

Validation responses remain backward-compatible: `assessment.local` is still populated as the provider-local rollup. When `capabilities[]` are present, each capability gets its own current/next levels, summaries, blockers, clean state, and unknown count, while `assessment.local` rolls up to the deterministic focus capability and sums clean/unknown state across capabilities. `highest_priority_capability` points at the same focus capability.

Capability priority is provider-agnostic and deterministic. The shared scorer sorts capabilities by required/blocking state before advisory debt before clean capability, then lower current level, higher severity counts (`BLOCKER`, `ERROR`, `WARNING`, `INFO`), stronger global impact (`foundation_blocker`, `safety_blocker`, `evolvability_gap`, `hardening_gap`, `capability_gap`, `advisory`, `unknown`), fixable finding count, and finally provider declaration order. The emitted `capabilities[]` list stays in provider declaration order for stable reports; each capability's `priority_rank` records the sorted focus order.

The shared report renderer preserves legacy single-ladder output when `assessment.capabilities[]` is empty. When capability assessments are present, the human report adds one line per capability with current level, status label/name, blocking/debt counts, current summary, next unlock, and a maximum-maturity line when no next level exists. Findings are grouped by capability and local level. JSON consumers should read the proto fields directly rather than parsing these report strings.

`cli-health` uses this shape to separate CLI manifest contract, proto bindings, runtime surface, measures metadata, entrypoint structure, discovery readiness, and `command_architecture` — the last classifies how far a scenario's commands have converged on cli-core renderer-separated primitives versus bespoke output-format control flow. A declared `architecture` primitive is only *verified* (top rung) when it matches a committed **static** primitive-evidence artifact (`.vrooli/generated/cli-primitive-evidence.json`) generated from the scenario's own registration metadata — cli-health never executes a scenario's commands to collect evidence, and manifest text alone never reaches the top rung; see `scenarios/cli-health/docs/reference/cli-architecture-maturity.md`. `ui-health` uses this shape to separate manifest contract, interop, freshness, runtime render, project standards, and `pwa_native_readiness`. Behavioral PWA/native readiness lives in `ui-health`: manifest install contract, launch/scope safety, service-worker/offline shell presence, and optional platform manifest fields when declared. Knowledge Observatory uses the same shape as the documentation-health reference adopter, separating documentation contract, required docs, append-log integrity, content quality, link health, reference integrity, and manifest coverage. `brand-manager` owns brand identity, public install assets, icon quality, theme color consistency, and social preview surfaces; it should cross-reference `ui-health` rather than duplicate runtime/offline rules.

`clean_requirement` is orthogonal to severity:

| Value | Local rung behavior | Debt behavior |
|---|---|---|
| `required` | Gates the provider-local rung even when severity is WARNING or INFO | Counts as actionable debt until resolved |
| `advisory` | Does not gate unless severity is ERROR or BLOCKER | WARNING/INFO remains visible but non-gating |
| `uncheckable` | Does not gate | Excluded from debt and surfaced through `local.unknown_count` |

Severity still owns phase pass/fail. ERROR and BLOCKER findings fail the provider phase; WARNING and INFO findings do not. `local.clean` reports whether zero REQUIRED findings remain, so agents can continue convergence after a phase has passed.

### Target scoping

A run is about exactly one target. Two rules keep a ladder a statement about *that* target rather than about whatever else the provider happened to inspect.

**A finding scores only its own target.** `AssessmentFinding.subject` names the target a finding is about; an empty subject means the run's own target. The engine scores a capability only from findings whose subject matches the run target — kind *and* id, since `resource:ollama` is not `scenario:ollama`. Callers that pass no `BuildInput.Target` get the implicit scenario target derived from `BuildInput.Scenario`, so this is correct by default for every provider that never opts in.

This exists because a provider may report on more than its own target. `storage-manager` validates every resource, tool, and safeguard inside its own scenario run; before scoping, one safeguard that declared no `storage.entries` pulled `storage-manager`'s `declaration_accountability` from L3 to L0 while its own storage was fully governed. Fleet state reaches the verdict through an explicit aggregate finding (`STORAGE_OWNER_GATE_FAILED`), not by leaking into the ladder.

**A capability scores only where it is meaningful.** `capabilities[].appliesTo` lists the target kinds a capability is about. A capability that does not apply to the run's target kind is omitted from the standing entirely, rather than reported at a rung the target can neither satisfy nor fail. An empty `appliesTo` applies everywhere, and an unresolved target kind never narrows — an unknown kind keeps every capability, because losing coverage silently is worse than scoring something inapplicable.

This is the declaration half of the same rule the provider enforces in its own analyzers (`quality-health`'s `Applies.MatchesTargetKind`, `storage-manager`'s `Kinds()`, `knowledge-observatory`'s `checkScope`). A `package` target has no Makefile and no `coverage/testing.json`, so a scenario-contract capability must not score it.

Two deliberate non-behaviors:

- **Excluded findings are still reported.** Scoping changes what is *scored*, never what is *visible*. An out-of-scope finding stays in `assessment.findings` and in the severity counts, so a provider bug shows up as a failing run rather than being silently neutralized into a false pass. `provider-conformance` reports the provider through `PROVIDER_SUBJECT_OUTSIDE_DECLARED_KINDS` and `PROVIDER_CAPABILITY_TARGET_COVERAGE_GAP`.
- **Scoping never empties a standing.** If every capability excludes the run's kind, the full set is kept: that condition is a mis-declared spec for conformance to report, not a reason to emit an unfalsifiable empty ladder.

## Human Workflow

Use the provider CLI without `--json` when investigating or fixing a scenario. The CLI should render the same shared `assessment` that the RPC returned, plus any provider-native detail it unpacks from `native_detail`.

```bash
cli-health validate scenario <target>
ui-health validate scenario <target>
quality-health audit run <target>
unit-health validate scenario <target> --execution
security-health validate scenario <target>
measures-health validate scenario <target>
proto-health validate scenario <target>
scenario-dependency-analyzer health <target>
architecture-cartographer audit run <target>
knowledge-observatory docs health <target>
tidiness-manager scan <target> --type tidiness
search-hub maturity scan
```

Search Hub's target maturity scan is not a Test Genie fleet-health rollup. It
discovers scenarios with `.vrooli/search.json`, calls Search Hub's shared
ScenarioValidationService, and runs full search validation by default, including
stored eval evidence and live corpus label checks. Use `--fast` only for a quick
inventory that intentionally skips live retrieval proof.

## Automation Workflow

Test Genie uses one generic provider runner for all adopted health providers. Inline descriptors call `ScenarioValidationService.ValidateScenario`; a descriptor that explicitly declares `deliveryMode: "durable-run"` calls `DurableValidationRunService.StartValidationRun`, persists the parent/child reference immediately, and waits once for the provider-owned terminal response. The deterministic parent-run-plus-phase key makes a restart reconcile the same child run rather than duplicate execution; explicit parent cancellation propagates `AbortValidationRun`, while client disconnects do not. In either mode, the runner validates the returned `assessment`, maps `assessment.findings` into `ArchitectureFinding`s, and writes the phase pointer. It also carries `local.current_level`, `local.next_level`, `local.clean`, and `local.unknown_count` into the phase summary. When a provider emits `assessment.capabilities[]`, Test Genie includes compact capability summaries and `highest_priority_capability` in the phase summary and observations, while keeping pass/fail based on shared `status` and severity. It does not shell out to provider CLIs or parse provider-native payloads.

Use `test-genie provider-contract check` to verify a single live provider contract after rebuilding or restarting it:

```bash
test-genie provider-contract check contracts <target>
test-genie provider-contract check cli-health <target> --json
```

The command probes adopted providers through the shared RPC and validates both `status` and `assessment`.

Use `test-genie provider-contract scan` for a fleet-wide adoption sweep across every delegated phase in the effective descriptor-backed registry:

```bash
test-genie provider-contract scan --json
test-genie provider-contract scan --target <fixture-scenario> --timeout 30s
```

For each provider the scan probes (default target `test-genie`, `include_execution=false`) and scores five dimensions: `reachable`, `contract_valid` (passes `assessment.ValidateAssessment` + non-unspecified status), `identity_ok` (provider/phase match the effective registry via `assessment.RequireIdentity`), `spec_valid` (`.vrooli/test-genie.json` loads, its embedded `maturity` block validates, and descriptor/provider/phase identity is coherent), and `metrics_adopted` (the response's metrics were persisted in terminal execution history). `adoption_score` is the fraction of dimensions satisfied. As of Plan 3, `metrics_adopted` is **required, not advisory**: the fleet has adopted it, so a *reachable* provider whose metrics are absent from durable history is now a hard violation. The command exits non-zero when a provider is mis-specified (`spec_valid=false`) or — while reachable — breaks the contract, mismatches identity, or has dropped metrics; unreachability stays a liveness signal that does not fail the gate. The hard-violation rule is one SSOT predicate, `selfhealth.IsHardViolation`, shared by the API conformance method and both CLIs (`provider-contract scan`, `test-genie health`).

## Failure Taxonomy

| Condition | Test Genie failure class | Operator action |
|---|---|---|
| Provider API cannot be discovered or reached | `provider_unavailable` / `missing_dependency` according to descriptor unavailable policy | Start or restart the provider through lifecycle; never run provider binaries directly |
| Shared RPC returns an unspecified or malformed status | `maturity_contract` | Restart the provider to rule out stale service state, then fix the provider validation handler |
| Shared RPC omits or malforms `assessment` | `maturity_contract` | Fix provider assessment construction; do not soften Test Genie validation |
| Shared RPC returns `VALIDATION_STATUS_ERROR` | `system` | Inspect provider logs and fix the provider execution failure |
| Shared RPC returns `VALIDATION_STATUS_FAILED` or ERROR findings | `test_failure` | Use the provider's normal human CLI output to fix target-scenario defects |
| Best-effort or advisory provider is unreachable | advisory skip, partial, or non-failing skip according to descriptor policy | Start the provider when that phase should run |
| `TEST_GENIE_SKIP_<PHASE>=1` is set | explicit skip | Remove the skip toggle when the phase should run |

## Test Genie Provider Inventory

Adopted provider phases use Connect-RPC `ScenarioValidationService.ValidateScenario`. Test Genie reads only `status` and `assessment`; native details are provider-owned.

| Test Genie phase | Provider scenario | Transport | Native detail type | Finding source |
|---|---|---|---|---|
| `structure` | `structure-health` | Connect-RPC `ScenarioValidationService` (all nine target kinds) | none | `FINDING_SOURCE_STRUCTURE` |
| `business` | `business-health` | Connect-RPC `ScenarioValidationService` | `BusinessContractReport` | `FINDING_SOURCE_BUSINESS` |
| `contracts` | `cli-health` | Connect-RPC `ScenarioValidationService` | none | `FINDING_SOURCE_CLI` |
| `ui-health` | `ui-health` | Connect-RPC `ScenarioValidationService` | none | `FINDING_SOURCE_UI` |
| `api` | `api-health` | Connect-RPC `ScenarioValidationService` | none | dimension-routed API readiness findings |
| `architecture` | `architecture-cartographer` | Connect-RPC `ScenarioValidationService` | `AuditRunResponse` | `FINDING_SOURCE_ARCHITECTURE` |
| `dependencies` | `scenario-dependency-analyzer` | Connect-RPC `ScenarioValidationService` | `DependencyHealthResponse` | `FINDING_SOURCE_DEPENDENCY` |
| `quality` | `quality-health` | Connect-RPC `ScenarioValidationService` | `AuditQualityResponse` | `FINDING_SOURCE_STANDARDS` |
| `docs` | `knowledge-observatory` | Connect-RPC `ScenarioValidationService` | `DocHealthResponse` | `FINDING_SOURCE_DOCS` |
| `performance` | `performance-health` | Connect-RPC `ScenarioValidationService` | none | none; phase gates on provider status and budgets |
| `tidiness` | `tidiness-manager` | Connect-RPC `ScenarioValidationService` | `TidinessScanResponse` | `FINDING_SOURCE_TIDINESS` |
| `security` | `security-health` | Connect-RPC `ScenarioValidationService` | none | `FINDING_SOURCE_SECURITY` |
| `measures` | `measures-health` | Connect-RPC `ScenarioValidationService` | `ScenarioCoverageReport` | `FINDING_SOURCE_MEASURES` |
| `proto` | `proto-health` | Connect-RPC `ScenarioValidationService` | none | `FINDING_SOURCE_PROTO` |
| `unit` | `unit-health` | Connect-RPC `ScenarioValidationService` | unit-health validation response | dimension-routed, including `FINDING_SOURCE_COVERAGE` |
| `storage` | `storage-manager` | Connect-RPC `ScenarioValidationService` | none | `FINDING_SOURCE_STORAGE` |
| `branding` | `brand-manager` | Connect-RPC `ScenarioValidationService` | none | `FINDING_SOURCE_BRANDING` |
| `search` | `search-hub` | Connect-RPC `ScenarioValidationService` | none | descriptor-defined search maturity findings |
| `provider-conformance` | `test-genie` | Connect-RPC `ScenarioValidationService` | none | descriptor-defined provider-conformance findings |

Provider-owned Test Genie descriptors live at `scenarios/<provider>/.vrooli/test-genie.json`; the provider maturity spec is embedded in that descriptor's `maturity` block. Some providers still use the single-ladder maturity shape; others use capability ladders. Do not infer capability support from the Test Genie phase name. Check the provider descriptor, or use `test-genie provider-contract scan --json`, when you need the current contract shape.

All default provider-health phases are descriptor-backed. If a future true in-process Test Genie phase exists, it must be documented as an explicit exception rather than hidden in provider metadata.

## Test Genie Self-Health

Test Genie also reports on **its own** reliability and on fleet conformance, turning the run data it already persists into one aggregated, queryable surface. The endpoint is `RunsService.GetSelfHealth` (typed Connect-RPC, read-only) and the CLI wrapper is `test-genie health` (`--json` for the raw proto payload; `--trend` to include the persisted snapshot series). The live read is compute-on-read; a background sweeper additionally persists a timestamped history so trend deltas are answerable (see "Trend history" below).

The payload stitches three reads:

- **Catalog summary** — total phases, delegated vs. native, and per-phase provider + finding source (from the in-process phase catalog).
- **Provider conformance** — the same scorecard `provider-contract scan` produces (reachable, contract-valid, identity, descriptor-embedded maturity valid, persisted `metrics_adopted`, concurrency declaration, adoption score), probed **live and time-boxed** (fixture target `test-genie`, `include_execution=false`, parallel). The response marks `conformance_freshness="live"`; pass `--skip-conformance` to omit the live scan. The conformance core is shared between the endpoint and the CLI `scan` verb (`internal/selfhealth`).
- **Reliability ledger** — over a recent window (default 30 days; `--window-days N`): suite availability and the run-level terminal-outcome histogram (`passed`/`failed`/`errored`/`aborted`/`timeout`), plus per-phase and per-provider availability %, failure rate, degraded counts, skip-reason + classification histograms, duration p50/p95/min/max/avg, and worst-scenarios-per-phase.
- **Autofix coverage** — per provider, the spec-derived autofix declaration rollup (see "Autofix coverage lens" below). Advisory; carried on each conformance scorecard (`autofix`).

Availability is denominator-correct because **every** terminal run — including catastrophic aborts/timeouts/engine-errors that produce no result — persists a `suite_executions` row with a `terminal_outcome` classification. The ledger aggregates that table (engine-neutral `SuiteExecutionRepository`); the per-scenario run index stays the freshness/git source, not the ledger source.

### Trend history

A background **self-health sweeper** persists timestamped rollups of the ledger + conformance summary into a per-domain `selfhealth_snapshots` table (single writer; the read path never writes). It is digest-deduplicated (identical content → no new row), env-tuned (`TEST_GENIE_SELFHEALTH_SWEEP_INTERVAL` default 1h, `…_DISABLED`, `…_START_JITTER`), and disableable. `GetSelfHealth` fills the ledger's `captured_at` and a `trend` delta (current vs the last *differing* snapshot); `--trend`/`include_trend` additionally returns the windowed `trend_series`. The compute-on-read path is unchanged when the sweeper is disabled (no `captured_at`/`trend`). This answers "is our test infrastructure getting better or worse?" over time.

Implementation references:

- `packages/proto/schemas/test-genie/v1/runs/runs.proto` (`GetSelfHealth`, `SelfHealth`, `ReliabilityLedger`, `TrendDelta`, `SelfHealthTrendPoint`)
- `scenarios/test-genie/api/internal/selfhealth/` (ledger + shared conformance core + `IsHardViolation` SSOT)
- `scenarios/test-genie/api/internal/selfhealthsnapshots/` (persisted snapshot store + sweeper)
- `scenarios/test-genie/api/internal/execution/` (`terminal_outcome`, aggregation queries, brownfield migration)
- `scenarios/test-genie/cli/health/` (the `test-genie health` verb)

The meta-optimization `toolchain-validator` member consumes this each heartbeat (`test-genie health --json --trend`), records a snapshot to `self-health/test-genie/<date>`, and — **baseline-gated** ("no baseline → no proposal") — raises an owned `toolchain-violation` decision when self-health regresses past a tunable threshold (availability drop, a new conformance hard-violation, or a metrics-adoption regression). Thresholds live in the member HEARTBEAT contract (human-tunable), never in Test Genie code; decision-raising is advisory to the team and never gates a build.

### Fleet health (fleet backbone)

Where self-health is Test Genie looking at *itself*, **fleet health** is the same compute-on-read idea applied to the *whole fleet*: `RunsService.GetFleetHealth` (CLI `test-genie fleet status`, `--json`/`--roster`) aggregates the **stored runs of every scenario** into per-scenario rollups (runs, availability, failure rate, issue counts, newest run age + outcome), a most-errored-first ranking, fleet-wide finding-source clustering, and — with a roster — the scenarios with **no run in the window**. It is an aggregation over runs that already executed; it does **not** launch a fleet run, and every datum is **as-of stamped** (rollup `captured_at`; per-scenario `last_run_at`/`age_days`) so stale data can never read as fresh.

The **write side** that keeps those stored runs fresh is a **default-OFF, priority-weighted background scheduler** (`TEST_GENIE_FLEET_SCHEDULER_ENABLED`), mirroring the sweeper pattern: each cycle selects the highest-priority, stalest scenarios from `scenario-completeness-scoring score list` (importance × score gap, scaled by test age; never-tested scenarios pulled hardest) and launches their suites through the durable run manager, **respecting** the one-in-progress-per-scenario invariant and bounded by explicit concurrency + per-cycle + wall-clock budgets. The priority query is itself enabled by SCS persisting **scenario-level test recency** (`ScoreRow.last_run_at`/`last_status`), so importance + staleness + last status are answerable fleet-wide in one read.

Implementation references:

- `packages/proto/schemas/test-genie/v1/runs/runs.proto` (`GetFleetHealth`, `FleetHealth`, `FleetScenarioHealth`, `FleetFindingSource`)
- `scenarios/test-genie/api/internal/selfhealth/fleet.go` (fleet ledger, compute-on-read with as-of stamps)
- `scenarios/test-genie/api/internal/fleetscheduler/` (default-OFF priority scheduler + launcher + SCS priority source)
- `scenarios/test-genie/api/internal/execution/aggregation.go` (`AggregateScenarioRuns`)
- `scenarios/scenario-completeness-scoring/api/internal/scoring/` (`last_run_at`/`last_status` persistence + `Migrate`)
- `scenarios/test-genie/cli/fleet/` (the `test-genie fleet status` verb)

## Autofix coverage lens

Each provider finding mapping in a descriptor `maturity` block may declare **how remediable its category is**, so the system can measure — and prioritize closing — the gap between findings that *could* be auto-fixed and fixers that *exist*. The declaration is two orthogonal fields plus a justification, deliberately **not** a flat two-state flag:

- **`fix_class`** (intent — can this category EVER be auto-remediated?): `auto` (deterministic in-process fixer) · `external` (delegated to another tool/scenario) · `manual` (inherently needs human judgment — the honest "never autofixable").
- **`fixer_status`** (only for `auto`/`external` — does the logic exist yet?): `implemented` · `pending`. **Defaults to `pending`** so a newly-declared fixable finding is a visible gap until a fixer is wired.
- **`reason`** — **required when `fix_class` is `manual`**: excluding a finding from the fixable universe is an explicit, reviewable justification, not a silent escape hatch (enforced by `ValidateSpec`).

`ValidateSpec` enforces the vocabulary, the `manual ⇒ reason` rule, and that `fixer_status` is only set on `auto`/`external`. **Absent `fix_class` derives to `manual`** (the conservative default — it never inflates the fixable universe) and an absent `fixer_status` on a fixable class derives to `pending`. Existing specs predating these fields stay valid; declaration *completeness* is an advisory conformance dimension, not a hard `ValidateSpec` gate.

Derived per-provider counts (on each conformance scorecard's `autofix`, and summarized by `test-genie health`):

- **fixable universe** = `auto` + `external`; **implemented**; **pending** (the gap); **manual**.
- **declaration_complete** — every finding carries an explicit `fix_class`. This is the advisory *dimension*; it is **not** the coverage ratio.
- **implementation_rate** = `implemented / (implemented + pending)` — informational only, never a gate. `pending` stays in the denominator, so the only honest way the rate rises is `pending → implemented`.

**Anti-gaming (why the model is shaped this way):** the surfaced optimization signal is the **absolute `pending` gap count**, not a percentage — there is no ratio to inflate. Shrinking the gap requires either *building a fixer* (`pending → implemented`) or an explicit, reason-bearing, baseline-anchored reclassification to `manual`. The distinct `pending` state is mandatory; collapsing it into `manual` would reward not building fixers. Runtime `autofix_available` (a fixer handled *this* instance) that disagrees with the declared class (`manual`, or `pending`) surfaces a **contract warning** via `assessment.ConsistencyWarnings` — advisory, not a hard failure.

Implementation references: `packages/maturity-go/assessment` (`FixClass`/`FixerStatus`, `ComputeAutofixCoverage`, `ConsistencyWarnings`, `ValidateSpec`); `scenarios/test-genie/api/internal/selfhealth/conformance.go` (per-provider rollup); `runs.proto` `AutofixCoverage`.

## Shared Fix RPC (deterministic remediation)

The coverage lens *measures* remediability; the shared Fix RPC *performs* it. `scenario-validation/v1.ScenarioValidationService` carries two additive, dry-run-aware methods beside `ValidateScenario`:

- **`PreviewFix(FixRequest) → FixResponse`** — reports the deterministic edits a provider could apply (`{scenario, path, rule_ids}` → `{candidates[], messages[]}`), writing nothing.
- **`ApplyFix(FixRequest) → FixResponse`** — applies the provider's deterministic edits and reports what changed (`applied=true`). Apply is never implicit; a caller reaches it only on an explicit `--apply`.

`FixCandidate` mirrors the shared `maturity-go/autofix.Candidate` shape (`rule_id`, `file_path`, `description`, `before`, `after`, `applied`). Providers built on the shared `autofix.Registry` mount the RPC with one line each (`Registry.PreviewFixResponse` / `ApplyFixResponse`); the `maturity-go/autofix.BuildFixResponse` helper stamps the canonical "no auto-fixable findings" message for empty results. Providers that ship **no** fixer simply leave the RPC unimplemented — consumers treat `Unimplemented` as "no deterministic fixer" and skip, so an absent fixer is a visible gap (the `pending` count), never a contract violation.

Test Genie aggregates this across a scenario's delegated providers: **`test-genie fix <scenario> --deterministic [--apply] [--rule …] [--provider …]`** fans out to each provider's Fix RPC, merges candidates into one report, and is dry-run by default (`internal/deterministicfix`). It is distinct from the agent-based `test-genie fix` path (`internal/fix`), which stays the default for non-deterministic findings.

**Fleet remediation** (`test-genie fix --fleet [--apply] [--max-scenarios N] [--concurrency N] [--json]`) composes that per-scenario aggregate with the Stage 3 fleet backbone: it walks the **priority-ordered fleet** (from `scenario-completeness-scoring score list`, the same ranking the background scheduler re-tests by), calls the deterministic aggregate per scenario (dry-run unless `--apply`), and emits a per-scenario remediation report (candidates found/applied, `clean`, or `error`/unreachable — a per-scenario failure never aborts the walk). It is bounded by `--max-scenarios` (count) and `--concurrency` (forced to 1 under `--apply` so writes stay auditable). The `--json` report is consumable by the meta-optimization loop, which can raise "N% of fleet scenarios have autofixable findings; run `test-genie fix --fleet --apply`" alongside the autofix-coverage `pending` gap.

> **Converged-fixer notes (debt):** knowledge-observatory's remediations are file *moves*, so its candidates carry the move in `description` with empty `before`/`after`.

## Adding A Provider

To make a future health-style scenario consumable by Test Genie:

1. Define a provider-owned Test Genie descriptor in `scenarios/<provider>/.vrooli/test-genie.json`. Put phase identity, policy, applicability, runnability, docs metadata, and the embedded `maturity` block there. Load the maturity block with `assessment.LoadSpecFromScenario` rather than a hand-rolled loader. Declare each finding's `fix_class` (and `fixer_status`/`reason`) so the autofix coverage lens reflects your provider honestly (see "Autofix coverage lens").
2. Implement `scenario-validation/v1.ScenarioValidationService.ValidateScenario`.
3. Populate `status` and a valid `common.v1.MaturityAssessment`.
4. Put rich provider-only detail in `native_detail` rather than extending the shared response.
5. (Optional) If your provider can deterministically remediate findings, mount the shared Fix RPC (`PreviewFix`/`ApplyFix`). Built on the shared `autofix.Registry`, this is `Registry.PreviewFixResponse`/`ApplyFixResponse` (see "Shared Fix RPC"); leave it unimplemented if you have no fixer.
6. (Optional) Instrument the validation with `packages/api-core/metrics` and pass `collector.Stop()` into `assessment.BuildValidationResponse` to emit `metrics`. Supply a `CaptureEnvironment` via the `vrooli-cli-go` host-inventory mapper. See `proto-health` for the reference wiring.
7. Render CLI/UI output from the same shared assessment and provider-native detail.
8. A provider may opt into `deliveryMode: "durable-run"` only after it owns a persistent ledger, idempotent Start/Get/Wait/Abort lifecycle, explicit cancellation and restart-recovery policy, ETA and terminal shared responses, opaque artifact references, and passes Test Genie's generic durable conformance probe plus an end-to-end delegated run. Existing providers remain inline until those conditions are met.
8. Verify with `test-genie provider-contract scan` (and `check` for a single provider), inspect applicability with `test-genie phases applicability <target> --phase <phase> --json`, inspect planning with `test-genie phases plan <target> --json`, and add phase docs.

## Related

- [../scenarios/VALIDATION.md](../scenarios/VALIDATION.md)
- [build-and-validation.md](build-and-validation.md)
- [../../packages/maturity-go/README.md](../../packages/maturity-go/README.md)
