---
name: "agent-manager"
description: "Use Agent Manager to run, read, and recall agent work: choose one bare run or a declared workflow, search retained conversations from incomplete clues, inspect bounded context with provenance, and record what the attempt taught in the agent-manager-usage scope."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools"]
  tags: ["agent-manager", "run", "workflow", "conversation-search", "recall", "subagent", "delegation", "result-spec", "classify", "friction", "findings", "investigation", "learning-spine"]
  icon: "terminal"
  status: "active"
  revision: 11
  createdAt: "2026-09-02T00:00:00Z"
  updatedAt: "2026-11-05T01:16:00Z"
  learning:
    scope: "agent-manager-usage"
    capture: "every attempt"
  requires:
    scenarios: ["agent-manager", "program-runtime", "test-genie", "vrooli-memory", "prompt-manager"]
    commands: ["agent-manager run create", "agent-manager run get", "agent-manager run report", "agent-manager run result", "agent-manager run events", "agent-manager measures cohort-report", "agent-manager run park", "agent-manager run wake", "agent-manager run await-result", "agent-manager run continue", "agent-manager run stop", "agent-manager investigation start", "agent-manager investigation get", "agent-manager investigation list", "agent-manager investigation wait", "agent-manager investigation cancel", "agent-manager run apply-investigation", "agent-manager workflow simulate", "agent-manager workflow start", "agent-manager workflow execution-wait", "agent-manager workflow execution-get", "agent-manager workflow execution-result", "agent-manager workflow trace", "agent-manager task create", "agent-manager profile list", "agent-manager conversation search", "agent-manager conversation context", "agent-manager conversation index status", "agent-manager measures get", "test-genie runs artifacts", "test-genie runs download-artifact", "search-hub query query", "vrooli-memory recall wake", "vrooli-memory recall recall", "vrooli-memory journal note", "prompt-manager skill read"]
  origin:
    kind: "authored"
---
## Tools focus: Agent Manager

Use Agent Manager when work must run as a governed agent run: a task the current session should not do itself, a multi-step workflow with typed hand-offs, or a diagnosis of runs that already happened. This skill holds the judgment `--help` does not: which shape to start, how to wait, and how to read a finished run. Command flags are in the CLI (`agent-manager <group> <command> --help`); several subcommands print only a usage line, so read the flag names here or in `cli/cmd_*.go`.

Required reading:
- `prompt-manager skill read vrooli-memory --output=skills --format=markdown` — the mechanics of the learning spine this skill uses.
- `prompt-manager skill read investigating-agent-runs --output=skills --format=markdown` — how to read a run report; cited, not restated.
- `prompt-manager skill read program-runtime --output=skills --format=markdown` — when a program's `agent.start`/`agent.collect` is the layer instead of this CLI.

Read each required skill once per task context and reuse it; use `skill show`
once for metadata, never as a progress check. Re-read only when the source
changed or the required guidance is missing from the active context.

If Prompt Manager or Search Hub is unavailable, do not repeat the same failed
lookup or stop unrelated work. Read the canonical source from its owning scenario
(`scenarios/<owner>/skills/`) or Prompt Manager's source pack
(`scenarios/prompt-manager/store/skills/packs/`), then continue; record that
registry usage was unavailable. If the source is absent, preserve that unknown
and continue independent work.

### 1. Scope

**In scope:** starting one run or one workflow execution; result specs; waiting; reading one run or a cohort; discovering retained conversations from incomplete clues; bounded context and provenance; parking and waking; caller-neutral finite investigations and findings; bounded learning retry; what to journal.

**Out of scope:** authoring workflow JSON (`path:scenarios/agent-manager/docs/guides/workflow-adoption.md`); profile and runner policy; regulating Agent Manager itself (`agent-manager-improve`); starting a scenario (`vrooli scenario start`).

### 2. Before acting

1. `vrooli-memory recall wake --scope agent-manager-usage` for the ambient set. [S1]
2. When the task names a workflow or a profile: `vrooli-memory recall recall "<workflow key or profile>" --scope agent-manager-usage --limit 5`. [S1]
3. Record advice IDs applied or rejected and the decision each changed. Advice is evidence, not authority. Keep task/attempt IDs, an ordinal, comparable context and observed start timestamps before orientation.

For a plan-family cohort, read `agent-manager-plan-family-supervision` before
creating a watch. It owns the watch/assessment path.

For arbitrary large efforts, use `agent-manager effort board` (or `--effort-ref`
for an exact selected effort) and the existing UI's Efforts view. Read
`large-effort-supervision` for judgment and
`path:scenarios/agent-manager/docs/reference/effort-supervision.md` for typed
enrollment and withdrawal requests and the delivery orchestration commands
(`run continue`, `run park`, `run wake --key`, `run report --children`, `effort epoch-check`).
Automatic discovery permits observation only; completion is independent of
accepted outcomes. Missing metrics remain unknown. Steer a delivery goal through
its goal home files, not the board. Explicit local operators can use
`--local-owner` on supported owner mutations without persisting tokens.

`effort signals <ref> --after <seq>` is a one-shot cursor read, not a wait,
subscription or check-in trigger. Read it at a real review event, then act on new
signals or wait through the relevant worker/owner route; do not poll the cursor
for activity. Advancing `--after` without a signal is not progress evidence.

`effort board` is also a snapshot, not a progress signal. Read the exact effort
when its reference is known; use paged discovery only when the target is unknown.
Do not repeat an unchanged board to infer progress. Use a supported owner wait or
return at the next material review event.

For a bounded diagnosis of an existing run set, use the typed investigation
operation or the `agent-manager.investigate` program. Its operation status is
independent from diagnostic condition and applicability. A completed result
may be `inconclusive`; missing coverage is never an observed zero. Start with
one stable request key, wait once through the owner, and reattach with get or
wait after a disconnect. If `learning.captureState` is `pending`, retry the
retained result through `POST /api/v1/investigations/{id}/learning-retry`; this
does not rerun diagnosis.

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
     ├─ known run id → inspect that run directly                                (§4)
     └─ unknown run id → search retained conversation history                   (§4.1)
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
- `--workload-key` is what `measures get workload-breakdown` and `measures get workload-efficiency` group on. Reuse the same key for the same recurring job; do not put a timestamp in it.
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

A bare run has no server-side wait verb. Use this table; do not poll `run get`
or `run list`. `run list` is a snapshot for discovery and inventory, not a
progress signal; repeated reads do not provide a wait guarantee.

| Situation | Durable pattern | Rung |
|---|---|---|
| You can choose the shape before starting | Declare a one-node workflow around the prompt and use `workflow execution-wait` | S1 |
| The run is already started and you are interactive | `agent-manager run events <id> --follow` streams to a terminal; it is a live socket, not a durable wait, and it ends when the session ends | S1 |
| The run is already started and you are headless | Record the run id in your journal entry and stop; the next session reads `run get <id>` once | S1 |
| Another scenario produces what the run needs | Park (§3.1); the producer wakes it | S1 |

`run events <id> --follow` uses the resolved API base, including `--api-base`,
and the saved API token for the WebSocket handshake. It forwards the current run
identity and CLI invocation provenance through the shared CLI hooks. No earlier
HTTP command is required. Ctrl+C or a normal server close ends the stream.
The stream decodes the server's protobuf envelope and displays event content,
progress, and status for the selected run. A terminal status is an observation;
it does not make this socket a durable wait. Decode failures are explicit errors.

For planned maintenance, `run quiesce --scenario <name> --timeout <duration>`
waits once under the server's timeout; ordinary HTTP deadlines must not truncate
that wait. Its scope/tag filters identify affected runs, not a global admission
barrier. Recheck other active runs before restarting Agent Manager itself. Without
`--force`, timeout leaves live work untouched. Do not infer global idleness from
a scenario-scoped drained response.

#### 3.1 Program contracts

When the task is about which Agent Manager capability to reuse, these declared
programs are the owner-routed leaves. Read the contract for full schemas.

| Need | Program | Purpose | Required inputs |
|---|---|---|---|
| Recover a retained conversation | `agent-manager.conversation-recall` | Merge bounded direct and federated conversation clues | `clues`; optional scope, time, mode, and result limits |
| Digest recurring friction | `agent-manager.friction-digest` | Rank deterministic friction fingerprints for one owner scenario | `scenario`, `window_days` |
| Start an investigation | `agent-manager.investigate` | Create and collect one durable investigation | `subject`, `scope` |
| Collect investigation evidence | `agent-manager.investigation-evidence` | Gather bounded run reports for explicit investigation subjects | `run_ids` |
| Read the Agent Manager setpoint | `agent-manager.setpoint-read` | Report measured supervision and run-health rows | optional window inputs |
| Read one supervision case | `agent-manager.supervision-case-read` | Inspect one durable case and its evidence | `case_id` |
| Evaluate supervision | `agent-manager.supervision-evaluate` | Score a bounded supervision decision against policy | `case_id`, `policy_id` |
| Read one supervision experiment | `agent-manager.supervision-experiment-read` | Inspect experiment state and outcome evidence | `experiment_id` |

Use `program-runtime library run <name> --input key=value`; inputs are
JSON-decoded, so quote numeric-looking strings as JSON strings (for example
`--input 'window="7"'`). Do not reconstruct these joins with direct database
reads.

For `agent-manager.conversation-recall`, a `partial` envelope with
`signals.degradations` means the listed Search Hub provider legs were not
searched completely. Preserve returned Agent Manager candidates and provenance,
but do not interpret an empty candidate set as a complete negative. Follow
`signals.next_actions` and check provider health before another search. Sibling
provider hits remain counts only and never become conversation candidates.

### 4. Reading runs

```
Reading
├─ One run, what happened?          run get <id> → status, phase, exit code                        [S1]
│   ├─ status FAILED or NEEDS_REVIEW  → run report <id>; then investigating-agent-runs                 [S1]
│   ├─ result missing or abstained    → run result <id>                                              [S1]
│   └─ succeeded but slow or costly   → run report <id>; reviewing-agent-run-efficiency              [S1]
├─ One run, what did the agent say hurt?   run report <id> --json (selfReportSpans)                 [S1]
├─ One run, which commands ran?            run report <id> --facts                                    [S1]
├─ Many runs, what recurs?
│   ├─ by tag         → measures cohort-report --filter-json '{"tag_prefix":"<p>"}' --limit 50                       [S1]
│   ├─ by explicit ids → measures cohort-report --run-ids a,b,c                                            [S1]
│   └─ for one owning scenario → run agent-manager.friction-digest (inputs: scenario, window_days)   [S3]
│         status ok       → signals.top_fingerprints are the fingerprints to act on
│         status partial  → act on what is present; errors[0].class names the gap
│         errors[0].class scenario_unreachable → agent-manager is down; journal, stop
│         errors[0].class no_runs_in_window    → widen window_days once, or accept the empty reading
│         errors[0].class invalid_input        → fix scenario or window_days; nothing was read
├─ Recurring friction problems              usage friction [--status open]                          [S1]
├─ What should I change? (quota, tools,     usage recommendations [--run ID]                       [S1]
│   programs, corrections — one ranked list; each action names its owner, effect in its own
│   unit, evidence and the verify command; finished sessions read "would have saved")
└─ Fleet rate for one week                  measures get run-success-rate --window last_7d          [S1]
   (any other declared measure: measures list, then measures get <id> [--param name=value])
```

Running an `[S3]` leaf: use `program-runtime library run <scenario>.<program> --input key=value`. The runner owns inputs, session, wait, and cleanup. Branch on the declared envelope; never copy source into a scratch session for ordinary use.

For Test Genie evidence, discover and download artifacts through its owner CLI:
`test-genie runs artifacts <run-id> --scenario <name>` followed by
`test-genie runs download-artifact <run-id> <artifact-id> --scenario <name> --out <path>`.
Artifact IDs are opaque. Do not construct internal HTTP artifact URLs or curl
the scenario's private routes; the CLI preserves its authorization, catalog,
and destination-safety behavior.

Reading `run report`: status and cost first, then result provenance, then failure counts. Its `Next:` lines are the drill-down; the report deliberately excludes transcript bodies. `selfReportSpans` in `run report --json` are the assistant's own phrases ("blocked", "cannot find"): a cue to read the run, not a verdict. Command friction is read from `measures get tool-profiles` and `usage friction`, whose coverage line says how many commands came from runs an older parser read (outcomes there may be missing).

### 5. Investigating

```
Investigate
├─ One or more failed runs → investigation start --file request.json (subject run_ids a,b) [S1]
│     the finite typed lifecycle returns an investigation id
│     → investigation wait/get/list                                                        [S1]
├─ A reusable program caller → program-runtime library run agent-manager.investigate       [S3]
│     with a stable request key and bounded subject/evidence inputs
├─ Historical legacy result with an explicit operator decision → run apply-investigation   [S1]
│     retained only for old approval/apply records; it is never part of typed admission
└─ The same fingerprint keeps returning → usage friction (plain description, owner, cost);
      usage friction-report <fingerprint> files it with scenario-qa              [S0]
```

New diagnosis requests use the typed investigation contract and the bounded
`agent-manager-process-investigation` method behind the owner workflow. Do not
start the retired `agent-manager/investigate` workflow or paste either
investigation skill into a bare run. The apply skill is a separate authorized
repair path for retained legacy records; a typed diagnosis never grants repair
authority.

For unknown-run recovery, debug order, safety details, in-use settings,
troubleshooting, joined supervision reads, and caller-identity refusals, see
`references/extended-operations.md`.

### 6. After acting, always

Use `vrooli-memory learning record --scope agent-manager-usage --attempt '<Attempt JSON>'`
once per selected operation. Read the shared Memory skill for the exact contract.
Retain task identity across retries, one attempt identity/ordinal per actual
attempt, observed timestamps, comparison context, operation and owner evidence.
Record applied/rejected advice IDs and assessed or unknown verdicts. Capture
observed first-action time and tool round trips; omit unknown values. Mark fixtures
test. Only evidenced outcomes are verified_success; failed, unavailable and unknown
remain distinct. A completed run is not proof that supervision helped it.
Do not also append an ordinary journal run-record/workflow-note. Keep work-records
for implementation changes. On capture failure retain the unchanged payload and
retry its ID; the operation outcome does not change. Supersede contradicted advice
through the shared curation path. Supervision policy changes follow the owner
replay/promotion process and never come directly from remembered prose.
