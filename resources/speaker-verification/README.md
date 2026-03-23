# Speaker Verification Resource

Local-first speaker verification and target speaker extraction using NVIDIA NeMo TitaNet and SpeechBrain SepFormer. Provides enrolled-speaker verification, embedding management, and target speaker extraction as a reusable Vrooli capability.

## Quick Start

```bash
# Install and start
resource-speaker-verification manage install

# Check status
resource-speaker-verification status

# Enroll a speaker profile
resource-speaker-verification content enroll --profile default --file enrollment.wav

# Verify a speaker
resource-speaker-verification content verify --profile default --file clip.wav

# List profiles
resource-speaker-verification content profiles list
```

## Core Commands

```bash
# Lifecycle
resource-speaker-verification manage install
resource-speaker-verification manage start
resource-speaker-verification manage stop
resource-speaker-verification manage restart
resource-speaker-verification manage uninstall

# Information
resource-speaker-verification status
resource-speaker-verification logs
resource-speaker-verification help

# Content/Domain
resource-speaker-verification content enroll --profile <id> --file <audio.wav>
resource-speaker-verification content verify --profile <id> --file <audio.wav> [--threshold 0.7]
resource-speaker-verification content profiles list
resource-speaker-verification content profiles get --profile <id>
resource-speaker-verification content profiles remove --profile <id>
resource-speaker-verification content info

# Testing
resource-speaker-verification test smoke
resource-speaker-verification test integration
resource-speaker-verification test unit
resource-speaker-verification test all
```

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Liveness check |
| GET | `/ready` | Readiness check (model loaded) |
| GET | `/v1/info` | Backend/model info |
| POST | `/v1/profiles` | Enroll a speaker profile |
| GET | `/v1/profiles` | List all profiles |
| GET | `/v1/profiles/{id}` | Get one profile |
| DELETE | `/v1/profiles/{id}` | Remove a profile |
| POST | `/v1/verify` | Verify audio against profile |
| POST | `/v1/extract` | Extract target speaker from audio mixture |
| POST | `/v1/embeddings` | Extract raw embeddings (debug) |

Default API URL: `http://localhost:8891`

## Configuration

Key environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `SPEAKER_VERIFICATION_PORT` | `8891` | API port |
| `SPEAKER_VERIFICATION_DEVICE` | `auto` | Compute device (auto/cpu/cuda) |
| `SPEAKER_VERIFICATION_MODEL` | `nvidia/speakerverification_en_titanet_large` | NeMo model |
| `SPEAKER_VERIFICATION_DEFAULT_THRESHOLD` | `0.7` | Cosine similarity threshold |
| `SPEAKER_VERIFICATION_TSE_ENABLED` | `true` | Enable target speaker extraction |
| `SPEAKER_VERIFICATION_TSE_MODEL` | `speechbrain/sepformer-wsj02mix` | SpeechBrain separation model |
| `SPEAKER_VERIFICATION_TSE_MIN_OUTPUT_RMS` | `1e-4` | Silence detection threshold for separated sources |
| `SPEAKER_VERIFICATION_TSE_MIN_SPEAKER_SCORE` | `0.25` | Minimum cosine similarity to accept a separated source |

See `docs/CONFIGURATION.md` for the full list.

## Architecture

The resource runs as a Docker container with a FastAPI service backed by NeMo TitaNet for speaker embedding extraction and SpeechBrain SepFormer for target speaker extraction. Profiles (embeddings + metadata) are stored on the host filesystem and mounted into the container.

## Target Speaker Extraction (TSE)

TSE isolates an enrolled speaker's voice from audio containing multiple speakers or background audio (TV, music). When enabled, the `/v1/extract` endpoint:

1. **Separates** the audio mixture into individual source signals using SpeechBrain SepFormer (blind source separation)
2. **Identifies** which separated source matches the enrolled speaker by extracting TitaNet embeddings and comparing via cosine similarity
3. **Returns** the best-matching source as 16kHz mono WAV, optionally with speaker verification score

This approach reuses existing TitaNet embeddings — no enrollment changes are needed.

### `/v1/extract` Endpoint

**Request:** Multipart form-data with `audio` (file), `profile_id` (string), `verify` (bool, default true).

**Response (200):** Raw WAV audio bytes with metadata headers:
- `X-Speaker-Score` — verification score of extracted audio against profile
- `X-Speaker-Matched` — whether the score exceeds the threshold
- `X-Duration-Ms` — server-side processing time
- `X-Audio-Seconds` — duration of the extracted audio

**Error codes:** 404 (profile not found), 400 (audio too short/invalid), 503 (TSE model not loaded).

### Performance

- SepFormer adds ~1–3s per segment on CPU, ~0.3–1s on GPU
- Combined with TitaNet embedding extraction: ~2–5s total on CPU
- Additional memory: ~200–500MB for the SepFormer model
- The readiness endpoint (`/ready`) includes `tse_model_loaded` status

## Documentation

- [API Reference](docs/API.md)
- [Configuration](docs/CONFIGURATION.md)
- [Troubleshooting](docs/TROUBLESHOOTING.md)
- [Testing](docs/TESTING.md)
