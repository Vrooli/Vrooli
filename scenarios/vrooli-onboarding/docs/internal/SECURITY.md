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

The bundled desktop runtime speaks over a loopback channel whose mutation
requests are authenticated by the shared personal-local provider. The runtime
creates an owner-only local session token, passes its path to the API, and the
trusted desktop renderer obtains the token through the origin-checked preload
bridge. The provider grants only the declared onboarding capabilities; neither
loopback location nor the API process's OS user is identity proof.

## Node-local trust

The target interceptor treats an empty or `local` target as the node-local
path. A named target is sent through Bridge, where the owner is established
before the signed frame is created. The agent is the only originator of a
node-local call; browser credentials never cross to a node. Mutation methods
still apply the onboarding operator authorization rule at their Connect
boundary.

## Decision D9 — verified local operator authority

The onboarding API never authorizes a mutation from the transport peer address
alone. In personal-local mode, the shared `api-core/authn` provider verifies a
runtime-owned session token on a loopback request and installs a
provider-neutral human principal with the onboarding capability set. The
desktop browser path is explicit and documented: its preload bridge returns the
token only to a trusted local renderer, and the onboarding Connect transport
adds it automatically. PTY callers use the same token through the supported
`VROOLI_AUTH_LOCAL_TOKEN` environment binding. Remote modes use the configured
verified identity providers. Anonymous, agent, expired, revoked, or
underprivileged callers fail before the domain service runs. A remote target
also requires explicit authorization metadata so the originating principal can
survive Bridge forwarding.

## Threat notes

| Concern | Mitigation |
|---|---|
| Value in argv leaks through the process table and shell history | The CLI rejects a value-bearing flag; stdin only |
| A hand-edited state file disables a system-required scenario | The manifest wins for `system_required`; the operator field is ignored |
| A stale binary truncates a newer state document | Merge-patch writes preserve unmodelled fields |
| An invalid write bricks configuration | Schema validation precedes the write; a rejection leaves the document untouched |
| Concurrent writes lose a decision | The operator-state authority merges under a process lock and a native cross-process sidecar lock; disjoint patches both survive |
| A tier without a catalog looks like a broken host | Typed degraded state naming the missing catalog, not a 500 |

## Current scanner receipt

On 2026-09-09, `security-health validate scenario vrooli-onboarding --json`
completed with `VALIDATION_STATUS_PASSED`. Gitleaks reported zero findings;
the earlier password-field matches in `StepReadiness.tsx` were scanner matches
on the `PasswordInput` component name, not credential values, and were removed
by aliasing the same RCL component as `SecureValueInput`. No scanner suppression
was added.

The latest receipt remains at security-health L1 Foundation with 336 advisory
findings (113 warnings and 223 informational findings): 70 gosec, 10
govulncheck, 52 pnpm-audit, and 204 OSV findings. The dependency governance
reindex completed after the onboarding UI lockfile remediation and reports zero
HIGH or CRITICAL dependency gaps; 19 lower-severity advisory gaps remain. These
scanner findings are retained as owner-triage debt and must not be described as
clean-launch clearance. Live remote peer-identity, host-key rotation, and
platform-specific authority exercises remain separate acceptance gates.
