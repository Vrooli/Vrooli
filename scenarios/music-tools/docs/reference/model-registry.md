# Model Registry Reference — Music Tools

The evidence behind every model choice: identifier, disk cost, VRAM, licence, and
commercial-use lane. [`../internal/DECISIONS.md`](../internal/DECISIONS.md) records
what was decided; this document records what it was decided *from*.

## Purpose Of This Document

Use this document to answer:

- Which model serves an operation, and what does it cost to hold?
- Is a model usable in commercially distributed output?
- Which models can be resident at the same time?
- How confident is a given figure?

## Confidence Labels

Every figure below carries one. Do not promote a claim to a stronger label without
new evidence.

| Label | Meaning |
|---|---|
| `verified` | Read from a primary source — model card, repository, or official docs |
| `vendor` | Stated by the publisher, not independently reproduced |
| `estimated` | Derived by arithmetic from verified figures; not measured |
| `unknown` | Could not be determined |

**Nothing here has been measured on the reference host.** Every throughput figure is
`vendor` or `estimated`. Benchmark before treating any of it as a budget.

## Licence Lanes

- **Permissive** — usable in commercially distributed output.
- **Restricted** — non-commercial, share-alike, or undeclared licence. Personal use
  only. An undeclared licence defaults here.

A permissive-lane build must still satisfy the composition and analysis contracts.
It does, but with weaker analysis quality — this is the cost of the lane split and
it is deliberate.

## Composition

| Model | Identifier | Disk | VRAM | Licence | Lane | Confidence |
|---|---|---|---|---|---|---|
| ACE-Step 1.5 bundle | `ACE-Step/Ace-Step1.5` | 10.1 GB | tier-dependent | MIT | permissive | `verified` |
| ACE-Step DiT 2B sft | `ACE-Step/acestep-v15-sft` | 4.79 GB | see tiers | MIT | permissive | `verified` |
| ACE-Step DiT XL (4B) | `ACE-Step/acestep-v15-xl-sft` | ~20 GB (fp32 on hub) | ≥12 GB offloaded | MIT | permissive | `verified` |
| ACE-Step LM planners | `acestep-5Hz-lm-{0.6B,1.7B,4B}` | ~1.2 / ~3.4 / ~8 GB | with DiT | MIT | permissive | `estimated` |

Publisher documentation is internally inconsistent about XL's parameter count:
the README says 4B, the model index labels the repositories 5B, and the files are
20 GB at fp32. Treat the file size as authoritative for disk planning.

### VRAM tiers (`verified`, from publisher GPU-compatibility docs)

| Free VRAM | DiT | LM | Offload / quant |
|---|---|---|---|
| ≤6 GB | 2B turbo | none | CPU + DiT offload, INT8 |
| 6–8 GB | 2B turbo | 0.6B | CPU + DiT offload, INT8 |
| **8–12 GB** | **2B turbo/sft** | **0.6B** | **CPU + DiT offload, INT8** |
| 12–16 GB | 2B sft | 1.7B | CPU offload only, INT8 |
| 16–20 GB | XL (marginal) | up to 4B | offload + quant |
| ≥20 GB | XL turbo/sft | 1.7B+ | none |

The reference host's ~9.9 GB free places it in the **8–12 GB tier**. XL is out of
reach; the 1.7B planner is out of reach without evicting a co-resident tenant.

Two failure modes to design against:

- **Offload targets the scarcest resource.** The tier that fits also pushes weights
  into system RAM, which on the reference host is already under swap pressure. The
  likely failure is thrashing, not an out-of-memory error.
- **Aggressive inference backends size their allocator against *total* device
  memory, not free memory.** With co-resident tenants this over-allocates. Pin the
  utilisation fraction explicitly or use the plain backend.

### Operation availability

Composition, cover, and section repaint are available on all variants. Track
separation, layer addition, and continuation are documented as **base-model only**
in the inference reference while the README's feature matrix marks them as
universal. The variant that fits the reference host is the one most likely to lack
them. This contradiction is unresolved upstream and is why separation is owned by
`music-mir` instead. See [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md).

## Analysis and embedding

| Model | Identifier | Disk | VRAM | Licence | Lane | Confidence |
|---|---|---|---|---|---|---|
| CLAP music+speech | `laion/larger_clap_music_and_speech` | 780 MB | ~1.5 GB | Apache-2.0 | **permissive** | `verified` / VRAM `estimated` |
| MuQ-large | `OpenMuQ/MuQ-large-msd-iter` | 1.33 GB | ~2.5 GB | CC-BY-NC-4.0 | restricted | `verified` / VRAM `estimated` |
| MuQ-MuLan-large | `OpenMuQ/MuQ-MuLan-large` | ~2.65 GB | ~3.5 GB | CC-BY-NC-4.0 | restricted | `verified` / VRAM `estimated` |
| MERT-95M | `m-a-p/MERT-v1-95M` | 378 MB weights | ~1 GB | CC-BY-NC-4.0 | restricted | `verified` / VRAM `estimated` |

MuQ outperforms MERT and MusicFM across nearly all tasks on the standard music
representation benchmark, on far less pretraining data (`vendor` — publisher's own
paper). MuQ-MuLan additionally provides zero-shot tagging *and* an audio-similarity
metric, which is what makes it valuable to a consumer building a taste model.

Two hard operating constraints (`verified`): MuQ requires **24 kHz** input, and must
run at **fp32** to avoid numerical failure — so there is no half-precision speedup
available for it.

CLAP is the only permissive-lane embedding model. A permissive build runs on CLAP
alone and loses the music-specific representation quality.

**Download selectively.** Several of these repositories ship duplicate weight
formats — a naive full-repository fetch roughly doubles their footprint.

## Supporting tools

| Tool | Identifier | Disk | VRAM | Licence | Lane | Confidence |
|---|---|---|---|---|---|---|
| Demucs (htdemucs) | `adefossez/demucs` | ~330 MB | ~7 GB default, ~3–4 GB segmented | MIT | permissive | `verified` |
| Demucs fine-tuned | `htdemucs_ft` | 4× | as above | MIT | permissive | `verified` |
| Band-split / Mel RoFormer | via `python-audio-separator` | ~640 MB each | ~4–6 GB | MIT wrapper | permissive | `estimated` |
| Structure + beats | `mir-aidj/all-in-one` | ~1.5 GB | ~7–8 GB | **undeclared** | **restricted** | `verified` / VRAM `estimated` |
| Audio to MIDI | `spotify/basic-pitch` | ~20 MB | CPU | Apache-2.0 | permissive | `verified` |
| Loudness | `csteinmetz1/pyloudnorm` | — | CPU | MIT | permissive | `verified` |
| Reference mastering | `sergree/matchering` | — | CPU | **GPL-3.0** | permissive, **process-isolated** | `verified` |
| Supervised tag heads | Essentia / MTG models | ~200 MB | CPU | CC-BY-NC-**SA**-4.0 | restricted | `verified` |

The structure-and-beats tool is the best available for its job and **declares no
licence**, so it sits in the restricted lane by the default-to-restricted rule. It
is also the hardest install in the stack: it pins an attention extension against an
exact torch build and pulls a dependency with chronic NumPy-major conflicts. This
is the direct reason `music-mir` exists as a separate runtime.

Reference mastering is GPL-3.0 and is therefore invoked as a subprocess and never
linked into the scenario binary.

## Residency

| Set | Peak VRAM | Co-resident? |
|---|---|---|
| Embedding pool (CLAP + MuQ + MuQ-MuLan) | ~8–9 GB `estimated` | Yes — the always-on tier, with capped chunk and batch size |
| Embedding pool, permissive lane only (CLAP) | ~1.5 GB `estimated` | Yes, comfortably |
| Stem separation | ~7 GB `verified` default | **No — exclusive** |
| Structure and beats | ~7–8 GB `estimated` | **No — exclusive** |
| Composition | ~9–10 GB `estimated` | **No — exclusive** |
| Adapter training | 16–17 GB `verified` | **No — exceeds the reference card entirely** |

**No two heavyweight models co-reside.** Each takes an exclusive lease through the
capacity broker; the embedding pool is the only persistent resident set.

## Registry disk budget

| Tier | Contents | Disk |
|---|---|---|
| Core | Composition bundle, embeddings, separation, structure, notation, loudness | ~28 GB `estimated` |
| + quality composition | Adds the 2B sft variant | ~33 GB `estimated` |
| Permissive lane only | Composition + CLAP + separation + notation + loudness | ~13 GB `estimated` |

Installs are checksum-verified and refuse to start when free disk is below the
declared floor. The reference host has 274 GB free on a volume at 85%.

## Rejected

| Model | Reason |
|---|---|
| MiniMax-Music3 | 57.4 GB and 22–24 GB VRAM; only its explicitly-slow streaming mode fits, and that needs system RAM the host lacks. Non-OSI licence with a revenue ceiling and mandatory UI attribution. |
| ACE-Step XL | ~20 GB on disk, 16–20 GB VRAM even with offload and quantisation. |
| Adapter training on the reference host | 16 GB minimum and ~17 GB typical against a 16 GB card. |
| HeartCLAP | **Never released.** A public request has stood unanswered since 2026-01-17. |
| HeartMuLa-oss-3B / HeartCodec | 15.8 GB and 6.64 GB for a strictly worse VRAM-to-quality tradeoff here. |
| MusicFM | Superseded by MuQ on the same benchmark, same licence class. |
| MERT-330M | 1.26 GB of weights for a representation MuQ already beats at smaller size. |
| Structure model with published SOTA scores | No code or weights released. |
| Multi-instrument transcription research models | Separate toolchain island for marginal gain over per-stem notation. |
| Generic beat/key from a feature library | A feature library, not a competitive estimator. Kept for I/O and chroma only. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — the three-runtime split
- [`../internal/DECISIONS.md`](../internal/DECISIONS.md) — what was decided from this
- [`../internal/PERFORMANCE.md`](../internal/PERFORMANCE.md) — cost of running it at library scale
- [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md) — unresolved upstream contradictions
