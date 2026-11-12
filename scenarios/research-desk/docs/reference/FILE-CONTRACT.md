# File Contract

The contract between the desk and the research skills, usable with a text editor and no desk.
Research home: `~/.vrooli/research`.

## 1. Writers

| File | Only writer |
|---|---|
| `cohorts/<id>/{README.md,STANCE-MESSAGE.txt,KERNEL.txt,PACKET.json}` | `packet.py build` (run by the desk or an agent); on the operator's `enable-steering` the desk appends `## Steering` with the one Steering sentence to an older `README.md` |
| `cohorts/<id>/OPERATOR.md`, `reviews/<id>/VERDICTS.md` | the desk for the operator, or the operator by hand; append-only |
| `cohorts/<id>/LAUNCH.json` | the desk |
| `cohorts/<id>/{STATE.md,LEDGER.md,BASELINE.json}` | the scheduler (`BASELINE.json` may be seeded once by `packet.py`) |
| `topics/<t>/MISSION.md`, `ATTENTION.md` | supervisors and the operator; never the desk |
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
status `open`, `active` or `approved`, in the operator's order, else `active`, `approved`,
`open` up to `max(2, min(4, 12 // topics))`. Commission id: `<slug>-<YYYYMMDD>` (local date),
then `-2`, `-3`.

Gates, in order, with limits from `limits.json`: `charter_bytes`, `stance_chars` (code
points), `stance_ending` (`Reply "ready" and wait for it.`), `kernel_chars`, `paths_exist`
(every absolute path under the home, the repository or the user's home, except ones
containing the commission id or followed by `*`, `{`, `$`, `<`, `[`), `missions` (no
`mission_check.py` error). A missing file fails its gate with a `null` value; an oversized
text also gives `gate_overflow` naming its largest parts.

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

## 5. Needs-you grammar

One grammar for `STATE.md` `## Needs you` and `ATTENTION.md` `## Actionable now`:

```markdown
### YTDLP-TARGET — yt-dlp target choice
One to three sentences: what is needed and why.
Recommended: keep — Keep yt-dlp uninstalled; use openly licensed footage.
Options: keep — Keep it uninstalled; propose — Ask for a maintained target proposal.
Packet: /abs/path/to/details.md
Secret: where the operator puts it
Delivered: hub 2026-11-12T14:00:00Z
Commission: programme-five-20261112
```

- An item starts at `### <ID> — <title>` (ID regex `[A-Z0-9][A-Z0-9_-]{0,127}`) inside
  the section and ends at the next `### ` or `## `. `Recommended:`, `Options:`, `Packet:`
  and `Secret:` lines are optional. Option keys match `[a-z0-9-]{1,32}`; without `Options:`
  they are `accept — Accept the recommendation` and `hold — Hold this for now`.
- An empty section's only line is `No actionable items are recorded.`
- IDs name the decision, never a workstream. `Delivered: <route> <UTC>` is written by a
  scheduler that delivered the item itself, or that copied the desk's receipt; a line whose
  route is `research-desk` or that names the Hub ask or notice of the desk's `delivered`
  entry is the desk's delivery. `Commission: <id>` appears only on `ATTENTION.md` copies.
- An item's identity is its commission plus its ID; an `ATTENTION.md` item without
  `Commission:` is global. Items of an ended commission are not shown.
- Text in the section that is not an item (an older numbered list) is one feed row per file,
  readable in place with where to act, counted apart from items ("2 items wait for an
  answer and 4 older lists"; an answered item is listed but not counted); never sent, never
  answerable.
- The scheduler removes an item after acting on its answer.

### Courier

The courier (`api/internal/courier`) is the one route by which items leave the files; these
are its rules, stated only here.

- **Switches.** Off for every commission the desk did not launch, and for global items, until
  the operator turns it on (`courier on <id>` / `--global`); on by default only when
  `LAUNCH.json` says `launched_by: desk`. With no switch on, the Hub is not read.
- **First sightings.** `first-sightings.json` holds when each item was first found in the
  files (the zero time: already there when the record began). Turning a switch on records a
  sighting of every item present at that moment, and an item first sighted at or before its
  switch was turned on is never sent. An item that leaves the files is forgotten.
- **What is sent.** An item with `Delivered:` or an answer is not sent. An item with
  `Recommended:` whose charter has the Steering sentence goes as a decision ask with its
  options and the Hub's default deadline; every other item (no `Recommended:`, any
  `Secret:`, a global item, a charter without Steering) as a plain notice pointing at the
  item. Items of a charter without the Decisions sentence, and global items, are sent only if
  they still have no `Delivered:` line two hours after their first sighting, because their
  scheduler delivers by itself.
- **Reused IDs.** An item whose ID already has an answer, closed ask or delivery from before
  its first sighting is not delivered and says "ID reused, not delivered"; the earlier answer
  is not shown as its answer.
- **Unreadable files.** While a commission's `STATE.md` or `OPERATOR.md` (or `ATTENTION.md`,
  for every commission) exists but cannot be read, nothing is sent for it, its records are
  kept as they were, and answering it and turning its switch on are refused.
- **A pass**, every 60 seconds in the lock holder, reads the items of non-ended commissions
  and `ATTENTION.md`, and the Hub's asks (source `research-desk`, matched by the question
  `<commission> · <ID> — <title>`) and notices (dedupe key `research-desk-notice/<owner>/<ID>`).
  If Vrooli Events or the Hub cannot be read, it sends nothing and says why. Per item, first
  match wins: answered in the Hub, not in `OPERATOR.md` → append `answer <ID> · via hub`; an
  open ask for an item answered in the desk or delivered → shown open until its deadline; an
  ask that reached a device, or a notice the Hub reports delivered → append `delivered <ID> ·
  via desk`; any other ask → shown with its state (an expired one leaves the item answerable
  in the desk); otherwise send. An open ask whose item left the files is listed with its
  deadline.
- An unchanged item is the same event, so a repeat is deduplicated and an outage loses
  nothing. The desk cannot withdraw an ask: an unneeded one stays open until its deadline,
  and a later answer to it is ignored. `needs-you` shows each item's planned action without
  acting; `courier-test` raises one 15-minute test ask, only when the operator asks.
