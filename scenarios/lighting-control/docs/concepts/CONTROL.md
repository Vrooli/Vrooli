# Lighting control domain

The control domain owns durable controller enrollment, zone identity mappings,
desired zone state, manual Off overrides and truthful command outcomes. Its
wire contract is `packages/proto/schemas/lighting-control/v1/control/control.proto`;
the generated Connect service is the only API and CLI transport.

## Backend boundary

OpenRGB is an optional resource. The API creates an SDK client only when the
resource exports `RESOURCE_PORT_SDK`; it always dials literal loopback and
never starts, installs or probes the resource during API startup. An absent
export reports a disabled backend without network activity. Explicit controller
listing and reconcile requests perform bounded SDK discovery. Repeated zone
commands reuse that discovery for five seconds, so changing a color does not
scan hardware for every update. A reconcile refreshes controller state.

Only SDK protocol v6 or newer is accepted because it supplies stable unique
controller IDs. Enumeration positions are transient and are never persisted.
Zone IDs are derived from the controller ID and a normalized zone name plus LED
count. If two current zones share that fingerprint, control fails closed because
the backend does not provide enough information to select one durably.

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
not discard intent. Reconcile later retries each saved zone once and records
the new outcome.

Manual Off is stored independently with controller scope, creation time and
optional expiry. While active, it has priority over policy or direct zone
requests: those requests remain persisted and return `deferred` until Off is
cleared or expires. Clearing Off triggers reconciliation. An expired override
is returned with `active=false` and its expiry timestamp remains visible.

## Operation truth and current limits

`local_write_completed` means the complete `UpdateZoneLEDs` packet was written
to the local SDK socket. For protocol v6, a correlated ACK contains the
acknowledged packet ID and a backend status. `backend_acknowledged` means that
ACK was received; status zero means OpenRGB processed the request successfully.
This reports backend processing only. It does not prove cached controller state,
device application, or visible output. A complete write without a valid ACK is
reported as `command_sent` with backend processing unknown. Observation fields
remain unset until a separate observer exists.

Static color is available only for a zone whose SDK v6 mode list contains a
`Static` mode. Brightness scales the requested RGB channels before the packet
is written. Effects remain unsupported in this control slice; clients receive
an explicit unsupported per-zone outcome and policy can apply its documented
static fallback. There is no frame-rate or background device scan in this
domain yet. Automatic boot/resume/backend restart reconciliation and physical
output qualification remain E3 work.

## Verification

The SDK adapter follows release_1.1 packet IDs: controller count 0, controller
data 1, ACK 10, protocol negotiation 40, server-name notification 51, and
UpdateZoneLEDs 1051. Protocol negotiation returns packets 40 and 51 and has no
ACK; controller count has no ACK; controller-worker operations return correlated
ACKs for protocol v6. The adapter drains unrelated notifications while waiting
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
