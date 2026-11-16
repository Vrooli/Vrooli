---
name: "research-coordination"
description: "Commission and supervise Vrooli research: turn topic missions into a packet, staff each workstream with an owner who builds for hours, keep every slot working, route around broken infrastructure, and measure progress by what landed."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["practice"]
  tags: ["research", "experiment", "coordination", "supervision"]
  icon: "compass"
  status: "active"
  revision: 39
  createdAt: "2026-10-26T00:00:00Z"
  updatedAt: "2026-11-15T08:00:00Z"
  requires:
    commands: ["python3"]
  origin:
    kind: "authored"
---
## Research coordination

Research is how Vrooli gets big things done. A topic is a programme with a large mission; a
commission staffs it for a window; owners build real capability end to end; the supervisor
keeps them focused, ambitious and unblocked. Your job is that the work moves: workstreams land,
the scorecard moves, nothing stalls. A tidy folder, a long ledger or a careful report is not
progress.

### Why this setup works

- **One clear entry point per agent.** An owner reads its brief and its topic's `MISSION.md`,
  not the whole folder. A clear direction beats a pile of rules. Keep entry files short.
- **Steering lives in files, never in chat.** Goal mode compacts often and replays recent chat
  messages, so a chat nudge keeps poisoning the context for hours. The supervisor is primed
  once (stance message, "ready", then the goal) and re-reads files: its charter, `STATE.md`,
  `OPERATOR.md`, and `STEWARD.md`. `OPERATOR.md` is authoritative; steward findings are dated,
  attributed guidance below operator words and above the supervisor's own judgment. Use
  `research-desk say` for operator direction and `research-desk feedback` for a verified
  agent's reviewed finding or correction.
- **The supervisor is a manager, not a worker.** It plans, staffs, reviews claims and removes
  blockers. Owners do the domain work for hours each and may spawn their own workers.
- **Infrastructure is a shortcut, never a dependency.** Everything works from files and the
  direct path. See the route rule below.

### The topic

`topics/<topic>/` in the research home (`~/.vrooli/research`). Its `MISSION.md` holds the
operator's words verbatim, the end state, a scorecard and workstreams of 20–200 agent-hours
([missions](references/mission.md)). Check missions with
`python3 -I scripts/mission_check.py ~/.vrooli/research`. The mode sets what owners build directly:

| Mode | Owners build directly | Goes to the operator |
|---|---|---|
| `build` | Everything in the repository | Only real boundaries: money, credentials, rights, private data |
| `propose-then-build` | Prototypes, experiments, improvements to existing capabilities | Proposals for new maintained capabilities; approved ones are built at full scale |
| `gated-effect` | The whole path with the effect stubbed | The effect (funding, broadcast, a physical device) |
| `artifact` | Finished artifacts and the pipeline that makes them | Taste judgments over alternatives |
| `evidence` | Datasets, pipelines, detectors at full scale | Publication or external use |

The rest of the folder is a small common shape each topic extends as it needs
([topic folder](references/topic-folder.md)); templates per mode are in `templates/topics/`.

### Commission a window

1. Start from the missions, never from the last checkpoint. Run `mission_check.py`. A topic
   with small workstreams, a stale architect pass or a pending ambition review gets the
   architect pass and ambition review first (`templates/review-briefs.md`).
2. Build the packet: `research-desk build <spec>` or
   `python3 -I scripts/packet.py preview|build --home ~/.vrooli/research --spec <spec.json>`.
   It renders the charter, `STANCE-MESSAGE.txt` and `KERNEL.txt` from `templates/` and checks
   the size gates in `templates/limits.json`. The spec carries the topics, allowance, the
   operator's stance quotes verbatim, and optional order, emphasis and card notes. Never ask
   the operator to write a prompt.
3. Launch: `research-desk launch <id>`, or by hand in a fresh Codex session send
   `STANCE-MESSAGE.txt`, wait for "ready", then `KERNEL.txt` as the `/goal`.
4. Asked only to prepare? Build the packet, realign missions if needed, and stop. Start no
   clock, goal, worker or effect, and say the exact action that commissions it.

One owner per workstream, re-tasked while productive (the charter counts slots). Too many
workstreams for the slots means rotating owners between workstreams at checkpoints, never shrinking them into
minute-long workers.

### Supervise

The charter carries the scheduler procedure; follow it. In short:

- **Keep every slot working.** An idle slot while a workstream has a next step is a defect.
  Polling status is not work.
- **Review plans and showcase claims once each.** One ambition review per workstream plan, one
  challenger (`templates/review-briefs.md`) per showcase claim. Never review increments. A
  rejected plan means "build the larger first item", not a longer plan.
- **Judge by what landed.** Measure owners with `effort_anatomy.py` (the charter has the
  commands) and topics with `python3 -I ../research-review/scripts/history.py topic <t>`.
  Lines changed are not landing; a workstream lands when its showcase runs. A window that
  moves no scorecard dimension is under-delivery: diagnose it (`effort-diagnosis`) and say so.
- **Closure is a challenger's call.** A study ends early only when a fresh challenger with the
  fixed §2a brief returns `NONE`; otherwise its top route is the next direction.
- **Waiting is not idling.** A showcase that needs the operator, elapsed days or capacity you
  cannot free is marked `waiting-operator` or `waiting-time`; its owner takes the next workstream.
- **Watch the footprint.** Owners stop what they start; the scheduler checks with
  `scripts/footprint.py` ([supervision](references/supervision.md)). Closed runs shed rehearsal
  copies and build trees with `scripts/archive.py plan|apply` (moved, pointer left, links kept).
- **Keep topic tools honest.** A topic's `TOOLS.md` extends what you monitor;
  `scripts/topic_health.py` names each tool that is down or `unknown`, with its fallback.

### The route rule: direct path first

Agent Manager, Program Runtime, Prompt Manager, Search Hub, Source Ledger, Notification Hub and
Research Desk are optional. Use one when it answers on the first try.

| Route fails | Direct path, same turn |
|---|---|
| Agent Manager | Native sub-agents with the same brief |
| Program Runtime | `program-runtime library run research/[<t>/]<name> --input k=v` falls back to the standalone runner by itself; else the underlying commands |
| Prompt Manager | Read the skill's `SKILL.md` by path (topic skills live in `topics/<t>/skills/`) |
| Search Hub, Source Ledger | `python3 -I scripts/find.py "<words>" [--topic <t>]` answers either way; `rg` for code |
| Notification Hub, Research Desk | The topic's `DECISIONS.md` item; `needs_you.py open`; files |

When a call fails, take the direct path that turn and log one line:
`Route failure · <UTC> · <route> · <command> · <first error line> · direct path taken`.
Retry it at most once per review. **Do not repair infrastructure inline.** Repair it only when
it is the workstream's subject, or when the same failure has blocked a proof at three reviews
and no direct path exists; then it becomes an owner's workstream or a bug report
(`prompt-manager skill read report-bug`, or its file by path), and research keeps moving.

### Decisions travel up, friction travels down

Each level (topic, supervisor, CEO) keeps its own open decisions, plans and inbox in its own
folder, so nothing is lost when a commission ends ([levels](../../../../docs/agent-system/LEVELS.md)).
A decision is a needs-you item with a ready-to-run packet written by whoever raised it; on
approval the next agent spawns a worker from that packet and re-derives nothing. Friction and
feedback go down to the level that owns the fix, through that level's inbox, written and drained
only with `scripts/inbox.py` (it stamps the time). Spawn messages are one line plus the stance:
owner "You own `<topic>/work/<ID>`. Read `<abs>/BRIEF.md` and the topic's `MISSION.md`, then build
toward the brief's showcase."; executor "Execute the approved decision `<ID>`. Read
`<abs>/BRIEF.md`; the answer is `<key>`."; reviewers the briefs in `templates/review-briefs.md`. Ask only for what the operator alone owns, and deliver it
as a page they can answer cold ([operator attention](references/operator-attention.md)).

### Plan ahead in files

A topic's planned work lives in `work/` as ready-to-run briefs: destination, scale, starting
points, showcase. The supervisor keeps that plan and its risks current
([topic folder](references/topic-folder.md#plans-and-risks)) and starts owners from those briefs
with one standard message. Planning is the supervisor's job; owners build.

### Close a window

At the allowance end or after `wind-down`, follow the charter's closing step: owners stop what
they started, the final `STATE.md` says per card what landed, moved, waits and is unverified, and
each `MISSION.md` (and `CURRENT.md` where the frontier moved) is updated.

### Traps

| Trap | Instead |
|---|---|
| A commission written from the last checkpoint | Start from `MISSION.md`; receipts are evidence, not the template |
| The operator's intent paraphrased smaller | Quote it verbatim; restatements go in End state and never shrink it |
| The most testable corner (one metadata branch, a four-case comparison) | The workstream with the largest scorecard movement |
| A deliverable proved only on a target its builder wrote | A reusable proving ground plus real targets at real size ([proving grounds](references/mission.md#proving-grounds)) |
| Fixing Agent Manager instead of researching | The route rule: direct path now, repair only as its own work |
| Minute-long workers, or forking the supervisor's history into them | One owner per workstream for hours, fresh brief, `fork_turns=none` |
| A run folder, ledger entry or reviewer per increment | One run per workstream per commission; one ledger entry per unit |
| A disclaimer paragraph after every result | One "Limits" line per run; under-delivery reported as plainly as overclaiming |
| Steering a live supervisor by chat | `research-desk say <id> direction` for operator words; `research-desk feedback <id> <text>` for verified-agent findings |
| A capacity warning closes a run | Schedule, wait through the owner, or move to a fleet node; staff other work meanwhile |
| Topic folders that look mature but read longer | Measure: `replay.py` old vs new, `history.py` before vs after |
