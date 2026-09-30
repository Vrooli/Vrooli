# `/public/` branding + PWA assets — intentional nesting

These files live in `ui/public/public/` **on purpose**. Vite's `publicDir` is
`ui/public/` and copies its contents to the dist root, so a file at
`ui/public/public/logo.svg` is served at the **URL** `/public/logo.svg`.

The fleet convention (see `docs/concepts/PUBLIC_ASSETS.md`) is that anything served
under the URL path prefix `/public/*` is world-readable by anonymous clients. The
contract is the **URL prefix**, not the framework source-dir name. Keeping the
branding/PWA/OG asset set here lets the tunnel-manager Cloudflare Access bypass
(scoped to `<host>/public`) make these fetchable by iOS Add-to-Home-Screen and OG
crawlers without weakening Access on the rest of the app.

Do **not** put anything sensitive here — it is public by contract. Runtime/root files
that must stay at the URL root (e.g. `robots.txt`) live one level up in `ui/public/`.
