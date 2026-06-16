# Performance — Image Tools

This document records performance budgets, current measurements, known
constraints, and regression procedures.

## Purpose Of This Document

Use this document to answer:

- What performance matters for this scenario?
- What budgets or thresholds apply?
- How are measurements captured?
- What performance risks remain?

## Budgets

Image-tools spans two very different cost classes — instant deterministic
ops and heavy AI jobs — so budgets are split accordingly.

| Surface | Budget | Measurement | Status |
|---|---|---|---|
| Cheap deterministic ops (resize/crop/convert/metadata/etc.) | Sub-second, synchronous, zero model download | per-op latency in cli/manifest.json measures | planned |
| AI op submission | Returns a job-id + ETA immediately; never blocks on the model; no client polling | job-submit latency measure | planned |
| GPU jobs | Serialized — one heavy job at a time; queue wait surfaced as ETA | queue wait time measure | planned |
| Cheap CPU ops under load | Run concurrently with (not behind) GPU jobs | concurrency / throughput measure | planned |
| Model cold-start (download + load) | Bounded and surfaced up front as ETA, not as a silent stall | cold-start latency measure | planned |
| Heavy resources (ComfyUI, per-op models) | Idle-shutdown delegated to the platform orchestrator | orchestrator idle-shutdown signal | planned |
| UI build | 5-10 minutes accepted for current Vite module graph | lifecycle/test-genie build logs | inherited |
| API / UI health | Responsive under lifecycle health timeout | `/health` check | active |

## Current Measurements

The measures domain (cli/manifest.json measure blocks, enforced by the
test-genie measures phase) will emit p50/p95 op latency per model,
throughput, queue wait time, fallback-tier usage, and VRAM headroom.

| Measurement | Value | Source | Date |
|---|---|---|---|
| None captured yet (pre-implementation). | n/a | n/a | 2026-06-16 |

## Known Constraints

- **Single-GPU contention.** One GPU serves all heavy jobs; the
  serializing queue mitigates contention but caps heavy throughput.
- **Model RAM/VRAM footprints.** Large diffusion/upscale models have
  significant memory footprints; the hardware-fit selector picks a model
  the probed host can actually run.
- **CPU-fallback slowness.** CPU-capable default models guarantee
  availability but are much slower than GPU; this is surfaced as an
  explicit time warning, not hidden.
- **Disk pressure from model weights.** Model installs consume
  substantial disk; disk-space awareness gates downloads and output writes.
- Vite production builds may process thousands of modules and take
  several minutes (inherited template constraint).

## Regression Procedure

1. Run `make test` (or `vrooli scenario test image-tools`), which exercises
   the test-genie measures and performance phases.
2. Compare the emitted measures (p50/p95 op latency per model, throughput,
   queue wait time, fallback-tier usage, VRAM headroom) against prior runs.
3. Use `git-control-tower baseline diff` to detect performance regressions
   between the working tree and the baseline run.
4. For UI interaction regressions, use `ui/perf/README.md` and the provided
   capture template.
5. Record persistent findings here (accepted constraints) or in
   [`PROBLEMS.md`](PROBLEMS.md) (unresolved debt).

## Cross-References

- [`DECISIONS.md`](DECISIONS.md) — async/queue and fallback decisions driving these budgets
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — signals and telemetry
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — release checklist
- [`TESTING.md`](TESTING.md) — coverage and test expectations
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved performance debt
