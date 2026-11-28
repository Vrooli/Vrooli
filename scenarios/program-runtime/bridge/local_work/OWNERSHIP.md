# Inactive native receipt and admission ownership

7 November 2026. This W3 continuation implements trusted receipt/selection and
service-admission storage components for the finite broker. Main remains
**disabled**. Nothing here installs a production credential provider, guard,
transport, process isolation, subscription or service. Canonical finite owner
adapters are now implemented but unregistered, as detailed below.
All execution evidence uses fixture authority and test-owned SQLite databases.
The work changes no live credentials, grants, security/network settings or data.

## Trusted native owner

`api/internal/localworkbridge/native_receipts.go` authenticates the existing
native HMAC webhook in Go, outside the old Python worker. Construction performs
no I/O. Invalid signatures, identity headers, ambiguous/duplicate JSON, changed
shapes and over-limit input fail before storage. Authenticated out-of-scope
records are ignored without storage. Operations have an eight-second maximum
and must cooperate with cancellation.

`NativeReceiptCredentials.VerificationKey` receives the exact trusted
owner/subject/source instance/epoch/allowlist scope. A deployment implementation
must establish that the dedicated key belongs only to that scope. The existing
native webhook does not sign an epoch claim, so a shared fleet key or reused
prior-epoch key cannot prove epoch ownership. There is no production provider in
this slice. Returned key buffers are cleared; this is not a claim of complete
secret-memory erasure.

`NativeReceiptGuard.WithCurrent` must hold the real shared native authority and
shutdown boundary through the database operation and bound its context by key
and authority expiry. Fixtures exercise this contract; no live joined authority
implementation exists. A disconnected mutex or precheck does not satisfy it.

The owner accepts only an explicitly supplied trusted SQLite database. Its
schema is local to this domain and uses the common `EnsureSchemas` owner. There
is no automatic registration in main, filesystem/path discovery, chmod or
migration of the Python outbox. The deployment owner must make the database and
all sidecar files inaccessible to the worker. Supplying a DB handle is not
proof of that filesystem boundary.

Only native event ID, exact type/source, authenticated body hash and receipt
time are stored. Payload bytes, signatures and keys are not retained. One
commissioning scope is pinned durably; changing it fails closed rather than
silently adopting old receipts. Exact duplicates remain acknowledged even at
capacity; conflicting duplicates and more than 10,000 retained records fail
closed. There is no implicit eviction.

Selection is a SQLite transaction, serialized across handles through the
singleton owner row. It either resumes the existing selection or selects at
most 64 authenticated records. The exact set, request key and five-minute
admission deadline commit together. Restart, a lost response, deadline expiry
and later receipts do not create a fresh attempt. Lock contention may fail a
call; after it finishes, a retry observes the same committed selection.

The result is an opaque `NativeSelection`. `BrokerSelection` requires the
approved gates to match the exact ID set and commissioned source scope. Its
private exact event identities are copied into broker-issued permits; the fixed
key/deadline and both approved gate digests are checked again at construction. A
canonical row with the same ID but a different otherwise-allowed type is
rejected. Static trusted `SourceBrokerSelection` configuration remains API
compatible, but it is not proof of native receipt provenance.

`NativeReceiptOwner.BindSelection` now durably pins the exact approved Events
and collector gate digests before broker use. Restart or a competing choice
cannot replace that binding. The older pure `NativeSelection.BrokerSelection`
projection is API-compatible, but is not durable commissioning proof and cannot
alone authorize retirement.

`ReconcileCollector` requires a still-current observe permit and the configured
canonical collector owner's `WithTerminal` callback. That callback holds source
authority and broker invalidation through the native SQL transaction, nested
under native authority. Completion records the exact selection, durable program
identity, terminal status and bounded successful-output hash, adds immutable
per-event membership, and retires the active selection atomically. A retry
reconciles the same tombstone without touching a later active selection. Later
selections exclude completed event IDs. Expiry and restart never renew an
existing key or deadline. An observe permit may reconcile after the admission
deadline, but only while its own source authority remains current.

**Bounded limitation:** this is collector-terminal reconciliation, not cloud
receipt, delivery, handler pickup or readback. Successful and failed attempts
remain distinguishable; neither is silently retried. Receipts and tombstones
are retained, with no pruning or source migration and the same 10,000-receipt
capacity. A missing/nonterminal collector or unavailable authority leaves the
selection retained. Joined native/source authority and inaccessible storage
remain deployment prerequisites, not consequences of an interface.

## Existing Program Runtime admission owner

The existing `programs` row now has `service_owner_json`. The immutable service
ownership identity is inserted in the same SQL statement as that row's execution identity;
there is no second admission/execution ledger and no invented verified-agent
identity. Existing ordinary callers retain their current behavior and default
empty ownership. `Save` cannot overwrite service ownership.

`programs.WithSourcePermit` accepts the concrete verified `SourcePermit` and a
trusted `ServiceAdmissionBoundary`. Owner identity is derived from the permit,
not caller metadata. The final check and insert run under the same callback
boundary that a production owner must share with source revocation and broker
session invalidation. Deadline expiry fails closed. Owned reads use retained
row identity, matching declaration/digest, owner/subject, binding/version, policy
and request key. Replay and wait use those owned reads. Unsupported in-memory
service storage fails closed; missing ownership is not manufactured.

`Service.SourceProgram` and `WithSourceProgram` project durable ownership.
Non-success partial stdout is suppressed; successful stdout is limited to 64 KiB.
No generic anonymous route has been tightened or claimed to be secure by these
components. Worker confinement remains the selected isolation mechanism.

## Inactive canonical owner adapters

`api/internal/library/source_collector.go` resolves the exact content-addressed
archived declaration and reviewed policy inputs through the existing contract
owner. It rejects non-read bindings and uses the canonical one-shot session
manager and program admission row. The final insert joins the supplied source
and broker authority boundary with the session manager's actual deletion and
reclamation boundary. Ordinary sessions and generic callers keep their existing
interfaces. Source policy's provenance enum is preserved, but caller identity is
empty and unverified; no agent identity is invented.

Admit, observe, wait and terminal reconciliation use retained row ownership.
One-shot sessions are reclaimed when terminal or when admission fails. Wait is
bounded by the verified request, and successful output alone is exposed. The
source authority boundary must be shared with real revocation and broker
invalidation, preserve cancellation and clip operations to authority expiry.
There is no live implementation of that external boundary in this slice.

`RecoverCollectorAfterStop` is an explicit per-selection recovery operation,
never a constructor side effect. It requires current owned observe authority
and `ServiceRecoveryBoundary.WithStoppedExecution`, which must prove the exact
old execution stopped/joined and exclude replacement through recovery. Only the
owned accepted/running row becomes failed with `runtime_interrupted`; unrelated
rows remain untouched and nothing is re-executed. Session cleanup is idempotent.
No production stop/join proof is supplied. Tests simulate interrupted persisted
rows after the fixture runner has completed; real process-crash recovery still
requires qualification. SQL admission itself is clipped to the fixed deadline,
including storage lock waits, even when the adapter is called without the broker.

`scenarios/vrooli-events/bridge.SelectedReader` reads the canonical Events
store directly. Its storage primitive fetches only one exact ID and bounds the
protobuf allocation in SQL. Canonical envelope ID/type/source must agree with
stored indexes and the verified selected identity. Missing records are explicit.
Pre-read and final rechecks occur under the supplied shared authority boundary.
No generic anonymous HTTP wrapper, new route or database discovery exists.

`localworkbridge.FiniteSourceOwners` composes the finite structural owner seams.
Neither module is registered into a live joined deployment. The Events boundary,
credential custody, source authority, launcher and transport are still supplied
integration requirements.

## Remaining integration and activation gates

1. Implement production native key provenance/custody and joined revocation,
   expiry, shutdown and recovery guards. Retain selection bindings and tombstones
   in trusted inaccessible storage; define explicit capacity/pruning policy
   separately without weakening replay protection.
2. Compose and qualify the canonical Events and Program Runtime adapters with
   the actual shared authority owners. Fixture tests cover real SQLite rows and
   session invalidation; they do not prove live revocation, stopped execution
   ownership or a commissioned cross-process owner path.
3. Implement the control-plane lifecycle owner, immutable runtime/storage,
   private broker transport and credential-free network-denied worker. Replace
   the old Python credential pipe/network path; it does not meet this boundary.
4. Keep native verification and cloud signing/egress outside the worker, with
   fixed destination and existing permission/checkpoint/pause/mute holds.
5. Qualify the actual joined runtime: generic endpoint access without bearer,
   socket/descriptor/proxy escapes, expiry/revocation at commit and IPC release,
   restart, lost responses, backpressure and stop/join. Then separately verify
   the authorized source-to-cloud-to-handler reconciliation/readback chain and
   two later separated authorized changes.

Action-time approval is still required before provisioning isolation/security
state, creating or placing persistent authority/credentials, subscribing,
enabling/deploying a service or making the named transmissions. Implementation
continuation is not activation approval. Approval also does not replace any of
the engineering proofs above.

Evidence is retained at
`.vrooli/evidence/local-work-bridge-20261106/ownership-20261107/`.
The protobuf freshness verifier still reports the same 23 unrelated generated
artifact differences; no generated output was edited. Focused tests do not
claim a passing full scenario or deployed least privilege.


## Qualified component evidence

Final checks passed: 78 bridge and 145 programs tests under the race detector,
3 selected API-main race regressions, 20 independent SQLite selection race
repetitions, bridge vet and Windows cross-compilation. Independent review
accepted the inert code and documentation, including real signed permit
composition, SQLite reopen, final revocation/expiry, exact event identity and
fixed native selection binding. Programs vet retains the pre-existing generated
protobuf mutex-copy warning at `service.go:952`; protobuf freshness remains
unclosed as described above.

## Adapter continuation evidence

The 7 November adapter continuation is retained at
`.vrooli/evidence/local-work-bridge-20261106/adapters-20261107/`.
Its final summary names the exact suites, review repairs and remaining limits.
The older ownership qualification above remains historical component evidence,
not a claim that subsequent changes inherit an unrerun full scenario pass.

Commit-boundary qualification remains explicit: the SQL transaction is created
with the expiry-bounded context, and cancellation plus current source authority
are checked immediately before commit. `database/sql.Tx.Commit` has no context
parameter, so expiry arriving after driver commit begins is not yet proven to
prevent a durable commit. An uncertain result must be reconciled through the
retained key/tombstone, never retried as a fresh attempt. Strict live commit-time
expiry and IPC release remain joined-runtime qualification requirements.

Final adapter checks passed: 97 bridge, 147 programs, 36 sessions, 8 library,
5 finite Events-reader, 36 Events-store and 3 API-main top-level race tests
(332 total, plus 410 subtests), with no failures or skips. Bridge/Events vet and
library/sessions vet passed; bridge Windows cross-compilation passed. Programs
vet retains only the pre-existing generated protobuf mutex-copy warning at
`service.go:952`. Independent review accepted the inactive scope after recovery,
non-success-output suppression and direct-admission deadline repairs; its
selected race regressions passed three repetitions. The same 23 unrelated
protobuf freshness differences remain; no full scenario or live qualification
is claimed.


## Inactive transport continuation

The Linux finite source transport is implemented as an unregistered component;
its protocol, ownership and partial-release limitations are in PRIVATE-IPC.md.
It does not replace the old Python credential/network path or install a
control-plane launcher. Existing admission, native selection and tombstone
owners are unchanged. A real helper-process crash during a truncated request
is tested; collector execution-crash recovery remains the earlier simulated
persisted-state fixture, not newly qualified process recovery.

The exact next engineering gate is the control-plane-owned immutable launcher,
real shared authority/release owner and actual canonical cross-process owner
composition. Strict SQL commit-time expiry and strict IPC release-time expiry
remain explicit qualification requirements. Provisioned confinement, native
key custody, persistent authority, activation and source-to-cloud delivery all
remain absent. No approval for these actions is implied by code continuation.
Latest retained evidence: `.vrooli/evidence/local-work-bridge-20261106/transport-20261107/`.
