---
name: effort-diagnosis
description: "Diagnose why a goal, supervisor, research cohort, hunt or other long-running agent effort under-delivers, starts too small, or decays over time. Measure the effort from its native transcripts (goal injections, compactions, sub-agent trees, worker lifetime, inherited context, changed lines, narrowing narration), test the cataloged mechanisms against their signatures, and route fixes to the goal text, the skills, AGENTS.md or the platform. Use when an operator says an effort is not bold enough, rounds are shrinking, workers do little, or a goal stopped early."
license: CC-BY-4.0
metadata:
  kind: skill
  schemaVersion: 1
  modes: [practice]
  tags: [goal, long-running, supervision, sub-agent, decay, diagnosis, orchestration, research]
  icon: activity
  status: active
  revision: 4
  createdAt: "2026-11-06T00:00:00Z"
  updatedAt: "2026-11-07T02:30:00Z"
  requires:
    scenarios: [prompt-manager, vrooli-memory]
    commands: [prompt-manager skill read, prompt-manager skill show, vrooli-memory journal note]
  origin:
    kind: authored
---

## Practice focus: Effort diagnosis

Find the mechanism that made a long agent effort smaller than the operator intended, prove it
from the effort's own transcripts, and fix it where it lives. A diagnosis that blames
wording alone is usually wrong: measure the structure first.

Required reading:
- `path:docs/agent-system/LONG_EFFORT_DESIGN.md` — the design rules, the effort structures,
  verified harness settings and the measured baselines.
- `references/mechanisms.md` — the mechanism catalog (M1–M23) with signatures and fix owners.
- `references/transcripts.md` — where transcripts, goals and records live and how to read them.

Paths under `references/` and `scripts/` are relative to this skill's directory
(`prompt-manager skill show effort-diagnosis --json` reports it as `skillDir`).

**Scope.** In: Codex `/goal` runs, Web Console Codex sessions, Claude Code sessions, Agent
Manager runs, sub-agent trees, research cohorts, ecosystem hunts, delivery epochs and any
effort that runs for hours. Out: one failed command (`scientific-debugging`), a scenario
defect (`scenario-work-ladder`), cost accounting alone (`agent-manager usage`), and
supervising a live delivery effort (`large-effort-supervision`). This skill reads
transcripts and records; it changes goals, skills and code only within the operator's
current request.

### 1. Frame the complaint

Write the operator's complaint in their words. Name the effort, its goal text, its start
and end, and what the operator expected (for example "thousands of changed lines across six
studies"). Classify the shape: **never started big**, **started big and decayed**, **stopped
early**, or **busy but hollow**. Each shape points at different mechanisms first:

| Shape | Check first |
|---|---|
| Never started big | M6 adverb ambition, M7 scope chosen by the cost-bearer, M8 short-worker cap, M11 bookkeeping; for research topics M17 epistemic framing, M18 mission erosion, M22 no denominator |
| Started big and decayed | M1 objective loss, M2 precedent ratchet, M13 frame drift, M10 delegation collapse |
| Stopped early | M3 exit ramps, M4 count destination |
| Busy but hollow | M4, M5 asymmetric scrutiny, M12 evidence grain, M14 shared-tree avoidance, M19 tractability bias, M20 rigor theater, M21 correction residue, M23 missing approval loop |

### 2. Measure

Run the analyzer. It is read-only and uses the Python standard library.

```bash
A=<skillDir>/scripts/effort_anatomy.py
python3 $A goals --contains "<distinctive goal words>" --show 5     # find the goal and its thread
python3 $A scan --root "<home>/sessions" --out <scratch>/scan.jsonl # one home, or omit --root for all
python3 $A tree <scratch>/scan.jsonl <root-thread-id> --list        # the sub-agent tree
python3 $A timeline <root-rollout.jsonl>                            # injections, compactions, spawns, hourly lines, narration
python3 $A drift <rollout.jsonl> --scan <scratch>/scan.jsonl --turns # per-turn units and tripwires; exit 3 when one fires
python3 $A summary <scratch>/scan.jsonl                             # lifetime table, per-day regime, top threads
python3 $A claude ~/.claude/projects/<project>/<session>.jsonl --events   # Claude Code sessions
```

`drift` is also the live tripwire. A scheduler runs it on each owner at each segment return.
It fires on these patterns:
- objective loss after a compaction (M1);
- a root-thread compaction that kept no user message carrying the stance (M1). Pass
  `--stance-marker "<distinctive stance words>"`; without it, the check fires only when no
  operator message survived, and misses a stance evicted behind later steering;
- turns over 3 h with two or more compactions (M1);
- shrinking units (M2);
- three no-change turns in a row (M3);
- half or more workers active under 6 min (M8);
- most spawns inheriting the full history (M9);
- rising narrowing and stop narration.

A tripwire calls for re-orientation (re-read the charter, an architect or challenger pass). It
is never a line quota. A thread continued in a second rollout file (`<thread>_<id>.jsonl`, same
`session_meta` id) is one thread. `drift` reads all of its files, and `load_scan` merges their
rows.

Lifetimes use active time (gaps over 10 minutes are idle); `hours` is the wall-clock span.
Lines are added plus removed from file-edit items, with a fallback to `apply_patch` text for
older rollouts; judge outcomes, and treat lines as a signal. Use `scan --since/--until` for a
period comparison; it selects threads by start date. Keep scan files in your scratch
directory. Scratch directories do not survive days; journal the numbers you rely on.

Build one timeline table: goal injections, compactions (objective retained or omitted, operator messages kept),
turn boundaries, operator messages, spawn waves, changed lines per hour, and the first
appearance of narrowing narration. Read the narration examples in context; family counts
are signals, not verdicts.

### 3. Test the mechanisms

For each candidate mechanism in `references/mechanisms.md`, check its signature against the
measurements. Prefer natural experiments over correlation:

- **Re-injection points.** Compare behaviour in the 30 minutes before and after a goal
  injection or an operator message. A sharp change confirms M1 or M2.
- **Regime changes.** Compare periods before and after a change to AGENTS.md, a skill, the
  harness version or the model (`summary` per-day table). Name confounds.
- **Exemplars.** Compare the effort with the highest-output effort of a similar kind
  (`summary` top threads). Record what that effort had that this one lacked.
- **Counterfactual prey.** Before accepting M16 (real diminishing returns), list open
  high-value items the effort could have taken.

Rank mechanisms by evidence strength. For each, state what is measured, what is inferred,
and what is unverified. Do not stop at the first mechanism; efforts usually fail through
three or four at once.

### 4. Prescribe and route

Route each fix to the layer where the mechanism lives. One mechanism can need fixes in two
layers.

| Layer | Owner and route |
|---|---|
| This goal's text | Rewrite with `harness-goal-authoring` (kernel, destination with magnitude, stance, exits, one unit per turn). |
| A method skill (hunt, research, orchestration) | Edit the canonical `SKILL.md` in `scenarios/prompt-manager/store/skills/packs/<pack>/<id>/`; refresh native copies per `docs/reference/cli-commands.md` §"Editing projected skills". |
| Root instructions | `AGENTS.md` (also `CLAUDE.md` through its symlink). Independent review is required (`ecosystem-hunt` §8 step 6). |
| Design canon | `docs/agent-system/LONG_EFFORT_DESIGN.md`: new structures, verified settings, new baselines. |
| Harness configuration | Declared in `resources/<harness>/native-settings.json` and applied by `resource-<harness> ensure` (run by `vrooli setup`); change the document, not `~/.codex/config.toml`. Verify each new setting with a probe before declaring it. |
| Platform capability | The owning scenario: Agent Manager for runs and goal-bearing children, Web Console for native sessions. File with `report-bug` when outside the current request. |

Recommend structure before wording. A fix that only adds exhortation to a goal leaves M8,
M9, M10 and M11 in place.

### 5. Record

1. Add each new mechanism, signature or harness fact to `references/mechanisms.md` or
   `references/transcripts.md`, and each new baseline or verified setting to
   `docs/agent-system/LONG_EFFORT_DESIGN.md`. One home per fact.
2. Journal the diagnosis: `vrooli-memory journal note --kind work-record --body-file <file>
   --trigger "<complaint>" --approach "<measurements>" --evidence "<rollout paths>"
   --outcome "<mechanisms and fixes>"`.
3. Give the operator the ranked mechanisms, the measured evidence, the fixes applied or
   proposed per layer, and the experiment that would confirm the ranking.

### Output expectations

You may read any transcript, goals database or effort record, and you may run the analyzer
anywhere. You must not change a live goal, stop a run or edit another agent's work unless the
operator's current request authorizes it. You must separate measured facts from inferences,
cite the rollout path for every number, and keep unknown values unknown.

### Anti-patterns

| Anti-pattern | Why it fails | Instead |
|---|---|---|
| Blame the wording first | Structure caps change size; wording cannot lift the cap | Measure worker lifetime, tree shape and injections first |
| One-cause diagnosis | Efforts fail through several mechanisms at once | Test every catalog row that fits the shape |
| Hypothesis confirmed by a delegate | A sub-agent handed your hypothesis tends to return it | Give the delegate the data and the question, not the answer |
| Model blamed without a control | Model tier and structure changed together | Compare lines per active hour across models and periods |
| Prescription as more exhortation | Adds another adverb the agent will treat as tone | Change the destination, the structure or the exits |

### Troubleshooting & Edge Cases

| Symptom | Response |
|---|---|
| A rollout is tens of MB and a regex over the whole file hangs | Use the analyzer; it streams line by line. Do not use `grep -o '.{0,9000}'` patterns on rollouts. |
| `spawn_agent` messages are encrypted | Read the child rollout: its behaviour, task name and narration show what it was asked. |
| The compaction summary is encrypted | Read `retained_context` for what survived verbatim (`references/transcripts.md`), plus the objective marker and the behaviour after compaction; say the summary content is unknown. |
| Changed lines look too low | FileChange items miss shell-based writes. Check the child's command list for `sed -i`, heredocs or scripts and report the gap. |
| A forked child shows the parent's goal | Lines before `subagent_history_start_ordinal` are inherited; the analyzer excludes them. |
| The effort ran in Claude Code | Use the `claude` subcommand; subagents are under `<session>/subagents/`. |
| A Web Console home has no rollouts for the period | The session may have run under `~/.codex/sessions` or Agent Manager; use `goals --contains` to find its thread. |
