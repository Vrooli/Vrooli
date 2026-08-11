# Starter catalog

The catalog's empty state offers to seed a starter set (`experience/pages/catalog.json`).
This file says what is in it.

It matters more than a seed file usually does: **the catalog is the product.** A
thin or incoherent starter set makes the whole scenario look like a toy on first
run, and first run is where the judgement gets made.

> **Status: proposed, not decided.** The coverage rule below is an engineering
> constraint and can be treated as settled. The specific style list is an
> art-direction call and belongs to the operator. Do not treat the table as
> approved content — treat it as a shape to fill.

## Coverage rule

The taxonomy declares 44 treatments, 14 subjects and 24 lineages, and it is an
open list. Covering all of it in a seed set is neither possible nor desirable, so
coverage is **tiered**. Each tier is independently shippable.

### Tier 1 — the floor (~15 styles)

The set that must exist before the catalog is worth showing anyone.

| Rule | Why |
|---|---|
| Every treatment **family** appears at least once — reprographic, quantization, ink mapping, optical, synthetic field, procedural structure, displacement, symbolic | Families are the shape of the vocabulary; a family with no example is a section of the product that does not exist |
| At least two styles per `strategy` | One example reads as a special case; two read as a category |
| At least 60% `procedural` or `procedural-treated` | The free, offline, zero-cost lanes carry the first-run experience. A starter catalog that mostly cannot run without a model is a demo, not a product |
| Every page `placement` declared by at least two styles | A placement no style supports is an empty filter result on first use |
| At least one style per surface kind | `product` and `store` mockups exercise different chrome paths (D-010) |
| No style names a living artist | The authoring rule in `taxonomy.md` §Axis 4 |

### Tier 2 — the showcase (~40 styles)

What an implementing agent should aim for when asked to build this out properly.
The review question this tier answers is *"can I see the whole space?"*

| Rule | Why |
|---|---|
| Every treatment with a Tier 1 or Tier 2 `image-tools` operation appears in at least one style | See `../internal/PROBLEMS.md` for the operation tiers — a landed operation with no style demonstrating it is invisible |
| At least one style per treatment family × strategy combination that makes sense | Not every cell is meaningful; a synthetic field does not need a `guided` variant |
| At least eight distinct lineages represented | Lineage is what makes two styles with the same treatment feel unrelated |
| Every store placement demonstrated | Otherwise the store lane is untested in practice |

### Tier 3 — the long tail

Everything else in the taxonomy, added as its operation or generator lands. **A
declared axis value with no style behind it is unbuilt, not wrong.** The list is
open by design; treat gaps as a backlog, not a defect.

> **Depth still beats breadth within a tier.** Fifteen strong styles beat forty
> thin ones — every style is an implicit claim about what good looks like here.
> Build the tier fully, then move up.

## Shape of the set

Proportions, not a prescription. Scale each slice by the tier being built.

| Slice | Tier 1 | Tier 2 | Strategy skew | Notes |
|---|---|---|---|---|
| Non-representational fields | 4–5 | 12–14 | `procedural` | Mesh, flow field, voronoi, reaction-diffusion, caustics, strange attractor. Carry the offline experience and need no subject at all — the widest slice, and the cheapest |
| Treated geometry | 3–4 | 8–10 | `procedural-treated` | Dithered terrain, halftone horizon, riso bands, contour relief, pixel-sorted gradient. Demonstrate the treatment layer on a drawable subject |
| Representational scenes | 4–5 | 10–12 | `guided` | Arcade, coastline, botanical, interior, geological. Where the scaffold earns its place |
| Intricate subjects | 2–3 | 4–5 | `synthesized` | Too complex to scaffold; the escape hatch, deliberately the smallest slice |
| Store-oriented | 2–3 | 4–6 | mixed | Quiet-centre compositions built for a device frame or a feature graphic |

## Open decisions for the operator

1. **Which lineages Vrooli speaks in.** The taxonomy lists ten. A brand does not
   credibly speak all of them at once; picking three or four as primary would make
   the catalog read as a point of view rather than a sampler.
2. **Whether the seed set is also the sellable pack.** `CAT-007` makes style packs
   portable and `MONETIZATION.md` treats them as a product. If the starter set is
   also the free tier, its coverage rule is a pricing decision too.
3. **How opinionated the defaults are.** A style ships with a treatment chain and
   gate thresholds. Tight defaults teach; loose defaults invite exploration.

## Related

- [`taxonomy.md`](taxonomy.md) — the axis enums the coverage rule is stated against
- [`surfaces.md`](surfaces.md) — the surface kinds a style declares
- [`../business/MONETIZATION.md`](../business/MONETIZATION.md) — free / metered placement
