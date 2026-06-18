# Flows — Scenario Authenticator

This document is the canonical workflow and state-transition map for
the scenario. Use it when behavior depends on ordered states, retries,
cancellation, stale completion, background jobs, polling, or mutually
exclusive UI modes.

## Purpose Of This Document

Use this document to answer:

- Which user/system workflows matter?
- Which workflows have explicit states and events?
- Which transitions are illegal?
- Which tests prove workflow correctness?
- Which flows are known but not modeled yet?

Plain CRUD with no meaningful ordering constraints does not need a
workflow model.

## Flow Inventory

> **Status: documentation-first orientation.** Every flow below is
> **planned (pre-implementation)** — at maturity Level 1 (inventoried),
> targeting the level it lists when its domain is built. No formal Quint
> model exists yet. `health` is a stateless reporting domain and ships
> no workflows.

| Flow | Domain | Trigger | Outcome | Statefulness | Validation |
|---|---|---|---|---|---|
| Account registration | identity | User/RP submits email + password to register. | Realm-scoped user created with Argon2id credential; no account-existence leak. | Validation + uniqueness within realm; no ordering states. | Target Level 2–3. |
| Sign-in (+ optional MFA) | identity → tokens (+ mfa) | User/RP submits credentials. | Access+refresh token issued, session created; or MFA challenge gate. | Stateful when MFA: `credentials_ok → mfa_pending → authenticated`. | Target Level 5 (MFA gate). |
| Refresh-token rotation + reuse detection | tokens | Client presents a refresh token. | New access+refresh pair (rotated); a **reused** token revokes the family. | Stateful family lifecycle with terminal `revoked`. | Target Level 5. |
| JWKS local-verify (RP path) | tokens (issuer) / RP (verifier) | RP needs to verify a presented token. | RP verifies RS256 signature locally against cached JWKS; `aud`/realm checked. | Cache lifecycle on the RP side; **no per-request call back to the IdP**. | Target Level 3 (RP-side cache). |
| Session revoke / log-out-everywhere | sessions | User/admin revokes a session or all sessions. | Session(s) dropped from Redis; subsequent use rejected. | Terminal revoke; idempotent. | Target Level 2–3. |
| OAuth social callback | federation | External IdP redirects to the callback with code + state. | CSRF state validated; external identity linked; tokens issued. | Stateful: `state_issued → callback_received → linked/authenticated`. | Target Level 4–5 (CSRF + linking). |
| Password reset | identity | User requests reset; later submits new password with token. | Single-use, expiring reset token consumed; credential rehashed. | Stateful: `requested → token_issued → consumed/expired`. | Target Level 4. |
| Attachment upload (**template example, remove**) | notes | User/CLI uploads a file for a note. | Blob stored, metadata persisted. | Stateful upload request with failure paths. | Template Level 5 reference. |

## Flow Details

Document each real flow here with its owner domain, trigger, inputs,
ordered steps, outputs, failure modes, retry/cancel behavior, tests, and
generated subpackages. The worked example below shows the expected shape.
The auth flows here are the **target** design; none are implemented yet.

### Account registration

- Owner domain: identity.
- Trigger: a `Register` Connect RPC, forwarded same-origin by an RP's
  API (never a cross-origin browser call).
- Inputs: realm (defaults to the single default realm at P0), email,
  password.
- Steps:
  1. Resolve the target realm and load its password policy.
  2. Validate email format and password against realm policy.
  3. Enforce uniqueness **within the realm** (the same email in another
     realm is a distinct principal).
  4. Hash the password with **Argon2id** at the documented cost.
  5. Persist the user + credential hash (no plaintext at rest).
  6. Record a sign-up audit event; optionally start email verification.
- Outputs: created user (or token pair, if the realm auto-signs-in).
- Failure modes: invalid input, policy violation, email already in realm.
- Anti-enumeration: error relay must not reveal whether an email already
  exists in a way that aids enumeration (PRD OT-P0-001).

### Sign-in (+ optional MFA challenge)

- Owner domain: identity → tokens (and `mfa` when a factor is enrolled).
- Trigger: a `Login` Connect RPC (forwarded same-origin by the RP).
- Inputs: realm, email, password; later an MFA code.
- Steps:
  1. Look up the realm-scoped user; verify the password against the
     Argon2id hash (constant-work on miss to avoid timing/enumeration).
  2. Apply rate limiting + account lockout (brute-force defense).
  3. If MFA is enrolled, return an **MFA challenge** — state is
     `mfa_pending`, no tokens issued yet — and verify the submitted
     TOTP/passkey before proceeding.
  4. Mint an RS256 access token with the carried-over claims (`user_id`/
     `sub`, `email`, `roles`, `iss: scenario-authenticator`, `aud` =
     realm) and a rotating refresh token.
  5. Create a server-tracked **session** in Redis.
  6. Record a sign-in audit event.
- Outputs: access + refresh token pair and a session, or a typed error.
- Failure modes: bad credentials, locked account, failed MFA, rate-limited.
- Illegal transition: issuing tokens while `mfa_pending` (MFA must pass
  first).

### Refresh-token rotation with reuse detection

- Owner domain: tokens.
- Trigger: a `Refresh` Connect RPC presenting a refresh token.
- Inputs: a refresh token (member of a token family).
- Steps:
  1. Look up the token's **family**; reject if the family is revoked.
  2. If the presented token has **already been rotated** (reuse), treat
     it as a compromise signal: **revoke the entire family**, audit it,
     and reject — the client must re-authenticate.
  3. Otherwise rotate: invalidate the presented token, mint a new
     access + refresh pair within the same family.
- Outputs: a fresh access + refresh pair, or a family-revoked error.
- Failure modes: unknown/expired token, revoked family, **reuse**.
- This is the security keystone of OT-P0-003 — a leaked-then-replayed
  refresh token cannot outlive its first reuse.

### JWKS local-verify (the device-sync-hub same-origin path)

This is the architectural keystone: **no cross-origin browser call and
no per-request call back to the authenticator.** The device-sync-hub
forwarder is the reference implementation.

- Owner: `tokens` issues + publishes JWKS; the **RP** verifies.
- Trigger: a browser action in an RP that requires identity.
- Steps:
  1. The browser calls **only its own scenario's API** (same-origin).
  2. That RP API, for sign-in/register, **forwards same-origin** to
     scenario-authenticator — resolving it **by slug**
     (`scenario-authenticator`) via `api-core/discovery`, no hardcoded
     URL/port — and relays the issued token back to its browser (the
     `internal/identity.Forwarder` pattern).
  3. On subsequent requests carrying that token, the RP **verifies it
     locally**: it fetches the authenticator's public key **once** from
     `/.well-known/jwks.json`, caches it, and verifies the RS256
     signature offline. The algorithm is locked to RS256 — `none` and
     HS-family confusion are rejected outright.
  4. The RP checks `aud` matches its realm and **trusts the claims**
     (`sub`/`user_id`, `roles`, `scopes`); it owns its own domain
     authorization from there.
- The authenticator is contacted only to (a) fetch the signing key the
  first time / on rotation and (b) forward an interactive sign-in — never
  on the verification hot path. A briefly-unreachable authenticator does
  not break already-issued sessions.

### Session revoke / log-out-everywhere

- Owner domain: sessions.
- Trigger: a per-session revoke or "log out everywhere" request
  (preserving the carried-over `/api/v1/sessions/{id}` contract, or its
  Connect equivalent in lockstep).
- Steps:
  1. Authorize the caller for the target session(s).
  2. Drop the session(s) from Redis (single id, or all sessions for the
     user on "log out everywhere").
  3. Record a revoke audit event.
- Outputs: revoked confirmation; later token use against the session is
  rejected. Idempotent — revoking an already-gone session is a no-op.

### OAuth social callback (P1)

- Owner domain: federation.
- Trigger: an external IdP (Google/GitHub/Microsoft) redirects to the
  **REST** callback (a non-RPC web standard, `RESTReasonThirdPartyShape`)
  with an authorization code and `state`.
- Steps:
  1. Validate the **CSRF `state`** against the value stored in Redis when
     the flow began (single-use); reject on mismatch.
  2. Exchange the code with the provider for the external identity.
  3. Link the external identity to a realm-scoped user (or create one),
     then issue tokens + a session as in sign-in.
  4. Audit the federated sign-in.
- Failure modes: CSRF mismatch/replay, provider error, link conflict.

### Password reset (P1)

- Owner domain: identity.
- Trigger: a reset request, then a reset submission.
- Steps:
  1. On request, mint a **single-use, expiring** reset token; deliver it
     out-of-band. Response must not reveal whether the email exists.
  2. On submission, validate the token (unconsumed + unexpired),
     re-hash the new password with Argon2id, consume the token, and
     audit the reset.
- Failure modes: expired/consumed/invalid token, weak new password.
- States: `requested → token_issued → consumed | expired`.

<!-- EXAMPLE-DOMAIN:notes START -->
### Example domain — `notes` (removed by `vrooli scenario detemplate`)

The template ships an `Attachment upload` flow on the `notes` domain as a
worked Level 5 temporal-workflow vertical slice. Copy its shape for your
own stateful flows, then remove it.

Add this row to the Flow Inventory above:

| Flow | Domain | Trigger | Outcome | Statefulness | Validation |
|---|---|---|---|---|---|
| Attachment upload | notes | User/CLI uploads a file for a note. | Blob is stored and metadata is persisted. | Stateful upload request with validation and failure paths. | Level 5 workflow tests: matrix, traces, declarative spec, checked Quint model, generated artifacts, and production replay. |

#### Attachment upload

- Owner domain: notes.
- Trigger: multipart upload request from UI or CLI.
- Inputs: note id, file key/name, file bytes, content type, file size.
- Steps:
  1. Parse multipart request.
  2. Validate note id and file metadata.
  3. Store opaque bytes through BlobStore.
  4. Persist attachment metadata through notes repository seam.
  5. Return proto-typed metadata response.
- Outputs: uploaded attachment metadata or typed error response.
- Failure modes: missing note id, missing file, invalid metadata, blob
  write failure, metadata persistence failure.
- Retry/cancel behavior: caller may retry after transport/storage
  failure; duplicate handling belongs to the owning real domain when
  product requirements demand it.
- Tests: `api/handlers/notes/attachments_handler_test.go`,
  `api/internal/notes/attachments_service_test.go`,
  `api/internal/notes/flow/flow_test.go`,
  `ui/src/features/notes/AttachmentUpload.test.tsx`, and
  `ui/src/features/notes/flow/flow.test.ts`.
- Generated subpackages: `api/internal/notes/flow/generated/`
  (`model.qnt`, `artifact.json`, `runtime.go`, `replay.go`) and
  `ui/src/features/notes/flow/generated/` (`model.qnt`, `artifact.json`,
  `runtime.ts`, `replay.helper.ts`).
- Requirements: template starter only.

These example state machines belong in the State Machines table below:

| Domain/Flow | States | Illegal Transitions | Enforcement |
|---|---|---|---|
| notes / attachment upload API | received, bytes_stored, metadata_recorded, failed | metadata before bytes, terminal-state escape, duplicate terminal events | `*.flow.json` contract, generated Quint model, generated formal artifact replay, side-effect cleanup tests |
| notes / attachment upload UI | idle, selected, uploading, succeeded, failed | start before select, stale completion after reset/reselect, retry without file context | `*.flow.json` contract, generated Quint model, generated formal artifact replay, attempt-id stale completion tests |
<!-- EXAMPLE-DOMAIN:notes END -->

## State Machines

List each modeled flow's states, illegal transitions, and how they are
enforced. Plain CRUD with no ordering constraints does not appear here.

These are the **target** state machines for the stateful auth flows;
none are modeled in a `*.flow.json` contract yet.

| Domain/Flow | States | Illegal Transitions | Enforcement (target) |
|---|---|---|---|
| identity / sign-in with MFA | `credentials_pending`, `mfa_pending`, `authenticated`, `failed` | issue tokens while `mfa_pending`; reach `authenticated` from `failed` | `*.flow.json` contract → generated Quint model → replay tests |
| tokens / refresh-token family | `active`, `rotated`, `revoked` | rotate a `revoked` family; accept a `rotated` (reused) token | reuse detection revokes the family; replay tests cover the reuse trace |
| federation / oauth callback | `state_issued`, `callback_received`, `linked`, `rejected` | accept a callback with no/mismatched CSRF `state`; reuse a consumed `state` | single-use Redis CSRF state; replay tests |
| identity / password reset | `requested`, `token_issued`, `consumed`, `expired` | consume an `expired`/already-`consumed` token | single-use expiring token; replay tests |

## Maturity Ladder

Temporal workflows mature in layers. Do not skip the executable layers
to add a standalone formal document.

| Level | Name | What exists |
|---|---|---|
| 0 | Unmodeled risk | Lifecycle behavior exists only inside handlers, components, callbacks, or jobs. |
| 1 | Inventory | The flow is listed here with owner, source links, risk, and next step. |
| 2 | Workflow model | State/status values, event values, `Transition`, and `CheckInvariants` live beside the owning domain or feature. |
| 3 | Matrix + traces | Tests cover every state/event pair and replay representative traces against production transition logic. |
| 4 | Declarative contract | A domain-local `*.flow.json` declares states, events, transitions, invariants, and named traces. |
| 5 | Checked formal model | Quint/TLA+ or an equivalent tool is generated from the contract, checked, and replayed by production tests. |

## Production Shape

Three (Go) or four (UI) files per flow at the top of the feature folder,
plus one `generated/` sibling. Everything in `generated/` is codegen output.

Every flow lives in a `flow/` subdirectory next to its consumer with
conventional file names. API domains that own durable lifecycle state use:

```text
api/internal/<domain>/
  flow/
    flow.json                   # hand: source of truth (schema v6)
    transition.go               # hand: wrapper (package flow)
    flow_test.go                # hand: thin replay delegation (package flow)
    generated/
      model.qnt
      artifact.json
      runtime.go                # package generated
      replay.go
```

UI features that own client-side modes use:

```text
ui/src/features/<domain>/
  flow/
    flow.json                   # hand: source of truth (schema v6)
    transition.ts               # hand: wrapper
    fixtures.ts                 # hand: replay fixtures
    flow.test.ts                # hand: thin replay delegation
    generated/
      model.qnt
      artifact.json
      runtime.ts
      replay.helper.ts
```

Every flow uses the same file names. The `flow/` directory IS the unit;
the contract no longer declares any output paths or module names.

The workflow owns state/status values, events, `Transition`, and
`CheckInvariants`. It should be pure or nearly pure. Effects live
outside the workflow behind seams: repositories, BlobStore, clocks,
timers, HTTP clients, or UI API modules.

The `*.flow.json` contract is the source of truth. Level 5 generated
Quint models, formal artifacts, and Go/TypeScript declarations are
checked-in source artifacts for reviewability, but they are refreshed
and checked by the `flow-verifier` scenario CLI; the
scenario lifecycle runs `make temporal-models` (which calls
`flow-verifier verify check`) before the normal test
suite. A Quint file by itself is not accepted: the model must typecheck,
test, verify named invariants, emit deterministic artifacts, and those
artifacts must replay against the production Go/TypeScript transition
functions.

The generated declarations keep state/event topology and formal
freshness metadata out of hand-maintained test lists. They also provide
pure status-transition helpers generated from the `*.flow.json`
transition matrix. For TypeScript flows, the same declarations can own
the discriminated state/event union shape and replay fixture contract.
Production workflow wrappers call those helpers for abstract validity
and next-status outcomes, while keeping payload validation, side-effect
orchestration, and rich state construction in hand-authored code. API
replay tests get expected paths, hashes, invariants, and generated checks
from `generated/<folder>/runtime.go`; UI replay tests import the same metadata
from `generated/<folder>/runtime.ts`. The generated `replay.{go,helper.ts}`
files own the assertion calls; the hand-authored top-level test simply binds
the wrapper's transition function and the fixtures and invokes
`RunReplay`/`runFormalReplay` once.

Formal artifacts use schema v6 coverage metadata. Matrix completeness,
terminal transition checks, named trace coverage, and generated MBT trace
coverage are separate fields. Do not treat generated trace
`allPairsCovered` as required proof of correctness; replay tests require
the complete transition matrix and named traces, while generated trace
coverage reports how much the model explorer happened to visit.

Schema v6 `flow.json` files carry no path or module information. The
`replay` block declares only `transition.function` (plus
`transition.statusAccessor` for TS or `transition.stateType` /
`transition.statusField` for Go). Everything else is derived from the
flow directory.

Go flows emit `flow/generated/replay.go` and require a hand-authored
`flow/flow_test.go` (package `flow`) that calls `generated.RunReplay`.
TypeScript flows emit `flow/generated/replay.helper.ts` and require a
hand-authored `flow/flow.test.ts` that calls
`runFormalReplay({ transition, fixtures })` at module top level.
`flow-verifier verify check` byte-compares every generated file and runs an
AST-level lint over the hand-authored test, so a silent bypass — missing
import, stubbed transition, or call buried inside a guarded block —
fails the check.

To scaffold a new flow:

```bash
flow-verifier flows new ui/src/features/<feature> --flow-id <flow-id> --lang ts --root .
flow-verifier flows new api/internal/<domain>     --flow-id <flow-id> --lang go --root .
```

The scaffold writes the hand-authored files and immediately runs
`generate`, so `check` is green from the moment it returns.

To add or rename a state/event:

1. Edit the owning `*.flow.json`.
2. Regenerate that flow with `flow-verifier verify run --flow <flow-id>`.
3. Update only payload-specific wrapper branches that need new runtime
   data; the abstract transition table is generated.
4. Update the UI replay fixture module. The generated formal replay fixture
   interface should make missing state/event fixtures a type error.
5. Run `make temporal-models` and the scenario tests.

## Deferred / Unmodeled Flows

| Flow | Risk | Next Step |
|---|---|---|
| All auth flows above | None — these are inventoried (Level 1) but unbuilt; the risk is building them without modeling the stateful ones. | Model sign-in-with-MFA, refresh-family, OAuth callback, and password-reset as `*.flow.json` contracts as their domains land. |
| TOTP/passkey MFA enrollment (P1) | Stateful enrollment + challenge not yet inventoried in detail. | Add when the `mfa` domain is built. |
| Client-credentials grant (P1) | Non-human auth path not yet detailed. | Add when the `apikeys` domain is built. |
| SAML ACS (P2) | Enterprise SSO assertion flow. | Add when the `federation` SAML path is built. |

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — owning domain map
- [`DATA.md`](DATA.md) — persisted state and retention
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — side-effect boundaries
- [`../internal/TESTING.md`](../internal/TESTING.md#temporal-workflow-tests) — matrix and trace testing
