---
name: "program-runtime"
description: "Use for high-arity or multi-scenario work: write one governed Python program to fan out across scenario bindings, join cross-scenario reads, discard intermediate data, and return bounded results instead of long tool-call loops."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools"]
  tags: ["skill","program","python","session","binding","typed inference","governance","runtime","high arity","multi-scenario","cross-scenario","fan-out","bounded results","return bounded results","discard intermediates","discard intermediate data","tool-call compression"]
  icon: "terminal"
  status: "active"
  revision: 3
  createdAt: "2026-08-06T00:00:00Z"
  updatedAt: "2026-08-19T00:00:00Z"
  requires:
    scenarios: ["prompt-manager", "test-genie", "vrooli"]
    commands: ["prompt-manager", "test-genie", "test-genie runs", "vrooli", "vrooli import", "vrooli scenario"]
  origin:
    kind: "authored"
---
## Tools focus: Program Runtime

Use `program-runtime` when a task needs a persistent, governed program session, typed scenario operations, bounded result handles, or provenance-bearing failure discovery. Keep every operation inside the declared binding registry and make materialization, grants, and provenance explicit. For construction patterns, read `scenarios/program-runtime/docs/guides/program-construction.md`.

It is especially appropriate for high-arity or multi-scenario work: fan out
across governed scenario bindings, join cross-scenario reads, discard
intermediate data inside the kernel, and return only bounded results. Prefer
this shape when a task would otherwise require a long tool-call loop over many
local capabilities.

### Scope

In scope:

- Create and reclaim program sessions.
- Submit programs with explicit provenance.
- Discover callable bindings and closed-set unbound reasons.
- Use bounded handles and explicit materialization.
- Read typed lifecycle telemetry, measures, and recurring failure shapes.
- Run bounded Python programs with scenario-qualified namespaces and typed
  inference helpers.

Out of scope:

- Direct scenario process execution.
- Ambient shell, model, or agent access from a program.
- Adding a new scenario dependency or editing generated dependency approvals.
- Replacing scenario-owned domain APIs with undocumented shortcuts.

Read `path:docs/agent-system/PROMOTION_LADDER.md` when repeated manual runtime work suggests a deterministic CLI or Action promotion.

### Core flow

1. Confirm that the scenario is running through the Vrooli lifecycle. Use `vrooli scenario start <name>` or the scenario's `make start` target.
2. Inspect the callable projection before writing a program:

   ```text
   program-runtime bindings list
   program-runtime bindings unbound
   program-runtime bindings doctor
   program-runtime bindings describe ai-gateway/inference/run
   ```

   A submitted program can read the same live contract without leaving the
   kernel. Pass either a binding id or its qualified dotted path and keep the
   bounded argument rows in a handle:

   ```python
   contract = describe("test-genie/runs/list")
   print(contract.head(10))
   ```

   The description is resolved through the registry on every call, so it is
   the same descriptor-backed contract that will accept or refuse the later
   binding invocation.

3. Create a named session when state must survive multiple submissions:

   ```text
   program-runtime sessions create --name <name>
   ```

4. Submit a short program with explicit provenance. Use the session returned by `sessions create`.

   ```text
   program-runtime programs submit --session-id <session> --source <program> --provenance operator
   ```

5. Read the bounded result with `program-runtime programs get <id>`. Treat `failure_detail` and `failure_shape` as diagnostic data, not as permission to bypass the registry. `failure_shape` carries a closed-vocabulary cause (for example `unknown_field`, `ambiguous_response`, `unreachable_scenario`), never a Python exception class.
6. Reuse the session only when the program requires prior variables. Reclaim it with `program-runtime sessions delete <session>` and state the reclaim reason.

Use this work table for operation choice:

| Need | Command | Required proof |
|---|---|---|
| Find callable operations | `bindings list` | The operation has a manifest binding and `run_eligible` is true. |
| Explain why an operation is unavailable | `bindings unbound` | The result contains one closed-set unbound reason. |
| Resolve the Act projection | `bindings act` | With no request payload, audits the owned 28-cell denominator and returns one measured verdict per cell with `partial` confidence. |
| Preserve variables | `sessions create` once, then reuse its ID | The same session ID appears in each submission. |
| Run a program | `programs submit` | The source and provenance are recorded in the returned program. |
| Find recurring failures | `programs mine` | The result contains a failure shape and count, or an explicit empty count. |
| Read lifecycle evidence | `telemetry events` | Events identify the program, session, status, and provenance. |

### Runnable Python programs

Programs execute in a persistent standard-library Python kernel. Binding names
are public Python attributes: scenario, group, and command segments are
normalized from hyphens to underscores. Results are bounded `Handle` values;
materialize only the small projection the caller needs.

```python
# Count and group without copying rows into program output.
rows = search_hub.query.query(query="program runtime", rows="ranked")
print(rows.count())
print(rows.group_by("kind"))
print(rows.meta().get("latencyMs"))
```

Independent calls run concurrently through the explicit `gather` helper;
pass zero-argument callables so each binding starts on a worker thread:

```python
queries = ["proto bindings", "telemetry", "scenario health"]
results = gather(*[
    lambda query=query: search_hub.query.query(query=query, rows="ranked")
    for query in queries
])
print([result.count() for result in results])
```

Typed inference is a governed ai-gateway call, not a direct provider path:

```python
result = ai.classify(
    "The request timed out after the provider retry.",
    schema={"type": "string", "enum": ["infra", "user"]},
    instruction="Choose the primary failure class.",
)
print(result.head(1))
```

The inference helpers accept `text=` and `texts=` as additive aliases for
`source=` on `ai.classify`, `ai.extract`, `ai.judge`, and `ai.write`.
`ai.classify(texts=[...])` uses the governed batch route. `describe` accepts
`binding_id=` as an additive alias for `binding=`. The canonical names remain
supported, but supplying multiple spellings in one call is a `TypeError` rather
than a silent precedence rule. For the common bounded classification shape,
`ai.classify(text="...", labels=["bug", "feature"])` desugars to the same
validated enum schema path as a caller-written `schema=`. `labels=` and
`schema=` are mutually exclusive, and labels must be a non-empty list of
strings.

`meta()` preserves response fields that are not rows, such as latency and
reranker information. `raw()` is available when a bounded decoded response is
needed for diagnostics; neither method bypasses the output limit.

```python
verdict = discover("book a flight to the moon")
row = verdict.head(1)[0]
if not row["binding_id"]:
    print({"stopped": True, "reason": row["reason"]})
```

`ai`, `agent`, and `lib` are **namespaces, not callables**: write
`ai.classify(...)`, `ai.extract(...)`, `ai.judge(...)`, `ai.write(...)`,
`ai.batch(...)`, `agent.start/collect/run`, and `lib.list()`. Calling `ai(...)`
directly raises `TypeError`. The remaining verbs are called directly.

Every call returns a `Handle` — never a plain object or dict. A `Handle` has no
`.get()` and no attributes named after response fields; take the row first with
`row = handle.head(1)[0]`, then read `row["field"]`.

`Handle.group_by(key)` returns a dict-shaped count mapping with a `count()`
helper for the represented source rows. `Handle.join` accepts `key="id"` or
the additive `on="id"` alias; supplying both names is an explicit error. Typed classification accepts either a
single text string or a small list of strings; the list form uses the governed
batch route and preserves input order. `recall` takes an intent and optional
depth only, not a binding id or arbitrary source payload.

```python
classified = ai.classify(source=["one", "two"], labels=["bug", "feature"])
print(classified.head(2))
```

`describe(binding_id="...")` returns argument rows. Use
`row = describe(binding_id="...").head(1)[0]` and read `row["name"]`; the
handle is not a single row containing an `arguments` field.

The convenience roles are `classify.fast`, `extract.structured`,
`judge.default`, and `write.default` (`ai.write(...)`). The first three
are deterministic and refuse a caller-supplied `temperature` with
`INVALID_REQUEST`; `write.default` is overridable, so `ai.write` alone accepts
`temperature=` and `max_output_tokens=`. Omit them to get the role's declared
sampling — and note that `temperature=0.0` is a deterministic *request*, not an
omission. If the ai-gateway binding is unavailable, the helper fails
closed with a stated bridge or provider error. Delegated agent work remains a
separate `agent` capability and requires its own governed bridge.

### Runtime verbs

Each verb takes its primary argument positionally or by keyword, returns a
`Handle`, and fails closed naming its unavailable dependency. None falls back to
a shell call or a direct provider call.

| Verb | Resolves through | Notes |
|---|---|---|
| `discover(intent)` | binding registry + Search Hub | One governed binding or an explicit null verdict. |
| `recall(intent, depth=, rows="ranked")` | search-hub | Returns the selected response rows directly; `ranked` is the default. `query=` is an additive alias for `intent=`. `depth="deep"` widens retrieval. |
| `validate(scenario, depth=, rows="runs")` | test-genie | Returns the latest recorded run rows directly. It does not start a run. |
| `capture(text, kind=)` | vrooli-memory | Returns a one-row Handle containing the append response; `kind="note"` or `"work-record"`. |
| `guide(intent, rows="results")` | prompt-manager | Composes `prompt-manager/discover/discover` and returns ranked skill/action discovery rows. `task=`, `query=`, and `text=` are aliases for `intent=`. |

Starting a test run stays a lifecycle operation: run `vrooli scenario test
<name>`, then block **once** with `test-genie runs wait`. Never poll, and never
implement a polling loop inside a program.

Runtime verbs obey the same bounded result contract as scenario bindings:
`count()`, `head()`, and `group_by()` operate on the owning operation's rows.
`meta()` carries non-row response fields plus `verb`, `binding_id`, and `owner`,
while `raw()` carries the decoded owning response. Use `rows="<field>"` to
select a different repeated response field; an invalid selection fails closed
and lists the available fields.

The public keyword vocabulary is recovery-oriented. `recall(query="...")`,
`validate(scenario="...")`, `capture(text="...")`, and `guide(task="...")`
are accepted forms.
Supplying two spellings of one value raises a `TypeError`. An unknown keyword
names the offending keyword and lists the accepted keywords; use that list
instead of guessing another spelling.

`discover` takes `mode="fast"`, `"judged"` (default), or `"deep"`. Its row
separates two outcomes that both carry an empty `binding_id`: `unavailable=False`
is the honest verdict that nothing governed serves the intent, and is a **stop**;
`unavailable=True` means discovery itself failed, is **not** evidence that the
capability is absent, and should be retried or downgraded to `mode="fast"`.

### Long-running programs

A synchronous submission is bounded at two minutes. Longer work is submitted
asynchronously and awaited once:

```text
program-runtime programs submit --session-id <id> --source-file work.py --provenance agent --async
program-runtime programs wait <program-id> --timeout 300s
```

`--async --wait-timeout 300s` does both in one command. Block **once**; the wait
is served by the runtime and wakes on the terminal transition, so a polling loop
is never correct. A synchronous submission that outruns its bound reports
`deadline_exceeded` and names the still-running program id to wait on.


Before hand-authoring a repeated shape, inspect the reviewed library first:

~~~python
catalog = lib.list()
print(catalog.head(20))
result = lib.concurrent_fanout()
~~~

The catalog is frozen when the session starts. Use the exact promoted name and
its documented arguments; if no entry fits, then author a bounded program.
Start a new session after a library version or current selection changes.

### Worked examples

Read `scenarios/program-runtime/docs/guides/program-construction.md` first: it
teaches when a program beats separate tool calls, the three addressing forms,
the allowlisted builtin surface, the fetch-shape-materialize discipline, and how
to read a failure cause. Its task-shaped pages under `docs/construction/` carry
the construction patterns.

The runnable files in `scenarios/program-runtime/docs/examples/` are the
executable form of those patterns:

| Example | Demonstrates |
|---|---|
| `fleet-fanout.py` | bounded summaries from several scenario bindings |
| `typed-inference.py` | concurrent typed classification over a small corpus |
| `delegated-run.py` | governed Agent Manager delegation and an evidence handle |
| `failure-triage.py` | grouping recurring program failures without eager materialization |
| `concurrent-fanout.py` | `await` plus `asyncio.gather` with elapsed-time evidence |

`delegated-run.py` is a live bridge example using the terminal
`development-toolchain-validator/skill-experiment-audit` workflow. Its
successful validation returned a bounded evidence handle with a succeeded
workflow, one challenged evidence finding, and zero gaming findings. The
bridge remains governed by Agent Manager's role-policy catalogs; it does not
silently substitute a direct agent or shell call when those catalogs are
unavailable.

Use `discover("typed inference")` to request one judged capability or
an explicit null verdict. A successful handle row includes `binding_id`,
`confidence`, `method`, and the resolved `arguments`; inspect
`row = result.head(1)` before invoking anything. A null row has an empty
`binding_id` and a stated `reason`. Stop and ask for a clearer intent or use
the governed `bindings describe` command; never shell out or guess a path.
The public namespace is flat: `<scenario>.<group>.<command>`, with hyphens becoming underscores. `vrooli.` addresses the project CLI only and never a scenario — and never a runtime verb, so `vrooli.discover(...)` and `vrooli.recall(...)` both fail; call verbs at the top level. `__vrooli__` is the stable root when a local variable shadows a scenario name.

**There is no `vrooli` module.** `import vrooli` and `from vrooli import recall` raise `ModuleNotFoundError`; every runtime name is already bound in the program's globals.

**A program is module scope, not a function body.** `return` at the top level is a `SyntaxError` — use `if`/`else`, or wrap the body in a `def` and call it.
For a bounded projection, use `count()`, `head(n)`, `filter(...)`, or
`group_by(...)`. Use `materialize(limit)` only when the rows themselves are
needed, and always keep the limit explicit.

The standard library is the runtime verbs, not lib; lib is the reviewed
program library. Start a new session after changing a library current version.

### Governance rules

- Use only bindings returned by `bindings list`.
- Do not call a method that `bindings unbound` classifies as `no_manifest`, `local_binding`, `omitted_rpc`, or `external_tool_only`.
- Treat destructive operations as denied until the session has the exact grant and the request includes confirmation.
- Use `operator` provenance only for direct operator work. Use `agent`, `test`, or `replay` only when the caller can identify that source.
- Keep program output bounded. Use handle operations such as `count`, `head`, `filter`, and `group_by`; call `materialize(limit)` only when rows are required.
- Treat `discover` with an empty `binding_id` as a null verdict. Do not
  invoke an alternative by guessing, and do not shell out to bypass discovery.
- Discovery uses `fast`, `judged` (the default), or bounded `deep` mode through
  the typed binding contract. A Search Hub outage is an explicit unavailable
  result, not a local boot-time fallback.
- Do not treat `ai.*` or `agent.*` as ambient capabilities. They
  are only enabled inside a live program session with the corresponding
  governed bridge and registry binding.

### Validation

After a change, run the narrow owner tests first:

```text
GOWORK=off go test ./...              # from scenarios/program-runtime/api
UPDATE_CLI_EVIDENCE=1 GOWORK=off go test ./...  # from scenarios/program-runtime/cli
python3 -m pytest -q                  # from scenarios/program-runtime/kernel
pnpm type-check && pnpm exec vitest run  # from scenarios/program-runtime/ui
```

Then run the scenario-owned suite:

```text
vrooli scenario test program-runtime
```

Wait once on the returned run ID with `test-genie runs wait --json program-runtime <run-id>`. Do not poll the run. Record skipped providers and non-comparable baselines as validation limits.

### Troubleshooting & Edge Cases

| Symptom | First check | Required response |
|---|---|---|
| Binding is unbound | `program-runtime bindings unbound` | Add a manifest binding through the owning scenario contract, or preserve the stated omission reason. |
| Submission says session not found | `program-runtime sessions get <id>` | Create a new session. Do not silently create one when state continuity matters. |
| Destructive call is refused | Session grants and confirmation | Request the exact grant through the session governance path. Do not weaken authorization. |
| Result is too large | Program source and handle operations | Replace eager row printing with `count`, `head`, `filter`, `group_by`, or bounded `materialize`. |
| Discovery returns a null verdict | `result.head(1)` and its `reason` | Clarify the intent or inspect the governed binding contract; do not shell out, guess a path, or invoke an alternative. |
| Discovery says unavailable | Search Hub health and `result.head(1)` | Preserve the explicit unavailable reason and retry through the governed runtime after Search Hub recovers; direct binding calls remain the only fallback. |
| Python adapter is unavailable | Scenario problem log and kernel health | Use the standard-library kernel protocol. Do not install packages with a raw package manager. |
| Template cleanup is blocked | `template-manager` service health | Keep the legacy proto methods explicitly omitted and document the blocker. Resume official detemplate when the service is repaired. |

The runtime skill remains prose while its decisions require judgment about provenance, permissions, and materialization. Promote a repeated deterministic one-command operation to a Prompt Manager Action after the CLI exposes a stable output contract.

### Output expectations

Every runtime workflow must leave a traceable session ID, program ID, provenance value, and validation command. A successful workflow may change only the session/program/telemetry state owned by `program-runtime` and files explicitly placed in its sandbox. It must not mutate shared bindings, generated proto output, dependency approvals, or unrelated scenarios.
