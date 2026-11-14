# Decisions

Each standing choice: what was chosen, the alternatives, why, and the cost to switch.
Evidence: the real research home, 2026-11-11 and 2026-11-12.

**A separate scenario, never in the loop.** Alternatives: Agent Manager goal homes, a Web
Console feature, skills only. Agent Manager refused programme-three's folder, so a decision
never reached the operator. A Web Console feature ties research to terminal restarts. Skills
alone give no one-tap launch and no view of what is moving. Switch: low; everything kept is a
file.

**No database.** Alternative: keep the template SQLite as an index. Top-level mtimes and
checkpoints give last activity without walking 3.3 million files. Switch: low; a cache
behind `internal/home`.

**`OPERATOR.md` for direction, answers and wind-down.** Alternatives: append to the charter
(programme-three reached 36 KB that way), write `STATE.md` (two writers), chat (replayed
after every compaction). Older charters get the Steering line only on the operator's action.
Switch: medium; the template and the writer change together.

**Commission state from evidence.** Alternatives: the supervisor's status prose (stale), Web
Console `native-thread-state` (managed sessions only). Supervisors never mark a goal
`complete`, so the TUI process and the allowance end decide `ended`. Switch: low; one
function.

**An ordinary Codex TUI through the governed launcher.** Alternatives: managed Codex
(`SetNativeGoal`), the CLIs. A TUI's app-server outlives Web Console restarts; the governed
launcher gives each session its own Codex home; the HTTP API with an idempotency key avoids
CLIs that rebuild mid-launch. Switch: medium; the terminal seam.

**Control by `/goal pause|resume` and quitting; sessions are never deleted.** Alternatives:
ask the supervisor in chat (it takes a kept message slot and depends on obedience), the
managed-session goal API (not available for a TUI session), deleting the session (removes the
Codex home and transcripts). The two slash commands act during a turn and take no kept slot
([measurements](internal/evidence/codex-measurements.md)). Switch: low; `Launcher.Pause`,
`Resume` and `Stop`.

**Relaunch is a new commission.** Alternative: attempts inside one folder. Owner run folders
are keyed by commission id, so a second attempt would resume the bad one's checkpoints.
Switch: low.

**One packet renderer, `packet.py`, in the skill.** Alternative: a Go builder. One renderer
serves the desk and an agent with no desk. Switch: medium.

**The spec passes through verbatim.** Alternative: a typed spec message. `packet.py`
validates every field, and a typed copy would be a second reader to keep in step. The desk
reads only `topics`, `run_alongside` and `supersedes`. Switch: medium (proto and UI form).

**One needs-you grammar, `### ID — title`.** Alternatives: numbered items, hashed ids.
`ATTENTION.md` already used this form; one parser; merge by ID. Switch: low.

**"Read" means `STATE.md` was rewritten after the entry.** Alternative: parse rollout tool
calls. A file time needs no transcript parsing and no supervisor duty. Switch: low.

**"Since last look" from a desk snapshot.** Alternatives: ledger timestamps (formats vary),
browser storage (not shared between phone and desktop). A deletable snapshot outside the
research home, recorded only by `mark-seen`, so reading never writes there; with none, the
last 24 hours, ledger text only under dated headings. Switch: low.

**Skills moved with compatibility links.** Alternative: no links. Launched charters name the
old paths. Switch: low; remove the links when no running commission names them (PROBLEMS.md).

**Only the live instance writes.** Alternatives: every instance writes under per-file locks
(two couriers would both append), a leader chosen through a peer (a new dependency). An
exclusive lock file; other instances read. Switch: low; `writer.Held`.

**Stalled after 70 minutes; the launch screen shows preflight and both messages.**
Alternatives: a stall threshold in `limits.json` or per commission, a launch wizard. A unit
rarely runs past an hour, and one screen keeps the manual copy beside the Launch button.
Switch: low; the constant `home.StallAfter`.

**One RPC per goal and verb (23).** Alternative: about twelve RPCs with modes. Each RPC maps
to one CLI verb, and the UI calls each. Switch: medium.

**One Build RPC; relaunch is a source.** Alternative: a `Relaunch` RPC. Same script, same
result, one fewer surface. Switch: low.

**The launch and courier tests are product commands.** Alternative: scripts under `test/`.
The operator re-runs `launch-test` after a Codex or Web Console upgrade, and `courier-test`
because the first failure was a decision silently not arriving. Neither runs by itself.
Switch: low.

**Automatic launch through whichever Web Console access exists, shown.** The desk's service
principal when its credential is present, else Web Console's local availability policy;
alternatives: refuse automatic launch until the credential exists (manual only, for no gain
in authority), forward the operator's browser token (needs an open browser). The access is
recorded and shown as an enum. The credential client is not built (PROBLEMS.md). Switch: low;
`webConsoleCredential` in `api/wiring.go`.

**Launch and Control for agents through shared operation grants.** Alternative: refuse every
agent (the first build), or a desk-local approval list. The operator directed the standard
approval route: the `research-management` policy and the canonical evaluator, with control
coverage for every superseded commission. Other operator actions stay operator-only. Switch:
low; `wire.ResearchOperation`.

**The courier publishes events and reads the Hub back; off until chosen.** Alternatives: the
supervisor delivers (fragile), Agent Manager decisions, the Hub's `Ask` directly (the Hub
refused the desk), an owner-scoped Hub grant (it would make the desk the recipient). Events
are the route every scenario decision takes; the Hub's reads need no credential. Cost: an ask
cannot be withdrawn and stays open until its deadline. Off for commissions the desk did not
launch, because older files hold items that must never page the operator. Switch: low;
`internal/courier/hub.go`.

**One first-sightings record for the courier.** Alternative: a per-switch item list plus
separate first-seen and first-held files (three records of one fact). Present-before-switch,
the two-hour wait and reused IDs all follow from when an item was first found. Switch: low;
`internal/courier`.

**Blockers are resolved in the request.** Alternatives: refuse and send the operator to stop
the old commission first (several screens for one launch); let any `unknown` commission block
(most of the home's are `unknown` and quiet for days). Rules: SYSTEM §5. A packet launched as
it is records a supersede in `LAUNCH.json` only, because the desk never rewrites a charter.
Switch: low; `launch/preflight.go`.

**The operator's word for an unprovable start.** Alternative: keep such a commission
unlaunchable. A `STATE.md` touched after the build cannot prove the packet unstarted, yet
the operator knows. The word is accepted only when the desk's own search finds nothing, and
is recorded with what was searched. Switch: low; `home.neverStarted`.

**The desk owns the words and the codes.** Operator entries and verdicts are rendered by the
one function the dry run and the append share, and every state a client branches on has a
code. Alternative: the UI rebuilds the grammar and reads prose (it drifted once). Switch: low.

Replaced: sending decisions through the Hub's `Ask` → "The courier publishes events";
speaking with the operator's local session → "Automatic launch through whichever Web Console
access exists" and "The courier publishes events"; refusing every agent at Launch and Control
→ "Launch and Control for agents through shared operation grants".
