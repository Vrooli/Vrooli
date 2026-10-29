# Effort supervision and delivery orchestration contract

Agent Manager's existing `agent_manager.v1.AgentManagerService` owns the effort
board RPCs. The canonical schema is
`packages/proto/schemas/agent-manager/v1/domain/effort.proto`; generated Go,
TypeScript and Python messages use the same contract. No plan family is required.
The board is an observation projection. Supervision and steering of a delivery
goal happen through its goal home files (`FEEDBACK.md`, epoch `## Directives`);
see `large-effort-orchestration` and `docs/agent-system/EFFORT_SUPERVISION.md`.

| CLI operation | RPC | Request message |
| --- | --- | --- |
| `effort board` | `GetEffortBoard` | `GetEffortBoardRequest` |
| `effort compact` | `GetEffortBoard` | `GetEffortBoardRequest` |
| `effort list` | `ListEfforts` | `ListEffortsRequest` |
| `effort discover` | `ReconcileEffortDiscovery` | `ReconcileEffortDiscoveryRequest` |
| `effort enroll` | `EnrollEffort` | `EnrollEffortRequest` |
| `effort reconcile-metadata` | `ReconcileEffortMetadata` | `ReconcileEffortMetadataRequest` |
| `effort withdraw` | `WithdrawEffort` | `WithdrawEffortRequest` |

Invoke these with `agent-manager` before the operation. Read operations accept
`--json`, `--page-size` (1–100) and `--page-token`; board operations accept
`--effort-ref`. Mutation operations accept `--request-file PATH` containing the
whole typed request, not only its nested enrollment. For example:

`effort compact` is a bounded presentation of the same one-call board join. Its
text output shows selected efforts, changed evidence references, owner usage,
provider quota observations and named waits with detail references. `--json`
returns the complete typed `EffortBoard` response from that same call.
Shared quota observations appear once at the owner-cut level; run-attributable
quota observations remain on their effort row.

```json
{
  "enrollment": {
    "effortRef": "effort:opaque-owner-identity",
    "displayName": "A bounded investigation",
    "destinationRef": "doc:accepted-destination",
    "targetRevision": "accepted-revision",
    "workShape": "investigation"
  },
  "expectedRevision": "0",
  "idempotencyKey": "unique-enrollment-operation"
}
```

To amend, pass the current enrollment and its
`revision` as `expectedRevision`. Reusing an idempotency key with different content
conflicts. Withdrawal requires `effortRef`, `expectedRevision`, `reason` and
`idempotencyKey`; it survives rediscovery. Automatic observation needs neither
operator credentials nor enrollment calls; explicit reconciliation is operator-only.

Provider quota observations are also readable through the existing pricing
owner:

```bash
agent-manager subscription quota list --provider openai --pool primary --json
```

The equivalent endpoint is `GET /api/v1/pricing/quota-observations` with
optional `provider`, `pool`, `window`, and `limit` (1–100) filters. Rows retain
the source run, provider pool/window, observed time, ingestion freshness,
provenance, reset time, and provider-reported percentage where available.
Missing absolute ceilings remain unknown; token totals and estimated dollars
are never converted into quota use. Native frames currently lack a credential
or account identifier, so their uncertainty explicitly says account scope is
unavailable; observations from different credentials must not be combined.

## Receipt-backed journey gates

`effort epoch-check <brief> [--acceptance] [--json]` resolves a slice cell
`gate G2=receipt workflow-health:<run-id>[,<run-id>...]` against Workflow Health's
durable Get operation. Set `- Scenario: <slug>` in the brief (or `--scenario`)
and declare the gate as `G2 journey: bas/cases/one.json,bas/cases/two.json @ workflow-health`.
Every exact case must have a passing actual execution and current owner-assessed
dependency identity. The newest supplied outcome for a case wins; unknown/stale,
failed, skipped, dry-run and leaked-isolation evidence cannot pass. Manual counts
and prior ACCEPTED prose cannot override this evidence. Other gate types and
independent acceptance retain their existing semantics. Readers without the
owner resolver leave receipt-backed gates unmet, never implicitly passing.

The provider's immutable result remains historical truth; current applicability
is a read projection. See Workflow Health's runbook for dependency scopes and
conservative fallback. This command makes no validation launch or source write.

## Authority

General enrollment, withdrawal and explicit discovery require the existing verified human
owner principal. Team-backed enrollment also accepts the current verified team
manager through Prompt Manager's existing `GetTeam` owner decision. The read
forwards the verified caller and returns `managementAuthorized` plus the current
team revision. The effort must match the finite delivery team's declared effort,
active leader and local product home. Initial enrollment is inert and contains
only that declared leader. Subsequent team management cannot replace another
actor's enrollment, revive withdrawal, replace subjects, or rewrite commissioned
metadata; the existing delivery state machine owns clocks and budget limits.
A team label or authored authority reference is not this owner decision. CLI invocation middleware forwards `VROOLI_AGENT_IDENTITY_TOKEN`
as `X-Agent-Identity-Token`; it is verified by the run identity owner and never
substitutes for an operator credential. Token strings must not be placed in
request JSON or evidence.

Local operators can explicitly add `--local-owner` to a mutation (or `discover`).
This exchanges the existing Unix machine-principal binding through api-core's
authenticator client for that one request. It is refused inside an identified
agent run.

`CreateRun.requestedScopes` is an optional typed narrowing with `scopes` and
`expectedOwnerSubject`. Its presence requires a verified owner credential. AM
refuses a wrong expected owner, expired credential, or scope outside that owner's
ceiling before reserving or dispatching a run.

Metadata reconciliation is intentionally a separate, lower-impact mutation. It
may update the display name, destination, source revision, target revision,
workspace and work shape. It cannot change the effort reference, subjects,
authority or withdrawal state. It uses the same expected-revision and
idempotency protections. Operators may call it with the owner credential; an
identified family-parent run may call it only when its verified identity is an
exact `orchestrator` subject on the effort and its attenuated claims contain
`agent-manager:effort-reconcile`. Worker and supervisor identities are refused.
The request must set `authority` to `WATCH_AUTHORITY_FAMILY_PARENT` for the run
identity path. This capability is suitable for Prompt Manager coordinator
wakes; it is not a general team or workspace permission.

For a human-owned reconciliation, use the explicit local exchange when no
configured owner token is available:

```bash
agent-manager effort reconcile-metadata --local-owner --request-file metadata.json --json
```

For a coordinator run, set `authority` to `WATCH_AUTHORITY_FAMILY_PARENT` in
`metadata.json`; the inherited `VROOLI_AGENT_IDENTITY_TOKEN` is forwarded as
the run credential and `--local-owner` must not be used.

An interactive session (an operator's attached coding session) can carry the
owner's metadata authority for one named effort through an owner-approved
grant. The grant names one effort and expires within 8 h; the owner confirms
it in the browser:

```bash
vrooli agent authorize <session-or-run> --policy effort-metadata \
  --target agent-manager:effort/<name> \
  --approval-base https://<authenticated-agent-manager-host>
agent-manager effort reconcile-metadata --request-file metadata.json --json
```

`<name>` is the effort reference without its `effort:` prefix. The request keeps
`authority: WATCH_AUTHORITY_OPERATOR`. Agent Manager checks the
`effort.reconcile` operation against that exact target, logs the session, owner,
grant and approver, and applies the same metadata-only rule. Other policies
(for example `agent-management`) and grants naming another effort are refused.

**Lifecycle lineage.** A run may wake its parked parent, and a run whose
profile declares `agent-manager:orchestrate` may stop, continue or wake its
direct children. Lineage is also effort-level: a running root run with that
scope that holds effort E as `orchestrator` may do the same to a run holding E
as `worker` whose parent held E as `orchestrator`. So a check-in can steer the
workers an earlier check-in started. A child inherits its parent's verified
effort references as `worker`; `ParentRunID` stays the provenance link.

## Native delivery commission

The existing enrollment owns `delivery`; `effort delivery` composes its revision-checked
operator mutation rather than creating another store or authorization path.

```bash
agent-manager effort delivery start --effort-ref <ref> --commission <new-id> --budget 24h --workers 2
agent-manager effort delivery pause --effort-ref <ref> --reason "named hold"
agent-manager effort delivery resume --effort-ref <ref>
agent-manager effort delivery stop --effort-ref <ref> --reason "operator handback"
agent-manager effort board --effort-ref <ref> --json
```

An eligible human can use the existing operator path. The current team manager can
commission the declared team enrollment that it prepared through the authenticated
team-owner path, subject to the existing delivery transitions and budget fences.
Other agents use the same commands with an explicitly approved
`delivery-management` / `effort.delivery` rule for the exact
`agent-manager:effort/<name>` target. For that delegated rule, the native handler retains the agent and approving
owner separately and permits only commission changes on an existing non-withdrawn
enrollment. It refuses creation or changes to subjects, destination, owner, authority
and withdrawal. No human shell command is part of delegated activation. `--local-owner`
remains only for an eligible local human shell; agents do not use it.

Metadata reconciliation cannot change a commission. General enrollment uses its
human owner path; the team-specific path is limited to the exact declared team,
leader and effort, and cannot replace another actor's enrollment.
New commissions
require terminal retained owners; source edits and heartbeat enablement start no clock.
Pause requires a reason and parked/terminal owners. Disabling heartbeat alone pauses
neither retained execution nor time. Blocked time counts unless explicitly paused.

For a product team activation, its verified creator starts the bounded native
commission through the team-owner path, then enables its configured heartbeat.
A non-creator uses the applicable existing owner or exact operation-grant path. Read back both owners and verify native execution.
If a later activation step fails, report or deliberately pause the running commission
through its owner; a disabled heartbeat does not stop the clock. Approval issuance,
product acceptance and unrelated run-creation authority remain separate operations.

| Budget consumed | Native pressure | Behavior |
| --- | --- | --- |
| <75% | ON TRACK | Deliver full acceptance intent |
| 75% | APPROACHING | Stop optional polish; prioritize remaining value |
| 100% | OVER | Prioritize intent; explicitly defer optional work |
| 150% | CONVERGE | Secure coherent partial delivery; admit no new scope |
| 200% | HARD STOP | Fence admission/queued execution; checkpoint and hand back |

The clock survives service restarts and accounts for explicit pauses. A 24h target
has a 48h hard deadline. The native owner gives two minutes from actual stop
observation for checkpointing, including late observation after restart, then stops
remaining commissioned runs and verifies terminal readback before freezing outcomes.
The decision offers accept-partial, recommission or rescope. Publication, answer and
acceptance remain distinct facts; no time threshold accepts a product or lowers gates.
A stopped commission cannot be resumed by heartbeat or a new continuation.

The delivery-stop decision names the product and effort and supplies both existing
operator explanations. “Why this matters” describes the stop, reported progress
and surface states, and the consequences of accepting partial work, requesting a
new commission or changing scope. “Technical details” includes the commission,
clock, source/target revisions, frozen observation time, surface targets, active
unit reports, the recorded handoff explaining remaining work, operator requirements,
observation limitations and CHARTER/STATUS/LOG
evidence references. Missing measurements remain unknown. Workspace reports do
not establish independent acceptance or current runtime state.

The outcome is persisted before its decision is published. Publication and
finalization retries reuse that outcome and any existing decision unchanged;
later workspace edits cannot rewrite the operator's question. Explanation text
uses the shared decision contract's Unicode limits and explicitly marks omitted
content. Published historical decisions retain their original content and answer
state. Both Notification Hub and Web Console render the same explanation fields.

The Efforts UI reads the same board: live surface states and active units,
commission/pressure, and frozen terminal outcome with its operator decision ID.
Missing observations remain unknown. A frozen outcome is separate from later live edits.

## Delivery orchestration commands

These commands serve the epoch model (`large-effort-orchestration`): one
long-lived orchestrator per goal. Independent units may execute concurrently within
the native commission limit; team heartbeat concurrency is a separate boundary.

| Command | Contract |
| --- | --- |
| `run continue <id> --message <text>` | The one way to talk to a run. A running interactive session receives the text as its next user message (the harness queues it behind the current turn); per-turn overrides are refused there. Any other continuable run starts a new turn. A codec-pipe run that is still mid-turn is refused; write an epoch directive instead. |
| `run park <id> --producer children --key <orchestrator-run-id> --timeout 1h` | Parks the calling run until a direct child exits, the timeout elapses, or a wake names the key. The wake message names only the children that ended since the last wake and the ones still active (`[wake: timer] …` or `[wake] …`). |
| `run wake --key <key> [--producer children] [--result <text>]` | `POST /api/v1/runs/wake-by-key`: the server matches every parked run whose await handle has exactly this producer and key and wakes it. A run identity may wake only runs its lineage permits (its parent, or its own children under the orchestrate scope). `run wake <id>` wakes one run. |
| `usage run <id> [--children] [--weights sol=10,luna=1]` | Non-cache tokens weighted by model tier (operator heuristic `operator-non-cache-v2-2026-10-27`: Sol 10, Terra 5, Luna 1; unmatched models are unknown and disable spend comparisons). This is non-cache-token policy, not billing or measured subscription quota. Usage that is not final is listed, never counted as zero. |
| `effort epoch-check <epoch-file> [--acceptance] [--runs ids] [--spend-threshold n] [--wake-key key]` | Parses an epoch file and prints the step-back triggers, typed exit gates and measured yield. It exits 3 for a flatline strategy review; estimate overruns report optimization and continue with exit 0; `--wake-key` wakes the orchestrator parked on that key (producer `children`). An accepted epoch never exits 3 or wakes. `--acceptance` is the orchestrator's check (exit 4). Spend uses the same weighting as `usage run`. Slice-log lines need `<time> \| <what changed> \| exit metric=<v>`; line counts default to zero, unknown `key=value` fields are ignored, and `ack=` takes free text whose `D<n>` IDs count as acknowledged. See [Goal-home commands](#goal-home-commands). |
| `effort lint <goal home>` | Checks the goal-home rules the orchestrator owns; exit 4 on a failing rule. |
| `effort handoff set <goal home> (--file F \| --stdin)` | Replaces the single `## Handoff` of `STATUS.md`; exit 4 when refused. |
| `effort park <goal home> [--run <id>] [--timeout D] [--json]` | Parks the verified calling orchestrator for as long as goal state allows; an explicit run must match its verified identity. Inspect the JSON park status and await handle; exit 4 when refused. |

**Parked-session compaction.** A wake re-sends the parked conversation, and the
model's prompt cache lasts about 30 minutes, so a late wake pays for the whole
context again. When a codec-pipe run stays parked on the same handle for 20
minutes, its runner declares `supports_session_compaction`, and its context is
at least 40k tokens, Agent Manager compacts the session and logs the outcome on
the run. How is the runner's business: today only Codex declares it, through
`codex app-server` (`thread/compact/start`; sending `/compact` through
`codex exec resume` is only text). The
thread ID is unchanged; a wake that arrives during compaction waits for it.
Interactive runs are never compacted, because resuming their thread in the
app-server clears the native goal. A failed compaction leaves the full session
for the wake.

A profile that declares the `agent-manager:orchestrate` scope lets its run
continue, stop and wake its own direct children from inside the run. It never
reaches beyond the caller's lineage. A child run may use the parent's model at
an equal or narrower effort, or a strictly cheaper known model tier (a Sol
supervisor may start a Luna repair run) at any effort; it must retain or narrow
the parent's execution restrictions.

## Goal-home commands

`epoch-check`, `lint`, `handoff set` and `park` read a delivery goal home
(`large-effort-orchestration` §3) through `api/internal/goalhome`'s current-format
reader. Runtime operations require CHARTER/STATUS/LOG/FEEDBACK; they do not fall
back to legacy registers. `effort migrate` is the explicit recoverable legacy import.
Working epoch briefs remain in `epochs/`; accepted-unit identity and worker
attribution live in LOG. Start with the [work entry point](../../../../.vrooli/work/README.md). `lint` and `handoff set` read and write files only and
run without the API. Exit codes: 3 is a worker step-back; 4 is an orchestrator
refusal (unmet acceptance, failing lint, refused handoff, refused park).

**Worker gate check.** A brief reports acceptance when its slice log has an `ACCEPTED` line
and no directive is dated after that line. Its fired triggers are listed as
"reported only"; it never exits 3 and never wakes. An open acknowledgement or a
malformed log line never reopens it. The orchestrator writes:

```
ACCEPTED <ISO time> | yield=<±n> <unit> | gates=G1,G3 | <summary>
```

`yield=` is the measured result; `gates=` names gates the orchestrator re-ran
itself. A legacy `ACCEPTED <time> — <summary>` line is still accepted; its
yield is unknown.

This brief record supports the worker gate check; it is not the product scoreboard.
After independent acceptance, the orchestrator appends the stable work-unit ID,
worker UUIDs, timestamp, yield and gates to `LOG.md`. A later LOG reopening retains
that ID. STATUS lists every independent active unit; a selected summary is not
an inventory of all workers. Unassessed surfaces stay unassessed until evidenced.

**Typed exit gates.** Items under `## Exit gate`:

```
- G1 test: cd ui && pnpm type-check
- G2 journey: J01,J02,J03 @ bas-goal
- G3 inventory: runtime_lines <= 253452 from 254178 (python3 docs/internal/refactor_inventory.py --no-git)
- G4 review: deletion-list
- G5 custom: offline archive playback — reason: no automated offline check
```

Amendments go under `## Gate amendments`, each dated and reasoned; an
amendment without an ISO date or a reason is reported and not applied:

```
- A1 2026-10-07T12:00:00Z G5 drop — reason: unverifiable; tool logged in WORKAROUNDS.md
- A2 2026-10-07T12:00:00Z G2 unverified — reason: shadow down after one authorized attempt
- A3 2026-10-07T12:00:00Z G6 add journey: J05 @ bas-goal — reason: …
- A4 2026-10-07T12:00:00Z G1 change test: cd ui && pnpm test — reason: …
```

Results are slice-log cells `gate G<n>=pass|fail|unverified [value]` (also
accepted as `G<n>=…`, and on the `ACCEPTED` line); the latest cell wins. Each
gate is reported met or unmet:
- `test`, `review`, `custom`: met on `pass`.
- `journey`: the value must cover every listed ID, as `12/12` or the IDs.
- `inventory`: the value must hold the measured number, compared with the
  bound; `pass` alone is not trusted. With `from <baseline>`, the measured
  minus the baseline is the epoch's yield.
- `unverified` is met only after an `unverified` amendment.

Evidence outside the admitted items is never read. An exit gate without typed
items reports `gate_status: legacy` with one warning and never fails.
`--acceptance` exits 4 on any unmet gate.

**Advisory estimates.** Slice budgets and estimates are optional. Work-unit, source-size,
elapsed-time and spend threshold observations are disabled by default. Large coherent
outcomes are valid; duration and token totals alone are not poor-performance evidence.
Explicitly enabled overrun, growth, spend and wall-clock observations enter `review` with
`review_only: true`;
human output says `OBSERVE (continue)`. They do not enter `step_back`, exit 3, wake a parent,
freeze source, stop execution, force partial handoff or fail acceptance by themselves.
Unavailable/unqualified spend is reported as unknown and does not fail the check. Continue
the admitted outcome and required proof while reviewing strategy and estimates. Flatline
still requests strategy review through the existing step-back signal; it is not a budget
stop. Explicit product gates, authority/safety limits and native commission HARD STOP remain.

**Yield.** The actual yield comes from `ACCEPTED yield=` or a measured
inventory gate, never from summed slice-log deltas (workers sometimes log
running totals; BAS E23 summed −656 for a measured −328). `net_runtime_lines`
remains that sum, for the growth trigger. The estimate is the brief header
`- Yield estimate: <±n> <unit>` (U+2212 is a minus sign).

**Lint.** `effort lint <goal home>` fails, naming each rule, when:

| Code | Rule |
| --- | --- |
| `resume-budget` | CHARTER.md + STATUS.md + open FEEDBACK (before `## Resolved`) exceed 30,000 bytes. Workarounds are already included in STATUS and counted once; an entry is open when its last ` · ` field starts with `open`, or everything before a resolved heading. |
| `handoff-count` | STATUS.md does not have exactly one handoff (any `##`/`###` heading naming a handoff); each one is named with its line. |
| `handoff-size` | The handoff exceeds 4 KB. |
| `idle-queue` | No `## Next` item has `[ready]` on its first line and `## Needs operator` has no list item or subheading (prose such as "(none; …)" is empty). |
| `feedback-unacknowledged` | A FEEDBACK `### <ID> — … (<date>, <source>)` entry with `Status: open` from the operator or the supervisor (including supervisor audits), or with no source, is not named in the handoff (in full or without the goal prefix, such as `FB-063`). Entries from other sources are not checked. |
| `diminishing-returns` | See below; blocking until `## Needs operator` holds an item tagged `[re-aim]`. |

It does not run inside `epoch-check`, so goal-home state never fails a worker.

**Diminishing returns.** Over the units accepted since CHARTER.md's
`- Destination set: <ISO time>` (all, when absent), the finding fires when:
- the last 3 accepted epochs each delivered under 25% of their yield estimate;
- the last 2 entries under `## Censuses`
  (`- <ISO time> | <scope> | admissible=<slice>|none`) found nothing admissible;
- the gap from STATUS `## Current: <n> @ <date>` to CHARTER `- Destination: <metric> <op> <n>`
  needs more than 30 epochs at the mean yield of the last 3 accepted epochs
  (yields in the destination metric only).

Missing inputs leave a rule "not evaluated", never fired. `epoch-check` prints
the finding for epochs under `<home>/epochs/`; only `--acceptance` and `lint`
fail on it.

**Handoff.** `effort handoff set` atomically replaces the one `## Handoff`
section (temporary file and rename, keeping the mode) and creates it after
`## Needs operator` when missing. Unchanged text writes nothing. It refuses
text over 4 KB, text with its own `#`/`##` or handoff heading, a queue with
more than one handoff (move old ones to `archive/` first; it never deletes
them), and a STATUS.md that changed between read and write.

**Park.** `effort park <goal home> --json` defaults to the verified calling run; omit `--run`. An explicit `--run` must match that verified identity. Inspect the returned park status and await handle; preserve a nonzero exit or refusal instead of masking it. The command lists the run's direct children and reads `## Needs operator`:
- a live (non-terminal) child allows up to 1h, a backstop for a hung worker;
- otherwise an item under `## Needs operator` allows up to 72h;
- otherwise it refuses with the admissible-slice rule: admit the next
  `[ready]` slice (or plan one), or record a decision under `## Needs operator`.

`--timeout` may only shorten the maximum; a longer request is refused. The park
itself is the `run park` call (producer `children`, key = the run). The server
applies the same `goalhome.DecidePark` to any `run park` on `children` by a run
that orchestrates an effort with a registered goal home. By default it only
records "park policy (report-only)" on the run timeline;
`VROOLI_EFFORT_PARK_ENFORCE=1` makes it refuse. An unreadable goal home never
refuses a park.

**Signals (DL-2).** Agent Manager keeps a durable per-effort signal log. `seq`
increases by one per effort; a dedupe key makes one fact one signal across
restarts. Kinds:

| Kind | Written by |
| --- | --- |
| `child_run_ended` | Agent Manager: a run holding the effort reference (not as orchestrator or supervisor) ended. |
| `cap_stop` | Agent Manager: a run of the effort stopped with `stop_reason=token_cap`. |
| `operator_answer` | Agent Manager: a decision was answered, defaulted, or answered late. |
| `step_back` | `effort epoch-check --effort <ref>` on STEP_BACK. |
| `delivery_pressure` | Agent Manager: one native threshold notice per commission/phase; never an acceptance record. |
| `feedback` | Whoever writes an operator or supervisor FEEDBACK.md entry: `effort signal <ref> --kind feedback --detail "FB-070 …"`. |

`effort signals <ref> [--after N] [--limit 100] [--json]` reads the log after a
cursor (RPC `ListEffortSignals`, response `{signals, latest_seq}`).

`effort feedback add <goal-home> --effort-ref <ref> --run <run-id> --id <id>
--title <text> --detail <text>` checks the run's exact public, verified, active
effort reference, writes a supervisor entry before `## Resolved` using a
compare-and-swap file update, then records a `feedback` signal with a stable
per-entry dedupe key. The FEEDBACK entry and signal are separate stores: a
signal failure reports a file-only partial result; retry the same ID and
content to converge. A changed body under an existing ID and an already
resolved entry are refused. This command records feedback only; it does not
create an operator decision or claim that work was accepted.

**Decisions (DL-5).** `effort decision raise <home> --id <id> --question <q>
--option key=label … --recommend <key> [--why …] [--default <key> --reversible]
[--deadline 24h] [--dry-run]` stores one record per (goal home, id), publishes
one `agent_manager.effort.decision_requested.v1` event (notification-hub asks
the operator once) and adds a `[decision <id>]` item under `## Needs operator`.
An identical re-raise changes nothing; a changed one is refused (use a new id).
A default needs `--reversible`; notification-hub applies it only after the
question was delivered. notification-hub's `ask.resolved` event records the
answer, appends an `operator_answer` signal and wakes a parked orchestrator of
the effort. `effort decision sync <home>` writes each answer verbatim into
FEEDBACK.md (a default reads "applied by default"), removes the item and marks
the decision recorded; it is idempotent. `effort decision answer <home> <id>
<key>` is the operator's desktop answer (agent runs are refused);
`effort decision list` lists them. `--home-ref` names the repository-relative
goal home when the directory is a copy.

**Board (DL-9).** Register `destination_ref =
"path:<repo-relative goal home>/CHARTER.md"` through revision-checked metadata
reconciliation. The board projects destination/current measurement, all active units
and worker sets, last accepted unit/yield, surface scoreboard, operator decisions,
open feedback/workarounds, usage/caps, lint and native delivery pressure. Missing
facts remain unknown. The selected active-unit summary is independent of last
acceptance and does not replace `active_units`. LOG acceptance and later reopening
retain stable authored IDs; status text alone never establishes acceptance.

**Token caps (DL-8).** A profile's `maxWeightedTokens` sets an observed-usage
stop threshold for each of its runs;
`CreateRunRequest.inline_config.max_weighted_tokens` may only lower it. Runs
without one fall back to `AGENT_MANAGER_TOKEN_CAPS`. Caps act only when
`AGENT_MANAGER_TOKEN_CAP_MODE` is `report` (timeline records) or `enforce` (an
85% nudge, typed into interactive runs, then a stop with
`stop_reason=token_cap` and a `cap_stop` signal). The default mode is `off`.

Current scope: the cap check is a post-persist hook on the continuation event
sink (`runEventSink`) only. The fresh headless executor, interactive, recovery
and import sinks do not apply it, so caps do not act on a run's first headless
turn or on interactive runs. Target (Usage Intelligence T-3, being implemented
by W3): one sink constructor applies the cap, quota and workflow-usage hooks on
every path; see [USAGE-INTELLIGENCE.md](../concepts/USAGE-INTELLIGENCE.md).

The threshold reacts to persisted provider usage receipts. It is not a strict
spend ceiling: usage can exceed it between receipts and while stopping. Codex
codec-pipe execution reports usage at `turn.completed`; interrupted-turn recovery
reads its rollout after exit and cannot enforce a mid-turn threshold. Interactive
Codex rollout `token_count` records can report completed model responses while the
turn continues, but neither in-flight usage nor reporting delay is bounded here.
Before the first receipt, the supervision view reports unknown usage, not zero.
Use the existing execution timeout and tool-call limits for their declared bounds;
do not claim they establish a token ceiling. A hard token ceiling requires a
qualified runner boundary before another unattended budget-critical experiment.

## Observation and trigger semantics

`row.changeIdentity` is the subject/evidence trigger. It excludes supervisor run
activity and aggregate supervisory accounting, but retains worker facts. `visibilityChangeIdentity`
and board `changeIdentity` include display-only supervisory changes. Consumers must
not use full board identity or enrollment CAS revision as an inference trigger:
new observed supervisor joins can advance the enrollment revision without changing
the subject cut.

Board `observedAt` is read time. Workspace row `observedAt` is its source capture
time, and each assignment records its owner-read time. Runtime completion is not
outcome acceptance. Active status with retained end/error evidence is inconsistent,
not proof of success or a safe replacement; its duration is excluded.

Aggregate runtime uses explicit orchestrator/worker assignments, not supervisor,
reviewer or investigator activity. Missing orchestrator coverage and conflicting
roles remain explicit; unknown registry relationships are not promoted to workers.
All observed roles remain in usage accounting. A finished executor is not an
accepted effort result.

Optional tokens, reported dollars and agent-seconds distinguish unknown from zero.
AM summary token totals currently lack qualified per-attempt attribution and remain
unknown, even when numerically populated. Estimates are not reported charges.
Execution identities deduplicate known durations. Observation and idle supervision
costs remain unknown unless supplied by an attributable source. Shared runs must not be summed once per board row.

## Bounded automatic discovery

`AGENT_MANAGER_EFFORT_ROOT` selects an absolute runtime root. Otherwise the
repository runtime-home contract supplies its plan-artifact effort directory,
optionally beneath `AGENT_MANAGER_RUNTIME_ROOT`. The existing scheduler scans at
`AGENT_MANAGER_EFFORT_SCAN_INTERVAL` (default one minute). No inference occurs in
discovery. `AGENT_MANAGER_SUPERVISION_ALLOWANCE_REF` labels standing observation's
allowance boundary but does not create a grant or measured usage.

Default manifest pages contain at most 100 immediate entries, rotate using a
durable cursor, and read regular bounded files (128 KiB default). A root containing
more than 1000 immediate entries is explicitly refused, not silently truncated.
Symlinks and path escapes are rejected. Unvisited pages do not imply removal.
Coverage/findings and timestamps expose partial availability. A real scan clears
bootstrap `not_scanned` even if another manifest remains malformed.

The runtime root can also contain planning or evidence directories that are not
Agent Manager enrollments. A child with neither a canonical `effort.json` nor the
legacy `handoffs/state.json` is unregistered when no retained enrollment names that
workspace; it is skipped without creating an effort row or invalid-manifest finding.
If an enrollment already names the workspace, the same missing sources mean the
observation is unavailable: retain its identity and prior evidence, and never infer
withdrawal or completion. Malformed canonical manifests and failures of declared
checkpoints or observation sources remain explicit failures.

Schema 1 reads the canonical `effort.json`; schema 2 contains a typed `enrollment`.
Explicit `target_revision` wins, then schema 1's declared
`execution.approved_source_digest`; both are attributed declarations, not verified
approval. The manifest hash is only a source revision. Optional `requirements.json`
and an explicitly declared bounded checkpoint beneath `handoffs/` or `evidence/`
are read. No credentials, transcripts or guessed driver state are read. An owner
identity (`authorizedBy`) is always stripped from discovered files.

Optional `observation_sources` declares up to eight non-secret relative files
under `handoffs/`, `evidence/` or `findings/`. Bounded hashes enter the subject cut,
not file contents. The transitional `handoffs/OPERATOR-ANSWERS.md` is also observed
when present, including manifestless drivers. New resolutions can therefore wake
supervision without a driver checkpoint change. Missing declared sources, unsafe
paths and oversized files retain unavailable evidence. Files are not executable
recovery instructions or grants; verify cited owner receipts before intervention.

Canonical public, verified, active AM `WorkReferences` with `kind=effort` and exact
effort ID automatically join owner runs. `relationship` supplies an observational
role, not authority. Registry pages are bounded to 100 runs and each effort to
100 subjects; incomplete accounting is explicit.

### Supervisor feedback handoff

`agent-manager effort feedback add <goal-home>` accepts only a run carrying exactly one
public, verified, active supervisor/orchestrator reference for the requested effort. Before
writing, it resolves the supplied directory and checks the existing effort board for the
single current enrollment and its canonical goal-home path. A copied, stale, withdrawn, or
ambiguous home is refused without changing `FEEDBACK.md` or signaling. Feedback IDs are
idempotent only when the file has exactly one matching open entry with the same content;
duplicate or resolved matches are conflicts. The file write and durable effort signal are
separate operations, so a signal failure reports a file-only partial result and an identical
retry reuses the same ID and signal dedupe key.

### Temporary-driver normalization

Scoped W3 repair (2026-09-12): missing manifests currently prevent legacy drivers
from appearing, although the accepted observation contract permits bounded driver
adapters. The adapter reads only `handoffs/state.json` under an already discovered
immediate effort directory when the manifest is absent or has no declared
checkpoint. A declared checkpoint takes precedence; malformed/unsupported
manifests are errors, not permission to bypass the manifest. No business workspace
is changed. Focused discovery fixtures are the validation boundary for this repair.

The legacy shape accepts absent `schema_version` (legacy v0) or integer version 1.
It requires nonempty string `effort` and at least one nonempty `next_action`,
`loop_state` or `orchestrator.loop_state`. Optional `assignment` is a string
reference, never a path to read or an accepted destination. Optional `children`
is an object of at most 100 rows. A row's valid UUID `run_id` becomes an AM worker
reference unless an explicit subject already supplies that run. `execution_id`,
`status` and `attempt` remain unverified driver metadata; they do not override AM
runtime state or supply measured usage. Invalid child identities remain explicit
limitations. Loop states remain attributed rationale, not runtime completion.

The existing rooted, symlink-rejecting byte limit applies to this one fixed fallback
file. Selected strings have explicit length limits; arbitrary nested files,
assignment documents, credentials and transcripts are not read. Source digest and
capture time identify the observation. Declared checkpoint/manifest values win.
Missing or malformed fallback evidence remains explicit uncertainty; prior rows
are retained if their sources disappear. A manifestless identity is provisional
`legacy-effort:<effort string>`; duplicate declarations conflict, and later
canonical identity adoption requires reconciliation. An existing row for the same
workspace retains its identity when its manifest disappears. No fallback field
grants authority, acceptance, measured costs or retirement.


### Home continuity and projection validation

Decisions belong to their enrolled effort across home relocation. `decision list <current-home>`
resolves that stable effort and projects the current location. The stored question home remains
immutable publication provenance, preserving event IDs and delayed phone replies. An identical
raise from the new home returns the original decision; changed terms and cross-effort lookups
remain refused. `decision sync <home>` records new answers and removes stale human holds
without recreating previously handled archived feedback. Explicit
`decision sync <home> --reconcile-recorded` restores already-recorded answers lost from
current FEEDBACK after a move or compaction; `--effort-ref <ref>` selects the stable effort.
The owner pages decision history through opaque continuation tokens; sync reads every page
before writing local policy and refuses ambiguous historical identities, including already-recorded
answers. Park steering reads the home freshly for every producer; board scan caches cannot
hide newly added or acknowledged feedback. Both sync modes are idempotent and do not republish asks, change answers or rewrite prior recording
timestamps. Reconcile existing dispositions when restoring history; an answered policy is
not a new request for operator authority.

Use `## Current: <n> @ <ISO date>` for a measured value, or `## Current` with
`- Current: unknown` when unmeasured. Unknown is not zero. Scoreboard columns are
`surface | target | state | owner`, optionally followed by evidence; owner is a single run UUID
or `-`. Active unit IDs are stable identifiers, with status/workers in their own fields.
Accepted LOG records belong in one `## Accepted units` section. Projection-losing state
formats produce lint findings and a partial board rather than silently healthy facts.
These findings do not stop a worker's implementation; repair the shared home with its owner.
Elapsed delivery pressure is separate from outcome progress and a completion forecast.

Before parking on children, the leader must act on open operator/supervisor feedback and
name its ownership/disposition in Handoff. Both `effort park` and the native agent park
entry point refuse `feedback-unacknowledged`, even with a live child or unchanged signal
cursor. This applies the existing lint obligation; it neither stops the child nor requires
product acceptance before a legitimate wait. Acknowledging an instruction does not resolve
its unmet product work.
