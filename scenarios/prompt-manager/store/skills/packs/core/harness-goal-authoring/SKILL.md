---
name: harness-goal-authoring
description: Write the message a coding agent runs under in harness goal mode (Claude Code and Codex `/goal`, Agent Manager `until`), or the assignment a coordinator hands a sub-agent. Ten slots, seven handoff shapes (including a thin scheduler over long-lived segment owners), how a goal survives compaction, and the split between what the docs and plan hold and what the goal message holds. Not for Swarm goal records; use swarm-manager-work-authoring for those.
license: CC-BY-4.0
metadata:
  kind: skill
  schemaVersion: 1
  modes: [practice]
  tags: [goal, harness, until, delegation, sub-agent, orchestration, prompt]
  icon: target
  status: active
  revision: 29
  createdAt: "2026-09-11T00:00:00Z"
  updatedAt: "2026-11-10T03:00:00Z"
  requires:
    scenarios: [prompt-manager]
    commands: [prompt-manager skill read]
  origin:
    kind: authored
---

## Practice focus: Harness Goal Authoring

Write a goal message that a fresh agent can finish from, that a transcript-only
evaluator can judge. Keep design in durable documentation and the selected
work-shape artifact (a plan only when the shape requires one), not in the
message. Keep goals concise for token efficiency, while respecting the selected
harness's actual limits; Agent Manager allows up to 1 MiB of UTF-8 bytes in
`until`.

Required reading:
- `docs/agent-system/LONG_EFFORT_DESIGN.md` for any goal or sub-agent tree expected to run
  for hours: the design rules (D1–D17), the effort structures and the verified Codex
  multi-agent settings. This skill applies them; it does not restate the evidence.
- `docs/TESTING.md` — validation scope. A goal selects a posture; it does not
  redefine the scope rules.
- `implementation-plan-execution` §"Divergence tiers" and §"Blocked" when the
  selected shape is plan-backed — the receiving agent's rules for adjacent
  defects and for the word "blocked". A goal selects among them; it does not
  rewrite them.

Read registered skills with `prompt-manager skill read`. If that route is
unavailable, read the canonical `SKILL.md` in the repository when accessible;
record which source was read and continue. A registry/read outage is not a
human blocker and does not stop independent work. Never claim a skill receipt
from an assignment that only named the skill.

### 1. Scope

In scope: the text of a harness goal (`/goal <condition>` in Claude Code or
Codex, `--until` on `agent-manager run create`, the `until` field of an Agent
Manager workflow run node) and the assignment message a coordinator gives a
sub-agent, with or without native goal support.

Out of scope: Swarm goal records (`swarm-manager goals`), milestones, and backlog
item text. Read `swarm-manager-work-authoring` for those. A Swarm goal states a
change in the world; a harness goal is an execution mechanism. Do not mix them.

### Path resolution

Relative paths in a goal are relative to the repository root unless a command
declares another root. Write `scenarios/<s>/...` and `docs/...`, never bare
fragments such as `requirements.json`; paths outside the repository are absolute.
The receiving agent verifies paths before editing them.

### 2. How the harness judges the message

| | Claude Code `/goal` | Codex `/goal` |
|---|---|---|
| Who decides "met" | A separate small model reads the transcript only. It runs no commands and opens no files. Verdicts: Not yet met, Met, Impossible. | The working model calls `update_goal(complete)`. Each goal-turn start injects the objective and an audit: enumerate deliverables, reject proxy signals, treat uncertainty as not achieved. Nothing is injected inside a turn, and one turn can last hours. |
| After compaction | The goal condition stays with the harness evaluator. | The objective survives only when short (`LONG_EFFORT_DESIGN.md` §4a gives the measured limit); a longer one becomes `[objective omitted; exceeds the evidence limit]` and returns at the next goal-turn start (§2a). |
| What this requires of the author | Proof must appear in the transcript. Name the command and require its output to be shown. | Proof must be enumerable. State deliverables the audit can tick one by one. |
| Loop guards | Halts after several turns without tool use. The Stop hook is overridden after repeated consecutive blocks; the hooks guide states the cap. No turn cap exists; add one in the text. | A continuation turn with zero tool calls suppresses the next one. On budget exhaustion the goal becomes `budget_limited` and the model is told to wrap up. |
| Size | 4,000 characters. `ProposeGoal` proposals: 500. | Write a kernel of 600 characters or fewer for anything that runs past one compaction (§2a). Agent Manager caps `until` at 1 MiB of UTF-8 bytes; `--until-file` supports large values without shell argument limits. |

Agent Manager's delivery depends on runner support and the native field size:

| Condition | Delivery |
|---|---|
| No native objective support, parser/watcher unavailable, or confirmed `/goal` submission lacks a matching acknowledgement | The task prompt carries the full `until`; the run records prompt-carried delivery. |
| Native support and `until` fits what the harness keeps verbatim: 600 characters for Codex (a longer objective is omitted at compaction), 4,000 for Claude Code (its field limit) | `/goal` carries the full `until`; the task prompt points to it. |
| Native support and `until` exceeds that limit | `/goal` carries a short delegation that survives compaction, and the task prompt carries the full `until`. |

An uncertain `/goal` command submission or a canceled launch is not treated as
prompt fallback: task text is withheld because the native command may still be
in the composer or executing. Write one condition that remains complete in
either route: its full prompt-carried form is a contract; its native form is
the complete objective when it fits, or a short delegation backed by the full
prompt when bounded. A short pointer is not a substitute for the condition
when the native field was bounded.

### 2a. Persistence over hours

After an hour of work the agent no longer sees most of what you wrote. Its context keeps the
system prompt, `AGENTS.md`, its own recent messages, inherited history and the latest tool
output. A long objective, the stance paragraph and skills read early lose force first, and
the agent starts copying its own recent units. Write every long-running goal so the parts
that matter are re-presented:

| Rule | How |
|---|---|
| **Stance message, kernel, charter** | Codex drops a long goal objective at compaction but keeps the last 8 user messages (`LONG_EFFORT_DESIGN.md` D3, §4a). Send the **stance message** first: an ordinary user message of 3,500 characters or fewer, most important text first, holding the operator's stance verbatim and in full, the destination with its scale and the charter re-read instruction (template below). Then set the **kernel** as the native goal, 600 characters or fewer: the destination itself in one line with its scale (never only "the destination in <file>": a transcript-only evaluator opens no files), a verbatim stance excerpt of about 150 characters, the charter path and "one unit per turn". The **charter** file (packet, epoch file or working brief) holds the full stance and everything else. Check each length with `wc -m` before sending. In an Agent Manager run the task prompt is the stance message: put the stance and destination in its first 1,400 characters. |
| **Steer through the charter** | Every later user message, goal change and Agent Manager wake pushes the stance message toward eviction, and an unattended agent cannot send itself a new one. Put steering in the charter or feedback ledger; the agent's own defense is the charter re-read. An operator or supervising session that can message the agent resends the stance message after six newer messages. |
| **One unit per turn** | Under a native goal, tell the agent to end its turn after each unit's record (a round, an epoch return, a study checkpoint). Codex then re-injects the objective and the audit before the next unit. Without it, one turn can run for hours through many compactions with no objective. Without a native goal, ending the turn ends the effort: run the effort as a native goal or an Agent Manager run, or omit this rule. |
| **Re-read at unit start** | The first action of every unit re-reads the charter and the method skill's stance section. After any compaction, re-read the charter before acting. |
| **Stance in every persistent layer** | The operator's words go in full into the stance message, the charter and every sub-agent brief, and as a verbatim excerpt into the kernel. A child has no user messages, so its brief is the only copy it gets. A stance that lives only in a long goal message is gone after the first compaction. |
| **Curated precedent** | Point the agent at pinned exemplars of a good unit, never at "the latest records". Show recent output as numbers against the destination. |

Stance message template:

```text
Stance for <effort>, in the operator's words: "<full stance, verbatim>"
Destination: <end state in one line, with its scale>.
Charter: <charter path>. Re-read it at every unit start and after every compaction; it holds the method, proof, boundary and budget, and it wins over your recent units as the template.
```

In a Codex root session, send it as its own message ending `The goal follows this message.
Reply "ready" and wait for it.`, then set the goal. In an Agent Manager run, it opens the task
prompt and `--until-file` carries the kernel; omit that line, because no goal message follows
and waiting would end the run.

The verified Codex multi-agent settings in `LONG_EFFORT_DESIGN.md` §4 also tell every
sub-agent to re-read its brief after compaction. They do not replace these rules for the root.
In Claude Code, the compaction hook from `resource-claude-code ensure` prompts the charter
re-read (`LONG_EFFORT_DESIGN.md` §3 "Harnesses").

In Swarm goal mode there is no per-session reviewer. Swarm's finalization
reviews the item after the run ends, so the proof clause must still name the
check and require its output in the transcript. A goal typed into a bare session
has no review behind it at all and carries the same proof clause.

### 3. The slots

Write the slots in this order. The first slot is the condition the evaluator
reads as its directive; the rest are instructions to the working agent. Make
the first slot unmistakably a directive toward a future target. A completion
predicate is not a report of current state.

| Slot | Content | Rule |
|---|---|---|
| **destination** | One future end state in the present tense, introduced as an instruction. Its size carries the ambition. | Start with “Work until this verified end state is true:” or “Bring `<target>` to this state:”. Do not open with an unmarked assertion such as “The work is complete,” which can sound like current-state context. An outcome that splits is a separate goal. No feeling-words ("clean", "production-ready"). When the operator wants large work, name an end state that is large on its face (a capability delivered end to end, a class of defects gone, every caller migrated), so small work is visibly unfinished. Prefer outcomes to counts: "N rounds" rewards cheap rounds. Before sending a destination for large work, have it checked by the ambition reviewer (`LONG_EFFORT_DESIGN.md` §2b); the author of a goal is not the one to judge whether it is big enough. For research, the destination comes from the topic's `MISSION.md` workstreams (`research-coordination` missions). |
| **stance** | The operator's posture in their own words, and what it licenses here. | Copy their words verbatim into a `Stance:` line; paraphrase strips the push, and a summary written by an agent is never a stance (`LONG_EFFORT_DESIGN.md` D13). Name what it licenses: large changes, deletions, migrations, other agents' uncommitted files. Repeat it in every sub-agent brief and the charter (§2a). |
| **proof** | The commands or artifacts whose output must appear in the transcript, and what counts as passing. | Name the check. "Validated" without a command is not a proof clause. |
| **sources** | What to read first, by name: skills, the selected plan or epoch file when applicable, and the scenario docs. | Point; do not paste. Docs first, then code. |
| **boundary** | Allowed paths and effects. Scope policy: `fixed` or `extend-with-record`. | Use the plan's `acceptance_allow` when a plan exists. |
| **dials** | Validation posture (targeted by default; name the heavy runs that are owed). Adjacent-defect posture (fix when understood and blocking; otherwise file and continue). Quality bar (no shims, no dead code, docs updated with the code). | Select a posture. The tiers themselves live in `implementation-plan-execution`. |
| **blocked** | A human-owned decision, credential, or approval with no safe alternative. | Include this boundary. Diagnose tool, owner-read, model, and infrastructure failures; pursue materially different repairs or routes while they can safely advance the outcome. Retry only when the condition or evidence changes. Record unavailable proof honestly and continue independent work; an observation failure is not a portfolio-wide block. |
| **budget** | A turn, time, or token clause, and the recovery or wrap-up action when it hits. For a sustained effort, the agent ends units, never the goal: closure before the budget needs a fresh challenger run with the fixed brief in `LONG_EFFORT_DESIGN.md` §2a. A verdict reached that way (`complete`, `blocked`, `abstained`) is final. | Claude Code has no turn cap of its own. Codex needs the wrap-up instruction to make `budget_limited` useful. Name only recovery the selected harness actually provides. If the run has no verified automatic continuation, require a checkpoint with the remaining work and exact resume route; do not assume Swarm or imply the limit is enforced. State that a verdict (`complete`, `blocked`, `abstained`) is final. |
| **handoff** | Where to checkpoint (Plan Manager log for plan-backed work, the epoch file in an orchestrated effort, otherwise the scenario's tracking doc) and the final report shape: changed, verified, remaining, unverified. | A report shape turns completion prose into checkable fields. |
| **non-goals** | What not to do: widen scope, weaken required behavior, remove uniquely protective checks merely for a greener result, or rerun unchanged validation for that result. | Cite `improvement-do-and-dont` for anti-gaming and `test` §5/§8 for justified simplification. |

Test the whole message with three questions. Is the first sentence clearly an
instruction to reach a future state rather than a claim that the state already
holds? Could a reviewer who wanted to disprove "this goal was met" do so from the
transcript alone? If not, the destination needs an explicit directive or the proof
is unnamed. Then run the **exits audit**: list every way the goal can end (complete,
closure, blocked, a count reached, budget, "nothing material"). The agent takes the
cheapest acceptable exit. Make each cheap exit expensive: no-change units do not
count, blocked names one human-owned decision and routes it while other work
continues, and closure of a sustained effort is decided by a fresh challenger run
with the fixed brief in `LONG_EFFORT_DESIGN.md` §2a, whose question the closer may
not rewrite. Count the clauses: when procedures for caution and
bookkeeping outnumber the concrete destination, the agent will optimize the
procedures.

### 4. What goes where

| Documentation and selected work-shape artifact | A skill the agent reads | The goal message |
|---|---|---|
| Target design, interfaces, acceptance criteria, the route when the route is the requirement, setpoint rows and sensors, `acceptance_allow`. | How to act under a goal: divergence tiers, the meaning of blocked, validation scope, checkpoint procedure, anti-gaming. | This run's finish line, the proof command, pointers, the posture dials, budget, report shape, non-goals. |
| Changes when the design changes. | Changes when doctrine changes. | Changes every dispatch. |

When the target documentation does not yet describe the intended design, the
first assignment is to write it there. Every later goal then points at it. A goal
that explains the design is lossy and stale on arrival.

For UI work, that document holds the effective target: selected mockups, applicable user
amendments, retained capabilities and verified agent refinements (`experiential-ui-design`
§2a). Name its version and applicable IDs in the assignment. When acceptance changes,
synchronize affected briefs and eligible native predicates through their owner and verify
receiving. Require the implemented, operated outcome against the current target; producing
a comparison ledger alone is not completion, and a verified expected difference is not a
failure. Do not require a regenerated atlas or a new human gate for routine refinements.

### 5. Shapes

Choose the cheapest shape that survives the work's real durability and authority
needs; a plan is one shape, not the default. A long delivery goal run as epochs
by a delivery team is none of these shapes: use `large-effort-orchestration`.
Replace every `<...>` field; delete a line only when
its slot does not apply, and say so in the handoff if a reviewer would expect it.

The shape of a worker assignment is independent of the parent effort or plan.
A worker inside a phased plan or adaptive mandate normally receives a bounded
task or investigation goal. It receives a plan-backed goal only when it owns a
separate plan-shaped change boundary. Do not
author a child plan merely to make a worker's handoff look formal.

Every sub-agent assignment also names its **model and lifetime**, and the lifetime
matches the outcome. Change size follows how long one agent owns one coherent
target (`LONG_EFFORT_DESIGN.md` §1): workers that live minutes never make large
changes. Use this table:

| Assignment | Role and lifetime |
|---|---|
| Well-specified retrieval, a count, one config value | Cheapest lookup role (`lookup`, Codex Luna read-only roles); minutes |
| Independent review or challenge | `reviewer` or an equivalent read-only role; one pass |
| Implementation, a migration, a hunt segment, a research direction | An implementation-capable role (`implementer`, Codex implementation roles) at the effort the judgment needs; owns the whole outcome for as long as it takes, often hours; re-task it on the next slice while its context stays useful (`followup_task` in Codex, a message to the running agent in Claude Code) |
| A different outcome | A fresh worker with its own brief |

Size each assignment around one coherent outcome, never around one step: do not split
source, tests and validation into separate workers to create more dispatches. Write
the brief from the charter, not from your recent context, and spawn with
`fork_turns=none`; a full-history fork inherits your drift, most of your context window
and your model. Carry the destination and the operator's stance verbatim. When the
worker should delegate further, say so explicitly ("you are expected to spawn
sub-agents for …"); the harness discourages spawning unless instructions ask for it.
Use the harness's real names: `implementer`/`opus` exist only in Claude Code, Codex
roles only in Codex.

**A. Plan-backed goal.** A required route, phase order or delivery dependency
needs a durable plan. Session count alone does not require a plan; sustained
research uses E. Receiving skill: `implementation-plan-execution`. A
convergence loop has no phase order and is never this shape (§6).

```text
/goal Plan <slug> is complete to its intent: every phase is done in Plan Manager with recorded evidence, or the handoff names exactly what remains and the decision it waits on.

Read first: prompt-manager skill read implementation-plan-execution harness-goal-authoring; then plan-manager exec continue <slug>. The plan holds the design; do not re-derive it here.
Proof: the phase validation commands named in the plan, output shown. Targeted checks by default; run a baseline or full suite only where a phase's completion policy names one.
Boundary: the plan's acceptance_allow. Scope policy: extend-with-record; run boundary-extend before any out-of-scope edit.
Adjacent defects: fix when you understand the cause and it blocks a phase; otherwise file with plan-manager log bug-add and continue. Do not turn this into a dependency-repair project.
Quality: production code, no shims or dead code, docs updated with the code.
Blocked means a consequential decision, credential, approval or external action the user alone owns, with no safe alternative. Infrastructure and owner-read failures are work for the agent to resolve or route around. Diagnose the failure, then choose a materially different repair, source, tool, execution path or useful pivot. Retry only when a changed condition or new evidence makes the attempt informative. Continue while a safe, authorized path can advance the outcome; when none remains, record the exact unresolved dependency and continue independent work. Never ask the user to debug infrastructure.
Budget: stop after <N> turns or <time>; checkpoint through Plan Manager first and write what changed, what was verified, what remains.
```

**B. Adaptive mandate goal.** The work is large or its architecture is not yet
clear, and the scenario has a documented target and an improve skill with
sensors. The goal points at a plan of shape `mandate`, authored with
`adaptive-mandate-authoring`. Receiving skills: `goal-loop`,
`scenario-improvement-campaign`, `<scenario>-improve`.
Through Swarm a mandate plan runs in goal mode; Swarm composes the goal message
from this shape, the item and the plan, and appends the operator note verbatim.

```text
/goal Bring every required row in <scenario>-improve in band on <scenario>.setpoint-read and pass the evidence audit. Build or repair a row's evidence producer only when it is a reusable instrument (one command, used by a gate, reused later); otherwise record the row unverified and continue. Continue while useful in-scope work remains.

Authority: <plan slug or Swarm item> grants development inside <acceptance_allow>.
Read first: goal-loop, scenario-improvement-campaign, <scenario>-improve, and the scenario docs. Write missing design into the docs before code.
Proof: show the board after each intervention; never edit a band or sensor to move a row.
Iteration: one evidence-based falsifiable intervention at a time; checkpoint through Plan Manager.
Adjacent defects in other scenarios: file them; repair at the owner only when the grant covers it.
Mandate lane: use authorized local and simulated qualification paths; do not block on excluded private data, physical devices, live keys, or paid access. Build only reusable instruments; journal every other unavailable row as unverified.
Blocked means a consequential decision, credential, approval or external action the user alone owns, with no safe alternative. Missing tools, reads and infrastructure are friction for the agent to resolve or route around. Diagnose the failure, then pursue materially different repairs, sources, tools or execution paths while each can advance the outcome safely. Retry only after a changed condition or when new evidence makes the attempt informative. When no useful safe route remains, record unavailable proof as unknown and keep unaffected rows moving.
Budget: <tokens or time>; when reached, checkpoint and summarize remaining rows.
```

**C. Bounded task goal.** The work fits one session and the route is recoverable
from the docs and code. No plan. A long-lived owner of a large outcome uses G, not C. Most coordinator sub-assignments that are not
product features, including plan authoring and docs-first authoring, take this
shape.

```text
/goal Work until this verified end state is true: <end state in present tense>, proven by <command> exiting 0 and <observable result>, both shown in the transcript.

Context: read <doc paths> first. Touch only <paths>.
Validation: <targeted command>. No full suite.
Adjacent defects: file with the report-bug skill; do not fix.
Blocked means a consequential decision, credential, approval or external action the user alone owns, with no safe alternative. For a failed command or unavailable service, diagnose the failure and pursue materially different repairs or routes while each can advance the outcome safely. Retry only after a changed condition or when new evidence makes the attempt informative. When no useful safe route remains, document the exact dependency and continue unrelated work.
Stop after 30 turns with a written summary of changed, verified, remaining.
Non-goals: <what not to touch or widen into>.
```

**D. Read-only investigation or review goal.** The commissioned deliverable is
a report or verdict. Use this shape only when the actual commission is read-only.
For sustained research or authorized construction, use E. Do not choose a read-only
shape merely because the first action is research.

```text
/goal A report at <path> answers "<question>" with a verdict per claim and file:line evidence for each, and no file outside <path> is modified.

Read first: <docs, plan, or the worker's handoff under review>.
Label every statement fact, hypothesis, or recommendation. Quote; do not paraphrase claims you are judging.
If a source is unavailable, record it as unverified and continue with accessible sources. Ask only when the missing source requires a human-owned choice or authorization that no alternative can replace.
Stop after 20 turns.
```

**E. Sustained research direction.** The route emerges from evidence. The
outcome may be a finding, architecture, experiment, prototype or authorized app.
Point at the commission's working brief and checkpoint. Read
`prompt-manager skill read adaptive-research`; use the canonical
`research-coordination/references/direction.md` for curation. An approval gate
applies only when the actual commission requires it. Give the investigator a
coherent direction and starting evidence, freedom to choose methods and follow
implications, and proof suited to the outcome. Do not impose D's read-only rule
or example turn cap. Preserve real cumulative limits and session recovery through
the selected owner; do not promise automatic continuation from prompt text.
A session limit requires a checkpoint or successor, not a smaller research goal.
For a bounded source question, a normal prompt may be cheaper than a native goal.

**F. Thin scheduler (root of a long effort; Codex).** The effort runs longer than one healthy
session (about 6 hours) or needs several parallel owners: a long hunt, a research cohort, a
multi-part migration. The root holds a native goal (or runs as an Agent Manager run with
`until`) and a ledger, and does no domain work. Its unit is one segment return or one due
review, and its turn spans the wait: it blocks on `wait_agent` at `timeout_ms: 3600000`, waits
again when a wake brings nothing to review, and ends the turn only after a return is reviewed
and recorded. A scheduler that ends its turn while owners are only working is re-injected at
once and loops every few minutes (`effort-diagnosis` M25). The charter stays small because the
state page and the ledger are separate files (`LONG_EFFORT_DESIGN.md` D15).
Each unit it spawns or re-tasks a segment owner (G) from the charter template, waits, has a
fresh ambition reviewer judge each plan once and a fresh challenger judge each showcase claim
once against the destination and the measured magnitude (`effort-diagnosis`
`scripts/effort_anatomy.py drift <owner rollout> --scan <scan>`, which exits 3 when a tripwire
fires), records the result and ends its turn. It does not review increments (D16).
Its context stays small because owners return summaries. Check the concurrency slots first:
the scheduler and each owner hold one, and the rest is shared by all workers and reviewers
(with 8 slots, four owners leave three), and on `agent
thread limit reached` do the work in-thread or wait instead of retrying (`LONG_EFFORT_DESIGN.md`
§3–§4). In Claude Code use two tiers (`LONG_EFFORT_DESIGN.md` §3 "Harnesses").

```text
/goal Work until this verified end state is true: <destination in one line, with its scale>. Stance: "<verbatim excerpt, about 150 characters>". Follow the scheduler procedure in <charter path>; re-read it at every unit start and after compaction. Do no domain work yourself. Block on wait_agent (timeout_ms 3600000) until a segment return or the hourly review; after it is recorded, end your turn.
```

Send the §2a stance message first, ending with `The goal follows this message. Reply "ready"
and wait for it.` Without that line the agent starts on the stance message and writes its own
goal, a paraphrase that drops the stance excerpt and the turn rule. Keep the filled kernel at
600 characters or fewer (pause and resume: `LONG_EFFORT_DESIGN.md` D4). The charter holds the scheduler procedure
(start or re-task one owner per free slot from the brief template with `fork_turns=none`; have
a fresh reviewer judge each plan once and each showcase claim once against the destination,
never each increment; record it in the ledger; use the §2a challenger brief before any closure), the full stance, proof, boundary, blocked, budget,
non-goals, the ledger path and the brief template.

**G. Segment owner brief.** One owner, one coherent outcome, hours of work: a study's next
epoch, ten hunt rounds, one migration. The brief is generated from the charter and names,
destination and stance first (an Agent Manager child keeps the start of a long task prompt
through compaction; how much of a native sub-agent's brief survives is unknown):
the segment's destination and expected magnitude, the stance verbatim, the charter path to
re-read at each round start and after compaction, explicit permission and expectation to
spawn its own workers, pinned exemplars of a good unit, a time box of at least 2 hours
unless the outcome finishes sooner, and the return shape (changed, verified, remaining,
unverified, outcomes, files, lines added and removed). It does not return before its end
state or time box. A segment that needs its own native goal and audit runs as an Agent
Manager child run. Pass `--until-file <kernel> --charter <charter> --parent-run-id <own run>`,
then wait with `run wait --children`. The parent must itself be an Agent Manager-launched run.
If Agent Manager refuses or is unreachable (exit 6), give the same brief to a native
sub-agent and continue. `LONG_EFFORT_DESIGN.md` §3 "Goal-bearing children" has the commands
and the fallback rule.

### 6. Convergence goals: done is a disproof, not a checklist

Some goals ask for genuine excellence, not a finished checklist: a maturity
mandate (B), a quality pass over existing work, or "bring this surface to
production grade". For these a passing suite is the most dangerous moment,
because it is where agents stop. Write the goal so that green is the trigger to
start looking, not the finish line. Write the destination as a disproof
condition, not as a list of features that exist. Orchestrated efforts follow
`large-effort-orchestration` instead (epoch gate, step-back, feedback).

- **The stop condition is an empty adversary, not a green board.** State that the
  agent is done only when a fresh, hostile review pass finds nothing material,
  twice in a row. The agent runs a new pass each time it believes the work is
  finished, records every finding, resolves it, and repeats until two consecutive
  fresh passes are empty. A green suite is a precondition of the pass, never the
  completion.
- **The pass judges more than features.** Direct it at correctness, edge cases,
  code maturity, cleanliness, maintainability, and, for a UI, visual polish
  against the named mockups. "It works" is not "it is done".
- **Require self-critique without assigning self-acceptance.** The implementer
  verifies its whole boundary; an independent reviewer, when one exists, keeps
  the acceptance decision.
- **Relentless points at the spec, simplicity and less debt, not accretion.**
  Less duplication, no shims, no dead code, simpler control flow. Over-engineering
  and any new debt are material findings, not progress.
- **Track the work in files, not a plan.** A convergence goal loops until dry, so
  it has no phase order and never takes a phased plan (A). Hand it off as a
  bounded task (C) that tracks findings and open issues in the scenario's durable
  files, re-read each pass. Use a mandate (B) only when the work needs an improve
  skill's sensors and setpoints.
- **Persistence is one fixed clause, plus the step-back rule and §2a.** Use: "Keep going
  through friction; when progress stalls, step back — write why, then park,
  switch or shrink — instead of stopping or adding more of the same." Do not
  stack further persistence controls; each one lengthens every resume and pushes
  agents toward accretion. Use the §2a stance message and kernel and point at the
  charter for everything else.
- **Stance is not a persistence control; carry it verbatim** (the §3 stance slot).
  Do not compress the operator's words into neutral procedure: paraphrase strips
  the push, and the receiving agent falls back to its cautious default. A stance
  without a destination of matching size is tone; pair it with one.
  `ecosystem-hunt` `references/goal.md` is an example.
- **State the real stakes.** Name why the work matters and what stays blocked
  while it is broken. Keep them true; invented stakes read as hollow.
- **Operator feedback is a durable, gating input.** The receiving agent appends
  each item verbatim to the scenario's feedback ledger before acting on it,
  re-reads the ledger each pass, and is not done while an item is open. Say so in
  the goal; a remark typed into the session does not survive compaction.

The gaming definitions live in `improvement-do-and-dont`; cite it, do not restate it.

Destination fragment for a bounded convergence task (C) or a mandate (B); point at
the mockups and the durable doc rather than describing them:

```text
Before handing off <target>, run two consecutive fresh, hostile review passes that find nothing material. Treat a green suite as the signal to start a new pass. Each pass judges correctness, edge cases, code cleanliness and maintainability, and visual polish against <mockups/spec>, not feature presence. Fix findings within the assigned boundary and record findings and resolutions in <durable doc>. Follow <acceptance protocol>; your self-review does not replace independent acceptance. Do not create a plan; <durable doc> is the tracking system. Leave <target> cleaner and with less technical debt than you found it; new debt is a finding. Record operator feedback in the assigned durable feedback surface and resolve every assigned item before handoff. Stakes: <why this matters and what stays blocked until it is fixed>.
```

### 7. Acting under a goal

A receiving agent that has no other skill for the work follows these rules.

1. Read the sources named in the goal before the code.
2. Show the proof. Run the named checks and leave their output in the transcript.
3. Say "blocked" only for a decision, credential, or approval you lack, and name it.
4. Checkpoint before the budget clause fires. Write changed, verified, remaining, unverified.
5. Do not widen scope, loosen a test, or rerun unchanged validation to produce a greener result.
6. When the harness offers `ProposeGoal` and the operator's words already state a verifiable outcome, propose the goal instead of asking; never propose a goal that widens scope.

### 8. Output expectations

You may write goal text into a chat, a `--until` flag, an Agent Manager workflow
run node, or a coordinator handoff. You must not restate a plan's design in a goal, add
a completion clause that a harness can satisfy by asserting it, or change a
receiving skill's doctrine from inside a goal.

### Anti-patterns

| Anti-pattern | Consequence | Correction |
|---|---|---|
| A feeling-word destination ("clean", "solid") | The goal never clears or clears falsely | Name a state a command can show |
| "Validated" with no command | The evaluator guesses; the agent stops early | Name the check and require its output |
| The same tolerance preamble retyped every dispatch | Drift between copies; long goals | Move the rule into the receiving skill; the goal selects a posture |
| A sub-agent with no model named | It inherits the parent's most expensive model | Name the agent definition or role (§5) |
| Many minute-long workers for one outcome | No worker can make a large change; output falls while dispatches multiply | One owner per outcome for its whole length (§5 table) |
| Full-history forks for workers | Children inherit drift and most of the context window; they compact early | Fresh brief from the charter, `fork_turns=none` |
| A long goal message with no kernel | The objective is omitted after compaction; the agent copies its recent units | Stance message, kernel ≤600 characters and charter; one unit per turn (§2a) |
| Ambition only as adjectives | The agent executes the procedures and treats "be bold" as tone | A destination large on its face, plus the stance slot |
| A count destination ("N rounds") | Cheap, empty rounds satisfy it | An outcome destination; no-change units do not count |
| A plan for every sub-assignment | 100k–200k tokens of authoring per worker | Choose the cheapest shape (§5) |
| A goal that explains the design | Lossy, stale on arrival | Write the design into the docs; point the goal at it |
| Completion by self-report | Harness "met" with no evidence | Proof in transcript; independent review for material outcomes |

### Troubleshooting & Edge Cases

- **The evaluator keeps returning Not yet met after the work is done.** The proof
  never entered the transcript. Rerun the named check so its output is visible.
- **The agent stops after a few turns with no goal clear.** Several turns passed
  without tool use. Restate the next concrete action in the goal or the handoff.
- **A long effort starts small or shrinks over hours.** Diagnose it with
  `effort-diagnosis` before rewriting the goal; the cause is usually structure (worker
  lifetime, forks, exits), not wording.
- **Children never spawn their own workers.** The brief did not ask them to, or the
  session still has the default Codex hint and slot limit (`LONG_EFFORT_DESIGN.md` §4).
- **Codex reports `budget_limited`.** This is not completion. Read the wrap-up
  summary, then start a new session from the handoff.
- **The goal repeats design already held in docs.** Move the durable design to
  the docs or receiving skill and point at it. Do not use the old 2,048-character
  Agent Manager limit as a writing rule; its current transport ceiling is 1 MiB.
- **Native support is absent.** Agent Manager still appends `until` to the
  prompt; the receiving skill's stop rules carry the loop. Do not invent a
  `--goal` flag; none exists.
