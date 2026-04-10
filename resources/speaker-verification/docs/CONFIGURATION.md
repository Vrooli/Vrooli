# Speaker Verification Configuration

## Environment Variables

All configuration is set via environment variables, with defaults in `config/defaults.sh`.

### Service

| Variable | Default | Description |
|----------|---------|-------------|
| `SPEAKER_VERIFICATION_PORT` | `8891` | HTTP API port |
| `SPEAKER_VERIFICATION_CUSTOM_PORT` | - | Override the default port |
| `SPEAKER_VERIFICATION_CONTAINER_NAME` | `speaker-verification` | Docker container name |

### Model and Device

| Variable | Default | Description |
|----------|---------|-------------|
| `SPEAKER_VERIFICATION_DEVICE` | `auto` | Compute device: `auto`, `cpu`, or `cuda` |
| `SPEAKER_VERIFICATION_MODEL` | `nvidia/speakerverification_en_titanet_large` | NeMo model identifier |
| `SPEAKER_VERIFICATION_DEFAULT_THRESHOLD` | `0.7` | Cosine similarity threshold for match (0.0-1.0) |

When `DEVICE=auto`, the service uses CUDA if available, otherwise falls back to CPU.

### Audio Constraints

| Variable | Default | Description |
|----------|---------|-------------|
| `SPEAKER_VERIFICATION_ENROLLMENT_MIN_SECONDS` | `3` | Minimum enrollment audio duration |
| `SPEAKER_VERIFICATION_VERIFY_MIN_SECONDS` | `1` | Minimum verification audio duration |
| `SPEAKER_VERIFICATION_SAMPLE_RATE` | `16000` | Target sample rate (Hz) |
| `SPEAKER_VERIFICATION_MAX_UPLOAD_MB` | `50` | Maximum upload file size (MB) |

### Data Directories

| Variable | Default | Description |
|----------|---------|-------------|
| `SPEAKER_VERIFICATION_DATA_DIR` | `~/.speaker-verification` | Root data directory |
| `SPEAKER_VERIFICATION_PROFILES_DIR` | `~/.speaker-verification/profiles` | Profile storage |
| `SPEAKER_VERIFICATION_CACHE_DIR` | `~/.speaker-verification/cache` | Model cache |
| `SPEAKER_VERIFICATION_LOG_DIR` | `~/.speaker-verification/logs` | Log files |

### Startup and Health

| Variable | Default | Description |
|----------|---------|-------------|
| `SPEAKER_VERIFICATION_STARTUP_MAX_WAIT` | `180` | Max seconds to wait for startup |
| `SPEAKER_VERIFICATION_STARTUP_WAIT_INTERVAL` | `5` | Seconds between health polls |
| `SPEAKER_VERIFICATION_INITIALIZATION_WAIT` | `10` | Initial wait after container start |
| `SPEAKER_VERIFICATION_HEALTH_CHECK_INTERVAL` | `5` | Health check polling interval |
| `SPEAKER_VERIFICATION_HEALTH_CHECK_MAX_ATTEMPTS` | `24` | Max health check attempts |
| `SPEAKER_VERIFICATION_API_TIMEOUT` | `30` | API request timeout (seconds) |

### Network

| Variable | Default | Description |
|----------|---------|-------------|
| `SPEAKER_VERIFICATION_NETWORK_NAME` | `speaker-verification-network` | Docker network name |

## Runtime Configuration

`config/runtime.json` controls startup behavior:

```json
{
  "startup_order": 570,
  "startup_timeout": 180,
  "startup_time_estimate": "30-120s",
  "dependencies": ["ffmpeg"],
  "recovery_attempts": 2,
  "priority": "medium"
}
```

## Threshold Tuning

The default threshold of `0.7` is conservative. Adjust based on your use case:

- **0.5-0.6**: Lenient, higher false acceptance rate
- **0.7**: Balanced default
- **0.8-0.9**: Strict, may reject legitimate speakers in noisy conditions

The threshold can be overridden per-request via the `threshold` parameter in the `/v1/verify` endpoint.

## GPU Support

GPU acceleration is auto-detected. Requirements:
- NVIDIA GPU with CUDA support
- `nvidia-smi` available on host
- Docker configured with NVIDIA runtime

When GPU is detected, the container starts with `--gpus all`.
