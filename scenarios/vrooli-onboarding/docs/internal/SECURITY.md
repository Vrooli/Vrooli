# Security

## Credential values

A value crosses exactly one boundary: a request body or standard input, straight
to the credential authority. It appears in no response, log line, request URL,
command argument, operator-state field, browser store, or CLI output — at any
point, in any form, on any tier.

The wizard shows configured/unconfigured status and the descriptor's declared
purpose and obtain link. It never reads a value back, because there is no read
path to read one with.

Enforced by `ONB-CRED-NO-DISCLOSURE`, which asserts the absence across every
egress rather than only the response body, so a later refactor that starts
logging request bodies fails a test.

## Privilege

Apply escalates privilege only where a manifest declares it, and only after the
operator consents to that specific safeguard with its `privilege` and `risk`
visible. Consent is per-safeguard; there is no blanket elevation.

Safeguard configuration is validated against the safeguard manifest's own schema
at the write boundary, so an invalid value never reaches durable state and never
reaches a host command.

## State the wizard must not touch

`trust_posture` selects token, break-glass, node-execution, and JWKS-cache
defaults for the whole install. `core` grants control-plane fallback protection
to a named scenario set. Both live in the document onboarding writes, and neither
is an onboarding decision.

The write path is therefore incapable of touching them: it merges named fields
into the loaded document rather than serializing a struct over it. Losing either
is silent, security-relevant, and unrecoverable under the schema's
`additionalProperties: false`, so this is a structural guarantee rather than a
review convention.

## Headless hosts

Every native credential store needs a logged-in graphical session. A VPS, a CI
runner, and a headless bundle host have none, and they are the **default**
condition for remote onboarding rather than an edge case.

There, the encrypted file store is the authority and must be initialized before
provisioning. A reachable TPM holds the wrap open across reboots; without one,
the store needs one unlock per login session. An unreachable native store never
silently falls back to the encrypted one.

## Transport and headers

The API sets `Strict-Transport-Security`, `X-Content-Type-Options: nosniff`,
`X-Frame-Options: DENY`, and disables the legacy XSS auditor, at one middleware
boundary so every handler — including errors and health probes — is covered.

The bundled desktop runtime speaks over an authenticated loopback channel only.

## Threat notes

| Concern | Mitigation |
|---|---|
| Value in argv leaks through the process table and shell history | The CLI rejects a value-bearing flag; stdin only |
| A hand-edited state file disables a system-required scenario | The manifest wins for `system_required`; the operator field is ignored |
| A stale binary truncates a newer state document | Merge-patch writes preserve unmodelled fields |
| An invalid write bricks configuration | Schema validation precedes the write; a rejection leaves the document untouched |
| Concurrent writes lose a decision | Merge happens under a write lock; disjoint patches both survive |
| A tier without a catalog looks like a broken host | Typed degraded state naming the missing catalog, not a 500 |
