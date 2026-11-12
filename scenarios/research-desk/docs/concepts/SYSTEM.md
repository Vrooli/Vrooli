# System Design

The one design document. File shapes are in
[`../reference/FILE-CONTRACT.md`](../reference/FILE-CONTRACT.md); choices and their
alternatives in [`../DECISIONS.md`](../DECISIONS.md); the skills plan in
[`../guides/skills-move.md`](../guides/skills-move.md).

## 1. Shape

A Go API, a thin CLI over it, and a phone UI over it. The API reads the research home and
the Codex session homes (`~/.vrooli/state/vrooli/web-console/sessions/codex/<session>/`),
runs `packet.py`, types into Web Console sessions only on an operator tap, and carries
decisions to and from Notification Hub. No database: a memory cache keyed by path, size and
mtime. One reader, `api/internal/home`, parses every file and returns each derived field
as `{value, known, source_path, observed_at}`; nothing else in `api/` parses research files.
One writer, `api/internal/writer`, makes every desk write (contract §1).
The optional scenarios (web-console, vrooli-events, notification-hub) are declared once, in
`.vrooli/service.json`, and the capability registry (`api/internal/capabilities`) is the one
table of whether each can be reached: launch and the courier dial a peer only when the
registry says it is available, every peer call has a timeout, and the overview lists each
with the registry's reason. `python3` (for the skill scripts) is not a registry entry; the
overview's `mission_check.py` lens says when it fails. Whatever is unavailable leaves the
fields that need it `unknown`.

## 2. Domains and surface

| Domain | RPCs | CLI (`research-desk …`) |
|---|---|---|
| activity | `GetOverview`, `ListCommissions`, `GetCommission`, `ListTopics`, `GetChanges` | `now`, `commissions [--state]`, `commission <id>`, `topics`, `changes <commission\|topic> [--since]` |
| launch | `GetLaunchOptions`, `Preflight`, `Build`, `GetLaunchMessages`, `Launch`, `GetLaunch`, `Control`, `TestLaunch`, `GetLaunchTest` | `launch-options [--topics]`, `preflight <spec-file\|id> [--supersede ids] [--alongside ids] [--assert-never-started]`, `build <spec-file> \| --relaunch-of <id> [--supersede ids] [--alongside ids]`, `launch-messages <id>`, `launch <id> [--supersede ids] [--alongside ids] [--assert-never-started]`, `launch-status <id>`, `control <id> pause\|resume\|stop\|discard\|resend-stance [--reason]`, `launch-test`, `launch-test-status` |
| operator | `ListNeedsYou`, `Answer`, `AddEntry`, `EnableSteering`, `ListReviews`, `AddVerdict`, `SetCourier`, `TestCourier`, `MarkSeen` | `needs-you`, `answer <id> <item> <option> [--note] [--dry-run]`, `say <id> direction\|wind-down [words] [--dry-run]`, `enable-steering <id>`, `reviews`, `verdict <review> <topic> <words> [--dry-run]`, `courier on\|off [<id>\|--global]`, `courier-test` (self-test), `mark-seen <commission\|topic> <id> \| --all` |

`ListNeedsYou` is also the courier's dry run: each item carries what the courier would do
and why. REST exceptions: `GET /health` (ops probe) and GET `route[example]:/api/v1/home/<path>` (read-only
research-home files for the browser, so `REVIEW.html` and its media render). Protos live in
`packages/proto/schemas/research-desk/v1/{activity,launch,operator}/`. The endpoint list is
generated: `make endpoints` writes `.vrooli/endpoints.json`.

## 3. Recognition

- **exact**: `LAUNCH.json` names the session; or a `thread_goals.objective` (any status) in
  any Codex home's `goals_1.sqlite` (opened `?mode=ro`), or a `history.jsonl` entry whose
  text starts with `/goal `, contains the commission's absolute charter path after
  trimming whitespace. An objective stored as "Read the Codex goal objective file at
  `<home>/attachments/<id>/goal-objective.md`" is read from that file.
- **related**: any other `history.jsonl` entry naming the folder (usually a preparation
  chat). Shown, never used for state.
- Several exact matches are listed, newest first; state uses the newest.
- Window: only goals and history entries at or after the packet's earliest file time
  minus one day (local dates) count.
- No match: older folders are `unknown`; transcripts are not searched.

## 4. Commission state

Derived on every read; never stored. Inputs: packet files, `LAUNCH.json`, goal status of
the exact thread, the TUI process (a non-app-server `codex` process whose environment has
`CODEX_HOME=<home>`), newest write (commission folder files, the home's newest rollout,
`topics/<t>/runs/<commission>/CHECKPOINT.md`), and the allowance end (goal `created_at_ms`
plus the `<N>h allowance` in `KERNEL.txt` or the packet spec).

| State | Rule, first match wins |
|---|---|
| `ended` | `discarded` in `LAUNCH.json`; or goal `complete`; or allowance end passed and no TUI |
| `prepared` | packet complete, no exact or related match, no `LAUNCH.json`, and proof nothing started it: no goal in any readable Codex home names the charter, `STATE.md` and `LEDGER.md` unchanged since the build; without that proof `unknown` |
| `unknown` | no exact match |
| `stopped` | no TUI process, goal not `complete`, allowance not passed |
| `live` | TUI present, goal `active`; flag `stalled` after 70 minutes without a write (`home.StallAfter`) |
| `paused` | TUI present, goal `paused` |
| `limited` | TUI present, goal `usage_limited`, `budget_limited` or `blocked` |

**The operator's assertion (P1).** An `unknown` commission carries `never_started`: whether the
operator may launch it on their word that it was never started, and why. It is assertable only
when the packet is complete, there is no `LAUNCH.json`, and no exact or related match and no
goal naming the charter was found in any Codex home the desk can read (homes it cannot read are
named in the reason). A launch with the assertion records it in `LAUNCH.json` `never_started`
(`at`, the evidence sentence); it is refused when any session or goal is found.

**When an operator entry is read (P5).** Each commission reports `entry_reading`, first match
wins: `never` (ended); `unknown` (whether the charter has the Steering line is unknown);
`not_until_steering` (no Steering line); `next_unit` (live); `after_resume` (paused, or limited
until the limit lifts); `at_start` (prepared or stopped: when a supervisor starts); else
`unknown`. Each comes with a sentence and `read_since`, the `STATE.md` time: an entry older than
it has been read.

The supervisor's own status line is shown beside the state as a claim. Supervisors running
= `live` + `paused` + `limited`. Overlap = two commissions sharing a topic whose intervals
(goal start to last write, or now) intersect.

**What moved on a topic (P8).** A topic reports three facts, each with its source file:
`last_research_write` (the newest file in its run folders, with the owning commission and
state), `mission` (`MISSION.md`) and `last_review` (the newest `REVIEW.md` with a section for
the topic). Topics sort by the research write; the overview carries the home's
`latest_research_write`. A write the owner's state does not explain carries a note: after a
paused goal's pause ("written while paused: an owner may still be running"), under a limited
goal, while the window is closed (stopped), or after an ended commission's end.

## 5. Launch

**Preflight** (spec or prepared commission): `packet.py preview` or `check` gates, plus a
verdict per topic over every commission naming it (topic evidence, first match wins:
`PACKET.json` spec, the first `topics/<t>/` link in a `### Card` block, any
`topics/<t>/MISSION.md`, `topics/<t>/README.md` or `topics/<t>/` link in the charter, run
folders `topics/<t>/runs/<commission>[-<suffix>]/`).

Blockers are few and are resolved in the request (P2). An earlier commission blocks only when it
is `live`, `paused` or `limited`, or `unknown` with an open (or unchecked) session, an unknown
last write, or a write in the last 24 hours (`launch.QuietAfter`); an older quiet `unknown` is
listed with `quiet` and never blocks. Each blocker carries `choices` (`supersede`,
`run_alongside`) and the desk's `recommended` choice with its reason: supersede when it is paused
or limited and its allowance has ended; none when it is live or the desk cannot tell. The
request carries a `resolutions` entry per blocker (the spec's `run_alongside` and `supersedes`
count as choices; the request wins), and the verdict echoes it as `chosen`. `stopped`, `ended`
→ nothing to do, its newest checkpoint offered as `read_first`; `prepared` → "another
unlaunched packet covers this topic". A commission that is itself `unknown` is launchable only
with the operator's assertion (§4).

**Supersede.** Before anything is launched, each superseded commission whose window is open
must pass the Stop checks of §6 (its own thread shown, input row idle); otherwise the launch is
refused with "cannot stop <id> …; nothing was launched". The launch then stops each one through
its own session (a failure is `failed` at step `supersede`, and nothing new is started), marks
it discarded with `superseded_by` and a `supersede` control, and records `supersedes` in the new
`LAUNCH.json`. Choices made before the packet exists go to `Build` (`resolutions`), which writes
them into the spec, so `packet.py` renders "Supersedes `a` and `b`." and "Runs alongside …" in
the charter (a relaunch supersedes the old commission the same way). Every superseded supervisor
is checked before anything is launched; if any one cannot be stopped the whole launch is refused,
naming it. Only a packet launched as it already is
(prepared, or hand-made) records a supersede chosen at Launch in `LAUNCH.json` alone; the desk
never rewrites `README.md`.

**Suggestions (P3).** `GetLaunchOptions` with topics returns stance quotes with their source:
the newest commission (by build time) naming those topics, its `PACKET.json` quotes, else the
quoted lines of its `STANCE-MESSAGE.txt`, else none with the reason. It also returns each
topic's workstreams with title, magnitude and showcase line.
Supervisor model and effort and the owner model tier are inputs with the `limits.json`
defaults. The remaining-quota lens (ruling B3) is not built (PROBLEMS.md): the desk does not read remaining usage; `limited` comes only from the goal row's status (`usage_limited`, `budget_limited`, `blocked`) with the TUI present (§4).

**Build** runs `packet.py build` (with the request's blocker choices written into the spec)
and returns its JSON. Prepared packets launch as they are
after `check`.

**Launch (automatic)** (`api/internal/launch/launcher.go`), asynchronous, through Web Console's HTTP API with
`X-Idempotency-Key: <commission>#1`: create a persistent session whose command is the
governed launcher (`vrooli-agent-launcher --agent codex -- --yolo -m <model> -c
model_reasoning_effort=<effort>`, falling back to `codex --yolo …` as Web Console's profile
does; only the governed launcher gives the session its own `CODEX_HOME`, so the fallback
fails at `goal_verified`); wait for the composer; paste `STANCE-MESSAGE.txt` and press Enter;
wait for `ready`; type `/goal <KERNEL.txt>` and Enter; verify the goal row is `active` with an
objective equal to the kernel. Every Enter follows a one-second settle, and the input row is
read back afterwards: Codex takes keys that arrive in one burst as a paste, so an Enter right
behind typed text inserts a newline (measured on the first launch test); one more Enter is
pressed if the text is still there, and the step fails if it stays. Each step is appended to
`LAUNCH.json`; nothing else is ever sent. An attempt cut by a desk restart is `failed:
interrupted`. **Manual**: `GetLaunchMessages` (a read, open to agents) returns the charter path, the
stance and the kernel to copy; recognition (§3) finds the session.

**Launch test** (`launch-test`, D19) runs this path on one throwaway session labelled
`research-desk launch test <stamp> (stopped; keep)` with a fixed harmless stance and a
591-character kernel that completes at once, then pauses, resumes and stops it, keeping a
report in the desk state folder (`launch-test-status`). It writes nothing in the research
home. Web Console's session owner shows who it believed made the call; without a desk credential
it is `ui-fail-open`, Web Console's availability policy, and the report says which access
was used (§11).

**Relaunch** is a new commission (`build --relaunch-of <id>`): `packet.py build` with the old
`PACKET.json` spec, `supersedes` set and `carry_baseline_from` the old commission. Older packets without
`PACKET.json` are rebuilt from the operator's choices.

## 6. Control (G12)

Only on an operator tap, only for an exact match with a TUI present, through Web Console
terminal input; each is recorded in `LAUNCH.json` `controls`. Before typing, the session must
show the commission's own thread (from its transcripts) with the cursor in Codex's input row
and no choice open; otherwise the control is refused with the manual path. Pause and Resume
may go in while a turn runs; Re-send stance and Stop wait for an idle input row, because
text typed during a turn queues as a message and Stop would cut the turn.

| Action | Mechanism |
|---|---|
| Wind down | `OPERATOR.md` entry `wind-down`; read at the next unit; no keystrokes |
| Pause | type `/goal pause`, Enter |
| Resume | type `/goal resume`, Enter |
| Stop | `/goal pause`; exit Codex (Ctrl+C twice when idle); stop that home's app-server daemon; verify no TUI. The session is never deleted |
| Discard | Stop if the window is open, then `discarded` with the operator's reason |
| Re-send stance | paste `STANCE-MESSAGE.txt` again and Enter; it becomes one of the kept messages |

A control runs detached from the request for up to two minutes, so a dropped connection never
leaves half a stop; `LAUNCH.json` holds the outcome.

`/goal pause` and `/goal resume` do not count among the user messages kept at compaction:
Codex records each as a goal-status item, and compaction drops those along with goal
continuations while it keeps the stance (measured, `docs/internal/evidence/launch-proof.txt`).
Typed with Enter during a running turn, each acts within a second and is not queued as a chat
message; the turn runs to its end, after a pause no goal turn follows, and after a resume a
goal-continuation turn starts when the running one ends (measured on session 3 of the same
file).
Re-send stance does take a kept slot. Retention looks budget-based, not a count of the newest
eight: another real session (2cf9ccfd-8dbf-408e-bd7d-9cb046589f6c) kept all 24 typed operator
messages across seven compactions (`launch-proof.txt`); the charter's stance limit is
unchanged. Codex applies both commands to a goal in any state and
restarts a completed goal on resume, so the desk pauses only an active goal and resumes only
a paused one.

## 7. Operator file, courier and "seen"

`say` and `answer` append to `OPERATOR.md`. `enable-steering` (for charters without the
Steering line) appends `## Steering` and that one sentence to the charter `README.md`, only
on the operator's action; until then every entry returns a warning that it will not be read.
`answer`, `say` and `verdict` take `dry_run`: the response holds the exact text that would be
appended (`writer.RenderOperator`, `RenderVerdict`, the same function the append uses),
`written: false`, and for operator entries the commission's `entry_reading` (§4); the UI shows
that text and never rebuilds the file grammar.

**Needs you.** An item in a commission's `ATTENTION.md` is answerable there. A global item
(the home's own `ATTENTION.md`) is answerable into the commission its `Commission:` line names;
without that line it is never answerable and carries `instruction` (the text of its first
labelled line naming the operator's part, such as `Needs you:` or `Decision:`, else its first
line) and `links` (Markdown links in its body, targets resolved against the folder of
the file it was read from). Every delivery state carries `delivery_reason_code`
(`DeliveryReason`: scheduler delivered, answered, secret, no commission, no steering, no
recommendation, ask, unreadable, ID reused, switch off, scheduler grace, Hub unavailable, in
Hub, Hub delivered, Hub answered, orphaned, baseline, notice) beside the sentence; each courier
step carries the same code; clients match on the code, never on the prose.

**Courier.** Its rules (switches, what is sent and how, holds, reused IDs, unreadable
files, the pass) are stated once, in [FILE-CONTRACT §5](../reference/FILE-CONTRACT.md#courier).
The route is the one every scenario-originated decision takes (api-core
`operatordecision`): the desk publishes `research_desk.needs_you.decision_requested.v1` and
`.notification.v1` events to Vrooli Events, and Notification Hub's own subscriptions turn
them into an ask or a notice. No grant is involved; the Hub's `ListAsks` and notification
`List` are read without a credential (D21). **Courier health** in `now` is its real state:
the last pass, the last read-back, the last delivery.
**Seen** = `STATE.md` rewritten after
the entry ("a unit has completed since").

## 8. What changed since I last looked (G13)

`mark-seen` (`OperatorService.MarkSeen`, an operator action) records a look in the desk
state folder, `looks/<commission|topic>-<id>.json` (`at`, ledger byte lengths, checkpoint
paths and mtimes, workstream statuses per topic). `GetChanges` diffs the present against
the recorded look, or, with `since` (`changes … --since <UTC>`), lists checkpoints and files
written after that time; with neither it reports the last 24 hours (`home.DefaultWindow`) and
says so. `baseline_kind` is `look`, `asked` or `last_day`. `mark-seen --all`
(`MarkSeen` with `all`, an operator action) records a look at every commission and topic at
once and returns the time and the counts. Against
a look it reports new ledger text, changed checkpoints, lines
containing `showcase` in that new text, workstream status changes, and files under the
commission and its run folders modified since `at` (walk capped at 20,000 entries;
`truncated` when hit). Deleting the snapshots only resets "since".

## 9. Focus (G6) and ceilings (B5)

What each role reads is listed once, in `read_sets` of
`skills/research-coordination/templates/limits.json`. The desk (`readsets.go` in the API package `internal/home`)
measures those lists per commission, the commission's own files in place of `charter` and
bare `*.md` items, and shows bytes, estimated tokens (bytes / 4) and flags over the role
ceilings, the charter gate and dated amendment headings. `ceilings_test.go` fails when a
role's total exceeds its ceiling.

## 10. Review (G7)

The `research-review` skill writes `reviews/<id>/REVIEW.md`, an optional `REVIEW.html` with
embedded media, and `proposals/`; the desk lists reviews with links to both (served by the
file route) and appends verdicts to `VERDICTS.md`. The desk never writes `MISSION.md`.

## 11. Identity and instances

**Who may act for the operator (F13).** Launch, Control, the two self-tests
(`launch-test`, `courier-test`), courier switches, EnableSteering, Say, Answer, AddVerdict and
`mark-seen` refuse a caller verified as an agent (an agent identity token, or an
authenticated agent or service principal) with "this is the operator's action; write a
needs-you item instead" and its run (`api/internal/wire/caller.go`); the CLI marks them
`run_eligible: false` and prints the same refusal. A caller whose agent token cannot be
verified is refused too. Reads, Build and Preflight stay open to agents. The desk keeps a
verified caller's own identity; the availability fallback never replaces it. **Accepted
limit:** a local caller with no token is indistinguishable from the operator's shell.

**Web Console access (F14, DECISIONS D20).** The desk calls Web Console with its own service
credential when it has one, otherwise under Web Console's documented local availability
policy; it never forwards the operator's session token. The access used is recorded in
`LAUNCH.json` and shown in `/health` and `now` ("Web Console access: service credential" or
"local availability policy (no desk credential)"). When Web Console refuses, automatic
launch is unavailable with Web Console's words and manual launch is the whole flow; no grant
is widened. The service principal is prepared by `scripts/issue-service-principal.sh`, which
the operator runs. The courier needs no credential (§7).

**Instances (F7).** Only the lifecycle-managed live instance writes, types or runs the
courier: it has `VROOLI_LIFECYCLE_MANAGED=true`, a `VROOLI_RUNTIME_INSTANCE_ID` and the
`research-desk` storage namespace; a test, a boot check or a shadow variant only reads. The
live instance takes the lock (`~/.vrooli/state/vrooli/research-desk/instance.lock`, contract
§1) and retries it while a restart overlaps the old process; a read never takes it. Tests
point the desk elsewhere with `RESEARCH_DESK_RESEARCH_DIR`, `RESEARCH_DESK_STATE_DIR` and
`RESEARCH_DESK_CODEX_DIR`.

## 12. Proofs on the real home

| Goal / invariant | Proof |
|---|---|
| G1, A9 | `packet.py` over the topic sets of the three latest packets: constant text identical, gates catch programme-three's 36 KB charter and 4,211-character stance; `check` passes the prepared packets |
| G2, G12 | one throwaway session launched with a completing kernel, then pause, resume, stop, discard; `LAUNCH.json` has every step |
| G3, I6 | `commissions` over all 25 folders: no error; programme-three `paused` or `ended` by its TUI; prepared packets `prepared`; older folders `unknown` |
| G4, G5 | Hub read-back at the seam (unit tests: ask raised once, delivered entry, a Hub answer recorded once, an unneeded ask shown open until its deadline); one real `courier-test` ask; nobody answers an ask on the operator's behalf |
| G6, B5 | `commission programme-three-20261111` flags the charter; the ceilings test runs in the unit phase |
| G7 | a review of ai-video lists `REVIEW.md` and records a verdict verbatim |
| G8, G9 | see the skills guide |
| G10, G11 | phone screenshots at 390 px; `detemplate` leaves no example |
| G13 | two `changes` calls a unit apart on a commission the operator runs anyway |
| I1 | stop the desk for one review interval during such a commission; its ledger continues |
| I2, I4 | path, size and mtime snapshot of the home before and after a read-only CLI session: unchanged |
| I3 | hand-launched programme-three and programme-four match `exact` |
| I5 | dependencies pointed at closed ports: projection works, launch and courier say why not |
| I7 | ceilings test |

Out of scope: scheduling owners; writing `STATE.md`, `LEDGER.md`, checkpoints or missions;
deleting sessions or research files; rewriting older folders; non-Codex harnesses.
