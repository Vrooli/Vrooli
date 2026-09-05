# Backdrop Studio

Art-directed ambient imagery for landing pages, app stores and social surfaces —
generated, measured, and released with its provenance intact.

An **ambient** image is the stage: attention passes *through* it to the copy on
top. That is the role this scenario produces, and it is not a synonym for
"background image" — a backdrop that competes with the headline has failed at
the one job it has.

## What it does

Give it a **style** and a **surface** and it returns a rendered candidate, the
perceptual measurements that candidate passed, and the reserved regions a
consumer should keep its copy inside.

- **A catalog of 40 styles** across 24 visual lineages, each classified on five
  axes and carrying its own treatment chain, parameters, ink defaults, reserved
  regions and quality bar. The catalog is versioned data, applied by version,
  and an operator-authored style is never overwritten by an upgrade.
- **Three source lanes.** `procedural` ships what a generator drew;
  `procedural-treated` runs an `image-tools` treatment chain over it; `guided`
  and `synthesized` reach an image model, the first conditioning generation on a
  composition scaffold through ControlNet.
- **Thirteen procedural generators** — horizon, arcade, terrain, metaball field,
  flow field, voronoi, reaction-diffusion, caustics, mesh gradient, contour,
  truchet, strange attractor and nebula. Each is a pure function of
  `(preset, size, seed, params)`: no clocks, no global RNG, no I/O.
- **18 declared surfaces** from a 1440×720 hero to a 2048×2732 App Store
  screenshot, each carrying the citation its geometry came from — because some
  geometry is ours to choose and some belongs to a store that will reject a
  wrong number at submission.
- **Two gates before anything ships.** A legibility gate measures overlay text
  contrast; a perceptual gate measures whether the composition survived its own
  treatment at all. The second exists because high-contrast noise passes the
  first with ease.

## Why the gates matter

An audit rebuilt this scenario from source and rendered every style. Twelve of
sixteen failed outright, and one of the four that rendered was illegible moire
where a colonnade should have been — while every Go unit test passed, because
the suite tested against a fake executor below the wire.

So the rule here is that **a style is not shippable until its exact bytes have
made a round trip through a running `image-tools` and come back as an image.**
Everything else is inference. `api/integration/` is that lane, and it refuses to
judge anything against a stale binary.

The perceptual gate came from the same audit. It measures four things on the
pixels — how much of the source composition is still readable, how much of the
tonal range is occupied, how much the ink density varies across the frame, and
whether reserved regions are quieter than the rest — and every threshold is
derived from rendering the whole catalog and scoring it, never chosen a priori.

## The studio

Ten pages, each reading the real catalog:

| Route | What it is for |
|---|---|
| `/catalog` | Every style as a rendered specimen, filtered by any taxonomy axis |
| `/styles/:id` | One style in full: chain, resolved parameters, prompt, regions, perceptual margins, and the candidate behind real page copy |
| `/sweep` | One style across a seed range — is the style good, or was that a good seed? |
| `/remix` | Fork a style, change one axis, see it beside its parent, save it with lineage |
| `/placements` | The same style at each placement it declares |
| `/compose` | Resolve a plan and see what it will cost before spending it |
| `/surfaces` | Every output target with its geometry and its authority |
| `/candidates` | A batch of candidates with the verdict each carries |
| `/backdrops` | Released backdrops |
| `/settings` | Theme and locale |

## Running it

```bash
make setup   # build API + UI, install deps, install the scenario CLI
make start   # start API + UI in the background
make test    # the scenario suite
```

Render something:

```bash
backdrop-studio catalog list
backdrop-studio render submit --style cyanotype-arcade --surface web.hero \
  --placement full_bleed --seed 7
```

Regenerate the committed evidence — contact sheets, treatment gallery,
generator sheet, perceptual corpus:

```bash
make integration-evidence
```

## What it depends on

`image-tools` for every deterministic treatment and for model-backed generation;
`brand-manager` for the palette a style's `$brand.*` slots resolve against, with
each style declaring its own ink defaults so a cold install still renders;
`asset-studio` for the provenance and disclosure record a model-backed release
carries. Only the first is required — the procedural catalog keeps working when
the other two are down, which is what makes this deployable as a desktop
product.

## Documentation map

| Need | Start here |
|---|---|
| The five classification axes and what draws what | [`docs/reference/taxonomy.md`](docs/reference/taxonomy.md) |
| Output surfaces and their cited geometry | [`docs/reference/surfaces.md`](docs/reference/surfaces.md) |
| What belongs in the seeded catalog and why | [`docs/reference/starter-catalog.md`](docs/reference/starter-catalog.md) |
| Architecture and domain boundaries | [`docs/concepts/ARCHITECTURE.md`](docs/concepts/ARCHITECTURE.md), [`docs/concepts/DOMAINS.md`](docs/concepts/DOMAINS.md) |
| Cross-scenario seams and their fakes | [`docs/internal/SEAMS.md`](docs/internal/SEAMS.md) |
| Known defects, and the ones already closed | [`docs/internal/PROBLEMS.md`](docs/internal/PROBLEMS.md) |
| How to reproduce every committed artifact | [`docs/internal/EVIDENCE.md`](docs/internal/EVIDENCE.md) |
| Per-style ship verdicts | [`docs/evidence/catalog/verdicts.md`](docs/evidence/catalog/verdicts.md) |
| Testing protocol | [`docs/internal/TESTING.md`](docs/internal/TESTING.md) |
| Env vars, ports, CLI config | [`docs/reference/configuration.md`](docs/reference/configuration.md) |
| API endpoints and CLI commands | [`docs/reference/api-endpoints.md`](docs/reference/api-endpoints.md), [`docs/reference/cli-commands.md`](docs/reference/cli-commands.md) |

## Working rules

1. **Evidence is reproducible or it is not evidence.** Every artifact under
   `docs/evidence/` is produced by a command named in
   [`docs/internal/EVIDENCE.md`](docs/internal/EVIDENCE.md). A screenshot from a
   throwaway probe has been deleted from this repository once already.
2. **A catalog retune is a seed version, not an edit.** Each
   `api/internal/catalog/seed/vN.json` carries `retune_reasons` naming every
   value it changes and why. A number chosen for a reason nobody wrote down gets
   re-litigated forever.
3. **Spatial treatment parameters are relative, never pixels.** A value in
   pixels ties a style to one delivery size; this catalog renders the same style
   from a 240px edge to a 2732px one.
4. **Refuse rather than substitute.** A subject no generator draws is an error,
   not an excuse to render a nearby picture. A model-backed release with no
   asset-studio is refused, not downgraded to a procedural candidate wearing a
   synthetic label.
5. **Keep the durable seams**: i18n wiring, accessibility roles, `data-testid`
   selectors, design tokens, the responsive shell floors, and the adopted
   primitives under `ui/src/components/ui/` — prefer
   `react-component-library adoptions apply` over hand-rolling one.
