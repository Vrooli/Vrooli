# Managed HyperFrames

HyperFrames is a bounded local rendering tool. It has no standing service or
resource lifecycle. The control plane owns installation and process containment;
the launch-video skill owns creative direction and review.

## Installation contract

The supported dependency operation is:

```
scenario-dependency-analyzer deps install npm/hyperframes@0.8.112 --scenario hyperframes --surface host-tool --exact --json
```

The default operation only previews governance. Add `--apply` after admission.
This explicit surface provisions the committed frozen closure in a private,
versioned candidate and atomically selects it after verification. It never
installs into a scenario UI. Lifecycle scripts remain disabled. The existing
observed lock is imported as provenance, not resolved again to latest versions.
`vrooli host install hyperframes` delegates the same governed operation.

Node >=22.12.0, FFmpeg/FFprobe and a compatible existing browser are separate
host prerequisites. Installation and rendering never download a browser.
Portable browser acquisition and non-Linux rendering remain unqualified.

## Rendering contract

```
vrooli host hyperframes --request render-request.json --out render-bundle --json
```

The version-1 request identifies an input directory and every regular input
file by relative name and SHA-256, composition subdirectory, and explicit Node,
browser, FFmpeg and FFprobe executable paths and SHA-256 values. It declares a
wall timeout and memory ceiling. The owner resolves the verified managed
HyperFrames closure. Rendering uses one worker, software graphics, strict
capture and no extracted-frame cache. It sets telemetry opt-out and selects
the declared browser, with no inherited browser override.

The requested deadline begins at invocation entry, including request parsing,
managed installation verification, input/executable hashing, snapshot copying
and reuse checks. Local file loops check cancellation between reads/entries.
This is cooperative cancellation; it cannot interrupt a kernel filesystem call
that does not return. Scope cleanup has a separate bounded recovery budget.
On Unix, source/request/hash reads open with nonblocking/no-follow flags and
verify the opened regular-file inode against the pre-open observation. A FIFO,
symlink or different regular file substituted before open is refused. Select
canonical regular executable paths; a symlink launcher is not a pinned binary.

The receipt binds declared **local bytes**. It does not isolate browser network
access or prove that composition code cannot fetch external assets. Production
must review HTML/CSS/JS and retain local-only asset evidence. A render receipt
alone must not be described as a complete network dependency closure.

The owner copies inputs into a private candidate, runs `check`, renders, fully
decodes and probes the output, and checks identities again. It publishes a
directory containing `render.mp4`, `receipt.json` and bounded logs by rename.
A reusable bundle must match the complete request/tool key and actual output
hash. The request schema and placeholder example are in `render-request.schema.json`
and `render-request.example.json`. Every file below input_dir must be declared,
including any frozen-input manifest placed there.

A mismatched existing destination is refused; callers select a new
destination. Failure and cancellation leave the prior destination intact and
stop only the operation-owned process scope. A technical receipt does not
establish aesthetic quality or listening.

## Debt retired

Fresh production uses this owner instead of the historical effort launcher
with hard-coded NVM/scratch paths, existence-only reuse and direct final writes.
Old films and historical scripts remain evidence, not the supported entrypoint.

The installation root resolves through the control-plane runtime-home cache
contract, under `host-tools/hyperframes/<closure-digest>-<os>-<arch>` (normally
`~/.vrooli/cache/host-tools/hyperframes`). It does not follow a scratch directory
or an arbitrary PATH installation. The imported npm lock retains its historical
root package metadata; every resolved dependency and integrity value is
preserved. The exact managed manifest and frozen lock are jointly identified.
No dependency is re-resolved during provisioning.

Dependency upgrades use SDA's explicit `--refresh-lock --exact` host-tool source
operation, separately from frozen runtime provisioning. SDA governs the target
version and runs a script-disabled, lock-only resolver in a private candidate.
The tool owner validates the exact root pin and integrity-bearing closure, fences
the original source hashes, and rolls back ordinary publication errors. Two file
replacements are not crash-atomic: an incomplete source pair fails the existing
exact-closure validator and requires recovery before any provisioning. A successful
source refresh reports `source_updated`, never `installed`; previous immutable
runtime bytes remain intact. Review the new pair and align the owner Version,
then rebuild the host CLI and activate SDA against that embedded closure before
separate frozen provisioning. A newly compiled owner selects its own closure
identity; it does not silently fall back to the previous installation. Preserve
the prior compiled runner for old production until the new runtime is qualified.
Host-tool dependency operations have a three-minute server budget and a bounded
CLI allowance of three minutes fifteen seconds. SDA retains the last 1 MiB of
package-manager output with an explicit truncation marker.

Browser profiles and operation caches use the candidate's HOME/TMPDIR. A 4 GiB
sampled temporary-disk budget cancels work when exceeded; it is not a filesystem
quota. The source set is at most 2 GiB and logs at most 8 MiB. Final verification
and creative reviews remain separate from renderer checks.
Chromium may delete temporary children during a sample: confirmed child ENOENT
is tolerated; an unavailable root, permission errors and other I/O failures cancel
the job. Size accounting checks the budget before addition to prevent overflow.

The fixed render profile v2 uses one capture worker, software browser rendering,
strict extraction, PNG frames, no frame cache, high quality, CPUWeight 50 and
TasksMax 512 per owned scope. Request memory and wall limits still apply (the
production request uses 4GiB). The complete fixed profile is included in the
render key and receipt; v1 task-limit receipts cannot be reused. The owner samples
the owned scope's task-denial counter every 250ms and immediately cancels through
canonical scope cleanup on any observed denial, including at its first sample.
Collected or unavailable counters remain unknown; sampling cannot certify zero
events after a short-lived scope has disappeared.
Each completed owned stage writes one bounded `owned_task_observation` JSON
summary to `render.log`: sample count, known/unknown counter reads, maximum
observed denials, peak observed tasks and observed ceiling, plus scope identity.
No known readings produce `null` maxima rather than zero. Version/probe output
captures use a separate sink and retain their native formats.

This owner does not independently bound extraction concurrency or native FFmpeg
thread pools; capture worker count and OMP/BLAS settings are not such limits.
The 0.8.112 upstream extraction call forwards the full configuration, fixing the
cache-only forwarding in 0.8.44. This profile does not set a separate FFmpeg
process timeout; the outer request deadline remains enforced. Upgrade runtime
qualification is required before inferring native behavior from source. Do not patch node_modules, substitute a
hidden binary wrapper, or raise the ceiling again if this profile still hits task
pressure; that requires governed upstream engineering.

## Qualification

The following native observations are historical qualification of 0.8.44.
The current 0.8.112 source closure requires separate frozen provisioning and native
qualification; these earlier results do not certify the upgraded runtime.

The previous full animatic stalled during four-video extraction, hit the old
256-task ceiling and was canceled after 252.7 s with verified owned cleanup and no
final output. That establishes observed task pressure, not its sole causality.
Profile v2 passed a four-original-video trial at actual source resolutions in
37.53 s. The sampled render reached 339 tasks under 512, with 116 known task-counter
reads and zero observed denials. Exact-input reuse returned the same verified
output hash. A separate 16.06 s in-render deadline failed as expected, verified
owned cleanup and published no final bundle. Fresh 960×540 timed animatics then
passed in 99.26 s / 71.92 s, sampled peaks 339/257 with zero observed denials.
Short executable probes sometimes completed before a known counter reading;
these remain unknown. These are bounded local trials, not all-codec or exhaustive
zero-event certification. Receipts: the2026-10-16 video-toolchain effort's
`qa/native-media-*`, `qa/native-profile2-cancel-result.json` and
`production/review/animatic-v3/receipts.json`.

2026-10-16, Linux amd64, experimental. Governed frozen installation succeeded
with lifecycle scripts disabled. The canonical source CLI rendered a one-second
320×180 composition through check/render/full-decode/probe in 12.34 seconds.
An identical request reused the verified bundle. A separate 60-second 1080p
composition hit its 16-second deadline during frame93/1800; the command returned
failed, verified owned-scope cleanup and published no final bundle. An earlier
trial failed the composition checker and is not cancellation evidence.

Receipts live under the runtime-home plan artifacts effort
`video-toolchain-reliability-2026-10-16/qa/`: `native-smoke-v2-result.json`,
`native-reuse-result.json`, `native-cancel-v2-result.json` and the request/bundles.
The explicit prerequisites were Node22.23.2, Chrome for Testing152.0.7977.30
and FFmpeg/FFprobe6.1.1. Browser acquisition, other platforms, arbitrary codecs,
physical-device QR scanning, aesthetic quality and listening are not certified.
The global installed control-plane executable was not replaced; source invocation
and activated scenario APIs are separate facts. Focused regression/race checks
cover snapshot identity, safe regular-file opening, cache refusal, publication,
deadlines and containment failure paths.


### Current 0.8.112 qualification (2026-10-16)

SDA refreshed and separately provisioned the reviewed exact lock with lifecycle
scripts disabled. The dependency tree decreased from189 to124 locked packages;
only HyperFrames changed version among retained packages. The lock-only resolver
reported Node20 engine warnings; rendering used the explicitly hashed Node22.23.2
required by the owner. This is not a claim of Node20 runtime support.

The retained four-video input completed check/render/full-decode/probe in48.39s
with exact output hash binding. Exact-input reuse passed. Observed render scope
peak was339 tasks under512,127 known counter reads and zero observed denials.
An in-render16s deadline returned failure in16.05s, published no final bundle,
and all six observed owned scopes were subsequently inactive/collected. Shared
host timing is not an isolated performance comparison with0.8.44. Native
extraction and FFmpeg thread pools remain unbounded independently of the scope
task ceiling. Linux amd64 remains experimental; portability, browser acquisition
and network isolation remain unqualified.

Evidence: `~/.vrooli/plan-artifacts/efforts/video-toolchain-reliability-2026-10-16/qa/hyperframes112-native-qualification.json`
and its exact request/result/log references. Prior0.8.44 receipts and immutable
installation are preserved; new closure identities prevent accidental reuse.
