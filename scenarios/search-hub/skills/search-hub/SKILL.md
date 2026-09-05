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
  revision: 1
  createdAt: "2026-09-02T00:00:00Z"
  updatedAt: "2026-09-02T00:00:00Z"
  requires:
    scenarios: ["search-hub", "program-runtime", "vrooli-memory"]
    commands: ["search-hub query query", "search-hub providers list", "search-hub federation status", "search-hub insights insights", "search-hub status", "search-hub configure", "vrooli-memory recall wake", "vrooli-memory recall recall", "vrooli-memory journal note", "vrooli-memory facets pin", "vrooli-memory facets supersede", "vrooli scenario status"]
  learning:
    scope: "search-hub-usage"
    capture: "on novel outcome"
  origin:
    kind: "authored"
---
## Tools focus: Search Hub

Find what this project already knows, has, or can do before building or asking: one federated query across every registered corpus, routed by the question's shape, at the effort the question deserves. Search Hub is a thin router: the answer's quality is the provider corpus's quality, so this skill teaches how to aim, how to read a verdict, and when to stop and go elsewhere.

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
| Shape unclear, or spans buckets | classifier | none: let the router classify | `search-hub query query "<q>" --explain` [S1] |
| One scenario's own leaves only | any | `--group <scenario>` | `search-hub query query "<q>" --group <scenario>` [S1] |

Effort, then one step:

| Effort | Selector | `--limit` | When |
|---|---|---|---|
| fast | classifier, or your `--type` set | 5 | a lookup you expect to hit once |
| standard | classifier with `--explain`, or `--type` | 10 | the default for design and debugging questions |
| deep | `--all` (every active provider) | 20 | you must prove absence, or the shape spans three buckets |

`run search-hub.research` with `query`, `effort` in `fast|standard|deep`, optional `types` (a list), `group`, `scope` **[S3]**. Branch on the envelope:

| `status` | `signals.verdict` / `errors[0].class` | Do |
|---|---|---|
| `ok` | `answered` | Read `signals.ranked`; `providers_hit` says which corpus answered; `routing_explanation` says why |
| `ok` | `zero_result` | Corpora were searched, nothing matched. Rephrase in the corpus's own vocabulary (a doc title, a command name), or add `--all`; if `signals.escalate_to_web` is true and the fact is external, go to web-search |
| `ok` | `no_provider_selected` | No corpus was searched: the classifier chose nothing or your tokens match no provider. Use explicit tokens from §3 |
| `partial` | `degraded_leg` | Some rows are valid; read `routing_degrade_reason`: `reranker_absent` or `reranker_unavailable` → rows are grouped by provider, not fused, still usable; `unreachable`, `timeout`, `http_error` → one provider's leg is missing; note the provider for §5 |
| `unavailable` | `scenario_unreachable` | Search Hub is down (§6). Do not conclude absence |
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
| `query-record` | A shape → tokens → verdict you will meet again | `vrooli-memory journal note "<shape>: --type <tokens> -> <verdict>, best provider <id>" --scope search-hub-usage --kind query-record --trigger "<question>" --approach "<tokens/effort>" --evidence "<providers_hit, verdict>" --outcome "<answered \| escalated \| dead end>"` [S1] |
| `provider-note` | A provider answered off-topic, timed out twice, or needs its own vocabulary | `vrooli-memory journal note "<provider_id>: <what to know>" --scope search-hub-usage --kind provider-note --trigger ... --approach ... --evidence ... --outcome ...` [S1] |

Curation inside the tree: a `query-record` confirmed on a third distinct question → `vrooli-memory facets pin <entry-id> --scope search-hub-usage` [S1]; advice that stopped working → `vrooli-memory facets supersede <entry-id> --scope search-hub-usage --replacement-entry-id <new-id>` [S1]; a repeated shape that should classify itself → `run vrooli-memory.scope-bootstrap` with a `rules` entry and read its dry-run before enabling [S3] (contract: `scenarios/vrooli-memory/.vrooli/program-runtime/scope-bootstrap.json`).

### 6. In-use settings and debug order

| Symptom | Move | Journal |
|---|---|---|
| Rows from one noisy provider crowd the list | Narrow `--type` to the bucket's tokens, or `--group <scenario>` | `provider-note` naming the provider |
| Zero rows on a shape that used to answer | `--all --limit 20` once; if still zero, `search-hub federation status` | `query-record` with the verdict |
| CLI reports `Unable to reach the search-hub API` while `vrooli scenario status search-hub` is healthy | `search-hub configure api_base http://localhost:<API_PORT>` with the port from `vrooli scenario status search-hub`; observed 2026-09-02: the CLI's RPC client dialed `127.0.0.1:2026` while the API listened on 19157, and `--api-base` corrected only `/health` | `provider-note` on `search-hub` with both ports; file `report-bug` against search-hub |
| A variant instance is under test | `search-hub --instance shadow query query "<q>"` | none |

Debug order: `vrooli scenario status search-hub` → `search-hub status` (dependencies: database, federation, ollama, qdrant) → `search-hub federation status` (per-provider reachability, classifier and reranker availability) → `search-hub insights insights --window 1` (today's zero-result rate, degradation reasons) → repeat the query with `--explain`.

### 7. Safety

- This skill is read-only. `providers register|remove`, `evals run|sweep --apply|generate --apply|promote|reap-orphans --confirm`, `federation repromote`, `strategy compare --apply`, and `embedding` mutations belong to `search-hub-improve` with evidence; never run them to make a query work.
- `--type web` reaches the external, rate-limited live web through web-search's governor. Never put credentials or private data in a query that may route there.
- Memory notes name providers and shapes, never result bodies.

### 8. Troubleshooting & Edge Cases

| Symptom | Likely cause | First check | Fix |
|---|---|---|---|
| `--type learning` returns zero and `federation status` shows `web-search.learnings` unreachable | web-search stopped | `vrooli scenario status web-search` | The ledger is unavailable, not empty; do not conclude the fact is unknown |
| Every result carries `degraded` with `reranker_absent` | TEI reranker down and the Ollama fallback also down | `search-hub status` | Rows are grouped by provider; ranking across providers is not comparable; read per-group |
| `--group <scenario>` returns zero rows | The scenario declares no `search.json` providers, or they are `capability_gap` | `search-hub providers list --state capability_gap` | Use the bucket's fleet tokens instead |
| `no_provider_selected` on a clear question | Classifier (Ollama) down; explicit tokens still work | `search-hub status` ollama line | Use §3 tokens explicitly |
| A `--type` token you copied from a doc returns nothing | The token is not declared by any provider. `program`, `skills`, and `docs` are common stale guesses; the registry declares `library`, `skill`, and `doc`. | `search-hub providers list --json` and read `.providers[].type` | Use a token from that list. It is the vocabulary of record, and `scenarios/program-runtime/.vrooli/search.json` declares the `library` provider. |
| Program call fails: "no determinable primary response field" | `rows=` omitted | none | Add `rows="ranked"` |
| Program call fails: proto syntax error naming your type string | `type="a,b"` string | none | Pass `type=["a", "b"]` |
| `insights` shows a high zero-result rate but your queries answer | Fleet-wide heartbeat queries dominate the window | `search-hub insights insights --window 1` | Not your defect; file nothing from the usage skill |
