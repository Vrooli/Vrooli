# Architecture

Backdrop Studio is a **composer**. It ships no raster implementation and no
model-provider configuration. What it owns is the judgement no engine beneath it
is positioned to make: *is this image right for this page?*

## The layer boundary

```
CONSUMERS   landing pages · content desk · SEO optimization
            mobile store listing asset consumers
                          ▲ references a backdrop by stable id (never bytes)
            ──────────────┼──────────────────────────────────────────────
JUDGEMENT   backdrop-studio
            style catalog · surfaces · scaffold · placement · legibility gate
            ──────────────┼──────────────────────────────────────────────
                          ▼ delegates every execution
EXECUTION   image-tools        brand-manager      asset-studio
            treatments,        palette slots,     provenance, cost,
            inference,         contrast           disclosure, release
            host routing
                          ▼ (transitive — never called directly)
                    execution gateway
```

### The rule that decides ownership

> A capability any scenario could want belongs in the engine.
> A judgement only landing pages need belongs in the studio.

| Question | Owner | Why |
|---|---|---|
| What does "halftone at 60lpi, 15°" mean? | `image-tools` | A generic verb. Deposited once, every scenario can screen an image forever. |
| Which model serves this request? | `ai-gateway` | Typed inference. A role and a profile, never a model name. |
| What is *our* blue, and does it clear contrast? | `brand-manager` | Already the palette authority and already carries a contrast domain. |
| What did this cost, is it AI, may it be released? | `asset-studio` | Already owns provenance, cost, candidate sets, disclosure-at-birth. |
| **Is this image right for this surface?** | **backdrop-studio** | Nothing above answers it. Needs the taxonomy, the surface registry, the placement model, and the legibility gate. |

## Why `ai-gateway` is not a direct dependency

`image-tools` already registers the gateway as a provider alongside its local
backends (`api/internal/ai/providers.go`), and selects between them from a
probed host-capability inventory (`api/internal/capabilities`, which reads
CPU, memory, GPU and VRAM through the root `vrooli` CLI). Its `resolver` package
composes model selection, backend tier, and technique into a single inspectable
`Resolution` value that can be read back *before* execution.

That is exactly the local-first ladder this scenario needs, already built and
already tested. Backdrop Studio therefore asks `image-tools` for an operation and
inherits the ladder. Calling `ai-gateway` directly would duplicate the
host-probing and tier-selection logic, and would produce a second, divergent
answer to "what can this machine do right now".

**Consequence for degradation:** when local hardware cannot serve a request,
`image-tools` routes it onward. Backdrop Studio does not decide that; it records
which path ran (`RND-004`) so a degraded result is attributable.

## The generation ladder

Four strategies. Each style declares exactly one.

| Strategy | Base image from | Model | Cost | Reproducible |
|---|---|---|---|---|
| `procedural` | seeded code | no | none | exactly, from seed |
| `procedural-treated` | seeded code, then treatment | no | none | exactly, from seed |
| `guided` | scaffold → conditioned generation → treatment | yes | metered | composition yes, pixels approximately |
| `synthesized` | prompt → generation → treatment | yes | metered | least |

### The invariant

> Every strategy terminates in the same deterministic treatment pass.
> Raw model output is never released.

This is the load-bearing claim of the design. A synthesized image screened at a
fixed angle and remapped onto two brand inks cannot read as generic model
output, because everything that makes generic model output recognisable — the
smooth photoreal gradient, the arbitrary palette, the absence of process — is
destroyed by the treatment. The property is **structural**: it does not depend
on anyone writing a sufficiently good prompt.

It is also what makes the catalog a *system* rather than a collection. Duotone,
posterization and dithering discard the source's colour and remap luminance onto
the active brand's inks, so every image is forced into the palette without
anyone art-directing each one.

### Why `guided` matters most

The scaffold is not a sketch of the picture. It is a diagram of its *structure*
— horizon, focal mass, framing geometry, depth ramp, and the reserved-region voids
drawn as flat featureless areas — rendered as a depth field or an edge
drawing and submitted as conditioning.

Three properties follow, none available from prompting alone:

1. **The quiet zone survives generation**, because it is expressed to the model
   as structure rather than hoped for.
2. **Composition is authored and reproducible**, so a family of images shares a
   layout rather than sharing a vibe.
3. **One scaffold plus many prompts yields many worlds at one composition** —
   which is what makes a remix feature meaningful rather than a reroll button.

`image-tools` already ships ControlNet with both `canny` and `depth`
preprocessors, plus LoRA and IP-Adapter, and carries `commercial_use` licensing
metadata per adapter (`api/internal/adapters`). The scaffold feeds straight in.

## Disclosure follows from the strategy

The AI-generated flag is **derived**, never set (`REL-001`). `guided` and
`synthesized` are AI-generated; `procedural` and `procedural-treated` are not.
Marking a procedurally drawn halftone as AI-generated would be a false claim, so
over-disclosure is refused on the same footing as under-disclosure.

An honest limitation worth stating: a treatment pass that quantises to two inks
will likely destroy an invisible pixel watermark. The durable record is the
signed manifest and `asset-studio`'s provenance row — not something recoverable
from the pixels.

## Storage posture

SQLite, in-process. This scenario persists **metadata and references only**:
styles, briefs, render jobs, candidates, legibility verdicts, released
references. Rendered blobs stay behind `image-tools`; released assets stay behind
`asset-studio`'s blob seam. See `DATA.md`.

## Related

- `DOMAINS.md` — the eight bounded contexts and their build order
- `DATA.md` — what is durable here and what is a reference
- `INTEGRATIONS.md` — the four outbound seams and their failure behaviour
- `../internal/DECISIONS.md` — durable decisions and their rationale
- `../internal/PROBLEMS.md` — upstream changes this scenario depends on
