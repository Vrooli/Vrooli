# Problems — Portal

Persistent register of known issues, tech debt, and deferred work
specific to **this** scenario. Future agents read this file to avoid
re-discovering the same constraint.

This file ships empty in newly generated scenarios. Append entries as
they appear.

## What belongs here

- **Known bugs** that are real but not yet worth fixing
- **Tech debt** — workarounds that need a real fix later
- **Deferred work** — features descoped from a phase, with the reason
- **Architecture drift** — code/docs/tests that no longer line up with
  the intended capability map or boundary model
- **Constraints discovered the hard way** that aren't visible from
  the code (e.g., "this resource needs warm-up before the first call;
  see commit X")

## What does NOT belong here

- **Generic template issues** — those go in
  [`../guides/troubleshooting.md`](../guides/troubleshooting.md)
- **Open feature requests** — track those in PRD operational targets
- **Code comments** — if the constraint is local to one file, a
  comment there is more discoverable
- **Test failures** — fix them, don't document them

## Entry template

Use this shape so entries are scannable. Append newest at the bottom.

```markdown
### YYYY-MM-DD — short title

**Symptom:** What goes wrong, observable from outside the system.

**Root cause:** What actually causes it (or "unknown" if not yet diagnosed).

**Workaround:** What to do today to keep moving.

**Real fix:** What needs to happen for this entry to be deleted.

**Owner:** Who should drive the fix (or "unassigned").

**Refs:** Code paths, related issues, prior commits.
```

## Entries
### 2026-10-22 - Remaining TV preview qualification after native delivery

Scoped W3 delivery implements the ordinary native preview. Native Home and Nooch rendering/return now have physical evidence, but repeat entry timed out in trial 3f0e3263-ebf7-4b9d-9d96-fc07b34c0812. Keep that cancelled terminal result; do not convert passed chapters into an overall PASS. Hardware qualification remains for favorites persistence/reorder, explicit offline/stale states, large text/reduced motion, full app remote operation and masked header parity. Reset only the preview between clean-search trials: owner text input appended Nooch to Nooch in trial 4. Nooch applied icons remain placeholders, so use the wordmark pending approved Brand Manager assets. Home-first paired identity/local TLS/automatic fallback, authenticated travel, isolated TV emulator, microphone and Home routing remain separate gates. Prior LAN firewall block and WebView-forward teardown issue remain unresolved; no host policy changes authorized or performed. Source and acceptance in android-tv/README.md.
## Work ladder

- Rung: W1/W2 implementation contract established; evidence remains planned.
- Evidence: the active Portal Everywhere plan requires native companion parity,
  desktop surfaces, embeds, and voice. The previous PRD only promised
  "Healthy Portal scaffold" and "Chat-ready contract foundation" at P0, while
  DOMAINS.md deferred embeds and voice. OT-P0-003 through OT-P0-009 now capture
  the new obligations, and 44 Portal acceptance cases have planned requirements.
- Gate: `vrooli scenario requirements validate portal` reports PASSED with no
  business-contract findings on 2026-09-05. This is structural evidence only;
  no acceptance case is claimed passed by that command.
- Remaining: implement and connect the owner APIs, native companion, providers,
  migration, and live acceptance producers. The broader archived front-door goal
  also names retrieval behavior beyond this plan; no claim of full goal parity
  is made here.
- Measured: 2026-09-05.

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| _None yet._ |  |  |  |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues

## 2026-10-22 — TV cannot reach the host application gateway

**Symptom:** The physical TV's native library times out; its WebView reports `ERR_CONNECTION_TIMED_OUT` for the same Portal origin.

**Root cause:** The host kernel records UFW BLOCK for TCP SYN packets from this TV to Portal's UI gateway during the asserted connection trial. Host-local requests to that listener return HTTP 200. ADB reachability is a separate, host-initiated connection and does not prove TV-to-host access.

**Workaround:** Local native TV shortcuts remain available. The existing HTTPS Portal route may provide an authenticated alternative; it must be qualified independently. No firewall or exposure change was performed.

**Real fix:** Admit the exact trusted TV to the required application gateway through an operator-managed host policy, or qualify an existing authenticated HTTPS route. Then prove catalog loading and an actual application render on the TV. The current privilege broker only supports bridge admission on its fixed port and cloud edge ports; do not repurpose those identities or add a private firewall implementation to Portal/Device Control.

**Owner:** Control plane for host admission; Portal for subsequent TV experience qualification.

**Refs:** Private experiment `tv-capabilities-20261022T144942Z/implementation-20261022`, `host-firewall-evidence.txt`, `v4-corrected-library-frame.png`, `v4-corrected-portal-frame.png`.
