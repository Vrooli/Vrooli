# Speaker Verification Resource

Local-first speaker verification using NVIDIA NeMo TitaNet. Provides enrolled-speaker verification and embedding management as a reusable Vrooli capability.

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

See `docs/CONFIGURATION.md` for the full list.

## Architecture

The resource runs as a Docker container with a FastAPI service backed by NeMo TitaNet for speaker embedding extraction. Profiles (embeddings + metadata) are stored on the host filesystem and mounted into the container.

## Documentation

- [API Reference](docs/API.md)
- [Configuration](docs/CONFIGURATION.md)
- [Troubleshooting](docs/TROUBLESHOOTING.md)
- [Testing](docs/TESTING.md)
