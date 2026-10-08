# AUTH-01 caller migration

> **Operator decision 2026-10-07:** runtime enforcement is off by default. The operator said: "at this stage it's too much friction." The source stays. A single switch re-enables enforcement once unattended authority (AUTH-02) exists and the setup is more mature. See `docs/agent-system/DELIVERY_RELIABILITY.md` P-18.

### The switch: `VROOLI_AUTH01_ENFORCE`

One environment value controls AUTH-01 caller enforcement. It is read in one place, `owneridentity.CreateRunCallerEnforced()` in `packages/api-core/owneridentity/create_run_enforcement.go`. Enforcement is on only when the value is `1`, `true`, `yes` or `on`. An unset or any other value means off. The value is read on every call, so a process uses the value in its own environment.

When the switch is off (the default):

- Agent Manager `POST /api/v1/runs` accepts a request without caller proof (`handlers/runs_create.go`, `orchestration/create_run_caller.go`). The request takes the anonymous creation path that existed before AUTH-01.
- Prompt Manager manual heartbeat and team triggers, the `/runs` proxy, member conversations, the executor, the team queue and finite-leader ticks/relaunches use `AdmitCreateRunCaller`, `RequireAdmittedCreateRunCaller` and `CarryAdmittedCreateRunCaller`. Scheduled ticks, keep-alive leader relaunch and agent-attributed triggers (including the existing delivery→supervision wake rule) are not refused for missing proof. After a restart, a persisted ordinary manual queue intent is discarded like a stale scheduled tick; it is not held as `caller_required`. The schedule starts the member again.
- Offered proof is still verified. A valid human Bearer with `agent-manager:write` is attached and forwarded for attribution. Heartbeat and team triggers forward only verified human proof. The `/runs` proxy forwards offered proof unchanged so that Agent Manager verifies it. Agent Manager still refuses an invalid offered owner or run token. The run-identity lifecycle guard, exact-parent and retained-lineage checks, scope narrowing, dependent-delegation qualification and finite effort authority do not change.

When the switch is on, all of the behavior described in the rest of this document applies unchanged.

Not covered: Swarm Manager (manual goal queue), System Monitor (investigations), Knowledge Observatory (deep search, doc healing) and Switchboard (dispatch) still call the strict primitives. They refuse a missing caller on their own side until they use the `Admit…` variants.

To re-enable enforcement:

1. Set `VROOLI_AUTH01_ENFORCE=true` in the environment that launches the Agent Manager and Prompt Manager API processes. The lifecycle runner passes its inherited environment to the scenario processes. Set the value where the control plane or supervisor that starts them can read it, not only in an interactive shell.
2. Restart Agent Manager first, then Prompt Manager.
3. Verify it: read `/proc/<api-pid>/environ` for each API. An anonymous `POST /api/v1/runs` must return 401 `create-run requires verified caller identity`.
4. Before you re-enable it, give unattended callers an authority source (AUTH-02). If you do not, keep-alive relaunch, scheduled heartbeats and the supervisor wake stop again (O28).

CreateRun requires verified existing human owner proof with `agent-manager:write`, or a verified signed run identity whose retained task and parent exactly match the request. Caller projections in JSON do not establish authority. Missing, expired, malformed, duplicate or wrong-channel proof must be refused before reservation and dispatch. Offering both channels does not permit ignoring an invalid channel. No credentials or grants are created by this repair.

Supported server adapters retain the complete offered header fields, verify existing human proof, and forward it only to Agent Manager CreateRun. They do not retry creation anonymously or forward caller secrets through redirects. A verified parent route must set the exact retained parent and task; an adapter without that route refuses run proof.

Prompt Manager manual heartbeat/team triggers qualify the human before control, task or queue effects. The in-memory queue retains original proof and expiry across request cancellation. Duplicate requests do not replace the original caller. Expired entries are held as `caller_required`; after restart, persisted manual intents are retained without proof and held. Old scheduled ticks retain their existing discard behavior. No proof is serialized. An operator must explicitly resolve a held intent and submit a new authorized manual intent; recovery does not renew it.

Swarm manual goal queue admission verifies existing human proof before queue mutation and carries the original proof in memory through Start and CreateGoalRun. Expired or recovered goal entries refuse Start before baseline capture, grants, profile/task creation or dispatch. Supplying a new token to Start does not renew a recovered intent. Submit a new authorized queue intent after explicitly resolving the old intent. Ordinary phased workflows retain their existing authority model.

UI and unattended callers still need an actual supported authority source before rollout. Available choices are an already authenticated human manual route, a separately qualified exact-parent route with existing proof, or holding unattended creation. A new service principal, credential issuance or broader grant requires explicit owner authorization and is outside this repair. Browser storage, a user-supplied actor label, and an invented background identity are not authority.

This is a source migration contract. Runtime rollout remains held until current caller compatibility, current-source build/test evidence and target authority are qualified. Sibling lifecycle surfaces are inventoried separately; this CreateRun repair does not silently authorize or alter them.

## Separately approved overnight source work

AUTH-02 A selects the corrected report's bounded finite-effort delegation B. Its [source contract](AUTH02_FINITE_EFFORT_CONTRACT.md) distinguishes explicit approval/custody/provider setup from original human JWT lifetime. The new source seams are unconfigured; browser closure, localhost, signed task labels and a dot/native tool do not activate them. Runtime rollout still requires the current caller/target evidence and passing or explicitly resolved owner gates above. Do not exchange credentials, enroll a client or create a grant merely to make a test pass.

## Disposable joined qualification and opt-in private owner transport

The source now includes an unregistered mTLS current-owner transport at `/internal/finite-effort/v1/current-owner`. It requires a verified and pinned client certificate authorized for an exact protected policy ID. The issuer loads that policy and derives its owner; arbitrary subject reads, extra credential channels, redirects, insecure TLS, missing client certificates and unbounded HTTP clients refuse. No session is exchanged or refreshed and no grant is issued by this route. Approval still belongs to the local authenticated human issuer. Server registration, peer enrollment, protected storage and runtime binding installation remain explicit setup actions.

Historical pre-isolation checkpoint: disposable owner-module Go test helpers qualified actual external finite-start HTTP, invalid-channel zero effects, a queued original binding recovered into the real PM executor, actual native task/run HTTP admission, private actual authenticator account/grant reads after session logout and five hours of authority clock advancement, exact native terminal settlement followed by one single-slot successor, old ingress replay and current-grant refusal. Native dispatch is closed. The acceptance reader uses a separately approved temporary commission; it does not accept the existing Planner review or E1. Live canonical accepted-owner installation and caller/profile migration remain rollout gates. That joined finite fixture has not been migrated to the required enabled launch installation and exact-unit terminal owner; its earlier positive result is not current launch-isolation compatibility evidence.

Finite queue obligations must be saved durably before worker launch or native dispatch. Enqueue/dequeue/recovery save failures hold work; BeginDispatch requires a durable writer for B; finite native 5xx outcomes retain uncertainty. Signed request-local proof is redacted from native refusal errors, including encoded echoes. Attachment-bearing task projection is presently unsupported and explicitly refuses before native transport. If preparation commits but first leader state cannot be saved, its exact key is returned and the unknown occupied slot is retained; missing evidence never authorizes replacement task/run work.

## Disabled finite native launch installation

The source contract in `internal/nativeisolation` and public package façade is opt-in owner startup composition. `InstallFiniteNativeIsolation` couples the concrete launch factory with exact-unit terminal settlement. The disabled qualification manifest at `config/finite-native-isolation.disabled.json` contains intentionally invalid placeholders; it creates no host state and is not mounted by main. A valid finite proof without an enabled installation is refused before CreateRun reservation or dispatch. Ordinary verified human/exact-parent callers retain their normal behavior while the installation is disabled.

An enabled target is finite-only, bound to exactly one immutable policy/repository/deadline and its complete owner-selected profile contract and a separate clean worktree view. It supports the initial codec-pipe native route. Finite children, continuation, failed-run recovery, interactive/imported execution, policy files, workspace write policies and protected/effect-contained sandbox profiles are held pending their own migration qualification. Existing exact-parent identity validation is retained; a valid identity does not bypass an unsupported launch contract. Unknown native descendants or manager outcomes retain occupancy and never constitute proven non-start or terminal settlement.

Source fixtures qualify construction, admission, compatibility and zero-effect rejection; they do not qualify systemd, networking, namespaces, native readiness or descendant protection on a host. Before activation, the owner must separately approve the exact nonmodel probe/setup, establish clean worktree provenance and result-return rules, prove private authority custody and current owner/profile contracts, qualify all fixed systemd properties and unit terminal evidence, and supply current build/authority evidence. No credentials, new UID, grant, host setting or deployment follows from source installation.

## Current source migration and startup boundary

The actual Claude, Codex and OpenCode codec `BuildEnv` routes are narrowed by the finite launcher before native planning. Inherited host HOME/PATH become the installed repository's `.native-home` and `/usr/bin:/bin`. Duplicate or malformed entries and active loader/proxy/bus settings refuse. Other uninstalled environment keys, including inherited credential names, are dropped; this is not provider credential enrollment or permission to expose authority credentials. Tests qualify all three actual codec environment builders and fixed env-shim planning. Provider/image/egress compatibility still requires its own current migration evidence.

A planning manifest is not launch readiness. The public planning constructor can describe an enabled manifest, but its concrete finite factory does not report enabled, cannot be installed and refuses binding checks unless it owns an explicitly captured continuing witness. Missing-witness CreateRun rejects before row or ledger effects. Continuing witness checks re-read the exact unit, invocation, process start identity, image, mount/workspace view, network posture and original bounded expiry; dead/restarted/stale evidence refuses. Capture and bootstrap are distinct explicit owner operations. A completed first probe cannot authorize later processes.

The separately unmounted private profile contract at `/internal/finite-effort/v1/profile/check` reads the complete persisted native profile object and computes its digest at the Agent Manager owner. Its exact enrolled mTLS peer, immutable policy/epoch/owner/client/deadline and profile-ID table are mandatory; human/run/finite-proof channels, foreign bindings, malformed/oversized input and redirects refuse. Public profile authentication is unchanged. PM's protected reader verifies this owner result rather than hashing its portable profile projection.

Typed AM, PM and issuer startup preparation freezes nonsecret bindings, refuses incomplete providers and publishes nothing when disabled. The ordinary mains remain disabled; setting `VROOLI_FINITE_ENABLED` without an owner-built installation refuses before their startup effects. The issuer does not create keys or reuse receipt signing keys. Enabled PM/issuer startup requires an opaque continuing witness, current acceptance/team/profile scope and dedicated existing finite custody. The private handlers are not registered by these source changes; no TLS listener, client enrollment, service operation or host setup is performed.

The older joined PM/private-broker/native fixture is currently incompatible with the mandatory launcher and exact-unit terminal owner. Its failed native-admission result is retained; prior positive results are historical. Component admission fixtures use a private test-only checker and closed dispatcher, explicitly separate from real launch readiness. Combined migrated caller qualification, current protected acceptance and custody owners, exact unit-operation authority, host probe/setup and canary remain held. Planner review `668d4191-3e83-47e8-ba03-93410d5ad924` remains NEEDS_REWORK; these changes do not accept E1.

## Protected owner source continuation (disabled)

The fixed-unit source owner retains immutable local native plans and rechecks current policy plus the accepted exact native receipt before effects. Native starts retain the existing local typed stdin/stdout/wait carrier. Its private control handler cannot publish arbitrary commands/properties/units/PIDs or start native model/stdio operations; remote native streaming remains unqualified. Separate fixed bootstrap and continuing-witness setup owners contain no finite engine or model grant. Source constructors do not authorize executing setup helpers. A witness alone cannot install or admit finite execution.

Dedicated finite-purpose custody uses the existing governed credential owner, freezes the exact public purpose/policy, checks current eligibility/revocation and signs only typed initial-run proofs. It exports no raw signer/key and supplies no enrollment/rotation writer. Broker new-admission reads and reservations recheck current purpose, including proofs signed before revocation. Canonical child/recovery reservation semantics remain supported; signing those broader proof purposes and actual finite child/recovery launch remain separately held. Existing owned read/settlement obligations retain their checks.

Private PM commissioning and admission are distinct reads. Commissioning can inspect the exact disabled authored target after complete acceptance/profile/team/member/binding checks, allowing future verified human consent without enabling execution. Live admission independently requires enabled current team/heartbeat state. Neither read approves work, enqueues, creates a run or renews a deadline. Private profile/read contracts can be composed before consent; ordinary mains remain disabled and handlers unmounted.

Swarm finite acceptance is designed within the existing versioned EffortControl aggregate, with authenticated owner context, revision/content/generation checks and revocation that blocks only future admission. Its actual component/module integration is HELD after a denied dependency-owner preview; the shared private acceptance transport alone does not establish installed protected acceptance. It cannot accept the original Planner review or E1.

Current disposable qualification includes mutual TLS over an in-process pipe, actual broker/custody/native admission with closed dispatch, purpose revocation with zero effects, PM commissioning→disposable approval→live-enabled read, and concrete three-codec startup composition using separately tagged synthetic capabilities. These are source checks, not host isolation, real model execution, complete three-service enabled startup or current full PM queue/restart/native migration. Historical joined results remain historical. Runtime rollout requires the complete supported-caller migration, current builds and actual target/authority evidence; no new credentials, grants, identities, host settings, unit operations or activation are implied.


## Current governed integration disposition (2026-10-04 19:44 UTC)

This dated disposition supersedes earlier Swarm integration holds and managed-namespace absence statements. The independently reviewed Swarm acceptance source is now integrated within the existing EffortControl aggregate. Its direct platform-go requirement and scoped freshness were applied by the existing scenario-dependency-analyzer owner after the same denied read was explicitly reauthorized through the supported reviewed execution route. No dependency registry approval, credential, service configuration or runtime installation was created. Focused acceptance qualification passed 13 tests under race and the inactive Swarm API build passed. The Swarm canonical unit run `20261004-193148-e4f97c60` failed; UI policy drift and an API-command failure remain recorded and do not constitute canonical clearance.

Bounded read-only observation associated the recorded Agent Manager, Prompt Manager and authenticator PIDs with ports 18800, 18060 and 18158 and their executable hashes. Boot, process-start identity and executable link remained stable across the observation. These serving hashes differ from the inactive repaired builds. This establishes bounded process/listener association only; it does not qualify unit properties, continuing isolation, custody, protected installation or deployment. The prior systemd-bus denial remains stopped.

The original root disposable joined HTTP/mTLS fixture carrier was independently authorized and historically passed. Its later fixed-owner failure is a source-migration refusal, not a listener denial. A separate supplemental websocket listener test was denied and remains stopped. Current disposable joined migration has passed with governed Swarm acceptance, actual disposable issuer consent/logout, concrete protected clients and typed three-service startup composition, durable PM queue/restart, CreateRun admission and exact mock-unit settlement. These separately tagged overlays remain outside production source; commissioning/unit/witness/model capabilities and policy-time advancement are synthetic, and dispatch is closed. The two retained case names exercise one private-broker topology. PM parent race detection does not extend to subprocess helpers; no host/model or five-hour wall-clock JWT-expiry qualification is claimed. Default production mains remain disabled, and complete owner-built startup publication remains an activation gate.

Runtime rollout and Workbook remain HELD. Preserve Planner review `668d4191-3e83-47e8-ba03-93410d5ad924` as NEEDS_REWORK; neither duplicate nor accept E1. No live dispatch, unit setup, attach, grants, credentials, access expansion or runtime replacement is authorized by this source disposition.


## Marketing source compatibility and remaining installation gates (2026-10-07)

This dated section supersedes earlier singleton-profile and missing ordinary
startup-composition descriptions. It does not supersede runtime holds, actual
caller authentication, or the original Planner NEEDS_REWORK disposition.

Disabled acceptance startup may carry a complete nonsecret `developmentSetup`
in its fixed protected role bundle. Only the acceptance role may carry it.
The actual server composes Development ownership against its canonical backlog
Handler before registering Development routes. Every supplied commission subject
must exactly equal the complete installed policy subject. No effort record,
acceptance, consent or run is created by composition; an absent record refuses.
The first Marketing effort draft is owner-review data with unknown native values,
not a record that may be submitted or a substitute principal.

Preparation captures exact direct artifacts and the complete explicit originals
manifest/member table. All hashes, sizes, paths and membership must match. Complete
archives have a 128 MiB aggregate and 16 MiB per-member ceiling; the existing 64
small direct artifacts and 4 MiB historical records are unchanged. Captured bytes
are immutable private memory for this prepared owner's lifetime. Durable installed
custody, restart reconstruction and live product/evidence-owner exclusion remain
separate qualification requirements. Unavailable acceptance evidence still refuses.

The native manifest supports the existing singleton form or an exact complete
2-profile table under the same one policy, repository, isolation and deadline.
Missing, extra, changed or ambiguous bindings refuse. The serial native adapter admits only the exact protected successor of an accepted
terminal predecessor. Recovery and wake dispatch remain refused. Finite parking checks current authority,
owner, complete native profile and exact occupied reservation before await/status
writes; its default wait ends no later than the original authority deadline.
Parking neither settles a reservation nor frees concurrency. Ordinary runs retain
their existing park/wake behavior.

Optional policy-bound serial edges and concrete owner handoff methods describe
serial parent/worker episodes. Handoff preparation is durable metadata, not an
execution reservation. A fresh episode requires the prior accepted episode's exact
native terminal settlement and consumes another start plus conservative ceilings.
Terminal flags and budgets are permanent; CAS contention refuses and is never
silently retried. The optional private owner carrier exposes metadata/read/reserve operations only
through the existing pinned mutual-TLS peer and explicit operation rights; source
support supplies no rights or installation. The owned codec terminal observer
consumes a bounded final JSON command, requires exact native-unit terminal proof,
and preserves the existing exact-parent qualification gate before metadata and
again before reservation. All episodes keep the original TaskID and unchanged
original task storage. A retained root-task input digest detects mutation; the
complete original task and sources remain in each bounded episode projection.
Prepared metadata and ledger receipts recover the same deterministic successor
key. Missing accepted run rows or uncertain reservations hold without redispatch.
PM follows protected receipts and refuses serial root relaunch/restart/reopen;
it saves episode observations without replacing the frozen leader profile.
Disposable qualification establishes this source contract, not live setup.
Never use a logical parked parent to release capacity or reuse a settled unit.

Marketing keeps its existing team and worker profile. The previously identified
shared orchestrator ID remains unchanged; it is not a currently compatible
Marketing execution binding. Current owner resolution must reconcile the high
orchestrator specification with the existing code.delivery medium-effort policy. Do not change global policy,
invent a fallback, add a worker member or expand access to force compatibility.
The original 8-hour window, at most 8 starts, concurrency 1 and all original turn,
tool and runtime ceilings remain upper bounds, not renewed authority.

Inactive compilation and disposable fixture results do not establish deployment.
Rollout requires current supported-caller migration, exact native target/principal/
revision/generation, complete installed custody, eligible existing TLS/purpose
handles, current private peer/build evidence and actual isolation/unit qualification.
The previously denied canonical protobuf verifier remains stopped; compiler success
cannot substitute for its generated-output qualification. No setup, credentials,
grants, team activation, dispatch or protected installation follows from this source
work. Planner review 668d4191-3e83-47e8-ba03-93410d5ad924 stays NEEDS_REWORK;
E1 acceptance, U2 replay and runtime rollout remain held.

Serial codec continuation contract (source-only): the uniquely selected successful
terminal assistant output must be strict JSON with exactly one `finite_serial`
object containing `version: 1`, `next_profile`, `title`, `instruction`, and
`summary`. The next profile must match an exact consented edge. Duplicate or
case-aliased keys, extra authority/model fields, oversized commands, unavailable
selection and unproved native terminality refuse. The native owner derives caller
identity, original TaskID, immediate ParentRunID, immutable payload/input digests,
existing profile and deterministic `serial-<source-run-UUID>` key. The model does
not receive credentials or gain private-network access. Each accepted episode
uses a fresh run/unit identity and reserves the full original ceilings. A serial
policy permits only one original root, plus exact replay of that root.

The resource-owned delivery model and declared effort are now carried together.
Missing, ambiguous, changed or mismatching effort and unsupported alternative
delivery models refuse before admission effects; no silent coercion occurs. The
current Marketing orchestrator's high request is incompatible with the current
medium declaration. The source-only compatibility candidate uses a separately bound medium parent
profile with the existing medium worker, through the existing complete-profile
commission mechanism. This preserves the shared high profile and resource policy.
The candidate is unapplied review data, not a live profile or accepted commission.
Future native owner consent must explicitly select its new identity and replace
Marketing's prior parent binding; no ID, timestamps, profile digest or acceptance
generation may be fabricated. Ordinary inline overrides remain refused for finite
callers. Historical high parents cannot become medium parents by replay. Qualification receipts and exact-parent restrictions stay unchanged.


### Marketing scoped medium preparation (source-only)

Comparison favors a separate compatible profile over a new per-run execution
override: existing policy/commission/profile contracts already pin complete
profile bytes. A new override table would need an additional authority-bearing
contract throughout acceptance, native inputs, model selection and replay. No
override table, policy-schema change or relaxed resource-effort gate was added.

The prepared parent declaration changes only name, profileKey and effort to
medium from the Prompt Manager shared parent declaration. All role, permission,
network, sandbox, scopes, spawn preferences and native ceilings remain equal.
Only that proposed parent and the existing worker would enter Marketing's exact
two-profile table. The original finite ceilings still narrow inherited defaults.
The shared high parent is excluded from that table; its live callers are untouched.

Disposable native owner/admission tests exercise this separate profile with
resource-owned medium, exact parent -> worker -> fresh parent receipts, full
charges and original expiry, restart replay and zero-effect pre-metadata refusals.
Synthetic receipt/terminal witnesses and a closed dispatcher do not establish
current profile installation, real model qualification or host execution. The
worker is a compatible disposable profile, not the actual installed worker.
Restart replay uses a reconstructed Orchestrator over retained disposable stores,
not a real process/host restart.

The serial source observer now requires its receipt runner/model/effort to equal
its own recorded execution tuple before handoff metadata, settlement or replay
effects. A corrupt medium source with a high receipt refuses without writes.
This integrity check does not change ordinary delegation: the existing global
rule permits equal or narrower effort and its documented cheaper-model tiers.

Eventual setup must first create the reviewed declaration through the native
profile owner, obtain its actual persisted ID and complete profile digest, and
compose the disabled Marketing heartbeat/profile table/serial edges/native
manifest/Development contract against that exact profile and existing worker.
This is a new Marketing parent selection requiring explicit owner consent; it is
not implicit consent to replace the shared profile or install private rights.
The native ApproveDevelopment request must then use owner-computed reference and
expected generation. No such setup, consent, selection or activation occurred.
