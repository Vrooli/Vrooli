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

## Authority

Enrollment, withdrawal and explicit discovery require the existing verified human
owner principal. CLI invocation middleware forwards `VROOLI_AGENT_IDENTITY_TOKEN`
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

## Delivery orchestration commands

These commands serve the epoch model (`large-effort-orchestration`): one
long-lived orchestrator run per goal that parks between check-ins and starts one
epoch worker at a time.

| Command | Contract |
| --- | --- |
| `run continue <id> --message <text>` | The one way to talk to a run. A running interactive session receives the text as its next user message (the harness queues it behind the current turn); per-turn overrides are refused there. Any other continuable run starts a new turn. A codec-pipe run that is still mid-turn is refused; write an epoch directive instead. |
| `run park <id> --producer children --key <orchestrator-run-id> --timeout 1h` | Parks the calling run until a direct child exits, the timeout elapses, or a wake names the key. The wake message names only the children that ended since the last wake and the ones still active (`[wake: timer] …` or `[wake] …`). |
| `run wake --key <key> [--producer children] [--result <text>]` | `POST /api/v1/runs/wake-by-key`: the server matches every parked run whose await handle has exactly this producer and key and wakes it. A run identity may wake only runs its lineage permits (its parent, or its own children under the orchestrate scope). `run wake <id>` wakes one run. |
| `run tokens <id> [--children] [--weights sol=10,luna=1]` | Non-cache tokens weighted by model tier (default Sol 10, Luna 1; an unknown model weighs 1). Usage that is not final is listed, never counted as zero. |
| `effort epoch-check <epoch-file> [--acceptance] [--runs ids] [--spend-threshold n] [--wake-key key]` | Parses an epoch file and prints the step-back triggers, typed exit gates and measured yield. It exits 3 when a step-back fires; `--wake-key` wakes the orchestrator parked on that key (producer `children`). An accepted epoch never exits 3 or wakes. `--acceptance` is the orchestrator's check (exit 4). Spend uses the same weighting as `run tokens`. Slice-log lines need `<time> \| <what changed> \| exit metric=<v>`; line counts default to zero, unknown `key=value` fields are ignored, and `ack=` takes free text whose `D<n>` IDs count as acknowledged. See [Goal-home commands](#goal-home-commands). |
| `effort lint <goal home>` | Checks the goal-home rules the orchestrator owns; exit 4 on a failing rule. |
| `effort handoff set <goal home> (--file F \| --stdin)` | Replaces the single `## Handoff` of `QUEUE.md`; exit 4 when refused. |
| `effort park <goal home> --run <id> [--timeout D]` | Parks the orchestrator for as long as goal state allows; exit 4 when refused. |

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
(`large-effort-orchestration` §3) through one shared parser,
`api/internal/goalhome`. `lint` and `handoff set` read and write files only and
run without the API. Exit codes: 3 is a worker step-back; 4 is an orchestrator
refusal (unmet acceptance, failing lint, refused handoff, refused park).

**Acceptance.** An epoch is accepted when its slice log has an `ACCEPTED` line
and no directive is dated after that line. Its fired triggers are listed as
"reported only"; it never exits 3 and never wakes. An open acknowledgement or a
malformed log line never reopens it. The orchestrator writes:

```
ACCEPTED <ISO time> | yield=<±n> <unit> | gates=G1,G3 | <summary>
```

`yield=` is the measured result; `gates=` names gates the orchestrator re-ran
itself. A legacy `ACCEPTED <time> — <summary>` line is still accepted; its
yield is unknown.

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

**Yield.** The actual yield comes from `ACCEPTED yield=` or a measured
inventory gate, never from summed slice-log deltas (workers sometimes log
running totals; BAS E23 summed −656 for a measured −328). `net_runtime_lines`
remains that sum, for the growth trigger. The estimate is the brief header
`- Yield estimate: <±n> <unit>` (U+2212 is a minus sign).

**Lint.** `effort lint <goal home>` fails, naming each rule, when:

| Code | Rule |
| --- | --- |
| `resume-budget` | GOAL.md + QUEUE.md + open FEEDBACK (before `## Resolved`) + open WORKAROUNDS exceed 30,000 bytes. A WORKAROUNDS entry is open when its last ` · ` field starts with `open`, or everything before a `## Resolved` heading. |
| `handoff-count` | QUEUE.md does not have exactly one handoff (any `##`/`###` heading naming a handoff); each one is named with its line. |
| `handoff-size` | The handoff exceeds 4 KB. |
| `idle-queue` | No `## Next` item has `[ready]` on its first line and `## Needs operator` has no list item or subheading (prose such as "(none; …)" is empty). |
| `feedback-unacknowledged` | A FEEDBACK `### <ID> — … (<date>, <source>)` entry with `Status: open` from the operator or the supervisor (including supervisor audits), or with no source, is not named in the handoff (in full or without the goal prefix, such as `FB-063`). Entries from other sources are not checked. |
| `diminishing-returns` | See below; blocking until `## Needs operator` holds an item tagged `[re-aim]`. |

It does not run inside `epoch-check`, so goal-home state never fails a worker.

**Diminishing returns.** Over the epochs accepted since GOAL.md's
`- Destination set: <ISO time>` (all, when absent), the finding fires when:
- the last 3 accepted epochs each delivered under 25% of their yield estimate;
- the last 2 entries under `## Censuses`
  (`- <ISO time> | <scope> | admissible=<slice>|none`) found nothing admissible;
- the gap from `- Current: <n> @ <date>` to `- Destination: <metric> <op> <n>`
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
them), and a QUEUE.md that changed between read and write.

**Park.** `effort park` lists the run's direct children and reads
`## Needs operator`:
- a live (non-terminal) child allows up to 1h, a backstop for a hung worker;
- otherwise an item under `## Needs operator` allows up to 72h;
- otherwise it refuses with the admissible-slice rule: admit the next
  `[ready]` slice (or plan one), or record a decision under `## Needs operator`.

`--timeout` may only shorten the maximum; a longer request is refused. The park
itself is the `run park` call (producer `children`, key = the run). Until the
server applies the same policy (`goalhome.DecidePark` in
`orchestration.ParkRunFromAgent`), a raw `run park` can still bypass it.

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
