# Error Handling

## Target Proposal Findings

Generic proposal operations must distinguish invalid selection/content, unsupported artifact owner, unavailable source, stale/relevant-base conflict, missing authority, held dependency, partial effect and retryable owner failure. Include proposal/revision/operation/entry identity and safe next actions, without sensitive payloads. Preserve draft input after failure. Repeated delivery uses the same operation identity; a transport timeout does not prove that an effect did not occur. These are target semantics, not a claim of existing error variants or implemented general proposal endpoints.

The current product domains use Connect-RPC error translation. General proposal
failure semantics above remain implementation targets.

## Proto-Typed Operations

Proto-typed UI, CLI, and inter-scenario calls use Connect-RPC. Errors
move through domain, transport, and client layers:

1. Domain/service code returns typed sentinels such as
   planning or ontology validation errors.
2. The API transport edge maps those sentinels to `connect.Error`
   values in `api/handlers/planning/connect_handler.go` and
   `api/handlers/ontology/connect_handler.go` (`toConnectError`).
3. The UI receives `ConnectError`, maps `ConnectError.code` to an
   `errors.<code>` i18n key with `ui/src/lib/errorMessage.ts`, and
   renders localized copy.

The CLI uses the same `connect.Error` values through cli-core. Human
output is English for now; future CLI i18n should use the same code
names as the UI catalog instead of string-matching messages.

## Sentinel Mapping

| Domain error | Connect code | UI i18n key |
|---|---|---|
| `planning.ErrInvalidArgument`, `ontology.ErrInvalidArgument` | `invalid_argument` | `errors.invalid_argument` |
| `ontology.ErrCapabilityCycle` | `invalid_argument` | `errors.invalid_argument` |
| `planning.ErrScenarioNotFound`, `planning.ErrProtoFileNotFound` | `not_found` | `errors.not_found` |
| `ontology.ErrCapabilityNotFound` | `not_found` | `errors.not_found` |
| Unknown service/repository error | `internal` | `errors.internal` |

These mappings describe current planning and ontology handlers. The graph handler
has its own mapping in `api/handlers/graph/connect_handler.go`; do not infer
general source-unavailability or proposal-conflict behavior from this table.

## Multipart REST Exceptions

No generic draft-upload or multipart application endpoint is established by
this guide. Current planned proto text travels through typed RPCs. Any future
opaque-byte transport needs an explicit endpoint exception and owner-defined
error contract. The shared UI `ApiError` translator is not evidence that such
an endpoint exists.

Do not introduce a second general JSON transport for internal scenario
calls. If the payload is structured and Vrooli-owned, add a proto
service method.
