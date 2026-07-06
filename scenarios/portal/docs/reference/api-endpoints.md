# API Endpoints — Portal

Portal is Connect/proto-first. The machine-readable endpoint source is
[`.vrooli/endpoints.json`](../../.vrooli/endpoints.json); wire shapes live in
`packages/proto/schemas/portal/v1/**`.

## System

### `GET /health`

Lifecycle health probe. This is the intentional REST ops-probe exception so
lifecycle systems and load balancers can read health without a Connect client.

| | |
|---|---|
| **Auth** | None |
| **Response** | `portal/v1/shared/health.proto` |
| **CLI** | `portal status` |

## Connect Services

Connect procedures are mounted at generated paths in the form
`/vrooli.portal.v1.<domain>.<Service>/<Method>`.

| Service | Methods | CLI Mirror |
|---|---|---|
| `ChatService` | `ListChats`, `CreateChat`, `GetChat`, `UpdateChat`, `DeleteChat`, `ListGroups`, `CreateGroup`, `UpdateGroup`, `DeleteGroup` | `portal chats ...` |
| `MessageService` | `GetTree`, `SendMessage`, `EditMessage`, `Regenerate`, `StreamCompletion` | `portal messages ...` |
| `IntegrationsService` | `Status`, `UpdateOverride` | `portal integrations ...` |
| `SearchService` | `Suggest` | `portal search suggest ...` |

## Adding a new endpoint

1. Add or extend the proto service under `packages/proto/schemas/portal/v1/`.
2. Run `make generate` from `packages/proto/`.
3. Implement the generated API handler.
4. Add endpoint metadata in the handler module and regenerate endpoints with
   `make endpoints` from `scenarios/portal/api`.
5. Add or intentionally omit CLI coverage in `cli/manifest.json`.
6. Update this document and tests.

## Cross-references

- [`cli-commands.md`](cli-commands.md) — CLI commands that mirror these endpoints
- [`configuration.md`](configuration.md) — env vars and config
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — contract flow
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — handler/service/repository seams
- [`../internal/TESTING.md`](../internal/TESTING.md) — endpoint test patterns
