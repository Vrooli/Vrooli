# Topic missions

A topic is a programme. Its `MISSION.md` is the charter every owner re-reads at each checkpoint
and after compaction. It keeps the operator's words verbatim, states an end state large on its
face, and makes small work visibly small through a scorecard and a workstream table, because
topics drifted small: findings instead of capabilities, paraphrased smaller at every handoff.

Keep it under 15 KB; receipts, evidence and history go to `LEDGER.md` and run folders.
Check it with `python3 -I scripts/mission_check.py <research-home> [--topic T]`.

## The file

```markdown
# <Topic title> — mission

Mode: <build | propose-then-build | gated-effect | artifact | evidence>
Last architect pass: <YYYY-MM-DD> · Last ambition review: <YYYY-MM-DD> <ACCEPT|REJECT→revised>

## Operator intent (verbatim)

> "<the operator's exact words>" — <YYYY-MM-DD>, <source>

## End state

<3–6 sentences: what is true when the mission is done.>

## Scorecard

| Dimension | Today | Target |
|---|---|---|

## Workstreams

| ID | Destination | Dimensions | Magnitude | Showcase | Status |
|---|---|---|---|---|---|

## Approval flow

<What owners do directly, what becomes a proposal, who decides.>

## Real boundaries

- <human-owned limit only>

## Exemplar

<The best unit so far, and why.>

## Not this

- <a trap from this topic's history>
```

| Section | Rule |
|---|---|
| Operator intent | Quotes only, dated, with source. Append; never paraphrase, shorten or reorder. Restatements go in End state. |
| End state | The mission done, not the next step. No "decision-ready", "narrower", "foundation for later" or "if justified" unless the mode is `evidence`. |
| Scorecard | 5–8 dimensions that cover the intent. "Today" is measured from code, runs or a command, never inferred from names. A dimension no workstream moves is a gap. |
| Workstreams | Each 20–200 agent-hours, moves a dimension, names the owners it changes or creates (a new resource or scenario by name), and names a showcase proved on real targets. Keep at least three open or active; check what landed and what the operator answered first; smaller items are tasks inside a row. Status is one word and a date or link: `open`, `active` (only while a live commission or owner is on it), `waiting-operator`, `waiting-time`, `landed`, `proposed`, `approved`, `declined`; waiting rows do not count toward the three. Rows are briefed or active work; the rest wait in order under `## Later` (`- **ID** — outcome · N–M h`). A row's prerequisites are briefed rows ordered before it in `work/README.md`; nothing in the table waits on a Later line. |
| Approval flow | The mode's row below, specialised. |
| Real boundaries | Six bullets or fewer: money, external contact, credentials, legal or rights acceptance, irreversible host or device actions, operator taste. All else is authorized within AGENTS.md. |
| Exemplar | One pinned unit that looks like what the operator wants more of. |
| Not this | Two to five traps from the topic's history. |

A finished topic gets `Retired: <date> — <why; what would reopen it>` under `Mode:`. Its folder
stays; commissions refuse it.

## Modes

| Mode | Owners build directly | Needs the operator | Approval becomes |
|---|---|---|---|
| `build` | Everything in the repository: owner changes, migrations, deletions, new modules, cross-scenario refactors | Only real boundaries | — |
| `propose-then-build` | Sandboxes, prototypes, external experiments, improvements to existing capabilities | A new maintained capability or scenario, as a proposal packet ([operator attention](operator-attention.md#proposals)) | A `build` workstream at the proposed scale, staffed next unit |
| `gated-effect` | The whole path with the effect stubbed: contracts, simulators, rehearsals | The named effect | The effect, run by its owner from the prepared runbook |
| `artifact` | Finished artifacts, alternatives and the pipeline that makes them | Taste judgments bound to exact versions | A direction the next workstream builds on |
| `evidence` | Datasets, pipelines, detectors at full scale | Publication, external use, accusations | — |

A pending proposal or decision never idles a topic; the other workstreams keep their owners.

## The architect pass

Run it for a new mission, a stale pass at commission start, open workstreams all under 20 hours,
or a window that moved no dimension. A fresh agent gets the architect brief from `templates/review-briefs.md`, the mission and
the topic README, and proposes a table. Then a fresh ambition reviewer runs §2b on it. Merge the
accepted table and record both dates on the mode line. Anyone may enlarge a workstream;
shrinking or splitting one needs the reviewer.

## Commission bounds

A commission picks which workstreams to staff and for how long. It never shrinks what the mode
authorizes: no single-candidate limits, "no production edits", "no downloads" or token caps far
below the work for a `build` topic; each ended a real study before it began.

## Proving grounds

A capability is worth what it was proved on (the Freelance review passed one of nine deliveries
tested only on targets their builder wrote).

- The builder never writes its only exam.
- Build the proving ground as a capability: the real software as a resource, a scenario that
  serves graded levels, a corpus of real public material. It outlives the task, so name it as a
  resource or scenario (a `propose-then-build` topic proposes it).
- Prove on the real thing too: real public instances, real sizes, the tool the client named.
  Simulate only what a real boundary forbids; label the stand-in.
- Use the trade's mature open-source tool; a from-scratch script needs a reason.
- Work in categories: one listing is an example of a kind of work; the kind is the unit.

## Shapes of workstreams

- **List workstreams** (capabilities, kits, songs): one file beside the mission, one row per
  item with a state, a `Wanted` section at the top where anyone adds a line, assessed at each
  checkpoint. Rows are never deleted; a dropped item stays `declined` with its reason.
- **Reduction workstreams** name what they replace: which owner absorbs which duplicate, which
  callers migrate, which files go. The line count is the readout, never the plan.
- **Operator suggestions** become named tasks in a workstream within the next unit.
- **Showcases that wait** for the operator, elapsed time or hardware: file the one request, set
  `waiting-operator` or `waiting-time` with a resume date, take the next workstream. Do not
  polish it while it waits.

## Rigor in proportion

In `build` and `propose-then-build`, validation is owner tests, relevant Test Genie phases and the
showcase run end to end. Use a controlled comparison (`local-experimentation`) only for
a choice that is expensive to reverse, such as a default model or a storage format. State limits
once per run in one "Limits" line. Measurement never replaces the deliverable.

Every checkpoint says which dimensions moved and which workstreams advanced, measured.
Overclaiming and under-delivering are both defects; honesty governs claims, never scope.

## README and runs

The topic README is the front page: one paragraph and a link to the mission, the scorecard or a
link to it, the current findings that change decisions (about one screen), and links to
`LEDGER.md` and `runs/`. One run folder per workstream per commission; repeated checks append to
its `CHECKPOINT.md`.
