# Constructing a program

Program Runtime is a governed programming surface for agents. You submit a
Python program; it runs in a session-persistent kernel where every
manifest-bound Vrooli operation is a typed callable, and results stay in the
kernel as bounded `Handle` values instead of being copied into your context.

This page teaches how to build one. It is not a catalog of programs to call —
you write the program. The task-shaped pages at the bottom show construction
patterns for specific shapes of work.

## When a program is the right move

Write a program when the work has **arity** or **discard**:

- **Arity** — the task touches many units. Inspecting 60 scenarios is one
  program with one result, not 60 tool calls with 60 responses.
- **Discard** — you need a small answer derived from a large intermediate.
  Counting, grouping, joining, and filtering all belong in the kernel; only the
  answer crosses back.

A single read with a small response is not worth a program. Call the scenario
CLI directly.

## The three addressing forms

```python
rows = search_hub.query.query(query="program runtime", rows="ranked")  # a scenario binding
vrooli.scenario.status(name="program-runtime")                          # the project CLI
verdicts = validate("program-runtime")                                  # a runtime verb
```

- **Scenario bindings** are flat top-level names. `<scenario>.<group>.<command>`,
  with hyphens becoming underscores: `search-hub` is `search_hub`.
- **`vrooli.`** is the project control plane and nothing else. It is permanent,
  so a `vrooli.`-prefixed call is always the project CLI, never a scenario.
- **Runtime verbs** are top-level names the runtime owns.

**Do not prefix a scenario binding with `vrooli.`.** `vrooli.search_hub…` does
not resolve.

### Shadowing and the escape hatch

Runtime-owned names — the verbs plus `vrooli` and `__vrooli__` — cannot be
assigned; the submission is refused and names the protected name. Scenario names
*can* be shadowed, because the set of scenarios grows and reserving a growing
set would break older programs. Submission warns when you shadow one, and
`__vrooli__` always reaches the binding:

```python
search_hub = ["a", "b"]                                    # legal, warns
__vrooli__.search_hub.query.query(query="x", rows="ranked")  # still reachable
```

## The core discipline: fetch wide, shape in-kernel, materialize narrow

```python
rows = search_hub.query.query(query="retention", rows="ranked")   # wide
top = rows.filter(lambda r: r["score"] > 0.5).group_by("provider_id")  # shaped
print(top)                                                         # narrow
```

`Handle` carries `count`, `head`, `filter`, `map`, `select`, `sort`, `unique`,
`agg`, `join`, `group_by`, `meta`, and `raw`. Call `materialize(limit)` only
when you need the rows themselves, and always pass a limit. Printed output is
bounded (4 KB by default, 64 KB when the submission opts into materialized
output), so an unbounded print is truncated rather than expensive.

`meta()` returns the response fields that are not rows — latency, routing,
totals. `raw()` returns the decoded response. Neither escapes the output bound.

## Ambiguous responses

A binding whose response has more than one repeated field cannot guess which one
is the rows. It fails closed and names the candidates:

```
binding search-hub/query/query has no determinable primary response field;
candidate repeated fields: corporaSearched, groups, ranked, routingExplanation
```

Name the field to proceed. This is opt-in per call and never defaults:

```python
rows = search_hub.query.query(query="retention", rows="ranked")
```

## Session state

Variables persist across submissions in the same session, so a long
investigation accumulates working state:

```python
# submission 1
registry = program_runtime.bindings.list()

# submission 2 — registry is still live
print(registry.group_by("effect"))
```

State is process memory and is deliberately not persisted. A handle is a live
reference; a reclaimed session loses it.

## Fan-out

Independent calls run concurrently through `gather`, which takes zero-argument
callables so each starts on its own worker:

```python
queries = ["proto bindings", "telemetry", "scenario health"]
results = gather(*[lambda q=q: search_hub.query.query(query=q, rows="ranked") for q in queries])
print([r.count() for r in results])
```

Bind the loop variable as a default argument (`lambda q=q:`) or every callable
closes over the last value.

## The Python surface

Programs run against an allowlisted builtin set, not the full interpreter. It
covers ordinary programming: containers, `type`, `getattr`, `next`, `iter`,
`round`, `format`, `sorted`, `zip`, comprehensions, `def`, `class`, `lambda`,
`with`, and **every standard exception**, so `except KeyError:` works. `import`
is available for the standard library.

Withheld on purpose: `eval`, `exec`, `compile`, `globals`, `locals`, `vars`,
`input`, `breakpoint`, `help`. Referencing one raises an ordinary `NameError`.
This is a guardrail against accidents, not a security boundary — the posture is
trusted-local-agent.

## What happens before your program runs

Submission statically resolves every name against the live registry before a
kernel executes anything. Three severities; only `error` refuses:

```
error   line 3  name "test_geni" does not resolve to a governed binding namespace
                or a built-in; nearest match: "test_genie"
warning line 7  scenario namespace "search_hub" is shadowed for the rest of this session;
                reach the binding as __vrooli__.search_hub
```

Resolution never guesses. A name that does not resolve is refused with a
suggestion, and no suggestion is offered when nothing is genuinely close — a
wrong suggestion is worse than none. Use `--explain` on `programs submit` to get
diagnostics without executing.

Names the program itself binds — function names, parameters, lambda arguments,
comprehension and loop variables, `with` and `except` targets — are never
flagged. Every refused name is recorded, which is how the fleet learns which
operations agents reach for and cannot invoke.

## Reading a failure

`failure_shape` carries a **cause**, never a Python exception class. The closed
vocabulary is:

| Cause | What to do |
|---|---|
| `unresolved_name` | Fix the name; the message names the nearest match. |
| `unknown_field` | The argument matches no proto field; the message lists the candidates. |
| `ambiguous_response` | Pass `rows="<field>"`. |
| `unreachable_scenario` | The owning scenario is not running. Start it through the lifecycle. |
| `refused_no_grant` | A destructive binding needs an explicit session grant. Request it; do not retry. |
| `refused_not_run_eligible` | The manifest declares the command ineligible. There is no override. |
| `inference_spend_exceeded` / `delegated_run_spend_exceeded` | The session ceiling stopped the work. |
| `deadline_exceeded` | The 120-second supervisor deadline fired and the kernel restarted; live variables were lost. |
| `kernel_syntax` | The program did not parse. |
| `kernel_runtime` | An uncaught Python exception. Read `failure_detail`. |
| `bridge_transport` | The call did not reach the owning scenario. |
| `unclassified` | No cause was derivable — reported honestly rather than guessed. |

`failure_cause` is the typed enum mirror of the same value.

## Finding the operation and its contract

```text
program-runtime bindings list                  # the callable namespace
program-runtime bindings describe test-genie/runs/list
program-runtime bindings unbound               # why something is not callable
```

In-kernel, `describe("test-genie/runs/list")` returns the same descriptor-backed
argument contract, and `discover("intent")` returns one governed capability or
an explicit null verdict. Treat a null verdict as a stop, not as permission to
guess a path or shell out.

## Runtime verbs

| Verb | What it does |
|---|---|
| `discover(intent)` | One governed binding for an intent, or an explicit null verdict. |
| `recall(intent, depth="fast")` | Governed records and docs through search-hub. `depth="deep"` widens the result set. |
| `validate(scenario, depth="fast")` | The **latest recorded** test-genie run verdicts for a scenario. |
| `capture(text, kind="note")` | Writes to vrooli-memory. `kind="work-record"` also accepts `trigger`, `approach`, `evidence`, `outcome`. |
| `ai.classify / extract / judge / write / batch` | Bounded typed inference through ai-gateway, schema-validated locally. `classify`, `extract`, and `judge` are deterministic and refuse a caller-supplied `temperature`; only `ai.write` accepts `temperature=` and `max_output_tokens=`. |
| `agent.start / collect / run` | A delegated agent-manager run and its evidence. |
| `gather(*callables)` | Concurrent fan-out. |
| `describe(binding_id)` | A binding's argument contract. |
| `reachable()` | Per-scenario reachability. |
| `lib.list()` / `lib.<name>()` | Promoted reusable programs frozen at session start. |

Each verb takes its primary argument positionally or by keyword:
`recall("retention")` and `recall(intent="retention")` are the same call.

**`validate` reads; it does not start a run.** Starting a run is a write that
test-genie exposes as run-ineligible, so no governed binding exists for it.
Start runs through the lifecycle (`vrooli scenario test <name>`) and block once
on the run id; never poll.

**`guide` is currently unavailable.** prompt-manager ships no resolved manifest
binding, so there is nothing typed to compose. The verb stays declared and fails
closed naming that reason rather than silently disappearing. Read skills through
`prompt-manager skill read <name>` until a binding exists.

Every verb fails closed and names its unavailable dependency. None of them falls
back to a shell call or a direct provider call.

## Task-shaped construction patterns

- [Namespace and contracts](../construction/namespace-and-contracts.md)
- [Handle shaping](../construction/handles.md)
- [Inference and discovery](../construction/inference-and-discovery.md)
- [Delegation](../construction/delegation.md)
- [Failure recovery](../construction/failure-recovery.md)

Runnable versions of these live in [`../examples/`](../examples/).
