# Resource Spec — `ace-step`

The governed composition runtime this scenario depends on, with acquisition and
live-service evidence recorded below.

## Purpose Of This Document

Use this document to answer:

- What does the `ace-step` resource contain, and what does it expose?
- What has been measured on the reference host, and what is still assumed?
- What must be obtained or decided before `resources/ace-step/` can be created?

`PRD.md` and [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) declare
`ace-step` as a hard dependency for every composition operation. The managed
resource now exists at `resources/ace-step/`, is checksum-pinned, and is exposed
through the shared activity edge on port 8895. This document remains the durable
record of its contract, measurements, and unresolved variants.

## Provenance Of The Figures Here

Every measurement below comes from a throwaway spike run on the reference host on
2026-09-18, driving `generate_music()` directly with the PyTorch backend on CUDA.
The recipe that produced them currently lives in the `music-generation` core-pack
skill. **This document is where that evidence becomes durable**; the skill is a
waypoint and retires when composition ships.

Confidence labels follow [`model-registry.md`](model-registry.md): `measured` on
this host, `vendor` as published upstream and unaudited, `estimated` as derived
arithmetic.

## What The Resource Wraps

ACE-Step 1.5 is **a transformers custom-code model, not a pip package.** It loads
through `trust_remote_code` against architecture
`AceStepConditionGenerationModel`, with `auto_map` pointing at
`modeling_acestep_v15_turbo.AceStepConditionGenerationModel`. There is no
`pip install acestep` that yields the shipped model.

Two traps are worth recording because both cost a cycle during the spike:

- **The repository name is ambiguous.** `github.com/ace-step/ACE-Step-1.5` is
  v1.5. `github.com/ace-step/ACE-Step` — no suffix — is **v1**, and defaults to
  `ACE-Step/ACE-Step-v1-3.5B`. Installing the unsuffixed repo silently gets the
  wrong generation of the model.
- **The upstream entrypoints are unusable headlessly.** The repo's own CLI is
  wizard-only, and its smoke test targets Apple Silicon (`backend="mlx"`). The
  resource drives `generate_music()` directly.

Licence: **MIT** (`vendor`). The publisher asserts commercial use is permitted on
the basis of licensed, royalty-free, and synthetic training data. That assertion is
unaudited and is recorded as a vendor claim, not a verified fact — it is the single
most load-bearing licence claim in this scenario and belongs in every provenance
record the resource emits.

## Component Inventory

| Component | Role | Size | Confidence |
|---|---|---|---|
| `acestep-v15-turbo` DiT | 8-step distilled generator, **no CFG** | part of the ~11 GB main bundle | `measured` (disk) |
| `acestep-v15-sft` DiT | 50-step generator, CFG active | ~4.5 GB | `measured` (disk) |
| `acestep-v15-base`, `acestep-v15-xl-*` | Larger variants | not downloaded | `vendor` |
| `acestep-5Hz-lm-1.7B` | Planner LM, ships in the main bundle | part of the ~11 GB bundle | `measured` (disk) |
| `acestep-5Hz-lm-0.6B` | Planner LM, **separate download** | small | `measured` |
| `acestep-5Hz-lm-4B` | Planner LM | not downloaded | `vendor` |

Total on-disk for the managed working set: **~7.7 GB** for the selected turbo,
0.6B planner, runtime, CUDA wheels, VAE, and text encoder closure. The 1.7B
planner is bundled to satisfy the upstream handler's directory preflight but is
not the selected runtime planner.

The bundle shipping 1.7B rather than 0.6B is a real trap: initialising the LM
handler with `lm_model_path="acestep-5Hz-lm-0.6B"` against a fresh bundle fails
with `LLM init failed: 5Hz LM model not found`, and the 0.6B weights must be
fetched separately. On the reference card the 1.7B planner **out-of-memories**,
which corroborates rather than contradicts the VRAM tier table in
[`model-registry.md`](model-registry.md).

## Measured Performance

Reference spike: 16 GB card shared with resident tenants, 45 s of audio, turbo
variant at 8 steps. Managed-service smoke: 2026-09-19, offload rung, one-step
request under the same shared-host contention:

| Condition | Wall clock | Confidence |
|---|---|---|
| DiT resident on GPU | **17.9 s** | `measured` |
| DiT offloaded to CPU between steps (`offload_dit_to_cpu=True`) | **55.2 s** | `measured` |
| Managed offload smoke, 1 s request / 1 step | **~46 s wall clock** | `measured` |
| Managed offload smoke output | 5.120 s, 48 kHz stereo PCM WAV | `measured` |

The offload path is **3.1× slower** and fits under roughly 6 GiB. This is not a
performance note — it is the first concrete instance of an `OT-P0-003` degradation
rung, and the resource declares it as one.

SFT behaviour is now measured through the shipped composition pipeline. On
2026-09-19, ten five-second jobs used the preserved spike briefs and requested
seeds with 50 inference steps and CFG 7. All ten produced 5.120-second, 48 kHz
stereo WAV takes on the `offload-dit` rung. Job wall clock was 16.613–570.561 s
(median 22.101 s); the 570.561 s outlier includes capacity/reclaim contention.
The control-plane footprint for the offload rung is 6.0 GiB (`measured`, one
footprint sample), while the per-job claim reservation is 7.0 GiB; this is not
an independently sampled instantaneous process peak. The raw receipts, WAVs,
and proxy table live under the plan artifact directory.

The proxies do not establish listening quality. Recomputed tempo, 30–120 Hz
sub-bass share, 6–14 kHz hi-hat energy, onset density, and RMS are recorded with
medium confidence for numeric extraction and low-to-medium confidence for
cross-run comparison because the spike's original analysis helper was not
retained. Tempo readings remain subject to half/double-time ambiguity. Against
the preserved turbo table, these measurements do not justify claiming SFT is
materially better on authored briefs. Turbo remains the default because it has
the shorter, established production path and SFT's extra CFG/step cost has no
grounded quality advantage in the available evidence.

## Profile Rungs

The resource declares ordered rungs so the capacity broker degrades quality rather
than failing the job. Two independent axes were established by the spike:

| Rung | DiT residency | Planner | Cost | Confidence |
|---|---|---|---|---|
| `full` | resident on GPU | 0.6B | 17.9 s / 45 s audio | `measured` |
| `offload-dit` | offloaded to CPU per step | 0.6B | 55.2 s / 45 s audio | `measured` |
| `refuse` | — | — | explicit failure | — |

There is no CPU-only rung. Composition on CPU is not a degraded rung, it is an
eternity, and declaring one would be dishonest. This has a consequence the rest of
the docs should state plainly: **an operator with no capable GPU gets no
composition at all.** See [`../business/MONETIZATION.md`](../business/MONETIZATION.md).

The 1.7B planner is **not** a rung above 0.6B on this card — it OOMs. It becomes a
rung on hardware with the headroom, which is the same revisit trigger as the
quality tier in [`../internal/DECISIONS.md`](../internal/DECISIONS.md).

## Capacity Declaration

The resource claims through the control-plane broker like every other GPU tenant.
The claim shape follows `resources/kokoro/resource.json`, whose `claim` block is
the working reference for a measured GPU declaration.

| Field | Value | Note |
|---|---|---|
| `resource_kind` | `vram` | |
| `default_preferred_bytes` | 7516192768 | current declaration; refine with a quiet-card managed run |
| `floor_bytes` | 6442450944 | current offload floor declaration; live smoke succeeds under contention |
| `default_priority` | `batch` | composition is optional work; it should yield to interactive tenants |
| `confidence` | `measured` | the spike measured peak; do not declare `estimated` |
| `yield_when_idle` | `true` | |
| `profile.steps` | one per rung above | |

Two host facts constrain this and are recorded so they are not rediscovered:

1. **The broker runs advisory on this host** (`enforce = advisory`,
   `preempt_enabled = false`). A claim that cannot be satisfied returns a queue
   verdict that never resolves, because nothing reclaims. The resource must
   therefore treat the broker as advisory — claim, read the verdict, degrade on
   its advice, and proceed. This is the pattern `image-tools` already ships in
   `api/internal/ai/capacity.go`, where a nil broker means no arbitration and any
   broker error leaves GPU selection untouched.
2. **Fit must be computed against free VRAM, not total.** Inference backends
   commonly size their allocator against total device memory, which over-allocates
   with co-resident tenants. `image-tools` reads
   `vrooli host inventory --json` and uses `vram_bytes - vram_used_bytes`. The
   absence of this check is what caused repeated OOMs during the spike as kokoro
   warmed (1.58 → 2.47 GiB) and ollama spawned llama-servers (3.7–3.8 GiB).

Under no circumstances does this resource stop, evict, or unload another tenant's
service to obtain VRAM. Contention degrades or waits; it never preempts.

## Generation Contract

The parameters that decide the result, and the one that is a defect if left
default:

```text
task_type          "text2music"
caption            the authored brief, verbatim
lyrics             "[instrumental]" for instrumental output
duration           seconds
bpm                a numeric field — NOT text inside the caption
keyscale           e.g. "G minor"
inference_steps    8 (turbo) | 50 (sft)
guidance_scale     1.0 (turbo — inert, the variant carries no CFG) | 7.0 (sft)
seed               int, the batch's only axis of variation
use_cot_caption    False    ← see below
use_cot_metas      False    ← see below
```

**`use_cot_caption` and `use_cot_metas` default to `True` upstream, and that
default is the single largest quality defect found in the spike.** With them on,
the planner LM rewrites the caption and invents metadata before the DiT sees
anything. Measured: an authored 144 BPM *"cold detuned metallic lead, menacing and
restless"* brief reached the model as 65 BPM *"shimmering synth pads, melancholic
arpeggio, ethereal pads"*. Disabling both, at a fixed seed, moved hi-hat band
energy 19.1% → 27.5% and onset density 6.2 → 9.6 per second, and the authored BPM
was honoured.

The resource therefore passes the caller's caption through intact by default. If a
rewrite is ever enabled, **the rewritten caption is part of the operation's
report** — this is a recorded contract decision, not an implementation detail. See
the caption-ownership decision in [`../internal/DECISIONS.md`](../internal/DECISIONS.md)
and the `caption_as_authored` / `caption_as_sent` columns in
[`../concepts/DATA.md`](../concepts/DATA.md).

The generalisable lesson, worth keeping when this model is replaced: **when a
generator disappoints, read the log for the prompt that actually reached it before
concluding the model is the limit.**

## Backend Selection

`backend="pt"` (PyTorch), not vLLM. vLLM is not installed and would size its
memory pool against total device memory rather than free memory, which
over-allocates against co-resident tenants — the same failure mode
[`model-registry.md`](model-registry.md) already records.

## Acquisition And Install Traps

The resource follows the `kokoro` composition pattern: a checksum-pinned
standalone Python runtime, lockfile-pinned wheels, and checksum-verified weights,
acquired natively with **no container runtime** (a recorded decision — the desktop
delivery tier cannot assume one).

Four install failures from the spike, each of which will recur:

| Trap | Symptom | Resolution |
|---|---|---|
| `flash-attn` does not declare torch as a build dependency | `ModuleNotFoundError: No module named 'torch'` during build | Exclude it from the lockfile; it is not required |
| Package-index timeouts on large CUDA wheels | `uv` fails on `nvidia-cuda-nvrtc-cu12` at its 30 s default | Raise the HTTP timeout (the spike used 300 s) |
| Weights are not in the main bundle | `5Hz LM model not found .../acestep-5Hz-lm-0.6B` | Fetch `ACE-Step/acestep-5Hz-lm-0.6B` as a separate acquisition target |
| Wrong upstream repo | Model loads but is v1, not v1.5 | Pin `ACE-Step-1.5`; see [What The Resource Wraps](#what-the-resource-wraps) |

Note also that the spike's venv was built with a raw package manager under an
explicit, deliberate waiver for throwaway work. **The resource does not inherit
that waiver.** Dependency acquisition flows through the governed path like every
other resource — see `docs/package-governance.md` at the repo root.

## Exposed Operations

The resource is a managed service holding an exclusive GPU lease while generating.
It exposes, at minimum:

| Operation | Purpose |
|---|---|
| `compose` | One take from a compiled caption, parameters, and a seed |
| `health` | Readiness, loaded variant, resident planner size |
| `variants` | Which DiT variants and planner sizes are installed |

Batching is **not** the resource's concern. A batch of takes is N seeded calls
orchestrated by this scenario's `composition` domain, because the take set, its
provenance, and its storage budget are scenario state. The resource generates one
take at a time and knows nothing about selection.

## Open Unknowns

These are remaining measurement and scope questions; they do not block the
already-created managed resource.

| Unknown | How to obtain |
|---|---|
| Full-rung peak VRAM on a quiet card | Broker measurement over a real managed run |
| SFT artifact and behaviour | **Measured on 2026-09-19:** checksum-verified artifact, ten controlled audio outputs, proxy table, and timing/capacity receipts are in the plan artifact directory. Default remains turbo because no grounded proxy or listening-quality advantage was established. |
| Scenario queue wait | Measure after the composition job/capacity domains are complete |
| `acestep-v15-sft` behaviour | **Measured on 2026-09-19.** Ten exact-brief/exact-seed jobs completed through music-tools with 50 steps, CFG 7, and the `offload-dit` rung. All ten WAVs are retained with receipts and proxy measurements. Initial attempts failed under contention, but a later governed headroom window completed the controlled run. No proxy or listening-quality number is presented as a quality verdict. |

The `sft` result is deliberately conservative. SFT carries active CFG and 50
steps, but this controlled run does not demonstrate a material proxy advantage
over the preserved turbo comparison. The default therefore stays turbo, with
SFT available as a measured alternative when an operator accepts its capacity
and timing tradeoff. A future listening study or a retained apples-to-apples
analysis implementation may revisit that decision.

## `music-mir`

Specified separately and at far lower confidence in
[`resource-music-mir.md`](resource-music-mir.md). Nothing about it has been
measured.

## Cross-References

- [`model-registry.md`](model-registry.md) — per-model licence, VRAM tier, and disk cost
- [`../internal/DECISIONS.md`](../internal/DECISIONS.md) — why ACE-Step, why no container, why the broker
- [`../internal/PERFORMANCE.md`](../internal/PERFORMANCE.md) — measurement status
- [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md) — the advisory-broker and scheduler gaps
- `resources/kokoro/` (repo root) — the working reference for a measured GPU resource
