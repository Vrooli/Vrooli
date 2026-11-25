# MB4 run/thread to BAS session association audit

Scope: read-only source audit for the Web Console managed-browser workspace. No
browser sessions were opened and no runtime identity headers were forwarded.

## Finding

There is a trusted authorization path from an Agent Manager run to BAS session
open, but Web Console has no trusted query that resolves that run to the exact
active BAS broker session and lease. Agent Manager derives run identity from the signed `X-Agent-Identity-Token`,
checks the run's persisted browser capability and grant receipt, and scopes
authorization to the capability's mode, profile, origin and resource role
(`scenarios/agent-manager/api/internal/handlers/browser_capability.go`). BAS's
`agentaccess.Broker` opens and owns sessions, calls Agent Manager before opening,
and its evidence recorder reports lifecycle changes using credentials resolved
from trusted context (`scenarios/browser-automation-studio/api/automation/agentaccess/{broker.go,evidence.go,agentmanager.go}`).

However, `/api/v1/browser/record-evidence` accepts the session ID in a
run-authenticated JSON request and labels the resulting event `run-agent`.
Agent Manager explicitly documents that this is run-reported evidence, not BAS
attestation. Thus a verified run/thread and a claimed session ID are correlated,
but the durable Agent Manager event alone does not prove BAS created or still
owns that exact session. Do not treat an event's `session_id` as an owner-backed
session locator or reuse it for takeover. MB14 further confirmed that current AM
browser evidence accepts a BAS broker-attested **closed-session summary** after
the session ends; it does not bind the active immutable BAS session/lease for a
live observer. The run grant authorizes open/profile/start URL, but the BAS owner
must still attest its live entry before a Web Console view receipt is safe.

## Existing safe primitives

- BAS already owns the session broker and typed `BrowserSessionService`; its
  API module accepts exactly one run identity header and one grant receipt and
  binds them in request context. The broker validates run context per operation.
- Agent Manager has authenticated run context resolution and authorize-open
  routes. These bind the request to the active run and stored browser grant;
  caller/run/receipt fields are not trusted from JSON.
- Web Console has scenario discovery (`api-core/discovery.ResolveScenarioURLDefault`)
  and same-origin owner-backed proxy patterns, including Device Control/Bridge
  (`api/desktop_proxy.go`). Continuity uses discovered Agent Manager origin;
  cross-scenario links are resolved from the incoming request origin and are
  withheld when resolution is unavailable (`api/main.go`).
- No existing Web Console BAS proxy or Web Console browser-session→run/thread
  authority was found in the inspected source. Never derive a BAS origin from
  environment/user input or expose a node/CDP/lease URL to the UI.

## Smallest safe integrated slice

Add one authenticated Web Console pane that attaches read-only to an already
open run-scoped BAS session. First add a BAS-owner operation that resolves an
exact session under the authenticated run grant and returns only a short-lived,
opaque observer attachment capability; BAS must verify its own session owner,
active lease, run, and grant before issuing it. The Web Console backend should
resolve BAS through scenario discovery, call that owner route with a
server-held service identity, then proxy a fixed allowlist of observe/stream
operations same-origin. The browser receives only the opaque short-lived
observer capability and sanitized surface data. Web Console must resolve the
selected run/thread from its authenticated Agent Manager context; a run ID or
session ID supplied by UI state is only a selector and never authority.

Do not make Agent Manager's `run-agent` evidence endpoint double as BAS proof,
forward `X-Agent-Identity-Token` or grant headers from the browser, return owner
credentials, mint browser receipts locally, or persist session URLs, node URLs,
lease IDs, or credentials in UI/local storage. Add controller/takeover and
handback after this observer contract proves exact owner association and revokes
observer grants on run termination/lease loss. Fresh customized sessions can be
a later explicit BAS open operation under the saved run capability and named
profile; they must not be silently conflated with a run's attached session.

## Evidence read

- `scenarios/agent-manager/api/internal/handlers/browser_capability.go`:
  run identity verification, capability lookup, authorize-open, and bounded
  run-reported evidence handling.
- `scenarios/agent-manager/api/internal/eventlog/types.go`:
  browser session evidence is explicitly not trusted until BAS attests it.
- `scenarios/browser-automation-studio/api/automation/agentaccess/http.go` and
  `handlers/agentaccess/module.go`: BAS-owned broker/API boundary and credential
  context extraction.
- `scenarios/browser-automation-studio/api/automation/agentaccess/{broker.go,evidence.go,agentmanager.go}`:
  session lifecycle owner, authenticated open authorization, run evidence
  reporter, and owner-to-AM authorization client.
- `scenarios/web-console/api/desktop_proxy.go`, `continuity_handlers.go`, and
  `main.go`: same-origin proxy, scenario discovery, and discovered scenario URL
  patterns.

This is a static source audit. Runtime route availability, deployment discovery
health, and the browser pane's live interaction were not verified.
