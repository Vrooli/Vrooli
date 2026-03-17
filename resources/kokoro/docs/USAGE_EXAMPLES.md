# Kokoro Usage Examples

This document demonstrates how to use Kokoro for various text-to-speech synthesis tasks.

## Basic Usage

### Simple Text Synthesis

```bash
# Install and start Kokoro
resource-kokoro manage install

# Synthesize text to MP3
curl -X POST "http://localhost:8880/v1/audio/speech" \
  -H "Content-Type: application/json" \
  -d '{"model":"kokoro","input":"Hello, world!","voice":"af_heart"}' \
  --output hello.mp3
```

### Different Voices

```bash
# Use a different voice
curl -X POST "http://localhost:8880/v1/audio/speech" \
  -H "Content-Type: application/json" \
  -d '{"model":"kokoro","input":"This is a different voice.","voice":"am_adam"}' \
  --output different_voice.mp3
```

### List All Voices

```bash
# Get available voices
curl http://localhost:8880/v1/audio/voices | jq .
```

## Output Formats

### MP3 Output (Default)
```bash
curl -X POST "http://localhost:8880/v1/audio/speech" \
  -H "Content-Type: application/json" \
  -d '{"model":"kokoro","input":"MP3 format output.","voice":"af_heart","response_format":"mp3"}' \
  --output output.mp3
```

### WAV Output
```bash
curl -X POST "http://localhost:8880/v1/audio/speech" \
  -H "Content-Type: application/json" \
  -d '{"model":"kokoro","input":"WAV format output.","voice":"af_heart","response_format":"wav"}' \
  --output output.wav
```

### Opus Output
```bash
curl -X POST "http://localhost:8880/v1/audio/speech" \
  -H "Content-Type: application/json" \
  -d '{"model":"kokoro","input":"Opus format output.","voice":"af_heart","response_format":"opus"}' \
  --output output.opus
```

### FLAC Output
```bash
curl -X POST "http://localhost:8880/v1/audio/speech" \
  -H "Content-Type: application/json" \
  -d '{"model":"kokoro","input":"FLAC format output.","voice":"af_heart","response_format":"flac"}' \
  --output output.flac
```

## Integration Examples

### With curl and jq

```bash
# List voices and pick the first one
VOICE=$(curl -s http://localhost:8880/v1/audio/voices | jq -r '.[0]')
echo "Using voice: $VOICE"

# Synthesize with selected voice
curl -X POST "http://localhost:8880/v1/audio/speech" \
  -H "Content-Type: application/json" \
  -d "{\"model\":\"kokoro\",\"input\":\"Hello from $VOICE\",\"voice\":\"$VOICE\"}" \
  --output "output_${VOICE}.mp3"
```

### With Python (OpenAI SDK)

```python
from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:8880/v1",
    api_key="not-needed"
)

# Synthesize speech
response = client.audio.speech.create(
    model="kokoro",
    voice="af_heart",
    input="Hello from Python!"
)

response.stream_to_file("python_output.mp3")
print("Saved to python_output.mp3")
```

### With Python (requests)

```python
import requests

response = requests.post(
    "http://localhost:8880/v1/audio/speech",
    json={
        "model": "kokoro",
        "input": "Hello from Python requests!",
        "voice": "af_heart",
        "response_format": "mp3"
    }
)

with open("output.mp3", "wb") as f:
    f.write(response.content)

print(f"Saved {len(response.content)} bytes to output.mp3")
```

### Batch Processing

```bash
#!/bin/bash
# Synthesize multiple texts

texts=(
    "Welcome to our service."
    "Please hold while we connect you."
    "Thank you for your patience."
    "Goodbye and have a great day."
)

for i in "${!texts[@]}"; do
    curl -s -X POST "http://localhost:8880/v1/audio/speech" \
      -H "Content-Type: application/json" \
      -d "{\"model\":\"kokoro\",\"input\":\"${texts[$i]}\",\"voice\":\"af_heart\"}" \
      --output "segment_${i}.mp3"
    echo "Generated segment_${i}.mp3: ${texts[$i]}"
done
```

## Troubleshooting

### Check Service Status
```bash
resource-kokoro status
```

### View Logs
```bash
resource-kokoro logs
```

### Test Connectivity
```bash
# Simple health check
curl -f http://localhost:8880/v1/audio/voices && echo "Kokoro is running"
```

## Performance Tips

1. **GPU Acceleration**: Use GPU mode for faster synthesis on supported hardware
2. **Output Format**: MP3 is fastest for generation; WAV for highest quality
3. **Batch Processing**: Process multiple texts sequentially for efficiency
4. **Text Length**: Shorter texts synthesize faster; split long texts into segments
