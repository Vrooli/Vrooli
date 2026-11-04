# Error Handling

The template uses one error path for proto-typed operations and one
documented exception path for multipart REST.

## Proto-Typed Operations

Proto-typed UI, CLI, and inter-scenario calls use Connect-RPC. Errors
move through several layers:

1. Domain/service code returns typed sentinels such as
   `<domain>.ErrInvalid<Entity>` or `<domain>.Err<Entity>NotFound`.
2. The API transport edge maps those sentinels to `connect.Error`
   values in `internal/<domain>/service_error_mapping.go`.
3. The UI receives `ConnectError`, maps `ConnectError.code` to an
   `errors.<code>` i18n key with `ui/src/lib/errorMessage.ts`, and
   renders localized copy.

The CLI uses the same `connect.Error` values through cli-core. Human
output is English for now; future CLI i18n should use the same code
names as the UI catalog instead of string-matching messages.

## Embedding failures and later retries

`GatewayClient.Embed` makes at most one `ExecuteRoute` RPC per invocation.
Transport timeouts, cancellation, missing or rejected responses, malformed
JSON, and empty vectors return an error without resubmitting. Transport and
decode errors retain their original error chain. A failed response does not
prove that the provider did no work or incurred no cost; Gateway request IDs
are correlation values, not replay keys.

Journal append persists an `embed` retry record when a derived vector is
missing. `ProcessEmbeddingRetries` later invokes `Embed` again for missing
vectors and acknowledges the queue only after they are stored. The queue has
no durable provider-operation receipt or outcome reconciliation. A later
invocation after an unknown outcome can repeat provider work, so this flow
does not guarantee exactly-once execution across invocations or processes.
Durable reconciliation for that boundary remains unresolved.

Sources: `path:scenarios/source-ledger/api/internal/inference/client.go` and
`path:scenarios/source-ledger/api/internal/journal/service.go`.

## Sentinel Mapping

| Domain error | Connect code | UI i18n key |
|---|---|---|
| `ErrInvalid<Entity>` | `invalid_argument` | `errors.invalid_argument` |
| `Err<Entity>NotFound` | `not_found` | `errors.not_found` |
| Unknown service/repository error | `internal` | `errors.internal` |

When you add a domain, keep the mapping file next to that domain's
service layer. The handler should call the mapper instead of switching
on domain error types inline.

## Multipart REST Exceptions

Opaque file bytes are not proto payloads. Upload endpoints use REST
multipart for bytes and return proto-typed metadata. These endpoints
still use a stable error envelope through `internal/httpx.WriteError`;
the UI maps `ApiError.code` through the same `errorMessage(...)`
utility as Connect errors.

Use this split:

- Connect-RPC for messages that can be described by proto.
- REST multipart for file bytes.
- Proto metadata responses for REST upload results.

Do not introduce a second general JSON transport for internal scenario
calls. If the payload is structured and Vrooli-owned, add a proto
service method.
