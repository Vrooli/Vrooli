# Testing — Scenario Authenticator

## Shared guidance

- [Test authoring standard](/docs/testing/UNIT-TEST-AUTHORING.md): boundaries,
  fixtures, and independently justified expectations.
- [Execution and validation scope](/docs/TESTING.md): focused checks and Test Genie.
- [Shared harness recipes](/scenarios/template-manager/docs/internal/TESTING-RECIPES.md): API, UI, CLI,
  cancellation, workflow replay, and coverage configuration.

The recipes describe template mechanics. This guide owns local behavior, test
prerequisites, fixtures, and exceptions; local test sources and configuration
identify the helpers and gates this scenario currently uses.

## Scenario-specific testing

Existing local entry points (choose the domain test for the behavior you change):

- [api/handlers/health/handler_test.go](../../api/handlers/health/handler_test.go)
- [ui/src/App.test.tsx](../../ui/src/App.test.tsx)
- [ui/src/features/health/HealthCard.test.tsx](../../ui/src/features/health/HealthCard.test.tsx)
- [ui/src/layout/AppShell.a11y.test.tsx](../../ui/src/layout/AppShell.a11y.test.tsx)
- [cli/app_test.go](../../cli/app_test.go)

## TL;DR — the canonical examples

These files are the source of truth. When in doubt, copy their shape:

- **API**: `api/handlers/health/handler_test.go` — table-driven, real
  middleware via `httpx.NewLiveServer`, fake pinger from `mocks/`,
  typed-proto decode via
  `assertx.MustUnmarshalProto[healthv1.Response]` (the wire shape lives
  in `packages/proto/schemas/scenario-authenticator/v1/health/health.proto`;
  assert on typed proto fields, not `map[string]any` chains). For
  endpoints whose wire shape isn't in proto yet, `MustDecodeJSON[T]`
  is the fallback — but adding the proto first is the right move.
- **UI composition**: `ui/src/App.test.tsx` — smoke-only composition
  test. App composes shell + features; feature behaviour belongs beside
  the feature.
- **UI feature**: `ui/src/features/health/HealthCard.test.tsx` —
  `renderWithProviders`, factory data, inline `vi.mock` factory
  closure, cimode assertions, and real-locale assertions.
- **UI a11y**: `ui/src/components/AppShell.a11y.test.tsx` and
  `ui/src/features/health/HealthCard.a11y.test.tsx` — shell and feature
  accessibility are tested at their ownership boundary.
- **CLI**: `cli/app_test.go` — smoke gate (NewApp, --version, --help).
  When domain commands arrive, extend with `clitest.NewAPIServer` +
  `clitest.CaptureStdout` from `cli/internal/testutil/`.

If your test doesn't look like one of those three, ask why before
shipping.

## Auth test strategy by shipped and deferred domain

> **Status: live testing contract.** Tests for shipped auth domains are
> present and tied
> each cluster to the requirements registry (`requirements/`, IDs
> `REQ-P0-001`…`REQ-P2-007`). The *mechanics* (live server, fakes,
> proto decode, coverage gates) are the template patterns in the sections
> below — this section says which auth-specific behaviors those patterns
> must prove. Two clusters are flagged **MUST-HAVE**: the carried-over
> crypto-invariant regression tests and the cross-realm-rejection test.
> A gap in either is a fleet-wide security defect, not a missing edge case.

### Unit (crypto + policy in isolation)

Pure logic, no DB, no network. Substitutes the `SigningKeyProvider`,
`RealmResolver`, hot-state `Store`, and `Clock` fakes ([`SEAMS.md`](SEAMS.md)).

| Cluster | What it proves | REQ |
|---|---|---|
| **RS256 sign/verify round-trip** (MUST-HAVE) | A token signed by the provider verifies; a tampered token fails. | `REQ-P0-002` |
| **`alg=none` / `alg=HS256` rejection** (MUST-HAVE) | Crafted tokens with the algorithm downgraded are rejected — algorithm-confusion defense. The verifier asserts the signing method is RSA before trusting the key. | `REQ-P0-002` |
| **JWKS round-trip** (MUST-HAVE) | The published JWK (`kty`/`use`/`alg`/`kid`/`n`/`e`) reconstructs a public key that verifies a freshly-signed token — i.e. an RP that consumes JWKS can verify locally. | `REQ-P0-002` |
| **Claims contract** (MUST-HAVE) | Minted tokens carry exactly `user_id`/`sub`, `email`, `roles`, `iss: scenario-authenticator`, `aud` — the carried-over shape device-sync-hub verifies unchanged. | `REQ-P0-002`, `REQ-P0-012` |
| **Argon2id hashing** | Hash + verify round-trips; wrong password fails; documented cost params; output is a hash, never the plaintext. | `REQ-P0-004` |
| **Refresh reuse-detection state machine** (MUST-HAVE) | Rotating a refresh token invalidates the old one; presenting a rotated (reused) token revokes the **whole family** and emits an audit event. Drive the state machine directly. | `REQ-P0-003`, `REQ-P0-007` |
| **`aud` accept/reject** (MUST-HAVE) | A token whose `aud` matches the verifying realm is accepted; a mismatched `aud` is rejected — at both issuance-stamp and verify time. | `REQ-P0-008` |
| **Rate-limit counters** | The limiter trips at the configured threshold and account lockout engages; counters use the hot-state `Store` fake for shared-state behavior. | `REQ-P0-006` |
| **RBAC policy** | Role/scope checks allow/deny correctly; under-privileged principals are denied at the policy layer. | `REQ-P0-009`, `REQ-P1-005` |

### Integration (real SQLite + configured hot-state store + an in-process stub RP)

Full chain over a real storage seam and a real hot-state implementation, with a test that
*plays the Relying Party* via the static discovery resolver
([`SEAMS.md`](SEAMS.md)).

| Cluster | What it proves | REQ |
|---|---|---|
| **End-to-end identity chain** | register → login → issue → **RP fetches JWKS and verifies locally** → refresh-rotate → revoke. The local-verify step uses the in-process stub RP, never a callback to `/validate`. | `REQ-P0-001`, `REQ-P0-002`, `REQ-P0-003`, `REQ-P0-012` |
| **Cross-realm token rejection** (MUST-HAVE) | Mint a token in realm A; present it to realm B's verifier; it is rejected. Two fixed realms via the `RealmResolver` fake. A misconfig here is a cross-tenant token leak ([`SECURITY.md`](SECURITY.md)). | `REQ-P0-008` |
| **Sessions + revocation** | Session list reflects logins; the generated `SessionsService` per-session revoke and "log out everywhere" operations actually invalidate sessions. | `REQ-P0-005` |
| **OAuth callback** (P1) | CSRF `state` is generated, one-time-use, validated and deleted on callback; account linking works; provider outbound goes through the `Doer` fake. | `REQ-P1-003` |
| **device-sync-hub forwarder** | The live consumer's same-origin forwarder, pointed at the Connect surface, completes the first-run owner-bootstrap flow unchanged — the migration gate that blocks P1. | `REQ-P0-012` |
| **Storage seam + hot state** | Identity records persist in SQLite via the seam (additive migrations, no shared Postgres); sessions/revocation/CSRF/rate-limit use the configured hot-state store, with shared Redis required for multi-replica tests. | `REQ-P0-011` |

### Business / BAS (UI flows)

Browser-substrate (BAS) playbooks over the three UI audiences (PRD §UX).

| Cluster | What it proves | REQ |
|---|---|---|
| Sign-in + registration (hosted login) | The hosted login/consent screens complete sign-in and registration; enumeration-safe error copy renders. | `REQ-P0-001`, `REQ-P1-008` |
| MFA enrollment + challenge (P1) | TOTP enrollment, challenge, and recovery-code flows are keyboard-accessible and announce state. | `REQ-P1-002` |
| Session review + revoke (self-service) | A user reviews active sessions and revokes one / all. | `REQ-P0-005`, `REQ-P1-008` |
| Admin management (admin console) | Realm/user/role/session/audit management with destructive-action confirmation gates. | `REQ-P1-007`, `REQ-P1-001` |

### Performance (stateless-verify benchmark)

| Cluster | What it proves | REQ |
|---|---|---|
| **Stateless local verify benchmark** | RP-side token verification is sub-millisecond, in-process, and makes **zero** network calls — the scale lever ([`PERFORMANCE.md`](PERFORMANCE.md)). | `REQ-P0-002` |
| Login / refresh latency | Argon2id login latency is the expected dominant cost; watch for *regressions*, not absolute slowness. | `REQ-P0-001`, `REQ-P0-004` |

### The carried-over-invariant regression suite (do not let it rot)

The crypto core is a **verbatim port** (PRD Appendix C), so its tests are
*regression* tests against a known-good contract, not exploratory tests of
new code. They must stay green on every change because a live consumer
(device-sync-hub) depends on the exact shape: RS256-only, JWKS-publishable,
`user_id`/`sub`/`email`/`roles`/`iss`/`aud` claims, persisted-keypair
stability. Treat any change that touches these as requiring the full
invariant suite to pass before merge.
