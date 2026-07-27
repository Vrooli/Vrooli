# Security — Vrooli Memory

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
| Journal entry prose | **high — unbounded** | journal | This is the scenario's defining sensitivity: memory content is whatever an agent or operator decided to write, spanning project work and the operator's personal domains (health, finance, life). It is never classified in advance and cannot be. Treat the journal as the most sensitive store in the fleet. |
| Facet texts and embeddings | high | journal | Derived from entry prose; an embedding is not a safe redaction of its source. |
| Summary nodes | high | forest | Compressed entry content. Same sensitivity as the leaves they summarize. |
| Attribution + run correlation | low | journal | Identifiers only. No run payloads are copied — `vrooli-events` remains the one truth about a run. |
| Provider descriptor (`.vrooli/search.json`) | low | federation | Routing metadata and tuning; contains no memory content. Versioned in git. |

**Consequence of D-005 (no access-control partitioning).** Unified read across
all scenarios is a deliberate product decision, and it means any scenario that
can reach the memory API can read every memory including personal-domain
content. That is accepted for single-tenant local infrastructure. It would be
unacceptable in a multi-tenant or shared-host deployment, so multi-tenancy is
explicitly out of scope until this decision is revisited.

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
| None by default | n/a | no | Add entries when resources or third-party APIs require secrets. |

## Threat Model

| Risk | Impact | Mitigation | Status |
|---|---|---|---|
| Unsafe file upload handling | Malicious or oversized upload could affect storage. | Multipart handler validates metadata and BlobStore seam isolates bytes. | template-reference |
| Personal-domain memory readable by any scenario | A coding agent can retrieve operator health/finance memories. | **Accepted by decision** (D-005) for single-tenant local use. Not mitigated — mitigating it would undercut the product. | accepted |
| Memory content leaving the host via inference calls | Entry prose is sent to a model for classification, embedding, and summarization. | All inference routes through **ai-gateway**, never a direct third-party call. The gateway owns model policy, so "does memory content leave this machine" is a gateway configuration question with one answer, not a per-scenario one. | mitigated by design |
| Memory content leaking into federated search results | Memory registers as a search-hub provider, so hits surface in cross-corpus query output. | Intended behaviour — federated reach is `VROOLIME-P0-009`. The same D-005 acceptance applies. | accepted |
| Prompt injection via memory content | A malicious or mistaken memory is injected into every agent's `wake` context, especially if pinned. | Pinning is operator-confirmed rather than agent-assertable; re-facet and unpin are first-class corrections. **Not fully mitigated** — an agent can still write a misleading memory that reaches other agents. | partial |
| Unbounded write growth | Any agent may append without limit. | Append-only is intentional and the journal is never trimmed; growth is bounded operationally by disk, not by policy. | accepted |
| Control-plane endpoints (reindex / config write-back) | Unauthenticated reindex or tuning write-back would let a caller degrade retrieval. | Search-hub's `SearchControlService` verbs are token-gated by the shared contract. | inherited |

## Security Gaps

| Gap | Severity | Revisit Trigger |
|---|---|---|
| No auth model on the memory API | conditional | Required before any non-local deployment. Local single-tenant use is the only supported posture today. |
| Prompt-injection surface via pinned memories | medium | If agents begin writing memories that are read by agents with materially different privileges. |
| No redaction path for a memory written in error | medium | Append-only means a mistaken secret written into a memory cannot be deleted, only superseded. If secret material is ever written, the honest remedy is rotating the secret — not editing the journal. Revisit if this happens in practice. |

## Cross-References

- [`../concepts/DATA.md`](../concepts/DATA.md) — data ownership and retention
- [`../concepts/INTEGRATIONS.md`](../concepts/INTEGRATIONS.md) — external services and secrets
- [`ERROR-HANDLING.md`](ERROR-HANDLING.md) — error response behavior
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved security debt
