# Program contracts

A program is one bounded unit of governed work. This guide is the standard for
programs a scenario owns: where they live, what they declare, how their source
is organized, and what they return. `program-construction.md` teaches how to
write the Python; this guide teaches how to make a program a dependable step
that a skill, a heartbeat, or another program can call without reading its
source.

The authoring brief's rules (`api/internal/harness/contract.json`) still apply
to every line of source. This guide adds the shape around the source.

## Where programs live

```
scenarios/<scenario>/.vrooli/program-runtime/
├── <name>.json        # the contract
└── <name>.py          # the source
```

Fixtures live inside the contract (`fixtures[]`), never in a sibling directory.

The scenario directory is the source of truth. program-runtime indexes each
contract as `<scenario>.<name>` in `program-runtime library search`, using its
purpose, inputs, and declared bindings for retrieval. Run a contract with
`program-runtime library run <scenario>.<name> --input k=v`; the command validates
inputs, creates the session, submits, waits once, prints the envelope, and
reclaims the session. A library row is never edited directly; edit the file.

Programs that start life in a session and are promoted with
`program-runtime library promote` stay library-owned until a scenario adopts
them by writing the pair into its own directory. Adoption is the durable form of
promotion.

A program that only exists to demonstrate a construction pattern is still a
program: program-runtime's own examples live in its
`.vrooli/program-runtime/` with contracts, and the docs link to them.

## The contract

One JSON file, validated against `schemas/program-contract.schema.json` in this
scenario. Every field is load-bearing for a caller.

| Field | What it declares | Who reads it |
|---|---|---|
| `name`, `version`, `purpose` | Identity and one sentence of intent | Library search, skills, humans |
| `inputs` | Each input's type, whether it is required, its default, and its enum when closed | The caller; the `programs` validation phase |
| `assumptions` | What must already be true (a scenario running, a workflow the caller may execute) | The caller, before calling |
| `invariants` | What the program promises never to do (no undeclared write, no row set above N materialized, one envelope on every path) | Reviewers; the validation phase checks the checkable ones |
| `bindings` | Every governed binding the source calls, with its effect (`read`, `write`, `destructive`) | The validation phase: each must exist, be run-eligible, and match the declared effect |
| `outputs` | The envelope: `status` enum, `phase` enum, `signals` shape, `errors[].class` vocabulary, `evidence` | The skill that branches on it |
| `budget` | Wall time, inference calls, delegated runs, and `async: true` when `wall_ms` exceeds the 120 s synchronous submission bound | The session's ceilings; the caller, which must submit with `--async` and wait once when `async` is set |
| `fixtures` | Inputs and the expected `status` (and optionally signals) | The validation phase, in a test-provenance session |
| `memory` | Present only when the program reads or writes a memory scope: the scope, the phase that reads, the phase that writes, the entry kinds | Reviewers; the skill-set validator |

Inputs arrive as one dict named `inputs`, bound by the caller before the
source runs (`library run` and `lib.<scenario>.<name>(...)` do this; a manual
caller may prepend `inputs = {...}` to the source). The kernel
withholds `globals`, so a program guards the name and falls back to the
contract defaults:

```python
try:
    inputs
except NameError:
    inputs = {}
workflow_id = inputs.get("workflow_id")
```

This keeps every program runnable stand-alone with its defaults. Session
variables persist across submissions, so a stale `inputs` from an earlier
submission in the same session would be reused: run each program call in its
own session.

Keyword arguments are matched to the request message's proto fields. Flat flag
names normalized to underscores (`window_seconds=`, `wake_budget=`) resolve; a
manifest flag carrying a `bind_waiver` is CLI-local and has no field to bind,
and list-valued arguments may fail to encode. Probe a write binding once in a
scratch session before a contract declares it.

Binding responses reach the kernel as protojson, so row keys are camelCase
(`governedShare`, `sampleProgramId`) while the CLI prints snake_case. Read the
keys from a probe (`print(handle.head(1)[0].keys())`) or from
`describe(...)` before writing a contract; a contract that names a field the
binding does not return is drift. A response's repeated field becomes the
handle's rows; every other field (totals, shares, latency) lives in
`handle.meta()`, so a scalar reading like `governedShare` is read from
`meta()`, not from `head(1)[0]`.

Protojson omits a scalar whose value is the type's zero: a `double` at `0.0`,
an `int` at `0`, an empty string, `false`. **A missing scalar is zero when the
response says the reading is available**; it is unavailable only when the
response itself says so (a `state` field, an error, an unreachable binding).
Read rates and counts as `float(m.get("rate", 0.0))` and
`int(m.get("count", 0))`. A program that reads absence as "unavailable" or
"out of band" can never observe the in-band value for a row whose target is
zero, which is exactly the row a sensor exists to confirm.

## Phases

Organize the source as named phases. Each phase is a function; a small state
machine drives them. Label each phase in a comment so a reader can find the
collecting, the classifying, and the acting without reading every line.

| Phase | Does | May not |
|---|---|---|
| `validate` | Checks inputs against the contract | Call any binding |
| `collect` | Governed reads, concurrent through `gather` | Call a binding with effect `write` or `destructive` |
| `classify` | Turns collected rows into labels: a deterministic table first, `ai.classify` only where judgment is real | Act |
| `decide` | A pure function of collected data and labels that produces a plan | Any I/O |
| `act` | Bindings with a declared effect, in the order the plan states | Retry; a refusal stops the program |
| `delegate` | `agent.start` and `agent.collect` for work that is genuinely agentic | Poll |
| `report` | Builds and prints the envelope | Skip; it runs on every path |

A program does not need every phase. A sensor-read program is
`validate → collect → report`. An orchestrator uses all seven. What is fixed is
the prohibitions and three ordering rules: `validate` first, `report` last, and
`collect` and `decide` before `act`. `classify` may run on collected rows or on
the result of `act` (an execution's outcome is only known after it ran), so
`validate → collect → act → classify → report` is a legal shape.

### The state machine

```python
STATES = {"validate": step_validate, "collect": step_collect, "classify": step_classify,
          "report": step_report}
state = "validate"
while state:
    try:
        state = STATES[state]()
    except Exception as exc:  # the one catch that guarantees an envelope on every path
        if envelope.get("phase") == "report":
            raise
        envelope["status"] = "failed"
        envelope["errors"].append({"class": "kernel_runtime", "detail": str(exc)[:240], "where": envelope.get("phase") or state})
        state = "report"
```

Each step returns the name of the next state or `None`. Any step may return
`"report"` early with a status set. There is no retry loop inside a program;
retry is the caller's decision, informed by the envelope.

The driver's catch is what makes "one envelope on every path" true: `group_by`,
`sort`, and `select` raise `KeyError` on a key protojson omitted, and a phase
that lets that escape would otherwise print nothing. Guard those calls where
you can; the driver is the last line.

Inside a phase, catch only what the bridge raises. `classify_transport` must
re-raise `NameError` and `AttributeError`: a kernel-bound name that is missing
(`gather`, `ai`, a scenario namespace) is a `kernel_runtime` failure, and a
program that labels it `binding_error` runs convincingly under plain Python and
lies about why it failed.

## The envelope

The program prints exactly one dictionary, on every path, including exceptions
caught inside a phase.

```python
envelope = {
    "program": "<scenario>.<name>", "version": "1",
    "status": "failed",            # set by the phases; see the table
    "phase": "validate",           # the last phase reached
    "inputs": {...},               # the inputs as resolved, for the record
    "signals": {},                 # numbers and labels the caller branches on
    "errors": [],                  # [{"class": ..., "detail": ..., "where": ...}]
    "evidence": [],                # ids and references, never bodies
}
```

| Status | Meaning | Program-runtime cause it usually follows | What a caller does |
|---|---|---|---|
| `ok` | Every phase ran; signals are complete | `PROGRAM_STATUS_SUCCEEDED` | Acts on the signals |
| `partial` | Some reads or steps failed; `signals` names what is missing and `errors` names why | succeeded, with errors | Acts on what is present; records the gap |
| `unavailable` | A dependency could not be reached; nothing is known | `UNREACHABLE_SCENARIO`, `BRIDGE_TRANSPORT` | Treats the reading as unknown; never as zero or as failed |
| `refused` | Governance stopped an action | `REFUSED_NO_GRANT`, `REFUSED_NOT_RUN_ELIGIBLE` | Requests the grant through the session path, or stops |
| `failed` | The action ran and the outcome is bad, an input was invalid, or a capability the program needs has no governed binding | succeeded, with a classified failure | Follows the error class |

Three cases are decided here so programs do not drift:

- A required input that is missing is `failed` with class `invalid_input`,
  never `partial`.
- A capability with no governed binding (`no_governed_binding`) is `failed`
  when the program cannot do its work, or `refused` when governance withheld it.
  It is never `unavailable`: unavailable means "retry later", and a missing
  binding is permanent until someone adds it.
- A program whose success shape is always `partial` (because one step has no
  binding yet) says so in its contract and lists only the statuses it can
  reach.

Two rules make the envelope trustworthy:

- **Unavailable is never zero.** A sensor that cannot be read is reported as
  `unavailable` with a reason. A caller that estimates in its place is wrong.
- **Errors are classified.** `errors[].class` is a value from the contract's
  declared vocabulary. Transport and governance classes map from the runtime's
  closed failure causes; domain classes (`selector_not_found`, `timeout`,
  `auth_required`) are the program's own and are the values a skill's decision
  tree branches on.

Program stdout is bounded (4 KB by default, `output_limit_bytes`), and an
overflow is truncated with a trailing `…` rather than refused. Size the
envelope to the bound: a handful of rows, short strings, deduplicated evidence.

A `fail(status, klass, detail, where)` helper is the one place a bad path is
recorded. It appends the error, sets the status, and returns `"report"`.

Transport classes are shared by every program. Until the bridge raises typed
exceptions, `classify_transport` maps the runtime's messages with this table
and nothing else; a program that adds its own substrings drifts from its
siblings.

| Runtime message contains | Status | Class |
|---|---|---|
| `is unreachable`, `bridge unavailable`, `scenario_not_running`, `no running runtime ports`, `connection refused` | `unavailable` | `scenario_unreachable` |
| `requires an explicit grant` | `refused` | `no_grant` |
| `not run eligible`, `run_eligible` | `refused` | `not_run_eligible` |
| `inference spend`, `delegated run spend` | `refused` | `inference_spend_exceeded`, `delegated_run_spend_exceeded` |
| `no determinable primary response field`, `rows must be one of` | `failed` | `ambiguous_response` |
| `accepts named proto fields`, `invalid arguments for`, `no proto field matches` | `failed` | `invalid_input` |
| `deadline` | `failed` | `deadline_exceeded` |
| anything else | `failed` | `binding_error` |

Copy this function into every program exactly. It is the only permitted form
until `lib` can share it; a review that finds a different table is a finding.

```python
def classify_transport(exc):
    """Map a bridge exception to (status, class). Copied verbatim from program-contracts.md."""
    if isinstance(exc, (NameError, AttributeError)):
        raise exc                                   # kernel_runtime: a bound name is missing; never relabel
    text = str(exc)
    for needle in ("is unreachable", "bridge unavailable", "scenario_not_running",
                   "no running runtime ports", "connection refused"):
        if needle in text:
            return ("unavailable", "scenario_unreachable")
    if "requires an explicit grant" in text:
        return ("refused", "no_grant")
    if "not run eligible" in text or "run_eligible" in text:
        return ("refused", "not_run_eligible")
    if "inference spend" in text:
        return ("refused", "inference_spend_exceeded")
    if "delegated run spend" in text:
        return ("refused", "delegated_run_spend_exceeded")
    if "no determinable primary response field" in text or "rows must be one of" in text:
        return ("failed", "ambiguous_response")
    for needle in ("accepts named proto fields", "invalid arguments for", "no proto field matches"):
        if needle in text:
            return ("failed", "invalid_input")
    if "deadline" in text:
        return ("failed", "deadline_exceeded")
    return ("failed", "binding_error")
```

Every contract's `outputs.errors.classes` therefore declares, at minimum:
`scenario_unreachable`, `no_grant`, `not_run_eligible`, `inference_spend_exceeded`,
`delegated_run_spend_exceeded`, `ambiguous_response`, `invalid_input`,
`deadline_exceeded`, `binding_error`, and `kernel_runtime` (the driver's catch),
plus the program's own domain classes.

A setpoint-read program's `signals.rows` has one shape fleet-wide so
`improve-cycle` can route any scenario's board:
`{row, reading, target, in_band, unavailable, reason}`. `target` and `in_band`
are `null` when the row has no band yet; the keys are always present.

`reason` is a closed vocabulary, and it decides what the reader does next:

| `reason` | Meaning | What `goal-loop` does |
|---|---|---|
| `no_governed_binding` | The sensor exists as a CLI command but has no program binding. Permanent until someone adds it. | Route as ladder W1 on the first read; never wait. |
| `kernel_invoke_budget` | The binding exists but its call outruns the kernel invoke budget (100 s). Permanent in-program. | Read it by hand from the CLI once per cycle; file W1 if the row matters. |
| `scenario_unreachable` | The owning scenario did not answer this time. Transient. | Wait one cycle; after three consecutive cycles, W3 (the scenario does not run). |
| `read_elsewhere:<program>` | Another program owns this row. Not pending. | Run that program; never count the row as unavailable. |
| `pending_telemetry` | No sensor exists yet. Permanent until a measure ships. | File a measures-adoption item; never band the row. |
| `unreliable:<why>` | The sensor answered but its own validity gate failed (sample too small, classified share below minimum). | Report the reason; do not band. |

A row with a permanent reason (`no_governed_binding`, `kernel_invoke_budget`,
`read_elsewhere`, `pending_telemetry`) does not lower the program's status: a
board whose only unavailable rows are permanent is `ok`. Only a transient
`scenario_unreachable` row or a failed read makes the board `partial`. The
contract lists only the statuses the program can reach.

Every `reading` is either a value the sensor returned or `null`. A program never
prints a number for a row it declined to evaluate; the improve skill's "today"
column copies `reading` and `reason` verbatim.

## Memory in programs

At steps up to S3, a program does not touch memory; the skill's decision tree
decides what to remember from the envelope. At S4, the tree lives in the
orchestrator program, and memory becomes contract-declared phases:

- reads happen in `collect` from the declared scope with the declared query;
- writes happen in `report` with the declared entry kinds;
- nothing reads or writes memory in `act` or `delegate`.

A memory dependency being down is degraded, not fatal: the program records a
`memory_unavailable` error, proceeds without recall, and skips capture.

## Calling a program

| From | How to call |
|---|---|---|
| A skill, a human, or a heartbeat | `program-runtime library run <scenario>.<name> --input key=value --json`. The command validates inputs, creates and reclaims its own session, submits with the contract's async budget, waits once, and prints one envelope. Manual session submission remains available for debugging. |
| Another program | `lib.<scenario>.<name>(input=value, ...)`, returning a Handle whose first row is the envelope. The contract index resolves the scenario namespace and enforces the declared inputs. |

The library walk projects `.vrooli/program-runtime/` into `lib.<scenario>` at
session start. The scenario contract and source remain the source of truth;
promoted library rows are separate reusable entries and are not used to
override a declared contract.

## Validation

A program is done when:

1. Its contract validates against the schema.
2. `program-runtime programs submit --explain` reports no diagnostics.
3. Every declared binding resolves with `bindings describe` and its effect
   matches the declaration.
4. The source calls no binding the contract does not declare.
5. Each fixture, run in a test-provenance session, produces the expected
   `status`; a fixture that cannot run because a dependency is down is recorded
   as `unavailable` with the reason, never skipped silently.
6. The envelope is printed once on every path.

Fixtures may carry a `note` for reviewers and `host_local: true` when they
depend on ids discovered on this host. Fixtures obey three rules: they never pin a host-specific id or port (discover
the subject at fixture time, or mark the fixture `host_local: true`); they never
mutate a live scope, workflow, or ledger (use a fixture-only scope or expect a
validation failure that creates nothing); and they expect only the status the
program should produce, never a broken outcome that happens to be current. A
fixture whose `requires` scenario is down is recorded as `unavailable`.

A `programs` test-genie phase owned by program-runtime is planned to turn these
checks into findings on the owning scenario's test run. Until it exists, run
them by hand before declaring a program done.

## Anti-patterns

| Anti-pattern | Why it fails | Instead |
|---|---|---|
| Retrying inside a program | Hides the failure class the caller needs, burns budget the caller did not grant | Return `failed` with the class; the caller retries once if its tree says so |
| Printing rows | Blows the output bound and puts data where a decision should be | `count`, `head(n)`, `group_by`; materialize only with a limit and a reason |
| A status of `failed` for a dead dependency | Teaches the caller's memory that the task fails | `unavailable` with the reason |
| Memory writes mid-action | Records outcomes before they are known | Write in `report`, once |
| A contract that names a field the binding does not return | Drift the caller only discovers at runtime | Read field names from `describe(...)` and keep the contract to what is verified |
| One program for every command | Skill sprawl in a different file type | A program earns its place by arity or by composition; a single call stays a CLI step |
| `partial` as the ordinary success status | The caller cannot tell a healthy run from a degraded one | Reach `ok` on the happy path; if one step has no binding yet, say so in the contract and list the reachable statuses |
| A default model, provider, or endpoint written into a program | Callers must not carry model slugs (`ai-gateway` guardrail); a silent default spends money the caller did not choose | Make the input required, or resolve the default from the owning scenario's registry in `collect` |
