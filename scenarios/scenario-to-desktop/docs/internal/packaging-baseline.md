# Packaged launch baseline — E1

The four selected references each have comparable ten-run cold and warm groups
with the required measurements, and the historical startup waterfall is
attributed below. Single-run probes and rejected repeats remain diagnostic and
are excluded from baseline medians.

## Current-source Linux artifacts

The four selected references were built through the scenario-to-desktop pipeline with current source. The selected set is hello-desktop, system-monitor, web-console, and LPBS (the D2-approved substitute for GCT). The system-monitor build is its first current-source build, replacing secrets-manager per DPF-FB-014. Its own host-metrics polling does real work during `idle_cpu_pct` windows; that contribution belongs in the measurement and must not be interpreted as idle cost from the packaged app alone.

| Reference | Pipeline run | Artifact | Bytes | SHA-256 |
|---|---|---|---:|---|
| hello-desktop | `0f9f7ac6-cc5e-e11b-6558-b69a0337bf87` | `Hello Desktop-0.1.0.AppImage` | 138,345,260 | `b0a9c7be052fd4d1890c994e306d3794dcd1a20869d16d95b69d12a64dccfb89` |
| system-monitor | `8178f443-c638-c9f5-9fd2-cbecaba0a17c` | `Vega-1.0.0.AppImage` | 156,681,719 | `dedb18edfa50e883fdf56f8f0acdf9dcceed43f7992759d76569caced12b755d` |
| web-console | `9be28246-c89b-59c9-9078-8ce1ef17f568` | `Aquila-0.0.1.AppImage` | 159,395,184 | `47ba5b74761c4abf2cc0506ca0c4c397b18b23eef454edfa144b760ebc7c232d` |
| landing-page-business-suite (GCT substitute) | `26353e9c-2a0f-1ebb-3338-0d986bb8691d` | `Silent Founder OS-1.1.0.AppImage` | 108,466,264 | `40918b1fcfcefcc228d3bd0634e329c835c218c7c2549d90c50bc779d27d5528` |

Web-console's earlier proxy and isolated-target failures remain historical diagnostics, not the selected evidence. The isolated `web-console@desktop-smoke` target still fails UI freshness decoding (`unknown field "requires"`), so the qualifying capture used the completed bundled pipeline and its app-specific journey. The artifact digest is identical across the cold and warm cohorts. Both cohorts use host fingerprint `sha256:25e724e0f92366352b955c92f1b246211378b8e690a2f265b3e8db72552466bf`, Xvfb, disabled profiler, and `power_state=unknown`; all ten journeys in each cohort passed. Every cold sample had 0% pre-launch page-cache residency. Required metrics are available in all 20 samples.

| Metric | Cold p50 | Cold p95 | Warm p50 | Warm p95 |
|---|---:|---:|---:|---:|
| First window | 473.0 ms | 631.9 ms | 467.1 ms | 605.1 ms |
| Backend ready | 1,599.0 ms | 1,839.4 ms | 1,671.1 ms | 1,781.5 ms |
| First contentful paint | 1,817 ms | 2,110 ms | 1,826 ms | 2,003 ms |
| App interactive | 1,765 ms | 2,050 ms | 1,773 ms | 1,946 ms |
| Interactive tree PSS | 363.3 MB | 479.5 MB | 351.2 MB | 634.8 MB |
| Steady PSS | 603.7 MB | 678.3 MB | 622.8 MB | 639.8 MB |
| Idle CPU | 11.07% | 12.87% | 11.36% | 16.01% |
| Artifact size | 159,395,184 B | 159,395,184 B | 159,395,184 B | 159,395,184 B |

## Superseded web-console diagnostics

The earlier bundled artifact (pipeline `fde289a3-0768-4b50-2c6b-5fc8b724d94a`)
predates the current UI readiness mark and had no `app_interactive`. Its live
scenario UI was stale because the Vite build had no freshness manifest. After
confirming no in-flight Test Genie run, a lifecycle restart rebuilt the UI,
returned healthy, and cleared freshness. A current proxy artifact was then
built through pipeline `9c89d777-00d2-f841-5667-de4b8194950f` with the source
mark present in served `ui/dist`. One generic cold diagnostic emitted
`app_interactive` at 1,506 ms and retained interactive tree PSS/USS
289.8/160.6 MB, steady PSS 417.2 MB, idle CPU 10.6%, and 0% cold residency.
Its `desktop.launch.baseline` receipt
`5a928154-d441-4cb6-b116-17f7dfee3c6b` failed at `key_press` with
`surface_unchanged`; later journey steps did not run. This current-source
diagnostic is excluded from all baselines. It was superseded by the selected
bundled web-console cohorts documented above; LPBS now substitutes for GCT.

The same current artifact was also tested with web-console's app-specific
`web-console` journey in experiment `E1-web-console-owner-journey-diagnostic-20261012`.
Capture `76e11f64-f017-492c-8615-2b633ef94b59` failed in its
`terminal_fixture` step because the journey target API URL was unavailable.
The packaged sample still recorded `app_interactive` at 9,326 ms, interactive
tree PSS/USS 290.8/161.9 MB, steady PSS 417.5 MB, idle CPU 24.0%, artifact
size 108,466,269 B, and 0% cold page-cache residency. System-monitor reported
normal memory pressure and no open `memory_pressure` alert before capture; its
swap-in rate was 1.26 MB/s. Since the app-specific journey failed, these
metrics are diagnostic only and do not qualify web-console as a reference.
Read-only status confirms `web-console@desktop-smoke` is `start-failed` because
the UI freshness decoder rejects the unknown manifest field `requires`. The
proxy journey resolver requires this isolated instance and refuses to reuse the
live instance, so the missing API URL is a consequence of that owner-side
readiness failure. No isolation fallback or web-console source change was made.

## LPBS fallback cold and warm baselines (GCT substitute)

The current generic `desktop.launch.baseline` fallback artifact produced two
comparable ten-run cohorts: `E1-LPBS-cold-baseline-20261012` and
`E1-LPBS-warm-baseline-20261012`. The artifact digest is
`sha256:40918b1fcfcefcc228d3bd0634e329c835c218c7c2549d90c50bc779d27d5528`;
both groups have the same host fingerprint, Xvfb display, disabled profiler,
and `power_state=unknown`. All twenty journeys passed and all required
interactive/tree PSS, steady PSS, and artifact-size metrics are available.
Every cold sample had 0% pre-launch page-cache residency. `backend_ready` is
unavailable in all twenty samples.

| Metric | Cold p50 | Cold p95 | Warm p50 | Warm p95 |
|---|---:|---:|---:|---:|
| First window | 481 ms | 730 ms | 489 ms | 634 ms |
| First contentful paint | 928 ms | 1,431 ms | 820 ms | 999 ms |
| App interactive | 913 ms | 1,311 ms | 806 ms | 985 ms |
| Interactive tree PSS | 122.1 MB | 360.8 MB | 110.5 MB | 359.3 MB |
| Steady PSS | 570.8 MB | 619.7 MB | 558.0 MB | 615.6 MB |
| Idle CPU | 91.4% | 131.5% | 89.6% | 96.9% |
| Artifact size | 108,466,264 B | 108,466,264 B | 108,466,264 B | 108,466,264 B |

The idle CPU values are unusually high and the interactive PSS distributions
are wide; they are retained without normalization. System-monitor pressure was
normal with no memory-pressure alert at each cohort's start, then rose during
the long batches. Both cohorts completed per protocol. These results meet the
per-sample cold/warm evidence gate for LPBS as one recorded substitute, while
the host-load-sensitive CPU and memory measures require cautious interpretation.

## Diagnostic hello-desktop observations

Host fingerprint: `sha256:25e724e0f92366352b955c92f1b246211378b8e690a2f265b3e8db72552466bf`.
Display is Xvfb; profiler is disabled; power state is unknown. The measured
flow now takes the 60-second steady window after the interaction steps and
before the final `quit_app` teardown, so the application remains measurable
and the journey still records its clean shutdown step. These probes remain
one-run diagnostics, not comparable baselines.

| Probe | Condition | Journey | First window | Backend ready | FCP | Interactive | Tree PSS / USS | Idle CPU | Artifact |
|---|---|---|---:|---:|---:|---:|---:|---:|---:|
| `E1-hello-desktop-warm-probe-20261012` | warm | pass | 483 ms | 1,639 ms | 1,770 ms | 1,704 ms | 280.7 / 165.0 MB | 0.53% | 138,345,260 B |
| `E1-hello-desktop-warm-steady-20261012` | warm | pass | 704 ms | 1,988 ms | 2,171 ms | 2,103 ms | 289.7 / 175.6 MB | 0.78% | 138,345,260 B |

The second probe's post-interaction steady PSS is 274.8 MB and available; its
60-second window precedes the explicit final quit step. Its journey disposition
is pass and the process-lifetime check passed.

The earlier cold probes predate the `/proc` fix or have failed journey /
unavailable metric evidence. They are retained in the evidence store but are
excluded from this table and all baseline calculations.

## First hello-desktop cold batch

Experiment `E1-hello-desktop-cold-baseline-20261012` contains ten comparable
current-source cold samples. All ten journeys passed, all ten required launch,
interactive-tree PSS, steady PSS, and artifact-size metrics are available, and
all ten page-cache residency measurements are 0%. The report's nearest-rank
summaries are:

| Metric | p50 | p95 | Min–max |
|---|---:|---:|---:|
| First window | 493 ms | 628 ms | 438–628 ms |
| Backend ready | 1,623 ms | 1,947 ms | 1,540–1,947 ms |
| First contentful paint | 1,875 ms | 2,238 ms | 1,742–2,238 ms |
| App interactive | 1,805 ms | 2,167 ms | 1,682–2,167 ms |
| Interactive tree PSS | 292.4 MB | 303.7 MB | 279.4–303.7 MB |
| Steady PSS | 246.7 MB | 300.0 MB | 154.5–300.0 MB |
| Idle CPU | 0.50% | 0.65% | 0.37–0.65% |
| Artifact size | 138,345,260 B | 138,345,260 B | 138,345,260 B |

## Rejected hello-desktop cold repeat

Experiment `E1-hello-desktop-cold-repeat-20261012` retained ten cold samples
with 0% artifact page-cache residency. Eight journeys passed and two failed
at `semantic_greet` with HTTP 404, so this is diagnostic evidence and does
not qualify as the second repeatability cohort. Its nearest-rank
`app_interactive` p50 is 1,706 ms, 5.5% below the valid baseline p50 of
1,805 ms; the numeric difference is not a repeatability result because the
journey requirement failed. The two failed receipts are
`35001a26-cc3c-44c1-905f-769efb269cf6:demo` and
`d3921b7f-f8b8-45d9-a232-526cc04d3e66:demo`.

The second failure confirmed that the running scenario-to-desktop API had not
loaded the transient-404 retry change. The API has since been rebuilt and
restarted through the scenario lifecycle; the focused regression for one
transient 404 followed by the expected greeting passes. A clean independent
second capture followed the rebuild and lifecycle restart, passed all ten
journeys, and is recorded in the repeatability section below. Other references
still need their cold and warm batches.

## Hello-desktop cold repeatability result

Experiment `E1-hello-desktop-cold-repeat2-20261012` contains ten comparable
current-source cold samples. All ten journeys passed, required interactive
and steady PSS plus artifact-size metrics are available, and all ten page-cache
residency measurements are 0%. Its `app_interactive` p50 is 1,737 ms versus
1,805 ms in the first cold batch, a 3.8% difference and a repeatability pass.

| Metric | p50 | p95 | Min–max |
|---|---:|---:|---:|
| First window | 471 ms | 592 ms | 369–592 ms |
| Backend ready | 1,572 ms | 1,781 ms | 1,467–1,781 ms |
| First contentful paint | 1,797 ms | 2,020 ms | 1,647–2,020 ms |
| App interactive | 1,737 ms | 1,947 ms | 1,585–1,947 ms |
| Interactive tree PSS | 293.6 MB | 303.9 MB | 274.9–303.9 MB |
| Steady PSS | 247.7 MB | 295.3 MB | 159.2–295.3 MB |
| Idle CPU | 0.45% | 0.64% | 0.30–0.64% |
| Artifact size | 138,345,260 B | 138,345,260 B | 138,345,260 B |

## Hello-desktop warm batch

Experiment `E1-hello-desktop-warm-baseline-20261012` contains ten comparable
current-source warm samples. All ten journeys passed, and all ten interactive
tree PSS, steady PSS, and artifact-size metrics are available. The discarded
warm-up also completed before these retained runs.

| Metric | p50 | p95 | Min–max |
|---|---:|---:|---:|
| First window | 459 ms | 537 ms | 393–537 ms |
| Backend ready | 1,528 ms | 1,850 ms | 1,438–1,850 ms |
| First contentful paint | 1,647 ms | 1,965 ms | 1,556–1,965 ms |
| App interactive | 1,585 ms | 1,902 ms | 1,492–1,902 ms |
| Interactive tree PSS | 288.7 MB | 300.5 MB | 273.4–300.5 MB |
| Steady PSS | 243.5 MB | 291.6 MB | 161.6–291.6 MB |
| Idle CPU | 0.45% | 0.65% | 0.33–0.65% |
| Artifact size | 138,345,260 B | 138,345,260 B | 138,345,260 B |

## Current sidecar startup waterfall

The commissioned legacy baseline's `runtime_spawn_to_ready` p50 is 6,178 ms
for `bundled` mode and artifact digest
`sha256:72aae12d30398b89d248d0ed10bdec1d0c0308db10dc20d08eea33ac9fbe16bb`.
The current E1 sidecar artifact has a different digest and its measured
`runtime_spawn` to `runtime_ready` p50 is 1,086 ms. The detailed buckets below
explain that current interval. Recovered per-launch telemetry for the legacy
bundled samples now supplies the historical attribution in the next section;
the current-sidecar data is kept separate because its artifact and deployment
mode differ.

For the first cold batch, `runtime_spawn` to `runtime_ready` has p50 1,086 ms
and p95 1,276 ms. These adjacent named intervals cover the complete measured
span in every trace; per-bucket nearest-rank medians sum to approximately
1,090 ms because each bucket ranks its own ten observations. This explains
the current cold p50. Both independent cold batches use the same current
source artifact and matching host/display/profiler attribution.

| Bucket | Start → end | p50 | p95 |
|---|---|---:|---:|
| Runtime exec / mark ordering | `runtime_spawn` → `runtime_exec_complete` | 0 ms | 1 ms |
| Runtime token | `runtime_exec_complete` → `runtime_token_available` | 152 ms | 153 ms |
| Service startup to first spawn | `runtime_token_available` → `service_spawned:api` / `service_spawned:ui` | 171 ms | 395 ms |
| IPC port publication | service spawn → `runtime_ipc_port_published` | 42 ms | 66 ms |
| Runtime health | `runtime_ipc_port_published` → `runtime_health_ready` | 23 ms | 28 ms |
| API readiness and retry | `runtime_health_ready` → `service_health_success:api:attempt-2` | 428 ms | 449 ms |
| Backend readiness | API health success → `backend_ready` | 274 ms | 301 ms |
| Final ready handoff | `backend_ready` → `runtime_ready` | 0 ms | 0 ms |

The UI health check succeeded on its first attempt when the UI service spawned.
The API succeeded on attempt two. Its 428 ms p50 span from runtime health ready
to the successful API health mark includes the retry delay and request time;
the current trace does not separate those two components. In a few runs the
runtime-exec-complete mark precedes runtime-spawn by less than 1 ms, so that
bucket is reported as 0 ms at p50 and kept as mark-order uncertainty.

### Temporary readiness-poll candidate experiment

To test the `health_success` 500 ms retry interval, a one-run current-bundled
control was captured from pipeline `2b7b85fa-3265-68d5-391e-2074ccf7492b`
(artifact digest `sha256:19f368cc0be99bbd92cd1e2bf51e4408b18edd3173e139797079f505c0fecfa4`).
Its passing journey measured `runtime_spawn` at 444.1 ms and `runtime_ready` at
1,528.98 ms (1,084.86 ms interval); API health succeeded at 1,245 ms, 800.88
ms after runtime spawn. A temporary source-only change lowered the retry
interval from 500 ms to 50 ms, then rebuilt via pipeline
`dc8f4244-7c28-dd45-15a3-fd26110c2cdb` (artifact digest
`sha256:47f073baf2247b290cd74047bfd3a158282a996ffafc3ee2fb07f387bb324d40`).
Its passing cold journey measured runtime spawn at 429.42 ms and runtime ready
at 806.82 ms (377.40 ms interval); API health succeeded at 761 ms, 331.58 ms
after runtime spawn. This single-run directional result supports the polling
interval as a roughly 469 ms contributor to current startup. It is a smaller contributor than the 4-second legacy GPU-detection stage
identified in the recovered historical traces below. Control and experiment
each have only one run, so their downstream timing difference is not a stable
estimate. The temporary source
change was restored byte-for-byte (original SHA256
`f830a8827792c2d3a30ef982692c327297053d9ccd9281a3e4808f589b3f752e`); no
optimization ships in E1. For E3, investigate the old runtime/service startup
path and preserve the poll interval as a confirmed smaller contributor.

### Temporary Electron readiness-loop poll experiment

To test `templates/vanilla/main.ts`'s 350 ms `/readyz` loop, a temporary
source-only variant reduced the delay to 10 ms and rebuilt hello-desktop through
pipeline `a9914446-baae-394e-372e-e36dd27a88c8` (artifact digest
`sha256:80272ae9876e97c246c55ced255aaea0bac616387a80df0c9f48ac17c2b0576b`).
The cold `hello-desktop` journey passed in sample
`0ad725bc-cc52-490f-8935-ec6a032c03f4:demo`. Its `runtime_spawn` to
`runtime_ready` interval was 835.8 ms, versus 1,084.9 ms in the one-run
current-bundle control (`sha256:19f368cc0be99bbd92cd1e2bf51e4408b18edd3173e139079f505c0fecfa4`;
control sample `733f50b0-37db-4707-b724-e2554b3b275a:demo`). API health
succeeded at 1,262 ms after process start in the variant versus 1,245 ms in
the control; the ready handoff after API health shortened from about 284 ms to
11 ms. This directional single-run result isolates roughly 273 ms of current
build handoff delay. `main.ts` was restored byte-for-byte to SHA256
`f6a319515ee144940978ef5f88a119b63653b3a1536c6db74b03966b722e7846`; no
optimization ships in E1. This is a directional single-run current-build result. The recovered legacy
trace below separately attributes the historical bundled interval; no
optimization ships in E1.

## Recovered historical bundled waterfall

`performance-baseline.json` records the legacy bundled artifact digest
`sha256:72aae12d30398b89d248d0ed10bdec1d0c0308db10dc20d08eea33ac9fbe16bb`
and a cold `runtime_spawn_to_ready` p50 of 6,178 ms across five samples (warm
p50: 6,171 ms across five samples). Although the AppImage bytes and the
scenario-to-desktop capture objects are no longer retained, all ten
`samples[].run_id` values match original app `:demo` launch marks by run ID;
for each mark window, the runtime log has one matching `runtime_start`, GPU
result, and service-start/readiness sequence by timestamp. The retained sources
are `~/.config/hello-desktop-desktop/deployment-telemetry.jsonl`
(SHA256 `2b728139abffab5229d6c9433843f5ac31a8d06a365b0aeb41c55420d20852be`),
`~/.config/hello-desktop-desktop/runtime/telemetry/deployment-telemetry.jsonl`
(SHA256 `3ab6ba717f5ecab0061d079d8c1b42647fdfa08812b76780ef846060fbb1b939`),
and this baseline JSON (SHA256
`77b19be0a29800b8c64e0397b813398b33e7042e3bcf110581123d0a0f5f99d3`).

The sequential bucket p50s below use the five cold traces and nearest-rank
aggregation. The four p50s sum to 6,176.977 ms, or 99.98% of the historical
6,178 ms p50. Independent per-bucket percentiles need not sum exactly to the
aggregate percentile.

| Historical bucket | Boundary | Cold p50 (range) | Warm p50 (range) |
|---|---|---:|---:|
| Runtime process exec and auth-token availability | Electron `runtime_spawned` → `runtime_token_available` | 2.453 ms (1.284–3.444) | 2.601 ms (2.342–3.043) |
| Runtime initialization through synchronous GPU probe | `runtime_token_available` → runtime `gpu_status` | 4,005.470 ms (4,005.090–4,007.060) | 4,004.425 ms (4,003.371–4,043.004) |
| GPU result through API service first ready | runtime `gpu_status` → `service_ready:hello-desktop-api` | 2,005.309 ms (2,002.050–2,011.790) | 2,002.663 ms (2,001.984–2,004.468) |
| Final readiness polling slack | `service_ready:hello-desktop-api` → Electron `runtime_ready` | 163.745 ms (161.480–181.150) | 158.966 ms (136.528–180.833) |

Service telemetry confirms the UI service starts and becomes ready in a p50 of
0.666 ms cold / 0.733 ms warm; its start overlaps the API's startup. The API
service's own `service_start` → `service_ready` p50 is 2,004.577 ms cold /
2,001.214 ms warm. In the historical runtime, `service_ready` is written only
after `WaitForReadiness` succeeds. That event is the available first-success
health milestone; the old runtime did not retain each probe attempt. The
control IPC health endpoint responds a p50 51.818 ms after `gpu_status` in cold
runs and 45.629 ms in warm runs; this overlaps service startup and is not added
again. Electron discovers the UI port a p50 1.888 ms after `runtime_ready`
(cold), outside the measured `runtime_spawn_to_ready` interval.

The approximately 4.005 s runtime-init/GPU bucket is not an unexplained
residual. At the nearest preceding committed source revision
`3e921a254a8fe6880997653e0106dfbc97891e08`, `Supervisor.Start` calls
`initGPUAndDomainObjects` synchronously after recording `runtime_start`;
`initGPUAndDomainObjects` calls `gpuDetector.Detect()` and writes `gpu_status`
only after it returns. That detector has a 4 s context deadline. Across all ten
matched launches, telemetry reports GPU unavailable (`reason=no GPU detected`)
and the `runtime_start` → `gpu_status` p50 is approximately 4,005.6 ms. The
repeatable measured duration aligns with the detector deadline and identifies
the blocking stage; the telemetry does not identify which lower-level GPU
probe consumed the deadline.

This closes the historical waterfall attribution gate from retained per-run
telemetry despite the missing binary. The two temporary E1 polling experiments
remain useful checks of smaller current-build contributors, not substitutes
for the historical measurements. They were restored byte-for-byte and no
optimization ships in E1.

## Completed gate evidence and historical diagnostics

### System-monitor cold and warm baselines

System-monitor was rebuilt from fresh lifecycle outputs after the first new
pipeline produced an artifact whose UI bundle omitted the source readiness
mark. The lifecycle refresh returned healthy and `vrooli scenario freshness`
reported `stale: false`; the second current-source artifact is pipeline
`8178f443-c638-c9f5-9fd2-cbecaba0a17c`, digest
`sha256:dedb18edfa50e883fdf56f8f0acdf9dcceed43f7992759d76569caced12b755d`,
size 156,681,719 B. Both ten-run cohorts use `desktop.launch.baseline`; all
twenty journeys passed, each sample reports the declared `app_interactive`
mark and complete interactive/steady PSS, and every cold artifact had 0%
page-cache residency. The host's swap-in pressure rose into elevated and
briefly critical ranges during portions of the already-running cohorts; captures
were not interrupted, and this variability is a limitation on interpreting the
timings as quiet-host performance.

The app's own host-metrics polling contributes real work to `idle_cpu_pct`.
The measured idle CPU p50 is therefore not an estimate of packaging-only idle
cost: 43.25% cold and 42.52% warm.

| Condition | First window | Backend ready | FCP | Interactive | Interactive tree PSS | Steady PSS | Idle CPU | Artifact |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| Cold (10) | 464 ms | 1,664 ms | 1,977 ms | 1,976 ms | 330.8 MB | 393.8 MB | 43.25% | 156,681,719 B |
| Warm (10) | 490 ms | 1,679 ms | 1,899 ms | 1,897 ms | 323.8 MB | 412.6 MB | 42.52% | 156,681,719 B |

Cold experiment: `E1-system-monitor-cold-baseline-20261012`. Warm experiment:
`E1-system-monitor-warm-baseline-20261012`. The predecessor current-source
artifact built before lifecycle refresh is not used; its UI bundle lacked the
readiness mark.

- The selected reference set is hello-desktop, system-monitor, web-console,
  and LPBS as the D2-approved GCT substitute. Each has comparable 10-run cold
  and warm groups; the current packaged report shows passing journeys and all
  required metrics. All 40 selected cold samples report 0% artifact page-cache
  residency. Earlier LPBS, GCT, onboarding, and web-console failures below are
  retained as diagnostics only.
- Hello-desktop cold repeatability passes using the 10/10 passing groups
  `E1-hello-desktop-cold-baseline-20261012` (p50 1,805 ms) and
  `E1-hello-desktop-cold-repeat2-20261012` (p50 1,737 ms): 3.77% difference.
  The earlier repeat group with two failed journeys is excluded.
- Both required Test Genie unit phases returned terminal PASS after DPF-FB-012
  pressure checks: performance-health run `20261012-141224-b8454193` and
  scenario-to-desktop run `20261012-141306-c439f4cf`; each was waited once.
  Their findings report 56 and 93 non-blocking unit-health warnings. The live
  Test Genie run index currently returns no rows for either scenario, and the
  scenario-to-desktop findings read reports `canonical terminal persistence
  unavailable: run not found in index`; the terminal wait results remain PASS,
  while durable indexed lookup is unavailable. The focused protobuf, Go
  package, and runtime Go suites passed in the final gate check; the final
  targeted forbidden-command scan is recorded in the 12:10Z E1 slice-log entry.

### Reference capture diagnostics (2026-10-12)

Web-console's initial failed capture was stopped after its first retained sample
showed that `app_interactive` and the corresponding process-tree snapshot were
unavailable. A later rebuilt artifact included the UI readiness mark, but its
two retained diagnostic samples still do not qualify: both observed the mark,
while `steady_pss_mb` was unavailable and the `web-console` journey failed
because the smoke target API URL was unavailable. The exact
`web-console@desktop-smoke` instance reports `start-failed` with
`component ui freshness: decode manifest: json: unknown field "requires"`;
the normal live `web-console` instance remains healthy. The capture was
interrupted after this repeated failure. These diagnostic samples are excluded
from every baseline and repeatability calculation. At this point in the investigation, the isolated owner-side smoke-target
failure prevented a proxy measurement. Later, the bundled app-specific journey
produced the selected 10-cold/10-warm web-console cohorts documented above; the
isolated target freshness failure remains a historical, non-selected path.

The Git Control Tower sidecar artifact was rebuilt through scenario-to-desktop
pipeline `13d883a0-9c57-cc22-10c5-a4f2bd41ff40` after a controlled lifecycle
restart refreshed its shared UI build. Its exact `@desktop-smoke` instance is
healthy, with the unrelated `scenario-auditor` dependency degraded. The first
cold capture was interrupted when system-monitor reported elevated memory
pressure before any sample was retained; no memory-pressure alert was active.
No GCT baseline is claimed. Two attempts were interrupted before a sample
was retained as memory pressure repeatedly returned to elevated. Retry after
the host sustains normal pressure.

A third cold-cohort attempt followed a normal-pressure reading and a healthy
`git-control-tower@desktop-smoke` lifecycle start. It failed during warmup with
HTTP 502: scenario-to-desktop could not find the measured-launch record
`4de66098-b76f-48d1-9830-92f02cabc1a4-demo`. No sample was retained and the
empty output is not baseline evidence. The isolated smoke target was stopped;
no GCT baseline is claimed.

The retained protocol launch and API log identify the GCT warmup failure:
`git-control-tower`'s bundled API panics with
`capability manifest invalid: read service manifest: open ../.vrooli/service.json:
no such file or directory`. The packaged runtime reports no `runtime_ready`
mark and exits; the measured-launch handler therefore returns HTTP 502 without
a demo sample. This is a current-source GCT packaging/source failure, not a
host-pressure result. The API owner path is outside E1's admitted source paths, so this GCT evidence
remains excluded. The later D2-approved LPBS cohort now fills its reference
slot; no GCT source change was made.

A proxy-mode current-source artifact was then built through pipeline
`b8c6be19-37b3-a3b8-dd42-7667a9b2ab17`, avoiding the bundled API path without
editing GCT source. Its one-run generic diagnostic
`E1-GCT-proxy-diagnostic-20261012` emitted `app_interactive` at 705 ms and
retained interactive tree PSS 123.4 MB, steady PSS 424.9 MB, idle CPU 13.1%,
artifact size 108,585,232 B, and 0% cold page-cache residency. The journey
receipt `91203bc1-fa87-4aa2-826a-89320069c128` failed at `pointer_click` with
`surface_unchanged`; subsequent steps did not run. This confirms a proxy launch
path but does not provide a passing current journey, so the diagnostic is
excluded and no GCT baseline is claimed. The generic UI parity failure is
outside E1's permitted GCT write paths.

`vrooli-onboarding` was the first D2 fallback for web-console. Its current-source
sidecar artifact was built through pipeline
`d93139c0-ec2f-cc07-578a-15cc2f1e8fa4`; its exact `@desktop-smoke` instance
started healthy. The first one-run diagnostic mistakenly passed the historical
plan ID `desktop.launch.baseline.v2` as the CLI journey capability; its stored
sample reports `capability_not_registered`, no app-interactive or PSS metrics,
and a failed journey, so it is excluded. A retry using the registered
`desktop.launch.baseline` capability began the warm-up and measured launch, but
memory pressure returned to elevated before a sample was retained; that
capture was interrupted. Both isolated smoke instances were stopped afterward.
No onboarding baseline is claimed. The valid capability was
`desktop.launch.baseline`; the plan ID must not be passed as the capture
capability.

Continuation investigation: the later proxy-mode sample in
`E1-onboarding-proxy-current-model-diagnostic-20261012` passed the generic
journey and recorded FCP (731 ms), first window (453 ms), and steady PSS
(287.6 MB), but still had no `app_interactive`, interactive tree PSS/USS, or
idle CPU. Read-only Connect POSTs to both `http://127.0.0.1:24767` (proxy) and
the live API on `http://127.0.0.1:19798` returned HTTP 200 for `GetStepModel`
and `GetOperatorState` with the same responses. That confirms these two
initialization endpoints are reachable through the proxy, but does not explain
why the renderer fails to complete initialization; no onboarding source change
or baseline is claimed.

Retained screenshots from that same sample clarify the mismatch: screenshot
`daafce6a-3ecc-41e2-997a-642b7cfa7e04` shows the visible error
“The onboarding step model could not be loaded.” Later screenshot
`eec96061-dc52-4298-9e2a-6d773b68c768` is a blank dark renderer. The generic
journey passed window-level actions despite this UI error, so its pass is not
evidence of onboarding app readiness. GCT's retained journey inventory was
also rechecked: its four journeys consist of two earlier
`capability_not_registered` attempts and two current generic runs that fail at
`pointer_click` (`surface_unchanged`). Neither candidate supplies a qualifying
current app journey under E1's admitted source and BAS constraints.

The rebuilt current-source sidecar artifact from pipeline
`adae8084-1a45-c0b2-4817-1700ebd0bed8` was captured with the correct
`desktop.launch.baseline` capability in
`E1-vrooli-onboarding-interactive-mark-check-20261012`. The retained one-run
sample has FCP 671 ms and 0% cold artifact page-cache residency, but no
`app_interactive`, interactive or steady process-tree metrics, or idle CPU
window. Its receipt (`8ff07220-6511-4568-a5ca-1f4da4ee0970`) fails at `quit`
because `target_window_visible` times out. The receipt's activation screenshot
shows the explicit UI error “The onboarding step model could not be loaded.”
The sample is diagnostic only and excluded from the baseline. The artifact was
generated in proxy mode with `http://127.0.0.1:24767`; the exact
`vrooli-onboarding@desktop-smoke` was not running for this capture. Its first
start attempt was refused because refreshing its shared build would replace
files served by the live onboarding instance. A later normal-pressure
`vrooli scenario restart vrooli-onboarding` refreshed the live UI; the isolated
smoke target then started healthy. A second diagnostic against that healthy
target still shows the same onboarding model-load error and has no
`app_interactive` or process metrics, so onboarding is superseded as the E1
fallback. Both samples are diagnostic only; no onboarding baseline is claimed.
The latest receipt is `dd2e895f-a56a-484f-aad5-bc059abe15a1` and has the same
`quit` visibility timeout. `landing-page-business-suite` is now the recorded
fallback candidate. Its prior passing desktop journeys are not current E1 evidence.

The refreshed current-source LPBS proxy artifact was captured once in
`E1-lpbs-mark-diagnostic-20261012` (artifact digest
`sha256:11270f0c8bd2749f9c0ba3711e4df522476da605185208806f223cf9cc957fa9`).
The packaged launch succeeded and FCP was 968 ms, but `app_interactive`,
process-tree PSS/USS, idle CPU, and steady PSS were unavailable. The
`monetization.trust-boundary.v1` receipt
`768d4d65-9beb-4537-9703-6ef331daa527` failed at `signin_shared_session`
because the bundled scenario renderer URL was unavailable; later journey steps
did not run. This is diagnostic only and excluded from all baseline cohorts.
At this point, LPBS did not yet have a qualifying cohort. Subsequent current
proxy runs produced the 10-cold/10-warm LPBS cohorts at the top of this file;
LPBS now substitutes for GCT while bundled web-console is restored as a
selected reference.


### LPBS generic-launch follow-up (2026-10-12)

The first generic `desktop.launch.baseline` retry could not open its artifact
because pipeline `cf7b4df0-5b50-9c7e-87e9-ab31056b4d5e`'s cached AppImage had
already been removed. Rebuilt it through pipeline `225abb54-f6ba-dc36-eb90-4c407591c49f`,
but that artifact pointed at stale proxy URL `http://127.0.0.1:21225`; telemetry
records `ERR_CONNECTION_REFUSED`. LPBS was healthy at UI port 23224. Pipeline
`26353e9c-2a0f-1ebb-3338-0d986bb8691d` was rebuilt with the live port. Its first
capture showed FCP but no interactive mark because the served `ui/dist` lacked
the source mark; the generic journey timed out at `quit` and metrics were
unavailable. After lifecycle-stopping LPBS, running its existing `pnpm run
build` UI owner script, confirming the mark in the built and served assets, and
starting LPBS through `make start`, a current one-run diagnostic emitted the
mark and interactive tree PSS. The diagnostic still failed: its 180-second
measured-demo hold expired during the post-journey steady-PSS window, so the
final `quit` readiness check timed out and steady PSS was unavailable. The
sample is excluded from baselines. Its `idle_cpu_pct` was 113.0% while host
pressure later reported elevated, so it is not a quiet-host measure.

The admitted measured-launch path now gives measured runs a 360-second app hold
and a 420-second executor bound. A focused regression checks that the hold
covers the measurement windows and that the executor bound exceeds the hold.
`go test ./procmetrics/... ./smoketest/... -count=1` passes. The API was rebuilt
and restarted through the lifecycle after confirming no Test Genie run was in
flight. The subsequent one-run generic diagnostic used the same current artifact
(pipeline `26353e9c-2a0f-1ebb-3338-0d986bb8691d`, digest
`sha256:40918b1fcfcefcc228d3bd0634e329c835c218c7c2549d90c50bc779d27d5528`).
Its `desktop.launch.baseline` journey passed (capture
`a3d86f3e-f3e9-42e8-9469-91cde57bc390`); app_interactive was 1,125 ms, FCP
1,173 ms, interactive tree PSS/USS 294.4/170.0 MB, and steady PSS 577.1 MB.
The AppImage was 108,466,264 B and cold page-cache residency was 0%. The
diagnostic's idle CPU was 103.9% during elevated host pressure, so it is
excluded from baselines. This confirms the measured hold fix and a working
current LPBS journey, but one diagnostic was not a 10-cold/10-warm reference
cohort. At the next pressure check the monitor reported normal memory pressure
but 2.9 MB/s swap-in, so cohort capture was deferred at that point. Later, both
LPBS 10-run cohorts completed and became the selected substitute for GCT, as
documented above.


## GCT exclusion

GCT does not qualify for the fourth slot. The current proxy diagnostic artifact emitted app-interactive and process metrics, but its journey failed at `pointer_click` (`surface_unchanged`). The bundled current-source attempt panicked because the API could not find `../.vrooli/service.json` inside the bundle. The retained current journey therefore fails the required semantic check; no GCT source change or BAS action was made. LPBS is the selected D2-approved substitute.
