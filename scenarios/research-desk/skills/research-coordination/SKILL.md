---
name: "research-coordination"
description: "Commission and manage Vrooli research: turn topic missions into a packet, staff each workstream with an owner who builds for hours, and judge progress by what landed."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["practice"]
  tags: ["research", "experiment", "coordination"]
  icon: "compass"
  status: "active"
  revision: 40
  createdAt: "2026-10-26T00:00:00Z"
  updatedAt: "2026-11-20T00:00:00Z"
  requires:
    commands: ["python3"]
  origin:
    kind: "authored"
---
## Research coordination

Rules: [RULES.md](../../../../docs/agent-system/RULES.md). The research manager is a goal session (`research-desk launch`); its charter opens with the same card.

```
Setpoint:  each window lands its cards' workstreams at the pace on its charter's first lines; a window that moves no scorecard dimension is under-delivery
Sensors:   `history.py` landed/stalled, `effort_anatomy.py drift`, `footprint.py`, `topic_health.py`, and in `research-desk now` the FLAG lines (`flags` in `--json`: a dead window, STATE.md `Goal:` contradicting the goal) and the `ANSWERED, waiting` lines (`answers_waiting`); `research-desk topics --json` gives `landed_10d` and `last_landed_at`. Only the flags are proven, by fixture tests (`home_test.go`); no live canary is planted, so the rest are unproven (R6)
Actuators: the packet and charter (`packet.py`), a commission's `STATE.md`/`LEDGER.md`, owner briefs, each topic's `work/`, `DECISIONS.md` and inbox; no domain work, no code
Manager:   the portfolio steward (it reads `research-desk now` and the flags)
Tooling:   `scripts/` here and in `research-review/`, `templates/limits.json`; each is kept only while a failure it caught recurs
```

A tidy folder, a long ledger or a report is not progress (R2).

### The topic

`topics/<topic>/` in `~/.vrooli/research`. `MISSION.md` holds the operator's words verbatim, the end state, a scorecard and workstreams of 20-200 agent-hours ([missions](references/mission.md)); the rest is a small shape the topic extends ([topic folder](references/topic-folder.md)). Check missions with `python3 -I scripts/mission_check.py ~/.vrooli/research`. Its mode sets what owners build directly and what goes to the operator (table in [missions](references/mission.md)).

### Commission a window

1. Start from the missions, not the last checkpoint. Small workstreams, a stale architect pass or a pending ambition review get the architect pass first (`templates/review-briefs.md`).
2. Build the packet: `research-desk build <spec>` or `python3 -I scripts/packet.py preview|build --home ~/.vrooli/research --spec <spec.json>`. It renders the charter, `STANCE-MESSAGE.txt` and `KERNEL.txt`. `stance_chars` and `kernel_chars` are hard gates (Codex drops a goal over about 700 characters); a `charter_bytes` overflow only warns and the build goes ahead. `mission_check` errors refuse only a defect that would make the window fail (no Mode, a missing section, nothing to staff). Read ceilings (`mission_bytes`, `skill_bytes`, the start sets) only flag, in the preflight and `research-desk now`: cut text, never raise a ceiling.
3. Launch: `research-desk launch <id>`, or in a fresh Codex session send `STANCE-MESSAGE.txt`, wait for "ready", then `KERNEL.txt` as the `/goal`.
4. Asked only to prepare? Build the packet and stop; say the exact action that commissions it.

One owner per workstream, re-tasked while productive; rotate owners between workstreams, never shrink them into minute-long workers. An idle slot while a workstream has a next step is a defect, and polling status is not work. Never wait on an agent when no worker is running.

### Judge by what landed (R2)

Measure topics with `python3 -I ../research-review/scripts/history.py topic <t>`. A workstream lands when its showcase runs; lines changed are not landing. Diagnose under-delivery (`effort-diagnosis`) and say so. A study ends early only when a fresh challenger returns `NONE`. A showcase that waits is `waiting-operator` or `waiting-time resume <date>`; its owner takes the next workstream. Owners stop what they start.

### Routes fail (R4)

Agent Manager, Program Runtime, Prompt Manager, Search Hub, Source Ledger, Notification Hub and Research Desk are optional. When a call fails, take the direct path that turn (native sub-agents, the command itself, a skill file by path, `rg`) and log `Route failure · <UTC> · <route> · <command> · <first error line> · direct path taken`.

### Decisions travel up, friction travels down

Each level keeps its decisions, plans and inbox in its own folder ([levels](../../../../docs/agent-system/LEVELS.md)). A decision is a needs-you item with a ready-to-run packet that a fresh executor runs on approval. An answer waiting in an inbox is approved work: start its executor (`needs_you.py spawn <ID>`) at once, never at the next review; a new charter lists them in its Start step. Friction goes down through the owning level's inbox, written and drained only with `scripts/inbox.py`. Ask the operator only for a bright line or a direction, never to write a prompt; each new ask names what silence does ([operator attention](references/operator-attention.md)).

### Traps

| Trap | Instead |
|---|---|
| The most testable corner | The workstream with the largest scorecard movement |
| A deliverable proved only on a target its builder wrote | A reusable proving ground plus real targets ([missions](references/mission.md#proving-grounds)) |
| A run folder, ledger entry or reviewer per increment | One run folder per workstream, one ledger entry per review, one reviewer per showcase |
| Steering a live manager by chat | `research-desk say <id> direction`; `research-desk feedback <id> <text>` for verified findings |
