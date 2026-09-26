# Error Handling

> **Status: partially implemented.** The provider, intent, provisioning,
> enrollment, reconciliation, expiry and transport packages now implement
> typed errors and bounded failure paths. The taxonomy below still includes
> post-launch branches that are not implemented.

Most scenarios treat error handling as a presentation problem: turn a
failure into a message a caller can act on. Here it is a cost problem
first. Every error in this scenario is one of four things, and which
one it is decides what happens next:

1. **A refusal**, where nothing was spent and nothing was created. This
   is the cheapest outcome available and it is often the correct one.
2. **A retryable fault**, where the same call may work later and
   retrying it does not risk creating a second billable machine.
3. **A non-retryable fault**, where retrying either cannot help or
   would cost money.
4. **A finding**, which is not an error at all in the transport sense.
   It is a durable observation an operator acts on.

The mistake this document exists to prevent is collapsing those four
into one path. That collapse is not hypothetical: the upstream
reference metering client does exactly it, and the consequence is
recorded in [`PROBLEMS.md`](PROBLEMS.md).

## Proto-Typed Operations

Proto-typed UI, CLI, and inter-scenario calls use Connect-RPC. Errors
move through three layers:

1. Domain and service code returns typed sentinels such as
   `intent.ErrDuplicateKey` or `instance.ErrNotFound`.
2. The API transport edge maps those sentinels to `connect.Error`
   values in `internal/<domain>/service_error_mapping.go`.
3. The UI receives `ConnectError`, maps `ConnectError.code` to an
   `errors.<code>` i18n key with `ui/src/lib/errorMessage.ts`, and
   renders localized copy.

The CLI uses the same `connect.Error` values through cli-core. Human
output is English for now; future CLI i18n should use the same code
names as the UI catalog instead of string-matching messages.

Two rules are specific to this scenario and neither is optional.

**A sentinel must survive the boundary it describes.** A provider
timeout, a rate limit, a quota exhaustion and an authentication failure
are four different situations with four different correct responses, so
they are four sentinels rather than one wrapped transport error. A
caller that cannot tell them apart cannot decide whether to retry, and
a caller that retries the wrong one creates a second machine.

**An error that may have created something is not the same as an error
that definitely did not.** The provider client returns a distinct
sentinel for a call whose outcome is unknown, and every caller treats
it as "possibly created" rather than "failed." This is the failure the
intent-before-action ordering exists for, and it only works if the
error path preserves the distinction that the ordering makes
recoverable.

## Sentinel Mapping

### Provider errors

The provider adapter's four methods produce one of these. The retry
column is the load-bearing one: **retryable here means it is safe to
issue the same call again**, which for `Create` means safe only with
the original idempotency key.

| Sentinel | Cause | Connect code | Retryable | What must happen |
|---|---|---|---|---|
| `provider.ErrTimeout` | The call did not return in time | `deadline_exceeded` | **No, not blindly.** | The outcome is unknown, so the instance may exist. Never re-issue a create without the original idempotency key. Mark the intent for reconciliation and let the sweep resolve it. This is the most dangerous error in the file, because it looks transient and is not. |
| `provider.ErrRateLimited` | The provider is throttling us | `resource_exhausted` | Yes, with backoff | Back off and retry. Carry the original idempotency key on a create. A sweep that trips this should slow down rather than drop the remaining pages, because a partial sweep reads as a clean one. |
| `provider.ErrQuotaExceeded` | The provider's own account limit is reached | `resource_exhausted` | No | Retrying cannot help; the limit is on their side and needs an operator. Distinct from a rate limit despite sharing a code, because one clears on its own and the other does not. Release the reservation and mark the intent abandoned. |
| `provider.ErrUnauthenticated` | The credential is missing, expired or revoked | `unauthenticated` | No | Fail fast and loudly. Never log the credential, the request, or a truncated form of either. A revoked token is the expected case, not an exotic one, because credentials resolve per call rather than at boot. |
| `provider.ErrInvalidSpec` | The requested size or region does not exist at this provider | `invalid_argument` | No | Refuse before any provider call where the adapter's declared facts already make the answer knowable. A validation that only happens at the provider is a round trip spent to learn something we could have known. |
| `provider.ErrNotFound` | The instance is absent at the provider | `not_found` | No | On `Destroy` this is success, not failure. See the destroy section below. On `Describe` it is a reconciliation finding. |
| `provider.ErrUnknownOutcome` | The call failed after the request was sent, with no usable response | `unavailable` | No | Treat as possibly created. The intent is already durable, so the sweep will match whatever exists. |

### Domain errors

| Domain error | Connect code | UI i18n key |
|---|---|---|
| `ErrInvalid<Entity>` | `invalid_argument` | `errors.invalid_argument` |
| `Err<Entity>NotFound` | `not_found` | `errors.not_found` |
| `meter.ErrOutOfCredit` | `resource_exhausted` | `errors.out_of_credit` |
| `meter.ErrCeilingExceeded` | `resource_exhausted` | `errors.ceiling_exceeded` |
| `enroll.ErrOnboardingKeyUnavailable` | `failed_precondition` | `errors.enrollment_unavailable` |
| Unknown service or repository error | `internal` | `errors.internal` |

When you add a domain, keep the mapping file next to that domain's
service layer. The handler should call the mapper instead of switching
on domain error types inline.

### The out-of-credit refusal, and why its branch order matters

A tenant with no credit is not a fault. It is the system working: the
reservation is refused, no provider call is made, nothing is created,
and the caller is told which ceiling it hit so it can do something
about it. The response must name the ceiling, because "refused" with no
number is indistinguishable from a bug to the person receiving it.

**Branch on the out-of-credit status before any circuit-breaker
logic.** This is the specific ordering that must not be reversed, and
it is written here rather than left to the client library because the
reference client gets it wrong. That client discards the decoded error
body on any non-success status and returns only the code, so a genuine
refusal and a server fault arrive identically, and it counts refusals
toward its breaker. The consequence is that a handful of users out of
credit opens the breaker for everyone, which converts a correct
per-tenant refusal into a fleet-wide outage. A working example exists
in the fleet: the switchboard metering client surfaces the response
body.

So the order is fixed:

```
response arrives
  -> is this the out-of-credit status?    -> refusal. Do not touch the breaker.
  -> is this a ceiling refusal?           -> refusal. Do not touch the breaker.
  -> otherwise                            -> fault. Now the breaker may count it.
```

A refusal is never a breaker event, never an alarm, and never an
`internal` code. It is a `resource_exhausted` that names a limit.

The same discipline applies to the per-tenant ceiling in
`COMPUTEM-P1-002`. Refusing a request that would cross a ceiling is the
product working, and a provider spend alert that only sends mail is
what this replaces.

### Reconciler findings are not errors

The reconciler produces rows, not failures. A finding is written,
returned in a list, and acted on by something else. Four things follow
from that and each has bitten a real implementation somewhere:

- **A sweep that produces findings is a successful sweep.** It does not
  return an error, does not alarm on its own, and does not fail a
  health check. Findings are the output, not the exception.
- **The reconciler never resolves.** It destroys nothing, settles
  nothing, releases nothing and adjusts nothing. A reconciler defect
  that can act is a reconciler defect that can destroy a paying
  customer's running node, which is why automated resolution would be a
  contract change rather than a feature.
- **A finding that the reconciler cannot classify is still a finding.**
  Divergence it does not understand is recorded as divergence, not
  swallowed. The row is the point.
- **A sweep that fails partway is an error, and it must not be reported
  as a clean sweep.** A provider rate limit that truncates the
  inventory listing produces a partial comparison, and a partial
  comparison that reads as complete will report every unlisted instance
  as missing. Fail the sweep, record why, and retry.

The one error the reconciler does raise is a cost divergence beyond
threshold, and even that is an alarm rather than an action. Provider
billing data lags by hours to more than a day, so it is a
reconciliation signal and never a control.

### A destroy that fails or times out

**This is the most expensive failure in the design, and it is expensive
because of a decision made deliberately elsewhere: destroy is the only
stop.** There is no pause to fall back to, because a stopped instance
still bills at the full rate on most providers surveyed. So a destroy
that does not happen is not a degraded state that costs a little. It is
an instance billing at the full rate for as long as nobody notices, and
the scenario has no second lever to pull.

The handling rules:

| Situation | Response |
|---|---|
| `Destroy` returns `ErrNotFound` | **Success.** The instance is gone, which is the outcome we wanted. Record `destroyed_at`, settle usage, and move on. Treating this as a failure is how a retry loop gets stuck on an instance that no longer exists. |
| `Destroy` times out | The outcome is unknown and the cost clock may still be running. Do not mark the instance destroyed. Retry with backoff, and confirm with `Describe` rather than assuming. An unconfirmed destroy is not a destroy. |
| `Destroy` returns a rate limit | Retry with backoff. Destroys are the calls that should keep their retry budget when a sweep has to shed work, because every other call can wait and this one is accruing charges. |
| `Destroy` fails repeatedly | Escalate rather than give up quietly. A destroy that has exhausted retries is an operator-visible condition with a currency value attached, not a log line. It belongs in the operator surface next to the instance it concerns. |
| The instance was already destroyed locally | Idempotent. Do not settle twice; a reservation reaches exactly one terminal state. |

Two ordering notes carry over from the retire flow. Node revocation
precedes destruction, so a machine is never unreachable while still
trusted; a revocation failure must not silently skip the destroy that
follows it. And settlement happens in the destroy path rather than in a
later batch, because a settlement that can be dropped is a charge that
never happens.

### Enrollment failure degrades, it does not fail the create

Enrollment is what makes an instance useful. It is not what makes it
cost money. Blocking creation on the trust plane would make capacity
unavailable for a reason unrelated to capacity, so an enrollment
failure after the instance is running is not an error the caller sees
as a failure.

What happens instead: the instance is created, metered and expiring as
normal, its enrollment is queued for retry, and it is visibly flagged
as not enrolled. The operator surface must render an un-enrolled state
as a state rather than as an error, because it is one.

There is exactly one enrollment condition that refuses instead, and it
refuses before anything is created rather than after. The bridge
onboarding public key has to be embedded in the first-boot
configuration at create time and cannot be retrofitted, so when the key
cache is cold and bridge is unreachable, provisioning refuses with
`enroll.ErrOnboardingKeyUnavailable`. Creating a machine that could
never be enrolled by the unattended path would be worse than refusing,
because it costs money and cannot become useful.

This is the only place in the scenario where a bridge failure blocks,
and it blocks in the cheap direction.

The contrast worth keeping in mind: the business suite is the one
dependency that fails closed, because a machine that boots unmetered is
cost with no compensating action available afterwards. Bridge degrades.
Treasury refuses agent-initiated requests only. Getting these three
wrong in either direction is either an outage or an unmetered bill.

### What must never reach an error path

- **A credential value.** Not in a message, not in a wrapped error, not
  in a log line, not in a debug dump, not truncated. A provider token
  is the ability to create unlimited billable machines, so its
  appearance in an error string is the worst outcome available in this
  scenario. Provider call logging records the idempotency key and the
  outcome, not the request.
- **A raw provider response body**, which can carry account
  identifiers, quota details and occasionally echoed request material.
  Map it to a sentinel at the adapter boundary and log the sentinel.
- **A stack trace on a refusal.** A refusal is an expected outcome.
  Rendering it like a crash trains the reader to ignore real crashes.

## Multipart REST Exceptions

Opaque file bytes are not proto payloads, so the template keeps one
documented exception: REST multipart for bytes, with proto-typed
metadata in the response. Those endpoints use a stable error envelope
through `internal/httpx.WriteError`, and the UI maps `ApiError.code`
through the same `errorMessage(...)` utility as Connect errors.

**This scenario has no multipart endpoints and is not expected to grow
one.** Everything it moves is structured and small: instance records,
intents, reservations, findings, and a rendered first-boot
configuration. The exception is documented here so that its absence is
a decision rather than an oversight, and so that anyone who reaches for
it has to justify the reach.

Use this split:

- Connect-RPC for messages that can be described by proto.
- REST multipart for file bytes, if bytes ever appear.
- Proto metadata responses for REST upload results.

Do not introduce a second general JSON transport for internal scenario
calls. If the payload is structured and Vrooli-owned, add a proto
service method.

## Cross-references

- [`PROBLEMS.md`](PROBLEMS.md): the reference client that collapses refusals into faults, the ten-minute reservation window, and the refund path that silently does nothing
- [`DECISIONS.md`](DECISIONS.md): why destroy is the only stop, why the business suite fails closed, and why bridge degrades
- [`SEAMS.md`](SEAMS.md): the boundaries these errors cross
- [`SECURITY.md`](SECURITY.md): the credential handling rules the error paths must not break
- [`../concepts/FLOWS.md`](../concepts/FLOWS.md): the orderings a failure has to unwind correctly
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md): what each dependency does when it is unavailable
