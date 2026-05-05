# Assets — Brand Asset Registry

Canonical registry of Vrooli's brand assets — logos, favicons, social previews, fonts. Pulled from by advertisers, publisher, designers, and any scenario rendering Vrooli's visual identity.

**Write rule:** operator-curated via accepted `brand-guideline-update` decisions. Agents propose updates (new asset added, asset deprecated, usage rule changed); they do not edit directly.

**Status:** Inventory of current assets in `path:assets/public/` and surrounding paths. **Eventually subsumed by the `brand-manager` scenario** when it ships (see [`prompt-manager skill read brand-manager`](../../scenarios/prompt-manager/store/skills/packs/core/brand-manager/SKILL.md)). Until then, this markdown file is the registry of record.

---

## Logos

### Primary logo

| File | Format | Size | Use |
|---|---|---|---|
| [`assets/public/logo-mask-512x512.webp`](../../assets/public/logo-mask-512x512.webp) | WebP | 512×512 | Primary square logo. App icons, social avatars, default favicon source. |
| [`assets/public/Vrooli-motto-shadow.png`](../../assets/public/Vrooli-motto-shadow.png) | PNG | (motto-bearing variant) | Logo with motto text — used in README hero and on contexts that benefit from the tagline embedded in the image. |
| [`assets/readme-display.png`](../../assets/readme-display.png) | PNG | (display) | README header image. |

### Logo design notes (canonical)

- **The logo is a rabbit.** It's named (and inspired by) the operator's pet rabbit, Jeff. 🐰
- **Speed cue.** The rabbit appears to be zooming — visual cue for speed and automation.
- **Hidden Easter egg.** The rabbit's shape is composed from the letters **V-R-O-O-L-I**. Look closely; they're there.
- **Not a public-facing claim.** The Easter egg and Jeff origin story are internal canon — fun to share when asked, not part of the lead pitch. Worth preserving here so the brand isn't accidentally re-derived without the story.

---

## Favicons

| File | Format | Use |
|---|---|---|
| [`assets/public/favicon.ico`](../../assets/public/favicon.ico) | ICO | Default browser favicon |
| [`assets/public/favicon-16x16.png`](../../assets/public/favicon-16x16.png) | PNG | 16×16 raster favicon |
| [`assets/public/favicon-32x32.png`](../../assets/public/favicon-32x32.png) | PNG | 32×32 raster favicon |
| [`assets/public/apple-touch-icon.webp`](../../assets/public/apple-touch-icon.webp) | WebP | iOS home-screen icon |
| [`assets/public/android-chrome-192x192.webp`](../../assets/public/android-chrome-192x192.webp) | WebP | Android home-screen 192 |
| [`assets/public/android-chrome-512x512.webp`](../../assets/public/android-chrome-512x512.webp) | WebP | Android home-screen 512 |
| [`assets/public/mstile-150x150.png`](../../assets/public/mstile-150x150.png) | PNG | Windows tile |
| [`assets/public/safari-pinned-tab.svg`](../../assets/public/safari-pinned-tab.svg) | SVG | Safari pinned-tab monochrome |

## Social previews

| File | Format | Use |
|---|---|---|
| [`assets/public/og-image.webp`](../../assets/public/og-image.webp) | WebP | Open Graph preview (Facebook, LinkedIn, etc.) and Twitter Card image |

## Fonts

| File | Format | Use |
|---|---|---|
| [`assets/public/sakbunderan-logo-only-webfont.woff2`](../../assets/public/sakbunderan-logo-only-webfont.woff2) | WOFF2 | Brand font, embedded in the logo wordmark. *Logo-only* — not a body or UI font. |

**Note:** No body or UI typography is canonically declared yet. Scenario UIs use their own typography choices today. When the `brand-manager` scenario ships, this file will declare body / heading / mono pairings.

## PWA / app manifests

| File | Use |
|---|---|
| [`assets/public/manifest.dark.manifest`](../../assets/public/manifest.dark.manifest) | Dark-mode PWA manifest |
| [`assets/public/manifest.light.manifest`](../../assets/public/manifest.light.manifest) | Light-mode PWA manifest |
| [`assets/public/browserconfig.xml`](../../assets/public/browserconfig.xml) | Windows tile config |

## Other static assets

| File | Use |
|---|---|
| [`path:assets/public/icons/`](../../assets/public/icons/) | App-specific shortcut icons (create / inbox / search variants in light + dark) |
| [`assets/public/robots.txt`](../../assets/public/robots.txt) | Crawler directives |
| [`assets/public/humans.txt`](../../assets/public/humans.txt) | Credits |
| [`assets/postgresql.svg`](../../assets/postgresql.svg) | PostgreSQL logo (third-party, used in resource-related materials) |

---

## Usage rules

- **Logos must not be re-colored**, distorted, or recomposed without an accepted `brand-guideline-update` decision. The rabbit-shaped letterforms are intentional; reshaping breaks the Easter egg.
- **Favicon sources are derived from the primary logo.** Don't generate variants from a different source.
- **Open Graph image** can be regenerated (e.g., for campaign-specific previews) but a regenerated OG image is a new asset, not a replacement — register it here with a clear distinction.
- **No third-party logos** (other companies, products) belong in this registry. Vrooli scenarios may render third-party logos in their own UIs (e.g., listing supported LLM providers); those don't get cataloged here.

## When ASSETS.md is updated

Trigger conditions for the brand-manager (member) to propose a `brand-guideline-update`:
- New brand asset is created and stored in `assets/`
- An asset is deprecated or replaced
- A usage rule changes (e.g., new approved logo color treatment)
- The `brand-manager` *scenario* ships and this file becomes a pointer at it (one-time transition)

## Cross-references

- [`docs/narrative/PITCH.md`](../narrative/PITCH.md) — the motto / tagline copy that pairs with the logo
- [`docs/narrative/PRESS_KIT.md`](../narrative/PRESS_KIT.md) — composition of assets for external coverage
- [`docs/marketing/IMAGE_STYLE.md`](IMAGE_STYLE.md) — AI image generation style guide (for non-logo imagery)
- [`docs/marketing/BRAND.md`](BRAND.md) — visual identity overview (currently a thin pointer at this file and STRATEGY.md)
- [`scenarios/prompt-manager/store/skills/packs/core/brand-manager/SKILL.md`](../../scenarios/prompt-manager/store/skills/packs/core/brand-manager/SKILL.md) — planned `brand-manager` scenario CLI; eventually subsumes this file
