---
name: "goal-loop"
description: "Run an improvement goal against one scenario until every setpoint row reads in band or the operator stops it: read sensors through the scenario's setpoint-read program, route by its improve skill, act or file, journal, record the goal in swarm-manager when it is up, and wake again."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["practice"]
  tags: ["goal", "loop", "self-improvement", "setpoint", "swarm-manager", "program-runtime", "heartbeat"]
  icon: "repeat"
  status: "active"
  revision: 3
  createdAt: "2026-09-02T00:00:00Z"
  updatedAt: "2026-09-02T23:00:00Z"
  requires:
    scenarios: ["prompt-manager", "program-runtime", "vrooli-memory"]
    commands: ["prompt-manager skill read", "program-runtime sessions create", "program-runtime programs submit", "program-runtime sessions delete", "program-runtime sessions get", "vrooli-memory journal note", "vrooli scenario status"]
    optional:
      scenarios: ["swarm-manager"]
      commands: ["swarm-manager goals get", "swarm-manager goals create", "swarm-manager goals targets-add", "swarm-manager milestones create", "swarm-manager milestones assign", "swarm-manager backlog create"]
      reason: "Swarm Manager is the record, never the gate. The loop runs when it is stopped and says so in the cycle note."
  origin:
    kind: "authored"
---
## Practice focus: Goal Loop

Given a sentence such as "self-improve and optimize the program-runtime scenario", regulate that scenario against its improve skill's setpoint until every row reads in band twice or the operator stops the loop. The loop decides nothing the improve skill does not already decide; it supplies the record and the stop rules. The wake cadence is an input from the caller (a `/loop` interval or a heartbeat interval); the loop has no default cadence and does not wake itself when none is given.

Required reading:
- `prompt-manager skill read <scenario>-improve` — the setpoint, sensors, routes, anti-gaming, and stop rules for the target. If it does not exist, the loop's only action is Phase 0.
- `path:scenarios/program-runtime/docs/guides/program-contracts.md` §"The envelope" — how to read `setpoint-read` output; its `reason` table is the vocabulary every route below keys on.
- `prompt-manager skill read improvement-do-and-dont` — applies to every action the loop takes.

### 1. Scope

**In scope:** one scenario per goal; resolving the target from the sentence; running the setpoint read; applying the improve skill's routes; curation moves in-cycle; filing heavy work; journaling; recording the goal and milestones in swarm-manager as a best-effort record; proposing close-out.

**Out of scope:** authoring the improve skill (`skill-set-authoring`); code changes (filed to the work ladder, executed under `implementation-plan-execution`); moving a goal to achieved (operator-only, through `swarm-manager goals close-out`); anything outside the target scenario's routes.

### 2. Process

#### Phase 0: Resolve and gate

**Entry:** a goal sentence.

1. Resolve the target: the exact scenario directory name. `ls scenarios/<name>` succeeds → that is the target. It fails, or the sentence names no single directory → stop and ask; never guess from a partial match.
2. Confirm the improve skill exists: `prompt-manager skill read <scenario>-improve`. If it does not, the loop's single proposal is a `skill-set-authoring` run for the scenario, journaled, and the loop ends.
3. Confirm `setpoint-read` exists: `ls scenarios/<scenario>/.vrooli/program-runtime/setpoint-read.json`. If it does not, same as step 2.
4. Read the improve skill's setpoint table into the loop's working set: one entry per row with `row`, `band`, `route`.

**Exit:** the target, the improve skill, and the program are known.

#### Phase 1: Record the goal (best effort)

**Entry:** Phase 0 passed.

1. `vrooli scenario status swarm-manager`. swarm-manager is not a requirement of this skill: if it is not running, skip this phase, note it in the cycle journal, and continue; never block on the record.
2. `swarm-manager goals get --name <scenario>-improve`. If it exists, reuse it. If not: `swarm-manager goals create --name <scenario>-improve --title "<scenario>: setpoint rows in band"` with no `--targets`: a goal target is a backlog item (`kind/name`), and none exists yet. After the first backlog item of Phase 2 is created, `swarm-manager goals targets-add --name <scenario>-improve --targets <kind>/<name>`.
3. One milestone per setpoint row, acceptance in Given/When/Then form as the command requires: `swarm-manager milestones create --goal <scenario>-improve --name <row> --title "<row> in band" --acceptance "Given <sensor command>, when the loop reads it, then the reading is <band>"`. Rows that are `pending-telemetry` get `--acceptance "Given <sensor command>, when the loop reads it, then a reading exists"`.

**Exit:** the goal and milestones exist, or their absence is recorded.

**Artifacts:** the goal name, for every later backlog item.

#### Phase 2: Cycle

**Entry:** Phase 0 passed. Repeats on every wake.

1. **Read.** One session per cycle:

   ```
   program-runtime sessions create --name <scenario>-improve-cycle-<n> --wall-budget <budget> --json
   program-runtime programs submit --session-id <id> --source-file scenarios/<scenario>/.vrooli/program-runtime/setpoint-read.py --provenance agent --json
   program-runtime sessions delete <id> --reason "cycle <n> read done"
   ```

   `--json` on both because the session id is read from `.session.id` and the envelope from `.program.stdout`. `<budget>` is the budget line of the improve skill's stop rules. Add `--async --wait-timeout <budget>` to the submit when the contract's `budget.async` is true; a synchronous submit is bound at 120 s by the runtime, and the wait is served once. Take `signals.rows` from the envelope. Keep the previous cycle's rows for comparison.
2. **Sort.** Rows `in_band` are recorded. Rows `unavailable` are routed by their `reason` (the closed vocabulary in `program-contracts.md` §"The envelope"):

| `reason` | Loop does |
|---|---|
| `no_governed_binding` | Files ladder W1 on the first read (step 3, ladder row); never waits |
| `pending_telemetry` | Files by the filing recipe (step 3) on the first read; never waits |
| `kernel_invoke_budget` | Reads the sensor by hand from the CLI the improve skill names, once per cycle; files W1 if the row matters |
| `scenario_unreachable` | Records and waits one cycle; three consecutive cycles → files ladder W3 (the scenario does not run) |
| `read_elsewhere:<program>` | Runs that program in its own session (same three commands) and reads the row from its envelope; never counts the row as unavailable |
| `unreliable:<why>` | Records the reason; does not band or route |

   Rows out of band are then ordered by the improve skill's setpoint order; a corpus row below floor moves to the front and blocks every other route this cycle.
3. **Route.** For the first out-of-band row, take the route the improve skill's table names. Apply the decision table:

| Route named | Loop does | Then |
|---|---|---|
| Curation move | Performs it with the exact command the skill names; one move per row per cycle | Records before value |
| `pending_telemetry` row | `swarm-manager backlog create --data '{"kind":"chore","name":"<scenario>-measure-<domain>","title":"Declare measure <domain>.<name> for <scenario>","description":"<which improve row needs it>"}' --json`; the worker that picks it up cites `measures-adoption`. `report-bug` is not the channel: a missing measure is not a defect. If swarm-manager is down, the journal entry carries the item text and it is filed on the next wake | Marks the row filed |
| Ladder rung W0 to W3 | `swarm-manager backlog create --data '{"kind":"fix","name":"<scenario>-<row>-<rung>","title":"<row>: <route text>","description":"rung <rung>; reading <reading> vs band <band>"}' --json` then `swarm-manager milestones assign --goal <scenario>-improve --milestone <row> --items fix/<scenario>-<row>-<rung>`; `--json` because the created `kind/name` is read from the response for `assign` and for the goal's first `targets-add` (Phase 1 step 2). If swarm-manager is down, the journal entry carries the item text and it is filed on the next wake | Marks the row filed |
| File against another owner | `report-bug` against that owner | Marks the row filed |
| Route needs a grant | Stops the cycle and asks; never works around | Journals the refusal |

4. **Verify.** Re-run `setpoint-read` once after a curation move, in a fresh session (the three commands of step 1 again): session variables persist across submissions, so a re-read in the same session would reuse the stale `inputs`. A row that got worse reverts that move if the skill names a reverse command; otherwise it is journaled and the row is marked "do not repeat".
5. **Journal.** One `vrooli-memory journal note "<goal> cycle <n>: <row> <route> <outcome>" --kind work-record` with `--trigger "<goal> cycle <n>: <row> <reading> vs <band>"`, `--approach "<route>"`, `--evidence "<before> -> <after> on <sensor>"`, `--outcome "<in band | filed <ref> | reverted | unavailable:<reason>>"`. Scope: when `scenarios/<scenario>/.vrooli/service.json` declares `skills.learning.scope` (`jq -r '.skills.learning.scope // empty' scenarios/<scenario>/.vrooli/service.json`), add `--scope <that scope>` so the cycle lands beside the usage skill's learning spine; otherwise omit `--scope` and the entry goes to the ledger default. Say which in the entry's first line.
6. **Regression check.** Any row that was in band last cycle and is out now is journaled as a regression before anything else is routed next cycle.

**Exit:** one row acted on or filed, or all rows in band or unavailable.

**Artifacts:** the journal entry; filed items; the cycle's row table kept for the next wake.

#### Phase 3: Wake or stop

**Entry:** a cycle ended.

| Condition | Action |
|---|---|
| Every readable row in band, and the same was true last cycle | Journal "in band ×2"; propose close-out to the operator; stop waking |
| At least one row out of band and a route was taken | Wake at the caller's cadence (the `/loop` interval or the heartbeat interval). No cadence given → journal the row table and stop; the loop never picks an interval |
| Every out-of-band row is filed and waiting on others | Wake at the caller's cadence; nothing to act on until a filed item lands |
| A row has read `scenario_unreachable` for three consecutive cycles | Journal it; file ladder W3 (the scenario does not run) if not already; keep waking |
| The operator stops the loop | Journal the last row table; stop |
| Cycle budget spent (the `--wall-budget` from the improve skill's stop rules, or the runtime's 120 s synchronous bound when the contract is not `async`) | Journal; stop; do not start a new session to continue |

### 3. Convergence patterns

```
wake ──▶ read setpoint ──▶ any row below floor? ──YES──▶ repair corpus route only
              │                    │NO
              │                    ▼
              │            first out-of-band row ──▶ route table (§2 step 3)
              │                    │
              │                    ▼
              │            verify (one re-read) ──▶ journal ──▶ Phase 3
              └── previous rows kept for regression check
```

Two agents at the same envelope must take the same route: the improve skill's table names it, the loop only orders rows.

### 4. Knowledge capture

Every cycle leaves a work record with before and after on one sensor. A filed item carries the reading and the route text so the executor does not re-derive them. When swarm-manager is up, the same evidence is appended to the milestone by the backlog item; when it is down, the journal is the record and Phase 1 is retried on the next wake.

### 5. Anti-patterns

| Anti-pattern | Why it fails | Better approach |
|---|---|---|
| Acting on an `unavailable` row's reading | Learns from a dead sensor | Route by `reason` (§2 step 2); a permanent reason is filed on the first read, a transient one waits |
| Two curation moves on one row in one cycle | Cannot attribute the change | One move, one re-read |
| Lowering a floor to get in band | Gaming (`improvement-do-and-dont` §1 "DON'T loosen or delete a tagged test") | Repair the corpus route; the floor derivation is recorded |
| Editing the scenario's registry or ledger to match a claim | Gaming (`improvement-do-and-dont` §1 "DON'T delete a known-issue ledger"; fails §2, the skeptic test) | Route to the ladder |
| Blocking on swarm-manager | The record is not the loop; swarm-manager is not in this skill's `requires` | Best effort; journal its absence |
| Polling a program or a run | Forbidden by the runtime | Wait once; the runtime wakes on terminal |
| Closing the goal | Operator-only | Propose close-out |

### 6. Output expectations

You may: run `setpoint-read`; perform curation moves the improve skill names; file backlog items, `measures-adoption` items, and bugs; write journal entries; create the goal and milestones.

You must: read before acting on every wake; keep one move per row per cycle; journal every cycle; propose close-out rather than declaring it.

You must not: change code; change measures, corpora, or floors; act on another scenario; continue after the operator stops the loop.

### 7. Troubleshooting & Edge Cases

| Symptom | Likely cause | First check | Fix |
|---|---|---|---|
| Every row reads `scenario_unreachable` | program-runtime cannot reach the scenario | `vrooli scenario status <scenario>` | Start it through the lifecycle; do not read sensors by hand |
| `programs submit` reports "session not found" | The cycle's session was deleted (`program-runtime sessions delete <id> --reason`) or reclaimed by the runtime | `program-runtime sessions get <id>` | Create a session per cycle; never reuse across wakes |
| A curation command is refused | The session lacks the grant | the refusal's binding id | Stop and ask; record in the journal |
| `swarm-manager goals create` says the goal exists | A prior loop or operator created it | `swarm-manager goals get --name <goal>` | Reuse; add the new item with `goals targets-add`; do not delete |
| The same row is routed every cycle with no movement | The route does not reach the cause | the work records for that row | After three cycles, file it as a ladder item with the three readings; stop routing it |
| Two setpoint rows share one sensor | Improve skill defect | the setpoint table | Report to `skill-set-authoring` as a repair; act on the first row only |
