# Whisper Usage Examples

This directory contains examples demonstrating how to use Whisper for various audio transcription tasks.

## Basic Usage

### Simple Transcription

```bash
# Install and start Whisper
vrooli resource install whisper
vrooli resource start whisper

# Transcribe an audio file
curl -X POST "http://localhost:8090/asr?output=json" \
  -F "audio_file=@meeting.mp3" \
  -F "task=transcribe"
```

### Translation to English

```bash
# Translate foreign language audio to English
curl -X POST "http://localhost:8090/asr?output=json" \
  -F "audio_file=@spanish_audio.mp3" \
  -F "task=translate"
```

### Specify Language for Better Accuracy

```bash
# Specify the source language when known
curl -X POST "http://localhost:8090/asr?output=json" \
  -F "audio_file=@french_interview.mp3" \
  -F "language=fr" \
  -F "task=transcribe"
```

## Output Formats

### JSON Output (Default)
```bash
curl -X POST "http://localhost:8090/asr?output=json" \
  -F "audio_file=@audio.mp3"
```

### Plain Text
```bash
curl -X POST "http://localhost:8090/asr?output=text" \
  -F "audio_file=@audio.mp3"
```

### SRT Subtitles
```bash
curl -X POST "http://localhost:8090/asr?output=srt" \
  -F "audio_file=@video_audio.mp3"
```

## Model Selection

### Use Different Model Sizes

```bash
# The active model is a capacity rung, not an install flag: the broker asks
# Whisper to move between them, and an operator can ask directly.
vrooli resource run whisper -- capacity degrade --to small
vrooli resource run whisper -- capacity upshift --to large-v3

# See which rungs this resource declares
vrooli resource acceleration explain whisper
```

## GPU Acceleration

Acceleration is declared, not requested per install. Whisper declares
`backends: ["cuda", "cpu"]` with `require: preferred` in its `resource.json`,
so the control plane selects CUDA when the host can reach it and falls back to
the CPU when it cannot — and reports which one happened either way:

```bash
# Which backend did it ask for, which did it get, and why
vrooli resource acceleration explain whisper

# The same answer as a status field
vrooli resource status whisper --json | jq '{declared_mode, observed_mode, mode_drift}'
```

A resource running below its declared backend reports `mode_drift: true` with
`healthy: false` and `serving: true`. It is degraded, not down.

> **Linux note.** whisper.cpp v1.9.2 publishes no Linux CUDA release asset, so
> on Linux this resource currently selects the CPU target and reports
> `mode_drift: true`. That is accurate, not a fault: see
> [whisper-cpp-managed-service-assessment.md](whisper-cpp-managed-service-assessment.md).

## Integration Examples

### With curl and jq
```bash
# Extract just the transcribed text
TRANSCRIPTION=$(curl -s -X POST "http://localhost:8090/asr?output=json" \
  -F "audio_file=@audio.mp3" | jq -r '.text')

echo "Transcription: $TRANSCRIPTION"
```

### With Python
```python
import requests

# Transcribe audio file
with open('audio.mp3', 'rb') as f:
    response = requests.post(
        'http://localhost:8090/asr?output=json',
        files={'audio_file': f},
        data={'task': 'transcribe', 'language': 'en'}
    )

result = response.json()
print(f"Transcription: {result['text']}")
```

## Troubleshooting

### Check Service Status
```bash
vrooli resource status whisper
```

### View Logs
```bash
vrooli resource logs whisper
```

### Test with Sample Audio
```bash
# Download a sample audio file
wget https://www.soundjay.com/misc/sounds/bell-ringing-05.wav -O test.wav

# Test transcription
curl -X POST "http://localhost:8090/asr?output=text" \
  -F "audio_file=@test.wav"
```

## Performance Tips

1. **Model Selection**: Use `small` or `medium` for real-time needs, `large` for best accuracy
2. **Language Hints**: Specify language when known for better accuracy
3. **File Size**: Keep files under 25MB for best performance
4. **GPU**: Use GPU acceleration for 5-10x speedup on supported hardware