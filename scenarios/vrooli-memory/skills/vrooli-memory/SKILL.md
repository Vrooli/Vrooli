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
  revision: 3
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

For a registered learning operation, call its scenario program directly.
New programs use Program Runtime's nine `learn.*` verbs for task identity,
step trees, recall, choice, typed notes, outcomes, inference, fragments, and
delegation. The deprecated `learning_task` declaration is a migration shim;
do not author new contracts with it or add a second manual record around a
verb-based program.
To select an operation dynamically, run `vrooli-memory.run-task` with `operation`
and structured `inputs`. Only registered operations are eligible.

| Need | Shared program |
|---|---|
| Execute an existing learning operation | `vrooli-memory.run-task` |
| Inspect outcome and pending delivery | `vrooli-memory.inspect-task` with `attempt_id` and private `resume_token` |
| Requeue frozen capture without domain replay | `vrooli-memory.resume-task` with the same recovery capability |
| Prepare bounded advice for a custom workflow | `vrooli-memory.prepare-attempt` |
| Recommend within current allowed options using already-prepared advice | `vrooli-memory.choose-option`; caller still owns authorization and verification |
| Record an attempt and linked observations | `vrooli-memory.finish-attempt` |
| Read comparable outcome measurements | `vrooli-memory.compare-outcomes` |

Read the contract with `program-runtime library get <name> --json`.
Invoke it with `program-runtime library run <name> --input key=value`.
Program authors compose these through `lib.vrooli_memory.<name_with_underscores>`.
The shared workflow contracts live in `path:scenarios/vrooli-memory/.vrooli/program-runtime/README.md`.

### Program contracts

Use these owner-routed programs when the memory task matches a declared shape:

| Program | Purpose | Required inputs |
|---|---|---|
| `vrooli-memory.choose-option` | Recommend among prepared options with explicit decisions | `attempt_id`, `options` |
| `vrooli-memory.compare-outcomes` | Read comparable outcome measurements | `operation`, `context_key` |
| `vrooli-memory.finish-attempt` | Record an attempt and linked observations | `attempt`, `observations` |
| `vrooli-memory.inspect-task` | Inspect outcome and pending delivery | `attempt_id`, `resume_token` |
| `vrooli-memory.prepare-attempt` | Prepare bounded advice for a custom workflow | `task_id`, `operation`, `context_key` |
| `vrooli-memory.resume-task` | Requeue frozen capture without domain replay | `attempt_id`, `resume_token` |
| `vrooli-memory.run-task` | Execute a registered learning operation | `operation`, `inputs` |
| `vrooli-memory.scope-bootstrap` | Create and declare a learning scope once | `scope`, `scenario` |

Invoke one with `program-runtime library run <name> --input key=value`; keep
attempt identity and capture status distinct.

Task outcome and delivery are separate. Retain the returned attempt receipt.
Recover capture without repeating the domain operation. An uncertain action needs
owner evidence before another attempt; a runtime completion is not outcome proof.
Unselected advice is not applied advice. Domain programs report explicit decisions
and retain unknown verdicts until evidence supports them.

The following tree applies to manual operations without automatic learning.

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

### Outcome-linked attempt capture and measurement

When a calling skill requests learning effectiveness, use `learning record`
instead of its ordinary task-record append. It stores one `task-record` in Source
Ledger with a versioned payload; it creates no second journal.

`vrooli-memory learning record --scope <scope> --attempt '<Attempt JSON>'`
accepts the typed `Attempt` in
`path:packages/proto/schemas/vrooli-memory/v1/learning/learning.proto`.
Required fields: `attemptId`, `taskId`, `attemptNumber` (starting at 1),
`taskStartedAt`, `startedAt`, `finishedAt` (RFC3339), `operation`, `contextKey`,
`trigger`, `approach`, `outcome`, `recallStatus`, and `provenance`.
Retain task identity, task start, and ordinal across retries. Increment the ordinal
for each actual attempt; retain the exact payload and ID when retrying capture.

- `outcome`: `verified_success`, `failed`, `unavailable`, or `unknown`.
  Success requires `evidenceRefs`; failure requires `failureFingerprint`.
- `recallStatus`: `matched`, `no_match`, or `unavailable`. Matched requires
  `advice`; the other states require an empty advice list.
- Each advice item carries `entryId`, `decision` (`applied` or `rejected`),
  `decisionChange`, `verdict` (`supported`, `contradicted`, `unknown`), and
  `evidenceRefs` for any assessed verdict. Cited memories must exist in the same
  scope before the attempt begins. Retrieval alone is not use or support.
- `provenance`: `operator` or `test`. This label is caller-declared; the journal
  separately preserves server-derived actor/run attribution. Never label a
  fixture as operator evidence.
- `contextKey` identifies comparable platform/profile/mode and tool/policy
  versions. Put changing artifact and run IDs in evidence, not this cohort key.

Optional observed effort fields: `firstActionAt` (within the attempt),
`toolRoundTrips`, `visualReasoningCalls` (nonnegative counts), and `reusedWorkflow`.
Omit unobserved values; an observed zero is different from missing. First-action
latency samples only the first attempt of a task that began inside the window.
Tool/vision medians and reuse rate are per observed attempt, with separate sample
counts; do not compare them across different capture practices.

Use `vrooli-memory learning measure --scope <scope>` to read the last seven days.

Program learning receipts are attempt trees, not one flat note. The root and
each child step carry bounded inputs, outcome evidence, and parent order. Recall
returns summaries under the caller's line budget; zoom is reserved for a
selected candidate. `choose` records the adopted option and reason, and the
later outcome comparison marks advice as derived only when the enclosing
outcome supports or contradicts it. Agent, test, and operator provenance stay
explicit throughout capture and delivery.
Optional `--from`/`--to` select a half-open completion window, at most 90 days;
`--operation`/`--context-key` select exact cohorts. Compare fixed windows in the
same context. The API returns recurrence counts, completed/unresolved task
counts, median attempts/time to first success, and assessed advice outcomes.
Absent denominators have absent optional medians/rates. Unavailable outcomes
remain visible. Timing excludes incomplete and left-censored task histories.

The reader excludes declared test attempts and flags legacy task records,
invalid records, and capped scans (at most 1,000 journal entries). A reliable
sample describes recorded attempts only: it does not prove every real attempt
was captured, authenticate the caller's evidence claims, or establish causality.
Targets and improvement claims require comparable baselines and owner evidence.
No data is `unreliable:no_eligible_attempts`, never a healthy zero.

Capture is idempotent by scope and attempt ID. A conflicting payload is refused.
If memory is unavailable, retain the uncaptured record with the work evidence;
report capture unavailable without changing the operation's outcome.

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

A skill that uses this one names its scope and operation entry point.
An automatically learned program declares `learning_task` in its existing contract;
Program Runtime composes Memory preparation and completion and owns pending delivery.
Nested calls share the parent attempt. Do not also declare manual task capture.
A custom capture program declares `memory` (scope, collect/report phases, entry kinds).
Record each attempt once; retain pending or failed delivery explicitly.

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
