# Decisions — Portal

## Approved staged direct access and host ownership — 2026-10-22

After the visual explanation, the operator approved direct LAN access through
project setup, then discovery/reconnection and trusted HTTPS. The pending-choice
language in the earlier research records is historical. The successful localhost
preview used temporary ADB reverse; daily operation must work independently.

The control plane now owns an optional parameterized local-service safeguard and
closed broker action family. Installed policy selects a device IP/interface and
scenario UI listener; callers supply only the binding ID. The broker derives the
live port and validates the running owner rather than accepting arbitrary ports.
IP admission is not authenticated device pairing. Portal continues to own its
catalogue, gateway and connection policy. No root/network repair moves into the
launcher or Device Control.

Full setup preview exposed unrelated pending host work, so a focused
`vrooli setup broker` operation reuses the existing project setup elevation
child. It updates only broker/policy artifacts and preserves other setup state.
Administrator authentication must occur in an operator terminal. The automated
attempt failed at that boundary before applying host rules; no replacement
authorization grant or plaintext password channel is introduced.

## Favorite-first refinement and setup boundary — 2026-10-22

The operator superseded recent-first Home ordering: favorites precede all other
shortcuts, with recency applied independently within each group. Keep the bounded
strip and both library destinations. Remove duplicate page headings and control
instructions. Settings uses a category rail and detail pane; address changes
belong in a focused dialog. Daily22 implements this scoped refinement without
changing Home override capabilities.

The operator rejected one-off elevated firewall commands. They were never
applied and are superseded. Host admission must follow project setup and the
control-plane safeguard/broker model; machine-specific values belong in typed
operator configuration. Portal and Device Control do not own host repair.
Research confirms that safeguards can already be declared by scenarios/resources
and consume validated operator-state parameters. Existing Bridge and cloud-edge
firewall identities are not substitutes for local-service admission.

The historical recommendation was trusted HTTPS on Portal's existing gateway with
setup-managed scoped admission, certificate lifecycle and home hostname
resolution. A shared stable ingress is an alternative when broader demand
justifies it. These networking choices, pairing/authentication and travel fallback
were later approved as staged work; see the current ownership decision above.
Certificate lifecycle, pairing and authenticated travel remain unqualified.
The comparison, exact owner paths and acceptance conditions are in
[`../../android-tv/README.md`](../../android-tv/README.md#portal-host-setup-research--historical-proposal).

## Daily launcher convenience and route discovery — 2026-10-22

Extend the operator-approved Horizon interface with native long-press app
options, persistent favorite stars, focus-selected tabs and icon input cards.
Android owns app-info/force-stop and uninstall confirmation. The launcher does
not gain shell privileges, device-admin privileges or new accessibility powers.
Keep the native source visible despite the root executable-name ignore rule.

Portal owns the local gateway and DNS-SD projection of its current UI port;
reuse `mdns-go`, the existing application catalogue and shared child proxy.
Android NSD supplies bounded endpoint hints. Validate the protocol before route
promotion, preserve navigation focus during refresh, and never switch an open
experience's route. Same hostname and successful TLS/catalogue verification
support automatic HTTPS port updates; DNS-SD alone is not host authentication.

The operator explicitly preferred HTTPS with the simpler LAN approach as a
fallback. Accordingly, daily21 adds an opt-in RFC1918 IPv4 HTTP route with
visible unencrypted labeling. This supersedes the earlier daily HTTPS-only
implementation constraint; it does not supersede HTTPS preference or the
prohibition on automatic downgrade after certificate/authentication failures.
Native origin validation and WebView subresource policy bound cleartext use.
HTTP cannot establish cryptographic host identity or browser voice readiness.

Direct TLS uses configured certificate/key files in the existing UI listener;
partial or broken TLS configuration fails closed. Certificate issuance, DNS,
firewall administration and general scenario LAN admission remain host/control-
plane concerns. The fresh physical LAN timeout is retained; no host firewall
implementation, root credential, permanent ADB route or VPN lives in Portal.

This document records durable decisions and tradeoffs future agents
should not accidentally relitigate.

## Purpose Of This Document

Use this document when a choice:

- affects multiple files or future agents,
- rejects a plausible alternative,
- changes architecture, deployment, data, security, monetization, or
  testing direction,
- needs a revisit trigger.

Routine implementation log entries belong in [`PROGRESS.md`](PROGRESS.md).
Known unresolved issues belong in [`PROBLEMS.md`](PROBLEMS.md).

## Decision Log

| Date | Decision | Context | Consequences | Revisit Trigger |
|---|---|---|---|---|
| 2026-07-05 | Use the generated `react-vite` scenario documentation contract. | Scenario scaffold was generated from the template. | Docs start with stubs and maturity metadata in `docs/manifest.json`. | Revisit when scenario adopts a different template or doc contract. |

## Superseded Decisions

| Date | Superseded Decision | Replacement | Details |
|---|---|---|---|
| None yet. | n/a | n/a | Add when a durable decision is replaced. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history

## TV home-first route direction — 2026-10-22

The operator requires direct home access by default and a saved secure-domain
fallback for travel. Separate connection routes from Portal capability profiles.
Use a paired host identity, a bounded home probe, and visible remote sign-in;
do not equate a private IP/SSID/HTTP 200 with home identity or replay writes on
another route. TV localhost is a development reverse endpoint, not the host's
ordinary network address.

Reuse `api-base`'s existing `createScenarioProxyHost` in Portal's UI gateway
for selected application pages, assets, APIs and sockets. App Monitor supplies
an existing integration precedent but was not running in this investigation.
Do not create a private proxy in Device Control or another launcher repository.
Gateway route context must be canonical across API URL resolution and clients;
current native origin validation and sibling-port resolution do not yet cover
proxy base paths or two-label `.local` hosts. Prefer trusted local HTTPS for
browser voice; split DNS is an optional ingress decision, not implemented here.

This is a recorded design direction, not delivered auto-fallback. Revisit after
local gateway/TLS, authentication, native route selection and repeatable physical
qualification pass. Preserve local navigation failures and microphone/Assistant
unknowns. Research evidence is recorded in the existing Drive experiments tab.

## Native TV preview and application gateway — 2026-10-22

Deliver the selected Horizon design as ordinary Android Views in Portal's
existing android-tv source. Native theme, focus/navigation state and bounded
artwork loading are separate classes; one WebView exists only for the selected
experience. Keep stock Home and Assistant selected. The build and device owners
remain Scenario to Android and Device Control. No extra repository, Compose
migration, Device Control product UI or daily ADB dependency is introduced.

Project product names and applied icons from existing scenario declarations;
retain scenario IDs as control identities. Suppress explicitly marked template
artwork. Nooch currently has a wordmark and placeholder public PNGs, with no
approved mark located in Brand Manager; the TV uses a wordmark rather than
presenting the gray placeholder as branding. The operator explicitly accepted
this wordmark for now in the active conversation. Brand Manager remains the owner
for a real approved Nooch mark. Aquila reuses its existing applied asset.

Native clients opt into relative gateway URLs through ApplicationCatalogService;
other callers retain direct URL behavior. Portal mounts api-base's shared host
proxy for child HTML, assets, APIs and sockets. Child browser routers must use
the injected primary proxy path as basename. Nooch's localized routing repair
keeps the direct phone route intact. Shared HTML injection must decode upstream
compression and remove stale content encoding, length and byte validators;
streamed and cached gzip/deflate/Brotli regressions now enforce that contract.

Favorites initially seed Aquila and Nooch, while preserving explicitly saved
preferences, including an empty list. All apps is an exploratory inventory;
none acquires a TV-ready label from runtime availability or one page render.
Home-first automatic route selection, paired identity, local TLS and authenticated
travel remain separate implementation gates. Revisit this architecture only
when measured TV feedback identifies a boundary this design cannot support.


## Optional daily Home override and evidence boundaries — 2026-10-22

Use the operator-approved daily-only accessibility route while retaining stock
Google TV, Setup Wizard and Assistant. Preserve the exact prior Home component;
route only its activity event, retaining one transient stock-visible boolean
from window metadata. Never retrieve screen nodes/text, filter keys or perform
gestures. Requested permission, live binding, effective Home and package-manager
default are distinct states. Pause persists before opening stock and stops the
foreground lifetime. New-boot and wake handling are bounded; normal same-boot
reconnection must not hijack another app. No polling host watchdog.

On Android11, never declare or dynamically add an accessibility package filter
that later needs clearing: the platform accumulates it. Gate paused event delivery
with eventTypes instead. Verify the compiled resource and actual policy in the
existing native build owner. Keep launcher behavior/status in Portal, generic
read-only target-scoped observations and actuation in Device Control, and stable
signed immutable channels in Scenario to Android. A passing Home or standby check
cannot establish reboot persistence; the daily17 reboot failure remains open.
