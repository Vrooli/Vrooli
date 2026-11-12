# Decisions

Each choice, its alternatives, why, and the cost to switch. Evidence gathered 2026-11-11 on
the real research home; review rulings in `.vrooli/work/research-desk/REVIEW-1-RULINGS.md`.

**D1 A separate scenario, never in the loop.** Alternatives: Agent Manager goal homes, a Web
Console feature, skills only. Agent Manager refused programme-three's folder ("outside an
enrolled goal home"), so the AV4 decision never reached the operator; a Web Console
feature ties research to terminal restarts; skills alone leave the five-minute launch and
no "what is moving" view. Switch: low; everything persisted is a file.

**D2 No database.** Alternative: keep the template SQLite as an index. Top-level mtimes and
checkpoints give last activity without walking 3.3 million files; one Codex home's files
read in under a second. Switch: low; a cache behind `internal/home`.

**D3 `OPERATOR.md` for direction, answers and wind-down.** Alternatives: append to the
charter (programme-three reached 36 KB with an appended correction), write `STATE.md`
(breaks one writer), chat (replayed after every compaction). Older charters get one
Steering line only by explicit operator action. Switch: medium; template and writer change
together.

**D4 Courier = direct Hub `Ask` and `Send` + `ListAsks` read-back, off until chosen.**
Alternatives: supervisor delivers (today's fragile route), Agent Manager decisions, a
vrooli-events publish with a webhook back (the first design), the `operatordecision.Raiser`
with a pending store. The Hub stores the requester key as `requester_ref` and makes keys
idempotent, so the key `research-desk/<commission>/<ID>` is the only identity and nothing
needs storing; `ListAsks` reads answers back on every pass, so a webhook would only add a
second route, a signing secret and a dead-letter case (vrooli-events dead-letters after five
tries in about 31 s). A reworded item keeps its first Hub wording; an expired ask is answered
in the desk. The courier is off for every commission the desk did not launch and for global
items until the operator turns it on, because 25 older commissions and `ATTENTION.md` hold
old items that must never page the operator. Items whose scheduler delivers by itself are
held two hours for its `Delivered:` line. Switch: low; one package behind the `Hub` seam.
Replaced by D21 for sending: the Hub refused the desk's `Ask` (D14), so decisions go out as
events; the read-back and the off-until-chosen switches stand.

**D5 Commission state from evidence.** Alternatives: the status prose (stale: two folders
say ACTIVE and are ended), Web Console `native-thread-state` (only for managed sessions;
refused for another session). Goal status has six values (`active`, `paused`, `blocked`,
`usage_limited`, `budget_limited`, `complete`); supervisors never mark `complete`, so the
TUI process (found by `CODEX_HOME` in its environment) and the allowance end decide
`ended`. Switch: low; one function.

**D6 Launch an ordinary Codex TUI session through the governed launcher.** Alternatives:
managed Codex (`SetNativeGoal`), CLIs. The probe proved paste, `ready`, `/goal` and goal
verification from files; the TUI's app-server outlives Web Console restarts, a managed
one is Web Console's child; the governed launcher gives each session its own Codex home;
the HTTP API with `X-Idempotency-Key` avoids CLIs that rebuild themselves mid-launch.
Switch: medium; the terminal seam.

**D7 Control by `/goal pause|resume` and quitting; never delete sessions.** Codex 0.167
accepts `/goal [<objective>|clear|edit|pause|resume]`. Session delete removes the Codex home
and transcripts. These slash commands do not take a retained user-message slot: compaction
drops goal-status items (measured, `docs/internal/evidence/launch-proof.txt`). Switch: low.

**D8 Relaunch is a new commission.** Alternative: attempts inside one folder. Owner run
folders are keyed by commission id, so a second attempt would resume the bad attempt's
checkpoints. Switch: low.

**D9 One packet renderer: `packet.py` in the skill.** Alternative: a Go builder. One
renderer serves the desk and an agent with no desk; improving the skill improves both.
Switch: medium.

**D10 One needs-you grammar, `### ID — title`.** Alternatives: numbered items with `[ID]`,
hash ids. `ATTENTION.md` already uses this heading form; one parser; merge by ID. Items
need `Recommended:` to be sent; secrets never go into files. Switch: low.

**D11 "Seen" = `STATE.md` rewritten after the entry.** Alternative: parse rollout tool
calls. File time needs no transcript parsing and no supervisor duty. Switch: low.

**D12 "Since last look" via a desk snapshot file.** Alternatives: timestamps in ledgers
(formats vary; programme-four's ledger starts as a state file), browser storage (not
shared between phone and desktop). A deletable snapshot in the desk state folder
(`~/.vrooli/state/vrooli/research-desk/looks/`), recorded only by the operator's `mark-seen`
(an operator action, `OperatorService.MarkSeen`, with `all` for everything at once),
only resets "since"; it stays out of the research home so reading never writes there.
`changes --since <UTC>` needs no recorded look; with neither, the last 24 hours (P7).
Switch: low.

**D13 Skills move with compatibility links.** Alternatives: no links, keep in the pack.
Launched charters name the old paths; the pack loader skips a directory without
`skill.json`. Switch: low; remove the links when no running commission names them.

**D14 Identity: the desk speaks as itself, never with the operator's session.** First built
with the operator's local session (`LocalOwnerTokenProvider`). Measured 2026-11-11: Notification
Hub refused it for `Ask` ("verified owner or approved decision-asking grant is required";
`identity.DecisionSubject`), and Web Console admitted it only through its availability
fallback (owner `ui-fail-open`, `api-core/authn.UIAvailabilityFallback`), so it was verified
nowhere. Replaced by D20 and D21. Switch: low.

**D15 Only the live instance writes.** An exclusive lock file; others read. Prevents two
couriers and two writers. Switch: low.

**D16 Defaults (operator, unanswered questions).** Launch screen shows preflight and both
messages with one Launch button and manual copy beside it; stalled = 70 minutes without
writes (the constant `home.StallAfter`; a change is a code edit); a topic covered by a live,
paused or limited commission needs the operator's supersede or run-alongside choice (D22).

**D17 The spec passes through verbatim.** Alternative: a typed spec message in the launch
proto. `packet.py` owns the spec (FILE-CONTRACT §2) and validates every field; a typed copy
would be a second reader to keep in step with each skill change. The desk reads only
`topics` and `run_alongside` for its topic verdicts. Switch: medium (a proto change and the
UI form).

**D18 One Build RPC; relaunch is a source.** `Build` takes `spec_json` or `relaunch_of`; the
CLI is `build <spec-file>` or `build --relaunch-of <id>`, because one RPC has one command.
Alternative: a separate `Relaunch` RPC. Same script, same result; one fewer surface. Switch:
low.

**D19 The launch test is a product command.** `launch-test` (RPC `TestLaunch`, read back with
`GetLaunchTest`) runs the real launch path on one throwaway, labelled session with a fixed
harmless stance and a ~600-character kernel that completes at once, then pauses, resumes and
stops it, as `courier-test` does for the Hub. It runs in the background and keeps its report
in the desk state folder, because a launch outlives the API's 30-second write timeout;
controls likewise run detached from the request for up to two minutes, so a dropped phone
connection never leaves half a stop. Alternative: a script under `test/`. The operator
re-runs it after a Codex or Web Console upgrade without an agent. Switch: low.

**D20 Automatic launch through whichever Web Console access exists, shown (rulings F14).**
Tried in order: the desk's own service principal when its credential is present; otherwise
Web Console's documented local availability policy (`api-core/authn.UIAvailabilityFallback`:
anyone who can reach the API receives that access). With agents refused at the desk (F13,
`api/internal/wire/caller.go`) the policy gives the desk nothing a local operator process
lacks. The access used is recorded in `LAUNCH.json` (`web_console_access`) and shown in
`/health` (`metrics.web_console_access`), `research-desk now` and `GetLaunchOptions`; a
refusal from Web Console sets `automatic_available=false` with Web Console's words and
manual launch is the whole flow (`api/internal/peer`). The operator's session token is never
forwarded. The service principal is prepared, not requested:
`scripts/issue-service-principal.sh` (the operator runs it) adds the `research-desk` profile
(resource `web-console`, scopes `web-console:read` and `web-console:write`, issuers copied
from the `agent-manager` entry) to scenario-authenticator's `VROOLI_AUTH_SERVICE_PRINCIPALS`,
restarts scenario-authenticator and issues the credential into the credential authority.
The desk's client for it (read the credential, exchange it at
`AccountsService.ExchangeServiceCredential`, as
`agent-manager/api/internal/adapters/webconsole/service_credential.go` does) is not built:
installing the credential-authority reader was refused by the dependency owner
(`disk_admission_denied`, disk at 95%). Switch cost once that install clears: low; one
function, `webConsoleCredential` in `api/wiring.go`, plus its test; the access line then reads
"service credential" with no other change. Alternatives not taken: refuse automatic launch
until the credential exists (the first build; it left the operator with only the manual
path for no gain in authority); forward the operator's browser credential (no background
use; a launch would depend on an open browser).

**D21 The courier sends events and reads the Hub.** Decisions and notices go out as
`research_desk.needs_you.decision_requested.v1` and `.notification.v1` through Vrooli Events,
the route every scenario-originated decision takes; the Hub subscribes to both patterns and
its reads need no credential. Cost: the desk cannot withdraw an ask (the Hub's withdrawal
event accepts Agent Manager only), so an unneeded ask stays open until the Hub's default
deadline; the overview's courier line and `courier` say so (F15). Courier health (last pass,
last read-back, last delivery) is shown in `research-desk now`. One operator-triggered
self-test, `courier-test` ("send me a test decision"), stays beside `launch-test` because
the 2026-11-11 failure was a decision silently not arriving; it never runs by itself. Alternative not taken: an owner-scoped Hub grant for the desk, which would make the
desk the ask's recipient rather than the operator. Switch: low; `internal/courier/hub.go`.

**D22 Blockers are resolved in the request (rulings P2).** Alternatives: refuse and send the
operator to stop or discard the old commission first (the review-3 path: several screens and a
command line for one launch); let any `unknown` commission block (18 of 24 on the real home are
`unknown`, most quiet for days). An earlier commission blocks only when live, paused, limited,
or `unknown` with an open session or a write within 24 hours (`launch.QuietAfter`); the request
carries supersede or run alongside per blocker, and the desk preselects supersede only for a
paused or limited commission whose allowance has ended. A supersede stop passes the F5 checks
before anything is launched or it refuses. A superseded commission is marked discarded with
`superseded_by`, so it derives `ended`. Switch: low; `launch/preflight.go`.

**D23 The operator's assertion for an unprovable start (rulings P1).** Alternative: keep
`unknown` unlaunchable until a packet proves itself unchanged. A packet whose `STATE.md` was
touched after the build cannot prove it, yet the operator knows. The desk accepts the
operator's word only when its own search finds nothing (no session, no goal naming the
charter, no related history entry, no `LAUNCH.json`) and records the word with what was
searched in `LAUNCH.json` `never_started`. Switch: low; `home.neverStarted`.

**D24 The desk owns the words (rulings P5).** Operator entries and verdicts are rendered by one
function that both the dry run and the append use, and delivery reasons are an enum. Alternative:
the UI rebuilds the grammar and reads the prose (it drifted once; review 4). Switch: low.

## Rulings not followed as written

- **P2 "the new charter records it" (review 4)**, narrowed. Choices made before the packet is
  built go to `Build` and into the spec, and `packet.py` renders them in the charter. The
  exception is a packet launched as it already is (prepared or hand-made) with a supersede
  chosen at Launch: the desk does not rewrite `README.md`, so that supersession is in both
  `LAUNCH.json` files only (`supersedes`, `superseded_by`) and in the one confirmation.

- **"About twelve" RPCs (review 1).** The desk has 23: activity 5, launch 9, operator 9. The
  departure is the goals added after that ruling: G12 (launch and control: `Build`,
  `Preflight`, `Launch`, `GetLaunch`, `Control`, the `TestLaunch`/`GetLaunchTest` self-test)
  and G13 (`GetChanges`, `MarkSeen`), plus the courier switch and its self-test
  (`SetCourier`, `TestCourier`), and review 4's `GetLaunchMessages` (P4, G1: the two messages for a manual launch are a
  read, not a mode of `Launch`). Review 4's "caught up" (P7, G13) is `MarkSeen` with `all`, not
  its own RPC. `GetLaunchMessages` stays apart from `GetLaunch`: one reads the packet's two
  messages, which exist before any attempt; the other reads the attempt's `LAUNCH.json`. Each
  maps to one goal and one CLI verb; review 3 (E6) accepted the count on that basis.
- **E6 "Remove `TestCourier`" (review 3)** was reversed by F15: `courier-test` stays, named a
  self-test beside `launch-test`, because the 2026-11-11 failure was a decision silently not
  arriving.
- **A6 "one Hub subject per item"** is met by one decision event per item whose question
  leads with `<commission> · <ID>` (no `Raiser`, so no per-subject cap applies).
