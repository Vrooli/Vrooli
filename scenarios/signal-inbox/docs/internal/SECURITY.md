# Security — Signal Inbox

This document records the scenario's security and privacy posture.
Update it before adding auth, user data, external APIs, payment flows,
secrets, or sensitive business data.

## Purpose Of This Document

Use this document to answer:

- What sensitive data exists?
- How is access controlled?
- Where do secrets come from?
- Which threats are known and how are they mitigated?

## Data Sensitivity

| Data | Sensitivity | Owner | Details |
|---|---|---|---|
| Operator-captured signals and archive content | sensitive personal-interest data | signals / sources | May include private bookmarks, likes, authored posts, and adult material. The journal is permanent. |
| Tier-1 OAuth access token | credential | sources / secret owner | Required only for an operator-enabled official API stream; never stored in signal rows, import runs, or browser-visible configuration. |

## Auth And Authorization

The generated template does not include an auth provider. Add auth only
when product requirements identify protected data or user-specific
behavior. UI and CLI must not enforce business authorization locally;
authorization belongs at the API/service layer.

## Secrets

| Secret | Source | Required? | Details |
|---|---|---|---|
| X OAuth client and user tokens | owning secret system | no, until X bookmark sync is enabled | Tier-1 X bookmarks stream only. Request the least scopes: `tweet.read`, `users.read`, `bookmark.read`. |

## Threat Model

| Risk | Impact | Mitigation | Status |
|---|---|---|---|
| Unsafe file upload handling | Malicious or oversized upload could affect storage. | Multipart handler validates metadata and BlobStore seam isolates bytes. | template-reference |
| Missing auth for product data | User/customer data could be exposed if added without access control. | Add API-layer auth before storing protected data. | deferred |
| Sensitive archive material sent to a hosted model | A provider could retain, moderate, or otherwise process material the operator intended to keep local. | Default archive ingestion, sensitive-content assessment, and classification to an operator-selected local ai-gateway profile. Do not rely on a classifier to make hosted processing safe. | planned source-stream implementation |
| One source toggle enables unrelated collection | A consent decision for bookmarks could accidentally activate likes, archives, or session replay. | Persist and enforce enablement, credentials, checkpoint, and risk tier per stream. | planned source-stream implementation |

## Security Gaps

| Gap | Severity | Revisit Trigger |
|---|---|---|
| No product-specific data classification | medium | Fill after PRD/domain map defines real data. |
| No auth model | conditional | Required before protected or multi-user data. |

## Cross-References

- [`../concepts/DATA.md`](../concepts/DATA.md) — data ownership and retention
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — external services and secrets
- [`ERROR-HANDLING.md`](ERROR-HANDLING.md) — error response behavior
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved security debt
