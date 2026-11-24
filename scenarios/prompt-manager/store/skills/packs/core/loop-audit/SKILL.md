---
name: "loop-audit"
description: "A manager's check of one agent, team or workspace below it against the agent rules: observe its loop card, measure movement, prove its sensors, price its tooling, then fix the setup inside your authority and packet the rest."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["practice"]
  tags: ["audit", "manager", "rules", "cascading-control", "workspace"]
  icon: "gauge"
  status: "active"
  revision: 2
  createdAt: "2026-11-20T00:00:00Z"
  updatedAt: "2026-11-20T00:00:00Z"
  origin:
    kind: "authored"
---
## Loop audit

You manage the target (R5). Your job is to keep it moving, so this audit ends in changes, not a report. Read `docs/agent-system/RULES.md` first; every finding cites one rule.

Judge from evidence (files, logs, run history, the live service), never from what the target's docs say it does. A claim you could not check is a finding (R6). Where a team's truth lives: its home folder, Prompt Manager (`team show`, `team execution-status`), Agent Manager (`run list`, `effort list`). When two of these disagree, the disagreement is a finding.

### 0. Can it still fire?

Before anything else, check whether the target can act without anyone deciding: its enabled flag, schedule or timer, team and global pause, and any open commission. An enabled target held back only by a global pause is the most urgent finding an audit can make (R6).

### 1. Observe the loop card

Fill in the five parts as the target actually runs:

| Part | Look for | Finding when |
|---|---|---|
| Setpoint | Its outcome, next gate and pace, written in its home | Missing, several outcomes, no pace ("no pace declared"), or the operator's stance paraphrased instead of quoted (R1, R3) |
| Sensors | What it reads to decide | A signal with no proof it ever caught a real or planted fault; a skip, suspension, timeout or "unknown" that nobody hears (R6) |
| Actuators | What it may change | Reach wider than its outcome needs, or a fix that routes through the thing it fixes (R3, R6) |
| Manager | The loop above it | None, or the last removed blocker is unknown (R5) |
| Tooling | Its own skills, programs, knowledge, checks | An item naming no friction it removed, or tooling that mostly serves other tooling (R10) |

Score each member's pipeline (intake, collection, analysis, promotion): a member fed by external or operator signals needs an intake path, collection tooling and a promotion route, a pure reviewer needs none; a missing stage is a finding that names the smallest tool for it (R10).

Also find its **home**, the folder whose files are the truth. If the truth lives only in a service or a chat, that is a finding (R4).

### 2. Measure

Use the target's last few periods, or its whole run if it has finished. Derive the planned pace from its destination and budget when none is declared.

- **Movement (R2):** count at the level of the target's own destination metric, such as surfaces accepted or scorecard dimensions moved, against the planned pace. Units, reviews and log entries count as writing, not movement.
- **Read cost (R7):** bytes an agent must read before it can act (entry file plus required reading), and contradictions between those files.
- **Debt (R8):** problems that recurred two or more times with no class fix; net lines, concepts or paths added or removed (`n/a` for a target with only history).
- **Operator load (R9):** open asks, and whether each has a recommendation, what silence does, and a default.

**Verdict**, first match wins:
1. `retire`: no outcome is left.
2. `theater`: writing grew while no destination-level gate moved.
3. `stalled`: nothing moved and little was written.
4. `slow`: it moved at under half the planned pace.
5. `moving`: anything else.

### 3. Prove one sensor

Take the sensor the target depends on most. If you can plant a fault safely (a stale file, a missing input, a test fixture), plant it, check the sensor reports it, then remove it. Otherwise cite the last real incident it caught, with its date. For a retired target, compare each service's view of it with the home; any view that disagrees is an unproven sensor. If none of these is possible, the sensor is unproven.

### 4. Act

- **Fix directly** what sits inside your authority: rewrite the loop card, cut the entry file, quote the stance verbatim, delete tooling that fails the removal test, merge duplicated facts into one home, make quiet failures loud.
- **Re-derive the roster** from the gates the target's scenarios enforce: where members exist only to hand hand-kept state to each other, move that state into a scenario or program that enforces the order, then collapse those members (keep any that own a knowledge topic or a coverage ledger).
- **Packet** what needs other work, as one outcome, the evidence and the gate. If you are a read-only auditor, every fix becomes a packet.
- **Ask the operator** only at a bright line or for direction, in R9 form. Decide everything else yourself.
- **Retire** a target that has no outcome left. First move each unresolved fact, tracked as `fact | file | receiving home | moved yes/no`. Retirement is complete only when every line says yes.

### 5. Record

Append the record to the target's home. If you are read-only, or the home is archived custody, append it to your own (the manager's) log instead. Keep it under 60 lines:

```
loop-audit <date> by <manager>
can-fire: <enabled/schedule/pause/commission>
card: Setpoint=… | Sensors=… (proven: …) | Actuators=… | Manager=… | Tooling=…
verdict: retire | theater | stalled | slow | moving
movement: <destination gates passed / planned>   read cost: <bytes>   net debt: <+/- or n/a>
findings: R<n> <evidence> -> <fix done | packet id | ask>
deleted: <what was removed>
```

### Calibrate before you trust it

A new auditor first runs this skill on a reference target with known findings: `~/.vrooli/research/topics/agent-architecture-convergence/audits/calibration-finite-team.md` (finite team `notification-hub-delivery`, verdict `retire`). If your verdict or key findings differ, fix your reading before you audit live targets (R6).
