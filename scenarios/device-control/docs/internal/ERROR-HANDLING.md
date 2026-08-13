# Error Handling — Device Control

This scenario has **two** error systems, and confusing them is the most
likely source of dishonest output.

1. **The disposition vocabulary** — how the scenario reports what a device or
   strategy can and cannot do. This is the primary contract and is described
   first, because most "errors" here are not errors at all.
2. **The transport error path** — Connect-RPC sentinels and codes for
   operations that genuinely failed. Standard template shape, described
   after.

A capability that cannot be exercised is usually a *successful answer of no*,
not a failure. Getting that boundary wrong is what turns an honest system
into one that either throws on every unsupported device or, worse, silently
returns success.

## The disposition vocabulary

Every statement this scenario makes about what a device can do resolves to
exactly one of four words.

| Disposition | Meaning | Terminal? |
|---|---|---|
| `available` / `passed` | Probed and proven, right now, on this target. | — |
| `unavailable` | Missing right now. Fixable. **Names the exact prerequisite and a next action.** | No |
| `unsupported` | Categorically impossible on this target. Nothing anyone can do changes it. | Yes |
| `failed` | Attempted and did not hold. Distinct from never having been attempted. | — |

`not_run` is not a disposition. A cell that was never selected is absent, not
reported — inventing a disposition for work that did not happen is how a
release gate ends up counting silence as success.

### Video container validity versus visible content

An MP4 can decode successfully while showing a blank device surface. Native
recordings are therefore checked in two stages: `ffprobe` must find frames and
measure their duration, then a bounded `ffmpeg` sample must contain non-black
content in the body of the display after the status and navigation bands are
excluded. A uniformly near-black body fails the recording step with a named
reason and is withheld from the evidence store. This catches a sleeping,
locked, protected, or otherwise broken capture path without exposing frame
content in the error.

### `unavailable` versus `unsupported`

This is the distinction that gets confused, and the one that matters most.
Apply a single test:

> Is there any action anyone could take that would make this work on this
> target?
>
> **Yes** → `unavailable`, and that action is the next action.
> **No** → `unsupported`.

| Situation | Disposition | Why |
|---|---|---|
| `android-adb` with `adb` off `PATH` | `unavailable` | Install the `android-sdk` resource. |
| `ios-xcuitest` with no Apple enrollment | `unavailable` | Enroll; WebDriverAgent can then be signed. |
| Vision rung with no `ai-gateway` visual route | `unavailable` | The gateway capability is a declared prerequisite (`D-005`). |
| Semantic tree on `ios-mirror` | `unsupported` | Mirroring exposes pixels and synthetic HID. There is no tree to read, ever. |
| Any iOS strategy on a Linux host | `unsupported` | No Apple toolchain exists for Linux. |

**A disposition is scoped to a cell — capability × strategy × device — never
to a capability globally.** "Semantic tree" is `unsupported` on `ios-mirror`
and `available` on `ios-xcuitest` *for the same physical iPhone*. Marking a
capability `unsupported` at the capability level would erase that, and would
make the resolution ladder (`D-004`) unrepresentable.

### Every `unavailable` carries a next action

**An `unavailable` with no next action is a defect, not a valid state.**
[`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) tracks the
count and expects zero. This section is the contract that metric measures.

```
✗  unavailable: capability not present
✗  unavailable: dependencies missing
✓  unavailable: adb not on PATH
   → install the android-sdk resource:
     scenario-dependency-analyzer deps install android-sdk
```

The prerequisite must be *named*, not categorized. "A tool is missing" tells
an operator to go looking; "adb not on `PATH`" tells them what to do. The
same rule applies to a strategy reporting its own probe results — see
[`SEAMS.md`](SEAMS.md#why-describe-and-capabilityprober-are-two-seams).

### A capability gap report is not an error

Checking a flow against a strategy **succeeds** even when the answer is no.

| Call | Outcome |
|---|---|
| "Can this flow run on this device?" → it cannot | **Success**, carrying a structured gap report |
| "Run this flow on this device" → dispatched despite a known gap | **Error** — `failed_precondition` |

The question was answered correctly in the first case. Returning an error
there would force every caller to parse error strings to recover structured
data that the API already has, and would make "this device cannot do X" —
the single most common honest answer this scenario gives — indistinguishable
from a broken request.

The second case is an error precisely *because* the caller was already told.

## Dispositions and Connect codes

A disposition becomes a transport error only when a verb was actually
attempted and could not complete.

| Condition | Connect code | Notes |
|---|---|---|
| Verb dispatched with no lease held | `failed_precondition` | Wrong state; the caller can fix it by acquiring one. |
| Lease held by another consumer | `aborted` | Contention. A later retry may succeed. |
| Lease expired mid-run | `aborted` | The run is void; evidence records where it stopped. |
| Device unreachable (host node offline) | `unavailable` | Retry may succeed once the host returns. |
| Strategy lacks the capability for this verb | `unimplemented` | Not transient. This strategy will never do it. |
| Bounded wait exceeded | `deadline_exceeded` | See below — never extended. |
| Flow dispatched despite a known capability gap | `failed_precondition` | The gap report was available and ignored. |
| Malformed step or unresolvable target intent | `invalid_argument` | |
| Redaction verification failed | `internal` | Detail is deliberately withheld; the detail *is* the leak. |

Connect's `unavailable` and `unimplemented` codes line up almost exactly with
the `unavailable` and `unsupported` dispositions, which makes the mapping easy
to remember. Keep the categories distinct anyway: **a disposition is data the
API returns successfully; a code is a transport failure.** They coincide only
when a verb was attempted.

## Bounded waits fail; they never extend

Every wait is a named policy with an explicit upper bound (`OT-P0-005`). When
a bound is exceeded:

- the step fails with `deadline_exceeded`;
- an evidence chapter records the **policy name, the bound, and the observed
  duration** — an exceeded bound is evidence-visible, never a silent retry;
- **no automatic retry and no automatic extension occurs.**

The failure this prevents is the most common way an automation suite rots:
someone makes a flaky flow green by raising a timeout. Raising a bound is a
real decision, and it must appear as a diff in the flow definition where a
reviewer can see it — not as adaptive behavior inside the executor.

## What an error may never contain

Errors do not pass through `EvidenceSink`, so they never get redaction
verification. That makes the error channel the one path by which device
content could leave without being checked.

**Errors carry identifiers and reasons. Never content.**

Never permitted in an error message, log line, or CLI output:

- frame bytes, or a filesystem path to a capture;
- text read from a device screen by OCR or from a semantic tree;
- clipboard values;
- device logs;
- lease tokens or bridge credentials.

The realistic leak is not malicious. A resolver that fails naturally wants to
say what it searched for and what it saw:

```
✗  could not find target "password" in frame; visible text was
   "Enter code 481920 to continue"
✓  target "login.submit" not resolved at any rung
   (semantic: no tree on this strategy; anchor: no reference captured;
    vision: unavailable — ai-gateway has no visual request kind)
```

The second names the *target identifier* and the *rung outcomes*. It is
strictly more useful for debugging and carries nothing off the device.

## Proto-Typed Operations

Proto-typed UI, CLI, and inter-scenario calls use Connect-RPC. Errors
move through three layers:

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

Because the CLI is the agent-facing control surface (`D-007`), CLI error
output must stay machine-readable in `--json` mode. An agent that has to
regex a human sentence to learn it lacked a lease will eventually guess
wrong.

## Sentinel Mapping

Template defaults:

| Domain error | Connect code | UI i18n key |
|---|---|---|
| `ErrInvalid<Entity>` | `invalid_argument` | `errors.invalid_argument` |
| `Err<Entity>NotFound` | `not_found` | `errors.not_found` |
| Unknown service/repository error | `internal` | `errors.internal` |

Planned scenario sentinels — **none of these exist in code yet**, listed here
so the mapping is settled before the domains land:

| Domain error | Connect code | Owning domain |
|---|---|---|
| `sessions.ErrNoLease` | `failed_precondition` | sessions |
| `sessions.ErrLeaseHeld` | `aborted` | sessions |
| `sessions.ErrLeaseExpired` | `aborted` | sessions |
| `devices.ErrUnreachable` | `unavailable` | devices |
| `strategies.ErrCapabilityUnsupported` | `unimplemented` | strategies |
| `flows.ErrBoundExceeded` | `deadline_exceeded` | flows |
| `flows.ErrRedactionUnverified` | `internal` | flows |

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

## Cross-References

- [`SEAMS.md`](SEAMS.md) — the seams that produce these errors, and the claim-versus-probe split
- [`SECURITY.md`](SECURITY.md) — redaction obligations the error channel must not bypass
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — the signals that measure this contract
- [`DECISIONS.md`](DECISIONS.md) — `D-002` (probed, never inferred), `D-004` (resolution ladder)
- [`../concepts/DOMAINS.md`](../concepts/DOMAINS.md) — which domain owns each error
