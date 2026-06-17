# Flows — Device Sync Hub

This document is the canonical workflow and state-transition map for
the scenario. Use it when behavior depends on ordered states, retries,
cancellation, stale completion, background jobs, presence, or mutually
exclusive UI modes. Owning domains mirror [`DOMAINS.md`](DOMAINS.md).

These flows describe the **intended** design (Phase 1). No flow is
modeled in code yet; the maturity ladder and production shape below
describe how each flow will be hardened as it lands.

## Purpose Of This Document

Use this document to answer:

- Which user/system workflows matter?
- Which workflows have explicit states and events?
- Which transitions are illegal?
- Which flows are known but not modeled yet?

Plain CRUD with no meaningful ordering constraints does not need a
workflow model.

## Flow Inventory

| Flow | Domain | Trigger | Outcome | Statefulness |
|---|---|---|---|---|
| Device pairing | devices | New device redeems a code/QR, or requests approval. | Device joins the owner's trust group. | Stateful: TTL, single-use, approve/reject branches. |
| Send item (file/text) | transfer | Trusted device uploads a file or text item. | Item stored and fanned out to receivers. | Stateful: streaming/chunked upload, store, fan-out. |
| Pull / download | transfer | Receiver fetches an item it has access to. | Bytes streamed back with original filename. | Lightly stateful: ACL check then stream. |
| Retention purge | transfer | Delivery (Live) or scheduler tick (Held). | Item + blob removed. | Stateful: Live/Held/Pinned lifecycle. |
| Device revocation | devices + auth | Owner revokes a device. | Device loses all access immediately. | Stateful: atomic session-kill + trust drop. |
| Presence | realtime | Device connects/disconnects WebSocket. | Online/offline visible to other devices. | Ephemeral state machine. |

## Flow Details

### Device pairing

- Owner domain: devices.
- Primary path (code/QR): the owner's existing trusted device (or the
  owner via UI) issues a short-TTL, single-use pairing code (rendered
  also as a QR). The new device submits the code; on a valid, unexpired,
  unused code the device is admitted to the trust group and the code is
  burned.
- Fallback path (request→approve): the new device requests access; a
  `pairing-request` banner is pushed over the WebSocket to the owner's
  online devices; the owner approves or rejects. Approve admits the
  device; reject discards the request.
- States: `idle → code_issued → redeemed → trusted`, with the parallel
  `requested → approved|rejected` fallback branch.
- Illegal transitions: redeeming an expired or already-used code;
  admitting a device without either a valid code or an explicit approve;
  trusting before owner identity is validated.
- Failure modes: expired/used code, code-issue rate limit exceeded,
  approval timeout, authenticator unavailable (fail closed — no pairing).
- Requirements: OT-P0-002, OT-P0-008.

### Send item (file/text)

- Owner domain: transfer.
- Trigger: a trusted device uploads a file (multipart, streamed) or
  posts a text item — both first-class with no feature disparity.
- Steps:
  1. Auth middleware validates the token; devices layer confirms trust.
  2. Quota check (per-device + global) **before** any bytes are
     written; reject if it would breach.
  3. Stream bytes to the blobstore (never buffer whole files); large
     files use chunked, resumable upload so an interruption resumes
     rather than restarting.
  4. Generate an image thumbnail server-side at ingest (for images).
  5. Persist the `items` metadata row (kind, name, mime, size, refs,
     retention stamped from the settings default, target).
  6. Fan out an `item-arrived` event over the WebSocket to the target
     audience: the whole trust group (broadcast default) or one chosen
     device (directed, OT-P1-003).
- Outputs: proto-typed item metadata, plus a live event on receivers.
- Failure modes: quota breach, blob write failure, metadata persist
  failure, no online receivers (item still stored per retention).
- Retry/cancel: chunked upload resumes from the last good chunk; a
  failed metadata persist after a blob write triggers blob cleanup.
- Requirements: OT-P0-001, OT-P0-004, OT-P0-007, OT-P1-001, OT-P1-002,
  OT-P1-003.

### Pull / download

- Owner domain: transfer.
- Trigger: a receiver lists/opens an item it is entitled to (broadcast
  to the trust group, or directed to that device).
- Steps: auth + trust check → per-item ACL check (is this device a
  valid target?) → stream blob bytes back preserving the original
  filename. Text items return their snippet directly.
- Illegal: a device pulling an item not targeted at it, or after its
  trust was revoked.
- Side effect: for a `Live` item, successful delivery to all connected
  targets triggers purge (see below).
- Requirements: OT-P0-001, OT-P0-007.

### Retention purge

- Owner domain: transfer; default sourced from settings.
- Lifecycle by policy:
  - **Live** — deliver-then-purge: once delivered to all connected
    target devices, the item and its blob are removed.
  - **Held** — auto-purge after the configured default (24 h
    out-of-the-box) when the scheduler tick finds `expires_at` passed.
  - **Pinned** — never auto-purged; removed only on manual delete.
- States: `stored → (delivered | expired | pinned) → purged`, where
  `pinned` never advances without an explicit delete.
- Enforcement: a background purge scheduler with an injected clock for
  deterministic tests; purge removes metadata row and blob together.
- Requirements: OT-P0-004.

### Device revocation

- Owner domains: devices (trust) + auth (session).
- Trigger: owner revokes a device from settings/devices.
- Steps (must be atomic in effect): drop the `devices` row from the
  trust group **and** ask `auth` to invalidate the device's
  `scenario-authenticator` session. Any partial failure must leave the
  device **locked out**, never half-trusted.
- Outcome: the revoked device can perform no further reads or writes;
  its in-flight WebSocket is dropped on the next auth/trust check.
- Illegal: a revoked device's cached validation permitting a read/write
  after revocation (fail-closed cache window default ≤ 60 s, and a
  stale cache must never re-admit a revoked device).
- Requirements: OT-P0-003, OT-P0-008.

### Presence

- Owner domain: realtime.
- Trigger: a device opens/closes its authenticated WebSocket.
- States: `offline → online → offline`; transitions broadcast to the
  owner's other online devices so the UI shows live presence (color +
  icon/label, never color alone).
- Storage: ephemeral, in-memory (optionally Redis for multi-instance);
  rebuilt as devices reconnect after a restart.
- Requirements: OT-P0-005, OT-P1-005.

## State Machines

| Domain/Flow | States | Illegal Transitions |
|---|---|---|
| devices / pairing | idle, code_issued, requested, redeemed, approved, rejected, trusted | redeem expired/used code; trust without code-or-approval; trust before auth validation |
| transfer / send | received, quota_ok, bytes_stored, metadata_recorded, fanned_out, failed | metadata before bytes; fan-out before persist; write before quota check |
| transfer / retention | stored, delivered, expired, pinned, purged | Live escaping purge after delivery; Pinned auto-expiring; purge of blob without metadata |
| devices / revocation | trusted, revoking, locked_out | locked_out → trusted without re-pairing; read/write while revoking |
| realtime / presence | offline, online | presence event for an untrusted/unauthenticated session |

## Maturity Ladder

Temporal workflows mature in layers. Do not skip the executable layers
to add a standalone formal document. The stateful flows above (pairing,
send, retention, revocation) are the candidates to drive to Level 5.

| Level | Name | What exists |
|---|---|---|
| 0 | Unmodeled risk | Lifecycle behavior exists only inside handlers, components, callbacks, or jobs. |
| 1 | Inventory | The flow is listed here with owner, source links, risk, and next step. |
| 2 | Workflow model | State/status values, event values, `Transition`, and `CheckInvariants` live beside the owning domain or feature. |
| 3 | Matrix + traces | Tests cover every state/event pair and replay representative traces against production transition logic. |
| 4 | Declarative contract | A domain-local `*.flow.json` declares states, events, transitions, invariants, and named traces. |
| 5 | Checked formal model | Quint/TLA+ or an equivalent tool is generated from the contract, checked, and replayed by production tests. |

## Production Shape

Every flow lives in a `flow/` subdirectory next to its consumer with
conventional file names. The `*.flow.json` contract is the source of
truth (schema v6); the `flow-verifier` scenario CLI regenerates and
checks the Quint model and formal artifacts, and the scenario lifecycle
runs `make temporal-models` before the test suite.

API domains that own durable lifecycle state use:

```text
api/internal/<domain>/
  flow/
    flow.json                   # hand: source of truth (schema v6)
    transition.go               # hand: wrapper (package flow)
    flow_test.go                # hand: thin replay delegation (package flow)
    generated/
      model.qnt
      artifact.json
      runtime.go                # package generated
      replay.go
```

UI features that own client-side modes use:

```text
ui/src/features/<domain>/
  flow/
    flow.json                   # hand: source of truth (schema v6)
    transition.ts               # hand: wrapper
    fixtures.ts                 # hand: replay fixtures
    flow.test.ts                # hand: thin replay delegation
    generated/
      model.qnt
      artifact.json
      runtime.ts
      replay.helper.ts
```

The workflow owns state/status values, events, `Transition`, and
`CheckInvariants`, and should be pure or nearly pure. Effects live
outside the workflow behind seams: repositories, BlobStore, the
`AuthClient`, clocks, timers, and UI API modules. To scaffold a new
flow:

```bash
flow-verifier flows new ui/src/features/<feature> --flow-id <flow-id> --lang ts --root .
flow-verifier flows new api/internal/<domain>     --flow-id <flow-id> --lang go --root .
```

## Deferred / Unmodeled Flows

| Flow | Risk | Next Step |
|---|---|---|
| WebRTC same-LAN fast path | P2 speed path; relay covers all cases today. | Model only if/when OT-P2-001 lands; keep the transfer byte-flow seam ready. |
| Multi-owner trust handoff | Out of scope in single-owner v1. | Revisit with OT-P2-002. |

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — owning domain map
- [`ARCHITECTURE.md`](ARCHITECTURE.md) — request path and seams
- [`DATA.md`](DATA.md) — persisted state, retention, and quotas
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — authenticator and presence dependencies
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — side-effect boundaries
- [`../internal/TESTING.md`](../internal/TESTING.md#temporal-workflow-tests) — matrix and trace testing
