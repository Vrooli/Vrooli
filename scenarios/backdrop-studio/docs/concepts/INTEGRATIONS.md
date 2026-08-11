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

### Known gap

The treatment operations do not exist yet. See `../internal/PROBLEMS.md`,
first entry — this is the critical path.

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

### Known gap

Whether a brand *token read* surface exists is unverified. See
`../internal/PROBLEMS.md`. The seam ships with a fake so `compose` proceeds.

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

### Known gap

`studio_conformance_verdicts.identity_version_id` is `NOT NULL` and its `basis`
CHECK admits only identity-flavoured values, so a placement-fitness verdict
cannot be recorded. See `../internal/PROBLEMS.md`, second entry. This gates the
model-backed lanes only.

---

## `BackdropConsumer` — the inbound surface

The compound-value seam. A consumer resolves a backdrop by stable identifier and
receives its URI, copy-safe region, measured contrast, disclosure state, and alt
text — **never bytes** (`REL-004`).

| Consumer | Uses it for | Status |
|---|---|---|
| `landing-page-business-suite` | Hero and sign-up backdrops | Hero currently hardcoded; see PROBLEMS |
| `content-desk` | Promotional surfaces | Not yet wired |
| `seo-optimizer` | Open-graph cards | Not yet wired |
| `scenario-to-desktop` / `-android` / `-ios` | Splash and store imagery | Candidate future consumer |

Passing the copy-safe region as data is what makes the seam worth having: the
consumer positions its own copy correctly without re-deriving a layout judgement
that was already made and measured here.

## Related

- `ARCHITECTURE.md` — the layer boundary these seams implement
- `FLOWS.md` §5 — degradation behaviour per unavailable dependency
- `../internal/SEAMS.md` — the seam registry and its fakes
