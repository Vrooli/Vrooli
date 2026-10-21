# Performance — Device Studio

## Purpose Of This Document

This document records the performance contract and dated measurements. The app
implements bounded photographic still/video workflows; physical rendering is
being integrated. The observations below have different scopes and do not imply
release acceptance. The agreed current-host gates at the end supersede earlier
notes that numerical targets were pending.

Record measurable output and resource budgets without converting hardware inventory into latency promises.

## Latest gate standing — 2026-10-20

| Gate | Latest qualified observation | Standing |
|---|---|---|
| Controls ≤200ms | Four actual mobile editor tab clicks24.0–29.9ms through two animation frames | Navigation case passes; other editor controls open |
| Warm preview ≤10s | Warm browser case10.5092s; later fresh-worker-cache browser case20.2384s; 960×540/64samples | Open; timing gate fails |
| Still ≥2560 long edge ≤60s | Public native phone stills20.305742s and30.769544s | Exact phone cases pass |
| Ten-second1080p3D ≤20min | Public static-screen phone, landscape30fps,1036.644515s | Exact case passes; full matrix open |

The dated measurements below preserve the original chronology, including earlier
“pending” observations. This table and the agreed gate definitions own current
standing. Recording-texture native proof, portrait/rate matrix and other devices
remain separate. Verified immutable-input reuse has passed source tests and is in the measured rebuilt workflow. This observation does not isolate its causal speedup.

The exact-input, test-only coordinator timing harness measured cold Submit3.746s
and warm Submit2.424s, with Blender profile resolution3.060s/1.976s. It used the
retained1206×2622 approved-study screen and the explicit78,440,306-byte API
executable in temporary routed storage with no native dispatch. It does not
measure the HTTP handler, browser, pump, or preview gate. Receipt:
`goal/evidence/E3/physical-preview/timing-harness/coordinator-submit.json`.

The public laptop pilot exported48frames at1920×1080/24fps in157.7517s through
verified download. Full decode/PTS passes, but appearance is rejected because
the lid/screen is absent and the base is cropped. This is a failed product
sample, not a qualified laptop throughput or ten-second performance result.

## Budgets

Initial video profile: 1920×1080 or 1080×1920, 24/30 fps, SDR. Stills use sufficient source/native detail. Each job declares maximum duration/frame count, CPU/memory/tasks/disk/VRAM admission, cache limit and any paid ceiling. Numeric default machine budgets are implementation configuration to calibrate on the qualified worker, not universal source facts. A request outside capacity returns a concrete limitation before dispatch.

UI preview uses cancellable low-cost proxies and staged quality; final renders are asynchronous. Measure interaction latency, preview seek readiness and queue responsiveness separately from final-frame throughput. Do not block the event loop waiting for native jobs. Thumbnail/catalog pagination and virtualized large libraries remain scoped by actual data size.

## Current Measurements

Historical constrained CPU observations are retained in the research: v5 high-quality 1920-wide scene stills took roughly 465–555 seconds; v6 phone composition of 96 frames took roughly 192 seconds excluding generation. They used different settings and resource limits and are not controlled comparisons. The current RTX 4070 Ti SUPER is detected; installed Blender 4.0.2 lacks CUDA kernel/compiler payload, so no GPU render benchmark is qualified. These numbers must not become a product SLA.

The 2026-10-20 integrated native still journey recorded approximately 3.30 seconds from creation to terminal update for the 1536×1024 phone job, 2.12 seconds for a distinct replay, and 1.93 seconds for the 1672×941 browser job. These timestamps include orchestration and are not isolated renderer throughput measurements. A separate 4000×4000 synthetic full-active fixture was cancelled while its exact worker PID was live; the observed cancel call took about 89 milliseconds, followed by native quiescence and release. It does not establish a cancellation SLA or photographic quality at 16 MP. Raw evidence is retained under [E2 public jobs](goal/evidence/E2/public-jobs/), including `integrated-cancel-fullactive.json`. Release-resolution video throughput and numerical performance acceptance remain unmeasured.

The 2026-10-20 public fixed-video journey measured 3.30 seconds from sample submission to attempt finish, 76.78 seconds for a five-second 1920×1080/24 fps final, and 75.84 seconds for its post-restart rerender. All 120 frames decoded with verified timestamps; both final files were byte-identical. These are single-host observations using an authentic phone recording and synthetic calibration scene, including orchestration. They do not qualify photographic craft, physical motion, or a performance target. See [observed timings](goal/evidence/E2V/p5-live/run-6-output-verification/observed-timings.json). Numerical performance acceptance remains pending before library expansion.

## Known Constraints

GLB inspection can cap mesh/texture sizes and replace lighting. Use separate native and preview assets when needed; do not reduce final detail invisibly. Texture physical scale differs from resolution. High-resolution screenshots cannot recover blurred/inadequate source plate detail. Path-tracer noise, motion aliasing and denoising can damage small UI text even when aggregate frame throughput improves.

## Regression Procedure

Pin source/tool/profile/device/driver, resolution, samples and scene revision. Run one approved-look frame, a short continuous motion sample, and full decode. Measure wall time, peak resident/VRAM/disk use, cancellation latency and cleanup. Compare like-for-like profiles. Record denied/unavailable tests honestly. Broaden testing only for changed settings or unresolved findings; never rerender unchanged work merely to collect a greener number.

## Cross-References

[rendering](../domains/rendering.md), [observability](../operations/OBSERVABILITY.md), [TESTING.md](TESTING.md), [research](../research/README.md).


The first bounded photographic screen-entry profile test rendered 120 frames at 1920×1080,
24 fps with a 48-frame transition. Public submission-to-attempt-finish was 101.823 seconds
for the final and 6.077 seconds for its sample on this host. This used the synthetic
calibration plate and authentic phone recording. It is a single observed run, not a
performance target or evidence that the 120-frame worst case is qualified. Receipt:
`goal/evidence/E2V/p7-camera/public/observed-timings.json`.


P8 observed final-attempt durations on this host (two-second synthetic calibration
clips; each one observation, not an SLA): landscape phone30fixed36.870s/camera48.473s;
portrait phone24fixed72.347s/camera83.835s; desktop browser30fixed66.721s/camera102.719s.
Portrait plate input2160×3840, output1080×1920. Browser trim starts sourceordinal30.
Exact output/tool/worker identities: goal/evidence/E2V/p8-matrix/public/matrix-results.json.
These timings do not establish photographic or3D quality, arbitrary clip lengths,
GPU performance or a user-approved performance threshold. The browser corner-remnant
correction subsequently passed a fresh public export, pixel checks and Chrome 150 playback;
see `goal/evidence/E2V/p8-matrix/clean-viewport/` and `browser-playback/`.

The P9 audible-marker final recorded 36.36 seconds from creation to terminal update
for a two-second 1920×1080/30 fps synthetic output. The original PCM audio is retained
in source storage; the H.264 output has no audio stream. This single observation
includes orchestration and is not a throughput target. Managed official Blender
5.2.2 CPU/CUDA/OptiX diagnostic renders passed. The first detailed960×540/64-sample phone study took2.126s inside the OptiX render and12.67s for the complete evidence harness. Its lighting was rejected. A subsequent attempt failed GPU memory allocation with only about701MiB free after the failure; existing local services held most GPU memory. These observations do not establish sustained throughput or a performance target. CPU scene measurement and corrected higher-resolution look remain pending.


The corrected phone lighting study on explicit CPU took39.816s for960×540,64
Cycles samples with two threads (52.09s complete evidence harness). Systemd's
retained native log reports491.0M RAM peak and6.6M swap peak. The output passed
full PNG validation and parent lighting/screen review; composition/contact work
remains. This is a different lighting revision from the earlier OptiX study, so
it is not a controlled CPU/GPU speed ratio. GPU availability changed again to
about8680MiB free; no other service was stopped or reconfigured.


The revised contact-balanced phone scene passed a native2560×1440/256-sample
OptiX still in10.411s render time and20.36s complete evidence harness. Its960×540/64
sample study took1.409s render/8.64s harness. Three960×540/96-sample yaw views
completed in12.98s harness. Exact output/script/asset identities and individual
render receipts are under `goal/evidence/E3/scene-studies/phone/runs/`. These are
fixed-screen evidence studies, not public app exports or continuous-motion proof.
No denoising or adaptive sampling was used. Operator appearance review approved the exact hero;
performance targets remain pending representative motion measurement.


The first continuous physical-phone study rendered48 frames at1920×1080,24fps,128 Cycles samples through OptiX in170.336s inside the render loop; the complete render/encode/decode evidence test took204.26s. The exact two-second H.264 file is1,117,677bytes, SHA256 `155968ad55d01548d48c3218ce0f98d67da36d781149284f737e529b2e7b1739`. Chrome150.0.7871.128 decoded all48 callbacks and reached ended at2s. Evidence: `goal/evidence/E3/scene-studies/phone/runs/motion-1/optix-motion-yaw-24/`. The app screenshot remains stationary while the device rotates; this is not public app integration or a10-second performance measurement. Other local work was active, and initial GPU availability delayed dispatch; no sustained-throughput or performance target is claimed.


## Agreed current-host release gates — 2026-10-20

Operator selected “Use these quality-first targets (recommended)” after the measured two-second physical experiment. Validate on this host with the approved render quality; report workload contention rather than quietly changing samples or substituting a lower-quality profile.

| Interaction/output | Agreed upper bound | Measurement boundary |
|---|---:|---|
| Editing controls | 200 ms | User action to visible control/preview-state response; native final rendering remains asynchronous |
| Warm preview | 10 s | Preview request to usable preview with required tools/assets already available |
| High-resolution still | 60 s | Submission to verified downloadable PNG, long edge at least2560pixels |
| Ten-second1080p3D export | 20 min | Submission to verified downloadable MP4; final approved profile,24/30fps and selected orientation |

These are acceptance gates, not current passing claims. The48-frame study is a measurement input, not a substitute for the10-second test. Cold tool acquisition is separate and must show progress. Verify the integrated public workflow; isolated renderer timings alone do not establish these end-to-end bounds.

The fresh physical still api2 failed GPU allocation before a final output with700MiB free of16376MiB at post-failure observation. This is retained resource-contention evidence, not a successful render timing or a failing throughput measurement. The managed Blender installation remained exactly pinned after the repaired launch. That attempt establishes no performance pass; see `goal/evidence/E3/public-physical-still/runs/api2/`.

After GPU memory became available naturally, public still api3 passed in20.305742s from submission to verified downloadable2560×1440 PNG (256 Cycles samples, explicit OptiX). Output SHA256 `51e40befd6f0dd7eacd5b1024504b2f9c127156b766f11e002670a02528a58d7`; exact save/reopen and idempotent submit replay also passed, primary project list unchanged. The worker rechecked the pinned Blender tree before completion; a separate post-profile reports PRESENT. Parent visually inspected the exact export against the accepted phone hero. This meets the≤60s gate for this candidate/scene/profile on this host in this observation, not every device or sustained throughput. Controls, warm preview and ten-second physical export gates remain unmeasured. Evidence: `goal/evidence/E3/public-physical-still/runs/api3/`.


A fresh typed-placement preset export, api4, passed in30.769544s through verified
download at2560×1440,256samples,OptiX. SHA256
`3071852e5f1801e3c0f76203008dff09859997def9ce902be6cdb34e774fe4dc`.
The prior public48-frame static-screen physical motion export passed in172.865458s,
including full decode and48 Chrome150 video-frame callbacks; it does not substitute
for the ten-second gate. Public ten-second/30fps attempt api2long30 failed before
job creation because closure serialization used the generic64KiB cap rather than
the existing5MiB physical closure cap. The bounded fix keeps the renderer manifest
cap4MiB, preserves legacy profiles and has240/300-frame plus overflow regressions.
Attempt api3long30 is running against that rebuilt owner; its timing remains pending.
Evidence: `goal/evidence/E3/physical-motion-public/runs/` and
`goal/evidence/E3/public-physical-still/runs/api4/`.


The ten-second static-screen physical export api3long30 passed in1036.644515s
(17m16.645s) through verified download:1920×1080,300frames,30fps,128Cycles
samples,OptiX,all300frames independently decoded withzero-basedPTS andBT.709
limited-range tags. MP4SHA256 `6873b1ffd3bb52a347d61f5c3f4074203a07b438af798b03301e5c41310f7dbe`.
The owned operation released; ordered300pre-encode frame digests are retained.
Chrome playback reached ended at10s without error,299video-frame callbacks
(the initially presentedframe0 preceded callback registration; finalpresented_frames300).
Parent reviewed sampledposes at0/2.5/5/7.5/9.967s against the approved scene.
This meets the20min gate for this exactstatic-screen landscape30fps configuration
onthishost; recordingtextures,portrait24/30 andotherdevices remain separate.
Evidence: `goal/evidence/E3/physical-motion-public/runs/api3long30/`.


A third fresh preview, api3warm, ran after GPU availability changed naturally to
9642 MiB free. Submit-to-verified-download was 10.013559 seconds, including
2.756781 seconds in submission; native attempt duration was 6.869347 seconds.
The job succeeded and execution released, but the ten-second gate still fails.
No rounding, cached-job substitution or profile/sample reduction is counted as a
pass. Output SHA-256 is
`554ae4d1db2debf6f1688440803884645c9acc175895be8503cd4cd3afc66f27`.
Evidence: `goal/evidence/E3/physical-preview/runs/api3warm/`.


One mobile editor Advanced-step interaction passed the control target: 23.6 ms
from the click handler to selected-state observation after two animation frames.
The parent read the actual BAS timeline result; 390 px document/viewport widths
match and every step target is at least 44 px high. This is one control and the
current revision, not a complete editor gate or native preview timing. Tool
dispatch observations use a different boundary. Evidence:
`goal/evidence/E3/physical-ui/live-current/mobile-control-parent-verification.json`.


The rebuilt fresh-shot browser proof measured **10.5092s** from the real preview-button click through verified960×540 PNG decode and two animation frames (before optional blob SHA reread). Job `e0a80d6f-71cc-4941-80eb-3ddcf3cf7cf6` and BAS execution `93cff8de-1437-4900-b2dc-d3adb66277d5` agree on outputSHA `0aced92627f83f155383498ffbefe12cd369906c35b8d51b744c1d138fe16bc2` (847458bytes); execution released. The native attempt took6.859716s. This fails the10s gate. Preserve the preceding invalid observations: a stale heading caused a120s observer wait; a reused shot then matched an older preview at142.4ms. Neither is a performance pass. The final flow explicitly waits for empty render history and no previous output before clicking. See `goal/evidence/E3/physical-preview/bas-prepared/live-run-3/parent-verification.json`.


### Active-progress follow-up — 2026-10-20

The existing job owner now wakes after submit/cancel and follows active progress
at250ms while returning to its750ms idle interval. Focused race regressions and
an independent source review pass. The fresh routed browser fixture in BAS
`28351582-96f5-4e06-8bf1-b05ba0367ec6` took20.2384s from actual click through
verified960×540 image decode and two animation frames. It fails the10s gate.
This new binary/lease had a first-use immutable worker input cache; do not claim
an isolated scheduling regression or speedup from comparison with the older
10.5092s case. The sole attempt was terminal and released before lease cleanup.
Exact output and timing: `goal/evidence/E3/physical-preview/bas-prepared/live-run-6/parent-verification.json`.

The same saved shot at390×844 passed all four editor sections and exact decoded
output/download checks. Real navigation clicks became visibly selected after
two animation frames in24.0,28.7,28.0 and29.9ms, below200ms. This covers mobile
navigation only; all other editing controls and desktop interaction timing
remain open. BAS execution `1376b6dd-9eab-4f0a-a237-58bb9de605bc`; retained
review: `goal/evidence/E3/physical-preview/bas-prepared/live-run-7/parent-verification.json`.

## Instrumented preview — 2026-10-21

A fresh UI submit on worker `5cb45fe7…fe38` reached a verified 960×540 PNG
and two animation frames in **19.1875 seconds**, failing the 10-second target.
Same-origin ResourceTiming exposed Server-Timing: handler 11.1083s, worker pin
2.5971s, Blender profile 8.2566s, shot resolution 0.2088s, and closure 0.0446s.
The job's created-to-updated interval was 7.2069s; that includes owner execution
and finalization, rather than measuring GPU work alone. The browser observed one
submit, 30 list responses, and output SHA `f43bd6b2…37d58`. Evidence:
`goal/evidence/E3/physical-preview/runs/uiTiming1/parent-review.json`.

### Installed owner experiment and fresh worker case

After the supported control-plane `make install` (CLI `13c58415…c0acd`), a
preview with unchanged resolution and 64 samples took **10.6595 seconds** on a
fresh routed worker object. It still fails the 10-second target. The handler
used 2.0365s, including worker pin 1.3833s and Blender profile 0.4245s; the job
interval was 8.3040s. The shared dirty-tree CLI rebuild and different cache
states prevent attributing the whole reduction to parallel hashing. Evidence:
`goal/evidence/E3/physical-preview/runs/uiTiming2/parent-review.json`.

### Explicit warm worker case — PASS

A new project and shot, with fully loaded empty render history, reached the
verified 960×540 image and two animation frames in **7.4909 seconds**. This
passes the accepted warm preview target. The same lease already held a verified
content-addressed worker object. Exactly one submit was observed, and no prior
output was shown or reused. The handler used 1.4234s, including worker pin
0.2666s and Blender profile 0.9581s. Resolution and sample count were unchanged.

This is one bounded current-host observation, not a percentile guarantee. The
fresh-worker case and historical failures remain recorded. Evidence:
`goal/evidence/E3/physical-preview/runs/uiTiming3Warm/parent-review.json`.
