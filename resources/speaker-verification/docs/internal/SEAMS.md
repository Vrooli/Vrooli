# Speaker Verification Resource — Internal Seams

This file enumerates the intentional testability and integration boundaries of
the speaker-verification resource. New seams should land here when added;
renames should update the row.

## HTTP Contract (cross-resource integration boundary)

| | |
|---|---|
| **Seam** | The HTTP API surface in `docker/server.py` — endpoint paths, multipart field names, and response JSON keys. |
| **Production wiring** | The FastAPI app in `docker/server.py`, served by uvicorn inside the resource image. |
| **External consumer** | `scenarios/audio-tools/api/internal/stt/pipeline/speaker_client.go` (`SpeakerClient`) calls `/ready`, `/v1/info`, `/v1/profiles` (GET/POST/DELETE), `/v1/verify`, `/v1/extract`. Its Go structs (`SpeakerResourceReady`, `SpeakerResourceInfo`, `SpeakerProfileList`, `SpeakerEnrollmentResponse`, `SpeakerVerifyResult`) pin the exact JSON keys. |
| **Why it exists** | This is the single coupling point between the resource and audio-tools. The server can be reimplemented (different model, different storage) so long as this contract holds byte-for-byte. Drift here silently breaks the audio-tools speaker-gate pipeline, so the contract is documented in `docs/API.md` and mirrored by the integration test in `test/integration-test.sh`. |

## Device selection (CPU/GPU)

| | |
|---|---|
| **Seam** | `SPEAKER_VERIFICATION_DEVICE` environment variable read by `server.py` at startup (`auto`/`cuda`/`cpu`; default resolves to `cuda` when a GPU is visible). |
| **Production wiring** | The image is built on the PyTorch CUDA runtime base (torch ships with a CUDA build). The manifest `gpu` block probes for NVIDIA and, when present, applies the GPU compose overlay (`runtime: nvidia` + device reservation) and sets `cuda`. `VROOLI_GPU=on/off/auto` overrides the probe. |
| **Test fake** | Set the env var directly; `server.py` downgrades `cuda` → `cpu` automatically when `torch.cuda.is_available()` is false, so the same image runs on CPU-only hosts. The resolved device is logged at boot and reported by `/v1/info` (`device`, `torch_version`, `cuda_available`). |
| **Why it exists** | Lets one image run on GPU (default when present) or CPU without code changes, and keeps device selection out of the Python logic that the audio-tools contract depends on. |

## Profile store

| | |
|---|---|
| **Seam** | The profile-store directory (`SPEAKER_VERIFICATION_PROFILE_DIR`, default `/data/profiles`). |
| **Production wiring** | One JSON file per `profile_id` storing the 192-dim embedding + metadata, persisted on the `speaker_verification_profiles` bind-mounted volume. |
| **Why it exists** | Persisting enrolled voiceprints across container restarts is required for verification to be useful. Keeping it a directory of per-profile JSON files (rather than an in-process map or external DB) makes enrollment idempotent and the store inspectable, with no extra runtime dependency. |
