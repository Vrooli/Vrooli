# `/public/` branding and PWA assets

These files intentionally live under `ui/public/public/`. Vite serves `ui/public/`
at the site root, so this nested directory maps to the URL prefix `/public/`.
The Vrooli public-asset convention keeps install icons, manifests, logos, and
link-preview images anonymously fetchable without exposing the rest of the app.

Do not put secrets or user data here.
