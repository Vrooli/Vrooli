# World performance

The hardware gate is the integrated-GPU (`igpu`) tier. The discrete-GPU tier is
a portability cross-check. Smoke runs reject SwiftShader when `--gpu` is used
and record renderer, GPU tier, device scale factor, actor count, timestamp and
timing method beside every result.

## Profiles and pixel axes

Low, medium and high render at device scale factor 1; ultra renders at 1.5.
MSAA is 0, 2, 4 and 4 respectively. Low omits shadows, AO, bloom, weather
particles and water. Medium enables bounded shadow refresh and bloom. High adds
low-quality AO. Ultra spends its extra pixel budget on DPR while leaving AO off
and retaining four-sample MSAA.

Budgets are calibrated from a hardware observation plus 15 percent headroom.
Every row records whether it is a target or an observed ceiling. The target
ultra rows use 16.7 ms; the 2026-09-04 pre-change observations were 47.96 ms for
park and 45.78 ms for office at DPR 1.5.

## Attribution

GPU timestamp queries attribute four spans: shadow-map rendering, main scene
rendering, post-processing, and total frame GPU time. Main is total after
subtracting shadow and post. Direct scene traversal separately attributes draw
calls and triangles to named top-level groups. The smoke gate rejects more than
10 percent unattributed draw calls.

Vegetation uses one instanced draw per prop/material, shadow refresh is explicit
and bounded, and demand rendering avoids frames when the simulation, camera and
weather have no motion. Diagnostic-only idle sessions render a four-frame-per-
second heartbeat; `capture=1` keeps validation sessions continuous.

## Calibration

Run `pnpm world:smoke --gpu --gpu-tier igpu --actors 25`, retain the generated
JSON and PNG records, then run the calibration script with the chosen headroom.
Do not publish a hardware budget from SwiftShader, an unlabelled fallback, or a
renderer whose provenance does not match the budget row.
