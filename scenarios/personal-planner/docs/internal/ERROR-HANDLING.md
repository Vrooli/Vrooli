# Error Handling

## Shared contract

Read the [Error Handling shared guide](/scenarios/template-manager/docs/internal/ERROR-HANDLING.md). Template Manager owns
the common contract and its implementation references.

## Scenario details

**DESIGN-STAGE.** The error model below is the *planned* contract from
the implementation plan §20.5. No product domain code exists yet; only
the `health` infrastructure domain and the fenced `notes` worked example
are real. The typed codes, conditions, and recovery behaviors are the
target this scenario builds against — treat them as the design spec, not
a description of shipped behavior. See
[`../concepts/FLOWS.md`](../concepts/FLOWS.md) for the flows each error
guards and [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md)
for the dependency failure modes several of these codes surface.

Errors travel as typed codes over the Connect error envelope; the same
code drives UI recovery, CLI exit behavior, and source/consumer client
handling. A command never invents a partial success: a failed mutation
returns a failed state and preserves the user's input rather than
claiming a save.

### Typed error codes and recovery

| Code | Typical condition | Caller / UI recovery |
|---|---|---|
| `VALIDATION_FAILED` | Invalid date, effort range, or unsupported interval at the boundary. | Show a field-level error; preserve the values the user entered; do not discard the draft. |
| `FORBIDDEN` | The subject is not authorized for the record (INV-01). | Follow host policy; **do not leak existence across tenants** — an unauthorized subject learns nothing about whether the record exists. |
| `NOT_FOUND` | The record is unavailable to this subject. | Same as `FORBIDDEN`: follow host policy without leaking existence across tenants. `FORBIDDEN` and `NOT_FOUND` are deliberately indistinguishable to an unauthorized caller. |
| `REVISION_CONFLICT` | A concurrent edit changed the expected entity/plan revision. | Fetch the current record, show the relevant difference, and offer retry. Never win by incrementing a local timer. |
| `PROPOSAL_STALE` | A material input (estimate, date, dependency, availability, busy interval) changed after the proposal was generated. | Keep the old proposal inspectable; generate an updated proposal. Never a partial apply under new assumptions (flow: Proposal → apply). |
| `CONSTRAINT_VIOLATION` | Protected time, a source restriction, a lock, or a dependency forbids the placement. | Identify the permitted explanation and the available alternatives; do not silently relax protected time. |
| `SESSION_ALREADY_ACTIVE` | A concurrent start, or another current session, would create a second running exclusive focus session (INV-09). | Offer resume, end, or switch through a revision-checked command; the database guard, not a preflight `SELECT`, is authoritative. |
| `SOURCE_UNAVAILABLE` | A source-owned action cannot be confirmed. | Show a pending/failed source action; **never display a false completion** for source-owned state. |
| `SOURCE_CONSTRAINT_STALE` | The source constraint revision an allocation was validated against is too old for the action (fixture F08). | Refresh constraints or leave the action unapplied; apply leaves accepted allocations unchanged and requests refreshed inputs. |
| `PROVIDER_REAUTH_REQUIRED` | External-provider credentials expired or were revoked. | Preserve last-known busy facts and show a reconnect path; never treat the failure as newly free time. |
| `INSUFFICIENT_INPUT` | Unknown effort or unknown release time prevents an honest forecast (INV-08). | Request the specific missing value or show a partial result labeled as such; unknown is never coerced to zero. |
| `LIMIT_EXCEEDED` | A recurrence-expansion, query, import, or proposal bound was hit. | Explain the bound and offer a smaller scope. |
| `RATE_LIMITED` / `TEMPORARILY_UNAVAILABLE` | Infrastructure or provider delay/backpressure. | Respect retry metadata (bounded exponential backoff with jitter); keep the current accepted state visible while retrying. |

### Message-safety rule

User-facing messages never expose raw provider errors, SQL text, private
source payloads, secrets/tokens, or hidden record IDs. Instead:

- return a **safe diagnostic code** plus a **request ID** to the caller,
- log the safe code + request ID (and internal detail) server-side for
  investigation, and
- redact query parameters where a token could appear.

This applies everywhere, but is load-bearing for two paths in
particular: provider adapter failures (a raw Google/Graph error must
become `PROVIDER_REAUTH_REQUIRED` or `RATE_LIMITED`, never a leaked
stack) and shared-viewer responses (a private cause becomes "Available
time changed," never the title, attendees, or location — INV-12). See
[`SECURITY.md`](SECURITY.md) for the credential and privacy posture and
[`../concepts/DATA.md`](../concepts/DATA.md) §"Privacy Notes" for the
secrets-never-in-logs rule.
