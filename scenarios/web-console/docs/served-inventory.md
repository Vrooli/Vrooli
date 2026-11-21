# Web Console served inventory harness

This first BAS proving ground is a source-defined served crawl. Web Console has
no client router: `App` mounts the workspace at `/`; query parameters can seed
the approvals and file-preview deferred intents, then are stripped. The
current harness therefore has one served route, `/`, with optional synthetic
query cases. `/health` and capabilities are boot dependencies; all other data
is fetched through Connect RPC under `/vrooli.web_console.v1.*` or REST below
`/api/`.

Run only when the operator has started Web Console and approved browser work on
that local instance. The runner leaves service lifecycle to the owner:

```sh
cd scenarios/browser-automation-studio/playwright-driver
node scripts/feature-inventory.mjs crawl http://localhost:PORT/ /tmp/wc-inventory \
  --scenario web-console --build BUILD_ID --revision GIT_REVISION \
  --synthetic-profile web-console \
  --entrypoint approval='/?approval=synthetic-decision' \
  --entrypoint file-preview='/?file=%2Fsynthetic%2Fexample.txt' \
  --safe-action-paths 'Settings > Account | Settings > Devices' \
  --safe-key-chords 'default::Ctrl+Shift+H | default::Cmd+Shift+H'
```

The inventory derives `identity.build` from the SHA-256 of the first served HTML
document and records its route/state/byte count in `identity.build_source`.
This identifies the document that selects the app assets, not the bytes of each
loaded JavaScript file and not a source revision. `--build` and `--revision` are
retained as unverified hints (`build_hint` and `source_revision_hint`); they do
not attest the served build.

Repeat `--entrypoint NAME=URL` for explicit synthetic deep-link states. Entry
names are short lowercase labels and URLs must stay on the served origin. The
inventory records the state label (for example `/ [state=approval]`) instead
of persisting the query string, so real identifiers cannot leak into evidence.
The base URL must not contain a query; name every query-bearing start state.
Each named entrypoint is scanned separately from the same ephemeral browser
profile, and the route limit counts route/state pairs.

`--safe-action-paths` is opt-in and requires the synthetic profile. It clicks
only the exact visible accessible names in each path, in order, preserving
button, radio, tab, link, or menu-item roles. Use it for local panels or
choices; do not put submit, delete, install, answer, or other external-effect
actions in this list. Prefix a path with its named entrypoint and `::` to run
it only in that state; use `default::` for the base pathname. For example,
`default::Settings > Account | approval::Pending > Resolved | file-preview::Dismiss`.
Unscoped paths retain the legacy behavior and are tried on every route/state.

`--safe-key-chords` is also opt-in and requires the synthetic profile. It accepts
only exact `Ctrl+Shift+<letter>` or `Cmd+Shift+<letter>` chords, optionally
scoped with the same `entrypoint::` prefix. Use a chord only after reviewing
the source handler and confirming the resulting state is safe; this option
does not enable arbitrary key sequences, text input, Enter/submit, or editing.
For example, Web Console's source-bound `Ctrl/Cmd+Shift+H` opens its archive
drawer; the archive/recovery/search controls inside that drawer remain
unallowlisted and must not be activated.

Feature crops record `screenshot_capture`: `element` is a locator crop,
`rect` is the inspected component's bounding box, `bounded-fallback` is the
locator's bounding box after its element screenshot failed or returned a
uniform image, and
`viewport-fallback` is a clipped viewport image when no usable box exists.
Fallback screenshots are always clipped because the remote browser backend can
return a blank texture for an unclipped screenshot. BAS also checks each PNG
for visible pixel variation; an all-uniform element crop retries at its live
bounding box, then the viewport. A uniform viewport result fails the crawl
instead of being retained as a successful crop. This catches blank textures,
not visual quality. Review the capture method when a crop looks broader than
its feature.

Action and component scans exclude elements with zero size, `display:none`,
hidden/collapsed visibility, zero effective opacity, or an `aria-hidden=true`
or `inert` ancestor. A control behind an accessibility-hidden modal is not
reported as a current user action in that state.

Before a route is scanned, BAS waits until visible text, landmark structure,
accessible labels, and their viewport geometry remain unchanged for 350 ms.
After an explicitly safe menu/tab/action/key interaction, BAS also waits for a
stable visible change before recording the state as observed. This allows lazy
UI chunks and short entrance transitions to finish; a control request by itself
does not count as a served state.

The `web-console` profile permits the served top-level document and same-origin
static assets. It returns synthetic fixtures for `/health`, all `/api/` paths,
all `vrooli.web_console.v1` Connect paths, and every `fetch`/XHR regardless of
origin. EventSource gets one empty synthetic event with the correct stream
media type. Other non-static requests are blocked. The browser receives an inert
closed WebSocket client before the app loads; no socket request or payload is
sent or inspected. Only pathname, method, resource type, and outcome are
retained in network evidence; query strings and bodies are omitted. Thus user,
account, file, transcript, and credential data cannot be returned by a network
service to this crawl. Synthetic API responses intentionally make this a
visual route/action inventory, not a validation of API behavior or populated
real-user states.

## Source-derived coverage contract

| Surface | Observed / eligible | Not established by this harness |
|---|---|---|
| Served routes | `/` and any same-origin anchors exposed while a declared state is open; route limit is BAS's configured maximum | Hidden routes, server-only routes, routes requiring real data, and unlinked deep links |
| Query/deep-link states | Base launch plus separately supplied synthetic `approval` and `file-preview` entrypoints | Approval/file content, which remains synthetic; populated user records and backend behavior |
| Visible actions | Visible buttons, links, menu items and submit controls are inventoried and cropped | Ordinary actions are not clicked; effect and backend behavior are unknown |
| Components | Semantic regions and visible React component identities/crops when runtime metadata exists | Hidden/unmounted components, component identity when production React metadata is unavailable |
| Loaded script paths | Per-route/state path lists for loaded same-origin `.js`/`.mjs` resources, with URL query strings omitted, support source-map reconciliation | Bundle contents, source-map equivalence, and component-to-file identity unless the asset bytes match an identified build |
| Menus/disclosures/tabs | BAS opens controls with supported ARIA menu/listbox, native disclosure, or tab semantics; captures exposed state and newly visible menu items | Nested interaction depth, unsupported custom state, and behavior beyond the first opened state |
| Gestures | React event prop names are recorded where exposed; explicitly supplied safe Ctrl/Cmd+Shift letter chords can capture their resulting state | Native listeners, handler values, drag, hover, all other keyboard sequences, nested scroll, and gesture outcomes |
| Network | All data requests receive synthetic responses or are blocked; static assets only from served origin | Live API fidelity, account/filesystem content, and actual service integration |

Web Console's source lists substantial workspace surfaces (conversation,
terminal, approvals, integrations, settings, file previews, launcher and
workspace panes), but visibility is state- and data-dependent. The inventory
must report only what the served crawl actually exposes. An absent component
means “not observed,” not “deleted.” The existing BAS diff emits embedded
`research-feature-decision/v1` keep/drop cards for each disappeared observed
feature, recommending keep; it does not infer whether the source is dead code.
Dead-code reconciliation and screenshot before/after review remain separate
EX8 work.

## Opt-in synthetic session states

Use BAS profile `web-console-session` when a served inventory needs a populated
workspace without a live Web Console session. It returns one invented
`synthetic-session-0001` record and matching workspace layout, an empty archived
session list, and `null` for the optional coding-backup state. Every other API
request still receives the generic empty response; event streams receive one
empty synthetic event, and the browser's WebSocket is inert. No fixture comes
from the running API, and the session id does not name a real process. This
profile is opt-in; the `web-console` profile continues to return empty API
responses.

For the phone sidebar and its archive view, the source-reviewed local path is:

```sh
cd scenarios/browser-automation-studio/playwright-driver
node scripts/feature-inventory.mjs crawl http://localhost:21233/ /tmp/web-console-ex8-session-20261113-final \
  --scenario web-console --build ex8-session-crawl-label \
  --revision source-revision-hint --synthetic-profile web-console-session \
  --safe-action-paths 'default::Open sessions > Archive 0'
```

The captured same-build inventory is at
`/tmp/web-console-ex8-session-20261113-final/inventory.json`. The crop for the
open mobile session sidebar is
`/tmp/web-console-ex8-session-20261113-final/evidence/component-7d7a518dec3adaa345a2.png`;
the empty archived-session panel is
`/tmp/web-console-ex8-session-20261113-final/evidence/component-c9683cfa333522808a8c.png`.
The inventory contains 296 observed features, 92 synthetic network requests,
and no runtime errors. This example only opens local navigation and the empty
archived-session panel. It does not start a session, send a terminal command,
or invoke a mutating UI action. API requests are fulfilled in BAS and are
recorded with their method and synthetic outcome.
