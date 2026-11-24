# Operator-input target and recovery contract

`ResolveOperatorInputs` applies answers to the target whose questions the
operator reviewed. The request target is required; an empty target is rejected
before revision lookup or provider dispatch. When the shared target proxy routes
a remote request, it rewrites the target to `local` at the remote node boundary.
The owner service places that node-local target on every capability action and
includes it in the stable idempotency key. Repeating the same answer on a
different target is a different operation.

If a provider reports a retryable failure or the response is lost after a
provider has committed, the pending question remains in the durable operator
queue. A same-target retry sends the same action key; provider idempotency is
what prevents an operation from running twice. The API returns `Unavailable`
with a sanitized retry instruction rather than leaking provider details. The
operator must retain the selected target and answers for retry; the API does
not persist answer values or automatically replay them.

This contract is covered by in-process Connect/HTTP tests with the target-proxy
interceptor, isolated file-backed queue, and fake idempotent provider. It does
not establish a live Bridge route, real remote node, production provider, or
served Onboarding behavior. Credential/device actions and provider-specific
semantics remain outside this evidence.
