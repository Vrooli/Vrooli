---
name: "brand-manager"
description: "Run the brand-identity pipeline with the brand-manager CLI: explore and pick logo candidates, refine and vectorize a mark, compose container-styled icon targets, apply them to a scenario's /public/* layout and electron assets, and validate the result."
license: "CC-BY-4.0"
metadata:
  kind: "skill"
  schemaVersion: 1
  modes: ["tools"]
  tags: ["branding","design-system","logo","icons","assets","validation"]
  status: "active"
  revision: 2
  createdAt: "2026-03-20T00:00:00Z"
  updatedAt: "2026-09-15T00:00:00Z"
  requires:
    scenarios: ["image-tools"]
    commands: ["brand-manager", "image-tools", "vrooli"]
  origin:
    kind: "authored"
---
## Tools focus: Brand Manager

brand-manager owns brand meaning: logo candidates, marks, container styles,
product lines, target profiles, render composition, apply and validation.
image-tools owns the pixel and vector operations (generate, edit, object
removal, vectorize, rasterize, icon pack); brand-manager calls it only over its
HTTP/Connect API.

The pipeline concept and the exact target profiles live in
`scenarios/brand-manager/docs/concepts/BRAND-ASSET-PIPELINE.md`.

### 1. When to use this

| Goal | Command |
|------|---------|
| List brands | `brand-manager brands list` |
| Create a brand | `brand-manager brands create --name <name> --display-name <name>` |
| Link a brand's slug, mark, style and line | `brand-manager brands set-identity <id> --slug <slug> --mark-asset <asset> --container-style <style-id> --product-line <line-id>` |
| Explore concepts | `brand-manager candidates explore --brand-id <id> --brief "<product>" --concept "<direction>" --variations 2` |
| Explore in a product line's style | `brand-manager candidates explore --brand-id <id> --style-reference-brand <sibling-slug> --concept "<subject>" --variations 2` |
| Inspect a candidate's prompt, role and model | `brand-manager candidates get <candidate-id>` |
| Import an existing image | `brand-manager candidates import --brand-id <id> --asset-id <asset> --concept "<label>"` |
| Refine a candidate | `brand-manager candidates refine <candidate-id> --instruction "…"` / `--mask-asset <asset>` / `--remove-background` / `--vectorize` |
| Pick the mark | `brand-manager candidates pick <candidate-id>` |
| List container styles / product lines | `brand-manager styles list` / `brand-manager styles lines` |
| Preview an apply | `brand-manager apply preview --brand-id <id> --scenario <scenario> --elements icons` |
| Apply the icon set | `brand-manager apply run --brand-id <id> --scenario <scenario> --elements icons` |
| Validate a scenario | `brand-manager provider validate <scenario>` |

### 2. The logo-refresh workflow

1. **Explore when the direction is open.** `candidates explore` renders every
   concept × variation in parallel with an illustration model
   (`image.generate.default`). Ask for variations once a direction is chosen,
   not before.
2. **Match the product line.** For a brand in a line whose first product has an
   approved mark, pass `--style-reference-brand <that-slug>`. Every concept is
   then rendered from that brand's approved raster through the edit role, so the
   new mark inherits the tile, glow, line weight and polish, and only the subject
   changes. Describe only the subject in `--concept`.
3. **Describe the subject, not the rendering.** explore writes the rendering
   brief itself: a finished icon on the container style's tile, with its accent
   and glow. Do not ask for "flat vector line art". The vector role
   (`--prefer-vector`) draws flat geometric marks that look unfinished, so use it
   only for a deliberately flat direction.
4. **Show candidates on the Logo page.** The UI (`/logo`) is the operator
   surface: gallery, compare, pick/reject/restore, refine and the target preview
   sheet. CLI `candidates list` also lists them.
5. **Pick deliberately.** `candidates pick` promotes one candidate to the
   brand's canonical mark. A raster pick is vectorized through image-tools first
   (keeping white and the container accent, clipped to the tile) and the
   VECTORIZED child is what is picked; the raster stays in history and becomes
   the style reference for the next product in the line.
6. **Apply.** The scenario declares its targets in `.vrooli/service.json`
   (`branding.brand` + `branding.targets`); `apply run --elements icons` writes
   every declared target to the `/public/*` layout, the marked `index.html`
   block, a relative-src `site.webmanifest`, the og/twitter meta, and the
   electron PNGs, `icon.ico` and `icon.icns`.
7. **Validate and publish.** `provider validate <scenario>` proves the declared
   targets; `vrooli scenario restart <scenario>` (use a UI-only restart when
   available) publishes the new assets.

### 3. Judgment rules

- **Never apply an unpicked candidate.** Apply renders the brand's picked mark.
- **Follow `docs/marketing/strategy/ASSETS.md`.** Do not recompose or recolour an
  accepted logo without an accepted decision; differences beyond anti-aliasing
  are shown to the operator as a candidate, never applied silently.
- **Propose a small mark when 16 px loses detail.** A small mark is an optional
  simplified mark for targets at or below `small_mark_threshold_px`; it is
  applied only after the operator picks it.
- **The branding declaration is a prerequisite for apply.** A scenario with no
  `branding` block has no targets to write.

### 4. Troubleshooting

| Symptom | Cause and action |
|---------|------------------|
| `image-tools is not reachable` | Start it: `vrooli scenario start image-tools`. |
| Concepts look flat, geometric or primitive | They came from a vector model. Check `candidates get <id>`: the role should be `image.generate.default` (or `image.edit.default` with a style reference) and the model an illustration model such as `bytedance-seed/seedream-4.5`. Drop `--prefer-vector` and any `--role image.*.vector`/`image.generate.logo`. |
| Concepts look muddy or ignore the prompt; a "local model" warning | They rendered on local SD 1.5. Do not pass `--fallback-policy local_only`; explore permits the cloud tier by default. |
| A new product does not look like its line | Re-run explore with `--style-reference-brand <first-product-slug>`. |
| Vectorize parity is off | Tune `--keep-color`, `--inset-px` and `--tolerance-px`, or re-pick. |
| UI still shows old icons after apply | Rebuild/restart the UI component; an icon-only change lives under `ui/public/**`, which the `pnpm_vite` builder treats as an input. |
| `provider validate` reports `declared-icon-targets` | Run `apply run --elements icons` for the declared brand, then re-validate. |

### 5. References

- Pipeline + profiles: `scenarios/brand-manager/docs/concepts/BRAND-ASSET-PIPELINE.md`
- Public asset convention: `docs/concepts/PUBLIC_ASSETS.md`
- Asset decisions: `docs/marketing/strategy/ASSETS.md`
