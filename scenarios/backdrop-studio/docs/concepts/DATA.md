# Data

## Storage posture

SQLite, in-process. No external resource dependency.

**This scenario persists metadata and references only.** Rendered image bytes
stay behind the `image-tools` blob path; released model-backed assets stay behind
`asset-studio`'s blob seam. Nothing here returns image bytes over a consumer RPC.

That constraint is deliberate and load-bearing: it keeps the release path honest
(one system of record for spend and disclosure) and keeps this database small
enough that a style catalog is cheap to export, diff, and ship as a product.

## Durable records

| Record | Durable | Notes |
|---|---|---|
| `style` | yes | The unit of the catalog. Versioned; immutable once released against. |
| `style_version` | yes | Full axis, strategy, treatment chain, gates, reserved regions, permitted surfaces. |
| `scaffold_preset` | yes | Named parameterised composition presets. |
| `surface` | yes | Output target: pixel geometry, permitted placements, geometry authority, confirmation date. |
| `brief` | yes | The operator's intent for one composition: subject text, brand, placement, seed. |
| `render_job` | yes | Lifecycle status, resolved plan, execution path, timing. |
| `candidate` | yes | One produced image *reference*, its seed, and its resolved plan. |
| `legibility_verdict` | yes | Measured worst-pixel ratio, threshold, overlay region, placement, pass/fail. |
| `release` | yes | The released backdrop reference and its consumer-facing metadata. |
| rendered bytes | **no** | Held by `image-tools` / `asset-studio`. |

## The Style record

The heart of the model. Illustrative shape — the proto contract is authoritative.

```jsonc
{
  "id": "cyanotype-arcade",
  "version": 3,

  // ── the five-axis taxonomy — validated against declared enums (CAT-001)
  "axes": {
    "role":      "ambient",
    "subject":   "statuary_architecture",
    "treatment": ["halftone", "duotone"],
    "lineage":   "cyanotype",
    "placement": ["full_bleed", "split_panel", "framed_inset"]
  },

  // ── exactly one, and it governs which blocks below are permitted (CAT-002)
  "strategy": "guided",

  // ── guided only
  "scaffold": {
    "preset": "arcade",
    "params": { "bays": 3, "horizon": 0.60, "focal": [0.5, 0.72] },
    "conditioner": "depth"                    // depth | canny
  },

  // ── model-backed only. A role and a profile, never a model name (CMP-005)
  "generation": {
    "role":    "image.generate.default",
    "profile": "PROFILE_QUALITY_FIRST",
    "prompt_template": "a colonnade of {bays} arches opening onto a still sea, {season} light",
    "negative": "text, watermark, lens flare, hyperreal skin"
  },

  // ── runs on every strategy, always last (RND-001)
  "treatment_chain": [
    { "op": "halftone", "params": { "lpi": 60, "angle": 15, "dot": "round" } },
    { "op": "duotone",  "params": {
        "paper":  "$brand.surface",           // resolved at render time (CMP-002),
        "ink":    "$brand.primary",           // never baked into the record
        "accent": "$brand.accent",
        "accent_band": [0.88, 1.0]            // spot ink on the top luminance band only
    } }
  ],

  // ── reserved regions: each declares how foreground content meets it (CAT-004, D-009)
  "reserved_regions": [
    { "kind": "overlay",   "role": "headline", "x": 0.05, "y": 0.19, "w": 0.56, "h": 0.55 }
    // an "occlusion" region — a device frame or card — is gated on focal
    // placement instead of contrast; nothing is measured beneath it
  ],
  "surfaces":  ["web.hero", "web.hero-mobile"],   // geometry comes from the surface registry (SUR-001)
  "gates":     { "min_contrast": 4.5, "scrim": "auto" },

  "lineage_ref": "riso-arcade@3"              // the version this was forked from
}
```

### Why palette slots are not resolved at write time

A `$brand.*` slot stays a slot in the durable record and resolves only during
composition (`CMP-002`). This is what lets **one style yield a correct image per
brand** — the property that turns a catalog into a system rather than a
collection of finished pictures. Persisting a resolved colour would silently
convert a reusable style into a single-brand asset.

### The accent band

`accent_band` applies the third ink only across a narrow luminance range, so the
brand accent reads as a deliberate highlight rather than a third of the image.
Two-plus-one spot printing works this way in practice, and it is the difference
between a duotone that looks branded and one that looks tinted.

## Verdicts are per placement, not per candidate

A `legibility_verdict` binds to a candidate **and a placement** (`LEG-004`). One
candidate carries several verdicts — it may pass as a desktop full-bleed hero and
fail as a mobile crop, and both facts must be recorded. A single pass/fail on the
candidate would hide the failure that actually ships.

## Retention

| Data | Retention |
|---|---|
| Styles and versions | Indefinite — the catalog is the product |
| Briefs | Indefinite; small, and they explain why a release looks as it does |
| Render jobs and candidates | Bounded; unselected candidates are collectable after a declared window |
| Verdicts | Follow their candidate |
| Releases | Indefinite — a consumer may still reference them |

Candidate references are collectable but **releases are not**. A consuming page
holds a stable identifier, so deleting a release breaks a live surface.

## Related

- `DOMAINS.md` — which domain owns which record
- `ARCHITECTURE.md` — why bytes live elsewhere
- `../reference/taxonomy.md` — the axis enums the catalog validates against
