# Implementation handoff

Authoritative plan ID: 4a78c13b-1b69-4ffc-b767-85cbf06bca47
Slug: portal-everywhere-native-companion-portable-desktop-control
Store: /home/matthalloran8/Vrooli/scenarios/plan-manager/data/plan-manager.db
Workspace: /home/matthalloran8/Vrooli
Owner-rendered mirror: /home/matthalloran8/.vrooli/plans/portal-everywhere-native-companion-portable-desktop-control.md

The plan is finalized and structurally validated. Status draft means implementation has not started. No product behavior or live-platform acceptance is claimed by authoring validation.

```bash
prompt-manager skill read implementation-plan-execution
plan-manager plans render portal-everywhere-native-companion-portable-desktop-control
plan-manager exec continue portal-everywhere-native-companion-portable-desktop-control
```

Read the next action and capture fresh producer baselines before implementation. Preserve unrelated work. Follow the phase dependencies, mandatory corpus, platform rows, and final whole-collection gate. The pending ledger is a template and intentionally fails validation. The local verifier checks structural completeness, not the truth of receipt claims.

The plan does not authorize public launch, paid purchases, OS permission bypass, or irreversible removal of user data. Build all reviewable deliverables before requesting missing authority. Do not call the overall plan complete while required evidence is unavailable.

## Execution continuation — 2026-09-05

Execution ID: `d3976663-bca9-4cc8-9177-ba079170319c`. The authoring-time draft
statement above is historical. Implementation has started; no phase has earned
completion. The full 31-phase objective remains active.

- Added P0 obligations to Portal, Device Control, Bridge, and Scenario-to-Desktop
  without renumbering old targets or replacing their requirement imports.
- Mapped all 93 mandatory cases to planned requirements in four owner modules;
  see `requirement-case-map.json`. These are obligations, not acceptance receipts.
- Updated Portal's active domain ownership and work-ladder evidence.
- Implemented shared safe target/surface/session references, capability expiry,
  conservative legacy readiness conversion, exact protocol negotiation, strict
  JSON decoding, common proto contracts, and Go conversions in
  `packages/api-core/targetmodel/` and `packages/proto/schemas/common/v1/surface.proto`.
- Generated affected clients through `make generate SCENARIO=common`. Recorded
  boundary extensions for generated closure and api-core module metadata. SDA
  installed the already-approved protovalidate annotation dependency.
- Portal and Bridge requirement validation pass; desktop ramp passes with one
  existing informational heading finding. Device Control reports 21 pre-existing
  unearned-completion findings (no sync snapshot). Do not silence these by
  manually changing completion status.
- Targetmodel tests passed before proto conversion; targeted contract tests pass
  after conversion. The full targetmodel regression command passed after the
  final edits (`go test ./targetmodel`, 8.416s). TypeScript JSON/binary parity passes against the shared Go fixture:
  `node scenarios/portal/docs/internal/plans/artifacts/portal-everywhere-20260905/validate-surface-wire.cjs`.
- Source-manifest verification found all 23 preserved snapshots intact. The
  current-source inventory is in `execution-state.json`, explicitly marked as
  collected after initial implementation edits. Available hosts and unverified
  primary rows are in `support-matrix.md`.

Plan Manager's `exec continue` automatically admitted before-state receipt
`4bcf673f-bdc6-4b98-9fbc-8ef08d0baf6f`. One Test Genie receipt wait returned
terminal FAILED, `IDENTITY_CHANGED`, before producer work began, at
2026-09-05T04:38:04Z. The root comparison includes concurrent search-hub changes.
The failed result is synced to Plan Manager. It is not a usable regression oracle.
Do not re-run automatic baseline capture merely to advance the process; the user
explicitly prefers targeted implementation validation. This divergence is logged.

Next substantive work: integrate shared references through actual Portal and Web
Console projections and owner APIs; complete phase-2 ownership/invariant/retention
contracts beyond the acceptance-case skeleton; then implement the federated
catalog and authenticated desktop-session/helper boundary. Wire-level validators
need actual ingress execution evidence; TypeScript serialization alone does not
execute protovalidate constraints. No native helper, companion, streaming,
migration, performance benchmark, or live platform acceptance is claimed by this
foundation. Keep all mandatory requirements and primary support rows intact.

## Catalog implementation continuation — 2026-09-05 05:03 UTC

Previous goal turn classified as progress: shared contracts and requirement
obligations changed authoritative source, with passing targeted evidence.

This continuation implemented Portal's `api/internal/surfaces` domain and mounted
`SurfaceCatalogService.List/Resolve` in the production API. Provider fanout has a
request deadline and global worker bound, retains healthy sources on failures,
rejects forged owners and duplicate identities, normalizes expired observations,
and resolves labels without choosing between ambiguous results. Production
consumes Web Console's existing typed target catalog. It identifies the API-local
terminal as "Web Console host" rather than claiming companion-local identity.
Terminal offers retain unknown admission state; no desktop readiness is inferred.

CLI `surfaces list` and `surfaces resolve` support labels and exact target/surface
fields. Canonical proto clients and endpoint metadata were regenerated, CLI
registration tests updated, and docs/seams describe the production adapter.
SDA installed the approved generated annotation dependency in Portal API and CLI
modules to satisfy the shared proto import.

Evidence:
- `vrooli scenario test portal unit --wait` run
  `20260905-045802-7f8129fe` ended FAIL, inspected through its producer wait and
  findings artifact. API checksum failure was repaired; UI translation failures
  and missing UI role remain unresolved. Do not call this a passing owner suite.
- Targeted `go test ./internal/surfaces ./handlers/surfaces ./internal/modules`
  passes after the catalog changes; tests exercise HTTP Connect serialization,
  typed provider consumption, partial results, spoofed owners, stale facts,
  ambiguous names, and workers that ignore cancellation.
- CLI `go test ./domains/...` passes, including the new manifest registration.
- `make -C scenarios/portal endpoints` passes.
- `make -C scenarios/portal start` completed healthy with API port 17476 and UI
  port 24965. Live `portal surfaces list --json` returned three terminal offers:
  minimouse, swarminator, and Web Console host. Name and exact-reference resolve
  both returned the single intended local terminal. No session was opened.

Next: extend provider inventory to Device Control/BAS/scenario views and actual
attached topology; expose shared target selection in Portal's ordinary UI;
implement authenticated companion host identity and destination session/helper
admission. Additional critical owner contracts, native adapters, streaming,
companion, migration, and live primary-platform evidence are still unfinished.
All 31 phases remain uncompleted; the full objective is unchanged.

## Surface inspection UI continuation — 2026-09-05 05:14 UTC

Portal Dashboard now mounts a typed SurfaceCatalog panel above the existing
ChatWorkspace. The generated Connect client consumes the production catalog.
Selection stores only exact SurfaceRef identity in tab-scoped sessionStorage;
duplicate labels remain distinct, missing providers preserve selection, failed
refreshes preserve prior entries, and expired capability observations display as
stale. Storage denial does not prevent browsing. English, Japanese, and Arabic
copy is generated through the existing strings pipeline. This panel explicitly
provides inspection only: no surface session or chat routing exists yet.

Fixed the local test wrapper to use Portal's initialized i18n instance and
production ThemeProvider composition. This repairs the observed locale/provider
failures in targeted HealthCard and AppShell tests without weakening assertions.
The earlier owner suite remains a failed receipt; its missing UI role issue has
not been investigated or resolved.

Evidence: 18 tests across SurfaceCatalog, HealthCard, and AppShell pass (Vitest
1.53 seconds); TypeScript noEmit and ESLint for the five changed code/test files
pass. Production lifecycle start succeeds. BAS preview screenshot and normalized
DOM show the panel above the intact conversation workspace at UI port 24965.
The DOM snapshot truncates the panel's descendants, so it does not prove live
option selection; exact-selection and refresh behavior have component evidence.
Temporary browser captures remain under /tmp, not retained as learning payloads.

Next: extend production provider inventory and authenticated destination/session
admission, then connect conversation routing and native companion surfaces.
No native helper, streaming, packaged companion, or primary-platform acceptance
is claimed. No phase is marked complete and the full 31-phase goal remains active.

## Capture readiness continuation — 2026-09-05

Previous turn classified as progress: the production catalog inspection panel
and targeted test harness repair changed source and yielded passing evidence.
Current source inspection found host-desktop readiness based only on DISPLAY
and executable discovery. The existing adapter now runs capture with a bounded
command deadline/output and fully decodes PNG pixels before advertising capture.
A successful header with truncated pixels is rejected. Linux capture explicitly
passes the adapter's display rather than relying on a subsequently changed
process environment. Input permission is no longer inferred from executable
presence; it remains unavailable pending user-session admission.

Targeted `go test ./strategy/hostdesktop ./strategy/registry` passes. Tests use
a present executable that returns denied, invalid, truncated, or valid capture
output; readiness follows actual image decoding. Cancellation and output bounds
are covered. These are adapter tests, not native-platform acceptance.

NAT-01 remains incomplete: the legacy command interface reports unavailable
and cannot distinguish OS permission denial reliably. No helper, session
authentication, isolation, epoch, or native semantic implementation is claimed.
Next work should implement the authenticated user-session/helper boundary and
control-session invariants, replacing the legacy adapter as the desktop path.
Do not treat conservative readiness as a substitute for the required native
control outcome. No phase or overall goal is complete.

## Desktop controller and durable receipt continuation — 2026-09-05

Previous turn classified as progress: bounded real capture readiness and
regression tests changed authoritative source. This turn adds a destination
controller under Device Control's existing internal/sessions domain. It binds
leases to shared SurfaceRef/SessionRef, actor, OS session, helper generation,
expiry, control permission, and monotonically increasing epoch. Open requires
authority verification and explicit takeover for a live lease. Act verifies
authority and current native permission/geometry before durable admission and
again before execution. Identical IDs return receipts; changed command digests
are rejected. Admission is committed before native execution. Failed native
execution or receipt commit yields outcome_unknown without blind retry.

DesktopRepository hides the engine. SQLiteDesktopRepository embeds a per-domain
desktop.sql schema through EnsureSchemas, acquires writer exclusion before
callbacks, and never retries callbacks with external effects. Epoch changes,
stop, and native execution share this exclusion. Stop persists revocation even
when ReleaseHeld reports failure. Receipt count is bounded per lease; payloads
are hashed, not retained in storage. The old session/lease APIs are unchanged.

Evidence: `go test -race ./internal/sessions` passes in 5.870 seconds. New tests
cover same-ID replay, conflict, database close/reopen, native error after effect,
failed completion persistence, wrong destination/session/helper/actor, expiry,
revocation, observation-only grants, invalid geometry, explicit takeover, old
controller stop rejection, failed input release, and takeover between admission
and effect. SDA installed the already-approved generated proto annotation
dependency in Device Control API, needed by shared targetmodel references.

Limits: this is a helper-side controller/storage substrate, not a running
helper. DesktopAuthority and DesktopNative are injection seams with test
implementations; authenticated IPC, OS-native implementations, startup epoch
invalidation, lease-expiry cleanup, existing manual/flow API integration, wire
messages, and live acceptance remain unfinished. No AUTH or NAT case is marked
complete based on these tests. Next work must connect this controller to the
authenticated user-session helper and shared manual/flow exclusion path.

## Signed grant / local peer continuation — 2026-09-05

Previous turn classified as progress: destination controller and durable SQLite
receipts added authoritative code and race-enabled evidence. This turn adds
SignedDesktopAuthority in the same sessions domain. It reuses api-core's
localprincipal.Peer for kernel-authenticated Unix peers and verifies a
domain-separated Ed25519 grant bound to principal, exact lease/session/helper,
actor, epoch, operations, issue time and expiry. The private context key is set
only after peer verification; arbitrary headers/body fields cannot create it.
Unknown operations, wrong protocol, future/expired/oversized grants, forged
signatures, wrong peers, wrong targets, and unavailable revocation status fail
admission. Revocation is consulted on each operation, not just connection setup.

The controller now rechecks authority against the epoch it is about to install,
preventing a valid grant for epoch N from advancing to N+1. Signed authority
tests use real accepted Unix connections and exercise controller Act/revocation
and explicit successor takeover. The signing entry point remains an owner-only
primitive; keys and bearer material are never stored in receipt state.

Evidence: shared-cache compilation first failed because referenced Go cache
artifacts disappeared. The same package then passed under isolated
GOCACHE=/tmp/portal-desktop-go-cache: `go test -race ./internal/sessions`
6.149 seconds. After strengthening the controller integration test, targeted
`-run 'TestSignedDesktop|TestDesktopGrant' -count=1` passed in 1.509 seconds.
Unix peer tests explicitly skip other OSes; this is Linux evidence only.

Remaining: no running helper listener/IPC service is mounted. Trusted public-key
provisioning and rotation, persistent grant issuance/revocation owner wiring,
Windows peer identity, OS-session verification, native adapters, expiry cleanup,
and production manual/flow integration are not implemented by this change.
Continue toward those connections; do not count the authority substrate as
complete native helper or cross-platform acceptance. Overall goal stays active.

## Helper lifecycle continuation — 2026-09-05

Previous turn classified as progress: signed authority and real Unix-peer
tests changed authoritative source. This turn adds trusted ActivateHelper,
ReapExpired, Shutdown, and MaintainLease operations to the desktop controller.
State now persists helper generation, cleanup-pending, and halted disposition.
Helper bootstrap uses prior-generation/epoch CAS, advances the epoch, revokes
the old lease, and preserves uncertain receipts. Stale helpers cannot reap or
stop their successors. Expiry and shutdown revoke leases even when release
fails; cleanup failure stays durable for retry. Failed takeover release also
revokes the old lease instead of preserving input authority. New admission
requires successful release, and shutdown's halted state prevents late reopen.

MaintainLease has a 250 ms expiry tick and bounded cleanup contexts. Its final
shutdown uses a fresh context even when the request/lifetime context is already
cancelled. Errors return to the future lifecycle supervisor; failed release is
never reported as success. Native implementations must honor cleanup contexts.

Evidence: isolated-cache `go test -race ./internal/sessions` passes in 9.908s.
After adding durable halted state, focused expiry/restart/takeover/lifecycle
tests pass in 2.481s. Tests cover failed-release retries, retained uncertainty,
old-helper rejection, fenced generation transition, and cancelled-context
shutdown. No live OS input was used.

Remaining: these are package operations, not a running helper lifecycle. The
helper executable/listener, OS-session bootstrap authentication, native adapters,
key/grant owner wiring, shared existing manual/flow admission, and primary
platform acceptance remain unfinished. ActivateHelper is trusted bootstrap,
not a public client RPC; do not expose it as caller-supplied generation change.
Next work should mount the authenticated helper and native boundary rather than
continue treating tested package seams as native capability delivery.

## Canonical helper transport continuation — 2026-09-05

Previous turn classified as progress: helper lifecycle transitions and tests
changed authoritative code. This turn adds the canonical experimental desktop
proto under device-control/v1/desktop with Open/Act/Stop, exact lease references,
and pointer/key/text action variants. `make -C packages/proto gen-code
SCENARIO=device-control` generated Go Connect, TypeScript, and Python artifacts.
The helper service is internal; it is not registered as a public scenario API.

DesktopUnixHelper now serves accepted Unix connections through the generated
Connect handler. Each request verifies kernel peer identity and signed grant
metadata before dispatch. The service uses bounded body/header sizes, deadlines,
and a 32-connection limit. It runs MaintainLease alongside the listener and
closes the listener/revokes control on lifetime end. Native action validation
remains mandatory behind the typed wire union; the wire admits no shell command.

Evidence: isolated-cache `go test -race ./internal/sessions -run
TestDesktopUnixTransport -count=1` passes in 1.273 seconds. The test uses a real
Unix listener, HTTP transport and generated Connect client with signed grants;
it exercises Open, action receipt/replay, revocation, Stop and post-stop denial.
A fake Native implementation counts effects; no actual desktop input occurred.

Remaining: this is an executable service component, not yet a lifecycle-managed
helper binary or native adapter. The owning bootstrap must provision trusted
keys/grant status, a protected socket, and verified OS-session identity. Windows
transport, Observe/semantic operations, native platform implementations, packaged
companion, public API routing, and full acceptance remain unfinished. Next work
should mount this service in the helper lifecycle and connect a real native
backend; package tests do not complete any primary-platform support row.

## Native X11 continuation — 2026-09-05

Previous turn classified as progress: canonical authenticated Unix transport
and a real socket test changed authoritative source. This turn adds a real
X11 backend under internal/native/x11. It uses explicit local display and cookie
inputs rather than ambient DISPLAY/XAUTHORITY, requires a session-check callback,
probes XTest/RandR, decodes root pixels, and binds pointer actions to captured
geometry plus RandR configuration timestamp. Checked XTest requests implement
move/down/up/click; held buttons are tracked before press and released on cleanup.
A bounded X server grab keeps geometry validation and input contiguous.

Dependency decision: existing XTest runtime library was present but development
headers were missing. Chose pure-Go jezek/xgb v1.3.0 over a new Xlib/XTest C build
toolchain. SDA initially refused the unrecorded package, then its approved
command recorded an exact-version Device-Control-only decision and installed
it. No raw package manager or manual registry edit was used. The governance
record documents BSD-3-Clause/GooglePatentClause metadata and bounded scope;
retain dependency notices and complete packaging/security evidence later.

Evidence: isolated-cache `go test -race ./internal/native/x11 -count=1` passes
in 1.123s after adding server-grab serialization. The test starts an owned Xvfb
fixture, paints known root pixels, captures and verifies RGBA, moves to 120,80,
presses a button, observes its server mask, releases it, and observes the cleared
mask. It rejects stale revisions and a denied session-check callback. No input
was sent to the user's desktop.

Limits: pointer only; keyboard/Unicode, semantic access, physical multi-monitor
acceptance, full format coverage, OS-login/lock callback implementation, helper
bootstrap/binary, and all other native backends remain unfinished. One-screen
32-bit little-endian TrueColor with bounded pixels is the implemented format;
unsupported formats fail rather than fabricate readiness. Next work should
connect native Observe and this backend to the managed helper and authenticated
OS-session bootstrap. No NAT case or primary-platform row is marked complete.

## Authenticated native observation continuation — 2026-09-05

Previous turn classified as progress: real X11 capture/pointer backend and
isolated display evidence changed authoritative source. This turn adds Observe
to the canonical helper proto and regenerated clients. DesktopObservation is
ephemeral pixel/geometry evidence shared by native and controller layers.
Observe requires its own grant permission, admits observation-only leases,
serializes capture with controller state, rechecks authority after capture, and
discards pixels when revocation occurs during capture. RPC PNG encoding has an
independent 32 MiB limit and does not persist pixels in the receipt journal.

A generated-client test now opens a signed session over a real Unix socket,
captures the real Xvfb backend, decodes the returned PNG, sends a click using its
returned display/revision, and verifies X server pointer position and release.
Native input remains confined to the isolated test display.

This integration exposed two defects, repaired locally: concurrent capture and
lease maintenance caused SQLITE_BUSY to terminate the helper; repository lock
acquisition now waits at most five seconds (honoring cancellation) before any
effect callback, never retrying callbacks. A cancellation racing the maintenance
tick was also surfaced as an erroneous shutdown failure; normal lifetime
cancellation now proceeds to the existing fresh-context final cleanup.

Evidence: race-enabled targeted native round-trip/observation-only/transport
tests passed after the lock fix. A broader run then exposed the shutdown race
while all session package tests passed in 4.253s; do not label that combined run
a pass. After the shutdown fix, targeted native round-trip/lifecycle/transport
tests pass (x11 1.621s, sessions 1.682s). Contention regression verifies cancelled
lock acquisition runs zero callbacks, and a callback returning SQLite-like busy
is executed only once. Observation regression rejects actuation for an
observation-only lease and returns no image after mid-capture revocation.

Remaining: no managed helper executable or verified OS-login bootstrap is
installed. Native keyboard/Unicode/semantics, other platforms, companion UI,
streaming, public owner routing and full acceptance remain incomplete. Current
Observe returns inline PNG; evidence artifact/retention integration is still
required. Continue toward managed bootstrap and session binding; do not mark
the full objective, any primary support row, or native phase complete.

## Linux OS-session association continuation — 2026-09-05

Previous turn classified as progress: native authenticated Observe/Act integration
and contention/shutdown fixes changed source and evidence. This turn adds live
desktop-session inspection in control-plane internal/hostinventory and the
canonical `vrooli host desktop-session --session-id ID --peer-pid PID --json`
command. CliDesktopSessionFacts was added to canonical cli/v1/runtime.proto.
The inspector requires explicit logind user/type/active/remote/lock facts and
exact X-server process session cgroup membership, checking process start time
before/after reads for PID reuse. It does not use cached platform inventory.

The X11 backend now extracts SO_PEERCRED from its connected Unix socket. Its
exported NewForSession constructor compares those PID/UID facts with the current
helper UID and fresh control-plane observations before capture/input. The
injected-check constructor is package-private for isolated conformance fixtures.
No session is guessed from DISPLAY and no unlock/elevation/host repair occurs.

Evidence: hostinventory, host application and canonical manifest-dispatch tests
pass for exact match and locked/missing/remote/inactive/user/session rejection.
Native X11 package race tests pass after the constructor change (2.481s).
Normal `make install` completed the supervisory build/vet gate and installed
the control plane. Live inspection matched Xorg PID 3224, UID 1000, session 2
(active local unlocked X11) and rejected PID 1 with peer_user_mismatch. This is
read-only live association evidence, not actual user desktop control acceptance.

Install work: missing shared annotation checksums in packages/vrooli-cli-go
and vrooli-autoheal/api blocked the normal gate. Both were repaired through SDA
using the already-approved annotation version; other supervisory modules were
checked and did not require changes. Boundary extensions recorded these two
manifest/checksum pairs and cli/manifest.json. The live command initially exposed
a missing canonical manifest/handler registration; added it and a dispatch test,
then repeated the normal install gate successfully.

Remaining: managed helper executable/startup and trusted session/cookie/key
provisioning are not yet implemented. Lock-change cleanup beyond pre-operation
refusal, keyboard/Unicode, semantic access, other native platforms, companion
packaging/UI/routing, streaming and primary-platform acceptance remain unfinished.
Do not mark NAT-02/NAT-10 or the overall goal complete from this association probe.

## Helper executable/bootstrap continuation — 2026-09-05

Previous turn classified as progress: Linux session association, installed host
command and live mismatch evidence changed authoritative state. This turn adds
api/cmd/desktop-helper, internal/desktophelper, and a conditional desktop-helper
sidecar in the scenario lifecycle manifest. The command uses the lifecycle
preflight guard and retries unavailable bootstrap without treating process
liveness as desktop readiness or tearing down the optional API/UI path.

Bootstrap loads strict private config, selects an exact local Xauthority cookie,
constructs the production OS-session-verified X11 backend, pins the configured
Ed25519 public key, and consults private expiring grant-status records on each
authorization. No private signing key is loaded by the helper. Private SQLite
state, an exclusive process lock, guarded stale-socket handling, generation
activation, and atomic safe registration precede serving the authenticated Unix
protocol. Files are bounded, owned and non-public; final-component symlinks and
FIFO/device inputs are refused without blocking.

Evidence: race-enabled bootstrap package tests pass (1.021s), helper main builds,
and the added FIFO regression passes (1.033s). Tests cover exact Xauthority
selection, ambiguity/truncation, private-file permissions, symlinks, exclusive
lock ownership, and stale/missing grant denial. `make -C scenarios/device-control
start` completed healthy with no helper config, API 16465 / UI 20698. The optional
helper did not prevent ordinary startup. This does not prove configured helper
launch: no owner bootstrap config was provisioned during this turn.

The first lifecycle attempt failed rebuilding the required Bridge API because
of the shared annotation checksum. Repaired only that required consumer through
SDA using the already-approved version and reran lifecycle successfully.

Remaining: admission-owner key pin/config provisioning, grant-status publishing
and active-revocation cleanup, configured lifecycle launch evidence, owner API
routing, and packaged user-session startup remain incomplete. Windows/macOS/
Wayland, keyboard/Unicode/semantics, streaming and companion delivery are also
unfinished. This is an authored and compiled managed helper entry point, not yet
a delivered configured companion. Continue toward provisioning and configured
launch; no primary support row or overall phase is complete.

## Idle-client revocation cleanup continuation — 2026-09-05

Previous turn classified as progress: managed helper entry point/bootstrap and
optional lifecycle startup changed source and evidence. Before live provisioning,
this turn closes the identified gap where grant revocation or OS lock could
leave held input until lease expiry if the client sent no new commands.

Open now persists the accepted non-secret grant ID after verifying it against
the proposed lease. SignedDesktopAuthority implements a maintenance interface
that checks current owner grant status without retaining a bearer token. The
existing lifecycle tick checks grant revocation and optional native session
liveness in addition to expiry. X11 supplies CheckSession through the existing
production login/lock guard. Failed checks revoke the lease and release held
input; failed release remains pending for cleanup retry. Stop and helper
activation clear stale grant associations.

Evidence: isolated-cache `go test -race ./internal/sessions ./internal/native/x11
-count=1` passes (sessions 4.223s, x11 1.790s). The real native helper/client test
presses a button, revokes the grant while the lease is unexpired, and observes
the X server button mask clear without another helper request. A controller
regression verifies lock detection revokes without client activity. This is
isolated Xvfb and package evidence, not physical OS-lock acceptance.

Next: admission-owner provisioning and a configured lifecycle launch, then
owner API/Portal routing. No configured helper was launched this turn. Broader
native capabilities, companion/platform delivery, streaming, and complete
acceptance still remain; no phase or primary support row is marked complete.

## Local admission and existing lease exclusion — 2026-09-05

Inspection found that the public SessionService accepts actor from its request;
that is not authenticated desktop authority. LocalDesktopAdmission therefore
requires an accepted Unix socket whose kernel peer matches the admission owner.
It derives the actor itself, pins the configured surface/device/helper binding,
and signs bounded observation or control grants without granting takeover. It
acquires the existing Service lease, sharing exclusion with ordinary device
flows. Grant liveness consults that lease; kill/release/expiry and owner restart
revoke grants. This internal owner is not mounted on the public API.

ReadRegistration now checks protected config and current SQLite helper state,
so it uses the current epoch instead of the startup registration epoch and
refuses halted, cleanup-pending, or replaced generations. Registration is still
identity metadata, not proof of a running or input-capable helper.

Evidence: GOCACHE=/tmp/portal-desktop-go-cache go test -race
./internal/desktophelper ./internal/control -run
'Test(ReadRegistration|LocalDesktopAdmission)' -count=1 passed (1.257s/1.060s).
Real local peer/signature tests prove bidirectional device lease exclusion,
observation scope, owner kill revocation, and exact registration binding. No
physical desktop was controlled and no configured helper was launched.

Next: protected owner key/config provisioning, bounded grant-status publishing,
and a lifecycle-managed owner IPC/proxy path that keeps helper tokens server
side. Wire LocalDesktopAdmission through that authenticated path; do not route
request-body actors into it. Then validate configured helper startup and owner
Open/Observe/Act/Stop. Full platform/companion/Portal/acceptance work remains;
all phase-completion flags remain false.

## Bounded cross-process grant-status publication — 2026-09-05

Previous turn made source and validation progress. This turn adds owner
GrantStatus snapshots derived from existing live device leases and the helper's
MaintainGrantStatus publisher. Snapshot expiry is capped at one second and at
the earliest included lease expiry. The existing helper reader now refuses
longer-lived snapshots, malformed IDs, duplicates, missing files and stale data.
Publisher refresh is 250ms, writes are atomic/private, and an exclusive file
lock prevents competing publishers from overwriting or clearing live authority.
Exit/failure writes an empty snapshot; process death loses authority at expiry.

Targeted race validation passed: go test -race ./internal/desktophelper
./internal/control -run 'Test(GrantStatus|LocalDesktopAdmission|ReadRegistration)'
-count=1 (1.822s/1.069s, isolated GOCACHE). Tests prove publication, revocation,
shutdown clearing, stale-owner expiry, rejection of long-lived snapshots, and
competing-writer refusal. No configured helper or physical desktop was launched.

Next remains owner provisioning and lifecycle/API IPC wiring. Call
desktophelper.MaintainGrantStatus with LocalDesktopAdmission.GrantStatus once
owner protected key/config loading is implemented. Coordinate initial issuance
and publication before helper Open to avoid a refresh race; keep tokens server
side. All platform, companion, routing and acceptance obligations remain open.

## Credential-authority-backed owner provisioning — 2026-09-05

Previous turn made source and test progress on expiring grant publication.
This turn reuses the existing credential-authority-go dependency for desktop
signing keys. Service.ProvisionDesktopAdmission derives a stable opaque
credential identity from the surface/OS-session binding, resolves or mints a
32-byte Ed25519 seed through the shared authority, and passes only the public
pin to desktophelper.ProvisionConfig. Config provisioning is serialized with
a private lock and atomic writes; existing destination/session configuration
or public-key pin cannot silently change. Existing pin is credential-loss
witness: lost store entry refuses reminting. Provider failures remain errors,
never evidence that minting is safe. No plaintext private-key file was added.

Focused race tests passed: go test -race ./internal/control
./internal/desktophelper -run
'Test(DesktopProvision|ReadRegistration|PrivateBootstrap)' -count=1
(1.082s/1.219s, isolated cache). Tests verify repeat provisioning retains keys,
private seed is absent from helper config, loss/provider failure cannot mint
or alter the existing pin, and changed OS-session binding is refused. Tests
use an in-memory credential backend; live credentials have not been provisioned.

Next: invoke provisioning from a managed owner startup/CLI path with protected
bootstrap input, run MaintainGrantStatus from the owner, and mount an
authenticated local IPC/proxy that keeps helper tokens server-side. Synchronize
first grant publication before Open. Configured helper launch and end-to-end
owner routing are still unverified; no phase or primary support row complete.

## Managed API owner startup continuation — 2026-09-05

Previous turn made provisioning source/test progress. The API now starts
Service.RunDesktopOwner when DEVICE_CONTROL_DESKTOP_OWNER_CONFIG is set.
Protected OwnerBootstrap contains device_id, helper_config_path, and helper
(the existing Config fields, with public_key omitted). The optional worker
provisions through credential authority, then runs the expiring grant-status
publisher. Failures report and retry every five seconds without failing the API.
API cleanup cancels the worker and waits before closing the owner database.

Bootstrap/config validation rejects insecure files, caller-provided pins,
unclean paths and direct config/status/Xauthority path collisions. The helper
sidecar still uses DEVICE_CONTROL_DESKTOP_HELPER_CONFIG pointing at the
provisioned helper_config_path. Its retry loop tolerates the owner creating that
file after helper process startup. This turn does not yet mount a local owner
IPC/proxy, so the new worker publishes an empty grant set until that path lands.

Evidence: main/control/bootstrap compile; focused bootstrap/provision race
tests pass (1.024s/1.073s); optional-owner failure/cancel test passes. Normal
GOCACHE=/tmp/portal-desktop-go-cache make start rebuilt/restarted Device Control
and reported healthy on API16465/UI20698, with zero failed dependencies. The
owner/helper config env vars were unset: this is optional-absent lifecycle
evidence, not a configured helper launch.

Next: authenticated owner Unix IPC/proxy and coordinated initial grant status
publication before helper Open; then provision live owner bootstrap through
the managed lifecycle and validate exact-session observe/control. Primary
platform, companion, remote, Portal routing and all acceptance work remain.

## Authenticated local owner IPC/proxy — 2026-09-05

Previous turn made lifecycle source and validation progress. Added canonical
DesktopOwnerService Open/Observe/Act/Stop RPCs to desktop.proto and regenerated
Device Control Go/TS/Python artifacts. Owner requests carry exact safe surface
or session references; owner responses omit helper grants/lease tokens. The
managed owner now opens desktop-owner.sock beside protected owner bootstrap,
with private permissions and exclusive stale-socket handling. Every request
checks the accepted socket's kernel principal against the admission owner.
The service is not mounted on public TCP/HTTP.

Owner Open acquires existing device exclusion, synchronously requests and awaits
status publication, then calls the generated helper client over Unix IPC.
Failure releases owner exclusion and republishes revocation. Observe/Act resolve
exact owner sessions and forward only server-held helper authority. Stop invokes
helper cleanup, releases the owner lease and publishes revocation. Lifecycle
shutdown clears publication and releases retained owner leases. Existing
helper durable command identity/epoch checks remain the effect boundary.

Evidence: race tests through real owner/helper Unix listeners and generated
clients pass. Helper authority reads the real published status file, proving
Open does not race the first tick. Observe pixels, Act duplicate suppression,
Stop invalidation/shared exclusion, and failed stale-helper Open rollback are
verified. Native operations are fixture-backed in this test; prior Xvfb helper
coverage is separate and no primary platform claim follows. Publisher tests
also pass. Initial combined control/helper durations 2.035s/2.064s.

Next: configured lifecycle launch with a real protected owner bootstrap and
credential backend, then generated owner-client live exact-session observation
and bounded input in an intended fixture. Add production CLI/Portal routing,
remote Bridge admission and the remaining native/companion/platform intent.
No phase or primary support row is complete.

## Configured live X11 owner/helper observation — 2026-09-05

Previous turn made proxy source/test progress. This turn provisioned the real
owner through credential authority and started both configured API owner and
helper with the scenario lifecycle. Protected runtime bootstrap lives at
/run/user/1000/vrooli-desktop-owner/owner.json; helper config is helper.json in
that directory. DEVICE_CONTROL_DESKTOP_OWNER_CONFIG and
DEVICE_CONTROL_DESKTOP_HELPER_CONFIG were set for make start. Target is verified
Bridge swarminator 697b6224-6283-4a31-90e2-73724e424c05, local OS session 2,
X0 whose kernel peer is Xorg PID3224 UID1000. Control-plane facts confirmed
active/unlocked/local x11. Device exclusion key is registered host-desktop.

First configured helper refused bootstrap. Scientific debugging prior-art
queries: search-hub record query for GDM/Xauthority/bootstrap and swarm-manager
scenario fixes search Xauthority (none). Related records described our new
bootstrap but no matching prior defect: no prior fix for this condition.
Hypotheses were cookie mismatch, session guard, or native format failure.
Non-secret authority metadata showed GDM records with empty display numbers;
our parser required an explicit number. The installed libXau
XauGetBestAuthByAddr accepted these same records for local display0, isolating
the mismatch. Parser now accepts empty-number display wildcard records but
still refuses conflicting cookies; exact Unix destination and logind/kernel
peer verification are unchanged. Focused race TestXAuthority* passed (1.020s).
Lifecycle rebuild/restart then produced an operational helper.

Live canonical owner Open(control=false), Observe, Stop succeeded. Capture
was 1920x1080, valid PNG 1,369,239 bytes. Pixels were hashed and discarded;
metadata/hash retained in live-owner-observation.json. The reproducible
observation-only driver is live-owner-observation.py. After Stop, private grant
status contained zero active IDs and durable helper state had no lease, no
grant ID and no cleanup pending. No physical native input was performed.

The configured owner/helper remain running, with no live desktop grant. Normal
startup env vars are not persisted into source configuration; future explicit
make start calls that should retain this setup must provide both runtime paths.
No full suite/baseline ran. This is real managed X11 observation evidence, not
complete X11 input/semantic acceptance or a full primary support row.

Next: production CLI/Portal surface/session routing, bounded native input
acceptance in an intended fixture, remaining keyboard/Unicode/semantic and
Wayland/Windows/macOS support, companion/Bridge/learning/acceptance intent.
All phase-completion flags remain false.

## Production desktop CLI and live observation — 2026-09-05

Previous turn verified configured physical X11 observation. Added installed
Device Control desktop open/observe/act/stop commands and manifest entries.
Each requires an explicit absolute --socket and typed --request JSON file;
there is no public API discovery or implicit API-host desktop fallback.
Observe requires --output, creates a private0600 PNG without overwriting,
validates PNG dimensions, and emits only safe metadata to stdout.

Generated common surface messages required the already-approved buf.validate
annotation checksum in Device Control CLI; installed via Scenario Dependency
Analyzer deps install on --surface cli. No raw dependency manager was used.
Focused race tests cover real Unix owner-client observation, private output,
no overwrite, pixel exclusion, unknown-field rejection and API independence.
Live execution caught undeclared --output access in Open; fixed and added
actual app-dispatch regressions for open/act/stop. Final focused app/control
race tests pass (1.030s/1.028s). Used canonical cli-core cli-installer to
replace the installed binary before live validation.

Installed CLI Open(control=false), Observe and Stop succeeded on live owner
socket /run/user/1000/vrooli-desktop-owner/desktop-owner.sock. Capture1920x1080,
PNG830486 bytes, mode0600. Temporary pixels removed; hash/safe metadata in
live-cli-observation.json. No physical input performed. API/helper remained
running; no scenario restart was needed for the CLI change.

Next: connect owner surface discovery/session routing to Portal and complete
bounded physical input validation plus native keyboard/Unicode/semantics,
remote Bridge, companion, remaining platforms and full acceptance intent.
All phase-completion flags remain false.

## Physical X11 pointer acceptance through installed CLI — 2026-09-05

Previous turn delivered CLI source/tests and live observation evidence. This
turn verified native pointer effects against a disposable real X11 window.
The validation driver live-cli-pointer.py uses Xlib only to create/query the
fixture and receive events. All pointer movement/click/restoration goes through
installed device-control desktop commands, owner admission and native helper.
It refuses to click unless the fixture is actually under the moved pointer.

Live session2 remained active/unlocked/local and X0 still belonged to PID3224
UID1000. Open(control=true), Observe and Act moved into the fixture. Two Act
requests with the same command ID produced an identical applied receipt and
exactly one native button1 press/release pair at fixture coordinates80,50.
The pointer was restored through the owner CLI, Stop succeeded, and the
window was destroyed. No input was sent to unrelated application windows.

Evidence is live-cli-pointer.json: duplicate receipt equality, received native
events, no held primary button, pointer restoration and Stop. Subsequent
read-only inspection confirmed zero active grant IDs, no durable helper lease
and no cleanup pending. Temporary observation pixels were deleted. No code
change to the production engine was needed and no suite/baseline was rerun.
The harness additionally cleans up its lease if observation fails before input.

This adds real managed X11 pointer/click/dedup evidence; it does not establish
keyboard, Unicode, semantic actions, display-change/lock/restart acceptance,
Wayland/Windows/macOS, remote Bridge, companion or complete primary support.
Next priority: Portal desktop surface discovery and authorized session routing,
then remaining native breadth and product/acceptance obligations. All phase
completion flags remain false.

## Portal desktop surface discovery — 2026-09-05

Previous turn verified real pointer input. Added canonical DesktopOwnerService
Describe RPC and regenerated Device Control proto artifacts. Describe uses the
protected current helper registration to return exact Bridge-host SurfaceRef
and OS session identity. It does not capture, acquire a lease or infer native
permission. Observe/pointer/semantic capability facts remain unknown with
owner_admission_required and bounded freshness.

Portal's DefaultCatalog now includes an optional Device Control provider using
explicit PORTAL_DESKTOP_OWNER_SOCKET. The socket stays server-side and the
provider validates desktop kind, owner namespace, Bridge host relation and OS
session. Missing/failed owner preserves the other healthy catalog sources.
No browser request selects a socket and no API-host identity is substituted.

Focused race validation passed: Portal internal/surfaces (1.047s) and Device
Control owner proxy (1.131s). Tests verify exact host retention, spoofed-owner
refusal, unknown permission facts and no lease acquisition during Describe.
Lifecycle restarts succeeded for Device Control (retaining configured owner/
helper env vars) and Portal with explicit owner socket. Installed portal
surfaces list now returns the live Desktop session2 on swarminator alongside
three existing terminal surfaces; both sources ready. Safe response retained
in live-portal-desktop-catalog.json. Grant status remained empty after discovery.

Portal currently runs with PORTAL_DESKTOP_OWNER_SOCKET=/run/user/1000/
vrooli-desktop-owner/desktop-owner.sock. Retain that env var on explicit future
starts intended to include this desktop. This proves production catalog
discovery, not Portal desktop session opening or conversational actuation.
Next: authorized Portal session routing/UI using exact selected refs, then
remaining native/remote/companion/platform/learning/acceptance requirements.
No phase or primary support row is complete.

## Destination authorization for forwarded Portal accounts — 2026-09-05

Previous turn integrated live catalog discovery. Before exposing desktop
sessions over Portal HTTP, inspection found Portal security headers/logging
but no operator identity context. Reused api-core/owneridentity (local RS256
verification against scenario-authenticator JWKS) at the destination owner
instead of trusting a browser actor field or the API process's Unix UID.

Protected OwnerBootstrap now accepts optional operator_subject. Runtime pins
that subject and initializes the shared validator. Forwarded Authorization:
Bearer credentials are validated on every Unix owner request and must match
that local pin with device-control:read. Control Open and Act additionally
require device-control:write. Any present invalid/mismatched/unsupported
credential denies without local-peer fallback. Bare local CLI requests retain
the existing kernel-peer path. Web-origin leases bind actor to verified subject,
cap helper lease/grant expiry at token expiry, and cannot be reused by dropping
the bearer or substituting another subject. Scope narrowing denies Act.

Focused race tests passed (control1.208s): pin requirement, scope separation,
wrong subject, provider failure, actor binding and expiry bound alongside
existing local owner/proxy exclusion tests. Verifier seam fixtures test the
destination policy; shared owneridentity owns cryptographic tests. No live
operator subject was pinned and no browser control endpoint was mounted.
The configured live owner still runs the previous binary until a lifecycle
restart; it remains authorized only for the already-verified local CLI path.

Next: canonical Portal desktop session proxy forwarding only bearer metadata
to configured Unix owner, then account session/login UI and explicit surface
Open/Observe/Act/Stop UX. Do not bypass the destination pin or omit forwarded
credentials on behalf of browser callers. All broader intent remains open.

## Portal desktop session transport — 2026-09-05

Previous turn added destination forwarded-account authorization. Added
canonical Portal DesktopSessionService Open/Observe/Act/Stop, mounted in
Portal main through DesktopModule using PORTAL_DESKTOP_OWNER_SOCKET. Requests
must have a nonempty bounded Bearer credential. Portal forwards only that
authorization metadata, not cookies, actor/test headers or arbitrary context.
Responses use no-store, bounded messages/deadlines and sanitized errors.
No native operation is retried by the proxy.

Introduced a separate destination DesktopAccountService method namespace.
This prevents older local-only owner versions from ignoring bearer metadata
and interpreting a Portal request as local-user authority. Account service
requires a bearer even from an authenticated Unix peer; the previously added
pin/scope/actor/expiry enforcement then applies. Local CLI OwnerService remains
separate. Proto generation includes the dependent Portal artifacts.

Focused Portal handler race tests pass1.035s: missing bearer never reaches
owner, only intended metadata forwards, errors do not expose owner details,
no-store response and old/local-only service refusal. Destination owner race
tests pass1.154s including account-namespace refusal without a bearer. Portal
module parity tests and endpoint generation pass. CLI manifest declares these
as browser transport omissions; native CLI operations remain device-control
desktop. No broad suite or baseline ran.

These source changes are not yet deployed in the live API processes. Before
positive browser validation, restart Device Control with its configured
bootstrap and a deliberately enrolled operator_subject, then Portal. The
separate AccountService ensures old owners cannot actuate via the new proxy.
Next: account/login UI, operator enrollment/pin, explicit selected-surface
session UX, and live authorized web observation/input. Full platform/native/
remote/companion/learning/acceptance obligations remain open.

## Portal account and desktop session UI — 2026-09-05

Previous turn implemented Portal/account RPC transport. This turn adds
OperatorSessionService Login/Register/Refresh/Logout as a same-origin transport
to discovery-resolved scenario-authenticator, following Device Sync Hub's
existing ownership pattern. Portal stores no passwords or account state.
Responses are no-store; unrelated browser credentials are not forwarded and
private provider errors are sanitized. Proto/module/endpoint metadata updated.

SurfaceCatalog now mounts DesktopSessionPanel for an exact selected desktop.
Panel supports sign-in/create-account, observation-only or explicit control
Open, fresh-image retrieval, geometry-bound primary click and Stop. Credentials
remain in React memory, passwords clear after submission, Blob image URLs are
revoked, and expired sessions disable/clear control. Selection is held during
pending/active desktop work. Unmount stops an active lease; a late Open after
unmount is also stopped. Stopped sessions cannot resurrect an image via a late
Observe response. Uncertain action outcomes prevent further clicks until an
explicit fresh observation. Stop remains available during pending work. The
image's labeled button supports pointer coordinates and keyboard activation
at its center. English/Japanese/Arabic strings are present.

Dependency Analyzer refreshed the local proto-types UI package; this exposed
a QueryClient type split between shared test helpers and the UI. Installed
approved @tanstack/react-query5.59.0 through Dependency Analyzer to align it.
No raw package manager install was used.

Validation: six targeted UI tests pass (desktop panel and existing catalog).
Tests cover memory-only credentials, observation refusal to act, exact geometry,
uncertain-outcome suppression, and cleanup after late Open. TypeScript checks
and targeted ESLint pass after fixes. Portal operator/proxy handler race tests
pass1.038s; module tests pass1.104s. Endpoint generation passes. No broad suite
or baseline ran. UI and new account owner source have not yet been deployed
for live account-mediated validation.

Next: explicit local operator enrollment/pin plus granted account scopes,
managed Device Control then Portal restarts, and rendered/browser proof of
login, exact-surface Open/Observe/control/Stop. Refresh RPC exists but the UI
currently expires its in-memory sign-in instead of automatically refreshing.
Broader streaming/native/platform/remote/companion/learning/acceptance intent
remains unfinished; no full phase or primary support row is complete.


## Live Portal account and browser observation — 2026-09-05 07:42 UTC

Managed Device Control and Portal restarts deployed the account namespace,
operator proxy, and desktop panel. Created a disposable authenticator account
through Register; used its supported GrantScope RPC for exactly
`device-control:read` and `device-control:write`, then Login to obtain current
claims. Verified Portal Open denied the valid scoped account while no protected
operator subject was pinned (401). Added that test subject to the private local
bootstrap and restarted Device Control through its lifecycle.

Live Portal Login/Open/Observe/Stop succeeded. The capture was 1920x1080;
only dimensions/hash/metadata were retained. Missing bearer returned 401 and
Act through the observation lease returned 403. Responses were no-store.
See live-portal-account-observation.json.

The installed rebrowser-playwright test dependency then drove the real Portal
UI: exact desktop option, sign-in, Open observation, decoded 1920x1080 Blob image,
disabled selection while active, disabled image input for observation, no
credentials in local/session storage, Stop removing the image, and sign-out.
See live-portal-browser-observation.cjs and its metadata-only JSON evidence.
The script expects PORTAL_TEST_IDENTITY to name a private credential file and
an already-authorized test operator; it never provisions or pins authority.
An initial browser probe timed out because option visibility was awaited;
corrected to attached state before selection, then the full flow passed.

Cleanup revoked both scopes through RevokeScope, verified empty ListScopes,
logged out the cleanup session, removed the temporary operator pin and private
credential file, and requested a managed Device Control restart. The unused
disposable account remains at the authenticator with no scope grants. Old
access tokens cannot authorize this desktop after the operator pin is removed.
No persisted screenshots or browser traces contain credentials or desktop pixels.

This proves live account-mediated observation on the physical Linux X11 host;
it does not claim browser input, streaming, automatic token refresh, other OS
platforms, or full companion/remote/learning acceptance. All phase completion
flags remain false. Next work should expand native functionality and the
remaining plan intent, using these paths as a validated foundation.

Cleanup confirmed at 07:42:38 UTC: managed restart healthy, operator pin and test credential file absent, active grant list empty, SQLite has no lease or cleanup pending.


## Native X11 keyboard actions — 2026-09-05

Previous turn was progress: it established real Portal account/browser
observation and cleanup. This turn extends native X11 beyond pointer input.
KeyAction now handles named keys and unshifted lowercase-letter/digit keysyms,
resolved from the current server map. Validate and Apply both check geometry
and session admission; Apply runs validation/input under the existing short
server grab. Held keys retain their original keycode for release across map
changes. External held keys cannot be acquired; unmatched UP and PRESS on an
owned held key are refused. ReleaseHeld handles keyboard and pointer ownership,
including session-lock cleanup. TextAction/Unicode remains unsupported and
requires further implementation, as do the other platforms and broader intent.

Targeted race tests passed: native/x11 1.802s, sessions 4.141s, control 3.719s.
New real Xvfb regression verifies actual key state, lock cleanup, preservation
of externally-held keys, named/letter/digit presses, and stale-layout refusal.
No broad suite or baseline ran. The production helper has not been restarted
with this keyboard change; physical keyboard and browser keyboard acceptance
remain pending. CLI keyboard payload and supported names documented in
Device Control docs/reference/cli-commands.md. All phase flags remain false.


## Physical keyboard acceptance and pointer ownership — 2026-09-05

Previous turn was progress: native keyboard implementation plus Xvfb evidence.
This turn managed a Device Control restart to deploy it, then ran
live-cli-keyboard.py against physical X11 session 2 through the installed CLI.
The fixture only used Xlib for disposable window/focus setup and event/query
inspection; all key injection went through owner/helper admission. Initial
key state was empty. One `a` press/release arrived despite duplicate command
submission; ShiftLeft DOWN was released by Stop. Prior focus was restored.
Only metadata is saved in live-cli-keyboard.json; temporary pixels were deleted.
Postcheck found no active grants, durable lease, or pending cleanup.

Inspection also found pointer ownership was weaker than keyboard ownership.
The native pointer validator now refuses unowned UP, refuses CLICK consuming an
owned DOWN, checks external button state before acquisition, and treats repeated
owned DOWN as already held. ReleaseHeld preserves externally-held buttons.
The real Xvfb regression verifies refusal and preservation of an externally
held primary button. Targeted native race tests pass 1.650s. This latest pointer
hardening has not yet been deployed; the live helper includes keyboard support.

Unicode remains unimplemented. Next substantive work includes Unicode/text,
wheel and full native platform/remote/companion intent; keyboard browser UI is
also missing. No entire phase or overall support matrix row is marked complete.


## Native wheel contract and execution — 2026-09-05

Previous turn was progress: physical keyboard evidence and pointer ownership.
Added WheelAction as Action oneof field 4 in canonical desktop.proto, with
observed display/coordinates and signed horizontal/vertical detents. Positive
means right/down; 1–20 absolute total ticks per command. Canonical protogen
published dependent Device Control and Portal Go/TS/Python outputs.

Native X11 validates session/geometry and bounds, then moves and emits bounded
button 4/5/6/7 press/release pairs under the existing brief server grab. Held
cleanup ownership is recorded before each press, removed after release succeeds.
The real Xvfb test verifies all four directions, exact counts and coordinates,
zero/excessive/min-int32 refusal, stale layout refusal, and no held buttons.
Race tests pass: native 2.018s, sessions 3.308s, control 3.057s; final expanded
four-direction focused test also passes. No broad suite or baseline used.

The live helper still predates wheel support and the pointer ownership patch;
managed deployment and physical wheel fixture acceptance are next. Portal UI
has no wheel controls, and its installed file-copy proto package will need the
normal SDA refresh when that UI is added. Unicode remains unimplemented, along
with the broad platform/remote/companion/learning intent. All phase flags false.


## Portal scroll controls and physical wheel boundary — 2026-09-05

Previous turn was progress: native wheel contract/execution and Xvfb tests.
Portal DesktopSessionPanel now has explicit up/down/left/right buttons, each
sending three detents at the observed frame center. Controls only appear for
control leases and follow busy/uncertain suppression. Applied actions refresh
observation; uncertain actions require explicit refresh. Added immediate ref
input lock shared by clicks/wheel to prevent overlapping input before rerender.
English/Japanese/Arabic labels and generated strings updated. Refreshed the
installed proto-types UI file package through SDA. Four targeted UI tests,
TypeScript, and targeted lint pass; wheel test verifies center geometry and
uncertain suppression. Portal production UI has not yet been rebuilt/deployed.

Managed Device Control restart and CLI installer build deployed wheel support.
First live fixture was refused: native tests had not covered the transport's
closed validDesktopAction switch, which lacked WheelAction. Added transport
bounds/finite-coordinate validation and regression; sessions race tests pass
4.955s. Managed restart deployed the fix. live-cli-wheel.py then passed on
physical X11 session 2: one up/right detent pair at the fixture coordinates,
identical duplicate receipt with no duplicate native events, pointer restored,
Stop succeeded, temporary pixels deleted. Metadata in live-cli-wheel.json.
Postcheck: no active grants, durable lease, or pending cleanup.

Live helper now includes wheel and prior pointer ownership hardening. The
physical test proves event delivery, not an application's content-scroll
assertion. Browser wheel end-to-end validation remains pending, as do Unicode,
keyboard UI, platforms, remote/companion/learning and overall acceptance intent.
All phase flags remain false.


## Unicode mechanism spike — 2026-09-05

Previous turn was progress: Portal scroll UI and physical wheel/transport fix.
This turn investigated the required Unicode path. No AT-SPI implementation or
Go D-Bus dependency exists in Device Control. Host has libatspi runtime, GTK3,
Python GI Gtk/Atspi bindings, xvfb-run, and dbus-run-session; atspi development
pkg-config files are absent. SDA withheld attempted godbus/dbus/v5 v5.1.0
install as unrecorded. No dependency was installed or approved. Avoid shell-
driven accessibility as the production implementation.

Primary upstream evidence: GNOME AT-SPI KeySynthType docs explicitly describe
complex/out-of-keymap limitations of KEY_STRING. EditableText.InsertText XML
specifies character offset and UTF-8 byte length. Chosen investigation path is
exact editable-object insertion, which also advances the plan's semantic layer.
Sources:
https://gnome.pages.gitlab.gnome.org/at-spi2-core/libatspi/enum.KeySynthType.html
https://raw.githubusercontent.com/GNOME/at-spi2-core/main/xml/EditableText.xml

atspi-unicode-spike.py runs a disposable GTK Entry under isolated Xvfb and an
isolated D-Bus session. Resolves only the exact child PID and fixture element,
inserts Japanese/Arabic/accented/combining/emoji text at character offset 7 with
39 UTF-8 bytes, and checks BOTH accessible readback and GTK's persisted file.
It passed; metadata is in atspi-unicode-spike.json. It is a mechanism spike,
not production helper integration or NAT-08 acceptance. No physical input,
clipboard access, or production bus mutation was used.

Initial run exposed inherited XDG_RUNTIME_DIR=/run/user/0 and GI get_text method
name collision. Fixed test invocation with XDG_RUNTIME_DIR=/run/user/1000,
GSETTINGS_BACKEND=memory, GIO_USE_VFS=local and explicit Atspi.Text.get_text.
The successful command is documented in the spike; fixture subprocess is
terminated/reaped and temporary persisted text removed.

Next implement a native Go AT-SPI client behind exact session/bus/element
binding, semantic observation/ref validation, and lease checks. Need govern
D-Bus dependency via SDA first (approve-observed supports rationale/reviewer/
range flags if observed evidence exists; inspect explicit approve command for
new exact scoped approval). Do not infer an arbitrary focused element solely
from a valid account or ambient DBUS_SESSION_BUS_ADDRESS. Do not hold an X
server grab across AT-SPI calls to an application, which needs its own event
loop. Unicode integration and overall plan remain incomplete, all phase flags
false.


## Native AT-SPI exact text client — 2026-09-05

Previous turn was progress: isolated GTK Unicode mechanism evidence. This turn
approved github.com/godbus/dbus/v5 v5.1.0 through SDA, exact range, scoped only
to Device Control/native-atspi-desktop; installed through SDA into API. Rationale:
pure-Go native protocol vs shell injection/missing C headers. Upstream LICENSE
is BSD-2-Clause; packaging must retain notices. No raw package manager used.

Added internal/native/atspi/text.go with a native godbus adapter. New accepts an
already-authenticated private accessibility connection and mandatory guard; it
never discovers an ambient bus or autostarts services. Ref binds bus ID, unique
owner, AT-SPI object path and PID. Each operation checks D-Bus GetId/process/UID
and caller guard. Insert compares current text to observed text, checks guard
again immediately before mutation, uses Unicode character offset + UTF-8 byte
length, and verifies exact surrounding-text preservation afterward. Bounded
2-second context and 16KiB text limits. Mutation failure/false/mismatched result
returns ErrOutcomeUnknown; no retry or rollback. Read operations return generic
refusal without leaking provider details.

Targeted race tests pass 1.010s with protocol fixture: mixed Unicode insertion,
character vs byte offsets, changed process/stale text refusal, guard recheck,
and effect-followed-by-timeout emits one mutation and unknown outcome.
This is client-level evidence; it is not real Go-to-GTK integration yet. The
previous Python GTK spike proves the protocol mechanism independently.

Next: protected local accessibility bus connection provisioning/peer checks,
exact semantic element observation/ref model, real Go/GTK integration, and
native session controller/helper wiring. New currently relies on its caller to
provide a trusted private connection and a guard that ties PID to the bound OS
session and active desktop lease. Do not expose it before that integration.
Avoid X server grabs across application D-Bus calls. Unicode TextAction is still
not wired and NAT-08 is incomplete. No phase flags changed.


## Explicit AT-SPI Unix bus transport — 2026-09-05

Previous turn was progress: native exact text client and governed dependency.
Added native/atspi Dial for Linux: only absolute clean Unix socket paths,
SO_PEERCRED PID and current UID verification, mandatory peer/session-policy
callback before any authentication bytes, EXTERNAL-only D-Bus authentication,
private connection with parent-context lifetime, and 2-second handshake budget.
No ambient address discovery, TCP/abstract bus fallback, autolaunch, cookie auth,
or Unix FD passing. Non-Linux implementation explicitly refuses. Caller still
owns trusted bootstrap path and session/lease policy; transport is not yet wired.

Real private dbus-daemon fixture tests verify kernel PID/UID, successful Hello
and GetId, refused peer policy, silent-server deadline and malformed-path
refusal. Added exported editable-text protocol object on this private bus and
ran the actual text Client through D-Bus serialization: mixed Unicode inserted
at character offset 2 after accented prefix, exact readback passed. This is a
real wire test with protocol fixture, not a real GTK application through Go.
Race package tests pass 1.121s. No broad suites, baseline, production mutations,
or dependency changes this turn.

Next: protected bootstrap accessibility bus binding and actual application/PID
session checks, semantic element refs/observation, real Go-to-GTK fixture, then
helper controller transport integration. Do not pass permissive production
Guard callbacks or infer destination from ambient environment. Native TextAction
remains unimplemented at the helper boundary; full Unicode acceptance and all
remaining plan intent remain open. All phase flags remain false.


## Real Go-to-GTK Unicode integration — 2026-09-05

Previous turn was progress: protected-path bus transport primitive and real
D-Bus protocol fixture. Added native/atspi Children and Name operations with
identity checks, 2-second per-operation deadlines, 128-child/4KiB-name limits,
and same-owner child enforcement. Cross-process embedded children require
separate admission instead of silent traversal.

Added TestGoClientInsertsUnicodeIntoGTK plus testdata/gtk_fixture.py. It creates
a private runtime directory, isolated D-Bus session and Xvfb, and real GTK Entry.
Fixture reports only its bus address and PID; Go connects through Dial, resolves
that exact PID's unique bus owner, traverses at most 32 objects using Children/
Name, reads original text, inserts Japanese/Arabic/accents/combining marks/emoji
at Unicode offset 2 after an accented prefix, and asserts GTK's persisted value
preserves the suffix. No Python accessibility mutation occurs. Process group
cleanup is bounded; private runtime/text files are removed. Uses installed GTK
fixture dependencies and skips if unavailable, without installing packages.

Full native/atspi targeted race package passed 1.578s, including actual GTK
integration. Source is authoritative evidence; no primary platform support row
or NAT-08 acceptance claimed because desktop lease/helper wiring remains absent.
Production bus bootstrap/session policy, semantic observation/ref contract and
TextAction integration are next. Avoid an X server grab across GTK calls.
All phase flags remain false; original broad goal remains active.


## Protected accessibility binding configuration — 2026-09-05

Previous turn was progress: native Go-to-GTK Unicode evidence. Added optional
paired accessibility_socket/accessibility_bus_id fields to the protected helper
Config (also carried by OwnerBootstrap). Socket must be absolute, clean, Unix
path length <=107; bus ID must decode to 16 bytes. Incomplete pairs and collisions
with authority, grant file, helper socket/database, bootstrap or generated
helper config are refused. Existing immutable provisioning comparison includes
the fields, so it cannot silently change an existing binding. No live config
was modified; live helper has no accessibility binding.

Added atspi.DialBound: checks configured bus ID shape, uses existing peer-checked
Dial, and verifies daemon GetId on the authenticated connection against the
protected pin. Mismatch closes connection and refuses. Same-path daemon
replacement does not silently rebind. Tests cover real private bus success and
wrong-ID refusal plus protected config pair/path/collision validation.
Race tests passed: native/atspi 1.585s (including real GTK), desktophelper 1.819s.
No broad suite or baseline.

Next: runtime helper must consume optional binding and composite native backend
must route semantic observation/text under existing lease admission. These
fields/connector are currently validated primitives, not active text capability.
Need exact semantic refs and observation revisions, application admission via
bound accessibility bus/session, and no X grab across D-Bus application calls.
Explicit operational rebinding after daemon replacement also remains to design;
do not delete key-pin/config witnesses as an ad-hoc workaround. Original full
plan and all phase flags remain incomplete.


## Lease-authorized semantic observation contract — 2026-09-05

Previous turn was progress: protected accessibility config and bus-ID binding.
Canonical desktop.proto now supports optional process_id on helper/owner
Observe, SemanticObservation (revision, expiry, process ID, opaque element IDs,
labels/editability) in ObserveResponse, and TextAction's element_id,
observation_revision and Unicode character position. Raw OS paths are not part
of the external text action. Canonical generation updated Device Control and
Portal dependent bindings.

DesktopController.Observe delegates to ObserveProcess(...,0); explicit process
requests require DesktopProcessObserver and pass the exact lease SessionID to
the native adapter. Existing observe authority/serialization/postcapture
revocation checks apply to pixels and semantics together. Semantic result must
match requested PID, expire within 5s, have bounded revision/elements/names and
unique IDs. Plain captures may not accidentally return semantic data. Helper
transport maps validated semantic fields and rechecks expiry after PNG encoding;
owner proxy forwards requested PID. Text transport requires opaque element and
observation revision plus nonnegative position.

Race tests: sessions/control 3.129s/3.093s; new semantic freshness/scope/refusal
regressions plus sessions package pass 3.017s. Unsupported process observation
is refused rather than silently returning pixels. Tests verify exact lease ID,
valid result, duplicate ID refusal, expiry refusal and midcapture revocation
with no pixels or semantics returned. No broad suites or baseline.

Next implement composite native adapter: optional protected AT-SPI connection,
process/element discovery with opaque ephemeral lease-scoped cache, semantic
ObserveProcess and TextAction resolve/validate/apply. Use tested native Go AT-SPI
client, bound bus and session guard; avoid X server grab across GTK RPCs.
Runtime still uses plain X11 backend, so process observation currently refuses
and Unicode TextAction still cannot execute. No live deployment this turn.
Portal UI file-copy proto refresh will be needed when semantic UI is added.
All phase flags remain false; broad original intent unfinished.


## Runtime composite semantic backend — 2026-09-05

Previous turn was progress: semantic contract/controller authorization path.
Added AT-SPI ProcessRoot (bounded unique-owner lookup for exact PID; ambiguity
refused) and Editable (role/interface check; password role 40 never cached).
Real GTK integration now exercises both, including true editable result.

Added desktophelper semanticBackend combining X11 pixels/input with AT-SPI
semantics. ObserveProcess captures pixels, enumerates bounded same-process tree,
issues random opaque IDs, caches up to 64KiB editable text in memory, and binds
revision/geometry/5-second expiry to lease SessionID. Text Validate checks cached
lease/element/revision/expiry/position and unchanged text; Apply rechecks session
and editability, clears cache before mutation, then uses exact AT-SPI insertion
with readback. No X server grab spans D-Bus application calls. Stop/ReleaseHeld
and plain Observe invalidate cache. Pointer/keyboard/wheel delegate to X11.

Runtime Run now consumes optional protected accessibility socket/bus-ID config,
uses DialBound with native session check and same bus identity guard, creates
AT-SPI client, and passes composite backend to DesktopController. Missing config
preserves plain X11. Configured but invalid bus refuses startup; no ambient
fallback. Live config has no binding, so this path has not been deployed live.

Race tests passed: desktophelper 1.780s, native/atspi 1.571s, sessions 3.175s.
Composite regression verifies opaque IDs, cross-lease refusal before effect,
changed text refusal, Unicode insertion with surrounding text, single-use
observation invalidation, and Stop clearing cache. Existing Go-to-GTK test
covers real process-root/editability protocol. End-to-end real GTK through
controller/helper transport remains next, followed by managed live binding
and Portal semantic UI. Review late authority changes around AT-SPI subcalls
as part of that integration; client guard currently checks session, while
controller performs lease authority checks at its own admission boundaries.
No entire phase or acceptance row complete; original broad goal remains active.


## Late semantic mutation authorization — 2026-09-05

Previous turn was progress: runtime semantic adapter plus cache tests. Closed
an authority timing gap before further live wiring. DesktopController stamps
Apply context with a private mutation-check callback and rechecks act authority
after native Validate. CheckDesktopMutation rejects absent/cancelled contexts
and revalidates the exact lease against current authority. Adapter callbacks
must use it synchronously within Apply and must not retain it.

AT-SPI InsertTextChecked accepts mandatory caller authority check and invokes
it after identity/session/text comparison, immediately before the D-Bus write.
Composite semantic adapter uses this method with sessions.CheckDesktopMutation.
Standalone InsertText retains its native session guard and wraps a neutral
extra callback for independent client usage; runtime exclusively uses Checked.
No public request header/field can manufacture the controller context key.

Race tests pass sessions 3.142s and atspi 1.561s; prior run also desktophelper
1.777s. New controller regression revokes inside native Apply, checks late
mutation guard, observes zero effects and uncertain durable receipt, then
reauthorizes and verifies duplicate command returns receipt without replay.
Client test confirms final caller denial after the two session/identity checks
produces zero mutations and unchanged text. Existing real GTK remains green.

No deployment or live binding changed. Next remains real GTK through combined
controller/helper transport, then safe explicit live accessibility binding and
Portal semantic controls. Composite unit fixture does not itself assert the
controller callback: the controller and client boundary regressions cover each
side; full combined integration is still required. Broad plan/all phase flags
remain incomplete.


## Signed helper-to-GTK Unicode integration — 2026-09-05

Previous turn was progress: late mutation authority check. Added combined
semantic_gtk_linux_test.go. Initial real-controller-to-GTK version passed,
then strengthened the same test to use SignedDesktopAuthority, real SQLite
repository, DesktopUnixHelper, kernel peer authentication and generated Connect
client over a private Unix socket. Open -> process Observe -> opaque element
TextAction -> duplicate Act -> plain Observe -> Stop -> rejected later Act.

Real GTK entry persists Japanese/Arabic/accent/combining/emoji insertion after
an accented prefix with suffix preserved. Duplicate command receipt matches and
text appears exactly once. Semantic adapter calls the controller's late mutation
check through the real AT-SPI client. No text is stored in durable SQLite state.
Plain observation returns no semantic cache, Stop removes lease without pending
cleanup, and missing-auth request is refused. Fixture processes/runtime/text are
removed. Screen pixels are still a tiny native fixture, explicitly not physical
X11 capture; actual GTK application/text/bus/helper/auth/receipt path is real.

Focused signed integration passed 1.866s; final desktophelper race package
passed 2.649s. No broad suite, baseline, live desktop binding or production
restart. Next explicit accessibility rebinding/provisioning workflow, managed
live helper validation and Portal semantic UI. Existing immutable provisioning
rejects silently changing the helper binding; do not delete pin witnesses to
work around that. Original broad goal and phase flags remain incomplete.


## Portal semantic field controls — 2026-09-05

Previous turn was progress: signed Unix helper-to-GTK Unicode integration.
Portal DesktopSessionPanel now exposes an advanced application-fields section:
explicit process ID, in-memory draft prepared before inspection, inspect button,
editable field selection by opaque ID (duplicate labels remain distinct),
Unicode character position, and insert. Shows the actual observed process ID
separately from the editable request field. Selection clears with semantic
revision changes; expiry disables selection/insertion; handler checks actual
current expiry again. Requires control lease, known editable element, valid
position, fresh revision and no pending/uncertain input. Applied insertion
clears draft and refreshes the same process; unknown/failure suppresses further
input until fresh observation. Draft clears when active lease ends. Existing
click/scroll/Stop behavior retained. Shared field styling and en/ja/ar strings
updated. SDA refreshed the local proto-types UI package.

Six targeted panel tests pass 971ms, including exact second-ID selection among
duplicate labels, Unicode/position payload, requested process forwarding, draft
clearance after applied receipt, and expired reference refusal. Initial tsc
caught spreading a generated Message into a partial fixture; replaced it with
proper initialization fields. Final tsc and targeted eslint pass. No broad
suite/baseline and no live deployment this turn.

This is an advanced explicit-process UI, not the final named application-picker
experience. Five-second reference freshness remains visible; users prepare the
draft before inspection. Need live accessibility binding workflow that preserves
key pins, then real browser-to-helper-to-GTK validation, named application
discovery/picker and other final UX/platform/remote/learning requirements.
All phase flags and overall goal remain incomplete.


## Preserved-key accessibility rebind and physical Unicode — 2026-09-05

Previous turn was progress: Portal semantic UI. ProvisionConfig now permits only
accessibility_socket/bus_id changes on existing config, preserving prior public
pin as credential resolution witness and all destination fields. Requires free
helper state lock. New RunBound checks protected owner bootstrap matches the
generated helper config before taking that lock; sidecar main passes both env
paths, preventing startup race with old generated bindings. Tests cover active
helper refusal before key resolution, preserved pin, other-field refusal,
replacement-pin refusal, disable/rebind, and stale config startup gate.
Initial test temp directory mode was too broad; corrected fixture to 0700.
Final helper race tests pass 2.411s; control race passes 3.832s.

Explicit live owner bootstrap now binds /run/user/1000/at-spi/bus_0 with daemon
GetId 24da8ec5bacb453fac5569676a980a42. (Authentication GUID differs; use GetId.)
No active grant during rebind. Managed Device Control start rebuilt/restarted
and reconciled generated helper config. Public signing pin SHA256 unchanged;
metadata in live-accessibility-binding.json. Installed CLI rebuilt through the
existing cli-installer utility. No key witness deleted and no fresh key minted.

live-cli-unicode.py passed on physical X11 session 2 with real GTK fixture,
explicit environment and protected native bus. Installed owner CLI Open/control,
Observe(process PID) with real 1920x1080 pixels, exact opaque field selection,
Unicode insertion (Japanese/Arabic/accent/combining/emoji) at character offset 2,
duplicate command unchanged receipt and exactly-once persisted text, then Stop.
Temporary pixels/text removed, fixture terminated/reaped, prior focus restored
via fixture Xlib setup/cleanup. Metadata in live-cli-unicode.json. Final check
zero active grants, no durable lease or cleanup pending.

Live accessibility binding remains enabled as intended capability. Operator
subject pin remains absent (browser control still requires explicit enrollment).
Portal UI semantic source is not deployed yet. Next managed Portal restart,
bounded account/pin browser-to-native Unicode validation, and final named
application picker/UX plus other platform/remote/companion/learning intent.
This adds physical Linux Unicode evidence, not full support matrix completion.
All phase flags remain false.


## Physical Portal browser Unicode — 2026-09-05

Previous turn was progress: preserved-pin rebind and physical CLI Unicode.
Managed Portal/Device Control deployment now serves semantic field UI. Bounded
disposable authenticator account received device-control:read/write scopes and
protected operator subject pin solely for live validation. Installed Chromium
opened exact desktop, signed in, acquired control, prepared mixed Japanese,
Arabic, accents, combining mark and emoji text, inspected real GTK process,
selected its opaque editable field, inserted at Unicode character offset 2,
and verified exact persisted GTK value. Stop and logout succeeded; no sensitive
browser storage. Evidence: live-portal-unicode.cjs/json.

Initial live fixture timeout was its exact label matcher: wrapping select label
also includes option text. Prefix label matcher resolved it; successful semantic
API data and UI were correct. No production change required for this finding.
Temporary scopes revoked through Accounts RPC, cleanup login logged out,
operator subject pin removed, credential file deleted, managed Device Control
restart requested to reload unpinned bootstrap. Zero active grants, no durable
lease or cleanup pending verified. Physical accessibility binding stays enabled.
Disposable account remains with no scopes; no account deletion API used.

This establishes Linux physical browser-to-native Unicode evidence. Advanced
PID entry remains interim UX: named application discovery/picker is next, plus
remaining platform, companion, remote, learning and acceptance requirements.
All phase flags and overall goal remain incomplete.


## Native application discovery — 2026-09-05

Previous turn was progress: real Portal browser-to-GTK Unicode and temporary
authority cleanup. Device Control cleanup restart completed healthy.
New atspi.Client.Applications reads the explicitly bound bus, bounded to two
seconds/512 bus names/128 applications/64KiB names. Retains exact Ref (unique
bus owner, PID, bus ID and root path); names are presentation only. Same-name
and same-PID different owners remain separate. Identity/UID/session checks
precede app data, and identity is rechecked before admitting a result. Does not
read editable field contents. Cancellation, overflow, wrong bus or guard refusal
return no list. Disappeared/non-accessible clients and foreign users are omitted.

Real GTK test initially also found registry root named main; fixed discovery to
require ATSPI_ROLE_APPLICATION (75, verified installed GI enum), excluding
registry desktop role14. Deterministic tests cover this, duplicate owners/names,
foreign UID, disappearance, cancellation and bounds. Full targeted native AT-SPI
race tests pass 1.605s, including actual GTK discovery and Unicode persistence.

Not wired to helper/controller/protobuf or Portal picker yet. Next: lease-scoped
opaque application references with freshness and exact retained owner for
inspection, generated transport and named picker. Do not resolve a later
selection by PID/name: both can be reused. Existing manual PID UI remains until
that integration. All phase flags and goal remain incomplete.


## Lease-bound application references — 2026-09-05

Previous turn was progress: native discovery and GTK evidence. Added ephemeral
sessions DesktopApplications/Application types and DesktopApplicationObserver.
Controller Applications uses observe authorization, admitted lease and serialized
repository before/after discovery, validates bounded unique IDs/names/PIDs and
30-second catalog freshness; no catalog persisted. ObserveApplication shares
existing capture/semantic admission and validates nonempty bounded ID/revision.

semanticBackend now discovers apps into UUID -> exact atspi.Ref cache bound to
lease session ID, catalog revision and 30-second expiry. Inspection uses retained
unique owner via common observeRoot, never re-resolves PID/name. Field references
retain existing independent five-second freshness and lease binding. Catalog
refresh clears old application and field refs; Stop/ReleaseHeld clears both.
Plain capture clears fields but leaves unexpired application refs available.
Native access discovery is an optional interface, permitting unsupported hosts
to refuse honestly while retaining pixel support.

Targeted desktophelper/session race packages pass 2.626s/4.357s. New regressions
prove cross-lease/revision/unknown/expired/refresh/Stop refusal, same-PID owner
replacement refusal without ProcessRoot calls, exact selected owner, duplicate
catalog IDs and stale catalogs refused, and revocation during discovery/capture
discards results. Existing signed helper real GTK Unicode test still passes.
New application selection is not yet transported or tested with live browser.

Next: canonical protobuf Applications RPC and Observe application ID/revision
across helper/owner/account/Portal; generated clients; UI application picker
replacing PID entry; end-to-end real GTK discovery selection. All phase flags
remain false and full plan stays active.


## Application transport and signed GTK selection — 2026-09-05

Previous turn was progress: lease-bound exact-owner references. Canonical
desktop proto now has Applications RPC on helper/owner/account services,
Application and ApplicationsResponse catalog messages, lease/session requests,
and application_id/application_revision fields3/4 on both Observe requests.
Portal DesktopSessionService exposes the same owner Applications types.
Generated Go/TS/Python through make packages/proto gen-code SCENARIO=device-control
(published both Device Control and Portal). No generated files hand edited.

Helper routes discovery to authorized controller, maps catalog, and routes
Observe application selectors to exact reference selection. Mixed PID and
application selectors refuse, as do partial application selectors. Owner
Applications resolves existing admitted session and keeps signed helper lease
internal; Observe forwards exact ID/revision. Account handler shares that owner
implementation under existing bearer identity checks. Portal Applications uses
account namespace and existing bearer-only forwarding/no-store/error redaction.
Added endpoint descriptor and browser-only CLI manifest omission.

Signed helper real GTK fixture now discovers one GTK application, verifies name
and PID, then selects its opaque reference to inspect fields and complete exact
Unicode insertion/dedup/Stop. Missing discovery auth and mixed selectors refuse.
Fixture guard now checks bound bus ID (per runtime) instead of one fixture PID,
allowing registry scan; native identity still validates actual current UID/PID.
Desktophelper race passes 2.708s. Owner proxy test checks full discovery plus
selected semantic forwarding: control race passes 3.321s. Portal handler test
checks discovery bearer requirement, metadata filtering and no-store: passes
1.044s. Initial sessions/control/helper race passed 5.934s/3.898s/2.929s; Portal
module tests passed 1.162s.

Next: refresh Portal generated dependency through SDA, replace manual PID UI
with named application picker/revision selection, validate targeted UI tests
and real browser-to-GTK application discovery after managed deployment. New
transport not deployed yet; installed CLI still older schema. All phase flags
remain false and full goal active.


## Portal named application picker — 2026-09-05

Previous turn was progress: generated discovery transport and signed GTK
selection. Refreshed Portal proto-types through SDA approved dependency install.
DesktopSessionPanel now replaces PID entry with Find applications + named
application select. Same-name applications are separate numbered options with
opaque values. Explicit accessible select label avoids option-text label match
issues. Discovery uses current lease/account, discards late responses after
Stop, clears old catalog/selection/fields before refresh, preserves prepared
text draft, and shows empty/expired results. Expiry disables selection/inspection
and handler rechecks actual time. Selection changes clear inspected fields.
Inspection forwards applicationId + applicationRevision only. After successful
text mutation, captures plain frame (never re-resolves by semantic PID). Stop
and inactive lease clear catalog and drafts. EN/JA/AR strings generated.

Eight targeted panel Vitest tests pass; TypeScript noEmit and targeted ESLint
pass. Tests exercise duplicate application names and exact second ID/revision,
Unicode field selection, expired catalogs/fields, late discovery after Stop,
clearing fields on app change and no PID lookup after insertion. Initial test
edit accidentally replaced create() schema argument as well as import; corrected
fixture, then all tests passed. No production issue from that failure.

New source is not deployed yet. Next managed Device Control and Portal starts
with explicit owner/helper env config, bounded account/pin and real browser GTK
fixture updated to Find applications and select fixture by name. Existing
live-portal-unicode.cjs still uses removed PID label; adapt while preserving
its old evidence as previous behavior. Cleanup temporary scopes/pin afterwards.
All phase flags and full goal remain incomplete.


## Live physical application picker — 2026-09-05

Previous turn was progress: Portal named application picker and targeted tests.
Managed Device Control and Portal starts deployed latest source healthy.
Bounded temporary authenticator account/scopes and operator pin enabled live
browser validation. New live-portal-application-picker.cjs retains original
PID-based fixture/evidence separately. Installed Chromium selected exact desktop,
signed in, acquired control, discovered applications, selected unique named
gtk_fixture.py option, prepared Unicode text, inspected chosen app, selected
opaque GTK field and inserted at character offset2. Persisted GTK text exactly
matched mixed Japanese/Arabic/accent/combining/emoji with prefix/suffix intact.
No sensitive browser storage; Stop/logout passed. Fixture child and temporary
text removed. Metadata: live-portal-application-picker.json.

Temporary scopes revoked, cleanup login logged out, operator subject pin and
private credential file removed. Disposable account remains with no scopes.
Zero active grants, no durable lease or cleanup pending verified. Managed Device
Control restart reloads unpinned bootstrap; accessibility binding remains enabled.
Picker is now deployed; installed device-control CLI schema may still be older.

This proves physical Linux browser application discovery/selection and Unicode
insertion. It does not close platform/companion/remote/learning acceptance. Next
re-read plan phase intent and current state to choose next major gap; avoid
repeating this passing slice. All31 phase flags remain false and goal active.


## Window ancestry in semantic observations — 2026-09-05

Previous turn was progress: physical application picker validation. Re-read
phase11 semantic/window requirements and phase14 UX; companion directory still
absent, many major plan gaps remain. Current flat fields lacked window context.
Added native AT-SPI Role read with identity guard, semantic parent/window opaque
IDs and role metadata through canonical proto/generated clients and helper
transport. Native frame23/dialog16/window69 roles verified using installed GI.
Role values currently AT-SPI native metadata, not portable normalized selectors.
Traversal retains per-node parent and nearest window identity; repeated nodes
and cycles refuse. Controller requires parent to precede child and window ID
to be self or actual ancestor. No raw native object paths leave helper.

Native/session/helper race tests pass 1.609s/4.619s/2.769s including real GTK
window-name/ancestry checks in signed helper Unicode fixture. Added duplicate
field-name two-window fixture showing distinct field IDs and proper containing
window IDs; final helper race passed. These are metadata foundations; Portal
window picker and selector zero/unique/ambiguous dispositions still pending.
Next expose window context in UI and implement explicit window-constrained
resolution. New ancestry proto not deployed or SDA-refreshed in Portal yet.
All31 phase flags and full goal remain incomplete.


## Explicit Portal window selection — 2026-09-05

Previous turn was progress: generated window ancestry and native tests. Refreshed
Portal proto-types through SDA. DesktopSessionPanel now presents named windows
from self window references, requires explicit selection (even one window), and
filters editable fields to that exact window. Duplicate labels retain distinct
opaque option values; no lowest-ID or first-item selection. Window change clears
field selection; semantic revision change clears window too. Text handler and
button both require a field belonging to selected window. Empty window/field
sets have explicit messages; missing ancestry cannot masquerade as a usable
window. EN/JA/AR strings generated.

Eight targeted panel tests pass 350ms; TypeScript noEmit and targeted ESLint
pass. Unicode test now has same-name windows and same-name fields, switches
windows, verifies old selection cleared, and inserts through exact second
window field reference. Existing expiry and Stop tests continue passing.

Source not deployed. Next adapt physical picker fixture to select named GTK
window, managed deploy, validate live and clean temporary permissions. Existing
GTK signed-helper fixture verifies window ancestry but full browser window
selection still pending. Automated zero/unique/ambiguous semantic resolver,
normalized cross-platform roles and remaining plan phases still incomplete.
All31 phase flags remain false and full goal stays active.


## Physical window picker and stale API workaround — 2026-09-05

Previous turn was progress: explicit window picker and tests. Managed starts
deployed UI and Device Control; Portal API remained stale despite healthy
restart. Its binary mtime05:00:30 preceded generated desktop.pb.go05:02:56,
while Device Control binaries rebuilt05:06:47. Two browser fixture failures
showed successful semantic responses with absent parent/window/role fields and
empty window selector. Each failure stopped lease and removed fixture.
Explicit make -C scenarios/portal build followed by managed restart fixed it.
Report-bug published scenario-qa knw-1788599328849144167 for imported generated
Go freshness detection. No lifecycle implementation changed in this work.

New live-portal-window-picker.cjs/json passes physical Chromium -> Portal account
-> signed helper -> real GTK application/window/field selection and exact
Unicode persisted text. Window options include Go AT-SPI Unicode fixture.
Stop/logout/no-sensitive-storage passed. Temporary permissions revoked, cleanup
login logged out, operator pin and private credentials removed. Zero grants,
no lease/cleanup pending verified; managed Device Control cleanup restart.
Disposable account remains without scopes. Prior evidence files preserved.

Window picker now deployed and live-validated. Next automated window-constrained
semantic resolver with explicit zero/unique/ambiguous results; existing strategy
SemanticResolver seam in api/strategy/contract.go (currently Android consumer
in internal/control/execution.go) should guide reuse, without bypassing new
helper lease authority. Portable role mapping, geometry/crops and many major
platform/companion/remote/learning requirements remain. All phase flags false.


## Authorized semantic resolution domain — 2026-09-05

Previous turn was progress: physical window selection plus reported lifecycle
freshness defect. Existing strategy SemanticResolver returns bounds/confidence
only and cannot express helper lease references or ambiguity. Added sessions
DesktopSelector/Resolution and optional DesktopSemanticResolver. Controller
Resolve enforces observe admission before/after serialized native resolution,
bounded exact window/name/revision, freshness and disposition/cardinality.
Helper retains observation metadata beside exact native entries, clears it on
cache invalidation, matches within exact selected window and returns absent,
unique, ambiguous with IDs/revision/geometry/expiry. Current names must still
match observed names; renamed/stale/cross-lease refs error. No lowest-ID choice.

Initial sessions/helper race passed 5.450s/2.819s. New resolver test covers
window restriction, absent/unique/ambiguous, geometry reference, cross-lease,
unknown window, expiry and Stop clearing. Final helper race passed.
Not yet exposed in protobuf/CLI/owner/Portal nor integrated with flow executor.
More controller revocation/cardinality tests and native rename/tree-mutation
coverage needed. Exact-name editable/window subset is only initial selector
contract; portable role/automation-ID/state constraints remain. All phase flags
false and full goal active.


## Resolver tree freshness and authority regressions — 2026-09-05

Previous turn was progress: initial window-scoped resolver. Inspection found
name-only freshness could miss newly added duplicate fields or changed
editability. Semantic entries now retain copied child Ref lists. Resolve
rechecks children, role and editability alongside name for selected window
elements; mismatches refuse instead of returning false unique/absent results.
Lists compare exact order conservatively. Native changes are not atomic across
D-Bus reads; no stronger claim is made. Existing five-second freshness and
independent mutation authority/text postcondition checks remain necessary.

Targeted sessions/helper race tests passed 3.645s/2.586s. New tree fixture tests
exercise added child, rename, role change and editability change after Observe.
Controller tests check lease ID forwarding, disposition/cardinality mismatch,
unknown dispositions and revocation inside resolution with discarded IDs.
Existing signed GTK observation/text fixture continues passing.

Next canonical Resolve RPC across helper/owner/account/Portal, real GTK resolver
transport tests, then integration with automation and broader selector intent.
Still not deployed or exposed externally; all31 phase flags remain false.


## Resolve RPC and actual GTK transport — 2026-09-05

Previous turn was progress: native tree freshness and revocation tests. Added
canonical SemanticSelector, ResolveRequest/OwnerResolveRequest, ResolveResponse
with enum unspecified/absent/unique/ambiguous, IDs, revision, geometry and expiry.
Resolve RPC now on helper/owner/account/Portal services; generated through
packages/proto gen-code SCENARIO=device-control. Helper rejects missing selector
and uses authorized controller; owner resolves current session and forwards
exact selector with internal signed grant; Portal uses existing bearer-only
account forwarding. Endpoint descriptor and browser-only CLI omission added.

Real signed GTK helper test verifies missing auth refused, unique exact field
resolution and absent result through generated RPC, then original Unicode
insertion/dedup/Stop. Sessions/control/helper race pass 3.733s/3.305s/2.539s.
Portal surfaces/modules race pass 1.045s/1.129s. Added explicit owner forwarding
regression checking every selector field and unique response; final control
race passes. No new live deployment or browser resolver integration yet.

Next expose installed CLI resolver/discovery surface and integrate automated
flow consumption with explicit ambiguity handling. Broader portable roles,
automation IDs/state constraints and major plan phases remain. Portal API may
need explicit make build on imported-proto-only changes (reported freshness
defect knw-1788599328849144167). All31 phase flags remain false.


## Installed CLI discovery and resolver — 2026-09-05

Previous turn was progress: generated Resolve transport and signed GTK tests.
Desktop CLI now exposes applications and resolve through the explicit Unix
owner socket, typed request files, bounded response and existing JSON output.
Observe automatically accepts generated application ID/revision fields. Resolve
preserves disposition and all ambiguous IDs; it does not choose or act.
Manifest local bindings/read governance and configuration reference usage added.
CLI control race tests pass 1.141s, including exact session/selector forwarding
and ambiguous result preservation. Rebuilt installed device-control through
cli-core cli-installer with forced freshness; installed help for both commands
verified socket/request/JSON flags.

No live owner Resolve invocation yet: managed Device Control deployment is
still before Resolve source. Next deploy and use existing physical CLI GTK
fixture adapted to application/window resolution; retain exact persistence and
cleanup. Then integrate with automation flow semantic path; legacy resolver
still returns bounds/confidence and must not bypass helper authority.
All31 phase flags remain false; broader goal active.


## Physical installed CLI resolution — 2026-09-05

Previous turn was progress: CLI commands and installer validation. Managed
Device Control start rebuilt/deployed Resolve healthy. Adapted prior CLI Unicode
fixture into live-cli-resolution.py, retaining prior evidence. Installed CLI
Open/control -> Applications exact GTK identity -> Observe application reference
-> window identity/name -> Resolve unique field -> Resolve absent name -> Act
using returned field ID -> duplicate -> persisted exact Unicode -> Stop passed
on physical session2 with1920x1080 capture. Metadata live-cli-resolution.json.
Temporary text/pixels removed, GTK child reaped, prior X focus restored. Final
zero active grants/no durable lease/no pending release; operator pin absent.
No temporary account used for local owner CLI.

Inspected existing control/execution.go semantic-target: it resolves via old
strategy bounds/confidence, then dispatches pointer center. That path cannot
represent desktop helper element references or exact Unicode text and must be
extended at its owner boundary, not fed guessed coordinates. Next design and
implement explicit desktop flow authority/semantic action path, retaining
existing Android behavior. All31 phase flags remain false; full goal active.


## Flow semantic reference adapter — 2026-09-05

Previous turn was progress: physical CLI resolver evidence. Inspected existing
Service.execute: strategy selection, capability validation and ordinary device
lease occur before step dispatch. Desktop owner already holds that device
exclusion, so integration cannot acquire a second lease or use old coordinate
semantic-target. Added internal/flows ResolveDesktopField with session-bound
resolver call seam. It selects an exact unique window name, requests editable
field resolution, preserves absent/ambiguous errors, validates result revision/
geometry/expiry and field membership, and returns TextAction identity with no
text/mutation. No lowest-ID selection or coordinate conversion.

Targeted flow adapter race test passes 1.018s covering exact identity, duplicate
windows without resolver call, ambiguous fields, foreign field and changed
geometry. Initial relative-path file writes failed; initial no-tests run is
not evidence. Correct absolute files then tested successfully.

Adapter is not called by execution yet. Next add owner-session flow execution
using existing flow language/result/library semantics while preserving single
admitted desktop lease and unknown-outcome receipts. Do not advertise flow
integration from this helper alone. All31 phase flags and full goal incomplete.


## Owner-client semantic text step — 2026-09-05

Previous turn was progress: flow resolution adapter. Added DesktopStepOwner
Observe/Resolve/Act interface matching generated owner/account clients, and
ExecuteDesktopText composition using existing session/application references.
Input validation precedes observation; unique window/field resolution precedes
Act; exact command ID and geometry/revision are retained. No lease acquisition,
no retry, no pointer fallback. Unknown receipt returns unchanged for executor
to halt. Caller owns auth metadata and session lifecycle.

Targeted TestDesktop flow race cases pass 1.017s. New tests prove ambiguous
resolution causes zero Act and unknown outcome causes exactly one Act with
exact Unicode field targeting. Existing identity/staleness adapter tests pass.
This is one-step composition, not full flow executor integration: multi-step
endpoint, run cancellation/tracking, saved-flow evidence/promotion and real
owner-client execution still pending. Keep this limitation explicit. Next
connect composition into admitted desktop flow run boundary and test live owner
path. All31 phase flags and full goal remain incomplete.


## Bounded desktop flow runner — 2026-09-05

Previous turn was progress: one-step owner-client composition. Added
internal/flows ExecuteDesktopFlow using existing execution.Flow/RunResult types.
Requires desktop transport and explicit admitted application binding, validates
entire flow before calls, allows only desktop-text steps, rejects unsupported
flags/arguments/capabilities, validates UTF8/text/position/timeouts, max32 steps
and30-second total. Per-step Observe/Resolve/Act uses unique runID:index command
IDs. First error/missing/non-applied receipt halts incomplete; chapter messages
do not expose text. Existing session lifecycle remains caller-owned.

Targeted TestDesktop race tests pass: invalid later step has zero mutation,
unknown first step stops sequence, two confirmed steps complete with distinct
commands, cancellation prevents next mutation. Initial compile error used
wrong prepared-step field name; fixed and tested. Full endpoint/run tracking/
saved-flow promotion integration still absent, and runner is not deployed.
Next bind it to desktop owner RPC with authoritative run identity/idempotency
and cancellation; do not infer full flow support from this internal runner.
All31 phase flags remain false and goal active.


## Durable desktop flow admission — 2026-09-05

Previous turn was progress: bounded runner. Added DesktopFlowClaim records
inside existing per-desktop repository state (no engine leak/new schema).
ClaimFlow validates control/act authority and admitted lease, requires bounded
run ID and SHA256 hex digest, commits before work, returns fresh=true once.
Duplicate matching claims return prior record/fresh=false even after repository
reopen; conflicting digest refuses. Max128 claims per lease; new admitted lease
clears claims alongside command receipts. Old lease references cannot reuse
new lease claims. No flow text, token or payload retained.

Sessions race tests pass including reopened repository duplicate, conflict,
concurrent only-one-fresh and revoked duplicate refusal. First equality test
exposed monotonic timestamp serialization mismatch; UTC claim timestamps fixed
canonical readback. This is admission only, not completion persistence: no
terminal flow receipt yet, no helper RPC nor runner wiring. Next add terminal
result tracking and expose internal helper claim operations before owner Run
endpoint. All31 phases remain incomplete.

Correction to preceding validation chronology: initial passing claim/journal
252f18f9-fb12-46e0-ad95-4f28ddf47635 was written before test result inspection.
After UTC fix, contention test returned SQLITE_BUSY. Attempted broader acquisition
retry did not resolve it and was reverted. Final contract test permits a bounded
busy refusal for the losing claim, requires exactly one fresh admission, then
retries admission sequentially and proves it returns fresh=false. No mutation
callback retry was added. Final full sessions race PASS 4.619s. Earlier premature
evidence should be read with this correction; no claim of contention-free
response delivery.


## Terminal flow metadata and command gate — 2026-09-05

Previous turn was progress: durable claims (corrected final evidence). ClaimFlow
now requires expected step count1..32, bounded colon-free run ID<=120, and
stores claimed disposition. Duplicate identity includes step count. FinishFlow
requires current control authority and matching claim/digest; passed requires
applied receipts for every canonical runID:index command. Incomplete may record
a confirmed prefix. Terminal disposition is immutable, timestamps UTC; no
flow text stored. Duplicate claim returns terminal metadata without execution.

Act checks claimed-run command index/range and terminal state at both admission
and final effect transaction. Existing identical command receipt remains
readable, but a terminal run cannot admit new effects. This checks helper
receipt completion, not independently the user's whole workflow intent.

Final sessions race pass3.527s. Tests prove unconfirmed pass refused, actual
command receipt permits pass, reopening returns same terminal claim, conflicting
terminal finish refused, incomplete blocks later mutation. Earlier incremental
runs pass4.066s/3.869s. Next helper ClaimFlow/FinishFlow RPC, owner flow endpoint
using stable admitted run ID (current internal runner still mints its own UUID),
then end-to-end tests. No endpoint wiring/deployment yet; all31 flags false.


## Signed flow journal protocol and stable runner identity — 2026-09-05

Previous turn was progress: terminal flow records. Canonical helper-only
ClaimFlow/FinishFlow RPC and FlowRecord now generated. Requests carry signed
lease plus run ID/digest/steps or finish disposition; response has fresh flag
and bounded receipt metadata. Controller enforces control authority. Owner/
account public flow endpoint still pending; internal journal not exposed there.
Signed GTK test claims before mutation, duplicate claim not fresh, premature
passed finish refused, actual Unicode command fixture-flow:0 applied, finish
passed with confirmed1, duplicate claim returns terminal passed without fresh
authority; duplicate command receipt still readable.

Sessions/helper/control race pass3.876s/2.790s/3.900s. Runner now offers
ExecuteDesktopFlowWithID for a caller holding fresh journal admission; default
wrapper remains. Targeted flow race pass1.018s, stable claimed-run:0/:1 IDs
verified. Runner call itself does not claim: caller must enforce fresh admission.
Next owner RunFlow endpoint: full preflight before claim, deterministic digest
of exact request, duplicate returns record, bind owner calls to original actor,
execute with admitted ID, finish without replay, cancellation and Stop handling.
No new live deploy; all31 phase flags false, full goal active.


## Owner RunFlow endpoint — 2026-09-05

Previous turn was progress: helper journal RPC and stable runner IDs. Added
OwnerRunFlowRequest with exact session/application refs, caller run ID and
existing flows.Flow message; RunFlow exposed on owner/account namespaces.
Canonical generation publishes Device Control/Portal. Owner converts flow,
preflights via shared ValidateDesktopFlow before claim, hashes deterministic
full request, resolves current caller/control authority, claims signed helper
journal. Duplicate returns record without execution; conflicting digest refuses.
Fresh claim runs with same caller context and stable ID, using p.Observe/Resolve/
Act (no extra device lease). Owner mutex released during run, so Stop can revoke.
Finish uses bounded2sec cancellation-detached context with preserved identity;
if admission revoked it refuses, leaving claimed rather than inventing success.
Response is durable FlowRecord; detailed run/library promotion still pending.

Initial control/flows race pass3.836s/2.106s. New Unix owner test covers actual
signed helper flow claim/execution/finish with native fixture, passed confirmed1,
duplicate no new effect and changed request same ID refused. Final control
race pass3.224s. Not real GTK flow execution yet; previous GTK journal and
physical CLI operation evidence remains narrower. No new deployment or CLI
run-flow command; Portal RunFlow forwarding pending. Next add cancellation/
unknown endpoint cases and real GTK multi-step endpoint fixture, then expose
CLI and saved-flow replay/promotion. All31 phase flags remain false.


## Physical multi-step desktop flow — 2026-09-05

Previous turn was progress: owner RunFlow endpoint. Added installed CLI run-flow
typed call/local write manifest. CLI owner timeout now40sec to accommodate
bounded30sec flow +finish. Rebuilt via cli-installer and managed Device Control
start deployed endpoint healthy. Existing CLI control race passes1.142s.

New live-cli-flow.py discovers actual physical GTK application, submits two
desktop-text steps through installed CLI using named window/field selectors,
verifies exact combined Japanese/Arabic/accent/combining/emoji text and suffix,
then resubmits identical run and gets identical terminal record with confirmed2
and no extra insertion. Stop passed; fixture/text temporary files removed and
prior X focus restored. Zero grants/lease/cleanup pending verified. Evidence
live-cli-flow.json. No temporary account/operator pin used. Usage/retry contract
documented in existing configuration reference.

Next owner cancellation/unknown-outcome endpoint coverage, Portal RunFlow
forwarding and saved-flow promotion/replay integration with explicit checks.
This proves Linux physical two-step text flow, not all flow/platform acceptance.
All31 flags remain false; full goal active.


## Portal RunFlow forwarding and response budgets — 2026-09-05

Previous turn was progress: physical CLI two-step flow. Added generated Portal
DesktopSessionService RunFlow reusing owner request/FlowRecord. Handler uses
account namespace, exact request and bearer-only forwarding/no-store/no retry.
35-second call deadline for RunFlow; other desktop operations remain10sec.
Owner HTTP client40sec. Found outer owner socket WriteTimeout15sec and Portal
server default30sec would cut off bounded30sec execution plus finish; changed
owner socket and Portal server WriteTimeout to40sec. Helper per-call deadline
unchanged. CLI omission and endpoint descriptor added.

Portal surfaces/modules race pass1.048s/1.174s, covering auth requirement, exact
run ID and claimed disposition, no-store, metadata filtering,35sec deadline
and no retry on error. Control race pass3.968s after owner timeout change.
Portal main compile-only check passed (no tests, not runtime deadline proof).
No live Portal flow request yet, no browser flow UI; current generated package
not SDA-refreshed. Next managed builds/deploy and bounded account Portal flow
validation, plus cancellation/unknown endpoint coverage and saved-flow linkage.
All31 phase flags remain false and full goal active.


## Physical Portal account flow — 2026-09-05

Previous turn was progress: Portal flow forwarding/timeout corrections. Explicit
Portal build plus managed restart, managed Device Control restart deployed
latest. Bounded disposable account/read-write scopes/operator pin used for
live-portal-flow.py. This is HTTP Portal API validation, not browser flow UI.
Login through Portal, acquire control, discover physical GTK, execute two
window-scoped semantic text steps, exact persisted mixed Unicode/suffix check,
identical repeat request same terminal record confirmed2/no extra edits passed.
Every Portal response asserted no-store. Stop/logout passed. Evidence
live-portal-flow.json. Short actual run (~1sec) does not prove35sec boundary.

Temporary scopes revoked, cleanup login logged out, operator pin and private
credential file removed; disposable account remains no scopes. Zero active
grants/no lease/no pending release verified; managed cleanup restart requested.
Fixture child/temp files removed, X focus restored.

Next interruption/unknown-outcome owner endpoint cases and saved-flow evidence
integration. Browser flow authoring/run UI, portable identity/roles and major
platform/companion/remote/learning intent still incomplete. All31 flags false.


## Owner flow uncertain-outcome regression — 2026-09-05

Previous turn validated physical Portal account flow. Extended existing real
Unix owner/signed helper/SQLite integration fixture to simulate native mutation
followed by lost acknowledgement. First-step uncertainty records incomplete,
confirmed0, and prevents second mutation. Second-step uncertainty preserves
confirmed1 and prevents third mutation. After native recovery, identical
requests preserve disposition/count/digest and produce no further effects.
Targeted owner regression passes under race detector (1.247s); diff check clean.
No production code change required. Native effects are fixture counters; this
does not establish physical interruption, concurrent Stop scheduling, or crash
recovery across owner restart. Existing physical deployments remain unchanged.

Next concurrent cancellation/Stop coverage and saved-flow evidence integration.
Browser flow UI, portable identities/roles, companion/platform/remote/learning
work remains incomplete. All31 phase flags remain false; full goal active.


## Stop interrupts active native work — 2026-09-05

Previous turn proved uncertain outcomes. Found owner p.mu held over helper
operations, with helper repository serialization additionally delaying Stop.
Owner Observe/Applications/Resolve/Act and flow Claim/Finish now release session
map lock before helper RPC. Stop retains owner lock during revocation. Helper
controller tracks native operation contexts and exact-lease pending Stops in
desktop_interrupt.go. Authenticated Stop cancels matching in-flight and newly
registered calls while it acquires the durable repository fence. Stale holder
cannot cancel successor. Observe/Applications/Resolve/Act use registered contexts.
No effect retry or cancellation-as-success introduced. Native code must honor
context; this is not a hard preemption mechanism or final latency acceptance.

Owner integration blocks native observation until cancellation, Stop completes
within one-second request budget, flow exits error with no effects and duplicate
under revoked lease refuses. Controller tests prove stale Stop leaves successor
observation active and matching Stop interrupts it; Stop during mutation retains
durable outcome_unknown and refuses duplicate. Control race3.983s, sessions4.679s,
new Stop regressions1.272s pass. Initial GTK helper shutdown failed SQLITE_BUSY
at semantic_gtk_linux_test.go:152; isolated helper rerun passed2.584s. Contention
observation filed separately; exact failing SQL statement remains unproven.
Diff check clean. Managed Device Control restart built and healthy at09:54:42Z;
no active grants before restart. No temporary operator pin or credentials added.

Next physical Stop latency/interrupt acceptance (50 trials p95<=250ms) and
saved-flow evidence integration. Native noncooperative calls and remote paths
not proven. All31 phase flags false; full goal remains active.


## Physical local Stop latency — 2026-09-05

Previous turn deployed native interruption and owner lock fix. Added bounded
installed-CLI live-cli-stop.py benchmark on physical X11 session2: each of50
trials opens control, captures physical display, Stops, reads helper durable
state and grant publication, then submits valid pointer-move input to current
position under stopped lease and requires permission_denied. No accepted input
performed. Ephemeral pixels/requests removed. All50 passed p95 9.891ms, maximum
18.173ms, including CLI startup. This upper-bounds lease-invalidation latency
for idle-after-completed-capture cohort only. No held input/concurrent mutation,
browser or remote performance claim. Binary SHA256 captured from running API
and helper /proc executables plus installed CLI; host facts stored in JSON.
No active grants or pending release; no credentials/pins added.

Evidence live-cli-stop.json; support-matrix append records narrow result.
Next saved-flow evidence/library integration and browser/companion flow access,
plus remaining loaded/interruption/platform acceptance. All31 flags false.
Full goal active. Prior transient GTK shutdown report knw-1788602079461973346.


## Explicit desktop text outcome assertion — 2026-09-05

Previous turn measured physical Stop. Inspected existing control/library.go:
SaveValidatedFlow requires complete passing run and terminal assertion; current
text-only desktop flows cannot meet promotion contract. Added distinct canonical
AssertTextAction (expected_text,element_id,observation_revision), Action oneof5.
Generated Go/TS/Python device-control and dependent Portal. Signed helper accepts
bounded assertion payload including empty expected text. Semantic backend uses
same lease/opaque cache/geometry/identity/editability checks; validates exact text
and re-reads at Apply, checks current authority, never calls InsertTextChecked.
Receipt uses existing applied completion convention; expected text stays out of
journal. Cache consumed, duplicate command returns prior receipt.

Flow runner now supports desktop-text-assert with window/text arguments; no
position permitted. Same fresh observation+window resolution path as insertion,
distinct assertion action. Existing ExecuteDesktopText public signature retained.
Signed real GTK fixture rejects wrong expectation, confirms correct one, repeats
without mutation and proves expected text absent from SQLite journal. Helper race
2.753s, sessions4.044s, desktop flow1.020s pass; diff clean.

Not yet deployed/rebuilt CLI or SDA-refreshed UI package. Live production remains
previous Stop-capable binaries. Library promotion rules NOT relaxed or extended
yet; owner RunFlow still does not populate service run evidence. Next durable
run/library linkage preserving actor/target/surface/context identity, explicit
assertion promotion, exact saved-version execution, then Portal/CLI live flow.
All31 flags false; broader goal active.


## Durable desktop run provenance — 2026-09-05

Previous turn added desktop assertions. Existing generic library service does
not carry authenticated desktop actor/surface scope; do not put desktop candidates
into its public run maps or expose them through generic GetSavedFlow. Added
flows.DesktopRuns repository and SQLite implementation, schema embedded beside
code in desktop_runs.sql via EnsureSchemas. Terminal records hold owner-derived
actor/device/surface/OS desktop session/source lease/run ID, exact request digest,
candidate Flow, disposition and helper-confirmed prefix. Candidates contain text
arguments intentionally; no pixels or grant tokens. No public getter mounted.

NewWithDB initializes repository. Owner RunFlow requires it, persists terminal
helper result before response, and repairs a missed write on duplicate terminal
claim without reexecution. Claimed result is not persisted as terminal evidence.
Immutable insert+compare prevents uncertainty rewrite; exact scope required by
Get. Caller run IDs cannot collide across actor/source lease.

Owner real Unix/signed helper/SQLite regression reconstructs repository, reads
passed and incomplete evidence, verifies repeated persistence, denies actor and
surface mismatch, and refuses rewriting incomplete as passed. Control race4.050s;
final owner race1.274s after OS-session scope addition. Diff check clean.
Not deployed; live binaries still Stop-capable version before AssertTextAction.

Next authenticated desktop promotion/saved-version replay using this source
evidence and existing terminal-assertion/repair rules. Design library access to
retain actor/target scope; generic library API must not expose desktop candidates.
Then managed builds, installed CLI refresh and physical assertion/saved flow.
All31 phase flags false; full goal active.


## Scoped desktop promotion and saved revisions — 2026-09-05

Previous turn persisted terminal source runs. Extended flows.DesktopRuns with
Promote(source scope,context,id,expected) and GetSaved(scope,id,version). Added
DesktopFlowScope actor/device/surface/context, SavedDesktopFlow including source
run scope/digest and immutable candidate. New table embedded in desktop_runs.sql
keeps desktop candidates outside generic public library. Promote loads stored
evidence, requires passed/all confirmed and valid desktop flow ending in
desktop-text-assert. Source promotion idempotent for same revision intent.
Exact version required for GetSaved; scope mismatch refuses. Repair uses existing
assertion/auth/redaction/transport preservation logic moved to shared
flows.PreserveFlowChecks; generic control wrapper retained. Atomic conditional
insert guards latest expected revision and leaves prior versions unchanged.

Targeted control desktop+generic promotion/property replay race passed1.272s,
covering reconstruction, idempotence, actor/surface/context denial, incomplete
and no-assert refusal, altered assertion and unlock-policy refusal, v2 repair,
stale expected version rejection, v1 preservation. Diff check clean. Tests use
stored source evidence fixture; no live promotion/owner API proof yet.

No deployment in this turn. Next authenticated owner/account RPCs for promotion,
GetSaved and saved-version execution, reusing RunFlow authority/dedup; then
Portal forwarding and CLI/live validation. Context is currently an explicit
comparison label; portable app identity and context verification remain work.
All31 flags false; full plan active.


## Authenticated desktop library APIs — 2026-09-05

Previous turn added scoped revision storage. Added PromoteFlow/GetSavedFlow/
RunSavedFlow canonical methods to DesktopOwnerService and DesktopAccountService,
plus typed request and SavedDesktopFlow messages. Generated DC+Portal bindings.
Helper protocol remains execution-only. owner desktop_library.go derives actor/
device/surface from live admission; promotion requires control/write and exact
source surface, reads require current admission, replay requires control/write.
Promotion/get recheck admission after storage before returning candidate text.
Source session is provenance only. No arbitrary actor accepted.

Saved replay loads exact revision and delegates private runFlow using same
current session/application refs and caller context. Digest envelope includes
saved ID/version/context so inline and saved requests cannot share a run claim.
DesktopRun persists typed optional SavedRevision. RunFlow public behavior keeps
its prior digest. Canonical response contains candidate/source digest but no
helper credentials. Source text remains off generic flow API.

Real Unix owner/signed helper fixture promotes assertion flow, repeats idempotent
promotion, gets exact revision, denies unauthenticated account and wrong context,
replays duplicate with same record/no input mutation, rejects inline collision,
verifies saved revision provenance, and refuses candidate read after Stop.
Native assertion in this test is a fixed fixture; real GTK assertion was proved
in earlier helper test. Control race4.029s and diff check pass.

Not deployed. Portal service forwarding and installed CLI commands still absent.
Next add those surfaces, managed API builds + CLI rebuild, then physical saved
flow validation across CLI and Portal. All31 flags false; full goal active.


## Portal/CLI surfaces and physical saved flow — 2026-09-05

Previous turn added owner library RPCs. Portal DesktopSessionService now exposes
PromoteFlow/GetSavedFlow/RunSavedFlow; forwards account auth only/no-store/no
retry,10sec promotion/get and35sec replay. CLI adds promote-flow/get-saved-flow/
run-saved-flow under explicit owner socket+typed request. Manifests and config
docs updated. Portal surfaces race1.061s, modules1.117s, CLI1.149s pass (first
module command used wrong ./modules directory; corrected ./internal/modules).

Installed CLI rebuilt. Managed Device Control restart healthy10:19:25 deployed
assertions+run provenance+saved library; explicit Portal build and managed
restart healthy10:19:46 deployed latest imported schema/forwarding. No UI saved
flow authoring/control added, no SDA UI package refresh.

Physical live-cli-saved-flow.py succeeded: source two Unicode insertions+exact
terminal assertion, identical run duplicate, promote, retrieve exact revision,
replace GTK with fresh process and rediscover, replay same saved revision with
new app refs, identical replay duplicate, exact persisted text, Stop. SavedID
50be8e30-3873-4afb-9b6b-0ca7886286b2 version1 retained for local principal;
context gtk-unicode:v1. Both recordsconfirmed3. Evidence live-cli-saved-flow.json.
All fixture children/temp pixels/requests cleaned and X focus restored. Zero
grants/no durable lease/no pending release verified. No account pin/creds added.

Next Portal account live saved flow and browser/companion flow UX. Context
compatibility/portable semantic identities, other platforms, remote and learning
acceptance remain incomplete. All31 flags false; full goal active.


## Physical Portal account saved flow — 2026-09-05

Previous turn deployed forwarding and validated CLI saved flow. Added
live-portal-saved-flow.py using account-authenticated Portal HTTP API (not
browser). Temporary disposable account/read-write scopes and owner subject
pin configured via managed restart. Registration login logged out. Portal
login, two insertions+terminal exact text assertion, duplicate source run,
promote/get exact version, restart GTK fresh process/discover, saved replay,
duplicate saved run, exact text, Stop/logout all passed. Every response no-store.
Evidence live-portal-saved-flow.json; saved30469cfd-88b3-406d-8590-8f8eeebda3a2 v1
under temporary account, context gtk-unicode:v1. Both recordsconfirmed3.

Read/write scopes revoked and empty list verified; cleanup login logged out,
operator subject pin/private credentials removed, no active grant/lease/pending
release. Disposable account remains without scopes. Fixture processes/tempfiles
removed, X focus restored. Managed cleanup restart requested.

Important next gap: CLI uses Unix principal; Portal uses account subject, so
these proofs used same flow content but DIFFERENT saved revisions. Plan requires
the same saved revision across surfaces. Add explicit account-authenticated CLI
mode (protected token file, DesktopAccountService namespace, no fallback to local
authority) then prove shared revision. Do not weaken actor isolation to make
existing CLI read another principal's library. Browser/companion flow UI and
portable identity/context/platform/remote/learning work remain. All31 flagsfalse.


## Same saved revision across Portal and CLI — 2026-09-05

Previous turn proved Portal saved flow but identified actor mismatch with CLI.
CLI now accepts optional --access-token-file on every desktop command, reflected
in manifest/docs. File must be regular, not symlink, group/other permission bits
zero, unchanged identity across Lstat/Open/Stat, bounded16KiB, one nonempty token.
Account client sends bearer through interceptor to DesktopAccountService only;
errors redacted retaining status, no local fallback. Without flag local owner
namespace unchanged. Tests verify header/namespace, denial no retry, private
file/symlink refusal, no secret in errors. Initial manual HTTP fixture used JSON
while client negotiated proto; corrected to generated Account handler. CLI
race1.170s passes; installed CLI rebuilt. Diff clean.

Physical live-shared-saved-flow.py: temporary scoped account+owner pin, Portal
executes two insertions+assertion and promotes; account CLI fetches SAME saved
ID14b2a745-135c-4a1d-ae8f-0f14d7c88b91 v1, rejects local-principal inheritance,
restarts GTK/new app refs, CLI replay passesconfirmed3, Portal repeats SAME
replay request and returns exactly equal record/no extra edit. Exact text and
Stop/logout pass. This proves same immutable revision API/CLI, not merely same
flow contents. Credential file inside fixture0700 tempfile directory0600.

Temporary scopes revoked/empty verified, cleanup login logged out, owner pin
and private credential file removed, fixture temp tokens/processes cleaned,
X focus restored, zero grants/no lease/no pending release. Managed cleanup
restart requested. Disposable account remains without scopes. Evidence
live-shared-saved-flow.json.

Next browser flow authoring/promotion/replay UX sharing same Portal services,
SDA refresh of UI proto package as needed, companion presentation identity,
portable identity/context/platform/remote/learning acceptance. All31 flagsfalse.


## Browser desktop flow editor — 2026-09-05

Previous turn proved same saved revision Portal/CLI. Added DesktopFlowPanel.tsx
inside existing DesktopSessionPanel keyed by account email. Draft rows support
text insertion and exact text assertion, window/field prefill from selected
semantic field, explicit character offset, max32steps. Uses canonical FlowSchema
and JSON Struct arguments (TS generated Struct field maps to JsonObject). Run
uses current application binding and unique ID; uncertain response retains
identical request for explicit Check same run outcome, no automatic fresh ID.
Parent manual-input uncertainty is fenced during/after unknown flow; fresh
flow/replay respects parent uncertainty. Current session changes abort pending
call, discard late result and clear pending status. Stop stays in parent toolbar.

Save enabled only complete passing terminal assertion with matching confirmed
count; draft edits clear verified source. Context label explicit. Load exact ID/
version, preview window/field/text, replay immutable saved version using current
application refs. Saved/draft metadata stays in memory across Stop while signed
in; no browser credential or candidate persistence added. Account change unmounts.
English/Japanese/Arabic strings added with parity. No dependency refresh required:
current node_modules canonical client already contains saved methods.

Targeted17 Vitest tests pass:4 new flow,8 existing desktop,5 locale. Typecheck,
ESLint (after async test corrections), strings check, production build pass.
Managed Portal restart requested with owner socket. This is service-fixture UI
validation; live Chromium author/run/save/replay/Stop journey next. Existing
physical API/CLI proof retained. No temporary account/owner binding used here.
Browser persistence across companion presentation and broader platform/remote/
learning intent remain incomplete. All31 flagsfalse; full goal active.


## Live browser flow lifecycle — 2026-09-05

Previous turn deployed DesktopFlowPanel. New live-portal-browser-flow.cjs uses
installed Chromium and physical GTK: login/request control, named application/
window/field selection, editor Add text step (Unicode position2), Add text check
(exact final text), Run draft confirms2, Save verified flow, Load saved revision,
replace GTK with fresh process, Find applications/reselect, Run saved revision
confirms2 and exact persisted text. UI operations use production Portal account
forwarding; no direct helper mutation. Saved c4d94e5e-5444-48f5-8c4d-fa90be79f668
v1 under temporary account/context gtk-browser:v1. Metadata and source/build/API/
helper hashes in live-portal-browser-flow.json. Browser local/session storage
checked for credential/fixture text absence. Stop/signout UI completed; cleanup
also attempted direct Stop/logout with fresh browser login token.

Temporary read/write scopes revoked and empty verified, cleanup login logged
out, owner pin and private credentials removed. No grant/lease/pending release.
Disposable account remains without scopes. Fixture children/work directory and
browser cleaned; outer Python wrapper restored original physical X focus.
Managed Device Control cleanup restart requested. No production code changes
required this turn.

Next companion presentation/conversation/flow-state continuity and remaining
portable app identity/context, OS adapters, remote, learning and acceptance.
Browser author/save/replay core now has physical evidence; this does not prove
all accessibility, interruption, multilingual or package journeys. All31 phase
flagsfalse and full goal active.


## Native presentation packaging primitive — 2026-09-05T11:01:08.776417+00:00

Implemented Scenario-to-Desktop-owned v1 `native_extension` contract in canonical
DesktopConfig (field 13) and PipelineConfig (field 30; 29 already tool_artifact_root),
generated bindings, Go JSON types/validation, Connect pipeline round trip and
GenerateStage. Only `presentation` with exact `window.presentation` permission;
platform declarations must cover all requested targets. No custom entrypoints,
imports or helper binaries accepted. Generator validates before filesystem output.

Generator installs native/presentation.ts alongside its existing main/preload,
serializes config as data, emits native-extension.json (compiled entrypoint
`dist/native/presentation.js`) and includes metadata in electron-builder files.
Vanilla builds expose no presentation bridge/handlers. Built-in bridge get/set
(expanded, palette, pill) resizes one BrowserWindow, restores prior normal bounds
and max state, clamps display removal, validates main window/frame plus initial
origin/file identity, removes IPC/listeners on close. No dependencies added.

Validation: Go generation admission + pipeline native roundtrip + existing config
roundtrip/GenerateStage selection passed; 18 Jest tests including actual generated
file layout, syntax transpilation, metadata packaging and test-file exclusion;
4 mocked-Electron Vitest cases; both TS typechecks, native ESLint, and generator
dist compilation passed. First TS host-map indexing and duplicate proto tag errors
were corrected. No real Electron window was launched in this step.

Next: adopt bridge in Portal with compact layouts that keep chat/flow components
mounted; handle API-host versus companion-host identity, global shortcut conflicts,
pre-focus context capture/restore, tray/background/quit disposition, and physical
presentation acceptance. Current bridge becomes authorized after initial
`did-finish-load`, so startup callers must account for readiness. Existing window
state manager currently observes compact bounds; restart restoration still needs
coordination. No Portal companion config exists yet. Native helper packaging,
signing/update-specific compatibility gates and arbitrary template mutation
protection remain beyond this initial closed module registry. Phase 20/21 and all
other phase completion flags remain false; overall goal remains active.


## Portal companion UI adoption — 2026-09-05T11:10:03.557422+00:00

Previous goal turn classified as progress. Added `features/companion/CompanionPresentation.tsx`
provider/toolbar at AppShell, keeping Router/Outlet and chat/desktop components
mounted across mode changes. Palette uses CSS-hidden chat/desktop tabs; pill hides
content but exposes registered active chat/desktop Stop actions. EN/JA/AR strings.
No native bridge retains normal web layout. Mode changes are serialized; failures
read back state without resending; malformed contract rejected. Provider does not
persist credentials. Real ChatWorkspace test selects alternate branch, types draft,
cycles modes, asserts identical DOM and no extra tree read/send.

DesktopSessionPanel Stop now retains exact active lease after RPC failure so retry
remains possible (new regression test); only successful Stop clears active state.
Shell preload uses a bounded 10s readiness handshake before IPC calls; main sends
ready on each trusted did-finish-load (supports same-document reload), retaining
original origin/file admission. Actual native reload not physically verified yet.

Added Portal `desktop/native-extension.json` and packaging CLI
`--native-extension-file` on pipeline run/start-active configuration. Reader accepts
regular max16KiB JSON and strict canonical proto fields. API still validates module,
permissions and target coverage. Installed CLI has NOT been rebuilt this turn;
source flag and tests work. Portal extension config contains no private Electron
source. Documentation describes usage and acceptance limits.

Evidence: first combined Portal run 25 tests (4 companion,4 chat,9 desktop,6 shell,
2 routes) passed. After adding real branch continuity test: 18 targeted tests
(4 companion,5 chat,9 desktop) passed. Portal TS noEmit, changed-file ESLint, string
sync, Vite production build passed. Native 4 Vitest tests/typecheck/lint passed;
10 generator extension tests and generator compilation passed. CLI targeted native
file and existing typed config tests passed. No dependencies installed.

Next: compile/install current packaging CLI/API if exercising pipeline; generate
and run Portal through the owner lifecycle/evidence path to obtain real Electron
mode/Stop/branch evidence. Implement shortcut conflict diagnostics, tray/background,
focus before activation/restore, quit disposition, restart state coordination and
actual companion-host binding. Compact layout currently scrolls full existing chat
controls and may need physical usability refinement. Remaining phase20 helper and
compatibility/signing/update governance and all other plan phases stay open. No
phase marked complete, no goal status mutation, no temporary accounts/leases created.


## Real companion packaging and launch diagnosis — 2026-09-05T11:25:25.253109+00:00

Previous goal turn = progress. Generated actual Portal wrapper through owner
pipeline, installed via SDA, compiled all generated TS and built unpacked Linux
Electron app. Final generation pipeline `65183fad-ba55-e4ac-57c9-ea2258ac2aa1`
completed. Artifact:
`scenarios/portal/platforms/electron/dist-electron/linux-unpacked/portal-desktop`.
Generated tree is ignored by git; canonical templates/declaration are source.
Fingerprint/evidence: native-companion-package.json (binary + app.asar + hooks).

Integration defects fixed in owners:
- Orchestrator canonicalizes linux to linux-amd64. Go/TS native validators now
  map recognized linux/darwin/windows + amd64/arm64 targets to declared OS
  permissions; unknown architectures remain rejected. Go tests and 11 generator
  tests pass. Initial pipeline b6e88c07... failed truthfully, b14f22c9... succeeded.
- CLI `--proxy-url` now reaches config; GenerateStage validates HTTP(S), excludes
  credentials/literal quotes/backslashes, applies to external ServerPath. Generated
  renderer URL is http://127.0.0.1:24965. Earlier analyzer defaults 20000/15000 were
  wrong for actual Portal; API_ENDPOINT remains fallback but UI uses its own API
  proxy. Native remote-host binding remains unresolved.
- Added existing emitted `runtime_ipc_port_published` to telemetry LaunchEventName;
  full generated tsc then passes (prior transpile-only tests had missed it).
- Linux livedesktop backend now reaps launched processes, uses done channel for
  running state, and returns bounded stderr when child exits in first150ms.
  Real /bin/sh failure + /bin/sleep later-exit race tests pass. This fixed a false
  launched/running report for the crashed app.

Governed dependencies: SDA initially reported electron unrecorded. Approved exact
33.4.11 for Portal-only local native companion validation, dev_tooling policy,
review expiry2026-09-06; MIT existing owner runtime, no production security claim.
SDA installed `npm/electron@33.4.11` on `portal/platforms/electron` with scripts
ignored. Entire generated package dependencies were materialized by that gateway.
Security Health query unavailable at configuredlocalhost2026; release security
review remains open. No raw dependency installer manually executed. Generator's
legacy raw npm install was bypassed via existing SKIP_DESKTOP_DEPENDENCY_INSTALL=1
on every managed restart. Build service also has legacy raw installs; avoided by
using generated tsc and electron-builder as build tools after SDA installation.
These owner dependency paths need governance follow-up in phase20.

Installed scenario-to-desktop CLI auto-rebuilt from source and now includes both
--native-extension-file and --proxy-url. Scenario-to-Desktop API rebuilt and
managed restarted, healthy API19925 UI22829 at11:23:44Z. Keep
SKIP_DESKTOP_DEPENDENCY_INSTALL=1 GOCACHE=/tmp/portal-desktop-go-cache when restarting
until legacy generator install path is addressed.

Native launcher evidence:
- POST /api/v1/livedesktop/sessions {scenario_name:portal,width:1280,height:900,platform:linux}
- POST /sessions/<id>/launch {app_path:<absolute unpacked portal-desktop>}
- Initial session7cbc0be9-f4c5-4a58-94e4-ca142f11e444 blank screenshot capture
  ecd3c4f8-42f6-49ea-8fc2-1696ab0640c3, process exitstatus133. Stopped/deleted owner.
- After diagnostics fix, sessionf5385de8-7f17-4be9-87d3-49d2c7d78c8e returned500:
  FATAL:setuid_sandbox_host.cc(163), unpacked chrome-sandbox not root-owned4755.
  This is proven cause, not a generic launch failure. Session stopped with200.
- Do NOT change host chmod/chown or carry private host remediation. Existing owner
  `launch-electron-validation` uses --no-sandbox on Linux on a managed display,
  scopedloopbackCDP and isolatedprofile. It requires context_id/scenario_name/
  artifact_digest/target_id/journey_id/profile_id/isolation_lease_id. Next step is
  obtain a real provider isolation context (do not invent a lease) and use this
  owner path, or add a properly scoped owner development launch policy. No
  validation identity was fabricated. No active desktop sessions remain from us.

REST live desktop routes are still registered in main.go despite docs migration;
API supports control screenshot,window_geometry,pointer_click,key_press. Validation
launcher returns target CDP endpoint for BAS after unique renderer selection.
`api/target/` is ignored by generic rg; use rg --no-ignore when inspecting it.

No phase/goal completed. Actual companion view transitions, shortcuts/tray/focus,
restart/quit disposition and companionhostbinding remain outstanding.


## Real Electron presentation acceptance slice — 2026-09-05T11:40:06.827663+00:00

Previous turn = progress. Owner validation launch now physically exercised.
Important correction to earlier interpretation: shell isolation_lease_id is a
correlation field for managed desktop/profile. Matrix executor itself assigns
runID:cellID before routed workflow isolation exists; it is not a bearer authority.
Native presentation validation binds it to the actual managed session ID. Evidence
explicitly does NOT claim routed application-data isolation.

New reproducible scripts live-native-session.py (launch or stop),
live-native-presentation.cjs, live-native-keyboard.cjs. Latest evidence
native-companion-live.json retains previous attempts, final cleanup, and hashes.
Latest managed session4cf4a1e6-7957-4968-9200-810ea7566160, renderer
B549588F4FFC3D335F3B15D5FFCA562B. Native geometry palette560x480, pill320x96,
expanded1200x800 restored. Draft and surface selection values AND DOM node identity
survive all transitions. No messages sent, no new account/grant/lease for physical
Desktop2; only managed virtual desktop+Electron profile created. Owner cleanup200.
Pill screenshot capture0bd98abf-b950-47b9-a19a-b5a1796f424b visually inspected.
First screenshot raced paint and showed stale Expanded label; final script waits
two animation frames before native screenshot. Keyboard ArrowDown/End/Home after
initial selector focus all pass without refocusing.

Integration repairs:
- appPID(session) now uses ElectronValidation.PID under session mutex, rather than
  0 for validation launches; owner native window controls select that process.
- Electron lacks Browser.getWindowForTarget; validation uses owner window_geometry.
- Native geometry/control's primary-window selection now accepts named windows
  >=32px; startup/readiness keeps prior100px threshold. Test
  TestCompactWindowGeometryDoesNotWeakenReadiness asserts pill geometry measured
  while readiness stays false. Existing detector tests pass. Do not describe this
  as changing startup readiness to make a test green.
- Portal CompanionToolbar no longer physically disables mode selector while a
  request is pending (which had lost focus). aria-disabled/aria-busy expose pending,
  key guard and existing inFlight prevent concurrent requests; unavailable bridge
  still physically disabled. Focus regression added to4 companion tests. Tests,
  TS noEmit, ESLint, production build all passed; real keyboard rerun passes.

Current Scenario-to-Desktop API19925 UI22829 healthy after managed restart11:33:55Z
with SKIP_DESKTOP_DEPENDENCY_INSTALL=1. Binary includes PID+compact-geometry fixes.
Portal UI dist rebuilt11:37:25Z; current proxy-based Electron loads it live.
Generated native artifact app.asar unchanged from previous turn; new UI is external.
No managed desktop sessions left from this work. Latest renderer/profile removed.

Next: implement actual shortcuts/conflict diagnostics, tray/background, pre-focus
capture/restore, hidden mode, active-task quit disposition; validate active desktop
flow transitions and APIhost!=companionhost. Expanded currentlynormalPortalUI,
pill still includes native title/menu chrome (not final polish). Window state
manager may persist compact bounds on quit: coordinate expanded state persistence
before claiming restart acceptance. Portal chat has no full attachment/task
presentation store yet beyond mounted state. Everyphase flag remainsfalse.

## 2026-09-05T12:04Z — Native global activation version 2

Version2 presentation declaration now requires window.presentation + global-shortcut
and a bounded accelerator. Version1 remains manual only. Go generation/pipeline,
Connect/proto, CLI declaration, TS generator and runtime carry activation_shortcut.
Portal declaration uses CommandOrControl+Shift+Space. Native snapshots include
revision + shortcut registration status; subscription events synchronize native
activation into Portal, ignoring late older IPC responses. Registration refusal
reports unavailable without disabling manual controls; only successful owned
registration unregisters on window close. Activation restores minimized window,
opens palette and focuses same renderer (fullscreen focuses without resizing).
Compact menu bar hidden, restored with expanded original visibility.

Targeted Go generation/pipeline, 12 generator Jest, 7 native Vitest, 6 Portal
companion Vitest, TypeScript/lint and UI/generator builds passed. Generated packaged
tsc and electron-builder --dir --linux passed after SDA exact Electron33.4.11 install.
Generation pipeline629a4857-969f-baef-45c8-74fb3d63ba5b. API/CLI rebuilt and managed
S2D restart with SKIP_DESKTOP_DEPENDENCY_INSTALL=1. SessionView includes display_id
(owner identity; test passes) for scoped native validation without guessing display.

Real managed native session b05abf59-e77a-4815-ae04-4c7f93157131:
WM_STATE Iconic confirmed with xprop after exact process-owned windowminimize;
owner key_press ctrl+shift+space restored palette560x480, revision0->1,
registered shortcut, same renderer and draft node/value. Existing presentation
and keyboard checks rerun passed. Owner cleanup200. Geometry alone cannot prove
minimization; test now checks WM_STATE explicitly (AltF9 didn't minimize this WM).
Conflict session2664c6a2-6f3a-45ac-a3c3-3c1056d05154:
ctypes XGrabKey fixture reserved Control+Shift+Space on owner-created :99 during
Electron launch, then released its connection. Native status unavailable and
visible Portal warning confirmed; manual palette560x480 preserved draft/renderer.
Owner cleanup200. No physical desktop grants or accounts created. Latest evidence
native-companion-live.json retains prior normal session and previous attempts.
Reproducible fixture native-shortcut-grab.py and live-native-shortcut.cjs. Package
app.asar + UI fingerprints recorded under fingerprints_v2.

Next: pre-focus capture/restoration, hidden mode, tray/background lifetime,
active-task quit disposition, expanded state persistence across restart; inspect
phase21 intent before extending native contract. Global activation currently
focuses without capturing prior application context. No acceptance claim for
macOS/Windows, active desktop flows, APIhost!=companionhost, attachment store or
complete plan phases. All phase flags remain false; goal remains active.

## 2026-09-05T12:10Z — Compact close preserves expanded geometry

Native installPresentation now returns PresentationController.persistenceState.
Returns null while expanded (normal live state capture); compact views provide
last expanded bounds, maximized intent and false fullscreen, clamped to current
work area. WindowStateManager.manage accepts optional persistenceState provider
and uses it during capture. Main installs presentation before managing and passes
provider. Same existing storage remains authoritative; no second geometry file.
This fixes pill/palette dimensions being saved as next expanded startup geometry.

32 native/window-state Vitest tests pass including fresh manager restart from
both compact modes, later moved/resized expanded state, maximized intent and
removed monitor clamping. tsc -p tsconfig.dev.json and generated full tsc pass.
Default template tsconfig targets generated src and has no inputs in template
root; use dev config. Targeted ESLint hits preexisting logger line44
no-base-to-string, filed knw-1788610150011572245; no logger change made.

Generation pipeline32404ad9-a09b-5fb8-fdc9-0ba1d55e5a2a completed. SDA exact
Electron33.4.11 reinstall then generated tsc/electron-builder --dir --linux passed.
Live managed sessionc941bf01-3b38-48fd-af98-dbc04cfea640: real Ctrl+Q native quit
from pill saved1200x800 expanded geometry in owner-created validation profile.
Window-state x40y50 corresponds content origin (native geometry frame41y72).
Evidence live-native-close.cjs/native-companion-live.json with latest app.asar
hash. Reads exact process cmdline owner profile; Electron process title combines
args into a single argv string, so extraction uses bounded /tmp/vrooli-electron-
validation-* regex. Initial argv-array assumption failed and was corrected.
Owner cleanup200; no native session left. No physical account/grant creation.

Full native process restart is still pending: managed validation launch normally
cleans its previous profile. Do not claim unit manager reconstruction is physical
application relaunch. Other pending phase21 behavior remains pre-focus capture /
restore, hidden/tray/background, explicit task quit disposition, active-task
transitions, and remote APIhost/companionhost binding. Goal and phase flags active.

## 2026-09-05T12:15Z — Real native process restart verified

Added target.ElectronSession.Restart: only exited, not closed/superseded session;
retains original profile and launch options, chooses fresh CDP port and renderer,
transfers profile cleanup ownership only after successful startup. Lifecycle
mutex serializes restart/Close; failed restart preserves profile for retry/cleanup.
LiveDesktop service and POST /sessions/{id}/restart-electron-validation expose
this operation without accepting a user-chosen profile path. HTTP409 on refusal.
API docs updated. Target files reside under ignored api/target (use --no-ignore).
Tests prove running refusal, failed-start profile retention, successful transfer,
old Close cannot delete active profile, superseded refusal, final cleanup. Targeted
Go target/livedesktop tests and restart race test pass. API rebuilt and managed
restart completed12:13:49Z, healthy19925/22829.

Live managed session45f9aa92-4db3-4474-800c-0ef331e77448:
native Ctrl+Q from pill saved1200x800, restart endpoint launched NEW process2960965
and renderer6958FB5D7669DFB29A3822BB098337C2 in SAME owner profile
vrooli-electron-validation-2497797. UI confirmed expanded, native bounds1200x800.
Final DELETE cleanup200 and actual /tmp profile absence confirmed. No native
sessions or physical account/grant changes left. New reproducible evidence script
live-native-restart.cjs; close script now records profile basename for identity
comparison. native-companion-live.json stores results and source/package hashes.
This closes the geometry full-process restart gap from last turn. It does not
claim conversation/attachments/task persistence across process restart, or
restart after abrupt termination. Phase21 remains incomplete; next substantive
work is hidden/tray/background lifetime and pre-focus context/restore plus
explicit running-task quit behavior and APIhost/companionhost acceptance.

## 2026-09-05T12:19Z — Hidden presentation with native recovery

Native mode set now includes hidden, admitted ONLY while global shortcut is
registered (v1/no shortcut and refused registration cannot hide). Snapshot adds
canHide; Portal's optional Hide entry requires canHide=true, so older native
builds do not expose an unsupported action. Hidden retains renderer and expanded
geometry, window.hide then authoritative revision event; shortcut restores palette
and same renderer. Preload type and Portal mode validation updated. EN/JA/AR label
added via locale sources and generated strings. No separate conversation store.

12 native Vitest tests and6 Portal tests pass, including hidden recovery/refusal,
draft DOM and mock active Stop registry continuity. Native dev TypeScript/lint,
Portal TypeScript/lint/Vite, generated full TypeScript and Linux package build pass.
Generation pipeline86a9286d-e261-dd5a-6aca-abf776e92fad; SDA exact Electron33.4.11
installation used. API unchanged this turn; UI dist and packaged app.asar updated.

Real managed sessiona10948a5-aa60-476f-9943-831cca62bf9e / renderer
92875FD2428C698F70E2F285AD4050B5: hidden xwininfo Map State IsUnMapped confirmed,
WM_STATE removed, native shortcut restores560x480 palette, revision8->10,
same draft DOM/value and same renderer. Keyboard selection passes ArrowDown to
palette, ArrowDown to pill, Home to expanded. Existing keyboard fixture updated
because End now chooses Hide (new last option); no assertion weakened. Hidden
fixture is live-native-shortcut.cjs --hidden. Evidence+hashes in
native-companion-live.json, cleanup200, no native session left.

Remaining: tray/menu actions and configurable background/close behavior;
pre-focus context capture and focus restore; task quit disposition; real active
flows across hidden modes; APIhost!=companionhost and other OS acceptance.
Hidden currently preserves mounted state but does not claim unthrottled renderer
background execution, task persistence after process exit, or tray recovery when
shortcut is unavailable. Goal and all phase flags remain active/incomplete.

## 2026-09-05T12:23Z — Shared reveal for tray and native activation

PresentationController now exposes reveal(expanded|palette|pill), guarded by the
same trusted document check. Restores minimized window, updates authoritative
mode/geometry, shows/focuses same renderer, publishes snapshot. Global shortcut
calls reveal(palette). Main retains presentationController and clears on closed.
Existing tray attempted for presentation extension even if generic systemTray
feature false; requires existing app icon. Show=>expanded, explicit Palette/Pill/
Expanded actions, doubleclick=>palette. Native app activate and second-instance
activation now reveal expanded (previously hidden window could stay hidden).
Vanilla fallback unchanged except second-instance now calls show before focus.
No new arbitrary native renderer IPC. Hidden still requires registered shortcut;
tray object creation is not proof of actual desktop tray availability.

13 native Vitest and12 generator Jest pass. Native dev tsc/lint and full generated
tsc --noEmit pass. Final generatepipelinea6fc89d6-0fad-59c4-b7a6-742fb388634d complete;
earlier864e7318 omitted later activation hook edits and is superseded. Generated
src current, BUT dist/app.asar NOT rebuilt this turn and still previous hidden-mode
artifact. No dependency install performed this turn; reused existing installed
compiler for noEmit. No native live session created; previous cleanup intact.

Next: finish configurable background/close semantics and explicit active-task quit
handling, then rebuild packaged artifact and validate actual tray interaction on
a desktop with notification area support. Physical tray/dock acceptance remains
unverified; do not represent controller unit test as actual tray interaction.
Other phase21 gaps remain pre-focus context/restore, actual active-task transitions,
remote host binding and cross-platform acceptance. All phase flags remain false.

## 2026-09-05T12:29Z — Configurable session background close and explicit Quit

Native PresentationController adds setBackground(bool):bool and prepareQuit().
Background defaultsfalse, enabling refuses without registered shortcut. Configured
window close prevents destruction and applies hidden; prepareQuit called from
main before-quit bypasses background. Normal close restored when preference off.
Fullscreen can hide without resizing; capture retains fullscreen, recovery shows
expanded fullscreen. Hidden maximized->compact unmaximizes before resize.

Main tray AND Window menu use backgroundMenuOption, shared checked state with
both menu instances synchronized. Label explicitly says '(this session)'; native
app-menu accelerator CmdOrCtrl+Shift+B. No preference persistence yet. Menu
available independently of tray because managed Openbox has no notification panel
(stalonetray/trayer/xfce4-panel/tint2 absent); no installation attempted. Physical
tray clicks remain unverified; actual app-menu accelerator checked below.

38 native/window-state Vitest +12 generator Jest pass; native dev tsc/lint,
generated tsc and Linux package build pass. Final pipeline8cdfdb96-3b3a-b3dd-
5ee3-4e12ec0382af supersedes207c960a before Window menu edit. SDA exact33.4.11
installation before package build. app.asar now contains current tray/background
code, unlike last turn's stale artifact. No API restart/change this turn.

Live session1514440b-0fc9-4ee2-8067-e6c8c10e7bbf:
Ctrl+Shift+B enables preference via real application menu; Ctrl+W native window
close yields X11 IsUnMapped, same renderer1F42CC95A09D6C808B4D361860F7EE14 survives.
Global shortcut restores palette560x480 with same draft node/value. Then Ctrl+Q
explicitly exits while background remains enabled, saves1200x800. Managed restart
reuses profile with new process3031739 / renderer7AEF1963905346B0D737A21234DA8542,
expanded1200x800 confirmed. Cleanup200 and profile actual absence confirmed.
No native sessions or physical grants/accounts remain. Evidence current
native-companion-live.json; live-native-shortcut.cjs --background reproduces.

Remaining phase21: persistent preference if desired, physical tray behavior,
pre-focus context capture/restore, real active-task transitions and explicit task
quit disposition, APIhost!=companionhost, platform acceptance. Background preserves
renderer but does not claim unthrottled background timers or active-task durability.
Goal remains active, all phase flags false.

## 2026-09-05T12:34Z — Chat admission Stop gap repaired before quit orchestration

Inspection for active-task quit found ChatWorkspace marked streaming/registers
companion Stop only AFTER await sendPortalMessage and await refreshTree. During
those requests Stop/quit registry missed the task; generation could start later.
Now AbortController and active streaming state set before admission; synchronous
abortRef guard prevents duplicate admission. SendPortalMessageInput.signal passes
to Connect call options. throwIfAborted after send reply and after tree refresh
prevents later generation even if underlying pending call resolves after Stop.
Abort-related transport errors do not surface as generic task errors.

Regression tests: companion toolbar Stop available DURING delayed send; exact
signal aborted; late successful message reply cannot start generation. Separate
pending tree refresh Stop likewise never starts generation. API wrapper test
asserts transport signal forwarding. 16 ChatWorkspace/chat API/companion tests
pass; TypeScript, targeted ESLint and Vite production build pass. New UI dist is
served live by existing packaged proxy Electron. No API/package change, native
session/account/grant creation this turn.

Quit disposition remains NOT IMPLEMENTED. Next follow backend task cancellation
semantics before claiming Stop-and-quit receipt: UI chat Stop is transport abort;
handlers/message/handler.go streamAgent passes ctx to agentchat.Stream, and
internal/integrations/agentmanager/events.go exits on ctx.Done. Initial rg found
no Stop/Cancel call in agentchat or agentmanager integration. Need inspect full
service and add durable run cancellation/reconciliation as appropriate; stream
observation ending is not proof remote agent stopped. DesktopSessionPanel.Stop
retains active exact lease on uncertain reply, but stop() catches error and returns
normally, so awaiting callback alone cannot authorize quit. Registry must wait
for actual resolved task state / authoritative receipts. Goal all phases active.

## 2026-09-05T12:37Z — Agent stream cancellation and run isolation

Portal internal/integrations/agentmanager/events.go now uses context.AfterFunc
closing the owned WebSocket on ctx cancellation, interrupting blocked ReadMessage
without waiting for next frame. Cancellation returns ctx.Err for read/write errors.
Stream rejects empty runID before dialing. Decoder now filters progress/status
payload run IDs as well as run_event; foreign terminal events cannot end selected
run. Missing identity frames discarded when selectedrun specified; outerenvelope
runID used as fallback and propagated into mapped events. Unknown unscoped frames
also ignored. No remote run stop action is implied by stream closure.

Race tests pass for integrations/agentmanager + agentchat. Real httptestWebSocket
silent peer cancellation test verifies immediate return and peerconnectionclose.
Foreign/unscoped status/progress/event cases tested. API built portal-api then
managed restart with PORTAL_DESKTOP_OWNER_SOCKET; healthy17476/24965 at12:36:44Z.
No agent run, native desktop, physical account/grant created this turn.

Next: explicit remote Stop API and durable Portal-owned chat->run association,
then readback/uncertainty through UI before quit authorization. Current
agentchat/service.go only stores runID in transient Stream local; AgentManager
interface only Start/StreamRunEvents. UI abort currently only ends observation.
Agent-manager owner endpoint handlers/runs_create.go:667 StopRun validates UUID,
checks denyRunInitiatedLifecycleOperation, calls svc.StopRun then GetRun, returns
StopRunResponse{status:stopped,run}. Inspect ownership/auth and terminal status
semantics before integrating. Do NOT equate stream return with stopped run or
blindly stop on arbitrary network disconnect. Goal all phase flags remainfalse.

## 2026-09-05T12:41Z — Exact-run Stop and independent readback client

Added APIClient/Service/HTTPClient Stop(ctx,runID) and Run(ctx,runID) to Portal
agentmanager integration. RunState{RunID,Status,Terminal}. Canonical nonnil UUID
validated before network; response identity must match requested ID. Stop uses
POST /api/v1/runs/{id}/stop once and requires owner-reported terminal state,
otherwise ErrStopUnconfirmed (including transport/decode/refusal failures).
Outer response status:'stopped' cannot override running run state. Completed or
cancelled are terminal with actual status preserved; not conflated with one
another. Run uses GET with no body for independent reconciliation, never repeats
Stop implicitly. Existing post helper factors through request(method,...).

Read owner implementation: handlers/runs_create.go StopRun calls owner svc.StopRun
then GetRun and returns protobuf Run. Orchestrator/run_creation.go StopRun selects
interactive/parked/terminator paths and transitions cancelled after handling.
GET GetRunResponse{Run} confirmed handlers/runs_create.go:440. Owner authorization
includes denyRunInitiatedLifecycleOperation; no bypass added.

Race tests pass integrations/agentmanager + agentchat; API build passes. New
service_test.go covers matching cancelled/completed, still-running despite claimed
stopped, wrong/missing identity, malformed/refused responses, invalid identity,
partial lost Stop reply then terminal GET with exactlyonePOST. NO live agent run
created/stopped. API binary rebuilt but not restarted this turn (current running
API from12:36 still events fix; new unused client only on disk).

Next: durable Portal chat/message->run association and explicit user Stop RPC,
including start-response uncertainty/idempotence and readback. UI must keep task
pending until owner outcome resolved; current abort only ends stream observation.
Do not expose arbitrary run IDs as Stop authority. Internal client is not yet
wired to agentchat or UI, and no Stop-and-quit acceptance claimed. Goal/allphases
remain active/incomplete.

## 2026-09-05T12:45Z — Durable chat-message agent admission / binding

New agentchat Repository{Reserve,Bind,Get}, Binding{ID,ChatID,MessageID,TaskID,RunID},
SQLite implementation with context-routed SQLExecutor seam, and domain Schema
CREATE agent_chat_runs. Unique(chat_id,message_id), unique run_id, runnullable
until bound. Immutable/repeatable same binding update; different binding refused.
Message handler Schema returns agentchat.Schema, module wires repository using
RoutedDB. No engine exposed to agentchat service, no foreign-domain schema edits.

Stream validates chat/prompt then reserves BEFORE Start. Existing reservation
returns ErrAlreadyAdmitted and no second remote Start, whether bound or unknown.
After successful Start, Bind uses5second context.WithoutCancel(ctx) retaining
routing values even if renderer canceled during response. Bind completes before
initial started event; exact owner run/task recorded when emit subsequentlyfails.
Missing Runs dependency fails unavailable (all production/test wiring updated).

Race tests integrations/agentmanager + agentchat pass; handler compiles, API build
passes. Tests canceled renderer during Start, persisted exactidentity, reconstructed
repository/service recovery, repeat stream no second Start, conflicting rebind,
wrongchat lookup and unbound reservationnoautomaticrelaunch. Managed Portal restart
applied schema healthy17476/24965 at12:44:06Z; new Stop client from previousturn also
now loaded. No live agent started/stopped and no native/physicalsessions created.

Remaining critical work: unbound admission reconciliation and owner idempotent
launch correlation. Existing HTTPcreateRun tag is portal-ChatID (not admission ID);
inspect owner tag semantics before reusing across messages. Consider binding.ID as
stable admission correlation, exact owner lookup on lost response; no blind retry.
API/UI currently only sees generic already-admitted error; explicit durable run
lookup/Stop RPC and uncertainty UI still needed. Do not expose arbitraryremoteID
as Stop authority; service must resolve binding by owned chat/message. Quit
orchestration not implemented. Allphaseflags false and goalactive.

## 2026-09-05T12:50Z — Admission reconciliation and bound service Stop

StartInput.AdmissionID now mandatory canonicalUUID in HTTP client, supplied by
agentchat durable reservation.ID. Run tag now portal-admission-<UUID>, replacing
portal-<ChatID> which couldn't distinguish messages. FindAdmission lists GET
/api/v1/runs?tag_prefix=<tag>&limit=2; requires exactlyone result, total1,
has_morefalse, exacttag (notprefix), canonical task+run UUID. Owner by-tag endpoint
returnsfirstmatch; deliberately use list to reject ambiguity. Inspected owner
ListRuns: total=len(returned), has_more=len==limit, so one withlimit2 proves complete
filtered result. Missing/unavailable/ambiguous remains unresolved, noautoStart.

Agentchat.Resolve(ctx,StreamInput) validates chat, reads chat/message binding,
reconciles only when rununbound, binds exact existing remote IDs. Service Run and
Stop accept Portal chat/message IDs, notremote IDs. Stop reads current run first;
alreadyterminal returns state, otherwise calls exactrun Stop once. Further Stop
invocations are explicit calls (no automatic retry); UI uncertain flow still must
use Run/readback. No RPC/proto/UI wiring yet.

Tests cover unique/missing/ambiguous/incomplete/prefixonly lookup; reconstructed
unboundassociation recovers withoutStart, binds once, Stop wrongmessage cannot
reachremote. Initialstart test proves persistedadmissionID passed toStart. Race
tests agentchat/integrationpass; APIbuild; managedPortalrestart healthy17476/24965
at12:49:52Z. No agent run/native session/physicalgrantcreatedorstopped.

Next: expose typed MessageService Run/Stop lookup for owned chat+message and UI
showing actual remoteagent run identity/uncertainty. Current UI abort still stops
observationonly. Need persist taskpending untilterminal owner state, distinguish
Stop and 'check status' afteruncertainreply, then implementnativequitdisposition.
Unbound no-match reservation deliberatelycannotrelaunch; userresolutionforfailed
admission (versusunknown) remainsfuturework. Older pre-admission-tag unknownrows
notauto-reconciled. Allphaseflags false; fullgoalactive.

## 2026-09-05T12:57Z — Typed run RPCs, CLI commands and UI API wrappers

MessageService adds GetAgentRun/StopAgentRun(AgentRunRequest{chat_id,message_id})
=>AgentRunResponse{run_id,status,terminal}; no remoteID accepted asauthority.
Proto Go/TS regenerated using makepackages/protogen-code SCENARIOportal.
Handler validates nonemptyidentity, delegates bound agentchat.Run/Stop,
missingbinding=>NotFound, uncertainStop/ownerUnavailable=>Unavailable, neverfalse
successfulterminalreply. Endpointdescriptors added, makeportalendpoints regenerated
.vrooli/endpoints.json and validatesCLIbindings.

CLI messages agent-run <chat-id> <message-id> and stop-agent-run sameargs added;
statusreadeligible, Stop write notruneligible. Thin generatedConnectclients render
ownerstate and recommend readbackaftererror. Installed through cli-coreinstaller
force freshness. Live read missing-chat missing-message returned expectedNotFound
from runningAPI, no remoteeffect. UI exportedgetPortalAgentRun/stopPortalAgentRun
wrappers only; ChatWorkspace does NOT yet callthem.

Real httptestConnect handler with isolatedSQLite tests unrelatedmessage=>NotFound,
emptyidentity=>InvalidArgument, lostStopreply=>Unavailable, subsequentreadterminal,
exactlyoneStop. Targetedrace handler/agentchat/integrations pass; CLIpackages pass;
4 UI APIwrapper tests pass; TSnoEmit and targetedESLint pass. InitialVitest matcher
toHaveBeenCalledExactlyOnceWith unsupportedv2 fixed tocount+args assertions.

GeneratedTS installedfiledependency wasstale afterprotogen (newmethodsmissing).
Refreshed via SDA approved deps install npm/@vrooli/proto-types@file:../../../
packages/proto/gen/typescript --scenarioportal --surfaceui --apply. No rawpackage
managerinvoked; pnpm-lockupdated. API rebuilt andmanagedrestart healthy17476/24965
at12:55:11Z. UI dist notrebuilt (newwrappersunused); nativeartifact unchanged.
No liveagentcreated/stopped, no nativeorphysicaldesktop sessions/grants.

Next: ChatWorkspace agentrun identity andpending lifecycle must use ownedsource
messageID; Stopagent RPC distinct fromabortstream. Lostreply retainspending and
shows Checkstatus (noautomaticStopretry), terminalread clearsregistry. Handle
normalagentcompletionwithownerstate verification andrendererreload/reconnect.
Then nativequitrequest/decision protocol usesactualtaskstates. Allphaseflagsfalse,
goalactive; no fullStop-and-quit acceptanceclaimed.

## 2026-09-05T13:03Z — Agent task UI uses owner Stop/readback

New useAgentTask hook retains {chatId,messageId} in stable ref + state. begin before
StreamCompletion in agent mode, after message admission/tree refresh cancellation
checks. inFlight serializes Stop/inspect, unknown Stop reply setsuncertain and
retains task. stop() then selectsGET whileuncertain (noautomaticrepeatPOST); successful
nonterminal read keeps task and permits later explicitStop; terminalowner reply
clears task. Missingrunidentity/status staysuncertain. inspectcalled afteractivity
streamends; streamenditself doesnotcleartask. Newmessageguard preventsreplacement
whileunresolved. No arbitraryremoteIDcapturedfromrenderer events.

ChatWorkspace handleStop: pendingagent=>boundStop/readback and abortlocalstream
onlywhenownerterminalconfirmed; pendingLLM/admission=>localabortexistingpath.
Companion Stop registry includesagenttask evenifstreamingfalse. useCompanionStop
optionalchecking flag changeslabeltoCheckstatus and retainscallback acrossviews.
Workspace buttondisabledwhileRPCpending; nativecompactcallback guard prevents
concurrentrequests. EN/JA/AR localizedcheckstatus/uncertaintynotice generated.

17 hook/ChatWorkspace/companion tests pass. ActualChatWorkspace fixture runsagent
stream, losesStopreply, transitionstopill, companionCheckstatus receives terminal,
onlyoneStopRPCwithoriginalchat/message IDs. Hooktests disconnectedstillrunningtask,
replacementrefusal, doubleStopserialization, missingownerstate. TS+ESLint+Vite
buildpass. ESLint controlflownarrowingthought agentCurrent.current null in finally;
callinspectunconditionally (hookreturnswithoutnetworkwhennoagenttask). Chattests
rerunafterthisadjustmentpass. UIdist nowcurrent andservedlive. NoAPI/nativeartifact
change, no liveagent/native/physicalsessioncreatedorstopped.

Remaining: reload/remount agenttask recovery (currenthookonlymountedstate), unknown
start/noadmission UIresolution, remote agent completion transcript persistence after
streamdisconnect, and nativequitrequest/decision protocol. NativeQuit still exits;
warningcurrentlyaskscheckstatusbutdoesnotenforcequitdecision. Next implementnative
quit handshake basedontaskregistry with cancel/keepbackground/stopandquit choices,
waitingforactualtaskresolution; do notawaitdesktopstopcallbackalone becauseitcatches
uncertainerrors. NofullphasecompletionorphysicalagentStopacceptanceclaimed.


## 2026-09-05T13:30Z — Native task-aware Quit handshake

Native presentation controller now guards Quit after the trusted preload opts in.
Each pending request has a monotonic token. Matching trusted decisions accept
cancel/background/quit; stale and untrusted decisions cannot exit. Repeated Quit
republishes the pending token, and registration returns it after renderer reload.
Main before-quit checks the guard before shutdown side effects. Non-background
window close uses the same guard; registered-shortcut background close keeps the
existing hide behavior. Apps without an opted-in renderer retain ordinary Quit.

Portal keeps task refs synchronously and offers Cancel, Keep running in background,
and Stop tasks and quit. Callback completion does not authorize exit: actual task
registry removal does. Agent lost-Stop response leaves Check status visible; terminal
owner readback clears the task and permits native Quit. Dialog focuses Cancel,
traps Tab among enabled buttons, supports Escape, and restores previous focus.
English/Japanese/Arabic strings and desktop-integration documentation updated.

Validation: 18 native tests, native TS/lint, Portal TS/lint/Vite build passed during
implementation. Latest companion tests: 10 pass, including keyboard focus/Escape.
Generated pipeline a51e5195-9ea1-0651-0083-24515f915978 completed; approved Electron
33.4.11 refreshed through SDA; generated TypeScript and Linux unpacked build passed.
app.asar SHA256: ea7eb617a94ee2b6448fd68a729ef68dea0b5b72ff322cc101ed9e1bd31765c3.

Real Electron session 4fa5ccdf-7102-4abc-ac31-7404b80e29ad passed native Ctrl+Q
Cancel, background choice, global-shortcut recovery, uncertain Stop retaining the
dialog/process, and terminal readback allowing exit. Task RPCs in this successful
attempt were replaced in the document, with an interception preflight before send.
Exactly one simulated Stop and two simulated owner reads. Terminal read response
was held until counters were captured, then released; native disconnected afterward.
See native-quit-fetch-live.json. Owner DELETE returned200; session stopped.
This is shell/UI handshake evidence, not physical cancellation of a real task.

VALIDATION INCIDENT: initial page.route/CDP Fetch interception did not intercept
Electron requests. Those invalid attempts submitted five Controlled quit fixture
messages to the existing Phase 9 CLI smoke chat; two had real agent admissions.
The two exact message bindings were read, Stop invoked, and independent readback
confirmed terminal failed for e3c6ebf2-74ee-4492-b6a0-4397145d7c1f and
b47df80c-0ef7-44d3-bade-270938c9c02d. The other three had no agent admission.
See quit-fixture-interception-incident.json. Messages retained as evidence. Do not
claim no real agent was affected by the whole turn. Root cause of bypass is not
established; native request hooks are a hypothesis. The old script now delegates
to the fetch fixture; it requires fixture ListChats and sentinel fetch success
before any submission. The separate successful script uses native-quit-fetch-live.json;
live-native-quit.cjs copies the current managed-session record before invoking it.

Still missing: task recovery after full renderer reload/remount (current agent hook
and task registry are mounted state), pre-focus capture/restore, physical tray
acceptance, real active-agent/desktop Quit checks, cross-host binding and remaining
plan phases. Pending native token recovery alone does not establish task recovery;
an empty registry after reload is not a complete safe-Quit implementation. All
phase-completion flags remain false; goal remains active.


## 2026-09-05T13:39Z — Durable admission enumeration for renderer recovery

Added Repository.List and Service.ListAdmissions, with a bounded page size
(default50, maximum100) and insertion-sequence cursor. SQL returns unbound as well
as bound admissions; no agent owner operation is invoked. Unknown launch outcomes
are not filtered or treated as completed. Cursor validates canonical positive int64;
invalid token/size fails before query. Uses existing routed SQL executor; no schema
change or new engine exposure. Pages are a traversal, not an atomic snapshot; cursor
is an ephemeral continuation, not a durable task identity.

New typed MessageService.ListAgentAdmissions request/response returns only Portal
chat_id/message_id plus opaque next_page_token. Terminal status still requires the
existing bound GetAgentRun operation. Handler maps invalid pages to InvalidArgument.
Go/TS generation, descriptor/manifests and Portal endpoint catalog regenerated.
CLI `portal messages agent-admissions [--page-token ...]` lists50 at a time with a
continuation hint; read governance/run eligible. UI listPortalAgentAdmissions wrapper
exists but is not yet used by ChatWorkspace. Local generated proto-types dependency
refreshed via SDA.

Targeted agentchat/handler race tests pass. Test reconstructs repository/service,
retains unknown admissions, traverses pages with a new insertion, rejects invalid
cursors/sizes, and asserts no Start/Find calls. Real HTTP Connect test lists bound
and unbound pages without Stop, validates invalid limit. CLI package tests pass;
five API wrapper tests, UI TS and targeted lint pass. API rebuilt; managed Portal
restart healthy17476/24965 at13:37:32Z and installed current CLI. Live CLI lists the
two existing durable admissions from the prior fixture incident, proving enumeration
across backend restart: agent-admissions-live.json. This turn performed no launch,
Stop, physical desktop mutation or native session creation. Native package unchanged;
UI dist not rebuilt because the new wrapper is not yet consumed.

NEXT: move agent task ownership above routed ChatWorkspace into an app-shell
provider so route changes cannot unregister remote tasks. Provider begins in
recovery-pending state (native Quit must not approve during enumeration), lists all
pages, retains each admission until owner terminal read, and exposes bounded retry
when enumeration/readback fails. Reconcile via GET only, never automatic Start/Stop.
Support multiple recovered tasks and per-task serialized Stop/readback. Existing
useAgentTask currently owns one mounted task and remains unchanged this turn; full
renderer reload is still unsafe/incomplete. ChatWorkspace admission guard should
respect recovery status before Send/Start. Add provider/remount/page-failure tests
and real native reload fixture with proven fetch interception. Desktop-task recovery
needs its own owner reconciliation; do not claim all-task Quit safety from agents
alone. All31 phase flags remain false and full goal remains active.


## 2026-09-05T13:54Z — App-shell agent recovery and native reload acceptance

Replaced mounted single-task hook ownership with AgentTasksProvider in AppShell
(inside CompanionPresentation, outside routed Outlet). Store uses synchronous
immutable snapshots and per-identity busy serialization. Multiple recovered tasks
register independently in companion registry. ChatWorkspace observes the first
unresolved task and refuses replacement while any task/recovery remains pending.
A new recovery registry entry exists from initial render until all admission pages
and bounded owner reads finish. Failed enumeration stays registered with Check
status retry; partial-page identities are retained. Repeated cursor rejects traversal
rather than looping. Owner reads run at most four concurrently. Only terminal owner
responses remove tasks; missing identity/failed reads remain uncertain. A recovered
or uncertain task selects GET before any further explicit Stop.

Route unmount aborts only the local observation stream, retaining remote ownership
in shell. Terminal owner removal also aborts the matching active local agent stream,
so it cannot keep Quit pending. New Send disabled during recovery; localized pending,
failure and retry UI added EN/JA/AR. Existing unknown-launch resolution remains
read-only; no automatic admission or repeated Stop.

Test caught a provider child-order bug: inserting registrations ahead of unkeyed
route children remounted ChatWorkspace and aborted activity. Corrected provider to
keep route children in a stable first slot and keyed registrations in a separate
array. Existing live-stream/lost-Stop test now passes. Shell accessibility test found
SurfaceCatalog nested complementary aside; changed to a labeled section and axe
passes. Existing in-scope accessibility requirement, no suppression.

Checks: seven task/provider tests (pagination, unknown state, failed enumeration
holding Quit, retry-only reads, partial-page preservation/repeated-cursor rejection,
route remount, reconstruction, serialization); eight ChatWorkspace tests; ten
CompanionPresentation tests; six AppShell tests; two route tests; shell axe. TS,
targeted lint and Vite build pass. UI dist current (App-UPniNIiC.js); native artifact
and backend unchanged this turn. Guide updated agent recovery and remaining desktop
lease gap. Old native quit fixture updated for the new read-only admission RPC.

Native validation session db4bff0c-5647-4a57-a4c4-0695e813374b:
live-native-agent-recovery.cjs replaced all chat/message fetches before interaction,
verified fixture sentinel, held admission response, then real Ctrl+Q. Dialog/process
stayed pending. Released recovery, requested Stop, simulated lost reply, then reloaded
renderer with native pending token. Recovery gate/dialog returned; terminal owner
read was held while counters captured, then released and native process exited.
Exactly one simulated Stop/two owner reads across reload. No Send/Start route exists
in this fixture; no real task mutation this turn. Evidence native-agent-recovery-live.json
includes UI fingerprint. Owner cleanup200. No physical tray/cross-platform claim.

Remaining: desktop lease/session recovery across renderer restart, richer identity
labels for multiple recovered tasks, background transcript completion persistence,
unknown admission/no-match disposition, prior-app focus capture/restore, physical
tray and other mandatory platform/host-binding acceptance and all remaining plan
work. Agent recovery evidence does not prove all-task safe Quit. All phase flags
remain false; full goal active.


## 2026-09-05T14:01Z — Preserve desktop Stop uncertainty through expiry

Inspected desktop recovery boundary. Portal DesktopSessionPanel owned account/lease
in mounted state, silently dropped active lease on browser-clock expiry, and could
accept late Observe/Applications while Stop was pending because a render restored
current.active. Stop also admitted duplicate clicks while another Stop was pending.

Now local expiry fences observations/input and clears stale frame/catalog but
retains active lease and its companion Stop registration. Expiry is not owner
cleanup confirmation. Credentials remain in memory while unresolved active lease
exists (no new credential persistence). Stop sets a synchronous observation fence
and uncertain state before awaiting RPC; a distinct pending guard serializes Stop
without preventing Stop from interrupting an existing input request. Late Observe
and Applications replies cannot repopulate state after Stop begins. Stop button
is disabled only while its own request is pending; explicit uncertain retry remains
available with same lease. New successful Open resets observation fence.

Eleven DesktopSessionPanel tests pass including expired lease retaining onActive,
no expiry-triggered Stop, late observation during pending/lost Stop discarded,
and duplicate Stop click one request. TS/lint and Vite build pass. UI dist current.
No backend/native rebuild, no managed desktop, no real account/lease/task mutation.

DEEPER RECOVERY GAP (not fixed): device-control/internal/control/desktop_proxy.go
Stop resolves from in-memory p.sessions, calls helper Stop, releases owner session,
publishes grants, then deletes binding even if joined errors remain. Subsequent
retry can return PermissionDenied because resolve requires active owner grant.
Helper DesktopController.Stop durably clears Lease/GrantID and records CleanupPending
but retains no stopped lease identity/receipt. Grant authorization rejects expired
or revoked grants, so current helper requests cannot read old Stop outcome.
A missing p.sessions entry, elapsed browser TTL, or inactive owner grant is not
proof ReleaseHeld succeeded. DesktopSessionPanel unmount still best-effort Stops
and loses its mounted reference; full renderer recovery remains incomplete.

NEXT: durable owner/helper Stop outcome keyed to exact original lease/epoch and
authenticated actor, with read-only reconciliation after revocation/restart that
cannot revive input authority. Preserve explicit cleanup-unknown versus confirmed
released. Then expose typed state/list recovery through Portal's desktop facade,
move account/lease ownership into shell provider (credentials only in allowed
storage), and test remount/lost reply/restart/expiry. Do not reuse generic legacy
SessionService.ListSessions as proof of helper cleanup. Missing owner record must
remain unknown. Agent recovery remains implemented; all31 phase flags false.


## 2026-09-05T14:08Z — Durable destination cleanup receipts

Device-control sessions DesktopState now carries pending DesktopCleanupReceipt
records keyed by lease epoch. Each receipt contains the full original DesktopLease
(including actor, helper generation, destination and expiry), Released bool, and
ObservedAt. Stop records cleanup before clearing lease/grant. Same recording applies
to expiry/revocation cleanup, shutdown, helper activation and takeover/Open cleanup.
Failed release persists Released=false. A later successful destination ReleaseHeld
confirms prior pending records without restoring input authority. New helper
activation now preserves old lease until its cleanup evidence has been recorded.

SQLiteDesktopRepository atomically writes cleanup receipts with live-state changes
to a separate embedded-schema table device_control_desktop_cleanup_receipts.
Confirmed entries are removed from live JSON; only pending cleanup remains for
lifecycle retry. Historical receipts do not enlarge every input transaction.
Destination+epoch is the storage key; epoch stored as decimal text for uint64 range.
Existing confirmed receipts cannot be downgraded; conflicting full lease identity
fails transaction. ReadCleanup performs a read-only exact identity lookup and
returns sql.ErrNoRows for missing evidence (NOT confirmed stopped). This repository
method is internal storage access, NOT an authenticated RPC or new input authority.

Tests: sessions, desktophelper and control package race tests pass. Added coverage
for failed Stop retained across separate SQLite connection/repository, successor
helper cleanup confirmation, old actor/epoch mismatch, successor failure preserving
old confirmation, no old lease actuation, expiry requiring actual cleanup, receipt
storage failure producing no confirmation, confirmed history absent from live JSON.
Final sessions race rerun and full device-control API `go build ./...` pass.
No raw package manager, no schema hand-migration. No running device-control restart
or helper deployment this turn: source build validated, live helper still prior
artifact. No physical session/input/task/account mutations, no Portal UI change.

NEXT: authenticated cleanup read protocol. Input grants expire/revoke, so history
must authenticate original actor via owner and bind exact lease/epoch without
reinstating input authority. Need a trusted read path to helper receipts that works
after grant revocation and owner/helper reconstruction. Owner currently p.sessions
in-memory and drops it on Stop; preserve/recover durable actor/session association
and cleanup outcome. Never infer cleanup from absent map entry or released owner
session. Then typed Portal desktop state/list recovery and shell-owned UI account/
lease lifecycle. No renderer restart or all-task Quit acceptance yet. All31 phases
remain false; goal active.


## 2026-09-05T14:17Z — Authenticated helper cleanup readback

Added helper-only DesktopHelperService.ReadCleanup(StopRequest)->CleanupResponse
(exact Lease, Released, ObservedAt). Missing receipt maps NotFound, not success.
Controller calls dedicated AuthorizeCleanupRead and read-only storage, validates
surface/OS desktop, and performs no native operation. Go/TS/proto descriptors and
manifests generated for device-control (Portal dependent output also regenerated).

SignedDesktopAuthority now has separate AuthenticateCleanupPeer path. Historical
signature is checked against strict grant schema/constraints at its issuance time,
refuses future-issued credentials, and verifies actual Unix peer principal. It can
read an expired/revoked grant's exact lease receipt without consulting active grant
state. Private cleanupOnly credential marker means normal Authorize/BindGrant
verification always refuses it, even when original grant is still live. Read also
requires original stop permission and full sameDesktopLease identity. No unsigned
lease/actor/header becomes authority. Owner must authenticate current actor before
forwarding old signed proof; this owner-facing part is not implemented yet.

Unix listener accepts `DesktopCleanup <signed grant>` ONLY at the exact generated
ReadCleanup procedure. Other helper paths retain DesktopGrant scheme and normal
revocation/expiry checks. Scheme swapping fails both directions. No TCP/browser
helper handler exported. New helper method has no owner or Portal facade yet.

Race tests sessions/control passed initially; GTK helper test exposed SQLite_BUSY
in its raw sql.Open fixture. Isolated run reproduced unexpected EOF plus SQLITE_BUSY
from helper cleanup. Replaced only that fixture's DB setup with shared
api-core/databasetest.NewSQLite (standard tuning, one connection, cleanup ordered
through t.Cleanup after helper shutdown). Isolated real GTK Unicode test then passed;
final sessions and full desktophelper race tests pass. This addresses the earlier
known SQLite fixture friction, not an auth bypass or a native input retry.
New tests cover expired/revoked historical read, original actor mismatch, wrong
kernel principal, malformed credential, input/Stop refusal even with live grant,
unchanged durable receipt, Unix RPC NotFound before cleanup and released receipt
after Stop/revocation, and swapped header scheme refusal. Device-control and Portal
API build ./... pass. Live services were not restarted; helper binary not deployed.
No live user desktop/account/grant mutation; GTK runs use their isolated fixture.

NEXT: owner durable association and actor-authenticated read/list facade. Current
owner p.sessions map and signed grant disappear on Stop/restart. Preserve exact
original lease, grant proof and actor in owner-controlled storage, derive history
reads from current authenticated actor, and require matching helper response. Do
not grant any input authority as part of history access. Then Portal desktop recovery
provider and readback UI instead of blind Stop retry. Public/API layer must not
accept actor identity from request JSON or interpret absent history as completion.
All phase flags remain false; full goal active.


## 2026-09-05T14:25Z — Durable owner admission and cleanup readback

New sessions.DesktopAdmissions repository stores immutable original DesktopGrant
metadata in device_control_desktop_admissions (embedded EnsureSchemas, routed DB
interface). No account bearer, signing key or owner lease-token is stored. Put
validates original grant constraints; duplicate identical metadata is idempotent,
conflicting ID/actor/epoch replacement fails. Get scopes SQL by authenticated actor
and session ID then verifies exact Surface/SessionRef and grant constraints.

LocalDesktopAdmission constructs the store from Service's routed DB. Owner Open
persists metadata after Acquire and before publish/helper.Open; storage failure
returns through existing owner-lease cleanup, so no helper admission occurs without
a durable association. Metadata remains after failed helper reply, Stop, and owner
map cleanup. This does not make unknown helper Open successful or imply cleanup.

New DesktopOwnerService and DesktopAccountService ReadCleanup(OwnerStopRequest)
returns OwnerCleanupResponse{SessionRef,released,observed_at}; no helper grant/token
projects to renderer. Actor comes from authenticated owner context/local Unix peer,
not JSON. Reads require sessionActorAllowed for current account, load exact stored
grant, regenerate signed proof using protected original key and original issuance
time, and call helper ReadCleanup with DesktopCleanup scheme. No active map/grant
is recreated. Exact helper Lease comparison and valid observation timestamp required
before projection; missing evidence/errors remain errors. Portal facade not yet wired.

Race tests sessions/control pass; reconstruction test uses real owner/helper Unix
transport then a new owner/proxy with empty active maps and proves cleanup still
readable. Another actor lookup and wrong-epoch helper receipt rejected; no additional
native effect. Store tests immutable replacement, idempotent identical write,
actor scoping and wrong desktop identity. Generated Go/TS descriptors/manifests for
device-control plus dependent Portal refreshed. Device-control API build passes;
Portal API build checked separately. Live helper/owner not restarted or deployed.
No live operator credentials, desktop mutations or new managed sessions this turn.

NEXT: Portal ReadCleanup facade, CLI/readback UI; bounded actor-scoped admission
listing for discovery after renderer restart; shared desktop provider in app shell.
Owner input map still deliberately does not revive after restart. Missing cleanup
receipt must stay unknown, especially unknown Open or old pre-metadata sessions.
Current owner metadata is sufficient for exact known-reference readback, not yet a
complete list/recovery UX. UI currently still offers explicit Stop retry; replace
uncertain retry with this read path once facade is live. Deploy helper+owner through
managed lifecycle with compatible schema/proto after required integration tests.
All31 phase flags false; full goal active.


## 2026-09-05T17:10:51.128697+00:00 — Portal cleanup readback and managed deployment

Added Portal DesktopSessionService.ReadCleanup forwarding only bearer authorization
to DesktopAccountService. Uses existing no-store and sanitized error boundary.
Endpoint catalog, proto Go/TS, descriptors/manifests generated; refreshed Portal UI
local proto-types through SDA. Portal CLI omission follows existing desktop-command
ownership in device-control CLI (cleanup CLI command still needs adding there).

DesktopSessionPanel first Stop stays a mutation. Lost response or local expiry
switches workspace/companion button to Check status; later requests use ReadCleanup
only. Exact protobuf SessionRef, present observedAt and released=true required to
clear active lease. Pending/missing/mismatched/error receipts retain uncertain state.
Unmount no longer sends another Stop if a Stop was already requested/unconfirmed.
Mounted-reference recovery works; route/reload still loses desktop-owned state.

Tests: 15 DesktopSessionPanel tests, TypeScript/lint, Portal surfaces handler race
tests (bearer required, forged metadata stripped, pending response preserved,
no-store header), API/UI builds pass. Explicit tests assert one Stop with multiple
readback failure shapes and no extra Stop on unmount.

Built device-control API/helper and Portal API/UI. Managed restart first stopped
device-control then failed on dependency UI artifact; best-effort reached own UI
but exposed missing RCL dist/hooks/useLocale/versions/1.1.1/useLocale.js. Built
packages/react-component-library via its normal build, refreshed device-control's
approved local file dependency through SDA, then managed best-effort start succeeded
17:08:58Z (API16465/UI20698). Portal managed restart succeeded17:09:15Z
(API17476/UI24965). No release asset source edited, no direct binary launch.
Report-bug skill used; published knw-1788628141895245832 for the missing generated
artifact/restart interruption; root cause unknown, build+SDA refresh is workaround.
Source root and installed dependency initially both lacked the export target.

Live negative reads: Portal ReadCleanup no bearer=>401 unauthenticated; owner Unix
ReadCleanup empty reference=>403 permission_denied. Current owner reports zero
active grants. Evidence desktop-cleanup-deployed.json. No live account, lease,
Stop, screenshot/input session created. Compatible owner/helper protocol changes
from prior turns now deployed. Successful physical cleanup readback remains untested;
unit/integration evidence does not replace platform-native acceptance.

NEXT: bounded actor-scoped owner admission listing (metadata already durable),
Portal facade and shared desktop provider surviving route/remount; account refresh
and same-actor reauthentication without exposing credentials. Add device-control
CLI cleanup readback. Missing historical receipt/unknown Open stays unresolved.
Agent recovery remains active; no full all-task restart-safe Quit claim. All31
phase flags false; full goal active.


## 2026-09-05T17:14:21.228436+00:00 — Desktop CLI cleanup readback

Added `device-control desktop read-cleanup --socket <absolute-owner.sock>
--request <typed-session.json> [--access-token-file <private-token-file>] --json`.
Uses OwnerStopRequest identity and ReadCleanup only. Existing explicit Unix socket
and account namespace rules remain; no API-host discovery or fallback to local
identity. Command manifest marks operation read/run-eligible. JSON emits default
values for this command so pending evidence explicitly includes released:false.
Documentation added to device-control/docs/reference/configuration.md.

CLI tests use real Unix HTTP transport for both owner/account namespaces, verify
exact session and pending result, missing receipt NotFound, and exactly two read
requests with no Stop fallback. Full CLI package tests pass. Primitive evidence
artifact required regeneration for new command; regenerated through the owning
UPDATE_CLI_EVIDENCE=1 test then normal tests pass. Installed via cli-core installer.
Live negative installed-command check rejects invalid session with permission_denied;
artifact desktop-cleanup-cli-live.json. No account/lease/input/Stop mutation or
service restart. Owner/helper/Portal remain deployed from prior turn.

NEXT remains bounded actor-scoped desktop admission listing and Portal shared
account/lease recovery provider above routed UI. Known-reference cleanup readback
now exists in helper, owner, Portal UI/API and CLI; discovery after reload and
credential refresh still incomplete. All31 phases remain false; goal active.

## Desktop admission discovery — 2026-09-05T17:26:23.357705+00:00

Added bounded SQLiteDesktopAdmissions.List(ctx,surface,actor,token,size): default50,
max100, canonical positive int64 rowid cursor; SQL scope matches actor and every
surface/target identity field. Includes historical expired/unknown helper admissions;
no grant refresh or cleanup inference. Full metadata is internal only.
New Owner/Account/Portal ListAdmissions RPC projects SessionRef, expires_at, control,
next_page_token only. Owner derives actor from authenticated context and applies
current read authority; surface comes from configured owner. Portal forwards bearer
only, no-store. Generated Go/TS code with protogen gen-code scoped device-control/portal;
Portal endpoints regenerated and CLI omission declared as destination-owner operation.

Device-control CLI list-admissions uses existing explicit socket/request/private token
flags and emits defaults for clear empty pages. Typed request has page_token/page_size.
Read-governance manifest entry, CLI primitive evidence regenerated, full CLI tests pass,
installed CLI. Operator pagination/unknown semantics documented in configuration.md.

Tests passed: device-control sessions/control TestDesktop; focused owner proxy rerun
after adding failed-helper-Open discovery assertion; Portal surfaces package; all CLI
packages. Repository covers actor/host isolation, expired metadata, reconstruction,
insertions between pages, canonical cursors and limits. Owner Unix/proxy tests verify
no native input, no active maps restored, another account sees empty history. Portal
transport tests bearer-only forwarding, cursor preservation and no-store. CLI tests
local/account namespaces and invalid cursor handling without fallback.

Built API binaries then managed restarts: device-control --best-effort healthy
17:24:43Z; Portal make restart healthy17:25:08Z. Installed CLI live list page_size1
succeeds; active grants remain0. Portal no-bearer ListAdmissions returns401 no-store.
Evidence desktop-admissions-deployed.json. No physical input or lease acquired.

NEXT: shell-owned desktop account/session provider above Outlet, authenticated
discovery on reload with recovery gate; exact cleanup reads for discovered entries;
same-actor refresh/reauth UX. DesktopSessionPanel still owns ephemeral state; do not
claim reload-safe desktop Quit yet. All31 phases remain false and goal active.

## Shell-owned live desktop session state — 2026-09-05T17:35:00.710821+00:00

New ui/src/features/surfaces/useDesktopSession.ts (not TSX; createElement matches
agent-store pattern) owns in-memory account, active SessionRef, opening/stopping,
stopRequested, uncertain, recoveryRequired via synchronous external store. AppShell
wraps Outlet inside DesktopSessionsProvider under CompanionPresentation. Provider
registers desktop Stop task and runs expiry timer. Open after route unmount retains
lease in shell; panel unmount does not send Stop. Stop serialized and uncertain
subsequent calls use exact-session ReadCleanup only; expiry never means cleanup.
Open error sets recoveryRequired and blocks further admission; reconciliation method
NOT IMPLEMENTED YET, so unknown-Open gate currently persists until future integration.

DesktopSessionPanel retains frame/app catalog/field draft only. Screenshots clear on
Stop/expired lease and are discarded on navigation. Store exact-reference fences
prevent old panel responses affecting successor state. SurfaceCatalog selects active
lease surface and prevents changing destination while shell work remains unresolved.

18 panel tests pass including late Open retained across navigation, uncertain Stop
survives remount/read-only reconciliation, late old input cannot poison successor,
and shell control Stop works without panel. SurfaceCatalog3 and AppShell6 tests pass.
TS and targeted lint pass. Shell axe test had async ModeIndicator query act warning;
wrapped axe async operation in act, rerun passes. No test-console suppression.

NOT DEPLOYED: current live Portal dist still pre-provider, API/owner admission listing
deployed17:24-17:25Z previous turn. Need finish provider discovery/credential lifecycle
before next UI build/deployment. Account stays memory-only; no reload recovery gate
yet. New recoveryRequired Open ambiguity branch is intentionally fail-closed but UI
Check-status recovery action still needs implementation.

NEXT: extend provider to enumerate actor-scoped admissions on authenticated login,
keep multiple unresolved historical identities, exact cleanup read each; after
reload show sign-in recovery gate using noncredential marker only (never token
storage). Add same-account identity/refresh/reauth lifecycle and UI actionable
recovery status. Then build/deploy and native reload/Quit verification. Goal active,
all31 phase flags unchanged false.

## Desktop authenticated recovery and credential rotation — 2026-09-05T17:43:14.592316+00:00

Extended useDesktopSession.ts shell store (still NOT DEPLOYED) with read-only
admission reconciliation. Login starts recovery gate. Enumerates pages50, max50pages,
rejects repeated cursor/oversized page; retains Map of exact historical identities
from partial/previous traversals; up to4 concurrent cleanup reads. Only exact released
receipt removes entry. Later empty pages do not erase retained unknowns. All calls
read-only, no Open/Stop retries. Companion Stop without active session uses recovery;
DesktopSessionPanel exposes Check status when recoveryRequired and no active lease.

Private openUnknown flag distinguishes an uncertain Open from normal admission
history. Empty history or old successful cleanup cannot clear that flag; no correlation
is currently available between caller Open and server-generated session ID. Thus
unknown-Open remains gated even after historical cleanup. This needs owner-issued
request disposition/correlation (or equivalent authoritative proof) before final
completion; do not infer that no rows means no effects. No document reload marker yet.

Account now includes id+realm from authenticated account response. Refresh is
serialized and uses rotating refresh token in memory, triggered15s before expiry.
Success retains same actor and active lease. Failure sets needsAuthentication and
never retries the same refresh automatically; UI shows localized original-email
reauth notice and hides native flow controls. Reauthentication accepts only same
id/realm while retaining unresolved references/Stop fence. No tokens stored on disk.
Read/input gates enforce current auth/recovery state synchronously.

23 panel tests pass plus SurfaceCatalog3, AppShell6, shell axe1; TypeScript/lint pass.
Added tests: multiple historical identities; later empty page preserves pending;
partial repeated-cursor traversal; empty history cannot resolve unknown Open;
serialized refresh and new-token Stop; wrong-actor reauth rejection with exact
cleanup using new token. Lease-expiry test now advances61s (lease60s, token120s)
to isolate lease expiry from separate credential expiry/refresh behavior.
Locales en/ja/ar updated generic uncertain text plus reauthentication notice;
strings generated. Still no UI build/deployment, no live sessions/input created.

NEXT: persistent noncredential actor/recovery marker and initial gate across reload;
handle multiple unresolved actor histories conservatively; authenticated reload
discovery with reauth UI reachable from shell. Owner unknown-Open correlation is
an explicit remaining gap, as is failed admission/no cleanup receipt disposition.
Then UI build/deploy and native reload/Quit validation. Full31-phase goal active.

## Persistent desktop recovery and native Quit proof — 2026-09-05T17:55:36.646939+00:00

Added desktopRecoveryMarker.ts: localStorage key portal.desktop-recovery.v1; version1,
actor id+realm, unknownOpen bool, protobuf JSON SessionRefs only. Strict parse/identity
bounds, max2500refs/4MiB; corrupt/unavailable storage restores a conservative gate.
No token/refresh/password/email/screenshot in marker. Store constructor recovers
references into read-only Map, expectedActor pin, restored/recoveryRequired/needsAuth.
No active lease restored. publish persists unresolved state; Open must successfully
write an opening marker BEFORE RPC. Persistence failure gates further control.
Positive exact cleanup removes references/marker; empty history cannot remove known
retained refs or unknownOpen. Same-actor reauth survives constructor reconstruction.

AppShell mounts DesktopRecoveryPanel above Outlet so recovery sign-in is available
on any route. DesktopSessionPanel surface now optional for recovery-only mode; no
Open without explicit surface. SurfaceCatalog avoids duplicate recovery UI. Added
localized generic original-account sign-in message (en/ja/ar); generated strings.
Configuration doc explains marker/security/recovery semantics and unknown-Open gap.

Tests: 26 desktop panel tests, AppShell6, axe1 pass; TypeScript/lint pass. Added full
provider reconstruction, wrong actor rejection, exact read-only cleanup, token-free
marker assertions, unknownOpen persists through remount, storage failure prevents
Open. Vite build passed (App-vfe1oc1s.js), managed Portal restart healthy17:51:29Z.
All previous shell ownership/discovery/refresh UI work is NOW DEPLOYED.

Native validation: reused managed live-native-session.py create/launch, session
69c91284-fa73-4740-acfb-f600d2981011. New live-native-desktop-recovery.cjs proves
window.fetch interception BEFORE actions, seeds fixture marker, intercepts all
Operator/Desktop RPCs and forbids mutation methods. Real native CtrlQ blocks before
reauth, pending Quit survives renderer reload, Cancel permits required reauth,
exact simulated positive cleanup allows guarded Quit. Result passed:2login,2read,
0Open,0Stop; no real desktop lease/input. native-desktop-recovery-live.json contains
evidence. Owner cleanup200; active real grants0. Native package unchanged, dev-only
Electron33.4.11 scoped validation, real tray click/otherOS remain unproven.

NEXT HIGH GAP: authoritative unknown-Open correlation/disposition. Client Open has
no request ID, owner creates SessionRef only after Acquire, so empty list cannot
prove that lost Open never ran. Failed helper admission may retain metadata with
no cleanup receipt forever. Need durable owner request identity/disposition and/or
helper epoch-fenced absence proof; inspect exact owner/helper ordering before
choosing. Do not clear unknownOpen from old receipts or absence alone.
Other remaining full-plan work unchanged; all31 flags false and goal active.

## Durable Open attempt state machine foundation — 2026-09-05T18:06:53.615908+00:00

Added Open attempt storage to existing sessions/desktop_admissions.go. Constructor
EnsureSchemas includes device_control_desktop_open_attempts(actor,request_id PK,
payload,expires_unix,state,grant_metadata), state reserved/forwarding/not_admitted.
No credentials/signing keys/owner lease tokens; grant metadata is internal only.
DesktopOpenIntent holds immutable surface/control/TTL. IDs canonical nonzero UUIDs.
ReserveOpen returns created=true only once, rejects payload changes, duplicate does
not extend deadline. ReadOpen is pure read and actor-scoped. BindOpen validates exact
actor/surface/control/TTL grant, CAS reserved->forwarding before helper boundary,
returns claimed=true once only; duplicate same grant false, replacement rejected.
RejectOpen only transitions reserved->not_admitted; cannot relabel forwarding as
no-effect. ExpireOpenReservations is explicit owner lifecycle reconciliation that
CAS-closes expired reservations; ReadOpen does NOT treat wall time alone as terminal.
Cancelled IDs cannot reserve again or be bound by a delayed handler.

Tests pass: all TestDesktop in sessions/control. New tests verify immutable payload,
actor scope, preserved deadline, one-time forward claim, second SQL connection
reconstruction, immutable grant, forwarding not cancelled by expiry, and explicit/
expired cancellation fences delayed binding/re-reservation. No RPC/provider wiring
yet, no deployment/service changes this turn. Existing Portal native recovery
17:51 deployment remains live; previous validation session already cleaned.

Reasoning: helper epochs alone do not correlate a lost Open to its request; no
cleanup receipt can also mean never-admitted. No epoch-based inference was added.
NEXT wire request IDs into OwnerOpenRequest and immutable attempt before Acquire;
claim before helper.Open, duplicate must never repeat acquisition/forwarding.
Expose exact actor-scoped request reconciliation and persist pending ID in Portal
marker. For an entirely unreceived Open, a reconcile/cancel tombstone must fence
a later arrival before certifying not_admitted; a plain missing SELECT is not proof.
Potential ReconcileOpen can reserve exact payload then CAS reject reserved, or return
forwarding binding if worker won; no helper mutation/retry. Forwarded requests still
require exact cleanup evidence. Failed helper Open with no receipt needs separate
authoritative disposition (e.g. grant revocation checked inside helper repository
exclusion and exact absence proof), not expiry/absence inferred in renderer.

Open attempt methods currently concrete on SQLiteDesktopAdmissions; add owning
interface when integrating. Ensure owner lifecycle calls ExpireOpenReservations
(currently unused), with compare-and-set preserving forwarded requests. Goal active;
all31 phase flags unchanged.

## Owner Open correlation and cancellation reconciliation — 2026-09-05T18:13:39.207586+00:00

Extended DesktopAdmissions interface with reservation/binding/rejection/reconcile/
expiry methods. admissionDB and control.routedDB now include BeginTx; actual sql.DB
and api-core RoutedDB support this, compilation/tests pass. BindOpen now atomically
updates forwarding state AND writes immutable desktop admission metadata in one
transaction. Metadata conflict rolls back claim, so helper must not execute.
ReconcileOpen reserves exact payload if unseen, then CAS rejects reserved; late
original Open sees existing request and cannot acquire/forward. If Bind already
won, it returns forwarding plus exact grant binding instead. No helper calls.

Proto: OwnerOpenRequest.request_id=4 optional for legacy callers. Added
DesktopOwnerService.ReconcileOpen and DesktopAccountService.ReconcileOpen using
OwnerOpenRequest -> OwnerOpenDisposition(request_id,state,session,expires_at,control).
Generated Go/TS via scoped protogen gen-code. Portal service facade NOT added yet.

Owner Open with request_id derives/authenticates actor+control first, reserves before
Acquire, and rejects duplicate requests without repeating acquisition. Deferred
pre-forward failures Reject reservation with independent bounded cleanup context.
For correlated requests Bind happens after grant publication and immediately before
helper.Open, also storing metadata atomically. Legacy no-ID path retains old Put.
ReconcileOpen checks configured surface and current original actor read authority;
returns exact reference only, no helper grant/token. GrantStatus lifecycle publication
now expires unforwarded reservations; forwarded rows remain immutable/uncertain.

Validation: all sessions/control TestDesktop, LocalDesktop/ForwardedDesktop tests
pass. Owner Unix fixture verifies cancel-before-arrival -> later Open denied with
no owner live lease/helper calls; correlated success -> same exact binding and
duplicate/reconcile no helper replay. Metadata collision test proves transactional
rollback to reserved, then cancellation safe. Device-control API build passes.
No managed restart/deployment this turn.

NEXT: Portal ReconcileOpen facade/no-store plus CLI command with effect appropriate
for admission cancellation (not a purely read-only command). UI must generate and
persist request_id+immutable surface/control/TTL before Open; recovery uses
ReconcileOpen to fence late arrival, accepts not_admitted as terminal for that
request only, or adds exact forwarding SessionRef to existing cleanup recovery.
Clear unknownOpen only for exact reconciled request, never for empty admission list.
Legacy unknown bool markers without request identity remain unresolved.
Still need helper authoritative disposition for forwarding claim whose helper
admission never occurred (no cleanup row), e.g. revoked-grant plus repository
exclusion before exact absence check. Do not infer this from expiry alone.
Full plan goal active; all31 flags unchanged.

## ReconcileOpen Portal facade and CLI deployment — 2026-09-05T18:17:58.885166+00:00

Added Portal DesktopSessionService.ReconcileOpen(OwnerOpenRequest)->OwnerOpenDisposition
proto, generated Go/TS; facade uses existing bearer-only desktopForward and no-store
wrapper. Endpoint desktop_reconcile_open generated; Portal CLI omission directs to
destination operator command. device-control desktop reconcile-open added, using
explicit socket, typed original request file, optional private account token file;
no fallback to Open/Stop/local identity. Manifest governance effect write (same as
Stop, run_eligible true), since reconciliation cancels admission reservations.

Portal transport tests validate bearer/no Cookie/X-Actor forwarding and request ID/
not_admitted/no-store; CLI Unix fixtures exercise local/account namespaces, ID and
state retention, invalid identity rejection, no unexpected method calls. Full CLI
tests pass after primitive evidence regeneration; CLI installed. Operator config
doc explains unique UUID request_id, immutable payload, cancellation and forwarding
disposition. API builds pass.

Managed deployment: device-control --best-effort restart healthy18:15:47Z (includes
previous owner attempt state machine/RPC/lifecycle); Portal make restart healthy
18:16:49Z. CLI invalid request rejects permission_denied; Portal no-bearer endpoint
returns401 no-store. Evidence desktop-open-reconciliation-deployed.json. Active
grants0; no actual Open/Stop/input during live checks.

NEXT: useDesktopSession.ts + desktopRecoveryMarker.ts must persist typed request ID
and immutable Open payload BEFORE issuing Open, and use ReconcileOpen during
recovery. Accept not_admitted only for exact request ID; for forwarding retain exact
SessionRef and require existing cleanup evidence. Current UI still sends no request
ID and retains legacy unknownOpen bool. Legacy marker without ID must not be
reinterpreted as cancelled from empty history. Add tests and native fixture for
unknown Open where original request never reached owner and late arrival is fenced.
Separate gap: forwarding claim but helper never admitted => no cleanup receipt;
needs authoritative helper absence/revocation proof, not renderer inference.
All31 phase flags unchanged; full goal active.

## Persistent UI Open identity and native cancellation recovery — 2026-09-05T18:26:37.940674+00:00

useDesktopSession.ts now deep-clones a typed OwnerOpenRequest with crypto UUID,
surface/control/TTL, and persists it BEFORE Open. Storage failure sends no Open.
Success replaces pending request with exact session; failure retains request.
Recovery calls ReconcileOpen first, validates exact returned request_id and state.
not_admitted without a session clears that pending request; forwarding requires
matching surface/control/session/expiry, enters read-only cleanup Map, and cannot
restore input. Other/mismatched/failed responses retain pending request.

Marker JSON version2 includes optional typed pendingOpen; version1 is still read.
Old binaries reject v2 rather than misread a correlated request as no work. Legacy
v1 unknownOpen bool remains uncorrelated and never cleared from empty history.
Credential-free actor/session/request fields only; no bearer/refresh/password.
Also fixed Stop during pending Open: queues cancellation, then stops the returned
lease before observation (or reconciles the uncertain request) once Open settles.

31 panel tests pass, TS/lint/build pass. Added reload exact-ID reconciliation,
wrong-ID rejection, forwarding then exact cleanup, legacy unknown marker retention,
and queued shell Stop. Documentation updated. Portal managed restart healthy
18:23:48Z, UI App-CJemKp8-.js; owner remains deployed18:15:47Z.

Extended existing live-native-desktop-recovery.cjs with --unknown-open (separate
artifact native-open-recovery-live.json). Real native CtrlQ blocks before reauth,
pending Quit survives reload, and exact simulated not_admitted disposition permits
Quit. Proved document fetch interception before actions; all desktop/account RPCs
intercepted, Open/Stop prohibited.2login,2reconcile,0cleanup-read,0Open/Stop. Managed
session478a8d75-2071-4592-9723-bc9aa5d53368 cleanup200; active real grants0. Scope
is native shell + controlled owner responses, not a physical desktop admission.

NEXT HIGH GAP: forwarding claim whose helper never admitted => missing cleanup
receipt remains forever unknown. Need owner revocation + authoritative helper
absence proof inside the repository exclusion boundary, since snapshot absence
or elapsed time alone cannot rule out an in-flight Open. Controller.Open checks
active signed grant again INSIDE repo.Update; proof can fence using revoked grant
and acquire same exclusion before reading exact lease/receipt. Preserve failed
cleanup receipts and reject wrong actor/principal/ref. Read-only credential alone
must never grant native authority. No helper absence inference implemented yet.
Full plan goal active; all31 phase flags unchanged.

## Signed helper revocation and never-admitted recovery — 2026-09-05T18:46:30.966613+00:00

Owner ReadCleanup signs permanent revocation only after its exact historical grant is inactive. Helper historical authorization validates this distinct signed statement; it cannot authorize Open/input. SQLite stores immutable grant-to-lease revocation under the same transaction exclusion used by Open/Act. Absence proof waits for in-flight admission, and original grants remain fenced after repository reconstruction or stale active publication. Existing live leases and failed cleanup receipts remain pending; lifecycle performs native release and only successful cleanup confirms them. No renderer timeout or empty inventory is treated as proof.

Targeted sessions/control tests and race tests pass, including injected revocation-write failure, stale grant reactivation, admitted cleanup failure/retry, concurrent admission and owner/helper Unix transport. API/helper built and managed device-control restart succeeded healthy at 18:41:56Z. Deployment evidence: desktop-revocation-deployed.json; active grants 0. Configuration docs updated. This closes the helper never-admitted gap identified above.

Full plan remains active. Next review the broader phase/acceptance scope: native focus capture/restore, context capture, platform adapter/support evidence, streaming, cross-target journeys and saved-flow parity still require implementation/evidence. Avoid treating this recovery slice as completion of any whole phase.

## Native shortcut reconfiguration — 2026-09-05T18:50:08.901426+00:00

Added session-scoped activation shortcut reconfiguration in canonical S2D native/presentation.ts and preload bridge. Trusted main-frame IPC validates same bounded accelerator grammar and extension v2 permission. Registers replacement BEFORE unregistering working shortcut; failure retains recovery path, same binding idempotent, cleanup unregisters current binding. Revisioned snapshot publishes successful change. Portal toolbar exposes localized setting/apply controls only on capable builds; serializes with mode transitions and reads authoritative snapshot after lost reply without repeating registration. Setting explicitly resets to packaged default on restart.

20 native presentation tests and template TypeScript passed; native presentation lint passed (preload ignored by existing lint config). Portal11 companion tests and focused lint passed; UI TypeScript/strings check passed before final test addition. Lost-reply regression test corrected to Vitest2-supported matchers. Removed Corepack incidental packageManager insertion. Config docs and EN/JA/AR strings updated through strings generator.

SOURCE ONLY: next build/regenerate native package through Scenario-to-Desktop, deploy Portal UI, and validate live shortcut replacement/conflict; no current package or physical acceptance claim. Pre-focus app/pointer context and prior-app focus restoration still missing. Full plan active.

## Packaged shortcut conflict recovery — 2026-09-05T18:53:38.061638+00:00

S2D generation b30a68f2-758f-6377-d023-9c5981f70934 completed; verified generated source includes new IPC/preload. SDA exact Electron33.4.11 install used existing dev-only approval; generated tsc and electron-builder --dir --linux passed. Portal Vite build passed (App-CqydN2Wh.js), managed restart healthy18:52:14Z. Current package fingerprints refreshed in native-companion-package.json.

Extended existing live-native-shortcut.cjs --reconfigure. Managed session37c936e1-990f-4a9e-b5e9-09134d8eaf61 reserved default CtrlShiftSpace before app startup: actual native status unavailable/canHidefalse. User toolbar applied ControlAltP; native status registered/canHidetrue. Hid real window, verified X11 unmapped, owner sent real ControlAltP, app reopened palette560x480 with same renderer/draft node and draft text. Passed18:52:51Z. No submitted chat or desktop admission; managed session cleanup200. Dedicated native-shortcut-reconfiguration-live.json preserves receipt.

Next missing companion requirement: capture prior active app/pointer context before native focus, then restore prior focus on dismissal using appropriate desktop owner/native boundary. Shortcut setting is intentionally this-session and UI labels state that. Cross-platform acceptance, overall companion context UX and full plan remain incomplete.

## Native pre-focus context foundation — 2026-09-05T18:56:36.871638+00:00

Added internal/native/x11/context.go CaptureActivation primitive in Device Control owner. Uses existing authenticated X server/session, bounded operation deadline/mutex, EWMH active window and process metadata, QueryPointer root coordinates/child, geometry revision and timestamp. Does not capture pixels/text or change focus. Requires mapped active window and exact bounded property types. Rechecks active window, geometry and session permission before returning. Returns zero context on error. X window/PID values explicitly evidence only, never capabilities; pointer child can be decoration, not unique app/element.

Isolated Xvfb test validates prior window/pointer values, unchanged focus, missing WM metadata, destruction, initial denied permission and mid-operation revocation. Targeted native capture/context race tests passed.

SOURCE FOUNDATION ONLY: not exposed in helper/controller or consumed by companion. Next move a portable context contract into sessions, forward through semanticBackend and authorized helper operation (observe permission), then add companion local-host-bound activation hook that awaits bounded capture BEFORE reveal/focus. Avoid API-host desktop substitution and exposing native IDs as authority. Focus restoration still requires owner-owned opaque token, lifetime/revalidation and native adapter behavior. No pre-focus product acceptance claim yet; full plan active.

## Authorized activation context domain operation — 2026-09-05T18:58:33.475358+00:00

Moved activation context contract to sessions.DesktopActivationContext (uint64 native window identities, int32 pointer coordinates for portable negative origins). X11 aliases it. DesktopController.CaptureActivation checks observe authority before and within repository exclusion, limits duration2sec, uses nativeContext so Stop interrupts, rechecks admission/authority/context cancellation after native operation, validates bounded display/geometry and nonzero window/PID plus timestamp freshness<=2sec. Errors return empty context. No input authority or durable receipt metadata added. semanticBackend forwards to optional native observer without reading accessibility text.

Targeted race tests passed across sessions, native/x11 and desktophelper: valid negative coordinates, denied/wrong-session no native call, midcapture revoke/expiry/cancel, stale/future/missing window, Stop interruption, existing semantic tests and physical isolated X11 context.

NOT EXPOSED ON RPC OR DEPLOYED. Next helper-owned ephemeral opaque context references bound to exact lease/actor/helper epoch and expiry; raw window/PID observations must not become renderer-provided focus authority. Add typed helper/owner transport then companion local-host-bound hook before reveal. Restoring focus requires revalidation and safe native owner action; no completion claim for pre-focus product behavior.

## Opaque activation references and helper RPC — 2026-09-05T19:01:45.166560+00:00

DesktopController retains at most one ephemeral activation context per destination, protected by activationMu under repository exclusion. CaptureActivationReference captures then rechecks current authority/admission before issuing UUID reference. New capture invalidates old; delayed older capture cannot replace newer. Expires within30sec or earlier lease expiry. ReadActivation requires original exact lease, current observe authority and current admission, no native recapture. Expired access clears cache; reconstructed controller has none. Native window/PID remain helper memory only. Raw cache is lazily cleared, never journaled.

DesktopHelperService adds CaptureActivation(StopRequest)->ActivationReference and ReadActivation(ReadActivationRequest)->ActivationReference. Safe wire reference only contextUUID/display/geometry/pointer coords/timestamps, no OS IDs. Typed Go/TS code, descriptors/manifests regenerated via canonical tooling (device-control and dependent Portal). Handler delegates domain checks.

Race tests pass: lease binding, revocation, replacement, expiry, reconstruction, earlier lease deadline, Stop invalidation. Actual authenticated Unix helper test covers missing and cleanup-only credential rejection, signed capture/read, revoked read denial and zero native effects. Existing domain activation/Stop tests pass.

SOURCE ONLY, not deployed. Next add owner/account proxy RPC and native companion local-host-bound pre-focus hook. Owner must sign observe grant and exact session; avoid API-host substitution. Plan pre-focus and restore-focus product cases remain incomplete. Full plan active.

## Owner activation context routing — 2026-09-05T19:04:13.428874+00:00

Added CaptureActivation and ReadActivation to typed DesktopOwnerService and DesktopAccountService, routing through existing exact-session resolve and signed helper observe grant. No target inference or local-account fallback. Rechecks owner admission and same grant after helper response. Validates UUID/nonzero/canonical ref, display/geometry bounds, timestamps, <=30sec lifetime and lease deadline; exact contextID required on read. Returns known-field projection only. Go/TS/descriptors/manifests regenerated. Configuration docs describe ephemeral behavior.

Owner/helper actual Unix integration test capture/read preserves ref, rejects other desktop-login ref. Response validation tests reject missing/invalid IDs, future capture, expired/overlong/context beyond lease. Targeted owner/helper/reference race tests pass; go build ./... passes.

SOURCE ONLY; helper/owner binaries not deployed. Next native companion local activation hook needs a managed local owner session reference and socket/principal binding, not Portal API host selection. It must capture before reveal/focus, bound wait and allow text UI on failure. Focus restoration still not implemented. Full plan active.

## Activation CLI and managed deployment — 2026-09-05T19:07:30.997040+00:00

CLI desktop capture-activation/read-activation added, explicit absolute owner socket + typed request JSON; optional private account token file selects account service only. No fallback or replay. Manifest capture effectwrite (replaces ephemeral context), read effectread. Primitive evidence regenerated through owning UPDATE_CLI_EVIDENCE test. Full CLI package tests passed, including both authority namespaces, exact RPC selection, error propagation, no extra calls, typed unknown-field rejection. Lifecycle auto-installed CLI.

Built API and helper binaries; managed device-control restart healthy19:06:47Z with explicit owner/helper configs. Live CLI empty-session capture and read return permission_denied. Active grants0. No real context or native input captured. Evidence desktop-activation-deployed.json. Docs include CLI operations.

NEXT: native companion activation hook still entirely pending. Current CLI accepts a supplied existing local/account owner session; shell must bind that to actual companion host and desktop before capture. Cannot infer companion identity from Portal API host or route a remote selected surface as prior local app. Need bounded await before native focus, opaque context delivery/status in UI, and no blocking ordinary chat if unavailable. Focus restoration remains missing. Full31phase plan active.

## Bounded native pre-focus sequencing hook — 2026-09-05T19:11:01.539993+00:00

Canonical S2D native/presentation.ts installPresentation accepts optional trusted-main-process ActivationCapture adapter (not renderer IPC). Global shortcut and replacement shortcut use same activate function. Capture runs BEFORE show/focus, one pending per window, max750ms; failures/timeouts open palette with unavailable context and abort underlying adapter. Safe UUID/expiry-only result retained in snapshot, expires<=30sec, native IDs excluded. Manual mode change/reveal, navigation, prepareQuit and close cancel pending capture; late results cannot focus or overwrite newer state. No callback means existing immediate activation unchanged.

23 native presentation tests pass, TypeScript and focused lint pass. Cases cover before-focus ordering, duplicate press coalescing, timeout/late result, manual mode/quit supersession. SOURCE ONLY. main.ts does NOT provide adapter, so packaged product still has no actual pre-focus capture.

NEXT REQUIRED: implement trusted local owner adapter and wire main.ts. It must use explicit owner socket/existing lease and verify companion process desktop-session identity through control-plane host inspection (or equivalent owner-attested binding), never infer API-host desktop. No renderer-supplied executable/socket/OS ID authority. Need opt-in native extension permission/version/config handling, bounded IPC cancellation, user-facing captured/unavailable status, live prior-app fixture. Existing helper/owner/CLI deployed19:06:47Z. Focus restoration remains missing; full plan active.

## Native local-owner activation adapter — 2026-09-05T19:15:12.121932+00:00

Added canonical S2D native/activation-owner.ts. Trusted binding exact Unix socket/session cloned once; local owner CaptureActivation only, no account fallback/Open/retry/networkhost discovery. Before each capture, static vrooli host desktop-session command verifies current Electron process PID/UID belongs to exact active unlocked local X11 logind session; parses canonical snake_case CLI facts with<=2sec freshness. Host inspection is control-plane owned. AbortSignal propagates through execFile/HTTP, bounded700ms calls/16KiB outputs. Safe returned ref only UUID/expiry projection; presentation hook validates UUID.

Tests pass for fresh/mismatched/stale/locked/remote facts, actual Unix transport with controlled host-probe reply, cloned binding, exact local RPC/no authheader, and zero network calls when host association denied. TypeScript and focused lint pass. This is NOT a real logind/Electron acceptance result.

SOURCE ONLY. main.ts still does not provide adapter. NEXT: private user-owned native binding config loading (no renderer configuration authority), explicit versioned native extension context permission and generator validation, then main wiring. Need robust unavailable fallback on malformed/missing config. Existing local observation lease is required; avoid auto Open/retry. Eventually user-facing setup must establish/manage it, rather than requiring permanent manual lease files. Windows/macOS and managed Xvfb session attestation remain unsupported by this adapter; do not silently skip verification. Full plan active; pre-focus product journey and focus restoration incomplete.

## Version3 native context permission/configuration and package — 2026-09-05T19:20:22.951710+00:00

Go generator, build-tools validator and runtime accept version3 with exact window.presentation/global-shortcut/desktop.context permissions. Older versions cannot receive capture adapter. Portal desktop/native-extension.json nowv3. main.ts passes configuredLocalActivationCapture only forv3. Trusted VROOLI_DESKTOP_ACTIVATION_CONFIG absolute path is reread each activation; Linux current-user mode0600 regular file<=16KiB, O_NOFOLLOW/O_NONBLOCK, no symlink/FIFO. Missing/invalid config rejects capture, not app startup. Existing lease binding required; private owner may renew atomically. Docs updated.

25 native tests, template TS/lint, Go Native tests,13 build-tools native extension/generation tests and build passed. Rebuilt S2D API and managed restart healthy19:18:13Z (SKIP_DESKTOP_DEPENDENCY_INSTALL=1 preserved). Generation0a1cbf70-0e02-483a-7e5a-18695fbb8955 complete, SDA Electron33.4.11 approved install, generated tsc + electron-builder Linux passed.

Real managed session64d5bff8-caa7-49ff-b8d1-c0c120a94786, native CtrlShiftSpace from hidden opened palette560x480, same draft/node; native activation state unavailable/no context with missing config. Dedicated native-context-unavailable-live.json; cleanup200. Package fingerprints refreshed. This verifies optional fallback only, NOT prior-app capture.

NEXT: user-facing native activation status/context reference in Portal; positive prior-app proof requires real companion logind session binding and owner existing lease. Current managed Xvfb harness lacks logind attestation; do not bypass production verification. Need owner-managed setup/renewal rather than permanent user-maintained files. Context pointer annotation and focus restoration still missing. Full plan active.

## Visible native activation status and expiry — 2026-09-05T19:23:37.267916+00:00

Portal CompanionPresentation validates optional native activation snapshot (capturing/unavailable/ready with UUID and bounded expiry), tracks revision and displays localized EN/JA/AR live status. Ready expires locally to unavailable without recapture or input/session effects. Existing web/older shell with no activation field displays none. Context stays shell memory only; no native ID/opaque UUID exposed in label or persisted. No attachment or focus authority claimed.

13 companion tests pass, focused lint/typecheck/strings/build pass (App-BVAh6G8m.js). Tests cover progress->ready->expiry, preserved chat, no recapture, malformed context rejection. Portal managed restart healthy19:22:44Z. Extended live-native-shortcut --expect-unavailable asserts visible status. Managed native session69885557-507d-4fed-a813-e6c97bc3c394 verified hidden->palette, unavailable label and same draft/node at19:23:04Z. Cleanup200, native-context-status-live.json. Package remains prior version3.

NEXT: positive prior-app capture proof and practical owner-managed local session setup/renewal. Current private binding requires existing local observation lease and actual logind X11 association; harness Xvfb alone is insufficient. Need investigate actual host/managed attestation support without bypassing checks. Context attachment/annotation, explicit removal and focus restoration remain incomplete. Full plan active.

## Positive real-X11 owner capture evidence — 2026-09-05T19:25:57.909977+00:00

Inspected loginctl: login2 UID1000 local X11 active/unlocked, display0 helper already bound. Agent process is user-service/vrooli-agent scope, not logind session2 scope, so do not use it as companion identity or weaken verification.

Actual deployed CLI owner Open controlfalse TTL30 exact configured surface, request620ac24b-47c8-4607-be54-ec8db7e06645. Session2434c8b6-ab71-4069-8af8-338cbcd4586d captured real native context0c30bdaf-1494-4783-97c4-9ac1cd5dbbd1 at19:25:02.692Z, then read exact same reference. Only safe display/geometry/pointer metadata returned. No screenshot/text capture, focus change or input command. Stop succeeded and ReadCleanup releasedtrue at19:25:02.772Z; post-stop read denied, active grants0. Dedicated desktop-activation-native-live.json.

This proves real native owner/helper capture and lifecycle, NOT packaged companion prior-app ordering. Existing native hook ordering has unit evidence only. Need positive packaged launch on correctly associated desktop or extend owner-managed validation attestation without bypass. Also investigate whether modern graphical app scopes live outside logind session cgroup: current companion PID-based check may reject legitimate launches; proof should use an authoritative native connection/session association instead of relaxing to UID/DISPLAY alone. User-managed context binding/renewal still incomplete. Full plan active.

## Correct companion identity boundary: XRes window ownership — 2026-09-05T19:29:21.415718+00:00

Read actual host process cgroups: Xorg3224 belongs session-2.scope, but GNOME Shell3555 and ordinary Chrome157512 are under user@1000.service/session.slice or app.slice. Thus current Electron-PID logind-cgroup adapter check rejects normal graphical launches. Do not relax to UID or DISPLAY: use native X resource ownership on helper verified display.

Added x11.Backend.VerifyWindowProcess(ctx,window uint64,expectedPID uint32). Checks window geometry root, session authority, XRes>=1.2, uniquely resolves client resource base/mask through QueryClients (<=4096), then QueryClientIds LocalClientPID exact base/mask/value. Final session recheck. PID MUST originate from owner kernel IPC peer, never request JSON. _NET_WM_PID is explicitly insufficient. No native effect. Existing xgb/res package reused; no dependency install.

XRes initially returned client base rather than supplied windowXID; diagnosed actual reply and corrected via unique base/mask resolution. Race test passes on isolated Xvfb: actual creatorPID accepted, wrong PID/root rejected, spoofed _NET_WM_PID rejected, session permission denied rejected. Protocol reference https://sources.debian.org/src/xorgproto/2025.1-1/resproto.txt .

SOURCE PRIMITIVE ONLY. Existing companion PID/cgroup guard has NOT been removed. NEXT: owner extracts authenticated Unix PID (not caller JSON), adds companion-window capture request, signs/forwards bounded helper verification, helper validates resource against that PID around capture. Native main supplies its own BrowserWindow handle through trusted adapter. Only then replace obsolete process-cgroup path. Positive companion prior-app proof still pending. Full plan active.

## Companion native-window identity routed from kernel peer — 2026-09-05T19:33:37.456337+00:00

Added api-core/localprincipal.PeerPID Linux SO_PEERCRED; other platforms fail unsupported. Owner local-only CaptureCompanionActivation request contains exact SessionRef and uint64companion_window, noPID. Rejects account-forwarded identity, derivesPID from actual Unix connection, uses existing current locallease resolution. Helper companion request carries owner-derivedPID under signed observe transport. DesktopController verifies native window/process before AND after capture inside repository exclusion, then issues existing ephemeral ref; errors no ref. semanticBackend forwards verifier to pixels.

Native activation-owner now calls CaptureCompanionActivation with BrowserWindow native handle read by trusted main closure. Validates4/8byte little-endian X11 handle<=uint32 nonzero. Removes disproven Electron-process logind-cgroup check and childprocesshostCLI dependency. Actual helper Xserver session verification remains unchanged. No renderer-suppliedPID/windowIPC, no account fallback. Source documentation updated.

Actual owner/helper Unix test derives testprocessPID and fake native verifier requires exactly it, rejects wrongwindow. Domain tests before/after ownershipfailure no context; real isolated XRes test wrong/spoofedPID denied. Targeted Go race tests pass, localprincipal tests pass; native adapter2tests/TypeScript/lint pass. Proto Go/TS/descriptors/manifests regenerated.

SOURCE ONLY: owner/helper live binaries and native package still prior process-cgroup implementation. NEXT build/deploy owner/helper, regenerate/repackage native through S2D with SDA exactElectron33.4.11, verify fallback and positive window binding. Real companion capture still requires same-UIDlocal admitted observation lease/private config setup. Managed Xvfb helper needs its own approved session attestation; realhost positive fullapp possible only managed lifecycle. Full plan active.

## Deployed native-window identity and positive real binding — 2026-09-05T19:36:32.133094+00:00

Built API/helper; managed device-control restart healthy19:34:42Z with explicit owner/helper config. S2D generationb6d1ee13-d9f7-7772-4edd-7afcc14110e6 completed, SDA exact Electron33.4.11 install and generatedtsc/electron-builder Linux passed. Current native package uses own-window+kernelPID path; old process-cgroup check removed.

Added reproducible live-native-companion-binding.py: creates unmapped1x1Xwindow on configured real display0, obtains observation-onlyTTL30 local owner lease, sends CaptureCompanionActivation from samePythonprocess via Unix socket. XRes/owner kernelPID binding succeeded200; rootwindow request refused503; no screenshot/text/input/focuschange. Session09f18405-6cc4-4d02-901e-3f0480929bae stopped, cleanupreleasedtrue, nativewindowdestroyed, activegrants0. Receipt native-companion-binding-live.json. This is real native-client binding, NOT an Electron prior-app capture journey.

Rebuilt Electron managed session3a2eaf6c-e56d-4f63-88ea-7819a2c99ff4, hidden->palette with absentconfig/unavailablevisiblelabel, same draft/node; cleanup200. native-window-identity-fallback-live.json; packagefingerprints refreshed.

NEXT full integration: practical owner-managed observation-session setup/renewal/private binding and positive packaged prior-app capture in properly bound desktop. Existing S2D validationprofileXvfb differs from realownerdisplay0; cannot point it to reallease and claim matchingdesktop. Need managed attestation for perfixturehelper OR managed native launch on selectedrealdesktop with explicit scope. Focusrestore/contextattach/annotation still missing; all31phase scope remains active.


## Companion reference dismissal — 2026-09-05T19:44:46.859702+00:00

Explicit companion reference dismissal added to canonical native presentation/preload and localized Portal toolbar. Main-frame-authorized dismissal aborts pending capture, drops local reference, publishes a newer revision and preserves mode/focus. Late capture cannot restore it. UI serializes dismissal and reads state once after a lost reply without replaying the operation. Label explicitly describes dismissing a local app reference; helper metadata cache is NOT eagerly deleted by this action and remains bounded by its existing lifetime/authority.

Validation: native presentation25tests and Portal companion15tests passed, including pending/ready dismissal, unauthorized frame, late result, lost reply, duplicate click, unchanged draft/node and no task Stop. Native template typecheck and presentation source lint passed (preload/test files ignored by lint configuration). Portal typecheck/stringscheck/targetedlint/build passed. S2D generation a74ade65-65cc-4f03-88d4-18d0eb0406c9, SDA exactElectron33.4.11 installation, generatedtsc and Linux electron-builder passed. Portal managed restart healthy19:44:20Z, App-daY3jJft.js. Package fingerprints refreshed; this generation has NOT had a live Electron dismissal journey. Previous live receipts belong to prior generation.

Remaining: helper-side explicit deletion/cache cleanup, practical independent observation authorization/setup, full same-desktop packaged capture journey, attachment/annotation/focus restore and remaining plan phases. Full goal stays active.


## Helper context lifecycle erasure — 2026-09-05T19:48:43.806271+00:00

Helper context lifecycle erasure deployed. DesktopController.pruneActivation under existing repository exclusion removes native metadata on expiry without reads, Stop, shutdown, admitted takeover (including failed input release), and stale helper generation. Existing250ms lifecycle loop owns expiry, no new timers/goroutines. Cleanup receipts unchanged. Authorized live context survives ordinary ticks; denied Stop cannot erase it. Cache and persisted-lease comparison uses existing sameDesktopLease/time.Equal; regression caught raw time.Time equality losing monotonic/location representation through SQLite. ReadActivation also accepts equivalent serialized lease timestamps.

Evidence: targeted race tests TestDesktop(Activation|DeniedStop|Expiry|HelperRestart) passed; cleanup/revocation/Stop/takeover/shutdown regression selection passed. New lifecycle test covers7transitions plus deniedStop and serialized-expiry read. API/helper built; managed device-control restart healthy2026-09-05T19:48:10Z. Configuration docs updated. This validates erasure in actual controller+SQLite tests, not introspection of deployed helper memory.

NEXT: explicit exact-reference deletion owner/helper RPC and native adapter integration; pending capture deletion needs cancellation fencing so delayed publication cannot resurrect it. Current UI Dismiss remains shell-only. Practical independent observation authority and positive same-desktop packaged capture still pending, plus remaining broad plan. Full goal active.


## Exact context deletion and native dismissal binding — 2026-09-05T19:54:18.210553+00:00

Exact context deletion wired end-to-end. Proto DesktopHelper/Owner/Account DeleteActivation reuses exact lease/session+contextID request and empty StopResponse. Helper canonical nonzeroUUID/currentobserve admission, repository exclusion, exact cached lease+ID erase. Absent/olderID succeeds without affecting newercontext. Owner resolves actor/session and forwards signedhelpercredential, returns known empty response. Domain negativeactor/revocation/invalidID/stoppedlease tests and realowner/helperUnix sequence pass under race detector.

Canonical native ActivationReference carries optional trusted discard closure bound to cloned original socket/session+ID. Only safe ID/expiry projected into IPC (function never sent). Dismiss cancels pendingcapture, clearslocalref andpublishes immediately, awaits exacthelperdelete; failure rejected so UI existingreadback/error preserves localdismissal withoutretry. Latecapture that delivers a reference after abort invokes its exactdiscard closure; if HTTPabort/lostreply prevents ever receivingID, lifecycle TTL still bounds retention, immediate erasure NOT proven. Tests28native/presentation+adapter pass, typecheck/lintpass. Currentbindingrenewal cannotredirectdeletion. CLI delete-activation command not yet added.

Deployed API/helper built and managed device-control restarthealthy19:53:27Z. Extended existinglive-native-companion-binding.py fixture: observation-onlysession340260bc-b93d-45e1-ba62-4f6d76413b59, nativeunmappedwindow, olddelete200/newread200/currentdelete200/deletedread503, Stopcleanupreleasedtrue/windowdestroyed. Receipt native-companion-binding-live.json. Domain test proves cache erased; live receipt confirms exact endpoint behavior, not remote memory introspection and NOT Electronjourney.

S2D generation61fd76d2-b309-7344-39d2-dc621d545f0e completed, SDA exactElectron33.4.11 approvedinstall, generatedtsc/Linuxelectron-builder passed, packagefingerprintsrefreshed. PortalUI unchanged thisturn. Full packaged prior-app capture+dismiss journey and practical independent observation authorization/setup remainmissing; pendingcapture deletion requires stronger requestidentity/fencing if immediate remoteerase is required. Remaining broadplanstillactive.


## Independent observation owner-session foundation — 2026-09-05T19:59:25.751140+00:00

Independent observation foundation in owner Service (source only). Inspection proved two exclusion layers: LocalDesktopAdmission.Acquire always calls Service.AcquireContext, and DesktopController.Open owns singleton DesktopState.Lease/epoch. Renewing existing companion observation lease would prevent control, so no auto-renew workaround added.

Session gains persisted ObservationOnly bool. Service.AcquireObservationContext strict0<TTL<=10min and64liveobservers/device; shares lifecycle records but not exclusivecontrolslot. Existing AcquireContext remains exclusiveamongcontrollers. Existing sessionForLease/ValidateLease rejects observationtokens; new ValidateObservationLease accepts liveobservation/control. Stop/Kill targets exactsession. Existing session schema moved to database.EnsureSchemas declaration beside interpretingservice, adding observation_only INTEGER NOT NULL DEFAULT0; no swallowedALTER. Load/list/insert preserveclassification. Legacycontrol defaultsverified. No publicendpoint or LocalDesktopAdmission switched yet; no productiondeployment/databasechange thisturn.

Tests under race pass: observation/controlcoexistence, secondcontrollerdenied, observerdirectactuationdenied/noaudit, wrongdevice, persistence/reconstruction, exactobserverkill leavescontrol/otherobserverlive, controlrelease leavesobserverlive, legacytableadditiveupgrade, TTL/capacitybounds. Existing selected lease/audit/reconstruction/directactuation/unlock/flowreuse/livesession/kill regressions passed.

NEXT must extend helper observation admission independently of controllerlease: bounded signed observe memberships, uniqueepoch/ref, currentgrantcheck, exactStop/cleanup/read-revocation proof, helperbootstrap/shutdown revocation, noReleaseHeld for observing Stop whilecontrolactive. Then switch LocalDesktopAdmission service acquire/validate bypurpose, ownerproxy Open/Stop paths and grantstatus to observer-aware methods. Keep existing control semantics and durableunknownOpen/revocation guarantees. Only then practical privatebinding/setup and packaged prior-app capture+dismiss journey. Full planactive.


## Independent helper observation admission — 2026-09-05T20:06:09.306080+00:00

Independent observation admission integrated and deployed. DesktopState now stores bounded64 observer memberships keyed by globally allocated uniqueepoch, separate from singleton controller. Observer Open checks signed grant/currentgeneration/exclusion/CAS but never ReleaseHeld or resetscontrolreceipts/flows. Globalepoch allocatesidentity; controller admission compares exactactivelease, so laterobserverepochs do not revoke control. Observer Stop interrupts only its nativecalls, removes exactmembership and persists released cleanup without nativeinputrelease. Lifecycle expires/revokes observer independently and helperbootstrap/shutdown removesmemberships. ReadRevokedCleanup recognizes liveobserver so revokedpendingOpen cannot falselyreportabsence. Existing legacy singletonobservation cleanup retained.

Owner LocalDesktopAdmission now AcquireObservationContext forcontrolfalse, AcquireContext fortrue; Active/GrantStatus use observationawarevalidator (controltokens alsoaccepted). Ordinaryinput validators stillrejectobservertokens. Prior source Session.ObservationOnly schema change now deployed by managedrestart. Nativecontextpruning checks observermembership instead of requiring singletonlease.

Evidence: full internal/sessions Go racepackage passes (changed admission touches cleanup/flow/revocation), targeted internal/control desktop+observation tests pass. Domain tests Stop/expiry/revokedobserverleavecontrollerinputuntouched; realUnixowner/helper signedgrant/publication test openscontrol+observer, Observe succeeds, observerStop producesreleasedcleanup and zeroReleaseHeldcalls, controllerObserve remainsvalid, controllerStop releasesonce. Existing test requiring observationtakeover changed to ordinary nonexclusiveOpen (desiredbehavior); no weakenedinput assertions.

Built API/helper; manageddevice-controlrestart healthy2026-09-05T20:05:08Z. Deployed observation-only nativebinding/capture/deletion fixture passed20:05:21 session334a898e-6b32-4b8b-b336-8fb790ce8f57, rootwindownegative, olderdeletepreservesnewer, currentdeleteunreadable, Stopcleanupreleasedtrue/windowdestroyed. Safe state read confirms {"active_grants": 0, "helper_observers": 0, "helper_controllers": 0}. Receipt native-companion-binding-live.json. No fullElectronjourney or livecontrollerinput claim.

NEXT: practical companion owner-observation setup/privatebinding renewal/recovery now possible withoutoccupyingcontrolslot. Need explicitconsent/sessionlease lifecycle, noambientcapture, Stoponquit/readback, same-desktop packaged prior-appcapture+dismiss. PendingcapturelostID immediateerasure stillTTLbounded; CLIdeleteparity remains. Broadplanstillactive.


## On-demand native observation setup — 2026-09-05T20:12:16.357217+00:00

Native on-demand observation setup implemented in canonical activation-owner.ts. Private config supports version2 {version:2,socket,surface} instead of requiring a manually opened SessionRef. Same regular-file/ownUID/no group-other/O_NOFOLLOW/16KiB restrictions. admittedLocalActivationCapture validates explicitdestination/nativewindow beforeauthority; eachshortcut activation makes freshUUID Open controlfalseTTL30, validates returned exactsurface/session/expiry, then native-window CaptureCompanionActivation beforefocus. No startupOpen/backgroundcapture/renewal/accountfallback. Existing explicitsessionbinding remains supported.

Shared boundedUnixownerCall used by capture/admission/cleanup. A successful reference holds trusted discard closure bound to originalsession; DeleteActivation thenStop/ReadCleanup. Failedcapture stopsknownsession. LostOpen calls ReconcileOpen withsameoriginalrequestID once, thenStop/readcleanup ifforwardingsessionknown. Never retriesOpen. Stoplostreply may be confirmed byexactreleasedcleanupreceipt; pendingorwrongcleanupthrows. Unreachableowner/abortbeforeID still relies on30sexpiry; this is not proof ofimmediatecleanup. Capturebudget750ms remains in presentation; cleanup mayfinishafterpaletteopens, independentlybounded. Rawcredentials/nativewindowIDs remainmainonly.

Tests32native/presentation+adapter passed; finaladapter6tests aftercleanupvalidationeditpassed; TypeScriptpassed; lint0errors/1redundantcastwarning(line90). RealUnixfixturecoversversion2privatefile, successOpenCaptureDeleteStopReadCleanup, lostOpenReconcile(noReplay), capturefailurecleanup, pendingcleanupnotconfirmed. Configuration docsshowversion2 setup andlimits. FullpositiveElectronjourney stillpending.

S2D generationaefb30ed-297e-bc3d-78f7-6619b5170be6 complete, SDA exactElectron33.4.11 approvedinstall, generatedtsc andLinux electron-builderpassed. Packagefingerprintsrefreshed. NoPortalUI/backenddeploymentneededthisturn; no liveElectron sessionlaunched. Prior backend observation+capture/deletionliveevidence remains separate.

NEXT: managedsame-desktop packagedprior-app capturejourney usingprivateversion2source; currentisolatedXvfbvalidationprofiledoesnotmatchrealhosthelperdisplay0, so either legitimatehelperattestationforfixture or managedrealdesktoplaunchneeded. SetupUI/CLI usability beyonddocumentedprivateconfig remains; no globalconsentGUIyet. Focusrestore/attachment/annotation, lostIDimmediatedeletion, CLIdeleteparity andbroadplanremainactive.


## Selected-desktop packaged context journey — 2026-09-05T20:22:04.769568+00:00

Positive packaged Electron context journey on selected real desktop completed. Reused S2D target.StartElectronSession managed test owner (process/CDP/temporaryprofile/exactartifactidentity/Closecleanup), rather than changing isolatedlivedesktopHTTP service to exposehostdesktop or weakeninghelperattestation. Added opt-in TestStartPortalCompanionContextOnSelectedDesktop in target/session_test.go, plus live-native-context.cjs checker. Repro command: python3 scenarios/portal/docs/internal/plans/artifacts/portal-everywhere-20260905/live-native-session.py --selected-desktop-context. Runner reads protectedhelperconfig for DISPLAY/Xauthorityfilename and creates mode0600temporaryversion2source, no cookiesprinted.

Test owns xmessage fixturewindow, identifies exactWM_STATEclientwindow, sets its correctfixtureprocessPID metadata (xmessage lacksproperty), confirmsfixtureactivebeforeglobalchord. App uses samehelperdesktop and trustednativewindow/XRes/kernelPID binding. UI setsandconfirmsControlAltP shortcut, hidden->palette, contextready, Dismiss, waits pendingfalse/no toolbarerror (exactcleanupread), same draftDOMnode/text, clearsdraft. No screenshot/textselection/chatSubmit/nativeactuation request. Go deferredmanagedClose plusfixtureKill/Wait cleanalltestprocesses/profile.

Initial fixturedebugging: name search also matchedframe; restrictedWM_STATE. xdotool set_window lacks--pid; corrected to xprop32c onownedfixture. Final reproduciblerunner passes1.85sec at2026-09-05T20:21:09Z, context284dd7c2-0717-450f-b5c4-ebd894d845e1. Receipt native-context-selected-desktop-live.json: prior_app_focusedtrue, afteractivationready, dismissalunavailable, cleanup_confirmed_by_ui true, samedraftnode; postrun activegrants0/observers0/controllers0. Currentpackage generationaefb30ed-297e-bc3d-78f7-6619b5170be6 tested unchanged.

Scope limitation: target adapter's existing Linuxvalidationlaunch uses --no-sandbox. Receipt explicitly marks validation_only_unsandboxedtrue. This proves functional packagedintegration, NOT release sandboxsecurity/productionhardening. Fixturewasforegroundbeforeactivation plus unit/nativeordering evidence; rawhelperwindowcontext remainsopaque, notdumped. No selectedtext/annotation/focusrestore/attachment proof.

NEXT: user-facing contextsource setup consent/UI, contextcapsule attachment/annotation/selection ambiguity, focusrestore, native releasehardening, pendinglostIDimmediatedeletion and CLIdeleteparity; continue full31phaseintent. Do not re-open solved same-desktop testblocker. Broader planactive.


## Captured client-window bounds — 2026-09-05T20:28:41.385037+00:00

Captured source-window geometry added before context attachment work. Current reference previously carried rootdisplayrevision/pointer only; missingwindowbounds would make crop/annotation transforms guess. DesktopBounds(X,Y int32,Width,Height uint32) added to nativeactivationcontext and opaqueActivationReference. Positive dimensions<=65535 and int32rootextent checks; no pixelsallocated. Proto source_bounds field8 and DesktopBounds message generatedcanonically (device-control+Portal). Helper projects knownbounds; owner requiresvalidboundsandcopiesknownfields. Cached read preserves originalbounds, never recalculates currentwindowposition.

X11 capture reads mappedclient GetGeometry and TranslateCoordinates(window->root,0,0), notparent-relativeGeometryX/Y. Samplesboundsbefore/after pointer/activewindow; changedboundsdeny. Clientcontent pixels excludeWMframe, noDIPclaim. NativeIDs/PIDs stillhelperonly. Tests isolatedXvfb source20,30/200x100, reparentunderframe100,120->root120,150, moveframe-negative->root-20,0; nofocus/inputchange. Domainmissingbounds andowner missing/zero/oversize/overflow refused. Existingcapturefixturesupdatedtoprovidevalidnativeevidence; targeted x11/sessions/control Go race tests pass.

Built API/helper; manageddevice-controlrestart healthy2026-09-05T20:27:50Z. Extendednativebindingfixture validatespositiveboundsandreadroundtrip. Deployedfixturepassed20:28:02, session0a5539d5-fb81-4ac6-917c-2271870bce3f, sourcebounds1920x1080(root0,0), olddeletepreservesnewer/currentdeleteunreadable, Stopcleanupreleasedtrue/windowdestroyed. Receipt native-companion-binding-live.json. ExistingElectronpackage unchanged; nofullElectronrerunneededforadditivebackendmetadata.

NEXT: contextcapsule snapshot/crop/annotation source model and bounded artifact ownership, then explicitattachtochat/agent request. CurrentnativebridgeprojectsID/expiry only; no boundsUI yet and no screenshotartifact/attachment capabilityclaim. Currentephemeralmetadata30seclease is insufficient for durableattachment; do notfake attachmentbyserializingan unusableID into chat text. Need scopedartifact retention/read/delete independentofinputauthority. SetupUI/focusrestore/pointerambiguity/releasehardening andbroaderplan remainactive.


## Selected-window image primitive — 2026-09-05T20:37:16.752553+00:00

Added optional X11 CaptureActivationImage native primitive. Existing CaptureActivation remains metadata-only. Pixel capture shares the bounded native operation and pre/post active-window, bounds, geometry, and session checks. It reads an existing X Composite named backing pixmap, excludes the client border, validates the root visual/depth and exact data dimensions, caps images at 16 million pixels, and frees the pixmap reference. It never redirects a production window or falls back to a root screenshot. Unsupported visuals, absent backing storage, permission loss, or changed/unmapped windows return no image.

Protocol basis: https://xorg.freedesktop.org/archive/X11R7.5/doc/compositeproto/compositeproto.txt (Composite backing storage includes borders and is independent of sibling clipping; naming requires redirected visible storage). Test fixture alone redirects its owned window.

Targeted race tests passed in internal/native/x11, internal/sessions, and internal/desktophelper. Verbose output confirms isolated Xvfb image test executed (not skipped): selected red client overlapped by green sibling produces all red content, blue border excluded, focus unchanged, repaint creates a new blue capture while prior capture remains red. No compositor refuses image while metadata works; initial and mid-operation permission denial, unmap, and allocation bound return no image. Existing activation authority, cancellation, cache deletion/expiry, and semantic tests passed. XGB emits existing closed-connection messages during fixture teardown; tests terminate successfully.

Source-only scope: no API/helper deployment, no live private pixels captured, no wire/API opt-in, cache retention, semantic forwarding, preview, or attachment added. Next integrate explicit image opt-in through native observer/controller with bounded helper-local ownership and cleanup, then scoped artifact transfer for durable attachments independent of input authority. Do not silently enable screenshots on existing metadata activation. Broader plan remains active.


## Authorized image context cache — 2026-09-05T20:40:55.955081+00:00

Connected the optional native image primitive to DesktopController context ownership. CaptureCompanionActivationImage is an explicit method, sharing current observation authority, bounded native cancellation, and companion window/PID verification before and after capture. Ordinary capture methods remain metadata-only. DesktopActivationImageObserver is forwarded through semanticBackend to the native pixel backend without accessibility enumeration.

The existing single-entry ephemeral cache now owns an optional RGBA image and HasImage flag. Validate positive source bounds, exact zero-origin image dimensions, stride and accessible byte length, and the 16-million-pixel cap before allocating. Copy on cache insertion and authorized read; backend repaint and consumer mutation cannot change cached pixels. ReadActivationImage uses the same exact lease/actor/context ID and expiry checks, rechecks authority/expiry after copying, never re-captures or renews. Existing delete, replacement, expiry, Stop, and lifecycle pruning release the entire cache entry; pixels are not serialized into receipts or desktop state. This is memory ownership removal, not a secure-memory-overwrite claim.

Targeted race tests passed: go test -race ./internal/sessions ./internal/desktophelper ./internal/native/x11 -run 'Test.*Activation|TestCompanionCapture|TestSemantic' -count=1. Added desired-behavior tests for explicit opt-in, metadata image refusal, both ownership checks, frozen backend/consumer copies, exact deletion, wrong actor, denied read, lifecycle expiry without read, Stop, nil/dimension/stride/buffer/oversize refusal, and revocation during capture. Existing metadata authority, cancellation, lifecycle, ownership and native image tests remain passing.

Source integration only: no wire/API image endpoint or native UI opt-in yet and no service deployment this turn. Next extend CompanionActivationRequest with explicit image selection, return HasImage in the bounded reference, expose authorized bounded PNG read with post-encode reference/authority check, and test the real owner/helper Unix transport. Then native preview/retention/crop/annotation and actual request attachment remain required. Thirty-second cache is staging, not durable attachment retention. Full plan remains active.


## Explicit context image API — 2026-09-05T20:44:55.092448+00:00

Exposed optional image staging through the generated desktop contract. CompanionActivationRequest include_image field4 and local-owner counterpart field3 default false; ActivationReference has_image field9; ActivationImage contains reference and PNG. ReadActivationImage added to helper, owner and account services. Proto generation published device-control and Portal consumers. Existing companion requests remain metadata-only; native shell not yet changed.

Helper dispatches explicit image capture through the domain ownership checks. Read clones authorized cached pixels, encodes PNG with the existing 32 MiB bounded buffer, then rereads the exact context before returning to catch deletion/expiry during encoding. Owner validates reference/ID/expiry/image flag, PNG header dimensions and 16-million-pixel/32-MiB bounds, then repeats owner/helper exact-reference read after transfer. Projection contains known reference fields and PNG only. Owner PNG validation checks header/dimensions, not full decode; helper owns PNG encoding. Account read uses existing account admission; companion capture remains local kernel-PID only.

Tests: targeted race sessions/control activation and owner proxy passed. Existing real Unix owner/helper signed-grant fixture now exercises metadata-only image refusal, explicit image capture, decoded expected pixel, exact reference equality, wrong session denial and deletion unreadability. Added image reply validation cases for missing image flag/reference, malformed header, wrong dimensions and expired reference; race pass. API/helper built successfully.

Managed device-control restart healthy 2026-09-05T20:44:21Z, API16465/UI20698. Deployed metadata binding fixture passed20:44:25, session e4ddcdb1-605b-46d7-884f-04aeaf9829c6; old deletion preserves successor, current deletion unreadable, Stop cleanup released and native fixture destroyed. No live private image was captured. Positive image transport evidence uses controlled fixture native pixels, with separate isolated X11 backing-pixmap tests from prior turn. Configuration docs updated.

NEXT: native explicit image configuration/preview and context artifact transfer; durable attachment ownership independent of temporary observation lease, crop/annotation/source transforms, setup consent, focus restoration and broader plan remain incomplete. Current native ownerCall response limit16KiB/700ms needs a separate bounded image read path, not a global relaxation. Preserve metadata fallback and do not leak image bytes through presentation snapshots. Full plan active.


## Native image preview bridge — 2026-09-05T20:49:01.530444+00:00

Canonical native template now supports includeImage boolean in private explicit bindings and version2 sources. Validation happens before Open. Explicit captures send includeImage true and require hasImage. A main-only readImage closure binds original socket/session/context/time/bounds; reads do not recapture. Dedicated ReadActivationImage transport limit45MiB JSON/2sec; ordinary calls retain16KiB/700ms. Validate bounded base64, 32MiB decoded PNG, signature/IHDR/dimensions, source bounds and original reference identity. Native returns bounded data URL with original source geometry. PNG header validation relies on trusted helper encoding, not full decoder validation.

Presentation retains read closure privately and publishes only hasImage with ID/expiry. New read-context-image IPC and preload readContextImage authorize current main frame before/after read, allow one pending image read, reject expiry, dismissal or replacement while pending. Image bytes never enter snapshots. Main has no persistent pixel cache. Capture still uses existing750ms bound and opens chat unavailable on failure; image opt-in not silently enabled by default.

34 native adapter/presentation tests passed, including real Unix selected image read, >16KiB image response envelope, invalid opt-in before Open, substituted ID/bounds/malformed PNG rejection, no recapture, successful main preview, iframe denial, concurrent read refusal and dismissal before response. Typecheck passes. Source ESLint0errors, existing redundant type assertion warning1; preload/tests are ignored by current lint configuration but covered by typecheck/tests. Docs updated. No package regeneration or live UI validation this turn.

NEXT: Portal bridge/state/toolbar preview consumes readContextImage and hasImage; clear preview on context replacement/expiry/dismiss, maintain same composer, then crop/annotation transforms and actual durable attachment transfer. Regenerate/package only after UI integration to validate the complete selected-desktop journey. Existing installed package remains prior metadata-only generation. Full plan remains active.


## Portal context preview UI — 2026-09-05T20:53:32.150352+00:00

Portal UI now offers Preview captured window only for ready hasImage context and a native readContextImage bridge. Click reads once into transient component state; no automatic read, request submission, draft insertion or durable storage. Preview labels explicitly local/unattached, offers Close preview and recoverable failure; strings added in English/Japanese/Arabic and generated constants updated. Per-context component key and unmount fence discard late replies; native expiry timer removes preview, replacement invalidates it, and Dismiss clears local context immediately before waiting for cleanup. Pill view omits preview. Exact response ID/expiry, PNG data URL prefix/length, MIME and bounded geometry validated; image decode error clears preview.

19 targeted companion UI tests pass; strings check, tsc and targeted ESLint pass after fixing runtime validation types and test async flushes. Tests verify no automatic read, successful preview, expiry/replacement/dismissal erase, late dismissed response ignored, failed preview leaves chat usable and exact composer node/draft preserved. UI build passed (App-DSAFCAig.js).

Generated canonical native template into Portal using pipeline3cf2b418-f4b0-ce00-00ed-1d07a3dd3b41, completed20:51:50Z. SDA installed approved development Electron33.4.11; generated tsc and electron-builder Linux package passed. Managed Portal restart healthy20:52:33Z API17476/UI24965. Existing selected-desktop packaged metadata activation/dismissal regression passed20:52:48Z (2.71sec), same draft node and cleanup confirmed, context00b69873-f6a4-4a68-86f3-ad9ef080815b. No image captured by that journey. Package receipt fingerprints updated. Existing launcher unsandboxed; not release validation.

NEXT: positive packaged image-preview journey using an owned source-window fixture and explicit includeImage. Real desktop compositor availability must be tested; if fixture must redirect backing storage, only its own test window may be redirected, production capture never changes compositing. Verify rendered natural image dimensions and removal after dismissal, with no private screenshot artifact output. Then cropping/annotations/transforms, durable attachment ownership/transfer and request integration, setup UX, focus restoration and full plan remain active.


## Packaged image-preview acceptance — 2026-09-05T20:56:02.103177+00:00

Completed positive packaged image-preview journey using current generation3cf2b418-f4b0-ce00-00ed-1d07a3dd3b41. Existing live-native-session.py accepts --selected-desktop-context --image, creates a private includeImage source and passes image opt-in to managed Go target integration. Existing live-native-context.cjs extends the owned xmessage fixture: exact client selected by unique title/WM_STATE, correct PID property, dedicated test-only X Composite automatic redirection for that window through a child process. Redirection connection is held until test cleanup, then unredirected/closed; production capture never redirects any window. Only test-owned source pixels captured; no image bytes persisted in evidence.

Packaged test passed2.35sec at2026-09-05T20:54:49Z. Fixture is foreground before registered shortcut, Portal hidden->palette capture ready hasImage true, context7baaff92-7c05-4ac8-b753-9f572ce4cfca. Click Preview captured window; browser decodes image natural261x54, matching current test client geometry. Local/unattached notice visible. Dismiss removes image and reference; pending false and no toolbar error confirms native Delete/Stop/ReadCleanup path. Exact composer node/text preserved then draft cleared. Managed target Close and owned fixture teardown complete. Read-only helper state after run: observers0, controllerfalse. Receipt native-context-image-selected-desktop-live.json; package receipt linked.

Scope: owned fixture deliberately provides Composite backing, so evidence does not assert every host compositor/app visual is supported. Existing Linux launch uses --no-sandbox, flagged validation_only_unsandboxed; not release security. No selected text, annotation, chat submission or persistent attachment validation. Previous turn classified progress: UI/package changed and tests passed; this turn closes the previously missing actual packaged image evidence.

Next Phase22 intent: region selection and annotation tied to immutable original source geometry, bounded artifact retention and actual request attachment. Preserve source references through compact/expanded states, explicit removal/recapture, and protect against old screenshot coordinates authorizing new actions. Current temporary30sec image cache and preview are staging only; durable artifacts/request capsules still missing. Full31phase plan remains active.


## Source-bound region and annotation UI — 2026-09-05T21:01:03.558136+00:00

Added region selection and freehand annotation overlay to Portal context preview. SVG viewBox uses frozen source image dimensions; pointer CSS coordinates map into that local pixel grid, independent of desktop origin or later window movement. Drag selects an integer bounded region; numeric x/y/width/height inputs provide keyboard editing and reject outside-image extents. Draw stores at most32 strokes with256 points each; reset clears region/marks, pointer cancellation discards unfinished gesture. Overlay strokes use constant visual width. All edits remain local to the exact reference, no input authority or attachment implied.

Preview now remains mounted but hidden in pill mode, preserving region and strokes across compact/expanded transitions. Source-ID/expiry key, expiry timer, replacement and immediate dismiss still clear all pixels/edits. English/Japanese/Arabic editor labels added. No cropped image artifact or durable capsule produced yet.

21 targeted UI tests pass, with explicit source coordinate mapping at two CSS scales, portrait shape, clipped coordinates, zero layout refusal, negative-origin source fixture, drag region, canceled gesture, bounded numeric edits, drawn source points, pill/expanded persistence and reset. Typecheck, strings check, targeted ESLint and UI build pass. Deployed Portal managed restart healthy2026-09-05T21:00:12Z, App-4qUzKIDi.js. Existing native package unchanged (UI supplied by configured Portal server).

Extended actual packaged selected-desktop --image journey to edit source region and draw via Playwright pointer input on the overlay, switch pill/expanded and compare exact source point strings and region attributes. Passed2.48sec at21:00:21Z, context769418a6-f728-4e1e-849b-4a8de8e82292, source261x54, stroke_count1, source_points_preserved and region_preserved true. Dismiss removes image and confirms cleanup; composer same node/draft. Receipt native-context-image-selected-desktop-live.json. Fixture-only compositing and unsandboxed validation launcher caveats remain; no raw image saved or chat submitted.

NEXT: lift edits into a bounded immutable context capsule with original image/transform provenance, persist via an authoritative attachment owner independent of temporary observation lease, and integrate actual chat/agent request attachment/removal. Thirty-second preview expiry still limits editing and is not final retention UX. Explicit recapture/setup UX, optional selected text, protected-field policy, focus restore, release/cross-platform and remaining broadplan requirements remain incomplete. Full plan active.


## Context capsule import boundary — 2026-09-05T21:05:30.287226+00:00

Investigated existing artifact ownership: Portal chat repository currently owns search attachments but no image attachment domain. api-core/blobstore provides reusable opaque byte storage; image-tools uses that substrate rather than being a generic private capture owner. Chose Portal context ownership/metadata plus shared blobstore substrate, avoiding an unrelated image operation service as attachment authority. Existing message handlers have no new context access boundary yet; API wiring must establish authenticated owner scope.

Added internal/contextcapture/capsule.go import preparation: source exact SurfaceRef, capture UUID/display/geometry/timestamp/bounds, crop region and source-coordinate strokes. Owner passed by calling service, explicit retention>0<=24h independent of input lease. Input source freshness<=30sec, timestamp/future rejection, image max16million pixels and32MiB, bounded dimensions/int32 origins, overflow-safe crop extents, max32strokes/256points, finite within-image points. Validate PNG config before full decode, full decode rejects corrupt input, canonical re-encode strips ancillary/trailing payloads while retaining full original pixels separate from crop/marks. SHA256 covers normalized original; output ID UUID and expiry explicit; caller byte/stroke buffers not aliased. Source claims are provenance only, not verified native capture/input authority.

Targeted Go race package tests pass. Cases cover negative origins, preserved original raster despite crop, exact source metadata/annotation copy, trailing payload stripping, stable image digest, independent IDs, caller mutation, missing owner/surface/ID, future/stale source, excessive bounds/retention/strokes/points, region overflow/emptyregion, nonfinite/outside coordinates, wrong dimensions and truncated PNG. DOMAINS.md documents actual implemented boundary and not-yet-wired scope.

Source-only primitive this turn: no durable write, attachment API, service deployment or model submission. Next add context repository metadata using per-domain declarative schema and shared blobstore with bounded ownership/read/delete/expiry cleanup. Need deterministic compensation for partial blob/metadata writes, count/byte quotas, exact chat association and authenticated access. Then native transfer must carry source provenance, region/strokes into attachment preparation and chat/agent transport must consume matching rendered context; current previews still expire30sec and are not durable attachments. Full plan active.


## Durable context repository and blob service — 2026-09-05T21:10:04.381949+00:00

Implemented durable context domain repository and blob-backed service. contextcapture.Schema declaratively owns context_capsules metadata with staging/ready states, document JSON, byte reservation, owner and expiry indices. Repository interface hides SQLite. Reserve commits metadata before any blob write and counts all rows (including expired/staging) until cleanup: owner64docs/256MiB, global256docs/1GiB, document JSON512KiB. Write exclusion acquired before quota checks. WithRecord validates exact owner/ID and holds transaction write exclusion during blob lifecycle callback plus publish/remove, preventing a delete from racing an in-flight Put. Immutable document JSON is not changed by callbacks.

Service uses existing api-core/blobstore; no private filesystem storage engine. Import normalizes then reserves, writes under exclusion, marks ready only after success and fresh expiry. Failure runs independent bounded cleanup; failed cleanup preserves reservation for later Reap. Crash after reservation/Put leaves tracked unreadable staging. Read requires ready, current owner and expiry, bounded bytes, MIME/size/SHA256 match, post-read expiry/context checks; returned pixels are copied by read. Delete removes metadata only after blob deletion; Reap processes bounded expired staging/ready records and retains failed deletion rows. Expired access is denied even before physical cleanup. Database write exclusion spans bounded blob I/O intentionally for correctness; throughput optimization can preserve this lifecycle contract later.

Targeted full contextcapture Go race package passes. Durable filesystem+SQLite adapter reconstruction returns exact document/pixels. Wrong owner read/delete denied, expiry rejects before reap, successful reap removes blob+row. Injected lost Put reply plus failed Delete leaves staged tracked pixels unreadable; subsequent expired cleanup succeeds. Simulated crash after reserved blob cannot be read until publication; corrupt bytes fail digest. Quotas count reservations, failed cleanup rollback retains usage, removal frees it. Existing normalization/geometry tests remain passing. Domain docs updated.

Source/domain integration only: no API handler, authenticated identity binding, startup schema registration, scheduled Reap worker or attachment/message UI wiring yet. No service restart. Next connect domain module via authoritative Portal actor access (never trust JSON owner), private storage placement, owned chat/request references, idempotent import intent, and scheduler. Then native transfer must include source provenance and edits, prepare rendered crop/annotation derivative, and actual LLM/agent requests must use the matching attachment. Shared original pixels and metadata should not enter durable learning records. Full plan active.


## Authenticated context API module — 2026-09-05T21:14:10.243466+00:00

Added typed contextcapture proto contract and handler module. ContextCaptureService Import/Read/Delete use explicit Source(SurfaceRef,captureID,display,geometry,timestamp,bounds), region, bounded strokes, PNG and retentionSeconds. Document projects opaqueID/source/region/strokes/hash/times without owner credentials. Proto generation completed for Portal and shared imports. Source metadata remains declared provenance, not native proof or input authority.

Handler reuses api-core/owneridentity.Validator; Authorization Bearer required, subject nonempty/currentexpiry, no anonymous/local-account fallback and no request owner field. Subject passed as service ownership scope. Read revalidates identity and document expiry after blob access. Generic errors do not reveal raw storage errors or private details. Module caps request45MiB and sets Cache-Control no-store; fromWire validates required nested objects/timestamps/target plus bounds before domain preparation, known-field response projection. Missing/wrong-account context returnsnotfound. Identity provider failure currently surfaces unauthenticated with generic operator unavailable message, no authority fallback.

Real httptest Connect module integration with SQLite and blobstore passes under race: unauthenticated Import no rows, authenticated roundtrip retains digest/pixels, no-store response, Bob Read/Delete denied, invalidtoken denied, AliceDelete then unreadable. Context domain race package also passes. Documentation updated. Current production Portal remains unchanged; module is not registered in main/staticregistry and no new service deployment this turn.

Next: runtime module/schema/endpoint registration, private storage path through storage.Resolver and scheduled bounded Reap with cancellation/shutdown; avoid testing DB routes writing untracked production blobs. Then native explicit transfer must include original provenance and edits, import retry/reconciliation must have a durable intent, and chat/agent attachments must consume the matching rendered crop/annotations. Account identity UI already exists via OperatorSessionService; reuse its token ownership, do not invent a separate attachment account. Full plan active.


## Context runtime and expiry cleanup — 2026-09-05T21:19:53.262156+00:00

Registered contextcapture module in main and static schema/endpoint/proto registries. Runtime resolves portal/contextcapture/blobs through api-core storage.Resolver, creates/chmods root0700 and uses shared filesystem blobstore. Production metadata binds Primary pool. Startup cleanup worker uses schedule.Clock: immediate Reap64, then everyminute, each cycle5sec deadline; errors log generic retry status, cancellation joins before DBclose.

Routed tests bind PoolForContext (strict live test lease, no primary fallback) and a cached handler/service for that exact *sql.DB with separate shared MemoryBlobStore. New test pool replaces cache; worker discards stale cache and reaps active test entries. Production blobs are never selected for test requests. Runtime routing source implemented; new focused tests cover cleanup scheduling/cancellation, while full test-pool/blob routing positive integration remains to be strengthened. Test cache intentionally ephemeral across runtime restart/pool replacement.

Endpoint generation first reported missing ContextCaptureService CLI coverage. Added explicit temporary omitted entries for Import/Read/Delete stating CLI support is pending under this plan, not final parity. No false binding or fake implemented CLI. Regenerated endpoints successfully. Targeted handlers/contextcapture, internal/contextcapture and internal/modules race tests pass; fake clock proves startup cleanup, failure report, scheduled retry and cancellation of a blocked cleanup call. APIbuild succeeds.

Managed Portal restart healthy2026-09-05T21:18:54Z API17476/UI24965. Deployed Import without bearer returns401 unauthenticated with Cache-Control no-store; receipt context-api-runtime-live.json at21:19:19Z. No real account image import, rawpixel output or chat submission in this live check. Existing fixture HTTP auth roundtrip tests remain prior positive evidence. No full suite run.

NEXT: remove temporary CLI omissions by implementing explicit credential-file and request-file commands; durable import idempotency/reconciliation before UI upload, native source/region/stroke transfer and authenticated operator UI binding. Then generate/retain crop+annotation derivative and attach exact references to chat/agent requests. Strengthen runtime test-pool isolation acceptance and full account runtime import/delete before claiming complete attachment flow. Broader plan remains active.


## Context CLI parity — 2026-09-05T21:24:48.564059+00:00

Implemented portal context import/read/delete CLI group with manifest bindings to all3ContextCaptureService RPCs; removed temporary omissions. Domain registration/aggregator test updated; endpoint generation now passes with real CLI coverage. Commands require explicit --access-token-file; import --request private protobufJSON, read --id/--output, delete --id. Uses explicit authenticated client without cli-core account-token override and rejects redirects. Input regular/private checks, bounded reads, inode consistency; no token values in errors. Unknown JSON fields rejected. No automatic retry or account fallback.

Read checks returned exactID/currentexpiry/32MiBbound/SHA256 before creating mode0600 O_EXCL output, refuses existing paths, syncs/closes and removes partial owned output after write failure. JSON/human rendering receives Document only, never ReadResponse PNG. Human output includes ID/hash. Exported file retention belongs to caller, documented separately from Portal retained deletion. Operator token/header is not printed. Import remains non-idempotent pending next step.

Targeted CLI race tests pass in domains/contextcapture, domains and root. Actual httptest Connect client checks explicit Bearer, import/read/delete calls, metadata-only result type, privateoutput and overwrite refusal, invalidpermissions/symlinktoken refusal without APIcall. Initial compile mismatch MutationReport Summary fixed to existing Result API. Endpoint generation pass. Installed portal context --help auto-rebuilt currentCLI fingerprint70d8c9b722312438e113d4134767513627c4498541b8e6d23819f6d8833c50d4; read help exposes token/id/output. Docs updated. No production account upload or raw privatepixel output this turn.

NEXT: durable import intent idempotency/reconciliation for lost replies, then native source/provenance and selected edit transfer to context API using existing operator session. Render matching cropped/annotated derivative and connect retained reference to real chat/agent submission. CLI live authenticated account roundtrip and stronger runtime test-pool acceptance still required as integrated validation. Full plan active.


## Durable context import intents — 2026-09-05T21:29:32.542475+00:00

Added caller-supplied requestID to domain Import and proto ImportRequest field6, Document requestID field8. Missing IDs retain legacy new-intent behavior; explicit IDs must be canonical nonzeroUUID. Normalized image hash plus canonicalSource(UTC capture time)/region/strokes/retention form ImportDigest. Document carries requestID/digest internally; API projects requestID without digest/owner.

Declarative context_import_intents table scoped(owner,requestID) reserves originaldocumentID/digest/createdtime and24hreceipt expiry in same transaction as capsule reservation. Receipt quota8192global/1024owner independent of capsulequota. Duplicate intent rejected before quota/insert. Service looks up knownintent to validate retry against original admission time, so a successful retry can resolve after capture30secfreshness without extending retention. Matchingdigest reads originalreadydocument; altereddata/retention conflicts; staging/failed/deleted/expiredunavailable and never rewrite/recreate. CapsuleDelete leaves digestreceipt. Reap removes expiredreceipts only after referenced capsulemetadata gone, retaining failedcleanup tracking. No rawpixel/sourceJSON in receipts.

Race tests domain+HTTP pass. New reconstruction/advance1minute test reuses sameID/expiry; injected Put failure after firstimport proves retry doesnotwritepixels. Changedregion andretention conflict. Deletedartifact cannotbe recreated while receiptalive; at24h receiptprunes andoldcapture fails freshness. APIHTTP duplicateinput returnsidenticalID/expiry. Existing quota fixtures now use distinctintentIDs per reservation. Canonical digest and schema tests remain passing.

Proto regenerated; APIbuilt and managedPortalrestart healthy2026-09-05T21:28:38Z API17476/UI24965. CLI request-file decoding can carry requestId after source rebuild; docs explain explicitID requirement, noautomaticretry,24h receiptwindow and caller-owned exportretention. No liveaccountupload ormodelsubmission thisturn.

NEXT: read-only reconcileImport(requestID) returning absent/staging/ready/unavailable and originaldocument metadata without requiring image bytes to be resent; implementAPI/CLIparity. Then preserve pendingintentIDs in UI, transfer nativeSource+region/strokes with explicitoperatorauth, and attach retained renderedcontext to chat/agent requests. Existing optionalemptyrequestID is legacy compatibility only, not safe retry. NativeUI preview stilltemporary, noactualattachment. Fullplanactive.


## Read-only context import reconciliation — 2026-09-05T21:36:25.181325+00:00

Added ContextCaptureService.ReconcileImport and portal context reconcile-import --access-token-file --request-id. Exact authenticated subject scopes receipt lookup; handler revalidates owner before returning known metadata. States absent/staging/ready/unavailable; only ready includes original document. No image access, import replay or retention extension. Ready means publication, not blob integrity: Read remains required before pixel use. Missing receipts are account-scoped and bounded by existing 24h pruning, not proof of never-imported. Repository failures propagate instead of reporting absent.

Targeted API context domain+handler and CLI context+domains+root race tests pass. Domain nil blob adapter test covers absent, staging, owner isolation, ready with unchanged document after time advance, expiry, removal, malformed ID and closed database error. Real Connect HTTP test covers absent/ready/bob absent/deleted unavailable/no-store; CLI fixture verifies bearer and status request mapping. Proto and endpoint manifest generated; API build passes; installed CLI help includes command. Managed Portal restart healthy 2026-09-05T21:35:58Z on API17476/UI24965. Docs explain state semantics. No live positive account import or model attachment this turn.

NEXT: native preview currently omits Source surface/display/geometry/capturedAt; project those frozen known fields through native bridge. Lift region/strokes into parent attachment state, use existing operator auth to import explicit request UUID, persist credential-free pending UUIDs and recover through ReconcileImport. Prepare actual crop/annotation derivative and wire retained owned references to real chat/agent requests. Runtime test-pool isolation and live authenticated positive roundtrip still need validation. Full 31-phase goal remains active and incomplete.


## Native image source provenance — 2026-09-05T21:39:32.309313+00:00

Native ActivationImagePreview now projects source compatible with retained ContextCaptureService Source: bound known SurfaceRef, captureId, displayId, geometryRevision, canonical capturedAt, copied client bounds. Original source reference is matched against image read; display/geometry must be nonempty bounded strings for opted-in image capture. Known fields constructed explicitly; no session/lease credentials or window IDs projected. Returned source surface and bounds are copied per read, preventing mutation of later reads. Existing sourceBounds remains compatible with preview editor.

Native 34 focused tests pass, including source equality, returned-object mutation isolation, display/geometry substitution rejection and invalid display refusal. Typecheck passes; source lint has no errors and one existing redundant assertion warning (test files ignored by lint). Generated Portal desktop pipeline5c60c85d-9b72-2bb7-8c7c-3041ce62a389 completed21:37:51Z; SDA installed already-approved development electron33.4.11, generated tsc and electron-builder --dir --linux pass. Package fingerprints refreshed.

Managed real selected-desktop image test passed2.59sec at21:38:52Z, receipt native-context-image-selected-desktop-live.json. Explicit readContextImage validates source capture/bounds match, surface/display/geometry/time present, no unknown source fields; records only booleans, no pixels. Native captured owned xmessage client261x54 before companion focus; crop/stroke preserved through pill/expanded; dismissal removes preview and confirms cleanup; same draft node survives. Test uses fixture-owned XComposite backing redirection and existing unsandboxed launcher, so no release-hardening claim.

NEXT: Portal UI currently reconstructs preview using only old fields and discards new source. Lift validated Source and crop/stroke state into parent attachment draft; explicit existing operator auth should import with request UUID and reconcile pending IDs. Need actual rendered region/annotation derivative, account-owned message attachment association and model/agent image use. Do not place temporary context IDs into chat text. Full plan remains active/incomplete.


## Validated companion context draft — 2026-09-05T21:43:07.350319+00:00

Portal CompanionPresentation now validates and retains ContextSource rather than dropping native source projection. Exact captureId and all bounds must match preview; capturedAt must be finite/not future and consistent with30s expiry; display/geometry bounded nonempty; SurfaceRef known fields bounded and owner device-control. Output constructs only known fields and copies nested source so unknown fields/session data never retained. Source is provenance, not authority. ContextDraft groups original image+source, selected region and strokes. Editor is controlled by preview parent; pointer gesture remains local until completion, reset updates region and strokes atomically. Dismiss/expiry/close still removes transient draft.

22 focused CompanionPresentation tests pass, including missing/stale/future/mismatched source and nested alias isolation; existing scaled/negative-origin region/stroke/pill continuity tests pass. Typecheck and source+test lint pass after explicit validator narrowing; production bundle App-D6p2s_Gy.js built. Managed Portal restart healthy21:42:27Z. Real managed packaged image test passes2.71s21:42:36Z: source capture/bounds/surface/display/geometry/time checks,261x54 client, region/stroke retained across modes, preview removed and cleanup confirmed, original composer node survives. Receipt native-context-image-selected-desktop-live.json. Fixture-specific XComposite backing and unsandboxed validation limitation remain. No actual attachment import or chat submission yet.

NEXT: ContextDraft is still private to preview parent, no attachment store/APIcall. Wire explicit authenticated import using existing DesktopSessionsProvider account. AppShell nests CompanionPresentation > AgentTasksProvider > DesktopSessionsProvider; toolbar is inside desktop provider. useDesktopSession throws withoutprovider; standalone toolbar tests currently omit it, so keep injection/context boundary deliberate. Existing useDesktopSession imports CompanionPresentation for Stop; avoid spreading circular dependency. Account has id/realm/token/refresh/expires; use operatorHeaders explicitbearer with ContextCaptureService client and durable request UUID. Preserve only credential-free pending IDs for reconciliation, never pixels in localstorage. Need expiry/account-switch fencing, rendered crop/marks derivative and actual owned message/model attachments. Full plan remains active.


## Reviewed context import client — 2026-09-05T21:46:16.206305+00:00

Added UI api/contextcapture.ts typed client plus prepareContextImport, validateImportedContext and importReviewedContext. ContextDraft type exported from CompanionPresentation for type-only API input. Request builder copies source via protobuf JSON, crop and all points, decodes bounded PNG data URL, validates UUID/freshness/retention/region/marks/source bounds. Original pixels are sent; crop/marks remain separate for later derivative. No request persistence because pixels are private. Explicit bearer and AbortSignal +15sec timeout, one import call and no replay. Receipt validates request ID, canonical nonzero documentUUID, exact proto source/region/strokes, hash shape and creation/expiry matching retention. Server remains image decoder/normalizer; client does not claim to validate PNG raster or blob digest from raw original because normalization can change bytes.

Four focused API tests pass: copied reviewed payload/no aliases, expired/bad region/nonfinite marks/mismatched bounds/bad base64 rejected, substituted receipt source/region/strokes/request/expiry refused, explicitbearer/timeout/abort/no retry. Combined26 API+companion tests passed, then corrected TypeScript unchecked-index guards/test fixtures and lint literal-condition; final API4tests and typecheck pass, API lint clean. No UI production build/restart because new client not yet called from runtime flow. No liveauthenticated upload claim.

NEXT: wire client to UI through account-bearing shared shell. Need persist credential-free pending request marker (actor id/realm + UUID, no pixels/token/source) BEFORE upload and retain uncertain outcome for reconcile. User action should clearly describe one-hour retained context and distinguish retained from actually sent-to-model. Account/expiry fencing and Delete/cancel required. Current ContextDraft is preview-local and expires with native30sec capture; do not claim attachment exists. Need retained shared attachment state, rendered crop/stroke derivative and actual chat/agent owned message reference/model image delivery. Full plan active.


## Authenticated retained context UI — 2026-09-05T21:53:17.652889+00:00

Wired explicit Retain context for1hour action into native preview via ContextRetentionProvider in AppShell inside existing DesktopSessionsProvider. Account injection uses existing id/realm/token/expires, no second login store or runtime import cycle. ContextRetentionStore owns one unresolved import per store; prepares copied reviewed draft, writes version1 marker containing only actor id/realm and random requestUUID BEFORE sending pixels, then calls typed import client with explicitBearer. Successful receipt is shared shell metadata, survives closing preview; no model send occurs. UI distinguishes localpreview, pendingimport and retained-not-sent; EN/JA/AR labels generated. Retained status/recover/delete controls appear in workspace. Timer withdraws expired metadata.

Recovery reconstructs marker across reload, requires original current account, uses ReconcileImport only; staging stays unresolved, ready publishes document metadata, absent/unavailable clears marker. Delete clears only after confirmed service result; lost reply retains marker. Account change aborts in-flight call, clears visible document, fences late import/reconcile/delete even if original actor signs back in; no automatic import replay. Component unmount aborts but preserves marker. Storage write failure prevents upload.

31 focused tests pass: API4, retentionstore4, companion23. New store tests prove marker written before API and excludes token/pixels, reconstruction recovers lost import without replay then deletes, failedstorage no upload, logout/login ABA late result rejected, wrongactor no recovery call, staging/unavailable semantics. New rendered preview test proves explicitclick required, reviewedregion sent once, retainednotice replaces local-onlynotice and receipt/delete action survives closingpreview. Typecheck and lint pass. Production bundle App-CThA7jDi.js built; managed Portalrestart healthy21:52:29Z. Existing managed packaged capture/provenance/annotation/dismissal journey passes2.95sec21:52:39Z; fixture backing and unsandboxed functional-validation limitations remain. This live run did NOT authenticate/import; upload behavior currently covered by mocked frontend and prior real Connect server tests separately.

NEXT REQUIRED: singleton localStorage marker is not coordinated across browser tabs/windows; add cross-document reservation/ownership protection before claiming robust recovery (e.g. Web Locks with storage verification, or per-request marker keys). Current per-store serialization alone doesnotprove multi-window safety. Corrupt/unavailable storage gives refusal; needs user-facing recovery guidance. Improve retention copy to explicitly state original window plus edits retained (crop doesnoterase original). Add realauthenticated UI/API positive import/read/delete journey. Then render crop/stroke derivative, associate retained context with account-owned conversation/message, deliver actual image to chat/agent model, and ensure expire/delete/account fences. Full31phase goal remains active/incomplete.


## Cross-window context recovery coordination — 2026-09-05T21:57:04.268845+00:00

ContextRetentionStore now acquires same-origin Web Lock portal.context-retention.v1 with ifAvailable (no waiting or capture replay), holds across bounded operation, reloads durable marker before authorizing/mutating. Missing lock support refuses operation. Injectable lock boundary supports independent-store race tests. Provider listens to storage events when idle, withdraws stale document when marker changes. Clear rechecks exact stored request and actor before removing; cannot erase a successor marker. Account abort checks remain inside acquired lock. Retain copy EN/JA/AR explicitly says original window image and edits retained1hour.

Corrected uncertain recovery: ABSENT no longer clears marker. An absent lookup is not proof a delayed original import cannot arrive; keep unresolved until authoritative unavailable/ready result. This leaves a pending cancellation UX requirement rather than silently losing the request. UNAVAILABLE is a known receipt and can clear.

30 focused retentionstore7+companion23 tests pass; typecheck and final lint pass. New tests use two stores and a shared exclusive lock: competing retain refuses, stale secondstore reloads marker and cannot upload after first lost reply; one APIcall only. Changed successor marker cannot be erased by old unavailable response. ABSENT remains unresolved. Existing account ABA, storagefail, explicitclick and retention tests remain green. Production bundle App-Ba-YpLRj.js built; managed Portalrestart healthy21:56:03Z. Real packaged selecteddesktop journey3.18sec passed21:56:19Z with actual browser lock API available and nested overlapping acquisition refused plus nativecapture/provenance/annotation/dismissal checks. This validates lock primitive in packagedrenderer, not two real logged-in window uploads. Fixturebacking/unsandboxed limitations unchanged; no liveaccount upload.

NEXT: liveauthenticated native UI import/read/delete and actual rendered attachment delivery stillrequired. Need safe cancel/abandon of unknown import by requestUUID, with durable server cancellation receipt preventing later arrival from publishing; ABSENT now intentionally stayspending and cannot be silentlyforgotten. Do not implementclient-onlyforget that loses eventualretainedartifact. Current retention model onependingartifact per origin; broader multipleattachments can extend later while preserving eachUUID. Need owned message association and rendered crop/annotation derivative before model/agent delivery. Fullgoalactive.


## Durable context import cancellation — 2026-09-05T22:02:39.672978+00:00

Implemented cancellation by context import requestUUID through repository/service/ConnectAPI/CLI/UI. Repository CancelIntent acquires same SQLite write exclusion as Reserve/WithRecord. Unknownrequest inserts bounded24h negative receipt with emptydocumentID/digest under same8192global/1024owner receiptquota; duplicatecancel doesnotrenew. Knownrequest retains originalreceiptidentity/time, deletesblob undertransaction then removesrow; deletionfailure rollsbackmetadata and remains tracked. Staging publication after successfulcancel cannotfindrow. Unknown lateReserve seesexistingintent, Service rejectsnegativeintent beforeprepare/onreservationcollision. Service also rejectsexpiredprepareddoc beforeReserve. Existing schema supportsnegative receipt withoutnewcolumn/migration.

Connect CancelImport(ReconcileImportRequest)->DeleteResponse requiresauthoritativebearerowner. CLI context cancel-import --access-token-file --request-id addedmanifest/binding andinstalledhelpverified. UI pendingimport offers Cancel import and delete retained image EN/JA/AR, holdsoriginlock and keepsmarkeruntilserverack; lostcancelreplypreservesmarkerforidempotentretry. Knownreadyretained Delete remainsexistingflow. Docs explainnegative24hreceipt, failurecleanup andretry.

Targeted APIcontextdomain+handler andCLIcontext+domains+root race tests pass. Newdomaincasesunknowncancel/preparedbeforecancel/lateImportrefused/ownerisolation/repeatwithouttimeextension, staged+readyblobremoval, failedcleanuptrackedthenretry, postcancelPublishrefused. RealConnectHTTP unauthcanceldenied, unknowncancelthenImportdenied, existingreadycancelunreadable. CLI fakecancelboundexplicitbearer. Frontend35tests(API4,store8,companion23),typecheck,lintpass; newstorecancel lostreplythenackclears marker. Proto andendpointgen APIbuildpass.

Initialfrontendtestfailedbecauseinstalledfiledependencyproto snapshotwasstale; resolvedthrough SDA deps install npm/@vrooli/proto-types@file:../../../packages/proto/gen/typescript --scenario portal --surface ui --apply. Generatedsourcehadmethod, installedcopynowhasmethod. Re-ranallrelevantfrontendchecks andrebuiltcorrectbundle App-szB_72f2.js. ManagedPortalrestartcompletedhealthy22:02Z. No liveauthenticatedUIuploadthisturn andnoadditionallivepackagedcapturetestneededforunchangedcapturepath.

NEXT: liveauthenticatednativeUI/API import/read/cancel/delete integration stillunproven asonejourney. Need renderedcrop/annotationderivative and actualaccount-ownedchat/messageattachments/modelimagedelivery; retainedmetadataisnotyetmodelattachment. Cancellationreceipts bounded24h, preservecontextcalltimeouts/freshnesschecks; no permanentrequestIDtombstoneclaim. Need raceverification withactivelyblockedPut/Cancel inadditiontostateorderingtests ifbroadeningconcurrencyclaim. All31phasegoalactive/incomplete.


## Retained context rendered derivative — 2026-09-05T22:08:15.513886+00:00

Added on-demand derived image rendering to retained context domain. Service.Render first uses existing owned/integrity/expiry Read; renderer checks decodeddimensions and localregion, copies only selectedpixels to newRGBA, clips polyline segments with Liang-Barsky againstcrop, paints red#dc2626 withtwo-source-pixel brush. Coordinates remainoriginalrasterlocal; negativedesktoporiginneveroffsetscrop. Bounded32strokes/256points,4Mi sampledstepbudget,existing32MiBPNGoutput and16Mpixelbounds,ctxchecks duringwork. Originalblob/metadataunchanged; finalWithRecord revalidatesready/hash/expiry so deleted/expired duringrenderdoesnotreturnderivedpixels. SHA256returnedseparately. No derivative persistence orretentionextension.

Connect Render(ReferenceRequest)->RenderResponse{document,png,renderedSha256} authenticatesownerbefore/after and rechecksexpiry. CLI context render usesexplicitbearer,privateO_EXCLoutput,verifiesderivedhash and returnsmetadata-onlyRenderResponsewithoutPNG. OriginalDocument.originalSha256 remainsoriginalidentity. Endpoint/CLI manifestsregistered,generatedproto refreshed viaSDA localfiledependencyinstall toavoidstalecopiedtypes. PreviewSVGannotationwidthnowtwo-source-pixels (removednon-scalingstroke); editorSVG remainsselectioninterface, notpixelidentical antialiasedgoldenforraster. FuturefinalreviewshouldshowactualserverderivedPNG.

Domain/APIcontext andCLIcontext/domains/root racepackagespass. Patterned8x6original,4x3crop atlocal2,1withnegativeDesktoporigin verifiestranslatedoriginalrowandredcrossingannotation,excludedoutsidepath,unchangedoriginalbytes,repeatedidenticalderivedbytes/hash,wrongownerandexpiredrefusal; badregionandcancelledctxdenied. RealConnectHTTP verifies2x2originalrendered1x1cropwithseparatedigest,originalhashretained,bobdenied. CLIfixtureverifiesrenderedbytesexportedprivatelywithoutPNGinJSONmetadata. Frontend35tests/typecheckpassafterSDArefresh;UIbuildApp-CrXUMT6R.js. APIbuildandendpointgenpass;managedPortalrestarthealthy22:07:40Z;installedrenderhelpverified. No liveauthenticatedrenderoractualmodelsubmissionthisturn.

NEXT: consume Render fromretainedUI forfinalreview,thenwireaccount-ownedmessageattachmentreferencesandactualmodel/agentimageinput. Liveauthenticatednativecapture->retain->render->delete/canceljourneyremainsneeded. Need testresourcebudgetandeffectiveconcurrentdelete-during-renderifclaimingthosebroadly (currentimplementationguardsbuttestsnotexplicitrace). Multipleattachments,modelcapabilityfallback,privacytargetandcrossplatformremainingplanunchanged. Full31phasegoalactive/incomplete.


## Retained context final rendered preview — 2026-09-05T22:11:58.853081+00:00

Retained context panel now offers explicit Preview retained crop and annotations, calling authenticated Render under existing originlock. Store checks exact Document protobuf equality, boundedPNGbytes/header/IHDRdimensions matchingselectedregion, SHA256actualbytes againstseparatederiveddigest, then rechecksabort/currentaccount/exactdocument/expiry afterhashing. TemporaryblobURLcreatedonlyaftervalidation, neverstoredindurablemarker. UIrendersactualserverderivedPNG withlocalizedEN/JA/ARalttext andclosebutton; decodeerrorwithdrawspreview. No modelsend.

CentralpublishrevokesoldblobURLwhenpreviewchangesorDocumentchanges. Logout/actorchange,documentexpiry/accountcredentialexpiry,markerchange,recoveryreplacement,Delete/Cancel/close revoke. DeleteandCancelclearpreviewbefore awaitingserverackso lostreplydoesnotleaveprivatepreviewvisible. Unknownoperationmarkerremainsrecoverable. Accountconditionstillcheckedafterrender/digesttoavoidlateviewonlogout.

32 focused tests passed(store9+companion23),thenfinalstore9passedafterdelete/account-expirywithdrawaladjustment. NewrealSHA256testwith1x1PNG rejectsbadderivativehashbeforecreatingURL, acceptsvalidhash, keepshash/pixelsoutoflocalstorageandrevokesURLonactorchange. Existingrecovery/ABA/storagefail/canceltestsremainpassing. Typecheck/lintpass;productionbundleApp-PYdEIZCu.jsbuilt;managedPortalrestarthealthy22:11:23Z. Noactualauthenticatedbrowserrenderjourneyornewlivepackagedtestthisturn; GUIrequestbehaviorisimplementationplusstoretests, notfullliveacceptance.

NEXT: liveauthenticatednativecapture->retain->render->delete/cancelintegrationandactualmodelattachmentsremainneeded. Newpreviewonlyviewsretainedartifact; itdoesnotattachtochat. Needaccount-ownedmessageassociation,source-contextreferenceinchatSend/agentrequestandmodelimagepayload,withpostfetchlifecyclechecksandcapabilityrefusal. Retainedmetadata/providercurrentlynotexposedtoconsumersoutsidepanel; defineexplicituse/consumecontractratherthanglobaltokenstate. All31phasegoalactive/incomplete.


## Multimodal provider and completion boundary — 2026-09-05T22:17:41.700818+00:00

Traced actual MessageService/completion path. Existing chats/messages schema hasNOaccountownership and allGetTree/Send/Edit/Regenerate/Stream handlerscurrentlyglobal. Privateimageassociationmustextendchatownershipincludinggeneratedrepliesbeforewiringruntime; cannotjustpassowneridalongwithgloballyreadablemessages. This is nextowningboundary, notexternalblocker.

Added OpenRouter Message.Images[][]byte withcustomMarshalJSON retainingstringcontentfortextonly and producingorderedtextthenimage_url(data:image/png;base64) partsforuserimagesonly. Verifiedprimarydocumentation https://openrouter.ai/docs/guides/overview/multimodal/image-understanding. Supportsmax4images/32MiBsumperrequestcheckedbeforemarshal; eachPNGconfigmax16Mpixelandfulldecodebeforewire; noURL/pathinput. Text-onlybehaviorunchanged. PrivateimageHTTPrequests areoneattempt (no429automaticretrybecausefreshnesswouldneedrenewal), genericstatuserrorwithoutproviderbodyecho; contextcancellationpreserved. Noactualexternalproviderrequestmadethisturn.

CompletionConfig.Images MessageImageResolver interface resolvesauthenticatedliveimagesby(chatID,messageID). BuildOpenRouterRequestinvokesonlyselectedactivepathusernodes,copiesbytesintooutgoingmessages,boundsaggregatecount/size,propagatesfailurebeforesendingorwritingassistant. ProductionconfigcurrentlyImagesnil; noimageassociationavailableyet, so thisisimplementedtesteddependencyboundary, notfinishedprivatechatfeature.

Targetedopenrouter+completionracepackagespass; finalproviderretestpassaftercancellationrefinement. NewrealhttptestproviderreceivesexactPNGbase64aftertextandstreamstokens; invalidPNG/system-roleimages/aggregateoverflowneverreachHTTP;429privatebodyechoisnotreturnedandcallcount1. Completiontestresolvesonlychosenbranch(noteditedsibling),copiesbytes; unresolvedimagepreventsprovidercallandassistantwrite. Existingtextcompletiontestsremainpassing. APIbuildpassed. No managedrestartneededforcurrentlyunwirednewimagepath; noUIchange/deployment, liveaccount/modeluploadnotclaimed. DOMAINSdocumentsboundaryandremainingownershipwork.

NEXT PRIORITY: implementaccount-ownedchat/messagepersistenceandallaccesspaths (includinglist/tree/edit/regenerate/stream/agentadmissionsandderivedassistanttext) beforeprivatecontextassociation. SharedlegacytextchatcanremainexplicitlylegacybutmustnotexposeownedrowsorclaimthemfromrendererIDs. Existingapi-coreowneridentityalreadyservescontextcapture; commonprincipalcontext/authmiddlewareandrepositorystorefiltersmaybenecessary. ThenpersistopaquecontextIDsagainstownedusermessage,configureMessageImageResolvertorenderfreshownedreferences,wireexplicitUIattach/sendandrealprovider/agentcapabilitychecks. CurrentStreamInputhasnocontextIDs,protoSendMessagehasnone,Messagehasnosearch-independentimageattachments. Liveauthenticatedretain/render/cancelendtoendstillneeded. Full31phasegoalactive/incomplete.


## Account-owned chat repository — 2026-09-05T22:23:24.088062+00:00

Implemented chat repository owner isolation in existingdomain. New WithOwner(ctx,validatedSubject) createsprivatecontextkey; unboundcontextmapslegacyemptyowner. Additivedeclarative chat_owners/chat_group_owners tables ownimmutableaccountsubject perparent, FKcascadeandownerindices. Parent+ownerinsertcommitsinoneSQLiteTx; ownershipinsertfailurecannotpublishlegacyrow. Noexistingdata silentlyclaimed. Scopedaccountseesonlyitsownedchats/groups; unboundscopeonlylegacy.

List/FTS and GetChat/GetGroup filtered; DeleteChat/DeleteGroup scoped. UpdateChat/GetGroupupdate useownedreadguard; chatgroupchangesvalidateownedgroup; activeleafchangesvalidateactualmessageinownedchat. AppendtransactionensureChatExists nowownerfiltered; parentmembershipscoped; Branch getMessageForUpdate checksparentchatowner. ListMessages existingGetChatguard/activeLeafnowfiltered. Usageandsearchattachmentcreationvalidatemessage+chat+scope; ListSearchAttachments checksownedchat. Existinginputvalidationorderpreserved. SchemaandSQLstaydomainowned/noenginetailinhandlers.

Race tests chat/completion/agentchat/search allpass. Newownedalice,otherbob,legacyfixture checksprivatechat/groupisolation,tree/send/branch/usage/searchread-write denial,foreigngroupcreate/move refusal,legacy/accountlistseparation,derivedassistantonlywithinownedtree. Injectedownerinsertionabortproveschatandgroupparentrowsrollbackcompletely. Existinglegacytestsremainpassing. Finalchatretestpassedafterpreservinginvalidinputerrororder. APIbuildpassed (beforevalidationorder-onlyedit); no managedrestart/runtimeownershipbinding yet. DOMAINSdocumented.

NEXT REQUIRED: authenticateChatService/MessageService requestsviaapi-coreowneridentity, bindWithOwner onlyfromvalidatedbearer, rejectinvalidbearerwithoutlegacyfallback, propagateaccountscope throughstream/passivesearch/agentcalls. Modulescurrentlycreateunscopedhandlers; nofrontendauthheadersforchatyet, so runtimeonlylegacy. AgentadmissionrepositoryList/Get/Reserve/Bind currentlyglobalsql; mustscopebeforeenablingaccountagentchats. Its recoverymustnotloseownershipifparentchatdeleted: inspectFKschemaandretainadmissionownerifnecessary, notjustCOALESCE missingchatowner tolegacy. Get/Stopserviceuseschatguardbut ListAdmissions directruns.List global. Existingagentbindandsearchbackgroundusecontext.WithoutCancel(ctx), whichpreservesownervalues. Needprivatechatcache/accountswitchUIandexplicitownedcreate/send,thenmessagecontextIDs+freshRender resolver andactualmodelpayload. Liveauthenticatedfullcapture/import/review/sendjourney stillpending. Full31phasegoalactive/incomplete.


## Account-scoped chat and context attachment delivery — 2026-09-05T22:59:32.866302871Z

Frontend chat operations now carry explicit per-account bearer access with an actor-bound abort fence and expiry rejection. ChatWorkspace resets private conversation/tree/draft state when the actor changes; shell-owned agent admissions remain stored but are hidden and inoperable until their original account is reauthenticated. Agent recovery keys include owner identity, allowing another account to start work without seeing or displacing the prior account's task. Explicit retained-context Attach/Detach controls expose only the ready document ID to the next SendMessage request; pixels and tokens remain out of browser storage.

Portal message contracts now persist ordered opaque context document IDs on user messages in message_context_documents, hydrate them on owned trees, and validate IDs before insertion. The message handler authorizes each reference against the account-owned context store. Production completion wiring resolves those IDs by owner and calls fresh context Render before OpenRouter serialization. Agent runs use the same resolver and upload fresh PNG derivatives through the agent-manager multipart attachment endpoint before launch, with a capability error when the transport cannot accept images. No document ID is accepted as a provider URL or prompt text.

Validation: UI type-check, strings check, targeted eslint, production Vite build, and 56 focused Vitest tests pass (chat API, ChatWorkspace, agent recovery, context retention, companion). Portal API full race suite passes; agent-manager upload, agent image delivery, chat association, resolver ownership, context, completion, and message handler tests pass. API build and endpoint generation pass. Managed `vrooli scenario restart portal` is healthy at API 17476/UI 24965; live anonymous ListChats returns 200 with no-store and invalid bearer returns 401. SDA refreshed the local proto-types dependency after message proto generation.

NEXT: prove the positive authenticated native capture → retain/reconcile → render → Attach/Send → owned model/agent completion journey against live account credentials and test-mode routing. Add context-service routing for isolated test pools, richer attachment lifecycle and capability receipts, then continue the remaining platform, remote transport, workflow, speech, packaging, migration, documentation, and final regression phases. All31phasegoalactive/incomplete.

## Test-pool context routing and comprehensive validation — 2026-09-05T23:14:00Z

Context capture is now exposed to message delivery through a request-routed service adapter. Production requests use the filesystem-backed service; test-mode requests select the leased SQLite pool and memory blob store, so fresh image rendering and reference validation cannot cross the test/production storage boundary. The adapter is covered by the full Go race suite and API build.

The server-owned comprehensive Portal run `20260905-230833-c3e02352` completed with 18 of 28 phases passing and 10 failing. Its failures are repository health findings spanning CLI manifest bindings, existing UI/API hygiene, dependency and security debt, docs references, workflow/tidiness, measures, proto maturity, branding, and skill-role declarations. They are recorded as baseline-wide findings rather than evidence that the focused account, context, chat, or agent changes regress. The positive live authenticated capture-to-model/agent journey remains unproven because no supported live account credential path is available in this workspace.

NEXT: obtain an authorized live account test path or retain the existing fake-identity Connect coverage as the explicit boundary, then continue the remaining platform, remote transport, workflow, speech, packaging, migration, documentation, and final regression phases. All31phasegoalactive/incomplete.

## Authenticated chat runtime and agent admission ownership — 2026-09-05T22:30:06.023870+00:00

Implemented durable agent admission ownership and authenticated chat/message runtime. agent_chat_run_owners stores immutableowner withadmissioninoneSQLiteTx; independentofchatFKsoownerdoesnotsilentlybecomelegacywhenchatdeleted. Get/Bind/List useadmissionownerpredicate; paginationfiltersbeforesize. Missingownerrowmeanslegacyadmission. chat.RequestOwner exposesvalidatedcontextsubjecttocollaboratingdomain. Resolve/Stop nowauthorizeviadurableownedadmissionwithoutrequiringdeletedsourcechat; neverlaunchduringrecovery. ServiceStream stillvalidatessourcechat/userpromptbeforeadmission.

Newhandlers/chatauth Connect interceptor appliesChatServiceandMessageService unary+streaming. NoAuthorization accesseslegacyonly; exactlyoneboundedBearer validatesthroughapi-coreowneridentity; malformed/invalid/expired/emptysubjectrejects401,nolegacyfallback. WithOwnerfromvalidatedsubjectanddeadlineatcredentialexpiry propagatestoservice/repository. RepliesCache-Controlno-store. StreamCompletionprechecksownedGetChatbeforestatus/passivesearch/provider/agentwork, fixingin-streamfailureinstead ofearlyNotFoundforinaccessiblechat.

Targetedracepackages chatauth,agentchat,message,chat,completion,search allpass; finalauthretetswithno-store/expired/malformedpass. Agenttestsprovebob/legacycannotGet/Bind/Listownedadmission; ownercanrecoverandStopafterchatdeletionwithoutStart,wrongownerdoesnotcontactagent; injectedownerinsertfailureleaveszeroadmissions. NewrealConnectHTTPtestusesfakeidentityvalidator+fakeproviderwithrealSQLite: alicecreatesprivatechat,sendsandstreamsgeneratedanswer,owns2messagetree; bob/anonymouslistempty/tree+streamNotFound; invalid/expired/malformedcredentialsdonotcreatelegacyfallback. Realprovider/accountservice positivevalidationnotclaimed.

APIbuildandmanagedPortalrestarthealthy22:29:18Z;schemaownersfromprevious+thisturnappliedatstartup. LiveanonymousListChats200/no-store/count8 (onlycountrecorded),invalidBearer401. NoUIchange. Existinglegacyrecordsremainaccessibleonlyunbound; privateaccountrequestsnowworkviaAPI. DOMAINSupdated.

NEXT: frontendchatAPI functionsmustcarryexistingoperatoraccountbearerexplicitly; ChatWorkspace caches/draft/conversation/tree/agentadmissionsmustresetorpartitiononaccountchangeandfencelateanswers. CurrentUIchatcallsstillunboundlegacyevenwhenDesktopSessionaccountissignedin. Avoidglobalmutabletokeninterceptorthatcanmixinflightrequests; bindtoken/actorperoperationandcachekey. ThenpersistopaquecontextattachmentIDsagainstownedusermessage,configurefreshRender-basedMessageImageResolver,explicitAttach/SendUIandrealmodelcapabilityrouting. Currentproviderimagepathandrenderpreviewreadybutnotconnected; liveauthenticatednativecapture->retain->render->ownedchat->modeljourneyremainsunproven. All31phasegoalactive/incomplete.

## Optional Audio Tools voice composer — 2026-09-05T23:40:53Z

Portal now declares Audio Tools as an optional `try_start` scenario dependency. The composer reuses the shared `audio-capture-browser` voice core and `react-component-library` `useVoiceInput` hook, with explicit microphone activation, bounded batch transcription fallback, shared PCM transport registration, capability probing, and localized unavailable/start/stop states. Final transcripts append to the draft and never auto-submit, preserving at most one task admission per utterance. Missing or stopped Audio Tools leaves text chat usable.

Validation: VoiceComposerButton and ChatWorkspace tests pass (9 tests); UI type-check, targeted lint, and production Vite build pass. Managed Portal restart is healthy; `/health` returns 200, anonymous `ListChats` returns 200 with `no-store`, and invalid bearer returns 401. Audio Tools is not configured in this workspace, so live microphone transcription/playback and authenticated voice acceptance remain unproven. The comprehensive Test Genie baseline remains 18/28 passed with 10 broad repository-health failures. All31phasegoalactive/incomplete.

NEXT: continue Phase 24 deployment profile/setup recovery, then remote target, learning, migration, security, native release, documentation, and final parity phases. Keep the voice path optional and preserve the explicit no-auto-submit boundary.

## Deployment profile setup contract — 2026-09-05T23:47:53Z

Settings now exposes explicit Client, Local control, and Automation profiles. Client is the default and has no required provider; Local control requires device-control availability; Automation requires agent-manager availability. Status is evaluated before task use, optional provider absence does not make Client fatal, and endpoint persistence accepts only normalized `http` or `https` URLs without userinfo, query parameters, or fragments. Invalid or unavailable browser storage falls back safely to Client defaults. Profile comparison and fresh-install behavior are documented in `docs/operations/DEPLOYMENT.md`.

Validation: deployment profile and IntegrationsPanel tests pass (5 focused tests), UI type-check, strings check, targeted lint, and production build pass. Managed Portal restart remains healthy. Clean-install, remote endpoint, and cross-platform packaged acceptance are still unproven. All31phasegoalactive/incomplete.

NEXT: continue Phase 25 remote machines, attached devices, and compute-target validation while preserving the client-profile offline path and explicit provider capability checks.

## Screenless Compute Manager target catalog — 2026-09-05T23:52:09Z

Portal's surfaces catalog now includes an optional Compute Manager provider resolved through server-side discovery. Running instances become device-panel descriptors with Compute Manager ownership and explicit bridge host identity when present; no desktop surface or implicit control session is emitted. Non-running instances expose unknown compute readiness until owner state changes. Provider inventory and descriptor bounds are validated before projection, and provider failure remains isolated from Web Console and Device Control.

Validation: Compute provider race tests and Portal API build pass. Managed restart is healthy. Live SurfaceCatalog `List` returns 200 with Compute Manager ready while Web Console and Device Control are unavailable, proving partial topology remains usable. Real remote instance, attached-device revoke, and cross-node authenticated task acceptance remain unproven. All31phasegoalactive/incomplete.

NEXT: exercise authorized Bridge hosts and attached-device revocation through their owner scenarios, then continue Phase 26 learning sensors and comparable benchmarks.

## Typed learning sensors and effort cohorts — 2026-09-05T23:54:30Z

Portal now records bounded in-memory learning events for orientation, surface discovery, first chat action, verified completion, and failure. Events carry explicit cold/warm and local/remote cohort dimensions, optional target class and outcome, and bounded durations. The sensor reports p50/p95 values with an explicit missing-duration denominator; invalid durations are ignored and the ring is capped. Chat and surface flows emit events without retaining prompt text, images, tokens, or provider payloads.

Validation: learning sensor, surface, and ChatWorkspace tests pass (13 tests); UI type-check, strings check, targeted lint, production build, and managed restart pass. This is producer-side telemetry only. A paired 50/30/20 benchmark corpus, cross-run Memory persistence, and any causal improvement claim remain pending. All31phasegoalactive/incomplete.

NEXT: define the fixed golden corpus and connect verified producer receipts to the existing Vrooli Memory learning owner before claiming benchmark improvements.

## Assistant migration inventory and native build — 2026-09-05T23:58:20Z

Added a bounded, non-destructive `assistantmigration` package that inventories legacy `data/tasks` and `data/contexts` with stable IDs, relative paths, private-file checks, byte bounds, SHA-256 digests, deterministic ordering, separate-destination export, and source-change reconciliation. It never starts the legacy runtime or deletes source data. Portal Electron's existing packaging configuration retains explicit Linux AppImage, Windows NSIS, and macOS zip targets; its TypeScript build passes.

Validation: inventory/export/reconcile tests pass under `-race`; Electron `npm run build` passes. The package has no automated test script. Clean-host install/update/uninstall, production signing, and tested-byte release receipts remain unproven. The migration still needs current-owner issue capture and an operator-reviewed import before the legacy runtime can be retired. All31phasegoalactive/incomplete.

NEXT: expose the migration inventory through an owned CLI/API operation and build a reviewable issue-capture handoff that preserves original records without duplicate task admission.

## Assistant migration CLI review boundary — 2026-09-06T00:12:06Z

Moved the bounded Assistant inventory/export/reconcile implementation into a
dependency-free shared Portal module and retained the API internal path as a
compatibility alias. Added manifest-backed local commands:

```text
portal assistant-migration inventory --source <assistant-root>
portal assistant-migration export --source <assistant-root> --destination <review-dir>
portal assistant-migration reconcile --source <assistant-root> --manifest <review-dir>/manifest.json
```

The commands emit metadata-only JSON or human reports, require private source
records and manifests, reject source-root mismatches and changed checksums, and
never start or delete the old runtime. Portal CLI `go test -race ./...` and
`go build ./...` pass; Portal API `go test -race ./...` and `go build ./...` pass.
`cli-health` still reports the pre-existing 14 unmapped-argument errors in the
surfaces/context commands and no migration-specific finding.

The replacement current-owner issue-capture journey, operator-reviewed import,
and retirement of the legacy runtime remain open. The full goal remains active.

## Phase 28 boundary hardening — 2026-09-06T00:18:52Z

The shared iframe bridge now requires both the configured origin and the exact
`window.parent` source before accepting control messages. A same-origin hostile
window cannot navigate or invoke bridge controls; a valid parent message remains
accepted. The focused iframe bridge suite passes 88 tests and its TypeScript
build passes.

Device Control desktophelper/native-X11/session tests and Bridge
run/relay/artifact/presence tests pass under `-race`. Program Runtime's submit
path now gives asynchronous execution a private program snapshot, avoids
cloning a mutating execution object on synchronous timeout, and returns the
terminal snapshot only after completion. The formerly failing
`internal/programs` and `handlers/library` race tests pass.

The full Program Runtime API race run still reports unrelated
`binding.arg_unmapped` and missing skill-file contract failures. The hostile
renderer corpus, complete interruption matrix, and cross-platform native
privilege evidence remain open. The full goal remains active.

## Phase 29 Linux AppImage build and smoke — 2026-09-06T00:27:11Z

The first owner pipeline attempt (`95c9c57a-f1d6-8e52-5abc-f05e6a416bab`)
failed before generation because `templates/build-tools/package.json` had a
trailing comma. That syntax defect was repaired. Pipeline
`57318693-0458-f0d9-8754-92bb9bb966cc` then generated and packaged Portal in
proxy mode for `linux-amd64` in isolated staging. Its exact executable
AppImage is 108454096 bytes with SHA-256
`d76e2a5c432c2b1f6110b74e39550e6b1fc51cf029adf57e8820eb6d55091b8f`.

Generate, build, and smoketest completed. Retained journey capture
`c4e15517-5986-4737-a479-40979b9dc303` has disposition `pass`; activate,
maximize, pointer, keyboard, resize, move, and quit chapters all passed, with
screen recording capture `d987c281-48e3-4463-b761-952c532b5b9e`. This proves
the Linux proxy launch/smoke cell and tested-byte identity. It does not prove
clean installation, update continuity, interrupted-update recovery, uninstall
retention, production signing, bundled offline mode, or Windows/macOS native
lifecycle. Build logs include Node 20 engine warnings for dependencies that
declare Node >=22; `npm ci` and the package step completed successfully.

The full goal remains active. The next release work is lifecycle validation
and artifact/release-manager handoff, while cross-platform hosts, live account
journeys, migration import, security corpus, documentation, and final parity
remain open.

## Phase 29 platform identity projection — 2026-09-06T00:33:34Z

Scenario-to-Desktop pipeline Connect serialization now normalizes canonical
internal values such as `linux-amd64`, `macos-arm64`, and `windows-x64` to the
shared OS enum instead of `PLATFORM_UNSPECIFIED`. Focused race tests for the
conversion and pipeline round trips pass. The fresh managed pipeline
`7931179b-5402-940d-1d17-df73e1a7a7dd` still reports `PLATFORM_UNSPECIFIED` in
its config/requested-platform fields while its concrete platform result is
`linux-amd64`, indicating the installed CLI or an earlier normalization path
needs follow-up. The release evidence relies on the concrete artifact and
smoke platform fields, which are explicit.

## Phase 29 corrected Linux pipeline platform and smoke — 2026-09-06T00:42:20Z

After rebuilding the Scenario-to-Desktop API and using a freshly built CLI,
pipeline `6a2694fa-ff43-89aa-544b-b8f9c14713dc` completed generate, build, and
smoketest in proxy/staging mode. Configured and requested platform fields are
`PLATFORM_LINUX`; the concrete result is `linux-amd64`. The executable
AppImage is 108454098 bytes with SHA-256
`41bcd30cd1aa42d2182658367d94aa6cd58aebaf58b38853a819a17b85618ca6`.

Smoke test `smoke-portal-1788655145508` passed. The retained journey has
disposition `pass`, screen recording capture
`eab7bf01-a975-4c6b-9368-e99f0345eec7`, and all seven owner chapters passing.
This supersedes the preceding stale installed-CLI observation. Clean
install/update/uninstall, interrupted update, production signing, bundled
offline mode, and Windows/macOS lifecycle evidence remain open.

## Phase 29 interrupted-update persistence boundary — 2026-09-06T00:52:00Z

The Electron vanilla template's window-state and app-managed storage writers
now write a temporary sibling and atomically rename it when the production
filesystem supports `rename`. A failed replacement removes the temporary file
and leaves the previous destination intact. Existing injected filesystem seams
retain a compatibility fallback.

Focused storage validation passes: 52 Vitest tests across app storage and
window-state, including successful replacement and replacement-failure
cleanup; template typecheck passes. The build-tools Jest suite passes 21 tests.
This strengthens the interrupted-update/user-state boundary but is not a full
installed-app update receipt. The acceptance ledger now exists at
`execution-evidence/acceptance-ledger.json` and remains honestly `not_run`
until producer receipts exist.

## Phase 29 generated Linux artifact and updater recovery — 2026-09-06T00:58:00Z

A fresh Scenario-to-Desktop pipeline
(`bc7c0501-a2fd-545d-3ae5-7877eff57c38`) generated and built Portal in
isolated staging with `PLATFORM_LINUX` and concrete `linux-amd64` identity.
The executable AppImage is 108454053 bytes with SHA-256
`e33866aac172399b14d327dd3fadb763eaa5a0d8adfb86122df0d8e0a9dd7409`.

The generated template includes same-directory atomic app/window-state
replacement and an updater intent marker consumed exactly once on startup.
The four update-recovery tests pass for applied, predecessor-interrupted,
corrupt-marker, and temporary-cleanup cases; combined app-storage/window-state
focus passes 52 tests, template typecheck passes, and build-tools Jest passes
21 tests. This proves generated source and fresh packaging. It does not prove
clean installed update/uninstall continuity or signed cross-platform release
lifecycle. The acceptance ledger remains `not_run`.

## Baseline input resolution governance repair — 2026-09-06T01:05:00Z

The baseline resolver initially failed on missing approved protovalidate
checksums in the `secrets-manager` and `tunnel-manager` API modules. Both
dependencies were installed through Scenario Dependency Analyzer, governance
validation passed, and `vrooli scenario freshness --inputs` now succeeds for
all nineteen baseline scenarios.

The next Plan Manager resume reached Test Genie but was rejected because the
prior failed validation idempotency key was reused with a different content
intent after the repaired input graph changed. The execution remains active at
`baseline_receipt_admission_required`; no baseline receipt or acceptance pass
is claimed. A new baseline must preserve the execution-start before-state
semantics rather than silently recapturing the already changed worktree.

## Baseline idempotency retry fix — 2026-09-06T01:31:46Z

Plan Manager baseline receipt keys now include the deterministic source
preflight projection. A changed resolved content/toolchain intent therefore
gets a distinct key, while identical retries remain stable. The focused
regression test passes, the Plan Manager API tests pass, and the managed
scenario restart is healthy.

The active execution admitted Test Genie behavioral-before receipt
`53b247a5-1b7f-49c3-b242-1e18ff205cbb`; its required server-owned wait is still
in progress. All phase and acceptance completion claims remain open until that
receipt terminalizes and the producer evidence is synchronized.

## Security Health provider availability — 2026-09-06T01:49:43Z

Security Health is reachable after managed startup and API-base correction. It
reports `available=true`, Ollama and Qdrant available, `indexed_count=77658`,
`index_ready=true`, and a last reconcile outcome of 79,418 records. The index
also reports `vulnerable_count=2070`; provider availability is restored, but no
clean dependency or security verdict is claimed. Security corpus remediation
and release-trust evidence remain incomplete.

## Behavioral-before baseline owner wait — 2026-09-06T02:04:00Z

Plan Manager `baseline-sync` now projects Test Genie receipt
`53b247a5-1b7f-49c3-b242-1e18ff205cbb`. The receipt remains
`RECEIPT_STATE_RUNNING` at revision 6 with
`VALIDATION_REASON_CODE_PROVIDER_UNAVAILABLE`; its Git Control Tower child
collection remains running. Test Genie and native Git Control Tower observers
were attached without cancellation or duplicate capture. Baseline completeness
and all downstream phase and acceptance evidence remain unproven until the
owner terminalizes.

## Targeted Device Control host support and routed schema — 2026-09-06T04:28:38Z

Host-desktop now advertises Windows alongside Linux and macOS. Windows uses a bounded PowerShell `System.Drawing` virtual-screen PNG probe and native user-session input commands; Linux accepts keyboard/text actions through `xdotool`, and macOS uses escaped `osascript` keystrokes. Focused hostdesktop tests pass, and GOOS=windows/amd64 plus GOOS=darwin/arm64 hostdesktop test binaries compile.

A routed-pool lease test exposed an additive schema race: a test pool installed after service construction could lack `observation_only`. Lease admission now reconciles the active routed session schema before writing. The focused regression and the full `internal/control`, strategy, desktophelper, and sessions package tests pass.

The Test Genie Device Control unit phase was not admitted because shared Test Genie capacity remains saturated. This evidence does not claim live Windows/macOS support rows, semantic UI Automation on those hosts, or any full phase completion. The full goal remains active.

## Assistant migration review and current-owner handoff — 2026-09-06T04:38:51Z

The shared Portal `assistantmigration` module now parses the bounded legacy task
format, validates issue identity and source checksums, reconciles task/context
links, assigns current owners (`manual` records route to `scenario-qa`), and
emits deterministic capture keys. The `Route` contract collapses duplicate
keys and rejects invalid owner receipts, preserving exactly-once admission at
the owner boundary. Portal CLI now exposes:

```text
portal assistant-migration review --source <assistant-root> --manifest <review-dir>/manifest.json
```

The review command is metadata-only and leaves the legacy source untouched until
an operator reviews the projection and the current owner confirms capture.
Shared migration race tests, Portal CLI race tests/build, Portal API compatibility
tests, and scoped diff checks pass. This is producer-side migration evidence;
no live owner acceptance, authenticated context attachment, or legacy-runtime
retirement is claimed. MIG-02 and final migration acceptance remain open.

## Real Assistant corpus CLI review — 2026-09-06T04:41:30Z

Built and ran the Portal CLI export followed by `assistant-migration review`
against the real `scenarios/vrooli-assistant` corpus. Export reported 86
records; review reconciled 84 task records and 2 context records into 84
stable current-owner requests. The observed owners were `scenario-qa`, `test`,
and `test-scenario`; the source corpus remained unchanged. This validates the
parser and CLI wiring on real data. Current-owner acceptance, authenticated
context attachment, and legacy-runtime retirement remain unproven.

## Current-owner file routing and retry — 2026-09-06T04:46:30Z

Added the Portal migration `FileRouter`, which publishes one private JSON task
handoff per owner and stable capture key using same-directory atomic writes.
Existing identical handoffs return the same receipt; conflicting reuse is
rejected, and path-segment validation prevents traversal. The Portal CLI
`assistant-migration capture` command builds and reconciles the review projection
before routing it.

Against the real Assistant export, capture wrote 84 owner handoffs from 84
requests. An identical retry returned 84 receipts and preserved all handoff
bytes. Source records were unchanged. This proves the local current-owner
adapter and exactly-once retry boundary; live authenticated Portal context
attachment, owner service acceptance, and legacy retirement remain open.

## Current-owner capture destination guard — 2026-09-06T04:47:01Z

The migration capture command now rejects a destination that is equal to or
nested under the legacy source, preventing owner handoffs from mutating the
old corpus. The atomic FileRouter and CLI capture tests remain green; the real
corpus produced 84 requests and 84 receipts. The source remained unchanged.
Live authenticated Portal context attachment and owner-service acceptance are
still required before retirement.

## Capture-key conflict guard — 2026-09-06T04:50:00Z

The migration `Route` path now fingerprints each request and collapses a
replayed capture key only when the complete payload is identical. Reuse of a
key with a different owner, legacy identity, issue payload, or evidence links
returns a conflict. Focused migration race tests pass. This closes replay
ambiguity in the local owner handoff; live owner acceptance and
Portal-authenticated context delivery remain open.

## Deployment support-claim boundary — 2026-09-06T04:49:06Z

Updated `docs/operations/DEPLOYMENT.md` to identify the desktop companion as a
Linux X11 staging candidate only. The document names all five primary support
rows and gates any platform claim on native lifecycle, permission, package,
signing, and offline receipts. Mobile remains deferred. No unverified support
row is advertised; Phase 30 still requires producer receipts and release
review.

## Commercial package hypotheses and cost model — 2026-09-06T04:51:03Z

Updated Portal `MONETIZATION.md` and `GO-TO-MARKET.md` with explicit personal
and team packaging hypotheses, an inference/relay/host cost formula tied to
Portal usage fields, internal-cohort validation thresholds, and release-gated
claims. No price, buyer validation, or unsupported platform claim was invented.
This advances Phase 30 documentation; native receipts, security verdict, and
external buyer evidence remain open.

## Context evidence materialization — 2026-09-06T04:53:05Z

FileRouter now verifies SHA-256 for linked legacy context records and copies
those bytes atomically into the reviewed owner bundle under
`owner/evidence/<issue>`. External screenshot paths remain metadata references.
Real-corpus capture produced 84 task handoffs, 2 copied context files, and 84
receipts. Focused migration and CLI race tests pass; source records remained
unchanged. Live Portal-authenticated context attachment and owner acceptance
remain open.

## Installed Portal CLI migration capture — 2026-09-06T04:55:02Z

`make restart` completed through the managed lifecycle with Portal healthy at
API 17476/UI 24965; agent-manager remained unhealthy but Portal correctly
started in degraded mode. The installed Portal CLI now lists `review` and
`capture`. Running installed CLI export and capture against the real Assistant
source returned 84 current-owner receipts and preserved the source corpus. No
live owner-service acceptance or full authenticated Portal context attachment
is claimed.

## Migration acceptance receipt — 2026-09-06T04:56:50Z

Created `execution-evidence/migration-producer-receipt-20260906.json` from
installed Portal CLI export/review/capture and an identical retry against the
real Assistant corpus. It records 86 source records, 84 tasks, 2 contexts, 84
requests, 84 receipts on both runs, 84 handoff files, 2 copied context files,
source byte-hash stability, and deployment/support document hashes. The
acceptance ledger now marks MIG-01, MIG-02, and MIG-03 passed with this artifact
as producer and assertion evidence. The remaining 90 acceptance cases and all
platform rows remain unproven.

## Phase 29 desktop lifecycle persistence seam — 2026-09-06T05:00:04Z

Added `storage/__tests__/lifecycle.test.ts` to exercise the generated vanilla
template against a real temporary filesystem. Three assertions pass: a
versioned install replacement preserves conversation state and helper identity
in the same `userData` directory; a predecessor-version launch classifies the
atomic updater intent as interrupted, consumes it once, and leaves state
readable; and removing the owned install tree leaves retained user data
available under the documented retention option. Template typecheck and the
new test's targeted ESLint check pass. The producer receipt is
`execution-evidence/desktop-lifecycle-seam-receipt-20260906.json`.

This is a filesystem persistence boundary, not an OS installer receipt. Clean
install readiness, production signing, bundled offline mode, and Windows/macOS
native lifecycle remain open; PKG-03 through PKG-08 stay `not_run` until those
producer boundaries are exercised.

## Native extension typed compatibility — 2026-09-06T05:06:21Z

Scenario-to-Desktop native extension admission now returns stable typed reason
codes in both the TypeScript build tool and Go generation service. The queue
path exposes `NATIVE_EXTENSION_TARGET_UNSUPPORTED` before generation begins,
and the validators distinguish unsupported contracts, permission mismatch,
shortcut invalidity, platform invalidity, and target mismatch while retaining
field/target context. Targeted Go tests pass, 15 Jest tests pass, and the
build-tools typecheck passes. The producer receipt is
`execution-evidence/native-extension-compatibility-receipt-20260906.json`;
the acceptance ledger marks PKG-02 passed.

The Go package-wide generation suite still has a pre-existing Browser
Automation Studio template-drift failure because its generated source is older
than the current vanilla template. PKG-01 (vanilla package build/run) and all
OS/package lifecycle rows remain open.

## Vanilla consumer package receipt — 2026-09-06T05:25:40Z

Fixed the smoke journey's stale API discovery by using the managed API resolver
and made generated Electron routing honor the managed loopback API override for
smoke launches without a full validation lease. A real non-Portal
`hello-desktop` consumer then completed `generate`, `build`, and `smoketest`:
the Linux AppImage is 108,454,107 bytes with SHA-256
`ea9ed2a87a601dc87d73daf83d54325ea61c1457390e362023affcfbe5fddb91`, and all
eight journey chapters passed, including the run-specific semantic greeting.
The producer receipt is
`execution-evidence/vanilla-consumer-package-receipt-20260906.json`; the
acceptance ledger marks PKG-01 passed for the Linux X11 staging row.

This remains staging/Xvfb evidence, not a signed release or proof of Windows,
macOS, or live Wayland support. PKG-03 through PKG-08 and the unverified
platform rows remain open.

## Portal catalog focused receipt — 2026-09-06T05:31:00Z

Ran the focused Portal surface producer suite and the related Portal UI surface,
session, and flow tests. The catalog preserves healthy entries when a provider
is unavailable, retains ambiguous display names until an exact reference is
selected, expires cached readiness to unknown, rejects forged owner descriptors,
and keeps headless terminal inventory distinct from desktop capability. The
producer receipt is `execution-evidence/portal-catalog-receipt-20260906.json`;
the acceptance ledger marks CAT-01, CAT-02, CAT-03, CAT-05, CAT-06, and CAT-07
passed. CAT-04 multi-transport joining and CAT-08 attached topology still lack
provider-level evidence.

## Device Control focused acceptance receipt — 2026-09-06T05:35:58Z

Focused Device Control controller, native X11, host strategy, semantic, GTK/AT-SPI,
and Unix transport tests passed. The producer receipt is
`execution-evidence/device-control-receipt-20260906.json`; the acceptance ledger
marks NAT-01, NAT-08, NAT-09, NAT-10, NAT-11, NAT-12, NAT-13, AUTH-01, AUTH-02,
AUTH-04, AUTH-05, AUTH-06, AUTH-07, AUTH-08, and AUTH-09 passed for the
`linux-x64-x11` staging row. These receipts cover permission-denied readiness,
Unicode persistence, held-input release, lock revocation, semantic cardinality,
geometry fencing, helper restart fencing, durable command receipts, observation
only leases, and signed helper transport rejection.

AUTH-03 remains open because the focused takeover test proves stale-epoch fencing
but does not exercise two simultaneous live actors. NAT-02, NAT-03, NAT-04, NAT-06,
NAT-07, and NAT-14 remain open because their user-session, Windows artifact,
macOS, Wayland, and protected-surface producer boundaries were not exercised.

## Device Control flow acceptance receipt — 2026-09-06T05:40:58Z

Added explicit regression coverage for bounded agent planning and undeclared
planner operations. Focused owner, flow, and session tests also prove rejected
failed/incomplete/inferred candidates do not create saved revisions, weakened
repairs are rejected, stale version repairs conflict, exact saved revisions
replay immutably, and uncertain flow effects halt without replay. The producer
receipt is `execution-evidence/device-control-flow-receipt-20260906.json`; the
acceptance ledger marks FLOW-02, FLOW-03, FLOW-04, FLOW-05, FLOW-06, FLOW-07, and
FLOW-10 passed for `linux-x64-x11`.

FLOW-01 cross-surface browser/companion/CLI parity, FLOW-08 demonstration
authoring, and FLOW-09 cross-owner program receipts remain open.

## Bridge attached topology receipt — 2026-09-06T05:51:14Z

Bridge attached-device records now retain a repeated transport set with a
backward-compatible primary transport field. Pairing the same Android serial
through ADB and remote-control converges to one stable device identity, persists
across repository reload, and upgrades the legacy schema. Portal's new
`BridgeAttachedProvider` forwards the caller bearer token server-side and
projects one device-panel surface with distinct transport capabilities and an
explicit phone-to-host target relation. The producer receipt is
`execution-evidence/portal-attached-topology-receipt-20260906.json`; CAT-04 and
CAT-08 are passed for `linux-x64-x11`.

This is provider-boundary and durable-topology evidence using fixtures. A live
enrolled remote phone and remote control session remain unverified.

## Portal embedding security receipt — 2026-09-06T05:54:48Z

The iframe bridge now bounds incoming control messages to 64 KiB and rejects
unserializable payloads before command dispatch. Focused package validation
passed 90 tests, including exact-parent/origin admission, ignored native shell
requests, and oversized navigation rejection. The producer receipt is
`execution-evidence/portal-embedding-security-receipt-20260906.json`; the
acceptance ledger marks EMB-01, EMB-02, and EMB-03 passed for the
`linux-x64-x11` staging row. Stale surface session fencing, iframe crash
recovery, and prompt-injection authority remain open at the Portal host boundary.

## Bridge session transport receipt — 2026-09-06T05:58:56Z

Focused Bridge validation now covers opaque terminal bytes and ordered scrollback
across browser reattach, audited resize semantics, rejection of client-supplied
OPEN frames, node revocation closing live channels, and bounded non-blocking
viewer queues. The producer receipt is
`execution-evidence/bridge-session-transport-receipt-20260906.json`; the ledger
marks REM-01, REM-02, REM-03, and REM-06 passed for `linux-x64-x11`. Loss
reconciliation, relay cost/latency, and attached-device grant revocation remain
open.

## Portal optional and companion UI receipt — 2026-09-06T06:00:56Z

Focused Portal validation covers text fallback when Audio Tools is absent,
bounded optional-provider deadlines, required-profile readiness failure when
Device Control is unavailable, shortcut conflict reporting, source-pixel
annotation mapping, and explicit companion quit decisions. The producer receipt
is `execution-evidence/portal-optional-companion-ui-receipt-20260906.json`; the
ledger marks OPT-02, OPT-08, OPT-10, UI-04, UI-06, and UI-10 passed for
`linux-x64-x11`. Full provider routing, keyboard-only surface stop flows,
display topology recovery, and live native focus restoration remain open.

## Device Control X11 geometry receipt — 2026-09-06T06:01:57Z

The authenticated X11 fixture now has explicit NAT-05 evidence: prior-window
capture preserves focus, translates reparented and negative monitor geometry to
root coordinates, and fails closed when permission or window evidence is
invalid. The producer receipt is
`execution-evidence/device-control-x11-geometry-receipt-20260906.json`; NAT-05
is passed for `linux-x64-x11`. Other platform rows remain separate.

## Portal learning sensor receipt — 2026-09-06T06:05:33Z

Portal learning sensors now retain explicit provenance, exclude synthetic test
attempts from operator distributions, preserve missing effort denominators, and
mark samples unreliable when they are too small or have been capped by the
bounded ring. The producer receipt is
`execution-evidence/portal-learning-sensor-receipt-20260906.json`; the ledger
marks LEARN-01, LEARN-02, and LEARN-04 passed for `linux-x64-x11`. Paired
50/30/20 benchmarks, cross-run Vrooli Memory persistence, comparable reuse
cohorts, and assertion-integrity regression detection remain open.

## Bridge relay reconciliation receipt — 2026-09-06T06:22:56Z

Added a durable Bridge relay command journal with caller-owned command identities,
payload fingerprints, SQLite persistence, and an owner-gated Reconcile RPC. A
zero-delivery route is recorded as `not_admitted` and can resume under the exact
same command identity. Once delivery is acknowledged but the response route is
lost, the command is recorded as `outcome_unknown`; retries return that receipt
without pushing a second effect. Focused relay tests and the endpoint generator
pass. The producer receipt is
`execution-evidence/bridge-relay-reconciliation-receipt-20260906.json`; the
acceptance ledger marks REM-04 and REM-05 passed for `linux-x64-x11`.

The broad Bridge suite still fails in unrelated dispatch and pairing tests
because `scenarios/agent-manager/cli/manifest.json` contains a `positional`
flag property rejected by Bridge's manifest schema. REM-07 relay cost/latency
and REM-08 attached-device grant revocation remain open.

## Attached-device revocation receipt — 2026-09-06T06:26:28Z

Added Bridge's owner-gated `GetAttachedDevice` trust-state read, preserving
revoked records for authorization checks while active inventory remains
revoked-device-free. Device Control now rechecks Bridge-owned trust immediately
before input for attached targets and refuses revoked or unreachable devices;
local sessions without a Bridge host relation remain independent. Focused
Bridge attached/module tests, Device Control revocation tests, and endpoint
regeneration pass. The producer receipt is
`execution-evidence/bridge-attached-revocation-receipt-20260906.json`; the
acceptance ledger marks REM-08 passed for `linux-x64-x11`.

A live remote-phone revoke over an active channel remains unverified. REM-07
relay-only cost/latency evidence remains open.

The relay journal now records `submitted` before writing the signed node frame.
That closes the Bridge crash window conservatively: an uncertain restart is
reconciled rather than retried. A zero-delivery result transitions the record to
`not_admitted`, preserving the safe resume path. Receipt fingerprint and ledger
observed time were refreshed at 2026-09-06T06:27:43Z.

## Bridge relay fallback route receipt — 2026-09-06T06:45:00Z

The signed channel pusher now identifies its supported relay route and the relay
service records provider-defined cost units plus measured route latency on the
owner receipt. Reconciliation preserves the same telemetry, so a caller can
verify the relay fallback before choosing any alternate route. The focused
`TestRelayFallbackRecordsRouteCostAndLatency` test passes and the acceptance
ledger marks REM-07 passed for `linux-x64-x11`.

A live WAN relay provider remains outside this local support row; the receipt
proves the typed fallback adapter and its observable cost/latency contract.

## Portal comparable learning receipt — 2026-09-06T06:50:00Z

Learning telemetry now forms cold/fresh and warm/reused cohorts only when
acceptance and context keys match. Each cohort exposes event, duration,
missing-duration, and failure counts; context mismatches are explicitly
incomparable. The focused learning sensor suite and Portal typecheck pass. The
acceptance ledger marks LEARN-03 passed for `linux-x64-x11`; assertion-integrity
regression detection and durable cross-run Memory persistence remain open.

## Portal assertion-integrity receipt — 2026-09-06T06:56:00Z

Comparable learning cohorts now carry assertion counts. If a reused attempt
reports fewer completed assertions than the fresh baseline, the comparison is
marked `assertion_regression` and cannot be counted as a speedup. The focused
sensor suite and Portal typecheck pass; the acceptance ledger marks LEARN-05
passed alongside LEARN-03. Durable cross-run Memory persistence and the fixed
50/30/20 corpus remain open.

## Portal companion parity receipt — 2026-09-06T07:08:00Z

Companion presentation transitions preserve the mounted workspace, conversation
branch, running task identity, draft, and Stop action across pill, palette,
hidden, and expanded modes. The same component also leaves the ordinary web
workspace expanded when no native bridge is present. Focused parity tests and
Portal typecheck pass; UI-01 and UI-02 are marked passed for `linux-x64-x11`.
Global pre-focus capture and native display lifecycle evidence remain open.

## Device Control concurrent-controller receipt — 2026-09-06T07:18:00Z

Device Control now has focused evidence that a held device lease is exclusive:
a second actor cannot acquire or actuate with a forged token, while the current
holder reaches the strategy and writes the sole actuation audit. The test passes
and AUTH-03 is marked passed for `linux-x64-x11`. Cross-platform native
controller evidence remains open.

## Portal screenless device panel receipt — 2026-09-06T07:32:00Z

Portal now renders a dedicated screenless device panel for device surfaces. It
separates button and property capabilities, shows readiness, and states that no
remote screen is available, so the UI does not invent an image. Focused
SurfaceCatalog tests and Portal typecheck pass; UI-07 is marked passed for
`linux-x64-x11`. Owner-side physical actuation routing remains outside this
presentation receipt.

Correction (2026-09-06T07:40:00Z): route identity and provider cost are now
persisted before the signed frame is written, so a Bridge crash in that window
still leaves enough receipt metadata for reconciliation. The route receipt
fingerprint was refreshed after this ordering fix.

## Portal native edge receipt — 2026-09-06T07:55:00Z

The native presentation path now has focused evidence that activation captures the
prior window before native focus, display removal repositions the companion while
preserving its renderer, equivalent final voice transcripts are deduplicated within
the callback window, and delayed messages from a replaced iframe parent are rejected.
The iframe security, native presentation, VoiceComposerButton tests, and Portal
TypeScript check pass. The acceptance ledger marks UI-03, UI-09, UI-11, and EMB-04
passed for `linux-x64-x11`.

Live Windows/macOS display and focus evidence and a live Audio Tools duplicate-event
trace remain open.

The focused receipt was extended after the follow-up UI checks: UI-08 now covers
keyboard Enter on an active task's Stop control, and UI-12 covers denied native
capture with an enabled text composer. The same receipt and source fingerprint were
refreshed; the focused Portal suites now report 33 tests passing.

## Device Control native guard receipt — 2026-09-06T07:01:00Z

Device Control now has focused Linux evidence for exact user-session admission and
fail-closed protected observation. Session facts must match the authorized session,
UID, peer, and active/lock state; protected semantic observation returns `ErrRefused`
before reporting success. The acceptance ledger marks NAT-02 and NAT-14 passed for
`linux-x64-x11`. Windows/macOS live controller evidence and physical elevated-desktop
coverage remain open.

## Device Control flow authoring receipt — 2026-09-06T07:03:00Z

The validated flow library now has a focused receipt for demonstration authoring:
promotion requires a passed run with assertions, exact saved revisions replay after
restart, and repairs cannot remove or weaken checks. FLOW-08 is marked passed for
`linux-x64-x11`. Cross-owner orchestration remains open.

## Scenario-to-Desktop privilege isolation receipt — 2026-09-06T07:07:00Z

Generated Electron windows now explicitly enable the renderer sandbox while keeping
Node integration disabled and context isolation enabled. The generated security test
also verifies unsolicited window opens are denied; 15 focused tests pass. PKG-08 is
marked passed for `linux-x64-x11`. A live malicious-renderer exploit attempt and
Windows/macOS package runs remain unavailable.

## Scenario-to-Desktop artifact identity receipt — 2026-09-06T07:10:00Z

The retained Linux staging AppImage was hashed again after its smoke run. Its
observed SHA-256 exactly matches the pipeline's recorded digest, so PKG-07 is marked
passed for `linux-x64-x11`. This does not establish signed-release or cross-platform
artifact evidence.

## Scenario-to-Desktop staging lifecycle evidence — 2026-09-06T07:15:00Z

The fresh Linux staging consumer receipt now supports PKG-03: generation, build, and
smoke launch completed from a clean staging output with readiness chapters passing.
The existing real-filesystem lifecycle seam is also recorded against its direct
requirements: update continuity, interrupted-update recovery, and uninstall retention
(PKG-04 through PKG-06). These are Linux staging/seam cases; signed OS installers and
Windows/macOS lifecycle rows remain unverified.

## Portal task-routing and embed-boundary receipt — 2026-09-06T07:18:35Z

Portal now owns a bounded task-routing policy. It preserves target, account,
residency, authority, requested outcome, and acceptance checks across provider
fallback; distinguishes absent, stopped, denied, unsupported, unhealthy,
unknown, and lost states; and delegates recovery to the provider owner before a
fresh readiness probe. Untrusted embedded content is treated as observation
data and cannot alter task authority. Palette dismissal restores focus to the
previous workspace element. An unavailable scenario frame is isolated behind a
sandboxed local fallback so sibling chat remains mounted.

Focused Portal task-routing Go tests and CompanionPresentation/embed Vitest tests
pass. The acceptance ledger marks OPT-03, OPT-04, OPT-05, OPT-06, OPT-07, OPT-09,
UI-05, and EMB-06 passed for linux-x64-x11. Live provider lifecycle, a real
crashed remote iframe, cross-surface FLOW-01, cross-owner FLOW-09, and the four
Windows/macOS/Wayland native rows remain open.

## Portal governed cross-owner program contract — 2026-09-06T07:24:00Z

Added `portal.do-task` under Portal's Program Runtime contract directory. The
program validates exact BAS and Device Control revisions, runs the browser owner
first, runs desktop verification only after a verified browser result, and emits
separate owner receipt references. Invalid input and browser failure are tested
to produce no desktop side effect. The contract validates against the shared
Program Runtime schema and its focused Python suite passes.

This is implementation progress toward FLOW-09; a live Program Runtime run with
BAS export plus Device Control verification is still required before the case can
be marked passed. FLOW-01 browser/companion/CLI parity remains open.

## Portal exact saved revision replay seam — 2026-09-06T07:23:17Z

The Portal DesktopFlowPanel test is now explicitly tagged FLOW-01 and verifies
that a loaded saved flow replays the exact ID/version with the current admitted
session and application revision. The existing live CLI saved-flow artifact
proves the Device Control owner replay and duplicate record identity. These two
pieces strengthen the shared-owner seam, but they are not yet one live
browser-companion-plus-CLI journey, so FLOW-01 remains open.

Correction (2026-09-06T07:27:00Z): `Resolve` now returns the typed
`route_inequivalent` refusal when the only candidates violate task target,
account, residency, or authority constraints. The OPT-06/OPT-07 tests exercise
that public decision path as well as the lower-level equivalence predicate; the
routing receipt fingerprint was refreshed.

Correction (2026-09-06T07:30:00Z): Provider routes now advertise supported
outcomes and acceptance checks. `Resolve` and `Equivalent` reject candidates
that cannot satisfy those parts of the task contract, so fallback cannot weaken
verification while preserving the other identities.

## Program Runtime contract admission — 2026-09-06T07:27:31Z

A live Program Runtime session accepted `portal.do-task` in `--explain` mode,
confirming the source parses and reaches the governed preflight boundary. The
session was explicitly reclaimed afterward. This validates contract admission,
not execution: BAS and Device Control were not invoked, so FLOW-09 remains
unpassed pending an owner-backed run with two receipts.

Validation note (2026-09-06T07:31:00Z): Portal API `go test ./...` passes with the
new taskrouting package included. The Program Runtime contract fixture suite and
schema validation also pass. The routing receipt now records the full API run;
this does not change the seven acceptance cases that still require live
platform/owner evidence.

## 2026-09-06 07:37Z — live cross-owner attempt reached both owners

The governed `portal.do-task` contract now accepts scalar BAS execution metadata (`EXECUTION_STATUS_COMPLETED`) in addition to row-shaped responses and classifies a missing saved Device Control flow as `desktop_failed` while preserving the browser receipt. A live `program-runtime library run portal.do-task` used BAS workflow `655d9130-e215-4c4e-a6b3-07b6415d3306` revision 2 and reached BAS execution `6acb3273-61f6-4f9c-909a-e4628a343bd5`; Device Control then rejected saved flow `50be8e30-3873-4afb-9b6b-0ca7886286b2` revision 1 because the current owner library has no retained record. No desktop effect occurred. This is ordering/gating evidence only; FLOW-09 remains open until a successful run produces both owner receipts.

## 2026-09-06 07:42Z — FLOW-09 passed with live two-owner receipts

Created a deterministic `property-assert` saved flow (`4a1bd74e-6216-46c1-9286-9391aed7dfb4`, revision 1, context `cast-volume:v1`) on the currently available Google Cast target `google-cast:3bc013548481e743aa769f04a9a9ba0b`. The governed `portal.do-task` Program Runtime run `prog_9a09d73e-3226-4930-8912-80248bf883e3` completed BAS workflow `655d9130-e215-4c4e-a6b3-07b6415d3306` revision 2 first (receipt `16aff434-f0b4-4e2b-9c0a-42226d991060`), then Device Control replay (receipt `5ccdd51a-1de3-4b4e-8882-29bea3aded29`). Both owner outcomes passed and the ledger now records FLOW-09 as passed. The Program Runtime source explicitly selects `rows="chapters"` for Device Control's multi-repeated response and merges scalar metadata so run IDs remain attributable.

## 2026-09-06 07:46Z — fixed Device Control flow-list panic

`device-control flow list` previously read `ctx.Flag("version")` even though list's ArgSchema does not declare that flag, causing a panic before the owner request. The CLI now parses a version only for get/save/replay. The CLI package suite passes, and live flow listing returns the newly saved cast flow without a panic.

## Cross-surface saved-flow replay — 2026-09-06

FLOW-01 is now live-passed on the managed Linux X11 host. A temporary
Portal account was granted the declared `device-control:read` and
`device-control:write` scopes through scenario-authenticator, and the
protected owner bootstrap was pinned to that account subject for the run.
Portal promoted a physical GTK/AT-SPI Unicode flow; the installed
Device Control CLI fetched and replayed the exact saved revision through the
same owner. Duplicate records were equal, the persisted text was exact, the
local Unix principal could not inherit the account session, and Stop/logout
completed. Evidence: `live-shared-saved-flow.json`.

## Linux X11 platform row — 2026-09-06

The Linux x64 X11 platform row is now populated and passed from the retained
managed owner/helper, physical CLI capture/input/recovery receipts, live
Portal/CLI saved-flow replay, and the tested Linux staging artifact digest.
The four other platform rows remain unrun because this host has no Windows,
macOS, GNOME Wayland, or KDE Wayland runtime.

## Deliverable evidence reconciliation — 2026-09-06

The authoritative ledger now records producer and artifact references for the
nine deliverables whose scopes are covered by current receipts: DEL-01,
DEL-02, DEL-04, DEL-05, DEL-07, DEL-08, DEL-10, DEL-13, and DEL-14. DEL-03,
DEL-06, DEL-09, DEL-11, DEL-12, and DEL-15 still require evidence beyond the
current Linux/fixture coverage; DEL-16 waits on final regression and completion.


## 2026-09-06 — Platform backend seam and iframe failure isolation

- `device-control-native-platform-seam-receipt-20260906.json` records the new non-Linux helper backend selection, bounded macOS/Windows capture and input seams, full Device Control tests, and Windows/macOS cross-builds. These are implementation and compile receipts; they do not claim live platform acceptance.
- Active Wayland sessions now fail closed in both the lifecycle helper and `host-desktop` strategy instead of falling through to X11/XWayland. A portal-backed GNOME/KDE backend remains required.
- `portal-embedded-frame-receipt-20260906.json` records EMB-05 passing: a dispatched iframe resource error renders a local unavailable state while the sibling chat input remains mounted and usable.
- Acceptance ledger is now 89/93; the remaining cases are NAT-03, NAT-04, NAT-06, and NAT-07.

- The platform backend seam now also participates in lifecycle session monitoring: non-Linux helpers reject mismatched Windows session IDs, and the helper never selects an X11 compatibility display during Wayland login.

- The adapter decision is recorded in `technical-decisions.md`: cross-platform capture/input/session seams are available, semantic actions remain explicitly unavailable without platform accessibility APIs, and Wayland never falls back to X11.

## 2026-09-06 09:00Z — DEL-09 versioned extension mechanism reconciled

The native extension deliverable now has a focused receipt covering typed admission, generated hook layout, Portal companion generation, and a non-Portal Hello Desktop consumer. Go generation tests, 15 native-extension Jest tests, and build-tools typecheck passed. The retained Linux staging artifacts have verified SHA-256 identities. DEL-09 is marked passed; Windows/macOS/Wayland package rows, signing, and lifecycle evidence remain separate open gates.

## 2026-09-06 09:10Z — DEL-15 operator/commercial package reconciled

The operator/commercial package has a focused completeness receipt. The support matrix names all five required rows and preserves unverified status; deployment, runbook, quickstart, troubleshooting, monetization, and go-to-market material are present; retained browser, saved-flow, and native companion demos are non-empty. The commercial material remains explicitly hypothesis-only and unpriced. DEL-15 is marked passed without creating platform, signing, buyer, or cost claims that lack evidence.

## 2026-09-06 09:20Z — DEL-06 surface embedding owner integration

Portal SurfaceCatalog now renders provider-opted-in scenario surfaces through the same-origin `/embedded/<owner>/` proxy and reuses EmbeddedScenarioFrame's sandboxed failure boundary. It refuses to invent a URL when the provider does not advertise `vrooli.scenario.embed.v1`. Ten targeted Portal/frame tests, Portal typecheck, and five iframe-bridge hostile-message tests passed. DEL-06 is marked passed; a live provider emitting a scenario embed descriptor and remote process crash journey remain outside this receipt.

## 2026-09-06 09:35Z — DEL-11 optional speech integration

Portal chat now exposes localized assistant-message speech controls. Configured
Audio Tools TTS is used with an AbortSignal; browser `speechSynthesis` remains a
bounded fallback when the provider is absent or unhealthy, and stop cancels both
pending synthesis and playback without changing the message. Existing optional
STT behavior remains covered. Five focused speech/voice tests, twelve chat
workspace/speech tests, and Portal type-check passed. DEL-11 is marked passed
with `portal-speech-integration-receipt-20260906.json`; live Audio Tools audio
bytes remain unavailable on this host.

## 2026-09-06 09:55Z — non-Linux input seam tightened

The macOS and Windows native helper seam now applies bounded wheel events and
rechecks the configured interactive session immediately before validation and
input. Host tests pass, and Windows amd64/macOS arm64 helper-package test
binaries cross-compile. This strengthens the implementation boundary but does
not claim live platform permission or semantic accessibility evidence; those
NAT cases and platform rows remain open.

## 2026-09-06 08:55Z — GNOME Wayland environment probe

A bounded probe successfully started GNOME Shell 46.0 as a nested Wayland
compositor with xdg-desktop-portal and AT-SPI services, but there is no
persistent desktop fixture or portal-backed Device Control helper to execute the
required flow. The result is recorded in
`gnome-wayland-runtime-probe-receipt-20260906.json` as environment evidence
only; NAT-06 and its platform row remain unverified.

## 2026-09-06 09:15Z — explicit semantic capability degradation

The host-desktop strategy now includes `semantic-tree` in capability inventory
and reports it unavailable with a concrete accessibility-adapter next action
when no native semantic provider is provisioned. Wayland inventory uses the same
portal-backed reason and never advertises the capability. Focused host-desktop
strategy tests pass; this makes unsupported behavior observable without claiming
semantic support.

## 2026-09-06 09:30Z — clean-install lifecycle seam

The generated vanilla lifecycle seam now covers a clean install: an empty
install tree creates a fresh client profile and does not inherit predecessor
conversation state. The four lifecycle tests pass, and the lifecycle receipt
now records clean install alongside update continuity, interrupted recovery, and
uninstall retention. This remains filesystem-seam evidence; OS installer,
signing, and cross-platform lifecycle claims remain open.

## 2026-09-06 09:45Z — typed macOS capture permission evidence

Native macOS capture failures now return `ErrPermissionDenied` from the
Screen Recording boundary, so a revoked grant cannot be mistaken for stale
success. A focused regression test and the native/desktophelper/host-desktop
suite pass. Live macOS acceptance still requires a real revoked-grant journey.

## 2026-09-06 10:10Z — DEL-12 lifecycle deliverable reconciled

The five lifecycle acceptance cases are now consolidated into
`desktop-deployment-lifecycle-deliverable-receipt-20260906.json`: clean install,
update continuity, interrupted-update recovery, uninstall retention, and exact
Linux staging artifact identity. The receipt references the independent vanilla
package, lifecycle seam, and digest receipts and marks DEL-12 passed for
`linux-x64-x11`. It explicitly does not claim signed OS installer or
Windows/macOS/Wayland lifecycle support.

## 2026-09-06 10:25Z — focused final regression

A bounded changed-owner regression now passes across Device Control native/helper
and host-desktop packages, Portal chat/speech/embedding/surface/security paths,
generated desktop lifecycle/recovery, and the Program Runtime Portal contract:
27 Portal tests, 8 generated desktop tests, 6 Python contract tests, and the
focused Device Control Go packages. The result is recorded in
`focused-final-regression-receipt-20260906.json`. DEL-16 remains open because
this is not the required whole-collection regression and the four live platform
rows are still unverified.

## 2026-09-06 09:12Z — Windows amd64 staging package generated

The repaired Wine auto-installer now resolves the current x86_64 AppImage release
through the upstream release API, validates downloads, and exposes both global and
local shims. A clean scenario-to-desktop generate/build pipeline completed for
Windows amd64 (pipeline `aef15ddf-b58a-654a-8249-30d6f2fcb772`) and retained
`Vrooli Portal Setup 0.0.1.exe` (80,412,740 bytes,
SHA-256 `957b7286ace260e8faa372e983e4b23a469a135ce4ce753373ab6d77b3af262f`).
The result is recorded in `windows-package-generation-receipt-20260906.json`.
This is packaging evidence only; Windows runtime, semantic accessibility,
permission, and revoke-flow acceptance remain open.

## 2026-09-06 09:25Z — Windows UI Automation semantic adapter implementation

The Windows native backend now uses bounded PowerShell UI Automation calls to
enumerate top-level windows, retain opaque helper-issued semantic element IDs,
resolve unique controls, assert current text, and apply Unicode-rune-position
text insertion through `ValuePattern`. Session checks run before discovery and
input, and semantic observations expire or clear after mutation. The focused
host tests and Windows amd64/macOS arm64 cross-builds pass; evidence is recorded
in `windows-uia-implementation-receipt-20260906.json`. A live Windows fixture,
elevation/IME/scaling checks, and permission recovery are still required before
NAT-03 or the Windows support row can pass.

## 2026-09-06 09:30Z — Windows semantic stale-value protection

The UI Automation observation now retains the observed ValuePattern value and
rejects text assertion/insertion when the live value differs, then invalidates
the semantic cache after a successful mutation. ControlType IDs are retained in
the semantic response. Focused host tests and the Windows cross-build remain
green; the revision is recorded in
`windows-uia-implementation-receipt-20260906-r2.json`. Live Windows fixture and
permission evidence remain required.

## 2026-09-06 09:35Z — Windows catalog lease binding

Windows UI Automation application catalogs are now bound to the observing lease
and expire with their short-lived revision. An application ID or revision from a
different lease, or after expiry, is rejected before semantic discovery. The
focused native tests remain green; this hardening is included in the r2 UI
Automation receipt.

## 2026-09-06 09:40Z — deterministic Windows semantic resolution

The Windows semantic resolver now retains traversal order for ambiguous matches,
so repeated observations produce stable element ordering while IDs remain opaque
and lease/revision scoped. The final focused host/desktop-helper/strategy tests
remain green; the UIA r2 receipt records the current source fingerprint.

## 2026-09-06 09:50Z — macOS empty-capture permission classification

The macOS native capture seam now treats an empty successful `screencapture`
result as permission-denied evidence, matching the existing failed-command
classification. A focused regression and the desktop-helper/host-desktop
packages pass. This improves revoke diagnostics but does not substitute for the
required live macOS grant/revoke journey.

## 2026-09-06 10:00Z — lifecycle helper cross-build confirmation

The lifecycle-managed Device Control desktop-helper executable now compiles
with the updated native backend for Windows amd64 and macOS arm64. The produced
PE and Mach-O binaries are retained in `/tmp` for validation only; this is build
compatibility evidence and does not claim live host execution or permissions.

## 2026-09-06 10:15Z — Windows semantic assertion invalidation

Checked semantic assertions now invalidate the UI Automation observation cache,
matching text mutation behavior and preventing accidental reuse of an old
control reference. Focused native/helper/strategy tests pass; the UIA r2 receipt
contains the updated source fingerprint. Live Windows evidence remains open.

## 2026-09-06 11:00 — Wayland portal backend seam

Implemented `device-control/internal/native/wayland` and integrated explicit `gnome`/`kde` backend selection into the lifecycle-managed helper. Capture now uses the XDG Screenshot portal and validates bounded PNG output; input uses RemoteDesktop portal grants and refuses absolute pointer actions without a compositor stream identifier. Protocol fakes, race validation, changed native package tests, and a Linux amd64 cross-test passed. Receipt: `execution-evidence/wayland-portal-backend-receipt-20260906.json` (source fingerprint `c6901e36e9403e456628867515c9aadb4d38d3d851324d486310b3f8275307d4`). No live GNOME/KDE compositor or user bus is available, so NAT-06/NAT-07 and their support rows remain not-run; semantic accessibility support remains open.

### 2026-09-06 11:20 follow-up

The Wayland backend now issues the generic portal `Session.Close` during helper close, and all `internal/native/...`, desktophelper, and hostdesktop tests pass, including race validation for Wayland/helper packages. Live compositor evidence is unchanged and remains required before NAT-06/NAT-07 can move from `not_run`.

### 2026-09-06 13:00 — named Wayland acceptance fixtures

Added producer tests named `TestAcceptanceNAT06GNOMEWaylandPortalCaptureAndInput` and `TestAcceptanceNAT07KDEWaylandPortalCaptureAndInput`. Both exercise portal capture, granted keyboard input, and coordinate-bound wheel input through protocol fixtures. These strengthen implementation evidence only; the acceptance ledger correctly remains `not_run` until live compositor, permission, and support-row evidence exists.

### 2026-09-06 15:15 — Wayland portal protocol conformance

Corrected the backend to match the portal wire contract: `session_handle` accepts its documented string encoding, RemoteDesktop and ScreenCast selection methods pass `a{sv}` option dictionaries, notification methods include options, wheel uses the `(dx,dy)` axis form, and monitor sources are selected on the shared RemoteDesktop session before `RemoteDesktop.Start`. Protocol fixture tests and the changed native/control/helper suites pass. Live compositor evidence remains unavailable.

### 2026-09-06 10:13Z — Wayland object-path validation hardening

The portal session-handle adapter now validates both native object paths and the
documented string encoding with the D-Bus object-path grammar before binding a
remote-desktop session. A malformed-path regression was added; package tests and
race validation pass. The receipt fingerprint is now
`b7a5ef2949f7485eb6cf2b87a6c4827aaabfb6df46789dcf334071d06f91e004`.

### 2026-09-06 10:15Z — bounded GNOME Wayland runtime probe

A private D-Bus session with an Xvfb backing display started GNOME Shell 46.2;
the nested Wayland socket, AT-SPI registry, GNOME portal backend, and Shell
Screencast service all became available. AT-SPI exposed the screenshot grant,
but the real Screenshot portal request did not complete because the portal
runtime lacked usable PipeWire capture, so this strengthens environment
evidence without passing NAT-06 or DEL-03.
The outcome is recorded in
`execution-evidence/gnome-wayland-runtime-probe-receipt-20260906.json`.

### 2026-09-06 10:36Z — Wayland user-session revalidation

Wayland `Observe`, `Validate`, and `Apply` now recheck the D-Bus user-session
identity before using a portal grant. A changed-session regression is covered
by the focused race suite, and the updated source fingerprint is recorded in
`execution-evidence/wayland-portal-backend-receipt-20260906.json`. This closes
the stale-lease path while live GNOME/KDE acceptance remains unavailable.

### 2026-09-06 10:46Z — macOS Accessibility semantic seam

The non-Linux native backend now supports a bounded macOS System Events
Accessibility seam: application/window discovery, opaque short-lived element
references, exact resolution, and checked rune-position text mutation. The
focused native/helper race suite and macOS arm64 cross-build pass. The receipt
records this as implementation evidence; a real macOS host is still required
for Accessibility/Screen Recording grant, revoke, and artifact acceptance, and
the capture path still needs the plan's ScreenCaptureKit delivery.

### 2026-09-06 10:55Z — macOS revoke regression

Added the named `TestAcceptanceNAT04MacOSRevokeDoesNotReuseStaleCapture`
regression. It observes a successful capture, simulates revocation, and verifies
the next observation returns `ErrPermissionDenied` with no image from the prior
grant. The native race suite passes; this remains deterministic seam evidence
until a real macOS permission-revocation journey is run.

## Semantic invoke continuation — 2026-09-06

The desktop action contract now includes `InvokeAction`, bounded to helper-issued
semantic element IDs and observation revisions. Native host adapters validate the
retained element and execute Windows UI Automation `InvokePattern` or macOS System
Events click commands; the AT-SPI adapter performs an identity-checked
`DoAction(0)` and reports uncertain transport outcomes without retry. Semantic
cache state is cleared after an attempted activation. Named NAT-03 seam coverage,
AT-SPI/desktop-helper tests, race tests, vet, and Windows/macOS helper cross-builds
pass. NAT-03 remains `not_run` because no live Windows fixture is available.

The companion Windows-specific receipt is
`execution-evidence/windows-uia-implementation-receipt-20260906-r3.json`.

## Semantic invoke continuation — 2026-09-06

The desktop action contract now includes `InvokeAction`, bounded to helper-issued
semantic element IDs and observation revisions. Native host adapters validate the
retained element and execute Windows UI Automation `InvokePattern` or macOS System
Events click commands; the AT-SPI adapter performs an identity-checked
`DoAction(0)` and reports uncertain transport outcomes without retry. Semantic
cache state is cleared after an attempted activation. Named NAT-03 seam coverage,
AT-SPI/desktop-helper tests, race tests, vet, and Windows/macOS helper cross-builds
pass. NAT-03 remains `not_run` because no live Windows fixture is available.

## Device Control Test Genie terminal result — 2026-09-06

Run `20260906-110022-34e4e27d` reached a terminal `failed` result after 691
seconds (`provider_unavailable` in the final report): 15 phases passed, 12
failed, and 1 skipped. The failing phases reported existing contracts,
program-runtime fixture, and proto-health provider findings; the run produced
no live Windows, macOS, GNOME Wayland, or KDE acceptance evidence. The run is
retained as a failed validation receipt and does not change the acceptance
ledger's four platform cases or two deliverable gaps.

Fresh validation after AT-SPI invoke compatibility hardening also passes the
native host/Wayland/AT-SPI/helper/session race suite, focused vet, and Windows
amd64/macOS arm64 helper cross-builds. No live-platform acceptance status was
changed.

## DEL-03 ownership boundary — 2026-09-06

The lifecycle ownership check confirms that
`scenarios/device-control/.vrooli/service.json` declares `desktop-helper` as a
`sidecar`, builds `api/cmd/desktop-helper`, and starts it only when
`DEVICE_CONTROL_DESKTOP_HELPER_CONFIG` is set. The lifecycle-managed helper
artifact is present at `api/device-control-desktop-helper`, and the running
Device Control scenario reports the API, helper, and UI processes healthy.
This closes the local ownership/build-delivery question; it does not create
live Windows, macOS, GNOME Wayland, or KDE Wayland permission/effect evidence.
DEL-03 therefore remains `not_run` pending those host journeys.

## Terminator adapter selection — 2026-09-06

Completed the bounded adapter-selection gate for Phase 8. The pinned upstream
Terminator commit `73a381c0c1c33eda55f2c0ecb1d918bf5ec7561a` is MIT-licensed,
but its own support table is Windows-only and it does not provide Vrooli's
helper IPC, signed lease/session binding, or embeddable packaging contract.
The decision record keeps the lifecycle-managed Go helper as the single native
authority; the evaluation is recorded in
`execution-evidence/terminator-adapter-decision-receipt-20260906.json`.
This closes the selection decision only. Live Windows fixture evidence remains
required for Phase 8 and DEL-03.

## CLI contract repair — 2026-09-06

Reconciled Device Control's CLI manifest with its implemented runtime surface.
`device watch` is now declared, the three flow validation flags no longer use
redundant proto binds, and all local desktop owner/helper methods are listed in
`omitted[]` with the explicit authenticated Unix-socket transport reason.
`UPDATE_CLI_EVIDENCE=1 go test ./...` refreshed the primitive evidence and the
targeted `cli-health validate scenario device-control --include-execution`
now passes; only non-blocking architecture/evidence warnings remain. This
removes the prior CLI contract error, but does not change live-platform rows.

The narrow server-owned Test Genie run `20260906-113435-cd91ce53` for the
`contracts` phase completed with `PASS`. It reports CLI/proto coverage and
runtime-surface levels complete; 23 architecture primitive declarations remain
warnings for a later maturity step. This is a valid contract receipt and does
not substitute for the unavailable native platform journeys.

## Dependency reconciliation — 2026-09-06

Scenario Dependency Analyzer reconciled Device Control API's missing local
`github.com/vrooli/cliresolve` replace and normalized the module requirements.
`go mod tidy -diff` is clean, the complete Device Control API package suite
passes, and a follow-up reconciliation reports no candidates. An attempted
lifecycle restart encountered an existing concurrent lifecycle operation, but
authoritative status remains healthy with API, helper, and UI processes running.
This removes the missing-replace dependency finding without changing the
native platform evidence gaps.
## Dependency phase receipt — 2026-09-06

The server-owned Test Genie dependency phase now passes after the governed
Scenario Dependency Analyzer reconciliation and CLI module tidy. Run
`20260906-113903-517c1d77` completed with `PASS`; API and CLI package tests
pass, and `GOWORK=off go mod tidy -diff` is clean. This validates module
parity and does not close the unavailable native platform acceptance rows.
## Lifecycle validation — 2026-09-06

Restarted the dependency and target scenarios through the lifecycle control
plane after the dependency-phase run. `vrooli-bridge` and `device-control`
now report healthy; Device Control is serving API port 16465 and UI port
20698 with its managed processes.
## macOS semantic window-index hardening — 2026-09-06

Corrected the macOS System Events semantic script so each element carries
the index of its owning window record while its opaque runtime ID retains the
OS window ordinal. Multiple-window trees therefore resolve against the right
window instead of accidentally using another element as the window identity.
Added a regression assertion and re-ran the host/native race, vet, and
Windows/macOS cross-build checks successfully.

## Unit validation repair — 2026-09-06

Repaired the Device Control unit gate after isolating runner and test-contract
issues. Trimpath-sensitive Go tests now embed their source/artifact fixtures;
the UI test scopes status assertions to each rendered surface; the testing
manifest identifies Vitest; the governed React Query dependency is pinned to a
single 5.59.0 version; and the English locale no longer contains stray
story-contract keys. The attached-device trust path also falls back to a
closed-list lookup when an optimized reader is unavailable.

With the repository Go toolchain selected explicitly
(`GOROOT=/home/matthalloran8/.vrooli/opt/go/go`), the server-owned Test Genie
unit run `20260906-120207-0f2d314d` completed `PASS` (34 UI test files, 154
tests, 98% statement coverage and 85.04% branch coverage); API and CLI Go
coverage suites, UI type-check, and JSON/diff checks also pass. Dependency
approved validation passes in advisory mode with two pre-existing warnings
(`mdns-go` unrecorded and the `github.com/vrooli/vrooli` range mismatch).
This closes the local unit/contract validation repair, but it does not create
Windows, macOS, GNOME Wayland, or KDE Wayland evidence; DEL-03, DEL-16, and the
final regression remain open.

## Post-repair contracts validation — 2026-09-06

The server-owned Device Control contracts run `20260906-120521-afb76999`
completed `PASS` after the unit-gate repairs. Manifest, proto bindings, runtime
surface, and measures metadata are complete; 23 architecture primitive
undeclared findings remain non-blocking warnings. This receipt confirms the
post-repair contract surface and does not change the native platform gaps.

## Completion audit — 2026-09-06

A fresh ledger audit confirms 89/93 acceptance cases passed, 1/5 primary
platform rows passed, and 14/16 deliverables passed. NAT-03, NAT-04, NAT-06,
and NAT-07 remain `not_run`; the four corresponding Windows/macOS/GNOME/KDE
rows remain `not_run`; DEL-03 and DEL-16 remain `not_run`; and both
`finalRegression` and `planCompletion` remain `not_run`. The evidence verifier
therefore correctly fails on missing receipts rather than treating unavailable
platforms as supported. The current host is Linux x86_64 on a tty session with
no DISPLAY or WAYLAND_DISPLAY, no weston or kwin_wayland, and no matching
Windows or macOS arm64 host in Bridge inventory. These are external evidence
gates, not claims of implementation success.

## Portal owner unit validation — 2026-09-06

The Portal UI composition test now mounts the production `App` directly so it
checks the real provider boundary without adding a second provider tree. Added
focused coverage for optional speech health/transcription paths and deployment
profile validation. Direct Portal UI validation now passes 47 test files and
260 tests with 95.62% statements, 88.34% functions, and 85.19% branches; UI
type-check and targeted ESLint for the changed tests pass. Portal API and CLI
`go test ./...` suites pass with the repository Go toolchain. The attempted
server-owned Portal unit run `20260906-120904-20b89bdc` remained in
`preparing:provider_readiness` after repeated waits and produced no terminal
verdict, so it is not counted as a pass or failure.

## Portal UI lint and i18n repair — 2026-09-06

The Portal UI quality pass now completes without lint errors. Embedded scenario
failure copy is routed through the i18n registry, locale catalogs are aligned
with actual callsites, the companion annotation labels use explicit generated
keys, and tests use stable copy constants. A runtime guard preserves safe
speech cancellation when browser speech synthesis is absent. Final direct UI
checks pass: `pnpm test:coverage` (47 files, 260 tests, 95.62% statements,
88.34% functions, 85.19% branches), `pnpm run type-check`, and `pnpm run lint`.
Lint emits one existing non-blocking Fast Refresh warning for the exported
transcript helper. Portal native-platform and final regression evidence remain
unchanged.

## Portal governed unit receipt — 2026-09-06

After correcting the Portal UI testing framework declaration and documenting
provider-free feature harnesses, the server-owned Test Genie Portal unit run
`20260906-122742-3c4818d2` completed `PASS`. Framework, surface discovery,
execution, and coverage gates are now accepted. Unit Health retains advisory
architecture findings for injectable ambient dependencies and the Portal CLI
test utility, plus low-coverage/no-assertion observations; none are blocking.
The earlier provider-readiness and projection-drift failures are retained as
historical failed runs and are superseded by this receipt.

## Portal CLI contract repair — 2026-09-06

Reconciled Portal CLI bindings after the contract phase identified unmapped
arguments. The surfaces resolve label now binds to the proto `display_label`
field; exact SurfaceRef flags and context file/path flags are explicitly local
handler inputs with documented waivers. `cli-health validate scenario portal
--include-execution --json` passes, the CLI Go suite passes with regenerated
primitive evidence, and the server-owned Test Genie contracts run
`20260906-123224-826ac570` completed `PASS`. The run retains 28 non-blocking
architecture primitive warnings for a later maturity step. This closes the
Portal manifest/proto binding gap without changing the unavailable native
platform rows.

## macOS Bridge availability probe — 2026-09-06

Bridge inventory confirms the connected `minimouse` node is online and
dispatchable, but it is `darwin/amd64` rather than the required arm64 row. Its
authoritative host inventory reports no attached display and no interactive
user session; system Screen Sharing is active but does not provide the bound
desktop session required by the helper. `xcodebuild` readiness also fails. A
typed `host inventory` dispatch passed (`bcbdbbc5-f123-48d4-b463-7b9bea5755be`),
while starting Device Control failed during dependency rollback. The probe is
recorded in `macos-node-availability-probe-receipt-20260906.json` and confirms
that NAT-04 and the macOS platform row remain unverified for environmental
reasons.

## Darwin ScreenCaptureKit seam — 2026-09-06

Added a Darwin cgo Objective-C seam that captures a permitted display frame
through ScreenCaptureKit, converts the first frame to bounded PNG bytes, and
returns typed failure without stale image reuse. The Go helper prefers this
native path when built with the Apple framework SDK and retains the existing
injected/`screencapture` compatibility path for older packaged helpers. Native
preference and fallback tests pass; Go vet and Windows/Darwin non-cgo
cross-compilation pass. The current Linux host has no Darwin clang or Apple
framework SDK, so the cgo path and live permission journey remain unverified.
Cross-platform non-cgo packaging still uses the compatibility path until a
Darwin release helper is built with the Apple SDK and cgo enabled.
Receipt: `screencapturekit-seam-implementation-receipt-20260906.json`.

## Acceptance ledger evidence reconciliation — 2026-09-06

The four unavailable native cases now carry their focused implementation or
runtime probe receipts, source fingerprints, observed timestamps, support-row
links, and explicit limitation notes. The four unverified platform rows now
carry the observed environment and probe references while retaining empty
artifact digests and `not_run` status. The verifier therefore reports only the
real missing platform artifacts and final-regression/completion receipts; no
unavailable probe is promoted to a passing acceptance result.

## Activation image session-transition guard — 2026-09-06

The native activation image path now checks the bound desktop session after
the screenshot operation as well as before and after the metadata probe. A
session transition during capture is rejected before PNG decoding or image
publication. `TestActivationImageRejectsSessionChangeAfterCapture` passes in
the focused race suite, and the ScreenCaptureKit seam receipt was refreshed
with the current implementation fingerprint. This strengthens the local
session-isolation guarantee but does not create the unavailable Windows,
macOS, or Wayland live evidence required by the acceptance ledger.
