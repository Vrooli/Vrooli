# Local port proxy browser regression

`https-tunnel.test.ts` exercises the shared proxy in a browser through a local
HTTPS edge that forwards to the proxy over loopback HTTP. It models TLS
termination, `Host` preservation, and `X-Forwarded-Proto: https`; it does not
connect to Cloudflare or validate Access policy.

The test starts at a separate-site chat page, clicks the tunneled proxy URL,
follows the capability redirect, then checks that the proxied app's HTML, CSS,
JavaScript, cookie, and opaque-origin RPC all work. The RPC assertion covers
both its CORS preflight and actual response. Chromium runs by default:

```bash
pnpm exec vitest run src/__tests__/e2e/local-port-proxy/https-tunnel.test.ts
```

WebKit can be added when Playwright's WebKit browser and host libraries are
installed (`pnpm exec playwright install webkit`; Playwright may also require
its platform dependencies):

```bash
VROOLI_LOCAL_PROXY_WEBKIT=1 pnpm exec vitest run src/__tests__/e2e/local-port-proxy/https-tunnel.test.ts
```

The HTTPS certificate and private key are test-only fixtures; Playwright
ignores certificate validation for this local edge.
