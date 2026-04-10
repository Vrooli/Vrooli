# Speaker Verification API Reference

Base URL: `http://localhost:8891`

## Health Endpoints

### GET /health

Liveness check. Returns 200 if the process is running.

**Response:**
```json
{
  "status": "ok",
  "version": "1.0.0"
}
```

### GET /ready

Readiness check. Confirms model is loaded and profile store is accessible.

**Response:**
```json
{
  "status": "ready",
  "model_loaded": true,
  "profile_store_ok": true,
  "temp_dir_ok": true
}
```

## Info

### GET /v1/info

Backend, model, and device information.

**Response:**
```json
{
  "backend": "nemo-titanet",
  "model": "nvidia/speakerverification_en_titanet_large",
  "device": "cpu",
  "sample_rate": 16000,
  "version": "1.0.0",
  "embedding_dim": 192
}
```

## Profile Management

### POST /v1/profiles

Create or replace a speaker profile from enrollment audio.

**Request:** `multipart/form-data`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `audio` | file | yes | Audio file (WAV, MP3, etc.) |
| `profile_id` | string | yes | Unique profile identifier |
| `display_name` | string | no | Human-readable name |
| `notes` | string | no | Optional notes |

**Response:**
```json
{
  "profile_id": "default",
  "display_name": "Default Speaker",
  "embedding_dim": 192,
  "sample_rate": 16000,
  "enrollment_audio_seconds": 5.2,
  "model_name": "nvidia/speakerverification_en_titanet_large",
  "created_at": "2026-03-22T12:00:00+00:00"
}
```

**Errors:**
- `400`: Audio too short, silence, or unreadable
- `422`: Missing required fields

### GET /v1/profiles

List all stored profiles.

**Response:**
```json
{
  "profiles": [
    {
      "id": "default",
      "display_name": "Default Speaker",
      "created_at": "2026-03-22T12:00:00+00:00",
      "updated_at": "2026-03-22T12:00:00+00:00",
      "model_name": "nvidia/speakerverification_en_titanet_large",
      "embedding_dim": 192,
      "sample_rate": 16000,
      "enrollment_audio_seconds": 5.2,
      "notes": ""
    }
  ],
  "count": 1
}
```

### GET /v1/profiles/{profile_id}

Get a single profile's metadata.

**Response:** Same schema as individual profile in the list response.

**Errors:**
- `404`: Profile not found

### DELETE /v1/profiles/{profile_id}

Remove a profile and its stored embedding.

**Response:**
```json
{
  "status": "deleted",
  "profile_id": "default"
}
```

**Errors:**
- `404`: Profile not found

## Verification

### POST /v1/verify

Compare verification audio against a stored profile.

**Request:** `multipart/form-data`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `audio` | file | yes | Audio file to verify |
| `profile_id` | string | yes | Profile to verify against |
| `threshold` | float | no | Override default threshold (0.0-1.0) |
| `return_embedding` | bool | no | Include embedding in response (default: false) |

**Response:**
```json
{
  "profile_id": "default",
  "matched": true,
  "score": 0.892,
  "threshold": 0.7,
  "duration_ms": 45.2,
  "backend": "nemo-titanet",
  "model": "nvidia/speakerverification_en_titanet_large",
  "audio_seconds": 2.1
}
```

**Errors:**
- `400`: Audio too short, silence, or unreadable
- `404`: Profile not found

## Embeddings (Debug)

### POST /v1/embeddings

Extract raw speaker embeddings from audio. Development/debug endpoint.

**Request:** `multipart/form-data`

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `audio` | file | yes | Audio file |

**Response:**
```json
{
  "embedding": [0.123, -0.456, ...],
  "embedding_dim": 192,
  "audio_seconds": 3.5,
  "model": "nvidia/speakerverification_en_titanet_large",
  "backend": "nemo-titanet"
}
```
