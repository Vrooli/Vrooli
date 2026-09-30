# Performance — Music Tools

Budgets, what has actually been measured, and the constraints that bound both.

## Purpose Of This Document

Use this document to answer:

- What is an acceptable duration for each operation class?
- What does analysing a real library actually cost?
- Which numbers are measured, and which are guesses?

## Honesty statement

Composition now has both spike and managed-service measurements on the reference
host. Analysis, separation, and queue wait remain unmeasured. SFT has now been
measured through ten successful checksum-verified managed-service jobs; the
numeric audio results remain proxies and do not constitute a listening-quality
verdict. Every number
without a receipt below is still `vendor` or `estimated`.

Treat every budget below as a hypothesis with a falsification procedure.

## Budgets

Reference host: 16 GB VRAM card with roughly 9.9 GB free, 32 cores, 60 GB RAM under
existing swap pressure, 274 GB free disk.

| Operation class | Budget | Basis |
|---|---|---|
| Deterministic ops (trim, convert, loudness) | < 1 s per track, no GPU | CPU-only, exact arithmetic |
| Embedding a track | seconds | `estimated` |
| Structure and beats | tens of seconds | `estimated` from vendor throughput on faster hardware |
| Stem separation | tens of seconds | `estimated` from third-party measurement on comparable hardware |
| Composition, short clip | seconds to a minute | `vendor` |
| Composition, full track | under a few minutes | `vendor` |
| Queue wait under contention | Reported, never hidden | Policy, not measurement |

The last row matters more than the others. Because the GPU is shared and heavyweight
models take exclusive leases, **queue wait is often larger than compute time**. An
operation that reports its wait honestly is behaving correctly even when slow.

## Current Measurements

Composition — and only composition — has been measured, now including the SFT
variant. Everything else in the budget table above is still `vendor` or
`estimated`, and the distinction is the point: do not read this section as
though the scenario has been profiled.

Measured 2026-09-18 on the reference host (RTX 4070 Ti SUPER, 15.56 GiB usable,
**five model services already resident**), ACE-Step 1.5 turbo, 8 steps, 45 s of
48 kHz stereo output. Full figures and corrections in
[`../reference/model-registry.md`](../reference/model-registry.md).

| Measurement | Value | Confidence |
|---|---|---|
| Composition, 45 s clip, DiT resident | 17.9 s | `measured` (spike) |
| Composition, 45 s clip, `offload_dit_to_cpu` | 55.2 s | `measured` (spike) |
| Managed offload smoke, 1 s request / 1 step | ~46 s wall clock | `measured` (2026-09-19) |
| Managed offload smoke output | 5.120 s, 983084-byte WAV | `measured` (2026-09-19) |
| Degradation cost of the CPU-offload rung | **3.1×** | `measured` |
| Cold model load | ~30 s | `measured` |
| Peak process VRAM | ~7.0 GiB | `measured` |
| SFT, 5.12 s output, 50 steps / CFG 7, offload rung | 16.613–570.561 s; median 22.101 s | `measured` (10 jobs; one capacity outlier) |
| SFT offload control-plane footprint | 6.0 GiB; one footprint sample | `measured` reservation/footprint, not instantaneous per-job peak |

### The first concrete degradation rung

`OT-P0-003` requires operations to declare ordered profile rungs and to degrade
under contention rather than fail. That requirement now has one real instance
with a real price: **CPU-offloaded DiT costs 3.1× wall-clock and buys the
ability to run in roughly 4 GiB instead of ~7 GiB.**

### SFT measurement

The SFT comparison used the ten preserved spike briefs and requested seeds,
`duration=5`, 50 inference steps, CFG 7, and the same caller-controlled caption
path. All ten jobs produced 5.120-second, 48 kHz stereo WAVs. The proxy table is
at `~/.vrooli/plan-artifacts/music-tools-implement-and-validate-the-ace-step-resource/sft-run-20260919/proxy-metrics.json`;
it records tempo, 30–120 Hz sub-bass share, 6–14 kHz hi-hat energy, onset
density, and RMS. These are evidence about reproducible signal features, not a
quality verdict. The measured run did not establish a material SFT advantage,
so turbo remains the default; SFT is the higher-cost measured alternative.

This is the first rung to declare, and it is worth declaring precisely because
the price is high enough that a caller should be told it was applied. A rung
that silently triples a job's duration is indistinguishable from a hang.

### What the measurement does not establish

The spike ran outside the scenario, outside the capacity broker, and outside
any job queue. The managed smoke now establishes that the acquired resource,
offline model closure, capacity rung, and HTTP boundary work together. It still
establishes nothing about queue wait, lease scheduling, concurrent operations,
or the analysis and separation tiers.

## Known Constraints

### Nothing heavyweight co-resides

The embedding pool is the only persistently resident model set. Composition,
separation, and structure analysis each take an exclusive GPU lease. Throughput for
mixed workloads is therefore governed by lease scheduling, not by model speed.

### Offload targets the scarcest resource

The composition tier that fits the free-VRAM budget also pushes weights into system
RAM — which on this host is already under swap pressure. The expected failure is
**thrashing, not an out-of-memory error**, and it will present as wildly variable
wall-clock rather than a clean failure.

### An aggressive allocator will fight resident tenants

Inference backends commonly size their memory pool against *total* device memory
rather than free memory. With co-resident tenants this over-allocates. The
utilisation fraction must be pinned explicitly.

### One embedding model cannot use half precision

The strongest music-representation model must run at fp32 to avoid numerical
failure, and requires a fixed input sample rate. There is no half-precision speedup
available for it — a permanent constraint, not a tuning opportunity.

### Library-scale cost, by layer

For a 10,000-track library, all `estimated`:

| Layer | Batch-viable? | Notes |
|---|---|---|
| Loudness | **Yes** — minutes on 32 cores | CPU, exact. Do this first |
| Embeddings | **Yes** — hours, one overnight pass | The core substrate |
| CPU tag heads | **Yes** — runs concurrently with GPU work | Different resource entirely |
| Structure and beats | Marginal — one to a few days | One-time; benchmark before committing |
| Stem separation | **No** — days, and ~1 TB output | On-demand only, by construction |
| Audio to MIDI | **No** — gated entirely by separation | Lazy |

The ordering matters: cheap exact layers first, embeddings next, structure as a
deliberate one-time job, separation never in batch.

### The disk constraint binds before the compute constraint

Separated stems for a 10,000-track library approach a terabyte against 274 GB free.
This is why there is no library-wide separation path. See
[`../concepts/DATA.md`](../concepts/DATA.md).

## Regression Procedure

1. **Establish a baseline before optimising anything.** Run the composition model's
   own profiling entrypoint on the reference host with the current tenant set
   resident — not on an idle card, which is not the operating condition.
2. **Benchmark structure analysis on 20 real tracks.** Published throughput for this
   tool is inconsistent with independently measured throughput of its own internal
   separation step, so the vendor figure is not trustworthy. Measure it.
3. **Record the applied profile rung with every sample.** A fast result at a degraded
   rung is not a faster system.
4. **Measure queue wait separately from compute.** They have different causes and
   different fixes.
5. **Re-measure after any change to the resident tenant set.** Free VRAM is the
   independent variable that moves most.
6. Replace the estimates above with measured values and relabel them `measured`.

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — residency policy
- [`../concepts/DATA.md`](../concepts/DATA.md) — the size arithmetic
- [`../reference/model-registry.md`](../reference/model-registry.md) — VRAM tiers and confidence labels
- [`PROBLEMS.md`](PROBLEMS.md) — why nothing is measured yet
