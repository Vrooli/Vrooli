# whisper.cpp managed-service assessment

Evaluation date: 2026-07-23. This is an evidence spike only; it does not change
the Whisper production archetype.

## 1. Current maturity

Whisper is M4 after this plan: manifest-authoritative compose lifecycle, pinned
images, readiness through activity-edge, Go live integration coverage, and no
resource Bash files. It remains below M5 because the cleanest non-Docker
archetype has not passed a CUDA/platform-delivery evaluation.

| Dimension | Score | Evidence |
|---|---:|---|
| Contract | 2 | `resource.json`, `/asr`, `/detect-language`, activity-edge readiness |
| Archetype | 1 | compose-service is reliable but Docker is avoidable in principle |
| Operator surface | 2 | Go resource CLI / shared control plane |
| Runtime and health | 2 | digest pins and readiness probe |
| Tests | 2 | `cli/live_test.go` plus audio-tools consumer path |
| Portability | 1 | Linux Docker works; native artifact delivery unproven |
| Legacy debt | 2 | no resource shell files |

## 2. Active contract and consumers

The active host contract is `POST /asr`, `GET /detect-language`, and `GET /`
via the activity-edge companion on port 8090. `audio-tools` is the material
consumer. The currently running production image reports `ASR_ENGINE=faster_whisper`
and `ASR_MODEL=medium`.

## 3. Target archetype

The candidate is `managed-service`: a supervised native `whisper-server` binary
with model artifacts managed like Vault artifacts. It could enable Metal on
macOS and remove the Docker requirement for batch STT. Compose-service remains
the selected production archetype for now: this operator has an NVIDIA GPU but
no CUDA toolkit (`nvcc` absent), so the required native CUDA build could not be
produced locally. A CPU-only build was possible but is not parity-equivalent.

## 4. Deployment profile

| Target | Candidate delivery | Requirement / result |
|---|---|---|
| linux-amd64 CUDA | signed `whisper-server` + GGML models | **blocked**: build with `-DGGML_CUDA=ON` failed because CUDA Toolkit/nvcc is absent |
| linux-amd64 CPU | built source artifact | works, but has no VRAM use and large-v3 is materially slower |
| macos-arm64 | signed Metal artifact | plausible target; not measured on this Linux operator |
| windows-amd64 | signed CUDA/CPU artifact | unmeasured |

Artifact-catalog sketch (successor migration):

```json
{
  "version": "v1.8.5",
  "artifacts": {
    "linux-amd64-cuda": {"url": "<release-or-CI-url>", "sha256": "<signed-release-sha256>", "binary_path": "whisper-server"},
    "linux-amd64-cpu": {"url": "<release-or-CI-url>", "sha256": "a8d427d72d5c6885a3b71a374b45f14bc6af715736ec24ba5f6a1b03d681fb83", "binary_path": "whisper-server"},
    "macos-arm64-metal": {"url": "<release-or-CI-url>", "sha256": "<signed-release-sha256>", "binary_path": "whisper-server"}
  }
}
```

## 5. Gap list

- No reproducible, pinned CUDA native artifact exists for the operator.
- Native server exposes `/inference`, not the `/asr` activity-edge contract;
  a compatibility adapter would be required.
- No managed model/artifact updater, signature verification policy, or Windows/
  macOS validation exists.
- Current production is faster-whisper **medium**; this spike measured GGML
  small and large-v3, not a same-rung engine parity comparison.

## 6. Migration plan

1. Publish signed, checksummed CUDA, CPU, and Metal server artifacts from one
   pinned whisper.cpp revision.
2. Build a managed-service wrapper and adapter preserving `/asr` and activity
   reporting; preserve current model cache during rollback.
3. Add GGML model catalog/install verification and profile-specific selection.
4. Run same-rung corpus parity (medium-equivalent and large-v3) on CUDA and
   Metal before a cutover; retain compose rollback until consumer smoke passes.

## 7. Validation matrix

| Engine / model | Corpus | Transcript | WER | Latency | Peak memory |
|---|---|---|---:|---:|---:|
| faster-whisper medium (GPU) | `quality_speech.wav`, reference `the quick brown fox jumps` | `the quick brown fox jumps.` | 0.00 | 0.29 s | 2,640 MiB GPU process observed |
| whisper.cpp v1.8.5 small (CPU) | same | `the quick brown fox jumps.` | 0.00 | 2.35 s | server request RSS 10.9 MiB; resident model not sampled |
| whisper.cpp v1.8.5 large-v3 (CPU) | same | `the quick brown fox jumps.` | 0.00 | 15.56 s | resident server RSS 3,498,344 KiB; 0 MiB GPU |

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

**Verdict: NO-GO for production migration now.** Accuracy on the one canonical
smoke utterance was equal, but it is insufficient corpus evidence; CPU native
large-v3 was 54x slower than the running GPU faster-whisper request, and native
CUDA could not build on this operator. Proceed only after artifact delivery and
same-rung CUDA/Metal corpus measurements meet an explicitly approved latency
and WER budget. Compose-service remains the honest production choice.
