# Performance — Device Studio

## Purpose Of This Document

Foundation contract, 2026-10-19. Product behavior below is specified for implementation; the generated reference scaffold does not implement it. Research demonstrates individual techniques, not a released scenario.

Record measurable output and resource budgets without converting hardware inventory into latency promises.

## Budgets

Initial video profile: 1920×1080 or 1080×1920, 24/30 fps, SDR. Stills use sufficient source/native detail. Each job declares maximum duration/frame count, CPU/memory/tasks/disk/VRAM admission, cache limit and any paid ceiling. Numeric default machine budgets are implementation configuration to calibrate on the qualified worker, not universal source facts. A request outside capacity returns a concrete limitation before dispatch.

UI preview uses cancellable low-cost proxies and staged quality; final renders are asynchronous. Measure interaction latency, preview seek readiness and queue responsiveness separately from final-frame throughput. Do not block the event loop waiting for native jobs. Thumbnail/catalog pagination and virtualized large libraries remain scoped by actual data size.

## Current Measurements

Historical constrained CPU observations are retained in the research: v5 high-quality 1920-wide scene stills took roughly 465–555 seconds; v6 phone composition of 96 frames took roughly 192 seconds excluding generation. They used different settings and resource limits and are not controlled comparisons. The current RTX 4070 Ti SUPER is detected; installed Blender 4.0.2 lacks CUDA kernel/compiler payload, so no GPU render benchmark is qualified. These numbers must not become a product SLA.

## Known Constraints

GLB inspection can cap mesh/texture sizes and replace lighting. Use separate native and preview assets when needed; do not reduce final detail invisibly. Texture physical scale differs from resolution. High-resolution screenshots cannot recover blurred/inadequate source plate detail. Path-tracer noise, motion aliasing and denoising can damage small UI text even when aggregate frame throughput improves.

## Regression Procedure

Pin source/tool/profile/device/driver, resolution, samples and scene revision. Run one approved-look frame, a short continuous motion sample, and full decode. Measure wall time, peak resident/VRAM/disk use, cancellation latency and cleanup. Compare like-for-like profiles. Record denied/unavailable tests honestly. Broaden testing only for changed settings or unresolved findings; never rerender unchanged work merely to collect a greener number.

## Cross-References

[rendering](../domains/rendering.md), [observability](../operations/OBSERVABILITY.md), [TESTING.md](TESTING.md), [research](../research/README.md).
