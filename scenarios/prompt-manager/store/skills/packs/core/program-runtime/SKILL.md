## Tools focus: Program Runtime

Use `program-runtime` when a task needs a persistent, governed program session, typed scenario operations, bounded result handles, or provenance-bearing failure discovery. Keep every operation inside the declared binding registry and make materialization, grants, and provenance explicit. For construction patterns, read `scenarios/program-runtime/docs/guides/program-construction.md`.

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

`meta()` preserves response fields that are not rows, such as latency and
reranker information. `raw()` is available when a bounded decoded response is
needed for diagnostics; neither method bypasses the output limit.

```python
verdict = discover("book a flight to the moon")
row = verdict.head(1)[0]
if not row["binding_id"]:
    print({"stopped": True, "reason": row["reason"]})
```

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
| `recall(intent, depth=)` | search-hub | Governed records and docs. `depth="deep"` widens retrieval. |
| `validate(scenario, depth=)` | test-genie | Reads the **latest recorded** run verdicts. It does not start a run. |
| `capture(text, kind=)` | vrooli-memory | `kind="note"` or `"work-record"`; a work record also accepts `trigger`, `approach`, `evidence`, `outcome`. |
| `guide(task)` | prompt-manager | **Currently unavailable** — prompt-manager exposes no governed binding, so the verb fails closed with that reason. Use `prompt-manager skill read <name>` meanwhile. |

Starting a test run stays a lifecycle operation: run `vrooli scenario test
<name>`, then block **once** with `test-genie runs wait`. Never poll, and never
implement a polling loop inside a program.


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
The public namespace is flat: `<scenario>.<group>.<command>`, with hyphens becoming underscores. `vrooli.` addresses the project CLI only and never a scenario; `__vrooli__` is the stable root when a local variable shadows a scenario name.
For a bounded projection, use `count()`, `head(n)`, `filter(...)`, or
`group_by(...)`. Use `materialize(limit)` only when the rows themselves are
needed, and always keep the limit explicit.

The library holds explicitly promoted programs only and is frozen when the
session starts; there are no seeded entries. Use `lib.list()` to enumerate what
the session froze and `lib.<promoted_name>(...)` to call one. The standard
library is the runtime verbs, not `lib`.
library workflow; start a new session after changing a library current version.

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
