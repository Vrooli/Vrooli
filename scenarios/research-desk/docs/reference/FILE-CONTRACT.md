# File Contract

The contract between the desk and the research skills, usable with a text editor and no desk.
Research home: `$VROOLI_RESEARCH_HOME`, else `~/.vrooli/research` (the shared
`api-core/researchhome` resolver; Prompt Manager and Program Runtime read the same).

## 1. Writers

| File | Only writer |
|---|---|
| `cohorts/<id>/{README.md,STANCE-MESSAGE.txt,KERNEL.txt,PACKET.json}` | `packet.py build` (run by the desk or an agent); `enable-steering` adds missing operator and steward authority instructions to an older charter |
| `cohorts/<id>/OPERATOR.md`, `reviews/<id>/VERDICTS.md` | the desk for the operator, or the operator by hand; append-only |
| `cohorts/<id>/STEWARD.md` | `research-desk feedback` from a verified agent run; dated and attributed; append-only |
| `cohorts/<id>/LAUNCH.json` | the desk |
| `cohorts/<id>/{STATE.md,LEDGER.md,BASELINE.json}` | the scheduler (`BASELINE.json` may be seeded once by `packet.py`) |
| `topics/<t>/MISSION.md`, `topics/<t>/DECISIONS.md`, `ATTENTION.md` | supervisors, the level's agents and the operator; never the desk |
| `topics/<t>/INBOX.md`, `INBOX.md` (the supervisor's) | anyone appends (the desk: every answer to a decision, §5); the level's next agent drains it to `history/INBOX.md` |
| `topics/<t>/runs/<cohort>/CHECKPOINT.md` | the owner |
| `reviews/<id>/{REVIEW.md,REVIEW.html,proposals/}` | the `research-review` agent (the operator saves the exported feedback JSON beside `REVIEW.html`) |

Only the live instance writes, holding an exclusive `flock` on
`~/.vrooli/state/vrooli/research-desk/instance.lock` (retried while a restart overlaps the old
process; reads never take it). JSON is written to a temp file and renamed; a Markdown append
is one `O_APPEND` write. The desk's own state, in that folder outside the research home:
`courier.json` (the operator's courier switches, `{on, at}`), `first-sightings.json` (when
the courier first found each needs-you item), `looks/<commission|topic>-<id>.json` (the
operator's last look), `launch-tests/<UTC stamp>.json` (one per `launch-test`) and
`retention/` (the shared retention sweep's receipt; the desk sets no budgets). Deleting the folder resets "since" and the sightings and
returns the switches to their defaults.

The desk serves the home read-only at GET `route[example]:/api/v1/home/<path>` (no hidden
parts, symbolic links only inside the home), so `REVIEW.html` and its relative media links
render the same from disk and from the desk.

Topic `ListTopics` also exposes links to optional `CURRENT.md`, `TOOLS.md` and regular
files under `topics/<t>/feedback/`. The files remain topic-owned and are never parsed or
copied into Desk state. Their absence is normal; unreadable and bounded/truncated feedback
listings remain explicit evidence. The existing route is the direct-file fallback.

Skills and programs live as files and resolve by prefix id, read fresh on every call:
`topics/<t>/skills/<name>/SKILL.md` is `research/<t>/<name>` and `skills/<name>/SKILL.md` is the
supervisor's `research/<name>` (`prompt-manager skill read <id>`); `programs/<name>.json` with
`<name>.py`, at either level, is the same id (`program-runtime library run <id>`), and the
contract's `name` must equal that id. Both CLIs read or run the files themselves when their
service is down. `ListTopics` reports each topic's and the home's skills and programs with
their files and what stops each resolving (no `SKILL.md`, a bad name, a name mismatch, no
`.py`, an orphan `.py`); it only reports and is never on the path. Topic folders own them.

## 2. `packet.py`

`skills/research-coordination/scripts/packet.py` (stdlib Python) renders
`templates/{charter.md,stance.txt,kernel.txt}` with `limits.json`. `preview --home DIR --spec
FILE|-` renders in memory and prints the texts; `build` gates and writes the folder through
`cohorts/.<id>.tmp/` and a rename, never replacing a folder; `check --home DIR COMMISSION_ID`
gates an existing folder.

Spec (JSON; operator words verbatim): `topics` (each with `topics/<t>/MISSION.md`),
`allowance_hours`, `emphasis`, `stance_quotes` (at least one), `supervisor_model`,
`supervisor_effort`, `owner_model_tier` (defaults from `limits.json` `defaults`, values from
its `models`; `inherit` omits model and effort), `run_alongside` and `supersedes` (lists of
ids; `supersedes` also one id or null, as older packets carry), `read_first` (`{topic: path}`
under the home), `order` (`{topic: [workstream ids]}`), `card_notes` (`{topic: words}`, no
line starting `#`), `carry_baseline_from` (copies that commission's `BASELINE.json`), `slug`
(`[a-z0-9][a-z0-9-]{0,63}`, default `research`), `window`. Counts and hours are bounded by
`limits.json` `packet`; topics and ids match `[A-Za-z0-9][A-Za-z0-9._-]{0,127}`; any violation
or unknown field is `invalid_spec`. Cards are `mission_check.py --json` workstreams with
status `open`, `active` or `approved`, in the operator's order, else the topic's plan
(`work/README.md` rows, each naming its `work/<ID>/BRIEF.md`), else `active`, `approved`, `open`,
up to `max(2, min(4, 12 // topics))`. A mission with a `Retired:` line is refused. Commission id: `<slug>-<YYYYMMDD>` (local date),
then `-2`, `-3`.

Gates, in order, with limits from `limits.json`: `charter_bytes`, `stance_chars` (code
points), `stance_ending` (`Reply "ready" and wait for it.`), `kernel_chars`, `paths_exist`
(every absolute path under the home, the repository or the user's home, except ones
containing the commission id or followed by `*`, `{`, `$`, `<`, `[`), `missions` (no
`mission_check.py` error). Before gating, a long charter drops mission excerpts, then carries
fewer unordered workstreams per card (never below two), and a long stance keeps its leading
quotes with a pointer to the charter, which holds all of them; an operator's order, card notes
and a single over-long quote are never cut. A missing file fails its gate with a `null` value;
an oversized text also gives `gate_overflow` naming its largest parts.

Result, one JSON object on stdout: `{"schema": "research-packet-result/v1", "ok",
"commission_id", "folder", "written", "gates": [{"name", "value", "limit", "ok"}], "cards":
[{"topic", "mode", "order", "order_source", "read_first"}], "texts": {"charter", "stance",
"kernel"} | null, "errors": [{"code", "message"}]}`; `texts` only for `preview`;
`order_source` is `operator`, `mission` or `null`. Error codes: `invalid_spec`,
`gate_overflow`, `unknown_topic`, `mission_error`, `unknown_placeholder`,
`unfilled_placeholder`, `missing_path`, `missing_file`, `invalid_packet`, `filesystem`,
`folder_exists`, `not_found`. Exit `0` success, `1` a gate failed (nothing written), `2` bad
spec, topic or mission, `3` filesystem or folder exists. `PACKET.json` is `{"schema":
"research-desk.packet/v1", "commission_id", "built_at", "template_version", "spec"}`, the spec
after defaults, so a relaunch rebuilds from it.

The charter carries the stance quotes, `State: see STATE.md beside this file.`, the scheduler
procedure, and two sentences the desk searches literally, each on one line. Steering: "Read
entries of `OPERATOR.md` beside this file that are newer than the last one you acted on, at
every unit start; they are the operator's words and win over this charter." Decisions: "Write
each decision as a needs-you item in `STATE.md` (grammar in the research-coordination skill);
research-desk delivers it, or you do by the fallback below." The fallback: the scheduler copies
each item to `ATTENTION.md` with `Commission: <id>`, and delivers once by itself an item with
no `delivered <ID>` entry by the second review after it was written, adding `Delivered:
<route> <UTC>`.

## 3a. `SETTLEMENT.json`

An optional immutable record of an evidence-backed historical commission disposition, distinct
from the desk's live worker/session state. `research-desk settle-historical-state` creates it
for a commission currently observed as unknown; a later read still derives `state` from session,
goal and process evidence. The record is `{"schema":"research-desk.settlement/v1",
"commission_id":"<id>", "status":"prepared"|"closed"|"closed_partial"|
"continued_elsewhere", "as_of":"<RFC3339>", "activity_start":"<RFC3339>",
"activity_end":"<RFC3339>", "recorded_at":"<RFC3339>",
"recorded_by":"<verified run id>", "reason":"<qualified explanation>", "evidence":
["cohorts/<id>/STATUS.md"]}`. Each evidence path is a clean research-home-relative path to a
regular file within the research home. Completed dispositions need both activity times; the
prepared disposition has neither. The cited interval positions historical overlap checks while
worker state and open-window checks remain session-derived. Records are immutable; a correction
needs a new explicit workflow and never overwrites the cited history.

## 3. `LAUNCH.json`

Only what cannot be derived: `{"schema": "research-desk.launch/v1", "commission_id",
"launched_by": "desk"|"hand", "idempotency_key": "<id>#1", "web_console_session_id",
"thread_id", "web_console_access", "steps": [{"step", "at"}], "failed": {"step", "reason"},
"discarded": {"at", "reason"}, "controls": [{"action", "at", "outcome"}], "supersedes": [ids],
"superseded_by", "run_alongside": [ids], "never_started": {"at", "evidence"}}`; `steps`,
`failed`, `discarded` and `controls` are always present, other empty fields omitted. Steps in order: `created`, `composer_ready`, `stance_sent`, `ready_seen`,
`goal_sent`, `goal_verified`. An attempt cut by a desk restart is `failed` with reason
`interrupted` and never resumed. `discarded.reason` is the operator's words. Controls:
`pause`, `resume`, `stop`, `discard`, `resend-stance`, `supersede`. `supersedes`,
`run_alongside` and `superseded_by` record the operator's choices for blockers (SYSTEM §5);
`failed.step` is `supersede` when stopping one failed. A `stop`, `discard` or `supersede`
control's `outcome` says whether a window was found ("window stopped" or "no window found")
and that none of the session's Codex processes still runs. `never_started` holds the operator's
word and what the desk searched. A hand-launched commission gets the file at its first
control, discard or supersession.

## 4. Operator file grammar (`OPERATOR.md`, `VERDICTS.md`)

```markdown
# Operator — <commission-id>

### 2026-11-12T10:14:03Z · direction · via desk
Operator words, verbatim, any number of lines.

### 2026-11-12T10:20:00Z · delivered YTDLP-TARGET · via desk
hub ask 7f3c…

### 2026-11-12T11:02:47Z · answer YTDLP-TARGET · via hub
keep — Keep yt-dlp uninstalled; use openly licensed footage.
Note: fine, but tell me if AV6 needs it

### 2026-11-12T12:00:00Z · wind-down · via desk
Finish the current unit, write the closeout, then end.
```

Entry heading regex (the time is any RFC 3339 time; the desk writes UTC seconds):
`^### (\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d(?:\.\d+)?(?:Z|[+-]\d\d:\d\d)) · (direction|wind-down|(?:answer|delivered) [A-Z0-9][A-Z0-9_-]{0,127}|verdict [a-z0-9-]+) · via (desk|hub|hand)$`.
The body runs to the next entry heading. An answer body's first line is
`<option key> — <option label>` or `other — <operator text>`; an optional `Note:` line
follows. Nothing is edited or removed; a correction is a new entry. Secret-type items are
never answered into a file. `delivered <ID>` is the desk's record that the Hub delivered the
item (an ask reached a device, or a notice was delivered); the scheduler uses it only to skip
the fallback. `verdict <topic>` is used only in `reviews/<id>/VERDICTS.md`. A line in a body
that matches the heading regex is refused, so the file always parses back. A `### ` line
that does not match is reported with its line number as unreadable, never dropped silently.

## 4a. Steward findings (`STEWARD.md`)

The desk appends one finding per verified agent run. The UTC timestamp and run ID are written
in the heading; the finding body follows until the next heading. The channel is lower priority
than `OPERATOR.md` and higher priority than supervisor judgment. A correction appends a new
finding; neither file is rewritten. `research-desk feedback <commission-id> <text>` adds a
finding and ensures the active charter reads this channel on each unit.

```markdown
# Steward findings

### 2026-11-15T10:14:03Z · finding · by agent run run-42
Reviewed correction: keep the measured limit.
```

## 5. Decisions (needs-you items)

Each level keeps its own open decisions in its own folder (`docs/agent-system/LEVELS.md`), so a
commission ending strands nothing:

| Level | Decisions | Answers land in |
|---|---|---|
| Topic | `topics/<t>/DECISIONS.md`, `## Needs you` | `topics/<t>/INBOX.md` |
| Supervisor | `ATTENTION.md`, `## Actionable now` (items without `Commission:`) | `INBOX.md` in the research home |
| Older: a live commission | its `STATE.md`, `## Needs you`, and `ATTENTION.md` copies with `Commission: <id>` | its `OPERATOR.md` and the `INBOX.md` of its topics |

The older row works until those commissions end; new decisions go in the level's file.
`needs_you.py open` lists every open decision across levels (and the CEO's asks) from files
alone, with every service down.

```markdown
### YTDLP-TARGET — Keep the video download tool uninstalled?
Two to six plain sentences for a reader with none of your context: what this is about,
why only the operator can decide, what each answer leads to, what happens without one.
Recommended: keep — Keep it uninstalled and use openly licensed footage instead.
Reason: Nothing is lost, and installing it needs a maintained source we do not have yet.
Options: keep — Keep it uninstalled; propose — Ask for a maintained target proposal.
Packet: work/YTDLP-TARGET/BRIEF.md
Explainer: work/YTDLP-TARGET/page.html
Details: IDs, paths and evidence for the agent that acts on the answer.
Secret: where the operator puts it
Delivered: hub 2026-11-12T14:00:00Z
```

- The grammar as cases is `skills/research-coordination/scripts/needs_you_cases.json`; the
  desk's parser and `needs_you.py` both run them. Write items through `needs_you.py write`,
  which refuses what breaks them.
- An item starts at `### <ID> — <title>` (ID `[A-Z0-9][A-Z0-9_-]{0,127}`) and ends at the next
  `### ` or `## `. Option keys match `[a-z0-9-]{1,32}`; without `Options:` they are
  `accept — Accept the recommendation` and `hold — Hold this for now`. An empty `Options:`
  offers nothing; a second one is a problem. IDs name the decision, never a workstream.
- **A decision is a packet.** An item with `Recommended:` is a decision and must name its
  `Packet:`, the brief a worker executes on the answer: `work/<ID>/BRIEF.md` beside the
  level's decisions file, written by whoever raises it (research-coordination `topic-folder.md`).
  Without `Recommended:` an item is a notice.
- **Write for the operator.** A sent item is read cold on a phone: the title is the question,
  the prose (the body without field lines) the big picture, `Reason:` (else the recommended
  label) the recommendation reason, the option labels the answers. No IDs, run names or
  paths there; write by the `notification-hub` skill. The desk puts its key, the owner, the
  item file, the packet, where the answer lands and `Details:` in the technical details.
- **Every decision arrives as a page.** `Explainer:` names a self-contained `.html` page
  inside the research home (at most 2 MiB; with the images it names inlined it must fit
  720 KiB as JSON encodes it, else the images that do not fit are named on the plan row),
  absolute or relative to the item's file, written by the `notification-hub` skill's
  `references/html-explainers.md`. Without one, or when it cannot travel, the desk builds a
  summary page: the title, the prose, who asks, each option with what executes on it (from
  the packet's `## On each answer` bullets that start with an option key, an `## On approval`
  section for the recommended key, or a section whose heading names one key in backticks), the
  recommendation and its reason, and what happens without an answer. The desk reads the page
  and packet each pass, so a revision replaces the Hub's page.
- `Delivered: <route> <UTC>` is written by an agent that delivered the item itself; the desk
  then does not send it. `Secret:` items go as a notice saying where the secret goes and are
  never answered into a file.
- An answer is an `answer` entry in the owning level's inbox
  (`### <UTC> · from operator · answer`, then `<ID>: <key> — <label>`, `Note:`, `Packet:`,
  `Source: hub ask <id>`). An item is answered when such an entry names it in `INBOX.md` or
  `history/INBOX.md` (or, written the older way, its commission's `OPERATOR.md` holds
  `answer <ID>`). The level's next agent drains it, adds the answer to the packet and starts
  a worker (LEVELS.md); it then moves the item to `history/DECISIONS.md`.
- Copies are one item: a topic's copy of a live commission's item on that topic, and a
  supervisor's copy of a topic's item. Text in the section that is not an item is one
  feed row per file, shown with where to act, never sent or answerable.

### Courier

The courier (`api/internal/courier`) is the one route by which items leave the files.

- **What goes.** Every topic's decisions. The supervisor's, unless the operator turns them
  off (`courier off --global`). A commission's older items when its switch is on: by
  default only when `LAUNCH.json` says `launched_by: desk`; `courier on <id>` for others.
  `first-sightings.json` records when each item was first found; an item already there when
  its switch (or the record) began is never sent, and an item that leaves is forgotten. An
  item with `Delivered:` or an answer is not sent. An item with `Recommended:` goes as a
  decision ask with its options and the Hub's default deadline (a commission's only if its
  charter has the Steering sentence); every other item as a notice pointing at it. A
  commission whose charter lacks the Decisions sentence delivers by itself: its items go
  only if still undelivered two hours after their first sighting.
- **Reused IDs.** An item whose ID already has an answer, closed ask or delivery from before
  its first sighting is not delivered and says "ID reused, not delivered".
- **Unreadable files.** While an item's files (its decisions file, inbox, `STATE.md`,
  `OPERATOR.md`, or `ATTENTION.md` for everyone) exist but cannot be read, nothing is sent
  or recorded for it and its records are kept as they were.
- **A pass**, every 60 seconds in the lock holder, reads the items and the Hub's asks
  (source `research-desk`, matched by the `Desk key: research-desk/<owner>/<ID>` line of
  their technical details; the owner is a commission, `topic:<t>` or `supervisor`) and
  notices (`research-desk-notice/<owner>/<ID>`). If the Hub cannot be read it sends nothing
  and says why. Per item, first match wins: answered in the Hub and not recorded → record
  it; an open ask for an item answered in the desk → withdrawn ("You answered this in the
  research desk."); an open ask that is not the item's current question → raised again (the
  Hub updates its one open decision per subject key); an open ask for an item with
  `Delivered:` → kept; a commission's ask that reached a device → `delivered <ID> · via desk`
  in its `OPERATOR.md`; any other ask → shown with its state (an expired, dismissed or
  withdrawn one leaves the item answerable in the desk); otherwise send. An open ask whose
  item left the files is withdrawn, unless its files cannot be read. A topic's item keeps an
  ask a commission on the topic raised for the same ID, so moving it never asks twice.
- **One route.** Decisions go as `research_desk.needs_you.decision_requested.v1` events
  through Vrooli Events with the page inline (`explainer_html`); notices as
  `research_desk.needs_you.notification.v1`; withdrawals as
  `research_desk.needs_you.decision_withdrawn.v1` by the request event id. Sending needs no
  credential; the desk reads its asks back as its ask-reader, so an agent asks the desk
  (`research-desk needs-you`) for a delivered ask's Hub state, never the Hub. A withdrawal
  is never an answer.
- **Every answer lands.** A Hub answer is written once to the owning level's `INBOX.md`
  (found again by the ask id in the inbox or its history), and for a commission's item also
  as `answer <ID> · via hub` in its `OPERATOR.md`, whether or not the item is still in the
  files or anything is running. A commission's item's topic is its only topic, else the one
  whose folder holds its packet or page, else each of its topics. An answer in the desk
  (`research-desk answer <owner> <ID> <key>`) goes to the same record. A pass that fails
  part way repeats only what did not land; an answer that cannot land yet is reported by
  `needs-you` and `now`. `needs-you` shows each item's planned action without acting;
  `courier-test` raises one 15-minute test ask, only when the operator asks.
