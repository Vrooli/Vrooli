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
| GET | `/api/v1/observability` | `third_party_shape` | Byte-for-byte proxy to `playwright-driver` `/observability`. Downstream owns the schema; wrapping in proto would only add a `Struct` envelope. | observability |
| POST | `/api/v1/observability/refresh` | `third_party_shape` | Proxy to `playwright-driver`. Same rationale as `/observability`. | observability |
| POST | `/api/v1/observability/diagnostics/run` | `third_party_shape` | Proxy to `playwright-driver` diagnostics. Open-shape response. | observability |
| GET | `/api/v1/observability/sessions` | `third_party_shape` | Proxy to `playwright-driver` session inventory. | observability |
| POST | `/api/v1/observability/cleanup/run` | `third_party_shape` | Proxy to `playwright-driver` cleanup run. | observability |
| GET | `/api/v1/observability/metrics` | `third_party_shape` | Proxy returning metrics in the upstream-defined shape (parsed from Prometheus text format by `playwright-driver`). | observability |
| POST | `/api/v1/observability/pipeline-test` | `third_party_shape` | Proxy to autonomous end-to-end pipeline test. | observability |
| GET | `/api/v1/observability/config/runtime` | `third_party_shape` | Proxy returning runtime config map; key set is owned by `playwright-driver` and changes independently of BAS releases. | observability |
| PUT | `/api/v1/observability/config/{envVar}` | `third_party_shape` | Proxy update for downstream-owned runtime config. | observability |
| DELETE | `/api/v1/observability/config/{envVar}` | `third_party_shape` | Proxy reset for downstream-owned runtime config. | observability |
| GET | `/api/v1/observability/debug-mode` | `ops_probe` | In-process debug-mode toggle consumed by diagnostics UI; free-form JSON. | observability |
| POST | `/api/v1/observability/debug-mode` | `ops_probe` | Companion writer to debug-mode toggle. | observability |
| GET | `/api/v1/projects/{id}/files/*` | `third_party_shape` | Streams arbitrary file bytes with MIME types decided by extension; consumed by the browser via `<img>`, `<a download>`, and file viewers. The JSON metadata sub-routes (`/files/tree`, `/files/mkdir`, `/files/write`, etc.) now live on `ProjectFilesService` over Connect-RPC. | project_files |

## Pending evaluation

As phase migrations land, the following candidate multipart routes will be
reviewed; if they actually consume `multipart/form-data` or stream large
binary blobs, they will be moved here. Otherwise their JSON metadata moves to
Connect.

- HAR / video / trace export endpoints under `/api/v1/executions/{id}/...` —
  to be classified during Phase 7.
