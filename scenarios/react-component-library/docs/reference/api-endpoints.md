# API Endpoints — React Component Library

Human-readable reference for the API. The machine-readable
source of truth is [`.vrooli/endpoints.json`](../../.vrooli/endpoints.json) —
doc generators, Postman collection builders, and SDK stubs read it
directly. The CI gate fails if the JSON drifts from the registered
handlers or from the CLI commands it claims to mirror.

Wire shapes for every endpoint live in
`packages/proto/schemas/react-component-library/v1/<domain>/<file>.proto`.
Proto-typed calls use generated Connect-RPC handlers and clients.
Tests, handlers, UI clients, and CLI handlers all consume generated
types — no hand-written struct mirror exists to drift.

Connect-RPC errors use Connect's canonical error envelope and code set.
REST exceptions, such as multipart uploads, use the template error
envelope (`packages/proto/schemas/react-component-library/v1/errors/errors.proto`):

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
| **CLI** | `react-component-library status` |

```bash
curl "http://localhost:${API_PORT}/health"
```

The proto type lives at `packages/proto/schemas/react-component-library/v1/health/health.proto`
and mirrors `api-core/health.Response` field-for-field.

---

## Components

The components domain owns authoring and indexing of Git-tracked source
under `library/components/<slug>/`. The API writes source files and
manifests; UI and CLI stay thin generated-client callers.

### `POST /vrooli.react_component_library.v1.components.ComponentsService/InitializeComponent`

Create `component.json` plus `versions/<initial_version>/<file>.tsx`,
run the indexer, and return the registry row.

| | |
|---|---|
| **Request** | `InitializeComponentRequest { library_id, slug, display_name, description, tags[], initial_version, file_name, initial_source }` |
| **Response** | `InitializeComponentResponse { component, manifest_path, source_path }` |
| **Errors** | `invalid_argument` — invalid slug/version/file/header<br>`already_exists` — duplicate library id or slug<br>`internal` — filesystem/index failure |
| **CLI** | `react-component-library components init <slug>` |

### `POST /vrooli.react_component_library.v1.components.ComponentsService/CreateComponentVersion`

Create a new version folder for an existing component, update the
manifest's `latest` or `draft` pointer, re-index, and return the version
row.

| | |
|---|---|
| **Request** | `CreateComponentVersionRequest { component_id, version, from_version, intent, file_name, source, changelog_md }` |
| **Response** | `CreateComponentVersionResponse { component, version, source_path }` |
| **Errors** | `not_found` — component missing<br>`invalid_argument` — invalid version/file/header<br>`internal` — filesystem/index failure |
| **CLI** | `react-component-library components version-create "<component-id>" "<version>"` |

### `POST /vrooli.react_component_library.v1.components.ComponentsService/UpdateComponentManifest`

Update display metadata and explicit version pointers in
`component.json`, then re-index.

| | |
|---|---|
| **Request** | `UpdateComponentManifestRequest { component_id, display_name, description, tags[], latest_version, draft_version, deprecated_versions[] }` |
| **Response** | `UpdateComponentManifestResponse { component }` |
| **Errors** | `not_found` — component missing<br>`invalid_argument` — invalid manifest values<br>`internal` — filesystem/index failure |
| **CLI** | `react-component-library components manifest-update "<component-id>"` |

---

## Adding a new endpoint

For a new domain, copy the notes vertical slice first, then replace it
once your real domain is green.

For an endpoint inside an existing domain:

1. Add or extend the `.proto` messages and service in
   `packages/proto/schemas/react-component-library/v1/<domain>/`, then run
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
