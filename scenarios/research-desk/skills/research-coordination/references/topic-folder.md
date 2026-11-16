# The topic folder

One common shape, kept small, that each topic extends as its work needs. Files are the
authority: every view works with grep alone, and services such as Source Ledger, Prompt Manager
and Program Runtime only make it faster. Start a topic from its mode's template
(`python3 -I scripts/new_topic.py`, see `templates/topics/README.md`).

## The shape

| Path | Answers | Who keeps it |
|---|---|---|
| `README.md` | What this is, where it stands now, the next meaningful proof, where things are. One screen. | Supervisor |
| `MISSION.md` | The operator's words, end state, scorecard, workstreams, boundaries ([missions](mission.md)) | Supervisor; operator words verbatim |
| `INBOX.md` | Messages for this topic's next agent: feedback, friction, answers, clarifications | Anyone appends; the next agent drains |
| `DECISIONS.md` | Open decisions this topic raised, each with its `work/<ID>/` packet | Whoever raised it; answered items move to `history/` |
| `TOOLS.md` | Which capabilities matter to the proofs, and the fallback when each is down | Supervisor |
| `work/` | Planned and active work as ready-to-run briefs, plus risks | Supervisor |
| `knowledge/` | What the topic has learned, organised for this topic's subject | Owners, curated by the supervisor |
| `feedback/` | Operator concerns that must persist, with each recurrence | Supervisor |
| `skills/`, `programs/` | Topic-owned methods and automation | Supervisor |
| `runs/<run>/` | One workstream in one commission: `CHECKPOINT.md` and evidence | That run's owner |
| `LEDGER.md` | Dated receipts, newest first. Nobody reads it whole. | Owners and supervisor |
| `history/` | Superseded pages, moved here instead of deleted | Supervisor |

Only `README.md` and `MISSION.md` are required. Add the rest when the work needs it; an empty
file is clutter. An owner reads its brief and `MISSION.md`, then what its card names; nothing
in this folder should be required reading at start beyond that.

## Knowledge

Organise knowledge around what the topic is trying to understand, not around runs. Runs are a
historical record; knowledge is the current understanding, rewritten as it improves.

- An engineering topic keeps the architecture: every scenario involved, current versus target,
  a bird's-eye Mermaid view and one drill-down page per area, with known debt and gaps. Plans
  are derived from the gap between current and target.
- A rules topic (debt reduction) keeps invariants per layer, state machines (explicit and
  implicit, where they live), a growing taxonomy and the tools for each kind.
- An evidence topic keeps a claims register: each claim, its sources, its confidence.
- An artifact topic keeps the taste register: judged versions, what the operator liked and why.

Mark consequential claims observed, inferred, proposed or unknown, and link observed ones to the
run or code that shows them. Update a page when evidence changes a boundary; prefer the smallest
map that tells an owner where to work and what an end-to-end proof must cover.

Search: `search-hub query "<question>" --type doc` (or Source Ledger) when it answers; `rg` over
the topic folder always works. Never copy whole files into another store.

## Plans and risks

`work/` makes the next commission cheap to start and hard to shrink. Each planned piece of work is
a folder `work/<ID>/BRIEF.md` written so a fresh owner can start it with one standard message
and that path, re-deriving nothing:

- the destination and the mission workstream it advances, with magnitude in agent-hours;
- why now, and the evidence and starting points (paths, commands, prior attempts);
- the scenarios or APIs it changes, and the showcase that proves it landed;
- what is decided and what the owner may change.

`work/README.md` lists the briefs in order with a one-word state (`planned`, `active`, `waiting`,
`landed`) and holds the risk list: each risk, its likelihood and impact in a word, the mitigation,
the owner, and the date it was last reviewed. The supervisor keeps plan and risks true at every
review; mission status cells stay authoritative for workstreams.

The standard message is: "You own `<topic>/work/<ID>`. Read `<abs path>/BRIEF.md` and the topic's
`MISSION.md`, then build toward the brief's showcase. [Operator stance, verbatim.]"

## Inbox

`INBOX.md` carries messages to an agent that is not running, so nobody steers a live goal by
chat. Write and drain it only with `scripts/inbox.py`, which stamps the clock:
`inbox.py append <folder> --from <level> --kind <kind> --body-file F`, `inbox.py list <folder>`,
`inbox.py drain <folder> --entry <stamp> --did "<what you did>"` (to `history/INBOX.md`). The
agent at that level drains at start and every review, acting on or filing each entry (senders,
kinds and routing: [levels](../../../../../docs/agent-system/LEVELS.md)).

## Feedback

Create a record in `feedback/<slug>.md` when an operator concern should outlive the
conversation. It holds the concern in the operator's words with date and source, the likely
owner, each recurrence as a dated line with evidence, the disposition (`untriaged`,
`investigating`, `addressed`, `verified`, `recurring`, `declined`), the next action and the exact
verification condition. A repeat never replaces the original: add the occurrence, mark it
`recurring`, re-check the earlier fix, and raise its priority. `addressed` is not `verified`.

## Tools

`TOOLS.md` lists only capabilities whose failure changes the work: per tool what it is used for,
the `Fallback:`, the `Monitoring scenario:` and known issues. Observations go to runs.
`scripts/topic_health.py` reads it and names each tool down or `unknown` with its fallback.

## Skills and programs

`skills/<name>/SKILL.md` holds a method this topic repeats (for example how to present UX
redesign decisions). `programs/<name>.json` with `<name>.py` holds automation that turns many
tool calls into one (contract `name` = its id; `research/run-freshness` is a small model).
Refer to them as `research/<topic>/<name>`; Prompt Manager and Program Runtime resolve that
prefix, and the file path always works. Mark a new skill experimental until a
real task used it. The supervisor creates and maintains them; owners report friction.
