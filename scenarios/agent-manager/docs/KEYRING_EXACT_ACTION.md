# Keyring exact-action source slice

Status: source implementation with protected ordinary-startup setup wiring. No live
issuer, grant, enrollment, activation, hosted signer or runtime qualification is
established by these tests. Empty installation refuses. Approved exact mutations
remain limited to owner-reviewed `disposable-thin-fixture-v1` efforts.

Ed25519 verifies Site v1 canonical envelope JSON. Native recomputes SHA-256 of
actual deterministic ReconcileEffortMetadataRequest protobuf bytes and compares
strict decoded action equality. Bindings include purpose `keyring.exact-action.v1`,
instance, audience `agent-manager`, issuer/key version, current grant/epoch,
initiating account, verified executor, operation/target, action/revision/request.
Signed bytes are limited to 16,000. Native accepts only the final 60 seconds before
signed expiry, bounded by policy deadline; Site separately bounds issuance lifetime.
No unique-dot/process-possession guarantee is implied. A copied same-executor
proof plus bearer may perform its exact action before consumption.

The native identity remains zero-scope and may remain ownerless; a nonempty owner
must match the proof account. Private request authority and actor provenance pass
RequireOperation, effortGrantActor, service and repository. Missing/contradictory
provenance refuses before legacy replay. Trusted context constructors do not defend
against arbitrary malicious server code. Thin requests pin effortRef/authorizedBy/
authorityRef, empty supervisor/subjects, unchanged withdrawal, and empty unsupported
mutable metadata. Nonempty omitted stored metadata is refused, never erased.

The shared SaveEffort transaction helper retains domain revision and existing
supervision_effort_transitions receipts. Exact admission acquires a same-store
writer fence before current-policy, domain and replay decisions. Semantic identity
binds original executor/account/grant/epoch/issuer/instance/operation/target/request/
payload; refreshed expiry/signature or equivalent action JSON ordering is excluded.
Another executor cannot replay the mutation. SQLite contention is an uncommitted
error: retain the exact request, then retry with fresh current authority. Busy or
failed revoke is not committed revocation. A committed revoke blocks later exact
decisions; committed effects remain. Work admitted before expiry may finish its
short transaction; no instantaneous cancellation or cross-store ordering is claimed.

## Protected local flow

`agent-manager effort exact-reconcile --request-file <typed-request.json> --grant <existing-id>`
requires an existing protected native credential. The process emits a bounded
challenge, stays alive while the private tool returns Site `{protocol,bytes,signature}`
JSON on stdin followed by EOF, and submits once. It does not attach, print/persist
bearers, exchange operator authority or create tokens. Every final wire request
rechecks protected authority; refusal or any leaf change ends the flow. The manual
attach CLI prints a bearer and is unsuitable. Real launcher custody/local task
carrier still require T06–T09/T16 qualification.

# Current native receipt-read contract

Supersedes Library v1's per-reader/source-policy contract. Use the original standing
policy ID; no reader enrollment, second read policy or receiptSource record.
Explicit operations may include `effort.reconcile` and `receipt.read`; read requires
`readEligibility:"standing-account-v1"`, empty `reader`, null `receiptSource`.
Legacy reader-specific policies refuse rather than silently widening.

`agent-manager effort exact-receipt --request-file <request.json> --grant <original-standing-id>`
uses existing protected native credential custody, pinned for every wire request.
POST `/api/v1/keyring/exact-read-challenge?grant=<original-standing-id>` then POST
`/api/v1/keyring/exact-receipt` with signed exact proof/signature headers.
Signer output: `{protocol:"keyring.receipt-read.v1",bytes,signature}` over private
stdin. Native never exchanges the operator credential or attaches to recover.

Ordered action:
```json
{"operationKey":"reconcile-metadata:<original-request>","effortRef":"effort:<id>","originalExecutor":"<original-run-uuid>","originalPayload":"<original-protobuf-sha256>"}
```
SHA-256 commitment uses compact UTF-8 typed JSON in that order, Go-compatible
escaping of `<`, `>`, `&`, U+2028/U+2029. Unknown/trailing fields refuse. Envelope
field order remains `v,purpose,instance,audience,issuer,keyVersion,action,grant,
epoch,initiator,executor,operation,target,payload,request,expires`. Purpose is
`keyring.receipt-read.v1`, operation `receipt.read`, executor the currently verified
reader UUID, request the operationKey, target `agent-manager:effort/<id>`.
The signer must authenticate the initiating account and bind this reader to the
actual verified native challenge; possession of an account name is not proof.
Native allows an ownerless zero-scope identity with the separate account-signed
request; any nonempty native identity owner must agree with the policy account.

Native checks current grant/key/epoch/deadline/revocation under the same owning
store fence and joins immutable receipt grant/account/instance/effort and original
executor/payload/request. Prior receipt epochs/key versions may be lower than
current ones after qualified rotation; future values refuse. Original receipt
issuer/key/epoch and executor remain unchanged. Mutation replay cannot adopt a new
executor or renewed policy identity. Read and mutation purposes cannot substitute.
Read rolls back even on success: no domain revision, receipt or approval is written.
Revoking the original grant blocks agent read. Existing independently verified
human receipt-read authority retains its separate contract.

Signed bytes max 16,000; final use window max 60 seconds, bounded by policy and
available reader expiry. Site separately checks issuance age. Fresh session
hosting/ingress authentication, protected signing-key custody and real workers
remain separate runtime gates; source tests do not establish those proofs.

## Protected native setup contract

Ordinary Agent Manager schema creates empty authority/setup tables. Ordinary route
assembly installs a verifier sharing the effort service's repository; no nil
verifier/manual injection or environment activation is needed. Empty installation
fails closed. Existing current independent human authority is mandatory for setup;
agent attribution, forwarded human authority, local UID or loopback alone cannot
approve. Stage/activate/revoke need `agent-manager:write`; status needs read.

`agent-manager effort exact-setup stage|activate|revoke --request-file <typed-public-request.json>`
and `agent-manager effort exact-setup status --grant <original-id>` use existing
operator authentication through POST/GET `/api/v1/keyring/authority/setup`. Each
setup operation also accepts explicit `--local-owner`, using the existing Unix
account binding for that request only. Input is validated before exchange;
identified agent runs and exchange refusals stop without setup or fallback.
The token remains in native process memory; it is never printed or persisted.
Setup requests refuse all redirects without contacting the destination. An
unconfirmed mutation may already have committed: reconcile the same original
grant with current independent owner authority before any retry. The ordinary
client and its provenance remain unchanged; native human/scope checks still apply.
Public stage request `{action:"stage",policy:<ExactPolicy>,expectedEpoch:0}` starts
epoch 1 against an existing exact thin reviewed effort. Policy includes only a
public Ed25519 key. It creates inactive state and a nonsecret receipt, never a native
run identity. Installation pins its first independently verified owner/instance.
No private key is imported, generated or returned by native setup.

Stage receipt includes grant/epoch/policyDigest/state/instance/approvedBy/approvedAt
and ordered activationChallenge `{v:1,purpose:"keyring.native-activation.v1",
instance,owner,grant,epoch,policyDigest}`. The existing protected key custodian signs
exact compact Go-compatible JSON of that challenge with the staged public key.
Activate request `{action:"activate",grant,expectedEpoch,policyDigest,keyProof}`
uses base64url Ed25519 signature (64 bytes). Independent owner approval and key
possession are both necessary; stage alone admits no operation.

The policyDigest is lowercase SHA-256 of compact typed native ExactPolicy JSON in
order `id,owner,instance,issuer,keyVersion,epoch,deadline,effortRef,authorizedBy,
authorityRef,profile,publicKey,operations,readEligibility,reader,receiptSource`.
publicKey is standard base64, deadline integer UTC seconds. Use returned digest
and activationChallenge; do not reconstruct a digest from arbitrary input JSON.
Accepted profile is `disposable-thin-fixture-v1`, explicit bounded operations,
existing effortRef/authorizedBy/authorityRef, future deadline at most 30 days.
This scope remains narrow exact metadata authority, not arbitrary execution.

Unknown outcome: status the original grant; identical stage replay preserves
original approval/state, active activation replay is idempotent. Revoked grants
cannot be revived by old activation/stage replay. Rotation requires expected
current epoch and next epoch, immutable account/instance/effort bindings; changing
key/issuer requires increased keyVersion. Rotation stages inactive before new key
possession activation. Revoke `{action:"revoke",grant,expectedEpoch,policyDigest}`
is idempotent. Foreign/legacy orphan records refuse; setup never adopts them.
Human expiry is rechecked after the writer fence and before commit; refused setup
rolls back installation, policy and setup state together. SQLite contention has
an uncommitted retry result; unknown network outcomes require original status.

## Qualification boundary

Disposable tests cover actual AttachRun/token verification, middleware, typed handler,
RequireOperation/service, SQLite/receipts, wrong bindings/provenance, two identities
under one synthetic grant, fresh-proof replay, concurrent pools/reopen, revocation/
contention and CLI custody. They do not establish hosted signing/enrollment, real
fresh ChatGPT sessions or spawned workers. Live policy/key activation and hosted signing remain held.
No generated changes are needed; the historically denied protobuf verifier is
excluded, not claimed passed. Other storage engines/journal modes need qualification.

Work ladder: bounded W3 repair against October 11 approved canonical proof-only
contract. Unmeasured W0–W2 and whole-scenario readiness are not claimed passed.


## Ordinary CLI and future direct-dot custody

The production route is `main → NewApp → App.Run → domains.effortGroup →
cmdEffort`. The catalog registers `exact-setup`, `exact-reconcile` and
`exact-receipt` with typed argument metadata. Public setup requests use
`agent-manager effort exact-setup stage|activate|revoke --request-file public.json`
and `agent-manager effort exact-setup status --grant <original-id>`.
Site's owner-gated `POST /api/proof/activation` signs the staged
`keyring.native-activation.v1` challenge through its configured protected adapter;
ordinary proof qualification is a separate gate. Native activation still requires
independently verified current operator authority and the matching policy digest.

Start a **future** direct-dot coding session through the existing native launcher:

```text
vrooli agent launch --runner codex --require-native-identity --cwd <workspace>
vrooli-agent-launcher --agent codex --require-native-identity -- [coding-agent argv]
```

The launcher resolves the existing broker/custody before considering attachment.
A responding refusal or unavailable authority stops before bootstrap, lease and
child execution. With allowed absence it reuses the existing narrow AttachRun
contract, privately retains the returned native leaf, rechecks the same authority,
and delivers the leaf only in the harness environment. Changed/refused custody
prevents child execution; cleanup of a newly created owned attempt is attempted through the same protected
transport, without stopping any other run. If denial also blocks cleanup, the
original refusal and nonsecret owned run ID report cleanup as unconfirmed; no
alternative credential or transport is used. Existing authorized custody is adopted without a second
attachment. The opt-in flag has no anonymous fallback. No new identity service,
execution grant, permission or credential store is introduced. Default launcher's
best-effort attribution behavior remains its existing separate contract.

Within that session run the ordinary exact command. One CLI process resolves and
pins the current native leaf, emits a public challenge, awaits the Site proof on
stdin, then submits with the same identity. It reconsults live custody during the
signing interval and each wire request. Credential change or broker refusal
terminates that flow; never change TMPDIR, socket namespace, inherited token or
executor to evade refusal. An already running unidentified session is not
retroactively attached by an exact command. Protected launch prepares future
sessions; it is not authority to attach this executor or expand retained consent.

Genuine child runs still require actual exact-parent CreateRun admission. Native
execution phases mint the distinct child leaf under the parent's/owner's existing
ceiling and deliver it through the runner environment. Keyring's one standing
metadata/receipt grant supplies no native run execution authority and requires no
additional reader-specific approval. Receipt reads retain the immutable original
mutation executor, payload and request.

Disposable v3 qualification executes the compiled ordinary CLI through the real
launcher, actual configured Site activation/mutation/read proofs, fresh attached
native roots, genuine CreateRun parent/child phase custody, replay and zero-effect
refusals under one original standing grant. Synthetic owner verification, D1/key
custody and in-process runner observations are explicit fixtures. Actual installed
builds, hosted owner ingress, protected Site settings/enrollment, SO_PEERCRED
session binding, containment/editor lease registration and real worker execution
remain live qualification gates; no live setup or rollout is claimed.

## Command-only direct task session

An already-running local dot task can invoke installed `vrooli agent exec --
agent-manager effort exact-reconcile --request-file <public.json> --grant <id>`
(or exact-receipt). Use `--task <uuid>` only for an actual existing native task row;
a cloud thread/task identifier is not presumed to be that UUID. For a multi-command
session, pass an explicitly selected script executable as argv. The wrapper never
constructs shell text and starts no coding model. Standalone equivalent:
`vrooli-agent-launcher --command -- <command argv>`.

The wrapper remains alive while the child executes, reports its own actual PID and
fresh command-session label, and privately delivers the zero-scope attached leaf.
Existing custody is adopted without another attach. This proves an attached local
command process, not privileged identity of its enclosing cloud executor. Broker
refusal/unavailable/changed custody is terminal, including after attachment; same
route cleanup failure is reported as unconfirmed. No per-actor standing approval.
A new wrapper/session does not renew a finite or expired grant. Current installed
build, authorized standing setup, protected Site custody and native actual-leaf
verification remain live prerequisites. No cloud task launch hook is required for
this explicit in-task command route; automatic wrapping remains a platform choice.

### Independent installation operator

Native stage/activate accepts the existing scenario-authenticator human provider
for resource `agent-manager`, or the configured Cloudflare Access provider with
existing canonical owner/scope policy. Scenario-authenticator's production
`/vrooli.scenario_authenticator.v1.accounts.AccountsService/Login` accepts that
resource and existing account/MFA requirements. Same-origin browser proof can reach
`POST /api/v1/keyring/authority/setup`; originless setup requires protected bearer
delivery. Existing `agent-manager:write`, current expiry, native audience/JWKS and
owner binding must agree. Activation also requires Site key possession. Bridge
OS1 enrollment is not an Agent Manager provider in the inspected wiring. This is
installation authority, not per-command actor approval; no key/token enrollment or
operator identity exchange occurs inside the command wrapper. Use existing protected
session handling, never copy secret proof into argv, output or public request files.

The explicit in-task exec route closes the source gap without a cloud startup hook.
It does not automatically wrap every cloud task, prove this executor privileged,
or override a live broker denial. This source qualification does not install the
new binaries or migrate this current task. Runtime owners still need a qualified
current build, the already-authorized standing setup and actual protected Site/session
custody evidence before enabling real use. Neither this wrapper nor a fresh attached
session extends or renews finite-policy authority.
