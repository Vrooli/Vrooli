# Performance — Search Hub

This document records performance budgets, current measurements, known
constraints, and regression procedures.

## Purpose Of This Document

Use this document to answer:

- What performance matters for this scenario?
- What budgets or thresholds apply?
- How are measurements captured?
- What performance risks remain?

## Budgets

| Surface | Budget | Measurement | Status |
|---|---|---|---|
| UI build | 5-10 minutes accepted for current Vite module graph | lifecycle/test-genie build logs | inherited |
| API health | responsive under lifecycle health timeout | `/health` check | active |
| UI health | responsive under lifecycle health timeout | `/health` check | active |
| Federated fan-out | `ceil(active providers / concurrency) × provider timeout < query timeout` | deterministic budget invariant test | active |
| Address resolution | one lookup per scenario/port within the cache TTL | resolver fake-clock/cache tests | active |

## Current Measurements

| Measurement | Value | Source | Date |
|---|---|---|---|
| Query budget | 34 active leaves at concurrency 8 and 4s provider timeout fit in 20s of a 25s query budget | `internal/routing/fanout_budget_test.go` | 2026-08-12 |
| Address cache | successful entries live for 2s by default; failures invalidate immediately | `packages/api-core/discovery` fake-clock tests | 2026-08-12 |
| Live p50/p95 | 1,344 / 6,825 ms | `search-hub metrics federated-latency --window last_7d --json` | 2026-08-15 |
| Degraded-query rate | 5,629 / 17,735 (31.74%) | `search-hub metrics degraded-query-rate --window last_7d --json` | 2026-08-15 |
| Federation state | 31 providers; 0 demoted; 1 quality-withheld | `search-hub federation status --json` | 2026-08-12 |

## Honest Comparison

The three columns below preserve the audit, restored-runtime baseline, and
current snapshot. They are directional evidence, not a claim of a clean
single-variable benchmark: the final snapshot includes later owner tuning,
expanded suite registration, and the current degraded resource state.

| Signal | Pre-restoration audit | Honest baseline | Final/current snapshot | Direction and reason |
|---|---:|---:|---:|---|
| Answer coverage | 17% (6/36) | 25% (9/36) | 22.22% (8/36) | Down from the restored baseline; current coverage is the live NOW projection and remains denominator-confident only for its declared model. |
| Demoted providers | 18/34 | 0 observed | 0/31 | Improved and held; restart plus successful probes cleared transport-era demotions. |
| Aggregate eval met | 0/235 | 191/235 (222 graded) | 190/237 (222 graded) | Slightly down in met count, but not a like-for-like denominator: two suites were added and the graded denominator stayed constant. |
| 7-day p95 latency | 19,594 ms | 20,203 ms | 20,203 ms | Flat versus the restored baseline; no latency win is claimed while the provider fleet remains degraded. |
| 7-day zero/degraded rate | 31% | 31.51% (328/1,041) | 31.54% (334/1,059) | Flat within measurement drift; the additional queries are retained rather than hidden. |
| Blocking maturity findings | 23 | 18 | 19 | Improved versus the phase-1 audit and remains below the `<23` certification threshold; the remaining findings are recorded debt, not hidden. |

## Attenuation audit baseline — 2026-08-13

This is the immutable pre-change baseline for the Search Hub end-to-end
retrieval-trust plan. The source fingerprint was Git SHA `6cace1549d5749f178e79d16c9d1d53ae8d7c53d`; the durable collection is
`search-hub-end-to-end-retrieval-trust-routing-attribution-baseline`. Raw
responses are retained in
`tmp/baseline-2026-08-13/` so the closing measurements can be repeated from
the same command surfaces.

### Quality headline

The latest saved run for each tier and suite was selected by `created_at`.
Unavailable cases are not included in `graded`; this is why the live
denominators differ from the earlier audit in the plan problem statement.

| Tier | Met | Graded | Rate | Source |
|---|---:|---:|---:|---|
| `provider_direct` | 167 | 195 | 85.64% | `search-hub evals list --json`; `search-hub evals runs <suite_id> --json --limit 12` |
| `federated` | 7 | 51 | 13.73% | same commands; aggregate in `tmp/baseline-2026-08-13/headline-aggregates.json` |

### Fleet telemetry and recovery state

| Signal | Baseline | Source |
|---|---:|---|
| 7-day latency p50 / p95 | 1,344 / 6,825 ms | `search-hub metrics federated-latency --window last_7d --json` |
| 7-day degraded queries | 5,629 / 17,735 (31.74%) | `search-hub metrics degraded-query-rate --window last_7d --json` |
| All-time zero-result queries | 882 / 7,674 (11.49%) | `search-hub insights --json` |
| Registered providers | 30 | `search-hub providers list --json` |
| Routeable federation providers | 26 | `search-hub federation status --json` |
| Demoted providers | 2 | `provider_demotion_state` query and federation status |
| Persisted probation rows | 2 | `provider_demotion_state` query |
| Stuck rows (demoted and decay deadline elapsed) | 2 | `provider_demotion_state` query at capture time |

The complete per-provider route/hit inventory is in
`tmp/baseline-2026-08-13/insights.json`; the complete demotion-state row
inventory is in `tmp/baseline-2026-08-13/provider-demotion-state.tsv`. The
routeable inventory includes zero-hit providers such as
`content-desk.editorial-history`, `signal-inbox.signals`, and
`web-search.live`.

### Certification and readiness board

| Signal | Baseline | Source |
|---|---:|---|
| Maturity scan | 5 passed / 17 failed / 1 unavailable of 23 | `search-hub maturity scan --json` |
| Maturity levels | L0: 15, L2: 2, L4: 5, unavailable: 1 | same response, `.results[].current_level` |
| Answer projection | NOW 14 / in-reach 18 / missing 4 of 36 | `meta-optimization-manager coverage status --json` |
| Answer coverage ratio | 0.3888888889 | same response, `PROJECTION_ANSWER.coverage_ratio` |

The maturity command returned the complete first JSON object followed by a
duplicate trailing fragment, so strict JSON parsing fails even though the
summary is present. The raw response and stderr are retained unchanged; the
summary above is taken from the valid first object. This is recorded as a
phase finding for remediation rather than silently normalizing baseline data.

### Raw baseline command set

The exact command family used was: `search-hub evals list --json`, one
`search-hub evals runs <suite_id> --json --limit 12` per suite,
`search-hub insights --json`, `search-hub metrics federated-latency --window last_7d
--json`, `search-hub metrics degraded-query-rate --window last_7d --json`,
`search-hub federation status --json`, `search-hub providers list --json`,
the documented SQLite `provider_demotion_state` query, `search-hub maturity
scan --json`, and the four Meta-Optimization Manager board commands:
`coverage status`, `coverage validate-docs`, `focus next`, and
`condition status`, each with `--json`. The saved aggregate is
`tmp/baseline-2026-08-13/headline-aggregates.json`.

### Model leg costs and phase-4 reading

The measured host leg costs used for the phase-4 bounds are: TEI
cross-encoder p50 approximately 25 ms (ten-request probe); classifier
`qwen3:1.7b` warm approximately 620–790 ms and cold approximately 16.41 s;
`nomic-embed-text` warm approximately 10–30 ms. The cross-encoder bound is
therefore 500 ms, while the LLM fallback bound is 8 s. These are explicit
environment-overridable budgets, not a claim that cold model loading is
acceptable.

| Signal | Phase-1 baseline | Phase-4 post-change reading | Delta |
|---|---:|---:|---:|
| 7-day p50 | 1,344 ms | 1,344 ms | 0 ms |
| 7-day p95 | 6,825 ms | 6,825 ms | 0 ms |
| 7-day degraded rate | 31.74% | 31.74% | 0 percentage points |

The recaptured readings came from `search-hub metrics federated-latency
--window last_7d --json` and `search-hub metrics degraded-query-rate --window last_7d
--json`. They are historical fleet telemetry, not a controlled benchmark;
the host-level model residency change was not applied, so no cold-start win is
claimed. Search Hub does not manage model residency: the classifier and
embedding models must be kept warm by host/control-plane configuration.

## Phase 12 — description-index scaling and attribution

The persistent provider-description index embeds only changed descriptors and
stores model/dimension/policy metadata with each vector. A failed descriptor
embedding is dropped from the index and reported; routing remains bounded by
the deterministic relevance-scored fallback. The local benchmark below is a
construction/shortlist smoke benchmark, not a production latency SLO:

| Registered descriptions | Time (1 iteration) |
|---:|---:|
| 25 | 81,150 ns/op |
| 100 | 140,230 ns/op |
| 250 | 343,220 ns/op |

Command: `cd scenarios/search-hub/api && go test ./internal/routing -run '^$' -bench BenchmarkDescriptionIndexShortlist -benchtime=1x -count=1`.
The production cache path is scenario data, and cache entries are invalidated
when the descriptor fingerprint or embedding metadata changes.

## Closing audit readings

The baseline remains the comparison point: provider-direct 167/195 (85.64%),
federated 7/51 (13.73%), p50 1,111 ms, p95 10,129 ms, degraded 25.54%, and 23
maturity targets with 5 passed, 17 failed, and 1 unavailable. The post-change
fleet has 31 registered providers, 2 incubating providers, and exposes router
ordering (`score` or `rerank_score`), routing precision, retrieval recall,
corpus live/hard/stale counts, and zero-yield accounting as separate signals.
These readings are not claimed as a controlled improvement while the shared
Ollama/reranker substrate remains degraded; the Code Facts provider gap and
stale provider-owned labels are recorded in `PROBLEMS.md`.

### 2026-08-13 — Code Facts corpus re-registration verification

The first post-edit direct run was found to be executing the old identifier
corpus from the Search Hub store. Both Code Facts descriptors and suites were
re-registered through the ordinary CLI path, then executed again. The corrected
readings are deliberately unfavorable: each suite has 3/15 met (all three
negative cases) and 12/15 natural-language positives below expectation. The
`code-facts.code` p95 is 1,500 ms and `code-facts.contracts` p95 is 152 ms. The
12 unanswerable positives are now `candidate` cases with explanatory notes;
they do not certify provider capability. The required live probe still returns
no Code Facts hit, so Phase 8 remains blocked on the provider-owned query/index
implementation. Evidence: direct runs `6188e2f1-a182-48dd-8af1-79000fdcddd8`
and `146a3ffc-73cf-47c1-aaa2-d6507e97e34e`.

The Search Hub evaluator was then corrected to exclude `candidate` cases from
provider execution and from the persisted graded denominator, matching the
documented acceptance model. The post-fix direct runs are
`cf4dd7db-df1d-4d6d-8794-d02e354efbb8` (`code-facts.code.primary`) and
`7244e1aa-759b-46db-b16f-66b9d0522b5a` (`code-facts.contracts.primary`): each
records 12 candidates as `n/a` and grades only the three reviewed negatives.

## 2026-08-14 final validation snapshot

The durable closing comparison used the original collection
`search-hub-end-to-end-retrieval-trust-routing-attribution-baseline` (captured
at Git SHA `6cace1549d5749f178e79d16c9d1d53ae8d7c53d`) and fresh member runs
`20260814-043927-e20729a8` (Code Facts), `20260814-043928-bcb09354` (Meta-
Optimization Manager), and `20260814-043929-e902b635` (Search Hub). The
collection verdict was `clean` / `preexisting`: all three suites retained
their existing failures and introduced no new baseline regression.

| Closing signal | Closing reading | Comparison / explanation |
|---|---:|---|
| Router composed suite | 212 cases; 155 gradeable; routing precision 0.4258; retrieval recall 0.8788; p95 7,562 ms | The two rates are now separately persisted and attributable. Retrieval quality clears the observed corpus bar; routing remains below the 0.85 target and is recorded as router/model/provider-evidence debt rather than hidden in `pass_rate` (0.1548). |
| Code Facts direct quality | `code-facts.code` 15/15 and `code-facts.contracts` 15/15 in the focused direct runs | The natural-language provider-owned cases are executable and no longer degenerate; the managed maturity suite still has a preexisting Code Facts unit-phase failure. |
| Registered providers | 33 registered; 25 production, 7 experimental, 1 fixture; no lifecycle-unset rows | Baseline was 30. The increase is additive; no provider was removed. Experimental providers remain visible under `incubating`. |
| Demotion recovery safety | 0 rows with `demoted=1`, `probation=1`, and expired `decay_deadline` | The recovery latch fix and startup reconciliation leave no stuck provider in the current store. |
| Code-location probe | `code-facts.code` selected; `demotion.go` returned at rank 2 of the declared top 5 | The query was verified with `--explain`; the owning source location is present even when the live reranker/resource substrate is degraded. |
| Search Hub self-certification | `search-hub.docs.primary` registered and run `3ede3f2e-0fc5-4578-86af-2227f362ec59` stored | The suite is now visible to the maturity ladder. Its one reviewed positive was below expectation on this host because `knowledge-observatory` could not start: shared `@vrooli/proto-types` provisioning (`make generate`) failed during setup. |
| Reranker substrate | `gpu-degraded`; `/dev/nvidiactl` permission denied | This is host infrastructure, not Search Hub code. The fast cross-encoder leg remains bounded; no resource implementation was changed by this plan. |

The managed scenario runs are intentionally reported separately from the
collection result: Search Hub and Meta-Optimization Manager remain red on
pre-existing UI, dependency, storage, security, maturity, and contract debt.
The fresh Plan Manager validation nevertheless passed at
`c4e78f3c-5629-48eb-a73c-7cb1b7eccd39` with `STALENESS_TIER_FRESH` and recorded
the three member outcomes as `preexisting`.

## Known Constraints

- Vite production builds may process thousands of modules and take
  several minutes.
- Live before/after p50 and p95 comparisons require the honest baseline with
  qdrant and ollama healthy. The current values are recorded above, but the
  shared runtime remains degraded, so no latency improvement is claimed.

## 2026-08-14 authoritative closure snapshot

The managed run `20260814-064118-c97a610f` passed all 21/21 Search Hub phases
in 178 seconds with zero failures. This is validation evidence, not a
single-variable performance benchmark.

| Signal | Reading | Command / evidence |
|---|---:|---|
| Registered providers | 33 (29 active, 4 capability-gap) | `search-hub providers list --json` |
| Router suite | 212 cases; 155 gradeable; routing precision 0.4258; retrieval recall 0.8788; p95 7,562 ms | `search-hub evals runs router.routing --json`, run `e514f3ac-3215-412c-aa2f-21634ca642f9` |
| Search Hub self suite | 2/2 met; recall 1.0; p95 79 ms; indexed 14,170 | `search-hub evals run search-hub.docs.primary --tier provider_direct --json`, run `7fa1b03e-c24a-4c5f-8942-a3d5056edcc6` |
| Code-location probe | `code-facts.code` only; `demotion.go` at rank 1/5 | `search-hub query "where is the function that computes provider demotion" --type code --limit 5 --explain --json` |
| Comprehensive suite | 21 passed, 0 failed, 0 skipped | `vrooli scenario test search-hub --json` |

The fleet-wide maturity command still reports 23 blocking findings owned by
other providers, but its Search Hub result is `passed` with fresh live self
evaluation. No fleet provider debt is presented as a Search Hub performance
regression. The reranker remains host-degraded because of GPU device
permissions; Search Hub's single-provider grouped path intentionally remains
healthy and ordered by provider score when that optional leg is unavailable.

## Regression Procedure

1. Run `make test`.
2. Capture relevant API/UI command timing.
3. For UI interaction regressions, use `ui/perf/README.md` and the
   provided capture template.
4. Record persistent findings in this document or
   [`PROBLEMS.md`](PROBLEMS.md) depending on whether they are accepted
   constraints or unresolved debt.

## Cross-References

- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — signals and telemetry
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — release checklist
- [`TESTING.md`](TESTING.md) — coverage and test expectations
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved performance debt
