# File Contract

The whole contract between the desk's Go code and the research skills: the packet script
(§2), the launch record (§3), the operator file grammar (§4) and the needs-you grammar (§5).
Everything here works with a text editor and no desk. Research home: `~/.vrooli/research`.

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

Desk writes: only the live instance writes, holding an exclusive `flock` on
`~/.vrooli/state/vrooli/research-desk/instance.lock`; other instances are read-only. JSON is
written to a temp file in the same directory and renamed. Markdown appends are one
`O_APPEND` write. The desk's own state lives in that folder, outside the research home:
`courier.json` (the operator's courier switches), `courier-wait.json` (when the courier first
held an item for its scheduler), `looks/<commission|topic>-<id>.json` (since-last-look) and
`launch-tests/<UTC stamp>.json` (one report per `launch-test`, never in the research home).
Deleting the folder resets "since" and returns the switches to their safe defaults.

The desk serves research-home files read-only at GET `route[example]:/api/v1/home/<path>` (no hidden
parts; symlinks only when they resolve inside the home; sandboxed). `REVIEW.html` links its
media by paths relative to itself that resolve inside the research home, so it renders the
same from disk and from the desk.

## 2. `packet.py`

`scenarios/research-desk/skills/research-coordination/scripts/packet.py` (stdlib Python)
renders `../templates/{charter.md,stance.txt,kernel.txt}` with `limits.json`; an unknown or
unfilled `{{name}}` placeholder is an error.

| Subcommand | Effect |
|---|---|
| `preview --home DIR --spec FILE\|-` | render in memory; print result with texts; write nothing |
| `build --home DIR --spec FILE\|-` | render, gate, write the folder atomically; print result without texts |
| `check --home DIR COMMISSION_ID` | gate an existing folder (prepared or older packet); write nothing |

Spec (JSON): `{"topics": ["ai-video-production-and-funnels"], "allowance_hours": 12,
"emphasis": "", "stance_quotes": ["operator words"], "supervisor_model": "gpt-6-luna",
"supervisor_effort": "medium", "owner_model_tier": "inherit", "run_alongside": [],
"read_first": {"<topic>": "/abs/CHECKPOINT.md"}, "order": {"<topic>": ["AV6", "AV4"]},
"card_notes": {"<topic>": "operator words"}, "supersedes": [],
"carry_baseline_from": null, "slug": "programme-five", "window": "fifth"}`. Operator
words are verbatim.

Rules: 1–6 topics, each with `topics/<t>/MISSION.md`; `allowance_hours` 1–72; at least
one quote; unknown fields are `invalid_spec`. `slug` matches `[a-z0-9][a-z0-9-]{0,63}`;
topics and commission ids match `[A-Za-z0-9][A-Za-z0-9._-]{0,127}`. Missing model fields
take `limits.json` `defaults` (the research defaults;
`owner_model_tier: inherit` renders "omit `model` and `reasoning_effort`" in the brief).
`run_alongside` and `supersedes` are lists of commission ids (`supersedes` also accepts one id
or null, as packets built before lists carry), recorded and rendered as one line each in the
charter ("Supersedes `a` and `b`.");
`read_first` values must be existing files under the research home and become `Read first:`
on the card; `carry_baseline_from` copies that commission's `BASELINE.json`.
`supervisor_model`, `supervisor_effort` and `owner_model_tier` must be in `limits.json`
`models.<same key>`. Cards: the `workstream_rows` of `mission_check.py --json` (`{id,
destination, magnitude, showcase, status}` in table order) whose status is `open`, `active`
or `approved`; `order.<topic>` (1–4 distinct such IDs, topic selected) chooses and orders
them; without it a card lists `active`, then `approved`, then `open` rows, up to
`max(2, min(4, 12 // topics))`. `card_notes.<topic>` (non-empty, no line starting `#`) is
rendered verbatim under "Operator's notes for this card, verbatim:". `Not this` and
`Boundaries` cite the mission sections by path. Each violation is `invalid_spec`.

Commission id: `<slug>-<YYYYMMDD>` (local date; slug default `research`), then `-2`, `-3`
if taken. Build writes `cohorts/.<id>.tmp/` and renames it; it never replaces a folder.

Gates (values in `limits.json`): charter ≤ 25,000 bytes; stance ≤ 3,500 code points and
ending `Reply "ready" and wait for it.`; kernel ≤ 600 code points; every absolute path in
the texts under the research home, the repository or the user's home exists (except paths
containing the commission id and patterns followed by `*`, `{`, `$`, `<` or `[`); no
`mission_check.py` error for a selected topic. Gate names, in order: `charter_bytes`,
`stance_chars`, `stance_ending` (value: the actual ending), `kernel_chars`, `paths_exist`
(value: count missing), `missions` (value: topics with errors); a missing file gives a
`null` value and a failed gate. Each oversized text also gives a `gate_overflow` error naming
the overflow and the largest parts, e.g. "charter is 25,230 bytes, 230 over 25,000; largest
parts: <section or card> <bytes> (notes <bytes>); …".

Result (stdout, one JSON object, all subcommands):

```json
{
  "schema": "research-packet-result/v1",
  "ok": true,
  "commission_id": "programme-five-20261112",
  "folder": "<research home>/cohorts/programme-five-20261112",
  "written": false,
  "gates": [{"name": "charter_bytes", "value": 21034, "limit": 25000, "ok": true}],
  "cards": [{"topic": "...", "mode": "build", "order": ["AV1", "AV2"], "order_source": "mission", "read_first": null}],
  "texts": {"charter": "...", "stance": "...", "kernel": "..."},
  "errors": [{"code": "unknown_topic", "message": "..."}]
}
```

`texts` is `null` for `build` and `check`. `check` derives cards from `### Card` headings
and the mission's workstream IDs in first-mention order; `order` is `null` when no card
names a mission. `order_source` is `"operator"`, `"mission"`, or `null` (`check` on an older
layout). Error codes: `invalid_spec`, `gate_overflow`, `unknown_topic`, `mission_error`,
`unknown_placeholder`, `unfilled_placeholder`, `missing_path`, `missing_file`,
`invalid_packet`, `filesystem`, `folder_exists`, `not_found`. Exit codes: `0` success; `1` a
gate failed (nothing written); `2` invalid spec, unknown topic or mission error; `3`
filesystem error or folder exists.

`PACKET.json` (written by build): `{"schema": "research-desk.packet/v1",
"commission_id", "built_at", "template_version", "spec"}`; `spec` is the full input after
defaults, so a relaunch rebuilds from the same inputs.

The rendered charter must contain: the stance quotes; `State: see STATE.md beside this
file.`; a Steering section ("Read entries of `OPERATOR.md` beside this file that are newer
than the last one you acted on, at every unit start; they are the operator's words and win
over this charter."); a Decisions section ("Write each decision as a needs-you item in
`STATE.md` (grammar in the research-coordination skill); research-desk delivers it, or you do
by the fallback below. This overrides research-coordination step 9 and LONG_EFFORT_DESIGN D17
on delivery."), which also tells the scheduler to copy each item to `ATTENTION.md` with
`Commission: <id>` and gives the fallback: an item with no `delivered <ID>` entry by the
second review after it was written is delivered once by the scheduler, which then adds
`Delivered: <route> <UTC>`; the route-discipline procedure; the scheduler procedure and a
pointer to `templates/owner-brief.md`. The desk searches both sentences literally, each on
one line.

## 3. `LAUNCH.json`

Only what cannot be derived:

```json
{
  "schema": "research-desk.launch/v1",
  "commission_id": "programme-five-20261112",
  "launched_by": "desk",
  "idempotency_key": "programme-five-20261112#1",
  "web_console_session_id": "4a72d7a3-...",
  "thread_id": "01a1cc78-...",
  "steps": [{"step": "created", "at": "2026-11-12T09:01:10Z"}],
  "failed": null,
  "discarded": null,
  "controls": [{"action": "pause", "at": "..."}],
  "supersedes": ["programme-three-20261111"],
  "run_alongside": ["programme-four-20261110"],
  "never_started": {"at": "2026-11-12T09:00:58Z", "evidence": "no session or goal names this charter in ..."}
}
```

Steps in order: `created`, `composer_ready`, `stance_sent`, `ready_seen`, `goal_sent`,
`goal_verified`. `failed` is `{"step", "reason"}`; an attempt the desk was running when it
restarted is written `failed` with reason `interrupted` and never resumed. `discarded` is
`{"at", "reason"}` with the operator's words. `controls` records `pause`, `resume`, `stop`,
`discard`, `resend-stance` and `supersede`.
`supersedes` and `run_alongside` are the operator's choices for blocking commissions (SYSTEM
§5); `failed.step` is `supersede` when stopping one of them failed and nothing new started.
A superseded commission's own file gets `superseded_by` (the new commission), `discarded` and
the `supersede` control. `never_started` is the operator's assertion that an `unknown`
commission was never started, with the sentence saying what the desk searched. The optional
fields are omitted when empty.
For a hand-launched commission the file exists only after an operator control or discard,
with `launched_by: "hand"`. One commission has one attempt; a relaunch is a new commission.

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

Entry heading regex:
`^### (\d{4}-\d\d-\d\dT\d\d:\d\d:\d\dZ) · (direction|wind-down|(?:answer|delivered) [A-Z0-9][A-Z0-9_-]{0,127}|verdict [a-z0-9-]+) · via (desk|hub|hand)$`.
The body runs to the next entry heading. An answer body's first line is
`<option key> — <option label>` or `other — <operator text>`; an optional `Note:` line
follows. Nothing is edited or removed; a correction is a new entry. Secret-type items are
never answered into a file. `delivered <ID>` is the desk's record that the Hub delivered the
item (an ask reached a device, or a notice was delivered); the scheduler uses it only to skip
the fallback. `verdict <topic>` is used only in `reviews/<id>/VERDICTS.md`. A line in a body
that matches the heading regex is refused, so the file always parses back.

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
  scheduler that delivered the item itself. `Commission: <id>` appears only on
  `ATTENTION.md` copies.
- An item's identity is its commission plus its ID; an `ATTENTION.md` item without
  `Commission:` is global and never merged with a commission's item. Items of an ended
  commission are not shown.
- Text in the section that is not an item (older numbered lists) is shown as "legacy, not
  sent".
- The scheduler removes an item after acting on its answer.

### Courier

The courier (`api/internal/courier`) is the one route by which items leave the files; these
are its rules, stated only here.

- **Switches.** Off for every commission the desk did not launch and for global items until
  the operator turns it on (`courier on <id>` / `--global`); on by default only when
  `LAUNCH.json` says `launched_by: desk`. Switches live in the desk state folder
  (`courier.json`, entries `{on, at, baseline}`). Turning one on records the items present
  then as its baseline; those are never sent, only items first seen afterwards. With no
  switch on, the Hub is not read.
- **What is sent.** An item with `Delivered:` or an answer is not sent. An item with
  `Recommended:` whose charter has the Steering sentence goes as a decision ask with its
  options and the Hub's default deadline; every other item (no `Recommended:`, any
  `Secret:`, a global item, a charter without Steering) as a plain notice pointing at the
  item. Items of a charter without the Decisions sentence, and global items, are sent only if
  they still have no `Delivered:` line two hours after the courier first held them
  (`courier-wait.json`), because their scheduler delivers by itself.
- **Reused IDs.** First sightings are kept in `courier-seen.json`. An item whose ID already
  has an answer or delivery from before it first appeared is not delivered and says "ID
  reused, not delivered"; the earlier answer is not shown as its answer.
- **Unreadable files.** While a commission's `STATE.md` or `OPERATOR.md` (or `ATTENTION.md`,
  for every commission) exists but cannot be read, nothing is sent for it, its records are
  kept as they were, and answering it and turning its switch on are refused.
- **A pass**, every 60 seconds in the instance holding the write lock: read the items of
  non-ended commissions and `ATTENTION.md`; if Vrooli Events cannot be reached, send nothing
  and say why per item; otherwise read the Hub's asks from source `research-desk` (each found
  again by its question `<commission> · <ID> — <title>`) and the desk's notices (dedupe key
  `research-desk-notice/<owner>/<ID>`); a failed read sends nothing. Then per item, first
  match wins: answered in the Hub and not in `OPERATOR.md` → append `answer <ID> · via hub`;
  an open ask for an item answered in the desk or carrying `Delivered:` → shown as still open
  until its deadline; an ask that reached a device with no `delivered` entry → append
  `delivered <ID> · via desk`; any other ask → shown with its state (an expired ask leaves
  the item answerable in the desk); otherwise send, per the rules above. A notice the Hub
  reports delivered gets a `delivered` entry on the next pass. An open ask whose item left
  the files is listed with its deadline.
- An unchanged item is the same event (its id hashes the content), so a repeat is
  deduplicated and a desk outage loses nothing. The desk cannot withdraw an ask: one it no
  longer needs stays open until its deadline, and a later answer to it is ignored.
- `needs-you` shows each item's planned action without acting. `courier-test` (self-test)
  raises one ask titled as a delivery test with a 15-minute deadline, only when the operator
  asks.
