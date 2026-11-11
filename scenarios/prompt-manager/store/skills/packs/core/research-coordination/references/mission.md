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
| <T1> | <end state of this workstream> | <dimension names> | <agent-hours · owners> | <the demo that proves it landed> | <open \| active \| waiting-operator \| waiting-time \| landed \| proposed \| approved \| declined> <date or link> |

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
| Workstreams | Each is a destination of roughly 20–200 agent-hours that moves at least one dimension, names the owners it will change or create (a new resource or scenario is named as such, when the work needs a home no owner provides), and names a showcase. The showcase says what it is proved on: a proving ground Vrooli owns and real-world targets, never only a target its builder wrote ([proving grounds](#proving-grounds)). Keep at least three open or active. Before writing a row, recall what already landed (topic ledger, hunt and delivery records) and which decisions the operator already answered; a row that re-opens landed work or queues an answered decision is a defect (two engineering missions did both on their first draft). Smaller items are tasks inside a workstream, not rows. A workstream finishes by landing (its showcase runs, with validation), being declined by the operator, or being merged into another; "no change needed" is a challenger verdict (`LONG_EFFORT_DESIGN.md` §2a), not a status. The status cell is one word from the list and a date or link; evidence, verdict text and hashes go to the run folder. `waiting-operator` and `waiting-time` mark open work that only the operator or elapsed time can move ([showcases that wait](#showcases-that-wait)); they do not count toward the three. |
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
`drift` tripwire fires. Spawn a fresh agent (`fork_turns=none`, model and effort omitted; see docs/configuration/coding-agents.md "Choosing a model for a sub-agent") with
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

An owner stops every resource, service and model it starts when the job that needed it ends.
A written rule did not prevent the failure it describes: on 2026-11-09 an Image Tools owner
left a model it had started holding 7.5 GiB for nine hours, and its own GPU showcase stayed
blocked the whole time. So the check is mechanical, with `scripts/footprint.py`:

- **Baseline, once.** Before any owner starts, the scheduler records what is already running:
  `python3 -I scripts/footprint.py snapshot --out <cohort folder>/BASELINE.json`. It covers GPU
  processes, running resources, capacity claims, containers, user timers, the sandbox count
  and free disk. It refuses to overwrite a baseline.
- **Check.** `python3 -I scripts/footprint.py diff --baseline <cohort folder>/BASELINE.json
  --keep-file <cohort folder>/FOOTPRINT-KEEP.txt` prints only what is running now that was not
  running at the baseline, including a model loaded into a service that was already up, and
  exits 1 while a line remains. `--disk --watch <dir>` adds disk growth and takes about a
  minute. The script observes; it never stops anything.
- **When.** An owner runs the check before every return message, before it asks for GPU work,
  and whenever admission is refused. The scheduler runs it at every hourly review and once at
  the allowance end.
- **Whose line is it.** The host is shared, so a line is not proof the cohort started it. Each
  owner lists what it starts in its checkpoint under `Started` and stops only its own lines,
  when no job is using them. The scheduler matches lines to those lists, stops a line whose
  starter has ended, and writes a line nobody claims into the keep file, with a comment, so it
  stops being reported. Something meant to outlive the window (a report timer) goes in the
  keep file and in `STATE.md` under `Left running`. An agent may stop a resource it names
  explicitly; it never stops another owner's or agent's work.
- **`NOT CHECKED`.** A section the script could not read is not clean: read that source
  directly before a GPU decision.
- **Refused admission.** Run the check first: the usual holder is something this cohort started.
  Then read the remaining holders (`vrooli capacity list --active`, `nvidia-smi`), collect the
  jobs that need the GPU into one window, and retry in the same unit. Parked GPU work is
  retried at every checkpoint.

## Showcases that wait

Some showcases need something agents cannot produce: an operator's review receipt, a week of
elapsed reports, hardware that is not on this host. When the remaining routes of a workstream
all need that, file the one request (`Needs you`, [operator
attention](operator-attention.md#needs-you-first)) or schedule the elapsed check, set the
workstream's status to `waiting-operator` or `waiting-time` with the date it can resume, and
take the next open workstream in the mission. Do not polish the same case through further
revisions, and do not add reviews of it, while it waits.

## Reduction workstreams

A workstream that cuts code names the structures it replaces: which owner absorbs which
duplicate, which callers migrate, which files are deleted when they do. The net-line target is
the readout of that plan. A plan that is only a per-package line budget produces trimming:
Image Tools IT1 moved 18 lines in ten hours against a 3,058-line target while four reviews
rejected its plans. When a reduction and a capability workstream are both open, the mission's
ranking and the operator's stated priority order them, and reduction also lands through the
replacements that capability work makes possible.

## List workstreams

Some workstreams deliver a list: capabilities, kits, detectors, songs. Keep the list in one file
beside the mission (`CAPABILITIES.md`, or a name that fits), one row per item, with a state per
row and an order that is the order of work. Give it a `Wanted` section at the top where anyone,
the operator first, adds an item as one sentence at any time. The owner re-reads the file at
every checkpoint, assesses each new line in the unit that finds it (how it would be done, its
rights and hardware fit, its size), moves it into the table at the place the file names for new rows and builds in list order. The
mission row points at the file and keeps one status for the workstream; the file carries the
state of each item. Rows are never deleted: a dropped item stays as `declined` with its reason.
Without the file, a list workstream is one row marked `open`, nobody can see which items moved,
and a new item has nowhere to go: Image Tools IT4 named seven capabilities in one cell, and
after a 30-hour window four of them had no work and nothing recorded that.

## Proving grounds

A capability is worth what it was proved on. This matters most when the result will be used on
targets we do not control: a client's website, their WordPress site, their spreadsheets, a
vendor's record system. The Freelance operator review of 2026-11-10 passed one of nine
deliveries. Each had been tested only on a small target its own builder wrote (a 128-line
scraper against a three-record local page). The operator's judgement: "they all similarly did
not go nearly far enough", and the agent "didn't think of any way we can you know build or reuse
any of our infrastructure to help with this sort of task or you know any related tasks in the
future".

- **The builder does not write its only exam.** A target made for one deliverable, sized so that
  the deliverable passes, shows that the code runs and nothing else.
- **Build the proving ground as a capability.** Ask what environment would let any tool of this
  kind be tested from easy to very hard, with known correct answers: the real software running
  as a resource (n8n, WordPress, a record system), a scenario that serves graded levels (a
  website made to be scraped), a corpus of real public material. It outlives the task and serves
  other work, so the workstream names it as a resource or scenario, and a
  `propose-then-build` topic proposes it. A proving ground is part of the destination. It is
  not setup to be kept small.
- **Prove on the real thing as well.** Real public instances, real sizes, the tool the client
  named. Simulate only what a real boundary forbids (a client's accounts, paid services, private
  data), and label the stand-in.
- **Stand on mature foundations.** Find the established open-source tool the trade uses (a
  crawling framework, a documentation-site generator) and build on it. A from-scratch script
  needs a stated reason.
- **Work in categories.** One listing, ticket or sample is an example of a kind of work. The unit
  is the kind: tens of agent-hours each, kept in a list file ([list
  workstreams](#list-workstreams)) whose rows say what each item was proved on.
- **Show the work.** A review request shows the deliverable, its full source, what it was tested
  on, who made that target, how big it is, and the hours spent
  ([operator attention](operator-attention.md#review-requests)).

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
result is a defect. Measurement never stands in for the deliverable: by-hand
comparisons, timing ledgers, sealed evaluations and hash receipts around a few hundred lines of
code tested on a few rows mean the effort went into measuring. Build the stronger deliverable
and prove it on a real target ([proving grounds](#proving-grounds)).

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
