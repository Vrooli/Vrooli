# Problems — Scenario Authenticator

Persistent register of known issues, tech debt, and deferred work
specific to **this** scenario. Future agents read this file to avoid
re-discovering the same constraint.

This file ships empty in newly generated scenarios. Append entries as
they appear.

## What belongs here

- **Known bugs** that are real but not yet worth fixing
- **Tech debt** — workarounds that need a real fix later
- **Deferred work** — features descoped from a phase, with the reason
- **Architecture drift** — code/docs/tests that no longer line up with
  the intended capability map or boundary model
- **Constraints discovered the hard way** that aren't visible from
  the code (e.g., "this resource needs warm-up before the first call;
  see commit X")

## What does NOT belong here

- **Generic template issues** — those go in
  [`../guides/troubleshooting.md`](../guides/troubleshooting.md)
- **Open feature requests** — track those in PRD operational targets
- **Code comments** — if the constraint is local to one file, a
  comment there is more discoverable
- **Test failures** — fix them, don't document them

## Entry template

Use this shape so entries are scannable. Append newest at the bottom.

```markdown
### YYYY-MM-DD — short title

**Symptom:** What goes wrong, observable from outside the system.

**Root cause:** What actually causes it (or "unknown" if not yet diagnosed).

**Workaround:** What to do today to keep moving.

**Real fix:** What needs to happen for this entry to be deleted.

**Owner:** Who should drive the fix (or "unassigned").

**Refs:** Code paths, related issues, prior commits.
```

## Entries

### 2026-06-18 — Why this rewrite exists: the shared-Postgres blast radius

**Symptom:** The old working scenario ran on the fleet's **shared
Postgres** and **shared Redis** under a shared `vrooli` database role
(`db.DB.QueryRow(... WHERE email = $1 ...)`, `db.RedisClient` package
globals — see `/tmp/scenario-authenticator-OLD-reference/api`). Reconciling
the authenticator's database password / role during operational work could
disrupt **sibling scenarios** sharing that same role and instance. For a
*foundational auth service* — the one capability every other scenario
depends on — a shared-infrastructure operation that can take down neighbors
is an unacceptable blast radius.

**Root cause:** Shared multi-tenant Postgres + a shared DB role couples the
authenticator's lifecycle (migrations, credential rotation, recovery) to
every other scenario on the same database. Auth is the worst possible place
for that coupling: it sits at the bottom of the stack, so its outages
cascade.

**Workaround (the rewrite):** Move to **per-scenario SQLite via the
`api-core/storage` seam** (OT-P0-011 / `REQ-P0-011`). SQLite-per-scenario
removes the entire shared-DB failure class — the authenticator's database
is its own file, owned by its own process, with no cross-scenario role.
**Redis is kept** for hot/shared state (sessions, revocation, OAuth CSRF,
distributed rate limiting) because that state is genuinely hot and the
coupling there is intentional and bounded. The storage seam keeps a clean
path to a managed DB at scale (OT-P2-006) without reintroducing the shared-
role coupling.

**Real fix:** This entry is the design rationale, not a bug to close. It
stays as the durable "why" so a future agent does not "simplify" the
authenticator back onto shared Postgres.

**Owner:** n/a (design rationale).

**Refs:** PRD §Overview + §Operational risks; `REQ-P0-011`;
[`SEAMS.md`](SEAMS.md) (api-core/storage resolver seam);
`/tmp/scenario-authenticator-OLD-reference/api` (old shared-DB code).

### 2026-06-18 — Documentation-first: PRD + requirements + docs authored, ZERO implementation

**Symptom:** The scenario is the freshly regenerated react-vite scaffold
plus the `health` domain and the fenced `notes` worked example. The auth
domains (`realms`, `identity`, `tokens`, `sessions`, `authorization`,
`audit`, and the P1 `mfa`/`federation`/`apikeys`) are documented as the
**target** map but **not implemented**. No auth code, no auth tests, no
proto for the auth domains yet.

**Root cause:** Intentional. This is a documentation-first orientation pass.
Work was deliberately **stopped for review before Gate 6** (the first real
domain green / `example-domain-removed`). The internal docs
([`SECURITY.md`](SECURITY.md), [`SEAMS.md`](SEAMS.md),
[`PERFORMANCE.md`](PERFORMANCE.md), [`ERROR-HANDLING.md`](ERROR-HANDLING.md),
[`TESTING.md`](TESTING.md), this file) transfer the design captured in
`PRD.md` and `DOMAINS.md` so implementation can begin from a grounded plan.

**Workaround:** None needed — nothing claims to be built. Do not read the
internal docs as "as-built"; they describe the intended design. The
`detemplate` step (Gate 7) that removes the fenced `notes` example has not
run.

**Real fix:** Implement P0 in the launch sequence (PRD §Launch sequencing):
accounts + Argon2id + RS256/JWKS/keypair + refresh rotation w/ reuse
detection + Redis sessions + rate limiting/lockout + audit + default realm
w/ `aud` scoping + Connect surface + CLI parity + SQLite seam, then the live
device-sync-hub migration before any P1 work.

**Owner:** unassigned (P0 implementer).

**Refs:** `PRD.md` §Launch sequencing; `requirements/` (`REQ-P0-001`…
`REQ-P0-012`); [`../concepts/DOMAINS.md`](../concepts/DOMAINS.md); PROGRESS.md.

### 2026-06-18 — Security-boundary risk: port the crypto verbatim, do not re-derive it

**Symptom:** The single biggest risk in the rewrite is re-deriving proven
crypto incorrectly and either (a) breaking the live consumer
(device-sync-hub verifies today against the exact RS256/JWKS/claims
contract) or (b) silently weakening it (e.g. accidentally accepting
`alg=none`, dropping the `aud` check, or regenerating the signing key on
each boot and invalidating live tokens).

**Root cause:** A scaffold-regen invites a clean-room rewrite mindset. For
the auth core that mindset is dangerous — the old RS256 signing, JWKS
publication, claims shape, and load-or-generate keypair persistence are
**correct and in production**.

**Workaround / discipline:** Treat the crypto core as a **verbatim port**
(PRD Appendix C), not a reimplementation. Lock the algorithm to RS256 and
reject `none`/HS confusion; keep the persisted load-or-generate keypair;
keep the claims/issuer shape byte-for-byte; keep JWKS at
`/.well-known/jwks.json`. The **one deliberate strengthening** is password
hashing: the old scenario uses **bcrypt** (`bcrypt.DefaultCost`,
`golang.org/x/crypto/bcrypt`); the rewrite moves to **Argon2id** at the
password boundary only — this does not touch the token crypto. Cover all of
it with the carried-over-invariant regression suite and the must-have
cross-realm rejection test ([`TESTING.md`](TESTING.md)).

**Real fix:** Land the invariant regression tests *before or alongside* the
crypto port, so any drift from the live contract fails the build.

**Owner:** unassigned (P0 implementer).

**Refs:** PRD Appendix C; [`SECURITY.md`](SECURITY.md);
[`TESTING.md`](TESTING.md) (invariant + cross-realm clusters);
`/tmp/scenario-authenticator-OLD-reference/api/auth/jwt.go` (RS256 +
keypair), `.../handlers/auth.go` (bcrypt), `.../handlers/jwks.go` (JWKS/kid).

### 2026-06-18 — Deferred work (P2 and explicitly out of P0/P1 scope)

**Symptom:** Several capabilities a "complete" IdP might be expected to have
are intentionally **not** in P0/P1.

**Root cause:** Scope discipline — P0 ships the foundational core + the live
consumer migration; P1 adds realms-as-true-tenant, MFA, social federation,
API keys, scopes, passkeys, and the polished UIs. Everything below is **P2**
(PRD §P2 / `requirements/` `REQ-P2-001`…`REQ-P2-007`):

- **SAML 2.0 / enterprise SSO** (`REQ-P2-001`).
- **OIDC-provider mode + token introspection** ("Login with Vrooli")
  (`REQ-P2-002`).
- **Groups / teams / org hierarchy** above the user (`REQ-P2-003`).
- **Delegated authorization / policy engine** — fine-grained "can-they"
  stays with the RP until/unless this lands (`REQ-P2-004`).
- **Per-realm key isolation + automated rotation** with overlapping `kid`s
  (`REQ-P2-005`). The P0 single-key + `kid`-in-JWKS shape is what makes this
  expressible later without breaking RPs.
- **Managed-DB backing + multi-instance HA** via the storage seam
  (`REQ-P2-006`).
- **SCIM provisioning** (`REQ-P2-007`).

**Workaround:** Not needed — these are deferred by plan, not blocked. The
`realms` domain ships `aud`-scoping in P0 specifically so that multi-tenancy
(and the P2 items that build on it) is a configuration/extension step, not a
re-architecture.

**Real fix:** Pull each into a phase when an adopter needs it; the
deferred-domains table in [`../concepts/DOMAINS.md`](../concepts/DOMAINS.md)
records the revisit triggers.

**Owner:** unassigned.

**Refs:** PRD §P2; `requirements/` `REQ-P2-*`; DOMAINS.md (Deferred Domains).

### 2026-06-18 — Live consumer (device-sync-hub) depends on the JWT/JWKS/sessions contract

**Symptom:** device-sync-hub is **live today** as the first Relying Party.
It resolves scenario-authenticator by slug, fetches JWKS, and verifies
tokens locally; its same-origin forwarder calls the auth surface, and it
calls the `/api/v1/sessions/{id}` revoke contract. Any change to the
RS256/JWKS/claims/sessions contract that is not made in lockstep breaks a
running consumer.

**Root cause:** Foundational interface-enablers have downstream consumers by
definition. The rewrite must keep the carried-over contract byte-compatible
(or migrate the consumer in the same change).

**Workaround / discipline:** The P0 plan migrates device-sync-hub's
forwarder from REST to the typed Connect client **in lockstep** with the P0
launch and proves the live first-run flow end-to-end **before** any P1 work
begins (OT-P0-012 / `REQ-P0-012`). No compatibility shims or legacy dead
code — when a contract changes, both sides change together.

**Real fix:** Keep the device-sync-hub forwarder integration test
([`TESTING.md`](TESTING.md), integration cluster) as a standing gate.

**Owner:** unassigned (P0 implementer, coordinating with device-sync-hub).

**Refs:** PRD Appendix A/C + §Launch sequencing; `REQ-P0-012`;
`scenarios/device-sync-hub/`; [`TESTING.md`](TESTING.md) (forwarder cluster).

### 2026-06-18 — Tooling: prd-control-tower marked two UI targets "complete" in error

**Symptom:** `OT-P1-007` (Admin Console UI) and `OT-P1-008` (End-User
Self-Service UI) repeatedly auto-check to `[x]` in `PRD.md`, and
`REQ-P1-007` / `REQ-P1-008` flip to `status: complete` (validation entries to
`implemented`) — even though **nothing is implemented**. Resetting the local
files works until the next `make test` / `vrooli scenario orient`, which
flips them back.

**Root cause:** `prd-control-tower requirements generate` recorded those two
modules as `complete`/`implemented` in **prd-control-tower's own
(authoritative) store** when the registry was generated — a generator error
(it appears to have pattern-matched the "production-polished UI" wording to
"done"). The test-genie requirements-sync (the `business` phase, run by
`make test` and by `vrooli scenario orient`'s scaffold-health gate) treats
prd-control-tower's store as source of truth and **rewrites the local module
status + the PRD checkboxes from it** on every run. The local file edit
cannot win against the recomputation. Note: only the **status** fields are
overwritten; the requirement descriptions and validation strategies are
preserved. All 26 other modules generated correctly (`planned`).

**Workaround:** Local files have been reset to the accurate state
(`status: planned`, validation `pending`, PRD boxes `[ ]`). This holds as
long as the sync is not re-run. The on-disk / committed snapshot is correct;
a subsequent `make test` may transiently re-flip these two until the store is
corrected.

**Real fix:** Correct the two requirement statuses in prd-control-tower's
store (there is currently no `requirements update`/`set-status` CLI verb — a
gap worth closing). Re-running `prd-control-tower requirements generate`
would reset the store but re-authors the whole validated 28-module registry,
so it was not done here. Filed against prd-control-tower (scenario-qa
bug-inbox). Until fixed, treat these two checkboxes as **not** authoritative
and trust the implementation reality (nothing built).

**Owner:** prd-control-tower maintainer (generator + a status-update verb).

**Refs:** `requirements/19-admin-console-ui/`, `requirements/20-end-user-self-service-ui/`;
`PRD.md` OT-P1-007/008; test-genie `business` phase / requirements-sync.

## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| _None yet._ |  |  |  |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
