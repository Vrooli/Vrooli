# Identity, Authentication, Desktop, and Entitlement Migration Ledger

Status: implementation delivered within the declared plan boundary; post-execution contract audit and
targeted repairs captured 2026-09-07. This ledger is the implementation matrix for the full identity
migration plan. It records observed behavior, the target contract, and the
evidence that retires each legacy path. A `partial` entry is not a claim that
the target behavior is shipped.

## Contract owners

| Boundary | Owner | Must not be moved into |
| --- | --- | --- |
| Person, service, machine, and agent principal verification | `scenario-authenticator` plus configured external providers, composed by `packages/api-core/authn` | A relying scenario or browser UI |
| Canonical principal shape and failure/status vocabulary | `packages/api-core/identity` | Scenario-local claim structs |
| Coarse capability vocabulary and catalog | `packages/api-core/scopecatalog` from governed manifests | Authenticator domain-policy code |
| Scenario/object/mutation authorization | Each relying scenario | The identity provider or LPBS |
| Agent attenuation and provenance | `agent-manager` | Human identity claims |
| Desktop shell-to-supervisor process credential | `scenario-to-desktop` runtime | Human authentication |
| Business accounts, subscriptions, downloads, and commercial entitlements | LPBS | `scenario-authenticator` |
| Reusable account/security presentation | `react-component-library` | Scenario authorization decisions |
| Cloudflare Access verification and shared gated-UI policy | Active `cloudflare-access-human-authority-shared-gated-ui` plan | This plan |

## Consumer matrix

| Consumer | Current observed provider / credential | Current audience or scope behavior | Current browser/storage behavior | Target migration | Status and retirement evidence |
| --- | --- | --- | --- | --- | --- |
| `scenario-authenticator` | Owns local account, password, MFA, session, machine exchange, RS256/JWKS, and refresh rotation | Realm audience is persisted and minted; default is `scenario-authenticator:default`; authorization service owns coarse grants | Hosted login uses HttpOnly same-origin access/refresh cookies; i18n/preferences remain unrelated UI state | Provider authority plus secure same-origin session or code/device flow; publish resource-audience migration and capability catalog | `partial`; cookie, first-admin, scope-RBAC, refresh, audit, and audience tests pass; resource-audience and broader management evidence remain |
| `vrooli-bridge` | Normal bearer verification delegates to `api-core/authn`; enrolled/break-glass paths remain distinct | Frozen default authenticator audience and bridge-specific scope handling | API/CLI credentials are process or client credentials; no browser bearer persistence in the provider package | Canonical principal plus explicit resource audience; preserve local enrollment and break-glass as distinct credential kinds; use shared scope grammar | `partial`; bridge auth tests and private-parser scan pass; resource-audience conformance remains |
| `agent-manager` | Owner-token verification now consumes `api-core/authn.TokenVerifier`; scenario-local agent claims/attenuation remain domain-owned | Owner scopes are narrowed by agent profiles and requested scopes; agent identity is separate from human authority | Run identity is carried as server-side/provenance state; no browser token persistence in the inspected API seam | Canonical owner principal plus explicit agent provenance; attenuation must never create human authority | `partial`; wiring/orchestration tests and focused consumer conformance pass; unrelated full-suite baseline failures remain |
| `device-sync-hub` | Normal bearer verification delegates to `api-core/authn` | Lifecycle-injected `scenario-authenticator:device-sync-hub`; compatibility default remains only for direct unconfigured construction | Device token remains local pairing state; owner JWT now uses an HttpOnly same-origin cookie and legacy localStorage values are migrated out | Canonical verifier with a `device-sync-hub` resource audience and explicit device/owner distinction | `partial`; focused auth/cookie/migration tests and same-`kid` rotation pass; configured audience path is covered; compatibility fallback and live browser evidence remain |
| Git Control Tower | `policygate` normal authenticator verification delegates to `api-core/authn`; Cloudflare/provider composition remains adjacent-plan-owned; personal-local is a loopback/OS-user provider | Manifest separates Cloudflare application audience from `scenario-authenticator:git-control-tower`; mode selects provider set | Browser login now receives an HttpOnly same-origin cookie; UI has no bearer-token cookie/localStorage writer | Canonical authn provider, explicit resource audience transition, and preserved domain/human-control policy | `partial`; adversarial policy, personal-local, mode-matrix, canonical verifier, UI build/typecheck tests pass; live desktop/browser evidence remains |
| `portal` | Context/chat handlers now consume `api-core/authn.TokenVerifier` and `identity.Principal` | Human kind, verified state, subject, expiry, and lifecycle-injected resource audience are checked before account binding | API interceptor accepts request credentials; no password store in portal | Canonical principal adapter; portal remains a relying party and never stores LPBS/local account credentials | `partial`; `go test ./...` passes for portal API; no owneridentity imports remain in portal API |
| LPBS | Owns magic-link user auth, admin auth, customer/business account, subscription, download, and entitlement services | LPBS consumer JWT and signed entitlement lease use LPBS-owned keys/audiences; local Vrooli identity is not email-matched | Browser consent issues a one-use scoped desktop-link code; the desktop template redeems it only with a verified local proof and persists only the signed lease, never a website token | Keep LPBS commercial authority; explicit scoped local-account link and entitlement lease consumption without copying website sessions | `partial`; durable account/membership selection, link/revocation, signed scoped lease, exact resource audience/scope validation, browser consent/client redemption, and wrong-installation/expiry/replay tests pass; the generated desktop template now wires the declared Unix-socket authenticator proof, while offline revocation is bounded by lease expiry plus status refresh and live multi-installation evidence remains |
| Desktop runtime/templates | Supervisor loopback bearer authenticates Electron to supervisor; template has auth manager and encrypted credential/lease paths | Bundle/resource route is separate from human identity; four documented modes need typed manifest/runtime enforcement | Personal-local must work offline; desktop tokens must use credential authority/native secure storage | Add manifest-declared mode/provider/audience/status/recovery semantics and explicit setup/recovery for `personal_local`, `local_multi_user`, `remote_vrooli`, and `shared_provider` | `partial`; runtime and template tests pass; mode profiles, protected selection/rollback, atomic non-secret state, and shared-lease refusal are implemented; local-provider bootstrap and live mode evidence remain |
| Deployment/bundle manifests | Service manifests have versioned authentication profiles; GCT now separates Cloudflare and scenario audiences | `VROOLI_AUTH_MODE` selects provider composition; personal-local avoids authenticator and Cloudflare runtime binding | Non-secret metadata is manifest state; secrets remain credential-authority state | Versioned authentication profile with mode, resource audience, public routes, capability namespace, agent eligibility, and recovery behavior | `partial`; service/CLI schemas, mode-provider matrix, six reference profiles, shared capability matching, and startup dependency checks pass; desktop bundle projection and live mode evidence remain |
| React Component Library | Account/monetization/auth/entitlement assets exist in catalog/library | Components must be presentation/state surfaces only | No authority should be encoded in component visibility | Standardize account, session, machine, link, entitlement, sign-in, refusal, offline, and recovery surfaces with API-enforced mutation tests | `partial`; reusable assets are present; authenticator and LPBS adoption/accessibility evidence remains |

## Contract decisions and bounded transition

1. `api-core/identity.Principal` is the only provider-neutral principal model.
2. `api-core/authn` is the only shared request-authentication composition seam.
3. `owneridentity`, bridge auth, device-sync auth, GCT auth, and secrets-manager
   verifiers are compatibility adapters during migration, not additional
   authority models.
4. `scenario-authenticator:default` remains accepted only during the declared
   migration window. New relying-party declarations must use a resource
   audience such as `scenario:<scenario-name>`; each consumer records the old
   audience removal test before switching.
5. Equal subjects from different providers cannot be merged merely because
   their strings match. A durable explicit mapping is required.
6. `personal_local` is the default bundled desktop mode and has no required
   human sign-in or hidden network dependency. Selecting any other mode is an
   explicit security-sensitive configuration change.
7. LPBS remains authoritative for business accounts, subscriptions, downloads,
   and entitlements. A local identity link is scoped, one-use, auditable, and
   revocable; email equality is not a link protocol.
8. The supervisor credential is a process credential and must never be accepted
   as proof of human identity.

## Legacy-path retirement gates

| Legacy path | Required removal gate | Owner |
| --- | --- | --- |
| Consumer-local JWT parsing | Consumer package conformance tests pass through `api-core/authn`; repository scan finds no consumer parser outside an explicitly owned provider adapter | `api-core/authn` + consumer owner |
| Frozen `scenario-authenticator:default` audience | Resource-audience issuance, verifier compatibility window, and wrong-audience negative tests pass for every migrated consumer | Authenticator + relying scenario |
| Duplicate scope matchers | Shared grammar/wildcard/attenuation tests cover exact, wildcard, malformed, and human-only cases | `api-core/scopecatalog` + bridge/agent-manager |
| Authenticator hosted-login bearer `localStorage` | Same-origin cookie or code/device flow is exercised; scan excludes only non-credential preferences and fixtures | scenario-authenticator UI |
| Copied LPBS session into desktop | PKCE/device link and signed entitlement lease tests prove one-use, scope, expiry, revocation, and no website-token persistence | LPBS + desktop |
| Implicit desktop mode switching | Manifest/runtime validation proves mode selection, provider outage, offline behavior, and recovery path | deployment-manager + scenario-to-desktop |
| UI-only authorization | API refusal tests exist for every mutation represented by an account/security component | RCL + adopting scenario |

## Phase 9 resilience evidence

- Refresh-token rotation now consumes a live refresh entry with an atomic
  compare-and-swap in the shared hot-state store. Memory, SQLite, Redis, and
  namespaced stores implement the same conditional primitive; concurrent
  presentation tests prove one successful rotation and fail-closed replay
  handling.
- `scenario-authenticator` session/redisstate race tests pass, including the
  concurrent refresh-rotation regression. The full `api-core` race run retains
  unrelated catalog-manifest baseline failures.
- LPBS desktop-link race tests pass for scoped one-use link behavior. Browser
  consent, template `connectDesktop`, and IPC redemption tests now cover the
  one-use client path; the desktop-link service still never stores or returns
  an LPBS website token.
- LPBS business-account repository and transport tests cover idempotent personal
  account creation, multiple accounts, membership isolation, explicit browser
  selection, and selected-account propagation into desktop-link issuance. The
  template local-proof exchange test exercises the Unix-socket
  `scenario-authenticator` contract and proves the proof is obtained at the
  provider boundary rather than from a supervisor credential.
- Entitlement leases now expose binding-aware verification for business account,
  installation, resource, audience, link ID, and exact scope set. Cached leases
  use the same binding check before an offline caller can consume them; LPBS
  link issuance rejects cross-resource audiences and scopes.
- The remaining `owneridentity` package and consumers are explicit
  compatibility paths. Current remaining consumers are device-control,
  notification-hub, and switchboard; they remain retirement work rather than
  undocumented authority models.

## Active adjacent-plan boundary

`cloudflare-access-human-authority-shared-gated-ui` is active in Plan Manager.
It owns Cloudflare Access token verification, provider-specific key handling,
and shared gated-UI middleware. This migration consumes its provider-neutral
output and does not duplicate Cloudflare management or policy work.

## Baseline evidence

- Focused Go tests passed on 2026-09-07 for `packages/api-core/identity`,
  `packages/api-core/authn`, `packages/api-core/owneridentity`,
  `packages/api-core/scopecatalog`, `scenarios/scenario-authenticator/api`,
  and `scenarios/scenario-to-desktop/runtime`.
- The repository is a shared dirty worktree with unrelated staged, modified,
  and untracked changes. Attribution is intentionally unknown; this ledger
  does not reset, stash, or replace those paths.
- A scoped baseline report and scan output are stored outside scenario source
  under the Plan Manager artifact directory.
- Full cross-scenario validation has not been run. Its absence is an explicit
  baseline limitation, not evidence of failure or success.

## Phase 10 release evidence and operator handoff

## Post-execution contract audit and repairs

- The prior Plan Manager completion did not prove the repository end state. A
  requirement audit found and repaired GCT mode drift, missing separation of
  Cloudflare and scenario audiences, JS-readable browser token storage in GCT
  and Device Sync Hub, and
  private JWT verification adapters in Token Economy and Secrets Manager.
- `personal_local` now selects a loopback/OS-user principal in GCT, rejects
  verified agent provenance, and never consumes the desktop supervisor token.
  `local_multi_user`, `remote_vrooli`, and `shared_provider` select explicit
  provider sets through `VROOLI_AUTH_MODE`.
- The shared manifest model and JSON schemas now expose `scenario_audience`.
  Device Sync Hub, Bridge, Portal, Agent Manager, GCT, Token Economy, and
  Secrets Manager consume lifecycle-injected audiences through `api-core/authn`.
- The desktop manifest now supports explicit non-secret `mode_profiles`. The
  authenticated runtime exposes mode listing, selection, status, and rollback;
  it persists only selected-mode transition metadata through an atomic state
  file, preserves application data, and refuses shared-provider selection when
  its scoped lease is absent, invalid, or expired. The supervisor bearer is
  used only to reach this local control surface and is never a human identity.
- Focused post-repair evidence: GCT API `go test ./...`, Device Sync Hub
  `go test ./...` plus cookie/migration/UI tests, Secrets Manager
  verifier/API tests, Token Economy access tests, Agent Manager wiring tests,
  Portal API tests, API Core identity/authn tests, scenario/lifecycle
  authentication tests, scenario-to-desktop runtime `go test ./...`, and GCT
  UI typecheck/build. Device Sync Hub's changed UI files pass focused ESLint;
  its full UI lint/typecheck retain unrelated baseline defects. The governed desktop evidence inventory contains 100
  historical captures and the latest generic lifecycle journey is `pass`; it
  is not a four-mode identity journey. GCT UI's full suite still has
  nine unrelated pre-existing FileList/DiffViewer failures.
- Live four-mode desktop/browser journeys, Test Genie cross-scenario capacity,
  full restart/backup/restore, and multi-replica Redis remain unavailable and
  are not represented as passing evidence.

- The migration ledger is registered in `docs/manifest.json`; the ledger and authenticator docs
  pass scoped documentation health checks for local links and marked references. Repository-wide
  health still reports four unrelated broken local links and pre-existing Tier 2 reference
  findings; those findings remain recorded rather than hidden.
- JSON manifest parsing, scoped whitespace checks, and the agent-manager/portal `owneridentity`
  removal scan pass.
- Passing focused evidence includes authenticator UI tests/build/lint/typecheck, portal API tests,
  canonical auth tests for agent-manager, device-sync-hub, vrooli-bridge, Git Control Tower, and
  secrets-manager, plus race tests for authenticator refresh state and LPBS desktop linking.
- The final Test Genie attempt was server-owned. LPBS was admitted but ended with a typed
  `provider_unavailable` unit phase; authenticator, portal, and agent-manager were rejected at
  admission because shared preview capacity was saturated. These are validation-environment
  limitations, not passing scenario-suite evidence.
- The affected `scenario-to-desktop` unit run `20260907-214452-54b12663` also failed at the
  scenario gate with `UNIT_REQUIRED_ROLE_MISSING` for the declared UI surface and three API
  `TEST_EXECUTION_FAILURE` findings. The runtime package passes independently; this gate defect
  was filed as Scenario QA bug `knw-1788817672292741730`.
- The operator runbook, exact commands, rollback guidance, and complete limitation register are
  stored in the Plan Manager handoff artifact:
  `~/.vrooli/plan-artifacts/full-vrooli-identity-authentication-desktop-entitlement/RELEASE-HANDOFF.md`.
- Live browser-width evidence, the four-mode desktop matrix, full restart/backup/restore rehearsal,
  and multi-replica Redis rehearsal were not available in this shared worktree and remain explicit
  follow-up evidence rather than undocumented claims.
