# AudioCraft Resource

Meta's comprehensive audio generation framework featuring MusicGen for text-to-music, AudioGen for sound effects, and EnCodec for neural audio compression.

## Overview

AudioCraft provides state-of-the-art AI models for complete audio generation:
- **MusicGen**: Generate music from text descriptions
- **AudioGen**: Create sound effects and ambient audio
- **EnCodec**: Neural audio codec for compression
- **Melody Conditioning**: Guide generation with existing melodies

## Features

- Text-to-music generation with multiple model sizes (300M-1.5B parameters)
- Sound effect generation from natural language
- Neural audio compression with 10:1 ratio
- Melody-guided music generation
- Support for various audio formats (WAV, MP3)
- REST API for easy integration
- GPU acceleration support

## Installation

```bash
# Install AudioCraft resource
vrooli resource audiocraft manage install

# Or using the resource CLI directly
resource-audiocraft manage install
```

## Usage

### Basic Commands

```bash
# Start the service
resource-audiocraft manage start

# Check status
resource-audiocraft status

# Generate music
curl -X POST http://localhost:7862/api/generate/music \
  -H "Content-Type: application/json" \
  -d '{"prompt": "relaxing jazz piano", "duration": 30}'

# Generate sound effects
curl -X POST http://localhost:7862/api/generate/sound \
  -H "Content-Type: application/json" \
  -d '{"prompt": "ocean waves crashing on beach"}'

# Stop service
resource-audiocraft manage stop
```

### API Endpoints

- `POST /api/generate/music` - Generate music from text
- `POST /api/generate/sound` - Generate sound effects
- `POST /api/generate/melody` - Melody-conditioned generation
- `POST /api/encode` - Compress audio with EnCodec
- `POST /api/decode` - Decompress audio
- `GET /api/models` - List available models
- `GET /health` - Service health check

### Configuration

Environment variables:
- `AUDIOCRAFT_PORT` - API port (default: 7862)
- `AUDIOCRAFT_MODEL_SIZE` - Model variant: small/medium/large (default: medium)
- `AUDIOCRAFT_USE_GPU` - Enable GPU acceleration (default: auto-detect)
- `AUDIOCRAFT_MAX_DURATION` - Maximum generation duration in seconds (default: 120)

## Examples

### Generate Background Music
```bash
# Generate 60 seconds of ambient music
curl -X POST http://localhost:7862/api/generate/music \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "cinematic orchestral background music, epic and emotional",
    "duration": 60,
    "model": "large"
  }'
```

### Create Sound Effects
```bash
# Generate thunder sound effect
curl -X POST http://localhost:7862/api/generate/sound \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "loud thunder strike with echo",
    "duration": 5
  }'
```

### Melody-Guided Generation
```bash
# Generate music based on uploaded melody
curl -X POST http://localhost:7862/api/generate/melody \
  -F "melody=@input.wav" \
  -F "prompt=transform into electronic dance music" \
  -F "duration=30"
```

## Integration with Scenarios

AudioCraft enhances various Vrooli scenarios:

- **game-dialog-generator**: Dynamic background music
- **video-processor**: Automated soundtracks
- **creative-assistant**: Audio for multimedia projects  
- **podcast-generator**: Intro/outro music and effects

## Technical Requirements

- **Memory**: 8GB minimum, 16GB recommended
- **Storage**: 20GB for models
- **GPU**: Optional but recommended (10x faster generation)
- **Python**: 3.9+ with PyTorch 2.1.0

## Troubleshooting

### Service won't start
- Check port 7862 is available: `netstat -tlnp | grep 7862`
- Verify Docker is running: `docker ps`
- Check logs: `resource-audiocraft logs`

### Slow generation
- Enable GPU if available: `AUDIOCRAFT_USE_GPU=true`
- Use smaller model: `AUDIOCRAFT_MODEL_SIZE=small`
- Reduce duration or batch size

### Out of memory
- Use smaller model variant
- Reduce max duration setting
- Enable memory-efficient mode

## License

- Code: MIT License
- Model Weights: CC-BY-NC 4.0 (non-commercial use)
- Commercial usage requires alternative models or licensing

## Support

For issues or questions:
- Check the [PRD](./PRD.md) for detailed specifications
- View logs: `resource-audiocraft logs`
- Test health: `resource-audiocraft test smoke`