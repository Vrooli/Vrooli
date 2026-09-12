# Temporal Flows — Vrooli Bridge

## Checkpoint Flows

Machine enrollment is durable and restart-safe at these boundaries:

1. Create Machine intent and locators under optimistic version control.
2. Create EnrollmentAttempt with immutable input snapshot and correlation id. A retry names a terminal `retry_of_attempt_id`, creates a new attempt/correlation, and never reopens the predecessor.
3. Verify SSH trust and persist only trust reference/fingerprint state.
4. Issue pairing against that correlation; retain no plaintext code.
5. Redeem pairing, durably associate Node lineage, then record credential/code
   consumption atomically where storage permits or through a reconciled saga.
6. Install/start the service and observe Presence. A failure here leaves the
   Machine, attempt, and Node correlated for typed repair/retry.
7. Record the immutable terminal attempt result.

After API restart, the reconciler loads non-terminal attempts, revalidates each
checkpoint postcondition, and either resumes a safe stage, converges a known
pairing result, or stops with an actionable repair status. It never resumes by
parsing a human diagnostic or discovering identity from a locator.

## Failure Ladder

| Failure point | Durable state expected | Outcome |
|---|---|---|
| Before pairing issue | Machine + failed attempt only | retry creates a new attempt |
| After issue, before redemption | correlation + issued code state | expire/reissue according to typed policy |
| After Registry Node, credential, or code write | correlation identifies partial saga | reconciler converges or exposes repair; no duplicate current Node |
| After pairing, before service install/Presence | Machine + attempt + Node lineage | typed paired-but-not-ready repair/retry |
| Host-key mismatch | trust review required | fail closed; no automatic retry |
| Remote cleanup fails after local revoke | cleanup tombstone pending | retry or explicit acknowledgement |

## Concurrency Rules

- One current Node per Machine is storage-enforced and service-checked.
- The pairing-code burn is single-use; duplicate redemption returns the known
  correlated result or a typed already-used outcome, never a second Node.
- Version conflicts on Machine edits are explicit; callers refresh the composed
  projection rather than overwriting state.
