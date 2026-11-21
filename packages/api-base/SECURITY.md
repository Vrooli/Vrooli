# API Base security: browser origin policy

Every scenario UI server built with `createScenarioServer` enforces one browser
origin policy. This document describes what it protects, how it decides, and how
a scenario declares the cross-origin access it needs. The code is
`src/server/origin-policy.ts`, and the behavioural specification is
`src/__tests__/server/origin-policy.test.ts`.

## Threat

The operator's browser can reach every scenario UI server: through
`http://localhost:<port>` and through the tunnel (`https://<scenario>.<domain>`).
A UI server proxies its scenario's API, and some APIs treat loopback callers as
the owner. Without a policy, any page the operator happens to visit could run
`fetch('http://localhost:<port>/api/...')`, then read the answer or change data.

Before November 2026, almost every scenario set `corsOrigins: '*'`. Each one
answered `Access-Control-Allow-Origin: *`, so any site could read
unauthenticated data. CORS also never stops a "simple request" from being sent,
such as a cross-site `<form>` POST or a `no-cors` fetch, so writes went through
regardless. The wildcard had been a workaround. The old allow-list compared
`Origin` against a fixed list instead of the request's own host, so it refused
legitimate same-origin requests through the tunnel. Module scripts send
`Origin` even when same-origin.

## Policy

The browser reports where each request comes from in Fetch Metadata
(`Sec-Fetch-Site`). The server applies the following rules before any route,
proxy, static file, or SPA fallback runs:

| Request | Result |
|---|---|
| `Sec-Fetch-Site: same-origin`, or `none` (typed address, bookmark) | Served |
| No `Sec-Fetch-Site` and no `Origin` (CLI, service, relay) | Served |
| No `Sec-Fetch-Site`, `Origin` equal to `Host` (older browsers, including through the tunnel) | Served |
| Cross-origin, `Origin` listed in `allowedOrigins` | Served with credentialed CORS (exact origin echo) |
| Cross-origin GET/HEAD (or its preflight) to a path listed in `publicPaths` | Served with `Access-Control-Allow-Origin: *`, no credentials |
| Cross-origin GET/HEAD navigation (link, frame), except `object`/`embed` | Served: the page is shown to the operator, never to the calling site |
| Anything else: cross-site or same-site fetches, form posts, `no-cors` subresources, preflights from undeclared origins, `Origin: null` | `403 Cross-origin request refused`, `Cache-Control: no-store` |

Notes on the rules:

- **Same-site counts as cross-origin.** Every `localhost` port is same-site with
  every other, and so is every subdomain of the tunnel domain. One scenario's
  page cannot call another scenario's UI server unless the target declares it.
- **WebSocket handshakes** follow the same rules, without the navigation and
  public-path exemptions. Browsers do not apply CORS to WebSockets: any page can
  open one and read every frame. Express middleware never sees upgrades, so
  `proxyWebSocketUpgrade` checks the handshake itself. Its default is
  same-origin; `createScenarioServer` passes the scenario's policy. A server
  that terminates WebSockets itself calls
  `SAME_ORIGIN_POLICY.evaluateUpgrade(req)` (see app-monitor's `ui/server.js`).
- **CORS headers belong to the browser-facing server.** `proxyToApi`, the
  embedded proxy, and the host proxy drop `Access-Control-*` headers from
  upstream responses. An API's own wildcard therefore cannot widen the UI
  server's policy.

## Relays

Some servers forward a browser request to a different origin: Web Console's
`/_local/<port>` proxy, the `/embedded/<scenario>` proxy, and host scenarios'
`/apps/<id>/proxy` routes (app-monitor, portal). Each relay applies its own
policy first: the scenario policy, plus the capability check in the local-port
proxy. It then drops `Origin`, `Referer`, and `Sec-Fetch-*` before forwarding
(`crossOriginRelay: true`, `withoutBrowserContext`). Those headers describe the
relay's page, often an opaque sandbox origin, not the upstream. The upstream
sees a non-browser caller, and the relay's own decision stands.

The local-port proxy is mounted before the scenario policy. Its sandboxed
documents send `Origin: null` by design, and its capability policy, which
`local-port-proxy.ts` describes, governs them.

A scenario's own `/api` proxy keeps the browser context, so an API can run its
own Fetch Metadata checks as a second layer (money-ledger does).

## Declaring access

Most scenarios declare nothing. The scenario's own pages, the tunnel, Web
Console's `/_local` proxy, host proxies, desktop shells that load the UI URL,
CLIs, and services all work with the default.

```javascript
startScenarioServer({
  uiPort, apiPort, distDir: './dist', serviceName: 'my-scenario',

  // A browser extension or another origin that must call this server with
  // credentials. Each entry is one exact origin.
  allowedOrigins: ['chrome-extension://abcdefghijklmnopabcdefghijklmnop'],

  // Content any origin may read without credentials: versioned public code,
  // or capability URLs whose unguessable path is the authorization. This is
  // typically what opaque-origin sandbox frames import.
  publicPaths: ['/preview/runtime/', /^\/api\/v1\/sessions\/[^/]+\/runtime\//],
})
```

Refused at startup, so an open policy cannot ship by mistake:

- `corsOrigins` (removed): delete the option, or move exact origins to
  `allowedOrigins`;
- `'*'`, patterns such as `https://*.example.com`, `null`, URLs with paths, and
  comma-joined strings in `allowedOrigins`;
- `/`, wildcard, or relative entries in `publicPaths`.

Current declarations:

| Scenario | Declaration | Why |
|---|---|---|
| web-console | `publicPaths` for session file-preview `runtime/` resources | HTML previews render in opaque sandbox frames; the preview id is the capability |
| react-component-library | `publicPaths: ['/preview/runtime/']` | Design compositions in opaque sandbox frames import the versioned React runtime |

## Diagnosing a refusal

A refused request returns JSON with `error`, `detail` (the origin and why), and
`remedy`. With `verbose: true` the server also logs each refusal. A legitimate
caller is refused only when it is a different origin. Open the scenario at its
own address, route the call through a server-side relay, or declare the exact
origin in `allowedOrigins`.

## Out of scope

- **Scenario API ports.** The policy protects UI servers. A scenario API that
  listens on a loopback port and answers browsers directly is a separate
  surface: some Go APIs set `Access-Control-Allow-Origin: *`, and simple
  requests reach any of them. Those APIs need their own Fetch Metadata check.
  `money-ledger`, `landing-page-business-suite`, `lighting-control`, and
  `scenario-to-cloud` already have one.
- **Framing.** Cross-site frame navigations are served, as navigations.
  Defending against clickjacking is the job of `frame-ancestors` and
  `X-Frame-Options` on the documents themselves.
- **Legacy UI servers that do not use `createScenarioServer`.** These set their
  own CORS and are not covered by this policy.
