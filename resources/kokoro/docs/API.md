# Kokoro API Reference

Complete API documentation for the Kokoro text-to-speech synthesis service.

## Base URL

```
http://localhost:8880
```

## Endpoints

### List Available Voices

**GET** `/v1/audio/voices`

Returns a list of available voices for synthesis. This endpoint also serves as the health check.

**Example Request:**
```bash
curl http://localhost:8880/v1/audio/voices
```

**Response:**
```json
["af_heart", "af_bella", "af_nicole", "af_sarah", "af_sky", "am_adam", "am_michael", "bf_emma", "bf_isabella", "bm_george", "bm_lewis"]
```

### Speech Synthesis

**POST** `/v1/audio/speech`

Synthesize speech from text input. This endpoint is OpenAI-compatible.

**Request Body (JSON):**
- `model` (string, required): Model name (use `"kokoro"`)
- `input` (string, required): Text to synthesize
- `voice` (string, optional): Voice to use (default: `af_heart`)
- `response_format` (string, optional): Output format - `mp3`, `wav`, `opus`, `flac` (default: `mp3`)

**Example Request:**
```bash
curl -X POST "http://localhost:8880/v1/audio/speech" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "kokoro",
    "input": "Hello, this is a test of text-to-speech synthesis.",
    "voice": "af_heart",
    "response_format": "mp3"
  }' \
  --output output.mp3
```

**Response:**
- Content-Type: `audio/mpeg` (for mp3), `audio/wav`, `audio/opus`, or `audio/flac`
- Body: Raw audio binary data

### Health Check

Use the voices endpoint as a health check:

```bash
curl -f http://localhost:8880/v1/audio/voices
```

Returns HTTP 200 with voice list when healthy.

## Supported Output Formats

- **mp3** - MPEG Audio Layer 3 (compressed, good quality)
- **wav** - Waveform Audio (uncompressed, highest quality)
- **opus** - Opus Audio (efficient compression)
- **flac** - Free Lossless Audio Codec (lossless compression)

## Error Responses

**400 Bad Request:**
```json
{
  "detail": "Missing required field: input"
}
```

**422 Unprocessable Entity:**
```json
{
  "detail": [
    {
      "loc": ["body", "input"],
      "msg": "field required",
      "type": "value_error.missing"
    }
  ]
}
```

**500 Internal Server Error:**
```json
{
  "detail": "Synthesis failed"
}
```

## OpenAI Compatibility

The Kokoro API is designed to be compatible with the OpenAI TTS API. You can use OpenAI client libraries by pointing them to the Kokoro base URL:

### Python (OpenAI SDK)
```python
from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:8880/v1",
    api_key="not-needed"  # Kokoro doesn't require API keys
)

response = client.audio.speech.create(
    model="kokoro",
    voice="af_heart",
    input="Hello from Kokoro!"
)

response.stream_to_file("output.mp3")
```

### Node.js (OpenAI SDK)
```javascript
import OpenAI from 'openai';

const openai = new OpenAI({
  baseURL: 'http://localhost:8880/v1',
  apiKey: 'not-needed',
});

const mp3 = await openai.audio.speech.create({
  model: 'kokoro',
  voice: 'af_heart',
  input: 'Hello from Kokoro!',
});

const buffer = Buffer.from(await mp3.arrayBuffer());
await fs.promises.writeFile('output.mp3', buffer);
```

## Rate Limits

- Concurrent requests: 10
- Request timeout: 60 seconds
- No file size limit (text input only)

## Model Information

Kokoro uses a single 82M parameter model. There is no model selection -- all requests use the same model. The model is loaded once at container startup.
