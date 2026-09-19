# Error Handling — Switchboard

## Purpose Of This Document

How this scenario decides what is an error, and where each kind surfaces. It
inherits the template's transport error path unchanged; what it adds is a
taxonomy, because **this scenario produces more non-error negative outcomes than
any other in the ecosystem**, and treating them as errors would break the
product in three separate ways.

## What Is Not An Error

Four outcomes look like failures, are counted as failures by every off-the-shelf
convention, and are **successes here**. Getting this wrong is the most likely
error-handling defect in the scenario.

| Outcome | Why it is a success | What must never happen |
|---|---|---|
| **A refused turn** | The trust guard did its job. `OT-P0-010` makes refusal the correct result of an empty scope | Never a `connect.Error`. Never an error metric. Never an alert. It is a *message*, delivered on the thread, with a stated reason |
| **A dropped redelivery** | `OT-P0-005` de-duplicates at ingress. The transport retried; we already have it | Never surfaced to the sender, never retried, never logged at error level. Count it — a rising rate means a transport is unhappy — but as its own signal |
| **Silence from speaking discipline** | `OT-P1-003` makes silence correct in a group with more than one human | Never an error. It leaves a quiet trace on the thread so a person can tell it from a dead bot |
| **A suppressed agent-to-agent turn** | `OT-P1-004`'s loop breaker working exactly as designed | Never an error, and never a retry — a retry here is the loop the breaker exists to stop |

**The rule:** an outcome the system *chose* is not an error, however negative it
looks. Only an outcome the system *could not complete* is.

If any of these is modelled as an error, three things break: the operator's
overview fills with alarms about correct behaviour, error-rate alerting becomes
useless, and a retry path may be attached to something that must never retry.

## Where An Error Surfaces

The transport that delivered a message is where its failure belongs, because the
person waiting for the answer is in the thread and not in a log.

| Failure | Surfaces on the thread | Surfaces in the console | Fails the process |
|---|---|---|---|
| Adapter transient send failure | Yes — after backoff is exhausted | Yes, on channel health | No |
| Adapter terminal send failure | **Yes, always** | Yes | No |
| Turn dispatch failure (`agent-manager` unreachable) | Yes — stated as unavailable, not as a refusal | Yes | No |
| Ambiguous address-to-agent binding | Yes, to the owner only | Yes | No |
| Media exceeding the descriptor's declared limit | Yes — **before the send is attempted** | No | No |
| Invalid channel descriptor | No thread exists yet | Yes | **Yes — at boot** |
| Storage unavailable | Yes, once reachable | Yes, via health | No |

Two deliberate asymmetries:

- **An invalid descriptor fails startup**, naming the file and the field
  (`OT-P0-001`). A registry silently missing a channel is worse than a process
  that refuses to start, because the channel simply never answers and nobody
  knows why.
- **An ambiguous binding is rejected, never guessed** (`OT-P0-007`). Choosing
  one of two candidate agents would route a private conversation to the wrong
  agent, which is unrecoverable in a way a rejection is not.

## Transient Versus Terminal

Every adapter classifies its own failures, because only the adapter knows its
transport. The contract is that it returns one of two kinds and never leaves the
caller to guess from a string.

- **Transient** — rate limited, timeout, 5xx, socket dropped. Backed off per the
  descriptor's declared `rate_per_min`, retried, and shown on the thread as
  retrying with the next attempt stated. Exhausting retries converts to terminal.
- **Terminal** — unknown recipient, revoked credential, payload rejected,
  message too large for the declared limit. Never retried. Stated on the thread
  immediately.

Retrying a terminal failure spends metered inference on work that cannot
succeed, so the classification is a cost control as well as a correctness one.

## Proto-Typed Operations

Unchanged from the template. Proto-typed UI, CLI, and inter-scenario calls use
Connect-RPC, and errors move through three layers:

1. Domain code returns typed sentinels — `<domain>.ErrInvalid<Entity>`,
   `<domain>.Err<Entity>NotFound`.
2. The transport edge maps them to `connect.Error` in
   `internal/<domain>/service_error_mapping.go`, kept beside the domain's
   service layer.
3. The UI receives `ConnectError`, maps `ConnectError.code` to an `errors.<code>`
   i18n key through `ui/src/lib/errorMessage.ts`, and renders localized copy.

The CLI uses the same `connect.Error` values through cli-core. When CLI i18n
arrives it must use the same code names as the UI catalog rather than
string-matching messages.

## Sentinel Mapping

Template rows, plus the sentinels this scenario's domains add.

| Domain error | Connect code | UI i18n key |
|---|---|---|
| `ErrInvalid<Entity>` | `invalid_argument` | `errors.invalid_argument` |
| `Err<Entity>NotFound` | `not_found` | `errors.not_found` |
| Unknown service or repository error | `internal` | `errors.internal` |
| `channels.ErrDescriptorInvalid` | *(boot only — never served)* | — |
| `channels.ErrChannelUnavailable` | `failed_precondition` | `errors.channel_unavailable` |
| `channels.ErrLimitExceeded` | `invalid_argument` | `errors.media_over_limit` |
| `threads.ErrAmbiguousBinding` | `failed_precondition` | `errors.ambiguous_binding` |
| `turns.ErrBudgetExhausted` | `resource_exhausted` | `errors.budget_exhausted` |
| `trust.ErrScopeEmpty` | *(not an error — see above)* | — |

Two rows are deliberately blank. `ErrDescriptorInvalid` never reaches a
transport because it fails the process at boot. `ErrScopeEmpty` never reaches a
transport because a refusal is a message, not a fault — it is listed here only
so nobody adds a mapping for it later.

## Multipart REST Exceptions

Unchanged from the template, and load-bearing here because media is a first-class
message part.

Opaque file bytes are not proto payloads. Upload endpoints use REST multipart for
bytes and return proto-typed metadata, with a stable error envelope through
`internal/httpx.WriteError`; the UI maps `ApiError.code` through the same
`errorMessage(...)` utility as Connect errors.

- Connect-RPC for anything proto can describe.
- REST multipart for file bytes.
- Proto metadata responses for REST upload results.

Do not introduce a second general JSON transport for internal scenario calls. If
the payload is structured and Vrooli-owned, add a proto service method.

**One scenario-specific rule:** an upload is validated against the *destination
channel's* declared limits before it is accepted, not after. A file accepted here
and rejected by the transport later has already cost the owner an upload and told
them nothing useful.

## Cross-References

- `docs/concepts/FLOWS.md` — the flows these outcomes occur in
- `docs/operations/OBSERVABILITY.md` — which of these are counted, and how
- `docs/internal/SECURITY.md` — why refusals are fail-closed by construction
- `docs/internal/SEAMS.md` — the adapter contract that classifies transient versus terminal
- `PRD.md` — `OT-P0-001`, `OT-P0-005`, `OT-P0-007`, `OT-P0-010`, `OT-P1-003`, `OT-P1-004`
