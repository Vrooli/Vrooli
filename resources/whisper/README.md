# Whisper Speech-to-Text Resource

Whisper is OpenAI's automatic speech recognition (ASR) system that provides robust transcription capabilities across 100+ languages.

## Features

- **Multi-language Support**: Transcribe audio in 100+ languages
- **Translation**: Translate audio from any language to English
- **High Accuracy**: State-of-the-art transcription accuracy
- **Multiple Formats**: Supports WAV, MP3, OGG, M4A, FLAC, AAC, WMA
- **Configurable Models**: From tiny (39MB) to large (1.5GB) models
- **GPU Acceleration**: Optional NVIDIA GPU support for faster processing

## Quick Start

```bash
# Install with default large model
./manage.sh --action install

# Install with smaller model (faster, less accurate)
./manage.sh --action install --model small

# Install with GPU support
./manage.sh --action install --gpu yes

# Check status
./manage.sh --action status

# Transcribe audio
./manage.sh --action transcribe --file audio.mp3
```

## Available Models

| Model | Size | Speed | Accuracy | Use Case |
|-------|------|-------|----------|----------|
| tiny | 39 MB | Fastest | Lower | Quick drafts, real-time |
| base | 74 MB | Very Fast | Good | General use, quick results |
| small | 244 MB | Fast | Better | Balanced performance |
| medium | 769 MB | Moderate | Very Good | Professional transcription |
| large | 1.5 GB | Slower | Best | Maximum accuracy (default) |
| large-v2 | 1.5 GB | Slower | Best | Latest v2 improvements |
| large-v3 | 1.5 GB | Slower | Best | Latest v3 improvements |

## API Usage

### Transcription Endpoint

```bash
# Basic transcription
curl -X POST "http://localhost:8090/asr?output=json" \
  -F "audio_file=@audio.mp3" \
  -F "task=transcribe"

# With language specification
curl -X POST "http://localhost:8090/asr?output=json" \
  -F "audio_file=@spanish.mp3" \
  -F "language=es" \
  -F "task=transcribe"

# Translation to English
curl -X POST "http://localhost:8090/asr?output=json" \
  -F "audio_file=@foreign.mp3" \
  -F "task=translate"
```

### Language Detection Endpoint

```bash
# Detect the language of an audio file
curl -X POST "http://localhost:8090/detect-language" \
  -F "audio_file=@audio.mp3"
```

### Response Formats

#### Transcription Response
```json
{
  "text": "Hello, this is a transcription of the audio file.",
  "segments": [
    {
      "id": 0,
      "seek": 0,
      "start": 0.0,
      "end": 3.5,
      "text": "Hello, this is a transcription",
      "tokens": [50365, 25674, 50563],
      "temperature": 0.0,
      "avg_logprob": -0.4147,
      "compression_ratio": 0.333,
      "no_speech_prob": 0.551
    }
  ],
  "language": "en"
}
```

#### Language Detection Response
```json
{
  "detected_language": "english",
  "language_code": "en",
  "confidence": 0.9987
}
```

## Technical Notes

### API Health Check

The Whisper service uses a 307 redirect pattern for its health check:

- **Root endpoint** (`/`) returns HTTP 307 redirect to `/docs`
- This is normal behavior and indicates the service is healthy
- The Vrooli resource provider accepts both 200 and 307 status codes
- Interactive API documentation is available at `/docs`
- OpenAPI specification is available at `/openapi.json`

### Model Management

The Whisper service currently loads a single model at startup:
- Model size is specified via the `ASR_MODEL` environment variable
- Changing models requires restarting the container
- The `/models` endpoint is not available in the current implementation

## Management Commands

```bash
# Start/stop/restart
./manage.sh --action start
./manage.sh --action stop
./manage.sh --action restart

# View logs
./manage.sh --action logs

# List available models
./manage.sh --action models

# Show detailed information
./manage.sh --action info

# Clean up old upload files (older than 1 day by default)
./manage.sh --action cleanup

# Uninstall
./manage.sh --action uninstall
```

## Configuration

### Environment Variables

- `WHISPER_CUSTOM_PORT`: Override default port (8090)
- `WHISPER_DEFAULT_MODEL`: Override default model size
- `WHISPER_IMAGE`: Custom Docker image for GPU
- `WHISPER_CPU_IMAGE`: Custom Docker image for CPU

### File Locations

- Models: `~/.whisper/models/`
- Uploads: `~/.whisper/uploads/`
- Config: Integrated with Vrooli resource configuration

## Troubleshooting

### Model Loading Takes Too Long

Large models can take 1-3 minutes to load initially. Once loaded, subsequent transcriptions are fast.

### Out of Memory Errors

Try using a smaller model:
```bash
./manage.sh --action install --model small --force yes
```

### GPU Not Detected

Ensure NVIDIA drivers and nvidia-docker are installed:
```bash
nvidia-smi  # Should show your GPU
```

### Port Already in Use

Either stop the conflicting service or use a custom port:
```bash
export WHISPER_CUSTOM_PORT=8091
./manage.sh --action install
```

### Audio Format Not Supported

Convert to a supported format using ffmpeg:
```bash
ffmpeg -i input.xyz -acodec pcm_s16le -ar 16000 output.wav
```

## Performance Tips

1. **Model Selection**: Start with `small` or `medium` for good balance
2. **GPU Acceleration**: Use `--gpu yes` for 5-10x speedup
3. **Batch Processing**: Process multiple files sequentially for efficiency
4. **Audio Quality**: Higher quality audio yields better transcriptions
5. **Language Hints**: Specify language when known for better accuracy

## Integration with Vrooli

Once installed, Whisper is automatically configured in Vrooli's resource registry and can be used by AI agents for speech-to-text tasks.

## Security Considerations

- Audio files are temporarily stored in `~/.whisper/uploads/`
- Use `./manage.sh --action cleanup` to remove old upload files
- Consider scheduling cleanup via cron for production deployments:
  ```bash
  # Add to crontab to run daily at 3 AM
  0 3 * * * /path/to/whisper/manage.sh --action cleanup
  ```
- Container runs in isolated environment
- No external network access required after model download

## Links

- [OpenAI Whisper](https://github.com/openai/whisper)
- [Whisper Models](https://huggingface.co/openai)
- [API Documentation](https://github.com/ahmetoner/whisper-asr-webservice)