# Long effort design

Use this guide when a goal, manager or sub-agent tree must sustain ownership across hours or context compaction. It holds what [RULES.md](RULES.md) does not: how closure is decided (R5), the briefs that set ambition (R3), the effort structures, the verified harness settings and the measured baselines. A bounded handoff needs only an outcome, constraints and acceptance evidence.

- Writing a sustained handoff? Start with `harness-goal-authoring` and the [design rules](#2-design-rules).
- Recovering after compaction? Re-read the charter; see [what Codex keeps](#4a-what-codex-keeps-through-compaction).
- Diagnosing an under-delivering effort? Use `effort-diagnosis` (mechanisms M1-M30: `scenarios/prompt-manager/store/skills/packs/core/effort-diagnosis/references/mechanisms.md`) and the [baselines](#5-baselines-measured-2026-11-06).

Method lives in skills: `harness-goal-authoring` writes goals and briefs; `research-coordination`, `ecosystem-hunt` and `scenario-improvement-campaign` run efforts.

## 1. Two facts the design follows

**Change size tracks ownership time.** One agent changes about 500-650 lines per active hour (added plus removed; gaps over 10 minutes excluded). Across 1,336 Web Console Codex threads (2026-10-14 to 11-05), none of 446 non-review workers active under 6 minutes reached 1,000 changed lines; 67% of workers active 2-8 hours and 91% of those over 8 hours did. This is a correlation, but the consequence holds either way: a large change needs many agent-hours on one target, and a structure that ends every worker after minutes cannot produce it (R3). Magnitude means outcomes (classes closed, owners deleted, callers migrated); lines are a signal, never a target.

**Agents act on what is near and concrete.** After hours and several compactions, an agent's context holds the system prompt, AGENTS.md, its recent messages and the latest tool output. A long goal, a skill read hours ago and a stance paragraph lose force first. Procedures beat adjectives: an agent executes "record unknown as unknown" and treats "be ambitious" as tone. In every high-output session measured, the operator was the control loop (13-43 steering messages per session) and even those sessions were less ambitious than intended. An unattended effort needs a structure that does that job: re-present the goal, propose large scope, reject small returns, keep owners working.

## 2. Design rules

| # | Rule | Counters |
|---|---|---|
| D1 | **Ambition is a destination, not an adverb.** Name an end state that is large on its face, so small work is visibly unfinished ("Device Studio renders scenes, models and motion end to end"). | M6 |
| D2 | **The agent ends units, never the goal.** The manager one loop up decides closure (R5); the [challenger brief](#2a-the-challenger-brief) is its instrument, and the agent that wants to close does not write the question. Open-ended work never reaches `NONE`, so the manager sets a checkpoint (a date or scorecard value) at which closure is proposed to the operator with its costs. Blocked means one named human-owned decision, routed while other work continues. No-change units do not count toward a count destination. | M3, M4 |
| D3 | **Kernel, stance message, charter.** The native goal is a kernel of 600 characters or fewer: destination, a stance excerpt in the operator's words, the charter path, "one unit per turn". The stance message is an ordinary user message of 3,500 characters or fewer sent before the goal: the operator's full stance verbatim (R3), the destination with its scale, and "re-read <charter> at every unit start and after compaction". Codex keeps both through compaction while fewer than eight newer user messages follow (§4a); a long goal it does not keep. In an Agent Manager run the stance message opens the task prompt. Every operator message counts toward the eight, status questions included, so the charter re-read is what carries the stance through a long effort. A status question is answered from the state page (D15) and is not steering. | M1, M13, M15, M27 |
| D4 | **One unit per turn, and the turn spans the wait.** Under a native goal, the agent ends its turn after each unit's record, so Codex re-injects the objective and completion audit before the next unit. The scheduler is the effort's manager (R5); its unit is one owner return (a showcase claim, an operator-gated item or a failed owner) or one due review; it blocks for that inside the turn with `wait_agent` at `timeout_ms: 3600000`, and a wake that brings neither is followed by another wait. Ending the turn while owners are only working makes the goal re-inject at once and the scheduler repeats its unit-start reads every few minutes (§5). Without an active native goal, ending the turn ends the effort. The agent can pause its goal but only the operator can re-activate it (`/goal resume`): after a pause, the first line of the reply to "resume" names that action, and the scheduler holds its wait inside the turn until the goal reads active. | M1, M24, M25 |
| D5 | **Structure for ownership time.** A thin scheduler at the root; segment owners that own a coherent outcome for hours and are re-tasked with `followup_task` while productive; workers sized to an outcome, not a step. Short or cheap workers are for lookups and reviews only (§3). | M8 |
| D6 | **Fresh briefs from the charter.** Every brief is generated from the charter and carries the stance verbatim. Use `fork_turns=none`: a full-history fork inherits the parent's drift, most of its context window and its model. | M9 |
| D7 | **Curated precedent.** Show workers pinned exemplars of a good unit and recent output as numbers against the destination. Keep no-change verdicts in a separate list, out of the narrative a new worker wakes into. | M2 |
| D8 | **Scope is set by someone who does not pay for it (R5).** An architect pass proposes large targets; an ambition review rejects plans as too small; executors then deliver (§2b). The executor may enlarge scope and may not shrink it without the reviewer. | M7, M19 |
| D9 | **Tools do the bookkeeping.** Identities, allowances, worker state and usage come from owner commands and the analyzer, not manager prose. The manager keeps every slot staffed with an owner, re-tasks and challenges small returns. Idle slots while outcomes are open are a defect; polling is not work. | M11 |
| D10 | **Measure with tools; tripwires, not floors.** Track worker active time, outcomes per unit, added and removed lines separately, no-change streaks, slot use and time since the charter was read (`effort_anatomy.py drift`). A tripwire triggers re-orientation (re-read charter, architect pass), never a line quota; a reviewer who rejects returns by line count teaches padding. | M2, M5 |
| D11 | **Cluster before assigning.** Group queue findings into classes and assign the class; write each verdict back to its finding so the queue drains; time-share focus areas instead of strict priority. | M12 |
| D12 | **Name the winning skill.** When an effort reads several skills, its method skill wins over another skill's stop or threshold rule. Say so in the charter. | M13 |
| D13, D14 | Intent quoted verbatim (R3); under-delivery and overclaiming both reported plainly (R2). Every charter carries a denominator (scorecard, workstream table) so small units are visibly small. Rigor follows reversibility: owner tests and a running showcase validate building; controlled trials are for choices expensive to reverse. | M18-M22 |
| D15 | **Charter, state and ledger are three files (R4).** The charter (stance, destination, cards, procedure, brief template) is stable and under about 25 KB, because the scheduler re-reads it at every unit start and after every compaction. Current state is one overwritten page (`STATE.md`): what needs the operator, then each card's status, owner, last measured change and next review. History is an append-only `LEDGER.md` nobody re-reads in full. Owners read their card and mission, never the cohort ledger. | M11, M28 |
| D16 | **Review workstreams, not increments.** One ambition review per workstream plan and one challenger per showcase claim (§2a, §2b). Increments are verified by owner tests and the running showcase, not by a fresh agent, and are not reported one by one. A rejected plan is answered by building its larger first item, not a longer plan. Code that moves money or deletes data gets one independent review before its showcase claim. A reduction target names the structures it replaces; line count is the readout. | M4, M26 |
| D17 | **What needs the operator comes first, in plain words (R9).** Each genuine choice goes through `notification-hub conversations ask` (for an enrolled effort, `agent-manager effort decision raise`), written by the `notification-hub` skill for a reader with no context. The Hub delivers the ask; the answer is written to `DECISIONS.md` in the home ([LEVELS.md](LEVELS.md)), which stays the truth (R4). The state page and every reply open with `Needs you`: one or two plain sentences, the recommendation, next action, answer link and the absolute path to the packet. Never ask for a chat answer or repeat a terminal Hub decision after compaction. A workstream files one genuine request and its owner takes the next workstream while waiting. Under a charter that hands delivery to research-desk, the scheduler writes the item to `STATE.md` and delivers it only if the desk has not within two reviews. | M23, M29, M30 |

### 2a. The challenger brief

Use verbatim whenever a unit, study or goal is proposed for closure or "nothing material". Spawn the challenger fresh (`fork_turns=none`) with the original commission or charter, the destination and the evidence the closer cites. The closer may add evidence; it may not change the question.

```text
You are a challenger. Read <charter or commission path> and the cited evidence. The destination is: <destination>. List every consequential route still open toward it, ranked by value: for each, what it would deliver (outcome, owners, scale), the first concrete step, and why it is or is not permitted under the charter. Include the 10× version of the remaining work toward the destination. A route counts only while the destination is not yet true; list larger work beyond it separately as proposed next destinations, which do not block closure. Do not judge effort already spent. Return the list, or "NONE" only if no permitted consequential route toward the destination exists, with the reason for each route you rejected.
```

Closure proceeds when the list is `NONE`, or, for open-ended work, when the manager's checkpoint is reached and the operator accepts the proposed closure (D2). Otherwise the top permitted route becomes the next unit.

### 2b. The architect and ambition-review briefs

Run the architect pass when an effort starts, when every open item is small, when a unit moved no scorecard dimension, or when a `drift` tripwire fires. Run the ambition review on the architect's output and on any plan an executor proposes before staffing it. Spawn both fresh (`fork_turns=none`, model and effort omitted so they inherit the parent's policy-chosen model; see [Choosing a model](../configuration/coding-agents.md#choosing-a-model-for-a-sub-agent)). Neither executes anything.

```text
You are the architect for <effort or topic>. Read <charter or MISSION.md> and <topic README or ledger>; the operator's words in it are the intent. Ignore how large past units were. Propose the workstreams that would make the end state true: for each, the destination, the scorecard dimensions it moves, the owners and files it will change, the agent-hours it needs (most should be 20–200), and the showcase that proves it landed. Include work that deletes, consolidates or migrates as well as work that adds, and anything the operator asked for that no current item covers. Rank them by value toward the end state. Return the table and one line on the largest risk.
```

```text
You are the ambition reviewer for <effort or topic>. Read <charter or MISSION.md>; the operator's words in it are the bar. Review <the proposed table or plan>. For each item, answer: if this lands exactly as written, how much of the end state is true, and what would the operator say? Return REJECT when an item is a probe, a datapoint, a screen, a polish of one case, a "decision-ready" report where building is authorized, or under about 20 agent-hours without being a task inside a larger item; when the items together leave a dimension or a stated operator wish untouched; or when the work stops short of the showcase. Give the larger replacement for each rejected item. Return ACCEPT only when landing the whole table would make the operator say the end state was pursued at full scale.
```

The executor may enlarge an accepted item. Shrinking or splitting it needs the ambition reviewer again.

## 3. Effort structures

| Tier | Role | Lifetime | Holds |
|---|---|---|---|
| Root scheduler | Reads the charter and state page, spawns or re-tasks owners from the charter template, blocks until an owner returns or a review is due (D4), has a fresh challenger audit each showcase claim (D16), overwrites the state page, appends one ledger entry, ends its turn. No domain work. | The whole effort | The native goal kernel, state page, ledger |
| Segment owner | Owns one coherent outcome (a study's next epoch, ten hunt rounds, one migration), spawns its own workers, re-reads the charter at each round start, does not return before its end state or time box. | At least 2 hours unless the outcome finishes sooner; about 6 hours; re-tasked at a return inside the healthy window | A brief generated from the charter |
| Worker | Implements, hunts or reviews one outcome sized to the work. | Matches the outcome | Its brief |

Segments of about 2-6 hours stay inside the healthy window measured on the ecosystem hunt (about 6 hours and 7 compactions before decay). At a showcase claim or after about 6 hours the scheduler re-tasks the owner, unless it is past 6 hours or 8 compactions (`drift` tripwire `owner-age`); then a fresh owner is briefed from the charter and reads the old owner's checkpoint first. An owner is not replaced mid-work unless stuck or failed; an owner whose turn ended or whose checkpoint is over two hours old is re-tasked or replaced at the hourly review. An owner messages the scheduler for four things only: its plan, a showcase claim, an item needing the operator, and a need it cannot meet itself. Progress goes to its checkpoint file; every message wakes the scheduler, so progress messages turn its wait into polling.

**Slots.** Codex runs a fixed number of concurrent threads per session and every waiting ancestor holds one. Vrooli raises the default 4 to 8 (§4). The scheduler holds one slot, each owner one, and the rest is shared by workers and reviewers: run no more owners than leaves about one worker slot each, write the real number in the charter, and rotate owners between outcomes at checkpoints. On `agent thread limit reached`, work in the current thread or wait; never retry the spawn in a loop.

**Harnesses.** `followup_task`, `fork_turns` and multi-level trees are Codex shapes. In Claude Code, sub-agents spawn through the Agent tool only when their agent type has it (`implementer`, `lookup`, `researcher` and `reviewer` do not), so use two tiers: the session is scheduler and owner, its workers do not delegate. `resource-claude-code ensure` installs a `SessionStart` hook with the `compact` matcher that prints a reminder to re-read the charter; Codex has no such hook.

**Goal-bearing children.** `spawn_agent` cannot give a child a native goal; Agent Manager can. Contract: [effort supervision, "Nested runs"](../../scenarios/agent-manager/docs/reference/effort-supervision.md#nested-runs).

| Need | Command |
|---|---|
| Start a goal-bearing owner | `agent-manager run create --task-id <t> --profile-id <p> --parent-run-id <own run> --until-file <kernel> --charter <abs charter path>` |
| Wait without polling | `agent-manager run wait --children <own run> [--since <observed_at>]`, or `run wait <child>`; exit 0 settled, 5 timed out, 6 unreachable |
| See the tree | `agent-manager run tree <root> [--json]`; `orphaned` marks a live child whose parent ended |

Children deeper than `runExecution.maxChildDepth` (default 3) are refused. The root's owner window bounds the whole tree's authority. Only a run Agent Manager launched can be a parent; start the scheduler as an Agent Manager root run with `--until-file` and `--charter`. **Agent Manager is optional, never required** (R4): on exit 6, a refusal or any outage, do the same work with native sub-agents and the same brief, record the switch in the ledger, and do not wait for recovery.

**Cost.** Per-request cost grows with context only until the first compaction. One coherent outcome gets one owner for its whole length; a new outcome gets a fresh owner with a curated brief.

## 4. Codex multi-agent settings

Verified with depth probes on Codex 0.166-0.167, 2026-11-06 to 11-10. Values are declared in [`resources/codex/native-settings.json`](../../resources/codex/native-settings.json), their one source; `resource-codex ensure` applies them (`settings plan|apply|decline` to inspect). Vrooli writes a key that is absent or still holds the value it last wrote; a value the user chose is reported as a conflict and left alone unless `apply --force`. To change a value, edit the document and re-run `ensure`; do not hand-edit `config.toml`. The config is read at session start. `codex features list` reports `multi_agent_v2 … false` even when sessions run it; read `multi_agent_version` in a rollout's `session_meta`.

| Setting | Default | Verified effect |
|---|---|---|
| `features.multi_agent_v2.max_concurrent_threads_per_session` | 4 | With 4 a chain stopped at level 3 (`agent thread limit reached`); with 8 it reached level 7. No depth limit hit up to 7. |
| `features.multi_agent_v2.multi_agent_mode_hint_text` | "…Do not spawn sub-agents unless the user or applicable AGENTS.md/skill instructions explicitly ask…" | Replaces the `<multi_agent_mode>` message for root and every child; survives compaction. |
| `features.multi_agent_v2.subagent_developer_instructions` | none | Injected as a developer message into every child at every depth. |
| `agents.max_depth` | none reached at depth 7 | Unverified: `-c agents.max_depth=3` did not stop a chain. The thread cap is the working guard against runaway nesting. |
| `features.multi_agent_v2.min_wait_timeout_ms` | none observed | A floor for every `wait_agent` call (0.167: a 5000 ms request with floor 90000 returned after 90.0 s). Vrooli sets 120000. |
| `features.multi_agent_v2.default_wait_timeout_ms` | unknown | The timeout of a `wait_agent` call that passes none. Vrooli sets 600000. |
| `wait_agent` `timeout_ms` | n/a | 3600000 is accepted and held (a probe blocked 848.3 s using 9,956 tokens in all); a full hour is not yet measured. A blocked wait makes no model request. A wait ends early when a sub-agent messages or finishes and when the operator types. A wait with no sub-agent running does not return early: with a 600000 floor, a 5000 ms call blocked 600.1 s, which is why the floor is two minutes. |

`usage_hint_text`, `root_agent_usage_hint_text`, `subagent_usage_hint_text`, `max_wait_timeout_ms` and `preserve_fork_prefix` are unverified: probe before relying on one and record the result here.

A native goal has three agent-settable states (`complete`, `blocked`, `paused`; the `update_goal` tool has no `active`). After the agent pauses at the operator's request, only the operator can re-activate it (`/goal resume`). An agent's own `update_goal` and `create_goal` calls write no `thread_goal_updated` event.

### 4a. What Codex keeps through compaction

Measured on Codex 0.166, 2026-11-06: 377 compaction records in Web Console homes (117 root threads, 260 children), 119 from one parked Agent Manager orchestrator, and one probe. Compaction is remote; the summary is an encrypted item.

| Item | Kept |
|---|---|
| Ordinary user messages | The 8 most recent, verbatim up to about 3,600 characters each; a longer one keeps about 1,800 from each end. Goal set/clear events and Agent Manager wakes (`[wake] …`) are user messages and count toward the 8; goal-turn injections do not. |
| A native goal's objective | Verbatim when short: the longest kept was 669 characters and the shortest dropped 711, so the kernel stays at 600 or fewer (`runner.NativeGoalVerbatimLimit`). Otherwise replaced by "[objective omitted; exceeds the evidence limit]"; it returns in full at the next goal-turn start. |
| A sub-agent's brief | A child has no user messages (0 of 260 retained any). Its task stays as encrypted items, so how much of a long brief survives is unknown. |
| Assistant messages | The last 8, short. |
| Skill text and tool output | Not kept except through the encrypted summary. |
| `compact_prompt` | No effect observed (one probe; re-verify after an upgrade). |

So the stance survives when it is in a user message of 3,500 characters or fewer, most important text first, with fewer than 8 newer user messages after it. A parked run woken more than seven times loses its task prompt (in one orchestrator 111 of 119 compactions kept eight wakes and nothing else), so only the charter re-read carries the stance. A brief puts stance and destination first. `retained_context` in each `compacted` record shows what survived (`effort-diagnosis` `references/transcripts.md`).

## 5. Baselines (measured 2026-11-06)

Re-measure with `effort_anatomy.py` (`scan`, then `summary`) before citing these for another period.

| Measure | Value |
|---|---|
| Changed lines per active hour (all threads) | ~650 before 2026-10-29, ~515 after |
| Non-review sub-agents by active time: count, median changed lines, share at 1,000 lines or more (Web Console, 1,307 threads) | <6 min: 446, 0, 0%. 6-30 min: 191, 94, 2%. 30 min-2 h: 51, 346, 22%. 2-8 h: 21, 2,143, 67%. >8 h: 11, 15,852, 91% |
| Regime change 2026-10-29/30 (Codex 0.161 to 0.166 and "delegate cheaply, fresh worker per outcome") | workers/day 1-26 to 84-325; median worker life 30 min-hours to 1-3 min; children starting with >100k inherited tokens 0% to 36-47% |
| Highest-output sessions (2026-10-17..21) | 35k-93k lines; 9-23 long-lived owners; 13-43 operator messages; one owner re-tasked 160 times |
| Native goals ending `blocked` | 36% of 484 |
| Goals of 1,500 characters or more | 47% paused by the operator, 17% complete |
| Compactions that dropped the objective (2026-11-05) | 206 of 283; every retained objective was 730 characters or fewer |
| Research manager 2026-11-05 | closed itself 1 h 15 min into a 20 h allowance; median worker life 6.8 min; ~4.3k code lines in 9.5 h |
| Four-programme cohort 2026-11-08..10 | 173 threads, 82k changed lines in 12 active hours. The scheduler paused its goal at the operator's request and could not resume it: 11 h on 14 operator messages, one owner idle 129 min. After re-activation: 31 goal turns (4.6-min median), 48M input tokens an hour for the scheduler, a compaction every 20 min, 12-14 messages an hour to each owner. The charter grew from 18 KB to 177 KB, re-read 4-7 times per turn. 92 of 172 sub-agents were reviewers or challengers; a 25% reduction target moved 18 lines in 10 h while four reviews rejected the plan. The notification consent request produced one push in 30 h |
| Model tiers (not controlled) | 10 weekly-quota points in about 40 min on `gpt-6-astra` at `max`; about 5 in 10.5 h on `gpt-6-luna` at `high` with four owners; Luna changed lines per hour were about 60% of the Astra burst |

**Delivery-loop lessons (October 2026, retired programme).** Recurring causes of lost time: ratchets that let agents add code and never remove it; a deletion-list rule misread as "admit nothing" (3.5 days idle, a 93 KB goal home against a 20 KB budget); a destination nobody reconciled with an owner map; an operator decision that waited 3.5 days because nothing notified the operator; and rules held only in skill prose, which a long-lived run ignored within a day because it held the text it read days earlier. The fix each time was a tool or a file the agent re-reads (R4, R7), never more prose.

## 6. Open platform gaps

| Gap | Current workaround | Owner |
|---|---|---|
| Codex re-injects the objective only at goal-turn starts, never after compaction | D3 kernel and stance message, D4 one unit per turn; `subagent_developer_instructions` tells children to re-read their brief | Codex (upstream) |
| Drift signals needing per-turn lines or narration are not computed live; Claude Code runs get only `short-workers` | Agent Manager shows the other triggers in `run tree`, `effort board` and the Efforts page; run `effort_anatomy.py drift` (exit 3 on a tripwire) at each segment return | Agent Manager |
| `spawn_agent` cannot carry a native goal | Goal-bearing segments run as Agent Manager child runs (§3) | Codex (upstream) |
| Agent Manager wakes are user messages, so a parked run woken eight times loses its task prompt | Runs created with `--charter` end every wake with "Re-read your charter <path> before acting." Give long parked runs a charter | Agent Manager (resolved for chartered runs) |
| Attached sessions cannot parent Agent Manager runs | Start the scheduler as an Agent Manager root run | Agent Manager |
| A paused native goal has no agent-side resume, and a native session has no timer | D4: name `/goal resume` first and hold the wait inside the turn; `drift` fires `paused-goal`. A managed root run can park on a review timer (`scenarios/agent-manager/docs/internal/TEMPORAL-FLOWS.md`, "Optional review timer") | Codex (upstream); Agent Manager for managed runs |
