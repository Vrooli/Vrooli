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
| `ModelRegistryService` | `api/handlers/models` |
| `HealthService` | `api/handlers/health_status` |

The authoritative wire contracts are the generated schemas under
`packages/proto/schemas/audio-tools/v1/`. Handlers use generated Connect-RPC
types. Clients must use generated clients instead of hand-written JSON shapes.

## REST and transport exceptions

| Path | Method | Reason |
|---|---|---|
| `/health` and `/api/v1/health` | GET | Lifecycle and load-balancer probe |

`STTService.TranscribeStream` is the bidirectional browser and consumer voice
stream. Browser clients use the generated Connect client through their
scenario-owned adapter. The server emits partial, segment, error, and terminal
events. A terminal event that falls back to unary transcription explicitly
identifies buffered mode.

`ModelRegistryService` is a read-only projection of the validated model/voice
seed (`api/internal/modelcatalog`). `ListModels` filters the registry;
`ResolveModel` applies the default-selection policy and fails closed on
licence when `commercial = true`. Neither method writes state, installs
weights, or probes a provider — live availability belongs to `health_status`,
`provider_lifecycle`, and `stt.ListEngines`.

## Adding a new endpoint

1. Extend the owning proto domain and regenerate generated clients.
2. Add a thin handler method in the owning `api/handlers/<domain>` package.
3. Keep product rules in `api/internal/<domain>`.
4. Add endpoint metadata and tests for each touched layer.
5. Update this reference when the public contract changes.

## Cross-references

- [`cli-commands.md`](cli-commands.md)
- [Endpoint manifest](../../.vrooli/endpoints.json) — generated metadata; update through endpoint tooling, not by hand.
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md)
- [`../domains/stt/streaming-pipeline.md`](../domains/stt/streaming-pipeline.md)
