---
name: "web-search-improve"
description: "Regulate web-search against its setpoint: findings surfaced and used rates, never-surfaced decay, live-versus-local routing ratio, cache and governor telemetry, provider lifecycle, and external friction. Routes each out-of-band row to a ledger curation move, a work-ladder rung, or an owner."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["practice"]
  tags: ["web-search", "improve", "self-improvement", "control-loop", "setpoint", "findings", "curation", "meta-optimization"]
  icon: "gauge"
  status: "active"
  revision: 2
  createdAt: "2026-09-02T00:00:00Z"
  updatedAt: "2026-09-02T20:00:00Z"
  requires:
    scenarios: ["web-search", "search-hub", "program-runtime", "prompt-manager", "vrooli-memory"]
    commands: ["web-search findings effectiveness", "web-search findings count", "web-search findings used-rate", "web-search findings never-surfaced", "web-search findings gc", "web-search findings flag", "web-search findings supersede", "web-search findings prune", "web-search disputes list", "web-search disputes resolve", "search-hub insights insights", "search-hub providers list", "search-hub evals runs", "search-hub evals run", "program-runtime programs submit", "prompt-manager skill read", "vrooli-memory journal note"]
  origin:
    kind: "authored"
---
## Practice focus: Web Search Improve

Regulate web-search — the live-web ladder and its self-curating findings ledger — against the setpoint below. The plant is the ledger's usefulness (do stored findings get surfaced and used, do decayed ones leave) and the live path's economy (cache, governor, routing share). This skill is read by an agent whose task is web-search itself; it never edits another scenario, it files.

Required reading:
- `prompt-manager skill read web-search` — the usage skill; every ledger verb this skill names is documented there.
- `prompt-manager skill read improvement-do-and-dont` — anti-gaming, cited by section in §6.
- `prompt-manager skill read scenario-work-ladder` — where code routes go.
- `prompt-manager skill read measures-adoption` — how a `pending_telemetry` row becomes a measure.

### 1. Focus and scope

**In scope:** the setpoint rows below; curation of the findings ledger (flag, supersede, gc, prune, dispute resolution) with evidence; filing ladder rungs against web-search; filing against search-hub for provider-lifecycle and routing reads.

**Out of scope:** answering research questions (usage skill); editing search-hub, SearXNG, or any provider owner's files; changing `WEB_SEARCH_DECAY_HALF_LIFE` or any lever without a dated reason in the journal; the usage skill's content.

### 2. Setpoint

Bands are targets. Readings are dated observations; re-read every cycle with `run web-search.setpoint-read`. Readings marked "web-search stopped" were taken while the scenario was down; they are unknown, not zero.

| Row | Sensor | Band | Today (2026-09-02) |
|---|---|---|---|
| surfaced-rate | `web-search findings effectiveness --limit 500 --include-disputed` → share of rows with `surfaced_count > 0` | ≥ 0.50 | unavailable — web-search stopped |
| used-rate | `web-search findings used-rate --window last_7d` | ≥ 0.20 | re-read from the windowed effectiveness measure |
| never-surfaced-share | `web-search findings never-surfaced --window last_7d` → count; divide only when a separately recorded findings count is available | ≤ 0.20 | re-read from the windowed measure |
| live-vs-local-ratio | `search-hub insights insights` → `web-search.live` `times_routed` ÷ `web-search.learnings` `times_routed` | falling: each 30 d window below the previous one; pending-baseline (OT-P2-002 states the direction, "reach the live web progressively less", not a number, and `insights` has no windowed read yet) | 281 ÷ 585 = 0.48 all-time — the reference point, not a band |
| cache-hit-rate | no measure; `SearchResponse.cached` exists per call only | ≥ 0.30 | pending_telemetry |
| budget-exhaustion | no measure; `degraded_reason` names the governor per call only | 0 governor declines per day | pending_telemetry |
| provider-lifecycle | `search-hub providers list` → lifecycle of `web-search.learnings` and `web-search.live` | learnings `production`; live `fixture` or better | learnings `experimental`, live `fixture` — out of band |
| capture-volume | `web-search findings count --window last_30d` (the declared measure) | rising; pending-baseline | unavailable — web-search stopped |
| external-friction | `run agent-manager.friction-digest` with inputs `scenario=web-search`, `window_days=7` → `recurring_count` | 0 recurring fingerprints | 0 recurring, 0 episodes for this scenario across the last 40 runs (all created 2026-09-02; the program reads `run_limit=40` runs) |

Report figure, not a setpoint row: `[S3]` leaves among rung-labelled leaves in the usage skill were 1 of 11 = 0.09 on 2026-09-02 (hand count). No sensor produces it; `skill-improvement-suggestions` E10 names the promotion candidates.

### 3. Sensors

Read every row through `run web-search.setpoint-read` (contract: `.vrooli/program-runtime/setpoint-read.json`). Rows the program marks `unavailable` remain unavailable; do not replace them with a hand-derived reading. `cache-hit-rate` and `budget-exhaustion` are unavailable by construction until web-search declares measures for them; that is the first route in §5, not a reason to estimate.

For the curation rows, `run web-search.findings-curate` reads the effectiveness ledger, `findings gc --dry-run`, and `disputes list` and returns proposals with evidence; it never mutates. External sensors outrank self-reported ones: search-hub's routed counts and the search-hub eval runs for `web-search.learnings.primary` are read from search-hub, not from web-search.

Fleet sensors every scenario has: `program-runtime bindings condition` for web-search's bindings, and `run agent-manager.friction-digest` (inputs `scenario`, `window_days`) for `web-search` commands.

### 4. Golden corpora

| Suite | Cases | Floor | Derivation |
|---|---|---|---|
| `web-search.learnings.primary` (`.vrooli/search.json` tests) | 3 reviewed positives (python, rust, go release findings) + 1 gibberish negative | pending-baseline | No floor recorded. The newest federated run (2026-09-02, `search-hub evals runs web-search.learnings.primary --limit 1`) met 1 of 4 while the provider was unreachable; a run taken with the provider down is not comparable and does not derive a floor |
| `web-search.live.primary` | 1 smoke reachability case | none by design | Smoke corpus; it proves reachability, not recall |

Derive the learnings floor from two comparable runs taken with web-search running (`search-hub evals run web-search.learnings.primary --tier provider_direct`), record the derivation in the suite's description, then a run below floor is a stop for every other route. This skill never lowers a floor.

### 5. Actuators and ladder routing

`Actuator` rows are ledger curation moves the agent running this skill performs in-cycle without a diff. `Filing` rows hand off: a work-ladder rung against web-search, a `measures-adoption` item, or `report-bug` against another owner.

| Kind | Row out of band | Route | Sensor that should move |
|---|---|---|---|
| Actuator | surfaced-rate below band and `web-search.learnings` `times_routed` ≥ 10 in the window (the classifier does route here; the findings themselves are not surfacing) | Curation: read `findings-curate` proposals of kind `gc`; run `web-search findings gc --dry-run`, confirm the ids match, then `web-search findings gc`. Decayed never-surfaced findings leave; the denominator shrinks honestly | surfaced-rate, never-surfaced-share |
| Filing | surfaced-rate below band and `web-search.learnings` `times_routed` < 10 in the window | The classifier does not route external-world questions to learnings. File `report-bug` against search-hub with the routed count and three sample queries from the usage skill's §2; do not edit the provider description from here (W1 against web-search's `.vrooli/search.json` if the description is the cause) | live-vs-local-ratio, surfaced-rate |
| Actuator | used-rate below band | Curation: proposals of kind `review-unused` — surfaced but never used, low effective confidence. For each: read the finding; if its claim is stale, `findings supersede` with a replacement from a fresh L2 run; if its claim is wrong, `findings flag --reason`; if it is fine, leave it (usage may lag) | used-rate |
| Actuator | never-surfaced-share above band | Curation: `findings gc` after `--dry-run`; if gc lists nothing while the share is high, the decay gate (2× half-life) is not reached — journal and wait; do not shorten `WEB_SEARCH_DECAY_HALF_LIFE` | never-surfaced-share |
| Filing | live-vs-local-ratio rising for two windows | Read `search-hub insights insights` routing reasons: explicit `--type web` callers are legitimate; classifier-routed live hits mean OT-P2-002 fired early. File W2 against web-search (evidence: routed counts) | live-vs-local-ratio |
| Filing | cache-hit-rate pending | `measures-adoption`: declare a `search cache-hit-rate --window` measure over the livesearch cache counters (W1: obligation) | cache-hit-rate |
| Filing | budget-exhaustion pending | `measures-adoption`: declare a `search governor-declines --window` measure (W1) | budget-exhaustion |
| Filing | provider-lifecycle out of band | Promote `web-search.learnings` to `production` in `.vrooli/search.json` only after two comparable passing eval runs (§4) and a routed count ≥ minimum_samples; `web-search.live` stays `fixture` by design (external, rate-limited). Promotion is a W0 contract change on web-search | provider-lifecycle |
| Filing | S3-share report figure below 0.25 | Promote the next `[S1]` leaf whose calls recur with the same shape: the capture table in the usage skill §5 (add + supersede as one program) is the candidate; author it under `.vrooli/program-runtime/` and relabel the leaf | the report figure |
| Filing | external-friction recurring fingerprint | Read the fingerprint's episode; if the command is web-search's, W3 here; if the fix is skill prose, `skill-improvement-suggestions` on the usage skill | external-friction |
| Filing | capture-volume falling for two cycles | L2 `--capture` and L3 runs are not happening. Read `disputes list` and `findings effectiveness` for the last capture date; if the usage skill is not being loaded, file `skill-improvement-suggestions` on `web-search`; if L3 runs fail, W3 against web-search with the run ids | capture-volume |
| Actuator | `disputes list` has entries older than 30 days | Curation: `web-search disputes resolve <id> --resolution keep\|supersede --reason` with the evidence from a fresh L2 run; never resolve without evidence | used-rate |

### 6. Anti-gaming

`improvement-do-and-dont` §1 and its three DON'T subheadings (tagged test, known-issue ledger, suppression) and §2 (the skeptic test) apply verbatim. Web-search's own gaming moves, each worth zero credit and a review flag:

- Recording `findings use` on findings that did not answer a question, to lift used-rate.
- Running `findings gc` or `findings prune --force` without a `--dry-run` whose ids you read, to shrink never-surfaced-share.
- Lengthening `WEB_SEARCH_DECAY_HALF_LIFE` so decayed findings read as trusted.
- Setting `web-search.learnings` to `production` in `search.json` without two comparable passing runs.
- Reading effectiveness without `--include-disputed` so disputed rows leave the denominator.
- Deriving the learnings floor from a run taken while the provider was unreachable.
- Adding findings by hand (`findings add`) to raise capture-volume without citations that a reviewer can open.

### 7. Evidence

One `vrooli-memory journal note --kind work-record` per cycle:

```
--trigger  "web-search improve cycle <n>: <row> <reading> vs <band>"
--approach "<route row text>"
--evidence "<before> -> <after> on <sensor command>; finding ids touched"
--outcome  "<in band | filed <ref> | curated <n> findings | unavailable: <reason>>"
```

A sensor unavailable for three cycles is a `scenarios/web-search/docs/internal/PROBLEMS.md` entry with the three dated readings. Filings against other owners (search-hub, the searxng resource, browser-automation-studio) use `report-bug` with the sensor row as the observation.

### 8. Stop rules

| Condition | Action |
|---|---|
| `web-search.learnings.primary` below its recorded floor | Only the corpus route runs this cycle (re-run with the provider up; then curation) |
| A row reads `unavailable` because web-search is stopped | Journal; do not start the scenario from an improve cycle without the operator; re-read next cycle |
| A curation move would touch more than 25 findings | Stop; journal the proposal list; ask the operator |
| A route needs a grant (`refused_no_grant`) | Stop and request the grant through the session path |
| Every readable row in band for two consecutive cycles | Propose close-out to the operator; stop |
| The session's inference or delegation ceiling is reached | Stop; journal the ceiling and the row in progress; do not open a new session to continue |

### 9. Troubleshooting & Edge Cases

| Symptom | Likely cause | First check | Fix |
|---|---|---|---|
| `setpoint-read` reports the ledger rows unavailable and the routing rows readable | web-search stopped; search-hub up | `vrooli scenario status web-search` | The reading is honest; journal it; the operator starts the scenario |
| `findings-curate` proposes zero moves while used-rate is low | Findings are surfaced and unused but not yet decayed below 0.5 | `web-search findings effectiveness --limit 20` | Wait a cycle; usage signals lag surfacing; do not lower `decayed_below` to manufacture proposals |
| `findings gc --dry-run` lists ids that `effectiveness` shows as surfaced | The two reads happened across a reconcile | Re-run both | Only gc ids present in both reads |
| `insights` shows `web-search.*` with `degradation_rate` 1.0 | Every routed call found the provider unreachable | `vrooli scenario status web-search` | Not a routing defect; the live-vs-local row is still readable from `times_routed` |
| `search-hub evals run web-search.learnings.primary` fails every positive | Finding ids in `search.json` cases were superseded or pruned | `web-search findings get <expect_id>` | Update the case ids to the replacement findings (W2 evidence repair); never delete the cases |
| `program-runtime` names `capture` as a protected name in a curation program | `capture` is a runtime verb | preflight diagnostics | Rename the variable; the shipped programs already do |
