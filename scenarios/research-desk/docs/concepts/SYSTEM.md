# System Design

The one design document. File shapes, the courier's rules and the desk's state files are in
[`../reference/FILE-CONTRACT.md`](../reference/FILE-CONTRACT.md); choices, alternatives and
switch costs in [`../DECISIONS.md`](../DECISIONS.md).

## 1. Shape

A Go API, a thin CLI and a phone UI over it. The API reads the research home and the Codex
session homes (`~/.vrooli/state/vrooli/web-console/sessions/codex/<session>/`), runs the
skill scripts, types into Web Console sessions only on the operator's action, and carries
decisions to and from Notification Hub. No database: a memory cache keyed by path, size and
mtime. `api/internal/home` parses everything the desk shows and returns each derived field
as `{value, known, source_path, observed_at}`. Three other readers each have one purpose:
`api/internal/writer` reads a file it appends to (and a charter for the Steering line),
`api/internal/launch` reads the stance and kernel it types, and the skill scripts
(`packet.py`, `mission_check.py`) parse packets and missions. `api/internal/writer` makes
every desk write.

The optional peers (web-console, vrooli-events, notification-hub) are declared in
`.vrooli/service.json`; the capability registry says whether each can be reached, and the
desk dials one only then, with a timeout. Whatever is unavailable, `python3` included, leaves
its fields `unknown`; a check that times out says "not reached just now", and a start command
is offered only for a scenario its lifecycle does not list running.

## 2. Surface

activity: `GetOverview`, `ListCommissions`, `GetCommission`, `ListTopics`, `GetChanges`.
launch: `GetLaunchOptions`, `Preflight`, `Build`, `GetLaunchMessages`, `Launch`, `GetLaunch`,
`Control`, `TestLaunch`, `GetLaunchTest`. operator: `ListNeedsYou`, `Answer`, `AddEntry`,
`EnableSteering`, `ListReviews`, `AddVerdict`, `SetCourier`, `TestCourier`, `MarkSeen`. The
CLI has one verb per RPC (`research-desk help`; `cli/manifest.json`). `ListNeedsYou` is also
the courier's dry run. REST exceptions: `GET /health` and the home file route
(FILE-CONTRACT §1). Protos: `packages/proto/schemas/research-desk/v1/`; `make endpoints` generates
`.vrooli/endpoints.json`.

**Codes for clients, words for people.** Clients branch on codes, never on sentences:
`DeliveryReason`, `EntryReading`, `WebConsoleAccess`, `DerivedTime.at_least`, the capability
`status` (`available`, `unavailable`, `unknown`), the changes `Baseline`, and each blocker's
`recommended`. A sentence says what is wrong and what the operator can choose and names no
command; a command-line next step travels beside it (`shared.CommandHint` on an error,
`Lens.command` on a lens), printed by the CLI as "(next: …)". Limits come from the skill's
`limits.json` through the API (`packet_limits`, `entry_chars_max`), never from a copy.

## 3. Recognition

**exact**: `LAUNCH.json` names the session, or the commission's absolute charter path appears
in a `thread_goals.objective` (any status; `goals_1.sqlite` opened `?mode=ro`) or a
`history.jsonl` entry starting `/goal ` in any Codex home. An objective that points at
`<home>/attachments/<id>/goal-objective.md` is read from that file. **related**: any other
`history.jsonl` entry naming the folder; shown, never used for state. State uses the newest
exact match; only matches from a day before the packet's earliest file onward count. No match: `unknown`; transcripts are not searched.

## 4. Commission state

Derived on every read, never stored, from: packet files, `LAUNCH.json`, the exact thread's
goal status, the TUI process (a non-app-server `codex` whose environment has
`CODEX_HOME=<home>`), the newest write (commission files, the home's newest rollout,
`topics/<t>/runs/<commission>/CHECKPOINT.md`) and the allowance end (goal `created_at_ms`
plus the allowance in `KERNEL.txt` or the spec).

| State | Rule, first match wins |
|---|---|
| `ended` | discarded in `LAUNCH.json`; goal `complete`; or allowance passed and no TUI |
| `prepared` | complete packet, no match, no `LAUNCH.json`, no goal naming the charter in any readable Codex home, `STATE.md` and `LEDGER.md` unchanged since the build |
| `unknown` | no exact match |
| `stopped` | no TUI, goal not `complete`, allowance not passed |
| `live` | TUI, goal `active`; `stalled` after 70 minutes without a write |
| `paused` | TUI, goal `paused` |
| `limited` | TUI, goal `usage_limited`, `budget_limited` or `blocked` |

**The operator's word.** An `unknown` commission carries `never_started`: it may launch on
the operator's word only when the packet is complete, there is no `LAUNCH.json`, and no
match or goal naming the charter was found in any Codex home the desk can read (unreadable
homes are named).

**When an entry is read** (`entry_reading`): `never` (ended), `not_until_steering`,
`next_unit` (live), `after_resume` (paused or limited), `at_start` (prepared or stopped), else
`unknown`.

The supervisor's status line is shown beside the state as a claim. Two commissions overlap
when they share a topic and their intervals intersect (start to now while a window is open,
else to the known end, else to the last write: "stopped or ended (allowance unknown)");
`open_overlaps` are the pairs open now. A
topic reports its newest run-folder write (with its commission), `mission` and `last_review`;
a write its commission's state does not explain carries a note.

## 5. Launch

**Preflight** gates a spec or a commission (`packet.py preview` or `check`) and gives each
topic a verdict over every commission naming it (found from the `PACKET.json` spec, a
`### Card` link, a charter link, or run folders `topics/<t>/runs/<commission>[-<suffix>]/`).

**Blockers and supersession** (the one statement of these rules). An earlier commission
blocks when it is `live`, `paused` or `limited`, or `unknown` with an open or unchecked
session, an unknown last write, or a write in the last 24 hours; an older quiet `unknown` is
listed as `quiet`. Each blocker offers `supersede` and `run_alongside` and a `recommended`
choice with its reason: supersede when paused or limited past its allowance; run alongside
when `unknown` with no session or goal found ("nothing to stop; it stays launchable"); none
otherwise. The request resolves each blocker; the spec's `run_alongside` and `supersedes`
count as choices and the request wins. `stopped` and `ended` commissions block nothing (their
newest checkpoint is offered as `read_first`). A commission that is itself `unknown`
launches only on the operator's word. Choices made before a packet exists go to `Build`,
which writes them into the spec, so the charter names them. Before anything launches, every
superseded commission with an open window must pass the checks of §6, or the whole launch is
refused naming it; the launch then stops each through its own session (a failure is step
`supersede`, nothing new starts) and marks it discarded with `superseded_by`, so it derives
`ended`. A packet launched as it is records a supersede chosen at Launch in the two
`LAUNCH.json` files only; the desk never rewrites `README.md`.

**Suggestions.** `GetLaunchOptions` with topics returns stance quotes, as written, from the
newest desk-built `PACKET.json` naming them, else the quoted lines of the newest started
one's `STANCE-MESSAGE.txt`, else none and why; plus workstreams and `limits.json` defaults.
Preflight also gives `owner_reading` and, for an existing packet, `packet_state` (its
`STATE.md`, first 12 lines or 800 characters, verbatim); neither blocks. **Build** runs `packet.py build`. **Relaunch** (`build --relaunch-of <id>`) rebuilds
from the old `PACKET.json` spec with `supersedes` set and `carry_baseline_from` when the old
commission has a `BASELINE.json`.

**Automatic launch** (`launch/launcher.go`) runs in the background through Web Console's HTTP
API with `X-Idempotency-Key: <commission>#1`: a persistent session running
`vrooli-agent-launcher --agent codex -- --yolo -m <model> -c model_reasoning_effort=<effort>`
(its own `CODEX_HOME`); paste the stance; wait for `ready`; type `/goal <kernel>`; verify the
goal is `active` with the kernel as objective. Each Enter waits one second and the input row
is read back ([Codex measurements](../internal/evidence/codex-measurements.md)). Steps go to
`LAUNCH.json` (FILE-CONTRACT §3); nothing else is sent. **Manual**: `GetLaunchMessages` gives
the charter path, stance and kernel to copy. **Launch test** runs the automatic path on one
labelled throwaway session with a completing kernel, then pauses, resumes and stops it.

## 6. Control

Only on the operator's action, only for an exact match with a TUI, through Web Console
terminal input; each is recorded in `LAUNCH.json` `controls`. Before typing, the session must
show the commission's own thread with the cursor in Codex's input row and no choice open, or
the control is refused. Pause and Resume may go in during a turn; Re-send stance and Stop
wait for an idle input row.

| Action | Mechanism |
|---|---|
| Wind down | `OPERATOR.md` entry `wind-down`, read at the next unit |
| Pause, Resume | `/goal pause` or `/goal resume`, Enter; only an active goal is paused, only a paused one resumed |
| Stop | `/goal pause`; Ctrl+C twice when idle; stop that home's app-server; verify no TUI. Never delete the session |
| Discard | Stop if open, then `discarded` with the operator's reason |
| Re-send stance | paste `STANCE-MESSAGE.txt` again, Enter |

A control runs detached from the request for up to two minutes, so a dropped connection
never leaves half a stop.

## 7. Operator file and courier

`say`, `answer` and `verdict` append in the grammar of FILE-CONTRACT §4; `dry_run` returns
the exact text. `enable-steering` adds the Steering line to an older charter, only on the
operator's action (`dry_run` shows text and path); until then each entry warns that it will
not be read. A global needs-you item is answerable into the
commission its `Commission:` line names; without one it carries `instruction` and `links`.
An answer is **read** by the supervisor once `STATE.md` is rewritten after it. The courier
(rules: FILE-CONTRACT §5, "Courier") publishes `research_desk.needs_you.decision_requested.v1`
and `.notification.v1` to Vrooli Events; Notification Hub turns them into an ask or a notice
and is read back without a credential. `now` shows the courier's last pass.

## 8. Since last look

`mark-seen` records the operator's look (ledger lengths, checkpoint mtimes, workstream
statuses); `--all` records every one. `GetChanges` compares with the look, or with `since`,
or covers the last 24 hours, and says which (`baseline_kind` `look`, `asked`, `last_day`):
new ledger text and its `showcase` lines, changed checkpoints, workstream status changes and
modified files (walk capped at 20,000 entries, `truncated` when hit). Without a look, ledger
text is the sections under headings dated after `since`, and `not_shown` names what needs a
look.

## 9. Focus and review

The desk measures each role's read set per commission against `limits.json` `read_sets`;
`ceilings_test.go` fails when a role is over its ceiling. An owner's total includes its
topic's `MISSION.md` (`owner_topic`), held not to the ceiling but to a `mission_bytes` flag.
The `research-review` skill writes
`reviews/<id>/REVIEW.md`, an optional `REVIEW.html` and `proposals/`; the desk lists reviews,
reads their topics from `## <topic>` headings or `topics/<topic>` links (else unknown), and
appends verdicts to `VERDICTS.md`.

## 10. Identity and instances

**Who may act.** Reads, `Preflight`, `Build` and `GetLaunchMessages` are open to every
verified caller, so an agent's `Build` writes a packet folder into the real home. `Launch`
and `Control` take an agent only with an approved `research-management` grant
(`research.launch` or `research.control` on `research-desk:commission/<id>`), checked for
every superseded commission before any effect and again in the background run; without one
the shared `vrooli agent authorize` request is returned. Every other action refuses agents.
`cli/manifest.json` marks exactly those commands not runnable (a test keeps them equal). A failed
verification never falls back to operator access; a local caller with no token passes as the
operator (accepted; PROBLEMS.md).

**Web Console access.** The desk's service credential when it has one, else Web Console's
local availability policy; the operator's token is never forwarded. Shown as
`WebConsoleAccess` (`SERVICE_CREDENTIAL`, `AVAILABILITY_POLICY`, `UNAVAILABLE`). A refusal
turns automatic launch off with Web Console's words; manual launch remains.

**Instances.** Only the lifecycle-managed live instance writes, types or runs the courier,
under the lock in FILE-CONTRACT §1. Tests point the desk elsewhere with
`RESEARCH_DESK_{RESEARCH,STATE,CODEX}_DIR`.

## 11. Proofs and scope

Each requirement's validation is in `requirements/01-must-ship/module.json`; evidence from the
real home is in `docs/internal/evidence/` and `make real-home-check` re-runs it. Out of scope:
scheduling owners; writing `STATE.md`, `LEDGER.md`, checkpoints or missions; deleting sessions
or research files; rewriting older folders; non-Codex harnesses.
