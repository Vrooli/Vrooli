# The topic folder

One common shape, kept small, that each topic extends as its work needs. Files are the authority (R4): every view works with grep alone; services only make it faster.

| Path | Answers | Who keeps it |
|---|---|---|
| `README.md` | What this is, where it stands, the next proof. One screen. | Manager |
| `MISSION.md` | Operator's words, end state, scorecard, workstreams, boundaries ([missions](mission.md)) | Manager; operator words verbatim |
| `INBOX.md` | Messages for the next agent: feedback, friction, answers | Anyone appends; the next agent drains |
| `DECISIONS.md` | Open decisions, each with its `work/<ID>/` packet | Whoever raised it; answered items move to `history/` |
| `TOOLS.md` | Capabilities the proofs need, each with a `Fallback:` | Manager |
| `work/` | Ready-to-run briefs plus risks | Manager |
| `knowledge/` | What the topic has learned, by subject | Owners, curated by the manager |
| `feedback/` | Operator concerns that must persist | Manager |
| `skills/`, `programs/` | Topic-owned methods and automation | Manager |
| `runs/<run>/` | One workstream in one commission: `CHECKPOINT.md`, evidence | That run's owner |
| `LEDGER.md` | Dated receipts, newest first; nobody reads it whole | Owners and manager |
| `history/` | Superseded pages, moved instead of deleted | Manager |

Only `README.md` and `MISSION.md` are required; an empty file is clutter. An owner reads its brief and `MISSION.md`, then what its card names.

## Knowledge

Organise knowledge around what the topic tries to understand, not around runs: an engineering topic keeps the architecture (current versus target, drill-down pages, known debt); a rules topic keeps invariants and state machines; an evidence topic keeps a claims register; an artifact topic keeps a taste register. Mark consequential claims observed, inferred, proposed or unknown, and link observed ones to the run or code that shows them. Search: `search-hub query "<question>"` when it answers; `rg` always works.

## Plans and risks

`work/<ID>/BRIEF.md` is a ready-to-run brief a fresh owner starts from alone: destination and magnitude, why now with starting points, what it changes and the showcase, what is decided. `work/README.md` lists the briefs in order with a one-word state (`planned`, `active`, `waiting`, `landed`), holds the risk list (likelihood, impact, mitigation, owner, a `Reviewed` date only when re-checked) and the standard owner message once: "You own `<topic>/work/<ID>`. Read `<abs path>/BRIEF.md` and the topic's `MISSION.md`, then build toward the brief's showcase. [Operator stance, verbatim.]" Mission status cells stay authoritative.

## Inbox

`INBOX.md` carries messages to an agent that is not running, so nobody steers a live goal by chat. Write and drain it only with `scripts/inbox.py` (`append`, `list`, `drain --entry <stamp> --did "<what you did>"`, which moves it to `history/INBOX.md`). The owning agent drains at start and every review. Senders and routing: [levels](../../../../../docs/agent-system/LEVELS.md).

## Feedback

`feedback/<slug>.md` keeps an operator concern in their words with date and source, the likely owner, each recurrence as a dated line, a disposition (`untriaged`, `investigating`, `addressed`, `verified`, `recurring`, `declined`) and the exact verification condition. A repeat adds the occurrence and raises priority; `addressed` is not `verified`.

## Tools, skills, programs

`TOOLS.md` lists only capabilities whose failure changes the work; `scripts/topic_health.py` names each one down or `unknown`. `skills/<name>/SKILL.md` holds a method the topic repeats; `programs/<name>.json` with `<name>.py` holds automation; refer to them as `research/<topic>/<name>`.
