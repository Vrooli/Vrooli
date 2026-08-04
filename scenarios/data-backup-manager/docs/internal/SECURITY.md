# Security — Data Backup Manager

This document records the scenario's security and privacy posture.
Update it before adding auth, user data, external APIs, payment flows,
secrets, or sensitive business data.

## Purpose Of This Document

Use this document to answer:

- What sensitive data exists?
- How is access controlled?
- Where do secrets come from?
- Which threats are known and how are they mitigated?

> **Posture is design-intent.** Nothing below is implemented yet; it
> records the locked security design the implementation must satisfy.

## Data Sensitivity

This scenario backs up other scenarios' runtime state, so it
concentrates sensitive data: a single destination may contain the
contents of databases, caches, and filesystem trees that individually
hold secrets, credentials, and user data.

| Data | Sensitivity | Owner | Notes |
|---|---|---|---|
| Source artifacts (DB dumps, filesystem trees, cache/vector snapshots) | high (inherits the most sensitive source) | the registering scenario | Treated as potentially secrets-bearing. Always lands in an encrypted destination. |
| Kopia repository contents | high | data-backup-manager | Encrypted at rest by the engine; a leaked repository without its passphrase is not readable. |
| Manager catalog + run history (SQLite) | low–medium | data-backup-manager | Holds target/destination/plan metadata and run outcomes — no source bytes, no secret values. |
| Restore output | high (inherits source) | operator performing the restore | Written to an operator-chosen location; a restore is a privileged operation. |

## Auth And Authorization

The generated template does not include an auth provider. Authorization
belongs at the API/service layer, never enforced locally in UI or CLI.
For this scenario, **restore and destination-management are privileged
operations** — restoring rehydrates sensitive source data and writing a
new destination touches secrets. The intended model gates these behind
service-layer authorization once an auth provider is wired; until then
they run only in the trusted local stack. Self-registration of targets
is owner-keyed (owner+name), so a scenario can manage only its own
targets.

## Secrets

Repository passphrases are provisioned in the Vrooli credential authority
under a per-repository identity. S3/backend credentials and source
credentials continue to use the `vault` resource at runtime. No secret is
written to config files or passed on the command line. Where the wrapped
`kopia` (or source) CLI needs a secret, it is provided through the
environment or stdin, not argv, so it cannot leak through process listings.

| Secret | Source | Required? | Notes |
|---|---|---|---|
| Destination (kopia repository) passphrase | credential authority (`vrooli/kopia/<repository>` / `repository-passphrase`) | yes, per destination | Read at runtime; used to open/create the encrypted repository. Never persisted to the manager's config or DB. |
| Destination backend access keys (S3/MinIO etc.) | `vault` | when backend requires | Read at runtime; passed via env/stdin to kopia, never argv. |
| Source access credentials (Postgres, Redis, Qdrant, object-storage) | `vault` (via each source's resource CLI) | per source kind | Provided to the source CLI through its standard secret path; never inlined into argv. |

## Threat Model

| Risk | Impact | Mitigation | Status |
|---|---|---|---|
| Destination repository exfiltrated | Full disclosure of all source data in that repository. | Encryption ON by default for every destination; passphrase held only in the credential authority, never in the repo or config. | design-locked |
| Secret leaked via process listing/argv | Repo passphrase or backend key exposed to any local process. | Secrets passed to wrapped CLIs via env/stdin only; never argv; never persisted. | design-locked |
| Destination lives under the root it protects | A single incident destroys both source and its only backup. | Separate-root rule: destinations validated to be outside the protected root; offsite preferred for at least one tier. | design-locked |
| Backup that cannot restore | Silent loss — recovery fails when it is needed most. | Verified-restore is first-class; it gates removal of any committed runtime data from git. | design-locked |
| Silent eviction to stay under a cap | The only recovery copy is deleted automatically. | Storage limits default to alert+block; eviction only via explicit retention policy. | design-locked |
| Unauthorized restore rehydrating sensitive data | Sensitive source data written to an attacker-chosen location. | Restore is a privileged, service-layer-authorized operation. | design-locked (auth pending) |
| Backing up a secrets-bearing source | Secrets propagate into backups. | Backups inherit destination encryption; this is expected and accepted, not a leak path, provided the destination is encrypted. | design-locked |

## Security Gaps

| Gap | Severity | Revisit Trigger |
|---|---|---|
| No auth provider wired yet | conditional | Required before exposing restore/destination management outside the trusted local stack. |
| Restore-access controls are intent, not enforced | medium | Implement service-layer authorization for restore and destination mutation. |
| Redis source is best-effort, non-transactional | low (correctness, not confidentiality) | Documented limitation; revisit if a transactional snapshot path appears. See `PROBLEMS.md`. |

## Cross-References

- [`../concepts/DATA.md`](../concepts/DATA.md) — data ownership and retention
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — external services and secrets
- [`ERROR-HANDLING.md`](ERROR-HANDLING.md) — error response behavior
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved security debt
