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
  revision: 1
  createdAt: "2026-09-11T00:00:00Z"
  updatedAt: "2026-09-11T00:00:00Z"
  requires:
    scenarios: [prompt-manager]
    commands: [prompt-manager skill read]
  origin:
    kind: authored
---

## Practice focus: Harness Goal Authoring

Write a goal message that a fresh agent can finish from, that a transcript-only
evaluator can judge, and that stays under 1,500 characters because the design it
serves lives in the documentation and the plan, not in the message.

Required reading:
- `docs/agent-system/SWARM_MANAGER_WORK.md` §"Work shapes" — which shape of work
  to hand off, and when a plan is warranted. This skill does not restate it.
- `docs/TESTING.md` — validation scope. A goal selects a posture; it does not
  redefine the scope rules.
- `implementation-plan-execution` §"Divergence tiers" and §"Blocked" — the
  receiving agent's rules for adjacent defects and for the word "blocked". A goal
  selects among them; it does not rewrite them.

### 1. Scope

In scope: the text of a harness goal (`/goal <condition>` in Claude Code or
Codex, `--until` on `agent-manager runs create`, the `until` field of an Agent
Manager workflow run node) and the assignment message a coordinator gives a
sub-agent, with or without native goal support.

Out of scope: Swarm goal records (`swarm-manager goals`), milestones, and backlog
item text. Read `swarm-manager-work-authoring` for those. A Swarm goal states a
change in the world; a harness goal is an execution mechanism. Do not mix them.

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

Inside a Swarm `goal-session` execution, an independent review child inspects
every session result, so the `until` text can be lean about proof. A goal typed
into a bare session has no review behind it and must carry its own proof clause.

### 3. The slots

Write the slots in this order. The first slot is the condition the evaluator
reads as its directive; the rest are instructions to the working agent.

| Slot | Content | Rule |
|---|---|---|
| **destination** | One end state in the present tense. | An outcome that splits is a separate goal. No feeling-words ("clean", "production-ready"). |
| **proof** | The commands or artifacts whose output must appear in the transcript, and what counts as passing. | Name the check. "Validated" without a command is not a proof clause. |
| **sources** | What to read first, by name: skills, the plan, the scenario docs. | Point; do not paste. Docs first, then code. |
| **boundary** | Allowed paths and effects. Scope policy: `fixed` or `extend-with-record`. | Use the plan's `acceptance_allow` when a plan exists. |
| **dials** | Validation posture (targeted by default; name the heavy runs that are owed). Adjacent-defect posture (fix when understood and blocking; otherwise file and continue). Quality bar (no shims, no dead code, docs updated with the code). | Select a posture. The tiers themselves live in `implementation-plan-execution`. |
| **blocked** | "Blocked means a decision, credential, or approval you lack. Name it. Friction you can diagnose is not blocked." | Include this sentence verbatim or by skill reference. Agents define "blocked" for themselves when the goal does not. |
| **budget** | A turn, time, or token clause, and the wrap-up action when it hits. | Claude Code has no turn cap of its own. Codex needs the wrap-up instruction to make `budget_limited` useful. |
| **handoff** | Where to checkpoint (Plan Manager log, a progress file) and the final report shape: changed, verified, remaining, unverified. | A report shape turns completion prose into checkable fields. |
| **non-goals** | What not to do: widen scope, loosen or delete tests, rerun unchanged validation for a greener result. | Cite `improvement-do-and-dont` for the anti-gaming rules. |

Test the whole message with one question: could a reviewer who wanted to
disprove "this goal was met" do so from the transcript alone? If not, the
destination is a feeling or the proof is unnamed.

### 4. What goes where

| Documentation and plan | A skill the agent reads | The goal message |
|---|---|---|
| Target design, interfaces, acceptance criteria, the route when the route is the requirement, setpoint rows and sensors, `acceptance_allow`. | How to act under a goal: divergence tiers, the meaning of blocked, validation scope, checkpoint procedure, anti-gaming. | This run's finish line, the proof command, pointers, the posture dials, budget, report shape, non-goals. |
| Changes when the design changes. | Changes when doctrine changes. | Changes every dispatch. |

When the target documentation does not yet describe the intended design, the
first assignment is to write it there. Every later goal then points at it. A goal
that explains the design is lossy and stale on arrival.

### 5. Shapes

Choose the shape with `docs/agent-system/SWARM_MANAGER_WORK.md` §"Work shapes".
Templates follow. Replace every `<...>` field; delete a line only when its slot
does not apply, and say so in the handoff if a reviewer would expect it.

**A. Plan-backed goal.** The route or phase order matters, or several sessions
will touch the work. Receiving skill: `implementation-plan-execution`.

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
sensors. Receiving skills: `goal-loop`, `scenario-improvement-campaign`,
`<scenario>-improve`. Through Swarm this is strategy `adaptive-improvement` or
`goal-session`; the workflow supplies the `until` text.

```text
/goal Every readable setpoint row in <scenario>-improve is in band on two consecutive reads of run <scenario>.setpoint-read, or the handoff names each out-of-band row with its blocker.

Authority: <plan slug or Swarm item> grants development inside <acceptance_allow>.
Read first: goal-loop, scenario-improvement-campaign, <scenario>-improve, and the scenario docs (START-HERE, ARCHITECTURE, PROBLEMS). The docs are the target; where the intended design is missing from them, write it there before the code.
Proof: paste the setpoint board after each intervention. Never move a row by editing its band or sensor.
Iteration: one falsifiable intervention at a time, chosen from evidence, checkpointed through Plan Manager before the next.
Adjacent defects in other scenarios: file them; repair at the owner only when the grant covers it.
Blocked means a decision, credential, or approval you lack. A row reading unavailable is journaled, not estimated.
Budget: <tokens or time>; when reached, checkpoint and summarize remaining rows.
```

**C. Bounded task goal.** The work fits one session and the route is recoverable
from the docs and code. No plan. Most coordinator sub-assignments that are not
product features, including plan authoring and docs-first authoring, take this
shape.

```text
/goal <end state, present tense>, proven by <command> exiting 0 and <observable result>, both shown in the transcript.

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

### 6. Acting under a goal

A receiving agent that has no other skill for the work follows these rules.

1. Read the sources named in the goal before the code.
2. Show the proof. Run the named checks and leave their output in the transcript.
3. Say "blocked" only for a decision, credential, or approval you lack, and name it.
4. Checkpoint before the budget clause fires. Write changed, verified, remaining, unverified.
5. Do not widen scope, loosen a test, or rerun unchanged validation to produce a greener result.
6. When the harness offers `ProposeGoal` and the operator's words already state a verifiable outcome, propose the goal instead of asking; never propose a goal that widens scope.

### 7. Output expectations

You may write goal text into a chat, a `--until` flag, an Agent Manager workflow
run node, or a coordinator handoff. You may add a Shape A example to an effort
workspace's leader mandate. You must not restate a plan's design in a goal, add
a completion clause that a harness can satisfy by asserting it, or change a
receiving skill's doctrine from inside a goal.

### Anti-patterns

| Anti-pattern | Consequence | Correction |
|---|---|---|
| A feeling-word destination ("clean", "solid") | The goal never clears or clears falsely | Name a state a command can show |
| "Validated" with no command | The evaluator guesses; the agent stops early | Name the check and require its output |
| The same tolerance preamble retyped every dispatch | Drift between copies; long goals | Move the rule into the receiving skill; the goal selects a posture |
| A plan for every sub-assignment | 100k–200k tokens of authoring per worker | Choose the shape with the Work shapes rule |
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
