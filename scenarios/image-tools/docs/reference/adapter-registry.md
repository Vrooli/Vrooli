# Adapter Registry Reference

The canonical, machine-readable adapter catalog seed is
[`api/internal/adapters/adapters.seed.json`](../../api/internal/adapters/adapters.seed.json)
(schema + load/validate policy + field reference: the file's own
`field_reference` block and the package doc in
[`api/internal/adapters/adapters.go`](../../api/internal/adapters/adapters.go)).
This page is the human catalog of the **conditioning layer**: what an
adapter is, the three kinds and their execution contracts, how a request
carries an adapter stack, and the no-vaporware gating that keeps an
un-proven adapter off the execution path.

Authored 2026-06-27 alongside the conditioning/adapter layer + bring-your-own
import work. It is the sibling of the [Model Registry](model-registry.md):
models *serve* operations; adapters *modify* an operation on a compatible base
model. See also [`../internal/TECHNIQUE-SUBSTRATE.md`](../internal/TECHNIQUE-SUBSTRATE.md)
for how an adapter stack composes with a technique, and
[`../concepts/DOMAINS.md`](../concepts/DOMAINS.md) for the `adapters` domain
boundary.

## What an adapter is (and why it is a separate catalog)

An **adapter** is a conditioning modifier — a LoRA, ControlNet, or
IP-Adapter — that changes *how* an existing operation runs on a compatible
base model rather than serving an operation of its own. It has a different
ontology from a base model, so it lives in its own package
(`internal/adapters`) instead of becoming a `kind` field on `models.Model`
(decision C1, stated in `adapters.go`):

- An adapter has **no operations**. Modelling it as a model would force every
  adapter to pretend it serves operations.
- It carries a **single compatible architecture**, a **consent weight**, a
  **scale range**, and (for ControlNet) a **preprocessor** — fields a base
  model does not have.
- Adapters and models have **independent governance and licensing
  lifecycles**; entangling them would couple two unrelated catalogs.

What the two catalogs *do* share is the download/checksum/artifact spine
(`internal/fetch`) and the architecture SSOT (`internal/models`) — so an
adapter's compatible architecture can never drift from the model registry's
notion of that architecture, and adapter weights install through the same
governed fetch path as model weights.

## Adapter fields

Each catalog entry (`adapters.Adapter`) carries:

| Field | Meaning |
|---|---|
| `id` | Stable catalog id referenced by the resolver, picker, and CLI. |
| `name` | Human display name. |
| `kind` | `lora` \| `controlnet` \| `ip-adapter` — the conditioning contract (below). |
| `architecture` | The **single** base architecture this adapter conditions. Compatibility is data: `adapter.architecture == model.architecture`. Empty / `none` is rejected at load (it would match nothing). |
| `weight` | The consent weight contributed (`none`/`low`/`high`); the effective op weight is `max(op, adapters...)`. |
| `preprocessor` | ControlNet only: `canny`/`depth`/`pose`/`segment`/`none` — the analysis step that derives the control image from a raw input. Any other kind must declare `none` (or omit it). |
| `scale_range` | `min`/`max`/`default` conditioning strength. Required; `min <= default <= max`. |
| `size_mb_approx` | Approximate download size. |
| `source` | Fetch strategy: `assets[]` (single-file) \| `repo` (diffusers snapshot, revision must be pinned before enable) \| `local_path` (mutually exclusive). Checksum is pinned on first real download, never hand-written. |
| `capability_labels` | License, `commercial_use`, NSFW capability, base-model lineage, known risks, provenance. |
| `ready` | Execution proven on this architecture by an attended GPU run. **Always `false` in the seed** (no vaporware); flipped only after proving. |
| `pending` | For a not-Ready adapter, what blocks proving it. |
| `enabled` | Seed default enabled state (overlaid at runtime by the SQLite store). An adapter must *also* be Ready before it is offered for execution. |

## The three kinds and their execution contracts

Each kind has a distinct execution contract, proven on an attended GPU run
before it flips Ready. The Go arg-builders live in
[`internal/technique`](../../api/internal/technique/technique.go); the Python
appliers live in
[`internal/sidecar/py/image_tools_sidecar/_adapters.py`](../../api/internal/sidecar/py/image_tools_sidecar/_adapters.py).
Adapters run **only on the diffusers sidecar** — `stable-diffusion.cpp` fails
closed on any conditioning request (it never silently drops a requested
modifier), so the resolver routes a conditioned request to a diffusers-backed
model.

| Kind | What it does | Needs an image? | Wire format (Go → Python) |
|---|---|---|---|
| `lora` | A low-rank weight delta fused into the base UNet / text encoder (style / subject / step-count). Self-contained. | No | `--lora <path>:<scale>` — single weight file inside the installed dir; `load_lora_weights` + `set_adapters` + `fuse_lora`. |
| `controlnet` | A structural conditioner driven by a preprocessed control image (canny edges / depth / pose / segmentation). | Yes (control map) | `--controlnet <dir>:<scale>:<image>` — installed diffusers repo dir (`ControlNetModel.from_pretrained`); the scale + image are the trailing two colon-free fields (right-split). |
| `ip-adapter` | An image-prompt adapter driven by a reference image (identity / style transfer). | Yes (reference) | `--ip-adapter <weightfile>:<scale>:<reference>` — the adapter weight (`load_ip_adapter` + `set_ip_adapter_scale`); one IP-Adapter per pipeline (a second spec is an error, not a silent drop). An IP-Adapter also needs its CLIP **image encoder**, so it installs from a pinned `repo` subset (`allow_patterns` for the weight + `image_encoder/`); the weight resolver walks the install dir and skips the `image_encoder/` subdir so the adapter weight (not the encoder's `model.safetensors`) is picked. |

The scale + trailing image are colon-free local paths so the Python parser can
right-split (`rsplit(":", 2)`) cleanly; a Go↔Python parity test asserts the
emitter and parser agree (see TECHNIQUE-SUBSTRATE.md). LoRA / IP-Adapter
appliers must run **before** any CPU offload (offload moves weights off the GPU
and breaks late fusion).

## The compatibility rule

Compatibility is **data, not code** (decision CC2):

```
adapter.CompatibleWith(modelArch)  ⇔  adapter.architecture == modelArch
```

`Registry.Compatible(arch)` is the raw architecture filter. The
resolver/picker further restrict it to **installed + enabled + Ready** before
offering an adapter for a generation — so the catalog filter and the
"runnable right now" set are deliberately different surfaces.

## Ready / no-vaporware gating

Every seeded adapter ships `ready: false` (enforced at load by
`validateSeedInvariants` — a seed adapter that ships `ready: true` fails boot).
An un-Ready adapter is **inspectable and installable but never offered for
execution**: `ResolveConditioning` fails closed on the first adapter that is
not compatible, not enabled, not Ready, or not installed, and on a missing
required reference image. The honest rejection (never a silent drop, never an
unconditioned run) is the point.

Flipping Ready is an **attended, operator-only** step (agents never commit and
never flip Ready):

1. Prove the kind×architecture on a GPU host via the matching `make gpu-e2e-*`
   target (see [`../internal/TESTING.md`](../internal/TESTING.md)).
2. The operator flips `ready: true` for that adapter in
   `adapters.seed.json` and pins the repo revision (ControlNets).
3. The operator stamps `last_validated_at` on the matching `type: manual`
   validation in the requirement module, then re-runs the target as the
   durable regression guard.

Until then, a request that names the adapter is rejected with the adapter's
`pending` reason.

## Scale range + consent weight

- **Scale range.** A request's `scale` of `0` means "use the adapter's
  default"; any value is clamped to `[min, max]`. A zero/absent range is
  treated as "no clamp".
- **Consent weight (decision C4).** The effective op weight is
  `EffectiveWeight = max(opWeight, adapter weights...)`. IP-Adapters (and
  identity-class LoRAs) carry `high` because a reference image can carry a real
  person's likeness; structural ControlNets carry `none`. So conditioning an
  op can *only raise* its consent requirement, never lower it.

## Conditioning images

LoRAs need no image. ControlNet and IP-Adapter ride an image input, and the
plumbing is the same for both:

1. The request carries a **`conditioning_image_key`** — a blob key (or path)
   for the control map (ControlNet) or reference image (IP-Adapter).
2. At execution the engine **materializes** each adapter's conditioning image
   from its blob to a local file in the job's temp dir and rewrites the key in
   place (`Engine.materializeAdapterImages` in `internal/ai/engine.go`),
   failing closed on a missing blob.
3. The technique arg-builder then emits the local path into the
   `--controlnet …:<image>` / `--ip-adapter …:<reference>` spec, and the
   sidecar opens it (`PIL.Image.open(...).convert("RGB")`).

For a ControlNet, the conditioning image handed to the sidecar is **already
the final control map** — the sidecar never re-preprocesses. The intended
auto-preprocess path runs the adapter's declared preprocessor op as a Look step
*before* the conditioned generate, so by execution time the control image is a
concrete map; a pre-made map (e.g. the output of the `canny` op) is accepted
the same way. An IP-Adapter's reference image is mandatory (it is the adapter's
prompt). A ControlNet with no image *and* a real preprocessor is allowed with a
warning (it will auto-derive the map); a ControlNet with neither is rejected.

## Seed catalog

All seven seed entries ship `ready: false` (no vaporware). License-conditional
entries (OpenRAIL-family ControlNets) also ship disabled by default and carry a
`commercial_use_notes` pin-a-revision caveat.

| ID | Kind | Arch | Weight | Preprocessor | Scale (min/def/max) | ~Size | License (commercial) | Enabled (seed) |
|---|---|---|---|---|---|---|---|---|
| `lcm-lora-sdv1-5` | lora | sd15 | none | — | 0.0 / 1.0 / 2.0 | ~135 MB | MIT ✅ | yes |
| `ip-adapter-sd15` | ip-adapter | sd15 | high | — | 0.0 / 0.6 / 1.0 | ~45 MB | Apache-2.0 ✅ | yes |
| `controlnet-canny-sd15` | controlnet | sd15 | none | canny | 0.0 / 1.0 / 2.0 | ~1450 MB | OpenRAIL-M ⚠️ conditional | no |
| `controlnet-depth-sd15` | controlnet | sd15 | none | depth | 0.0 / 1.0 / 2.0 | ~1450 MB | OpenRAIL-M ⚠️ conditional | no |
| `controlnet-openpose-sd15` | controlnet | sd15 | none | pose | 0.0 / 1.0 / 2.0 | ~1450 MB | OpenRAIL-M ⚠️ conditional | no |
| `controlnet-canny-sdxl` | controlnet | sdxl | none | canny | 0.0 / 1.0 / 2.0 | ~2500 MB | OpenRAIL++-M ⚠️ conditional | no |
| `controlnet-depth-sdxl` | controlnet | sdxl | none | depth | 0.0 / 1.0 / 2.0 | ~2500 MB | OpenRAIL++-M ⚠️ conditional | no |

> `canny` is a **deterministic** preprocessor (`internal/ops/canny.go`);
> `depth` reuses the existing `depth_map` op and `segment` reuses the `segment`
> op as their preprocessor. SDXL ControlNets carry a higher-VRAM caveat
> (SDXL + ControlNet may exceed 8 GB) and their e2e is gated behind a VRAM
> check.

## CLI verbs

The `adapters` domain (`cli/domains/adapters/`) mirrors the Connect-RPC
`AdaptersService` — read, enable/disable, install/remove, guided import, and
compatibility:

| Command | Purpose |
|---|---|
| `image-tools adapters list [--kind <k>] [--architecture <arch>]` | List catalog entries with effective enabled/Ready/install state. |
| `image-tools adapters get <id>` | Show one adapter in detail (scale range, license, pending reason, install record). |
| `image-tools adapters compatible [--model <id>] [--architecture <arch>]` | Adapters compatible with a base model (by architecture). |
| `image-tools adapters enable <id> [--disable]` | Toggle the persisted enabled overlay over the read-only seed. |
| `image-tools adapters install <id> [--wait]` | Submit a governed download job for the weights (`--wait` blocks once). |
| `image-tools adapters remove <id>` | Remove installed weights. |
| `image-tools adapters inspect <source>` | Dry-run preview of an import source: inferred kind + architecture (with confidence + evidence), license, size, proposed id. Installs nothing. |
| `image-tools adapters import <source> [--kind …] [--architecture …] [--preprocessor …] [--id …] [--name …] [--attest-commercial-rights]` | Register + install a custom adapter from an HF repo / URL / local path. |
| `image-tools adapters doctor` | Catalog integrity check — enabled adapters must declare a concrete fetch strategy (assets / repo / local_path); exits non-zero on findings. |

An imported adapter follows the same provenance + commercial-rights discipline
as an imported model: it registers at local tier with a user-imported
provenance label, and public/BYOK serving of an unverified-license import is
refused unless `--attest-commercial-rights` is passed.

## How a request carries an adapter stack

A generation request carries a **typed, ordered adapter stack** built from the
repeatable conditioning flags on the `ai` submit verbs
(`generate` / `img2img`, plus the read-only `explain`):

```bash
image-tools ai generate --model sd-1.5 --prompt "a serene mountain lake" \
  --lora lcm-lora-sdv1-5:1.0 \
  --controlnet controlnet-canny-sd15:1.0:<control-image-key> \
  --ip-adapter ip-adapter-sd15:0.6:<reference-image-key>
```

Spec forms (the first colon-delimited field is always the adapter id):

- `--lora id[:scale]`
- `--controlnet id[:scale[:conditioning_image_key]]`
- `--ip-adapter id[:scale]:reference_image_key`

A trailing numeric field is the scale; a trailing non-numeric field is the
conditioning/reference image key.

The resolver (`ResolveConditioning`) then validates each entry against the
chosen model's architecture, orders the stack
**LoRA → ControlNet → IP-Adapter** (decision C6), clamps each scale to its
range, and elevates the consent weight to `max(op, adapters...)`. Add
`--explain` to any submit verb for a read-only dry run that prints which
model/technique would run and the resolved adapter stack (id, kind, scale)
without submitting.

## Not yet built (future work)

These were intentionally deferred this round (no vaporware — they are named
where relevant, not implied to exist):

- **`pose` preprocessor.** The OpenPose `pose_estimate` model is not wired, so
  `controlnet-openpose-sd15` cannot auto-derive its skeleton. It still works
  today by supplying a pre-made pose map via `conditioning_image_key`.
- **Compound `controlnet-{canny,depth,pose}` Looks.** The auto-preprocess
  "run the preprocessor op as a Look step, then the conditioned generate"
  convenience Looks were not built. The ControlNet path works today by passing
  a pre-made / pre-preprocessed conditioning image, which the engine
  materializes from a blob to a local file before execution.

## Keeping this current

- **Adding/changing an adapter:** hand-edit `adapters.seed.json` (do not
  mass-edit). Every entry validates structurally at load; the seed must uphold
  the license + no-vaporware invariants (`ready: false`, a `pending` reason,
  no outright non-commercial entry, conditional entries disabled by default
  with notes).
- **Never hand-write a checksum** — it is captured on first real download.
- **Pin a repo revision** before enabling a `repo`-sourced (diffusers
  snapshot) ControlNet.
- **Proving Ready is attended + operator-only** — see the gating section above
  and [`../internal/TESTING.md`](../internal/TESTING.md).
</content>
</invoke>
