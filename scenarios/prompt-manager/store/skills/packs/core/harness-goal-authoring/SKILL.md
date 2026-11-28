---
name: harness-goal-authoring
description: Write the message a coding agent runs under in harness goal mode (Claude Code and Codex `/goal`, Agent Manager `until`), or the assignment a coordinator hands a sub-agent. Ten slots, handoff shapes, how a goal survives compaction, and the split between what docs and plans hold and what the goal holds. Not for Swarm goal records; use swarm-manager-work-authoring for those.
license: CC-BY-4.0
metadata:
  kind: skill
  schemaVersion: 1
  modes: [practice]
  tags: [goal, harness, until, delegation, sub-agent, orchestration, prompt]
  icon: target
  status: active
  revision: 31
  createdAt: "2026-09-11T00:00:00Z"
  updatedAt: "2026-11-23T00:00:00Z"
  requires:
    scenarios: [prompt-manager]
    commands: [prompt-manager skill read]
  origin:
    kind: authored
---

## Practice focus: Harness Goal Authoring

Write a goal message that a fresh agent can finish from and a transcript-only evaluator can judge. Design lives in durable docs and the work-shape artifact, never in the message (R4, R7). The ambition and stance rules are RULES.md R3; the evidence and design rules for efforts that run for hours are `docs/agent-system/LONG_EFFORT_DESIGN.md` (D1-D17, §4a measured limits), which this skill applies without restating. Validation scope follows `docs/TESTING.md`. For a plan-backed shape the receiving rules are `implementation-plan-execution` §"Divergence tiers" and §"Blocked".

In scope: `/goal <condition>` (Claude Code, Codex), `--until` on `agent-manager run create`, the `until` of a workflow run node, and a coordinator's assignment to a sub-agent. Out of scope: Swarm goal records, milestones and backlog text (`swarm-manager-work-authoring`); a Swarm goal states a change in the world, a harness goal is an execution mechanism. An ambitious multi-hour outcome is a research topic, not a goal message (RULES.md "Which construct to use"). Relative paths mean repository root; the receiver verifies paths before editing. If the skill registry is down, read the canonical `SKILL.md`, say which source you read, and continue.

### 1. How the harness judges the message

| | Claude Code `/goal` | Codex `/goal` |
|---|---|---|
| Who decides met | A small model reads the transcript only; it opens no files and runs nothing | The working model calls `update_goal(complete)`; each goal turn injects the objective and an audit (enumerate deliverables, reject proxies, uncertainty is not achieved) |
| Author must | show proof in the transcript: name the command and require its output | state enumerable deliverables the audit can tick |
| After compaction | the condition stays with the evaluator | survives only when short; longer is omitted until the next goal turn (§2) |
| Guards | halts after turns with no tool use; no turn cap, so add one | a zero-tool turn suppresses the next; budget exhaustion becomes `budget_limited` |
| Size | 4,000 characters (`ProposeGoal`: 500) | kernel of 600 or fewer for anything past one compaction; Agent Manager `until` up to 1 MiB, `--until-file` for large values |

Agent Manager delivery: with no native objective support, or an unacknowledged `/goal` submission, the task prompt carries the full `until`; with native support and `until` within the verbatim limit (600 Codex, 4,000 Claude Code) `/goal` carries it; beyond it, `/goal` carries a short delegation and the prompt carries the full `until`. An uncertain or canceled submission is not a prompt fallback. Write one condition that is complete in either route.

### 2. Persistence over hours

After an hour the agent no longer sees most of what you wrote; a long objective, the stance and early skill reads lose force first, and the agent starts copying its own recent units.

| Rule | How |
|---|---|
| Stance message, kernel, charter | Send a stance message first: an ordinary user message of at most 3,500 characters, most important text first, holding the operator's stance verbatim and in full, the destination with its scale and the charter re-read instruction. Then set the kernel as the native goal, at most 600 characters: the destination in one line with its scale (a transcript-only evaluator opens no files), about 150 characters of the stance verbatim, the charter path and "one unit per turn". The charter file holds everything else. Check lengths with `wc -m`. In an Agent Manager run the task prompt is the stance message: stance and destination in its first 1,400 characters. |
| Steer through the charter | Every later message pushes the stance message toward eviction, so steering goes in the charter or feedback ledger; a session that can message the agent resends the stance after six newer messages |
| One unit per turn | Under a native goal, end the turn after each unit's record so the objective and audit are re-injected. Without a native goal, ending the turn ends the effort; omit the rule |
| Re-read | First action of every unit and after any compaction: the charter and the method skill's stance section |
| Stance everywhere | The operator's words, in full, in the stance message, the charter and every sub-agent brief; a child has no user messages |
| Curated precedent | Point at pinned exemplars of a good unit, not "the latest records" |

```text
Stance for <effort>, in the operator's words: "<full stance, verbatim>"
Destination: <end state in one line, with its scale>.
Charter: <charter path>. Re-read it at every unit start and after every compaction; it holds the method, proof, boundary and budget, and it wins over your recent units as the template.
```

In a Codex root session end the message with `The goal follows this message. Reply "ready" and wait for it.` and then set the goal; in an Agent Manager run omit that line. Swarm goal mode has no per-session reviewer, so the proof clause must still require the check's output in the transcript.

### 3. The slots

In this order. The first is the directive the evaluator reads; the rest instruct the worker.

| Slot | Content and rule |
|---|---|
| destination | One future end state in the present tense, introduced as an instruction ("Work until this verified end state is true:"), never an unmarked claim of current state. Its size carries the ambition: for large work name an end state large on its face (a capability end to end, a class of defects gone, every caller migrated) so small work is visibly unfinished. No feeling-words; prefer outcomes to counts ("N rounds" rewards cheap rounds); an outcome that splits is separate goals. Have the ambition reviewer check a large destination (`LONG_EFFORT_DESIGN.md` §2b); its author does not judge size. |
| stance | The operator's words verbatim in a `Stance:` line, and what they license (large changes, deletions, migrations, other agents' files). A paraphrase strips the push; an agent-written summary is never a stance (D13). |
| proof | Commands or artifacts whose output must appear in the transcript, and what passes. "Validated" with no command is not proof. |
| sources | What to read first, by name; point, do not paste. Docs before code. |
| boundary | Allowed paths and effects; scope policy `fixed` or `extend-with-record`; a plan's `acceptance_allow` when there is a plan. |
| dials | Validation posture (targeted by default, name any heavy run owed), adjacent-defect posture (fix when understood and blocking, else file and continue), quality bar (no shims, no dead code, docs with the code). The tiers live in `implementation-plan-execution`. |
| blocked | A human-owned decision, credential or approval with no safe alternative. Tool, owner-read, model and infrastructure failures are the agent's to diagnose and route around with materially different repairs; retry only when something changed; record unavailable proof as unknown and continue independent work. |
| budget | A turn, time or token clause and the wrap-up action. A sustained effort ends units, never the goal; closure before the budget needs a fresh challenger run with the brief in `LONG_EFFORT_DESIGN.md` §2a, and that verdict (`complete`, `blocked`, `abstained`) is final. Name only recovery the harness really provides; otherwise require a checkpoint with the remaining work and exact resume route. |
| handoff | Where to checkpoint and the report shape: changed, verified, remaining, unverified. |
| non-goals | Do not widen scope, weaken required behavior, remove protective checks, or rerun unchanged validation for a greener result. Cite `improvement-do-and-dont` and `test` §5 and §8. |

Test the message: Is the first sentence an instruction to reach a future state? Could a reviewer who wanted to disprove "met" do so from the transcript alone? Then run the **exits audit**: list every way the goal can end (complete, closure, blocked, a count, budget, "nothing material"); the agent takes the cheapest exit, so make each expensive (no-change units do not count, blocked names one human decision, closure of a sustained effort comes from a challenger). When cautions and bookkeeping outnumber the destination, the agent optimizes the procedures.

### 4. What goes where

| Docs and the work-shape artifact | A skill the agent reads | The goal message |
|---|---|---|
| target design, interfaces, acceptance, setpoints, `acceptance_allow`; changes when the design changes | how to act under a goal: divergence tiers, blocked, validation scope, checkpoints, anti-gaming; changes when doctrine changes | this run's finish line, proof, pointers, dials, budget, report shape, non-goals; changes every dispatch |

If the docs do not yet describe the intended design, the first assignment is to write it there. A goal that explains the design is lossy and stale on arrival. For UI work, the doc holds the effective target (mockups, amendments, verified refinements; `experiential-ui-design` §2a); name its version and IDs, and require the implemented, operated outcome, not a comparison ledger.

### 5. Shapes

Pick the cheapest shape that survives the work's durability and authority needs; a plan is one shape, not the default. A worker's shape is independent of its parent's: a worker inside a plan normally gets a bounded task, and a plan-backed goal only when it owns a separate plan-shaped boundary. Each assignment names a **model and lifetime** matched to the outcome, because change size follows how long one agent owns one coherent target (`LONG_EFFORT_DESIGN.md` §1): minutes-long workers never make large changes.

**Choose the artifact home before dispatch.** For ambitious, multi-hour, or cross-scenario work, use a research topic as the authoritative home. Put each run's brief, checkpoint, receipts, and evidence under `~/.vrooli/research/topics/<topic>/runs/<run-id>/`, using the actual runtime home declared by `.vrooli/repo-contract.json`. Include that run path in the charter and every worker brief. Do not put cross-scenario run evidence under `.vrooli/work` or a scenario tree; `.vrooli/work/README.md` and `docs/internal/research-experimentation/STORAGE.md` define discovery and storage. For bounded scenario-owned work, keep durable product evidence with its owner scenario under `docs/evidence/`. Keep reusable test inputs beside their tests in `testdata/` or the owning test fixture directory.

| Assignment | Role and lifetime |
|---|---|
| Specified retrieval, a count, one config value | cheapest lookup role; minutes |
| Independent review or challenge | read-only `reviewer`; one pass |
| Implementation, migration, hunt segment, research direction | implementation-capable role at the effort the judgment needs; owns the whole outcome for hours; re-task it on the next slice while its context stays useful |
| A different outcome | a fresh worker with its own brief |

Size each assignment around one outcome, not one step. Write briefs from the charter, spawn with `fork_turns=none` (a full-history fork inherits your drift and your model), carry the destination and stance verbatim, say explicitly when the worker should spawn its own workers, and use real names (`implementer`/`opus` exist only in Claude Code, Codex roles only in Codex).

| Shape | Use when | Receiving skill |
|---|---|---|
| A. Plan-backed | a required route, phase order or delivery dependency needs a durable plan (not a convergence loop, not sustained research) | `implementation-plan-execution` |
| B. Adaptive mandate | large work, unclear architecture, a documented target and an improve skill with sensors; a `mandate` plan from `adaptive-mandate-authoring` | `goal-loop`, `scenario-improvement-campaign`, `<scenario>-improve` |
| C. Bounded task | one session, route recoverable from docs and code, no plan; most coordinator sub-assignments | the skill the task names |
| D. Read-only review | the commission really is a report or verdict | none |
| E. Sustained research direction | the route emerges from evidence; point at the commission's brief and checkpoint, give a coherent direction and freedom of method, no read-only rule or turn cap; a session limit needs a checkpoint or successor, not a smaller goal | `adaptive-research`, `research-coordination` |
| F. Thin scheduler (root of an effort longer than about 6 hours, or several parallel owners) | holds a native goal and a ledger and does no domain work; each unit starts or re-tasks a segment owner, blocks on `wait_agent` at `timeout_ms: 3600000`, has a fresh reviewer judge each plan and each showcase claim once against the destination, records, and ends the turn only after a return is reviewed (ending early loops every few minutes, `effort-diagnosis` M25). Check concurrency slots first. | `LONG_EFFORT_DESIGN.md` §3-§4 |
| G. Segment owner | one owner, one coherent outcome, hours: destination and magnitude, stance verbatim, charter path to re-read, permission to spawn workers, pinned exemplars, a time box of at least 2 hours, return shape; does not return before its end state. A segment needing its own goal runs as an Agent Manager child: `--until-file <kernel> --charter <charter> --parent-run-id <own run>`, then `run wait --children`; on exit 6 give the brief to a native sub-agent. | `LONG_EFFORT_DESIGN.md` §3 |

Plan-backed template (A); B, C and D replace the first line and proof only:

```text
/goal Plan <slug> is complete to its intent: every phase is done in Plan Manager with recorded evidence, or the handoff names exactly what remains and the decision it waits on.

Read first: prompt-manager skill read implementation-plan-execution harness-goal-authoring; then plan-manager exec continue <slug>. The plan holds the design.
Proof: the phase validation commands named in the plan, output shown. Targeted checks by default.
Boundary: the plan's acceptance_allow. Scope policy: extend-with-record.
Adjacent defects: fix when understood and blocking; otherwise file with plan-manager log bug-add and continue.
Blocked: a decision, credential, approval or external action only the user owns, with no safe alternative; everything else is yours to diagnose and route around.
Budget: stop after <N> turns or <time>; checkpoint through Plan Manager first: changed, verified, remaining.
```

Bounded task (C) first line: `/goal Work until this verified end state is true: <end state>, proven by <command> exiting 0 and <observable result>, both shown in the transcript.` then Context, Validation, Adjacent defects, Stop after N turns with changed, verified, remaining, and Non-goals. Read-only (D): `/goal A report at <path> answers "<question>" with a verdict per claim and file:line evidence for each, and no file outside <path> is modified.` Mandate (B): `/goal Bring every required row in <scenario>-improve in band on <scenario>.setpoint-read and pass the evidence audit`, build a producer only when it is a reusable instrument, never edit a band or sensor to move a row. Scheduler (F) kernel: `/goal Work until this verified end state is true: <destination, with scale>. Stance: "<about 150 characters verbatim>". Follow the scheduler procedure in <charter path>; re-read it at every unit start and after compaction. Do no domain work yourself. Block on wait_agent (timeout_ms 3600000) until a segment return or the hourly review; after it is recorded, end your turn.` Keep a filled kernel at 600 characters or fewer.

### 6. Convergence goals: done is a disproof

For goals that ask for excellence (a maturity mandate, a quality pass, "production grade"), a passing suite is where agents stop, so make green the trigger to start looking.

- The stop condition is an empty adversary: done only when a fresh, hostile review pass finds nothing material twice in a row. Each pass judges correctness, edge cases, maturity, cleanliness, maintainability and (UI) polish against the named mockups, not feature presence.
- The implementer self-critiques its whole boundary; an independent reviewer keeps the acceptance decision.
- Relentless means toward the spec, simplicity and less debt, not accretion: new debt and over-engineering are findings.
- Track findings in the scenario's durable files, not a plan (a convergence loop has no phase order); use a mandate only when sensors and setpoints are needed.
- Persistence is one clause plus the stance message and kernel: "Keep going through friction; when progress stalls, step back: write why, then park, switch or shrink, instead of stopping or adding more of the same." Do not stack more controls; each lengthens every resume.
- Carry the stance verbatim and pair it with a destination of matching size; a stance without one is tone. Name the real stakes, truthfully.
- Operator feedback is a gating input: the agent appends each item verbatim to the feedback ledger before acting, re-reads it each pass, and is not done while an item is open.

```text
Before handing off <target>, run two consecutive fresh, hostile review passes that find nothing material. Treat a green suite as the signal to start a new pass. Each pass judges correctness, edge cases, cleanliness and maintainability, and visual polish against <mockups/spec>. Record findings and resolutions in <durable doc>; your self-review does not replace independent acceptance. Leave <target> cleaner and with less debt than you found it; new debt is a finding. Stakes: <why this matters and what stays blocked until it is fixed>.
```

### 7. Acting under a goal

Read the named sources before code; show the proof; say "blocked" only for a decision, credential or approval you lack, and name it; checkpoint before the budget fires; never widen scope, loosen a test or rerun unchanged validation for a greener result. When the harness offers `ProposeGoal` and the operator's words already state a verifiable outcome, propose it instead of asking, and never one that widens scope. A goal must not restate a plan's design, add a completion clause the harness can satisfy by assertion, or change a receiving skill's doctrine.

### Anti-patterns

| Anti-pattern | Correction |
|---|---|
| Feeling-word destination, count destination, or ambition only as adjectives | A state a command can show, large on its face, plus the stance slot |
| "Validated" with no command; completion by self-report | Name the check, require its output; independent review for material outcomes |
| The same tolerance preamble retyped every dispatch; a goal that explains the design | Put the rule in the receiving skill and the design in docs; the goal selects a posture |
| A sub-agent with no model named; many minute-long workers for one outcome; full-history forks | Name the role; one owner per outcome; fresh brief with `fork_turns=none` |
| A long goal message with no kernel | Stance message, kernel of 600 or fewer, charter, one unit per turn |
| A plan for every sub-assignment | Choose the cheapest shape |

### Troubleshooting & Edge Cases

- **Evaluator keeps returning Not yet met**: the proof never entered the transcript; rerun the check so its output shows.
- **Agent stops after a few turns**: several turns without tool use; restate the next concrete action.
- **A long effort starts small or shrinks**: diagnose with `effort-diagnosis` before rewording; the cause is usually structure (worker lifetime, forks, exits).
- **Children never spawn workers**: the brief did not ask them to, or the session has the default Codex hint and slot limit (`LONG_EFFORT_DESIGN.md` §4).
- **Codex reports `budget_limited`**: not completion; read the wrap-up and start a new session from the handoff.
- **No native support**: Agent Manager appends `until` to the prompt and the receiving skill's stop rules carry the loop; do not invent a stop condition the receiver lacks.
