---
name: "vrooli-memory"
description: "Use vrooli-memory as the durable, scoped memory behind a skill's learning spine: create a scope with a facet vocabulary, journal one structured entry per attempt, recall by wake or by query, and curate with pins, supersession, corrections, and dry-run classification rules."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools"]
  tags: ["vrooli-memory", "source-ledger", "memory", "learning-spine", "scope", "recall", "journal", "pins", "rules"]
  icon: "brain"
  status: "active"
  revision: 2
  createdAt: "2026-09-02T00:00:00Z"
  updatedAt: "2026-09-02T20:00:00Z"
  requires:
    scenarios: ["vrooli-memory"]
    commands: ["vrooli-memory scopes create", "vrooli-memory scopes list", "vrooli-memory journal note", "vrooli-memory recall wake", "vrooli-memory recall recall", "vrooli-memory recall siblings", "vrooli-memory facets pin", "vrooli-memory facets supersede", "vrooli-memory facets correct", "vrooli-memory facets candidates", "vrooli-memory facets proposals", "vrooli-memory rules create", "vrooli-memory rules dry-run", "vrooli-memory rules enable", "vrooli-memory rules list", "vrooli-memory forest frontier"]
  origin:
    kind: "authored"
---
## Tools focus: Vrooli Memory

Use vrooli-memory when a skill needs to remember what happened across sessions in a form the next agent can recall under a fixed line budget. It is the mechanics behind every learning spine (`path:docs/agent-system/SKILL_AUTHORING.md` §"The learning spine"): other skills cite this one and say only which scope and which entry kinds they use. The engine underneath is source-ledger; the append-only journal is the sole authority, and every derived view is rebuildable.

### 1. Scope

**In scope:** scopes and their facet vocabularies; journal entries; recall by wake and by query; curation: pins, supersession, facet corrections, classification rules with a dry run; reading the compaction frontier.

**Out of scope:** what a given skill should remember (its own decision tree decides); harness memory import (`harness` group; see `path:scenarios/vrooli-memory/README.md`); editing or deleting entries (the journal is append-only; supersede instead); the shared `agent-memory` scope's wake policy (operator-owned).

### 2. The decision tree

```
I need memory for a skill or a program
│
├─ Does the scenario already declare a scope? (service.json skills.learning.scope)
│   ├─ no  → create it once (§3 step 1), then declare it                          [S1]
│   └─ yes → continue
│
├─ Before acting: what do I need to know?
│   ├─ "what usually matters here"  → recall wake --scope <scope>                 [S1]
│   ├─ "what happened with <subject>" → recall recall "<subject>" --scope <scope> --limit 5   [S1]
│   └─ "what else did this run record" → recall siblings <entry-id>               [S1]
│
├─ After acting: one entry per attempt, success or not                             [S1]
│     journal note --scope <scope> --kind <kind> --trigger --approach --evidence --outcome
│
└─ Curating what recall keeps showing
    ├─ an entry was right for the third time            → facets pin <entry-id>                       [S1]
    ├─ an entry's advice stopped working                → facets supersede <old> --replacement-entry-id <new>   [S1]
    ├─ an entry was filed under the wrong facet         → facets correct <entry-id> --facet <facet>     [S1]
    ├─ the same body pattern keeps needing one facet    → rules create → rules dry-run → rules enable   [S1]
    ├─ pins pile up                                     → facets proposals; resolve the redundant ones  [S1]
    └─ wake is crowded with old entries                 → forest frontier; raise the frontier only with a reason  [S0]
```

### 3. Setup, once per scenario

1. Create the scope with the vocabulary the skill's entry kinds will use as facets. Budgets are per-scope and bound the wake output, so growth of the corpus never grows the prompt.

   ```
   vrooli-memory scopes create <scenario>-usage --label "<Scenario> usage learnings" \
     --wake-budget 48 --max-entry-lines 2 \
     --facets-json '[{"id":"<scenario>-site","label":"Site"},{"id":"<scenario>-flow","label":"Flow"},{"id":"<scenario>-failure","label":"Failure"}]'
   ```

   Each facet is an object: `id` and `label` are required; `guidance`, `retention_policy`, `compaction_eligible`, and `resident_budget` are optional. A bare string array is rejected. Facet ids are unique across ALL scopes, so prefix them with the scenario (`bas-site`, not `site`); a bare id that another scope already owns fails with a unique-constraint error. Facets are fixed at creation: there is no add-facet verb.

2. Declare it in `scenarios/<scenario>/.vrooli/service.json` under `skills.learning.scope`, and in the usage skill's frontmatter `metadata.learning`.
3. Add starter rules for facets that a body pattern identifies deterministically (a selector reference, a workflow id). Create disabled, dry-run against the current corpus, read the samples, then enable.

   ```
   vrooli-memory rules create --id <scenario>-selector --scope <scenario>-usage --facet <scenario>-selector --body-pattern '@selector/[A-Za-z0-9_.]+'
   vrooli-memory rules dry-run <scenario>-selector --scope <scenario>-usage
   vrooli-memory rules enable <scenario>-selector --scope <scenario>-usage
   ```

`scope-bootstrap` (`run vrooli-memory.scope-bootstrap`) does step 3 idempotently and creates a facet-less scope when none exists. The facet vocabulary is a CLI-local flag with no program binding today, so step 1 stays a CLI command [S1]; the program reports `facets_not_settable` rather than creating a scope without the vocabulary you asked for.

### 4. Writing an entry

One entry per attempt. The four work-record fields are the contract the next reader relies on; fill all four.

| Field | Content | Example |
|---|---|---|
| `--trigger` | What was asked, with the subject named | `vendor-x invoices-export: smoke` |
| `--approach` | The command or program and the settings that mattered | `run bas.smoke-flow; wait_for=networkidle; profile=vendor-x-sso` |
| `--evidence` | Ids and references, never bodies | `execution exec_…; step 6 selector_not_found` |
| `--outcome` | The result, then one line for next time | `failed: selector_not_found; next time: registry entry renamed` |

Rules:
- `--kind` is one of the entry kinds the calling skill declares (for example `task-record`, `site-note`, `workflow-verdict`, `work-record`). Do not invent a kind per entry.
- Keep `body` to two lines; the scope's `max-entry-lines` truncates the wake view, not the journal.
- Write after the outcome is known, never mid-action. A program writes in its `report` phase only (`path:scenarios/program-runtime/docs/guides/program-contracts.md` §"Memory in programs").
- Never record a dead dependency as a failure of the task. Record `unavailable: <reason>` in `--outcome`.

### 5. Reading

| Need | Command | Read it as |
|---|---|---|
| Ambient set for any task in the scope | `vrooli-memory recall wake --scope <scope>` | Pinned entries first, then facet-ranked entries, within `wake-budget` lines |
| Targeted | `vrooli-memory recall recall "<subject>" --scope <scope> --limit 5` | Ranked hits with scores; treat a score below the caller's threshold as no match |
| Same run's other entries | `vrooli-memory recall siblings <entry-id>` | Context for one attempt |

Human-first output is the default. Use `--json` only inside a program or when the caller must branch on ids.

### 6. In-use settings

| Symptom | Move | Journal |
|---|---|---|
| Wake shows stale advice that still ranks | `facets supersede <old> --replacement-entry-id <new> --scope <scope>` | the pair of ids |
| A confirmed entry keeps falling below the frontier | `facets pin <entry-id> --scope <scope>` | why it is standing |
| Pins exceed what wake can show | `facets proposals --scope <scope>`, resolve redundant ones | which were merged |
| New entries land in the wrong facet | `rules create` + `rules dry-run` + `rules enable` | the rule id and its match count |
| Wake output is too long for the caller | lower `--wake-budget` at scope creation, or pin fewer | the budget chosen |

### 7. Safety

- The journal is append-only. There is no delete; supersede.
- Do not write secrets, bodies of fetched pages, or transcripts into entries; write ids and references.
- Do not create a rule enabled. Dry-run first, read the samples, then enable.
- Do not write to another scenario's scope. A cross-scenario fact goes to the shared `agent-memory` scope with `--kind work-record`.

### 8. Output expectations

A skill that uses this one names its scope and its entry kinds and nothing else about memory mechanics. A program that uses it declares `memory` in its contract (scope, `reads_in: collect`, `writes_in: report`, entry kinds). Every attempt leaves exactly one entry.

### 9. Troubleshooting & Edge Cases

| Symptom | Likely cause | First check | Fix |
|---|---|---|---|
| `journal note` fails with an embedding error | Ollama or the embedding provider is down; the note is queued | `vrooli-memory journal retry-embeddings --help`; `vrooli scenario status vrooli-memory` | The entry is journaled; embeddings replay later with `journal retry-embeddings`. Do not re-append |
| `recall wake` returns nothing for a new scope | No entries yet, or no pins and the frontier is empty | `vrooli-memory scopes list` | Expected on day one; write entries; wake fills as the frontier forms |
| `recall recall` scores are all low | Query subject differs from how entries name it | read three entries' bodies | Name subjects the same way in `--trigger` every time (site then flow) |
| A rule dry-run matches everything | `--body-pattern` too broad | the dry-run samples | Tighten the pattern; never enable a rule whose samples include unrelated kinds |
| `scopes create` fails with `UNIQUE constraint failed: facet_definitions.id` | A facet id is already owned by another scope | `vrooli-memory facets list --scope <other>` | Prefix facet ids with the scenario and create again |
| `rules list --scope X` and `facets list --scope X` return the same rows for every scope, including a scope that does not exist | Both listings ignore `--scope` and answer for `agent-memory` (verified 2026-09-02; filed against vrooli-memory) | `vrooli-memory rules list --scope no-such-scope --json` returns rows | Read `--json` and filter on the row's own `scope` field. A program that trusts the flag will treat another scope's rules as its own |
| `scopes create` says the scope exists | Bootstrap ran before | `scopes list --json` | Reuse it; budgets are changed with `source-ledger policy set`, not by re-creating |
| Health shows degraded | A dependency (embeddings, qdrant) is degraded | `vrooli scenario status vrooli-memory` | Reads and journal appends still work; classification and embeddings replay when it recovers |
