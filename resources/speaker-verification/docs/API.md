# Speaker Verification API Reference

Complete API documentation for the Speaker Verification service, a thin FastAPI
wrapper around SpeechBrain's ECAPA-TDNN speaker-embedding model.

## Model

| Property | Value |
|---|---|
| Backend | `speechbrain` |
| Model | `speechbrain/spkrec-ecapa-voxceleb` |
| Architecture | ECAPA-TDNN (Emphasized Channel Attention, Propagation and Aggregation in TDNN) |
| Embedding dimension | **192** |
| Input sample rate | **16000 Hz** (mono) |
| Reported EER | 0.8% on VoxCeleb1-test (cleaned) |
| Verification | Cosine similarity between the enrollment embedding and a test-clip embedding, compared against a caller-supplied threshold |
| Device | GPU (CUDA) when an NVIDIA GPU is detected; downgrades to CPU automatically otherwise. Reported by `/v1/info` (`device`, `torch_version`, `cuda_available`). |

Embeddings are L2-normalized before storage; the verification `score` is the
cosine similarity in `[-1, 1]` (in practice `[0, 1]` for normalized speech
embeddings). A clip is `matched` when `score >= threshold`.

## Base URL

```
http://localhost:11452
```

The host port is `11452` (in the AI-services `11xxx` range, deconflicted against
the [port registry](/home/matthalloran8/Vrooli/scripts/resources/port_registry.sh)).
The container listens on port `8000`. The base URL is exported to other
scenarios as `SPEAKER_VERIFICATION_URL`.

## Audio Input

Upload fields accept any common audio container (WebM/Opus, WAV, MP3, FLAC, raw
PCM, ...). The server decodes via `torchaudio`, falling back to `ffmpeg` for
container formats torchaudio cannot read directly, then resamples to 16 kHz mono
before embedding. The audio-tools browser embed sends WebM/Opus, which is
handled by the ffmpeg path.

## Endpoints

### Readiness

**GET** `/ready`

Liveness/readiness probe. Backs the compose healthcheck.

```json
{
  "status": "ok",
  "model_loaded": true,
  "profile_store_ok": true,
  "temp_dir_ok": true
}
```

`model_loaded` is `false` until the first embedding request triggers the
one-time model load/download; the service is still considered ready (`200`).

### Info

**GET** `/v1/info`

```json
{
  "backend": "speechbrain",
  "model": "speechbrain/spkrec-ecapa-voxceleb",
  "device": "cuda",
  "torch_version": "2.5.1+cu124",
  "cuda_available": true,
  "sample_rate": 16000,
  "version": "0.3.0",
  "embedding_dim": 192,
  "extraction_model": "speechbrain/sepformer-wsj02mix",
  "extraction_sample_rate": 8000,
  "extraction_match_threshold": 0.25
}
```

### List Profiles

**GET** `/v1/profiles`

```json
{
  "profiles": [
    {
      "id": "alice",
      "display_name": "Alice",
      "created_at": "2026-05-27T12:00:00+00:00",
      "updated_at": "2026-05-27T12:00:00+00:00",
      "model_name": "speechbrain/spkrec-ecapa-voxceleb",
      "embedding_dim": 192,
      "sample_rate": 16000,
      "enrollment_audio_seconds": 4.2,
      "notes": "primary operator"
    }
  ],
  "count": 1
}
```

The raw 192-dim embedding is never returned; only metadata is surfaced.

### Enroll a Profile

**POST** `/v1/profiles` (`multipart/form-data`)

| Field | Type | Notes |
|---|---|---|
| `profile_id` | string | May be empty — the server generates a hex id |
| `display_name` | string | Optional human label |
| `notes` | string | Optional free-form metadata |
| `audio` | file | Enrollment audio clip (any common format) |

Computes the embedding, persists the profile to the profile-store volume, and
returns:

```json
{
  "profile_id": "alice",
  "display_name": "Alice",
  "embedding_dim": 192,
  "sample_rate": 16000,
  "enrollment_audio_seconds": 4.2,
  "model_name": "speechbrain/spkrec-ecapa-voxceleb",
  "created_at": "2026-05-27T12:00:00+00:00"
}
```

Re-enrolling an existing `profile_id` overwrites the embedding but preserves the
original `created_at`.

### Verify

**POST** `/v1/verify` (`multipart/form-data`)

| Field | Type | Notes |
|---|---|---|
| `profile_id` | string | Must reference an enrolled profile (404 if missing) |
| `threshold` | float string | Cosine threshold; `matched = score >= threshold` |
| `audio` | file | Clip to verify |

```json
{
  "profile_id": "alice",
  "matched": true,
  "score": 0.71,
  "threshold": 0.25,
  "duration_ms": 134.2,
  "backend": "speechbrain",
  "model": "speechbrain/spkrec-ecapa-voxceleb",
  "audio_seconds": 2.5
}
```

### Extract (Target-Speaker Extraction)

**POST** `/v1/extract` (`multipart/form-data`)

| Field | Type | Notes |
|---|---|---|
| `profile_id` | string | Enrolled profile to isolate against (required) |
| `verify` | `"true"`/`"false"` | Reserved; the body IS the cleaned audio |
| `audio` | file | The mixture to isolate the enrolled speaker from |

Target-speaker **extraction** isolates the enrolled speaker's voice from a
mixture (a second person, background speech). It is **source separation**
(SepFormer splits the mixture into candidate voices) **+ ECAPA target-selection**
(the separated source whose embedding best matches the enrolled profile is
returned). See [`extraction.md`](extraction.md) for the model choice and tuning.

**Response** — the cleaned audio as raw bytes, not JSON:

```
HTTP/1.1 200 OK
Content-Type: application/octet-stream
X-Speaker-Score: 0.842137          # cosine similarity of the selected source
X-Speaker-Matched: true            # score >= SPEAKER_EXTRACTION_MATCH_THRESHOLD
X-Duration-Ms: 1832.4
X-Audio-Seconds: 2.500

<16 kHz mono signed-16 little-endian PCM bytes>
```

`404` if the profile is unknown; `400` on empty/undecodable audio.

The separation model, its sample rate, and the match threshold are tunable via
the `SPEAKER_EXTRACTION_MODEL`, `SPEAKER_EXTRACTION_SAMPLE_RATE`, and
`SPEAKER_EXTRACTION_MATCH_THRESHOLD` environment variables.

### Delete a Profile

**DELETE** `/v1/profiles/{profile_id}`

Returns `200 {}` on success, or `404 {"error": "profile not found"}` when the
profile does not exist.

## Persistence

- Profiles are stored one JSON file per `profile_id` under the
  `speaker_verification_profiles` volume (`/data/profiles` in-container).
- The HuggingFace model cache is mounted at `/data/model-cache` via the
  `speaker_verification_models` volume so weights are downloaded once.
