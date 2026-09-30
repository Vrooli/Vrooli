---
name: harness-goal-authoring
description: Write the message a coding agent runs under in harness goal mode (Claude Code and Codex `/goal`, Agent Manager `until`), or the assignment a coordinator hands a sub-agent. Nine slots, four handoff shapes, and the split between what the docs and plan hold and what the goal message holds. Not for Swarm goal records; use swarm-manager-work-authoring for those.
license: CC-BY-4.0
metadata:
  kind: skill
  schemaVersion: 1
  modes: [practice]
  tags: [goal, harness, until, delegation, sub-agent, orchestration, prompt]
  icon: target
  status: active
  revision: 13
  createdAt: "2026-09-11T00:00:00Z"
  updatedAt: "2026-09-30T12:00:00Z"
  requires:
    scenarios: [prompt-manager]
    commands: [prompt-manager skill read]
  origin:
    kind: authored
---

## Practice focus: Harness Goal Authoring

Write a goal message that a fresh agent can finish from, that a transcript-only
evaluator can judge, and that stays under 2,048 characters (Agent Manager's
`until` cap) because the design it serves lives in durable documentation and
the selected work-shape artifact (a plan only when the shape requires one), not
in the message.

Required reading:
- `docs/TESTING.md` — validation scope. A goal selects a posture; it does not
  redefine the scope rules.
- `implementation-plan-execution` §"Divergence tiers" and §"Blocked" when the
  selected shape is plan-backed — the receiving agent's rules for adjacent
  defects and for the word "blocked". A goal selects among them; it does not
  rewrite them.

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
| Who decides "met" | A separate small model reads the transcript only. It runs no commands and opens no files. Verdicts: Not yet met, Met, Impossible. | The working model calls `update_goal(complete)`. Each continuation turn injects an audit: enumerate deliverables, reject proxy signals, treat uncertainty as not achieved. |
| What this requires of the author | Proof must appear in the transcript. Name the command and require its output to be shown. | Proof must be enumerable. State deliverables the audit can tick one by one. |
| Loop guards | Halts after several turns without tool use. The Stop hook is overridden after repeated consecutive blocks; the hooks guide states the cap. No turn cap exists; add one in the text. | A continuation turn with zero tool calls suppresses the next one. On budget exhaustion the goal becomes `budget_limited` and the model is told to wrap up. |
| Size | 4,000 characters. `ProposeGoal` proposals: 500. | Agent Manager caps `until` at 2,048 characters. |

Agent Manager delivers `until` as prompt text on every runner and also types
`/goal <until>` into the session when the runner declares native objective
support for the selected sandbox mode. Write one text that works both ways.

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
| **destination** | One future end state in the present tense, introduced as an instruction. | Start with “Work until this verified end state is true:” or “Bring `<target>` to this state:”. Do not open with an unmarked assertion such as “The work is complete,” which can sound like current-state context. An outcome that splits is a separate goal. No feeling-words ("clean", "production-ready"). |
| **proof** | The commands or artifacts whose output must appear in the transcript, and what counts as passing. | Name the check. "Validated" without a command is not a proof clause. |
| **sources** | What to read first, by name: skills, the selected plan or epoch file when applicable, and the scenario docs. | Point; do not paste. Docs first, then code. |
| **boundary** | Allowed paths and effects. Scope policy: `fixed` or `extend-with-record`. | Use the plan's `acceptance_allow` when a plan exists. |
| **dials** | Validation posture (targeted by default; name the heavy runs that are owed). Adjacent-defect posture (fix when understood and blocking; otherwise file and continue). Quality bar (no shims, no dead code, docs updated with the code). | Select a posture. The tiers themselves live in `implementation-plan-execution`. |
| **blocked** | "Blocked means a decision, credential, or approval you lack. Name it. Friction you can diagnose is not blocked." | Include this sentence verbatim or by skill reference. Agents define "blocked" for themselves when the goal does not. |
| **budget** | A turn, time, or token clause, and the wrap-up action when it hits. | Claude Code has no turn cap of its own. Codex needs the wrap-up instruction to make `budget_limited` useful. State that an involuntary interruption (usage window, timeout, crash, session lost) is resumed by Swarm under `continuation: until-allowance`, and that a verdict (`complete`, `blocked`, `abstained`) is final. |
| **handoff** | Where to checkpoint (Plan Manager log for plan-backed work, the epoch file in an orchestrated effort, otherwise the scenario's tracking doc) and the final report shape: changed, verified, remaining, unverified. | A report shape turns completion prose into checkable fields. |
| **non-goals** | What not to do: widen scope, loosen or delete tests, rerun unchanged validation for a greener result. | Cite `improvement-do-and-dont` for the anti-gaming rules. |

Test the whole message with two questions: is the first sentence clearly an
instruction to reach a future state rather than a claim that the state already
holds, and could a reviewer who wanted to
disprove "this goal was met" do so from the transcript alone? If not, the
destination needs an explicit directive or the proof is unnamed.

### 4. What goes where

| Documentation and selected work-shape artifact | A skill the agent reads | The goal message |
|---|---|---|
| Target design, interfaces, acceptance criteria, the route when the route is the requirement, setpoint rows and sensors, `acceptance_allow`. | How to act under a goal: divergence tiers, the meaning of blocked, validation scope, checkpoint procedure, anti-gaming. | This run's finish line, the proof command, pointers, the posture dials, budget, report shape, non-goals. |
| Changes when the design changes. | Changes when doctrine changes. | Changes every dispatch. |

When the target documentation does not yet describe the intended design, the
first assignment is to write it there. Every later goal then points at it. A goal
that explains the design is lossy and stale on arrival.

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

**A. Plan-backed goal.** The route or phase order matters, or several sessions
will touch the work. Receiving skill: `implementation-plan-execution`. A
convergence loop has no phase order and is never this shape (§6).

```text
/goal Plan <slug> is complete to its intent: every phase is done in Plan Manager with recorded evidence, or the handoff names exactly what remains and the decision it waits on.

Read first: prompt-manager skill read implementation-plan-execution harness-goal-authoring; then plan-manager exec continue <slug>. The plan holds the design; do not re-derive it here.
Proof: the phase validation commands named in the plan, output shown. Targeted checks by default; run a baseline or full suite only where a phase's completion policy names one.
Boundary: the plan's acceptance_allow. Scope policy: extend-with-record; run boundary-extend before any out-of-scope edit.
Adjacent defects: fix when you understand the cause and it blocks a phase; otherwise file with plan-manager log bug-add and continue. Do not turn this into a dependency-repair project.
Quality: production code, no shims or dead code, docs updated with the code.
Blocked means a decision, credential, or approval you lack. Name it in the handoff. Friction you can diagnose is not blocked.
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
Blocked means only missing decision, credential, approval, or external access. In-boundary gaps: fix, validate, continue.
Budget: <tokens or time>; when reached, checkpoint and summarize remaining rows.
```

**C. Bounded task goal.** The work fits one session and the route is recoverable
from the docs and code. No plan. Most coordinator sub-assignments that are not
product features, including plan authoring and docs-first authoring, take this
shape.

```text
/goal Work until this verified end state is true: <end state in present tense>, proven by <command> exiting 0 and <observable result>, both shown in the transcript.

Context: read <doc paths> first. Touch only <paths>.
Validation: <targeted command>. No full suite.
Adjacent defects: file with the report-bug skill; do not fix.
Blocked means a missing decision or credential; name it and stop.
Stop after 30 turns with a written summary of changed, verified, remaining.
Non-goals: <what not to touch or widen into>.
```

**D. Investigation or review goal.** The deliverable is a report or a verdict,
not a change. Read-only. Never give this shape a plan.

```text
/goal A report at <path> answers "<question>" with a verdict per claim and file:line evidence for each, and no file outside <path> is modified.

Read first: <docs, plan, or the worker's handoff under review>.
Label every statement fact, hypothesis, or recommendation. Quote; do not paraphrase claims you are judging.
Blocked means a source you cannot access; name it and report what you could verify.
Stop after 20 turns.
```

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
- **Persistence is one fixed clause, plus the step-back rule.** Use: "Keep going
  through friction; when progress stalls, step back — write why, then park,
  switch or shrink — instead of stopping or adding more of the same." Do not
  stack further persistence controls; each one lengthens every resume and pushes
  agents toward accretion. Keep a continuous goal to about 800 characters and
  point at docs for everything else.
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
| A plan for every sub-assignment | 100k–200k tokens of authoring per worker | Choose the cheapest shape (§5) |
| A goal that explains the design | Lossy, stale on arrival | Write the design into the docs; point the goal at it |
| Completion by self-report | Harness "met" with no evidence | Proof in transcript; independent review for material outcomes |

### Troubleshooting & Edge Cases

- **The evaluator keeps returning Not yet met after the work is done.** The proof
  never entered the transcript. Rerun the named check so its output is visible.
- **The agent stops after a few turns with no goal clear.** Several turns passed
  without tool use. Restate the next concrete action in the goal or the handoff.
- **Codex reports `budget_limited`.** This is not completion. Read the wrap-up
  summary, then start a new session from the handoff.
- **The goal is over 2,048 characters.** It is carrying design or doctrine. Move
  that content to the docs or the receiving skill and point at it.
- **Native support is absent.** Agent Manager still appends `until` to the
  prompt; the receiving skill's stop rules carry the loop. Do not invent a
  `--goal` flag; none exists.
