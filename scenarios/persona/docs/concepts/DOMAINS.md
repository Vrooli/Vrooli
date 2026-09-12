# Domains — Persona

This document is the canonical map of product capabilities, bounded
contexts, and ownership for this scenario. Keep it current whenever a
domain is added, renamed, split, merged, or removed.

Persona owns seven real domains plus the scaffold's `health`. The
organising idea behind the split: **a persona is a small object with a
large policy surface**, so the record (`personas`) is kept deliberately
separate from the rules about who may use it (`access`), the ways it
receives things (`channels`), the things it cannot do alone
(`handoffs`), the sensitive material it points at but never holds
(`documents`), the trail it leaves (`journal`), and what it has left
behind in the world (`accounts`).

## Purpose Of This Document

Use this document to answer:

- What product capabilities does this scenario expose?
- Which domain owns each concept, table, proto, endpoint, UI feature,
  CLI command, and test surface?
- Which concepts are shared, deferred, or deliberately not domains?

System-level architecture belongs in [`ARCHITECTURE.md`](ARCHITECTURE.md).
Workflow details belong in [`FLOWS.md`](FLOWS.md). Storage details
belong in [`DATA.md`](DATA.md).

## Domain Inventory

| Domain | Responsibility | Purpose | Owns Data | Primary Archetype | Secondary Traits | Glossary | Source Paths |
|---|---|---|---|---|---|---|---|
| health | Report runtime readiness and dependency reachability. | Expose API/database readiness and show the UI can read live backend state. | No product data. | reporting | query | HealthHandler | `api/handlers/health/`, `ui/src/features/health/`, `packages/proto/schemas/persona/v1/shared/health.proto` |
| personas | Own the durable identity record and its declared legal basis. | Give an agent something specific to *be* when it acts outward. | Persona records, kinds, legal basis, lifecycle state, addresses. | crud | lifecycle, policy | Persona, LegalBasis, PersonaKind, Address | `api/internal/personas/`, `api/handlers/personas/`, `cli/domains/personas/`, `ui/src/features/personas/`, `packages/proto/schemas/persona/v1/personas/` |
| access | Decide who may act as a persona and what a caller is entitled to see. | Make identity delegation enforceable rather than advisory. | ACL entries, act-as sessions, entitlement policy, attestations. | policy | authorization, integration | Grant, ActAs, Entitlement, DelegationChain | `api/internal/access/`, `api/handlers/access/`, `cli/domains/access/`, `ui/src/features/access/`, `packages/proto/schemas/persona/v1/access/` |
| channels | Receive what a persona must receive: mail and one-time codes. | Close the gap that stops an agent completing an ordinary signup. | Channel bindings, retrieval adapters, code-fetch records. | service | adapter-registry, integration | Channel, ControlledAddress, CodeRetrieval | `api/internal/channels/`, `api/handlers/channels/`, `cli/domains/channels/`, `ui/src/features/channels/`, `packages/proto/schemas/persona/v1/channels/` |
| handoffs | Model every step a machine must not take as resumable state. | Turn a hard wall into one pre-filled action for a person. | Handoff records, checkpoints, delivery attempts, resumptions. | workflow | state-machine, notification | Handoff, Checkpoint, Resumption, Wall | `api/internal/handoffs/`, `api/handlers/handoffs/`, `cli/domains/handoffs/`, `ui/src/features/handoffs/`, `packages/proto/schemas/persona/v1/handoffs/` |
| documents | Bind a persona to identity documents held by `document-manager`. | Make document custody usable without ever holding the bytes. | Bindings and release records only — never document content. | policy | integration, custody | DocumentBinding, Release | `api/internal/documents/`, `api/handlers/documents/`, `cli/domains/documents/`, `ui/src/features/documents/`, `packages/proto/schemas/persona/v1/documents/` |
| journal | Record every persona action append-only with its authorising human. | Make "who did this, as whom, on whose authority" always answerable. | Append-only action records. | reporting | audit, query | JournalEntry, Actor, Verb | `api/internal/journal/`, `api/handlers/journal/`, `cli/domains/journal/`, `ui/src/features/journal/`, `packages/proto/schemas/persona/v1/journal/` |
| accounts | Track what a persona has created and owes out in the world. | Make retirement and renewal possible without relying on memory. | Account links, obligations, staleness findings. | crud | reporting, lifecycle | AccountLink, Obligation, Staleness | `api/internal/accounts/`, `api/handlers/accounts/`, `cli/domains/accounts/`, `ui/src/features/accounts/`, `packages/proto/schemas/persona/v1/accounts/` |

## Domain Details

### health

- Purpose: expose API/database readiness and show the UI can read live
  backend state.
- Primary archetype: reporting / query.
- Secondary traits: operational health.
- Owns: health response construction and dependency status mapping.
- Does not own: product data, business rules, or scenario-specific
  domain behavior.
- API: `api/handlers/health/`.
- CLI: built-in `status` command is provided through cli-core.
- UI: `ui/src/features/health/HealthCard.tsx`.
- Storage: none; probes configured database reachability.
- Requirements: starter scaffold health only.
- Tests: handler, module, UI feature, and accessibility tests.
- Related docs: [`../reference/api-endpoints.md`](../reference/api-endpoints.md).

### personas

- Purpose: hold the durable object an agent acts as, so that outward
  action names a specific declared identity rather than an anonymous
  process.
- Primary archetype: CRUD / entity.
- Secondary traits: lifecycle (draft → active → suspended → retired),
  policy (creation refuses without a declared legal basis).
- Owns: the persona record; `PersonaKind` (`personal` | `business`) and
  the identifier set each kind admits; the immutable `LegalBasis`
  naming the human or entity the persona acts for; lifecycle state;
  billing and postal addresses as owned attributes.
- Does not own: who may *use* the persona (that is `access`), what it
  can receive (`channels`), or what it has created (`accounts`).
- Key invariant: **legal basis is declared at creation and immutable
  thereafter.** Changing who a persona represents is a new persona, not
  an edit, because every journal row already written asserts the old
  basis.
- Key invariant: a `business` persona may not borrow a `personal`
  persona's documents, addresses, or basis, and the reverse also holds.
- API: `api/internal/personas/`, `api/handlers/personas/`.
- CLI: `cli/domains/personas/` — `create`, `list`, `show`, `suspend`,
  `retire`, `addresses`.
- UI: `ui/src/features/personas/`, `ui/src/api/personas.ts`.
- Storage: domain-owned SQLite schema in
  `api/internal/personas/schema.sql`.
- Requirements: `PSN-P0-001`, `PSN-P0-010`, `PSN-P0-013`, `PSN-P1-002`.
- Tests: repository, service, handler, CLI, UI feature, accessibility,
  and a lifecycle test proving legal basis cannot be mutated.
- Related docs: [`DATA.md`](DATA.md), [`FLOWS.md`](FLOWS.md).

### access

- Purpose: make delegation enforceable — decide whether this caller,
  on this run, may act as this persona, and what it is entitled to see
  once it may.
- Primary archetype: policy / authorization.
- Secondary traits: integration (binds to `agent-manager` claims and
  reads grants from `prompt-manager`).
- Owns: ACL entries (which humans may act as, which may only propose);
  act-as session records; the entitlement rule that decides what a
  resolution returns; emitted attestations.
- Does not own: the run token itself, scope attenuation, or account
  identity — all three belong to `agent-manager` and are consumed here,
  never reimplemented.
- Key invariant: **entitlement is decided in one place.** A consumer
  receives what it is entitled to and nothing more, so no call site
  re-derives the rule.
- Key invariant: **fail closed.** If `agent-manager` is unreachable or
  the token is unverifiable, act-as is refused and the refusal is
  journaled. There is no degraded evidence grade.
- Key invariant: an agent cannot widen its own access; persona scopes
  ride the existing `IntersectScopes`/`Attenuate` machinery, so a child
  run can only ever hold a subset.
- API: `api/internal/access/`, `api/handlers/access/`.
- CLI: `cli/domains/access/` — `grant`, `revoke`, `list`, `resolve`,
  `attest`.
- UI: `ui/src/features/access/`, `ui/src/api/access.ts`.
- Storage: domain-owned SQLite schema in
  `api/internal/access/schema.sql`.
- Requirements: `PSN-P0-002`, `PSN-P0-003`, `PSN-P0-004`, `PSN-P0-012`,
  `PSN-P1-006`, `PSN-P1-007`, `PSN-P2-004`.
- Tests: repository, service, handler, CLI, UI, accessibility, a
  fail-closed test with the verification authority stubbed unreachable,
  and an attenuation test proving a child cannot widen.
- Related docs: [`INTEGRATIONS.md`](INTEGRATIONS.md),
  [`../internal/SECURITY.md`](../internal/SECURITY.md).

### channels

- Purpose: let a persona receive the two things a bare process cannot —
  mail at an address it controls, and the one-time codes that gate most
  real signups.
- Primary archetype: service.
- Secondary traits: adapter registry, external integration.
- Owns: channel bindings per persona; the registry of retrieval
  adapters; code-fetch records with their expiry.
- Does not own: the mailbox credential (that is `secrets-manager`) or
  the device lease used to read a phone (that is `device-control`).
- Key invariant: **one contract, many adapters, no privileged path.**
  Email, an SMS provider, and `device-control` reading a leased phone
  all satisfy the same retrieval contract; none is special-cased.
- Key invariant: a retrieved code is single-use and carries its expiry
  to the caller, because a silently expired code is indistinguishable
  from a wrong one at the point of failure.
- API: `api/internal/channels/`, `api/handlers/channels/`.
- CLI: `cli/domains/channels/` — `bind`, `list`, `fetch-code`, `test`.
- UI: `ui/src/features/channels/`, `ui/src/api/channels.ts`.
- Storage: domain-owned SQLite schema in
  `api/internal/channels/schema.sql`.
- Requirements: `PSN-P0-005`, `PSN-P0-006`, `PSN-P2-001`.
- Tests: repository, service, handler, CLI, UI, accessibility, an
  adapter-parity test proving every adapter satisfies the contract
  identically, and an expiry test.
- Related docs: [`FLOWS.md`](FLOWS.md),
  [`INTEGRATIONS.md`](INTEGRATIONS.md).

### handoffs

- Purpose: represent every step no machine may take as a first-class,
  resumable state rather than as a failure, so a blocked flow becomes a
  single pre-filled action for a person.
- Primary archetype: workflow / state machine.
- Secondary traits: notification, escalation.
- Owns: handoff records and their state; the checkpoint capturing
  everything already completed; delivery attempts; resumption records.
- Does not own: delivery transport (`notification-hub` is an optional
  relay) or the flow being blocked (usually driven by
  `browser-automation-studio`).
- Key invariant: **a handoff is never an error.** It is a named state
  with a named required human action and a resumable checkpoint.
- Key invariant: **the built-in queue always works.** Relay through
  `notification-hub` is an enhancement; with the hub absent the operator
  can still see and complete every open handoff.
- Key invariant: a handoff never instructs a human to defeat a
  verification control — it routes them to complete it legitimately.
- API: `api/internal/handoffs/`, `api/handlers/handoffs/`.
- CLI: `cli/domains/handoffs/` — `list`, `show`, `open`, `complete`,
  `cancel`, `resume`.
- UI: `ui/src/features/handoffs/`, `ui/src/api/handoffs.ts` — the
  scenario's primary operator surface.
- Storage: domain-owned SQLite schema in
  `api/internal/handoffs/schema.sql`.
- Requirements: `PSN-P0-007`, `PSN-P1-004`, `PSN-P1-008`, `PSN-P2-003`,
  `PSN-P2-006`.
- Tests: repository, service, handler, CLI, UI, accessibility, a
  state-machine test enumerating illegal transitions, and a
  relay-absent degradation test.
- Related docs: [`FLOWS.md`](FLOWS.md),
  [`../operations/RUNBOOK.md`](../operations/RUNBOOK.md).

### documents

- Purpose: make identity documents usable in a flow without this
  scenario ever holding them.
- Primary archetype: policy.
- Secondary traits: integration, custody.
- Owns: the binding between a persona and a document held in
  `document-manager`; release records naming the handoff a release was
  made into.
- Does not own: document bytes, parsing, sensitivity classification, or
  the custody journal — all four belong to `document-manager`.
- Key invariant: **no agent-readable read path exists.** No scope, no
  token, and no role returns document content to an agent. Release is
  into a named handoff and nowhere else.
- Key invariant: a release names a pre-declared handoff, so a merchant
  page cannot talk an agent into requesting a release it does not need.
- API: `api/internal/documents/`, `api/handlers/documents/`.
- CLI: `cli/domains/documents/` — `bind`, `list`, `release`, `revoke`.
- UI: `ui/src/features/documents/`, `ui/src/api/documents.ts`.
- Storage: domain-owned SQLite schema in
  `api/internal/documents/schema.sql` — bindings and release records
  only.
- Requirements: `PSN-P0-008`, `PSN-P0-009`.
- Tests: repository, service, handler, CLI, UI, accessibility, and a
  negative test asserting no request shape returns document bytes.
- Related docs: [`INTEGRATIONS.md`](INTEGRATIONS.md),
  [`../internal/SECURITY.md`](../internal/SECURITY.md).

### journal

- Purpose: keep "who did this, as whom, on whose authority, when"
  permanently answerable.
- Primary archetype: reporting / query.
- Secondary traits: audit, append-only.
- Owns: the append-only action record and its read surface.
- Does not own: the actions themselves; every other domain writes here
  and none may edit what it wrote.
- Key invariant: **no verb rewrites or deletes a row.** Correction is a
  new compensating row, never an update.
- Key invariant: a refusal is journaled as loudly as a success —
  refused act-as, refused release, and expired handoffs are the rows an
  audit actually needs.
- API: `api/internal/journal/`, `api/handlers/journal/`.
- CLI: `cli/domains/journal/` — `list`, `show`, `export`.
- UI: `ui/src/features/journal/`, `ui/src/api/journal.ts`.
- Storage: domain-owned SQLite schema in
  `api/internal/journal/schema.sql`.
- Requirements: `PSN-P0-011`.
- Tests: repository, service, handler, CLI, UI, accessibility, and an
  immutability test proving update and delete are unreachable.
- Related docs: [`DATA.md`](DATA.md),
  [`../internal/SECURITY.md`](../internal/SECURITY.md).

### accounts

- Purpose: remember what a persona has created and what it owes, so
  retirement and renewal are driven by a register rather than by
  memory.
- Primary archetype: CRUD / entity.
- Secondary traits: reporting, lifecycle.
- Owns: account links (site, login seam, recovery path); obligations
  (what is owed, when it renews, how to cancel); staleness findings
  (expiring documents, failing mailbox, unreachable code route).
- Does not own: the money half of an obligation, which belongs to
  `treasury`; this domain records that a commitment exists, never its
  amount or payment.
- Key invariant: **retirement is blocked while linked accounts have no
  recorded recovery path**, because that is precisely how an account
  gets orphaned.
- API: `api/internal/accounts/`, `api/handlers/accounts/`.
- CLI: `cli/domains/accounts/` — `link`, `list`, `obligations`,
  `staleness`.
- UI: `ui/src/features/accounts/`, `ui/src/api/accounts.ts`.
- Storage: domain-owned SQLite schema in
  `api/internal/accounts/schema.sql`.
- Requirements: `PSN-P1-001`, `PSN-P1-003`, `PSN-P1-005`, `PSN-P2-002`.
- Tests: repository, service, handler, CLI, UI, accessibility, and a
  retirement-guard test.
- Related docs: [`FLOWS.md`](FLOWS.md),
  [`../business/MONETIZATION.md`](../business/MONETIZATION.md).

## Shared Concepts

| Concept | Meaning | Owner |
|---|---|---|
| Domain | Product capability boundary that should be easy to find, test, and delete. | `DOMAINS.md` defines the map; code owns implementation. |
| Surface | API, UI, CLI, or contract layer exposing the same product capability. | `ARCHITECTURE.md`. |
| Seam | Test-substitutable boundary wired once in production. | `../internal/SEAMS.md`. |
| Requirement | Implementation-facing measurement tied back to the PRD. | `requirements/`. |
| Persona | A durable declared identity an agent acts as, naming the human or entity that authorised it. | `personas`. |
| Legal basis | The human or legal entity a persona represents; immutable after creation. | `personas`. |
| Delegation chain | legal person → persona → account subject → run token → child token. The outer two links are ours; the inner three are `agent-manager`'s. | `access`. |
| Entitlement | What a given caller is allowed to receive from a persona resolution. | `access`. |
| Wall | A step no machine may take: a CAPTCHA, a biometric, a photo ID check, a human review. | `handoffs`. |
| Handoff | The typed, resumable state produced when a flow meets a wall. | `handoffs`. |
| Release | The single verb by which sensitive material reaches a named handoff. | `documents`. |

## Deferred Domains

| Candidate Domain | Why Deferred | Revisit Trigger |
|---|---|---|
| `reputation` | Whether a persona is trusted by a given counterparty is real, but there is no signal to compute it from until many accounts exist. | Once `accounts` holds links across several counterparties and a flow needs to choose between personas. |
| `provisioning` | Creating mailboxes and phone numbers on demand is a genuine capability, but it is provider-shaped and would drag a paid dependency into P0. | When `channels` has two or more real adapters and manual setup is measurably the bottleneck. |
| `kya-exchange` | Verifying a counterparty's agent attestation is the mirror of `PSN-P1-007`. Deferred because the standard is still moving. | When KYA-OS stabilises and a real counterparty asks Vrooli to present one. |

## Non-Domains

These are important but should not become product domains:

- `api/internal/server/` — HTTP composition substrate.
- `api/internal/module/` — shared module descriptor type.
- `api/internal/modules/` — thin registry for boot/codegen.
- `api/internal/database/` — cross-cutting database infrastructure.
- `api/internal/testutil/` — cross-domain test harnesses.
- `ui/src/components/` — shared presentation primitives.
- `ui/src/test-utils/` — cross-feature testing support.

Scenario-specific additions to this list:

- **Verification of run identity** — belongs to `agent-manager`. This
  scenario calls it and fails closed; it never reimplements token
  parsing, scope intersection, or attenuation.
- **Document storage, parsing, and sensitivity** — belongs to
  `document-manager`. `documents` here is a binding table and a release
  verb, nothing more.
- **Credential storage** — belongs to `secrets-manager`.
- **Team and member rosters** — belong to `prompt-manager`. `access`
  reads grants; it does not own an org chart.

If one of these starts using product vocabulary, split the product
piece into an owning domain instead of growing infrastructure.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system shape and extension rules
- [`FLOWS.md`](FLOWS.md) — workflows and state transitions
- [`DATA.md`](DATA.md) — data ownership and storage
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — dependency contracts
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — boundary registry
- [`../internal/TESTING.md`](../internal/TESTING.md) — test strategy
