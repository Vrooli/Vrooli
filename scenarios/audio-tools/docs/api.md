# Audio Tools API Documentation

## Overview
The Audio Tools API provides comprehensive audio processing capabilities through RESTful endpoints. All endpoints support both JSON and multipart/form-data for file uploads.

## Base URL
```
http://localhost:{API_PORT}/api
```

## Authentication
Currently, the API does not require authentication. Future versions may include API key authentication.

## Common Headers
```
Content-Type: application/json OR multipart/form-data
Accept: application/json
```

## Endpoints

### Health Check
**GET** `/health`

Check API health and service status.

**Response:**
```json
{
  "status": "healthy",
  "version": "1.0.0",
  "service": "audio-tools",
  "database": "connected",
  "storage": "filesystem|minio",
  "ffmpeg": "available",
  "timestamp": 1234567890
}
```

### Audio Editing Operations

#### Edit Audio
**POST** `/api/edit`

Apply editing operations to audio files.

**Request Body (JSON):**
```json
{
  "audio_file": "path/to/file.mp3",
  "operations": [
    {
      "type": "trim",
      "parameters": {
        "start_time": 10.5,
        "end_time": 30.0
      }
    },
    {
      "type": "volume",
      "parameters": {
        "level": 1.5
      }
    }
  ]
}
```

**Supported Operations:**
- `trim`: Cut audio to specified time range
  - `start_time` (float): Start time in seconds
  - `end_time` (float): End time in seconds
- `volume`: Adjust volume level
  - `level` (float): Volume multiplier (1.0 = no change)
- `fade_in`: Add fade-in effect
  - `duration` (float): Fade duration in seconds
- `fade_out`: Add fade-out effect
  - `duration` (float): Fade duration in seconds
- `merge`: Merge with another audio file
  - `second_file` (string): Path to second file
- `split`: Split audio at specified points
  - `points` (array): Split points in seconds

**Response:**
```json
{
  "success": true,
  "output_file": "path/to/edited.mp3",
  "duration": 19.5,
  "operations_applied": 2
}
```

### Format Conversion

#### Convert Audio Format
**POST** `/api/convert`

Convert audio between different formats.

**Request Body (JSON):**
```json
{
  "audio_file": "path/to/input.wav",
  "output_format": "mp3",
  "quality": "high",
  "bitrate": "320k"
}
```

**Multipart Form:**
- `audio`: Audio file (binary)
- `format`: Target format (mp3, wav, flac, aac, ogg)
- `quality`: Quality setting (low, medium, high)
- `bitrate`: Bitrate (optional, e.g., "128k", "320k")

**Supported Formats:**
- MP3 (`.mp3`)
- WAV (`.wav`)
- FLAC (`.flac`)
- AAC (`.aac`, `.m4a`)
- OGG Vorbis (`.ogg`)

**Response:**
```json
{
  "success": true,
  "output_file": "path/to/output.mp3",
  "format": "mp3",
  "size_bytes": 5242880,
  "duration": 180.5
}
```

### Audio Enhancement

#### Normalize Audio
**POST** `/api/normalize`

Normalize audio levels for consistent volume.

**Request Body (JSON):**
```json
{
  "audio_file": "path/to/audio.mp3",
  "target_level": -16.0,
  "prevent_clipping": true
}
```

**Response:**
```json
{
  "success": true,
  "output_file": "path/to/normalized.mp3",
  "peak_level": -0.5,
  "average_level": -16.0
}
```

#### Enhance Audio Quality
**POST** `/api/enhance`

Apply enhancement filters to improve audio quality.

**Request Body (JSON):**
```json
{
  "audio_file": "path/to/audio.mp3",
  "noise_reduction": 0.8,
  "auto_level": true,
  "remove_clicks": true
}
```

**Response:**
```json
{
  "success": true,
  "output_file": "path/to/enhanced.mp3",
  "improvements": [
    "noise_reduced",
    "levels_normalized",
    "clicks_removed"
  ]
}
```

### Audio Analysis

#### Extract Metadata
**GET** `/api/metadata?file={file_path}`
**POST** `/api/metadata` (with file upload)

Extract detailed metadata from audio files.

**Response:**
```json
{
  "duration": "3:45.123",
  "format": "mp3",
  "codec": "mp3",
  "sample_rate": "44100",
  "channels": 2,
  "bitrate": "320000",
  "size": 14680064,
  "tags": {
    "title": "Song Title",
    "artist": "Artist Name",
    "album": "Album Name",
    "date": "2024"
  }
}
```

#### Analyze Audio
**POST** `/api/analyze`

Perform comprehensive audio analysis.

**Request Body (JSON):**
```json
{
  "audio_file": "path/to/audio.mp3",
  "include_spectrum": true,
  "include_peaks": true,
  "include_loudness": true
}
```

**Response:**
```json
{
  "duration": 225.5,
  "format": "mp3",
  "sample_rate": 44100,
  "channels": 2,
  "peak_amplitude": -0.3,
  "rms_level": -18.5,
  "dynamic_range": 12.5,
  "frequency_spectrum": {
    "dominant_frequency": 440.0,
    "bass_energy": 0.35,
    "mid_energy": 0.45,
    "treble_energy": 0.20
  }
}
```

### Voice Processing

#### Voice Activity Detection (VAD)
**POST** `/api/vad`

Detect speech segments in audio.

**Request Body (JSON):**
```json
{
  "audio_file": "path/to/recording.mp3",
  "threshold": -40.0
}
```

**Multipart Form:**
- `audio`: Audio file (binary)
- `threshold`: Silence threshold in dB (default: -40)

**Response:**
```json
{
  "speech_segments": [
    {
      "start_time": 0.5,
      "end_time": 10.3,
      "duration": 9.8
    },
    {
      "start_time": 12.1,
      "end_time": 25.7,
      "duration": 13.6
    }
  ],
  "total_duration_seconds": 30.0,
  "speech_duration_seconds": 23.4,
  "silence_duration_seconds": 6.6,
  "speech_ratio": 0.78,
  "processing_time_ms": 150
}
```

#### Remove Silence
**POST** `/api/remove-silence`

Remove silent portions from audio.

**Request Body (JSON):**
```json
{
  "audio_file": "path/to/recording.mp3",
  "threshold": -40.0,
  "min_silence_duration": 0.5
}
```

**Response:**
```json
{
  "success": true,
  "output_file": "path/to/no_silence.mp3",
  "original_duration": 60.0,
  "new_duration": 45.5,
  "silence_removed": 14.5
}
```

### Speed and Pitch Modification

#### Change Speed
**POST** `/api/speed`

Modify playback speed without affecting pitch.

**Request Body (JSON):**
```json
{
  "audio_file": "path/to/audio.mp3",
  "speed_factor": 1.5,
  "preserve_pitch": true
}
```

**Response:**
```json
{
  "success": true,
  "output_file": "path/to/speed_modified.mp3",
  "original_duration": 60.0,
  "new_duration": 40.0,
  "speed_factor": 1.5
}
```

#### Change Pitch
**POST** `/api/pitch`

Modify pitch without affecting speed.

**Request Body (JSON):**
```json
{
  "audio_file": "path/to/audio.mp3",
  "pitch_shift": 2.0,
  "preserve_tempo": true
}
```

**Response:**
```json
{
  "success": true,
  "output_file": "path/to/pitch_modified.mp3",
  "pitch_shift_semitones": 2.0
}
```

### Equalization and Filtering

#### Apply Equalizer
**POST** `/api/equalizer`

Apply frequency equalization.

**Request Body (JSON):**
```json
{
  "audio_file": "path/to/audio.mp3",
  "bands": {
    "60": 3.0,
    "250": 0.0,
    "1000": -2.0,
    "4000": 1.5,
    "16000": 2.0
  }
}
```

**Response:**
```json
{
  "success": true,
  "output_file": "path/to/equalized.mp3",
  "bands_applied": 5
}
```

## Error Responses

All endpoints return consistent error responses:

```json
{
  "error": true,
  "message": "Description of the error",
  "code": "ERROR_CODE",
  "details": {
    "field": "additional context"
  }
}
```

**Common HTTP Status Codes:**
- `200 OK`: Successful operation
- `400 Bad Request`: Invalid request parameters
- `404 Not Found`: Resource not found
- `413 Payload Too Large`: File size exceeds limit
- `415 Unsupported Media Type`: Unsupported audio format
- `500 Internal Server Error`: Server processing error
- `503 Service Unavailable`: Service temporarily unavailable

## Rate Limiting

Currently, no rate limiting is enforced. Future versions may include:
- 60 requests per minute per IP
- 10 concurrent processing operations
- 500MB maximum file size

## Webhooks (Future)

Webhook support for async operations is planned:
```json
{
  "webhook_url": "https://your-server.com/webhook",
  "events": ["processing_complete", "error"]
}
```

## Examples

### cURL Examples

#### Upload and convert audio file:
```bash
curl -X POST http://localhost:8120/api/convert \
  -F "audio=@song.wav" \
  -F "format=mp3" \
  -F "quality=high"
```

#### Detect voice activity:
```bash
curl -X POST http://localhost:8120/api/vad \
  -H "Content-Type: application/json" \
  -d '{
    "audio_file": "/path/to/recording.mp3",
    "threshold": -35
  }'
```

#### Apply multiple edits:
```bash
curl -X POST http://localhost:8120/api/edit \
  -H "Content-Type: application/json" \
  -d '{
    "audio_file": "/path/to/audio.mp3",
    "operations": [
      {"type": "trim", "parameters": {"start_time": 5, "end_time": 60}},
      {"type": "normalize", "parameters": {}},
      {"type": "fade_out", "parameters": {"duration": 3}}
    ]
  }'
```

## SDK Examples (Future)

### JavaScript/TypeScript
```javascript
const audioTools = new AudioToolsSDK({
  baseUrl: 'http://localhost:8120',
  apiKey: 'your-api-key' // future
});

// Convert format
const result = await audioTools.convert({
  file: audioFile,
  format: 'mp3',
  quality: 'high'
});

// Detect voice activity
const vad = await audioTools.detectVoiceActivity({
  file: recordingFile,
  threshold: -40
});
```

### Python
```python
from audio_tools import AudioToolsClient

client = AudioToolsClient(base_url='http://localhost:8120')

# Remove silence
result = client.remove_silence(
    audio_file='recording.mp3',
    threshold=-40,
    min_silence_duration=0.5
)

# Enhance audio
enhanced = client.enhance(
    audio_file='noisy.mp3',
    noise_reduction=0.8,
    auto_level=True
)
```

## Best Practices

1. **File Management**: Clean up temporary files after processing
2. **Format Selection**: Use lossless formats (FLAC, WAV) for intermediate processing
3. **Batch Processing**: Use the CLI for batch operations on multiple files
4. **Error Handling**: Always check response status and handle errors gracefully
5. **Performance**: For large files, consider chunked uploads (future feature)
6. **Caching**: Cache processed results when possible to avoid reprocessing

## Changelog

### v1.0.0 (Current)
- Initial release with P0 requirements
- Voice Activity Detection (VAD) 
- Silence removal
- All basic editing operations
- Format conversion support
- Audio enhancement features

### Planned Features
- Audio transcription with Whisper integration
- Speaker diarization
- Real-time streaming support
- WebSocket API for live processing
- Batch API endpoints
- OAuth2 authentication
- Advanced AI-powered features