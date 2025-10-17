# MusicGen Resource

AI-powered music generation using Meta's MusicGen model for creating original music and audio content.

## Overview

MusicGen is Meta's state-of-the-art AI model for generating high-quality music from text descriptions. This resource enables Vrooli scenarios to:
- Generate original music from text prompts
- Create background music for videos and presentations
- Produce sound effects and ambient audio
- Support creative AI workflows

## Features

- **Text-to-Music Generation**: Create music from natural language descriptions
- **Multiple Models**: Support for small, medium, and large MusicGen variants
- **REST API**: Simple HTTP interface for integration
- **Batch Processing**: Generate multiple tracks programmatically
- **Format Support**: Output in standard WAV format

## Installation

```bash
# Install MusicGen resource
vrooli resource musicgen install

# Or using the resource CLI directly
resource-musicgen install
```

## Usage

### Basic Commands

```bash
# Check status
resource-musicgen status

# Start service
resource-musicgen start

# Generate music
resource-musicgen generate "upbeat electronic dance music" 30

# List available models
resource-musicgen list-models

# Stop service (only if necessary)
resource-musicgen stop
```

### API Endpoints

- `GET /health` - Health check
- `POST /generate` - Generate music from prompt
- `GET /models` - List available models

### Generate Music via API

```bash
curl -X POST http://localhost:8765/generate \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "peaceful piano melody with rain sounds",
    "duration": 20,
    "model": "facebook/musicgen-melody"
  }'
```

## Integration with Scenarios

### Example: Background Music Generator

```json
{
  "name": "Background Music Generator",
  "resources": ["musicgen"],
  "workflow": {
    "steps": [
      {
        "type": "musicgen",
        "action": "generate",
        "params": {
          "prompt": "cinematic orchestral theme",
          "duration": 60
        }
      }
    ]
  }
}
```

### Example: Multi-Track Album Creator

```python
# inject/album_creator.py
import requests
import json

tracks = [
    {"title": "intro", "prompt": "ambient electronic intro", "duration": 30},
    {"title": "main", "prompt": "energetic synth wave", "duration": 180},
    {"title": "outro", "prompt": "calm piano outro", "duration": 45}
]

for track in tracks:
    response = requests.post('http://localhost:8765/generate', json={
        "prompt": track["prompt"],
        "duration": track["duration"]
    })
    # Save each track
    with open(f"/outputs/{track['title']}.wav", "wb") as f:
        f.write(base64.b64decode(response.json()["audio"]))
```

## Models

Available models with different capabilities:
- `facebook/musicgen-melody` - Best for melodic content
- `facebook/musicgen-small` - Fast generation, lower quality
- `facebook/musicgen-medium` - Balanced speed and quality
- `facebook/musicgen-large` - Highest quality, slower

## Resource Requirements

- **Memory**: 8GB minimum, 16GB recommended
- **Storage**: 10GB for models
- **GPU**: Optional but recommended for faster generation
- **CPU**: Multi-core processor recommended

## Troubleshooting

### Model Download Issues
Models are downloaded on first use. Ensure stable internet connection.

### Memory Errors
Reduce batch size or use smaller models if encountering OOM errors.

### Slow Generation
- Use GPU acceleration if available
- Consider using smaller models for faster iteration
- Preload models to avoid startup delay

## Examples

See `/resources/musicgen/examples/` for:
- `generate_samples.sh` - Basic generation examples
- `batch_processing.py` - Batch music creation
- `style_transfer.py` - Musical style experiments

## Best Practices

1. **Prompt Engineering**: Be specific about genre, mood, instruments
2. **Duration**: Start with shorter clips (10-30s) for testing
3. **Model Selection**: Use appropriate model for your use case
4. **Caching**: Reuse generated audio when possible
5. **Resource Management**: Stop service when not in use for extended periods

## Integration Points

- **ComfyUI**: Generate music for video projects
- **FFmpeg**: Mix and process generated audio
- **n8n/Huginn**: Automate music generation workflows
- **OBS Studio**: Stream with AI-generated background music

## Security Considerations

- API runs on localhost only by default
- No authentication required for local access
- Generated content is stored locally
- Consider copyright implications of generated music

## Future Enhancements

- [ ] MIDI export support
- [ ] Real-time generation streaming
- [ ] Custom model fine-tuning
- [ ] Audio style transfer
- [ ] Multi-track composition

## Related Resources

- [AudioCraft Documentation](https://github.com/facebookresearch/audiocraft)
- [MusicGen Paper](https://arxiv.org/abs/2306.05284)
- [Prompt Engineering Guide](https://github.com/facebookresearch/audiocraft/blob/main/docs/MUSICGEN.md)