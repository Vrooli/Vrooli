# Local-work wake bridge (disabled)

## Implemented path

The Vrooli Events API owns one narrow local-to-Flywheel path. Canonical event
admission and the minimized wake occupy the existing event/subscription delivery
transaction. No second collector, outbox, Python service or polling bridge is
introduced. Existing unrelated subscriptions keep their existing behavior.

The legacy reserved `SubscriptionSpec()` still selects only
`agent_manager.run.notification.v1` from `agent-manager`. The separate dormant
`TerminalSubscriptionSpec()` selects exactly `agent_manager.run.terminal.v1`
from the same owner. Both specifications are disabled and require independent
explicit registration/enablement; there is no wildcard or automatic widening.
Atomic admission matches each reserved subscription's exact type and source
before projection, so either selection cannot queue the other event variant.
The canonical protobuf must agree with its event ID, source, type and occurrence
indexes; its occurrence must be present. The occurrence index has millisecond
precision, while the canonical protobuf and emitted occurrence retain all
nanoseconds using protobuf JSON timestamp formatting. Its structured notification category
must be `run-completed` or `run-failed`, and `correlation.run_id` must be a nonzero
UUID. This is an observed producer notification, not authenticated approval or
proof that a run has been accepted. The cloud reader must reconcile the selected
canonical event and run with its registered source and Work lineage.

The `flywheel.local-work-wake.v2` body carries one event ID, type, source,
occurrence, SHA-256 of the exact stored protobuf bytes, and the typed
`agent-manager.run-event.v1` locator. Its `owner_id` is the fixed native service
`agent-manager`, not a human principal. The configured source instance, scope
epoch/digest/checkpoint and permission hold never come from the event. There is
no title, body, instruction text, arbitrary metadata, URL, file path or raw
payload in this projection. Purpose is read-only reconciliation; activation is
`not_approved` and effect authority is `none`.

Both inert subscription specifications share the non-URL target marker
`flywheel:local-work:v2`. Their names are `flywheel-local-work-v2` (legacy) and
`flywheel-local-work-terminal-v1` (terminal). Neither is created or enabled at
startup. Generic delivery/reconcile routes reject either reserved name or target. Queued v2 bodies
stay on the reserved path even if someone subsequently edits the subscription.
While the forwarder is absent, SQL selects past held bridge rows before applying
the batch limit, so a bridge backlog cannot starve ordinary webhooks.

## Sender boundary

`bridge.NewForwarder` binds one approved HTTPS origin and the fixed path
`/api/bridge/local-work-events`, plus a dedicated `Signer` and a `WakeBoundary`.
This existing constructor remains usable for projection, but delivery now fails
closed without private Sites admission. Trusted transport composition uses the
additive `bridge.NewSitesForwarder` with a `SitesAdmissionProvider` as well.
There is no secret lookup, fallback to the generic webhook secret, credential
generation, or environment toggle. Neither constructor performs I/O.

The admission provider returns an already-approved complete Bearer header for
only the fixed origin/path, inside the same current policy boundary on every
attempt. The sender checks its bounded token syntax and attaches it as
`OAI-Sites-Authorization: Bearer <token>`. Sites dispatch consumes this header;
it provides neither visitor identity nor application/source authorization. The
independent bridge HMAC remains required. No actual provider/custody/rotation
implementation or credential is installed. Missing, invalid, expired/revoked
(provider-rejected), or cancelled admission never falls back to HMAC-only delivery.
Provider error details are discarded, and the credential never enters queued
bodies, batch IDs, HMAC inputs, logs, or response handling.
The boundary must serialize revocation and scope replacement with projection
and the entire send. The queued scope must still equal the current policy.
Cancellation and a ten-second upper deadline remain effective inside callbacks.

The signing message is exactly:

```
flywheel.local-work-http.v1\n<unix seconds>\nPOST\n<origin>/api/bridge/local-work-events\n<canonical JSON bytes>
```

The dedicated signer returns a 32-byte HMAC-SHA256 and its key ID. The three
headers are `X-Flywheel-Bridge-Key-Id`, `X-Flywheel-Bridge-Timestamp`, and
`X-Flywheel-Bridge-Signature` (`v1,` plus base64). The body and batch ID remain
identical on retries; the HTTP signature timestamp is fresh. Unknown transport
outcomes retain the existing bounded retry/dead-letter cycle. Cloud durable
idempotency makes repeated delivery harmless, including response loss.

The HTTP client has no proxy, redirect, cookie or ambient credential behavior.
It has bounded request/header/TLS timeouts, 8 KiB maximum response headers and a
4 KiB maximum response body. Only 2xx is success. Response bodies and transport
error details are discarded, not logged or treated as commands.

## Activation gate

The production `Server.flywheel` remains nil. No service was started, no bridge
subscription was created, and no real key, endpoint deployment, or transmission
was performed. The code is integrated, not activated.

Before activation the operator must approve the specific source selection and
fixed destination. Provisioning a real persistent credential requires separate
action-time approval. The approved integration must supply the dedicated signer
and an actual shared revocation/scope boundary, plus destination-bound private
Sites admission custody with expiry/revocation/rotation. Bind the matching cloud
source/key/scope configuration, install the forwarder through trusted server
composition, then explicitly register/enable the selected reserved subscription
(the terminal specification for the terminal vertical, not the legacy selection). These
providers and their credential lifecycle are not fabricated by fixture success.
No startup/configuration shortcut currently performs those steps.

A scope change does not reproject old queue rows: old rows fail current-policy
validation and require owner review. Agent Manager's existing terminal
notification publisher is best-effort; this bridge cannot recover notifications
that never reached Events. The bridge therefore does not claim complete local
work coverage. `checkpoint=unknown` and permission holds must remain cloud holds.
A later operational acceptance check must cover real owner/provider composition,
crash/recovery, revocation, and approved end-to-end delivery before enabling use.

## Verification (2026-11-07)

- Events root module `go test -race ./...`: passed across 15 packages.
- `go test -race ./bridge ./internal/subscription ./internal/store`: passed.
- API module `go test -race ./...`: passed, including existing subscription and
  atomic admission regressions.
- Eight bridge test groups cover canonical minimization/digest/locator, malformed
  indexes and payloads, retry signature/byte identity, redirects/status/oversize
  responses, scope tampering/cancellation, unknown coverage/default-disabled,
  invalid destinations, and canonical protobuf timestamp precision.
- Five API bridge groups cover event/queue rollback and duplicate/restart,
  response-loss retry bytes, disabled/mutated subscriptions, direct delivery and
  reconcile bypass rejection, and a 30-row held backlog before ordinary work.
- Independent static review found the disabled backlog starvation issue; the
  selector and regression above fix it. Follow-up review found no remaining
  actionable issues in the reviewed code.
- `bridge/testdata/local-work-wake-v2.json` is the native canonical wire golden.
  Cloud tests admitted these exact native bytes and the native-generated
  signature through the actual HTTP handler, retained byte-for-byte payloads,
  and returned the original identity for a duplicate. No network was involved.
- Required read-only `packages/proto: make verify-committed-gen` failed with
  23 pre-existing findings: template-validation-python-uv-react-vite generated
  outputs missing, plus descriptor drift. No generated file was edited. Focused
  compiled tests passed, but this is not a repository-wide freshness pass.
- A service-owned Test Genie scenario suite was not launched under the explicit
  no-service-start/no-activation scope. No deployment or live operational proof
  is claimed.

## Dormant exact-byte selected evidence

`bridge.SelectedReader.ReadSelectedEvidence` adds a minimized native read without
changing the existing `ReadSelected` protobuf-JSON contract or registering a
route. It requires the existing boundary to implement `EvidenceBoundary`; this
trusted owner callback supplies the current source instance under the same live
authority/session serialization as ordinary selected reads. Caller input and the
event cannot supply that binding. No production implementation is installed.

The new response includes only source instance, canonical event identity/type/
source/occurrence, SHA-256 of the exact stored protobuf bytes, typed run locator,
and either the legacy notification category or native terminal status. It validates occurrence against the storage
index without the generic converter's missing-occurrence fallback. Authority is
rechecked before storage and before release. Unsupported events fail closed;
missing records return nil. It does not provide run freshness, outcome acceptance,
or Flywheel Work lineage. Those remain unavailable pending their owner evidence.

`bridge/testdata/selected-event-evidence-v2.json` is a synthetic native method
output checked byte-for-byte by the Go regression. Its input reuses `wakeFixture`
with an additional valid unknown protobuf field, proving the digest binds stored
bytes that JSON re-serialization cannot reproduce. It contains no real run data.

## Dormant native terminal acceptance (2026-11-07)

`Server.acceptTerminalEventIntent` accepts the exact immutable JSON seed and
SHA-256 identity retained by Agent Manager's native terminal receipt. Its wire
type is `agent_manager.run.terminal.v1`, distinct from the best-effort
`agent_manager.run.notification.v1` wake input. It validates the six bounded
native facts: source instance/epoch, run ID, lifecycle version, terminal
observation digest, and terminal status (`complete`, `failed`, `cancelled`, or
`unknown`). It does not infer accepted work or promote unknown outcomes.

The shared `eventbus.RetainedDomainEventEnvelope` codec deterministically
encodes both the inner protobuf `Struct` and outer envelope. The Events owner
uses its existing transaction and subscription-matching implementation to commit
that exact canonical event, applicable delivery rows, and a separate
`terminal_event_admissions` receipt together. Only a successful commit can return
`vrooli-events.domain-event-ack.v1`, binding event ID, canonical byte digest, and
original store ID. Missing transaction/receipt capabilities fail closed; generic
HTTP conflict or best-effort publisher success is not this acknowledgment.

The append-only admission ledger retains exact intent JSON, canonical protobuf
bytes, digest, and store ID independently of pruneable events. SQL update,
delete, and replacement attempts are rejected. Replay checks retained bytes and
all canonical event indexes, including the existing millisecond occurrence
index, before returning the original acknowledgment. Once retention removes the
event, replay still returns the receipt without recreating events, rebuilding
payloads, or enqueuing delivery to old or newly added subscriptions. The ledger
has no automatic pruning policy; safe reclamation needs an explicit upstream
replay-retirement protocol before activation.

Actual SQLite fixtures cover partial fan-out and receipt failures, a deferred
constraint failure at COMMIT, cancellation, simultaneous retries, both possible
durable states behind an uncertain commit error, an abrupt process exit after
SQLite COMMIT but before acknowledgment, closed/reopened databases, corruption
of retained bytes/indexes, and replay after event/delivery pruning. A finite
native-owner fixture consumes Agent Manager's actual retained seed, reopens and
replays Events admission, and returns the resulting acknowledgment for native
retention. The fixture performs no network requests.

The private admission method has an optional owner-injected route and native
production composition described below. The route is absent at normal startup,
and no new runtime toggle installs it. The terminal reserved subscription remains
disabled and unregistered; durable acceptance alone does not enqueue a Flywheel
wake or install an authorized selected-evidence resolver.
Real source binding, credentials, subscriptions, deployment, and operational
end-to-end validation remain behind the activation gate above.


## Dormant terminal selected evidence and projection

The existing `ReadSelectedEvidence` and wake projector accept the terminal type
without changing legacy locator/output bytes or the ordinary `ReadSelected`
contract. Terminal facts must be exactly the six fields admitted above. The
`agent-manager.run-terminal.v1` locator contains `owner_id`, `run_id`,
`source_instance`, `source_epoch`, `lifecycle_version` (a positive canonical
int64 decimal string), and `terminal_observation_sha256`. The selected response
contains native `status` instead of legacy `category`; `unknown` remains an
observation, not completion. No payload text or Work lineage is projected.

The locator's source instance/epoch bind the original Agent Manager receipt.
They are distinct from the outer `source_instance`, which the trusted Events
boundary supplies, and from the approved wake scope. Event facts cannot choose
that Events authority binding. The canonical event digest still hashes exact
stored protobuf bytes, including unknown wire fields. Source identity, receipt
identity, and canonical byte digest must all agree with independently selected
native evidence before a consumer can reconcile the observation.

`Deliver` also requires the packet event type to equal the reserved
subscription's event pattern before signing. A terminal packet cannot inherit
legacy notification subscription authority. The separate terminal specification
can now validate and transport that packet through the injected Sites/HMAC
boundaries, but there is still no live terminal subscription or terminal transport
composition. The additive projection and reader remain dormant behind the
existing activation gate.

The retained acceptance row from
`terminal-outbox-20261107/final-cross-owner-fixture/events.sqlite` supplies the
exact `bridge/testdata/terminal-canonical.pb` fixture. The selected reader and
projector generate `selected-terminal-event-evidence-v2.json` and
`terminal-wake-v2.json` from those bytes in isolated SQLite tests. Only the
terminal test clock is fixed at `2026-11-07T05:16:55.481988288Z`; legacy fixtures
are unchanged. The final `go test -race -json ./bridge -count=1` run passed
102 tests/subtests in one package, with zero failures or skips. Malformed
bindings, versions, statuses, extra fields, indexes, and occurrences fail
closed; terminal-to-legacy subscription mismatch fails before signing.

Scoped W3 evidence and final source/check hashes are retained under
`.vrooli/evidence/local-work-bridge-20261106/terminal-evidence-20261107/`
in `events-bridge-race.jsonl` and `events-bridge-sha256.txt`. These fixture results
are not a readiness or live-delivery claim.


## Terminal transport qualification (2026-11-07)

This scoped W3 repair closes two concrete implementation gaps: terminal packets
had no valid reserved subscription, and the sender had no private Sites admission
header. Legacy notification wire goldens and the exact retained terminal fixture
are unchanged. Admission now also checks exact subscription/event type and source
before projection; adding the terminal selector must not create cross-type rows.

Independent review found one additional disabled-backlog edge: a terminal-named
subscription with an edited target and malformed/untyped queued bodies could
occupy the first page and starve ordinary webhooks. `DueExcludingNames` now
quarantines both names before limiting. The original `DueExcluding` API remains
available and delegates to the new selector. Regression coverage includes 30
name-only quarantined rows, edited names/targets with typed bodies, and ordinary
queue progress without consuming any held bridge attempts.

The isolated tests cover separate exact/default-disabled selectors, terminal
canonical admission rollback and replay, queue reconstruction and unknown-response
retry, identical body/batch identity with fresh Sites credentials and signatures,
HMAC/Sites independence, provider failure and cancellation, malformed/header
injection attempts, policy replacement, fixed destination and no redirects/proxy,
direct/reconcile/non-atomic bypasses, and unchanged legacy goldens. Fake admission
values and RoundTrippers are test-only; no external request is performed.

Evidence is retained in
`.vrooli/evidence/local-work-bridge-20261106/terminal-transport-20261107/`.
The current root-module and API race JSONL results, source hashes and summary
record the actual test outcomes and scope. Protobuf freshness still reports the
same 23 unrelated generated-artifact findings (descriptor drift and missing
template-validation outputs); no generated output was edited or regenerated.
Service-owned Test Genie phases were not launched under the explicit
no-service-start scope. These are isolated owner/transport proofs, not a live
private Sites dispatch or operational readiness claim.

The inactive native composition below now binds the Agent Manager capture/outbox
lifecycle to acknowledged Events admission through explicit dependencies. Before
activation, approved native source authority and transport must be supplied; Events must
supply current source policy/revocation, dedicated signing and Sites admission,
and explicitly install the forwarder/terminal subscription; the native selected
Events and receipt readers need current owner authority and bounded transport;
the existing attention consumer needs fresh unattended environment selection,
offline holds and immutable selector handoff. Deployment, persistent access and
first minimized-data transmission remain separate approval/qualification gates.
The first production result should be a durable truthful hold/no-op with no Work
mutation or association write, followed by duplicate/recovery/revocation testing.

Final observed results: changed bridge/subscription/store race suites passed;
bridge alone passed 119 tests/subtests. The API race suite passed 131 tests/subtests
with one explicit `TestTerminalEventAdmissionNativeOutboxFixture` skip because
its external fixture directory was not supplied. The new terminal transport and
API tests all ran. The broader root-module race run passed 14 packages but failed
`internal/config:TestLoad_Defaults`: its unisolated default path reached the live
operator-storage guard, which correctly blocked resolution. This unrelated
config-test isolation defect was not repaired by this transport change. No full
root-module or scenario-suite pass is claimed.


## Inactive native owner composition (2026-11-07)

`NewServer()` still calls `NewServerWithTerminalEvents(nil)`. The optional
`wiring.TerminalEventsConfig` has no environment/config-file loader. The ordinary
constructor uses the original receipt-disabled repositories and creates no
terminal worker, timer, transport, or source binding. Explicit composition
requires the existing source instance/epoch, an acknowledged Events acceptor,
a current publication-authority boundary, and bounded retry/batch settings.

`NewTerminalPersistence` is used by the real orchestrator graph and the isolated
fixture. The exact same repository objects capture terminal transitions and feed
the outbox. It checks an existing immutable source binding before returning the
graph, so a conflicting installation/epoch cannot start serving and later break
run completion. Empty stores remain unbound until their first prospective
terminal transition. Dispatch independently rechecks the durable binding.

The native server starts the worker in its existing long-lived recovery worker
step. Each pass is finite (1–100 pending receipts, a fifteen-second deadline),
uses the retained seed unchanged, and runs under current publication authority.
The trusted boundary must serialize revocation and scope replacement through the
whole pass. Authority denial leaves pending receipts untouched. Local capture can
continue while publication is held; a receipt is evidence, not publication
permission. Shutdown cancels and joins dispatch before closing native storage.
Restart builds a fresh worker against the same receipt and acknowledgment ledgers.
The worker adds no persistence, historical backfill, Work mutation, or association.

`eventbus.NewTerminalAdmissionClient` accepts only an explicit fixed origin and
injected bounded transport. The fixed path is
`/api/v1/internal/terminal-admissions`; the canonical request contains only schema,
event ID and original intent JSON. There is no endpoint discovery, default/global
transport, automatic retry, ambient credential lookup, cookie jar, proxy, or
redirect following. A custom transport is a trusted capability and must enforce
that same destination/cancellation contract; it is not sandboxed by the client.
Success requires HTTP 200 and the exact matching durable Events acknowledgment.
Generic 2xx, conflict, oversized/ambiguous responses, cancellation and response
loss leave the native receipt pending. This native owner transport is separate
from private Sites admission and Flywheel signing.

Events registers the route only when its trusted `TerminalAdmissionBoundary` is
explicitly supplied; production `main` leaves it nil. The boundary must authenticate
and authorize the local caller and supply the current native source instance/epoch
from its existing authority owner, never from the retained facts. The request's
scope must match. Admission retains the original request context and storage
routing while joining boundary cancellation. Current authority is required on
every attempt, including replay after pruning.

Authority covers the actual admission transaction through durable COMMIT. The
acknowledgment proves that already-authorized commit, not current permission or
acceptance of work; it may reach the sender after authority has subsequently
changed. A boundary error or cancellation suppresses acknowledgment. A retry
still reauthorizes. No new authority provider, credential, grant, source policy,
subscription, service, or deployment was installed by this change.

The no-network cross-owner fixture invokes a compiled Events test through local
request/response files, using the real bounded client, optional route, existing
atomic SQLite admission, production-used native repository composition and
worker lifecycle. It first rejects revoked and mismatched source authority,
then loses an acknowledgment after a real Events commit. Reopened native and
Events owners replay the identical request and original acceptance exactly once;
a subsequent native restart performs no further exchange. Independent Events
regressions cover canonical pruning without event resurrection or repeated fanout.
Synthetic subscriptions only populate the fixture queue; none are delivered.
These tests prove local code composition, not live service or credential readiness.

Evidence and current test outcomes are retained under
`.vrooli/evidence/local-work-bridge-20261106/terminal-composition-20261107/`.
The current protobuf freshness check passes; the earlier 23 findings reported by
prior stages are no longer present in the current shared checkout. No generated
files were edited by this composition work. Service-owned Test Genie phases and
real operational restart/dispatch remain unrun under the no-service-start scope.
Source/destination approval, real authority/credential custody, deployment and
first minimized-data transmission remain separate activation gates.

Final native composition qualification: 89 native tests/subtests, 20 shared-client
tests/subtests, and 78 Events tests/subtests passed with race detection. The
qualification manifest lists older opt-in fixture skips separately; the new
actual-owner composition fixture ran explicitly and passed. Final scoped native,
shared-client and Events production-source vet checks passed. Independent final
review found no remaining blocking issues after the source-binding, storage-
routing and cancellation fixes described above.
