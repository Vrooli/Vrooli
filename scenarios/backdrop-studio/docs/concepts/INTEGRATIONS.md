# Integrations

Four outbound seams. Each is a named interface with a fake, so every domain test
runs without a live dependency.

| Seam | Scenario | Used for | Hard dependency |
|---|---|---|---|
| `ImageEngine` | `image-tools` | Treatments, model inference, conditioning adapters | **yes** |
| `BrandPalette` | `brand-manager` | Palette slot resolution, contrast authority | **yes** |
| `AssetRegistry` | `asset-studio` | Provenance, cost, disclosure, release of model-backed work | no |
| `BackdropConsumer` | inbound | Released backdrop references | n/a |

---

## `ImageEngine` → `image-tools`

**Everything raster and every model call goes here** (`CMP-004`). This scenario
contains no pixel-manipulation implementation, no provider URL, and no inference
credential.

### Why `ai-gateway` is reached transitively

`image-tools` already registers the gateway as a provider alongside its local
backends, and selects between them from a probed host inventory:

- `api/internal/capabilities` reads OS, CPU, memory, GPU and VRAM through the
  root `vrooli` CLI, and treats unknown VRAM conservatively rather than
  disqualifying a GPU.
- `api/internal/resolver` composes model selection, backend tier, native-vs-derived
  support, and consent weight into one inspectable `Resolution`, readable
  *before* execution via `ExplainResolution`.
- `api/internal/ai/ai_gateway.go` is a provider-neutral client — image-tools holds
  no provider URL, credential, or remote model configuration, and the gateway
  writes only to a caller-supplied output path.

That is the local-first ladder this scenario needs, already built and tested.
Calling `ai-gateway` directly would duplicate host probing and tier selection and
produce a second, divergent answer to "what can this machine do right now".

**Backdrop Studio therefore has no direct `ai-gateway` dependency.** It names a
role and a profile; the ladder resolves the rest.

### Failure behaviour

Unavailable ⇒ nothing renders. There is no fallback, because a local
reimplementation would violate the boundary that keeps treatments reusable.

The tier-one treatment operations are implemented in `image-tools`; this seam
keeps Backdrop Studio independent of their pixel implementation.

---

## `BrandPalette` → `brand-manager`

Resolves `$brand.*` treatment parameters into concrete colours for the active
brand at render time (`CMP-002`), and is the authority for contrast questions the
legibility gate asks about *intent* (the measurement itself is local).

### Why resolution is late

A slot stays a slot in the durable style record. Resolving at write time would
convert a reusable style into a single-brand asset and destroy the property that
makes the catalog a system.

### Failure behaviour

Unavailable ⇒ nothing renders, in either lane. An unresolved slot is refused by
name and **never defaulted** (`CMP-003`); a silently substituted colour would
defeat the palette lock the treatment layer exists to provide.

`brand-manager` exposes the documented `BrandsService/GetTokens` read surface.
The seam still ships with a fake so compose tests remain deterministic.

---

## `AssetRegistry` → `asset-studio`

Releases model-backed candidates so provenance, cost, and disclosure are recorded
once (`REL-002`).

### The split is by cost, not convenience

| Candidate | Path | Why |
|---|---|---|
| Model-backed (`guided`, `synthesized`) | through `asset-studio` | Real spend, real disclosure obligation, non-reproducible pixels — all things a provenance ledger exists for |
| Procedural (`procedural`, `procedural-treated`) | released locally | No spend, no disclosure obligation, reproduces from a seed — a ledger would add a dependency that buys nothing |

`REL-003` makes this a requirement rather than an optimisation: the procedural
catalog must keep working when `asset-studio` is down. That is what makes the
scenario deployable as a desktop product.

### What `asset-studio` already provides

Verified against its PRD and schema:

- `OT-P0-007` render provenance — spec version, backend, model, seed, parameters
- `OT-P0-008` cost accounting — estimated and actual, reportable by campaign
- `OT-P0-012` disclosure metadata at birth
- `OT-P0-014` asset reference by stable identifier
- `studio_assets` carries status, alt text, disclosure, `ai_generated`, and
  credential claims with **no identity column** — the release spine is generic

The generalized conformance verdict accepts an automated measurement without an
identity version while preserving operator-only identity verdicts. Model-backed
release paths can therefore record placement fitness through `asset-studio`.

### What this plan added: `StudioService.IngestExternalAsset`

Every other asset in `asset-studio` comes into existence through `CreateRender`,
which means `asset-studio` produced it. That left a producing scenario two bad
options — duplicate the disclosure rules locally, or ship model-backed output
with no disclosure — and Backdrop Studio took a third, refusing to release
model-backed work at all. Correct, and it meant a working capability shipped
disabled.

The RPC admits bytes with their producing-scenario provenance and returns an
asset in `in_review`. Three properties make it usable here:

- **It requires no identity record.** Backdrop Studio binds a scaffold and a
  palette, not a character. The conditioning field is generic — a scaffold, a
  reference image set, a trained adapter, or a look — which is what `OT-P0-016`
  already modelled and this proves.
- **It refuses unlabelled synthetic media.** A request declaring a model-backed
  strategy with no model id or no prompt is refused rather than recorded with a
  gap, because such an image cannot be reproduced or audited.
- **It is a door into the release path, not around it.** An ingested asset lands
  in review and runs every check `ReleaseAsset` applies.

The facts on the request come from the *render* that produced the candidate, not
from whoever called Backdrop Studio's release API — see `SEAMS.md`,
`release.ProvenanceSource`.

---

## `BackdropConsumer` — the inbound surface

The compound-value seam. A consumer resolves a backdrop by stable identifier and
receives its URI, surface, reserved regions, measured contrast, disclosure state, and alt
text — **never bytes** (`REL-004`).

| Consumer | Uses it for | Status |
|---|---|---|
| `landing-page-business-suite` | Hero and sign-up backdrops | Resolves a released reference and placement with a complete fallback |
| `content-desk` | Promotional surfaces | Not yet wired |
| `seo-optimizer` | Open-graph cards | Not yet wired |
| `scenario-to-android` | Play listing backdrops and device-frame composition | Supplies screenshots and surface ids; receives conforming composed bytes |
| `scenario-to-ios` | App Store listing backdrops and device-frame composition | Supplies screenshots and surface ids; receives conforming composed bytes |
| `scenario-to-desktop` | Splash imagery | Candidate future consumer |

Passing the reserved regions as data is what makes the seam worth having: the
consumer positions its own copy correctly without re-deriving a layout judgement
that was already made and measured here.

## Related

- `ARCHITECTURE.md` — the layer boundary these seams implement
- `FLOWS.md` §5 — degradation behaviour per unavailable dependency
- `../internal/SEAMS.md` — the seam registry and its fakes
