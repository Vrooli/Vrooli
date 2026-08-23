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

## Phase 4 regression anchor — 2026-08-16

The retrieval-correctness plan's anchor window began after the reranker had
remained continuously healthy from `2026-08-16T00:00:38Z` through the required
one-hour stability gate. At `2026-08-16T01:01:22Z`, `vrooli resource status
reranker --json` reported `running=true`, `healthy=true`, and a live
`capacity-sync` companion; `search-hub federation status --json` reported
`reranker_leg=cross-encoder:BAAI/bge-reranker-v2-m3`.

Every accepted anchor run below has the stored config
`reranker_leg=cross-encoder:BAAI/bge-reranker-v2-m3`,
`embed_model=mixed:ai-gateway:embedding.default,nomic-embed-text`, and a
non-empty `indexed_count`; `selector_leg` distinguishes `provider_direct`
from `cross_encoder`. Three provider-direct attempts that reported
`reranker_leg=none` from provider-owned status endpoints were rejected and
retaken after the shared substrate snapshot fallback was repaired. `n/a`
means that the run had no applicable graded denominator, not that the value
was silently treated as zero.

| Suite | Tier | Run | Pass rate | Routing precision | Retrieval recall | Graded | Indexed / selector |
|---|---|---|---:|---:|---:|---:|---|
| architecture-cartographer.domain-map.primary | federated | `2cdd3f7a-cbb4-4e26-b534-1c258b110c82` | 0.1111 | 0.2222 | 0 | 18 | 100 / cross_encoder |
| architecture-cartographer.domain-map.primary | provider_direct | `002368a9-d7dc-4b54-98d2-76b975ed8957` | n/a | n/a | n/a | n/a | 100 / provider_direct |
| business-health.intent.primary | federated | `6870edea-1215-496f-a477-d10b9dd3f964` | 0.2000 | 0 | n/a | 15 | 100 / cross_encoder |
| business-health.intent.primary | provider_direct | `a1969250-1506-4066-a006-98957c45460d` | 1 | n/a | n/a | 15 | 100 / provider_direct |
| cli-health.commands.primary | federated | `50661325-d39c-4cc3-a1e7-bd1356badd32` | 0.3226 | 0.4872 | 1 | 31 | 100 / cross_encoder |
| cli-health.commands.primary | provider_direct | `dc1f1f04-c58e-4763-86ff-96b6aaf3552c` | 0.8710 | n/a | n/a | 31 | 3056 / provider_direct |
| code-facts.code.primary | federated | `4f2a1e70-9273-4650-a513-2be27e702020` | 0.6000 | 0.6000 | 1 | 15 | 100 / cross_encoder |
| code-facts.code.primary | provider_direct | `22967cb8-aa87-4b9e-b0b2-6dc4078c457d` | 1 | n/a | n/a | 15 | 7240 / provider_direct |
| code-facts.contracts.primary | federated | `66db8194-d9f3-4075-924f-a01ad99c50c8` | 0.4667 | 0.9333 | 0.5 | 15 | 100 / cross_encoder |
| code-facts.contracts.primary | provider_direct | `2be2c67e-66cf-4045-b5e7-fe07a7710b59` | 1 | n/a | n/a | 15 | 7240 / provider_direct |
| content-desk.editorial-history.primary | federated | `f07884be-e2c0-488d-af8b-411622621bec` | n/a | 0 | n/a | n/a | 100 / cross_encoder |
| content-desk.editorial-history.primary | provider_direct | `33c0e517-b18d-409c-a28a-b0183e6230fb` | n/a | n/a | n/a | n/a | 100 / provider_direct |
| knowledge-observatory.docs.starter | provider_direct | `6e6df60e-9451-4a45-9ce6-0b8ea5974cc0` | 0.8696 | n/a | n/a | 23 | 14181 / provider_direct |
| knowledge-observatory.docs.starter | federated | `de3f92ce-ec70-4deb-825e-c3b92057ae43` | 0.1304 | 0.2174 | 0.4 | 23 | 100 / cross_encoder |
| measures-health.measures.primary | federated | `a2e94e54-ee01-4ac2-b891-950ec0f24b52` | 0.2500 | 0.5556 | 0.5 | 8 | 100 / cross_encoder |
| measures-health.measures.primary | provider_direct | `18ac4a0c-cf6e-4438-ab98-1c6ba8a29d26` | 0.8750 | n/a | n/a | 8 | 100 / provider_direct |
| prompt-manager.action.primary | federated | `776b0165-63cc-4b1f-b9f0-36ce5f7ea16e` | 0.3333 | 0 | n/a | 3 | 100 / cross_encoder |
| prompt-manager.action.primary | provider_direct | `77f022e5-a2ea-446e-aab3-ff6e2fe42924` | 1 | n/a | n/a | 3 | 100 / provider_direct |
| prompt-manager.skill.primary | federated | `25704ef2-14ee-4cfe-933a-4c29adb70692` | 0.6667 | 0.3333 | 1 | 3 | 100 / cross_encoder |
| prompt-manager.skill.primary | provider_direct | `5d7a2166-0812-40b0-abda-a3d92f5a8bde` | 1 | n/a | n/a | 3 | 100 / provider_direct |
| scenario-dependency-analyzer.dependencies.primary | federated | `e29d9d67-a005-427c-8d63-3f136a4937ec` | 0.2500 | 0.1250 | 1 | 8 | 100 / cross_encoder |
| scenario-dependency-analyzer.dependencies.primary | provider_direct | `c9caf6f0-9136-4c58-a2ea-38285f8021a4` | 0.8750 | n/a | n/a | 8 | 100 / provider_direct |
| scenario-dependency-analyzer.resources.primary | federated | `c87fa649-3fe9-4b33-a119-391ccef605c1` | 0.0769 | 0 | n/a | 13 | 100 / cross_encoder |
| scenario-dependency-analyzer.resources.primary | provider_direct | `e96120c0-583d-41ff-a66e-4b9be56462f9` | 0.7692 | n/a | n/a | 13 | 100 / provider_direct |
| scenario-dependency-analyzer.scenarios.primary | federated | `97adcaa8-fd62-4427-8ded-0d438571a193` | 0.5385 | 0.4615 | 1 | 13 | 100 / cross_encoder |
| scenario-dependency-analyzer.scenarios.primary | provider_direct | `68f0311b-c567-41e1-bb1c-0c8f739675b4` | 1 | n/a | n/a | 13 | 100 / provider_direct |
| search-hub.docs.primary | provider_direct | `33e48762-5513-4b85-af62-08faaed4e760` | 1 | n/a | n/a | 2 | 14181 / provider_direct |
| search-hub.docs.primary | federated | `98880d13-5eea-4efa-9b96-4c41430b0803` | 1 | 0.5 | 1 | 2 | 100 / cross_encoder |
| source-ledger.agent-memory.primary | federated | `e1552f24-fdf2-4e0b-9d1c-41260e3c4c87` | 1 | 0.5 | 1 | 2 | 100 / cross_encoder |
| source-ledger.agent-memory.primary | provider_direct | `61089557-ba40-4f0b-b107-a5f304505972` | 0.5 | n/a | n/a | 2 | 100 / provider_direct |
| source-ledger.scopes.primary | federated | `908cdbec-7d2f-478c-ad7d-4e5816955182` | 0.5 | 0 | n/a | 2 | 100 / cross_encoder |
| source-ledger.scopes.primary | provider_direct | `1bb7a0f3-861c-4f6d-aa92-00236c28dc28` | 0.5 | n/a | n/a | 2 | 100 / provider_direct |
| swarm-manager.initiative.primary | federated | `7bcf80df-34cb-49f7-ad5a-94b0e221a574` | 0.5 | 0 | n/a | 2 | 100 / cross_encoder |
| swarm-manager.initiative.primary | provider_direct | `d9a00638-57f7-469e-a1f3-5b0345cd64b7` | n/a | n/a | n/a | 2 | 100 / provider_direct |
| swarm-manager.records.primary | federated | `77fbe2bc-dd5c-4c34-b6f6-7cd225d733c8` | 0.1429 | 0 | n/a | 14 | 100 / cross_encoder |
| swarm-manager.records.primary | provider_direct | `8d5affdf-e76f-47de-83b3-66a0f79d3202` | 0.1429 | n/a | n/a | 14 | 100 / provider_direct |
| swarm-manager.records.starter | federated | `3e27cfb5-fdfd-4762-a600-d4752168c729` | 0.25 | 0 | n/a | 4 | 100 / cross_encoder |
| swarm-manager.records.starter | provider_direct | `d0fbcdd8-e27f-4e21-9566-316070bcd5c6` | n/a | n/a | n/a | 1 | 100 / provider_direct |
| template-manager.debt.ledger | federated | `46a79d47-e38b-4aaf-8f55-92306630600b` | 0.0769 | 1 | 0 | 13 | 100 / cross_encoder |
| template-manager.debt.ledger | provider_direct | `2444109e-ae01-4cd4-988d-d1967a54671c` | n/a | n/a | n/a | n/a | 100 / provider_direct |
| template-manager.docs.factory | provider_direct | `3772aa49-057e-4bf4-8563-ae93cbf8efc1` | 1 | n/a | n/a | 13 | 14181 / provider_direct |
| template-manager.docs.factory | federated | `2fa46c31-bcb8-4bd4-9494-fbf0e0ffa131` | 0.3077 | 0.4615 | 1 | 13 | 100 / cross_encoder |
| ui-health.surfaces.primary | federated | `8964a5f9-d799-4b24-a0a5-88643adc52b7` | 0.2222 | 0 | n/a | 9 | 100 / cross_encoder |
| ui-health.surfaces.primary | provider_direct | `736e9eef-225f-47ff-9aac-9d84cd094ee1` | 1 | n/a | n/a | 9 | 1564 / provider_direct |
| ui-health.surfaces.starter | federated | `79372190-4ffb-4f9a-9ad6-8884860cba3e` | 0.25 | 0 | n/a | 4 | 100 / cross_encoder |
| ui-health.surfaces.starter | provider_direct | `18cc4574-69d5-4596-a772-68a14453aea3` | n/a | n/a | n/a | 1 | 1564 / provider_direct |
| web-search.learnings.primary | federated | `5133445e-225d-42e2-a66f-7b41993dab45` | 0.25 | 0 | n/a | 4 | 100 / cross_encoder |
| web-search.learnings.primary | provider_direct | `6124bdd9-f9f7-432a-8e5c-167add6f72e2` | 1 | n/a | n/a | 4 | 100 / provider_direct |
| web-search.live.primary | federated | `01e182cc-58cc-41da-bce5-a131202f60fb` | n/a | 0 | n/a | 1 | 100 / cross_encoder |
| web-search.live.primary | provider_direct | `42a5586f-a582-4da9-bfa5-71cf2b154760` | n/a | n/a | n/a | n/a | 100 / provider_direct |
| workflow-health.fragments.primary | federated | `134d1734-8bcb-4a55-a84c-081f535e6191` | 0.5 | 0.5 | 1 | 8 | 100 / cross_encoder |
| workflow-health.fragments.primary | provider_direct | `8b7716d7-2e6e-4b18-a971-eef3145fa93f` | 1 | n/a | n/a | 8 | 100 / provider_direct |
| workflow-health.tests.primary | federated | `b3aad4ae-5ba9-41aa-b5e5-2e0ef343abb1` | 0.125 | 0.75 | 1 | 8 | 100 / cross_encoder |
| workflow-health.tests.primary | provider_direct | `d60074e3-b32a-40e2-87a9-c51dae0bab56` | 1 | n/a | n/a | 8 | 100 / provider_direct |
| workflow-health.workflows.primary | federated | `87de99c1-852d-4ee1-8b6b-06de35a91292` | 0.125 | 0.375 | 1 | 8 | 100 / cross_encoder |
| workflow-health.workflows.primary | provider_direct | `d4fa2bf5-3d2d-4a74-857d-4129cbef03bc` | 1 | n/a | n/a | 8 | 100 / provider_direct |
| router.routing | federated | `220f3a84-9e62-4eda-8ccf-f0752954c5bc` | 0.1643 | 0.3756 | 0.675 | 213 | 100 / cross_encoder |

The anchor `router.routing` reading is `routing_precision=0.3756`,
`retrieval_recall=0.6750`, `pass_rate=0.1643`, and `latency_p95_ms=2512`.
The full-window insights snapshot at `2026-08-16T01:08:00Z` was
`latency_p50_ms=1196`, `latency_p95_ms=6832`, and `zero_result_rate=0.1935`,
with 218 `reranker_down` substrate events in the accumulated telemetry. The
anchor p95 and the insights p95 exceed the 2000 ms condition budget; this is
an explicit baseline constraint for the fan-out phase, not a hidden pass.

The same-window Meta-Optimization Manager Answer snapshot was deterministic
and reported 36 total cells, 32 in reach, 4 missing, `coverage_ratio=0`, and
24 corpus-capable cells. The recorded Answer cell list is `answer/1` through
`answer/36`; missing cells were `answer/4`, `answer/9`, `answer/26`, and
`answer/32`. This snapshot is retained as the board-side anchor rather than
conflated with provider-direct retrieval quality.

The fleet-wide maturity command still reports 23 blocking findings owned by
other providers, but its Search Hub result is `passed` with fresh live self
evaluation. No fleet provider debt is presented as a Search Hub performance
regression. The reranker remains host-degraded because of GPU device
permissions; Search Hub's single-provider grouped path intentionally remains
healthy and ordered by provider score when that optional leg is unavailable.

## Phase 5 bounded fan-out — 2026-08-16

Phase 5 replaced the automatic top-1 truncation with a strategy-defined,
bounded provider set. The active `lexical-cross-encoder` row now carries
`fanout_width=6`, capped by `router_factors.max_fanout_width=6`. The reranker
therefore receives hits from multiple provider groups and the response can
carry one unified `rerank_score` ordering instead of silently stopping after
the first provider.

The width decision was measured on the same stored substrate contract as the
Phase 4 anchor. Width 1 was retaken after an earlier LLM-fallback run was
discarded as incomparable.

| Width | Run | Routing precision | Pass rate | Retrieval recall | p95 latency |
|---:|---|---:|---:|---:|---:|
| 1 | `cdaa7d5d-dd3d-483f-83bc-01d3b2fc5864` | 0.3192 | 0.1596 | 0.9118 | 4548 ms |
| 3 | `719a5c39-50b4-4cf8-b541-a8843a1717d7` | 0.5070 | 0.1080 | 0.3333 | 3099 ms |
| 6 | `acae3fcc-719a-4133-8c03-89d38e43b3d6` | 0.6150 | 0.0845 | 0.1450 | 5130 ms |

Width 6 is selected because it is the only tested width that satisfies the
declared multi-owner acceptance probe at the stable cross-encoder leg. The
query `What surfaces does plan-manager expose?` returned both
`ui-health.surfaces` and `cli-health.commands`, plus four related providers,
with `reranker_leg=cross-encoder:BAAI/bge-reranker-v2-m3`,
`ordered_by=rerank_score`, and `degraded=false`. The final chosen router run
`026bc611-ba7b-4134-a43d-68eb2b5eca18` measured routing precision `0.6995`,
pass rate `0.0939`, retrieval recall `0.1544`, and p95 `3680 ms`.

The selected route exceeds the 2000 ms condition budget; this is an explicit
decision, not an omitted measurement. Width 3 was faster in one golden run but
failed the owner-set probe deterministically. The 29-suite `fanout-chosen-final`
sweep and `search-hub evals compare` calls were completed against the Phase 4
federated anchors. Suites with pass-rate movement greater than 0.05 are
recorded as a fan-out tradeoff: widening changes candidate competition and
unified rerank order, so provider routing precision can improve while the
provider-specific expected-hit grade falls. The affected suite IDs are
`cli-health.commands`, `code-facts.code`, `code-facts.contracts`,
`knowledge-observatory.docs`, `prompt-manager.skill`,
`scenario-dependency-analyzer.dependencies`,
`scenario-dependency-analyzer.scenarios`, `search-hub.docs`,
`source-ledger.agent-memory`, `template-manager.docs`,
`workflow-health.fragments`, and `router.routing`; no movement is hidden or
treated as a clean quality win. Phase 6's semantic selector is the tested
lever for recovering this tradeoff without abandoning the multi-owner route;
the guarded candidate remains unpromoted until its paired evidence clears the
significance and held-out guards.

The post-change insights snapshot recorded `zero_result_rate=0.1822`, below the
Phase 4 anchor `0.1935`; accumulated query telemetry reported p50 `1259 ms`,
p95 `6473 ms`, and 1126 `reranker_down` events. The latency breach and the
substrate-degradation count remain visible for the next phase rather than
being attributed to corpus quality.

## Phase 6 strategy comparison — 2026-08-16

The initial strategy comparison was run through the guarded
`CompareStrategies` RPC with `router.routing`, federated execution, and
per-case limit 10. The initial semantic candidate embedded every automatically
eligible leaf, sent the resulting provider ordering to the cross-encoder,
applied the bounded six-leaf fan-out, and fused lexical evidence over
registered provider metadata. The lexical arms retained their six-leaf
shortlist. The federated evaluator captures one automatic-eligibility snapshot
for the complete comparison and records owners with a freshness, lifecycle, or
evidence exclusion as `unavailable`; those cases remain visible in
`unavailable_cases` but do not turn an enforced policy decision into a selector
miss. The comparison launcher defaults to the live strategy catalog and uses a
45-minute client deadline because this is a 213-case local-model benchmark.

| Arm | Run | Gradeable denominator | Routing precision | Pass rate | Retrieval recall | Unavailable | Held-out | Paired significance |
|---|---|---:|---:|---:|---:|---:|---|---|
| lexical-cross-encoder (incumbent) | `1a2f2c73-8c0b-4bcd-9081-70378f7dee37` | 177 / 213 | 0.8983 | 0.2373 | 0.2893 | 36 | holds | incumbent |
| semantic-cross-encoder (candidate) | `bda4979c-84e1-4fbb-a594-e9ff7c731f78` | 177 / 213 | 0.8983 | 0.0565 | 0.0629 | 36 | holds | CI lower 0.0000; not significant |
| lexical-fallback | `2438517f-8f61-4036-93fc-a8059936f8eb` | 177 / 213 | 0.8983 | 0.0056 | 0.0189 | 36 | holds | CI lower 0.0000; not significant |

The policy-aware routing precision clears the plan's `0.85` threshold on the
177 cases whose owners were allowed by the automatic gate. The 36 excluded
cases remain attributable and visible rather than being silently deleted. The
live declaration/code probes still pass: `sqliteDemotionStore`,
`ProviderHealth`, `SearchQuery`, and `ProviderScope` all rank `code-facts.code`
first. However, the semantic candidate does not clear the paired significance
guard and has materially lower pass rate and retrieval recall than the
incumbent, so it is not promoted. The measured lexical-cross-encoder remains
active; lexical-fallback and semantic-cross-encoder remain explicit, named
recovery/experiment arms.

### Phase 6 follow-up evidence-window experiment — 2026-08-16

The first semantic candidate admitted the full dense/lexical fused ordering to
the provider cross-encoder and materially underperformed the incumbent. The
implementation now ranks every eligible leaf in the embedding index, but
passes a data-defined top-three dense/lexical evidence union to the expensive
selector before retaining the six-provider output bound. This preserves exact
identifier evidence without pretending that a dense ordering is authoritative.

The focused comparison completed after that change with 176 gradeable cases
and 37 policy-withheld cases. The incumbent measured `pass_rate=0.0909` and
`routing_precision=0.9091`; the semantic candidate measured
`pass_rate=0.1023` and `routing_precision=0.8580`, but its paired routing delta
was negative (`mean=-0.0511`, 95% CI lower `-0.0966`) and the held-out routing
precision guard failed. The candidate therefore remains an explicit guarded
experiment, not the active strategy. The run IDs are
`9ca1d3d7-e9e0-4972-8644-5affa74fc454` for the incumbent,
`a87985bb-d2f7-4279-8597-62db2aaf3a56` for the semantic candidate, and
`5ca31a63-9c00-45d4-899a-bdbca268f500` for lexical fallback.

### Phase 6 final lexical-floor hardening — 2026-08-16

The semantic index now embeds the complete registry routing context alongside
each description segment (provider identity, type, group, and description),
with policy version `4` invalidating the prior description-only cache. The
semantic arm still ranks every eligible leaf and sends the full eligible set
through the cross-encoder, but promotion is guarded by cross-encoder score
advantage plus a relative lexical/type safety floor. This prevents a generic
record leaf from displacing a documentation or code signal merely because its
description shares broad query words.

The fresh comparison retained the same eligible denominator and held-out
result: incumbent `62f646f5-f460-46a2-ac50-a94b4d6610bd` measured
`routing_precision=0.8983`, `pass_rate=0.1921`, and `retrieval_recall=0.2201`;
semantic candidate `aa87e386-ccc4-4f8d-89fa-7e8f93638266` measured
`routing_precision=0.8983`, `pass_rate=0.0904`, and
`retrieval_recall=0.1384`. `heldout_holds=true`, but the paired routing delta
was exactly zero, so the significance guard correctly kept the lexical
incumbent active. The candidate remains named and reproducible rather than
being promoted on a tie.

### Phase 6 final write-back audit — 2026-08-16

After restoring the strict lexical-floor policy, the authoritative comparison
was repeated with `--apply` so promotion could only occur through the guarded
write-back path. The incumbent run `d7bfd60b-561c-4cc2-8346-3ec68b2b3b8f`
measured `routing_precision=0.8939`, `pass_rate=0.1453`, and
`retrieval_recall=0.1750` over 179 gradeable cases. The semantic candidate run
`8e9ce24e-758f-416a-aa0f-1ad992e16022` measured the same routing precision but
`pass_rate=0.0950` and `retrieval_recall=0.1125`; held-out precision held, but
the paired routing delta remained exactly zero. The command therefore returned
`write-back refused: no strategy cleared both paired significance and held-out
validation`, leaving `lexical-cross-encoder` active. A temporary weaker
weakest-floor gate was also measured (`32d8c4d2-9f22-47f0-9b30-bea6901b50cf`)
and reverted because it produced no routing gain and no retrieval-quality
recovery. This is the final promotion evidence; the semantic arm is not
promotable on the current registered-provider corpus.

## Phase 7/8 operational evidence — 2026-08-16

Provider freshness is now descriptor-driven with a default 24-hour budget;
status reports an explicit stale-index exclusion reason, and production
registration rejects descriptors without a status endpoint. Warm federation
status completed in under two seconds after the first bounded probe. The eight
existing production providers without status endpoints were filed to
`scenario-qa` as owner defects rather than silently treated as fresh.

The answer board now reports corpus-capable coverage beside end-to-end answer
coverage. Focus rolls multiple `router_quality_debt` cells into the shared
cause `search-hub/router_quality_debt`; the live board reported 14 affected
cells and ranked that cause above individual condition gaps. Stopping reranker
produced `condition/substrate/reranker` with `reranker leg unavailable`; after
managed restart the item disappeared and healthy focus returned with
`degraded=false`. The reranker resource's auxiliary metrics port is explicitly
declared at 11454 to avoid the host's MinIO port 9000.

## Regression Procedure

## Phase follow-up — route-discriminative semantic diagnostics (2026-08-16)

The route-profile and stage-trace follow-up was validated on the restarted
Search Hub process after rebuilding the API and UI artifacts. The full managed
suite passed 21/21 in Test Genie run `20260816-155029-1135242f`. The lifecycle
restart reported the known Ollama configuration-build warning, but Search Hub
became healthy and the comparison runs recorded the cross-encoder leg and an
available provider-description index.

The authoritative three-arm comparison was requested with `--apply`; the
write-back guard refused promotion:

| Arm | Run | Gradeable / unavailable | Routing precision | Pass rate | Retrieval recall | P95 | Trace cases |
|---|---|---:|---:|---:|---:|---:|---:|
| `lexical-cross-encoder` (incumbent) | `17c28c65-7c12-4414-8aac-374d33be510c` | 179 / 34 | 0.8939 | 0.1788 | 0.2625 | 6381 ms | 213 / 213 |
| `semantic-cross-encoder` | `1f7145fd-4b6f-435c-9403-ff86df27139c` | 179 / 34 | 0.8939 | 0.0000 | 0.00625 | 4843 ms | 213 / 213 |
| `lexical-fallback` | `6fda3d94-559c-40cf-bd85-36db62632aa3` | 179 / 34 | 0.8939 | 0.0726 | 0.1000 | 5781 ms | 213 / 213 |

For the semantic arm, expected-owner evidence was present in the dense top
1/3/6 for 52/87/117 of the 179 gradeable cases (29.1%/48.6%/65.4%). The
positive lexical/dense evidence union retained the expected owner for 173/179
cases (96.6%), and the selected provider set retained it for 160/179 (89.4%),
matching the incumbent's routing precision. The remaining loss is downstream:
the owner can be selected and still fail to return the expected item within
the case's declared top-K, so this comparison separates provider retrieval
quality from route selection instead of calling it a semantic routing win.

The candidate held the held-out routing fold, but its paired routing delta was
exactly zero with a 95% CI lower bound of zero. The guarded `--apply` request
therefore returned `write-back refused: no strategy cleared both paired
significance and held-out validation`; `lexical-cross-encoder` remains active.
The run's traces also show a second concrete limitation: 62/179 gradeable
cases miss the semantic dense top-six window, while five more lose the owner
at the guarded selector. The route profile is now useful evidence, but the
current provider corpus and query representation are not sufficient grounds
for semantic promotion.

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
