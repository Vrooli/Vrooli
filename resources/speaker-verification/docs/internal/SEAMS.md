# Speaker Verification Resource — Internal Seams

This file enumerates the intentional testability and integration boundaries of
the speaker-verification resource. New seams should land here when added;
renames should update the row.

## HTTP Contract (cross-resource integration boundary)

| | |
|---|---|
| **Seam** | The HTTP API surface in `server/server.py` — endpoint paths, multipart field names, and response JSON keys. |
| **Production wiring** | The FastAPI app in `server/server.py`, served by the composed native Python runtime. |
| **External consumer** | `scenarios/audio-tools/api/internal/stt/pipeline/speaker_client.go` (`SpeakerClient`) calls `/ready`, `/v1/info`, `/v1/profiles` (GET list/detail/clips, POST append, DELETE profile + clip), `/v1/verify`, `/v1/extract`. Its Go structs (`SpeakerResourceReady`, `SpeakerResourceInfo`, `SpeakerProfileList`, `SpeakerProfileDetail`, `SpeakerProfileClipList`, `SpeakerEnrollmentResponse`, `SpeakerVerifyResult`, `SpeakerClipDeleteResult`) pin the exact JSON keys. |
| **Why it exists** | This is the single coupling point between the resource and audio-tools. The server can be reimplemented (different model, different storage) so long as this contract holds byte-for-byte. Drift here silently breaks the audio-tools speaker-gate pipeline, so the contract is documented in `docs/API.md` and mirrored by the live-gated Go test in `cli/live_test.go`. |

## Voice-activity detector (embedding pre-trim)

| | |
|---|---|
| **Seam** | `VoiceActivityDetector` in `server/vad.py` — `trim(waveform, sr) -> (voiced_waveform, voiced_seconds)`, selected by `SPEAKER_VAD` (`silero`\|`energy`\|`none`). |
| **Production wiring** | `_voiced_embedding` routes both enrollment and verification through the same VAD trim before ECAPA. Silero is the default and is fixed on CPU; failure to load it falls back to `EnergyVAD`. ECAPA and SepFormer retain the configured serving device. |
| **Test fake** | Pure helpers are tested on lists of frame RMS; `EnergyVAD.trim` is exercised on synthetic torch tensors (skipped when torch is absent). `SPEAKER_VAD=none` is a passthrough `NoOpVAD`. Tests live in `tests/test_vad.py`. |
| **Why it exists** | CPU-only Silero removes the CPU-input/GPU-model mismatch class while keeping the small VAD real-time for clip-sized audio. |

## Score aggregation (multi-clip maximum)

| | |
|---|---|
| **Seam** | `_score_profile` in `server.py` scores a profile as the maximum cosine similarity over its enrolled clip embeddings. |
| **Production wiring** | A profile holds N clip embeddings. The server returns the best matching clip and uses max aggregation without an environment switch. The aggregation is unit-tested in `tests/test_profiles.py`. |
| **Why it exists** | Multiple recording conditions strengthen one identity without pulling a centroid toward a neutral voiceprint. |

## Device selection (CPU/GPU)

| | |
|---|---|
| **Seam** | `SPEAKER_VERIFICATION_DEVICE` environment variable read by `server.py` at startup (`auto`/`cuda`/`cpu`; default resolves to `cuda` when a GPU is visible). |
| **Production wiring** | The manifest's GPU predicate selects the CUDA wheel lock and sets `SPEAKER_VERIFICATION_DEVICE=cuda`; the CPU target selects the CPU wheel lock. |
| **Test fake** | Set the env var directly; `server.py` downgrades `cuda` → `cpu` automatically when `torch.cuda.is_available()` is false, so the same image runs on CPU-only hosts. The resolved device is logged at boot and reported by `/v1/info` (`device`, `torch_version`, `cuda_available`). |
| **Why it exists** | Lets one image run on GPU (default when present) or CPU without code changes, and keeps device selection out of the Python logic that the audio-tools contract depends on. |

## Profile store

| | |
|---|---|
| **Seam** | The profile-store directory (`SPEAKER_VERIFICATION_PROFILE_DIR`, default `/data/profiles`). |
| **Production wiring** | One JSON file per `profile_id` storing the 192-dim embedding + metadata, persisted under the declared resource data directory. |
| **Why it exists** | Persisting enrolled voiceprints across container restarts is required for verification to be useful. Keeping it a directory of per-profile JSON files (rather than an in-process map or external DB) makes enrollment idempotent and the store inspectable, with no extra runtime dependency. |
