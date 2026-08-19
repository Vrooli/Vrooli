# Performance — Music Tools

Budgets, what has actually been measured, and the constraints that bound both.

## Purpose Of This Document

Use this document to answer:

- What is an acceptable duration for each operation class?
- What does analysing a real library actually cost?
- Which numbers are measured, and which are guesses?

## Honesty statement

**No figure in this document has been measured on target hardware.** Nothing in
this scenario executes a model yet. Every number is `vendor` (stated by a model
publisher) or `estimated` (arithmetic from verified figures). They are written down
so they can be *replaced*, not relied upon.

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

| Measurement | Value | Confidence |
|---|---|---|
| — | none taken | — |

This table stays empty until the profiling procedure below has run. Populating it
with vendor figures would be dishonest.

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
