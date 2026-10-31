# Portal Android TV launcher

Current installed daily release: **0.4.12/code30**, with final upgrade
qualification recorded below. Earlier daily25 fixed catalogue continuation and
viewport clipping. The selected direct LAN connection is working with a
setup-installed, verified persistent admission rule. Trusted local HTTPS and
endpoint-change/travel qualification remain separate work.

## Interaction polish target — 23 October 2026

Operator-directed W3 repair: switch tabs without network/package-discovery or
whole-page rebuild work on the input path; preserve the previous tab, card and
scroll when returning from native TV apps or embedded Vrooli apps. Home routing
must still bring the launcher forward, but must not reset its selected tab.
Prevent automatic focus recovery while changing browser/native surfaces from
activating the Home tab. Keep one continuous opening overlay through route
resolution and WebView loading. TV apps and Vrooli each use one list: favorites
alphabetically, then other apps alphabetically, with the same corner star.
Remove All apps/Favorites/TV-ready controls and TV-control qualification labels;
retain search and truthful stopped/offline availability.

Bounded handoff: reuse the current native owners, stable signing identity and
Device Control leases. Add desired-behavior regressions, measure actual native
tab-handler work before/after, then install and visually qualify the final
daily release. Performance Health's browser/React profiler is not a measurement
of this native surface; use bounded shell-only app-status timing metadata.
Private evidence belongs in `interaction-polish-20261023` alongside earlier TV
experiments. Budget: six submitted physical flows, zero reboots; separate
read-only observations and final lease checks. Do not change Home permissions,
host policy or the selected Portal connection. Preserve every failed receipt.

### Implementation and qualification

Tab content is retained in native page containers. Input handlers switch those
containers rather than rebuilding grids, querying Package Manager or cancelling
network connections. Installed-app discovery, labels and banner decoding use a
background snapshot. HTTP disconnect runs off the UI thread. Catalogue refresh
is asynchronous and freshness-gated; unchanged successful results retain views.
Nearby rows are materialized initially, later rows still append as needed, and
idle-time preparation creates the first library pages before normal navigation.
Changed favourites, installed apps, catalogue data, search and cleared recents
invalidate their affected pages. Home retains recent-first ordering within its
favorite/non-favorite groups; both full libraries use alphabetical order within
those groups. Both libraries use the same native corner-star drawable. Removed
library-mode preferences are ignored; existing favorites remain intact.

Home intents return to the current launcher page. Explicit app-return state
restores the departure page after external lifecycle changes, including instance
state restoration. Embedded close preserves its card, search and scroll. A
focus-recovery event cannot select a tab: Left/Right on a tab commits the adjacent
selection directly and retains focus there; pointer clicks still select tabs.
This replaced a rejected input-scope flag because Android delivered focus events
after key dispatch. The explicit Home tab remains available to select Home.
One centered opening overlay survives both route resolution and WebView startup;
the old bottom loading bar is removed. Actual WebView errors retain one return
control. No Home/accessibility permission or network preference is changed.

The owner build runs navigation, Home-policy/resource, origin and catalogue
checks. Desired preservation and initial-row regressions failed before repair
and pass afterwards; additional tests cover alphabetical groups, native app
return markers and adjacent-tab navigation. Native status timing rows use
`count/last_us/average_us/max_us`, with no screen text or account content.
Three D-pad cycles on baseline26 measured Vrooli average 165.223 ms / max 199.025 ms
and TV apps average 235.397 ms / max 853.489 ms. The daily29 run measured Vrooli
average 1.734 ms / max 1.885 ms and TV apps average 1.558 ms / max 1.869 ms, including one
additional TV-app selection for the app-return check. Ready pages were prepared
with a 2.5-second startup settle. These are native handler measurements, not GPU
presentation, network latency or Performance Health browser results.

Run `da99e57d-a92d-4512-8f54-716678b4d156` confirms SmartTube foreground and
return to the selected TV-app tab. Run
`9bd3cca4-5025-4cd3-b5db-aae47bf51c29` captures the unified favorite-first
library, one centered loader and Back returning to the Vrooli tab with the Aquila
search retained; Clear restores the full list. Separate owner status confirms
`ui.page=vrooli`, browser/loading false and Home override active. Aquila's child
page displays an Invalid URL error; filed as Scenario QA
`knw-1792728989890967531`. This is not a claim that Aquila is usable on TV.

Keep failed evidence: daily27's native-return flow passed foreground assertions
but its frame showed Home; its Home navigation counter rose from 3 to 4. Daily28
failed the tab-selection precondition and opened Netflix; the exact SmartTube
foreground assertion refused it. Neither is accepted as a correct app return.
The six-flow budget is closed. Final code review added Home invalidation when
clearing recents and removed the now-unused focus guard; code30 has one separate
bounded upgrade smoke check, recorded in `final-invalidation-scope.json`.
Operator history is not cleared during validation. Private receipts remain in
`tv-capabilities-20261022T144942Z/interaction-polish-20261023`.

Final daily30 artifact:
`android-release:ec96821e680ab8286854373b22a660a508188068d7a98688bd1e99ca12f78772`.
The signed owner build, all four independently executed native checks, immutable
digest and public signer continuity pass. Final upgrade smoke
`9abb1699-4d45-47eb-a2af-9c367e6b2607` passes. Its frames show the unified library,
actual Nooch content through the saved local gateway, and Back returning to
Vrooli. This confirms rendering and return, not every Nooch feature or remote
control. Existing recents/favorites were retained; no history was cleared.
The final native foreground assertion passes and all leases are released.
Seven total flows were submitted, including the explicitly recorded final
invalidation smoke; no reboot, permission or host-policy change occurred.

## Approved connection implementation — 22 October 2026

The operator approved the staged direct-LAN, discovery/reconnection and trusted
HTTPS work after reviewing the visual explanation. This supersedes the pending
decision language in the research section below. Use a bounded implementation
task, with this document as the design and evidence handoff. The first checkpoint
is a setup-installed local-service admission capability in the control plane;
Portal remains the existing gateway. Ordinary setup installs the root-owned
policy from typed operator configuration. Runtime callers select a binding ID,
not arbitrary addresses, ports or commands. The broker resolves the live UI
listener from the canonical runtime registry and verifies its owner before
admission. Match only exact managed firewall rules; preserve unrelated rules.
Converge listener changes, support revocation, and report unavailable capability
honestly. Pin the selected TV's current private address and home interface in
configuration; IP scope is not cryptographic pairing and DHCP reassignment must
not silently broaden admission.

Harness handoff: Work until the approved direct connection is reproducible via
project setup, with focused control-plane policy/registry/firewall regressions
passing and physical catalogue/child-page evidence, then qualify discovery and
reconnection. Read docs/TESTING.md and this document first. Scope is control-plane
admission/setup, Portal connection behavior and their maintained documentation;
reuse existing owners. Run affected Go packages and scoped scenario checks only.
Keep the current native Home active. Record any missing administrator credential,
DNS control or certificate provisioning authority after preparing a reviewable
implementation; do not substitute one-off elevated commands or weaken TLS.
Checkpoint changed, verified, remaining and unverified here on interruption.

### Setup admission checkpoint — 22 October 2026

Implemented the optional `local_service_admission` safeguard and its broker
action family in the control plane. Portal declares that optional host
requirement. Setup installs root-controlled bindings; requests select a binding
ID only. The broker derives a current live UI claim from the read-only canonical
runtime registry, checks process ownership/identity and the home interface, then
uses exact managed UFW rules. Verified replacement precedes stale-rule removal;
stopped/unverified listeners lose managed admission, revocation survives broker
restarts, and removed setup bindings lose orphaned rules. The broker converges
every thirty seconds and records changed/failed operations. This is a scoped
IPv4/UFW implementation; it does not authenticate an IP as the physical TV.

Added `vrooli setup broker --sudo-mode=ask` because full setup preview included
unrelated pending host work. The focused project setup command reuses the normal
installation child and excludes other host requirements, resources, credentials,
onboarding and global completion state. `make install` completed its managed
build/vet gates and installed the updated command. The TV/Portal binding is
persisted through the existing host operator-state owner, not embedded in source.

Focused broker, safeguard, registry/schema, parameter, setup and CLI tests pass.
The opt-in read-only live listener test verified Portal's actual UI port24965,
owner UID1000, expected scenario/variant and home interface. It did not contact
UFW or the broker. Portal Test Genie dependencies run
`20261023-030047-37a4b222` passes. Proto owner freshness was verified before Go
consumer checks. An unrelated safeguard environment invariant failed on
agent-session-containment's live-test input; filed as Scenario QA
`knw-1792724261714121445` without modifying that file or claiming attribution.

The initial automated installation stopped before root execution because sudo
requires an operator terminal. The operator subsequently authenticated the
focused setup command. Live qualification exposed a service configuration bug:
UFW creates Python temporary files, but the broker kept `/tmp` and `/var/tmp`
read-only without a writable `TMPDIR`. Installed UFW can acknowledge a kernel
update without persisting its rules after that failure. Do not equate live TV
traffic with a saved, verified rule. A regression now requires
`TMPDIR=/run/vrooli`, using the existing writable broker directory while keeping
host temporary roots read-only. The safeguard also retains bounded broker
failure details instead of hiding the actionable result code.

The operator authenticated the corrected setup at 23:18 local time. The live
unit now has the expected `TMPDIR`; `/etc/ufw/user.rules` was updated, and the
host safeguard reports `already_present` after exact managed-rule verification.
The operator sees the native Vrooli catalogue over the saved direct LAN origin.
HTTPS/certificate/DNS lifecycle, automatic endpoint-change recovery and
authenticated travel still require separate qualification. No password belongs
in chat. Private receipts:
`tv-capabilities-20261022T144942Z/setup-admission-20261022`.

### Catalogue continuation and viewport qualification — 22 October 2026

The operator reported that All apps stopped in the P's. Portal's actual RPC
returns 128 apps, with Personal Planner at position60 and Workspace Sandbox last.
The native library discarded cards after60 despite saying more results existed;
this was a rendering cutoff, not a paginated server response. Daily
**0.4.5/code23** replaces it with incremental view creation: sixty initial cards,
then batches of thirty when scrolling or D-pad focus approaches the bottom.
Append rows in place and update focus links across the old boundary. Search and
Favorites/TV-ready filters use the full fetched catalogue; deep focus restoration
materializes the saved card before restoring scroll. The existing 512-entry
protocol safety bound remains explicit; no unsupported cursor is invented.

Artifact:
`android-release:5cb4e42989830a489bb03fa4858bd20ed38870514fdd722d180e322e946bb945`.
The owner build and independent navigation, Home policy/resource, Portal origin
and catalogue traversal checks pass. Artifact digest and signer continuity
verify. Installation and explicit native foreground entry pass. Physical run
`77281ad6-515b-4f71-be9f-80f337708159` uses D-pad input through the old cutoff;
frames show Personal Prompt Manager beyond Personal Planner and focus on the
final Workspace Sandbox card. This is manual visual catalogue evidence, with a
separate terminal native-foreground assertion. A prior duplicated wait-ID flow
was refused at preflight without device inputs; retain that receipt.

The scrolled frames also exposed cards drawing over the fixed filter/connection
area. Daily **0.4.6/code24** enabled ScrollView clipping, but the physical frame
still shows overlap. Retain this failed visual hypothesis. The signed owner
build and digest/signer checks pass. Artifact:
`android-release:393f4f0fd2d1be60660f242c7c0831b6d2cc993bd3c0b13f32e9b5ad8d1dcb4d`.
Physical run `30d61961-a5b1-4bd4-8a7a-8ceb7dd2eaa6` again reaches Workspace
Sandbox and its real child HTML/assets render through the selected local
gateway. The child frame still has a loading indicator; it is not evidence for
full child API/socket or remote operation. The frame named restored-tail shows
Home, so deep Back restoration is unqualified. The run exceeded its85-second
limit during the closing capture, after the Home command. Its overall result is
cancelled; passed chapters do not turn it into a passing journey. A separate
state read confirms native foreground and no leases remain. No reboot occurred.

The original five-flow budget is closed. A separate bounded known-defect repair
records one additional short visual check: the grid's parent was also disabling
child clipping through `TvTheme.column`. Daily **0.4.7/code25** enables parent
clipping while retaining the focused-card inset. This uses the existing launcher
polish authority and changes no Home permissions or network configuration.
Source, build and visual qualification are in `viewport-repair-scope.json` beside
the retained earlier receipts.

Daily25 is installed from
`android-release:9044e6c3dd0811c369ab67d226c8535b9bcae1604eb737b34f375002b92f4920`.
The signed owner build runs all four native checks. Independent artifact digest
and public signer continuity checks pass. Physical run
`69eb4f1e-0b54-43de-b012-5e4ba6e77048` passes; its scrolled frame confirms cards
are clipped below the fixed connection status, and its closing frame shows
native Home before the terminal foreground assertion. The catalogue batching
model and filtering logic are unchanged from the fully traversed daily23/24;
daily25's short physical check qualifies viewport clipping, not a second full
128-card traversal. No Home permission, selected server or fallback preference
was changed. All leases are released. The retained child-page frame proves
HTML/assets rendering, not complete application behavior. Deep Back restoration,
actual endpoint-change recovery, trusted HTTPS and authenticated travel remain
unqualified.

## Daily22 refinement target — 22 October 2026

Operator instruction supersedes recent-first Home ordering: show all favorites
first, ordered by their recent launcher use, followed by other apps ordered by
their recent launcher use. Unused favorites retain saved order; other unused
apps retain installed catalogue order. Retain the bounded Home strip and both
library destinations. Remove repeated page titles/subtitles and control hints.
Use a two-pane Settings layout with Home, Portal and Preferences categories;
connection editing belongs in a focused dialog, not a long mixed form.

Portal networking was research-only during this refinement. The operator
subsequently approved the staged implementation recorded above. One-off elevated
commands remain rejected; use the project setup/host-owner model. The earlier
manual UFW commands are superseded and must not be used.

## Daily22 refinement release — 22 October 2026

Daily **0.4.4/code22** is installed using the existing stable signing identity.
Immutable artifact:
`android-release:7a00f138603d26878467477d0d580667809250583394419e34e89266720504e9`.
Home now places every favorite before other shortcuts and orders each group by
recent launcher use. The strip remains bounded to twelve app shortcuts, followed
by the two library destinations. TV apps, Vrooli and Inputs omit repeated page
headings and control hints. Necessary connection/error state remains visible.
Settings separates Home, Portal and Preferences into a category rail and a
focused detail pane. Address editing uses a native dialog; only Connect saves.
The existing Home integration and permissions are unchanged.

The desired-order regression failed against the old implementation and passes
after the repair. The signed22 source snapshot passes navigation, Home override
policy/resource and Portal origin checks. The artifact digest and signer
continuity were independently verified. Physical frames confirm the three
cleaner pages, all Settings categories and the address dialog. Actual launcher
entry to Netflix moved it from third to first among the four starred favorites;
the other favorites still precede non-favorites. Netflix and SmartTube foreground
assertions and subsequent native Home returns pass. The interrupted dialog-close
sequence captured Home rather than Preferences; it is not evidence for D-pad
category navigation or Cancel persistence. A separate final capture confirms
Preferences without clearing history. No address, fallback, Home permission or
input source was changed during qualification.

The first install/Home assertion failed on stock Google TV. Explicit native
launch recovered the screen, and later real app-to-Home checks passed. Preserve
that failed receipt; this release does not establish unattended activation after
APK replacement or fix the earlier full-reboot binding issue. Closing app status
reports requested/allowed/connected/foreground true, pause reason none and
boot_seen67. No reboot was performed. The final frame is native Home and all
Device Control leases were released.

Final passing run `f808e63c-e518-4eec-b7a5-95512958c2c8` captures before its
terminal Home assertion. Device Control saved it as flow
`615fa217-4686-44f0-b366-4f0104d4555b`, version1, context
`hisense:api30:daily0.4.4:home-override`. Its initial Settings taps require native
Home at 1920×1080; it is not a portable Settings navigation test. Private build,
regression and physical receipts are under
`vrooli-experiments/tv-capabilities-20261022T144942Z/launcher-refinement-20261022`.

## Portal host setup research — historical proposal

This section is the pre-implementation research baseline. The operator's later
approval and the current implementation/authentication checkpoint are above.

The one-off UFW proposal did not meet the project's reproducible setup model.
It was never applied. No firewall, DNS, certificate, VPN or privileged installation
was changed during this research. Localhost on the TV refers to the TV; ordinary
home operation should connect directly to the host over the LAN. ADB reverse
remains an explicit development experiment, not the daily transport.

### Existing owners and gaps

- `internal/setup/setup.go` resolves project, scenario and resource host
  requirements. It installs the broker before applying the remaining requirements
  and verifies it again during setup completion. The parent retains the operator
  session; `--sudo-mode=ask` elevates the fixed broker installation child.
- `internal/safeguards/<name>/safeguard.json` declares host-state changes; Go
  handlers register in `internal/runtime/registry.go`. Scenario/resource
  `hostSafeguards[]` declarations feed that same setup resolver. Tool manifests
  provide installable prerequisites. A Portal-specific shell installer is not
  needed. See [host safeguards](../../../docs/configuration/host/safeguards.md).
- Parameter schemas and operator-state choices already support machine-specific
  configuration. `internal/hostreq/config.go` merges defaults and recorded values
  and validates them. Persist policy through the owning onboarding API; do not
  hand-edit operator state or put this TV's addresses in source code.
- `internal/privilegebroker` supplies a root-owned Unix-socket service, peer-UID
  checks, closed action validators, fixed argument construction, verification and
  audit records. Current UFW families admit Bridge's fixed18767 or public edge
  ports80/443. Neither is a general local-scenario admission policy. Do not
  impersonate Bridge or use the public allow-any-source edge action for this TV.
- `internal/cloudtarget/edge_route.go` already owns managed Caddy snippets,
  validation/reload and rollback for fenced cloud deployments. Reuse appropriate
  primitives if a shared local ingress becomes necessary; its deployment
  authority/profile does not automatically authorize a development-host route.
- Portal already serves its catalogue, child applications and sockets through
  one UI gateway, with optional TLS and bounded DNS-SD hints for the current UI
  port. The scoped owner scan did not find a ready local DNS-01 certificate
  issuance/renewal workflow. TLS files alone do not provide certificate lifecycle,
  local name resolution, firewall admission or application authentication.

The broker continues to perform bounded root operations after setup installs it;
ordinary callers do not run sudo or gain arbitrary root commands. This is the
existing meaning of the single setup elevation boundary. If the desired policy
instead forbids all later privileged mutations, use a setup-fixed ingress rule
and user-owned routing behind it; do not promise automatic dynamic firewall
changes under that stricter interpretation.

### Options

| Option | Benefit | Remaining work / tradeoff |
|---|---|---|
| Scoped LAN HTTP on the existing gateway | Smallest connectivity slice; exercises current discovery and proxy | New setup-managed admission still required; unencrypted, no authenticated host identity or browser microphone secure context. Keep it an explicit fallback. |
| Trusted HTTPS on the existing gateway — recommended first | Reuses working native/gateway boundaries, supports secure contexts, avoids another proxy | Generic local-service admission safeguard/broker capability; managed certificate renewal; stable home hostname; pairing/authentication and physical qualification. |
| Shared stable local HTTPS ingress | One listener remains constant while upstream scenario ports move | Larger host capability: local profile of the existing ingress primitives, setup-managed listener/TLS/DNS, lifecycle route reconciliation and rollback. Choose when several clients/services need it, rather than adding it solely to repair this TV. |

For trusted HTTPS, prefer a dedicated hostname under an operator-controlled
public DNS domain with DNS-01 certificate issuance and a home DNS answer pointing
to the host's private address. This does not require public router port forwards;
normal service traffic stays on the LAN. DNS-provider credentials need the
existing secret owner, limited scope and renewal evidence. The hostname remains
the TLS identity while DNS-SD supplies the current gateway port. The current TV
client reads the advertised URL, so DNS-SD's resolved IP alone will not make that
public hostname resolve privately. Router/resolver capability and Android name
resolution must be qualified before selecting this provisioning profile.

A private CA is an alternative when public-domain DNS automation is unsuitable,
but requires deliberate TV/app trust provisioning and rotation. Installing a CA
on the host does not establish trust on Android. Do not disable certificate
validation or trust every user CA to avoid that step. A Cloudflare origin-only
certificate is not a substitute for a certificate the TV can validate directly.
See [Caddy automatic HTTPS](https://caddyserver.com/docs/automatic-https) for
DNS challenge and private-CA distinctions, and [Android trust configuration](https://developer.android.com/privacy-and-security/security-config)
for app trust anchors.

### Recommended implementation boundary and acceptance

Propose an optional local-service-admission safeguard in the control plane,
declared by Portal's host requirements. Its typed configuration identifies the
permitted device binding, home interface/network, scenario and listener name;
network values are operator configuration. Resolve the current UI listener
through lifecycle state, not a request-supplied port or port range. Treat a LAN
IP as firewall scope, not device identity: DHCP changes must be reconciled
against a verified binding, and pairing must authenticate the application.
Broker validation must enforce the installed scope independently of a scenario
request. Give managed rules exact inspect/apply/verify/revoke behavior, preserve
unrelated rules, remove stale rules after verified port transitions, and support
revocation. Do not enable UFW, open all scenario ports, or repair host networking
inside Portal or Device Control. DNS-SD multicast admission, if actually blocked,
requires its own narrow home-interface policy; opening TCP alone does not prove
discovery works.

Keep connection selection in Portal: bounded home probe and verified catalogue,
paired host identity, separate saved secure-domain travel route, visible sign-in
when required. A missing home route may offer the remote route; certificate or
authentication failure must not silently downgrade to HTTP. Switching routes
must not replay an operation or move an open experience. Direct LAN traffic
bypasses Cloudflare Access, whose policy is enforced at its proxy. Review Portal
and child application authorization before enabling sensitive sessions; neither
TLS nor a source-IP firewall rule replaces that authorization. The current
gateway has no blanket authentication gate, and chat's interceptor permits an
absent bearer header. This is a qualification requirement, not a claim that all
Portal operations are anonymous. See [Cloudflare's application boundary](https://developers.cloudflare.com/cloudflare-one/access-controls/applications/http-apps/).

Accept the host change only after setup reruns converge, invalid policy is
refused, arbitrary ports/targets cannot be requested, rollback/revocation pass,
and lifecycle port changes leave only the intended admission. Then prove on the
TV: discovery, catalogue, a real child page and socket, restart/port recovery,
no ADB dependency, trusted TLS, blocked invalid certificates, home outage and
authenticated remote fallback. Keep route/network qualification separate from
the still-open full-reboot Home activation check. Implementation status is
recorded in the checkpoint above; use its project setup authentication handoff.

## Launcher convenience target — 22 October 2026

Extend the approved Horizon Home: hold OK (or Menu) on a TV app for favorites,
Android app info/force stop and system-confirmed uninstall. Show favorite stars
on cards. Focus alone selects the four page tabs and keeps focus in navigation,
including asynchronous library refresh. Inputs use the same card grid with
source icons and connection state. Preserve all Home override behavior.

Portal investigation distinguishes an unset daily connection (preview and daily
preferences are separate), a stale dynamic endpoint, blocked LAN access and
remote authentication. Test these separately; port discovery alone cannot fix
TLS, host firewall or sign-in. Use existing service discovery/port owners; do not
scan ports, bake this host's port into the APK or depend on ADB for daily access.

## Daily21 convenience release — 22 October 2026

Daily **0.4.3/code21** uses the existing signing identity. Its immutable artifact
is `android-release:af5d0cdf41b89fcece739c648a0e6e2dd5b7ed4f46cfd05ab6276b2a0b0999d8`.
Preview remains isolated. The root ignore rule for the `vrooli` executable also
matched the Java package directory; the native `.gitignore` now explicitly
retains these sources so they can be reviewed and checked in.

Hold OK on a TV app card for its context menu. Menu is a keyboard/remote
alternative. Favorites have a star and persist in the existing preferences;
app info opens Android's exact app page, where Force stop remains a deliberate
system action. Non-system apps also offer Android's confirmed uninstall flow.
No app was force-stopped or uninstalled during qualification. Page tabs select
on focus and retain that focus through content refresh; Down enters the page.
Inputs use a sorted four-column card grid with HDMI/source vector icons and
Android's reported connected, standby or disconnected state.

### Portal discovery and transport

Portal advertises `_vrooli-portal._tcp.local` through the existing `mdns-go`
package. The advertisement uses the lifecycle's current `UI_PORT`, the gateway
protocol tag and an interface-specific instance name. It does not advertise
each child app or require separate app ports. Set `PORTAL_MDNS_ADVERTISE=false`
to disable it. Multicast is optional; manual origin entry remains available.

The native client browses for eight seconds, resolves at most eight services
sequentially (Android11's resolution limit), and releases its multicast lock.
Choose a home endpoint in Settings once. Subsequent matching-host port changes
must return a valid bounded Portal catalogue before they replace the route.
HTTPS upgrades are preferred when available. Unknown hosts, conflicting
endpoints, failed TLS, authentication redirects and malformed responses never
silently replace a connection. An open experience retains its route.

The operator authorized HTTPS preference with a home-LAN HTTP fallback.
The fallback is an explicit Settings preference, off by default. It accepts
only a selected RFC1918 IPv4 origin, labels successful connections as
unencrypted, and rejects other cleartext WebView navigation/subresources.
Android's static cleartext policy must permit this user-selected dynamic
origin; PortalClient and WebView enforce the narrower application policy.
TLS errors never enable the fallback or copy secure-origin cookies to it.
Plain HTTP cannot provide cryptographic host identity or browser microphone
secure-context support. A same-IP port update is a hint qualified by the
catalogue protocol, not proof of host identity.

For direct HTTPS, configure `PORTAL_TLS_CERT_FILE`, `PORTAL_TLS_KEY_FILE` and
`PORTAL_HOME_HOST` through the existing scenario lifecycle environment. The
certificate must be valid for that hostname and trusted by the TV, and the
hostname must resolve to the home gateway. The existing UI gateway then serves
HTTPS on its assigned `UI_PORT`; pages, catalogue RPCs, assets and child sockets
share that listener. Missing half of the TLS configuration or invalid files
fail startup rather than downgrading. No private key belongs in this repository.
Certificate issuance/local DNS and authenticated travel remain unqualified.

The daily app initially had no saved Portal address: daily and preview use
separate preferences. The current host gateway returns a valid catalogue
(128 applications), but the fresh TV-to-LAN request timed out. This is consistent
with the earlier confirmed UFW rejection and is not a successful Portal
connection. The approved fallback preference and proposed home origin are saved
on the daily TV. No ADB reverse route or firewall change was substituted.

The earlier device-specific UFW command proposal is superseded by the
operator's project-setup requirement. Those commands were never applied and
must not be used. Research the setup-managed broker/safeguard boundary before
implementing network admission. Device bindings and lifecycle-resolved ports
belong in persisted policy, with inspect/apply/verify/revoke behavior owned by
the control plane. Portal and Device Control must not carry private root setup
or firewall implementations. The operator will decide after reviewing options.

Focused Portal application/discovery tests and six gateway transport/proxy
checks pass. Test Genie dependency run `20261023-015135-45434432` passes after
SDA repaired the newly used Go module's tidy drift; the earlier failed run
`20261023-014708-36075ae6` is retained. The signed21 source snapshot independently
passes navigation, Home policy/resource and Portal route policy checks.
Earlier broad unit-suite failures and the reboot binding gap remain open.
Final physical run `b87a62f3-751e-4dd6-90e1-526fd18e8932` passed SmartTube
foreground and Home return. Reviewed frames confirm TV apps, Vrooli and Inputs
select with focus retained on each tab, including after the asynchronous
connection/discovery attempt; sorted input icons/state and the themed app menu
render correctly. Closing frame is native Home with the earlier favorites
preserved. This is UI/Home evidence, not a Portal network success.
Library promotion was correctly refused because the run ends with a frame
capture rather than its outcome assertion. No new replay revision was published;
the next qualification flow must capture before its terminal Home assertion.
Private physical receipts and frames are under
`vrooli-experiments/tv-capabilities-20261022T144942Z/launcher-polish-20261022`.

## Historical Daily17 qualification — 22 October 2026

Daily `com.vrooli.portal.tv.home` 0.3.9 (code 17) is installed through the
stable signing identity. SmartTube and YouTube entry plus actual Home returns
pass. The native pause control opens Google TV, leaves Home there and stops the
foreground keep-alive; resuming restores native Home. A screen-off readback
showed Google TV underneath, and wake returned to the native launcher without
Home or launch input. The single restart incremented boot count 66 → 67, but its first foreground
was Google TV's profile chooser and Android reported the service waiting to
bind. Confirming the already highlighted profile still left Google TV open. Explicit
owner launch restored the native screen, but its status showed permission and
request true, connected/foreground false and boot_seen still 66. Reboot
persistence has not passed; do not infer it from standby. Android Settings
independently disabled and re-enabled permission, but binding still failed.
One maintenance reinstall of the exact same signed17 artifact restored
connected/foreground true and boot_seen67. That is recovery, not a reboot fix.
Do not depend on reinstalling from the host for daily startup. Closing recovery
confirmed SmartTube entry, actual Home return and a native frame; the TV was
left on the restored native Home. The full-reboot activation phase stays open. Preview remains installed as 0.3.4 (code 12).

Two concrete defects were repaired. Device Control's stale Settings window
pointer overwrote the correctly resumed legacy activity; assertions now prefer
resumed activity and retain independent window disagreement. Android 11 adds
accessibility package filters cumulatively: setting `packageNames = null` does
not clear a previously declared own-package filter. The service XML now omits
the filter entirely; pause gates event types instead of adding another filter.
A build-owned regression checks the actual resource and prohibited capabilities.
The service receives only window-state metadata and retains no screen text.

Native captures work in the latest qualification; historical failed captures
remain failed and establish no particular security mechanism. Read-only owner
queries distinguish configured permission, live binding, app-owned status,
activity/window facts and bounded process-exit history. The app-owned status
provider requires Android's `DUMP` permission, refuses writes and exposes only
Home metadata. No Portal URL, account data or private screen content is returned.

The immutable signed daily artifact is
`android-release:870f1d185fd3193f8c824d92ec326d7ab8e101992a7d87fdc282b56dc87a2abb`.
APK+AAB hash, public certificate continuity and compiled no-content/no-key/no-gesture
capabilities were independently verified. The passing SmartTube/YouTube Home
flow is saved in Device Control under context
`hisense:api30:daily0.3.9:home-override`; retrieve its exact version before replay.
The saved flow ID is `6d276722-a5d1-45dd-bcf4-aa3a83eb20db`, version `1`.
Replay it only on the exact device/context after the operator returns the TV:

```bash
device-control flow replay \
  --id 6d276722-a5d1-45dd-bcf4-aa3a83eb20db --version 1 \
  --device-id android-0df87078c18abfb8 \
  --context-key hisense:api30:daily0.3.9:home-override \
  --actor tv-home-qualification --json
```

Private protocols and receipts are under
`vrooli-experiments/tv-capabilities-20261022T144942Z/home-wake-20261022` and
`home17-qualification-20261022` beside it. Focused owner regressions and native
build checks pass; earlier broad Device Control unit failures remain unresolved.

## Standby repair target — 22 October 2026

Active-user runtime evidence marks the enabled override as disconnected/crashed.
The newest application exit is signal termination, not a recorded Java exception;
the sender and precise kill policy remain unknown. Use Android's standard
foreground-service lifetime while override is requested and its permission is
allowed, with a low-priority ongoing notification opening native Settings. Pause,
permission revocation and unbind must stop this keep-alive. No extra microphone,
screen-content, gesture or key-filtering permission is needed.

Add a bounded first/new-boot entry and deferred wake handling. Window-state
metadata is used transiently to keep one boolean: whether stock Home is the
current window. Receiving other window-state metadata clears that boolean so
a delayed wake callback cannot take focus from another app; never read or store
other apps' screen text. Each wake gets at most two deferred checks, not a
polling watchdog. A normal service reconnect in the same boot must not force
Home over another app. Native Home arrival acknowledges successful routing,
so repeated healthy Home presses do not trip the failure burst limiter.

Keep permission, live binding, effective Home and package-manager default
distinct in Settings and evidence. This target is not a declaration that the
new lifetime or startup behavior has passed physical acceptance.

## Earlier wake diagnosis — operator report, 22 October 2026

Daily 0.3.6 (code 14) was installed with the optional Home override. The operator
enabled the system service, confirmed a physical Home press stayed in Vrooli,
then reported that power off/on returned Google TV. Fresh active-user readback
still shows the service configured/enabled. That proves permission persistence,
not runtime binding or successful Home return from another app. Wake and reboot
activation remain unqualified. No stock packages were disabled.

The earlier setup was interrupted for normal TV use; its incomplete experiment
is retained separately. The resumed bounded diagnosis checks service binding,
missed wake/startup events, and Device Control's activity/window observations
before further launcher changes. Its Settings foreground readings conflict with
operator-visible screens, and failed capture is not proof of any particular
security mechanism. Do not promote those failed checks or claim true default
Home from the permission flag alone.

## Active Home override experiment — operator direction, 22 October 2026

The operator approved proceeding with the custom override first to get daily
Home working soon. Implement a daily-only, opt-in accessibility service that
routes solely on the previously observed stock Home activity's window event.
Other window-state metadata only clears its transient stock-visible boolean.
It launches the native Home alias, preserves system packages and Assistant, and
does not request screen-content retrieval, gestures or key filtering. Android's
accessibility settings own enabling/disabling the service. Settings must show
its actual connected/requested state and offer a persistent pause-and-open-stock
escape. A burst limiter pauses the override on repeated redirects instead of
creating an unbounded crash/relaunch loop. No host watchdog or Portal connection
may be required. Preview must not expose this service.

Qualify stock recovery first, then Home from native pages and normal TV apps,
sleep/wake and one reboot. Android may continue reporting Google TV as the
resolved Home; describe the result truthfully as an effective Home override,
not a changed package-manager default. This is a new bounded routing experiment;
the earlier code12 campaign and its failures remain unchanged. No package
disabling is included in this route. Physical acceptance is pending.

## Earlier code12 delivery checkpoint — 22 October 2026

Daily `com.vrooli.portal.tv.home` version 0.3.4 (code 12) was installed with
the stable signing identity. Physical checks cover native app entry, root Back,
the bounded recent-app row and both end destinations, restored focus, the gear
button and preserved recency through an in-place signed update. The operator
confirmed Netflix visibly open when its screenshot could not be decoded. These
checks establish app entry, not playback or every account/streaming path.

The separate preview package was updated from code 7 to code 12 on the TV.
Both packages then reported 0.3.4; preview entry, unchanged stock Home routing,
daily entry and retained daily recency passed in an independent bounded flow.
This proves channel isolation while stock Home is selected. Isolation with the
daily launcher actually selected remains a later activation check.

**Persistent activation is pending.** The operator selected Vrooli in Android's
Home dialog, but independent Home resolution and an actual Home-button test
still selected Google TV. One checked owner `home-set` request also returned
Android's `Success` while readback stayed stock; it was not retried or accepted
as successful activation. Read-only inspection observed Home priorities 2 for
Google TV, 1 for Setup Wizard recovery, and 0 for Vrooli. The Home role holder
could not be read reliably and remains unknown.

Both system packages remain installed at their original DEFAULT enabled setting
for Android user 0. That earlier experiment stopped before package disabling or
an accessibility override. Those are separate rollout choices, each requiring
its own bounded qualification and recovery evidence; neither is delivered by
the successful launcher build. Reboot, sleep/wake persistence and host-disconnected
Home qualification remain pending until an effective routing method is chosen.

Device Control now provides checked `home-set`, read-only `home-state`, a typed
one-shot restart request and checked `package-enabled-set`. The package setter
compares expected state and independently reads the result back. Its focused
tests prove exact DEFAULT restoration with fake device responses; they do not
prove physical restoration or authorize a system-package change. See
[the owner contract](../../device-control/docs/reference/capabilities.md#checked-home-selection-and-device-restart).

| Potential package-route change | Original setting | Exact recovery setting |
|---|---|---|
| `com.google.android.apps.tv.launcherx` | `default` (0), user 0 | `default`, expected `disabled-user` |
| `com.google.android.tungsten.setupwraith` | `default` (0), user 0 | `default`, expected `disabled-user` |

If that route is chosen, change one package at a time under the exact device
lease, inspect the outcome before proceeding, and keep recovery independent
of the launcher UI. Restore Setup Wizard first, then Google TV, and verify
actual stock Home routing. Do not uninstall, clear data, assume explicit
`enabled` equals the original setting, or retry an uncertain change. Qualify
Settings, normal TV apps, casting/network integration and boot before adoption.
An accessibility-based alternative would preserve these packages but needs a
new native implementation, explicit system permission, and its own input,
startup, failure and recovery experiments. Its effectiveness is unproven here.

Scenario to Android's scoped unit phase passes. Device Control's focused Home,
foreground, restart and checked-package regressions pass; its broader unit phase
remains failed. The readiness review is yellow and rules checks were unavailable.
This checkpoint does not certify whole-scenario readiness.

## Daily Home target — operator priority, 22 October 2026

The operator now wants this launcher active and persistent for daily TV use.
Reliable installed TV apps are the activation requirement; unfinished Portal
experiences may remain visibly experimental and improve while Home stays live.
Home uses a horizontal scrolling row of at most 12 shortcuts: recent launches
from Vrooli first, then pinned apps, then installed apps to fill remaining spaces.
The complete installed inventory remains under All TV apps. The last two cards are All TV
apps and All Vrooli apps. Settings uses a gear with no filled button background;
remote focus adds an outline and icon tint, and its accessible label is retained.
Persist recency and restore visible focus; do not
request usage-access permission to infer launches from other launchers.
This supersedes the earlier preview-only target for the next rollout. It does
not mean Home has already been selected or that activation checks have passed.

| Outcome | Acceptance boundary | Owner |
|---|---|---|
| Dependable native Home | System Home reaches native Home from every preview/library/WebView state; Back at Home stays there; app shortcuts and inputs remain usable without Portal, computer or ADB. | Portal android-tv |
| Usable TV apps | Exact native package/component launch, clear unavailable state, verified SmartTube/YouTube/Netflix/Prime entry and return, remote focus and persisted favorite order after restart. | Portal; Device Control supplies qualification evidence |
| Selected Home survives normal life | Verify actual OEM Home selection, sleep/wake and authorized reboot/cold start; retain stock launcher and a tested recovery route. Never depend on a host process continually reopening the app. | Android/OEM owns Home; Portal integrates; Device Control verifies |
| Sustainable updates | Stable signing identity, versioned immutable builds, distinct daily/preview channels or an equivalent verified isolation boundary, preserved preferences and a rehearsed rollback. Debug-to-release key migration is explicit. | Scenario to Android, secrets-manager and existing distribution owners |
| Professional project integration | Native lifecycle/navigation/app catalogue/connection components have clear boundaries; important regression cases run through existing build/test owners; docs, requirement evidence and reviewed changes match actual support. | Portal and adjacent owners |
| Experimental services cannot disrupt TV | Portal, WebView, authentication or host outage cannot block Home or native app launch; failures return to native navigation. | Portal android-tv and shared gateway |

The Home eligibility intent and system-role/default-app selection must be
implemented and qualified on this Hisense Android 11 TV. Prefer Android's
supported selection UI; discover an OEM-specific supported alternative only if
needed. Keep default-Home mutation, inspection and restoration generic in Device
Control if a new typed owner operation is required. Never put raw ADB in the
launcher or bypass a genuinely human-only system decision. Preserve Assistant,
stock launcher packages and existing account/network policy.

Release-core checks are bounded, assertion-bearing cases, not one long journey.
One passing command is not proof that an app opened; record its actual foreground
and visible/semantic entry. A cancelled long flow is not promoted. Exercise the
normal remote, then remove the experiment connection before claiming daily-use
independence. A Home app must recover when Android recreates its activity; an
always-running watchdog is not the persistence mechanism.

After activation, retain the daily Home while developing candidates separately.
Qualify updates and recovery before applying them. Portal home-first identity,
local TLS/auth, secure-domain travel fallback, full per-experience TV controls
and voice remain the next product work; none blocks a working native Home.
The existing Home/Portal design, no generated substitute branding, shared gateway
and owner boundaries remain in force.

## Build and iterate on the native launcher

The source belongs to Portal. Scenario to Android owns source snapshots, Gradle,
profile projection, signing and retained APK/AAB bytes. Device Control owns
device identity, leases, installation, input and qualification. Android owns
the selected Home; neither a host watchdog nor an ADB reverse route is needed
for installed TV apps. Keep release promotion separate from building.

From the repository root, build an ordinary preview with:

```bash
ANDROID_PROJECT_KIND=native \
ANDROID_SOURCE_REF="$PWD/scenarios/portal/android-tv" \
ANDROID_NATIVE_PROFILE=preview \
SCENARIO_TO_ANDROID_HTTP_TIMEOUT=360 \
scenario-to-android android build --json
```

Build the daily candidate with the stable existing signing identity:

```bash
ANDROID_PROJECT_KIND=native \
ANDROID_SOURCE_REF="$PWD/scenarios/portal/android-tv" \
ANDROID_NATIVE_PROFILE=daily \
ANDROID_SIGNING=required \
ANDROID_SIGNING_IDENTITY=vrooli/portal/tv-home-signing-key \
SCENARIO_TO_ANDROID_HTTP_TIMEOUT=360 \
scenario-to-android android build --json
```

Start the build owner with `vrooli scenario start scenario-to-android` if needed.
Never build this source tree in place. Every owner build runs the actual
dependency-free navigation model's JVM verification through Gradle `preBuild`.
Rollback must respect Android version ordering: retain the known-good owner
artifact/source and build a qualified higher-version restoration with the same
certificate. Do not uninstall the daily Home to force an older APK onto it.
Increment `version_code` and `version_name` in `vrooli-native.json` before an
in-place update. Preserve the daily signing identity; do not provision a new key
for each release. Metadata defaults to the ordinary preview profile so omitted
profile selection cannot accidentally produce a Home candidate.

The preview package is `com.vrooli.portal.tv`; the daily package is
`com.vrooli.portal.tv.home`. Both use the Java namespace `com.vrooli.portal.tv`.
Only the daily profile enables alias `com.vrooli.portal.tv.HomeActivity`.
Their preferences and installed package identities are separate. The build
result includes `native_profile`, package, version, signing identity and retained
paths. Verify those fields, the APK+AAB combined checksum and public signing
certificate before installation. A successful build is not device acceptance.

Use Device Control's existing short, lease-owned flows for install, launch,
exact foreground checks and redacted observations. Inspect the actual target
before clicking; semantic lookup is currently unqualified on this TV. Netflix
capture availability is separate from whether it opens normally. Keep account
screens and personal captures out of shared evidence bundles.

Choose Home through Settings only after the core TV-app and recovery checks.
With the optional override enabled, `Pause override · open Google TV` first
persists the paused state and stops its keep-alive, then opens the saved previous
Home. `Enable Home override` resumes routing; Android accessibility permission
is a separate state. Android accessibility settings remain an independent disable
route. On this TV the path is Settings → System → Accessibility → Vrooli
Home override → Enable. The right-hand two-pane preview and opened service
page have different positions; inspect the fresh frame rather than replaying
preview coordinates. Confirm the visible Stop/Use dialog and independently
read permission and runtime binding. The latest off/on recovered permission
but not the post-reboot binding failure. These controls do not change the
package-manager Home choice. Recovery
must also restore the system Home choice when that choice was changed.
Keep stock packages enabled. Android release downgrades are not a promised
rollback path: recover to stock, then build a corrected or prior known-good
source with a higher version code and the same key. Do not clear user data to
force a downgrade. Update preview independently and verify the daily Home and
its preferences stay selected before promoting a new daily build.

Portal origin, cookies and catalogue cache are optional. Daily builds prefer HTTPS;
the narrowly scoped `ANDROID_PREVIEW_CLEARTEXT_HOST=localhost` switch is for
debug preview experiments only. It does not implement a paired local-first
production connection or authenticated travel fallback. Daily21 adds a separate
explicit private-LAN HTTP fallback; see its transport policy above.

## Delivered preview v0.2.1 — 22 October 2026

The ordinary preview now implements scenic native Home, four editable installed
app favorites, a visible Vrooli destination, native TV-app and experience grids,
search/favorites filters, applied product labels/artwork, bounded loading and
unavailable states, and focus/filter/scroll return state. The wallpaper brief
and exact generation prompt are in [the image brief](../docs/mockups/image-generation-brief.md).
One WebView opens the selected experience; local navigation stays native.

The installed debug artifact is
`android-debug:fc2fab95f9cb51ebf1b8b4a3713a0afe41de15a0bde44b0d514622ec46ef85d6`
(version code 7). The owner-built APK+AAB digest and size were verified before
installation. No dependencies were added to the Android project.

Physical evidence establishes Home/focus movement/TV apps/Back, then Nooch's
actual `Your day` rendering through the localhost Portal gateway, D-pad input,
Back to the focused Nooch wordmark card with the `Nooch` query preserved, and
Back to native Home. Trial `3f0e3263-ebf7-4b9d-9d96-fc07b34c0812` reached those
checks, then was **cancelled at its 90-second deadline** during second library
entry. It is not an overall passing flow. Recovery
`af7ee62f-8693-457c-b8f3-e65a171a8589` passed: saved HTTPS origin restored, stock
Home foreground, no sessions, owned reverse route restored. No food/account
writes, Home selection, Assistant selection or OS/network policy changes occurred.

The operator's gray icon was Nooch's published placeholder PNG. Portal now
suppresses explicitly marked template art and the card uses its text wordmark;
the operator explicitly chose this wordmark on 22 October. An approved
Nooch icon remains a future branding-owner task. Nooch launch uncovered
and repaired two real defects: shared gateway compressed HTML handling, then
Nooch's missing browser-router mount path. Direct phone URLs remain supported.

Validation: 32 affected api-base checks; ten Nooch route checks; seven Portal
catalogue/artwork checks; focused native navigation/ordering, CLI, gateway and
Device Control checks pass. Nooch Test Genie unit and Portal dependencies pass.
Portal and Device Control unit phases remain failed on existing desktop fixture
and coverage gates; no whole-scenario readiness claim is made.

Remaining qualification: repeated entry after the deadline, persisted favorites
editing/reordering on hardware, explicit offline/stale and large-text/reduced-
motion trials, complete remote web operation, isolated TV emulator, header
parity behind the privacy mask, daily home route with paired identity/TLS,
authenticated travel, voice and default-Home routing. These are separate from
one successful page render. Split future physical flows into focused ≤90-second
cases. Restart only the preview for a clean query; the device owner's text
operation appends and does not promise replacement of selected text.

Private receipts: `tv-capabilities-20261022T144942Z/polished-preview-20261022`.
The existing Drive plan's Experiments tab holds the shared evidence index.

## Earlier ordinary-preview delivery boundary — 22 October 2026

The operator authorized the proposed polished ordinary-app slice after the
design review. Work shape: bounded implementation and qualification, W3 against
the selected Horizon/library design and this README. Deliver native Home,
favorites, branded library, explicit connection/loading/error states and exact
Back/focus restoration. Keep the existing catalogue, build and device owners.
Scope includes adjacent owner fixes needed for repeatable observation; record
extensions and their evidence here. No HOME/Assistant selection, package removal,
new microphone permission, network policy, account or authentication-policy
change is included. A design review approval does not certify a working product.

Harness handoff: Work until the ordinary TV preview implements the selected
Horizon/library interaction and its affected checks pass, with native owner-build
artifacts, physical frame/focus/return/recovery receipts and remaining gates
recorded here and in the existing Drive experiment report. Read the selected
concepts, local-experimentation and docs/TESTING.md first. Keep going through
friction; step back and shrink or switch when evidence warrants it. Two fresh
review passes must find no material in-scope correctness or visual findings;
green tests alone do not prove design acceptance. Operator feedback is the
request for a professional finish and quick, proper feedback; the earlier test
app does not meet it. Missing decisions or credentials are reported precisely;
diagnosable implementation friction is not a permission request. Checkpoint
this bounded slice before expanding to travel auth, microphone or Home routing.

Adjacent owner extension: Portal's shared browser gateway is required to open
one app over the selected connection without separate per-app reverse routes.
Use the existing api-base engine and an opt-in catalogue resolution mode; retain
direct URL clients. Device Control's semantic assertions are read-only but the
existing retry classifier excludes them and drops failure-time frames. Qualify
bounded assertion retries and redacted failure frames at that owner before
repeated physical catalogue trials; input actions remain one-shot.

The operator independently reported a gray Nooch icon and a failed web launch
while the phone route worked. The published Nooch PNGs are placeholder artwork,
not a native decoding fault. The browser gateway also forwards gzip negotiation
to an HTML stream injector that reads encoded bytes as UTF-8 and retains the
encoding header. Fix that protocol defect in api-base's shared host engine and
prove streamed/cached gzip and Brotli behavior before another vertical trial.

Native Android Views own Horizon navigation and installed TV app shortcuts.
One system WebView is created only for a selected Vrooli experience. No HOME
intent filter, default launcher changes, accessibility service, boot receiver,
ADB dependency, or background server is part of ordinary use.

Build through Scenario to Android, with `ANDROID_PROJECT_KIND=native` and
`ANDROID_SOURCE_REF` pointing here. Its builder copies and fingerprints source
before using its existing Gradle, signing and artifact pipeline. Do not build
this project in place. `vrooli-native.json` is shared by the builder and Gradle.

The Vrooli library consumes Portal's ApplicationCatalogService Connect JSON
contract in `packages/proto/schemas/portal/v1/surfaces/surfaces.proto`. Runtime
inventory comes from the control plane; TV compatibility remains unverified
until tested. Selecting an application resolves its current browser URL.
Application sign-in remains in that application's WebView; this client never
mints credentials or bypasses authentication.

Use Settings to enter a Portal web origin. A `vrooli-portal://connect?server=…`
link opens a confirmation screen. It never silently changes the origin.
HTTPS is the default. LAN HTTP qualification must use an explicitly scoped
network security domain in the build; no global cleartext setting is enabled.

Qualification and known limitations are recorded in Portal's existing
`docs/internal/PROGRESS.md`. This preview is not a certified replacement Home.

## Design-readiness baseline before delivery — 22 October 2026

The selected visual source is Drive's **03 — Vrooli TV — Visual Directions**,
document `11W3kDDQ2MDPAyU3sWyJk0xrboyVMP56uBemWUT8iUL0`. Horizon Home
(`1UnkJDeZbD0rePTo1YMUf6s9UN8FyLP64`) and the shared Portal library
(`14uwqqWjS157hurs9PQoHcmsvI_XoX1kE`) are concept references. They are
generated images with illustrative branding, not screenshots or wallpaper assets.
The owner selected Horizon previously and now requests a professional finish.
The earlier v0.1.4 procedural scene, greeting and button-list catalogue were
technical preview treatments. The delivery status above supersedes those
implementation gaps; the selected visual target remains applicable.

| Surface | Target composition | Gap at design review / required proof |
|---|---|---|
| Home | Scenic still, restrained top navigation, clock and one familiar app row | Replace the placeholder scene with approved clean artwork; remove the unsolicited greeting. Keep Vrooli visible in the first row. |
| TV apps | Installed TV activities, actual package branding, explicit favorites editing | Current Home and TV apps both enumerate all activities. Favorites, persisted order and banner/icon fallback need implementation. |
| Vrooli | Shared destination from tile/tab; native three-column grid, authentic names/art and honest qualification | Current catalogue derives labels from scenario slugs and renders text buttons. Add metadata through Portal's existing catalogue owner. Aquila is web-console's product name; reuse its applied asset. |
| Inputs / Settings | Short native lists; clear connection route and recovery | Input enumeration exists; input switching, focus restoration and home/remote route selection remain unqualified. |
| Open experience | Full-screen app; cancellable loading, auth/unavailable states, exact return | A rendered Nooch page is evidence of rendering only. D-pad operation, authenticated travel use and return-to-card remain unqualified. |

Use the actual device's 960 × 540dp layout grid (1920 × 1080px at density 320).
Begin with 48dp horizontal and 27–30dp vertical safe areas, leaving additional
room for focus outlines. These are trial tokens, not measured overscan results.
Use 18sp labels, 16sp secondary text, a 28sp library title and 21–24sp clock.
Retain one immediate ivory focus outline with a dark keyline, optional 1.025×
scale and 80–120ms interpolation. Reduced motion removes scaling. Separate
selected-tab indication from keyboard focus. Keep geometry stable while loading.

The initial Home composition can fit four owner-selected installed apps plus
the library tile in five columns; avoid placing the library after every installed
utility. One possible fit is five 158dp tiles plus four 16dp gutters inside
the safe area with focus inset. Validate long labels, focused-edge clipping,
large font scale, bright/dark art and real sofa readability before freezing it.
Use the installed TV banner when suitable, then the actual icon and a readable
label. Never stretch or regenerate another app's accepted mark.

Design the following states before judging parity: focused/unfocused/pressed,
loading/cancel, no matches, offline/stale, sign-in required, app unavailable,
and return from a web experience. Voice states remain conditional on an actual
input path; the mockup must not imply a recording microphone. Keep zero Home
widgets as the proposed first slice; Ambient and automatic seasonal art can
follow later. These are scope recommendations, not additional owner decisions.

Keep the ordinary native Views implementation for the next slice. Extract the
theme/tile/navigation components, stable item identities, focus/history state,
and catalogue/connection state from the single Activity as those changes are
implemented. A Compose rewrite is not justified by the current evidence. Keep
daily navigation free of Device Control; keep network/auth logic out of paint
and focus handlers. No new repository or launcher-specific proxy is needed.

### Feedback loop: observed capability and missing qualification

One unchanged owner build of v0.1.4 completed in **11.646 seconds**, including
CLI/API overhead. APK and AAB bytes reproduce the previous combined artifact
identity `android-debug:a2262a69ab1771a94a71f9debf438d176cb93e2c50214f0f6fce3fc141cd0b7f`.
The checksum is SHA256(APK bytes + NUL + AAB bytes), not APK-only SHA256.
Total output is 169,753 bytes; source files remained unchanged. This is one
build sample with an already provisioned toolchain, not a cold-cache,
design-change, deployment or complete visual-cycle benchmark. Nothing was installed during that design-readiness benchmark; the later
delivery trials below are separate evidence.

Use three feedback stages:

1. Review the chosen mockups and real assets against a named state matrix.
2. Build through Scenario to Android, then use an isolated Android TV target
   and fixed catalogue fixtures for native focus/state captures. This target
   is still to be qualified: the two inspected local AVDs are generic Google
   API 36 images configured at 320 × 640, not TV profiles. The target owner
   reports emulator prerequisites ready while visual validation lacks a GUI
   session. Do not treat that result as TV-ready or boot an existing AVD as
   a substitute for a capacity-admitted TV profile.
3. Validate a small selected candidate on this physical Android 11 TV through
   Device Control's leased flows. Capture foreground, focus/semantics, image,
   failure-time diagnostics and exact recovery. Keep the stock launcher selected.
   A physical capture's status-bar privacy mask obscures the top 90 pixels;
   it is not evidence of clipped launcher navigation. Header parity needs a
   qualified native reference capture or an owner-supported scoped policy.

Before automating repeated visual trials, fix the observed text-append/filter
state friction, use exact semantic controls instead of assumed initial focus,
and capture failures before an assertion aborts the flow. Prefer bounded
condition readiness to longer fixed sleeps. Qualify WebView observation cleanup
at its Device Control owner before using CDP; tracked issue
`knw-1792696363648007859` remains open. Screenshot success alone does not prove
web navigation, frame latency, physical contrast or accessibility.

An Android ramp probe reported ADB enumeration timeout while an exact Device
Control read of this TV succeeded during this investigation. Preserve both
receipts; broad discovery failure is not proof the known TV is disconnected.
The TV was in stock ambient during that read-only design-readiness check.
Later delivery trials use separately recorded leases and recovery receipts.

Research record: private experiment
`tv-capabilities-20261022T144942Z/design-readiness-20261022` and the existing
Drive plan's Experiments tab. Google TV layout, focus and typography guidance:
[layouts](https://developer.android.com/design/ui/tv/guides/styles/layouts),
[focus](https://developer.android.com/design/ui/tv/guides/styles/focus-system),
[type](https://developer.android.com/design/ui/tv/guides/styles/typography).

For temporary development qualification, the debug builder also accepts the
literal `localhost` scope through `ANDROID_PREVIEW_CLEARTEXT_HOST=localhost`.
The origin validator accepts loopback HTTP only in a debuggable app. Use Device
Control's flow-scoped `reverse-tcp` operation for the required ports; the flow
must return a verified restoration receipt. This is a test route, not a daily
use dependency or proof of release networking. The APK does not request audio
recording permission or implement an Assistant service. Portal sessions and
AudioTools remain the intended host owners for future voice work.

## Local-first connection target (research, not yet implemented)

Keep a paired host identity and separate home and remote routes. These routes
are independent of Portal's Client / Local control / Automation capability
profiles. At home, choose the verified direct route before contacting the
remote domain. TV `localhost` denotes the television; it reaches the host only
while an explicit reverse tunnel exists. Ordinary home use needs a reachable
LAN/overlay origin or a host gateway. The current debug reverse route is a
qualification tool, not a permanent service.

Remember the user-approved home address, discover candidates through the
existing DNS-SD/Bridge discovery capability where useful, and use Android
network-change callbacks to invalidate stale probes. An SSID, private address,
service advertisement, TCP connection or HTTP 200 is not host identity. Check
the paired server identity and a valid Portal protocol response before using
an address. Start with a bounded probe of the known home route; contact the
saved HTTPS remote route only when home is unavailable and remote fallback
is enabled. Preserve a manual Remote action for travel. Identity/certificate
mismatch, captive-portal HTML, authentication-required and service-unavailable
are separate outcomes; a TLS failure must not downgrade to HTTP.

Show the chosen route and reconnect status. Keep a healthy route for the
current interaction. Do not retry chat submissions, uploads or tool commands
on another route automatically. Switch at an idle/reconnect boundary, with
existing session IDs and owner replay/resume rules. Keep cookies and device
credentials scoped to their validated origin; never copy Cloudflare cookies
to LAN addresses or add trusted-identity headers on the TV.

Use the same selected route consistently for catalogue resolution, scenario
pages, API calls and audio transport. The current URL resolver preserves
private-IP routes but cannot derive a sibling application from a two-label
`.local` hostname; its fallback is localhost. A stable home gateway or an
explicit canonical host-route model must address that case before `.local`
is offered as a working browser route. Bridge's current proxy accepts unary
Connect RPCs; it is not an HTML/assets/WebSocket gateway. App Monitor already
hosts `api-base`'s scenario proxy with browser-path and WebSocket handling;
reuse that shared engine in Portal's existing UI gateway for a single home
entrypoint, rather than creating a proxy implementation or depending on the
entire App Monitor administration UI. App Monitor was not running during this
research; its implementation is precedent, not a live TV qualification. The delivered preview opts into relative gateway resolution on its selected
Portal origin; existing direct URL clients retain their contract. Connection
settings still accept origin roots only. Child applications must support the
injected proxy mount path, as Nooch now does. Remote application exposure also
needs validation against Tunnel Manager rather than assuming every running
scenario has a public sibling subdomain.

For browser speech, plain HTTP on a LAN IP is not a secure context. Prefer
trusted local HTTPS, or qualify a native Android capture frontend that uses
the existing authenticated host audio service. A split-DNS HTTPS origin could
keep the same domain local at home and Cloudflare-backed away, but requires
explicit DNS, certificate, ingress and authentication work. An overlay/VPN is
another route option, not implicit authorization to install or enroll one.

The native RPC client deliberately does not follow login redirects. It needs
an explicit sign-in state and a cookie retry bridge into its own origin.
WebView permits HTTPS redirects; actual IdP/Cloudflare behavior must be tested.
Google OAuth cannot be assumed to support an embedded WebView. A system
browser or existing owner pairing/device flow may be necessary; none is
claimed implemented here. Microphone runtime permission, origin-scoped WebView
permission, audio input availability, provider health and Assistant-role/key
routing remain separate gates. Do not request all WebView permissions merely
to enable microphone capture.
