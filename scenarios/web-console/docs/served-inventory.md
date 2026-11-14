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
  --safe-action-paths 'Settings > Account | Settings > Devices'
```

`--safe-action-paths` is opt-in and requires the synthetic profile. It clicks
only the exact visible button labels in each path, in order. Use it for local
panels or tabs that do not expose menu/tab semantics; do not put submit,
delete, install, answer, or other external-effect actions in this list.

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
| Query/deep-link states | Base launch plus separately supplied synthetic `?approval=synthetic-decision` and `?file=%2Fsynthetic%2Fexample.txt` launch variants | Approval/file content, which remains synthetic; automated multi-variant orchestration is a follow-up |
| Visible actions | Visible buttons, links, menu items and submit controls are inventoried and cropped | Ordinary actions are not clicked; effect and backend behavior are unknown |
| Components | Semantic regions and visible React component identities/crops when runtime metadata exists | Hidden/unmounted components, component identity when production React metadata is unavailable |
| Menus/disclosures/tabs | BAS opens controls with supported ARIA menu/listbox, native disclosure, or tab semantics; captures exposed state and newly visible menu items | Nested interaction depth, unsupported custom state, and behavior beyond the first opened state |
| Gestures | React event prop names are recorded where exposed | Native listeners, handler values, drag, hover, keyboard, nested scroll, and gesture outcomes |
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
