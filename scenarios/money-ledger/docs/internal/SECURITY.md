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

No auth provider ships in the scaffold, and none is required for the single-operator local
deployment that is the P0 target. What **is** required before any multi-user or networked
deployment:

- Authorization belongs at the API/service layer. The CLI and UI must never be the place a
  rule is enforced — the CLI is a typed mirror of the API and both are equally reachable.
- **`OT-P2-003` (multiple books) is an authorization boundary, not a filter.** The moment a
  second party's data shares an instance, book scoping stops being a query convenience and
  becomes the access-control model. Do not implement multi-book as a `WHERE` clause and
  call it done.
- Correction is a permission, not a shape. Writing a reversing entry must be authorized
  separately from reading, because reversal is the only sanctioned way to alter effective
  history.

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
| Reversal used as a de facto edit | The audit trail exists but stops being meaningful. | A reversing entry references the entry it reverses and carries a reason; reversal is authorized separately from write. | planned |
| Unsafe file upload handling (template `notes`) | Malicious or oversized upload affects storage. | Multipart handler validates metadata; BlobStore seam isolates bytes. | template-reference |

## Security Gaps

| Gap | Severity | Revisit Trigger |
|---|---|---|
| No at-rest encryption of the SQLite database | medium | Before any deployment where the host is not solely the operator's own machine — notably `scenario-to-cloud` or a shared device. |
| No auth model | conditional | Required before `OT-P2-003` (multiple books) or any networked deployment. Multi-book is the trigger, not an enhancement to it. |
| Reversal authorization not yet separated from write | medium | Implement with `JRNL-004`, in the same slice as the journal — not after. |
| Export path not threat-modelled | medium | Before `OT-P1-006` tax categorisation export. An export is a bulk disclosure surface and deserves its own row here. |
| Audit-trail integrity is enforced by convention, not cryptography | low | Only if a scenario or regulation requires proving the trail was not altered by someone with database access. Hash-chaining is the known answer; not justified for a single-operator local deployment. |

## Cross-References

- [`../concepts/DATA.md`](../concepts/DATA.md) — data ownership and retention
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — external services and secrets
- [`../operations/OBSERVABILITY.md`](../operations/OBSERVABILITY.md) — what may and may not be emitted
- [`ERROR-HANDLING.md`](ERROR-HANDLING.md) — error response behavior
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved security debt
