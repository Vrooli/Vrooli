# Error Handling

Code Facts uses one error path for proto-typed operations and one documented
exception path for the operational health probe.

## Proto-Typed Operations

Proto-typed UI, CLI, and inter-scenario calls use Connect-RPC. Errors
move through three layers:

1. Target resolution, family validation, and analyzer code return ordinary
   errors or typed provider-condition errors.
2. The API transport edge maps invalid requests to `connect.CodeInvalidArgument`
   and preserves provider conditions inside the typed report.
3. The CLI and UI consume the same Connect response shape; warnings and
   evidence are rendered instead of being converted into false success.

The CLI uses the same `connect.Error` values through cli-core. Human
output is English for now; future CLI i18n should use the same code
names as the UI catalog instead of string-matching messages.

## Sentinel Mapping

| Condition | Connect/report behavior | Consumer behavior |
|---|---|---|
| Invalid target or page token | `invalid_argument` | Fix request before retrying |
| Provider unavailable in non-strict mode | successful report with `unknown` evidence | Surface warning and preserve partial facts |
| Unsupported family/kind | successful report with `unsupported` evidence | Do not claim coverage |
| Unexpected analyzer/storage failure | `internal` | Retry or investigate service health |

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
