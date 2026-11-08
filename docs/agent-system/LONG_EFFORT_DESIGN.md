# Long effort design

How to structure goals, supervisors and sub-agent trees so an unattended effort stays as
ambitious at hour ten as at minute one, and can produce changes measured in thousands of
lines when the destination calls for it.

This document owns the design rules, the effort structures, the verified harness settings
and the measured baselines. Method lives in skills: `harness-goal-authoring` writes goals and
briefs; `research-coordination`, `ecosystem-hunt`, `large-effort-orchestration` and
`scenario-improvement-campaign` run efforts; `large-effort-supervision` supervises delivery
orchestrators; and `effort-diagnosis` investigates an effort that under-delivers. Its mechanism catalog
(M1–M23) is in `scenarios/prompt-manager/store/skills/packs/core/effort-diagnosis/references/mechanisms.md`.

## 1. Two facts the design follows

**Change size tracks ownership time.** One agent changes roughly 500–650 lines per active
hour (added plus removed; active time excludes gaps over 10 minutes). Across 1,336 Web Console
Codex threads (2026-10-14 to 11-05), none of the 446 non-review workers active under 6 minutes
reached 1,000 changed lines; 67% of workers active 2–8 hours and 91% of those active over 8
hours did. This is a correlation: large tasks also keep workers alive longer. The design
consequence holds either way: a large change needs many agent-hours on one target, and a
structure that ends every worker after minutes cannot produce it, whatever the goal says.
Magnitude means outcomes (classes closed, owners deleted, callers migrated, capabilities
working end to end); lines are a signal, never a target.

**Agents act on what is near and concrete.** After hours of work and several compactions,
an agent's context holds the system prompt, AGENTS.md, its own recent messages, inherited
history and the latest tool output. A long goal, a skill read hours ago and a stance
paragraph lose force first. Procedures beat adjectives: an agent executes "record unknown
as unknown" exactly and treats "be ambitious" as tone.

In every high-output session measured, the operator was the control loop. They re-stated
the goal, set the scope and rejected small work (13–43 steering messages per session). Even
those sessions were less ambitious than intended. An unattended effort needs a structure
that does the operator's job: re-present the goal, propose large scope, reject small
returns and keep owners working.

## 2. Design rules

| # | Rule | Counters |
|---|---|---|
| D1 | **Ambition is a destination, not an adverb.** Name an end state that is large on its face, so small work is visibly unfinished ("Device Studio renders scenes, models and motion end to end", "the 129 friction findings are closed by class-level mechanisms"). | M6 |
| D2 | **The agent ends units, never the goal.** Closure, "complete" and "nothing left" are decided by a fresh challenger run with the fixed brief in §2a; the agent that wants to close does not write the question. Blocked means one named human-owned decision, routed while other work continues. No-change units do not count toward a count destination. | M3, M4 |
| D3 | **Kernel, stance message, charter.** The native goal is a kernel of 600 characters or fewer: destination, a stance excerpt in the operator's words, the charter path, "one unit per turn". The stance message is an ordinary user message of 3,500 characters or fewer, sent before the goal: the operator's full stance verbatim, the destination with its scale, and "re-read <charter> at every unit start and after compaction". Codex keeps both through compaction while fewer than eight newer user messages follow the stance message (§4a); a long goal it does not keep at all. In an Agent Manager run the stance message opens the task prompt. The charter file holds the rest and is re-read at every unit start and after every compaction. | M1, M13, M15 |
| D4 | **One unit per turn.** Under a native goal, the agent ends its turn after each unit's record, so Codex re-injects the objective and the completion audit before the next unit. Without a native goal, ending the turn ends the effort: start the effort as a native goal or an Agent Manager run, or do not end the turn. A scheduler's unit is one segment return; its turn spans the wait, which is safe only because it does no domain work and its context barely grows (not yet measured). | M1 |
| D5 | **Structure for ownership time.** A thin scheduler at the root; segment owners that own a coherent outcome for hours and are re-tasked with `followup_task` while productive; workers sized to an outcome, not to a step. Short or cheap workers are for lookups and reviews only (§3). | M8 |
| D6 | **Fresh briefs from the charter.** Every brief is generated from the charter and carries the stance verbatim. Use `fork_turns=none`; a full-history fork inherits the parent's drift, most of its context window and its model. | M9 |
| D7 | **Curated precedent.** Show workers pinned exemplars of a good unit and the recent output as numbers against the destination. Keep no-change verdicts in a separate list, out of the narrative a new worker wakes into. | M2 |
| D8 | **Scope is set by someone who does not pay for it.** An architect pass proposes large targets; an ambition review rejects plans as too small; executors then deliver (fixed briefs in §2b). The executor may enlarge scope; it may not shrink it without the reviewer. | M7, M19 |
| D9 | **Tools do the bookkeeping.** Identities, allowances, worker state and usage come from owner commands and the analyzer, not from supervisor prose. The supervisor's job is architect and driver: keep every slot staffed with an owner, re-task, challenge small returns. Idle slots while outcomes are open are a defect; polling is not work. | M11 |
| D10 | **Measure with tools; use tripwires, not floors.** Track worker active time, outcomes per unit, added and removed lines separately, no-change streaks, slot use and time since the charter was read (`effort_anatomy.py drift`). A tripwire triggers re-orientation (re-read charter, architect pass), never a line quota; a reviewer who rejects returns by line count teaches padding. | M2, M5 |
| D11 | **Cluster before assigning.** Group queue findings into classes and assign the class; write each verdict back to its finding so the queue drains; time-share between focus areas instead of strict priority. | M12 |
| D12 | **Name the winning skill.** When an effort reads several skills, the effort's method skill wins over another skill's stop or threshold rule. Say so in the charter. | M13 |
| D13 | **Intent is quoted, never paraphrased.** The operator's words live verbatim in the charter (a research topic's `MISSION.md`) and travel verbatim into every brief. Restatements may enlarge them, never shrink them; a commission is generated from the charter, not from the last checkpoint. | M18 |
| D14 | **Both failure directions count; rigor follows reversibility.** Under-delivery is reported as plainly as overclaiming, with a cause; honesty rules govern claims, never scope. Validate building with owner tests and a running showcase; reserve controlled trials for choices that are expensive to reverse. Every charter carries a denominator (scorecard, workstream table) so small units are visibly small. | M19–M22 |

### 2a. The challenger brief

Use this brief verbatim whenever a unit, study or goal is proposed for closure, completion or
"nothing material". Spawn the challenger fresh (`fork_turns=none`) with the original commission
or charter, the destination, and the evidence the closer cites. The closer may add evidence; it
may not change the question.

```text
You are a challenger. Read <charter or commission path> and the cited evidence. The destination is: <destination>. List every consequential route still open toward it, ranked by value: for each, what it would deliver (outcome, owners, scale), the first concrete step, and why it is or is not permitted under the charter. Include the 10× version of the remaining work toward the destination. A route counts only while the destination is not yet true; list larger work beyond it separately as proposed next destinations, which do not block closure. Do not judge effort already spent. Return the list, or "NONE" only if no permitted consequential route toward the destination exists, with the reason for each route you rejected.
```

Closure proceeds only when the list is `NONE`. Otherwise the top permitted route becomes the
next unit.

### 2b. The architect and ambition-review briefs

D8 separates whoever sets scope from whoever pays for it. Use these two briefs verbatim. Run the
architect pass when an effort or commission starts, when every open item is small, when a unit
moved no scorecard dimension, or when a `drift` tripwire fires. Run the ambition review on the
architect's output and on any plan an executor proposes before staffing it. Spawn both fresh
(`fork_turns=none`), on the strongest available model. Neither one executes anything.

```text
You are the architect for <effort or topic>. Read <charter or MISSION.md> and <topic README or ledger>; the operator's words in it are the intent. Ignore how large past units were. Propose the workstreams that would make the end state true: for each, the destination, the scorecard dimensions it moves, the owners and files it will change, the agent-hours it needs (most should be 20–200), and the showcase that proves it landed. Include work that deletes, consolidates or migrates as well as work that adds, and anything the operator asked for that no current item covers. Rank them by value toward the end state. Return the table and one line on the largest risk.
```

```text
You are the ambition reviewer for <effort or topic>. Read <charter or MISSION.md>; the operator's words in it are the bar. Review <the proposed table or plan>. For each item, answer: if this lands exactly as written, how much of the end state is true, and what would the operator say? Return REJECT when an item is a probe, a datapoint, a screen, a polish of one case, a "decision-ready" report where building is authorized, or under about 20 agent-hours without being a task inside a larger item; when the items together leave a dimension or a stated operator wish untouched; or when the work stops short of the showcase. Give the larger replacement for each rejected item. Return ACCEPT only when landing the whole table would make the operator say the end state was pursued at full scale.
```

The executor may enlarge an accepted item. Shrinking or splitting it into smaller rows needs
the ambition reviewer again.

## 3. Effort structures

### Three tiers

| Tier | Role | Lifetime | Holds |
|---|---|---|---|
| Root scheduler | Reads the ledger, spawns or re-tasks segment owners from the charter template, has a fresh challenger audit each return against the destination and the measured magnitude, records, ends its turn. Does no domain work, so its context stays small. | The whole effort | The native goal (kernel), the ledger |
| Segment owner | Owns one coherent outcome: a study's next epoch, ten hunt rounds, one migration. Spawns its own workers. Re-reads the charter at each round start. Does not return before its end state or time box. | A time box of at least 2 hours unless the outcome finishes sooner; up to about 8 hours; re-tasked while productive | A brief generated from the charter |
| Worker | Implements, hunts or reviews one outcome sized to the work, not to a step. | Matches the outcome | Its brief |

Sequential segments of about 2–6 hours stay inside the healthy window measured on the
ecosystem hunt (rounds 1–29: about 6 hours and 7 compactions before decay). The scheduler
stays healthy because each segment return ends a turn (D4).

### Slots

Codex runs a fixed number of concurrent threads per session, and every waiting ancestor
holds one. With the default of 4, a scheduler and one segment owner leave two worker slots;
Vrooli's config raises the limit to 8 (§4). Run at most slots − 2 segment owners at once, so
each has room for a worker or reviewer; rotate owners between outcomes at checkpoints instead
of shrinking them. When a spawn fails with `agent thread limit reached`, do that work in the
current thread or wait for a slot; do not retry the spawn in a loop.

### Harnesses

Shapes with `followup_task`, `fork_turns` and multi-level trees are Codex shapes. In Claude
Code, sub-agents spawn through the Agent tool only when their agent type has it (the
`implementer`, `lookup`, `researcher` and `reviewer` definitions do not), so use two tiers:
the session is the scheduler and owner, its workers do not delegate, and re-tasking happens
by message to a running agent. Verify deeper nesting before relying on it.

Claude Code has a hook that fires after compaction; Codex has none. `resource-claude-code ensure`
installs a `SessionStart` hook with the `compact` matcher, which prints a reminder to re-read the
active charter or brief before the next action (`resource-claude-code effort compact-context`).

### Goal-bearing children

`spawn_agent` cannot give a child a native goal. Agent Manager can: a child run gets its own
session, slots, native `/goal` (from `until`) and turn re-injection. The contract is
[effort supervision, "Nested runs"](../../scenarios/agent-manager/docs/reference/effort-supervision.md#nested-runs):

| Need | Command |
|---|---|
| Start a goal-bearing owner | `agent-manager run create --task-id <t> --profile-id <p> --parent-run-id <own run> --until-file <kernel> --charter <abs charter path>` |
| Wait for it without polling | `agent-manager run wait --children <own run> [--since <observed_at>]`, or `run wait <child>`; exit 0 settled, 5 timed out, 6 Agent Manager unreachable |
| See the tree | `agent-manager run tree <root> [--json]`; `orphaned` marks a live child whose parent ended (never stopped automatically) |

Every child records `depth`. Children deeper than `runExecution.maxChildDepth` (default 3; 0
disables Agent Manager children) are refused with `POLICY_CHILD_DEPTH_EXCEEDED`. Each child's
prompt starts with a short lineage header: its run, depth and maximum, parent, the charter to
re-read at every unit start and after compaction, and whether it may start children.

The root's owner window bounds the whole tree's authority. Start a long effort with an owner
credential that covers its whole budget, and watch `authority=` in `run tree`. A resumed
failed child keeps its parent, depth and charter.

Only a run Agent Manager launched can be a parent. An attached or imported session (a Web
Console or terminal session you started by hand) cannot. To use this path, start the
scheduler itself as an Agent Manager root run with `--until-file` and `--charter`.

**Agent Manager is optional, never required.** On exit 6, a refusal, a depth-cap error, or
any Agent Manager outage, do the same work with native sub-agents and the same brief: Codex
`spawn_agent` with `fork_turns=none`, or the Claude Code Agent tool. Record the switch in the
ledger, and do not stop or wait for Agent Manager to recover. Prefer Agent Manager children
for segments longer than a few hours that need their own goal persistence. Native sub-agents
are enough for shorter work.

### Cost

Per-request cost grows with context length only until the first compaction; after that it is
bounded by the compaction window. A long-lived owner re-tasked on the same outcome reuses its
relevant context. A fresh worker is cheaper per request but repays orientation each time. Choose
by outcome: one coherent outcome, one owner for its whole length; a new outcome, a fresh owner
with a curated brief.

## 4. Codex multi-agent settings

Verified with depth probes on Codex 0.166, 2026-11-06. The values are declared in
[`resources/codex/native-settings.json`](../../resources/codex/native-settings.json), which is
their one source. `resource-codex ensure` applies them: `vrooli setup` runs it for every
installed agent resource. `resource-codex settings plan|apply` shows and applies them on
demand. They go into `$CODEX_HOME/config.toml`, following the symlink from a Web Console
session home to `~/.codex/config.toml`.

Ownership is per key. Vrooli writes a key that is absent, or one that still holds the value
Vrooli last wrote (recorded in `.vrooli-native-settings-state.json`). A value the user chose is
reported as a conflict and left alone, unless the operator runs `settings apply --force`.
`settings decline <id>` stops management of a key on that machine. The mechanism lives in
`packages/agentharness` (`native_settings.go`) and is shared by every harness resource. To
change a value, edit the document and re-run `ensure`. Do not hand-edit the config.

`codex features list` reports `multi_agent_v2 … false` even when sessions run it. Read
`multi_agent_version` in a rollout's `session_meta` instead; Codex 0.166 `exec` sessions report
`v2` with no feature flag set.

| Setting | Default | Verified effect |
|---|---|---|
| `features.multi_agent_v2.max_concurrent_threads_per_session` | 4 | With 4, a chain stopped at level 3 (`collab spawn failed: agent thread limit reached`). With 8, a chain reached level 7. No depth limit was hit up to 7. |
| `features.multi_agent_v2.multi_agent_mode_hint_text` | "Any earlier instruction enabling proactive multi-agent delegation no longer applies. Do not spawn sub-agents unless the user or applicable AGENTS.md/skill instructions explicitly ask…" | Replaces the `<multi_agent_mode>` message for the root and every child. The message survives compaction. |
| `features.multi_agent_v2.subagent_developer_instructions` | none | Injected as a developer message into every child at every depth. |
| `agents.max_depth` | none reached at depth 7 | Unverified. `-c agents.max_depth=3` did not stop a chain, which reached depth 6 and stopped at the thread cap. The binary contains the message "Agent depth limit reached. Solve the task yourself." The thread cap is the working guard against runaway nesting. |

Other fields in the same table (`usage_hint_text`, `root_agent_usage_hint_text`,
`subagent_usage_hint_text`, wait timeouts, `preserve_fork_prefix`) are unverified. Verify a
field with a probe before relying on it and record the result here.

### 4a. What Codex keeps through compaction

Measured on Codex 0.166, 2026-11-06, from three separate samples: 377 compaction records in Web
Console homes (2026-11-03..06; 117 root threads, 260 children), 119 records from one parked Agent
Manager delivery orchestrator, and one probe. Compaction is remote: the summary is an encrypted `compaction` item, and the
rest of the replacement history is the developer messages plus a `retained_context` record.

| Item | Kept |
|---|---|
| Ordinary user messages | The 8 most recent, verbatim up to about 3,600 characters each. A longer message keeps about 1,800 characters from its start and its end, with `<truncated omitted_approx_tokens="N" />` in between (an 11,185-character message kept 3,598). Goal set and clear events (`<codex_internal_context source="user_goal">`, 137 retained) and Agent Manager wakes (`[wake] …`) are user messages and count toward the 8; goal-turn injections (`source="goal"`) do not (none retained). |
| A native goal's objective | Verbatim when short: across every October–November Codex rollout the longest kept objective was 669 characters and the shortest dropped one 711, so keep the kernel at 600 or fewer (Agent Manager enforces 600 in `runner.NativeGoalVerbatimLimit`); otherwise replaced by `User set the goal: [objective omitted; exceeds the evidence limit].` It returns in full only at the next goal-turn start. |
| A sub-agent's brief | A child has no user messages (all 260 child compactions retained none). Its `NEW_TASK` and later parent messages stay in the replacement history as `agent_message` items whose payload is encrypted, so how much of a long brief survives is unknown. |
| Assistant messages | The last 8, short. |
| Skill text and other tool output | Not kept, except through the encrypted summary. |
| `compact_prompt` | No effect observed. With a compaction prompt requiring a canary line, the agent could not report the canary after compaction. Codex 0.166 compacts remotely, and the setting appears to apply only to local compaction. One probe; re-verify after a Codex upgrade. |

So the operator's stance survives when it is in a user message of 3,500 characters or fewer with
the most important text first, and fewer than 8 newer user messages follow it. Goal set and
clear events and Agent Manager wakes count, so it lasts the whole run only when the run is neither
re-goaled nor woken often. An Agent Manager task prompt is a user
message too: when it is longer than about 3,600 characters, its lineage header and the first part
of the brief survive, so a brief puts the stance and destination first. A parked run that is woken
more than seven times loses its task prompt: in one delivery orchestrator, 111 of 119 compactions
kept eight wakes and nothing else. From then on only the charter re-read carries the stance. `retained_context` in each `compacted` record shows what survived
(`effort-diagnosis` `references/transcripts.md`).

## 5. Baselines (measured 2026-11-06)

Re-measure with `effort_anatomy.py` before citing these for another period. The figures come from
`scan` (with `--root` limited to the Web Console homes for the first row), then `summary`.
`load_scan` merges the several rollout files of one continued thread into one row.

| Measure | Value |
|---|---|
| Changed lines per active hour (all threads) | ~650 before 2026-10-29, ~515 after |
| Non-review sub-agents by active time: count, median changed lines, share ≥1,000. Web Console homes (1,307 threads) | <6 min: 446, 0, 0%. 6–30 min: 191, 94, 2%. 30 min–2 h: 51, 346, 22%. 2–8 h: 21, 2,143, 67%. >8 h: 11, 15,852, 91% |
| The same, all Codex homes including `~/.codex/sessions` (6,328 threads) | <6 min: 538, 0, 0%. 6–30 min: 199, 97, 2%. 30 min–2 h: 67, 308, 21%. 2–8 h: 23, 1,095, 52%. >8 h: 23, 4,280, 78% |
| Non-review sub-agent active time, split at 2026-10-29 | Web Console homes: 71 sub-agents and 307 active hours before (~4.3 h each), 649 and 109 after (~10 min each). All homes: 194 and 798 h before, 656 and 137 h after |
| Regime change 2026-10-29/30 (Codex 0.161 → 0.166 and AGENTS.md "delegate cheaply, fresh worker per outcome") | workers/day 1–26 → 84–325; median worker life 30 min–hours → 1–3 min; children starting with >100k inherited tokens 0% → 36–47%; worker output 70–148k → ~26k lines/week |
| Highest-output sessions (2026-10-17..21) | 35k–93k lines; 9–23 long-lived owners; 13–43 operator messages; one owner re-tasked 160 times and messaged 456 times by its parent and 372 times by peers |
| Native goals ending `blocked` | 36% of 484 |
| Goals ≥1,500 characters | 47% paused by the operator, 17% complete |
| Compactions that dropped the objective | 206 of 283 compaction records dated 2026-11-05 across all Web Console homes, counted per compaction line, including threads started on 11-04; every retained objective was ≤730 characters. `scan --since/--until` selects threads by start date, so its totals differ |
| Research supervisor 2026-11-05 | closed itself 1 h 15 min into a 20 h allowance; median worker life 6.8 min; ~4.3k code lines in 9.5 h; no depth-2 workers |

## 6. Open platform gaps

| Gap | Current workaround | Owner |
|---|---|---|
| Codex re-injects the objective only at goal-turn starts, never after compaction | D3 kernel and stance message (§4a) and D4 one unit per turn; `subagent_developer_instructions` tells children to re-read their brief | Codex (upstream) |
| Drift signals that need per-turn changed lines or narration (`shrinking-units`, `no-change-streak`, `closing-narration`) are not computed live; Claude Code runs get only `short-workers` | Agent Manager shows the other step-back triggers live in `run tree`, `effort board`, `effort compact` and the Efforts page. For the rest, run `effort_anatomy.py drift` (exit 3 on a tripwire) at each segment return | Agent Manager |
| `spawn_agent` cannot carry a native goal | Goal-bearing segments run as Agent Manager child runs (§3), with native sub-agents as the fallback | Codex (upstream); Agent Manager route exists |
| Agent Manager wakes are user messages, so a parked run woken eight times no longer retains its task prompt (§4a) | Runs created with `--charter` end every wake with "Re-read your charter <path> before acting." A run without a charter has no carrier: give long parked runs a charter | Agent Manager (resolved for chartered runs) |
| Attached sessions cannot parent Agent Manager runs | Start the scheduler as an Agent Manager root run, or delegate natively | Agent Manager |
