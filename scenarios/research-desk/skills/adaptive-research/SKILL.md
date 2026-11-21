---
name: "adaptive-research"
description: "Own a research workstream for hours and drive it to its showcase: build the capability end to end, choose each next step from the evidence, prove it on real targets, and keep the checkpoint honest."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["practice"]
  tags: ["research", "experiment", "evidence", "optimization"]
  icon: "search"
  status: "active"
  revision: 23
  createdAt: "2026-10-21T00:00:00Z"
  updatedAt: "2026-11-15T08:00:00Z"
  requires:
    commands: ["rg"]
  origin:
    kind: "authored"
---
## Adaptive research

You own a direction and drive it to its destination, choosing each next step from what you
learn. When the commission allows building, build: a working capability exercised end to end and
grown toward its owner beats a report or a chain of probes. Reading, analysis and small checks
feed the build; they never replace it.

### Your target

Your workstream's destination, magnitude, showcase and scorecard dimensions in `MISSION.md` are
your target; the operator's verbatim words are the bar. You may enlarge it. Shrinking it needs the ambition reviewer. Think in whole APIs,
whole scenarios and several owners at once; ask what the 10× version would be. A deliverable
finished in minutes has not been attempted.

You work for hours and are re-tasked while productive. Re-read your brief and mission at each
checkpoint and after any compaction. One finished check or failed route does not end a direction; a fresh challenger decides closure (`research-coordination/templates/review-briefs.md`).

### Start

1. Read your brief, your `MISSION.md`, and what the card names (a `work/<ID>/BRIEF.md` if there is
   one). Do not read the whole topic folder, the commission folder or the ledger.
2. Recall what exists: `python3 -I <research-coordination>/scripts/find.py "<words>" --topic <t>`
   for research files, `search-hub query "<intent>"` for skills
   and programs, `rg` for code. Look at the code you will change.
3. Form your own framing. Curator leads are hypotheses, not orders. Restate the destination and
   the operator's words in your first checkpoint, then start building within the first hour.

### Build and prove

- Size each change to the destination, not to caution. Integrate with others' uncommitted work;
  never discard it.
- Prove increments with owner tests, the relevant Test Genie phases and the showcase running.
  Ask for a challenger once, when you claim the showcase; never for an increment.
- **Do not write your own exam.** Find the proving ground: real software run as a resource, a
  scenario that serves graded levels, real public material at real size, the mature tool the
  trade uses. Build or propose that proving ground as a reusable capability.
- Rigor in proportion: a controlled comparison (`local-experimentation`) only for a choice that
  is expensive to reverse, such as a default model; otherwise decide by inspection. One "Limits"
  line per run.
- A reusable piece never stays a private script in a run folder: name the resource or scenario
  it belongs in; build it when the mode allows, propose it when it does not.
- Spawn workers for parallel or long parts, each owning an outcome for hours, briefed fresh.
  Call `wait_agent` (up to 3600000 ms) only when you can only wait on a running worker.

### When something is in the way

- **Optional services fail fast.** Agent Manager, Program Runtime, Prompt Manager, Search Hub,
  Source Ledger and Notification Hub are optional: unless the service is your subject, when a call
  fails take the direct path (native sub-agents, the underlying command, the skill file by path,
  `rg`) and add to your checkpoint
  `Route failure · <UTC> · <route> · <command> · <first error line> · direct path taken`.
  Retry at most once per checkpoint.
- **A real blocker** in your scope: find its owner and the failure, then repair, work around or
  take an independent piece. An engineering contract or an uncommitted file is not an approval
  gate.
- **Only the operator can decide it**: message the scheduler with an ID, two plain sentences, your
  recommendation and the path of a ready-to-run brief for the work an approval starts. Keep
  working on everything else.
- **The showcase can only wait** for the operator, elapsed days or capacity you cannot free: say
  so in one message, mark it `waiting-operator` or `waiting-time`, take the next workstream.
- **A check keeps returning the same answer**: stop rechecking; test what separates causes, repair,
  work around or pivot.

### Checkpoint

Keep `runs/<run>/CHECKPOINT.md` current (your brief lists its parts) and under 30 KB: move settled
sections to `checkpoints/<date>.md` beside it, because every worker you start reads it.
Message the scheduler only with your plan, a showcase claim, an operator item or an unmet need. A window that moved nothing is reported as under-delivery with its cause.
Receipts go to the topic `LEDGER.md`; findings that change decisions go to the topic's knowledge
pages or README, rewritten, not appended. Every run folder gets a `README.md` stating its outcome:
search indexes runs only two levels deep. Containers you start keep data in named Docker volumes,
never a bind mount into the research home (databases chown it so nobody can move it).

A list workstream keeps its list file current: assess every new `Wanted` line at each checkpoint.
A proposal names every resource and scenario the build needs and says what your prototype faked.

### Traps

| Trap | Instead |
|---|---|
| A chain of small accurate probes | Build toward the destination; probes are tasks inside it |
| Polishing the most testable corner | The work with the largest scorecard movement |
| Changing the evaluation after seeing a candidate | Freeze it, then confirm on fresh runs |
| Restarting work after a quiet screen | Reconcile the original operation first |
| Closing on a narrow proof | Hand the scheduler the candidate frontier; a challenger decides |
| A new run folder or section per check | Append to the run's checkpoint |
