# API Endpoints — Audio Tools

> Plan: `~/.vrooli/plans/audio-tools-greenfield-scenario-web-console-adoption.md`

## Service map

| Service | Path prefix | Methods | Handler |
|---|---|---|---|
| `STTService` | `/vrooli.audio_tools.v1.stt.STTService/*` | Transcribe, TranscribeStream (bidi), GetStreamConfig, UpdateStreamConfig, GetWakeWordConfig, UpdateWakeWordTemplate, DeleteWakeWordTemplate, GetSpeakerConfig, UpdateSpeakerConfig, GetSpeakerStatus, ListSpeakerProfiles, EnrollSpeakerProfile, ClearSpeakerProfileBinding, RemoveSpeakerProfile, DeleteSpeakerProfile | `handlers/stt` |
| `TTSService` | `/vrooli.audio_tools.v1.tts.TTSService/*` | Synthesize, ListVoices, GetCache, GetConfig, UpdateConfig, GetStatus, RecordPlaybackEvent, NormalizeForSpeech, SplitParagraphs | `handlers/tts` |
| `SummarizeService` | `/vrooli.audio_tools.v1.summarize.SummarizeService/*` | Summarize | `handlers/summarize` |
| `AudioProcessingService` | `/vrooli.audio_tools.v1.audio.AudioProcessingService/*` | Transcode, Trim, Merge, Split, Fade, Volume, Normalize, ExtractMetadata | `handlers/audio` |
| `SessionService` | `/vrooli.audio_tools.v1.session.SessionService/*` | OpenSession, CloseSession, SendText, SendCancel, Subscribe (server-streaming) | `handlers/session` |
| `SettingsService` | `/vrooli.audio_tools.v1.settings.SettingsService/*` | GetProviderConfig, UpdateProviderConfig, ListBYOKCredentials, UpsertBYOKCredential, DeleteBYOKCredential, GetVoiceOverrides, SetVoiceOverride | `handlers/settings` |
| `UsageService` | `/vrooli.audio_tools.v1.usage.UsageService/*` | ListRecent, GetSummary | `handlers/usage` |

## REST exceptions (the four legitimate escape hatches)

| Path | Method | `RESTReason` | Why |
|---|---|---|---|
| `GET /health` | GET | `ops_probe` | Load balancers, `curl`, lifecycle probes must reach without a Connect client. |
| `POST /api/v1/voice/transcribe` | POST | `multipart_upload` | Audio bytes via multipart form-data; inline proto JSON would be inefficient. |
| `POST /api/v1/audio/transcode` | POST | `multipart_upload` | Same. |

## Transport exceptions

| Path | Method | Transport | Reason |
|---|---|---|---|
| `GET /api/v1/voice/stream` | GET (WS upgrade) | WebSocket | `TransportReason: websocket_transport` — pending template constant (R-PROTO); today tagged `ops_probe` with explicit note. |

## Per-request headers

| Header | Tier | Purpose |
|---|---|---|
| `X-Audio-BYOK-Provider` | BYOK | Names which adapter handles the request (e.g., `openai-whisper`). |
| `X-Audio-BYOK-Key` | BYOK | The user's third-party API key. NEVER logged. |
| `X-Audio-LPBS-Token` | Vrooli/LPBS | LPBS auth token. |
| `X-Audio-User-Identity` | All | User identifier for usage attribution. |

The machine-readable source of truth is [`.vrooli/endpoints.json`](../../.vrooli/endpoints.json) —
doc generators, Postman collection builders, and SDK stubs read it
directly. The CI gate fails if the JSON drifts from the registered
handlers or from the CLI commands it claims to mirror.

Wire shapes for every endpoint live in
`packages/proto/schemas/audio-tools/v1/<domain>/<file>.proto`.
Proto-typed calls use generated Connect-RPC handlers and clients.
Tests, handlers, UI clients, and CLI handlers all consume generated
types — no hand-written struct mirror exists to drift.

Connect-RPC errors use Connect's canonical error envelope and code set.
REST exceptions, such as multipart uploads, use the template error
envelope (`packages/proto/schemas/audio-tools/v1/errors/errors.proto`):

```json
{ "code": "<canonical_code>", "message": "<human readable>", "details": [...] }
```

Canonical REST codes used today: `invalid_request` (400),
`not_found` (404), `internal` (500). Add to the proto enum when a new
REST-exception failure mode appears.

---

## System

### `GET /health`

Service health check. Returns API readiness plus dependency status.
Also mounted at `/api/v1/health` for client callers.
This is an operational REST exception by design: lifecycle systems,
load balancers, and curl probes must be able to read it without a Connect
client.

| | |
|---|---|
| **Auth** | None |
| **Response** | `Response { status: string, readiness: bool, service: string, timestamp: string, version: string, uptime_seconds: int64, dependencies: map<string, DependencyStatus> }` |
| **Errors** | None — always returns 200 with `status: "unhealthy"` if a dependency fails |
| **CLI** | `audio-tools status` |

```bash
curl "http://localhost:${API_PORT}/health"
```

The proto type lives at `packages/proto/schemas/audio-tools/v1/health/health.proto`
and mirrors `api-core/health.Response` field-for-field.

---

## Notes (CRUD reference)

The `notes` domain is the canonical worked example. Copy its layering
when adding the first non-trivial mutation in your scenario.

### `POST /vrooli.audio_tools.v1.notes.NotesService/ListNotes`

List notes through the generated Connect-RPC service, newest-first.

| | |
|---|---|
| **Auth** | None (template default; scenarios add auth as needed) |
| **Response** | `ListNotesResponse { notes: Note[] }` (capped at 100 by `notes.Service`) |
| **Errors** | `500 internal` — repository read failure |
| **CLI** | `audio-tools notes list` |

```bash
curl -X POST "http://localhost:${API_PORT}/vrooli.audio_tools.v1.notes.NotesService/ListNotes" \
  -H 'Content-Type: application/json' \
  -d '{}'
```

UI and CLI code should normally use the generated client instead of
calling this path by hand.

### `POST /vrooli.audio_tools.v1.notes.NotesService/CreateNote`

Create a note through the generated Connect-RPC service.

| | |
|---|---|
| **Auth** | None (template default) |
| **Request** | `CreateNoteRequest { title: string (required), body: string (optional) }` |
| **Response** | `CreateNoteResponse { note: Note }` |
| **Errors** | `invalid_argument` — missing/whitespace-only title<br>`internal` — repository write failure |
| **CLI** | `audio-tools notes create --title <title> [--body <body>]` |

```bash
curl -X POST "http://localhost:${API_PORT}/vrooli.audio_tools.v1.notes.NotesService/CreateNote" \
  -H 'Content-Type: application/json' \
  -d '{"title":"first","body":"hello"}'
```

Title validation (non-empty after whitespace trim) lives in
`internal/notes/service.go`, **not** the handler. The Connect handler
only translates `notes.ErrInvalidNote` into `invalid_argument`.

### `POST /vrooli.audio_tools.v1.notes.NotesService/GetNote`

Fetch a note by id through the generated Connect-RPC service.

| | |
|---|---|
| **Auth** | None (template default) |
| **Request** | `GetNoteRequest { id: string }` |
| **Response** | `GetNoteResponse { note: Note }` |
| **Errors** | `not_found` — no note with that id<br>`internal` — repository read failure |
| **CLI** | `audio-tools notes get <id>` |

```bash
curl -X POST "http://localhost:${API_PORT}/vrooli.audio_tools.v1.notes.NotesService/GetNote" \
  -H 'Content-Type: application/json' \
  -d '{"id":"abc123"}'
```

`notes.ErrNoteNotFound` returned by the service is translated into the
typed `not_found` Connect error at the handler edge.

### `POST /api/v1/notes/{id}/attachments`

Upload opaque file bytes through the documented REST multipart exception.
The response is still proto-typed metadata.

| | |
|---|---|
| **Auth** | None (template default) |
| **Path params** | `id` — note identifier |
| **Request** | `multipart/form-data` with `file` part |
| **Response** | `UploadAttachmentResponse { attachment: Attachment }` |
| **Errors** | `400 invalid_request` — malformed multipart or missing file<br>`404 not_found` — no note with that id<br>`500 internal` — blob or metadata persistence failure |
| **CLI** | `audio-tools notes attach <id> --file <path>` |

```bash
curl -X POST "http://localhost:${API_PORT}/api/v1/notes/abc123/attachments" \
  -F file=@./example.png
```

### `Note` shape

| Field | Type | Notes |
|---|---|---|
| `id` | string (UUID) | Server-generated |
| `title` | string | Required, non-empty after trim |
| `body` | string | Optional |
| `created_at` | `google.protobuf.Timestamp` | Server-set on create |
| `updated_at` | `google.protobuf.Timestamp` | Server-set on create / future update |
| `attachment_keys` | `string[]` | Keys of uploaded note attachments |

Defined in `packages/proto/schemas/audio-tools/v1/notes/notes.proto`.

---

## Adding a new endpoint

For a new domain, copy the notes vertical slice first, then replace it
once your real domain is green.

For an endpoint inside an existing domain:

1. Add or extend the `.proto` messages and service in
   `packages/proto/schemas/audio-tools/v1/<domain>/`, then run
   `make generate`.
2. Implement the generated handler method in
   `handlers/<domain>/connect_handler.go`; keep it thin.
3. Update endpoint metadata in `handlers/<domain>/module.go`.
4. If the endpoint has a CLI mirror, update
   `api/cmd/gen-endpoints/cli_commands_seed.json`.
5. Run `make endpoints`; do not edit
   [`.vrooli/endpoints.json`](../../.vrooli/endpoints.json) by hand.
6. Update this document and add tests for the touched layers.
7. Add a row to [`internal/SEAMS.md`](../internal/SEAMS.md) if you
   introduced a new interface that production wires once and tests
   substitute.

The CI gate enforces endpoint-manifest freshness and command-seed
consistency.

## Cross-references

- [`cli-commands.md`](cli-commands.md) — CLI commands that mirror these endpoints
- [`configuration.md`](configuration.md) — env vars (e.g., `API_PORT`)
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md#proto-as-the-canonical-contract) — proto bridge details
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — handler/service/repository seams
- [`../internal/TESTING.md`](../internal/TESTING.md) — endpoint test patterns
