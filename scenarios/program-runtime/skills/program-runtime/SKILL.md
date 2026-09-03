---
name: "program-runtime"
description: "Use for high-arity or multi-scenario work: decide whether a governed Python program is the right shape, pick the verb, command, or scenario-owned program that fits, run it in a fresh session, and branch on the envelope instead of a long tool-call loop."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools"]
  tags: ["skill","program","python","session","binding","typed inference","governance","runtime","high arity","multi-scenario","cross-scenario","fan-out","bounded results","return bounded results","discard intermediates","discard intermediate data","tool-call compression","library","candidate","delegation"]
  icon: "terminal"
  status: "active"
  revision: 5
  createdAt: "2026-08-06T00:00:00Z"
  updatedAt: "2026-09-02T20:00:00Z"
  requires:
    scenarios: ["program-runtime", "prompt-manager", "vrooli-memory", "ai-gateway", "agent-manager"]
    commands: ["program-runtime bindings", "program-runtime sessions", "program-runtime programs", "program-runtime library", "program-runtime space", "vrooli-memory recall", "vrooli-memory journal", "prompt-manager skill read", "vrooli scenario"]
  learning:
    scope: "program-runtime-usage"
    capture: "every attempt"
  origin:
    kind: "authored"
---
## Tools focus: Program Runtime

Use `program-runtime` when one task would otherwise be a long tool-call loop over many governed capabilities: fan out across scenario bindings, join cross-scenario reads, classify a corpus, or start several delegated runs, and return only bounded results. This skill holds the judgment `program-runtime <group> help` does not print: when a program is the right shape, which verb or scenario-owned program fits, and what to do with the envelope. Construction rules live in `path:scenarios/program-runtime/docs/guides/program-construction.md`; the standard for a scenario-owned program and its envelope lives in `path:scenarios/program-runtime/docs/guides/program-contracts.md`.

### Scope

In scope: choosing between a CLI step, a runtime verb, a library program, and a new program; sessions, grants, provenance, and spend ceilings; reading an envelope or a failure shape; the candidate harvest and explicit promotion.

Out of scope: direct scenario process execution; ambient shell, model, or agent access from a program; adding a scenario dependency or editing generated dependency approvals; regulating program-runtime itself (`prompt-manager skill read program-runtime-improve`).

### Before acting

Run `vrooli-memory recall wake --scope program-runtime-usage`. Apply a pinned note before choosing a leaf. For one binding or workflow, run `vrooli-memory recall recall "<binding id or task>" --scope program-runtime-usage --limit 5`. Memory mechanics are `prompt-manager skill read vrooli-memory`'s.

### The tree: I have a task

```
Is the task one command against one scenario?
├─ YES → run that scenario's CLI directly. No session, no program.                        [S1]
└─ NO → does a governed binding exist for every operation it needs?
    ├─ Unsure → program-runtime bindings list | bindings unbound (closed-set reason)        [S1]
    │           a needed operation is unbound → stop; file W1 against its owner (report-bug)
    └─ YES → what shape is the work?
        ├─ Reads across 2+ scenarios, counts or keys only
        │     ├─ you need each surface's row keys → run program-runtime.fleet-fanout         [S3]
        │     └─ you need the fan-out latency → run program-runtime.concurrent-fanout        [S3]
        │       ok → use signals.surfaces; unavailable → the named scenario
        │       is down: journal, do not estimate; binding_error → read errors[0].detail
        ├─ A label per text, closed label set
        │     ├─ ≤ 8 texts → run program-runtime.typed-inference                            [S3]
        │     └─ 9–32 texts → run program-runtime.batch-inference (one call)                [S3]
        │       inference_spend_exceeded → raise the ceiling on a NEW session or stop
        ├─ Join, sort, or aggregate rows before reading them
        │     → copy the shape of program-runtime.handle-shaping into your program          [S3]
        ├─ Genuinely agentic work (an Agent Manager workflow must run)
        │     ├─ one or two runs → run program-runtime.delegated-run                        [S3]
        │     └─ 3–8 runs plus an outcome label → run program-runtime.watch-set              [S3]
        │       workflow_rejected → the owner does not declare that key: fix the request,
        │       never retry the same key; delegated_run_spend_exceeded → stop and journal;
        │       scenario_unreachable → agent-manager or the bridge is down (delegation-live
        │       is an improve row; do not work around it)
        ├─ Why do my programs keep failing?
        │     → run program-runtime.failure-triage; branch on signals.top[0]                [S3]
        │       kernel_runtime/kernel_syntax → read programs get <sampleProgramId>
        │       unresolved_name → programs mine-unresolved (typo vs missing binding)
        │       refused_no_grant → the grant step below
        ├─ Which bindings are reachable right now?
        │     → run program-runtime.registry-sweep (dry-run plan, counts by scenario)        [S3]
        │       or program-runtime bindings condition --scenario <s> for freshness           [S1]
        ├─ Someone probably wrote this already
        │     → program-runtime library search "<intent>"; a promoted hit → lib.<name>()      [S1]
        │       in a program; a candidate hit → read its source, then author (below)
        └─ None of the above fits
              → author a bounded program (rules in program-construction.md); if it will
                recur, give it a contract (program-contracts.md) in your scenario's
                .vrooli/program-runtime/ and name it in that scenario's usage skill          [S0]
```

An `[S3]` leaf runs a contract program: `program-runtime sessions create --name <task> --json`, then `programs submit --session-id <id> --source-file scenarios/<scenario>/.vrooli/program-runtime/<name>.py --provenance <agent|operator> --json`, then `sessions delete <id> --reason "<why>"`. Bind inputs by prepending `inputs = {...}` to a copy of the source in your scratch space; never edit the scenario file for one run. Branch only on the envelope's `status` and `errors[0].class`; the contract JSON beside the source declares every value. `unavailable` is unknown, never zero.

### Runtime verbs: which one, and when a binding beats it

| Verb | Reach for it when | Reach for a binding instead when |
|---|---|---|
| `discover("<intent>")` | You do not know which binding serves an intent; you want one governed id or an explicit null | You already know the id: `describe("<id>")` and call it |
| `recall("<intent>")` | A program needs Search Hub ranked rows; `depth="deep"` only when the first pass is empty | You need one scenario's own search: call that scenario's `search` binding |
| `validate("<scenario>")` | You need the latest recorded test-genie run rows without starting a run | You need a new run: `vrooli scenario test <name>` from the shell, then `test-genie runs wait` once |
| `capture("<text>", kind=)` | The program itself should leave a memory line at S4 (contract-declared `memory`) | The tree is still in the skill (S1 to S3): journal from the shell after the run |
| `guide("<intent>")` | A program needs prompt-manager discovery rows for a next step | A human is choosing: `prompt-manager discover` prints the same rows |
| `lib.<name>()` | `library search` returned a promoted entry whose description matches; the catalog is frozen at session start | The entry is a candidate: read its source, adopt the shape, do not call it |

### Commands where judgment applies

| Command | Use it when | Do not use it when |
|---|---|---|
| `bindings condition` | A program returned `unavailable` for a scenario that `vrooli scenario status` says is running; the row says dormant, stale, or degraded | You want to know whether a binding exists (`bindings list`) |
| `bindings sweep` | You need the eligible set before a fleet read; keep `--dry-run` (the default) | `--execute` on a shared host without an operator asking; it exercises every eligible read binding |
| `programs governance-share` | An improve cycle or an audit asks what share of calls in a window went through bindings; `governed_share` is in the summary, the rows are the observed names | You want per-program failures (`programs mine`) |
| `programs mine-refusals` | Programs keep ending `refused`; the rows name the binding and the grant it lacked | The refusal was `not_run_eligible`: there is no grant for that |
| `programs mine-unresolved` | `unresolved_name` recurs; a name close to a governed one is a typo, a real capability with no binding is W1 for its owner | You already know the binding id: use `bindings describe` |
| `library promote` | A program succeeded and a second agent will call the same shape; pass `--reason` naming the binding set it deduplicates | The program printed rows instead of counts, or an operator-provenance fixture run is the only success |
| `library set-current` | Two promoted versions share a binding set; select the newest validated one, keep history | You want to delete a version: there is no delete, and none is wanted |
| `library search` | Before authoring anything with two or more bindings | Locating a scenario-owned program: read `scenarios/<scenario>/.vrooli/program-runtime/`. There is no `library run` verb yet: call a program with the submit recipe in `path:scenarios/program-runtime/docs/guides/program-contracts.md` §"Calling a program" |
| `space` | A coverage-space consumer (meta-optimization-manager) needs this scenario's denominator | Any other reason; it is a read contract for machines |
| `discovery eval --suite evals/discovery.primary.json --mode judged` | An improve cycle reads the discovery floor, or a descriptor change needs its effect on selection measured. `judged` is the default; `fast` skips the judge, `deep` widens retrieval | Inside a program: the command is local to the CLI and has no governed binding, so a program reports the row `unavailable` with reason `no_governed_binding` |
| `authoring eval` | The authoring brief or the construction guide changed and first-attempt authoring needs re-measuring against the versioned corpus | Inside a program: the binding exists and is run-eligible, but one run exceeds four minutes against the kernel's 100 s invoke budget, so a program reports the row `unavailable` with reason `kernel_invoke_budget`. Run it from a shell and read `met`, `floor`, and `missed` |

### Sessions, ceilings, and the candidate harvest

- One session per program call. Session variables persist across submissions, so a stale `inputs` from an earlier submission would be reused by the guard in every contract program.
- A synchronous submission is bounded at 120 s. Longer work: `programs submit ... --async --wait-timeout 300s`, or `programs wait <program-id> --timeout 300s` once. Never poll.
- `sessions create --inference-ceiling-micros N --delegation-ceiling-micros N` sets the two spend ceilings. A program that crosses one ends with `inference_spend_exceeded` or `delegated_run_spend_exceeded`; the work is stopped, not retried. Decide the ceiling before the run from the contract's `budget`; raising it mid-task means a new session and a journal line saying why.
- Every successful submission, including a fixture run, enters the library's `candidate` tier automatically (`library list` shows `origin: agent-authored`, `tier: candidate`). Promotion is explicit and stays so: a candidate becomes reusable only through `library promote`, and a promoted program becomes scenario-owned only when its owner writes the `.py` and `.json` pair into `.vrooli/program-runtime/`. Do not promote to make the library look tidy; the improve skill's `library-curate` step groups candidates by binding set first.

### Provenance

| You are | `--provenance` | Never |
|---|---|---|
| Acting on an operator's direct instruction in this session | `operator` | Label a heartbeat or a fixture run `operator` to make `programs mine` look quieter |
| A heartbeat, a goal-loop cycle, or any run a human did not type | `agent` | Use `agent` to exercise a destructive binding without a grant |
| Running a contract fixture from the `programs` test-genie phase | `test` | Count `test` runs in `agent-failure-rate` |
| Replaying a recorded program for a diff | `replay` | Promote a `replay` success into the library |

### Authoring a program that will recur

1. `library search "<intent>"` first; a promoted hit ends the authoring.
2. Copy the shape of `scenarios/program-runtime/.vrooli/program-runtime/setpoint-read.py`: envelope first, `fail()` helper, `classify_transport()`, labeled phase functions, one `print(envelope)` on every path.
3. Probe every binding's row keys and argument names in a scratch session before the contract names them (`describe("<id>")`, `print(h.head(1)[0].keys())`); `bindings describe <id> --json` for the CLI view.
4. Write the contract beside the source; validate it against `scenarios/program-runtime/schemas/program-contract.schema.json`.
5. `programs submit --explain` in a fresh session until `diagnostics` is null.
6. Run every fixture in its own session; a fixture whose dependency is stopped is recorded `unavailable` with the reason, not skipped.
7. Name the program in the owning scenario's usage skill leaf as `run <scenario>.<name>` and in its `.vrooli/service.json` `skills` block.

A read that can fail on its own (one scenario down among five) is wrapped so the row reads `unavailable` and the envelope reads `partial`; a `gather` that raises on the first failure loses the four healthy rows.

### Output bound and materialization

| The caller needs | Use | Never |
|---|---|---|
| A number | `count()`, `agg(field, "sum")` | `len(materialize(...))` |
| A distribution | `group_by(key)` (dict-shaped, has `.count()`) | printing rows and counting by eye |
| A few rows to look at | `head(n)` with `n` written in the contract's `materialize_limit` | `head(1000)` |
| Rows for a join | `filter` and `join` inside the kernel, then `head(n)` of the result | joining after materializing both sides |
| The response's non-row fields | `meta()` (`governedShare`, `totalTokens`, latency) | reading them from `head(1)[0]` where they are not |
| A diagnostic dump | `raw()` once, in a scratch session | `raw()` inside a contract program's envelope |

### In-use settings

| Symptom | Setting move | Journal |
|---|---|---|
| Program ends `deadline_exceeded` | Resubmit with `--async --wait-timeout 300s` in a new session | the program id and the wall time |
| Program ends `inference_spend_exceeded` | New session with a higher `--inference-ceiling-micros`, only if the contract's `budget.inference_calls` justifies it | old and new ceiling, the contract name |
| Program ends `refused_no_grant` | `sessions grant <id> --grants <exact grant>` after an operator confirms; destructive calls also need confirmation in the request | the grant string and who confirmed |
| CLI answers about a stale port after an API restart | `export PROGRAM_RUNTIME_API_PORT=$(for p in $(pgrep -f program-runtime-api); do tr '\0' '\n' < /proc/$p/environ \| grep '^API_PORT=' \| cut -d= -f2; done \| head -1)` | that auto-detect was wrong (a W3 item if it repeats) |
| Row keys in the kernel do not match the CLI | None; keys are protojson camelCase in-kernel (`governedShare`), snake_case in the CLI. Probe with `print(handle.head(1)[0].keys())` | the binding and the key that differed |

### Debug order

1. `program-runtime programs get <id>`: read `failure_shape` (closed vocabulary) before `failure_detail`.
2. `unresolved_name` or `unknown_field`: the detail lists the candidates; fix the name, resubmit in a new session.
3. `ambiguous_response`: pass `rows="<field>"`; the detail lists the repeated fields.
4. `unreachable_scenario` or `bridge_transport`: `vrooli scenario status <scenario>`, then `bindings condition --scenario <scenario>`. Do not start the scenario from inside a program.
5. `refused_*`: `programs mine-refusals`; request the grant through the session path or stop.
6. `kernel_runtime` naming `vrooli`: there is no `vrooli` module and `vrooli.` is never a scenario prefix; every runtime name is already bound.
7. `unclassified`: journal it with the program id; it is a missing failure cause and an improve row.

### Governance rules

- Use only bindings returned by `bindings list`. Do not call a method that `bindings unbound` classifies as `no_manifest`, `local_binding`, `omitted_rpc`, or `external_tool_only`.
- Treat destructive operations as denied until the session has the exact grant and the request includes confirmation.
- Use `operator` provenance only for direct operator work. Use `agent`, `test`, or `replay` only when the caller can identify that source.
- Keep program output bounded: `count`, `head(n)`, `filter`, `group_by`, `meta()`; `materialize(limit)` only when rows are required and the limit is explicit.
- Treat `discover` with an empty `binding_id` and `unavailable=False` as a null verdict: stop. `unavailable=True` means discovery itself failed: retry or downgrade to `mode="fast"`. Never shell out to bypass discovery.
- `ai.*` and `agent.*` are namespaces enabled only inside a live session with the governed bridge; they are not ambient capabilities.
- No retry loop and no polling loop inside a program. Retry is the caller's decision, informed by the envelope.

### Safety

A program may change only session, program, telemetry, and library-candidate state owned by program-runtime, and files inside a declared sandbox workspace. `bindings sweep --execute` and `library promote` are the two writes this skill's tree reaches; both need a stated reason. Never edit a library row; edit the scenario file and let the projection follow.

### After acting, always

One `vrooli-memory journal note --scope program-runtime-usage --kind task-record --trigger "<task>" --approach "<leaf taken, program or command>" --evidence "<session id, program id, envelope status, errors[0].class>" --outcome "<ok | partial: <gap> | unavailable: <scenario> | refused: <grant> | failed: <class>>" "<one line>"` per attempt. Use `--kind binding-note` when the finding is about one binding's keys or arguments. After the third identical confirmation of a note, pin it; when a note's advice fails, supersede it (`prompt-manager skill read vrooli-memory`).

### Troubleshooting & Edge Cases

| Symptom | Likely cause | First check | Fix |
|---|---|---|---|
| Every program reports `unavailable` | API restarted; CLI resolved a stale port | `vrooli scenario status program-runtime`; `ss -ltnp \| grep program-runtime` | Export `PROGRAM_RUNTIME_API_PORT` (in-use settings); file W3 if auto-detect keeps failing |
| Submission says session not found | Session reclaimed or never created | `program-runtime sessions get <id>` | Create a new session; do not reuse an id from a journal |
| A contract program ignores my inputs | Stale `inputs` variable in a reused session, or the preamble was not prepended | `sessions get <id>`; the copied source's first line | Fresh session per call; prepend `inputs = {...}` to the copy |
| `protected runtime name "lib" cannot be assigned` | The program used `lib`, `ai`, `agent`, or a verb name as a variable | the assignment line | Rename the variable; runtime names are reserved |
| `Handle` has no `.get()` | Read a field from the handle instead of a row | the offending line | `row = handle.head(1)[0]` then `row["field"]` |
| `ai.batch` returns one row | The batch response is one row whose `results` list carries the ordered items | `result.head(1)[0].keys()` | Read `head(1)[0]["results"]`; `ai.classify(texts=[...])` returns one row per text |
| Result is too large or the envelope is cut off | Eager row printing or a long list in `signals` | the program's `print` calls | Counts, `head(n)`, capped lists; the envelope must fit the output bound |
| Delegation returns `NOT_FOUND_WORKFLOWREVISION` | The owner no longer declares that `workflow_key` | `agent-manager workflow` help for the owner's keys | Fix the request; do not retry the same key |
| Discovery returns a null verdict | Nothing governed serves the intent | `result.head(1)[0]["reason"]` | Clarify the intent or read `bindings describe`; do not guess a path |
| `library promote` refused | Source program did not succeed, or name@version exists | `programs get <id>`; `library get <name>` | Promote a succeeded run under a new version |
| Candidate count climbs during validation | Fixture runs are successful submissions and are harvested | `library list --json` filtered to `origin == agent-authored` | Expected; the improve skill's `library-curate` groups them. Do not promote fixtures |

### Output expectations

Every runtime workflow leaves a traceable session id, program id, provenance value, envelope status, and one journal line. A successful workflow may change only the state named under Safety. It must not mutate shared bindings, generated proto output, dependency approvals, library rows, or unrelated scenarios.
