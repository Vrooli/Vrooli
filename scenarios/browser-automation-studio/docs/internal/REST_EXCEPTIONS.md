# BAS REST Exceptions Registry

This file enumerates every REST/non-Connect HTTP route deliberately kept in
`scenarios/browser-automation-studio/api/main.go` after the proto+Connect-RPC
migration (plans:bas-migration-to-proto-connect-rpc).

Each entry MUST be tagged inline in code with both:

- A `// RESTException` comment immediately above the route registration.
- A `// RESTReason: <reason>` comment naming one of the canonical reasons from
  `api-steer §6.3`: `multipart_upload`, `webhook_receiver`, `third_party_shape`,
  `ops_probe`.

Every route NOT listed here must move to Connect-RPC. The Phase 10 drift gate
greps the UI for `fetch(.*api/v1` and `axios.*api/v1` and refuses any hit not
backed by an entry below.

| Method | Path | RESTReason | Why it cannot be Connect | Owner |
|---|---|---|---|---|
| GET | `/health` | `ops_probe` | Liveness/readiness probe consumed by container orchestrators and `vrooli scenario` health checks; must be cheap, schema-free, and reachable without RPC envelope decoding. | platform |
| GET | `/api/v1/health` | `ops_probe` | Mirror of `/health` under the API prefix for consistency. | platform |
| GET | `/ws` | `third_party_shape` | WebSocket upgrade for browser clients. Not RPC. | recording |
| GET | `/ws/recording/{sessionId}/frames` | `third_party_shape` | WebSocket binary frame stream from `playwright-driver`. Not RPC. | recording |
| GET | `/ws/execution/{executionId}/frames` | `third_party_shape` | WebSocket binary frame stream from `playwright-driver`. Not RPC. | recording |
| POST | `/api/v1/recordings/import` | `multipart_upload` | Recording archive (.zip) ingestion. Proto JSON would force base64 round-trip on multi-megabyte uploads. Metadata side may later move to Connect with the multipart sub-route remaining REST. | recording |
| GET | `/api/v1/projects/{id}/files/*` | `third_party_shape` | Streams arbitrary file bytes with MIME types decided by extension; consumed by the browser via `<img>`, `<a download>`, and file viewers. The JSON metadata sub-routes (`/files/tree`, `/files/mkdir`, `/files/write`, etc.) now live on `ProjectFilesService` over Connect-RPC. | project_files |
| POST | `/api/v1/internal/ai-navigate/callback` | `webhook_receiver` | Fire-and-forget step and completion event sink for the out-of-process `playwright-driver`. The driver protocol is not RPC-shaped; migrating it would require driving changes through the driver fork. User-facing list/start/status/abort/resume are served by `VisionNavigationService` over Connect-RPC. | vision_navigation |
| GET | `/api/v1/screenshots/*` | `third_party_shape` | Streams screenshot PNG bytes from MinIO directly to `<img src=...>` consumers in the UI. Path-addressed by storage object name, not by RPC entity. No JSON metadata sibling surface exists today; if/when a screenshot catalog is added it will be served via Connect-RPC with the byte-streaming sub-route remaining REST. | recording |
| GET | `/api/v1/screenshots/thumbnail/*` | `third_party_shape` | Thumbnail variant of the screenshot byte-serve. Same `<img>`-consumer shape; not RPC. | recording |
| GET | `/api/v1/artifacts/*` | `third_party_shape` | Streams arbitrary artifact bytes (videos, HAR files, traces, logs) from MinIO directly to `<video>`/`<a download>` consumers. Path-addressed by storage object name. No JSON metadata sibling surface exists today; if/when an artifact catalog is added it will be served via Connect-RPC with the byte-streaming sub-route remaining REST. | recording |
| POST | `/api/v1/internal/history-callback` | `webhook_receiver` | Fire-and-forget history-entry sink posted by the out-of-process `playwright-driver`; bound to the driver protocol, not RPC-shaped. User-facing history list/delete/navigate operations are served by `RecordingsService` over Connect-RPC. | recordings |
| GET | `/api/v1/recordings/assets/{executionID}/*` | `third_party_shape` | Streams stored recording asset bytes (frames, screenshots, video) from disk directly to `<img>`/`<video>`/`<a download>` consumers in the UI Replay tab and CLI renderer. Path-addressed by storage filename, not by RPC entity. | recording |
| POST | `/api/v1/executions/{id}/export` | `third_party_shape` | Replay export endpoint: streams binary mp4/gif/webm bytes or writes files to a caller-supplied `output_dir` on the server filesystem. The shape is fixed by downstream rendering pipelines (`ffmpeg`/`gifski`) consumers, not RPC. | executions |
| POST | `/api/v1/executions/{executionId}/frames` | `webhook_receiver` | Fire-and-forget frame-callback sink posted by the out-of-process `playwright-driver` during execution; same shape as `/internal/history-callback`. Bound to the driver protocol, not RPC. | executions |

The /api/v1/observability/* routes previously listed here moved to
`ObservabilityService` over Connect-RPC during Phase 4 of the migration.
Free-form playwright-driver payloads are round-tripped via
google.protobuf.Struct so contract drift between API/UI/CLI is captured
in proto even though the upstream schema remains opaque.

## Record Mode (`/recordings/live/*`) — deferred

Record Mode lives on chi REST as a coherent block because every route is
either (a) a webhook callback from the out-of-process `playwright-driver`
(`/action`, `/frame`, `/page-event`) or (b) part of a long-running browser
session lifecycle bound to driver IPC (start/stop/navigate/viewport/input,
multi-tab create/activate/close). The shape is dictated by the driver
protocol and the binary frame stream, not by RPC semantics. The READ-side
storage/cookies/history/tabs catalog already lives on `RecordingsService`
over Connect-RPC; the live-recording control plane stays REST for the same
reason `/internal/history-callback` and `/internal/ai-navigate/callback` do.

| Method | Path (prefix) | RESTReason | Why it cannot be Connect | Owner |
|---|---|---|---|---|
| POST | `/api/v1/recordings/live/{sessionId}/action` | `webhook_receiver` | playwright-driver action callback during recording. | recordings |
| POST | `/api/v1/recordings/live/{sessionId}/frame` | `webhook_receiver` | playwright-driver frame callback during recording. | recordings |
| POST | `/api/v1/recordings/live/{sessionId}/page-event` | `webhook_receiver` | playwright-driver page-event callback. | recordings |
| ANY  | `/api/v1/recordings/live/...` (other) | `third_party_shape` | Live recording session lifecycle bound to driver IPC + binary frame stream; shape fixed by driver protocol, not RPC. | recordings |

## Phase 10 deferrals (non-blocking, tracked as future work)

- Typed flag hydration in the generic CLI dispatcher (`cli/internal/protodispatch`)
  uses a single schemaless map decode today; per-method typed request hydration
  is a future polish item.
- Hand-coded UX wrappers around a couple of CLI verbs (recordings import,
  capture command, status command) remain on the legacy code path because they
  need bespoke output formatting / file I/O the dispatcher does not yet model.
  These do not touch the proto+Connect transport — they shell out to the same
  generated Connect clients underneath.

Both are explicitly out of scope for Phase 11 close-out.
