# Security — Asset Studio

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
| Identity records and blocks | low | identities | Descriptions of **fictional** persona-actors, environments, and products. Not personal data about real people, and that boundary is enforced rather than assumed — see the likeness check below. |
| Reference images and character sheets | low | identities | Generated artifacts, not photographs of identifiable people. If that ever ceases to be true, this document must be updated before the first such record is stored. |
| Credential-claims field | **policy-critical** | identities | Required and required-empty on persona-depicting records. Not sensitive to disclose; consequential to get wrong. A non-empty value blocks release. |
| Specs and resolved payloads | low | specs | Prompt text. May reveal unreleased campaign direction, which is a confidentiality concern rather than a personal-data one. |
| Provenance and cost records | low | renders | Operational audit data. Cost figures are business-sensitive in aggregate. |
| Produced artifact bytes | **medium** | assets | Unreleased marketing material. Local-first storage is the correct default; these are the bytes an accidental public deployment would expose. |
| Conformance verdicts | low | conformance | Operator judgements with attribution. Audit surface for published material. |

<!-- EXAMPLE-DOMAIN:notes START -->
The shipped worked-example `notes` domain carries placeholder data only
(removed by `template-manager detemplate`):

| Data | Sensitivity | Owner | Details |
|---|---|---|---|
| Template notes data | low | notes reference | Local development data only; replace with real scenario data classification. |
| Attachment bytes | unknown | notes reference | Treat as potentially sensitive if retained in product scope. |
<!-- EXAMPLE-DOMAIN:notes END -->

## Auth And Authorization

The generated template does not include an auth provider. Add auth only
when product requirements identify protected data or user-specific
behavior. UI and CLI must not enforce business authorization locally;
authorization belongs at the API/service layer.

## Secrets

| Secret | Source | Required? | Details |
|---|---|---|---|
| None. | n/a | no | **This scenario holds no credential of any kind.** Model access is ai-gateway's (D-008); publishing identity and platform tokens are the scheduler's, held as vault references. A schema change introducing a token or handle column here is a defect, not a feature. |

## Threat Model

| Risk | Impact | Mitigation | Status |
|---|---|---|---|
| A persona makes a false credential claim | Deception rather than embellishment. Brand damage, and in regulated domains a legal exposure. | `credential_claims` is a required, required-empty field on persona-depicting records, and release is refused when it is non-empty (`ASSET-P0-013`). Structural rather than a review note. | designed |
| Generated likeness resembles a real identifiable person | Impersonation. AI generation makes accidental likeness easier, and the AI-UGC canon bans it outright. | A likeness policy check in `conformance`, judged by the operator against the frame. Not automatable today and not pretended to be. | designed |
| An AI-generated asset is published without disclosure | Platform policy violation and FTC exposure. | Disclosure state is set at creation by the producing path, not by the caller, and travels with the asset reference (`ASSET-P0-012`). The label never depends on a later step remembering. | designed |
| Unreleased marketing material exposed | Campaign direction and unpublished artifacts leak. | Local-first storage, no hosted deployment, no external network egress except ai-gateway. Desktop packaging (P2) keeps it that way deliberately. | designed |
| Runaway generation spend | Real money consumed before a human sees a frame. | Cost estimated before submission and recorded per attempt (`ASSET-P0-008`); budgets with explicit confirmation at P1 (`ASSET-P1-006`). | partial — the budget is P1 |
| Prompt content reaching an unintended model | Campaign direction disclosed to a third-party vendor. | Every call routes through ai-gateway, which owns routing and BYOK/hosted policy. This scenario speaks no vendor protocol and cannot bypass it. | designed |
| Unsafe artifact bytes served to a browser | Stored-content attack through the asset surface. | Bytes stay behind the BlobStore seam and are served with declared content types; artifact bytes never enter proto payloads. | template-reference |

## Security Gaps

| Gap | Severity | Revisit Trigger |
|---|---|---|
| Likeness checking is entirely manual | medium | If persona volume grows past what an operator can review frame by frame. There is no automated check today and none is claimed. |
| No auth model | conditional | Local single-operator use needs none. Required before any multi-user or hosted deployment, which is explicitly not the plan (see `content-desk` D-017 for the same posture). |
| Spend has accounting but no cap at P0 | medium | `ASSET-P1-006`. Until then the control is that P0 renders still images, which are cheap. Broadening P0 to video without the budget would make this high. |
| No artifact-pruning policy | low | Once real generation volume exists. A pruning policy must preserve provenance and verdicts. |

## Cross-References

- [`../concepts/DATA.md`](../concepts/DATA.md) — data ownership and retention
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — external services and secrets
- [`ERROR-HANDLING.md`](ERROR-HANDLING.md) — error response behavior
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved security debt
