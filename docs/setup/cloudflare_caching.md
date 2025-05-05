Below is a self-contained **README: Cloudflare Caching for Vrooli** you can drop into your docs folder (e.g., `docs/cloudflare-caching.md`). It covers the key cache layers you’ll touch, the knobs you can turn, and a decision matrix for picking sane defaults.

---

# Cloudflare Caching for Vrooli

> **TL;DR**  
> • Cache static assets aggressively at the edge.  
> • Cache bot-only metadata HTML for ~1 h with _stale-while-revalidate_.  
> • Never cache proxy passthroughs (set `cacheTtl: 0` in the Worker).  
> • Use **Cache Rules** to fine-tune by path if needed.

---

## 1. Cache layers

| Layer | What it stores | Typical TTL | Control point |
|-------|----------------|-------------|---------------|
| **Browser cache** | Assets already on the user’s device | hours-to-days | `Cache-Control` header from your origin or Worker |
| **Edge cache (CDN)** | Copies of responses in Cloudflare’s 300+ PoPs | seconds-to-days | `Cache-Control`, **Cache Rules**, or Worker `cf` options |
| **Cloudflare cache KV** (optional) | Key/value objects you write from Workers | until you delete / TTL expires | Worker KV APIs |

---

## 2. Key Cloudflare headers / settings

| Setting (source) | Effect |
|------------------|--------|
| `Cache-Control: public, max-age=31536000` (origin header) | Tell browsers & edge to keep static assets (hashed JS/CSS) for 1 year. |
| `s-maxage=X` | Overrides `max-age` **for edge only**; browsers still respect `max-age`. Good for HTML where you want shorter edge TTL. |
| `stale-while-revalidate=X` | Edge may serve a stale object while it revalidates in background. Smooths cold-cache spikes. |
| `cf: { cacheTtl: 0 }` (Worker fetch option) | Bypass edge cache for this fetch (good for proxy passthrough). |
| **Cache Rules** (Dashboard → Rules → Cache) | Declarative path-matching overrides (e.g., “/api/* → bypass”, “*.webp → edge TTL 1 month”). |
| **Custom Purge** | Manual or API purge by URL, prefix, tag, or everything. Useful when you deploy breaking asset changes. |

---

## 3. Recommended defaults for Vrooli

| Content | Path pattern | Edge TTL | Browser TTL | How to set |
|---------|--------------|----------|-------------|------------|
| **Static build outputs** (`dist/*.js`, `.css`, images) | `/assets/*` | 1 year | 1 year | `Cache-Control: public, max-age=31536000, immutable` emitted by Vite; confirm origin header. |
| **Metadata HTML for crawlers** (served from Worker) | `/`, `/u/*`, `/search*`, etc. | 1 h `s-maxage`, 24 h `stale-while-revalidate` | 0 (`max-age=0`) | `createHtmlResponse()` already sets `public, s-maxage=3600, stale-while-revalidate=86400`. |
| **User-facing SPA HTML** (proxied) | anything not matched by Worker routes | **Bypass** | 0 | Worker `proxyToOrigin()` uses `cf: { cacheTtl: 0 }`. |
| **API JSON** | `/api/*` | Bypass | 0 | Add a **Cache Rule**: _If path matches `/api/*` then Bypass cache_. |
| **Uploads / Media** (future) | `/media/*` | 1 week | 1 day | Serve via a separate static bucket or Pages Functions with explicit `Cache-Control`. |

---

## 4. Choosing TTLs – a quick decision tree

1. **Does the response change only when you redeploy?**  
   **Yes:** set a *very long* TTL (1 month-1 year) and include a content hash in the filename.  
   **No:** continue ↓  

2. **Is the response personalised (cookies, auth)?**  
   **Yes:** bypass edge cache (`cacheTtl: 0` or Cache Rule → Bypass).  
   **No:** continue ↓  

3. **Does freshness matter more than speed?**  
   *Example: a price ticker.*  
   **Yes:** choose short edge TTL (≤60 s) or bypass.  
   **No:** continue ↓  

4. **Is the response for bots only?**  
   *OG tags, sitemap, etc.*  
   **Yes:** 5 min–1 h edge TTL + `stale-while-revalidate` so the first user per region doesn’t pay the DB cost.  
   **No:** pick a middle-ground (10 m–1 h) and monitor hit ratios.

---

## 5. Purging & debugging

### Purge options

| Method | Use when |
|--------|----------|
| **Single URL purge** | One page shows stale data. |
| **Prefix purge** (`/assets/`) | After a new front-end build. |
| **Tag purge** | If you tag related objects from Workers (e.g., all user-profile pages). |
| **Everything** | Major incident / infrastructure reset. |

### Debug flow

1. `curl -I <url>`: look at `CF-Cache-Status` (HIT, MISS, BYPASS, EXPIRED).  
2. If it’s a HIT but the content is wrong → purge that URL.  
3. If it’s a MISS every time → check your headers (maybe `private` or `no-store`).  
4. Use **Cloudflare → Caching → Analytics** to track hit ratios & savings.

---

## 6. Worker snippets you already use

```ts
// Bypass cache for origin proxy
fetch(originURL, { cf: { resolveOverride: env.ORIGIN_HOST, cacheTtl: 0 } });

// Cache bot-HTML for 1 h, let edge serve stale for a day
new Response(html, {
  headers: {
    "Cache-Control": "public, s-maxage=3600, stale-while-revalidate=86400"
  }
});
```

---

## 7. References

* Cloudflare Docs – [Understand Cache-Control](https://developers.cloudflare.com/cache/how-to/configure-cache-control/)  
* Cloudflare Docs – [Cache Rules](https://developers.cloudflare.com/rules/cache/)  
* Cloudflare Docs – [Workers `fetch` `cf` properties](https://developers.cloudflare.com/workers/runtime-apis/request/#cf-properties)  
* Blog – “Caching best practices & max-age gotchas” (Cloudflare, 2024)

---

_Commit suggestion:_  
```bash
git add docs/cloudflare-caching.md
git commit -m "docs: add Cloudflare caching guidelines"
```

You now have a concise guide for anyone working on Vrooli’s infra to understand how, why, and where each cache knob is set.