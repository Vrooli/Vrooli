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
| `effort epoch-check <epoch-file> [--runs ids] [--spend-threshold n] [--wake-key key]` | Parses an epoch file and prints the step-back triggers. It exits 3 when a step-back fires; `--wake-key` wakes the orchestrator parked on that key (producer `children`). Spend uses the same weighting as `run tokens`. Slice-log lines need `<time> \| <what changed> \| exit metric=<v>`; line counts default to zero, unknown `key=value` fields are ignored, and `ack=` takes free text whose `D<n>` IDs count as acknowledged. |

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
