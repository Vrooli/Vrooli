---
name: "web-search"
description: "Answer external-world questions (versions, vendors, current events, third-party facts) by climbing web-search's ladder: the findings ledger first, then L0 raw hits, L1 cited synthesis, L2 fetch-and-read, L3 agentic research; record what answered so the ledger learns."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools"]
  tags: ["web-search", "research", "findings", "citations", "searxng", "ladder", "external-world", "learning-ledger"]
  icon: "globe"
  status: "active"
  revision: 2
  createdAt: "2026-09-02T00:00:00Z"
  updatedAt: "2026-09-02T20:00:00Z"
  requires:
    scenarios: ["web-search", "search-hub", "program-runtime"]
    commands: ["web-search search search", "web-search research l2", "web-search research l3", "web-search research status", "web-search findings search", "web-search findings use", "web-search findings add", "web-search findings supersede", "web-search findings flag", "web-search findings get", "web-search findings gc", "web-search disputes list", "web-search disputes resolve", "vrooli scenario status", "vrooli resource status", "vrooli-memory journal note"]
  learning:
    ledger: "findings"
    capture: "on novel outcome"
  origin:
    kind: "authored"
---
## Tools focus: Web Search

Answer questions about the world outside this repository — software versions, vendor facts, current events, "what is the latest X" — with the cheapest rung of web-search's ladder that yields a cited answer, and leave the findings ledger better than you found it. The ledger is this skill's memory: it replaces a vrooli-memory scope, so recall and capture are `web-search findings` verbs, not journal notes.

Required reading:
- `prompt-manager skill read search-hub` — project-internal questions belong there; this skill starts where the project's own corpora end.
- `path:scenarios/web-search/docs/reference/configuration.md` — the boot-time levers named in §4.

### 1. Scope

**In scope:** choosing the rung; reading the findings ledger before the live web; recording use, contradiction, and supersession; starting an L3 run and reading its status once.

**Out of scope:** questions about this project's code, scenarios, or history (`search-hub`); curating the ledger for its own health (`web-search-improve`); registering or tuning the search-hub providers `web-search.live` and `web-search.learnings`; editing the SQLite store by hand.

### 2. Before acting: recall from the ledger

Read the ledger before any live call. Findings decay with a 180-day half-life, so the date on a hit is part of the answer.

```
web-search findings search "<question>" --limit 5          [S1]
```

| Hit shape | Do |
|---|---|
| status `active`, not `weak`, confidence ≥ 0.5, retrieved within 180 days | Answer from the finding. Cite its citations. Go to §5. |
| status `disputed` | Read both cited sources. Answer with the "sources conflict" warning. Do not record use. Continue to L1 or L2 to resolve it; then §5. |
| status `active` but retrieved more than 180 days ago | Treat as a lead, not an answer. Continue to L1; if the live answer differs, §5 supersedes it. |
| `weak`, or no hit | Continue to the ladder. |

### 3. The ladder

```
                What does the answer need?
                          │
   ┌──────────────┬───────┴────────┬────────────────────┐
   URLs to open   one cited        page content read    multi-hop research
   (a list)       paragraph from   (docs, changelogs,   with reconciliation
                  snippets         comparisons)         into the ledger
   │              │                │                    │
   ▼              ▼                ▼                    ▼
   L0 [S1]        L1 [S1]          L2 [S1]              L3 [S1]
   web-search     web-search       web-search research  web-search research l3 "<q>"
   search search  search search    l2 "<q>" --top-n 3   then: web-search research
   "<q>"          "<q>"            [--capture]          status <id>
   --limit N      --synthesis
```

One step for L0–L2 with the ledger read folded in: `run web-search.research` with `query`, `effort` in `l0|l1|l2`, `mark_used` **[S3]**. Branch on the envelope:

| `status` | `errors[0].class` | Do |
|---|---|---|
| `ok` | — | Read `signals.answer_kind`: `stored_finding` → cite `signals.citations`, `signals.confidence`; `cited_synthesis` → cite `signals.citations`; `raw_hits` → open the URLs; `none` → climb one rung |
| `partial` | `rate_limited` | The governor declined the live call. Answer from `signals.stored_hits` if any; otherwise wait one minute and rerun once |
| `partial` | `upstream_degraded` | SearXNG engines were down for this call. Rerun at `l2` (page fetch does not need the engine list) or later |
| `partial` | `no_governed_binding` | `effort=l3` was asked. Start L3 by CLI (the L3 column above) |
| `unavailable` | `scenario_unreachable` | web-search is not running. `vrooli scenario status web-search`; do not answer from memory |
| `refused` | `no_grant`, `not_run_eligible` | Stop. Request the grant through the session path |
| `failed` | `invalid_input` | Fix `effort`, `limit` (1..50), or `top_n` (1..10) |

Rung rules:
- L0 and L1 never write to the ledger. L2 writes only with `--capture`. L3 captures by default.
- L1 and L2 abstain when sources are thin or conflict (`abstained: true`). An abstention is an answer: report "sources insufficient or disagree", do not climb to force a paragraph.
- L3 returns a run id and runs as an agent-manager run. Do not poll it. Journal the id (`vrooli-memory journal note "web-search l3 <id>: <question>" --kind work-record --trigger "<question>" --approach "research l3" --evidence "run <id>" --outcome "started; read status next session"`) and stop; read `web-search research status <id>` once at the start of the next session. web-search has no `research wait` verb (W1 against web-search), and L3 has no governed binding, so no program can start or wait on it for you.
- A `cached: true` L0/L1 result is at most `WEB_SEARCH_CACHE_TTL` (default 5 m) old; that is fresh enough for every question this skill covers.

### 4. In-use settings

Boot-time levers, set in the scenario's environment and read once at start (`configuration.md` §Tuning levers). A change needs `vrooli scenario stop web-search` then `start`; journal it as a work-record.

| Symptom | Setting move | Journal |
|---|---|---|
| `rate_limited` several times an hour under normal use | `WEB_SEARCH_GOVERNOR_CAPACITY` (default 60 calls per rolling minute) | `vrooli-memory journal note --kind work-record` with the declined count and the new value |
| Same query re-fetched within minutes across agents | `WEB_SEARCH_CACHE_TTL` (default 5m) | as above, with the observed repeat rate |
| L2 fetches abandon slow origins | `WEB_SEARCH_FETCH_TIMEOUT` (default 15s) | as above, with the origin |
| L3 converges before it has covered the question | `WEB_SEARCH_L3_MAX_LOOPS` (default 10) | as above, with the run id |
| L3 supersedes findings you would have flagged | `WEB_SEARCH_HIGH_CONFIDENCE_THRESHOLD` (default 0.75; raise to flag more, supersede less) | as above, with the finding ids |

Never change `WEB_SEARCH_DECAY_HALF_LIFE` to make stale findings look fresh; that lever belongs to `web-search-improve` with a recorded reason.

### 5. After acting, always: capture into the ledger

The ledger learns only from these verbs. Pick the first row that matches.

| What happened | Do |
|---|---|
| A stored finding answered and was correct | `web-search findings use <id>` [S1] (or `mark_used: true` on the program) |
| A stored finding was wrong or outdated and you found the current fact | `web-search findings add --claim "<fact>" --confidence <0..1> --query "<question>" --source manual --citations "url\|title,url\|title"` then `web-search findings supersede <old-id> --replacement <new-id> --reason "<why>"` [S1] |
| Two credible sources disagree and you could not resolve it | `web-search findings flag <id> --reason "<sources and the conflict>"` [S1]; the dispute queue (`web-search disputes list`) is the review surface |
| You resolved a dispute with new evidence | `web-search disputes resolve <id> --resolution keep\|supersede [--replacement <id>] --reason "<evidence>"` [S1] (CLI only; run-ineligible for programs by design) |
| An L2 answer is worth keeping and you ran without `--capture` | Rerun `web-search research l2 "<q>" --capture` [S1]; do not `findings add` a paraphrase of a synthesis you did not verify |
| L0/L1 answered a one-off question | Nothing. Raw hits and snippet syntheses are not findings |

Confidence in `findings add` is your estimate that the claim is true today. Use 0.9 only with two agreeing primary sources.

### 6. Debug order

1. `vrooli scenario status web-search` — stopped is the common cause of every "no results".
2. `web-search findings count --window last_7d` — proves the API answers and the store opens.
3. `web-search search search "test" --limit 1` — read `degraded` and `degraded_reason`: the governor ("rate-limited") versus SearXNG down.
4. `vrooli resource status searxng` — the upstream. If it is down, `web-search.learnings` still answers; only the live rungs are lost.
5. `web-search research l2 "test" --top-n 1` — the fetch stack; thin excerpts with browser escalation on point to browser-automation-studio.

### 7. Safety

- L0–L3 send the query to external engines and fetch external pages. Never put credentials, private paths, or customer data in a query.
- `findings prune --force` archives superseded rows; `findings gc` without `--dry-run` soft-retires decayed rows. Preview both with `--dry-run` first. Neither is this skill's job; they belong to `web-search-improve`.
- Never edit the findings SQLite file or the Qdrant collection directly; every mutation has a verb that keeps provenance.

### 8. Troubleshooting & Edge Cases

| Symptom | Likely cause | First check | Fix |
|---|---|---|---|
| `web-search: command not found` | The scenario CLI is not installed on this host | `ls ~/.vrooli/bin/web-search` | `vrooli scenario start web-search` installs it through setup; do not run `cli/` binaries by hand |
| Every rung returns `degraded: true`, reason names the budget | Governor exhausted (60 calls per rolling minute) | `web-search search search "test" --limit 1` after 60 s | Answer from stored hits; batch questions; §4 lever only with evidence |
| `degraded_engines` lists every engine | SearXNG resource down or its image stale | `vrooli resource status searxng` | Use `--type learning` via search-hub or wait; file `report-bug` against the resource if it stays down |
| `findings search` returns text matches only, `method: text` | Embeddings unavailable (Ollama or Qdrant down) | `vrooli scenario status web-search` health line | Answers are still valid; recall is lexical until the index reconciles |
| L1 `abstained: true` on a well-known fact | Snippets thin or in conflict | none | Run L2 with `--top-n 3`; abstention is honest, not a failure |
| L2 excerpts are near-empty for a JS-heavy site | Browser escalation off or browser-automation-studio unreachable | `WEB_SEARCH_BROWSER_ESCALATION` in the scenario env | Leave escalation on; file `report-bug` against browser-automation-studio if its capture fails |
| L3 `research status` still `running` when you read it next session | Agent-manager run still looping (max 10 loops by default) | `web-search research status <id>` once | Journal the second reading and stop; the hard timeout is agent-manager's; do not start a second run for the same question and do not loop on status |
| A finding you superseded still appears | `--include-archived` set, or the replacement id was wrong | `web-search findings get <id>` | Superseded rows are kept by design; check `superseded_by` on the old row |
| `run web-search.research` says `no_governed_binding` for `l3` | Expected: `research l3` is run-ineligible | none | Start L3 by CLI |
