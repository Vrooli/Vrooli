# Device identity, the size lease, and the devices/machines surface

Durable records for the multi-device work in Web Console. Authored 2026-08-29 on
branch `agi`. These are the source artifacts the implementation plan
`web-console-device-identity-lease-and-fleet-surface` was written from.

| File | What it holds |
| --- | --- |
| [`audit-2026-08-29.html`](./audit-2026-08-29.html) | Defect audit. How device identity, the size lease and follower presentation actually work today; six traced defects (D1-D6) with file:line evidence and a fix sketch each; the debt and test-topology findings. |
| [`devices-and-machines-mockups.html`](./devices-and-machines-mockups.html) | Static render of all five design artboards: Today, Proposed, Device card states, Narrow, Layout alternatives. Opens offline in any browser. |
| [`mockups/`](./mockups/) | Editable artboard sources (`*.dc.html`) plus `canvas.json`. One file per artboard. |

Live editable canvas (same content as the static render):
<https://claude.ai/code/artifact/ba410e76-bb10-4b63-9179-abab20e36fd5>

Defect audit as a hosted page:
<https://claude.ai/code/artifact/33edd54f-9468-4450-8fa4-eb72b4d39279>

## The one-sentence finding

Device identity is per device (a `localStorage` UUID). Control authority is per
socket (`leaseOwner chan []byte`). Nothing reconciles the two, so a device can
become its own remote.

## Predecessor

The size lease and the device silhouettes were built by plan
`web-console-multi-device-terminal-sizing-size-lease`
(id `d9de66e9-80b5-476d-8765-d8010e10c2e1`, 10 phases, all `done`). That plan is
correct and is not superseded. This work repairs the seams it left open and adds
the operator surface it never had.

## Design decisions that carry forward from the predecessor

- One tmux attach process per session, so per-device ideal sizing is physically
  impossible. A lease is the only workable policy.
- The device archetype is declared from `screen`, never from `navigator.userAgent`
  and never from the live terminal grid.
- `terminal_ws` is a documented Connect exception (`docs/internal/SEAMS.md`), so
  JSON frame fields on that socket are correct and need no proto change.

## Shipped state

The browser declares a stable local device id, label, and screen family on every
terminal WebSocket connection. The server groups live sockets by that identity,
reclaims the size lease for a returning device after a liveness probe, and marks
stale sockets for supersession. Lease and follower state are reducer-owned in the
UI, so a follower cannot accidentally resize the shared PTY or echo its own
presentation.

The `DeviceService` roster is a live snapshot backed by the existing lifecycle
event stream. The Devices & machines drawer shows connected browser devices and
linked terminal machines as separate horizontal rails. Device cards expose
recognition-only labels and safe actions; the caller's own device cannot be
disconnected by the service.

### Audit disposition

- Shipped: D1-D6 (identity/lease reconciliation, reconnect reclaim, follower
  self-echo, touch target coverage, role-transition sizing, and pane stacking).
- Shipped: the device roster, lifecycle deltas, shared fleet card grammar, and
  narrow horizontal overlay layout.
- Deferred: persistence for devices that have no live connection. The roster is
  intentionally live-only; last-seen history would require a separate durable
  ownership decision.
