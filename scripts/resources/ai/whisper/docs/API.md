# Whisper API Reference

Complete API documentation for the Whisper speech-to-text service.

## Base URL

```
http://localhost:8090
```

## Endpoints

### Health Check

**GET** `/`

Check if the Whisper service is running and healthy.

**Note:** The root endpoint returns a 307 redirect to `/docs` when healthy.

**Response:**
- Status: 307 Temporary Redirect
- Location header: `/docs`

For API documentation, visit `/docs` for the interactive Swagger UI or `/openapi.json` for the OpenAPI specification.

### Transcribe Audio

**POST** `/asr`

Transcribe audio file to text.

**Parameters:**
- `audio_file` (file, required): Audio file to transcribe
- `task` (string, optional): Task type - `transcribe` or `translate` (default: `transcribe`)
- `language` (string, optional): Language code (e.g., `en`, `es`, `fr`) (default: auto-detect)
- `output` (string, optional): Output format - `json`, `text`, `srt`, `vtt` (default: `json`)

**Example Request:**
```bash
curl -X POST "http://localhost:8090/asr?output=json" \
  -F "audio_file=@audio.mp3" \
  -F "task=transcribe" \
  -F "language=en"
```

**Response:**
```json
{
  "text": "Hello, this is a transcription of the audio file.",
  "segments": [
    {
      "start": 0.0,
      "end": 3.5,
      "text": "Hello, this is a transcription"
    },
    {
      "start": 3.5,
      "end": 5.2,
      "text": "of the audio file."
    }
  ],
  "language": "en"
}
```

### Translate Audio

**POST** `/asr`

Translate audio from any language to English.

**Parameters:**
- `audio_file` (file, required): Audio file to translate
- `task` (string, required): Must be set to `translate`
- `output` (string, optional): Output format (default: `json`)

**Example Request:**
```bash
curl -X POST "http://localhost:8090/asr?output=json" \
  -F "audio_file=@spanish.mp3" \
  -F "task=translate"
```

### Detect Language

**POST** `/detect-language`

Detect the language of an audio file without transcribing the full content.

**Parameters:**
- `audio_file` (file, required): Audio file to analyze

**Example Request:**
```bash
curl -X POST "http://localhost:8090/detect-language" \
  -F "audio_file=@audio.mp3"
```

**Response:**
```json
{
  "detected_language": "english",
  "language_code": "en",
  "confidence": 0.95
}
```

## Supported Audio Formats

- WAV
- MP3
- OGG
- M4A
- FLAC
- AAC
- WMA

## Response Formats

### JSON (default)
Complete transcription with timestamps and metadata.

### Text
Plain text transcription only.

### SRT
SubRip subtitle format with timestamps.

### VTT
WebVTT subtitle format.

## Error Responses

**400 Bad Request:**
```json
{
  "error": "Missing audio file"
}
```

**413 Payload Too Large:**
```json
{
  "error": "File too large. Maximum size: 25MB"
}
```

**415 Unsupported Media Type:**
```json
{
  "error": "Unsupported audio format"
}
```

**500 Internal Server Error:**
```json
{
  "error": "Transcription failed"
}
```

## Rate Limits

- Maximum file size: 25MB
- Concurrent requests: 5
- Request timeout: 300 seconds

## Language Codes

Common language codes supported:
- `en` - English
- `es` - Spanish  
- `fr` - French
- `de` - German
- `it` - Italian
- `pt` - Portuguese
- `ru` - Russian
- `ja` - Japanese
- `ko` - Korean
- `zh` - Chinese

For a complete list, see the [OpenAI Whisper documentation](https://github.com/openai/whisper#available-models-and-languages).