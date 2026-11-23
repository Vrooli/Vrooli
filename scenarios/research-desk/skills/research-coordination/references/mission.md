# Topic missions

A topic is a programme. Its `MISSION.md` is the charter every owner re-reads at each checkpoint and after compaction. It keeps the operator's words verbatim, states an end state large on its face, and makes small work visibly small through a scorecard and a workstream table (R3, R7).

Keep it under 20 KB (`ceilings.mission_bytes`; a mission over it is flagged for trimming, never refused). History, receipts and evidence go to `history/`, `LEDGER.md` and run folders. Check with `python3 -I scripts/mission_check.py <research-home> [--topic T]`.

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
## Real boundaries
## Exemplar
## Not this
```

| Section | Rule |
|---|---|
| Operator intent | Quotes only, dated, with source. Append; never paraphrase or reorder. |
| End state | The mission done, not the next step. No "decision-ready", "narrower", "foundation for later" or "if justified" unless the mode is `evidence`. |
| Scorecard | 5-8 dimensions that cover the intent; "Today" measured from code, runs or a command. A dimension no workstream moves is a gap. |
| Workstreams | Each 20-200 agent-hours, moves a dimension, names a showcase proved on real targets. Keep at least two open or active. Status is one word and a date: `open`, `active` (only while a live commission or owner is on it), `waiting-operator`, `waiting-time resume <date>`, `landed`, `proposed`, `approved`, `declined`. An expired `resume` date is counted in the README's `Waits expired`. Unbriefed rows wait under `## Later` (`- **ID** — outcome · N–M h`). |
| Real boundaries | Six bullets or fewer: the bright lines in [RULES.md](../../../../../docs/agent-system/RULES.md) and operator taste. All else is authorized. |
| Exemplar, Not this | One pinned unit; two to five traps from the topic's history. |

A finished topic gets `Retired: <date> — <why; what would reopen it>` under `Mode:`; commissions refuse it, and its landed rows still count in `history.py`.

## Modes

| Mode | Owners build directly | Needs the operator | Approval becomes |
|---|---|---|---|
| `build` | Everything in the repository | Bright lines | |
| `propose-then-build` | Sandboxes, prototypes, improvements to existing capabilities | A new maintained capability, as a proposal ([operator attention](operator-attention.md#proposals)) | A `build` workstream |
| `gated-effect` | The whole path with the effect stubbed | The named effect | The effect, run from the prepared runbook |
| `artifact` | Finished artifacts and their pipeline | Taste judgments on exact versions | A direction |
| `evidence` | Datasets, pipelines, detectors at full scale | Publication, external use | |

A pending proposal or decision never idles a topic.

## The architect pass

Run it for a new mission, a stale pass at commission start, open workstreams all under 20 hours, or a window that moved no dimension: a fresh agent gets the architect brief (`templates/review-briefs.md`) and proposes a table; a fresh ambition reviewer reviews it; record both dates. Anyone may enlarge a workstream; shrinking or splitting one needs the reviewer. A commission never shrinks what the mode authorizes.

## Proving grounds

A capability is worth what it was proved on. The builder never writes its only exam: build the proving ground as a reusable capability (the real software as a resource, a scenario serving graded levels, a corpus of real public material), prove on the real thing too, use the trade's mature open-source tool, and work in categories. Simulate only what a bright line forbids, and label the stand-in.

## Shapes of work

- **List workstreams** keep one file beside the mission: a row per item with a state and a `Wanted` section at the top, assessed at each checkpoint. A dropped item stays `declined` with its reason.
- **Reduction workstreams** name what they replace: which owner absorbs which duplicate, which callers migrate, which files go.
- A showcase that waits for the operator, time or hardware files one request, sets `waiting-operator` or `waiting-time resume <date>`, and the owner takes the next workstream.

Validation in `build` and `propose-then-build` is owner tests, relevant Test Genie phases and the showcase end to end; a controlled comparison (`local-experimentation`) only for a choice expensive to reverse. One "Limits" line per run. Every checkpoint says which dimensions moved, measured.

The topic README is the front page: one paragraph, a link to the mission, the findings that change decisions, links to `LEDGER.md` and `runs/`. One run folder per workstream per commission.
