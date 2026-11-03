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

## E6b private plan, execute, and receipt

A missing registered feature is refused before policy evaluation. A policy
snapshot error fails closed; zero approved records produces a non-eligible plan.
The plan repository binds one operation ID to one request digest and refuses a
changed retry. Execute reloads that plan and reruns the same eligibility path.
If eligibility changes, the saved route no longer leads, or the provider
executor is unavailable, execute persists a terminal failure receipt without
falling back to another route. Receipt persistence uses a detached context after
execute accepts the request, so caller disconnect does not cancel the outcome.

The Hub decision lookup returns unavailable when its verifier is not configured,
unauthorized for a missing or invalid dedicated service identity, and not-found
for an ask that does not exactly match the supplied source, event, requester,
digest, scope, or explicit approve/reject options. The shared provider-decision
receipt verifier rejects non-answered, defaulted, rejected, unverified-human,
or out-of-window receipts. The composed Hub client refuses to send when
credential lookup, exchange, expiry, or exact subject/audience/capability
validation fails. It returns bounded errors without credential, token, or Hub
response-body content. E6b plan/execute does not yet invoke the Hub client or
receipt verifier. No live credential, ask, or answer is created to test these
paths.

## Media Receipt Replay and Unknown Outcome

Submit binds an idempotency key to the entire semantic request, including its
Gateway request/correlation fields, ordered input references, output reference,
prompt and count. Only key whitespace is normalized. An identical request
reattaches to the retained receipt; changed content returns
`ErrMediaIdempotencyConflict` as Connect `already_exists`. Unreadable retained
request JSON returns `ErrMediaRetainedSubmissionInvalid` as `internal` rather
than returning an unrelated result.

Startup recovery quarantines interrupted `RUNNING` receipts as terminal
`FAILED` with `error_code=outcome_unknown` before decoding retained requests.
The error message states that the provider outcome and cost are unknown and
that execution was not replayed. Get and Wait return this terminal receipt;
Cancel returns it unchanged; identical Submit and repeated recovery retain it.
Retry returns `ErrMediaOutcomeUnknown` as Connect `failed_precondition` without
dispatch. A quarantine database-write failure is returned by Recover and logged
as incomplete startup recovery; the affected work is never reset or dispatched.

For `outcome_unknown`, absent optional `actual_cost_usd` means unknown; its
numeric zero default is not evidence of zero cost. Human CLI output prints
`actual_cost_usd=unknown` when absent. Structured CLI output keeps canonical
proto JSON: absent cost is omitted and explicitly present zero is retained.
Image Tools treats every non-success terminal receipt as an error before
mapping a media result.

Ordinary executor failure also does not prove that the provider failed or
charged nothing: errors may occur after provider work while decoding or writing
an output. Gateway accepts provider-reported cost only from Resource
OpenRouter's versioned normalized envelope and preserves optional presence
through receipts. When known cost accompanies local validation or output
failure, it is stored with FAILED under the RUNNING compare-and-set. Unknown
cost remains absent. This is the AM-22/D31 correction; AM-20 startup replay,
request identity, and outcome-unknown quarantine remain in force. See
[the SM12 flow inventory](../concepts/FLOWS.md#media-receipt-state-model-sm12).
