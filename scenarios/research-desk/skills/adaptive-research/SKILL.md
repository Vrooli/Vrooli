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
  revision: 24
  createdAt: "2026-10-21T00:00:00Z"
  updatedAt: "2026-11-20T00:00:00Z"
  requires:
    commands: ["rg"]
  origin:
    kind: "authored"
---
## Adaptive research

Rules: [RULES.md](../../../../docs/agent-system/RULES.md). You own a direction and drive it to its destination, choosing each next step from what you learn. When the commission allows building, build: a working capability exercised end to end beats a report or a chain of probes.

### Your target

Your workstream's destination, magnitude, showcase and scorecard dimensions in `MISSION.md` are your target; the operator's verbatim words are the bar. You may enlarge it; shrinking it needs the ambition reviewer. Think in whole APIs and scenarios; ask what the 10x version is. You are re-tasked while productive. Re-read your brief and mission at each checkpoint and after compaction. One finished check or failed route does not end a direction; a fresh challenger decides closure (`research-coordination/templates/review-briefs.md`).

### Start

1. Read your brief, your `MISSION.md`, and what the card names. Not the whole topic folder, the commission folder or the ledger.
2. Recall what exists: `search-hub query "<intent>"`, `rg` over the topic and code. Look at the code you will change.
3. Form your own framing; leads are hypotheses. Restate the destination and the operator's words in your first checkpoint and build within the first hour.

### Build and prove

- Size each change to the destination. Integrate with others' uncommitted work; never discard it. Delete what you replace (R8).
- Prove increments with owner tests, the relevant Test Genie phases and the showcase running. Ask for a challenger once, at the showcase claim.
- Do not write your own exam: use a proving ground (real software, a scenario serving graded levels, real public material at real size, the trade's mature tool), and build or propose it as a reusable capability, never a private script in a run folder.
- A controlled comparison (`local-experimentation`) only for a choice expensive to reverse; otherwise decide by inspection.
- Spawn workers for parallel or long parts, each owning an outcome for hours with a fresh brief and the evidence links it needs. Call `wait_agent` only when you can only wait, never with no worker running. A deliverable finished in minutes was not attempted at scale: enlarge it.

### When something is in the way

- **Optional services fail fast.** Take the direct path (native sub-agents, the command itself, the skill file by path, `rg`) and add `Route failure · <UTC> · <route> · <command> · <first error line> · direct path taken` to your checkpoint. Retry at most once per checkpoint.
- **A real blocker in your scope**: find its owner and the failure; repair, work around or take an independent piece. An engineering contract or an uncommitted file is not an approval gate.
- **Only the operator can decide it** (a bright line or a direction): message the manager with an ID, two plain sentences, your recommendation and the path of a ready-to-run brief. Keep working.
- **The showcase can only wait** for the operator, elapsed days or capacity: say so in one message, mark it `waiting-operator` or `waiting-time resume <date>`, take the next workstream.
- **A check keeps returning the same answer**: test what separates causes, repair, work around or pivot.

### Checkpoint

Keep `runs/<run>/CHECKPOINT.md` current (your brief lists its parts) and under 30 KB; move settled sections to `checkpoints/<date>.md`. Message the manager only with your plan, a showcase claim, an operator item or an unmet need. A window that moved nothing is reported as under-delivery with its cause. Receipts go to the topic `LEDGER.md`; decision-changing findings go to its knowledge pages or README, rewritten, not appended. Each run folder's `README.md` states its outcome. Containers you start use named Docker volumes, never a bind mount into the research home. A list workstream keeps its list file current: assess every new `Wanted` line. A proposal names every resource and scenario the build needs and what the prototype faked.

### Traps

| Trap | Instead |
|---|---|
| A chain of small accurate probes | Build toward the destination |
| Polishing the most testable corner | The work with the largest scorecard movement |
| Changing the evaluation after seeing a candidate | Freeze it, then confirm on fresh runs |
| Restarting work after a quiet screen | Reconcile the original operation first |
| Closing on a narrow proof | Hand the manager the candidate frontier; a challenger decides |
