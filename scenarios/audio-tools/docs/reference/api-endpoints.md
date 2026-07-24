# API Endpoints — Audio Tools

## Service map

| Service | Handler |
|---|---|
| `STTService` and `STTAdminService` | `api/handlers/stt` |
| `TTSService` | `api/handlers/tts` |
| `SummarizeService` | `api/handlers/summarize` |
| `AudioProcessingService` | `api/handlers/audio` |
| `CorpusService` and `ExperimentService` | `api/handlers/corpus`, `api/handlers/experiment` |
| `SettingsService`, `UsageService`, and `DiagnosticsService` | their matching handler domains |
| `HealthService` | `api/handlers/health_status` |

The authoritative wire contracts are the generated schemas under
`packages/proto/schemas/audio-tools/v1/`. Handlers use generated Connect-RPC
types. Clients must use generated clients instead of hand-written JSON shapes.

## REST and transport exceptions

| Path | Method | Reason |
|---|---|---|
| `/health` and `/api/v1/health` | GET | Lifecycle and load-balancer probe |
| `/api/v1/voice/transcribe` | POST | Multipart audio upload |
| `/api/v1/audio/transcode` | POST | Multipart audio upload |
| `/api/v1/voice/stream` | WS upgrade | Browser voice-stream transport |

`STTService.TranscribeStream` is a bidirectional Connect stream. The server
emits partial, segment, error, and terminal done events. A done event that
falls back to unary transcription explicitly identifies buffered mode.

## Adding an endpoint

1. Extend the owning proto domain and regenerate generated clients.
2. Add a thin handler method in the owning `api/handlers/<domain>` package.
3. Keep product rules in `api/internal/<domain>`.
4. Add endpoint metadata and tests for each touched layer.
5. Update this reference when the public contract changes.

## Cross-references

- [`cli-commands.md`](cli-commands.md)
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md)
- [`../domains/stt/streaming-pipeline.md`](../domains/stt/streaming-pipeline.md)
