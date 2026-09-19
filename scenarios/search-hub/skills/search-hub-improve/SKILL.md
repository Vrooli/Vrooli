---
name: "search-hub-improve"
description: "Regulate Search Hub against its setpoint: routing precision, end-to-end pass rate, retrieval recall, provider degradation, degraded-query rate, federated latency, under-utilized providers, zero-result rate, reachability, and eval floors. Routes each out-of-band row to a strategy or eval move, a work-ladder rung, or a corpus-quality filing against the provider owner with the measure that proves it."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["practice"]
  tags: ["search-hub", "improve", "self-improvement", "control-loop", "setpoint", "routing-precision", "evals", "corpus-quality", "meta-optimization"]
  icon: "gauge"
  status: "active"
  revision: 2
  createdAt: "2026-09-02T00:00:00Z"
  updatedAt: "2026-09-02T20:00:00Z"
  requires:
    scenarios: ["search-hub", "program-runtime", "prompt-manager", "vrooli-memory"]
    commands: ["search-hub evals list", "search-hub evals runs", "search-hub evals run", "search-hub evals show-run", "search-hub evals compare", "search-hub evals compare-strategies", "search-hub evals promote", "search-hub evals validate", "search-hub strategy list", "search-hub strategy compare", "search-hub strategy benchmark", "search-hub insights insights", "search-hub metrics federated-latency", "search-hub metrics degraded-query-rate", "search-hub metrics provider-degradation-rate", "search-hub federation status", "search-hub federation repromote", "search-hub providers list", "program-runtime programs submit", "prompt-manager skill read", "vrooli-memory journal note"]
  origin:
    kind: "authored"
---
## Practice focus: Search Hub Improve

Regulate Search Hub — the federated router, its measurement backbone, and its eval domain — against the setpoint below. The plant is routing quality (does the right corpus get asked) and federation health (do legs answer, how fast, how often empty). Retrieval quality inside a corpus belongs to that provider's owner; this skill measures it and files, it does not fix it. Read by an agent whose task is search-hub itself.

Required reading:
- `prompt-manager skill read search-hub` — the usage skill; token vocabulary and verdicts.
- `prompt-manager skill read improvement-do-and-dont` — anti-gaming, cited by section in §6.
- `prompt-manager skill read scenario-work-ladder` — where code routes go.
- `path:scenarios/search-hub/docs/internal/PROBLEMS.md` §"2026-08-16 — Router precision is policy-aware" — why 0.8983 precision and 0.2373 pass rate coexist.

### 1. Focus and scope

**In scope:** the setpoint rows below; strategy comparison and benchmark; eval suite hygiene (validate, promote reviewed candidates, compare runs); repromotion of a provider after its owner fixed it; filing corpus-quality items against provider owners with the run id and measure that prove them; filing ladder rungs against search-hub.

**Out of scope:** editing any provider's `search.json`, index, or documents; lowering any floor; running `--apply` or `--confirm` without the guard's own significance verdict; the usage skill's content; web-search's ledger (`web-search-improve`).

### 2. Setpoint

Bands are targets. Readings are dated observations; re-read every cycle with `run search-hub.setpoint-read` (7-day window unless stated). The newest stored `router.routing` run is a strategy-compare candidate arm from 2026-08-16; incumbent readings come from the PROBLEMS.md entry until a new incumbent run is stored.

| Row | Sensor | Band | Today (2026-09-02) |
|---|---|---|---|
| routing-precision | `search-hub evals runs router.routing --limit 1` → `aggregate.routingPrecision` | ≥ 0.90 | incumbent 159/177 = 0.8983 (2026-08-16); newest stored run 0.8939 is the semantic candidate arm |
| e2e-pass-rate | same run → `aggregate.passRate` | ≥ 0.50 | incumbent 0.2373 (2026-08-16); candidate run stores no passRate |
| retrieval-recall | same run → `aggregate.retrievalRecall` | ≥ 0.85 (PRD OT-P0-005 recall@K) | candidate 0.0063; incumbent pending-baseline |
| provider-degradation | `search-hub metrics provider-degradation-rate --window last_7d` | ≤ 0.20 (every descriptor's `degraded_rate_max`) | 1679/12980 = 0.129 — in band |
| degraded-query-rate | `search-hub metrics degraded-query-rate --window last_7d` | ≤ 0.20 | 2678/3277 = 0.817 — out of band |
| federated-latency | `search-hub metrics federated-latency --window last_7d` → p95 | p95 ≤ 4000 ms (every descriptor's `p95_ms`) | p50 1202, p95 4339 — out of band |
| under-utilized-providers | `search-hub insights insights --window 7` → `retirement_candidates` | 0 | 0 in 7 d; 2 all-time (`content-desk.editorial-history`: zero hits across 315 routed calls) |
| zero-result-rate | same read → `zero_result_rate` | ≤ 0.10 | 1776/3277 = 0.542 — out of band |
| provider-reachability | `search-hub federation status` → `reachable == false` count | 0 | 0 of 29 — in band |
| eval-floors | `search-hub evals runs <suite_id> --limit 1` for every registered suite → `aggregate.passRate` vs the suite's floor | 0 suites below floor | 30 suites read; ≥ 19 below full pass; floors pending-baseline for every suite |
| external-friction | `run agent-manager.friction-digest` with inputs `scenario=search-hub`, `window_days=7` → `recurring_count` | 0 recurring fingerprints with owner confidence `manifest-derived` | 5 recurring across 19 episodes in the last 40 runs (all created 2026-09-02); top fingerprint `command-failure` on `search-hub query`, 4 occurrences, `manifest-derived` — out of band |

Report figure, not a setpoint row: `[S3]` leaves among rung-labelled leaves in the usage skill were 2 of 15 = 0.13 on 2026-09-02 (hand count). No sensor produces it; `skill-improvement-suggestions` E10 names the promotion candidates.

### 3. Sensors

Read every row through `run search-hub.setpoint-read` (contract: `.vrooli/program-runtime/setpoint-read.json`). Rows the program marks `unavailable` are read by hand only with the exact command in the table, and the hand reading is journaled as such. The three declared measures (`metrics federated-latency`, `degraded-query-rate`, `provider-degradation-rate`) are external to the router's own telemetry aggregation and outrank `insights` when they disagree.

For provider filings, `run search-hub.provider-quality-read` joins insights, providers list, federation status, and the newest eval run per suite into one per-provider table (`degraded_over_band`, `unreachable`, `under_utilized`, `suites_below_full_pass`). A filing quotes its row.

Fleet sensors every scenario has: `program-runtime bindings condition` for search-hub's bindings, and `run agent-manager.friction-digest` (inputs `scenario`, `window_days`) for `search-hub` commands. The 2026-09-02 digest names `search-hub query` command failures as the recurring fingerprint; the usage skill's troubleshooting row on the stale CLI base is the likely cause and the first thing to confirm.

### 4. Golden corpora

| Suite | Owner | Floor | Derivation |
|---|---|---|---|
| `router.routing` (composed from every provider's reviewed cases) | search-hub | pending-baseline; the guard is significance, not a floor | Compared arms must clear the held-out routing-precision guard with a paired CI lower bound above 0; recorded in PROBLEMS.md 2026-08-16 |
| `<provider>.primary` and `.starter` (30 suites, `search-hub evals list`) | the provider's owner | pending-baseline per suite | No suite records a floor. Derive each from two comparable `provider_direct` runs (`search-hub evals run <suite_id> --tier provider_direct`) and record it in the suite description through the owner |
| `search-hub.docs.primary` (2 cases) | search-hub | pending-baseline | newest run pass 0.5, routing precision 0.5, 2026-09-02; derive after a second run |

A `router.routing` incumbent run below its recorded precision is a stop for every other route. This skill never lowers a floor and never removes a case to reach one.

### 5. Actuators and ladder routing

`Actuator` rows are strategy or eval moves the agent running this skill performs in-cycle without a diff. `Filing` rows hand off: a work-ladder rung against search-hub, or a corpus-quality `report-bug` against a provider owner.

| Kind | Row out of band | Route | Sensor that should move |
|---|---|---|---|
| Filing | routing-precision below band and the run's `below` count exceeds its `unavailableCases` (the misses are graded misroutes; `evals show-run` names the corpus chosen) | Corpus-quality filing: `report-bug` against the provider owner with the case id, the run id, and the descriptor sentence that misrouted; ask for a description change in their `search.json`. Never edit it from here | routing-precision |
| Filing | routing-precision below band and `unavailableCases` is at least the `below` count (the denominator, not the router, is the loss) | Expand the eligible denominator: for each withheld provider, `provider-quality-read` names the reason (lifecycle, stale index, no evidence); file against the owner with that reason | routing-precision, eval-floors |
| Filing | e2e-pass-rate below band while routing-precision is in band | The loss is inside corpora, not routing. Take the `suites_below_full_pass` list and file one corpus-quality item per owner with `evals show-run <run_id>` per-case outcomes; ask them to run `evals validate <suite_id>` (labels live/hard/stale) first | e2e-pass-rate, eval-floors |
| Actuator, then Filing | retrieval-recall below band on the incumbent | `search-hub strategy compare` (guarded; refuses insignificant winners). If a candidate clears the guard, `strategy compare --apply`; if none does, W3 against search-hub: the bounded lexical-evidence fusion is the next lever, recorded with run ids | retrieval-recall |
| Filing | provider-degradation above band | `provider-quality-read` → `degraded_over_band` with `degradation_reasons`: `unreachable`/`timeout` → `report-bug` against the owner (their scenario is down or slow); `reranker_absent`/`reranker_unavailable` → W3 against search-hub's rerank fallback chain | provider-degradation |
| Filing | degraded-query-rate above band | Read `insights` `substrate_degradation_reasons`: reranker-caused → W3 against search-hub (TEI resource health, Ollama fallback); provider-caused → the row above | degraded-query-rate |
| Filing | federated-latency p95 above band | `insights` per-provider `latency_p95_ms`: one slow leg → file against its owner with the number; every leg slow → W3 against search-hub (per-provider timeout, fan-out bound) | federated-latency |
| Actuator, then Filing | under-utilized-providers above 0 | Curation: for each retirement candidate, `providers list` lifecycle; `experimental` or `fixture` with zero hits → file against the owner to fix the corpus or retire the declaration; `production` → the descriptor misroutes → corpus-quality filing with sample queries | under-utilized-providers |
| Filing | zero-result-rate above band | `insights` zero-result queries by provider: dominated by heartbeat or eval traffic → journal, no route; dominated by classifier picks that return nothing → routing-precision route; dominated by explicit `--type` misses → usage-skill token table (`skill-improvement-suggestions` on `search-hub`) | zero-result-rate |
| Filing, then Actuator | provider-reachability above 0 | `report-bug` against the unreachable provider's owner with `federation status` row; after they fix it, `search-hub federation repromote <provider_id>` once, journaled | provider-reachability |
| Filing, then Actuator | eval-floors: a suite below its recorded floor | Owner's corpus regressed: file with `evals compare <baseline_run> <new_run>` output. If the suite has reviewed candidates, `evals promote <suite_id> --case <ids>` only for cases the owner reviewed | eval-floors |
| Filing | external-friction recurring fingerprint | Read the fingerprint's episode (`agent-manager run episodes <run-id>`); a `search-hub query` command failure caused by the stale CLI base is W3 against search-hub's CLI; a misuse of `--type` tokens is `skill-improvement-suggestions` on the usage skill | external-friction |
| Filing | S3-share report figure below 0.25 | Promote the next recurring `[S1]` leaf: the capture pair (query-record note + pin) is the candidate; author it under `.vrooli/program-runtime/` and relabel the leaf | the report figure |

### 6. Anti-gaming

`improvement-do-and-dont` §1 and its three DON'T subheadings (tagged test, known-issue ledger, suppression) and §2 (the skeptic test) apply verbatim. Search Hub's own gaming moves, each worth zero credit and a review flag:

- Marking cases `unavailable` by hand, or editing a suite's cases, to raise pass rate. Only the eligibility snapshot inside a guarded comparison may withhold a case, and every withheld case stays visible in the run.
- Lowering a recorded floor, or deriving one from a run taken with the provider unreachable.
- `strategy compare --apply` or `evals sweep --apply` after the guard refused; re-registering a strategy to dodge the guard.
- `evals reap-orphans --confirm` on a suite that fails rather than one whose provider is gone.
- `federation repromote` on a provider nobody fixed, to clear graded-empty evidence.
- Narrowing `--window` until degraded-query-rate or zero-result-rate reads in band.
- Deleting junk negatives (`expect_no_strong_hit`) so a corpus stops leaking.
- Reading `provider-degradation` with `--provider-id` on the healthiest provider and reporting it as the fleet number.

### 7. Evidence

One `vrooli-memory journal note --kind work-record` per cycle:

```
--trigger  "search-hub improve cycle <n>: <row> <reading> vs <band>"
--approach "<route row text>"
--evidence "<before> -> <after> on <sensor command>; search-hub://eval-run/<run_id>"
--outcome  "<in band | filed <ref> against <owner> | applied <strategy> | unavailable: <reason>>"
```

A sensor unavailable for three cycles is a `scenarios/search-hub/docs/internal/PROBLEMS.md` entry with the three dated readings. Filings against provider owners use `report-bug` with the `provider-quality-read` row and the run id as the observation.

### 8. Stop rules

| Condition | Action |
|---|---|
| `router.routing` incumbent below its recorded precision | Only the corpus route runs this cycle |
| A guarded comparison refuses (`insignificant`) | Stop that route; journal the CI; do not re-run with a different `--strategies` set to fish |
| A row reads `unavailable` | Journal; do not estimate; after three cycles, PROBLEMS.md and W2 |
| More than five filings would go to one owner in a cycle | Stop; file one item with the list; do not flood |
| A route needs a grant (`refused_no_grant`) | Stop and request the grant through the session path |
| Every readable row in band for two consecutive cycles | Propose close-out to the operator; stop |
| The session's inference or delegation ceiling is reached | Stop; journal the ceiling and the row in progress; do not open a new session to continue |

### 9. Troubleshooting & Edge Cases

| Symptom | Likely cause | First check | Fix |
|---|---|---|---|
| CLI reads fail with `dial tcp 127.0.0.1:2026` while `vrooli scenario status search-hub` is healthy | The CLI's RPC client resolves a stale base; `--api-base` fixes only `/health` (observed 2026-09-02) | `vrooli scenario status search-hub` ports line | Read through `run search-hub.setpoint-read` (the kernel path works); file W3 against search-hub's CLI base resolution |
| `setpoint-read` routing rows come from a `strategy-compare:*` run | No incumbent run stored since the comparison | reading's `arm` field | Store an incumbent run: `search-hub evals run router.routing --tier federated --tag incumbent`; do not read the candidate as the incumbent |
| `metrics` bindings refuse `window="last_7d"` from a program | The measure window is a `TimeWindow` message | none | Pass `window={"token": "TIME_WINDOW_TOKEN_LAST_7D"}`; the CLI flag form is `--window last_7d` |
| `insights` and `metrics degraded-query-rate` disagree | Different windows (insights takes days or durations, metrics takes tokens) | the two `--window` values | Compare on the same span; the measure is the reading of record |
| `retirement_candidates` empty in 7 d but present all-time | Candidates need `minimum_sample_count` routed calls in the window | `insights --window 30` | Read the longer window for this row; journal which window you used |
| `provider-quality-read` shows `degradation_rate` 1.0 for a group | That scenario is stopped | `vrooli scenario status <scenario>` | Not a routing defect; file only if the scenario should be running |
| A suite's newest run has `pass_rate: null` | Smoke or reachability suite with no graded cases | `evals show-run <run_id>` | Exclude smoke suites from the floor row by design; journal the id |
