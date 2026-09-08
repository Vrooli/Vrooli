# Execution support inventory — 2026-09-05

This is host-discovery evidence, not a product support claim. Preserve the five
primary rows from the acceptance ledger. Exact OS builds, compositor versions,
package profiles, installed artifact digests, and permission receipts must be
recorded by the live producer before any row passes.

| Required row | Candidate host | Observed facts | Remaining evidence |
|---|---|---|---|
| windows-x64 | None in current Bridge inventory | `vrooli-bridge nodes list` returned only Darwin and Linux nodes. | Host access, exact Windows build, helper/session permissions, all profiles and live acceptance. |
| macos-arm64 | No matching architecture or interactive desktop reported | `minimouse` (`25c7e426-c76c-421a-8351-aaf964589802`) is online but reports machine and binary architecture `amd64`, no attached display, no interactive user session, and failed `xcodebuild` readiness. | Identify an arm64 host with an attached, permissioned user session; do not substitute amd64/system Screen Sharing evidence. |
| linux-x64-x11 | `swarminator` (`697b6224-6283-4a31-90e2-73724e424c05`) | Ubuntu 24.04.4 LTS; loginctl session 2 is active, local, type x11, user matthalloran8. | Exact display/compositor facts, authenticated helper, permissions, package profiles and live acceptance. |
| linux-x64-gnome-wayland | None verified | Current desktop session is X11. | Provision or identify a named GNOME Wayland session and record exact versions; live native and package acceptance. |
| linux-x64-kde-wayland | None verified | No KDE Wayland session has been identified. | Provision or identify a named KDE Wayland session and record exact versions; live native and package acceptance. |

Additional useful targets: the connected Darwin/amd64 node can provide secondary
macOS development evidence. Bridge attached inventory reports a reachable Galaxy
A03s (USB) hosted by swarminator. Neither proves the desktop primary rows.

Read-only reproduction:

```bash
vrooli-bridge nodes list
vrooli-bridge nodes get 25c7e426-c76c-421a-8351-aaf964589802 --json
vrooli-bridge attached list
loginctl show-session 2 -p Type -p Desktop -p State -p Remote -p Name
cat /etc/os-release
```

The operator was asked which existing test hosts to use. Implementation continues
independently. No OS permission bypass, machine unlock, purchase, public release,
or platform waiver has been requested or performed.

## Local X11 Stop timing evidence — 2026-09-05

The managed development helper on swarminator completed 50 physical control
lease trials after completed capture. Installed CLI dispatch-to-Stop-receipt
p95 was9.891ms, maximum18.173ms (target250ms). Every trial checked durable lease
clearance, no pending release, no active grants, and refused post-Stop input.
See live-cli-stop.json for samples and running API/helper binary digests.
This supports the local idle-after-capture cohort only; it does not pass the
whole support row, companion package, concurrent mutation, or remote acceptance.


## Implementation update — 2026-09-06

The Device Control helper now has a runtime-selected macOS/Windows capture and
input seam and lifecycle session monitoring, with Windows x64 and macOS arm64
cross-build receipts. This does not change the live inventory above: no primary
Windows or macOS arm64 host was available, and no GNOME or KDE Wayland session
was identified. The host-desktop strategy and helper explicitly refuse X11 or
XWayland fallback when a Wayland session is active.


## Darwin lifecycle attempt — 2026-09-06

The control-plane command `vrooli scenario start device-control --node minimouse
--timeout 60 --json` exited without a start receipt; the subsequent signed
Bridge status call reported `device-control` stopped. A direct `device-control
device list` relay likewise returned the stopped-scenario diagnostic. No Darwin
helper or permission operation was inferred from these failures.
