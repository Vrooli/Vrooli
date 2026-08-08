## Tools focus: Program Runtime

Use `program-runtime` when a task needs a persistent, governed program session, typed scenario operations, bounded result handles, or provenance-bearing failure discovery. Keep every operation inside the declared binding registry and make materialization, grants, and provenance explicit.

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

3. Create a named session when state must survive multiple submissions:

   ```text
   program-runtime sessions create --name <name>
   ```

4. Submit a short program with explicit provenance. Use the session returned by `sessions create`.

   ```text
   program-runtime programs submit --session-id <session> --source <program> --provenance operator
   ```

5. Read the bounded result with `program-runtime programs get <id>`. Treat `failure_detail` and `failure_shape` as diagnostic data, not as permission to bypass the registry.
6. Reuse the session only when the program requires prior variables. Reclaim it with `program-runtime sessions delete <session>` and state the reclaim reason.

Use this work table for operation choice:

| Need | Command | Required proof |
|---|---|---|
| Find callable operations | `bindings list` | The operation has a manifest binding and `run_eligible` is true. |
| Explain why an operation is unavailable | `bindings unbound` | The result contains one closed-set unbound reason. |
| Resolve the Act projection | `bindings act` | Each requested cell returns a measured or sketch verdict. |
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
rows = vrooli.search_hub.search.query(query="program runtime")
print(rows.count())
print(rows.group_by("kind"))
```

Independent calls run concurrently through the explicit `vrooli.gather` helper;
pass zero-argument callables so each binding starts on a worker thread:

```python
queries = ["proto bindings", "telemetry", "scenario health"]
results = vrooli.gather(*[
    lambda query=query: vrooli.search_hub.search.query(query=query)
    for query in queries
])
print([result.count() for result in results])
```

Typed inference is a governed ai-gateway call, not a direct provider path:

```python
result = vrooli.ai.classify(
    "The request timed out after the provider retry.",
    schema={"type": "string", "enum": ["infra", "user"]},
    instruction="Choose the primary failure class.",
)
print(result.head(1))
```

The convenience roles are `classify.fast`, `extract.structured`, and
`judge.default`. If the ai-gateway binding is unavailable, the helper fails
closed with a stated bridge or provider error. Delegated agent work remains a
separate `vrooli.agent` capability and requires its own governed bridge.

### Worked examples

The runnable files in `scenarios/program-runtime/docs/examples/` cover the
common shapes an agent should copy and adapt:

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

Use `vrooli.discover("typed inference")` to find candidate capabilities, then
confirm the exact manifest argument names with
`program-runtime bindings describe <scenario>/<group>/<command>`. The public
namespace is `vrooli.<scenario>.<group>.<command>`; hyphens become underscores.
For a bounded projection, use `count()`, `head(n)`, `filter(...)`, or
`group_by(...)`. Use `materialize(limit)` only when the rows themselves are
needed, and always keep the limit explicit.

### Governance rules

- Use only bindings returned by `bindings list`.
- Do not call a method that `bindings unbound` classifies as `no_manifest`, `local_binding`, `omitted_rpc`, or `external_tool_only`.
- Treat destructive operations as denied until the session has the exact grant and the request includes confirmation.
- Use `operator` provenance only for direct operator work. Use `agent`, `test`, or `replay` only when the caller can identify that source.
- Keep program output bounded. Use handle operations such as `count`, `head`, `filter`, and `group_by`; call `materialize(limit)` only when rows are required.
- Do not treat `vrooli.ai.*` or `vrooli.agent.*` as ambient capabilities. They
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
| Python adapter is unavailable | Scenario problem log and kernel health | Use the standard-library kernel protocol. Do not install packages with a raw package manager. |
| Template cleanup is blocked | `template-manager` service health | Keep the legacy proto methods explicitly omitted and document the blocker. Resume official detemplate when the service is repaired. |

The runtime skill remains prose while its decisions require judgment about provenance, permissions, and materialization. Promote a repeated deterministic one-command operation to a Prompt Manager Action after the CLI exposes a stable output contract.

### Output expectations

Every runtime workflow must leave a traceable session ID, program ID, provenance value, and validation command. A successful workflow may change only the session/program/telemetry state owned by `program-runtime` and files explicitly placed in its sandbox. It must not mutate shared bindings, generated proto output, dependency approvals, or unrelated scenarios.
