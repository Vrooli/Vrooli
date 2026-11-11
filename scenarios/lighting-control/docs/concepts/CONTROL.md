# Lighting control domain

The control domain owns durable controller enrollment, zone identity mappings,
desired zone state, manual Off overrides and truthful command outcomes. Its
wire contract is `packages/proto/schemas/lighting-control/v1/control/control.proto`;
the generated Connect service is the only API and CLI transport.

## Backend boundary

OpenRGB is an optional resource. The API starts with an unbound runtime client,
and the native control-plane owner binds its verified loopback endpoint and
process incarnation through the operator-private current-process transport.
There is no inherited SDK endpoint environment variable. API startup never
installs, starts or probes the resource. An unbound client reports disabled
without network activity. Bounded SDK discovery after explicit binding lists
backend controllers; it is distinct from a physical detector rescan. Repeated
zone commands reuse the five-second discovery cache. Reconciliation refreshes
controller metadata.

Only SDK protocol v6 or newer is accepted. Its per-operation controller handle
is transient and is never persisted as identity. Durable IDs are derived from a
controller serial number, or from its OpenRGB-reported device location when no
serial is available. If neither is present, or two discovered controllers report
the same identity, the controller remains visible but cannot be enrolled or
controlled. Reconnects resolve the durable ID to the fresh SDK handle before a
write. Zone IDs are derived from the durable controller ID and a normalized zone
name plus LED count. If two current zones share that fingerprint, control fails
closed because the backend does not provide enough information to select one
durably.

The client has no lifecycle behavior and connects only to the resource's
loopback SDK socket. Each bounded operation uses a fresh TCP connection and
negotiates protocol v6 on that connection before sending controller data or
commands. OpenRGB stores the negotiated version in connection-local client
state. Readiness, device discovery and user control are separate:
a reachable protocol endpoint does not prove a device exists, a device list
does not prove permission to operate, and a successful socket write does not
prove the controller applied or displayed the color.

## Persistence and priority

Controller enrollment, zone mappings, desired protobuf messages, last command
results and the manual Off override live in the scenario's SQLite database.
The schema is idempotent and applied through the API module registry. Desired
state is written before attempting hardware output, so backend downtime does
not discard intent. Each new request records manual or automatic ownership.
Direct per-zone settings take precedence on every automatic policy tick.
Independent startup/reconnect reconciliation replays proven manual settings and
the durable Off fence. Automatic pending output awaits fresh source evaluation
by the policy owner, preventing a prior pulse from replaying after its source
evidence expires. Command results describe the request actually applied.

Explicit global preset selection releases manual zone ownership for enrolled
controllers and replaces manual Off. Before releasing ownership, one SQLite
transaction archives the previous manual/legacy state and its last command
bytes in `lighting_previous_zone_state`, changes ownership, publishes or releases
Off, and records the completed selection in `lighting_policy_selection`.
The latest archived manual state per zone survives further policy selections.
Ordinary ticks preserve manual ownership and never perform that adoption.

Before saving a new pending selection, policy captures the control owner's
persisted override revision under its existing writer lock. Each Off publication
or release advances that numeric revision in the same transaction as the
mutation. Selection compares its captured revision with the active Off's causal
order inside the ownership transaction. A later direct Off supersedes the
pending selection even when wall-clock timestamps tie or move backward. A new
explicit selection captures an existing Off and can replace it at the same
clock time. Wall-clock timestamps continue to govern expiry and display only.
The revision also records the causing selection identity, so an Off selection's
own committed publication does not supersede its acknowledgement retry.

An Off arriving after selection stays active during policy checkpoint and
evaluation. Retrying retains the original request time and expiry, including an
Off fence's creation time. An older timed Off selection cannot replace newer
indefinite or scoped Off. Legacy pending selections without a captured revision
preserve active Off because their causal order is unknown. A new explicit
selection captures the current revision and can replace that legacy Off.

Policy saves a unique selection identity before requesting control mutation.
Control retains the latest completed identity because policy has one serialized
pending selection. After a lost policy acknowledgement, the same completed
identity returns without rearchiving or adopting a newer manual zone, restoring
a cleared fence, or replaying earlier hardware output. Receipt persistence failure
rolls back ownership, archive, override and causal revision together. Backend effects follow
the commit; ordinary reconciliation/fresh evaluation retries output without
repeating adoption. Legacy pending selections checkpoint a new identity before
their first transition. No identity is inferred for ambiguous legacy zone rows.

Databases from before request ownership have ambiguous desired rows. Their
original bytes remain intact and ordinary reconciliation skips them. Reads
retain the saved values and expose review/adoption guidance. Existing Save &
apply explicitly adopts a zone as manual. Existing Use preset explicitly
archives and adopts enrolled zones for the selected policy. A new additive
ownership table is applied by the existing domain schema provider. No legacy
row is assigned a guessed origin.

Manual Off is stored independently with controller scope, creation time and
optional expiry. While active, it has priority over policy or direct zone
requests: those requests remain persisted and return `deferred` until Off is
cleared or expires. Clearing Off triggers reconciliation. An expired override
is returned with `active=false` and its expiry timestamp remains visible.
Older Off records remain authoritative even when their underlying zone rows
have ambiguous ownership. Expiry resumes proven manual bases, while automatic
zones await fresh evaluation and ambiguous legacy zones still require review.

## Maximum brightness

`PolicyConfig.brightness_cap_percent` is the single maximum for manual static,
manual effects and automatic policy output. Its default is 35 percent. Control
reads the existing policy store immediately before emission under its serialized
writer. A missing policy row uses the default; a failed or invalid policy read
defers illuminated output. Manual Off can still be emitted when policy storage
is unavailable.

The cap changes the emitted clone and last-command result, while the saved
manual desired state retains its original brightness. Raising a cap can therefore
restore the operator's saved level. A zero cap emits Off without publishing or
clearing a manual Off override. Changing the cap commits in order with direct
writes, reapplies manual state, and evaluates fresh policy evidence. Ordinary
preset definition edits remain drafts. Neither path adopts manual ownership or
releases an active Off fence.

Static brightness scales RGB channels. Native effects require a usable advertised
brightness range for limits below 100 percent; the raw level is the rounded-down
percentage of the advertised maximum. A requested level below the hardware
minimum, a fixed/reversed range, or a missing brightness control prevents an
effect write. Control then emits a safely scoped static Off fallback, including
when lowering the cap after an unbounded effect was already enabled. The result
describes Off while persisted desired effect intent and ownership remain intact.
If the hardware has no safely scoped static Off, the unsupported result states
that no Off command completed and prior output may remain unchanged. Exact zone
effects can work without static color support when their brightness control
honors the limit. SDK acknowledgement never proves visible brightness.

## Operation truth and current limits

`local_write_completed` means at least one complete mutation packet (zone
resize, mode update, or zone colors) was written to the local SDK socket. It
can remain true after a later step fails. The reason records completed resize
acknowledgement and unconfirmed rendering when those facts differ. A correlated
v6 ACK includes the packet ID and backend status. `backend_acknowledged` confirms
the requested rendering operation; an earlier resize ACK alone cannot establish
it. Nonzero rejection status is reported as failure. A completed local write
without the required rendering ACK is `command_sent`, with backend processing
of that render unknown. These facts do not prove cached state, device application,
or visible output. Observation fields remain unset until a separate observer
exists.

Static and native effect requests use parsed SDK v6 mode descriptors and
controller/zone scope checks. Controller-wide mode changes cannot overwrite
unrelated zones. Advertising a mode alone does not qualify physical output.
Brightness scales requested RGB channels, and unsupported effects return an
explicit per-zone outcome so policy can use its static fallback. Native mode
ACK and fresh same-connection controller readback are distinct from separately
observed physical output. The bounded reconciliation cadence does not perform
frame-rate physical detector scans. Physical output qualification remains open.

## Verification

The SDK adapter follows release_1.1 packet IDs: controller count 0, controller
data 1, ACK 10, protocol negotiation 40, server-name notification 51, and
UpdateZoneLEDs 1051. The exact pinned release protocol negotiation returns packets40 and51 followed
by ACK10 correlated to40; controller count replies with0 and ACK10 correlated
to0. Controller data1, mode1101/1103 and zone colors1051 require their correlated
ACKs. Known asynchronous updates may interleave, while malformed, unknown and
out-of-sequence frames fail closed. The adapter drains unrelated notifications while waiting
for the expected response and ACK.

Run focused owner tests:

```bash
cd scenarios/lighting-control/api && go test ./internal/control ./handlers/control ./internal/modules
cd scenarios/lighting-control/cli && go test ./domains/control ./domains
```

TCP fixtures encode literal IDs, payloads and response ordering from the pinned
release_1.1 server/client sources. They exercise protocol negotiation, stable
v6 IDs, zone capability parsing, correlated and failed ACKs, missing ACK after
a complete write, truncated and oversized frames, timeouts, loopback
enforcement, controller and zone writes. They do not qualify an operated SDK
server, real devices, or physical output.

## Explicit addressable zone lengths

Addressable controllers can advertise zero LEDs until their physical wiring is configured. `ControlService.ConfigureZone` (`lighting-control control configure-zone --controller-id … --zone-id … --led-count …`) requires existing controller enrollment, authenticated write authority, an unambiguous stable zone, and finite SDK v6 normal-size capabilities. Zero, effects-only size flags, inverted/unbounded limits, values outside the fresh controller range, and aggregate lengths outside the SDK's 65535-LED controller capacity are refused. The ASUS Aura Terminal's pinned 1.1 implementation advertises 0–500 per addressable header; this is source applicability, not hardware qualification.

The scenario stores the desired physical count separately from replaceable discovery rows. A configuration save preserves manual priority and manual Off; a new zone starts with durable Off. Existing policy-owned output continues through the normal policy tick rather than replaying a cached automatic output as manual intent. Per-zone `setup_needed`, current `led_count`, `configured_led_count`, and advertised limits distinguish selection, backend configuration, and output. Saving a count is not a physical output acknowledgement.

Before rendering, the authenticated private backend resolves the controller and zone on the current SDK connection, checks live limits, sends packet1000 (two little-endian signed-positive32 values: transient zone index and requested count), waits for the correlated v6 ACK, then refreshes controller data and resolves stable identities again. A missing/mismatched length readback prevents color output. Desired lengths replay after backend reconnect/resume through the existing writer. OpenRGB may save its role-private configuration; this does not write controller firmware.

Resizable zones use normalized name identity independent of current LED count. Fixed zones keep their original fingerprints. Discovery preserves an existing unambiguous legacy zone ID by normalized name before replacing its inventory row, retaining stored state/ownership/selection IDs verbatim. Ambiguous legacy names fail closed rather than silently binding another physical header. Renaming a header outside Light Desk requires deliberate identity review.

A fresh already-active controller Direct/per-LED mode permits scoped static colors and software brightness. Selecting a global Direct mode is allowed only during an explicit whole-controller manual Off with every current zone count known; multi-zone native effects remain unavailable without a controller-wide effect contract. Unknown/empty zones cannot establish whole-controller mode authority. A failed zone or partial Off result remains visible and never becomes an applied/output claim.

Current proof is hermetic SDK, database, public API/CLI and UI behavior only. Native access, actual SDK/controller behavior, observed static/Off light output, hardware reconnect and logged-out boot remain the parent delivery's separate physical qualification gates.

## Inspecting saved inventory while offline

`ListControllers(include_unavailable=true)` reads the existing canonical roster when SDK discovery fails, including after native quiescence. The response retains the unavailable/error backend status; enrollment, wiring, desired policy, manual Off and prior last-seen/capability observations are preserved without an inventory sync or a hardware command. The default current-only request still returns no controllers after failed discovery. A failed persisted-roster read is an error, not an empty successful inventory.

The public CLI can inspect this response with `lighting-control control controllers --include-unavailable true --json`. Light Desk also requests unavailable inventory and treats failed discovery as authoritative even if a separate protocol probe succeeded. It identifies saved capability information as last known, disables controls that need a ready backend, and keeps durable manual Off available. Reconnect uses the existing fresh identity/capability checks and normal reconciler; saved inventory never authorizes a stale SDK handle or proves current hardware output.

### Displaying manual Off scope

The public manual override distinguishes `all_enrolled` from an explicit list of controller IDs. A listed controller is covered across all its zones, including zones returned after reconnect; other controllers retain their independent desired state. The UI names the scoped controllers (with stable IDs to disambiguate names), reports unknown scope explicitly and keeps requested Off separate from physical output evidence.

Dashboard, Controller detail and Presets Off actions request **all enrolled controllers**. Only an active explicit all-enrolled override makes those actions redundant. A subset, an unknown scope, or even a list of every currently enrolled ID leaves the all-enrolled action available: future enrollment is a different scope. Expanding Off goes directly through `SetManualOff(all_enrolled=true)` without first clearing the existing protection. Backend availability does not block durable Off; editing and configuration still require current ready evidence. Controller desired-policy status applies Off only where that controller is covered. API scope evaluation remains authoritative.
