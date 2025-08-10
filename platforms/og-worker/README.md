# Dynamic Open‑Graph Metadata with a Cloudflare Worker (pnpm monorepo edition)

> **Goal:** Serve correct Open‑Graph / Twitter‑card tags for every dynamic URL (profiles, projects, chats…) while your main React/Vite SPA continues to live on your existing origin (DigitalOcean in this guide).  
> A tiny Cloudflare Worker runs **only for crawler UAs**, injects the meta tags, and proxies everyone else straight to the origin.

---

## 🟠 Prerequisites

| Item | Notes |
|------|-------|
| Cloudflare account | Free plan is fine. |
| Domain added to Cloudflare | **Add Site** → change nameservers **or** transfer registration. |
| Origin server (DigitalOcean droplet) | Site already serves `https://yourdomain.com`. |
| **Node ≥ 18 & pnpm ≥ 8** | Monorepo package‑manager. |

---

## 1  Add the Worker package to the monorepo

```bash
# from /packages (should already be done)
pnpm create cloudflare@latest og-worker    # choose “Hello World”, "Worker only", "TypeScript", "No" for git (already set up at project root), "No" for deploy (we'll do that later)
cd og-worker
```

### Workspace wiring

`pnpm-workspace.yaml`
```yaml
packages:
  - apps/*
  - packages/*
```

Root `package.json` (handy scripts):
```jsonc
{
  "scripts": {
    "dev:og": "pnpm --filter og-worker wrangler dev",
    "deploy:og": "pnpm --filter og-worker wrangler deploy",
    "tail:og": "pnpm --filter og-worker wrangler tail"
  }
}
```

---

## 2  Configure `wrangler.toml`

> **Path:** `apps/og-worker/wrangler.toml`

```toml
name              = "og-worker"
main              = "src/index.ts"
compatibility_date = "2025-05-01"
workers_dev       = false            # we’ll use zone routes

[vars]
BOT_UA      = "(facebookexternalhit|Twitterbot|Slackbot-LinkExpanding|Discordbot|LinkedInBot|Pinterest|BingPreview)"
ORIGIN_HOST = "do-origin.yourdomain.com"   # set in DNS step

[[routes]]  # apex
pattern   = "yourdomain.com/*"
zone_name = "yourdomain.com"
[[routes]]  # sub‑domains
pattern   = "*.yourdomain.com/*"
zone_name = "yourdomain.com"

# add staging / test zones the same way
```

---

## 3  DNS setup

| Record | Type | Value | Proxy |
|--------|------|-------|-------|
| `@`          | **A** | _DO IP_ | **Proxied 🟠** |
| `do-origin`  | **A** | _same IP_ | **DNS‑only ⚪** |

`@` must be orange so Cloudflare (and the Worker) sit in front of traffic.  
`do-origin` stays grey to stop recursive loops when the Worker calls the origin.

---

## 4  Worker source (minimal)

Create `src/index.ts` (replace template code):
```ts
import { parse } from "regexparam";

interface Meta { title: string; description: string; image: string; url: string; }
type Env = { BOT_UA: string; ORIGIN_HOST: string };

const routes = [
  { pattern: "/u/:username", api: getUserMeta },
  { pattern: "/p/:projectId", api: getProjectMeta },
  { pattern: "/c/:chatId",    api: getChatMeta }
].map(r => ({ ...r, route: parse(r.pattern) }));

export default {
  async fetch(req: Request, env: Env) {
    const ua = req.headers.get("user-agent") ?? "";

    // 1 — Humans → origin
    if (!new RegExp(env.BOT_UA, "i").test(ua)) {
      return proxyToOrigin(req, env);
    }

    // 2 — Bots → HTML with OG tags
    const url = new URL(req.url);
    for (const { route, api } of routes) {
      const match = route.pattern.exec(url.pathname);
      if (!match) continue;
      const params = Object.fromEntries(route.keys.map((k, i) => [k, match[i + 1]]));
      const meta = await api(params, env);
      return htmlResponse(render(meta));
    }
    return new Response("Not found", { status: 404 });
  }
};

// Proxy helper
const proxyToOrigin = (req: Request, env: Env) => {
  const url = new URL(req.url);
  url.hostname = env.ORIGIN_HOST;
  return fetch(url.toString(), {
    cf: { resolveOverride: env.ORIGIN_HOST },
    headers: req.headers,
    method: req.method,
    body: req.body,
    redirect: "manual"
  });
};

/* ------------ meta fetchers ------------ */
async function getUserMeta({ username }: any, env: Env): Promise<Meta> {
  const res = await fetch(`https://${env.ORIGIN_HOST}/api/v1/users/${username}`, {
    cf: { resolveOverride: env.ORIGIN_HOST }
  });
  if (!res.ok) return fallbackMeta(username);
  const u = await res.json();
  return {
    title: `${u.name} (${u.handle}) on Site`,
    description: u.bio ?? "",
    image: u.avatar ?? fallbackImg,
    url: `https://yourdomain.com/u/${username}`
  };
}
// …getProjectMeta / getChatMeta …

/* ------------ helpers ------------ */
const fallbackImg = "https://yourdomain.com/default.png";
const fallbackMeta = (id = ""): Meta => ({ title: "Not found", description: "", image: fallbackImg, url: `https://yourdomain.com/${id}` });
const esc = (s: string) => s.replace(/[&<>"']/g, c => ({"&":"&amp;","<":"&lt;",">":"&gt;","\"":"&quot;","'":"&#39;"}[c]!));
const render = (m: Meta) => `<!doctype html><html><head><meta charset='utf-8'>\
<meta property='og:title' content='${esc(m.title)}'>\
<meta property='og:description' content='${esc(m.description)}'>\
<meta property='og:image' content='${m.image}'>\
<meta property='og:url' content='${m.url}'>\
<meta name='twitter:card' content='summary_large_image'></head><body></body></html>`;
const htmlResponse = (html: string) => new Response(html, {
  headers: {
    "Content-Type": "text/html",
    "Cache-Control": "public, s-maxage=3600, stale-while-revalidate=86400"
  }
});
```

---

## 5  Deploy & develop

```bash
pnpm dev:og        # wrangler dev (local)
pnpm deploy:og     # push to Cloudflare
pnpm tail:og       # live logs in production
```

Wrangler CLI is workspace‑local, so scripts always use the version pinned in `apps/og-worker`.

---

## 6  Verify OG tags

```bash
curl -H "User-Agent: Twitterbot" https://yourdomain.com/u/matt | grep og:
```

Use **Facebook Sharing Debugger**, **X Card Validator**, **Slack preview** for real‑world tests.

---

## 7  Troubleshooting

| Symptom | Fix |
|---------|-----|
| Worker never runs | Apex A‑record must be 🟠; zone route `yourdomain.com/*` must exist. |
| 503 loop | Always call origin via `https://${env.ORIGIN_HOST}` + `resolveOverride`. |
| 404 on static assets | Narrow Worker routes to just `/u/* /p/* /c/*` **or** serve assets from a Worker `assets` binding. |
| Tail shows only /socket.io | Narrow routes or ignore those paths in code. |

---

## 8  Handy pnpm workspace commands

```bash
pnpm -r exec wrangler deployments               # list for all workers
pnpm --filter og-worker wrangler kv:namespace create META_KV  # create KV
pnpm --filter og-worker wrangler rollback <id>  # quick rollback
```

---

## 9  Optional enhancements

* **Workers KV** – cache rendered metadata for 1 h + `stale-while-revalidate`.
* **Dynamic OG images** – separate Worker route using Satori or pages‑plugin‑vercel‑og.
* **CI/CD** – GitHub Actions: `pnpm --filter og-worker wrangler deploy --branch $GITHUB_REF_NAME`.

---

### 🎉 You now have crawler‑perfect social previews integrated seamlessly into a pnpm monorepo.
Pull‑request‑friendly, CI‑friendly, and zero extra servers!

