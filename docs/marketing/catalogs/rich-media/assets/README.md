# Rich Media — Assets

Ground-truth uploaded assets that JSON entries reference. The schemas in [`../characters/`](../characters/), [`../scenes/`](../scenes/), and [`../products/`](../products/) point at files in this folder by relative path.

## Folder structure

```
assets/
  logos/                # canonical brand logos (logo.svg, logo-dark.svg, favicon.png, etc.)
  voice-samples/        # persona voice references for TTS / voice-model selection
  product-shots/        # canonical screenshots of Vrooli scenarios (UI, mobile, terminal)
  character-sheets/     # composite reference images per character
                        # (also accessible from characters/<slug>.character-sheet.png as the canonical-per-character file)
```

## Discipline

- **Single source of truth.** A logo, a voice sample, a UI screenshot lives once in this folder and is referenced from every JSON that needs it. Updates land here; the JSONs continue pointing at the same path.
- **No transient assets.** Per-render outputs (one-off generated images, intermediate compositing files) do **not** live here. Per-render outputs live alongside their `render_provenance.output_uri` (typically in a publish-log-tied storage path or a campaign-specific folder).
- **Versioning is by filename.** When a logo or screenshot supersedes a prior version, prefix with the date (e.g., `logo-2026-04-28.svg`) and update references. Do not silently overwrite — referenced JSONs must keep working.
- **Future home.** When the `brand-manager` or `rich-media-studio` scenario ships, this folder is replaced by structured asset storage with versioning and per-asset metadata. Until then, this is the operator-curated stand-in.

## What goes where

| Asset kind | Folder | Naming |
|---|---|---|
| Brand logo files | `logos/` | `logo.svg`, `logo-dark.svg`, `favicon.png`, `og-image.png` |
| Persona voice samples | `voice-samples/` | `<persona-slug>.<format>` |
| Vrooli scenario UI screenshots | `product-shots/` | `<scenario-slug>-<surface>-<state>.png` (e.g., `swarm-manager-web-backlog.png`) |
| Character reference sheets | `character-sheets/` | `<character-slug>.character-sheet.png`; mirror to `../characters/<slug>.character-sheet.png` for short-path access |

## Cross-references

- [`../characters/README.md`](../characters/README.md), [`../scenes/README.md`](../scenes/README.md), [`../products/README.md`](../products/README.md).
- [`../../../strategy/ASSETS.md`](../../../strategy/ASSETS.md) — top-level brand asset registry.
