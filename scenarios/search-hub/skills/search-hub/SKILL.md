---
name: "search-hub"
description: "Project research at effort levels: map a question's shape to a Search Hub bucket (DO, REUSE, KNOW, STATE) and the real --type tokens, pick fast, standard, or deep, read degraded, zero-result, and unavailable verdicts correctly, escalate to web-search only for external-world facts, and record what routed well in the search-hub-usage memory scope."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools"]
  tags: ["search-hub", "search", "federated", "routing", "research", "recall", "buckets", "effort-levels", "project-research"]
  icon: "search"
  status: "active"
  revision: 3
  createdAt: "2026-09-02T00:00:00Z"
  updatedAt: "2026-11-05T01:16:00Z"
  requires:
    scenarios: ["search-hub", "program-runtime", "vrooli-memory"]
    commands: ["search-hub query query", "search-hub providers list", "search-hub federation status", "search-hub federation provider-status", "search-hub insights insights", "search-hub status", "search-hub configure", "vrooli-memory recall wake", "vrooli-memory recall recall", "vrooli-memory journal note", "vrooli-memory facets pin", "vrooli-memory facets supersede", "vrooli scenario status"]
  learning:
    scope: "search-hub-usage"
    capture: "on novel outcome"
  origin:
    kind: "authored"
---
## Tools focus: Search Hub

Find what this project already knows, has, or can do before building or asking: one federated query across registered corpora, routed by the question's shape, at the effort the question deserves. Search Hub is a thin router: the answer's quality is the provider corpus's quality, so this skill teaches how to aim, how to read a verdict, and when to stop and go elsewhere.

Automatic provider selection is distinct from the retired interactive LLM classifier. The active strategy can use semantic provider-description search, lexical evidence, and cross-encoder reranking; a fallback or degraded leg is reported in routing metadata. Read the route explanation before interpreting an empty or partial result. An explicit `--type` bypasses automatic provider selection, but a selected retrieval provider may still use its own model or embeddings. A Program Runtime `inference_calls` budget does not account for model work performed inside Search Hub or its providers.

Required reading:
- `prompt-manager skill read vrooli-memory` — mechanics of the `search-hub-usage` scope (recall, capture, pins, rules). Not restated here.
- `prompt-manager skill read web-search` — where external-world questions go after this skill's KNOW bucket says no.

### 1. Scope

**In scope:** choosing bucket, `--type` tokens, depth and limit; reading routing verdicts; escalating; calling Search Hub from a program; learning which shapes route well.

**Out of scope:** registering, tuning, or repromoting providers; running or promoting evals; strategy comparison (`search-hub-improve`); the contents of any provider's corpus (its owner).

### 2. Before acting: recall

```
vrooli-memory recall wake --scope search-hub-usage                          [S1]
vrooli-memory recall recall "<question shape or provider>" --scope search-hub-usage --limit 5
```

Apply a `query-record` that matches the question's shape (its `--type` set and verdict) before choosing; apply a `provider-note` before trusting a provider it names. Mechanics: `vrooli-memory` skill.

### 3. From question shape to bucket and tokens

Tokens are the `type` values declared in `scenarios/*/.vrooli/search.json` and
served by the provider registry. Tokens absent from every provider return zero
rows silently, so copy them from this table.

| The question sounds like | Bucket | `--type` tokens (provider) | Rung |
|---|---|---|---|
| "How do I …", "which command / skill / action / binding / workflow" | DO | `command` (cli-health), `skill`, `action` (prompt-manager), `binding` (program-runtime), `workflow.flow` (workflow-health) | `search-hub query query "<q>" --type command,skill,action` [S1] |
| "Is there already something that does …", "what program / fragment / surface exists" | REUSE | `library` (program-runtime programs), `scenarios`, `dependency`, `resources` (scenario-dependency-analyzer), `surface` (ui-health), `workflow.fragment`, `workflow.test` | `--type library,scenarios,surface` [S1] |
| "How does X work", "what did we decide / learn", "where is the contract" | KNOW | `doc` (knowledge-observatory, search-hub, template-manager), `code`, `contract` (code-facts), `domain` (architecture-cartographer), `record` (source-ledger, swarm-manager), `signal`, `editorial-history` | `--type doc,code,contract,domain` [S1] |
| "What is the state of X", "which measure / requirement / debt / initiative" | STATE | `measure` (measures-health), `requirements` (business-health), `record` (template-manager debt), `initiative` (swarm-manager) | `--type measure,requirements,initiative` [S1] |
| An external-world fact (a version, a vendor, an event) | KNOW → external | `learning` (web-search.learnings, stored cited findings) first; `web` (web-search.live) only explicitly — it never joins default routing | `--type learning` then `--type web` [S1]; then `prompt-manager skill read web-search` [S0] |
| Shape unclear, or spans buckets | automatic selection | none: let Search Hub select providers | `search-hub query query "<q>" --explain` [S1] |
| One scenario's own leaves only | any | `--group <scenario>` | `search-hub query query "<q>" --group <scenario>` [S1] |

Effort, then one step:

| Effort | Provider selection | `--limit` | When |
|---|---|---|---|
| fast | automatic selection, or your `--type` set | 5 | a lookup you expect to hit once |
| standard | automatic selection with `--explain`, or `--type` | 10 | the default for design and debugging questions |
| deep | `--all` (every active provider) | 20 | you must prove absence, or the shape spans three buckets |

`run search-hub.research` with `query`, `effort` in `fast|standard|deep`, optional `types` (a list), `group`, `scope` **[S3]**. For the repeated task-skill lookup, set `types=["skill"]` and `include_skill=true`; add `include_discover=true` when Prompt Manager discovery is part of the sequence. Branch on `signals.skill_lookup.status` and `truncated` before using returned content. For the repeated lookup-and-record sequence, add `include_discover=true capture_note=true`; this explicitly writes one note containing only the verdict and result/discovery counts, never the query or returned content. Check `signals.memory_capture.status` before treating the note as recorded. To safely repeat a query after a lifecycle check, set `repeat_after_status=true`; the program performs one query, reads only Search Hub's lifecycle status, and repeats the identical query once only when status is `running`. Check `signals.status_check` and `first_query`; a non-running/unavailable status skips the repeat. Branch on the envelope:

| `status` | `signals.verdict` / `errors[0].class` | Do |
|---|---|---|
| `ok` | `answered` | Read `signals.ranked`; `providers_hit` says which corpus answered; `routing_explanation` says why |
| `ok` | `zero_result` | Corpora were searched, nothing matched. Rephrase in the corpus's own vocabulary (a doc title, a command name), or add `--all`; if `signals.escalate_to_web` is true and the fact is external, go to web-search |
| `ok` | `no_provider_selected` | No corpus was searched: automatic selection found no eligible provider, or the explicit tokens match none. Read the route explanation and provider state; use valid explicit tokens from §3 to bypass automatic selection |
| `partial` | `degraded_leg` | Some rows are valid; read `routing_degrade_reason` and the provider line `▸ <provider> — degraded: <reason>` in the output. `reranker_absent` or `reranker_unavailable` means rows are grouped by provider, not fused; `unreachable`, `timeout`, or `http_error` means that provider's leg is missing. Preserve the provider and reason when you report partial results. |
| `unavailable` | `scenario_unreachable` or `degraded_leg` with no hits | Search Hub is down or a selected provider did not answer. Do not conclude absence or treat the query as a completed zero; preserve the reported provider reason and choose an explicitly safe fallback if one is authorized |
| `refused` | `no_grant`, `not_run_eligible` | Stop; request through the session path |
| `failed` | `invalid_input` | `effort` outside the enum, or `types` not a list of strings |

### 4. Calling Search Hub from a program

Two forms; both return a bounded Handle (`count()`, `head(n)`, `meta()`):

```python
recall("<intent>", depth="fast", rows="ranked")                # one line; depth="deep" widens
search_hub.query.query(text="<q>", type=["doc", "skill"],       # full control
                       limit=10, explain=True, rows="ranked")   # all=True for --all; group="<scenario>"
```

Rules verified 2026-09-02: `rows="ranked"` is required (the response has several repeated fields; without it the call fails closed); `type` is a list of strings — a comma-joined string is refused by the proto encoder; routing meta (`corporaSearched`, `degraded`, `routingDegradeReason`, `partial`, `latencyMs`, `selectorLeg`, `rerankerLeg`) is in `meta()`, not in rows.

### 5. After acting, always: capture

One note per novel outcome (a shape you had not routed before, a verdict you did not expect, a provider that misled you). Skip notes for a repeat of a recorded shape with the same verdict.

| Kind | When | Command |
|---|---|---|
| `query-record` | A shape → tokens → verdict you will meet again | Call `search-hub.research` with `include_discover=true capture_note=true`; inspect `signals.memory_capture`. [S3] |
| `provider-note` | A provider answered off-topic, timed out twice, or needs its own vocabulary | `vrooli-memory journal note "<provider_id>: <what to know>" --scope search-hub-usage --kind provider-note --trigger ... --approach ... --evidence ... --outcome ...` [S1] |

Curation inside the tree: a `query-record` confirmed on a third distinct question → `vrooli-memory facets pin <entry-id> --scope search-hub-usage` [S1]; advice that stopped working → `vrooli-memory facets supersede <entry-id> --scope search-hub-usage --replacement-entry-id <new-id>` [S1]; a repeated shape that should classify itself → `run vrooli-memory.scope-bootstrap` with a `rules` entry and read its dry-run before enabling [S3] (contract: `scenarios/vrooli-memory/.vrooli/program-runtime/scope-bootstrap.json`).

### 6. In-use settings and debug order

| Symptom | Move | Journal |
|---|---|---|
| Rows from one noisy provider crowd the list | Narrow `--type` to the bucket's tokens, or `--group <scenario>` | `provider-note` naming the provider |
| Zero rows on a shape that used to answer | `--all --limit 20` once; if still zero, `search-hub federation status` | Use `search-hub.research` with `include_discover=true capture_note=true` only when recording this result is useful and authorized |
| CLI reports `Unable to reach the search-hub API` while `vrooli scenario status search-hub` is healthy | `search-hub configure api_base http://localhost:<API_PORT>` with the port from `vrooli scenario status search-hub`; observed 2026-09-02: the CLI's RPC client dialed `127.0.0.1:2026` while the API listened on 19157, and `--api-base` corrected only `/health` | `provider-note` on `search-hub` with both ports; file `report-bug` against search-hub |
| A variant instance is under test | `search-hub --instance shadow query query "<q>"` | none |

Debug order: `vrooli scenario status search-hub` → `search-hub status` (dependencies: database, federation, ollama, qdrant) → `search-hub federation status` (per-provider reachability, active selector and reranker legs; the classifier-availability field is legacy and remains false) → `search-hub insights insights --window 1` (today's zero-result rate, degradation reasons). When the task calls for the same query again after checking lifecycle state, use `search-hub.research` with `repeat_after_status=true`; it records the first result's counts and does not repeat when the scenario is unavailable.

When one provider has already been selected for the next action and its health can change that action, use `search-hub federation provider-status <provider_id>`. This resolves only that exact active provider's scenario route and reads its declared status/freshness endpoint when available. It does not sweep the fleet or execute a search; exercise the selected provider for the actual task after reading its status. Use `federation status` only when the question is federation-wide. If Search Hub itself is unavailable, use the documented direct fallback for the task (for repository/topic search, `rg` over only the approved paths); keep Search Hub coverage `unknown` rather than treating fallback search as proof of complete federation coverage.

### 7. Safety

- Queries and discovery are reads. `capture_note=true` performs one explicit append-only Memory write with verdict/counts only; do not enable it unless that write is appropriate and authorized. `providers register|remove`, `evals run|sweep --apply|generate --apply|promote|reap-orphans --confirm`, `federation repromote`, `strategy compare --apply`, and `embedding` mutations belong to `search-hub-improve` with evidence; never run them to make a query work.
- `--type web` reaches the external, rate-limited live web through web-search's governor. Never put credentials or private data in a query that may route there.
- Memory notes name providers and shapes, never result bodies.

### 8. Troubleshooting & Edge Cases

| Symptom | Likely cause | First check | Fix |
|---|---|---|---|
| `--type learning` returns zero and `federation status` shows `web-search.learnings` unreachable | web-search stopped | `vrooli scenario status web-search` | The ledger is unavailable, not empty; do not conclude the fact is unknown |
| Every result carries `degraded` with `reranker_absent` | TEI reranker down and the Ollama fallback also down | `search-hub status` | Rows are grouped by provider; ranking across providers is not comparable; read per-group |
| `--group <scenario>` returns zero rows | The scenario declares no `search.json` providers, or they are `capability_gap` | `search-hub providers list --state capability_gap` | Use the bucket's fleet tokens instead |
| `no_provider_selected` on a clear question | Automatic selection found no eligible provider, or the requested token has no active provider; this status alone does not identify a classifier outage | Read the `--explain` route explanation, then inspect declarations with `search-hub providers list --json` and availability with `search-hub federation status` | Try an exact active `--type` token from §3 to bypass automatic provider selection; if that also fails, inspect provider reachability and the selected retrieval/reranker legs |
| A `--type` token you copied from a doc returns nothing | The token is not declared by any provider. `program`, `skills`, and `docs` are common stale guesses; the registry declares `library`, `skill`, and `doc`. | `search-hub providers list --json` and read `.providers[].type` | Use a token from that list. It is the vocabulary of record, and `scenarios/program-runtime/.vrooli/search.json` declares the `library` provider. |
| Program call fails: "no determinable primary response field" | `rows=` omitted | none | Add `rows="ranked"` |
| Program call fails: proto syntax error naming your type string | `type="a,b"` string | none | Pass `type=["a", "b"]` |
| `insights` shows a high zero-result rate but your queries answer | Fleet-wide heartbeat queries dominate the window | `search-hub insights insights --window 1` | Not your defect; file nothing from the usage skill |
