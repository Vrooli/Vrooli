# Security — Personal Planner

This document records the scenario's security and privacy posture.
Update it before adding auth, user data, external APIs, payment flows,
secrets, or sensitive business data.

## Purpose Of This Document

This is the security and privacy contract for Personal Planner. It exists
to answer, for reviewers and future agents:

- What sensitive data exists and how sensitive is it?
- How is access controlled, per record, on every path?
- Where do secrets come from and where are they forbidden?
- Which threats are known, and which of the mitigations are actually
  built versus only designed?

**Maturity note (read first):** Product domains now exist through real
proto/API/SQLite/CLI/UI paths. The API uses the shared `api-core` auth
boundary when authentication is configured, and the central router applies
baseline security headers. The current local workspace storage is not yet
subject-scoped on every domain query, and provider/share secrets are not yet
implemented; those gaps remain explicit below. Security Health currently
passes the scenario with no ERROR finding; advisory dependency/toolchain
intelligence remains visible in the validation receipt.

## Data Sensitivity

Personal Planner stores an honest model of a person's time, and much of
that model is genuinely private. Two categories deserve special care:
**personal planning data** (goals, commitments, effort estimates, delays,
forecasts, actuals, review notes — a detailed picture of what someone is
behind on and why) and **private external-calendar text** (imported event
titles, attendees, and locations that the user never authored inside this
product). Neither may cross a workspace boundary, and neither may leak
through a share.

| Data | Sensitivity | Owner | Details |
|---|---|---|---|
| Native planning records (goals, work items, milestones, commitments, allocations, actuals) | High | owning domain (goals/work/calendar/commitments/focus) | A truthful record of what a person committed to, what slipped, and why. Workspace-scoped (INV-01). |
| Forecasts, change records, review/learning insights | High | forecasts / review | Derived interpretation of the user's history; reveals risk and cause. Never a second authority; still private. |
| Imported external-calendar events (titles, attendees, locations, times) | High | integrations | Untrusted, private text the user did not author here. Subtracted from capacity but never surfaced through a share. |
| Provider access/refresh tokens | Secret | credential owner | Read-only external-calendar credentials. Never in records, logs, bundles, or exports. |
| Tokenized share URLs / share secrets | Secret | credential owner | Recipient-bound viewer access. Redacted from logs; excluded from exports. |
| Share grants (which fields, to whom) | Medium | sharing | Grant metadata, not the shared content itself; still workspace-scoped and revocable. |
| Appearance / availability / notification preferences | Low–Medium | workspace / notifications | Reveal habits and protected time; scoped like all other workspace data. |

## Auth And Authorization

**Partially implemented.** The shared API boundary and baseline transport
hardening are implemented, but local records are not yet subject/workspace
scoped on every domain query. The intended authorization model is:

- **Identity comes from scenario-authenticator** (a required dependency —
  see [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md)).
  Subject, workspace, and capabilities are resolved from authenticated
  context. A payload-supplied tenant or owner is never trusted; there is
  no client-supplied owner reassignment.
- **Per-record authorization on every path (INV-01).** Every lookup,
  every nested relationship traversal, every export, every background
  job, and every notification subscription re-checks that the record
  belongs to the authorized workspace and subject. Authorization is
  enforced at the API/service layer inside the one authoritative command
  surface — never in the UI or CLI, which must not enforce business
  authorization locally. A second identity cannot read the first's
  workspace, and a nested read (e.g. a commitment's linked private event)
  cannot bypass the parent check.
- **Sharing uses a recipient-bound viewer identity.** Selective sharing
  does not mint a public link; it grants a view to a specific
  scenario-authenticator viewer identity. If that guest identity is
  unavailable, the sharing feature stays explicitly blocked rather than
  silently downgrading to a public URL.
- **No-leak sharing (INV-12) is a server-side construction, not a UI
  concern.** Shared views are built server-side as DTOs from an explicit
  field allowlist bound to the recipient. Fields are never merely hidden
  in CSS or filtered on the client. Nested data and derived explanations
  obey the same mask: a private event that delays a shared commitment
  yields **"Available time changed,"** never its title, attendees, or
  location. Revocation applies on the next authorized fetch.
- **Bounded requests.** Import, recurrence expansion, and request sizes
  are bounded so a single caller cannot exhaust the process (see
  "Threat Model").

## Secrets

**Design-stage; no secrets are handled by code yet.** The intended rules,
which are hard constraints:

| Secret | Source | Required? | Details |
|---|---|---|---|
| External-calendar provider tokens (access/refresh) | Credential storage owner | Conditional (only if a user connects a calendar) | Read-only scopes only. Stored in the credential owner, never in domain records, UI state, logs, analytics, frontend bundles, or exports. Refreshed through the owner. |
| Tokenized share URLs / recipient viewer secrets | Credential storage owner | Conditional (only if a user creates a share) | Never persisted into records or logs; excluded from all exports and native backups. |

Governing rules for all secrets:

- Secrets live in the credential owner and **nowhere else** — never in
  records, logs, analytics, frontend bundles, or exports (INV, plan §17,
  §23.3).
- **Redact query parameters** wherever a token could appear in a URL
  (provider callbacks, share links) before anything is logged.
- Native exports and backups explicitly **exclude** credentials, access/
  session tokens, and reusable share secrets. A restored connection
  requires reconnection; a restored share is inactive until re-granted.

## Threat Model

All mitigations below are **designed, not built** (design stage). Status
reflects design intent, not verified enforcement.

| Risk | Impact | Intended mitigation | Status |
|---|---|---|---|
| Cross-workspace read (second identity reads another's plan) | High — full disclosure of private planning data | Per-record workspace/subject check on every lookup, nested traversal, export, job, and subscription (INV-01) | designed |
| Private cause leaks through a share (INV-12) | High — a viewer infers private events/attendees from a delayed commitment | Server-side DTOs from a recipient-bound allowlist; nested data and explanations masked; "Available time changed" copy; never CSS-hidden fields | designed |
| Secret exposure (provider tokens / share secrets in logs, bundles, exports) | High — account or share takeover | Secrets only in credential owner; query-param redaction; exports exclude secrets | designed |
| Prompt/markup injection via imported calendar text or task titles | Medium — untrusted text treated as instructions or rendered as active markup | Treat imported descriptions and titles as untrusted; sanitize or render as plain text; never interpret as instructions | designed |
| CSRF / cross-origin state change | Medium — a mutation triggered from another origin | Origin/CSRF checks at the command surface; mutations require authenticated context, not ambient cookies alone | designed |
| Malicious or forged provider callback | Medium — token bound to the wrong account, or an unsolicited callback | Provider callback bound to the initiating connection/state; only the initiating flow can complete a connection | designed |
| Over-broad provider scope (write/delete granted) | High — a bug or compromise could mutate a user's real calendar | Request read-only scopes; enforce read-only in the adapter as well; no provider mutation path exists in R1 | designed |
| Resource exhaustion via unbounded import/recurrence/request | Medium — denial of service or memory blowup | Bounded import size, bounded recurrence expansion, paginated history, bounded request sizes | designed |
| Stale share/credential reactivation on restore | Medium — a revoked share or expired token silently resumes | Restore never reactivates shares, credentials, or a running timer; connections require reconnection; shares require re-grant | designed |
| Unsafe file upload handling (notes example) | Low — malicious/oversized upload affects storage | Multipart handler validates metadata; BlobStore seam isolates bytes | template-reference (notes example only) |

## Security Gaps

The shared auth boundary is real, but the product is not yet safe to treat as
a multi-subject hosted workspace. The gaps below separate verified
enforcement from design intent so a passing security-health scan is not
mistaken for complete authorization coverage.

| Gap | Severity | Revisit Trigger |
|---|---|---|
| Auth boundary is configured, but subject/workspace ownership is not yet enforced in every domain query and mutation | High | Before multi-subject or hosted workspace data is enabled (P01–P02). |
| Local domain records are not yet proven by cross-workspace authorization regression tests | High | Alongside subject-scoped storage and the first hosted deployment path. |
| No-leak sharing DTOs / field masks (INV-12) not implemented | High | With the sharing feature (P09). |
| Secret handling (credential owner, redaction) not implemented | High | With external-calendar integration and sharing (P08–P09). |
| Untrusted-text sanitization for imported events not implemented | Medium | With provider/ICS import (P08). |
| CSRF/origin and provider-callback binding not implemented | Medium | With the command surface and provider connect flow (P07-ish, P08). |
| Read-only provider scope enforcement not implemented | High | With the first real provider adapter (P08). |
| Bounded import/recurrence/request limits not implemented | Medium | With import, recurrence expansion, and history endpoints (P08, P10). |
| No security regression tests yet prove cross-workspace no-leak behavior | High | Alongside subject-scoped storage and sharing; no-leak tests are a release gate. |

## Cross-References

- [`../concepts/DATA.md`](../concepts/DATA.md) — data ownership, retention, and privacy notes
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — external services, secrets, and credential owner
- [`../concepts/FLOWS.md`](../concepts/FLOWS.md) — sharing, provider, and notification flows
- [`ERROR-HANDLING.md`](ERROR-HANDLING.md) — error response behavior
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved security debt
