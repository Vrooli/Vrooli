# Security — Money Ledger

This document records the scenario's security and privacy posture.
Update it before adding auth, user data, external APIs, payment flows,
secrets, or sensitive business data.

## Purpose Of This Document

Use this document to answer:

- What sensitive data exists?
- How is access controlled?
- Where do secrets come from?
- Which threats are known and how are they mitigated?

## The one-paragraph summary

This scenario holds the most sensitive dataset in the fleet: a complete record of where an
operator's money is and where it went. It is **not regenerable** — it cannot be recomputed
from anything else. Three properties carry the posture: the journal is append-only so
history cannot be quietly rewritten, no adapter may hold a reusable bank credential, and
every figure carries a basis so a tampered or degraded input is visible rather than
absorbed. Confidentiality matters here, but **integrity is the primary security property**,
because a ledger that silently shows a wrong number is worse than one that will not open.

## Data Sensitivity

| Data | Sensitivity | Owner | Details |
|---|---|---|---|
| Postings — amount, date, account, direction | **high** | `journal` | The complete financial history. Discloses income, spending patterns, counterparties, and location-by-inference. Not regenerable. |
| Accounts and books | **high** | `books` | Account names and kinds reveal institutions held and the personal/business split. |
| Audit trail | **high** | `journal` | Records who changed what and when. Its integrity is what makes every other guarantee checkable; tampering here defeats the append-only design. |
| Adapter credentials | **critical** | `ingest` | Never stored by this scenario. Held only as references into the platform secret store. See Secrets. |
| Sync cursors and ingestion receipts | medium | `ingest` | Leak the existence and timing of external accounts even without amounts. |
| Provenance and basis fields | medium | `journal` | Low value alone; load-bearing for integrity, so treated as tamper-relevant. |
| Goal declarations and thresholds | medium | `position` | Disclose financial targets and runway posture. |
| Tax category tags | medium | `journal` | Deductibility categorisation only — never a computed tax figure (PRD non-goal). |
| Position snapshots | medium | `position` | Derived; regenerable from the journal by construction. |

## Auth And Authorization

No local process may change these records without a verified, authorised identity. One
policy, `api/internal/access/policy.go` (`Policy`), names every route the API serves and
who may use it; one middleware, `access.Gate`, enforces it in front of every route. A
route missing from the table is refused before it runs, and a test fails if a mounted
route is undeclared, so a new RPC cannot ship ungated by omission.

### Who may write

The gate resolves each request to an **actor**, the name the journal records, or to a
refusal. It trusts only the principal the shared `api-core/authn` provider chain verified;
nothing a request says about itself (headers, body fields) makes it a writer. Request
messages carry no `actor` field (the proto reserves it), so attribution cannot be
asserted by a client.

| Caller | Becomes | Condition |
|---|---|---|
| A verified person | `operator` | The host's owner-binding policy (`~/.config/vrooli/owner-bindings.json`, see `path:docs/concepts/AUTHORIZATION.md`) binds one of their verified provider subjects to their canonical subject, and the binding grants `money-ledger:write` (or `*`). |
| A verified person not so bound | refused, `NOT_OWNER` | Signing in proves who someone is, not that they own these records. Scenario Authenticator holds many accounts and its scopes are self-service (an account can grant itself `money-ledger:write`), so only the operator-owned binding file distinguishes the owner. |
| A declared service principal | its actor | Issued by Scenario Authenticator, its subject is listed in `serviceWriters`, and it holds `money-ledger:write`: `treasury-money-ledger` → `treasury`, `landing-page-business-suite-money-ledger-writer` → `landing-page-business-suite`. |
| Any other service | refused, `UNDECLARED_SERVICE` | |
| An agent run | refused, `AGENT` | Agents never write the journal, whatever credential they carry. An agent that needs a record changed asks the operator, who runs the CLI with `--local-owner` or uses the console. |
| No credential | refused, `SIGNED_OUT` (HTTP 401 / `Unauthenticated`) | |
| A credential that does not verify, or a provider that cannot be reached | refused, `UNVERIFIED` (401) | Fails closed. |
| Two different verified identities in one request | refused, `CONFLICT` | For example the tunnel sign-in and a browser session for another account. |

`AccessService/GetWriteAccess` reports this same decision to a client before it offers a
write, so the console and CLI can explain a refusal instead of failing a submitted form.
It is advisory: the gate decides every write independently.

### The route policy

| Route | Effect | Callers |
|---|---|---|
| `/health`, `/api/v1/health`, `/api/v1/capabilities/describe` | read | open |
| `AccessService/GetWriteAccess` | read | open |
| `AccessService/SignIn`, `SignOut` | session | open; changes no records |
| `BooksService/ListBooks`, `ListAccounts` | read | open |
| `BooksService/CreateBook`, `ArchiveBook`, `CreateAccount` | write | operator |
| `JournalService/GetPosting`, `ListPostings` | read | open |
| `JournalService/ReversePosting`, `Transfer` | write | operator |
| `PositionService/GetPosition`, `GetStatement`, `ListGoals` | read | open |
| `PositionService/DeclareGoal`, `ArchiveGoal`, `ReparentGoal` | write | operator |
| `IngestService/ListAdapters`, `OperatorInputStatus`, `PreviewCryptoObservation` | read | open |
| `IngestService/GetCryptoReceipt`, `ListCryptoReceipts` | read | treasury, operator |
| `IngestService/AcceptCryptoReceipt` | write | treasury |
| `IngestService/RegisterAdapter` | write | operator, treasury |
| `IngestService/IngestEvent` | write | operator, treasury, landing-page-business-suite |
| `IngestService/RunAdapter`, `ImportFile`, `ImportOperatorInputs` | write | operator |

Reserved adapter ids belong to one actor even on a route several actors may use:
`treasury` to Treasury, `manual` and `transfer` to the operator. `treasury-base-usdc` is
written only inside an accepted crypto receipt's transaction, never by `RegisterAdapter` or
`IngestEvent`.
`ImportOperatorInputs` is operator-only even as a rehearsal, because it reads a host file
the caller names.

### Why reads stay open to local programs

The API listens only on this host. Local consumers (Offer Desk, Treasury's checks) read
positions and books without a credential, and any process running as the operator can
read the SQLite file directly, so requiring identity to read would add no confidentiality
against a local program. It would add friction to every consumer. Integrity is the
primary property (see DECISIONS), and every change is gated.

Browsers are the exception, because the operator's browser also runs other websites'
script and most scenario UI servers answer any origin. The gate refuses every request a
browser marks as cross-site or same-site (`Sec-Fetch-Site` other than `same-origin` or
`none`) before consulting the policy, so another website — or another localhost
scenario's page — can neither read the ledger through localhost nor ride the owner's
session. Page script cannot forge that header; local programs send none. Through the
tunnel, Cloudflare Access admits only the people its policy names to the UI at all.

Accepted crypto receipts carry customer credit attribution, so they are read only by the
service that accepted them and the owner.

### How each surface proves the owner

- **Console, local.** The platform's same-origin sign-in facade (`api-core/browsersession`,
  the mechanism Notification Hub uses for its installed app). `AccessService/SignIn`
  forwards the email, password and second factor to Scenario Authenticator for the
  `money-ledger` audience and sets HttpOnly, SameSite=Lax cookies that page script never
  reads; `GetWriteAccess` renews an expired session from its refresh cookie; `SignOut`
  clears the cookies and revokes the session. Without a session the console is
  **viewing only**: reads work, every write control is disabled, and a notice says why
  and offers the sign-in when signing in could change the answer.
- **Console, through the tunnel.** The Cloudflare Access sign-in is the identity, bound by
  the same owner-binding policy; no form is needed.
- **CLI.** Every write verb takes `--local-owner`, which exchanges the operator's
  machine-linked Scenario Authenticator session through the local authenticator socket
  and presents it for that one request. It is refused inside an agent run. Without it a
  write goes out with no identity and is refused with a sentence saying what to do.
- **Services.** Treasury and Landing Page Business Suite present their declared service
  principal's bearer token, issued by `scenario-authenticator service-principals sync
  --apply` (an operator command).

## Secrets

| Secret | Source | Required? | Details |
|---|---|---|---|
| Aggregator / commerce API keys | Platform secret store | only per authenticating adapter | Referenced by handle. **Never** in scenario config, environment files, or the database. |
| Bank passwords / login credentials | **prohibited** | never | PRD non-goal, restated here because it is the highest-liability thing this scenario could be asked to do. Aggregator API, file import, or nothing. |
| Database encryption key, if at-rest encryption is added | Platform secret store | deferred | See Security Gaps. |

Rules that hold for every adapter:

1. A credential is read at use and never copied into scenario state.
2. An adapter that cannot reach its credential reports **unavailable with a reason**. It
   must never fall back to an unauthenticated path or return an empty result — a silent
   zero from a credential failure is the exact defect `OT-P1-004` exists to prevent.
3. Credentials never appear in logs, error messages, or ingestion receipts. Receipts record
   *that* an adapter authenticated, never *with what*.

## Threat Model

| Risk | Impact | Mitigation | Status |
|---|---|---|---|
| Silent data tampering | A modified posting changes every derived figure with no trace. | Append-only journal; corrections are reversing entries; audit trail records actor, timestamp, prior value. | by-design |
| Adapter credential compromise | Read access to a real financial institution. | No password-based adapters, ever. Aggregator API or file import only. Credentials by reference from the secret store. | by-design |
| A degraded adapter reported as zero | Operator makes a decision on a confidently wrong number. This is the scenario's signature failure. | `OT-P1-004`: unavailable is a distinct state carrying reason and age; position reports partial with the gap named. | by-design |
| Malicious or malformed file import | Crafted CSV/OFX causes injection, resource exhaustion, or silent misparse. | File import is an ordinary adapter behind the one typed contract; parse failures reject the batch rather than partially applying it. Idempotent ingestion (`OT-P0-007`) bounds re-run damage. | planned |
| Local database exfiltration | Full financial history disclosed. | Local-first with no network exposure by default; at-rest encryption deferred (see Gaps). | partial |
| Backup handling of a non-regenerable dataset | Permanent, unrecoverable loss. | `DATA.md` must declare the dataset non-regenerable so retention policy cannot treat it as reproducible. | required |
| Log leakage of amounts or account identifiers | Financial disclosure through an operational surface. | Amounts and account identifiers are never logged at info level; see `OBSERVABILITY.md`, which forbids financial values in metric labels. | by-design |
| Reversal used as a de facto edit | The audit trail exists but stops being meaningful. | A reversing entry references the entry it reverses and carries a reason; only the owner may reverse, and the journal records the verified actor. | by-design |
| A local process or agent writes the journal | Fabricated or altered financial history with plausible attribution. | Every write passes the access gate: only the owner-bound person, a declared service principal, or nobody. Agent runs are refused whatever they carry. The actor is the verified caller, never a request field. | by-design |
| Another website reads or writes through the operator's browser | Disclosure of the ledger, or a write riding the owner's session. | Cross-site and same-site browser requests are refused by `Sec-Fetch-Site` before any route runs; session cookies are HttpOnly and SameSite=Lax. | by-design |
| A self-registered identity-provider account claims the owner's capability | Any local account holder becomes a writer. | Ownership comes only from the operator's owner-binding file, never from provider scopes or being able to sign in. | by-design |
| Browser session cookies are shared across localhost ports | Another scenario's page on the same host name could present or overwrite the session cookie. | Cookies are HttpOnly; cross-site and same-site requests are refused; the console names a conflicting identity rather than guessing. Port-scoped cookie names are a platform follow-up (see Gaps). | partial |
| Unsafe file upload handling (template `notes`) | Malicious or oversized upload affects storage. | Multipart handler validates metadata; BlobStore seam isolates bytes. | template-reference |

## Security Gaps

| Gap | Severity | Revisit Trigger |
|---|---|---|
| No at-rest encryption of the SQLite database | medium | Before any deployment where the host is not solely the operator's own machine — notably `scenario-to-cloud` or a shared device. |
| One owner, not per-book authorization | conditional | Every write is the owner's or a declared service's. `OT-P2-003` (multiple books shared by different people) needs book-scoped authorization in the gate's policy, not a `WHERE` clause. |
| Browser cookie names are not scoped to a port | low | Browsers share cookies across ports of one host, so every localhost scenario using the platform sign-in shares the `vrooli_access_token` name. Revisit when the platform scopes session cookies per relying scenario. |
| Reads are open to local programs | low (by decision) | Revisit if the API ever listens beyond loopback, or a consumer runs as a different OS user than the operator. |
| Export path not threat-modelled | medium | Before `OT-P1-006` tax categorisation export. An export is a bulk disclosure surface and deserves its own row here. |
| Audit-trail integrity is enforced by convention, not cryptography | low | Only if a scenario or regulation requires proving the trail was not altered by someone with database access. Hash-chaining is the known answer; not justified for a single-operator local deployment. |

## Cross-References

- [`../concepts/DATA.md`](../concepts/DATA.md) — data ownership and retention
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — external services and secrets
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — what may and may not be emitted
- [`ERROR-HANDLING.md`](ERROR-HANDLING.md) — error response behavior
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved security debt
