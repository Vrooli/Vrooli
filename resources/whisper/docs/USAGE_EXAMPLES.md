# Whisper Usage Examples

This directory contains examples demonstrating how to use Whisper for various audio transcription tasks.

## Basic Usage

### Simple Transcription

```bash
# Install and start Whisper
./manage.sh --action install

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
# Install with small model for faster processing
./manage.sh --action install --model small

# Install with large model for best accuracy
./manage.sh --action install --model large
```

## GPU Acceleration

```bash
# Install with GPU support for faster processing
./manage.sh --action install --gpu yes --model large
```

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
./manage.sh --action status
```

### View Logs
```bash
./manage.sh --action logs
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