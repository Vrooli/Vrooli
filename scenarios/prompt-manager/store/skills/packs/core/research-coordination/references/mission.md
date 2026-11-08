# Topic missions

Read when creating, realigning or commissioning work on a research topic, and at the start of
every unit of a sustained commission. A topic is a **programme**: a durable mission that
successive commissions advance. A commission is a staffing window on that mission, not a new
question. A run is an evidence container, not a unit of progress.

## Why missions exist

Research topics drifted small for reasons wording cannot fix (`effort-diagnosis` M17–M23):

- **The unit of output was a finding.** Topics, runs and closure rules were built for knowledge
  production, so 240 runs across 21 topics produced receipts ("this does not establish…") rather
  than capabilities. Image Tools spent its windows on JPEG metadata branches; debt reduction ran 48
  runs with no landed simplification named in its summary.
- **Agents rewrote the mission.** Each prepared commission paraphrased the last checkpoint, so the
  operator's intent eroded with every handoff ("massively improve the infra" became "a
  decision-ready foundation… a narrower recommendation is useful").
- **Nothing showed the gap.** With no target state, a fixed edge case looked like progress.

A mission file fixes all three: the operator's words are kept verbatim, the end state is large on
its face, and a scorecard plus a workstream table make small work visibly small.

## The mission file

Each topic has `topics/<topic>/MISSION.md`. It is the charter that owners re-read at every unit
start and after every compaction (`LONG_EFFORT_DESIGN.md` D3). Validate it with
`python3 scripts/mission_check.py <research-home>`; the same tool renders the portfolio table.

```markdown
# <Topic title> — mission

Mode: <build | propose-then-build | gated-effect | artifact | evidence>
Last architect pass: <YYYY-MM-DD> · Last ambition review: <YYYY-MM-DD> <ACCEPT|REJECT→revised>

## Operator intent (verbatim)

> "<the operator's exact words>" — <YYYY-MM-DD>, <source>

## End state

<3–6 sentences: what is true when the mission is done. Large on its face.>

## Scorecard

| Dimension | Today | Target |
|---|---|---|
| <dimension> | <one line, linked evidence> | <one line> |

## Workstreams

| ID | Destination | Dimensions | Magnitude | Showcase | Status |
|---|---|---|---|---|---|
| <T1> | <end state of this workstream> | <dimension names> | <agent-hours · owners> | <the demo that proves it landed> | <open \| active \| landed \| proposed \| approved \| declined> <link> |

## Approval flow

<What agents do directly, what becomes a proposal, who decides, what approval converts into.>

## Real boundaries

- <human-owned limit only>

## Exemplar

<Link to the best unit so far and one line on why; or a described target unit.>

## Not this

- <a trap this topic's history shows>
```

### Rules for each section

| Section | Rule |
|---|---|
| Operator intent | Quote only. Append new operator statements with date and source; never paraphrase, summarize, shorten or reorder them. An agent's restatement belongs in End state, never here. Search transcripts and topic feedback for the operator's own words (`search-hub query "<topic>" --type record,doc`; `agent-manager` conversation search). |
| End state | Write the mission complete, not the next step. It must not contain "decision-ready", "evidence for a decision", "narrower", "foundation for later", "if justified" or other self-shrinking qualifiers unless the mode is `evidence`. |
| Scorecard | 5–8 dimensions that together cover the operator intent. "Today" is current measured state, read from code, run evidence or a command, never inferred from file or package names (the first Image Tools draft called deliberate layers duplicates because their names looked alike); "Target" is the end state for that dimension. A dimension no workstream moves is a gap for the next architect pass. |
| Workstreams | Each is a destination of roughly 20–200 agent-hours that moves at least one dimension, names the owners it will change, and names a showcase. Keep at least three open or active. Before writing a row, recall what already landed (topic ledger, hunt and delivery records) and which decisions the operator already answered; a row that re-opens landed work or queues an answered decision is a defect (two engineering missions did both on their first draft). Smaller items are tasks inside a workstream, not rows. A workstream finishes by landing (its showcase runs, with validation), being declined by the operator, or being merged into another; "no change needed" is a challenger verdict (`LONG_EFFORT_DESIGN.md` §2a), not a status. |
| Approval flow | Taken from the mode table below, then specialised. |
| Real boundaries | Human-owned limits only: money, external contact, credentials, legal or rights acceptance, irreversible host or device actions, operator taste. Six bullets or fewer. Everything not listed is authorized within AGENTS.md. Evidence standards, validation scope and owner etiquette are not boundaries. |
| Exemplar | One unit that looks like what the operator wants more of. Pin it; do not rotate in the latest unit. |
| Not this | Two to five traps taken from the topic's history, named concretely. |

## Modes

| Mode | What agents build directly | What needs the operator | What approval converts into |
|---|---|---|---|
| `build` | Everything inside the repository: owner changes, migrations, deletions, new operations, model upgrades, new modules, cross-scenario refactors, tests and docs. | Only the real boundaries. | — |
| `propose-then-build` | Exploration lane: sandboxes, task-owned prototypes, external experiments, demos, research outside Vrooli, and ordinary improvement of existing capabilities. | A new maintained capability, scenario or significant extension of Vrooli: send a proposal packet ([operator attention](operator-attention.md#proposal-packets)). | An approved proposal becomes a `build` workstream in this file with the same magnitude rules and a showcase. It is staffed in the next unit, not queued behind exploration. |
| `gated-effect` | Everything up to the gated effect: contracts, fixtures, simulators, dry runs, rehearsals and the complete path with the effect stubbed. | The named effect (funding, broadcast, contacting a person, operating a physical device). | The effect, executed by its owner with the prepared runbook. |
| `artifact` | Complete artifacts, alternatives, and the production capability that makes them (pipelines, tools, evaluators). | Taste: batch judgments over alternatives, with each judgment bound to an exact version. | A judged direction the next workstream builds on. |
| `evidence` | Datasets, pipelines, detectors and analyses at full scale. | Publication, external use or accusations. | — |

A pending proposal or decision never idles a topic: the other workstreams keep their owners.

## The architect pass

Run it when a mission file is created, when a commission starts, when every open workstream is
smaller than about 20 agent-hours, when the scorecard has not moved in a commission, or when the
`drift` tripwire fires. Spawn a fresh agent (`fork_turns=none`, strongest available model) with
the architect brief in `LONG_EFFORT_DESIGN.md` §2b, the mission file and the topic README. It
proposes; it does not execute. Then run the ambition review (§2b) on the proposed table. Merge
the accepted result into Workstreams and record both dates on the mode line. The executor may
enlarge a workstream; shrinking or splitting one into smaller rows requires the reviewer.

## Commission bounds

A commission picks which workstreams to staff and for how long; it never shrinks what the
mission's mode authorizes. Do not write single-candidate limits, "no production edits", "no
downloads", "do not turn a prototype into a scenario", token caps far below the work, or launch
ceilings into a commission for a `build` topic. Each of those ended a real study before it
started: the local-model study allowed "one existing model/runtime, one baseline and at most three
settings candidates" with "no production edits", and a desktop pilot capped at 40,000 tokens
closed after 18 minutes with zero launches.

## Capacity and gates

A capacity warning, host-pressure reading or admission refusal is a scheduling fact, not a
result. Wait through the owner's wait command, schedule the job for an idle window, move it to a
fleet node, or work another task in the workstream meanwhile. Never close a run as "investigate"
or "blocked" on capacity. Four desktop cohorts, the artist-world study and the freelance
baseline all closed that way.

## Operator suggestions

Every suggestion the operator makes ("try yt-dlp for the references", "analysis before
generation") becomes a named task inside a workstream within the next unit, with its result
recorded. An ordering preference sequences one workstream; it never stalls the others.

## Sizing rigor to the decision

In `build` and `propose-then-build`, validation is owner tests, the relevant Test Genie phases
and the showcase running end to end. Use a controlled comparison (`local-experimentation`) only
to choose between candidates when the choice is expensive to reverse, such as a default model or
a storage format. Do not build a frozen evaluator for a choice an engineer can make by reading
two outputs. State limits once per run, in one "Limits" line; a paragraph of disclaimers per
result is a defect.

## Reporting both failure directions

Overclaiming and under-delivering are both reportable defects. Every checkpoint states which
scorecard dimensions moved, which workstreams advanced, and the measured change
(`effort_anatomy.py tree`). A commission that moves no dimension is reported as under-delivery
with a cause from `effort-diagnosis`, not as a careful result. Honesty rules govern what is
claimed; they never set scope.

## The topic README

The README is the topic's front page, not its log:

1. One paragraph and a link to `MISSION.md`.
2. The scorecard snapshot, or a link to it.
3. Current findings that change decisions, curated, about one screen.
4. Links to `LEDGER.md` (dated per-run receipts, newest first) and `runs/`.

Per-run "cohort update" sections go to `LEDGER.md`, never the README. A new owner reads the
mission and the exemplar, not the latest receipts (`LONG_EFFORT_DESIGN.md` D7).

## Runs

One run per workstream per commission. Create a new run folder for a new commission or a new
workstream, not for each check, screen or poll. Hourly feed screens and repeated probes append to
that run's checkpoint.
