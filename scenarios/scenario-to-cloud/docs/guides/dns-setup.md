# DNS Setup for VPS Deployments

Scenario-to-cloud supports two DNS patterns when deploying to a VPS:

1) **Cloudflare Worker proxy (OG meta injection)**: Cloudflare sits in front of the VPS and a Worker injects Open Graph metadata for crawlers.
2) **Basic DNS (no Worker)**: DNS points directly to the VPS with no Cloudflare Worker layer.

This doc explains both setups and how they relate to scenario-to-cloud preflight checks.

---

## Option A: Cloudflare Worker proxy (recommended when you need dynamic OG tags)

**When to use:** You want the Cloudflare Worker in `platforms/og-worker/` to intercept crawler requests and inject OG meta tags before serving content from your VPS.

**How it works (high level):**
- Public traffic goes through Cloudflare (orange cloud).
- The Worker runs on matching routes (apex and subdomains).
- The Worker fetches the origin from a **different hostname** that is DNS-only (grey cloud) to avoid loops.

**Example DNS records**

| Type | Name | Value | Proxy | Purpose |
|------|------|-------|-------|---------|
| A | @ | 203.0.113.10 | Proxied (orange) | Public site (Worker intercepts) |
| A | do-origin | 203.0.113.10 | DNS-only (grey) | Origin host the Worker fetches |
| A | api | 203.0.113.10 | Proxied (orange) | API subdomain |
| CNAME | www | @ | Proxied (orange) | WWW redirect |

**Critical notes**
- The apex (`@`) **must be proxied** so the Worker runs in front of traffic.
- The origin host (example: `do-origin.example.com`) **must be DNS-only** so the Worker can call it without recursion.
- Your Worker config must route your public domain and set a different `ORIGIN_HOST`.

**Worker config alignment (wrangler.jsonc)**
- Routes: `example.com/*` and `*.example.com/*`
- `ORIGIN_HOST`: `do-origin.example.com` (or another DNS-only hostname)

---

## Option B: Basic DNS (no Worker)

**When to use:** You do not need dynamic OG tag injection and want a direct path to the VPS.

**How it works (high level):**
- DNS points directly to the VPS.
- Your reverse proxy (Caddy) handles HTTPS and routing.

**Example DNS records**

| Type | Name | Value | TTL | Purpose |
|------|------|-------|-----|---------|
| A | @ | 203.0.113.10 | 300 | Public site (direct to VPS) |
| A | api | 203.0.113.10 | 300 | API subdomain |
| CNAME | www | @ | 300 | WWW redirect |

---

## How scenario-to-cloud validates DNS

Scenario-to-cloud preflight checks include:
- DNS for the public domain resolves to the VPS.
- Ports 80/443 are reachable for Caddy/Let's Encrypt.

In **Option A**, DNS still points to the VPS, but through Cloudflare for the public hostnames. The Worker-origin hostname (DNS-only) must also resolve directly to the VPS so the Worker can fetch it. In **Option B**, the public domain records point directly to the VPS with no Cloudflare proxy layer.

---

## Quick decision guide

- Need OG meta injection for crawlers? Use **Option A** and the Cloudflare Worker.
- No dynamic OG requirements? Use **Option B** for the simplest DNS setup.
