# Effort supervision client contract

Agent Manager's existing `agent_manager.v1.AgentManagerService` owns these Connect
RPCs. The canonical schema is
`packages/proto/schemas/agent-manager/v1/domain/effort.proto`; generated Go,
TypeScript and Python messages use the same contract. No plan family is required.

| CLI operation | RPC | Request message |
| --- | --- | --- |
| `effort board` | `GetEffortBoard` | `GetEffortBoardRequest` |
| `effort list` | `ListEfforts` | `ListEffortsRequest` |
| `effort discover` | `ReconcileEffortDiscovery` | `ReconcileEffortDiscoveryRequest` |
| `effort enroll` | `EnrollEffort` | `EnrollEffortRequest` |
| `effort withdraw` | `WithdrawEffort` | `WithdrawEffortRequest` |
| `effort direct` | `RequestEffortDirective` | `RequestEffortDirectiveRequest` |
| `effort directives` | `ListEffortDirectives` | `ListEffortDirectivesRequest` |
| `effort update-directive` | `UpdateEffortDirective` | `UpdateEffortDirectiveRequest` |
| `effort assess` | `RecordEffortAssessment` | `RecordEffortAssessmentRequest` |
| `effort issue-dispatch` | `IssueSupervisorDispatch` | `IssueSupervisorDispatchRequest` |
| `effort revoke-dispatch` | `RevokeSupervisorDispatch` | `RevokeSupervisorDispatchRequest` |

Invoke these with `agent-manager` before the operation. Read operations accept
`--json`, `--page-size` (1–100) and `--page-token`; board and directives accept
`--effort-ref`. Mutation operations accept `--request-file PATH` containing the
whole typed request, not only its nested enrollment/directive. For example:

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

This example grants no steering. To amend, pass the current enrollment and its
`revision` as `expectedRevision`. Reusing an idempotency key with different content
conflicts. Withdrawal requires `effortRef`, `expectedRevision`, `reason` and
`idempotencyKey`; it survives rediscovery. Automatic observation needs neither
operator credentials nor enrollment calls; explicit reconciliation is operator-only.

## Authority and recurring wakes

Enrollment, withdrawal and explicit discovery require the existing verified human
owner principal and supervision scope. Directive/assessment requests explicitly set
`authority` to `WATCH_AUTHORITY_OPERATOR` or `WATCH_AUTHORITY_FAMILY_PARENT`.
The latter reuses signed AM run identity without requiring a fake plan family.
CLI invocation middleware forwards `VROOLI_AGENT_IDENTITY_TOKEN` as
`X-Agent-Identity-Token`; this header is eligible only for agent authority and is
verified by the existing run identity owner. It never substitutes for an operator
credential. Token strings must not be placed in request JSON or evidence.

Local operators can explicitly add `--local-owner` to a mutation (or `discover`).
This exchanges the existing Unix machine-principal binding through api-core's
authenticator client, retains its token only for that request and preserves CLI
attribution. It does not create a binding, persist a session, retry a denied agent
request as owner, or accept agent-authority requests. Supervisors use their signed
run identity instead; `--local-owner` is refused inside an identified agent run.

An operator grant binds `authorityRef`, future `authorityExpiresAt`,
`maximumDirectives` (1–100), optional `cooldownSeconds`, `targetRevision`,
`permittedActions` (NUDGE, CONTINUE, or separately granted RECOVER_FRESH) and one
exact orchestrator subject.
The supervisor can be a pinned `supervisorRunId`, or an exact pair
`supervisorOwnerSubject` + `supervisorScope` matching verified signed claims.
The scope is not a wildcard. PM must arrange legitimate attenuated claims on fresh
wakes; team/profile text, parent conversation IDs and work references do not grant
permission. The purpose-bound dispatcher below supplies recurring identities;
live autonomous steering still requires separate qualification and effort grants.

Scoped identity plumbing (W3, 2026-09-12): `CreateRun.requestedScopes` is an
optional typed narrowing with `scopes` and `expectedOwnerSubject`. Its presence
requires a verified owner credential. Empty scopes grant nothing. AM refuses a
wrong expected owner, expired credential, or scope outside that owner's ceiling
before reserving or dispatching a run. Persisted narrowing and credential expiry
also constrain later token minting. The bearer credential stays in the request
header, never in run data, prompts, work references or error evidence.

The human-owner `CreateRunDelegated` client is for explicit one-request owner
credentials, not PM's recurring provisioning path. PM does not automatically
exchange, retain or refresh a human-owner token. Recurring dispatch uses a typed
authorization on an existing AM enrollment and AM's existing signer; there is no
new credential store or dependency on a native client-credentials exchange.

The delegated dispatch endpoint checks the live authorization, exact binding,
expiry and revocation before creating each child. Child identity verification
also checks that authorization, so revocation invalidates existing children.
The dispatcher credential is not a run token, owner token or general API grant;
ordinary CreateRun and lifecycle operations must reject it. Child claims contain
only the allowed owner subject and scope subset, and expire no later than the
authorization. Enrollment amendment/withdrawal remains the operator-owned
revocation path. Public workload/profile/team declarations cannot supply this
authorization. Typed effort directives still enforce the separate target,
action, expiry and cumulative allowance. Diagnostic authorization grants no
repair-worker launch or business steering. Direct verified-run Create/Continue/
Stop guards remain unchanged; authorized steering uses typed effort directives.

## Durable dispatcher implementation contract

`IssueSupervisorDispatch` requires the current enrollment owner, the exact
supervisor owner/scope binding and a future expiry of at most 30 days. It selects
one team/member/profile, 1–1000 cumulative run admissions and a 60–86400 second
minimum interval. Rate and allowance survive AM/PM restarts. It saves server-owned typed
authorization and a credential hash on the existing enrollment; the bearer goes
directly to the canonical authority at
`vrooli/prompt-manager/effort-supervision`, field `dispatcher`. Responses contain
metadata only. Repeating an issuance key must recover that same authorization;
it must not restore revoked or superseded authority.

The verified owner's scopes must cover `agent-manager:supervise` under the
canonical capability matcher. Legacy `agent-manager:write` permits some operator
controls but does **not** grant recurring supervisor delegation. A refusal with
`issue-dispatch predicate=owner_scope_missing` requires an explicit grant through
the canonical authorization owner and a freshly verified credential; do not infer
the scope from operator status, profile text or enrollment declarations.
Issuance refusals identify a fixed configuration, enrollment, owner/scope, expiry
or replay predicate without returning principal values, held scopes, credentials
or signing material. Refused pre-issuance checks leave the enrollment unchanged.

`CreateSupervisorRun` alone accepts that purpose-bound credential. It checks the
live enrollment, exact dispatcher binding, expiry, revocation and durable run
allowance. Replays retain their original admission. The child inherits only the
granted owner/scope, profile and expiry, with an explicit authorization reference.
Every child identity verification rechecks the grant. Enrollment amendment,
withdrawal or explicit revocation invalidates the binding. Discovery preserves
an existing unchanged owner grant but strips file-supplied authorizations.

Creation acceptance is retained independently of the deletable run row in the
run owner's `run_creation_receipts` table. Run creation and its receipt commit in
one transaction. Receipts have no TTL or cascading foreign key; run deletion,
task deletion and purge cannot make an accepted key reusable. Schema application
idempotently adopts existing runs. It cannot reconstruct runs already deleted
before receipt retention was deployed.

`RunRepository.GetCreationReceipt(ctx, key)` is the read-only identity lookup for
owner recovery. `GetByIdempotencyKey` returns `accepted-result-unavailable` when
the original acceptance exists but the run is gone. General and supervisor
CreateRun check this before fresh admission, even after ordinary cache expiry.
Missing results never authorize replacement. A transaction that failed before
run persistence retains no acceptance receipt and can retry its original key.

Regression evidence: the prior implementation enqueued a second executor under
an exhausted one-run grant after completed-run deletion and cache eviction.
The fixed RPC fixture requires exactly one executor and refusal of that replay;
repository tests cover task cascades, immutable key retention and pre-effect
rollback. The cause was loss of the accepted effect identity, not loss of the
supervisor's cumulative admission counter.

Use a dedicated standing-service enrollment with no business target, subjects or
permitted steering actions. Its identity grant is independent of each effort's
action grant, so one wake can assess several efforts with a common verified
owner/scope. The profile is selected by AM from the grant, not from work references
or request overrides. Keep PM's configured profile equal to the granted profile.
Its persisted `declaredScopes` ceiling must permit `agent-manager:supervise`;
issuance and dispatch refuse an empty or excluding ceiling. That declaration
narrows the explicit owner grant; it cannot grant authority by itself. Finite
leaders are a distinct purpose and must not reuse this dispatcher credential.

Operator activation: first prepare
`standing-enrollment.json` with the already-bound human owner subject:

```json
{
  "enrollment": {
    "effortRef": "service:standing-supervision",
    "displayName": "Standing supervisor identity",
    "supervisorOwnerSubject": "EXISTING_OWNER_SUBJECT",
    "supervisorScope": "agent-manager:supervise"
  },
  "expectedRevision": "0",
  "idempotencyKey": "standing-enrollment-v1"
}
```

Enroll explicitly, then use the returned revision in `dispatch-grant.json`:

```bash
agent-manager effort enroll --local-owner --request-file standing-enrollment.json --json
```

```json
{
  "effortRef": "service:standing-supervision",
  "expectedRevision": "ENROLLMENT_REVISION",
  "teamId": "TEAM",
  "memberId": "MEMBER",
  "profileKey": "QUALIFIED_PROFILE",
  "expiresAt": "FUTURE_RFC3339_WITHIN_30_DAYS",
  "maximumRuns": 24,
  "minimumIntervalSeconds": 300,
  "idempotencyKey": "standing-dispatch-v1"
}
```

The uppercase strings are operator-selected values, not literal valid inputs.
Issuance requires the verified enrollment owner and held supervision scope.
The human credential authorizes this explicit future interval; PM never renews it.

```bash
agent-manager effort issue-dispatch --local-owner --request-file dispatch-grant.json --json
vrooli credentials status --identity vrooli/prompt-manager/effort-supervision --field dispatcher --format json
prompt-manager team heartbeat-enable TEAM MEMBER --supervision \
  --profile=QUALIFIED_PROFILE --accounting-ref=service:standing-supervision \
  --min-wake-interval-seconds=300 \
  --dispatch-effort-ref=service:standing-supervision \
  --dispatch-authorization-id=AUTHORIZATION_ID
```

Use the returned `dispatchAuthorization.authorizationId` for `AUTHORIZATION_ID`.
This command enables the selected heartbeat; do not run it before deployment
qualification and the desired effort grants exist. There is one canonical
purpose credential slot in this version: issuing another dispatcher explicitly
replaces that slot. Revoke obsolete grants separately. A configuration without
the binding keeps its existing unauthenticated diagnostic behavior.

For immediate revocation, read the current enrollment revision using
`agent-manager effort list --json`, then prepare `dispatch-revoke.json`:

```json
{
  "effortRef": "service:standing-supervision",
  "authorizationId": "AUTHORIZATION_ID",
  "expectedRevision": "CURRENT_ENROLLMENT_REVISION",
  "reason": "Operator ended recurring supervision",
  "idempotencyKey": "standing-revoke-v1"
}
```

```bash
agent-manager effort revoke-dispatch --local-owner --request-file dispatch-revoke.json --json
```

Revocation invalidates child authority as well as future dispatch. It does not
stop running processes. Missing/expired/revoked grants require the named operator
issuance/revocation workflow; they never trigger owner exchange or wider fallback.
Admission failures after reservation conservatively spend the run allowance;
the same request key can reconcile its original run, never mint a replacement.

No token belongs in any JSON file or command arguments. PM resolves the
credential through `credential-authority-go` at dispatch, only when the selected
heartbeat has an explicit `supervision.dispatchAuthorization` binding. Missing,
locked or unavailable credentials refuse that dispatch with a named provisioning
condition. Ordinary observation remains available without such configuration.

CreateRun carries bounded typed `workReferences` into the existing run record.
These are attributable workload declarations, not grants. An authenticated fresh
supervisor can record an assessment from its exact canonical public active
supervisor reference and target revision before periodic discovery joins it to
the board. The owner reads that run once; no directory scan, enrollment rewrite,
operator exchange or steering permission is implied.

Directive request JSON includes `directive`, `expectedEnrollmentRevision` and
`authority`. The directive needs `effortRef`, `targetRevision`, `targetRunId`,
`kind`, `scope`, `evidenceRefs`, bounded `adjustment`, `expectedResult`, future
`expiresAt`, and `idempotencyKey`. Optional `hypothesis` and `comparison` are
required before later claiming supported/contradicted benefit. Pending delivery,
owner acceptance, orchestrator acknowledgment, action and assessment are separate.
Only the target run can acknowledge, defer (with `wait` and reason), challenge
(with evidence and reason), or attach its action. Supervisor/operator updates can
assess or supersede. Updates require `directiveId`, `expectedRevision`,
`idempotencyKey` and `authority`.

For a proven missing managed session, `WATCH_ACTION_KIND_RECOVER_FRESH` uses the
execution owner's conservative recovery operation; CONTINUE and NUDGE do not
authorize it. Supply `recoveryExpectation`, `hypothesis` and `comparison` before
dispatch. The server records `recoveredRunId` with exact predecessor lineage;
the replacement, not the failed predecessor, can acknowledge and provide progress
evidence. While the original grant remains current, the owner reconciles its
orchestrator subject to that replacement and retains the predecessor as
`previous-orchestrator`. Expiry, scope, cumulative directive usage and original
source snapshot do not change. A withdrawn, expired, superseded or replaced grant
does not transfer. Missing or unavailable owner receipts stay uncertain; no
second creation is authorized by a timeout or a missing row.

Explicit historical receipt repair uses the existing `effort update-directive`
request with `reconcileDelivery: true`, the current `expectedRevision` and a new
stable `idempotencyKey`. Do not combine it with another update. It only reads the
original admission and retains its result; it never resends a continuation or
replacement. The server requires a retained uncertain transition and rejects a
proven `refusalBeforeEffects`. Original uncertain deliveries continue receipt
reconciliation after expiry, withdrawal or supersession. A later owner tick can
finish an already-accepted fresh replacement's subject projection only while the
original unchanged grant still permits it. This is not recovery verification or
product acceptance.

Allowance/cooldown admission uses complete durable reservations under an atomic
enrollment lock, not the board's directive page. Superseded uncertainty is retained
and fenced against further sends. Original owner-effect reconciliation remains
necessary; supersession does not prove an earlier uncertain effect did not happen.

An explicit CONTINUE directive can also recover a failed exact target with a
retained session and nonempty `hypothesis`/`comparison` describing the resolution.
NUDGE retains its cooperative-review boundary. Cancelled and completed finite
runs are not restarted; active/parked targets retain their owner. Missing sessions
require qualified owner fresh-run recovery, not an invented continuation. The
execution owner can still refuse a retained but unusable session. No directive
executes a temporary driver's shell command. Unavailable or stale workspace
evidence refuses new steering; grants, cumulative reservations and replay rules
still apply. See the cross-owner
[recovery contract](../../../../docs/agent-system/EFFORT_SUPERVISION.md#resolution-evidence-and-recovery).

New recovery directives supply `recoveryExpectation` with a falsifiable
`progressCondition` and 1–30 `baselineEvidenceRefs`. AM retains this immutable
expectation and initializes `recoveryVerification.state` to `pending`; delivered
does not mean recovered. Legacy directives without an expectation retain their
historical semantics and cannot acquire a fabricated baseline after dispatch.

After inspecting actual owner results, the authorized supervisor uses
`effort update-directive` with `recoveryVerification`: `state` is
`progress-observed`, `owner-wait`, or `failed`; include `reason`, new
`evidenceRefs`, and post-delivery `observedAt`. Unresolved states also require
`nextOwnerCondition`. The server binds `verifier` to the authenticated caller,
checks the current grant/revision and, for observed progress, reads the exact
target's lifecycle. The references must include new evidence distinct from the
baseline. The verifier still owns interpretation of the referenced artifact;
this is attributed supervision evidence, not independent product acceptance.
The original directive's revision/idempotency/history apply. Verification never
resends an operation or resets its allowance. Pending verification and owner
conditions appear on the CLI/UI board separately from causal-benefit assessment.

## Observation and trigger semantics

`row.changeIdentity` is the subject/evidence trigger. It excludes assessments,
supervisor run activity and aggregate supervisory accounting, but retains worker
facts and directive delivery/acknowledgment/action facts. `visibilityChangeIdentity`
and board `changeIdentity` include display-only supervisory changes. Consumers must
not use full board identity or enrollment CAS revision as an inference trigger:
new observed supervisor joins can advance the enrollment revision without changing
the subject cut. PM scheduling/sample behavior needs its own integration test.

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
costs remain unknown unless supplied by an attributable source. A shared assessment
is recorded once by `sharedOperationRef`, linked to all `effortRefs` and exact
`targetRevisions`, with `sourceLedgerRef`, evidence and usage. This implementation
retains shared usage fully unallocated; partial allocation is not yet supported.
Do not sum the same linked assessment once per board row.

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
are read. No credentials, transcripts or guessed driver state are read. Stable
delegation and steering grants are always stripped from discovered files.

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
100 subjects; incomplete accounting is explicit. A new orchestrator invalidates
the old target grant; worker/supervisor joins do not enlarge it.

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
grants authority, acceptance, steering permission, measured costs or retirement.
