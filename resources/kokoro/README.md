# Kokoro Text-to-Speech Resource

Kokoro is a high-quality text-to-speech synthesis service using the Kokoro 82M model, exposed via an OpenAI-compatible API.

## Features

- **High-Quality Synthesis**: Natural-sounding speech from the Kokoro 82M model
- **Multiple Voices**: Built-in voice selection for different use cases
- **OpenAI-Compatible API**: Drop-in replacement for OpenAI TTS endpoints
- **Multiple Output Formats**: MP3, WAV, Opus, FLAC
- **GPU Acceleration**: Optional NVIDIA GPU support for faster synthesis
- **Lightweight**: Single 82M parameter model, no model selection needed

## Quick Start

```bash
# Install with default settings
resource-kokoro manage install

# Install with GPU support
resource-kokoro manage install --gpu yes

# Check status
resource-kokoro status

# Synthesize text
resource-kokoro content synthesize --text "Hello, world!"

# List available voices
resource-kokoro content voices
```

## API Usage

### Speech Synthesis (OpenAI-Compatible)

```bash
# Basic synthesis
curl -X POST "http://localhost:8880/v1/audio/speech" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "kokoro",
    "input": "Hello, this is Kokoro text-to-speech.",
    "voice": "af_heart",
    "response_format": "mp3"
  }' \
  --output speech.mp3

# With different voice and format
curl -X POST "http://localhost:8880/v1/audio/speech" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "kokoro",
    "input": "Testing different voices.",
    "voice": "af_bella",
    "response_format": "wav"
  }' \
  --output speech.wav
```

### List Available Voices

```bash
curl http://localhost:8880/v1/audio/voices
```

## Output Formats

| Format | Description | Use Case |
|--------|-------------|----------|
| mp3 | Compressed audio | General use, streaming |
| wav | Uncompressed audio | High quality, post-processing |
| opus | Efficient compression | Low-bandwidth streaming |
| flac | Lossless compression | Archival, highest quality |

## GPU vs CPU

| Mode | Image | Performance | Memory |
|------|-------|-------------|--------|
| GPU | `ghcr.io/remsky/kokoro-fastapi-gpu:latest` | Fast | 8-16 GB |
| CPU | `ghcr.io/remsky/kokoro-fastapi-cpu:latest` | Moderate | 4-8 GB |

## Management Commands

```bash
# Start/stop/restart
resource-kokoro manage start
resource-kokoro manage stop
resource-kokoro manage restart

# View logs
resource-kokoro logs

# Show detailed status
resource-kokoro status

# Uninstall
resource-kokoro manage uninstall
```

## Configuration

### Environment Variables

- `KOKORO_CUSTOM_PORT`: Override default port (8880)
- `KOKORO_DEFAULT_VOICE`: Override default voice (af_heart)
- `KOKORO_IMAGE`: Custom Docker image for GPU
- `KOKORO_CPU_IMAGE`: Custom Docker image for CPU

### File Locations

- Voice data: `~/.kokoro/voices/`
- Config: Integrated with Vrooli resource configuration

## Troubleshooting

### Service Takes Long to Start

The Kokoro model takes 20-45 seconds to load on first start. Once loaded, synthesis is fast.

### Out of Memory Errors

Try using the CPU image which has lower memory requirements:
```bash
export KOKORO_GPU_ENABLED=no
resource-kokoro manage install
```

### GPU Not Detected

Ensure NVIDIA drivers and nvidia-docker are installed:
```bash
nvidia-smi  # Should show your GPU
```

### Port Already in Use

Either stop the conflicting service or use a custom port:
```bash
export KOKORO_CUSTOM_PORT=8881
resource-kokoro manage install
```

## Integration with Vrooli

Once installed, Kokoro is automatically configured in Vrooli's resource registry and can be used by AI agents for text-to-speech tasks. Pairs well with Whisper for a complete voice pipeline (STT + TTS).

## Links

- [Kokoro-FastAPI](https://github.com/remsky/Kokoro-FastAPI)
- [Kokoro Model](https://huggingface.co/hexgrad/Kokoro-82M)
