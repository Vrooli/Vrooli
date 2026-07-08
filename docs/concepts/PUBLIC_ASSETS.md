# The `/public/*` Public-Asset Convention

> **Owner:** operator-direct for the security contract; agents may extend the
> "what belongs here" guidance as new world-readable surfaces appear, subject to
> operator review. **Siblings:** [`ARCHITECTURE.md`](./ARCHITECTURE.md) (the
> technical how), [`ECOSYSTEM.md`](./ECOSYSTEM.md) (how a scenario fits the
> whole). The enforcement lives in `tunnel-manager` (edge Access bypass) and
> `brand-manager` (validation + autofix); this doc is the canon they cite.

## What this convention is

**Anything a scenario serves under the URL path prefix `/public/*` is publicly
fetchable by anonymous, unauthenticated clients — by design.**

The contract is the **URL path**, not any framework's source-directory name. A
file that ends up at `https://<host>/public/apple-touch-icon.png` is covered; a
file at `https://<host>/apple-touch-icon.png` is not, regardless of where it
lives in the repo.

## Why it exists

Every scenario routed through the Cloudflare named tunnel sits behind a
**Cloudflare Access (Zero Trust)** application: every request returns a `302`
redirect to the Access login page unless the caller carries the Access session
cookie. That is correct for the app itself, but it breaks **anonymous system
fetchers** that legitimately need a handful of assets:

- **iOS Add-to-Home-Screen** fetches the web manifest + `apple-touch-icon` with
  a cookieless system fetcher → it receives login HTML, not a PNG → the home
  screen falls back to a generic letter glyph.
- **Link unfurlers / Open Graph crawlers** are likewise anonymous → blank
  previews.

The most-missed sub-point: the **manifest itself must be public**. If
`/site.webmanifest` returns `302`, the browser discards the *entire* manifest
(name + every icon) via a CORS failure, so exempting only the PNGs still fails.

`tunnel-manager` enforces the convention at the Cloudflare edge with a narrow,
guard-railed Access **Bypass** policy scoped to `<host>/public` (a plain
"Allow + Everyone" policy does **not** work — it still shows a login; only a
**Bypass** decision serves the raw asset with no redirect). Cloudflare's
most-specific-path precedence keeps the operator's host-level protective app
authoritative for everything outside `/public`.

## The security contract (read before putting anything here)

- `/public/*` is **world-readable by design**. Treat it as if it were on a
  public CDN with no auth, ever.
- Therefore **nothing sensitive may ever be served under `/public/`** — no
  tokens, no per-user data, no internal API responses, no anything that depends
  on the Access gate for its confidentiality.
- The exemption is *only* a Bypass for the `/public` path prefix. Every other
  path stays fully gated by the operator's primary Access application, which
  `tunnel-manager` never modifies or deletes.

## What belongs under `/public/`

- Branding + PWA assets: favicons, `apple-touch-icon`, `icon-192`/`icon-512`,
  the web manifest (`site.webmanifest`), `logo.svg`.
- Open Graph / Twitter-card preview images (`og-image.png`).
- `robots.txt`, `sitemap.xml`, and anything else meant to be world-readable.

## What must never go there

- Anything private or sensitive (see the security contract above).

## Adopter requirements

- Emit explicit `<link>` / `<meta>` tags in the document head that point at
  `/public/...` (absolute paths). iOS A2HS and OG crawlers read those tags, so
  pointing them at the public surface is what makes the flow work.
- A PWA manifest at `/public/site.webmanifest` resolves its icon `src` values
  relative to the manifest URL, so relative icon paths keep working — but
  `start_url` and `scope` must point at the **app root** (`/`), not into
  `/public/`, so the installed app launches into the app and not the asset
  folder.

## Accepted edges

- A bare-root `/favicon.ico` probe (browsers fetch it with no `<link>` tag)
  stays gated. This is cosmetic only: real flows (A2HS, OG) read explicit tags,
  which adopters point at `/public/...`.
- Framework source-dir nesting can look odd (e.g. Vite serves `ui/public/` at
  URL root `/`, so the convention surface lives at `ui/public/public/` →
  `/public/`). The contract is the **URL** prefix, not the directory name.
