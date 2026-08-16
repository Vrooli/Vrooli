# Error Handling

The template uses one error path for proto-typed operations and one
documented exception path for multipart REST.

## Proto-Typed Operations

Proto-typed UI, CLI, and inter-scenario calls use Connect-RPC. Errors
move through the domain, transport, and client layers:

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

## Sentinel Mapping

| Domain error | Connect code | UI i18n key |
|---|---|---|
| `ErrInvalid<Entity>` | `invalid_argument` | `errors.invalid_argument` |
| `Err<Entity>NotFound` | `not_found` | `errors.not_found` |
| Unknown service/repository error | `internal` | `errors.internal` |

When you add a domain, keep the mapping file next to that domain's
service layer. The handler should call the mapper instead of switching
on domain error types inline.

## In-Response Inference Errors

Typed inference does not raise transport errors for domain outcomes. A failed
run returns a `RunResponse` carrying an `InferenceError`, so a caller can choose
to abstain or raise while transport errors stay reserved for malformed RPCs.

| Code | Meaning | `construct` |
|---|---|---|
| `INVALID_REQUEST` | The request itself is wrong. | the offending field |
| `UNSUPPORTED_SCHEMA` | The schema uses a keyword outside the enforceable subset. | the keyword |
| `VALIDATION_FAILED` | The provider's value failed local validation after every attempt. | — |
| `PROVIDER_FAILED` | No candidate completed. | — |
| `UNAVAILABLE` | The role catalog is not loaded, or no candidate was reachable. | — |
| `UNSUPPORTED_SAMPLING` | No remaining candidate's provider can honour a caller-supplied sampling control. | `sampling.temperature` |

`UNSUPPORTED_SAMPLING` and `INVALID_REQUEST` both arise from sampling and are
kept distinct because they need different fixes. A caller who sends a
temperature to a role that has not declared itself overridable gets
`INVALID_REQUEST`: the role forbids it, and the caller can correct the request.
A caller who sends one to an overridable role whose candidates all decline it
gets `UNSUPPORTED_SAMPLING`: that is provider incapacity, and no request change
repairs it. Collapsing them would tell the caller to fix the wrong thing.

Neither is ever a silent downgrade. This follows the schema gate's governing
rule — a rejected schema is never degraded to unconstrained generation — because
a caller-supplied control is a promise the gateway keeps or refuses. A
*role-declared* default is different: it is a preference, so the gateway sends it
best-effort to anything that will not fail on it, and drops it silently only for
a candidate declaring `rejected`, where sending would 400 the call.

On the routing path there is no typed error surface, so the same condition is
reported as route evidence with `failure_class=unsupported_sampling`. It is
never recorded against provider health: the provider failed nothing, and
tripping a circuit breaker over a policy mismatch would suppress a healthy
route.

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
