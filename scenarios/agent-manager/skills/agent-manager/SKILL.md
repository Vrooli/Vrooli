---
name: "agent-manager"
description: "Use Agent Manager to run and read agent work: choose one bare run or a declared workflow, choose a result spec or a classify set, wait the durable way, read a run through report, episodes, and findings, and record what the run taught in the agent-manager-usage scope."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools"]
  tags: ["agent-manager", "run", "workflow", "subagent", "delegation", "result-spec", "classify", "episodes", "friction", "findings", "investigation", "learning-spine"]
  icon: "terminal"
  status: "active"
  revision: 1
  createdAt: "2026-09-02T00:00:00Z"
  updatedAt: "2026-09-02T00:00:00Z"
  learning:
    scope: "agent-manager-usage"
    capture: "every attempt"
  requires:
    scenarios: ["agent-manager", "program-runtime", "vrooli-memory", "prompt-manager"]
    commands: ["agent-manager run create", "agent-manager run get", "agent-manager run report", "agent-manager run result", "agent-manager run events", "agent-manager run episodes", "agent-manager run messages-friction", "agent-manager run invocation-facts", "agent-manager run episode-cohort", "agent-manager run cohort-report", "agent-manager run park", "agent-manager run wake", "agent-manager run await-result", "agent-manager run continue", "agent-manager run stop", "agent-manager run investigate", "agent-manager run apply-investigation", "agent-manager findings list", "agent-manager workflow simulate", "agent-manager workflow start", "agent-manager workflow execution-wait", "agent-manager workflow execution-get", "agent-manager workflow execution-result", "agent-manager workflow trace", "agent-manager task create", "agent-manager profile list", "agent-manager measures run-success-rate", "vrooli-memory recall wake", "vrooli-memory recall recall", "vrooli-memory journal note", "prompt-manager skill read"]
  origin:
    kind: "authored"
---
## Tools focus: Agent Manager

Use Agent Manager when work must run as a governed agent run: a task the current session should not do itself, a multi-step workflow with typed hand-offs, or a diagnosis of runs that already happened. This skill holds the judgment `--help` does not: which shape to start, how to wait, and how to read a finished run. Command flags are in the CLI (`agent-manager <group> <command> --help`); several subcommands print only a usage line, so read the flag names here or in `cli/cmd_*.go`.

Required reading:
- `prompt-manager skill read vrooli-memory` — the mechanics of the learning spine this skill uses.
- `prompt-manager skill read investigating-agent-runs` — how to read a run report; cited, not restated.
- `prompt-manager skill read program-runtime` — when a program's `agent.start`/`agent.collect` is the layer instead of this CLI.

### 1. Scope

**In scope:** starting one run or one workflow execution; result specs; waiting; reading one run or a cohort; parking and waking; investigations and findings; what to journal.

**Out of scope:** authoring workflow JSON (`path:scenarios/agent-manager/docs/guides/workflow-adoption.md`); profile and runner policy; regulating Agent Manager itself (`agent-manager-improve`); starting a scenario (`vrooli scenario start`).

### 2. Before acting

1. `vrooli-memory recall wake --scope agent-manager-usage` for the ambient set. [S1]
2. When the task names a workflow or a profile: `vrooli-memory recall recall "<workflow key or profile>" --scope agent-manager-usage --limit 5`. [S1]
3. Apply what the entries say before choosing a branch below.

### 3. The decision tree

```
I have work for an agent
│
├─ Is it one prompt with one answer?
│   ├─ yes → a bare run                                                        (§3.1)
│   └─ no: two or more agent steps with typed hand-offs, or a wait between them
│        → a declared workflow                                                 (§3.2)
│
├─ Am I already inside a program-runtime program with arity over many units?
│   └─ yes → agent.start(**req) / agent.collect(h, wait_seconds<=300); not this CLI   [S3, program-runtime]
│
└─ Is the work reading runs that already happened?                              (§4)
```

#### 3.1 A bare run

```
Bare run
├─ Need a task first?  agent-manager task create --title "<t>" --scope-path <path>      [S1]
├─ Which profile?      agent-manager profile list; pick by role, not by model             [S1]
├─ Result shape
│   ├─ the answer is one of a fixed set → --classify a,b,c                                [S1]
│   ├─ the answer has fields the caller reads → --result-schema-file <schema.json>        [S1]
│   └─ the caller reads prose only → neither; read `run result <id>`                      [S1]
├─ Start:  agent-manager run create --task-id <task> --profile-id <profile> --prompt "<p>"
│            [--classify ...|--result-schema-file ...] --workload-key <stable-key>
│            --workload-kind adhoc --idempotency-key <k>                                   [S1]
└─ Wait: there is NO blocking wait for a bare run (§3.3)
```

Rules:
- Exactly one of `--classify`, `--result-schema`, `--result-schema-file`; the CLI refuses two.
- `--workload-key` is what `measures repeated-work-rate` and `token-attribution` group on. Reuse the same key for the same recurring job; do not put a timestamp in it.
- `--idempotency-key` makes a retry of `run create` return the same run instead of a second one.
- A run that needs another producer's result parks itself: `agent-manager run park <id> --producer <scenario> --key <k>`; the producer's owner wakes it with `run wake <id> --result "<text>"`; the run reads it with `run await-result <id>`. [S1]

#### 3.2 A declared workflow

```
Workflow
├─ Which one?      agent-manager workflow list --owner <scenario>                              [S1]
├─ Dry plan first: agent-manager workflow simulate --owner <o> --key <k> --input-file in.json  [S1]
├─ Start:          agent-manager workflow start --owner <o> --key <k> --input-file in.json
│                    --idempotency-key <k>                                                      [S1]
├─ Wait ONCE:      agent-manager workflow execution-wait <execution-id> --timeout-seconds 1800 [S1]
├─ Read:           workflow execution-get <execution-id>; workflow trace <execution-id>        [S1]
└─ Payloads:       workflow execution-result <execution-id> --explicitly-authorized            [S1]
```

`execution-wait` is a server-side wait: it returns on the terminal transition or at the bound. Block once. A `--timeout-seconds 0` blocks until terminal. Never wrap it in a loop.

#### 3.3 Waiting on a bare run

A bare run has no server-side wait verb. Use this table; do not poll `run get`.

| Situation | Durable pattern | Rung |
|---|---|---|
| You can choose the shape before starting | Declare a one-node workflow around the prompt and use `workflow execution-wait` | S1 |
| The run is already started and you are interactive | `agent-manager run events <id> --follow` streams to a terminal; it is a live socket, not a durable wait, and it ends when the session ends | S1 |
| The run is already started and you are headless | Record the run id in your journal entry and stop; the next session reads `run get <id>` once | S1 |
| Another scenario produces what the run needs | Park (§3.1); the producer wakes it | S1 |

### 4. Reading runs

```
Reading
├─ One run, what happened?          run get <id> → status, phase, exit code                        [S1]
│   ├─ status FAILED or NEEDS_REVIEW  → run report <id>; then investigating-agent-runs                 [S1]
│   ├─ result missing or abstained    → run result <id>                                              [S1]
│   └─ succeeded but slow or costly   → run episodes <id>; reviewing-agent-run-efficiency              [S1]
├─ One run, what did the agent say hurt?   run messages-friction <id>                                 [S1]
├─ One run, which commands ran?            run invocation-facts <id>                                  [S1]
├─ Many runs, what recurs?
│   ├─ by tag         → run episode-cohort --tag-prefix <p> --limit 50                               [S1]
│   ├─ by explicit ids → run cohort-report --run-ids a,b,c                                            [S1]
│   └─ for one owning scenario → run agent-manager.friction-digest (inputs: scenario, window_days)   [S3]
│         status ok       → signals.top_fingerprints are the fingerprints to act on
│         status partial  → act on what is present; errors[0].class names the gap
│         errors[0].class scenario_unreachable → agent-manager is down; journal, stop
│         errors[0].class no_runs_in_window    → widen window_days once, or accept the empty reading
│         errors[0].class invalid_input        → fix scenario or window_days; nothing was read
├─ Recurring findings from investigations   findings list [--severity recurring] [--fingerprint f]    [S1]
└─ Fleet rate for one week                  measures run-success-rate --window last_7d              [S1]
```

Running the `[S3]` leaf: `program-runtime sessions create --name friction-digest --json`, then `programs submit --session-id <id> --source-file <copy> --provenance <agent|operator> --json` where `<copy>` is the scenario source with `inputs = {"scenario": "<slug>", "window_days": 7}` prepended in your scratch space, then `sessions delete <id> --reason "<why>"`. One fresh session per run; session variables persist and a stale `inputs` would be reused. Branch only on the envelope's `status` and `errors[0].class`; `.vrooli/program-runtime/friction-digest.json` declares every value. `unavailable` is unknown, never zero.

Reading `run report`: status and cost first, then result provenance, then failure counts. Its `Next:` lines are the drill-down; the report deliberately excludes transcript bodies. Reading `run episodes`: each row is a bounded episode with `pattern`, `causeScope`, `fingerprint`, `severity`, `suspectedOwnerScenario`, and `ownerConfidence`. A row with `ownerConfidence: unknown` has no owner scenario; do not attribute it by hand.

### 5. Investigating

```
Investigate
├─ One or more failed runs → run investigate --run-ids a,b --depth standard              [S1]
│     returns an investigation run id; read it with run get / run report
├─ Findings approved by the operator → run apply-investigation <investigation-run-id>    [S1]
└─ The same fingerprint keeps returning → findings list --fingerprint <f>;
      publish-recurring-friction is the improve skill's move, not this one                [S0]
```

The investigation and apply runs render `agent-manager-process-investigation` and `agent-manager-process-investigation-apply` as their prompts; do not paste those skills into a bare run.

### 6. After acting, always

One entry per attempt, success or not:

```
vrooli-memory journal note "<two lines>" --scope agent-manager-usage --kind <run-record|workflow-note> \
  --trigger "<task, with workflow key or profile named>" \
  --approach "<run create|workflow start; result spec; wait pattern>" \
  --evidence "<run id or execution id; status; report line that decided>" \
  --outcome "<complete|failed:<class>|unavailable:<reason>; next time: <one line>>"
```

Entry kinds: `run-record` for a bare run, `workflow-note` for an execution. Curation leaves: pin an entry on its third confirmation; supersede an entry whose wait or profile advice failed; propose a rule when the same workflow key keeps needing the same facet (`prompt-manager skill read vrooli-memory` §2). `run vrooli-memory.scope-bootstrap` creates starter rules for this scope. [S3]

### 7. In-use settings

| Symptom | Move | Journal |
|---|---|---|
| Result parses as prose when the caller needs a label | Restart with `--classify a,b,c` | the label set and the run id |
| Result has fields but `run result` shows abstained | Restart with `--result-schema-file`; keep the schema in the caller's repo | the schema path |
| Runs of one job are not grouped in `measures repeated-work-rate` | Set one stable `--workload-key` | the key |
| `run create` retried and made a second run | Add `--idempotency-key` | both run ids |
| Workflow wait returned at the bound, execution still running | Call `execution-wait` again with a longer `--timeout-seconds`; one call, not a loop | the execution id and bound |
| Run stalls on another scenario's output | Park; name the producer and key | producer, key |

### 8. Debug order

1. `vrooli scenario status agent-manager`.
2. `agent-manager run get <id>` — status, phase, exit code.
3. `agent-manager run report <id>` — then `investigating-agent-runs`.
4. `agent-manager run events <id> --failed`.
5. `agent-manager workflow trace <execution-id>` for a workflow node.

### 9. Safety

- Never run an agent binary directly to bypass a run; every agent execution is a run.
- `run approve`, `run reject`, and `run apply-investigation` change files; run them only with an operator decision in hand.
- `workflow execution-result` reveals payloads; pass `--explicitly-authorized` only when the caller may see them.
- Do not put secrets in `--prompt`; runs are retained and reported.

### 10. Troubleshooting & Edge Cases

| Symptom | Likely cause | First check | Fix |
|---|---|---|---|
| `--help` on `run create`, `task create`, `workflow start`, `findings list` prints no flags | Help renders only the usage line for those subcommands | `agent-manager run create --task-id x` errors name the flags | Use the flag names in this skill; file `report-bug` against agent-manager |
| `run report` through program-runtime returns `501 GetRunReport is not implemented` | The binding exists; the RPC is unimplemented | `program-runtime bindings describe agent-manager/run/report` | Read the report with the CLI; file against agent-manager |
| `run create` refuses: "use exactly one of --result-schema, --result-schema-file, or --classify" | Two result specs given | the command line | Keep one |
| `execution-wait` returns immediately with a terminal status you did not expect | The idempotency key matched an earlier execution | `workflow execution-get <id>` `created_at` | Use a new key for new work |
| `run episodes <id>` returns zero rows on a finished run | The run has no deterministic friction, or it is still finalizing | `run get <id>` `finalizationStatus` | Re-read after `RUN_FINALIZATION_STATUS_SUCCEEDED` |
| `measures <x>` prints `state: unreliable` | Classified share is below 90 percent, or the sample is under 5 | the `validity.reason` line | Read the number as unreliable; do not quote it as a rate |
| `run wake <id>` refused | The run is not `RUN_STATUS_PARKED` | `run get <id>` | Only a parked run wakes |
| Heartbeat runs crowd `run list` | Heartbeat members create runs every cycle | `run list --tag-prefix <your prefix>` | Filter by tag or workload key |
