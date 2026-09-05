# whisper.cpp managed-service assessment

Evaluation date: 2026-08-17. This document records the native migration and its
remaining target-specific qualification gaps. It is no longer an evidence-only
spike: Linux uses the managed-service path, while unsupported targets remain
explicitly non-promoted.

## 1. Current maturity

Whisper is M5 on the Linux CPU managed-service path: manifest-authoritative
native lifecycle, checksum-pinned server and model acquisition, readiness
through activity-edge, Go live integration coverage, and no Docker path. It
remains below cross-platform promotion because CUDA, macOS, and Windows target
smoke evidence is not yet complete.

| Dimension | Score | Evidence |
|---|---:|---|
| Contract | 2 | `resource.json`, `/asr`, `/detect-language`, activity-edge readiness |
| Archetype | 2 | managed-service supervises upstream whisper-server directly |
| Operator surface | 2 | Go resource CLI / shared control plane |
| Runtime and health | 2 | digest pins and readiness probe |
| Tests | 2 | `cli/live_test.go` plus audio-tools consumer path |
| Portability | 1 | Linux native acquisition is verified; Windows is un-smoked and macOS is unsupported |
| Legacy debt | 2 | no resource shell files |

## 2. Active contract and consumers

The active host contract is `POST /asr`, `POST /detect-language`, and `GET /`
via the activity-edge companion on port 8090. `audio-tools` is the material
consumer. The native supervisor runs whisper.cpp v1.9.2 on port 18090 and the
edge maps the historical `audio_file` form to whisper.cpp's `/inference` form.

## 3. Target archetype

The selected archetype is `managed-service`: a supervised native
`whisper-server` binary with model artifacts acquired like other managed-service
data. The Linux x64 and arm64 upstream archives and the medium GGML model are
checksum-verified on a clean artifact root. The current resource intentionally
launches the CPU candidate; the CUDA candidate, signed macOS server build, and
Windows smoke run remain promotion work.

## 4. Deployment profile

| Target | Candidate delivery | Requirement / result |
|---|---|---|
| linux-amd64 CPU | upstream v1.9.2 archive + medium GGML model | **verified**: clean acquisition and live `/asr` smoke |
| linux-arm64 CPU | upstream v1.9.2 archive + medium GGML model | acquisition declared and tree digest recorded; target smoke pending |
| linux-amd64 CUDA | separate signed CUDA artifact | not promoted; no runtime driver qualification recorded |
| macos-arm64 | signed native server build | **unsupported**: upstream publishes an XCFramework, not a server executable |
| windows-amd64 | upstream x64 archive + medium GGML model | acquisition declared; target smoke pending |

The authoritative artifact catalog is now the `managed_service.acquisition`
block in `resources/whisper/resource.json`. It records the v1.9.2 archive
digests, extracted-tree digests, model digest, and the explicit macOS
unsupported target. The Windows cuBLAS candidate is also declared there and is
selected only when the target facts include CUDA compute capability 8.9 or
newer.

## 5. Gap list

- No Linux CUDA native artifact is published by the v1.9.2 upstream release;
  the declared cuBLAS candidate is Windows x64 only and remains un-smoked.
- Native server exposes `/inference`; the tested activity-edge adapter now
  preserves the `/asr` multipart contract.
- Linux model acquisition and verification are live; Windows and macOS target
  validation remains open.
- Same-corpus medium smoke WER is 0.00 before and after migration. The native
  CPU service is configured for 12 threads and measured 5.3 s for the clean
  speech smoke on this host, versus the recorded faster-whisper GPU 0.29 s.
  This is a quality match on the smoke corpus, not latency parity across hosts.
- The live audio-tools diagnostics suite now passes provider readiness, clean
  speech WER, and no-speech hallucination suppression. Native output such as
  `(beep)` and `[BLANK_AUDIO]` is filtered by the shared egress policy.

## 6. Migration plan

## Accelerator declaration and VRAM claim (2026-08-21)

Whisper declares `backends: ["cuda", "cpu"]` with `require: preferred`. On Linux
it selects the CPU target, because upstream v1.9.2 publishes no Linux CUDA
release asset, and the control plane reports that honestly:

```
declared_mode: cuda   observed_mode: cpu   mode_drift: true
healthy: false        serving: true        status_code: mode_drift
```

That is the accurate state, not a fault. The resource transcribes correctly; it
does so about 13x slower than the recorded GPU figure below.

### Where the VRAM claim's numbers come from

The claim previously reserved **5 GiB preferred / 2 GiB floor**, unmeasured, for
a resource holding **zero** device bytes — 5 GiB of a 16 GiB card reserved for
nothing. The rungs are now derived from the one real GPU measurement in this
document: **2,640 MiB observed for a medium Whisper model** (the faster-whisper
row below).

| Rung | Amount | Derivation |
|---|---|---|
| `large-v3` | 5,016 MiB | 1.9x the medium measurement, the GGML parameter-count ratio |
| `medium` | 2,640 MiB | **measured** — the observed resident figure recorded below |
| `small` | 924 MiB | 0.35x the medium measurement, the GGML parameter-count ratio |

The `medium` rung is a measurement. The other two are ratios from it, and are
labelled as such rather than presented as observations. They are still a
material improvement on the previous figures, which were neither.

**These are not measurements of a whisper.cpp CUDA build.** No such build exists
to measure: see the blocked work below.

### Blocked: the Linux CUDA artifact

Producing one needs two things this work could not obtain:

1. **A CUDA toolkit.** This host has no `nvcc` and no `/usr/local/cuda`.
   Installing one is a privileged host change, and privilege is only ever
   requested through `vrooli setup`.
2. **A place to publish the result.** Vrooli's managed-service acquisition
   consumes upstream release assets pinned by digest — whisper's own targets
   point at `github.com/ggml-org/whisper.cpp/releases`. There is no Vrooli
   release channel for a binary Vrooli builds itself, and creating one is a
   distribution decision, not an implementation detail.

Until both exist, a digest-pinned CUDA target would point at nothing. The CPU
target remains the declared Linux fallback, and the drift is reported rather
than hidden.

1. Produce and smoke a Linux CUDA artifact, or explicitly retain the CPU
   target with its measured latency budget.
2. Build and sign the macOS native server artifact; smoke it on macOS arm64.
3. Smoke the Windows CPU and cuBLAS targets and record target-specific results.
4. Extend the model catalog beyond the current medium artifact only when each
   selected model has same-corpus WER and latency evidence.

## 7. Validation matrix

| Engine / model | Corpus | Transcript | WER | Latency | Peak memory |
|---|---|---|---:|---:|---:|
| faster-whisper medium (GPU) | `quality_speech.wav`, reference `the quick brown fox jumps` | `the quick brown fox jumps.` | 0.00 | 0.29 s | 2,640 MiB GPU process observed |
| whisper.cpp v1.8.5 small (CPU) | same | `the quick brown fox jumps.` | 0.00 | 2.35 s | server request RSS 10.9 MiB; resident model not sampled |
| whisper.cpp v1.8.5 large-v3 (CPU) | same | `the quick brown fox jumps.` | 0.00 | 15.56 s | resident server RSS 3,498,344 KiB; 0 MiB GPU |
| whisper.cpp v1.9.2 medium (native managed-service CPU, 12 threads) | same | `the quick brown fox jumps.` | 0.00 | 5.3 s | clean acquired GGML model; resident model not sampled |
| whisper.cpp v1.9.2 medium (native managed-service CPU, 12 threads) | same | `the quick brown fox jumps.` | 0.00 | **3.85 s / 4.06 s** | 2026-08-21 re-measurement on the same host, two consecutive `POST /inference` calls; `nvidia-smi` shows the `whisper-server` pid holding **0 MiB** of device memory while the resource declares `cuda` |

Commands are reproducible from the recorded scratch directory:

```bash
/tmp/whisper-cpp-spike.WiV88I/build-cpu/bin/whisper-server -m /tmp/whisper-cpp-spike.WiV88I/src/models/ggml-large-v3.bin --port 18092 --host 127.0.0.1 --convert
curl -F 'file=@scenarios/audio-tools/api/internal/diagnostics/smokedata/quality_speech.wav' http://127.0.0.1:18092/inference
```

Pinned spike inputs: source tarball SHA-256
`cd702189cb5e608c8bc487f4b151db593c4455925b37cc06ef76b44861911db1`;
small model SHA-256 `1be3a9b2063867b937e64e2ec7483364a79917e157fa98c5d94b5c1fffea987b`;
large-v3 SHA-256 `64d182b440b98d5203c4f9bd541544d84c605196c4f7b845dfa11fb23594d1e2`.

## 8. Risks and decision

**Verdict: Linux native managed-service is live; cross-platform promotion is
not complete.** The clean install acquired the v1.9.2 server archive and
medium model without hand-staged files. The canonical `quality_speech.wav`
smoke through the unchanged `/asr` contract returned `the quick brown fox
jumps.` with one segment, and `/detect-language` returned English. This proves
the Linux CPU delivery and compatibility edge, not CUDA parity or macOS/Windows
readiness. Those targets remain explicitly recorded rather than inferred from
the Linux run.
